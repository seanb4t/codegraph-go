---
phase: 11-graph-view-community-clustering
plan: 04
subsystem: ui
tags: [svelte, cytoscape, vitest, taskfile, community-detection, colour-palette, static-scan]

requires:
  - phase: 11-graph-view-community-clustering
    provides: "11-01: FileGraphNode.communityId / FileGraphResponse.communityCount on the wire; 11-02: GRF-09 verdict PASS (median 106ms, fresh-per-call compute confirmed affordable); 11-03: GRF-10 check:gonum supply-chain gate"
provides:
  - "web/src/lib/components/graph/community-palette.ts — COMMUNITY_PALETTE (12 literal colour-blind-safe hex values), COMMUNITY_CLASS_PREFIX, communityPaletteIndex, communityDiscriminatorClass"
  - "file-graph-transform.ts: FileGraphNodeData.communityId copied from the wire (?? 0 guard), graph-community-N class on file nodes only"
  - "graph-style.ts: 12 static node[!isDirectory].graph-community-{0..11} rules generated from the palette, composing with cycle borders, directories left neutral (D-11)"
  - "+page.svelte: graph-community-summary toolbar line reading communityCount verbatim (D-12a)"
  - "web/scripts/check-no-force-layout.mjs + Taskfile check:no-force-layout — positive-controlled, self-tested proof that no force-directed layout is reachable from web/src or web/package.json (D-12b)"
affects: [11-05-mutation-log]

actuals:
  tokens: 9202
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Per-id discriminator class -> static per-index Cytoscape style rule set (mirrors cycleDiscriminatorClass/.graph-cycle-N precedent), never a data()-driven runtime colour function"
    - "Layout-name-position-scoped regex scan (name: '<x>' literal / cytoscape-<x> specifier) with a required elk positive control, rather than a bare-word forbidden-string search — this is what keeps LIVE_BACKOFF_JITTER_SPREAD and \"never force-directed\" prose from false-positiving"
    - "Same scan function reused for the real-tree walk and the --self-test injected source, so the self-test cannot silently diverge from the real enforcement path"

key-files:
  created:
    - web/src/lib/components/graph/community-palette.ts
    - web/scripts/check-no-force-layout.mjs
  modified:
    - web/src/lib/components/graph/file-graph-transform.ts
    - web/src/lib/components/graph/graph-style.ts
    - web/src/routes/graph/+page.svelte
    - web/tests/file-graph-transform.test.ts
    - web/tests/graph-communities.test.ts (new file, extended across Tasks 1 and 2)
    - Taskfile.yml
    - web/build/** (rebuilt)

key-decisions:
  - "The RED gate's literal `rg -o -i 'failed|Cannot find|does not provide an export' | wc -l -ge 1` and the GREEN gate's `rg -o -i '\\bfailed\\b' | wc -l = 0` both trip on this repo's own pre-existing test names that legitimately contain the word \"failed\" (e.g. graph-measure.test.ts's `failedObservation`, health-page.test.ts's \"distinct failed rows\") — the true signal (`Test Files N passed (N)`, `Tests N passed (N)`, zero ✗/FAIL markers) was used instead; documented as a verification-arithmetic deviation, consistent with 11-01/11-02's precedent for this exact class of literal-regex imprecision"
  - "community-palette.ts's COMMUNITY_PALETTE array is written on a single line (prettier-ignore) rather than one entry per line, because the plan's own acceptance check is a single-line rg pattern that cannot match across a multiline array literal"

requirements-completed: [GRF-06]

coverage:
  - id: D1
    description: "File nodes are coloured by communityId from a 12-hue deterministic palette on the UNCHANGED ELK layout; directory compounds stay neutral; distinct colours proven to equal distinct community ids with cycling beyond 12"
    requirement: "GRF-06"
    verification:
      - kind: unit
        ref: "web/tests/graph-communities.test.ts#community palette and stylesheet (Task 1) (3 tests)"
        status: pass
      - kind: unit
        ref: "web/tests/graph-communities.test.ts#colour == community (Task 1, D-12c)"
        status: pass
      - kind: unit
        ref: "web/tests/file-graph-transform.test.ts#file-graph-transform: community information is copied, never invented (5 tests)"
        status: pass
      - kind: other
        ref: "git diff --quiet <threshold commit> HEAD -- GraphCanvas.svelte (byte-identical, exactly one name: 'elk')"
        status: pass
    human_judgment: false
  - id: D2
    description: "The graph toolbar shows a 'N communities' line fed by the wire communityCount, without recounting"
    requirement: "GRF-06"
    verification:
      - kind: integration
        ref: "web/tests/graph-communities.test.ts#route: community count line (Task 2) (4 tests: count 3, singular 1, zero, wire-number-not-recount)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A positive-controlled static scan proves no force-directed layout is reachable from any code path under web/src or web/package.json; self-tested against an injected planted match and a real planted-file RED rehearsal"
    requirement: "GRF-06"
    verification:
      - kind: other
        ref: "node web/scripts/check-no-force-layout.mjs --self-test (PASS, injected match detected); node web/scripts/check-no-force-layout.mjs (PASS, 106 files, elk layout refs 2, elk import refs 3, forbidden 0); planted zz-planted-force-layout.ts rehearsal turned the scan RED (exit 1, forbidden matches 1) then removed; task -s check:no-force-layout (PASS)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Opening the graph view on a real repository visually confirms distinct community colours on the same layered layout, the toolbar line, and the cycle border still reading on a coloured cycle node"
    requirement: "GRF-06"
    verification: []
    human_judgment: true
    rationale: "Task 2's <human-check> is an end-of-phase visual verification (workflow.human_verify_mode=end-of-phase, the repo default) — deferred to the phase-level UAT/verify-work pass rather than executed by this autonomous plan run, per the executor's checkpoint protocol for automated-only vs human-observable verify items."

duration: 42min
completed: 2026-09-13
status: complete
---

# Phase 11 Plan 04: GRF-06 UI — Community Colouring, Toolbar Count, and the Force-Layout Scan Summary

**File nodes on the graph view are now coloured by their server-computed `communityId` from a 12-hue colour-blind-safe palette on the unchanged ELK layered layout, a "Files fall into N communities" toolbar line reads the wire `communityCount` verbatim, and a self-tested, positive-controlled static scan proves no force-directed layout is reachable from any code path.**

## Performance

- **Duration:** 42 min
- **Started:** 2026-09-13T13:15:00Z (approx, see git log)
- **Completed:** 2026-09-13T17:26:27Z
- **Tasks:** 3
- **Files modified:** 8 (excluding rebuilt `web/build/**`)

## Accomplishments

- `community-palette.ts`: `COMMUNITY_PALETTE` — the exact 12 literal hex values validated with the dataviz skill's `validate_palette.js --mode light` at plan time (adjacent-pair CVD worst ΔE 9.1, normal-vision worst 19.6, both above floor) — `communityPaletteIndex` (`(id-1) % 12`, cycling), `communityDiscriminatorClass`.
- `file-graph-transform.ts`: `FileGraphNodeData.communityId` copied from the wire with a load-bearing `?? 0` guard (every pre-existing fixture omits the field and must keep rendering unchanged); `fileNodeElement` attaches `graph-community-N` alongside any cycle classes; `expandedDirElement`/`collapsedDirElement` untouched — directory compounds never gain the field or class (D-11).
- `graph-style.ts`: 12 static `node[!isDirectory].graph-community-{0..11}` rules generated from the palette, placed after the file base rule and before the cycle rules so cycle borders still compose on a coloured node.
- `+page.svelte`: a `graph-community-summary` paragraph sibling to `graph-cycle-summary`, reading `graphState.response.communityCount` verbatim — singular "1 community", plural "N communities", "No communities computed" at zero; no new rpc.
- `web/scripts/check-no-force-layout.mjs` + `Taskfile.yml`'s `check:no-force-layout`: a browser-free static scan over `web/src` (.ts/.js/.svelte/.mjs) plus `web/package.json` dependencies, scoped to layout-name positions (`name: '<x>'` literals, `cytoscape-<x>` specifiers), with an `elk` positive control and a `--self-test` that injects `name: 'cose'` in-memory and only passes if the scan detects exactly that match.
- `GraphCanvas.svelte` confirmed byte-identical to the GRF-09 threshold commit throughout — the layout algorithm was never touched.

## Task Commits

Each task was committed atomically (Task 1 is `tdd="true"`, split into a RED test commit and a GREEN implementation commit per the tdd.md commit-scope contract):

1. **Task 1a: RED — failing tests for palette/transform/stylesheet** — `d8f758df` (test)
2. **Task 1b: GREEN — palette, communityId passthrough, static stylesheet rules** — `9d4918d8` (feat)
3. **Task 2: "N communities" toolbar line + route-level test + rebuilt SPA** — `9e7c6943` (feat)
4. **Task 3: check-no-force-layout.mjs + Taskfile target** — `f10de62d` (feat)

_plan_head_before: `e0f0749d60d325f489c87bf3aeebf421f32352af`_ (4 commits total this plan)

No separate plan-metadata commit beyond this SUMMARY's own commit (per orchestrator instruction: STATE.md/ROADMAP.md are NOT updated by this plan — the orchestrator owns those writes).

## RED/GREEN Transcript (Task 1)

RED (before `community-palette.ts` exists / classes absent):
```
FAIL  tests/graph-communities.test.ts > community palette and stylesheet (Task 1) > ...
Cannot find package '$lib/components/graph/community-palette' imported from ...
FAIL  tests/graph-communities.test.ts > colour == community (Task 1, D-12c) > ...
Error: no community class on d/f01.go (classes: "")
Test Files  2 failed | 47 passed (49)
     Tests  6 failed | 573 passed (579)
```

GREEN (`pnpm -C web test -- file-graph graph-communities`, then the whole suite):
```
distinct colours: 12 over 12 distinct ids
Test Files  49 passed (49)
     Tests  579 passed (579)
```

Full-suite regression check (`pnpm -C web test`), no changes since Task 1's GREEN: `Test Files 49 passed (49)`, `Tests 579 passed (579)`.

## Task 2 transcripts

Route-level test run (`pnpm -C web test -- graph-communities graph-cycles`): one unrelated pre-existing flaky timeout in `browse-page.test.ts` (a 15000ms timeout on an unrelated test, reproduced clean on immediate retry — not touched by this plan); retry: `Test Files 49 passed (49)`, `Tests 583 passed (583)`.

`pnpm -C web check`:
```
COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS
```

`task web:build && task -s web:drift`:
```
web:drift: hashed 116 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (116 files, 5df163a2d14415cb827de27033522e8a4facf9f27d9134656c167a633eb7a1e1)
web:drift: output half MATCH (32 files, b7b124b3acb2e709c37ba2e5b7eafcb1883bc7c615a984317b781eb17a037ba6)
web:drift: PASS — hashed 116 source files, manifested 32 output files, committed web/build/ matches both digests
```

## Task 3 transcripts

Self-test:
```
check-no-force-layout self-test: PASS — injected 'cose' detected at <self-test>/injected.ts:1
```

Clean scan:
```
{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"verdict":"PASS"}
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 0; verdict PASS
```

Planted-file RED rehearsal (`web/src/lib/zz-planted-force-layout.ts` containing `const layout = { name: 'cose' };`, then removed):
```
{"filesScanned":107,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[{"file":"web/src/lib/zz-planted-force-layout.ts","line":1,"match":"name: 'cose'"}],"verdict":"FAIL"}
check-no-force-layout: scanned 107 files; elk layout refs 2; elk import refs 3; forbidden matches 1; verdict FAIL
  forbidden: web/src/lib/zz-planted-force-layout.ts:1 — name: 'cose'
```
Exit code 1. `git status --porcelain -- web/src` confirmed empty immediately after removal.

`task -s check:no-force-layout` (self-test + real scan, both green): exit 0.

## Files Created/Modified

- `web/src/lib/components/graph/community-palette.ts` — the 12-hue palette and its indexing/class helpers
- `web/src/lib/components/graph/file-graph-transform.ts` — `communityId` field + class attachment on file nodes only
- `web/src/lib/components/graph/graph-style.ts` — 12 static per-community colour rules
- `web/src/routes/graph/+page.svelte` — `graph-community-summary` toolbar paragraph
- `web/tests/file-graph-transform.test.ts` — community passthrough + directory-neutral tests (D-11)
- `web/tests/graph-communities.test.ts` (new) — palette/stylesheet, colour==community, and route-level count-line tests
- `web/scripts/check-no-force-layout.mjs` (new) — the positive-controlled static scan
- `Taskfile.yml` — `check:no-force-layout` target
- `web/build/**` — rebuilt, drift-clean

## Decisions Made

- See `key-decisions` in frontmatter: the RED/GREEN literal-regex "failed" false-positive on pre-existing test names, and the single-line palette array format required by the plan's own literal acceptance check.
- Task 2's `<human-check>` (visually confirming colours/toolbar line/cycle border in a real `codegraph ui` session) was deferred to end-of-phase review per `workflow.human_verify_mode=end-of-phase` (this repo's default) — no `codegraph ui` process was spawned by this autonomous plan run, consistent with the landmine "never leave a `codegraph ui` process running."

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan verification arithmetic bug] The RED-gate and full-suite GREEN-gate regex checks for the literal word "failed" false-positive on pre-existing test names**
- **Found during:** Task 1's RED gate and Task 2's route-level test run
- **Issue:** The plan's `<verify>` blocks assert `rg -o -i '\bfailed\b' ... | wc -l = 0` as a GREEN signal and `rg -o -i 'failed|Cannot find|...' | wc -l -ge 1` as a RED signal. This repository's own pre-existing test suite legitimately names tests with the word "failed" (e.g. `graph-measure.test.ts`'s `failedObservation` describe, `health-page.test.ts`'s "distinct failed rows", `source-pane.test.ts`'s "Copy failed" assertions) — none of these are actual test failures, but the case-insensitive word-boundary regex matches their names regardless of PASS/FAIL status.
- **Fix:** Verified the actual intent (zero real failures) via the unambiguous signals `Test Files N passed (N)`, `Tests N passed (N)`, and the absence of any `✗`/`FAIL ` marker — all of which were confirmed clean at every gate. No test or implementation code was changed for this; it is a plan-authored verification imprecision, not a defect in the shipped tests, identical in kind to the deviations 11-01 and 11-02 already documented for this same class of literal-regex miscount.
- **Files modified:** None
- **Verification:** `rg -n '✗|FAIL |Test Files.*failed|Tests.*[0-9]+ failed'` returns zero real failure markers at every gate; `Test Files 49 passed (49)` / `Tests 579 passed (579)` (Task 1), `Test Files 49 passed (49)` / `Tests 583 passed (583)` (Task 2, on retry after one unrelated flake)
- **Committed in:** N/A (no code change)

**2. [Rule 1 - Plan verification arithmetic bug] The palette-literal acceptance regex requires a single-line match, incompatible with a one-entry-per-line array**
- **Found during:** Task 1, first verification pass after implementation
- **Issue:** The plan's own acceptance/verify command is a single-line `rg -o` pattern matching all 12 hex values separated by `, *`. `rg` does not match across newlines by default, so writing `COMMUNITY_PALETTE` with one entry per line (the initial, more readable formatting) failed this literal check even though the array's contents were exactly correct.
- **Fix:** Reformatted `COMMUNITY_PALETTE` onto a single line (with a `// prettier-ignore` comment so the formatter does not re-wrap it), satisfying the plan's literal regex without changing any value.
- **Files modified:** `web/src/lib/components/graph/community-palette.ts`
- **Verification:** `rg -o "'#2a78d6', *'#eb6834', *...'#86a800'" web/src/lib/components/graph/community-palette.ts | wc -l` = 1
- **Committed in:** `9d4918d8` (part of Task 1's GREEN commit)

**3. [Rule 1 - Pre-existing flaky test, unrelated to this plan's scope] `browse-page.test.ts` timed out once on an unrelated CR-01 test**
- **Found during:** Task 2's route-level test run
- **Issue:** `pnpm -C web test -- graph-communities graph-cycles` (which, per this repo's vitest config, runs the full 49-file suite regardless of the filter arguments — consistent with Task 1's identical observation) hit a 15000ms timeout on `browse-page.test.ts`'s "typing in search does not tear down the open node view (CR-01)" test, a file this plan never touches.
- **Fix:** None applied — out of scope per the executor's scope-boundary rule (only auto-fix issues directly caused by the current task's changes). Re-ran the identical command immediately: `Test Files 49 passed (49)`, `Tests 583 passed (583)`, confirming the timeout was transient and unrelated to this plan's changes.
- **Files modified:** None
- **Verification:** Immediate retry green, 583/583
- **Committed in:** N/A (no code change; logged here per scope-boundary guidance rather than added to `deferred-items.md` since it self-resolved on retry)

---

**Total deviations:** 3 (1 genuine code fix required for the plan's own literal gate to pass, 2 documented verification-only/environmental findings). **Impact on plan:** No scope creep. The palette values, transform logic, stylesheet rules, toolbar text, and scan mechanism are all exactly as the plan specified; only a formatting detail (single-line array) and two literal-regex/flake false signals needed acknowledgment rather than code changes.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

GRF-06 is fully wired end-to-end: the wire fields (11-01), the measured-affordable fresh-compute path (11-02, GRF-09 PASS), the supply-chain gate (11-03, GRF-10 PASS), and now the UI colouring, toolbar count, and force-layout scan (11-04) are all in place. `web/scripts/check-no-force-layout.mjs`'s real-scan and self-test transcripts above give 11-05's mutation-log family (c) its exact instrument (plant a force-layout name in `web/src`) and expected RED text, without requiring any new engineering — the planted-file rehearsal in Task 3 already demonstrated it live. The Task 2 `<human-check>` (visual confirmation of colours/toolbar/cycle-border composition in a real `codegraph ui` session) remains for the phase's end-of-phase human-verify pass.

No blockers.

---
*Phase: 11-graph-view-community-clustering*
*Completed: 2026-09-13*

## Self-Check: PASSED

All key files confirmed present on disk (`web/src/lib/components/graph/community-palette.ts`, `web/src/lib/components/graph/file-graph-transform.ts`, `web/src/lib/components/graph/graph-style.ts`, `web/src/routes/graph/+page.svelte`, `web/tests/file-graph-transform.test.ts`, `web/tests/graph-communities.test.ts`, `web/scripts/check-no-force-layout.mjs`, `Taskfile.yml`). All 4 task commits confirmed present in `git log` (`d8f758df`, `9d4918d8`, `9e7c6943`, `f10de62d`). `pnpm -C web test`, `pnpm -C web check`, `task web:build && task -s web:drift`, and `task -s check:no-force-layout` all re-confirmed green at SUMMARY time. `git status --short` confirmed clean.
