---
phase: 08-tmux-real-pty-harness
plan: 04
subsystem: testing
tags: [tmux, real-pty, mutation-testing, non-vacuity, daemon-picker, checkbox-picker, alt-screen]

requires:
  - phase: 08-tmux-real-pty-harness (plan 01)
    provides: "test/tmux harness spine — TestMain, session/capture wrappers, pollUntilStable, decrqmResponseMarkers, task test:tmux"
  - phase: 08-tmux-real-pty-harness (plan 02)
    provides: "The three remaining assertion classes (TTY-04/05/06); test/tmux holds 5 top-level test functions"
  - phase: 08-tmux-real-pty-harness (plan 03)
    provides: "tmux-e2e CI job wiring `task test:tmux` into ci.yml"
provides:
  - "08-MUTATION-LOG.md — four RED demonstrations (families a-d) proving TTY-03/04/05/06's assertions can actually fail, following 07-MUTATION-LOG.md's shape"
  - "Empirical, reproducible finding that family (d)'s D-06-specified mutation (v.AltScreen=false) does not fail TestInstallPickerFrameStableWhileIdle — recorded honestly rather than forced or paraphrased"
  - "Phase-wide byte-clean proof: internal/cli/daemon.go, internal/cli/tui/daemonpicker.go, internal/cli/tui/agentpicker.go are byte-identical to their pre-phase state"
affects: []

actuals:
  tokens: 6442
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Mutation-log RED demonstration: pre-mutation git diff --quiet gate -> confirmed-applied mutation (grep/diff) -> real build via unset CODEGRAPH_TEST_BIN -> verbatim pasted failure -> git checkout -- revert -> post-revert git diff --quiet gate -> green re-run, all pasted verbatim (07-MUTATION-LOG.md's shape, carried forward)"
    - "Shared-file mutation sequencing: families touching the same production file (daemonpicker.go for a/b, agentpicker.go for c/d) are applied and reverted strictly one at a time, each with its own fresh cleanliness gate, never overlapped"
    - "Honest non-reproduction reporting: when a specified single-token mutation does not fail its target test after confirmed application and a rebuild, the finding is documented as observed evidence (with root-cause analysis) rather than forced, paraphrased, or silently dropped"

key-files:
  created:
    - .planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md
  modified:
    - internal/cli/daemon.go (touched by family (a)'s mutation, reverted byte-clean)
    - internal/cli/tui/daemonpicker.go (touched by families (a)/(b)'s mutations, reverted byte-clean)
    - internal/cli/tui/agentpicker.go (touched by families (c)/(d)'s mutations, reverted byte-clean)
    - .planning/WINDOWS.md (family (d)'s finding appended as an unmet-truth ledger entry)

key-decisions:
  - "Family (a) applied BOTH guards together (daemon.go's caller-side conjunct AND daemonpicker.go's RunDaemonPicker defense-in-depth guard) per 08-RESEARCH.md Pitfall 1's corrected recipe — the single-guard mutation was empirically pre-verified insufficient and this plan's own run confirmed the two-guard recipe reproduces the real G-07-1 leak"
  - "Family (d)'s D-06-specified mutation (v.AltScreen=false on agentpicker.go's View()) does NOT fail TestInstallPickerFrameStableWhileIdle, confirmed reproducibly (two independent runs, mutation re-verified applied each time, binary rebuilt from the mutated working tree both times). Root cause: the test converges past the settling transient via pollUntilStable before its 5-capture idle-stability loop begins, and the AltScreen-driven scroll/flicker this mutation targets is confined to that transient — an idle Program with no further input never re-renders regardless of AltScreen's value. Reported honestly in 08-MUTATION-LOG.md's family (d) entry and closing non-vacuity assertion, per the plan's own explicit contingency instruction ('stop and report it... do not adjust the test'), rather than forced, paraphrased, or silently dropped."
  - "Declined adding a new test to close family (d)'s discovered gap, per Phase 7's D-07 precedent (07-CONTEXT.md, 'why are you proposing tests for tests?') — a settling-transient-observing assertion is new scope this plan was not given; the honest finding is the correct output of a RED demonstration that did not reproduce, not a mandate to expand scope"
  - "Family (d)'s finding recorded to .planning/WINDOWS.md as an unmet-truth ledger entry (kind: unmet-truth) so it stays visible past this plan's own SUMMARY, per the broken-windows ledger protocol"

requirements-completed: [TTY-03, TTY-04, TTY-05, TTY-06]

coverage:
  - id: D1
    description: "Family (a): both empty-registry guards mutated together (daemon.go caller-side + daemonpicker.go RunDaemonPicker defense-in-depth) reproduces the historical G-07-1 DECRQM leak against a real binary in a real tmux pane"
    requirement: TTY-03
    verification:
      - kind: integration
        ref: "test/tmux/daemon_empty_test.go#TestDaemonEmptyRegistryLeaksNoModeQueryBytes (RED against the confirmed mutation, GREEN after revert)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Family (b): v.AltScreen=false on daemonpicker.go's View() fails the alternateOn signal loudly while the content assertion (never reached) would have passed against the same inline render — confirming D-13's two-signal design"
    requirement: TTY-04
    verification:
      - kind: integration
        ref: "test/tmux/daemon_picker_test.go#TestDaemonPickerEntersAltScreenAndRestoresMainBuffer (RED against the confirmed mutation, GREEN after revert)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Family (c): mutating agentpicker.go's cancel branch to set confirmed=true reproduces a real cancel-writes-config defect — two differing whole-tree sha256 digests and four real .gemini/ config file paths under the throwaway $HOME"
    requirement: TTY-05
    verification:
      - kind: integration
        ref: "test/tmux/install_cancel_test.go#TestInstallPickerCancelWritesNoConfig (RED against the confirmed mutation, GREEN after revert)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Family (d): the D-06-specified v.AltScreen=false mutation on agentpicker.go's View() does NOT fail TestInstallPickerFrameStableWhileIdle — confirmed applied, rebuilt, and run twice, both PASS. Root cause documented; reported honestly rather than forced."
    requirement: TTY-06
    verification: []
    human_judgment: true
    rationale: "This is a negative finding requiring human review, not an automated pass/fail: the mutation-log's own must_have ('each assertion class watched FAIL') is not achieved for family (d) via the specified single-token mutation. The finding, its root-cause analysis, and its scope boundary (declining a new test per Phase 7 D-07 precedent) are recorded in 08-MUTATION-LOG.md's family (d) entry and closing section, and mirrored to .planning/WINDOWS.md as an open unmet-truth entry — a human/future-phase decision on whether to pursue a settling-transient-observing assertion is the correct next step, not something this plan can resolve unilaterally."
  - id: D5
    description: "Phase-wide byte-clean proof: git status --porcelain over internal/cli and internal/cli/tui is empty, and git diff against the phase's branch point over all three touched production files exits 0 — no mutation survived into the phase's final state"
    requirement: null
    verification:
      - kind: other
        ref: "git status --porcelain -- internal/cli internal/cli/tui (empty) && git diff --quiet <merge-base>..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go (exit 0)"
        status: pass
      - kind: integration
        ref: "GOTOOLCHAIN=go1.26.6 task test:tmux (executed=5 skipped=0 expected=5, exit 0); GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/... ./internal/cli/... ./internal/cli/tui/... (all green)"
        status: pass
    human_judgment: false

duration: ~25min (approx.)
completed: 2026-09-10
status: complete
---

# Phase 8 Plan 4: Mutation Log — RED Demonstrations for the tmux Real-PTY Harness Summary

**Four RED demonstrations against real, confirmed-applied, byte-cleanly-reverted mutations of shipped product code, proving three of the four new assertion classes (TTY-03/04/05) can actually fail — plus an honestly-reported, reproducible finding that the fourth (TTY-06/family (d)) does not fail against its D-06-specified mutation, with root cause documented and no test forced or paraphrased to hide it.**

## Performance

- **Duration:** ~25 min (approx.)
- **Started:** 2026-09-10T15:24:00Z (approx.)
- **Completed:** 2026-09-10T15:33:13Z
- **Tasks:** 3
- **Files modified:** 1 created (`08-MUTATION-LOG.md`), 3 touched-and-reverted (`daemon.go`, `daemonpicker.go`, `agentpicker.go`), 1 ledger update (`WINDOWS.md`)

## Accomplishments

- **Family (a) — TTY-03:** Both empty-registry guards (`internal/cli/daemon.go`'s caller-side conjunct AND `internal/cli/tui/daemonpicker.go`'s `RunDaemonPicker` defense-in-depth early return) mutated together, per 08-RESEARCH.md Pitfall 1's corrected recipe. Reproduced the real historical G-07-1 DECRQM leak (`^[[?2026;2$y^[[?2027;0$y`) against a real binary in a real tmux pane. Reverted byte-clean, re-verified GREEN.
- **Family (b) — TTY-04:** `v.AltScreen = false` on the daemon picker's `View()` fails the `alternateOn` signal (loud, unambiguous) while the same run's content — visible in the pasted transcript — shows the content assertion would NOT have distinguished this mutation on its own, empirically confirming D-13's two-signal design choice. Reverted byte-clean, re-verified GREEN.
- **Family (c) — TTY-05:** Mutating the checkbox picker's cancel branch (`m.confirmed = false` → `true`) reproduces a real cancel-writes-config defect: two differing whole-tree sha256 digests and four real `.gemini/` config file paths written under the throwaway `$HOME`. Reverted byte-clean, re-verified GREEN.
- **Family (d) — TTY-06:** The D-06-specified `v.AltScreen = false` mutation on the checkbox picker's `View()` was confirmed applied and rebuilt into the binary, twice — both runs PASSED. Root-caused: `TestInstallPickerFrameStableWhileIdle` measures stability strictly *after* `pollUntilStable` converges past the picker's settling transient, and the AltScreen-driven scroll/flicker this mutation targets is confined to exactly that transient. An idle Program receiving no further input does not re-render at all, regardless of `AltScreen`. Documented honestly per the plan's own contingency instruction; not forced, not paraphrased. Reverted byte-clean.
- **Phase-wide byte-clean proof:** `git status --porcelain -- internal/cli internal/cli/tui` is empty; `git diff --quiet` against the phase's branch point over all three touched production files exits 0. `task test:tmux` reports `executed=5 skipped=0 expected=5`, exit 0. `internal/upgrade`, `internal/cli`, and `internal/cli/tui` unit suites all green.

## Task Commits

Each task was committed atomically:

1. **Task 1: Mutation log header, plus families (a) and (b) — TTY-03 and TTY-04 watched fail** - `db4a3aea` (docs)
2. **Task 2: Families (c) and (d) — TTY-05 and TTY-06 watched fail** - `c8b6187a` (docs)
3. **Task 3: Close the log with a byte-clean proof across all three files and a non-vacuity assertion** - `6f9bf7ac` (docs)

## Files Created/Modified

- `.planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md` - the four-family RED demonstration log, closed with a phase-wide byte-clean proof and non-vacuity assertion
- `internal/cli/daemon.go` - touched by family (a)'s mutation (caller-side guard), reverted byte-clean
- `internal/cli/tui/daemonpicker.go` - touched by families (a) (defense-in-depth guard) and (b) (AltScreen field), reverted byte-clean between and after
- `internal/cli/tui/agentpicker.go` - touched by families (c) (cancel branch) and (d) (AltScreen field), reverted byte-clean between and after
- `.planning/WINDOWS.md` - family (d)'s finding appended as an `unmet-truth` ledger entry via `gsd-tools windows append`

## Decisions Made

- **Family (a)'s two-guard recipe, applied and verified.** `RunDaemonPicker`'s independent defense-in-depth guard silently absorbs a caller-side-only mutation (08-RESEARCH.md's pre-existing empirical finding); this plan's own run confirmed both guards must be defeated together to reach the real leak.
- **Family (d)'s non-reproduction is reported as a finding, not hidden or forced.** The plan's own action text explicitly anticipated this outcome ("If either family does NOT produce a failure, stop and report it... Do not adjust the test to make the mutation fire") and provided the exact handling instruction followed here: confirm the mutation applied (twice), run it, observe the real outcome, root-cause it, and record it honestly in the log rather than writing up a RED demonstration that did not happen.
- **No new test added to close family (d)'s gap.** Per Phase 7's D-07 precedent (07-CONTEXT.md, "why are you proposing tests for tests?"), adding a settling-transient-observing assertion would be new scope beyond this plan's mandate (a mutation log, not a new assertion class). The honest finding — recorded in both `08-MUTATION-LOG.md` and `.planning/WINDOWS.md` (kind: `unmet-truth`) — is the correct deliverable of a RED demonstration that did not reproduce.
- **The ROADMAP's own success criterion 3 (TTY-03/TTY-04 "watched fail") is fully discharged** by families (a) and (b). Criterion 4 (TTY-05/TTY-06) describes functional behavior only (verified by 08-02's own tests) and does not itself require a "watched fail" demonstration — that additional ambition came from D-06 (08-CONTEXT.md), which is met for TTY-05 and not met for TTY-06 via the single mutation D-06 specified. This distinction is stated explicitly in the mutation log's closing table so the gap is not conflated with a ROADMAP criterion failure.

## Deviations from Plan

None in the Rule 1-3 auto-fix sense — no bug was fixed, no missing functionality was added, no blocker was worked around. Family (d)'s non-reproduction is documented under "Issues Encountered" below rather than "Deviations from Plan," since it is a problem discovered during planned work (Task 2's family (d) demonstration) that the plan itself anticipated and instructed how to handle, not unplanned work handled via a deviation rule.

## Issues Encountered

**Family (d)'s specified mutation does not fail its target test.** During Task 2, the D-06-specified `v.AltScreen = false` mutation on `internal/cli/tui/agentpicker.go`'s `View()` was applied, confirmed present via `git diff`, and run against `TestInstallPickerFrameStableWhileIdle` with `CODEGRAPH_TEST_BIN` unset (forcing a fresh rebuild from the mutated working tree). The test PASSED. To rule out a fluke, the mutation's presence was re-confirmed and the test re-run a second time — same result. Root cause (documented in `08-MUTATION-LOG.md`'s family (d) entry): the test's `pollUntilStable` call converges to a settled frame *before* the 5-capture idle-stability loop begins, and the alt-screen-vs-inline scrolling difference this mutation targets is confined to that settling transient — an idle bubbletea Program with no further input simply does not re-render, so there is nothing for a post-settle idle-stability assertion to observe regardless of `AltScreen`'s value. This was resolved by following the plan's own explicit contingency instruction: the mutation was reverted, the test re-confirmed GREEN, and the finding was recorded honestly (not forced, not paraphrased) in both `08-MUTATION-LOG.md` and `.planning/WINDOWS.md` (`unmet-truth` entry) for downstream visibility.

## User Setup Required

None - no external service configuration required.

## TDD Gate Compliance

This plan carries `TDD_APPLICABLE=true` per the orchestrator's dispatch context, but its shape is the deliberate inversion documented in the plan's own `<objective>` (citing 07-04's precedent) and restated in the executor's `<the_inversion>` briefing: **the RED runs here are deliberate mutations of shipped PRODUCT code, not failing tests written ahead of an implementation.** There is no `test(...)` → `feat(...)` gate pair to validate, because no new product behavior is being implemented — the deliverable is proof that already-shipped, already-correct assertions can detect already-fixed historical defects (and, for family (d), an honest report that one specified mutation does not).

Per this plan's own commit discipline (07-04 precedent, restated in `<objective>`) and the executor briefing's explicit instruction: **the RED evidence for this plan lives in `08-MUTATION-LOG.md` as VERBATIM pasted failing output, not in a `test(...)` commit.** `gsd_run check tdd-red-evidence` was correctly NOT invoked — that gate validates a compile-time RED against new test code, which does not exist in this plan; the actual non-vacuity proof is the mutation log itself, whose every family entry records: a pre-mutation `git diff --quiet` gate, the confirmed-applied mutation, the verbatim command output (RED or, for family (d), the honestly-reported non-RED outcome), the revert, a post-revert `git diff --quiet` gate, and a green re-run. All three commits in this plan (`db4a3aea`, `c8b6187a`, `6f9bf7ac`) are `docs(08-04): ...` — there is no `test(08-04): ...` or `feat(08-04): ...` commit, and none was expected, matching 08-01-SUMMARY.md's and 08-02-SUMMARY.md's identical precedent for the mutation-log-as-RED-evidence shape.

No mutation of product code was ever committed — every mutation in this plan was applied, confirmed, exercised, and reverted byte-clean within the same shell session before the corresponding log entry was written and committed, verified by the phase-wide byte-clean proof in Task 3 (`git status --porcelain` empty, `git diff` against the branch point exits 0 for all three touched files).

## Next Phase Readiness

- Phase 8 (tmux Real-PTY Harness) is functionally complete: all four plans executed, `test/tmux` holds 5 real-PTY assertions wired into CI (`tmux-e2e` job), and this plan's mutation log proves three of the four new assertion classes (TTY-03/TTY-04/TTY-05) can actually fail against real historical or newly-possible defects.
- **One open item carries forward past this plan: family (d)'s TTY-06 finding.** `.planning/WINDOWS.md` now carries an open `unmet-truth` entry recording that `TestInstallPickerFrameStableWhileIdle` does not fail against the D-06-specified `AltScreen=false` mutation, with the root cause (the test measures post-settle idle stability, not the settling transient the mutation affects). This does not block phase completion — ROADMAP's own success criterion 4 (TTY-05/TTY-06's functional behavior) was already independently verified by 08-02's own tests, and criterion 3 (the only ROADMAP criterion that explicitly demands "watched fail") is fully discharged by families (a) and (b) — but it is a real, unresolved gap in D-06's *extended* ambition that a future phase or todo should decide whether to close (e.g., by adding a settling-transient-observing assertion, which this plan deliberately declined to add per the Phase 7 D-07 precedent against scope creep in a mutation-log plan).
- **TTY-07's `TMUX_EXPECTED_VERSION` bootstrap remains open from 08-03**, unrelated to this plan: the sentinel `UNPINNED-BOOTSTRAP` is still in place pending the first real `ci.yml` run of `tmux-e2e` on this branch.
- **08-03's `user_setup` item (GitHub ruleset required-status-check) also remains open**, unrelated to this plan.
- `.planning/STATE.md` should be advanced to reflect Phase 8 fully executed (4/4 plans) once this SUMMARY's state update runs.

## Self-Check: PASSED

All key-files confirmed present on disk (`08-MUTATION-LOG.md`, `WINDOWS.md` modified). All 3 commits (`db4a3aea`, `c8b6187a`, `6f9bf7ac`) confirmed present in `git log --oneline --all`. Phase-wide byte-clean proof re-verified immediately before writing this SUMMARY: `git status --porcelain -- internal/cli internal/cli/tui` empty; `git diff --quiet` against the phase's branch point over all three touched production files exits 0.

---
*Phase: 08-tmux-real-pty-harness*
*Completed: 2026-09-10*
