---
phase: 07-interactive-tui-daemon-picker-install-multi-select
reviewed: 2026-07-18T00:00:00Z
depth: deep
files_reviewed: 17
files_reviewed_list:
  - .github/workflows/ci.yml
  - internal/cli/daemon.go
  - internal/cli/daemon_test.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/uninstall.go
  - internal/cli/tui/tty.go
  - internal/cli/tui/tty_test.go
  - internal/cli/tui/agentpicker.go
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/tui/daemonpicker.go
  - internal/cli/tui/daemonpicker_test.go
  - internal/cli/tui/doc.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/registry.go
  - internal/daemon/registry_test.go
  - internal/daemon/watchdog.go
  - internal/daemon/watchdog_posix.go
  - internal/daemon/watchdog_windows.go
  - internal/daemon/watchdog_test.go
  - internal/daemon/stop.go
  - internal/daemon/stop_posix.go
  - internal/daemon/stop_windows.go
  - internal/daemon/stop_test.go
  - internal/githooks/githooks_test.go
  - test/integration/piped_never_hang_test.go
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-07-18
**Depth:** deep
**Files Reviewed:** 26 (17 non-test + 9 test siblings)
**Status:** issues_found

## Summary

This phase adds the interactive daemon picker, the install/uninstall bubbletea
multi-select, the PPID watchdog, the cross-process `daemon stop`/registry
machinery, and the CI hardening (windows cross-vet + mingw toolchain) that goes
with it. I traced the six flagged high-risk surfaces end-to-end:

1. **Goroutine lifecycle in `daemon.Run`** — I walked every defer-registration
   order (release → Deregister → cancel/stop → w.Close, unwound LIFO) against
   every early-return path (policy-disabled, acquire failure, watch.Open
   failure, normal ctx-cancel, watchdog-triggered cancel, ErrWatcherClosed).
   In every case the watchdog goroutine is joined (via `stop()`) strictly
   before Deregister/release run, and `wg.Wait()`/`deb.Stop()`/`deb.Wait()`
   all execute inline before any defer fires — so a debounce flush already in
   flight is always joined before the lock is released. This matches
   `daemon_test.go`'s explicit regression coverage
   (`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`,
   `TestRunReturnsErrWatcherClosedAndReleasesLock`,
   `TestRunWatchdogCancelsRunOnSimulatedReparent`). I did not find a leak or
   deadlock path.
2. **PPID watchdog** — `parentChanged` on POSIX compares against the
   *captured* original ppid (subreaper-safe), not `== 1`; on Windows it uses
   `OpenProcess(SYNCHRONIZE)+WaitForSingleObject(0)` rather than the
   documented-unreliable `STILL_ACTIVE` check. The `getppid` seam is a
   package var assigned once (in tests) strictly before the watchdog
   goroutine is spawned, so there's no data race on it in the paths I traced.
   The select loop is ticker-driven (1s), not busy-spun, and exits promptly on
   `ctx.Done()` without waiting for the ticker.
3. **Cross-process signaling** — `stopTargets` re-derives `isStale` (via the
   same `processStartTime` corroboration `lock.go` already uses) immediately
   before each `sendStop`/`stopSignal` call, on both POSIX (SIGTERM) and
   Windows (hard-kill) — the same corroboration gates both. The residual
   PID-reuse race between that check and the actual signal delivery is an
   inherent POSIX/Windows signaling limitation (no pidfd-style atomic
   signal-if-still-this-instance primitive is used), already documented and
   accepted by the existing `lock.go` WR-02 commentary this phase's
   `isStale` reuses verbatim — not a new gap introduced here.
4. **Registry integrity** — `Register`/`Deregister` are keyed by pid via
   `fsatomic.WriteFile` (atomic rename, no partial-read window); `List`
   self-heals stale/malformed records without deleting anything it can't
   parse for reasons other than genuine staleness (a JSON decode failure or a
   vanished file is skipped, not treated as "prune this").
5. **Never-hang invariant** — every interactive call site
   (`install.go`, `uninstall.go`, `daemon.go`'s bare RunE) gates on
   `interactiveAllowed(cmd)` (== `tui.InteractiveAllowed`, which itself
   requires both `stdinIsInteractive` AND `stdoutIsTTY`) strictly before
   calling `runAgentPicker`/`runDaemonPicker`. `test/integration/piped_never_hang_test.go`
   exercises this against the real binary with piped stdio and a bounded
   `time.After`, and asserts no ANSI escape leaks onto piped output.
6. **bubbletea Models** — both `agentPickerModel` and `daemonPickerModel`
   intercept enter/quit keys before ever forwarding to `list.Model.Update`,
   so there's no keybinding collision with the list's own dispatch; the
   empty-registry edge in `daemonPickerModel.Init`/`Update` returns
   `tea.Quit` without ever indexing into an empty `records` slice.
7. **CI** — the new mingw-w64 install step and the `GOOS=windows
   go vet ./internal/daemon/` step are well-formed YAML, correctly scoped
   (flat package, no subdirectories to miss), and ordered so the toolchain is
   installed immediately before the step that needs it.

No Critical findings surfaced from this pass. The three Warnings below are
real UX/consistency gaps in the new interactive dispatch path, not
correctness/security defects; the two Info items are narrow cosmetic/registry
edge cases.

## Warnings

### WR-01: Interactive daemon picker gives zero confirmation of what it actually stopped

**File:** `internal/cli/tui/daemonpicker.go:182-222`, `internal/cli/daemon.go:74-76`
**Issue:** The non-interactive `daemon stop [--all]` path
(`internal/cli/daemon.go:194-236`) explicitly loops over every record
`daemonStopMatching`/`daemonStopAll` actually signaled and prints `"stopped
pid %d (%s)\n"` per record (`printStoppedDaemons`, line 240-244), plus a
"no running daemon(s)" notice when nothing matched. The interactive path does
none of this: `dispatchDaemonAction` (`daemonpicker.go:188-198`) discards the
`[]daemon.Record` `stopMatching`/`stopAll` return and only propagates the
`error`:
```go
case daemonActionStopOne:
    _, err := stopMatching(target.RepoRoot)
    return err
```
`RunDaemonPicker` then returns that error straight to `newDaemonCmd`'s RunE
(`daemon.go:74-76`), which prints nothing on success. A user who presses
Enter (or `a`) to stop a daemon (or every daemon) sees the TUI exit and gets
**no on-screen confirmation** that anything happened — success and "the
target had already died since the list was drawn" (a real, documented
possibility given the picker doesn't re-`List()` before dispatch) are
indistinguishable to the user. For `stop-all` specifically, a partial failure
(2 of 3 daemons stopped, 1 errored) surfaces only the error for the failed
one; the two successes are never reported, whereas the non-interactive
`daemon stop --all` reports all three outcomes.
**Fix:** Have `dispatchDaemonAction` return the signaled `[]daemon.Record`
alongside the error, and have `RunDaemonPicker` print the same
`printStoppedDaemons`-shaped confirmation lines (or a "nothing to stop"
notice) after the Program exits, so both entry points give equivalent
feedback for the equivalent operation.

### WR-02: `daemon stop --all --path <p>` silently ignores `--path` with no validation

**File:** `internal/cli/daemon.go:194-236`
**Issue:** `newDaemonStopCmd`'s RunE checks `if all { ...; return }` before
ever resolving `path` — if a caller passes both `--all` and `-p/--path`
(a plausible copy-paste mistake, e.g. from a `daemon start --path X` command
line), `--path` is silently dropped with no error, no warning, and no
indication in `--help` that the two flags are mutually exclusive.
**Fix:** Either `cmd.MarkFlagsMutuallyExclusive("path", "all")` (cobra
supports this directly) or explicitly error when both are set, so the
conflicting input is surfaced instead of one flag winning silently.

### WR-03: `SortRecordsCurrentFirst` uses plain string equality while the actual stop path resolves symlinks

**File:** `internal/cli/tui/daemonpicker.go:56-79`, `internal/daemon/stop.go:32-42`
**Issue:** `SortRecordsCurrentFirst` (shared by both `RunDaemonPicker`'s
Model and `daemon.go`'s non-TTY `printDaemonList` fallback, per its own doc
comment claiming both must never diverge) compares `RepoRoot` via exact
string equality:
```go
iCur := sorted[i].RepoRoot == currentRepo
```
but `StopMatching`'s actual targeting logic normalizes both sides through
`resolveRepoRoot` (`filepath.EvalSymlinks`) specifically to handle symlinked
paths (`stop.go:32-42`). If a user's cwd resolves through a symlink
differently than however the running daemon's `RepoRoot` was recorded (e.g.
macOS's `/tmp` → `/private/tmp`, or a project accessed via two different
symlink paths), the picker/list's "current repo first" ordering can silently
fail to put the current repo first, while `daemon stop` (which does use
`resolveRepoRoot`) would still correctly match it. This is display-ordering
only — no wrong daemon is ever signaled, since dispatch always uses the
target record's own `RepoRoot`, not the sort key — but it's an inconsistency
between the "same ordering" guarantee the doc comment asserts and the
"same repo" guarantee the stop path actually provides.
**Fix:** Route `SortRecordsCurrentFirst`'s current-repo comparison through
the same `resolveRepoRoot` helper `stop.go` uses (exporting it or duplicating
it per the project's existing cross-package-duplication precedent), so the
"is this my repo" answer is consistent everywhere it's asked.

## Info

### IN-01: Daemon picker/list rendering does not dedupe records by pid

**File:** `internal/cli/tui/daemonpicker.go:111-123`, `internal/cli/daemon.go:92-103`
**Issue:** `stopTargets` (`internal/daemon/stop.go:61-67`) explicitly
de-duplicates targets by pid before signaling (`seen[rec.PID]`), but
`newDaemonPickerModel` and `printDaemonList` render every record `List()`
returns verbatim, with no pid-based dedup. In the normal Register/Deregister
lifecycle this can't occur (one file per pid), but a hand-edited or
migrated-from-TS registry directory containing two on-disk records for the
same live pid would render as two rows in the picker/list, even though only
one underlying process exists (and `stopTargets` would correctly collapse
them to one signal if acted on).
**Fix:** Not urgent — noting for awareness. If ever hardened, dedupe by pid
in `newDaemonPickerModel`/`printDaemonList` the same way `stopTargets` does.

### IN-02: `daemon.List()` runs its self-heal side effect on every bare `codegraph daemon` invocation, before the interactivity check

**File:** `internal/cli/daemon.go:69-77`
**Issue:** Not a bug — `daemonList()` (== `daemon.List()`) is called
unconditionally before checking `interactiveAllowed(cmd)`, so every
`codegraph daemon` invocation (interactive or not, including from a script
or CI) prunes stale registry records as a side effect of merely listing them.
This is explicitly documented/intended behavior (`registry.go:74-83`'s "no
background reaper" design), but worth flagging for awareness since it means
a read-only-looking command mutates `~/.codegraph/daemons/` on disk.
**Fix:** None needed; documenting as expected behavior, not a defect.

---

_Reviewed: 2026-07-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
