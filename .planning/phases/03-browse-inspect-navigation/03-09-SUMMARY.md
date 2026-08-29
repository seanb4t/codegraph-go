---
phase: 03-browse-inspect-navigation
plan: 09
subsystem: ui
tags: [svelte, status, degrade-states, build, drift-guard, uiserver]

requires:
  - phase: 03-browse-inspect-navigation (plan 03-01)
    provides: "vitest/jsdom/@testing-library/svelte JS test harness, task web:test, and the intended-RED web:drift window this plan closes"
  - phase: 03-browse-inspect-navigation (plan 03-04)
    provides: "rpc-errors.ts's classifyRpcError — the error half of D-04's degrade contract this plan's status.ts completes the answer half of"
  - phase: 03-browse-inspect-navigation (plan 03-06/03-07)
    provides: "+layout.svelte's existing page.url-derived active-navigation state, extended here with the status gate's own effect"
  - phase: 03-browse-inspect-navigation (plan 03-08)
    provides: "SourcePane's completed permalink/click-to-definition/copy surface, extended here with the indexStale-split source-absent message"
provides:
  - "web/src/lib/status.ts — the shared status gate (D-04/D-05): classifyStatus (five-member StatusVerdict + orthogonal CommitKnowledge), navigationIdentity, createStatusGate(client, initialNavigationIdentity)"
  - "web/src/lib/components/StatusBanner.svelte — the layout-level banner every view inherits"
  - "web/src/routes/+layout.svelte wired with the gate's ONE construction site and ONE navigation trigger, shared to descendants via context"
  - "web/src/lib/components/browse/SourcePane.svelte's indexStale-split source-absent message (D-03)"
  - "A rebuilt, re-committed web/build/ with both drift digests reflecting this phase's full source — closing the intended-RED web:drift window 03-01 opened"
affects: [04, 05, 06]

actuals:
  tokens: 7296
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "A shared reactive gate exposed via the store contract (subscribe(run) => unsubscribe, invoked synchronously at least once) so a plain .ts module can be consumed with Svelte context/$state wiring in the layout AND independently subscribed to a second time by a descendant route reading the same data — one fetch trigger, many listeners."
    - "A constructor-supplied initial identity (rather than a post-construction call) is what makes an identity guard's first-effect-run-is-a-no-op property testable at all — closing a double-fetch class of bug structurally rather than by convention."

key-files:
  created:
    - web/src/lib/status.ts
    - web/src/lib/components/StatusBanner.svelte
    - web/tests/status.test.ts
    - web/tests/degrade-states.test.ts
  modified:
    - web/src/routes/+layout.svelte
    - web/src/routes/browse/+page.svelte
    - web/src/lib/components/browse/SourcePane.svelte
    - web/build (rebuilt, 27 files)
    - web/build/.build-manifest

key-decisions:
  - "createStatusGate exposes its reactive status via the Svelte/store contract (subscribe(run)) rather than a bespoke callback shape, so both +layout.svelte's own $state binding and browse/+page.svelte's independent subscription (for the D-03 stale-source-message split) can consume the SAME gate instance with zero adapter code, and subscribing never triggers a second fetch."
  - "The shared gate is passed to descendant routes via Svelte context (setContext('statusGate', gate) at the layout), not re-derived: browse/+page.svelte subscribes to the SAME gate the layout's banner reads from, so the stale-source-message split never becomes a second fetch trigger."
  - "The 'unknown' verdict (a rejected GetStatus call, or the transient pre-first-fetch state) renders no banner, matching the 'ok' verdict's silence — it is not one of NAV-04's three named degrade states, and an alarming banner on every page load before the first fetch resolves would be worse than brief silence."

patterns-established:
  - "The plan-level gate sweep in Task 3 binds its own deliverables (status.test/degrade-states executing, StatusBanner mounted, this plan's own banner text in the rebuilt bundle, the manifest digest actually moving) rather than re-running only pre-existing green targets — the template this phase's own review history flagged as vacuous in 03-01/03-04's cycle reviews."

requirements-completed: [NAV-04]

coverage:
  - id: D1
    description: "GetStatus's field combination classifies into a named five-member health verdict (ok/stale/no-index/indexing/unknown) that never collapses the two false-initialized cases, plus an orthogonal commit-knowledge field that never becomes a sixth verdict"
    requirement: NAV-04
    verification:
      - kind: unit
        ref: "web/tests/status.test.ts (12 tests: all five verdicts, no-index vs indexing paired distinctness, commit-knowledge orthogonality with two paired tests, a rejected call never throwing)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The status gate fetches exactly once on creation and once per genuinely-different navigation identity, never on a timer, and the constructor's initial identity closes the initial-load double-fetch by construction (1 -> 1 -> 2 fetch-count contract)"
    requirement: NAV-04
    verification:
      - kind: unit
        ref: "web/tests/status.test.ts (no-polling test pairing an unchanged call count across advanced fake-timer time with an increased count after navigation; identity-guard test asserting 1/1/2)"
        status: pass
      - kind: other
        ref: "command: rg -o 'setInterval|setTimeout|requestAnimationFrame' web/src/lib/status.ts | wc -l = 0, paired with rg -o 'getStatus' web/src/lib/status.ts | wc -l = 2"
        status: pass
    human_judgment: false
  - id: D3
    description: "No index, a stale index and a missing symbol each render an explicit named state; the stale banner and an in-view not-found state render together without either suppressing the other; the source-absent message splits by the index's stale flag (D-03)"
    requirement: NAV-04
    verification:
      - kind: unit
        ref: "web/tests/degrade-states.test.ts (12 tests: four StatusBanner verdicts including ok/unknown silence, pairwise-different degraded text, not-found and invalid-input in-view states, stale-banner-plus-not-found co-presence, the two source-absent messages differing)"
        status: pass
      - kind: manual_procedural
        ref: "live codegraph ui, three real scenarios (screenshots in .planning/phases/03-browse-inspect-navigation/uat-03-09/): a directory with no .codegraph/ shows the no-index banner naming `codegraph init`; a genuinely staled index (touched source file, no re-index) shows the stale banner naming `codegraph index`; a nonexistent symbol on that same stale index shows the in-view not-found state naming the symbol while the stale banner still renders — both simultaneously, confirmed via accessibility-tree snapshot and screenshot"
        status: pass
    human_judgment: false
  - id: D4
    description: "The rebuilt, committed web/build/ demonstrably contains this plan's own work (not merely a chain of pre-existing green targets), and task web:drift is green — closing the intended-RED window 03-01 opened"
    requirement: NAV-04
    verification:
      - kind: integration
        ref: "commands: task web:build (source-sha256 fc4ae27b...->f9a3632a..., 22->73 source files; output-sha256 c599a63e...->5162b279..., 25->27 output files); task web:drift (PASS, source half MATCH 73 files, output half MATCH 27 files); rg -o 'codegraph init' web/build/ | wc -l = 1 (observed 0 pre-phase)"
        status: pass
      - kind: manual_procedural
        ref: "codegraph ui built against the rebuilt embedded bundle, run against this repository's own real index; GET / and GET /browse both return 200 with Cache-Control: no-store (SPA fallback) and no dev-server process running; live browser confirmed both routes render the real app (Status page shows 'Index is healthy.', /browse shows the idle state) — the embed directive resolves correctly"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 9: Status Gate, Degrade Rendering & Bundle Close-Out Summary

**A shared status gate (fetch-once-on-load-and-navigation, five-member health verdict plus orthogonal commit knowledge) drives a layout-level banner and a source-pane message split, and the rebuilt, re-committed `web/build/` closes the intended-RED `web:drift` window 03-01 opened — verified green with a manifest digest that provably moved.**

## Performance

- **Duration:** 9 min (commit span, `f01422fb` to `636ca39e`)
- **Started:** 2026-08-28T23:43:54-04:00
- **Completed:** 2026-08-29T03:52:18Z
- **Tasks:** 3
- **Files modified:** 9 (4 created, 3 modified, `web/build/` rebuilt with 30 changed entries)

## Accomplishments

- `web/src/lib/status.ts` (D-04/D-05, Task 1): `classifyStatus` turns `GetStatus`'s field combination into a named `StatusVerdict` (`ok`/`stale`/`no-index`/`indexing`/`unknown`, exactly five members) plus an orthogonal `commit: known | unknown` field — reading `store_exists`/`indexing_in_progress` rather than `initialized` alone, so the two false-initialized degrade cases never collapse. `navigationIdentity(url)` is the one normalizer both call sites use. `createStatusGate(client, initialNavigationIdentity)` fetches exactly once on creation, records the identity it fetched for, and exposes `notifyNavigated(identity)` as a same-identity no-op — closing the initial-load double-fetch by construction, proven by a `1 -> 1 -> 2` fetch-count test.
- `web/src/lib/components/StatusBanner.svelte` (Task 2): renders nothing for `ok`/`unknown`, and a distinct named message for each of `stale` (names `codegraph index`), `no-index` (names `codegraph init` verbatim), and `indexing`.
- `web/src/routes/+layout.svelte` (Task 2): creates the ONE status gate with the page's initial navigation identity, mounts `StatusBanner` above the route slot, shares the gate via Svelte context, and wires its ONE navigation-notify caller through the same exported `navigationIdentity` normalizer used at construction.
- `web/src/routes/browse/+page.svelte` and `SourcePane.svelte` (Task 2, D-03): the browse route subscribes to the SAME shared gate (no second fetch) and passes `indexStale` down; `SourcePane`'s single-def source-absent message now names re-indexing when the index is stale, and states plainly there is no source otherwise — the two texts differ.
- `web/build/` rebuilt and re-committed (Task 3): source digest `fc4ae27b...` -> `f9a3632a...` (22 -> 73 files), output digest `c599a63e...` -> `5162b279...` (25 -> 27 files). `task web:drift` is GREEN, closing the intended-RED window 03-01 opened (opened at `35c382ff`, red for 52 subsequent commits, closed at `636ca39e`).

## Task Commits

Each task followed RED-GREEN TDD discipline (Tasks 1-2) or a single commit (Task 3, `type="auto"` without `tdd="true"`):

1. **Task 1: The shared status gate**
   - `f01422fb` — `test(03-09): add failing test for shared status gate` (RED: `Failed to resolve import "$lib/status"`)
   - `ba01b69b` — `feat(03-09): implement shared status gate (NAV-04)` (GREEN: 12/12)
2. **Task 2: The layout banner and the three explicit states**
   - `d9529229` — `test(03-09): add failing test for status banner and degraded-state rendering` (RED: `Failed to resolve import "$lib/components/StatusBanner.svelte"`, confirmed via move-file/run/restore since the module and the `indexStale` branch already existed in the working tree by the time this test file was authored)
   - `d747735c` — `feat(03-09): implement status banner and completed degrade rendering (NAV-04)` (GREEN: 12/12; full suite 127/127)
3. **Task 3: Rebuild the committed bundle and run the full phase gate sweep**
   - `636ca39e` — `feat(03-09): rebuild and re-commit web/build (closes the 03-01 intended-RED web:drift window)`

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/src/lib/status.ts` — the shared status gate
- `web/src/lib/components/StatusBanner.svelte` — the layout-level banner
- `web/src/routes/+layout.svelte` — gate construction, mount, context, the one navigation trigger
- `web/src/routes/browse/+page.svelte` — subscribes to the shared gate, passes `indexStale` to `SourcePane`
- `web/src/lib/components/browse/SourcePane.svelte` — `indexStale`-split source-absent message
- `web/tests/status.test.ts`, `web/tests/degrade-states.test.ts` — new test coverage
- `web/build/`, `web/build/.build-manifest` — rebuilt, re-committed

## Decisions Made

See `key-decisions` in frontmatter. Summary: the gate exposes its status via the Svelte store contract so both the layout and a descendant route can subscribe to the SAME instance with no adapter code and no second fetch; the shared gate is threaded to descendants via Svelte context; the `unknown` verdict renders no banner, matching `ok`'s silence, since it is not one of NAV-04's three named degrade states.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `degrade-states.test.ts`'s `loadBrowseTarget` calls were missing `BrowseParams`'s required `unknown` field**
- **Found during:** Task 2, running `pnpm check` after authoring the test file
- **Issue:** `BrowseParams` (03-04's `browse-url.ts`) requires an `unknown: Array<[string, string]>` field even when empty; three `loadBrowseTarget` calls in the new test file omitted it, failing `svelte-check` with "Property 'unknown' is missing".
- **Fix:** Added `unknown: []` to all three call sites.
- **Files modified:** `web/tests/degrade-states.test.ts`
- **Verification:** `pnpm check` — 0 errors, 0 warnings
- **Committed in:** `d747735c` (Task 2 GREEN commit — the test file's RED commit already existed by this point, so the fix landed in the same working-tree state as the GREEN implementation commit)

---

**Total deviations:** 1 auto-fixed (blocking type error in this plan's own new test file).
**Impact on plan:** No scope creep — a one-field fix to code this plan itself authored.

## TDD Gate Compliance

Task 1 and Task 2 (both `tdd="true"`): RED confirmed via genuine "Failed to resolve import" errors before each GREEN commit. Task 1's RED was observed directly (module did not exist at test-authoring time). Task 2's RED was confirmed via move-file/run/restore (`StatusBanner.svelte` moved aside, `SourcePane.svelte` reverted to its pre-Task-2 committed content, suite run, both restored byte-identically) — recorded because the implementation was already written in the working tree by the time the RED proof was demonstrated. Task 3 (`type="auto"`, no `tdd="true"`) is a single commit per the plan's own type declaration.

`git log --oneline --grep="^test(03-09)"` returns 2 commits; `--grep="^feat(03-09)"` returns 3 commits (status gate, degrade rendering, bundle rebuild).

## Manual UAT — Verbatim Observations (Task 2 + Task 3)

Against a real `codegraph ui` server, three scenarios, screenshots saved to `.planning/phases/03-browse-inspect-navigation/uat-03-09/`:

**1. No index at all.** A freshly created directory with no `.codegraph/` anywhere. `codegraph ui --path <dir>` -> `/browse` renders the red banner: *"No index was found for this repository. Run `codegraph init` to create one."* (`uat_noindex.png`)

**2. A deliberately staled index.** A fresh git repo, `codegraph init` (files=1 nodes=2 edges=1), then `touch pkg/foo.go` with no re-index. `GetStatus` returned `stale: true`. `/browse` renders the amber banner: *"The index is stale — it may not reflect recent changes. Run `codegraph index` to refresh it."*

**3. A symbol that does not exist, on that same stale index.** `/browse?symbol=NoSuchSymbolXYZ` renders the in-view red state *"Not found: query: symbol \"NoSuchSymbolXYZ\" not found"* — naming the exact symbol requested — WHILE the stale banner from scenario 2 still renders above it, both present simultaneously (accessibility-tree snapshot confirmed both the `status` region and the not-found paragraph in the same tree; screenshot `uat_stale_and_notfound.png` shows both bands rendered together, neither suppressing the other).

**Task 3, step (e) — embed directive resolution.** After rebuilding and re-committing `web/build/`, the Go binary was rebuilt to embed it and run against THIS repository's own real index. `GET /` and `GET /browse` both returned `200 OK` with `Cache-Control: no-store` (the SPA fallback header) and no separate dev-server process was ever started — the response is served entirely from the compiled binary's embedded `embed.FS`. A live browser session confirmed both routes render the real application: `/` shows "Index is healthy." (the `ok` verdict's own status page, distinct from the layout banner which correctly renders nothing for `ok`), and `/browse` shows the idle state.

## Observed Gate Sweep (Task 3, step (c))

| Command | Result |
|---|---|
| `pnpm --dir web test` | exit 0, `numTotalTests=127 numPassedTests=127`; output names `status.test` (12 occurrences) and `degrade-states` (12 occurrences) |
| `task web:build:verify` | PASS — 27 files built into scratch, structural invariants held, committed tree untouched |
| `task web:drift` | PASS — hashed 73 source files, manifested 27 output files, both halves MATCH |
| `task web:deps:strict` | PASS — strictDepBuilds true, allowBuilds committed with 0 entries (0 denials) |
| `task web:lockfile` | PASS — 227 packages, lockfileVersion 9.0, 227/227 integrity-bearing, 0 rejected sources |
| `task web:audit` | PASS — CLEAN, 0 advisories |
| `task web:test` | PASS — observed `numTotalTests=127 numPassedTests=127` |
| `task proto:drift` | PASS — 4 generated files byte-identical to the pinned toolchain's regeneration |
| `task test:unit` | exit 0 — 50 `ok` packages, 0 `FAIL` lines, 57 total package lines (7 no-test-files) |
| `task lint:actions` | exit 0 |
| `go vet ./...` | exit 0 |
| `cd web && pnpm check` | 871 files, 0 errors, 0 warnings |

**Step (b) observed counts (already reflected above):** `web:drift`'s source-file count 73, output-file count 27; `web:build:verify`'s scratch-build file count 27.

## `web/build/.build-manifest` Before and After (verbatim, Task 3 step (a))

**Before** (unchanged since Phase 2, per 03-01's own recorded opening observation):
```
source-files: 22
source-sha256: fc4ae27b478f9a94106a7bd7d055223c87109193cff97777d2213dc56d3106eb
output-files: 25
output-sha256: c599a63ecd65a45ef677a209b93f62e7296d3b2b9df91b0ca291dff35c0b57ec
```

**After** (`task web:build`):
```
source-files: 73
source-sha256: f9a3632a90031569f30bfab7962e9541281a6a930194a6758f97451692c7dd92
output-files: 27
output-sha256: 5162b2794793aa5ef9f25276b7a8578db0e9c96ea57b579d8339bcdbc769c397
```

Both digests differ from their pre-phase values. `git status --porcelain web/build` is empty after committing.

## Test-Count Growth (Task 3 step (d))

- **JS suite:** 1 executed test at the end of plan 03-01 -> **127** executed tests at the end of this phase (`pnpm --dir web test`, `numTotalTests=127 numPassedTests=127`). Strictly greater.
- **Go suite:** 50 `ok` packages / 57 total package lines (7 `[no test files]`) observed at the end of this phase (`task test:unit`, 0 `FAIL` lines). 03-01-SUMMARY.md did not record a Go package count at its own close (it was scoped to the JS harness only), so no direct before/after comparison is available for the Go half — recorded here transparently rather than inventing a baseline number.

## Intended-RED `web:drift` Window — CLOSED

| | |
|---|---|
| Opened | plan 03-01 (wave 1), commit `35c382ff` (`feat(03-01): install JS test harness and prove vitest RED-then-GREEN`) |
| Red for | 52 subsequent commits (every plan from 03-01 through 03-08, plus this plan's own Tasks 1-2) |
| Closed | THIS plan, Task 3, commit `636ca39e` (`feat(03-09): rebuild and re-commit web/build...`) |
| Verification | `task web:drift` run AFTER the rebuild commit: `PASS — hashed 73 source files, manifested 27 output files, committed web/build/ matches both digests` |

The window is closed as of `636ca39e`. Every plan between 03-01 and this one correctly expected `web:drift` red; this is the plan that rebuilds and re-commits, closing it.

## Issues Encountered

None beyond the one recorded deviation above.

## User Setup Required

None — no external service configuration required. No new npm packages this plan.

## Known Stubs

None.

## Next Phase Readiness

- NAV-04 is fully satisfied: all three degrade states (no-index, stale, symbol-not-found) render explicit named states, never a blank pane, and the two arrival mechanisms (GetStatus answer vs. Connect error) are both handled by shared plumbing every future phase inherits.
- The committed `web/build/` is drift-clean and demonstrably contains this phase's own work — Phase 4/5/6 build on a correct, verified bundle rather than a stale Phase-2 one.
- `web/src/lib/status.ts`'s `createStatusGate`/`navigationIdentity` and the context-sharing pattern in `+layout.svelte` are the seam Phase 6 (Live Push) replaces — deliberately small, no timer, no polling loop to tear out.
- Phase 3 is now complete pending its own end-of-phase verification; no blockers for Phase 4 planning.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

- `test -f web/src/lib/status.ts` → FOUND
- `test -f web/src/lib/components/StatusBanner.svelte` → FOUND
- `test -f web/tests/status.test.ts` → FOUND
- `test -f web/tests/degrade-states.test.ts` → FOUND
- `test -f .planning/phases/03-browse-inspect-navigation/uat-03-09/uat_noindex.png` → FOUND
- `test -f .planning/phases/03-browse-inspect-navigation/uat-03-09/uat_stale_and_notfound.png` → FOUND
- `git log --oneline --all | grep -q f01422fb` → FOUND
- `git log --oneline --all | grep -q ba01b69b` → FOUND
- `git log --oneline --all | grep -q d9529229` → FOUND
- `git log --oneline --all | grep -q d747735c` → FOUND
- `git log --oneline --all | grep -q 636ca39e` → FOUND
- `pnpm --dir web test` → exit 0, 127/127 tests, names status.test/degrade-states
- `cd web && pnpm check` → exit 0, 0 errors, 0 warnings
- `task web:drift` → PASS (GREEN — intended-RED window closed)
- `task web:build:verify` → PASS
- `task test:unit` → exit 0, 0 FAIL lines
- `go vet ./...` → exit 0
- `rg -o 'codegraph init' web/build/ | wc -l` → 1
- `git status --porcelain web/build` → empty
