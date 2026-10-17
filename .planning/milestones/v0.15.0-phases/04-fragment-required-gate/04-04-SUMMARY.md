---
phase: 04-fragment-required-gate
plan: 04
subsystem: ci
tags: [github-actions, pull_request_target, merge-back, release-please, govulncheck]

requires:
  - phase: 04-fragment-required-gate
    provides: "plan 04-03's landing PR #95 (ci/fragment-required-gate, tip d2baf7c6) carrying the nine Phase 4 paths"
provides:
  - "origin/main 65a2b9a9: the fragment-required workflow, its policy script and tests, with the ruleset-drift guard and its required-check fixture retired (maintainer's squash merge of PR #95)"
  - "One D-06 merge commit b47a52ce on gsd/v0.15.0-milestone (parents a5a40dba and 65a2b9a9), no rebase, no reset, nothing pushed"
  - "Milestone tree byte-identical to its pre-merge tip, and agreeing with main on the gate's own files"
affects: [04-05]

actuals:
  tokens: 3000
  tasks: 2
  commits: 2
plan_head_before: a5a40dba0cd23ed7d9680b658b510fe41d953f31
plan_head_after: b47a52ce9da52e66f5d3c36d0271e33c08c47d87

tech-stack:
  added: []
  patterns:
    - "Landing PR squash-merged by the maintainer, then main merged back into the milestone with git merge --no-ff (D-06); the milestone tree does not change because the squash carries content the milestone already holds"

key-files:
  created:
    - .planning/phases/04-fragment-required-gate/04-04-SUMMARY.md
  modified: []

key-decisions:
  - "Maintainer merged PR #95 while govulncheck and test were red from pre-existing dependency advisories; the in-PR fix (quick task 261011-sor) was dropped, so the advisories remain an open follow-up outside this plan"
  - "D-06 merge-back was a clean merge with zero review edits on the nine paths, so no hand resolution and no recorded divergence"

requirements-completed: [GATE-01]

coverage:
  - id: D1
    description: "PR #95 is MERGED by the maintainer; origin/main carries .github/workflows/fragment-required.yml, scripts/fragment_policy.py and scripts/test_fragment_policy.py and lacks scripts/check-ruleset-drift.sh and .github/required-status-checks.txt"
    requirement: GATE-01
    verification:
      - kind: other
        ref: "gh pr view 95 --json state,mergedAt,mergeCommit,mergedBy -> MERGED 2026-10-12T14:18:12Z 65a2b9a9 seanb4t; git cat-file -e origin/main:<path> exits 0 for the three present paths and 128 for the two retired ones"
        status: pass
    human_judgment: false
  - id: D2
    description: "gsd/v0.15.0-milestone gained exactly one merge commit b47a52ce with first parent a5a40dba (pre-merge tip) and second parent 65a2b9a9 (origin/main); origin/main is an ancestor of HEAD; merge changes none of the nine Phase 4 paths"
    requirement: GATE-01
    verification:
      - kind: other
        ref: "plan Task 2 automated verify blocks 1 and 2 -> exit 0; acceptance criteria 1 and 2 -> exit 0; git diff a5a40dba HEAD -> empty"
        status: pass
    human_judgment: false
  - id: D3
    description: "The milestone's regression gates stay green after the merge-back"
    requirement: GATE-01
    verification:
      - kind: integration
        ref: "GOTOOLCHAIN=go1.26.6 task test:unit; task check:fragment-policy (36 of 36); task check:changie (13 of 13); task lint:actions -> all exit 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "GATE-01's 'runs on every pull request' now holds structurally, since pull_request_target reads the workflow from main; no real pull request to main has exercised it yet"
    requirement: GATE-01
    verification: []
    human_judgment: true
    rationale: "The workflow's behavior on a live PR to main is first observable on the next PR opened after #95 and is judged in 04-05 and phase verification, not by anything this plan ran"

duration: 5 min
completed: 2026-10-12
status: complete
---

# Phase 4 Plan 4: Land the Gate on Main and Merge Back Summary

**The maintainer squash-merged PR #95 to main (65a2b9a9), and main is merged back into the milestone with one clean `--no-ff` merge commit (b47a52ce) that leaves the milestone tree unchanged, so the fragment-required gate now lives where `pull_request_target` reads it and no history was rewritten.**

## Performance

- **Duration:** 5 min for this continuation (Task 1 was the maintainer's act, resolved before it started)
- **Started:** 2026-10-12T14:21:48Z
- **Completed:** 2026-10-12T14:26Z
- **Tasks:** 2 (Task 1 a blocking-human checkpoint, Task 2 automated)
- **Files modified:** 0 source files changed by the merge relative to its first parent; this summary and the GSD state files

## Accomplishments

- Re-ran Task 1's verification as Task 2's precondition and confirmed it: PR #95 is MERGED, and after `GIT_TERMINAL_PROMPT=0 git fetch origin` the gate workflow is on origin/main.
- Recorded that main received exactly what the milestone carries: `git diff ci/fragment-required-gate origin/main -- <nine paths>` is empty (0 bytes), so there are no review edits.
- Merged origin/main into the milestone with `git merge --no-ff` and no conflict; proved the parents, the empty change on the nine paths, and that origin/main is now an ancestor.
- Re-ran the milestone's four regression gates after the merge; all pass. Nothing was pushed.

## Maintainer record (Task 1, quoted verbatim)

- **Reply:** `95 merged` (the plan's resume signal was `merged`; the maintainer's wording named the PR number as well, and the state was verified independently below).
- **PR #95:** MERGED, `mergedAt` 2026-10-12T14:18:12Z, `mergeCommit` `65a2b9a9cac4596332a11579470e1eca99140f81`, `mergedBy` seanb4t. Issue #94 is CLOSED through `Closes #94`.
- **origin/main:** `65a2b9a9`, subject "ci: add the fragment-required gate and retire the ruleset-drift guard (#95)", a squash commit on the old merge-base `fbfd8a0c`. `git diff --stat fbfd8a0c origin/main` touches exactly the nine Phase 4 paths.

## Merge record (Task 2)

- **P0 (pre-merge milestone tip):** `a5a40dba0cd23ed7d9680b658b510fe41d953f31`
- **M1 (origin/main):** `65a2b9a9cac4596332a11579470e1eca99140f81`
- **Review-edit diff:** none (`git diff ci/fragment-required-gate origin/main -- <nine paths>` printed nothing; the local `ci/fragment-required-gate` is still at `d2baf7c674c66ddf3f00b81d4b4c06275bb03cf2`).
- **Merge commit:** `b47a52ce9da52e66f5d3c36d0271e33c08c47d87`, subject `chore(04-04): merge main into the milestone after the fragment-required gate landed (D-06)`. `git rev-list --parents -n 1 HEAD` printed `b47a52ce... a5a40dba... 65a2b9a9...`, so the parents are P0 then M1. The strategy was `ort`, with auto-merges on `.github/workflows/ci.yml`, `Taskfile.yml` and `internal/upgrade/taskfile_shape_test.go` and no conflict. The message carries no CI-skip marker.
- **Proofs:**
  - `git diff a5a40dba HEAD` is empty for the whole tree, so the merge changed nothing relative to its first parent, and a fortiori nothing on the nine paths.
  - `git merge-base --is-ancestor origin/main HEAD` exits 0.
  - The reflog for the milestone branch shows only commits and the one `merge origin/main` entry; there is no rebase or reset entry (T-04-16).
- **Verify and acceptance (verbatim from the plan, run under bash):**

| Check | Exit |
|---|---|
| Automated 1 (single chore(04-04) merge on the first-parent chain, 3-word parents line, `X^2` on origin/main with the gate workflow, origin/main in HEAD) | 0 |
| Automated 2 (`B` ancestor of `X^1`; merge's change on the nine paths equals the PR's review edits) | 0 |
| Automated 3 (`GOTOOLCHAIN=go1.26.6 task test:unit && task check:fragment-policy && task check:changie && task lint:actions`), run as four separate commands | 0, 0, 0, 0 |
| Acceptance 1 (every cherry-pick source SHA is an ancestor of HEAD) | 0 |
| Acceptance 2 (on the milestone branch, `git diff --quiet origin/main HEAD` over `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py`, `.github/workflows/fragment-required.yml`) | 0 |

## Regression gates (final lines)

- `GOTOOLCHAIN=go1.26.6 task test:unit` exit 0. Final lines: `ok  github.com/seanb4t/codegraph-go/tools/transcriptfreeze 0.316s` and `ok  github.com/seanb4t/codegraph-go/web 0.370s`. Zero `--- FAIL` or `FAIL` lines in the log. No flake occurred, so no package was re-run.
- `task check:fragment-policy` exit 0: `Ran 36 tests in 0.001s`, `OK`, `check:fragment-policy: executed 36 of 36 defined test methods (floor 36)`.
- `task check:changie` exit 0: `check:changie: 13 of 13 checks passed against a scratch copy (source tree byte-unchanged)`.
- `task lint:actions` exit 0: `GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml`, no findings.
- **Unpublished commit count:** `git rev-list --count origin/gsd/v0.15.0-milestone..HEAD` is **52** at the time of the merge, before this plan's docs commit. The branch was not pushed.

## Task Commits

1. **Task 1: Maintainer merges the landing PR** - no executor commit; the maintainer's squash merge is `65a2b9a9` on main (checkpoint:human-action).
2. **Task 2: Merge main back into the milestone** - `b47a52ce` (chore, `--no-ff` merge; a TDD RED commit does not apply because the merge changes no behavior).

**Plan metadata:** the docs(04-04) close-out commit carrying this SUMMARY, STATE.md, ROADMAP.md, REQUIREMENTS.md and state.json.

## Files Created/Modified

- `.planning/phases/04-fragment-required-gate/04-04-SUMMARY.md` - this summary
- The merge itself modifies no tracked file relative to its first parent. Relative to origin/main it adds nothing either, because the squash on main carries content the milestone already had.

## Decisions Made

- Merge-back left unresolved by hand and untouched by review edits: with an empty review-edit diff and a clean `ort` merge, nothing needed recording against T-04-17.
- The dirty `.planning/STATE.md` and `.planning/state.json`, the untracked `.boost/` and the untracked quick-task directory `261011-sor-fix-pr-95-dependency-advisories-x-net-go/` rode through the merge untouched and were not staged in the merge commit. `.boost/` and the quick-task directory are not staged in the docs commit either.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Dropped the redundant `-c commit.gpgsign=false` from the merge command**
- **Found during:** Task 2, step 3
- **Issue:** The harness's git-merge guard refused `git -c commit.gpgsign=false merge --no-ff -F <file> origin/main` as a merge onto an "undetermined branch" (the `-c` global option defeated its branch detection). The current branch was gsd/v0.15.0-milestone, which the guard allows for any merge.
- **Fix:** Re-ran the same merge without `-c commit.gpgsign=false`. `git config --get commit.gpgsign` already prints `false`, so the commit was unsigned exactly as intended. No guard was bypassed or altered; the first attempt made no change to the repository.
- **Files modified:** none
- **Verification:** `git rev-list --parents -n 1 HEAD` shows the intended parents; commit message and trailers as specified.
- **Committed in:** `b47a52ce`

---

**Total deviations:** 1 auto-fixed (1 blocking, tooling only)
**Impact on plan:** None on the merge content.

### Notes on the plan's assumptions (not deviations by the executor)

- **The plan assumed green checks and no bypass; the maintainer merged PR #95 with two checks red.** Task 1's instructions said to merge "when its required checks are green". When the maintainer merged, two checks were still FAILURE: `govulncheck (DIST-03, blocking)` and `test`, where the `test` failure comes from `task web:audit`. The maintainer chose to merge anyway, overriding an earlier ruling to fix the advisories inside PR #95.
  - Both failures are pre-existing on main and come from dependency advisories, not from Phase 4's code:
    - `golang.org/x/net` v0.58.0: GO-2026-6612, GO-2026-6613 and GO-2026-6617, fixed in v0.60.0;
    - stdlib `net/http` at go1.26.6, fixed in go1.26.9;
    - 18 pnpm advisories in `devalue`, `undici` and `source-map-js`.
  - An in-PR fix was planned as quick task `261011-sor` but never executed, and it was dropped. Its directory is still untracked in the working tree and was not touched.
  - The advisories are still present on main and on the milestone. No dependency was bumped in this plan. The `GOTOOLCHAIN=go1.26.6` pin stayed in place, because go.mod was not changed and the ambient Homebrew Go 1.27.1 breaks `cockroachdb/swiss`.
- **Reply wording:** the maintainer wrote `95 merged`, not the bare `merged` in the resume signal. It was accepted on the strength of the independent `gh pr view` and `git cat-file` checks.

## Issues Encountered

- None blocking. The plan commit ledger `.git/gsd-plan-head-before-04-04` read `a5a40dba`, which equals P0, so `plan_head_before:` is measured and not narrated. The count `commits: 2` is `git rev-list --count a5a40dba..HEAD` at SUMMARY time: the merge commit and the squash commit `65a2b9a9` it brought in (one first-parent commit).

## Open follow-up

- **Pre-existing dependency advisories on main and the milestone** (x/net to v0.60.0 or later, a Go toolchain at go1.26.9 or later, and the 18 pnpm advisories). Until fixed, `govulncheck` and `test` stay red on every pull request to main. This affects 04-05, which makes `fragment-required` a required check: a ruleset that requires checks must not be set up in a way that relies on those two red checks being green. Owner: the maintainer to schedule a dependency-bump change; it is not this plan's work.
- **Publishing the milestone:** 52 commits on the milestone are unpublished at merge time (plus this plan's docs commit). Pushing is the maintainer's act.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 04-05: the workflow now exists on main, so `fragment-required` can be added to ruleset protect-main (D-05).
- Concern for 04-05: the red `govulncheck` and `test` checks above, and that GATE-01's live behavior on a real PR has not been observed yet.

## Self-Check: PASSED

- FOUND: `.planning/phases/04-fragment-required-gate/04-04-SUMMARY.md`
- FOUND commit `b47a52ce9da52e66f5d3c36d0271e33c08c47d87` (ancestor of HEAD); parents `a5a40dba` and `65a2b9a9` confirmed with `git rev-list --parents -n 1`
- FOUND `65a2b9a9` as an ancestor of HEAD and as origin/main
- All five verify and acceptance exit statuses recorded above are 0

---
*Phase: 04-fragment-required-gate*
*Completed: 2026-10-12*
