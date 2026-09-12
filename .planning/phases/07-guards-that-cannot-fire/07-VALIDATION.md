---
phase: "07"
slug: "guards-that-cannot-fire"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-09"
---

# Phase 07 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.26.6 via `GOTOOLCHAIN=go1.26.6`), shellcheck 0.x, task (go-task) |
| **Config file** | none — `go.mod` at repo root; no test config file |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/ ./internal/query/archtest/ ./internal/upgrade/ && shellcheck scripts/inject-cosign-key.sh` |
| **Full suite command** | `task test` |
| **Estimated runtime** | quick: ~1 s (measured 2026-09-09: 0.09 s + 0.37 s + 0.43 s); full: not re-measured this audit |

---

## Sampling Rate

- **After every task commit:** Run the quick run command
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | GRD-01 | T-07-02 | Two degenerate-current table rows fail against the pre-fix `CheckRegression` (RED proof, one-time) | unit (RED) | one-time RED: `go test ./internal/bench/ -run TestCheckRegression` produced exactly 2 `--- FAIL: …/degenerate_current` lines against the pre-fix build (pasted in 07-MUTATION-LOG.md family (a)); standing re-check: `test "$(rg -c -- '--- FAIL: TestCheckRegression/degenerate_current' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md \|\| true)" -ge 2` | ✅ | ✅ green |
| 07-01-02 | 01 | 1 | GRD-01 | T-07-01, T-07-04 | `CheckRegression` returns `invalid current: FilesPerSec` / `invalid current: PeakRSSBytes` for non-positive current readings | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/...` (27 `TestCheckRegression` subtests green) and `test "$(rg -v '^\s*//' internal/bench/regression.go \| rg -c 'bench: invalid current: (FilesPerSec\|PeakRSSBytes) must be positive' \|\| true)" = "2"` | ✅ | ✅ green |
| 07-01-03 | 01 | 1 | GRD-06 | T-07-02, T-07-03 | Mutation log carries family (a) with pasted pre-fix failing output and the cleanliness-gate line | artifact | `test "$(rg -c -- '--- FAIL: TestCheckRegression/degenerate_current' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md \|\| true)" -ge 2 && rg -q 'git diff --quiet -- internal/bench/regression.go' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` | ✅ | ✅ green |
| 07-02-01 | 02 | 2 | GRD-02 | T-07-05, T-07-06, T-07-07, T-07-08 | Archtest refuses a zero-package, partial (`PrintErrors`), or missing-production load; forbids wire-layer and indexer-root transitively; logs a positive count | arch | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/` and `test "$(GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/ 2>&1 \| rg -c 'loaded [1-9][0-9]* package' \|\| true)" -ge 1` | ✅ | ✅ green |
| 07-02-02 | 02 | 2 | GRD-02 | T-07-09 | Both forbidden sets shown RED via real imports, tree restored byte-clean | mutation (one-time) | standing re-check: `git diff --quiet -- internal/query/traverse.go && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/` | ✅ | ✅ green |
| 07-02-03 | 02 | 2 | GRD-06 | T-07-06, T-07-07 | Family (b) carries two verbatim RED transcripts; T-01-18 todo moved to completed | artifact | `test "$(rg -c -- '--- FAIL: TestQueryImportsNoWireLayerOrIndexerRoot' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md \|\| true)" -ge 2 && test -n "$(git ls-files -- .planning/todos/completed/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md)"` | ✅ | ✅ green |
| 07-03-01 | 03 | 3 | GRD-03 | T-07-10, T-07-11, T-07-12, T-07-13 | Script injects exactly one `--key=` line into a generated copy, reports the count, lints clean, refuses bad arity / unreadable input / quote-unsafe key path | script | `T=$(mktemp -d) && bash scripts/inject-cosign-key.sh .goreleaser.yaml "$T/gen.yaml" "$T/cosign.key" \| rg -q 'injected --key= lines: 1' && shellcheck scripts/inject-cosign-key.sh` | ✅ | ✅ green |
| 07-03-02 | 03 | 3 | GRD-03 | T-07-13 | Both release targets call the script (before goreleaser runs); no inline `keyline` block survives | shape | `task --list-all && test "$(rg -c --include-zero 'scripts/inject-cosign-key.sh' Taskfile.yml)" = "2" && test "$(rg -c --include-zero 'keyline' Taskfile.yml)" = "0"` — **corrected**: the plan's `rg -c 'keyline' Taskfile.yml \|\| true` prints an empty string, not `0`, on zero matches (ripgrep 15.2), so its `= "0"` comparison can never pass in the green state; the 07-03 executor recorded an `\|\| echo 0` fallback | ✅ | ✅ green |
| 07-03-03 | 03 | 3 | GRD-03, GRD-06 | T-07-10, T-07-14 | Re-indented anchor is refused with `found 0`; committed `.goreleaser.yaml` untouched; family (c) pasted | script (re-runnable RED) | `T=$(mktemp -d) && sed 's/^      - "sign-blob"$/        - "sign-blob"/' .goreleaser.yaml > "$T/reindented.yaml" && bash scripts/inject-cosign-key.sh "$T/reindented.yaml" "$T/gen.yaml" "$T/cosign.key" 2>&1 \| rg -q 'found 0' && git diff --quiet -- .goreleaser.yaml` | ✅ | ✅ green |
| 07-04-01 | 04 | 4 | GRD-04 | T-07-15, T-07-16, T-07-17, T-07-19 | Every job in `post-release-verify.yml` carries the conclusion guard verbatim; zero-job parse is an error; inspected count logged | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/ -run 'TestPostReleaseJobsDeclareConclusionGuard'` (2 tests green) and `test "$(GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/upgrade/ -run 'TestPostReleaseJobsDeclareConclusionGuard$' 2>&1 \| rg -c 'inspected [1-9][0-9]* job' \|\| true)" -ge 1` | ✅ | ✅ green |
| 07-04-02 | 04 | 4 | GRD-04 | T-07-18 | Tautological tap-secrets test deleted outright; `homebrewTapCredentialNames` survives; package green | shape | `test "$(rg -c --include-zero 'TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets' internal/upgrade/release_workflow_shape_test.go)" = "0" && test "$(rg -c --include-zero 'homebrewTapCredentialNames' internal/upgrade/release_workflow_shape_test.go)" -ge 3 && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/` — **corrected**: same `rg -c` empty-on-zero defect as 07-03-02 | ✅ | ✅ green |
| 07-04-03 | 04 | 4 | GRD-04, GRD-06 | T-07-20 | Family (d) carries two RED transcripts (removed, inverted); workflow byte-clean; exactly four families; all four folded todos moved | artifact | `test "$(rg -c -- '--- FAIL: TestPostReleaseJobsDeclareConclusionGuard' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md \|\| true)" -ge 2 && git diff --quiet -- .github/workflows/post-release-verify.yml && test "$(rg -c '^## Family' .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md \|\| true)" = "4" && test -z "$(git ls-files -- .planning/todos/pending/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md .planning/todos/pending/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md .planning/todos/pending/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md .planning/todos/pending/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md)"` — **corrected**: the plan's `pending todos == 1` count was true at plan close (ad51bd96, 20:54) and drifted when code review filed the graphstore-archtest sibling todo (1879aea1, 21:10); the assertion now names the four files the phase moved | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Audit method (2026-09-09): all 41 declared `<automated>` commands were executed verbatim at HEAD e3c1ca2b, not read. 37 passed. The 4 that did not are the three corrected rows above plus 07-01-01's one-time RED count (post-fix the count is 0 by design; the standing evidence is the pasted transcript). Test counts were taken from `-v` output, not from exit status: `TestCheckRegression` 27 subtests, `TestQueryImportsNoWireLayerOrIndexerRoot` 1, `TestPostReleaseJobsDeclareConclusionGuard` + `_EmptyDocIsError` 2.

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `release:dry-run-signed` and `release:rehearse-notarize` run end to end through the rewired script | GRD-03 (rewiring only; the guard itself is automated above) | Both targets declare darwin-host, zig, syft and cosign preconditions and invoke a real GoReleaser build; D-05/D-07 scope automated verification to the Taskfile parse plus the script's standalone behaviour | On a darwin host with zig, syft and cosign installed: `task release:dry-run-signed`; confirm the log shows `injected --key= lines: 1` before the goreleaser build starts and no keyless OIDC prompt appears |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-09-09
