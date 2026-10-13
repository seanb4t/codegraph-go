---
phase: 04-fragment-required-gate
verified: 2026-10-13T17:00:00Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - .github/workflows/ci.yml
  - .github/workflows/fragment-required.yml
  - .planning/phases/04-fragment-required-gate/04-01-PLAN.md
  - .planning/phases/04-fragment-required-gate/04-01-SUMMARY.md
  - .planning/phases/04-fragment-required-gate/04-02-PLAN.md
  - .planning/phases/04-fragment-required-gate/04-02-SUMMARY.md
  - .planning/phases/04-fragment-required-gate/04-03-PLAN.md
  - .planning/phases/04-fragment-required-gate/04-03-SUMMARY.md
  - .planning/phases/04-fragment-required-gate/04-04-PLAN.md
  - .planning/phases/04-fragment-required-gate/04-04-SUMMARY.md
  - .planning/phases/04-fragment-required-gate/04-05-PLAN.md
  - .planning/phases/04-fragment-required-gate/04-05-SUMMARY.md
  - Taskfile.yml
  - internal/upgrade/taskfile_shape_test.go
  - scripts/fragment_policy.py
  - scripts/test_fragment_policy.py
covered_digest: "v3:sha256:5d57bd45d4d0d108af207b631e0273e657ed79d943ef66732ccd6fd7d290e600"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 4: Fragment-Required Gate Verification Report

**Phase Goal:** A pull request that changes shipped code cannot merge without a changelog fragment or an explicit, reasoned exemption: a `fragment-required` check runs on every pull request, fails closed, and is a required status check on `main`.
**Verified:** 2026-10-13T17:00:00Z
**Status:** passed
**Re-verification:** No, initial verification

## Goal Achievement

### Observable Truths

The five ROADMAP success criteria are the contract. SUMMARY claims were not used as evidence; each row cites something read or run in this session.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A PR touching `cmd/**` or `internal/**` (excluding `*_test.go`, `testdata/`) with no added file under `.changes/unreleased/` fails; the identical shape plus a fragment passes; output recorded; executed count asserted; the path rule lives in one place shared by workflow and tests (GATE-01, GATE-04) | VERIFIED | `scripts/fragment_policy.py:105-115` `is_shipped_code` is the only definition; `scripts/test_fragment_policy.py` imports it (`import fragment_policy`). `test_shape_without_fragment_fails` and `test_identical_shape_with_fragment_passes` use the same `BASE_SHAPE` with and without an ADDED fragment. I ran `task check:fragment-policy`: `Ran 36 tests ... OK`, `executed 36 of 36 defined test methods (floor 36)`. The Taskfile guard (`Taskfile.yml:2258-2298`) asserts `ran == defined`, `defined >= 36` and an exact `OK` line, and is wired into `ci.yml:236-238`. |
| 2 | `<!-- changelog-exempt: <reason> -->` with a non-empty reason passes; an empty reason does not exempt; grammar mirrors `pr-template-exempt` (GATE-02) | VERIFIED | `fragment_policy.py:49` `CHANGELOG_EXEMPT_MARKER` is `pr_template_policy.EXEMPT_MARKER` with only the name changed, and `test_marker_grammar_mirrors_pr_template_exempt` pins that. `test_exempt_marker_with_reason_passes`, `..._is_case_insensitive`, `..._without_reason_does_not_exempt` (three empty forms) and `test_other_marker_names_do_not_exempt` all pass. |
| 3 | An empty changed-file list, an API error and an unreadable body each fail with a distinct message; none passes (GATE-03) | VERIFIED | `evaluate()` returns `empty-file-list`, `api-error`, `body-unreadable` with distinct codes and messages (`fragment_policy.py:150-172`). `test_fail_closed_messages_are_distinct` asserts 3 distinct codes and 3 distinct messages; `test_fail_closed_precedence`, `test_main_fails_closed_when_gh_exits_nonzero`, `..._on_truncated_file_list`, `..._on_malformed_json`, `..._without_pr_body` and `test_main_rejects_malformed_pr_context` cover `main()`. Completeness is proved by comparing `pulls/{n}` `changed_files` to the fetched count (`:242-246`), and the fetch is paginated REST, not the 100-capped `gh pr view --json files`. |
| 4 | The gate is RED against a confirmed-applied, byte-cleanly-reverted mutation of its path rule, recorded in the mutation log; the job body is a `task` invocation covered by `TestWorkflowRunBodiesInvokeTask` (GATE-04) | VERIFIED | `04-MUTATION-LOG.md` exists (families a-e plus RED-first transcripts). I re-ran three mutations independently in a scratch copy (source untouched): drop `internal` from `SHIPPED_ROOTS` gave 14 failures; disable the `_test.go` exclusion gave 4; make `is_fragment` ignore status gave 5. The real suite is 36/36 OK. `fragment-required.yml:65` runs `task check:fragment-required`; `TestWorkflowRunBodiesInvokeTask` and `TestWorkflowFilePopulationMatchesDisk` pass, and `{fragment-required.yml, fragment-required}` is in `inScopeJobs` (`taskfile_shape_test.go:153`) and `inScopeWorkflowFiles` (`:1462`). |
| 5 | `fragment-required` is a required status check in ruleset `protect-main`, edit timestamp recorded, ruleset re-read before and after (GATE-05, amended by D-09) | VERIFIED | Read-only `gh api repos/seanb4t/codegraph-go/rulesets/20157557`: `enforcement: active`, `updated_at: 2026-10-13T12:16:55.257-04:00`, 9 required contexts including `{"context":"fragment-required","integration_id":15368}`, same integration id as the other eight; bypass `RepositoryRole 5` and `strict: false` unchanged. `04-05-ruleset-pre.json` / `-put-body.json` / `-post.json` exist and `04-05-SUMMARY.md` records the timestamp. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Live evidence for "runs on every pull request / is required"

PR #99 (against `main`, files `.github/workflows/release.yml` and `internal/upgrade/release_workflow_shape_test.go`, opened 2026-10-13T16:30:54Z, after the ruleset edit) shows a `fragment-required` check, `status COMPLETED`, `conclusion SUCCESS`, 1m1s, via `gh pr view 99 --json statusCheckRollup`. The shape is correct: it touches no `cmd/**` or `internal/**` non-test file, so the gate passes it without a marker. That is a live observation of the check reporting on a fresh PR after the ruleset edit. At observation time `mergeStateStatus` was `BLOCKED` while `test` was still pending; I cannot attribute the block to a single required context from this read. GitHub enforcing a required context, and the live failing direction on GitHub, are third-party behavior and are cited as observation, not tested (rule `eyva2svrcz`, D-01). The failing and passing verdicts are proved on our own code by the unit suite.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `scripts/fragment_policy.py` | Pure `evaluate()` + `main()` with paginated gh fetch | VERIFIED | 323 lines, substantive; identical to `origin/main` |
| `scripts/test_fragment_policy.py` | stdlib unittest, literal inputs only | VERIFIED | 36 test methods, identical to `origin/main` |
| `.github/workflows/fragment-required.yml` | `pull_request_target`, base checkout, job named `fragment-required`, `task` body | VERIFIED | No `ref:` on checkout; `contents: read` + `pull-requests: read` only; untrusted values only in `env:`; identical to `origin/main` |
| `Taskfile.yml` `check:fragment-required` / `check:fragment-policy` | Production entry and offline guard | VERIFIED | Both present with preconditions; second asserts the executed count |
| `ci.yml` step "Fragment policy unit suite" | Wires the guard into the test job | VERIFIED | `ci.yml:236-238` |
| `taskfile_shape_test.go` | Population registered; ruleset-guard tests removed | VERIFIED | Registered at `:153` and `:1462`; `go test ./internal/upgrade/` passes |
| `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt` | Deleted (D-08) | VERIFIED | Absent on disk and on `origin/main`; `rg` for `check-ruleset-drift`, `required-status-checks`, `TestRequiredCheckNamesPreserved`, `readRequiredCheckNames` outside `.planning/` and `CHANGELOG.md` returns no hits |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `fragment-required.yml` | `scripts/fragment_policy.py` | `task check:fragment-required` runs `python3 scripts/fragment_policy.py` | WIRED | Taskfile `:2242-2256` |
| `ci.yml` test job | `test_fragment_policy.py` | `task check:fragment-policy` | WIRED | `ci.yml:236-238` |
| `fragment-required` job name | ruleset context | job `name: fragment-required` equals ruleset context string | WIRED | Live ruleset lists it; PR #99 reports it |
| `fragment_policy.py` | `pr_template_policy.py` | mirrored `EXEMPT_MARKER`, pinned by test | WIRED | `pr_template_policy.py` untouched (D-10) |

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| `main()` | changed files | `gh api pulls/{n}` then `gh api --paginate --slurp pulls/{n}/files` | Yes, count cross-checked against `changed_files` | FLOWING |
| `main()` | PR body | `PR_BODY` from `github.event.pull_request.body` via `env:` | Yes; absent var is "unreadable", empty string is a readable empty body | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Unit suite and count guard | `task check:fragment-policy` | 36 of 36, `OK` | PASS |
| Shape tests over workflows | `go test ./internal/upgrade/ -count=1` (`GOTOOLCHAIN=go1.26.9`) | `ok` | PASS |
| Path-rule mutation (drop `internal`) | scratch copy, unittest | 14 failures | PASS (RED) |
| Test-exclusion mutation | scratch copy, unittest | 4 failures | PASS (RED) |
| Fragment-status mutation | scratch copy, unittest | 5 failures | PASS (RED) |
| Regression: changie baseline | `task check:changie` | 13 of 13 | PASS |
| Regression: workflow lint | `task lint:actions` | clean | PASS |

### Probe Execution

Step 7c: SKIPPED. The phase declares no `scripts/*/tests/probe-*.sh` probes.

### Requirements Coverage

Requirement IDs declared across the five PLAN frontmatters: GATE-01, GATE-02, GATE-03, GATE-04 (04-01), GATE-04 (04-02), GATE-01 (04-03, 04-04), GATE-05 (04-05). Every ID that `REQUIREMENTS.md` maps to Phase 4 is claimed; no orphans.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| GATE-01 | 04-01, 04-03, 04-04 | Gate fails when shipped code changes and no fragment is added; one shared path rule | SATISFIED | Truth 1; landed on `main` via PR #95 (`65a2b9a9`); the three gate files are byte-identical to `origin/main` |
| GATE-02 | 04-01 | Reasoned exemption marker, grammar mirrors `pr-template-exempt` | SATISFIED | Truth 2 |
| GATE-03 | 04-01 | Fail closed on empty list, API error, unreadable body; distinct messages | SATISFIED | Truth 3 |
| GATE-04 | 04-01, 04-02 | Both directions demonstrated, executed count asserted, RED mutation recorded | SATISFIED | Truths 1 and 4 |
| GATE-05 | 04-05 | `fragment-required` required in `protect-main`, timestamp recorded (amended D-09) | SATISFIED | Truth 5; `REQUIREMENTS.md` carries the amended wording and is `[x]` / Complete |

### CONTEXT Decision Compliance

D-01 (tests exercise our own code only): honored; the suite uses literal inputs and canned runner tuples, and the one subprocess call, `_run_gh`, is never executed by it. D-02 (`pull_request_target`, base checkout): honored. D-03 (no comment, read-only scopes): honored. D-04 (ADDED-only fragments, `.gitkeep` excluded): honored. D-05/D-06 (one landing PR, merge-back): PR #95 and merge `b47a52ce` are in the log. D-08 (guards removed): honored. D-09 (GATE-05 amended): honored. D-10 (`pr_template_policy.py` untouched): honored. The RESEARCH pitfalls (100-file cap, `$GITHUB_OUTPUT` heredoc escaping the delimiter guard) were avoided by design.

### Anti-Patterns Found

None blocking. The `TBD`/`FIXME`/`XXX` scan over the phase's source files and the empty-implementation scan found nothing that flows to a verdict.

### Code review findings (advisory, none blocking)

`04-REVIEW.md` (deep, 0 critical, 3 warning, 6 info; all dispositions `open` in `04-REVIEW-DISPOSITION.md`). I checked the one that could touch the goal:

- **WR-01, cubic regex backtracking in `CHANGELOG_EXEMPT_MARKER`: confirmed.** `find_exemption("<!-- changelog-exempt:" + " " * n)` took 0.027s at n=500, 0.20s at 1000, 1.64s at 2000 (cubic). A large crafted body can run into the 10-minute job timeout. This does not break the goal: a timeout fails the check, so it cannot cause a pass. It is a runner-minutes DoS the PR author inflicts on their own PR, and the same pattern already exists in `pr_template_policy.py:83`. Recommended fix is in `04-REVIEW.md`; GRD-15 (Phase 5) already edits `pr_template_policy.py`.
- **WR-02 / WR-03** are test-coverage gaps in fail-closed branches and in escaping of the exemption reason. I read the code and the escaping is correct today; the gap is that mutations in those branches would survive the suite. Not a goal failure.

### Human Verification Required

None. The remaining unobserved item, the failing direction on a live GitHub PR, is third-party behavior covered by observation of PR #99 plus the unit suite (rule `eyva2svrcz`).

### Gaps Summary

No gaps. Every success criterion is backed by code I read and commands I ran. The gate exists on `main`, is byte-identical on the milestone branch, fails closed, has tests proven RED under independent mutation, and is a required context in the live ruleset. Phase 3's `03-VERIFICATION.md` and Phases 1 and 2 (`passed`, 4/4 each) are unaffected: `task check:changie` is 13/13 and the upgrade package tests pass.

---

_Verified: 2026-10-13T17:00:00Z_
_Verifier: Claude (gsd-verifier)_
