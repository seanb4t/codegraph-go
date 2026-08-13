---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
plan: 03
subsystem: agents
tags: [agent-skills, mcp, claims-drift-guard, mutation-testing, skill-md]

# Dependency graph
requires:
  - phase: 06-agent-skill-package-skill-md-sessionstart-nudge
    provides: "06-01's .claude/hooks/session-nudge.sh + settings.json + hooks.json, 06-02's SKILL.md skeleton and skill_claims_drift_test.go guard"
provides:
  - "SKILL.md's worked-examples section: three examples in D-05's locked order, the first reproducing the 2026-08-08 misdirection incident end to end and framed as history, not live behavior"
  - "skill_claims_drift_test.go extended with a bounded worked-example counter (countSkillWorkedExamples), its non-vacuity proof, and a locked-count gate (skillWorkedExampleCount = 3)"
  - "The nudge script's emitted text brought under the same GUARD-01 derived-honesty checks as SKILL.md (second layer over hookpackage_test.go's byte-exact pin)"
  - "16 total demonstrated-red mutations in test/wireoracle/MUTATION-PROOF.md (11 pre-existing + 5 appended this plan), proving every Phase-6 guard actually fires"
affects: ["06-04-rehearsal-verification: the worked examples are now the content a rehearsal transcript would exercise"]

# Actuals (#2632)
actuals:
  tokens: 7405
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Locked user-decision counts (skillWorkedExampleCount, citing D-05) are recorded as named constants with a doc comment stating their non-derivable source, distinguishing them from every other checker in the file that compares against a real runtime value"
    - "Second-layer derived-honesty checks over an already byte-pinned string: hookpackage_test.go's nudgeLine pins the emitted bytes exactly; skill_claims_drift_test.go's TestNudgeText* checks the same text against allToolNames()/envVarTokenRe/hostFactsIn/numericClaimsMultiset/countClaimsIn, so an edit cannot pass both layers while naming something that doesn't exist"
    - "Mutation-proof entries record which OTHER guards stayed green from the same edit (asymmetry), continuing 05-03-PLAN's discipline"

key-files:
  created: []
  modified:
    - .claude/skills/codegraph/SKILL.md
    - internal/mcp/skill_claims_drift_test.go
    - test/wireoracle/MUTATION-PROOF.md

key-decisions:
  - "Mutation 12 renamed the node companion to peek (not reusing mutation 7's status->health target) to keep this plan's evidence distinct from Phase 5's own recorded mutation on the same rename pattern"
  - "Mutation 14 (resume matcher) was re-run fresh rather than citing 06-01-SUMMARY.md's mention of 'a targeted one-field mutation to hooks.json' — that summary doesn't name which field was mutated, so citing it in place of a fresh, specific run would have been a weaker record than the plan's own instruction intended"
  - "countSkillWorkedExamples bounds its count to the worked-examples ## section (first heading containing 'example' case-insensitively, through the next ## heading or EOF) rather than counting every ### in the document, so a future section's own sub-headings can never silently inflate SKILL-02's count"

requirements-completed: [SKILL-02]

coverage:
  - id: D1
    description: "SKILL.md carries exactly three worked examples in D-05's locked order; example 1 reproduces the 2026-08-08 misdirection incident end to end, cites the resolved debug log by path, names CODEGRAPH_MCP_TOOLS as the actual gate, and states the instructions string has since been corrected"
    requirement: "SKILL-02"
    verification:
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestSkillCarriesExactlyThreeWorkedExamples"
        status: pass
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestSkillNamesOnlyRealTools"
        status: pass
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestSkillResourceURIsResolve"
        status: pass
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestSkillDefersNumericFactsToResources"
        status: pass
    human_judgment: false
  - id: D2
    description: "The worked-example count is gated by a derived, bounded counter proven to discriminate at both boundaries (2 and 4) against the real file, and the nudge script's emitted text is checked against the same tool-roster/env-var/host-fact/numeric-claim guards SKILL.md carries"
    requirement: "SKILL-02"
    verification:
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestSkillWorkedExampleCounterIsNotVacuous"
        status: pass
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestNudgeTextNamesOnlyRealTools"
        status: pass
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestNudgeTextCarriesNoUnpinnedFacts"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every guard introduced across 06-01/06-02/06-03 has been observed failing against a real mutation of the real tree, with verbatim failure text recorded, and reverted byte-clean"
    verification:
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutations 12-16"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-08-12
status: complete
---

# Phase 6 Plan 3: SKILL.md Worked Examples & Mutation Proofs Summary

**Added SKILL-02's three worked examples to SKILL.md (the first reproducing the 2026-08-08 misdirection incident verbatim from its resolved debug log), extended the drift guard with a bounded worked-example counter and second-layer nudge-text checks, and demonstrated five new mutations red against the real tree — bringing this phase's total to 16 proven guards.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-12T23:05:29Z (prior commit `408e12c` — Wave 1 tracking update)
- **Completed:** 2026-08-12T23:24:51Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- `.claude/skills/codegraph/SKILL.md`: a new `## Worked examples` section, three `### ` sub-headings in D-05's locked order — the 2026-08-08 misdirection incident (cited by path, closing with the correction statement), impact analysis routed to `codegraph_impact`, and cross-file dynamic-dispatch lookup routed to `codegraph_explore` — inserted after the decision table and skip-condition sections, all existing 06-02 guards still green.
- `internal/mcp/skill_claims_drift_test.go`: `countSkillWorkedExamples` (bounded to the worked-examples section, immune to a `#### ` false match), `skillWorkedExampleCount = 3` (a documented locked constant, not a derived value), `nudgeScriptPath`, and four new test functions — `TestSkillCarriesExactlyThreeWorkedExamples`, `TestSkillWorkedExampleCounterIsNotVacuous` (0/1/2/3/4 examples, out-of-section heading, `####` heading), `TestNudgeTextNamesOnlyRealTools`, `TestNudgeTextCarriesNoUnpinnedFacts` — all passed on first run against the already-authored real files.
- Demonstrated both worked-example count boundaries red against the real `SKILL.md` (delete one example -> "carries 2 worked example(s), want 3"; duplicate one -> "carries 4"), each reverted byte-clean before the next.
- `test/wireoracle/MUTATION-PROOF.md`: appended Mutations 12-16 (renamed companion tool, dead resource URI with asymmetry recorded, resume-matcher drift, one-character nudge-text change, renamed nudge script), continuing the file's numbering from its actual state (11, verified by `grep -c '^## Mutation'` before writing anything) — file now records 16 mutations total.
- Full suite run twice on the reverted tree; `internal/daemon`'s documented load-dependent flake (STATE.md, GitHub issue #17) fired on a different test each run, both confirmed passing in isolation; no mutation touches `internal/daemon`.

## Task Commits

Each task was committed atomically:

1. **Task 1: The three worked examples (SKILL-02)** - `74e8e22` (feat)
2. **Task 2: Gate the example count, extend derived guards to nudge text** - `204f484` (test)
3. **Task 3: Demonstrate every Phase-6 guard red against the real tree** - `1d6f146` (test)

**Plan metadata:** (this commit, made after this SUMMARY)

_Note: Task 2 carried `tdd="true"` in the plan, but its assertions targeted files 06-01/06-02 had already authored correctly — both real-file tests passed on first run, so there was no RED-then-fix cycle to split across commits; this is recorded explicitly in the task's own doc comments per the plan's instruction to "record whether each was green on first run ... or required a correction."_

## Files Created/Modified

- `.claude/skills/codegraph/SKILL.md` - added the worked-examples section (3 examples, 18 lines net)
- `internal/mcp/skill_claims_drift_test.go` - added `countSkillWorkedExamples`, `skillWorkedExampleCount`, `nudgeScriptPath`, `workedExampleHeadingRe`, and 4 test functions (177 lines net)
- `test/wireoracle/MUTATION-PROOF.md` - appended Mutations 12-16 plus a numbering note and closing statement (278 lines net)

## Decisions Made

- Mutation 12 renamed the `node` companion to `peek` rather than reusing Mutation 7's `status`->`health` rename, keeping this plan's evidence distinct from Phase 5's own recorded mutation on the identical 3-site rename pattern.
- Mutation 14 (the `resume` matcher) was run fresh against the real tree rather than citing 06-01-SUMMARY.md's mention of "a targeted one-field mutation to `hooks.json`" — that summary text doesn't specify which field was mutated or record the verbatim failure, so a fresh, specific run is the stronger record the plan's own instruction intended.
- `countSkillWorkedExamples` bounds its count strictly to the worked-examples `## ` section rather than counting every `### ` heading in the document, so a future section's own sub-headings (e.g. a later reference or troubleshooting section) can never silently inflate SKILL-02's locked count.

## Deviations from Plan

None — plan executed exactly as written. Both real-file tests in Task 2 passed on first implementation attempt (no fix-attempt iterations needed); Task 3's five mutations all applied, failed, and reverted exactly as the plan specified.

## Issues Encountered

The full-suite run (`go test ./... -count=1`) hit `internal/daemon`'s documented load-dependent flake (STATE.md's "Daemon extreme-load tail" condition, GitHub issue #17) twice, on two different named tests across two separate runs (`TestRunWatchdogCancelsRunOnSimulatedReparent`, then `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`). Both were re-run in isolation and passed (1.06s and 5.17s respectively), matching STATE.md's own description of the condition exactly. Neither this plan's edits nor its mutations touch `internal/daemon` or anything it imports — recorded per the plan's own acceptance criteria as the known pre-existing condition, not absorbed as a new failure.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SKILL.md is now structurally and content-complete for SKILL-01/SKILL-02: decision table, skip condition, three worked examples, and resource-pointer reference section, all guarded by `skill_claims_drift_test.go`.
- The nudge script's text is now covered by the same derived-honesty discipline as SKILL.md, closing the gap 06-01's `nudgeLine` doc comment flagged as "proves nothing about their honesty" on its own.
- 06-04 (rehearsal verification) can proceed: the worked examples this plan added are exactly the content a fresh "after" session transcript would exercise against the 2026-08-08 "before" transcript D-02 already locks.
- `test/wireoracle/MUTATION-PROOF.md` now records 16 mutations; any future phase extending this file must re-read its actual highest `## Mutation` heading before appending, per this plan's own (and 05-03-PLAN's) corrected practice.

## Self-Check: PASSED

- FOUND: `.claude/skills/codegraph/SKILL.md` (worked-examples section present, 3 `### ` headings)
- FOUND: `internal/mcp/skill_claims_drift_test.go` (extended)
- FOUND: `test/wireoracle/MUTATION-PROOF.md` (16 `## Mutation` headings)
- FOUND: commit `74e8e22` (Task 1)
- FOUND: commit `204f484` (Task 2)
- FOUND: commit `1d6f146` (Task 3)

---
*Phase: 06-agent-skill-package-skill-md-sessionstart-nudge*
*Completed: 2026-08-12*
