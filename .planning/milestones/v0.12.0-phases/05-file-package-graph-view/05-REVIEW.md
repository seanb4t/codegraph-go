---
phase: 05-file-package-graph-view
reviewed: 2026-08-31T16:37:54Z
depth: deep
files_reviewed: 31
files_reviewed_list:
  - internal/query/filegraph_cycles.go
  - internal/query/filegraph_cycles_test.go
  - internal/query/filegraph_test.go
  - internal/query/filesymbols.go
  - internal/query/filesymbols_test.go
  - internal/query/traverse.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/filegraph_test.go
  - internal/uiserver/filesymbols_test.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - web/scripts/graph-collapse-affordance-check.mjs
  - web/scripts/graph-expand-check.mjs
  - web/scripts/graph-measure.mjs
  - web/scripts/graph-verdict.mjs
  - web/src/app.d.ts
  - web/src/cytoscape-elk.d.ts
  - web/src/lib/components/graph/GraphCanvas.svelte
  - web/src/lib/components/graph/edge-kind-columns.ts
  - web/src/lib/components/graph/file-graph-transform.ts
  - web/src/lib/components/graph/graph-style.ts
  - web/src/routes/graph/+page.svelte
  - web/tests/file-graph-transform.test.ts
  - web/tests/graph-collapse.test.ts
  - web/tests/graph-cycles.test.ts
  - web/tests/graph-edge-detail.test.ts
  - web/tests/graph-expand.test.ts
  - web/tests/graph-expansion.test.ts
  - web/tests/graph-measure.test.ts
  - web/tests/graph-tracer.test.ts
  - web/tests/graph-verdict.test.ts
findings:
  critical: 2
  warning: 1
  info: 1
  total: 4
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-08-31T16:37:54Z
**Depth:** deep
**Files Reviewed:** 31
**Status:** issues_found

## Summary

This phase's Go backend (`internal/query/filegraph*.go`, `filesymbols.go`, `traverse.go`'s
`FileGraph` addition, and the two new `uiserver` rpcs) is careful and correct: the two-scan
rollup, the D-08 package-pseudo-node exclusion, the iterative Tarjan cycle detector, and the
`FileSymbols` path-confinement all read as intended, and their test suites consistently pair
every zero-count/upper-bound assertion with a positive control or a floor — I sampled every
Go test in the phase's file list (filegraph_test.go's 9 cases, filegraph_cycles_test.go's 6,
uiserver's filegraph_test.go's 6 and filesymbols_test.go's 8) and found zero vacuous guards
among them. The `.mjs` measurement harness (`graph-verdict.mjs`/`graph-measure.mjs`) is
genuinely fail-closed: `judgeMetric` resolves an absent, non-numeric, or self-contradictory
metric to FAIL, never a skip, and `runSession`'s `finally` writes the raw observation before
any teardown is awaited.

The frontend renderer seam (`GraphCanvas.svelte`) and the route (`+page.svelte`) are where
this review found real defects. `file-graph-transform.ts` correctly copies cycle membership
from the wire and never re-derives it. But two separate, concrete interactions between the
route's THREE independent pieces of client state (`expandedDirs`, `expandedFiles` +
`fileSymbolsCache`, and the cytoscape instance's live element set) are not coordinated, and
both are reachable through ordinary user interaction, not synthetic edge cases. I traced each
by reading the exact code paths involved (not by running the app), and cross-checked the
existing test suites in `web/tests/graph-expand.test.ts` to confirm neither scenario is
covered by the tests the phase already shipped.

I examined roughly 40 zero-count/upper-bound assertions across the nine frontend test files in
scope (`rg` count: 40 matches for `toBe(0)`/`toHaveLength(0)`/`toBeGreaterThanOrEqual`/
`toBeLessThanOrEqual`) and directly read the surrounding context for 12 of them, sampled across
`file-graph-transform.test.ts`, `graph-collapse.test.ts`, and `graph-cycles.test.ts`; every one
I read paired the zero-count assertion with an adjacent `toBeGreaterThan(0)` fixture-size check
or an explicitly labeled "positive control" sibling test. `toBeLessThanOrEqual` appears zero
times in the frontend suite; `toBeGreaterThanOrEqual` appears twice, both as genuine floors.
I did not individually re-verify the remaining ~28 occurrences given review scope, but found no
counter-example in the sample and no structural reason (e.g. a shared broken helper) to expect
one.

## Critical Issues

### CR-01: Rapid re-expand before a file-symbols request resolves issues a second request and can crash the renderer with a duplicate element id

**File:** `web/src/routes/graph/+page.svelte:255-313` (`toggleFile`)
**Issue:**

`toggleFile`'s three branches are, in order: collapse (if `expandedFiles.has(id)`), re-expand
from cache (if cached), else fetch. The fetch branch adds `id` to `expandedFiles`
*optimistically*, specifically so a second click arriving while the request is still in flight
is recognized as a collapse rather than a duplicate fetch (documented at lines 250-254, and
proven by the "STALE resolved-after-collapse" test in `web/tests/graph-expand.test.ts:499-528`).

That test proves exactly two clicks (expand, then collapse) issued before the request settles.
It does **not** cover a third click — a second *expand* attempt — arriving before the same
request settles. Trace it:

1. Click 1: `expandedFiles` = `{}`, not cached → fetch **A** issued; `expandedFiles` optimistically becomes `{id}`.
2. Click 2 (before A resolves): `expandedFiles.has(id)` is true → collapse branch; `cached` is still `undefined` (A hasn't resolved) → `removedElementIds = []` (no-op); `expandedFiles` becomes `{}`.
3. Click 3 (still before A resolves): `expandedFiles.has(id)` is now `false`, and `cached` is still `undefined` → falls into the **fetch branch again**, issuing a second request **B** for the same path, and re-adding `id` to `expandedFiles`.

When A resolves, its `.then` sees `expandedFiles.has(id)` true (set by click 3) and calls
`addedElements = symbolElementsForFile(id, respA)` → `GraphCanvas`'s `addedElements` effect
calls `renderer.add()` → `cy.add()` with the symbol element ids (`symbol\0<path>\0<wireId>`,
`file-graph-transform.ts:333-335`). When B resolves shortly after, its `.then` sees the same
`expandedFiles.has(id)` true and does the **identical** `addedElements = symbolElementsForFile(id, respB)` → a second `renderer.add()` call with the **same deterministic ids** (the
symbol id is a pure function of file path + wire symbol id, so two responses for the same file
produce identical ids). Cytoscape's `add()` throws when asked to create a second element with
an id that already exists in the live collection. This is not a rare timing window — it is any
double-click-to-re-expand landing before the round trip completes (localhost RPC latency is
frequently well under typical double-click intervals), and it directly contradicts the
documented, tested "one request, never doubled" contract this same function's comments assert
(lines 241, 254, 291).

**Fix:** Track in-flight requests per file id (e.g. a `Set<string>` or `Map<string, Promise<...>>`)
and consult it in the fetch branch before issuing a new `uiClient.fileSymbols` call — if a
request for `id` is already outstanding, either no-op (the eventual resolution already handles
re-adding via the existing `expandedFiles.has(id)` check) or await/reuse the existing promise
instead of starting a second one:

```ts
let pendingFileSymbols = new Set<string>();
// ...
// fetch branch:
if (pendingFileSymbols.has(id)) {
  // an identical request is already in flight; expandedFiles already
  // reflects the optimistic "expanded" state, nothing further to do
  return;
}
pendingFileSymbols.add(id);
uiClient.fileSymbols({ path: id }).then((response) => {
  pendingFileSymbols.delete(id);
  // ...existing body...
}).catch((err) => {
  pendingFileSymbols.delete(id);
  // ...existing body...
});
```

### CR-02: Expanding a file's symbols, then toggling any directory elsewhere, silently destroys the rendered symbols without updating `expandedFiles`

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:279-290` (`replace`), interacting with `web/src/routes/graph/+page.svelte:78-80` (`elements` derivation) and `255-313` (`toggleFile`'s `addedElements` seam)
**Issue:**

`elements` in `+page.svelte` is `$derived(rollupToElements(graphState.response, expandedDirs))`
— it never includes symbol elements; those exist in the live cytoscape instance *only* because
`toggleFile` pushed them in through the separate `addedElements` incremental seam
(`renderer.add()`). Any change to `expandedDirs` (expanding or collapsing **any** directory,
including one unrelated to the currently-expanded file) recomputes `elements`, which fires
`GraphCanvas`'s second `$effect` (line 583-590) and calls `renderer.replace(current)`. `replace`
is implemented as:

```js
replace(newElements: unknown[]) {
    opts.cy.startBatch();
    opts.cy.elements().remove();       // removes EVERYTHING, including symbols added via add()
    opts.cy.add(newElements as any);   // re-adds only the directory-level rollup — no symbols
    opts.cy.endBatch();
    runLayout(performance.now(), false);
},
```

`opts.cy.elements().remove()` unconditionally clears the *entire* live element set, including
any symbol children a prior `toggleFile` added via the incremental `add()` path. The subsequent
`opts.cy.add(newElements)` only re-adds what `rollupToElements` produced — the directory/file
rollup, never symbols. The result: a user who expands file A's symbols, then clicks *any*
directory node (expand or collapse — anything that changes `expandedDirs`), silently loses the
rendered symbol view for file A, with **no error, no state update, and no user-visible
indication**. `+page.svelte`'s `expandedFiles` set still contains `A`, so the "Expanded — click
to collapse" button for A (`graph-collapse-file-A`, rendered at lines 425-435) remains visible
and implies A's symbols are still showing when they are not. Clicking that stale button calls
`toggleFile(A)`'s collapse branch, which calls `removeByIds` with ids that no longer exist in
the live instance (silently no-ops, per `GraphCanvas.svelte:340-344`'s `el.length > 0` guard) —
so the *symptom* self-heals on the next click, but the interim state (button present, claiming
an expansion that has already been silently destroyed) is a real, reachable correctness defect
in the phase's flagship interaction (GRF-02 directory expansion combined with GRF-03 file
expansion). No test in `web/tests/graph-expand.test.ts` or `graph-expansion.test.ts` exercises
"expand a file, then toggle an unrelated directory" — I checked (`rg` for `toggleDirectory` /
mixed-interaction patterns in `graph-expand.test.ts` returns nothing beyond a single unrelated
`simulateTap('x')` call for a directory-only test).

This also contradicts `05-07-SUMMARY.md`'s own description of the incremental seam as "layered
on top of whatever the directory-level `elements` prop currently holds, never recomputed by,
and **never invalidating, that separate full-replace path**" — the implementation does exactly
what that sentence says it does not do.

**Fix:** Either (a) have `replace()` re-apply any currently-expanded files' cached symbol
elements after the directory-level swap (the route already has `fileSymbolsCache` +
`expandedFiles` — `+page.svelte` could pass the full element set, including live symbol
elements, through one `elements` value instead of splitting directory-rollup and symbol
expansion across two seams that don't compose), or (b) have `toggleDirectory` proactively
collapse every currently-expanded file (clearing `expandedFiles`/issuing the matching
`removedElementIds`) before applying a directory-level change, so the route's own state and the
rendered graph never diverge. Option (a) is more consistent with this component's stated
"renderer swap seam" design — the seam should not have two independently-mutating element
sources that can silently clobber each other.

## Warnings

### WR-01: `focus()` changes the viewport but never republishes the geometry seam, leaving click coordinates stale

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:363-376` (`focus`), `600-604` (the third `$effect`)
**Issue:**

The rendered-geometry seam (`window.__codegraphFileGraphGeometry`, documented at lines 69-108)
publishes `x`/`y` as container-relative pixel coordinates "accounting for the current
pan/zoom/fit" — the doc comment is explicit that this is what lets a real-mouse driver click a
specific node, since canvas rendering has no per-node DOM element. `removeByIds()` was already
found and fixed (05-07) to republish this seam directly, because removing elements changes the
model without going through `layoutstop`. But `focus()` — called whenever `focusNodeIds`
changes (the cycle-focus control in `+page.svelte:133-140`) — calls `opts.cy.fit(matched)`,
which changes the current pan/zoom exactly as much as a layout settle does, yet never calls
`computeGeometry`/`onGeometry` afterward. Any previously-published geometry entry's `x`/`y`
becomes stale the instant a cycle-focus fit runs: the pixel coordinates it reports no longer
correspond to the actual on-screen position of that node.

This is the same class of bug the project already found and fixed once in `removeByIds()`
(05-07-SUMMARY.md's Deviation 1) — the fix pattern is established but was not applied to this
second geometry-affecting operation. No current automated check clicks through the geometry
seam after activating cycle-focus, so this has not yet produced a visible failure, but it is a
real gap in a seam whose own documentation promises "the live, current element list... not
'current as of the last layout.'"

**Fix:** Call `computeGeometry(opts.cy)` + `opts.onGeometry(...)` at the end of `focus()`,
mirroring `removeByIds()`'s fix:

```js
focus(ids: string[]) {
    if (ids.length === 0) return;
    if (typeof opts.cy.collection !== 'function' || typeof opts.cy.getElementById !== 'function') {
        return;
    }
    let matched: any = opts.cy.collection();
    for (const id of ids) {
        matched = matched.union(opts.cy.getElementById(id));
    }
    if (matched.length > 0) {
        opts.cy.fit(matched);
        const geometry = computeGeometry(opts.cy);
        if (geometry !== undefined) opts.onGeometry(geometry);
    }
}
```

## Info

### IN-01: `int32` truncation of `int64` counters at the wire boundary is unguarded, though not realistically reachable

**File:** `internal/uiserver/handlers.go:1004,1017,1077` (`fileGraphToProto`, `fileGraphNodeToProto`, `fileSymbolsToProto`)
**Issue:** `int32(result.CycleCount)`, `int32(n.CycleID)`, and `int32(result.Total)` narrow Go
`int`/`int64` values to `int32` with no bounds check. A repository would need over 2^31 distinct
file-graph cycles or symbols in one file to overflow this, which is not realistic for any real
corpus this project targets (even `google/guava` at full scale measures in the low thousands).
Not worth gating the phase on, but worth a one-line comment noting the assumption if a future
corpus target grows dramatically.
**Fix:** No action required now; optionally add a doc comment noting the assumed upper bound.

## Resolution

_Applied by the gsd-code-fixer against this review's findings, in the fix scope (critical +
warning). See `05-REVIEW-FIX.md` for the full report._

- **CR-01 — fixed.** `web/src/routes/graph/+page.svelte`'s `toggleFile` now consults a
  `pendingFileSymbols` set at the top of the fetch branch and no-ops the dispatch (while
  still marking the file expanded) if a request for the same path is already in flight — a
  re-expand arriving before the earlier request settles is applied by that request's own
  `.then` instead of issuing a duplicate one. A new "CLICK FOUR and FIVE" test in
  `web/tests/graph-expand.test.ts` extends the existing three-click contract to a fourth and
  fifth click and was confirmed RED (2 requests instead of 1) before the fix.
  Commit: `4b695eb7`.
- **CR-02 — fixed.** Took option (a) from the review's own fix guidance: `GraphCanvas.svelte`'s
  `createFileGraphRenderer` now tracks `liveAddedElements` (everything `add()` merged in,
  minus what `removeByIds()` removed) and, inside `replace()`, re-applies whichever of them
  survive the swap — an added element whose own `parent` id is present among the new element
  set. This composes the incremental symbol seam with the full-replace path instead of the
  full-replace path silently destroying it; a symbol whose file collapses away is correctly
  dropped (its parent id no longer exists), while a symbol whose file remains visible now
  survives an unrelated directory's own expand/collapse. A new regression test in
  `web/tests/graph-expand.test.ts` (expand a root file's symbols, then toggle an unrelated
  directory, assert the symbols survive) was confirmed RED (symbols destroyed) before the
  fix. Commit: `2e07077d`.
- **WR-01 — fixed, as suggested.** `GraphCanvas.svelte`'s `focus()` now calls
  `computeGeometry()`+`onGeometry()` directly after `cy.fit()`, mirroring `removeByIds()`'s
  existing republish. A new regression test (real headless cytoscape) asserts the geometry
  seam republishes after `focus()` and was confirmed RED (no republish) before the fix.
  Commit: `23a3b377`.
- **IN-01 — deliberately deferred**, per this review's own text and an explicit maintainer
  instruction accompanying this fix pass: the `int32` narrowing is judged not realistically
  reachable (would require over 2^31 cycles/symbols in one file), and the review's own Fix
  section states "No action required now." Left unfixed; recorded here rather than silently
  dropped.
- The committed `web/build/` bundle was rebuilt (commit `98cd41dd`) and `task web:drift`
  reconfirmed PASS after the three source fixes above. A staging gap in that commit (9
  new output files `git commit web/build` failed to add, though `task web:drift` still
  reported PASS since it re-hashes disk state rather than the git tree) was found and
  closed in a follow-up commit (`fad3b39`) — see `05-REVIEW-FIX.md` for detail.

---

_Reviewed: 2026-08-31T16:37:54Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
