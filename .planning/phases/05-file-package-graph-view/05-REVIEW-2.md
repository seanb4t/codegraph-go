---
phase: 05-file-package-graph-view
reviewed: 2026-08-31T17:04:34Z
depth: deep
files_reviewed: 3
files_reviewed_list:
  - web/src/routes/graph/+page.svelte
  - web/src/lib/components/graph/GraphCanvas.svelte
  - web/tests/graph-expand.test.ts
findings:
  critical: 0
  warning: 3
  info: 0
  total: 3
status: issues_found
---

# Phase 5 Fix Pass: Code Review Report (Re-Review 2)

**Reviewed:** 2026-08-31T17:04:34Z
**Depth:** deep (fix-commit audit, not full phase re-review)
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Scope: commits `4b695eb7` (CR-01), `2e07077d` (CR-02), `23a3b377` (WR-01), and the bundle-rebuild pair `98cd41dd`+`fad3b39c`.

All three regression-shaped fixes are **functionally correct and empirically verified**, not merely trusted from the commit message. For each of CR-01, CR-02, and WR-01 I reverted only that commit's production-code hunk (leaving its own new test in place) and re-ran the specific new test — all three went RED with the exact symptom the commit message claims, then GREEN again once restored:

- CR-01 revert → `expected 2 to be 1` (duplicate dispatch)
- CR-02 revert → `expected +0 to be 2` (symbols wiped)
- WR-01 revert → `expected 1 to be 2` (no republish)

Full suite: 415/415 passing (confirmed by direct run, not taken on faith). `svelte-check`: 0 errors/0 warnings. `git ls-files web/build | wc -l` = 33, `find web/build -type f | wc -l` = 33, `git status --porcelain web/build` clean — the `fad3b39c` staging correction is complete and no further gap exists (the known `web:drift` asymmetry, WINDOWS.md #29, is not re-reported).

None of the two contracts CR-01 must not weaken (once-per-file cache-not-discard; three-click cumulative-request-count-1) were broken — both existing tests covering them still pass unmodified, and `pendingFileSymbols` is cleared in both the `.then` and `.catch` branches (`+page.svelte:319,342`), so it cannot leak on a rejected request.

That said, three defects survive this fix pass, none of them the ones the fix commits set out to close:

1. A **fourth recurrence** of a review/plan-id leak into shipped source comments — a defect class this exact codebase has already run a dedicated cleanup commit for three times (Phase 4, `05-08`, and `3f9fc91e`'s `05-07` pass), reintroduced by these very fix commits and invisible to the project's own established detection command.
2. A genuine, empirically-reproduced state-desync gap in CR-02's compose logic: collapsing and re-expanding the *same* directory an expanded file lives in silently drops that file's symbols with no path back to them short of a second manual collapse/re-expand of the file itself — untested by the new regression test, which only covers the "unrelated directory" case named in its own title.
3. WR-01's regression test proves a republish call fired, but never asserts the republished coordinates actually differ from the pre-`focus()` values — the specific "stale values, not just a missing publish" failure mode the fix's own commit message says it is closing.

Assertions examined in the three new tests: **11** total (CR-01: 4, CR-02: 4, WR-01: 3). Zero of the 11 are zero-count assertions (none assert `.toBe(0)` or similar) and therefore none require a positive control. One assertion (WR-01's `toBeGreaterThan(0)`) is a floor-style check, not an upper bound; none of the 11 are upper-bound-style (`toBeLessThanOrEqual`, `toBeLessThan`) assertions with or without a non-zero floor.

## Warnings

### WR-01a: `CR-01`/`CR-02`/`WR-01` review-finding ids leaked into shipped source comments — a defect class already cleaned up three times in this codebase

**File:** `web/src/routes/graph/+page.svelte:77`, `:311`
**File:** `web/src/lib/components/graph/GraphCanvas.svelte:229`, `:302`, `:415`

**Issue:** Commit `3f9fc91e` ("docs(05-07): copy pass — remove leaked plan/review identifiers") explicitly documents that this exact pattern — task/review-finding identifiers such as `(05-07, GRF-03)` and `(review M-3)` leaking into shipped source doc comments — had *already* shipped once in Phase 4 and had to be fixed again in `05-08`, and lists the banned-substring check it ran to prove the cleanup: `rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0' web/src/routes/graph/ web/src/lib/components/graph/` returning zero matches. That commit also explicitly carves out `D-XX`/`T-05-NN` decision/threat references as accepted "established precedent, not caught by this check" — but review-finding-style tags are precisely the banned category.

This fix pass (`4b695eb7`, `2e07077d`, `23a3b377`) reintroduces the exact same class of leak with a new tag shape the established check does not cover: `(CR-01)`, `(CR-01/CR-02 review)`, `(CR-02, found in review)`, `(WR-01, found in review)`. Re-running the project's own established check confirms it is still green and blind to this leak:

```
$ rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0' web/src/routes/graph/ web/src/lib/components/graph/
(no output, exit 1)
$ rg -n 'CR-01|CR-02|WR-01' web/src/routes/graph/+page.svelte web/src/lib/components/graph/GraphCanvas.svelte
web/src/routes/graph/+page.svelte:77:	// handlers below (CR-01).
web/src/routes/graph/+page.svelte:311:			// (CR-01).
web/src/lib/components/graph/GraphCanvas.svelte:229:		// compose with it rather than silently destroying it (CR-01/CR-02
web/src/lib/components/graph/GraphCanvas.svelte:302:			// unrelated directory toggle (CR-02, found in review). After
web/src/lib/components/graph/GraphCanvas.svelte:415:			// instant a cycle-focus fit runs (WR-01, found in review).
```

This is exactly the "(a) the fix satisfies its own literal check while the symptom persists" shape: the established leak-detection command is green, the leak is present.

**Fix:** Rewrite the five comments to convey the same rationale without the `CR-`/`WR-`/"found in review" substrings, mirroring `3f9fc91e`'s own rewrite pattern (state the *mechanism* — "an added element whose own parent id is present among the new element set" — not the review artifact that found it). Then widen the banned-substring check to also catch this tag shape, e.g. `rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0|\bCR-[0-9]|\bWR-[0-9]|found in review'`, so a fifth recurrence is caught before merge rather than by a subsequent re-review.

### WR-01b: CR-02's compose logic does not restore a file's symbols when its *own* directory (not an unrelated one) is collapsed and re-expanded — untested, and confirmed by direct reproduction

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:314-328` (replace's survivor-composition logic)
**File:** `web/src/routes/graph/+page.svelte:225-244` (`toggleDirectory` never touches `expandedFiles`/`fileSymbolsCache`)

**Issue:** `replace()`'s survivor filter (`liveAddedElements.filter((el) => newIds.has(el.data.parent))`, `GraphCanvas.svelte:322`) correctly and *intentionally* drops a symbol whose file just left the visible element set — that is the documented, correct behavior for the case CR-02 targets (an *unrelated* directory's toggle must not touch it). But the file-graph transform (`file-graph-transform.ts`) only renders a non-root file's node at all when its own directory is in `expandedDirs` — so collapsing the *very* directory a file's symbols were expanded in removes that file's own parent id from `newIds` too, and the same orphan-drop logic silently discards the symbols. Nothing in `+page.svelte` reconciles `expandedFiles`/`fileSymbolsCache` when this happens — `toggleDirectory` (`+page.svelte:225`) only ever reads/writes `expandedDirs`.

The result is a real, reproducible state desync: `expandedFiles` still lists the file as expanded (and its response is still cached) after its owning directory is collapsed and re-expanded, but the rendered symbol count is silently back to zero, and no new `fileSymbols` request fires to restore it (since it's still "cached"). The first tap on the reappeared file node then takes the *collapse* branch (`toggleFile`'s `expandedFiles.has(id)` case, `+page.svelte:294-301`) — a visual no-op, since there is nothing to remove — rather than restoring the symbols; a *second* tap is required before the cached response is reapplied.

I confirmed this directly rather than by inference alone: I inserted a scratch test (not committed) reproducing exactly this sequence — expand `dir`, expand `dir/b.go`'s symbols (child count 2), collapse `dir`, re-expand `dir` — and observed `childCountOf('dir/b.go')` is `0` after the sequence, with the `fileSymbols` call count still `1` (i.e., `expandedFiles` still considers it served-from-cache, but nothing reapplies the cache). The scratch test was reverted (`git checkout -- web/tests/graph-expand.test.ts`); it is not part of the shipped diff.

The new CR-02 regression test (`web/tests/graph-expand.test.ts:603`) covers only the "unrelated directory" case its own title names — the reverse order the review brief specifically asked about ("toggle directory then expand file") and the same-directory collapse/re-expand case are both untested.

**Fix:** Either (a) have `toggleDirectory` drop any file under the collapsing directory from `expandedFiles`/clear its `addedElements` bookkeeping so the state re-syncs (a subsequent re-expand of the file would then correctly re-fetch-or-serve-from-cache and re-`add()` on tap), or (b) have the `elements` derivation / the `replace()` composition re-apply cached symbols for any still-`expandedFiles` file whose parent reappears in `newElements`, not only ones tracked in `liveAddedElements` from a prior `add()`. Add a regression test for both: same-directory collapse+re-expand of an expanded file, and the reverse toggle order (directory toggle before file expansion) the review scope explicitly asked to check.

### WR-01c: WR-01's regression test proves a republish call fired, not that the republished coordinates changed

**File:** `web/tests/graph-expand.test.ts:807-853`
**File:** `web/src/lib/components/graph/GraphCanvas.svelte:426-431` (the fix under test)

**Issue:** The commit message states the bug is that "any previously-published geometry entry's x/y went stale the instant a cycle-focus fit ran" — i.e., the defect is about *stale values*, not merely a missing callback invocation. `computeGeometry` (`GraphCanvas.svelte:149-`) returns `renderedBoundingBox`-derived, pan/zoom-dependent screen coordinates, so `focus(['a'])`'s `cy.fit(matched)` (fitting the viewport to a single node out of two) should genuinely change at least the reported x/y for both nodes relative to the post-`start()` publication.

The test asserts only:
- `geometryCallCount` incremented by exactly 1 (proves a call fired — a real, non-vacuous check that correctly goes RED when the fix is reverted, confirmed by direct revert)
- `lastGeometry.map(g => g.id).sort()` equals `['a', 'b']` (proves the *same two node ids* are present — true both before and after `focus()`, since `focus()` never changes which nodes exist)

Neither assertion reads the geometry's `x`/`y` fields at all. A regression where `focus()` calls `onGeometry` but with a stale/cached/unchanged geometry snapshot (e.g. a future refactor that memoizes `computeGeometry`'s result across a `fit()` call, or calls `onGeometry` with a value captured before `fit()` ran instead of after) would still pass this test: the call count would still increment, and the id set is invariant across `focus()` by construction. This is not the shape-(a)/(b) failure this fix pass itself introduced — the fix is correct, verified by direct revert — but the new test does not lock down the property the bug report actually cared about, and would not catch the most likely future regression of the same class.

**Fix:** Capture the geometry entries for `a`/`b` published by `start()` and assert the corresponding entries published by `focus()` differ (e.g. `x`/`y` not equal, or explicitly assert they moved in the direction a single-node fit implies) — not just that the id set is unchanged and a call count incremented.

## Resolution

_Applied by the gsd-code-fixer against this re-review's findings (all three in scope — this
pass has no critical/warning split, all three are Warnings). See the fix commits below for
detail; no separate REVIEW-FIX.md was produced for this iteration since the orchestrator
directed the resolution to be recorded here instead._

- **WR-01b — fixed.** `toggleDirectory`'s collapse branch (`web/src/routes/graph/+page.svelte`)
  now drops any `expandedFiles` entry whose immediate directory (`dirOf`, newly exported from
  `file-graph-transform.ts` for this reuse) matches the collapsing directory, so `expandedFiles`
  never claims a file is expanded once its directory round-trip has silently dropped its
  symbols. `fileSymbolsCache` is left untouched, so a further tap on the file re-applies the
  cached response with zero new `fileSymbols` requests — the once-per-file contract survives.
  Took the review's own option (a) (confined to `+page.svelte`, no change to GraphCanvas's
  compose logic) over option (b) (teaching `replace()` about the route's cache), matching this
  fix pass's own established preference for the lower-risk, more confined change. Two new
  regression tests (both directions — expand-then-toggle-own-directory, and toggle-before-
  expand plus a second round trip) were confirmed RED against the pre-fix code before this
  change landed. A third, small follow-up commit updated `graph-collapse.test.ts`'s
  module-surface contract test to include the newly exported `dirOf`.
  Commits: `209a5fc0`, `420b18cb`.
- **WR-01c — fixed.** The `focus()` geometry regression test in `web/tests/graph-expand.test.ts`
  now captures each node's published geometry after `start()`, then asserts the corresponding
  entry published by `focus()` genuinely differs — not just that a call fired and the same node
  ids are present. Proved this catches what the old assertions missed by temporarily
  reintroducing the exact "stale value" regression the original fix's own commit message
  targeted (capture geometry before `fit()`, publish it after) directly in
  `GraphCanvas.svelte`: the call-count and id-set assertions still passed, but the new
  coordinate assertion failed — confirming the old test was vacuous for this failure mode and
  the new one is not. The scratch modification was reverted before committing;
  `GraphCanvas.svelte` is unchanged by this fix. Commit: `b7fc3897`.
- **WR-01a — fixed.** Rewrote all five leaked `CR-01`/`CR-02`/`WR-01`/"found in review" doc
  comments (`+page.svelte:78,343`; `GraphCanvas.svelte:229,303,417`) to state the invariant each
  documents rather than the review artifact that found it. Left the six pre-existing,
  out-of-phase instances (`browse/+page.svelte:62`, `SearchPanel.svelte:123/125/134`,
  `AnalysisPanel.svelte:108`, the generated `ui_pb.ts:1551`) untouched per this finding's own
  scope discipline. Verified by hand that the widened pattern
  (`phase 5|phase-5|GRF-0|ENG-03|05-0|\bCR-[0-9]|\bWR-[0-9]|found in review`) now returns zero
  matches in `web/src/routes/graph/` and `web/src/lib/components/graph/`.
  **Declined:** did not add a new automated leak-check gate. The project's established check is
  a manual `rg` command documented in planning prose (recorded in `05-07-SUMMARY.md` and
  elsewhere), not an enforced script, test, or CI step anywhere in this repository (confirmed:
  no hit for `GRF-0`/`ENG-03` outside `.planning/`, no `leak`-named script under `web/` or in
  `Taskfile.yml`) — there is nothing existing to "adjust." The finding's own Fix text frames
  widening the check as conditional ("If you add or adjust a leak-check, make it discriminate"),
  not a required part of this fix, and authoring a brand-new permanent CI gate was judged outside
  this fix pass's scope. Commit: `c48a56c5`.
- The committed `web/build/` bundle was rebuilt for the three source fixes above and
  `git add web/build` staged explicitly, with `git status --porcelain web/build` confirmed
  empty before committing (WINDOWS.md #29's staging gap, from the prior fix pass, did not
  recur). `task web:drift` reconfirmed PASS (108 source files / 32 output files, both digests
  matched). Commit: `2d057f9b`.

**Post-fix verification:**

- `cd web && pnpm exec vitest run` → **417/417 passing** (415 baseline + 2 new WR-01b regression
  tests; WR-01c strengthened an existing test rather than adding one).
- `cd web && pnpm check` → **0 errors, 0 warnings, 1152 files.**
- `task web:build` (rebuilt for the WR-01a/b/c fixes) → `task web:drift` → **PASS**.
- `git status --porcelain web/build` → **empty** after staging.
- `GOTOOLCHAIN=go1.26.5 task test:unit` → **PASS**, all Go packages (backend untouched by this
  fix pass).

---

_Reviewed: 2026-08-31T17:04:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Resolution recorded: 2026-08-31_
_Fixer: Claude (gsd-code-fixer)_
