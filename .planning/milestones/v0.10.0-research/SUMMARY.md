# Project Research Summary

**Project:** codegraph-go (v0.10.0 milestone: Agent Onboarding Skill & MCP Resources)
**Domain:** Agent-education UX for a code-graph MCP server via SKILL.md, MCP Resources capability, SessionStart/PreToolUse hooks
**Researched:** 2026-08-12
**Confidence:** HIGH (grounded in official specs, wire-oracle tests, archtest patterns, and this repo's own documented incident history)

## Executive Summary

The v0.10.0 milestone adds a self-teaching agent-onboarding UX layer to codegraph-go by combining three complementary mechanisms: an Agent Skills SKILL.md (decision-procedure-first, not catalog-first), an MCP Resources capability serving reference content, and soft enforcement hooks. This is not three separate features — it is one cohesive UX designed to solve a specific, documented problem: an agent that has read documentation but still reaches for grep first when it should use `codegraph_explore`.

The recommended approach is to build in phases: (1) MCP Resources capability + hand-authored reference markdown, (2) SKILL.md with worked examples and a decision table, (3) rewrite the `instructions` wire-string to defer correctly, (4) SessionStart soft nudge, (5) claims-guarded-by-tests discipline across all three surfaces, then (6) PreToolUse guard hook, and finally (7) `codegraph install` distribution. This ordering is not arbitrary — resources must exist before the skill points to them, and the `instructions` string must never name something that doesn't exist (the exact bug this milestone was invented to fix).

**Critical risk:** This milestone must not repeat the documented "Phase 3" broken promise (instructions string named a capability that never shipped) or the "SURF-01" drift pattern (hand-typed claims diverging from code). Every numeric claim, tool count, and default value in skill/resources/instructions must be either derived from source constants or gated by an automated test. The highest-implementation risk is the new skill-directory and hooks.json install/uninstall safety across 8 agent targets — this mirrors v1.0 Phase 5's githooks data-loss bugs and Phase 6's swallowed-I/O-errors findings. Reuse the existing `internal/fsatomic` and marker-fence discipline, never invent new file-safety primitives.

## Key Findings

### Recommended Stack

No new Go module dependencies are needed. Everything required already exists in `go.mod` (`modelcontextprotocol/go-sdk@v1.7.0`) or is plain-text/JSON authoring:

**Core technologies:**
- **Agent Skills open standard (SKILL.md + frontmatter)** — A SKILL.md file authored once can be consumed by Claude Code, Cursor, Codex CLI, opencode, and Antigravity via the shared `agentskills.io` standard. This is a genuine multi-agent surface, not Claude-Code-only. Frontmatter has 6 fields; only `description` is load-bearing (it's the only thing always in context, so it must be trigger-first: "use when the user asks X, Y, Z" — not a summary of what the tool does). Body must stay under ~500 lines / <5k tokens, with decision procedure (table) and 2-3 worked examples first, tool catalog second or moved to resources.

- **MCP Resources API (go-sdk v1.7.0)** — The `Server.AddResource(resource, handler)` / `AddResourceTemplate(template, handler)` API is already stable in the pinned SDK version. For a static doc set (8 tools + a handful of concept pages), use `AddResource` per document, one call per doc. Register unconditionally at server startup (not gated on `.codegraph/` existing — reference content is useful whether or not an index resolves). Content is served verbatim as `text/markdown` per the resource's `MIMEType` field.

- **Claude Code plugin `hooks/hooks.json`** — PascalCase events (`SessionStart`, `PreToolUse`, `UserPromptSubmit`), matcher field per-event (tool name for PreToolUse, startup/resume for SessionStart, no matcher for UserPromptSubmit), command-type hooks as plain shell scripts. Distribution: write via `codegraph install` into `~/.claude/hooks/hooks.json` or `.claude/hooks/hooks.json` (location scope reuses existing global/local distinction). The single-highest-risk finding: a separate hooks.json schema exists for Cursor (camelCase), Codex CLI (PascalCase, different event set), Antigravity (missing SessionStart/UserPromptSubmit), and Kiro — no one schema works across all 8 agents. **Scope hooks delivery to Claude Code only for v1; port to other agents as a documented follow-up.**

**Multi-agent findings (corrects v0.4/v0.5 assumptions):**
- Claude Code: Skills ✓, Hooks ✓ (PascalCase, `SessionStart`/`PreToolUse`/`UserPromptSubmit`)
- Cursor: Skills ✓ (same `.cursor/skills/` standard), Hooks ✓ (camelCase, different event set)
- Codex CLI: Skills ✓ (`.agents/skills/`), Hooks ✓ (PascalCase, requires trust review, different event set)
- opencode: Skills ✓ (reads `.claude/skills/` as fallback), no hooks
- Antigravity: Skills ✓ (newly confirmed, `.agents/skills/`), Hooks ✓ (different event set: no `SessionStart`/`UserPromptSubmit`)
- Kiro: Skills ✓, Hooks ✓ (PascalCase, `PreTaskExec`/`PostTaskExec` additions)
- Gemini: Skills unclear, Hooks ✓ (schema not fully verified)
- Hermes: Neither confirmed

**The SKILL.md is genuinely portable across Claude/Cursor/Codex/opencode with zero changes.** Hooks are not — each agent needs its own `hooks.json` translation.

### Expected Features

**Must have (table stakes) — Phase 1:**
1. **MCP Resources capability** — Serves tool-by-tool usage docs, `CODEGRAPH_MCP_TOOLS` semantics, index-state preconditions. This is the missing "Phase 3" hand-off, and everything else in this milestone points at it.
2. **SKILL.md decision-procedure-first content** — A crisp table ("which tool for which question?") + 2-3 worked examples (including the actual 2026-08-08 misdirection scenario), tool catalog moved to resources. Not a catalog masquerading as a decision procedure.
3. **Rewritten `instructions` wire-string** — Short, correct, points at skill + resources instead of the broken "Phase 3" promise.
4. **SessionStart soft nudge** — Cheap (one-shot, `.codegraph/`-gated file-existence check, not MCP round-trip), no false-positive surface (fires once, never blocks).
5. **Claims guarded by tests** — Every numeric claim (tool count, default value, env var name) in skill/resources/instructions is either derived from code constants or checked by an automated test. No hand-typed facts without a gate.

**Should have (competitive) — Phase 2:**
1. **PreToolUse guard redirecting narrow, high-confidence grep-as-code-search patterns** — Not aggressive (default soft-warn, not hard-deny), triggers only on "looks like where-is-X", scoped to first binary in pipe (echo | grep bar passes through), never blocks Read. Paired with false-positive corpus test AND true-positive corpus test.
2. **`codegraph install` writing skill+hooks into target agent directories** — Versioned with the binary, refreshed by `codegraph upgrade`. Uses shared helper pattern reusing existing `internal/fsatomic`.

**Defer (v2+):**
1. **Resource `subscribe`/`listChanged`** — Only needed once tool roster changes mid-session; static per-process today.
2. **Auto-redirect argument rewriting** — Adds false-positive risk with incremental UX gain; defer until evidence shows soft-warn insufficient.
3. **Hook telemetry** — Explicitly out of scope; conflicts with zero-passive-phone-home claim.

### Architecture Approach

The architecture fits cleanly into existing codebase with minimal new packages and zero new import boundaries violated:

**MCP layer (new, minimal):**
- `internal/mcp/resources.go` — New file in existing package. Registers resources via `AddResource`/`AddResourceTemplate`. Resource catalog derived from `companionNames`/`allToolNames()`.
- `internal/mcp/resourcedocs/*.md` — Hand-authored reference markdown, `go:embed`'d. One file per resource.
- `internal/mcp/resources_contract_test.go` — Drift guard (mirrors existing `instructions_contract_test.go`). Two checks: numeric-claim pinning, tool-name cross-check.

**Agent distribution layer (new, minimal):**
- `internal/agents/skillfiles.go` — Shared helper reusing existing `atomicWriteFile`/`recordFile` machinery.
- `internal/agents/skillfiles/<target>/...` — Embedded directory trees (SKILL.md + hooks.json + scripts), one per supporting agent.

**Wire-oracle regression (critical, one-step re-capture):**
- Add `resources/list`/`resources/read` scenarios to `scenarios.go`, re-run capture tool, accept diff as one deliberate unit per `MUTATION-PROOF.md`.

**`instructions` wire-string rewrite (last, not first):**
- Rewrite to defer to skill + resources. **Must not merge before those actually exist.** Extend `instructions_contract_test.go` with fourth anchor.

### Critical Pitfalls

1. **The skill is inert** — reads well but doesn't change agent behavior
   - *Avoid:* Lead with decision table. Validate via fresh-session transcript diff on where-is-X prompt.

2. **MCP Resources drift from tool behavior** — repeats SURF-01
   - *Avoid:* Every numeric claim derived from code at test time. Extend wire-oracle to `resources/*`. Add `resources_contract_test.go`.

3. **PreToolUse guard fires false positives** — grep-is-not-always-wrong problem
   - *Avoid:* Scope narrowly (first-binary-only, never block Read). Soft-warn default. Test against legitimate grep corpus.

4. **PreToolUse guard too passive** — never actually fires
   - *Avoid:* Require both false-positive AND true-positive corpus tests before shipping.

5. **Install/uninstall corrupts new artifact types** — mirrors v1.0 Phase 5 data-loss
   - *Avoid:* Extend `AgentTarget` contract with shared tested helper. Per-agent round-trip tests asserting byte-invariance.

6. **Instructions string repeats Phase 3 broken-promise** — the bug this milestone was invented to fix
   - *Avoid:* Gate rewrite behind actual existence (test assertion). **Sequence: resources + skill verified working BEFORE rewrite.**

## Implications for Roadmap

Suggested 8-phase structure with critical-path sequencing:

### Phase 1: MCP Resources Capability
Resources must exist before skill points to them. Everything depends on this.
- Delivers: `resources.go`, `resourcedocs/*.md`, catalog derived from source constants
- Research flags: None — official stable API

### Phase 2: Wire-Oracle Re-Capture
Resources capability adds `capabilities.resources` — must capture as one deliberate commit before using resources elsewhere.
- Delivers: Updated scenarios, re-captured `.golden` transcripts
- Research flags: None — proven pattern in repo

### Phase 3: Resources Claims Drift Guard
Must ship same phase as resources, not follow-up. Ungated resources worse than none.
- Delivers: `resources_contract_test.go` with numeric-claim + tool-name checks
- Research flags: None — extends existing `tools_schema_drift_test.go` pattern

### Phase 4: SKILL.md Authoring
Can parallelize with 1–3 (disjoint package), but cannot ship until resources exist.
- Delivers: Decision-procedure-first skill, 2-3 worked examples, transcript-diff verification
- Research flags: Behavior verification (transcript diff) is non-standard — add to acceptance criteria

### Phase 5: SessionStart Soft Nudge Hook
Low-risk, high-value. Cheap file-existence check, cannot false-positive.
- Delivers: `hooks.json` + `session-nudge.sh`, passing hook tests
- Research flags: None — documented in official Claude Code docs

### Phase 6: Rewrite `instructions` Wire-String
Must happen last, after resources + skill verified working. Never name something that doesn't exist.
- Delivers: Updated `instructions` constant, updated marker block, extended `instructions_contract_test.go`
- Research flags: None — mirrors existing pattern

### Phase 7: `codegraph install` Skill + Hooks Distribution
The real distribution decision. Bundles via `go:embed`, versioned with binary, refreshed by `upgrade`.
- Delivers: `skillfiles.go` helper, per-target install/uninstall, per-agent round-trip tests (all 8 agents)
- Research flags: **Highest-risk phase given v1.0 Phase 5–6 install/uninstall history.** Recommend deep-review pass.

### Phase 8: PreToolUse Guard Hook (P2, post-validation)
Ship after skill+resources+nudge proven sufficient. Friction-adding fallback, not first lever.
- Delivers: `redirect-guard.sh`, false-positive + true-positive corpus tests, soft-warn default
- Research flags: Guard heuristics domain-specific — recommend corpus-building + rehearsal in phase plan

### Phase Ordering Rationale

1. Resources before skill — skill references resource URIs
2. Wire-oracle re-capture before downstream — one deliberate commit, keeps test suite green
3. Drift guard same phase as resources — ungated resources repeat known bugs
4. Skill parallels 1–3 — disjoint package, no upstream dependency until resources exist
5. Nudge before instructions — instructions points to both; need both working before rewriting pointer
6. Instructions rewrite last — never name something that doesn't exist
7. Install distribution after skill stabilizes — avoids shipping moving targets through install
8. Guard hook after Phase 1–7 validation — friction-adding fallback; only add if evidence shows nudge insufficient

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | **HIGH** | Official go-sdk API verified against vendored source; SKILL.md standard cross-checked across multiple agents; hooks.json schema confirmed for Claude Code; multi-agent findings from live documentation |
| Features | **HIGH** | Table stakes/anti-features grounded in official Anthropic docs + this repo's PROJECT.md incident log; prior-art hooks (Pare, jCodeMunch) cross-checked; ecosystem consensus on resources vs tools |
| Architecture | **HIGH** | Every recommendation anchored to actual codebase files; wire-oracle re-capture discipline direct from project's MUTATION-PROOF.md; archtest patterns verified |
| Pitfalls | **HIGH** | Grounded in this repo's documented incident history (MCP-01, SURF-01, Phase 3 promise, v1.0 Phase 5–6 install/uninstall bugs). Not theoretical — already shipped and paid down |

**Overall confidence:** **HIGH**

### Gaps to Address

1. **Per-agent hooks.json schema differences** — Cursor (camelCase), Codex/Antigravity (different event sets), Kiro (new events). Scope to Claude Code v1; document per-agent porting as P2.

2. **Behavioral verification methodology** — Skill-authoring phase must include fresh-session transcript-diff test proving grep → codegraph_explore redirect. Standard guidance doesn't emphasize this; add to acceptance criteria.

3. **Guard heuristics corpus definition** — Define what constitutes realistic "where-is-X" prompt corpus vs legitimate "non-where-is-X grep" corpus in Phase 8 plan, not during implementation.

4. **opencode/Gemini/Hermes skills scope** — Antigravity + Kiro confirmed; others unclear. Document v1 scope before Phase 4.

5. **Multi-agent SKILL.md portability test** — Load SKILL.md in each client's skill UI in Phase 4 before shipping, not follow-up.

## Sources

### Primary (HIGH confidence)

- **Official MCP spec** (`modelcontextprotocol.io/specification/2026-07-28`) — resources wire shapes, capabilities
- **go-sdk v1.7.0 source** (vendored) — `AddResource` API, `ServerCapabilities.Resources` auto-derivation
- **Claude Code official docs** (`code.claude.com/docs/`) — SKILL.md, hooks.json, plugins
- **Anthropic engineering blog** ("Equipping agents for the real world with Agent Skills") — best practices, why catalogs fail
- **This repo's own documents** — PROJECT.md (incident log), scoping todo, test patterns

### Secondary (MEDIUM confidence)

- **Cross-agent skills/hooks research** — Cursor, Codex, Antigravity, Kiro live docs
- **Prior-art hooks** — Pare, jCodeMunch GitHub implementations
- **MCP ecosystem guidance** — WorkOS, DVNC, llmbestpractices articles

### Tertiary (LOW–MEDIUM confidence)

- **Gemini/Hermes skills/hooks** — docs not fully traced; re-verify before Phase 7 scope
- **opencode plugin details** — SKILL.md fallback confirmed; in-process system details incomplete

---

*Research completed: 2026-08-12*
*Ready for roadmap: yes*
