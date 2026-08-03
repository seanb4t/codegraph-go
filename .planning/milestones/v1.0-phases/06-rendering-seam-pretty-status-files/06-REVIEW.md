---
phase: 06-rendering-seam-pretty-status-files
reviewed: 2026-07-17T00:00:00Z
depth: deep
files_reviewed: 12
files_reviewed_list:
  - internal/cli/present/status.go
  - internal/cli/present/status_test.go
  - internal/cli/present/sanitize.go
  - internal/cli/present/main_test.go
  - internal/cli/present/progress.go
  - internal/cli/present/progress_test.go
  - internal/cli/present/files.go
  - internal/cli/progress_cli.go
  - internal/cli/init.go
  - internal/cli/index.go
  - internal/cli/sync.go
  - internal/cli/present/archtest/import_graph_test.go
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
resolution: "iter-2 CR-01 (signal-handling regression) + WR-05 (its missing test) both resolved by reverting the WR-02 signal handling in commit e2470ea — the signal path no longer exists, so init/index/sync are interruptible again; the original WR-02 cosmetic spinner-on-Ctrl-C is accepted as a documented minor known-issue. WR-01/WR-03/WR-04 remain fixed and confirmed."
---

# Phase 6: Code Review Report (re-run — iteration 2, verifies WR-01..WR-04 fixes)

**Reviewed:** 2026-07-17T00:00:00Z
**Depth:** deep
**Files Reviewed:** 12
**Status:** issues_found

## Summary

This is the `--auto` re-review after commits 50bbeb9 (WR-01), c8b7fe9 (WR-02/goleak),
7aac3f0 (WR-03/dedup), and 455c5d8 (WR-02/signal handling) landed. I re-read every
touched file, re-ran `go build ./...`, `go vet ./...`, and `go test ./internal/cli/...
./internal/cli/present/... ./test/integration/... -race -count=1`, all green, and traced
the call chain and Go stdlib semantics behind the two signal-handling commits rather than
trusting the docstring's own claims.

**Three of the four fixes are correct and fully resolved, with no regressions:**

- **WR-01 (status.go sanitization) — RESOLVED.** `sanitizeControl` is now applied to
  `projectPath` (line 129) and, correctly, to a *copy* of `WorktreeMismatch.WorktreeRoot`/
  `IndexRoot` (lines 138-141) reconstructed into a throwaway `*gitmeta.Mismatch` before
  calling `.Warning()` on it — this is the right adaptation, since `gitmeta.Mismatch` has
  exactly those two fields (confirmed against `internal/gitmeta/detect.go:10-13`) and
  `Warning()`'s own literal `\n` separators (from its message template, not attacker
  input) survive because they're never passed through `sanitizeControl` themselves. The
  new `TestRenderStatus_SanitizesControlChars` fixture-drives both injected-ANSI paths and
  asserts the warning's own newline structure (`"\n  Index from: ...\n"`) survives; it
  passes, as does the pre-existing `TestRenderStatus_WorktreeWarning` (clean-string fast
  path in `sanitizeControl` returns the identical string, so pre-fix behavior for clean
  input is unchanged). I re-traced `r.Backend` and `kc.Key` back to their sources
  (`internal/query/status.go:301` — `Backend` is the fixed literal `"pebble"`, never
  attacker-derived) — no residual unsanitized interpolation site into the pretty sink
  remains in `status.go`. `files.go` was re-checked too and is unchanged/still correct
  (all three sites still wrapped).
- **WR-04 (goleak convention) — RESOLVED.** `internal/cli/present/main_test.go`'s
  `TestMain` now gates the whole package on `goleak.VerifyTestMain(m)`, and
  `progress_test.go`'s old `stabilizedGoroutineCount` polling loop is gone —
  `TestProgress_NoGoroutineLeak` now only asserts goroutine count *increases* while
  running (the "Start launched something" half); the "did it actually go away" half is
  now covered deterministically by the package-wide goleak scan. I ran
  `go test ./internal/cli/present/... -race -count=1 -v`: every test in the package
  passes, including the goleak-gated `TestMain`, confirming no other test in the package
  (files_test.go, sanitize_test.go, tty_test.go, styles/status tests) leaves a stray
  goroutine that would make the new package-wide gate flaky.
- **WR-03 (duplicated spinner wiring) — RESOLVED, mechanically correct.** The five-line
  block is now the single `startProgress` helper in the new `internal/cli/progress_cli.go`,
  and all three call sites (`init.go:66`, `index.go:67`, `sync.go:45`) call it identically
  apart from the label string. `TestProgressCLINonTTYReachability`/
  `TestProgressCLIQuietSuppressesSummary`/`TestProgressWiringDoesNotBreakErrorPaths` in
  `internal/cli/progress_cli_test.go` all pass and confirm the non-TTY/--quiet/error-path
  behavior is byte-for-byte unchanged.

**The fourth fix (WR-02, "install a signal handler so `Progress.Stop()` runs on
Ctrl-C") introduces a new BLOCKER.** See CR-01 below — the fix, as implemented, does not
just fail to abort `indexer.Run`/`Sync` early (an acknowledged, reasonable limitation the
code's own comment names), it silently disables the terminal's normal Ctrl-C-kills-the-
process behavior for the *entire* duration of every interactive `init`/`index`/`sync` run,
directly contradicting a factual claim in its own docstring. I verified this both by
reading the Go 1.26 stdlib source for `signal.NotifyContext` and by an isolated
reproduction (see CR-01) that empirically confirms three successive `SIGINT`s delivered to
a process that mirrors this codebase's exact pattern have zero effect on process
lifetime — the simulated long-running call runs to completion regardless.

Phase invariants re-checked and holding: `internal/query`/`internal/mcp` remain charm-free
(`TestNoCharmInServeReachablePackages` green — `internal/cli` correctly stays excluded from
the closure so the new `progress_cli.go` signal-handling code isn't itself a violation);
`test/integration`'s `TestStatusFilesPlainByteIdentity` (the `StatusFilesPlain` golden)
passes, confirming the plain path is still byte-identical and untouched by any of the four
fixes; `internal/cli/present` and `internal/cli` both pass fully under `-race`.

## Critical Issues

### CR-01: WR-02's signal-handling fix neutralizes Ctrl-C for the entire duration of `init`/`index`/`sync` — a new regression, not present pre-fix

**File:** `internal/cli/progress_cli.go:34-51` (also reached via `internal/cli/init.go:66`, `internal/cli/index.go:67`, `internal/cli/sync.go:45`)
**Issue:** `startProgress` registers `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` and spawns a goroutine that calls the idempotent `prog.Stop()` when the context is canceled — this correctly clears the stray spinner line the original WR-03 finding complained about. But `signal.NotifyContext`'s `stop()` (which is what un-registers the handler and restores the OS default, process-terminating disposition for `SIGINT`/`SIGTERM`) is only called from the closure `startProgress` returns, and that closure is `defer`red at the RunE call site — meaning `stop()` doesn't run until *after* `indexer.Run`/`indexer.Sync` has already returned. Since `indexer.Run`/`Sync` take no `context.Context` parameter at all (confirmed: `func Run(repoRoot, storeDir string, opts Options) (Stats, error)` in `internal/indexer/pipeline.go:76`, `func Sync(repoRoot, storeDir string, opts Options) (Stats, error)` in `internal/indexer/sync.go:34`), cancellation is never threaded into the actual long-running work.

The docstring added by this fix (`progress_cli.go:29-31`) asserts: *"a second signal falls through to the process's normal (now-restored) signal disposition."* This is factually incorrect. Go's `signal.Notify` (which `NotifyContext` wraps — `os/signal/signal.go:289` in the Go 1.26 stdlib, `Notify(c.ch, c.signals...)`) intercepts the named signals at the OS/runtime level and keeps them from reaching the default disposition **for as long as the registration is active**, regardless of how many times the context has already been canceled — restoration only happens when `Stop(c.ch)` is called, which is exactly the `stop()` this code defers until after the run completes. `NotifyContext`'s own internal watcher goroutine (`os/signal/signal.go:290-298`) reads from the signal channel exactly once (a `select` that exits after the first `case s := <-c.ch:`), so from the *second* signal onward there isn't even a consumer left — the first `SIGINT` clears the spinner and is otherwise silently absorbed, and every subsequent `SIGINT`/`SIGTERM` for the rest of the run does *nothing at all*: no cancellation, no process exit, no visible effect.

I confirmed this empirically with an isolated reproduction of the exact pattern (`NotifyContext` + a goroutine that reacts to `ctx.Done()`, wrapping a loop that — like `indexer.Run`/`Sync` — never checks the context):
```
$ ./sigtest & PID=$!; sleep 1; kill -INT $PID; sleep 1; kill -INT $PID; sleep 1; kill -INT $PID; wait $PID
working... 0
working... 1
ctx canceled: spinner-stop equivalent ran   # first SIGINT
working... 2
working... 3
...
working... 19                                # 2nd and 3rd SIGINT: no visible effect at all
done, stop() called
exit code: 0
```
The process ran to completion regardless of three delivered `SIGINT`s — it never terminated early, and no error/exit code reflects the interrupt attempts.

**Practical impact:** this only manifests on the interactive path (TTY, not `--quiet`, no `NO_COLOR`) — precisely the case where a human operator is watching the spinner and is most likely to want to abort a mistakenly-triggered `codegraph index`/`init` on a large monorepo. Pre-fix, pressing Ctrl-C during a long run killed the process immediately (ungracefully, leaving a stray spinner frame — the original WR-03 complaint). Post-fix, Ctrl-C no longer kills the process at all during the run; the operator has no way to abort short of `kill -9`ing the process from another shell. This is a genuine, silent behavioral regression, not merely a residual gap in the original finding's scope, and it is completely uncovered by any test: `progress_cli_test.go`'s own comment (`TestProgressWiringDoesNotBreakErrorPaths`, line ~88-94) states plainly that this exact code path is "out of reach for this in-process, non-TTY harness" — meaning the CI suite cannot and does not catch it.

**Fix:** After the spinner is cleared, re-raise the signal (or otherwise force termination) instead of letting `indexer.Run`/`Sync` run to completion unabated — restore the pre-fix "Ctrl-C kills the process" contract while keeping the new clean-spinner behavior:
```go
func startProgress(ctx context.Context, quiet bool, label string) func() {
	if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
		return func() {}
	}
	prog := present.NewProgress(os.Stderr)
	prog.Start(label)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case sig := <-sigCh:
			prog.Stop() // clear the spinner line first
			signal.Stop(sigCh)
			// Re-raise so the process terminates the way it did
			// pre-fix (indexer.Run/Sync take no ctx and cannot be
			// cancelled early) — only the stray spinner frame is
			// what this fix should have changed.
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(sig)
		case <-done:
		}
	}()

	return func() {
		close(done)
		signal.Stop(sigCh)
		prog.Stop()
	}
}
```
Alternatively, if intentionally choosing to let the run complete rather than kill the process (i.e. keeping today's behavior but making it deliberate rather than accidental), the docstring's incorrect claim must be corrected and a warning line should be printed to stderr on receipt of the signal (e.g. `"interrupt received — finishing current run (Ctrl-C again is not currently supported); this may take a while"`), so the operator isn't left believing their second and third Ctrl-C presses did something. Either way, add a regression test — the non-TTY harness can't drive this, but a small in-process test can call `startProgress` directly with a synthetic `context.Context` (bypassing the TTY gate by calling the unexported constructor logic, or refactoring the signal-wiring into a smaller, directly-testable unit) and assert on the actual termination/re-raise behavior rather than relying only on the "clear-line sequence reaches the writer" manual smoke test the commit message describes.

## Warnings

### WR-05: The SIGINT/SIGTERM code path added by WR-02 has zero automated test coverage

**File:** `internal/cli/progress_cli.go:34-51`, `internal/cli/progress_cli_test.go`
**Issue:** `progress_cli_test.go`'s own doc comment on `TestProgressWiringDoesNotBreakErrorPaths` (lines ~84-94) explicitly states that a "real TTY-driven" signal assertion is "out of reach for this in-process, non-TTY harness" and that `execCmd`-based tests never exercise the pretty/TTY branch at all (`ChoosePresentation`'s TTY gate can't fire under `go test`). As a direct consequence, none of the three tests in that file, nor any test in `internal/cli/present`, ever invokes the `signal.NotifyContext` branch inside `startProgress` — the entire mechanism this iteration's WR-02 fix added is untested, which is exactly how CR-01's regression shipped unnoticed (the commit message describes only a manual, one-off smoke test of the clear-line behavior, not the termination/re-raise behavior).
**Fix:** Refactor the signal-wiring portion of `startProgress` into a smaller unit that accepts an injectable "on signal" callback and a synthetic context/channel, so a test can drive it without a real TTY or an actual OS signal — e.g. extract `func watchForCancel(ctx context.Context, onCancel func())` and unit-test that `onCancel` fires exactly once when `ctx` is canceled, plus a separate integration-style test (using `syscall.Kill(os.Getpid(), syscall.SIGINT)` against a real subprocess spawned via `exec.Command`, similar to how `execCmd` already shells out) that asserts the process's actual exit behavior on interrupt once CR-01 is fixed.

---

_Reviewed: 2026-07-17T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
