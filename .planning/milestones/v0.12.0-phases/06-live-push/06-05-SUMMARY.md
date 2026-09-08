---
phase: 06-live-push
plan: 05
subsystem: ui
tags: [svelte5, cytoscape, elk, live-push, graph-layout, playwright]

requires:
  - phase: 06-live-push
    provides: "06-03's live-client/live-store (epoch-scoped generation gate, the liveStore context key) and 06-04's real WatchGraph transport — this plan's own live event source"
provides:
  - "createFileGraphRenderer's liveUpdate(): D-06's two-path live-update seam — a no-layout fast path when the rendered node id set is unchanged, and a structural path (nodes added/removed) whose survivor positions are authoritatively written back after layout settles, exact under numeric equality"
  - "A per-renderer layout-generation token (runLayout) serializing start()/replace()/add()/liveUpdate(): a layoutstop whose token is stale performs no restoration and publishes no geometry"
  - "The graph route's own live subscription (issueLiveGraphRefresh): re-fetches fileGraph (+ fileSymbols for any expanded file) through the rpcs it already owns, coalesced with a pending-generation flag, feeding the result through a NEW liveElements prop rather than the plain replace() path"
  - "web/scripts/graph-live-update-check.mjs: a self-contained real-browser measurement owning its own codegraph ui lifecycle, measuring survivor displacement at guava scale"
  - "corpora/graph-live-update-check.json: the committed guava-scale record — 0px max leaf displacement across 349 survivors"
affects: [06-06, 06-07]

actuals:
  tokens: 23736
  tasks: 3
  commits: 6
  plan_head_before: 49c98c17

tech-stack:
  added: []
  patterns:
    - "Identity-comparison guard (lastAppliedElements / lastAppliedLiveElements) replacing a bare boolean skip-first-run flag on a prop-tracking $effect: comparing the CURRENT reactive value against what was last actually applied is robust to an effect firing more than once for a single logical update, where a boolean flag silently eats whichever invocation happens to be first regardless of whether it's genuinely new data"
    - "Layout-generation token minted once per runLayout call (not per call site), so every layout-triggering method (start/replace/add/liveUpdate) inherits serialization for free; a stale layoutstop (superseded by a newer run before it fires) is a checked no-op rather than a race"
    - "Two-path live-update decision keyed on node ID SET equality (not object identity): unchanged node ids take a synchronous, no-layout data-merge path; a changed id set takes an async layout path with a pre-swap position capture and post-layout write-back, restricted to non-parent survivors since a compound's position is derived from its children in cytoscape"
    - "compoundIds inferred at measurement time from the SAME id/parent-path convention this codebase already uses (a leaf id is `${parentId}/${name}`) rather than any new seam — a script-side heuristic, not a production contract"

key-files:
  created:
    - web/tests/graph-live-update.test.ts
    - web/scripts/graph-live-update-check.mjs
    - corpora/graph-live-update-check.json
  modified:
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/routes/graph/+page.svelte

key-decisions:
  - "D-06's write-back is the ONLY source of the zero-displacement guarantee: LAYOUT_OPTIONS gained three ELK interactive-strategy options (org.eclipse.elk.interactive, layered.cycleBreaking.strategy=INTERACTIVE, layered.layering.strategy=INTERACTIVE, layered.crossingMinimization.semiInteractive=true), verified present in the installed elkjs@0.9.3 bundle at its un-hoisted pnpm store path, paired against a known-present control term per this session's own documented trap. These options are the approximate half (where NEW nodes land); the write-back in runLayout is the exact half (where SURVIVORS end up), matching the ELK maintainer's own stated limitation (eclipse-elk#355) that the layered algorithm cannot precisely fix positions, only somewhat preserve topology."
  - "elements (the graph route's full-replace prop) converted from a $derived of graphState to an explicit $state, updated only at the two points a genuine user gesture changes what should render (the initial fetch, toggleDirectory's two branches) — deliberately NOT updated by a live refresh, so a live event can never accidentally route through GraphCanvas's plain replace() path, which carries no position write-back. graphState.response IS updated by a live refresh, so a SUBSEQUENT user gesture reads fresh data rather than reverting the live update on its next toggle."
  - "Two real, distinct double-invocation bugs were found and fixed this session, both via real-browser testing at guava scale (neither was caught by 13 unit tests with real headless cytoscape+ELK, nor by two additional ad hoc headless reproductions built specifically to isolate this — the write-back ALGORITHM was proven correct in isolation both times; the bug was in HOW OFTEN it was invoked): (1) GraphCanvas's construction effect ran twice at initial mount once `elements` became an independent $state write instead of a $derived tied to graphState's own update — fixed by replacing a boolean skip-first-run guard with an identity comparison against the array reference actually applied (lastAppliedElements). (2) The SIXTH effect (liveElements tracker) fired twice for a single logical live update — the second, spurious invocation saw the first invocation's own synchronous swapElements() as \"already current\", took the fast path, and published a geometry snapshot where every survivor sat at cytoscape's (0,0) post-swap default before the first invocation's own asynchronous write-back ever ran. Fixed with the identical identity-comparison discipline (lastAppliedLiveElements). Neither root cause (why Svelte 5 re-invokes these specific effects twice) was fully diagnosed; the fix is robust regardless, since it only ever treats a REPEATED reference as a no-op."
  - "Deviation from the plan's literal 'create ONE new source file' for Task 3's mutation: measured directly, before writing the script, that a single new file adds exactly one FileGraphResponse.nodes entry regardless of its own symbol count (internal/query/traverse.go's FileGraph() aggregates strictly per FilePath) — verified via a real RPC diff (3233 -> 3234, one path added) against the live guava corpus. A single file cannot reach the required addedNodeCount >= 5 floor, and reaching it via a follow-on symbol-expansion click would itself re-scramble survivor positions (add() has no write-back), defeating the measurement. The script creates 6 minimal new files instead, recorded exactly in the committed record's mutationDescription."
  - "The measurement script waits for the geometry seam to be STABLE (two consecutive identical reads, 150ms apart) before ever snapshotting, not merely present — GraphCanvas's construction effect publishing an ephemeral, differently-sized geometry array during the double-mount (before this session's fix) would otherwise have been captured as the 'settled' baseline."
  - "The script's directory-expansion step tries a bounded candidate pool (up to 15, largest fileCount first) and SKIPS any candidate whose reported click position doesn't land, rather than aborting the whole run on the first miss — mirroring graph-expand-check.mjs's own established candidate-retry discipline for this exact corpus's edge density."

patterns-established:
  - "Real-browser measurement scripts that mutate a live corpus checkout must wait for geometry STABILITY, not mere presence, before treating a snapshot as authoritative — a component's construction effect firing more than once for a single logical mount is a real, encountered failure mode in this stack, not a hypothetical."

requirements-completed: [LIV-04, LIV-02]

coverage:
  - id: D1
    description: "A live update whose node id set is unchanged updates data in place, runs no layout, and leaves every position untouched under exact numeric equality"
    requirement: LIV-04
    verification:
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#fast path: an unchanged node id set updates data in place, runs NO layout, republishes geometry once, and leaves every position untouched"
        status: pass
    human_judgment: false
  - id: D2
    description: "A live update that adds or removes nodes restores every surviving leaf node to exactly its prior position, under real headless cytoscape+ELK"
    requirement: LIV-04
    verification:
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#structural path (nodes added): every surviving leaf node ends at EXACTLY its prior position, under exact numeric equality"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#structural path (nodes removed): every surviving leaf node ends at EXACTLY its prior position"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#structural path (nodes added AND removed): every surviving leaf node ends at EXACTLY its prior position"
        status: pass
    human_judgment: false
  - id: D3
    description: "Overlapping layout runs are serialized by a generation token: a stale layoutstop performs no restoration and publishes no geometry, proven with positive controls"
    requirement: LIV-04
    verification:
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#two live structural updates back to back: the SECOND run's survivor map is the one restored, the first is a no-op"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#a live update racing a user-driven add(): only the LATER run's geometry publishes, and it reflects add()'s own state"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#a stale layoutstop invoked manually after a newer run has settled publishes nothing and restores nothing, paired with the current run which does publish"
        status: pass
    human_judgment: false
  - id: D4
    description: "The graph route re-fetches through its own rpcs on a new live generation, coalesced with a pending-generation flag, and routes exclusively through the new seam (not the plain replace path)"
    requirement: LIV-02
    verification:
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#a live event issues exactly ONE additional file-graph call; a replayed lower generation issues none"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#the live path applies via the NEW seam (no full remove+add), paired with a user-driven expansion still using the plain replace path"
        status: pass
      - kind: unit
        ref: "web/tests/graph-live-update.test.ts#an event arriving during an in-flight re-fetch produces no second concurrent call; the newest generation is applied once it settles"
        status: pass
    human_judgment: false
  - id: D5
    description: "At real guava scale (3,233 files), a real re-index that adds real nodes leaves every surviving leaf node exactly where it was, measured in a real browser against a real codegraph ui process"
    requirement: LIV-04
    verification:
      - kind: e2e
        ref: "corpora/graph-live-update-check.json (leafSurvivorCount 349, maxLeafDisplacementPx 0, addedNodeCount 6)"
        status: pass
    human_judgment: false

duration: 640min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 5: Graph Layout Stability Across a Live Update Summary

**A two-path live-update seam on GraphCanvas.svelte — a no-layout fast path for data-only changes and an authoritative position write-back for structural ones — measured at guava scale in a real browser: 349 surviving leaf nodes, 0px maximum displacement, across a real re-index that added 6 real files.**

## Performance

- **Duration:** ~10.5 hours (includes two rounds of real-browser regression diagnosis — see Deviations)
- **Started:** 2026-09-07T~14:00:00Z (approx)
- **Completed:** 2026-09-07T~00:30:00Z (approx, next-day wall clock)
- **Tasks:** 3 (2 `type="auto"`, one `tdd="true"`)
- **Files modified:** 5 (3 created, 2 modified)

## Accomplishments

- **Task 1 (`tdd="true"`, GraphCanvas.svelte):** `createFileGraphRenderer` gained `liveUpdate()`, D-06's two-path decision. The FAST PATH (node id set unchanged) merges data in place via `.data()`/`.classes()` calls and republishes geometry synchronously — zero layout calls, proven via a `vi.spyOn` assertion. The STRUCTURAL PATH (nodes added/removed) captures every currently-rendered node's model position before the element swap, runs the shared layered ELK layout with three added interactive-strategy options, and — on `layoutstop`, before geometry republishes — writes each surviving NON-PARENT node's captured position back exactly. `LAYOUT_OPTIONS` stayed a single shared configuration (verified via the plan's own grep gate: `LAYOUT_OPTIONS` count 2, `LAYOUT_OPTIONS[_A-Z]` count 0). `runLayout` gained a per-renderer generation token minted before every layout start, so `start()`/`replace()`/`add()`/`liveUpdate()` all inherit serialization: a `layoutstop` whose captured token is stale performs no restoration and publishes no geometry — closing the race where cytoscape's own layout events bubble to `cy` and an older run's completion can invoke a newer run's still-pending listener (confirmed by reading `extension.mjs`'s `bubble: true` this session). `replace()`'s own element-swap logic was extracted into `swapElements()`, now shared with `liveUpdate()`'s structural path. 13/13 tests pass (`web/tests/graph-live-update.test.ts`), observed RED first (`renderer.liveUpdate is not a function`, 12/13 failing) before the implementation existed.
- **Task 2 (`+page.svelte`):** The graph route reads `liveStore` from context and, on a new generation, re-issues the `fileGraph` call it already owns plus a `fileSymbols` call for every file currently showing its symbols, coalesced with a pending-generation flag mirroring `health`/`browse`'s own 06-03 pattern. The combined rollup feeds `GraphCanvas` through a NEW `liveElements` prop into Task 1's seam — never through `elements`, whose own effect calls the plain `replace()` path with no write-back. `elements` was converted from a `$derived` of `graphState` to an explicit `$state`, recomputed only at the two points a genuine user gesture changes what should render. 3 new route-level tests pass, using a fuller fake cytoscape core (`getElementById`-aware) proving: exactly one additional `fileGraph` call per live event with replayed-lower-generation suppression; the live path routes through `liveUpdate` (no `elements().remove()`) while a user-driven expansion still uses `replace()`; and the in-flight coalescer.
- **Task 3 (real-browser measurement):** `web/scripts/graph-live-update-check.mjs` owns its own lifecycle — resolves the guava corpus checkout path from `corpora/graph-render-threshold.json`'s pinned repo/sha (the same slug formula `internal/corpora/manifest.go`'s `Entry.Dir` uses, reimplemented in JS and verified against the real on-disk directory), starts a real `codegraph ui` child process, expands a real directory (232 files) via real trusted mouse input with a bounded, skip-on-miss candidate pool, mutates the live corpus checkout with 6 new files and a real `codegraph sync`, and measures survivor displacement. **Result: 349 leaf survivors, 0px maximum and mean displacement; 17 compound survivors (non-binding), also 0px; 6 nodes added, 0 removed; `fileLevelNodeCountBefore`/`After` 366/372.** Confirmed the gate can go red: with the write-back temporarily disabled, the identical measurement produced `maxLeafDisplacementPx: 599.55` and `success: false` (see Deviations for the full record).

## Task Commits

1. **Task 1 RED:** `bfcc747a` — `test(06-05): add failing tests for the live-update seam` (13 tests, 12 failing with the intentional `liveUpdate is not a function`).
2. **Task 1 GREEN:** `7da5ed67` — `feat(06-05): the live-update seam — fast path, write-back, generation guard` (13/13 pass).
3. **Task 2:** `fbf132ec` — `feat(06-05): graph route subscribes to live updates through its own rpcs` (463/463 pass; includes a fix for a route-level test-fake bug this task's own tests surfaced, folded into the same commit — see Deviations).
4. **Fix (found during Task 3):** `e52b161a` — `fix(06-05): replace the fragile skip-first-run guard with a reference check`.
5. **Fix (found during Task 3):** `790e60df` — `fix(06-05): guard the live-update effect against a second spurious invocation`.
6. **Task 3:** `290ecdca` — `feat(06-05): measure survivor displacement at guava scale in a real browser`.

**Plan metadata:** this commit (docs).

## Files Created/Modified

- `web/src/lib/components/graph/GraphCanvas.svelte` — `liveUpdate()`, `swapElements()`, the layout generation token, the three ELK interactive options, `lastAppliedElements`/`lastAppliedLiveElements` identity guards, the `liveElements` prop and its tracking effect.
- `web/src/routes/graph/+page.svelte` — `elements` converted to explicit `$state`; `liveElements` state; `issueLiveGraphRefresh` and the live-store subscription effect; the `<GraphCanvas>` markup gained `{liveElements}`.
- `web/tests/graph-live-update.test.ts` — new: 16 tests total (13 from Task 1, 3 from Task 2).
- `web/scripts/graph-live-update-check.mjs` — new: the real-browser guava-scale measurement.
- `corpora/graph-live-update-check.json` — new: the committed diagnostic record.

## Decisions Made

See `key-decisions` in frontmatter for full detail. Highlights:
- D-06's write-back (exact) vs. ELK's interactive strategies (approximate) is the split this plan implements literally: the interactive options place NEW nodes sensibly; the write-back is what makes a SURVIVOR's displacement provably zero.
- `elements` (route) decoupled from live refreshes entirely, by construction, so a live event can never accidentally take the plain `replace()` path.
- Two real double-invocation bugs (see Deviations) were fixed with the SAME identity-comparison pattern, applied twice, rather than two different fixes — the pattern generalizes to "any `$effect` tracking a $state-backed prop that might fire more than once for one logical change."

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] GraphCanvas's construction effect fires twice at initial mount, silently swallowing the first real `elements` change**
- **Found during:** Task 3's real-browser verification (`graph-collapse-affordance-check.mjs`, an EXISTING, previously-passing precedent script, newly failed against both the guava corpus AND this repository's own index after Task 2's changes).
- **Issue:** Converting the graph route's `elements` from a `$derived` of `graphState` to an explicit `$state` (Task 2, needed to decouple live refreshes from the full-replace path) caused GraphCanvas's construction effect (`if (!container) return; ...`) to run TWICE in quick succession during initial mount, with a cleanup in between, for reasons not fully root-caused within this session's budget (isolated to the `$derived`→`$state` conversion specifically via bisection — reordering the two state writes in `onMount`'s `.then()` callback did not change the behavior; unifying the template's branch condition to depend on `graphState` alone did not change it either). The SURVIVING (second) mount's own "apply elements" effect used a boolean `trackedInitialized` flag that unconditionally treated its own FIRST invocation as "matches construction" — but since a genuine `elements` change (a user's directory-expand click) could arrive in the window before that effect's own natural first run, the boolean silently ate it: `replace()` was never called, so clicking a directory did nothing, confirmed against BOTH the guava corpus and this repo's own smaller index.
- **Fix:** Replaced the boolean guard with `lastAppliedElements`, an identity comparison against the actual array reference construction handed to cytoscape (or that `replace()` last applied) — correct regardless of how many times construction happens to run for one logical mount.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`
- **Verification:** `graph-collapse-affordance-check.mjs` re-run against this repository's own index before (fatal: `clickCanvasUntil: condition never became true`) and after (success: true, all 5 phases) the fix. Full suite 463/463 unaffected.
- **Commit:** `e52b161a`

**2. [Rule 1 - Bug] The SIXTH effect (liveElements tracker) fires twice per live update, publishing a pre-write-back geometry snapshot**
- **Found during:** Task 3's own guava-scale displacement measurement — `maxLeafDisplacementPx` was 2988.96px on the first real run (expected 0), with EVERY survivor (leaf and compound alike) landing at nearly the identical rendered coordinate.
- **Issue:** Isolated via targeted instrumentation (a debug counter inside `liveUpdate()` itself, since two independent headless reproductions of the write-back algorithm — a 20/25-node and a 360-node compound-with-children scenario matching guava's own shape — both passed with 0 mismatches, ruling out the write-back logic itself): the sixth effect (tracking `liveElements`) invoked `renderer.liveUpdate(batch)` TWICE for a single logical live-update value. The first invocation correctly detected the structural change (`nodeSetUnchanged: false`, 366→372) and started the async ELK layout with `survivorPositions` captured. Before that layout's own `layoutstop` ever fired, a SECOND invocation of the same effect ran, observed `opts.cy.nodes()` already reflecting the FIRST invocation's own synchronous `swapElements()` result, computed `nodeSetUnchanged: true` (fast path), and published geometry IMMEDIATELY — a snapshot where every node still sat at cytoscape's `(0,0)` post-swap default (confirmed directly: the "converged" rendered coordinate matched exactly what `modelPosition=(0,0)` maps to under the page's own recorded pan/zoom). The first invocation's own write-back never had a chance to run before this premature snapshot was captured and treated as "settled" by the measurement script's own poll condition.
- **Fix:** `lastAppliedLiveElements`, the identical identity-comparison discipline as fix #1 above, applied to the sixth effect: a repeated invocation carrying the SAME `liveElements` reference is now a no-op.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`
- **Verification:** Guava-scale re-measurement after the fix: `maxLeafDisplacementPx: 0` (was 2988.96px). Confirmed the gate genuinely detects a broken write-back: re-measured with the write-back temporarily disabled (`if (survivorPositions && false)`), producing `maxLeafDisplacementPx: 599.55, success: false` — see the recorded RED evidence below. Full suite 463/463 unaffected.
- **Commit:** `790e60df`

**3. [Rule 1 - Deviation, disclosed] Task 3's mutation creates 6 new files instead of the plan's literal "ONE new source file"**
- **Found during:** Writing Task 3's script, before any real-browser run — verified directly via a real RPC diff against the live guava corpus (`FileGraphResponse.nodes` length 3233 → 3234 after adding one file with 5 methods; `added: [the one new path]`).
- **Issue:** `internal/query/traverse.go`'s `FileGraph()` aggregates strictly per `FilePath` (`fileAggs` keyed by `n.FilePath`), so a single new file contributes exactly ONE new entry to the file-graph rollup regardless of how many symbols it declares. The plan's own `addedNodeCount >= 5` floor cannot be satisfied by one file at this granularity, and satisfying it via a follow-on symbol-expansion click (`add()`, which has no write-back) would itself re-scramble survivor positions — defeating the exact thing being measured.
- **Fix:** The script creates 6 minimal new Java files (each an empty final class with a private no-op constructor) inside the same already-expanded directory, in ONE structural swap. The exact mutation is recorded verbatim in `corpora/graph-live-update-check.json`'s `mutationDescription` field — never narrated as "a file was added".
- **Files modified:** `web/scripts/graph-live-update-check.mjs`
- **Verification:** `addedNodeCount: 6` in the committed record, satisfying the `>= 5` floor with margin.
- **Committed in:** `290ecdca` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 bugs found and fixed via real-browser testing at scale, 1 disclosed measurement-design deviation with direct evidence). **Impact:** The two bug fixes are essential — without them, the graph view's click-to-expand interaction was silently broken and the live-update write-back was silently ineffective at real corpus scale, despite 13 unit tests (with real headless cytoscape+ELK) passing throughout. Both bugs were invisible to fixture-scale testing and were caught specifically by this plan's own mandate to measure at guava scale in a real browser — direct vindication of that requirement. No scope creep beyond what was needed to make the plan's own gates genuinely pass.

## Issues Encountered

**The root cause of BOTH double-invocation bugs (why specific Svelte 5 `$effect`s fire twice for a single logical value change, tied to `$state`-backed array props) was not fully diagnosed within this session's budget.** Extensive bisection (reordering statements, isolating individual prop changes, unifying template branch conditions, comparing against the pre-06-05 working baseline) narrowed the trigger to "an `$effect` tracking an array-shaped `$state` value that changes via an independent write (not a `$derived` recomputation tied to the SAME flush as another change)" but did not identify the exact Svelte-internal mechanism. The chosen fix (identity-comparison guards, `lastAppliedElements`/`lastAppliedLiveElements`) is robust regardless of the underlying cause — it makes a repeated invocation for an unchanged reference a correctness no-op — but a future contributor investigating Svelte 5's own effect-scheduling semantics for this pattern may find and eliminate the double-invocation at its source. Recorded here rather than left silent.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `task web:test` — PASS: `web:test: observed numTotalTests=463 numPassedTests=463 (vitest exit 0)`.
- `pnpm -C web run check` — exits 0: `COMPLETED 1159 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`.
- `node web/scripts/graph-live-update-check.mjs` — PASS, `success: true`, record committed and byte-identical across two independent runs.
- `git status --porcelain -- web/build` — empty at every commit in this plan; `web/build` was rebuilt locally (uncommitted) multiple times purely to exercise the real-browser measurement against the current frontend source, and reverted to its committed state after every use. 06-06 still performs the phase's single staged rebuild.

### Guava-scale measurement (the binding record)

```json
{
  "expandedDirectories": ["guava-tests/test/com/google/common/collect"],
  "mutationDescription": "Created 6 new minimal Java files (...LivePushProbe0.java .. LivePushProbe5.java) inside \"guava-tests/test/com/google/common/collect\" (package com.google.common.collect), each declaring one empty final class with a private no-op constructor and zero other symbols; ran a real `codegraph sync` re-index against the live corpus checkout.",
  "fileLevelNodeCountBefore": 366,
  "fileLevelNodeCountAfter": 372,
  "addedNodeCount": 6,
  "removedNodeCount": 0,
  "leafSurvivorCount": 349,
  "maxLeafDisplacementPx": 0,
  "meanLeafDisplacementPx": 0,
  "compoundSurvivorCount": 17,
  "maxCompoundDisplacementPx": 0,
  "corpusCleanAtExit": true,
  "success": true
}
```

### RED evidence — the same measurement with the write-back deliberately disabled

Recorded live, this session, before restoring the fix (proving the gate is not vacuously green):

```json
{
  "fileLevelNodeCountBefore": 366,
  "fileLevelNodeCountAfter": 372,
  "addedNodeCount": 6,
  "removedNodeCount": 0,
  "leafSurvivorCount": 349,
  "maxLeafDisplacementPx": 599.5479546763196,
  "meanLeafDisplacementPx": 54.71407703215581,
  "compoundSurvivorCount": 17,
  "maxCompoundDisplacementPx": 64.54589364377257,
  "corpusCleanAtExit": true,
  "success": false
}
```

## Known Stubs

None. The live-update seam, the route's subscription, and the measurement script are all fully implemented and exercised end-to-end — 16 unit tests against real headless cytoscape+ELK and a fully controllable fake cy, plus a real-browser, real-server, real-corpus measurement. No placeholder bodies remain in this plan's files.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Graph layout stability across a live update is proven at guava scale: 0px displacement across 349 real survivors, with the gate demonstrated capable of going red. `06-06` can proceed with its own real-browser gates (the fixed-port proxy and the three-tab check) without further graph-layout work. `06-07`'s real-process gate and the phase's single `web/build` rebuild are unaffected by anything in this plan. `LIV-04` and `LIV-02` are both satisfied by this plan's own coverage; no blockers for the remaining two waves.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- All 5 files (3 created, 2 modified) — FOUND on disk.
- Commits `bfcc747a`, `7da5ed67`, `fbf132ec`, `e52b161a`, `790e60df`, `290ecdca` — all FOUND in `git log --oneline --all`.
- Plan-level `<verification>` re-run live: `task web:test` PASS (463/463), `pnpm -C web run check` 0 errors/0 warnings, `node web/scripts/graph-live-update-check.mjs` PASS with `success: true`, `git status --porcelain -- web/build` empty.
