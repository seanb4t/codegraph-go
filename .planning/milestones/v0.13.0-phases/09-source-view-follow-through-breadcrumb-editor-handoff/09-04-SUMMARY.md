---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 04
subsystem: ui
tags: [svelte, localstorage, editor-handoff, playwright, tdd]

# Dependency graph
requires:
  - phase: 09-01
    provides: GetEditorLink rpc, EditorLinkAvailability/EditorTemplateSource enums, EditorPreset, the regenerated TS client
  - phase: 09-03
    provides: per-line source rows and gutter cells to attach editor-link click targets to, breadcrumb-check.mjs to extend, the $derived+untrack reactivity pattern and the poll-until-stable live-gate pattern
provides:
  - "web/src/lib/editor-prefs.ts: the SPA's first localStorage use — readEditorOverride/writeEditorOverride/clearEditorOverride (try/catch-wrapped, degrade to no-override) and templateForRequest, the single seam that resolves an override into a GetEditorLinkRequest.template"
  - "SourcePane.svelte: a load-time GetEditorLink probe per opened target, a resolved <a data-testid=\"editor-link\"> header link, gutter cells that become one-rpc-per-click buttons when BUILDABLE, and the picker toggle/mount point"
  - "web/src/lib/components/browse/EditorLinkPicker.svelte: a pure view over GetEditorLinkResponse/EditorOverride — three wire presets, a custom template field, \"Use server default\", and provenance display"
  - "web/scripts/breadcrumb-check.mjs extended with --editor-url and three recorded fields (editorLinkHref, gutterLinkCount, gutterClickIssuedRpc) proving the whole handoff in a real Chromium against this repository's own index"
affects: [09-05-security-doc]

# Actuals (#2632)
actuals:
  tokens: 15256
  tasks: 3
  commits: 7
plan_head_before: 5f72540c1e7f171060d1ad7dba1f7f577d4273cd

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "editor-prefs.ts: every Storage access wrapped in try/catch, degrading to \"no override\" on absent/malformed/throwing storage — the SPA's first localStorage consumer"
    - "templateForRequest as the single resolution seam: the probe effect and a gutter click both call it, so they always resolve identically for the same override (the assumption-delta invariant)"
    - "Primitive-valued $deriveds (editorTargetPath/Line/Col) gate the probe $effect on VALUE equality, not raw prop identity — the same discipline 09-03 established for filePath/stableClient"
    - "untrack() around the effect's own read of lastPresets, mirroring 09-03's fileSymbolsCache fix, to avoid the effect re-triggering itself when its own .then writes that state"
    - "A picker component that holds no rpc client and no storage access — SourcePane owns both and passes callbacks, keeping the picker a pure view over props"

key-files:
  created:
    - web/src/lib/editor-prefs.ts
    - web/src/lib/components/browse/EditorLinkPicker.svelte
    - web/tests/editor-prefs.test.ts
    - web/tests/editor-link-picker.test.ts
    - web/tests/source-pane-editor-link.test.ts
  modified:
    - web/src/lib/components/browse/SourcePane.svelte
    - web/tests/setup.ts
    - web/scripts/breadcrumb-check.mjs
    - corpora/breadcrumb-check.json
    - web/build/** (rebuilt via task web:build)

key-decisions:
  - "[Rule 3 - blocking] Node >=26 ships a global Web Storage API on by default that silently no-ops without --localstorage-file, and it shadows jsdom's own window.localStorage before jsdom can install it — discovered live via this plan's own localStorage-dependent tests (typeof window.localStorage was 'undefined' before the fix). Fixed with a probe-and-replace in-memory Storage shim in web/tests/setup.ts, scoped to test infrastructure only; a real browser's localStorage is unaffected."
  - "customValue in EditorLinkPicker.svelte is seeded from `override` ONCE at mount ($state, not $derived) so a user's mid-typed custom template is never overwritten by a later override change or a TEMPLATE_INVALID answer — svelte-ignore state_referenced_locally documents the choice"
  - "[Rule 1 - Bug] My own doc-comment prose in EditorLinkPicker.svelte tripped the plan's own case-insensitive Zed-absence guard ('no rendered string here may name Zed') — reworded without the literal, same recurring substring-proxy-gate pattern this project has hit before (08-01/08-02/08-03)"
  - "gutterClickInFlight is a single flag shared across every gutter cell (not per-line), matching T-09-20's 'one rpc per click, never a burst' mitigation exactly"

patterns-established:
  - "First-localStorage-consumer degrade discipline: read/write/clear all try/catch-wrapped, an absent or throwing Storage is indistinguishable from 'no override' to every caller"

requirements-completed: [BRW-11, BRW-12]

coverage:
  - id: D1
    description: "editor-prefs.ts round-trips an override through localStorage and degrades to null/no-op on every failure mode; templateForRequest resolves null/preset/custom identically for the probe and a gutter click"
    requirement: BRW-12
    verification:
      - kind: unit
        ref: "web/tests/editor-prefs.test.ts (15 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "SourcePane's header shows a resolved <a data-testid=\"editor-link\"> from one load-time probe per opened target (file line1/col1, single-def startCol+1); the DISABLED case renders nothing, NO_TEMPLATE/TEMPLATE_INVALID show the reason and open the picker once per mount"
    requirement: BRW-11
    verification:
      - kind: unit
        ref: "web/tests/source-pane-editor-link.test.ts (11 tests)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Gutter cells become one-rpc-per-click buttons only when BUILDABLE; a click burst issues exactly one rpc (in-flight guard) and navigates via window.location.assign, never window.open"
    requirement: BRW-11
    verification:
      - kind: unit
        ref: "web/tests/source-pane-editor-link.test.ts#SourcePane: gutter one-rpc-per-click handoff"
        status: pass
      - kind: e2e
        ref: "corpora/breadcrumb-check.json (gutterLinkCount:418, gutterClickIssuedRpc:true)"
        status: pass
    human_judgment: false
  - id: D4
    description: "EditorLinkPicker renders exactly the wire's three presets, a custom template field, \"Use server default\", and all six EditorTemplateSource provenance mappings; a TEMPLATE_INVALID answer keeps the typed custom value editable rather than clearing it"
    requirement: BRW-12
    verification:
      - kind: unit
        ref: "web/tests/editor-link-picker.test.ts (14 tests)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Choosing a preset or \"Use server default\" from SourcePane's mounted picker writes/clears through editor-prefs.ts and re-runs the probe with the new template"
    requirement: BRW-12
    verification:
      - kind: unit
        ref: "web/tests/source-pane-editor-link.test.ts#SourcePane: picker integration (Task 3)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Zed appears nowhere under web/src (case-insensitive, word-boundary check) and no preset scheme literal exists outside the generated client; D-11 scope guard confirms getEditorLink is referenced only in SourcePane.svelte and the generated client"
    requirement: BRW-12
    verification:
      - kind: unit
        ref: "structural rg gates (positive-controlled) in Task 2/3's <verify> blocks"
        status: pass
    human_judgment: false
  - id: D7
    description: "The whole handoff — header href, gutter buttons, click->rpc — observed in a real Chromium against this repository's own index"
    requirement: "BRW-11, BRW-12"
    verification:
      - kind: e2e
        ref: "corpora/breadcrumb-check.json (success:true, editorLinkHref ends ':1:1', gutterLinkCount:418, gutterClickIssuedRpc:true, all 5 breadcrumb observations agree)"
        status: pass
    human_judgment: true
    rationale: "Chromium proves the rpc fires and the header href resolves; the actual custom-scheme OS handoff (does vscode:// really open the file) and Safari's transient-activation behavior for the async gutter path are outside what headless Chromium can assert — recorded as unverified in 09-SECURITY.md per research Pitfall 1/A3."

duration: 25min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 4: Editor Handoff — Header Link, Gutter, and Picker Summary

**"Open in editor" now resolves server-side into a real `<a href>` from a load-time probe, every line-number gutter cell becomes a one-rpc-per-click target when a link is buildable, and a popover picker lets a developer choose VS Code, Cursor, JetBrains, or a custom template for their browser only — stored in localStorage, sent as the request's template, and proven end-to-end in a real Chromium against this repository's own index (gutterLinkCount 418, a real click-to-rpc observed).**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-12T18:43:00Z
- **Completed:** 2026-09-12T19:04:04Z
- **Tasks:** 3 (all `tdd="true"`, each RED->GREEN; Task 3 carried an additional test(09-04) commit for the live-gate extension)
- **Files modified:** 34 (9 source/test files + rebuilt web/build/** + corpora/breadcrumb-check.json)

## Accomplishments

- `editor-prefs.ts` — the SPA's first localStorage consumer: a try/catch-wrapped override store and `templateForRequest`, the single function both the probe and a gutter click use to resolve a stored preference into the request's template field
- `SourcePane.svelte` widened with `EditorLinkClient`: one `GetEditorLink` probe per opened target drives a resolved header `<a data-testid="editor-link">`, hides everything when the operator disabled links (D-16), and shows a reason + picker toggle otherwise — opening the picker once per mount on first use (D-14)
- Gutter cells become `<button data-testid="gutter-line-N">` only when BUILDABLE; a click pays exactly one rpc (a shared in-flight guard ignores a burst) and navigates via `window.location.assign`, never `window.open`
- `EditorLinkPicker.svelte`: a pure view rendering the wire's three presets, a custom template field, "Use server default", and every `EditorTemplateSource` provenance mapping — a `TEMPLATE_INVALID` answer keeps the typed value editable rather than clearing it, and Cursor/JetBrains carry the honest `[ASSUMED]` note from 09-RESEARCH.md
- `breadcrumb-check.mjs` extended with `--editor-url` and three new recorded fields; a real Chromium run against this repository's own index observed `editorLinkHref` ending `:1:1`, `gutterLinkCount:418`, and a real gutter click issuing `GetEditorLink` with `"line":3`

## Task Commits

Each task followed the RED -> GREEN TDD cycle:

1. **Task 1: editor-prefs.ts**
   - `6a274520` `test(09-04): add failing editor-prefs tests` (RED: 5 failed/10 passed)
   - `f6dc89d9` `feat(09-04): per-browser editor override storage and request-template resolution` (GREEN: 15/15)
2. **Task 2: SourcePane header link + gutter**
   - `70ff5e94` `test(09-04): add failing SourcePane editor-link tests` (RED: 8 failed/1 passed)
   - `340b3b8f` `feat(09-04): header open-in-editor link from the load-time probe and one-rpc-per-click gutter` (GREEN: 30/30 across the four required test files, 523/523 full suite)
3. **Task 3: picker + live gate**
   - `0616e1f1` `test(09-04): add failing editor-link picker tests` (RED: 13 failed/1 passed)
   - `ffa5d505` `feat(09-04): editor-link picker with wire presets, custom template and server-default reset` (GREEN: 14/14 picker, 539/539 full suite)
   - `9e7ed84` `test(09-04): live gate editor-link phases and rebuilt SPA` (real Chromium, success:true)

## TDD Gate Compliance

| Task | RED commit | GREEN commit | RED evidence | GREEN evidence | Status |
|------|-----------|---------------|---------------|-----------------|--------|
| 1 (editor-prefs.ts) | `6a274520` | `f6dc89d9` | 5 failed / 10 passed against null/undefined stubs | 15/15 pass | Pass |
| 2 (SourcePane) | `70ff5e94` | `340b3b8f` | 8 failed / 1 passed against the pre-Task-2 component | 30/30 pass across the four required files; 523/523 full suite | Pass |
| 3 (EditorLinkPicker) | `0616e1f1` | `ffa5d505` | 13 failed / 1 passed against a stub component | 14/14 pass; 539/539 full suite | Pass |

All RED phases were verified INTENTIONAL (assertion failures against real stub responses or real component rendering — never a compile error) before any GREEN commit landed. GSD's TAP-only `tdd-red-evidence` check was not run per this plan's own commit-discipline note (vitest's own `<verify>` gates with `<fails_when>` are the authority here); every RED transcript is pasted below verbatim.

### Task 1 RED transcript (`pnpm -C web exec vitest run tests/editor-prefs.test.ts`)

```
 × templateForRequest > resolves a preset override to that preset template from the given presets list
   AssertionError: expected undefined to be 'cursor://file/{path}:{line}:{col}'
 × templateForRequest > returns a custom template verbatim with no client-side validation
   AssertionError: expected undefined to be 'x://{path}'
 × templateForRequest > returns the identical value for the same override whether called for the probe or a gutter click
   AssertionError: expected undefined to be 'vscode://file/{path}:{line}:{col}'
Test Files  1 failed (1)
     Tests  5 failed | 10 passed (15)
```

### Task 2 RED transcript (`pnpm -C web exec vitest run tests/source-pane-editor-link.test.ts`)

```
 × SourcePane: header "Open in editor" load-time probe > resolves a real <a href> ... (timeout: editor-link never appears)
 × SourcePane: gutter one-rpc-per-click handoff > BUILDABLE gutter cells are buttons ...
Test Files  1 failed (1)
     Tests  8 failed | 1 passed (9)
```

### Task 3 RED transcript (`pnpm -C web exec vitest run tests/editor-link-picker.test.ts`)

```
ReferenceError: path is not defined  (n/a — stub component renders nothing)
 × EditorLinkPicker: renders exactly the wire's three presets ... (getByTestId fails: no such element)
 × EditorLinkPicker: callbacks > calls onUseServerDefault on that button's click
   TestingLibraryElementError: Unable to find an element by: [data-testid="editor-use-server-default"]
Test Files  1 failed (1)
     Tests  13 failed | 1 passed (14)
```

### GREEN transcripts (PASS counts)

- Task 1: `total 15 passed 15`
- Task 2: `total 30 passed 30 failed 0` (source-pane-editor-link + source-pane-breadcrumb + source-pane + browse-page); full suite `total 523 passed 523 failed 0`
- Task 3: `total 14 passed 14` (editor-link-picker); full suite `total 539 passed 539 failed 0`
- Final `task web:test`: `web:test: PASS — 539 of 539 tests passed`
- `pnpm -C web check`: `COMPLETED 1169 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`
- `task web:drift`: `PASS — hashed 114 source files, manifested 32 output files, committed web/build/ matches both digests`
- Live gate (`node web/scripts/breadcrumb-check.mjs`):
  ```
  breadcrumb-check: editor-link href=vscode://file//Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:1:1
  breadcrumb-check: gutterLinkCount=418
  breadcrumb-check: gutterClickIssuedRpc=true
  breadcrumb-check: wrote .../corpora/breadcrumb-check.json — success=true
  ```

## Files Created/Modified

- `web/src/lib/editor-prefs.ts` — `EDITOR_OVERRIDE_STORAGE_KEY`, `EditorOverride`, `readEditorOverride`, `writeEditorOverride`, `clearEditorOverride`, `templateForRequest`
- `web/src/lib/components/browse/EditorLinkPicker.svelte` — the picker popover and its test ids
- `web/src/lib/components/browse/SourcePane.svelte` — `EditorLinkClient`, the probe `$effect`, `editorLinkSurface` snippet, `handleGutterClick`, `handleChoosePreset`/`handleApplyCustom`/`handleUseServerDefault`/`handlePickerKeydown`
- `web/tests/editor-prefs.test.ts`, `web/tests/editor-link-picker.test.ts`, `web/tests/source-pane-editor-link.test.ts` — unit coverage for the above
- `web/tests/setup.ts` — the Node-webstorage-shim fix (see Deviations)
- `web/scripts/breadcrumb-check.mjs` — `--editor-url`, `editorLinkHref`/`gutterLinkCount`/`gutterClickIssuedRpc`/`notes` recording
- `corpora/breadcrumb-check.json` — re-recorded GREEN diagnostic
- `web/build/**` and `web/build/.build-manifest` — rebuilt (`task web:build`), verified via `task web:drift`

## Decisions Made

See `key-decisions` in frontmatter. In short: the Node 26 localStorage shim fix, seeding the picker's custom-value field once at mount (not `$derived`), a doc-comment reword to clear the plan's own Zed-absence guard, and a single shared in-flight flag for the gutter's click guard.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - blocking] Node >=26's built-in global Web Storage API is on by default but non-functional, and shadows jsdom's real localStorage**
- **Found during:** Task 2 GREEN phase, first run of `source-pane-editor-link.test.ts`'s pre-seeded-override test
- **Issue:** `typeof window.localStorage` was `'undefined'` in this jsdom test environment (contra 09-PATTERNS.md's "jsdom provides a working localStorage" assumption) — Node 26 installs a global `localStorage`/`sessionStorage` getter that no-ops with an `ExperimentalWarning` unless the process is started with `--localstorage-file`, and this global is installed on `globalThis` before jsdom constructs its own window, so jsdom's real Storage implementation never wins the property.
- **Fix:** `web/tests/setup.ts` now probes `globalThis.localStorage` at suite startup (a real set/get/remove round trip) and, if it does not work, replaces it with a minimal in-memory Storage shim — scoped to test infrastructure only; a real browser's `localStorage` (and the live-gate's real Chromium) is unaffected.
- **Files modified:** `web/tests/setup.ts`
- **Verification:** every localStorage-dependent test in `editor-prefs.test.ts` and `source-pane-editor-link.test.ts` passes; the whole 539-test suite is green.
- **Committed in:** `340b3b8f` (part of Task 2's GREEN commit)

**2. [Rule 1 - Bug] My own doc-comment prose tripped the plan's own Zed-absence guard**
- **Found during:** Task 3 GREEN phase, the `<verify>` gate `rg -o -i '\bzed\b|zed://' web/src`
- **Issue:** `EditorLinkPicker.svelte`'s header comment explained the Zed-absence rule by naming the excluded editor directly ("no rendered string here may name Zed in any case") — the case-insensitive positive-controlled guard counts occurrences across the whole file text, comments included, and cannot distinguish my own explanatory prose from a real violation.
- **Fix:** Reworded the sentence to describe the same rule without repeating the literal name.
- **Files modified:** `web/src/lib/components/browse/EditorLinkPicker.svelte`
- **Verification:** `rg -o -i '\bzed\b|zed://' web/src` returns 0.
- **Committed in:** `ffa5d505` (part of Task 3's GREEN commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 1, 1 Rule 3). **Impact:** the Rule 3 fix is test-infrastructure-only and does not touch production code; the Rule 1 fix is a documentation-wording correction with zero behavior change. No scope creep.

## Issues Encountered

None beyond the two auto-fixed items above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- BRW-11/BRW-12's UI half is complete and proven end-to-end in a real Chromium against this repository's own index. `09-05` (`09-SECURITY.md`) can now write BRW-13's threat register against a stable implementation — T-09-02/T-09-06/T-09-09/T-09-13/T-09-20/T-09-19 all have a concrete test or a recorded verdict to cite.
- **Cursor and JetBrains preset templates remain `[ASSUMED]`** (09-RESEARCH.md A1/A2), visibly flagged in the picker's own UI and logged to `.planning/WINDOWS.md` (entry 35, `unrun-verify`) — pending an end-of-phase human check against a real installed IDE, per this project's "found live, not assumed" discipline.
- No blockers.

## Self-Check: PASSED

- All 5 created files verified present on disk (`web/src/lib/editor-prefs.ts`, `web/src/lib/components/browse/EditorLinkPicker.svelte`, three test files).
- All 7 commit hashes (`6a274520`, `f6dc89d9`, `70ff5e94`, `340b3b8f`, `0616e1f1`, `ffa5d505`, `9e7ed84`) verified present via `git log --oneline`.
- Full `pnpm -C web exec vitest run` (539 tests), `pnpm -C web check` (0 errors/0 warnings), and `task web:drift` re-confirmed green at HEAD before writing this SUMMARY.

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*
