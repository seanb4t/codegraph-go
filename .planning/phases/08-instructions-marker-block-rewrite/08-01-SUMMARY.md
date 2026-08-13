---
phase: 08-instructions-marker-block-rewrite
plan: 01
subsystem: api
tags: [mcp, wire-contract, go-sdk, tdd]

requires:
  - phase: 05-mcp-resources-capability-claims-drift-guard
    provides: "resources/list + resources/read live capability (registerResources), newTestSession/ListResources/ReadResource test pattern"
  - phase: 06-agent-skill-package-skill-md-sessionstart-nudge
    provides: "claudeassets.SkillMarkdown()/SkillMarkdownPath — the embedded .claude/skills/codegraph/SKILL.md source of truth"
  - phase: 07-codegraph-install-skill-hooks-distribution-claude-code
    provides: "codegraph install actually writing the Claude Code skill file from claudeassets.SkillMarkdown() bytes — the fact WIRE-03's skill guard resolves against"
provides:
  - "Rewritten internal/mcp instructions const naming resources/list and the Claude-Code-scoped codegraph skill, within the existing 600-byte wire budget"
  - "WIRE-03 resolvability guards (resourcesClaimResolves, skillClaimResolves) proven non-vacuous via a synthetic-input table test"
  - "session.InitializeResult().Instructions verified byte-identical to the server.go const (end-to-end wire proof)"
affects: [08-02-marker-block-rewrite, 08-03-wire-oracle-recapture]

actuals:
  tokens: 3627
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Parameterized resolvability checkers (resourcesClaimResolves, skillClaimResolves) that take claim/read as arguments rather than reaching for package state — makes the checker itself table-testable against synthetic inputs, independent of a live session or a real embedded asset"

key-files:
  created: []
  modified:
    - internal/mcp/server.go
    - internal/mcp/instructions_contract_test.go

key-decisions:
  - "Cut the anchor-free 'optional path argument' clause (per 08-RESEARCH.md Pitfall 2) to make room for the new resources/skill clauses, keeping default/CODEGRAPH_MCP_TOOLS/codegraph init verbatim (D-02)"
  - "Scoped the skill claim explicitly to Claude Code ('in Claude Code, codegraph install also adds the codegraph skill') per 08-RESEARCH.md Pitfall 1 resolution 1 — Phase 7 only installs the skill for Claude Code, so an unscoped claim would misdirect the other 7 agent targets"
  - "No codegraph:// URI enumerated in the const (D-03) — generic 'resources/list' pointer only, keeping the wire-budget-constrained literal narrow"

patterns-established:
  - "WIRE-03 resolvability guard pattern: a checker function taking claim + real-source accessors as parameters, proven non-vacuous by a synthetic table test mirroring TestREADMEGateCheckerIsNotVacuous's shape"

requirements-completed: [WIRE-01, WIRE-03]

coverage:
  - id: D1
    description: "instructions const rewritten to name resources/list, all 3 pre-existing anchors (default, CODEGRAPH_MCP_TOOLS, codegraph init) preserved verbatim, no codegraph:// URI enumerated"
    requirement: "WIRE-01"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsDescribesEveryVisibilityMechanism"
        status: pass
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsStaysWithinWireBudget"
        status: pass
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsCarriesNoWireContractViolation"
        status: pass
    human_judgment: false
  - id: D2
    description: "instructions const names the Claude-Code-scoped codegraph skill, resolvable to the exact bytes codegraph install writes"
    requirement: "WIRE-03"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsSkillClaimIsResolvable"
        status: pass
    human_judgment: false
  - id: D3
    description: "resources/list claim resolves against a live ListResources/ReadResource round-trip, never a hardcoded URI list"
    requirement: "WIRE-03"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsResourcesClaimIsResolvable"
        status: pass
    human_judgment: false
  - id: D4
    description: "session.InitializeResult().Instructions is byte-identical to the server.go const — the claim reaches the wire, not just the package"
    requirement: "WIRE-01"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsReachesTheWireVerbatim"
        status: pass
    human_judgment: false
  - id: D5
    description: "Both WIRE-03 resolvability guards proven to discriminate over synthetic inputs (never a vacuous nil-returning checker)"
    requirement: "WIRE-03"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsClaimGuardsAreNotVacuous"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-13
status: complete
---

# Phase 8 Plan 1: Instructions Wire-Contract Rewrite Summary

**Rewrote the MCP `instructions` wire const (server.go) to name `resources/list` and a Claude-Code-scoped `codegraph skill` pointer, backed by two new derive-from-source resolvability guards proven non-vacuous — 554 of 600 bytes, all 5 anchors present.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-13T20:13:13Z
- **Tasks:** 3
- **Files modified:** 2 (`internal/mcp/server.go`, `internal/mcp/instructions_contract_test.go`)

## Accomplishments

- `instructions` const rewritten: dropped the anchor-free "optional path argument" clause, compressed the default/init clause while preserving `default`/`CODEGRAPH_MCP_TOOLS`/`codegraph init` verbatim, and appended two new generic pointers — `resources/list` for tool-by-tool reference docs, and a Claude-Code-scoped "codegraph skill" claim. Final value: 554 bytes (46 bytes of headroom under the 600-byte budget).
- `TestInstructionsDescribesEveryVisibilityMechanism` extended from 3 to 5 anchor rows.
- Two new WIRE-03 resolvability guards added, both parameterized (never reaching for package state), both with **no `t.Skip` branch**:
  - `resourcesClaimResolves` / `TestInstructionsResourcesClaimIsResolvable` — proven against a live `ListResources`/`ReadResource` round-trip through `newTestSession`.
  - `skillClaimResolves` / `TestInstructionsSkillClaimIsResolvable` — proven against `claudeassets.SkillMarkdown()`/`SkillMarkdownPath`, the exact bytes `internal/agents/claude.go`'s `Install` reads and writes; never a second hand-typed path.
- `TestInstructionsCarriesNoWireContractViolation` added (T-08-01 mitigation): pure-ASCII + no-absolute-path-token guard.
- `TestInstructionsReachesTheWireVerbatim` added: asserts `session.InitializeResult().Instructions` equals the const byte-for-byte — the end-to-end proof the claim reaches a real client.
- `TestInstructionsClaimGuardsAreNotVacuous` added: an 11-subtest table (6 for `resourcesClaimResolves`, 5 for `skillClaimResolves`) proving both checkers discriminate against every equivalence class and boundary neighbor, including the whitespace-only skill-content row.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — the resources claim reaches a live client and provably resolves** - `fddeb18` (feat, TDD)
2. **Task 2: Add the Claude-Code-scoped skill claim and prove it resolves to what install writes** - `b8a19d0` (feat, TDD)
3. **Task 3: Prove both resolvability guards discriminate (non-vacuity table test)** - `f80803f` (test, TDD)

## Files Created/Modified

- `internal/mcp/server.go` - `instructions` const rewritten (580 → 554 bytes)
- `internal/mcp/instructions_contract_test.go` - +247 lines: `resourcesAnchor`, `skillAnchor` consts; `resourcesClaimResolves`, `skillClaimResolves` checkers; `TestInstructionsResourcesClaimIsResolvable`, `TestInstructionsSkillClaimIsResolvable`, `TestInstructionsCarriesNoWireContractViolation`, `TestInstructionsReachesTheWireVerbatim`, `TestInstructionsClaimGuardsAreNotVacuous`; extended `TestInstructionsDescribesEveryVisibilityMechanism`

## Decisions Made

- Cut the "optional path argument" clause (zero anchor overlap, per 08-RESEARCH.md Pitfall 2) rather than the default/init clause, which carries two of three required anchors.
- Scoped the skill claim explicitly to Claude Code — "in Claude Code, codegraph install also adds the codegraph skill" — per 08-RESEARCH.md Pitfall 1 resolution 1, since Phase 7 only installs the skill for Claude Code and an unscoped claim would misdirect the other 7 agent targets.
- No `codegraph://` URI enumerated in the const (D-03) — kept the resources pointer generic (`resources/list`), consistent with the standard MCP client-side-discovery pattern.

## Demonstrated RED Output (per task, before the corresponding server.go edit)

**Task 1** — before editing `server.go`:
```
=== RUN   TestInstructionsDescribesEveryVisibilityMechanism
    instructions_contract_test.go:111: instructions never mentions the resources
    reference surface (no "resources/list" anywhere in it)...
--- FAIL: TestInstructionsDescribesEveryVisibilityMechanism (0.00s)
=== RUN   TestInstructionsResourcesClaimIsResolvable
    instructions_contract_test.go:198: claim "..." never mentions "resources/list",
    so a client reading it has no way to learn resources/list exists
--- FAIL: TestInstructionsResourcesClaimIsResolvable (0.05s)
```

**Task 2** — before editing `server.go`:
```
=== RUN   TestInstructionsDescribesEveryVisibilityMechanism
    instructions_contract_test.go:123: instructions never mentions the Claude Code
    skill pointer (no "codegraph skill" anywhere in it)...
--- FAIL: TestInstructionsDescribesEveryVisibilityMechanism (0.00s)
=== RUN   TestInstructionsSkillClaimIsResolvable
    instructions_contract_test.go:244: claim "..." never mentions "codegraph skill",
    so a client reading it has no way to learn the codegraph skill exists
    (source: .claude/skills/codegraph/SKILL.md)
--- FAIL: TestInstructionsSkillClaimIsResolvable (0.00s)
```

## One-Time Demonstrated-RED Mutations (per acceptance criteria, both reverted byte-clean)

**Task 2** — `skillClaimResolves`'s `read` argument temporarily pointed at a function returning nil bytes:
```
=== RUN   TestInstructionsSkillClaimIsResolvable
    instructions_contract_test.go:244: claim "..." names "codegraph skill", but its
    content is empty (or whitespace-only) after TrimSpace (source:
    .claude/skills/codegraph/SKILL.md)
--- FAIL: TestInstructionsSkillClaimIsResolvable (0.00s)
```
Revert confirmed: `git diff internal/mcp/instructions_contract_test.go` showed no residual change and `go test ./internal/mcp/ -count=1` passed.

**Task 3** — `skillClaimResolves` body temporarily replaced with `return nil` unconditionally:
```
--- FAIL: TestInstructionsClaimGuardsAreNotVacuous (0.00s)
    --- FAIL: TestInstructionsClaimGuardsAreNotVacuous/skillClaimResolves (0.00s)
        --- FAIL: .../claim_missing_anchor,_readable_content (0.00s)
        --- FAIL: .../claim_present,_read_returns_an_error (0.00s)
        --- FAIL: .../claim_present,_read_returns_empty_bytes (0.00s)
        --- FAIL: .../claim_present,_read_returns_whitespace-only_bytes (0.00s)
        --- PASS: .../claim_present,_read_returns_real_content (0.00s)
```
While mutated, `TestInstructionsResourcesClaimIsResolvable` and `TestInstructionsSkillClaimIsResolvable` were independently re-run and confirmed to **stay green** — the exact point of the mutation (a vacuous checker would keep those two live tests passing forever while proving nothing). Revert confirmed via `git diff --stat` (only the intended 82-line net addition remained) and `go test ./internal/mcp/ -count=1` passing.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for all three tasks.

### Documentation Note (not a code deviation)

**Task 1's acceptance-criteria byte-count check is imprecise, not a defect in the implementation.** The plan's literal check (`rg -o 'resources/list' internal/mcp/server.go | wc -l` `= 1`) assumes zero pre-existing occurrences of the substring `resources/list` in `server.go`. In fact `server.go` already contained one unrelated occurrence at line 763 (`case "resources/list", "resources/read":` inside the middleware method-dispatch switch, pre-existing since Phase 5). The actual count is 2, not 1. The substantive property the criterion exists to check — **the anchor appears exactly once inside the `instructions` const itself** — is verified true by direct inspection of the const (server.go:56). No code change was needed; this is purely a note that the shell one-liner in the plan's acceptance criteria didn't anticipate the pre-existing switch-case literal.

---

**Total deviations:** 0 code deviations; 1 documentation note (imprecise acceptance-criteria shell check, substantive property verified true by other means).
**Impact on plan:** None on scope or correctness.

## Issues Encountered

None during this plan's own scope. For context (not a blocker for this plan): `go test ./test/wireoracle/...` now fails 38/42 `TestFrozenTranscriptsMatch` subtests, because the `instructions` const changed and those golden transcripts still carry the pre-rewrite string. This is expected, out-of-scope intermediate state — the wire-oracle re-capture is explicitly plan 08-03's responsibility (per 08-CONTEXT.md: "re-frozen in plan 08-03"), and this plan's own `<verification>` section scopes only to `internal/mcp` (`go test ./internal/mcp/`, `go build ./...`, `go vet ./internal/mcp/`, `gofmt -l internal/mcp/` — all pass). Flagging here so the phase orchestrator and 08-03's executor are not surprised by the wire-oracle suite's red state between now and that plan's completion.

## Next Phase Readiness

- `internal/mcp`'s `instructions` const and its full guard suite are complete and green; ready for 08-02 (marker-block rewrite, `internal/agents/instructions.go`) and 08-03 (wire-oracle re-capture, which depends on this plan's final const value being frozen — it now is).
- `test/wireoracle`'s `TestFrozenTranscriptsMatch` is in an expected red state for the 38 scenarios carrying the `instructions` string until 08-03 re-freezes them — not a regression to chase before then.

## Self-Check: PASSED

- FOUND: `internal/mcp/server.go`
- FOUND: `internal/mcp/instructions_contract_test.go`
- FOUND: `.planning/phases/08-instructions-marker-block-rewrite/08-01-SUMMARY.md`
- FOUND commit: `fddeb18` (Task 1)
- FOUND commit: `b8a19d0` (Task 2)
- FOUND commit: `f80803f` (Task 3)

---
*Phase: 08-instructions-marker-block-rewrite*
*Completed: 2026-08-13*
