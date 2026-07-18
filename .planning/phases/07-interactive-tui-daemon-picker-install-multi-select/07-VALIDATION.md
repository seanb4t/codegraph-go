---
phase: 7
slug: interactive-tui-daemon-picker-install-multi-select
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-18
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Authoritative requirement→test map: `07-RESEARCH.md` §"Validation Architecture".
> The per-task map below is seeded from RESEARCH and reconciled by validate-phase.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` + `go.uber.org/goleak` (already gates `internal/daemon`) |
| **Config file** | none — plain `go test`; CI orchestration in `.github/workflows/ci.yml` |
| **Quick run command** | `go test ./internal/daemon/... ./internal/cli/... ./internal/cli/tui/... ./internal/githooks/...` |
| **Full suite command** | `go build ./... && GOOS=windows GOARCH=amd64 go vet ./internal/daemon/ ./internal/graphstore/ && go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` |
| **Estimated runtime** | ~60–120 seconds |

---

## Sampling Rate

- **After every task commit:** Run the quick run command above
- **After every plan wave:** Run the full suite command above (incl. the extended Windows `go vet` line covering the new `internal/daemon` Windows-tagged files)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 7-XX-XX | TBD | TBD | DMON-01 | — | Picker opens on TTY; plain list off-TTY, exit 0 | unit + integration | `go test ./internal/cli/tui/... -run TestDaemonPicker`; `go test ./test/integration/... -run TestDaemonBarePlainList` | ❌ W0 | ⬜ pending |
| 7-XX-XX | TBD | TBD | DMON-02 | — | start/stop/stop --all lifecycle, no auto-spawn | unit | `go test ./internal/daemon/... -run TestDaemonStartStop` | ❌ W0 | ⬜ pending |
| 7-XX-XX | TBD | TBD | DMON-03 | — | Watchdog cancels ctx on captured-ppid change; Windows poll typechecks | unit + compile-only | `go test ./internal/daemon/... -run TestWatchdogCancelsOnReparent`; `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` | ❌ W0 | ⬜ pending |
| 7-XX-XX | TBD | TBD | DMON-04 | — | Registry self-heals a stale record on List() | unit | `go test ./internal/daemon/... -run TestRegistryListPrunesStale` | ❌ W0 | ⬜ pending |
| 7-XX-XX | TBD | TBD | TUI-03 | — | Checkbox picker pre-checks detected agents; resolves to same install pipeline; `-y` skips | unit + existing install_test.go | `go test ./internal/cli/tui/... ./internal/cli/... -run TestAgentPicker` | ❌ W0 (delegate); ✓ resolution | ⬜ pending |
| 7-XX-XX | TBD | TBD | TUI-04 | — | Every interactive component falls back off-TTY, never hangs | integration (piped, timeout) | `go test ./test/integration/... -run TestPipedNeverHang` | ❌ W0 | ⬜ pending |
| 7-XX-XX | TBD | TBD | TEST-03 | — | githooks install→edit→remove byte-invariant; piped never-hang | unit + integration | `go test ./internal/githooks/... -run TestInstall_EditThenRemove_ByteInvariant`; `go test ./test/integration/... -run TestPipedNeverHang` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/registry_test.go` — DMON-04 register/list/self-heal
- [ ] `internal/daemon/watchdog_test.go` — DMON-03 POSIX reparent-cancel (injectable ppid source, mirroring the `onSyncStart`/`onWatchOpen` test-seam convention in `daemon.go`)
- [ ] `internal/daemon/stop_test.go` (POSIX) — signal delivery to a real short-lived test process
- [ ] `internal/cli/tui/daemonpicker_test.go` — DMON-01 Model.Update transitions (stop-one/stop-all/cancel), no pty needed (feed synthetic `tea.Msg`)
- [ ] `internal/cli/tui/agentpicker_test.go` — TUI-03 checkbox delegate toggle + pre-check-from-DetectAll
- [ ] `internal/cli/daemon_test.go` additions — cobra tree wiring (`start`/`stop`/`stop --all` routing; bare `daemon` TTY-gates)
- [ ] `internal/githooks/githooks_test.go` addition — `TestInstall_EditThenRemove_ByteInvariant` (D-16, the genuine gap)
- [ ] `test/integration/piped_never_hang_test.go` (new) — D-17: real binary, `daemon` + `install`, piped/closed stdin+stdout under `context.WithTimeout`, assert prompt exit + non-interactive output
- [ ] Extend `.github/workflows/ci.yml`'s `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` line to also cover `./internal/daemon/` once `watchdog_windows.go`/`stop_windows.go` exist

*Wave 0 gaps are non-trivial — DMON-03/04 and TUI-03/04 are genuinely new test surfaces; TEST-03's githooks half has a real, specific gap (existing `TestRemove_WithUserContent_PreservesRemainderBytes` covers only remove-preserving-remainder, not the full install→edit-outside-marker→remove == pre-install-original round trip).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual rendering of the bubbletea daemon picker (colors, layout, selection highlight) on a real TTY | DMON-01 / TUI-04 | ANSI/terminal visual output not byte-asserted; Model.Update state is unit-tested but pixel/style rendering needs eyes | `codegraph daemon` on a real terminal with ≥1 running daemon; verify current-project-first ordering, stop-one/stop-all/cancel actions |
| Visual rendering of the install/uninstall checkbox multi-select on a real TTY | TUI-03 | Same — checkbox glyphs / pre-check marks are visual | `codegraph install` on a real terminal; verify detected agents pre-checked, toggle works, Enter installs the selection |
| Windows `daemon stop` graceful vs hard-kill semantics | DMON-02 / DMON-03 | No Windows CI runner (project precedent: compile-only vet); real termination behavior needs a Windows host | Manual on Windows: `daemon start`, then `daemon stop` from another shell; confirm the daemon exits and its registry record is cleared |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
