---
phase: 03-watcher-on-mcp-default
reviewed: 2026-07-16T19:01:58Z
depth: deep
files_reviewed: 5
files_reviewed_list:
  - internal/cli/serve.go
  - internal/cli/serve_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/watch/debounce_test.go
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 3: Code Review Report (Round 6 — FINAL re-review, iteration 3 of --auto)

**Reviewed:** 2026-07-16T19:01:58Z
**Depth:** deep
**Files Reviewed:** 5
**Status:** clean

## Summary

Adversarial re-review of the iteration-2 delta (commits 0a4daac..b194d3d: WR-01's three lifecycle tests plus the `syncFn`/`onWatchOpen` seams, IN-01 comment honesty, IN-02 errors.As symmetry in serve, IN-03 full D-12 line pin, and the b194d3d sync.Once guard). Every question from the review mandate was traced through source and re-verified by execution: `go build ./...` green, `go vet` clean on all three touched packages, `go test -race -count=1 -p 1` green across `internal/daemon`, `internal/watch`, `internal/cli` (goleak TestMain gates included), and the new requeue test run 3× consecutively under `-race` without flaking.

**1. The two production seams (`syncFn`, `onWatchOpen`) — clean.** Both are unexported fields with no exported setter and no Option, matching the established `onSync`/`onSyncStart` convention. Verified by symbol search: outside test files, the only references are the field declarations and their nil-guarded use sites in `daemon.go`. With `syncFn` nil, `flush` executes `syncFn := indexer.Sync` and calls it with the identical three arguments the old direct call passed — byte-equivalent behavior. With `onWatchOpen` nil, Run skips the call entirely; its placement (after `watch.Open`, before the watcher goroutine and before any Debouncer exists) cannot perturb lifecycle ordering. serve.go's `errors.As` case is the only deliberate production-behavior change in the delta (see 3). No production path is accidentally altered.

**2. The three new tests — deterministic and pinning what they claim.**
- `TestDaemonFlushLockRequeueGivesUpPerEpisode` (daemon_test.go:327): all synchronization is event-driven through the `onSync` channel; no sleep-based ordering. The arithmetic pin is real: re-derived against the wrapper (daemon.go:261-280), one organic event under injected contention yields fires at n=1..5 (each requeues) and n=6 (give-up, no requeue) — exactly `maxFlushLockRequeues+1 = 6` attempts, which the test counts exactly, then a 1s negative gap proves the give-up branch never requeues (flake-proof in the pass direction: `.codegraph/` is excluded from the watch set via `indexer.ShouldSkipDir`, so `touchPending`'s sidecar writes generate no events and nothing legitimate can fire). The per-episode reset is genuinely pinned: without the give-up reset, episode 2's first attempt would land at n=7 and give up after one attempt, timing out the test's wait for attempt 2. The theoretical fail-direction flake (a straggler fsnotify event for the episode's single WriteFile arriving after the whole ~6-window chain) requires the kernel to split CREATE/WRITE delivery of one write across 60ms+ while six timers fire on schedule — the events sit adjacent in the same queue, so a delay shifts the whole chain rather than splitting it; mid-chain stragglers coalesce harmlessly into the pending requeue. 3× `-race` reruns green.
- **The syncFn-seam adaptation is sound.** The wrapper branches on `errors.Is(flushErr, graphstore.ErrStoreLocked)` over flush's returned error; the seam returns `fmt.Errorf("%w: injected contention", graphstore.ErrStoreLocked)` — the same wrapped-sentinel shape `classifyOpenError` produces (pebble_store.go:124), and `indexer.Sync` propagates `graphstore.Open`'s error verbatim (`return Stats{}, err`, sync.go:52-54), so the seam-injected error is indistinguishable from the real one at the branch point and everything downstream of flush's return (log, sidecar left in place, onSync, requeue/give-up) is the genuine production path. The one link the seam bypasses — the bare `return err` inside Sync — is exercised with real Pebble LOCK contention by the integration live-sync test's deliberately-collidable burst window (on unfixed requeue behavior its poll deadline fails), and the fixer's documented reason for the adaptation checks out: failed `pebble.Open`s do leak disk-health ticker goroutines that would trip `internal/daemon`'s goleak TestMain (soak_test.go:21).
- `TestRunReturnsErrWatcherClosedAndReleasesLock` (daemon_test.go:418): fully channel-driven. `w.Close()` (idempotent via the atomic swap in watcher.go:123-128, so Run's deferred second Close is a safe no-op) closes fsnotify's channels; watchLoop returns on `ok=false`; `loopExited` fires with a live Background ctx, deterministically selecting `runErr = ErrWatcherClosed`. Pins all three claims: the sentinel via `errors.Is`, lock release through the same deferred `release()` (asserted by `readLock` absence AFTER RunWithRetry returns — ordering sound, the defer completes before Run returns), and immediate surfacing (onDeferred `t.Error`s; ErrLockLive is unreachable with no second holder).
- `TestDebounceAddAfterCancelIsNoOp` (debounce_test.go:141): zero timing dependence for the load-bearing assertions — `cancel()` synchronously sets `ctx.Err()` before returning, and the structural check (timer nil, pending empty, under `d.mu` from the same package) plus the immediate-`Wait` check pin exactly IN-04's gate ordering (`ctx.Err()` return precedes `fireWG.Add(1)`, debounce.go:71-74). The 75ms behavioral belt can only add latency, never flake: no timer was armed, so nothing can fire.

**3. serve.go errors.As change — clean.** The case is now `case errors.As(runErr, &de):` (serve.go:132), symmetric with cli/daemon.go; a bare or `%w`-wrapped `ErrWatchDisabled` sentinel from a hypothetical future producer falls to `default:`, rendering the error via `%v` — `DisabledError.Error()` embeds the reason (`policy.go:30-32`), so the IN-02 malformed empty-reason line is unreachable by construction. Case ordering is safe: `runErr == nil` matches before `errors.As` is ever evaluated. The D-12 string is byte-untouched by the delta (diff-verified: only the reason-extraction plumbing changed), and the new full-line pin asserts against production output, not a replica: `TestServeWatchStartDisabledPrintsVerbatimMessage` drives the real `serveWatchStart` (noWatch=true → the real `daemon.Run` policy gate → the real Fprintf) and matches its actual stderr buffer against the complete literal including the `[CodeGraph MCP] ` banner and trailing `\n` — character-by-character identical to the production format string with the pinned reason substituted. The test is also cancellation-order-safe: Run's policy check is its first action, before any ctx check, so the immediate `cancel()` cannot suppress the message.

**4. The sync.Once guard — correct minimal handling, not a mask.** The double-fire it absorbs is legitimate Debouncer behavior (two event bursts separated by more than a window are, by contract, two flushes), and the test's invariants are unweakened by a second parked fire: both fires hold `fireWG` counts, so the "Run must still be blocked" and "lock must still be held" assertions get strictly stronger, and `deb.Wait()` joins both before Run returns and before the final lock check. A straggler fire after `Wait()` returns is impossible by construction: `Wait` returning means `fireWG` hit zero, a new fire requires a new armed timer, post-cancel `Add`s are no-ops (the IN-04 gate), and a pre-cancel-armed timer that fires post-cancel exits at fire's own ctx check before reaching `onSyncStart`. The alternative (asserting exactly one fire) would over-constrain correct production behavior. The guard is the right shape.

**5. Nothing else fresh in the delta survives scrutiny.** The IN-01 comment additions (Run's doc parenthetical at daemon.go:209-212 and the backstop block at daemon.go:312-323) accurately describe the bounded loopExited-with-live-ctx requeue window — cross-checked against the actual gates, both genuinely ctx-only. No import, dependency, or non-scope file changed (5-file diff stat confirmed; go.mod untouched).

**Adjudicated residuals honored (not re-reported):** the bounded ~400ms lock-retry window; the windows sharing-violation imprecision; the hasIndex startup snapshot; the bounded loopExited-with-live-ctx requeue window (comment-only outcome accepted, and the comment now matches the mechanism).

All reviewed files meet quality standards. No issues found.

## Narrative Findings (AI reviewer)

None. Zero Critical, zero Warning, zero Info.

---

_Reviewed: 2026-07-16T19:01:58Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
