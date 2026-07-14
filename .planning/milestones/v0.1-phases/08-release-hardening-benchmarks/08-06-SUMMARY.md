---
phase: 08-release-hardening-benchmarks
plan: 06
subsystem: infra
tags: [benchmarking, regression-gate, tdd, ci-gate]

# Dependency graph
requires:
  - "internal/bench.Metrics (Plan 08-02) — the data holder this gate consumes"
provides:
  - "internal/bench.CheckRegression(baseline, current Metrics, ceilingBytes int64) error"
  - "internal/bench.DefaultThroughputTolerance / DefaultRSSTolerance tolerance constants"
affects: [08-07-head-to-head-runner, 08-08-ci-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure comparison function (no I/O, no baseline mutation) proven test-first, consumed later by an I/O wrapper (runner + CI gate) — mirrors internal/bench's existing rss.go/metrics.go discipline"
    - "Absolute ceiling checked independently of relative-delta tolerance bands, so a baseline that's already near budget can't mask further growth behind a large denominator"

key-files:
  created: [internal/bench/regression.go, internal/bench/regression_test.go]
  modified: []

key-decisions:
  - "Fixed a self-contradictory RED test fixture (see Deviations) with a per-case baseline override so the absolute-ceiling test genuinely isolates the ceiling check from the relative RSS-tolerance-band check"

patterns-established:
  - "Table-test cases carry an optional per-case baseline override (zero value falls back to a shared default baseline) so band-boundary tests and ceiling tests can each set up their own independent baseline/current pair"

requirements-completed: [PERF-02, INDX-06]

coverage:
  - id: D1
    description: "Throughput regression beyond DefaultThroughputTolerance (10%) fails; within-band passes"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/throughput_*"
        status: pass
    human_judgment: false
  - id: D2
    description: "Peak-RSS growth beyond DefaultRSSTolerance (15%) fails; within-band passes"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/peak_RSS_*"
        status: pass
    human_judgment: false
  - id: D3
    description: "Absolute peak-RSS ceiling (INDX-06 bounded memory) fails independently of the relative RSS delta"
    requirement: "INDX-06"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/above_absolute_ceiling_*, below_absolute_ceiling_passes"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero/negative baseline never panics — returns a clear error"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/zero_baseline_*"
        status: pass
    human_judgment: false
  - id: D5
    description: "CheckRegression never mutates baseline/current or any file; re-blessing is documented as a separate, explicit action never triggered by this function"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "internal/bench/regression.go doc comment + function has no write side effects (no os/io imports)"
        status: pass
    human_judgment: false

# Metrics
duration: 2min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 06: Committed-Baseline Regression Gate Summary

**`CheckRegression` — a pure, tested tolerance-band (throughput -10%/RSS +15%) plus absolute peak-RSS ceiling (INDX-06) gate over the Plan 08-02 `Metrics` type, proven RED→GREEN, with re-blessing kept an explicit non-automatic action.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-07-13T18:22:28Z
- **Completed:** 2026-07-13T18:24:10Z
- **Tasks:** 2 completed
- **Files modified:** 2 (`internal/bench/regression.go` created, `internal/bench/regression_test.go` created then corrected)

## Accomplishments
- `internal/bench.CheckRegression(baseline, current Metrics, ceilingBytes int64) error` implements the PERF-02 committed-baseline gate: fails on >10% throughput regression, fails on >15% peak-RSS growth, and independently fails when `current.PeakRSSBytes` exceeds an absolute ceiling regardless of the relative delta (INDX-06 bounded-memory requirement)
- `DefaultThroughputTolerance = 0.10` and `DefaultRSSTolerance = 0.15` exported as tune-able D-05 starting-point constants
- A degenerate baseline (`FilesPerSec <= 0` or `PeakRSSBytes <= 0`) returns a descriptive error instead of dividing by zero or panicking
- Every failure error names the offending metric plus the observed delta/value and the configured budget, satisfying the "fail loud, actionable" requirement
- Doc comment on `CheckRegression` states explicitly that the function never mutates the baseline — re-blessing is a separate, explicit `-rebless` action on the runner (Plan 08-07), reviewable as its own `baseline.json` diff
- TDD gate proven in git history: `test(08-06)` commit (`83ec810`) fails to build (`undefined: CheckRegression`, RED), `feat(08-06)` commit (`ef83c03`) passes both `go test` and `go vet` (GREEN)

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Write failing tolerance-band + absolute-ceiling tests** - `83ec810` (test)
2. **Task 2 (GREEN): Implement CheckRegression with tolerance constants + absolute ceiling** - `ef83c03` (feat)

**Plan metadata:** (this commit)

_Note: no separate REFACTOR commit — the test-fixture bug found while running GREEN (see Deviations) was folded into the Task 2 commit alongside the implementation, since both files needed to change together for the suite to prove the intended policy._

## Files Created/Modified
- `internal/bench/regression_test.go` - `TestCheckRegression` table test: clean-pass case, throughput within-band/over-band pair, RSS within-band/over-band pair, absolute-ceiling-fails-in-band-delta case, below-ceiling-passes case, zero-baseline-errors-not-panics case; `containsFold` helper for case-insensitive error-message substring assertions
- `internal/bench/regression.go` - `CheckRegression` + `DefaultThroughputTolerance` + `DefaultRSSTolerance`, no I/O, no panics

## Decisions Made
- Per-case `baseline` override field added to the table test (zero-value `Metrics{}` falls back to a shared default baseline) so the absolute-ceiling test cases could construct a baseline/current pair where the relative RSS delta is genuinely in-band while the absolute value still crosses the ceiling — this is what actually isolates and proves ceiling-independent-of-delta behavior (see Deviations for why the original fixture couldn't).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected a self-contradictory RED test fixture for the absolute-ceiling cases**
- **Found during:** Task 2 (GREEN implementation, first test run)
- **Issue:** The original RED test's "above absolute ceiling" and "below absolute ceiling" cases both used the shared default baseline (`PeakRSSBytes: 500_000_000`) with current values of `1_100_000_000` and `900_000_000` respectively. Both of those already represent relative RSS growth (+120% and +80%) far outside the 15% tolerance band, so `CheckRegression` correctly failed both on the *RSS-tolerance* check before ever reaching the ceiling check — the tests couldn't actually isolate "ceiling fires independently of relative delta" as the plan's behavior spec required, and the error-hint assertions (`"ceiling"`) failed against the RSS-tolerance error message instead.
- **Fix:** Added a per-case `baseline` override to the table test. The ceiling-specific cases now use a baseline already close to the ceiling (`950_000_000`) with a current value only +10.5% or +3.2% higher — genuinely inside the 15% RSS tolerance band — so the absolute-ceiling check is what actually decides pass/fail for those two cases, proving the two checks are independent as the plan's `<behavior>` block specifies.
- **Files modified:** `internal/bench/regression_test.go`
- **Verification:** `go test ./internal/bench/... -run TestCheckRegression -v` — all 8 subtests pass, including both ceiling cases now genuinely exercising the ceiling path (confirmed the RSS-tolerance check alone would have passed both, isolating the ceiling as the deciding check).
- **Committed in:** `ef83c03` (Task 2 commit, alongside `regression.go`)

---

**Total deviations:** 1 auto-fixed (1 test-fixture bug — a self-contradictory test data setup found while driving RED to GREEN, not a plan-authoring error in the behavior spec itself)
**Impact on plan:** No behavioral change to `CheckRegression`'s policy; the fix only corrected which test-fixture numbers actually exercise the intended independent-ceiling-check code path. No scope creep.

## Issues Encountered
None beyond the test-fixture bug documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- `internal/bench.CheckRegression` is ready for Plan 08-07 (head-to-head runner, which will call it after computing a candidate `Metrics` run) and Plan 08-08 (the CI gate that treats a non-nil error as a blocking failure)
- The re-bless path itself (an explicit `-rebless` flag rewriting `baseline.json`) is Plan 08-07's responsibility — this plan only guarantees `CheckRegression` itself never performs that mutation
- No blockers

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## TDD Gate Compliance
- RED commit found: `83ec810` (`test(08-06): failing tolerance-band + absolute-ceiling regression tests`)
- GREEN commit found: `ef83c03` (`feat(08-06): tolerance-band + absolute-RSS-ceiling regression gate`)
- No REFACTOR commit — none needed; gate sequence satisfied.

## Self-Check: PASSED
- FOUND: internal/bench/regression.go
- FOUND: internal/bench/regression_test.go
- FOUND: 83ec810 (Task 1 commit)
- FOUND: ef83c03 (Task 2 commit)
