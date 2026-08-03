---
phase: 03-watcher-on-mcp-default
fixed_at: 2026-07-16T18:49:21Z
review_path: .planning/phases/03-watcher-on-mcp-default/03-REVIEW.md
iteration: 2
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report

**Fixed at:** 2026-07-16T19:55:00Z
**Source review:** .planning/phases/03-watcher-on-mcp-default/03-REVIEW.md (Round 4 — 0 Critical, 0 Warning, 10 Info)
**Iteration:** 1 (fix_scope: all — Info findings in scope per orchestrator)

**Summary:**
- Findings in scope: 10 (IN-01..IN-10)
- Fixed: 10
- Skipped: 0

**Verification (full battery, all green):** `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./testdata/golden/...`, `go test ./test/integration/... -count=1`, `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` (fireWG accounting + goleak soak intact), `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/`.

**Guardrail compliance:** zero new dependencies; `internal/agents/*.go` untouched; serve's D-12 disabled message byte-identical (only `codegraph daemon`'s own message dropped the `[CodeGraph MCP]` branding, per guardrail 3); `errors.Is(err, watch.ErrWatchDisabled)` preserved via `DisabledError.Is`; IN-03 reset happens only in the give-up branch, which the outer `ctx.Err() == nil` gate makes unreachable under cancellation (BL-01); IN-02 uses a test seam (`openLockRetrySleep`), not wider timing margins.

## Fixed Issues

### IN-01: Pinned-pebble provenance claim in unix lock-classifier test comment

**Files modified:** `internal/graphstore/locked_unix_test.go`
**Commit:** 47170db
**Applied fix:** Reworded the PathError case's comment (and case name) to state the truth: the pinned pebble/v2 returns the bare errno unwrapped (the bare-EWOULDBLOCK case is the real shape); the PathError case pins that `errors.Is` traversal would still classify correctly if a future pebble ever wrapped it. Comment/name only — no behavior change.

### IN-02: Time-based `TestOpenConvergesWhenHolderCloses` made event-synchronized

**Files modified:** `internal/graphstore/pebble_store.go`, `internal/graphstore/open_lock_test.go`
**Commit:** 292fb27
**Applied fix:** Hoisted `Open`'s between-attempts sleep behind an unexported `var openLockRetrySleep = time.Sleep` (onSyncStart-convention test seam, no exported setter). The test overrides it to signal attempt boundaries on an unbuffered channel: the closer goroutine releases the holder after observing attempt 2's sleep begin, and any later attempt's sleep-send blocks until `Close` has returned (happens-before), making convergence deterministic instead of a 150ms wall-clock guess against the ~400ms budget. Original restored via `t.Cleanup`. The sibling retry-budget test still measures real elapsed time (seam defaults to `time.Sleep`).

### IN-03: Requeue give-up log off-by-one + counter never reset after exhaustion

**Files modified:** `internal/daemon/daemon.go`
**Commit:** aec7945
**Applied fix:** The give-up branch now logs `n` (the true count of consecutive lock-lost syncs: 1 organic + 5 requeued) instead of `n-1`, and resets `lockRequeues` to 0 at exhaustion so the budget is genuinely per-contention-episode as `maxFlushLockRequeues`' doc comment promises (doc comment updated to match). No unbounded loop possible: the give-up branch never calls `deb.Add`, and the outer `ctx.Err() == nil` gate keeps a cancelled shutdown from ever reaching the reset (BL-01: nothing recorded under a cancelled ctx).
**Status note:** fixed — requires human verification (logic change; race battery + daemon suite green, but the exact give-up/reset sequence has no dedicated test pinning the new count).

### IN-04: Requeue-vs-shutdown TOCTOU delaying Run's return up to one debounce window

**Files modified:** `internal/watch/debounce.go`, `internal/daemon/daemon.go`
**Commit:** aec7945
**Applied fix:** `Debouncer.Add` now returns immediately when `d.ctx.Err() != nil` — the early return precedes `fireWG.Add(1)`, so Wait's accounting is untouched (guardrail 1 verified by the -race battery). This is the structural, every-caller fix per the review's recommended shape; `Run` additionally gained the belt-and-suspenders idempotent `deb.Stop()` between `wg.Wait()` and `deb.Wait()`.
**Status note:** fixed — requires human verification (concurrency logic; `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/...` green including goleak soak).

### IN-05: Watch-disabled reason re-derived at three sites; daemon command MCP-branded

**Files modified:** `internal/watch/policy.go`, `internal/daemon/daemon.go`, `internal/cli/serve.go`, `internal/cli/daemon.go`
**Commit:** f0a4b70
**Applied fix:** Introduced `watch.DisabledError{Reason}` with `Error()` rendering the exact string the old `fmt.Errorf("%w: %s", ...)` wrap produced and `Is(target) bool { return target == ErrWatchDisabled }` so every existing `errors.Is` check keeps working (guardrail 3). `daemon.Run` returns the typed error; both CLI sites extract the reason via `errors.As` and both `WatchDisabledReason` re-derivation calls were deleted. serve's MCP-context stderr message is BYTE-IDENTICAL (verbatim D-12 TS string); `codegraph daemon`'s standalone message deliberately drops the `[CodeGraph MCP]` banner (recorded in a comment, permitted by guardrail 3). `path/filepath` import dropped from cli/daemon.go (no longer needed).

### IN-06: `codegraph daemon`'s disabled branch had zero test coverage

**Files modified:** `internal/cli/daemon_test.go` (new file — required by the fix: the test the finding demands)
**Commit:** ce44fff
**Applied fix:** New `TestDaemonCmdPolicyDisabledExitsCleanly` using the package's existing in-process pattern (`execCmd`/`copyFixture`, per guardrail 4 — no subprocess machinery): indexed fixture root, `t.Setenv("CODEGRAPH_NO_WATCH", "1")`, asserts nil error (clean exit), verbatim `"File watcher disabled — CODEGRAPH_NO_WATCH=1 is set"` + `codegraph sync` guidance on stderr, and absence of the `[CodeGraph MCP]` banner (pinning the IN-05 decision and the `errors.As` extraction path).

### IN-07: `daemon.Run` zombie lock-holder when the watch loop exits without ctx cancellation

**Files modified:** `internal/daemon/daemon.go`
**Commit:** aec7945
**Applied fix:** The watcher goroutine now `defer close(loopExited)`; `Run` replaced `<-ctx.Done()` with `select { case <-ctx.Done(): case <-loopExited: }`. When the loop exited with `ctx.Err() == nil` (abnormal fsnotify channel close), Run proceeds through the same `wg.Wait()`/`deb.Stop()`/`deb.Wait()` teardown, releases the lock via the existing deferred `release()`, and returns the new `ErrWatcherClosed` sentinel — neither `ErrLockLive` nor `ErrWatchDisabled`, so `RunWithRetry` surfaces it immediately and serve's watcher goroutine logs it to stderr (review's exact recommended shape; guardrail 5 — lock released via session teardown).
**Status note:** fixed — requires human verification (lifecycle logic; goleak soak + race battery green).

### IN-08: CI process substitution masking partial `go list` failure

**Files modified:** `.github/workflows/ci.yml`
**Commit:** e444082
**Applied fix:** Applied the review's exact recommendation: `pkgs_raw=$(go list ./...)` materializes the list so `set -e` checks go list's exit code, then `mapfile -t pkgs < <(printf '%s\n' "$pkgs_raw" | grep -v '/internal/daemon$')`. Snippet validated locally (38 packages, internal/daemon excluded). No `${{ github.* }}` interpolation introduced (workflow's env-indirection discipline preserved).

### IN-09: `hasIndex` startup-time snapshot undocumented

**Files modified:** `internal/cli/serve.go`
**Commit:** be8fa03
**Applied fix:** Extended `serveWatchStart`'s `!hasIndex` doc paragraph with the review's mechanical-minimum sentence: an index created mid-session is served live by per-call query resolution but does NOT retroactively start the watcher — auto-sync begins on the next `serve --mcp` session; the D-04a stale/mtime fallback keeps staleness observable meanwhile. Doc-only; the behavioral alternative was deliberately not taken (per the review and guardrail 6).

### IN-10: `checkTargetOverwrite` probe race window widening undocumented

**Files modified:** `internal/migrate/migrate.go`
**Commit:** 4b1807d
**Applied fix:** Appended the review's sentence to the WR-03 comment block: `graphstore.Open` retries a lock-held open for ~400ms (5×100ms, 03-REVIEW.md CR-01), so a holder exiting mid-probe can be missed across a slightly wider window — acceptable for this best-effort refusal check; revisit if migrate-vs-live-daemon coordination becomes a real workflow. Doc-only; no behavior change.

## Skipped Issues

None — all 10 findings fixed.

---

# Iteration 2 (round-5 re-review: 1 Warning + 3 Info, fix_scope: all)

**Fixed at:** 2026-07-16T18:49:21Z
**Source review:** .planning/phases/03-watcher-on-mcp-default/03-REVIEW.md (Round 5, reviewed 2026-07-16T16:43:41Z — post-fix regression hunt over the 10 Info fixes above)

**Summary:**
- Findings in scope: 4 (WR-01, IN-01, IN-02, IN-03)
- Fixed: 4
- Skipped: 0

## Fixed Issues

### WR-01: aec7945's concurrency-lifecycle logic gets dedicated deterministic tests

**Files modified:** `internal/daemon/daemon.go`, `internal/daemon/daemon_test.go`, `internal/watch/debounce_test.go`
**Commits:** 0a4daac (tests + seams), b194d3d (follow-up hardening)
**Applied fix:** Three deterministic, goleak-clean tests covering the three untested branches, plus two unexported test-only seams (both mirroring the established `onSync`/`onSyncStart` convention — no exported setters):

1. **Give-up count/reset arithmetic** (`TestDaemonFlushLockRequeueGivesUpPerEpisode`, internal/daemon): pins exactly 6 sync attempts per contention episode (1 organic + `maxFlushLockRequeues` requeues; requeues at n=1..5, give-up at n=6), that the give-up branch never requeues (1s quiet-gap negative check — cannot flake in the pass direction since no new events exist), that a second fresh episode gets its own 6 (a broken at-give-up reset would give up after 1 attempt and fail the test by timeout), and that a post-contention event syncs successfully. **Adaptation from the review's recipe:** contention is injected via a new `syncFn` seam returning `graphstore.ErrStoreLocked` (the exact sentinel the wrapper's `errors.Is` branch keys on) rather than holding a real `graphstore.Open` handle — the real-lock approach was implemented first and PASSED the arithmetic assertions, but each `pebble.Open` failing on the held LOCK can leak pebble's disk-health ticker goroutine (vfs/disk_health.go: an in-flight op restarts the ticker after the Open-error path's FS Close, and nothing ever closes the replacement stopper), which tripped the package's goleak `TestMain` gate. Real ErrStoreLocked propagation through `indexer.Sync` remains covered by `test/integration/watch_live_sync_test.go`. **This clears the iteration-1 "requires human verification" flag on IN-03's arithmetic: the count/reset sequence is now machine-pinned.**
2. **ErrWatcherClosed path** (`TestRunReturnsErrWatcherClosedAndReleasesLock`, internal/daemon): a new `onWatchOpen` seam hands the test the live watcher, whose `Close()` shuts fsnotify's channels out from under a running `Run`. Driven through `RunWithRetry` so one test pins all three assertions: `errors.Is(err, ErrWatcherClosed)`, lockfile released (no zombie lock-holder), and immediate surfacing (onDeferred `t.Error`s if called — no retry). Clears the iteration-1 IN-07 verification flag.
3. **Add-after-cancel no-op** (`TestDebounceAddAfterCancelIsNoOp`, internal/watch): structural same-package pin — after `cancel()`, `Add` must leave `d.timer` nil and `d.pending` empty (fully deterministic, no timing), `Wait` must return immediately (fireWG untouched), and no flush fires within ~2 windows (behavioral belt). Clears the iteration-1 IN-04 verification flag.

Follow-up (b194d3d): while running the full suite under parallel load, the PRE-EXISTING `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` flaked once — fsnotify delivered the write's events straddling a debounce window, producing a second overlapping fire whose `onSyncStart` panicked on a double `close(flushStarted)`. Guarded with `sync.Once` (the straggler fire now parks on `<-releaseFlush` alongside the first, serialized by syncMu). Squarely within WR-01's deterministic-daemon-tests mandate; the flake did not reproduce across two subsequent full-suite runs and 3 isolated reruns.

### IN-01: Backstop Stop comment no longer overstates its coverage

**Files modified:** `internal/daemon/daemon.go`
**Commit:** c94d394
**Applied fix:** Comment-only, per the review's mechanical minimum. The backstop `deb.Stop()` comment now records that both requeue defenses are ctx gates covering only the cancellation teardown: in the loopExited-with-live-ctx path a lock-lost in-flight flush can requeue past the Stop and extend `deb.Wait()` by up to `maxFlushLockRequeues` windows plus sync durations — bounded and invariant-safe (chain terminates via success/non-lock-error/give-up; lock correctly held throughout). Run's doc comment gets a matching parenthetical so it no longer implies prompt teardown on the IN-07 path. The behavioral alternative (internal `context.WithCancel` cancelled when loopExited fires with ctx alive) was NOT taken: it is not a one-line tightening — it restructures teardown gating, exactly what the fix scope forbade; the existing `deb.Stop()`-before-`Wait()` ordering is already in place, so no safe smaller tightening exists.

### IN-02: serve's disabled branch gated on errors.As, symmetric with cli/daemon.go

**Files modified:** `internal/cli/serve.go`
**Commit:** f454081
**Applied fix:** The switch case itself is now `case errors.As(runErr, &de):` (was `errors.Is(runErr, watch.ErrWatchDisabled)` followed by a best-effort As), printing `de.Reason` directly. A bare or `fmt.Errorf("%w", ...)`-wrapped sentinel from any future producer falls through to the `default:` branch, which renders `runErr` via `%v` — `DisabledError.Error()` embeds the reason, and a bare sentinel prints its own text — so the malformed `"disabled — ."` empty-reason shape is unreachable by construction. The D-12 format string is byte-untouched (now verified by the IN-03 full-line pin below).

### IN-03: serve's FULL D-12 message byte sequence pinned, banner included

**Files modified:** `internal/cli/serve_test.go`
**Commit:** b4bee7e
**Applied fix:** `TestServeWatchStartDisabledPrintsVerbatimMessage` now asserts one `strings.Contains` on the complete verbatim line — `[CodeGraph MCP] File watcher disabled — CODEGRAPH_NO_WATCH=1 is set. The graph will not auto-update; run ` `` `codegraph sync` `` ` (or install the git sync hooks via ` `` `codegraph init` `` `) to refresh.` plus trailing newline — replacing the reason-substring partial match. A regression dropping the banner or mangling the trailing guidance's punctuation now fails this test (presence side; `internal/cli/daemon_test.go` already pins banner absence for `codegraph daemon`).

## Skipped Issues (iteration 2)

None — all 4 findings fixed.

## Verification (iteration 2)

All gates run in the fix worktree after the final commit:
- `go build ./...` — green
- `go vet ./...` — green
- `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` — green
- `go test ./...` — green (one load-induced flake of a pre-existing test on the first run, root-caused and hardened in b194d3d; two subsequent full runs green, 37/37 packages)
- `go test ./testdata/golden/...` — green
- `go test ./test/integration/... -count=1` — green
- `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` — green (goleak TestMain gates included)
- Invariants held: BL-01 (nothing recorded under cancelled ctx — untouched), teardown invariants (no goroutine outlives Run; no Sync after lock release; deb.Wait joins every armed flush — the `onWatchOpen`/`syncFn` seams are observability/injection only, no lifecycle change), D-12 serve string byte-identical (now test-pinned), zero new dependencies, `internal/agents/` untouched.

---

_Fixed: 2026-07-16T18:49:21Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
