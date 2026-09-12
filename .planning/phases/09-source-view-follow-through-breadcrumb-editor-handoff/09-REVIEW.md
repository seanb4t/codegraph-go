---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
reviewed: 2026-09-12T20:18:20Z
depth: deep
files_reviewed: 27
files_reviewed_list:
  - corpora/breadcrumb-check.json
  - internal/cli/editordiscovery_test.go
  - internal/cli/editordiscovery.go
  - internal/cli/editorurl_test.go
  - internal/cli/editorurl.go
  - internal/cli/ui_test.go
  - internal/cli/ui.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/editorlink_test.go
  - internal/uiserver/editorlink.go
  - internal/uiserver/editorpresets.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/server_test.go
  - internal/uiserver/server.go
  - web/scripts/breadcrumb-check.mjs
  - web/src/lib/breadcrumb.ts
  - web/src/lib/components/browse/EditorLinkPicker.svelte
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/lib/editor-prefs.ts
  - web/src/lib/source-lines.ts
  - web/tests/breadcrumb.test.ts
  - web/tests/editor-link-picker.test.ts
  - web/tests/editor-prefs.test.ts
  - web/tests/setup.ts
  - web/tests/source-lines.test.ts
  - web/tests/source-pane-breadcrumb.test.ts
  - web/tests/source-pane-editor-link.test.ts
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 09: Code Review Report

**Reviewed:** 2026-09-12T20:18:20Z
**Depth:** deep
**Files Reviewed:** 27
**Status:** clean

## Summary

This is iteration 3 (final) of the review→fix loop, scoped to verifying
WR-05's fix (commit `422952f8`) and re-checking for regressions across the
full file set. Independently re-ran `GOTOOLCHAIN=go1.26.6 go build ./...`,
`go vet ./internal/cli/... ./internal/uiserver/...`, `go test
./internal/cli/... ./internal/uiserver/...` (all passing, including the
34s `internal/uiserver` suite), `pnpm -C web test` (542/542), and `task
web:drift` (source half MATCH at 114 files, output half MATCH at 32 files,
committed `web/build/**` reflects the current source with no manual edits)
in this checkout — no residual process was left running.

**WR-05 is genuinely resolved.** Read `git show 422952f8` directly rather
than trusting the fixer's own description:

- `EditorLinkPicker.svelte:81-84`'s root dropped `aria-modal="true"` and
  changed `role="dialog"` to `role="region"`, keeping `aria-label="Editor
  link"` — a `role="region"` requires an accessible name per WAI-ARIA, and
  this one has it (verified directly in the source, not just inferred from
  the fixer's claim; the test suite does not separately assert the
  `aria-label`, but the markup carries it unconditionally).
- `rg -n "aria-modal" web/src` returns zero attribute occurrences — the
  only matches anywhere are inside code comments in `SourcePane.svelte`
  explaining *why* `aria-modal` was deliberately not used, and inside two
  test files' own comments/assertion names. `rg -n 'role="dialog"'
  web/src -g '*.svelte'` also returns zero matches — no other component
  regressed this pattern in either direction.
- The focus-restore path (`toggleButtonEl?.focus()`,
  `SourcePane.svelte:578`) uses optional chaining against a `$state`
  binding that Svelte sets back to `undefined` on unmount, so it cannot
  throw if the toggle button was removed from the DOM before the picker
  closes (e.g. `editorLinkState` transitioning away from `loaded` while
  `pickerOpen` is still true) — confirmed by inspecting the binding site,
  not merely assumed.
- WR-03's behavior (focus-in on auto-open, focus-restore on close) is
  unchanged in this diff — only the two `role`/`aria-modal` attributes and
  surrounding comments moved — and the regression tests in both
  `editor-link-picker.test.ts` and `source-pane-editor-link.test.ts` now
  pin `role="region"` plus `hasAttribute('aria-modal') === false` on top of
  the pre-existing focus assertions, so a future accidental reintroduction
  of `aria-modal` or the modal role would fail loudly in two independent
  test files, not just one.
- No keyboard trap was introduced: the panel wrapper is `role="presentation"`
  with only an `Escape` keydown handler (`handlePickerKeydown`) and no
  `Tab`/`Shift+Tab` interception, consistent with `role="region"`'s
  non-modal semantics — a sighted keyboard user can freely tab in and out
  of the panel, matching what a screen-reader user now also perceives
  (the panel is present in, not removed from, the accessibility tree).

No other file in the fixed commit, or in the wider file set, showed a
regression. `internal/uiserver`'s threat register
(`.planning/phases/09-.../09-SECURITY.md`) still reports `threats_open: 0`
at ASVS L1 — I found no unmitigated `high`-severity threat, and this
iteration's single fix (dropping two attributes and updating comments)
does not touch any of the register's mitigated rows.

**IN-01 carries forward, unfixed** (correctly — it is Info-severity and out
of `critical_warning` fix scope per the fixer's own report): see below.

## Info

### IN-01: `TestEditorDiscoverySourceNeverSpawnsAProcess`'s trailing zero-check is dead code

**File:** `internal/cli/editordiscovery_test.go:331-339`

**Issue:** The loop over `required` (`internal/cli/editordiscovery_test.go:332-337`)
already calls `t.Fatalf` — which halts the test via `runtime.Goexit` — on
the first missing substring, so `inspected` can only ever equal
`len(required)` (2) by the time the loop completes normally. The trailing
`if inspected == 0 { t.Fatal(...) }` at line 338 can never execute. This is
harmless — it does not weaken the guard, since the `t.Fatalf` inside the
loop already provides the "positive control" protection the surrounding
comment describes — but it is unreachable code that reads as if it
verifies something the loop hasn't already guaranteed.

**Fix:** Either remove the trailing dead check, or restructure to actually
need it (count matches without failing early inside the loop, then assert
the total once after the loop):

```go
inspected := 0
for _, r := range required {
	if strings.Contains(text, r) {
		inspected++
	}
}
if inspected != len(required) {
	t.Fatalf("editordiscovery.go is missing %d of %d required probe-only substrings %v — positive control failed",
		len(required)-inspected, len(required), required)
}
```

---

_Reviewed: 2026-09-12T20:18:20Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
