# Phase 4: Incremental Sync & File Watcher - Research

**Researched:** 2026-07-11
**Domain:** Incremental graph maintenance (content-hash diffing, subgraph pruning), native filesystem watching, single-writer daemon lifecycle
**Confidence:** HIGH (grounded directly in the actual Phase 1-3 codebase; fsnotify/goleak/pebble API claims verified via Context7 official docs)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Incremental Sync Engine (INDX-03)**
- D-01: `codegraph sync` is a thin Cobra command delegating to a new `internal/indexer.Sync()` incremental entry that reuses the existing `extract`/`resolve`/`symbolindex`/`discover` internals — not a second pipeline. `sync` resolves the nearest `.codegraph/` (same resolution idiom as query commands) and errors clearly if uninitialized.
- D-01a: Change detection = stat pre-filter → content-hash confirm. Compare on-disk file mtime/size against the graph's per-file record to cheaply shortlist candidates, then confirm an actual change by recomputing SHA-256 into `File.content_hash`. Only files whose hash changed are reparsed. This is exactly the "stat comparison + content hashing" SYNC-03 also needs, so the same reconcile routine backs both `sync` and MCP-reconnect (D-07 in this doc's numbering — see D-06 below).
- D-01b: `sync` flags mirror `index` (`--quiet`/`--verbose`, concise default summary: files reparsed, files pruned, nodes/edges added/removed, dependents recomputed, duration; `--json` executor's discretion). Writes through the single coordinated `GraphStore.Writer` in batched windows (never per-symbol), stamping `Meta.last_sync_unix_ms` on commit.

**Subgraph Pruning & Keyspace — the load-bearing decision (INDX-04)**
- D-02: Add a file-owned secondary index so a file's nodes + outgoing edges can be pruned by range-delete. Introduce an additive namespace under the reserved-namespace mechanism, populated at `init`/`index`/`sync` time, so `DeleteFileSubgraph(path)` becomes an O(subgraph) prune. Additive keyspace only — `SchemaVersion` stays `1`.
- D-02a: Dangling-edge prevention via dependent recomputation, NOT a persisted reverse index. Each sync builds the Phase-3 D-04 query-time reverse-adjacency map once to find files with an edge targeting a symbol in a changed/deleted file, adds them to the recompute set, and re-resolves them. A persisted reverse index stays Phase 8.
- D-02b: Backfill/fallback for pre-Phase-4 graphs. A `sync` that detects the secondary index's absence (Meta flag or missing namespace) performs a one-time full re-index to populate it, then subsequent syncs are incremental. Silent-safe and logged; never a silently-wrong partial prune.

**Rename / Move Handling (INDX-04)**
- D-03: Treat rename/move as delete(old path) + add(new path) at the file level. Content-hashed node ids mean a symbol whose content is unchanged keeps a stable `n/<id>` **for symbols in files that were not themselves renamed**; the fixture suite asserts no orphaned nodes/dangling edges after create, modify, delete, rename, move.

**Native Watcher & Debounce (SYNC-01, SYNC-02)**
- D-04: Build on `github.com/fsnotify/fsnotify` (new direct dependency) with an explicit recursive-add-on-Create loop (fsnotify does not recurse). Coalesce a burst of events into one `Sync` over the union of changed paths; debounce default 2000ms, tunable via env (`CODEGRAPH_DEBOUNCE_MS`). Always ignore `.codegraph/` and honor the same discovery exclusions `discover.go` already applies. No polling as default.
- D-04a: Staleness banner (SYNC-02): while a sync is pending/debouncing, agent-facing output shows a staleness warning — prepended to `explore` markdown output and reflected as a `stale`/`pendingSync` signal in `status`. Pending state set by watcher/daemon on first event, cleared on sync commit; a stat/`last_sync_unix_ms` comparison is the fallback signal when no daemon is running. Threads through MCP `explore` too (one formatter, D-08b).

**Daemon, Lock & unlock (SYNC-04, SYNC-05)**
- D-05: `codegraph daemon` = long-lived local process owning the watcher + the single `GraphStore.Writer`. Guard with a lockfile in `.codegraph/` carrying pid + start timestamp (single-writer invariant, INDX-05). In-process fallback: the watcher runs inside `serve` (or a CLI invocation) at the same single-writer discipline where a separate daemon is unsupported/undesired. `codegraph unlock` removes a stale lock after a crash — checks pid liveness (and timestamp staleness) before clearing.

**MCP Reconnect Reconciliation (SYNC-03)**
- D-06: On MCP server (re)connect/init, run the D-01a stat+content-hash reconciliation (a `Sync` pass) before serving tools. Cheap no-op when nothing changed. Reuses the exact `sync` reconcile routine — no second reconciliation code path.

**Leak-Free Lifecycle & Soak (SYNC-06)**
- D-07: Lifecycle via `context` cancellation + `sync.WaitGroup` join on shutdown — every goroutine (watcher loop, debounce timer, sync worker) is owned by a cancelable context and joined before the daemon/watcher returns. A soak test exercises many watch→debounce→sync cycles and asserts goroutine-leak-free (via `go.uber.org/goleak` or a `runtime.NumGoroutine()` delta gate; library choice executor's discretion).

### Claude's Discretion
- Exact secondary-index key shape for D-02 and its backfill-detection mechanism (Meta flag vs namespace probe).
- Sync writer batch granularity (per-file vs per-N) so long as no per-symbol commits and Phase-2 determinism holds.
- Debounce data structure/timer reset strategy; whether to leverage the CGo parser's incremental-reparse path or full-reparse changed files.
- Lockfile format/contents beyond pid+timestamp; `daemon` foreground vs background and its IPC (if any) with `serve`.
- Exact staleness banner text/placement and the `status` field name; `--json` shape of the `sync` summary.
- Soak-test iteration counts and the leak-detection library.

### Deferred Ideas (OUT OF SCOPE)
- Persisted reverse-edge index — Phase 8 scale work; adopt only if the D-02a query-time reverse scan profiles as a bottleneck on a 100k-file monorepo.
- Interface→impl dispatch synthesis, `provenance: heuristic` tagging, framework `route` nodes, non-Go languages — Phase 5. Also WR-01 (same-package func/method collision) and WR-02 (selector-on-non-identifier) — sync must not work around them.
- 100k-file monorepo scale, peak-RSS bounding, performance-regression gates — Phase 8.
- HTTP/SSE MCP transport, remote/multi-user daemon — v2 (SERVER-01); Phase 4 daemon is local single-user.
- `install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry` — Phase 6.
- Migration from TS SQLite `.codegraph/` — Phase 7.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INDX-03 | `codegraph sync` incrementally reparses only changed files via content-hash diffing with dependent-file edge recomputation | §Architecture Patterns Pattern 1 (Sync algorithm), §Common Pitfalls Pitfall 1 (symbol-index-from-store), Pitfall 2 (dependent files still need re-extraction) |
| INDX-04 | File renames/deletes/moves prune stale symbols/edges — no orphans/dangling edges, fixture-suite verified | §Architecture Patterns Pattern 2 (D-02 secondary index design), §Validation Architecture (fixture suite) |
| SYNC-01 | Native per-OS watchers, debounced (default 2000ms, env-tunable), consolidating edit bursts into one sync | §Architecture Patterns Pattern 3 (watcher + debounce), §Code Examples |
| SYNC-02 | Staleness warning banner while sync is pending/debouncing | §Architecture Patterns Pattern 4 (staleness signal threading) |
| SYNC-03 | MCP (re)connect reconciles via stat comparison + content hashing | §Architecture Patterns Pattern 5 (reconnect reconcile) |
| SYNC-04 | `codegraph daemon` shared watch/index server with in-process fallback | §Architecture Patterns Pattern 6 (daemon + lock) |
| SYNC-05 | `codegraph unlock` clears stale locks after a crash | §Architecture Patterns Pattern 6, §Code Examples (pidfile pattern) |
| SYNC-06 | Watcher/daemon goroutine-leak-free, soak-test verified | §Architecture Patterns Pattern 7 (context+WaitGroup lifecycle), §Validation Architecture (soak test) |
</phase_requirements>

## Summary

Phase 4's spine is the file-owned secondary index (D-02) that the codebase was **deliberately built to accommodate**: `keys.go:104-118` and `Writer.DeleteFileSubgraph` exist today as a Phase-1-planted hook that prunes only the `f/` record. Extending it correctly requires an additive namespace (`x/`, this research's recommended byte) that maps file path → owned node ids and owned outgoing-edge triples, populated whenever `PutNode`/`PutEdge` are staged. Because node ids are content-hashes of `(kind, qualifiedName, filePath)` — **not** sequential or file-clustered — the file's own nodes and edges are scattered throughout the `n/`/`e/` keyspace, so pruning a file's subgraph is not literally one contiguous `DeleteRange` over `n/`+`e/`; it is a prefix-scan of the new `x/<path>/...` index (itself one contiguous, cheaply range-deletable region) that yields the exact keys to point-delete from `n/`/`e/`, followed by range-deletes of `x/<path>/...` and `f/<path>`. This is still strictly O(file's own subgraph size), never a full-graph scan.

The second load-bearing finding is that **`Resolve()`'s existing symbol index cannot simply be re-run over a subset of files** — `newSymbolIndex(results)` builds `(importPath, name) → nodeID` purely from the in-memory `results` slice passed to it, so calling it with only the changed-file batch would make every OTHER file's symbols invisible to cross-file resolution, breaking `resolveUnqualified`/`resolveSelector` for anything referencing an untouched file. `Sync()` must instead seed its symbol index from the **stored graph** (`IterateNodes()` on a snapshot, reconstructing each node's Go import path from `node.FilePath` via `indexer.importPathFor(modulePath, filePath)` — no schema change needed) and overlay freshly-extracted symbols for the reparse batch on top, superseding only the import-path entries the batch actually touches.

Third: because `goextract.FileResult.Unresolved` (the list of not-yet-resolved call/embed/import/contains references) is never persisted to the store — only successfully *resolved* edges are — a "dependent file" whose caller must be re-resolved (D-02a) has to be **re-extracted (Pass 1) even though its own content hash is unchanged**, purely to regenerate its `Unresolved` list in memory. This does not violate "not a full re-index of a subset": the reparse batch is bounded by (changed files) ∪ (their direct dependents via the reverse-adjacency scan), never the whole repo — but the planner should not read D-01a's "only files whose hash changed are reparsed" as literally excluding dependent files from Pass-1 extraction; it excludes them from the *content-hash-changed* set, not from the *batch Extract() runs over*.

Fourth: `File` currently has no `mtime`/`size` fields — D-01a's "stat pre-filter → content-hash confirm" cannot be implemented against today's schema without adding two additive fields (`mtime_unix_ns`, `size_bytes`) to `File`, stamped by both `index` and `sync`.

Fifth: the CGo parser's incremental-reparse path (`Parser.Parse(source, oldTree)`) is real but currently **unused** — every call site passes `oldTree=nil` — and using it correctly requires tree-sitter byte-range edit tracking (`TSTree.Edit`) that does not exist anywhere in this codebase. Given the "sync-equals-reindex" determinism gate this phase itself introduces as its correctness bar, this research recommends **full re-parse of changed files** (not tree-sitter incremental reparse) for v1, deferring true incremental reparse to a performance phase — the risk of a subtly different tree from an unvalidated edit-tracking path threatens the very determinism property the phase must prove.

**Primary recommendation:** Build `internal/indexer.Sync()` as store-seeded incremental resolve (not a `Resolve()` subset call), add the `x/` file-owned secondary index inside `internal/graphstore` (additive, `SchemaVersion` stays 1, plus two new `File` fields for the stat pre-filter and one new `Meta` field for backfill detection), export `query.BuildReverseAdjacency` for `Sync()` to reuse (a clean, non-circular new `indexer → query` import edge), and build the watcher/daemon/lockfile/unlock/goleak-soak surface directly on `fsnotify` v1.10.1 + `go.uber.org/goleak` v1.3.0 per the verified API shapes below.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Content-hash diffing + stat pre-filter | Indexer (`internal/indexer`) | Storage (`internal/graphstore`, `File` schema fields) | Diffing logic belongs with the pipeline that already extracts hashes; the fields it compares against are storage schema |
| File-owned secondary index (prune keyspace) | Storage (`internal/graphstore`) | — | D-02 explicitly: "D-02's secondary index lives **inside** `internal/graphstore`" (archtest boundary) |
| Dependent-file detection (reverse scan) | Query (`internal/query`, exported) | Indexer (consumer) | Reuses Phase-3's `buildReverseAdjacency`; indexer imports query, not vice versa (no cycle) |
| Symbol resolution (store-seeded index) | Indexer (`internal/indexer`) | — | Same package that owns `resolveRefs`/`symbolindex.go`; extends, does not fork |
| Native filesystem watching | New watcher subsystem (`internal/watch` or similar) | Indexer (`Sync()` invocation) | fsnotify is OS-facing infrastructure, cleanly separable from graph logic |
| Debounce/coalesce | Watcher subsystem | — | Pure timer/event-batching concern, no graph knowledge needed beyond a path set |
| Daemon process + single-writer lock | New `daemon` subsystem (CLI + lockfile) | Storage (respects `GraphStore`'s single-writer invariant) | Owns the watcher + the one `GraphStore.Writer`; lockfile is process-lifecycle, not graph data |
| Staleness banner | Query (`explore`/`status` formatters) | CLI/MCP (surfaces it) | D-04a: "one formatter path" — same rule Phase 3 D-08b established |
| MCP reconnect reconcile | MCP (`internal/mcp`) triggers | Indexer (`Sync()` does the work) | MCP server calls the same `Sync()` entry `sync`/daemon use — no second reconciliation path |
| Goroutine lifecycle (leak-free) | Watcher/daemon subsystem | — | Owns every goroutine it spawns; context+WaitGroup discipline is local to that subsystem |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/fsnotify/fsnotify` | v1.10.1 (latest; CLAUDE.md says "v1.9.x line" but the module proxy's current tag is v1.10.1 — recommend latest unless a compat reason surfaces) [VERIFIED: Go module proxy] | Cross-platform native filesystem watcher (inotify/kqueue/ReadDirectoryChangesW/FEN) | Mandated by `.claude/CLAUDE.md` §fsnotify; only realistic cross-platform primitive in the Go ecosystem for native (non-polling) events |
| `go.uber.org/goleak` | v1.3.0 (latest) [VERIFIED: Go module proxy] | Test-only goroutine-leak detection (SYNC-06 soak gate) | CONTEXT D-07 names it as the recommended (executor's-discretion) leak-detection library; wide adoption, `VerifyTestMain`/`VerifyNone` API fits directly |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — no new non-test dependency beyond fsnotify) | — | — | Lockfile pid-liveness check uses only `os`/`syscall` from the standard library (see Code Examples); debounce uses `time.Timer`, no third-party timer lib needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `fsnotify` native watching | Polling walker (`filepath.WalkDir` on an interval) | Explicitly rejected by REQUIREMENTS.md ("Out of Scope: Polling as default file watching") and CLAUDE.md's "What NOT to Use" table — reject for v1 default; keep as the documented degraded fallback only |
| `go.uber.org/goleak` | Manual `runtime.NumGoroutine()` delta assertions | goleak gives named-goroutine stack traces on failure (much better soak-test diagnostics) for one small test-only dependency; a manual delta check is viable if the team wants zero new deps at all, but loses debuggability |
| Store-seeded symbol index (this research's recommendation) | Re-run full `Discover`+`Extract` over every file every sync, but only re-`Resolve`/write the delta | Simpler to implement (no store-seeding logic) but directly violates D-01a/INDX-03's "not a full re-index of a subset" — Pass 1 (parsing) would touch every file every sync regardless of what changed. Rejected. |
| CGo parser incremental reparse (`Parser.Parse(src, oldTree)`) | Full re-parse of each changed/dependent file | Incremental reparse is real in the API but unused (no call site threads `oldTree`, no byte-edit tracking exists) — building that correctly is nontrivial engineering with a real risk of diverging from a from-scratch parse, directly threatening the "sync-equals-reindex" determinism gate this phase needs to pass. Defer to a future performance phase (Phase 8 territory) once profiling shows full re-parse is the bottleneck. |

**Installation:**
```bash
go get github.com/fsnotify/fsnotify@v1.10.1
go get -t go.uber.org/goleak@v1.3.0   # test-only; go.mod records it in the require block but callers should confirm `go mod tidy` doesn't need to promote it to non-indirect for a test-only import (it will land as a direct test dependency automatically once imported from a _test.go file)
```

**Version verification:** Verified directly against the Go module proxy (authoritative for this ecosystem):
```
$ go list -m -versions github.com/fsnotify/fsnotify
... v1.8.0 v1.9.0 v1.10.0 v1.10.1   (v1.10.1 is latest)
$ go list -m -versions go.uber.org/goleak
... v1.2.1 v1.3.0                    (v1.3.0 is latest)
```

## Package Legitimacy Audit

Both packages are already-established, high-reputation, high-adoption Go modules — `fsnotify` is explicitly mandated by this project's own `.claude/CLAUDE.md` (not a WebSearch discovery), and `goleak` is an Uber OSS project with a long release history (`v0.10.0` → `v1.3.0` since 2018). Verified directly against the Go module proxy (`go list -m -versions`), the authoritative source for this ecosystem — not WebSearch or training-data guesswork. No `package-legitimacy check` seam exists for the Go ecosystem in this environment (npm/pypi/crates only); the module-proxy verification above is the ecosystem-appropriate equivalent per the package-legitimacy protocol's Step 2.

| Package | Registry | Age | Downloads/Adoption | Source Repo | Verdict | Disposition |
|---------|----------|-----|---------------------|--------------|---------|-------------|
| `github.com/fsnotify/fsnotify` | Go module proxy | ~14 yrs (fsnotify.v0 predecessor since ~2012; current module since 2018) | De facto standard — used by Kubernetes, Docker, Terraform, Prometheus, and this project's own CLAUDE.md mandate | `github.com/fsnotify/fsnotify` | OK | Approved |
| `go.uber.org/goleak` | Go module proxy | ~8 yrs (v0.10.0 2018 → v1.3.0 2024) | Uber OSS, adopted broadly across Go test suites | `github.com/uber-go/goleak` | OK | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                 codegraph daemon (D-05)                  │
                    │  ┌───────────────┐   debounce    ┌──────────────────┐    │
  filesystem   ───► │  │ fsnotify.     │──2000ms──────►│ Sync() trigger   │    │
  events (OS-       │  │ Watcher (D-04)│  timer, union  │ (union of paths) │    │
  native)           │  │ + recursive-  │  of changed    └────────┬─────────┘    │
                    │  │ add-on-Create │  paths                  │              │
                    │  └───────────────┘                         ▼              │
                    │                                  ┌──────────────────┐    │
                    │  lockfile (pid+ts) ◄──guards──────│ internal/indexer │    │
                    │  in .codegraph/                   │ .Sync()          │    │
                    │                                   └────────┬─────────┘    │
                    └────────────────────────────────────────────┼──────────────┘
                                                                  │
     ┌────────────────────────────────────────────────────────────────────────────┐
     │  Sync() incremental algorithm (Architecture Pattern 1, below)               │
     │                                                                              │
     │  1. Discover() (cheap walk)          4. Prune changed/deleted subgraphs      │
     │  2. Stat pre-filter (mtime/size)  ──►5. Reverse-scan for dependents (D-02a)  │
     │  3. Content-hash confirm             6. Extract() batch = changed ∪ deps     │
     │                                       7. Resolve against STORE-SEEDED index  │
     │                                       8. Single GraphStore.Writer commit     │
     └────────────────────────────────────────────────────┬───────────────────────┘
                                                            │
                                                            ▼
                                        ┌──────────────────────────────────────┐
                                        │  GraphStore (Pebble, internal/       │
                                        │  graphstore) — n/ e/ f/ m/ + new x/  │
                                        │  file-owned secondary index (D-02)   │
                                        └───────────────────┬───────────────────┘
                                                            │ Snapshot (lock-free read)
                    ┌───────────────────────────────────────┼───────────────────────┐
                    ▼                                       ▼                       ▼
          codegraph explore/status         MCP serve --mcp (reconnect       codegraph unlock
          (staleness banner, D-04a)        reconcile, D-06 — same          (stale-lock pid
                                            Sync() call as sync/daemon)     liveness check)
```

### Recommended Project Structure
```
internal/
├── graphstore/
│   ├── keys.go            # + prefixFileIndex 'x', fileIndexNodeKey/fileIndexEdgeKey/fileIndexPrefix builders
│   ├── store.go            # + Reader.IterateFileIndex; Writer.DeleteNode/DeleteEdge; PutEdge gains ownerPath param
│   ├── batch.go             # + DeleteNode/DeleteEdge impls; DeleteFileSubgraph extended to also range-delete x/<path>/...
│   └── pebble_store.go       # + pebbleFileIndexIterator
├── schema/
│   └── graph.proto           # + File.mtime_unix_ns=7, File.size_bytes=8; Meta.has_file_index=7 (all additive)
├── indexer/
│   ├── sync.go                # NEW: Sync() entry — store-seeded incremental resolve (Pattern 1)
│   ├── resolve.go              # refactor: resolveRefs's idx-building extracted so Sync can inject a store-seeded idx
│   ├── symbolindex.go            # + newSymbolIndexFromStore(reader, modulePath) — seeds byImportAndName from IterateNodes()
│   └── discover.go                # extract shouldSkipDir(name) helper so watcher reuses the exact same exclusion rule
├── watch/                          # NEW package: fsnotify wrapper + recursive-add-on-Create + debounce
│   ├── watcher.go
│   └── debounce.go
├── daemon/                          # NEW package: lockfile + daemon process lifecycle + unlock
│   ├── lock.go
│   └── daemon.go
├── query/
│   ├── traverse.go                    # buildReverseAdjacency → BuildReverseAdjacency (exported for indexer reuse)
│   └── status.go                       # PendingChanges/WorktreeMismatch stop being inert placeholders; staleness signal wired
└── cli/
    ├── sync.go                          # NEW: thin `codegraph sync` command
    ├── daemon.go                          # NEW: thin `codegraph daemon` command
    └── unlock.go                          # NEW: thin `codegraph unlock` command
```

### Pattern 1: Store-seeded incremental Sync() (INDX-03, the core algorithm)

**What:** `internal/indexer.Sync(repoRoot, storeDir string, opts Options) (Stats, error)` — mirrors `Run`'s signature/lifecycle (opens the store once, closes on every return path) but performs an incremental resolve instead of a from-scratch one.

**When to use:** Every invocation of `codegraph sync`, every debounced watcher/daemon cycle, and every MCP-reconnect reconcile (D-06) — one routine, three callers.

**Algorithm (concrete recommendation):**
1. Open `GraphStore`, take a `Reader` snapshot `r0` (Pebble snapshots are immune to concurrently-committing batches — a `NewSnapshot()` fixes the visible sequence number at call time [VERIFIED: pebble official docs, see Sources]).
2. `meta, err := r0.GetMeta()`. If `err == graphstore.ErrNotFound` or `!meta.GetHasFileIndex()` → run the D-02b backfill: delegate to the same full-index code path `codegraph index --force` uses (this naturally populates the new `x/` index via the extended `PutNode`/`PutEdge`), then return those `Stats`. Every subsequent `Sync()` call is incremental.
3. `files, modulePath, err := Discover(repoRoot)` — cheap (a directory walk + build-tag match, no parsing); needed every sync regardless of what changed, to detect creates/deletes.
4. **Stat pre-filter (D-01a):** for each discovered file, `os.Stat` its mtime/size and compare against `r0.GetFile(path).GetMtimeUnixNs()`/`GetSizeBytes()` (new additive `File` fields — see Pitfall 3). Files whose stat differs, or that have no stored `File` record (new), are "candidates."
5. **Content-hash confirm (D-01a):** for each candidate, read the file and compute `sha256` (reuse `goextract.Extract`'s existing hash-first-thing behavior, or a lightweight pre-hash before full extraction) and compare to `r0.GetFile(path).GetContentHash()`. Only genuinely-hash-changed files become the **changed set**, split into `added` (no prior `File` record), `modified` (hash differs), and — by diffing `r0.IterateFiles()` against the freshly `Discover`ed set — `deleted` (had a `File` record, no longer discovered).
6. **Prune (D-02):** for every `deleted`/`modified` file's path, enumerate `r0.IterateFileIndex(path)` (the new `x/` namespace scan) and stage `w.DeleteNode(id)` / `w.DeleteEdge(src,kind,dst)` for every owned record, then `w.DeleteFileSubgraph(path)` (range-deletes `f/<path>` and `x/<path>/...` in one call). Collect every deleted node id into `goneIDs`.
7. **Dependent detection (D-02a):** `rev, err := query.BuildReverseAdjacency(r0)` (export the existing unexported `buildReverseAdjacency` — see Pitfall 4). For every id in `goneIDs`, `rev[id]` gives the edges whose target is now gone; each edge's `Source` node's `FilePath` (via `r0.GetNode(edge.Source)`) joins the **dependent set** (skip if already in the changed set).
8. **Narrow-prune dependents:** for each dependent file (content unchanged, so its own nodes/file record stay), enumerate only the **edge** entries of its `x/` index and `w.DeleteEdge(...)` them (do NOT call `DeleteFileSubgraph` — that would wrongly delete its still-valid nodes and `f/` record).
9. **Extract() the reparse batch** = `added ∪ modified ∪ dependent` (deleted files are excluded — nothing to extract). Reuses `Extract(files, workers)` unchanged, just over a filtered `[]DiscoveredFile` slice.
10. **Build the merged symbol index:** `idx := newSymbolIndexFromStore(r0, modulePath)` scanning `r0.IterateNodes()`, skipping `KindFile`/`kindPackage` nodes (mirrors `newSymbolIndex`'s existing skip), computing each node's import path via `importPathFor(modulePath, node.GetFilePath())`, **excluding** any node whose `FilePath` is in the reparse batch (about to be superseded). Then overlay the batch's freshly-extracted `goextract.FileResult.Nodes` on top (same per-result loop `newSymbolIndex` already has, just seeded non-empty).
11. **Resolve** the batch's `Unresolved` refs against `idx` — extract `resolveRefs`'s per-ref resolution loop (the `switch ref.Kind` block in `resolve.go`) into a helper that accepts an externally-built `idx` parameter, so `Resolve()` (full) keeps building its own and `Sync()` injects the merged one. `collapseEdges` scoped to just the batch's newly-generated edge candidates is safe and deterministic (see Pitfall 5) — no need to re-collapse the whole graph.
12. **Write:** stage `PutNode`/`PutEdge`(with `ownerPath`)/`PutFile` for the batch through the SAME `Writer` opened in step 6 (one commit, D-01b's "never per-symbol" rule) — deleted-file pruning and batch-write land in one atomic `Commit()`. Stamp `Meta` with updated `NodeCount`/`EdgeCount` (deltas, not a full rescan — or a cheap `readGraphCounts`-style re-derivation post-commit, executor's discretion) and `last_sync_unix_ms = time.Now()`.
13. Return `Stats` (reuse the existing shape plus whatever `sync`-specific fields D-01b's summary needs — files reparsed/pruned, nodes/edges added/removed, dependents recomputed, duration).

**Example (store-seeded symbol index skeleton):**
```go
// Source: derived from internal/indexer/symbolindex.go's existing newSymbolIndex,
// internal/indexer/discover.go's importPathFor, and internal/graphstore's Reader
// interface (internal/graphstore/store.go) — no single upstream source; this is
// this research's concrete recommendation, not a copied snippet.
func newSymbolIndexFromStore(r graphstore.Reader, modulePath string, exclude map[string]bool) (*symbolIndex, error) {
	idx := &symbolIndex{byImportAndName: make(map[string]map[string]string)}
	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()
	for it.Next() {
		n := it.Node()
		if n.Kind == goextract.KindFile || n.Kind == kindPackage || exclude[n.FilePath] {
			continue
		}
		importPath := importPathFor(modulePath, n.FilePath)
		names := idx.byImportAndName[importPath]
		if names == nil {
			names = make(map[string]string)
			idx.byImportAndName[importPath] = names
		}
		names[n.Name] = n.Id
	}
	return idx, it.Err()
}
```

### Pattern 2: File-owned secondary index (D-02, the pruning keyspace)

**What:** A new additive Pebble namespace (`prefixFileIndex byte = 'x'`) mapping `path → {owned node ids, owned outgoing-edge triples}`, so `DeleteFileSubgraph`-class pruning is O(subgraph) instead of a full-graph scan.

**When to use:** Populated on every `PutNode`/`PutEdge` call (full `index` and incremental `sync` alike); consulted by the prune step of `Sync()`.

**Key shape (concrete recommendation — mirrors `keys.go`'s `appendSegment` collision-safety discipline exactly):**
```go
// Source: extends internal/graphstore/keys.go's existing appendSegment/
// rangeUpperBound primitives — same file, same discipline.
const prefixFileIndex byte = 'x' // NEW: file-owned reverse index (D-02, Phase-4 hook)

const (
	fileIndexKindNode byte = 0x01 // fixed marker byte — code-controlled, not attacker data
	fileIndexKindEdge byte = 0x02
)

func fileIndexNodePrefix(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path)+1)
	buf = append(buf, prefixFileIndex)
	buf = appendSegment(buf, path)
	return append(buf, fileIndexKindNode)
}

func fileIndexNodeKey(path, nodeID string) []byte {
	buf := fileIndexNodePrefix(path)
	return appendSegment(buf, nodeID)
}

func fileIndexEdgePrefix(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path)+1)
	buf = append(buf, prefixFileIndex)
	buf = appendSegment(buf, path)
	return append(buf, fileIndexKindEdge)
}

func fileIndexEdgeKey(path, src, kind, dst string) []byte {
	buf := fileIndexEdgePrefix(path)
	buf = appendSegment(buf, src)
	buf = appendSegment(buf, kind)
	return appendSegment(buf, dst)
}

// fileIndexPrefix bounds ALL of one file's index entries (node AND edge
// sub-entries together — 0x01 sorts before 0x02, but both fall inside
// [fileIndexPrefix(path), rangeUpperBound(...)) since appendSegment's
// length prefix isolates path from the marker byte and everything after
// it, exactly like fileSubgraphPrefix's existing collision-safety argument.
func fileIndexPrefix(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path))
	buf = append(buf, prefixFileIndex)
	return appendSegment(buf, path)
}
```

**Writer interface additions (Go-API-level change only — no wire-format/`SchemaVersion` impact):**
- `PutNode(n *schema.Node) error` — **unchanged signature**; `pebbleWriter.PutNode` internally *also* stages `fileIndexNodeKey(n.FilePath, n.Id)` when `n.FilePath != ""` (skips the `kindPackage` pseudo-node, which has no owning file).
- `PutEdge(e *schema.Edge, ownerPath string) error` — **signature change** (blast radius: exactly one existing call site, `resolve.go`'s `writeGraph`, which already computes `nodeFilePath[e.Source]` for `collapseEdges`' tiebreak — trivial to thread through). `pebbleWriter.PutEdge` stages `fileIndexEdgeKey(ownerPath, e.Source, e.Kind, e.Target)` when `ownerPath != ""`.
- `DeleteNode(id string) error` / `DeleteEdge(source, kind, target string) error` — **new** point-delete methods (today's `Writer` has no delete-by-id at all, only the range-delete `DeleteFileSubgraph`). Needed so `Sync()`'s prune step can delete individually-enumerated `n/`/`e/` records found via the `x/` index scan.
- `DeleteFileSubgraph(path string) error` — **extended implementation**: still one Writer method, now issues **two** `DeleteRange` calls internally (unchanged `f/<path>` range, plus the new `x/<path>/...` range) — still a single logical "prune this file entirely" call from the caller's perspective.
- `Reader.IterateFileIndex(path string) (FileIndexIterator, error)` — **new** Reader method, a contiguous scan over `[fileIndexPrefix(path), rangeUpperBound(fileIndexPrefix(path)))` yielding typed entries (`IsNode bool; NodeID string; Source, Kind, Target string`) so `Sync()`'s prune/narrow-prune steps can decode what to delete.

### Pattern 3: fsnotify recursive watch + debounce (SYNC-01)

**What:** fsnotify does **not** recurse — `Add(dir)` only watches that directory's direct children [VERIFIED: fsnotify official docs]. Recursive coverage requires walking the tree at startup and calling `Add` on every directory, plus re-`Add`ing any newly-created directory discovered via a `fsnotify.Create` event.

**When to use:** `codegraph daemon` and the in-process `serve` fallback watcher, both.

**Example (recursive-add-on-Create + debounce skeleton):**
```go
// Source: fsnotify official docs (Context7 /fsnotify/fsnotify) for the
// Watcher API shape; the recursive-add loop and debounce coalescing are
// this research's synthesis for this project's requirements (SYNC-01,
// D-04) — fsnotify does not ship a recursive-watch helper itself.
func addRecursive(w *fsnotify.Watcher, root string, skip func(name string) bool) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skip(d.Name()) { // reuse discover.go's exclusion predicate
				return filepath.SkipDir
			}
			return w.Add(p)
		}
		return nil
	})
}

func watchLoop(ctx context.Context, w *fsnotify.Watcher, debounce time.Duration, sync func(paths map[string]struct{})) {
	pending := make(map[string]struct{})
	var timer *time.Timer
	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = addRecursive(w, ev.Name, skipDir) // new subdir: start watching it too
				}
			}
			pending[ev.Name] = struct{}{}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() { sync(pending); pending = make(map[string]struct{}) })
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
			// log, do not crash the loop (RESEARCH Pitfall — see Common Pitfalls)
		}
	}
}
```

**Debounce default/env-tunable (D-04):**
```go
const defaultDebounceMs = 2000

func debounceDuration() time.Duration {
	if v := os.Getenv("CODEGRAPH_DEBOUNCE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultDebounceMs * time.Millisecond
}
```

**Ignore-rule agreement (D-04's "honor the same discovery exclusions"):** `discover.go`'s actual exclusion logic is `d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".")` (which already covers `.codegraph/` — it is dot-prefixed) plus `go/build.Context.MatchFile` build-tag filtering. There is **no `.gitignore` parser anywhere in this codebase today** — recommend extracting `discover.go`'s directory-skip predicate into a small exported/shared helper (e.g. `shouldSkipDir(name string) bool`) that both `Discover`'s `WalkDir` callback and the watcher's `addRecursive` call, so the two never silently diverge. Do not introduce a `.gitignore`-parsing library — CONTEXT's "honor .gitignore" phrasing means "match discover.go's existing exclusions," not "add real gitignore semantics."

### Pattern 4: Staleness banner threading (SYNC-02, D-04a)

**What:** `explore`'s markdown output (`RenderExplore`, `internal/query/render_markdown.go`) and `status`'s JSON/plain output (`internal/query/status.go`) both need a staleness signal. `StatusResult` already has inert placeholder fields for exactly this (`PendingChanges{Added,Modified,Removed}`, `WorktreeMismatch *string`) stamped "present-but-inert" in Phase 3 — Phase 4 makes them **live**.

**When to use:** Every `explore`/`status` call, CLI and MCP alike (one formatter path per D-08b).

**Recommended mechanism:** the watcher/daemon writes a lightweight, cheap-to-check pending-sync marker (e.g. a `Meta`-adjacent flag stamped via a tiny separate write, or — simpler, no extra store write — a small sidecar file `.codegraph/.sync-pending` touched on first debounced event and removed on commit) that `Status()`/`Explore()` check before rendering. **Fallback signal when no daemon is running** (per D-04a): compare the newest on-disk file mtime under the repo against `Meta.last_sync_unix_ms` — if any file is newer, render the banner even without a live watcher. Recommend prepending a single bolded line to `RenderExplore`'s output (`"**⚠ Index may be stale — a sync is pending.**\n\n"` or similar; exact text is executor's discretion per CONTEXT) before the existing `"**Exploration: %s**"` header, and adding a `stale`/`pendingSync bool` field to `StatusResult` (replacing/supplementing the current inert `PendingChanges`/`WorktreeMismatch` placeholders — those were explicitly documented in `query/status.go`'s doc comment as "Phase-4 sync concept; present-but-inert placeholder," this is the phase that makes them real).

### Pattern 5: MCP reconnect reconciliation (SYNC-03, D-06)

**What:** `internal/mcp.BuildServer` currently does no reconciliation — every tool handler opens a fresh `query.OpenAt` snapshot per call (D-02/D-08b) but never checks staleness against the filesystem. D-06 adds a `Sync()` call at server startup, **before** the first tool call is served.

**When to use:** `codegraph serve --mcp` startup (`internal/cli/serve.go`'s `newServeCmd`), once, before `server.ServeStdio(s)` blocks.

**Recommended integration point:** in `newServeCmd`'s `RunE`, after resolving `hasIndex`/`repoPath` and before calling `mcp.BuildServer`, invoke the same `indexer.Sync(repoPath, storeDir, opts)` entry `codegraph sync` uses (D-01/D-06: "no second reconciliation code path"). A no-op sync (nothing changed) must be cheap — the stat pre-filter (Pattern 1 step 4) makes this true by construction, since no file's mtime/size will have changed.

### Pattern 6: Daemon + lockfile + unlock (SYNC-04, SYNC-05, D-05)

**What:** A pidfile-style lock in `.codegraph/` (e.g. `.codegraph/daemon.lock`) storing at minimum the owning process's pid and a start timestamp, guarding the single-writer invariant across multiple agent sessions sharing one daemon.

**Example (pid-liveness stale-lock check, POSIX):**
```go
// Source: pattern synthesized from nightlyone/lockfile and trbs/pid prior
// art (WebSearch, MEDIUM confidence — see Sources) using only Go stdlib
// os/syscall, no new dependency.
type lockInfo struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}

func isStale(l lockInfo) bool {
	proc, err := os.FindProcess(l.PID)
	if err != nil {
		return true // no such process — definitely stale
	}
	// On POSIX, FindProcess always succeeds; liveness is proven by signal 0.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return true // ESRCH or EPERM-on-dead-pid — treat as stale
	}
	return false
}
```

**Caveat (flag explicitly to the planner):** bare PID liveness has a real collision risk in containers, where PID namespaces restart numbering at 1 [MEDIUM confidence, WebSearch]. Recommend corroborating with the stored `StartedAt` timestamp — if a live process with that PID exists but its own process-start-time (platform-specific to obtain; on Linux, `/proc/<pid>/stat`'s start-time field) predates `StartedAt`, or is simply "implausibly different," treat the lock as stale too. This corroboration is executor's discretion per CONTEXT ("Lockfile format/contents beyond pid+timestamp") but should not be skipped silently — document the residual risk if the simpler pid-only check is chosen for v1.

**`codegraph unlock`:** reads the lockfile, runs `isStale`, and only removes the lockfile if stale — never blindly deletes (would stomp a live daemon). Errors clearly if the lock is live ("daemon still running, pid=%d — stop it first").

**In-process fallback (D-05):** where a separate daemon process is unsupported/undesired, `serve` runs the identical watcher+debounce+`Sync()` loop **inside its own process**, still acquiring the same lockfile (so a `daemon` and an in-process `serve` watcher can never both hold the writer simultaneously) — same single-writer discipline, different process topology.

### Pattern 7: Leak-free lifecycle (SYNC-06, D-07)

**What:** every goroutine the watcher/daemon spawns (the fsnotify event-consumption loop, the debounce timer's `AfterFunc` callback goroutine, any `Sync()`-invocation goroutine) is owned by a single cancelable `context.Context` and joined via `sync.WaitGroup` before the daemon/watcher's top-level `Stop()`/`Close()` returns.

**Example (soak-test shape):**
```go
// Source: go.uber.org/goleak official docs (Context7 /uber-go/goleak).
func TestWatcherSoak(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	d := newDaemon(ctx, tmpRepo)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); d.Run(ctx) }()

	for i := 0; i < soakIterations; i++ {
		touchFile(t, tmpRepo, fmt.Sprintf("f%d.go", i))
		time.Sleep(debounceWindow + slack)
	}

	cancel()
	wg.Wait() // MUST complete before goleak.VerifyNone runs, or every
	          // still-shutting-down goroutine is a false-positive leak
}
```
`time.AfterFunc`'s callback runs in its OWN goroutine outside the caller's `context` tree by default — the debounce timer's callback must itself check `ctx.Err()` before doing work, and the watcher's shutdown path must call `timer.Stop()` (not just let it fire into a closed channel) to avoid a leaked/late-firing timer goroutine racing `VerifyNone`.

### Anti-Patterns to Avoid
- **Calling `Resolve()` with a subset of `results` for `Sync()`:** breaks cross-file resolution for every symbol declared in a file NOT in that subset — `newSymbolIndex` only sees what's handed to it. Always seed from the store (Pattern 1).
- **Treating "dependent files" as needing only Pass-2 re-resolution, skipping Pass-1 extraction:** `Unresolved` refs are never persisted; without re-extracting, there is nothing to re-resolve. Extract dependents too — the batch stays bounded by dependency fan-in, not the whole repo.
- **A `.gitignore`-parsing library for watcher/discoverer agreement:** neither `discover.go` nor the watcher needs real gitignore semantics — reuse the existing dot-prefix/`vendor` predicate, don't add scope.
- **Polling as the default watcher:** explicitly rejected by REQUIREMENTS.md and CLAUDE.md; only acceptable as a documented, non-default degraded fallback.
- **Tree-sitter incremental reparse without edit-tracking infrastructure:** the `Parser.Parse(src, oldTree)` path exists but nothing computes byte-range edits today; using it half-built risks silently diverging parse trees from a from-scratch parse, which is exactly what the sync-equals-reindex determinism gate exists to catch.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Cross-platform filesystem events | A custom per-OS syscall watcher (inotify/kqueue/ReadDirectoryChangesW bindings) | `fsnotify` | Exactly the problem it solves; already the mandated project dependency |
| Goroutine-leak detection in soak tests | Manual goroutine-count-diffing with sleep-and-hope timing | `go.uber.org/goleak` | Named stack traces on failure, `IgnoreTopFunction`/`IgnoreAnyFunction` for known-benign background goroutines, `VerifyTestMain` handles `t.Parallel()` correctly |
| Point-in-time consistent reads during concurrent writes | Manual locking/copy-on-write of the graph | Pebble's `NewSnapshot()` (already the project's Reader implementation) | Sequence-number-based MVCC snapshot is exactly this guarantee, already wired through `GraphStore.Snapshot()` |
| Reverse call-graph adjacency | A second bespoke reverse-edge scanner inside `internal/indexer` | `query.BuildReverseAdjacency` (export the existing Phase-3 `buildReverseAdjacency`) | D-02a's explicit instruction: reuse, don't duplicate; avoids two implementations drifting |

**Key insight:** almost everything this phase needs already exists as an unused hook (`DeleteFileSubgraph`'s Phase-4 note, `File.content_hash`/`Meta.last_sync_unix_ms` fields, the reverse-adjacency scan) or a single well-audited external library (`fsnotify`, `goleak`) — the actual net-new engineering is the `x/` secondary index and the store-seeded symbol index, both scoped inside packages that already own the relevant concerns.

## Common Pitfalls

### Pitfall 1: `newSymbolIndex(results)` cannot be reused unmodified for `Sync()`
**What goes wrong:** Calling `resolveRefs`/`Resolve()` with only the changed-file batch silently fails to resolve any reference to a symbol declared in an unchanged file — `resolveUnqualified`/`resolveSelector` only see what's in `results`.
**Why it happens:** `newSymbolIndex` builds `byImportAndName` purely from its `results []goextract.FileResult` argument; it has no notion of "the rest of the graph."
**How to avoid:** Seed the index from the store (`IterateNodes()` + `importPathFor`), overlay the batch (Pattern 1 step 10).
**Warning signs:** A fixture test where an unchanged file calls into a changed file (or vice versa) starts reporting spurious `unresolvedCount` increases after a `sync`, even though a from-scratch `index` resolves the same call cleanly.

### Pitfall 2: Dependent files need Pass-1 re-extraction, not just Pass-2 re-resolution
**What goes wrong:** Interpreting D-01a's "only files whose hash changed are reparsed" as literally excluding dependent files from `Extract()` leaves nothing to resolve — `Unresolved` refs are transient, never persisted.
**Why it happens:** `goextract.FileResult.Unresolved` lives only in memory during one `Resolve()` call; the store only ever holds successfully-resolved edges.
**How to avoid:** Include dependent files (content unchanged) in the `Extract()` batch alongside changed files; only their own `File` record's `ContentHash`/`NodeCount` stay unchanged on write (their `EdgeCount` may still change).
**Warning signs:** A rename-fixture test where file B calls a function in renamed file A: after `sync`, B's call edge is missing entirely (not even pointing at a stale id) — because B was never re-extracted, so its `Unresolved` ref pointing at A's old symbol name was never regenerated to re-resolve against A's new node id.

### Pitfall 3: `File` has no mtime/size fields — the stat pre-filter is unimplementable as-is
**What goes wrong:** D-01a's "stat pre-filter → content-hash confirm" needs something cheap to compare against; `schema.File` today only has `path, content_hash, language, node_count, edge_count, errors`.
**Why it happens:** These fields were never needed until Phase 4's incremental diffing.
**How to avoid:** Add `int64 mtime_unix_ns = 7;` and `int64 size_bytes = 8;` to `File` (additive, `SchemaVersion` stays 1 — matches `graph.proto`'s own additive-only doctrine), stamped by both `index` and `sync` on every `PutFile`.
**Warning signs:** If skipped, `sync` degrades to "hash every file every time" — still correct, but re-reads and re-hashes every file's full content on every sync, defeating the stat pre-filter's whole purpose (avoiding I/O on unchanged files at monorepo scale — relevant even though INDX-06's 100k-file scale gate itself is Phase 8).

### Pitfall 4: `buildReverseAdjacency` is unexported — a straight `internal/indexer` import won't compile
**What goes wrong:** `internal/query/traverse.go`'s `buildReverseAdjacency` (lowercase) is package-private; `internal/indexer` cannot call it without exporting it.
**Why it happens:** Phase 3 had no cross-package reuse need for this function yet.
**How to avoid:** Rename to `BuildReverseAdjacency` (exported), update its 4 existing call sites within `internal/query` (`explore.go`, `node.go`, `traverse.go`'s own `Callers`/`Impact`/`Affected`). Confirmed non-circular: `internal/query` already imports `internal/indexer/goextract` (a *different* package from `internal/indexer` itself), and `internal/indexer` imports nothing from `internal/query` today — `internal/indexer → internal/query` is a new, safe edge.
**Warning signs:** A build error (`buildReverseAdjacency undefined` or `unexported`) is the loud failure mode here — low risk of silent misbehavior, but flag it early in planning so the export rename lands in the same wave as `Sync()`'s first use of it.

### Pitfall 5: Scoping `collapseEdges` to just the reparse batch is safe — but only because node ids are globally unique per declaring file
**What goes wrong (if NOT understood):** A planner might assume `collapseEdges` must run over the WHOLE graph's edges every sync (expensive, and re-touches unchanged files' already-committed edges unnecessarily).
**Why it's actually fine to scope it:** `edgeTriple{source, kind, target}` — `source` is always a content-hash node id that belongs to exactly one file (the file that declares that symbol). Since `collapseEdges`' whole purpose is deduping multiple *candidate* edges for the same triple (e.g. two call-sites at different lines targeting the same callee from the same source), and all candidates for a given `source` can only ever be generated by extracting **that source's own declaring file**, scoping `collapseEdges` to the reparse batch's own generated candidates produces an identical, deterministic per-source pick to what a full-graph collapse would — because no unchanged file could ever contribute a competing candidate for a `source` node it doesn't declare.
**Warning signs:** None if implemented as scoped — this is a correctness argument to include in code comments/plan rationale, not a live bug risk, but worth stating explicitly so a future contributor doesn't "fix" it into an unnecessary full-graph re-collapse.

### Pitfall 6: `w.Errors` channel on `fsnotify.Watcher` must be drained, or the watcher deadlocks
**What goes wrong:** fsnotify's `Errors` channel is unbuffered/small; if nothing reads from it, an internal error (e.g. a removed watch target) can block the watcher's internal goroutine indefinitely.
**Why it happens:** Standard Go channel-consumer discipline — both `Events` and `Errors` must be serviced in the same `select` loop (see Pattern 3's example).
**How to avoid:** Always `select` on both channels in the watch loop; never `range w.Events` alone.
**Warning signs:** A soak test that stops receiving new `Events` after the watcher hits ANY internal error (e.g. a watched directory gets deleted out from under it) — the leak manifests as a stuck-not-crashed watcher goroutine, which `goleak` will catch as a still-running goroutine at test end.

## Code Examples

Verified patterns from official sources — see Patterns 3, 6, 7 above for the full skeletons; source citations are inline in each code block's comment header.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `Writer.DeleteFileSubgraph` prunes only `f/<path>` (Phase 1 stub) | Extended to also range-delete/point-delete the file's owned `n/`/`e/` records via a new `x/` secondary index | This phase (Phase 4, D-02) | Closes the deliberately-deferred pruning gap documented since `keys.go:113-118` |
| `PendingChanges`/`WorktreeMismatch` in `StatusResult` are inert always-zero/null placeholders | Live signals reflecting actual pending-sync state | This phase (D-04a) | `status` becomes agent-actionable for staleness, matching TS parity intent |
| No `mtime`/`size` on `File` | Additive `mtime_unix_ns`/`size_bytes` fields | This phase (D-01a) | Enables the cheap stat pre-filter; without it every sync must hash every file |

**Deprecated/outdated:** none — this phase extends existing hooks rather than replacing prior design.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Recommending latest fsnotify v1.10.1 over CLAUDE.md's literally-stated "v1.9.x line" | Standard Stack | Low — CLAUDE.md's stack research predates the current registry state; v1.9.x → v1.10.x is a routine patch/minor bump with no known breaking API change for the `NewWatcher`/`Add`/`Events`/`Errors` surface this phase uses. Confirm with the user only if strict pinning to v1.9.x is actually intended. |
| A2 | Exact `x/` namespace key-byte choice (`'x'`) and `PutEdge` signature change (adding `ownerPath`) | Architecture Patterns Pattern 2 | Low-medium — this is explicitly Claude's-discretion territory per CONTEXT ("exact secondary-index key shape... executor's discretion"); an alternative shape (e.g. a single combined node+edge entry format, or a different marker-byte scheme) would also satisfy D-02's constraints. The concrete shape here is a recommendation, not a locked fact. |
| A3 | PID-liveness-only stale-lock detection may be insufficient in containerized environments (PID reuse) | Architecture Patterns Pattern 6 | Medium if this project's deployment targets containers with PID namespace resets — a corroborating start-timestamp check is recommended but left to executor discretion; if skipped, a rare false-negative (treating a live daemon's lock as stale) is possible. |
| A4 | Full re-parse (not tree-sitter incremental reparse) is the correct v1 choice for changed files | Standard Stack, Architecture Patterns anti-patterns | Low — this is a recommendation grounded in the observed absence of edit-tracking infrastructure and the phase's own determinism gate (sync-equals-reindex), explicitly named as executor's discretion in CONTEXT ("whether to leverage the CGo parser's incremental-reparse path... or full-reparse changed files"). |

**If this table is empty:** N/A — see entries above.

## Open Questions

1. **Should the `x/` file-index's edge sub-entries be keyed by the file's `File.path` (source-of-truth path string) or should node/edge index entries instead simply carry a back-pointer stored as the VALUE of a `n/<id>`/`e/<...>` record (a "which file owns me" field on Node/Edge themselves) instead of a separate index namespace?**
   - What we know: `schema.Node` already HAS a `FilePath` field (used for exactly this purpose in `resolve.go`'s `nodeFilePath` map already) — so node ownership is already recoverable without ANY new index, by scanning `n/` and filtering. The genuinely new problem is EDGE ownership (edges have no `FilePath` field), and the genuinely new problem is making prune **efficient** (avoiding a full `n/`/`e/` scan per prune) — a separate `x/` index solves both by giving a *contiguous, per-file* enumeration.
   - What's unclear: whether adding an `owner_path` field directly onto `schema.Edge` (additive, field 8) instead of/in addition to a separate `x/` edge-index would simplify things — it would still require the SAME `x/`-style contiguous secondary index to make "find all edges owned by path" efficient (an `owner_path` field on Edge alone doesn't give a Pebble-native contiguous range without also indexing on it), so it doesn't eliminate the need for `x/`, just adds a redundant field.
   - Recommendation: keep the `x/` secondary index as the sole source of ownership truth (avoids storing the same fact twice); this research's Pattern 2 is the recommended concrete shape but the planner should treat the exact encoding as flexible per CONTEXT's stated discretion.

2. **Where exactly should the staleness sidecar/flag live — a file on disk (`.codegraph/.sync-pending`), or a `Meta`-adjacent Pebble key?**
   - What we know: `Meta` is only written on a full `Writer.Commit()` — using a Pebble key for "pending" would require either a SEPARATE tiny writer/commit outside the main sync transaction (cheap, but adds a second write path) or overloading `Meta` itself with a `pending` bool that's flipped independently of a full graph write.
   - What's unclear: whether a plain sidecar file is simpler/more robust (survives daemon crash cleanly — an orphaned "pending" marker just means the NEXT `status`/`explore` call renders a (harmless, if slightly stale) banner until the next successful sync) versus a Pebble-native signal (more "in the store" but requires more careful write-path plumbing).
   - Recommendation: sidecar file for v1 — simplest, no risk of corrupting the `Meta` record's write-once-per-commit discipline, and the `last_sync_unix_ms`-vs-newest-mtime fallback (Pattern 4) already covers the case where the sidecar itself goes stale/missing.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Whole project | ✓ | go 1.26.5 (per `go.mod`) | — |
| `github.com/fsnotify/fsnotify` | SYNC-01 native watching | Not yet in `go.mod` — must be added | v1.10.1 latest [VERIFIED: Go module proxy] | Polling fallback only if native events genuinely unsupported on a target platform (documented degraded path, never default) |
| `go.uber.org/goleak` | SYNC-06 soak-test leak detection | Not yet in `go.mod` — must be added (test-only) | v1.3.0 latest [VERIFIED: Go module proxy] | `runtime.NumGoroutine()` delta assertion (less diagnosable, zero new deps) |
| `cockroachdb/pebble/v2` | Storage substrate (already present) | ✓ | v2.1.6 (per `go.mod`) | — |
| POSIX signal-based PID liveness (`syscall.Signal(0)`) | Lockfile stale-detection (Pattern 6) | ✓ (Linux/macOS); Windows needs a different liveness check (`OpenProcess`/`golang.org/x/sys/windows`) since POSIX signal semantics don't apply | stdlib `os`/`syscall` | On Windows, use `os.FindProcess` + a process-handle-open check instead of `Signal(0)` — flag as a per-OS branch the planner must account for, not a single cross-platform snippet |

**Missing dependencies with no fallback:** none — both new dependencies (`fsnotify`, `goleak`) are simply not yet added to `go.mod`, which is expected (this is their introduction phase).

**Missing dependencies with fallback:** fsnotify's native-watch path has a documented (non-default) polling fallback if a target platform's native backend proves unsupported/unstable in soak testing.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (`go test`) — matches every existing Phase 1-3 test file's convention (`*_test.go`, no external test framework) |
| Config file | none — `go test ./...` is the project's existing convention (no `go.mod` test-runner config beyond the module itself) |
| Quick run command | `go test ./internal/indexer/... ./internal/graphstore/... -run TestSync -count=1` |
| Full suite command | `go test ./... -race -count=1` (existing project convention per Phase 2's determinism-gate note: "verified under -race with GOMAXPROCS(8)") |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INDX-03 | `sync` reparses only changed files, dependent-edge recomputation | unit + property | `go test ./internal/indexer/... -run TestSync -x` | ❌ Wave 0 (`internal/indexer/sync_test.go`) |
| INDX-03 | sync-equals-reindex determinism (byte-identical `Export()` after normalizing `last_sync_unix_ms`) | property | `go test ./internal/indexer/... -run TestSyncEqualsReindex` | ❌ Wave 0 |
| INDX-04 | Rename/delete/move fixture suite — no orphaned nodes/dangling edges | fixture-driven integration | `go test ./internal/indexer/... -run TestPruneFixtures` | ❌ Wave 0 (extend `internal/indexer/testdata/gofixture`) |
| SYNC-01 | Debounced watcher coalesces a burst of edits into one sync | integration (real fsnotify against a temp dir) | `go test ./internal/watch/... -run TestDebounce` | ❌ Wave 0 (`internal/watch/` new package) |
| SYNC-02 | Staleness banner appears in `explore`/`status` while pending | unit (formatter-level, no real watcher needed) | `go test ./internal/query/... -run TestStalenessBanner` | ❌ Wave 0 |
| SYNC-03 | MCP reconnect reconciles offline changes before serving | integration | `go test ./internal/mcp/... -run TestReconnectReconcile` | ❌ Wave 0 |
| SYNC-04 | `daemon` shared watch/index server, in-process fallback | integration | `go test ./internal/daemon/... -run TestDaemonSharedWriter` | ❌ Wave 0 (`internal/daemon/` new package) |
| SYNC-05 | `unlock` clears only genuinely-stale locks | unit | `go test ./internal/daemon/... -run TestUnlockStaleOnly` | ❌ Wave 0 |
| SYNC-06 | Goroutine-leak-free soak | soak (longer-running, still `go test`) | `go test ./internal/watch/... ./internal/daemon/... -run TestSoak -timeout 120s` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** targeted package test (e.g. `go test ./internal/indexer/... -run TestSync`)
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** full suite green (including the soak test) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/indexer/sync_test.go` — covers INDX-03 (Sync algorithm, store-seeded index, dependent recomputation)
- [ ] `internal/indexer/sync_determinism_test.go` — covers INDX-03's sync-equals-reindex property gate
- [ ] `internal/indexer/prune_fixtures_test.go` + extended `testdata/gofixture` fixtures for create/modify/delete/rename/move — covers INDX-04
- [ ] `internal/graphstore/fileindex_test.go` — covers the new `x/` namespace's key encoding + `IterateFileIndex`/`DeleteNode`/`DeleteEdge` (mirrors `keyenc_test.go`'s existing collision-safety test style)
- [ ] `internal/watch/watcher_test.go`, `internal/watch/debounce_test.go` — new package, covers SYNC-01
- [ ] `internal/query/status_staleness_test.go`, extend `render_markdown_test.go` — covers SYNC-02
- [ ] `internal/mcp/reconnect_test.go` — covers SYNC-03
- [ ] `internal/daemon/lock_test.go`, `internal/daemon/daemon_test.go` — new package, covers SYNC-04/SYNC-05
- [ ] `internal/watch/soak_test.go` (or a shared `internal/daemon/soak_test.go`) with `goleak.VerifyTestMain` — covers SYNC-06
- [ ] Dependency install: `go get github.com/fsnotify/fsnotify@v1.10.1 && go get -t go.uber.org/goleak@v1.3.0`

## Security Domain

`.planning/config.json` has `security_enforcement: true`, `security_asvs_level: 1` — this section is required.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Local single-user daemon, no auth surface (v2/SERVER-01 territory) |
| V3 Session Management | No | No session concept in a local CLI/daemon |
| V4 Access Control | No | Filesystem permissions are the OS's job; no in-app access control layer |
| V5 Input Validation | Yes | Watcher-supplied file paths flow into the SAME key-encoding path (`fileKey`/`nodeKey`/new `fileIndexNodeKey`) already hardened against crafted-path key-forging (`keys.go`'s `appendSegment` discipline, T-01-02 mitigation) — no new validation gap, but confirm the new `x/` key builders reuse `appendSegment`, never raw string concatenation (Pattern 2 already specifies this) |
| V6 Cryptography | Yes | `File.content_hash` MUST stay SHA-256 (never MD5) for the stat-confirm step, per the existing `graph.proto` doc comment's V6 mandate — Phase 4 reuses, does not re-derive, this hash |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A crafted/symlinked path fed into the watcher escaping the repo root, or a `..`-containing relative path reaching `os.Stat`/`os.ReadFile` outside the confined root | Tampering (path traversal) | Reuse `internal/query/engine.go`'s existing repo-root-confinement pattern (`NewWithRoot`'s on-disk source-read confinement, T-03-06-Path) for any new filesystem access the watcher/sync path performs; never trust a raw fsnotify event path without joining/validating it against the watched root |
| A daemon crash leaving a stale lockfile that a naive `unlock` blindly removes, letting a second writer start concurrently with a still-live daemon (data corruption via two writers) | Tampering / Denial of Service | `unlock`'s pid-liveness check (Pattern 6) — never unconditionally delete the lockfile |
| A malicious/adversarial file burst (e.g. thousands of rapid creates) exhausting fd/watch limits (inotify `max_user_watches`, kqueue fd-per-file) causing the watcher to silently stop receiving events | Denial of Service | Document the platform-specific watch/fd limits (Sources) in the daemon's error path; surface a clear error/log line rather than silently degrading to "watcher stopped working" — this is an operational hardening item, not a code fix this phase must build, but the failure mode should be visible, not silent |
| A crafted file path/id colliding across the new `x/` namespace's segment boundaries, bleeding one file's prune range into an unrelated file's records | Tampering | `appendSegment`'s length-prefix discipline (already used throughout `keys.go`/`nodeid.go`) applied identically to the new `x/` key builders — Pattern 2's example already follows this; do not hand-roll a `'/'`-delimited concatenation for the new namespace |

## Sources

### Primary (HIGH confidence)
- `/fsnotify/fsnotify` (Context7) — `NewWatcher`/`NewBufferedWatcher`/`Add`/`Close` API shapes, recursive-non-recursion behavior, Events/Errors channel semantics
- `/uber-go/goleak` (Context7) — `VerifyNone`/`VerifyTestMain` usage, `t.Parallel()` interaction guidance, `IgnoreTopFunction`/`IgnoreAnyFunction`
- `/cockroachdb/pebble` (Context7) — `NewSnapshot()` point-in-time consistency guarantee, `NewBatch` vs `NewIndexedBatch`, Batch concurrent-use constraint
- Direct codebase reads (this session): `internal/graphstore/{keys.go,store.go,batch.go,pebble_store.go}`, `internal/schema/{graph.proto,meta.go}`, `internal/indexer/{pipeline.go,discover.go,extract.go,resolve.go,symbolindex.go,nodeid/nodeid.go}`, `internal/indexer/goextract/{types.go,goextract.go}`, `internal/parser/{parser.go,cgo/parser_cgo.go}`, `internal/query/{engine.go,traverse.go,explore.go,status.go,render_markdown.go}`, `internal/cli/{status.go,serve.go,index.go,root.go,init.go,uninit.go,query.go}`, `internal/mcp/server.go` — every claim about the CURRENT codebase's shape (node id preimage, absence of gitignore parsing, `Writer`/`Reader` interfaces, `File`/`Meta` proto fields, CLI thin-delegation pattern) is grounded here, not inferred
- `go list -m -versions github.com/fsnotify/fsnotify` / `go list -m -versions go.uber.org/goleak` against the Go module proxy — authoritative version verification for this ecosystem

### Secondary (MEDIUM confidence)
- WebSearch: fsnotify per-OS watch-limit/directory-event-semantics caveats (inotify `max_user_watches`, kqueue fd-per-file, ReadDirectoryChangesW coalescing, `ErrEventOverflow`) — cross-referenced against the official `pkg.go.dev`/GitHub README search-result summaries, not independently re-verified against the raw README text this session
- WebSearch: Go pidfile/stale-lock pid-liveness pattern (`nightlyone/lockfile`, `trbs/pid` prior art) — community pattern, not an official spec; the PID-reuse-in-containers caveat is this research's own risk analysis layered on top

### Tertiary (LOW confidence)
- none — every claim in this document is either grounded in a direct codebase read, an official-docs Context7 fetch, an authoritative registry query (`go list -m`), or explicitly flagged in the Assumptions Log

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — both new dependencies verified against the Go module proxy directly, not inferred from training data
- Architecture (Sync algorithm, secondary index design): HIGH — grounded directly in reading every relevant existing source file this session; the store-seeded-symbol-index and dependent-re-extraction findings are non-obvious but derived from the actual code's contracts (`newSymbolIndex`'s input shape, `Unresolved` not being persisted), not speculation
- Pitfalls: HIGH — each pitfall traces to a specific, cited code behavior (e.g. `symbolindex.go`'s `byImportAndName` construction, `goextract/types.go`'s `Unresolved` field never appearing in any Pebble write path)
- fsnotify/goleak/pebble API details: HIGH — Context7 official docs, cross-checked against this session's own code reads for how the project already uses `pebble.Snapshot`/`Batch`
- Daemon/lockfile pattern: MEDIUM — WebSearch-sourced community pattern (no single authoritative Go stdlib API for this), with an explicitly flagged residual risk (PID reuse) rather than presented as settled fact

**Research date:** 2026-07-11
**Valid until:** ~30 days for the architectural findings (codebase-grounded, stable); ~7 days for the exact fsnotify/goleak version pins (fast-moving enough that a newer patch release is plausible within that window — re-verify via `go list -m -versions` before implementation if this research is more than a couple weeks old)
