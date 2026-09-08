---
phase: 03-browse-inspect-navigation
plan: 07
subsystem: ui
tags: [svelte, sveltekit, url-state, history, navigation, tdd]

requires:
  - phase: 03-browse-inspect-navigation (plan 03-04)
    provides: "browse-url.ts's parseBrowseParams/serializeBrowseParams (the one URL grammar) and browse-state.ts's load-seam shape, both extended (never duplicated) by this plan"
  - phase: 03-browse-inspect-navigation (plan 03-06)
    provides: "SearchPanel.svelte and the PLACEHOLDER-03-07 marker naming this plan as its intended replacement"
provides:
  - "web/src/lib/browse-nav.ts — the ONE URL writer for the Browse view: createBrowseNavigator(gotoFn), NAV_INTENT.NAVIGATE/REFINE, symbol/file/line target-kind clearing with the disambiguation-triple exception"
  - "web/src/lib/browse-state.ts extensions — single-def/multi-def BrowseTargetState variants, loadBlastRadius (Impact rpc), NavigationGeneration/createNavigationGate (the cross-load-race guard)"
  - "web/src/lib/components/browse/NeighborsPanel.svelte — callers/callees/blast-radius regions, click-through and depth-control wired through NAV_INTENT"
  - "the browse route's URL-as-state contract: every navigate/refine write goes through browse-nav.ts; +page.svelte holds no view state of its own"
affects: [03-08, 03-09]

actuals:
  tokens: 14926
  tasks: 3
  commits: 8

tech-stack:
  added: []
  patterns:
    - "hasOwn(delta, key) distinguishes 'delta explicitly sets this key to undefined' (remove it) from 'delta does not mention this key at all' (leave it), the load-bearing distinction browse-nav.ts's whole delta-merge semantics depend on — a plain `!== undefined` check cannot tell these apart"
    - "Type-only import of goto's own type (`import type { goto } from '$app/navigation'; export type GotoFn = typeof goto;`) instead of hand-typing the options shape — the module stays free of any SvelteKit runtime dependency (createBrowseNavigator takes the function as a parameter) while never drifting from the installed package's real signature"
    - "NavigationGate: one monotonically-increasing generation minted ONCE per URL change by the caller (never by a loader) and stamped on every load started for that URL — closes the cross-load race a per-call AbortController alone cannot (a response already in flight can still arrive after abort fires)"
    - "SourceRender as a nested, OPTIONAL object on the single-def state variant (unlike the flattened, always-present 'file' variant's fields) — because a single-def candidate's source genuinely CAN be absent (not gathered), a case the file variant never has"

key-files:
  created:
    - web/src/lib/browse-nav.ts
    - web/src/lib/components/browse/NeighborsPanel.svelte
    - web/tests/browse-nav.test.ts
    - web/tests/browse-state.test.ts
    - web/tests/browse-history.test.ts
    - web/tests/neighbors-panel.test.ts
  modified:
    - web/src/lib/browse-state.ts
    - web/src/lib/components/browse/SourcePane.svelte
    - web/src/lib/components/browse/SearchPanel.svelte
    - web/src/routes/browse/+page.svelte

key-decisions:
  - "Symbol/file/line target-kind clearing rule, fully specified beyond the plan's prose: setting symbol without file clears file+line; setting file without symbol clears symbol (and line too, unless the SAME delta also sets line — the 'open this file at this line, no symbol' case); setting BOTH symbol and file in one delta (the disambiguation triple, or any caller-directed pair) clears nothing, because the caller named both target-kind fields itself. Chosen because every click-through call site in this phase that has BOTH a symbol and a file (neighbour clicks, symbol search selections) produces the full symbol+file+line triple by construction, so the ambiguous partial case (symbol+file, no line) never arises from a real call site in this plan."
  - "NavigationGeneration is a plain module-scoped counter (createNavigationGate() returning advance()/isCurrent()) rather than anything tied to $app/state or Svelte runes — kept this way specifically so the cross-load race proof (browse-state.test.ts's 'one generation governs both loads' test) could be written and RED-proven with no SvelteKit runtime at all, matching this plan's own read_first note that Task 2 stays testable without one."
  - "SourcePane.svelte extended for single-def AND multi-def rendering even though neither file is in the plan's declared files_modified list (Rule 3 — Task 2's own `pnpm check` acceptance criterion broke the moment BrowseTargetState grew a union member SourcePane's bare `{:else}` didn't handle; Rule 2 — the plan's own must-haves and success criteria require source+callers+callees+blast-radius all visible together for an opened symbol, and no other file in this plan renders single-def source). Multi-def gets a minimal placeholder only — full picker rendering is 03-08's job, recorded as a stub below."
  - "Typing in search now writes q into the URL via a REFINE intent (SearchPanel.svelte's new optional onQueryChange prop, wired in +page.svelte) — not in the plan's task-level <behavior> bullets, but required by the plan's own must_haves ('typing in search... REPLACE the current entry') and D-11. Caught live during the mandated manual UAT, not by any automated test — see Deviations."

patterns-established:
  - "Invert-then-restore TDD substitution, applied a SECOND time in this plan (browse-history.test.ts, after 03-04/03-06 established the pattern): when a task's own new test passes immediately against an EARLIER task's already-correct implementation in the same plan, temporarily reintroduce the exact defect class the test guards against, observe the specific assertion fail (here: also cascading into the earlier task's own intent tests, which is itself evidence the shared implementation is truly load-bearing), restore byte-identically, re-run GREEN."

requirements-completed: [BRW-02, BRW-03, NAV-01, NAV-02]

coverage:
  - id: D1
    description: "Opening a symbol renders its verbatim source together with callers, callees and blast radius in one view; clicking any caller/callee/blast-radius entry opens that node and the view continues from there (BRW-02, BRW-03)"
    requirement: BRW-02
    verification:
      - kind: unit
        ref: "web/tests/neighbors-panel.test.ts (three labelled regions with entries in supplied order, click -> NAVIGATE intent with the symbol/file/line triple, empty-callers explicit text, failed-vs-empty blast-radius text distinguishability)"
        status: pass
      - kind: unit
        ref: "web/tests/browse-state.test.ts (single-def state carries node/calls/calledBy/source; empty-calls list still classified single-def; loadBlastRadius depth passthrough and response-echoed depth)"
        status: pass
      - kind: manual_procedural
        ref: "live codegraph ui against this repo's own index via agent-browser: opened resolveSourcePath (internal/query/node.go:33) — source rendered with visible syntax colour, 'No callers.' shown (genuinely zero callers), one callee (invalidArgumentf) and one blast-radius entry (itself) rendered; clicked through 5 neighbours in sequence (resolveSourcePath -> invalidArgumentf -> Search -> validateLimit -> Callees), each click updating the URL to the clicked target's symbol+file+line triple — screenshot uat_symbol_view.png"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every reachable view state (target, depth, limit) is encoded in the URL via ONE writer (browse-nav.ts), never the shallow-routing pushState/replaceState pair; a copied URL reconstructs the same view including depth (NAV-01)"
    requirement: NAV-01
    verification:
      - kind: unit
        ref: "web/tests/browse-nav.test.ts (11 cases: navigate/refine -> replaceState boolean and that the two differ, noScroll/keepFocus always set, delta preserves unrelated/unknown params, a value set to undefined is removed, symbol/file/line target-kind clearing with the disambiguation-triple exception, byte-identical serialization via serializeBrowseParams)"
        status: pass
      - kind: manual_procedural
        ref: "set blast-radius depth to 3 on the validateLimit view (URL grew depth=3), copied that exact URL, opened it in a FRESH tab via agent-browser: search box, source pane, callers/callees and the depth spinbutton (showing '3') all reconstructed identically, and the blast-radius list visibly grew from 1 entry (default depth) to 7 entries at depth 3 — screenshot uat_depth_reconstruction.png"
        status: pass
      - kind: other
        ref: "shallow-routing prohibition, discriminated against two planted violations (single-line and -U-wrapped) then restored to 0/0: conjunct A (quoted import, -U) = 0, conjunct B (\\bpushState\\b, wrap-proof) = 0, positive control (from '$app/navigation') >= 1 — held after every task in this plan"
        status: pass
    human_judgment: false
  - id: D3
    description: "Opening a node, clicking a neighbour and selecting a search result each PUSH a history entry; typing in search and adjusting depth/limit each REPLACE it — back/forward walk the pushed states in order and skip refinements (NAV-02)"
    requirement: NAV-02
    verification:
      - kind: integration
        ref: "web/tests/browse-history.test.ts, the AUTOMATED half of NAV-02 (URL<->state contract over a REAL jsdom history stack): A->B->C push then one back() yields B's parsed params exactly, forward() yields C's; and the push/replace STACK-DEPTH proof — a refine between B and C means one back() from C yields the refinement B' (not bare B), and a SECOND back() yields A, discriminating correct wiring [A,B',C] from wrong wiring [A,B,B',C], which the first back alone cannot"
        status: pass
      - kind: manual_procedural
        ref: "the MANUAL half (that a real popstate re-derives page.url and re-renders — needs a real router, stays manual by design): (1) clicked through 5 neighbours, pressed back twice, landed exactly on the 3rd state (Search) with the address bar matching; pressed forward twice, landed exactly on the 5th (Callees); (2) typed a multi-character query ('traverse depth') on top of the 5th state, then pressed back ONCE: landed on the 4th PUSHED state (validateLimit), not on any partial one-character-back state — proving the refine was replaced away entirely, never stepped through"
        status: pass
    human_judgment: false
  - id: D4
    description: "One NavigationGeneration governs both the node-detail load and the blast-radius load, so two loads started from two different URL states can never be combined into one rendered view"
    requirement: BRW-02
    verification:
      - kind: unit
        ref: "web/tests/browse-state.test.ts#'resolving A LAST after B still commits ONLY state Bs node detail and blast radius' — resolves B's two loads first, then A's node-detail LAST of all four, asserting BOTH committed halves are B's, by value"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 7: Navigation, Neighbours and History Summary

**One URL writer (`browse-nav.ts`) turns the tracer's single file view into a full click-through browse loop — a symbol's source, callers, callees and blast radius render together, every neighbour click and search selection pushes history, typing and depth adjustments replace it, and a `NavigationGeneration` gate stops a stale symbol's node-detail from ever being paired with a different symbol's blast radius.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-08-28T22:50:11-04:00 (Task 1 RED commit)
- **Completed:** 2026-08-29T03:05:17Z
- **Tasks:** 3
- **Files modified:** 10 (6 created, 4 modified)

## Accomplishments

- `web/src/lib/browse-nav.ts` — the one URL writer (D-11): `createBrowseNavigator(gotoFn)` takes goto's own type (never the shallow-routing history exports) as an injected parameter; `NAV_INTENT.NAVIGATE`/`REFINE` name the push/replace choice as a value; symbol/file/line target-kind clearing with the disambiguation-triple exception.
- `web/src/lib/browse-state.ts` extended (not duplicated): single-def and multi-def `BrowseTargetState` variants selected purely by the response's `mode` field; `loadBlastRadius` sends `depth` verbatim (including above the server's ceiling) and reports the response's own echoed depth, never the request's raw value; `NavigationGeneration`/`createNavigationGate` close the cross-load race a per-call `AbortController` alone cannot.
- `web/src/lib/components/browse/NeighborsPanel.svelte` — callers, callees and blast-radius regions; every entry click and depth-control change routes back through the caller-supplied `onNavigate`, stating its own intent.
- `web/src/routes/browse/+page.svelte` — replaces 03-06's direct-state-assignment placeholder with the real navigator; mints one `NavigationGeneration` per URL change; renders `NeighborsPanel` alongside `SourcePane` for a single-def target; wires typing-in-search as a `REFINE` write (deviation, see below).
- `web/src/lib/components/browse/SourcePane.svelte` — extended (deviation) to render single-def source (deriving highlight language from `Node.language`, which the FILE mode never carries) and a minimal multi-def placeholder.
- Four new test files: `browse-nav.test.ts`, `browse-state.test.ts`, `browse-history.test.ts`, `neighbors-panel.test.ts` — 73/73 tests passing (up from 44 at 03-06's close).

## Task Commits

Each task followed RED-GREEN TDD discipline:

1. **Task 1: The navigation module — one URL writer, two intents**
   - `836aadf7` — `test(03-07): add failing test for browse navigation module` (RED: `Failed to resolve import "$lib/browse-nav"`)
   - `5aaf5f29` — `feat(03-07): implement browse-nav.ts — one URL writer, two intents` (GREEN: 56/56)
2. **Task 2: Symbol targets and blast radius in the load seam**
   - `36af1a18` — `test(03-07): add failing test for symbol targets and blast radius load seam` (RED: 9/9 new tests fail — `loadBlastRadius`/`createNavigationGate` not functions)
   - `6ea7cbc7` — `feat(03-07): implement symbol targets, blast radius, and navigation generation` (GREEN: 65/65)
3. **Task 3: The neighbours panel, the click-through wiring, and the history proof**
   - `edc6e0f1` — `test(03-07): add failing test for neighbours panel rendering and the automated history proof` (RED for neighbors-panel.test.ts; browse-history.test.ts passed immediately — documented substitution below)
   - `1ffd8e6e` — `feat(03-07): wire neighbours panel, click-through navigation, and blast radius into the browse route` (GREEN: 73/73)
   - `354fb816` — `fix(03-07): wire typing-in-search as a REFINE URL write (D-11)` (deviation, caught in manual UAT)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/src/lib/browse-nav.ts` — the URL writer
- `web/src/lib/browse-state.ts` — extended: single-def/multi-def states, loadBlastRadius, NavigationGeneration
- `web/src/lib/components/browse/NeighborsPanel.svelte` — callers/callees/blast-radius regions
- `web/src/lib/components/browse/SourcePane.svelte` — extended: single-def source rendering, multi-def placeholder (deviation)
- `web/src/lib/components/browse/SearchPanel.svelte` — doc comment freshened; new optional `onQueryChange` prop (deviation)
- `web/src/routes/browse/+page.svelte` — real navigation wiring, NavigationGate, query-refine wiring
- `web/tests/browse-nav.test.ts`, `web/tests/browse-state.test.ts`, `web/tests/browse-history.test.ts`, `web/tests/neighbors-panel.test.ts` — new test coverage

## Decisions Made

See `key-decisions` in frontmatter. Summary: the full symbol/file/line clearing rule beyond the plan's own prose; NavigationGate kept as a plain counter for SvelteKit-runtime-free testability; SourcePane extended for single-def/multi-def rendering (necessary, not in the plan's declared file list); typing-in-search wired as a REFINE write (necessary, not in Task 3's task-level behavior bullets but required by the plan's must_haves).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] SourcePane.svelte's bare `{:else}` broke the moment BrowseTargetState grew new variants**
- **Found during:** Task 2, running the task's own `pnpm check` acceptance criterion
- **Issue:** Extending `BrowseTargetState` with `single-def`/`multi-def` members made SourcePane's catch-all `{:else}` (which assumed every non-idle/loading/failed state was `'file'`-shaped) a type error — `pnpm check` failed with 7 errors
- **Fix:** Narrowed the branch to explicit `state.kind === 'file'`, added a `single-def` branch (rendering source via `highlightSource`, deriving the highlight language from `Node.language`) and a minimal `multi-def` placeholder
- **Files modified:** `web/src/lib/components/browse/SourcePane.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings
- **Committed in:** `6ea7cbc7` (Task 2 commit)

**2. [Rule 1 - Bug] Svelte 5 `$state()`-rune naming collision in +page.svelte**
- **Found during:** Task 3, running `pnpm check` after wiring `+page.svelte`
- **Issue:** `let state = $state<BrowseTargetState>(...)` — the same known Svelte 5 compiler quirk 03-06 already documented ("used before its declaration"/implicit-any on the variable itself) — now triggered because the file grew enough surrounding code for the checker to flag it
- **Fix:** Renamed to `targetState` throughout the script and template
- **Files modified:** `web/src/routes/browse/+page.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings
- **Committed in:** `1ffd8e6e` (Task 3 commit)

**3. [Rule 2 - Missing Critical] Typing in search never wrote `q` into the URL**
- **Found during:** Task 3's mandated manual history UAT — typing a multi-character query left the address bar completely unchanged
- **Issue:** The plan's own must_haves state "typing in search and adjusting depth or limit each REPLACE the current entry", and D-11 says the same, but no code path wrote the search query back into the URL — `SearchPanel`'s `handleInputChange` only updated the local search-controller store
- **Fix:** Added an optional `onQueryChange(query)` prop to `SearchPanel.svelte`, called from `handleInputChange`; `+page.svelte` wires it to `navigator.navigate(page.url, { q: query || undefined }, NAV_INTENT.REFINE)`
- **Files modified:** `web/src/lib/components/browse/SearchPanel.svelte`, `web/src/routes/browse/+page.svelte`
- **Verification:** Live UAT re-run after the fix: typing "traverse depth" updated the URL's `q` param in place (same symbol/file/line, `q` replaced); pressing back once landed on the previous PUSHED state, not a partial keystroke state — see Coverage D3
- **Committed in:** `354fb816`

---

**Total deviations:** 3 auto-fixed (1 blocking, 1 bug, 1 missing-critical). **Impact on plan:** All three necessary for the plan's own acceptance criteria and must_haves to be honestly satisfied — none was scope creep; each is scoped to the exact gap found, two caught by automated `pnpm check`, one caught by the plan's own mandated manual UAT step.

## TDD Gate Compliance

Task 1 (`type="tdd" tdd="true"`): RED confirmed (`test(03-07)`, `browse-nav.test.ts` — `Failed to resolve import "$lib/browse-nav"`) before the module existed. GREEN confirmed (`feat(03-07)`, 56/56 passing).

Task 2 (`type="tdd" tdd="true"`): RED confirmed (`test(03-07)`, `browse-state.test.ts` — 9/9 new tests fail, `TypeError: loadBlastRadius/createNavigationGate is not a function`). GREEN confirmed (`feat(03-07)`, 65/65 passing).

Task 3 (`type="auto" tdd="true"`): RED confirmed for `neighbors-panel.test.ts` (`Failed to resolve import ".../NeighborsPanel.svelte"`). `browse-history.test.ts` passed IMMEDIATELY (2/2) against Task 1's already-correct `browse-nav.ts` — no natural RED was possible, matching 03-04/03-06's documented substitution precedent (`tdd.md`: "Feature may already exist — investigate"). Substituted an invert-then-restore proof: temporarily flipped `browse-nav.ts`'s intent mapping (`replaceState: intent === NAV_INTENT.NAVIGATE`, the exact inverse of the correct code); re-ran `browse-history`+`browse-nav` — the stack-depth test failed asserting `depth` on the refined state, AND (as a genuine side effect, not engineered) Task 1's own two intent tests failed too, proving `browse-history.test.ts` really does exercise the shared, load-bearing implementation rather than a mock; restored byte-identically (`git diff --stat web/src/lib/browse-nav.ts` empty); re-ran — 67/67 GREEN. Then implemented `NeighborsPanel.svelte` and the route wiring — GREEN, 73/73.

`git log --oneline --grep="^test(03-07)"` and `--grep="^feat(03-07)"` both return non-empty (3 of each) for this plan's commits.

## Shallow-Routing Prohibition — Discrimination Proof

Per rule `84d1gfpywd`, the guard was demonstrated to discriminate before being trusted:

- **Baseline:** conjunct A (`import {...pushState|replaceState...} from '$app/navigation'`, quoted + `-U`) = 0; conjunct B (`\bpushState\b`, wrap-proof) = 0.
- **Planted SINGLE-LINE violation** (`import { pushState, replaceState } from '$app/navigation';`): conjunct A = 1, conjunct B = 1. Removed.
- **Planted WRAPPED violation** (`import {\n  pushState\n} from '$app/navigation';`): conjunct A WITHOUT `-U` = 0 (evaded, demonstrating why `-U` is load-bearing), conjunct A WITH `-U` = 3, conjunct B = 1 (caught either way). Removed.
- **Restored, final state:** conjunct A = 0, conjunct B = 0, positive control (`from '$app/navigation'`) >= 1 (this plan's own `goto` import, type-only in `browse-nav.ts` and real in `+page.svelte`).

The `\$` single-quoting pitfall (cycle-1's own documented trap) was reproduced live during this session's shell scripting: a double-quoted `"from .\$app/navigation."` command substitution silently collapsed to a bare `$` end-of-line anchor and reported the control as 0 even though the import genuinely existed; the corrected single-quoted form immediately reported 1. Recorded here as the fourth site this exact trap has now been observed at.

## Manual History UAT — Verbatim Observations

Against a real `codegraph ui` server on this repository's own index (SPA rebuilt to a scratch directory, temporarily swapped into `web/build/`, `codegraph` binary rebuilt to embed it, restored byte-identically afterward — `git diff --stat web/build` empty, `git status --porcelain web/build` empty).

**1. Five-click chain and the back/forward proof.** Opened `resolveSourcePath` (`internal/query/node.go:33`), then clicked through 4 more neighbours in sequence:

1. `http://127.0.0.1:56762/browse?symbol=resolveSourcePath&file=internal%2Fquery%2Fnode.go&line=33&q=resolveSourcePath`
2. `http://127.0.0.1:56762/browse?symbol=invalidArgumentf&file=internal%2Fquery%2Ferrors.go&line=70&q=resolveSourcePath`
3. `http://127.0.0.1:56762/browse?symbol=Search&file=internal%2Fquery%2Fsearch.go&line=152&q=resolveSourcePath`
4. `http://127.0.0.1:56762/browse?symbol=validateLimit&file=internal%2Fquery%2Fvalidate.go&line=107&q=resolveSourcePath`
5. `http://127.0.0.1:56762/browse?symbol=Callees&file=internal%2Fquery%2Ftraverse.go&line=300&q=resolveSourcePath`

Pressed back twice from state 5: address bar showed state 4's URL, then state 3's URL exactly (`...symbol=Search&file=internal%2Fquery%2Fsearch.go&line=152&q=resolveSourcePath`) — the third state, as required. Pressed forward twice: address bar showed state 4's URL, then state 5's URL exactly — back to the fifth.

**2. Search-refine skips history entirely on back.** From state 5, typed `traverse depth` into the search box (a multi-character query, typed via a single `fill` call representing the debounced end state): URL became `...symbol=Callees&file=internal%2Fquery%2Ftraverse.go&line=300&q=traverse+depth` — the SAME symbol/file/line, only `q` changed (a REPLACE, confirmed by the stack not growing). Pressed back ONCE: landed on `...symbol=validateLimit&file=internal%2Fquery%2Fvalidate.go&line=107&q=resolveSourcePath` — state 4, the previous PUSHED entry — not on any partial one-character-back state (e.g. `q=traverse dept`) and not on state 5's pre-typing URL either (which the refine had already replaced away). This is the direct, live demonstration of D-11's rule: refine replaces the CURRENT top of the stack, so a single back always skips it entirely.

**3. A depth-adjusted URL reconstructs in a fresh tab, including depth.** From state 4 (`validateLimit`), set the blast-radius depth control to `3`: URL became `...symbol=validateLimit&file=internal%2Fquery%2Fvalidate.go&line=107&depth=3&q=resolveSourcePath`. Copied that exact string, opened a brand-new tab, navigated directly to it: the search box showed `resolveSourcePath`, the source pane rendered `validate.go`'s source, the depth spinbutton showed `3`, and the blast-radius list visibly grew from 1 entry (the default-depth view) to 7 entries — confirming depth was not merely echoed in the URL but genuinely re-sent to the server and re-rendered.

## Automated vs. Manual Coverage Split (NAV-02)

- **Automated** (`web/tests/browse-history.test.ts`): the URL<->state contract over a REAL jsdom `history` stack — that `parseBrowseParams` over `location.search` after `back()`/`forward()` yields the exact expected params, and that the push/replace choice produces the correct STACK DEPTH (proven by the second-back discrimination, not a single step).
- **Manual** (this section, live browser + live router): that SvelteKit's actual router responds to a real `popstate` event by re-deriving `page.url` and re-rendering the visible view — this needs the router, which needs the app, which needs a browser; jsdom does not run it.

## User Setup Required

None — no external service configuration required. No new npm packages this plan.

## Next Phase Readiness

- `browse-nav.ts`, the extended `browse-state.ts`, and `NeighborsPanel.svelte` exist in their final architectural shape for this phase — 03-08 (truncation/permalink, the multi-def picker) extends them rather than adding parallel implementations. The multi-def state (`BrowseTargetState.kind === 'multi-def'`) is already produced by `loadBrowseTarget`; only its full picker RENDERING is 03-08's job (SourcePane's current multi-def branch is a minimal placeholder — recorded in Known Stubs).
- The intended-RED `task web:drift` leg (opened 03-01) is still open and unaffected in the way that matters: SOURCE-half mismatch grew further (68 source files now vs 22 at 03-01), OUTPUT half MATCH (`web/build/` verified restored byte-identically after the manual-UAT temporary swap — `git diff --stat web/build` empty, `git status --porcelain web/build` empty). Closes at 03-09 Task 3.
- No blockers for 03-08 or any other wave-5+ plan.

## Known Stubs

- `web/src/lib/components/browse/SourcePane.svelte`'s `multi-def` branch renders a fixed placeholder line ("N definitions found for ... — a picker is not yet built") rather than BRW-05's actual disambiguation picker. Deliberate: the state itself (`BrowseTargetState.kind === 'multi-def'`, carrying `definitions`/`totalCandidates`) is fully produced by this plan's Task 2; rendering it is explicitly plan 03-08's job per this plan's own Task 2 action text ("rendering of that state lands in plan 03-08"). Recorded in `.planning/WINDOWS.md` (entry #23, kind `stub`) so it stays visible at ship time.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

- `test -f web/src/lib/browse-nav.ts` → FOUND
- `test -f web/src/lib/components/browse/NeighborsPanel.svelte` → FOUND
- `test -f web/tests/browse-nav.test.ts` → FOUND
- `test -f web/tests/browse-state.test.ts` → FOUND
- `test -f web/tests/browse-history.test.ts` → FOUND
- `test -f web/tests/neighbors-panel.test.ts` → FOUND
- `git log --oneline --all | grep -q 836aadf7` → FOUND
- `git log --oneline --all | grep -q 5aaf5f29` → FOUND
- `git log --oneline --all | grep -q 36af1a18` → FOUND
- `git log --oneline --all | grep -q 6ea7cbc7` → FOUND
- `git log --oneline --all | grep -q edc6e0f1` → FOUND
- `git log --oneline --all | grep -q 1ffd8e6e` → FOUND
- `git log --oneline --all | grep -q 354fb816` → FOUND
- `cd web && pnpm test` → exit 0, 73/73 tests, names browse-nav/browse-state/browse-history/neighbors-panel
- `cd web && pnpm check` → exit 0, 0 errors, 0 warnings
- `task web:test` → observed numTotalTests=73 numPassedTests=73, PASS
- `rg -o 'PLACEHOLDER-03-07' web/src/ | wc -l` → 0 (pre-edit observed 1)
- `task web:drift` → intended-RED (SOURCE-half mismatch, 68 vs marker 22); OUTPUT-half MATCH — web/build restored byte-identically after manual UAT
