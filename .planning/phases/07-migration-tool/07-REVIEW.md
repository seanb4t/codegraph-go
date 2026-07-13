---
phase: 07-migration-tool
reviewed: 2026-07-13T01:37:23Z
depth: deep
files_reviewed: 13
files_reviewed_list:
  - internal/migrate/migrate.go
  - internal/migrate/reader.go
  - internal/migrate/translate.go
  - internal/migrate/progress.go
  - internal/migrate/swap.go
  - internal/migrate/validate.go
  - internal/migrate/migratetest/fixture.go
  - internal/cli/migrate.go
  - internal/cli/root.go
  - internal/graphstore/store.go
  - internal/graphstore/batch.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/keys.go
findings:
  critical: 0
  warning: 5
  info: 5
  total: 10
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-07-13T01:37:23Z
**Depth:** deep (cross-file call-chain + import-graph analysis)
**Files Reviewed:** 13
**Status:** issues_found

## Summary

The Phase 7 migration tool is, on the whole, unusually disciplined for its I/O-heavy,
partial-recovery profile. I verified the specific defect classes this project has been
bitten by before, and the high-risk ones are handled correctly:

- **Silent I/O truncation:** every `database/sql` `rows.Next()` loop is followed by a
  `rows.Err()` check (`ScanTable`, `presentColumns`, fixture `closeReferentialGaps`).
  Every iterator loop checks `.Err()`. No swallowed truncation found.
- **Read-only source:** `OpenSource` uses `mode=ro&_pragma=query_only(1)`; the source is
  never written and no `-wal`/`-shm` sidecar is created (D-08 holds).
- **Checkpoint ordering (D-06):** data batch commits *before* the progress cursor, so the
  cursor never advances ahead of durable data; a crash re-scans idempotently (same keys).
- **Field-mapping fidelity (D-01/D-02/D-05):** verified `wantedColumns` against the real
  `testdata/golden/ts-schema.sql` DDL. `start_column`/`end_column` are carried (StartCol/
  EndCol), `line`/`col` carried, nullable columns coerce to proto zero, dropped TS-only
  columns (`is_async`/`is_static`/`is_abstract`/`decorators`/`type_parameters`/`updated_at`/
  `indexed_at`) match the documented D-05 drop list, and FTS shadow tables / `unresolved_refs`
  are not read as data. `msToNs` epoch-ms→ns uses exact integer `*1_000_000`.
- **Edge-count reconciliation (D-09.1):** validate compares against
  `CountDistinctEdges()` (DISTINCT source,kind,target), which correctly matches the Pebble
  key's line/col-collapsing behavior. The DDL's `UNIQUE INDEX idx_edges_identity ... (source,
  target, kind, IFNULL(line,-1), IFNULL(col,-1))` confirms multi-callsite collapse is real,
  and the check accounts for it. `file:`-prefixed endpoints are correctly exempted from the
  dangling check; `--drop-dangling` is opt-in with fail-loud default.
- **Graphstore additive record:** `m/migration` is a clean length-prefixed sibling of
  `m/schema` — no key collision; `getRaw` copies bytes out of Pebble's reused buffer.

No Critical (crash / security / core-data-loss) defect was proven. The findings below are
real correctness/robustness defects concentrated in the `--drop-dangling` path, the atomic
swap, resource lifecycle, and the overwrite-refusal check.

## Warnings

### WR-01: `File.edge_count` is left stale/over-counted after `--drop-dangling`

**File:** `internal/migrate/migrate.go:176-180`, `internal/migrate/validate.go:330-368`
**Issue:** `recomputeFileEdgeCounts` (which derives each `File.EdgeCount` by counting that
file's `x/` file-index edge entries) runs at line 176 — *before* `validate` at line 180.
When `--drop-dangling` is set, `scanDangling`→`dropDanglingEdges` then deletes both the `e/`
record and the `x/` file-index entry for each dropped edge whose **source resolved but target
dangled** (`d.MissingSource == false`, so `DeleteFileIndexEdge(ownerPath, …)` runs). Those
`x/` entries were already counted into `File.EdgeCount`, and nothing recomputes the count
afterward.

**Concrete failure scenario:** TS index has file `a.go` owning edge `(sym:a, calls, sym:missing)`
where `sym:missing` is in no `nodes` row. Migrate with `--drop-dangling`. `recompute` counts
that `x/` entry → `a.go.edge_count = 1`. `validate` drops the edge and deletes the `x/` entry
→ `IterateFileIndex(a.go)` now yields 0 edges, but the persisted `File.edge_count` remains 1.
The migration exits 0 with silently-wrong per-file edge counts (inconsistent with the actual
`x/` index that downstream sync/query relies on). Core Meta/node/edge totals are correct;
only the derived per-file field is wrong.

**Fix:** Re-run `recomputeFileEdgeCounts(store)` *after* `validate` returns nil when
`report.Dropped > 0`, or move the recompute to after validation unconditionally:
```go
report, err := validate(src, store, opts)
if err != nil {
    return Result{Report: report}, err
}
if report.Dropped > 0 {
    if err := recomputeFileEdgeCounts(store); err != nil {
        return Result{}, err
    }
}
```

### WR-02: Final `batchWriter` Pebble batch is never `Close()`d (leak + contract violation)

**File:** `internal/migrate/migrate.go:448-462` (`commitData`), `152-174` (write loop)
**Issue:** `commitData` commits `bw.w`, then *eagerly opens a fresh empty writer*
(`bw.w = w; bw.n = 0`). After the last table's final `commitData`, that fresh writer holds an
open `*pebble.Batch` with `n == 0`. Nothing ever commits or closes it — `Run` proceeds to
`recomputeFileEdgeCounts`, opens a separate `mw` for meta, and eventually `store.Close()`.
The `Writer` interface doc (store.go:244-248) explicitly states an abandoned Writer MUST be
`Close()`d to return its batch to Pebble's `sync.Pool`. Here `bw.w` is always abandoned
un-closed at the end of every successful run (and on the error return paths after
`newBatchWriter`).

**Concrete failure scenario:** every migration leaks one Pebble batch (its backing memory is
not returned to the pool and is only reclaimed by GC). Low impact for a short-lived CLI, but
it is a real lifecycle defect and a documented-contract violation; in a future long-lived /
server-hosted migration path it becomes a per-run leak.

**Fix:** Give `batchWriter` a `Close()` that closes `bw.w`, and `defer bw.Close()` after
`newBatchWriter`. Or make `commitData` lazy (only open a new batch on the next `Put`, not
eagerly after commit), so a trailing no-op commit leaves nothing open.

### WR-03: `checkTargetOverwrite` writes a Pebble store into the target dir while *refusing*

**File:** `internal/migrate/migrate.go:288-315`
**Issue:** When `force == false` and the target is a non-empty directory, the "is this a prior
healthy migration?" probe calls `graphstore.Open(filepath.Join(target, "store"))`.
`graphstore.Open` → `pebble.Open(dir, &pebble.Options{})` **creates** the directory and writes
`MANIFEST`/`OPTIONS`/`CURRENT`/etc. if absent. For a target that is *not* a prior migration
(e.g. an in-place `.codegraph/` still holding the TS source `*.db`), this litters a fresh
`store/` subdirectory into the target — and then the function returns the "refusing" error
anyway. This mutates the target (and, for the in-place default where `from == to`, the source
`.codegraph/` directory) during what is supposed to be a read-only refusal check, brushing
against the D-08 non-destructive guarantee.

**Reachability:** guarded from the `codegraph migrate` CLI (the CLI flips `force = true` after
its own confirm before calling `Run`), but `migrate.Run` is documented as the "authoritative
last line of defense" public API and is reachable directly with `Options{Force: false}`.

**Fix:** Probe read-only. Open with `pebble.Options{ReadOnly: true}` (returns an error instead
of creating), or `os.Stat(filepath.Join(target,"store"))` first and only attempt the
health-read when the store already exists.

### WR-04: A crash inside the swap window leaves no target and no automatic recovery

**File:** `internal/migrate/swap.go:78-95`, `internal/migrate/migrate.go:265-274`
**Issue:** `atomicSwapDir` does rename-aside (`targetDir` → `targetDir.old`) then rename-into-
place (`tmpDir` → `targetDir`). Its restore logic only fires when the *second `os.Rename`
returns an error*; it does not (cannot) cover a process crash/kill in the window **between**
the two renames. If the process dies there, `.codegraph/` does not exist, the original is at
`.codegraph.old`, and the validated new store is at `.codegraph.migrate-partial`. On the next
`codegraph migrate` run, `resolveSourceDB(from)` does `os.Stat(".codegraph")` → `IsNotExist` →
hard error `migrate: resolve source …`. The tool cannot detect "target missing + `.old` +
complete partial present" and finish the swap, so a *fully validated* migration is left
requiring manual `mv` to recover, breaking the resumability guarantee (D-06/D-07) for this
window. Data is not lost, but the tool cannot self-heal.

**Fix:** On startup, before resolving the source, detect an orphaned swap state (target absent
or empty, `<target>.old` present, and/or a partial store carrying a `StatusComplete` cursor)
and complete/roll back the swap. At minimum, if the deterministic partial store exists with a
`complete` cursor, prefer finishing it over erroring on the missing source.

### WR-05: Source DB path is concatenated into a `file:` URI DSN without escaping

**File:** `internal/migrate/reader.go:78-93`
**Issue:** `dsn := "file:" + abs + "?mode=ro&_pragma=query_only(1)&_txlock=deferred"`. `abs`
is a raw filesystem path spliced directly into a URI. A path containing URI-significant
characters is mis-parsed: a space is not percent-encoded, and — more dangerously — a `?` in
the path (legal on POSIX) terminates the path and turns the remainder into query parameters,
while a `#` truncates it. This can break `sql.Open`/`Ping` for repos under paths like
`/Users/me/My Repo/…` or, in the pathological `?` case, cause the trailing path text to be
interpreted as connection/pragma parameters.

**Concrete failure scenario:** a repo checked out at `…/proj?x/.codegraph/index.db` yields DSN
`file:…/proj?x/.codegraph/index.db?mode=ro&…`; SQLite parses `x/.codegraph/index.db` as a
query key, opens the wrong (or an empty) database, and migration proceeds against garbage or
fails confusingly rather than pointing at the path.

**Fix:** Build the DSN with proper URI escaping, e.g. construct a `net/url.URL{Scheme:"file",
Path: abs}` and set `RawQuery` via `url.Values`, or percent-encode `abs` before concatenation.
Also normalize Windows `C:\…` paths to the `file:/C:/…` form.

## Info

### IN-01: `msToNs` float path loses precision above 2^53 ns

**File:** `internal/migrate/translate.go:143`, `160-161`
**Issue:** `int64(t * 1e6)` for a `float64` millisecond value ≥ ~9.0e6 ms produces a
nanosecond magnitude > 2^53, so the low ~8 bits of the nanosecond value are rounded. The
comment acknowledges "does not panic or truncate-crash," but the mtime is silently quantized
to ~256 ns. Harmless in practice (mtimes need second-ish resolution). No action required
beyond noting the accepted lossiness.

### IN-02: `checkWritableDir` ignores `f.Close()` and `os.Remove` errors

**File:** `internal/migrate/swap.go:21-22`
**Issue:** After the create-probe, `f.Close()` and `os.Remove(name)` return values are
dropped. If `Remove` fails, a `.codegraph-migrate-writable-check-*` temp file leaks into the
parent dir while the check still reports "writable." Low impact.
**Fix:** Check and, at minimum, wrap/log the `Remove` error; the leftover temp file otherwise
accumulates on repeated runs.

### IN-03: `normalizeFilePath` drive-letter heuristic can strip a legitimate `x:` prefix

**File:** `internal/migrate/translate.go:175-177`
**Issue:** `if len(p) >= 2 && p[1] == ':' { p = p[2:] }` treats any 2nd-char colon as a Windows
drive letter. A POSIX-legal relative path whose first segment contains a colon at index 1
(e.g. `a:b/foo`) would be corrupted to `b/foo`. The captured corpus is repo-relative
forward-slash so this is defensive-only, but the heuristic is broader than "single ASCII
letter followed by `:`".
**Fix:** Gate on `unicode.IsLetter(rune(p[0])) && p[1] == ':'` (single-letter drive).

### IN-04: Resume does not verify the cursor's `SourceSchemaVersion` matches the current source

**File:** `internal/migrate/migrate.go:126-140`, `internal/migrate/progress.go:20-26`
**Issue:** The deterministic partial store is keyed only by target parent directory. If a run
is interrupted and then re-invoked with a *different* `--from` into the same `--to`, `Run`
resumes the foreign partial store without comparing the persisted
`Progress.SourceSchemaVersion` (or a source fingerprint) against the freshly-read
`srcVersion`. In most cases `validate`'s count reconciliation is a strong backstop and fails
loud, but the mixed-source partial is only caught incidentally.
**Fix:** In the resume branch, reject (or discard and restart) a partial whose
`SourceSchemaVersion` differs from the current source's; consider recording a source
path/hash in `Progress` for a stronger match.

### IN-05: Iterator `Close()` errors dropped via `_ =` in `recomputeFileEdgeCounts`

**File:** `internal/migrate/migrate.go:597`, `602`, `605`
**Issue:** `_ = it.Close()` on the files iterator discards close errors, inconsistent with the
package's otherwise fail-loud posture. Pebble iterator `Close()` surfaces accumulated iterator
errors; dropping them could mask a late read error. Very low likelihood in practice.
**Fix:** Capture and return the `Close()` error on the success path (join with any pending
error), matching the `rows.Err()` discipline used elsewhere.

---

_Reviewed: 2026-07-13T01:37:23Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
