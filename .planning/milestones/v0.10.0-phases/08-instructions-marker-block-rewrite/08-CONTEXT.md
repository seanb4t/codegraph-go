# Phase 8: Instructions & Marker-Block Rewrite - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 8 rewrites two agent-facing wire artifacts so neither promises a hand-off that doesn't exist and both correctly point at what Phases 5-7 actually shipped: (1) the MCP `initialize`/`server/discover` `instructions` string (`internal/mcp/server.go`), which currently says nothing about the skill or resources at all, and (2) the marker-fenced `codegraphInstructionsBlock` (`internal/agents/instructions.go`) that `codegraph install` injects into 4 of 8 agent targets (Claude, Codex, opencode, Gemini), whose package comment still describes a stale "Phase 3" deferral to a hand-off that was never built. Covers WIRE-01 (rewrite the wire `instructions` string), WIRE-02 (marker block matches it and promises nothing unshipped), and WIRE-03 (every named capability resolvable at test time — this phase ships LAST specifically so nothing it names can outlive its promise). Includes re-freezing the wire-oracle transcripts that carry the `instructions` string byte-for-byte, with the diff attributed.

**Not in scope:** authoring SKILL.md/resource content (done, Phases 5-6), skill/hooks distribution mechanics (done, Phase 7), porting skill/hooks to any agent besides Claude Code (v2, AGENT-04…07), and the PreToolUse guard hook (v2, GUARD-HOOK-01/02). The marker fences themselves (`<!-- CODEGRAPH_START/END -->`) are a byte-exact cross-implementation contract with TS CodeGraph and must not change — only the content between them.

</domain>

<decisions>
## Implementation Decisions

### Wire budget vs. new content (WIRE-01)
- **D-01:** The `instructions` const's existing ~600-byte / single-paragraph / no-newline budget (`internal/mcp/instructions_contract_test.go`'s `instructionsMaxBytes`) is **kept as-is, not raised**. The current string already measures 580/600 bytes, so fitting a skill/resources pointer requires trimming or restructuring existing content, not just appending. This matches unanimous research consensus (MCP's own server-instructions blog, OpenAI's MCP guidance, niteagent's authoring checklist): instructions should stay under ~200-300 words, must not become "a manual," and should state only what tool/resource descriptions can't already convey — longer instruction blocks get measurably degraded model attention. — **Reversibility:** reversible — the byte cap is a repo-local test constant; raising it later is a one-line test change, not a migration.
- **D-02:** The three existing pinned anchors (`TestInstructionsDescribesEveryVisibilityMechanism`'s "default"/`CODEGRAPH_MCP_TOOLS`/"codegraph init" tokens) must all still be present in the rewritten string — this phase adds a skill/resources pointer on top of, not instead of, that existing contract.

### Resource URI naming granularity (WIRE-01, WIRE-03)
- **D-03:** The rewritten `instructions` string points at the skill and resources **generically** — e.g. "see the codegraph skill" / "call resources/list for tool-by-tool reference docs" — and does **not** enumerate the 10 individual `codegraph://` URIs (8 per-tool + `tools-filter` + `index-state`) by name. No research source found suggests re-listing discoverable resource URIs inside a top-level instructions string; the standard MCP pattern is client-side discovery via `resources/list`. This also keeps WIRE-03's drift-guard surface narrower: the guard needs to prove the skill path and the resource *capability* are real, not keep 10 literal URI strings pinned inside the wire-budget-constrained instructions const itself. — **Reversibility:** reversible — nothing else depends on the string being generic vs. explicit; a later change is a wire-oracle re-capture, not a migration.

### Marker-block content per agent target (WIRE-02, WIRE-03)
- **D-04:** `codegraphInstructionsBlock` stays **one shared constant across all 4 marker-block agent targets** (Claude, Codex, opencode, Gemini) — it is rewritten to describe `codegraph_explore` / `codegraph explore` usage directly and does **not** name the installed skill by path or claim one exists. Only Claude Code received the actual SKILL.md+hooks install (Phase 7, v1-scoped); Codex/opencode/Gemini did not (AGENT-04…07, v2/deferred). Research on CLAUDE.md/AGENTS.md authoring strongly favors a short "see the skill" pointer over restating content, but that pattern is only valid where the skill genuinely exists — pointing Codex/opencode/Gemini at a skill file that was never installed for them would recreate exactly the "promise nothing delivers" defect WIRE-01/02/03 exist to close. A skill-agnostic shared block satisfies WIRE-03 uniformly for all 4 targets without introducing a second variant to keep in sync. — **Reversibility:** costly — **rationale:** once Claude Code's own marker block starts referencing the installed skill (if a later phase does that), reverting to one shared block again means re-auditing all 4 targets' text for accuracy, not just editing one constant.
- **D-05:** The package comment in `internal/agents/instructions.go` describing "explicitly defers full tool guidance to the MCP initialize response (Phase 3)" is corrected as part of this rewrite — it's a stale internal-numbering reference (the milestone was later split into Phases 5-8) and no longer describes what the block actually does post-rewrite. This is a code-comment accuracy fix Claude carries out directly, not a decision requiring user input.

### Claude's Discretion
- Exact wording of the generic skill/resources pointer phrase in both the `instructions` const and the marker block — within D-01/D-02/D-03/D-04's constraints (byte budget, required anchors, generic-not-enumerated, skill-agnostic-for-marker-block).
- What existing clause in the current 580-byte `instructions` string gets trimmed or restructured to make room, since D-01 keeps the budget fixed — a wording/prioritization call, not a vision decision.
- Exact mechanics of the wire-oracle re-capture and diff-attribution (success criterion 4) — follows the existing `test/wireoracle` reviewed-diff discipline already established in Phase 5.
- How WIRE-03's "every named capability is resolvable at test time" guard is implemented for the (now generic) skill/resources pointer — e.g. asserting the skill file path exists and `resources/list` returns a non-empty set, vs. some other mechanism.

### Reviewed Todos (not folded)
- `2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` (mcp area, score 0.9 — highest match) — read in full. It documents a `tools/list` response **arrival-order** nondeterminism in the wire-oracle harness itself (JSON-RPC response ids arriving out of order under parallel CI load), unrelated to the `instructions` string's *content*. It touches the same `test/wireoracle` machinery this phase's re-capture step uses, but is a pre-existing, separately-tracked flake (also carried in PROJECT.md's "Open threads carried out of v0.3.0") — not something this phase's re-capture should silently absorb or fix.
- Five lower-score (0.6) todos — release dry-run diff guard, post-release-verify conclusion guard, golangci-lint addition, brew tap trust docs, tap secret-distinctness test — all keyword-match only (generic terms like "test"/"install"/"internal"), no actual overlap with instructions/marker-block content.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 8: Instructions & Marker-Block Rewrite" — goal, 4 success criteria (WIRE-01/02/03 + byte-identical re-captured transcripts), and the explicit framing: "This phase is last by requirement, not by convenience — WIRE-03 makes 'everything named already exists' the acceptance condition." Also notes the marker fences are a byte-exact cross-implementation contract with TS CodeGraph — do not alter `<!-- CODEGRAPH_START/END -->` themselves.
- `.planning/REQUIREMENTS.md` §"Instructions Rewrite" — WIRE-01, WIRE-02, WIRE-03 full text; traceability table confirms all three map only to Phase 8 and are currently `Pending`.

### Existing code — the two artifacts this phase rewrites
- `internal/mcp/server.go` — the `instructions` const (~line 56) and its doc comment stating the 600-byte/no-newline/single-paragraph wire-contract constraints (T-03-19: no interpolation, ever — it's JSON-encoded into every wire-oracle transcript). `BuildServer`'s `ServerOptions.Instructions` wiring (~line 537) reaches both the classic `initialize` result and `server/discover` through the identical SDK field.
- `internal/mcp/instructions_contract_test.go` — the full existing guard suite: `TestInstructionsNamesTheNarrowingFilter`, `TestInstructionsDescribesEveryVisibilityMechanism` (3 pinned anchors: "default", `CODEGRAPH_MCP_TOOLS`, "codegraph init" — D-02 requires all 3 survive the rewrite), `TestInstructionsStaysWithinWireBudget` (the 600-byte/no-newline/non-empty checks D-01 keeps as-is), `docNamesCompanionsWithoutTheFilter`/`TestREADMEDocumentsToolVisibilityGate`. WIRE-03's new guard extends this file's derive-from-source discipline, not a parallel mechanism.
- `internal/agents/instructions.go` — `codegraphSectionStart`/`codegraphSectionEnd` marker fences (byte-exact, do not alter) and `codegraphInstructionsBlock`, the shared const D-04 rewrites to be skill-agnostic. Package comment currently describes the stale "Phase 3" deferral D-05 corrects.
- `test/wireoracle/scenarios.go`, `oracle_test.go`, `COVERAGE-BASELINE.md`, `MUTATION-PROOF.md` — the reviewed-diff re-capture discipline every prior capability change (most recently Phase 5's 13 new scenarios) has followed; this phase's success criterion 4 requires the same discipline applied to the rewritten `instructions` string.

### Phase 5 (dependency — resource URIs this phase points at, does not restate)
- `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-CONTEXT.md` — URI naming scheme this phase references generically per D-03: `codegraph://tools/<name>` ×8, `codegraph://tools-filter`, `codegraph://index-state`. D-01 there already established fact-sheet-only content with no cross-references — this phase's generic pointer is consistent with that division, not a new one.

### Phase 6 (dependency — skill this phase points at, does not restate)
- `.planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-CONTEXT.md` — `.claude/skills/codegraph/SKILL.md` is the canonical skill location this phase's generic pointer (D-03) and Claude-Code-only skill awareness (D-04's rationale) refer to.

### Phase 7 (dependency — the actual per-target install state this phase must describe accurately)
- `.planning/phases/07-codegraph-install-skill-hooks-distribution-claude-code/07-CONTEXT.md` — confirms skill+hooks distribution is Claude-Code-only in v1 (D-01/D-02 there: global `--location` default, no per-repo scoping guard). This is the load-bearing fact behind D-04's "marker block stays skill-agnostic for all 4 targets" decision — Codex/opencode/Gemini genuinely have no skill installed to point at.

### Research consulted this session (informs D-01, D-03, D-04)
- MCP Blog, "Server Instructions: Giving LLMs a user manual for your server" (blog.modelcontextprotocol.io, 2025-11-03) — "Don't write a manual"; instructions should state cross-tool relationships/constraints tool descriptions can't convey, not restate them.
- OpenAI MCP server guidance (developers.openai.com/plugins/build/mcp-server) — "Keep the most important details in the first 512 characters."
- niteagent, "MCP Server Instructions: Giving LLMs a User Manual for Your Tools" (2026-06-10) — explicit checklist: under 300 words, no repeated tool descriptions, model-agnostic language.
- Multiple sources on CLAUDE.md/AGENTS.md authoring (Claude Help Center, teamvince.com, kuril.in, claudecode-lab.com) — consistent "keep short, point at skills for detail, one-liner pointer beats restating content" guidance, directly informing D-04's marker-block approach (with the caveat, unique to this phase, that the pointer is only valid where the pointed-at thing exists).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/mcp/instructions_contract_test.go`'s existing anchor-pinning pattern (literal-token assertions for mechanisms with no derivable source value) is the precedent WIRE-03's new "skill/resources exist" guard should follow in shape, even though what it pins is different (capability existence, not a numeric/env-var claim).
- `test/wireoracle`'s reviewed-diff re-capture process (already exercised in Phase 5 for the 13 new resource scenarios) is the exact mechanism success criterion 4 needs — not a new capture tool.

### Established Patterns
- **Wire-contract literal-only discipline (T-03-19):** `instructions` must stay a compile-time literal with zero interpolation — no repo path, no resolved value, ever. The generic pointer phrasing (D-03) must be written as a fixed string, same as today's const.
- **Marker-fence byte-exactness:** `codegraphSectionStart`/`codegraphSectionEnd` are a cross-implementation contract with TS CodeGraph; this phase only ever touches the content between them.

### Integration Points
- `internal/mcp/server.go`'s `instructions` const and `internal/agents/instructions.go`'s `codegraphInstructionsBlock` are edited independently but must describe a mutually consistent story (WIRE-02: "marker block matches the rewritten instructions") — same skill/resources pointer framing, adapted to each artifact's own byte/format constraints.
- Any wire-oracle scenario carrying the `instructions` string (multiple of the existing 42 frozen transcripts) needs re-capture once the const changes, per success criterion 4.

</code_context>

<specifics>
## Specific Ideas

No specific wording examples were dictated — the user's substantive input was resolving the three structural tensions above (byte budget, URI enumeration, per-agent divergence), all decided toward the leaner/safer option research and WIRE-03 both point at. Exact phrasing is Claude's discretion within those constraints.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. See Reviewed Todos above for the 6 todo matches considered and not folded.

</deferred>

---

*Phase: 8-Instructions & Marker-Block Rewrite*
*Context gathered: 2026-08-13*
