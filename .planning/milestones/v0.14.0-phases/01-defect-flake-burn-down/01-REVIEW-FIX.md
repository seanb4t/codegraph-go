---
phase: 01-defect-flake-burn-down
fixed_at: 2026-09-15T20:35:00Z
review_path: .planning/phases/01-defect-flake-burn-down/01-REVIEW.md
iteration: 2
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-09-15T20:35:00Z
**Source review:** .planning/phases/01-defect-flake-burn-down/01-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope (`fix_scope: critical_warning` — CR-\*/BL-\*/WR-\* only): 1 (WR-01; CR-01 was
  already fixed and verified closed in iteration 1/this review's re-verification, see below)
- Fixed: 1
- Skipped: 0

IN-01 (Info) is out of `critical_warning` scope and was left untouched per the
orchestrator's explicit scope-discipline instruction.

**Verification environment:** `workflow.use_worktrees` is `false` in
`.planning/config.json` — all edits, commits, and gate verification for this
iteration ran directly in the main checkout at
`/Volumes/Code/github.com/seanb4t/codegraph-go` on branch
`gsd/v0.14.0-milestone`. No isolated worktree was created.

## Carried Forward From Iteration 1

### CR-01: The overlapping-layout generation guard does not scope a `layoutstop` bubble to the layout instance that emitted it

Already fixed and committed in iteration 1 (commit `fbffefc1`). This review's
iteration-2 pass re-traced the fix against the installed `cytoscape@3.34.2`
source and confirmed it closed: the `layoutstop` listener is now registered
on `thisLayoutRun` (the layout instance) rather than on `cy`, so a stale,
superseded instance's real completion can no longer bubble into a newer
generation's `.one()` callback. No further action needed. See
`01-REVIEW-FIX.md`'s prior content (superseded by this file) or `git show
fbffefc1` for full detail.

## Fixed Issues

### WR-01: Five `web/tests/graph-*.test.ts` fake cytoscape cores throw an unhandled `TypeError` on mount because their `edges()` stub omits `.style()`/`.removeStyle()`

**Files modified:**
- `web/tests/graph-tracer.test.ts`
- `web/tests/graph-expand.test.ts`
- `web/tests/graph-expansion.test.ts`
- `web/tests/graph-communities.test.ts`
- `web/tests/graph-cycles.test.ts`
- `web/tests/graph-edge-detail.test.ts` (extension, see below)
- `web/tests/graph-live-update.test.ts` (extension, see below — three
  separate fakes in this file, not just `RouteFakeCore`)

**Commits:** `37ec9c39` (the five named files), `119dc37c` (extension to
`graph-edge-detail.test.ts` and `graph-live-update.test.ts`, see below)

**Applied fix:**

`GraphCanvas.svelte`'s `start()` unconditionally calls
`opts.cy.edges().style('display', 'none')` (line 544) before the first
layout runs, and the `runLayout` `layoutstop` callback unconditionally calls
`opts.cy.edges().removeStyle('display')` (line 415) on the first settle of
any renderer — both introduced by this phase's FIX-05 (hide-until-layoutstop
reveal, commit `676b2540`). The five files' `edges()` fakes modeled the
cytoscape core as returning a bare `{ length }` object, so any test path
that mounts the real component throws `TypeError: opts.cy.edges(...).style
is not a function` synchronously inside the mount `$effect`.

Added `.style()` and `.removeStyle()` no-ops to each of the five `edges()`
fakes, matching the pattern already used in
`web/tests/graph-live-update.test.ts`'s own fakes:

```js
edges() {
	return {
		length: this.els.filter((e) => 'source' in e.data).length,
		style: () => {},
		removeStyle: () => {}
	};
}
```

Applied verbatim (adjusted only for each file's own element-array field name
— `this.elements` vs `this.els`) to all five files. No production source
under `web/src` was touched; `web/build/**` is unaffected and confirmed
unchanged by `task web:drift`.

**RED (pre-fix), captured before applying the fix:**

```
$ pnpm vitest run tests/graph-tracer.test.ts
 Test Files  1 failed (1)
      Tests  2 failed | 6 passed (8)
     Errors  3 errors
```
Failure: `TypeError: opts.cy.edges(...).style is not a function` at
`GraphCanvas.svelte:544` inside `Object.start`, reproduced deterministically
in isolation (matching the review's own confirmation of three consecutive
standalone runs).

**GREEN (post-fix):**

```
$ pnpm vitest run tests/graph-tracer.test.ts
 Test Files  1 passed (1)
      Tests  8 passed (8)
```

Each of the other four fixed files also runs clean standalone:
- `graph-expand.test.ts`: 26 passed (26)
- `graph-expansion.test.ts`: 13 passed (13)
- `graph-communities.test.ts`: 9 passed (9)
- `graph-cycles.test.ts`: 14 passed (14)

**Full suite, run twice for stability (`cd web && pnpm vitest run`, one-shot):**

- Run 1 (after committing `37ec9c39`, before the extension below):
  `Test Files 49 passed (49)` / `Tests 585 passed (585)` / **exit code 1** /
  `Vitest caught 11 unhandled errors during the test run.`

  The orchestrator caught this: per repo rule 84d1gfpywd, "585 passed" is a
  negative-only assertion (no test failed) but the *positive* signal — the
  process exit code — was still 1, and CI runs this suite, so the run was
  not actually green. Grepping the "This error originated in" lines showed
  the 11 unhandled errors split across **two** files, not one:
  `tests/graph-edge-detail.test.ts` (6 errors, `TypeError:
  opts.cy.edges(...).style is not a function`) and
  `tests/graph-live-update.test.ts` (5 errors — 3× `.style is not a
  function` from `RouteFakeCore`'s `edges()` fake at the time, line 952-954,
  and 2× `.removeStyle is not a function` from the two `makeGuardFakeCy`/
  `makeBubblingFakeCy` fakes at lines 486-495 and 733-742, which had
  `removeStyle` but not `style` — GraphCanvas's unconditional
  `opts.cy.edges().style('display', 'none')` call in `start()` threw first
  in `RouteFakeCore`'s case; the two `removeStyle`-only fakes are reached
  via a call path where `start()`'s hide call is not exercised but a later
  `layoutstop` settle still fires the reveal, which is why their error was
  `removeStyle is not a function` rather than `style is not a function`).

- **Extension applied in the same iteration** (commit `119dc37c`): added
  `.style()`/`.removeStyle()` no-ops to `graph-edge-detail.test.ts`'s single
  `edges()` fake, and added the missing `.style()` no-op to all three
  `edges()` fakes in `graph-live-update.test.ts` (`RouteFakeCore`, plus the
  two `removeStyle`-only fakes used by `makeGuardFakeCy`/
  `makeBubblingFakeCy`, which already had `removeStyle` but were still
  missing `style`).

- **Post-extension, run 1:** `Test Files 49 passed (49)` / `Tests 585 passed
  (585)` / **exit code 0** / zero lines matching `unhandled errors`
  (`rg -i "unhandled errors"` on the captured log: 0 matches).
- **Post-extension, run 2:** identical — `Test Files 49 passed (49)` /
  `Tests 585 passed (585)` / **exit code 0** / zero lines matching
  `unhandled errors`.

Both post-extension runs are byte-for-byte identical in their summary
counts and both exit 0 with no unhandled-error block at all — WR-01 is now
fully closed, not deferred.

**Tree-wide sweep for remaining gaps:** ran `rg -n 'edges\(\)' web/tests/`
and inspected every match by hand. All 9 `edges()` fake definitions across
`web/tests/` (7 files; `graph-live-update.test.ts` alone defines 3) now
include both `style: () => {}` and `removeStyle: () => {}`. No seventh (or
eighth/ninth) file with the gap remains for a future review pass to find.

## Gate Verification

Re-run after the extension (commit `119dc37c`):

- **`cd web && pnpm check`** — exit 0, `COMPLETED 1172 FILES 0 ERRORS 0
  WARNINGS 0 FILES_WITH_PROBLEMS`.
- **`cd web && pnpm vitest run`** (full suite, one-shot, run twice) —
  **exit code 0 both times**, `585 passed (585)` both times, `49 passed
  (49)` test files both times, **zero** lines matching `unhandled errors`
  (case-insensitive grep) in either run's captured log.
- **`GOTOOLCHAIN=go1.26.6 task web:drift`** — PASS, `hashed 119 source
  files, manifested 36 output files, committed web/build/ matches both
  digests`. Confirms `web/build/**` is unchanged across both commits
  (expected: no `web/src` change was needed for this fix).
- TypeScript syntax check (Tier 2), both commits: ran
  `npx tsc --noEmit -p tsconfig.json` before and after each edit (via
  `git stash`/`git stash pop`) and confirmed every reported error in
  `graph-expand.test.ts`, `graph-expansion.test.ts`, and
  `graph-live-update.test.ts` (a pre-existing `createFileGraphRenderer`
  typing gap and pre-existing implicit-`any` parameters, unrelated to
  `edges()`) is pre-existing and merely shifted line numbers by the same
  offset as the inserted lines — not newly introduced. `graph-tracer.test.ts`,
  `graph-communities.test.ts`, `graph-cycles.test.ts`, and
  `graph-edge-detail.test.ts` report zero errors.

## Skipped Issues

None — the one in-scope finding (WR-01) was fixed, including the extension
requested after the coordinator caught the non-zero exit code. IN-01 was
correctly left untouched per scope discipline (Info, out of
`critical_warning` scope).

---

_Fixed: 2026-09-15T20:35:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
