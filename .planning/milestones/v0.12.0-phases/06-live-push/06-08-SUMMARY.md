---
phase: 06-live-push
plan: 08
subsystem: ui
tags: [svelte, live-push, connect-rpc, gap-closure, playwright, build]

requires:
  - phase: 06-live-push
    provides: "06-02's publisher and generation counter; 06-03's liveStore and the coalescing/requestId pattern established on health/+page.svelte; 06-04's WatchGraph streaming handler; 06-07's committed bundle, which this plan rebuilds on top of"
provides:
  - "web/src/routes/+page.svelte: the root Status route subscribed to liveStore, re-issuing the SAME GetStatus call it already owns on every newer generation"
  - "one monotonic requestId shared by the mount fetch and every live-triggered fetch, closing CR-01's race class at this site"
  - "web/tests/live-route-refetch.test.ts: the root-route describe block — the coverage whose absence let a landing page render `Stale: no` over a generation-old count"
  - "a second web/build rebuild, staged and committed with the assertion placed AFTER the commit"
affects: []

actuals:
  tokens: 24000
  tasks: 3
  commits: 2
  plan_head_before: 05d9b64fb99271be3e27a0bb1fa283827fe0ffea

tech-stack:
  added: []
  patterns:
    - "one fetch function parameterised by `generation: bigint | null` — null for the mount fetch, the event's generation for a live-triggered one — so both paths issue an identical rpc and share one ordering token, rather than health's two near-duplicate functions"
    - "the store's synchronous initial subscribe delivery treated as a BASELINE not a trigger, so mounting a view while an event is already current issues no redundant fetch"
---

# Phase 6 Plan 8: The Root Status Route's Live Subscription (criterion-1 gap closure) Summary

## Performance

- 3 tasks, 2 commits, one RED→GREEN cycle, no deviations.
- Frontend suite 464 → **467 passing** (42 files). `svelte-check`: 1159 files, **0 errors, 0 warnings**.
- **Zero** new dependencies: `git diff --stat 05d9b64f..HEAD -- go.mod go.sum web/package.json web/pnpm-lock.yaml` is empty, positive-controlled against `web/src/routes/+page.svelte` in the same range (`76 insertions(+), 2 deletions(-)`), proving the command and range are live.

## Accomplishments

Closed the criterion-1 gap the live browser UAT found **after** this phase had already verified 5/5: the root Status route (`/`) — the default landing view `codegraph ui` opens — never applied live events.

## Task Commits

| Task | Commit | Subject |
|---|---|---|
| plan | `4e3d1dca` | docs(06): add 06-08 gap-closure plan for the root route live subscription |
| 1–3 | `110fbb7a` | fix(06): root Status route applies live events (LIV-02) |

## Files Created/Modified

- `web/src/routes/+page.svelte` — +76/−2
- `web/tests/live-route-refetch.test.ts` — +106
- `web/build` — rebuilt (content hashes rotated: 5 chunk/entry/node files deleted, 5 added)

## The defect, as measured

The root route called `uiClient.getStatus({})` once inside `onMount` and subscribed to
nothing. Five files subscribed to `liveStore` — `+layout`, `browse`, `graph`, `health`,
`AnalysisPanel` — and the root route was not among them.

Measured in a real browser against the shipped binary, page open and untouched, store
advanced by a real `codegraph sync`:

| | page rendered | store held |
|---|---|---|
| Files | 617 | **618** |
| Edges | 41,766 | **41,768** |
| Live events delivered to the page | `[3, 4]` | — |
| Reloads | 0 (`navigations: 1`) | — |

It held those values for the full 10-second window. The event **reached the page and was
ignored**. Simultaneously the same `<dl>` rendered **`Stale: no`** — not merely a failure
to refresh, but a positive assertion of freshness over data provably a generation behind.
That is the precise failure the phase goal names: *"Open views stop going quietly stale."*

## Why it survived every prior gate

1. **The route is Phase 2 code.** `+page.svelte:2` cites D-19, T-02-05-05, T-02-05-07 —
   all Phase 2 ids — and calls itself *"the shell's live GetStatus render"*. "Live" there
   meant *real data rather than a placeholder*. Phase 6 wired the four surfaces it thought
   of as views and never re-read a file whose own comment already claimed the word.
2. **The test suite mirrored the omission exactly.** `live-route-refetch.test.ts` carried
   describe blocks for `health`, `AnalysisPanel` and `browse` — and none for the root
   route. The one view that did not subscribe was the one view with no coverage asserting
   it should. That symmetry is why 464 green tests said nothing.
3. **The verifier had the evidence and read it the safe way.** It reported "four surfaces
   subscribe", listing `health:81`, `browse:140`, `graph:243`, `AnalysisPanel:152`. That
   list is *correct*. The defect is what is absent from it. **Enumerating what exists can
   never surface an omission unless you also enumerate what should exist** — the same
   shape as rule `84d1gfpywd`'s silent zero, one level up.

## RED Observations (this phase's discipline: a gate never seen red proves nothing)

All three new cases were demonstrated RED against unmodified `+page.svelte` **before**
Task 2 was written:

```
Test Files  1 failed (1)
     Tests  3 failed | 5 skipped (8)
```

Each failed identically — `expected 2, received 1` on the call counter, with the rendered
DOM still showing `Loading status…`. The live event delivered and the route issued
nothing. The 5 pre-existing cases were skipped by the `-t 'root Status route'` filter,
and the 3 failures are named, not inferred from an exit status.

## Decisions Made

- **One fetch function, not two.** Health carries `issueLiveHealthFetch` alongside a
  near-identical mount fetch. Here a single `issueStatusFetch(generation: bigint | null)`
  serves both, with `null` meaning "mount fetch, not part of the coalescer". Same
  semantics, one code path, no chance of the two drifting.
- **`onMount` retained** rather than converted to health's `$effect`+`AbortController`
  shape. The conversion is churn without benefit here; the ordering guarantee this route
  needs comes from the shared `requestId`, not from abort.
- **The event stays a trigger.** No `WatchGraphEvent` field enters the render path
  (D-05). The nine `<dl>` rows and `classify()` are untouched — this plan changed *when*
  the data is fetched, never *what* is displayed.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

| Check | Result |
|---|---|
| 3 new cases RED before the fix | 3 failed / 3, named |
| 3 new cases GREEN after | 8/8 in the file |
| Full frontend suite | **467/467**, 42 files |
| `svelte-check` | 1159 files, 0 errors, 0 warnings |
| `task web:drift` | **PASS** — source half MATCH (110 files), output half MATCH (32 files) |
| `git status --porcelain` after the bundle commit | empty |
| New bundle files tracked | 4/4 `git ls-files --error-unmatch` succeed; positive control confirms a deleted chunk is correctly absent |
| Zero new dependencies | empty diff, positive-controlled |

## Live browser re-confirmation (the measurement that exposed the gap)

Freshly built binary embedding the rebuilt bundle, `codegraph ui` on `/`, baseline pinned
in the page **before** the sync so the post-fix update could not be mistaken for it:

| | baseline | after real `codegraph sync` | store |
|---|---|---|---|
| Files | 616 | **617** | 617 |
| Edges | 41,766 | **41,768** | 41,768 |
| Nodes | 14,047 | **14,049** | — |

`changed: true`, `appliedWithinMs: 0` (already applied before the 50ms poll began),
generations `[1, 2, 3]`, `navigations: 1` — no reload, no user action. `Stale: no` is now
a true statement rather than a false one.

An earlier attempt at this measurement was *discarded as unsound*: the fix applied so
quickly that the "before" read already contained the updated value, making a real pass
look like no change. The baseline had to be pinned in a separate tool call first. Noting
this because the discarded run would have read as a FAIL, and a gate that can be misread
in the failing direction is worth recording.

## Deviations from Plan

None. All three tasks executed as written.

## Issues Encountered

None blocking. Two findings from the same UAT are **out of this plan's scope** and remain
open (see Next Phase Readiness).

## Known Stubs

None.

## User Setup Required

None.

## Next Phase Readiness

Criterion 1 now holds for every index-data surface. Two open items from the same UAT,
both **pre-existing (Phase 2 app-shell era), neither a Phase 6 defect**:

1. **Stock Svelte favicon that also violates CSP.** `web/src/lib/assets/favicon.svg` is
   the SvelteKit template logo (`<title>svelte-logo</title>`, `#ff3e00`). Vite inlines it
   as a `data:` URI; `spa.go:99` sets `default-src 'self'` with no `img-src`, so it is
   blocked — a console error on every page load and a blank tab icon. Two independent
   defects cancelling into "no visible logo". Note `spa_test.go:439` asserting `default-src`
   is exactly `['self']` is *correct* and deliberate (T-02-02-06) — the CSP is behaving as
   designed; the asset choice is the defect.
2. **Unconfirmed count observations.** `codegraph sync` reports `nodes=24498` where the
   status rpc reports `14047`; and within a *single* `GetHealth` response, "Files by
   language" summed one higher than "Nodes by kind → `file`". Both reproduced but neither
   root-caused — recorded as observations, **not asserted as bugs**.

Also: `.playwright-mcp/` is written into the repo root by the browser MCP and is not
gitignored (removed manually after each UAT).

## Self-Check: PASSED
