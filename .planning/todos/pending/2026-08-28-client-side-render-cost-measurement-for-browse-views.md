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
