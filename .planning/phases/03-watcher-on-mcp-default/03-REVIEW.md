---
phase: 03-watcher-on-mcp-default
reviewed: 2026-07-16T14:57:32Z
depth: deep
files_reviewed: 11
files_reviewed_list:
  - .github/workflows/ci.yml
  - internal/cli/serve.go
  - internal/cli/serve_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/soak_test.go
  - internal/watch/policy.go
  - internal/watch/policy_test.go
  - test/integration/main_test.go
  - test/integration/watch_default_test.go
  - test/integration/worktree_notice_test.go
findings:
  critical: 1
  warning: 4
  info: 4
  total: 9
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-07-16T14:57:32Z
**Depth:** deep
**Files Reviewed:** 11
**Status:** issues_found

## Summary

Reviewed the watcher-on-MCP-default phase at deep depth, tracing the full cross-file lifecycle: `serve.go` RunE → `serveWatchStart` goroutine → `daemon.RunWithRetry` → `daemon.Run` → `watch.Open`/`Debouncer` → `indexer.Sync` → `graphstore.Open` (Pebble), plus the lockfile state machine (`lock.go`), the watch-policy port, and the subprocess integration harness.

**What holds up well (verified, not assumed):**

- **No CR-01 reintroduction.** `serve.go:216` passes `BuildServer(hasIndex, allowlist, repoPath, start)` with `start` captured before `repoPath` is overwritten; `TestServeKeepsStartPathDistinctFromConfinementRoot` pins the real `serveServerPaths`, and the D-20 integration anchor (`TestWorktreeNoticeReachesServeMCPExplore`) asserts the glyph against the **real subprocess payload** (glyph sourced from `gitmeta.Mismatch.Notice()` itself, with correct U+FE0F disambiguation in `containsBareNoticeGlyph`). The anchor is genuine, not a replica.
- **No Phase-2 BL-01 recurrence in `RunWithRetry`.** Nothing is cached or recorded under a cancelled ctx: the loop either returns `d.Run`'s result directly or returns `ctx.Err()` from the retry-sleep select. `jitter` is panic-safe (`spread <= 0` guard before `rand.Int63n`).
- **Lock released on every `Run` exit path**: policy-disabled returns before `acquire()`; `acquire()` failure has nothing to release; post-acquire paths release via defer, in the correct LIFO order (`deb.Wait()` → `w.Close()` → `release()`), and `deb.Wait()` genuinely joins an in-flight flush (pinned by `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`).
- **Watch policy precedence matches the TS contract**: NO_WATCH (flag OR strict `=="1"` env) beats FORCE_WATCH beats WSL2 auto-off; `/mnt/[a-z]` single-letter regex correctly excludes `/mnt/wsl`; unconditional backslash normalization matches TS's `normalizePath`. `DetectWSL` caching is `sync.Once`-guarded (thread-safe); the reset hook is test-only.
- **Teardown cannot hang `ServeStdio`'s return path**: every branch of the watcher goroutine ends in `close(watchDone)`; `RunWithRetry` unblocks on cancel from every state (mid-sleep via select, mid-watch via `<-ctx.Done()`); worst case the deferred `<-watchStartDone` waits out one in-flight `indexer.Sync`, which is the intended lock-safety property, not a hang.
- **Watcher failure never kills the MCP server**: all failure branches in `serveWatchStart`'s goroutine write to stderr and return; `log.Printf` in daemon/watch paths goes to stderr, never corrupting stdio JSON-RPC.
- **No zombie subprocesses in the harness**: `t.Cleanup(c.Close)` is registered before `Initialize`, and `exec.CommandContext` in the `WithCommandFunc` seam kills the child on transport-context cancellation. The stderr reader in `TestNoWatchEnvDisablesViaStderr` is mutex-guarded and started before `Initialize` — the race handling is correct.

**What does not hold up:** one Critical emergent interaction (the exact Phase-2 failure class (b) the phase context warned about) and four Warnings, detailed below.

## Critical Issues

### CR-01: Default-on in-process flush collides with every other Pebble open — tool calls fail mid-sync, failed syncs are never retried, and a second session's serve can die at startup

**File:** `internal/cli/serve.go:182-187, 196-199`; `internal/daemon/daemon.go:258-280`; (interacting, unchanged: `internal/query/engine.go:160-190`, `internal/indexer/sync.go:52-56`, `internal/graphstore/pebble_store.go:67-73`)

**Issue:** Every path into the graph store — the daemon flush (`indexer.Sync`), the startup reconcile `Sync` (serve.go:184), and every `codegraph_explore`/query (`query.OpenAt`) — calls `graphstore.Open`, which is `pebble.Open(dir, &pebble.Options{})`: a **read-write open holding Pebble's exclusive directory LOCK for the entire call**, failing immediately (non-blocking flock, plus Pebble's in-process lock tracking) when any other open of the same store is live, in the same process or another.

Before this phase that collision surface was opt-in (`--watch` / standalone `codegraph daemon`). This phase makes the flush **default-on in every `serve --mcp` session**, turning three collisions into default-path behavior:

1. **Same-process, the core agent workflow:** agent edits a file → 2s debounce → `flush` holds the store open for the full `indexer.Sync` duration → the agent's next `codegraph_explore` (which lands exactly then, because the agent just edited and now queries) hits `OpenAt` → `graphstore.Open` fails → the tool call returns an error. Nothing serializes these: `syncMu` (daemon.go:59) only serializes flushes against each other; query opens never touch it.
2. **Same-process, inverted:** an explore holds the store open when the debounce fires → the flush's `Sync` fails → `daemon: sync: ...` is logged and the `.sync-pending` sidecar stays — but **a failed flush is never retried**. The debouncer only fires on new `Add`s, so if the edit burst is over, the graph stays stale (sidecar set, content old) until an unrelated future event or the next session's reconcile.
3. **Cross-process, startup:** session B's synchronous reconcile `Sync` (serve.go:184, on the RunE path **before** `ServeStdio`) overlaps session A's in-flight flush (or explore) → `Sync` returns the Pebble lock error → `return err` → **session B's MCP server never starts at all**. The whole WATCH-04 retry-convergence machinery covers only the daemon lockfile; the store open underneath it has no retry.

No test can catch this because no test edits a file during a live serve session (see WR-04). This is precisely the phase-context failure class (b): individually-correct pieces (per-call opens, exclusive Pebble lock, default-on flush) composing into a defect.

**Fix:** Serialize or retry at the store-open seam. Minimum viable, in ascending scope:
1. **Same-process (scenarios 1 and 2):** give the serve process one shared serialization point between flush and query opens — e.g. thread the daemon's `syncMu` (or a package-level per-storeDir mutex in `graphstore`/`query`) through `OpenAt` so an in-process explore blocks briefly on an in-flight sync instead of erroring, and vice versa.
2. **Cross-process (scenario 3 and daemon-vs-other-process):** wrap `graphstore.Open` calls on the write/reconcile paths in a bounded retry-with-backoff on Pebble's lock-held error (e.g. 5×100ms), and make serve's startup reconcile failure on a lock error **degrade to a stderr warning instead of `return err`** — a transiently-locked store means another writer is actively syncing, which is exactly the "graph will be fresh" case; killing the server over it is strictly worse than starting with a possibly seconds-stale graph.
3. **Scenario 2 specifically:** on flush failure with a lock-held error, re-arm the debouncer (e.g. `deb.Add` of a sentinel path or a direct retry after backoff) so a failed sync is not silently terminal until the next organic event.

## Warnings

### WR-01: `daemon.Run`'s new policy gate silently changes `codegraph daemon`'s CLI contract with no CLI-side handling

**File:** `internal/daemon/daemon.go:156-158`; `internal/cli/daemon.go:33-52` (unchanged, now behaviorally different)

**Issue:** `Run` now returns `watch.ErrWatchDisabled` before touching anything when policy disables watching. `internal/cli/daemon.go` propagates it raw. Consequences for the existing `codegraph daemon` command:
- On a WSL2 `/mnt/<drive>` repo, a command that previously started now exits nonzero with `daemon: watching is disabled by policy: project is on a WSL2 /mnt/ drive, ...` — no D-12 guidance (`codegraph sync` / git hooks), unlike serve.go:130-132 which prints the friendly verbatim message.
- With `CODEGRAPH_NO_WATCH=1` exported (e.g. set globally by a user for their MCP config), `codegraph daemon` now refuses to start with the same terse error.
- `codegraph daemon` grew **no** `--watch`/`--no-watch` flags, so the only escape hatch is the undocumented-at-this-surface `CODEGRAPH_FORCE_WATCH=1` env var.

The shared enforcement point is the right design (WATCH-03), but the CLI presentation layer for the standalone command was not updated to match.

**Fix:** In `internal/cli/daemon.go`'s RunE, `errors.Is(err, watch.ErrWatchDisabled)` and print the same D-12/D-13 stderr message serve.go prints (with the `codegraph sync` guidance), then return a clean exit (or at minimum the friendly message plus nonzero). Optionally add the same `--watch`/`--no-watch` flag pair threaded via `WithProbe` for surface symmetry.

### WR-02: `TestServeWatchStartDeferred` asserts an ordering the Go scheduler does not guarantee — false-positive flake built in

**File:** `internal/cli/serve_test.go:79-116`

**Issue:** The test spawns `serveWatchStart` (which starts its goroutine and returns), then sets `atomic.StoreInt32(&returned, 1)` **after** the call returns. The hook fires as the goroutine's first action and does `atomic.LoadInt32(&returned) == 0 → t.Error`. There is **no happens-before edge** between the goroutine's start and the caller's store: on GOMAXPROCS>1, the spawned goroutine can legitimately begin executing `onWatchWorkStart` on another P while the caller is still executing the `return` statement and the store — observing `returned == 0` and failing the test against fully-correct production code. The doc comment claims this is "a deterministic synchronization hook ... without a sleep/timeout race," but the assertion itself is a scheduling race. This is exactly the flake class the CI file quarantines `internal/daemon` for — being newly minted in `internal/cli`.

**Fix:** Make the goroutine's real work *wait* for the caller's release instead of racing it: have the test's `onWatchWorkStart` block on a channel the caller closes immediately after `serveWatchStart` returns, and assert only that `serveWatchStart` returned (i.e. the calling goroutine reached the close) while the hook was still blocked. E.g.:

```go
released := make(chan struct{})
onWatchWorkStart := func() { <-released; workStarted <- struct{}{} }
cancel, done := serveWatchStart(...)
close(released) // serveWatchStart returned => the mutation (work-before-spawn) would have deadlocked/fired the hook synchronously
```

A synchronous-mutation `serveWatchStart` would block forever inside the hook before returning (caught by a test timeout), while correct code passes deterministically.

### WR-03: CI never runs the race detector, in the phase whose entire surface is goroutine lifecycles

**File:** `.github/workflows/ci.yml:59-93`

**Issue:** No step passes `-race`. This phase's deliverables are almost entirely concurrency: `serveWatchStart`'s goroutine + teardown, `RunWithRetry`'s cancel-vs-sleep select, `Debouncer.fireWG` accounting, the two-session convergence soak, and the stderr-reader goroutine in the integration harness. The soak test comments explicitly size iteration counts for "the 120s CI timeout **under -race**" — but CI never applies it, so the assumption those comments encode is unverified on every PR. The goleak gate catches leaks, not races; they are complementary, not substitutes.

**Fix:** Add `-race` to at least the isolated daemon step (`go test ./internal/daemon/ -count=1 -race`) and ideally the main filtered step (or a dedicated `-race` job on `internal/daemon`, `internal/watch`, `internal/cli` if full-suite `-race` time is a concern).

### WR-04: No test exercises the feature this phase ships — edit-during-live-session → auto-sync → fresh query

**File:** `test/integration/watch_default_test.go` (gap); `test/integration/worktree_notice_test.go` (gap)

**Issue:** The integration suite proves the default-on watcher *doesn't block the handshake* (`TestDefaultWatchHandshakePrompt`), *prints the disabled message* (`TestNoWatchEnvDisablesViaStderr`), and *doesn't break the worktree notice* — but nothing anywhere (unit, soak, or integration) writes a file while a `serve --mcp` session is live and then asserts a subsequent `codegraph_explore` reflects the change. That is WATCH-01's actual value proposition ("the graph auto-updates"), and it is exactly the scenario where CR-01's flush-vs-query store collision lives. The phase's own history (Phase-2 CR-01: "green suite, dead production path") repeats here in miniature: the wiring around the feature is tested; the feature's end-to-end effect is not.

**Fix:** Add an integration test: spawn `serve --mcp` (default-on) on an indexed fixture with `CODEGRAPH_DEBOUNCE_MS` lowered via env, write a new symbol file into the fixture, poll `codegraph_explore` for the new symbol within a bounded window, and assert no tool-call errors occur during the polling loop (which would also have surfaced CR-01's lock collision).

## Info

### IN-01: `daemon.Run` holds the lock blocked on `<-ctx.Done()` even if the watch loop exits early — silent zombie lock-holder

**File:** `internal/daemon/daemon.go:183`; `internal/watch/watcher.go:94-97, 107-110`

**Issue:** `watchLoop` returns if `w.Events`/`w.Errors` close (`!ok` arms), but `Run` blocks on `<-ctx.Done()` *before* `wg.Wait()`. If the fsnotify channels ever close without `Close()` (abnormal fsnotify teardown), `Run` keeps holding the daemon lock with no watcher running, and every other session's `RunWithRetry` defers forever to a watcher that no longer watches. Today this path is practically unreachable (fsnotify only closes channels in `Close`), but the `!ok` arms exist precisely because it isn't impossible.

**Fix:** Have the watcher goroutine signal early exit (e.g. close a `loopDone` channel) and make `Run` select on `ctx.Done()` **or** `loopDone`, returning an error in the latter case so `RunWithRetry`/serve can surface or restart it.

### IN-02: serve's disabled branch recomputes `WatchDisabledReason` instead of carrying the reason from the error

**File:** `internal/cli/serve.go:129`; `internal/daemon/daemon.go:156-157`

**Issue:** `daemon.Run` already embeds the reason in the wrapped error (`fmt.Errorf("%w: %s", watch.ErrWatchDisabled, reason)`); serve.go re-derives it with a second `WatchDisabledReason(repoPath, probe)` call. Today the two calls are consistent (both operate on the same absolute `repoPath` — `hasIndex` guarantees `repoPath` came from `ResolveCodegraphDir`, which absolutizes, matching `daemon.New`'s `filepath.Abs`), but it is a duplicate derivation that a future change to either side can silently desynchronize (worst case: an empty recomputed reason yielding the malformed message `"File watcher disabled — . The graph..."`).

**Fix:** Carry the reason on a typed error (e.g. `watch.DisabledError{Reason string}` satisfying `errors.Is(_, ErrWatchDisabled)`) and extract it in serve.go instead of recomputing.

### IN-03: CI's process substitution masks a `go list` failure mode

**File:** `.github/workflows/ci.yml:63-66`

**Issue:** `set -euo pipefail` does not propagate failures out of `<(go list ./... | grep -v ...)` — `mapfile` succeeds regardless. A total `go list` failure is caught indirectly (empty `pkgs` → `go test` in a root with no Go files errors), but a *partial* emission before failure would silently test a subset with a green check.

**Fix:** Materialize the list first with its own failure check: `pkgs_raw=$(go list ./...)` then filter, so `set -e` sees the `go list` exit code.

### IN-04: `hasIndex` is a startup-time snapshot — an index created mid-session never starts the watcher

**File:** `internal/cli/serve.go:169, 196-199`

**Issue:** With no `.codegraph/` at serve start, `serveWatchStart` returns the no-op pair permanently. If the user runs `codegraph init` mid-session, query paths pick the index up live (per-call `OpenAt` resolution) but the watcher never starts until reconnect — the graph then silently stales as the agent edits. Consistent with MCP-03's current design; flagging because the "picked up live, no restart" story now has a watcher-shaped asymmetry worth a documented decision or a future retry-on-no-index tier in `RunWithRetry`.

**Fix:** Document the asymmetry, or treat `ErrNotInitialized` from `daemon.New` as a retryable state on the same jittered cadence.

---

_Reviewed: 2026-07-16T14:57:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
