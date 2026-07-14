---
phase: 07-migration-tool
plan: 04
subsystem: migration
tags: [resumability, atomic-swap, progress-cursor, graphstore, crash-safety]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 02)
    provides: graphstore.Writer.PutMigration/Reader.GetMigration additive interface pair backing the m/migration meta key
  - phase: 06-agent-integrations-cli-lifecycle
    provides: internal/upgrade/swap.go — the single-file atomic-swap analog (checkWritable/atomicSwap/swapWindows/WR-04 restore-on-failure discipline) directly mirrored here for a directory target
provides:
  - "internal/migrate.Progress / saveProgress / loadProgress — the durable resumable migration cursor (D-06)"
  - "internal/migrate.siblingTempDir / atomicSwapDir / checkWritableDir — crash-safe atomic directory swap (D-07)"
affects: [07-05, 07-06, 07-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Resumable cursor: JSON-marshal a small struct, PutMigration (caller commits, no internal Commit call) so the cursor advances durably in its own small batch alongside data batches"
    - "loadProgress ErrNotFound-mapping mirrors 07-02's getRaw discipline: graphstore.ErrNotFound -> (zero, false, nil); any other error (including JSON corruption) wrapped and returned, never silently treated as absent"
    - "Directory atomic swap extends internal/upgrade/swap.go's single-file temp+rename to a 3-step rename-aside/rename-in/remove-old dance (os.Rename onto a non-empty existing directory fails on most platforms, unlike a single file)"
    - "WR-04 restore-on-failure: a rename-in failure after rename-aside triggers an attempted restore from the .old path; if restore also fails, both errors are reported and the .old path is named as the manual recovery location"

key-files:
  created:
    - internal/migrate/progress.go
    - internal/migrate/progress_test.go
    - internal/migrate/swap.go
    - internal/migrate/swap_test.go
  modified: []

key-decisions:
  - "saveProgress does not call w.Commit() — the caller (the not-yet-built orchestrator) commits the cursor in its own small batch after each data batch, per D-06's durability-alongside-data requirement"
  - "atomicSwapDir's step-3 (.old cleanup) failure is returned as a genuine (non-nil) error rather than logged-and-discarded, since the swap itself has already succeeded by that point — 'never swallowed' is satisfied by returning it, not by a separate logging side channel; the caller decides how non-fatal to treat it"
  - "Windows weaker-atomicity caveat documented as a doc comment only (per the plan's explicit instruction), no runtime.GOOS branch — the same temp+rename structure already bounds the torn-state window on all platforms"

requirements-completed: [MIGR-02]

coverage:
  - id: D1
    description: "A migration progress cursor (source+target schema versions, last completed/partial table, last rowid, status) is persisted durably through graphstore and read back to resume"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/progress_test.go#TestSaveProgressLoadProgress_RoundTrip"
        status: pass
      - kind: unit
        ref: "internal/migrate/progress_test.go#TestSaveProgressLoadProgress_StatusComplete"
        status: pass
    human_judgment: false
  - id: D2
    description: "On a fresh store with no cursor, loadProgress reports absent cleanly (not an error, not a crash)"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/progress_test.go#TestLoadProgress_AbsentReportsCleanly"
        status: pass
    human_judgment: false
  - id: D3
    description: "A corrupt/garbled cursor fails loud (wrapped error) rather than being silently treated as 'start clean'"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/progress_test.go#TestLoadProgress_CorruptFailsLoud"
        status: pass
    human_judgment: false
  - id: D4
    description: "A completed new-format store directory replaces the target atomically: fresh target, and existing-non-empty target via rename-aside/rename-in/remove-old"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/swap_test.go#TestAtomicSwapDir_FreshTarget"
        status: pass
      - kind: unit
        ref: "internal/migrate/swap_test.go#TestAtomicSwapDir_ExistingNonEmptyTarget"
        status: pass
    human_judgment: false
  - id: D5
    description: "A step-2 (rename-in) failure after rename-aside restores the original to target and reports both errors on a double failure — the target is never left pointing at nothing"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/swap_test.go#TestAtomicSwapDir_RestoreOnFailure"
        status: pass
    human_judgment: false
  - id: D6
    description: "The temp store is a sibling of the final target (same parent) so the swap rename is always same-filesystem, never EXDEV"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/swap_test.go#TestSiblingTempDir_DistinctSameParent"
        status: pass
      - kind: unit
        ref: "internal/migrate/swap_test.go#TestAtomicSwapDir_SameFilesystem"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-12
status: complete
---

# Phase 7 Plan 4: Progress Cursor + Atomic Directory Swap Summary

**`internal/migrate/progress.go` (JSON-encoded resumable migration cursor persisted via 07-02's `PutMigration`/`GetMigration`, fail-loud on corruption) and `swap.go` (three-step rename-aside/rename-in/remove-old atomic directory swap mirroring `internal/upgrade/swap.go`'s WR-04 restore-on-failure discipline, extended from a single file to a directory target) — the two highest silent-failure-risk mechanics of the migration tool (D-06 resumability, D-07 crash-safety).**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 4 (all created, 0 modified)

## Accomplishments
- `Progress{SourceSchemaVersion, TargetSchemaVersion, LastTable, LastRowID, Status}` with `StatusInProgress`/`StatusComplete` constants round-trips through the real `graphstore.Writer.PutMigration`/`Reader.GetMigration` boundary
- `loadProgress` maps `graphstore.ErrNotFound` to a clean `absent=false, nil-error` signal (never an error), and fails loud (wrapped error) on non-JSON cursor bytes so a garbled cursor can never be silently treated as "start clean"
- `siblingTempDir` places the migration's temp store in `filepath.Dir(targetDir)` — never `os.TempDir()` — guaranteeing the later swap rename is same-filesystem and cannot hit `EXDEV`
- `atomicSwapDir` performs the three-step directory swap (rename-aside → rename-in → remove-old) for both a fresh target and an existing non-empty target, with a WR-04 restore-on-failure path proven by a real injected rename-in failure (target restored, `.old` gone, both errors reported if the restore itself fails)
- `checkWritableDir` provides the fail-fast write-access precondition probe (mirrors `internal/upgrade/swap.go`'s `checkWritable` "fail fast before doing real work" rationale)

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: failing test for progress cursor round-trip/absent/corrupt** — `e5ae6e4` (test)
2. **Task 1 GREEN: progress.go implementation** — `c2a5dfa` (feat)
3. **Task 2 RED: failing test for atomic directory swap** — `7e99abc` (test)
4. **Task 2 GREEN: swap.go implementation** — `eb391fc` (feat)

_No REFACTOR commits — both GREEN implementations were minimal on first pass; no cleanup needed._

## Files Created/Modified
- `internal/migrate/progress.go` — `Progress` struct, `StatusInProgress`/`StatusComplete`, `saveProgress`, `loadProgress`
- `internal/migrate/progress_test.go` — round-trip, absent, StatusComplete distinguishability, corrupt-fails-loud tests against a real `graphstore.Open(t.TempDir())` store
- `internal/migrate/swap.go` — `siblingTempDir`, `checkWritableDir`, `atomicSwapDir`
- `internal/migrate/swap_test.go` — sibling-dir distinctness, fresh-target, existing-non-empty-target, restore-on-failure, same-filesystem, checkWritableDir tests

## Decisions Made
- `saveProgress` does not call `Commit()` internally — the future orchestrator (07-05/07-06) commits the cursor in its own small batch after each data batch, per D-06's "cursor advances durably with the data" requirement. This was an explicit plan instruction, not a deviation.
- `atomicSwapDir`'s step-3 `.old` cleanup failure is returned as a real, non-nil error (not logged-and-discarded) — the plan's "never swallowed" prohibition is satisfied by returning it; the caller (a later plan's orchestrator) decides how non-fatally to treat a successful swap with a leftover `.old` directory.
- The Windows weaker-atomicity caveat is documented purely as a doc comment (no `runtime.GOOS` branch in code), per the plan's explicit instruction — the temp+rename structure already bounds the torn-state window identically on every platform.

## Deviations from Plan

None — plan executed exactly as written. Both tasks followed the plan's `<action>` guidance directly; the restore-on-failure test was implemented by passing a non-existent `tmpDir` to force the rename-in step to fail after the rename-aside step succeeded, which the plan anticipated ("assert via an injected/simulated rename failure").

## Issues Encountered
None. `go build ./...` and `go test ./internal/migrate/... ./internal/graphstore/... ./internal/cli/...` are all green; no daemon-flake package was touched or exercised by this plan.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `saveProgress`/`loadProgress` and `siblingTempDir`/`atomicSwapDir`/`checkWritableDir` are ready for the migration orchestrator (a later 07-xx plan) to drive: write into a `siblingTempDir`, commit progress after each batch via `saveProgress`, run the D-09 validation pass, then call `atomicSwapDir` to swap the completed temp store into place.
- No blockers for subsequent 07-xx plans (orchestration, validation, CLI wiring).

---
*Phase: 07-migration-tool*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created files verified present on disk:
- internal/migrate/progress.go (FOUND)
- internal/migrate/progress_test.go (FOUND)
- internal/migrate/swap.go (FOUND)
- internal/migrate/swap_test.go (FOUND)

All four task commit hashes (e5ae6e4, c2a5dfa, 7e99abc, eb391fc) verified present in git log.
