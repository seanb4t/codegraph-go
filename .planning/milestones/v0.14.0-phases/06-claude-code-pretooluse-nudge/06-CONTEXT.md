# Phase 6: Claude Code PreToolUse Nudge - Context

**Gathered:** 2026-09-19
**Status:** Ready for planning

<domain>
## Phase Boundary

In a `.codegraph/`-indexed repo, a Claude Code `PreToolUse` hook adds one factual line of context pointing at `codegraph_explore` when the agent reaches for a search tool (Bash `grep`/`rg`/`find` as the first command, `Grep`, `Glob`, or a code-file `Read`). It fires on the first matched call and then at most once a minute, per session and separately per subagent. It never denies, blocks or asks. It always exits 0. In an un-indexed repo it is silent, at the cost of one directory check. `codegraph install` registers it as an opt-in, and `uninstall` removes it, both through the existing exact-identity hook writer, alongside the default SessionStart nudge (NUDGE-03…06).

Out of scope:
- the Codex nudge (Phase 7, CODEX-05), which reuses this contract verbatim;
- the published per-harness capability table (Phase 7, AGENT-14);
- nudges for other harnesses (v2, NUDGE-07…10).

</domain>

<decisions>
## Implementation Decisions

### Carried forward (binding)
- **D-00 (from Phase 4/5):** tests assert only what this repo owns: the guard's and the subcommand's bytes and behaviour, the settings entries we write, and ownership. They never assert Claude Code's matcher or delivery behaviour, and never Go dependency management. That Claude Code actually matches a call and delivers the context is shown by the live session (D-18), not by a unit test. Every new guard is positive-controlled in `06-MUTATION-LOG.md`. Go RED evidence is the `test(06-NN):` commit plus a pasted `--- FAIL:` transcript.
- **Contract (2026-09-14 reframe; carried verbatim into CODEX-05):** the hook emits only `hookSpecificOutput.additionalContext`. It exits 0 on every path, never emits `permissionDecision`, `decision` or `continue`, and never exits 2. State the contract harness-neutrally, so Phase 7 reuses it rather than re-deriving it.
- **Ownership:** hooks are owned by exact command string, never by matcher (`242ec0a`). A hand-edited own entry is duplicated, not overwritten. An unrelated `PreToolUse` entry under the same event stays byte-identical.

### A. Hook form & trigger heuristic
- **D-01:** Maintainer decision, option 1: a Go subcommand plus a sh guard, chosen over Python/uv and pure sh. The logic lives in a hidden Go subcommand, `codegraph hook pretooluse`, reached through a tiny embedded POSIX `sh` guard.
  - **Guard:** dogfooded under `.claude/hooks/`, embedded via `claudeassets` beside `session-nudge.sh` (v0.10.0 D-03/D-04). It is the registered hook command, so it carries the owned identity.
  - **Guard job:** (a) do the D-04 directory check, exiting 0 at once when the repo is un-indexed (no binary started); (b) exit 0 silently when the codegraph binary is missing or not executable (never a "hook error" notice); (c) run the binary with stdin passed through, then `exit 0` whatever the binary did, so even a crash cannot produce a non-zero exit.
  - **Why Go:** real JSON parsing, portable time handling for D-05, direct Go unit tests, no new runtime for users (the project's core value), and a harness-neutral core that CODEX-05 reuses in Phase 7.
  - **Rejected alternatives:**
    - Python via `uv`: users may lack uv; per-call interpreter start; possible Python downloads; the macOS `python3` stub dialog.
    - Pure `sh`: fragile JSON handling without `jq`; the BSD/GNU `find -mmin` difference.
- **D-01a:** The subcommand is hidden. It is registered with one bare allowlist line in `internal/cli/testdata/cli-reference-allowlist.txt`, following the `codegraph man` precedent, so `docs/CLI-REFERENCE.md` does not change for it.
  - It must not open the index, the store or the daemon. It never returns an error to cobra, so `cmd/codegraph/main.go`'s error path (print, then exit 1) is never taken.
  - It recovers panics, and writes nothing to stdout except the one pinned JSON object when it fires.
  - The core is a pure harness-neutral function (input facts in, fire-or-silent out). The Claude stdin/stdout envelope is a thin adapter around it.
- **D-01b:** The binary's location must not be part of the registered command's identity, because an `ExecPath` change would otherwise make the old entry look foreign and create duplicates. The recommended approach is to render the absolute `ExecPath` into the guard script from an embedded template at install time. The settings entry then stays the stable script path (mirroring SessionStart), and install and upgrade refresh the script as own content (v0.10.0 Phase 7 D-05, D-10 below). Leaning on `PATH` alone is rejected, because hook processes do not reliably inherit an interactive `PATH`.
- **D-02:** Bash detection has two layers. First, the handlers' `if` rules pre-filter, one Bash handler each for `Bash(grep *)`, `Bash(rg *)` and `Bash(find *)`, so ordinary Bash calls start no process. Second, the Go core confirms that the command's first word, after stripping leading `VAR=val` assignments, is `grep`, `egrep`, `fgrep`, `rg` or `find`. This matters because `if` also matches pipe tails such as `git log | grep`. On any parse doubt the hook stays silent.
- **D-03:** `Grep` and `Glob` always qualify. `Read` qualifies unless `file_path` is an obvious non-code file (`*.md`, `*.json`, `*.yaml`/`*.yml`, `*.toml`, `*.txt`, `*.lock`).
- **D-04:** The repo counts as indexed when the guard's `[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]` holds, the exact NUDGE-02 check. It runs before any process is started or stdin is read.

### B. Cooldown & scope (maintainer decisions 2026-09-19, replacing "once per session")
- **D-05:** Maintainer decision, option C at once per minute. Fire on the first matched call, then at most once every 60 seconds per key. The 60 s value is one named Go constant. NUDGE-04 and ROADMAP criterion 2, the goal, and the CODEX-05 note are amended to match. Expected ceiling: about 60 fires, roughly 3.6k tokens, per hour of continuous searching.
- **D-06:** Maintainer decision, from the question "why not sub agents?". Subagents are included, each with its own cooldown. The key is (session, agent): `session_id` plus `agent_id`, or the literal `main` when `agent_id` is absent. A subagent starts with a fresh context and did not see the main thread's nudge.
- **D-07:** The session id is `$CLAUDE_CODE_SESSION_ID`, falling back to `session_id` parsed from stdin. With neither available, stay silent and never fire unkeyed. `/clear` produces a new id, so the next matched call fires at once. `--resume` keeps the id, so the cooldown continues.
- **D-08:** The sentinel is per key, under `os.TempDir()` (which honours `TMPDIR`), in `codegraph-nudge-<uid>/`:
  - The directory is created mode 0700. Use `Lstat`, and stay silent if the directory exists but is a symlink or is not owned by the current uid.
  - Recording a fire (the file's mtime, or a timestamp) must never write through a symlink. Refuse a symlinked sentinel, and never truncate someone else's file.
  - The age is `time.Since(modTime)` against the D-05 constant. The clock is injectable for tests (D-17).
  - If two parallel hooks race, a rare double fire is accepted and harmless.
  - Any failure (unwritable directory, stat error, etc.) means silent, exit 0.

### C. Opt-in & lifecycle
- **D-09:** Opt-in is a Claude-only bool flag on `install`, shaped like `--auto-allow` (working name `--pretool-nudge`). It goes through the CLI-reference drift gate and flag accounting. There is no picker row. Print a `note:` when the flag is given but Claude is not among the resolved targets.
- **D-10:** The opt-in is sticky. It is recorded in the manifest as new Files keys for the guard script and `settings.json#hooks.PreToolUse`. `install` and `upgrade` refresh it while the manifest records it. Only an explicit `--pretool-nudge=false` (detected with cobra `Changed`) or `uninstall` removes it. `upgrade`'s refresh (`upgrade.go:48-73`) must carry the recorded opt-in; today it passes only `AutoAllow:false`.
- **D-11:** `uninstall` always attempts `removeHookEntry("PreToolUse", own)` and the guard-script removal, reporting `not-found` when the user never opted in. The ownership table (`TestOwnershipExactIdentity`) runs the Claude leaves with the opt-in on and plants an unrelated `PreToolUse` block under the same matcher, which must survive byte-identical.
- **D-12:** Registration mirrors SessionStart:
  - shell form, `${CLAUDE_PROJECT_DIR}/.claude/hooks/<guard>` for local and the absolute guard path for global (`claudeHookCommand`); the command never contains the binary path (D-01b);
  - one handler per matcher/`if` rule;
  - an explicit short `"timeout"` (the default is 600 s, and a PreToolUse hook delays the tool call);
  - no `statusMessage`.
- **D-13:** Reporting and dogfooding:
  - `HookFiles` for `claude-json` names the new guard script.
  - `--print-config-style` keeps `hooks=claude-json`; the golden is unchanged, and the opt-in detail belongs to Phase 7's AGENT-14 table.
  - This repo's `.claude/settings.json` registers the hook.
  - `TestHookRegistrationMatchesFragmentAndScript` extends to PreToolUse.

### D. Text & validation
- **D-14:** The nudge text is a new factual one-liner, as the hooks docs advise, since imperative "system-command" wording can trip prompt-injection defences. It names only `codegraph_explore` and the `codegraph explore` CLI fallback, and is a byte-exact pinned Go constant. The existing nudge drift guards (`TestNudgeTextNamesOnlyRealTools`, `TestNudgeTextCarriesNoUnpinnedFacts`) extend to cover it. No new env token is introduced outside the drift guard's allowlist.
- **D-15:** The corpora are harness-neutral `testdata` rows `{tool, input, want}`:
  - true positives: where-is-X patterns;
  - false positives: legitimate grep use (literals, logs, config, non-code reads).

  A Go table test drives the pure core with every row. CODEX-05 reuses the rows through its own envelope adapter.
- **D-16:** Forced-error coverage has two levels, both asserting exit 0, stdout either empty or exactly the pinned JSON, and no `permissionDecision`/`decision`/`continue` key.
  - **Subcommand level (Go):**
    - no stdin, malformed JSON, oversized input;
    - `CLAUDE_CODE_SESSION_ID` and `session_id` both absent;
    - an unwritable or foreign-owned sentinel directory, a symlinked sentinel;
    - a sentinel inside and outside the cooldown;
    - parallel runs;
    - a forced panic (recovered).
  - **Guard level:** an exec test of the real embedded guard covering:
    - `.codegraph` missing, and `.codegraph` existing as a file;
    - the binary missing or not executable;
    - the binary exiting non-zero or crashing (the guard still exits 0);
    - `CLAUDE_PROJECT_DIR` unset.

  Both levels are positive-controlled by planted mutations: an `exit 2` in the guard, a returned error that reaches cobra's exit-1 path, and an emitted `permissionDecision` (06-MUTATION-LOG.md).
- **D-17:** The cooldown is tested with an injected clock and planted sentinel mtimes (`os.Chtimes`), never `sleep` in the suite. The test pins "fires when outside the cooldown, silent inside it" per key, including separate keys for `main` and a subagent.
- **D-18 (live check; pass bar locked here, before any session runs):**
  - **Setup:** the orchestrator runs Herdr-driven fresh `claude --debug-file <path>` sessions in a scratch indexed repo with a local install including the opt-in, plus a negative control in an un-indexed repo.
  - **PASS requires all of:**
    1. In a fresh main-thread session, the first matched call produces exactly one fire, visible as the added context in the transcript or debug log.
    2. No two fires share a key within 60 s. Every fire is at least 60 s after the previous one for the same key.
    3. After a gap of at least 60 s, the next matched call fires again.
    4. A subagent's first matched call fires once for that subagent, whatever the main thread's cooldown.
    5. After `/clear`, the next matched call fires.
    6. The un-indexed control has 0 fires.
    7. There are 0 "hook error" notices, and no permission prompt, deny or block attributable to the hook.
  - **Recorded, not gated:** matched-call count and fires per session (the NUDGE-05 fire rate), the share of fires that land on true-positive calls, whether the agent then uses `codegraph_explore` (uptake), and the hook's wall time per run from the debug log (the Go binary's start cost on an indexed repo).
  - Claude Code's matcher and delivery are never unit-tested (D-00). The five points the docs do not confirm are settled here:
    - `additionalContext` with no permission decision;
    - whether a subagent carries the parent's `session_id`;
    - the first version exporting `CLAUDE_CODE_SESSION_ID`;
    - dedup of same-command handlers with different `if` rules;
    - the stability of stdin key order.

### Claude's Discretion
- The guard script name, the exact flag name (working: `--pretool-nudge`), the subcommand's package placement, the timeout value, and the nudge wording within D-14.
- The sentinel mechanics within D-08 (mtime vs content timestamp; `O_NOFOLLOW` vs an `Lstat` check before writing).
- How the binary path reaches the guard, within D-01b (templated guard recommended).

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `claudeassets.go:33-36,50-58`: embeds `session-nudge.sh`. The new script needs its own `//go:embed` line and accessor, and must stay at the repo root under `.claude/hooks/`.
- `.claude/hooks/session-nudge.sh:12-15`: the template (one `[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]`, `printf`, `exit 0`; stateless, reads no stdin).
- `internal/agents/shared.go`:
  - `writeHookEntry` :202-269, generic over `event`, with `isOwned` exact-command at :214-236;
  - `removeHookEntry` :366-462;
  - `writeEmbeddedFile` :301-341, which self-heals the executable bit;
  - `readJSONFileStrict` :156.
- `internal/agents/claude.go`:
  - `claudeFragmentCommand` :151
  - `claudeHooksScriptPath` :193
  - `claudeHookCommand` :253-258
  - `claudeSessionStartBlocks` :269-322 (mirror as `claudePreToolUseBlocks`)
- `internal/agents/manifest.go:35-37`: the manifest keys `hooks/session-nudge.sh` and `settings.json#hooks.SessionStart`; `hashOwnedHookBlocks` :147.
- `internal/agents/hookpackage_test.go`:
  - `runSessionNudge` exec harness :64-110
  - pinned `nudgeLine` :43
  - `TestSessionNudgeOutputIsPinnedAndStateless` :220
  - `TestHookRegistrationMatchesFragmentAndScript` :371
- `internal/mcp/skill_claims_drift_test.go:113,807,839`: the nudge-text guards. The only allowed env token is `CODEGRAPH_MCP_TOOLS`.
- `claude_skillpackage_test.go:213-265,503-693,1037-1087`: existing fixtures for survival of an unrelated `PreToolUse` block.

### Established Patterns
- Claude `Install` order:
  1. MCP entry
  2. instructions
  3. the AutoAllow-gated permission (:466-473)
  4. skill
  5. executable script (:525-539)
  6. `writeHookEntry` SessionStart (:548)
  7. the manifest, only if every write succeeded (:569-580)
- `Uninstall`: `uninstallSkillPackage` with Claude-exclusive keys (:637) → script removal → `removeHookEntry` SessionStart (:654).
- Opt-ins are bare bool flags (`--auto-allow`, install.go:133). `InstallOptions` has `AutoAllow` and `ExecPath` (types.go:112-122).
- The ownership table (ownership_test.go:535, 32 leaves) uses `InstallOptions{ExecPath}` only (:472). `reproduce242ec0aPrecondition` (:277-288) plants a foreign SessionStart `startup` block. `assertOwnEntriesGoneAfterUninstall` (:384-455) checks SessionStart only and must extend to PreToolUse.

### Integration Points
- `capabilities.go:156-173`: `HookFiles` for `claude-json` returns `[settings.json, session-nudge.sh]`, and the new script must be added. `describeDeclaredPaths` (:278-326) panics on an undeclared mechanism (WR-01).
- `printconfigstyle.go:67-68`: `hooks=<mechanism>`, golden-frozen and unchanged by D-13.
- `upgrade.go:48-73`: `refreshInstalledSkills` must carry the recorded opt-in (D-10), and re-render the guard with the current `ExecPath` (D-01b).
- `internal/cli/root.go:124`: `AddCommand`, where the hidden `hook` command registers. `internal/cli/man.go:50` and `renamed.go:63,78` are the hidden-command precedents. `internal/cli/testdata/cli-reference-allowlist.txt` takes one bare entry for the hidden command (`cli_reference_test.go:130` treats flags on a hidden command as covered by it).
- `docs/CLI-REFERENCE.md`: the new flag goes through `task docs:cli` and `TestEveryRegisteredFlagIsAccountedFor`.
- `.claude/settings.json` (dogfooded): SessionStart only today, gaining the PreToolUse registration.

### Current hooks reference (verified 2026-09-19; code.claude.com/docs/en/hooks, plus env-vars, permissions and CHANGELOG; local `claude` 2.1.277)
- **Matcher:** `Grep|Glob|Read` and `Bash` are exact names. A matcher containing any other character is an unanchored JS regex. MCP tools are named `mcp__<server>__<tool>`.
- **The `if` field** (v2.1.85+, compound fix v2.1.89) takes one permission rule per handler. For Bash it checks each subcommand, pipe parts included. The docs call it best-effort.
- **Stdin:** `session_id`, `transcript_path`, `cwd`, `permission_mode`, `hook_event_name`, plus `agent_id`/`agent_type` inside a subagent. PreToolUse adds `tool_name`, `tool_input` and `tool_use_id`:
  - Bash: `{command, …}`
  - Grep: `{pattern, path, glob, …}`
  - Glob: `{pattern, path}`
  - Read: `{file_path (absolute), …}`
- **Env:** `CLAUDE_PROJECT_DIR` is the session-start root. `CLAUDE_CODE_SESSION_ID` equals `session_id` and is updated on `/clear`.
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"…"}}`, delivered as a system reminder beside the tool result, with a 10,000-character cap. Plain stdout from a PreToolUse hook goes to the debug log only, so the hook must print JSON and nothing else.
- **Exit codes:** JSON is parsed on every exit code. Exit 2 blocks. A non-zero exit with empty or plain stdout shows a "hook error" notice.
- **Timeout:** the default is 600 s, and a timed-out command hook does not block.
- **Running:** matching hooks run in parallel. `once: true` works only in skill frontmatter.

</code_context>

<specifics>
## Specific Ideas

- The maintainer chose a **1-minute cooldown** over once-per-session. The single-shot nudge was fragile: if it was ignored or compacted away, it was gone for the rest of the session. The maintainer also asked to **include subagents**, since they do the where-is-X grepping. The 60 s value lives in one named constant, so tuning it later is a one-line change.
- The maintainer asked whether the hook could be Python via `uv` instead of shell. After weighing the costs (users may lack uv; per-call interpreter start; possible Python downloads; the macOS `python3` stub dialog; the "no bundled runtime" core value), the maintainer chose **option 1: the logic in a hidden Go subcommand behind a tiny `sh` guard**.

</specifics>

<deferred>
## Deferred Ideas

- Exec-form (`args`) or quoting for the *existing* SessionStart entry. Its unquoted absolute global path breaks on paths with spaces, but changing it changes the owned identity, which would create duplicates on upgrade. It needs its own decision.
- `once: true` via skill-frontmatter hooks (declined: it puts a Claude-only key into the portable SKILL.md).
- A SessionEnd sentinel-cleanup hook (the OS reaps `$TMPDIR`; the cooldown makes it unnecessary).
- A `nudge` field or column in the capability table (Phase 7, AGENT-14).
- The Codex nudge (CODEX-05, Phase 7) and other harness nudges (v2), both reusing the D-15 corpus.
- Pre-existing: `install --yes` discards an explicit `--target` (open todo). It also affects whether the new flag's Claude selection is honoured.

</deferred>
