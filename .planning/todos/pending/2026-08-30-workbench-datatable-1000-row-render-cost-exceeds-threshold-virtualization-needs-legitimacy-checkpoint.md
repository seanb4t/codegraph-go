---
created: 2026-08-30T00:00:00.000Z
title: Workbench DataTable's 1000-row render cost exceeds D-08's threshold — virtualization needs a package-legitimacy checkpoint
area: ui
severity: minor
files:
  - web/src/lib/components/workbench/DataTable.svelte
  - web/tests/data-table-render-cost.test.ts
threat_ref: T-04-31
---

## Problem

`04-07-PLAN.md` Task 3 (D-08, measurement-first) measured the Workbench's
shared `DataTable` shell at `MaxLimit` = 1000 rows
(`internal/query/validate.go:26`) — the real worst case any of the four
analyses (`Impact`/`Affected`/`Callers`/`Callees`) can return.

The plan fixed its threshold BEFORE the number was seen (jsdom, a coarse
order-of-magnitude detector, not a performance budget):
- median initial render ≤ 400ms
- median sort-toggle ≤ 200ms

**Observed, reproduced across 4 separate runs this session**
(`task web:render-cost`, `web/tests/data-table-render-cost.test.ts`):

| Metric | Threshold | Observed median (4 runs) |
|---|---|---|
| Initial render, 1000 rows | ≤ 400ms | 408–421ms |
| Sort-toggle, 1000 rows | ≤ 200ms | 363–480ms |

Both metrics consistently and stably exceed the threshold — this is not
run-to-run noise.

Per D-08 and 04-07-PLAN.md's own prohibitions, **the plan HALTED rather
than installing `@tanstack/svelte-virtual@3.13.36`**: that package carries
the same `[SUS]`/too-new verdict as this phase's other npm packages
(04-RESEARCH.md confirms it is available and Svelte-5-compatible at
`3.13.36`) and needs its own separate, explicit, `blocking-human`
package-legitimacy checkpoint — which 04-07-PLAN.md explicitly states it
does not pre-authorize. No client-side row cap was added either (D-08
names that as rejected: never re-cap what the server already bounded).

## Solution

Whoever picks this up:

1. Run a `blocking-human` package-legitimacy checkpoint for
   `@tanstack/svelte-virtual@3.13.36` (same pattern as this phase's other
   `[SUS]` npm packages, e.g. 04-01-SUMMARY.md's checkpoint for
   `@tanstack/svelte-table`/`shadcn-svelte`): verify the package on
   npmjs.com, its GitHub repo, download counts, and confirm the `[SUS]`
   verdict is the same "reads the LATEST version's publish timestamp, not
   package age" false-positive shape already approved elsewhere in this
   project.
2. If approved: wire virtualization into `DataTable.svelte` (or a sibling
   variant) using `@tanstack/svelte-virtual`'s Svelte 5 API, re-run
   `web/tests/data-table-render-cost.test.ts` with `RENDER_COST_ASSERT=1`
   (`task web:render-cost`) to confirm the threshold now passes, and update
   `04-07-SUMMARY.md`'s over-threshold record with the resolution.
3. If declined: record why (e.g. the jsdom threshold is judged not
   representative of real browser performance, or the numbers are accepted
   as-is for v1) and close this todo with that rationale instead.

Do NOT add a client-side row cap as an alternative — D-08 explicitly
rejects re-capping what the server (`MaxLimit`) already bounds.
