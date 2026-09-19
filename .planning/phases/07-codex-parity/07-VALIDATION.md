---
phase: "7"
slug: "codex-parity"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-19"
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`), plus the `//go:build tmux` real-PTY suite (`test/tmux`) |
| **Config file** | none; flag-driven (`-update-plain-goldens`, the wire-oracle re-freeze path, `TMUX_EXPECTED_TESTS` in `Taskfile.yml`) |
| **Quick run command** | `go test ./internal/agents/... ./internal/cli/... -run '<Pattern>' -count=1` |
| **Full suite command** | `go test ./...` (excludes tmux); `task test:tmux` (needs tmux) |
| **Estimated runtime** | ~60–120 seconds (full, non-tmux) |

---

## Sampling Rate

- **After every task commit:** the narrowest `-run` pattern covering the touched target or file.
- **After every plan wave:** `go test ./internal/agents/... ./internal/cli/... ./internal/mcp/... -count=1`.
- **Before `/gsd-verify-work`:** `go test ./...` green, plus `task test:tmux` for the re-anchored TTY-05 assertion (local tmux or the `tmux-e2e` CI job), plus `task docs:cli:drift`.
- **Max feedback latency:** 120 seconds.

---

## Per-Task Verification Map

Seeded per requirement. The planner assigns task IDs, and validate-phase fills Status.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-TOML | TOML fix | 1 | CODEX-02 | T-07 (TOML tampering) | Splice and strip never touch bytes outside our own table and subtables, at any indentation and with CRLF; inline or dotted codegraph keys are refused | unit | `go test ./internal/agents/ -run 'TestSpliceTOMLTable\|TestStripTOMLTable\|TestFindTOMLTableRange' -count=1` | ✅ file, ❌ W0 fixtures | ⬜ pending |
| 07-FIX03 | FIX-03 | 1 | FIX-03 | — | N/A | unit + tmux | `go test ./internal/cli/tui/ -run 'TestAgentPicker\|TestDaemonPicker' -count=1`; `task test:tmux` | ✅ file, ❌ W0 height test | ⬜ pending |
| 07-YES | --yes/--target | 1 | CODEX-02 (D-13) | — | Explicit `--target` is honoured under `--yes` (no silent widening to all targets) | unit | `go test ./internal/cli/ -run 'TestInstall.*Yes.*Target\|TestUninstall.*Yes.*Target' -count=1` | ❌ W0 | ⬜ pending |
| 07-LIVE1 | CODEX-01 live | 2 | CODEX-01 | T-07 (real-HOME tampering) | Real `~/.codex`/`~/.agents` sha256 unchanged; scratch HOME/CODEX_HOME only | manual/live | evidence file `07-LIVE-SESSIONS.md` (L1–L4 plus smoke tests A1/A2) | n/a | ⬜ pending |
| 07-SCOPE | scope flip | 3 | CODEX-02, CODEX-03, CODEX-04 | T-07 (ownership) | Local install writes only `.codex/config.toml`, `.agents/skills/codegraph`, and the `AGENTS.md` marker block; shared `AGENTS.md` is kept while another target still requests it | unit + golden | `go test ./internal/agents/ -run 'TestCodex_\|TestOwnershipExactIdentity\|TestOpencode_\|TestSharedInstructions' -count=1`; `go test ./internal/cli/ -run 'TestPrintConfigStyle' -count=1` | ✅ files, ❌ W0 cases | ⬜ pending |
| 07-NUDGE | Codex nudge | 4 | CODEX-05 | T-07 (hook stdin, ownership, consent) | Opt-in only; skipped on explicit `hooks = false`; adapter silent on any doubt; exit 0 always; only `additionalContext`; ExecPath never in `hooks.json` | unit | `go test ./internal/agents/ -run 'TestCodex.*Hook\|TestCodex.*PreTool' -count=1`; `go test ./internal/cli/ -run 'TestHookPreToolUse' -count=1`; `go test ./internal/nudge/ -count=1` | ❌ W0 | ⬜ pending |
| 07-LIVE2 | CODEX-06 live | 5 | CODEX-05, CODEX-06 | T-07 (evidence spoofing) | Pass bar L5–L7 locked in 07-CONTEXT D-06 before any session; negative controls; negative-space logging | manual/live | evidence file `07-LIVE-SESSIONS.md` | n/a | ⬜ pending |
| 07-DOCS | AGENT-14 | 6 | AGENT-14 | T-07 (doc drift) | Doc table equals `Capabilities()` for 8 targets × 2 scopes; MCP instructions ≤600 bytes with the skill sentence inside the first 512 | unit | `go test ./internal/agents/ -run 'TestCapabilityDoc' -count=1`; `go test ./internal/mcp/ -count=1`; `go test ./test/wireoracle/... -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/agents/toml_test.go`: fixtures for indentation, CRLF, inline-table refusal, dotted-key refusal, subtables inside the range, multi-line strings and multi-line arrays (D-07), plus a planted column-0-only mutation logged in `07-MUTATION-LOG.md`.
- [ ] `internal/cli/tui/agentpicker_test.go`: model-level `lipgloss.Height(View()) <= 30` at 100×30 with all 8 names (D-25). `daemonpicker` gets the equivalent.
- [ ] `test/tmux`: TTY-05 re-anchored on the footer text (`space: toggle`) instead of the title workaround. `TMUX_EXPECTED_TESTS` is updated in the same commit only if the top-level Test count changes.
- [ ] `internal/cli/install_test.go` / `uninstall_test.go`: RED-first `--target X --yes` resolves to exactly X (D-13).
- [ ] `internal/agents/codex_test.go`: flip the local-unsupported tests; add tests for local MCP, the skill, `AGENTS.md` sharing (D-11), the `AGENTS.override.md` Note (D-12), and the trust Note (D-10).
- [ ] `internal/agents/ownership_test.go`: `case Codex:` in `ownershipWantSkillDir`; codex/local rows.
- [ ] Codex hook tests: `hooks.json` write, remove and ownership; opt-in Keep/On/Off; skip on `[features] hooks = false` (and `codex_hooks = false`); guard render; the quoted command form (D-20).
- [ ] `internal/cli` Codex adapter tests: argv versus string `tool_input.command`, the session/agent key, D-16 forced-error contract, shell-only D-15 corpora rows.
- [ ] AGENT-14 doc-drift test modeled on `TestMatrix_DocMirrorsDescriptor`, with a planted mutation.
- [ ] Framework install: none (everything needed is already in `go.mod`). Local tmux via Homebrew (3.7c) is the orchestrator's environment step for D-25; CI `tmux-e2e` asserts tmux 3.4.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Codex loads project `.codex/config.toml` only when trusted; skill path; `AGENTS.md` chain; the hook command form is shell-expanded | CODEX-01 | Codex's loading, trust and delivery belong to Codex (D-00); only a live session proves them | 07-CONTEXT D-02..D-06, checks L1–L4 plus the A1 smoke test (a guard side-effect marker) in scratch HOME/CODEX_HOME with `auth.json` symlinked; record in `07-LIVE-SESSIONS.md` with dated citations |
| A fresh session reaches for codegraph unprompted; the nudge fires once, then observes the cooldown | CODEX-05, CODEX-06 | Model behaviour and harness delivery | Checks L5–L7: `codex exec --json` plus one interactive Herdr TUI session for `/hooks` trust; inspect raw PreToolUse stdin for any subagent id (research A4) |
| The picker footer is visible in a real 100×30 PTY | FIX-03 | Real terminal rendering | `task test:tmux` (local Homebrew tmux or the `tmux-e2e` CI job) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
