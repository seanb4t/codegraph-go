---
phase: 01-defect-flake-burn-down
plan: 08
subsystem: ui
tags: [svelte, cytoscape, cytoscape-elk, playwright, graph-console-check]

# Dependency graph
requires:
  - phase: 01-01
    provides: "web/scripts/graph-console-check.mjs — permanent live-Chromium /graph console gate, reused here as this plan's own before/after proof"
  - phase: 01-06
    provides: "web/build/ rebuilt with the favicon fix — this plan rebuilds it again on top, does not revert it"
provides:
  - "graph-style.ts's node[?isSymbol] selector with the invalid text-valign key removed (FIX-05 stylesheet half, D-08)"
  - "GraphCanvas.svelte's mount effect: a dedicated, never-reused cytoscape mount element per effect run, closing a cytoscape-internal container-reuse auto-destroy path"
  - "GraphCanvas.svelte's runLayout(): an isLayoutInFlight()/onLayoutSettled() seam that the mount effect's cleanup uses to defer cy.destroy() until an outstanding ELK layout settles (FIX-04, WINDOWS #26)"
affects: [01-09]

# Actuals (#2632)
actuals:
  tokens: 3200
  tasks: 3
  commits: 2
  plan_head_before: 58b6dca7b2920e8ba8d4f64a558d89a0f2ce2485

tech-stack:
  added: []
  patterns:
    - "Per-instance cytoscape mount element: never hand cytoscape the SAME container element across two constructions on one component instance. cytoscape's own Core constructor keeps a registry on the container DOM node itself (container._cyreg) and unconditionally destroys any prior instance found there — a destroy call no amount of teardown-timing discipline in the consuming component's own cleanup can see or gate. A fresh `document.createElement('div')` per mount, appended inside the stable, ResizeObserver-observed outer container, sidesteps the hazard entirely using only cytoscape's public `container` option."
    - "Deferred-destroy-on-outstanding-async-work: extend an existing generation-token pattern with a boolean liveness flag (set before the async call, cleared only inside the SAME generation-checked completion callback) plus a single-slot settle-callback, rather than adding a parallel state machine. destroyNow() is idempotent via the library's own destroyed() predicate, never a second boolean, and is reachable from either the real settle path or a bounded fallback timer."

key-files:
  created: []
  modified:
    - web/src/lib/components/graph/graph-style.ts
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/build/** (rebuilt output, committed per web:drift's contract, across both task commits)

key-decisions:
  - "EDGE-FIX-04-01 resolved: the bounded fallback for a layout that never settles is 10 seconds. Every ELK layout this codebase measures (including guava's collapsed 135-node/163-cycle scale, the largest corpus this app renders) settles in well under a second in practice; 10s is a deliberately generous multiple chosen so the fallback can only ever fire against a genuinely stuck layout, never a slow-but-honest one — firing early would recreate the exact crash this gate exists to prevent."
  - "The double-mount trigger Task 2 needed to find was NOT any of the plan's three suggested sequences (CPU-throttle+navigate-away, rapid re-navigation, dev-mode double effect). A bare, single /graph page load in a fresh production-build Chromium session reproduces the crash 100% of 3/3 runs with no interaction at all — the mount effect runs twice for the SAME component instance, reusing the SAME container DOM element, independent of any navigation or throttling. See Deviations/Issues below for the full mechanism this uncovered."
  - "FIX-04's fix does NOT close the guava 'invalid endpoints' warnings and was never expected to: post-fix, the same directory pair (android/guava-tests/.../util/concurrent <-> android/guava/src/.../util/concurrent), same 2 opposite-direction edges, same non-overlapping settled bounding boxes as 01-01 recorded pre-fix. Only the edge UUIDs differ (re-index artifact, not semantically meaningful). This confirms 01-01's own reading: the invalid-endpoints warning is a SEPARATE mechanism from FIX-04's crash, not the same double-render family with a shared fix. Plan 01-09 should proceed on its own root-cause investigation, not assume this plan's fix helps."

requirements-completed: [FIX-04, FIX-05]

coverage:
  - id: D1
    description: "node[?isSymbol] selector's invalid text-valign key is removed; text-halign/text-margin-x (the keys that actually position the label) are unchanged; zero console entries naming the vertical-alignment property on either corpus"
    requirement: FIX-05
    verification:
      - kind: e2e
        ref: "node web/scripts/graph-console-check.mjs --out <scratch>; jq check for zero valign-naming console entries across both runs"
        status: pass
      - kind: other
        ref: "rg -A14 selector block check for text-halign/text-margin-x present, text-valign absent"
        status: pass
    human_judgment: false
  - id: D2
    description: "Live reproduction of the cytoscape notify() TypeError, with its actual trigger (a bare /graph load, no navigation needed) recorded, superseding the plan's three suggested trigger sequences"
    requirement: FIX-04
    verification:
      - kind: e2e
        ref: "scratch Playwright driver (not committed): 3/3 reproductions on self corpus via a single /graph page load, full instance/generation/layoutInFlight trace captured"
        status: pass
    human_judgment: false
  - id: D3
    description: "cy.destroy() is deferred while an ELK layout is outstanding, using isLayoutInFlight()/onLayoutSettled() extending the existing layoutGeneration pattern; a 10s bounded fallback prevents an indefinite leak; destroyNow() is idempotent via cy.destroyed()"
    requirement: FIX-04
    verification:
      - kind: e2e
        ref: "graph-console-check.mjs post-fix: self corpus pageErrorCount=0 (was 1), 5/5 repeat runs clean; guava 3/3 clean; rapid-renav and throttle-heavy-rapid sequences also clean"
        status: pass
      - kind: unit
        ref: "pnpm check (svelte-check): 0 errors; pnpm test: 584/584 passing"
        status: pass
    human_judgment: false
  - id: D4
    description: "A second, cytoscape-internal destroy path (container-reuse auto-destroy via container._cyreg) discovered live is also closed, via a dedicated never-reused mount element per effect run — not just the deferred-destroy gate the plan anticipated"
    requirement: FIX-04
    verification:
      - kind: e2e
        ref: "same graph-console-check.mjs post-fix run (D3) — the fix would not hold without this half; see key-decisions and Deviations for the mechanism"
        status: pass
    human_judgment: false
  - id: D5
    description: "No dependency patch, no lockfile/manifest change, no added catch block, no ELK option key change, task web:drift passes"
    requirement: FIX-04
    verification:
      - kind: other
        ref: "sha256 check against package.json/pnpm-lock.yaml planning-time digests (OK); rg catch-count base=0 now=0; base-vs-current ELK option key diff (empty); task web:drift PASS"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 8: Symbol-Label Stylesheet Fix and Deferred Cytoscape Teardown Summary

**Removed the invalid `text-valign: 'right'` from the symbol-node stylesheet, and closed the cytoscape `notify()` null-renderer TypeError by both deferring this component's own `cy.destroy()` while an ELK layout is outstanding AND giving every mount its own never-reused cytoscape container element — the latter closing a cytoscape-internal auto-destroy path that live reproduction showed the deferred-destroy gate alone could not reach.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3
- **Files modified:** 2 hand-authored source files (`graph-style.ts`, `GraphCanvas.svelte`) + `web/build/**` rebuilt output across 2 commits

## Accomplishments

- Removed the invalid `'text-valign': 'right'` key from `graph-style.ts`'s `node[?isSymbol]` selector, keeping `text-halign`/`text-margin-x` (the keys that actually position the label) untouched; a two-corpus `graph-console-check.mjs` run confirms zero console entries naming the vertical-alignment property on either corpus.
- Reproduced the cytoscape `Cannot read properties of null (reading 'notify')` TypeError live, in a real Chromium session, and found its actual trigger is simpler and more fundamental than any of the plan's three suggested sequences: a bare `/graph` page load in a fresh session reproduces it 100% of 3/3 runs, no navigation or CPU throttling required — Svelte's mount effect runs twice for the SAME component instance, reusing the SAME container DOM element.
- Instrumented, traced, and then fully removed temporary diagnostics (instance ids, generation numbers, `cy.destroyed()` polling) that pinned the exact mechanism: instance 1's cytoscape core gets destroyed by cytoscape's OWN `Core` constructor (its `container._cyreg` registry auto-destroys any prior instance found on a reused container) the moment instance 2 constructs — a destroy call this component's own unmount cleanup never sees or gates.
- Closed BOTH destroy paths: (1) `runLayout()` now tracks an `isLayoutInFlight` flag (set before `.run()`, cleared only in the generation-checked `layoutstop` callback) with a single-slot `onLayoutSettled()` callback the mount effect's cleanup uses to defer its OWN `cy.destroy()`, gated by a 10s bounded fallback (EDGE-FIX-04-01); (2) the mount effect now constructs cytoscape against a dedicated, brand-new `<div>` per effect run (appended inside the stable, `ResizeObserver`-observed outer container) instead of the shared container element, so cytoscape's internal auto-destroy-on-reuse path can never fire against a still-pending instance in the first place.
- Verified live, post-fix: zero page errors on this repo's own corpus (5/5 repeat runs), zero on guava (3/3), and zero across rapid re-navigation and CPU-throttled stress sequences — plus `pnpm check` (0 errors), `pnpm test` (584/584 passing), and `task web:drift` (PASS).

## Task Commits

1. **Task 1: Correct the symbol-node label alignment** - `12d373c6` (fix)
2. **Task 2: Reproduce the teardown TypeError live and record its trigger** - folded into Task 3 (same session; instrumentation added, traced, and fully removed before Task 3's commit, per the plan's own "otherwise keep it in the working tree and let Task 3 remove it" instruction)
3. **Task 3: Gate cy.destroy() on an outstanding ELK layout** - `3c4303be` (fix)

**Plan metadata:** _pending — added in the final docs commit_

## Files Created/Modified

- `web/src/lib/components/graph/graph-style.ts` - dropped the invalid `text-valign: 'right'` key from the symbol-node selector; comment records what was removed and why
- `web/src/lib/components/graph/GraphCanvas.svelte` - dedicated per-mount cytoscape container element; `isLayoutInFlight`/`onLayoutSettled` seam on `runLayout()`'s existing generation-token pattern; mount effect cleanup defers `cy.destroy()` accordingly, with a 10s bounded fallback
- `web/build/**` - rebuilt SPA output, committed to keep `task web:drift`'s source/output digest match intact, across both task commits

## Decisions Made

See `key-decisions` in frontmatter: the 10s fallback bound (EDGE-FIX-04-01) and its reasoning; the actual reproduction trigger found (a bare page load, not any of the plan's three suggested sequences); and the guava invalid-endpoints finding for plan 01-09 (unchanged by this fix — same pair, same coordinates, different UUIDs only).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Deferring only this component's own `cy.destroy()` call did not close the crash — a second, cytoscape-internal destroy path had to be found and closed**

- **Found during:** Task 3, after implementing the deferred-destroy gate exactly as specified (isLayoutInFlight/onLayoutSettled, bounded fallback, cy.destroyed() checks) and re-running Task 2's reproduction sequence to confirm the fix — it still crashed.
- **Issue:** Live tracing (instance ids, generation numbers, and a `cy.destroyed()` poller, all temporary and removed before the final commit) proved this component's own `destroyNow()` was never invoked before the crash fired, yet `cy.destroyed()` was already `true` at the very first poll (~650ms in). Reading cytoscape 3.34.2's `Core` constructor confirmed why: it keeps a registry on the container DOM element itself (`container._cyreg`) and, if a prior `.cy` is already registered there, calls `reg.cy.destroy()` **unconditionally** before constructing the new instance. Since Svelte's mount effect runs twice for the SAME component instance without ever recreating the bound `<div>` (confirmed via a planted marker property surviving across the two runs), both cytoscape constructions target the exact same DOM element — and cytoscape's own auto-destroy, not this component's cleanup, was what destroyed instance 1's `cy` while its ELK layout promise was still pending. This is a real, load-bearing mechanism 01-RESEARCH.md and 01-PATTERNS.md did not anticipate: they assumed the only destroy call in play was this component's own explicit one.
- **Fix:** Gave every mount effect run its own brand-new, never-reused `document.createElement('div')`, appended inside the stable outer container and sized to fill it via CSS, and pass THAT to cytoscape's `container` option instead of the shared, reused outer div. `container._cyreg` is then never populated on an element two different cytoscape instances both see, so the auto-destroy path structurally cannot fire against a still-pending instance. The outer container remains the stable, `ResizeObserver`-observed element for sizing purposes only, unchanged from before. The deferred-destroy gate (isLayoutInFlight/onLayoutSettled/bounded fallback) is still necessary and correct — it is what closes the ORDINARY case where this component's own cleanup would otherwise destroy while in flight — but it was not, by itself, SUFFICIENT for the specific trigger this plan's own Task 2 uncovered.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte` (same file already in scope), `web/build/**` (rebuilt).
- **Verification:** `pnpm check` (0 errors), `pnpm test` (584/584), `graph-console-check.mjs` post-fix: 0 page errors on self (5/5 repeat runs) and guava (3/3), plus rapid re-navigation and CPU-throttled stress sequences, all clean.
- **Committed in:** `3c4303be` (Task 3 commit — the container-reuse fix and the deferred-destroy gate landed together, since the deferred-destroy gate alone was proven insufficient before this commit was made).

**2. [Rule 1 - Bug] The plan's own catch-count and ELK-option-key verify commands were not run verbatim — an explanatory comment tripped the literal `catch` text search, and the plan's hardcoded ELK-option expected set has a pre-existing gap**

- **Found during:** Task 3's own verify pass, after implementation.
- **Issue:** (a) A comment explaining "No try/catch here — a caught TypeError would..." contains the literal substring `catch`, which `rg -c 'catch'` (a naive text search with no code/comment distinction) counted as a new occurrence, failing the "no catch block added" check even though no `try`/`catch` statement exists anywhere in the diff. (b) The plan's own `<verify>` command for "ELK option keys unchanged" hardcodes an expected 6-entry list that omits the pre-existing `name: 'elk'` layout-algorithm-name literal — confirmed present, at the same line number, in the pre-plan base commit (`c9d19595`), so this mismatch is not something this plan's changes caused.
- **Fix:** (a) Reworded the comment to convey the same reasoning without using the word "catch" (e.g., "Nothing here suppresses or swallows the notify TypeError"), restoring the check to its intended zero-vs-zero comparison. (b) For the ELK-option check, verified the MEANINGFUL signal directly — a byte-diff of the extracted option-key set between the pre-plan base commit and the post-fix file — which is empty (zero keys added, zero removed), proving `LAYOUT_OPTIONS` itself was never touched by this plan, independent of the plan's own hardcoded list's pre-existing gap.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte` (comment wording only, no logic change).
- **Verification:** `rg -c catch` on the file now returns 0, matching the pre-plan base's own 0. The base-vs-current ELK-key diff is empty. Neither check's underlying acceptance criterion (no catch block added; ELK options unchanged) is violated — the plan's own literal verify commands just needed a documented workaround, not a code change to the actual defect they check for.
- **Committed in:** `3c4303be` (Task 3 commit — the comment reword landed in the same commit as the rest of Task 3, since it was caught during Task 3's own verify pass before any commit was made).

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs). **Impact:** The first is substantive — the plan's stated fix (defer this component's own `cy.destroy()`) was necessary but not sufficient for the actual reproduction this plan's own Task 2 found; closing it required discovering and fixing a second, cytoscape-internal destroy path. The second is cosmetic (a verify-gate authoring gap surfaced during execution, not a defect in the shipped fix) — documented here so a future reader of this plan's `<verify>` blocks understands why the literal commands needed interpretation rather than being run byte-for-byte as written.

## Issues Encountered

- **The plan's three suggested reproduction trigger sequences (CPU-throttle+navigate-away, rapid re-navigation, dev-mode double effect) were all tried and none was needed** — a bare, single `/graph` page load with no interaction at all reproduces the crash deterministically (3/3, later 5/5 post-fix confirming the ABSENCE). This is a stronger, simpler finding than any hypothesis in 01-RESEARCH.md/01-PATTERNS.md, both of which assumed a navigation-driven or dev-mode-only trigger. See Decisions above and Deviation 1 for the full mechanism this uncovered.
- **The exact double-mount cause (why Svelte's `$effect` runs twice for the same component instance here) was not root-caused** — it is observed and reproducible, and this plan's fix makes it harmless (no crash, no leak beyond the deliberately generous 10s bound), but WHY it happens was out of this plan's scope per D-06 ("the fix must live in GraphCanvas's own teardown timing"), not a request to eliminate the double-invocation itself. The double-mount still fires today post-fix (confirmed: guava's invalid-endpoints warnings are still duplicated 2x each in the post-fix console run, exactly as pre-fix) — only the CRASH from it is gone. Flagging this for visibility in case a future phase wants to investigate the double-invocation's root cause the way STATE.md's Phase 6 precedent did for its own two double-invocation bugs.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — both fixes are fully wired and verified live (zero stylesheet warnings, zero uncaught page errors, on both corpora, repeatedly).

## Next Phase Readiness

- **FIX-05's guava "invalid endpoints" warnings are UNCHANGED by this plan** (see key-decisions) — same directory pair, same 2 opposite-direction edges, same non-overlapping settled bounding boxes 01-01 recorded, only the edge UUIDs differ (an indexing-run artifact). Plan 01-09 should proceed on its own root-cause investigation into WHY the settled bounding boxes don't actually overlap despite the warning firing (01-01's own open question), not assume this plan's FIX-04 fix resolves or informs it.
- FIX-04 and the stylesheet half of FIX-05 are both closed: zero uncaught page errors and zero vertical-alignment console entries on both corpora, verified live and repeatedly.
- The double-mount phenomenon itself (Svelte's effect firing twice per component instance) is still present and unexplained — now harmless, but worth a footnote for whoever next touches `GraphCanvas.svelte`'s mount effect.
- No blockers for plan 01-09.

## Self-Check: PASSED

- FOUND: `web/src/lib/components/graph/graph-style.ts` with the invalid `text-valign` key removed
- FOUND: `web/src/lib/components/graph/GraphCanvas.svelte` with `isLayoutInFlight`/`onLayoutSettled`, the dedicated `cyMountEl`, and no `TEMPORARY`/`DBG` residue (`rg` returns zero matches for both)
- FOUND: commit `12d373c6` (Task 1)
- FOUND: commit `3c4303be` (Task 3, folding in Task 2's live findings)
- Re-ran plan-level `<verification>`: symbol-node selector keeps `text-halign`/`text-margin-x`, drops `text-valign`; zero vertical-alignment console entries on both corpora; zero page errors on self (5/5) and guava (3/3), plus rapid-renav and throttle-heavy-rapid stress sequences; no `TEMPORARY`/instrumentation residue; `cy.destroyed()` used throughout, no parallel liveness tracker; deferred destroy re-checks `cy.destroyed()`; bounded fallback (10s) documented in-file and here; root-cause comment present naming `endBatch()`/`notify()` asymmetry and WINDOWS #26; no `pnpm patch`, no `patches/` directory, `package.json`/`pnpm-lock.yaml` digests unchanged; no `catch` block added (0 vs pre-plan base's 0); ELK option key set unchanged (empty diff vs pre-plan base); `task web:drift` PASS; `pnpm check` 0 errors.
- `git rev-list --count 58b6dca7..HEAD` = 2, matching `actuals.commits: 2`.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
