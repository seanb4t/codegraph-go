# Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge - Context

**Gathered:** 2026-08-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 6 delivers two disjoint agent-facing content artifacts, authored and verified together: (1) **SKILL.md** — a decision-procedure-first skill teaching an agent when to reach for codegraph tools over grep/find/Read, and (2) a **SessionStart hook package** (`hooks.json` + a nudge script) that tells an agent codegraph is available at the moment a session starts in a `.codegraph/`-indexed repo. Both artifacts point at Phase 5's live resource URIs rather than restating their content.

**Not in scope:** `codegraph install` writing these artifacts into other repos' agent directories (Phase 7), the `instructions` wire-string rewrite (Phase 8), multi-agent hooks porting beyond Claude Code (v2, AGENT-04…07), and the PreToolUse guard hook (v2, GUARD-HOOK-01/02 — deferred fallback if skill + resources + nudge prove insufficient).

</domain>

<decisions>
## Implementation Decisions

### SKILL-03 verification method
- **D-01:** SKILL-03's "transcript diff, not asserted" proof is a **committed rehearsal transcript** — a real before/after session pair, captured and saved as a markdown/JSON artifact in the phase directory, reviewed once at ship time. Not re-run by CI; this codebase has no precedent for driving a real agent session from an automated test, and research (`PITFALLS.md`) already flags this as needing a concrete mechanism, not "skill reviewed."
- **D-02:** The "before" half of that pair **reuses the 2026-08-08 misdirection debug log** (`.planning/debug/resolved/mcp-server-one-tool-only.md`) rather than a freshly captured "before" — it's a real historical incident with full root-cause documentation already on file. Pair it with one fresh "after" capture: same class of prompt, skill installed. — **Reversibility:** reversible — if the fresh capture later needs re-doing, nothing about this choice is load-bearing elsewhere.

### Dogfooding & canonical source location
- **D-03:** SKILL.md and hooks.json are **dogfooded into this repo's own `.claude/`** — `.claude/skills/codegraph/SKILL.md` and `.claude/hooks/hooks.json` (+ nudge script), committed. This repo's `.claude/CLAUDE.md` currently reports "No project skills found"; Phase 6 ends that. Dogfooding also makes the SKILL-03 rehearsal transcript possible without extra scaffolding — a real session in a real repo.
- **D-04:** `.claude/` **IS the canonical source**, not a copy of one authored elsewhere. Phase 7's `go:embed` directive will point directly at `.claude/skills/codegraph/` and `.claude/hooks/hooks.json`. One copy, no duplication/drift risk between an "authoring" location and an "installed" location. — **Reversibility:** costly — **rationale:** Phase 7's distribution mechanism (`go:embed` source path) depends on this location; relocating later means re-pointing the embed directive and re-verifying install/uninstall round-trips across all 8 agent targets Phase 7 covers.

### SKILL-02 worked examples
- **D-05:** Three worked examples, in this order:
  1. **The 2026-08-08 misdirection incident** — an agent grepped first, was misled by the `instructions` string's index-state framing, when the real gate was `CODEGRAPH_MCP_TOOLS`. Full root cause in `.planning/debug/resolved/mcp-server-one-tool-only.md`. (Doubles as D-02's "before" transcript.)
  2. **Impact analysis before a refactor** — "what breaks if I change this function's signature?" via `codegraph_impact` instead of manual call-site grepping.
  3. **Cross-file symbol lookup across dynamic dispatch** — "where is X defined / how does Y work" through an interface implementation or dynamic-dispatch hop, the case grep structurally cannot follow but `codegraph_explore`'s call-graph resolution can.

### NUDGE-01/02 message content & cadence
- **D-06:** Nudge text is a **minimal one-line pointer**: this repo has a codegraph index — prefer `codegraph_explore` / `codegraph explore` over grep for where-is-X / how-does-Y questions. No tool list, no examples, no fallback-syntax reminder — that detail lives in the skill, not the nudge.
- **D-07:** The nudge **fires on both SessionStart matchers** (`startup` and `resume`) — each gets exactly one nudge for that event. No cross-event suppression logic needed; "one-time" means one-time per matcher, not one-time per session lifetime.

### Claude's Discretion
- Exact markdown structure/headers within SKILL.md beyond "decision table first, worked examples second, tool catalog deferred to Phase 5 resources" (per roadmap success criterion 2 and research's decision-procedure-first framing).
- How the nudge script performs its `.codegraph/`-existence check (shell one-liner vs. a small Go helper) — mechanism, not vision; NUDGE-02 constrains it to a single file-existence check, no MCP round-trip, no index read.
- Exact hooks.json event/matcher JSON shape — follows the Claude Code hooks schema documented in `.planning/research/SUMMARY.md` (PascalCase events, matcher field per-event).
- Where within `.claude/skills/codegraph/` the rehearsal transcript artifact itself is filed (e.g., alongside SKILL.md vs. a `verification/` subdirectory) — planner's call.

### Folded Todos
- **`2026-08-08-author-a-codegraph-usage-skill-for-agents.md`** (score 0.9, area: agents) — the origin ticket for this phase. Its problem statement (no skill teaches agents when to use codegraph; the marker block and `instructions` string both defer to a hand-off that was never built) and its solution sketch (research current skill-authoring conventions, decide the distribution question, lead with decision procedure not catalog, guard the claims) directly shaped D-01 through D-07 above. Close this todo once Phase 6 ships.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge" — goal, 5 success criteria, notes on portability (SKILL.md) vs. non-portability (hooks.json)
- `.planning/REQUIREMENTS.md` §"Agent Skill" and §"SessionStart Nudge" — SKILL-01/02/03, NUDGE-01/02 full text

### Origin incident & todo (folded)
- `.planning/debug/resolved/mcp-server-one-tool-only.md` — the 2026-08-08 misdirection incident: full root cause, timeline, and evidence. Canonical source for SKILL-02's first worked example and SKILL-03's "before" transcript.
- `.planning/todos/pending/2026-08-08-author-a-codegraph-usage-skill-for-agents.md` — folded origin ticket (see Folded Todos above)

### Milestone-level research (already produced by earlier project research)
- `.planning/research/SUMMARY.md` — Agent Skills open standard details (frontmatter fields, `description` trigger-first requirement, ~500-line/<5k-token budget), Claude Code `hooks/hooks.json` schema, per-agent skill/hook support matrix
- `.planning/research/PITFALLS.md` — "Inert skill" and "transcript-diff verification is non-standard" pitfalls (directly informs D-01/D-02); nudge false-positive/false-negative guidance (informs D-06/D-07); claims-drift-guard warning
- `.planning/research/ARCHITECTURE.md` — resource-before-skill sequencing rationale, wire-oracle re-capture discipline (Phase 5, not this phase, but explains why SKILL.md can safely reference `codegraph://` URIs now)

### Phase 5 (dependency — must be live)
- `.planning/phases/05-mcp-resources-capability-claims-drift-guard/05-CONTEXT.md` — URI naming SKILL.md must reference, not restate: `codegraph://tools/<name>` (explore, node, search, callers, callees, impact, files, status), `codegraph://tools-filter`, `codegraph://index-state`. Also D-01: resource docs are fact-sheet-only, no worked examples — SKILL.md owns worked examples, resources own facts. This division must not blur.

### Existing code (marker-fence convention, not modified by this phase)
- `internal/agents/instructions.go` — current marker-fenced block agents read today; references the "Phase 3" deferral this milestone retires in **Phase 8**, not this phase. Read for context on the existing pattern; do not edit.
- `internal/mcp/tools_schema_drift_test.go`, `internal/mcp/instructions_contract_test.go` — the claims-drift-guard pattern (derive-from-source, never hand-type) that any numeric/tool-count claim in SKILL.md or the nudge must follow.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 5's `codegraph://` resource URIs are the source of truth for per-tool facts — SKILL.md's decision table and worked examples reference them by URI rather than duplicating params/defaults/return-shapes (mirrors Phase 5's own D-01 anti-duplication decision, extended to this phase's content too).
- `internal/agents/instructions.go`'s marker-fence constants (`codegraphSectionStart`/`codegraphSectionEnd`) are the existing convention for marker-fenced agent-facing blocks in this repo — informative precedent, not a file this phase touches.

### Established Patterns
- **Claims-drift-guard discipline** (`tools_schema_drift_test.go`, `instructions_contract_test.go`): every existing guard reads both the claimed value and the expected value from real source, never hardcoding either side. Any tool count, default, or env-var name SKILL.md or the nudge states must follow this same shape — PITFALLS.md explicitly warns this would be a third occurrence of the "wire-contract claim drifts from behavior" bug class if skipped.
- **Live/rehearsal verification precedent**: this repo already accepts "executed against a live system, evidence captured" as valid proof alongside automated Go tests (e.g., v0.5.0 Phase 4's `04-06-01` TMPDIR-based live verification). D-01's committed rehearsal transcript follows this precedent rather than inventing a new verification category.

### Integration Points
- `.claude/skills/codegraph/SKILL.md` and `.claude/hooks/hooks.json` (+ nudge script) are new files in this repo's own currently-empty agent-config directory.
- Phase 7 will later `go:embed` these exact paths for distribution into other repos via `codegraph install` — this phase's file locations are load-bearing for that later phase (see D-04's reversibility rating).

</code_context>

<specifics>
## Specific Ideas

- The nudge message is deliberately minimal — a single line pointing at the skill/tools, no restated tool catalog or fallback syntax (D-06).
- The misdirection incident worked example should read as a real incident, not a hypothetical — draw directly from the debug log's documented root cause and timeline (D-05.1).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

### Reviewed Todos (not folded)
None — the one matching todo was folded in (see Folded Todos above).

</deferred>

---

*Phase: 6-Agent Skill Package — SKILL.md & SessionStart Nudge*
*Context gathered: 2026-08-12*
