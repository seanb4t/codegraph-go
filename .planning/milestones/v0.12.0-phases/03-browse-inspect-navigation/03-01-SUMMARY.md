---
phase: 03-browse-inspect-navigation
plan: 01
subsystem: testing
tags: [vitest, jsdom, testing-library, highlight.js, pnpm, taskfile, ci]

requires:
  - phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
    provides: "web/ SvelteKit app shell, pnpm-workspace.yaml with strictDepBuilds:true, the WEB_HASH_LIB/web:build/web:drift BLD-03 digest pipeline"
provides:
  - "A working JS test harness (vitest + jsdom + @testing-library/svelte + @testing-library/jest-dom) under web/, proven to actually execute via a RED-then-GREEN proof against a real compiled .svelte fixture"
  - "task web:test — a Taskfile target that runs the JS suite and asserts a positive executed-test count before judging pass/fail, wired into CI immediately after the frozen install"
  - "highlight.js@11.12.0 installed as a runtime dependency, ready for plan 03-04's syntax-highlighting module"
  - "The intended-RED web:drift window, opened here and closed by 03-09 Task 3"
affects: [03-04, 03-06, 03-07, 03-08, 03-09]

actuals:
  tokens: 11930
  tasks: 3
  commits: 2

tech-stack:
  added: ["vitest@4.1.11", "jsdom@30.0.1", "@testing-library/svelte@5.4.2", "@testing-library/jest-dom@7.0.1", "highlight.js@11.12.0"]
  patterns: ["vitest test: block lives in vite.config.ts, never a separate vitest.config.ts, so web_source_files() keeps seeing test-config changes in the BLD-03 digest", "resolve.conditions: ['browser'] guarded on process.env.VITEST, so .svelte imports resolve client-side under jsdom without affecting vite build/dev"]

key-files:
  created:
    - web/tests/setup.ts
    - web/tests/harness.test.ts
    - web/tests/fixtures/Harness.svelte
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/vite.config.ts
    - Taskfile.yml
    - .github/workflows/ci.yml

key-decisions:
  - "@testing-library/jest-dom newly tripped the too-new legitimacy flag at install time (published 2026-08-09, after 03-RESEARCH.md was written) and was approved under the same fallback reasoning as the two tabled [SUS] packages — a recent release on a 2019-vintage, tens-of-millions-weekly-download project, not a new or hijacked package."
  - "resolve.conditions: ['browser'] added to vite.config.ts, guarded on process.env.VITEST — the default jsdom resolution picked Svelte's server build for .svelte imports (lifecycle_function_unavailable from mount()), which the plan anticipated as a possible deviation and instructed be recorded here rather than falling back to a DOM-only assertion."

patterns-established:
  - "JS unit-test Taskfile targets print observed counts (numTotalTests, numPassedTests) before any pass/fail judgment, mirroring web:lockfile's existing rule-84d1gfpywd shape — the same discipline every subsequent web: guard in this phase should follow."

requirements-completed: [BRW-06]

coverage:
  - id: D1
    description: "pnpm --dir web test executes the vitest runner, reports a non-zero executed-test count, and exits 0"
    requirement: BRW-06
    verification:
      - kind: unit
        ref: "web/tests/harness.test.ts#renders a real compiled .svelte fixture under jsdom, with DOM matchers registered"
        status: pass
    human_judgment: false
  - id: D2
    description: "The RED-then-GREEN harness proof renders a real compiled Svelte component from web/tests/fixtures/Harness.svelte"
    requirement: BRW-06
    verification:
      - kind: unit
        ref: "web/tests/harness.test.ts (observed RED: expect(1).toBe(2), exit 1, 'Tests  1 failed (1)'; observed GREEN: renders Harness.svelte, exit 0, 'Tests  1 passed (1)')"
        status: pass
    human_judgment: false
  - id: D3
    description: "task web:test prints the observed executed-test count before comparing it to a floor, and fails loudly on a suite that executed zero tests"
    requirement: BRW-06
    verification:
      - kind: unit
        ref: "command: task web:test (RED demonstration against tests-empty-glob-red-proof/**/*.test.ts: numTotalTests=0, exit 1; GREEN after restore: numTotalTests=1 numPassedTests=1, exit 0)"
        status: pass
    human_judgment: false
  - id: D4
    description: "highlight.js resolves at the exact version recorded in pnpm-lock.yaml, and every newly added package entered pnpm-lock.yaml under the top-level packages: mapping"
    requirement: BRW-06
    verification:
      - kind: unit
        ref: "command: rg -n \"^  (highlight\\.js@|vitest@|jsdom@|'@testing-library/(svelte|jest-dom)@)\" web/pnpm-lock.yaml (5 matches: highlight.js@11.12.0, vitest@4.1.11, jsdom@30.0.1, @testing-library/svelte@5.4.2, @testing-library/jest-dom@7.0.1)"
        status: pass
    human_judgment: false
  - id: D5
    description: "task web:deps:strict, task web:lockfile, task web:audit all pass against the post-install lockfile; pnpm install --frozen-lockfile reproduces the same tree with no rewrite"
    requirement: BRW-06
    verification:
      - kind: integration
        ref: "commands: task web:deps:strict (allowBuilds PRESENT, 0 entries/0 denials); task web:lockfile (215 packages, lockfileVersion 9.0, 215/215 integrity-bearing, 0 rejected sources); task web:audit (CLEAN, 0 advisories); pnpm install --frozen-lockfile (sha256 of pnpm-lock.yaml identical before/after)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The intended-RED web:drift window is recorded as opened, with the closing plan named"
    requirement: BRW-06
    verification:
      - kind: integration
        ref: "command: task web:drift (SOURCE-half mismatch observed and recorded below; OUTPUT half MATCH)"
        status: pass
    human_judgment: false

duration: ~5min of active tool-executed work across two waits (the blocking-human package-legitimacy checkpoint, and this session's own read/verify overhead)
completed: 2026-08-28
status: complete
---

# Phase 3 Plan 1: JS Test Harness Bring-Up Summary

**vitest + jsdom + @testing-library/svelte wired into web/, proven RED-then-GREEN against a real compiled `.svelte` fixture, with `task web:test` asserting a positive executed-test count and running in CI**

## Performance

- **Duration:** see frontmatter `duration` — the blocking-human checkpoint (Task 1) is excluded from active-work time; ~5 min of actual command execution across Tasks 2-3
- **Started:** 2026-08-28 (checkpoint gate)
- **Completed:** 2026-08-28T22:52:11Z
- **Tasks:** 3 (Task 1 checkpoint, Task 2 harness bring-up, Task 3 Taskfile/CI wiring)
- **Files modified:** 8 (3 created, 5 modified)

## Accomplishments

- Package-legitimacy checkpoint (Task 1) answered by a human: all five new npm packages (`highlight.js@11.12.0`, `vitest@4.1.11`, `jsdom@30.0.1`, `@testing-library/svelte@5.4.2`, `@testing-library/jest-dom@7.0.1`) confirmed by name and version, live registry data re-verified immediately before presenting and matched independently by the orchestrator. `@testing-library/jest-dom` newly tripped `too-new` (published 2026-08-09, after 03-RESEARCH.md was written) and was approved under the same fallback reasoning as the two tabled `[SUS]` packages — a recent release on a 2019-vintage, 63M-weekly-download project.
- `pnpm --dir web test` now works: vitest configured in `web/vite.config.ts`'s `test:` block (jsdom environment, `tests/**/*.test.ts`, `tests/setup.ts`), never a separate `vitest.config.ts`.
- The harness was proven to actually execute, not just configured: `web/tests/harness.test.ts`'s first committed-to-disk content was a deliberately failing assertion (`expect(1).toBe(2)`), observed RED — exit code **1**, `Tests  1 failed (1)` — before being rewritten to render a real compiled `web/tests/fixtures/Harness.svelte` via `@testing-library/svelte`, observed GREEN — exit code **0**, `Tests  1 passed (1)`.
- `task web:test` added: depends on `web:deps`, runs vitest via pnpm with the JSON reporter written to a scratch file (never piped — the exact RED-reads-as-GREEN shape this repo shipped once in Phase 2), prints observed `numTotalTests`/`numPassedTests` before comparing to `FLOOR=1`. Demonstrated RED against an empty include glob (`numTotalTests=0`, exit 1) and restored byte-identically before demonstrating GREEN.
- `task web:test` wired into CI's existing `test` job as step `web unit tests (vitest)`, immediately after `Install JS deps (frozen lockfile, BLD-01)`.
- `highlight.js` installed as a runtime dependency, ready for plan 03-04.

## Task Commits

1. **Task 1: Package legitimacy gate for the five new npm packages** — no code commit (pure human decision checkpoint; approved)
2. **Task 2: Install packages, configure vitest+jsdom, prove RED before GREEN** — `35c382f` (feat)
3. **Task 3: Add web:test Taskfile target and wire into CI** — `281fd43` (feat)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/tests/setup.ts` — global vitest setup: registers `@testing-library/jest-dom/vitest` and `@testing-library/svelte/vitest`
- `web/tests/harness.test.ts` — the RED-then-GREEN proof; final GREEN state renders `Harness.svelte` and asserts its label via a jest-dom matcher
- `web/tests/fixtures/Harness.svelte` — minimal real Svelte 5 runes component (`{ label } = $props()`), the only implementable shape for the compiled-component proof
- `web/package.json` — added `test`/`test:watch` scripts, five new dependency entries
- `web/pnpm-lock.yaml` — 215 packages (up from 129 pre-Phase-3), all five new packages under the `packages:` mapping with integrity checksums
- `web/vite.config.ts` — added the `test:` config block and a `resolve.conditions: ['browser']` deviation guarded on `process.env.VITEST`
- `Taskfile.yml` — new `web:test` target, placed after `web:deps:strict` and before `web:build`
- `.github/workflows/ci.yml` — new `web unit tests (vitest)` step in the `test` job

## Decisions Made

- **`@testing-library/jest-dom`'s fresh `too-new` flag** was approved under the checkpoint's existing fallback reasoning rather than treated as a new blocker, since its download count (63M/week) and multi-year GitHub history under `testing-library/jest-dom` are inconsistent with a hijacked or newly-created package.
- **`resolve.conditions: ['browser']`** was added to `vite.config.ts`, guarded strictly on `process.env.VITEST` (which vitest itself sets), after the unguarded default resolution picked Svelte's server-side build for `.svelte` imports under jsdom and threw `lifecycle_function_unavailable` from `mount()`. This was explicitly anticipated by the plan as a possible deviation; per the plan's own instruction, no DOM-only fallback assertion was substituted — the fix was applied at the resolution layer instead, so the harness genuinely compiles and renders a `.svelte` component.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `.svelte` fixture import resolved to Svelte's server build under jsdom**
- **Found during:** Task 2, step (g) (the GREEN conversion)
- **Issue:** `render(Harness, ...)` from `@testing-library/svelte` threw `Svelte error: lifecycle_function_unavailable — mount(...) is not available on the server`. Vite's default package-export condition resolution under jsdom picked `svelte/src/internal/server/*` instead of the client build.
- **Fix:** Added `resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined` to `web/vite.config.ts`, scoped to vitest only so `vite build`/`vite dev` resolution is unaffected.
- **Files modified:** `web/vite.config.ts`
- **Verification:** `pnpm test` went from failing with the above error to `Tests  1 passed (1)`, exit 0.
- **Committed in:** `35c382f` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, explicitly anticipated and pre-authorized by the plan text).
**Impact on plan:** Necessary for the harness to genuinely compile and render a `.svelte` component, which is the entire point of BRW-06's proof requirement. No scope creep — scoped to the one config key the plan named in advance.

## Package Approval Outcome (`pnpm approve-builds --all`)

`pnpm approve-builds --all` reported **"There are no packages awaiting approval"** — the second of the two possible outcomes the plan named. `web/pnpm-workspace.yaml` was left byte-identical: `git diff --exit-code web/pnpm-workspace.yaml` exits 0. The hand-authored empty `allowBuilds: {}` map (committed in 02-07) still stands unmodified.

## RED-Then-GREEN Harness Proof (raw observations)

**RED** (`web/tests/harness.test.ts` = `expect(1).toBe(2)`):
```
FAIL  tests/harness.test.ts > harness RED proof > is deliberately false, observed failing before any GREEN result is trusted
AssertionError: expected 1 to be 2 // Object.is equality
 Test Files  1 failed (1)
      Tests  1 failed (1)
```
Exit code: **1**

**GREEN** (rewritten to render `web/tests/fixtures/Harness.svelte`):
```
 Test Files  1 passed (1)
      Tests  1 passed (1)
```
Exit code: **0**

**Plan `<verify>` (JSON reporter mechanism, same one `task web:test` uses):**
```
observed numTotalTests=1 numPassedTests=1
```
Exit code: **0**

## `task web:test` RED-Then-GREEN Demonstration (raw observations)

**RED** (vitest `include` temporarily pointed at `tests-empty-glob-red-proof/**/*.test.ts`, a directory with no test files):
```
web:test: observed numTotalTests=0 numPassedTests=0 (vitest exit 1)
::error::web:test: executed 0 tests, want at least 1 — a suite that ran ZERO tests must fail loudly, never read as a clean pass. This floor catches 'executed nothing'; it is not a suite-growth ratchet and must never be raised.
```
Task exit code: **201** (task's wrapper for the underlying shell's exit 1)

`web/vite.config.ts` restored byte-identically afterward — `git diff web/vite.config.ts` was clean before the GREEN re-run.

**GREEN** (glob restored to `tests/**/*.test.ts`):
```
web:test: observed numTotalTests=1 numPassedTests=1 (vitest exit 0)
web:test: PASS — 1 of 1 tests passed
```
Task exit code: **0**

## Intended-RED `web:drift` Window (reproduced verbatim from the plan)

| | |
|---|---|
| Window opens | this plan (wave 1), at the `web/package.json` + `web/vite.config.ts` commit |
| Window closes | plan **03-09 Task 3**, at `task web:build` + the re-committed `web/build/` |
| Red leg | the `web:drift` step of the `test` job in `.github/workflows/ci.yml` |
| Expected other legs | every OTHER leg of the `test` job stays green throughout; a second red leg is a real regression, not this window |

**Observed opening (this session):**
```
web:drift: hashed 22 source files
web:drift: manifested 25 output files
::error::web:drift: SOURCE-half mismatch — the committed build output is stale relative to its source (marker: 22 files / fc4ae27b478f9a94106a7bd7d055223c87109193cff97777d2213dc56d3106eb; recomputed: 22 files / 450af9024d2c75b9fbc3ebcb129d35d757a133ace427ea7e1d6647ae85d3d112). Run `task web:build` to rebuild.
web:drift: output half MATCH (25 files, c599a63ecd65a45ef677a209b93f62e7296d3b2b9df91b0ca291dff35c0b57ec)
```
Task exit code: **201**. The OUTPUT half matches (nobody hand-edited `web/build/`); only the SOURCE half diverges, exactly as predicted — `web/package.json`, `web/pnpm-lock.yaml` and `web/vite.config.ts` changed in this plan, and `web/build/` was not rebuilt. This is the plan's own scheduled, expected-red leg; every other `test` job leg (`web:deps`, `web:deps:strict`, `web:audit`, `web:build:verify`) stayed green in this session.

## Issues Encountered

None beyond the one recorded deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `pnpm --dir web test` and `task web:test` are both live and CI-wired; every remaining `<verify>` in plans 03-04 and 03-06..03-09 that names a JS test command now has a real runner underneath it.
- `highlight.js@11.12.0` is installed and lockfile-visible, ready for plan 03-04's syntax-highlighting registration module.
- The intended-RED `web:drift` leg is now live in CI and will stay red until 03-09 Task 3 rebuilds `web/build/` — this is expected, not a regression, for every plan in between.
- No blockers for 03-02 (path confinement RPC-boundary regression test), which does not depend on the JS harness.

## Self-Check: PASSED

- `test -f web/tests/setup.ts` → FOUND
- `test -f web/tests/harness.test.ts` → FOUND
- `test -f web/tests/fixtures/Harness.svelte` → FOUND
- `git log --oneline --all | grep -q 35c382f` → FOUND
- `git log --oneline --all | grep -q 281fd43` → FOUND
- `task web:test` → exit 0, `numTotalTests=1 numPassedTests=1`
- `task web:deps:strict && task web:lockfile && task web:audit` → all exit 0
- `task lint:actions` → exit 0
- `go test ./internal/upgrade/ -run '^TestTaskfileWrapperIsSerial$' -count=1 -v` → `--- PASS: TestTaskfileWrapperIsSerial`
- `go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v` → `--- PASS: TestWorkflowRunBodiesInvokeTask`
- `task web:drift` → exit non-zero, SOURCE-half mismatch as predicted (intended-RED window)

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-28*
