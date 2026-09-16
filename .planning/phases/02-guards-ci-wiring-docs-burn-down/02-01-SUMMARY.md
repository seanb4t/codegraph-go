---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 01
subsystem: testing
tags: [ci-guards, web-drift, mutation-log, requirements-wording, taskfile]

# Dependency graph
requires: []
provides:
  - "02-MUTATION-LOG.md Family (a): the RED replay of web:drift against commit 98cd41dd's exact incident shape, plus a green control on HEAD"
  - "REQUIREMENTS.md GRD-09 and ROADMAP.md Phase 2 criterion 1 reworded to the proof-not-fix statement (D-03)"
affects: [02-07 (closes WINDOWS #29 citing this entry), 02-02, 02-04 (append Families (b)/(c) to the same 02-MUTATION-LOG.md)]

# Actuals (#2632)
actuals:
  tokens: 3338
  tasks: 2
  commits: 2
  plan_head_before: 1b1184caeee77b7677af1015cc4a61609921fe25

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Historical-incident replay via git worktree add --detach <scratch> <commit> — proving a guard is RED against its real recorded incident with zero code mutation, since the RED condition is the commit's own historical tree shape (mirrors 07-MUTATION-LOG.md Family (a))"
    - "Green control on HEAD in a second scratch worktree, run right after the RED replay, to prove the failure is a property of the incident commit's tree and not an artifact of running the gate inside a worktree at all"

key-files:
  created:
    - .planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md
  modified:
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md

key-decisions:
  - "No change to Taskfile.yml — D-01 held: CI's clean-checkout find enumeration already fails on 98cd41dd's exact shape (verified live), so the ledger's suggested find-vs-git-ls-files paired assertion would only test staging hygiene, not a real gate blind spot"
  - "GRD-09 is shared with plan 02-07 (requirements.ready-ids reported 0/1 ready) — not marked complete here; 02-07 closes WINDOWS #29 citing this plan's transcript once it also finishes"

patterns-established:
  - "Pattern: replay a historical commit in a detached scratch worktree under /tmp to prove a guard's RED condition is the tree's own shape, never touching the main working tree or the guard's source"

requirements-completed: []  # GRD-09 is shared with 02-07; requirements.ready-ids reported 0/1 ready — not marked complete by this plan

coverage:
  - id: D1
    description: "web:drift replayed against commit 98cd41dd in a clean detached scratch worktree: exit 201, SOURCE half MATCH (108 files), OUTPUT-half mismatch (32 vs 22 files, 8658e6fe... vs a76add11...), matching the exact incident shape WINDOWS #29 describes — committed to 02-MUTATION-LOG.md Family (a) alongside a green control on HEAD (exit 0, 119 source / 36 output files, both digests MATCH)"
    requirement: "GRD-09"
    verification:
      - kind: other
        ref: "task web:drift inside git worktree add --detach <scratch> 98cd41dd (RED) and HEAD (green control) — plan-level <verify> automated gates, all 4 passed"
        status: pass
    human_judgment: false
  - id: D2
    description: "REQUIREMENTS.md's GRD-09 row and ROADMAP.md's Phase 2 success criterion 1 (plus its Notes clause) reworded to the proof-not-fix statement per D-03: both name 98cd41dd, clean checkout, and retained-by-design, and neither contains paired/git-ls-files/both-halves-enumerate language; ROADMAP.md still carries all 7 Phase N headings"
    requirement: "GRD-09"
    verification:
      - kind: other
        ref: "rg-based content assertions over .planning/REQUIREMENTS.md and .planning/ROADMAP.md, plus a commit-file-list check — plan-level <verify> automated gates, all 5 passed"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-09-16
status: complete
---

# Phase 2 Plan 01: web:drift Historical-Incident Replay & GRD-09 Rewording Summary

**Replayed `web:drift` against commit `98cd41dd`'s real incident shape in a scratch worktree (exit 201, matching the exact recorded digest/count mismatch) with a green control on HEAD, committed as `02-MUTATION-LOG.md`, and reworded `REQUIREMENTS.md`/`ROADMAP.md` to the proof-not-fix statement — no code changed.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-09-16T01:25:04Z
- **Completed:** 2026-09-16T01:32:11Z
- **Tasks:** 2 completed
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments
- Proved, with zero code change, that `web:drift` already goes RED on a clean checkout of `98cd41dd` — the exact incident shape WINDOWS #29 describes (build output present on disk but never staged) — reproducing exit 201, a SOURCE-half MATCH at 108 files, and an OUTPUT-half mismatch of 32 vs 22 files with digests `8658e6fe64bb…` vs `a76add11cadd…`, matching the RESEARCH-verified transcript exactly.
- Ran a green control on HEAD in a second scratch worktree (`task web:drift` exit 0, 119 source files / 36 output files, both digests MATCH), proving the RED above is a property of `98cd41dd`'s tree and not an artifact of the gate running inside any worktree.
- Committed both transcripts as `02-MUTATION-LOG.md` Family (a), in the `07-MUTATION-LOG.md` shape, recording that WINDOWS #29's suggested `find`-vs-`git ls-files` paired assertion is declined per D-01.
- Reworded `REQUIREMENTS.md`'s GRD-09 row and `ROADMAP.md`'s Phase 2 success criterion 1 (and its Notes clause) via scoped `Edit` calls to state the proof that was actually delivered, with no surviving "paired assertion" language.

## Task Commits

Each task was committed atomically:

1. **Task 1: Replay 98cd41dd end-to-end — clean worktree → `task web:drift` → RED, plus the green control on HEAD** - `0e925685` (docs)
2. **Task 2: Reword REQUIREMENTS.md GRD-09 and ROADMAP.md criterion 1 to the proof-not-fix statement (D-03)** - `cd8b4bc9` (docs)

**Plan metadata:** committed in the same pass as this SUMMARY (see below).

_Note: this is a `type: execute` plan, not a TDD plan — TDD is applicable to this dispatch overall (`workflow.tdd_mode` active), but Task 1 is a `type="tracer"` replay/measurement task and Task 2 is a documentation-wording edit; neither adds new behavior (`tracker-id`-less, no `<behavior>` block, and the files touched are a mutation-log artifact and two tool-owned planning docs, not production source), so no RED-commit gate applies to either task._

## Files Created/Modified
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` - New phase artifact; Family (a) records the RED replay and green control (Families (b)/(c) appended later by plans 02-02/02-04)
- `.planning/REQUIREMENTS.md` - GRD-09 row reworded to the proof-not-fix statement (value change only, checkbox and bold ID unchanged)
- `.planning/ROADMAP.md` - Phase 2 success criterion 1 and its Notes clause reworded to match (value changes only; all 7 `#### Phase N:` headings intact)

## Decisions Made
- No change to `Taskfile.yml`: D-01 held after this session's own live replay confirmed CI's clean-checkout `find` enumeration already fails on `98cd41dd`'s exact shape — a `find`-vs-`git ls-files` paired assertion would only test staging hygiene (whether the developer ran `git add`), not close a real gate blind spot.
- GRD-09 is declared by both this plan and 02-07 (`requirements.ready-ids` reported `0/1 requirement(s) ready to mark complete`), so it is intentionally left unchecked in `REQUIREMENTS.md` and absent from `requirements-completed` here — 02-07 will close WINDOWS #29 citing this plan's transcript once it also finishes, at which point the shared ID becomes ready.

## Deviations from Plan

None — plan executed exactly as written. Both replays reproduced the RESEARCH-verified transcript byte-for-byte (counts, digests, exit code), so no fix attempt or investigation was needed.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `02-MUTATION-LOG.md` exists with its Family (a) entry and the shared header/scope preamble that plans 02-02 and 02-04 append Families (b) and (c) to.
- `REQUIREMENTS.md`/`ROADMAP.md` now carry the corrected GRD-09 language other plans (notably 02-07's WINDOWS-closing citation) can reference verbatim.
- GRD-09 stays open in `REQUIREMENTS.md` until 02-07 also completes — no blocker, expected per the shared-ID gate.

---
*Phase: 02-guards-ci-wiring-docs-burn-down*
*Completed: 2026-09-16*
