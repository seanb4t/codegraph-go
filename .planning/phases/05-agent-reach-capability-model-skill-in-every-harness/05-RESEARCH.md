# Phase 5: Agent Reach — Capability Model & Skill in Every Harness - Research

**Researched:** 2026-09-18
**Domain:** Multi-harness agent-config installer (in-repo `internal/agents` file-writing logic) + current per-harness skill/instructions documentation for 5 external coding-agent products (Cursor, opencode, Antigravity, Gemini CLI, Kiro)
**Confidence:** HIGH for everything read directly from this repo's source this session (`[VERIFIED: path:lines]`); HIGH for per-harness doc claims fetched as raw primary-source markdown today (`[CITED: url, fetched 2026-09-18]`) — several of these **correct** the milestone-level `STACK.md`'s 2026-09-14 WebSearch-synthesized claims, see "Critical Findings" below; LOW for anything that requires a live session on a harness not installed on this machine (Gemini CLI, Kiro — `[ASSUMED]` per D-10, ticketed for a live probe if the maintainer installs them).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Cross-cutting**
- **D-00:** The v0.13.0 Phase 12 ruling and Phase 4's D-00 apply: tests assert **our** contract — which paths we write, which bytes we leave, what the table declares — never a harness's behaviour and never Go dependency management. Whether a harness actually *reads* a path is established by a live session (D-09…D-12), not by a unit test. Every new guard is positive-controlled (rule `84d1gfpywd`): a planted violation goes RED, recorded in `05-MUTATION-LOG.md` in the Phase 2/3/4 shape with a byte-clean revert. TDD mode is on: Go RED evidence is the `test(05-NN):` commit before the implementation plus the pasted `--- FAIL:` transcript (rule `x1cjy9vyhq`).

**Capability table (AGENT-08)**
- **D-01:** The table lives on the targets: a new `Capabilities() Capabilities` method on `AgentTarget`, returning **one struct literal per target file** — additive, mirroring `SupportsLocation` (ROADMAP note). No central map in `registry.go`. `Install`/`Uninstall` semantics do not change because of the table's existence.
- **D-02:** `Capabilities` declares: `Scopes []Location`; per-location `MCPConfig` path; per-location `Instructions` path (`""` = none); per-location `SkillDirs` (ordered; the first entry is the one written, later entries are documented read-paths only); `Hooks` enum {`none`, `claude-json`, `codex-json`}; and `ConfigFormat` (json/jsonc/toml/yaml — the "style" in `--print-config-style`). `SupportsLocation`, `DescribePaths`, `Detect`'s path inputs and `--print-config-style` become **derivations** of the table — no target keeps a second hand-written copy of any path the table holds.
- **D-03:** The guard (ours): one table test over all eight targets × both locations asserting `DescribePaths(loc)` equals the path set derived from `Capabilities()` and that `--print-config-style` renders exactly the table; a planted divergence in one target (a path in `DescribePaths` the table does not hold, or vice versa) goes RED — Family (a) in `05-MUTATION-LOG.md`.
- **D-04:** `--print-config-style` **does not exist today** (discovery finding, 2026-09-18: only a doc comment in `types.go` names it; TS's `--print-config <id>` was never ported). This phase adds it as a **read-only** flag on `install`: it opens no picker, writes nothing, honours `-l/--location` and `-t/--target` for filtering, and prints one line per target — `<target>: scopes=<…> mcp=<path> format=<style> instructions=<path|none> skill=<path|none> hooks=<mechanism>` — through Phase 4's `present.KV`/`Line` helpers behind the shared `resolveColor` (plain and byte-stable on a pipe; the plain output is golden-frozen as ours). It flows into `docs/CLI-REFERENCE.md` via `task docs:cli` in a reviewed diff; `TestEveryRegisteredFlagIsAccountedFor` stays green with no allowlist entry (it is a visible documented flag). — **Reversibility:** reversible (additive flag).

**Skill package writes (AGENT-09, AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11)**
- **D-05:** The shared package — the embedded `SKILL.md` plus the sidecar manifest — is written to `.agents/skills/codegraph/` (local) / `~/.agents/skills/codegraph/` (global) by **one shared writer, once per install run**, however many selected targets list it; later targets in the same run see `unchanged`. **Claude Code's existing `.claude/skills/codegraph/` package (SKILL.md + session-nudge.sh + hooks) is untouched** — it is proven by v0.10.0/Phase 7 live evidence and not migrated.
- **D-06:** Harness-specific skill directories by default: **only** Gemini CLI (`.gemini/skills/codegraph/` / `~/.gemini/skills/codegraph/`) and Kiro (`.kiro/skills/codegraph/` / `~/.kiro/skills/codegraph/`), whose docs do not list `.agents/skills/`. Cursor, opencode and Antigravity rely on the shared path; a harness-specific directory is added for one of them **only** if its live session (D-09) shows the shared path is not read.
- **D-07:** One sidecar manifest per written skill directory, in the existing `skillManifest` format extended **additively** with `targets: [<target ids that requested this dir>]` (schema_version bumped; an older manifest without `targets` is read as "owned, requester set unknown" and upgraded in place on the next install).
- **D-08:** Uninstall of a shared directory removes **only the uninstalling target** from the manifest's `targets`; the package (SKILL.md + manifest) is deleted only when `targets` becomes empty. A harness-specific directory (D-06) has a single requester and is removed with its target. `removeSkillDirIfEmpty` semantics are kept.

**Live verification (the evidence rows)**
- **D-09:** Method: **Herdr-driven fresh sessions** on the maintainer's machine for the harnesses installed there — Cursor (`cursor-agent`), opencode, Antigravity (`agy`) — started with `herdr agent start <name> --kind cursor|opencode|agy --pane <id>` in a sibling pane: install codegraph into a scratch project, ask the agent to list its skills and to answer a where-is-X question, capture the transcript with `herdr agent read`; then a **negative control** — the same prompt in a scratch project with nothing installed must not show the skill. This is the v0.10.0 live-session standard with its negative-space check (research pitfall 12). The orchestrator runs these; they are not delegated to executor subagents (a backgrounded subagent cannot drive another pane's TTY reliably).
- **D-10:** Gemini CLI and Kiro are **not installed** on the maintainer's machine (checked 2026-09-18). Their documented paths are written (D-06) and AGENT-10/AGENT-11 evidence is recorded **`[ASSUMED]`** with the doc citation and fetch date — never claimed verified — unless the maintainer installs them before execution (installing them is the maintainer's call, via `dotfiles-upgrade`).
- **D-11:** Cursor's repo-root `AGENTS.md` pickup (AGENT-04) is probed live: plant a distinctive sentence in a scratch repo's `AGENTS.md` (no `.cursor/rules/` file present), ask a fresh `cursor-agent` to repeat project instructions; **only** an observed pickup switches Cursor's instructions target from `.cursor/rules/codegraph.mdc` to repo-root `AGENTS.md`. The outcome is recorded either way; the probe precedes any Cursor instructions-target change.
- **D-12:** All live evidence goes in `05-LIVE-SESSIONS.md`: per harness — the exact command, the transcript excerpt, the negative control, and the verdict (`read` / `not read` / `[ASSUMED]`). A duplicate-skill check is part of each positive session: harnesses documented to read **both** `.claude/skills/` and `.agents/skills/` (Cursor, opencode) are probed with Claude's package also installed, and a doubled codegraph skill is recorded as a finding (it decides whether D-05's shared write needs a guard for that pairing).

**Ownership & reversal (AGENT-13)**
- **D-13:** One planted-foreign-entry table test over all eight targets × both locations, written **RED-first**: seed a foreign MCP server entry, a foreign `.agents/skills/other/` directory, a foreign `.agents/skills/codegraph/` directory **without** a codegraph manifest, and a foreign marker-less section in every instructions file → `install` → `uninstall` → every foreign byte identical, every own entry gone (Family (b), and a planted ownership-recovery mutation goes RED).
- **D-14:** Ownership of a skill directory = **our sidecar manifest**. A `codegraph/` skill directory without it is foreign: never overwritten, never removed, reported as `kept (foreign)` in the write result — the same exact-identity rule as `writeHookEntry`.
- **D-15:** Commit `242ec0a` (the reverted hook-block ownership recovery that closed an authorization differential) is cited in the D-13 test file's doc comment, naming the differential it closed, and in the plan's review notes (AGENT-13's "cited in review").
- **D-16:** A hand-edited **own** file (manifest hash mismatch) keeps the existing manifest D-05 response — codegraph-owned content is rewritten and the mismatch recorded; only unmanifested content is untouchable.

### Claude's Discretion
- The exact `Capabilities` field names and Go types, and whether per-location paths are methods or maps on the struct — subject to D-02's content and D-01's one-literal-per-file shape.
- File layout for the shared skill writer (a new `skillshared.go` vs extending `shared.go`).
- The scratch-project layout and exact prompts for the live sessions, provided each has a negative control (D-09/D-12).
- Plan ordering, subject to: capability table + its guard first (D-01…D-04) → ownership test RED-first (D-13) → shared/harness-specific skill writers (D-05…D-08) → per-harness wiring → live sessions (D-09…D-12) → `docs/CLI-REFERENCE.md` regeneration through the drift gate.

### Deferred Ideas (OUT OF SCOPE)
- Migrating Claude Code's skill package to the shared `.agents/skills/` path — declined for this phase (A2 Q1); revisit only if D-12's duplicate-skill finding shows a real conflict.
- Installing Gemini CLI and Kiro to upgrade AGENT-10/AGENT-11 from `[ASSUMED]` to verified — the maintainer's call (D-10).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AGENT-08 | `AgentTarget` exposes per-target capabilities so `install`, `uninstall`, `Detect` and `--print-config-style` derive from one table | See "Architecture Patterns → The Capability Table" and "Code Examples" for the concrete `Capabilities` struct/field values derived from all 8 target files read this session; D-04's non-existence confirmed live (`rg -n "print-config" internal/`). |
| AGENT-09 | Skill package written once to `.agents/skills/codegraph/` (project) / `~/.agents/skills/codegraph/` (global) with sidecar manifest; live-verified per harness; harness-specific dir only if shared path is not read | See "Critical Findings" (Gemini's `.agents/skills/` precedence, Antigravity's workspace-only `.agents/skills/` support) and "Common Pitfalls" (the real `~/.claude/skills/codegraph` → `~/.agents/skills/codegraph` symlink found on this machine). |
| AGENT-04 | Cursor — skill package installed; repo-root `AGENTS.md` pickup probed live before changing Cursor's instructions target | Cursor's skill-dir precedence and `.claude`/`.codex` compatibility reads, and the exact AGENTS.md doc text the D-11 probe is designed against, both fetched raw from `cursor.com/docs/skills.md` / `cursor.com/docs/rules.md` today — see "Per-Harness Findings → Cursor". |
| AGENT-06 | opencode — skill package at a path opencode reads; SKILL.md frontmatter compatibility verified; existing instructions block retained | opencode's exact discovery paths and frontmatter contract fetched raw from `opencode.ai/docs/skills.md` today; codegraph's existing frontmatter (`name`, `description` only) confirmed compatible — see "Per-Harness Findings → opencode". |
| AGENT-07 | Antigravity — skill package via `.agents/skills/`; instructions block in `AGENTS.md` | Antigravity's skill *and* rules docs fetched raw from `antigravity.google/docs/skills.md` and `.../rules-workflows.md` today reveal AGENT-07's literal "AGENTS.md" wording does not match Antigravity's actual global mechanism — see "Critical Findings" #3. |
| AGENT-10 | Gemini CLI — skill package to `.gemini/skills/` + global equivalent; discoverable via `activate_skill`; existing instructions block retained | Gemini CLI's discovery-tier doc (raw `docs/cli/skills.md` from `google-gemini/gemini-cli`) fetched today directly contradicts the milestone `STACK.md`'s claim that Gemini does not read `.agents/skills/` — see "Critical Findings" #1. Live evidence stays `[ASSUMED]` per D-10 (not installed here). |
| AGENT-11 | Kiro — skill package to `.kiro/skills/` + `~/.kiro/skills/`; `AGENTS.md` retained as a steering source | Kiro's steering doc fetched raw today gives the exact two literal `AGENTS.md` read locations (`~/.kiro/steering/AGENTS.md` global, project-root + nested `AGENTS.md` workspace) — a concrete, checkable design for "AGENTS.md retained as a steering source." Live evidence stays `[ASSUMED]` per D-10. |
| AGENT-13 | Exact-identity ownership per new harness write; planted-foreign-entry test; `uninstall` reverses byte-identically; `242ec0a` cited | `git show 242ec0a` read in full this session — the exact differential, the revert's shape, and the regression test it added are all quoted in "Critical Findings" #5 / "Code Examples". |
</phase_requirements>

## Summary

This phase is almost entirely **in-repo Go refactoring and file-writing logic** — no new external dependency, no new registry, and (per D-00) no test may assert a harness's read behavior. The two things worth researching hard are (1) the exact shape of the existing code the capability table must derive from without duplicating, and (2) what each of the five non-Claude harnesses' **current** (2026-09-18) primary documentation says about skill discovery, precedence, and instructions — because the milestone's own `STACK.md`/`PITFALLS.md` (dated 2026-09-14, several claims sourced via WebSearch synthesis rather than a direct fetch) are already stale on at least three points this session's direct fetches corrected. All eight target files, `types.go`, `registry.go`, `shared.go`, `manifest.go`, `instructions.go`, `claudeassets.go`, `install.go`/`uninstall.go`, and the `242ec0a` revert were read directly this session (line evidence throughout); every per-harness claim below not read from this repo's own source is either quoted from a primary-source `.md` fetched today, or explicitly marked `[ASSUMED]`.

**Primary recommendation:** implement `Capabilities()` as eight one-literal-per-file additions (mirroring the existing `SupportsLocation` shape), derive `--print-config-style` and the D-03 table-equality guard from it with zero new hand-written path duplication; build the shared `.agents/skills/` writer as a new `skillshared.go` reusing `writeEmbeddedFile`/`writeManifest` verbatim; and — before landing D-06's default (only Gemini/Kiro get a harness-specific dir) — read "Critical Findings" below, because current docs show Gemini's `.agents/skills/` alias actually **outranks** its native `.gemini/skills/` (the opposite of D-06's stated rationale for singling Gemini out), and Antigravity's documented **global**-scope skill/instructions paths are `~/.gemini/antigravity-cli/skills/` and `~/.gemini/GEMINI.md` respectively — neither of which is `~/.agents/skills/` or a literal `AGENTS.md` file. Both are real candidates for the "add a harness-specific dir/file for one of them" escape valve D-06 already provides, and having the concrete replacement path ready *before* the D-09 live session runs saves a re-plan cycle if the live evidence confirms the doc.

## Critical Findings

These five findings correct or sharpen the milestone-level research (`STACK.md`/`PITFALLS.md`, 2026-09-14) and the phase's own `CONTEXT.md` rationale, based on primary-source pages fetched today (2026-09-18) or on this repo's own source/git history read directly this session. Each is `[CITED]`/`[VERIFIED]` as marked; none is a request to override a locked D-decision — they are evidence for the planner and for `05-LIVE-SESSIONS.md` to weigh.

### 1. Gemini CLI's `.agents/skills/` alias takes PRECEDENCE over its native `.gemini/skills/` — the opposite of D-06's stated rationale

`[CITED: raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md, fetched 2026-09-18]` — the current, primary-source doc states:

> "User skills: Located in `~/.gemini/skills/` or the `~/.agents/skills/` alias." … "Workspace skills: Located in `.gemini/skills/` or the `.agents/skills/` alias." … "If multiple skills share the same name, the version from the higher-precedence location is used. Within the same tier (user or workspace), the `.agents/skills/` alias takes precedence over the `.gemini/skills/` directory."

D-06's stated rationale for singling out Gemini CLI for a harness-specific directory is "whose docs do not list `.agents/skills/`." That premise is now contradicted by the current doc: Gemini CLI **does** list `.agents/skills/`, and it wins ties. This does not by itself void D-06 (D-06 is a locked decision and AGENT-10 explicitly names `.gemini/skills/` as a required write target regardless), but it does mean: (a) the harness-specific `.gemini/skills/codegraph/` write is not *necessary* for Gemini CLI's discovery per current docs — the shared `.agents/skills/codegraph/` write alone would already be picked up, and would win any name collision; (b) writing both is harmless (same content, D-05's shared writer and AGENT-10's harness-specific writer both source from `claudeassets.SkillMarkdown()`, so no divergent-content risk); (c) the milestone `STACK.md`'s claim ("Gemini CLI... does **not** appear to treat AGENTS.md... not found in docs fetched today — do not assume it is read" — about instructions, a separate claim from skills, which stands) should not be conflated with the *skills* claim, which is what changed.

### 2. Antigravity's documented global-scope skill paths are `~/.gemini/antigravity-cli/skills/` (CLI) / `~/.gemini/config/skills/` (2.0/IDE) — NOT `~/.agents/skills/`

`[CITED: antigravity.google/docs/skills.md, fetched 2026-09-18]` — the current, primary-source doc's per-surface tables:

> Antigravity 2.0: `~/.gemini/config/skills/<skill-folder>/` — "Global (all workspaces)"
> Antigravity CLI: `~/.gemini/antigravity-cli/skills/<skill-folder>/` — "Global (all workspaces)" (plus `~/.gemini/antigravity-cli/plugins/<name>/skills/` for plugin-provided skills)
> Antigravity IDE: `~/.gemini/config/skills/<skill-folder>/` — "Global (all workspaces; legacy `~/.gemini/antigravity/skills/` is also supported)"

`.agents/skills/` **is** documented — but only as the **workspace**-scoped path (`<workspace-root>/.agents/skills/<skill-folder>/`), identical across all three surfaces. `antigravityTarget` in this repo is **global-only** today (`[VERIFIED: internal/agents/antigravity.go:25-27]` — `func (antigravityTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }`), so the only scope this phase can write for Antigravity is exactly the scope whose documented path is *not* `.agents/skills/`. D-05/D-06 plan to rely on the shared writer's global leg (`~/.agents/skills/codegraph/`) for Antigravity by default; per this doc, that write lands somewhere Antigravity's own docs never list as a global read location. This is strong, doc-sourced prior evidence — obtained *before* any live session — that Antigravity is a real candidate for D-06's "add a harness-specific dir for one of them if the live session shows the shared path is not read" escape valve. The concrete harness-specific candidate, for the `agy` CLI kind D-09's herdr session will actually exercise, is `~/.gemini/antigravity-cli/skills/codegraph/`.

### 3. Antigravity's actual global instructions mechanism is `~/.gemini/GEMINI.md`, not a global `AGENTS.md` — matches the EXISTING code, not AGENT-07's literal wording

`[CITED: antigravity.google/docs/rules-workflows.md, fetched 2026-09-18]`:

> "Antigravity 2.0 file locations — Global rules: saved to `~/.gemini/GEMINI.md`... Workspace rules: saved to the `.agents/rules/` directory..." "CLI file locations — ...Global rules: place markdown rules in `~/.gemini/antigravity-cli/rules/` or define persistent global constraints in `~/.gemini/GEMINI.md`." "IDE file locations — ...Global rules: stored in `~/.gemini/GEMINI.md`."

There is **no** literal global `AGENTS.md` file in Antigravity's own documented mechanism at all — its global instructions file, across all three surfaces, is `~/.gemini/GEMINI.md` (workspace-scope uses a `.agents/rules/` *directory* of markdown files, not a single `AGENTS.md`). This matches — and validates — what this repo's code **already does**: `[VERIFIED: internal/agents/antigravity.go:14-15]` — "Writes no instructions file of its own; it shares `~/.gemini/GEMINI.md`, written only by the Gemini target." AGENT-07's requirement text ("the instructions block in `AGENTS.md`") is therefore best read as shorthand for "the shared marker-fenced `codegraphInstructionsBlock`," not a literal new `~/.gemini/AGENTS.md` file — writing one would be inert, since Antigravity's own docs never list it as a read path (and a community source independently corroborates Gemini CLI itself does not read `~/.gemini/AGENTS.md` either). No code change to Antigravity's instructions handling appears to be required by this finding; flagged so the planner does not introduce a new, doc-contradicted `AGENTS.md` write for Antigravity under a literal reading of AGENT-07.

### 4. Kiro's `AGENTS.md` discovery has concrete, primary-sourced paths — a real design for AGENT-11's "retained as a steering source"

`[CITED: kiro.dev/docs/steering/, fetched 2026-09-18]`:

> "Kiro supports providing steering directives via the AGENTS.md standard. AGENTS.md files are in markdown format, similar to Kiro steering files; however, AGENTS.md files do not support inclusion modes and are always included. You can add AGENTS.md files to the global steering file location (`~/.kiro/steering/`), or to the root folder of your workspace, and they will get picked up by Kiro automatically. AGENTS.md files are also discovered in subdirectories throughout your workspace."

Concretely: Kiro reads a literal `AGENTS.md` at `~/.kiro/steering/AGENTS.md` (global) and at the workspace root `./AGENTS.md` (local) — and, notably, the workspace-root path is the **same literal path and filename** opencode already writes at local scope (`[VERIFIED: internal/agents/opencode.go:80-82]` — `opencodeInstructionsPath(loc)` returns `"AGENTS.md"` for local). Since `upsertInstructionsEntry`/`replaceOrAppendMarkedSection` is a marker-fenced, idempotent upsert (`[VERIFIED: internal/agents/shared.go:670-683,576-614]`), a second target writing the identical marker block to the same file is a no-op collision, not a conflict — Kiro's local-scope instructions write can safely point at the same `"AGENTS.md"` project-root path opencode/Codex/Cursor already use, with no new collision-handling code needed. `kiroInstructionsPath(loc)` — local: `"AGENTS.md"`; global: `filepath.Join(home, ".kiro", "steering", "AGENTS.md")` — is a concrete, doc-grounded design point for AGENT-11 that this repo's own `codex.go`/`opencode.go`/`gemini.go` shape (`instructionsBody()` + `upsertInstructionsEntry`) already supports with zero new helper code.

### 5. `242ec0a`'s exact authorization differential — for the D-15 citation and the D-13 test's doc comment

`[VERIFIED: git show 242ec0a, read in full this session]` — the commit reverted a "matcher+shape recovery" fallback that had been added to `writeHookEntry` to fix a duplicate-hook-block nuisance. The differential it closed, precisely: the recovery path let codegraph **silently claim and overwrite** an unrelated, user-authored hook block that happened to share Claude's own matcher name (`"startup"`) whenever a codegraph manifest was *already present at that install location* — i.e., ownership was granted by (manifest-presence + matcher-name-and-shape match), not by the hook block's own command-string identity. The revert restored exact command-string-only ownership (`[VERIFIED: internal/agents/shared.go:179-201]`, doc comment: "Ownership of a block is determined SOLELY by exact command-string match... never by the block's matcher value or shape"). The commit's own regression test, `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher`, is the template shape for AGENT-13's per-harness ownership test: it (1) installs once to establish a manifest at the location, (2) hand-replaces codegraph's own hook block with an unrelated single-command hook under the identical matcher name, (3) re-installs, and (4) asserts the unrelated hook survives byte-for-byte while codegraph's own blocks are appended alongside it, never merged in. AGENT-13's D-13 test should cite `242ec0a` by SHA in its doc comment (per D-15) and reuse this exact "manifest-present, matcher-shared, command-string-different" precondition — not a weaker "directory exists" precondition — since that is the literal shape the vulnerability took.

## Architectural Responsibility Map

This phase's "architecture" is a CLI installer, not a client/server web app — the Browser/SSR/API/CDN/Storage tiers don't map cleanly. The table below adapts the framework: "API/Backend" = in-process Go logic this repo owns; "External Storage" = third-party config/skill files this repo edits but does not own; "External Runtime" = the harness's own process, whose read behavior this repo's tests can never assert (D-00).

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `Capabilities()` table declaration | API/Backend (`internal/agents/*.go`) | — | Pure Go struct literals; no I/O, no CLI concern |
| `--print-config-style` rendering | API/Backend (`internal/cli/install.go` RunE) | — | Read-only report derived entirely from `Capabilities()`; no new state |
| Shared/harness-specific skill package bytes | External Storage (`.agents/skills/`, `.gemini/skills/`, `.kiro/skills/`, etc.) | API/Backend (the writer functions) | The bytes live in third-party config trees this repo doesn't own; `internal/agents` owns only the writer logic that produces them |
| Ownership/exact-identity guard | API/Backend (`internal/agents/shared.go`) | — | Pure Go comparison logic; the vulnerability class (`242ec0a`) was entirely in this tier |
| Live-session "does the harness read this path" evidence | External Runtime (the harness's own process, herdr-driven) | — | Categorically outside this repo's test suite by D-00; recorded in `05-LIVE-SESSIONS.md`, never asserted by `go test` |

## Standard Stack

No new external dependency is required by this phase. All work is additive Go logic inside `internal/agents` and `internal/cli`, reusing helpers already in the package.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| (none new) | — | — | The shared skill writer reuses `writeEmbeddedFile`/`writeManifest`/`removeSkillDirIfEmpty` verbatim `[VERIFIED: internal/agents/shared.go:301-341,479-500]`; the capability table is plain Go structs; `--print-config-style` reuses Phase 4's `present.KV`/`Line`/`resolveColor` `[VERIFIED: internal/cli/present/line.go:16,36; internal/cli/colorflag.go:230]`. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/tailscale/hujson` | already a direct dep (opencode's config editing) `[VERIFIED: internal/agents/opencode.go:10]` | Comment-preserving JSONC patch | Not touched by this phase — opencode's skill *directory* writes go through `writeEmbeddedFile`, not through hujson at all; only opencode's MCP-entry write uses hujson, and that's pre-existing, unmodified code. Listed here only to confirm no new JSON/YAML/TOML library is warranted for the skill-package work. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing `writeEmbeddedFile`/`writeManifest` for the shared writer | A brand-new set of shared-writer primitives | Rejected: `writeEmbeddedFile`'s raw-byte idempotency (D-07 of v0.10.0) and `writeManifest`'s "no needless-rewrite" idempotency are exactly what D-05's "written once per install run, later targets see `unchanged`" behavior needs — reinventing them for the new `.agents/skills/` path would duplicate logic that already has test coverage. |

**Installation:** N/A — no `go get` needed.

**Version verification:** N/A — no package versions to verify.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero new external packages (Go or otherwise) — confirmed by reading every target file, `shared.go`, `manifest.go`, and `instructions.go` this session; the only import touched by the new work is the existing `internal/version` (already a direct, in-repo package, not a registry dependency). The Package Legitimacy Gate protocol is skipped per its own trigger condition ("Every phase that installs external packages").

## Architecture Patterns

### System Architecture Diagram

```
                         codegraph install / uninstall / --print-config-style
                                            │
                                            ▼
                              internal/cli/install.go RunE
                         (resolves --target/--location, loops registry)
                                            │
                         ┌──────────────────┼───────────────────────────┐
                         ▼                  ▼                           ▼
                 t.Capabilities()    t.Install(loc,opts)         t.Uninstall(loc)
                 (AGENT-08, new)     (existing, per-target)      (existing, per-target)
                         │                  │                           │
              ┌──────────┘          ┌───────┴────────┐          ┌───────┴────────┐
              ▼                     ▼                ▼          ▼                ▼
      --print-config-style   MCP-entry write   instructions   skill package    ownership
      table render (new)     (existing helpers) write (existing) write (NEW, D-05/D-06) guard (D-13/D-14,
              │               writeMcpEntry/       upsertInstructions  shared writer    exact-identity)
              ▼               spliceTOMLTable/       Entry (marker-        │
      one line per target     hermesSplice...        fenced)               ▼
      (this repo's own                                          .agents/skills/codegraph/
       byte-stable golden)                                      (+ harness-specific dir
                                                                   only per D-06's escape valve)
                                                                              │
                                                                              ▼
                                                          ═══ process boundary (D-00) ═══
                                                                              │
                                                                              ▼
                                                    the harness's OWN discovery/read logic —
                                                    verified only by a live herdr session
                                                    (D-09…D-12), never by go test
```

### Recommended Project Structure

No new directories — additive files inside the existing package:

```
internal/agents/
├── types.go          # AgentTarget interface — grows Capabilities() Capabilities (D-01/D-08)
├── registry.go        # unchanged (D-01: no central capability map here)
├── shared.go           # unchanged; reused by the new writer, not modified
├── skillshared.go       # NEW (Claude's Discretion): the shared .agents/skills/ writer + its
│                         #   manifest-with-targets upgrade logic (D-05/D-07/D-08)
├── manifest.go         # skillManifest grows targets []TargetID additively (D-07)
├── claude.go            # unchanged Install/Uninstall; adds Capabilities() literal
├── cursor.go            # adds Capabilities() literal; AGENT-04's AGENTS.md-target
│                         #   change is gated on D-11's live probe outcome
├── codex.go             # adds Capabilities() literal (no new write this phase — Phase 7)
├── opencode.go          # adds Capabilities() literal (skill dir wiring, AGENT-06)
├── hermes.go            # adds Capabilities() literal (Skill:false, Hook:none — no change)
├── gemini.go            # adds Capabilities() literal; AGENT-10's harness-specific
│                         #   .gemini/skills/ writer alongside the shared write
├── antigravity.go       # adds Capabilities() literal; AGENT-07 — no instructions
│                         #   change per Critical Finding #3; skill write is the shared
│                         #   path by default, watch D-09's agy live session per Finding #2
└── kiro.go              # adds Capabilities() literal; AGENT-11 — kiroInstructionsPath(loc)
                          #   is a NEW function (Kiro currently writes none) per Finding #4
internal/cli/
└── install.go           # adds --print-config-style flag + its render function (D-04)
```

### Pattern 1: One capability literal per target file, mirroring `SupportsLocation`

**What:** Every target already implements `SupportsLocation(Location) bool` as a one-line method on its own zero-field struct type — e.g. `[VERIFIED: internal/agents/codex.go:28]` `func (codexTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }`. D-01 asks for the same shape: `Capabilities() Capabilities` returning one struct literal, defined in the target's own file, never in a shared map.
**When to use:** Every one of the 8 target files, added in the same commit as the `AgentTarget` interface's new method (a compile-time-enforced, mechanical change — Go will not build until all 8, plus both test fakes below, implement it).
**Example:**
```go
// Source: this repo, internal/agents/codex.go:26-28 (existing pattern this phase extends)
func (codexTarget) ID() TargetID                       { return Codex }
func (codexTarget) DisplayName() string                { return "Codex CLI" }
func (codexTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }

// NEW, same file, same shape (illustrative — exact field names are Claude's Discretion per D-02):
func (codexTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal},
		ConfigFormat: ConfigFormatTOML,
		MCPConfig:    map[Location]string{LocationGlobal: "~/.codex/config.toml"}, // resolved, not literal — see note
		Instructions: map[Location]string{LocationGlobal: "~/.codex/AGENTS.md"},
		SkillDirs:    nil, // Codex skill install is Phase 7 scope (out of bounds this phase)
		Hooks:        HooksNone,
	}
}
```
**Note on avoiding a second hand-written path:** per D-02, "no target keeps a second hand-written copy of any path the table holds" — so the literal above must call the target's own existing path functions (`codexConfigPath()`, `codexInstructionsPath()`) rather than restating the strings, and `DescribePaths`/`Detect` should be rewritten to call `Capabilities()` and derive their return values from it, closing the loop D-03's guard checks.

### Pattern 2: The shared skill writer as a `.agents/skills/`-scoped `writeEmbeddedFile` + manifest pair

**What:** D-05's "one shared writer, once per install run" reuses the exact `writeEmbeddedFile`/`writeManifest` idempotency this repo already trusts for Claude's package — applied to a *new* path pair (`.agents/skills/codegraph/SKILL.md` / `~/.agents/skills/codegraph/SKILL.md`) and a manifest extended with `targets`.
**When to use:** Called once per `install` invocation (not once per target) — the CLI's install loop (`printAgentResults`, `[VERIFIED: internal/cli/install.go:141-209]`) currently calls `do(t)` per target with no shared state across targets; D-05's "later targets in the same run see `unchanged`" requires either (a) a package-level memo keyed by `(loc)` for the duration of one `install` run, or (b) relying on `writeEmbeddedFile`'s own byte-comparison idempotency so the *n*-th identical write in the same run is naturally `ActionUnchanged` even without a memo. Option (b) is simpler and requires no new state — `writeEmbeddedFile` already reads-before-write and returns `ActionUnchanged` on a byte match `[VERIFIED: internal/agents/shared.go:301-323]`; the only wrinkle is the manifest's `targets` field accumulating requesters across calls within one run, which does need each target's `Install` to append its own ID to the shared manifest read fresh each time (not memoed), since D-08's uninstall semantics depend on `targets` reflecting every requester, not just the first one that ran this session.
**Example:**
```go
// Illustrative shape for skillshared.go (file layout is Claude's Discretion per CONTEXT.md)
// Source: pattern lifted from internal/agents/claude.go:430-444's existing
// writeEmbeddedFile call + manifest.go's writeManifest, generalized to a
// caller-supplied requester ID rather than being Claude-only.
func installSharedSkillPackage(loc Location, requester TargetID) (WriteResult, error) {
	dir := sharedSkillDirPath(loc) // .agents/skills/codegraph or ~/.agents/skills/codegraph
	skillPath := filepath.Join(dir, "SKILL.md")
	content, err := claudeassets.SkillMarkdown() // same embed, no new asset (phase boundary)
	if err != nil {
		return WriteResult{}, err
	}
	var result WriteResult
	fr, werr := writeEmbeddedFile(skillPath, string(content), false)
	recordFile(&result, skillPath, fr, werr)

	manifestPath := filepath.Join(dir, ".codegraph-manifest.json")
	existing, present, _ := readManifest(manifestPath)
	targets := existing.Targets // read fresh, not memoed (D-08 needs every requester)
	if present && !containsTarget(targets, requester) {
		targets = append(targets, requester)
	} else if !present {
		targets = []TargetID{requester}
	}
	// ... writeManifest with schema_version bumped, targets: targets ...
	return result, nil
}
```

### Anti-Patterns to Avoid
- **A package-level "already wrote the shared package this run" boolean:** tempting for D-05's "later targets see `unchanged`" but wrong — it would make the *second* target in a multi-target run never get its ID appended to `targets`, breaking D-08's per-target uninstall removal. Read the manifest fresh on every call within the run instead; `writeEmbeddedFile`'s own byte-comparison already gives the SKILL.md write its idempotency for free.
- **Restating a path string in `Capabilities()` instead of calling the existing path function:** directly contradicts D-02's "no target keeps a second hand-written copy of any path the table holds" and is exactly the shape D-03's planted-divergence guard exists to catch.
- **Writing a literal `~/.gemini/AGENTS.md` for Antigravity under a literal reading of AGENT-07:** per Critical Finding #3, no primary source (Antigravity's or Gemini CLI's own docs) lists that path as read; it would be an inert write.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Idempotent embedded-file writes for the new `.agents/skills/SKILL.md` targets | A second byte-comparison-and-write helper | `writeEmbeddedFile` (`[VERIFIED: internal/agents/shared.go:301-341]`) | Already handles the executable-bit self-heal case (CR-02) and the byte-identity short-circuit (D-07); a second copy would need to re-earn both correctness properties. |
| Directory-empty-on-removal cleanup for the shared skill dir | A second "is this dir empty" check | `removeSkillDirIfEmpty` (`[VERIFIED: internal/agents/shared.go:479-500]`) | Already handles the ENOTEMPTY-vs-generic-error distinction correctly across platforms; a recursive delete here would risk removing a user's unrelated file in `.agents/skills/codegraph/` (prohibited). |
| Marker-fenced instructions upsert for Kiro's new `AGENTS.md` write (Finding #4) | A Kiro-specific instructions writer | `upsertInstructionsEntry`/`removeMarkedSection` (`[VERIFIED: internal/agents/shared.go:670-683,616-668]`), the same helpers Codex/opencode/Gemini already call | These already implement the create/replace/append/no-markers branches T-06-02 requires; Kiro's write is a location-path change, not new logic. |
| Ownership-recovery-by-matcher-shape for a new harness's hook mechanism (none this phase, but flagged for future Codex parity in Phase 7) | Any "recover ownership if the manifest is present and the shape looks familiar" heuristic | Exact command-string (or, for a non-hook artifact, exact-manifest-presence) identity, per `242ec0a` | This is the literal vulnerability class the milestone's own standing rule exists to prevent (commit `242ec0a`, cited in D-15) — a shape/position heuristic was tried once in this exact codebase and reverted after a security review found it exploitable. |

**Key insight:** every helper this phase needs to reuse already exists and is already unit-tested for the property this phase needs (idempotency, exact-identity ownership, marker-fenced upsert) — the work is almost entirely *wiring*, not new primitives. The risk is in getting the per-harness *paths* right (see Critical Findings) and in not accidentally re-deriving a weaker version of an already-hardened helper for a "new" harness that looks superficially different.

## Common Pitfalls

### Pitfall 1: The `AgentTarget` interface change breaks two test fakes that must grow the same method

**What goes wrong:** Adding `Capabilities() Capabilities` to the `AgentTarget` interface is a compile-time-enforced change across every implementer — including the two hand-written test fakes that implement the full interface for unit testing: `fakeTarget` in `internal/agents/registry_test.go` (`[VERIFIED: internal/agents/registry_test.go:13-26]`, methods `ID/DisplayName/SupportsLocation/Detect/Install/Uninstall/DescribePaths`) and `fakeAgentTarget` in `internal/cli/tui/agentpicker_test.go` (`[VERIFIED: internal/cli/tui/agentpicker_test.go:33]`, at minimum `DescribePaths`). Neither currently has a `Capabilities()` method; the package will not compile until both grow one.
**Why it happens:** The interface lives in `internal/agents/types.go`; its two test-fake implementers live in two different packages (`agents` and `cli/tui`), so a `grep` scoped to `internal/agents/*.go` alone will miss the second one.
**How to avoid:** Before landing the interface change, `rg -n "func \(f fake" internal/` (or equivalent) to enumerate every fake implementer, and add a minimal `Capabilities()` stub to each in the SAME commit as the interface change — otherwise the build breaks in a package the plan may not have touched yet.
**Warning signs:** `go build ./...` failing with "does not implement AgentTarget (missing method Capabilities)" pointing at a `_test.go` file in a package the plan's `files_modified` list didn't name.
**Phase to address:** The capability-table task (D-01), same commit as the interface change.

### Pitfall 2: On the maintainer's own machine, `~/.claude/skills/codegraph` is ALREADY a symlink into `~/.agents/skills/codegraph` — for a reason unrelated to this project

**What goes wrong:** `[VERIFIED: this session, `readlink ~/.claude/skills/codegraph` → `../../.agents/skills/codegraph`; `stat -f "%i"` on both directories' manifest files returns the same inode]`. This is the maintainer's own personal dotfiles-managed skill-sharing setup (their global CLAUDE.md: "For non-GSD personal skills, use `npx skills` and `~/.agents/skills` as the canonical installation. Keep agent-specific links where needed.") — **not** anything this project's code wrote. It happens to already contain a `codegraph` skill (this project's own `SKILL.md`, byte-identical to `.claude/skills/codegraph/SKILL.md` in this repo, confirmed via `diff`) with a manifest dated 2026-09-15, written by a prior real `codegraph install --target claude --location global` run on this machine. Because of the symlink, that manifest's *physical* location is `~/.agents/skills/codegraph/.codegraph-manifest.json` even though `claudeManifestPath(LocationGlobal)` computes `~/.claude/skills/codegraph/.codegraph-manifest.json` — the OS resolves the symlink transparently. **If any global-scope live verification for this phase runs against this real `$HOME`** (rather than a faked one), the new shared writer (D-05, targeting `~/.agents/skills/codegraph/`) and Claude's existing writer (targeting `~/.claude/skills/codegraph/`) will, on THIS machine only, resolve to the exact same physical directory — a collision D-05's design assumes cannot happen ("Claude Code's existing... package is untouched").
**Why it happens:** Personal dotfiles tooling and this project's own install mechanism happen to use the same directory name (`codegraph`) under two paths that are siblings under `~/`, and this particular machine has a pre-existing symlink between them for an entirely unrelated reason.
**How to avoid:** Every unit test in this package already uses `fakeHome(t)` (`[VERIFIED: internal/agents/testhelpers_test.go:15-23]`) to fully isolate `$HOME`/`$XDG_CONFIG_HOME`/`$HERMES_HOME` per test — that isolation already prevents this collision for `go test`. The risk is specifically in **live-session verification** (D-09) if it is ever run against the real `$HOME` for a global-scope target: before any such live run, `ls -la ~/.claude/skills/ ~/.agents/skills/` to check whether this symlink is present, and if so, either (a) scope the live session's `HOME` to a scratch directory the way the Go tests already do, or (b) explicitly account for the collision when interpreting the result — a byte-identical outcome at both paths on this machine is not proof the shared write is being read at a genuinely distinct location.
**Warning signs:** A live-session global-scope test reporting `~/.agents/skills/codegraph/` and `~/.claude/skills/codegraph/` as both "written"/"read" when only one write actually happened; `readlink` on either path returning non-empty.
**Phase to address:** Before any global-scope live session (D-09/D-10) runs on the maintainer's real machine — a one-line pre-flight check.

### Pitfall 3: opencode's own troubleshooting doc explicitly warns about duplicate skill names across its discovery locations — directly relevant to D-12's duplicate-skill check

**What goes wrong:** `[CITED: opencode.ai/docs/skills.md, fetched 2026-09-18]` — opencode's own "Troubleshoot loading" section lists, as step 3, "Ensure skill names are unique across all locations." opencode discovers `.opencode/skills/`, `.claude/skills/`, and `.agents/skills/` at both project and global scope (`[CITED: same page]` — "It loads any matching `skills/*/SKILL.md` in `.opencode/` and any matching `.claude/skills/*/SKILL.md` or `.agents/skills/*/SKILL.md` along the way"). Once this phase's shared writer lands `.agents/skills/codegraph/SKILL.md` (`name: codegraph`) alongside Claude's pre-existing `.claude/skills/codegraph/SKILL.md` (also `name: codegraph`, byte-identical content), opencode will see the SAME skill name at two locations simultaneously — exactly the condition its own docs flag as a loading problem, with no documented precedence/dedup rule given for this specific ambiguity (unlike Gemini CLI's explicit precedence rule, Finding #1).
**Why it happens:** D-05 deliberately reuses the identical `SKILL.md` content and name for the shared package (no second skill file is in scope, per the phase boundary) — which is the right call for content, but means any harness that reads both `.claude/skills/` and `.agents/skills/` will see a genuine same-name collision, not just a superficially similar one.
**How to avoid:** This is precisely what D-12 already schedules — the opencode live session must be run WITH Claude's package also installed, and the transcript must show what opencode actually does (list the skill twice, silently pick one, or visibly warn) rather than assuming either outcome. If it silently picks one, that is not itself a bug (content is identical either way), but if it produces a visible duplicate or an error, that's the trigger for D-06's "add a guard for that pairing" escalation named directly in D-12's text.
**Warning signs:** An opencode live-session transcript showing two `codegraph` entries in its skill list, or an error/warning about duplicate skill names.
**Phase to address:** D-12's opencode live session, exactly as already planned — this finding sharpens *what to look for* in that transcript, it doesn't change the plan's shape.

### Pitfall 4: Cursor's and Kiro's SKILL.md `name` validation both require the folder name to match `name:` exactly — codegraph's frontmatter already satisfies this, but a future SKILL.md edit could silently break it per-harness

**What goes wrong:** `[CITED: cursor.com/docs/skills.md, fetched 2026-09-18]` — "`name` ... Must match the parent folder name." `[CITED: opencode.ai/docs/skills.md, fetched 2026-09-18]` — "Match the directory name that contains `SKILL.md`." `[CITED: kiro.dev/docs/skills/, via WebFetch summary, fetched 2026-09-18]` — "Must match folder name." All three enforce this at validation time; codegraph's frontmatter (`[VERIFIED: .claude/skills/codegraph/SKILL.md:1-3]` — `name: codegraph`) already matches its folder name (`codegraph`) at every write location this phase plans (`.agents/skills/codegraph/`, `.gemini/skills/codegraph/`, `.kiro/skills/codegraph/`), so there is no live risk today — but any future change that renames the *directory* the skill is written into (without also updating the frontmatter `name:` field, or vice versa) would silently break discovery on three harnesses simultaneously, with no compiler or `go vet` to catch it.
**Why it happens:** The directory name and the frontmatter `name:` field are two independent strings that happen to agree today by construction (the shared writer always creates a directory literally named `codegraph`), but nothing enforces the invariant going forward.
**How to avoid:** If a future change to the shared writer's directory-naming logic is ever proposed, add (or extend) a unit test asserting the written directory's basename equals the embedded `SKILL.md`'s `name:` frontmatter value, parsed from the same embedded content the writer itself uses — this is a cheap, purely in-repo assertion (no live session needed) that closes the gap for all three harnesses at once.
**Warning signs:** A PR that changes `sharedSkillDirPath`'s literal `"codegraph"` segment without touching `.claude/skills/codegraph/SKILL.md`'s frontmatter (or the reverse).
**Phase to address:** Not required for this phase (no such directory-rename is planned) — noted for the D-01…D-08 implementation tasks as a cheap defensive assertion worth adding alongside the capability-table guard, since it costs nothing and protects three harnesses at once.

### Pitfall 5 (carried from milestone `PITFALLS.md` #10, sharpened): shared-array-entry ownership by shape/position is a REVERTED vulnerability in THIS exact codebase

See Critical Finding #5 above for the precise differential and the regression test's shape. The milestone-level pitfall entry (PITFALLS.md #10) states the general rule; this phase's specific application is: **the D-13 planted-foreign-entry test's precondition must be "manifest present at this location, AND an unrelated entry shares codegraph's own matcher/directory-name/key," not merely "an unrelated entry exists somewhere."** A weaker precondition (e.g. just planting an unrelated `.agents/skills/other/` directory, which the phase's own D-13 text also includes) tests a different, easier property (non-interference with an unrelated *name*) and would not by itself have caught the `242ec0a` vulnerability, which required the *same* matcher name. Both preconditions belong in the test table — the same-name/same-key one is the one that actually reproduces the historical incident.

### Pitfall 6 (carried from milestone `PITFALLS.md` #12, sharpened for this phase's specific harnesses): "fresh session" evidence for `agy`/`opencode`/`cursor-agent` needs a real transcript with a negative-space check — this phase's D-09/D-12 already mandate this, but the *how* is concrete now

`herdr agent read <target> --source recent-unwrapped` (`[VERIFIED: this session, `herdr agent read --help`]`, `--source` accepts `visible|recent|recent-unwrapped|detection`) is the mechanism named in this phase's own Additional Context for reading back a driven session's transcript without terminal line-wrap corrupting the captured text — use `recent-unwrapped` specifically for any transcript excerpt that will be pasted verbatim into `05-LIVE-SESSIONS.md`, not `visible` (which is viewport-bounded) or the default `recent`. `herdr agent start <name> --kind cursor|opencode|agy --pane <id>` (`[VERIFIED: this session, `herdr agent start --help`]`) is confirmed to accept exactly the three kinds D-09 names, plus `codex`/`claude` for symmetry if a future phase needs them.

## Code Examples

### The `Capabilities` struct — concrete field values derived from all 8 target files read this session

```go
// Source: derived this session from reading all 8 internal/agents/*.go
// target files' existing path-resolution functions and Install/Uninstall
// bodies. Field NAMES/types are Claude's Discretion per D-02; the VALUES
// below are load-bearing (each cites the line evidence it came from).
type Capabilities struct {
	Scopes       []Location
	MCPConfig    map[Location]string // resolved path, not a literal — call the target's existing path func
	Instructions map[Location]string // "" (absent key) = none
	SkillDirs    map[Location][]string // ordered; [0] is written, rest are documented read-paths only
	Hooks        HookMechanism // enum: HooksNone | HooksClaudeJSON | HooksCodexJSON
	ConfigFormat ConfigFormat  // enum: json | jsonc | toml | yaml
}
```

| Target | Scopes | ConfigFormat | Instructions (today) | SkillDirs (today) | Hooks (today) |
|---|---|---|---|---|---|
| Claude (`claude.go`) | global, local `[VERIFIED:33]` | json | `.claude/CLAUDE.md` local / `~/.claude/CLAUDE.md` global `[VERIFIED:101-110]` | `.claude/skills/codegraph` local / `~/.claude/skills/codegraph` global `[VERIFIED:133-142]` | `claude-json` (SessionStart today; PreToolUse is Phase 6) `[VERIFIED:469]` |
| Cursor (`cursor.go`) | global, local `[VERIFIED:25]` | json | none today `[VERIFIED:14-16]` — AGENT-04 may add repo-root AGENTS.md, gated on D-11 | none today — AGENT-04 adds the shared `.agents/skills/` write | none |
| Codex (`codex.go`) | global only `[VERIFIED:28]` | toml | `~/.codex/AGENTS.md` global only `[VERIFIED:38-44]` | none this phase (Phase 7 scope) | none this phase |
| opencode (`opencode.go`) | global, local `[VERIFIED:34]` | jsonc | `AGENTS.md` local / `<cfgdir>/opencode/AGENTS.md` global `[VERIFIED:79-88]` | none today — AGENT-06 adds the shared `.agents/skills/` write | none |
| Hermes (`hermes.go`) | global only `[VERIFIED:31]` | yaml | none, by design `[VERIFIED:20-22]` | none (v2, AGENT-12) | none |
| Gemini (`gemini.go`) | global, local `[VERIFIED:22]` | json | `GEMINI.md` local (project root) / `~/.gemini/GEMINI.md` global `[VERIFIED:35-46]` | none today — AGENT-10 adds `.gemini/skills/codegraph` (harness-specific) AND, per Finding #1, is also covered by the shared write | none |
| Antigravity (`antigravity.go`) | global only `[VERIFIED:26-27]` | json | none of its own — shares Gemini's GEMINI.md `[VERIFIED:14-15]`; see Finding #3 | none today — AGENT-07 targets the shared `.agents/skills/` write; see Finding #2 for the global-path risk | none |
| Kiro (`kiro.go`) | global, local `[VERIFIED:27]` | json | none today `[VERIFIED:16-18]` — AGENT-11 adds `AGENTS.md` per Finding #4 | none today (legacy steering self-heal-deleted) — AGENT-11 adds `.kiro/skills/codegraph` (harness-specific) | none |

### `242ec0a`'s regression-test shape — the template for AGENT-13's D-13 test

```go
// Source: git show 242ec0a — internal/agents/claude_skillpackage_test.go,
// TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher
// (quoted structure, not verbatim — see the commit for the full body)
first := c.Install(LocationGlobal, opts)                 // 1. establish a manifest at this location
// ... assert manifest present ...
writeFile(t, settingsPath, `{
  "hooks": { "SessionStart": [
    {"matcher": "startup", "hooks": [
      {"type": "command", "command": "/opt/some-other-tool/on-startup.sh"}
    ]}
  ]}
}`)                                                        // 2. plant an unrelated entry under the SAME matcher
second := c.Install(LocationGlobal, opts)                  // 3. re-install
// 4. assert: the unrelated command survives byte-for-byte;
//    codegraph's own blocks are appended alongside it, never merged in
```

### herdr commands for D-09's live sessions (confirmed available on this machine, 2026-09-18)

```bash
# start a fresh agent in an existing pane, at the interactive shell prompt
herdr agent start scratch-cursor --kind cursor --pane <id>
herdr agent start scratch-opencode --kind opencode --pane <id>
herdr agent start scratch-agy --kind agy --pane <id>

# submit the prompt and wait, then read back the transcript WITHOUT line-wrap corruption
herdr agent prompt scratch-cursor "list your available skills" --wait
herdr agent read scratch-cursor --source recent-unwrapped
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Skill written only to `.claude/skills/` (Claude-only, v0.10.0 Phase 7) | Shared `.agents/skills/` write reaches Cursor/opencode/Antigravity/Gemini in one write; Gemini/Kiro get an additional harness-specific dir | This phase (v0.14.0 Phase 5) | Five more harnesses gain skill discovery from one shared writer, per D-05 |
| `codegraph.go`'s "no per-project config" comment (v1.0-era, already corrected in v0.14.0 planning per `PITFALLS.md` Pitfall 11 — not this phase's scope, Phase 7) | N/A for this phase | Phase 7 | Not in this phase's boundary; noted only because AGENT-08's capability table's `Scopes` field for Codex should stay `[LocationGlobal]` until Phase 7 flips it, not anticipate the flip early |
| `--print-config-style` named only in a doc comment, never implemented (`[VERIFIED: internal/agents/types.go:157]`) | Implemented as a read-only reporting flag on `install`, deriving from `Capabilities()` | This phase | Closes a documentation/implementation gap that has existed since the comment was written |

**Deprecated/outdated:**
- The milestone `STACK.md`'s claim that "Gemini CLI still primarily uses its own `GEMINI.md`... Gemini CLI does **not** appear to treat `AGENTS.md` as a first-class instructions file" remains correct for *instructions*, but its skills-related framing needs updating: Gemini CLI's `.agents/skills/` support (a *different* mechanism from `GEMINI.md`) is real and precedence-favored — see Critical Finding #1. Don't conflate the two claims when updating downstream docs.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Gemini CLI's `.agents/skills/` precedence-over-`.gemini/skills/` claim (Finding #1) holds in the *installed* version the maintainer would actually run, not just in the `main` branch docs fetched today | Critical Finding #1 | Gemini's harness-specific write (AGENT-10) stays necessary rather than optional; no harm if kept — this only affects whether it's *redundant*, not whether it's *wrong* |
| A2 | Antigravity's `agy` CLI kind (the one D-09 will actually herdr-drive) resolves its global skill path to exactly `~/.gemini/antigravity-cli/skills/codegraph/` as documented, with no undocumented override env var | Critical Finding #2 | If wrong, the harness-specific escape-valve path chosen after a failed shared-path live check would also fail; the live session itself (D-09) is the actual falsification/confirmation step, not this research |
| A3 | Antigravity's own `AGENTS.md`-adjacent claim (no literal global `AGENTS.md` file is read) generalizes across all three Antigravity surfaces (2.0, CLI, IDE), not just the one page fetched | Critical Finding #3 | Low risk either way — the existing code (share GEMINI.md, write nothing new) already matches every surface's documented global mechanism in the page fetched |
| A4 | Kiro and Gemini CLI's documented behavior (Findings #1, #4) matches what a live session on those products would actually show — genuinely unverifiable this session per D-10 (neither is installed on this machine) | Phase Requirements table (AGENT-10, AGENT-11); D-10 | If the maintainer installs Gemini CLI or Kiro before execution, these should be upgraded from `[ASSUMED]`/`[CITED]` to a live-verified `[VERIFIED]` row in `05-LIVE-SESSIONS.md` per D-10's own escape hatch |
| A5 | Cursor's undocumented duplicate-skill-name resolution (not stated in `cursor.com/docs/skills.md`) behaves reasonably (last-write-wins by precedence order, or shows both) rather than erroring | Per-Harness Findings → Cursor; D-12 | If Cursor errors or silently drops the skill entirely on a name collision, D-12's duplicate-skill check on Cursor needs the SAME live-session rigor as opencode's, not less — the doc gap here is real, not just unresearched |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Does opencode's undocumented duplicate-name behavior (Pitfall 3) resolve silently or visibly?**
   - What we know: opencode's own docs flag "ensure names are unique across locations" as a troubleshooting step, implying this IS a real failure mode, not a handled case.
   - What's unclear: whether it silently keeps one copy (and which), errors, or lists both under one collapsed entry.
   - Recommendation: this is exactly what D-12's opencode live session (with Claude's package also installed) is scheduled to determine — no further research action needed before that session runs.

2. **Should the harness-specific Antigravity skill write (if D-09's live session confirms the shared path is not read, per Finding #2's prior doc evidence) target the CLI path (`~/.gemini/antigravity-cli/skills/`) or the 2.0/IDE path (`~/.gemini/config/skills/`), or both?**
   - What we know: the herdr-drivable surface is specifically `agy` (CLI); the 2.0/IDE surfaces are GUI products this milestone's live-session method (herdr, a terminal tool) cannot drive.
   - What's unclear: whether the maintainer's actual day-to-day Antigravity usage is the CLI, the IDE, or both — this determines which path(s) matter in practice, beyond what D-09's live session alone can confirm.
   - Recommendation: if D-06's escape valve is triggered for Antigravity, default to the CLI path (`~/.gemini/antigravity-cli/skills/codegraph/`) since that's what D-09 can actually verify live; flag the IDE/2.0 path as `[ASSUMED]` alongside it if written, same as Gemini/Kiro's D-10 treatment.

3. **Does the capability table's `Hooks` enum need a third, non-`claude-json`/`codex-json` value this phase, given no harness besides Claude has a hook mechanism today?**
   - What we know: D-02 names exactly `{none, claude-json, codex-json}` — `codex-json` anticipates Phase 7's work, not this phase's.
   - What's unclear: nothing material — this is settled by D-02's literal text, not a research gap.
   - Recommendation: implement exactly the three-value enum D-02 specifies; every non-Claude target's `Capabilities()` literal returns `HooksNone` this phase.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `claude` (Claude Code CLI) | D-09 negative control / existing behavior baseline | ✓ | 2.1.277 (verified this session; CONTEXT.md recorded 2.1.275 on 2026-09-18 morning — normal auto-update drift) | — |
| `codex` | Symmetry check only (Codex is Phase 7 scope) | ✓ | codex-cli 0.155.0 | — |
| `cursor-agent` | D-09/D-11 live sessions (AGENT-04) | ✓ | 2025.09.12-4852336 | — |
| `opencode` | D-09/D-12 live session (AGENT-06) | ✓ | 1.18.30 | — |
| `agy` (Antigravity CLI) | D-09 live session (AGENT-07) | ✓ | 1.1.11 | — |
| `gemini` (Gemini CLI) | Would upgrade AGENT-10 from `[ASSUMED]` to verified | ✗ | — | D-10: record `[ASSUMED]` with doc citation; maintainer may install via `dotfiles-upgrade` |
| `kiro` / `kiro-cli` | Would upgrade AGENT-11 from `[ASSUMED]` to verified | ✗ | — | D-10: record `[ASSUMED]` with doc citation |
| `hermes` | Out of scope this phase (AGENT-12, v2) | ✗ | — | N/A — not required |
| `tmux` | Not required this phase (D-09 uses Herdr panes, not tmux) | ✗ | — | N/A |
| `herdr` | D-09/D-11/D-12 live-session driver | ✓ | (CLI present; `agent start`/`agent read` subcommands confirmed with `--kind cursor\|opencode\|agy\|codex\|claude\|...` and `--source recent-unwrapped` this session) | — |

**Missing dependencies with no fallback:** none — every missing tool (`gemini`, `kiro`, `hermes`, `tmux`) has an explicit, already-decided fallback (D-10, or simply out of scope).

**Missing dependencies with fallback:** `gemini`, `kiro` — `[ASSUMED]` doc-cited evidence per D-10.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's standard `testing` package (`go test`) |
| Config file | none — this repo has no test-framework config beyond `go.mod`'s toolchain pin |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... ./internal/cli/... -count=1` |
| Full suite command | `GOTOOLCHAIN=go1.26.6 go test $(go list ./... \| rg -v 'internal/daemon$') -count=1` plus `GOTOOLCHAIN=go1.26.6 go test ./internal/daemon/... -count=1` run separately (WINDOWS #37 open flake under cross-package parallel load — this phase's own changes are outside `internal/daemon`, so this is isolation, not a new flake risk) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AGENT-08 | `DescribePaths(loc)` equals the path set derived from `Capabilities()`, for all 8 targets × both locations; `--print-config-style` renders exactly the table | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... -run TestCapabilitiesTableConsistency -count=1` (new test, name illustrative) | ❌ Wave 0 — new test file |
| AGENT-09 | Shared writer: one write per install run, `unchanged` on the 2nd+ target in the same run; manifest `targets` grows additively | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... -run TestSharedSkillPackage -count=1` | ❌ Wave 0 — new test file (`skillshared_test.go`, mirroring `claude_skillpackage_test.go`'s style) |
| AGENT-04/06/07/10/11 | Per-harness install/uninstall writes the documented path(s); instructions marker upsert where applicable | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... -run TestCursor_\|TestOpencode_\|TestAntigravity_\|TestGemini_\|TestKiro_ -count=1` | ✅ existing `*_test.go` files per target, extended |
| AGENT-04, AGENT-06, AGENT-07 (live evidence) | The harness actually reads the written path unprompted | manual-only (D-00: never a `go test` assertion) | N/A — recorded in `05-LIVE-SESSIONS.md` via herdr | ❌ Wave 0 — `05-LIVE-SESSIONS.md` |
| AGENT-13 | Planted-foreign-entry survives byte-identical through install→uninstall, for all 8 targets × both locations; a planted ownership-recovery mutation goes RED | unit, RED-first (TDD mode) | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... -run TestOwnershipExactIdentity -count=1` | ❌ Wave 0 — new test file, written RED-first per D-13/tdd_mode |

### Sampling Rate
- **Per task commit:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... ./internal/cli/... -count=1`
- **Per wave merge:** the full suite command above (both `go test` invocations)
- **Phase gate:** full suite green before `/gsd-verify-work`, plus `05-LIVE-SESSIONS.md` populated for every harness D-09/D-10 name

### Wave 0 Gaps
- [ ] `internal/agents/skillshared_test.go` — covers AGENT-09 (new file, no existing test to extend)
- [ ] `internal/agents/capabilities_test.go` (or similarly named) — covers AGENT-08's D-03 table-equality guard (new file)
- [ ] `internal/agents/ownership_test.go` (or similarly named) — covers AGENT-13's D-13 planted-foreign-entry table, written RED-first per `tdd_mode: true`
- [ ] `internal/agents/registry_test.go`'s `fakeTarget` and `internal/cli/tui/agentpicker_test.go`'s `fakeAgentTarget` — both need a `Capabilities()` stub added (Pitfall 1) before ANY of the above compiles
- [ ] `.planning/phases/05-.../05-LIVE-SESSIONS.md` — the D-12 evidence artifact; not a Go test, but a required phase-gate artifact per D-00/D-12

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A — no auth surface touched |
| V3 Session Management | no | N/A |
| V4 Access Control | **yes** | Exact-identity ownership (never shape/position) for every shared-array-entry and shared-directory write, per `242ec0a` — this IS the access-control property this phase's AGENT-13 exists to enforce: "who owns this entry" is the access-control question, and the answer must never be inferable from an attacker/user-controlled shape alone. |
| V5 Input Validation | **yes** | Every path this phase writes is either a fixed literal or derived from `os.UserHomeDir()`/`os.Getwd()` — no user-supplied string is interpolated into a filesystem path this phase adds (consistent with the existing 8 targets' pattern); the one adversarial-capable value already sanitized elsewhere (`sanitizePathForDisplay`, `[VERIFIED: internal/cli/install.go:180,187,201]`) applies unchanged to any new path this phase prints via `--print-config-style`. |
| V6 Cryptography | no | The manifest's `sha256` hashing (`hashContent`, `[VERIFIED: internal/agents/manifest.go:59-62]`) is explicitly documented as a drift signal, never a security/authenticity control (`[VERIFIED: internal/agents/manifest.go:9-15]`) — this phase's `targets` field addition does not change that classification. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Ownership-by-shape/position recovery heuristic silently overwriting an unrelated user-authored config entry (the exact `242ec0a` incident) | Tampering / Elevation of Privilege (codegraph gains write authority over content it never wrote) | Exact-identity matching only (command string, manifest-hash, or directory-manifest-presence) — never matcher name, key name, or array position. AGENT-13's D-13 test, RED-first, is the regression guard. |
| A duplicate same-named skill across `.claude/skills/` and `.agents/skills/` causing a harness to silently prefer stale/wrong content | Tampering (content confusion, not malicious but a real correctness risk) | D-05 mitigates by construction — both locations always carry byte-identical content sourced from the same embed, so even an undocumented "last one wins" resolution can never surface divergent content. D-12's live check confirms this holds in practice for the two harnesses (Cursor, opencode) that read both paths. |
| A hand-edited manifest (hash mismatch) being treated as an authenticity signal rather than a drift signal | Repudiation (a user could not "sign" a manifest, and this project must not imply they can) | `manifest.go`'s own doc comment already states this explicitly (`[VERIFIED: internal/agents/manifest.go:9-15]`) and D-16 preserves that posture for the new `targets` field — no code in this phase may treat a hash mismatch as a security event. |

## Sources

### Primary (HIGH confidence)
- This repo, read directly this session: `internal/agents/{types,registry,shared,manifest,instructions,claude,cursor,codex,opencode,hermes,gemini,antigravity,kiro}.go`, `claudeassets.go`, `internal/cli/{install,uninstall}.go`, `internal/cli/present/{line,palette}.go`, `internal/cli/colorflag.go`, `internal/agents/{testhelpers_test,registry_test,claude_skillpackage_test}.go`, `internal/cli/tui/agentpicker_test.go`, `.claude/skills/codegraph/SKILL.md`
- `git show 242ec0a` (this repo's own history, read in full this session)
- `cursor.com/docs/skills.md`, `cursor.com/docs/rules.md` (raw markdown, fetched 2026-09-18)
- `opencode.ai/docs/skills.md`, `opencode.ai/docs/rules.md`, `opencode.ai/docs/config.md` (raw markdown, fetched 2026-09-18)
- `antigravity.google/docs/skills.md`, `antigravity.google/docs/rules-workflows.md` (raw markdown, fetched 2026-09-18)
- `kiro.dev/docs/steering/` (rendered HTML, text-extracted, fetched 2026-09-18)
- `raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md` (raw markdown, fetched 2026-09-18)
- This session's direct shell verification: `readlink`/`stat -f "%i"`/`diff` on `~/.claude/skills/codegraph` vs `~/.agents/skills/codegraph`; `command -v`/`--version` for all 10 named CLI tools; `herdr agent --help`/`herdr agent start --help`/`herdr agent read --help`

### Secondary (MEDIUM confidence)
- `.planning/research/STACK.md`, `.planning/research/PITFALLS.md`, `.planning/research/ARCHITECTURE.md` (milestone-level research, 2026-09-14 — several claims corrected by this session's direct fetches, see Critical Findings)
- `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-CONTEXT.md` (locked decisions, 2026-09-18)
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md` (project decisions/history, read this session)

### Tertiary (LOW confidence)
- WebSearch-synthesized summaries used only to locate the primary-source URLs above (not cited as evidence in their own right anywhere in this document) — e.g. Medium/dev.to Antigravity-skills posts, thepromptshelf.dev's AGENTS.md guide. None of these appear as a standalone citation for a factual claim; every factual claim above traces to a primary-source fetch or this repo's own source.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency, confirmed by reading every touched file
- Architecture (capability table, shared writer): HIGH for the existing-code patterns being extended (all read directly); MEDIUM for the exact `Capabilities` field types (explicitly Claude's Discretion per D-02, not yet decided)
- Per-harness skill/instructions paths: HIGH for Cursor/opencode/Antigravity/Kiro (raw primary-source docs fetched today); MEDIUM-HIGH for Gemini CLI (raw primary source fetched, but the harness itself is not installed here to cross-check — D-10 applies); LOW for anything requiring a live session not yet run (that is expected — D-00 forbids a unit test from ever reaching HIGH confidence on this axis; only `05-LIVE-SESSIONS.md` can)
- Pitfalls: HIGH — the symlink collision (Pitfall 2) and the interface-fake break (Pitfall 1) were both directly observed/verified this session, not inferred

**Research date:** 2026-09-18
**Valid until:** ~14 days for the per-harness doc claims (Cursor/opencode/Antigravity/Gemini/Kiro docs are all actively-developed 2026-era products that have already changed once this milestone, per the milestone `PITFALLS.md`'s own Pitfall 11 finding about Codex) — re-fetch before relying on any un-live-verified per-harness claim if execution slips more than ~2 weeks past this research date. The in-repo architecture findings (capability table shape, existing helper reuse) are stable until the code itself changes.
