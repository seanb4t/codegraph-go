---
created: 2026-08-28T00:00:00.000Z
title: Client-side render-cost measurement for browse views (no budget, no gate — measurement only)
area: ui
severity: minor
files:
  - web/src/lib/highlight.ts
  - web/src/lib/call-targets.ts
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/lib/components/browse/NeighborsPanel.svelte
  - web/src/lib/components/browse/SearchPanel.svelte
---

## Problem

Cycle-1 cross-AI review of `03-08-PLAN.md` (Codex, cross-plan MEDIUM,
uncontradicted by the other review lane) observed that Phase 3 plans no
measurement for three client-side costs, all of which land in this plan's
own surface area:

1. **Rendering and highlighting a source blob at the server's truncation
   cap.** `internal/uiserver/truncate.go`'s `sourceLineCap` (4096 lines) /
   `sourceByteCap` (256 KiB) bound what the server SENDS; nothing measures
   what it costs the browser to run `highlight.js`'s tokenizer plus this
   plan's own `decorateCallTargets` TreeWalker pass over a blob at that
   cap, on the main thread, before the source pane paints.
2. **Rendering hundreds of callers/callees.** `NeighborsPanel.svelte`
   (03-07) renders every entry in `GetNodeDetailResponse.calls`/
   `called_by` with no virtualization; a highly-connected symbol in a
   large codebase could carry hundreds of neighbours, each a real DOM
   node with a click handler.
3. **The DOM cost of a large search result list.** `SearchPanel.svelte`
   (03-06) renders every live `Search`/`Files` match as a `Command.Item`
   with no virtualization either.

**Disposition, recorded at 03-08 planning time (not incorporated into that
plan's tasks) — the reasons, in order of weight:**

- **No requirement in scope asks for it.** Phase 3's fourteen requirement
  IDs (BRW-01..BRW-09, NAV-01..NAV-04, SRV-05) are all behavioural. Adding
  a performance budget now would mean inventing both the threshold and the
  measurement methodology with no requirement backing either — exactly the
  kind of gate that gets relaxed the first time it is inconvenient, which
  this project's own prohibitions elsewhere are written against.
- **The dominant cost is already bounded server-side, and that bound is
  tested.** `truncate.go`'s line-then-byte cap (RPC-05) bounds every
  `SourceBlob` before it leaves the server; `validateLimit`/`MaxLimit`
  bound result counts for every traversal RPC. The client renders what it
  is given — there is no UNBOUNDED-input version of this risk on this
  wire today.
- **The remaining risk is real but reads as a Phase 4 concern.**
  Highlighting a capped blob plus decorating it is O(source length) on
  the main thread; hundreds of neighbour entries or search results is a
  list-virtualization question. Both are worth measuring against the
  workbench's denser views (Phase 4, interactive `impact`/`affected`/
  `callers`/`callees` with tweakable depth/limits) rather than against
  this phase's single-pane, single-list views.

## Solution

When this is picked up (Phase 4 or later, whichever phase first ships a
denser view than Browse's), take three concrete measurements before
deciding whether a budget or a virtualization change is warranted:

1. **Source-pane render+highlight+decorate time** for a blob at
   `sourceLineCap`/`sourceByteCap` (build a synthetic fixture at exactly
   4096 lines / 256 KiB, or find/construct a real repository file near
   that size), measured via the browser's own Performance API
   (`performance.mark`/`performance.measure`) around the `highlightSource`
   call and the `callTargets` action's initial `decorateCallTargets` pass,
   not a synthetic microbenchmark disconnected from the real render path.
2. **NeighborsPanel render time and frame cost** for a symbol with a
   few hundred callers/callees — pick or synthesize a real high-fan-in
   node (a widely-used interface method or a core utility function is a
   plausible real-world candidate in most codebases).
3. **SearchPanel render time and frame cost** for a query matching a few
   hundred symbols/files — a short, common substring against a large
   indexed repository.

Record the three numbers (and the repository/fixture size they were taken
against) in whatever plan closes this todo. Only THEN decide, informed by
real numbers rather than assumption, whether: (a) the numbers are fine and
this todo closes as "measured, no action needed"; (b) a render-cost budget
is worth adding, with an explicit owner and threshold; or (c) list
virtualization is warranted for `NeighborsPanel`/`SearchPanel` specifically
(the source pane has no equivalent "virtualize" lever — it is a single
`<pre><code>` block, not a list — so that path, if the numbers warrant
action, is more likely a re-look at `sourceLineCap`/`sourceByteCap`
themselves than a client-side virtualization change).

## Resolution (2026-08-30, Phase 4, 04-07-PLAN.md)

**Redirected by 04-CONTEXT.md D-08**, not resolved against the original
three-part ask verbatim: this todo's own text named "the workbench's denser
views (Phase 4, interactive `impact`/`affected`/`callers`/`callees` with
tweakable depth/limits)" as the natural place to take this measurement, and
D-08 made that redirection the phase's actual decision — measure the
Workbench's shared `DataTable` shell (D-06) at `MaxLimit` = 1000 rows
(`internal/query/validate.go:26`), the real worst case any of the four
analyses can return, rather than re-measuring `SourcePane`/`NeighborsPanel`/
`SearchPanel` from Phase 3.

**Method:** `web/tests/data-table-render-cost.test.ts` mounts `DataTable`
(via a concretely-typed test host, `web/tests/support/data-table-location-host.svelte`)
with a 1000-row `Location` fixture whose insertion order is a fixed,
deterministic, non-monotonic permutation (`(i * 457) % n`) — neither
ascending nor descending, so a sort toggle is proven to do real reordering
work rather than a no-op over already-sorted data. `performance.now()`
around 5 independent mounts (initial render) and 5 sort-toggle clicks on
one mount (sort toggle), MEDIAN reported for each, plus a positive-control
assertion that exactly 1000 `<tr>` rows actually rendered.

**Observed (reproduced consistently across 4 separate runs this session,
`RENDER_COST_ASSERT=1 pnpm exec vitest run tests/data-table-render-cost.test.ts`
/ `task web:render-cost`):**

| Metric | Threshold (plan-fixed) | Observed median (4 runs) |
|---|---|---|
| Initial render, 1000 rows | ≤ 400ms | 408–421ms |
| Sort-toggle, 1000 rows | ≤ 200ms | 363–480ms |

Both medians consistently exceed the plan's fixed jsdom threshold (a coarse
order-of-magnitude detector, not a performance budget) — this is a stable,
reproducible over-threshold result, not run-to-run noise.

**Branch taken: OVER THRESHOLD → HALT, per D-08 and this plan's own
prohibitions.** `@tanstack/svelte-virtual@3.13.36` is NOT installed —
adding it carries the same `[SUS]`/too-new verdict as this phase's other
npm packages and needs its own separate, explicit, blocking-human
package-legitimacy checkpoint, which 04-07-PLAN.md explicitly states it
does not pre-authorize. No client-side row cap was added either (D-08
names that as rejected: never re-cap what the server already bounded). The
measured numbers are reported here and in `04-07-SUMMARY.md`; the
executing session returned a `CHECKPOINT REACHED` requesting that
package-legitimacy decision from the maintainer.

**Scope note:** `SourcePane`/`NeighborsPanel`/`SearchPanel` (Phase 3's
Browse views, the todo's original three-part ask) were NOT separately
re-measured in this plan — D-08's redirection scoped Phase 4's obligation
to the Workbench table specifically. If those three components' render
cost remains a live concern, it needs its own fresh todo; this resolution
does not claim to have measured them.

Deterministic, unconditional assertions (1000 rows rendered; sort toggle
genuinely reorders by rendered text content) run on every `task web:test`
invocation and are documented as passing (260/260) in `04-07-SUMMARY.md`.
The threshold comparison itself lives behind the new opt-in
`task web:render-cost` target, wired into no workflow.
