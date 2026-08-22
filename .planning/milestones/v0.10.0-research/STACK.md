# Stack Research

**Domain:** Agent-onboarding skill/plugin + MCP Resources capability + enforcement hooks, for an existing Go MCP server/CLI (codegraph-go)
**Researched:** 2026-08-12
**Confidence:** MEDIUM (official docs for Claude Code/MCP spec/go-sdk are current and cross-checked; per-agent hook/skill claims for the 4 non-Claude roster members are single-search-pass and should be spot-checked against their live docs before the skill ships, especially Antigravity/Kiro which have moved fast in 2026)

This milestone adds **no new Go module dependencies**. Everything needed already exists in the repo's `go.mod` (`modelcontextprotocol/go-sdk@v1.7.0`) or is plain-text authoring (SKILL.md, hooks.json, shell scripts) that `codegraph install` writes to disk, exactly like it already writes the `<!-- CODEGRAPH_START -->` marker block. This file is about *format/schema*, not packages.

## Recommended Stack

### Core Technologies (already in the repo — no version bump needed)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/modelcontextprotocol/go-sdk` | v1.7.0 (pinned, current in `go.mod`) | MCP Resources capability (`resources/list`/`resources/read`) | `(*mcp.Server).AddResource` / `AddResourceTemplate` are stable, documented API on the exact version this repo already runs — SPEC-05's `AddTool`/`RemoveTools` re-check pattern in `internal/mcp/server.go` extends directly to `AddResource`/`RemoveResources` with no new import |
| Claude Code Agent Skills format (SKILL.md + YAML frontmatter) | current as of v2.1.21x-era docs (verify `claude --version` at ship time) | Teaches WHEN/HOW to use codegraph's tools | This is the format the milestone goal names explicitly, and — new finding this pass — is also the **same open standard** (agentskills.io) Cursor, Codex CLI, and Antigravity now read natively. One SKILL.md authored once is NOT Claude-Code-only |
| Claude Code plugin `hooks/hooks.json` | current schema (`SessionStart`/`PreToolUse`/`UserPromptSubmit` events) | SessionStart nudge + PreToolUse/UserPromptSubmit guard toward `codegraph_explore` | Matches the milestone's named events exactly; `command`-type hooks are plain shell scripts, no runtime to bundle |

### Format-only additions (no library — files `codegraph install` writes)

| Artifact | Location convention | Purpose | Why this shape |
|----------|---------------------|---------|-----------------|
| `SKILL.md` | `skills/codegraph/SKILL.md` (plugin) or `~/.claude/skills/codegraph/SKILL.md` (standalone fallback) | Decision-procedure-first tool guidance | Frontmatter `name`+`description` only (~100 tokens, always in context); body under ~1,500-2,000 words, decision table + 2-3 worked examples first, tool-by-tool catalog last or moved to a resource |
| `hooks/hooks.json` | plugin root | SessionStart nudge, PreToolUse/UserPromptSubmit guard | `{"description": "...", "hooks": {"SessionStart": [...], "UserPromptSubmit": [...]}}` — plugin wrapper form, not the bare `settings.json` direct form |
| `.claude-plugin/plugin.json` | plugin root's `.claude-plugin/` subdir ONLY | Plugin manifest (name/description/version/author) | Required if distributing as an installable plugin rather than a standalone `.claude/skills/` copy; unlocks `/codegraph:*` namespacing and `/plugin marketplace` distribution |
| MCP `resources/list` + `resources/read` handlers | `internal/mcp/resources.go` (new file, same package) | Detailed reference content the skill points to instead of embedding (tool-by-tool docs, `CODEGRAPH_MCP_TOOLS` semantics, index-state preconditions) | Keeps SKILL.md lean; content lives server-side so it can be derived/tested (guarding the "never hand-type numbers into prose" requirement) rather than baked into two separate static text blobs |

## MCP Resources — go-sdk v1.7.0 API surface

The Go SDK's resource API (from `design/design.md` and `docs/server.md`, both current for the version this repo pins) is a direct structural parallel to the tool-registration seam `internal/mcp/server.go` already built for SPEC-05:

```go
type ResourceHandler func(context.Context, *ServerSession, *ReadResourceParams) (*ReadResourceResult, error)

func (*Server) AddResource(*Resource, ResourceHandler)
func (*Server) AddResourceTemplate(*ResourceTemplate, ResourceHandler)
func (s *Server) RemoveResources(uris ...string)
func (s *Server) RemoveResourceTemplates(uriTemplates ...string)
```

- **Static vs. templated URIs:** `AddResource` registers one fixed URI (e.g. `codegraph://docs/tools/explore`); `AddResourceTemplate` registers an RFC 6570 URI *pattern* (e.g. `codegraph://docs/tools/{name}`) served by one handler for every matching URI. A template with no accompanying `list`-style enumeration only ever appears in `resources/templates/list`, never in `resources/list` — this repo's reference content (fixed tool count, fixed doc set) is fully enumerable, so **prefer `AddResource` per document over a template** unless the doc set becomes dynamic.
- **Content shape:** `ReadResourceResult.Contents []*ResourceContents{URI, MIMEType, Text}` for text (use `text/markdown` for the reference docs — matches SURF-06's existing MCP JSON→markdown conversion precedent) or `Blob` (base64) for binary; this milestone needs text only.
- **Capabilities:** `ServerCapabilities.Resources` must be set **explicitly**, same as the existing D-11 finding for `Tools` in `server.go` (`Server.capabilities()` only advertises a capability key when it's non-nil) — omitting it silently drops `"resources"` from the `initialize` response's capabilities object, exactly the "did the feature register or not" ambiguity D-11 already fixed once for tools. Wire it in the same `BuildServer` construction block: `Capabilities: &mcp.ServerCapabilities{Tools: ..., Resources: &mcp.ResourceCapabilities{ListChanged: false}}` (`ListChanged` false is correct here — the doc set changes with the binary, not with index state, so there's no live-mutation case Phase 3's `AddTool`/`RemoveTools` re-check pattern needs to mirror).
- **No new dependency, no new CGo, no new import boundary violation:** `AddResource`/`RemoveResources` live on the same `*mcp.Server` type `registerTools`/`unregisterTools` already hold; a `registerResources`/`unregisterResources` pair follows the identical shape and can sit in the same `internal/mcp` package without crossing D-08b's architest boundary.

## MCP spec compliance (2026-07-28, the revision this server already targets)

- `resources/list` and `resources/read` both support pagination (`nextCursor`) and the caching envelope (`ttlMs`, `cacheScope: "private"|"public"`) — this server already corrects `cacheScope` to `"private"` for `tools/list` and `server/discover` (D-09/D-03) because the tool catalog depends on local `.codegraph/` state; the new resources catalog does **not** have that dependency (reference docs are fixed per binary build, not per repo), so `resources/list`'s default `cacheScope: "public"` is actually correct here and should be left alone — do not blindly copy the D-09 correction.
- `resources/templates/list` is a separate optional method; only implement it if `AddResourceTemplate` is actually used (see URI-shape guidance above — likely unnecessary for a fixed doc set).
- Resource object fields available: `uri`, `name`, `title`, `description`, `mimeType`, `size`, `icons` — `title` (human-readable, distinct from the ID-like `name`) is new since `2024-11-05` and worth using for a friendlier resource listing (e.g. `name: "tools/explore"`, `title: "codegraph_explore reference"`).

## Claude Code Skill authoring — current conventions (verified against `code.claude.com/docs`, `platform.claude.com/docs`, and the shipped `anthropics/claude-code` `plugin-dev` skills, which are the same convention this repo's own `plugin-dev:skill-development`/`hook-development` skills already surface)

### SKILL.md frontmatter (only `description` is truly required; `name` strongly recommended)

| Field | Required | Notes |
|-------|----------|-------|
| `name` | No (defaults to directory name) | Lowercase, hyphens, ≤64 chars, no "anthropic"/"claude" |
| `description` | Recommended (de facto required — Claude reads *only* this to decide whether to trigger) | ≤1024 chars per the Agent Skills spec; Claude Code's own listing truncates the combined `description`+`when_to_use` at 1,536 chars. **Third person, imperative, front-load the trigger phrases**: `"This skill should be used when the user asks to 'X', 'Y', or mentions Z."` |
| `when_to_use` | No | Extra trigger context, appended to `description`, counts toward the same 1,536-char cap |
| `disable-model-invocation` | No | Set `true` for a skill only invokable via `/codegraph` — **not** the right choice here, since the whole point is Claude reaching for it unprompted |
| `user-invocable` | No | Set `false` to hide from `/` menu but keep auto-loadable — worth considering if `/codegraph` as a manual command adds no value over auto-trigger |
| `allowed-tools` / `disallowed-tools` | No | Pre-approve/restrict tools while the skill is active for the current turn only |

### Progressive disclosure — the three levels (directly answers the milestone's "lead with decision procedure, minimal tool catalog" requirement)

1. **Level 1 (always in context, ~100 tokens):** `name` + `description` only.
2. **Level 2 (loaded on trigger, target <5k tokens / 1,500-2,000 words):** SKILL.md body. **This is where the decision-procedure-first structure goes** — a "which tool for which question" table plus 2-3 worked examples, per the todo's explicit design constraint. A full tool-by-tool catalog does NOT belong here at length.
3. **Level 3 (loaded only if Claude reads it / calls a script, unlimited):** `references/`, `scripts/`, `assets/` bundled in the skill directory, OR — the milestone's actual design choice — **MCP resources served by the codegraph server itself**, fetched via `resources/read` rather than a bundled file. This is a legitimate Level-3 substitute: it keeps the reference content live/derivable (satisfying "guard the claims" — the resource handler can read the same `companionNames`/`allToolNames()` this repo already treats as source of truth, rather than hand-typed prose) instead of a static file that drifts from the binary the way `internal/agents/instructions.go`'s stale "Phase 3" promise already did once.

### Common mistake this milestone must specifically avoid (per Anthropic's own docs and the todo's stated failure mode)

> "Most skills fail for one reason: the description reads like documentation instead of matching what you actually type." — the description is a **trigger router**, not a summary. It must contain the literal phrases an agent's own prompt-matching would see in a task like "where is X defined" / "how does Y work" — the exact failure class the 2026-08-08 debug session hit.

## Claude Code hooks.json — schema for SessionStart / PreToolUse / UserPromptSubmit

Plugin-form `hooks/hooks.json` (the shape `codegraph install` should write, distinct from the direct `settings.json` form):

```json
{
  "description": "codegraph availability nudge + grep/find redirect guard",
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/session-nudge.sh", "timeout": 5 }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/redirect-guard.sh" }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Grep|Bash|Read",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/scripts/redirect-guard.sh" }
        ]
      }
    ]
  }
}
```

- `SessionStart` matcher values: `startup`, `resume`, `clear`, `compact`, `fork` — filters on how the session began, not on repo state; the `.codegraph/`-exists gate belongs **inside** the script, not the matcher (the script should be a fast, silent no-op when no index resolves — mirroring `hasIndex`'s own MCP-03 zero-tools rule so the hook never nags a repo with no index).
- `PreToolUse`/`PostToolUse` matcher is a regex over `tool_name` (`Bash`, `Grep`, `Read`, or `Edit|Write`-style alternation) — this is the mechanism for "guard grep/find/Read on where-is-X questions."
- `UserPromptSubmit` has **no matcher support at all** — it fires on every prompt submission, unconditionally; any "is this a where-is-X question" filtering has to happen inside the hook script itself (e.g. a cheap keyword/regex check against the prompt text passed on stdin), not in the hooks.json matcher field.
- Hook input arrives as JSON on stdin (`tool_name`, `tool_input` for `PreToolUse`; `user_prompt` for `UserPromptSubmit`); a `PreToolUse` hook can return `hookSpecificOutput.permissionDecision: "ask"|"deny"|"allow"` plus `additionalContext` — this is the mechanism for a genuine *guard* (not just a nudge) that surfaces `codegraph_explore` as the better option before letting a matched `Grep`/`Bash grep` call through.
- `${CLAUDE_PLUGIN_ROOT}` is the load-bearing path variable for any bundled script reference — installed plugins are copied into a cache directory, so a hard-coded or relative path outside the plugin root breaks silently.

## Plugin structure & distribution — the milestone's real design decision

Full plugin layout (only `plugin.json` goes inside `.claude-plugin/`; everything else is plugin-root-level):

```
codegraph-plugin/
├── .claude-plugin/
│   └── plugin.json          # name, description, version, author
├── skills/
│   └── codegraph/
│       └── SKILL.md         # decision-procedure-first guidance
├── hooks/
│   └── hooks.json           # SessionStart nudge + PreToolUse/UserPromptSubmit guard
└── scripts/
    ├── session-nudge.sh
    └── redirect-guard.sh
```

**Distribution options, mapped onto the todo's open question:**

| Option | Mechanism | Fit for `codegraph install` |
|--------|-----------|------------------------------|
| Standalone skill only | `codegraph install` writes `SKILL.md` directly into `~/.claude/skills/codegraph/` (personal) or `.claude/skills/codegraph/` (project) | Simplest; matches the existing `AgentTarget` write-a-file pattern exactly (same shape as the `codegraphInstructionsBlock` marker injection today), but **gets no hooks** — Claude Code's hooks system for a *standalone, non-plugin* skill is limited to hooks declared in the skill/agent's own frontmatter (a narrower mechanism than `hooks/hooks.json`), which cannot express `PreToolUse` tool-name matching cleanly |
| Full plugin (`.claude-plugin/plugin.json` + `skills/` + `hooks/`) written to a fixed local directory, self-registered via project `.claude/settings.json`'s `extraKnownMarketplaces`/`enabledPlugins` | `codegraph install` writes the whole plugin tree under (e.g.) `~/.codegraph/claude-plugin/` and adds an `extraKnownMarketplaces` (pointing at that local directory, source type `"directory"` or a local git repo) + `enabledPlugins` entry to the target's `.claude/settings.json` | **Closes the milestone's stated design goal** (versioned with the binary, updated by `codegraph upgrade` — since `codegraph install`/`upgrade` already own writing agent config files, this is a natural extension of the existing `AgentTarget` pattern, not a new mechanism) AND is the only path that gets the hooks capability at all |
| In-repo, manual `--plugin-dir` load | Ship the plugin directory in the codegraph-go repo itself, document `claude --plugin-dir ./path/to/plugin` | Lowest engineering risk, but explicitly the option the todo calls "leaves install's output still deferring to something thin" — does not close the hand-off |

**Recommendation:** the full-plugin route is the only one that satisfies both "hooks work" and "distribution is versioned with the binary, updated by `codegraph upgrade`" — the two things the milestone goal names explicitly. It composes cleanly with the existing `AgentTarget` registry: Claude Code's target implementation gains a second write (plugin tree + `settings.json` entries) alongside its existing MCP-config write and `codegraphInstructionsBlock` marker injection, with the same idempotent install→uninstall round-trip discipline this repo already holds every other `AgentTarget` to.

## Other agent harnesses — this is emphatically NOT Claude-Code-only

This is the most consequential finding of this research pass, and it corrects an assumption embedded in this repo's own comments (`hermes.go`: "Hermes has no AGENTS.md-equivalent instructions convention"; `antigravity.go`/`kiro.go`: "Writes no instructions file of its own" / "Writes NO instructions file") — those were accurate for *instruction files* as of the Phase 6 research (v0.4/v0.5 era) but **skills and hooks are a materially different, newer surface**, and at least two of the three "no instructions" agents have since shipped one or both:

| Agent | Skills (Agent Skills open standard, SKILL.md) | Hooks | Notes |
|-------|-----------------------------------------------|-------|-------|
| **Claude Code** | Yes — canonical implementation | Yes — `hooks.json`, PascalCase events (`SessionStart`, `PreToolUse`, `UserPromptSubmit`, ...) | This milestone's primary target |
| **Cursor** | Yes — `.cursor/skills/` or `.agents/skills/`, same `SKILL.md` shape | Yes — `hooks.json`, but **camelCase** events (`sessionStart`, `preToolUse`, `beforeShellExecution`, `afterFileEdit`, `stop`) — schema is NOT drop-in compatible with Claude Code's | A second, real hooks target if this milestone's scope grows — but the event *names* differ, so the hook scripts (which read stdin JSON) likely still port, only `hooks.json` itself needs a per-agent variant |
| **Codex CLI** | Yes — `.agents/skills/` (repo) / `~/.agents/skills` (global), same open standard | Yes — `hooks.json` or inline `[hooks]` TOML in `config.toml`, PascalCase events closely matching Claude Code's set (`SessionStart`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, `UserPromptSubmit`, `Stop`, `SubagentStart/Stop`, `PreCompact`/`PostCompact`) | Requires **explicit hook trust** (hash-pinned review) before a non-managed hook runs — a `codegraph install`-written hook will sit untrusted until the user reviews it in `/hooks`; document this rather than assume it "just works" |
| **opencode** | Yes — native `skill` tool auto-discovers `SKILL.md`; critically, it *also reads `.claude/skills/` directly* as a documented Claude Code compatibility fallback | No `hooks.json` at all — instead an **in-process TypeScript/JS plugin system** (`opencode.json` `plugins[]`, `ctx.tool.hook(...)`, `ctx.session.hook(...)`) — structurally incompatible with a shell-script hooks approach | The skill can likely be shared **verbatim** via the `.claude/skills/` fallback path with zero opencode-specific work; hooks would need a bespoke JS plugin, out of scope for a "thin" milestone |
| **Gemini CLI** | Not found as a native mechanism (extensions use `contextFileName`/`GEMINI.md` instead) | Yes — `hooks/hooks.json` inside an extension directory (schema not fully captured this pass — low confidence, verify before implementing) | Lower priority; would need an extension package, not a skill |
| **Hermes** | Not found | Not found | Still appears to have no equivalent mechanism — the existing `hermesTarget` comment is likely still accurate here specifically |
| **Antigravity** | **Yes, newly confirmed** — `.agents/skills/` (workspace) or `~/.gemini/config/skills/` (global), explicitly the same open Agent Skills standard, progressive disclosure documented | **Yes, newly confirmed** — `hooks.json` in `.agents/` or `~/.gemini/config/`, but event set differs: `PreToolUse`/`PostToolUse`/`PreInvocation`/`PostInvocation`/`Stop` — **no `SessionStart`, no `UserPromptSubmit`** | This repo's `antigravityTarget` comment ("Writes no instructions file... shares `~/.gemini/GEMINI.md`") predates this — Antigravity has since grown a real skills+hooks system separate from the shared GEMINI.md context file. Re-verify against a live Antigravity install before relying on this |
| **Kiro** | Docs list "Skills" as a first-class feature (alongside Hooks, Custom Agents) in Kiro's own feature comparison, though the SKILL.md schema specifics weren't captured this pass — MEDIUM confidence it exists, LOW confidence on exact shape | **Yes, newly confirmed** — `.kiro/hooks/*.json`, schema closely resembling Claude Code's: `PostFileSave`, `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `SessionStart`, `Stop`, plus Kiro-specific `PreTaskExec`/`PostTaskExec` | Same correction as Antigravity — `kiroTarget`'s "Writes NO instructions file" comment is about the *marker-fenced instructions block* mechanism specifically and is still true for that; it does not mean Kiro lacks skills/hooks entirely |

**Implication for scope:** the milestone's stated goal ("Give agent harnesses... a thin, high-signal skill") is achievable for Claude Code, Cursor, Codex CLI, and (via the `.claude/skills/` fallback, free) opencode using **one shared SKILL.md** with zero or near-zero per-agent variation, since all four converge on the same open standard. Hooks are more fragmented — three different schemas observed (Claude Code/Codex PascalCase-with-`SessionStart`+`UserPromptSubmit`; Cursor camelCase; Antigravity PascalCase-without-those-two-events) — so a single hooks.json cannot be shared verbatim across agents even though the underlying shell scripts likely can. **Recommend scoping this milestone's hooks deliverable to Claude Code only** (as the todo's target already implies) and treating "port hooks.json to Cursor/Codex/Antigravity/Kiro" as a documented follow-up rather than in-scope now — each is a small, mechanical, per-agent hooks.json translation of the same two scripts, not new design work.

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| Bundling the full tool-by-tool reference as SKILL.md body text | Blows the ~1,500-2,000 word / <5k token Level-2 budget and duplicates content that can drift from the binary (the exact `instructions.go` failure this milestone exists to fix) | Serve it via the new MCP `resources/list`/`resources/read` capability; SKILL.md links to it |
| Hand-typing tool counts/defaults/flag names into SKILL.md or the resource content | This repo has already had two wire-contract-drift incidents from exactly this pattern (SURF-01's "default 5", the `instructions` visibility claim) — a skill is explicitly called out in the todo as "a third such surface" | Derive resource content from the same `companionNames`/`allToolNames()`/`ResolveCompanions` functions `internal/mcp/server.go` already treats as source of truth, and gate it with a test the way `instructions_contract_test.go` does |
| A single cross-agent `hooks.json` | Cursor uses camelCase event names, Antigravity's event set omits `SessionStart`/`UserPromptSubmit` entirely — no shared schema exists across the roster today | One `hooks.json` per agent that has the mechanism, sharing the same underlying shell scripts |
| Writing the plugin as a bundled Node/Python runtime component | Violates the repo's stated single-static-binary / no-bundled-runtime constraint | Plain `SKILL.md` (markdown), `hooks.json` (JSON), and POSIX shell scripts only — no interpreter dependency beyond what's already assumed present (`sh`) |
| Treating `AddResourceTemplate` as the default resource-registration path | Adds URI-template parsing complexity for a doc set (8 tools + a handful of concept pages) that is fully known and static at server-build time | `AddResource` per document, one call per doc, mirroring `registerTools`' explicit-loop style already in this file |

## Version Compatibility

| Package/Format | Compatible With | Notes |
|-----------------|------------------|-------|
| `modelcontextprotocol/go-sdk@v1.7.0` | MCP spec `2026-07-28` (already the server's declared/asserted protocol version per VRFY-02) | `AddResource`/`AddResourceTemplate` are part of the stable public API surface documented in the SDK's own `design/design.md` and `docs/server.md` — not an experimental/unstable feature gated behind a build tag |
| Claude Code Agent Skills frontmatter | Agent Skills open spec (agentskills.io) — 6-field cap (`name`, `description`, `when_to_use`, `disable-model-invocation`, `user-invocable`, `allowed-tools`/`disallowed-tools`) when authoring for cross-tool portability | Claude Code accepts all 6 plus Claude-Code-only extras (dynamic context injection via `` !`cmd` `` in body) — **do not use Claude-Code-only frontmatter fields** if the same SKILL.md is meant to be read by Cursor/Codex/Antigravity via their shared-standard support, or verify each target's frontmatter allowlist doesn't reject the extra keys (opencode, for instance, explicitly documents only 5 recognized fields and the behavior on an unrecognized key is unverified this pass) |
| Claude Code plugin `hooks/hooks.json` | Claude Code CLI/IDE/Desktop/web — all surfaces fire the same hook events per current docs | Distinct schema from Cursor's `hooks.json` (camelCase) and Antigravity's (different event set) — do not assume portability without translation |

## Sources

- `code.claude.com/docs/en/skills` (web/exa, MEDIUM confidence, official) — SKILL.md frontmatter reference, progressive disclosure levels, dynamic context injection
- `platform.claude.com/docs/en/agents-and-tools/agent-skills/overview` (web/exa, MEDIUM confidence, official) — required fields, 3-level loading table with token costs
- `github.com/anthropics/claude-code/blob/main/plugins/plugin-dev/skills/skill-development/SKILL.md` and `.../hook-development/SKILL.md` (web/exa, MEDIUM confidence, official first-party source — this is literally the skill this repo's own `plugin-dev:skill-development`/`hook-development` skills surface) — writing-style rules (third person, imperative), hooks.json plugin-wrapper format, event/matcher reference
- `code.claude.com/docs/en/hooks` and `code.claude.com/docs/en/plugins-reference` (web/exa, MEDIUM confidence, official) — full hook event table, matcher field-per-event table, plugin directory structure, `.claude-plugin/plugin.json` schema, marketplace.json schema
- `code.claude.com/docs/en/plugins` and `code.claude.com/docs/en/plugin-marketplaces` (web/exa, MEDIUM confidence, official) — plugin quickstart, `extraKnownMarketplaces`/`enabledPlugins` project-scope self-registration mechanism
- `modelcontextprotocol.io/specification/2026-07-28/server/resources` (web/exa, MEDIUM confidence, official spec — this is the exact protocol revision codegraph-go's server already targets) — resources/list, resources/read, resources/templates/list wire shapes, caching envelope
- `modelcontextprotocol/go-sdk` `design/design.md` and `docs/server.md` via Context7 (docs/context7, MEDIUM confidence, official repo source) — `AddResource`/`AddResourceTemplate`/`ResourceHandler`/`RemoveResources` API, `Example_resources` full worked example, `ServerCapabilities.Resources` explicit-set requirement (parallel to this repo's own D-11 finding for `Tools`)
- `cursor.com/docs/skills`, `cursor.com/docs/rules` (web/exa, MEDIUM confidence, official) — Agent Skills standard support, `.cursor/skills`/`.agents/skills` locations, camelCase hooks.json event names
- `developers.openai.com/codex/skills`, `.../codex/hooks`, `.../codex/config-reference` (web/exa, MEDIUM confidence, official) — `.agents/skills` locations, PascalCase hooks.json/inline-TOML schema, hook-trust review requirement
- `opencode.ai/docs/skills/`, `opencode.ai/docs/rules/`, `opencode.ai/v2/docs/build/plugins` (web/exa, MEDIUM confidence, official) — native `skill` tool, `.claude/skills/` compatibility fallback (load-bearing finding for zero-cost opencode support), in-process plugin/hook system as the non-hooks.json alternative
- `github.com/google-gemini/gemini-cli/blob/main/docs/extensions/reference.md` and `writing-extensions.md` (web/exa, LOW-MEDIUM confidence, official repo docs but hooks.json schema not fully traced this pass) — `gemini-extension.json` manifest, `hooks/hooks.json` existence confirmed but shape not captured
- `kiro.dev/docs/hooks/`, `kiro.dev/docs/steering/`, `kiro.dev/docs/getting-started/first-project/` (web/exa, MEDIUM confidence, official, dated 2026-08-06 — very current) — `.kiro/hooks/*.json` schema with PascalCase events including `SessionStart`/`UserPromptSubmit`, "Skills" named as a first-class feature
- `antigravity.google/docs/hooks`, `antigravity.google/docs/skills`, `antigravity.google/docs/rules-workflows` (web/exa, MEDIUM confidence, official) — confirmed Agent Skills standard support at `.agents/skills/`, `hooks.json` schema with `PreToolUse`/`PostToolUse`/`PreInvocation`/`PostInvocation`/`Stop` (notably missing `SessionStart`/`UserPromptSubmit`)
- `internal/agents/hermes.go`, `internal/agents/antigravity.go`, `internal/agents/kiro.go` (codebase, this repo — ground truth for current per-agent implementation and the Phase 6 research comments this pass partially corrects)

---
*Stack research for: Agent-onboarding skill/plugin + MCP Resources + hooks (codegraph-go v0.10.0 milestone)*
*Researched: 2026-08-12*
