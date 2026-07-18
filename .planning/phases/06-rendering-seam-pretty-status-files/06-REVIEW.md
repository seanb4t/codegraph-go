---
phase: 06-rendering-seam-pretty-status-files
reviewed: 2026-07-17T00:00:00Z
depth: deep
files_reviewed: 18
files_reviewed_list:
  - internal/cli/present/archtest/import_graph_test.go
  - internal/cli/present/styles.go
  - internal/cli/present/tty.go
  - internal/cli/present/tty_test.go
  - internal/cli/present/status.go
  - internal/cli/present/status_test.go
  - internal/cli/present/files.go
  - internal/cli/present/files_test.go
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
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-07-17T00:00:00Z
**Depth:** deep
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Reviewed the rendering-seam phase (ANSI isolation, TTY/NO_COLOR gating, styled status/files renderers, and the hand-rolled stderr progress spinner) at deep depth, including cross-file tracing between `internal/cli/present` and its call sites in `internal/cli`, and a byte-for-byte structural comparison against the frozen `internal/query` plain renderers this phase must not modify.

The archtest (`import_graph_test.go`) correctly targets the `/v2`-suffixed charm paths, correctly excludes `internal/cli` from the walked closure with a documented rationale, and its self-defeat guard (`assertCharmImporterExists`) is real — it scans the whole module including test files and fails closed if nothing imports `charm.land/lipgloss/v2`. `go build ./...`, `go vet`, and `go test -race ./internal/cli/present/...` all pass. `ChoosePresentation` is pure as required (no `os.Getenv`/`term.IsTerminal` inside `present`). `present.RenderStatus`/`present.RenderFiles` are structurally line-for-line parity with `query.RenderStatusText`/`printFileTree` (verified by direct comparison of `writeStatLine`/`writeBreakdownText`/`writeFileTree` against the frozen originals) — no byte-identity divergence found. `Progress.Stop()` is race-free under `-race` and correctly blocks on `doneCh` before returning.

The one BLOCKER-level finding is a genuine escape-sequence-injection gap: file/directory names sourced from an indexed (potentially adversarial) repository flow unsanitized into the new pretty renderer's styled terminal output. The WARNING-level findings are quality/robustness gaps in the new spinner wiring (weaker goroutine-leak test convention than the rest of the codebase uses, no interrupt-safety story for the spinner's line-clear guarantee, and triplicated wiring code).

## Critical Issues

### CR-01: Unsanitized repo-controlled file names/paths enable terminal escape-sequence injection in the pretty `files` renderer

**File:** `internal/cli/present/files.go:17-26` (`writeFileTree`), `internal/cli/present/files.go:35-46` (`RenderFiles`)
**Issue:** `n.Name`, `n.Name+"/"`, and `f.Path` are interpolated directly into the styled output stream with no control-character filtering:
```go
fmt.Fprintf(b, "%s%s\n", indent, headerStyle.Render(n.Name+"/"))
...
fmt.Fprintf(b, "%s%s (%s)\n", indent, n.Name, labelStyle.Render(n.Language))
...
fmt.Fprintf(&b, "%s (%s)\n", f.Path, labelStyle.Render(f.Language))
```
`FileTreeNode.Name` and `FileEntry.Path` are populated straight from the on-disk file tree walked during indexing (`internal/query/files.go:139,183`) with no sanitization anywhere in the pipeline. POSIX filenames may contain any byte except `NUL` and `/`, including raw `ESC` (`0x1b`). A crafted/adversarial repository (this project's own threat model explicitly anticipates "arbitrary, occasionally adversarial ... third-party monorepo code," per `.claude/CLAUDE.md`'s Parser Decision section) can therefore contain a file named e.g. `innocent\x1b]0;pwned\x07.go`. When a user runs `codegraph files` (or `files --format tree`) on a real TTY, that raw escape sequence is written verbatim to the terminal and interpreted by the terminal emulator — enabling OSC/CSI injection (title-bar spoofing, and on some emulators OSC 52 clipboard writes or other terminal-specific side effects). This is new exposure introduced by this phase's pretty-rendering path: the pre-existing plain renderer (`internal/cli/files.go`'s `printFileTree`) has the identical gap but is explicitly frozen/out-of-scope for this phase ("must NOT be modified" per the plain byte-identity contract) — `present/files.go` is new code that could and should have added sanitization without touching the frozen plain path.
**Fix:** Strip or escape non-printable/control bytes (anything `< 0x20` except none needed here, plus `DEL`/`0x7f`) from `n.Name` and `f.Path` before interpolating them into the pretty output, e.g.:
```go
func sanitizeForTerminal(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '?'
		}
		return r
	}, s)
}
```
and apply it at the two interpolation points in `writeFileTree`/`RenderFiles`. Consider filing a follow-up to apply the same sanitization to `internal/cli/files.go`'s `printFileTree` and `internal/query`'s plain renderers, which share the same unmitigated exposure but are out of this phase's blast radius.

## Warnings

### WR-01: `Progress` goroutine-leak test uses ad hoc polling instead of the codebase's established `goleak` convention

**File:** `internal/cli/present/progress_test.go:121-157`
**Issue:** `internal/watch` and `internal/daemon` — the two other packages in this codebase that launch background goroutines — both gate their entire test package on `goleak.VerifyTestMain(m)` in a `TestMain` (see `internal/watch/main_test.go:17`, `internal/daemon/soak_test.go:22`), which identifies leaked goroutines precisely (by stack trace) and is depended on (`go.uber.org/goleak` is already a `go.mod` requirement). `present`'s new `Progress` type also launches a background goroutine, but its leak test (`TestProgress_NoGoroutineLeak`) instead does manual `runtime.NumGoroutine()` before/after comparisons with a 500ms polling settle window (`stabilizedGoroutineCount`). This is weaker (can't distinguish a Progress leak from any other concurrently-running test's goroutine, and the tolerance window can mask a slow leak or produce a flaky pass/fail under load) and inconsistent with the pattern the rest of the codebase already uses for exactly this kind of assertion.
**Fix:** Add a `TestMain` to `internal/cli/present` that wraps the package's tests in `goleak.VerifyTestMain(m)`, and drop the hand-rolled `stabilizedGoroutineCount`/`NumGoroutine` comparison in favor of the standard `goleak.VerifyNone(t)` (or the package-wide `TestMain` form already established elsewhere).

### WR-02: Spinner's "always clears the line" guarantee does not survive process interruption (SIGINT/SIGTERM)

**File:** `internal/cli/init.go:71-75`, `internal/cli/index.go:69-73`, `internal/cli/sync.go:47-51`
**Issue:** `Progress.Stop()`'s deterministic cleanup (clear the line, join the ticker goroutine) is only reached via Go's `defer` on a normal return or panic unwind. None of `init`/`index`/`sync` install `signal.NotifyContext` or any other interrupt handling (unlike `internal/cli/daemon.go:51`, which does use `signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)`). If a user hits Ctrl-C while a long `codegraph init`/`index`/`sync` run is in progress, the Go runtime's default SIGINT handling terminates the process immediately without running deferred calls, leaving the spinner's last partial frame (and a mid-line cursor position) stuck on the user's terminal. This isn't a correctness bug in `Progress` itself (its unit tests correctly prove `Stop()` is race-free and leak-free on the paths that do call it) but it means the phase's documented guarantee ("Stop() must not race/leak the ticker goroutine ... no goroutine is ever left running after Stop returns") has a real gap in the one scenario (user interrupt during the exact long-running operation the spinner exists for) where terminal-state cleanliness matters most.
**Fix:** Either wire `signal.NotifyContext` into `init`/`index`/`sync` (mirroring `daemon.go`) and call `prog.Stop()` in the signal path before exiting, or explicitly document this as an accepted v1 limitation if `indexer.Run`/`indexer.Sync` are not currently context-cancellable.

### WR-03: Spinner wiring block duplicated verbatim across three call sites

**File:** `internal/cli/init.go:71-75`, `internal/cli/index.go:69-73`, `internal/cli/sync.go:47-51`
**Issue:** The exact same five-line block (`ChoosePresentation` gate against `os.Stderr`'s fd + `os.Getenv("NO_COLOR")`, `present.NewProgress(os.Stderr)`, `.Start(label)`, `defer prog.Stop()`) is copy-pasted into `init.go`, `index.go`, and `sync.go`, differing only in the label string (`"indexing"` vs `"syncing"`). Any future change to this gating logic (e.g., WR-02's fix) must be applied identically in three places, and nothing enforces that.
**Fix:** Extract a small helper, e.g. `func maybeStartProgress(quiet bool, label string) (stop func())` in `internal/cli`, and call it once per command.

## Info

### IN-01: `fmt.Fprintf` used where `strings.Builder.WriteString` would be marginally cheaper

**File:** `internal/cli/present/status.go:84,94,104,110,125`
**Issue:** Several of these `Fprintf` calls format simple string concatenations that could be written with `WriteString`/string concatenation instead, avoiding `fmt`'s reflection-based format-string parsing.
**Fix:** Not worth changing — `RenderStatus` runs at most once per CLI invocation over a handful of lines; the overhead is immaterial. No action needed.

### IN-02: Archtest's `internal/cli` exclusion has a theoretical coverage gap (inherited precedent, not a phase-6 defect)

**File:** `internal/cli/present/archtest/import_graph_test.go:73-91`
**Issue:** `isModuleInternalPackage` stops the closure walk from descending into `internal/cli` (needed so `internal/cli/present`'s own charm import isn't treated as a violation). A side effect: if a guarded package (e.g. `internal/query`) ever came to import something under `internal/cli` transitively through an intermediate, non-excluded package, the forbidden-import check would not catch the resulting charm reachability, because the check only inspects each *reachable* package's own direct `Imports` map, and the intermediate package's own import statement (of `internal/cli/...`, not of `charm.land/...` directly) would not itself match `forbiddenImportPaths`. This mirrors an already-accepted precedent (`internal/graphstore/archtest`'s equivalent D-06b exclusion) rather than being a new gap introduced by this phase, and the architectural assumption (none of the six guarded packages should ever import `internal/cli` for unrelated layering reasons) is reasonable. Flagged for awareness only — no action required.

---

_Reviewed: 2026-07-17T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
