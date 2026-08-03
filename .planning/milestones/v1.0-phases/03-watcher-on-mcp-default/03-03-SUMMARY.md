---
phase: 03-watcher-on-mcp-default
plan: 03
subsystem: infra
tags: [watcher, mcp, cobra, cli, goroutine, tdd]

# Dependency graph
requires:
  - phase: 03-watcher-on-mcp-default
    provides: "internal/watch/policy.go — WatchDisabledReason(projectRoot, Probe), watch.ErrWatchDisabled (03-01)"
  - phase: 03-watcher-on-mcp-default
    provides: "daemon.RunWithRetry(ctx, d, interval, onDeferred), daemon.WithProbe(watch.Probe) (03-02)"
provides:
  - "serve --mcp watches by default whenever an index exists; --no-watch (new, opt-out) / --watch (repurposed, force-on) via cobra MarkFlagsMutuallyExclusive"
  - "internal/cli/serve.go's serveWatchStart(...) seam — all watcher startup (daemon.New, policy check, acquire, watch.Open) deferred into one goroutine, provably off the handshake path"
  - "verbatim TS D-12/D-13 disabled message printed to stderr on watch.ErrWatchDisabled, including for the --no-watch flag case (Pitfall 2)"
  - "TestServeWatchStartDeferred — mutation-proof structural seam test pinning WATCH-02, manually verified to turn red"
affects: [03-04, 03-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Test-only synchronization hook (onWatchWorkStart) mirroring daemon.Daemon's onSyncStart convention, for deterministic happens-before assertions on deferred goroutine startup"
    - "cobra MarkFlagsMutuallyExclusive for hard flag-contradiction errors instead of hand-rolled validation"

key-files:
  created: []
  modified: [internal/cli/serve.go, internal/cli/serve_test.go]

key-decisions:
  - "serveWatchStart re-derives watch.WatchDisabledReason for the disabled-message reason string (rather than string-parsing the wrapped error), since the same Probe is already in scope — cheaper and clearer than the plan's alternative"
  - "RunWithRetry's ctx.Canceled outcome (fired when RunE tears down mid-retry-sleep) is treated as a clean shutdown, not an error to print — d.Run's own ctx.Done() return path already returns nil, so treating RunWithRetry's ctx.Err() the same way keeps shutdown behavior consistent regardless of which phase of the retry loop RunE's teardown lands in"
  - "--watch's Go variable renamed watchMode -> forceWatch (D-03 the flag is now force-on semantics, not a watch/no-watch toggle) — flag name itself (\"--watch\") is unchanged for v0.1 invocation compatibility"

patterns-established:
  - "Manual mutation-proof verification: temporarily hoisting the deferred work (onWatchWorkStart + daemon.New) synchronously above the goroutine boundary, confirming the seam test turns red, then reverting before commit (not committed) — documented in this SUMMARY per the plan's acceptance criteria"

requirements-completed: [WATCH-01, WATCH-02]

coverage:
  - id: D1
    description: "serve --mcp watches by default whenever an index exists (no --watch flag required); --no-watch opts out; --watch is repurposed force-on; --no-watch --watch together is a hard cobra flag error"
    requirement: "WATCH-01"
    verification:
      - kind: unit
        ref: "internal/cli/serve_test.go#TestServeWatchStartDisabledPrintsVerbatimMessage"
        status: pass
      - kind: other
        ref: "grep -n 'no-watch\\|MarkFlagsMutuallyExclusive' internal/cli/serve.go"
        status: pass
    human_judgment: false
  - id: D2
    description: "All watcher startup (daemon.New, policy check, lock acquisition, watch.Open) is deferred into serveWatchStart's spawned goroutine, provably off the path to server.ServeStdio — proven by a deterministic, mutation-proof structural test"
    requirement: "WATCH-02"
    verification:
      - kind: unit
        ref: "internal/cli/serve_test.go#TestServeWatchStartDeferred"
        status: pass
      - kind: other
        ref: "manual mutation check: hoisting onWatchWorkStart+daemon.New above the goroutine boundary turned TestServeWatchStartDeferred red (reverted, not committed)"
        status: pass
    human_judgment: false
  - id: D3
    description: "serve --mcp --no-watch prints the verbatim TS-parity disabled message to stderr rather than silently skipping the watcher goroutine (Pitfall 2); reconcile indexer.Sync and CR-01's two-arg BuildServer call are byte-for-byte unchanged"
    verification:
      - kind: unit
        ref: "internal/cli/serve_test.go#TestServeWatchStartDisabledPrintsVerbatimMessage"
        status: pass
      - kind: other
        ref: "git diff --exit-code internal/agents/ && grep -F 'File watcher disabled —' internal/cli/serve.go"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-07-16
status: complete
---

# Phase 3 Plan 3: Watcher-on-MCP Default Flip Summary

**`serve --mcp` now watches by default whenever an index exists — `--no-watch` opts out, `--watch` is repurposed as explicit force-on, and every byte of watcher startup (`daemon.New`, the policy check, lock acquisition, `watch.Open`'s recursive walk) is deferred into one goroutine a new `serveWatchStart` seam spawns before returning, proven off the MCP handshake path by a mutation-proof structural test.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-16T10:09:17-04:00
- **Completed:** 2026-07-16T10:16:40-04:00
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `serve --mcp` watches live in-process by default whenever `.codegraph/` exists — no `--watch` flag required, restoring TS's zero-config auto-sync experience through the byte-identical `serve --mcp` invocation `install` already writes for all 8 agents
- `--no-watch` (new) opts out and `--watch` (repurposed, D-03) is the explicit force-on escape hatch; `cmd.MarkFlagsMutuallyExclusive("no-watch", "watch")` makes the contradictory combination a hard cobra flag error before RunE ever runs
- Extracted `serveWatchStart` (WR-01/D-08 precedent): spawns exactly one goroutine and returns `(cancel, done)` immediately, before `daemon.New`, the watch-policy check (inside `daemon.Run`), lock acquisition, or `watch.Open`'s recursive fsnotify walk ever executes — the WATCH-02 guarantee, provable by reading `newServeCmd`'s `RunE` top-to-bottom
- The verbatim TS D-12/D-13 disabled message (`[CodeGraph MCP] File watcher disabled — {reason}. ...`) prints to stderr on `watch.ErrWatchDisabled`, for BOTH the `CODEGRAPH_NO_WATCH=1` env case and the `--no-watch` flag case (Pitfall 2: the flag does not silently swallow the message)
- `TestServeWatchStartDeferred` — a deterministic, mutation-proof structural test using a test-only `onWatchWorkStart` synchronization hook (no sleep/timeout race) — asserts `serveWatchStart` returns strictly before its goroutine's real work begins; manually verified to turn red when `daemon.New` is hoisted synchronously above the goroutine boundary (reverted before commit, not part of the shipped diff)
- The D-07 reconcile `indexer.Sync` block and the CR-01 two-arg `mcp.BuildServer(hasIndex, allowlist, repoPath, start)` call are byte-for-byte unchanged; `internal/agents/*.go` untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: Rework serve.go — flags, serveWatchStart seam, default-on goroutine, verbatim disabled message** - `75d6945` (feat)
2. **Task 2: Mutation-proof structural seam test (WATCH-02/D-08) in serve_test.go** - `a45fa7b` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/cli/serve.go` - `--no-watch` flag (new), `--watch` repurposed as force-on, `MarkFlagsMutuallyExclusive`, new `serveWatchStart` seam + `watchRetryInterval` const, RunE's watcher block replaced with an unconditional `serveWatchStart` call gated only by `hasIndex` inside the seam
- `internal/cli/serve_test.go` - `TestServeWatchStartDeferred` (mutation-proof structural seam test), `TestServeWatchStartDisabledPrintsVerbatimMessage` (Pitfall 2 regression guard)

## Decisions Made
- `serveWatchStart` re-derives `watch.WatchDisabledReason(repoPath, probe)` for the disabled-message reason string rather than parsing it out of the wrapped error's text — the same `Probe` value is already in scope at that point, so re-computing is cheaper and clearer than string surgery (the plan offered both as acceptable options).
- `RunWithRetry`'s `context.Canceled` outcome (returned when RunE's teardown cancels `watchCtx` while the goroutine is mid-retry-sleep) is treated as a clean shutdown, not an error worth printing — `daemon.Run`'s own `<-ctx.Done()` path already returns `nil` on cancellation while actively watching, so folding `RunWithRetry`'s `ctx.Err()` into the same "no message" branch keeps shutdown behavior consistent regardless of which phase of the retry loop teardown happens to land in.
- Renamed the `--watch` flag's backing Go variable from `watchMode` to `forceWatch` (D-03: the flag is now force-on semantics, not a watch/no-watch toggle) — the flag's CLI name (`--watch`) itself is unchanged, preserving v0.1 invocation compatibility.

## Deviations from Plan

None — plan executed exactly as written. The manual mutation-proof verification step (plan's own acceptance criterion) was performed as specified: `daemon.New` + the `onWatchWorkStart` signal were temporarily hoisted synchronously above the goroutine boundary, confirmed `TestServeWatchStartDeferred` turned red, then reverted before any commit (not part of the shipped diff — verified via `git diff --exit-code internal/cli/serve.go` against the Task 1 commit after reverting).

## Issues Encountered

`go test ./... -count=1` intermittently failed `internal/daemon`'s `TestConvergenceTwoSessions` (a Pebble MANIFEST-creation race under full-suite parallel disk contention) — this test and the package it lives in are untouched by this plan (`git diff --exit-code internal/daemon/` is clean on both of this plan's commits) and it's 03-02's territory. Verified it passes reliably in isolation (`go test ./internal/daemon/... -count=1` and a 3x rerun of the specific test, all green). Logged as an out-of-scope deferred item rather than fixed, per the executor's scope-boundary rule — see [deferred-items.md](./deferred-items.md).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- WATCH-01 and WATCH-02 are both complete: `serve --mcp` watches by default, opts out via `--no-watch`, force-on via `--watch`, and zero watcher code executes before `server.ServeStdio` by construction.
- 03-04/03-05 (TEST-04 subprocess integration harness) can now write real end-to-end cases against `serve --mcp`'s default-on behavior and the `CODEGRAPH_NO_WATCH=1` off-switch, per D-21 — the CLI-level plumbing this harness needs is in place.
- One pre-existing, out-of-scope flake (`internal/daemon.TestConvergenceTwoSessions` under full-suite parallel load) logged in [deferred-items.md](./deferred-items.md) for a future flake-fix pass — not a blocker for this phase.

---
*Phase: 03-watcher-on-mcp-default*
*Completed: 2026-07-16*

## Self-Check: PASSED

All created/modified files (`internal/cli/serve.go`, `internal/cli/serve_test.go`, `03-03-SUMMARY.md`, `deferred-items.md`) and both task commit hashes (`75d6945`, `a45fa7b`) verified present in the working tree and git history.
