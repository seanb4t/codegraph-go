---
phase: 03-watcher-on-mcp-default
reviewed: 2026-07-16T19:40:00Z
depth: deep
files_reviewed: 19
files_reviewed_list:
  - .github/workflows/ci.yml
  - internal/cli/daemon.go
  - internal/cli/serve.go
  - internal/cli/serve_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/soak_test.go
  - internal/graphstore/locked_unix.go
  - internal/graphstore/locked_unix_test.go
  - internal/graphstore/locked_windows.go
  - internal/graphstore/locked_windows_test.go
  - internal/graphstore/open_lock_test.go
  - internal/graphstore/pebble_store.go
  - internal/watch/policy.go
  - internal/watch/policy_test.go
  - test/integration/main_test.go
  - test/integration/watch_default_test.go
  - test/integration/watch_live_sync_test.go
  - test/integration/worktree_notice_test.go
findings:
  critical: 0
  warning: 0
  info: 10
  total: 10
status: issues_found
---

# Phase 3: Code Review Report (Round 4 — fresh full-scope pass, Info items in scope per --all)

**Reviewed:** 2026-07-16T19:40:00Z
**Depth:** deep
**Files Reviewed:** 19
**Status:** issues_found

## Summary

Fresh full-scope deep pass over all 19 Phase-3 files after three review rounds and two fix passes, re-tracing every cross-file chain from scratch rather than trusting prior verdicts: the `Open` retry loop → `classifyOpenError` → `isLockHeldOS` seam (both build-tagged arms), the daemon `Run` → `Debouncer` → `flush` → requeue lifecycle (including the fireWG accounting in `internal/watch/debounce.go` and the `deb.Stop()`/`deb.Wait()` teardown ordering), the serve RunE reconcile → downgrade → `serveWatchStart` → `RunWithRetry` chain, the lock.go acquire/release/stale-detection substrate, and the subprocess integration harness.

**Fresh verification results (checked, not assumed):**

- **Round-3 WR-01 is genuinely fixed:** the `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` step exists at `.github/workflows/ci.yml:116-117` (commit 3a4c2f6), so `locked_windows_test.go:18-20`'s claim about the cross-GOOS gate is now true. The windows classifier arm, its tests, and the unix arm all remain exactly as round 3 verified them.
- **The `ErrStoreLocked` sentinel confinement holds.** `classifyOpenError` has exactly one call site (`pebble_store.go:142`, inside `Open`'s loop); `isLockHeldOS` is called only from `classifyOpenError`; both consumers (`daemon.go:214`, `serve.go:201`) branch via `errors.Is` on the sentinel only. No chain-sniffing API remains anywhere.
- **No new teardown/lifecycle defects.** Re-derived the debouncer accounting independently: `Add`'s `fireWG.Add(1)` at arm time + `fire`'s deferred `Done` + the `timer.Stop()`-undo in both `Add` and `Stop` are balanced on every path; `Run`'s `<-ctx.Done()` → `wg.Wait()` → `deb.Wait()` joins every Sync before the deferred `release()` runs. The requeue closure's capture of `deb` is safely ordered (assignment happens-before the watcher goroutine's spawn, and timers only arm after Adds from that goroutine).
- **No sidecar feedback loop:** `.codegraph/` is excluded from the watch set (`indexer.ShouldSkipDir` via `addRecursive`), and both `touchPending`/`clearPending` and pebble's store writes land inside it, so daemon-driven writes can never re-trigger the debouncer.
- **The integration harness is sound:** `TestMain`'s hard-stop build, `copyFixture`/`buildWorktreeFixture`'s hermetic git usage (skip-on-failure), `newServeClient*`'s `t.Cleanup(c.Close)`-before-`Initialize` ordering, the live-sync test's 500ms-re-touch-vs-100ms-debounce starvation math, and the stderr-reader goroutine's termination on pipe close all check out.

**Adjudicated residuals honored (re-verified as correctly characterized, not re-reported):** the bounded ~400ms `Open` lock-retry window that can surface one tool-call error under a long-lived holder (accepted; the full fix is a shared in-process store handle, out of scope), and the windows any-sharing-violation-classifies-as-lock imprecision (round-3 IN-03, accepted trade-off — the wrap preserves the original error text for diagnosis).

**What remains:** zero Critical, zero Warning. Ten Info items — nine are previously-reported items re-verified still present at current line numbers (in scope this round because the orchestrator requested Info-level fixes, --all), plus one new coverage gap (`codegraph daemon`'s disabled branch, added by fix round 1, has no test). Every finding below carries a mechanical fix.

## Info

### IN-01: `locked_unix_test.go`'s provenance comment is factually wrong — pebble's cross-process fcntl form is a bare errno, not PathError-wrapped

**File:** `internal/graphstore/locked_unix_test.go:26-28`
**Issue:** (Round-3 IN-01, re-verified present.) The case comment claims *"Pebble's vfs surfaces the errno wrapped; errors.Is must traverse"* and synthesizes `&fs.PathError{Op: "fcntl", ...}`. In the pinned `pebble/v2@v2.1.6`, `unix.FcntlFlock` returns the **bare** `syscall.Errno` and nothing between `vfs/file_lock_unix.go:63-65` and `pebble.Open`'s caller wraps it — the real pinned shape is the suite's "bare EWOULDBLOCK" case (EWOULDBLOCK == EAGAIN on every shipped unix target). Matching is functionally correct either way; only the stated provenance is wrong, and a future maintainer trusting it would draw incorrect conclusions about which pebble internals are load-bearing.
**Fix:** Reword the comment on the PathError case (and its name at line 28 if desired): the PathError shape tests that `errors.Is` traversal *would* work if pebble ever wrapped the errno; the bare-errno case below is the real shape pebble v2.1.6 produces (verified: `FcntlFlock` returns the errno unwrapped and the chain to `pebble.Open` is verbatim).

### IN-02: `TestOpenConvergesWhenHolderCloses` is time-based (150ms release vs ~400ms budget) rather than event-synchronized

**File:** `internal/graphstore/open_lock_test.go:60-67`; seam: `internal/graphstore/pebble_store.go:134-137`
**Issue:** (Round-3 IN-02, re-verified present.) The holder releases via `time.Sleep(150 * time.Millisecond)` against `Open`'s retry attempts at t≈0/100/200/300/400ms. The ~250ms scheduling margin is generous, but the test runs inside CI's parallel full-suite step — the same environment where this repo has already documented time-sensitive tests flaking (the isolated `internal/daemon -count=1` step exists for exactly that reason). If the close goroutine alone is delayed past the final attempt under heavy parallel load, the test fails spuriously.
**Fix:** Event-synchronize via a test seam matching the repo's own convention (`daemon.Daemon.onSyncStart`): hoist `Open`'s `time.Sleep(openLockRetryBackoff)` behind an unexported package-level `var openLockRetrySleep = time.Sleep` (`pebble_store.go:136`), and have the test override it to signal attempt boundaries on a channel — the closer goroutine then releases the holder deterministically after observing the second attempt's sleep begin, instead of after a wall-clock guess. Restore the original in a `t.Cleanup`. (Production behavior unchanged; the var is unexported with no setter.)

### IN-03: Requeue give-up log undercounts by one, and the counter never resets after exhaustion — contradicting `maxFlushLockRequeues`' own doc comment

**File:** `internal/daemon/daemon.go:215-219` (behavior); `internal/daemon/daemon.go:55-60` (doc comment it contradicts)
**Issue:** (Round-2 IN-01, re-verified present.) Two defects in one branch: (1) at give-up, `n == maxFlushLockRequeues+1 == 6` — six consecutive lock-lost syncs have occurred (1 original + 5 requeued) — but the log prints `n-1 = 5` "consecutive times"; every post-exhaustion lock-lost flush then logs again with a still-off-by-one, ever-growing count (7→"6", 8→"7", …) that misleadingly spans separate episodes hours apart. (2) The `maxFlushLockRequeues` doc comment (line 58) says the budget is "per-contention-episode", but only `err == nil` resets the counter (line 213): once exhausted, every later lock-lost flush — triggered by organic events in a genuinely fresh contention episode — sees `n > 5` and gets zero requeues until some flush succeeds in between. Impact is bounded (organic events still drive flush attempts; the sidecar keeps staleness observable) but code and comment disagree.
**Fix:** In the `else` branch: log `n` (or `"giving up after %d requeues"` with `maxFlushLockRequeues`) instead of `n-1`, and add `atomic.StoreInt32(&lockRequeues, 0)` so the budget genuinely resets per episode as documented. This cannot create an unbounded loop: the give-up branch never calls `deb.Add`, so each episode's requeue chain remains bounded at 5 and a new episode only starts from a fresh organic watcher event.

### IN-04: Requeue-vs-shutdown TOCTOU can delay `Run`'s return by up to one full debounce window (2s default)

**File:** `internal/daemon/daemon.go:214-221` (the `ctx.Err() == nil` gate + `deb.Add`); `internal/watch/debounce.go:66-84` (Add re-arms unconditionally)
**Issue:** (Round-2 IN-02, re-verified present — `Run` still has no post-`wg.Wait()` `deb.Stop()`, and `Debouncer.Add` has no ctx gate.) The requeue's `ctx.Err() == nil` check and its `deb.Add(flushRetryPath)` are not atomic: cancellation can land between them, after `watchLoop` has already run its `deb.Stop()`. The re-armed timer is then never cancelled, and `Run`'s `deb.Wait()` (line 241) — `fireWG` is incremented at arm time — blocks until it fires, up to a full debounce window later. `fire()`'s own ctx check makes the late fire a no-op (no Sync, no lock violation, no goroutine leak — re-verified), so this is purely added shutdown latency on `codegraph serve` exit / daemon SIGTERM in a narrow race.
**Fix:** Close it at the seam rather than in the caller: add `if d.ctx.Err() != nil { return }` as `Debouncer.Add`'s first statement (before taking `d.mu`). This makes arming a timer post-cancel structurally impossible for every Add caller (the requeue AND watchLoop's tail events), matches `NewDebouncer`'s existing documented contract ("once ctx is cancelled, no further flush fires"), and has no accounting hazard (the early return skips `fireWG.Add`). A belt-and-suspenders `deb.Stop()` between `wg.Wait()` and `deb.Wait()` in `Run` (daemon.go:241) is also safe/idempotent but is insufficient alone — the Add can land after it — so the Add-side gate is the required part.

### IN-05: The watch-disabled reason is re-derived at three independent sites, and `codegraph daemon`'s copy is MCP-branded

**File:** `internal/cli/serve.go:130`, `internal/cli/daemon.go:74-82`, `internal/daemon/daemon.go:177-178`
**Issue:** (Round-1 IN-02 + round-2 IN-04 merged, re-verified present.) `daemon.Run` already embeds the reason in its wrapped error (`fmt.Errorf("%w: %s", watch.ErrWatchDisabled, reason)`, daemon.go:178), yet both CLI consumers re-derive it with fresh `watch.WatchDisabledReason` calls: serve.go:130 (with the session's probe) and cli/daemon.go:78 (with a zero `watch.Probe{}` on a re-absolutized root). Today all three derivations agree, but any future desynchronization (e.g. `filepath.Abs` failing in cli/daemon.go:75 leaves `root` relative → the WSL `/mnt` check misses → empty reason) produces the malformed message `"File watcher disabled — . The graph..."`. Additionally, `codegraph daemon` prints the `[CodeGraph MCP]` banner prefix (cli/daemon.go:79) on a command that has nothing to do with MCP — defensible as verbatim parity, but currently an accident rather than a decision.
**Fix:** Introduce a typed error in internal/watch — `type DisabledError struct{ Reason string }` with `Error()` and `Is(target error) bool { return target == ErrWatchDisabled }` — have `daemon.Run` return `&watch.DisabledError{Reason: reason}` (daemon.go:178; `errors.Is(err, watch.ErrWatchDisabled)` keeps working everywhere), and have both CLI sites extract the reason via `errors.As` instead of recomputing. Delete both `WatchDisabledReason` re-derivation calls in the CLI layer. While touching cli/daemon.go, either drop the `[CodeGraph MCP]` prefix for the standalone command or add a one-line comment recording that the verbatim-parity branding is deliberate.

### IN-06: `codegraph daemon`'s disabled branch has zero test coverage — the only shipped consumer of round-1 WR-01's fix is unexercised

**File:** `internal/cli/daemon.go:68-83` (gap); no `internal/cli/daemon_test.go` exists
**Issue:** New this round. The friendly-disabled-message branch added by fix commit a527a71 (errors.Is on `watch.ErrWatchDisabled` → print D-12 message → return nil) is tested nowhere: `serve_test.go` and the integration suite pin the *serve* path's disabled message at two levels, but nothing executes `newDaemonCmd`'s RunE at all (verified: `newDaemonCmd` is referenced only from root.go and daemon.go). A regression — the branch dropped, the exit code flipped back to nonzero, or the message malformed by IN-05's re-derivation drift — ships silently. Given this phase's own history (round-2's Critical was exactly an untested platform branch rotting invisibly), the shipped CLI surface should not depend on an unexercised branch.
**Fix:** Add `internal/cli/daemon_test.go` with one test: create an indexed fixture root (reuse the package's existing fixture helper pattern), `t.Setenv("CODEGRAPH_NO_WATCH", "1")`, build `cmd := newDaemonCmd()`, wire `cmd.SetArgs([]string{"--path", root})` and `cmd.SetErr(&stderr)`, run `cmd.ExecuteContext(context.Background())`, and assert: returned error is nil (clean exit for a policy-disabled watcher) and `stderr` contains the verbatim `"File watcher disabled — CODEGRAPH_NO_WATCH=1 is set"` plus the `codegraph sync` guidance. The env-driven disable makes the test hermetic (no flags beyond --path), and Run's policy gate returns before any lockfile/watcher work, so the test is instant. If IN-05's typed-error refactor lands first, this test also pins the extraction path.

### IN-07: `daemon.Run` holds the lock blocked on `<-ctx.Done()` even if the watch loop exits early — silent zombie lock-holder

**File:** `internal/daemon/daemon.go:230`; `internal/watch/watcher.go:94-97, 107-111`
**Issue:** (Round-1 IN-01, re-verified present.) `watchLoop` returns when `w.Events`/`w.Errors` close (`!ok` arms), but `Run` blocks on `<-ctx.Done()` *before* `wg.Wait()`. If fsnotify's channels ever close without `Close()` (abnormal teardown), `Run` keeps holding the daemon lockfile with no watcher running — every other session's `RunWithRetry` then defers forever to a watcher that no longer watches, and the graph silently stops auto-updating with the lock still "live" (pid alive, so `isStale` never clears it). Today the path is practically unreachable (fsnotify only closes its channels in `Close`), but the `!ok` arms exist precisely because it is not impossible.
**Fix:** Have the watcher goroutine signal loop exit — e.g. `loopExited := make(chan struct{})`, `go func() { defer wg.Done(); defer close(loopExited); w.Run(ctx, deb) }()` — and replace `<-ctx.Done()` with `select { case <-ctx.Done(): case <-loopExited: }`; when the loop exited without ctx being done, proceed through the same `wg.Wait()`/`deb.Wait()` teardown and return a non-nil error (a new sentinel or wrapped description) so `RunWithRetry` surfaces it (it is neither `ErrLockLive` nor `ErrWatchDisabled`, so RunWithRetry correctly returns it immediately) and serve's watcher goroutine logs it to stderr.

### IN-08: CI's process substitution masks a partial `go list` failure

**File:** `.github/workflows/ci.yml:62-66`
**Issue:** (Round-1 IN-03, re-verified present verbatim.) `set -euo pipefail` does not propagate failures out of `<(go list ./... | grep -v ...)` — `mapfile` succeeds regardless of the substitution's exit status. A *total* `go list` failure is caught only indirectly (empty `pkgs` → `go test` with no args errors in a rootdir with no Go files); a *partial* emission before failure would silently test a subset of packages under a green check.
**Fix:** Materialize the list with its own failure check so `set -e` sees `go list`'s exit code:

```yaml
      - name: Test (excluding internal/daemon — isolated below)
        run: |
          set -euo pipefail
          pkgs_raw=$(go list ./...)
          mapfile -t pkgs < <(printf '%s\n' "$pkgs_raw" | grep -v '/internal/daemon$')
          go test "${pkgs[@]}"
```

### IN-09: `hasIndex` is a startup-time snapshot — an index created mid-session never starts the watcher

**File:** `internal/cli/serve.go:170, 216-219`; `internal/cli/serve.go:91-95` (the permanent no-op pair)
**Issue:** (Round-1 IN-04, re-verified present.) With no `.codegraph/` at serve start, `serveWatchStart` returns the closed-channel no-op pair permanently. If the user runs `codegraph init` mid-session, the query paths pick the index up live (per-call `OpenAt` resolution) but the watcher never starts until reconnect — the graph then silently stales as the agent edits, with no disabled message ever printed (the watcher goroutine never existed). Consistent with MCP-03's design, but the "picked up live, no restart" story now has an undocumented watcher-shaped asymmetry.
**Fix (documentation, the mechanical minimum):** extend `serveWatchStart`'s doc comment (serve.go:72-74, the `!hasIndex` paragraph) with one sentence recording the decision: an index created mid-session is served live by per-call query resolution but does NOT retroactively start the watcher — auto-sync begins on the next `serve --mcp` session; the D-04a stale/mtime fallback keeps staleness observable meanwhile. (The behavioral alternative — treating `daemon.ErrNotInitialized` as a retryable state on `RunWithRetry`'s jittered cadence — is a design change; do not take it in a mechanical fix pass.)

### IN-10: `checkTargetOverwrite`'s probe race window widened ~400ms by Open's retry — still undocumented at the probe

**File:** `internal/migrate/migrate.go:371-378`; cause: `internal/graphstore/pebble_store.go:132-148`
**Issue:** (Round-2 IN-03, re-verified: the WR-03 comment at migrate.go:371-375 still says nothing about the retry.) `graphstore.Open`'s bounded lock retry means the migrate probe's read-only health check now waits up to ~400ms against a contended target, marginally widening the pre-existing race in which a lock holder exiting during the probe window lets `checkTargetOverwrite` proceed against a store another live process was just using. Semantics are unchanged (a still-locked target still refuses); only the window is wider, and the fact is on record nowhere near the code.
**Fix:** Append one sentence to the WR-03 comment block (migrate.go:371-375): `graphstore.Open` retries a lock-held open for ~400ms (5×100ms, 03-REVIEW.md CR-01), so a holder exiting mid-probe can now be missed across a slightly wider window than before — acceptable for this best-effort refusal check; revisit if migrate-vs-live-daemon coordination ever becomes a real workflow.

---

_Reviewed: 2026-07-16T19:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
