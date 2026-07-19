---
phase: 7
slug: interactive-tui-daemon-picker-install-multi-select
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-18
validated: 2026-07-19
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
| **Quick run command** | `go test -count=1 ./internal/daemon/... ./internal/cli/... ./internal/cli/tui/... ./internal/githooks/...` |
| **Full suite command** | `go build ./... && CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/ && go test ./... && go test ./test/integration/...` |
| **Estimated runtime** | ~60–120 seconds |

---

## Sampling Rate

- **After every task commit:** Run the quick run command above
- **After every plan wave:** Run the full suite command above (incl. the extended Windows `go vet` line covering the new `internal/daemon` Windows-tagged files)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~120 seconds

---

## Per-Task Verification Map

| Requirement | Source Plan(s) | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|----------------|-----------------|-----------|-------------------|-------------|--------|
| DMON-01 | 07-07 | Picker opens on TTY (current project first); plain list off-TTY, exit 0, never blocks stdin | unit + integration | `go test -count=1 ./internal/cli/tui/... -run 'TestDaemonPickerModel\|TestSortRecordsCurrentFirst\|TestPrintDaemonPickerResult'`; `go test -count=1 ./internal/cli/ -run 'TestDaemonBareCmd'`; `go test -count=1 ./test/integration/... -run 'TestPipedNeverHang/daemon_bare'` | ✓ | ✅ green |
| DMON-02 | 07-04, 07-05, 07-07 | start/stop/stop --all lifecycle, no auto-spawn; real SIGTERM delivery | unit | `go test -count=1 ./internal/cli/ -run 'TestDaemonStopCmd\|TestDaemonStartCmd'`; `go test -count=1 ./internal/daemon/... -run 'TestSendStop\|TestStop'` | ✓ | ✅ green |
| DMON-03 | 07-03, 07-05 | Watchdog cancels ctx on captured-ppid change; joins without firing on clean cancel; Windows poll typechecks | unit + compile-only | `go test -count=1 ./internal/daemon/... -run TestWatchdog`; `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` | ✓ | ✅ green |
| DMON-04 | 07-02, 07-05 | Registry self-heals a stale record on List(); atomic pid-keyed writes | unit | `go test -count=1 ./internal/daemon/... -run TestRegistry` | ✓ | ✅ green |
| TUI-03 | 07-06 | Checkbox picker pre-checks detected agents; resolves to same install pipeline (ascending, dedup); `-y` skips | unit | `go test -count=1 ./internal/cli/tui/... -run TestAgentPickerModel` | ✓ | ✅ green |
| TUI-04 | 07-01, 07-06, 07-07, 07-08 | Every interactive component falls back off-TTY, never hangs | unit + integration (piped, timeout) | `go test -count=1 ./internal/cli/tui/... -run TestInteractiveAllowed`; `go test -count=1 ./test/integration/... -run TestPipedNeverHang` | ✓ | ✅ green |
| TEST-03 | 07-08 | githooks install→edit→remove byte-invariant; piped never-hang | unit + integration | `go test -count=1 ./internal/githooks/... -run TestInstall_EditThenRemove_ByteInvariant`; `go test -count=1 ./test/integration/... -run TestPipedNeverHang` | ✓ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/daemon/registry_test.go` — DMON-04 register/list/self-heal (5 tests green)
- [x] `internal/daemon/watchdog_test.go` — DMON-03 POSIX reparent-cancel + join-without-fire (2 tests green; injectable ppid source)
- [x] `internal/daemon/stop_test.go` (POSIX) — signal delivery to a real short-lived test process (`TestSendStop`/`TestStopAll`/`TestStopMatching` green)
- [x] `internal/cli/tui/daemonpicker_test.go` — DMON-01 Model.Update transitions (stop-one/stop-all/cancel/empty-guard), synthetic `tea.Msg`, no pty (9 tests green)
- [x] `internal/cli/tui/agentpicker_test.go` — TUI-03 checkbox delegate toggle + pre-check-from-DetectAll + ascending-resolve (4 tests green)
- [x] `internal/cli/daemon_test.go` — cobra tree wiring (start/stop/stop --all/no-match/aggregated-error/mutually-exclusive routing; bare `daemon` TTY-gates) (11 tests green)
- [x] `internal/githooks/githooks_test.go` — `TestInstall_EditThenRemove_ByteInvariant` (D-16, the genuine gap) green
- [x] `test/integration/piped_never_hang_test.go` — D-17: real binary, `daemon` + `install`, piped/closed stdin+stdout under `context.WithTimeout`, asserts prompt exit + no ANSI leak (2 subtests green)
- [x] `.github/workflows/ci.yml` `GOOS=windows GOARCH=amd64 go vet` extended to `./internal/daemon/` (mingw-w64 cross-CGO gate; reproduced locally, exit 0)

*All Wave 0 gaps closed during execution and reconciled here. DMON-03's Windows cross-vet — listed as compile-only/CI-gated in the plan — was reproduced locally with a real `x86_64-w64-mingw32-gcc` toolchain (exit 0), so it counts as automated coverage rather than manual-only.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions | UAT Result |
|----------|-------------|------------|-------------------|------------|
| Visual rendering of the bubbletea daemon picker (colors, layout, selection highlight, alt-screen) on a real TTY | DMON-01 / TUI-04 | ANSI/terminal visual output not byte-asserted; Model.Update state is unit-tested but rendered frame needs eyes | `codegraph daemon` on a real terminal with ≥1 running daemon; verify current-project-first ordering, stop-one/stop-all/cancel actions | ✅ UAT Test-1 PASS (07-UAT.md) — alt-screen picker renders, enter stopped pid, terminal restored |
| Visual rendering of the install/uninstall checkbox multi-select on a real TTY | TUI-03 | Same — checkbox glyphs / pre-check marks are visual | `codegraph install` on a real terminal; verify detected agents pre-checked, toggle works, Enter installs the selection | ✅ UAT Test-2 PASS (07-UAT.md) — 8 agents pre-checked `[x]`, picker renders |
| Windows `daemon stop` graceful vs hard-kill semantics + PPID-liveness watchdog runtime behavior | DMON-02 / DMON-03 | No Windows CI runner (project precedent: compile-only vet); real process termination needs a Windows host | Manual on Windows: `daemon start`, then `daemon stop` from another shell; confirm the daemon exits and its registry record is cleared | ⏭️ UAT Test-3 SKIPPED — macOS tester, no Windows host (cross-vet passes) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-07-19 (all 7 requirements COVERED by green automated tests; 0 gaps)

---

## Validation Audit 2026-07-19

| Metric | Count |
|--------|-------|
| Requirements audited | 7 |
| COVERED (automated, green) | 7 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Resolved | 0 |
| Escalated (manual-only) | 0 |

**Method:** State A reconciliation. Reconciled the pre-execution draft (placeholder `7-XX-XX`/`TBD` task IDs, `status: draft`) against the executed codebase. Confirmed all 9 Wave-0 test files exist, enumerated their named test functions, and ran the full mapped suite fresh with `-count=1` (no cache) — 14 tui + 10 daemon + 11 cobra + byte-invariance + 2 integration subtests all PASS. Reproduced the DMON-03 Windows cross-vet gate locally (`CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/`, exit 0). No `gsd-nyquist-auditor` spawn needed — zero gaps. Manual-only items (visual TUI rendering, Windows runtime kill) are inherently non-automatable and were exercised by human UAT (07-UAT.md: 2 pass, 1 Windows skip).
