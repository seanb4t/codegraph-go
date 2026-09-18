# Phase 5: Agent Reach — Capability Model & Skill in Every Harness - Context

**Gathered:** 2026-09-18
**Status:** Ready for planning

<domain>
## Phase Boundary

One per-target capability table drives `install`, `uninstall`, `Detect` and `--print-config-style` for all eight targets, and every harness that has a skill mechanism receives the codegraph skill package — written once to the shared `.agents/skills/` path and verified live to be read, with a harness-specific directory only where a live session (or, for harnesses not installed on the maintainer's machine, the harness's own docs, labelled `[ASSUMED]`) shows it is needed — every write exact-identity-owned and reversed by `uninstall` leaving unrelated content byte-identical. Requirements: AGENT-08, AGENT-09, AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11, AGENT-13.

In scope: `internal/agents` (types, registry, the eight target files, shared writers, manifest), `internal/cli/install.go`/`uninstall.go` (the new `--print-config-style` flag and its rendering), the regenerated `docs/CLI-REFERENCE.md`, and the phase's evidence artifacts. Out of scope: Codex's per-project config, skill and hooks (Phase 7 — Codex enters the table at its **current** global-only, MCP+`AGENTS.md` shape and Phase 7 edits that one literal); nudge hooks for any harness other than Claude Code (NUDGE-07…10, v2); Hermes beyond its existing MCP config (AGENT-12, v2); the published capability table in docs and the `instructions.go` "4 of 8" comment (AGENT-14, Phase 7); any second skill file (the one embedded `claudeassets.SkillMarkdownPath` is the only skill content).

</domain>

<decisions>
## Implementation Decisions

### Cross-cutting
- **D-00:** The v0.13.0 Phase 12 ruling and Phase 4's D-00 apply: tests assert **our** contract — which paths we write, which bytes we leave, what the table declares — never a harness's behaviour and never Go dependency management. Whether a harness actually *reads* a path is established by a live session (D-09…D-12), not by a unit test. Every new guard is positive-controlled (rule `84d1gfpywd`): a planted violation goes RED, recorded in `05-MUTATION-LOG.md` in the Phase 2/3/4 shape with a byte-clean revert. TDD mode is on: Go RED evidence is the `test(05-NN):` commit before the implementation plus the pasted `--- FAIL:` transcript (rule `x1cjy9vyhq`).

### Capability table (AGENT-08)
- **D-01:** The table lives on the targets: a new `Capabilities() Capabilities` method on `AgentTarget`, returning **one struct literal per target file** — additive, mirroring `SupportsLocation` (ROADMAP note). No central map in `registry.go`. `Install`/`Uninstall` semantics do not change because of the table's existence.
- **D-02:** `Capabilities` declares: `Scopes []Location`; per-location `MCPConfig` path; per-location `Instructions` path (`""` = none); per-location `SkillDirs` (ordered; the first entry is the one written, later entries are documented read-paths only); `Hooks` enum {`none`, `claude-json`, `codex-json`}; and `ConfigFormat` (json/jsonc/toml/yaml — the "style" in `--print-config-style`). `SupportsLocation`, `DescribePaths`, `Detect`'s path inputs and `--print-config-style` become **derivations** of the table — no target keeps a second hand-written copy of any path the table holds.
- **D-03:** The guard (ours): one table test over all eight targets × both locations asserting `DescribePaths(loc)` equals the path set derived from `Capabilities()` and that `--print-config-style` renders exactly the table; a planted divergence in one target (a path in `DescribePaths` the table does not hold, or vice versa) goes RED — Family (a) in `05-MUTATION-LOG.md`.
- **D-04:** `--print-config-style` **does not exist today** (discovery finding, 2026-09-18: only a doc comment in `types.go` names it; TS's `--print-config <id>` was never ported). This phase adds it as a **read-only** flag on `install`: it opens no picker, writes nothing, honours `-l/--location` and `-t/--target` for filtering, and prints one line per target — `<target>: scopes=<…> mcp=<path> format=<style> instructions=<path|none> skill=<path|none> hooks=<mechanism>` — through Phase 4's `present.KV`/`Line` helpers behind the shared `resolveColor` (plain and byte-stable on a pipe; the plain output is golden-frozen as ours). It flows into `docs/CLI-REFERENCE.md` via `task docs:cli` in a reviewed diff; `TestEveryRegisteredFlagIsAccountedFor` stays green with no allowlist entry (it is a visible documented flag). — **Reversibility:** reversible (additive flag).

### Skill package writes (AGENT-09, AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11)
- **D-05:** The shared package — the embedded `SKILL.md` plus the sidecar manifest — is written to `.agents/skills/codegraph/` (local) / `~/.agents/skills/codegraph/` (global) by **one shared writer, once per install run**, however many selected targets list it; later targets in the same run see `unchanged`. **Claude Code's existing `.claude/skills/codegraph/` package (SKILL.md + session-nudge.sh + hooks) is untouched** — it is proven by v0.10.0/Phase 7 live evidence and not migrated.
- **D-06:** Harness-specific skill directories by default: **only** Gemini CLI (`.gemini/skills/codegraph/` / `~/.gemini/skills/codegraph/`) and Kiro (`.kiro/skills/codegraph/` / `~/.kiro/skills/codegraph/`), whose docs do not list `.agents/skills/`. Cursor, opencode and Antigravity rely on the shared path; a harness-specific directory is added for one of them **only** if its live session (D-09) shows the shared path is not read.
  - **Research correction (2026-09-18, primary docs fetched that day — 05-RESEARCH.md Findings 1–2):** (a) **Gemini CLI does read `.agents/skills/`** at both tiers, and the alias *outranks* `.gemini/skills/` within a tier (`google-gemini/gemini-cli` docs/cli/skills.md). D-06's stated reason for Gemini is therefore wrong, but the decision stands: AGENT-10 names `.gemini/skills/` explicitly and writing both is harmless (identical content; `.agents` wins the tie silently). (b) **Antigravity's documented global skill paths are `~/.gemini/antigravity-cli/skills/` (CLI) and `~/.gemini/config/skills/` (2.0/IDE) — never `~/.agents/skills/`**, which it reads only at workspace scope (antigravity.google/docs/skills.md). Antigravity is global-only in this repo, so D-06's escape valve applies on primary-doc evidence: Antigravity's `SkillDirs(global)` is `~/.gemini/antigravity-cli/skills/codegraph/` (the surface `agy` exercises; confirmed by the D-09 session), and the 2.0/IDE path is recorded `[ASSUMED]`, documented-only, **not written** (no live surface to verify it). (c) AGENT-07's "instructions block in `AGENTS.md`" is read as the shared marker-fenced block Antigravity already receives through `~/.gemini/GEMINI.md` (its documented global rules file on all three surfaces) — no new `AGENTS.md` write for Antigravity. (d) AGENT-11's "`AGENTS.md` retained as a steering source" is read as **no change** to Kiro's instructions: `.kiro/steering/codegraph.md` stays; Kiro also reads `./AGENTS.md` / `~/.kiro/steering/AGENTS.md` (kiro.dev/docs/steering), so a codegraph block written there by opencode/Cursor is picked up too — recorded as an advisory (possible doubled instructions), not changed.
- **D-07:** One sidecar manifest per written skill directory, in the existing `skillManifest` format extended **additively** with `targets: [<target ids that requested this dir>]` (schema_version bumped; an older manifest without `targets` is read as "owned, requester set unknown" and upgraded in place on the next install).
  - **Planner amendment (2026-09-18, accepted):** a pre-`targets` manifest is read as owned by `claude`, not as "requester set unknown". Claude was the only writer before this phase, and any other reading lets the D-17 symlinked layout delete the package out from under Claude. The `schema_version` bump is flagged `costly` in plan 05-02: a released binary will read manifests this one writes.
  - **Planner note (accepted):** Gemini writes only its first `SkillDirs` entry (`.gemini/skills/codegraph/`, per D-02). `.agents/skills/` is recorded as a documented read path for Gemini, not a second write.
- **D-08:** Uninstall of a shared directory removes **only the uninstalling target** from the manifest's `targets`; the package (SKILL.md + manifest) is deleted only when `targets` becomes empty. A harness-specific directory (D-06) has a single requester and is removed with its target. `removeSkillDirIfEmpty` semantics are kept.

### Live verification (the evidence rows)
- **D-09:** Method: **Herdr-driven fresh sessions** on the maintainer's machine for the harnesses installed there — Cursor (`cursor-agent`), opencode, Antigravity (`agy`) — started with `herdr agent start <name> --kind cursor|opencode|agy --pane <id>` in a sibling pane: install codegraph into a scratch project, ask the agent to list its skills and to answer a where-is-X question, capture the transcript with `herdr agent read`; then a **negative control** — the same prompt in a scratch project with nothing installed must not show the skill. This is the v0.10.0 live-session standard with its negative-space check (research pitfall 12). The orchestrator runs these; they are not delegated to executor subagents (a backgrounded subagent cannot drive another pane's TTY reliably).
- **D-10:** Gemini CLI and Kiro are **not installed** on the maintainer's machine (checked 2026-09-18). Their documented paths are written (D-06) and AGENT-10/AGENT-11 evidence is recorded **`[ASSUMED]`** with the doc citation and fetch date — never claimed verified — unless the maintainer installs them before execution (installing them is the maintainer's call, via `dotfiles-upgrade`).
  - **Maintainer decision (2026-09-18):** the maintainer has **no Cursor account**, so `cursor-agent` cannot be authenticated and no live Cursor session can run. Cursor joins Gemini CLI and Kiro as `[ASSUMED]`: its shared-path read is recorded from Cursor's docs (URL + fetch date), never claimed verified. Its install behaviour is unchanged; unit tests still pin exactly what it writes (D-00).
- **D-11:** Cursor's repo-root `AGENTS.md` pickup (AGENT-04) is probed live: plant a distinctive sentence in a scratch repo's `AGENTS.md` (no `.cursor/rules/` file present), ask a fresh `cursor-agent` to repeat project instructions; **only** an observed pickup switches Cursor's instructions target from `.cursor/rules/codegraph.mdc` to repo-root `AGENTS.md`. The outcome is recorded either way; the probe precedes any Cursor instructions-target change.
  - **Maintainer decision (2026-09-18):** the probe **cannot run** (no Cursor account). The recorded verdict is `D-11 verdict: not probed`. D-11 only permits the instructions-target switch after an observed pickup, so no Cursor instructions file is written and Cursor keeps writing none. 05-07 takes its no-change branch. AGENT-04's "probe outcome recorded either way" is met by recording that the probe could not run and why.
- **D-12:** All live evidence goes in `05-LIVE-SESSIONS.md`: per harness — the exact command, the transcript excerpt, the negative control, and the verdict (`read` / `not read` / `[ASSUMED]`). A duplicate-skill check is part of each positive session: harnesses documented to read **both** `.claude/skills/` and `.agents/skills/` (Cursor, opencode) are probed with Claude's package also installed, and a doubled codegraph skill is recorded as a finding (it decides whether D-05's shared write needs a guard for that pairing).
  - **Research correction (2026-09-18, 05-RESEARCH.md Pitfalls 2–3):** Pre-flight before any global-scope session: `readlink ~/.claude/skills/codegraph ~/.agents/skills/codegraph` — on the maintainer's machine the former is a symlink into the latter (the `npx skills` convention), so a byte-identical result at both paths proves nothing about the shared write being read at a distinct location. Cursor and opencode sessions run at **project** scope in a scratch repo (their local scope is supported); only Antigravity (global-only) uses the real `$HOME`, with a before/after listing of `~/.gemini/antigravity-cli/skills/` and a `codegraph uninstall` cleanup. opencode's own docs warn that duplicate skill names across its locations are a loading problem, so its D-12 duplicate check is expected to be informative, not a formality.

### Ownership & reversal (AGENT-13)
- **D-13:** One planted-foreign-entry table test over all eight targets × both locations, written **RED-first**: seed a foreign MCP server entry, a foreign `.agents/skills/other/` directory, a foreign `.agents/skills/codegraph/` directory **without** a codegraph manifest, and a foreign marker-less section in every instructions file → `install` → `uninstall` → every foreign byte identical, every own entry gone (Family (b), and a planted ownership-recovery mutation goes RED).
- **D-14:** Ownership of a skill directory = **our sidecar manifest**. A `codegraph/` skill directory without it is foreign: never overwritten, never removed, reported as `kept (foreign)` in the write result — the same exact-identity rule as `writeHookEntry`.
- **D-15:** Commit `242ec0a` (the reverted hook-block ownership recovery that closed an authorization differential) is cited in the D-13 test file's doc comment, naming the differential it closed, and in the plan's review notes (AGENT-13's "cited in review").
- **D-16:** A hand-edited **own** file (manifest hash mismatch) keeps the existing manifest D-05 response — codegraph-owned content is rewritten and the mismatch recorded; only unmanifested content is untouchable.
- **D-17 (added 2026-09-18 from 05-RESEARCH.md Pitfall 2 — a correctness case the accepted D-05/D-08 did not cover):** Claude's `.claude/skills/codegraph/` and the shared `.agents/skills/codegraph/` can resolve to the **same physical directory** (a symlink — the `npx skills` convention, present on the maintainer's machine today). Both writers compare the two locations with `filepath.EvalSymlinks` before writing: when they resolve to the same directory, there is **one** package with **one** manifest, the Claude target is recorded in that manifest's `targets` like any other requester, and Claude's uninstall follows D-08's last-requester rule instead of deleting the package out from under Cursor/opencode. Claude's session-nudge script and hooks are outside the skill directory and unaffected. A test with a symlinked `fakeHome` layout pins it (RED first).

### Claude's Discretion
- The exact `Capabilities` field names and Go types, and whether per-location paths are methods or maps on the struct — subject to D-02's content and D-01's one-literal-per-file shape.
- File layout for the shared skill writer (a new `skillshared.go` vs extending `shared.go`).
- The scratch-project layout and exact prompts for the live sessions, provided each has a negative control (D-09/D-12).
- Plan ordering, subject to: capability table + its guard first (D-01…D-04) → ownership test RED-first (D-13) → shared/harness-specific skill writers (D-05…D-08) → per-harness wiring → live sessions (D-09…D-12) → `docs/CLI-REFERENCE.md` regeneration through the drift gate.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/agents/types.go` — `AgentTarget` interface (`ID`, `DisplayName`, `SupportsLocation`, `Detect`, `Install`, `Uninstall`, `DescribePaths`); `TargetID` constants for the eight targets.
- `internal/agents/registry.go` — `registerTarget`, `GetTarget`, `AllTargetIDs`, `AllTargets`, `DetectAll`, `ResolveTargetFlag`.
- `internal/agents/shared.go` — `writeHookEntry` (exact-identity ownership), `replaceOrAppendMarkedSection` (marker-fenced upsert), `removeSkillDirIfEmpty`, raw-byte idempotent artifact writes (D-07 of v0.10.0).
- `internal/agents/manifest.go` — `skillManifest{SchemaVersion, CodegraphVersion, InstalledAt, Location, Files}` + `hashContent` (sha256) — the drift signal, explicitly not tamper-detection.
- `internal/agents/instructions.go` — `codegraphSectionStart`/`End` markers (a hard contract) and `codegraphInstructionsBlock` (Claude, Codex, opencode, Gemini get it today).
- `claudeassets.go` (repo root, package `claudeassets`) — `FS` embeds exactly `.claude/skills/codegraph/SKILL.md`, `.claude/hooks/hooks.json`, `.claude/hooks/session-nudge.sh`; must stay at the root (golang/go#46056). The shared package reuses `SkillMarkdownPath` — no new embed.
- Phase 4's `present.KV`/`Line`/`NewLineWriter`, `resolveColor(cmd)` and the plain-golden harness (`internal/cli/plain_golden_test.go`) — for `--print-config-style`.

### Established Patterns
- Today's per-target writes: Claude → `.claude/settings`/`.mcp.json`, `.claude/skills/codegraph/`, hooks; Cursor → `.cursor/mcp.json` + `.cursor/rules/codegraph.mdc`; Codex → `~/.codex/config.toml` + `~/.codex/AGENTS.md` (global-only); opencode → `opencode.json(c)` + `AGENTS.md`; Gemini → `.gemini/settings.json` + `GEMINI.md`; Antigravity → `~/.gemini/antigravity/mcp_config.json` (global-only); Kiro → `.kiro/settings/mcp.json` + `.kiro/steering/codegraph.md`; Hermes → MCP only (global-only).
  - **Correction (2026-09-18, found by the planner and confirmed in `cursor.go`):** the line above was built from path strings, not from what the code does with them. `.cursor/rules/codegraph.mdc` and Kiro's `.kiro/steering/codegraph.md` are **legacy files the current code deletes** (pre-#529); Cursor and Kiro write **no instructions file today**. D-06(d) and D-11's "no change" therefore mean "keep writing none", which is what the plans do.
- Exact-identity ownership, never shape/position ownership — `242ec0a` reverted a recovery path that created an authorization differential (research pitfall 10).
- Uninstall reverses every write and leaves unrelated content byte-identical; tests use planted siblings.
- Local harness CLIs present (2026-09-18): `claude` 2.1.275, `codex` 0.154.0, `cursor-agent` 2025.09.12, `opencode` 1.18.30, `agy` 1.1.11; absent: `gemini`, `kiro`/`kiro-cli`, `hermes`; `tmux` absent. `~/.agents/skills/` already exists on the maintainer's machine (other skills live there — foreign siblings are real, not hypothetical).

### Integration Points
- `internal/cli/install.go` (flags: `-t/--target`, `-l/--location`, `--auto-allow`, `-y/--yes`) — add `--print-config-style`; `printAgentResults` already has Phase 4's styled branch.
- `internal/cli/uninstall.go` — reaches the shared writer's reversal through each target's `Uninstall`.
- `docs/CLI-REFERENCE.md` via `task docs:cli` / `task docs:cli:drift`; `internal/cli/cli_reference_test.go`.

</code_context>

<specifics>
## Specific Ideas

- Research must resolve, per harness, from current primary docs (not memory — pitfall 11): the exact skill read-paths and precedence, whether duplicate same-named skills across `.claude/skills/` and `.agents/skills/` are de-duplicated or shown twice, SKILL.md frontmatter requirements (opencode compatibility is an explicit AGENT-06 criterion), Antigravity's instructions location for AGENT-07 (its `AGENTS.md` path at global scope), and what "AGENTS.md retained as a steering source" means for Kiro (AGENT-11) given Kiro writes `.kiro/steering/codegraph.md` today.
- The `herdr agent` kinds available locally include `cursor`, `opencode`, `agy`, `codex`, `claude` — use them for D-09 so agent state (idle/working/blocked) is tracked by Herdr rather than inferred from pane text.
- Every `[ASSUMED]` carries a URL and fetch date so Phase 7's AGENT-14 table can be kept honest.

</specifics>

<deferred>
## Deferred Ideas

- Migrating Claude Code's skill package to the shared `.agents/skills/` path — declined for this phase (A2 Q1); revisit only if D-12's duplicate-skill finding shows a real conflict.
- Installing Gemini CLI and Kiro to upgrade AGENT-10/AGENT-11 from `[ASSUMED]` to verified — the maintainer's call (D-10).

</deferred>
