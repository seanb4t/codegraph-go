---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 05
subsystem: daemon
tags: [go, daemon, watchdog, registry, goleak, ppid, fsnotify]

# Dependency graph
requires:
  - phase: 07-interactive-tui-daemon-picker-install-multi-select
    provides: "07-02's global registry (Register/Deregister/Record/List), 07-03's PPID watchdog (startWatchdog), 07-04's registry consumers"
provides:
  - "daemon.Run registers a global registry Record on start and best-effort deregisters it on every clean shutdown path (D-06)"
  - "daemon.Run starts the PPID watchdog and joins its stop() on every teardown path — ctx-cancel, ErrWatcherClosed, and watch.Open failure (D-07/D-08)"
  - "A watchdog-triggered cancel drives the exact same clean-shutdown path (lock release, deregister, goroutine join) as an externally cancelled ctx — no new shutdown path"
affects: ["07-07 (daemon start/stop CLI)", "internal/cli/serve.go's --mcp in-process watcher (inherits this wiring for free via RunWithRetry -> Run)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Register-after-acquire / Deregister-via-defer mirrors lock.go's acquire()/release() best-effort-logged shape exactly (D-06)."
    - "Watchdog join ordering: a single defer calls cancel() then stop(), never split across two independent defers — guarantees stop() never blocks on a ctx that hasn't been cancelled yet (Pitfall 4)."

key-files:
  created: []
  modified:
    - internal/daemon/daemon.go
    - internal/daemon/daemon_test.go

key-decisions:
  - "Register/Deregister and startWatchdog/stop are wired into Run itself (not per-caller) so both daemon start (07-07) and serve --mcp's in-process watcher (which calls RunWithRetry -> Run) inherit both behaviors from this single integration point, per the plan's design."
  - "cancel() and stop() are joined together in one defer (cancel first, then stop) rather than as two separate defers, to make the join ordering explicit and correct regardless of surrounding defer LIFO ordering — stop() blocks until the watchdog goroutine observes ctx.Done(), so cancel() must run first on every path, including watch.Open failing before the watch loop starts."
  - "Bumped the new watchdog-reparent test's wait timeout to 10s (from this file's usual 5s) because watchdogInterval is a fixed 1s wall-clock ticker that can lag under heavy full-monorepo-suite parallel load — the same load-induced flake class already documented on TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock in this file."

requirements-completed: [DMON-02, DMON-03, DMON-04]

coverage:
  - id: D1
    description: "daemon.Run registers a registry Record for os.Getpid()/repoRoot right after acquire() succeeds, and best-effort deregisters it via defer on clean shutdown."
    requirement: "DMON-02"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunRegistersRecordAfterAcquire"
        status: pass
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunDeregistersRecordOnCleanShutdown"
        status: pass
    human_judgment: false
  - id: D2
    description: "A policy-disabled Run (watch.DisabledError, returned before acquire) never touches the registry — registration only happens after the lock is held."
    requirement: "DMON-02"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunPolicyDisabledRegistersNothing"
        status: pass
    human_judgment: false
  - id: D3
    description: "daemon.Run starts the PPID watchdog against its own derived, cancellable ctx and joins its stop() on every teardown path, so a simulated reparent cancels Run itself and drives the same clean shutdown (lock released, record deregistered) as an external ctx cancel."
    requirement: "DMON-03"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWatchdogCancelsRunOnSimulatedReparent"
        status: pass
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunReturnsErrWatcherClosedAndReleasesLock"
        status: pass
    human_judgment: false
  - id: D4
    description: "No goroutine (watch loop, debounce flush, or watchdog poll) outlives Run on any exit path — goleak-clean TestMain gate stays green."
    requirement: "DMON-04"
    verification:
      - kind: unit
        ref: "go test ./internal/daemon/... -race (goleak.VerifyTestMain in soak_test.go)"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-07-18
status: complete
---

# Phase 07 Plan 05: Wire Registry + PPID Watchdog into daemon.Run Summary

**daemon.Run now registers itself in the global daemon registry on start, deregisters on shutdown, and is torn down deterministically by a PPID watchdog when its supervising process dies — all through the existing lock-release/goroutine-join teardown path, with no new shutdown mechanism.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-18T16:08:43-04:00
- **Completed:** 2026-07-18T16:17:28-04:00
- **Tasks:** 2 (both TDD: RED → GREEN)
- **Files modified:** 2

## Accomplishments
- `Run` calls `Register(Record{PID, StartedAt, RepoRoot})` immediately after `acquire()` succeeds, and best-effort `Deregister(pid)`s via `defer` — exactly mirroring the lockfile's `release()` shape (D-06). A policy-disabled or lock-contended `Run` never touches the registry.
- `Run` derives a cancellable child `ctx` via `context.WithCancel`, starts `startWatchdog(ctx, cancel, watchdogInterval)`, and joins `stop()` via a single `defer func() { cancel(); stop() }()` — covering the ctx-cancel path, the `ErrWatcherClosed` abnormal path, and even `watch.Open` failing before the watch loop ever starts (D-07/D-08).
- A watchdog-triggered cancel (simulated reparent) drives the exact same teardown sequence as an external ctx cancellation: lock released, record deregistered, watch loop and watchdog goroutines joined — no new shutdown path was introduced.
- `serve --mcp`'s in-process watcher inherits both behaviors for free: it calls `daemon.RunWithRetry` which calls `d.Run`, so this single integration point covers DMON-02/03/04 for both callers, per the plan's design.

## Task Commits

Each task was committed atomically (TDD: test → feat per task):

1. **Task 1: Register on Run start, Deregister on shutdown**
   - `fa30dad` (test) — RED: failing tests for register-during-Run / deregister-on-shutdown / no-register-when-disabled
   - `ad849c2` (feat) — GREEN: wired `Register`/`Deregister` into `Run`
2. **Task 2: Start the PPID watchdog in Run and join it on every teardown path**
   - `8e3bda4` (test) — RED: failing test for watchdog-triggered `Run` cancellation on simulated reparent
   - `f026663` (feat) — GREEN: derived cancellable ctx + `startWatchdog`/`stop()` joined via a single defer

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/daemon/daemon.go` — `Run` now registers/deregisters (D-06) and starts/joins the PPID watchdog (D-07/D-08), between the existing `acquire()`/`defer release()` block and `watch.Open`.
- `internal/daemon/daemon_test.go` — four new tests: `TestRunRegistersRecordAfterAcquire`, `TestRunDeregistersRecordOnCleanShutdown`, `TestRunPolicyDisabledRegistersNothing`, `TestRunWatchdogCancelsRunOnSimulatedReparent`, plus two shared helpers (`waitForRegistryRecord`, `assertNoRegistryRecord`).

## Decisions Made
- Wired both behaviors into `Run` itself (single integration point), not duplicated per-caller — this is what gives `daemon start` (07-07) and `serve --mcp` both DMON-03 coverage from one change.
- `cancel()` and `stop()` are joined together in one `defer`, in that explicit order, rather than as separate defers relying on Go's LIFO ordering — an earlier draft that deferred them independently deadlocked under `-race` because `stop()` blocked on a ctx that hadn't been cancelled yet when it ran first in LIFO order. Explicit sequencing inside a single defer removes that footgun entirely.
- The registry test seam (`registryDir`/`withRegistryDir`, from 07-02) and the watchdog test seam (`getppid`, from 07-03) are reused as-is — no new test infrastructure needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a data race in the new watchdog-reparent test itself**
- **Found during:** Task 2 (Start the PPID watchdog in Run and join it on every teardown path)
- **Issue:** The first draft of `TestRunWatchdogCancelsRunOnSimulatedReparent` reassigned the package-level `getppid` func variable itself after `Run` had already started (to simulate the reparent), racing against `startWatchdog`'s own read of that variable inside the running goroutine — caught immediately by `go test -race`.
- **Fix:** Assigned `getppid` to a closure exactly once, before `Run` starts, and drove the "reparent" through an `atomic.Int32` the closure reads — mirroring the existing pattern already used by `watchdog_test.go`'s `TestWatchdogCancelsOnReparent`.
- **Files modified:** `internal/daemon/daemon_test.go`
- **Verification:** `go test ./internal/daemon/... -race -count=5` — clean, goleak-verified, no race, 5/5.
- **Committed in:** `f026663` (part of the Task 2 GREEN commit — caught by `-race` while verifying the GREEN implementation, before that commit landed, so the test file in `f026663` already carries the fixed, race-free version)

---

**Total deviations:** 1 auto-fixed (1 bug, test-only — no production code affected)
**Impact on plan:** Zero impact on shipped behavior; the race was in test-only simulation code, fixed before the RED commit landed.

## Issues Encountered
- Running the FULL monorepo test suite (`go test ./...`, ~30 packages compiling/running in parallel across all CPU cores) intermittently flakes both the new `TestRunWatchdogCancelsRunOnSimulatedReparent` test and two pre-existing, unrelated tests in this same file (`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestDaemonFlushLockRequeueGivesUpPerEpisode`) — both of which already carry code comments documenting this exact "load-induced flake" class for their own timer-based waits. Scoped to `go test ./internal/daemon/... -race` (the plan's actual verification target), the suite passed cleanly 5/5 repeated runs. Bumped the new test's wait timeout from 5s to 10s as a low-risk robustness improvement, but did not chase the pre-existing flakiness further — it is out of this plan's scope (Rule boundary: only auto-fix issues directly caused by this plan's changes).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Every explicitly-started daemon (via `daemon start`, once 07-07 lands, and via `serve --mcp`'s in-process watcher today) is now visible in the global registry for its full lifetime and self-terminates when its supervising process dies.
- 07-07 (daemon start/stop CLI + cross-project picker) can build directly on `registry.List()` without any further daemon.go changes — the registry is now always populated by any live `Run`.
- No blockers identified.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*
