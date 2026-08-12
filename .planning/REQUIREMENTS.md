# Requirements: CodeGraph Go — v0.10.0 Agent Onboarding Skill & MCP Resources

**Defined:** 2026-08-12
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

**Milestone goal:** Give agent harnesses a thin, high-signal skill that teaches WHEN and HOW to use codegraph's tools — leading with a decision procedure, not a tool catalog — while pushing detailed reference content into MCP Resources served by the server itself, backed by a soft SessionStart nudge, so an agent stops reaching for grep first.

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### MCP Resources Capability

- [ ] **RSRC-01**: Agent client can call `resources/list` and see one resource for each of the 8 tools plus `CODEGRAPH_MCP_TOOLS` semantics and index-state preconditions
- [ ] **RSRC-02**: Agent client can call `resources/read` on any listed resource URI and receive the full reference doc as `text/markdown`
- [ ] **RSRC-03**: Resources register unconditionally at server startup — available even when zero index-gated tools are visible, so an unindexed repo can still teach an agent how the tools work

### Claims Drift Guard

- [ ] **GUARD-01**: Every tool count, default value, and env var name stated in resources/skill/instructions is derived from source constants or checked by an automated test — no hand-typed fact goes unguarded
- [ ] **GUARD-02**: Adding, removing, or renaming a tool fails a test if resource content wasn't updated to match

### SKILL.md Authoring

- [ ] **SKILL-01**: SKILL.md leads with a decision table ("which tool for which question") before any tool catalog
- [ ] **SKILL-02**: SKILL.md includes 2-3 worked examples, including the class of misdirection incident (2026-08-08 debug session) that motivated this milestone
- [ ] **SKILL-03**: An agent given a fresh session, the skill installed, and a "where is X" prompt selects `codegraph_explore` over grep/find/Read — verified by transcript diff, not asserted

### SessionStart Nudge

- [ ] **NUDGE-01**: On session start in a `.codegraph/`-indexed repo, the agent receives a one-time, low-noise nudge toward codegraph tools (file-existence check only, no MCP round-trip)
- [ ] **NUDGE-02**: The nudge never fires, and adds no overhead, in a repo without `.codegraph/`

### Instructions Rewrite

- [ ] **WIRE-01**: The MCP `initialize` instructions string correctly defers to the skill + resources instead of the stale "Phase 3" promise in `internal/agents/instructions.go`
- [ ] **WIRE-02**: The `codegraph install` marker block matches the rewritten instructions and promises nothing not yet shipped
- [ ] **WIRE-03**: This rewrite ships only after RSRC and SKILL are verified working — never names something that doesn't exist yet

### Install Distribution (Claude Code only, v1)

- [ ] **AGENT-01**: `codegraph install` writes SKILL.md + hooks.json into Claude Code's skills/hooks directories, byte-identical and idempotent across install→uninstall round-trips
- [ ] **AGENT-02**: `codegraph uninstall` cleanly removes the skill+hooks files it installed without touching user-authored content
- [ ] **AGENT-03**: The skill/hooks package is versioned with the binary and refreshed by `codegraph upgrade`

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Guard Hook

- **GUARD-HOOK-01**: PreToolUse hook redirects narrow, high-confidence grep-as-code-search patterns toward `codegraph_explore` (soft-warn default, first-binary-in-pipe scoping only, never blocks `Read`)
- **GUARD-HOOK-02**: Guard hook is validated against both a false-positive corpus (legitimate grep use) and a true-positive corpus (where-is-X patterns) before shipping

### Multi-Agent Distribution

- **AGENT-04**: Skill + hooks porting to Cursor (camelCase hooks.json)
- **AGENT-05**: Skill + hooks porting to Codex CLI (different event set, requires trust review)
- **AGENT-06**: Skill + hooks porting to opencode (`.claude/skills/` fallback — verify frontmatter field compatibility first)
- **AGENT-07**: Skill + hooks porting to Antigravity (missing `SessionStart`/`UserPromptSubmit` events)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Hard-block PreToolUse guard (hard-deny, not soft-warn) | Research + prior art (GitHub issue `anthropics/claude-code#43191`) both flag hard-blocking as the anti-pattern that gets hooks disabled |
| Resource `subscribe`/`listChanged` | Tool roster is static per-process today; only needed once it changes mid-session |
| Auto-redirect argument rewriting (rewriting a grep call into an explore call automatically) | Adds false-positive risk for incremental UX gain over a soft-warn; defer until evidence shows soft-warn insufficient |
| Hook telemetry | Conflicts with the project's existing zero-passive-phone-home claim (`codegraph telemetry`) |
| Kiro, Gemini CLI, Hermes skill/hook support | Schemas not yet verified (Kiro), unclear (Gemini), or unconfirmed (Hermes) — revisit once the Claude-Code-only v1 slice is proven |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| RSRC-01 | TBD | Pending |
| RSRC-02 | TBD | Pending |
| RSRC-03 | TBD | Pending |
| GUARD-01 | TBD | Pending |
| GUARD-02 | TBD | Pending |
| SKILL-01 | TBD | Pending |
| SKILL-02 | TBD | Pending |
| SKILL-03 | TBD | Pending |
| NUDGE-01 | TBD | Pending |
| NUDGE-02 | TBD | Pending |
| WIRE-01 | TBD | Pending |
| WIRE-02 | TBD | Pending |
| WIRE-03 | TBD | Pending |
| AGENT-01 | TBD | Pending |
| AGENT-02 | TBD | Pending |
| AGENT-03 | TBD | Pending |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 0 (roadmap not yet created)
- Unmapped: 16 ⚠️

---
*Requirements defined: 2026-08-12*
*Last updated: 2026-08-12 after initial definition, informed by 4-agent research (STACK/FEATURES/ARCHITECTURE/PITFALLS) and SUMMARY.md's 8-phase suggested structure*
