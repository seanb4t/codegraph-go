---
phase: 01-defect-flake-burn-down
plan: 05
subsystem: infra
tags: [ci, bench, github-actions, process-decision]

# Dependency graph
requires: []
provides:
  - "GH #20 follow-up 1 (Namespace cache volume on 8x16) recorded as won't-do in `tools/bench/BASELINE.md`, with the bar it was measured against and the two facts that decide it"
  - "GH #20 follow-up 2 (baseline drift) attributed to FLEET in `tools/bench/BASELINE.md`, backed by a discriminator run of the old-baseline commit (d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8) on today's ubuntu-latest runner: 16569.160272289788 files/s measured, +46.90% same-code/different-hardware vs +3.15% same-hardware/different-code (inside DefaultThroughputTolerance's 10% budget)"
  - "The stale-gate consequence recorded explicitly: while stale, the 10% tolerance band could only fire below 10151.63 files/s against real ~16569.16 files/s — a ~38.7% regression would have passed green — named as a deferred gate capability, not fixed by this closure"
  - "GH #20 closed with both decisions in the closing comment"
  - "FIX-11 marked complete in REQUIREMENTS.md"
affects: [none — this is a documentation/process closure, not code]

# Actuals (#2632) — chars/4 over the realized diff, not a harness token count
# commits/plan_head_before are MEASURED via `git rev-list --count`, filtered
# to this plan's own commit-subject scope (01-05) — see "State Reconciliation"
# below for why the raw ledger range is NOT used as-is: two sibling plans
# (01-06, 01-07) committed on this shared branch between this plan's halt and
# resume, so the raw ${PLAN_HEAD_BEFORE}..HEAD range is 11, not 3.
actuals:
  tokens: 3070
  tasks: 3
  commits: 3
  plan_head_before: a86ef59e747c84bd081a80ee72a7cf0889f6e1ce

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
  - "Follow-up 2 attributed to FLEET, not code: three held-constant comparisons (July-vs-today same commit = +46.90%; today same-runner old-vs-new code = +3.15%; total = +51.52%). The +3.15% code contribution sits inside `DefaultThroughputTolerance`'s 10% budget, i.e. indistinguishable from noise, so essentially the entire drift is GitHub runner hardware — consistent with the AMD EPYC 9V74 part GH #20's own A/B already observed."
  - "Named the stale-gate consequence as a fact, not just an attribution: while the baseline sat stale at 11279.59, the 10% tolerance band could only fire below 10151.63 files/s against real performance of ~16569.16 files/s — a regression of up to ~38.7% would have passed the gate green for the duration the baseline stayed stale. Recorded as a deferred gate capability (a staleness check), explicitly NOT built by this closure — GH #20's own follow-up observation is confirmed with a number, not resolved with new code."
  - "The two external actions (pushing/deleting the temporary discriminator ref and dispatching the bench run; posting the GH #20 closing comment and closing the issue) were performed by the orchestrator after explicit user approval of the Task 2 checkpoint, outside this dispatch's own execution — not run directly by this executor session. Both are verified read-only post-hoc: `gh issue view 20` reports CLOSED, the closing comment carries both decisions, and `tools/bench/baseline.json`'s sha256 is unchanged."

requirements-completed: [FIX-11]

coverage:
  - deliverable: "GH #20 follow-up 1 (Namespace cache volume) recorded as won't-do"
    verification:
      - kind: command
        ref: "rg -A40 '^## GH #20 follow-up decisions' tools/bench/BASELINE.md | rg -i \"won't-do|wont-do|will not\""
        status: pass
    human_judgment: false
  - deliverable: "GH #20 follow-up 2 (baseline drift) attributed to FLEET with the discriminator measurement recorded"
    verification:
      - kind: command
        ref: "rg -A80 '^## GH #20 follow-up decisions' tools/bench/BASELINE.md | rg -i 'fleet|code change|unresolved|unmeasured'"
        status: pass
    human_judgment: false
  - deliverable: "GH #20 closed with both decisions in the closing comment"
    verification:
      - kind: command
        ref: "gh issue view 20 --json state --jq '.state' | rg '^CLOSED$'"
        status: pass
      - kind: command
        ref: "gh issue view 20 --json comments --jq '.comments[-1].body' | rg -i 'cache volume|namespace'"
        status: pass
    human_judgment: false
  - deliverable: "tools/bench/baseline.json byte-unchanged across the whole plan"
    verification:
      - kind: command
        ref: "shasum -a 256 -c against committed sha 0bcdc60b...8552c"
        status: pass
    human_judgment: false
  - deliverable: "FIX-11 marked complete in REQUIREMENTS.md"
    verification:
      - kind: command
        ref: "rg 'FIX-11' .planning/REQUIREMENTS.md → '[x] FIX-11' and '| FIX-11 | Phase 1 | Complete |'"
        status: pass
    human_judgment: false

duration: ~40 min (35 min Task 1 session + ~5 min resume session for Task 3, excluding external-action wait time)
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 5: GH #20 Follow-Up Decisions Summary

**Both GH #20 perf-gate follow-ups closed on the record: cache volume adoption won't-do, baseline drift attributed to FLEET (+46.90% hardware vs +3.15% code, the latter inside the gate's own noise tolerance) — with the stale-gate consequence (a ~38.7% regression would have passed green) named explicitly as a deferred capability, not fixed here.**

## Plan Status: COMPLETE — resumed after Task 2's `checkpoint:human-action` was approved and performed

This plan carries `autonomous: false` and an `external_action_gate` that forbids
performing any outward-facing action (git push, `gh` writes, workflow dispatch)
without explicit per-action approval. Task 1 (fully local) ran and committed in
the first session. Task 2's two external actions — pushing/dispatching/deleting
the temporary discriminator ref, and posting the GH #20 closing comment and
closing the issue — were approved by the user and **performed by the
orchestrator outside this executor's own dispatch** (see "External Actions"
below for the run URL, comment URL, and the read-only verification this
resumed session ran against them). Task 3 (attribution + close) then ran and
committed in this resumed session.

**FIX-11 is now marked complete in REQUIREMENTS.md.** GH #20 is CLOSED.
`tools/bench/baseline.json` remains byte-unchanged across the whole plan
(verified via its committed sha256, `0bcdc60b...8552c`).

## Performance

- **Duration:** ~40 min total (35 min Task 1 session + ~5 min resume session for Task 3; excludes external-action wait time performed outside this dispatch)
- **Started:** 2026-09-15 (session start)
- **Completed:** 2026-09-15 (plan complete)
- **Tasks:** 3/3 completed (Task 1 and Task 3 committed by this executor; Task 2's external actions approved and performed by the orchestrator, verified read-only in this resumed session)
- **Files modified:** 1

## Accomplishments

- Located and confirmed, via `git log -S'11279' -- tools/bench/baseline.json` and a `git show` read-back, the exact commit (`d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8`, 2026-07-31) where the old `11279.59` baseline was recorded.
- Computed the drift arithmetic from the two real figures either side of this plan (`11279.591291175333` -> `17090.87527197409` = **+51.5%**) and reconciled it against GH #20's own stated `+44.8%` (which anchors a different, earlier intermediate reading), recording both with an explanation rather than silently picking one.
- Recorded follow-up 1 (Namespace cache volume, 8x16 profile) as a **won't-do**, on the record, in `tools/bench/BASELINE.md`, with the adoption bar, the deciding facts, and the reason a cache volume cannot reach the leading unrefuted explanation (host-placement variance).
- Fully specified follow-up 2's discriminator dispatch — ref, job (`rebless`, the only regression-measuring option present in the historical commit's own `bench.yml`), flags, trial count, and runner class — so it is reproducible from `tools/bench/BASELINE.md` alone without re-deriving anything from the issue.
- **External Actions (Task 2, approved and performed by the orchestrator):** a temporary ref `bench-gh20-discriminator` was pushed at `d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8`, `bench.yml` was dispatched with `job=rebless trials=7`, and the run succeeded — [run 34980422924](https://github.com/seanb4t/codegraph-go/actions/runs/34980422924), job "record candidate perf baseline (PERF-02, manual only)", 2026-09-15T14:15:41Z → 14:30:10Z, on `ubuntu-latest`. The `baseline-candidate` artifact reported `files_per_sec: 16569.160272289788`. The temporary ref was deleted afterward.
- Attributed follow-up 2's drift to **FLEET**: +46.90% same-code/different-hardware (11279.59 → 16569.16) vs +3.15% same-hardware/different-code (16569.16 → 17090.88, the committed baseline) — the code contribution sits inside `DefaultThroughputTolerance`'s 10% budget, indistinguishable from noise.
- Recorded the stale-gate consequence explicitly, as the durable lesson the closure does NOT resolve: while stale at `11279.59`, the gate's 10% tolerance could only fire below `10151.63` files/s against real performance of `~16569.16` — a regression of up to **~38.7%** would have passed the gate green. Named as a deferred gate capability (a staleness check), not fixed by this closure.
- Noted the staleness-check idea from GH #20 as a deferred capability, not part of this closure.
- **External Actions (Task 3, approved and performed by the orchestrator):** GH #20 closed with a comment carrying both decisions, cache volume first, at [issue comment 5682423921](https://github.com/seanb4t/codegraph-go/issues/20#issuecomment-5682423921), 2026-09-15T14:52:35Z, `state=CLOSED` (verified).
- Marked **FIX-11 complete** in `.planning/REQUIREMENTS.md` via `gsd-tools requirements mark-complete`.

## Task Commits

Each locally-executed task was committed atomically:

1. **Task 1: Locate the old-baseline commit and record follow-up 1's won't-do decision** - `7d5815e0` (docs)
2. **Task 2: Dispatch the one discriminator bench run** - external action, no local commit (see "External Actions" above; the halt-documentation from the first session's pause was committed separately as `81e95b65`)
3. **Task 3: Attribute the drift, then close GH #20 with both decisions** - `177d9fce` (docs; local BASELINE.md edit) + external `gh issue close 20` (no local commit — GH state, not repo state)

**Plan metadata:** this SUMMARY + STATE.md/ROADMAP.md/REQUIREMENTS.md commit follows this document being written.

## Files Created/Modified

- `tools/bench/BASELINE.md` - `## GH #20 follow-up decisions` section: follow-up 1's won't-do decision, follow-up 2's discriminator specification, the measured figure (`16569.160272289788` files/s), the three-way fleet/code attribution table, the FLEET verdict, and the stale-gate consequence paragraph (~38.7% regression window)

## Decisions Made

See `key-decisions` in frontmatter above.

## Deviations from Plan

None - both locally-executed tasks (1 and 3) ran exactly as written. No auto-fixes were needed; both were investigation/documentation-only work (`git log`, `git show`, `gh issue view`, a `tools/bench/BASELINE.md` edit). Task 2's external actions were performed by the orchestrator per the checkpoint-resolution protocol, not by this executor directly — this is the designed flow for a `checkpoint:human-action` task, not a deviation.

## Issues Encountered

None. The one known tooling gap — `state.record-session`/`sync` counting a halted plan's SUMMARY.md as a completed plan before this resume — is reported under "State Reconciliation" below rather than as a plan-execution issue, since it did not block or corrupt any work in this plan.

## Checkpoint Resolution (Task 2)

The `checkpoint:human-action` this plan halted at in the prior session was resolved by the user approving both external actions named in that checkpoint. Both were performed by the orchestrator, outside this executor's own dispatch, per the `<checkpoint_resolution>` supplied to this resumed session:

- **Discriminator run:** pushed `bench-gh20-discriminator` at `d4672cf5c72a1e83f56e09181ae5b4a7fa3e1ba8`, dispatched `bench.yml` with `job=rebless trials=7`, run succeeded ([34980422924](https://github.com/seanb4t/codegraph-go/actions/runs/34980422924)), temp branch deleted afterward. Verified read-only in this session: `tools/bench/baseline.json`'s committed sha256 is unchanged (`0bcdc60b...8552c`).
- **GH #20 closure:** comment posted ([5682423921](https://github.com/seanb4t/codegraph-go/issues/20#issuecomment-5682423921)) carrying both decisions in issue order (cache volume first, drift second) plus the explicit staleness-not-resolved note; issue closed. Verified read-only in this session: `gh issue view 20 --json state` reports `CLOSED`, and the closing comment mentions both `cache volume` and `namespace`.

This executor did not run any `git push`, workflow dispatch, or writing `gh` command in this resumed session — only read-only `gh` verification and the local `tools/bench/BASELINE.md` edit + commit for Task 3.

## State Reconciliation

Before this plan halted, `.planning/STATE.md` showed `completed_plans: 5` while
only 4 plans (01 through 04) genuinely carried `status: complete`
SUMMARY.md files — the halted 01-05 SUMMARY.md (with `status: halted`) was
being counted by the state tooling as complete. That gap persisted through
two subsequent plans (01-06, 01-07) executing in between this plan's halt
and resume, so at the start of this resumed session `completed_plans: 7`
while only 6 SUMMARY.md files on disk actually carried `status: complete`
(01-01, 02, 03, 04, 06, 07 — 01-05 still `halted`).

Now that this SUMMARY.md is flipped to `status: complete`, re-running
`state.update-progress` (see below) recalculates the count from disk and
should land on the now-correct `7` — which happens to numerically match the
stale pre-resume value, but for the right reason this time (a genuine 7th
complete SUMMARY.md now exists, rather than a halted one being
miscounted). This is coincidental convergence, not evidence the counting
bug is fixed: the underlying tool still appears to count SUMMARY.md file
*existence* rather than gating on the `status:` field, which will
misreport again the next time a plan halts before this resume flow closes
it. Recording the gap here rather than hand-editing `STATE.md` (per the
project's planning-artifacts rule — STATE.md is tool-owned).

## Threat Flags

None — this plan's threat model (T-01-05-01 through T-01-05-SC) was already
fully addressed by the plan's own design (blocking-human checkpoint, ref
deletion, prohibition on reblessing `baseline.json`, CI-skip grep) rather
than something this SUMMARY needs to flag as new surface.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

FIX-11 is complete. Both GH #20 follow-ups have recorded decisions in
`tools/bench/BASELINE.md` and the issue is closed with both decisions in the
closing comment. `tools/bench/baseline.json` remains byte-unchanged across
the whole plan. No further work is needed on this plan; ready for
`/gsd-verify-work` or the next plan in this phase (01-08).

## Self-Check: PASSED

- FOUND: `tools/bench/BASELINE.md`
- FOUND: `.planning/phases/01-defect-flake-burn-down/01-05-SUMMARY.md`
- FOUND: commit `7d5815e0` in `git log --oneline --all`
- FOUND: commit `177d9fce` in `git log --oneline --all`
- FOUND: `## GH #20 follow-up decisions` section with the FLEET attribution and stale-gate consequence in `tools/bench/BASELINE.md`
- VERIFIED: `gh issue view 20 --json state` reports `CLOSED`
- VERIFIED: `tools/bench/baseline.json` sha256 unchanged (`0bcdc60bcf8ef1e782210473d362ac1f675f02d593ac0ad8e0340d5954e8552c`)
- VERIFIED: `.planning/REQUIREMENTS.md` shows `[x] FIX-11` and `| FIX-11 | Phase 1 | Complete |`

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
