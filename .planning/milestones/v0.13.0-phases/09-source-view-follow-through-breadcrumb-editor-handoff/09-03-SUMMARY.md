---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 03
subsystem: ui
tags: [svelte, breadcrumb, source-view, playwright, tdd]

# Dependency graph
requires:
  - phase: 09-01
    provides: regenerated TS client (web/) used by SourcePane's client prop
provides:
  - splitHighlightedLines (web/src/lib/source-lines.ts) — balanced per-line splitting of highlight.js markup
  - innermostSymbolAt / firstFullyVisibleLine / SymbolRange (web/src/lib/breadcrumb.ts)
  - SourcePane's per-line DOM restructure (both file and single-def branches) with an always-rendered plain gutter
  - SourcePane's sticky single-line breadcrumb bar (source-breadcrumb) over FileSymbols ranges
  - web/scripts/breadcrumb-check.mjs — real-Chromium, real-index live gate with an independent oracle
  - 09-MUTATION-LOG.md opened with the cleanliness-gate convention and family (a)
affects: [09-04, 09-05]

# Actuals (#2632)
actuals:
  tokens: 16081
  tasks: 3
  commits: 5
  plan_head_before: aafee950ad35260c29f9d42666d5d5ea374293f9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-line DOM rows (splitHighlightedLines) instead of a single {@html} blob — the one rendering change both the breadcrumb and 09-04's gutter links depend on"
    - "$derived wrappers (filePath, stableClient) to gate an $effect on VALUE equality rather than raw prop-read identity, avoiding spurious re-dispatch on unrelated parent re-renders"
    - "untrack() around a $state cache read inside the same effect that writes that cache, to avoid a self-triggering re-run loop"
    - "Live-gate poll-until-STABLE (two consecutive identical DOM reads) instead of poll-until-changed-plus-fixed-settle, for two template bindings sharing one rAF-throttled dependency"

key-files:
  created:
    - web/src/lib/source-lines.ts
    - web/src/lib/breadcrumb.ts
    - web/tests/source-lines.test.ts
    - web/tests/breadcrumb.test.ts
    - web/tests/source-pane-breadcrumb.test.ts
    - web/scripts/breadcrumb-check.mjs
    - corpora/breadcrumb-check.json
    - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md
  modified:
    - web/src/lib/components/browse/SourcePane.svelte
    - web/build/** (rebuilt via task web:build)

key-decisions:
  - "innermostSymbolAt and firstFullyVisibleLine live in web/src/lib/breadcrumb.ts as pure, DOM-free functions, unit-tested at every boundary/tie/empty case from the plan's <behavior> list"
  - "The gutter is plain text in this plan (D-10): 09-04 turns cells into links when the editor link is buildable — no link markup added here"
  - "The breadcrumb's symbol is a <button>, never an <a href=\"#...\">, to avoid colliding with the browse route's own URL-driven navigation identity"
  - "breadcrumb-check.mjs's oracle re-implements innermost-range derivation independently (sort-based, not the SPA's iterative min-tracking) and fetches FileSymbols directly over HTTP — never importing the SPA's own module — so a wrong SPA derivation cannot agree with itself"

requirements-completed: [BRW-10]

coverage:
  - id: D1
    description: "Two pure functions (splitHighlightedLines, innermostSymbolAt, firstFullyVisibleLine) covering every boundary/tie/empty case in the plan's <behavior> list"
    requirement: BRW-10
    verification:
      - kind: unit
        ref: "web/tests/source-lines.test.ts (7 tests incl. 50-input property loop)"
        status: pass
      - kind: unit
        ref: "web/tests/breadcrumb.test.ts (15 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "SourcePane renders one DOM row per source line with an always-rendered plain gutter cell, in BOTH the file and single-def branches"
    requirement: BRW-10
    verification:
      - kind: unit
        ref: "web/tests/source-pane-breadcrumb.test.ts#SourcePane: per-line rows and gutter (both views)"
        status: pass
      - kind: unit
        ref: "web/tests/source-pane.test.ts (unmodified, still green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Sticky single-line breadcrumb naming the innermost FileSymbols range at the first fully visible line, with an honest empty state and a truncated-cap notice, file view only"
    requirement: BRW-10
    verification:
      - kind: unit
        ref: "web/tests/source-pane-breadcrumb.test.ts#SourcePane: sticky breadcrumb bar (file view only) (7 tests)"
        status: pass
      - kind: e2e
        ref: "corpora/breadcrumb-check.json (real Chromium, real index, independent oracle, success:true)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The breadcrumb demonstrated wrong (absent) against the pre-fix build, recorded as 09-MUTATION-LOG.md family (a)"
    requirement: BRW-10
    verification:
      - kind: e2e
        ref: "09-MUTATION-LOG.md#Family (a) — pre-fix binary at aafee950, breadcrumbPresent:false, exit 1"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 3: Per-Line Source Rows and the Sticky BRW-10 Breadcrumb Summary

**Restructured `SourcePane`'s single `{@html}` blob into per-line DOM rows and built a sticky "which symbol am I inside" breadcrumb on top, verified in a real Chromium against this repository's own index and watched FAIL against the pre-fix build.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3
- **Files modified:** 45 (9 source/test/script/doc files + 36 web/build/ output files rebuilt by `task web:build`)

## Accomplishments

- `web/src/lib/source-lines.ts`'s `splitHighlightedLines` re-arranges highlight.js's own `<span>`/`</span>` tokens into balanced per-line strings with a single linear scan, never un-escaping text and never becoming a second markup-from-repository-bytes site (T-09-05)
- `web/src/lib/breadcrumb.ts`'s `innermostSymbolAt` and `firstFullyVisibleLine` are the pure "which symbol am I inside" derivation, with every boundary, tie and empty case from the plan pinned by unit tests
- `SourcePane.svelte` now renders one DOM row per source line (`[data-line="N"]`, id `L{N}`) with an always-rendered plain gutter cell, in both the file and single-def branches — the one rendering change 09-04's editor-link gutter will reuse
- A sticky, single-line breadcrumb bar (`source-breadcrumb`) renders in the file view only, naming the innermost `FileSymbols` range at the first fully visible line, with an honest empty state (never the nearest preceding symbol) and a truncated-cap notice
- `web/scripts/breadcrumb-check.mjs` proves the breadcrumb in a real Chromium against this repository's own real index with an independent oracle, and the same script demonstrates the breadcrumb absent against the pre-fix build — recorded in `09-MUTATION-LOG.md` family (a)

## Task Commits

Each task was committed atomically (TDD RED→GREEN for Tasks 1-2; Task 3's RED is the pre-fix binary, not a test commit, per the plan's own framing):

1. **Task 1: Two pure functions** — `9af6f095` (test, RED) → `9985fb04` (feat, GREEN)
2. **Task 2: SourcePane per-line rows and breadcrumb** — `86e8ca8d` (test, RED) → `ff2e5a64` (feat, GREEN)
3. **Task 3: Live gate, pre-fix RED, mutation log** — `966026a7` (test)

_TDD tasks: RED commit exports the target functions/test-ids with wrong-answer stubs so tests fail on assertions, not module resolution; GREEN commit implements them._

## TDD Gate Compliance

| Task | RED commit | GREEN commit | RED evidence | GREEN evidence |
|------|-----------|---------------|---------------|-----------------|
| 1 | `9af6f095` | `9985fb04` | 20 failed / 3 passed (23 total) against `[]`/`null`/`0` stubs | 23/23 pass |
| 2 | `86e8ca8d` | `ff2e5a64` | 8 failed / 1 passed (9 total) against pre-restructure component | 27/27 pass across source-pane-breadcrumb, source-pane, browse-page, browse-tracer |
| 3 | n/a (RED is the pre-fix binary) | `966026a7` | `breadcrumbPresent: false`, exit 1, against binary built from `aafee950` | `success: true`, exit 0, against `./codegraph` built from HEAD |

RED transcript, Task 1 (`pnpm -C web exec vitest run tests/source-lines.test.ts tests/breadcrumb.test.ts`):
```
FAIL  tests/source-lines.test.ts > splitHighlightedLines > never un-escapes text
AssertionError: expected [] to deeply equal [ '&lt;script&gt;', '&amp;' ]
...
Test Files  2 failed (2)
     Tests  20 failed | 3 passed (23)
```

RED transcript, Task 2 (`pnpm -C web exec vitest run tests/source-pane-breadcrumb.test.ts`):
```
Test Files  1 failed (1)
     Tests  8 failed | 1 passed (9)
```

RED transcript, Task 3 (pre-fix binary at `aafee950`):
```
breadcrumb-check: oracle has 13 symbols for internal/query/node.go
breadcrumb-check: wrote /tmp/breadcrumb-prefix.json — success=false
breadcrumb-check: fatal — breadcrumb-check: FAILED — breadcrumbPresent=false error=pollUntil: condition did not become true within 15000ms
```
Full transcripts and the recorded JSON diagnostics are pasted verbatim in `09-MUTATION-LOG.md` family (a).

## Files Created/Modified

- `web/src/lib/source-lines.ts` — `splitHighlightedLines`, balanced per-line splitting of highlight.js markup
- `web/src/lib/breadcrumb.ts` — `SymbolRange`, `innermostSymbolAt`, `firstFullyVisibleLine`
- `web/tests/source-lines.test.ts`, `web/tests/breadcrumb.test.ts`, `web/tests/source-pane-breadcrumb.test.ts` — unit coverage for the above plus SourcePane's new behavior
- `web/src/lib/components/browse/SourcePane.svelte` — per-line rows/gutter in both branches, `FileSymbolsClient`, the sticky breadcrumb, scroll/resize-driven `currentLine` tracking
- `web/scripts/breadcrumb-check.mjs` — the live-browser gate and its independent oracle
- `corpora/breadcrumb-check.json` — the committed GREEN diagnostic record
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md` — opened with the cleanliness-gate convention and family (a)
- `web/build/**` and `web/build/.build-manifest` — rebuilt to embed the restructured SPA (`task web:build`, verified via `task web:drift`)

## Decisions Made

- `innermostSymbolAt`/`firstFullyVisibleLine` are pure and DOM-free, unit-tested independently of the component that consumes them (D-02/D-03 made assertable in isolation).
- The gutter renders plain digits in this plan (D-10) — no link markup, deferred to 09-04.
- The breadcrumb's symbol control is a `<button>`, never a hash-fragment `<a>`, to avoid colliding with the browse route's own URL-driven navigation identity.
- `breadcrumb-check.mjs`'s oracle is a genuinely independent (sort-based, not iterative) re-implementation of the innermost-range rule, fetching FileSymbols directly over HTTP rather than importing the SPA's own derivation module.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reactivity dependency leak caused spurious FileSymbols re-fetch and premature abort**
- **Found during:** Task 2, writing the "fetches FileSymbols exactly once per path" test
- **Issue:** The fetch effect initially read the `client` prop and the `fileSymbolsCache` `$state` Map directly. Testing-library's `rerender()` reassigns its whole props container on every call, which dirties raw prop reads regardless of whether the referenced value actually changed; separately, the effect's own cache write (on fetch resolution) re-triggered itself since the effect read that same cache. Together these caused the effect to re-run on a same-path re-render, aborting the just-completed request.
- **Fix:** Wrapped `client` in a `$derived` (`stableClient`) so Svelte's own value-equality dedup (`Object.is`) gates re-runs on the referenced object actually differing, and wrapped the cache read in `untrack()` so writing to the cache doesn't re-trigger the effect that wrote it.
- **Files modified:** `web/src/lib/components/browse/SourcePane.svelte`
- **Verification:** `web/tests/source-pane-breadcrumb.test.ts`'s once-per-path/abort-on-change test passes; confirmed via temporary debug instrumentation that the effect ran exactly twice (once per genuine path) rather than three times.
- **Committed in:** `ff2e5a64` (part of the task's own feat commit — found and fixed before that commit, not a follow-up)

**2. [Rule 1 - Bug] Live-gate observation race read a torn DOM state**
- **Found during:** Task 3, first live-browser run
- **Issue:** `data-current-line` and the crumb-derived `data-empty`/button text are two separate template bindings sharing the `currentLine` dependency, committed inside a `requestAnimationFrame`-throttled flush. A fixed "wait for line-changed, then two more animation frames" settle intermittently read `data-current-line` already updated while `data-empty` had not yet flipped — reproduced twice, including once even after adding the double-rAF barrier.
- **Fix:** Replaced the fixed-wait settle with poll-until-STABLE: read the full observation tuple repeatedly until two consecutive reads agree, which cannot land on a torn intermediate state by construction.
- **Files modified:** `web/scripts/breadcrumb-check.mjs`
- **Verification:** 3 consecutive full live-gate runs all succeeded with `success: true`.
- **Committed in:** `966026a7`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found and fixed during the plan's own implementation, not scope creep).
**Impact on plan:** Both fixes were necessary for the plan's own stated contracts (once-per-path fetch, an honest live-gate proof) to actually hold; no scope was added beyond what the plan specified.

## Issues Encountered

None beyond the two auto-fixed reactivity/timing bugs documented above, both found and resolved during implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The per-line DOM restructure now exists once, in both the file and single-def branches, ready for 09-04 to turn the plain gutter cells into editor links without touching the `<pre>` block again. `09-MUTATION-LOG.md` is open with the cleanliness-gate convention and family (a); families (b)-(d) are 09-05's responsibility. No blockers.

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*

## Self-Check: PASSED

- All 9 created files verified present on disk (`web/src/lib/source-lines.ts`, `web/src/lib/breadcrumb.ts`, three test files, `web/scripts/breadcrumb-check.mjs`, `corpora/breadcrumb-check.json`, `09-MUTATION-LOG.md`, this SUMMARY).
- All 6 commit hashes (`9af6f095`, `9985fb04`, `86e8ca8d`, `ff2e5a64`, `966026a7`, `d4592a2`) verified present via `git log --oneline --all`.
- Full `pnpm -C web exec vitest run` (45 files, 499 tests) and `pnpm -C web check` (0 errors) re-confirmed green at HEAD before writing this SUMMARY.
