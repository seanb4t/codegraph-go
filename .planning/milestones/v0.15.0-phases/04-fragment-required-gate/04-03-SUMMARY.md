---
phase: 04-fragment-required-gate
plan: 03
subsystem: ci
tags: [github-actions, cherry-pick, release-please, pull-request]

requires:
  - phase: 04-fragment-required-gate
    provides: "plan 04-01's fragment-required gate and 04-02's ruleset-drift guard retirement, authored as code-only commits on the milestone branch"
provides:
  - "Local and remote branch ci/fragment-required-gate: six cherry-picked ci/test commits (-x) on origin/main fbfd8a0c, carrying exactly the nine Phase 4 paths"
  - "Tracking issue #94 and non-draft PR #95 (base main, head ci/fragment-required-gate) with the Closes #94 issue line"
  - "04-03-PR-BODY.md, the feature-template body as opened"
affects: [04-04, 04-05]

actuals:
  tokens: 16800
  tasks: 3
  commits: 2
plan_head_before: 16047a4c24b1a138eb86a14666cce1d0439bd704
plan_head_after: 4c7d0361fd013e8406fcd6494342414e26b1fd2f

tech-stack:
  added: []
  patterns:
    - "Landing branch built in a separate worktree from origin/main with cherry-pick -x, so the milestone checkout never switches (D-06)"
    - "Outward-facing acts gated by a blocking-human decision checkpoint; each outward command run exactly once"

key-files:
  created:
    - .planning/phases/04-fragment-required-gate/04-03-PR-BODY.md
  modified: []

key-decisions:
  - "Maintainer chose open-new-issue: issue #94 opened first, PR #95 carries Closes #94 (the #90 precedent), instead of an existing-issue reference or the pr-issue-exempt marker"
  - "PR title kept as proposed (no replacement given): ci: add the fragment-required gate and retire the ruleset-drift guard"

requirements-completed: [GATE-01]

coverage:
  - id: D1
    description: "Local branch ci/fragment-required-gate on origin/main (fbfd8a0c) carries exactly the nine Phase 4 paths, line-equal to the milestone's Phase 4 diff, in ci/test commits with cherry-pick provenance"
    requirement: GATE-01
    verification:
      - kind: other
        ref: "git diff --shortstat fbfd8a0c d2baf7c6 -> 9 files changed, 916 insertions(+), 424 deletions(-); gh pr view 95 --json files lists exactly the nine paths"
        status: pass
    human_judgment: false
  - id: D2
    description: "Branch published and one non-draft PR (#95) opened against main with the approved title, a feature-template body and the Closes #94 issue line"
    requirement: GATE-01
    verification:
      - kind: other
        ref: "git ls-remote --heads origin ci/fragment-required-gate -> d2baf7c6; gh pr view 95 -> OPEN, isDraft false, base main, head ci/fragment-required-gate"
        status: pass
    human_judgment: false
  - id: D3
    description: "GATE-01's 'runs on every pull request' becomes true only when 04-04 merges PR #95 to main; until then the workflow does not exist on the base branch"
    requirement: GATE-01
    verification: []
    human_judgment: true
    rationale: "pull_request_target reads the workflow from main, so the gate cannot report on any PR until the maintainer merges #95 (04-04)"

duration: 2 sessions (Task 1 and checkpoint in the first; Tasks 2-3 in this continuation)
completed: 2026-10-11
status: complete
---

# Phase 4 Plan 3: Landing PR Summary

**The Phase 4 gate and the ruleset-drift guard removal are published as PR #95 to main: six ci/test commits cherry-picked onto origin/main, exactly nine paths, issue #94 linked, nothing else from the milestone carried.**

## Performance

- **Duration:** two sessions (Task 1 and the blocking-human checkpoint, then this continuation for Tasks 2-3)
- **Completed:** 2026-10-11
- **Tasks:** 3 (Task 2 is the maintainer's decision checkpoint)
- **Files modified:** 1 planning artifact on the milestone branch (`04-03-PR-BODY.md`); the landing branch changes 9 paths (+916/-424)

## Accomplishments

- Built `ci/fragment-required-gate` from origin/main in a separate worktree (the milestone checkout never switched) and proved it carries exactly the nine Phase 4 paths, line-equal to the milestone's own Phase 4 diff.
- Recorded the maintainer's answer, opened tracking issue #94, pushed the branch and opened one non-draft PR (#95) with `Closes #94`.
- Left merging and the ruleset edit to the maintainer (04-04, 04-05).

## Landing records

- **origin/main at branch creation (M0, also the merge-base):** `fbfd8a0c43eb5bae7035df58a6b4e3e3cfe09bb5`
- **Cherry-picked commits (milestone SHA to landing SHA, in order):**

| Milestone | Landing | Subject |
|-----------|---------|---------|
| `484fc4b1` | `d030e870` | test(04-01): expect shipped-code changes without a fragment to fail the gate (GATE-01, GATE-04) |
| `1989a4c7` | `4e3c3b10` | ci(04-01): add the fragment-required gate on pull_request_target (GATE-01) |
| `7d85f860` | `cb04c25a` | test(04-01): expect exemptions, fail-closed states and a complete file list (GATE-02, GATE-03) |
| `f1fa5347` | `1dea433d` | ci(04-01): honor changelog-exempt and fail closed on unseen changes (GATE-02, GATE-03) |
| `484a8e27` | `17d5ab2f` | ci(04-01): run the fragment-policy suite in the test job with an executed-count guard (GATE-04) |
| `565e3e25` | `d2baf7c6` | ci(04-02): retire the ruleset-drift guard and its required-check fixture (D-08) |

- **Gates (Task 1, run in the first session, reported in its checkpoint return as all passing):** path set equals the nine paths; per-path +/- lines equal the milestone's Phase 4 diff; `GOTOOLCHAIN=go1.26.6 go build ./...`, `go test ./internal/upgrade/ -count=1`, `task check:fragment-policy` (36 of 36) and `task lint:actions` in a worktree of the branch. This continuation did not re-run them; it re-verified the tip SHA, the empty remote branch before the push, and (after publishing) the remote tip and the PR's file list below.
- **Task 2, maintainer answer (verbatim):** `open-new-issue` (no replacement title).
- **Issue:** #94, https://github.com/seanb4t/codegraph-go/issues/94, "ci: require a changelog fragment on pull requests that change shipped code".
- **Pushed tip:** `d2baf7c674c66ddf3f00b81d4b4c06275bb03cf2` (`git ls-remote --heads origin ci/fragment-required-gate` matches the local ref).
- **PR:** #95, https://github.com/seanb4t/codegraph-go/pull/95. View JSON (files abbreviated to the path list):

```json
{"number":95,"state":"OPEN","isDraft":false,"baseRefName":"main","headRefName":"ci/fragment-required-gate","headRefOid":"d2baf7c674c66ddf3f00b81d4b4c06275bb03cf2","title":"ci: add the fragment-required gate and retire the ruleset-drift guard","files":[".github/required-status-checks.txt",".github/workflows/ci.yml",".github/workflows/components-drift.yml",".github/workflows/fragment-required.yml","Taskfile.yml","internal/upgrade/taskfile_shape_test.go","scripts/check-ruleset-drift.sh","scripts/fragment_policy.py","scripts/test_fragment_policy.py"]}
```

- `gh pr list --head ci/fragment-required-gate --state open` returns exactly one PR; the file list equals the nine paths.
- GATE-01's "runs on every pull request" becomes true when 04-04 merges #95, because `pull_request_target` reads the workflow from `main`.

## Task Commits

1. **Task 1: Build the landing branch, prove it, draft the PR body** - `4db7dd19` (docs; the PR body on the milestone branch). The branch itself is the six landing commits above.
2. **Task 2: Maintainer approval and issue-line choice** - checkpoint, no commit; answer `open-new-issue`.
3. **Task 3: Publish** - `4c7d0361` (docs, the final PR body with `Closes #94`); issue #94, the branch push and PR #95 are remote records, not commits.

**Plan metadata:** the docs(04-03) close-out commit that carries this SUMMARY, STATE.md, ROADMAP.md and REQUIREMENTS.md.

## Files Created/Modified

- `.planning/phases/04-fragment-required-gate/04-03-PR-BODY.md` - the landing PR body as opened, first line `Closes #94`
- `.planning/phases/04-fragment-required-gate/04-03-SUMMARY.md` - this summary

## Decisions Made

- `open-new-issue` over `open-existing-issue` and `open-exempt`: the issue satisfies require-issue-link the way #90 did (`Closes #89`), and the gate gets its own thread. The issue closes on merge although the ruleset step (04-05) remains, which the maintainer accepted by choosing it.
- PR title unchanged from the proposal; it is `ci:`-typed, so the squash commit on main is not releasable by release-please.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Stale plan commit ledger for 04-03 from an earlier milestone**
- **Found during:** SUMMARY close-out (measuring `commits:`)
- **Issue:** `.git/gsd-plan-head-before-04-03` held `9a93b2e2` ("docs(04-02): complete fang-verdict-and-denylist-widening plan", 2026-09-17), a commit that is not an ancestor of the milestone branch. The ledger was left by a previous milestone's phase 4 plan 3 and the "write only if absent" guard kept it, so `rev-list --count` from it returned 134.
- **Fix:** Used the real pre-task base, `16047a4c` (the parent of the first 04-03 commit, the 04-02 close-out commit), for `plan_head_before:` and the count, and overwrote the ledger file with that SHA. This ledger is under `.git/` and is not tracked.
- **Files modified:** none tracked
- **Verification:** `git rev-list --count 16047a4c..4c7d0361` is 2 (the two task-side commits; the close-out commit follows)
- **Committed in:** n/a

---

**Total deviations:** 1 auto-fixed (1 blocking, bookkeeping only)
**Impact on plan:** None on the landing branch, issue or PR. The 04-04 and later ledgers (`gsd-plan-head-before-04-04`, `-04-05`) already exist from the earlier milestone too, so the same stale-base hazard applies to them.

## Issues Encountered

- The earlier session's note that the ledger held the plan's start SHA was wrong; see the deviation above.
- GitHub reported 11 Dependabot alerts on the default branch in the push output. They are unrelated to this change and out of scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 04-04: the maintainer merges PR #95 (squash subject is the PR title) and the workflow then exists on main. Then 04-05 adds `fragment-required` to ruleset protect-main.
- The `fragment-required` check does not run on #95 itself (D-02), and require-issue-link, pr-title and pr-template-format are expected to evaluate it; their results are third-party behavior and were not used as evidence here.

## Self-Check: PASSED

- FOUND: `.planning/phases/04-fragment-required-gate/04-03-PR-BODY.md`
- FOUND commits on the milestone branch: `4db7dd19`, `4c7d0361`
- FOUND landing commits on `ci/fragment-required-gate`: `d030e870`, `4e3c3b10`, `cb04c25a`, `1dea433d`, `17d5ab2f`, `d2baf7c6`
- FOUND remote branch tip `d2baf7c6`; PR #95 open, non-draft, base main, nine paths

---
*Phase: 04-fragment-required-gate*
*Completed: 2026-10-11*
