# 06-MUTATION-LOG — FIXT-07 two-family mutation rehearsal (BENCH-02)

**Phase:** 06-benchmark-de-coupling-memory-sweep (Plan 06-03, Task 3)
**Date:** 2026-08-16
**Rehearsal target:** `internal/bench/regression.go`'s `CheckRegression` — the committed-baseline
regression gate (PERF-02).

**Both oracles this rehearsal drives, unchanged by the rehearsal itself:**

1. The pre-existing table suite in `internal/bench/regression_test.go` (`TestCheckRegression`).
2. Task 2's `internal/bench/baseline_gate_test.go`
   (`TestCheckRegressionAgainstCommittedBaseline`), which loads the COMMITTED
   `tools/bench/baseline.json` and drives the same gate with it — closing cross-AI review HIGH 4:
   the file ROADMAP success criterion 2 names is what participates in this proof, not an
   in-memory stand-in.

Neither oracle's source is touched by this rehearsal. The only file mutated, in either family, is
`internal/bench/regression.go`, and only its tolerance constant — never a guard, an error string,
a json tag or a test case.

## Pre-mutation cleanliness gate (review finding)

Before every tracked-file mutation AND revert, `git diff --quiet -- internal/bench/regression.go`
was asserted. Result across both families: **the gate never fired (exit 0 every time)** — no
pre-existing tracked edit was overwritten, and no revert was a destructive blind checkout.

The whole rehearsal ran under a shell `trap 'git checkout -- internal/bench/regression.go' EXIT`,
so an interrupted run could not leave a weakened gate in the tree (T-06-01).

---

## Family A — throughput tolerance

**Constant mutated:** `DefaultThroughputTolerance` (declared at `internal/bench/regression.go:9`).

**Mutation applied:** `const DefaultThroughputTolerance = 0.10` → `const
DefaultThroughputTolerance = 1.0`, so an 11%-slower throughput regression no longer exceeds the
band.

**Step 1 — pre-mutation cleanliness gate:** `git diff --quiet -- internal/bench/regression.go` →
exit code `0` (clean).

**Step 2 — mutation applied.**

**Step 3 — confirm the mutation actually applied (`git diff --stat -- internal/bench/regression.go`,
pasted verbatim, non-empty):**

```
 internal/bench/regression.go | 2 +-
 1 file changed, 1 insertion(+), 1 deletion(-)
```

**Step 4 — observe RED in BOTH oracles** (`go test -count=1 ./internal/bench/... -run
'TestCheckRegression' -v`, pasted verbatim — both `--- FAIL` lines that carry the band name):

```
=== RUN   TestCheckRegressionAgainstCommittedBaseline
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails
    baseline_gate_test.go:108: CheckRegression(committed baseline) = nil, want error
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails
--- FAIL: TestCheckRegressionAgainstCommittedBaseline (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes (0.00s)
    --- FAIL: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
=== RUN   TestCheckRegression
=== RUN   TestCheckRegression/clean_run:_faster_and_smaller,_under_ceiling_passes
=== RUN   TestCheckRegression/throughput_9%_slower:_within_band_passes
=== RUN   TestCheckRegression/throughput_11%_slower:_exceeds_band_fails
    regression_test.go:514: CheckRegression() = nil, want error
=== RUN   TestCheckRegression/peak_RSS_14%_larger:_within_band_passes
=== RUN   TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails
=== RUN   TestCheckRegression/above_absolute_ceiling_fails_even_when_relative_RSS_delta_is_in-band
=== RUN   TestCheckRegression/below_absolute_ceiling_passes
=== RUN   TestCheckRegression/zero_baseline_throughput_yields_a_clear_error,_not_a_panic
=== RUN   TestCheckRegression/GOOS/GOARCH_mismatch_between_baseline_and_current_fails_even_when_metrics_are_identical
=== RUN   TestCheckRegression/GOOS_differs_while_GOARCH_matches_fails
=== RUN   TestCheckRegression/GOARCH_differs_while_GOOS_matches_fails
=== RUN   TestCheckRegression/matching_GOOS/GOARCH_on_both_sides_passes
=== RUN   TestCheckRegression/unattributed_platform_on_both_sides_passes
=== RUN   TestCheckRegression/attributed_current_against_unattributed_baseline_fails
=== RUN   TestCheckRegression/runner_mismatch_between_baseline_and_current_fails_even_when_GOOS/GOARCH_match
=== RUN   TestCheckRegression/matching_runner_on_both_sides_passes
=== RUN   TestCheckRegression/empty_baseline_runner_against_non-empty_current_runner_fails
=== RUN   TestCheckRegression/non-empty_baseline_runner_against_empty_current_runner_fails
=== RUN   TestCheckRegression/unattributed_runner_on_both_sides_passes
=== RUN   TestCheckRegression/runner_mismatch_refused_before_numeric_comparison_even_when_metrics_would_pass
=== RUN   TestCheckRegression/scratch_fs_mismatch_between_baseline_and_current_fails_even_when_runner_and_GOOS/GOARCH_match
=== RUN   TestCheckRegression/matching_scratch_fs_on_both_sides_passes
=== RUN   TestCheckRegression/empty_baseline_scratch_fs_against_non-empty_current_scratch_fs_fails
=== RUN   TestCheckRegression/non-empty_baseline_scratch_fs_against_empty_current_scratch_fs_fails
=== RUN   TestCheckRegression/unattributed_scratch_fs_on_both_sides_passes
--- FAIL: TestCheckRegression (0.00s)
    --- PASS: TestCheckRegression/clean_run:_faster_and_smaller,_under_ceiling_passes (0.00s)
    --- PASS: TestCheckRegression/throughput_9%_slower:_within_band_passes (0.00s)
    --- FAIL: TestCheckRegression/throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegression/peak_RSS_14%_larger:_within_band_passes (0.00s)
    --- PASS: TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
    [... remaining 19 platform/runner/scratch-fs subtests: all PASS, omitted for brevity ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/bench	0.077s
FAIL
```

**Reading the RED, not paraphrasing it:** widening `DefaultThroughputTolerance` to `1.0` makes
`CheckRegression` return `nil` for the 11%-slower case, which SUPPRESSES the tolerance-violation
error branch at `internal/bench/regression.go:113-126` — that branch never runs. The RED a correct
rehearsal produces is exactly `CheckRegression() = nil, want error`
(`regression_test.go:514`)/`CheckRegression(committed baseline) = nil, want error`
(`baseline_gate_test.go:108`), each nested under the band-naming subtest
(`throughput_11%_slower:_exceeds_band_fails` /
`committed_baseline_throughput_11%_slower:_exceeds_band_fails`). No category-mismatch phrase
(platform/runner/scratch-filesystem mismatch) appears anywhere in this output — this is the
numeric band failing, not a category error (Pitfall 2 avoided).

**Step 5 — revert:** `git checkout -- internal/bench/regression.go`.

**Byte-clean proof:** `git diff --stat -- internal/bench/regression.go` after revert:

```
(empty)
```

**Green re-run** (`go test -count=1 ./internal/bench/... -run 'TestCheckRegression' -v`):

```
--- PASS: TestCheckRegressionAgainstCommittedBaseline (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
--- PASS: TestCheckRegression (0.00s)
    [... all 25 subtests PASS ...]
PASS
ok  	github.com/seanb4t/codegraph-go/internal/bench	0.057s
```

---

## Family B — peak-RSS tolerance

**Constant mutated:** `DefaultRSSTolerance` (declared at `internal/bench/regression.go:15`).

**Mutation applied:** `const DefaultRSSTolerance = 0.15` → `const DefaultRSSTolerance = 1.0`, so a
16%-larger peak-RSS growth no longer exceeds the band.

**Step 1 — pre-mutation cleanliness gate:** `git diff --quiet -- internal/bench/regression.go` →
exit code `0` (clean).

**Step 2 — mutation applied.**

**Step 3 — confirm the mutation actually applied (`git diff --stat -- internal/bench/regression.go`,
pasted verbatim, non-empty):**

```
 internal/bench/regression.go | 2 +-
 1 file changed, 1 insertion(+), 1 deletion(-)
```

**Step 4 — observe RED in BOTH oracles** (`go test -count=1 ./internal/bench/... -run
'TestCheckRegression' -v`, pasted verbatim — both `--- FAIL` lines that carry the band name):

```
=== RUN   TestCheckRegressionAgainstCommittedBaseline
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails
=== RUN   TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails
    baseline_gate_test.go:131: CheckRegression(committed baseline) = nil, want error
--- FAIL: TestCheckRegressionAgainstCommittedBaseline (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- FAIL: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
=== RUN   TestCheckRegression
=== RUN   TestCheckRegression/clean_run:_faster_and_smaller,_under_ceiling_passes
=== RUN   TestCheckRegression/throughput_9%_slower:_within_band_passes
=== RUN   TestCheckRegression/throughput_11%_slower:_exceeds_band_fails
=== RUN   TestCheckRegression/peak_RSS_14%_larger:_within_band_passes
=== RUN   TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails
    regression_test.go:514: CheckRegression() = nil, want error
=== RUN   TestCheckRegression/above_absolute_ceiling_fails_even_when_relative_RSS_delta_is_in-band
=== RUN   TestCheckRegression/below_absolute_ceiling_passes
=== RUN   TestCheckRegression/zero_baseline_throughput_yields_a_clear_error,_not_a_panic
=== RUN   TestCheckRegression/GOOS/GOARCH_mismatch_between_baseline_and_current_fails_even_when_metrics_are_identical
=== RUN   TestCheckRegression/GOOS_differs_while_GOARCH_matches_fails
=== RUN   TestCheckRegression/GOARCH_differs_while_GOOS_matches_fails
=== RUN   TestCheckRegression/matching_GOOS/GOARCH_on_both_sides_passes
=== RUN   TestCheckRegression/unattributed_platform_on_both_sides_passes
=== RUN   TestCheckRegression/attributed_current_against_unattributed_baseline_fails
=== RUN   TestCheckRegression/runner_mismatch_between_baseline_and_current_fails_even_when_GOOS/GOARCH_match
=== RUN   TestCheckRegression/matching_runner_on_both_sides_passes
=== RUN   TestCheckRegression/empty_baseline_runner_against_non-empty_current_runner_fails
=== RUN   TestCheckRegression/non-empty_baseline_runner_against_empty_current_runner_fails
=== RUN   TestCheckRegression/unattributed_runner_on_both_sides_passes
=== RUN   TestCheckRegression/runner_mismatch_refused_before_numeric_comparison_even_when_metrics_would_pass
=== RUN   TestCheckRegression/scratch_fs_mismatch_between_baseline_and_current_fails_even_when_runner_and_GOOS/GOARCH_match
=== RUN   TestCheckRegression/matching_scratch_fs_on_both_sides_passes
=== RUN   TestCheckRegression/empty_baseline_scratch_fs_against_non-empty_current_scratch_fs_fails
=== RUN   TestCheckRegression/non-empty_baseline_scratch_fs_against_empty_current_scratch_fs_fails
=== RUN   TestCheckRegression/unattributed_scratch_fs_on_both_sides_passes
--- FAIL: TestCheckRegression (0.00s)
    --- PASS: TestCheckRegression/clean_run:_faster_and_smaller,_under_ceiling_passes (0.00s)
    --- PASS: TestCheckRegression/throughput_9%_slower:_within_band_passes (0.00s)
    --- PASS: TestCheckRegression/throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegression/peak_RSS_14%_larger:_within_band_passes (0.00s)
    --- FAIL: TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
    [... remaining 19 platform/runner/scratch-fs subtests: all PASS, omitted for brevity ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/bench	0.089s
FAIL
```

**Reading the RED, not paraphrasing it:** widening `DefaultRSSTolerance` to `1.0` makes
`CheckRegression` return `nil` for the 16%-larger case, which SUPPRESSES the tolerance-violation
error branch at `internal/bench/regression.go:113-126` for the RSS delta the same way Family A
suppressed it for throughput. The RED is `CheckRegression() = nil, want error`
(`regression_test.go:514`)/`CheckRegression(committed baseline) = nil, want error`
(`baseline_gate_test.go:131`), each nested under the band-naming subtest
(`peak_RSS_16%_larger:_exceeds_band_fails` /
`committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails`). No category-mismatch phrase appears
anywhere in this output.

**Step 5 — revert:** `git checkout -- internal/bench/regression.go`.

**Byte-clean proof:** `git diff --stat -- internal/bench/regression.go` after revert:

```
(empty)
```

**Green re-run** (`go test -count=1 ./internal/bench/... -run 'TestCheckRegression' -v`):

```
--- PASS: TestCheckRegressionAgainstCommittedBaseline (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
--- PASS: TestCheckRegression (0.00s)
    [... all 25 subtests PASS ...]
PASS
ok  	github.com/seanb4t/codegraph-go/internal/bench	0.059s
```

---

## Failure-mode discipline (Pitfall 2)

Neither family's RED was a category-error RED (see `06-03-PLAN.md`'s `<action>` for what that
would look like and why it would prove nothing). Both REDs are the numeric band's own subtest
going red on `CheckRegression() = nil, want error` / `CheckRegression(committed baseline) = nil,
want error` — the assertion the mutated path actually emits, not the gate's own
tolerance-violation strings (which the mutation suppresses by construction). A category-error RED
would not have demonstrated anything about the numeric band; neither family produced one.

## Conclusion — FIXT-07 positive reading for BENCH-02

Both tolerance families were red-demonstrated against the suite in its committed form, each
reddening BOTH oracles (the pre-existing table AND the new committed-baseline test), each with a
byte-clean revert:

| Family | Constant mutated | Table subtest reddened | Committed-baseline subtest reddened | Revert | Green re-run |
|---|---|---|---|---|---|
| A — throughput | `DefaultThroughputTolerance` 0.10 → 1.0 | `TestCheckRegression/throughput_11%_slower:_exceeds_band_fails` | `TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails` | `git checkout -- internal/bench/regression.go` | `-run 'TestCheckRegression'` ok |
| B — peak RSS | `DefaultRSSTolerance` 0.15 → 1.0 | `TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails` | `TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails` | `git checkout -- internal/bench/regression.go` | `-run 'TestCheckRegression'` ok |

**The gate still fires against the committed baseline** — both halves of BENCH-02 are now
satisfied by demonstration: 06-01 removed the comparison runner, and this rehearsal proves
`CheckRegression` catches a real throughput regression and a real peak-RSS regression when driven
with the exact file ROADMAP success criterion 2 names, `tools/bench/baseline.json`, loaded off
disk by Task 2's `TestCheckRegressionAgainstCommittedBaseline`.

Post-rehearsal working tree: `git status --porcelain internal/bench/` is empty. **This phase's
commits carry no mutation byte** — every mutation above lived only in the transient working-tree
edit between "mutation applied" and "revert", never staged, never committed, confirmed by the
empty `git diff --stat -- internal/bench/regression.go` after each revert and the final
`POST_REHEARSAL_TREE_CLEAN=true` assertion recorded at Task 3's close.
