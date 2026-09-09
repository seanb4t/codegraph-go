---
phase: 07-guards-that-cannot-fire
verified: 2026-09-08T00:00:00Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - ".github/workflows/post-release-verify.yml"
  - ".planning/phases/07-guards-that-cannot-fire/07-01-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-01-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-02-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-02-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-03-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-03-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-04-PLAN.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-04-SUMMARY.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-CONTEXT.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md"
  - ".planning/phases/07-guards-that-cannot-fire/07-REVIEW.md"
  - "Taskfile.yml"
  - "internal/bench/regression.go"
  - "internal/bench/regression_test.go"
  - "internal/query/archtest/import_direction_test.go"
  - "internal/upgrade/release_workflow_shape_test.go"
  - "scripts/inject-cosign-key.sh"
covered_digest: "v1:sha256:684345637ea57f9121a6cde26d6d28f41292faaf0566c4c1864b9a0037c84d14"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 7: Guards That Cannot Fire — Verification Report

**Phase Goal:** Every guard in this repo's known cannot-fire set now fails when the property it claims to check is violated — each one proven by a RED demonstration against the actual failure condition, with the proof committed rather than asserted.
**Verified:** 2026-09-08
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

All five ROADMAP success criteria (GRD-01, GRD-02, GRD-03, GRD-04, GRD-06) were checked directly against the codebase — running the actual tests and scripts myself, reproducing a RED mutation live, and reading the fix code line-by-line — not by trusting SUMMARY.md claims.

| # | Truth | Status | Evidence |
| - | ----- | ------ | -------- |
| 1 | `CheckRegression` refuses a non-positive *current* throughput or peak-RSS reading, naming the degenerate field; historical `ceiling=1`/`PeakRSSBytes=0` frame used, not a synthetic one (GRD-01) | ✓ VERIFIED | `internal/bench/regression.go:120,123` emits `bench: invalid current: FilesPerSec/PeakRSSBytes must be positive, got …`, positioned after the two baseline checks and before `throughputDelta` (line 126) — order preserved per D-10. `go test -count=1 ./internal/bench/...` green. `07-MUTATION-LOG.md` family (a) pastes the pre-fix transcript: PeakRSSBytes row got `nil`, FilesPerSec row got the wrong error (`throughput regressed 100.0%`) — the two distinct historical failure modes, reproduced from backlog 999.4, not asserted |
| 2 | Archtest forbids `internal/query` from resolving a wire-layer dependency or the `internal/indexer` root in its resolved transitive set, reports packages loaded, and carries a positive control (GRD-02) | ✓ VERIFIED | `internal/query/archtest/import_direction_test.go` walks `pkg.Imports` recursively (`transitiveDeps`), not a single hop or regex. Forbidden set is the D-02 union (4 entries) checked over every loaded variant; the indexer-root rule is exact-path, production-scope only, correctly excluding `engine_test.go`. Zero-package and missing-production-package are both fatal. Positive controls assert `graphstore`, `goextract`, `nodeid`, and `parser` (transitive-only witness) are present. `go test -count=1 ./internal/query/archtest/` green. `07-MUTATION-LOG.md` family (b) pastes two distinct RED transcripts (`connectrpc.com/connect`, `internal/indexer`), both reverted byte-clean (confirmed live: `git diff --quiet` clean before/after) |
| 3 | `release:dry-run-signed`'s guard asserts the `--key=` injection actually occurred (not just an additions-only diff), and a broken anchor turns it red (GRD-03) | ✓ VERIFIED | `scripts/inject-cosign-key.sh` (mode 100755) counts injected `--key=` lines and refuses any count ≠ 1. Live re-run against the real `.goreleaser.yaml` reproduced exactly the mutation log's claim: `injected --key= lines: 1`, exit 0. `Taskfile.yml:1839,2355` — both `release:dry-run-signed` and `release:rehearse-notarize` call the one script; no inline injection block remains (`rg -c 'keyline' Taskfile.yml` → 0). `07-MUTATION-LOG.md` family (c) pastes the re-indented-anchor RED (`found 0`, exit 1) against a temp-dir copy, with the committed `.goreleaser.yaml` proven byte-unchanged |
| 4 | A test parses `post-release-verify.yml`'s job map, reports jobs inspected (>0), and fails when the guard is removed or inverted on any job (GRD-04) | ✓ VERIFIED | `internal/upgrade/release_workflow_shape_test.go` — `TestPostReleaseJobsDeclareConclusionGuard` compares each job's parsed `if:` against one verbatim const, no fixed job-id list, no normaliser; `TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError` companion exists. I live-reproduced the d1 "guard removed on gatekeeper" mutation myself (not just read the log): identical failure text to `07-MUTATION-LOG.md` family (d), reverted byte-clean. `go test -count=1 ./internal/upgrade/...` green |
| 5 | A committed mutation log carries four demonstrations (GRD-01..04), none summarised away, each with pasted failing output and a byte-clean revert (GRD-06) | ✓ VERIFIED | `07-MUTATION-LOG.md` contains exactly 4 `## Family` sections (a-d) plus a one-line GRD-05 deletion record (D-09) and two addendum sections (CR-01, WR-01 fixes from code review). Every family pastes verbatim `--- FAIL` transcripts (confirmed present, not paraphrased) and either documents an explicit "no revert" shape deviation (family a: RED was absence-of-fix) or a live-verified byte-clean revert (families b, c, d) |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/bench/regression.go` | Two new current-metrics positivity checks | ✓ VERIFIED | Lines 120, 123; correct order relative to baseline checks and delta math |
| `internal/bench/regression_test.go` | Two new degenerate-current table rows | ✓ VERIFIED | Both present, both pass, `ceiling: 1,` literal confirmed used for the historical row |
| `internal/query/archtest/import_direction_test.go` | New persisted archtest package | ✓ VERIFIED | Exists, package doc names T-01-18, D-01 scoping documented, CR-01 fix (`PrintErrors`) present |
| `scripts/inject-cosign-key.sh` | New extracted script with count assertion | ✓ VERIFIED | Mode 100755, live-run confirms `injected --key= lines: 1`; WR-01 fix (quote/backslash refusal) present |
| `Taskfile.yml` | Both release targets call the one script | ✓ VERIFIED | Two call sites (`:1839`, `:2355`), no inline injection block remains |
| `.github/workflows/post-release-verify.yml` | Unchanged (guard target, not guard itself) | ✓ VERIFIED | `git diff --quiet` clean; five jobs each carry the verbatim disjunct |
| `internal/upgrade/release_workflow_shape_test.go` | New conclusion-guard test + companion; tap test deleted | ✓ VERIFIED | `TestPostReleaseJobsDeclareConclusionGuard` + `_EmptyDocIsError` present; `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` absent (rg count 0); `homebrewTapCredentialNames` survives (6 references) |
| `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` | Header + 4 families + GRD-05 record | ✓ VERIFIED | All present, committed, ends with closing non-vacuity section |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `regression_test.go` errHint strings | `regression.go` error text | Literal prefix match | ✓ WIRED | `containsFold` check passes; errHints (`invalid current: PeakRSSBytes`/`FilesPerSec`) are true prefixes of the emitted error strings |
| `release:dry-run-signed` / `release:rehearse-notarize` | `scripts/inject-cosign-key.sh` | Task target invocation | ✓ WIRED | Both call sites pass `<committed-config> <generated-config> <cosign-key>` positionally; live-run confirmed working |
| `TestPostReleaseJobsDeclareConclusionGuard` | `post-release-verify.yml` | `decodeFullWorkflowDoc` parse via `fullWorkflowJob.If` | ✓ WIRED | Live mutation reproduced the exact documented failure; field flows into the comparison, not dropped |
| `internal/query/archtest` | `internal/query`'s real import graph | `go/packages.Load` with `NeedDeps`+`Tests:true` | ✓ WIRED | Positive controls (graphstore, goextract, nodeid, parser) all present in the live green run; not a fixture graph |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full guard suite green | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/bench/ ./internal/query/archtest/ ./internal/upgrade/` | `ok` × 3 | ✓ PASS |
| `go build ./...` | `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 | ✓ PASS |
| GRD-03 script live run | `bash scripts/inject-cosign-key.sh .goreleaser.yaml <tmp> <tmp>` | `injected --key= lines: 1`, exit 0 | ✓ PASS |
| GRD-04 d1 live reproduction | `if:` line deleted from `gatekeeper`, re-run `TestPostReleaseJobsDeclareConclusionGuard` | Identical failure text to mutation log, byte-clean revert confirmed | ✓ PASS |
| No stray tracked-file mutation | `git status --short` at verification start/end | Clean | ✓ PASS |
| `go.mod`/`go.sum` untouched | `git diff --quiet -- go.mod go.sum` | exit 0 | ✓ PASS |

### Code Review Follow-Through

`07-REVIEW.md` found 1 critical (CR-01), 1 warning (WR-01), 1 info (IN-01). Both CR-01 and WR-01 were fixed after review (commits a90b5457, bf3a4562, def48e4a per task notes) with mutation-log addenda; both fixes verified present in the live source:
- CR-01 fix (`packages.PrintErrors(pkgs)` non-zero-count refusal) — present at `import_direction_test.go:111`.
- WR-01 fix (quote/backslash refusal before writing) — present at `scripts/inject-cosign-key.sh:58`.
- IN-01 (hand-rolled `containsFold` instead of `strings.ToLower`) — left as-is, correctly: it is info-level only, cosmetic, and does not affect guard discrimination.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`) or warning-level markers (`TODO`/`HACK`/`PLACEHOLDER`) found in any file this phase modified. `Taskfile.yml:4713`'s `XXXXXX` is a `mktemp` template pattern, not a debt marker.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| GRD-01 | 07-01 | `CheckRegression` current-metrics positivity | ✓ SATISFIED | Verified above, truth #1 |
| GRD-02 | 07-02 | `internal/query` dependency-direction archtest | ✓ SATISFIED | Verified above, truth #2 |
| GRD-03 | 07-03 | `release:dry-run-signed` injection guard | ✓ SATISFIED | Verified above, truth #3 |
| GRD-04 | 07-04 | `post-release-verify.yml` conclusion guard | ✓ SATISFIED | Verified above, truth #4 |
| GRD-06 | 07-01/02/03/04 | Mutation log, four families | ✓ SATISFIED | Verified above, truth #5 |

No orphaned requirements: REQUIREMENTS.md maps exactly these 5 IDs to Phase 7, all appear in plan frontmatter, all marked Complete in REQUIREMENTS.md's own tracking table (lines 106-110), and the per-phase total (5) matches the milestone tally at line 140. GRD-05 was correctly de-scoped to v2 (declined 2026-09-08, REQUIREMENTS.md line 79) and is NOT part of this phase's requirement set — its deletion (not rewrite) is documented in D-09 and recorded as a one-line entry in the mutation log, consistent with the phase goal (D-09 explicitly makes GRD-05 an intentional non-demonstration, not a gap). GRD-07/GRD-08 remain correctly out of scope (declined for this milestone, REQUIREMENTS.md line 77-78).

### Human Verification Required

None. All five must-haves are mechanically verifiable (test runs, script invocations, source inspection, live mutation reproduction) and were verified directly rather than deferred.

### Gaps Summary

None. All five ROADMAP success criteria hold, all five requirement IDs are accounted for and satisfied, both code-review-critical/warning fixes are present in the live source, the phase's own guard suite is green, the full build succeeds, and no tracked file was left in a dirty state by any of the phase's own RED demonstrations (confirmed both by reading the mutation log and by independently reproducing two of the four RED demonstrations live and reverting them myself).

---

_Verified: 2026-09-08_
_Verifier: Claude (gsd-verifier)_
