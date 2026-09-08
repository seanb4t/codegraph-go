---
phase: 04-query-workbench-index-health
plan: 06
subsystem: ui
tags: [svelte, sveltekit, tanstack-table, connectrpc, doublestar, glob, tdd]

requires:
  - phase: 04-query-workbench-index-health (plan 02)
    provides: "internal/query/files.go's recursive-glob fix (doublestar.Match crossing `/`) — the observable dependency this plan's nested-result test proves"
  - phase: 04-query-workbench-index-health (plan 04)
    provides: "AnalysisPanel.svelte's widened AnalysisResult { rows, summary } contract, the four-tab route shell, table-features.ts's shared sorting registration"
provides:
  - "web/src/lib/debounced-rpc.ts — createDebouncedRpc<TResult>: the ONE debounce + per-dispatch AbortController + monotonic request-identity mechanism, extracted from search.ts and configured twice"
  - "web/src/lib/file-search.ts — createFileSearchController, escapeGlobLiteral: arbitrary-depth Files search with typed glob metacharacters neutralized before they reach Engine.Files"
  - "web/src/lib/components/workbench/FilePicker.svelte — search-and-add with removable chips, props-in/callback-out, no internal selection state"
  - "web/src/lib/components/workbench/affected-columns.ts — the fourth *-columns.ts file"
  - "web/src/routes/workbench/+page.svelte — the Affected tab wired; all four Workbench tabs now live"
affects: [04-07]

actuals:
  tokens: 16809
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A debounce/AbortController/request-identity mechanism extracted into its own module (debounced-rpc.ts) and configured twice via a callback-shaped options object (dispatch/onResult/onFailure/onBelowMinimum), rather than sharing two exported constants across two independent implementations — D-15's 'one implementation, configured twice' resolved structurally."
    - "onBelowMinimum as the seam between mechanism and state-shaping: the extracted controller owns abort-and-invalidate (mechanism), the caller owns clearing its own visible state (shaping) — the minimum-character rule stays in exactly one place instead of being re-derived per caller."
    - "escapeGlobLiteral: user-typed text destined for a glob pattern is backslash-escaped in a single regex pass before being wrapped in the module's own structural wildcards (`**/*…*`) — the wildcards are ours, the term is theirs. Verified against the actual matcher (doublestar.Match), not assumed from library docs."
    - "FilePicker.svelte: props-in/callback-out with zero internal selection state (files in, onChange out), matching NeighborsPanel.svelte's shape — the URL stays the single source of truth for what's selected."

key-files:
  created:
    - web/src/lib/debounced-rpc.ts
    - web/src/lib/file-search.ts
    - web/src/lib/components/workbench/FilePicker.svelte
    - web/src/lib/components/workbench/affected-columns.ts
    - web/tests/debounced-rpc.test.ts
    - web/tests/file-search.test.ts
    - web/tests/workbench-affected.test.ts
  modified:
    - web/src/lib/search.ts
    - web/src/routes/workbench/+page.svelte

key-decisions:
  - "Task 1 extracted debounced-rpc.ts from search.ts's live path rather than writing a second debounce/abort/identity implementation in file-search.ts (cycle-1 review finding, HIGH) — createSearchController is not parameterisable as it stood (it hard-codes two symbol-specific RPCs), so the genuinely-common mechanism was pulled out and configured twice. web/tests/search.test.ts passes BYTE-UNCHANGED (git diff empty) as the regression proof for the refactor — a NECESSARY, not sufficient, proof per the plan's own qualification."
  - "Added onBelowMinimum() to the extracted mechanism's options (cycle-2 review finding, MEDIUM) so the caller's visible-state clear survives the extraction alongside the abort-and-invalidate mechanism it used to be bundled with. New tests in both debounced-rpc.test.ts and file-search.test.ts cover the below-minimum-while-in-flight transition that search.test.ts's existing suite does not (its min-length suite tests a one-character query issuing no request; its cancellation suite tests long-to-longer overlap, never a below-minimum term arriving mid-flight)."
  - "escapeGlobLiteral implemented as a single regex pass (`/[\\\\*?[\\]{}]/g` with a replacer distinguishing the backslash case) rather than two sequential passes, so a backslash in the term is escaped exactly once and never re-scanned by a later pass over the other metacharacters — avoids the double-processing pitfall the plan's action step warns against by construction rather than by ordering two passes correctly."
  - "The Affected tab renders FilePicker OUTSIDE the {#if files.length === 0}/{:else} split so the picker (and the ability to add the FIRST file) is always visible, while AnalysisPanel itself is only mounted once at least one file is selected — this keeps AnalysisPanel's own generic 'Enter a symbol...' idle text from ever being the empty-state message users see for a files-driven analysis; a dedicated 'Select at least one file to run Affected' paragraph renders instead, and no AnalysisPanel instance (hence no request) exists until then."
  - "AnalysisPanel.svelte and workbench-url.ts were read but NOT modified — both are out of this plan's files_modified. The Affected empty state is achieved by conditionally mounting AnalysisPanel rather than teaching it a per-caller idle message."

requirements-completed: [WRK-02, WRK-04]

coverage:
  - id: D1
    description: "The debounce/AbortController/request-identity mechanism exists in exactly ONE place (debounced-rpc.ts), configured twice (search.ts's live path, file-search.ts) — search.ts's public API, debounce interval, minimum-character rule and existing test suite are unchanged by the extraction"
    requirement: WRK-02
    verification:
      - kind: unit
        ref: "web/tests/debounced-rpc.test.ts — 10/10 passed (min-length gate, one-dispatch-per-window, coalescing with zero aborts, genuine overlap with one abort, out-of-order discard, rejection handling, dispose clearing timer + aborting in-flight, below-minimum-while-in-flight aborting+invalidating+never-firing-onResult, second below-minimum firing the callback again with no further dispatch)"
        status: pass
      - kind: unit
        ref: "web/tests/search.test.ts — 15/15 passed, `git diff web/tests/search.test.ts` empty (byte-unchanged)"
        status: pass
      - kind: other
        ref: "rg -c 'setTimeout' web/src/lib/debounced-rpc.ts -> 2; rg -n 'setTimeout' web/src/lib/search.ts web/src/lib/file-search.ts -> no match; rg -c 'createDebouncedRpc' web/src/lib/search.ts web/src/lib/file-search.ts -> 2 and 2; rg -n 'new AbortController' web/src/lib/file-search.ts -> no match, positive-controlled by rg -c 'new AbortController' web/src/lib/debounced-rpc.ts -> 1; rg -o 'SEARCH_DEBOUNCE_MS *=' web/src/lib -g '*.ts' | wc -l -> 1; rg -o 'SEARCH_MIN_CHARS *=' web/src/lib -g '*.ts' | wc -l -> 1; rg -c 'onBelowMinimum' web/src/lib/debounced-rpc.ts -> 5, web/src/lib/search.ts -> 2"
        status: pass
    human_judgment: false
  - id: D2
    description: "File search finds a result at arbitrary depth (proving 04-02's recursive-glob fix reaches the UI), cannot be corrupted by an out-of-order response or a response the user already dismissed by backspacing below the minimum, and treats a typed glob metacharacter as a literal"
    requirement: WRK-02
    verification:
      - kind: unit
        ref: "web/tests/file-search.test.ts — 15/15 passed (min-length gate, nested-result assertion, coalescing, genuine overlap, out-of-order safety, backspace-below-minimum-while-in-flight — test named containing the literal 'below the minimum while in flight', rejection handling, tree-format union discipline, 5x metacharacter escaping via it.each over `[ { * ? \\`, plain-term byte-identity)"
        status: pass
      - kind: other
        ref: "rg -c 'escapeGlobLiteral' web/src/lib/file-search.ts -> 4 (definition + comment + use); rg -c 'below the minimum while in flight' web/tests/file-search.test.ts -> 1"
        status: pass
    human_judgment: false
  - id: D3
    description: "A developer searches, adds several files as removable chips (de-duplicated, order-preserving), and the chip set round-trips through repeated file= URL parameters in both directions, including a path containing a comma"
    requirement: WRK-02
    verification:
      - kind: unit
        ref: "web/tests/workbench-affected.test.ts (picker half) — 6/6 passed: append on select, de-duplication, remove-preserves-order with an accessible name naming the path, write-direction (chips -> file= entries in order), read-direction (file= entries -> chips in order, asserted separately), comma-in-path round trip"
        status: pass
      - kind: other
        ref: "rg -n \"params\\.get\\('file'\\)|searchParams\\.get\\('file'\\)\" web/src/lib/components/workbench web/src/routes/workbench -> no match (workbench-url.ts itself carries one match, but it is a pre-existing 04-01 DOC COMMENT describing the deliberate get/getAll divergence with browse-url.ts, not code — see Deviations); positive control rg -c \"getAll\\('file'\\)\" web/src/lib/workbench-url.ts -> 1; rg -n \"join\\(','\\)|split\\(','\\)\" web/src/lib/workbench-url.ts web/src/lib/components/workbench/FilePicker.svelte -> no match; cd web && pnpm check -> 0 errors"
        status: pass
    human_judgment: false
  - id: D4
    description: "Running Affected over the selected files renders affected_tests as a sortable table with the echoed files as a header summary preceding the table (never a column, never a second request); zero files selected issues no request and renders an explicit empty state; all four Workbench tabs are live"
    requirement: WRK-04
    verification:
      - kind: unit
        ref: "web/tests/workbench-affected.test.ts (Affected half) — 8/8 passed: URL -> Affected rpc -> table with exactly one call; echoed files precede the table in document order and are absent from the <th> set (exactly ONE call recorded); GetStatus call count unchanged when adding a chip ('file' already in ROUTE_LOCAL_PARAMS, with a Browse positive control proving the exclusion is scoped); zero-chip empty state with zero requests; adding a chip issues a new request and replaces rows without remounting the route or calling goto; depth=9999 reaches the stub unbounded; Name-column sort ascending then descending; every WORKBENCH_MODES entry (iterated from the module) renders no not-yet-wired placeholder"
        status: pass
      - kind: other
        ref: "task web:test -> PASS — 257 of 257 (up from 218 at end of 04-05); cd web && pnpm check -> 1014 files, 0 errors; rg -in 'not yet wired|placeholder|coming soon|TODO' web/src/routes/workbench/+page.svelte -> no match; rg -c '<AnalysisPanel' web/src/routes/workbench/+page.svelte -> 4 (see Deviations re: the plan's literal 'AnalysisPanel' grep pattern); rg -n 'Math\\.min|Math\\.max|max=' web/src/routes/workbench/+page.svelte -> no match; GOTOOLCHAIN=go1.26.5 task test:unit -> all packages ok"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 6: Multi-file picker driving Affected Summary

**Search-and-add file picker with removable chips over one extracted debounce/abort/identity mechanism (configured twice), driving Affected's sortable table — the fourth and final Workbench tab.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3
- **Files modified:** 9 (5 created, 2 modified source; 3 test files, one of them extended across two tasks)

## Accomplishments

- Extracted `debounced-rpc.ts` from `search.ts`'s live path — the ONE debounce + per-dispatch `AbortController` + monotonic request-identity mechanism, now configured twice (symbol/file live search, and 04-06's new file picker), resolving a cycle-1 review finding that an earlier draft of this plan both forbade and mandated a second implementation.
- Added `onBelowMinimum()` to the extracted mechanism (cycle-2 finding) so a caller's visible-state clear survives the extraction; a stale response arriving after a user backspaces below the minimum can never land, proven at both the mechanism level and the file-search level.
- `file-search.ts` searches `Files` at arbitrary depth via a `**/*term*` pattern — the observable proof that 04-02's recursive-glob fix reaches this second caller — with `escapeGlobLiteral` neutralizing every typed glob metacharacter (`\ * ? [ ] { }`) before it reaches `Engine.Files`, closing a cycle-1 HIGH finding about unescaped user text in a glob.
- `FilePicker.svelte` composes the vendored Command primitive over `file-search.ts`, props-in/callback-out with zero internal selection state — search, add, de-duplicate, remove, all driving `workbench-url.ts`'s `files` array as the single source of truth, round-tripping through repeated `file=` URL parameters (including a comma-containing path) in both directions.
- The Affected tab is wired: `AffectedRequest.files` from the picker, `affected_tests` rendered as a sortable table via a fourth `AnalysisPanel` call site, echoed `files` rendered as a header summary preceding the table through 04-04's widened `AnalysisResult` contract (never a second request). All four Workbench tabs are now live.

## Task Commits

1. **Task 1: Extract the ONE debounced-RPC controller, then configure it for Files** - `fc27fdf0` (feat, tdd)
2. **Task 2: FilePicker.svelte — search-and-add with removable chips bound to the URL** - `0431cc8c` (feat, tdd)
3. **Task 3: Wire the Affected tab and close the four-tab surface** - `388d1fe5` (feat, tdd)

_Each task's RED tests were written and observed passing on first GREEN run against the accompanying implementation — no separate red-then-fix cycle was needed for any of the three tasks._

## Files Created/Modified

- `web/src/lib/debounced-rpc.ts` - `createDebouncedRpc<TResult>`: the extracted mechanism
- `web/src/lib/file-search.ts` - `createFileSearchController`, `escapeGlobLiteral`, `FilesClient`
- `web/src/lib/search.ts` - live path refactored onto `createDebouncedRpc`; public API, debounce interval, minimum-character rule, `dispose()` semantics unchanged
- `web/src/lib/components/workbench/FilePicker.svelte` - search-and-add chip picker
- `web/src/lib/components/workbench/affected-columns.ts` - the fourth `*-columns.ts` file
- `web/src/routes/workbench/+page.svelte` - Affected tab wired; placeholder removed
- `web/tests/debounced-rpc.test.ts` - mechanism-level test suite (new)
- `web/tests/file-search.test.ts` - file-search controller test suite (new)
- `web/tests/workbench-affected.test.ts` - picker half (Task 2) + Affected-tab half (Task 3)

## Decisions Made

See `key-decisions` in frontmatter. Summary: extraction over duplication for the debounce mechanism (cycle-1), `onBelowMinimum` as the mechanism/state-shaping seam (cycle-2), single-pass regex escaping to avoid double-processing, and mounting `FilePicker` outside the empty-state branch so the picker is always available while `AnalysisPanel` (and its request) only exists once files are selected.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TypeScript inference on untyped `vi.fn()` mocks in debounced-rpc.test.ts**
- **Found during:** Task 1, `pnpm check` after the first GREEN test run
- **Issue:** Two `vi.fn(() => Promise.resolve('x'))` mocks with no explicit parameter types caused `mock.calls[0]` to be inferred as an empty tuple (`[]`), producing three TS errors ("Tuple type '[]' of length '0' has no element at index...") when later indexed for assertions.
- **Fix:** Added explicit `(_term: string, _signal: AbortSignal)` parameter types to the two affected `vi.fn()` calls.
- **Files modified:** `web/tests/debounced-rpc.test.ts`
- **Verification:** `cd web && pnpm check` — 0 errors (was 3); test suite re-run, still 38/38 passing.
- **Committed in:** `fc27fdf0` (Task 1 commit)

**Observed discrepancies in the plan's literal acceptance-criteria grep patterns (documented, not code changes):**

- Task 2's acceptance criterion `rg -n "params\.get('file')|searchParams\.get('file')" ... web/src/lib/workbench-url.ts` reports no match" in fact finds ONE match — a pre-existing 04-01 DOC COMMENT (`workbench-url.ts:7`, "single-value `params.get('file')` that silently drops every occurrence after the first") that explains the deliberate `get`/`getAll` divergence with `browse-url.ts`. This is prose describing D-11, not code, predates this plan, and `workbench-url.ts` is not in this plan's `files_modified` — left as-is. All actual READS of the `file` key in code (`workbench-url.ts`'s own `parseWorkbenchParams`, `FilePicker.svelte`) use `getAll` exclusively, confirmed by the criterion's own positive control.
- Task 3's acceptance criterion "`rg -c 'AnalysisPanel' web/src/routes/workbench/+page.svelte` reporting exactly 4" is unsatisfiable as literally written against this file at any point in the phase — the 04-04 baseline (before this plan touched the file) already reports 9 matching lines for the bare substring `AnalysisPanel` (doc comments, the import statement, three JSX-like tags). The narrower, satisfiable, and clearly-intended reading is `rg -c '<AnalysisPanel'` (the actual component-usage tag): 3 at the 04-04 baseline, 4 after this plan wires Affected — matching the plan's own prose ("a fourth call site"). Verified against that reading; reported here rather than silently substituted.

---

**Total deviations:** 1 auto-fixed (Rule 1, type inference), 2 documented grep-pattern discrepancies (no code impact — both are pre-existing/pre-scoped facts about the acceptance criteria's literal wording, not defects in the implementation).
**Impact on plan:** None on scope or behavior. The auto-fix was a pure type-annotation addition with no runtime effect. Both grep discrepancies were verified against their evident intent (confirmed via git history for the AnalysisPanel case) and the underlying properties they exist to check hold.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None. Every file created or modified in this plan is wired to a real data source (the file-search controller's live RPC, the Affected RPC, the URL's `files` array) — no hardcoded empty values, no placeholder text, no unwired props.

## Threat Flags

None beyond the plan's own `<threat_model>` table (T-04-22 through T-04-26), all of which are addressed by this implementation as specified (escapeGlobLiteral for T-04-22, the extracted debounce/abort/identity mechanism for T-04-23, repeated `file=` keys read via `getAll` for T-04-24).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All four Workbench tabs (Impact, Affected, Callers, Callees) are now live — WRK-01 through WRK-04 complete for this milestone.
- `task web:drift` remains EXPECTED RED (04-01 opened this window; 04-07 closes it) — not touched by this plan, per its own scope.
- No blockers for 04-07.

## Self-Check: PASSED

All 10 files created/modified by this plan verified present on disk; all 3 task commits (`fc27fdf0`, `0431cc8c`, `388d1fe5`) verified present in git history.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-30*
