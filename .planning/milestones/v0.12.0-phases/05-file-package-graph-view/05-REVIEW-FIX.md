---
phase: 05-file-package-graph-view
fixed_at: 2026-08-31T16:50:11Z
review_path: .planning/phases/05-file-package-graph-view/05-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 5: Code Review Fix Report

**Fixed at:** 2026-08-31T16:50:11Z
**Source review:** .planning/phases/05-file-package-graph-view/05-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (critical + warning): 3
- Fixed: 3
- Skipped: 0
- Deliberately deferred (Info, out of scope by orchestrator instruction): 1 (IN-01)

## Fixed Issues

### CR-01: Rapid re-expand before a file-symbols request resolves issues a second request and can crash the renderer with a duplicate element id

**Files modified:** `web/src/routes/graph/+page.svelte`, `web/tests/graph-expand.test.ts`
**Commit:** `4b695eb7`
**Applied fix:** Added `pendingFileSymbols`, a plain `Set<string>` consulted only inside
`toggleFile`'s fetch branch, to gate dispatch. A re-expand arriving while a request for the
same path is already in flight now no-ops the dispatch (the file is still marked expanded
optimistically, as before) rather than issuing a second `uiClient.fileSymbols` call; the
in-flight request's own `.then` applies the response once it resolves, since it already
checks `expandedFiles.has(id)`. Cleared in both `.then` and `.catch` so a rejected request
cannot wedge the file permanently. The existing once-per-file / cache-not-discard contract
and the pre-existing three-click test are unchanged.

**RED-before-fix evidence:** Added a "CLICK FOUR and FIVE" test extending the sequence to a
third click (re-expand) followed by a fourth (collapse) and fifth (re-expand), all before the
first request settles. Run against the pre-fix code:

```
FAIL  tests/graph-expand.test.ts > route: file tap expand / collapse / re-expand (Task 2) >
  CLICK FOUR and FIVE, a re-expand THEN another collapse arriving before the first request
  settles, issue no further requests and apply correctly once it resolves (CR-01)
AssertionError: expected 2 to be 1 // Object.is equality
  ❯ tests/graph-expand.test.ts:550:48
```

Genuine RED: click 3 dispatched a second `fileSymbols` request (cumulative count 2) before
the fix. GREEN after the fix (22/22 in `graph-expand.test.ts`).

### CR-02: Expanding a file's symbols, then toggling any directory elsewhere, silently destroys the rendered symbols without updating `expandedFiles`

**Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`, `web/tests/graph-expand.test.ts`
**Commit:** `2e07077d`
**Applied fix:** Took option (a) from the review's own fix guidance ("have `replace()`
[become] aware of currently-added incremental elements and re-apply them after the swap, so
the two seams compose") rather than option (b) (folding symbols into the single
`rollupToElements` pure function) — (a) is confined to `GraphCanvas.svelte`, needs no change
to `+page.svelte`'s state model, and carries zero risk to the three-click request-count
contract CR-01's fix also depends on.

`createFileGraphRenderer` now tracks `liveAddedElements` — everything `add()` has merged in,
minus what `removeByIds()` has since removed. Inside `replace()`, after the full swap, it
re-applies whichever of `liveAddedElements` survive: an element whose own `data.parent` id is
present among the new element set's own ids. A symbol's parent is the file id that declares
it (`file-graph-transform.ts`'s `symbolElementsForFile`); that file id is present in the new
element set only when its directory is still expanded, so a symbol whose file just collapsed
away is correctly dropped (cytoscape's `add()` cannot attach a child to a parent absent from
the same batch), while a symbol whose file remains visible now survives an unrelated
directory's own expand or collapse.

**RED-before-fix evidence:** Added a test that expands a repository-root file's symbols, then
taps an unrelated directory (which recomputes `elements` and fires the full-replace path), and
asserts the symbols survive. Run against the pre-fix code:

```
FAIL  tests/graph-expand.test.ts > route: file tap expand / collapse / re-expand (Task 2) >
  expanding a root file's symbols survives toggling an UNRELATED directory elsewhere (CR-02)
AssertionError: expected +0 to be 2 // Object.is equality
  ❯ tests/graph-expand.test.ts:622:52
```

Genuine RED: `childCountOf('a.go')` fell to 0 (the symbols were silently destroyed) before the
fix. GREEN after the fix (23/23 in `graph-expand.test.ts`).

### WR-01: `focus()` changes the viewport but never republishes the geometry seam, leaving click coordinates stale

**Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`, `web/tests/graph-expand.test.ts`
**Commit:** `23a3b377`
**Applied fix:** First independently confirmed the claim by reading `focus()`: it calls
`opts.cy.fit(matched)` and returns, with no call to `computeGeometry`/`onGeometry` anywhere in
the method — matching the review's description exactly. Applied the review's suggested patch
verbatim in shape: `focus()` now calls `computeGeometry(opts.cy)` + `opts.onGeometry(...)`
directly after `cy.fit()`, synchronously (no `layoutstop` event to hang it on, mirroring
`removeByIds()`'s existing fix from 05-07).

**RED-before-fix evidence:** Added a real-headless-cytoscape test that starts the renderer,
counts `onGeometry` invocations, then calls `focus(['a'])` and asserts the count increased by
exactly one. Run against the pre-fix code:

```
FAIL  tests/graph-expand.test.ts > the renderer (real headless cytoscape, no DOM):
  position-displacement finding > focus() republishes the geometry seam, since fit() changes
  the viewport just like a layout settle does (WR-01)
AssertionError: expected 1 to be 2 // Object.is equality
  ❯ tests/graph-expand.test.ts:852:29
```

Genuine RED: `geometryCallCount` did not increase after `focus()` before the fix. GREEN after
the fix (24/24 in `graph-expand.test.ts`).

## Deliberately Deferred (out of fix scope)

### IN-01: `int32` truncation of `int64` counters at the wire boundary is unguarded, though not realistically reachable

Per explicit orchestrator instruction for this fix pass ("Do not fix it") and the review's own
text ("A repository would need over 2^31 distinct file-graph cycles or symbols in one file to
overflow this, which is not realistic for any real corpus this project targets... Not worth
gating the phase on"). Left unfixed. Recorded in `05-REVIEW.md`'s new Resolution section for
traceability rather than silently dropped.

## A note on a staging defect found and fixed during this fix pass

The bundle-rebuild commit (`98cd41dd`) passed `web/build` as a bare pathspec to `git commit`,
which staged deletions of stale files and the modifications to `index.html`/`version.json`/
`.build-manifest`, but silently failed to stage 9 brand-new output files (3 chunks, 2 entry
files, 4 nodes) that `task web:build` had already written to disk — files the committed
`index.html` already referenced by name. `task web:drift` still reported PASS at the time
(it re-hashes whatever is on disk, not what git tracks, so a disk/git divergence like this is
invisible to it), which is why this went unnoticed until a plain `git status` check after the
commit surfaced the untracked files. Fixed by explicitly `git add web/build` and a follow-up
commit (`fad3b39`) — no source or build content changed, only the staging gap was closed.
Recorded here per this project's own fix-pass hazard: this could otherwise have shipped a
committed `web/build/` whose `index.html` referenced files absent from the git tree.

## Post-fix verification

- `cd web && pnpm exec vitest run` → **415/415 passing** (412 baseline + 3 new regression
  tests: CR-01's CLICK FOUR/FIVE, CR-02's unrelated-directory-survival, WR-01's focus
  republish).
- `cd web && pnpm check` → **0 errors, 0 warnings, 1152 files.**
- `task web:build` (bundle rebuilt for the three source changes) → `task web:drift` →
  **PASS** — 108 source files / 32 output files, both digests match (re-confirmed after the
  staging fix below; `git ls-files web/build` now returns 33, matching `find web/build -type
  f`).
- `GOTOOLCHAIN=go1.26.5 task test:unit` → **PASS**, all Go packages (backend untouched by
  this fix pass; `internal/query` and `internal/uiserver` re-ran and passed as part of the
  full suite).

## Commits

1. `4b695eb7` — `fix(05): CR-01 guard toggleFile against a duplicate in-flight fileSymbols request`
2. `2e07077d` — `fix(05): CR-02 compose the incremental symbol seam with replace()'s full swap`
3. `23a3b377` — `fix(05): WR-01 republish the geometry seam on focus(), not only on layoutstop`
4. `98cd41dd` — `feat(05): rebuild the committed bundle for the CR-01/CR-02/WR-01 fixes`
5. `fad3b39` — `fix(05): stage the build output files the prior bundle-rebuild commit missed`

---

_Fixed: 2026-08-31T16:50:11Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
