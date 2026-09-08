---
phase: 05-file-package-graph-view
plan: 05
subsystem: ui
tags: [cytoscape, cycles, edge-detail, data-table, tdd]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-08's rollupToElements(response, expandedDirs), collapsedDirElement's cycleIds union, GraphCanvas.svelte's construction/replace renderer split, and the maintainer's release-collapsed decision that unblocked this plan's precondition"
provides:
  - "file-graph-transform.ts: a graph-cycle class (plus a per-cycle graph-cycle-N discriminator) copied from wire cycle fields onto file nodes, collapsed directories, and in-cycle edges — never computed"
  - "graph-style.ts: node.graph-cycle / edge.graph-cycle selectors distinguishing cycle members on two non-colour channels (dashed border/line, distinct arrow shape) plus colour"
  - "GraphCanvas.svelte: a focusNodeIds prop (fit-to-exactly-these-ids via getElementById+union, never a selector string) and an onEdgeSelected/onBackgroundTapped event pair alongside the existing onNodeSelected"
  - "+page.svelte: a stated cycle count with an explicit no-cycles sentence, a wrapping next-cycle focus control grouped by typed cycleId/cycleIds (never a class list), and an edge detail region rendering per-kind counts through the shared DataTable"
  - "edge-kind-columns.ts: the {kind,count} ColumnDef set for the shared table, mirroring workbench/callers-columns.ts"
affects: [05-06, 05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 12700
  tasks: 3
  commits: 7

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Presentation-vocabulary-from-wire-data: a `classes` string on cytoscape element descriptors is a second encoding of a wire field already on `data`, never a new computation — file-graph-transform.ts's own header comment states this as a module contract, and a dedicated no-derivation test (adjacency closes a loop, wire cycle fields unset, zero classes produced) makes it observable, not merely asserted in prose."
    - "Typed-data grouping over class-string parsing: the route groups cycle members by reading `data.cycleId`/`data.cycleIds` directly, never an element's rendered class list — proven by a fixture that strips the discriminator classes while leaving the typed ids in place and requires the grouping to keep working (review M-8)."
    - "GraphCanvas's seam grows by PROP/EVENT PAIR, not by leaking cytoscape: focusNodeIds (data in) mirrors elements/style; onEdgeSelected/onBackgroundTapped (event out) mirror onNodeSelected — every new surface stays in the seam's existing vocabulary."

key-files:
  created:
    - web/src/lib/components/graph/edge-kind-columns.ts
    - web/tests/graph-cycles.test.ts
    - web/tests/graph-edge-detail.test.ts
  modified:
    - web/src/lib/components/graph/file-graph-transform.ts
    - web/src/lib/components/graph/graph-style.ts
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/routes/graph/+page.svelte
    - web/build (rebuilt)
    - .planning/WINDOWS.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "Cycle discriminator classes extend to COLLAPSED DIRECTORY nodes, not just file nodes — the gate-status briefing flagged this explicitly: 05-08's collapsed default means a file's cycleId is invisible until its directory is expanded, so collapsedDirElement's existing cycleIds union (a server-computed union, D-06) needed its own classes too. This was authored INTO the plan's Task 1 GREEN implementation, not left as a gap the route would have to work around."
  - "Cycle-focus grouping reads BOTH data.cycleId (file nodes) and data.cycleIds (collapsed directories), so the same code groups correctly whether a cycle's members are currently showing as files or collapsed into their parent directory — every file's cycle id is represented by exactly one element in any given view, so this always yields exactly cycleCount distinct groups regardless of expansion state."
  - "GraphCanvas's edge-selected handler defensively checks for source()/target() before calling them — an EARLIER task's (05-08's) own minimal cytoscape test double hands a node-shaped target to every 'tap'-keyed handler because it collapses selector-scoped registrations into one list; rather than editing that unrelated, already-passing test file, the new handler tolerates the shape it cannot rely on cytoscape's real selector scoping to exclude under that double. Found only because running the full suite (not just the new test file) surfaced a regression in web/tests/graph-expansion.test.ts."
  - "The literal edge-kind-columns.ts row shape ({kind,count}) intentionally does NOT import health/CountTable.svelte's CountRow type, even though the shapes are identical — the two are unrelated wire concepts (a health metric key vs. an edge's dependency kind) that happen to share a column layout; importing one from the other's module would couple them for no benefit."

patterns-established:
  - "A route-level headless test suite extends graph-tracer.test.ts's fake-cytoscape mocking pattern with test-file-local additions (collection/getElementById/fit for focus; selector-aware listener keys plus tapEdge/tapBackground helpers for edge/background taps) rather than modifying the shared fake — each test file owns exactly the surface its own behaviors need."

requirements-completed: [GRF-04]

coverage:
  - id: D1
    description: "A file node whose wire cycleId is non-zero, or a collapsed directory whose files union to a non-empty cycleIds set, renders with a cycle class; the unmarked side of the same fixture carries none."
    requirement: GRF-04
    verification:
      - kind: unit
        ref: "web/tests/graph-cycles.test.ts (marked/unmarked node assertion, collapsed-directory union assertion)"
        status: pass
    human_judgment: false
  - id: D2
    description: "An edge the server marked in-cycle carries the cycle class; an edge in the same fixture the server did not mark does not. A fixture whose adjacency closes a loop but whose wire cycle fields are unset produces zero cycle classes — the client never re-derives connectivity."
    requirement: GRF-04
    verification:
      - kind: unit
        ref: "web/tests/graph-cycles.test.ts (edge marked/unmarked; no-derivation guard over a closed-loop fixture with unset wire fields)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Cycle members and edges are distinguished on more than colour alone (dashed border/line, distinct arrow shape, plus colour), confirmed both as a plain-data style-sheet assertion and live in a real browser against this repository's own index and against google/guava (135 collapsed nodes, 163 cycles)."
    requirement: GRF-04
    verification:
      - kind: unit
        ref: "web/tests/graph-cycles.test.ts (style-sheet non-colour-property assertion)"
        status: pass
      - kind: automated_ui
        ref: "agent-browser live session against the built binary, screenshots /tmp/graph-after-focus.png and /tmp/guava-focus-1.png"
        status: pass
    human_judgment: true
    rationale: "Whether a dashed-border-plus-colour treatment reads as genuinely legible (not merely present) is a visual-quality judgment; the live screenshots are recorded evidence for a human to confirm, not a substitute for one."
  - id: D4
    description: "The graph view states how many dependency cycles exist without interaction (an explicit no-cycles sentence at zero), and a next-cycle focus control — absent entirely at zero cycles — steps through every cycle and wraps, grouping members by the typed cycleId/cycleIds already on element data rather than any class list."
    requirement: GRF-04
    verification:
      - kind: unit
        ref: "web/tests/graph-cycles.test.ts (count text, zero-cycle absence, exact focus-group membership, stable visiting order with wrap, class-stripped-fixture grouping)"
        status: pass
      - kind: automated_ui
        ref: "agent-browser live session: 'This repository has 10 dependency cycles.' visible pre-interaction; guava: 'This repository has 163 dependency cycles.', control usable, ~1s round trip per activation"
        status: pass
    human_judgment: false
  - id: D5
    description: "Selecting an edge shows its two file paths, its total count, and one row per wire kind with that kind's count, through the shared DataTable — a kind absent from the sparse map produces no row, and the per-kind counts sum to the shown total."
    requirement: GRF-02
    verification:
      - kind: unit
        ref: "web/tests/graph-edge-detail.test.ts (source/target/total text, exact row count equals map key count, absent-kind produces no row, counts sum to total, shared-table test id, replace-not-append, background-tap clears)"
        status: pass
      - kind: automated_ui
        ref: "agent-browser live session: 'internal/cli → internal/watch', 'Total dependency count: 2', rows calls:1 + instantiates:1 = 2"
        status: pass
    human_judgment: false
  - id: D6
    description: "The committed bundle is regenerated from this plan's source and the two-part drift guard reports both hashed counts; the shipped supply-chain gates (audit, lockfile, deps:strict) stay green."
    verification:
      - kind: other
        ref: "task web:drift (108 source files / 32 output files, both digests match), task web:audit (0 advisories), task web:lockfile (242/242 integrity-bearing), task web:deps:strict (strictDepBuilds true)"
        status: pass
    human_judgment: false

# Metrics
duration: 3h40min
completed: 2026-08-31
status: complete
---

# Phase 5 Plan 5: Cycle Visibility and Per-Kind Edge Detail Summary

**Cycle members and edges get a dashed-plus-colour style and a stated, steppable count; a selected edge's per-kind counts render through the shared DataTable — verified live against this repository's own index (10 cycles) and against a freshly indexed google/guava (163 cycles, 135 collapsed nodes).**

## Performance

- **Duration:** 3h 40min
- **Started:** 2026-08-31T14:04:00Z (approx, first Read tool call)
- **Completed:** 2026-08-31T17:44:00Z
- **Tasks:** 3 (each RED then GREEN, per this plan's `type: tdd`), plus a bundle rebuild and a live-browser verification pass
- **Files modified:** 7 source/test files (1046 insertions / 15 deletions in web/src + web/tests), plus web/build regenerated (23 files changed in the committed bundle)

## Accomplishments

- **Cycle classes copy wire fields onto element data, never compute them.** `file-graph-transform.ts` adds a `graph-cycle` class (plus a per-cycle `graph-cycle-N` discriminator) to a file node whose `cycleId` is non-zero, to a collapsed directory whose files union to a non-empty `cycleIds` set (05-08's own union, extended with styling this plan), and to an edge whose `inCycle` flag is true. A dedicated test proves the negative: a fixture whose edges close a loop in the adjacency but whose wire cycle fields are unset produces zero cycle classes.
- **Cycle members are visually distinguished on two non-colour channels plus colour.** `graph-style.ts`'s `node.graph-cycle` / `edge.graph-cycle` selectors add a dashed border/line and a distinct arrow shape (`diamond`) alongside a literal hex colour chosen to match the app's `--destructive` design token (cytoscape does not resolve CSS custom properties in style values — confirmed against its own docs before choosing this approach).
- **A stated cycle count and a wrapping focus control remove the hunting GRF-04's wording rules out.** The route renders `graphState.response.cycleCount` as plain text with an explicit no-cycles sentence at zero, and — only when the count is positive — a control that groups cycle members by the TYPED numeric `cycleId`/`cycleIds` already on element data (never a class list, proven by a fixture with discriminator classes stripped) and steps through every cycle, wrapping after a full lap.
- **`GraphCanvas.svelte`'s seam grows by exactly one prop and two events, in its existing vocabulary.** `focusNodeIds` fits the viewport to exactly the named ids via `getElementById`+`union` (never a composed selector string, since a repository-relative path can contain characters a CSS-like selector would mis-parse); `onEdgeSelected`/`onBackgroundTapped` mirror `onNodeSelected`'s data-out contract. No renderer instance or viewport API crosses back to the route.
- **An edge's per-kind counts are readable through the ONE shared table.** `edge-kind-columns.ts` defines a `{kind,count}` `ColumnDef` set mirroring `workbench/callers-columns.ts`'s convention; the route builds rows by iterating the SELECTED edge's own sparse `kindCounts` map entries directly, so a kind the server never reported produces no row — verified live: `internal/cli → internal/watch`, total 2, rows `calls:1` + `instantiates:1` summing to 2 exactly.
- **The committed bundle was rebuilt and every supply-chain gate re-verified green:** `task web:drift` (108 source files / 32 output files, both digests match), `task web:audit` (0 advisories), `task web:lockfile` (242/242 integrity-bearing resolutions), `task web:deps:strict` (strictDepBuilds true, 0 denials).
- **Live-browser verification against a freshly indexed google/guava clone** (shallow, current HEAD — not the pinned SHA, since this was a usability check of the NEW cycle-focus feature, not a re-run of GRF-01's own locked measurement) confirmed the control stays usable at scale: 163 cycles, ~1s per activation including layout settle, a legible dashed-red highlighted group. It also surfaced a new, pre-existing (present on first paint, before any interaction) cytoscape console warning at that density — recorded honestly below and in `WINDOWS.md` entry 28, not silently worked around.

## Task Commits

Each TDD task carries a RED-then-GREEN pair; Task 3 also carries a dedicated bundle-rebuild commit:

1. **Task 1 RED:** `fd849830` — `test(05-05): add failing tests for cycle classes, cycle styling, and route cycle summary/focus control` (11 of 25 new assertions failed; two negative-control "no cycle classes" cases correctly passed against the as-yet-unmodified transform)
2. **Task 1 GREEN:** `d1c81f36` — `feat(05-05): copy wire cycle fields onto element classes and style cycle members and edges`
3. **Task 2 GREEN:** `59779615` — `feat(05-05): state the cycle count and add a next-cycle focus control` (Task 2's tests were authored together with Task 1's in the same RED commit above — see Deviations)
4. **Task 3 RED:** `d37ed2c8` — `test(05-05): add failing tests for readable per-kind edge counts through the shared table` (6 of 6 new assertions failed)
5. **Task 3 GREEN:** `3c97c5c7` — `feat(05-05): render per-kind edge counts through the shared table, and an edge/background tap seam`
6. **Task 3 bundle rebuild:** `c12ebfc2` — `feat(05-05): rebuild the committed bundle for the cycle and edge-detail source changes`
7. **Live-verification finding:** `ce078cf7` — `docs(05-05): record a guava-scale cytoscape layout warning found during live verification`

**Plan metadata:** (this commit) — completes this SUMMARY, `STATE.md`, `ROADMAP.md`, `REQUIREMENTS.md`.

_Note: TDD RED/GREEN commits per this plan's `type: tdd`; see Deviations for why Task 1 and Task 2's RED tests share one commit._

## Files Created/Modified

- `web/src/lib/components/graph/file-graph-transform.ts` — `graph-cycle`/`graph-cycle-N` classes copied from wire fields onto file nodes, collapsed directories, and in-cycle edges
- `web/src/lib/components/graph/graph-style.ts` — `node.graph-cycle` / `edge.graph-cycle` selectors
- `web/src/lib/components/graph/GraphCanvas.svelte` — `focusNodeIds` prop + `focus()`; `onEdgeSelected`/`onBackgroundTapped` events + their `cy.on('tap', ...)` wiring
- `web/src/lib/components/graph/edge-kind-columns.ts` (new) — the `{kind,count}` `ColumnDef` set
- `web/src/routes/graph/+page.svelte` — cycle summary/focus-control region, edge detail region with the shared `DataTable`
- `web/tests/graph-cycles.test.ts` (new, 25 tests) — Task 1 + Task 2 assertions
- `web/tests/graph-edge-detail.test.ts` (new, 6 tests) — Task 3 assertions
- `web/build` — rebuilt; source-file count 107→108 (`edge-kind-columns.ts` entered the hashed set; the two new `*.test.ts` files stay outside it, matching 05-08's precedent)
- `.planning/WINDOWS.md` — entry 28, the guava-scale cytoscape warning found during live verification
- `.planning/REQUIREMENTS.md` — GRF-04 marked complete (GRF-02 stays open: shared with 05-03/05-07/05-08 per the shared-ID gate, and 05-07 has not yet run)

## Decisions Made

See `key-decisions` in the frontmatter above for the four decisions made during execution (collapsed-directory cycle classes, dual-shape cycle grouping, the defensive edge-tap guard, and keeping `edge-kind-columns.ts` independent of `health/CountTable.svelte`'s `CountRow`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `GraphCanvas.svelte`'s new edge-tap handler crashed an unrelated, already-passing test file**
- **Found during:** Task 3's full-suite verification run (not the scoped `graph-edge-detail` filter — the scoped run alone would have missed this)
- **Issue:** `web/tests/graph-expansion.test.ts`'s own minimal cytoscape test double (from an earlier task) registers all `cy.on('tap', ...)` handlers under one listener key regardless of selector, so its `simulateTap(nodeId)` helper invoked the new `onEdgeSelected` handler with a node-shaped target lacking `source()`/`target()`, throwing `TypeError: target.source is not a function` and failing 6 previously-green tests in that file.
- **Fix:** Added a guard in the edge-tap handler (`typeof target.source !== 'function' || typeof target.target !== 'function'`) with a comment explaining it exists for an earlier task's test double, not for production — real cytoscape's own selector scoping never hands this handler a non-edge target.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`
- **Verification:** Full suite 391/391 after the fix; `graph-expansion.test.ts` untouched and green.
- **Committed in:** `3c97c5c7` (Task 3 GREEN commit)

**2. [Rule 1 - Bug] A test-authoring bug in Task 1's own marked/unmarked assertion**
- **Found during:** Task 1's first GREEN run
- **Issue:** The "two files sharing a cycle id" test expected the unmarked-node set to be exactly `['a/z.go']`, but with directory `a` expanded (as the fixture requires to render the individual files), `rollupToElements` also emits the expanded-directory COMPOUND PARENT node itself (`id: 'a'`) — correctly uncycled, but the test's own expectation omitted it.
- **Fix:** Corrected the expected unmarked set to `['a', 'a/z.go']`, with a comment explaining why the parent compound appears.
- **Files modified:** `web/tests/graph-cycles.test.ts`
- **Verification:** Test passes; the transform's implementation was not the bug.
- **Committed in:** `d1c81f36` (folded into the Task 1 GREEN commit, since the fix landed before that commit)

**3. [Rule 1 - Bug] A test-authoring bug in Task 2's exact-membership assertion**
- **Found during:** Task 2's first GREEN run
- **Issue:** The "focus control passes exactly the members of one cycle" test assumed file-path ids (`'a/x.go'`, `'a/y.go'`), but the route's default `expandedDirs` is empty, so the collapsed default renders one node PER DIRECTORY — the actual fitted ids were the collapsed directory ids (`'a'`, `'b'`, `'c'`), not file paths.
- **Fix:** Corrected the expected possible-groups list to `[['a'], ['b'], ['c']]`, with a comment explaining the collapsed-default behavior this test exercises.
- **Files modified:** `web/tests/graph-cycles.test.ts`
- **Verification:** Test passes; the route's grouping implementation was not the bug — this is the SAME property review M-8 asked to be proven (grouping reads typed cycle ids correctly regardless of whether a member is currently a file or a collapsed directory).
- **Committed in:** `59779615` (Task 2 GREEN commit)

### Process deviation (not a Rule 1-4 fix, a commit-granularity choice)

**4. Task 1 and Task 2's RED tests share ONE commit instead of two.**
- Both tasks' `<behavior>` blocks extend the SAME new file, `web/tests/graph-cycles.test.ts`. All 25 assertions (Task 1's element-data/style-sheet tests and Task 2's route-level cycle-count/focus tests) were authored together and observed RED together — genuinely, before any implementation existed (11 of 25 failed; the RED log is quoted in the Task 1 commit message) — before either task's implementation began.
- Rather than artificially splitting one already-written, already-RED-verified file across two commits via `git add -p`, the RED evidence was committed as one `test(05-05)` commit, followed by two SEPARATE `feat(05-05)` GREEN commits (one per task's actual implementation files). This still satisfies the TDD gate sequence (a `test(05-05)` commit precedes every `feat(05-05)` commit) and keeps the RED evidence honest and verbatim; it does not split as cleanly into "one RED commit per task" as the plan's task boundaries suggest.
- **Impact:** None on correctness or verifiability — `git log --grep` still finds one `test(05-05)` commit before the `feat(05-05)` commits, and both tasks' RED failure counts are recorded verbatim in the commit messages.

---

**Total deviations:** 3 auto-fixed (2 test-authoring bugs, 1 real cross-test-file bug) + 1 process/commit-granularity choice.
**Impact on plan:** All three auto-fixes were necessary corrections found during the task's own verification loop (two in this plan's own new tests, one in an unrelated pre-existing test file that a new production code path incidentally exercised). No scope creep; the commit-granularity choice is documented for transparency, not hidden.

## Issues Encountered

- **A live-browser test-suite flake (390/391 on one `task web:test` run, 391/391 on 13 subsequent runs across two invocation styles).** Investigated: could not reproduce after the one occurrence; the official gate run recorded for this plan is the clean 391/391 result. Not chased further given the reproduction rate (1 in ~14 attempts) and this project's own precedent of documenting rather than indefinitely chasing rare, non-reproducing test-infrastructure flakes (see `internal/daemon`'s watchdog test, `data-table-render-cost.test.ts`'s own "flake generator" comment). If this recurs, it is worth a dedicated investigation into `waitFor` timeout margins under mixed real-headless-cytoscape / fake-cytoscape worker parallelism.
- **A previously undocumented cytoscape console warning at google/guava scale.** Recorded as `WINDOWS.md` entry 28 (see Accomplishments and Deviations). Not a defect in this plan's own code — present on first paint before any interaction, self-labeled "expected behaviour" by cytoscape, and not reproduced against this repository's own smaller index. Recorded, not investigated further; out of this plan's scope.
- **Live guava-scale verification used a shallow clone at current HEAD, not the pinned SHA `94f39958baf7ad51ddf9c70e406ed6b188194daa`.** This was an exploratory usability check of the NEW cycle-focus/edge-detail features at scale, not a re-run of GRF-01's own locked measurement (which 05-08 already re-verified PASS against the exact pinned commit). The collapsed node/edge counts observed (135/830) are close to but not identical to the pinned corpus's 134/819 — expected, since HEAD has moved since the pin. No threshold artifact was touched.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **GRF-04 is complete.** GRF-02 remains open pending 05-07 (shared-ID gate: 05-03, 05-05, 05-07, 05-08 all declare it; 05-07 has not yet produced a SUMMARY).
- **05-06 and 05-07 gain a working precedent for reading typed element data rather than class strings** — both plans' own `<precondition>` blocks and wave numbers were already re-pointed at 05-08's remedy before this plan started; this plan adds no new blocker for either.
- **05-07 still owns the collapse-by-click hit-testing defect** (`WINDOWS.md` entry 27, unchanged by this plan) and now also inherits `WINDOWS.md` entry 28 as background context, though entry 28 is not assigned to any specific plan — it is a general finding about rendering density at guava scale.
- **The edge-selected/background-tapped seam surface (`GraphCanvas.svelte`) is now established** for any future plan needing to build on selection state in this view.

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-31*

## Self-Check: PASSED
