---
phase: "4"
slug: "fragment-required-gate"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-26"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | python stdlib `unittest` (`scripts/test_fragment_policy.py`, 36 literal-input cases) + go test (`internal/upgrade` workflow/Taskfile shape guards) + Taskfile gates (`check:fragment-policy`, `check:fragment-required`, `lint:actions`) + read-only `git`/`jq`/`gh api` GET checks |
| **Config file** | none — `Taskfile.yml` (`check:fragment-required`, `check:fragment-policy`) and `go.mod` |
| **Quick run command** | `task check:fragment-policy` (prints `check:fragment-policy: executed 36 of 36 defined test methods (floor 36)`) |
| **Full suite command** | `GOTOOLCHAIN=go1.26.9 task test:unit && task check:fragment-policy && task check:changie && task check:workflow-output-delimiter && task lint:actions` |
| **Estimated runtime** | quick ~0.3 seconds; full ~15 seconds (observed 2026-10-13 on a warm cache) |

`GOTOOLCHAIN=go1.26.9` is mandatory on every `go`/`task` invocation on this machine (ambient Go 1.27.1 breaks the build). The plans pin `go1.26.6`, which `go.mod` (>= 1.26.9 since PR #96) now rejects; the commands below carry the current pin.

---

## Sampling Rate

- **After every task commit:** Run `task check:fragment-policy`
- **After every plan wave:** Run `GOTOOLCHAIN=go1.26.9 task test:unit && task check:fragment-policy && task check:changie && task check:workflow-output-delimiter && task lint:actions`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

Re-run on this checkout 2026-10-13 (branch `gsd/v0.15.0-milestone`, `GOTOOLCHAIN=go1.26.9`, read-only commands only). One-shot events (a PR being opened or merged, a ruleset write) are not re-runnable tests; they are marked "recorded evidence" with the read-only re-check that still holds. Plan commands that pin `go1.26.6` were run with `go1.26.9`.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 4-01-1 | 01 | 1 | GATE-01, GATE-04 | T-04-01, T-04-05 | Tracer, both directions end to end: the shape without a fragment fails with `missing-fragment` naming the shipped paths, the identical shape plus one added fragment passes with `has-fragment`. The workflow checks out no `ref:`, holds only `contents: read` and `pull-requests: read`, sets `cache: false`, and its sole run body is `task check:fragment-required` | e2e (fake `gh` on PATH) + unit + go shape guards + workflow-shape shell assertions | (1) fake-gh run of `task check:fragment-required` for PR 7 with and without `.changes/unreleased/Fixes-1.yaml` (plan L348); (2) `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` (>= 9 ok); (3) `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestWorkflowRunBodiesInvokeTask\|TestWorkflowFilePopulationMatchesDisk\|TestInScopeJobsPopulationMatchesDisk)$' -count=1 -v` (3 PASS); (4) `task lint:actions` plus the exact-`uses:`/permissions/`ref:`/`cache:` assertions on `.github/workflows/fragment-required.yml` (plan L354); (5) `git log --first-parent` order check, `test(04-01)` then `ci(04-01)` (RED before GREEN) | ✅ `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py`, `.github/workflows/fragment-required.yml`, `Taskfile.yml`, `internal/upgrade/taskfile_shape_test.go` | ✅ green (all five exit 0) |
| 4-01-2 | 01 | 1 | GATE-02, GATE-03, GATE-04 | T-04-02, T-04-03, T-04-04 | A reasoned `changelog-exempt` marker passes (reason echoed as `::notice::`), an empty reason does not exempt; an empty file list, an API error, an unreadable body, a truncated or malformed listing and malformed PR context each fail closed with a distinct message; untrusted paths cannot start workflow commands; `gh` is called with argument lists | unit (36 literal-input cases) + RED transcript check | (1) `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` with exact `OK` and >= 36 `ok` lines (plan L452); (2) RED transcript in `04-MUTATION-LOG.md`: `## RED transcript (04-01 Task 2)` and `FAIL: <name> (` for the six named tests (plan L454); (3) commit order `test(04-01)`/`ci(04-01)` x2 and `git diff --quiet cbbfeafb HEAD -- scripts/pr_template_policy.py CONTRIBUTING.md` (plan L456) | ✅ `scripts/test_fragment_policy.py` (`test_exempt_marker_*`, `test_marker_grammar_mirrors_pr_template_exempt`, `test_empty_file_list_fails_closed`, `test_api_error_fails_closed`, `test_unreadable_body_fails_closed`, `test_fail_closed_messages_are_distinct`, `test_main_fails_closed_on_*`, `test_escape_data_encodes_command_characters`) | ✅ green (all three exit 0) |
| 4-01-3 | 01 | 1 | GATE-04 | T-04-07 | CI fails when fewer tests ran than are defined, when fewer than 36 are defined, or when the result line is not exactly `OK` (no vacuous pass) | task (executed-count guard) + CI-step shape check + go shape guards | (1) `task check:fragment-policy` → `OK` and `executed 36 of 36 defined test methods (floor 36)` (plan L508); (2) ci.yml step `Fragment policy unit suite (GATE-01..GATE-04)` follows `No-force-layout guard (GRD-11/GRF-06)` and runs `task check:fragment-policy` (plan L510); (3) `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 && task lint:actions` | ✅ `Taskfile.yml` (`check:fragment-policy`), `.github/workflows/ci.yml` | ✅ green (all three exit 0) |
| 4-02-1 | 02 | 2 | GATE-04 | T-04-10 | The ruleset-drift guard and its required-check fixture are retired in one code-only commit; no reference to the retired names remains; the census control file is never committed (`.github` clean before and after) | go test + build/vet/gofmt + positive-controlled census | (1) `GOTOOLCHAIN=go1.26.9 go build ./... && go vet ./internal/upgrade/ && test -z "$(gofmt -l internal/upgrade)" && go test ./internal/upgrade/ -count=1 && task lint:actions` (plan L159); (2) `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestGateStancesStated\|TestTaskfileShapeHelpersFailLoudly\|TestWorkflowRunBodiesInvokeTask)$' -count=1 -v` (3 PASS); (3) census `rg -n --hidden -e 'required-status-checks' -e 'check-ruleset-drift' -e 'equiredCheckNames' -e 'equiredStatusChecksPath' -e 'Ruleset drift check' --glob '!.planning/**' --glob '!.git/**' .` → exit 1, zero hits. The planted-control half (5 control hits) is a one-shot instrument that writes a file; it was not re-run, and its result is recorded in 04-02-SUMMARY | ✅ `internal/upgrade/taskfile_shape_test.go` | ✅ green (build/vet/test and the 3 PASS exit 0; census clean, exit 1; control half: recorded evidence) |
| 4-02-2 | 02 | 2 | GATE-04 | T-04-09 | Every guard the gate relies on goes RED against a confirmed-applied mutation of our own code, reverted byte-clean in a disposable worktree; no mutation leaks into history or the checkout | recorded mutation transcripts (10 mutations, Families a-e) + read-only cleanliness check | (1) `04-MUTATION-LOG.md` assertions: five `## Family (x)` headings, >= 9 `Byte-clean proof`, the named `FAIL:` lines, `unittest ran N of 36 defined test methods`, `result line is not exactly OK`, `--- FAIL: TestWorkflowRunBodiesInvokeTask` (plan L230); (2) no `p04-scratch-wt` worktree and `git status --porcelain -- scripts .github internal Taskfile.yml` empty (plan L232) | ✅ `.planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md` | ✅ green (both exit 0; the mutations themselves are recorded evidence and were not re-applied) |
| 4-02-3 | 02 | 2 | GATE-04 (amends GATE-05 text, D-09) | — | GATE-05 and the ROADMAP Phase 4 text are amended with no new heading; the milestone regression gates stay green | exact-text asserts + heading-count equality + full regression suite | (1) GATE-05 and ROADMAP exact-text asserts (plan L279); (2) heading counts of `REQUIREMENTS.md`/`ROADMAP.md` equal those at `cbbfeafb` (plan L281); (3) `GOTOOLCHAIN=go1.26.9 task test:unit && task check:fragment-policy && task check:changie && task check:workflow-output-delimiter && task lint:actions` | ✅ `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `Taskfile.yml` | ✅ green (all three exit 0) |
| 4-03-1 | 03 | 3 | GATE-01 | T-04-11, T-04-12 | The landing branch carries exactly the nine Phase 4 paths, line-equal to the milestone's Phase 4 diff, in `ci`/`test`/`chore`-typed `(04-01\|04-02)` commits with `cherry picked from commit` provenance (no release-please trigger) | one-shot landing proof (git diff set-equality, subject-type regex) | (1) path-set equality `git diff --name-only $(git merge-base origin/main ci/fragment-required-gate) ci/fragment-required-gate` = the nine paths (plan L158); (2) per-path +/- line equality against the milestone diff since `cbbfeafb` (plan L160); (3) commit-subject type and provenance check (plan L162); (4) `go test` + `task check:fragment-policy` + `task lint:actions` in a detached worktree of the branch (plan L164) | ✅ local branch `ci/fragment-required-gate` (`d2baf7c6`) | ✅ green, recorded evidence (re-ran 1 and 3: exit 0; 2 holds for 8 of 9 paths, `Taskfile.yml` differs only by the later `go1.26.6` → `go1.26.9` bump of PR #96 (`0555b5ab`), judged by intent; 4 creates a worktree and was not re-run, 36 of 36 and `go test ./internal/upgrade/` are re-proved by 4-01-3). 04-03-SUMMARY records the original pass |
| 4-03-2 | 03 | 3 | GATE-01 | T-04-13 | Publishing needs the maintainer's recorded decision (open a new issue, then the PR with `Closes #94`) | checkpoint:decision (blocking-human) | n/a — no automated verify; the choice is recorded in 04-03-SUMMARY (maintainer chose open-new-issue) | N/A (human decision) | ✅ green, recorded evidence (decision in 04-03-SUMMARY) |
| 4-03-3 | 03 | 3 | GATE-01 | T-04-13, T-04-14 | Exactly what was approved is published: branch pushed, one non-draft PR to `main` with a conventional title and exactly the nine paths; body passed with `--body-file` | one-shot publish proof (`git ls-remote`, `gh pr list`, `gh pr view`) | (1) `git ls-remote --heads origin ci/fragment-required-gate` = branch tip (plan L269); (2) `gh pr list --head ci/fragment-required-gate --state open` shows one non-draft PR on `main` with a conventional title (plan L271); (3) `gh pr view <n> --json files` = the nine paths (plan L273). Post-merge re-check: `gh pr view 95 --json state,files,isDraft,title` → `MERGED` 2026-10-12T14:18:12Z (`65a2b9a9`), base `main`, `isDraft` false, files = the nine paths | ✅ PR #95, issue #94, `04-03-PR-BODY.md` | ✅ green, recorded evidence (the open-state forms of 1 and 2 are stale by design: the PR is merged and the head branch is deleted, `git ls-remote` returns nothing; 3 still holds against the merged PR) |
| 4-04-1 | 04 | 4 | GATE-01 | T-04-15 | Only the maintainer merges the landing PR; the agent never runs `gh pr merge` | checkpoint:human-action (blocking-human) | n/a — verify block is the Task 2 precondition; read-only re-check `gh pr view 95 --json state,mergedAt,mergeCommit` → `MERGED`, `65a2b9a9`; `git cat-file -e origin/main:.github/workflows/fragment-required.yml` exits 0 | N/A (human action) | ✅ green, recorded evidence (04-04-SUMMARY; re-check holds) |
| 4-04-2 | 04 | 4 | GATE-01 | T-04-16, T-04-17 | Main is merged back with one `--no-ff` merge commit (no rebase/reset), whose change on the nine paths equals the landing branch's, and the regression gates stay green | git topology assertions + full regression suite | (1) one `chore(04-04): merge main into the milestone` merge, 2 parents, `X^2` an ancestor of `origin/main`, `origin/main` an ancestor of `HEAD` (plan L144); (2) the merge's diff on the nine paths = `git diff ci/fragment-required-gate X^2` (plan L146); (3) `GOTOOLCHAIN=go1.26.9 task test:unit && task check:fragment-policy && task check:changie && task lint:actions` | ✅ merge `b47a52ce` on `gsd/v0.15.0-milestone` | ✅ green (all three exit 0 on this checkout) |
| 4-05-1 | 05 | 5 | GATE-05 | T-04-18 | The PUT body equals the pre-read plus exactly one context (`fragment-required`), with bypass actors, conditions, enforcement, rule types, strict policy unchanged and no read-only keys | jq assertions over committed ruleset reads | (1) `jq -e --slurpfile pre 04-05-ruleset-pre.json '…' 04-05-ruleset-put-body.json` (plan L139); (2) `git show origin/main:.github/workflows/fragment-required.yml` has `^  fragment-required:$` and `^    name: fragment-required$` (plan L141) | ✅ `04-05-ruleset-pre.json`, `04-05-ruleset-put-body.json` | ✅ green (both exit 0) |
| 4-05-2 | 05 | 5 | GATE-05 | T-04-19 | The ruleset is written only on the maintainer's explicit `run-prepared` reply, once, with the prepared command | checkpoint:human-action (blocking-human) | n/a — no automated verify; the reply and the single PUT are recorded in 04-05-SUMMARY | N/A (human action) | ✅ green, recorded evidence (04-05-SUMMARY) |
| 4-05-3 | 05 | 5 | GATE-05 | T-04-18, T-04-20 | `protect-main` (20157557) now requires `fragment-required` exactly once (8 to 9 contexts), nothing else changed, and the edit's timestamp is recorded | jq assertion over the committed post-read + read-only live `gh api` GET | (1) `jq -e --slurpfile pre 04-05-ruleset-pre.json '…' 04-05-ruleset-post.json` (plan L208); (2) `gh api repos/seanb4t/codegraph-go/rulesets/20157557 --jq '.rules[] \| select(.type=="required_status_checks") \| .parameters.required_status_checks[].context'` lists `fragment-required` exactly once (plan L210), live GET today shows 9 contexts including it | ✅ `04-05-ruleset-pre.json`, `04-05-ruleset-post.json` | ✅ green (both exit 0; edit timestamp `2026-10-13T12:16:55.257-04:00` recorded in 04-05-SUMMARY) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The `fragment-required` workflow actually runs on a live pull request to `main` under `pull_request_target` and reports the `fragment-required` context | GATE-01 | GitHub runs the workflow, not this repo (rule `eyva2svrcz`); `pull_request_target` reads the workflow from `main`, so it has been observable only since PR #95 merged (2026-10-12) and no PR has exercised it yet. The policy it runs is covered by 4-01-1..3 | On the next PR to `main`, read `gh pr checks <n>` and confirm a `fragment-required` check; expect it to fail on a PR touching `cmd/**` or `internal/**` without a `.changes/unreleased/` fragment or a reasoned `changelog-exempt` marker, and pass otherwise |
| GitHub blocks merging a PR to `main` until `fragment-required` has passed | GATE-05 | Enforcement of a required status check is GitHub's behavior (rule `eyva2svrcz`). This repo can only record the ruleset state (4-05-3: `protect-main` lists `fragment-required` once, edit `2026-10-13T12:16:55.257-04:00`) | After the next PR event, confirm the PR shows `fragment-required` as a required check and cannot merge while it is red or absent. Open PRs (including #88) may show it as "Expected" until their next event |
| Maintainer-only acts: publishing PR #95, merging it, and the single ruleset PUT | GATE-01, GATE-05 | Outward-facing, one-shot, blocking-human checkpoints (4-03-2, 4-04-1, 4-05-2); not repeatable tests | Recorded in 04-03-SUMMARY, 04-04-SUMMARY, 04-05-SUMMARY; read-only re-checks are in 4-03-3, 4-04-1 and 4-05-3 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-10-13

## Validation Audit 2026-10-13

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
