# Feature Research

**Domain:** Agent-onboarding UX for a code-graph MCP server — SKILL.md/plugin, MCP `resources/`, SessionStart/PreToolUse hooks
**Researched:** 2026-08-12
**Confidence:** MEDIUM (official spec/doc sources for MCP resources and Claude Code hooks/skills; broader ecosystem patterns from cross-checked but individually LOW-tier web sources — see Sources)

## Feature Landscape

### Table Stakes (Users Expect These)

Features an agent-onboarding package is judged incomplete without.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Trigger-first `description` in skill frontmatter | It is the *only* thing pre-loaded at session start (name+description of every skill); a vague or workflow-summarizing description either never fires or fires wrong. The 2026-08-08 incident is a strict subset of this: the `instructions` wire-string is codegraph's only always-loaded surface today, and it was actively wrong. A skill adds a second always-loaded surface with the same failure mode | LOW | Third-person "Use when the user asks X, Y, Z" naming concrete trigger phrases (`where is X defined`, `how does Y work`, `what calls Z`) — not a summary of what `codegraph_explore` does internally |
| Decision procedure before tool catalog | The todo's own framing: "the failure mode to design against is an agent that has read the skill and still reaches for grep first." A tool-by-tool reference is the least valuable part per every skill-authoring source found; a "which tool for which question" table plus 2-3 worked examples is what changes behavior | LOW–MEDIUM | A single table (question shape → tool → why) beats prose. Anthropic's own guidance: "state what to do, not why" |
| SKILL.md under ~500 lines, detail pushed to one-level-deep reference files | Every skill-authoring source (Anthropic official, Claude Code docs, third-party guides) converges on this number and the one-level-deep rule specifically; violating "one level deep" causes Claude to `head -100` preview nested references instead of reading them fully | LOW | Maps directly onto the milestone's own plan: SKILL.md stays thin, `CODEGRAPH_MCP_TOOLS` semantics / tool-by-tool docs / index-state preconditions move to MCP resources instead of a second bundled reference file — this project has two progressive-disclosure tiers (skill files AND MCP resources) available, most skills only have one |
| 2-3 worked examples, not an exhaustive one | The todo explicitly names this ("2-3 worked examples") and it matches the ecosystem consensus: concrete examples over abstract descriptions, and examples anchored to real symptom phrasing (the actual 2026-08-08 "server shows one tool" framing is a gift here — a skill that includes *that exact* worked example directly prevents its own root-cause incident from recurring) | LOW | Worked examples should show the WRONG first instinct (grep/Read) and the redirect, not just the right answer — this is what teaches judgment, not documentation |
| MCP `resources/list` + `resources/read` capability | This milestone's core scope item. `go-sdk` supports it natively (`Server.AddResource`/`AddResourceTemplate`, `ResourceHandler` returning `ReadResourceResult`) — additive, does not touch the existing 8-tool surface or the tool-gating logic | LOW–MEDIUM | Server must declare the `resources` capability in `initialize`; `listChanged`/`subscribe` are optional and not needed here (codegraph's resource set is static per binary version, not per-session) |
| Resources hold reference material, not action surface | Ecosystem-wide rule, stated identically across every MCP resources/tools source found (official spec, WorkOS, llmbestpractices, DVNC): resources = read-only application/model-pulled context (docs, schemas, semi-static data); tools = model-invoked actions. `codegraph`'s 8 tools already do the "act" side — resources are purely the missing "explain yourself" side | LOW | Concretely: tool-by-tool usage docs, `CODEGRAPH_MCP_TOOLS` semantics/current allowlist, index-state precondition explanation. NOT a `codegraph_explore` result — that stays a tool call, it does real work |
| `instructions` wire-string rewritten to defer correctly | Named directly in scope. Today it makes a promise ("Phase 3") that was never kept and was actively wrong once. The fix is not "make it longer" — every skill/tool-description source agrees the initialize-time string is the cheapest, most-always-loaded real estate a server has and should therefore be terse and point outward, exactly like a skill description points outward to SKILL.md | LOW | `instructions` is the MCP-protocol equivalent of a skill's `description` field — same failure mode class, same fix class (short, correct, pointing at deeper material) |
| A soft SessionStart nudge, no hard PreToolUse block by default | Table-stakes on the "not naggy / not false-positive-prone" side of the milestone's own framing. Every real-world prior-art hook examined (Pare, jCodeMunch) that ships a *block* also ships careful scoping (first-binary-only in a pipe, `Read` never blocked, "warn not block" as the default posture) specifically because over-blocking is the top complaint pattern. `anthropics/claude-code#43191` (a live, open feature request) exists precisely because teams doing MCP-tool-vs-grep redirection today find hooks to be a *friction-adding fallback*, not a good primary mechanism | LOW–MEDIUM | SessionStart hook: print a 1-2 line stdout nudge ("this repo has a codegraph index; prefer `codegraph_explore` for where-is-X questions") gated on `.codegraph/` existing — cheap, no false positives possible since it only fires once and never blocks |

### Differentiators (Competitive Advantage)

Features that go beyond "has a skill file" and would set codegraph's onboarding apart from typical MCP-tool skill packages.

| Feature | Value Proposition | Complexity | Notes |
|---------|--------------------|------------|-------|
| PreToolUse guard that *redirects with a specific pointer*, not a generic denial | The GitHub-issue evidence (`#43191`) is explicit that the current-best pattern (a redirect hook) still costs a full block→read-error→retry round trip, but is strictly better than a bare "denied" message with no next step — the deny reason should name the exact tool+args to retry with (`mcp__codegraph__codegraph_explore` with the extracted query), turning the extra round trip into a single corrective step rather than a dead end | MEDIUM | Requires parsing `tool_input.command` for Bash-invoked `grep`/`rg`/`find`, extracting a plausible query term, and only firing when `.codegraph/` exists AND the tool surface includes `codegraph_explore` — false-positive surface is the risk, see Anti-Features |
| Scoped, narrow matcher — first-binary-in-pipe only, `Read` never touched | Directly differentiates from the naive "block all Grep/Bash" hooks that generate the most user complaints in the wild. Pare's and jCodeMunch's own docs call out this exact scoping decision as what separates a hook people keep from one they disable | LOW–MEDIUM | `echo foo | grep bar` must pass through untouched (grep isn't the *acting* command); `git grep`/build tooling that happens to invoke `grep` internally must not misfire |
| `codegraph_explore` fallback path taught explicitly (works without MCP server via `codegraph explore` shell command) | This is a genuinely differentiating fact about codegraph specifically (most MCP-only tools have no CLI-parity fallback), and the todo names it as "currently buried." Surfacing it in both the skill's decision table AND the SessionStart nudge means an agent that loses MCP access mid-session (or is a CLI-first agent like a raw shell loop) still gets the redirect | LOW | One line in the decision table: "MCP unavailable? Same tool via shell: `codegraph explore <query>`" |
| Resources exposing *live* state (current `CODEGRAPH_MCP_TOOLS` value, current tool roster), not static prose | Static instructions have already caused one real incident (wrong-because-stale). A resource that's generated from the live server config at `resources/read` time (not hand-typed prose) can never drift the way `instructions.go` did — this directly satisfies the todo's "guard the claims" requirement structurally rather than by adding a test that could itself go stale | MEDIUM | Resource content built from the same `AddTool`/`RemoveTools` state and `CODEGRAPH_MCP_TOOLS` parsing the tool-gating code already tracks — no new source of truth, just a new read path on an existing one |
| `codegraph install` writes the skill + hooks into the target agent's directory, versioned with the binary | The real design decision the todo flags. Doing this (vs. leaving it in-repo for manual install) is what makes the skill/hooks actually reach the same users the existing marker-block injection already reaches, and lets `codegraph upgrade` naturally refresh a stale skill the same way it refreshes the binary — closing the exact kind of drift that caused this milestone | MEDIUM–HIGH | Reuses the existing `AgentTarget` registry pattern (Phase 6) rather than inventing new distribution machinery; per-agent skill-directory conventions differ (Claude Code `.claude/skills/`, others vary) and are a real per-agent research/implementation cost, not a detail |

### Anti-Features (Commonly Requested, Often Problematic)

Features that look like the natural next step for "teach the agent to use my tool" but create the exact problems the source material warns about.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Hard PreToolUse block on all `Grep`/`Bash grep`/`find` calls whenever `.codegraph/` exists | Feels like the strongest guarantee — "just don't let it happen" | This is the exact anti-pattern named across every source examined: legitimate grep use exists inside an indexed repo (searching prose/config/logs, non-code content, or paths the index doesn't cover) and a hard block on all of it produces friction and false positives users disable the hook to escape. `anthropics/claude-code#43191`'s author explicitly built and then walked back a version of this ("PreToolUse hooks... This is enforcement rather than guidance — it adds friction... requiring runtime hook intervention") | Redirect only narrow, high-confidence cases: `grep`/`rg` invoked with what looks like a symbol/identifier search pattern, only when `codegraph_explore` is actually registered (not just `.codegraph/` present — the tool could be filtered out by `CODEGRAPH_MCP_TOOLS`), and default to a soft stderr nudge (exit 0, `additionalContext`) rather than exit-2 deny, at least initially |
| A tool-by-tool reference as the primary/first thing in SKILL.md | Feels complete — "document every tool so nothing is missed" | This is the named failure mode in the todo itself and independently confirmed by every skill-authoring source: an agent that reads a tool catalog first treats it as documentation to skim, not judgment to apply, and still reaches for grep. The catalog isn't wrong to have, it's wrong to lead with | Decision procedure (question-shape → tool) first in SKILL.md; full tool-by-tool reference moved to an MCP resource (this milestone already has that home for it) so it costs zero tokens until an agent actually asks for full detail |
| Embedding large/growing reference content (all tool docs, all flag semantics, `CODEGRAPH_MCP_TOOLS` full doc) directly in the `instructions` wire-string | The `instructions` string is the most "always there" surface, so the instinct is to put everything of value in it | The wire-string is sent on every `initialize`, so it is a permanent token tax on every session regardless of whether that session ever needs the detail — exactly the problem progressive disclosure exists to solve, just recreated at the protocol layer instead of the skill layer. It's also the surface that already went stale once | Keep `instructions` as short as a skill `description` (one to two sentences: what codegraph is, when to reach for `codegraph_explore`, and where the resources/skill live) and move everything else behind `resources/read` |
| Telemetry / usage analytics on hook fires (how often grep was redirected, which agents ignore the nudge) | Reasonable-sounding product instinct — "measure if this is working" | Out of scope for this milestone per its own stated goal ("agent-education UX, not new indexing/query capability"), and the project's `codegraph telemetry` command explicitly promises zero passive phone-home with `upgrade` as the sole network path (v0.5.0 Phase). A hook that phones home usage data contradicts a shipped, tested claim | If effectiveness needs to be measured, do it locally/manually (does a debug session still misdirect on grep?) — not via a new network path |
| Auto-redirect that silently rewrites the tool call (`updatedInput`) instead of asking the agent to retry | Sounds like it removes the extra round trip entirely — best of both worlds | `updatedInput` on a `PreToolUse` hook can rewrite Bash *arguments*, but cannot swap `Bash(grep ...)` for a call to a different tool (`mcp__codegraph__codegraph_explore`) — those are different tools with different schemas, not different arguments to the same tool. Attempting a same-tool-family "rewrite" that isn't really equivalent (e.g. quietly turning a `grep` into a `rg` call) fixes the wrong problem and hides the actual redirect from the transcript, making the guard harder to debug when it misfires | Deny with a specific, actionable reason (tool name + suggested args) via `permissionDecisionReason`, and let the agent make the corrective call itself — visible in the transcript, matches the pattern every real prior-art hook actually ships |

## Feature Dependencies

```
MCP resources/list + resources/read capability (go-sdk AddResource/AddResourceTemplate)
    └──requires──> existing tool-gating state (CODEGRAPH_MCP_TOOLS parsing, tool roster) as the resource's data source
                       └──requires──> the mcp-server-one-tool-only fix already landed (PROJECT.md: sequence after it)

SKILL.md decision procedure
    └──enhances──> nothing upstream required, but its worked examples reference resource URIs
                       └──requires──> resources capability shipped first (so the skill can point at something real, not a promise — repeats the exact "Phase 3" mistake otherwise)

instructions wire-string rewrite
    └──requires──> resources capability + skill both exist (it points to both)
    └──conflicts──> leaving internal/agents/instructions.go under its prior "must not change" constraint (PROJECT.md flags this needs re-examining, not silently violating)

PreToolUse guard hook
    └──requires──> decision procedure content already exists (the deny-reason text should quote/match the skill's own worked examples, not invent separate wording)
    └──requires──> knowing whether codegraph_explore is actually registered (CODEGRAPH_MCP_TOOLS-aware), not just whether .codegraph/ exists

codegraph install writing skill+hooks to target agent directories
    └──requires──> SKILL.md + hooks finalized and versioned with the binary
    └──enhances──> the existing AgentTarget registry / marker-fenced instruction injection (Phase 6) — same mechanism, new payload
```

### Dependency Notes

- **Resources must ship before (or atomically with) the skill's worked examples reference them.** The todo's central complaint is a hand-off that was promised ("Phase 3") and never delivered; a skill that again points at "detailed reference content" that doesn't exist yet repeats the identical failure. Sequence resources first or in the same phase as the skill, never after.
- **The PreToolUse guard depends on the skill's decision-procedure wording, not the reverse.** If the guard's redirect message is authored independently, the milestone risks a third divergent surface (the todo already names two: the marker block and the `instructions` string, both thin/wrong). The guard should quote or directly reference the same "which tool for which question" language the skill uses.
- **The guard also depends on tool-gating state, not just index presence.** `.codegraph/` existing is necessary but not sufficient — if `CODEGRAPH_MCP_TOOLS` has narrowed the surface to exclude `explore`-adjacent tools (though `codegraph_explore` itself is always-visible per the MCP server instructions text), a naive guard could redirect toward a tool that isn't actually registered for this session. This is a real edge case worth a plan-time decision, not an afterthought.
- **`codegraph install` distribution conflicts with the standing "must not change" constraint on `internal/agents/instructions.go`** — PROJECT.md explicitly calls this out as needing re-examination rather than silent violation. This is a plan-time decision point, not a research gap.

## MVP Definition

### Launch With (v1 of this milestone)

- [ ] MCP `resources/list` + `resources/read` on the Go MCP server (go-sdk `AddResource`/`AddResourceTemplate`), serving tool-by-tool docs, `CODEGRAPH_MCP_TOOLS` semantics, and index-state preconditions — the missing "Phase 3" hand-off, and the thing everything else in this milestone points at
- [ ] SKILL.md (or Claude Code plugin) leading with a decision-procedure table (question shape → tool → why) and 2-3 worked examples including the actual 2026-08-08 misdirection scenario, tool catalog demoted to a pointer at the new resources
- [ ] Rewritten `instructions` wire-string: short, correct, points at skill + resources instead of a broken "Phase 3" promise
- [ ] SessionStart hook: cheap, one-shot, `.codegraph/`-gated nudge toward `codegraph_explore` — no blocking, no false-positive surface possible
- [ ] Claims (tool counts, `CODEGRAPH_MCP_TOOLS` defaults, flags) in skill/resources/hooks are derived from live server state or gated by a test, per the todo's explicit "guard the claims" requirement

### Add After Validation (v1.x)

- [ ] PreToolUse guard redirecting narrow, high-confidence grep/find-as-code-search patterns toward `codegraph_explore`, soft-warn by default (not hard-deny) — add once the skill+resources are proven to reduce misdirection on their own, since a hook is explicitly the friction-adding fallback layer per the ecosystem's own experience report (`#43191`), not the first lever to pull
- [ ] `codegraph install` writing the skill/hooks package into target-agent skill directories, versioned with the binary and refreshed by `codegraph upgrade` — the real distribution decision the todo flags; can follow once the skill content itself is stable, since shipping a moving target through the install mechanism multiplies the surface that can drift

### Future Consideration (v2+)

- [ ] Resource `subscribe`/`listChanged` support (e.g. notifying a client when the tool roster changes mid-session) — not needed while the roster is static per-process; only relevant once index rebuilds or config reload can change registered tools live
- [ ] Auto-redirect / argument-rewriting hook sophistication (e.g. extracting a query term from a `grep` invocation to pre-fill the suggested `codegraph_explore` call) — real value, but adds guard complexity and false-positive risk that should wait for evidence the soft-warn version isn't sufficient
- [ ] Any usage telemetry on hook fires — explicitly out of scope; conflicts with the project's zero-passive-phone-home `codegraph telemetry` claim

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|----------------------|----------|
| MCP resources capability (list/read) | HIGH | MEDIUM | P1 |
| SKILL.md decision-procedure-first content | HIGH | LOW | P1 |
| `instructions` wire-string rewrite | HIGH | LOW | P1 |
| SessionStart soft nudge hook | MEDIUM | LOW | P1 |
| Claims-guarded-by-test discipline | HIGH | LOW–MEDIUM | P1 |
| PreToolUse redirect guard (soft-warn) | HIGH | MEDIUM | P2 |
| `codegraph install` skill/hook distribution | HIGH | MEDIUM–HIGH | P2 |
| PreToolUse hard-block variant | LOW | MEDIUM | P3 (anti-feature, do not build by default) |
| Live tool-roster resource (vs. static prose) | MEDIUM | LOW (reuses existing state) | P1 (cheap enough to fold into the base resources work) |
| Auto-redirect argument rewriting | LOW–MEDIUM | HIGH | P3 |
| Hook telemetry | LOW | MEDIUM | Out of scope |

**Priority key:**
- P1: Must have for this milestone
- P2: Should have, add once P1 is proven in real sessions
- P3: Future consideration / explicitly deferred

## Competitor / Prior-Art Feature Analysis

| Feature | Pare (`hooks/pare-prefer-mcp.sh`) | jCodeMunch (`AGENT_HOOKS.md`) | codegraph-go's Approach |
|---------|-----------------------------------|-------------------------------|--------------------------|
| Grep/Bash redirect scope | Matches only the first binary in a Bash pipe/chain (`echo foo | grep bar` passes) across 28 mapped server packages | Intercepts `Bash`/`Grep`/`Glob` when they "look like" code exploration; explicitly does NOT block `Read` because Edit/Write require a prior Read | Adopt the same narrow-scope discipline: first-command-only matching, never block `Read`, gate on both `.codegraph/` presence AND `codegraph_explore` actually being registered |
| Default posture | Hard deny (`permissionDecision: deny`) with a message pointing at the correct MCP tool | Two-tier: hard block for read/search (Read Guard), soft warn-and-pass for edit-adjacent (Edit Guard), explicitly documented as "may block legitimate edge cases" | Start softer than either — SessionStart nudge only for v1; if/when a PreToolUse guard ships (P2), default to soft-warn (`additionalContext`, exit 0) per this milestone's own "must not be naggy/false-positive-prone" bar, escalate to deny only with evidence |
| Where guidance lives | Shell-hook logic only; no skill/plugin layer described | Layered: "prompt policies" (CLAUDE.md-style soft rules) as basic setup, "tool hooks" as advanced/hard enforcement — explicitly two tiers of increasing strength | Matches this milestone's intended shape almost exactly: skill+resources = prompt-policy layer (teach), hooks = enforcement layer (guard), landed in that order |
| Root-cause acknowledgment | N/A (no discussion of *why* the model needs a hook at all) | "The common failure mode isn't forgetting — it's skipping. The agent sees the rule in CLAUDE.md and reaches for Read/Grep anyway... A prompt policy can't stop this." | Directly validates the todo's premise: a skill/CLAUDE.md-block alone is necessary but not sufficient; this is *why* the milestone also includes hooks, not redundant with them |

## Sources

- [Skill authoring best practices — Claude Platform Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices) (web, official, cross-checked — treat as HIGH despite seam-default LOW tier for the `exa` provider; this is Anthropic's own current documentation)
- [Extend Claude with skills — Claude Code Docs](https://code.claude.com/docs/en/skills) (web, official — HIGH)
- [Equipping agents for the real world with Agent Skills — Anthropic Engineering](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) (web, official Anthropic engineering blog — HIGH)
- [Hooks reference — Claude Code Docs](https://code.claude.com/docs/en/hooks) (web, official — HIGH)
- [Claude Code power user customization: How to configure hooks — Claude by Anthropic](https://claude.com/blog/how-to-configure-hooks) (web, official — HIGH)
- [Resources — Model Context Protocol spec (2026-07-28)](https://modelcontextprotocol.io/specification/2026-07-28/server/resources) (web, official protocol spec — HIGH)
- [modelcontextprotocol/go-sdk design.md + server.md](https://github.com/modelcontextprotocol/go-sdk) (Context7, MEDIUM confidence per classify-confidence seam; official SDK repo) — `AddResource`/`AddResourceTemplate`/`ResourceHandler` API, runnable example
- [\[FEATURE\] MCP tool preference / priority configuration — anthropics/claude-code#43191](https://github.com/anthropics/claude-code/issues/43191) (web, GitHub issue from a maintainer of a real codebase-indexer MCP server describing this exact problem — LOW tier per seam, but high relevance/specificity; corroborates the todo's own incident pattern independently)
- [Pare `hooks/README.md` — `pare-prefer-mcp.sh`](https://github.com/Dave-London/Pare/blob/main/hooks/README.md) (web, real shipped hook implementation — LOW tier per seam, used as concrete prior art, not as an authority claim)
- [jCodeMunch `AGENT_HOOKS.md`](https://github.com/jgravelle/jcodemunch-mcp/blob/main/AGENT_HOOKS.md) (web, real shipped hook implementation with documented rationale — LOW tier per seam, used as concrete prior art)
- [MCP Resources vs Tools: Production Rule — DVNC Dev](https://dvnc.dev/blog/mcp-resources-vs-tools-production-server) (web, LOW tier per seam, cross-checked against the official spec and WorkOS source below — consistent)
- [Designing an MCP server from a REST API — WorkOS](https://workos.com/blog/designing-mcp-server-from-rest-api) (web, LOW tier per seam, practitioner guide consistent with official spec)
- [MCP: Resources — llmbestpractices](https://llmbestpractices.com/ai-agents/mcp-resources) (web, LOW tier per seam, consistent with official spec)
- `.planning/PROJECT.md` and `.planning/todos/pending/2026-08-08-author-a-codegraph-usage-skill-for-agents.md` (project-internal — source of truth for scope, the 2026-08-08 incident, and standing constraints)

---
*Feature research for: agent-onboarding UX (skill/plugin + MCP resources + hooks) on codegraph-go*
*Researched: 2026-08-12*
