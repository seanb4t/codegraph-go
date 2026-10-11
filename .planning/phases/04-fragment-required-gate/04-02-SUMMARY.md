---
phase: 04-fragment-required-gate
plan: 02
subsystem: ci
tags: [github-actions, python-unittest, mutation-testing, ruleset, taskfile]

requires:
  - phase: 04-fragment-required-gate
    provides: "plan 04-01's scripts/fragment_policy.py, its 36-case suite, the check:fragment-policy executed-count guard and the fragment-required workflow"
provides:
  - "D-08: the ruleset-drift guard and its required-check fixture retired in one code-only ci(04-02) commit"
  - "04-MUTATION-LOG.md Families (a)-(e): ten mutations of our own code, each confirmed applied, shown RED and reverted byte-clean"
  - "D-09: GATE-05 and the Phase 4 ROADMAP text amended with no new structure"
affects: [04-03, 04-04, 04-05]

actuals:
  tokens: 26494
  tasks: 3
  commits: 3
plan_head_before: f85cba7a96c8331f4b59c6f4810aeabb3a7fbc7a
plan_head_after: 368403804e23ce04ddbcabc47948c74b6b4fa475

tech-stack:
  added: []
  patterns:
    - "Disposable detached worktree for mutations: pre-gate, apply, numstat plus -U0 hunk, run the named guard, checkout revert, diff --quiet"
    - "Positive-controlled census: plant the five tokens in an untracked file under a trap, require 5 hits, then require zero hits and exit 1 on the clean tree"

key-files:
  created: []
  modified:
    - internal/upgrade/taskfile_shape_test.go
    - .github/workflows/ci.yml
    - .github/workflows/components-drift.yml
    - Taskfile.yml
    - .planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md

key-decisions:
  - "D-08 ships as one code-only ci(04-02) commit touching exactly six paths, so plan 04-03 can cherry-pick it onto the landing PR"
  - "parseWorkflowJobNames and mustWorkflowJobNames stay: TestGateStancesStated and TestTaskfileShapeHelpersFailLoudly still use them"
  - "The plan's prose says nine mutations but its family list holds ten (two per family); all ten were run and recorded"

patterns-established:
  - "Mutation targets are our own code only (path rule, fragment rule, exemption rule, fail-closed branches, count guard, single-definition registration), never git, GitHub or changie"

requirements-completed: [GATE-04]

coverage:
  - id: D1
    description: "D-08: required-check fixture, loader, six tests, drift script, its CI step and its run-body exception are gone; shared helpers and the shape suite stay green"
    requirement: GATE-04
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -count=1"
        status: pass
      - kind: other
        ref: "census with positive control: 5 control hits, then zero hits and exit 1 on the clean tree"
        status: pass
    human_judgment: false
  - id: D2
    description: "GATE-04 RED: ten mutations of the path rule, fragment rule, exemption rule, fail-closed branches, executed-count guard and single-definition registration each turned a named test or guard RED and were reverted byte-clean"
    requirement: GATE-04
    verification:
      - kind: other
        ref: ".planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md Families (a)-(e)"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-09: GATE-05 line, ROADMAP ordering bullet, success criterion 5 and Notes carry the amended text; heading counts unchanged"
    requirement: GATE-04
    verification:
      - kind: other
        ref: "plan Task 3 automated checks (exact-text assertions plus heading-count equality against cbbfeafb)"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-11
status: complete
---

# Phase 4 Plan 02: Retire the Ruleset Guards and Prove the Gate's Guards Bite Summary

**The ruleset-drift guard and its required-check fixture are gone in one cherry-pickable code-only commit, and ten mutations of the fragment gate's own rules, count guard and single-definition registration each went RED against a confirmed-applied, byte-cleanly-reverted mutation.**

This phase changes CI only. There is no user-visible change to codegraph.

## Performance

- **Duration:** 8 min
- **Started:** 2026-10-11T23:24:42Z
- **Completed:** 2026-10-11T23:32:17Z
- **Tasks:** 3
- **Files modified:** 9 (6 code and workflow paths in the D-08 commit, plus the mutation log, REQUIREMENTS.md and ROADMAP.md)

## Accomplishments

- D-08: `requiredStatusChecksPath`, `readRequiredCheckNames`, the two `TestRequiredCheckNamesPreserved` tests, the four `TestReadRequiredCheckNames_` tests, the GRD-12 `runBodyExceptions` entry and its doc paragraph are deleted. `scripts/check-ruleset-drift.sh` and `.github/required-status-checks.txt` are removed with `git rm`, and ci.yml's `Ruleset drift check (GRD-12)` step and comment block are gone. `runBodyExceptions` holds one entry. The prose that named the fixture in ci.yml (IN-04), Taskfile.yml (`web:components:drift` desc) and components-drift.yml (header) is rewritten.
- GATE-04: ten mutations in a detached scratch worktree, families (a)-(e), all RED. The two count-guard runs show unittest itself printing `OK` and `OK (skipped=1)` while `task check:fragment-policy` exits non-zero, which proves the guard rather than unittest caught them.
- D-09: GATE-05 now reads that `fragment-required` is a required status check in ruleset `protect-main`, with the amendment note. The ROADMAP ordering bullet, success criterion 5 and Notes carry the amended text. Heading counts are unchanged (REQUIREMENTS 14, ROADMAP 16).

## Task Commits

1. **Task 1: retire both ruleset guards (D-08)** - `565e3e25` (ci)
2. **Task 2: record the mutation families (a)-(e)** - `dcbc5e31` (docs)
3. **Task 3: amend GATE-05 and the Phase 4 roadmap text (D-09)** - `36840380` (docs)

**Plan metadata:** the `docs(04-02): complete ...` commits that carry this file and the state updates.

### D-08 commit: `565e3e25`

Subject: `ci(04-02): retire the ruleset-drift guard and its required-check fixture (D-08)`. Exactly six paths, none under `.planning/`:

- `.github/required-status-checks.txt` (deleted)
- `.github/workflows/ci.yml`
- `.github/workflows/components-drift.yml`
- `Taskfile.yml`
- `internal/upgrade/taskfile_shape_test.go`
- `scripts/check-ruleset-drift.sh` (deleted)

Plan 04-03 cherry-picks it. The post-commit deletion check listed only the two intended deletions.

### Census transcript (positive control first, then the clean tree)

```
--- control transcript (untracked .github/__census_control__.md, created under a trap)
./.github/__census_control__.md:1:required-status-checks
./.github/__census_control__.md:2:check-ruleset-drift
./.github/__census_control__.md:3:requiredCheckNames
./.github/__census_control__.md:4:requiredStatusChecksPath
./.github/__census_control__.md:5:Ruleset drift check
control lines: 5
ls: .github/__census_control__.md: No such file or directory
--- clean census
exit=1
```

The instrument is `rg -n --hidden -e required-status-checks -e check-ruleset-drift -e equiredCheckNames -e equiredStatusChecksPath -e "Ruleset drift check" --glob '!.planning/**' --glob '!.git/**' .`. It found all 5 planted lines, the control file was gone afterwards, and the clean tree printed nothing and exited 1.

### Mutation families (`04-MUTATION-LOG.md`)

| Family | Mutations | Went RED in |
|--------|-----------|-------------|
| (a) path rule | a1 `SHIPPED_ROOTS` loses `"internal"`; a2 testdata clause removed | `test_path_rule_boundaries`, `test_shape_without_fragment_fails`, `test_rename_out_of_shipped_code_counts`, `test_test_files_and_testdata_are_not_shipped_code` |
| (b) fragment and exemption rules | b1 `is_fragment` ignores status; b2 `find_exemption` accepts an empty reason | `test_fragment_must_be_added`, `test_exempt_marker_without_reason_does_not_exempt` |
| (c) fail closed | c1 empty-file-list branch deleted; c2 `changed_files` cross-check deleted | `test_empty_file_list_fails_closed`, `test_main_fails_closed_on_truncated_file_list` |
| (d) executed-count guard | d1 `MainTests` loses `unittest.TestCase`; d2 one test gains `@unittest.skip` | `task check:fragment-policy`: `unittest ran 24 of 36 defined test methods`; `result line is not exactly OK` |
| (e) single-definition coverage | e1 run body becomes a direct `python3` call; e2 `inScopeJobs` entry deleted | `TestWorkflowRunBodiesInvokeTask` naming `fragment-required.yml job "fragment-required"`; `TestInScopeJobsPopulationMatchesDisk` naming `fragment-required.yml/fragment-required` |

Each entry records the pre-mutation gate, `diff --numstat` plus the `-U0` hunk, the verbatim output, the revert and `git diff --quiet` (exit 0). The scratch worktree was removed and `git status --porcelain -- scripts .github internal Taskfile.yml` is empty in the main checkout.

### Regression gates (milestone branch, after all three tasks)

| Command | Result |
|---------|--------|
| `GOTOOLCHAIN=go1.26.6 task test:unit` | exit 0, 53 `ok` package lines, no `FAIL` |
| `task check:fragment-policy` | `check:fragment-policy: executed 36 of 36 defined test methods (floor 36)` |
| `task check:changie` | `check:changie: 13 of 13 checks passed against a scratch copy (source tree byte-unchanged)` |
| `task check:workflow-output-delimiter` | `check-workflow-output-delimiter: PASS — both workflows round-trip the attack payload intact` |
| `task lint:actions` | exit 0, no actionlint finding |
| `GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -count=1` | `ok` |

The `test:unit` run printed many `(cached)` lines because the packages this plan did not touch were unchanged since 04-01.

## Files Created/Modified

- `internal/upgrade/taskfile_shape_test.go` - the required-check family removed (204 lines), one `runBodyExceptions` entry left, WR-03 sentence rewritten
- `.github/workflows/ci.yml` - GRD-12 comment and step removed, IN-04 comment no longer names the fixture
- `.github/workflows/components-drift.yml` - header sentence rewritten
- `Taskfile.yml` - `web:components:drift` desc rewritten
- `.planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md` - Families (a)-(e), teardown, verdict
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` - D-09 value edits, four lines in total

## Decisions Made

See `key-decisions` in the frontmatter. All follow the plan.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan defect] "Nine mutations" versus ten in the family list**
- **Found during:** Task 2
- **Issue:** The plan's must_haves, objective prose and done text say nine mutations, but its family list (a1, a2, b1, b2, c1, c2, d1, d2, e1, e2) holds ten. The verify command's floor is "at least 9", which ten satisfies.
- **Fix:** All ten were run and recorded; the log's verdict paragraph says ten and notes the discrepancy. Nothing was dropped to match the count.
- **Files modified:** `04-MUTATION-LOG.md`
- **Committed in:** `dcbc5e31`

**2. [Rule 3 - Blocking] A hook rejected an apostrophe in a heredoc commit message**
- **Found during:** Task 2 commit
- **Issue:** The `git-merge-guard` PreToolUse hook denied a heredoc-fed `git commit` whose subject contains `gate's` ("unbalanced quoting"). No commit was made by that attempt.
- **Fix:** Committed with `git commit -F <message file>` instead. Subject and trailers are unchanged.
- **Files modified:** none
- **Committed in:** n/a

---

**Total deviations:** 2 auto-fixed (1 plan arithmetic defect, 1 blocking tooling quirk)
**Impact on plan:** None on scope or artifacts.

## Issues Encountered

- A first attempt to build the log's verdict paragraph used an unquoted heredoc, so the shell expanded the backticks and the paragraph was lost. It was caught before anything was committed; the paragraph was rewritten from a quoted heredoc file and the log's assertions were re-run.
- `.planning/config.json` was already modified in the working tree when execution started. It was left alone and is in no commit of this plan.

## Authentication Gates

None.

## Known Stubs

None.

## Threat Flags

None. The only new surface is the planted census control file, which existed under a trap and was never staged (T-04-10).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 04-03 can cherry-pick `565e3e25` (D-08) with the five 04-01 code commits onto a branch off `main`. D-08 is what stops `check-ruleset-drift.sh` from turning red on every PR to `main` once 04-05 adds the `fragment-required` context.
- After D-08, nothing in git mirrors `protect-main`'s required contexts. T-04-08 is accepted: a required context that never reports blocks every PR loudly, and 04-05 re-reads the live ruleset before and after its edit.

## Self-Check: PASSED

- Files: `04-MUTATION-LOG.md`, `REQUIREMENTS.md` and `ROADMAP.md` present and modified; `scripts/check-ruleset-drift.sh` and `.github/required-status-checks.txt` absent.
- Commits present on this branch: `565e3e25`, `dcbc5e31`, `36840380`.
- Plan acceptance criteria re-run: parse helpers present, one `Step:` in `runBodyExceptions`, deletions, ci.yml step order, D-08 commit file set, prose assertions, only-04-01 commits touched the mutated files, no CI-skip marker, every code commit typed `ci`, `test` or `chore`, REQUIREMENTS commit subject. All pass.

---
*Phase: 04-fragment-required-gate*
*Completed: 2026-10-11*
