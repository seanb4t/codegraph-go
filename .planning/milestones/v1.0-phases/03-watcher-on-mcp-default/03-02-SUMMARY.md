---
phase: 03-watcher-on-mcp-default
plan: 02
subsystem: infra
tags: [daemon, watcher, wsl2, lockfile, retry, jitter, goleak, tdd]

# Dependency graph
requires:
  - phase: 03-watcher-on-mcp-default
    provides: "internal/watch/policy.go — WatchDisabledReason(projectRoot, Probe) string, watch.ErrWatchDisabled (03-01)"
provides:
  - "Daemon.Run enforces watch.WatchDisabledReason as its FIRST action, before acquire() — a policy-disabled watcher never touches the lockfile (D-11)"
  - "daemon.Option / daemon.WithProbe(watch.Probe) — backward-compatible variadic constructor option; internal/cli/daemon.go unchanged"
  - "daemon.RunWithRetry(ctx, d, interval, onDeferred) — jittered defer-and-retry convergence loop on ErrLockLive (D-14)"
  - "internal/daemon.jitter(interval) — bounded pseudo-random backoff spread, unexported"
  - "Two-session convergence proven goleak-clean in the package's existing TestMain (D-15)"
affects: [03-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Variadic functional-option constructor (Option/WithProbe) preserving a pre-existing two-arg call site byte-for-byte"
    - "Retry loop as a thin wrapper around an already-self-healing operation (RunWithRetry adds no new staleness/liveness machinery — reuses acquire()'s existing stale-lock recovery, D-16)"
    - "Two-Daemon-instances-sharing-one-root soak fixture for lock-convergence testing, extending the existing goleak TestMain rather than adding a second one"

key-files:
  created: []
  modified: [internal/daemon/daemon.go, internal/daemon/daemon_test.go, internal/daemon/soak_test.go]

key-decisions:
  - "ErrWatchDisabled already lived in internal/watch (from 03-01) — no new sentinel needed in internal/daemon; Run wraps it directly via fmt.Errorf(\"%w: %s\", watch.ErrWatchDisabled, reason)"
  - "RunWithRetry and jitter live in internal/daemon (not internal/cli) specifically so they inherit the package's existing goleak TestMain — internal/cli has no goleak harness at all"
  - "Convergence soak proves sole-writer via separate onSync channels per Daemon instance (not lockfile PID, which is identical for both sessions since they share one test process) — the discriminating signal a same-pid same-process test needs"
  - "The soak's post-cancelA convergence check waits for waitForLock() a second time (lock re-acquired by B) before writing the convB.go probe file — writing immediately after A's shutdown confirmation raced B's not-yet-open fsnotify watcher and intermittently missed the create event; this is a test-timing fix, not a behavior change (see Deviations)"
  - "Requirement WATCH-04 marked complete by this plan; WATCH-02 is intentionally left unmarked — this plan delivers only the policy-gate-before-acquire piece of WATCH-02 inside internal/daemon, while the off-handshake-path structural guarantee (D-06/D-08's serveWatchStart seam) ships in 03-03, which is the plan that actually closes WATCH-02's full acceptance bar"

patterns-established:
  - "Test-only Probe{IsWSL: func() bool { return false }} pinning for any daemon/soak test that must not depend on the host's real WSL detection"

requirements-completed: [WATCH-04]

coverage:
  - id: D1
    description: "Daemon.Run enforces watch.WatchDisabledReason as the FIRST action, before acquire() — a policy-disabled Run returns an ErrWatchDisabled-wrapped error and never creates the lockfile"
    requirement: "WATCH-02"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunPolicyDisabled"
        status: pass
    human_judgment: false
  - id: D2
    description: "New gains a backward-compatible variadic Option/WithProbe(watch.Probe); internal/cli/daemon.go is unchanged and inherits the policy gate for free"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunHonorsDefaultProbeOnNonWSLHost"
      - kind: other
        ref: "git diff --exit-code internal/cli/daemon.go"
        status: pass
    human_judgment: false
  - id: D3
    description: "RunWithRetry loops on ErrLockLive with a jittered backoff honoring ctx.Done(); returns cleanly on nil, ErrWatchDisabled, or any other non-ErrLockLive error without calling onDeferred"
    requirement: "WATCH-04"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWithRetryReturnsNilOnCleanShutdown"
        status: pass
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWithRetryReturnsImmediatelyOnDisabled"
        status: pass
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWithRetryReturnsImmediatelyOnGenuineError"
        status: pass
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWithRetryCtxCancelDuringSleep"
        status: pass
    human_judgment: false
  - id: D4
    description: "Two concurrent sessions converge to exactly one writer: at most one writer while the lock holder is alive, exactly one writer once it exits — goroutine-leak-free under the package's existing goleak TestMain"
    requirement: "WATCH-04"
    verification:
      - kind: unit
        ref: "internal/daemon/soak_test.go#TestConvergenceTwoSessions"
        status: pass
      - kind: other
        ref: "go test ./internal/daemon/ -race -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-07-16
status: complete
---

# Phase 3 Plan 2: Watcher Policy Enforcement + Defer-and-Retry Convergence Summary

**`Daemon.Run` now gates on `watch.WatchDisabledReason` before ever touching the lockfile, and a new `RunWithRetry` helper replaces defer-once with a jittered defer-and-retry loop so concurrent `serve --mcp` sessions converge to exactly one writer once the lock holder exits.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-16T13:57:00Z
- **Completed:** 2026-07-16T14:05:44Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- `Daemon.Run` enforces `watch.WatchDisabledReason(d.repoRoot, d.probe)` as its literal first statement — before `acquire()` — so a policy-disabled watcher (WSL2/`/mnt` or `CODEGRAPH_NO_WATCH=1`) returns an `ErrWatchDisabled`-wrapped error without ever creating `daemon.lock` (D-11)
- `New` gained a backward-compatible variadic `Option`/`WithProbe(watch.Probe)` — `internal/cli/daemon.go` is byte-for-byte unchanged (`git diff --exit-code` clean) and inherits the gate for free
- `RunWithRetry(ctx, d, interval, onDeferred)` loops on `ErrLockLive` with a jittered backoff (`jitter`, unexported, ~0–20% spread) honoring `ctx.Done()`; returns immediately on `nil`, `ErrWatchDisabled`, or any other genuine error without ever calling `onDeferred`
- Two-session convergence proven deterministically in `internal/daemon`'s existing goleak `TestMain`: session A holds the lock while session B retries and defers (at most one writer always), then A exits and B converges to sole writer (exactly one eventually) — verified via separate `onSync` channels per `Daemon` instance and a committed-graph assertion, not just lockfile presence
- `go test ./internal/daemon/ -race -count=1` green, including both the new convergence soak and the pre-existing `TestSoak`

## Task Commits

Each task was committed atomically:

1. **Task 1: Policy-gate-first in Run + Probe field + WithProbe option + RunWithRetry/jitter** - `00054b5` (feat)
2. **Task 2: Two-session convergence soak (WATCH-04, goleak-clean) in soak_test.go** - `eaecc9e` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/daemon/daemon.go` - `probe watch.Probe` field, `Option`/`WithProbe`, variadic `New`, policy-gate-first `Run`, `RunWithRetry`, `jitter`
- `internal/daemon/daemon_test.go` - `TestRunPolicyDisabled`, `TestRunHonorsDefaultProbeOnNonWSLHost`, `TestRunWithRetryReturnsNilOnCleanShutdown`, `TestRunWithRetryReturnsImmediatelyOnDisabled`, `TestRunWithRetryReturnsImmediatelyOnGenuineError`, `TestRunWithRetryCtxCancelDuringSleep`
- `internal/daemon/soak_test.go` - `TestConvergenceTwoSessions` (two-Daemon lock convergence, goleak-clean)

## Decisions Made
- `watch.ErrWatchDisabled` (already exported from `internal/watch` by 03-01) is reused directly — no duplicate sentinel added to `internal/daemon`.
- `RunWithRetry`/`jitter` live in `internal/daemon`, not `internal/cli`, so they inherit the package's existing `goleak.VerifyTestMain` coverage; `internal/cli` has no goleak harness at all (RESEARCH Pattern 3's explicit recommendation).
- The convergence soak distinguishes session A from session B via separate `onSync` channels (not lockfile PID, which is identical for both since they run in the same test process/pid) — the only discriminating signal available in-process.
- Requirement bookkeeping: only **WATCH-04** is marked complete by this plan. WATCH-02's full acceptance bar (watcher startup never delays the MCP handshake — the off-handshake-path structural guarantee) ships in 03-03's `serveWatchStart` seam; this plan only delivers the policy-gate-before-acquire piece of WATCH-02 inside `internal/daemon`, so REQUIREMENTS.md's WATCH-02 row is left open for 03-03 to close.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Convergence soak's post-convergence probe file raced session B's not-yet-open fsnotify watcher**
- **Found during:** Task 2, first run of `TestConvergenceTwoSessions`
- **Issue:** After cancelling session A and confirming its `RunWithRetry` returned, the test immediately wrote `convB.go` to prove session B had converged and become the sole writer. But `RunWithRetry`'s retry sleep means B may not have re-acquired the lock (and therefore not yet re-opened its `watch.Open` fsnotify watcher) at the instant A's shutdown was observed — the file create landed before B's watch was registered and was silently missed, since fsnotify only reports events for changes after a watch is active. The test failed with "session B did not converge and become the sole writer."
- **Fix:** Added a second `waitForLock(t, codegraphDir)` call after confirming A's shutdown, before writing the `convB.go` probe file — mirroring the existing `waitForLock`-then-write discipline every other daemon test in this package already uses (`TestDaemonSharedWriter`, `TestSoak`). This is a test-timing fix only; no production code changed.
- **Files modified:** `internal/daemon/soak_test.go`
- **Verification:** `go test ./internal/daemon/ -run TestConvergenceTwoSessions -count=1` and `-race` both green afterward; full package (`go test ./internal/daemon/ -count=1` and `-race`) green including the pre-existing `TestSoak`.
- **Committed in:** `eaecc9e` (Task 2 commit — the fix landed before the task's single commit, not as a separate follow-up)

---

**Total deviations:** 1 auto-fixed (1 bug, test-only)
**Impact on plan:** No scope creep — a test-fixture timing bug caught and fixed before the task's commit, no production behavior affected.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `daemon.RunWithRetry`, `daemon.WithProbe`, and the policy-gate-before-acquire ordering in `Run` are ready for 03-03 to wire into `serve.go`'s background watcher goroutine (D-06/D-08: `daemon.New`, the policy check, lock acquisition, and `watch.Open`'s walk all move inside the goroutine; RunE calls a `serveWatchStart`-shaped seam then immediately calls `server.ServeStdio(s)`).
- `internal/cli/daemon.go` needs zero changes to inherit the policy gate — confirmed via `git diff --exit-code`.
- WATCH-02's REQUIREMENTS.md row is intentionally left open for 03-03 (see Decisions above) — do not mark it complete until the off-handshake-path structural seam test lands.
- No blockers.

---
*Phase: 03-watcher-on-mcp-default*
*Completed: 2026-07-16*

## Self-Check: PASSED

All created/modified files (`internal/daemon/daemon.go`, `internal/daemon/daemon_test.go`, `internal/daemon/soak_test.go`) and both task commit hashes (`00054b5`, `eaecc9e`) verified present in the working tree and git history.
