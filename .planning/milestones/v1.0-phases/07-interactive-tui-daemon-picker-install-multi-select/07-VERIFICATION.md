---
phase: 07-interactive-tui-daemon-picker-install-multi-select
verified: 2026-07-18T00:00:00Z
status: passed
score: 7/7 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Visual rendering of the bubbletea daemon picker on a real TTY (colors, layout, selection highlight, current-project-first ordering)"
    expected: "codegraph daemon (with ≥1 running daemon) opens a legible checkbox-free single-select list, current repo's daemon(s) shown first, stop-one/stop-all/cancel all work and render correctly"
    why_human: "ANSI/terminal visual output and interactive keypress UX are not byte-assertable; Model.Update transitions are unit-proven via synthetic tea.Msg but the actual rendered frame on a real pty has not been eyeballed"

  - test: "Visual rendering of the install/uninstall checkbox multi-select on a real TTY"
    expected: "codegraph install (no --target, no -y) shows a checkbox list pre-checked for already-installed agents; space toggles, enter confirms, q/esc cancels — legible glyphs, correct pre-check state"
    why_human: "Same as above — checkbox glyphs and pre-check marks are visual; only the underlying state machine is unit-tested"

  - test: "Windows daemon stop real termination semantics (hard-kill via TerminateProcess) and Windows PPID-liveness watchdog behavior on a real Windows host"
    expected: "`daemon start` then `daemon stop` from another shell terminates the daemon process and its registry record is cleared; a daemon whose supervising process is killed exits on its own within ~1-2s"
    why_human: "No Windows CI runner exists for this project (documented precedent: compile-only `GOOS=windows go vet`, cross-verified locally in this verification via a real mingw-w64 CGo cross-toolchain) — real Windows process-termination and OpenProcess/WaitForSingleObject behavior needs a Windows host to observe"
---

# Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select Verification Report

**Phase Goal:** The `daemon` command becomes an interactive picker (resolving the TS name collision) backed by explicit start/stop lifecycle and a PPID watchdog, `install`/`uninstall` present a multi-select, and every interactive surface auto-falls back to non-interactive behavior when piped — never hanging.
**Verified:** 2026-07-18
**Status:** passed — canonicalized 2026-07-26 from `human_needed` after human UAT; one accepted platform gap (see Acknowledged Gaps: AG-07-01)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | DMON-01/TUI-04: `codegraph daemon` (no args) opens a bubbletea picker on TTY (current project first); off-TTY prints a plain running-daemon list and exits 0, never blocking stdin | ✓ VERIFIED | `internal/cli/daemon.go:59-79` gates on `interactiveAllowed(cmd)` before calling `runDaemonPicker` (= `tui.RunDaemonPicker`); off-TTY calls `printDaemonList` (never opens a `tea.Program`). `internal/cli/tui/daemonpicker.go` is a fully substantive bubbletea Model (222 lines) with real `list.Model`, sort-current-first, stop-one/stop-all/cancel dispatch. `TestPipedNeverHang/daemon_bare` (real spawned binary, piped stdio) passes and asserts no ANSI escape + "no running daemons" output within 10s bound. |
| 2 | DMON-02: `daemon start` / `daemon stop` / `daemon stop --all` subcommands exist and route correctly; no silent auto-spawn | ✓ VERIFIED | `internal/cli/daemon.go` cobra tree: `newDaemonCmd()` → `AddCommand(newDaemonStartCmd(), newDaemonStopCmd())`. `daemon start` moves the old foreground `RunE` verbatim (incl. `watch.DisabledError` friendly-exit branch). `daemon stop [--all]` dispatches to `daemon.StopMatching`/`daemon.StopAll` — never calls `daemon.Run`. Bare `daemon` and `stop` never call `daemon.Run` either (grep-confirmed: only `newDaemonStartCmd`'s RunE calls `d.Run(ctx)`). `internal/cli/daemon_test.go` covers start/stop/stop-all/no-match/aggregated-error routing (9 tests, all pass). |
| 3 | DMON-03: PPID watchdog cancels daemon/watcher ctx on parent death; goleak-clean; Windows-tagged file compile-verified | ✓ VERIFIED | `internal/daemon/watchdog.go` + `watchdog_posix.go` (`getppid() != original`, subreaper-robust) + `watchdog_windows.go` (`OpenProcess`+`WaitForSingleObject` liveness poll, x/sys/windows). Wired into `daemon.Run` (`daemon.go:250-266`): derived cancellable ctx, `stop()` joined via one `defer{cancel();stop()}` on every teardown path. `go test ./internal/daemon/... -race -count=1` passes (16.4s, goleak.VerifyTestMain gate). `TestWatchdogCancelsOnReparent`, `TestWatchdogJoinsOnCtxCancelWithoutFiringCancel`, `TestRunWatchdogCancelsRunOnSimulatedReparent` all pass — this is a genuine behavioral test of the cancellation invariant, not presence-only. `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` (and `go build`) both exit 0 locally, confirming the Windows file typechecks with a real cross-CGO toolchain — matching CI's `.github/workflows/ci.yml` mingw-w64 gate (lines 139-146). |
| 4 | DMON-04: global `~/.codegraph/daemons` registry with self-healing `List()` reusing `lock.go`'s `isProcessLive`/`isStale` | ✓ VERIFIED | `internal/daemon/registry.go`: `Record{PID,StartedAt,RepoRoot}`, `Register` via `fsatomic.WriteFile` (atomic, pid-keyed `<pid>.json`), `Deregister` (IsNotExist-is-nil), `List()` calls `isStale(lockInfo{...})` (same-package, unexported, no rival liveness impl) and prunes in place. `TestRegistryRegisterDeregister`, `TestRegistrySameRepoRootDistinctFiles`, `TestRegistryListPrunesStale`, `TestRegistryListMissingDir`, `TestRegistryListEmptyDir` all pass. Wired into `daemon.Run` (register-after-acquire, deregister-via-defer) — `TestRunRegistersRecordAfterAcquire`, `TestRunDeregistersRecordOnCleanShutdown`, `TestRunPolicyDisabledRegistersNothing` all pass. |
| 5 | TUI-03: install/uninstall bubbles multi-select with `-y`/`--yes`; charm confined to `internal/cli/tui` | ✓ VERIFIED | `internal/cli/tui/agentpicker.go` (197 lines): real `list.Model` + hand-rolled `checkboxDelegate` (toggle in delegate's own `Update`, avoiding list.Model keymap collision), pre-checked from `agents.DetectAll(loc)`, resolves via `selectByIndices` (dedup+ascending, mirrors legacy). `install.go`/`uninstall.go` both add `-y`/`--yes` short-circuiting FIRST in the switch (before the TTY branch) to `agents.ResolveTargetFlag`. `rg -n "charm.land" internal/agents/*.go internal/daemon/*.go` returns nothing — charm confined to `internal/cli/tui`. `TestAgentPickerModel_PreChecksDetectedTargets`, `_SpaceTogglesFocusedRow`, `_EnterResolvesCheckedSetInAscendingOrder`, `_QuitYieldsEmptySelection` all pass. |
| 6 | TEST-03: `internal/githooks` byte-invariance test + `test/integration` piped never-hang test | ✓ VERIFIED | `internal/githooks/githooks_test.go#TestInstall_EditThenRemove_ByteInvariant` (lines 392-466): install → user edit outside marker → remove asserted byte-identical to (pre-install original + edit); also backstops install→install fixed point and remove→remove no-op. `test/integration/piped_never_hang_test.go#TestPipedNeverHang`: spawns the real binary for `daemon` (bare) and `install` with piped stdio under a 10s bounded goroutine+select (hang → `t.Fatalf`, never blocks the suite); asserts no ANSI escape leakage. Both pass (`go test ./internal/githooks/... -run TestInstall_EditThenRemove_ByteInvariant` and `go test ./test/integration/... -run TestPipedNeverHang -v`). |
| 7 | The Phase-6 ANSI-isolation archtest (`internal/cli/present/archtest`) stays GREEN with bubbletea/bubbles added — never reaches `internal/daemon`/`internal/mcp`/`internal/query`/etc. | ✓ VERIFIED | `go test ./internal/cli/present/archtest/ -run TestNoCharmInServeReachablePackages` passes. `guardedPackages` already includes `internal/daemon`, `internal/mcp`, `internal/query`, `internal/watch`, `internal/indexer`, `internal/graphstore`; `forbiddenImportPaths` already lists `charm.land/bubbletea/v2` + `charm.land/bubbles/v2`; `internal/cli` is excluded by construction so the new `internal/cli/tui` package sits outside the guarded closure. Self-defeat guard (`assertCharmImporterExists`) also passes, proving the test isn't vacuously green. |

**Score:** 7/7 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/cli/tui/tty.go` | `InteractiveAllowed` dual-TTY gate | ✓ VERIFIED | Composes `present.ChoosePresentation` + stdin-TTY check; 4 injectable seams unit-tested |
| `go.mod` bubbletea/v2 + bubbles/v2 | pinned exact versions | ✓ VERIFIED | `charm.land/bubbles/v2 v2.1.1`, `charm.land/bubbletea/v2 v2.0.8` present as direct requires |
| `internal/daemon/registry.go` | Record/Register/Deregister/List | ✓ VERIFIED | Substantive, 115 lines, reuses `isStale`/`isProcessLive`, no charm import |
| `internal/daemon/registry_test.go` | round-trip, self-heal, empty/missing dir | ✓ VERIFIED | 5 tests, all pass |
| `internal/daemon/watchdog.go` + `watchdog_posix.go` + `watchdog_windows.go` | ppid poll/reparent-cancel, joinable stop() | ✓ VERIFIED | All three present; POSIX+Windows both compile; behavioral tests pass |
| `internal/daemon/stop.go` + `stop_posix.go` + `stop_windows.go` | SIGTERM/hard-kill split + StopMatching/StopAll | ✓ VERIFIED | Present, isStale re-corroboration before signaling, tests pass incl. real-SIGTERM-to-live-process test |
| `internal/cli/tui/agentpicker.go` | RunAgentPicker + checkboxDelegate | ✓ VERIFIED | 197 lines, fully substantive, no stub patterns |
| `internal/cli/tui/daemonpicker.go` | RunDaemonPicker + sort/dispatch | ✓ VERIFIED | 222 lines, fully substantive, no stub patterns |
| `internal/cli/daemon.go` | cobra tree: bare/start/stop | ✓ VERIFIED | Restructured; watch.DisabledError branch preserved verbatim |
| `internal/cli/install.go` / `uninstall.go` | `-y`/`--yes` + picker wiring | ✓ VERIFIED | Both wired; `-y` short-circuits first in the switch |
| `.github/workflows/ci.yml` | GOOS=windows vet extended to `./internal/daemon/` | ✓ VERIFIED | mingw-w64 cross-CGO toolchain step added; locally reproduced, exits 0 |
| `internal/githooks/githooks_test.go` | byte-invariance test | ✓ VERIFIED | `TestInstall_EditThenRemove_ByteInvariant` present and passing |
| `test/integration/piped_never_hang_test.go` | daemon+install piped never-hang | ✓ VERIFIED | Present and passing, bounded-timeout wrapper correctly structured |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/tui.InteractiveAllowed` | `present.ChoosePresentation` | composed call | ✓ WIRED | `tty.go:50` |
| `daemon.go` bare RunE | `tui.RunDaemonPicker` / plain list | `interactiveAllowed` gate | ✓ WIRED | `daemon.go:74-78` |
| `daemon.go` stop cmd | `daemon.StopMatching`/`StopAll` | func-var seam | ✓ WIRED | `daemon.go:181-182, 206, 223` |
| `registry.List` | `lock.go isStale` | same-package call | ✓ WIRED | `registry.go:107` |
| `registry.Register` | `fsatomic.WriteFile` | atomic write | ✓ WIRED | `registry.go:57` |
| `daemon.Run` | `registry.Register`/`Deregister` | defer shape | ✓ WIRED | `daemon.go:241-248` |
| `daemon.Run` | `startWatchdog`/`stop()` | joined defer | ✓ WIRED | `daemon.go:262-266` |
| `install.go`/`uninstall.go` RunE | `tui.InteractiveAllowed` → `tui.RunAgentPicker` | func-var seam | ✓ WIRED | `install.go:22-23,87-88`; `uninstall.go:52-53` |
| `stop.go stopTargets` | `sendStop` (platform split) | func-var seam | ✓ WIRED | `stop.go:14, 65` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Archtest stays green with new deps | `go test ./internal/cli/present/archtest/ -run TestNoCharmInServeReachablePackages` | PASS | ✓ PASS |
| Full build | `go build ./...` | exit 0 | ✓ PASS |
| Full vet | `go vet ./...` | exit 0 | ✓ PASS |
| Full test suite | `go test ./...` | all packages ok | ✓ PASS |
| Daemon package race+goleak | `go test ./internal/daemon/... -race -count=1` | ok (16.4s) | ✓ PASS |
| Windows watchdog/stop cross-typecheck | `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` | exit 0 | ✓ PASS |
| Windows cross-build | `CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build ./internal/daemon/...` | exit 0 | ✓ PASS |
| Piped never-hang integration | `go test ./test/integration/... -run TestPipedNeverHang -v -count=1` | PASS (0.5s) | ✓ PASS |
| githooks byte-invariance | `go test ./internal/githooks/... -run TestInstall_EditThenRemove_ByteInvariant` | PASS | ✓ PASS |
| tui package tests (agentpicker + daemonpicker + tty) | `go test ./internal/cli/tui/... -v -count=1` | 12/12 PASS | ✓ PASS |
| daemon package targeted behavioral tests | `go test ./internal/daemon/... -v -count=1 -run 'TestRegistry\|TestWatchdog\|TestSendStop\|TestStop\|TestRun'` | 25/25 PASS | ✓ PASS |

Note: plain `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` (without `CGO_ENABLED=1` explicitly set and without a `CC` cross-compiler) fails with `tree_sitter.Node undefined` — this is the expected, documented cross-compile artifact of `internal/daemon` transitively importing the CGo tree-sitter bindings via `internal/indexer` (noted in the verification brief and in the plans' own `07-RESEARCH.md`/CI comments). CI's real gate uses `CGO_ENABLED=1` + a `gcc-mingw-w64-x86_64` cross-CGO toolchain (`.github/workflows/ci.yml:139-146`), which was reproduced locally above and passes cleanly.

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| DMON-01 | 07-07 | ✓ SATISFIED | Cobra tree + bubbletea picker + plain-list fallback, all tested |
| DMON-02 | 07-04, 07-05, 07-07 | ✓ SATISFIED | start/stop/stop-all routing, no auto-spawn, SIGTERM/hard-kill split |
| DMON-03 | 07-03, 07-05 | ✓ SATISFIED | Watchdog wired into Run, goleak-clean, Windows compile-verified |
| DMON-04 | 07-02, 07-05 | ✓ SATISFIED | Registry self-heals, atomic writes, wired into Run |
| TUI-03 | 07-06 | ✓ SATISFIED | Checkbox multi-select for install+uninstall, `-y`/`--yes` |
| TUI-04 | 07-01, 07-06, 07-07, 07-08 | ✓ SATISFIED | Dual-TTY gate; every interactive surface falls back off-TTY |
| TEST-03 | 07-08 | ✓ SATISFIED | Byte-invariance test + piped never-hang integration test |

No orphaned requirements — REQUIREMENTS.md's Phase 7 row set (DMON-01..04, TUI-03/04, TEST-03) exactly matches the union of `requirements:` fields declared across the 8 plans.

### Anti-Patterns Found

None. Scanned all phase-touched files (`daemon.go`, `install.go`, `uninstall.go`, `tui/tty.go`, `tui/agentpicker.go`, `tui/daemonpicker.go`, `daemon/registry.go`, `daemon/watchdog*.go`, `daemon/stop*.go`, `daemon/daemon.go`, `githooks_test.go`, `piped_never_hang_test.go`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|not yet implemented|coming soon` — zero matches.

### Human Verification Required

1. **Visual rendering of the bubbletea daemon picker on a real TTY**
   **Test:** Run `codegraph daemon` with ≥1 running daemon on a real terminal.
   **Expected:** A legible list, current-repo's daemon(s) shown first, `enter` stops the focused one, `a` stops all, `q`/`esc` cancels — with correct visual feedback.
   **Why human:** ANSI rendering and interactive keypress feel are not byte-assertable; only the `Model.Update` state machine is unit-proven via synthetic `tea.Msg`.

2. **Visual rendering of the install/uninstall checkbox multi-select on a real TTY**
   **Test:** Run `codegraph install` (no `--target`, no `-y`) on a real terminal.
   **Expected:** A checkbox list, pre-checked for already-installed agents, toggled via space, confirmed via enter.
   **Why human:** Same as above — glyph rendering is visual, not unit-testable.

3. **Windows `daemon stop` and PPID-watchdog behavior on a real Windows host**
   **Test:** On Windows, `daemon start`, then `daemon stop` from another shell; separately, kill the daemon's supervising process and observe the daemon exit on its own.
   **Expected:** `daemon stop` hard-kills the process and clears its registry record; the watchdog detects parent death within ~1-2s and shuts the daemon down cleanly.
   **Why human:** No Windows CI runner exists (documented project precedent — compile-only `go vet`). This verification reproduced the Windows cross-typecheck/build locally with a real mingw-w64 CGo toolchain (both pass), but the actual runtime termination/liveness-poll behavior needs a Windows host to observe.

### Gaps Summary

No gaps. All 7 must-have truths, all artifacts (3 levels: exists, substantive, wired), and all key links verified directly against the codebase — not inferred from SUMMARY.md claims. `go build`, `go vet`, `go test ./...`, the daemon package under `-race`, the archtest, the piped-never-hang integration test, and a Windows cross-compile (reproduced locally with `CGO_ENABLED=1` + a real mingw-w64 toolchain, matching CI) all pass. The only open items are the three human-verification items above (visual TUI rendering + Windows runtime semantics), which are inherent to this phase's content (interactive terminal UI + a platform with no CI runner) rather than defects — status is `human_needed`, not `gaps_found`.

## Acknowledged Gaps

Recorded 2026-07-26 during `/gsd-verify-work 7`, when `status` was canonicalized
`human_needed` → `passed`. Human-verification items 1 and 2 were observed and
passed on a real TTY (see `07-UAT.md` tests 1-2, including the two rendering bugs
they caught: G-07-1 and G-07-2). Item 3 was **never observed** — it is accepted
here as a platform gap, not claimed as verified.

### AG-07-01 — Windows `daemon stop` termination and PPID-watchdog runtime semantics are unobserved

- **Maps to:** `07-UAT.md` test 3 (`result: skipped`); Human Verification Required item 3.
- **Unverified claim:** on Windows, `daemon stop` hard-kills the target process
  (`TerminateProcess`) and clears its registry record; and a daemon whose
  supervising process dies exits on its own within ~1-2s via the PPID watchdog.

- **Why unverified:** the project has no Windows CI runner, and the tester is on
  macOS with no Windows host available. This is a platform-access gap, not a
  suspected defect — nothing observed suggests the Windows path is broken.

- **What *is* verified:** the Windows-tagged code compiles and type-checks under a
  real mingw-w64 CGo cross-toolchain (`CGO_ENABLED=1`, `GOOS=windows go vet`
  exit 0), reproduced locally during verification and matching CI. Every
  platform-independent path (registry read/write/self-heal, corroborated stop
  targeting, piped-never-hang, the picker state machines) is covered by passing
  automated tests on macOS/Linux.

- **Residual risk:** low-to-moderate and confined to Windows. A regression here
  would surface as `daemon stop` reporting success while the process survives, or
  an orphaned daemon outliving its parent — both operationally visible, neither
  silently corrupting state.

- **Accepted by:** repository maintainer, 2026-07-26.
- **To close:** run `07-UAT.md` test 3 on a real Windows host and flip it from
  `skipped` to `pass`, or stand up a Windows CI runner and automate it. Until
  then, treat Windows daemon lifecycle as compile-verified only.

---

*Verified: 2026-07-18*
*Verifier: Claude (gsd-verifier)*
*Gap acknowledged and status canonicalized to `passed`: 2026-07-26 (`/gsd-verify-work 7`) — see Acknowledged Gaps above.*
