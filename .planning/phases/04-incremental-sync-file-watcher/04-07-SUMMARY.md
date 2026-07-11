---
phase: 04-incremental-sync-file-watcher
plan: 07
subsystem: infra
tags: [daemon, lockfile, single-writer, fsnotify, go, sync-04, sync-05]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/indexer.Sync(repoRoot, storeDir, opts) from Plan 04-03 — the incremental entry the daemon's debounced flush calls"
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/watch.Watcher/Open/Run + debouncer from Plan 04-05 — the recursive fsnotify watcher the daemon owns"
provides:
  - "internal/daemon package: Daemon (New/Run), a long-lived local process (or in-process-fallback-ready) owning one watcher + the single indexer.Sync writer"
  - "internal/daemon.Unlock(codegraphDir) — stale-only lockfile removal engine for `codegraph unlock` (SYNC-05)"
  - "internal/watch.Debouncer/NewDebouncer/DebounceDuration exported (renamed from unexported debouncer/newDebouncer/debounceDuration) so a daemon in a different package can construct and drive one"
affects: [04-08, 04-09]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "pidfile + start-timestamp lockfile, stdlib-only (os.FindProcess + Signal(0)), stale-only unlock — never stomps a live daemon"
    - "context+sync.WaitGroup lifecycle: Run blocks on ctx.Done() then wg.Wait()s the spawned watcher goroutine before releasing the lock and returning"
    - "syncMu belt-and-suspenders mutex around indexer.Sync, independent of debounce coalescing, so a slow sync can never overlap the next debounced flush"
    - "write-side/read-side sidecar split: internal/daemon touches/clears .codegraph/.sync-pending (write), internal/query/status.go reads it (Phase 04-06) — no shared exported constant, package-local duplication per established project precedent"

key-files:
  created:
    - internal/daemon/lock.go
    - internal/daemon/lock_test.go
    - internal/daemon/daemon.go
    - internal/daemon/daemon_test.go
  modified:
    - internal/watch/debounce.go
    - internal/watch/debounce_test.go
    - internal/watch/watcher.go
    - internal/watch/watcher_test.go

key-decisions:
  - "indexer.Sync manages its own GraphStore.Open/Writer/Commit/Close lifecycle internally (confirmed from Plan 04-03), so Daemon never holds a graphstore.GraphStore/Writer directly — the single-writer invariant is enforced at two levels: the lockfile (only one daemon process system-wide) and an in-process syncMu (no two debounced flushes overlap within that process). internal/daemon therefore never imports internal/graphstore or Pebble."
  - "Unlock returns (message string, error) rather than error alone, so the CLI layer (Plan 04-08) can print the exact human-readable outcome (\"removed stale lock (pid=%d)\" / \"no lock present...\") verbatim without re-deriving it from error values."
  - "codegraphDirName/storeDirName/staleSidecarName are redeclared locally in internal/daemon rather than imported from internal/cli or internal/query — daemon sits below cli in the dependency direction (cli will depend on daemon in Plan 04-08, not vice versa), and this mirrors the exact cross-package-duplication precedent internal/query already set for internal/indexer.ShouldSkipDir (04-06-SUMMARY.md)."
  - "isStale is pid-liveness-only for v1 (os.FindProcess + Signal(0), POSIX); StartedAt is recorded in the lockfile but not yet cross-checked against the OS's own process-start-time. Documented as the accepted residual PID-reuse risk (T-04-07-02) rather than implemented, matching RESEARCH Pattern 6's explicit 'executor's discretion, document if skipped' guidance — isProcessLive is isolated in its own function specifically so a future corroboration pass (or a Windows OpenProcess-based variant) is a localized change."

patterns-established:
  - "Pattern: a package that composes another package's debounce/timer primitive needs that primitive exported — internal/watch's debouncer/newDebouncer/debounceDuration were unexported in Plan 04-05 (single-package use at the time); Plan 04-07 needed them from internal/daemon and exported them (Debouncer/NewDebouncer/DebounceDuration) rather than reimplementing debounce logic in internal/daemon."

requirements-completed: [SYNC-04, SYNC-05]

coverage:
  - id: D1
    description: "A daemon acquires an exclusive .codegraph/ lockfile (pid + start timestamp) before owning the writer; a second acquire attempt while it holds the lock is rejected"
    requirement: "SYNC-04"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestDaemonSharedWriter"
        status: pass
      - kind: unit
        ref: "internal/daemon/lock_test.go#TestAcquireRejectsLiveLock"
        status: pass
    human_judgment: false
  - id: D2
    description: "unlock removes a lock ONLY when it is genuinely stale (dead pid); it refuses to clear a live daemon's lock, and no-ops on an absent lock"
    requirement: "SYNC-05"
    verification:
      - kind: unit
        ref: "internal/daemon/lock_test.go#TestUnlockStaleOnly"
        status: pass
      - kind: unit
        ref: "internal/daemon/lock_test.go#TestUnlockAbsentIsNoOp"
        status: pass
    human_judgment: false
  - id: D3
    description: "The daemon owns exactly one indexer.Sync writer path: editing a watched file drives a debounced Sync that updates the committed graph"
    requirement: "SYNC-04"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestDaemonSharedWriter"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every goroutine the daemon spawns is context-owned and joined before Run returns (no leak) — clean shutdown releases the lock and Run is safely re-runnable afterward"
    requirement: "SYNC-04"
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestDaemonCleanShutdown"
        status: pass
    human_judgment: false
  - id: D5
    description: "Whole-repo build/vet/race-test suite and the archtest Pebble-import boundary stay green after internal/daemon's introduction"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && go test ./... -race -count=1"
        status: pass
      - kind: unit
        ref: "internal/graphstore/archtest#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

# Metrics
duration: 22min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 7: internal/daemon — Shared Watch/Index Server + Lockfile Summary

**A pidfile-guarded `internal/daemon.Daemon` that owns one `watch.Watcher` and drives every debounced flush through `indexer.Sync` — the single coordinated writer multiple agent sessions share — plus `Unlock`'s stale-only lockfile removal engine, both on a context+`sync.WaitGroup` shutdown discipline that leaves no goroutine or lock behind.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-07-11T20:36:00Z
- **Completed:** 2026-07-11T20:58:18Z
- **Tasks:** 2
- **Files modified:** 8 (4 created under `internal/daemon/`, 4 modified under `internal/watch/`)

## Accomplishments
- `internal/daemon.lockInfo{PID, StartedAt}` (JSON) + `isStale`/`isProcessLive` (`os.FindProcess` + `Signal(0)`, POSIX v1) — the pid-liveness check isolated in its own function for a future Windows variant
- `acquire` rejects acquisition with `ErrLockLive` when a live lock already exists, but self-heals a stale one (a crashed daemon's lock doesn't block a new daemon from starting)
- `Unlock(codegraphDir) (string, error)` — the `codegraph unlock` engine (Plan 04-08's CLI wraps it): absent lock is a clean no-op, a stale lock is removed with a confirmation message, a live lock is refused via `ErrLockLive` and left untouched (T-04-07-01)
- `Daemon.New(repoRoot)` resolves `.codegraph/` layout (`ErrNotInitialized` if absent); `Daemon.Run(ctx)` acquires the lock, opens a `watch.Watcher` over the repo, and drives the debounced flush through `indexer.Sync` — exactly one writer at a time, enforced both across processes (the lockfile) and within one process (a `syncMu` mutex independent of debounce timing)
- Every debounced flush touches `.codegraph/.sync-pending` before syncing and clears it only on a successful commit (D-04a) — the write side of the staleness signal Plan 04-06 already wired into `status`/`explore`
- `Run` blocks on `ctx.Done()` then `wg.Wait()`s the spawned watcher goroutine before releasing the lock and returning — no goroutine outlives `Run` (D-07); a `Daemon` is safely re-runnable after a clean shutdown
- Exported `internal/watch.Debouncer`/`NewDebouncer`/`DebounceDuration` (mechanical rename from the Plan-04-05-committed unexported `debouncer`/`newDebouncer`/`debounceDuration`) so `internal/daemon` — a different package — can actually construct and drive one

## Task Commits

Each task was committed atomically:

1. **Task 1: lockfile format + stale detection + unlock-stale-only** - `8a11039` (feat)
2. **Task 2: daemon lifecycle — single-writer Sync loop, context+WaitGroup shutdown** - `c1e397d` (feat)

**Plan metadata:** (this commit) - `docs(04-07): complete plan`

## Files Created/Modified
- `internal/daemon/lock.go` - `lockInfo`, `isStale`/`isProcessLive`, `acquire`/`release`, `Unlock`, `ErrLockLive`
- `internal/daemon/lock_test.go` - `TestUnlockStaleOnly`, `TestUnlockAbsentIsNoOp`, `TestAcquireRejectsLiveLock`, `TestAcquireClearsStaleLock`, `TestIsStale`
- `internal/daemon/daemon.go` - `Daemon`, `New`, `Run`, `flush`/`touchPending`/`clearPending`, `ErrNotInitialized`
- `internal/daemon/daemon_test.go` - `TestDaemonSharedWriter`, `TestDaemonCleanShutdown`, fixture/lock-poll/node-inspection helpers
- `internal/watch/debounce.go` - `debouncer`→`Debouncer`, `newDebouncer`→`NewDebouncer`, `debounceDuration`→`DebounceDuration` (exported, mechanical rename)
- `internal/watch/watcher.go` - `Watcher.Run`/`watchLoop` signatures updated to `*Debouncer`
- `internal/watch/debounce_test.go`, `internal/watch/watcher_test.go` - updated to the exported names (in-package tests, no behavior change)

## Decisions Made
- `indexer.Sync` owns its own `graphstore.Open`/`Writer`/`Commit`/`Close` lifecycle internally (verified against Plan 04-03's implementation), so `Daemon` never holds a `graphstore.GraphStore`/`Writer` field itself — the single-writer invariant is enforced at the process level (the lockfile) and, defensively, within one process (a `syncMu` mutex guarding against a slow sync overlapping the next debounced flush). `internal/daemon` therefore imports no `internal/graphstore`/Pebble at all, trivially satisfying the archtest boundary and the `.claude/CLAUDE.md` "internal/daemon imports no Pebble directly" instruction.
- `Unlock` returns `(message string, error)` rather than `error` alone, so Plan 04-08's thin CLI layer can print the exact confirmation text ("removed stale lock (pid=%d)" / "no lock present...") without re-deriving it.
- `codegraphDirName`/`storeDirName`/`staleSidecarName` are redeclared locally in `internal/daemon` rather than imported from `internal/cli`/`internal/query` — `daemon` sits below `cli` in the dependency direction (Plan 04-08's `codegraph daemon` command will depend on `daemon`, not vice versa), matching the cross-package-duplication precedent `internal/query` already established for `internal/indexer.ShouldSkipDir`.
- `isStale`/`isProcessLive` implement pid-liveness only for v1 (no `StartedAt` cross-check against the OS's own process-start-time) — the residual PID-reuse risk (T-04-07-02) is documented, not eliminated, matching RESEARCH Pattern 6's "executor's discretion; document if skipped" guidance. `isProcessLive` is isolated in its own function so a future corroboration pass or a Windows `OpenProcess`-based variant is a localized change.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `internal/watch`'s debounce primitive was unexported, making it uncallable from `internal/daemon`**
- **Found during:** Task 2 (constructing the daemon's debounced flush loop)
- **Issue:** Plan 04-05 (already committed) implemented `debouncer`/`newDebouncer`/`debounceDuration` and `Watcher.Run(ctx, deb *debouncer)` all lowercase/unexported — correct for Plan 04-05's own in-package tests, but 04-05-SUMMARY.md's own "Next Phase Readiness" section explicitly said the daemon would call `newDebouncer(ctx, debounceDuration(), flushFn)`. As written, `internal/daemon` (a different package) has no way to name or construct the unexported `*debouncer` type at all — `go build` fails outright, not a runtime bug.
- **Fix:** Mechanically renamed `debouncer`→`Debouncer`, `newDebouncer`→`NewDebouncer`, `debounceDuration`→`DebounceDuration` across `internal/watch/debounce.go`, `watcher.go`, and both existing in-package test files (`debounce_test.go`, `watcher_test.go`). No behavior change — confirmed via `internal/watch`'s own test suite staying green.
- **Files modified:** `internal/watch/debounce.go`, `internal/watch/watcher.go`, `internal/watch/debounce_test.go`, `internal/watch/watcher_test.go`
- **Verification:** `go build ./... && go test ./... -race -count=1` all green; `internal/watch`'s own `TestWatcherRecursiveAdd`/`TestWatcherErrorsDrained`/`TestDebounceCoalescesBurst`/`TestDebounceEnvTunable`/`TestDebounceNoFlushAfterCancel` unaffected
- **Committed in:** `c1e397d` (Task 2 commit, alongside `daemon.go`'s first use of the now-exported names)

---

**Total deviations:** 1 auto-fixed (1 blocking — a visibility gap in a prior plan's committed code, not a design flaw in this plan)
**Impact on plan:** Necessary for Task 2 to compile at all. Scope contained entirely to `internal/watch`'s export surface; no behavior or test logic changed.

## Issues Encountered
None beyond the export-visibility deviation documented above.

## User Setup Required
None - no external service configuration required. `CODEGRAPH_DEBOUNCE_MS` (existing, Plan 04-05) remains the only tunable env var, reused by the daemon's debounce window.

## Next Phase Readiness
- `internal/daemon.New`/`Run` and `Unlock` are ready for Plan 04-08's thin `codegraph daemon`/`codegraph unlock` CLI commands — no further daemon-package plumbing anticipated; the CLI layer resolves paths/flags and delegates.
- Plan 04-08's in-process `serve` fallback can reuse `Daemon.Run` directly (same lockfile, same single-writer discipline) rather than re-implementing the watch→debounce→Sync loop — this was an explicit plan goal ("Expose the pieces the in-process serve fallback needs") and is satisfied by `Daemon`'s exported `New`/`Run` surface alone; no additional exports were needed.
- Plan 04-09's leak-free soak test can drive `Daemon.Run` through many watch→debounce→sync cycles directly; the `context`+`sync.WaitGroup` shutdown discipline this plan established (`Run` blocks on `wg.Wait()` before returning) is exactly the property that soak test verifies. No rework anticipated, though 04-09 may still choose to add `goleak` on top for a stronger guarantee (CONTEXT's "Claude's Discretion").
- The residual PID-reuse corroboration (`StartedAt` cross-checked against the OS's actual process-start-time, T-04-07-02) remains unimplemented — flagged here for any future hardening pass; not a blocker for 04-08/04-09.
- No CLI wiring (`sync.go`/`daemon.go`/`unlock.go` Cobra commands) was built in this plan — that remains Plan 04-08's scope per 04-CONTEXT.md.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED
All created files (internal/daemon/lock.go, lock_test.go, daemon.go, daemon_test.go, this SUMMARY.md) found on disk; both task commit hashes (8a11039, c1e397d) found in git log.
