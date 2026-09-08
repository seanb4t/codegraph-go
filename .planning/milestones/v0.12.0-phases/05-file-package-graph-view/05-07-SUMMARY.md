---
phase: 05-file-package-graph-view
plan: 07
subsystem: ui
tags: [cytoscape, cytoscape-elk, playwright, expansion, hit-testing, collapse-affordance]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-08's rollupToElements(response, expandedDirs), the construction/replace renderer split, and the geometry seam; 05-06's FileSymbols rpc (Engine.FileSymbols, uiv1.FileSymbolsRequest/Response, the shared 15-field Node message); 05-05's onEdgeSelected/onBackgroundTapped seam and cycle styling; 05-08's assignment of WINDOWS.md entry 27 to this plan"
provides:
  - "file-graph-transform.ts's symbolElementsForFile(filePath, response) and symbolElementIdsForFile(filePath, response) — the second element builder, parenting symbol leaves to their file at creation time, with collision-proof NUL-separated ids"
  - "graph-style.ts's node[?isSymbol] selector"
  - "GraphCanvas.svelte's addedElements/removedElementIds incremental (non-replace) props, createFileGraphRenderer.add()/removeByIds(), and a three-way onNodeSelected kind discriminator (directory/file/symbol)"
  - "+page.svelte's toggleFile/toggleDirectory dispatchers, file-symbols expansion state (fileSymbolsCache, expandedFiles, fileExpansionFailures), a truncation notice, a per-file failure state, and an explicit collapse-affordance button list (WINDOWS.md 27's required fix, serving both directory and file levels)"
  - "web/scripts/graph-collapse-affordance-check.mjs — a real-browser, trusted-mouse-input proof that the collapse-affordance button reliably re-collapses a compound at both levels, at both this repository's and google/guava's scale; corpora/graph-collapse-affordance-check.json is the committed repo-scale record"
  - "A fix to GraphCanvas.svelte's removeByIds(): it now republishes the geometry seam directly, since a removal-only operation has no layoutstop event to hang the republish on — found live, not by a pre-written test"
affects: []

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 21463
  tasks: 3
  commits: 9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A SECOND, incremental element seam alongside the existing full-replace one: addedElements/removedElementIds are plain-data props whose IDENTITY change (a new array reference) triggers GraphCanvas's own add()/removeByIds(), layered on top of whatever the elements prop (rollupToElements' directory-level output) currently holds — never recomputed by, and never invalidating, that separate full-replace path."
    - "Deterministic, collision-proof symbol ids: `symbol\\0<filePath>\\0<wireSymbolId>`, the same control-character-separator discipline buildEdgeElements already used for its own composite key, extended to a second id-construction site. symbolElementIdsForFile shares the exact same construction function symbolElementsForFile uses, so the route's removal-prop ids can never drift out of sync with the ids the elements themselves carry."
    - "Optimistic expandedFiles + stale-response cache-not-discard: a file is added to the expanded set BEFORE its request settles, so a second click arriving mid-flight is recognised as a collapse (not a second expand); the guard against applying a response that outlived its consumer sits at the point of APPLICATION, not dispatch, and always caches the response regardless of whether it applies it — the once-per-file contract and the stale-response guard hold together, not one at the other's expense."
    - "Explicit collapse affordance over canvas re-click: a real DOM `<button>` list, calling the SAME toggleDirectory/toggleFile the canvas tap handler calls, closes a genuine sub-2px hit-testing defect a canvas-rendered compound cannot be made reliably clickable to fix (three prior mitigations — cytoscape padding, ELK graph-level padding, ELK per-node padding — all measured ineffective in 05-08). One collapse rule, two ways to reach it."

key-files:
  created:
    - web/scripts/graph-collapse-affordance-check.mjs
    - corpora/graph-collapse-affordance-check.json
  modified:
    - web/src/lib/components/graph/file-graph-transform.ts
    - web/src/lib/components/graph/graph-style.ts
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/routes/graph/+page.svelte
    - web/tests/graph-expand.test.ts (new)
    - web/tests/graph-expansion.test.ts
    - web/tests/graph-collapse.test.ts
    - web/build (rebuilt twice — once for the feature, once more for the removeByIds fix)
    - .planning/WINDOWS.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "The window-27 fix is a real DOM button list, not a better-aimed canvas click. The orchestrator's own defect brief was explicit that three padding-based mitigations already failed and that a genuinely hit-testable control was required, naming a caret/disclosure button as one acceptable shape. A plain HTML button sidesteps cytoscape's canvas hit-testing entirely rather than trying to win a sub-2px margin, and serves BOTH directory and file collapse through the exact same toggleDirectory/toggleFile functions the canvas tap handler already calls — no second collapse rule."
  - "addedElements/removedElementIds are a SEPARATE prop pair from `elements`, not a change to rollupToElements' contract. Symbol expansion is per-file and incremental; folding it into the directory-level rollup would mean every file-symbol toggle recomputes and replaces the WHOLE graph's element array, discarding the accepted-consequence tradeoff 05-08 already made explicit for directory expansion (full re-layout on every change) and applying it needlessly to a per-file action that does not need to touch anything outside one file's own children."
  - "removeByIds() must republish the geometry seam even though it never runs a layout. Discovered by running the live-browser real-mouse check this plan added, not predicted in advance: the geometry seam only republished from inside runLayout's layoutstop handler, so a collapse-only operation (deliberately layout-free, since removing children never needs to reposition anything else) left the seam reporting a symbol id the model no longer had. The on-screen pixels were correct throughout (cytoscape redraws on any mutation independent of layout) — only the JS seam a script or future consumer reads was stale. Fixed by calling computeGeometry()+onGeometry() directly and synchronously inside removeByIds()."
  - "The canvas-click check script must re-query the graph canvas's own on-screen position before EVERY click, not once at page load. Also discovered live: clicking the DOM collapse-affordance button scrolls it into view, which can shift the page's scroll offset and therefore the canvas element's screen coordinates — a cached container box silently aimed the NEXT canvas click at the wrong page location."

patterns-established:
  - "Two-tier click-target reliability: canvas-click for EXPAND (where the target is always a full-area leaf node at first contact, reliably hit-testable) and a real DOM button for COLLAPSE (where the target has shrunk to a sub-2px compound margin once children exist). This is not a general 'canvas clicks are unreliable' finding — expand-by-canvas-click is proven reliable at both directory and file level in this same plan's own live checks — it is specifically the collapse-of-a-compound-with-children case that needed a different mechanism."

requirements-completed: [GRF-03, GRF-02]

coverage:
  - id: D1
    description: "Clicking a file node in the graph reveals the symbols declared in that file, inside the same view, as children of that file node — via one file-symbols request, parented at creation time, never a second grouping mechanism."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (symbolElementsForFile shape/count/parent tests; CLICK ONE case: exactly one request, child count equals response symbol count)"
        status: pass
      - kind: e2e
        ref: "corpora/graph-collapse-affordance-check.json (real Playwright, this repository: file expand 111->112 nodes); guava-scale run 136->142 nodes, recorded in this SUMMARY"
        status: pass
    human_judgment: false
  - id: D2
    description: "One file, three clicks, one request: expand (fetch), collapse to zero (no request), re-expand from cache (no request, exact count, never doubled)."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (CLICK ONE/TWO/THREE cases, each asserting the cumulative fileSymbols request count for the path stays exactly 1 and click three asserts the EXACT child count, not merely non-zero)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Collapsing an expanded file removes its symbol children and leaves every other node, including another expanded file's own children, untouched."
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts ('leaves every other file node and every edge element untouched'; CLICK TWO's other-file child-count assertion)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A truncated symbol list says so, naming the shown count and the true total; an untruncated response renders no such statement."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (both truncation directions asserted in the same test)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A failed file-symbols request renders the named failure from classifyRpcError, leaves the file unexpanded, and leaves the rest of the graph intact and usable."
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts ('a rejected file-symbols request renders the named failure...')"
        status: pass
    human_judgment: false
  - id: D6
    description: "A file-symbols response resolving after its file was collapsed, or after the route unmounted, applies ZERO elements and constructs NO children; the stale response is cached (not discarded) so a later re-expand still needs no second request."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (STALE resolved-after-collapse: zero applied elements, total count restored, then re-expand from cache with no new request; STALE resolved-after-unmount: exactly one destruction, zero additions after it, no throw)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Expansion changes only the expanded file's subtree: node/edge counts for every other file are unchanged, and every unaffected file node keeps its identity and its compound parent across the expansion."
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (identity-and-containment case: 3+ unaffected file nodes checked by id+parent before/after)"
        status: pass
    human_judgment: false
  - id: D8
    description: "Re-running the layered layout after an expansion MAY move unrelated nodes (D-04's accepted consequence) — measured, not asserted away, as the finding Phase 6's LIV-04 inherits."
    verification:
      - kind: unit
        ref: "web/tests/graph-expand.test.ts (real headless cytoscape+elk: 0 of 6 unaffected nodes moved, 0.00 model-unit displacement, at an 8-node fixture)"
        status: pass
    human_judgment: true
    rationale: "The measured number at this plan's small fixture scale is honest but not necessarily representative of a large real corpus's re-layout behaviour; a human should read this alongside the guava-scale live-browser observations recorded in this SUMMARY's Accomplishments before treating the fixture's 0.00px figure as the whole story."
  - id: D9
    description: "The explicit collapse affordance (WINDOWS.md 27) reliably re-collapses a compound, by real trusted mouse input, at both the directory level and the file level, at both this repository's own scale and google/guava's scale."
    verification:
      - kind: e2e
        ref: "corpora/graph-collapse-affordance-check.json (this repository: dir 106->111->106 exact, file 111->112->111 exact, 0 page errors); a second run confirmed non-flaky; a third run against a fresh google/guava clone (135 collapsed dirs, 163 cycles): dir 135->136->135 exact, file 136->142->136 exact, 0 page errors — recorded verbatim in this SUMMARY"
        status: pass
    human_judgment: true
    rationale: "WINDOWS.md 27 is a real-mouse hit-testing defect; the ledger entry itself (recorded fixed via gsd-tools windows.fixed 27) is the audit trail, but a human reading this SUMMARY should see the actual before/after node counts from all three runs rather than trusting a boolean success flag alone."
  - id: D10
    description: "The committed bundle is regenerated from this plan's source and the two-part drift guard reports both hashed counts; supply-chain gates (audit, lockfile, deps:strict) stay green."
    verification:
      - kind: other
        ref: "task web:drift (108 source files / 32 output files, both digests match), task web:audit (0 advisories), task web:lockfile (242/242 integrity-bearing), task web:deps:strict (strictDepBuilds true, 0 denials)"
        status: pass
    human_judgment: false

# Metrics
duration: ~100min
completed: 2026-08-31
status: complete
---

# Phase 5 Plan 7: File-to-Symbol Expansion Summary

**Clicking a file expands it into its symbols in place — one request, cached thereafter — and an explicit collapse-affordance button (not a canvas re-click) closes WINDOWS.md 27 at both directory and file level, proven with real trusted-mouse Playwright input against this repository's own index and a freshly indexed google/guava.**

## Performance

- **Duration:** ~100 min (extensive live-browser investigation and two live-discovered bug fixes accounted for a large share of this)
- **Started:** 2026-08-31T15:37:00Z (approx, first Read tool call)
- **Completed:** 2026-08-31T16:26:00Z
- **Tasks:** 3 (each RED then GREEN, per this plan's `type: tdd`, plus a copy pass, a bundle rebuild, and a live real-mouse verification pass)
- **Files modified:** 10 source/test files (1566 insertions / 37 deletions in web/src + web/tests + web/scripts, excluding web/build), plus web/build rebuilt twice (22 + 21 files changed across the two rebuilds)

## Accomplishments

- **`symbolElementsForFile(filePath, response)` builds symbol elements DOM-free, parented at creation time.** A second, deterministic id-construction function (`symbol\0<filePath>\0<wireSymbolId>`) guarantees no collision with any file or directory path already in the graph — proven against a fixture whose file and a symbol inside it share the same name. `symbolElementIdsForFile` shares the exact same construction, so the route's removal-prop ids can never drift out of sync with the ids the elements themselves carry.
- **`GraphCanvas.svelte` grows a second, incremental element seam.** `addedElements`/`removedElementIds` props (plain data, array-identity-triggered) drive `createFileGraphRenderer.add()`/`.removeByIds()` — a MERGE and a REMOVE, layered on top of whatever the existing `elements` prop's full-replace path currently holds, never recomputing it. `onNodeSelected` now carries a three-way kind discriminator (`directory`/`file`/`symbol`), read from element data, never a class list.
- **`+page.svelte`'s `toggleFile` delivers the verified three-click contract: expand (1 request), collapse to zero (0 requests), re-expand from cache (0 requests, exact count).** Expansion state is added to `expandedFiles` OPTIMISTICALLY before a request settles, so a second click arriving mid-flight is recognised as a collapse rather than a duplicate expand — this is what keeps the cumulative request count at exactly 1 across all three clicks. The stale-response guard sits at the point of APPLICATION (not dispatch): a response is always cached, but only applied to the canvas when its file is still expanded AND the route is still mounted — carrying forward 05-03's pending-request-outliving-its-consumer lifecycle to this second async path in the same vocabulary.
- **A truncation notice and a per-file failure state, both scoped.** A capped symbol list names the shown count and the true total; a rejected request renders the one existing `classifyRpcError` failure and leaves the rest of the graph intact — one file failing to expand is not the view failing.
- **WINDOWS.md 27 is CLOSED with real trusted-mouse evidence, at both levels and both scales.** The maintainer-assigned defect (a canvas-rendered compound's own border and its topmost child are under ~2px apart once it has children — three prior padding-based mitigations all measured ineffective) is fixed with an explicit DOM `<button>` collapse-affordance list, calling the SAME `toggleDirectory`/`toggleFile` the canvas tap handler calls. `web/scripts/graph-collapse-affordance-check.mjs`, a Playwright check driving real mouse input (never dispatched events), proved the five-phase sequence (canvas-expand dir -> button-collapse dir -> canvas-re-expand dir -> canvas-expand file -> button-collapse file) against this repository's own index (`106 -> 111 -> 106 -> 111 -> 112 -> 111`, exact at every step, 0 page errors, confirmed non-flaky across two runs) AND against a freshly cloned google/guava (`135 -> 136 -> 135 -> 136 -> 142 -> 136`, exact, 0 page errors). The repo-scale record is committed at `corpora/graph-collapse-affordance-check.json`; the ledger entry itself was marked `fixed` via `gsd-tools query windows.fixed 27` — the tool's own verb, never a hand-edited table row.
- **Two real bugs found and fixed during the live-browser verification pass, neither predicted by the pre-written test suite:**
  1. `removeByIds()` removed the element from the cytoscape model correctly (confirmed via `cy.getElementById(...).length === 0` and the correct on-screen redraw) but never republished the geometry seam, since that seam only republished from inside `runLayout`'s `layoutstop` handler and a removal-only operation deliberately never runs a layout. Fixed by calling `computeGeometry()`+`onGeometry()` directly inside `removeByIds()`. A new regression test (real headless cytoscape) asserts the seam updates synchronously; confirmed genuine RED against the reverted fix before landing it.
  2. The live-check script's own canvas-click helper cached the graph canvas's on-screen position once at page load — but clicking the new DOM collapse button scrolls it into view, which can shift the page's scroll offset and silently misaim the next canvas click. Fixed by re-querying the container's bounding box before every canvas click.
- **A copy pass removed 8 instances of leaked plan/requirement identifiers** — `(05-07, GRF-03)`, `(review M-3)`, `(WINDOWS.md 27)` — that this plan's own earlier commits had introduced into source comments (the same class of leak Phase 4 shipped once and 05-08 had to fix again). `graph-summary` copy now also mentions selecting a file reveals its symbols; the collapse-affordance list gained a caption ("Expanded — click to collapse:").
- **The committed bundle was rebuilt twice and every supply-chain gate re-verified green each time:** final state — `task web:drift` (108 source files / 32 output files, both digests match), `task web:audit` (0 advisories), `task web:lockfile` (242/242 integrity-bearing), `task web:deps:strict` (strictDepBuilds true, 0 denials), `task web:test` (412/412), `GOTOOLCHAIN=go1.26.5 task test:unit` (all packages green, backend untouched), `task proto:drift` (4 generated files, byte-identical), `cd web && pnpm check` (0 errors/0 warnings).
- **Bundle-size change:** this plan's own change (05-05's post-rebuild baseline to this plan's final commit) is +3,808 bytes (2,714,541 -> 2,718,349 bytes). The cumulative change across the whole phase (end of Phase 4's `04-07` rebuild to now) is +1,889,730 bytes (828,619 -> 2,718,349 bytes) — dominated by the cytoscape + cytoscape-elk + elkjs dependency tree the graph route bundles, first introduced in 05-03, not by this plan alone.
- **`GRF-02` and `GRF-03` both marked complete in REQUIREMENTS.md**, via `gsd-tools query requirements.mark-complete` (the tool's own verb). All four of this phase's GRF requirements (GRF-01 through GRF-04) are now `Complete`. Phase 5 now has 8/8 plans summarized.

## Task Commits

Each TDD task carries a RED-then-GREEN pair; the live-verification pass added one Rule-1 fix commit:

1. **Task 1 RED:** `839cfaf6` — `test(05-07): add failing tests for symbol elements parented to their file` (8/8 new assertions failed, genuinely: 7x "symbolElementsForFile is not a function", 1x style selector undefined)
2. **Task 1 GREEN:** `9b4a2376` — `feat(05-07): symbol elements parented to their file, built DOM-free`
3. **Task 2 RED:** `f8464a51` — `test(05-07): add failing tests for tap-to-expand/collapse/re-expand` (10 of 11 new cases failed genuinely; the directory-no-request case passed as an intended negative control both before and after)
4. **Task 2 GREEN:** `9a16b501` — `feat(05-07): tap-to-expand a file into its symbols, tap again to collapse`
5. **Task 2 (M-9 finding, added after GREEN):** `13a5b8a5` — `test(05-07): measure post-expansion node displacement against a real layout (review M-9)` (measurement test, no implementation change needed; 0 of 6 unaffected nodes moved at fixture scale)
6. **Task 3a (copy pass):** `3f9fc91e` — `docs(05-07): copy pass — remove leaked plan/review identifiers, add expansion copy`
7. **Task 3b (bundle rebuild):** `cf31d8e0` — `feat(05-07): rebuild the committed bundle for the file-symbol expansion source changes`
8. **Live-verification Rule-1 fix:** `1f01502e` — `fix(05-07): republish the geometry seam on removeByIds, not only on layoutstop`
9. **Live-verification evidence (WINDOWS.md 27 closure) + second bundle rebuild:** `d9b6a1ed` — `test(05-07): close WINDOWS.md 27 with real-mouse proof of the collapse-affordance button`

**Plan metadata:** (this commit) — completes this SUMMARY, `STATE.md`, `ROADMAP.md`, `REQUIREMENTS.md`.

_Note: TDD tasks each carry a RED then GREEN commit per this plan's `type: tdd`; Task 2 additionally carries the M-9 measurement addition, and the live-verification pass (part of Task 3) carries its own Rule-1 fix commit before the evidence-and-rebuild commit._

## Files Created/Modified

- `web/src/lib/components/graph/file-graph-transform.ts` — `symbolElementsForFile`, `symbolElementIdsForFile`, the extended `FileGraphNodeData` type (`isSymbol`/`kind`/`startLine`).
- `web/src/lib/components/graph/graph-style.ts` — `node[?isSymbol]` selector.
- `web/src/lib/components/graph/GraphCanvas.svelte` — `addedElements`/`removedElementIds` props, `add()`/`removeByIds()` renderer methods (the latter fixed mid-plan to republish geometry), the three-way `onNodeSelected` kind discriminator, `runLayout`'s `resizeAfter` parameter.
- `web/src/routes/graph/+page.svelte` — `toggleDirectory`/`toggleFile`, file-symbols expansion state, truncation notice, per-file failure state, the collapse-affordance button list.
- `web/tests/graph-expand.test.ts` (new, 21 tests) — Task 1 + Task 2 assertions, the M-9 displacement measurement, and the removeByIds geometry-republish regression test.
- `web/tests/graph-expansion.test.ts` — extended mock with a `fileSymbols` stub; one test reworded to state what it actually now proves (isolation between the two async paths) since 05-08's original "a file tap does nothing" assumption is exactly what this plan supersedes.
- `web/tests/graph-collapse.test.ts` — extended the module-surface SET-equality guard to include the two new transform exports.
- `web/build` — rebuilt twice (source-file count unchanged at 108 both times; the new script and corpora JSON stay outside the hashed set, matching prior precedent).
- `web/scripts/graph-collapse-affordance-check.mjs` (new) — the real-browser, trusted-mouse-input WINDOWS.md-27 closure check.
- `corpora/graph-collapse-affordance-check.json` (new) — the committed repo-scale success record.
- `.planning/WINDOWS.md` — entry 27 marked `fixed` (via the ledger's own verb).
- `.planning/REQUIREMENTS.md` — GRF-02 and GRF-03 marked complete (all four GRF requirements now Complete).

## Decisions Made

See `key-decisions` in the frontmatter above for the four decisions made during execution (the DOM-button collapse affordance, the separate incremental element seam, the removeByIds geometry-republish fix, and the live-check script's per-click container-box re-query fix).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `removeByIds()` never republished the geometry seam**
- **Found during:** Task 3's live-browser real-mouse proof of the collapse-affordance button (not predicted by the pre-written unit test suite, which asserts over the FakeCore's own element list rather than the geometry seam)
- **Issue:** The geometry seam only republished from inside `runLayout`'s `layoutstop` handler; `removeByIds()` deliberately never runs a layout (removing a symbol's children needs no re-placement of anything else), so a file collapse via the button left the seam reporting a symbol id the cytoscape model no longer had. The on-screen pixels were correct throughout (cytoscape redraws on any mutation independent of layout) — only the seam a script (or any future consumer) reads was stale.
- **Fix:** `removeByIds()` now calls `computeGeometry(opts.cy)` and `opts.onGeometry(...)` directly, synchronously, after the batch removal.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`, `web/tests/graph-expand.test.ts` (new regression test)
- **Verification:** New test confirmed genuine RED against the reverted fix (manually reverted, observed failing, restored), then GREEN with the fix; full suite 412/412 after.
- **Committed in:** `1f01502e`

**2. [Rule 3 - Blocking] The live-check script's own canvas-click helper misaimed after a button click**
- **Found during:** Building `graph-collapse-affordance-check.mjs` — Phase 3 (re-expand a directory after collapsing it via button) consistently failed to hit the target across all jitter offsets, while Phases 1-2 (before any button click) worked reliably.
- **Issue:** The container's on-screen bounding box was computed once at page load and reused for every subsequent canvas click. Clicking the new DOM collapse-affordance button (Playwright's `.click()` scrolls its target into view before clicking) shifted the page's scroll offset, invalidating the cached box for every canvas click after that point.
- **Fix:** Re-query the container's bounding box (`getContainerBox(page)`) immediately before every canvas click, never cached.
- **Files modified:** `web/scripts/graph-collapse-affordance-check.mjs`
- **Verification:** All five phases of the check sequence then succeeded reliably, confirmed across two separate runs against this repository and a third against google/guava.
- **Committed in:** `d9b6a1ed`

**3. [Rule 2 - Missing Critical] The plan's own text did not describe a collapse-affordance UI control; the orchestrator's dispatch prompt explicitly assigned WINDOWS.md 27 to this plan and required it**
- **Found during:** Reading the dispatch prompt's `<you_own_a_known_open_defect>` section before starting Task 2
- **Issue:** 05-07-PLAN.md predates 05-08's discovery and formal assignment of the collapse-by-click hit-testing defect (WINDOWS.md 27); the plan's own Task 2 action text describes only the canvas tap-to-expand/collapse mechanism, which inherits the identical defect at file level exactly as it blocks directories.
- **Fix:** Added the explicit DOM-button collapse-affordance list (see Accomplishments and key-decisions above), serving both directory and file levels through the existing `toggleDirectory`/`toggleFile` functions.
- **Files modified:** `web/src/routes/graph/+page.svelte`, `web/tests/graph-expand.test.ts`, `web/scripts/graph-collapse-affordance-check.mjs`, `corpora/graph-collapse-affordance-check.json`
- **Verification:** Unit tests for the button's collapse behavior; real-mouse Playwright proof at both levels and both scales (see Accomplishments)
- **Committed in:** `9a16b501` (UI + unit test), `d9b6a1ed` (real-mouse proof)

---

**Total deviations:** 3 auto-fixed (1 Rule 1 bug, 1 Rule 3 blocking issue in a verification script, 1 Rule 2 missing-critical addition explicitly directed by the dispatch prompt). **Impact on plan:** All three were necessary — the geometry-seam bug was a genuine defect discovered only through live verification, the click-helper bug was blocking the verification itself, and the collapse affordance is the exact fix the orchestrator's own defect brief required before window 27 could be closed. No scope creep beyond what was explicitly directed or discovered through the plan's own required verification.

## A note on a pre-existing requirement-tracking oddity (not a deviation from THIS plan, disclosed for transparency)

`.planning/REQUIREMENTS.md` already showed `GRF-03` (**User can drill into a file to see its symbols**) as `[x] Complete` BEFORE this plan started — marked by `05-06-SUMMARY.md`, which delivered only the `FileSymbols` wire rpc, not the UI drill-down the requirement text actually describes. This plan is the one that genuinely delivers the described capability, so the requirement's Complete status is now correct in substance even though it was marked complete a plan early. Not corrected here (REQUIREMENTS.md's checkbox state is a tool-owned value already applied via `requirements.mark-complete`, and re-running that verb this plan is idempotent, not destructive) — recorded as an observation for whoever next audits this phase's requirement traceability.

## Issues Encountered

- **A CSP violation appears in the console on EVERY route, not introduced by this plan.** `Loading the image 'data:image/svg+xml,...svelte-logo...' violates the following Content Security Policy directive: "default-src 'self'"` — confirmed present on both `/graph` and `/` (the home route) before any interaction, tracing to the default SvelteKit favicon's inline data-URI and this app's own strict `default-src 'self'` CSP with no `img-src` exception. Not a cytoscape/elk renderer error, not introduced by this plan, out of this plan's scope to fix — recorded honestly per Task 3's own instruction to record console findings rather than silently narrow the claim.
- **WINDOWS.md entry 26 (the deterministic cytoscape-elk adapter `notify` error) did not fire in any session this task ran** (repo-scale checks, guava-scale checks, or the screenshot passes) — `pageErrorCount: 0` in every recorded run. Per the plan's own warning, this is recorded as an observation, not treated as evidence the defect is gone; it is a known-flaky, non-fatal error that has not fired consistently across sessions in this phase's history either.
- **At google/guava scale (135 collapsed directories, 163 cycles, 833 edges), the rendered graph is visually very dense** — individual node boxes are small and edge lines cross heavily at default zoom, confirmed via screenshot. This matches 05-05's own prior finding at the same scale. Expansion (both directory and file) and collapse (both canvas and button) all remained FUNCTIONALLY reliable at this density — the density is a legibility observation, not a functional failure, and is not new to this plan.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **Phase 5 is now complete: 8/8 plans have summaries, and all four GRF requirements (GRF-01 through GRF-04) are marked Complete.**
- **Phase 6's LIV-04 inherits a sized constraint, not a surprise:** re-running the layered layout after a file expansion MAY move unrelated nodes (D-04's accepted consequence). At this plan's own small fixture scale, the measured displacement was 0 of 6 unaffected nodes / 0.00 model units — genuinely measured, not asserted away, but not necessarily representative of a large real corpus. Phase 6 will either need incremental layout or an explicit position-preservation pass if it wants stable in-place live updates at scale.
- **The collapse-affordance button pattern (a real DOM control mirroring a canvas gesture) is now established** for any future plan that needs a reliably clickable interaction this renderer's canvas hit-testing cannot guarantee.
- No blockers.

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-31*

## Self-Check: PASSED
- `test -f web/scripts/graph-collapse-affordance-check.mjs` → FOUND
- `test -f corpora/graph-collapse-affordance-check.json` → FOUND
- `git log --oneline --all | grep -q 839cfaf6` → FOUND
- `git log --oneline --all | grep -q 9b4a2376` → FOUND
- `git log --oneline --all | grep -q f8464a51` → FOUND
- `git log --oneline --all | grep -q 9a16b501` → FOUND
- `git log --oneline --all | grep -q 13a5b8a5` → FOUND
- `git log --oneline --all | grep -q 3f9fc91e` → FOUND
- `git log --oneline --all | grep -q cf31d8e0` → FOUND
- `git log --oneline --all | grep -q 1f01502e` → FOUND
- `git log --oneline --all | grep -q d9b6a1ed` → FOUND
- `cd web && GOTOOLCHAIN=go1.26.5 pnpm exec vitest run --reporter=verbose` → PASS (412/412)
- `cd web && pnpm check` → PASS (0 errors, 0 warnings, 1152 files)
- `task web:drift` → PASS (108 source / 32 output, both digests match)
- `task web:audit` → PASS (0 advisories)
- `task web:lockfile` → PASS (242/242 integrity-bearing)
- `task web:deps:strict` → PASS (strictDepBuilds true, 0 denials)
- `GOTOOLCHAIN=go1.26.5 task test:unit` → PASS (all packages)
- `GOTOOLCHAIN=go1.26.5 task proto:drift` → PASS (4 generated files, byte-identical)
- `rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0' web/src/routes/graph/ web/src/lib/components/graph/ | wc -l` → 0 (positive-controlled)
- `rg -l "from 'cytoscape" web/src | wc -l` → 1 (exact)
- `git status --porcelain web/build | wc -l` → 0 (clean); `git ls-files web/build | wc -l` → 33 (tracked)
