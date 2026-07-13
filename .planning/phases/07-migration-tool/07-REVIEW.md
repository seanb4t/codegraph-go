---
phase: 07-migration-tool
reviewed: 2026-07-13T00:00:00Z
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
  critical: 1
  warning: 1
  info: 4
  total: 6
status: issues_found
---

# Phase 7: Code Review Report (Re-Review)

**Reviewed:** 2026-07-13
**Depth:** deep (cross-file call-chain + import-graph analysis)
**Files Reviewed:** 13
**Status:** issues_found

## Summary

This is the second deep pass, after WR-01..WR-05 from the prior review were fixed
(07-REVIEW-FIX.md). I re-verified all five fixes against the current code and
scrutinized the fixes themselves for regressions. **All five are genuinely and
completely fixed** — details in the re-verification section below.

The fresh full-surface pass, however, surfaced a defect the prior review missed
that is more serious than any of WR-01..WR-05: the read-only source handle is held
open across the atomic directory swap. On the primary (default, in-place) code path
this is invisible on POSIX — an open fd survives a directory rename/unlink — but on
Windows it deterministically breaks the swap, because Windows refuses to rename a
directory that still contains an open file handle. The default `codegraph migrate`
command therefore cannot complete on Windows, and re-running does not self-heal.

The remaining findings are lower-severity robustness/reporting issues in the
resume/recover paths.

## Previously-fixed findings re-verified

- **WR-01 (File.edge_count after `--drop-dangling`): FIXED.** `migrate.go:217-221`
  re-runs `recomputeFileEdgeCounts(store)` after `validate` returns nil when
  `report.Dropped > 0`. `dropDanglingEdges` (validate.go:330-368) deletes the `x/`
  file-index entry only for edges whose source resolved (`!d.MissingSource`), which
  is exactly the set whose owning `File.EdgeCount` was over-counted; the post-drop
  recompute re-derives each count from the live `x/` index taken from a fresh
  snapshot. `Meta.EdgeCount` is computed afterward (line 227) from the post-drop
  `countEdges`, so Meta stays consistent with the store. No regression.

- **WR-02 (trailing `batchWriter` batch leaked): FIXED.** `batchWriter.Close()`
  (migrate.go:565-572) nils out and closes the currently-open (always fresh,
  per `commitData`'s eager reopen) Writer, and `defer bw.Close()` (line 173)
  releases it on every exit path. Idempotent (guards `bw.w == nil`) and safe after
  Commit (`pebbleWriter.Close` swallows `pebble.ErrClosed`, batch.go:162-167).

- **WR-03 (overwrite-refusal probe created a `store/` dir): FIXED.**
  `checkTargetOverwrite` now `os.Stat`s `target/store` first (migrate.go:339-340)
  and only opens the Pebble store when it already exists, so a non-migration target
  (e.g. a TS `.codegraph/` with only `*.db`) is never mutated during refusal. (One
  residual noted as IN-03 below.)

- **WR-04 (interrupted swap self-heal): FIXED, conservative.**
  `recoverInterruptedSwap` (migrate.go:396-443) runs before `resolveSourceDB` and
  acts only when the target is absent/empty AND the deterministic partial carries a
  `StatusComplete` cursor — an `in_progress`/unvalidated partial, a populated
  target, or a missing partial are all declined, so an unvalidated store is never
  swapped in. No loop or deadlock (linear, single pass). It finishes via
  `finishFromComplete` and removes the stale `<target>.old`. Recovery cannot delete
  the "wrong" dir: it only ever renames the already-validated partial into an
  absent/empty target. (Two edge behaviors noted as WR-01/IN-01/IN-02 below.)

- **WR-05 (source path spliced into `file:` DSN unescaped): FIXED.** `sourceDSN`
  (reader.go:105-114) carries the path through `net/url.URL{Scheme:"file", Path:…}`,
  percent-encoding spaces / `?` / `#` / `%`, then appends the fixed
  `?mode=ro&_pragma=query_only(1)&_txlock=deferred` verbatim (pragma preserved
  byte-for-byte). Windows `C:\…` normalizes via `ToSlash` + leading-slash to
  `file:///C:/…`. Correct.

## Critical Issues

### CR-01: In-place migration holds the source DB open across the directory swap — deterministically breaks the default command on Windows

**File:** `internal/migrate/migrate.go:109-113` (source opened, `defer src.Close()`),
`internal/migrate/migrate.go:268-273` and `461-484` (`finishFromComplete`),
`internal/migrate/swap.go:78-100`

**Issue:** `Run` opens the source read-only at line 109 and defers its `Close()` to
function return (line 113). The atomic swap runs *before* that deferred close — at
line 271 on the happy path, and inside `finishFromComplete` (line 482) on the
resume-complete path. For the **default in-place migration** (`--from` and `--to`
both default to `.codegraph/` — see `cli/migrate.go:48-63`), the source `*.db`
resolved by `FindDBFile` lives *inside* the target directory that `atomicSwapDir`
renames.

`atomicSwapDir` step 1 (`swap.go:80`) does `os.Rename(targetDir, asidePath)` — it
renames `.codegraph/` (which still contains the open `index.db`) aside to
`.codegraph.old`.

- On **POSIX** this works: the open fd follows the inode through the rename, and
  step 3's `os.RemoveAll(.codegraph.old)` unlinks a still-open file (valid until
  `src.Close()`). Invisible on the dev platform.
- On **Windows**, a directory containing an open file handle cannot be renamed —
  `MoveFile` returns a sharing violation. Step 1 fails and `atomicSwapDir` returns
  `"migrate: rename existing target aside"` (swap.go:81), so a fully written,
  validated, `Meta.healthy=true` store is **never swapped into place**.

**Concrete failure scenario (Windows):** User runs the documented default
`codegraph migrate` in a repo with a TS `.codegraph/`. Migration reads, validates,
stamps the partial complete — then the swap fails at step 1 because `index.db` is
still open. The command exits non-zero. On re-run, `recoverInterruptedSwap` is a
no-op (the target `.codegraph/` still exists and is populated with the TS source),
so `Run` proceeds, hits the `StatusComplete` branch (line 148), re-opens the source
(line 109, handle open again), and `finishFromComplete`'s swap fails at step 1
identically. **The default command can never complete on Windows, and every retry
fails the same way** — no data loss (source untouched, partial preserved), but a
hard functional break on a supported platform for the primary use case.

**Fix:** Close the source before any swap. `src` is unused after `validate`
(reconcileCounts is its last consumer), so close it explicitly and drop the defer,
or close it right after `validate`:
```go
report, err := validate(src, store, opts)
if err != nil {
    return Result{Report: report}, err
}
if err := src.Close(); err != nil {
    return Result{}, err
}
// ... recompute / counts / meta ...
// store.Close(); atomicSwapDir(...)
```
Apply the same ordering to the `StatusComplete` resume branch (line 148-154): close
`src` before calling `finishFromComplete`. (This also tightens POSIX hygiene — it
stops step 3 from unlinking the live source out from under the open handle.)

## Warnings

### WR-01: `finishFromComplete` returns a zero-valued `Report`, so a resumed/recovered migration prints "migrated: N/0" source counts

**File:** `internal/migrate/migrate.go:461-493`, `internal/cli/migrate.go:125-133`

**Issue:** The happy path builds `Result.Report` from `validate` and the CLI prints
`files=%d/%d nodes=%d/%d edges=%d/%d (migrated/source)` using
`result.Report.Files.Source` etc. But `finishFromComplete` (used by both the
`StatusComplete` resume branch at line 153 and the WR-04 recovery at line 427)
returns `Result{Nodes, Edges, Files, Resumed, HealthMessage}` with `Report` left at
its zero value — it never re-opens the source to populate `Report.*.Source` (and in
the in-place recovery case it *cannot*, the source is gone).

**Concrete failure scenario:** Any migration that is interrupted after the partial
is stamped complete and then re-run prints, e.g.,
`migrated: files=5/0 nodes=42/0 edges=88/0 (migrated/source)` — the reconciliation
line a user relies on to trust the migration shows every source count as 0, looking
like a failed/empty migration even though it succeeded and was fully validated.

**Fix:** Persist the reconciled counts into the partial store's Meta at
completion (Meta already carries `NodeCount`/`EdgeCount`) or into the `Progress`
record, and have `finishFromComplete` read them back into `Report.*.Source`; or have
the CLI suppress the `/source` denominator (and print a "resumed — source counts
unavailable" note) when `result.Resumed` is true and the Report is zero.

## Info

### IN-01: `recoverInterruptedSwap` leaks the opened partial store on `finishFromComplete`'s error paths

**File:** `internal/migrate/migrate.go:407-430`

**Issue:** Unlike the main `Run` (which sets `defer store.Close()` at line 140),
`recoverInterruptedSwap` opens the partial store (line 407) with no deferred close;
it closes it explicitly on the decline paths (lines 423, 417) and relies on
`finishFromComplete` to close it on success. But if `finishFromComplete` errors
*before* its own `store.Close()` (any of the `countNodes/countEdges/countFiles/
getStoreMeta` calls fail, migrate.go:462-477), it returns without closing, and
`recoverInterruptedSwap` returns `true, res, ferr` without closing either — leaking
the Pebble handle (and its `LOCK`) for the remainder of the process.

**Fix:** `defer store.Close()` immediately after `graphstore.Open` in
`recoverInterruptedSwap` (Pebble `Close` is idempotent, so the later explicit close
in `finishFromComplete` remains a safe no-op).

### IN-02: `targetPopulated` treats an existing-but-unreadable target as empty, letting recovery bypass the D-08 overwrite guard

**File:** `internal/migrate/migrate.go:448-454`

**Issue:** `targetPopulated` returns `false` whenever `os.ReadDir(target)` errors —
including permission-denied or "target is a regular file", not just not-exist. In
those cases `recoverInterruptedSwap` proceeds as if the target were absent and, if a
`StatusComplete` partial happens to exist, swaps it into place via
`finishFromComplete` — which never calls `checkTargetOverwrite` and never consults
`Force`. Recovery thus bypasses the D-08 non-destructive guard for a populated but
unreadable target.

**Fix:** Distinguish `os.IsNotExist` from other `ReadDir` errors; on a
non-not-exist error, treat the target as populated (decline recovery) so the normal
`Run` path applies the overwrite guard and surfaces the real error.

### IN-03: WR-03 residual — the health probe still opens an existing store read-write during the "no changes made" refusal check

**File:** `internal/migrate/migrate.go:341`

**Issue:** WR-03's fix stops *creating* a `store/` dir, but when `target/store`
already exists the probe calls `graphstore.Open` (→ `pebble.Open(dir,
&pebble.Options{})`), which opens read-write: Pebble writes a fresh `OPTIONS-NNN`,
rewrites `MANIFEST`/`CURRENT`, and replays/creates a WAL on open. So probing a prior
migration's store to read `Meta.Healthy` mutates it, contradicting the error
message's "(no changes made)". Harmless in practice (that store is a prior migration
we may overwrite anyway), but it is a residual write during a check documented as
non-destructive.

**Fix:** Open the probe read-only: `pebble.Open(dir, &pebble.Options{ReadOnly:
true})` behind a small graphstore helper (a read-only open also returns an error
rather than mutating when the store is absent, subsuming the `os.Stat` guard).

### IN-04: Carry-forward — original IN-01..IN-05 remain open

**File:** various (see 07-REVIEW-FIX.md §Skipped)

The five Info findings from the first review — `msToNs` float precision
(translate.go:143), `checkWritableDir` dropped `Remove` error (swap.go:22),
`normalizeFilePath` drive-letter heuristic (translate.go:175), resume
source-fingerprint check (migrate.go:126-140/progress.go), and iterator `Close()`
errors dropped in `recomputeFileEdgeCounts` (migrate.go:724-733) — were explicitly
out of the fix scope and are still present. None is elevated by this pass; noting
them so the record stays complete.

---

_Reviewed: 2026-07-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
