# Phase 7: Codex Parity - Context

**Gathered:** 2026-09-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Codex receives everything Claude Code does:
- project-local MCP config through a **fixed** TOML splice;
- the codegraph skill package at the path Codex is live-proven to read;
- a repo-root `AGENTS.md` block, shared safely with opencode;
- an opt-in PreToolUse nudge carrying Phase 6's contract.

Every mechanism is verified in a live scratch project before a line of `codex.go` changes. The phase also covers:
- FIX-03's agent-picker footer, fixed at its cause;
- AGENT-14's published per-harness capability table, kept honest by a drift test;
- evidence that a genuinely fresh Codex session reaches for codegraph unprompted.

The milestone's docs close here (CODEX-01…06, FIX-03, AGENT-14).

Out of scope:
- quoting Claude's existing hook commands (AR-06-08);
- a Codex SessionStart nudge;
- Codex plugin packaging;
- `upgrade` refreshing non-Claude packages;
- live checks for Cursor, Gemini CLI and Kiro, which stay `[ASSUMED]`.

</domain>

<decisions>
## Implementation Decisions

### Carried forward (binding)
- **D-00:** Tests assert only what this repo owns: our file bytes, our entries and ownership. They never assert Codex's matching, loading or delivery, and never Go dependency management. Harness behaviour is proven by the orchestrator's live sessions in Herdr panes (05 D-09/D-12, 06 D-18), with negative controls, the negative-space rule, and a pass bar locked before any session. Every new guard is positive-controlled in `07-MUTATION-LOG.md`, and Go RED evidence is a `test(07-NN):` commit plus a pasted `--- FAIL:`. Exact-command hook ownership (`242ec0a`), one handler per block (06-05) and the Phase 5 capability table (one `Capabilities()` literal per target, from which `SupportsLocation`/`DescribePaths`/`Detect`/`--print-config-style` derive) all apply.
- **Nudge contract, carried verbatim from Phase 6:** `hookSpecificOutput.additionalContext` only. Exit 0 on every path, never `permissionDecision`/`decision`/`continue`, never exit 2. First matched call, then at most once a minute per (session, agent), with `main` when there is no agent id and never firing without a key. The hidden `codegraph hook pretooluse` Go subcommand runs behind a POSIX `sh` guard; the ExecPath is rendered into the guard so the registered command never carries the binary path. The harness-neutral `nudge.Qualifies` core plus an envelope adapter, and the D-15 corpora, D-16 forced-error levels and D-08 sentinel rules all apply.

### Plan order (binding)
- **D-01:** The plans run in this order:
  1. The TOML fix (D-07, D-08).
  2. FIX-03 (D-24, D-25), landing before CODEX-02 as the roadmap requires.
  3. The CODEX-01 live verification, before any `codex.go` change. The `codex.go` "no per-project config" comment is corrected in the same commit as the first `codex.go` change.
  4. The scope flip and its companions (CODEX-02/03/04).
  5. The nudge (CODEX-05).
  6. The CODEX-06 live uptake session, with the picker tmux assertion re-run after the flip.
  7. AGENT-14 docs, last.

### A. Live verification (CODEX-01, CODEX-06)
- **D-02:** A genuinely fresh Codex means a scratch `HOME=$S/home` and `CODEX_HOME=$S/home/.codex`, with `auth.json` **symlinked** (not copied) from the real `~/.codex`, and both scopes installed into the scratch HOME. The real `~/.codex/config.toml`, `~/.codex/hooks.json`, `~/.codex/AGENTS.md` and `~/.agents/skills/codegraph/SKILL.md` are sha256-recorded before and after and must be unchanged. The maintainer's real HOME already has every codegraph surface, so any positive there is confounded.
- **D-03:** Evidence tiers run model-free first:
  - `codex mcp list --json` for MCP per scope and trust gating;
  - `codex debug prompt-input` for the skill list and the `AGENTS.md` chain;
  - `codex features list`.

  Model sessions are used only for CODEX-06 uptake and CODEX-05 fires.
- **D-04:** The driver is orchestrator-scripted `codex exec --json -C <repo>` plus the session JSONL under `$CODEX_HOME/sessions`. It is backed by **one** interactive Herdr-pane TUI session for the real trust prompt and the real `/hooks` trust flow. Scripted hook runs may use `--dangerously-bypass-hook-trust` only after the TUI has proven the real trust path.
- **D-05:** Trust and pitfall 11: the local install is run **untrusted** first as a negative control, recording whether Codex warns about the skipped project layer. It is then trusted through the TUI prompt inside the scratch `CODEX_HOME`. Whether a `-c 'projects."<path>".trust_level="trusted"'` override grants trust is recorded separately.
- **D-06:** Pass bar, locked here before any session:

  | Check | Passes when |
  |---|---|
  | L1 | In a trusted local scratch repo, `codex mcp list --json` shows `codegraph` from the project `.codex/config.toml`; with a global install it shows at global scope too (CODEX-06 "both scopes") |
  | L2 | In the same repo left untrusted, the project-layer entry is NOT loaded; any warning is recorded |
  | L3 | `codex debug prompt-input` lists the codegraph skill (from the shared `.agents/skills`) and the codegraph `AGENTS.md` block |
  | L4 | An uninstalled scratch repo in the same scratch HOME shows no codegraph surface at all |
  | L5 (CODEX-06) | A fresh session, from a prompt that never names codegraph, lists the skill **and** makes at least one `mcp__codegraph__codegraph_explore` call or `codegraph explore` run |
  | L6 (CODEX-05) | With the nudge opted in and the hook trusted, the first matched Bash search call (grep/rg/find first word) produces exactly one `additionalContext` delivery; the 60 s per-key cooldown holds; an un-indexed repo gets 0 fires; no hook error or block is attributable to our hook |
  | L7 | The real-HOME checksums are unchanged |

  For L5, every grep/rg/find the agent runs is logged as negative space. Every absence claim is shown to find the thing when present.

### B. Project-local scope, TOML splice, shared AGENTS.md (CODEX-02, CODEX-04)
- **D-07:** Fix the TOML splice at its cause, RED-first, with a fixture mirroring the maintainer's indented layout (reproduced 2026-09-19 against the live `~/.codex/config.toml`: `[mcp_servers.codegraph]` at 2-space indent, with the next column-0 header 70 lines later). The rules:
  - A table ends at the next table header at ANY indentation, never inside a multi-line array or string.
  - CRLF input keeps CRLF.
  - Our own `[mcp_servers.codegraph.*]` subtables are inside our range for both splice and strip.
  - An inline `codegraph = {…}` under `[mcp_servers]` or a dotted `mcp_servers.codegraph.*` key is **refused with an error**, never answered with a duplicate key.
  - `toml_test.go` gains cases for indentation, CRLF, inline tables, subtables and multi-line arrays.
  - A planted-regression mutation (column-0-only end scan) must turn the fixture RED.
- **D-08:** Released binaries carry this data-loss bug, and **no patch release is cut now** (maintainer decision 2026-09-19, B2). The fix is Phase 7's first plan, with a WINDOWS ledger row recorded through the tool verb. Until it ships, the maintainer must not run `codegraph install`/`uninstall --target codex` or `--target all` on this machine; `STATE.md` carries that warning.
- **D-09:** The scope flip goes through the table. Codex's `Capabilities` literal gains:
  - `Scopes` {global, local};
  - per-location `MCPConfig` (`.codex/config.toml` locally);
  - per-location `Instructions` (repo-root `AGENTS.md` locally, `~/.codex/AGENTS.md` globally, unchanged);
  - `SkillDirs = sharedSkillDirs`;
  - `Hooks = codex-json`, with a `HookFiles` case for `HooksCodexJSON`.

  `Install`/`Uninstall` read paths from the table; the `loc != LocationGlobal` early returns and direct `codexConfigPath()`/`codexInstructionsPath()` calls go. `Detect`, `DescribePaths` and `--print-config-style` follow automatically. The plain goldens, the D-03 table rows, the D-13 ownership rows (codex/local) and `codex_test.go`'s local-unsupported tests change.
- **D-10:** Every local Codex install adds a `WriteResult.Notes` line saying the project must be trusted, naming the TUI trust prompt or `[projects."<root>"] trust_level = "trusted"`. codegraph never writes the trust entry itself.
- **D-11:** Repo-root `AGENTS.md` is a shared instructions file (Codex and opencode at local scope). Build the rule 05-07 planned but never built: uninstall removes the marker block only when no OTHER registered target declaring the same instructions path at that location still reports `Detect(loc).AlreadyConfigured`; otherwise it reports the file `kept`. Every uninstall order (codex→opencode, opencode→codex, `--target all`) leaves `AGENTS.md` byte-identical to its pre-install bytes once the last of them is gone, and the D-13 ownership guard covers it.
- **D-12:** A repo-root `AGENTS.override.md` shadows `AGENTS.md` for Codex; when one exists, install adds a Note. The block is still appended to `AGENTS.md`, and the user's override is never written.
- **D-13:** Fix the `install --yes`-discards-`--target` todo in this phase, RED-first as the todo prescribes, with an explicit `--target` checked before `--yes`, and uninstall checked too. The live scripts use `--target codex --yes`.

### C. Skill path (CODEX-03)
- **D-14:** Codex writes the shared `.agents/skills/codegraph` (local) and `~/.agents/skills/codegraph` (global) via `sharedSkillDirs`, and `codex` joins the manifest `targets`. These are the only documented paths, and `codex debug prompt-input` shows Codex reading `~/.agents/skills` today. There is no `.codex/skills` write: Codex does not merge same-name skills, so a second copy would appear twice.
- **D-15:** `.codex/skills` and `$CODEX_HOME/skills` are listed as read-only `SkillDirs[1:]` entries only if the live check shows Codex reading them; otherwise they are omitted.
- **D-16:** Whether project `.agents/skills` is trust-gated is settled live (untrusted vs trusted via `codex debug prompt-input`), and the D-10 Note's wording follows the result.
- **D-17:** Codex truncating the skill description in its list ("…what changing X breaks in a") is recorded as an advisory, with no SKILL.md change; the front-loaded trigger words survive.

### D. Codex nudge (CODEX-05)
- **D-18:** Codex hooks are on by default, so opt-in stays explicit: `--pretool-nudge` widens from Claude-only to Claude **and** Codex, and Codex's `hooks.json` is written only on opt-in. When the user has explicitly set `[features] hooks = false` (or the deprecated `codex_hooks = false`), the write is skipped with a Note. The D-09 scope note from Phase 6 is updated (Claude or Codex must be selected). CODEX-05 and the ROADMAP wording were amended on 2026-09-19.
- **D-19:** Codex skips a new or changed hook until it is trusted in `/hooks`, and the trust hash covers the normalized definition. Every opt-in install prints a Note saying so. The registered definition stays byte-stable across upgrades: the ExecPath is rendered into the guard, never into the command. Users are never told to use `--dangerously-bypass-hook-trust`.
- **D-20:** The command form is **quoted from day one**, so AR-06-08 is not inherited:
  - local: `"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"` (the docs' repo-local form);
  - global: a quoted absolute path to `$CODEX_HOME/hooks/codegraph-pretooluse.sh`.

  One group, `matcher: "^Bash$"`, one handler, an explicit short `timeout`, at both scopes.
- **D-21:** The adapter is a Codex envelope on the same hidden subcommand (e.g. `codegraph hook pretooluse --harness codex`). It:
  - maps `tool_name` `Bash`, and defensively `exec_command`/`shell`, to the shell rule;
  - accepts `tool_input.command` as a string or an argv array (the last element when an argv, and any doubt means silent);
  - keys on stdin `session_id` (the parent's for subagents) plus `agent_id`, else `main`;
  - emits the same output object;
  - reuses only the shell rows of the D-15 corpora (Codex has no Grep/Glob/Read tools).
- **D-22:** Codex has no `CLAUDE_PROJECT_DIR`, so the indexed check works differently:
  - the local guard derives the repo root from its own path (two directories up from `.codex/hooks/`), with no process spawn;
  - the global guard checks `$PWD` (the hook's cwd is the session cwd);
  - the Go core re-checks stdin `cwd`.

  With no `if` pre-filter every Bash call runs the guard; 06 measured about 3 ms un-indexed and 12 ms indexed.
- **D-23:** The opt-in's stickiness evidence is our exact-identity group in `hooks.json`, since Codex's skill manifest is shared with cursor and opencode. The group is appended LAST (as `writeHookEntry` already does), so the position-keyed trust entries of the maintainer's other hooks are preserved. The live check records whether removing our group re-flags later foreign hooks for review.

### E. FIX-03 and AGENT-14
- **D-24:** Fix FIX-03 at its cause. `checkboxDelegate.Render` writes a trailing newline while `Height()` declares 1, and bubbles v2's `populatedView` inserts its own separator, so each row costs 2 lines (overflow N−1 = 7 at 8 targets, matching the recorded 35 vs 28). Drop the trailing newline, and fix the identical `daemonDelegate` defect. This is a code-reading hypothesis until a failing test proves it; the footer size and pagination are never patched to mask it.
- **D-25:** The guard is a model-level test (`lipgloss.Height(View()) <= 30` at 100×30 with all 8 target names present), plus the tmux TTY-05 assertion re-anchored on the footer text (`space: toggle`) instead of the title workaround. Both are shown RED on the current code and re-run after CODEX-02.
- **D-26:** The picker keeps all 8 rows (`AgentTargets`/`AllTargets()`); CODEX-02 changes only Codex's local pre-check. The ROADMAP premise that the scope flip changes the target count was corrected on 2026-09-19.
- **D-27:** The capability table lives in a new `docs/AGENT-CAPABILITIES.md`, linked from the README's agent section. A Go drift test, mirroring `TestMatrix_DocMirrorsDescriptor`, requires the code-derived columns (scopes, MCP config, format, instructions, skill, hooks, nudge) to equal `Capabilities()` for 8 targets × 2 scopes; a planted mutation proves it goes RED. A hand-kept verification column carries `verified <date> (<evidence file>)` or `[ASSUMED] (<doc URL>, fetched <date>)`. Cursor, Gemini CLI and Kiro are `[ASSUMED]`.
- **D-28:** The nudge column is derived in the drift test from `Hooks` (claude-json = SessionStart plus opt-in PreToolUse; codex-json = opt-in PreToolUse). There is no new `Capabilities` field and no new `--print-config-style` field.
- **D-29:** The MCP `instructions` skill sentence becomes harness-neutral and true for the 7 skill-receiving targets (all but Hermes). The whole const stays ≤600 bytes, with the skill sentence inside the first 512 bytes (Codex's guidance). The WIRE-03 guard is updated, and the 38 wire transcripts are re-frozen in one reviewed diff.
- **D-30:** The "4 of 8" comments are updated as comments only (`instructions.go:13-28`, `shared.go:743`, `codex.go:13-19`). The installed marker-block text is unchanged, so no user's block is rewritten.

### Claude's Discretion
- The Codex guard filename, the adapter flag spelling, the timeout value, and the exact Note wording within D-10/D-12/D-18/D-19.
- How the drift test parses `docs/AGENT-CAPABILITIES.md` (a table-row format of the planner's choosing).
- Test fixture shapes for D-07.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Capability plumbing:**
  - `Capabilities` struct: `capabilities.go:74-95`.
  - `globalOnlyPath`: `:190`.
  - `installDeclaredSkill`/`uninstallDeclaredSkill`: `:238`/`:260`.
  - `describeDeclaredPaths`: `:285`. It panics on an undeclared hook mechanism; the `HookFiles` `HooksCodexJSON` case is at `:175`.
  - `sharedSkillDirs`: `skillshared.go:49-69`.
- **Instructions:**
  - `upsertInstructionsEntry`: `shared.go:747`. `removeMarkedSection`: `shared.go:695`.
  - Markers: `instructions.go:8-11`.
  - opencode already writes the repo-root `AGENTS.md` locally (`opencode.go:97-101`).
- **Hooks JSON:**
  - `writeHookEntry` (`shared.go:248`) appends owned groups after foreign ones, and `removeHookEntry` is at `:388`. Both use `blockOwnsAnyCommand`/`commandIsOwned` (06 WR-02).
  - `writeJSONFile` re-marshals with sorted keys (`:84`).
- **Nudge:**
  - `nudge.Qualifies` (`classify.go:53`), `CooldownWindow` (`cooldown.go:18`), `SessionKey` (`:35`), `Gate` (`:54`), `nudge.Text` (`text.go:8`).
  - The Claude adapter is `internal/cli/hook_pretooluse.go:26-146`.
  - The guard template is `.claude/hooks/pretooluse-nudge.sh`, with the render token at `claude_pretooluse.go:23` and the renderer at `:135`.
- **Notes channel:** `WriteResult.Notes` (`types.go:96-99`; Kiro precedent).
- **Doc-drift precedent:** `docs/LANGUAGE-CAPABILITY-MATRIX.md` plus `TestMatrix_DocMirrorsDescriptor` (`internal/indexer/capability/matrix_test.go:113`).

### Established Patterns
- Plain goldens (`internal/cli/testdata/plain/print-config-style{,-local}.golden`) are regenerated with `-update-plain-goldens`. Codex currently reads `scopes=global … skill=none hooks=none` and `(local not supported)`.
- `docs/CLI-REFERENCE.md` is regenerated only via `task docs:cli` under the drift gate.
- The tmux harness isolates with `env HOME=<tmp>` (`test/tmux/install_cancel_test.go:33`).

### Integration Points
- **`codex.go`:**
  - `:13-19` holds the comment to correct.
  - `:37-45` is the Capabilities literal.
  - `:104` and `:139` are the global-only early returns.
  - `:108` and `:127` are direct path calls.
  - `Detect` (`:80-100`) and `DescribePaths` (`:169-172`) already derive from the table.
- **`toml.go`:** `findTOMLTableRange` at `:80-101`, whose column-0 end scan is the bug.
- **Tests that gain codex/local rows:** `ownership_test.go:585`, `codex_test.go:16,26`, `agentpicker_test.go`.
- **`install.go`:** `--pretool-nudge` help `:69-80`, the D-09 note, and the `--yes` ordering `:111-118`.
- **`upgrade.go:79-86`** refreshes Claude only.
- **MCP `instructions` const:** `internal/mcp/server.go:57` (554 bytes; the skill sentence starts at byte 490).
- **FIX-03:**
  - `agentpicker.go:34-35` (`Height()=1`, `Spacing()=0`), `:64` (the trailing newline), `:195` (`AllTargets()`).
  - `daemonpicker.go:59`.
  - bubbles v2.1.1 `list.go:1223`.

### Current Codex reference (verified 2026-09-19; developers.openai.com/codex/* and learn.chatgpt.com/docs/*, Context7 `/openai/codex`; local codex-cli 0.155.0)
- **Project config:** `.codex/config.toml` loads only for trusted projects (closest file wins). An untrusted project skips all `.codex/` layers: config, hooks and rules. Trust lives in `projects.<path>.trust_level` in the user config, and the project root is the `.git` directory.
- **MCP:** project-scoped servers work for trusted projects only. `codex mcp list --json` shows the merged set, with no scope column.
- **Skills:** `$CWD/.agents/skills` up to the repo root, `$HOME/.agents/skills`, `/etc/codex/skills` and system skills. Symlinks are followed. Same-name skills are NOT merged. Frontmatter needs `name` and `description`. Source (not docs) also reads `$CODEX_HOME/skills` and `.codex/skills`.
- **AGENTS.md:** global `~/.codex/AGENTS.override.md` else `AGENTS.md`. In the project, one file per directory from the git root to cwd, where an override shadows `AGENTS.md`. There is a 32 KiB combined cap.
- **Hooks:**
  - **Stable and on by default** (`[features] hooks`; `codex_hooks` is a deprecated alias).
  - Non-managed hooks must be reviewed and trusted, by hash, in `/hooks`.
  - Locations: `~/.codex/hooks.json`, `~/.codex/config.toml [hooks]`, `<repo>/.codex/hooks.json`, `<repo>/.codex/config.toml`. Project hooks need trust.
  - `matcher` is a regex on `tool_name`, and shell runs as `Bash`. There is no per-handler `if`. Default timeout is 600 s. The cwd is the session cwd.
  - PreToolUse stdin carries `session_id` (the parent's in subagents), `turn_id`, `agent_id`/`agent_type`, `cwd`, `tool_name`, `tool_use_id` and `tool_input.command`.
  - Output `hookSpecificOutput.additionalContext` adds context without blocking; plain stdout is ignored.
- **Transcripts:** under `$CODEX_HOME/sessions/YYYY/MM/DD/`.
- **Local probes:**
  - `codex features list` → `hooks stable true`.
  - `codex debug prompt-input` is model-free and renders `<skills_instructions>` and the AGENTS.md chain.
  - `codex exec` has `--json`, `-C`, `--skip-git-repo-check`, `--ephemeral`, `--ignore-user-config` and `--dangerously-bypass-hook-trust`.
- **Not confirmed, settled live or recorded `[ASSUMED]`:**
  - whether exec loads project config when trust is unset, and whether a warning appears;
  - whether a `-c` trust override grants trust;
  - whether project `.agents/skills` and repo `AGENTS.md` are trust-gated;
  - whether `.codex/skills` and `$CODEX_HOME/skills` are read;
  - the type of `tool_input.command`;
  - how position-keyed `hooks.state` entries behave when groups shift;
  - whether `instructions` beyond 512 chars is truncated;
  - whether PreToolUse `additionalContext` is actually delivered.

</code_context>

<specifics>
## Specific Ideas

- **Blocking finding (orchestrator-confirmed 2026-09-19):** the maintainer's `~/.codex/config.toml` has `[mcp_servers.codegraph]` at line 125, 2-space indented, and the next column-0 header is `[memories]` at line 195. A released-binary `codegraph install` or `uninstall --target codex` would delete every MCP server in between (computer-use, context7, deepwiki, engram, exa, fal, firecrawl and the rest). The maintainer accepted the fix-first path, with no patch release now.
- The maintainer accepted every recommended answer ("go ahead"), including widening `--pretool-nudge` to Codex and quoting the Codex hook command from day one.

</specifics>

<deferred>
## Deferred Ideas

- Exec-form or quoting for Claude's existing SessionStart and PreToolUse commands (AR-06-08). It changes the owned identity and would duplicate entries on upgrade.
- A Codex SessionStart nudge (hooks.json supports it; the Phase 7 goal lists only PreToolUse).
- Codex skill metadata `agents/openai.yaml` (`dependencies.tools` type `mcp`, `allow_implicit_invocation`) and Codex plugin packaging (v2).
- `codegraph upgrade` refreshing non-Claude skill packages and the Codex guard (05-07 advisory 2), unless D-23's evidence needs it.
- Uninstall leaving an empty parent skills directory (carried todo).
- Kiro reading the codegraph block twice through `AGENTS.md` written by opencode or Codex (D-06(d) advisory).
- Live verification for Cursor, Gemini CLI and Kiro (they stay `[ASSUMED]`).
- Filtering picker rows per location.
- A patch release carrying the TOML fix ahead of the milestone (declined for now, B2).

</deferred>
