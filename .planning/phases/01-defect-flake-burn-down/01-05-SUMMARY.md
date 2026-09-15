---
phase: 01-defect-flake-burn-down
plan: 05
subsystem: infra
tags: [ci, bench, github-actions, process-decision]

# Dependency graph
requires: []
provides:
  - "GH #20 follow-up 1 (Namespace cache volume on 8x16) recorded as won't-do in `tools/bench/BASELINE.md`, with the bar it was measured against and the two facts that decide it"
  - "GH #20 follow-up 2's discriminator fully specified and reproducible from `tools/bench/BASELINE.md` alone: the old-baseline commit located and confirmed (d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8, 2026-07-31, files_per_sec 11279.591291175333), the drift computed against the current committed baseline (+51.5%, reconciled against the issue's own +44.8%), and the exact `workflow_dispatch` ref/job/flags/trials/runner-class the discriminator run must use"
affects: [none — this is a documentation/process closure, not code]

# Actuals (#2632) — chars/4 over the realized diff, not a harness token count
actuals:
  tokens: 1430
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - tools/bench/BASELINE.md

key-decisions:
  - "Follow-up 1 closed WON'T-DO on the record: `ubuntu-latest` is free and measured 28.6x headroom (0.35% disagreement) against Namespace 4x8's 2.30x (4.36%), the issue's own adoption bar was 'match ubuntu-latest's stability', and a cache volume only reaches the already-refuted storage-latency hypothesis (Round 3) — it cannot reach the leading unrefuted explanation, host-placement variance."
  - "Reconciled the two different drift percentages in play: BASELINE.md's own arithmetic (11279.59 -> 17090.88, the current committed baseline) gives +51.5%; GH #20's stated +44.8% anchors the same 11279.59 against an earlier intermediate reading (16330.41, one of two 2-session disk-scratch control medians from the same investigation, not the eventual 7-trial rebless value). Recorded both, with which figures produced each, rather than silently picking one."
  - "Specified the discriminator dispatch against the historical commit's OWN workflow definition (job: rebless — the only regression-measuring option that existed in `.github/workflows/bench.yml` at that commit; `disk-control-github` postdates it), not today's bench.yml options, since `workflow_dispatch` runs the workflow file as it exists on the dispatched ref."

requirements-completed: []
# FIX-11 is NOT complete. See 'Plan Status' below — Task 2 (external checkpoint)
# and Task 3 (attribution + issue close) are outstanding, gated on human
# authorization of two outward-facing actions this dispatch was not permitted
# to perform.

coverage: []

duration: 35min
completed: 2026-09-15
status: halted
---

# Phase 1 Plan 5: GH #20 Follow-Up Decisions (Task 1 of 3) Summary

**Follow-up 1 (Namespace cache volume) closed won't-do on the record; follow-up 2's drift discriminator fully specified and reproducible from `tools/bench/BASELINE.md` alone — the measurement, attribution, and GH #20 closure remain outstanding pending human authorization of two external actions.**

## Plan Status: PARTIALLY COMPLETE — halted at Task 2's `checkpoint:human-action`

This plan carries `autonomous: false` and an `external_action_gate` that forbids
performing any outward-facing action (git push, `gh` writes, workflow dispatch)
without explicit per-action approval. Task 1 (fully local) is done and
committed. Task 2 is a blocking human-action checkpoint gating the one
remaining measurement; Task 3 (attribution + `gh issue close 20`) cannot run
until Task 2 is approved and its result is known.

**FIX-11 is NOT marked complete in REQUIREMENTS.md.** GH #20 is still OPEN.
`tools/bench/baseline.json` is byte-unchanged (verified via its committed
sha256, `0bcdc60b...8552c`).

## Performance

- **Duration:** ~35 min
- **Started:** 2026-09-15 (session start)
- **Completed:** 2026-09-15 (halted at checkpoint)
- **Tasks:** 1/3 completed (Task 2 halted awaiting human action; Task 3 not started)
- **Files modified:** 1

## Accomplishments

- Located and confirmed, via `git log -S'11279' -- tools/bench/baseline.json` and a `git show` read-back, the exact commit (`d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8`, 2026-07-31) where the old `11279.59` baseline was recorded.
- Computed the drift arithmetic from the two real figures either side of this plan (`11279.591291175333` -> `17090.87527197409` = **+51.5%**) and reconciled it against GH #20's own stated `+44.8%` (which anchors a different, earlier intermediate reading), recording both with an explanation rather than silently picking one.
- Recorded follow-up 1 (Namespace cache volume, 8x16 profile) as a **won't-do**, on the record, in `tools/bench/BASELINE.md`, with the adoption bar, the deciding facts, and the reason a cache volume cannot reach the leading unrefuted explanation (host-placement variance).
- Fully specified follow-up 2's discriminator dispatch — ref, job (`rebless`, the only regression-measuring option present in the historical commit's own `bench.yml`), flags, trial count, and runner class — so it is reproducible from `tools/bench/BASELINE.md` alone without re-deriving anything from the issue.
- Noted the staleness-check idea from GH #20 as a deferred capability, not part of this closure.

## Task Commits

Each task was committed atomically:

1. **Task 1: Locate the old-baseline commit and record follow-up 1's won't-do decision** - `7d5815e0` (docs)

**Task 2 (checkpoint:human-action):** halted, not executed — see "Checkpoint" below.
**Task 3:** not started — depends on Task 2's outcome.

**Plan metadata:** pending — see final commit note below (this SUMMARY + STATE.md/ROADMAP.md commit follows this document being written).

## Files Created/Modified

- `tools/bench/BASELINE.md` - New `## GH #20 follow-up decisions` section: follow-up 1's won't-do decision (complete) and follow-up 2's discriminator specification (complete; measured figure and attribution left as explicit PENDING markers for Task 3)

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

None - Task 1 executed exactly as written. No auto-fixes were needed; the task was read-only investigation (`git log`, `git show`) plus a documentation edit.

## Issues Encountered

None during Task 1. Task 2 was not attempted — its entire premise is that it requires human authorization for outward-facing actions, so there is nothing to report as an "issue" here; it is the plan's designed stopping point.

## CHECKPOINT REACHED

**Type:** human-action
**Gate:** blocking-human (Task 2's `gate` is implicit in its type; this plan's own `external_action_gate` additionally forbids these actions without further approval)
**Plan:** 01-05
**Progress:** 1/3 tasks complete

### Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Locate the old-baseline commit and record follow-up 1's won't-do decision | `7d5815e0` | `tools/bench/BASELINE.md` |

### Current Task

**Task 2:** Dispatch the one discriminator bench run
**Status:** blocked — awaiting explicit human authorization for two outward-facing actions
**Blocked by:** This dispatch's `external_action_gate` forbids `git push` (including the temporary ref) and any `gh` write command (including `gh workflow run` / `gh issue close`) without further explicit approval not present in this dispatch.

### Checkpoint Details

Everything automatable is done. Task 1 located the historical commit, computed
the drift arithmetic, and wrote the discriminator's exact parameters into
`tools/bench/BASELINE.md`. What remains requires a human to authorize and
perform two publishing actions against the remote:

**1. Push a temporary ref at the located historical commit:**
```
git push origin d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8:refs/heads/bench-gh20-discriminator
```

**2. Dispatch the regression bench against it, matching the parameters `tools/bench/BASELINE.md` now records exactly:**
```
gh workflow run bench.yml --ref bench-gh20-discriminator -f job=rebless -f trials=7
```
(At that historical commit, `.github/workflows/bench.yml`'s `workflow_dispatch`
only offers `job: headtohead | rebless | both` — `disk-control-github` did not
exist yet. `rebless` is the only option that runs `-mode regression`. It takes
no write token: the measured candidate never reaches this repository's
committed `tools/bench/baseline.json`, only an ephemeral checkout, a
downloadable artifact, and a job-summary delta table.)

**3. Wait for it and read the result:**
```
gh run watch
```
Read the reported `files_per_sec` from the job summary / step output, and the
runner label (`RUNNER_OS`/`ImageOS`) the run actually executed on.

**4. Delete the temporary ref:**
```
git push origin --delete bench-gh20-discriminator
```

**What each action changes that is visible to anyone outside this machine:**
- Step 1 creates a new branch `bench-gh20-discriminator` on `origin`, publicly visible in the repo's branch list and any watchers' notifications, pointing at an already-public historical commit (no new code content is exposed — the commit is already in history).
- Step 2 triggers a real GitHub Actions run, consuming CI minutes and appearing in the repo's Actions tab / run history permanently (run history is not deleted by deleting the ref).
- Step 4 removes the branch from the visible branch list, but the Actions run history from step 2 remains.
- The eventual `gh issue close 20 --comment "..."` (Task 3) posts a public comment and closes a public issue — visible to anyone watching the repo, and to anyone who receives issue-close notifications.

**Prepared GH #20 closing comment text** (for Task 3, to be posted only after Task 2's measurement is known and this checkpoint is separately approved):

> **Follow-up 1 — Namespace cache volume on the 8x16 profile: won't-do.** `ubuntu-latest` is free on this public repo and measured 28.6x headroom (0.35% session-to-session disagreement) against the Namespace 4x8 profile's 2.30x (4.36%) — the adoption bar this issue set was matching `ubuntu-latest`'s stability, not merely beating overlayfs. A cache volume only reaches the storage-latency hypothesis (Round 3), which the tmpfs experiment already refuted as the driver; it does not reach host-placement variance, the leading explanation left unrefuted (each Namespace session in this investigation ran on a different ephemeral VM instance). Full reasoning: [`tools/bench/BASELINE.md` § GH #20 follow-up decisions](../../../tools/bench/BASELINE.md).
>
> **Follow-up 2 — the baseline drift: [PENDING Task 3 — insert the measured `files_per_sec`, the runner label, and the fleet/code/unresolved attribution here once Task 2's run completes].** The old-baseline commit (`d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8`, 2026-07-31, `11279.591291175333` files/s) was re-measured on today's `ubuntu-latest` runner with the same `-seed 42 -count 120000`, same code as originally shipped, only the hardware/fleet held variable. Full numbers and reasoning: [`tools/bench/BASELINE.md` § GH #20 follow-up decisions](../../../tools/bench/BASELINE.md).
>
> Note: the "the gate needs a staleness check, not just a one-time refresh" observation in this issue is tracked separately as a deferred capability, not resolved by this closure.

**Anything not determinable locally that the discriminator run is supposed to answer:**
- The actual measured `files_per_sec` at the historical commit on today's hardware — this is the entire point of the run; it cannot be computed, only measured.
- Whether that figure lands close to today's `17090.88` (attribution: FLEET — the code did not get faster, the hardware/fleet did), close to the historical `11279.59` (attribution: CODE — a genuine speedup happened in the two-day window), or within the documented 10% `DefaultThroughputTolerance` noise band of both, in which case the correct call is UNRESOLVED rather than a forced pick.
- The exact runner label / fleet identity (`ImageOS`, instance ID) the dispatched run actually lands on, needed to fully corroborate or refute the fleet-hardware-change hypothesis GH #20 itself raised.
- Whether the historical commit's toolchain (`go.mod`'s Go version pin, `actions/setup-go` behavior at that commit) still builds cleanly on today's `setup-go` action version — this is a real risk the checkpoint instructions call out; it can only be discovered by running it.

### Awaiting

Explicit user approval to run the two commands in steps 1-2 above (and, after
the run completes, explicit approval to post the Task 3 closing comment and
close GH #20). Report back the run URL, the measured `files_per_sec`, and the
runner label once approved and run — that becomes Task 3's input.

## User Setup Required

None - no external service configuration required. (The outstanding work is
authorization to run existing CI infrastructure, not new setup.)

## Next Phase Readiness

Task 1's output is a stable, complete artifact: `tools/bench/BASELINE.md`'s
new section is fully specified and requires no further editing to make the
discriminator reproducible. The only blocker to closing FIX-11 is human
approval of the two external actions named above. Once approved and the
measurement is in hand, Task 3 is a short, mechanical follow-up (write the
attribution sentence, close the issue) with no remaining ambiguity.

## Self-Check: PASSED

- FOUND: `tools/bench/BASELINE.md`
- FOUND: `.planning/phases/01-defect-flake-burn-down/01-05-SUMMARY.md`
- FOUND: commit `7d5815e0` in `git log --oneline --all`

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15 (partial — halted at checkpoint)*
