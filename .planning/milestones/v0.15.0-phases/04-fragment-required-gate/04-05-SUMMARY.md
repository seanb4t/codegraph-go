---
phase: 04-fragment-required-gate
plan: 05
subsystem: ci
tags: [github-rulesets, required-status-checks, release-please, changie]

requires:
  - phase: 04-fragment-required-gate
    provides: "plan 04-04's fragment-required workflow on origin/main (job name fragment-required), so the context can report"
provides:
  - "Ruleset protect-main (id 20157557) requires the fragment-required status check, 9 required contexts, edit timestamp 2026-10-13T12:16:55.257-04:00"
  - "04-05-ruleset-pre.json, 04-05-ruleset-put-body.json and 04-05-ruleset-post.json: the read before, the reviewed PUT body, and the read after"
affects: [phase-04-verification]

actuals:
  tokens: 6000
  tasks: 3
  commits: 3
plan_head_before: 02691ceb7b7606bcbfccf7fa758feb6c65a2718e
plan_head_after: 0f2643b4d6be6e917dda315747f1b1a1e1a75d43

tech-stack:
  added: []
  patterns:
    - "Ruleset edit as a blocking-human checkpoint: authenticated pre-read, jq-asserted PUT body (pre-read plus exactly one context), one PUT, authenticated post-read asserted against the pre-read"

key-files:
  created:
    - .planning/phases/04-fragment-required-gate/04-05-ruleset-pre.json
    - .planning/phases/04-fragment-required-gate/04-05-ruleset-put-body.json
    - .planning/phases/04-fragment-required-gate/04-05-ruleset-post.json
    - .planning/phases/04-fragment-required-gate/04-05-SUMMARY.md
  modified: []

key-decisions:
  - "Maintainer chose run-prepared: the executor ran the prepared PUT once, after re-verifying the live ruleset still equalled the pre-read"
  - "The checkpoint was first held (2026-10-13) so the two red required checks on main could be fixed first (PR #96, d465f8ed), then v0.14.1 was released; the PUT ran only after that"

requirements-completed: [GATE-05]

coverage:
  - id: D1
    description: "protect-main's required status checks now include fragment-required; the sorted context set equals the pre-read set plus fragment-required (8 to 9); enforcement active; bypass_actors and the strict policy unchanged; updated_at changed"
    requirement: GATE-05
    verification:
      - kind: other
        ref: "plan Task 3 automated verify 1 (jq -e of 04-05-ruleset-post.json against 04-05-ruleset-pre.json) -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The live ruleset lists fragment-required exactly once"
    requirement: GATE-05
    verification:
      - kind: other
        ref: "plan Task 3 automated verify 2 (live gh api read, rg -x -F count = 1) -> exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "The edit changed nothing but the one context and updated_at: a line diff of the key-sorted pre and post reads shows only the appended {context, integration_id 15368} entry and the updated_at value"
    requirement: GATE-05
    verification:
      - kind: other
        ref: "git diff --no-index 04-05-ruleset-pre.json 04-05-ruleset-post.json -> 5 insertions, 1 deletion (the entry and updated_at)"
        status: pass
    human_judgment: false
  - id: D4
    description: "How a live pull request to main reports the fragment-required context, and whether open PRs (including #88) show it as expected until their next event, is GitHub's behavior and was not exercised by this plan"
    requirement: GATE-05
    verification: []
    human_judgment: true
    rationale: "The plan states this as an expectation at the checkpoint (rule eyva2svrcz), not a test; it is first observable on the next PR event after the edit and belongs to phase verification"

duration: 4 min
completed: 2026-10-13
status: complete
---

# Phase 4 Plan 5: Require fragment-required on protect-main Summary

**Ruleset protect-main now requires the `fragment-required` status check (8 contexts to 9), applied by one maintainer-authorized PUT of a body jq-proven to be the pre-read plus exactly that context, with the edit timestamp recorded.**

## Performance

- **Duration:** about 4 min for this continuation (Task 1 ran earlier; Task 2 was the maintainer's decision)
- **Started:** 2026-10-13T16:12Z (continuation)
- **Completed:** 2026-10-13T16:18Z
- **Tasks:** 3 (Task 1 automated, Task 2 a blocking-human checkpoint, Task 3 automated)
- **Files modified:** 3 planning evidence files plus this summary and the GSD state files

## Accomplishments

- Re-read ruleset 20157557 read-only and proved it still equalled the Task 1 pre-read immediately before the write (see Maintainer record).
- Ran the exact prepared PUT once. It succeeded (exit 0) and returned `enforcement: active`, `updated_at: 2026-10-13T12:16:55.257-04:00`.
- Re-read the ruleset, saved it key-sorted as `04-05-ruleset-post.json`, and proved against the pre-read that only `fragment-required` was added.

## Maintainer record (Task 2, quoted verbatim)

- **Reply:** `run-prepared` (given through an explicit choice prompt).
- **Path that applied the edit:** the authorized command, run by the executor, once: `gh api --method PUT repos/seanb4t/codegraph-go/rulesets/20157557 --input .planning/phases/04-fragment-required-gate/04-05-ruleset-put-body.json`. Not the UI. Exit 0, HTTP success, no retry.
- **History of the checkpoint:** it was first answered `hold` on 2026-10-13 ("can we just fix the other red items first?"). The two red required checks on main (`govulncheck` and `test`, from the dependency advisories noted in 04-04) were fixed through PR #96, merged as `d465f8ed`, and v0.14.1 was then released (`88e79013`). The maintainer then replied `run-prepared`.
- **Live state immediately before the write** (read-only GET by the orchestrator, then again by this executor): `updated_at` 2026-09-16T07:29:44.197-04:00 (equal to the pre-read), `enforcement` active, the same 8 contexts, `fragment-required` absent. This executor compared before writing: the sorted live contexts plus `fragment-required` equalled the sorted contexts of `04-05-ruleset-put-body.json`; live `bypass_actors`, `conditions` and the strict policy equalled the body's.

## Required status checks, before and after

**Pre-read (count 8):**

1. test
2. actionlint (workflow static analysis)
3. perf regression gate (PERF-02, INDX-06)
4. pr-title
5. reproducibility (double-build hash-diff, DIST-04)
6. govulncheck (DIST-03, blocking)
7. goreleaser check (config validation, DIST-01)
8. tmux e2e (real-pty harness, TTY-01..TTY-07)

**Post-read (count 9):** the same 8, then 9. `fragment-required` (integration_id 15368, the id every existing entry carries).

## Edit timestamp (GATE-05)

- **Post-edit `updated_at`:** `2026-10-13T12:16:55.257-04:00` (the ruleset's own timestamp; 16:16:55Z). The pre-read's was `2026-09-16T07:29:44.197-04:00`.
- **Observation time (UTC, `date -u +%Y-%m-%dT%H:%M:%SZ`):** `2026-10-13T16:17:04Z`, when the read-only post-read was taken.
- **Unchanged:** `enforcement` active, `bypass_actors` (admin bypass, RepositoryRole 5), `conditions`, rule types, and the strict policy (`false`).

## Verification (run under bash)

| Check | Exit |
|---|---|
| Task 3 automated 1 (post-read vs pre-read: contexts = pre + fragment-required, count + 1, active, bypass_actors and strict equal, updated_at differs) | 0 |
| Task 3 automated 2 (live read lists `fragment-required` exactly once) | 0 |
| Task 1 automated 1 and 2 (body vs pre-read; job name on origin/main) | 0, 0 (recorded by Task 1, commit 1a7fc999) |

Acceptance criteria for this task (summary records `updated_at`, an ISO timestamp and `fragment-required`; the post-read commit subject matches `^docs\(04-05\): record fragment-required as a required check`) are met by this file and by commit `0f2643b4`.

## Task Commits

1. **Task 1: Prepare the protect-main edit** - `1a7fc999` (docs)
2. **Task 2: Maintainer checkpoint** - no executor commit; the maintainer's reply was `run-prepared` and the PUT left no repository change of its own
3. **Task 3: Record fragment-required as a required check** - `0f2643b4` (docs, the post-read)

**Plan metadata:** the docs(04-05) close-out commit carrying this SUMMARY, STATE.md and ROADMAP.md.

`plan_head_before` is the ledger value `02691ceb`. Quick tasks 261011-sor and 261013-esj were run between Task 1 and Task 3 (`0555b5ab`, `88b07d76`, `09cdf67d`, `d465f8ed`, `88e79013`, `3b21f39d`, `3b93121d`), so `git rev-list --count 02691ceb..HEAD` reads 9 at SUMMARY time. `actuals.commits: 3` counts only this plan's own commits (`1a7fc999`, `0f2643b4`, and the close-out docs commit).

## Files Created/Modified

- `.planning/phases/04-fragment-required-gate/04-05-ruleset-pre.json` - protect-main as read before the edit
- `.planning/phases/04-fragment-required-gate/04-05-ruleset-put-body.json` - the PUT body: the pre-read plus one context
- `.planning/phases/04-fragment-required-gate/04-05-ruleset-post.json` - protect-main as read after the edit
- `.planning/phases/04-fragment-required-gate/04-05-SUMMARY.md` - this summary

## Decisions Made

- Re-verified the live ruleset against the prepared body before the write rather than relying on the orchestrator's earlier read, because the checkpoint had been held across several repository changes; the comparison matched, so the write went ahead.
- Wrote nothing to the ruleset except the one PUT.

## Deviations from Plan

None - plan executed exactly as written. The checkpoint was held once and then resumed; that is checkpoint flow, not a deviation.

## Issues Encountered

None. The PUT was accepted first time.

## Threat Flags

None. The only new surface is the ruleset change this plan was built to make (T-04-18 through T-04-21 in the plan's register).

## Known Stubs

None.

## Open follow-up

- **Live behavior of the gate on a pull request to main** has not been observed: open PRs (including #88) show `fragment-required` as expected until their next opened, edited, reopened, synchronize or ready_for_review event. This is phase verification's to check.
- **Recovery path if the gate misbehaves** (T-04-21, accepted): the admin bypass (RepositoryRole 5) or a ruleset edit by the maintainer.
- The milestone branch is unpublished; pushing is the maintainer's act.

## User Setup Required

None - the one repository-settings action (D-07) is done.

## Next Phase Readiness

- All five Phase 4 plans are executed. GATE-05 is met as D-09 amended it. Ready for phase verification.

## Self-Check: PASSED

- FOUND: `04-05-ruleset-pre.json`, `04-05-ruleset-put-body.json`, `04-05-ruleset-post.json`
- FOUND commits `1a7fc999` and `0f2643b4` as ancestors of HEAD
- Task 3 verify and the live read exit statuses recorded above are 0

---
*Phase: 04-fragment-required-gate*
*Completed: 2026-10-13*
