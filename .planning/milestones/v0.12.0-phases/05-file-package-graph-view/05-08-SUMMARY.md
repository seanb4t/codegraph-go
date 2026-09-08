---
phase: 05-file-package-graph-view
plan: 08
subsystem: ui
tags: [cytoscape, cytoscape-elk, playwright, rollup, expansion, gate, verdict, checkpoint]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-01's locked corpora/graph-render-threshold.json; 05-02's FileGraph rpc and measured response size (3,713,528 bytes); 05-03's GraphCanvas.svelte measurement seam; 05-04's GRF-01 FAIL verdict and the maintainer's `halt-collapse-default` remedy selection (05-04-SUMMARY.md)"
provides:
  - "web/src/lib/components/graph/file-graph-transform.ts's rollupToElements(response, expandedDirs) — the single pure function that builds both the collapsed default and every progressive expansion, with plannedNodeCount and EXPANSION_NODE_CEILING (1500)"
  - "The collapsed-by-default graph route (web/src/routes/graph/+page.svelte) with in-place directory expand/collapse, an over-ceiling refusal, and a rendered-geometry seam (window.__codegraphFileGraphGeometry) alongside the existing metrics seam"
  - "corpora/graph-render-observations-collapsed.json — GRF-01's re-measure at collapsed scale, verdict PASS, computed by the unmodified comparator"
  - "web/scripts/graph-expand-check.mjs — a real-browser interaction check, plus its honest diagnostic record at corpora/graph-expand-check.json documenting a genuine, unresolved collapse-by-click hit-testing defect"
  - "The maintainer's Task 4 answer, `release-collapsed`, recorded here and in .planning/STATE.md's decision log — the precondition 05-05, 05-06 and 05-07 already carry"
affects: [05-05, 05-06, 05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 31253
  tasks: 4
  commits: 7

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One pure rollup, two views: rollupToElements(response, expandedDirs) is the sole element builder for both the collapsed default (empty set) and every expansion (non-empty set) — the whole-graph builder it retires is gone from the module's exported set, checked by SET equality in tests, not substring search."
    - "Construction/application split in the renderer: the effect that constructs the cytoscape instance depends only on the container; createFileGraphRenderer(cy, ...) exposes start() (first layout) and replace(elements) (batched remove+add on the LIVE instance, re-running the same layered layout) so no expansion ever tears down and rebuilds the instance, and the metrics seam publishes once, ever."
    - "A second, geometry seam (window.__codegraphFileGraphGeometry) alongside the existing metrics seam: canvas rendering leaves no per-node DOM element, so this typed array (id, rendered position, expandable) is what lets a real browser — and a Playwright driver — click a specific node."
    - "Edge aggregation and cycle-id union are one pass over the response's edges/nodes with map lookups, never a nested scan or a client-side connectivity computation — summed per-kind counts, summed totals, an OR'd in-cycle flag, an aggregatedFrom count, and a sorted-distinct cycleIds list on each collapsed directory."

key-files:
  created:
    - web/tests/graph-collapse.test.ts
    - web/tests/graph-expansion.test.ts
    - web/scripts/graph-expand-check.mjs
    - corpora/graph-render-observations-collapsed.json
    - corpora/graph-expand-check.json
  modified:
    - web/src/lib/components/graph/file-graph-transform.ts
    - web/src/lib/components/graph/graph-style.ts
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/routes/graph/+page.svelte
    - web/src/app.d.ts
    - web/scripts/graph-verdict.mjs
    - web/tests/file-graph-transform.test.ts
    - web/tests/graph-tracer.test.ts
    - web/tests/graph-verdict.test.ts
    - web/build
    - .planning/WINDOWS.md
    - .planning/STATE.md

key-decisions:
  - "Task 4 maintainer decision: `release-collapsed` (2026-08-30) — the collapsed default cleared all four unchanged locked bars with wide margin; 05-05, 05-06 and 05-07 resume."
  - "The collapse-by-click hit-testing defect (WINDOWS.md 27) does not block the gate — assigned to 05-07 as an explicit collapse-affordance requirement, not reopened here."

patterns-established:
  - "A checkpoint:decision gate's answer, once given, is recorded verbatim in BOTH the plan's own SUMMARY and STATE.md's decision log — never paraphrased, so a later reader audits the decision rather than infers it."

requirements-completed: [GRF-01, GRF-02]

coverage:
  - id: D1
    description: "The graph view's default first paint on the pinned corpus is the collapsed directory view: one node per directory that contains files, zero file nodes."
    requirement: GRF-01
    verification:
      - kind: unit
        ref: "web/tests/graph-collapse.test.ts (27 tests)"
        status: pass
      - kind: e2e
        ref: "corpora/graph-render-observations-collapsed.json bindingObservation (live browser, guava corpus, nodeCount=134 edgeCount=819)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Selecting a directory reveals its files in place and selecting it again closes it, with zero additional requests and exactly one renderer instance throughout."
    requirement: GRF-02
    verification:
      - kind: unit
        ref: "web/tests/graph-expansion.test.ts (13 tests) — replace()-not-reconstruct, publish-once metrics"
        status: pass
      - kind: automated_ui
        ref: "web/scripts/graph-expand-check.mjs live run, corpora/graph-expand-check.json"
        status: fail
    human_judgment: true
    rationale: "The unit-level toggle logic is fully proven (48 headless-cytoscape tests across graph-collapse + graph-expansion). The live real-mouse RE-COLLAPSE gesture is a proven, unresolved defect (corpora/graph-expand-check.json success:false; WINDOWS.md 27) — expansion works reliably, collapse does not, root-caused to a ~1.1px clickable margin under cytoscape-elk. Maintainer Task 4 answer explicitly rules this non-blocking for GRF-01 and assigns the fix to 05-07; a human must read that record, not a test."
  - id: D3
    description: "An expansion past the shipped ceiling (EXPANSION_NODE_CEILING=1500) is refused in readable copy and leaves the graph unchanged."
    verification:
      - kind: unit
        ref: "web/tests/graph-expansion.test.ts — over-ceiling refusal, within-ceiling application"
        status: pass
    human_judgment: false
  - id: D4
    description: "The rollup carries every server-computed answer through to the directory level: summed per-kind counts, aggregatedFrom, in-cycle flag, and cycleIds on collapsed directories."
    verification:
      - kind: unit
        ref: "web/tests/graph-collapse.test.ts — edge aggregation, cycle-flag union, cycleIds union (both directions each)"
        status: pass
    human_judgment: false
  - id: D5
    description: "GRF-01's re-measure at the collapsed scale, against the SAME unmodified locked bars, verdict PASS, with the locked artifact, the recorded FAIL and the pinned measurement wrapper all provably untouched."
    requirement: GRF-01
    verification:
      - kind: e2e
        ref: "corpora/graph-render-observations-collapsed.json (generatedBy=web/scripts/graph-verdict.mjs, verdict=PASS)"
        status: pass
      - kind: other
        ref: "git log --oneline -- corpora/graph-render-threshold.json | wc -l (=1); git diff --exit-code over threshold/observations.json/graph-measure.mjs (clean)"
        status: pass
    human_judgment: true
    rationale: "GRF-01's own text requires a human to read the verdict before release — this checkpoint IS the human read, recorded at Task 4 below. Never auto-approved even under auto-advance (gate=blocking-human)."

# Metrics
duration: "active execution ~2h17m (2026-08-30 19:07-21:30 ET) across Tasks 1-3; Task 4 blocking-human checkpoint held the plan open until the maintainer's 2026-08-31 answer, recorded below"
completed: 2026-08-31
status: complete
---

# Phase 5 Plan 8: Collapsed-Default Graph View & GRF-01 Re-Measure Summary

**Collapsed-directory-by-default file graph with progressive in-place expansion, one pure rollup function replacing the retired whole-graph builder, and a live GRF-01 re-measure that PASSED all four unchanged locked bars at the pinned google/guava corpus — maintainer answer `release-collapsed` releases 05-05, 05-06 and 05-07.**

## Performance

- **Duration:** Tasks 1-3 executed 2026-08-30 19:07-21:30 ET (~2h17m active). Task 4 (`checkpoint:decision`, `gate="blocking-human"`) then held the plan open — never auto-approved under this project's `auto_advance: true` — until the maintainer answered on 2026-08-31.
- **Started:** 2026-08-30T23:07:11Z (first RED commit, `4003d58b`)
- **Completed:** 2026-08-31T18:18:42Z (this close-out)
- **Tasks:** 4 (RED/GREEN TDD for the rollup; RED/GREEN TDD for the renderer/route; the live re-measure; the maintainer checkpoint)
- **Commits:** 7 (6 implementation/measurement + this metadata commit)

## Accomplishments

- Retired the whole-graph file-level element builder and replaced it with `rollupToElements(response, expandedDirs)` — one pure function of the decoded wire response and a typed `Set<string>` of expanded directory paths, producing both the collapsed default (empty set: 134 directory nodes, zero file nodes, matching the locked threshold's pre-recorded `recordedNonBinding.collapsedView` figures exactly) and every progressive expansion.
- Shipped a node ceiling (`EXPANSION_NODE_CEILING = 1500`, strictly between the 714-node scale measured interactive and the 3,233-node scale that was not) with a `plannedNodeCount` predicate sharing the same internals as the builder, so the two can never disagree; an over-ceiling expansion is refused in readable copy with the graph left exactly as it was.
- Separated cytoscape-instance construction from element application in `GraphCanvas.svelte` — `createFileGraphRenderer(cy, ...)` exposes `start()` and `replace(elements)`, a batched remove+add on the LIVE instance. No expansion or collapse ever tears down and rebuilds the renderer, and the existing metrics seam is guarded to publish once, ever. A second, new rendered-geometry seam (`window.__codegraphFileGraphGeometry`) publishes after every settle, carrying id/position/expandable/fileCount per node — the seam that makes a real browser click possible, since canvas rendering leaves no per-node DOM element.
- Re-measured GRF-01 live at the pinned google/guava corpus under the collapsed default, against the SAME unmodified locked bars: **PASS on all four metrics**, with the collapsed node/edge counts (134/819) independently re-derived live and equal to the threshold's own pre-recorded figures.
- Wrote `web/scripts/graph-expand-check.mjs`, a real-browser interaction check driving trusted (never dispatched) mouse input, and used it to find and document — not silently work around — a genuine collapse-by-click hit-testing defect, now WINDOWS.md entry 27 and assigned to 05-07.
- Maintainer Task 4 decision recorded: **`release-collapsed`**. 05-05, 05-06 and 05-07's preconditions and wave numbers (already re-pointed at this plan's artifact by 05-05/05-06/05-07's own commit `56190931`) are confirmed correct and unblocked.

## Task Commits

Each task was committed atomically (Tasks 1 and 2 each carry a RED then GREEN pair per this plan's `type: tdd`):

1. **Task 1 RED:** `4003d58b` — `test(05-08): add failing tests for the collapsed-default rollup and verdict output-path flags` (35 failures: 27 new `graph-collapse.test.ts` + 7 new `graph-verdict.test.ts` + 1 pre-existing count-drift assertion, all on not-yet-implemented exports).
2. **Task 1 GREEN:** `5b5922e6` — `feat(05-08): one pure rollup for the collapsed default and every expansion, a shipped node ceiling, and a non-destructive comparator write path` (graph-collapse 27/27, graph-verdict 25/25, file-graph-transform 8/8, graph-tracer 8/8; `pnpm check` 0/0).
3. **Task 2 RED (largely already-GREEN by construction, documented honestly):** `409049ab` — `test(05-08): add tests for the collapsed default first paint, progressive expansion, and the renderer's replace/publish-once seam` (13 new tests in `graph-expansion.test.ts`; 12/13 passed on first write against Task 1's already-shipped route code, one arithmetic error in the test itself fixed, not the implementation).
4. **Task 2 GREEN:** `45bec654` — `feat(05-08): expand and collapse a directory in place, refuse over-ceiling expansions, and publish a rendered-geometry seam` (graph-expansion 13/13, graph-tracer 8/8 unmodified, `pnpm check` 0/0).
5. **Task 2 fix (Rule 1 - Bug, scoped to this task's own acceptance check):** `675eebf4` — `fix(05-08): remove leaked planning vocabulary from doc comments in file-graph-transform.ts and graph-style.ts` (10 leaked planning-identifier matches in doc comments, rewritten to 0, positive-controlled).
6. **Task 3:** `92329657` — `feat(05-08): re-measure the collapsed default at the pinned corpus — PASS on every locked bar` (full frontend suite 371/371, up from 324 before this plan; `pnpm check` 0/0; `task web:drift` both halves match, 107 source files unchanged; backend `task test:unit` green, untouched).
7. **Task 4 + plan metadata (this commit):** records the maintainer's `release-collapsed` answer, commits the expand-check diagnostic evidence and the pending `.planning/WINDOWS.md` entry, and closes out this SUMMARY.

_Note: TDD tasks each carry a RED then GREEN commit per the plan's `type: tdd`; Task 2 additionally carries a scoped Rule-1 fix commit._

## Task 3: Re-Measure Command Outputs (verbatim, as recorded by the executing session)

Per `92329657`'s own commit message (the authoritative record of what ran, since this continuation agent did not re-execute Task 3 — see "Continuity note" below):

```
GREEN confirmed: full frontend suite 371/371 (up from 324 before this
plan), `pnpm check` 0/0, `task web:drift` both halves match (107
source files, unchanged — the new script is outside the hashed set).
Backend `task test:unit` green, untouched. Threshold, recorded FAIL,
measurement wrapper, package.json and pnpm-lock.yaml all clean diff;
threshold still exactly 1 commit.
```

**Comparator verdict, read from `corpora/graph-render-observations-collapsed.json`:**

| Metric | Measured | Bar | Comparison | Verdict |
|---|---|---|---|---|
| `timeToInteractiveMs` | 1179.2000000029802 ms (median of 3 cold reloads: 1179.2 / 1365.7 / 1033.3) | 5000 | `<=` | PASS |
| `panZoomFrameTimeMs` | 8.300000000000182 ms | 33.3 | `<=` | PASS |
| `panZoomFrameTimeP95Ms` | 9 ms | 100 | `<=` | PASS |
| `fileGraphResponseBytes` | 3,713,528 (cited verbatim from `05-02-SUMMARY.md` `TestFileGraphResponseSizeGuava`, unaffected by which elements render) | 16,777,216 | `<=` | PASS |

Overall `verdict: "PASS"`. `generatedBy: "web/scripts/graph-verdict.mjs"` (the unmodified comparator). `browserIdentity: {name: "chromium", version: "151.0.7922.34"}`. `layoutDurationMs: 714.5`, `frameSampleCount: 463` (well above the recorded floor of 60).

**Derived-versus-rendered collapsed-count comparison** (the acceptance criterion that any mismatch be written down as a finding, never reconciled):

| Source | Nodes | Edges |
|---|---|---|
| `corpora/graph-render-threshold.json`'s pre-measurement `recordedNonBinding.collapsedView` (recorded before any measurement existed) | 134 | 819 |
| This re-measure's live-rendered `bindingObservation.nodeCount`/`edgeCount` | 134 | 819 |
| This re-measure's independently live-derived `bindingObservation.collapsedView` (from the shipped rollup, a second browser session) | 134 | 819 |
| The expanded file-level view that originally FAILED (`05-04`, unchanged) | 3,233 | 21,554 |

**No mismatch.** All three collapsed-scale figures agree exactly — the pre-recorded lock, the rendered paint, and the independently re-derived count from the shipped rollup all read 134/819.

## Expansion Check Record (verbatim, `corpora/graph-expand-check.json`)

```json
{
  "success": false,
  "collapsedNodes": 134,
  "liveGeometryLengthAtFailure": 135,
  "candidateAttempts": 5,
  "candidatesTried": [
    "guava-gwt/src-super/com/google/common/collect/super/com/google/common/collect",
    "guava-testlib/src/com/google/common/testing",
    "guava-gwt/test/com/google/common",
    "android/guava-tests/test/com/google/common/cache",
    "android/guava-tests/benchmark/com/google/common/eventbus"
  ],
  "lastError": "clickUntilCondition: condition never became true across 17 offsets around (684.3360805896725, 574.7729262395179)",
  "metricsAtFirstPaint": {
    "timeToInteractiveMs": 1266.4000000059605,
    "layoutDurationMs": 721.5,
    "nodeCount": 134,
    "edgeCount": 819
  },
  "pageErrorCount": 0,
  "pageErrors": [],
  "browserIdentity": {
    "name": "chromium",
    "version": "151.0.7922.34"
  }
}
```

**Page-error census: 0 errors, empty list** — this run did not hit the pre-existing, unrelated, non-fatal `cytoscape-elk` adapter error (WINDOWS.md 26).

**Honest finding, not silently worked around:** across 5 candidate directories (chosen for on-screen isolation) x 17 real-mouse jitter offsets each (85 total real-mouse attempts, per `92329657`'s commit message), the check reliably EXPANDED a directory via a real, trusted mouse click at its geometry-seam-reported position (confirmed repeatedly across many directories and corpora in prior sessions this task, per the same commit: `134->165`, `134->241`, `134->210`). Reliably **RE-COLLAPSING** the same compound via a second real click could not be demonstrated within this task's engineering budget. Root cause measured, not assumed: `cytoscape-elk`'s layered algorithm leaves under ~2px (measured ~1.1px via `renderedBoundingBox`) of clickable margin around a compound directory's own border once it has children. Three mitigations were tried and each measured ineffective — cytoscape's own CSS `padding` (confirmed cosmetic-only under this external layout extension by reading `cytoscape-elk` source), ELK's graph-level `elk.padding`, and ELK's per-node `nodeLayoutOptions` padding hook (the officially correct per-compound mechanism). `text-events: 'yes'` was added to directory nodes (a real, defensible improvement — the collapsed-view label band is now hit-testable) but did not resolve the collapse case either.

**Consequence for Task 3's literal `<verify>` gate:** the plan's automated check for the expand-check record asserts `r.expandedNodes > r.collapsedNodes` and `r.restoredNodes === r.collapsedNodes` and `r.metricsRepublished === false`. The failure-path record above (`success: false`) does not carry `expandedNodes`/`restoredNodes`/`metricsRepublished` fields — it carries the honest diagnostic shape (`candidateAttempts`, `candidatesTried`, `lastError`) instead. Re-running that literal `node -e` check against this record would throw. This is recorded here rather than concealed: the toggle LOGIC is fully proven at the unit level (48 headless-cytoscape tests across `graph-collapse.test.ts` and `graph-expansion.test.ts` — expand, collapse, publish-once, geometry, all pass), and only the real-mouse collapse GESTURE is unreachable at this corpus's rendered density. This is exactly the gap the Task 4 checkpoint below was presented with, unreconciled, per the plan's own instruction not to adjust anything to make a mismatch agree.

**Broken-windows ledger:** `.planning/WINDOWS.md` entry 27 (`kind: unrun-verify`, `phase: 05`, `web/src/lib/components/graph/GraphCanvas.svelte`, `status: open`) records this defect in full, including the owner assignment to 05-07. That edit was already staged by the prior session before the Task 4 checkpoint; committed here alongside this SUMMARY.

## Task 4: Maintainer Decision (verbatim)

**MAINTAINER DECISION, 2026-08-30: `release-collapsed`.**

**The gate is released.** GRF-01's re-measure at collapsed scale cleared all four locked bars with wide margin, and the orchestrator independently verified both the verdict and the controls that make it trustworthy:

| Metric | Measured | Bar | Margin |
|---|---|---|---|
| `timeToInteractiveMs` | 1,179.2 ms | ≤ 5000 | 4.2× |
| `panZoomFrameTimeMs` | 8.3 ms | ≤ 33.3 | 4.0× |
| `panZoomFrameTimeP95Ms` | 9 ms | ≤ 100 | 11× |
| `fileGraphResponseBytes` | 3,713,528 | ≤ 16,777,216 | 4.5× |

Controls independently re-verified by the orchestrator: `corpora/graph-render-threshold.json` has exactly **1 commit** with a clean diff; `web/scripts/graph-measure.mjs` unmodified, so the re-measure's pan/zoom gesture is byte-identical to the FAIL's; `corpora/graph-render-observations.json` byte-identical and still reading FAIL; rendered **134/819** exactly equal to the `approxNodes`/`approxEdges` pre-recorded in the locked artifact before any measurement existed. **No threshold value was widened, lowered or re-scoped.**

**The collapse-by-click defect is acknowledged and does NOT block the gate.** GRF-01 asks whether the qualifying renderer stays interactive at the largest corpus; it does, decisively. The defect is a hit-testing problem, not a scale failure, so it does not reopen D-02, D-04 or D-07.

**It is assigned to plan 05-07**, recorded as `WINDOWS.md` entry **27**. The reasoning: 05-07's verified three-click contract is expand → collapse-to-zero → re-expand, so the identical ~1.1px-margin defect will block it at *file* level exactly as it blocks directories here. 05-07 must ship an **explicit collapse affordance** — a caret/disclosure control, or a hit-testable label band — rather than depending on clicking a sub-2px margin. The toggle logic itself is already correct and covered by 48 headless-cytoscape tests; only the real-mouse gesture is unreachable.

The honest presentation of that failure — a diagnostic record written on every run, `"success": false` reported plainly rather than dressed up, and three mitigations each measured ineffective rather than assumed to work — is exactly the right handling and is noted approvingly.

**Consequence for gated plans:** per this plan's own "Changes this plan forces on 05-05, 05-06 and 05-07" section, all three plans' `<precondition>` blocks and wave numbers must be re-pointed at `corpora/graph-render-observations-collapsed.json` recording PASS and at this SUMMARY's `release-collapsed` answer, before any of them executes. **This was already done** — commit `56190931` (`docs(05): re-point gated plans at 05-08's re-measure — GRF-01 remedy halt-collapse-default`), predating this checkpoint's answer, already carries:
- `05-05-PLAN.md`: `wave: 6`, precondition reading `corpora/graph-render-observations-collapsed.json records an overall verdict of PASS, and 05-08-SUMMARY.md records the answer \`release-collapsed\`` (verified present, unmodified by this task).
- `05-06-PLAN.md`: `wave: 7`, identical precondition text (verified present, unmodified by this task).
- `05-07-PLAN.md`: `wave: 8`, identical precondition text (verified present, unmodified by this task).

Both clauses of all three preconditions are now true: the observation records PASS, and this SUMMARY records `release-collapsed`. **05-05, 05-06 and 05-07 are unblocked.**

**Control outputs (re-asserted at this close-out, verbatim):**

```
$ git log --oneline -- corpora/graph-render-threshold.json | wc -l
       1
```

```
$ git diff --exit-code corpora/graph-render-threshold.json web/scripts/graph-measure.mjs corpora/graph-render-observations.json
(exit 0, no output — clean)
```

The recorded FAIL from the earlier run (`corpora/graph-render-observations.json`) still exists, byte-identical, confirmed by the clean diff above.

## Continuity Note

This SUMMARY is written by a continuation agent that resumed at Task 4 only — Tasks 1-3 were executed and committed in a prior session (commits `4003d58b` through `92329657`), which then halted at the `checkpoint:decision`/`gate="blocking-human"` per this plan's own instruction that the gate is never auto-approved. This continuation agent did not re-run Task 1-3's tests or the live measurement; Task 3's command outputs and test counts above are reproduced verbatim from that session's own commit messages (the authoritative record of what ran), not re-executed. The controls required by Task 4's acceptance criteria (threshold commit count, clean diffs) WERE re-run independently by this continuation agent, above, rather than trusted from the prior session's report.

## Files Created/Modified

- `web/src/lib/components/graph/file-graph-transform.ts` — the single element builder (`rollupToElements`), `plannedNodeCount`, `EXPANSION_NODE_CEILING`.
- `web/src/lib/components/graph/graph-style.ts` — the collapsed-directory selector.
- `web/src/lib/components/graph/GraphCanvas.svelte` — construction/application split, `createFileGraphRenderer`, the geometry seam, `text-events: 'yes'`, `replace()` no longer re-fitting the viewport.
- `web/src/routes/graph/+page.svelte` — expansion state (`Set<string>`), tap-to-expand/collapse, ceiling refusal copy.
- `web/src/app.d.ts` — the geometry global declaration, `fileCount` on geometry entries.
- `web/scripts/graph-verdict.mjs` — `--out`/`--note` flags, `resolveOutputPath`, exported `writeVerdict`.
- `web/scripts/graph-expand-check.mjs` (new) — the real-browser interaction check.
- `web/tests/graph-collapse.test.ts` (new, 27 tests), `web/tests/graph-expansion.test.ts` (new, 13 tests).
- `web/tests/file-graph-transform.test.ts`, `web/tests/graph-tracer.test.ts`, `web/tests/graph-verdict.test.ts` — restated against the rollup / extended for the new flags.
- `web/build` — rebuilt and re-committed; source-file count unchanged (107, new scripts/tests outside the hashed set).
- `corpora/graph-render-observations-collapsed.json` (new) — the PASS re-measure.
- `corpora/graph-expand-check.json` (new, this commit) — the honest expand-check diagnostic evidence, committed verbatim from the prior session's run (`/tmp/graph-expand-check.json`), unedited.
- `.planning/WINDOWS.md` — entry 27, the collapse-by-click defect, committed here (was staged by the prior session, uncommitted at the checkpoint).
- `.planning/STATE.md` — the `release-collapsed` decision recorded in the Decisions section; the superseded `05-04` blocker ("05-05/05-06/05-07 stay BLOCKED until the collapse-default work...") resolved/removed from Blockers/Concerns.

## Decisions Made

- **Task 4 answered `release-collapsed`** — see "Task 4: Maintainer Decision" above for the verbatim answer and reasoning.
- **The collapse-by-click defect does not reopen the gate** — assigned to 05-07 as an explicit collapse-affordance requirement, per the maintainer's own reasoning above.

## Deviations from Plan

### Auto-fixed Issues (from the prior session, Tasks 1-3, reproduced here for completeness)

**1. [Rule 1 - Bug] Removed leaked planning vocabulary from doc comments**
- **Found during:** Task 2 (its own acceptance check: `rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0' web/src/routes/graph/ web/src/lib/components/graph/`)
- **Issue:** 10 matches, all inside doc comments introduced earlier in this task, leaking plan/requirement identifiers into source comments.
- **Fix:** Rewrote every flagged comment to convey the same rationale without the banned substrings.
- **Files modified:** `web/src/lib/components/graph/file-graph-transform.ts`, `web/src/lib/components/graph/graph-style.ts`
- **Verification:** Same grep now reports 0, positive-controlled by a substring known present.
- **Committed in:** `675eebf4`

**2. [Rule 1 - Bug] `replace()` re-fitting the whole viewport on every expansion**
- **Found during:** Task 3's live-browser investigation of the collapse defect.
- **Issue:** `fit: true` on every `replace()` call zoomed the entire graph out further with each click — a real UX bug independent of the targeting problem.
- **Fix:** `fit: false` on `replace()`, keeping `fit: true` only on the one initial `start()`.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`
- **Committed in:** `92329657`

**3. [Rule 1 - Bug] Directory click point derived from centroid instead of bounding box**
- **Found during:** Task 3's live-browser investigation.
- **Issue:** `computeGeometry` used `renderedPosition()` (a compound's centroid), which for a compound WITH children usually lands on a child, not the parent.
- **Fix:** Derive the click point from the rendered bounding box (`includeLabels: false`) instead — now reliable for LEAF nodes (files, collapsed directories).
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`
- **Committed in:** `92329657`

### Unresolved, honestly documented (not auto-fixed — outside this plan's engineering budget, explicitly acknowledged non-blocking at Task 4)

**4. [Genuine finding, not a deviation from correctness] Collapse-by-click hit-testing defect**
- **Found during:** Task 3's `graph-expand-check.mjs` run.
- **Issue:** Re-collapsing an expanded directory compound via a real mouse click could not be demonstrated in 85+ real-mouse attempts across 5 candidates x 17 jitter offsets. Root cause: cytoscape-elk leaves ~1.1px of clickable parent margin once a compound has children. Three mitigations tried, all measured ineffective (cytoscape CSS padding, ELK graph-level padding, ELK per-node padding).
- **Disposition:** NOT fixed in this plan. Recorded as `WINDOWS.md` entry 27. Assigned to 05-07 by maintainer decision (Task 4) — 05-07 must ship an explicit collapse affordance rather than depend on the sub-2px margin.
- **Files:** `web/src/lib/components/graph/GraphCanvas.svelte` (the compound layout/style surface; no code change fixed this).
- **Verification:** `corpora/graph-expand-check.json` (`success: false`), committed as evidence.

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bugs) + 1 genuine unresolved finding, explicitly acknowledged non-blocking by the maintainer at Task 4 and assigned to 05-07.
**Impact on plan:** The three auto-fixes were necessary corrections to real bugs found during live investigation, in scope of the task that found them. The unresolved finding does not affect this plan's own success criteria (GRF-01's re-measure PASSED; the collapsed default IS the first paint; expand/collapse toggle LOGIC is fully proven at the unit level) — it affects a live-mouse UX gesture that 05-07 must now solve explicitly, named rather than silently inherited.

## Issues Encountered

None beyond what is already documented above as deviations/findings. Task 4's checkpoint held the plan open (by design, `gate="blocking-human"`, never auto-approved) from `92329657`'s commit (2026-08-30T21:30:34-04:00) until the maintainer's answer (2026-08-31); this is the checkpoint mechanism working as designed, not an issue.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **05-05, 05-06 and 05-07 are unblocked.** Their preconditions (both clauses: PASS verdict + `release-collapsed` answer) are now satisfied, and their preconditions/wave numbers were already re-pointed by commit `56190931`, verified unmodified by this task.
- **05-07 carries a real, named piece of new scope:** an explicit collapse affordance for its file-level expansion, since its verified three-click contract (expand → collapse-to-zero → re-expand) will hit the identical hit-testing defect this plan found at directory level.
- **05-05 gains available data, unused so far:** collapsed directories now carry `cycleIds` and edges carry `aggregatedFrom` — 05-05's cycle styling and edge-detail table can read these directly, no derivation required.
- **corpora/graph-render-threshold.json remains at exactly 1 commit** — the pre-measurement lock is intact through this entire remedy, verified again at this close-out.

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-31*

## Self-Check: PASSED
