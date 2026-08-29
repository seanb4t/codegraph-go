---
phase: 03-browse-inspect-navigation
plan: 06
subsystem: ui
tags: [svelte, shadcn-svelte, bits-ui, connectrpc, search, keyboard, tdd]

requires:
  - phase: 03-browse-inspect-navigation (plan 03-04)
    provides: "browse-state.ts / rpc-errors.ts's client-stub-interface pattern and classifyRpcError, reused by search.ts; the +page.svelte tracer this plan extends"
  - phase: 03-browse-inspect-navigation (plan 03-05)
    provides: "serialization ordering on web/src/lib/gen/ui_pb.ts (no code dependency — the frontmatter's SERIALIZATION EDGE, see 03-06-PLAN.md header note)"
provides:
  - "web/src/lib/components/ui/command/ (+dialog/button/input-group/input/textarea) — the phase's first vendored shadcn-svelte surface, human-reviewed at add time per D-22"
  - "web/src/lib/search.ts — createSearchController(client): debounce+cancellation+request-identity-guarded live search (Search+Files), independent Explore-on-submit, never merges/ranks results"
  - "web/src/lib/components/browse/SearchPanel.svelte — Symbols/Files/Explore sections, /, Cmd+K/Ctrl+K, Escape, and the vendored primitive's own cross-group arrow-key traversal"
affects: [03-07, 03-08, 03-09]

actuals:
  tokens: 23532
  tasks: 3
  commits: 5

tech-stack:
  added:
    - "bits-ui@2.19.0 (direct devDependency, via shadcn-svelte's command add)"
    - "@internationalized/date@3.12.3 (direct devDependency, bits-ui peer)"
    - "10 transitive packages: @floating-ui/core, @floating-ui/dom, @floating-ui/utils, @swc/helpers, inline-style-parser, runed, style-to-object, svelte-toolbelt, tabbable, tslib"
  patterns:
    - "svelte/store's writable (built into the already-installed svelte package, no new dependency) as the shape for non-rune, framework-agnostic controller state — testable via get()/subscribe() in plain vitest, consumed reactively in a component via a single $effect bridging subscribe() into a $state rune."
    - "Never name a local $state()-backed variable literally `state` in a Svelte 5 <script> block — svelte-check reports 'used before its declaration' / implicit any on the variable itself, not on any real bug. Renamed to searchState; the failure mode and fix are recorded verbatim in SearchPanel.svelte's own git history for the next person who hits it."
    - "An RPC dispatch that must be inspectable/awaitable in tests should route the AbortSignal through the SAME options-object shape client.ts's other calls already use ({ signal }) — kept identical between browse-state.ts's NodeDetailClient (03-04) and search.ts's SearchClient so a caller doesn't need two different calling conventions."
    - "When a vendored keyboard primitive already owns Enter-selects-the-highlighted-item, don't hijack Enter globally for a second action (submit) — that races the two behaviors unpredictably depending on what happens to be selected. Route the second action through the SAME item-select mechanism (a single always-first, unlabeled Command.Item whose onSelect calls the second action) instead of adding a parallel keydown handler."

key-files:
  created:
    - web/src/lib/components/ui/command/ (14 files)
    - web/src/lib/components/ui/dialog/ (11 files)
    - web/src/lib/components/ui/button/ (2 files)
    - web/src/lib/components/ui/input-group/ (7 files)
    - web/src/lib/components/ui/input/ (2 files)
    - web/src/lib/components/ui/textarea/ (2 files)
    - web/src/lib/search.ts
    - web/src/lib/components/browse/SearchPanel.svelte
    - web/tests/search.test.ts
    - web/tests/search-panel.test.ts
    - .planning/todos/pending/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md
    - .planning/todos/pending/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/src/routes/browse/+page.svelte

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human): vendor APPROVED as a set — bits-ui@2.19.0 and @internationalized/date@3.12.3 directly, 10 transitive packages as a set, and the 6-directory/36-file vendored surface (command's own registry-composed dependencies: dialog, button, input-group, input, textarea — none independently requested). textarea/ and input-group-textarea.svelte KEPT despite being currently unimported by command's own files, because they are part of input-group's atomic registry unit and stripping files out of it would defeat the very source-match property the registry-version-pinning todo exists to check."
  - "Task 1: shadcn-svelte registry v1.5.1. Two vendored-directory unescaped-HTML counts both zero (rg -o '{@html' and rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' over web/src/lib/components/ui/), positive-controlled by the orchestrator (rg -o 'script' over the same tree returns 79 hits across 31/36 files — the two zero counts are real absences, not an empty scan, per rule 84d1gfpywd)."
  - "Task 1: keyboard-capability spike finding — the vendored Command primitive traverses Command.Group boundaries in visual order (bits-ui's updateSelectedByItem/getValidItems walk ONE flat array of visible items regardless of group membership; group membership only affects sort-scoring and scroll-into-view). Confirmed twice: once via a throwaway jsdom spike (deleted before the Task 1 commit per human instruction), once live against a real browser and this repository's real index (Task 3's manual UAT). Task 3 therefore used real Command.Group sections, never a flat-list workaround."
  - "Task 2: Files' live-search pattern is `*` + term + `*` — the best available single glob within FilesOptions.Pattern's actual (whole-path, path/filepath.Match) semantics. Discovered live during Task 3's manual UAT (not assumed): Go's `*` never crosses `/` and has no recursive `**`, so this pattern only matches ROOT-LEVEL files — confirmed empirically (filepath.Match('*detail*','internal/query/detail.go')=false; filepath.Match('*claude*','claudeassets.go')=true). Documented in search.ts and filed as a todo; Search's own file-kind pseudo-node matches already cover arbitrary-depth file discovery via the Symbols section (confirmed live), so BRW-01 is functionally met today, just not by the Files RPC for nested paths."
  - "Task 3: Explore submission is triggered by a single, unlabeled, always-first Command.Item ('Ask \"<query>\"') whose onSelect calls submit() — not a hijacked global Enter handler. The vendored primitive's own onkeydown already binds Enter to item.click() on whatever is currently selected; a parallel Enter handler would race against the ALSO-required 'Enter on a highlighted item invokes the open callback' behavior. Reusing the tested item-select mechanism for both actions avoids that race entirely, and the primitive's own default-selects-first-item behavior means a bare \"type, then press Enter\" reaches it naturally."
  - "Task 3: SearchSelection is a 3-way union (symbol / file / explore-file) rather than fabricating a fake FileEntry for ExploreGroup selections — ExploreGroup carries only a path on the wire, no language/nodeCount/edgeCount."

patterns-established:
  - "Manual UAT against a live server with REAL indexed data (not flat unit-test fixtures) is what caught the Files-pattern depth limitation — no unit test would have, since every fixture in search.test.ts/search-panel.test.ts used single-segment file paths. Recorded as a reminder that this project's mandated manual-UAT step is load-bearing, not ceremonial."

requirements-completed: [BRW-01, BRW-08, NAV-03]

coverage:
  - id: D1
    description: "Typing in the search box issues Search and Files live, debounced (150ms) and cancellable (AbortController + monotonic request-identity guard), and renders their results as two labelled sections (Symbols, then Files) in the order each RPC returned them — never merged, sorted, or ranked client-side"
    requirement: BRW-01
    verification:
      - kind: unit
        ref: "web/tests/search.test.ts (10 cases: min-chars gating, debounce coalescing, out-of-order cancellation by value, order preservation, Files format discipline)"
        status: pass
      - kind: unit
        ref: "web/tests/search-panel.test.ts#'shows Symbols then Files, each in the order supplied, given both result kinds'"
        status: pass
      - kind: manual_procedural
        ref: "live codegraph ui against this repo's own index via agent-browser: typing 'Engine' showed 26 live Symbols results as-you-type (screenshot uat-1); typing 'claude' showed both Symbols (27 matches) and Files (1 match, claudeassets.go, confirmed present in DOM though scrolled below the fold) sections together (screenshot uat-3, DOM-text-content check)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Pressing Enter issues Explore and renders its relevance-selected results as a third, separately labelled section alongside the live results — never replacing them; an empty outcome and a failed outcome render distinguishable text"
    requirement: BRW-08
    verification:
      - kind: unit
        ref: "web/tests/search.test.ts#'submit() issues exactly one Explore call', #'an Explore response with empty=true...'; web/tests/search-panel.test.ts#'shows a third section with explore results, without disturbing the live sections', #'renders a no-results line...distinct from the failure line'"
        status: pass
      - kind: manual_procedural
        ref: "live: typed 'how does search work', Home+Enter -> real Explore section rendered with 4 real ranked file matches from this repo's own index (screenshot uat-4); typed a nonsense query, Home+Enter -> 'No results for...' rendered (screenshot uat-5)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The whole surface is drivable from the keyboard: / (except while an editable element has focus, where it inserts normally), Cmd+K/Ctrl+K (preventDefault, focuses from anywhere), Escape (dismisses), ArrowDown crossing the Symbols->Files boundary, Enter opening the highlighted item"
    requirement: NAV-03
    verification:
      - kind: unit
        ref: "web/tests/search-panel.test.ts (6 cases covering every one of the five shortcuts plus the /-insertion-when-editable case, using fireEvent's return value to assert preventDefault was/was-not called)"
        status: pass
      - kind: manual_procedural
        ref: "live, via agent-browser: / from document.body focused the input; / while already focused in the input inserted the character ('path' -> 'path/'); Ctrl+K re-focused and reopened from a blurred state; Escape hid the 'No results for...' text; ArrowDown sequence on a 1-symbol+1-file query observed ask -> symbol:claudeassets.go... -> file:claudeassets.go -> (held), confirming the cross-section transition live"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every vendored shadcn-svelte component file was read and reviewed at add time, as committed code, not an opaque dependency (plan's own backstop truth)"
    verification:
      - kind: manual_procedural
        ref: "Task 1's blocking-human checkpoint: all 36 files read in full; rg scans for fetch/XMLHttpRequest/WebSocket/require/process./dynamic import/readFile/writeFile/eval/new Function all zero hits; human-approved 2026-08-28"
        status: pass
    human_judgment: true
    rationale: "The checkpoint decision itself (approve/reject each package and file) is inherently a human judgment call by design (gate=blocking-human) — recorded here for the audit trail even though its outcome is already reflected in the commit history and this SUMMARY's key-decisions."

duration: ~50min active work (excludes wait time for the Task 1 human checkpoint decision)
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 6: Search Surface (Command + Controller + Panel) Summary

**Search-as-you-type over `Search`/`Files` (150ms debounce, AbortController + request-identity cancellation) with `Explore`-on-Enter rendering a third section, built on a human-reviewed vendored shadcn-svelte `Command` primitive, fully keyboard-drivable (`/`, `Cmd+K`/`Ctrl+K`, `Escape`, cross-section arrow-key traversal) — with a real depth-limitation in the `Files` RPC's glob pattern discovered live and documented, not silently shipped.**

## Performance

- **Duration:** ~50 min active work (excludes the Task 1 human checkpoint wait)
- **Tasks:** 3 (1 checkpoint:human-verify/blocking-human, 2 TDD)
- **Files:** ~44 created (36 vendored + 6 authored + 2 todos), 3 modified
- **Commits:** 5

## Accomplishments

- Vendored shadcn-svelte's `command` component (registry v1.5.1) plus its five own registry-composed dependencies (`dialog`, `button`, `input-group`, `input`, `textarea` — 36 files total), human-reviewed file-by-file for network/filesystem/dynamic-eval access (zero hits, positive-controlled) and unescaped-HTML surface (zero `{@html}`, zero `innerHTML`/`outerHTML`/`insertAdjacentHTML`/`createContextualFragment`).
- `web/src/lib/search.ts`: `createSearchController(client)` — a `svelte/store`-shaped controller exposing `setQuery()` (debounced Search+Files, AbortController + monotonic request-identity guard, no client-side sort/rank/clamp) and `submit()` (independent Explore lineage, never clears live results).
- `web/src/lib/components/browse/SearchPanel.svelte`: Symbols/Files/Explore sections in D-15's order, `/`/`Cmd+K`/`Ctrl+K`/`Escape` document-level shortcuts, arrow-key traversal and Enter-select delegated entirely to the vendored primitive (confirmed to already cross group boundaries), Explore submission via a single leading "Ask" item reusing that same mechanism.
- Mounted into `web/src/routes/browse/+page.svelte`, reading the `q` URL param; selection sets page state directly via the named, single-occurrence `PLACEHOLDER-03-07` marker that 03-07 Task 3 removes.
- Manual UAT against a real `codegraph ui` server (rebuilt binary, scratch-built SPA temporarily swapped into `web/build/`, restored byte-identical afterward) on this repository's own index, via `agent-browser` — every behavior in the plan's `<behavior>` list exercised live, not just in jsdom.
- Filed two todos: the already-required shadcn-svelte registry-version-pinning deferral (D-22), and a new one for the `Files` RPC's glob-cannot-cross-directories limitation discovered during UAT.

## Task Commits

Each task was committed atomically, following RED-GREEN TDD discipline for Tasks 2 and 3:

1. **Task 1: Vendor the Command component and review what it brought with it** (checkpoint:human-verify, gate=blocking-human)
   - `205da68` — `feat(03-06): vendor shadcn-svelte Command component, human-approved`
2. **Task 2: The search controller — debounce, cancellation, and the trigger split**
   - `9ca7214` — `test(03-06): add failing test for search controller debounce/cancellation/trigger-split` (RED: `Failed to resolve import "$lib/search"`)
   - `e504838` — `feat(03-06): implement search controller — debounce, cancellation, trigger split` (GREEN: 10/10 passing, 34/34 total)
3. **Task 3: The SearchPanel component, its three sections, and the keyboard bindings**
   - `e3ef6aa` — `test(03-06): add failing test for SearchPanel rendering and keyboard bindings` (RED: `Failed to resolve import "$lib/components/browse/SearchPanel.svelte"`)
   - `1ebb841` — `feat(03-06): implement SearchPanel — three sections, keyboard bindings` (GREEN: 10/10 passing, 44/44 total)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/src/lib/components/ui/{command,dialog,button,input-group,input,textarea}/` — vendored shadcn-svelte source (36 files), human-reviewed
- `web/src/lib/search.ts` — the search controller
- `web/src/lib/components/browse/SearchPanel.svelte` — the search UI + keyboard bindings
- `web/src/routes/browse/+page.svelte` — mounts SearchPanel, PLACEHOLDER-03-07 selection handler
- `web/tests/search.test.ts`, `web/tests/search-panel.test.ts` — edge coverage
- `web/package.json`, `web/pnpm-lock.yaml` — bits-ui, @internationalized/date, 10 transitive packages
- `.planning/todos/pending/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md` — required by Task 1's acceptance criteria
- `.planning/todos/pending/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md` — filed from the Task 3 UAT finding

## Decisions Made

See `key-decisions` in frontmatter for full rationale on: the vendor-approval scope (human checkpoint), the `textarea`/`input-group-textarea.svelte` keep decision, the keyboard-capability finding, the Files-pattern depth limitation, the Explore-submission-via-item-select design, and the `SearchSelection` union shape.

## Deviations from Plan

### Auto-fixed / Documented Issues

**1. [Rule 1 - Bug] Svelte 5 `$state()`-rune naming collision — `svelte-check` failed on a variable literally named `state`**
- **Found during:** Task 3, running the plan's own `pnpm check` acceptance criterion after first draft of SearchPanel.svelte
- **Issue:** `let state = $state<T>(...)` produced three `svelte-check` errors ("used before its declaration", implicit `any`, "untyped function calls may not accept type arguments") — a known Svelte 5 compiler quirk when a `$state`-backed local variable is named `state`
- **Fix:** Renamed the local variable to `searchState` throughout the component (declaration, effect assignment, all template references)
- **Files modified:** `web/src/lib/components/browse/SearchPanel.svelte`
- **Verification:** `pnpm check` — 0 errors, 0 warnings after the rename
- **Committed in:** `1ebb841` (Task 3 commit)

**2. [Rule 1 - Bug] `document.body.innerHTML +=` in a test silently broke `@testing-library/svelte`'s cleanup for every later test in the same file**
- **Found during:** Task 3, first run of `search-panel.test.ts` — 7 of 10 tests failed with "multiple elements found" errors showing duplicate `data-testid="search-panel"` divs accumulating across tests
- **Issue:** `document.body.innerHTML += '<input .../>' ` re-serializes and recreates the ENTIRE body subtree, including the just-rendered component's container — testing-library's cleanup-by-reference then silently no-ops on the stale reference, leaking every subsequent test's render into a shared, growing DOM
- **Fix:** Replaced with `document.createElement('input')` + `appendChild` (and an explicit `.remove()` at the end of that one test), which doesn't touch the rendered component's own container
- **Files modified:** `web/tests/search-panel.test.ts`
- **Verification:** all 10 tests pass individually and as a full-file run afterward
- **Committed in:** `e3ef6aa` (Task 3 RED commit, discovered and fixed before the RED observation was recorded)

**3. [Rule 1 - Bug] Ambiguous `getByText` matches against a `Command.Item`'s own sibling `<span>` (file path / line number)**
- **Found during:** Task 3, first run of `search-panel.test.ts`
- **Issue:** `screen.getAllByText(/^(Zeta|Alpha)$/)` and similar plain-text queries returned elements whose full `textContent` included the item's sibling metadata span (e.g. `"Zeta a.go:1 "`), not the bare symbol name — an artifact of how the two text nodes share one parent `Command.Item`
- **Fix:** Added `data-testid` attributes (`search-item-${key}`) to every rendered item and rewrote the affected assertions to query by testid instead of text content
- **Files modified:** `web/src/lib/components/browse/SearchPanel.svelte`, `web/tests/search-panel.test.ts`
- **Verification:** all affected tests pass with unambiguous single-element matches
- **Committed in:** `1ebb841` (component) / `e3ef6aa` (test, fixed before RED was recorded)

**4. [Documented, not "fixed" — real architectural constraint] `Files` RPC's glob pattern cannot cross directory boundaries**
- **Found during:** Task 3's mandated manual UAT against a live `codegraph ui` server on this repository's real index — not caught by any unit test, since every test fixture used single-segment file paths
- **Issue:** `Files{pattern: "*detail*"}` returned zero matches for `internal/query/detail.go` even though it is a real, indexed file — `path/filepath.Match`'s `*` never crosses `/` and Go's glob dialect has no recursive `**` (confirmed empirically: `filepath.Match("*/*/*detail*", "internal/query/detail.go")` = true, proving the depth-exactness requirement)
- **Disposition:** Not fixed — the fix is server-side and cross-cutting (`FilesOptions.Pattern`'s matching semantics are shared by the CLI and the MCP `files` tool, not just this RPC caller); changing it would be a Rule 4 architectural decision out of this plan's scope. Documented directly in `search.ts`'s dispatch comment and filed as a todo. Mitigating: `Search`'s own file-kind pseudo-node matches already provide arbitrary-depth file discovery via the Symbols section (confirmed live), so BRW-01's actual intent is met today, just not via the `Files` RPC for nested paths.
- **Files modified:** `web/src/lib/search.ts` (comment only, no behavior change)
- **Committed in:** `1ebb841` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — real bugs caught by running the plan's own acceptance criteria and mandated UAT), 1 documented architectural constraint (not a bug, no clean fix within scope). **Impact:** All three fixes were necessary for the plan's own acceptance criteria (`pnpm check` clean, all tests passing and correctly asserting) to be honestly satisfied. The documented constraint does not block any acceptance criterion — none of Task 2/3's criteria specify Files' pattern-matching depth — but is recorded because it is a real, user-visible gap in the shipped feature's completeness for nested-path repositories, discovered specifically because the mandated manual UAT step ran against real data instead of trusting unit-test fixtures.

## Issues Encountered

None beyond the four items recorded above (all caught, fixed or documented, and verified within the same task rather than escalated).

## TDD Gate Compliance

Task 2 (`type="tdd"`): RED confirmed (`test(03-06): add failing test for search controller...`, `web/tests/search.test.ts` failed to resolve `$lib/search`) before `web/src/lib/search.ts` existed. GREEN confirmed (`feat(03-06): implement search controller...`, 10/10 new tests passing).

Task 3 (`type="auto" tdd="true"`): RED confirmed (`test(03-06): add failing test for SearchPanel...`, `web/tests/search-panel.test.ts` failed to resolve `$lib/components/browse/SearchPanel.svelte`) before the component existed. GREEN confirmed (`feat(03-06): implement SearchPanel...`, 10/10 new tests passing, 44/44 total). No REFACTOR commit — no cleanup was needed beyond what landed directly in the GREEN commit (the three Rule-1 fixes above were resolved before the RED/GREEN split was finalized, not as a separate refactor step).

`git log --oneline --grep="^test(03-06)"` and `--grep="^feat(03-06)"` both return non-empty, confirming both gate commit types exist for this plan.

## User Setup Required

None — no external service configuration required. All new npm packages (`bits-ui`, `@internationalized/date`, 10 transitive) were reviewed and approved in Task 1's human checkpoint; no environment variables or account setup needed.

## Next Phase Readiness

- `web/src/lib/search.ts` and `web/src/lib/components/browse/SearchPanel.svelte` exist in their final architectural shape for this phase — 03-07 (URL writes/history) replaces the `PLACEHOLDER-03-07` call site with a real `goto()`-driven write, without needing to touch either file's own internals.
- The vendored `command`/`dialog`/`button`/`input-group`/`input`/`textarea` surface is available for any later plan in this phase that needs a dialog, button, or text input — no re-vendoring needed.
- The intended-RED `task web:drift` leg (opened 03-01) is still open and unaffected by this plan in the way that matters: SOURCE-half mismatch grew further (65 source files now vs 22 at 03-01, since this plan adds source files), OUTPUT half MATCH (same digest as before this plan — `web/build/` was verified restored byte-identically after the manual-UAT temporary swap via `git diff --stat web/build`, empty). Closes at 03-09 Task 3.
- One real, documented gap for future consideration (not blocking): the `Files` RPC's glob pattern cannot express "contains term at any depth" — filed as a todo, not scoped into 03-07/03-08/03-09 unless a maintainer decides otherwise.
- No blockers for 03-07 or any other wave-5 plan.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

- `test -d web/src/lib/components/ui/command` → FOUND (14 files)
- `test -f web/src/lib/search.ts` → FOUND
- `test -f web/src/lib/components/browse/SearchPanel.svelte` → FOUND
- `test -f web/tests/search.test.ts` → FOUND
- `test -f web/tests/search-panel.test.ts` → FOUND
- `test -f .planning/todos/pending/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md` → FOUND
- `test -f .planning/todos/pending/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md` → FOUND
- `git log --oneline --all | grep -q 205da68` → FOUND
- `git log --oneline --all | grep -q 9ca7214` → FOUND
- `git log --oneline --all | grep -q e504838` → FOUND
- `git log --oneline --all | grep -q e3ef6aa` → FOUND
- `git log --oneline --all | grep -q 1ebb841` → FOUND
- `cd web && pnpm test` → exit 0, 44/44 tests, names `search.test` and `search-panel`
- `cd web && pnpm check` → exit 0, 0 errors, 0 warnings
- `task web:test` → observed numTotalTests=44 numPassedTests=44, PASS
- `task web:deps:strict` / `web:lockfile` / `web:audit` → all PASS
- `rg -o 'PLACEHOLDER-03-07' web/src/ | wc -l` → 1
- `git diff --stat web/build` → empty (byte-identical to HEAD after manual UAT swap)
