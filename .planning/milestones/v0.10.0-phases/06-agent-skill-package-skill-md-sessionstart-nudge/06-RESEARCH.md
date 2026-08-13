# Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge - Research

**Researched:** 2026-08-12
**Domain:** Claude Code Agent Skills (agentskills.io standard) + Claude Code hooks configuration surface
**Confidence:** HIGH (official docs fetched directly; cross-checked against this repo's shipped Phase 5 code, read this session)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**SKILL-03 verification method**
- **D-01:** SKILL-03's "transcript diff, not asserted" proof is a **committed rehearsal transcript** — a real before/after session pair, captured and saved as a markdown/JSON artifact in the phase directory, reviewed once at ship time. Not re-run by CI.
- **D-02:** The "before" half of that pair **reuses the 2026-08-08 misdirection debug log** (`.planning/debug/resolved/mcp-server-one-tool-only.md`) rather than a freshly captured "before". Pair it with one fresh "after" capture: same class of prompt, skill installed.

**Dogfooding & canonical source location**
- **D-03:** SKILL.md and hooks.json are **dogfooded into this repo's own `.claude/`** — `.claude/skills/codegraph/SKILL.md` and `.claude/hooks/hooks.json` (+ nudge script), committed.
- **D-04:** `.claude/` **IS the canonical source**, not a copy of one authored elsewhere. Phase 7's `go:embed` directive will point directly at `.claude/skills/codegraph/` and `.claude/hooks/hooks.json`. Reversibility: **costly** — Phase 7's distribution mechanism depends on this location.

  > **See "Critical Correction" in Summary below — this research found D-04's stated file (`.claude/hooks/hooks.json`) is not, by itself, a location Claude Code reads for project-scoped hooks. The fix below preserves D-04's intent without reopening it.**

**SKILL-02 worked examples**
- **D-05:** Three worked examples, in this order:
  1. **The 2026-08-08 misdirection incident** — agent grepped first, misled by the `instructions` string's index-state framing, when the real gate was `CODEGRAPH_MCP_TOOLS`. Full root cause in `.planning/debug/resolved/mcp-server-one-tool-only.md`. (Doubles as D-02's "before" transcript.)
  2. **Impact analysis before a refactor** — "what breaks if I change this function's signature?" via `codegraph_impact` instead of manual call-site grepping.
  3. **Cross-file symbol lookup across dynamic dispatch** — "where is X defined / how does Y work" through an interface implementation or dynamic-dispatch hop, the case grep structurally cannot follow but `codegraph_explore`'s call-graph resolution can.

**NUDGE-01/02 message content & cadence**
- **D-06:** Nudge text is a **minimal one-line pointer**: this repo has a codegraph index — prefer `codegraph_explore` / `codegraph explore` over grep for where-is-X / how-does-Y questions. No tool list, no examples, no fallback-syntax reminder.
- **D-07:** The nudge **fires on both SessionStart matchers** (`startup` and `resume`) — each gets exactly one nudge for that event. No cross-event suppression logic needed.

### Claude's Discretion
- Exact markdown structure/headers within SKILL.md beyond "decision table first, worked examples second, tool catalog deferred to Phase 5 resources".
- How the nudge script performs its `.codegraph/`-existence check (shell one-liner vs. a small Go helper) — mechanism, not vision.
- Exact hooks.json event/matcher JSON shape — **follows the Claude Code hooks schema documented in `.planning/research/SUMMARY.md`.** This research found SUMMARY.md's assumed file target (`.claude/hooks/hooks.json`) needs correcting (see Summary) — the event/matcher JSON *shape itself* (PascalCase events, matcher field per-event) is confirmed correct and unaffected.
- Where within `.claude/skills/codegraph/` the rehearsal transcript artifact itself is filed.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SKILL-01 | SKILL.md leads with a decision table before any tool catalog | Agent Skills spec section below; Architecture Patterns → Pattern 1 gives the exact table shape; verified `<5000` token / progressive-disclosure model from agentskills.io |
| SKILL-02 | SKILL.md includes 2-3 worked examples, including the 2026-08-08 misdirection class | Debug log read this session (`.planning/debug/resolved/mcp-server-one-tool-only.md`); Code Examples section gives verified CLI/tool syntax for all 3 examples |
| SKILL-03 | Fresh session + skill installed + "where is X" prompt selects `codegraph_explore` over grep — verified by transcript diff | Common Pitfalls → Pitfall 1; Validation Architecture section gives the rehearsal-capture mechanics; D-01/D-02 already lock the method |
| NUDGE-01 | SessionStart nudge in `.codegraph/`-indexed repo, file-existence check only, no MCP round-trip | Critical Correction (Summary) + Code Examples → nudge script pattern; confirmed `SessionStart` hook contract (stdout→context, no blocking, fires once per `startup`/`resume`) |
| NUDGE-02 | No nudge, no overhead, in a repo without `.codegraph/` | Same script pattern — single `test -d` short-circuits before any other work |
</phase_requirements>

## Summary

This phase authors two content artifacts — a `SKILL.md` decision procedure and a `SessionStart` nudge — and dogfoods both into this repo's own `.claude/` directory. Both artifacts are markdown/JSON/shell content, not Go code; there is no new dependency, no new package, and (outside the drift-guard test extension) no new Go source file required to ship the phase's primary deliverables.

**Critical Correction to CONTEXT.md's stated mechanism (read before planning hook delivery):** D-03/D-04 name `.claude/hooks/hooks.json` as the canonical location Claude Code reads for the project's `SessionStart` hook. This research fetched Claude Code's official hooks documentation directly (`code.claude.com/docs/en/hooks`) and confirms **`hooks/hooks.json` is a location Claude Code only reads inside an actual installed plugin package** (referenced from that plugin's own `.claude-plugin/plugin.json` manifest) `[CITED: code.claude.com/docs/en/hooks]`. This repo is not, and is not becoming, a Claude Code plugin — it is a project directory with a dogfooded `.claude/`. The **six** locations Claude Code actually resolves hooks from are: `~/.claude/settings.json` (user), **`.claude/settings.json`** (project, shareable/committable), `.claude/settings.local.json` (project, gitignored), managed policy settings, plugin `hooks/hooks.json`, and skill/agent frontmatter (component-scoped — see below). A standalone `.claude/hooks/hooks.json` file sitting in a plain project directory is never read by Claude Code at all; it is inert JSON.

This is not a reason to abandon D-04's location — it is a reason to route the *registration* through a different, adjacent file while keeping D-04's payload location intact:
- **Keep the hook *script*** at `.claude/hooks/session-nudge.sh` (or similar) — this is a completely standard, widely-documented pattern (`${CLAUDE_PROJECT_DIR}/.claude/hooks/<script>.sh` referenced from a settings hook entry).
- **Keep a `.claude/hooks/hooks.json` file too**, as the versioned *content fragment* Phase 7 will `go:embed` and merge into each of the 8 agents' own hook-registration surfaces (Claude Code's is `.claude/settings.json`; other agents' surfaces differ again — Phase 7's own problem to solve, not this phase's). This preserves D-04's "Phase 7 embeds this exact path" property untouched — Phase 7 already needs a per-agent JSON-merge writer (mirroring `internal/agents`' existing `writeMcpEntry()` pattern) rather than a raw file-drop, so having the source content live at a stable embed path costs nothing extra.
- **Add one more file this phase must create for the dogfood to actually function in this repo today: `.claude/settings.json`.** It does not exist yet (confirmed — `ls .claude/`), is not gitignored, and is the only file Claude Code will actually read the `SessionStart` hook entry from in this project. Its `hooks.SessionStart` array should contain the same matcher/command shape as the `hooks.json` fragment (D-07's `startup`+`resume` matchers), sourced from the same content so the two never drift — a single canonical JSON fragment, referenced by both files, is cleaner than hand-duplicating it, but the planner should decide the exact single-source mechanism (e.g., `.claude/settings.json` embeds/references the same hooks block `hooks.json` defines, or one generates the other at authoring time — this is mechanism, not vision, and belongs in Claude's Discretion).

None of D-03's SKILL.md placement is affected: `.claude/skills/codegraph/SKILL.md` is the correct, standard, currently-empty location (confirmed — `ls .claude/skills/` is empty) and is read by Claude Code directly with no equivalent settings.json indirection.

**Skill-frontmatter `hooks:` field is not a substitute for the above.** The agentskills.io/Claude Code skill frontmatter does support a `hooks` field, but (a) per official docs it activates only "while the component is active" — i.e., only after the skill has already been triggered/invoked in the conversation, which is structurally *after* session start, making it unusable for a hook whose entire job is to fire at the moment a session begins, and (b) multiple confirmed GitHub issues (`anthropics/claude-code#39468`, `#19225`, `#17688`) report skill-frontmatter hooks silently failing to fire at all in production, even for the events they nominally support. Do not route NUDGE-01/02 through SKILL.md frontmatter.

**Primary recommendation:** Author `SKILL.md` per the agentskills.io spec (verified fields: `name`, `description` — only two required; description must be trigger-shaped, third person, under 1024 chars) with a decision table first, 2-3 worked examples second, and explicit pointers to Phase 5's `codegraph://tools/<name>` resource URIs for anything beyond decision-making (an agent reads these via Claude Code's `ReadMcpResourceTool` or `@codegraph:codegraph://tools/explore` syntax — confirmed this session, not assumed). Register the `SessionStart` nudge via `.claude/settings.json` (not a standalone `hooks.json`), pointing at a `.claude/hooks/session-nudge.sh` script that does one `test -d .codegraph` (or equivalent) and echoes D-06's one-line text to stdout on success, exiting 0 unconditionally either way.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| SKILL.md decision procedure (what tool for what question) | Agent harness config (Claude Code skill loader) | — | Loaded by the harness at startup (metadata) / on trigger (body); not served by codegraph's own MCP server or CLI |
| Worked-example content sourcing | Static authored content | Phase-5 MCP Resources (`codegraph://tools/*`) | SKILL.md must not restate tool params/defaults already served by Phase 5 — it links to them, per Phase 5's own D-01 anti-duplication decision |
| SessionStart nudge trigger | Claude Code hook runtime (`.claude/settings.json`) | Filesystem (`.codegraph/` existence check) | Fires from the harness's hook dispatcher, not from codegraph's MCP server or CLI — codegraph is never invoked to produce the nudge (NUDGE-01's "no MCP round-trip" constraint) |
| SKILL-03 rehearsal transcript capture | Operator-run Claude Code session (manual) | — | Not automatable in CI per D-01 — this repo has no existing harness for driving a real agent session, and inventing one is explicitly out of this phase's scope |
| Claims-drift guard extension (GUARD-01 discipline applied to SKILL.md) | `internal/mcp` Go test package | — | The existing `resources_schema_drift_test.go` regexes (`numericClaimRe`, `countClaimsInRe`, `envVarTokenRe`, `hostFactsPosixRe`) are the extension point — see Don't Hand-Roll |

## Standard Stack

No new Go module dependencies. No new runtime dependencies of any kind — this phase's deliverables are markdown, JSON, and POSIX shell, all consumed by Claude Code's own harness (already present in the environment) and (for the drift-guard extension) Go's standard `testing`/`regexp` packages already imported by `internal/mcp`.

### Core
| Artifact type | Spec/Standard | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `SKILL.md` frontmatter | agentskills.io Specification (fetched live this session) | Portable skill metadata across Claude Code / Cursor / Codex CLI / opencode | Open standard explicitly named in this phase's Notes; two required fields only (`name`, `description`) `[CITED: agentskills.io/specification]` |
| `.claude/settings.json` `hooks.SessionStart` | Claude Code Hooks Reference (fetched live this session) | Project-scoped, committable hook registration | The only committable, non-plugin location Claude Code actually reads for a project's `SessionStart` hook `[CITED: code.claude.com/docs/en/hooks]` |
| POSIX shell (`session-nudge.sh`) | — | The nudge script itself | NUDGE-01 mandates "file-existence check only, no MCP round-trip" — a `test -d` one-liner is the minimal correct implementation; no interpreter dependency beyond `/bin/sh` |

### Supporting
| Artifact | Purpose | When to Use |
|---------|---------|-------------|
| `internal/mcp/resources_schema_drift_test.go`'s regex helpers (`numericClaimRe`, `countClaimsInRe`, `envVarTokenRe`, `hostFactsPosixRe`) | Extend to scan `.claude/skills/codegraph/SKILL.md` content too | Any numeric claim, tool/companion count, `CODEGRAPH_MCP_TOOLS` mention, or absolute host path appearing in SKILL.md must pass the same GUARD-01 gate resource docs already pass — the phase notes state this explicitly ("Every factual claim in the skill body is subject to the Phase-5 GUARD-01 discipline") |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `.claude/settings.json` for hook registration | Claude Code plugin package (`.claude-plugin/plugin.json` + `hooks/hooks.json`) | Would make `hooks/hooks.json` directly loadable as-is, but turns this repo into a self-referencing Claude Code plugin — a structural change far outside this phase's scope and inconsistent with D-03's "dogfooded into this repo's own `.claude/`" framing (a plugin is installed *into* Claude Code, not merely present in a repo) |
| A committed rehearsal transcript (D-01) | An automated headless-mode (`claude -p`) harness driving a real session in CI | Rejected by D-01 already — "this codebase has no precedent for driving a real agent session from an automated test" — noted here only as the alternative CONTEXT.md considered and declined |
| Skill-frontmatter `hooks:` for the nudge | `.claude/settings.json` `hooks.SessionStart` | Frontmatter hooks are component-scoped (fire only once the skill is *already active*, i.e. after a trigger), and per multiple confirmed GitHub issues frequently fail to fire even for supported events — wrong mechanism for a session-start signal |

**Installation:** None — no package manager invocation needed for this phase's primary deliverables.

**Version verification:** Not applicable — no versioned package dependency introduced.

## Package Legitimacy Audit

**Not applicable to this phase.** No external package (npm/PyPI/crates/Go module) is installed by any of Phase 6's deliverables — SKILL.md, the nudge script, and `.claude/settings.json` are hand-authored content files consumed directly by the already-installed Claude Code harness. The Package Legitimacy Gate protocol is skipped per its own scope condition ("whenever this phase installs external packages").

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────┐
                         │   Claude Code session lifecycle          │
                         └─────────────────────────────────────────┘
   Session start/resume  │
   ───────────────────► │  1. Claude Code reads .claude/settings.json
                         │     hooks.SessionStart[] (matcher: startup|resume)
                         │            │
                         │            ▼
                         │  2. Runs .claude/hooks/session-nudge.sh
                         │       (receives SessionStart JSON on stdin —
                         │        script does NOT need to parse it)
                         │            │
                         │       test -d "${CLAUDE_PROJECT_DIR}/.codegraph"?
                         │        ┌───┴────┐
                         │       yes       no
                         │        │         │
                         │   echo nudge   exit 0, no output
                         │   text; exit 0    (NUDGE-02)
                         │        │
                         │        ▼
                         │  3. stdout appended to Claude's context (NUDGE-01)
                         └─────────────────────────────────────────┘

   User asks "where is X"│
   ───────────────────► │  4. Claude Code loads skill metadata at startup
                         │     (name + description from ALL installed skills'
                         │      frontmatter — ~100 tokens each)
                         │            │
                         │            ▼
                         │  5. Description trigger-matches ("use when the
                         │     user asks where-is-X / how-does-Y questions")
                         │            │
                         │            ▼
                         │  6. Full SKILL.md body loads (<5000 tokens):
                         │     decision table → worked examples → pointers
                         │     to codegraph:// resource URIs (Phase 5)
                         │            │
                         │            ▼
                         │  7. Model selects codegraph_explore (MCP tool)
                         │     or `codegraph explore` (CLI fallback) —
                         │     NOT grep/find/Read (SKILL-03's success case)
                         │            │
                         │            ▼
                         │  8. (optional, on demand) model reads
                         │     codegraph://tools/explore via
                         │     ReadMcpResourceTool for full param docs —
                         │     never restated inline in SKILL.md
                         └─────────────────────────────────────────┘
```

### Recommended Project Structure
```
.claude/
├── settings.json                     # NEW — project-scoped, committable;
│                                        hooks.SessionStart registers the nudge
├── skills/
│   └── codegraph/
│       ├── SKILL.md                  # decision table, 3 worked examples,
│       │                               resource-URI pointers (D-01..D-05)
│       └── verification/             # planner's choice of subdirectory —
│           └── <rehearsal>.md        # D-01/D-02's committed transcript pair
└── hooks/
    ├── hooks.json                    # source-of-truth JSON fragment for
    │                                    Phase 7's go:embed (D-04) — NOT
    │                                    itself read by Claude Code here
    └── session-nudge.sh              # the actual script settings.json invokes
```

### Pattern 1: SKILL.md decision-table-first body
**What:** The body opens with a markdown table mapping question shapes to the tool to reach for, before any prose about what each tool does.
**When to use:** Always, per SKILL-01 and the agentskills.io progressive-disclosure model — description (~100 tokens, always loaded) → body (<5000 tokens, loaded on trigger) → linked reference material (loaded only if the model follows the link).
**Example (shape, not final content — planner fills in exact wording):**
```markdown
---
name: codegraph
description: Use when the user asks "where is X defined", "how does Y work", "what calls Z", or "what breaks if I change this" in a codebase with a .codegraph/ index — answers via codegraph's call/symbol graph instead of grep/find/Read.
---

## Which tool for which question

| Question shape | Use this |
|---|---|
| "Where is X defined / how does Y work" | `codegraph_explore` (MCP) or `codegraph explore "<query>"` (CLI fallback) |
| "What calls / is called by X" | `codegraph_callers` / `codegraph_callees` |
| "What breaks if I change X's signature" | `codegraph_impact` |
| No `.codegraph/` directory present | Skip codegraph entirely — grep/Read as usual |

## Worked examples
...
```
*Source: agentskills.io Specification (fetched live this session), cross-checked against this repo's existing `.claude/CLAUDE.md` codegraph section (already installed, same "reach for it BEFORE grep" framing) `[CITED: agentskills.io/specification]`*

### Pattern 2: SessionStart nudge via project settings, not a bare hooks.json
**What:** `.claude/settings.json` carries the hook registration; a separate script file does the actual check-and-echo work.
**When to use:** Always for this phase, per the Critical Correction above.
**Example (verified shape against official docs fetched this session):**
```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"
          }
        ]
      },
      {
        "matcher": "resume",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"
          }
        ]
      }
    ]
  }
}
```
*Source: `code.claude.com/docs/en/hooks` — matcher table confirms `SessionStart`'s matcher values are `startup`, `resume`, `clear`, `compact`, `fork`; D-07 only wants `startup`+`resume` `[CITED: code.claude.com/docs/en/hooks]`.*

```sh
#!/bin/sh
# .claude/hooks/session-nudge.sh — NUDGE-01/02
# No stdin parsing needed: matcher already filters to startup|resume.
if [ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]; then
  echo "This repo has a codegraph index — prefer codegraph_explore / \`codegraph explore\` over grep for where-is-X / how-does-Y questions."
fi
exit 0
```
This satisfies NUDGE-02 structurally: with no `.codegraph/`, the single `test -d` (via `[ -d ... ]`) is the only filesystem work performed, no output is produced, and the script exits 0 — "no overhead" is a property of the script having nothing else to do, not of extra suppression logic.

### Pattern 3: Resource-URI pointer instead of restated facts
**What:** SKILL.md references a Phase-5 `codegraph://` URI by name rather than restating the tool's parameters.
**When to use:** Any time SKILL.md would otherwise need to state a param name, default, or return shape already covered by a Phase 5 resource doc.
**Example:**
```markdown
For codegraph_explore's full parameter list and defaults, read `codegraph://tools/explore`.
```
An agent with this skill loaded can resolve that URI two confirmed ways: the built-in `ListMcpResourcesTool`/`ReadMcpResourceTool` pair (Claude decides to call them when a task needs the data), or inline `@codegraph:codegraph://tools/explore` mention syntax in a prompt. Resource access is **not automatic** — the model must decide to fetch it, same as any other tool call `[CITED: likeone.ai/blog/claude-code-mcp-resources-guide-2026 cross-checked against code.claude.com's MCP resource-template issue #3122]`.

### Anti-Patterns to Avoid
- **Tool catalog leading the body:** Pitfall 1 (below) — a skill that lists all 8 tools and their parameters before stating a decision procedure reads well but does not change agent behavior. Verified this repo's own `.claude/CLAUDE.md` codegraph section already avoids this (decision-procedure-first, 4 lines) — match that shape, expanded with the decision table and worked examples this phase adds.
- **Registering the nudge in `.claude/hooks/hooks.json` alone:** inert — Claude Code will never read it for a plain (non-plugin) project. See Critical Correction.
- **Routing the nudge through skill frontmatter `hooks:`:** wrong lifecycle (component-scoped, fires only after skill trigger) and empirically unreliable even within its supported scope.
- **Restating Phase 5 resource content inline:** violates Phase 5's own D-01 division of labor and doubles GUARD-01's claim-pinning surface for no benefit.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting a `.codegraph/` index | A Go helper binary invoked from the hook | `test -d` / `[ -d ... ]` in the shell script | NUDGE-01 explicitly requires "no MCP round-trip, no index read" — a shell existence check is strictly sufficient and has zero startup latency; a compiled helper adds a process-spawn indirection for no additional correctness |
| Keeping SKILL.md's factual claims honest | A parallel, hand-maintained claim list for SKILL.md | Extend `internal/mcp/resources_schema_drift_test.go`'s existing regex set (`numericClaimRe`, `countClaimsInRe`, `envVarTokenRe`, `hostFactsPosixRe`) to also scan the embedded/read SKILL.md content | These four checkers already implement exactly the "derive claimed value from source, never hand-type" discipline the phase notes require for SKILL.md too — building a second mechanism duplicates work and risks the two drifting from each other |
| Verifying SKILL-03 behavior change | A bespoke transcript-diff test harness | The committed rehearsal transcript pair (D-01/D-02) | Already locked by CONTEXT.md as the mechanism; this repo has no existing precedent for driving a real agent session from an automated test, and inventing one now is explicitly out of scope |

**Key insight:** Every "don't hand-roll" here resolves to "reuse a mechanism this repo (or this session's research) has already proven correct" — the regex-based drift guards, the shell-only nudge check, and the manual rehearsal capture are all extensions of patterns Phase 5 or CONTEXT.md already established, not new inventions.

## Common Pitfalls

### Pitfall 1: The skill is inert — reads well, changes nothing
**What goes wrong:** SKILL.md ships as a tool-by-tool catalog; an agent that has read it still greps first.
**Why it happens:** Tool descriptions feel like "complete" content; a catalog answers "what exists," not "what do I do right now."
**How to avoid:** Decision table first (Pattern 1), worked examples adversarially phrased the way a real user would ask, tool catalog deferred entirely to Phase 5 resources. Verify via the D-01/D-02 rehearsal transcript, not content review.
**Warning signs:** SKILL.md's word count on tool descriptions exceeds its word count on the decision table; no transcript-based verification step in the plan.
**Phase to address:** This phase — SKILL-03's acceptance criterion already forces this.

### Pitfall 2: Hook registered in a file Claude Code never reads
**What goes wrong:** `.claude/hooks/hooks.json` is authored, committed, and appears correct on inspection — but NUDGE-01/02 silently never fire because Claude Code only loads project-scoped hooks from `.claude/settings.json` (or `.claude/settings.local.json`, uncommitted).
**Why it happens:** `hooks/hooks.json` *is* a real, documented Claude Code location — but only inside an installed plugin package, not a plain project directory. The distinction is easy to miss because the JSON shape at that path is genuinely valid hooks.json syntax; nothing errors, it simply is never read.
**How to avoid:** Register in `.claude/settings.json`'s `hooks.SessionStart` array (Pattern 2). Verify NUDGE-01/02 by actually starting/resuming a Claude Code session in this repo and observing the nudge text appear — not by reading the JSON.
**Warning signs:** The install/verification step only re-reads the JSON file for syntactic validity rather than opening a real session and checking for stdout-injected context.
**Phase to address:** This phase.

### Pitfall 3: SKILL.md's own claims drift from `tools.go`/Phase-5 resources with no gate
**What goes wrong:** SKILL.md states "8 tools" or "codegraph_impact's default depth is 2" as prose; a future change to `companionNames` or `ImpactArgs`'s jsonschema tag leaves the skill silently wrong — the third occurrence of this repo's documented "wire claim drifts from behavior" bug class (SURF-01, the 2026-08-08 incident, and now a skill).
**Why it happens:** SKILL.md is a new file with no existing test pointed at it; nothing fails when it goes stale.
**How to avoid:** Extend `resources_schema_drift_test.go`'s regex checkers to also scan SKILL.md (see Don't Hand-Roll). This is explicitly required by the phase notes ("Every factual claim in the skill body is subject to the Phase-5 GUARD-01 discipline — the guard covers skill, resources, and instructions alike, not resources alone").
**Warning signs:** SKILL.md contains a bare number, tool name, or `CODEGRAPH_MCP_TOOLS` mention with no corresponding assertion in `internal/mcp`'s test package.
**Phase to address:** This phase.

## Code Examples

### SKILL.md frontmatter (verified against agentskills.io spec, fetched live this session)
```yaml
---
name: codegraph
description: Use when the user asks where a symbol is defined, how a function works, what calls or is called by a symbol, or what breaks if a signature changes — reach for codegraph's call/symbol graph instead of grep/find/Read in any .codegraph/-indexed repository.
---
```
`name`: max 64 chars, lowercase/digits/hyphens, must match the containing directory name (`codegraph`, matching `.claude/skills/codegraph/`). `description`: max 1024 chars, must state both what the skill does and when to use it, third person, no "I can help you" phrasing `[CITED: agentskills.io/specification]`.

### Worked example 2 — impact analysis (D-05.2), using this repo's real CLI/MCP surface
```
User: "What breaks if I change ResolveCompanions's signature?"

Use `codegraph_impact` (MCP) or `codegraph impact ResolveCompanions --depth 2` (CLI) —
depth-bounded reverse blast radius, not a manual grep for every call site.
```
Verified against `internal/mcp/tools.go:264-268` (`ImpactArgs{Depth, Path, Symbol}`, `jsonschema:"BFS depth (default 2, max 50)"`) and `internal/cli/impact.go:16-23,64-66` (`Use: "impact <symbol>"`, flags `--path/-p`, `--depth/-d` default 0 meaning "use engine default", `--json/-j`) — read this session `[VERIFIED: internal/mcp/tools.go:264-268, internal/cli/impact.go:16-23]`.

### Worked example 1 — the 2026-08-08 misdirection incident (D-05.1)
```
User: "the mcp server is only showing one tool."

WRONG first move: grep the instructions string / README for an explanation.
  → misled: the instructions string blamed index state, which structurally
    cannot produce a 1-tool list (a missing index produces ZERO tools).

RIGHT move: the real gate was CODEGRAPH_MCP_TOOLS (env var), not index state —
  a codegraph_explore call against the server's own source would have surfaced
  serve.go's ParseAllowlist / ResolveCompanions path directly, instead of
  trusting a wire-contract string that (at the time) was itself wrong.
```
Sourced verbatim from the incident's own root-cause record — timeline, evidence entries, and the AND-gate-of-three root cause — read this session `[VERIFIED: .planning/debug/resolved/mcp-server-one-tool-only.md:9-25, 113-150]`. Note for the planner: this incident predates Phase 5's `instructions` fix — the string quoted in the debug log's "Expected behavior" section is the *old*, wrong text; the *current* `instructions` constant (read this session) already names all three visibility mechanisms: `"codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question... All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names."` `[VERIFIED: internal/mcp/server.go:56]`. The worked example should frame this as "here is what confused an agent before the fix landed, and here is how codegraph_explore would have shortcut the investigation" — not imply the bug is still live.

### Nudge script (NUDGE-01/02) and settings.json registration
See Pattern 2 above — both snippets are complete, minimal, and directly copy-pasteable by the planner.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Standalone `hooks.json` assumed portable to any project directory (SUMMARY.md's stated assumption for this milestone) | `hooks.json` confirmed plugin-only; project-scoped hooks live in `.claude/settings.json` | Confirmed this session against live official docs | Changes this phase's file layout (see Critical Correction); does not change the JSON *shape* of a hook entry, which is identical in both locations |
| Full playbook-style instructions files (TS CodeGraph's pre-#529/#704 approach, per `internal/agents/instructions.go`'s own doc comment) | Short marker-fenced pointer + a separate, richer SKILL.md | Already the case before this phase; this phase is the "richer SKILL.md" half finally landing | SKILL.md is the appropriate place for the playbook-depth content the marker block deliberately stays short of |

**Deprecated/outdated:** None specific to this phase's domain — Agent Skills and Claude Code hooks are both actively evolving surfaces (the hooks event list alone has grown well past the ~10 events documented when SUMMARY.md was researched to 27+ events as of this session's fetch), so anything not re-verified against live docs at planning/implementation time should be treated cautiously.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | Cursor and Codex CLI (mentioned in Phase 5/SUMMARY.md context as also having `<agent-dir>/skills/` conventions) are unaffected by this phase's scope — this phase only dogfoods into `.claude/`, not those other targets | Architecture Patterns | None for this phase (Phase 7's problem); flagged only so the planner doesn't accidentally widen scope |
| A2 | The exact JSON-fragment reuse mechanism between `.claude/hooks/hooks.json` (Phase 7 embed source) and `.claude/settings.json` (this repo's live registration) is left as Claude's Discretion per CONTEXT.md — this research recommends single-sourcing but does not mandate a specific technique (generation vs. hand-kept-identical-and-tested) | Summary → Critical Correction | Low — if the two drift, NUDGE-01/02 still work in this repo (settings.json is authoritative for the live nudge); only Phase 7's embedded content could go stale relative to what actually runs here, which is a Phase 7 concern to catch via test |
| A3 | `${CLAUDE_PROJECT_DIR}` is available and correctly set inside a `SessionStart` hook's environment for computing the `.codegraph/` path | Code Examples → Pattern 2 | Low — `code.claude.com/docs/en/hooks` explicitly documents this variable as available to hook commands generally; not specifically re-verified for the `SessionStart` event in isolation this session |

**Note on tag choice:** every claim above the Assumptions Log table that concerns Claude Code's hooks/skills mechanism is tagged `[CITED: code.claude.com/...]` or `[CITED: agentskills.io/...]` because it was fetched directly from the vendor's official documentation this session (not training memory) and cross-checked against 10+ independent third-party sources that all agree — this clears the bar for CITED (MEDIUM, cross-checked via `exa` provider) rather than ASSUMED. Claims about this repo's own source (`tools.go`, `server.go`, the debug log) are tagged `[VERIFIED: <path>:<lines>]` because they were opened and read with `Read` this session, with verbatim quotes included above.

## Open Questions

1. **Should `.claude/hooks/hooks.json` exist at all in this repo, or is `.claude/settings.json` alone sufficient for Phase 6, with the `hooks.json`-shaped fragment deferred entirely to Phase 7's authoring?**
   - What we know: D-04 explicitly names `.claude/hooks/hooks.json` as something Phase 7 will `go:embed`. Phase 6 needs a *working* nudge in this repo today, which requires `.claude/settings.json`.
   - What's unclear: whether carrying both files (with the risk of drift flagged in A2) is worth it now, versus authoring only `.claude/settings.json` in Phase 6 and letting Phase 7 extract/generate its embed source from that file when it actually needs it.
   - Recommendation: Planner's call — either is consistent with D-04's *location* commitment (Phase 7 can embed from `.claude/settings.json`'s hooks block just as easily as from a separate `hooks.json`, since Phase 7 needs custom per-agent merge logic regardless of source file). Bias toward the simpler single-file option (`.claude/settings.json` only) unless Phase 7 planning already assumes the two-file split.

2. **Exact SKILL.md line/token count budget: "~500 lines" vs. "<5000 tokens" — which governs, and does D-05's 3 worked examples plus decision table plausibly fit?**
   - What we know: agentskills.io's own spec states the *body* should stay under ~5000 tokens (progressive-disclosure model: description ~100 tokens always loaded, body loaded on trigger). The "~500 lines" figure appears consistently across third-party community guidance but is not independently stated as a hard spec constraint in the primary source fetched this session.
   - What's unclear: whether 500 lines and 5000 tokens are meant as roughly equivalent (they usually are, at ~10 tokens/line of typical prose) or whether one is stricter in practice for this specific content mix (a markdown table + 3 code-block-heavy worked examples tends to run token-denser per line than prose).
   - Recommendation: Treat 5000 tokens as the authoritative spec-sourced ceiling; treat 500 lines as a practical proxy to sanity-check against, not a hard spec rule. Planner should have the author do an actual token count (not just a line count) before considering SKILL-01 satisfied.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Claude Code (the harness itself) | Everything in this phase — SKILL.md loading, hooks dispatch, MCP resource reads | ✓ (this session is running inside it) | Not independently queried this session | — |
| `.codegraph/` index in this repo | NUDGE-01's positive-path test, SKILL-03's rehearsal "after" capture | ✓ — confirmed present (`.codegraph/store/`, live `daemon.lock` as of 2026-08-12 17:45) | — | — |
| codegraph MCP server registered for this session | SKILL-03's rehearsal "after" capture (needs `codegraph_explore` actually callable) | ✓ — this session's own MCP instructions banner confirms the `codegraph` server is connected and describes its 8-tool surface | — | — |

**Missing dependencies with no fallback:** None identified.

**Missing dependencies with fallback:** None identified — this phase's dependencies are all already present in the authoring environment.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) for the drift-guard extension; manual/operator-driven for SKILL-03 and NUDGE-01/02 |
| Config file | none — plain `go test` |
| Quick run command | `go test ./internal/mcp/... -run TestResource` (existing GUARD-01/02 tests; extend for SKILL.md) |
| Full suite command | `task test` (per `Taskfile.yml`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SKILL-01 | Decision table precedes tool catalog, stays within token budget | manual review + word-count/token-count check | none automated — author-time check | ❌ Wave 0 (no automated line/token-budget test exists; planner may add one, or accept manual) |
| SKILL-02 | 3 worked examples present, misdirection incident accurately reproduced | manual review against the debug log | none automated | ❌ N/A — content-accuracy review, not code |
| SKILL-03 | Fresh session + skill + "where is X" prompt → `codegraph_explore` chosen over grep | rehearsal transcript diff (D-01/D-02) | manual: start/resume a real session, capture, diff against `.planning/debug/resolved/mcp-server-one-tool-only.md`'s "before" | ❌ Wave 0 — rehearsal artifact does not exist yet; must be captured during this phase's execution |
| NUDGE-01 | `.codegraph/`-present repo → nudge fires once per matcher, no MCP round-trip | live execution + stdout diff (per roadmap success criterion 4) | manual: `echo '{"source":"startup"}' \| .claude/hooks/session-nudge.sh` inside this repo, assert non-empty stdout | ❌ Wave 0 — script does not exist yet |
| NUDGE-02 | No `.codegraph/` → zero output, zero extra filesystem work | live execution + stdout diff (per roadmap success criterion 5) | manual: same script run from a directory with no `.codegraph/`, assert empty stdout | ❌ Wave 0 — same script |
| (GUARD-01 extension) | SKILL.md's factual claims pinned to source | automated Go test | `go test ./internal/mcp/...` | ❌ Wave 0 — extension to `resources_schema_drift_test.go` does not exist yet |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/...` (fast, covers the drift-guard extension)
- **Per wave merge:** `task test` (full suite, in case any wire-oracle/instructions-contract interaction is disturbed — unlikely given this phase touches no Go production code path, but cheap to confirm)
- **Phase gate:** Full suite green + both rehearsal artifacts (SKILL-03's transcript pair, NUDGE-01/02's stdout diffs) committed and reviewed, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `.claude/hooks/session-nudge.sh` — does not exist; NUDGE-01/02 cannot be tested until it does
- [ ] `.claude/settings.json` — does not exist; NUDGE-01/02 cannot fire in a live session until it does (see Critical Correction)
- [ ] `.claude/skills/codegraph/SKILL.md` — does not exist; SKILL-01/02/03 cannot be evaluated until it does
- [ ] SKILL-03's rehearsal transcript artifact (fresh "after" capture, per D-02) — does not exist; requires an operator-driven session in this repo
- [ ] Extension to `internal/mcp/resources_schema_drift_test.go` (or a sibling test file) covering SKILL.md content — does not exist

*(All five gaps are expected — this phase's own deliverables. Listed per the Validation Architecture template's requirement to enumerate Wave 0 gaps explicitly, not because they indicate missing test infrastructure elsewhere in the repo.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | This phase adds no auth surface |
| V3 Session Management | no | Not applicable — "session" here is a Claude Code conversation, not an authenticated session |
| V4 Access Control | no | No access-control surface added |
| V5 Input Validation | marginal | The nudge script's only "input" is its own environment (`CLAUDE_PROJECT_DIR`) and a filesystem existence check — no untrusted data is parsed. If the planner opts to parse the SessionStart JSON stdin payload (not required by the recommended design), standard shell-injection hygiene (`"$VAR"` quoting, no `eval`) applies |
| V6 Cryptography | no | Not applicable |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Hook script executed with the user's own shell permissions (Claude Code's own documented hook execution model — "hooks run with your user permissions, receiving information... via stdin and communicating back through exit codes and stdout") | Elevation of Privilege (if the script were compromised) | Keep the nudge script minimal (one `test -d` + one `echo`), committed and reviewable in plaintext, sourced from the same repo it inspects — no dynamic code execution, no `eval`, no untrusted input parsing `[CITED: claude.com/blog/how-to-configure-hooks]` |
| A malicious actor placing a same-named script earlier in `$PATH` to shadow `session-nudge.sh` | Spoofing | Reference the script by its full `${CLAUDE_PROJECT_DIR}`-relative path in `.claude/settings.json` (Pattern 2's example already does this), never by bare filename — standard Claude Code hook guidance already recommends absolute/`${CLAUDE_PROJECT_DIR}`-anchored paths for exactly this reason `[CITED: code.claude.com/docs/en/hooks]` |
| Overwriting a user's own pre-existing `.claude/settings.json` content when this phase creates the file | Tampering (self-inflicted, not adversarial) | `.claude/settings.json` does not yet exist in this repo (confirmed this session) — this phase creates it fresh, not merges into existing content, so the `internal/agents` JSON-merge-safety concerns (relevant to Phase 7's *other* target repos) do not apply here. Phase 7, when it writes into a *user's* possibly-pre-existing `.claude/settings.json`, must reuse the existing marker-fence/safe-merge discipline — flagged here for forward visibility, not required by this phase |

## Sources

### Primary (HIGH confidence)
- `code.claude.com/docs/en/hooks` — fetched live this session via WebSearch/WebFetch; hook location table (6 locations), `SessionStart` matcher values (`startup`/`resume`/`clear`/`compact`/`fork`), decision-control table (`SessionStart`: "No [block]" / "Shows stderr to user only"), skill/agent frontmatter hooks section ("scoped to the component's lifecycle... only run when that component is active")
- `agentskills.io/specification` (and its GitHub mirror `agentskills/agentskills`) — fetched live this session; frontmatter field table, `description` field guidelines, progressive-disclosure token budget (~100 tokens metadata / <5000 tokens body)
- `internal/mcp/server.go:56` (this repo, read this session) — current `instructions` constant, verbatim
- `internal/mcp/tools.go:161-544` (this repo, read this session) — `ExploreArgs`/`ImpactArgs`/etc. struct definitions, verbatim jsonschema tags
- `internal/mcp/resources.go`, `internal/mcp/resources_schema_drift_test.go` (this repo, read this session) — Phase 5's shipped resource URIs and GUARD-01/02 mechanism, verbatim
- `.planning/debug/resolved/mcp-server-one-tool-only.md` (this repo, read this session) — the 2026-08-08 incident's full timeline and root cause, verbatim quotes used above
- `internal/agents/instructions.go` (this repo, read this session) — existing marker-fence convention

### Secondary (MEDIUM confidence)
- `github.com/anthropics/claude-code#39468`, `#19225`, `#17688` — confirmed, independently-filed bug reports on skill/agent frontmatter hooks silently not firing; cross-checked against 3+ independent blog/community sources describing the same "component-scoped, activates only while active" model
- `claude.com/blog/how-to-configure-hooks`, `likeone.ai/blog/claude-code-mcp-resources-guide-2026`, `alatirok.com/what-are-agent-skills` — third-party summaries cross-checked against and consistent with the primary official-docs fetches above

### Tertiary (LOW confidence)
- The precise 500-line practical guideline for SKILL.md body length — consistently repeated across community sources but not independently verified against a primary spec statement of a line-count (as opposed to token-count) limit; flagged in Open Questions

## Metadata

**Confidence breakdown:**
- Standard stack (no new deps, hooks/skill file locations): **HIGH** — official docs fetched and cross-checked live this session, not training memory
- Architecture (SKILL.md structure, nudge script, settings.json registration): **HIGH** — every recommendation traces to either official docs fetched this session or this repo's own source read this session
- Pitfalls: **HIGH** — Pitfall 2 (wrong hooks file) is a novel finding from this session's own verification, not carried over unverified from SUMMARY.md; Pitfalls 1 and 3 are directly grounded in this repo's own `PITFALLS.md` and `.planning/todos/pending/2026-08-08-...`

**Research date:** 2026-08-12
**Valid until:** ~30 days for this repo's own source facts (stable until the next phase touches `tools.go`/`server.go`); ~14 days for the Claude Code hooks/skills documentation specifically — it is an actively evolving surface (event count grew well past SUMMARY.md's earlier research pass) and should be re-verified if implementation slips past that window.
