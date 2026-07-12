---
phase: 06-agent-integrations-cli-lifecycle
plan: 04
subsystem: cli
tags: [cli, cobra, installer, agent-integration, mcp, tdd]

# Dependency graph
requires:
  - phase: 06-agent-integrations-cli-lifecycle
    provides: "internal/agents registry + AgentTarget interface (06-01) and all 8 per-agent target implementations (06-02, 06-03)"
provides:
  - "codegraph install: --target auto|all|none|<csv> / --location global|local, os.Executable() exec-path resolution (D-04), TTY multi-select w/ non-interactive auto fallback (D-03), per-agent status reporting"
  - "codegraph uninstall: mirrors install's flag/resolve shape, removed/not-configured/unsupported per-agent status (D-08), defaults to \"all\" (not a prompt) with no --target"
  - "Both commands registered in internal/cli/root.go"
affects: ["06-VALIDATION.md Manual-Only Verifications", "phase 8 release docs (install/uninstall are now the user-facing drop-in-swap entrypoints)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cobra thin-command delegation extended to install/uninstall: resolve flags -> resolve os.Executable() -> iterate internal/agents registry -> print via cmd.OutOrStdout() (no agent-specific branching in internal/cli)"
    - "Injectable-seam TTY detection (installStdinIsInteractive package var) mirrors upgrade.go's upgradeRunFunc pattern so the no-TTY fallback is exercised by execCmd's strings.Reader stdin without a real pty"
    - "printAgentResults shared by install and uninstall: skips (reports \"unsupported\") a target that fails SupportsLocation(loc) before ever calling Install/Uninstall, since those return an empty WriteResult for an unsupported location that would otherwise print as a confusing no-op"

key-files:
  created:
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - internal/cli/install_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "uninstall defaults --target to \"all\" (not \"auto\") when omitted, with no interactive prompt at all — a destructive-by-default reversal is safe to attempt across the whole roster since Uninstall never errors on an unconfigured agent (D-08), which also trivially satisfies the no-TTY/CI non-blocking requirement without needing its own TTY-detection branch"
  - "printAgentResults checks AgentTarget.SupportsLocation(loc) in the CLI layer before calling Install/Uninstall, rather than relying on each target's own empty-WriteResult-on-unsupported-location behavior, so the command can print an explicit \"unsupported\" status line instead of a silent \"(no files touched)\""
  - "installStdinIsInteractive is a package-level var (not a direct os.Stdin.Stat() call inline) so install_test.go's execCmd-driven tests exercise the real non-interactive code path without needing a pty — mirrors upgrade.go's upgradeRunFunc injectable-seam convention from 06-06"
  - "TTY detection uses stdlib only (cmd.InOrStdin() == os.Stdin AND os.ModeCharDevice), per RESEARCH.md's explicit no-new-terminal-dependency guidance — golang.org/x/term is already an indirect dependency via sigstore-go but was deliberately not promoted to direct for this"

patterns-established:
  - "Shared printAgentResults(cmd, targets, loc, do, statusOf) reporting loop parameterized by a WriteResult-producing closure and a status-rollup function — reused verbatim by both install (installStatus: unchanged/configured) and uninstall (uninstallStatus: removed/not-configured), avoiding duplicated per-agent-loop/per-file-line printing logic"

requirements-completed: [AGNT-01, AGNT-02, AGNT-03]

coverage:
  - id: D1
    description: "codegraph install resolves --target auto|all|none|<csv> at --location global|local, resolves os.Executable() once, and reports per-agent status idempotently"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_TargetAll_WritesAndReportsPerAgent"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_TargetCSV_SelectsExactlyThose"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_TargetNone_InstallsNothing"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_ExecPathAppearsInWrittenConfig"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_Idempotent_RerunReportsUnchanged"
        status: pass
    human_judgment: false
  - id: D2
    description: "An unknown --target csv id is a clear error and writes nothing (no partial write, T-06-04-01)"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_TargetCSV_UnknownID_ErrorsNoWrite"
        status: pass
    human_judgment: false
  - id: D3
    description: "No --target and no TTY resolves straight to auto without blocking on stdin (D-03, T-06-04-02); auto falls back to Claude when zero agents are detected"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_NoTargetNonTTY_ResolvesAutoWithoutBlocking"
        status: pass
    human_judgment: false
  - id: D4
    description: "--auto-allow toggles Claude Code's permissions.allow list (D-05)"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_AutoAllow_TogglesPermission"
        status: pass
    human_judgment: false
  - id: D5
    description: "codegraph uninstall reverses install, reporting removed/not-configured/unsupported per agent, never erroring on an agent never installed (D-08)"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_ReportsRemovedAndNotConfigured"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_ReportsUnsupportedForWrongLocation"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_NeverInstalledAgent_NoError"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_NoTargetDefaultsToAllWithoutPrompting"
        status: pass
    human_judgment: false
  - id: D6
    description: "A command-level install->uninstall round trip preserves an unrelated sibling MCP entry and restores pre-install state modulo the CodeGraph section (D-01, D-07)"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstallUninstallRoundTrip_PreservesSiblingEntry"
        status: pass
      - kind: integration
        ref: "internal/cli/install_test.go#TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState"
        status: pass
    human_judgment: false
  - id: D7
    description: "A live agent (Claude Code / Cursor / Codex / opencode / Gemini / Antigravity / Hermes / Kiro) actually lists codegraph_explore after a real `codegraph install` and drops it after `codegraph uninstall`, with unrelated config preserved"
    verification: []
    human_judgment: true
    rationale: "No live agent runtime is available in this autonomous execution environment to perform a real MCP handshake. Substituted with TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState (a temp-$HOME install->uninstall round trip asserting the written Claude Code MCP config and CLAUDE.md instructions block shape, and that uninstall restores pre-install state) as automated evidence. The residual live-agent handshake is deferred as a manual follow-up per 06-VALIDATION.md's Manual-Only Verifications — see 'User Setup Required' below."

# Metrics
duration: 25min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 04: install/uninstall CLI Commands Summary

**`codegraph install`/`codegraph uninstall` Cobra commands wired thinly to the internal/agents registry — os.Executable() exec-path resolution, TTY multi-select with non-interactive auto fallback, and removed/not-configured/unsupported status reporting, with no agent-specific logic in internal/cli.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-07-12T20:32:00Z
- **Completed:** 2026-07-12T20:57:15Z
- **Tasks:** 3 (2 autonomous TDD tasks + 1 checkpoint, auto-handled per pipeline mode)
- **Files modified:** 4 (install.go, uninstall.go, install_test.go, root.go)

## Accomplishments
- `codegraph install` — thin Cobra command resolving `--target auto|all|none|<csv>` and `--location global|local`, resolving `os.Executable()` once and threading it through every target's `InstallOptions.ExecPath` (D-04), with a TTY-only interactive multi-select (prefilled with `agents.DetectAll`) that degrades to non-interactive `auto` under no TTY/CI (D-03)
- `codegraph uninstall` — mirrors install's shape, reports `removed`/`not-configured`/`unsupported` per agent (D-08), defaults `--target` to `all` (not a prompt) so a destructive reversal command never blocks
- Shared `printAgentResults` reporting loop used by both commands: explicitly reports `unsupported` for a target/location combination the agent doesn't support, rather than relying on an empty `WriteResult`
- Both commands registered in `root.go`; zero imports of `internal/graphstore`/`internal/indexer`/`internal/query` (boundary grep clean)
- Command-level RED→GREEN test coverage in `install_test.go`: 8 install behaviors + 4 uninstall status/no-error behaviors + 2 install→uninstall round trips (sibling-entry preservation; a temp-`$HOME` full round trip standing in for the live-agent checkpoint)

## Task Commits

Each task was committed atomically; both `tdd="true"` tasks landed as a RED test commit followed by a GREEN implementation commit:

1. **Task 1: install.go — flag resolution, exec-path, multi-select, status** - `2fba6f8` (test, RED) → `59e0dcb` (feat, GREEN)
2. **Task 2: uninstall.go — per-agent status reversal** - `832285a` (test, RED) → `f0cdc38` (feat, GREEN)
3. **Task 3: Live-agent MCP handshake verification** - auto-handled (no commit; see "User Setup Required" below)

## Files Created/Modified
- `internal/cli/install.go` - `newInstallCmd()`, `promptAgentMultiSelect()`, `selectByIndices()`, `installStatus()`, shared `printAgentResults()`, `parseLocationFlag()`, `installStdinIsInteractive` seam
- `internal/cli/uninstall.go` - `newUninstallCmd()`, `uninstallStatus()`
- `internal/cli/install_test.go` - `fakeHome()`/`readJSONMap()`/`readFileString()` test helpers plus 14 test functions covering both commands
- `internal/cli/root.go` - registers `newInstallCmd()` and `newUninstallCmd()`

## Decisions Made
- `uninstall` defaults `--target` to `"all"` (not `"auto"`) with no interactive prompt at all when the flag is omitted — a destructive-by-default reversal is safe to attempt across the whole roster since `Uninstall` never errors on an unconfigured agent (D-08); this also trivially satisfies the no-TTY/CI non-blocking requirement without a dedicated TTY-detection branch on the uninstall path.
- `printAgentResults` checks `AgentTarget.SupportsLocation(loc)` in the CLI layer before ever calling `Install`/`Uninstall`, rather than relying on each target's own empty-`WriteResult`-on-unsupported-location behavior — produces an explicit `"unsupported"` status line instead of a silent `"(no files touched)"`.
- `installStdinIsInteractive` is a package-level var (not an inline `os.Stdin.Stat()` call) so `install_test.go`'s `execCmd`-driven tests exercise the real non-interactive fallback code path without a pty — mirrors `upgrade.go`'s `upgradeRunFunc` injectable-seam convention from 06-06.
- TTY detection uses stdlib only (`cmd.InOrStdin() == os.Stdin` AND `os.ModeCharDevice`), per RESEARCH.md's explicit no-new-terminal-dependency guidance; `golang.org/x/term` is already an indirect dependency via `sigstore-go` but was deliberately not promoted to direct for this.

## Deviations from Plan

None — plan executed exactly as written. The one adaptation was procedural, not behavioral: since this was an autonomous `--auto` pipeline run, Task 3's `checkpoint:human-verify` (a live-agent MCP handshake) could not be performed for real. Per the auto-mode substitution rule, `TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState` was added as automated evidence (a temp-`$HOME` install→uninstall round trip asserting the written Claude Code MCP config's `command`/`args` shape and the `CLAUDE.md` instructions block's marker fences and `codegraph_explore` reference, then asserting uninstall restores pre-install state modulo the CodeGraph section). The residual live-agent handshake itself is deferred as a manual follow-up — see "User Setup Required" below.

## Issues Encountered
None.

## User Setup Required

**Manual follow-up (residual live-agent verification, per 06-VALIDATION.md's Manual-Only Verifications):** This plan's Task 3 checkpoint asked for a real coding-agent runtime to confirm `codegraph_explore` actually appears after `codegraph install` and disappears after `codegraph uninstall`. That cannot be exercised in this autonomous execution environment. Before relying on `install`/`uninstall` in production, a human should:

1. Build the current binary: `go build -o /tmp/codegraph ./cmd/codegraph`
2. If any of the roster agents are installed locally (Claude Code, Cursor, Codex, opencode, Gemini, Kiro, Hermes, Antigravity), run `/tmp/codegraph install --target auto` in a repo with a `.codegraph/` index (`/tmp/codegraph init` first if needed).
3. Open that agent and confirm the `codegraph_explore` MCP tool is listed and returns results on a sample query (Kiro: enable MCP in Settings first — `install` surfaces this note).
4. Run `/tmp/codegraph uninstall --target auto` and confirm the tool disappears and unrelated config entries are untouched (diff against a pre-install copy).
5. If no roster agent is installed locally, hand-inspect a written config (e.g. `install --target claude --location local` in a scratch dir, then read `./.mcp.json`) to confirm the entry shape.

## Next Phase Readiness
- `codegraph install`/`codegraph uninstall` are the complete, tested drop-in-swap entrypoints for all 8 roster agents (AGNT-01/02/03 closed for this phase's CLI surface)
- `install`/`uninstall`/`upgrade`/`version`/`telemetry` are now all registered in `root.go` — phase 6's full command surface is wired
- No blockers for phase 7 (migration tool) or phase 8 (release engineering, which will finalize the live-agent verification as part of its own integration testing)

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 4 created/modified files found on disk (install.go, uninstall.go, install_test.go, root.go); all 4 task commit hashes (2fba6f8, 59e0dcb, 832285a, f0cdc38) found in git log.
