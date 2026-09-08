---
phase: 03-browse-inspect-navigation
plan: 08
subsystem: ui
tags: [svelte, browse, click-to-definition, disambiguation, clipboard, permalink, tdd]

requires:
  - phase: 03-browse-inspect-navigation (plan 03-04)
    provides: "highlight.ts's highlightSource — the ONE markup-producing helper this plan's decorator must never duplicate"
  - phase: 03-browse-inspect-navigation (plan 03-05)
    provides: "GetPermalink's frozen wire shape (path/line/end_line -> url/availability/reason) and the three-valued PermalinkAvailability enum this plan's SourcePane consumes"
  - phase: 03-browse-inspect-navigation (plan 03-07)
    provides: "browse-nav.ts's NAV_INTENT/BrowseNavDelta, browse-state.ts's single-def/multi-def BrowseTargetState variants, and the SourcePane multi-def PLACEHOLDER this plan closes"
provides:
  - "web/src/lib/call-targets.ts — buildCallTargetIndex/lookupCallTarget (exact-match only), IDENTIFIER_PATTERN (Unicode-aware tokenizer), decorateCallTargets (idempotent, teardown-returning DOM decorator built entirely from DOM APIs), and callTargets (the Svelte action wrapping it)"
  - "web/src/lib/components/browse/CopyAction.svelte — one-action clipboard copy, absent (not disabled) for an empty value"
  - "web/src/lib/components/browse/DefinitionPicker.svelte — BRW-05's disambiguation surface, mounted from +page.svelte for the multi-def target state"
  - "SourcePane.svelte extended: click-to-definition wiring on single-def source, the truncation notice's paired permalink surface (three distinct availability states), and copy affordances for path/symbol name"
affects: [03-09]

actuals:
  tokens: 16275
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Idempotent DOM decoration via a marker attribute + ancestor-walk TreeWalker filter, independent of any caller lifecycle — a second decoration pass over the same subtree is a structural no-op, not merely unlikely in practice"
    - "Teardown-by-exact-restoration: a decorator that replaces a text node records the ORIGINAL node plus its replacement set, and teardown re-inserts that exact original rather than calling Node.normalize() (which the plan's own acceptance grep forbids as a disguised 'normalize(' hit)"
    - "PermalinkClient/NodeDetailClient/ImpactClient convention extended a third time: any component issuing its own RPC takes the client as a prop, declared independently of the real generated client, so a test stub satisfies it with no Connect transport"

key-files:
  created:
    - web/src/lib/call-targets.ts
    - web/src/lib/components/browse/CopyAction.svelte
    - web/src/lib/components/browse/DefinitionPicker.svelte
    - web/tests/call-targets.test.ts
    - web/tests/source-pane.test.ts
    - web/tests/definition-picker.test.ts
    - .planning/todos/pending/2026-08-28-client-side-render-cost-measurement-for-browse-views.md
  modified:
    - web/src/lib/components/browse/SourcePane.svelte
    - web/src/routes/browse/+page.svelte
    - .planning/WINDOWS.md

key-decisions:
  - "Truncated-file permalink request carries NO anchor at all (neither line nor end_line) — the open question 03-RESEARCH.md left for this plan. Rejected linking to the single start line as a middle ground: the point of the link is 'the rest of the file is what the local view could not show' (D-20), and any anchor still frames the remote view around a specific point in the truncated portion rather than the whole file. A bare file link is the strict reading."
  - "The 'state' prop's local binding is destructured as 'target' throughout SourcePane.svelte (`let { state: target, ... } = \$props()`), keeping the external prop name 'state' for callers. A bare local named 'state' collides with any `\$state(...)` rune elsewhere in the same component (Svelte reads `\$identifier` as store-auto-subscription syntax) — the exact pitfall 03-06/03-07 already documented for other files, now hit a third time and following the same fix."
  - "SourcePane's `client` prop for GetPermalink is OPTIONAL, defaulting to no permalink surface when absent, rather than required — this let 03-04's pre-existing browse-tracer.test.ts fixtures (which render SourcePane with no client) stay untouched rather than widening this plan's file scope to fix an unrelated test file for a prop it never needed."
  - "decorateCallTargets's click-to-definition callback receives the FULL CallTargetIndex entry (the list of Node candidates sharing the clicked identifier's name), and SourcePane's wiring navigates by symbol NAME ALONE (never symbol+file+line) — a click is text-driven and D-18 states resolution is inherently ambiguous; the re-issued GetNodeDetail by symbol name is what may land in this same plan's own disambiguation picker."

patterns-established:
  - "Invert-then-restore TDD substitution was not needed this plan — every RED was a genuine 'Failed to resolve import' before the module existed, confirmed live via move-file/run/restore for all three tasks."

requirements-completed: [BRW-04, BRW-05, BRW-07, BRW-09]

coverage:
  - id: D1
    description: "An identifier in rendered source matching a name in the opened node's call list is clickable and jumps to that definition; matching is exact in both directions; a name split across highlight.js sibling elements is deliberately not clickable; decoration is idempotent and its teardown restores original DOM and releases listeners (BRW-04)"
    requirement: BRW-04
    verification:
      - kind: unit
        ref: "web/tests/call-targets.test.ts (13 tests: index ambiguity, exact-match with paired negative/positive for case/normalization/prefix-superstring/empty-index, tokenizer keyword/non-ASCII cases, decoration+callback, idempotence, teardown, split-identifier bound)"
        status: pass
      - kind: manual_procedural
        ref: "live codegraph ui against this repository's own index via agent-browser: DefinitionPicker UAT screenshots (uat_definition_picker.png, uat_singledef_after_pick.png) show the single-def source pane rendering after a pick, syntax-highlighted with copy affordances and the permalink surface visible"
        status: pass
    human_judgment: false
  - id: D2
    description: "A bare symbol name resolving to several definitions offers a picker listing every RETURNED candidate plus the server's true total, marking ungathered candidates from the wire's own flag (never emptiness), pre-selecting nothing, in server order, and selecting one navigates with symbol+file+line so the result is NodeDetailModeSingleDef and directly shareable (BRW-05)"
    requirement: BRW-05
    verification:
      - kind: unit
        ref: "web/tests/definition-picker.test.ts (10 tests: candidate listing + true total, omitted-count paired equal/unequal, ungathered-marker paired true/false-flag with empty calls, nothing pre-selected, order-preservation, symbol+file+line delta with the symbol-not-dropped assertion, the delta round-tripped through loadBrowseTarget yielding NodeDetailModeSingleDef, ArrowUp/ArrowDown keyboard traversal)"
        status: pass
      - kind: manual_procedural
        ref: "live UAT: symbol=New in this repo's own index resolved to 2 candidates (internal/query/engine.go:66, internal/daemon/daemon.go:172); selecting the second navigated the address bar to symbol=New&file=internal%2Fdaemon%2Fdaemon.go&line=172 and rendered that definition's own source, callers, callees and blast radius"
        status: pass
    human_judgment: false
  - id: D3
    description: "A file path or symbol name is copyable in one action, writing the exact displayed string untrimmed/unnormalized; a copy affordance is absent (not disabled) for an empty value (BRW-07)"
    requirement: BRW-07
    verification:
      - kind: unit
        ref: "web/tests/source-pane.test.ts (CopyAction describes: empty-vs-non-empty presence pairing, exact-string copy including whitespace, two-copies-in-succession leaves the SECOND value, and a SourcePane integration test asserting both distinctly-labelled controls appear for a single-def target)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The current file/line opens on GitHub pinned to the indexed commit, with the permalink's own certainty stated honestly across all three availability states; a truncated file states the truncation and offers the permalink as the route to the rest (BRW-09, D-20)"
    requirement: BRW-09
    verification:
      - kind: unit
        ref: "web/tests/source-pane.test.ts (truncation-notice absence/presence pairing with both line counts; three pairwise-DISTINCT availability renders with the unverified reason visible; truncated-vs-opened-node request-shape pairing asserting endLine absent vs present)"
        status: pass
      - kind: manual_procedural
        ref: "live UAT against this repository's own index: before a re-index, GetPermalink returned NO_LINK with reason 'this index has no recorded commit SHA (a pre-upgrade graph) — re-index to enable permalinks'; after `codegraph index --force`, the SAME call returned a real url (https://github.com/seanb4t/codegraph-go/blob/<sha>/internal/query/node.go#L33) with availability LINKABLE_UNVERIFIED (the commit was not yet observed on a remote-tracking branch — an honest, expected result for unpushed local work, not a bug)"
        status: pass
    human_judgment: false

duration: ~13min (commit-to-commit span; excludes required-reading time before the first commit)
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 8: Source Pane Completion — Click-to-Definition, Disambiguation, Copy, Permalinks Summary

**A pure text-driven identifier matcher with a from-scratch DOM decorator lights up click-to-definition inside verbatim highlighted source; a keyboard-drivable disambiguation picker replaces the old placeholder for ambiguous names; a file path or symbol name copies in one action; and GetPermalink's three availability states render honestly, with the truncated-file permalink deliberately carrying no line anchor at all.**

## Performance

- **Duration:** ~13 min (commit span)
- **Started:** 2026-08-28T23:20:56-04:00 (Task 1 RED commit)
- **Completed:** 2026-08-29T03:33:09Z
- **Tasks:** 3
- **Files modified:** 10 (7 created, 3 modified)

## Accomplishments

- `web/src/lib/call-targets.ts` (D-18, BRW-04): exact-match-only identifier index over a node's `calls` list; a Unicode-aware `IDENTIFIER_PATTERN` (via `\p{ID_Start}`/`\p{ID_Continue}`) so a non-ASCII identifier is one token, never split; `decorateCallTargets` walks a `TreeWalker` over text nodes and builds replacement elements exclusively through `createElement`/`createTextNode`/`replaceWith` — never an HTML string — idempotent via a marker attribute the walker's own `acceptNode` filter respects, with a teardown that restores the EXACT original text node (not a `.normalize()`-approximated one) and releases every listener; the deliberate split-identifier bound (a name highlight.js splits across sibling elements is simply not clickable) is documented and pinned by test. Wrapped in `callTargets`, a Svelte action with `update`/`destroy`.
- `web/src/lib/components/browse/CopyAction.svelte` (BRW-07): writes the exact displayed string via the Clipboard API, no trim/case-fold; renders nothing for an empty value rather than a disabled control.
- `web/src/lib/components/browse/DefinitionPicker.svelte` (BRW-05, D-21): lists every candidate `GetNodeDetail` returned and states the true `totalCandidates` alongside, naming the difference when they diverge; marks a candidate ungathered from the wire's own `detailGathered` flag, never from an empty call list; renders in server order (no `.sort()`/`localeCompare` anywhere in the file); pre-selects nothing; selecting a candidate navigates with the full symbol+file+line triple, which is what keeps the server in `NodeDetailModeSingleDef` narrowing rather than falling into `buildNodeDetail`'s `symbol == ""` FILE-mode branch. Arrow-key traversal between candidates; Enter is native via real `<button>` elements.
- `SourcePane.svelte` extended: the single-def source's rendered element carries `use:callTargets`, wired to a `handleCallTargetSelect` that re-navigates by symbol name alone (text-driven, may land in the picker); the truncation notice's absence/presence is unchanged, and a `permalinkSurface` snippet renders GetPermalink's three availability states distinctly, requesting a range permalink for a non-truncated opened node and a bare file link (no anchor) for a truncated one; copy affordances for path (file mode) and path+name (single-def mode); the old multi-def placeholder branch is gone entirely.
- `+page.svelte`: mounts `DefinitionPicker` for the multi-def target state instead of `SourcePane`, and passes `uiClient`/the navigate handler through to `SourcePane`.
- Filed `.planning/todos/pending/2026-08-28-client-side-render-cost-measurement-for-browse-views.md`, the visible disposition of cycle-1 review's deferred client-side render-cost measurement (recorded per the plan's own "Deferred with reason" section).

## Task Commits

Each task followed RED-GREEN TDD discipline:

1. **Task 1: The click-to-definition matcher and its DOM decorator**
   - `ff1c50bd` — `test(03-08): add failing test for click-to-definition matcher and DOM decorator` (RED: `Failed to resolve import "$lib/call-targets"`)
   - `28673d66` — `feat(03-08): implement click-to-definition matcher and DOM decorator (BRW-04)` (GREEN: 13/13; full suite 86/86)
2. **Task 2: Copy affordance, truncation notice, and the permalink surface**
   - `e9b42c31` — `test(03-08): add failing test for truncation notice, permalink surface, and copy affordance` (RED: `Failed to resolve import "$lib/components/browse/CopyAction.svelte"`)
   - `cfa682dc` — `feat(03-08): implement copy affordance, truncation notice, and permalink surface (BRW-07, BRW-09, D-20)` (GREEN: 7/7; full suite 93/93)
3. **Task 3: The disambiguation picker**
   - `ede4c01d` — `test(03-08): add failing test for the disambiguation picker` (RED: `Failed to resolve import "$lib/components/browse/DefinitionPicker.svelte"`)
   - `d24ffa52` — `feat(03-08): implement the disambiguation picker (BRW-05, D-21)` (GREEN: 10/10; full suite 103/103)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/src/lib/call-targets.ts` — the identifier matcher and DOM decorator
- `web/src/lib/components/browse/CopyAction.svelte` — the copy affordance
- `web/src/lib/components/browse/DefinitionPicker.svelte` — the disambiguation picker
- `web/src/lib/components/browse/SourcePane.svelte` — extended: click-to-definition wiring, truncation+permalink surface, copy affordances; multi-def placeholder removed
- `web/src/routes/browse/+page.svelte` — mounts DefinitionPicker for multi-def; passes client/onNavigate through to SourcePane
- `web/tests/call-targets.test.ts`, `web/tests/source-pane.test.ts`, `web/tests/definition-picker.test.ts` — new test coverage
- `.planning/todos/pending/2026-08-28-client-side-render-cost-measurement-for-browse-views.md` — the deferred-measurement disposition
- `.planning/WINDOWS.md` — entry #23 marked fixed via `gsd-tools query windows fixed 23`

## Decisions Made

See `key-decisions` in frontmatter. Summary: the truncated-file permalink carries no anchor at all (the strict reading of "the rest of the file is what the local view could not show"); SourcePane's local prop binding renamed from `state` to `target` to avoid the `$state` rune naming collision (03-06/03-07's documented pitfall, hit a third time); `client` on SourcePane is optional so 03-04's pre-existing tracer test fixtures needed no change; the click-to-definition callback carries the full matched `Node[]` entry, and SourcePane's wiring re-navigates by symbol name alone, deliberately never symbol+file+line, since a click is text-driven per D-18.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] SourcePane's local `state` binding collided with the `$state` rune**
- **Found during:** Task 2, adding the permalink `$state<PermalinkState>(...)` variable
- **Issue:** `pnpm check` reported `store_rune_conflict` and a `non_reactive_update` warning, escalating to hard TypeScript errors, because a plain destructured prop named `state` in the same component as a `$state(...)` call makes Svelte's compiler read `$state` as auto-subscribing to a store literally named `state`.
- **Fix:** Destructured the prop as `let { state: target, ... } = $props()`, keeping the external prop name `state` (callers still write `state={targetState}`), and renamed every internal `state.`/`state)` reference to `target` via a scoped regex substitution that deliberately excluded `$state<...>`, `permalinkState`, `PermalinkState` and `BrowseTargetState`.
- **Files modified:** `web/src/lib/components/browse/SourcePane.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings
- **Committed in:** `cfa682dc` (Task 2 commit)

**2. [Rule 3 - Blocking] `client` made optional on SourcePane rather than required**
- **Found during:** Task 2, running `pnpm check` after adding the `client` prop
- **Issue:** Making `client` a required prop broke type-checking on `web/tests/browse-tracer.test.ts` (03-04's pre-existing fixture), which renders `SourcePane` with only a `state` prop and is outside this plan's `files_modified`.
- **Fix:** `client?: PermalinkClient` (optional); the permalink-loading effect treats an absent client identically to absent request params — idle, no permalink surface, no error.
- **Files modified:** `web/src/lib/components/browse/SourcePane.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings; `browse-tracer.test.ts` unchanged and still passing
- **Committed in:** `cfa682dc` (Task 2 commit)

**3. [Rule 3 - Blocking] `<ul onkeydown>` triggered an a11y lint error**
- **Found during:** Task 3, running `pnpm check` after wiring keyboard traversal
- **Issue:** `a11y_no_noninteractive_element_interactions` — a keydown listener on a non-interactive `<ul>` is flagged by svelte-check's accessibility linting.
- **Fix:** Moved the listener onto each candidate `<button>` (inherently interactive), computing the next-focus target from the button's siblings rather than from a container-level handler.
- **Files modified:** `web/src/lib/components/browse/DefinitionPicker.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings
- **Committed in:** `d24ffa52` (Task 3 commit)

**4. [Rule 1 - Bug] Duplicate `{#each}` key in a test fixture, not the component**
- **Found during:** Task 3, first run of `definition-picker.test.ts`
- **Issue:** `each_key_duplicate` — two test fixtures called `definition('Foo', {})` twice with identical `filePath`/`startLine`/`name`, colliding under the component's original `filePath:startLine:name` key.
- **Fix:** `candidateKey` now includes the loop index as a tiebreaker (`...:${index}`), which is correct for a static, never-reordered list regardless of whether two real candidates could ever coincidentally share file+line+name.
- **Files modified:** `web/src/lib/components/browse/DefinitionPicker.svelte`
- **Verification:** `10/10` definition-picker tests pass
- **Committed in:** `d24ffa52` (Task 3 commit)

**5. [Plan-text correction] The `{@html}` count acceptance criterion assumed 1; the actual pre-existing baseline was already 2**
- **Found during:** Task 1, running the plan's own `rg -o '\{@html' web/src/ ... | wc -l` acceptance check
- **Issue:** The plan's acceptance criteria state this count "still returns 1", but 03-07 already established TWO legitimate `{@html}` sites in `SourcePane.svelte` (the `file`-mode branch, present since 03-04, and the `single-def`-mode branch 03-07 added) — the literal expected number in the plan text was stale relative to work this plan itself was required to read (03-07-SUMMARY.md).
- **Resolution:** Verified the invariant that actually matters — no NEW `{@html}` site was introduced by this plan's changes — held: the count was 2 before any of this plan's edits and remained 2 (both still in `SourcePane.svelte`) after. One incidental true positive was fixed along the way: this module's own doc comment originally used the literal text `` `{@html}` `` and was rephrased to "Svelte's html directive" so the acceptance grep's zero/one counts are not inflated by a code comment.
- **Files modified:** `web/src/lib/call-targets.ts` (comment wording only)
- **Impact:** No behavioral change; the plan's literal "1" was corrected to "2 (unchanged)" as the actually-meaningful assertion.

---

**Total deviations:** 4 auto-fixed (all Rule 3/1, blocking issues or a bug preventing `pnpm check`/tests from passing) + 1 plan-text correction (a stale literal in an acceptance criterion, resolved by verifying the underlying invariant rather than the literal number). **Impact:** All necessary for the plan's own acceptance criteria to be honestly satisfiable; none expands scope beyond what each task already specified.

## TDD Gate Compliance

All three tasks: RED confirmed via genuine "Failed to resolve import" errors (module/component did not yet exist) before each GREEN commit — verified live via move-file/run-tests/restore-file for every task, not merely written-then-assumed. `git log --oneline --grep="^test(03-08)"` and `--grep="^feat(03-08)"` each return 3 commits for this plan.

## Manual UAT — Verbatim Observations

Against a real `codegraph ui` server on this repository's own index (SPA rebuilt to a scratch directory via `CODEGRAPH_WEB_BUILD_DIR`, temporarily swapped into `web/build/`, `codegraph` binary rebuilt to embed it, restored byte-identically afterward — `git diff --stat web/build` empty, `git status --porcelain web/build` empty, twice, once per swap).

**1. Permalink availability, pre- and post-re-index (Task 2's required UAT).** `GetPermalink({path: "internal/query/node.go", line: 33})` before re-indexing:
```json
{"availability":"PERMALINK_AVAILABILITY_NO_LINK","reason":"this index has no recorded commit SHA (a pre-upgrade graph) — re-index to enable permalinks"}
```
After `codegraph index --force` (this repo's `.codegraph/` genuinely predated commit-SHA recording, confirmed via `GetStatus` returning no `commitSha` field before, and a populated one — `cfa682dc7347d8d07b837d87a4fea359658b3160`, this plan's own Task 3 commit at the time — after):
```json
{"url":"https://github.com/seanb4t/codegraph-go/blob/cfa682dc7347d8d07b837d87a4fea359658b3160/internal/query/node.go#L33","availability":"PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED","reason":"this commit is not observed on any remote-tracking branch; it may be unpushed, and the link may 404 until it is pushed"}
```
A real URL now renders (LINKABLE_UNVERIFIED, not LINKABLE, because that exact commit was genuinely unpushed local work at observation time — an honest exercise of D-07's own uncertainty case, not a bug).

**2. The disambiguation picker (Task 3's required UAT).** `symbol=New` in this repository's own index resolves to 2 candidates: `internal/query/engine.go:66` and `internal/daemon/daemon.go:172`. Opening `/browse?symbol=New` rendered `DefinitionPicker` with both listed (screenshot: `uat_definition_picker.png`). Clicking the second candidate navigated the address bar to `?symbol=New&file=internal%2Fdaemon%2Fdaemon.go&line=172` and rendered that exact definition's own syntax-highlighted source, callers, callees and blast radius panels (screenshot: `uat_singledef_after_pick.png`) — proving the server resolved to `NodeDetailModeSingleDef` for the picked candidate specifically, not a re-render of the picker or a fall-through to FILE mode.

**Incidental observation, not a defect:** between the two UAT passes, the recorded `commitSha` reverted to empty again (a subsequent `GetStatus` call showed no `commitSha`, with `nodeCount`/`edgeCount` also changed) — some background re-sync appears to run against this repo's live `.codegraph/` store during active development and does not itself populate `commitSha` the way a full `codegraph index --force` does. Out of scope for this plan (BRW-09's contract — degrade to NO_LINK with a named reason — is exactly what this produced); noted here in case it is useful context for a future plan touching indexing/sync behavior.

## User Setup Required

None — no external service configuration required. No new npm packages this plan.

## Known Stubs

None. WINDOWS.md entry #23 (the multi-def SourcePane placeholder this plan was scoped to close) is resolved.

## Next Phase Readiness

- The source pane is now feature-complete for this phase's scope: click-to-definition, disambiguation, copy, and permalinks all render and are wired end to end.
- The intended-RED `task web:drift` leg (opened 03-01) is still open and unaffected in the way that matters: SOURCE-half mismatch grew further (71 source files now vs marker 22), OUTPUT-half MATCH (`web/build/` verified restored byte-identically after each of the two manual-UAT temporary swaps this plan performed). Closes at 03-09 Task 3, per that plan's own scope.
- No blockers for 03-09.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

- `test -f web/src/lib/call-targets.ts` → FOUND
- `test -f web/src/lib/components/browse/CopyAction.svelte` → FOUND
- `test -f web/src/lib/components/browse/DefinitionPicker.svelte` → FOUND
- `test -f web/tests/call-targets.test.ts` → FOUND
- `test -f web/tests/source-pane.test.ts` → FOUND
- `test -f web/tests/definition-picker.test.ts` → FOUND
- `test -f .planning/todos/pending/2026-08-28-client-side-render-cost-measurement-for-browse-views.md` → FOUND
- `git log --oneline --all | grep -q ff1c50bd` → FOUND
- `git log --oneline --all | grep -q 28673d66` → FOUND
- `git log --oneline --all | grep -q e9b42c31` → FOUND
- `git log --oneline --all | grep -q cfa682dc` → FOUND
- `git log --oneline --all | grep -q ede4c01d` → FOUND
- `git log --oneline --all | grep -q d24ffa52` → FOUND
- `cd web && pnpm test` → exit 0, 103/103 tests, names call-targets/source-pane/definition-picker
- `cd web && pnpm check` → exit 0, 0 errors, 0 warnings
- `task web:test` → observed numTotalTests=103 numPassedTests=103, PASS (up from 73 at 03-07's close)
- `rg -o '\{@html' web/src/ --glob '!**/components/ui/**' | wc -l` → 2, both in SourcePane.svelte (unchanged from pre-plan baseline; see Deviation 5)
- `rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' web/src/ --glob '!**/components/ui/**' | wc -l` → 0
- `rg -o '\.sort\(|localeCompare' web/src/lib/components/browse/DefinitionPicker.svelte | wc -l` → 0
- `task web:drift` → intended-RED (SOURCE-half mismatch, 71 vs marker 22); OUTPUT-half MATCH — web/build restored byte-identically after both manual-UAT swaps
