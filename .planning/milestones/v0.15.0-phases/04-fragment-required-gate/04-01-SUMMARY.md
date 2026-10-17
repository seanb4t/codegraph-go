---
phase: 04-fragment-required-gate
plan: 01
subsystem: ci
tags: [github-actions, pull_request_target, python-unittest, taskfile, changie, fail-closed]

requires:
  - phase: 02-changie-capability
    provides: ".changes/unreleased/ fragment directory and the changie task path the gate's guidance names"
provides:
  - "scripts/fragment_policy.py: the one home of the shipped-code path rule, the changelog-exempt grammar, the pure evaluate() verdict and a gh-fetching main()"
  - "scripts/test_fragment_policy.py: 36 literal-input unittest cases (D-01)"
  - ".github/workflows/fragment-required.yml: pull_request_target workflow, job and check context `fragment-required`"
  - "Taskfile targets check:fragment-required (production entry point) and check:fragment-policy (offline suite with the executed-count guard)"
  - "ci.yml test-job step `Fragment policy unit suite (GATE-01..GATE-04)`"
  - "04-MUTATION-LOG.md with both RED and GREEN transcripts and the executed-count run"
affects: [04-02, 04-03, 04-04, 04-05]

actuals:
  tokens: 9300
  tasks: 3
  commits: 8
plan_head_before: ac6f356b0067eaa2cefaf9dad8319782c3ac39ed
plan_head_after: 79fd79c8c2a1810f994829d1b45c1ef99112656d

tech-stack:
  added: []
  patterns:
    - "Python stdlib policy module with a pure verdict function and a runner-injected main(), tested over literal inputs only"
    - "Executed-count guard: a Taskfile target compares unittest's `Ran N` with the defined `def test_` count"

key-files:
  created:
    - scripts/fragment_policy.py
    - scripts/test_fragment_policy.py
    - .github/workflows/fragment-required.yml
    - .planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "The unreadable-body state is exactly PR_BODY absent from the environment; a present empty string is a readable empty body (RESEARCH Open Question 1, locked)"
  - "main() calls gh itself with argument lists (never a shell string) and cross-checks the fetched count against the PR's changed_files, so a truncated listing fails closed"
  - "A rename counts through previous_path, so moving a file out of cmd/** or internal/** still needs a fragment"
  - "Two Taskfile targets rather than a mode flag: check:fragment-required for production, check:fragment-policy for the offline suite"
  - "_run_main in the test module returns (rc, output, calls); a crash inside main comes back as rc None so it reads as an assertion failure, and the recorded calls let a test prove the runner was never reached"

patterns-established:
  - "RED skeleton first: every public name defined with its final signature, every verdict a placeholder, so RED is assertion failures and never an ImportError"
  - "Task-2-only names reached as module attributes behind a hasattr assertion"

requirements-completed: [GATE-01, GATE-02, GATE-03, GATE-04]

coverage:
  - id: D1
    description: "GATE-01 path rule and verdict, both directions: the BASE_SHAPE file list fails with missing-fragment and passes with one added .changes/unreleased/ fragment; the rule has a single home"
    requirement: GATE-01
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_shape_without_fragment_fails"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_identical_shape_with_fragment_passes"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_path_rule_boundaries"
        status: pass
    human_judgment: false
  - id: D2
    description: "GATE-02 changelog-exempt marker: a reasoned marker passes with the reason in a ::notice:: line, an empty or whitespace reason does not, and the grammar equals pr-template-exempt's"
    requirement: GATE-02
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_exempt_marker_with_reason_passes"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_exempt_marker_without_reason_does_not_exempt"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_marker_grammar_mirrors_pr_template_exempt"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_main_reports_exemption_reason_as_notice"
        status: pass
    human_judgment: false
  - id: D3
    description: "GATE-03 fail-closed states: an API error, an empty file list and an unreadable body each fail with a distinct code and message, a truncated or malformed list fails, and malformed PR context never reaches gh"
    requirement: GATE-03
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_fail_closed_messages_are_distinct"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_main_fails_closed_on_truncated_file_list"
        status: pass
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_main_rejects_malformed_pr_context"
        status: pass
    human_judgment: false
  - id: D4
    description: "fragment-required.yml shape: pull_request_target with a base checkout, read-only scopes, a single `task check:fragment-required` run body, registered in both shape-test population fixtures"
    requirement: GATE-01
    verification:
      - kind: integration
        ref: "go test ./internal/upgrade/ -run '^(TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk)$'"
        status: pass
      - kind: other
        ref: "task lint:actions plus the Task 1 workflow-shape shell assertions"
        status: pass
    human_judgment: true
    rationale: "The workflow has never run on a real pull request. It cannot report on any PR until it is on main (D-02), so GATE-01's 'runs on every pull request' becomes live only when 04-04 lands it there."
  - id: D5
    description: "GATE-04 executed-count guard: check:fragment-policy requires ran == defined, at least 36, and an exact OK line, and runs in ci.yml's test job"
    requirement: GATE-04
    verification:
      - kind: other
        ref: "task check:fragment-policy (executed 36 of 36 defined test methods (floor 36))"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-11
status: complete
---

# Phase 4 Plan 01: Fragment-Required Gate Summary

**A `pull_request_target` gate, `fragment-required`, that fails any PR touching `cmd/**` or `internal/**` without a new `.changes/unreleased/` fragment or a reasoned `changelog-exempt` marker, and fails closed whenever it cannot see the whole change; one stdlib-Python module holds the rule and a 36-case suite with an executed-count guard proves it.**

This phase changes CI only. There is no user-visible change to codegraph. The changie capability's verify:post step is expected to skip at this phase's close (ROADMAP Phase 3 Notes: a second observation of CAP-03's skip path).

## Performance

- **Duration:** 8 min
- **Started:** 2026-10-11T23:12:00Z
- **Completed:** 2026-10-11T23:20:00Z
- **Tasks:** 3 (Task 1 tracer, Task 2 RED/GREEN, Task 3)
- **Files modified:** 7 code and workflow files, plus the mutation log

## Accomplishments

- Tracer: one PR's changed files flow from `gh` through the verdict to a failing or passing check, end to end, with the workflow already under `TestWorkflowRunBodiesInvokeTask`. A stub `gh` on PATH proves both directions through `task check:fragment-required`.
- GATE-02: `CHANGELOG_EXEMPT_MARKER` is `pr_template_policy.EXEMPT_MARKER` with only the marker name changed (pattern and flags asserted equal by a test). An empty or whitespace-only reason does not exempt.
- GATE-03: three distinct fail-closed codes (`api-error`, `empty-file-list`, `body-unreadable`) with a fixed precedence. `main()` also fails closed on a truncated listing (fetched count vs the PR's `changed_files`), malformed JSON, a non-zero `gh` result (including the 124 and 127 the default runner maps) and malformed `GITHUB_REPOSITORY` or `PR_NUMBER`, the last without calling `gh` at all.
- GATE-04: `task check:fragment-policy` passes only when unittest's executed count equals the number of `def test_` methods, at least 36, with a final line of exactly `OK`. It now reports `executed 36 of 36 defined test methods (floor 36)` and runs in ci.yml's `test` job right after the no-force-layout guard.
- Every untrusted value that reaches the output passes through `escape_data`, so a fork-authored path containing a newline and `::warning::` cannot start a workflow command.

## Task Commits

1. **Task 1 tracer, RED:** `484fc4b1` (test) - failing suite plus RED skeleton
2. **Task 1 tracer, GREEN:** `1989a4c7` (ci) - policy module, Taskfile target, workflow, shape-test fixtures
   - log: `276d9d3b` (docs)
3. **Task 2, RED:** `7d85f860` (test) - 27 added methods
4. **Task 2, GREEN:** `f1fa5347` (ci) - exemptions, fail-closed states, complete-list cross-check
   - log: `376ee630` (docs)
5. **Task 3:** `484a8e27` (ci) - executed-count guard and ci.yml step
   - log: `79fd79c8` (docs)

**Plan metadata:** the `docs(04-01): complete fragment-required gate plan` commit that carries this file.

## TDD Gate Compliance

RED evidence is the `test(04-01):` commit that landed before its implementation, plus the pasted unittest `FAIL:` transcript. `gsd-tools check tdd-red-evidence` was not used (rule x1cjy9vyhq). The full verbatim transcripts are in `04-MUTATION-LOG.md`.

| Gate | Commit | Order |
|------|--------|-------|
| RED (tracer) | `484fc4b1` test(04-01) | 1 |
| GREEN (tracer) | `1989a4c7` ci(04-01) | 2 |
| RED (GATE-02/GATE-03) | `7d85f860` test(04-01) | 3 |
| GREEN (GATE-02/GATE-03) | `f1fa5347` ci(04-01) | 4 |

The first four commits touching the two policy files are test, ci, test, ci. GREEN commits use the `ci(04-01):` type, not `feat(04-01):`, because D-05 restricts the landing PR to `ci:`, `test:` and `chore:` so release-please opens no release. A scope-only `git log --grep` for `feat(04-01)` also matches unrelated commits from an earlier milestone's Phase 4, not this plan.

### Tracer RED (before `1989a4c7`): `FAILED (failures=10)`, 9 tests, zero `ERROR:` lines

```
FAIL: test_identical_shape_with_fragment_passes (...EvaluateTests...)
FAIL: test_shape_without_fragment_fails (...EvaluateTests...)
FAIL: test_test_files_and_testdata_are_not_shipped_code (...EvaluateTests...)
FAIL: test_escape_data_encodes_command_characters (...MainTests...)
FAIL: test_main_fails_and_names_paths_when_fragment_missing (...MainTests...)
FAIL: test_main_fails_closed_when_gh_exits_nonzero (...) (returncode=1)
FAIL: test_main_fails_closed_when_gh_exits_nonzero (...) (returncode=124)
FAIL: test_main_fails_closed_when_gh_exits_nonzero (...) (returncode=127)
FAIL: test_main_passes_when_fragment_added (...MainTests...)
FAIL: test_untrusted_paths_cannot_start_workflow_commands (...MainTests...)
Ran 9 tests in 0.001s
FAILED (failures=10)
```

Representative assertion: `test_shape_without_fragment_fails` failed with `AssertionError: True is not false` on `verdict.ok`, against the skeleton's placeholder pass.

### GATE-02/GATE-03 RED (before `f1fa5347`): `FAILED (failures=19)`, 36 tests, zero `ERROR:` lines

```
FAIL: test_empty_file_list_fails_closed
FAIL: test_exempt_marker_is_case_insensitive
FAIL: test_exempt_marker_with_reason_passes
FAIL: test_fail_closed_messages_are_distinct
FAIL: test_fail_closed_precedence
FAIL: test_marker_grammar_mirrors_pr_template_exempt
FAIL: test_rename_out_of_shipped_code_counts
FAIL: test_unreadable_body_fails_closed
FAIL: test_main_fails_closed_on_malformed_json
FAIL: test_main_fails_closed_on_truncated_file_list
FAIL: test_main_fails_closed_without_pr_body
FAIL: test_main_rejects_malformed_pr_context (override={'PR_NUMBER': ''})
FAIL: test_main_rejects_malformed_pr_context (override={'PR_NUMBER': '0'})
FAIL: test_main_rejects_malformed_pr_context (override={'PR_NUMBER': '07'})
FAIL: test_main_rejects_malformed_pr_context (override={'PR_NUMBER': '7; rm -rf /'})
FAIL: test_main_rejects_malformed_pr_context (override={'GITHUB_REPOSITORY': ''})
FAIL: test_main_rejects_malformed_pr_context (override={'GITHUB_REPOSITORY': 'noslash'})
FAIL: test_main_reports_exemption_reason_as_notice
FAIL: test_main_uses_paginated_rest_listing
Ran 36 tests in 0.003s
FAILED (failures=19)
```

Fourteen distinct methods fail, including all 13 the plan names. `test_main_fails_closed_on_malformed_json` fails because the tracer's `main()` raised on malformed JSON, which the test helper turns into an assertion failure. GREEN: `Ran 36 tests ... OK`.

## Code commits for 04-03 to cherry-pick

All five touch no `.planning/` path and use a `ci(04-01):` or `test(04-01):` subject, in this order:

| SHA | Subject |
|-----|---------|
| `484fc4b1` | test(04-01): expect shipped-code changes without a fragment to fail the gate (GATE-01, GATE-04) |
| `1989a4c7` | ci(04-01): add the fragment-required gate on pull_request_target (GATE-01) |
| `7d85f860` | test(04-01): expect exemptions, fail-closed states and a complete file list (GATE-02, GATE-03) |
| `f1fa5347` | ci(04-01): honor changelog-exempt and fail closed on unseen changes (GATE-02, GATE-03) |
| `484a8e27` | ci(04-01): run the fragment-policy suite in the test job with an executed-count guard (GATE-04) |

The Taskfile targets sit between `check:workflow-output-delimiter` and `check:graph-console`, and the ci.yml step sits right after `No-force-layout guard (GRD-11/GRF-06)`, so the D-05/D-06 anchors are regions that exist unchanged on origin/main.

## Stub-gh end-to-end transcripts

The stub `gh` on PATH is our test double. Nothing called GitHub (D-01).

No fragment (`task check:fragment-required` exits non-zero):

```
::error::fragment-required: FAIL (missing-fragment): this PR changes shipped code (cmd/** or internal/**, excluding *_test.go and testdata/) but adds no changelog fragment under .changes/unreleased/
::error::fragment-required: shipped-code paths: internal/cli/root.go
::error::fragment-required: add a fragment: CI=true task changie -- new -k <Kind> -b "<one sentence>" -m PR=<number>, then commit the new .changes/unreleased/*.yaml file (kinds are declared in .changie.yaml)
::error::fragment-required: or, if this change needs no changelog entry, put <!-- changelog-exempt: <reason> --> in the PR body with a non-empty reason
fragment-required: shipped-code path: internal/cli/root.go
task: Failed to run task "check:fragment-required": exit status 1
```

With an added `.changes/unreleased/Fixes-1.yaml` (exit 0):

```
fragment-required: PASS (has-fragment): shipped code changed and a new .changes/unreleased/ fragment was added
```

## 04-MUTATION-LOG.md sections

All in `.planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md`:

- `## RED-first transcript (04-01 Task 1)`
- `## GREEN after the tracer (04-01 Task 1)` (suite plus both stub-gh directions)
- `## RED transcript (04-01 Task 2)`
- `## GREEN after GATE-02/GATE-03 (04-01 Task 2)`
- `## Executed-count guard (04-01 Task 3)` (`check:fragment-policy: executed 36 of 36 defined test methods (floor 36)`)

04-02 Family (d) demonstrates the executed-count guard RED.

## Decisions Made

See `key-decisions` in the frontmatter. Each follows the plan. The one choice the plan left open: the test helper `_run_main` returns a three-tuple so Task 2's tests can assert that malformed PR context never reached the runner.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan defect] Taskfile-order acceptance one-liner extracts the wrong field**
- **Found during:** Task 1 acceptance run
- **Issue:** The criterion pipes `rg -n '^  check:(...):$' Taskfile.yml | cut -d: -f3`. With a single file `rg -n` prints `LINE:  check:name:`, so field 3 is `name` without the `check:` prefix, and the string comparison can never be equal. The ordering itself is correct.
- **Fix:** No code change. Verified the same property with `cut -d: -f2- | tr -d ' '`, which printed `check:workflow-output-delimiter: check:fragment-required: check:fragment-policy: check:graph-console:` after Task 3, and with the `rg -n` line numbers (2220, 2242, 2258 after Task 1). Task 3's equivalent criterion has the same defect and was checked the same way.
- **Files modified:** none
- **Committed in:** n/a

**2. [Rule 3 - Blocking] Stale per-plan commit ledger from an earlier milestone**
- **Found during:** commit protocol 0c, before the first commit
- **Issue:** `.git/gsd-plan-head-before-04-01` already existed, written on 2026-09-17 by an unrelated plan 04-01 from an earlier milestone's Phase 4, so `[ -f ] ||` kept a base of `86f3052b` and would have measured the wrong commit count.
- **Fix:** Overwrote it with the current HEAD `ac6f356b`, which is the base recorded in `plan_head_before`. The file is local git metadata, not tracked work.
- **Files modified:** none tracked
- **Committed in:** n/a

---

**Total deviations:** 2 auto-fixed (1 plan-criterion defect, 1 blocking environment state)
**Impact on plan:** None on scope. The plan's behavior and every artifact are as specified.

## Issues Encountered

- `04-MUTATION-LOG.md` carries a literal carriage return inside the verbatim RED transcript (the `escape_data("50%\r\nx")` assertion diff prints it). Git warns "CRLF will be replaced by LF the next time Git touches it". The transcript is otherwise unchanged and the log's `rg` assertions all pass.
- `.planning/config.json` was already modified in the working tree when execution started. It was left untouched and is in no commit of this plan.

## Authentication Gates

None.

## Known Stubs

None. The RED skeleton's placeholder verdicts existed only between the RED and GREEN commits and were replaced by the implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 04-02 can mutate this plan's rule, grammar, precedence and executed-count guard (Families a-e) and remove the two ruleset guards (D-08). The shape-test fixtures it will edit are the same two arrays this plan appended to.
- GATE-01's "runs on every pull request" becomes live only when 04-04 lands the workflow on `main`. Until then the workflow has run only against a stub `gh`, never against GitHub. Assumptions A-1 (python3 on the Namespace test-job image) and A-2 (the runner's `gh` supports `--slurp`) are unverified until the landing PR's checks run. Both fail loudly rather than pass.

## Self-Check: PASSED

- Created files present: `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py`, `.github/workflows/fragment-required.yml`, `04-MUTATION-LOG.md`.
- Commits present on this branch: `484fc4b1`, `1989a4c7`, `276d9d3b`, `7d85f860`, `f1fa5347`, `376ee630`, `484a8e27`, `79fd79c8`. `gsd-tools check evaluation-scope --plan 04-01 --commits-only` resolved them.
- Task acceptance criteria re-run: suite 36 of 36, `task check:fragment-policy` 36 of 36, stub-gh both directions, three population and run-body Go tests, `go test ./internal/upgrade/`, `task lint:actions`, `gofmt -l internal/upgrade` empty, workflow shape assertions, import set, single rule home, and the commit-composition criteria all pass. `pr_template_policy.py` and `CONTRIBUTING.md` are unchanged from `cbbfeafb`. No code commit carries a `.planning/` path.

---
*Phase: 04-fragment-required-gate*
*Completed: 2026-10-11*
