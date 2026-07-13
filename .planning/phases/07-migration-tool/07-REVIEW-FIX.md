---
phase: 07-migration-tool
fixed_at: 2026-07-12T22:00:00Z
review_path: .planning/phases/07-migration-tool/07-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 7: Code Review Fix Report

**Fixed at:** 2026-07-12T22:00:00Z
**Source review:** .planning/phases/07-migration-tool/07-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (all Warning — the 5 Info findings IN-01..IN-05 excluded per instructions, left open)
- Fixed: 5
- Skipped: 0

Each fix ships with a regression test that fails on the pre-fix code and passes after (verified by temporary revert for WR-01, WR-02, WR-04; and by DSN inspection for WR-05 / os.Stat probe for WR-03).

## Fixed Issues

### WR-01: `File.edge_count` left stale/over-counted after `--drop-dangling`

**Files modified:** `internal/migrate/migrate.go`, `internal/migrate/migrate_test.go`
**Commit:** `83b4f88`
**Applied fix:** `recomputeFileEdgeCounts` still runs before `validate` (the happy path needs it), but Run now re-runs it AFTER `validate` returns nil when `report.Dropped > 0`, so any owning file whose `x/` entry was deleted during the drop has its `File.edge_count` re-derived from the post-drop `x/` index. Regression test `TestRun_DropDangling_FileEdgeCountReconciled` builds a minimal purpose-built TS source where a File record's own symbol (`func:a` in `pkg/a.go`) owns the dangling edge — the shared `migratetest` fixture cannot express this because its seeded node file_paths correspond to no files-table row, so no File record ever carries an `x/` edge entry. It proves `pkg/a.go`'s persisted `edge_count` reconciles to 0 (the live `x/` count) after the drop; without the fix it reads 1.

### WR-02: Final `batchWriter` Pebble batch never `Close()`d (leak + contract violation)

**Files modified:** `internal/migrate/migrate.go`, `internal/migrate/batchwriter_test.go` (new)
**Commit:** `9e12afc`
**Applied fix:** Added an idempotent `batchWriter.Close()` that returns the currently-open (always fresh/uncommitted, per `commitData`'s eager-reopen) Writer's batch to Pebble's pool, and `defer bw.Close()` in Run so the trailing batch is released on every exit path (Close-after-Commit is a documented safe no-op). Regression test `TestBatchWriter_ClosesTrailingWriter` wraps the store in a `countingStore` that tracks Writer opens vs terminals (Commit/Close), drives the exact commit rhythm Run uses, asserts the eagerly-opened trailing Writer is left outstanding after `commitData`, and that `Close()` returns it (and is idempotent). Verified to fail when `Close()` is stubbed to a no-op.

### WR-03: `checkTargetOverwrite` wrote a Pebble store into the target while *refusing*

**Files modified:** `internal/migrate/migrate.go`, `internal/migrate/migrate_test.go`
**Commit:** `3f4fdba`
**Applied fix:** The overwrite-refusal probe now `os.Stat`s `target/store` first and only opens the pebble store for the health-read when it already exists — `graphstore.Open` → `pebble.Open` creates the store directory otherwise, mutating the target (and, for the in-place `from==to` default, the source `.codegraph/`) during a check that must be non-destructive (D-08). Regression test `TestRun_RefusesWithoutCreatingStore` seeds a non-empty, non-migration target, calls `Run` with `Options{Force:false}`, asserts it refuses, and asserts no `store/` directory was created.

### WR-04: A crash inside the swap window left no target and no automatic recovery

**Files modified:** `internal/migrate/migrate.go`, `internal/migrate/migrate_test.go`
**Commit:** `4e7bc57`
**Applied fix:** Added `recoverInterruptedSwap`, run at the top of Run before resolving the source. When the target is absent/empty AND the deterministic partial store carries a `StatusComplete` cursor (data fully written and D-09-validated before the crash), it finishes the swap via `finishFromComplete` and removes the stale `<target>.old`. It is conservative: a target that is populated, a missing partial, or an `in_progress` (unvalidated) partial are all left to the normal Run/resume path — an incomplete store is never swapped in. This restores the D-06/D-07 resumability guarantee for the in-place case where the source directory became `<target>.old`. Regression tests: `TestRun_RecoversInterruptedSwap` stages the exact interrupted-swap on-disk state (partial + `.old`, target absent) and proves in-place recovery to a healthy `.codegraph/` with matching counts and the `.old` removed; `TestRun_RecoveryLeavesInProgressPartialAlone` proves an `in_progress` partial is declined by recovery and completed by the normal resume path instead.

### WR-05: Source DB path concatenated into a `file:` URI DSN without escaping

**Files modified:** `internal/migrate/reader.go`, `internal/migrate/reader_test.go`
**Commit:** `6fae7ff`
**Applied fix:** Added `sourceDSN`, which carries the absolute path through `net/url.URL{Scheme:"file", Path:…}` so URI-significant characters (spaces, and a literal `?`/`#` legal on POSIX) are percent-encoded instead of mis-parsed as the query/fragment delimiter. The fixed `mode=ro&_pragma=query_only(1)&_txlock=deferred` query is appended verbatim after escaping the path, preserving the exact pragma behavior byte-for-byte; Windows drive paths normalize to `file:///C:/…`. Regression test `TestOpenSource_PathWithURISpecialChars` copies a fixture DB under a directory named `My Repo ?x` and asserts `OpenSource` + `CountRows` succeed against the right database (before the fix the raw `?` truncated the path and the open pointed at a nonexistent/empty DB).

## Skipped Issues

None — all 5 in-scope Warning findings (WR-01..WR-05) were fixed. The 5 Info findings (IN-01 msToNs float precision, IN-02 `checkWritableDir` dropped `Remove` error, IN-03 drive-letter heuristic, IN-04 resume source-fingerprint check, IN-05 iterator `Close()` errors dropped) were excluded from scope per the task instructions and remain open.

## Verification

- `go build ./...` — clean
- `go vet ./internal/migrate/` — clean
- `go test ./... -count=1 -p 1` (serial) — all packages pass, including `internal/migrate`
- `go test ./internal/daemon/ -count=1` (isolated) — passes; the one failure seen under the default parallel `go test ./...` was `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` (a debounced-flush-lock timing test), the known pre-existing flake under parallel load, unrelated to the migrate changes

---

_Fixed: 2026-07-12T22:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
