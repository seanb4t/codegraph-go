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

## Resolution (2026-08-30, Phase 4, 04-07-PLAN.md continuation)

**Approved.** Maintainer decision, verbatim: "Approve @tanstack/svelte-virtual
and wire it". Verified against live npm before the decision: `time.created`
2022-07-19 (~4 years old, not a new package), `dist-tags.latest` 3.13.36
(published 2026-08-18), repo `github.com/TanStack/virtual` (same org as
`@tanstack/svelte-table`), one transitive dependency
(`@tanstack/virtual-core@3.17.8`, same org), ~78,281 weekly downloads — the
same `[SUS]`/too-new false-positive shape already adjudicated for
`@tanstack/svelte-table`/`shadcn-svelte` (04-01) and confirmed again here.

**Installed exactly the approved pair, nothing more:**
`pnpm add -D @tanstack/svelte-virtual@3.13.36` (exact-pinned, no caret) →
brought in `@tanstack/virtual-core@3.17.8` transitively. Verified via
`web/pnpm-lock.yaml`: exactly these two new entries, no others.
`task web:lockfile` → 233 packages (231 + 2), 233/233 integrity-bearing,
zero non-registry sources. `task web:audit` → CLEAN, zero advisories.

**Wired into the ONE shared `DataTable.svelte` shell (D-06 — no second
table component or variant created):** a two-padding-row virtualization
(the standard technique for native `<table>` markup, since a `<tr>` cannot
be individually absolute-positioned without breaking table layout), scroll
container height 600px, uniform row-height estimate (37px — no
`measureElement`/ResizeObserver dependency, unnecessary for a uniform row
height and unreliable under jsdom). The row MODEL
(`table.getRowModel().rows`) still carries every row the server returned —
sorting, selection and a new `aria-rowcount` attribute all operate over
the FULL set; only the DOM footprint shrinks to the visible window. No
client-side row cap was added (D-08 remains satisfied: virtualization
renders a window of the full result set, it does not truncate one).

**A real, non-obvious jsdom gap surfaced and was fixed at its root, not
worked around per-test:** `@tanstack/virtual-core`'s `observeElementRect`
reads the scroll container's `offsetWidth`/`offsetHeight` SYNCHRONOUSLY on
every `setOptions` call — jsdom has no layout engine at all, so these are
always 0, which zeroed the virtualizer's visible range and made
`DataTable.svelte` render ZERO rows under every jsdom-based test
regardless of row count (an `initialRect` fallback does NOT help — the
synchronous real measurement overwrites it immediately). This broke 15
existing tests across every file that renders a `DataTable`
(`workbench-*.test.ts` AND `health-page.test.ts`, via `CountTable.svelte`,
D-06's shared shell) the first time virtualization landed. Fixed with a
global-but-scoped stub in `web/tests/setup.ts` — `offsetWidth`/
`offsetHeight` return a realistic non-zero size ONLY for the element
carrying `data-testid="data-table-scroll"` (`DataTable.svelte`'s own
scroll container); every other element keeps jsdom's native 0 behavior, so
the stub cannot mask an unrelated layout bug elsewhere.

**Deterministic render-cost assertions restated against the row MODEL, per
the maintainer's explicit instruction, not deleted:** "renders exactly
1000 rows" now reads `aria-rowcount` (the model count) rather than DOM
`<tr>` count, WITH a complementary check that the DOM count is now
strictly LESS than the model count (proving virtualization is genuinely
windowing the DOM, not merely claiming to via the attribute) and greater
than zero. "Sort toggle genuinely reorders" now compares the rendered
(windowed) slice against the SAME leading slice of the full sorted name
list, since only the top of the scroll position is in the DOM. Both keep
running unconditionally on every `task web:test` — no wall-clock assertion
was added to a PR-required job.

**Measured medians, before and after (both `RENDER_COST_ASSERT=1
pnpm exec vitest run tests/data-table-render-cost.test.ts` / `task
web:render-cost`, reproduced across multiple runs):**

| Metric | Threshold | Before (no virtualization) | After (virtualized) |
|---|---|---|---|
| Initial render, 1000 rows | ≤ 400ms | 408–421ms (FAILED) | 17.78–20.22ms (PASS) |
| Sort-toggle, 1000 rows | ≤ 200ms | 363–480ms (FAILED) | 10.59–12.19ms (PASS) |

Both metrics now pass with wide margin (roughly a 20–40x improvement) —
`task web:render-cost` exits 0.

**Bundle rebuilt and re-committed** (`task web:build` → `task web:drift`
GREEN, 103 source files / 31 output files — one new immutable chunk for
`@tanstack/virtual-core`), closing the window `DataTable.svelte`'s change
opened. `task web:components:drift` re-verified GREEN (50 files / 8
components, unaffected by this change). Full gate sweep green: `task
web:test` 260/260, `pnpm check` 1019 files / 0 errors, `GOTOOLCHAIN=go1.26.5
task test:unit` all packages ok, `task proto:drift` 4/4.

See `04-07-SUMMARY.md` for the full account, including the jsdom
`offsetHeight` root-cause investigation.
