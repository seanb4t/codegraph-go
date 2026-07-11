---
phase: 04-incremental-sync-file-watcher
reviewed: 2026-07-11T21:31:51Z
depth: deep
files_reviewed: 26
files_reviewed_list:
  - internal/graphstore/keys.go
  - internal/graphstore/store.go
  - internal/graphstore/batch.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/export.go
  - internal/indexer/sync.go
  - internal/indexer/resolve.go
  - internal/indexer/symbolindex.go
  - internal/indexer/discover.go
  - internal/indexer/extract.go
  - internal/indexer/pipeline.go
  - internal/indexer/goextract/types.go
  - internal/watch/watcher.go
  - internal/watch/debounce.go
  - internal/daemon/daemon.go
  - internal/daemon/lock.go
  - internal/query/traverse.go
  - internal/query/status.go
  - internal/query/render_markdown.go
  - internal/query/explore.go
  - internal/query/node.go
  - internal/cli/sync.go
  - internal/cli/daemon.go
  - internal/cli/unlock.go
  - internal/cli/serve.go
  - internal/cli/status.go
  - internal/cli/root.go
findings:
  critical: 4
  warning: 5
  info: 2
  total: 11
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-07-11T21:31:51Z
**Depth:** deep
**Files Reviewed:** 26
**Status:** issues_found

## Summary

This review focused on the concurrency lifecycle of `internal/watch`/`internal/daemon`, the
pid-liveness lockfile in `internal/daemon/lock.go`, and the incremental-sync correctness of
`internal/indexer/sync.go` against the new `x/` file-owned secondary index in
`internal/graphstore`. Four BLOCKER-level defects were found, none of which the stated green
test suite (`go test ./... -race`, goleak soak, prune fixtures, determinism gate) would catch,
because each requires either a genuinely concurrent shutdown race, a two-daemon race window, or
a specific *multi-generation* incremental-sync scenario (cross-file receiver methods, or a
dependent file whose resolved edge identity changes across syncs) that a single reindex-vs-sync
comparison does not exercise.

The most serious findings are: (1) `Daemon.Run` releases the daemon lock and can return while a
debounced flush's `indexer.Sync` is still writing to the store on an untracked background
goroutine, violating the single-writer invariant the package explicitly documents; (2)
`daemon/lock.go`'s `acquire()` has a classic TOCTOU race that lets two daemons simultaneously
believe they hold an exclusive lock; (3) `indexer/sync.go`'s `Sync` builds its `ownerPath`
lookup only from the reparsed batch's own nodes, so a `contains` edge whose source type lives in
an *unchanged* sibling file (the common "type in one file, methods in another" Go idiom) is
committed with an empty owner, permanently escaping the `x/` index and becoming an unprunable
dangling edge; and (4) the "narrow prune" path for dependent files deletes `e/` edge records
without ever deleting the corresponding `x/` file-index entries for those edges, so the index
accumulates stale references and `Meta.EdgeCount` drifts over repeated sync cycles.

## Critical Issues

### CR-01: Daemon.Run can release the lock while a Sync is still writing (untracked flush goroutine)

**File:** `internal/daemon/daemon.go:102-130` (see also `internal/watch/debounce.go:64` and `internal/watch/debounce.go:91-98`)

**Issue:** `Daemon.Run` starts exactly one tracked goroutine — the watcher's `watchLoop`
(`wg.Add(1)` / `go func(){ defer wg.Done(); w.Run(ctx, deb) }()`) — and after `<-ctx.Done()`
calls only `wg.Wait()` before returning and letting the deferred `release(d.codegraphDir)` run.

The debounced flush itself does **not** run on that tracked goroutine. `Debouncer.Add`
schedules `d.timer = time.AfterFunc(d.window, d.fire)` (`debounce.go:64`), and per the stdlib's
own contract, `AfterFunc` "calls f in its own goroutine" and `Timer.Stop` "does not wait for f
to complete before returning." `Debouncer.Stop` (called from `watchLoop` on `ctx.Done()`) only
calls `d.timer.Stop()` — it cannot retroactively join a `fire()` invocation that has already
started running `d.flush` (`daemon.go:143-161`), which itself does the expensive
`indexer.Sync(...)` call under `d.syncMu`.

Sequence that breaks the invariant:
1. The debounce timer fires (before or racing with `ctx` cancellation) and `fire()` passes its
   `ctx.Err() != nil` check (still nil at that instant), then calls `d.flush(paths)`, which
   starts `indexer.Sync(...)`.
2. `ctx` is cancelled (SIGINT/SIGTERM). `watchLoop` selects the `ctx.Done()` case, calls
   `deb.Stop()` (a no-op against the already-running `fire()`), and returns. `wg.Done()` fires.
3. `Run`'s `wg.Wait()` unblocks immediately — it was only ever waiting on the watcher goroutine.
   `Run` returns; the deferred `release(d.codegraphDir)` removes `daemon.lock` — **while the
   Sync from step 1 is still mid-write inside the (separate) GraphStore.Writer.Commit().**

This directly contradicts the package's own documented contract ("Run blocks until ctx is
cancelled and every spawned goroutine has joined (D-07)... no goroutine outlives Run
(SYNC-06)") and the single-writer invariant (INDX-05) the lockfile exists to protect: a caller
that treats `Run`'s return as "safe to start another daemon" or that exits the process
immediately after `Run` returns can race a second `GraphStore.Open`/Writer against the first
one's still-in-flight commit, or truncate a Sync mid-commit if the whole process exits. The same
bug is reachable via `internal/cli/serve.go`'s `--watch` in-process fallback, which drives the
identical `daemon.Run(watchCtx)` lifecycle.

**Fix:** Make `Run` wait for any in-flight flush before returning — the simplest correct fix is
to add a final barrier on `syncMu` after `wg.Wait()`, since every flush invocation acquires it
before doing any work:

```go
<-ctx.Done()
wg.Wait()
// Block until any flush that started before ctx was cancelled has finished
// its Sync — a flush that hasn't started yet will see ctx.Err() != nil in
// fire() and never reach syncMu at all.
d.syncMu.Lock()
d.syncMu.Unlock()
return nil
```

(A cleaner longer-term fix is to have `Debouncer` itself track in-flight `fire()` calls with a
`sync.WaitGroup` and expose a `Wait()` the caller can join, rather than relying on `syncMu` as an
implicit barrier.)

---

### CR-02: `daemon/lock.go` `acquire()` has a TOCTOU race — two daemons can both "win" the lock

**File:** `internal/daemon/lock.go:98-117`

**Issue:** `acquire()` is a classic read-then-write race:

```go
info, ok, err := readLock(codegraphDir)   // no lock exists: ok == false
...
data, err := json.Marshal(lockInfo{...})
return os.WriteFile(lockPath(codegraphDir), data, 0o644)  // unconditional overwrite
```

`os.WriteFile` opens with `O_WRONLY|O_CREATE|O_TRUNC` — it does not fail if the file already
exists, and there is no exclusive-create (`O_EXCL`) or filesystem-level locking anywhere in this
path. If two processes call `acquire()` for the same repo within the same race window (exactly
the scenario this package exists to serve — "multiple agent sessions share" a daemon, per the
package doc comment), both `readLock` calls can observe "no lockfile" (`ok == false`), both skip
the `isStale` check entirely, and both then unconditionally `WriteFile` a lockfile recording
their own pid. Both `acquire()` calls return `nil` — both processes believe they are the sole
daemon, both open watchers and both begin driving `indexer.Sync` against the same store. This
directly violates the single-writer invariant (INDX-05) and the package's own stated purpose,
and is a more fundamental gap than the already-documented PID-reuse staleness risk (which at
least fails safe by refusing to acquire).

**Fix:** Use atomic exclusive creation for the initial acquire attempt, falling back to the
existing stale-check-and-remove path only on `EEXIST`:

```go
func acquire(codegraphDir string) error {
	data, err := json.Marshal(lockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath(codegraphDir), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		_, werr := f.Write(data)
		cerr := f.Close()
		if werr != nil {
			return werr
		}
		return cerr
	}
	if !os.IsExist(err) {
		return err
	}
	// Lockfile already exists — fall through to the existing stale-check path,
	// then retry the O_EXCL create once (still racy against a third
	// concurrent acquirer, but closes the "nothing exists yet" window this
	// bug report is about).
	...
}
```

---

### CR-03: Incremental `Sync` drops the `x/` file-index entry for cross-file `contains` edges — permanent dangling edges

**File:** `internal/indexer/sync.go:247-303` (root cause interacts with `internal/indexer/resolve.go:90-104`)

**Issue:** `Sync` builds its edge-ownership map only from the reparsed batch's own nodes:

```go
nodeFilePath := make(map[string]string, len(nodes))
for _, n := range nodes {
    nodeFilePath[n.Id] = n.FilePath
}
...
for _, e := range collapsedEdges {
    if err := w.PutEdge(e, nodeFilePath[e.Source]); err != nil { ... }
}
```

`nodes` here is `resolveRefsWithIndex`'s first return value — i.e., only nodes freshly extracted
from `batchFiles` (added ∪ modified ∪ dependent) this cycle, **not** the whole graph.

For three of the four `UnresolvedRef` kinds (`calls`, `embeds`, `imports`), `Source` is always
`ref.FromID`, the enclosing symbol in the file currently being parsed — always in-batch, so the
lookup always succeeds. But `RefKindContains` (resolve.go:90-104) is the one kind Go idiomatically
triggers across files: a method's receiver type declared in a *different, unchanged* file of the
same package (e.g. `type.go` declares `type Foo struct{}`, `methods.go` declares `func (f *Foo)
Bar()`). For this kind, `Source: typeID` comes from `idx.resolveUnqualified(...)`, and `idx` is
the store-seeded index (`newSymbolIndexFromStore`) — so when only `methods.go` is edited,
`typeID` resolves to a node whose owning file (`type.go`) was **not** reparsed this cycle and is
therefore absent from `nodeFilePath`. The map lookup silently returns `""`, and `PutEdge`
(`graphstore/batch.go:49-51`) treats an empty `ownerPath` as "no owner — skip the `x/` entry
entirely."

Consequence: the `contains` edge is written correctly into the `e/` namespace (queries still see
it), but it is never indexed under `x/<type.go>/...`. If `type.go` is later modified or deleted,
`pruneFileSubgraph`'s `IterateFileIndex(path)` scan (sync.go:404-432) will never enumerate this
edge, so it is never deleted — it becomes a permanently orphaned `e/` record whose `Source` node
may since have been removed, inflating `IterateEdges`/`BuildReverseAdjacency` results and
corrupting `Meta.EdgeCount` bookkeeping forever. Note that this bug does **not** manifest on a
from-scratch `Run`/`writeGraph` (resolve.go:256-269), because there `nodeFilePath` is built from
every node in the whole repo — this is exclusively an incremental-`Sync` regression, and would
not be caught by a "sync equals reindex" determinism gate run against a single before/after
snapshot, since it only shows up across a second, later sync cycle that touches the type's file.

**Fix:** `Sync`'s ownerPath resolution needs full-graph node coverage, not just the batch's own
nodes. E.g., fall back to a store lookup when the batch map misses:

```go
func ownerPathFor(id string, batch map[string]string, r0 graphstore.Reader) string {
    if p, ok := batch[id]; ok {
        return p
    }
    if n, err := r0.GetNode(id); err == nil {
        return n.GetFilePath()
    }
    return ""
}
...
if err := w.PutEdge(e, ownerPathFor(e.Source, nodeFilePath, r0)); err != nil { ... }
```

---

### CR-04: `pruneOwnedEdgesOnly` deletes `e/` edge records but never the matching `x/` index entries — index drift/leak on every dependent re-sync

**File:** `internal/indexer/sync.go:434-458` (contrast with `pruneFileSubgraph`, `sync.go:404-432`, and its caller loop `sync.go:150-159`)

**Issue:** When a file is directly modified or deleted, its whole subgraph is pruned via
`pruneFileSubgraph` **followed by** `w.DeleteFileSubgraph(path)` (sync.go:150-159), which
range-deletes the entire `x/<path>/...` region — both node and edge sub-ranges — so the index
stays consistent with the deleted records.

Dependent files (files that merely referenced a now-gone symbol, and must be re-extracted so
their `Unresolved` refs regenerate) go through a different, narrower path:
`pruneOwnedEdgesOnly` (sync.go:434-458), called from the loop at sync.go:201-206. For each edge
entry found via `IterateFileIndex(path)`, it calls `w.DeleteEdge(entry.Source, entry.Kind,
entry.Target)` — a point-delete of the `e/` record — but there is **no** corresponding call to
remove the `x/<path>/e/<source>/<kind>/<target>` index entry itself (there is no such API on
`Writer` at all — `graphstore/store.go`'s `Writer` interface exposes only `DeleteNode`,
`DeleteEdge`, and `DeleteFileSubgraph`, none of which point-delete a single `x/` edge entry).

Since the dependent file's content is unchanged, its re-extraction can legitimately resolve to a
*different* edge target than before (that's the whole point of re-resolving it). Whenever that
happens, the **old** `x/<path>/e/...` entry — pointing at a triple that no longer has a matching
`e/` record — is never cleaned up and persists forever. On a later sync cycle where this same
file is *directly* modified, `pruneFileSubgraph`'s `IterateFileIndex(path)` scan will enumerate
this stale entry too, issue a no-op `DeleteEdge` against a key that doesn't exist, and still
increment `edgesRemoved` — corrupting the `newMeta.EdgeCount = meta.GetEdgeCount() -
edgesRemoved + len(collapsedEdges)` arithmetic (sync.go:308-309) with a phantom removal that
never happened. Over the life of an actively-developed repo with dependent-edge churn, this
drifts `Meta.EdgeCount` (and the growing `x/` namespace itself) away from ground truth — a
"partial failure leaves the graph inconsistent" scenario that a single-shot determinism gate
comparing one sync against one reindex will not exercise (it requires at least two sync
generations with a changing dependent-edge target).

**Fix:** Add a `Writer.DeleteFileIndexEdge(ownerPath, source, kind, target string) error` (a
point-delete of `fileIndexEdgeKey(ownerPath, source, kind, target)`) and call it alongside
`DeleteEdge` in `pruneOwnedEdgesOnly`:

```go
func pruneOwnedEdgesOnly(r0 graphstore.Reader, w graphstore.Writer, path string, edgesRemoved *int) error {
    ...
    for it.Next() {
        entry := it.Entry()
        if entry.IsNode {
            continue
        }
        if err := w.DeleteEdge(entry.Source, entry.Kind, entry.Target); err != nil {
            return err
        }
        if err := w.DeleteFileIndexEdge(path, entry.Source, entry.Kind, entry.Target); err != nil {
            return err
        }
        *edgesRemoved++
    }
    return it.Err()
}
```

## Warnings

### WR-01: `pebbleStore.Snapshot`/`NewWriter` check-then-act on `closed` is not atomic with the underlying Pebble call

**File:** `internal/graphstore/pebble_store.go:56-71`

**Issue:** Both methods do `if s.closed.Load() { return nil, ErrClosed }` and then call
`s.db.NewSnapshot()` / `s.db.NewBatch()` as a separate step. `pebbleStore.Close()`
(`pebble_store.go:78-83`) can run concurrently between the load and the Pebble call. The file's
own comments acknowledge that `pebble/v2 panics rather than returning an error once its
*pebble.DB is closed` and explicitly flag the analogous race for `Commit` as "not guarded by
this sentinel" — but the same unguarded race exists for `Snapshot`/`NewWriter` themselves, not
just `Commit`. Every current call site in this codebase opens/closes its own `pebbleStore`
instance without sharing it across goroutines, so this is not exercised today, but it is a latent
panic risk the moment a store handle is shared (e.g. a future long-lived MCP-server-owned store,
or the team-scale concurrent-access direction called out in `CLAUDE.md`).

**Fix:** Guard the closed-check and the Pebble call under the same `sync.RWMutex` (readers take
`RLock`, `Close` takes `Lock`), or accept and document this as intentionally single-goroutine-use
only.

### WR-02: `lockInfo.StartedAt` is recorded but never consulted — PID-reuse can permanently wedge daemon startup

**File:** `internal/daemon/lock.go:40-43, 68-70`

**Issue:** `isStale` is `!isProcessLive(info.PID)` only; `StartedAt` is written into every
lockfile but never read back anywhere in this file. The type's own doc comment says this is a
known, deliberate v1 gap ("documented, not eliminated"), but it means the residual PID-reuse
false-negative is real today: if the original daemon's pid gets reused by an unrelated process
after a crash (routine in containers/PID-namespace-reused environments), `acquire()` and
`Unlock()` will both treat the stale lock as live indefinitely, requiring a human to notice and
manually intervene (there is no automatic recovery path). Given this is explicitly in the
phase's stated review priorities, it's worth flagging even though disclosed.

**Fix:** At minimum, cross-check the OS's own process start time (e.g. via `/proc/<pid>/stat` on
Linux) against `StartedAt` when corroboration data is available, falling back to the current
liveness-only check on platforms where that's unavailable.

### WR-03: Stat-pre-filter false-modified-but-hash-equal files never get their stored mtime/size refreshed

**File:** `internal/indexer/sync.go:82-104`

**Issue:** When `stored.GetMtimeUnixNs()`/`SizeBytes` differ from disk but the recomputed content
hash matches, the file is correctly treated as unmodified (`continue`) — but it is not added to
`modified`, so its `schema.File` record (still carrying the *old* mtime/size) is never rewritten.
Every subsequent `Sync` invocation will re-fail the cheap stat pre-filter for this file and pay
the full `contentHash` read+hash cost again, forever (e.g. after a touch, git checkout without
content change, or a mtime-only editor save). This is a correctness-adjacent metadata-staleness
issue, not just a performance one — the stored `File.MtimeUnixNs` permanently diverges from
on-disk truth for such files.

**Fix:** When the hash matches but stat metadata differs, still stage a `PutFile` refresh (with
the new mtime/size, same content hash) so the pre-filter's fast path is restored on the next sync.

### WR-04: `Daemon.opts` is never populated — per-daemon `indexer.Options` customization is dead

**File:** `internal/daemon/daemon.go:49, 74-91`

**Issue:** `Daemon.opts indexer.Options` is declared and used at `daemon.go:149`
(`indexer.Sync(d.repoRoot, d.storeDir, d.opts)`), but `New()` never assigns it and there is no
setter — it is permanently the zero value. Any intended `--workers`/`--verbose`/`--quiet`
wiring through to the daemon's own sync calls is unreachable.

**Fix:** Either thread `indexer.Options` through `New(repoRoot string, opts indexer.Options)` (and
update `cli/daemon.go`'s caller), or remove the field if daemon-level customization is
intentionally out of scope for v1.

### WR-05: `dependentPaths` can include already-deleted files, wasting a redundant prune pass

**File:** `internal/indexer/sync.go:172-196`

**Issue:** Dependent-file detection walks the pre-sync reverse-adjacency map for every `goneIDs`
entry and adds the caller's `FilePath` to `dependentPaths` whenever it isn't already in
`changedSet` (added ∪ modified). It does not check whether that caller's file is itself in
`deleted`. A now-deleted file that used to call a pruned symbol is added to `dependentPaths`,
runs through `pruneOwnedEdgesOnly` redundantly (harmless no-op deletes against records already
removed by the earlier `pruneFileSubgraph` pass over the same path), and is then silently
dropped from `batchFiles` because it's absent from `discovered`. Not a correctness bug (the
redundant work is idempotent), but unnecessary churn and a slightly confusing code path.

**Fix:** Skip paths already present in `deleted` (as a set) when building `dependentPaths`.

## Info

### IN-01: `Watcher.root` field is dead code

**File:** `internal/watch/watcher.go:33, 47`

**Issue:** `root` is stored on `Watcher` at construction (`Open`) but never read anywhere else in
the package (confirmed via `grep -rn "\.root" internal/watch/`).

**Fix:** Remove the field, or use it (e.g. to expose `Watcher.Root()` for callers/tests) if it
was intended to be read.

### IN-02: `CalleesResult`/`CallersResult`/`ImpactResult` cap logic applies `limit` before `MaxLimit`, silently reordering which truncation "wins" when both are tighter than the actual result count

**File:** `internal/query/traverse.go:169-174, 209-214`

**Issue:** Not a bug in the current code (the two clamps compose fine, since both simply shrink
the slice further and clamp order doesn't change the final result for a monotonic min), but
worth a note for future maintainers: `if limit > 0 && limit < len(locs) { locs = locs[:limit] }`
followed unconditionally by `if len(locs) > MaxLimit { locs = locs[:MaxLimit] }` means a caller
passing `limit > MaxLimit` silently gets `MaxLimit` results with no indication the requested
limit was reduced. Consider surfacing that at the API boundary (e.g. a `Truncated bool` on the
result) if callers need to distinguish "there were more results" from "you asked for more than
we allow."

---

_Reviewed: 2026-07-11T21:31:51Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
