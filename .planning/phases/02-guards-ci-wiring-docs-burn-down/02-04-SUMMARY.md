---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 04
subsystem: ci-guards
tags: [github-ruleset, branch-protection, ci-guards, tmux-e2e, goreleaser, gsd-tools]

# Dependency graph
requires:
  - phase: 02-guards-ci-wiring-docs-burn-down
    provides: "02-03: .github/required-status-checks.txt (7-entry fixture), scripts/check-ruleset-drift.sh, the wired 'Ruleset drift check (GRD-12)' ci.yml step, honestly RED against the live 6-vs-7 divergence"
provides:
  - "GRD-12 fully closed: the live protect-main ruleset (20157557) and the shared fixture agree at 8 required-status-check contexts, verified by re-running the real drift script and Go test (never asserted from a hand-narrated result)"
  - "02-04-ruleset-put-body.json: a committed, maintainer-reviewable PUT body (name/target/enforcement/bypass_actors/conditions/all five rule types preserved, exactly two D-08 contexts appended) built from an authenticated gh api read"
  - "The Phase 08-03 user_setup blocker that sat in STATE.md Blockers since v0.13.0 is resolved through gsd-tools state resolve-blocker"
  - "Family (c) in 02-MUTATION-LOG.md carries both the pre-flip RED (6 vs 7, exit 1) and post-flip GREEN (8 vs 8, PASS, exit 0) transcripts from the real script and Go test"
affects: []

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 2860
  tasks: 2
  commits: 2
  plan_head_before: 6448b60f6e753a157b40c0f696a96c9009afd533

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Maintainer-reviewable settings-change package: an authenticated read builds the exact PUT body, the executor never applies it unprompted, and a read-only precondition re-check gates the dependent work item — the same pattern generalizes to any repository-settings change an agent must not make silently"
    - "Precondition halts as blocking-human, never auto-approved, even under auto-mode: a missing/unmet fact about the external world (here, GitHub's live ruleset state) is not something an executor can verify itself into satisfying"

key-files:
  created:
    - .planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-ruleset-put-body.json
  modified:
    - .github/required-status-checks.txt
    - .planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md
    - .planning/STATE.md

key-decisions:
  - "Task 2 did not start on the orchestrator's word alone — the continuation agent independently re-ran the read-only curl precondition check itself (8 contexts, enforcement active, both D-08 strings present) before touching the fixture, per the plan's own prohibition against a 'the fixture leads, live follows' shortcut"
  - "The maintainer authorized running Task 1's prepared gh api --method PUT command rather than applying the change through the GitHub UI; both paths were offered in Task 1's package and the choice was the maintainer's, recorded in Family (c)"
  - "GRD-12's requirements-completed stays empty in this SUMMARY only if a sibling plan is not yet closed — all three siblings (02-01, 02-02, 02-03) already carry SUMMARY.md, so this plan is the last declarer and GRD-12 is marked complete via requirements.ready-ids in the metadata step"

patterns-established:
  - "RED-then-GREEN for an external-state divergence (no code mutation, no test mutation) is recorded the same way as a code-level mutation test: a Pre-mutation gate, an Observed failure with exit code, and — once the world changes — a matching Green re-run with exit code, so the guard's discriminating power is on the record for both directions"

requirements-completed: [GRD-12]

coverage:
  - id: D1
    description: "The live protect-main ruleset (20157557) and .github/required-status-checks.txt agree at exactly 8 contexts in both directions"
    requirement: GRD-12
    verification:
      - kind: other
        ref: "bash scripts/check-ruleset-drift.sh — run this session"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestRequiredCheckNamesPreserved"
        status: pass
    human_judgment: false
  - id: D2
    description: "02-04-ruleset-put-body.json is a valid, complete PUT body carrying exactly the two D-08 contexts on top of the live six, with bypass_actors and all five rule types preserved"
    requirement: GRD-12
    verification:
      - kind: other
        ref: "Task 1 <verify> jq -e assertion (enforcement active, name protect-main, 8 contexts, 5 rule types, bypass_actors present, no id field) — automated, run in the prior session and re-confirmed unchanged this session"
        status: pass
    human_judgment: false
  - id: D3
    description: "The maintainer applied D-08 (the ruleset change) via the prepared gh api command, never the executor unprompted; the executor verified the live state read-only both before and after"
    requirement: GRD-12
    verification: []
    human_judgment: true
    rationale: "Whether the maintainer's authorization was genuinely informed (not just a mechanical approval) is a judgment call outside what an automated check can assert — the checkpoint transcript and Family (c)'s record are the evidence a human reviewer should read."
  - id: D4
    description: "STATE.md's [Phase 08-03] user_setup blocker is resolved through the gsd-tools tool verb, not a hand edit, and no other Blockers bullet was disturbed"
    requirement: GRD-12
    verification:
      - kind: other
        ref: "gsd-tools query state.resolve-blocker --text '[Phase 08-03] user_setup NOT completed' → {resolved: true}; git diff -- .planning/STATE.md shows exactly one bullet removed"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-09-16
status: complete
---

# Phase 02 Plan 04: D-08 Ruleset Update and GRD-12 Closure Summary

**Grew the shared required-status-check fixture from 7 to 8 entries only after independently re-verifying (read-only) that the maintainer's protect-main ruleset change had actually landed, turning the GRD-12 drift guard from a recorded RED into a recorded GREEN.**

## Performance

- **Duration:** 12 min (continuation segment; Task 1 ran in a prior session)
- **Started:** 2026-09-16T11:20:00Z (continuation resume)
- **Completed:** 2026-09-16T11:32:00Z
- **Tasks:** 2/2 (Task 1 complete before this continuation; Task 2 completed in this session)
- **Files modified:** 3 (this continuation) + 2 created (prior session)

## Accomplishments

- Re-verified, read-only, that the maintainer's D-08 ruleset change had landed: `curl` against `rulesets/20157557` showed `enforcement: active` and exactly 8 required contexts, including both `goreleaser check (config validation, DIST-01)` and `tmux e2e (real-pty harness, TTY-01..TTY-07)` — confirmed independently rather than trusting the orchestrator's report alone.
- Grew `.github/required-status-checks.txt` to 8 non-blank, newline-terminated lines (appended the tmux context), only after the precondition confirmed the live set.
- `bash scripts/check-ruleset-drift.sh` now prints `live has 8 contexts, fixture has 8 contexts`, `self-check PASS`, `ruleset-drift: PASS`, exit 0.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -count=1 -run 'TestRequiredCheckNamesPreserved$' -v` passes and logs `read 8 required contexts`.
- Resolved the standing `[Phase 08-03] user_setup NOT completed` blocker in `.planning/STATE.md` via `gsd-tools query state.resolve-blocker` (tool verb, not a hand edit); confirmed via `git diff` that exactly one bullet was removed and the Blockers/Concerns section survived intact.
- Recorded the GREEN transcript (both the drift script and the Go test, verbatim) under Family (c)'s `Green re-run (after D-08)` heading in `02-MUTATION-LOG.md`, alongside the pre-existing pre-flip RED transcript from Task 1.

## Task Commits

Each task was committed atomically:

1. **Task 1: Re-capture the RED, then build the maintainer's one-glance ruleset update package** - `6025f848` (docs) — completed in the prior session before the `blocking-human` precondition checkpoint.
2. **Task 2: After D-08 lands — grow the fixture to 8, watch the step go GREEN, resolve the STATE blocker** - `8e4b2aa` (ci) — completed in this continuation session, after independently re-verifying the precondition.

**Plan metadata:** commit to follow (docs: complete plan).

## Files Created/Modified

- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-ruleset-put-body.json` - the maintainer-reviewable PUT body (created Task 1)
- `.github/required-status-checks.txt` - grown from 7 to 8 lines (appended `tmux e2e (real-pty harness, TTY-01..TTY-07)`)
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` - Family (c) now carries both the RED and GREEN transcripts
- `.planning/STATE.md` - `[Phase 08-03] user_setup NOT completed` blocker removed via tool verb

## Decisions Made

- The continuation agent did not proceed on the orchestrator's word that D-08 had landed — it independently re-ran the read-only `curl` precondition check itself before touching the fixture, per the plan's precondition and the executor's own "never auto-approve a precondition" rule.
- The maintainer chose to authorize the prepared `gh api --method PUT` command (rather than the GitHub UI path); both options were presented in Task 1's package, and the choice and its outcome are recorded verbatim in Family (c).
- GRD-12 is marked complete in this plan's `requirements-completed` because all three sibling plans that also declare it (02-01, 02-02, 02-03) already carry `*-SUMMARY.md` — this plan is the last declarer, so `requirements.ready-ids` reports it ready.

## Deviations from Plan

None - plan executed exactly as written. The precondition halt in Task 2 and its resolution via a human-authorized `gh api` run were the plan's own designed `checkpoint:human-verify` / `blocking-human` gate, not a deviation.

## Issues Encountered

None. The precondition re-check, fixture edit, drift script, Go test, and blocker resolution all passed on the first attempt.

## User Setup Required

None remaining. The single `user_setup` item declared in this plan's frontmatter (D-08: the maintainer adding the two required-status-check contexts to ruleset 20157557) was satisfied before Task 2 began — verified read-only, independently, by this continuation agent.

## TDD Note

This plan is `type: execute` (a settings hand-off plus a one-line data-file flip), not `type: tdd`; no RED-commit gate applies. The RED→GREEN pattern recorded in `02-MUTATION-LOG.md` Family (c) is a mutation-log entry, not a TDD cycle — the "mutation" is external ruleset state, not a code or test change, and there is no `test(...)` → `feat(...)` → `refactor(...)` commit sequence to report.

## Next

Phase 02 plans 02-01 through 02-04 are all complete. GRD-12 is fully closed. Ready for the remaining phase 02 plans (02-05 through 02-07 already have summaries per phase state) or `/gsd-verify-work 02` once all phase 02 plans have summaries.

## Self-Check: PASSED

- FOUND: `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-SUMMARY.md`
- FOUND: `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-04-ruleset-put-body.json`
- FOUND commit `6025f848` (Task 1, docs)
- FOUND commit `8e4b2aa` (Task 2, ci)
