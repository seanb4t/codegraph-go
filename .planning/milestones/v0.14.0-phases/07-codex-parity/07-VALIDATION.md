---
phase: "7"
slug: "codex-parity"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
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

As built. Each row cites the test names that actually exist at HEAD, not the planned ones.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01 | 01 | 1 | CODEX-02 | T-07-01..03 | Splice and strip never touch bytes outside our own table at any indentation, CRLF is kept, a leading BOM round-trips, and inline or dotted codegraph keys are refused | unit + real binary | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'TestSpliceTOMLTable\|TestStripTOMLTable\|TestFindTOMLTableRange\|TestTOMLTableConflict\|TestTOMLLineEnding' -count=1` (26 top-level TOML tests) | ✅ | ✅ green |
| 07-02 | 02 | 2 | CODEX-02 | T-07-04 | An explicit `--target` is honoured under `--yes`, never widened | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'Test(Install\|Uninstall)_YesWithExplicitTarget_HonoursTarget' -count=1` | ✅ | ✅ green |
| 07-03 | 03 | 3 | FIX-03 | — | One line per picker row, so the help footer stays inside a 100×30 pane | unit (model-level) | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -run 'TestAgentPickerFootprint\|TestDaemonPickerFootprint' -count=1` (4 tests, 5 boundary subtests) | ✅ | ✅ green |
| 07-03 | 03 | 3 | FIX-03 (real PTY) | — | Same footer visible in a real terminal | tmux (manual) | `task test:tmux` — NOT RUN (maintainer decision 2026-09-19, #75); accepted override in 07-VERIFICATION.md; CI `tmux-e2e` is the outstanding run | ✅ compiles (`go vet -tags tmux`) | ⚠️ manual, deferred to CI |
| 07-04 | 04 | 4 | CODEX-01 | T-07-07..10 | Scratch HOME/CODEX_HOME only; the real HOME's four files are sha256-identical before and after | manual/live | `07-LIVE-SESSIONS.md` — `CODEX-01 verdict: PASS` (L1/L2/L3/L4/L7) | ✅ | ✅ recorded |
| 07-05 | 05 | 5 | CODEX-01/02/03 | T-07-11..14 | Local install writes only our paths; a conflicting table is a visible error; the trust Note matches live behaviour | unit + golden + real binary | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'TestCodex_' -count=1`; `go test ./internal/cli/ -run 'TestPlainGolden' -count=1` | ✅ | ✅ green |
| 07-06 | 06 | 6 | CODEX-04 | T-07-15..17 | The shared `AGENTS.md` block survives while another configured target still declares it; every uninstall order restores the original bytes | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'TestSharedAgentsMD_\|TestOwnershipSharedInstructions\|TestCodex_Install_OverrideNote' -count=1` | ✅ | ✅ green |
| 07-07 | 07 | 7 | CODEX-05 | T-07-18..24 | additionalContext only, exit 0 on every path; silent on doubt; ExecPath never in the registered command | unit + real binary | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestHookPreToolUseCodex_\|TestHookPreToolUse_UnknownHarnessIsSilent' -count=1`; `go test ./internal/agents/ -run 'TestCodexPreToolUseGuard\|TestRenderCodexPreToolGuard\|TestCodexHooksFragmentShape' -count=1` | ✅ | ✅ green |
| 07-08 | 08 | 8 | CODEX-05 | T-07-25..28 | Sticky opt-in from our own group; skipped with a Note when the user disabled Codex hooks; foreign groups untouched | unit + drift gate | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'TestCodexPreToolNudge_\|TestOwnershipExactIdentity' -count=1`; `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` | ✅ | ✅ green |
| 07-09 | 09 | 9 | CODEX-05, CODEX-06 | T-07-29..32 | Pass bar locked before the sessions; absences positive-controlled; real HOME unchanged | manual/live | `07-LIVE-SESSIONS.md` — `CODEX-05 live verdict: PASS`, `CODEX-06 verdict: PASS` (L1-post/L5/L6/L7) | ✅ | ✅ recorded |
| 07-10 | 10 | 10 | AGENT-14 | T-07-33..34 | The published table equals `Capabilities()` for 8 targets × 2 scopes; the verification column never overclaims | unit (drift) | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'TestCapabilityDoc_' -count=1` | ✅ | ✅ green |
| 07-11 | 11 | 11 | AGENT-14 | T-07-35..36 | The skill sentence is true for what ships and stays inside the wire budget; transcripts match the shipped string | unit + wire oracle | `GOTOOLCHAIN=go1.26.6 go test ./internal/mcp/ -count=1`; `GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1` | ✅ | ✅ green |
| 07-12 | review-fix | — | CODEX-02 (CR-01/WR-03), CODEX-05 (WR-01) | T-07 (review) | A BOM cannot cause a duplicate table or an unremovable entry; an update never repositions a foreign hook block | unit + real binary | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -run 'UTF8BOM\|TestWriteHookEntry_' -count=1` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

All items below landed during execution (RED-first where the plan required it); nothing is outstanding.

- [x] `internal/agents/toml_test.go`: fixtures for indentation, CRLF, inline-table refusal, dotted-key refusal, subtables inside the range, multi-line strings and multi-line arrays (D-07), plus a planted column-0-only mutation logged in `07-MUTATION-LOG.md`.
- [x] `internal/cli/tui/agentpicker_test.go`: model-level `lipgloss.Height(View()) <= 30` at 100×30 with all 8 names (D-25). `daemonpicker` gets the equivalent.
- [x] `test/tmux`: TTY-05 re-anchored on the footer text (`space: toggle`) instead of the title workaround. `TMUX_EXPECTED_TESTS` is updated in the same commit only if the top-level Test count changes.
- [x] `internal/cli/install_test.go` / `uninstall_test.go`: RED-first `--target X --yes` resolves to exactly X (D-13).
- [x] `internal/agents/codex_test.go`: flip the local-unsupported tests; add tests for local MCP, the skill, `AGENTS.md` sharing (D-11), the `AGENTS.override.md` Note (D-12), and the trust Note (D-10).
- [x] `internal/agents/ownership_test.go`: `case Codex:` in `ownershipWantSkillDir`; codex/local rows.
- [x] Codex hook tests: `hooks.json` write, remove and ownership; opt-in Keep/On/Off; skip on `[features] hooks = false` (and `codex_hooks = false`); guard render; the quoted command form (D-20).
- [x] `internal/cli` Codex adapter tests: argv versus string `tool_input.command`, the session/agent key, D-16 forced-error contract, shell-only D-15 corpora rows.
- [x] AGENT-14 doc-drift test modeled on `TestMatrix_DocMirrorsDescriptor`, with a planted mutation.
- [x] Framework install: none (everything needed is already in `go.mod`). The planned local Homebrew tmux step for D-25 was NOT taken: tmux is retired on this machine (replaced by herdr), the maintainer skipped the local real-PTY evidence on 2026-09-19, and CI `tmux-e2e` (tmux 3.4) remains the outstanding run — GitHub issue #75.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Codex loads project `.codex/config.toml` only when trusted; skill path; `AGENTS.md` chain; the hook command form is shell-expanded | CODEX-01 | Codex's loading, trust and delivery belong to Codex (D-00); only a live session proves them | 07-CONTEXT D-02..D-06, checks L1–L4 plus the A1 smoke test (a guard side-effect marker) in scratch HOME/CODEX_HOME with `auth.json` symlinked; record in `07-LIVE-SESSIONS.md` with dated citations |
| A fresh session reaches for codegraph unprompted; the nudge fires once, then observes the cooldown | CODEX-05, CODEX-06 | Model behaviour and harness delivery | Checks L5–L7: `codex exec --json` plus one interactive Herdr TUI session for `/hooks` trust; inspect raw PreToolUse stdin for any subagent id (research A4) |
| The picker footer is visible in a real 100×30 PTY | FIX-03 | Real terminal rendering | `task test:tmux` (local Homebrew tmux or the `tmux-e2e` CI job) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-19

---

## Validation Audit 2026-09-19

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

State A audit (the strategy was seeded at plan time and is updated here against what shipped). Every automatable requirement in the phase has named, green tests; the per-task map above cites the as-built test names rather than the planned ones. No `gsd-nyquist-auditor` run was needed.

Three items are manual by nature and recorded as such, not as gaps: CODEX-01 and CODEX-06 are live-session evidence (`07-LIVE-SESSIONS.md`, both PASS), and FIX-03's real-PTY assertion is deferred to CI's `tmux-e2e` job under the accepted override in `07-VERIFICATION.md` (GitHub issue #75).

Sampling continuity holds: no three consecutive tasks lack an automated verify, and the full suite (54 packages plus `internal/daemon` separately) was run green at HEAD as the phase's regression gate.
