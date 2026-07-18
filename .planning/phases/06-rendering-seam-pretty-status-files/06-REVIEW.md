---
phase: 06-rendering-seam-pretty-status-files
reviewed: 2026-07-17T00:00:00Z
depth: deep
files_reviewed: 19
files_reviewed_list:
  - internal/cli/present/archtest/import_graph_test.go
  - internal/cli/present/styles.go
  - internal/cli/present/tty.go
  - internal/cli/present/tty_test.go
  - internal/cli/present/status.go
  - internal/cli/present/status_test.go
  - internal/cli/present/files.go
  - internal/cli/present/files_test.go
  - internal/cli/present/sanitize.go
  - internal/cli/present/sanitize_test.go
  - internal/cli/present/progress.go
  - internal/cli/present/progress_test.go
  - internal/cli/status.go
  - internal/cli/files.go
  - internal/cli/init.go
  - internal/cli/index.go
  - internal/cli/sync.go
  - internal/cli/progress_cli_test.go
  - test/integration/status_files_plain_test.go
  - go.mod
findings:
  critical: 0
  warning: 4
  info: 0
  total: 4
status: issues_found
---

# Phase 6: Code Review Report (re-run — verifies the CR-01 fix)

**Reviewed:** 2026-07-17T00:00:00Z
**Depth:** deep
**Files Reviewed:** 19
**Status:** issues_found

## Summary

This is a re-run after the prior deep review's CR-01 (ANSI/OSC escape injection via unsanitized filesystem-derived strings in the pretty renderer) was fixed. I verified the fix directly and traced every string interpolation site in the pretty (TTY) rendering path across `internal/cli/present` and its three call sites (`status.go`, `files.go`, `init.go`/`index.go`/`sync.go`).

**CR-01 fix verification: correct where applied, but coverage is incomplete.**
- `sanitizeControl` (`internal/cli/present/sanitize.go`) is implemented correctly: it strips every Unicode `Cc` control character, which — because Go's `unicode.IsControl` fast-paths on the Latin-1 range and the entire `Cc` category (C0 `0x00-0x1F`, DEL `0x7F`, C1 `0x80-0x9F`) lies within that range — catches both 7-bit ESC-prefixed sequences (`\x1b[...`) and 8-bit single-byte CSI/OSC introducers (`0x9B`/`0x9D`). The fast path (`strings.ContainsFunc` before `strings.Map`) correctly preserves reference-equality for clean strings, and is directly tested (`TestSanitizeControl_CleanStringIdentity`).
- `internal/cli/present/files.go` wires `sanitizeControl` at all three points where filesystem-derived names reach the pretty sink (`writeFileTree`'s directory and leaf branches, `RenderFiles`'s flat-format branch) — this closes the original CR-01 vector completely for the `files` command.
- `internal/cli/present/status.go`, however, was **not** revisited for the same class of injection: it interpolates `r.WorktreeMismatch.Warning()` (which embeds two git-derived filesystem paths) and `projectPath` directly, unsanitized, into the same pretty TTY sink `files.go` was fixed to protect. See WR-01 below. I assessed the practical exploitability of these two specific strings as lower than the original `n.Name`/`f.Path` vector (worktree/index roots and the CLI's own start path are local directory paths the operator placed, not content parsed out of a possibly-untrusted repository's tracked files), so I classified this WARNING rather than re-opening a BLOCKER — but it is a real, demonstrable gap in the stated invariant ("every untrusted-string interpolation into the pretty path" is sanitized) and should be closed for consistency and defense-in-depth. I also traced `kc.Key` (node-kind/language breakdown keys) and `r.Backend` back to their sources (`internal/query/status.go:230-249`) and confirmed they are fixed-vocabulary enum values (extension-based language classification, parser-assigned symbol kinds), not attacker-influenced content — no finding there.

**Prior WARNINGs (WR-01/02/03 in the previous review) re-assessed: all three remain unfixed**, confirmed by direct inspection — renumbered WR-02/03/04 below since the new status.go gap is the most directly on-topic finding for this re-run.

No BLOCKER-severity issues were found. `TUI-01` (ANSI isolation) and `TUI-02` (byte-identity) both still hold: `internal/cli/present/archtest/import_graph_test.go`'s closure walk correctly excludes `internal/cli` and includes all six serve-reachable roots with a working self-defeat guard, and `test/integration/status_files_plain_test.go` proves the plain path is unaffected by `NO_COLOR`/pretty-branch wiring. `internal/cli/status.go`'s and `internal/cli/files.go`'s plain-output branches are confirmed untouched (still calling `query.RenderStatusText`/`printFileTree` directly — neither references `sanitizeControl`), consistent with the frozen-plain-path requirement.

## Warnings

### WR-01: `present/status.go` interpolates two untrusted/host-path strings into the pretty sink without `sanitizeControl` — CR-01 fix coverage is incomplete

**File:** `internal/cli/present/status.go:125` and `internal/cli/present/status.go:127-129`
**Issue:** `RenderStatus` writes `projectPath` (line 125) and `r.WorktreeMismatch.Warning()` (lines 127-129) straight into the `strings.Builder` that becomes the pretty TTY sink, with no `sanitizeControl` call — even though `internal/cli/present/files.go` was specifically fixed to wrap every comparable interpolation site. `Warning()` (`internal/gitmeta/notice.go:19-29`) embeds `m.WorktreeRoot` and `m.IndexRoot`, both real filesystem paths resolved via `git rev-parse`-equivalent calls in `internal/gitmeta/detect.go`; `internal/query/status.go`'s own doc comment (line 35) explicitly flags this as "a scoped exception" that "intentionally carries absolute host paths" — a comment written before the CR-01 escape-injection class of bug was identified, so it never considered the terminal-injection angle. `projectPath` is `resolveStartPath`'s result (`--path` flag value or `os.Getwd()`), also unsanitized. Neither is covered by `status_test.go`'s fixtures (`TestRenderStatus_WorktreeWarning` only exercises clean ASCII paths `/a/main` and `/a/main/.claude/worktrees/probe`), so this gap is untested as well as unfixed.

Practical exploitability is lower than the original `n.Name`/`f.Path` vector `files.go` fixed — these two strings are local directory paths the operator placed on disk (via `git clone`/`git worktree add`/`cd`), not content parsed out of a possibly-adversarial repository's tracked files — but the mechanism is identical (an unsanitized string reaching the same lipgloss-styled TTY `io.Writer`), and closing it is a small, mechanical, low-risk change consistent with the fix already applied next door in `files.go`.

**Fix:** Apply `sanitizeControl` at both sites, mirroring `files.go`'s pattern:
```go
// internal/cli/present/status.go
fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Project:"), sanitizeControl(projectPath))

if warning := r.WorktreeMismatch.Warning(); warning != "" {
	b.WriteString(sanitizeControl(warning) + "\n")
}
```
Add a fixture-driven regression test in `status_test.go` (a `WorktreeMismatch` with an embedded `\x1b[` or control byte in `WorktreeRoot`/`IndexRoot`, and a `projectPath` containing one) asserting the ANSI-stripped output no longer contains raw control bytes, mirroring `sanitize_test.go`'s cases.

### WR-02: `progress_test.go` still uses manual `runtime.NumGoroutine()` polling instead of the repo's `goleak` convention

**File:** `internal/cli/present/progress_test.go:121-157` (`TestProgress_NoGoroutineLeak`, `stabilizedGoroutineCount`)
**Issue:** Unchanged from the prior review. `go.uber.org/goleak` is already a direct dependency (`go.mod:28`) and is the established convention for goroutine-leak assertions elsewhere in this codebase (`internal/daemon/soak_test.go:16-22` and `internal/watch/main_test.go:9-17` both use `goleak.VerifyTestMain(m)` at the package `TestMain` level). `progress_test.go` instead hand-rolls a polling loop (`stabilizedGoroutineCount`) that samples `runtime.NumGoroutine()` for up to 500ms and compares against a baseline — this is inherently flaky under load (any unrelated background goroutine from the Go runtime, GC, or a parallel test in the same binary can transiently push the count above baseline within the poll window) and diverges from the pattern already established elsewhere in this codebase for exactly this kind of assertion.
**Fix:** Add a `TestMain` to the `present` package gated on `goleak.VerifyTestMain(m)` (matching `internal/watch/main_test.go`'s shape exactly), and delete `stabilizedGoroutineCount`/`TestProgress_NoGoroutineLeak`'s manual polling in favor of the package-wide goleak gate — `Stop()`'s existing `close(stopCh); <-doneCh` join already guarantees the goroutine has exited by the time `Stop()` returns, so goleak's post-test-run scan will pass deterministically instead of via a timing-sensitive poll.

### WR-03: `init.go`/`index.go`/`sync.go` never install a signal handler, so `Progress.Stop()`'s line-clear/goroutine-join never runs on Ctrl-C

**File:** `internal/cli/init.go:71-75`, `internal/cli/index.go:69-73`, `internal/cli/sync.go:47-51`
**Issue:** Unchanged from the prior review. Each of these three call sites does `defer prog.Stop()` immediately after `prog.Start(...)`, relying on Go's normal deferred-function unwind to clear the spinner line and join the ticker goroutine when `RunE` returns (including on `indexer.Run`/`indexer.Sync` error). But none of the three commands ever calls `signal.NotifyContext` — contrast with `internal/cli/daemon.go:51-52`, which explicitly does `ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM); defer stop()` for exactly this reason. On SIGINT (Ctrl-C) during a long-running `indexer.Run`/`indexer.Sync` call, the Go runtime's default signal disposition terminates the process directly — deferred functions, including `prog.Stop()`, never execute. The user is left with a stray, un-cleared spinner frame (and a cursor potentially left mid-line) on their terminal after every interrupted `init`/`index`/`sync`.
**Fix:** Thread a `signal.NotifyContext`-derived context through, and register a handler that calls `prog.Stop()` explicitly before letting the interrupt propagate, e.g.:
```go
ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
// ... existing spinner wiring, but also:
go func() {
	<-ctx.Done()
	prog.Stop() // idempotent (Progress.Stop already tolerates double-call)
}()
```
or equivalent — the goal is simply ensuring `Progress.Stop()`'s line-clear runs before the process actually exits on an interrupt, matching `daemon.go`'s existing pattern for the same concern.

### WR-04: The spinner-wiring block is duplicated verbatim across `init.go`, `index.go`, and `sync.go`

**File:** `internal/cli/init.go:71-75`, `internal/cli/index.go:69-73`, `internal/cli/sync.go:47-51`
**Issue:** Unchanged from the prior review. The same five-line block (TTY/`NO_COLOR` gate via `present.ChoosePresentation`, `present.NewProgress(os.Stderr)`, `.Start(label)`, `defer .Stop()`) is copy-pasted three times, differing only in the spinner label string (`"indexing"` in `init.go`/`index.go`, `"syncing"` in `sync.go`). Any future fix to this wiring (e.g. WR-03's signal-handling fix above) has to be applied identically in three places, and none of the three copies has drifted yet, but nothing enforces that they stay in sync.
**Fix:** Extract a shared helper, e.g. in a new `internal/cli/progress_cli.go`:
```go
// startProgress wires the TTY-gated stderr spinner shared by init/index/sync
// (TUI-05/D-07/D-08). Returns a cleanup func the caller must defer
// unconditionally (a no-op when the pretty branch didn't fire).
func startProgress(quiet bool, label string) func() {
	if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
		return func() {}
	}
	prog := present.NewProgress(os.Stderr)
	prog.Start(label)
	return prog.Stop
}
```
and replace each of the three call sites with `defer startProgress(quiet, "indexing")()` / `defer startProgress(quiet, "syncing")()`. This also gives WR-03's future signal-handling fix a single point of change instead of three.

---

_Reviewed: 2026-07-17T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
