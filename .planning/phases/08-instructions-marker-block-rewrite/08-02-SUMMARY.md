---
phase: 08-instructions-marker-block-rewrite
plan: 02
subsystem: cli
tags: [marker-block, install, tdd, honesty-guard]

requires:
  - phase: 08-instructions-marker-block-rewrite
    plan: "08-01"
    provides: "Rewritten internal/mcp instructions const naming resources/list and the Claude-Code-scoped codegraph skill — the vocabulary this plan's marker-block bullet matches"
provides:
  - "codegraphInstructionsBlock gains a Reference docs bullet naming resources/list and resources/read"
  - "blockNamesUnshippedCapability + TestInstructionsBlockNamesOnlyShippedCapabilities — the marker block never names an unshipped capability (skill) for any of its 4 shared targets"
  - "Corrected doc comment above codegraphInstructionsBlock — the stale MCP-initialize/Phase-3 deferral is gone from internal/agents"
affects: []

actuals:
  tokens: 1630
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "blockNamesUnshippedCapability(block string) error — takes the block string as a parameter, never reads codegraphInstructionsBlock from package scope, so it stays table-testable against synthetic inputs and a doc comment discussing the rule cannot make the gate vacuous"

key-files:
  created: []
  modified:
    - internal/agents/instructions.go
    - internal/agents/registry_test.go

key-decisions:
  - "Reference-docs bullet placed immediately after the Shell bullet, before the no-.codegraph/ paragraph, matching the plan's exact placement instruction"
  - "Doc comment cross-references 08-RESEARCH.md Pitfall 1 rather than restating its content — records the deliberate wire-vs-marker-block divergence on the skill claim as a decision, not folklore"

requirements-completed: [WIRE-02, WIRE-03]

coverage:
  - id: D1
    description: "Marker block names resources/list and resources/read using the same vocabulary as the rewritten wire instructions const, still names codegraph_explore and codegraph explore"
    requirement: "WIRE-02"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestInstructionsBlock_HasMarkersAndCodegraphExploreReference"
        status: pass
    human_judgment: false
  - id: D2
    description: "blockNamesUnshippedCapability returns non-nil for a block naming a skill, missing codegraph_explore, or missing resources/list; nil otherwise — proven against the real const"
    requirement: "WIRE-02"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestInstructionsBlockNamesOnlyShippedCapabilities"
        status: pass
    human_judgment: false
  - id: D3
    description: "The honesty guard discriminates across all three failure classes plus their passing neighbours, including a mixed-case skill row"
    requirement: "WIRE-03"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestInstructionsBlockGuardIsNotVacuous"
        status: pass
    human_judgment: false
  - id: D4
    description: "The stale MCP-initialize/Phase-3 deferral clause is gone from internal/agents, replaced by a comment describing the deliberate wire-vs-marker-block skill-claim divergence"
    requirement: "WIRE-02"
    verification:
      - kind: unit
        ref: "rg -o 'defers full tool guidance' internal/agents/ (0 matches)"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-13
status: complete
---

# Phase 8 Plan 2: Marker Block Rewrite Summary

**Added a resources/list + resources/read bullet to the shared `codegraphInstructionsBlock` and a `blockNamesUnshippedCapability` honesty guard proving the block never names a skill for any of its 4 shared agent targets, replacing the stale "Phase 3" MCP-initialize deferral in the doc comment.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-13
- **Tasks:** 2
- **Files modified:** 2 (`internal/agents/instructions.go`, `internal/agents/registry_test.go`)

## Accomplishments

- `codegraphInstructionsBlock` gained one new bullet — `**Reference docs** (MCP only): call \`resources/list\` then \`resources/read\` for the per-tool reference beyond this summary.` — placed after the Shell bullet, before the no-`.codegraph/` paragraph. Marker fences (`codegraphSectionStart`/`codegraphSectionEnd`) untouched.
- `blockNamesUnshippedCapability(block string) error` added to `registry_test.go`: fails if the block lacks `codegraph_explore`, lacks `resources/list`, or contains the case-insensitive substring `skill` (D-04's honesty property). Takes the block as a parameter — never reads `codegraphInstructionsBlock` from package scope — so Task 2's doc comment discussing the skill-absence rule cannot self-invalidate the gate.
- `TestInstructionsBlockNamesOnlyShippedCapabilities` applies the checker to the real const value — the direct proof of ROADMAP success criterion 2.
- `TestInstructionsBlockGuardIsNotVacuous` — 6-row table test proving the checker discriminates across all three failure classes (missing `codegraph_explore`, missing `resources/list`, names a skill) plus their passing neighbours, including a mixed-case skill row (`Skill`).
- Doc comment above `codegraphInstructionsBlock` rewritten: the stale "explicitly defers full tool guidance to the MCP initialize response (Phase 3)" clause is gone, replaced with a comment stating which 4 of 8 targets receive the block, that it names no skill on purpose (only Claude Code got one, Phase 7/AGENT-01), and cross-referencing `08-RESEARCH.md` Pitfall 1 for why the wire `instructions` const and this marker block deliberately diverge on the skill claim.

## Task Commits

Each task was committed atomically:

1. **Task 1: Marker block gains the reference-docs pointer and an honesty guard that names nothing unshipped** - `2cb6dd5` (feat, TDD)
2. **Task 2: Retire the stale deferral in the doc comment and prove the honesty guard discriminates** - `a98988f` (docs, TDD)

## Files Created/Modified

- `internal/agents/instructions.go` — added the Reference docs bullet to `codegraphInstructionsBlock`; rewrote the doc comment above it (D-05)
- `internal/agents/registry_test.go` — added `fmt` import, `blockNamesUnshippedCapability`, `TestInstructionsBlockNamesOnlyShippedCapabilities`, `TestInstructionsBlockGuardIsNotVacuous`

## Decisions Made

- Reference-docs bullet wording and placement followed the plan's action text exactly (label "Reference docs", phrase "per-tool reference", immediately after the Shell bullet).
- Doc comment cites `08-RESEARCH.md` Pitfall 1 by name rather than re-deriving the divergence rationale inline, keeping the comment itself short while making the cross-reference discoverable.

## Demonstrated RED Output (per task, before the corresponding instructions.go edit)

**Task 1** — `TestInstructionsBlockNamesOnlyShippedCapabilities`, before adding the Reference docs bullet:
```
=== RUN   TestInstructionsBlockNamesOnlyShippedCapabilities
    registry_test.go:223: codegraphInstructionsBlock marker block "<!-- CODEGRAPH_START -->\n## CodeGraph\n\n...
    (full pre-rewrite block text embedded)... never mentions resources/list, so an agent
    reading it has no pointer to the per-tool reference docs (WIRE-02)
--- FAIL: TestInstructionsBlockNamesOnlyShippedCapabilities (0.00s)
FAIL
```

## One-Time Demonstrated-RED Mutations (per acceptance criteria, both reverted byte-clean)

**Task 1** — temporarily appended `See the installed codegraph skill for more.` to the block body:
```
=== RUN   TestInstructionsBlockNamesOnlyShippedCapabilities
    registry_test.go:223: codegraphInstructionsBlock marker block "...See the installed
    codegraph skill for more.\n<!-- CODEGRAPH_END -->" names a skill, but 3 of its 4
    targets (Codex, opencode, Gemini) never receive one (D-04) — this block is shared
    across all 4 and must stay skill-agnostic
--- FAIL: TestInstructionsBlockNamesOnlyShippedCapabilities (0.00s)
FAIL
```
Revert confirmed: `git diff internal/agents/instructions.go` showed only the intended 1-line Reference-docs bullet addition remaining, `go test ./internal/agents/ -count=1` passed.

**Task 2** — `blockNamesUnshippedCapability` body temporarily replaced with an unconditional `return nil` at the top:
```
=== RUN   TestInstructionsBlockGuardIsNotVacuous
    --- PASS: .../explore_+_resources/list,_no_skill_word (0.00s)
    --- FAIL: .../same_block_plus_a_sentence_naming_an_installed_skill_file (0.00s)
    --- FAIL: .../same_block,_skill_word_in_mixed_case (0.00s)
    --- FAIL: .../resources/list_present,_codegraph_explore_missing (0.00s)
    --- FAIL: .../codegraph_explore_present,_resources/list_missing (0.00s)
    --- FAIL: .../empty_block (0.00s)
FAIL
```
5 of 6 subtests failed (the vacuous checker kept only the always-nil-passing case green, proving nothing about discrimination). Revert confirmed: `git diff internal/agents/registry_test.go` showed no residual change, `go test ./internal/agents/ -count=1` passed, full `TestInstructionsBlockGuardIsNotVacuous` green again with all 6 subtests passing.

## Final Block Body (verbatim)

```
<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.
- **Reference docs** (MCP only): call `resources/list` then `resources/read` for the per-tool reference beyond this summary.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->
```

## Deviations from Plan

None — plan executed exactly as written for both tasks.

## Issues Encountered

None. This plan's scope (`internal/agents`) is independent of plan 08-03's wire-oracle re-capture (`internal/mcp`/`test/wireoracle`) — no overlap, no blocking dependency in either direction beyond the already-satisfied 08-01 dependency (shared vocabulary: `resources/list`).

## Next Phase Readiness

- `internal/agents`'s `codegraphInstructionsBlock` and its full guard suite (`TestInstructionsBlock_HasMarkersAndCodegraphExploreReference`, `TestInstructionsBlock_ExactMarkerText`, `TestInstructionsBlockNamesOnlyShippedCapabilities`, `TestInstructionsBlockGuardIsNotVacuous`) are complete and green.
- No wire-oracle transcript involves `internal/agents` content — plan 08-03's re-capture scope (`internal/mcp/server.go`'s `instructions` const, already frozen by 08-01) is unaffected by this plan.

## Self-Check: PASSED

- FOUND: `internal/agents/instructions.go`
- FOUND: `internal/agents/registry_test.go`
- FOUND: `.planning/phases/08-instructions-marker-block-rewrite/08-02-SUMMARY.md`
- FOUND commit: `2cb6dd5` (Task 1)
- FOUND commit: `a98988f` (Task 2)

---
*Phase: 08-instructions-marker-block-rewrite*
*Completed: 2026-08-13*
