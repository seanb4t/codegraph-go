---
phase: 03-watcher-on-mcp-default
reviewed: 2026-07-16T16:05:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - internal/graphstore/pebble_store.go
  - internal/graphstore/locked_unix.go
  - internal/graphstore/locked_windows.go
  - internal/graphstore/open_lock_test.go
  - internal/graphstore/locked_unix_test.go
  - internal/graphstore/locked_windows_test.go
  - internal/daemon/daemon.go
  - internal/cli/serve.go
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 3: Code Review Report (Round 3 — targeted delta review of commits 7699e9c + 1bdcd9c)

**Reviewed:** 2026-07-16T16:05:00Z
**Depth:** standard (targeted: round-2 CR-01/WR-01/WR-02 fix delta only)
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Targeted final-round review of the ErrStoreLocked sentinel refactor (7699e9c) and its direct unit tests (1bdcd9c), verified against the pinned `pebble/v2@v2.1.6` source and by actually cross-compiling and running the code. The core fix is sound: all five checkpoints from the tasking hold. One Warning remains — the CI compile gate that `locked_windows_test.go`'s own comment claims protects the windows arm does not exist in `.github/workflows/ci.yml` — plus three Info items (a factually wrong provenance claim in a test comment, a timing-margin note on the convergence test, and an accepted windows classification imprecision).

**What was verified (executed, not assumed):**

1. **The windows classifier compiles and semantically matches pebble v2.1.6.** `GOOS=windows GOARCH=amd64 go build ./internal/graphstore/`, `GOOS=windows go vet ./internal/graphstore/` (which typechecks `locked_windows_test.go` and `open_lock_test.go` too), and `GOOS=windows GOARCH=arm64 go build` all pass locally. Semantics verified against the pinned source: `vfs/file_lock_windows.go:26-41` acquires the LOCK via `windows.CreateFile(..., shareMode=0, CREATE_ALWAYS, ...)`; x/sys/windows' generated wrapper returns the failure as a **bare `syscall.Errno`** (its `errnoErr` returns the errno itself; `ERROR_SHARING_VIOLATION` is typed `syscall.Errno = 32` in `zerrors_windows.go`), and the chain back to `pebble.Open`'s caller is verbatim — `diskHealthCheckingFS.Lock` (vfs/disk_health.go:790-792) delegates without wrapping, `LockDirectory` (open.go:1174-1177) and `Open` (open.go:129-132) return the error unchanged. So `errors.Is(err, syscall.Errno(32))` in `locked_windows.go:28` matches by direct equality at the top of the chain. `const errSharingViolation = syscall.Errno(32)` is a valid constant (Errno is uintptr-based) and correctly avoids importing x/sys/windows. No windows-only identifier leaks into shared code: `pebble_store.go` references only `isLockHeldOS`, defined in both build-tagged files; host `go build ./...` and `go vet` pass.
2. **Classification is confined to pebble.Open provenance.** `classifyOpenError` has exactly one call site (`pebble_store.go:142`, inside `Open`'s loop, on `pebble.Open`'s own error) and `isLockHeldOS` is called only from `classifyOpenError` — verified by grep over the whole module; `IsLockHeld` is fully deleted with zero stragglers. Every Sync-chain path propagates `graphstore.Open`'s error **unwrapped** (sync.go:38-40 via `needsFileIndexBackfill`, sync.go:52-55 main open, sync.go:470-474, pipeline.go:116-119; `daemon.flush` returns Sync's error raw), so the sentinel survives to the consumers, and no Sync-chain EACCES (Discover/WalkDir/contentHash) ever passes through the classifier. The unix classifier deliberately excludes EACCES (so pebble's own `os.Create(LOCK)` permission failure at vfs/file_lock_unix.go:52-54 stays fatal), pinned by tests on both the platform arm and the shared path.
3. **Consumers are correct.** `daemon.go:214` requeues on `errors.Is(err, graphstore.ErrStoreLocked) && ctx.Err() == nil` — the `n <= maxFlushLockRequeues` bound, the ctx gate, and the BL-01 invariant (a cancelled or non-lock failure falls through the switch recording nothing; sidecar left in place) are all preserved verbatim from round 2. `serve.go:201` restores the stated contract: `if !errors.Is(err, graphstore.ErrStoreLocked) { return err }` — every non-lock reconcile error (including permission errors anywhere in Sync's chain) is fatal again; only genuine lock contention degrades to the stderr warning.
4. **The new tests pin what they claim and pass** (`go test ./internal/graphstore/` — all green, 1.0s). The double-open test exercises the real platform lock shape end-to-end (in-process string form on unix — a pebble reword now turns this red at unit speed on every CI run, since graphstore is in the main `go test` step), asserts the sentinel wrap, and asserts only a **lower** bound on elapsed time (`(openLockRetryAttempts-1) × openLockRetryBackoff`) — deterministic, cannot flake slow. The holder-release test pins retry convergence (see IN-02 for its timing margin). The EACCES false-pin exists three times: unix arm, windows arm (ERROR_ACCESS_DENIED analogue), and the platform-neutral shared path, which also pins identity pass-through of non-lock errors (`got != eacces` check) so no spurious wrapping can creep in.
5. **No collateral damage to other Open callers.** `migrate.checkTargetOverwrite` (migrate.go:378) branches only on `openErr == nil`; a lock-held probe now returns the sentinel-wrapped error — still non-nil, same refusal semantics, timing unchanged from round 2 (IN-03 there stands as noted). `query/engine.go:166`, `indexer/pipeline.go:116`, and the CLI one-shots propagate the error generically; the only observable change is the improved error text prefix (`graphstore: store lock held: ...`), which preserves the original pebble text for diagnostics (pinned by `TestClassifyOpenErrorWrapsUnixLockForms`).

Round-2 Info items IN-01 through IN-05 were explicitly out of this fix's scope and remain open as documented there (spot-checked: the `n-1` give-up log and no-reset-after-exhaustion at daemon.go:218 are unchanged).

## Warnings

### WR-01: The `GOOS=windows go vet` gate that `locked_windows_test.go` cites as guarding the windows arm does not exist in CI — the windows classifier has no automated protection against regression

**File:** `internal/graphstore/locked_windows_test.go:18-20` (claim); `.github/workflows/ci.yml` (gap — no `GOOS=windows` step anywhere)
**Issue:** The windows test file's doc comment states: *"on other platforms the classifier's cross-GOOS integrity is held by `GOOS=windows go vet ./internal/graphstore/` plus the platform-neutral tests in open_lock_test.go."* No such step exists in `.github/workflows/ci.yml` (verified — the workflow's only jobs are test/govulncheck/reproducibility/perf-regression, all `ubuntu-latest`, no cross-GOOS build or vet), and there is no windows runner, so `locked_windows_test.go` never executes anywhere. The windows arm compiles and vets clean **today** (verified locally for windows/amd64 and windows/arm64 during this review), but the exact failure mode round 2 rated Critical — the windows leg of the lock handling silently rotting while unix CI stays green — is currently prevented only by a comment describing a check nobody runs. A future pebble bump, x/sys change, or refactor that breaks `locked_windows.go`'s compile or semantics would ship undetected to both first-class windows release targets. Round 2's CR-01 fix guidance explicitly called for the compile check to exist, not merely be referenced.
**Fix:** Add the one-line gate to the `test` job in `.github/workflows/ci.yml` (cheap — typechecks both windows-tagged source and test files without needing a windows runner or CGo):

```yaml
      - name: Cross-GOOS typecheck of the windows lock classifier (03-REVIEW-3.md WR-01)
        run: GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/
```

(Or, if the comment's claim is intentionally aspirational, reword it — but adding the step is strictly better and costs seconds.)

## Info

### IN-01: `locked_unix_test.go`'s provenance comment is factually wrong about pebble's cross-process error shape — the fcntl form is a bare errno, not a `PathError`-wrapped one

**File:** `internal/graphstore/locked_unix_test.go:26-27` (and the case name at :28)
**Issue:** The comment claims *"Pebble's vfs surfaces the errno wrapped; errors.Is must traverse"* and the case synthesizes `fs.PathError{Op: "fcntl", ...}`. In the pinned `pebble/v2@v2.1.6`, `unix.FcntlFlock` returns the **bare** errno (x/sys/unix's `errnoErr` returns the Errno itself), `vfs/file_lock_unix.go:63-65` returns it verbatim, and nothing between there and `pebble.Open`'s caller wraps it (`diskHealthCheckingFS.Lock` delegates; `LockDirectory`/`Open` return unchanged). The real shape is therefore the bare errno — which the suite does cover, via the "bare EWOULDBLOCK" case (EWOULDBLOCK == EAGAIN on every shipped unix target), so matching is functionally correct and fully pinned either way. Only the stated provenance is wrong; a future maintainer trusting it could draw incorrect conclusions about which pebble internals are load-bearing.
**Fix:** Reword the comment: the PathError case tests that traversal *would* work if pebble ever wrapped the errno; the bare-errno case is the real pinned shape.

### IN-02: `TestOpenConvergesWhenHolderCloses` is time-based (150ms release vs ~400ms budget) rather than event-synchronized — small residual flake window under heavy parallel CI load

**File:** `internal/graphstore/open_lock_test.go:56-84`
**Issue:** The holder releases via a `time.Sleep(150 * time.Millisecond)` goroutine while `Open`'s retry budget spans attempts at t≈0/100/200/300/400ms. The ~250ms scheduling margin is generous, and process-wide slowness stretches both sides roughly equally, but the test runs inside CI's parallel full-suite step (`go test "${pkgs[@]}"`) — the same environment in which this repo has already documented time-sensitive tests flaking (internal/daemon's isolated `-count=1` step exists for exactly that reason). If the close goroutine alone is delayed past the final attempt, the test fails spuriously.
**Fix:** None required now; if it ever flakes, synchronize on the event instead of time (e.g., close the holder from a goroutine gated on the first lock failure, or widen `openLockRetryAttempts`' budget in the test via a test hook) rather than moving it to the isolated step.

### IN-03: On windows, ANY `ERROR_SHARING_VIOLATION` escaping `pebble.Open` — not just the LOCK-file collision — classifies as ErrStoreLocked

**File:** `internal/graphstore/locked_windows.go:27-29`
**Issue:** `errors.Is(err, Errno(32))` matches a sharing violation from any file `pebble.Open` touches, e.g. an antivirus/indexer holding MANIFEST or a WAL segment — a real-world occurrence on windows. Consequence: 5 in-call retries (appropriate — AV holds are transient) and, if persistent, `serve`'s reconcile degrades to the "likely an in-flight sync" warning instead of failing fast with a scary-but-accurate error. This is the accepted imprecision of shape-based classification within Open provenance; the wrap preserves the original pebble error text, so the true cause stays diagnosable in the warning line. The unix arm has the same theoretical property for a non-lock EAGAIN inside `pebble.Open`, which is even less plausible.
**Fix:** None — recording the accepted trade-off so it is a decision, not a surprise. If windows AV-contention reports ever materialize, tightening to `*fs.PathError`s whose `Path` basename is `LOCK` is the escalation path.

---

_Reviewed: 2026-07-16T16:05:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard (targeted delta: commits 7699e9c, 1bdcd9c)_
