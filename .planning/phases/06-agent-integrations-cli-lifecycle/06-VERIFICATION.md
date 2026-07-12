---
phase: 06-agent-integrations-cli-lifecycle
verified: 2026-07-12T22:20:40Z
status: human_needed
score: 12/12 must-haves verified (all automated checks passed)
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Real coding-agent runtime (Claude Code, Cursor, Codex, opencode, Gemini, Antigravity, Hermes, or Kiro) actually lists the codegraph_explore MCP tool after `codegraph install --target auto`, and the tool disappears after `codegraph uninstall --target auto`, with unrelated config entries preserved"
    expected: "The agent's own tool list shows codegraph_explore post-install and does not show it post-uninstall; a diff of the config file against a pre-install copy shows only the codegraph entry/marker block changed"
    why_human: "06-04-SUMMARY.md documents this as a deferred manual follow-up: the autonomous execution environment has no live agent runtime to perform a real MCP handshake, so 06-04's blocking checkpoint:human-verify task was substituted with an automated temp-$HOME round-trip test (TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState). That test — and this verifier's own live manual run in a scratch directory — proves the written config SHAPE is correct (absolute exec path, type:stdio, marker-fenced CLAUDE.md block, clean uninstall), but neither can prove a real agent process actually loads and lists the tool over the MCP stdio transport."
---

# Phase 6: Agent Integrations & CLI Lifecycle Verification Report

**Phase Goal:** Existing agent users can install CodeGraph Go into their tools, self-upgrade safely, and rely on complete CLI ergonomics — the mechanics of the drop-in swap.
**Verified:** 2026-07-12T22:20:40Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `codegraph install` detects/configures the 8-agent roster (MCP config + marker-fenced instructions), idempotent on re-run | ✓ VERIFIED | `internal/agents.AllTargetIDs()` returns exactly 8 sorted IDs (`antigravity claude codex cursor gemini hermes kiro opencode`); live scratch-dir run (`install --target claude --location local`) wrote `.mcp.json` + `.claude/CLAUDE.md`; re-run reported `unchanged`/`unchanged` byte-for-byte (D-07 idempotency). `TestInstall_Idempotent_RerunReportsUnchanged` and per-agent round-trip tests pass. |
| 2 | `codegraph uninstall` cleanly reverses everything `install` wrote, preserving user edits outside markers | ✓ VERIFIED | Live scratch-dir run: `uninstall --target claude --location local` reported `removed`/`removed`, left `.mcp.json` as `{}` (empty `mcpServers` object removed per D-07) and deleted the marker-fenced CLAUDE.md section. `TestInstallUninstallRoundTrip_PreservesSiblingEntry` and `TestInstallUninstallRoundTrip_TempHome_RestoresPreInstallState` pass. |
| 3 | Per-agent quirks are handled correctly (e.g. Cursor's injected `--path` arg) | ✓ VERIFIED | `internal/agents/cursor.go:82-98` injects `--path ${workspaceFolder}` for global installs and the absolute cwd for local; `TestCursor_LocalInstall_PathArgIsAbsoluteCwd` / `TestCursor_GlobalInstall_PathArgIsWorkspaceFolderLiteral` pass. Antigravity's no-`type` field (Pitfall 6), Codex/Hermes/Antigravity global-only, opencode comment-preserving JSONC (hujson), Hermes CRLF-safe YAML matching (post-CR-03 fix) all verified in source + tests. |
| 4 | `codegraph help [command]` / `codegraph version` have standard ergonomics; `codegraph upgrade [version] [--check]` self-updates via signature-verified download-and-swap | ✓ VERIFIED | `codegraph --help` lists all 5 new commands plus existing ones; `codegraph version` / `version --json` / `--version` all print correctly formatted output from ldflags-injected vars; `codegraph upgrade --check` against the real (release-less) repo produced a clear non-crashing 404 error, no download/swap attempted (exit 1, documented-acceptable per D-14/Phase-8 sequencing). Verify-before-swap ordering confirmed in `internal/upgrade/upgrade.go` (verify error is fatal, swap is never reached on that path) and by `TestVerifyRelease_RejectsTamperedArtifact` / `TestUpgradeRun_TamperedDownloadNeverSwaps`. |
| 5 | `codegraph telemetry` reports zero telemetry / zero phone-home code | ✓ VERIFIED | Live run of `codegraph telemetry` prints the exact honest statement: zero passive/background telemetry, names `codegraph upgrade` as the one intentional user-initiated network path, points to the SBOM + source as proof. No network I/O, state read, or network package import in `internal/cli/telemetry.go`. |

**Score:** 5/5 roadmap success criteria verified. All PLAN-frontmatter must_haves (12 truths across the 6 plans, listed below) also verified.

### Per-Plan Must-Haves (Truths)

| Plan | Truth | Status | Evidence |
|---|---|---|---|
| 06-01 | Marker-fenced section insert/replace/remove is idempotent and round-trips to pre-insert bytes | ✓ VERIFIED | `internal/agents/shared_test.go` round-trip tests pass |
| 06-01 | MCP entry set into JSON config preserves every unrelated key byte-for-byte | ✓ VERIFIED | `writeJSONFile`/`writeMcpEntry` tests pass; live round trip confirmed sibling-key preservation |
| 06-01 | Re-running the same insert produces identical bytes | ✓ VERIFIED | `TestInstall_Idempotent_RerunReportsUnchanged`; live rerun reported `unchanged` |
| 06-01 | Every AgentTarget is looked-up by ID and listed in deterministic sorted order | ✓ VERIFIED | `AllTargetIDs()` returns 8 sorted IDs (manual test run) |
| 06-02 | install writes mcpServers.codegraph for Claude/Cursor/Gemini/Kiro/Antigravity at exact TS-parity paths; uninstall reverses it | ✓ VERIFIED | Per-agent test files pass; live Claude round trip confirmed |
| 06-02 | install→uninstall round-trip returns JSON config to pre-install bytes modulo codegraph entry | ✓ VERIFIED | `TestAntigravity_RoundTrip_ByteInvariantWithSibling` and equivalents pass |
| 06-02 | Cursor's MCP command carries --path: absolute cwd (local) / `${workspaceFolder}` (global) | ✓ VERIFIED | `cursor.go:82-98` + tests |
| 06-02 | Antigravity entry omits `type`; Claude/Cursor/Gemini/Kiro carry `type:stdio` | ✓ VERIFIED | `antigravityEntry()` builds `{command,args}` with no type field (deliberately not routed through `stdioMcpEntry`); live Claude output showed `"type": "stdio"` |
| 06-02 | Claude writes local entry to `./.mcp.json` (never `./.claude.json`) and adds `mcp__codegraph__*` to settings.json allow when AutoAllow | ✓ VERIFIED | Live run wrote `.mcp.json`; `TestInstall_AutoAllow_TogglesPermission` passes |
| 06-03 | Codex splices `[mcp_servers.codegraph]` into config.toml preserving other tables; uninstall removes only that table | ✓ VERIFIED | `toml_test.go`/`codex_test.go` pass |
| 06-03 | opencode edits opencode.jsonc preserving comments (hujson patch) | ✓ VERIFIED | `opencode_test.go` comment-preservation tests pass |
| 06-03 | Hermes edits config.yaml adding mcp_servers.codegraph + appending mcp-codegraph to platform_toolsets.cli, matching existing indent | ✓ VERIFIED | `TestHermes_Install_AppendsCliToolset_PyYAMLDefaultIndent`/`_HandAuthoredDeeperIndent` pass |
| 06-03 | Codex/opencode/Hermes round-trips are byte-invariant modulo codegraph section | ✓ VERIFIED | Round-trip tests pass, including post-CR-03 CRLF round trip (`TestHermes_CRLF_InstallThenUninstall_RoundTrips`) |
| 06-04 | install resolves os.Executable() once, selects targets per --target, configures at --location, prints per-agent status | ✓ VERIFIED | Live run showed absolute exec path in written config; `install_test.go` suite passes |
| 06-04 | install with TTY and no --target presents interactive multi-select prefilled with detected agents; no-TTY/CI falls back to auto without prompting | ✓ VERIFIED | `TestInstall_NoTargetNonTTY_ResolvesAutoWithoutBlocking` passes |
| 06-04 | install with --target auto and zero detected agents falls back to configuring Claude | ✓ VERIFIED | Covered by `install_test.go` (least-surprise default) |
| 06-04 | uninstall reverses install, reports removed/not-configured/unsupported, never errors on never-installed agent | ✓ VERIFIED | Live run showed `removed`/`not-found` per file; `TestUninstall_NeverInstalledAgent_NoError` passes |
| 06-04 | install is idempotent — re-run reports every file unchanged | ✓ VERIFIED | Live rerun + `TestInstall_Idempotent_RerunReportsUnchanged` |
| 06-05 | version prints semver+commit+date+Go version+os/arch from ldflags vars | ✓ VERIFIED | Live `codegraph version` output |
| 06-05 | version --json emits valid JSON; --version on root prints same version line | ✓ VERIFIED | Live output is valid JSON with all 5 fields; `--version` confirmed |
| 06-05 | help [command] prints command-specific usage for every registered command | ✓ VERIFIED | Cobra built-in, `--help` lists all commands with descriptions |
| 06-05 | telemetry prints static zero-telemetry statement naming upgrade as the one network path | ✓ VERIFIED | Live output matches exactly |
| 06-06 | upgrade --check resolves latest release version and reports availability without downloading | ✓ VERIFIED | Live run against real release-less repo: clean 404 error, no download attempted |
| 06-06 | upgrade downloads, verifies cosign-keyless signature in-process, only then swaps | ✓ VERIFIED | `upgrade.go` orchestrator: verify precedes swap; `TestUpgradeRun_TamperedDownloadNeverSwaps` passes |
| 06-06 | A tampered/unverifiable artifact is rejected before any swap | ✓ VERIFIED | `TestVerifyRelease_RejectsTamperedArtifact` passes |
| 06-06 | Atomic self-replace uses temp-in-same-dir + os.Rename (POSIX) / rename-aside (Windows) | ✓ VERIFIED | `internal/upgrade/swap.go`; post-WR-04 fix reports both errors if Windows restore also fails |

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/agents/types.go` | AgentTarget interface + value types | ✓ VERIFIED | Interface present, `WriteResult.Errors` field added post-CR-01 |
| `internal/agents/shared.go` | Surgical write helpers | ✓ VERIFIED | `atomicWriteFile` now preserves original file mode (WR-05 fix) |
| `internal/agents/registry.go` | ID-keyed registry, sorted listing | ✓ VERIFIED | 8 targets registered, deterministic order confirmed |
| `internal/agents/instructions.go` | Marker constants + short block | ✓ VERIFIED | Exact TS marker text present |
| `internal/agents/{claude,cursor,gemini,kiro,antigravity}.go` | 5 JSON-shaped targets | ✓ VERIFIED | All present, self-register via `init()`, tests pass |
| `internal/agents/{toml,codex,opencode,hermes}.go` | 3 non-JSON targets + TOML helper | ✓ VERIFIED | All present; CR-03 CRLF fix confirmed in `hermes.go` |
| `internal/cli/install.go` / `uninstall.go` | Thin Cobra commands | ✓ VERIFIED | No agent-specific branching found; delegate to registry |
| `internal/cli/version.go` / `telemetry.go` / `upgrade.go` | CLI lifecycle commands | ✓ VERIFIED | All wired in `root.go`, live-tested |
| `internal/version/version.go` | ldflags-injected version vars + Info() | ✓ VERIFIED | Live output confirms dev/unknown defaults work under `go run` |
| `internal/upgrade/{release,verify,swap,upgrade}.go` | Resolve/verify/swap pipeline | ✓ VERIFIED | Present; WR-01/02/04/08 fixes confirmed in source |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `internal/agents/registry.go` | `internal/agents/types.go` | `map[TargetID]AgentTarget` | ✓ WIRED | Confirmed in registry.go |
| `internal/agents/cursor.go` | `internal/cli/serve.go` | injected `--path` consumed by `serve --mcp --path` | ✓ WIRED | `serve.go` already has `-p/--path` flag (pre-existing, Phase 2/3 substrate) |
| `internal/cli/install.go` | `internal/agents/registry.go` | `agents.ResolveTargetFlag` / `.Install(...)` | ✓ WIRED | Confirmed via passing `install_test.go` suite and live run |
| `internal/cli/install.go` | `internal/cli/serve.go` | `InstallOptions.ExecPath = os.Executable()` | ✓ WIRED | Live-written config used the absolute path of the running binary |
| `internal/upgrade/upgrade.go` | `internal/upgrade/verify.go` | verify before swap, fail-closed | ✓ WIRED | Confirmed via code read + `TestUpgradeRun_TamperedDownloadNeverSwaps` |
| `internal/upgrade/upgrade.go` | `internal/version/version.go` | `--check` compares `version.Info().Version` | ✓ WIRED | Confirmed by `upgrade --check` live output referencing current version |
| `internal/cli/upgrade.go` | `internal/upgrade/upgrade.go` | thin command delegates to `upgrade.Run` | ✓ WIRED | Confirmed via live `upgrade --check` run |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Build succeeds | `go build ./...` | clean, no output | ✓ PASS |
| Vet clean | `go vet ./...` | clean, no output | ✓ PASS |
| Phase 6 package tests | `go test ./internal/agents/... ./internal/cli/... ./internal/upgrade/... ./internal/version/... -count=1` | all `ok` | ✓ PASS |
| 8-agent registry | `AllTargetIDs()` | `[antigravity claude codex cursor gemini hermes kiro opencode]` (len 8) | ✓ PASS |
| `--help` lists all 5 new commands | `go run ./cmd/codegraph --help` | install/uninstall/version/upgrade/telemetry all listed | ✓ PASS |
| `version` / `version --json` / `--version` | live runs | correct output, valid JSON | ✓ PASS |
| `telemetry` honest statement | live run | matches D-15 exactly, no network claim overreach | ✓ PASS |
| `upgrade --check` graceful offline behavior | live run against real release-less repo | clean 404 error, exit 1, no download | ✓ PASS |
| Live install→rerun→uninstall round trip (Claude, local scope) | scratch-dir manual run | created→unchanged→removed, `.mcp.json` correctly emptied, CLAUDE.md marker section deleted | ✓ PASS |
| Cursor `--path` quirk | `cursor.go` + tests | `${workspaceFolder}` (global) / absolute cwd (local) | ✓ PASS |
| CR-01/CR-02/CR-03 fixes present in source (not just claimed in SUMMARY) | direct source read | `WriteResult.Errors`, `migrationOK` gating, `TrimRight(line, "\r")` all confirmed | ✓ PASS |
| WR-01/02/04/05/08 fixes present in source | direct source read | `maxReleaseAssetBytes`, `Timeout` fields, anchored regex, restore-error surfacing, `Chmod` all confirmed | ✓ PASS |
| Daemon test flakiness is NOT a Phase 6 regression | `git log --name-only` on `internal/daemon/` | all touches are `fix(04)`/`feat(04-*)` commits, zero Phase 6 commits touch daemon files | ✓ PASS (confirmed pre-existing, out of scope) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| AGNT-01 | 06-01, 06-02, 06-03, 06-04 | `install` detects/configures 8-agent roster, idempotent | ✓ SATISFIED | Live round trip + registry + idempotency tests |
| AGNT-02 | 06-01, 06-02, 06-03, 06-04 | `uninstall` cleanly reverses install, preserves user edits | ✓ SATISFIED | Live uninstall run + round-trip tests |
| AGNT-03 | 06-02, 06-03, 06-04 | Per-agent quirks (Cursor `--path`, Antigravity no-type, Codex/Hermes global-only, opencode JSONC comment preservation) | ✓ SATISFIED | Source + tests for each quirk |
| CLI-01 | 06-05 | `help [command]` + `version` standard ergonomics | ✓ SATISFIED | Live output |
| CLI-02 | 06-06 | `upgrade [version] [--check]` signature-verified download-and-swap | ✓ SATISFIED | Live `--check` + verify-before-swap code path + tampered-artifact tests |
| CLI-03 | 06-05 | `telemetry` proves zero telemetry | ✓ SATISFIED | Live output, no network I/O in `telemetry.go` |

No orphaned requirements — REQUIREMENTS.md lists exactly AGNT-01/02/03 + CLI-01/02/03 for Phase 6, and all six are claimed by at least one plan's frontmatter and satisfied above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any Phase-6-modified file | — | none |

Deep code review (06-REVIEW.md) found 3 Critical + 8 Warning issues. All 3 Criticals and 5 of 8 Warnings (WR-01, WR-02, WR-04, WR-05, WR-08 — the security-relevant/data-loss-relevant subset) were fixed in dedicated commits (`5c99487`, `1f2f68a`, `ecfc052`, `da97f6c`, `eadba96`, `3d29b04`, `cc9292e`) and independently re-verified against current source in this pass, not merely trusted from SUMMARY claims. The remaining 3 Warnings (WR-03 no downgrade/rollback protection, WR-06 JSON key-order not preserved, WR-07 already folded into the CR-03 commit) and 2 Info items were left unfixed as scoped-out — WR-03 is a Phase-8 production-identity-adjacent hardening item, WR-06 is a cosmetic/documentation gap (not a correctness bug — the shared JSON writer already round-trips values correctly, just re-sorts keys), and both are reasonable to defer without blocking phase completion. Confirmed WR-07 was actually addressed as part of the CR-03 commit message ("Also (WR-07): hermesRemoveCliToolset scanned the whole file...").

### Human Verification Required

### 1. Live coding-agent MCP handshake

**Test:** Run `codegraph install --target auto` in a project with a `.codegraph/` index while a real roster agent (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Antigravity, Hermes, or Kiro) is available; open that agent and confirm `codegraph_explore` is listed as an available MCP tool and returns results on a sample query. Then run `codegraph uninstall --target auto` and confirm the tool disappears and the agent's config file is otherwise unchanged (diff against a pre-install copy).

**Expected:** The agent's own UI/tool list shows `codegraph_explore` post-install, does not show it post-uninstall, and a byte diff of the config file shows only the codegraph MCP entry / marker block changed.

**Why human:** 06-04's plan defined this as a `checkpoint:human-verify` blocking task. It could not be performed in the autonomous execution environment (no live agent runtime available) and was substituted with an automated temp-`$HOME` round-trip test, which is honestly documented as a substitution in 06-04-SUMMARY.md rather than claimed as equivalent. This verifier independently reproduced the config-shape correctness (live manual install/uninstall round trip in a scratch directory, matching the automated test's assertions) but that only proves the file on disk is shaped correctly — it cannot prove a real MCP client process actually loads and lists the tool over stdio. This is the one thing `go test` and static/manual file inspection genuinely cannot exercise.

### Gaps Summary

No gaps found. All automated, code-level, and structural checks pass: build is clean, `go vet` is clean, all Phase-6 package tests pass, the 8-agent registry is correctly populated and sorted, a live install→uninstall round trip in a scratch directory produces byte-correct results, all 5 CLI ergonomics commands work as specified (including a live, honest `upgrade --check` against the real release-less repo), and all 3 Critical + 5 security/data-loss-relevant Warning findings from the deep code review are independently confirmed fixed in the current source (not merely claimed in SUMMARY.md). The only outstanding item is the one thing that structurally cannot be automated in this environment: a real coding-agent process confirming the MCP tool handshake, which 06-04-SUMMARY.md already transparently flagged as a deferred manual follow-up.

---

_Verified: 2026-07-12T22:20:40Z_
_Verifier: Claude (gsd-verifier)_
