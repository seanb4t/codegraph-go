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

## Family (b) — GRD-02: `internal/query` dependency-direction archtest (T-01-18)

**Test name:** `TestQueryImportsNoWireLayerOrIndexerRoot` (`internal/query/archtest/import_direction_test.go`) — proves `internal/query`'s resolved transitive dependency set contains none of the forbidden wire-layer paths (`internal/uiserver`, `internal/mcp`, `internal/uiproto`, `connectrpc.com/connect`), checked over every loaded package variant, and that the production compilation unit's resolved set does not contain the `internal/indexer` root, while `internal/indexer/goextract` and `internal/indexer/nodeid` remain allowed leaves.

Two independent sub-demonstrations, both mutating the same tracked production file, `internal/query/traverse.go`, each fully reverted before the next began.

### b1 — wire layer (`connectrpc.com/connect`)

**Pre-mutation gate:** `git diff --quiet -- internal/query/traverse.go` → exit 0 (clean), checked immediately before the mutation.

**Mutation applied:** Added a blank import `_ "connectrpc.com/connect"` to `internal/query/traverse.go`'s import block. `connectrpc.com/connect` was used rather than a first-party wire package (`internal/uiserver` or `internal/mcp`) because both of those already import `internal/query` — using either would create an import cycle and fail at Go compile time rather than through the archtest's own assertion, which would prove nothing about the guard. `connectrpc.com/connect` is external, already present in `go.mod` (as an indirect dependency, `v1.20.0`), and creates no cycle.

**Observed failure (pasted, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/`):**

```
=== RUN   TestQueryImportsNoWireLayerOrIndexerRoot
    import_direction_test.go:127: package github.com/seanb4t/codegraph-go/internal/query resolves forbidden wire-layer dependency connectrpc.com/connect in its transitive dependency set (the dependency may be indirect) — internal/query must not depend on the wire layer it is consumed by
    import_direction_test.go:127: package github.com/seanb4t/codegraph-go/internal/query resolves forbidden wire-layer dependency connectrpc.com/connect in its transitive dependency set (the dependency may be indirect) — internal/query must not depend on the wire layer it is consumed by
    import_direction_test.go:127: package github.com/seanb4t/codegraph-go/internal/query_test resolves forbidden wire-layer dependency connectrpc.com/connect in its transitive dependency set (the dependency may be indirect) — internal/query must not depend on the wire layer it is consumed by
    import_direction_test.go:127: package github.com/seanb4t/codegraph-go/internal/query.test resolves forbidden wire-layer dependency connectrpc.com/connect in its transitive dependency set (the dependency may be indirect) — internal/query must not depend on the wire layer it is consumed by
    import_direction_test.go:169: loaded 7 packages; production internal/query resolved 397 transitive dependencies
--- FAIL: TestQueryImportsNoWireLayerOrIndexerRoot (0.11s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/query/archtest	0.297s
FAIL
```

**Revert:** `git checkout -- internal/query/traverse.go`.

**Post-revert gate:** `git diff --quiet -- internal/query/traverse.go` → exit 0 (clean).

### b2 — indexer root (`internal/indexer`, production scope)

**Pre-mutation gate:** `git diff --quiet -- internal/query/traverse.go` → exit 0 (clean), re-checked immediately before this mutation.

**Mutation applied:** Added a blank import `_ "github.com/seanb4t/codegraph-go/internal/indexer"` to the same import block in `internal/query/traverse.go`. This creates no import cycle — the `internal/indexer` root does not itself import `internal/query`.

**Observed failure (pasted, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/`):**

```
=== RUN   TestQueryImportsNoWireLayerOrIndexerRoot
    import_direction_test.go:139: production package github.com/seanb4t/codegraph-go/internal/query resolves the forbidden internal/indexer root github.com/seanb4t/codegraph-go/internal/indexer in its transitive dependency set (the dependency may be indirect) — only internal/indexer/goextract and internal/indexer/nodeid are allowed leaves; internal/query/engine_test.go is the one legitimate in-package importer of the root, and this rule is scoped to exclude only that test file, not to permit the root from production code
    import_direction_test.go:169: loaded 7 packages; production internal/query resolved 425 transitive dependencies
--- FAIL: TestQueryImportsNoWireLayerOrIndexerRoot (0.08s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/query/archtest	0.223s
FAIL
```

**Revert:** `git checkout -- internal/query/traverse.go`.

**Post-revert gate:** `git diff --quiet -- internal/query/traverse.go` → exit 0 (clean).

The two transcripts differ (different forbidden path named in each), confirming each forbidden set discriminates on its own rather than one rule masking the other.

**Byte-clean proof:** `git status --porcelain -- internal/query/traverse.go` is empty after both reverts — four cleanliness-gate checks total (before and after each of the two mutations) all returned exit 0.

**Green re-run (`GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/`):**

```
ok  	github.com/seanb4t/codegraph-go/internal/query/archtest	0.148s
```

---
