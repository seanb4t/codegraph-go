---
phase: 05-release-chain-release-please-retirement
plan: 08
subsystem: docs
tags: [release-runbook, d-15, gap-closure, docs-only]
gap_closure: true
gap_ids: [G-05-1]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: D-15 no-hand-pushed-prerelease-tag paragraph in docs/RELEASE-PROCEDURES.md section 4 (05-06, 05-07)
provides:
  - docs/RELEASE-PROCEDURES.md sections 1, 2 and 7 consistent with D-15
affects: [05-REVIEW-DISPOSITION (finding IN-03), phase 05 verification]

actuals:
  tokens: 400
  tasks: 1
  commits: 1
plan_head_before: a1cc773239698b005e80829780081cb6145861e2
plan_head_after: 5653cbf9dd04bc49c30e3b80195aa837897e695f

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - docs/RELEASE-PROCEDURES.md

key-decisions:
  - "Section 2's prerelease row is removed, not rewritten; a pointer sentence names D-15 and section 4 rather than repeating it"
  - "Section 1 drops the rc-or-stable parenthetical entirely; the historical v0.1 rc.1 failure sentence is kept as a record of an event"
  - "Section 7 deletes the prerelease-is-transient sentence and keeps the stable-release no-retag sentence"

requirements-completed: [DOCS-13]

coverage:
  - id: D1
    description: "Section 2 of the runbook lists only the Stable and Internal milestone marker tag shapes, under a Two distinct shapes intro, with a D-15 pointer to section 4"
    requirement: DOCS-13
    verification:
      - kind: other
        ref: "05-08-PLAN.md Task 1 automated verify (awk/rg assertions on docs/RELEASE-PROCEDURES.md)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Section 1 pre-tag gate sentence no longer names a prerelease kind; section 7 no longer treats prerelease tags as re-pushable"
    requirement: DOCS-13
    verification:
      - kind: other
        ref: "05-08-PLAN.md Task 1 automated verify (rc or stable / safer to absent; stable sentence kept)"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/upgrade test that reads the runbook still passes"
    requirement: DOCS-13
    verification:
      - kind: unit
        ref: "internal/upgrade#TestCosignIdentityPolicyBoundaryParityWithCompiledPattern"
        status: pass
    human_judgment: false

duration: 2min
completed: 2026-10-16
status: complete
---

# Phase 05 Plan 08: Align release runbook with D-15 Summary

**Removed the three stale hand-pushed prerelease-tag passages from docs/RELEASE-PROCEDURES.md (sections 1, 2, 7) so the runbook agrees with section 4's D-15 paragraph; closes UAT gap G-05-1.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-10-16T20:17:21Z
- **Completed:** 2026-10-16T20:18:30Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Section 2 now says "Two distinct shapes", its table has only the Stable and Internal milestone marker rows, and a new paragraph states there is no hand-pushed prerelease tag shape (D-15) and points to section 4.
- Section 1's pre-tag sentence reads "Before tagging **anything**, run the following for **all 4** release targets." The historical v0.1 `rc.1` failure sentence is unchanged.
- Section 7's tag-reuse paragraph ends after the stable-release sentence; the "prereleases are safer to delete/retag" sentence is gone.
- Review finding IN-03 is resolved by this commit, so the next code-review gate run can record its disposition. 05-REVIEW-DISPOSITION.md was not touched (tool-owned).

## Task Commits

1. **Task 1: Align runbook sections 1, 2 and 7 with D-15** - `5653cbf9` (docs)

**Plan metadata:** committed separately as `docs(05-08): complete ...` (SUMMARY, STATE, ROADMAP).

## Verification Results

Task automated verify (exact command from the plan): exit 0 (`VERIFY_EXIT=0`). It asserted the section 4 D-15 paragraph is byte-identical to 1c700149, the line-15 historical sentence is kept, section 2 has the two-shape intro, 4 table lines, D-15 and section 4 pointer, section 7 keeps its stable sentence, the stale passages are gone, Taskfile.yml is unchanged against 1c700149, and the test PASS line was matched:

```
--- PASS: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (0.00s)
```

Plan-level commit check (run right after the task commit, before this SUMMARY was committed): exit 0 (`COMMIT_CHECK_EXIT=0`). HEAD subject starts with `docs(05-08): `, contains no skip-CI phrase, and `git diff-tree` lists only `docs/RELEASE-PROCEDURES.md`. Post-commit deletion check: no deleted files.

## Files Created/Modified
- `docs/RELEASE-PROCEDURES.md` - sections 1, 2 and 7 aligned with D-15 (9 insertions, 6 deletions)

## Decisions Made
Followed the plan's recorded two-way-door wording decisions (remove row, pointer sentence, drop section 1 parenthetical, delete section 7 sentence). No new decisions.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None. Working tree held pre-existing uncommitted `.planning/STATE.md`, `.planning/state.json` and untracked `.boost/` before the plan started; `.boost/` was left alone.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Threat Flags

None. T-05-29 (hand-pushed prerelease tag presented as normal) is mitigated by the removed row and sentence plus the section 4 pointer.

## Next Phase Readiness
- G-05-1 is closed at the docs level. This was the last gap-closure plan for phase 05; phase verification and the next code-review gate run can proceed.

## Self-Check: PASSED

- FOUND: docs/RELEASE-PROCEDURES.md
- FOUND commit: 5653cbf9 (ancestor of HEAD)

---
*Phase: 05-release-chain-release-please-retirement*
*Completed: 2026-10-16*
