---
phase: 03-watcher-on-mcp-default
fixed_at: 2026-07-16T16:45:00Z
review_path: .planning/phases/03-watcher-on-mcp-default/03-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report

**Fixed at:** 2026-07-16T16:45:00Z
**Source review:** .planning/phases/03-watcher-on-mcp-default/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (fix_scope: critical_warning — IN-01..IN-04 excluded)
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: Default-on in-process flush collides with every other Pebble open

**Files modified:** `internal/graphstore/pebble_store.go`, `internal/daemon/daemon.go`, `internal/cli/serve.go`
**Commit:** c22ad98
**Applied fix:** Three coordinated changes at the store-open seam, per the locked project decision (bounded retries + degrade, NO shared-process singleton):
1. **Bounded lock retry in `graphstore.Open`** (covers scenarios 1 & 3 and every open site uniformly — `query.OpenAt` per tool call, `indexer.Sync`'s write opens, the startup reconcile): a lock-held `pebble.Open` failure is retried up to 5 attempts with 100ms backoff (~400ms worst-case added wait, never an unbounded block) before the original error surfaces. New exported `graphstore.IsLockHeld(err)` detects both forms — pebble's in-process `"lock held by current process"` message and the cross-process `fcntl(F_SETLK)` EAGAIN/EWOULDBLOCK/EACCES errnos. Lives in graphstore, the sole pebble-aware package (D-04a preserved; archtest green).
2. **Bounded flush requeue in `daemon.Run`** (scenario 2 — a failed flush is no longer silently terminal): the debouncer callback is wrapped so a `Sync` that still lost the lock race after Open's in-call retries re-arms the debouncer via a sentinel `Add`, bounded by `maxFlushLockRequeues = 5` consecutive failures (counter resets on success; BL-01 lesson honored — a `ctx.Err() != nil` gate prevents post-cancel requeues, and every failure path leaves the `.sync-pending` sidecar in place, so nothing is ever marked clean under a cancelled ctx). `flush` now returns Sync's error to feed the wrapper.
3. **Serve startup-reconcile downgrade** (scenario 3 — session B's MCP server no longer dies at startup): a lock-held reconcile `Sync` failure in serve.go's RunE degrades to a stderr warning + continue (the holder is actively syncing the same store — the stale banner covers the seconds-stale gap); every non-lock error stays fatal.

Behaviorally verified, not just syntax-checked: the WR-04 integration test below fails on the pre-fix code with the exact collision signature (`lock held by current process` as a tool error, 2/2 runs) and passes 5/5 with the fix.

**Documented residual (per orchestrator guardrail):** retries are bounded by design. A genuinely long-lived lock holder — e.g. a monorepo-scale sync holding the store well past ~400ms while an explore lands — can still exhaust the retry budget and surface the lock error to that one tool call; the flush side is likewise bounded at 5 consecutive requeues before giving up until the next organic event (sidecar stays set, so staleness remains observable). Eliminating the residual entirely requires a shared in-process store handle/serialization point, which is a design change explicitly out of scope for this fix pass.

### WR-01: `codegraph daemon` gets the friendly D-12 disabled message path

**Files modified:** `internal/cli/daemon.go`
**Commit:** a527a71
**Applied fix:** `newDaemonCmd`'s RunE now `errors.Is(err, watch.ErrWatchDisabled)` on `d.Run(ctx)`'s return and prints the same verbatim D-12/D-13 stderr message serve.go prints (including the `codegraph sync` / git-hooks guidance), then exits cleanly (nil) — a policy-disabled watcher is a deliberate, explained state, not a failure. The reason is recomputed via `watch.WatchDisabledReason` on the absolutized start path (matching what Run's own gate saw). `daemon.Run` still returns the raw wrapped sentinel, so `errors.Is` detectability is preserved for programmatic callers. No new flags added (the review listed `--watch`/`--no-watch` symmetry as optional only).

### WR-02: `TestServeWatchStartDeferred` scheduling race replaced with deterministic design

**Files modified:** `internal/cli/serve_test.go`
**Commit:** 30b9839
**Applied fix:** Replaced the atomic-flag happens-before assertion (which the Go scheduler could legitimately fail on GOMAXPROCS>1) with the review's block-until-released channel design: the hook's first action blocks on a `released` channel the test only closes AFTER `serveWatchStart` returns (delivered via a helper goroutine + bounded `retCh` select, so the mutated case fails a 5s timeout instead of deadlocking the test binary). Mutation-proofness re-verified empirically: temporarily moving the hook call above the goroutine boundary turned the test red with the WATCH-02 message; correct code passes deterministically (`-race -count=5` green).

### WR-03: CI now runs the race detector on the concurrency packages

**Files modified:** `.github/workflows/ci.yml`
**Commit:** 921ea1f
**Applied fix:** Added a targeted CI step `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` (daemon + watch per the guardrail, plus internal/cli per the review's "ideally" clause since serveWatchStart's goroutine seams live there) rather than a global `-race` run. `-p 1` preserves the same no-parallel-package-contention isolation the existing internal/daemon step documents. Verified locally: the exact command is green; YAML parses.

### WR-04: Live edit → auto-sync → fresh explore integration test added

**Files modified:** `test/integration/watch_live_sync_test.go` (new file — required by the finding; documented per fixer contract)
**Commit:** 1a1b07b
**Applied fix:** `TestLiveEditAutoSyncReachesExplore` rides the existing substrate (TestMain's binPath, copyFixture, runBinary — no second TestMain, no new dependency, one local `newServeClientWithEnv` helper for the `CODEGRAPH_DEBOUNCE_MS=100` env): it spawns a default-on `serve --mcp` subprocess on an indexed fixture, writes a 26-file burst of new symbols into the watched tree mid-session (the burst widens the sync's LOCK-held window so the CR-01 race is reliably exercised, and stays in the added set until the first sync succeeds), then streams `codegraph_explore` calls on a tight 10ms cadence until the new symbol's verbatim source appears — failing immediately on any transport error or IsError result. Re-touch cadence (500ms) deliberately exceeds the debounce window so the test can never starve its own flush. Verified both directions per the guardrail: fails on unfixed CR-01 (2/2 runs, lock-error signature), passes 5/5 with the fix (~1s each; 90s deadline for CI headroom).

## Verification

All gates run in the fix worktree after the final commit:
- `go build ./...` — green
- `go test ./...` — green (no failures; includes internal/daemon)
- `go test ./testdata/golden/...` — green
- `go test ./test/integration/... -count=1` — green
- `go test -race ./internal/daemon/... ./internal/watch/...` — green
- `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` (the new CI step, exact command) — green
- Zero new dependencies; verbatim TS strings untouched; `internal/agents/` untouched; archtest (D-04a pebble confinement) green.

---

_Fixed: 2026-07-16T16:45:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
