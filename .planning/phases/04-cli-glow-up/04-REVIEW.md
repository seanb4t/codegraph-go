---
phase: 04-cli-glow-up
reviewed: 2026-09-18T00:30:41Z
depth: standard
files_reviewed: 77
files_reviewed_list:
  - docs/CLI-REFERENCE.md
  - go.mod
  - internal/cli/affected.go
  - internal/cli/callees.go
  - internal/cli/callers.go
  - internal/cli/cli_reference_test.go
  - internal/cli/colorflag_test.go
  - internal/cli/colorflag.go
  - internal/cli/daemon.go
  - internal/cli/explore.go
  - internal/cli/files.go
  - internal/cli/githooks.go
  - internal/cli/impact.go
  - internal/cli/init.go
  - internal/cli/install_test.go
  - internal/cli/install.go
  - internal/cli/node.go
  - internal/cli/plain_golden_test.go
  - internal/cli/present/ansistrip_test.go
  - internal/cli/present/archtest/import_graph_test.go
  - internal/cli/present/explore_test.go
  - internal/cli/present/explore.go
  - internal/cli/present/files_test.go
  - internal/cli/present/files.go
  - internal/cli/present/help_test.go
  - internal/cli/present/help.go
  - internal/cli/present/line_test.go
  - internal/cli/present/line.go
  - internal/cli/present/node_test.go
  - internal/cli/present/node.go
  - internal/cli/present/palette_test.go
  - internal/cli/present/palette.go
  - internal/cli/present/results_test.go
  - internal/cli/present/results.go
  - internal/cli/present/status_test.go
  - internal/cli/present/status.go
  - internal/cli/present/styles.go
  - internal/cli/present/tty.go
  - internal/cli/root.go
  - internal/cli/search.go
  - internal/cli/serve.go
  - internal/cli/short_flags_test.go
  - internal/cli/status.go
  - internal/cli/sync.go
  - internal/cli/telemetry.go
  - internal/cli/testdata/plain/affected-empty.golden
  - internal/cli/testdata/plain/callees.golden
  - internal/cli/testdata/plain/callers.golden
  - internal/cli/testdata/plain/daemon-list-empty.golden
  - internal/cli/testdata/plain/daemon-stop-nomatch.golden
  - internal/cli/testdata/plain/daemon-unlock.golden
  - internal/cli/testdata/plain/explore.golden
  - internal/cli/testdata/plain/files-flat.golden
  - internal/cli/testdata/plain/files-tree.golden
  - internal/cli/testdata/plain/githooks-install.golden
  - internal/cli/testdata/plain/githooks-remove.golden
  - internal/cli/testdata/plain/githooks-status.golden
  - internal/cli/testdata/plain/impact.golden
  - internal/cli/testdata/plain/index-force.golden
  - internal/cli/testdata/plain/init.golden
  - internal/cli/testdata/plain/install-local.golden
  - internal/cli/testdata/plain/node-file.golden
  - internal/cli/testdata/plain/node-symbol.golden
  - internal/cli/testdata/plain/search-full.golden
  - internal/cli/testdata/plain/search.golden
  - internal/cli/testdata/plain/serve-mcp-stderr.golden
  - internal/cli/testdata/plain/status.golden
  - internal/cli/testdata/plain/sync.golden
  - internal/cli/testdata/plain/telemetry.golden
  - internal/cli/testdata/plain/ui-url.golden
  - internal/cli/testdata/plain/uninit-force.golden
  - internal/cli/testdata/plain/uninstall-local.golden
  - internal/cli/testdata/plain/upgrade-refresh-warning.golden
  - internal/cli/testdata/plain/version.golden
  - internal/cli/ui.go
  - internal/cli/uninit.go
  - internal/cli/upgrade.go
  - internal/cli/version.go
  - test/integration/status_files_plain_test.go
findings:
  critical: 2
  warning: 2
  info: 1
  total: 5
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-09-18T00:30:41Z
**Depth:** standard
**Files Reviewed:** 77
**Status:** issues_found

## Summary

Reviewed the CLI glow-up phase: the shared `--color` resolver (`colorflag.go`), the seven-role palette, per-verb styled renderers in `internal/cli/present`, and every RunE call site that gates a styled branch behind it. The overall discipline is strong — sanitization (`sanitizeControl`) is applied consistently to filesystem/repo-derived strings before they reach a styled `Render*` call, the plain/`--json` paths are structurally unreachable from the styled branch in every verb, the `serve --mcp` stdout path is never wrapped, and the `present`/`internal/query`/`internal/mcp` boundary (TUI-01 archtest) is intact and correctly widened by import-path prefix.

Two real defects were found, both squarely in the categories this review was asked to focus on:

1. `codegraph init` can resolve the shared colour mode **twice** in a single `RunE` invocation, which the codebase's own `sync.go`/`daemon.go` comments show the team knows must never happen (the `lipgloss.HasDarkBackground` OSC-11 terminal query is documented to fire "at most once per RunE," D-11) — but `init.go` was not updated to follow that discipline when the watch-fallback advisory was added.
2. The styled `explore`/`node` source-code renderer strips every Unicode control rune — including a literal tab (`0x09`) — from real source-code lines before styling them, silently destroying the indentation of any tab-indented source (Go, this project's own language, foremost among them) whenever a human runs `codegraph explore`/`codegraph node` on a color-capable TTY.

Two warnings and one info-level inconsistency round out the findings; none are blocking outside the two criticals above.

## Critical Issues

### CR-01: `codegraph init` resolves colour twice in one RunE — the OSC-11 dark-background query can fire twice

**File:** `internal/cli/init.go:78-79` (call sites), `internal/cli/init.go:123`, `internal/cli/init.go:189`

**Issue:**
`newInitCmd`'s `RunE` calls two helpers in sequence, and each independently calls `resolveColor(cmd)`:

```go
printSummary(cmd, stats, quiet, verbose)      // init.go:78
printWatchFallbackAdvisory(cmd, root)          // init.go:79
```

`printSummary` (when `!quiet`) calls `printSummaryMode(cmd, resolveColor(cmd), ...)` at `init.go:123`. `printWatchFallbackAdvisory` — which is **not** gated by `quiet` at all — independently calls `mode := resolveColor(cmd)` at `init.go:189` whenever `watch.WatchDisabledReason(...)` returns a non-empty reason (e.g. a WSL2 `/mnt` repo, `CODEGRAPH_NO_WATCH=1`, or any other watch-disabling condition — all reachable with default flags).

Per D-11 (`04-CONTEXT.md`) and `resolveColorFrom`'s own doc comment, `lipgloss.HasDarkBackground` is only supposed to be queried **once** per invocation, because the query puts stdin into raw mode and can block up to its own 2-second timeout on a non-answering terminal. `sync.go`'s `printSyncSummary` (`internal/cli/sync.go:79-83`) and `daemon.go`'s `printStoppedDaemons` (`internal/cli/daemon.go:241-246,301`) both go out of their way — with an explicit comment citing D-11 — to resolve colour exactly once and thread the same `colorMode` value through multiple print calls specifically to avoid this. `init.go` was not given the same treatment when the watch-fallback advisory was added.

Concretely: running `codegraph init` (no `--quiet`) interactively on a color-capable TTY, in a repository where the file watcher ends up disabled, will query the terminal's background colour twice — doubling the worst-case pause (up to ~4s instead of ~2s) and, on terminals that mishandle a second raw-mode OSC-11 round-trip in quick succession, risking a stray/garbled response landing in the user's shell.

**Fix:** Resolve colour once at the top of `newInitCmd`'s `RunE` and thread it through both helpers, mirroring `sync.go`'s `printSummaryMode` pattern:

```go
mode := resolveColor(cmd)
printSummaryMode(cmd, mode, stats, quiet, verbose)
printWatchFallbackAdvisoryMode(cmd, mode, root) // new: takes colorMode instead of re-resolving
```

`printWatchFallbackAdvisory` should be split (or overloaded) the same way `printSummary`/`printSummaryMode` already are, so `resolveColor` is called exactly once per `RunE`.

---

### CR-02: `sanitizeControl` strips tabs from real source code, corrupting indentation in styled `explore`/`node` output

**File:** `internal/cli/present/sanitize.go:21-30` (root cause); consumed by `internal/cli/present/explore.go:123-134` (`writeNumberedSource`) and `:141-162` (`writeNumberedSourceRange`), used from `RenderExplore` and `RenderNode` (`internal/cli/present/node.go:50,118`)

**Issue:**
`sanitizeControl` drops every rune for which `unicode.IsControl` is true, which includes the tab character (`0x09`) alongside genuinely dangerous bytes like ESC (`0x1b`) and CR. `writeNumberedSource`/`writeNumberedSourceRange` call `sanitizeControl` on every line of a file's **verbatim source content** before rendering it in `codegraph explore` and `codegraph node <file>|<symbol>`'s styled (TTY) branch.

Since gofmt (and this project's own source) indents with tabs, any real Go source line shown through the styled path loses its leading tab(s) — the exact fixture already committed in this phase proves it. `internal/cli/testdata/plain/node-file.golden` line 14 is:

```
14\t\treturn helper()
```

(line-number, `\t` separator, then the source line `\treturn helper()` — the leading `\t` is gofmt's own indentation of the function body.) Run the same fixture through the styled path (`--color=always` on `codegraph node -f pkga/pkga.go`) and `sanitizeControl` deletes that leading tab, producing `return helper()` with no indentation — visibly different from, and less useful than, the plain/piped output for the exact same file. This directly contradicts D-07's invariant that the styled renderer uses "the SAME sections, order and wording" as the plain markdown — "hue replaces markdown syntax... nothing is re-laid-out" — because the tab strip *is* a re-layout of the source body, not a styling change.

This is a known, tested tradeoff (see the comment in `internal/cli/present/node_test.go`'s `File` subtest, which explicitly documents "sanitizeControl (CR-01) strips control bytes — including the tab — on the styled path only"), but the tradeoff is broader than it needs to be: a bare tab poses no terminal-escape-injection risk (it cannot start an OSC/CSI/DCS sequence and cannot overwrite prior output the way CR/backspace can) — only ESC-initiated sequences and cursor-manipulating C0 controls are the actual injection vector CR-01 is meant to close. Stripping it anyway means every interactive (`--color=auto` on a TTY, the *default*) user of this project's flagship `explore`/`node` commands sees visibly wrong indentation on virtually all Go, Makefile, or other tab-indented source — a functional regression in the command whose entire purpose is faithfully showing source code.

**Fix:** Narrow `sanitizeControl` to exclude `\t` from the strip set (newlines within a line are already handled separately by the per-line split upstream, so this only affects the literal tab case):

```go
func sanitizeControl(s string) string {
	isDangerous := func(r rune) bool { return unicode.IsControl(r) && r != '\t' }
	if !strings.ContainsFunc(s, isDangerous) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isDangerous(r) {
			return -1
		}
		return r
	}, s)
}
```

Update the doc comment and the corresponding test fixtures/comments (`node_test.go`'s `File` subtest, `sanitize_test.go`) to reflect that tabs survive while ESC/C0 controls other than tab are still stripped.

## Warnings

### WR-01: `files.go` never styles the worktree notice, unlike every sibling read command

**File:** `internal/cli/files.go:63-73`

**Issue:** Every other read command that prints the compact worktree notice (`explore.go:61`, `node.go:68`, `search.go:108,148`, `callers.go:59`, `callees.go:60`, `impact.go:61`, `affected.go:140`) routes it through `present.RenderNotice(...)` in the styled branch, so on a colour-capable TTY the notice renders in the Warning role like the rest of the styled output. `files.go` instead prints the notice unconditionally, unstyled, *before* `resolveColor` is even called:

```go
notice := query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context()))
fmt.Fprint(out, notice)          // always plain

mode := resolveColor(cmd)
if mode.Styled {
    return present.RenderFiles(result, present.NewPalette(mode.Dark), mode.Writer(out))
}
```

The result: running `codegraph files --color=always` (or any TTY session) in a worktree-mismatched repo shows a plain-text notice line followed by fully-styled file listing — an inconsistent visual treatment that every other query verb in this same phase got and `files` did not.

**Fix:** Move the notice print after `resolveColor` and branch it the same way the other six verbs do:

```go
mode := resolveColor(cmd)
notice := query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context()))
if mode.Styled {
    w := mode.Writer(out)
    pal := present.NewPalette(mode.Dark)
    if err := present.RenderNotice(notice, pal, w); err != nil {
        return err
    }
    return present.RenderFiles(result, pal, w)
}
fmt.Fprint(out, notice)
...
```

### WR-02: `lineWriter.Write` returns `(0, err)` on a partial failure, discarding the count of bytes already flushed

**File:** `internal/cli/present/line.go:63-87`

**Issue:** `(*lineWriter).Write` iterates line segments and, if an inner `io.WriteString` fails partway through (e.g. the underlying stderr pipe breaks after the first of several complete lines has already been written), returns `(0, err)` rather than the number of bytes of `p` actually consumed before the failure:

```go
if _, err := io.WriteString(lw.w, lw.style.Render(clean)+"\n"); err != nil {
    return 0, err
}
```

This violates `io.Writer`'s general contract (`Write must return a non-nil error if it returns n < len(p)`, but `n` should still reflect what was actually written) and could confuse a caller that retries a "short write" by re-sending the same buffer from byte 0, causing already-emitted lines to be duplicated. Today's only callers (`serve.go`'s watcher stderr banner, `upgrade.go`'s refresh warning) don't retry on error, so this is latent rather than actively triggered, but it's a real correctness gap in a general-purpose `io.Writer` implementation this package exports.

**Fix:** Track and return the actual byte count written so far before propagating the error, e.g. accumulate `n` from the original `len(seg)` (plus separator) for each successfully-written segment before returning early.

## Info

### IN-01: `RenderStatus`'s "Project:" value is sanitized but never styled

**File:** `internal/cli/present/status.go:163`

**Issue:** Every other data field in `RenderStatus` is wrapped in a palette role (`pal.Count.Render(...)`, `pal.Value` implicitly via `writeStatLine`, etc.), but the `Project:` line renders the sanitized path as bare text with no style applied:

```go
fmt.Fprintf(&b, "%s %s\n", pal.Label.Render("Project:"), sanitizeControl(projectPath))
```

This isn't a security issue (the value is still sanitized) and isn't asserted against by any test, but it's an inconsistency worth a second look: every other renderer in this package applies `pal.Path` or `pal.Value` to its equivalent field.

**Fix:** Wrap the value in `pal.Path.Render(...)` (matching how paths are styled elsewhere, e.g. `RenderFiles`, `writeLocationLine`) unless there's a deliberate reason (not documented in the current comment) for leaving it undecorated.

---

_Reviewed: 2026-09-18T00:30:41Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
