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

## Family (c) — GRD-03: `scripts/inject-cosign-key.sh` additions-only diff guard (todo T-02-08)

**Guard:** `scripts/inject-cosign-key.sh`, called by both `release:dry-run-signed` and `release:rehearse-notarize`. It injects `--key=<path>` into a generated copy of a GoReleaser config at the `sign-blob` anchor, asserts the copy differs from its input by additions only, and — the new positive assertion this family proves — counts the added `--key=` lines and refuses any count other than exactly 1.

**What this proves:** the additions-only diff guard alone passes vacuously when the awk anchor stops matching (a re-indent, a requote, a renamed key): the injection becomes a no-op, the generated config is byte-identical to the input, the diff is empty, and every additions-only check is trivially satisfied by that empty diff. The count assertion closes that gap by refusing a count of zero.

**Mutation applied — deliberate deviation from the tracked-file mutation shape used by families (a)/(b):** no tracked file was mutated, and there is no `git checkout` revert step in this family. The mutation instead perturbs a **copy of the committed `.goreleaser.yaml` written into a temporary directory** — the committed release config must never be edited to prove a guard, since it is the actual file every real release pipeline invocation reads. The copy's `sign-blob` anchor line was re-indented by two extra spaces (`      - "sign-blob"` → `        - "sign-blob"`), which is enough to break the awk pattern's exact six-space-indent match while the file remains valid YAML. A reader should read the absence of a `git checkout` here as this deliberate choice, not an omission — see families (a) and (b) above for the tracked-file-mutation shape this family departs from.

**Pre-mutation cleanliness gate:** `git diff --quiet -- .goreleaser.yaml` → exit 0 (clean), checked immediately before writing the re-indented copy.

**Command (`sed` writes the re-indented copy into a temp dir; the script is invoked against that copy only):**

```
T=$(mktemp -d)
sed 's/^      - "sign-blob"$/        - "sign-blob"/' .goreleaser.yaml > "$T/reindented.yaml"
bash scripts/inject-cosign-key.sh "$T/reindented.yaml" "$T/gen.yaml" "$T/cosign.key"
```

**Observed failure (pasted verbatim, exit code 1):**

```
generated config differs from /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.TLfOKX6Wir/reindented.yaml by additions only:

injected --key= lines: 0
::error::expected exactly 1 injected --key= line, found 0 — zero means the sign-blob anchor no longer matches (the hang case), two or more means a duplicated sign-blob block
```

The additions-only checks passed on the empty diff — the vacuity this family exists to close — and the new count assertion was the only thing that refused, reporting a count of zero and exiting 1.

**Post-demonstration cleanliness gate:** `git diff --quiet -- .goreleaser.yaml` → exit 0 (clean) — the committed config was never touched; the entire mutation lived in the temporary directory shown above.

**Green re-run, against the real committed `.goreleaser.yaml` (`bash scripts/inject-cosign-key.sh .goreleaser.yaml "$T/gen.yaml" "$T/cosign.key"`, exit 0):**

```
generated config differs from .goreleaser.yaml by additions only:
330a331
>       - "--key=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.DLa0ndkBvu/cosign.key"
injected --key= lines: 1
```

---

## Family (d) — GRD-04: `post-release-verify.yml` event-aware conclusion guard (todo T-02-18)

**Test name:** `TestPostReleaseJobsDeclareConclusionGuard` (`internal/upgrade/release_workflow_shape_test.go`) — proves every job in `post-release-verify.yml` carries the event-aware conclusion guard verbatim on its `if:` line, with no fixed job-id list and no normaliser, and its companion `TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError` (a zero-job document is a decode error, never a usable zero value).

**What this proves:** ROADMAP criterion 4 requires the guard to fail when removed OR inverted. Removal alone would be satisfied by a bare presence check (`if != ""`); only the inverted case proves the assertion discriminates on *content*, not merely on presence — a plausible-looking but logically inverted expression must still fail.

Two independent sub-demonstrations, both mutating the same tracked production file, `.github/workflows/post-release-verify.yml`, each fully reverted before the next began.

### d1 — guard removed (`gatekeeper` job)

**Pre-mutation gate:** `git diff --quiet -- .github/workflows/post-release-verify.yml` → exit 0 (clean), checked immediately before the mutation.

**Mutation applied:** Deleted the entire `if:` line from the `gatekeeper` job only, leaving the other four jobs (`resolve-tag`, `verify-supply-chain`, `self-upgrade`, `notarized-suite`) untouched:

```diff
   gatekeeper:
     name: "Gatekeeper verdict (darwin/${{ matrix.goarch }})"
     needs: resolve-tag
-    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
     strategy:
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/upgrade/ -run 'TestPostReleaseJobsDeclareConclusionGuard$'`):**

```
    release_workflow_shape_test.go:1663: job "gatekeeper" if: = "", want "github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'" (the event-aware conclusion guard, verbatim)
    release_workflow_shape_test.go:1669: inspected 5 job(s) in ../../.github/workflows/post-release-verify.yml for the event-aware conclusion guard
--- FAIL: TestPostReleaseJobsDeclareConclusionGuard (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.205s
FAIL
```

**Revert:** `git checkout -- .github/workflows/post-release-verify.yml`.

**Post-revert gate:** `git diff --quiet -- .github/workflows/post-release-verify.yml` → exit 0 (clean).

### d2 — guard inverted (`resolve-tag` job)

**Pre-mutation gate:** `git diff --quiet -- .github/workflows/post-release-verify.yml` → exit 0 (clean), re-checked immediately before this mutation.

**Mutation applied:** On the `resolve-tag` job only, inverted the conclusion comparison so the expression requires the producing run to have NOT succeeded — the event-name half and every other job left untouched. This mutation keeps an `if:` present and keeps it plausible-looking, which is exactly why a bare presence check or a normaliser would miss it:

```diff
     name: resolve and validate the tag under verification
     # Event-aware conclusion guard (T-01-38) — see the header comment. This
     # exact disjunct is required, verbatim, on every job in this file.
-    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
+    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion != 'success'
     runs-on: namespace-profile-linux-amd64-4x8
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/upgrade/ -run 'TestPostReleaseJobsDeclareConclusionGuard$'`):**

```
    release_workflow_shape_test.go:1663: job "resolve-tag" if: = "github.event_name != 'workflow_run' || github.event.workflow_run.conclusion != 'success'", want "github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'" (the event-aware conclusion guard, verbatim)
    release_workflow_shape_test.go:1669: inspected 5 job(s) in ../../.github/workflows/post-release-verify.yml for the event-aware conclusion guard
--- FAIL: TestPostReleaseJobsDeclareConclusionGuard (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.154s
FAIL
```

The two transcripts name different jobs (`gatekeeper` vs. `resolve-tag`) and different found-values (empty string vs. a plausible-looking but inverted expression), confirming the removed and inverted modes are distinguished and that the assertion discriminates on content, not presence.

**Revert:** `git checkout -- .github/workflows/post-release-verify.yml`.

**Post-revert gate:** `git diff --quiet -- .github/workflows/post-release-verify.yml` → exit 0 (clean).

**Byte-clean proof:** `git status --porcelain -- .github/workflows/post-release-verify.yml` is empty after both reverts — four cleanliness-gate checks total (before and after each of the two mutations) all returned exit 0.

**Green re-run (`GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/...`):**

```
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.426s
```

---

## GRD-05 record — tap App secret-distinctness test: deleted, not demonstrated (D-09)

`TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` was **deleted** in Task 2 of this plan (07-04-PLAN.md) under D-09 rather than rewritten: it compared two constant lists declared inside the test file and read no workflow, so it could only fail if the test file itself was edited. GRD-05 is recorded in REQUIREMENTS.md as a v2 item declined for this milestone. A deletion carries no RED demonstration by construction — there is nothing here to prove RED against, since the defect was the test's existence, not its content.

---

## Closing — phase-wide non-vacuity property (ROADMAP criterion 5)

This log carries **four** demonstration families (a, b, c, d) — GRD-01 through GRD-04 — plus the GRD-05 deletion record above, which is explicitly not a fifth demonstration. Every family's RED evidence is pasted **verbatim failing output**, never a summary, and every tracked-file mutation used to produce that output (families b, c-copy-only, d) was proven reverted **byte-clean** via `git diff --quiet` both immediately before and immediately after. Family (a) is the one documented exception to the tracked-file-mutation shape: its RED condition was the absence of the fix rather than a deliberate mutation of correct code, so it has no revert step by construction, not by omission.
