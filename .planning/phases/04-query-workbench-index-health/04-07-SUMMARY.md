---
phase: 04-query-workbench-index-health
plan: 07
subsystem: supply-chain
tags: [shadcn-svelte, drift-guard, taskfile, github-actions, render-cost, vitest, web-build]

requires:
  - phase: 04-query-workbench-index-health
    provides: "04-01 (table vendored @1.5.1), 04-04 (tabs vendored @1.5.1), 04-05, 04-06 — all six frontend plans whose source changes invalidated the committed web/build/"
provides:
  - "task web:components:drift — pinned-CLI regeneration into scratch, disk-derived subject set, byte-compare, RED-proven twice"
  - ".github/workflows/components-drift.yml — schedule + workflow_dispatch invocation of the drift target, out of required CI"
  - "task web:render-cost — opt-in 1000-row render/sort measurement, never asserted inside task web:test"
  - "committed web/build/ rebuilt and matching source again — task web:drift GREEN"
  - "three folded todos closed with resolution records"
affects: ["05", "verify-work"]

actuals:
  tokens: 13200
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "web:components:drift mirrors proto:drift's regenerate-into-scratch-and-byte-compare shape (never web:drift's manifest-hash shape)"
    - "opt-in wall-clock assertions behind an env var + dedicated Taskfile target, kept out of any PR-required job"

key-files:
  created:
    - .github/workflows/components-drift.yml
    - web/tests/data-table-render-cost.test.ts
    - web/tests/support/data-table-location-host.svelte
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
    - web/build/

key-decisions:
  - "Task 1 live probe: all 8 vendored component families (including the six vendored via @latest in 03-06) reproduce byte-identically at shadcn-svelte@1.5.1 — no per-family exception or normalization needed."
  - "web:components:drift's comparison is scoped strictly to web/src/lib/components/ui/ — the CLI's own dependency-install step mutates package.json (a devDependency caret range) as a side effect, which the guard must never compare against."
  - "Render-cost measurement came back OVER the plan's fixed threshold (400ms/200ms) on both metrics, reproduced across 4 runs — @tanstack/svelte-virtual was NOT installed (this plan does not pre-authorize that package); a follow-up todo and a CHECKPOINT REACHED request the required blocking-human package-legitimacy decision."

requirements-completed: [WRK-04]

coverage:
  - id: D1
    description: "task web:components:drift regenerates every vendored component family from disk-derived subject set into scratch, byte-compares, reports counts before comparing, and has been watched fail two independent ways"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "task web:components:drift (manual run, recorded in SUMMARY)"
        status: pass
    human_judgment: false
  - id: D2
    description: ".github/workflows/components-drift.yml invokes the guard on schedule + workflow_dispatch only, with a bootstrap chain proven present and correctly ordered"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk'"
        status: pass
    human_judgment: false
  - id: D3
    description: "1000-row render/sort-toggle cost measured with median-of-five and a row-count positive control; wall-clock threshold opt-in only, never in task web:test"
    requirement: "WRK-04"
    verification:
      - kind: unit
        ref: "web/tests/data-table-render-cost.test.ts"
        status: pass
    human_judgment: false
  - id: D4
    description: "committed web/build/ rebuilt; task web:drift GREEN on both source and output halves"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "task web:drift"
        status: pass
    human_judgment: false
  - id: D5
    description: "render-cost measurement came back OVER the plan's fixed threshold on both metrics (reproduced across 4 runs); virtualization was correctly NOT installed without authorization — a blocking-human package-legitimacy decision for @tanstack/svelte-virtual@3.13.36 is requested via CHECKPOINT REACHED and a follow-up todo"
    verification: []
    human_judgment: true
    rationale: "Installing a new [SUS]/too-new npm dependency requires an explicit maintainer legitimacy decision this plan does not pre-authorize (D-08, T-04-31) — automation cannot self-approve this."

duration: 40min
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 07: Drift Guard, Render Cost, Bundle Rebuild Summary

**Closed the phase's three remaining obligations: a disk-derived byte-compare guard over vendored shadcn-svelte source (proven RED two independent ways and wired into a schedule-only workflow), a measurement-first render-cost record for the 1000-row workbench table (which came back OVER threshold, correctly halting before an unauthorized package install), and a rebuilt committed `web/build/` that closes the RED window opened at 04-01.**

## Performance

- **Duration:** ~40min
- **Started:** 2026-08-30T01:35:00Z
- **Tasks:** 3 completed
- **Files modified:** 40 (987 insertions, 47 deletions)

## Task Commits

Each task was committed atomically:

1. **Task 1: Prove the premise — determinism probe** — `0bf304c1` (docs)
2. **Task 2: `task web:components:drift` guard + workflow** — `b518b79e` (feat)
3. **Task 3: render-cost measurement + bundle rebuild** — `b96cc8d1` (feat)

**Plan metadata:** commit pending (this session's final `docs(04-07): complete` commit, made after this SUMMARY).

## Task 1: Determinism probe — recorded findings

**Read first:** `.planning/phases/04-query-workbench-index-health/04-RESEARCH.md` Open Question 2, `web/components.json`, 04-01-SUMMARY.md, 04-04-SUMMARY.md.

**Scratch-tree method used:** `rsync -a --exclude node_modules --exclude .svelte-kit --exclude build web/ "$SCRATCH/web/"` (an identical copy of `web/components.json` travels with the rest of the tree — no hand-typed copy), then `pnpm install --frozen-lockfile` inside the scratch tree (fast — 3.8s, mostly served from the local pnpm content-addressable store, no meaningful network cost) rather than a symlinked `node_modules`: pnpm's `[ERR_PNPM_UNSAFE_MODULES_DIR]` safety check refuses to operate through a `node_modules` symlink that resolves outside the scratch project root, so a real (store-hydrated) install is the only viable scratch shape — this is itself a finding Task 2's target design must carry (a symlink shortcut does not work; a real `pnpm install --frozen-lockfile` inside the scratch tree does, and is cheap because of the content-addressable store).

**Disk-derived family enumeration** (the exact command Task 2's target will also run):

```
git ls-files -- 'web/src/lib/components/ui/' | awk -F/ '{print $6}' | sort -u
```

Output: `button command dialog input input-group table tabs textarea` — **8 families**, matching the plan's own "not just this phase's two" framing.

**Trailing-slash counter-check** (T-04-29 evidence):

```
git ls-files -- 'web/src/lib/components/ui/*/' | wc -l   →  0
git ls-files -- 'web/src/lib/components/ui/' | wc -l      →  50
```

Confirmed live in this repository: the trailing-slash pathspec returns **zero** files; the no-slash form returns **50**. Per-family counts: button 2, command 12, dialog 11, input 2, input-group 7, table 9, tabs 5, textarea 2 (sums to 50).

### Question 1 — Is the output byte-identical to what is committed?

**Yes, for all 8 families, with zero differences.** Ran `pnpm dlx shadcn-svelte@1.5.1 add button command dialog input input-group table tabs textarea -y -o` inside the scratch tree, then `diff -rq` each committed family directory against its scratch-regenerated counterpart:

```
=== button ===
=== command ===
=== dialog ===
=== input ===
=== input-group ===
=== table ===
=== tabs ===
=== textarea ===
```

Every block is empty — `diff -rq` reports nothing when trees are identical. No import-path rewriting difference, no formatting difference, no trailing-newline difference, no `index.ts` ordering difference.

### Question 2 — Is it deterministic across two consecutive runs into two fresh scratch trees?

**Yes.** Built a second, fully independent scratch tree (`probe2`, its own `rsync`, its own `pnpm install --frozen-lockfile`, its own `pnpm dlx shadcn-svelte@1.5.1 add ... -y -o`) and diffed `probe1`'s regenerated output against `probe2`'s, family by family — all 8 `diff -rq` blocks empty. Two independently-installed, independently-regenerated scratch trees produce byte-identical component source.

### Question 3 — Does `-o` behave the same against a target directory that already contains DIFFERENT content as it does against an empty one?

**Yes.** Appended a deliberate corruption line to `probe1/web/src/lib/components/ui/button/button.svelte`, confirmed the diff showed the injected line, then re-ran `pnpm dlx shadcn-svelte@1.5.1 add button -y -o` over the now-dirty target. Result: the file was overwritten back to byte-identical with the committed canonical file. `-o` unconditionally overwrites regardless of the target's prior content — no divergence between the empty-scratch case the guard actually uses and a dirty-target case.

### Question 4 — Does the CLI mutate anything outside the component directory in the scratch tree?

**Yes — `package.json`.** `diff` of the scratch tree's `package.json` against the repo's showed exactly one line changed: `"@lucide/svelte": "^1.34.0"` (committed) became `"@lucide/svelte": "^1.35.0"` (scratch, post-`add`). The CLI's own "Installing dependencies with pnpm..." step re-resolves at least one existing devDependency's semver range as a side effect of vendoring `input-group` (which references a lucide icon). `components.json` and `web/src/app.css` were unchanged; a full `diff -rq` of `web/src` (excluding `node_modules`, `.svelte-kit`, and the 8 known component directories) showed no other file touched.

**Consequence for Task 2's design:** the guard's comparison MUST be scoped strictly to `web/src/lib/components/ui/` — never to `package.json` or `pnpm-lock.yaml` — both because that is the guard's actual mandate (T-04-27 is about component *source*, which no lockfile-based gate can see) and because the CLI's own side effect would otherwise make the guard fail every run for a reason that has nothing to do with drift.

### Question 5 — Per-family reproduction table

| Family | Vendored by | Original vendoring invocation | Reproduces at 1.5.1? |
|---|---|---|---|
| button | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| command | 03-06 | `pnpm dlx shadcn-svelte@latest add command popover` | ✅ byte-identical |
| dialog | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| input | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| input-group | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| table | 04-01 | `pnpm dlx shadcn-svelte@1.5.1 add table -y -o` | ✅ byte-identical |
| tabs | 04-04 | `pnpm dlx shadcn-svelte@1.5.1 add tabs -y -o` | ✅ byte-identical |
| textarea | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |

**Every one of the 8 disk-derived families reproduces byte-identically at `shadcn-svelte@1.5.1`** — including the 6 families 03-06 vendored via `@latest` at an earlier date, meaning `@latest` at that time already resolved to `1.5.1` (or the registry has not changed those component templates since). No family diverges.

### Branch taken

**Byte-identical and deterministic → Task 2 wires the byte-compare exactly as designed, with NO per-family exception and NO normalization.** The "likely HALT" outcome the plan flagged as probable did not occur; all evidence points at the straightforward design. `web/components.json` is copied verbatim as part of the whole-tree scratch copy (not hand-retyped), satisfying the read_first requirement that a scratch tree carry an identical copy of it.

No repository file outside `.planning/` was modified by this task; all scratch trees live under the session scratchpad directory (`/private/tmp/claude-...`) and were never enumerated with `git ls-files`.

## Task 2: `task web:components:drift` — enumerate from disk, report the count, byte-compare, and prove it goes RED

Added `web:components:drift` to `Taskfile.yml` (placed after `web:drift`, before `vuln:`), following `proto:drift`'s regenerate-into-scratch-and-byte-compare shape, corrected against Task 1's live evidence: the scratch tree is a full `rsync -a --exclude node_modules --exclude .svelte-kit --exclude build` copy of `web/` (so `components.json` and every alias-resolution input travel verbatim, never hand-retyped) plus a real `pnpm install --frozen-lockfile` inside the scratch copy — a symlinked `node_modules` is rejected outright by pnpm's own `ERR_PNPM_UNSAFE_MODULES_DIR` safety check because it resolves outside the scratch project root (Task 1's finding). The real install is cheap: it hydrates from the local content-addressable pnpm store, no meaningful network cost.

**(a)/(b) Disk-derived enumeration, reported before comparing:**

```
files=$(git ls-files -- 'web/src/lib/components/ui/')                          # NO trailing slash
components=$(printf '%s\n' "${files}" | awk -F/ '{print $6}' | sort -u)
echo "web:components:drift: compared ${nfiles} vendored component files across ${ncomponents} components"
```

Observed on the green run: `web:components:drift: compared 50 vendored component files across 8 components`, then `web:components:drift: PASS — all 50 vendored component files across 8 components byte-identical to shadcn-svelte@1.5.1's regeneration (scratch tree only — source tree untouched)`. Floors: `nfiles -lt 8`, `ncomponents -lt 2` — small structural minima, never today's observed count, matching `web:test`'s FLOOR / `web:drift`'s `SRC_FLOOR` convention.

**(c) Regenerate into scratch, never in place:** `pnpm dlx shadcn-svelte@1.5.1 add <disk-derived components> -y -o` — never `@latest`.

**(d) Byte-compare, scoped to `web/src/lib/components/ui/` only** — `package.json` is deliberately never compared (Task 1's finding: the CLI's own dependency-install step bumps an unrelated devDependency range as a side effect).

**(f) Both RED proofs, run live and recorded here verbatim:**

1. **Planted one-byte mutation.** Appended `// planted drift byte` to the committed `web/src/lib/components/ui/button/button.svelte`. Re-ran the target: exited non-zero (task's wrapped exit 201, underlying `exit 1`) and printed:
   ```
   ::error::web:components:drift: web/src/lib/components/ui/button/button.svelte differs from shadcn-svelte@1.5.1's regeneration
   ```
   Reverted the file (`cp` from a pristine copy) and re-ran: `web:components:drift: PASS — all 50 vendored component files across 8 components byte-identical ...` — clean `git diff --stat` on the reverted file confirmed no residue.

2. **Trailing-slash pathspec.** Temporarily replaced the pathspec in `Taskfile.yml` with the trailing-slash form, re-ran the target:
   ```
   web:components:drift: compared 0 vendored component files across 0 components
   ::error::web:components:drift: enumerated only 0 committed component files via `git ls-files -- 'web/src/lib/components/ui/*/'` — check the pathspec has NO trailing slash appended (appending one to this exact pathspec is verified, in 04-07-SUMMARY.md, to return ZERO files in this repository); a narrowed enumeration must fail loud, never read as a clean pass
   ```
   The population floor rejected the empty enumeration rather than passing silently. Restored the pristine `Taskfile.yml` from a pre-edit copy and re-ran: `PASS` again, with `git diff --stat Taskfile.yml` showing only the intended target addition (149 insertions), no residue from the trailing-slash experiment.

**(g) Workflow wiring:** `.github/workflows/components-drift.yml` — `schedule` (weekly, Monday 08:00 UTC, deliberately offset from `bench.yml`'s Monday 06:00 UTC) + `workflow_dispatch` only, no per-PR/per-push trigger. Bootstrap chain: `actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10` (same SHA as `ci.yml`) → `uses: ./.github/actions/install-task` → `actions/setup-node@820762786026740c76f36085b0efc47a31fe5020` (same SHA as `ci.yml`) with `node-version: "24"` → `run: task web:components:drift`. Ordering gate (extracting each line number, `test -n` guarded, asserting the full chain) ran and printed:
```
ci pins: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 actions/setup-node@820762786026740c76f36085b0efc47a31fe5020; drift lines: checkout=74 install-task=77 setup-node=86 node-version=88 run=91
GATE PASS
```
`rg -c 'run:' .github/workflows/components-drift.yml` → `1`; `rg -c 'uses:' ...` → `4`. `task lint:actions` (actionlint, `GOTOOLCHAIN=go1.26.5`) accepted the new workflow with no output.

**Note on the plan's own negative-grep discipline:** the plan's Task 2 action text (and this SUMMARY) discusses the trailing-slash pathspec and other forbidden literals in prose; the guarded files themselves (`Taskfile.yml`, `components-drift.yml`) were edited to describe these reasons WITHOUT reproducing the literal substrings the acceptance criteria assert are absent (e.g. `components/ui/*/`, `pull_request`, a second `run:`, an early match for `task web:components:drift` or `node-version: "24"` inside prose) — an initial draft of both files inadvertently included several of these literals in explanatory comments, which was caught by re-running the acceptance-criteria greps themselves and corrected before commit.

**Registration:** `internal/upgrade/taskfile_shape_test.go` — `inScopeWorkflowFiles` gained `"components-drift.yml"`; `inScopeJobs` gained `{Workflow: "components-drift.yml", JobID: "components-drift"}`. `requiredCheckNames` and `runBodyExceptions` are UNCHANGED (`git diff` confirmed). `GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk' -v` → 3/3 `--- PASS`, exit 0. Full `go test ./internal/upgrade/...` also green.

Todo `.planning/todos/pending/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md` moved to `.planning/todos/completed/` with a resolution record naming the target, the pinned CLI version, the observed counts, the workflow, and both RED-proofs.

`git diff Taskfile.yml` confirms no change to `web:drift`'s hashing pipeline, `SRC_FLOOR`, or `proto:drift`.

## Task 3: Measure the 1000-row render cost, rebuild the committed bundle, and close the last todo

### Measurement

Wrote `web/tests/data-table-render-cost.test.ts`, mounting the Workbench's shared `DataTable` shell through a thin, concretely `Location`-typed test host (`web/tests/support/data-table-location-host.svelte` — see its own header comment: `@testing-library/svelte`'s `render()` cannot infer `DataTable`'s generic `TRow` from a plain `.ts` call site, only from a real `.svelte` usage site; a fix, not a type-check workaround, since it mirrors exactly how `AnalysisPanel.svelte` really invokes `DataTable`).

Fixture: 1000 `Location` rows (`internal/query/validate.go:26` `MaxLimit` — the real worst case, not an invented one), inserted in a fixed, deterministic, non-monotonic order (`(i * 457) % n`, a bijection on 0..999 since 457 is prime and does not divide 1000). **First attempt used a plain descending-suffix order, which turned out to already equal one of TanStack's two full-sort states — the "genuinely reorders" assertion then failed because `afterOrder` legitimately equalled `beforeOrder` after an odd number of clicks landed back on the pre-existing descending order.** Confirmed the actual toggle cycle empirically (`unsorted → asc → desc → unsorted → asc → desc → ...`, verified live against a 3-row fixture) before fixing the fixture generator to a real shuffle — this is documented so a future reader does not re-invent the same false lead.

Three tests, matching the plan's `total >= 3` verification floor:
1. `renders exactly 1000 rows` — positive control (rule 84d1gfpywd): a truncated or empty render would read as fast and prove nothing.
2. `clicking the Name header genuinely reorders the rendered rows` — compares actual rendered row text content before/after one click, never a toggled indicator class.
3. `records median initial-render and sort-toggle durations over 5 repetitions` — 5 independent mounts (initial render) and 5 clicks on one mount (sort toggle), `performance.now()`-timed, MEDIAN reported and PRINTED unconditionally; the threshold comparison itself gated behind `process.env.RENDER_COST_ASSERT === '1'`.

`cd web && pnpm exec vitest run --reporter=json --outputFile=/tmp/rc.json tests/data-table-render-cost.test.ts` → `total=3 passed=3`, exit 0.

### The threshold, fixed by the plan before the number was seen, and OVER-THRESHOLD

Plan-fixed thresholds: median initial render ≤ 400ms, median sort toggle ≤ 200ms (jsdom — a coarse order-of-magnitude detector, not a performance budget). Added `task web:render-cost` to `Taskfile.yml` (`deps: [web:deps]`, sets `RENDER_COST_ASSERT=1`, runs only this one test file) — wired into **no** workflow.

**Observed, reproduced across 4 separate runs this session** (`pnpm exec vitest run` direct + three separate `task web:render-cost` invocations, before and after the bundle rebuild):

| Run | initial-render median | sort-toggle median |
|---|---|---|
| 1 | 421.39ms | 366.35ms |
| 2 | 409.26ms | 377.80ms |
| 3 | 416.68ms | 363.23ms |
| 4 | 411.16ms | 382.63ms |

Both metrics consistently exceed the fixed threshold (initial render: 400ms; sort toggle: 200ms, by a wide and stable margin) — a genuine, reproducible over-threshold result, not measurement noise. `task web:render-cost` exits non-zero (task-wrapped exit 201) with:
```
AssertionError: median initial render of 1000 rows in jsdom (coarse detector, not a budget): expected 411.1572500000002 to be less than or equal to 400
```

**Branch taken: OVER THRESHOLD → HALT, per D-08 and this plan's own prohibitions.** `@tanstack/svelte-virtual@3.13.36` was **NOT** installed. `web/package.json` has zero `svelte-virtual` entries (`rg -c "svelte-virtual" web/package.json` → 0). No client-side row cap was added either (D-08 names that rejected: never re-cap what the server already bounded). **This session returns a CHECKPOINT requesting a blocking-human package-legitimacy decision for `@tanstack/svelte-virtual@3.13.36`** before any virtualization work proceeds — see the CHECKPOINT REACHED section of this session's final response.

The deterministic, unconditional assertions (tests 1 and 2 above) are unaffected by the threshold outcome and pass on every `task web:test` run — see below.

### Bundle rebuild — closing the RED window opened at 04-01

`task web:drift` BEFORE rebuild:
```
web:drift: hashed 103 source files
web:drift: manifested 27 output files
::error::web:drift: SOURCE-half mismatch — the committed build output is stale relative to its source (marker: 73 files / 2484d457...; recomputed: 103 files / a4d55bc2...).
web:drift: output half MATCH (27 files, ea51d6a3...)
```
Confirms the intentional RED window exactly as the plan described it: source-half stale (73 → 103 files, accumulated across every 04-01..04-06 frontend plan), output-half still matched (nobody hand-edited the committed bundle).

`task web:build` → `web:build: wrote web/build/.build-manifest (source-files=103, output-files=30)`.

`task web:drift` AFTER rebuild:
```
web:drift: hashed 103 source files
web:drift: manifested 30 output files
web:drift: source half MATCH (103 files, a4d55bc2eb625d0a731b4a1ca2cbdcf28e9690f6760d9a392a71bb7cdb8d92ce)
web:drift: output half MATCH (30 files, 83a6ba811545ba8651edddfda12171c05b62037761cc6255fd4b358dba50d276)
web:drift: PASS — hashed 103 source files, manifested 30 output files, committed web/build/ matches both digests
```
Output file count moved 27 → 30 (three additional immutable chunks from this phase's added routes/components — expected, not a discrepancy). `web/build/`'s content-hashed filenames changed (Vite's documented non-determinism, `web:drift`'s own desc: text cites vitejs/vite#15555/#13071 — this is why `web:drift` hashes committed trees in place rather than regenerate-and-compare). `task web:components:drift` re-run after the rebuild: still `PASS — all 50 vendored component files across 8 components byte-identical`.

### Phase-level gate report (hand-off to `/gsd-verify-work`)

| Gate | Result |
|---|---|
| `task web:test` | `PASS — 260 of 260 tests passed` (up from 257 at end of 04-06; +3 new render-cost tests) |
| `task web:drift` | `PASS — hashed 103 source files, manifested 30 output files` |
| `task web:components:drift` | `PASS — all 50 vendored component files across 8 components byte-identical` |
| `task web:render-cost` | Ran once (and 3 more times for reproducibility); **FAILS by design** — over-threshold result recorded above, checkpoint requested, no virtualization installed |
| `task proto:drift` | `PASS — compared 4 generated files ... all 4 generated files byte-identical` |
| `GOTOOLCHAIN=go1.26.5 task test:unit` | all packages `ok` |
| `cd web && pnpm check` | `1016 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` |
| `task web:lockfile` | `PASS — 231 packages declared, lockfileVersion '9.0', 231 of 231 resolutions integrity-bearing, zero non-registry/non-link sources` |
| `task web:deps:strict` | `PASS — strictDepBuilds is true and allowBuilds is committed with 0 entries (0 denials)` |
| `task web:audit` | `PASS — pnpm audit exited 0, output classified CLEAN — zero advisories` |

### Todo closures

Both remaining folded todos moved to `.planning/todos/completed/` with resolution records:
- `2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md` (closed in Task 2).
- `2026-08-28-client-side-render-cost-measurement-for-browse-views.md` — resolution records the D-08 redirection (this todo's own text named the Workbench as the natural denser-view target), the observed numbers, the OVER-THRESHOLD branch, and an explicit scope note that `SourcePane`/`NeighborsPanel`/`SearchPanel` (the todo's original three-part ask) were NOT separately re-measured — D-08 scoped this phase's obligation to the Workbench table specifically.

`ls .planning/todos/pending/ | wc -l` → 4 (confirmed: the four pre-existing unrelated todos; none of the three Phase-4 todos remain pending — the third, `2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md`, was closed in 04-02, verified already present under `.planning/todos/completed/`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - blocking issue] `render(DataTable, ...)` from a plain `.ts` file failed `pnpm check`'s type-check**
- **Found during:** Task 3, writing `data-table-render-cost.test.ts`.
- **Issue:** `@testing-library/svelte`'s `render()` cannot infer `DataTable.svelte`'s generic `TRow` parameter from a `.ts` call site (only from real `.svelte` JSX-like usage), so TypeScript fell back to `TRow = RowData` and rejected the `Location`-typed `columns`/`getRowId` props.
- **Fix:** Added `web/tests/support/data-table-location-host.svelte`, a thin wrapper fixing `TRow = Location`, mirroring exactly how `AnalysisPanel.svelte` really invokes `DataTable` in the app. The test renders the host, never `DataTable` directly.
- **Files modified:** `web/tests/support/data-table-location-host.svelte` (new), `web/tests/data-table-render-cost.test.ts`.
- **Verification:** `cd web && pnpm check` → `1016 FILES 0 ERRORS 0 WARNINGS`.
- **Commit:** (Task 3 commit, below).

**2. [Rule 1 - bug] Render-cost fixture's initial insertion order coincided with one of TanStack's own full-sort states**
- **Found during:** Task 3, first test run — `afterOrder` unexpectedly equalled `beforeOrder` after 5 sort-toggle clicks.
- **Issue:** The first fixture generator produced rows in already-descending name order; TanStack's 3-state toggle cycle (`unsorted → asc → desc → unsorted → ...`) landed back on `desc` after 5 clicks, which was indistinguishable from the (already-descending) original order.
- **Fix:** Replaced the fixture generator with a fixed, deterministic, non-monotonic permutation (`(i * 457) % n`) so the original insertion order is neither ascending nor descending.
- **Files modified:** `web/tests/data-table-render-cost.test.ts`.
- **Verification:** re-ran; `afterOrder` now genuinely differs from `beforeOrder` and matches the expected ascending-sorted name order.
- **Commit:** (Task 3 commit, below).

**3. [Rule 3 - blocking issue] Task 2's Taskfile/workflow prose initially reproduced literals the acceptance criteria assert are absent**
- **Found during:** Task 2, re-running the acceptance-criteria greps after the first draft.
- **Issue:** Explanatory comments in `Taskfile.yml` and `components-drift.yml` initially spelled out the trailing-slash pathspec, the word `pull_request`, a second `run:` occurrence, and early matches for `task web:components:drift` / `node-version: "24"` inside header prose — each of which an acceptance-criteria grep asserts is absent (or, for the ordering gate, must be the FIRST match).
- **Fix:** Rephrased the affected comments to describe the same reasoning without reproducing the literal substrings (e.g., "check the pathspec has NO trailing slash appended" instead of spelling the glob; "never a per-PR or per-push trigger" instead of the word `pull_request`).
- **Files modified:** `Taskfile.yml`, `.github/workflows/components-drift.yml`.
- **Verification:** every negative/positive grep and the full bootstrap-ordering gate re-run clean (see Task 2 section above).
- **Commit:** `b518b79e` (folded into Task 2's commit, since it landed before that commit).

### Halted (not a bug — a designed stop)

**Render-cost measurement came back OVER THRESHOLD (D-08).** See "The threshold, fixed by the plan before the number was seen, and OVER-THRESHOLD" above. No virtualization dependency was installed; a `CHECKPOINT REACHED` is returned requesting the blocking-human package-legitimacy decision for `@tanstack/svelte-virtual@3.13.36`, per this plan's own prohibitions and D-08.

## Known Stubs

None — no stub was introduced. The over-threshold render-cost result is a genuine, honestly-reported measurement outcome, not a placeholder or an unwired data path.

## Threat Flags

None — T-04-27 through T-04-31 are all addressed by this plan's own deliverables (see the plan's Threat Model table); no new, un-modeled surface was introduced.

## Follow-up Todo Filed

`.planning/todos/pending/2026-08-30-workbench-datatable-1000-row-render-cost-exceeds-threshold-virtualization-needs-legitimacy-checkpoint.md` — records the over-threshold measurement, both observed numbers, and the pending `blocking-human` package-legitimacy checkpoint for `@tanstack/svelte-virtual@3.13.36`. This is a NEW todo, not one of the three this plan was chartered to close.

## Self-Check: PASSED

All created files found on disk; all three task commit hashes found in `git log`:
- `.github/workflows/components-drift.yml` — FOUND
- `web/tests/data-table-render-cost.test.ts` — FOUND
- `web/tests/support/data-table-location-host.svelte` — FOUND
- `.planning/todos/completed/2026-08-28-shadcn-svelte-registry-version-pinning-with-source-match.md` — FOUND
- `.planning/todos/completed/2026-08-28-client-side-render-cost-measurement-for-browse-views.md` — FOUND
- `.planning/todos/pending/2026-08-30-workbench-datatable-1000-row-render-cost-exceeds-threshold-virtualization-needs-legitimacy-checkpoint.md` — FOUND
- Commits `0bf304c1`, `b518b79e`, `b96cc8d1` — all FOUND in `git log --oneline --all`

