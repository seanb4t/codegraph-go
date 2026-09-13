---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
fixed_at: 2026-09-12T20:14:57Z
review_path: .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-REVIEW.md
iteration: 2
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 09: Code Review Fix Report

**Fixed at:** 2026-09-12T20:14:57Z
**Source review:** .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 1 (critical_warning scope: WR-05 — the iteration-1 CR-01,
  WR-01, WR-02, WR-03, WR-04 findings were already fixed in iteration 1, commits
  `28d5d795`..`67d15645`; see below)
- Fixed: 1
- Skipped: 0

Edits were made directly in the main checkout (`workflow.use_worktrees` is
`false` in `.planning/config.json`), on branch
`gsd/v0.13.0-guard-hardening-ui-follow-through`, starting from HEAD `67d15645`.
No worktree was created; no worktree cleanup was required. All verification
below (`pnpm -C web test`, `task web:test`, `task web:build`, `task web:drift`)
ran in the main checkout, so the numbers are reproducible directly from this
tree.

## Iteration 1 (already fixed, for record — see `git log`)

These five findings from iteration 1's `09-REVIEW.md` were fixed in the prior
run and are unaffected by this iteration's work:

- **CR-01** — persisted preset editor-link override silently dropped on the
  first probe of every page load. Commit `28d5d795`.
- **WR-01** — `buildEditorURL` unguarded slice on an unbalanced template.
  Commit `c1163a25`.
- **WR-02** — no source-level no-spawn guard for `editordiscovery.go`.
  Commit `2541a579`.
- **WR-03** — `EditorLinkPicker` self-opens as `role="dialog"` with no focus
  management (focus-in/focus-out only). Commit `4f33c60f`.
- **WR-04** — `CODEGRAPH_NO_EDITOR_URL` env parse running after
  `--no-editor-url` already settled the outcome. Commit `67d15645`.

This iteration's review re-verified all five and found no regressions; it
also found that WR-03's fix was only **partially** complete (see WR-05
below).

## Fixed Issues

### WR-05: `EditorLinkPicker`'s dialog declares `aria-modal="true"` with no focus trap

**Files modified:** `web/src/lib/components/browse/EditorLinkPicker.svelte`,
`web/src/lib/components/browse/SourcePane.svelte`,
`web/tests/editor-link-picker.test.ts`,
`web/tests/source-pane-editor-link.test.ts`, `web/build/**`
**Commit:** `422952f8`
**Applied fix:** The WR-03 fix pass had added `aria-modal="true"` to the
picker's `role="dialog"` root while its own new comment explicitly declined
to implement a focus trap — a half-state the review flagged as a genuine
regression (modern Chromium/Firefox remove everything outside a
`aria-modal="true"` dialog from the accessibility tree regardless of whether
a trap backs the promise, so a screen-reader user loses access to the rest of
the page while a sighted keyboard user can still `Tab` freely into it). Per
repo constraints, took the cheaper of the review's two complete options given
the picker is a small popover anchored to a toggle button (not a
page-blocking modal, matching the phase's D-18 "picker popover"
description): dropped `aria-modal="true"` and changed
`EditorLinkPicker.svelte`'s root from `role="dialog"` to `role="region"`
(non-modal semantics), kept Escape-to-close and the WR-03
focus-in-on-open/restore-on-close behavior unchanged, and rewrote the
`SourcePane.svelte` `$effect` comments so they describe a non-modal
disclosure panel instead of dialog/focus-trap semantics. Updated
`editor-link-picker.test.ts`'s prior `aria-modal="true"` assertion to assert
`role="region"` and the absence of `aria-modal`, and added the same
`aria-modal` absence assertion to `source-pane-editor-link.test.ts`'s
existing focus-management regression test so the invariant is pinned at both
the component and integration level. `pnpm -C web test` / `task web:test`
(542/542) and `task web:build` + `task web:drift` (committed SPA rebuilt,
source and output digests both MATCH) all pass.

## Skipped Issues

None — the single in-scope finding (WR-05) was fixed.

IN-01 (`internal/cli/editordiscovery_test.go:338-339`'s dead trailing
zero-check) is Info-severity and out of `critical_warning` fix scope for this
iteration; left untouched per repo constraints.

---

_Fixed: 2026-09-12T20:14:57Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
