---
phase: 05-file-package-graph-view
plan: 03
subsystem: ui
tags: [svelte, cytoscape, cytoscape-elk, elkjs, graph-rendering, tdd, tracer]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-01's Engine.FileGraph() rollup and stronglyConnectedCycles() cycle detection; 05-02's frozen 12th rpc (FileGraph) and its measured guava-scale response size (3,713,528 bytes)"
provides:
  - "cytoscape@3.34.2 + cytoscape-elk@2.3.0 (direct, exact-pinned) + elkjs@0.9.3 (transitive, pnpm-lock-pinned) — the three approved renderer dependencies, now in the committed bundle"
  - "file-graph-transform.ts — the pure, DOM-free, cytoscape-free FileGraphResponse-to-element-data transform (GRF-02's compound directory grouping)"
  - "GraphCanvas.svelte — GRF-05's renderer swap seam, the ONLY importer of cytoscape in web/src, publishing the timeToInteractiveMs/layoutDurationMs/nodeCount/edgeCount measurement seam 05-04 reads"
  - "/graph — filled with a working, hierarchically laid-out, directory-grouped file graph against this repository's own live index (measured: 706 nodes, 1068 edges, ~2.1s time-to-interactive)"
affects: [05-04, 05-05, 05-06, 05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 11118
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added:
    - "cytoscape@3.34.2 (direct, exact-pinned)"
    - "cytoscape-elk@2.3.0 (direct, exact-pinned)"
    - "elkjs@0.9.3 (transitive via cytoscape-elk's ^0.9.3 range, pnpm-lock-pinned only)"
  patterns:
    - "Renderer swap seam: exactly one component (GraphCanvas.svelte) imports the rendering library; everything else speaks in plain element-data objects a pure transform module produces"
    - "Two-part cytoscape test strategy: MOCK 'cytoscape'/'cytoscape-elk' for route-mount lifecycle assertions (construction/destroy counts) since jsdom has no <canvas> 2D context and a real, non-headless cytoscape instance throws at construction time; use a REAL cytoscape instance in headless:true mode, run directly against the transform's output, for graph-model assertions (parent chain, counts)"
    - "Boolean cytoscape data selectors use the truthy/falsy existence syntax ([?field]/[!field]), not an equality comparison ([field = true]) — the latter is silently invalid and never matches"

key-files:
  created:
    - web/src/lib/components/graph/file-graph-transform.ts
    - web/src/lib/components/graph/graph-style.ts
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/cytoscape-elk.d.ts
    - web/tests/file-graph-transform.test.ts
    - web/tests/graph-tracer.test.ts
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/src/app.d.ts
    - web/src/routes/graph/+page.svelte
    - web/build (rebuilt, 107 source files / 32 output files, both drift halves matching)

key-decisions:
  - "Maintainer approved all three packages (cytoscape, cytoscape-elk, elkjs) — verbatim decision recorded below under 'Maintainer Decision (Task 1)'. The [SUS]/too-new verdict on cytoscape was confirmed the same false-positive class already approved twice this milestone."
  - "cytoscape-elk resolved elkjs to 0.9.3 (its own ^0.9.3 range's ceiling), NOT the 0.12.0 whose bundle size 05-RESEARCH.md measured. An override to force 0.12.0 was offered and declined by the maintainer — 2.3.0 was tested against the 0.9.x line, and trading a known-good pairing for a matching bundle figure was the wrong direction."
  - "Ambient module declaration for cytoscape-elk lives in its own file (web/src/cytoscape-elk.d.ts), not folded into app.d.ts — confirmed empirically that svelte-check does not pick up the identical declaration when it lives inside app.d.ts alongside app.d.ts's own declare-global block, but does pick it up in a standalone .d.ts file."
  - "cytoscape-elk's own source imports elkjs/lib/elk.bundled.js (the synchronous, main-thread build), never elk-api.js's Web-Worker-based variant — confirmed by reading both its src/layout.js and its published dist bundle (zero occurrences of the string 'Worker'). No worker configuration is possible through its public API, so the CSP's absent worker-src directive (T-05-13) is never actually at risk from this dependency, and no CSP violation was observed live."

patterns-established:
  - "GraphCanvas.svelte's header comment states, in the locked artifact's own words, exactly where each of the seam's two timers starts and stops — a measurement seam is documented as a measurement seam, not as an ordinary API."

requirements-completed: [GRF-01, GRF-02, GRF-05]

coverage:
  - id: D1
    description: "The three renderer dependencies (cytoscape, cytoscape-elk, elkjs) were approved by a human, by name, before any install ran — a blocking-human package-legitimacy checkpoint, never auto-approved."
    requirement: GRF-05
    verification:
      - kind: manual_procedural
        ref: "Maintainer verbatim reply, 2026-08-30: 'approve all three' — recorded in this SUMMARY's Maintainer Decision section"
        status: pass
    human_judgment: true
    rationale: "Package-legitimacy approval before install is a one-way trust decision only a human can make — gate=blocking-human, never auto-approved even under auto_advance."
  - id: D2
    description: "file-graph-transform.ts turns a FileGraphResponse into compound-parented, directory-grouped Cytoscape element data, losslessly (exact node/edge counts), with zero DOM dependency, zero cytoscape import, and zero cycle-membership computation (copies wire fields unchanged)."
    requirement: GRF-02
    verification:
      - kind: unit
        ref: "web/tests/file-graph-transform.test.ts (8 tests: full compound chain, root-file no-parent, file/dir discrimination with exact count, edge verbatim copy + exact count, cycle passthrough zero and non-zero, directory dedup, empty response)"
        status: pass
    human_judgment: false
  - id: D3
    description: "GraphCanvas.svelte is the ONLY module in web/src that imports cytoscape or a cytoscape extension; the layout is elk/layered with hierarchyHandling INCLUDE_CHILDREN and no other layout name or force-directed algorithm appears anywhere; no worker is configured."
    requirement: GRF-05
    verification:
      - kind: unit
        ref: "rg -l \"from 'cytoscape\" web/src reports exactly 1 file (GraphCanvas.svelte); rg -c \"name: 'elk'\" GraphCanvas.svelte >= 1; rg -c 'fcose|cose-bilkent' web/src == 0; rg -c 'workerUrl|workerFactory|new Worker|blob:' GraphCanvas.svelte == 0"
        status: pass
      - kind: manual_procedural
        ref: "Live browser session against this repository's own index: no Content-Security-Policy violation in the console, confirmed after clearing and reloading (recorded below)"
        status: pass
    human_judgment: false
  - id: D4
    description: "/graph mounts, issues exactly one FileGraph call, renders a real loading/failed/empty/loaded state machine composing the one existing classifyRpcError classifier, and never imports cytoscape directly. Unmounting while the request is pending constructs zero renderer instances; unmounting after the canvas mounted destroys exactly one."
    requirement: GRF-02
    verification:
      - kind: unit
        ref: "web/tests/graph-tracer.test.ts (8 tests: one-call-on-mount + one construction, rejected-call failure state + zero construction, NotFound vs unknown failure states distinct, empty-graph real empty state + zero construction, pending-unmount zero construction/zero destroy, mounted-unmount exactly one destroy, measurement-seam event+global, real headless-cytoscape model test)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A developer opening /graph on a real repository sees files grouped into directory boxes with aggregated dependency edges, laid out hierarchically (never a hairball), with real user-facing copy containing no planning vocabulary."
    requirement: GRF-02
    verification:
      - kind: automated_ui
        ref: "Live browser session (agent-browser) against this repository's own index: screenshot confirms directory-grouped nested boxes with edges between files, not a hairball; subtitle text confirmed to contain no phase/requirement/plan identifier"
        status: pass
      - kind: manual_procedural
        ref: "Pan/zoom interactivity could not be conclusively confirmed via synthetic browser automation — see 'Issues Encountered' below"
        status: unknown
    human_judgment: true
    rationale: "Layout quality and gesture-level pan/zoom smoothness are visual/interaction properties this codebase's own validation strategy (05-VALIDATION.md's Manual-Only Verifications table) already designates as requiring a live human session, not automation — the render/style correctness half is machine- and agent-verified above, but true interactive smoothness needs a human's own hands on the trackpad."

duration: 40min
completed: 2026-08-30
status: complete
---

# Phase 5 Plan 3: GraphCanvas Seam + Filled /graph Route Summary

**Wired the thinnest production-quality path from the FileGraph rpc to a rendered, hierarchically laid-out, directory-grouped graph on `/graph` — cytoscape@3.34.2 + cytoscape-elk@2.3.0 (elkjs resolved to 0.9.3) behind a single-importer renderer seam, verified end-to-end against this repository's own live index (706 nodes, 1068 edges, ~2.1s time-to-interactive).**

## Performance

- **Duration:** ~40 min (resumed at Task 1's `blocking-human` checkpoint)
- **Started:** 2026-08-30T13:00:00-04:00 (approx, continuation resume)
- **Completed:** 2026-08-30T13:40:00-04:00 (approx, after live browser verification)
- **Tasks:** 3
- **Files modified:** 12 (6 created, 6 modified, including the rebuilt `web/build` tree)

## Accomplishments

- Maintainer approved all three renderer dependencies by name, verbatim, before any install ran (Task 1) — recorded below.
- `cytoscape`/`cytoscape-elk` installed as exact-pinned direct dependencies; `elkjs` resolved transitively to `0.9.3` (not the `0.12.0` research measured), pinned only by `pnpm-lock.yaml` per the maintainer's explicit decision not to override it.
- `file-graph-transform.ts` — a pure, DOM-free, cytoscape-free transform turning a `FileGraphResponse` into compound-parented, directory-grouped Cytoscape element data, losslessly.
- `GraphCanvas.svelte` — GRF-05's renderer swap seam, the only file in `web/src` that imports cytoscape or a cytoscape extension, publishing the four-value measurement seam (`timeToInteractiveMs`, `layoutDurationMs`, `nodeCount`, `edgeCount`) 05-04 will read.
- `/graph` fills the Phase-2 placeholder entirely: one `FileGraph` call on mount, a real loading/failed/empty/loaded state machine, real user-facing copy with zero planning vocabulary.
- `web/build` rebuilt and re-verified against the committed source tree; `task web:drift`, `web:audit`, `web:lockfile`, `web:deps:strict` all green.
- **Live browser verification against this repository's own index** (not merely assumed from passing tests): found and fixed a real bug (cytoscape boolean selector syntax), then confirmed the fix — directory-grouped, hierarchically laid-out rendering with no console errors and no CSP violation, and a real measured `timeToInteractiveMs` of ~2058ms / `layoutDurationMs` of ~1583ms against 706 nodes / 1068 edges.

## Task Commits

Each task was committed atomically, following TDD RED→GREEN discipline for Task 2:

1. **Task 1: Approve the package-legitimacy verdicts** — no commit (decision-only `blocking-human` checkpoint; the maintainer's verbatim answer is recorded in this SUMMARY, not a code artifact).
2. **Task 2 RED: failing test for file-graph-transform** — `b3539f9b` (test)
   **Task 2 GREEN: install packages + build the DOM-free transform** — `d963e14b` (feat)
3. **Task 3: GraphCanvas seam, filled /graph route, rebuilt bundle** — `42ffa5a4` (feat)
   **Task 3 fix: rebuild web/build (stale drift marker)** — `663fc303` (fix)

_No REFACTOR commit — the GREEN implementation was clean on first pass for Task 2; Task 3 is `type="tracer"`, not TDD, and its one live-bug fix (the boolean selector syntax) landed inside the Task 3 commit before it was pushed, plus one follow-up fix commit for the stale build marker._

## Files Created/Modified

- `web/package.json`, `web/pnpm-lock.yaml` — the two direct exact pins and the one transitive pin.
- `web/src/lib/components/graph/file-graph-transform.ts` — the pure wire-to-elements transform.
- `web/src/lib/components/graph/graph-style.ts` — the cytoscape style sheet as plain data (with the boolean-selector fix).
- `web/src/lib/components/graph/GraphCanvas.svelte` — the renderer swap seam.
- `web/src/cytoscape-elk.d.ts` — the ambient module declaration for cytoscape-elk.
- `web/src/app.d.ts` — the `window.__codegraphFileGraphMetrics` global augmentation.
- `web/src/routes/graph/+page.svelte` — the filled route.
- `web/tests/file-graph-transform.test.ts`, `web/tests/graph-tracer.test.ts` — 16 tests total.
- `web/build` — rebuilt, 107 source files / 32 output files, both drift halves matching the committed tree.

## Maintainer Decision (Task 1, verbatim)

**MAINTAINER DECISION, 2026-08-30: `approve all three`** — `cytoscape`, `cytoscape-elk`, `elkjs`.

The `[SUS]` / `too-new` verdict on `cytoscape` is the same false-positive shape already approved twice this milestone (03-01 `@testing-library/jest-dom`; 04-01 `@tanstack/svelte-table` and `shadcn-svelte`). The heuristic reads **latest-publish-date, not package age**: `cytoscape` was created 2012-10-18 and merely published four days ago, on 2026-08-25. Fourteen years old, 15,505,561 weekly downloads, six maintainers, MIT, zero dependencies.

The orchestrator independently re-verified against the live npm registry, confirming the executor's findings and the novel one it surfaced:
- **No `postinstall` script on any of the three.** (`cytoscape-elk` declares `postpublish`, which runs for the maintainer at publish time and never on consumer install — not a supply-chain concern here.)
- **`cytoscape-elk@2.3.0` declares `elkjs: ^0.9.3`, and only `0.9.0`–`0.9.3` exist, so the resolved install will be `elkjs@0.9.3` — NOT the `0.12.0` whose bundle size `05-RESEARCH.md` measured.** An override to force `0.12.0` was offered and **declined**: `cytoscape-elk@2.3.0` was tested against 0.9.x, and trading a known-good pairing for a matching bundle figure is the wrong direction.
- Net footprint is exactly **3 packages**, no further transitives.

**Required follow-through, discharged:** the resolved `elkjs` version and the measured bundle delta after install are recorded below (see "Resolved Versions and Measured Bundle Delta"), not carried forward as though research's estimate described what shipped.

## Resolved Versions and Measured Bundle Delta

- **Direct, exact-pinned:** `cytoscape@3.34.2`, `cytoscape-elk@2.3.0` (both in `web/package.json`, no caret/tilde — verified: `node -e '...'` printed `exact-pinned runtime deps: 2`).
- **Transitive, lockfile-pinned only:** `elkjs@0.9.3` — confirmed via `rg -n "^  elkjs@" web/pnpm-lock.yaml` and `rg -n "elkjs" web/pnpm-lock.yaml` (resolves under `cytoscape-elk@2.3.0`'s dependency block). NOT declared in `web/package.json` — confirmed via the plan's own `node -e` accounting check.
- **Measured bundle delta (not estimated):** pre-plan baseline (`git archive` of the last commit before this plan touched `web/build`, `53e969ea`): 892 KB total, **169,088 bytes gzip JS**. Post-plan (`web/build` as committed in `663fc303`): 2.7 MB total, **745,964 bytes gzip JS**. Delta: **+576,876 bytes gzip (~563 KB)** — closely matching D-07's ~560 KB gzip estimate (cytoscape core ~137 KB + elkjs ~423 KB), now confirmed against the actual shipped bundle rather than Bundlephobia's per-package figures.

## RED Output (Task 2, verbatim)

```
Error: Failed to resolve import "$lib/components/graph/file-graph-transform" from "tests/file-graph-transform.test.ts". Does the file exist?
  Plugin: vite:import-analysis
...
 Test Files  1 failed | 28 passed (29)
      Tests  268 passed (268)
```

Confirms the module did not exist before the test was written, as required.

## Live Browser Verification Findings (recorded per Task 3's own instruction)

Performed against a real, freshly-built binary (`task build:release`) running `./codegraph ui --no-open`, driven with `agent-browser`, against this repository's own live (self-indexing) index — not merely inferred from passing unit tests:

1. **First pass found a real bug:** the console showed `The selector "node[isDirectory = true]" is invalid` (and the `false` variant) on every load. Cytoscape's boolean data selectors use the truthy/falsy existence syntax (`[?field]` / `[!field]`), not an equality comparison (`[field = true]`) — the latter is silently invalid and the rule never matches, so every node was rendering under cytoscape's default style instead of the intended directory/file distinction.
2. **Fixed in `graph-style.ts`**, rebuilt (`task web:build`), rebuilt the binary, restarted the server, cleared the console, and reloaded: **zero console warnings or errors**, confirmed clean.
3. **No Content-Security-Policy violation** appeared at any point — consistent with the finding (recorded in `GraphCanvas.svelte`'s own header comment) that `cytoscape-elk`'s source imports `elkjs/lib/elk.bundled.js` (the synchronous, no-Worker build) and its published `dist/cytoscape-elk.js` contains the string `"Worker"` zero times.
4. **Rendering confirmed directory-grouped and hierarchical, not a hairball:** screenshot shows nested light-gray directory boxes containing small blue file dots, connected by faint directed edges across box boundaries — the expected shape for a layered, compound-aware layout.
5. **The measurement seam works end-to-end against a real rpc round trip:** `window.__codegraphFileGraphMetrics` read live as `{"timeToInteractiveMs":2057.5,"layoutDurationMs":1582.8,"nodeCount":706,"edgeCount":1068}` — real numbers from a real request against this repository's own (currently stale) index, not a fixture.
6. **Pan/zoom could NOT be conclusively confirmed via synthetic automation.** Multiple attempts at simulated mouse-wheel zoom and click-drag pan (via `agent-browser`'s CDP-backed mouse commands) produced no observable change across five separate screenshots. `cytoscape`'s `userPanningEnabled`/`userZoomingEnabled` remain at their default `true` — neither is disabled anywhere in this plan's code — so nothing in the implementation should block real interaction. This is consistent with `05-VALIDATION.md`'s own Manual-Only Verifications table, which already designates "pan/zoom stays smooth" as a property requiring a live human session rather than automation (canvas-based gesture recognition is not reliably exercisable through synthetic CDP mouse-wheel/drag events in this kind of library). Flagged honestly rather than assumed — a human should confirm real trackpad/mouse pan and zoom before this property is considered proven.

## Decisions Made

- **Task 1:** Maintainer approved all three packages verbatim (`approve all three`) — see "Maintainer Decision" above.
- **Ambient declaration placement:** moved to its own file (`web/src/cytoscape-elk.d.ts`) after confirming empirically that the identical `declare module 'cytoscape-elk' {...}` block is NOT picked up by `svelte-check` when placed inside `app.d.ts` alongside its existing `declare global` block, but IS picked up in a standalone `.d.ts` file. Root cause not fully diagnosed (svelte-check/TS module-graph quirk specific to this file, not a general TypeScript limitation — confirmed the identical text works standalone); documented as an empirical finding rather than left unexplained.
- **`onNodeSelected` is wired but unconsumed in this plan:** `GraphCanvas.svelte` fires a `node-selected`-shaped callback on tap (id + isDirectory), but `/graph` does not pass a handler for it — no drill-down exists yet (deferred to a later plan per D-10). Kept as a real, tested seam rather than removed, since GRF-03's in-place expansion will need exactly this event.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Ambient module declaration for `cytoscape-elk` in `app.d.ts` moved to its own file**
- **Found during:** Task 2/3, running `pnpm check` after installing `cytoscape-elk`.
- **Issue:** `declare module 'cytoscape-elk' { ... }` inside `web/src/app.d.ts` (alongside its pre-existing `declare global { namespace App {...} }` block) was NOT recognized by `svelte-check` — every import of `cytoscape-elk` still reported "Could not find a declaration file". The identical text in a fresh, standalone `.d.ts` file WAS recognized.
- **Fix:** Created `web/src/cytoscape-elk.d.ts` with the ambient declaration; removed it from `app.d.ts` (which keeps only the pre-existing `App` namespace and the new `Window.__codegraphFileGraphMetrics` augmentation).
- **Verification:** `pnpm check` — 0 errors, 0 warnings.
- **Files modified:** `web/src/app.d.ts` (removed block), `web/src/cytoscape-elk.d.ts` (new).
- **Committed in:** `d963e14b` (Task 2 GREEN commit).

**2. [Rule 1 - Bug] The initial ambient-declaration text (importing `cytoscape`'s real type) broke the single-importer property**
- **Found during:** Task 3, re-running the `rg -l "from 'cytoscape" web/src | wc -l` acceptance check.
- **Issue:** The ambient declaration originally wrote `import type cytoscape from 'cytoscape'; const register: (cy: typeof cytoscape) => void;` — this made `web/src/cytoscape-elk.d.ts` itself an importer of the `cytoscape` package specifier, so the count was 2, not the required exactly-1 (GRF-05's single-importer seam property).
- **Fix:** Retyped the registration function's parameter as `any` instead of the real cytoscape type, with a comment explaining why (importing the real type would break the seam property this file exists to protect).
- **Verification:** `rg -l "from 'cytoscape" web/src | wc -l` reports exactly 1 (`GraphCanvas.svelte`).
- **Files modified:** `web/src/cytoscape-elk.d.ts`.
- **Committed in:** `42ffa5a4` (Task 3 commit).

**3. [Rule 1 - Bug] Planning vocabulary in source comments, not just rendered copy**
- **Found during:** Task 3, running the plan's own `rg -in 'phase 5|phase-5|GRF-0|ENG-03|05-0' web/src/routes/graph/ web/src/lib/components/graph/` acceptance check.
- **Issue:** The check reported 19 matches, all inside doc comments (not rendered strings) referencing plan/requirement identifiers like "05-03 Task 2", "GRF-05", "ENG-03" — a normal convention used freely elsewhere in this codebase's comments (e.g. `browse-url.ts`, `DataTable.svelte`), but the acceptance criterion for THESE two specific directories is scoped stricter than the codebase norm, precisely because this is the exact route that shipped a leaked planning-vocabulary defect in Phase 4.
- **Fix:** Rewrote every doc comment in `file-graph-transform.ts`, `graph-style.ts`, `GraphCanvas.svelte`, and `+page.svelte` to convey the same rationale without the banned substrings (e.g. "GRF-05's seam" → "the renderer swap seam"; "05-01 locked" → "locked ahead of this work"). `D-06` (not in the banned pattern) was left as-is.
- **Verification:** the same grep now reports 0, positive-controlled by `rg -ic 'graph' web/src/routes/graph/+page.svelte` reporting 24.
- **Files modified:** all four files above.
- **Committed in:** `42ffa5a4` (Task 3 commit).

**4. [Rule 1 - Bug] Cytoscape boolean data selectors used invalid equality syntax**
- **Found during:** Task 3's mandatory live browser check — NOT caught by any of the 16 automated tests (both test suites mock or run cytoscape headless, neither of which surfaces an invalid-selector console warning the same way a real, styled render does).
- **Issue:** `graph-style.ts` used `node[isDirectory = true]` / `node[isDirectory = false]` — cytoscape logs "The selector ... is invalid" for this and the rule silently never matches, so directory/file styling was never actually applied (every node rendered under cytoscape's default style).
- **Fix:** Changed to `node[?isDirectory]` (truthy) / `node[!isDirectory]` (falsy) — cytoscape's documented existence-selector syntax for boolean fields.
- **Verification:** rebuilt, restarted the server, cleared the console, reloaded — zero warnings, and a screenshot confirms the intended directory/file visual distinction now renders.
- **Files modified:** `web/src/lib/components/graph/graph-style.ts`.
- **Committed in:** `42ffa5a4` (Task 3 commit).

**5. [Rule 1 - Bug] Committed `web/build` drift marker was stale relative to the actually-tracked source tree**
- **Found during:** Post-Task-3, re-running the plan's own `task web:drift` acceptance check.
- **Issue:** `task web:build` computes its source-file count via `git ls-files`, which only sees TRACKED files. `GraphCanvas.svelte` and `cytoscape-elk.d.ts` were still untracked (not yet `git add`ed) the second time `task web:build` ran (after the live-bug fix, before the Task 3 commit) — so the committed manifest recorded 105 source files instead of the true 107 once both were tracked in the commit. `task web:drift` caught the mismatch immediately and loudly, exactly as designed.
- **Fix:** Rebuilt `web/build` after the commit that tracks both files, in a dedicated follow-up commit.
- **Verification:** `task web:drift` — both halves MATCH (107 source files, 32 output files).
- **Files modified:** `web/build/**` (content-hash filename churn only, per Vite's documented non-determinism — vitejs/vite#15555).
- **Committed in:** `663fc303`.

---

**Total deviations:** 5 auto-fixed (all Rule 1 — bugs, none architectural). **Impact on plan:** all five are correctness fixes found through the plan's own verification discipline (acceptance-criteria greps and a mandatory live browser session), not scope creep. No threshold, bound, or acceptance criterion was weakened to reach green — each fix made a previously-failing or previously-undetected check genuinely pass.

## Issues Encountered

- **Pan/zoom interactivity not conclusively verified.** See finding 6 under "Live Browser Verification Findings" above. Flagged as `human_judgment: true` / `status: unknown` in the D5 coverage entry rather than silently assumed passing — this is a genuine gap for a human (or a future automated-UI pass with a different interaction primitive) to close, not a defect known to exist.
- No other issues.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The renderer composition works end-to-end against a real repository's live index: `/graph` renders directory-grouped, hierarchically laid-out files with real dependency edges, no CSP violation, no cytoscape console error.
- The measurement seam (`window.__codegraphFileGraphMetrics` + the matching event) is proven live, not just unit-tested — 05-04 has a working instrument to measure against at guava scale.
- `GraphCanvas.svelte`'s single-importer property, the elk/layered-only layout property, and the no-worker property are all machine-asserted and will regress loudly if violated.
- **D-10's authorized exception is discharged as designed:** this plan built ahead of GRF-01's verdict, and the composition proved out — but GRF-01's actual pass/fail against the locked threshold, at guava scale, remains 05-04's job entirely. Nothing here pre-empts that measurement.
- No blockers for 05-04. A human should independently confirm pan/zoom smoothness on a real trackpad/mouse before 05-05/05-06/05-07 add interaction-heavy features (cycle highlighting, in-place expansion) on top of this seam.

## Self-Check: PASSED

- `test -f web/src/lib/components/graph/file-graph-transform.ts` → FOUND
- `test -f web/src/lib/components/graph/graph-style.ts` → FOUND
- `test -f web/src/lib/components/graph/GraphCanvas.svelte` → FOUND
- `test -f web/src/cytoscape-elk.d.ts` → FOUND
- `test -f web/tests/file-graph-transform.test.ts` → FOUND
- `test -f web/tests/graph-tracer.test.ts` → FOUND
- `git log --oneline --all | grep -q b3539f9b` → FOUND
- `git log --oneline --all | grep -q d963e14b` → FOUND
- `git log --oneline --all | grep -q 42ffa5a4` → FOUND
- `git log --oneline --all | grep -q 663fc303` → FOUND
- `GOTOOLCHAIN=go1.26.5 task web:test` → PASS (284/284)
- `cd web && pnpm check` → PASS (0 errors, 0 warnings)
- `GOTOOLCHAIN=go1.26.5 task web:drift` → PASS (107 source / 32 output, both halves match)
- `GOTOOLCHAIN=go1.26.5 task web:audit / web:lockfile / web:deps:strict` → all PASS
- `rg -l "from 'cytoscape" web/src | wc -l` → 1

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-30*
