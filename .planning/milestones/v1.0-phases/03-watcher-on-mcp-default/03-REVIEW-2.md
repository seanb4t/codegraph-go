---
phase: 03-watcher-on-mcp-default
reviewed: 2026-07-16T15:35:12Z
depth: deep
files_reviewed: 14
files_reviewed_list:
  - .github/workflows/ci.yml
  - internal/cli/daemon.go
  - internal/cli/serve.go
  - internal/cli/serve_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/soak_test.go
  - internal/graphstore/pebble_store.go
  - internal/watch/policy.go
  - internal/watch/policy_test.go
  - test/integration/main_test.go
  - test/integration/watch_default_test.go
  - test/integration/watch_live_sync_test.go
  - test/integration/worktree_notice_test.go
findings:
  critical: 1
  warning: 2
  info: 5
  total: 8
status: issues_found
---

# Phase 3: Code Review Report (Re-review after fix pass 1)

**Reviewed:** 2026-07-16T15:35:12Z
**Depth:** deep
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Second-pass deep review after the round-1 fixes (CR-01 lock retry, WR-01 daemon message, WR-02 deterministic seam test, WR-03 CI -race, WR-04 live-sync test), focused on emergent interactions introduced BY the fixes. Verified against the actual pinned dependency source (`pebble/v2@v2.1.6` `vfs/file_lock_unix.go` / `file_lock_windows.go` / `open.go`), the full Debouncer/watchLoop teardown chain, `indexer.Sync`/`Discover`'s error propagation, `migrate.go`'s Open probe, and `query.computeStale`/`staleBanner`.

**What holds up (verified, not assumed):**

- **The flush requeue does NOT violate the teardown invariants.** Traced the worst-case TOCTOU (callback passes the `ctx.Err() == nil` gate, ctx cancels, `deb.Add(flushRetryPath)` re-arms a timer after `watchLoop` already ran `deb.Stop()`): `Debouncer.Add` does `fireWG.Add(1)` at arm time, so `Run`'s `deb.Wait()` joins even that late-armed timer; when it fires, `fire()`'s own `d.ctx.Err() != nil` check prevents the flush — **no `indexer.Sync` can start after `Run` releases the lock**, and no goroutine outlives `Run` (goleak-consistent). The only residual is shutdown latency (IN-02 below).
- **Nothing is recorded under a cancelled ctx.** The only state mutations in the wrapper are the counter reset (`err == nil` — a genuinely committed Sync, whose sidecar-clear matches reality) and the requeue (explicitly ctx-gated). A lock-lost flush under cancelled ctx falls through the switch recording nothing, sidecar left in place. No Phase-2 BL-01 recurrence.
- **The serve reconcile downgrade cannot silently serve a stale graph without a banner.** Traced the adversarial path (reconcile skipped on a lock held by a NON-syncing holder, e.g. another process's long explore; own watcher then live, so no "no-daemon" excuse): `query.computeStale` (internal/query/status.go:141-163) checks the sidecar **OR unconditionally** compares newest source mtime against `Meta.last_sync_unix_ms` — offline edits that the skipped reconcile would have absorbed keep `stale=true`, and `render_markdown.go`'s `staleBanner` prepends the warning to every explore payload. Staleness stays observable in all reconcile-skip paths; the next organic edit's full re-diff Sync then absorbs the offline changes.
- **`RunWithRetry` interacts safely with both new retry layers.** Flush errors never propagate to `Run`'s return (the wrapper swallows them), so the requeue can never masquerade as `ErrLockLive`; `graphstore.Open`'s in-call sleeps happen inside `indexer.Sync` under `syncMu` and are joined by `deb.Wait()` — bounded (~400ms) added shutdown latency at worst.
- **`Open`'s retry loop itself is clean**: sleeps before attempts 2-5 only (~400ms worst case, matching its comment), holds no resource across attempts, breaks immediately on non-lock errors, returns the original error unmodified after exhaustion. `migrate.checkTargetOverwrite`'s read-only probe semantics are unchanged (a locked target still refuses, now ~400ms slower — see IN-03), and `migrate.Run`'s partial-store open is contention-free by construction (fresh temp dir).
- **The WR-02 test fix is genuinely deterministic.** The block-until-released channel design has real happens-before edges in both directions; the mutated (synchronous-hook) case trips the bounded `retCh` select instead of deadlocking; the disabled-message test is safe even against an already-cancelled ctx because `Run`'s policy gate runs before any ctx check.
- **The WR-04 test does not leak on failure paths** (`t.Cleanup(c.Close)` registered before `Initialize`; `exec.CommandContext` kills the child on transport cancel), its re-touch design survives the watcher-startup race (Sync re-diffs the whole repo, so the re-touched anchor alone recovers the entire missed burst), and its 500ms re-touch cadence provably cannot starve the 100ms debounce.
- **CI's `-race` step runs the exact concurrency surface** (`internal/daemon`, `internal/watch`, `internal/cli`) with the documented `-p 1` isolation; `internal/daemon` running twice in the job is redundant wall time, not a correctness issue.

**What does not hold up:** the CR-01 fix is platform-conditional — it is a complete no-op on both Windows release targets (Critical below) — and `IsLockHeld`'s errno matching misroutes an entire class of non-lock errors through the new degrade/requeue paths (Warning), with zero direct test coverage of the classifier either way (Warning).

## Critical Issues

### CR-01: `IsLockHeld` matches neither of Pebble's Windows lock-failure forms — the entire round-1 CR-01 fix is inert on both shipped Windows targets

**File:** `internal/graphstore/pebble_store.go:99-107` (interacting: `internal/cli/serve.go:197`, `internal/daemon/daemon.go:214`; `.goreleaser.yaml` builds `codegraph-windows-amd64` and `codegraph-windows-arm64`)

**Issue:** `IsLockHeld` recognizes exactly two shapes, both unix-only. Verified against the pinned `pebble/v2@v2.1.6`:

- `vfs/file_lock_unix.go:49` — the in-process map failure `errors.New("lock held by current process")` (string-matched) — that file is `//go:build darwin || ... || linux ...` only.
- `vfs/file_lock_unix.go:63` — `unix.FcntlFlock(F_SETLK)` returning `EAGAIN`/`EWOULDBLOCK` (errno-matched).

On Windows, `vfs/file_lock_windows.go` has **no in-process tracking map at all** and acquires the lock by `windows.CreateFile(..., shareMode=0, CREATE_ALWAYS, ...)`. Every collision — same-process (a live flush vs. a query open in the same `serve --mcp` process) and cross-process alike — surfaces as `ERROR_SHARING_VIOLATION` (`windows.Errno(32)`), which matches none of `IsLockHeld`'s three checks (`EAGAIN`/`EWOULDBLOCK`/`EACCES` are different errno values on Windows, and the unix string never appears).

Consequence: on windows-amd64/arm64 — first-class release targets in `.goreleaser.yaml` — all three legs of the round-1 CR-01 fix silently do nothing, and the original Critical's failure modes return verbatim on the default-on watcher path:

1. `graphstore.Open` never retries → an agent's `codegraph_explore` landing inside a flush window returns a lock error as a tool-call failure (round-1 scenario 1).
2. `daemon.Run`'s requeue branch never fires → a flush that lost the race is silently terminal until the next organic event (scenario 2).
3. `serve.go`'s reconcile downgrade never triggers → a second session's `serve --mcp` dies at startup on a sibling's in-flight sync (scenario 3).

Nothing in CI can catch this: the integration anchor (`TestLiveEditAutoSyncReachesExplore`) runs on `ubuntu-latest` only, and there is no Windows unit test of the classifier (see WR-02). The fix report's "all_fixed" claim holds only for unix.

**Fix:** Add a platform-specific arm to the classifier via build-tagged files inside graphstore (the sole pebble-aware package), e.g.:

```go
// islockheld_windows.go
//go:build windows
package graphstore

import ("errors"; "syscall")

func isLockHeldOS(err error) bool {
    return errors.Is(err, syscall.Errno(32)) // ERROR_SHARING_VIOLATION from vfs/file_lock_windows.go's CreateFile(share=0)
}
```

with a unix counterpart carrying the current errno + string checks, and `IsLockHeld` delegating to `isLockHeldOS`. Better still, combine with WR-01's fix below: classify once inside `Open` (where the error is known to come from `pebble.Open`) and wrap in an exported sentinel, so both the platform matching and the misclassification problem are solved at one seam. At minimum, add a Windows leg (or a `GOOS=windows go vet`-style compile check plus a unit test gated on `runtime.GOOS`) so the classifier cannot silently regress per-platform again.

## Warnings

### WR-01: `IsLockHeld`'s errno matching over whole `indexer.Sync` error chains misroutes permission errors into the lock-degrade paths — a should-be-fatal reconcile error now warns-and-continues with a false diagnosis

**File:** `internal/graphstore/pebble_store.go:103`; consumers `internal/cli/serve.go:197`, `internal/daemon/daemon.go:214`

**Issue:** `IsLockHeld` was designed against `pebble.Open`'s error shapes, but both consumers apply it to the **entire error chain of `indexer.Sync`** — which propagates arbitrary filesystem errors, all as `*fs.PathError`s that `errors.Is` happily unwraps to a `syscall.Errno`:

- `Discover`'s `WalkDir` callback returns the walk error verbatim on an unreadable directory (`internal/indexer/discover.go:101-103`), and `build.Default.MatchFile` **opens and reads** every Go file for build-tag evaluation — an unreadable file yields `EACCES` right there.
- `contentHash` on a stat-mismatched file propagates its `os.Open` `EACCES` (`internal/indexer/sync.go:113-116`).
- Even within pebble itself, `vfs/file_lock_unix.go:51-54`'s `os.Create(LOCK)` on an unwritable store dir returns `EACCES` — a permanent permission failure, not a held lock.

Every one of these satisfies `errors.Is(err, syscall.EACCES)` → `IsLockHeld == true`. Consequences:

1. **serve.go's own stated contract is violated** ("Every non-lock error stays fatal", serve.go:196-198): a permanent permission error at startup reconcile degrades to the warning *"the graph store is locked by another codegraph process (likely an in-flight sync; the graph will refresh shortly)"* — an actively false diagnosis (nothing will refresh; every subsequent flush fails the same way) — and the MCP server starts against a graph it can never update, instead of failing fast with the real error.
2. `daemon.Run` burns 5 pointless requeues (plus `Open`'s 5×100ms in-call retries per attempt) on an error that can never succeed before giving up.

Note the errno branch buys nothing on the platforms actually shipped: Linux/macOS `F_SETLK` conflicts return `EAGAIN` (EWOULDBLOCK == EAGAIN there); `EACCES`-for-locks is a POSIX allowance for *other* systems — so including it is pure downside on every release target.

**Fix:** Classify once, at the only place the error provenance is known — inside `Open`'s retry loop, where `err` is guaranteed to be `pebble.Open`'s — and export a sentinel instead of a heuristic:

```go
var ErrStoreLocked = errors.New("graphstore: store lock held")

// in Open's loop, when the pebble error matches the lock forms:
lastErr = fmt.Errorf("%w: %v", ErrStoreLocked, err)
```

with `IsLockHeld(err)` reduced to `errors.Is(err, ErrStoreLocked)`. Since every consumer's error arrives via `graphstore.Open` (Sync propagates it unwrapped), Sync-chain `EACCES` from `Discover`/`contentHash`/`MatchFile` structurally can never match again, and the platform-specific raw matching (including CR-01's Windows arm) is confined to one internal helper. At minimum, drop `EACCES` from the match set and scope the errno checks to `*fs.PathError`s whose `Path` basename is `"LOCK"`.

### WR-02: The classifier and retry loop that now gate three error-handling decisions have zero direct tests — the string form is pinned only by a probabilistic integration test, the errno form by nothing at all

**File:** `internal/graphstore/pebble_store.go:99-136` (gap); `internal/graphstore/store_test.go` (every `Open` call is against a fresh, uncontended `t.TempDir()`)

**Issue:** `IsLockHeld` and `Open`'s retry loop are the load-bearing seam of the whole CR-01 fix, and nothing unit-tests them:

- The **in-process string form** (`"lock held by current process"`) is an unexported message in `pebble/v2@v2.1.6`'s vfs. A pebble version bump that rewords it silently turns `Open`'s same-process retry AND the daemon requeue AND serve's downgrade off. The only thing standing guard is `TestLiveEditAutoSyncReachesExplore` — a 90-second-deadline integration test whose detection of the regression depends on an explore probabilistically landing inside a flush window; the fix report itself measured the pre-fix failure at 2/2 runs, which is evidence, not a pin.
- The **cross-process errno form** (`EAGAIN` from `FcntlFlock`) has no test anywhere in the repo — the integration test's collisions are all same-process (flush and explores live in the one spawned serve subprocess). Whether `errors.Is` actually traverses pebble's wrapping for this form was verified only by this review's manual source read, not by any executable check.
- The retry loop's success-after-contention behavior (holder releases between attempts → `Open` succeeds) and its give-up-after-5 behavior are likewise untested.

A trivially cheap deterministic test exists: `Open(dir)` twice in-process (asserting the second error satisfies `IsLockHeld` and that `Open` returns it after the bounded retries), plus a goroutine that closes the first store mid-retry (asserting the second `Open` converges to success). That pins the string form against every future pebble bump at unit speed, on every platform CI runs.

**Fix:** Add `internal/graphstore` unit tests: (1) double-`Open` same directory → second fails, `IsLockHeld(err)` true, elapsed ≥ 4×backoff; (2) double-`Open` with the holder closing after ~150ms → second succeeds; (3) `IsLockHeld` false for `ErrNotFound`, a plain `fs.PathError{Err: EACCES}` from a non-LOCK path (locks in WR-01's fix), and `nil`. If CR-01's sentinel refactor lands, test the sentinel wrap instead — same three cases.

## Info

### IN-01: Requeue give-up log undercounts by one, and the counter never resets after exhaustion — a later contention episode gets zero requeues until one success intervenes

**File:** `internal/daemon/daemon.go:215-219`

**Issue:** At give-up (`n == maxFlushLockRequeues+1 == 6`) the log prints `n-1 = 5` "consecutive times", but the counter counts *failures* — 6 consecutive lock-lost syncs have occurred (1 original + 5 requeued). Separately, the doc comment says the budget is "per-contention-episode", but only `err == nil` resets the counter: once exhausted, every subsequent lock-lost flush (triggered by organic events, possibly hours later in a fresh episode) sees `n > 5` and is never requeued until some flush succeeds in between. Impact is bounded (organic events still drive flushes; sidecar stays observable), but code and comment disagree.

**Fix:** Log `n` (or "giving up after %d requeues" with `maxFlushLockRequeues`), and either reset the counter when logging the give-up or document the actual until-next-success semantics.

### IN-02: Requeue-vs-shutdown TOCTOU can delay `Run`'s return by one full debounce window (2s default)

**File:** `internal/daemon/daemon.go:214-221`; `internal/watch/debounce.go:66-84,134-136`

**Issue:** The `ctx.Err() == nil` gate and `deb.Add` are not atomic: cancel can land between them, after `watchLoop` has already run `deb.Stop()`. The re-armed timer is then uncancellable (nothing calls `Stop` again), and `Run`'s `deb.Wait()` — `fireWG` is incremented at arm time — blocks until it fires, up to a full debounce window later. `fire()`'s ctx check makes it a no-op (no Sync, no lock violation, no leak; verified), so this is purely added shutdown latency on `codegraph serve` exit / `codegraph daemon` SIGTERM in a narrow race.

**Fix:** Have `Run` call `deb.Stop()` once more after `wg.Wait()` (before `deb.Wait()`) — it is idempotent and cancels any timer the TOCTOU armed, restoring prompt shutdown.

### IN-03: `Open`'s retry changes timing (not semantics) for every pre-existing caller — worth one line of documentation at the two probes

**File:** `internal/graphstore/pebble_store.go:120-136`; `internal/migrate/migrate.go:371-390`

**Issue:** One-shot CLI commands (`status`/`explore`/`sync`/`init`) and `migrate.checkTargetOverwrite`'s read-only health probe now silently wait up to ~400ms when the store is contended. Semantics are preserved everywhere (verified: the migrate probe still refuses a still-locked target; `pebble.Open` fails at lock acquisition without compounding partial state across attempts), but the retry marginally widens the pre-existing race in which a lock holder exiting during the probe window lets `checkTargetOverwrite` proceed against a store another live process was just using. Pre-existing exposure, now 400ms wider.

**Fix:** None required; note the widened window in `checkTargetOverwrite`'s WR-03 comment so a future migrate-vs-live-daemon coordination decision has the fact on record.

### IN-04: `codegraph daemon`'s new disabled message is MCP-branded and is the third copy of the reason re-derivation (round-1 IN-02's pattern, now in one more place)

**File:** `internal/cli/daemon.go:74-82`

**Issue:** The standalone CLI daemon prints `[CodeGraph MCP] File watcher disabled — ...` — the MCP server's banner, on a command that has nothing to do with MCP (verbatim-parity is a defensible reason; flagging so it's a decision, not an accident). More substantively, this fix added a **third** independent `watch.WatchDisabledReason` re-derivation (serve.go:130, daemon.go RunE, plus the wrapped error's own embedded reason) — round-1 IN-02's typed-error suggestion (`watch.DisabledError{Reason string}`) would now collapse three sites instead of two, and eliminate the malformed-message risk (`"disabled — . The graph..."`) at all of them if the derivations ever desynchronize.

**Fix:** Same as round-1 IN-02, with one more call site as added motivation.

### IN-05: Round-1 Info findings IN-01 through IN-04 remain open

**File:** see `.planning/phases/03-watcher-on-mcp-default/03-REVIEW.md`

**Issue:** The fix pass scoped to critical+warning only (per its report); the four round-1 Info items — zombie lock-holder on abnormal fsnotify channel close (daemon.go:230 `<-ctx.Done()` before any loop-exit signal), the serve-side reason recompute, CI's process-substitution masking of a partial `go list` failure (ci.yml:63-65, unchanged), and the `hasIndex` startup-snapshot watcher asymmetry — are all still present as described. Re-verified, not re-litigated.

**Fix:** As written in round 1; none are blockers.

---

_Reviewed: 2026-07-16T15:35:12Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
