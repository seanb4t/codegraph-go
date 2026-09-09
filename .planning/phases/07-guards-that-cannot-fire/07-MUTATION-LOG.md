# 07-MUTATION-LOG — Guards That Cannot Fire

**Phase:** 07-guards-that-cannot-fire
**Date:** 2026-09-08
**Scope:** Four RED demonstrations, one per guard family (GRD-01 through GRD-04), each proving its guard fails against the real defect condition before the fix is called done. GRD-05 (`TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`) is a **deletion decision** (D-09), not a demonstration — it is removed rather than rewritten, and gets a one-line record below rather than a mutation entry, per D-11.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a destructive blind checkout of someone else's in-flight work. Every family entry below records this gate's result at the point it was checked.

---

## Family (a) — GRD-01: `CheckRegression` current-metrics positivity (backlog 999.4)

**Test name:** `TestCheckRegression` — subtests `degenerate current PeakRSSBytes is refused rather than read as no regression` and `degenerate current FilesPerSec is refused rather than reported as a throughput regression`.

**Pre-mutation gate:** `git diff --quiet -- internal/bench/regression.go` → exit 0 (clean), checked immediately before the RED run below.

**Mutation applied:** **None** — this family has **no tracked-file mutation and no revert step**, and that is a deliberate shape deviation from families (b)/(c)/(d) below, not an omission. The RED condition here is the *absence* of the fix, not a deliberately-broken copy of otherwise-correct code: the two new table rows were committed to `internal/bench/regression_test.go` first (RED phase, TDD) and watched fail against the byte-identical, already-committed `internal/bench/regression.go` — the production file the codebase shipped with backlog 999.4 still open. There is nothing to revert because nothing in the production file was changed to produce this failure; the failure is the bug itself, observed directly.

**Observed failure (pasted, `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/ -run TestCheckRegression`):**

```
--- FAIL: TestCheckRegression (0.00s)
    --- FAIL: TestCheckRegression/degenerate_current_PeakRSSBytes_is_refused_rather_than_read_as_no_regression (0.00s)
        regression_test.go:549: CheckRegression() = nil, want error
    --- FAIL: TestCheckRegression/degenerate_current_FilesPerSec_is_refused_rather_than_reported_as_a_throughput_regression (0.00s)
        regression_test.go:556: error "bench: throughput regressed 100.0% (budget: 10.0%): baseline=100.00 files/s current=0.00 files/s" does not mention expected hint "invalid current: FilesPerSec"
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/bench	0.153s
FAIL
```

The first subtest reproduces the historical Phase 10 audit frame exactly (`ceiling=1`, `current.PeakRSSBytes = 0`, otherwise-matching): the unfixed `CheckRegression` returned `nil` — a broken measurement read as "no regression". The second subtest is the companion case (`current.FilesPerSec = 0`): the unfixed build DID return an error, but the wrong one — a `"throughput regressed 100.0%"` message that misattributes a broken measurement as a real regression, failing the `errHint` assertion for `"invalid current: FilesPerSec"` rather than the `wantErr`/nil check. These are the two distinct failure modes `<behavior>` in the plan called for; getting one nil and one wrong-error confirms the rows discriminate correctly rather than both failing the same trivial way.

**Revert:** None (see "Mutation applied" above — there is nothing to revert).

**Byte-clean proof:** `git diff --quiet -- internal/bench/regression.go` exited 0 both immediately before this RED run and at the moment it was captured — `regression.go` was never edited to produce this failure, so there is no diff to clean up.

**Green re-run (after Task 2's fix landed, `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/...`):**

```
ok  	github.com/seanb4t/codegraph-go/internal/bench	0.055s
```

Both `degenerate_current_*` subtests pass alongside the full existing table, confirming `CheckRegression` now refuses a non-positive `current.FilesPerSec` or `current.PeakRSSBytes` with a named error rather than treating it as no regression or misreporting it as one.

---
