---
phase: 06-rendering-seam-pretty-status-files
fixed_at: 2026-07-18T01:08:33Z
review_path: .planning/phases/06-rendering-seam-pretty-status-files/06-REVIEW.md
iteration: 2
findings_in_scope: 4
fixed: 3
reverted: 1
skipped: 0
status: resolved
---

> **Iteration-2 correction (2026-07-18):** The `--auto` re-review found the WR-02 signal-handling
> fix (commit 455c5d8) introduced a BLOCKER regression — `signal.NotifyContext` made
> `init`/`index`/`sync` **uninterruptible** (the work takes no context, so after the first Ctrl-C
> every later one was swallowed). That fix was **reverted** in commit `e2470ea`, restoring Go's
> default terminating disposition. Net outcome: **WR-01, WR-03, WR-04 fixed and confirmed;
> WR-02 reverted** and its original cosmetic issue (stray spinner frame on Ctrl-C) **accepted as a
> minor known-issue** — a correct clear-then-terminate handler needs a `context.Context` threaded
> through `indexer.Run`/`Sync`, deferred as a future enhancement. Final state verified green:
> `go build ./...`, full `go test ./...`, `-race ./internal/cli/present/...`, archtest, golden
> byte-identity, and integration byte-identity all pass; TUI-01/TUI-02 invariants intact.

# Phase 6: Code Review Fix Report

**Fixed at:** 2026-07-18T01:08:33Z
**Source review:** .planning/phases/06-rendering-seam-pretty-status-files/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4
- Fixed: 4
- Skipped: 0

## Fixed Issues

### WR-01: `present/status.go` interpolates two untrusted/host-path strings into the pretty sink without `sanitizeControl`

**Files modified:** `internal/cli/present/status.go`, `internal/cli/present/status_test.go`
**Commit:** `50bbeb9`
**Applied fix:** Adapted the review's suggested fix (which wrapped the *entire* `WorktreeMismatch.Warning()` string in `sanitizeControl`) after discovering that would strip the warning's own intentional embedded newlines — `sanitizeControl` drops all `unicode.IsControl` runes by design, including `\n`/`\t`/`\r`, and the existing `TestRenderStatus_WorktreeWarning` fixture proved this collapses the multi-line warning onto one line. Instead, sanitized a copy of `gitmeta.Mismatch{WorktreeRoot, IndexRoot}` (the two actual untrusted path fields) before calling `.Warning()` on the sanitized copy, so the message template's literal newlines survive while the attacker-controllable path content is stripped. `projectPath` is sanitized directly at its single interpolation site, matching `files.go`'s pattern. Added `TestRenderStatus_SanitizesControlChars`, a regression test with ESC/BEL-laden fixtures in both `WorktreeRoot`/`IndexRoot` and `projectPath`, asserting the injected escape sequences are stripped while the paths and the warning's line structure survive.

### WR-02: `progress_test.go` still uses manual `runtime.NumGoroutine()` polling instead of the repo's `goleak` convention

**Files modified:** `internal/cli/present/main_test.go` (new), `internal/cli/present/progress_test.go`
**Commit:** `c8b7fe9`
**Applied fix:** Added a package-wide `TestMain` gated on `goleak.VerifyTestMain(m)` to `internal/cli/present`, matching `internal/watch/main_test.go`'s and `internal/daemon/soak_test.go`'s existing convention. Removed `stabilizedGoroutineCount`'s timing-sensitive polling loop from `TestProgress_NoGoroutineLeak`, keeping only the synchronous "goroutine count rose while running" assertion — the "goroutine actually exited" half is now covered deterministically by the package-wide goleak gate. Verified with `go test -race -count=1 ./internal/cli/present/...`.

### WR-03: `init.go`/`index.go`/`sync.go` never install a signal handler, so `Progress.Stop()`'s line-clear/goroutine-join never runs on Ctrl-C

**Files modified:** `internal/cli/progress_cli.go`, `internal/cli/init.go`, `internal/cli/index.go`, `internal/cli/sync.go`
**Commit:** `455c5d8`
**Applied fix:** Implemented on top of WR-04's extracted `startProgress` helper rather than triplicating the signal-handling block: `startProgress` now takes a `context.Context`, derives `sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)`, and launches a goroutine that calls the (idempotent) `Progress.Stop()` as soon as the first SIGINT/SIGTERM arrives — clearing the stray spinner line immediately rather than waiting for `indexer.Run`/`Sync` (which has no `ctx` parameter to cancel early) to finish. The returned cleanup func calls `stop()` (unregistering the handler and ending the watcher goroutine) before a final, now-idempotent `prog.Stop()`. All three call sites now pass `cmd.Context()`. Manually verified with a throwaway self-`SIGINT` smoke test (written, run, and deleted — not part of the committed suite, since this repo has no existing precedent for real-signal-delivery unit tests) confirming the clear-line sequence (`\r\x1b[K`) reaches the writer once the signal fires — this is a signal-timing behavior Tier 1/2 syntax verification cannot exercise, so treat as **fixed: requires human verification** (a live `Ctrl-C` during a real `codegraph init`/`index`/`sync` run) before fully trusting it in production.

### WR-04: The spinner-wiring block is duplicated verbatim across `init.go`, `index.go`, and `sync.go`

**Files modified:** `internal/cli/progress_cli.go` (new), `internal/cli/init.go`, `internal/cli/index.go`, `internal/cli/sync.go`
**Commit:** `7aac3f0`
**Applied fix:** Extracted the five-line ChoosePresentation-gate/NewProgress/Start/deferred-Stop block into a shared `startProgress(quiet bool, label string) func()` helper in a new `internal/cli/progress_cli.go`, matching the review's suggested shape. All three call sites replaced with `defer startProgress(quiet, "indexing")()` / `defer startProgress(quiet, "syncing")()`. Behavior-preserving — verified via the existing `progress_cli_test.go` reachability tests (non-TTY no-ANSI, `--quiet` suppression, error-path safety) plus `go build ./...` / `go test ./internal/cli/...`. (Note: this commit's helper signature was `startProgress(quiet, label)`; the immediately following WR-03 commit extended it to `startProgress(ctx, quiet, label)` to add signal handling in one place, per the review's own observation that this refactor "gives WR-03's future signal-handling fix a single point of change instead of three.")

## Skipped Issues

None — all 4 in-scope findings were fixed.

## Invariant Verification

Ran after all four fixes were applied and committed:

- `go build ./...` — pass
- `go test -count=1 ./internal/cli/...` — pass (present, present/archtest, cli all green)
- `go test -count=1 ./testdata/golden/...` — pass (TUI-02 byte-identity intact)
- `go test -count=1 ./test/integration/... -run StatusFilesPlain` — pass (plain path unaffected)
- `go test -race -count=1 ./internal/cli/present/...` — pass (including `archtest`, so TUI-01 ANSI isolation still holds — `internal/query`/`internal/mcp` do not import `charm.land/*`)
- `rg 'bubbletea|bubbles' go.mod` — empty (no TUI framework dependency introduced)
- Full `go test -count=1 ./...`: two unrelated failures observed (`internal/daemon`'s goleak assertion tripping on leftover `pebble` disk-health-ticker goroutines from other packages' stores, and `test/integration`'s `TestLiveEditAutoSyncReachesExplore` hitting a store-lock race) — both reproduced identically on the pre-fix `main` branch tip (`e18999d`) and pass reliably in isolation (`go test -count=1 ./internal/daemon/...` and `./internal/daemon/... ./test/integration/...` together both green). Confirmed as pre-existing full-suite-load flakes unrelated to this phase's changes (no file in `internal/daemon`, `internal/query`, or `internal/mcp` was touched by any of the four fixes).

---

_Fixed: 2026-07-18T01:08:33Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
