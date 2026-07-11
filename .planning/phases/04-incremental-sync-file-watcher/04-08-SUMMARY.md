---
phase: 04-incremental-sync-file-watcher
plan: 08
subsystem: cli
tags: [cobra, cli, daemon, watcher, go, sync-04, sync-05, indx-03]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/indexer.Sync(repoRoot, storeDir, opts) from Plan 04-03 — the incremental entry `codegraph sync` calls"
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/watch package from Plan 04-05 — the recursive fsnotify watcher `serve --watch`'s in-process fallback composes via internal/daemon"
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/daemon.Daemon (New/Run), Unlock, ErrLockLive from Plan 04-07 — the engine `codegraph daemon`/`unlock` and serve's fallback delegate to"
provides:
  - "codegraph sync — thin Cobra command delegating to indexer.Sync (INDX-03), same ErrNotInitialized guard and --quiet/--verbose flags as index"
  - "codegraph daemon — thin Cobra command delegating to daemon.Run(ctx), blocking until SIGINT/SIGTERM"
  - "codegraph unlock — thin Cobra command delegating to daemon.Unlock (SYNC-05), stale-only, no confirm prompt"
  - "serve --watch — in-process watcher fallback sharing the daemon lockfile, mutually exclusive with a standalone daemon"
affects: [04-09]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "printSyncSummary wraps printSummary rather than forking a second summary printer — sync's extra Stats fields (reparsed/pruned/nodesRemoved/edgesRemoved/dependentsRecomputed) print as an additive second line"
    - "in-process watcher fallback: goroutine running daemon.Run(ctx) started before ServeStdio, cancelled+joined via a deferred cancel+channel-receive on serve exit — the exact context+WaitGroup-style join discipline internal/daemon itself uses"
    - "ErrLockLive from the in-process watcher is a graceful defer-to-the-existing-daemon (stderr notice), not a serve failure — the shared lockfile is the single-writer enforcement mechanism, not an error condition to propagate"

key-files:
  created:
    - internal/cli/sync.go
    - internal/cli/sync_test.go
    - internal/cli/daemon.go
    - internal/cli/unlock.go
  modified:
    - internal/cli/root.go
    - internal/cli/serve.go

key-decisions:
  - "root.go registers newSyncCmd in Task 1's commit and newDaemonCmd/newUnlockCmd in Task 2's commit (not all three at once) so each task's own `go build ./...` verification step is independently green — daemon.go/unlock.go didn't exist yet when Task 1 committed"
  - "unlock.go performs no absent-lockfile guard of its own: daemon.Unlock already returns a clean human-readable no-op message (nil error) when no lockfile exists, so the CLI stays a pure delegate rather than duplicating that check"
  - "serve --watch only starts the in-process watcher when hasIndex is true — an uninitialized repo has no .codegraph/ for daemon.New to resolve, and MCP-03's absent-index behavior (server starts anyway, advertising zero tools) must stay unaffected by the flag"

requirements-completed: [INDX-03, SYNC-04, SYNC-05]

coverage:
  - id: D1
    description: "codegraph sync incrementally updates the graph on an initialized repo and errors clearly (ErrNotInitialized) on an uninitialized one"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/cli/sync_test.go#TestSyncCmdErrorsWhenUninitialized"
        status: pass
      - kind: unit
        ref: "internal/cli/sync_test.go#TestSyncCmdUpdatesGraph"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph daemon starts the shared watch/index server, blocking until signalled, and releases its lockfile cleanly on shutdown"
    requirement: "SYNC-04"
    verification:
      - kind: manual_procedural
        ref: "built binary: `codegraph daemon --path <dir> &` then SIGTERM; verified daemon.lock removed after exit"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph unlock clears a genuinely stale lock, refuses a live one (ErrLockLive), and no-ops cleanly on an absent lock"
    requirement: "SYNC-05"
    verification:
      - kind: manual_procedural
        ref: "built binary: unlock against a live daemon (refused), a kill -9'd stale lock (removed), and an absent lock (no-op message)"
        status: pass
    human_judgment: false
  - id: D4
    description: "serve --watch runs an in-process watcher fallback under the same lockfile a standalone daemon uses; reconcile + MCP-03 absent-index behavior preserved"
    requirement: "SYNC-04"
    verification:
      - kind: other
        ref: "go build ./... && go test ./internal/cli/... -race -count=1"
        status: pass
    human_judgment: true
    rationale: "The mutual-exclusion behavior (a live standalone daemon causes serve --watch to defer via ErrLockLive rather than crash) is implemented per the shared-lockfile pattern internal/daemon already tests, but no dedicated automated test exercises serve --watch's own goroutine lifecycle end-to-end against a stdio MCP session — a human should confirm the flag's runtime behavior once."

# Metrics
duration: 6min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 8: codegraph sync/daemon/unlock Commands + serve --watch Fallback Summary

**Three thin Cobra commands (`sync`, `daemon`, `unlock`) delegating to the already-built `internal/indexer.Sync`/`internal/daemon` engine packages, plus a `serve --watch` in-process watcher fallback sharing the daemon's own lockfile — completing the Phase-4 user-facing command surface (INDX-03/SYNC-04/SYNC-05).**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-11T21:02:21Z
- **Completed:** 2026-07-11T21:08:31Z
- **Tasks:** 2
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments
- `codegraph sync` (`internal/cli/sync.go`): thin delegation to `indexer.Sync`, mirroring `newIndexCmd`'s `ErrNotInitialized` guard and `--quiet`/`--verbose` flags exactly; `printSyncSummary` wraps `printSummary` with an additive line reporting reparsed/pruned/nodesRemoved/edgesRemoved/dependentsRecomputed
- `codegraph daemon` (`internal/cli/daemon.go`): mirrors `newServeCmd`'s `-p/--path` resolution, blocks on `daemon.Run(ctx)` with `ctx` cancelled by `signal.NotifyContext` on SIGINT/SIGTERM
- `codegraph unlock` (`internal/cli/unlock.go`): mirrors `newUninitCmd`'s `targetRoot(args)` shape, delegates the entire stale-vs-live decision to `daemon.Unlock` — no confirm prompt, since unlock only ever removes a genuinely stale lock
- `serve --watch` (`internal/cli/serve.go`): an in-process watcher fallback — when set and the repo has an index, a goroutine runs `daemon.Run(ctx)` under the exact same lockfile a standalone `codegraph daemon` would use, started before `ServeStdio` and cancelled+joined via a deferred cleanup on serve exit; a live standalone daemon's `ErrLockLive` is a graceful stderr notice, not a serve failure
- `root.go` registers all three new commands (`newSyncCmd` in Task 1, `newDaemonCmd`/`newUnlockCmd` in Task 2) and its package doc now names sync/daemon/unlock
- `internal/cli/sync_test.go`: `TestSyncCmdErrorsWhenUninitialized` (no `.codegraph/` → `ErrNotInitialized`) and `TestSyncCmdUpdatesGraph` (init a fixture, edit `main.go`, run `sync`, assert the summary output and a grown node count)

## Task Commits

Each task was committed atomically:

1. **Task 1: codegraph sync command + root wiring + summary** - `cc3f12b` (feat)
2. **Task 2: codegraph daemon + unlock commands + serve in-process fallback** - `02bf31b` (feat)

**Plan metadata:** (this commit) - `docs(04-08): complete plan`

## Files Created/Modified
- `internal/cli/sync.go` - `newSyncCmd`, `printSyncSummary`
- `internal/cli/sync_test.go` - `TestSyncCmdErrorsWhenUninitialized`, `TestSyncCmdUpdatesGraph`
- `internal/cli/daemon.go` - `newDaemonCmd`
- `internal/cli/unlock.go` - `newUnlockCmd`
- `internal/cli/root.go` - registers `newSyncCmd`/`newDaemonCmd`/`newUnlockCmd`; package doc names sync/daemon/unlock
- `internal/cli/serve.go` - `--watch` flag and the in-process watcher fallback goroutine (`daemon.New`/`Run` under `context.WithCancel`, deferred cancel+join)

## Decisions Made
- **`root.go`'s three new commands landed across two commits, not one** — `newSyncCmd` in Task 1, `newDaemonCmd`/`newUnlockCmd` in Task 2 — so Task 1's own `go build ./...` verification step passed independently before `daemon.go`/`unlock.go` existed (Task 1's action text says "register newSyncCmd," Task 2's says "register both... alongside newSyncCmd from Task 1").
- **`unlock.go` adds no absent-lockfile guard of its own.** `daemon.Unlock` (Plan 04-07) already returns a clean, human-readable no-op message with a nil error when no lockfile exists — duplicating that check in the CLI layer would violate the "no logic in RunE" thin-delegation discipline the objective calls for.
- **`serve --watch` gates on `hasIndex`.** An uninitialized repo has no `.codegraph/` for `daemon.New` to resolve; gating the in-process watcher on the same `hasIndex` flag `serve`'s reconcile step already checks keeps MCP-03's absent-index behavior (server starts anyway, zero tools advertised) completely unaffected by the new flag.
- **A live standalone daemon's `ErrLockLive` inside `serve --watch` is logged to stderr and treated as success**, not returned as a `RunE` error — the whole point of the shared lockfile (T-04-08-01) is that the two watchers are mutually exclusive by design; encountering that exclusion is expected behavior; the MCP server still starts normally, relying on the standalone daemon to keep the graph current.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- An initial manual smoke test of `codegraph daemon` via `go run ./cmd/codegraph daemon ... &` followed by `kill -TERM` left `daemon.lock` behind, appearing to indicate the lock wasn't released on shutdown. Re-testing against a compiled binary (`go build -o /tmp/codegraph-test ./cmd/codegraph`) showed a clean lock release — the leftover was `go run`'s wrapper process not propagating SIGTERM to its child, not a bug in `Daemon.Run`'s shutdown path. No code change was needed; documented here since it could otherwise look like an unresolved defect.

## User Setup Required
None - no external service configuration required. `CODEGRAPH_DEBOUNCE_MS` (existing, Plan 04-05) remains the only tunable env var, applying identically to `codegraph daemon` and `serve --watch`.

## Next Phase Readiness
- All three requirements this plan targets (INDX-03, SYNC-04, SYNC-05) are now fully surfaced to users: `codegraph sync`/`daemon`/`unlock`, plus `serve --watch`.
- Plan 04-09 (per 04-CONTEXT.md's Next Phase Readiness chain) can drive `Daemon.Run` through many watch→debounce→sync cycles for its leak-free soak test — no rework anticipated in `internal/daemon` from this plan, since `codegraph daemon`/`serve --watch` are pure callers of the exact `New`/`Run` surface Plan 04-07 already built and tested.
- `serve --watch`'s goroutine-lifecycle behavior against a live stdio MCP session has only been exercised via `go build`/`go test`/manual CLI smoke tests (`codegraph daemon`+`unlock` interplay), not an automated end-to-end MCP-session test — flagged as `human_judgment: true` in this SUMMARY's coverage block (D4) for the verifier to route to a human UAT pass if warranted.
- No blockers.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED
All created files (sync.go, sync_test.go, daemon.go, unlock.go, this SUMMARY.md) found on disk; both task commit hashes (cc3f12b, 02bf31b) found in git log.
