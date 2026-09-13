---
phase: 11-graph-view-community-clustering
fixed_at: 2026-09-13T18:05:33Z
review_path: .planning/phases/11-graph-view-community-clustering/11-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 11: Code Review Fix Report

**Fixed at:** 2026-09-13T18:05:33Z
**Source review:** .planning/phases/11-graph-view-community-clustering/11-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (warning): 3 (WR-01, WR-02, WR-03)
- Fixed: 3
- Skipped: 0

IN-01 and IN-02 are Info and out of `critical_warning` scope; left untouched, matching the review's own "no action required" / "not required before shipping" dispositions.

## Fixed Issues

### WR-01: `communityPaletteIndex` had no defense against a caller passing id <= 0, and JS `%` does not floor-modulo a negative dividend

**Files modified:** `web/src/lib/components/graph/community-palette.ts`, `web/tests/graph-communities.test.ts`, `web/build/**` (rebuilt output, same commit)
**Commit:** `52955fff`
**Applied fix:** `communityPaletteIndex` now throws `communityPaletteIndex: communityId must be an integer >= 1, got {id}` when `communityId` is not an integer or is `< 1`, exactly as the review's snippet specified. Added a vitest case asserting `communityPaletteIndex(0)`, `(-1)`, and `(1.5)` all throw, alongside the existing `(1)`→0, `(12)`→11, `(13)`→0 assertions (all still pass). Confirmed `file-graph-transform.ts`'s `if (communityId > 0)` guard (`file-graph-transform.ts:199`) remains the only production call path reaching `communityDiscriminatorClass`/`communityPaletteIndex`, so no production behavior changes — only the previously-unreachable misuse case now fails loudly instead of silently mis-rendering. Since `community-palette.ts` is a `web/src` file, `task web:build && task -s web:drift` were run; both halves MATCH, and the rebuilt `web/build/**` (a genuine content change — the affected chunk is only 78% similar to its predecessor, not a hash-only rename) was committed in the same commit as the source change.

### WR-02: the force-layout walker's directory recursion did not follow symlinked subdirectories

**Files modified:** `web/scripts/check-no-force-layout.mjs`
**Commit:** `1b61ecee`
**Applied fix:** `walk()` now decides recursion with `fs.statSync(full)` (which follows symlinks) instead of a `readdirSync` `Dirent`'s own `isDirectory()` (which reports `false` for a symlink-to-directory without resolving it), closing the blind spot the review identified. A `visitedRealDirs` set (seeded with the root's own realpath, threaded through recursive calls) tracks every directory's symlink-resolved real path so a symlink cycle terminates instead of looping forever. Symlinked files with a scanned extension were already reachable via the pre-existing `else if` branch (a symlinked-file `Dirent` also reports `isDirectory() === false`) — confirmed unchanged by this fix, now backed by `fs.statSync` for both branches uniformly. Added `selfTestSymlinkTraversal()`: creates two temp dirs under `os.tmpdir()` (`mkdtempSync`), writes `planted/injected.ts` containing `const layout = { name: 'cose' };`, symlinks `linkdir -> planted` under a second root, runs the real `walk()` over that second root, and asserts the forbidden match is found through the symlink — with guaranteed cleanup via `try/finally`. The self-test now exits 1 if this case is not detected (wired into the existing `selfTest()` gate alongside the pre-existing in-memory injected case). Real-tree report counts are unchanged: 106 files scanned, elk refs 2/3, 0 forbidden matches.

### WR-03: the force-layout scan's documentation overclaimed its actual coverage, and simple string indirection defeated it undetected

**Files modified:** `web/scripts/check-no-force-layout.mjs`, `Taskfile.yml`, `.planning/phases/11-graph-view-community-clustering/11-SECURITY.md`
**Commit:** `00dc535f`
**Applied fix:**
- (a) Narrowed the documented claim in three places — the script's own header comment, `Taskfile.yml`'s `check:no-force-layout` `desc:`, and `11-SECURITY.md`'s T-11-12 mitigation cell (a value-only edit to the existing table row, no new rows/columns) — from "no force-directed layout is reachable from any code path" to the scan's actual, narrower guarantee: no cytoscape layout is invoked with a forbidden name written as a string literal at the `name:` option position, or as a `cytoscape-<x>` import/dependency specifier; a string-built or variable layout name is not detected.
- (b) Added a **non-fatal, advisory** `unresolvedLayoutNames` finding: every `layout(`/`.layout(` call site under `web/src` whose `name` option could not be statically resolved to a quoted literal within its own line + the following 5 lines is reported with file:line, via `scanUnresolvedLayoutNames()` and a new `LAYOUT_CALL_RE`/`LITERAL_NAME_IN_WINDOW_RE` pair. Verified against the real tree first, per the task's decision tree: `GraphCanvas.svelte`'s one production call site (`opts.cy.layout({ ...LAYOUT_OPTIONS, fit })`) spreads an options object declared a few lines above (itself holding a literal `name: 'elk'`) rather than inlining the name — this line-windowed heuristic cannot trace that spread back to its source, so the call site trips the "unresolved" check today. Per the task's explicit guidance for this exact shape ("the real call uses a variable/imported options object"), this was implemented as **reported-but-non-fatal** (excluded from the PASS/FAIL verdict) rather than as a build-breaking finding, and the residual gap is documented in the script's header, in `printReport`'s advisory line, and in the `11-SECURITY.md` T-11-12 cell. `GraphCanvas.svelte` was **not** rewritten — confirmed byte-identical via `git diff --quiet HEAD -- web/src/lib/components/graph/GraphCanvas.svelte` before and after this fix (it is not a `web/src` change, so no `web:build`/`web:drift` gate applied to this commit). Added a self-test injected case (`cy.layout({ name: layoutName })`) asserting the unresolved check fires; the pre-existing forbidden-literal injected case (`cose`) still passes unaffected. Real-tree report counts are unchanged: 106 files, elk refs 2/3, 0 forbidden matches, 1 advisory unresolved finding (`GraphCanvas.svelte:374`).

## Skipped Issues

None — all three in-scope findings (WR-01, WR-02, WR-03) were fixed. IN-01 and IN-02 are Info-severity and outside `critical_warning` scope per the task's explicit instruction; left untouched.

## Verification

All verification below ran in the **main working tree** (no isolated worktree was created for this fix run, per the task's stated environment — `Isolation none (main checkout)`) — the numbers are reproducible directly from this checkout.

- `pnpm -C web check` — `1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`, run after each of the three fixes.
- `pnpm -C web test` — `49 test files / 584 tests passed`, run after each of the three fixes (including the new `communityPaletteIndex` throw-case assertions). No transient `browse-page.test.ts` timeout was observed in these runs.
- `node web/scripts/check-no-force-layout.mjs --self-test` — exit 0 after WR-02 and WR-03, including the new symlink-traversal self-test and the new WR-03b unresolved-name self-test, plus the pre-existing forbidden-literal injection.
- `node web/scripts/check-no-force-layout.mjs` — exit 0, `{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"unresolvedLayoutNames":[{"file":"web/src/lib/components/graph/GraphCanvas.svelte","line":374,...}],"verdict":"PASS"}` — report counts unchanged from pre-fix (106/2/3/0 forbidden); the new advisory field does not affect the verdict.
- `task check:no-force-layout` — exit 0.
- Confirmed no `check-no-force-layout-*` temp directories remained under `$TMPDIR`/`os.tmpdir()` after any self-test run.
- `git diff --quiet HEAD -- web/src/lib/components/graph/GraphCanvas.svelte` — clean (byte-identical; not touched by any of the three fixes).
- `task web:build && task -s web:drift` — run for the WR-01 commit only (the only fix touching a `web/src` file): both halves MATCH; the rebuilt `web/build/**` was a genuine content change (renamed/rehashed chunks reflecting the new `throw` branch, not hash-only churn) and was committed in the same commit as the source change (`52955fff`).
- `git status --porcelain` — empty after the final commit below.
- No `codegraph ui` process was left running; `web/scripts/breadcrumb-check.mjs` was never invoked; `corpora/graph-cluster-threshold.json`, `go.mod`, `.proto`, and all Go files were untouched.

No source files were left in a broken state; no uncommitted source changes remain outside this report and the pre-existing, untouched `11-REVIEW.md`.

---

_Fixed: 2026-09-13T18:05:33Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
