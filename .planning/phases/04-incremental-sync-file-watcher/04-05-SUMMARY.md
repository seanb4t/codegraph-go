---
phase: 04-incremental-sync-file-watcher
plan: 05
subsystem: infra
tags: [fsnotify, filesystem-watcher, debounce, go, sync-01]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/indexer.ShouldSkipDir (exported dir-skip predicate) and Sync(repoRoot, storeDir, opts) from Plan 04-03"
provides:
  - "internal/watch package: native fsnotify-backed recursive filesystem watcher"
  - "internal/watch.debouncer: env-tunable burst-coalescing debounce (CODEGRAPH_DEBOUNCE_MS)"
  - "context+WaitGroup-compatible lifecycle primitives (Watcher.Close idempotent, debouncer.Stop) for Plan 04-09's leak-free soak"
affects: [04-06, 04-07, 04-08, 04-09]

# Tech tracking
tech-stack:
  added: ["github.com/fsnotify/fsnotify v1.10.1"]
  patterns:
    - "recursive-add-on-Create: fsnotify does not recurse, so a directory Create event re-runs addRecursive over the new subtree"
    - "Errors-channel draining in the same select loop as Events (never range Events alone)"
    - "idempotent Close via atomic.Bool swap, borrowed from internal/graphstore/pebble_store.go"
    - "debounce timer-reset coalescing (time.AfterFunc) guarded by ctx.Err() check + explicit Stop() for cancellation-clean shutdown"

key-files:
  created:
    - internal/watch/watcher.go
    - internal/watch/watcher_test.go
    - internal/watch/debounce.go
    - internal/watch/debounce_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "debounce.go was written and committed alongside watcher.go (Task 1) rather than deferred to Task 2, since watcher.go's Run(ctx, *debouncer) signature requires the debouncer type to compile — debounce-specific tests (TestDebounce*) still landed in Task 2 as planned"
  - "fsnotify promoted to go.mod's direct require block by manual edit (not go mod tidy), per the established project convention (STATE.md Phase 1 decisions) that a broad tidy would strip other pre-pinned-but-unimported deps"
  - "TestWatcherErrorsDrained induces a synthetic error by sending directly into fsnotify.Watcher's exported (non-directional) Errors channel — a portable, OS-independent way to exercise the Errors-drain branch without fabricating a genuine platform-specific watch failure"

patterns-established:
  - "Pattern: internal/watch depends only on internal/indexer.ShouldSkipDir — no graphstore/pebble import, keeping the D-04a archtest boundary clean"

requirements-completed: [SYNC-01]

coverage:
  - id: D1
    description: "Native fsnotify watcher recursively covers a repo tree, re-adding newly-created subdirectories on Create events"
    requirement: "SYNC-01"
    verification:
      - kind: unit
        ref: "internal/watch/watcher_test.go#TestWatcherRecursiveAdd"
        status: pass
    human_judgment: false
  - id: D2
    description: "The watcher's Errors channel is drained in the same select loop as Events, so an internal error never stalls subsequent event processing"
    requirement: "SYNC-01"
    verification:
      - kind: unit
        ref: "internal/watch/watcher_test.go#TestWatcherErrorsDrained"
        status: pass
    human_judgment: false
  - id: D3
    description: "A burst of edits within the debounce window coalesces into exactly one flush over the deduplicated union of changed paths"
    requirement: "SYNC-01"
    verification:
      - kind: unit
        ref: "internal/watch/debounce_test.go#TestDebounceCoalescesBurst"
        status: pass
    human_judgment: false
  - id: D4
    description: "Debounce window defaults to 2000ms and is tunable via CODEGRAPH_DEBOUNCE_MS, falling back to the default on zero/negative/non-numeric values"
    requirement: "SYNC-01"
    verification:
      - kind: unit
        ref: "internal/watch/debounce_test.go#TestDebounceEnvTunable"
        status: pass
    human_judgment: false
  - id: D5
    description: "No flush fires after context cancellation — both the AfterFunc callback's ctx.Err() check and explicit timer.Stop() are cancellation-clean"
    requirement: "SYNC-01"
    verification:
      - kind: unit
        ref: "internal/watch/debounce_test.go#TestDebounceNoFlushAfterCancel"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 5: Native fsnotify Watcher + Env-Tunable Debounce Summary

**A recursive-add fsnotify watcher (`internal/watch.Open`/`Run`) feeding a coalescing debouncer (`CODEGRAPH_DEBOUNCE_MS`, default 2000ms) that flushes a burst of edits into one call over the union of changed paths — the write path's front door for SYNC-01.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-11T19:41:25Z
- **Completed:** 2026-07-11T19:47:00Z
- **Tasks:** 2
- **Files modified:** 6 (4 created under `internal/watch/`, `go.mod`/`go.sum` updated)

## Accomplishments
- `internal/watch.Open(root)` walks the tree at startup, watching every directory `indexer.ShouldSkipDir` does not exclude (so `.codegraph/`, `vendor/`, and dot-prefixed directories are never watched — agreeing exactly with the indexer's own discovery exclusions)
- `Watcher.Run`'s `watchLoop` selects on both `Events` and `Errors` in one loop; a `Create` event on a new directory re-runs `addRecursive` (fsnotify does not recurse); an internal error is logged, never fatal, so the loop keeps servicing subsequent events
- `Watcher.Close` is idempotent via an `atomic.Bool` swap, mirroring `pebbleStore.Close`'s idiom
- `debouncer` coalesces a burst of `Add(path)` calls within the window into one `flush` call over the deduplicated union of paths; a quiet gap longer than the window flushes and resets, so a subsequent burst flushes again
- `debounceDuration()` reads `CODEGRAPH_DEBOUNCE_MS` (positive integer milliseconds), falling back to a 2000ms default for a missing/zero/negative/non-numeric value
- The debounce timer's `AfterFunc` callback checks `ctx.Err()` before flushing, and `Stop()` cancels the pending timer explicitly — the two-part cancellation-clean guarantee Plan 04-09's leak-free soak gate depends on

## Task Commits

Each task was committed atomically:

1. **Task 1: fsnotify watcher — recursive add-on-Create, Errors draining, idempotent Close** - `a79261c` (feat)
2. **Task 2: debounce — coalesce a burst into one flush over the union of paths** - `f789578` (feat)

**Plan metadata:** (this commit) - `docs(04-05): complete plan`

## Files Created/Modified
- `internal/watch/watcher.go` - `Open`/`addRecursive`/`Run`/`watchLoop`/`Close`: the recursive fsnotify wrapper
- `internal/watch/watcher_test.go` - `TestWatcherRecursiveAdd`, `TestWatcherErrorsDrained`
- `internal/watch/debounce.go` - `debounceDuration`/`debouncer` (`Add`/`fire`/`Stop`): the burst-coalescing timer
- `internal/watch/debounce_test.go` - `TestDebounceCoalescesBurst`, `TestDebounceEnvTunable`, `TestDebounceNoFlushAfterCancel`
- `go.mod` / `go.sum` - `github.com/fsnotify/fsnotify v1.10.1` added as a direct dependency

## Decisions Made
- Wrote `debounce.go` in Task 1 (not deferred to Task 2) because `Watcher.Run(ctx, *debouncer)` requires the type to compile; debounce-specific behavior tests still landed in Task 2 as the plan specified, keeping the RED/behavior-verification boundary where the plan intended it.
- Promoted `fsnotify` to `go.mod`'s direct require block by manual edit rather than `go mod tidy`, per the project's established Phase 1 convention (a broad tidy would strip other deliberately pre-pinned-but-unimported dependencies like `pebble/v2` alternates, `wazero`, etc. — this plan only touched the two require lines fsnotify needed).
- `TestWatcherErrorsDrained` sends a synthetic error directly into `fsnotify.Watcher.Errors` (an exported, non-directional `chan error` field) instead of attempting to fabricate a genuine OS-level watch failure — this is portable across platforms and exercises the exact same `watchLoop` branch a real error would.

## Deviations from Plan

None — plan executed exactly as written, modulo the file-grouping note above (debounce.go landing in Task 1's commit as a compile dependency, which the plan's own interface contract implied).

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. `CODEGRAPH_DEBOUNCE_MS` is an optional env var consumers may set; no action required to use the 2000ms default.

## Next Phase Readiness
- `internal/watch` is ready to be wired into the daemon (Plan 04-07/04-08) and `serve`'s in-process fallback: `Watcher.Run(ctx, deb)` plus a `newDebouncer(ctx, debounceDuration(), flushFn)` where `flushFn` calls `indexer.Sync(repoRoot, storeDir, opts)` over the debounced path set.
- Lifecycle primitives (`Watcher.Close`, `debouncer.Stop`, the `ctx.Err()` guard in `fire`) are already shaped for Plan 04-09's `context`+`sync.WaitGroup` join-on-shutdown discipline and `goleak`-based soak test — no rework anticipated.
- No blockers.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED
All created files found on disk; both task commit hashes (a79261c, f789578) found in git log.
