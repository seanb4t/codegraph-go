---
phase: 04-query-workbench-index-health
plan: 01
subsystem: ui
tags: [svelte, tanstack-table, sveltekit, connectrpc, typescript, shadcn-svelte, tdd]

requires:
  - phase: 03-browse-inspect-navigation (plan 06)
    provides: "components.json / shadcn-svelte@1.5.1 vendoring precedent, the dlx-only rule (02-05), the [SUS]/too-new false-positive precedent for @testing-library/jest-dom that this plan's Task 1 checkpoint reused"
  - phase: 03-browse-inspect-navigation (plan 09)
    provides: "status.ts's classifyStatus/navigationIdentity/createStatusGate and its VIEW_LOCAL_PARAMS exclusion pattern, extended here with a route-scoped table rather than a second normalizer"
  - phase: 03-browse-inspect-navigation (plan 04)
    provides: "rpc-errors.ts's classifyRpcError, composed (never re-implemented) by workbench-failure.ts"
provides:
  - "web/src/lib/workbench-url.ts — the complete Workbench URL grammar (mode/symbol/file[]/depth/limit), sibling to browse-url.ts, importing its shape-integer primitives; getAll('file') for the multi-valued key (D-11)"
  - "web/src/lib/workbench-failure.ts — describeWorkbenchFailure, the four-kind failure taxonomy composed over classifyRpcError + IndexStatus (WRK-04 criterion 3)"
  - "web/src/lib/components/workbench/{table-features.ts,DataTable.svelte,callers-columns.ts} — the one sorting registration and the one generic (TRow extends RowData) table shell every Workbench analysis reuses (D-06), plus the first *-columns.ts (Callers)"
  - "web/src/lib/components/ui/table/ — 9 shadcn-svelte@1.5.1 vendored files, human-reviewed with a positive-controlled scan"
  - "web/src/lib/status.ts's ROUTE_LOCAL_PARAMS table — the route-scoped counterpart to VIEW_LOCAL_PARAMS, closing T-04-32 (Workbench control changes no longer refire GetStatus)"
  - "/workbench's Callers tab, wired end to end against the real Callers rpc"
affects: [04-04, 04-05, 04-06, 04-07]

actuals:
  tokens: 16402
  tasks: 3
  commits: 2

tech-stack:
  added:
    - "@tanstack/svelte-table@9.2.4 (exact-pinned devDependency) — the real Svelte 5 rune API (createTable, tableFeatures, FlexRender, table.atoms.<slice>.get()), NOT the store-based API the public docs site still describes"
    - "shadcn-svelte@1.5.1 table registry (9 files, dlx-only, package.json untouched)"
  patterns:
    - "Svelte 5 generic components (<script lang=\"ts\" generics=\"TRow extends RowData\">) for a table shell shared across row shapes that don't share a common interface beyond RowData — row identity passed as a getRowId prop, never hard-coded in the shell."
    - "createTable's columns/data/getRowId options must be passed as getters (get columns() { return columns }, not a bare columns shorthand) or svelte-check emits state_referenced_locally warnings — the table would silently stop reacting to prop changes."
    - "A route-scoped exclusion table (ROUTE_LOCAL_PARAMS: Record<pathname, string[]>) keyed by pathname, consumed inside the SAME navigationIdentity function alongside the existing global VIEW_LOCAL_PARAMS list — the right shape when a parameter is view-local under ONE route but a real distinct-view selector under another (the same param name meaning two different things depending on pathname)."
    - "Dynamically re-importing the same Svelte component per test with vi.resetModules() between imports is BROKEN — it loads a second copy of the svelte runtime module, producing '$effect can only be used inside an effect' on every import after the first. Import the component ONCE at module scope; swap a mutable dispatcher inside a stable mocked client instead of re-mocking per test."

key-files:
  created:
    - web/src/lib/workbench-url.ts
    - web/src/lib/workbench-failure.ts
    - web/src/lib/components/workbench/table-features.ts
    - web/src/lib/components/workbench/DataTable.svelte
    - web/src/lib/components/workbench/callers-columns.ts
    - web/src/lib/components/ui/table/ (9 files)
    - web/tests/workbench-tracer.test.ts
    - web/tests/workbench-url.test.ts
    - web/tests/workbench-failure.test.ts
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/src/lib/browse-url.ts (parseShapeInteger export only — no logic change)
    - web/src/lib/status.ts (ROUTE_LOCAL_PARAMS added; VIEW_LOCAL_PARAMS unchanged)
    - web/src/routes/workbench/+page.svelte (D-18 placeholder filled — Callers tab only)
    - web/tests/status.test.ts (route-scoping coverage extended)
    - web/tests/support/browse-page-state.svelte.ts (comment widened — reused, not duplicated)

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human): maintainer approved both artifacts. Verbatim maintainer answer: \"Approve both\". Both [SUS]/too-new verdicts confirmed as the same false-positive shape 03-01 already approved for @testing-library/jest-dom (the signal reads the LATEST version's publish timestamp, not package age). @tanstack/svelte-table@9.2.4: created 2022-05-06, 58,918 downloads/week, github.com/TanStack/table. shadcn-svelte@1.5.1: created 2023-05-26, 108,654 downloads/week, github.com/huntabyte/shadcn-svelte, already in use since 03-06. The orchestrator independently re-verified this evidence against live npm before and after the maintainer's decision; nothing exceeded the approved enumeration (git ls-files confirmed exactly the 9 named files, no shadcn-svelte entry landed in package.json)."
  - "Tracer feedback gate (execution_flow, interactive run — workflow.auto_advance/_auto_chain_active both false): stopped cleanly after committing Task 2, before Task 3, and returned a checkpoint:human-verify presenting the tracer's own automated <verify> result plus pnpm check plus the full web:test/web:lockfile/web:deps:strict/web:audit suite. The coordinator independently re-ran every claim rather than taking the report on trust; all held."
  - "DataTable's row-type generic requires an explicit `TRow extends RowData` constraint (RowData = Record<string,any> | Array<any>, from @tanstack/table-core) — a bare `generics=\"TRow\"` fails pnpm check with 5 TypeScript errors because createTable's TData parameter is itself constrained to RowData. Constraining the generic, not widening it to `any`, keeps the shell genuinely type-safe per-analysis while staying open to any row shape."
  - "status.ts's ROUTE_LOCAL_PARAMS is scoped to '/workbench' only, deliberately NOT added to the global VIEW_LOCAL_PARAMS list — symbol/file/depth/limit are ALSO Browse parameters where each one legitimately selects a distinct Browse view; a global exclusion would have silently stopped refreshing the index verdict as a developer moves through Browse, an invisible regression to a shipped Phase-3 contract. Verified in both directions: web/tests/status.test.ts's extended navigationIdentity block asserts the Workbench exclusion AND the unchanged Browse behavior in the same describe block."
  - "workbench-failure.ts's 'not-found' + IndexStatus.verdict === 'no-index' -> 'index-stale' mapping: a missing index, not a missing symbol, is the real cause when there is no index to have found the symbol in — asserted in both directions (verdict 'ok' still yields plain 'not-found') so the composition rule can't pass by always returning the same kind."

patterns-established:
  - "The tracer feedback gate (a mandatory post-tracer human-verify stop in interactive runs, distinct from any plan-authored checkpoint task) worked as designed: it caught nothing wrong here, but it forced an independent re-verification pass before Task 3 committed to the proven slice, and the audit trail (exact commands, exact observed counts) is now in this SUMMARY rather than only in a transient chat turn."

requirements-completed: [WRK-03, WRK-04]

coverage:
  - id: D1
    description: "Opening /workbench?mode=callers&symbol=NAME&limit=N runs the real Callers rpc against a stub client and renders its Location rows in a real HTML table"
    requirement: WRK-03
    verification:
      - kind: unit
        ref: "web/tests/workbench-tracer.test.ts#'parseWorkbenchParams -> Callers rpc -> DataTable renders all three rows, recording exactly one call with the URL symbol and limit'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Clicking the name column header sorts the rendered rows ascending; clicking again reverses to descending (WRK-04) via @tanstack/svelte-table's real client-side sort engine, not a hand-rolled comparator"
    requirement: WRK-04
    verification:
      - kind: unit
        ref: "web/tests/workbench-tracer.test.ts#'WRK-04: clicking the name column header sorts ascending, clicking again reverses to descending'"
        status: pass
    human_judgment: false
  - id: D3
    description: "A limit value typed by the developer reaches CallersRequest.limit unchanged, even above the server's MaxLimit — no client-side bound is applied anywhere in web/ (D-07)"
    verification:
      - kind: unit
        ref: "web/tests/workbench-tracer.test.ts#'D-07: a limit above MaxLimit (99999) reaches the stub client unchanged'"
        status: pass
      - kind: unit
        ref: "web/tests/workbench-url.test.ts#'an out-of-range but well-shaped integer parses to that exact number' and #'a NEGATIVE but well-shaped integer parses to that exact negative number and survives a full round-trip'"
        status: pass
      - kind: other
        ref: "rg -n 'MaxLimit|MaxDepth|> ?1000|> ?50' web/src/lib/workbench-url.ts web/src/routes/workbench/+page.svelte — 0 matches"
        status: pass
    human_judgment: false
  - id: D4
    description: "A failed Callers query renders one of four pairwise-distinct named failure kinds — never a single catch-all message (WRK-04 criterion 3)"
    requirement: WRK-04
    verification:
      - kind: unit
        ref: "web/tests/workbench-tracer.test.ts#'WRK-04 criterion 3: a NotFound rejection renders the named not-found failure state'"
        status: pass
      - kind: unit
        ref: "web/tests/workbench-failure.test.ts (8 cases: all four kinds reachable, four titles pairwise distinct via Set size 4, the no-index/ok composition rule in both directions, never throws on non-Error input)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A Workbench URL round-trips exactly: parse(serialize(p)) deep-equals p, including a repeated file= key (in both the parse and serialize direction) and unrecognized/repeated unknown parameters (D-12)"
    verification:
      - kind: unit
        ref: "web/tests/workbench-url.test.ts (9 cases: full-field round-trip, repeated file= both directions, repeated unknown params, out-of-range/negative/malformed integers, unknown mode)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Changing any Workbench control mints no new navigation identity — the shared status gate issues no additional GetStatus call; a Browse parameter change still does (T-04-32)"
    verification:
      - kind: unit
        ref: "web/tests/workbench-tracer.test.ts#'records exactly one getStatus call across a Workbench limit change, paired with a Browse symbol change that DOES refetch'"
        status: pass
      - kind: unit
        ref: "web/tests/status.test.ts (3 new cases in the navigationIdentity describe block: every WORKBENCH_PARAM_KEYS member route-local under /workbench; the same params still distinct under /browse; /workbench vs /graph stay distinct)"
        status: pass
    human_judgment: false
  - id: D7
    description: "DataTable is generic over its row type — the same shell type-checks for Location rows here and will for CountRow rows in 04-05, under pnpm check"
    verification:
      - kind: unit
        ref: "pnpm check — 991 files, 0 errors, 0 warnings"
        status: pass
      - kind: other
        ref: "rg -c 'generics=' web/src/lib/components/workbench/DataTable.svelte -> 1; rg -c 'Location' DataTable.svelte -> 0, positive-controlled by rg -c 'Location' callers-columns.ts -> 4"
        status: pass
    human_judgment: false
  - id: D8
    description: "The 9 vendored shadcn-svelte table files carry no raw-HTML sink, network access, filesystem access, or dynamic evaluation (T-04-01 mitigation) — the only review this unscanned supply-chain surface gets in this plan (D-09 asymmetry)"
    verification:
      - kind: other
        ref: "rg -o '{@html' table/ -> 0; rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' table/ -> 0; positive control rg -o 'class' table/ -> 33; supporting scan rg -o 'fetch(|XMLHttpRequest|eval(|new Function|require(|import(|node:fs|readFile|writeFile' table/ -> 0; all 9 files read directly"
        status: pass
    human_judgment: true
    rationale: "A grep-based scan proves the absence of the specific patterns checked, but 'this vendored third-party source is safe to trust' is ultimately a human supply-chain judgment (T-04-01's disposition is 'mitigate', not 'eliminate') — 04-07's web:components:drift byte-compare is the structural, ongoing follow-up, not a one-time automated clearance."

duration: 85min
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 1: Query Workbench Tracer Summary

**TanStack Table v9's real Svelte-5 rune API composed end to end into a working, sortable /workbench Callers table with a four-kind named failure taxonomy and a route-scoped status-gate exclusion — the phase's highest-risk unknown retired in one commit.**

## Performance

- **Duration:** 85 min (includes two checkpoint waits: Task 1 package-legitimacy approval, and the mandatory post-tracer human-verify stop before Task 3)
- **Started:** 2026-08-29T22:38:23Z
- **Completed:** 2026-08-30T00:03:46Z
- **Tasks:** 3 (1 checkpoint, 1 tracer, 1 test-lock + review)
- **Files modified:** 25

## Accomplishments
- `@tanstack/svelte-table@9.2.4`'s real rune-based API (`createTable`, `tableFeatures`, `FlexRender`) proven to compose with this repo's Svelte 5 + Vitest/jsdom toolchain — the public docs site's store-based API (`createSvelteTable`, `writable`) was confirmed absent from this version and never used.
- `/workbench?mode=callers&symbol=X&limit=N` runs the real `Callers` rpc, renders results in a sortable HTML table (click-to-sort, ascending then descending), and shows one of four pairwise-distinct named failure states on rejection.
- The complete Workbench URL grammar (`workbench-url.ts`) locked with 9 round-trip tests, including the deliberately divergent multi-valued `file=` reader (`getAll`, not `get`) in both parse and serialize directions.
- `status.ts`'s `navigationIdentity` gained a route-scoped exclusion so Workbench control changes stop refiring `GetStatus`, without touching the global `VIEW_LOCAL_PARAMS` list or Browse's own (still-correct) refetch behavior.
- 9 shadcn-svelte `table` files vendored and reviewed with a positive-controlled scan (two zero counts, one non-zero control over the same tree, plus a direct read of all 9 files) — the only review this unscanned supply-chain surface gets until 04-07's drift guard.

## Task Commits

Each task was committed atomically:

1. **Task 1: Approve the [SUS] package-legitimacy verdicts** - checkpoint, no code commit. Maintainer answer recorded above: "Approve both".
2. **Task 2: TRACER — one end-to-end path from a Workbench URL to a sorted Callers table** - `6b753435` (feat)
3. **Task 3: Review the vendored source and lock the URL and failure contracts with tests** - `2205935d` (test)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified
- `web/src/lib/workbench-url.ts` - complete Workbench URL grammar (D-10/D-11/D-12)
- `web/src/lib/workbench-failure.ts` - composed four-kind failure taxonomy over classifyRpcError + IndexStatus
- `web/src/lib/components/workbench/table-features.ts` - the one sorting registration for every analysis
- `web/src/lib/components/workbench/DataTable.svelte` - generic (`TRow extends RowData`) shared table shell
- `web/src/lib/components/workbench/callers-columns.ts` - Callers' `ColumnDef<Location>[]` + shared row-identity function
- `web/src/lib/components/ui/table/` (9 files) - shadcn-svelte@1.5.1 vendored table primitives
- `web/src/lib/browse-url.ts` - `parseShapeInteger` exported (only change)
- `web/src/lib/status.ts` - `ROUTE_LOCAL_PARAMS` route-scoped exclusion table added
- `web/src/routes/workbench/+page.svelte` - Callers tab wired; other 3 modes show a placeholder
- `web/tests/workbench-tracer.test.ts` - 6 tests, RED before implementation, GREEN after
- `web/tests/workbench-url.test.ts` - 9 tests locking the URL grammar
- `web/tests/workbench-failure.test.ts` - 8 tests locking the failure taxonomy
- `web/tests/status.test.ts` - 3 tests extending route-scoping coverage
- `web/tests/support/browse-page-state.svelte.ts` - comment widened for reuse across two test files
- `web/package.json`, `web/pnpm-lock.yaml` - `@tanstack/svelte-table@9.2.4` added (exact pin)

## Decisions Made

See `key-decisions` in the frontmatter above for the full record with rationale. Summary:
- Task 1 checkpoint: maintainer approved both `[SUS]` artifacts; verbatim answer and independent re-verification recorded.
- Tracer feedback gate: honored as a mandatory interactive-run stop, distinct from the plan's own Task 1/Task 3 boundaries; the coordinator independently re-ran every claim before authorizing Task 3.
- `DataTable`'s generic required an explicit `RowData` constraint (not `any`) to satisfy `pnpm check`.
- `ROUTE_LOCAL_PARAMS` scoped to `/workbench` only, never widened globally — verified in both directions.
- `workbench-failure.ts`'s `no-index` mapping rule verified in both directions (not just the positive case).

## Deviations from Plan

None — plan executed exactly as written. Three implementation-time issues were found and fixed during Task 2 before the GREEN commit (not deviations from the plan's specified behavior — the plan's `<action>` steps didn't specify test-infrastructure mechanics, so these are ordinary execution decisions, documented below for transparency):

### Issues Encountered

**1. `vi.resetModules()` between per-test dynamic imports of `+page.svelte` broke Svelte's runtime.**
- **Found during:** Task 2, step (k) — first GREEN attempt.
- **Symptom:** tests 3-5 (which each re-imported the page component with a fresh mock) failed with `` `$effect` can only be used inside an effect `` after tests 1-2 passed.
- **Root cause:** `vi.resetModules()` clears Vite's module cache, so the re-imported `+page.svelte` loaded a SECOND copy of the `svelte` runtime module — its `$effect` calls then ran against a runtime instance disjoint from the one `@testing-library/svelte`'s `render()` (imported once, at file top) was operating on.
- **Fix:** import `+page.svelte` exactly ONCE at module scope; each test swaps a mutable `currentCallersImpl` dispatcher inside one stable mocked `uiClient` object, rather than re-mocking `$lib/client` per test. Recorded as a `patterns-established` entry above.
- **Files modified:** `web/tests/workbench-tracer.test.ts`.
- **Verification:** all 6 tests pass with this shape; confirmed the broken shape by observing the failure first (not assumed).

**2. `DataTable`'s generic (`TRow`) needed an explicit `RowData` constraint.**
- **Found during:** Task 2, step (k) — `pnpm check` after the tracer went GREEN under vitest.
- **Symptom:** 5 TypeScript errors — `createTable`'s `TData` parameter is itself constrained to `RowData` (`Record<string,any> | Array<any>`), so a bare `generics="TRow"` (no constraint) was rejected.
- **Fix:** `generics="TRow extends RowData"`, importing `RowData` as a type from `@tanstack/svelte-table`.
- **Files modified:** `web/src/lib/components/workbench/DataTable.svelte`.
- **Verification:** `pnpm check` — 0 errors, 0 warnings.

**3. `createTable`'s `columns`/`getRowId` options needed getter syntax, not bare shorthand.**
- **Found during:** Task 2, step (k) — svelte-compiler warnings (`state_referenced_locally`) surfaced during the vitest run, before `pnpm check` was even reached.
- **Fix:** `get columns() { return columns }` / `get getRowId() { return getRowId }`, matching the `get data()` pattern already used for `rows`.
- **Files modified:** `web/src/lib/components/workbench/DataTable.svelte`.
- **Verification:** warnings gone on re-run; tests still 6/6 green.

**Impact on plan:** none of the three affected the plan's specified behavior or scope — all three are test/type-system mechanics discovered while making the plan's own Task 2 acceptance criteria pass.

## Issues Encountered

See "Issues Encountered" under Deviations above (all three overlap; documented there per the template's own convention when a discovery is mechanical rather than a scope change).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The tracer architecture (`DataTable` + `table-features.ts` + the `*-columns.ts` convention + the composed failure taxonomy) is proven and ready for 04-04 to build out the remaining three analysis tabs (Impact, Affected, Callees) without re-deriving any of it.
- 04-05's health tables can reuse `DataTable.svelte` directly — it is generic, not `Location`-typed, confirmed by `pnpm check` and the scoped-absence grep.
- `task web:drift` is EXPECTED RED from this plan onward (source half: 73→87 files, recomputed digest changed; output half: 27 files, digest MATCH — proving the committed `web/build/` bytes were not hand-edited, only left stale). This window is scheduled to close in 04-07 Task 3, exactly as 03-01 opened an equivalent window and 03-09 closed it. **This is an intended, characterized window, not a regression** — the next phase reader should not re-diagnose it.
- No blockers for 04-02 through 04-07; 04-02 (the recursive glob fix) already landed independently this wave and is untouched by this plan.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-30*

## Self-Check: PASSED

All 15 claimed files verified present on disk; both claimed commits (`6b753435`, `2205935d`) verified present in `git log`. No missing items.
