---
phase: 07-guards-that-cannot-fire
plan: 01
subsystem: testing
tags: [go, table-driven-testing, tdd, regression-gate, bench, mutation-log]

# Dependency graph
requires: []
provides:
  - "CheckRegression refuses a non-positive current.FilesPerSec or current.PeakRSSBytes with a named error, closing backlog 999.4"
  - "07-MUTATION-LOG.md created with header, pre-mutation cleanliness gate convention, and GRD-01 family (a) entry, ready for 07-02/03/04 to append"
affects: [07-guards-that-cannot-fire]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 2264
  tasks: 3
  commits: 3
  plan_head_before: 1a10c4e78450a825fe8c3845b394838c022f96ec

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN split across separate tracer/auto tasks within one plan, each with its own commit and its own <verify>"
    - "Mutation-log entry with no revert step when RED is the absence of a fix rather than a deliberate mutation of correct code — recorded as an explicit shape deviation from 03-MUTATION-LOG.md"

key-files:
  created:
    - .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md
  modified:
    - internal/bench/regression.go
    - internal/bench/regression_test.go

key-decisions:
  - "Followed D-10 exactly: current-metrics positivity checks inserted immediately after the two baseline positivity checks and before the delta math, mirroring the baseline lines' error-message shape verbatim"
  - "Used the literal ceiling=1 for the historical Phase 10 audit replay row per D-10/backlog 999.4, not the file's usual 1_000_000_000 ceiling constant"
  - "errHint set to the multi-word prefix 'invalid current: <Field>' rather than a bare 'current', since the pre-fix throughput message already contains 'current=0.00 files/s' and a bare hint would match the wrong error"
  - "No REFACTOR commit: the GREEN implementation already mirrors the existing baseline-check pattern exactly, so there was nothing to clean up"
  - "Family (a)'s mutation-log entry has no tracked-file mutation and no revert step, because the RED condition was the absence of the fix (the committed, unmodified production file) rather than a deliberately-broken copy of correct code — this is a documented shape deviation from 03-MUTATION-LOG.md, per the plan's flagged assumption 4"

requirements-completed: [GRD-01, GRD-06]

coverage:
  - id: D1
    description: "CheckRegression(baseline, current, ceiling=1) with current.PeakRSSBytes=0 and an otherwise-matching frame returns a non-nil error naming PeakRSSBytes"
    requirement: "GRD-01"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/degenerate_current_PeakRSSBytes_is_refused_rather_than_read_as_no_regression"
        status: pass
    human_judgment: false
  - id: D2
    description: "CheckRegression with current.FilesPerSec=0 returns an error naming FilesPerSec as an invalid current reading, not a misattributed throughput-regression message"
    requirement: "GRD-01"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/degenerate_current_FilesPerSec_is_refused_rather_than_reported_as_a_throughput_regression"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both degenerate-current rows were watched failing against the byte-identical committed regression.go before the fix, and that failure is pasted verbatim in 07-MUTATION-LOG.md"
    requirement: "GRD-06"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression (RED run, commit d767194a)"
        status: pass
    human_judgment: false
  - id: D4
    description: "07-MUTATION-LOG.md exists, is committed, and ends with a horizontal rule ready for three appended families"
    requirement: "GRD-06"
    verification:
      - kind: other
        ref: "git ls-files -- .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-09-09
status: complete
---

# Phase 7 Plan 1: CheckRegression degenerate-current guard Summary

**Closed the backlog 999.4 bypass in `CheckRegression` — a zero current peak-RSS or throughput reading now returns a named "invalid current" error instead of silently reading as no regression, proven RED against the unmodified build first and logged verbatim in a new mutation log.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-08T23:59:43Z
- **Completed:** 2026-09-09T00:07:00Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Two new `TestCheckRegression` table rows reproduce the historical Phase 10 audit frame (`ceiling=1`, `current.PeakRSSBytes=0`) and its companion (`current.FilesPerSec=0`), watched failing against the byte-identical, unmodified `internal/bench/regression.go` (nil error and a misattributed "throughput regressed 100.0%" error respectively).
- `CheckRegression` now returns `bench: invalid current: FilesPerSec must be positive, got %.4f` and `bench: invalid current: PeakRSSBytes must be positive, got %d` for a non-positive current reading, inserted immediately after the existing baseline positivity checks and before the delta math (D-10 ordering preserved: frame attribution still runs first).
- `07-MUTATION-LOG.md` created with its header, the pre-mutation cleanliness-gate convention, and a complete GRD-01 family (a) entry carrying the verbatim pre-fix failing transcript and the green re-run, ready for 07-02/03/04 to append.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — land the two degenerate-current cases and watch them fail** - `d767194a` (test)
2. **Task 2: GREEN — add the two current-metrics positivity checks and extend the function doc** - `71d62e28` (feat)
3. **Task 3: Create 07-MUTATION-LOG.md with its header and the GRD-01 family entry** - `0dd78e73` (docs)

**Plan metadata:** commit created by this SUMMARY's own atomic write+commit step.

_Note: Task 1 was `type="tracer" tdd="true"` — the plan's TDD RED phase; the tracer feedback gate was re-evaluated (auto mode active via `workflow.auto_advance=true`) by re-running the tracer's `<verify>` commands before Task 2 (GREEN/expansion) began. Both passed, so execution continued without a checkpoint._

## Files Created/Modified
- `internal/bench/regression_test.go` - Added two table rows reproducing backlog 999.4's degenerate-current bypass
- `internal/bench/regression.go` - Added the two current-metrics positivity checks and extended the doc comment
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` - New evidence artifact: header, cleanliness-gate convention, GRD-01 family (a) entry

## Decisions Made
- Ordering, ceiling literal, and errHint wording all followed D-10/D-11/the plan's `<behavior>` block exactly — no deviation from the specified shape was needed.
- No REFACTOR commit: the GREEN implementation is a direct mirror of the existing baseline-check pattern, so there was nothing to clean up post-GREEN.
- Family (a)'s mutation-log entry documents an intentional "no mutation, no revert" shape, since the RED condition was the absence of the fix rather than a deliberately-broken copy of correct code (flagged assumption 4 in the plan).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `07-MUTATION-LOG.md` exists, is committed, and ends with a horizontal rule — 07-02, 07-03, and 07-04 can append families (b), (c), (d), and the GRD-05 deletion record without editing existing content.
- `internal/bench/regression.go` is otherwise untouched apart from the D-10 insertion; `tools/bench/runner/main.go` (the only caller) needed no change.
- No blockers for 07-02 (archtest boundary, GRD-02).

---
*Phase: 07-guards-that-cannot-fire*
*Completed: 2026-09-09*

## Self-Check: PASSED

All created/modified files found on disk; all three task commits (d767194a, 71d62e28, 0dd78e73) found in git log.
