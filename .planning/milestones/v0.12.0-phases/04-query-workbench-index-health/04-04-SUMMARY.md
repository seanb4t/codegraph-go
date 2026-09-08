---
phase: 04-query-workbench-index-health
plan: 04
subsystem: ui
tags: [svelte5, tanstack-table, bits-ui, shadcn-svelte, connect-rpc, workbench]

# Dependency graph
requires:
  - phase: 04-01
    provides: "the Workbench tracer — @tanstack/svelte-table@9.2.4 installed, shadcn table vendored, workbench-url.ts/workbench-failure.ts/table-features.ts/DataTable.svelte/callers-columns.ts, the Callers tab wired end to end, status.ts's route-scoped navigationIdentity/ROUTE_LOCAL_PARAMS"
provides:
  - "shadcn-svelte tabs primitive vendored (5 files) under web/src/lib/components/ui/tabs/"
  - "AnalysisPanel.svelte — the ONE shared per-analysis wrapper (D-06), exporting AnalysisResult<TSummary> = { rows: Location[]; summary?: TSummary } from its module context"
  - "impact-columns.ts and callees-columns.ts — the second and third Location ColumnDef[] arrays"
  - "web/src/routes/workbench/+page.svelte rewritten around Tabs.Root/List/Trigger/Content with three of four analyses (Impact, Callers, Callees) live; Affected remains the explicit placeholder for 04-06"
  - "Impact's depth control (WRK-01) with an assertable no-remount, no-goto, no-extra-GetStatus property"
  - "Callers/Callees limit controls (WRK-03) with the same no-navigation property, asserted for the separate `limit` ROUTE_LOCAL_PARAMS entry"
  - "the WRK-04 criterion-3 four-way rendered failure distinctness assertion (Set-of-size-4 over rendered strings and testids)"
affects: [04-06, 04-07]

actuals:
  tokens: 13650
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "AnalysisPanel.svelte: a generic Svelte 5 component (generics=\"TSummary\") whose dispatch effect tracks ONLY a caller-supplied requestKey string, reading the run callback through untrack — mirrors the tracer's untrack(() => params) discipline so a parent re-render that leaves requestKey unchanged never re-triggers a request or an extra GetStatus call"
    - "Tab panels are gated by an {#if activeMode === '<mode>'} INSIDE each vendored Tabs.Content, not by relying on bits-ui's own always-mounted-but-hidden Content wrapper — necessary because Tabs.Content renders every value's content simultaneously (toggling only the `hidden` attribute), which would otherwise mount all three wired AnalysisPanels at once and fan every analysis's request out on every render"
    - "WRK-01's 'route not re-mounted' property is asserted via DOM node reference identity on an always-rendered heading (outside every {#if} branch), rather than an injected onMount spy — avoids adding test-only instrumentation to production code"

key-files:
  created:
    - web/src/lib/components/ui/tabs/index.ts
    - web/src/lib/components/ui/tabs/tabs.svelte
    - web/src/lib/components/ui/tabs/tabs-list.svelte
    - web/src/lib/components/ui/tabs/tabs-trigger.svelte
    - web/src/lib/components/ui/tabs/tabs-content.svelte
    - web/src/lib/components/workbench/AnalysisPanel.svelte
    - web/src/lib/components/workbench/impact-columns.ts
    - web/src/lib/components/workbench/callees-columns.ts
    - web/tests/workbench-impact.test.ts
    - web/tests/workbench-callers-callees.test.ts
  modified:
    - web/src/routes/workbench/+page.svelte

key-decisions:
  - "AnalysisPanel's dispatch effect depends ONLY on requestKey (a caller-composed JSON string), never on `run`'s own identity — reading `run` through `untrack` is what makes an unrelated parent re-render safe, and is the actual mechanism behind the must_have that a depth/limit edit issues no extra GetStatus call."
  - "Each tab's AnalysisPanel instantiation is nested inside an {#if activeMode === '<mode>'} block within its Tabs.Content, rather than relying on bits-ui's hidden-but-mounted Content wrapper alone — required for the cross-tab isolation property (switching tabs must not fan out to every analysis)."
  - "The WRK-01 no-remount assertion uses DOM node reference identity on the always-rendered <h1>Workbench</h1> heading instead of an onMount spy or module-scoped mount counter, to avoid adding test-only instrumentation to +page.svelte. A full remount is the only way that heading's node reference could change between two captures within the same render() call."
  - "Task 3's Callers/Callees limit wiring and the four-way failure taxonomy were already fully delivered by Task 2's holistic +page.svelte rewrite; Task 3 contributed a test file only, with genuine RED confirmed by temporarily reverting +page.svelte to the pre-Task-2 tracer and observing 3/8 failures, then restoring byte-identically."

patterns-established:
  - "AnalysisPanel<TSummary>: the shared per-analysis state machine (idle/loading/loaded/failed) all four Workbench tabs (04-06's Affected included) instantiate against their own columns.ts, run function, and optional summary snippet."

requirements-completed: [WRK-01, WRK-03, WRK-04]

coverage:
  - id: D1
    description: "shadcn-svelte tabs primitive vendored at pinned version 1.5.1, with a positive-controlled zero-raw-HTML review"
    verification:
      - kind: other
        ref: "ls web/src/lib/components/ui/tabs | wc -l (== 5); rg -o '\\{@html\\}' / rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' over the tree (both 0); rg -o 'class' positive control (17)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Four-tab Workbench shell (mode URL parameter, D-13) with Impact wired through the new shared AnalysisPanel and its depth control (WRK-01), including the header summary for node_count/edge_count and the no-remount/no-goto/no-extra-GetStatus properties"
    requirement: "WRK-01"
    verification:
      - kind: unit
        ref: "web/tests/workbench-impact.test.ts (10/10 passing)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Callers and Callees limit controls (WRK-03) with the same no-navigation property for the separate `limit` ROUTE_LOCAL_PARAMS entry, cross-tab isolation in both directions, D-07 unclamped limit passthrough, and column sorting on Callees"
    requirement: "WRK-03"
    verification:
      - kind: unit
        ref: "web/tests/workbench-callers-callees.test.ts (8/8 passing)"
        status: pass
    human_judgment: false
  - id: D4
    description: "WRK-04 criterion 3: four rejections through the same tab render four provably distinct failure messages (Set-of-size-4 over rendered strings and data-testids), for three of the four analyses"
    requirement: "WRK-04"
    verification:
      - kind: unit
        ref: "web/tests/workbench-callers-callees.test.ts#WRK-04 criterion 3: four rejections through the SAME tab render four rendered strings whose SET has size four, with four distinct data-testids"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-08-29
status: complete
---

# Phase 4 Plan 4: Four-Tab Workbench with Impact Depth and Callees Limit Controls Summary

**Extracted the tracer's inline Callers state machine into a shared, generic `AnalysisPanel.svelte` and instantiated it three more times behind vendored shadcn-svelte tabs — Impact with a depth control (WRK-01), Callers and Callees with a limit control (WRK-03) — all three provably free of navigation, remounts, or extra `GetStatus` calls on every control edit.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-29T20:35Z (approx.)
- **Completed:** 2026-08-30T00:53Z
- **Tasks:** 3
- **Files modified:** 11 (6 created under `web/src/lib/components/`, 5 vendored under `web/src/lib/components/ui/tabs/`, 1 route rewritten, 2 test files created)

## Accomplishments

- Vendored shadcn-svelte's `tabs` primitive at pinned `1.5.1` (5 files, zero raw-HTML directives, zero DOM raw-HTML sinks, no new npm dependency — `bits-ui`/`@internationalized/date`/`tailwind-variants` ranges already satisfied).
- Extracted the tracer's per-analysis idle/loading/loaded/failed state machine into `AnalysisPanel.svelte`, generic over a `TSummary` type parameter, exporting `AnalysisResult<TSummary> = { rows: Location[]; summary?: TSummary }` so Impact's `node_count`/`edge_count` (and 04-06's Affected echoed files) can reach a header summary without a second request or an unguarded closure side channel.
- Rewrote `/workbench` around the four-tab shell: `mode` URL parameter selects the active tab, an unrecognized mode falls back to Impact, and switching tabs preserves every other parameter already in the URL.
- Wired Impact (symbol + depth, `uiClient.impact`, `impactColumns`, header summary of node/edge counts) and finished Callers/Callees (symbol + limit, `uiClient.callers`/`callees`, `callersColumns`/`calleesColumns`) — all through the one shared `AnalysisPanel`.
- Proved, in code rather than by inspection, that moving the depth or limit control: re-runs only the affected analysis, replaces the rendered rows, never calls `goto`, never re-mounts the route (DOM node identity on the always-rendered heading), and issues no additional `GetStatus` call (driven through the real `navigationIdentity`/`createStatusGate` pair, once for `depth` and once for `limit`, since they are separate `ROUTE_LOCAL_PARAMS` entries).
- Proved cross-tab isolation in both directions: driving Callers issues zero Callees calls and vice versa — the `{#if}`-gated `AnalysisPanel` instantiation inside each `Tabs.Content` is what makes this true, since bits-ui's `Tabs.Content` keeps every tab's content in the DOM simultaneously (toggling only `hidden`).
- Proved WRK-04 criterion 3's four-way rendered failure distinctness as a `Set`-of-size-4 assertion over both rendered title strings and `data-testid` values, driven through four real rejections (`NotFound`+`ok` verdict, `NotFound`+`no-index` verdict, `InvalidArgument`, bare `Unavailable`) against the same Callers tab.

## Task Commits

1. **Task 1: Vendor the tabs primitive and review its source** - `922ca29` (feat)
2. **Task 2: The four-tab shell, the shared AnalysisPanel, and the Impact depth control** - `d2c45c6` (test, RED: 9 total/1 passed against the pre-Task-2 route), `09c42de` (feat, GREEN: 10/10)
3. **Task 3: Limit controls for Callers and Callees, and a four-way distinguishable failure assertion** - `28cba4f` (test, RED confirmed via temporary revert: 8 total/5 passed against the pre-Task-2 route, then GREEN: 8/8 with no additional application code — Task 2's holistic rewrite had already delivered Task 3's full scope)

**Plan metadata:** (this commit, docs: complete plan)

_Note: both TDD tasks show test-then-feat/test-only commit pairs per the RED/GREEN discipline; Task 3's implementation was already present from Task 2, so its commit is test-only._

## Files Created/Modified

- `web/src/lib/components/ui/tabs/index.ts`, `tabs.svelte`, `tabs-list.svelte`, `tabs-trigger.svelte`, `tabs-content.svelte` - vendored shadcn-svelte Tabs (thin wrappers over `bits-ui`'s `Tabs` primitive)
- `web/src/lib/components/workbench/AnalysisPanel.svelte` - the one shared per-analysis wrapper (D-06); owns the request lifecycle, status-gate subscription, and failure/summary/table rendering
- `web/src/lib/components/workbench/impact-columns.ts` - `impactColumns`, the second `ColumnDef<Location>[]`
- `web/src/lib/components/workbench/callees-columns.ts` - `calleesColumns`, the third `ColumnDef<Location>[]`
- `web/src/routes/workbench/+page.svelte` - rewritten around `Tabs.Root`/`List`/`Trigger`/`Content`; Impact/Callers/Callees wired through `AnalysisPanel`, Affected left as the explicit placeholder
- `web/tests/workbench-impact.test.ts` - 10 tests covering the Impact tab, its depth control, and the header summary
- `web/tests/workbench-callers-callees.test.ts` - 8 tests covering Callers/Callees limit controls, cross-tab isolation, D-07 passthrough, column sorting, and the four-way failure distinctness

## Decisions Made

- `AnalysisPanel`'s dispatch effect tracks only a caller-supplied `requestKey` string (reading `run` through `untrack`) — this is what makes an unrelated parent re-render safe from triggering an extra request or `GetStatus` call, and is the concrete mechanism behind the plan's `ROUTE_LOCAL_PARAMS`-quiescence must_have.
- Each tab's `AnalysisPanel` is nested inside an `{#if activeMode === '<mode>'}` block within its `Tabs.Content`, rather than relying on bits-ui's hidden-but-mounted content wrapper alone — required for cross-tab isolation, since `Tabs.Content` mounts every tab's content simultaneously.
- WRK-01's no-remount assertion uses DOM node reference identity on the always-rendered `<h1>Workbench</h1>` heading rather than an injected `onMount` spy or module-scoped mount counter, so no test-only instrumentation was added to production code.

## Deviations from Plan

None — plan executed as written. Two implementation notes worth recording as discretionary choices within the plan's stated bounds (not deviations from any must_have or prohibition):

1. **No-remount assertion technique.** The plan named "a module-scoped mount counter or an onMount spy" as the mechanism for WRK-01's no-remount property. This was implemented instead via DOM node reference identity on the always-rendered `<h1>` heading — a standard Testing Library technique that proves the same property (a full remount is the only way that node's reference could change between two captures in one `render()` call) without adding test-only instrumentation to `+page.svelte`. Documented in the test file's own header comment.
2. **Task 3 delivered test-only.** Task 2's holistic `+page.svelte` rewrite wired Callers, Callees, and Impact in one pass (the plan's own action text scoped Task 2 to "the four-tab shell, the shared AnalysisPanel, and the Impact depth control," but building the shared shell naturally produced all three call sites together). Task 3's application-code scope ("finishing the Callers and Callees limit controls... if Task 2 left anything incomplete") turned out to be already complete; Task 3 contributed the test file only. Verified this was not vacuous by temporarily reverting `+page.svelte` to the pre-Task-2 tracer and re-running `workbench-callers-callees.test.ts`: 8 total, 5 passed, 3 failed (the Callees-specific assertions the tracer route cannot satisfy) — confirming the test exercises real, Task-2-delivered behavior rather than passing trivially. Restored byte-identically afterward (`git status --porcelain` confirmed clean).

## Issues Encountered

- **Svelte rune-name shadowing.** `AnalysisPanel.svelte`'s first draft declared `let state = $state<PanelState<TSummary>>(...)` — naming the local variable `state` caused `svelte-check` to mis-parse every `$state(...)` call in the same script block as Svelte's legacy `$store`-auto-subscription syntax rather than the `$state` rune (a known Svelte 5 ambiguity: a local binding shadowing a rune name disables rune interpretation for that identifier in that scope). Fixed by renaming the local variable to `panelState` throughout. No other files were affected.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `AnalysisPanel.svelte` is ready for 04-06 to instantiate a fourth time for `Affected` — a `columns.ts` file plus a request function mapping `AffectedResponse` into `{ rows, summary: { files } }`, no new panel markup needed.
- `web/tests/workbench-impact.test.ts` and `workbench-callers-callees.test.ts` together bring the JS suite from 176 (end of 04-01) to 194 (`task web:test`).
- `task web:drift` remains EXPECTED RED (source half: 73 marker vs. 95 recomputed files — grew further this plan, as expected; output half still matches) — the window 04-01 opened stays open for 04-07 Task 3 to close, per the plan's own `<verification>` block.
- The intentional Affected-tab placeholder (`"This analysis is not yet wired."`) is recorded in `.planning/WINDOWS.md` (id 24, kind `stub`, phase 04) as an open entry that 04-06 resolves — not a defect, but tracked per the broken-windows ledger convention.

## Self-Check: PASSED

All 11 created/modified files verified present on disk; all 4 task commits (`922ca29`, `d2c45c6`, `09c42de`, `28cba4f`) verified present in git history.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-29*
