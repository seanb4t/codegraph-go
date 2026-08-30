---
phase: 04-query-workbench-index-health
plan: 05
subsystem: ui
tags: [svelte, sveltekit, tanstack-table, health-endpoint, connectrpc, tdd]

requires:
  - phase: 04-query-workbench-index-health (plan 01)
    provides: "the generic DataTable shell with a getRowId prop, table-features.ts's shared sorting registration, WORKBENCH_PARAM_KEYS/ROUTE_LOCAL_PARAMS, workbench-failure.ts's describeWorkbenchFailure taxonomy"
  - phase: 04-query-workbench-index-health (plan 03)
    provides: "GetHealth, the frozen 16-field GetHealthResponse (WorktreeMismatch/PendingChanges/IndexHealth), on the wire"
provides:
  - "web/src/lib/status.ts — IndexStatus widened additively with commitSha: string, carried alongside the existing known/unknown presence flag"
  - "web/src/lib/health-view.ts — toCountRows, hasWorktreeMismatch, describeFreshness (with snapshotAgreement), HealthClient"
  - "web/src/lib/components/health/{TrustVerdict,WorktreeMismatchWarning,CountTable}.svelte"
  - "web/src/routes/health/+page.svelte — filled: worktree warning, trust verdict, freshness block, three count tables"
affects: [04-07]

actuals:
  tokens: 10900
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A wire field duplicated onto a client-side classifier's output additively (IndexStatus.commitSha), never folded into an existing two-member presence flag or a five-member verdict — the widened field is compared, the presence flag stays a presence flag, and pnpm check plus a positive-controlled grep together prove every construction site (including one hidden from the compiler by context-Map bivariance) was updated."
    - "A second, independently-fetched snapshot (GetHealth) is reconciled against the shared per-navigation gate's verdict by a NAMED comparison field (snapshotAgreement) rather than a second verdict or a silent blend — the losing snapshot's data is never suppressed, only flagged."
    - "A presentational component that renders ALL branches of an existing shared enum (TrustVerdict renders all five StatusVerdict members, unlike StatusBanner which suppresses two) to answer a different question (\"should I trust this\" vs \"should I warn you\") from the SAME classification, with no second classifier."

key-files:
  created:
    - web/src/lib/health-view.ts
    - web/src/lib/components/health/TrustVerdict.svelte
    - web/src/lib/components/health/WorktreeMismatchWarning.svelte
    - web/src/lib/components/health/CountTable.svelte
    - web/tests/health-view.test.ts
    - web/tests/health-page.test.ts
  modified:
    - web/src/lib/status.ts
    - web/src/routes/+layout.svelte
    - web/src/routes/browse/+page.svelte
    - web/src/lib/components/workbench/AnalysisPanel.svelte
    - web/src/routes/health/+page.svelte
    - web/tests/status.test.ts
    - web/tests/degrade-states.test.ts
    - web/tests/browse-page.test.ts
    - web/tests/workbench-failure.test.ts

key-decisions:
  - "IndexStatus widened by exactly one additive field (commitSha: string, '' = not recorded) per the plan's cycle-2 disposition — StatusVerdict stays at five members, CommitKnowledge stays at two, both asserted as whole-line literal greps. snapshotAgreement compares status.commitSha against GetHealth's own commitSha, never the known/unknown presence flag."
  - "Two construction sites beyond the plan's named four were found and fixed: AnalysisPanel.svelte:80 (a literal IndexStatus initializer introduced by the sibling 04-04 plan after 04-05-PLAN.md was written — the plan's own 'if a fifth site appears' allowance covers this) and web/tests/workbench-failure.test.ts's status() helper (an explicit-return-type function that would otherwise fail pnpm check). Both are Rule 3 blocking-build fixes, not scope creep — pnpm check would not reach 0 errors without them."
  - "HLT-03's getComputedStyle loudness check was attempted first and found unable to resolve a class-derived value: health-page.test.ts renders HealthPage directly (never +layout.svelte), so app.css/Tailwind's compiled color rules are never loaded into jsdom, and getComputedStyle resolves the browser's unstyled default for every element regardless of its Tailwind classes. Verified experimentally (a throwaway probe test) before falling back to the plan's own documented alternative: asserting the warning's class list differs from TrustVerdict's, with the reason recorded in the test's comment rather than silently dropped."
  - "The freshness block's pendingChanges rendering reads GetHealthResponse.pendingChanges directly in the route rather than routing it through describeFreshness — the plan's <behavior> block for describeFreshness names only verdict/commitSha/schemaVersion/reindexRecommended as its return fields, and PendingChanges is already a self-contained wire struct needing no derivation."

requirements-completed: [HLT-01, HLT-02, HLT-03]

coverage:
  - id: D1
    description: "IndexStatus carries the indexed commit SHA alongside (never instead of) its known/unknown presence flag; the five verdicts and the presence flag are provably unchanged by the widening"
    requirement: HLT-02
    verification:
      - kind: unit
        ref: "web/tests/status.test.ts — 22/22 passed (16 pre-existing + 6 new: SHA travels through classifyStatus for both a populated and an empty SHA, the presence-flag/SHA-emptiness invariant, and the gate's emit carries a changed SHA to subscribers)"
        status: pass
      - kind: other
        ref: "rg -c \"^export type StatusVerdict = 'ok' | 'stale' | 'no-index' | 'indexing' | 'unknown';$\" web/src/lib/status.ts -> 1; rg -c \"^export type CommitKnowledge = 'known' | 'unknown';$\" web/src/lib/status.ts -> 1; rg -c 'commitSha: string' web/src/lib/status.ts -> 1 (positive control)"
        status: pass
    human_judgment: false
  - id: D2
    description: "health-view.ts's projections are pure, tested functions that consume the one verdict rather than computing a second one — toCountRows, hasWorktreeMismatch and describeFreshness"
    requirement: HLT-01
    verification:
      - kind: unit
        ref: "web/tests/health-view.test.ts — 12/12 passed (sort order, empty/undefined maps, blank-roots mismatch is false, verdict never recomputed even when indexHealth.reindexRecommended is true, all three snapshotAgreement outcomes)"
        status: pass
      - kind: other
        ref: "rg -n 'classifyStatus|initialized|storeExists|indexingInProgress' web/src/lib/health-view.ts -> no match, positive-controlled by rg -c 'IndexStatus' -> 7; rg -n 'status\\.commit\\b' -> no match, positive-controlled by rg -c 'commitSha' -> 7"
        status: pass
    human_judgment: false
  - id: D3
    description: "Opening /health issues exactly one GetHealth call and renders, in document order, the worktree warning (when present), the trust verdict, then the numeric blocks; a GetHealth failure renders a named failure state while the verdict still renders"
    requirement: HLT-02
    verification:
      - kind: component
        ref: "web/tests/health-page.test.ts#HLT-02 — compareDocumentPosition proves health-verdict-stale precedes health-freshness; #one call — exactly one getHealth invocation, zero getStatus calls; #failure path — workbench-failure-server-error renders with health-verdict-ok still present and health-freshness absent"
        status: pass
      - kind: other
        ref: "cd web && pnpm check -> 1007 files, 0 errors; rg -c 'createStatusGate' web/src/routes/health/+page.svelte -> 0 (no match), positive-controlled by the same grep on +layout.svelte -> 2; rg -n 'setInterval|setTimeout|requestAnimationFrame|setImmediate' web/src/routes/health web/src/lib/components/health -> no match"
        status: pass
    human_judgment: false
  - id: D4
    description: "A worktree mismatch renders WorktreeMismatchWarning with role=alert, both roots named, preceding the verdict; a clean repository renders NO such element — both directions asserted"
    requirement: HLT-03
    verification:
      - kind: component
        ref: "web/tests/health-page.test.ts#HLT-03 presence/order — role=alert, both roots in text, precedes health-verdict-ok; #HLT-03 absence — queryByTestId returns null for a nil worktreeMismatch"
        status: pass
    human_judgment: false
  - id: D5
    description: "HLT-03's 'impossible to miss' loudness is asserted by a distinct, loaded style hook where the test environment supports it, or an explicit documented fallback where it does not"
    requirement: HLT-03
    verification:
      - kind: component
        ref: "web/tests/health-page.test.ts#HLT-03 loudness — getComputedStyle was tried and found to resolve the unstyled jsdom default (no app.css loaded by this test file); falls back to a class-list-differs assertion between the warning and TrustVerdict's 'ok' branch, with the reason recorded in the test's own comment per the plan's documented fallback"
        status: pass
    human_judgment: false
  - id: D6
    description: "A CountTable per count listing (per-language file counts, node counts by kind, edge counts by kind) binds the ONE generic DataTable shell at CountRow via a getRowId prop — no second table implementation"
    requirement: HLT-01
    verification:
      - kind: other
        ref: "cd web && pnpm check -> 0 errors (the concrete proof CountTable binds the generic DataTable at CountRow); rg -c 'getRowId' web/src/lib/components/health/CountTable.svelte -> 3; rg -n '<table' web/src/lib/components/health/*.svelte -> no match"
        status: pass
    human_judgment: false
  - id: D7
    description: "A snapshot disagreement between the shared gate's commit and GetHealth's own commit surfaces a visible health-snapshot-differs notice without suppressing the numbers or recomputing the verdict — asserted in BOTH directions (present when they differ, absent when they agree)"
    requirement: HLT-02
    verification:
      - kind: unit
        ref: "web/tests/health-view.test.ts — all three snapshotAgreement outcomes (agree/differs/unknown) asserted, including a case where status.commit='known' but the SHAs genuinely match, proving the comparison reads commitSha and not the presence flag"
        status: pass
      - kind: component
        ref: "web/tests/health-page.test.ts#snapshot disagreement — health-snapshot-differs present with both the verdict and health-freshness still rendered when SHAs disagree; ABSENT when SHAs agree"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 5: The /health View Summary

**`/health` now answers "should I trust this index" with a single trust verdict above the raw numbers, a loud worktree-mismatch warning, per-language/per-kind counts, and a snapshot-disagreement notice — all from one `GetHealth` call and the one shared verdict, with zero second classifiers.**

## Performance

- **Duration:** 55 min
- **Tasks:** 3 completed
- **Files modified:** 15 (6 created, 9 modified)

## Accomplishments

- Widened `IndexStatus` additively with `commitSha: string`, fixing the cycle-2 defect where `snapshotAgreement` compared a two-member presence flag against a 40-character SHA (type-impossible as previously specified). `StatusVerdict` (5 members) and `CommitKnowledge` (2 members) are unchanged, asserted as whole-line literal greps.
- Built `health-view.ts`'s pure projection layer (`toCountRows`, `hasWorktreeMismatch`, `describeFreshness`) — computes no verdict of its own, downstream of the one classifier in `status.ts` (D-04).
- Filled `/health`: `TrustVerdict.svelte` (all five verdict branches), `WorktreeMismatchWarning.svelte` (`role="alert"`, both host-absolute roots named), three `CountTable.svelte` instances over the ONE generic `DataTable` shell, composed in document order — warning, verdict, freshness block, tables.
- Asserted the three claims that look manual and are not: HLT-02's DOM-order precedence via `compareDocumentPosition`, HLT-03's presence/absence/loudness in both directions, and HLT-01's single-render completeness — all in `web/tests/health-page.test.ts`.

## Task Commits

1. **Task 1: Widen IndexStatus, build health-view.ts's pure projections** - `158556e9` (feat)
2. **Task 2: Fill /health — verdict above the numbers, worktree warning first** - `5919d530` (feat)
3. **Task 3: Assert the ordering, the loudness, and the absence** - `f110659d` (test)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — blocking build issue] A fifth `IndexStatus` construction site, `AnalysisPanel.svelte:80`, was not in the plan's named four.**
- **Found during:** Task 1, running `pnpm check` after the widening.
- **Cause:** `AnalysisPanel.svelte` is a sibling 04-04 file that landed in this session, after `04-05-PLAN.md` was written naming exactly four sites. Its `let indexStatus: IndexStatus = $state({ verdict: 'unknown', commit: 'unknown' });` is a direct, typed literal — `pnpm check` fails on it once `commitSha` becomes required.
- **Fix:** added `commitSha: ''` to the literal. The plan itself anticipated this exact possibility ("If a fifth site appears, update it and record it in the SUMMARY").
- **Files modified:** `web/src/lib/components/workbench/AnalysisPanel.svelte`.
- **Verification:** `pnpm check` — 0 errors, both before and after confirmed via a scratch run.

**2. [Rule 3 — blocking build issue] `web/tests/workbench-failure.test.ts`'s `status()` helper has an explicit `IndexStatus` return type and would fail `pnpm check` once `commitSha` became required.**
- **Found during:** Task 1, same `pnpm check` pass.
- **Cause:** the helper's declared return type (`IndexStatus`) makes this a compiler-visible site, unlike the bivariance-hidden structural stubs the plan explicitly scoped out (`workbench-tracer.test.ts`, `workbench-callers-callees.test.ts`, `workbench-impact.test.ts`, all of which use a bare `{ verdict: string; commit: string }` param type through an untyped context `Map` and so remain unaffected and untouched).
- **Fix:** added `commitSha: ''` to the returned literal. No assertion in the file changed.
- **Files modified:** `web/tests/workbench-failure.test.ts`.
- **Verification:** `pnpm check` — 0 errors; `pnpm exec vitest run tests/workbench-failure.test.ts` unaffected.

### Test correction (found during first GREEN run, not a plan deviation)

The gate-emits-widened-value test (`status.test.ts`) initially asserted the FIRST `commitSha` a subscriber observes is the first fetch's result — but `StatusGate.subscribe` fires synchronously with the pre-fetch `UNKNOWN_STATUS` value at subscription time (its own documented contract), so the true sequence is `['', firstFetchSha, secondFetchSha]`, not `[firstFetchSha, secondFetchSha]`. Corrected the test to assert against the actual three-value sequence rather than changing the gate's behavior — a genuine test bug (Rule 1), not a product defect.

No other deviations. All plan-specified behavior, prohibitions and thresholds were met without further amendment.

## Verification

- `cd web && pnpm check` — 1007 files, 0 errors, 0 warnings.
- `task web:test` — `PASS — 218 of 218 tests passed` (up from 194 at the end of 04-04).
- `web/tests/health-view.test.ts` — 12/12 (floor 8).
- `web/tests/status.test.ts` — 22/22 (floor 18; 16 pre-existing + 6 new).
- `web/tests/health-page.test.ts` — 9/9 (floor 9).
- `rg -o 'export function classifyStatus' web/src/lib -g '*.ts' | wc -l` -> 1; `rg -l 'StatusVerdict =' web/src/lib -g '*.ts'` -> only `status.ts`.
- `task web:drift` remains EXPECTED RED per the phase's own verification note (04-01 opened the window, 04-07 closes it) — not run/fixed here, per instruction.

## Known Stubs

None. Every branch of `/health` renders live data from `GetHealth`; the loading and failure states are named, deliberate states, not stubs.

## Threat Flags

None. All new surface (`WorktreeMismatch`'s host-absolute paths, the `health-snapshot-differs` notice's provenance) was already registered in the plan's own threat model (T-04-17 through T-04-22) and mitigated as specified — no new surface outside that register was introduced.

## Self-Check

- `web/src/lib/status.ts` — FOUND
- `web/src/lib/health-view.ts` — FOUND
- `web/src/lib/components/health/TrustVerdict.svelte` — FOUND
- `web/src/lib/components/health/WorktreeMismatchWarning.svelte` — FOUND
- `web/src/lib/components/health/CountTable.svelte` — FOUND
- `web/src/routes/health/+page.svelte` — FOUND (modified, no longer the placeholder)
- `web/tests/health-view.test.ts` — FOUND
- `web/tests/health-page.test.ts` — FOUND
- Commit `158556e9` — FOUND in `git log`
- Commit `5919d530` — FOUND in `git log`
- Commit `f110659d` — FOUND in `git log`

## Self-Check: PASSED

All 8 claimed files verified present on disk; all 3 claimed commits verified present in `git log`. No missing items.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `/health` is fully filled; HLT-01, HLT-02 and HLT-03 are all satisfied and asserted in code, not left as manual checklist items.
- `IndexStatus`'s widening (`commitSha`) is available to any future consumer needing to compare snapshot freshness against the shared gate.
- No blockers for 04-06 or 04-07. `task web:drift` stays expected-red until 04-07 rebuilds the committed bundle, unaffected by this plan.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-30*
