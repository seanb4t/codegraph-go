---
phase: 05-file-package-graph-view
plan: 04
subsystem: ui
tags: [playwright, measurement, fail-closed, tdd, gate, verdict]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-01's locked corpora/graph-render-threshold.json; 05-02's FileGraph rpc and measured response size (3,713,528 bytes); 05-03's GraphCanvas.svelte measurement seam (window.__codegraphFileGraphMetrics) and the filled /graph route"
provides:
  - "web/scripts/graph-verdict.mjs — the fail-closed comparator (compareObservation) and its CLI entry point"
  - "web/scripts/graph-measure.mjs — the always-records measurement wrapper (runSession, withDeadline, OPS_KEYS, failedObservation, mergeRawObservations, readProtocol, sessionConfig)"
  - "corpora/graph-render-observations.json — GRF-01's committed measurement, verdict FAIL"
  - "A real, reproduced finding: the shipped renderer (cytoscape + cytoscape-elk/elkjs) does not become interactive within 60 seconds — and was independently observed still not interactive after 10+ minutes — at google/guava's expanded file-level scale (3,233 nodes / 21,554 edges)"
affects: [05-05, 05-06, 05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 21561
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added:
    - "@playwright/test@1.62.1 (direct, exact-pinned devDependency) — the browser driver behind graph-measure.mjs"
    - "@types/node@^24 (devDependency) — Rule 3 blocking-issue fix; pnpm check requires Node ambient types for the two new .mjs scripts"
  patterns:
    - "Injected-ops session runner: the six browser operations (launch, newPage, navigate, seamReady, sample, close) are passed into runSession() as a plain object, so a test can drive a never-settling promise into the exact live code path instead of testing a deadline helper in isolation"
    - "write-before-cleanup: the finally block writes the raw observation FIRST, then races teardown against its own deadline — a wedged close costs the session, never the record"
    - "closure-safe async result propagation: doWork() RETURNS its collected result rather than mutating outer let-bound variables from inside its own closure, specifically to keep TypeScript's control-flow narrowing sound under checkJs+strict (a variable mutated inside a nested closure keeps its widened declared type at every read site outside that closure)"

key-files:
  created:
    - web/scripts/graph-verdict.mjs
    - web/scripts/graph-measure.mjs
    - web/tests/graph-verdict.test.ts
    - web/tests/graph-measure.test.ts
    - corpora/graph-render-observations.json
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/build

key-decisions:
  - "Task 2's literal action text says `task build`, but that target is compile-check-only (go build ./...) and retains no binary. Used `task build:release` instead (Rule 3 — a blocking issue: the plan's literal command cannot produce the ./codegraph binary Task 2 needs to run `codegraph ui`)."
  - "Rule 3 blocking-issue fix: added @types/node@^24 as a devDependency. It was not anticipated by the plan (no prior .mjs script existed in web/), but `cd web && pnpm check` — a hard Task 1 acceptance criterion — fails without Node ambient types once graph-measure.mjs/graph-verdict.mjs use node:fs/node:path/node:url and process.*."
  - "The collapsed directory-level count's exact definition was reverse-engineered against the threshold's own recorded approxNodes/approxEdges (134/819) rather than guessed: 'top-level directory' collapsing produced 7/14 (wrong); 'immediate parent directory of each file, DIRECTED distinct cross-directory pairs, self-pairs excluded' produced 134/819 — an EXACT match to both figures the threshold already committed. Documented in graph-measure.mjs's --collapsed-nodes/--collapsed-edges CLI flags rather than hand-computed once and pasted into the artifact."
  - "Task 3 (checkpoint:decision, gate=blocking-human) is NOT answered in this SUMMARY. auto_advance is true for this project, but blocking-human checkpoints are never auto-approved in any mode — the executor halted and returned the recorded verdict to the orchestrator/human rather than selecting an option. This SUMMARY's status is `halted`, not `complete`, per the template's own guidance for a designed stop that intentionally leaves a task unfinished."

requirements-completed: []

coverage:
  - id: D1
    description: "A committed, fail-closed comparator (graph-verdict.mjs) that judges bindingObservation.metrics against the locked threshold, resolving every ambiguity (absent metric, non-numeric value, self-contradictory record, unreadable threshold, wrong-corpus binding) to FAIL or a hard throw, and copies additionalCorpora through untouched."
    requirement: GRF-01
    verification:
      - kind: unit
        ref: "web/tests/graph-verdict.test.ts (18 tests, all pass)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A committed, always-records measurement wrapper (graph-measure.mjs) whose finally block writes the raw observation before any cleanup is awaited, with every one of the six enumerated OPS_KEYS operations raced against a deadline read from the locked measurementProtocol block, and whose coverage is proven structurally (closed-set rejection + a six-case never-settling sweep) rather than by pattern-matching call sites."
    requirement: GRF-01
    verification:
      - kind: unit
        ref: "web/tests/graph-measure.test.ts (22 tests, all pass)"
        status: pass
      - kind: other
        ref: "Three RED experiments against scratch copies outside the repo — mechanism stubbed wholesale (all 6 sweep cases fail), deadline removed from sample alone (exactly 1 of 6 fails), cleanup awaited before write (wedged-close ordering assertion fails) — see 'Deadline Discrimination Experiments' below"
        status: pass
    human_judgment: false
  - id: D3
    description: "GRF-01's measurement exists as a committed artifact (corpora/graph-render-observations.json) whose verdict was computed by graph-verdict.mjs (never hand-typed), whose threshold provably predates it, and whose Task 2 verify command exits 0 for this well-formed FAIL — a FAIL is this gate working, not a broken plan."
    requirement: GRF-01
    verification:
      - kind: other
        ref: "Task 2's verify command (node -e ... over corpora/graph-render-observations.json), run inline during execution — see 'Task 2 Verify Output' below"
        status: pass
      - kind: other
        ref: "git merge-base check: the threshold's lock commit (2fb2774) is an ancestor of the observation commit (1cb8c71e)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A human reads the recorded FAIL verdict and either releases 05-05/05-06/05-07 or halts them onto one of the two threshold-named remedies."
    requirement: GRF-01
    verification: []
    human_judgment: true
    rationale: "Task 3 is a checkpoint:decision with gate=blocking-human, deliberately never auto-approved (even though this project's auto_advance is true) because an auto-selected 'release' would walk straight through a FAIL verdict unread, which D-02 forbids. Not yet answered — this is the open item this SUMMARY exists to surface."

duration: ~75min
completed: 2026-08-30
status: halted
---

# Phase 5 Plan 4: GRF-01 Measurement — Verdict FAIL Summary

**Built a fail-closed comparator and an always-records Playwright measurement wrapper (both TDD, 40 new tests), then measured the shipped file-graph view against the pinned `google/guava` corpus: the renderer's layout never became interactive within the locked 60-second seam deadline — independently reproduced still not interactive after 10+ minutes of unbounded observation — so GRF-01's committed verdict is FAIL, and Task 3's human decision checkpoint is now open.**

## Performance

- **Duration:** ~75 min (includes ~11 minutes of real browser measurement time: two 60-second binding timeouts, one successful ~15s additional-corpus run, plus a ~10-minute unbounded diagnostic probe)
- **Started:** 2026-08-30T17:40:00Z (approx)
- **Completed:** 2026-08-30T18:55:00Z (approx, at the Task 3 halt)
- **Tasks:** 2 of 3 complete (Task 3 is the open blocking-human checkpoint)
- **Files modified:** 8 (5 created, 3 modified — see key-files)

## Accomplishments

- `web/scripts/graph-verdict.mjs` — a pure, fail-closed comparator (`compareObservation`) with a CLI entry point that merges role-tagged raw files (via `mergeRawObservations`, imported from `graph-measure.mjs`) and writes `corpora/graph-render-observations.json`.
- `web/scripts/graph-measure.mjs` — the measurement wrapper. `runSession()` owns the whole browser session inside one `try`/`finally`; the `finally` ALWAYS writes the raw observation before any cleanup (`ops.close`) is awaited. All six `OPS_KEYS` operations (`launch`, `newPage`, `navigate`, `seamReady`, `sample`, `close`) are individually raced against `withDeadline()` using budgets read from the locked `measurementProtocol` block — no gesture or deadline default of its own anywhere in the file (grep-verified, see below).
- 40 new tests (18 `graph-verdict.test.ts` + 22 `graph-measure.test.ts`), all TDD RED→GREEN, all green; `task web:test` 324/324; `cd web && pnpm check` 0 errors.
- `@playwright/test@1.62.1` installed exact-pinned, chromium binary verified present; `web/build` rebuilt and `task web:drift`/`web:audit`/`web:lockfile`/`web:deps:strict` all green.
- **GRF-01 measured, twice, against the real shipped binary:** the binding corpus (`google/guava`) FAILED to become interactive within the locked 60-second seam deadline in both the exploratory run and the final canonical run (identical failure, reproducible). The additional corpus (this repository's own index) measured cleanly end-to-end — real numbers, 461 sampled frames, confirming the whole pipeline (including the real-mouse pan/zoom gesture) works correctly when the renderer actually completes.
- Overall verdict: **FAIL**, committed to `corpora/graph-render-observations.json`. Per D-10/the threshold's `onFailure` field, 05-05, 05-06 and 05-07 are blocked until Task 3 answers.

## Task Commits

1. **Task 1 RED: failing tests for graph-verdict and graph-measure** — `e5a66ff5` (test)
   **Task 1 GREEN: fail-closed verdict comparator + always-records measurement wrapper** — `b71d387625f7b8a8dd8d23af19779daa4cb1f456` (feat)
2. **Task 2: record GRF-01's measurement** — `1cb8c71e342a51d89db7d08011907a6db443976a` (docs)
3. **Task 3: OPEN** — checkpoint:decision, gate=blocking-human, awaiting the answer this SUMMARY surfaces.

## RED Output (Task 1, verbatim)

```
FAIL  tests/graph-measure.test.ts [ tests/graph-measure.test.ts ]
Error: Failed to resolve import "../scripts/graph-measure.mjs" from "tests/graph-measure.test.ts". Does the file exist?
  Plugin: vite:import-analysis

FAIL  tests/graph-verdict.test.ts [ tests/graph-verdict.test.ts ]
Error: Failed to resolve import "../scripts/graph-verdict.mjs" from "tests/graph-verdict.test.ts". Does the file exist?
  Plugin: vite:import-analysis

 Test Files  2 failed | 30 passed (32)
      Tests  284 passed (284)
```

Confirms both new script modules did not exist yet, as required before any implementation was written.

## GREEN Output (Task 1)

```
 Test Files  2 passed (2)
      Tests  40 passed (40)
```

`task web:test`: `web:test: observed numTotalTests=324 numPassedTests=324 (vitest exit 0)` → `web:test: PASS — 324 of 324 tests passed`.
`cd web && pnpm check`: `1146 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`.

## Deadline Discrimination Experiments (Task 1 step b3, verbatim)

All three run against scratch copies of `web/scripts/graph-measure.mjs` written outside the repository (never staged, never committed), then the real script's tests re-confirmed green afterward.

**Experiment 1 — the mechanism stubbed to a pass-through (`withDeadline` returns its promise unchanged).** ALL SIX never-settling sweep cases failed, each hitting its own 2000ms per-test timeout:

```
× a never-settling "launch" ... 2026ms   → Test timed out in 2000ms.
× a never-settling "newPage" ... 2002ms  → Test timed out in 2000ms.
× a never-settling "navigate" ... 2001ms → Test timed out in 2000ms.
× a never-settling "seamReady" ... 2002ms → Test timed out in 2000ms.
× a never-settling "sample" ... 2001ms   → Test timed out in 2000ms.
× a never-settling "close" ... 2000ms    → Test timed out in 2000ms.
 Tests  8 failed | 14 skipped (22)
```

**Experiment 2 — the deadline removed from `sample` alone.** EXACTLY ONE case failed at its timeout; the other five stayed green:

```
✓ a never-settling "launch" ...    50ms
✓ a never-settling "newPage" ...   27ms
✓ a never-settling "navigate" ...  26ms
✓ a never-settling "seamReady" ... 27ms
× a never-settling "sample" ...    2003ms  → Test timed out in 2000ms.
✓ a never-settling "close" ...     27ms
 Tests  2 failed | 6 passed | 14 skipped (22)
```

(The second failure is the downstream aggregate set-equality assertion, which correctly fails as a consequence of `sample`'s own failure — not a seventh independent miss.)

**Experiment 3 — cleanup awaited before the write.** The wedged-`close` ordering case failed:

```
× the never-settling close stub observes the file already on disk at the moment it is entered   57ms
AssertionError: expected false to be true
 - Expected: true
 + Received: false
```

`existedWhenCloseWasEntered` recorded `false` — the file had NOT been written by the time the (now-relocated) close call was entered, which is precisely the T-05-21 outcome this ordering criterion exists to catch.

**Real script confirmed GREEN before and after all three experiments** (40/40 both times).

## Task 2: Live Measurement (verbatim findings)

### Setup

- `task build:release` (see Decisions Made — the plan's literal `task build` retains no binary).
- `codegraph ui --no-open --path <indexed guava checkout>` → `http://127.0.0.1:58925` (binding server).
- `codegraph ui --no-open --path <this repo>` → `http://127.0.0.1:61706` (additional-corpus server, this repository's own live index at commit `b71d387625f7b8a8dd8d23af19779daa4cb1f456`).

### Binding corpus: `google/guava` @ `94f39958baf7ad51ddf9c70e406ed6b188194daa`

```
cd web && node scripts/graph-measure.mjs --role binding --repo google/guava \
  --sha 94f39958baf7ad51ddf9c70e406ed6b188194daa --out /tmp/graph-raw-binding.json \
  --url http://127.0.0.1:58925/graph \
  --response-bytes 3713528 \
  --response-bytes-method "cited from 05-02-SUMMARY.md TestFileGraphResponseSizeGuava — real server+client round trip, proto.Marshal on *uiv1.FileGraphResponse" \
  --collapsed-nodes 134 --collapsed-edges 819
```

Result (real command, ~60.4s wall clock, reproduced identically on a first exploratory run):

```
graph-measure: wrote /tmp/graph-raw-binding.json (role=binding, sessionError=seamReady exceeded its 60000ms deadline)
```

**This is a genuine measurement failure, not a bar that was measured and missed.** `window.__codegraphFileGraphMetrics` never became truthy within the locked `seamReadyTimeoutMs` (60000ms — twelve times the 5000ms bar, deliberately loose so a slow-but-finishing run would not be mistaken for a hang). All three browser-derived metrics (`timeToInteractiveMs`, `panZoomFrameTimeMs`, `panZoomFrameTimeP95Ms`) recorded `measurement-failed` with that reason.

**Independently reproduced with no deadline at all**, to characterize how far past the deadline this actually goes (a diagnostic probe, not part of the sanctioned measurement path — no artifact from this run was used for the committed observation): a raw Playwright session against the same URL, polling `window.__codegraphFileGraphMetrics` every ~3s with no timeout, was still returning `null` after **607 seconds (~10.1 minutes)**, at which point the probe was terminated. During that window: zero console errors, zero page errors beyond the pre-existing, unrelated CSP logo warning (Phase 2, `WINDOWS.md`-tracked, not touched by this phase); `ps aux` showed the chromium renderer process had accumulated **10:57 of CPU time** — evidence the page was actively computing (or at minimum consuming CPU), not idle-deadlocked or crashed. This is consistent with `cytoscape-elk`'s synchronous, main-thread `elkjs` layout genuinely taking an extreme amount of time at this scale (3,233 nodes / 21,554 edges), rather than a bug in the measurement harness.

**`fileGraphResponseBytes` PASSED**: 3,713,528 bytes (cited from 05-02's own measured figure, not re-derived) against the 16,777,216-byte ceiling — 22.1% of it, unaffected by the layout failure since it never depended on the browser session.

**Collapsed directory-level counts, derived from the same wire response** (a direct `fetch` to the `FileGraph` Connect endpoint, JSON protocol, confirmed `3233` nodes / `21554` edges matching the threshold's own corpus note exactly): **134 nodes / 819 edges** — both an EXACT match to the threshold's `recordedNonBinding.collapsedView.approxNodes`/`approxEdges` (134/819), which independently confirms both the wire response and the collapsing definition (immediate parent directory of each file; directed, distinct cross-directory pairs, same-directory pairs excluded) are correct. This is a derived count, not a rendered measurement — its `method` field says so.

Browser identity recorded: `chromium 151.0.7922.34` (the driver's own report, read off the launched browser — not typed in).

### Additional corpus: this repository's own index @ `b71d387625f7b8a8dd8d23af19779daa4cb1f456`

```
cd web && node scripts/graph-measure.mjs --role additional --repo seanb4t/codegraph-go \
  --sha b71d387625f7b8a8dd8d23af19779daa4cb1f456 --out /tmp/graph-raw-self.json \
  --url http://127.0.0.1:61706/graph
```

Result: `graph-measure: wrote /tmp/graph-raw-self.json (role=additional, sessionError=null)` — a fully successful, real end-to-end measurement:

| Metric | Value |
|---|---|
| `timeToInteractiveMs` (median of 3 reloads) | 2604.8ms (raw: 2700, 2602.3, 2604.8) |
| `layoutDurationMs` (non-binding, narrower) | 2135.7ms |
| `panZoomFrameTimeMs` (median) | 8.3ms |
| `panZoomFrameTimeP95Ms` | 8.9ms |
| `frameSampleCount` | 461 |
| `nodeCount` / `edgeCount` (laid out) | 714 / 1074 |
| `sessionError` | null |

This confirms the whole live measurement mechanism — three cold reloads, the real-mouse pan/zoom gesture (`page.mouse` trusted input, never `dispatchEvent`, per the orchestrator's own live-verification finding), and the `requestAnimationFrame` sampler — works correctly end-to-end when the renderer actually completes. This repository's own scale is evidence, never a bar that was cleared (it is `additionalCorpora`, never `bindingObservation`).

### Merge and verdict

```
cd web && node scripts/graph-verdict.mjs --binding /tmp/graph-raw-binding.json --additional /tmp/graph-raw-self.json
```

```
graph-verdict: wrote .../corpora/graph-render-observations.json — verdict: FAIL
```

### Task 2 Verify Output (verbatim)

```
metrics judged: 4 verdict: FAIL frameSampleCount: null failed: 3 additionalCorpora: 1
```

Exit 0 — **this is a well-formed FAIL, and the verify command's exit 0 IS the successful outcome** per the plan's own design (a FAIL is this gate working, not something to fix).

## Files Created/Modified

- `web/scripts/graph-verdict.mjs` — the fail-closed comparator and CLI entry point.
- `web/scripts/graph-measure.mjs` — the always-records measurement wrapper.
- `web/tests/graph-verdict.test.ts`, `web/tests/graph-measure.test.ts` — 40 tests.
- `web/package.json`, `web/pnpm-lock.yaml` — `@playwright/test@1.62.1` (exact-pinned) + `@types/node@^24`; lockfile package count 236 → 240 → 242.
- `web/build` — rebuilt; source-file count unchanged (107 — scripts/tests are not in the hashed set), output-file count unchanged (32), both digests changed (source: driver added to package.json; output: Vite's documented non-deterministic content-hash filenames — no `web/src` module imports the driver).
- `corpora/graph-render-observations.json` — GRF-01's committed measurement and computed verdict (FAIL).

## Decisions Made

- **`task build` vs `task build:release`** — see key-decisions above (Rule 3).
- **`@types/node` added** — see key-decisions above (Rule 3).
- **Collapsed-view definition reverse-engineered from the threshold's own recorded figures** — see key-decisions above.
- **Task 3 left open** — see key-decisions above; this is the deliberate design of a `gate=blocking-human` checkpoint, not an omission.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `task build` retains no binary; used `task build:release`**
- **Found during:** Task 2 step (a).
- **Issue:** The plan's literal action text says `GOTOOLCHAIN=go1.26.5 task build`, but that Taskfile target is documented as "compile check only, no binary retained" (`go build ./...`). Task 2 needs a real `./codegraph` binary to run `codegraph ui`.
- **Fix:** Ran `GOTOOLCHAIN=go1.26.5 task build:release` instead, producing `./codegraph` (63MB, version-stamped at commit `b71d387625f7b8a8dd8d23af19779daa4cb1f456`). Also ran the literal `task build` for completeness (compile check passes).
- **Verification:** `./codegraph --version` reports the correct commit; the binary served both UI sessions used for measurement.
- **Files modified:** none (build-only).

**2. [Rule 3 - Blocking] Installed `@types/node@^24`**
- **Found during:** Task 1, running `cd web && pnpm check` after implementing both scripts.
- **Issue:** `pnpm check` (a hard Task 1 acceptance criterion) failed with `Cannot find name 'node:fs'` / `Cannot find name 'process'`-class errors — this repository's `web/` had no prior `.mjs`/Node-target source file, so no Node ambient types were ever needed before this plan.
- **Fix:** `pnpm add -D "@types/node@^24"` (matching this repository's CI Node version, 24). Not exact-pinned like `@playwright/test` — this is a types-only, zero-runtime-footprint devDependency, and the plan's exact-pin requirement was scoped specifically to the browser driver for supply-chain/reproducibility reasons that do not apply here.
- **Verification:** `cd web && pnpm check` — 0 errors, 0 warnings (was 128 errors before this fix, mostly resolved by this package alone; the remainder were genuine JSDoc-typing gaps in the new files, fixed separately — see below).
- **Files modified:** `web/package.json`, `web/pnpm-lock.yaml`.

**3. [Rule 1 - Bug] JSDoc typing gaps in both new `.mjs` files and their tests**
- **Found during:** Task 1, iterating `pnpm check` after the `@types/node` fix (128 → 21 errors).
- **Issue:** `checkJs: true` + `strict: true` type-checks plain `.js`/`.mjs` files at the same strictness as `.ts`. The initial implementation had extensive implicit-`any` parameters, an unsound `let`-mutated-inside-a-closure narrowing pattern (`frameStats`/`ttiRaw` assigned inside the `doWork()` arrow function, which TypeScript cannot narrow at outer read sites), and several test fixtures that didn't carry the full `RawObservation` shape.
- **Fix:** Added comprehensive JSDoc `@typedef`/`@param`/`@returns` annotations throughout both scripts; restructured `runSession()` so `doWork()` RETURNS its collected result (assigned once, directly, in the outer function's own flow) rather than mutating outer-scope `let`s from inside its own closure — this is a real code-quality improvement (removes an unsound type-narrowing pattern that a genuine bug could hide behind), not merely a type-checker appeasement. Cast a small number of intentionally-partial test fixtures with documented `as unknown as RawObservation` / `as any` where the test's whole point is exercising a runtime guard against a shape TypeScript would otherwise catch at compile time.
- **Verification:** `cd web && pnpm check` — 0 errors, 0 warnings. All 40 tests still pass after the restructuring (re-ran full suite).
- **Files modified:** `web/scripts/graph-measure.mjs`, `web/scripts/graph-verdict.mjs`, `web/tests/graph-verdict.test.ts`, `web/tests/graph-measure.test.ts`.

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug). **Impact on plan:** none weaken any threshold, bound, or acceptance criterion — all three were necessary to satisfy this plan's OWN hard acceptance criteria (`pnpm check` exits 0, a real binary exists to run the measurement against) and the closure-narrowing restructure is a genuine correctness improvement.

## Issues Encountered

- **The renderer's own behavior at guava scale, not this plan's mechanism.** See "Task 2: Live Measurement" above — this is the finding Task 3 exists to act on, not a defect in this plan's own deliverables. Recorded here per the scope-boundary rule as a finding, not "fixed."
- No other issues. All of Task 1's fail-closed and always-records mechanisms performed exactly as designed against a REAL failure — this is direct, live confirmation that the mechanism this plan built works, not merely that its unit tests pass.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **05-05, 05-06 and 05-07 are BLOCKED** pending Task 3's answer (D-10 names all three).
- The recorded verdict is FAIL. Per `corpora/graph-render-threshold.json`'s own `onFailure` field (locked in 05-01, before any measurement existed), exactly two remedies are available:
  - **(a)** Make the collapsed directory view the default first paint with progressive expansion. Keeps `cytoscape`/`cytoscape-elk`. The already-measured collapsed-view counts (134 nodes / 819 edges — ~24x fewer nodes, ~26x fewer edges than the failing expanded view) size this remedy rather than leaving it speculative, and this repository's own successful measurement (714/1074 nodes/edges, 2.6s TTI) is a rough scale-comparable data point in the same direction.
  - **(b)** Reconsider the renderer and layout stack. Re-opens D-02, D-07 and D-04 together.
  - Widening, lowering or re-scoping the threshold is explicitly NOT an available remedy.
- No blockers on this plan's own deliverables — Tasks 1 and 2 are complete, verified, and committed.

## Self-Check: PASSED

- `test -f web/scripts/graph-verdict.mjs` → FOUND
- `test -f web/scripts/graph-measure.mjs` → FOUND
- `test -f web/tests/graph-verdict.test.ts` → FOUND
- `test -f web/tests/graph-measure.test.ts` → FOUND
- `test -f corpora/graph-render-observations.json` → FOUND
- `git log --oneline --all | grep -q e5a66ff5` → FOUND
- `git log --oneline --all | grep -q b71d387` → FOUND
- `git log --oneline --all | grep -q 1cb8c71e` → FOUND
- `cd web && pnpm exec vitest run tests/graph-verdict.test.ts tests/graph-measure.test.ts` → PASS (40/40)
- `GOTOOLCHAIN=go1.26.5 task web:test` → PASS (324/324)
- `cd web && pnpm check` → PASS (0 errors, 0 warnings)
- `GOTOOLCHAIN=go1.26.5 task web:drift / web:audit / web:lockfile / web:deps:strict` → all PASS
- `git diff --name-only corpora/graph-render-threshold.json` → empty
- `git merge-base` check: threshold lock commit is an ancestor of the observation commit → CONFIRMED

---
*Phase: 05-file-package-graph-view*
*Completed (Tasks 1-2; Task 3 open): 2026-08-30*
