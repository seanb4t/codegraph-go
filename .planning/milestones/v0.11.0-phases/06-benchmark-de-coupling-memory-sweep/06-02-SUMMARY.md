---
phase: 06-benchmark-de-coupling-memory-sweep
plan: 02
subsystem: ci
tags: [github-actions, ci-tooling, benchmark, shape-testing]

# Dependency graph
requires:
  - phase: 06-01
    provides: "tools/bench/runner -mode publish measuring only the Go binary over a two-entry realcorpus"
provides:
  - ".github/workflows/bench.yml's publish job — replaces headtohead, carries all four D-06 properties, structurally verified"
  - "internal/upgrade/bench_workflow_shape_test.go — TestBenchPublishJobShape (publish-job-scoped D-06 verifier) and TestBenchReblessJobShape (parsed-subtree rebless-untouched fixture)"
affects: [06-04]

# Actuals (#2632)
actuals:
  tokens: 7242
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Publish-job-scoped structural workflow verification: a Go test parses jobs.<id> as a subtree via yaml.Unmarshal and asserts every load-bearing property against that subtree — never by a file-wide grep another job could pre-satisfy"
    - "Parsed-subtree fixture with a run-body SHA-256 digest for an 'unchanged' claim: structure/step-names asserted by value, but bodies not central to the plan's subject are proven byte-identical by digest rather than transcribed in full"
    - "FIXT-07 post-GREEN discriminating perturbation: delete one job-scoped assertion's target line, confirm the RED names that job's own assertion (not a sibling job's identical line), revert byte-cleanly, re-confirm GREEN"

key-files:
  created:
    - internal/upgrade/bench_workflow_shape_test.go
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PREPLAN-SHA.txt
  modified:
    - .github/workflows/bench.yml
    - tools/bench/BASELINE.md
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "BASELINE.md:64's dangling head-to-head-capture citation was RE-AUTHORED, not dropped — the finding it reports (darwin/arm64 vs linux/amd64 CI platform spread) is about hardware/OS class, not a two-subject binary comparison, so it survives on its own terms once the deleted-file citation is removed."
  - "The publish job's name: was retitled from 'benchmark publish (PERF-01, non-blocking)' to 'benchmark publish (BENCH-01/BENCH-03, non-blocking)' since PERF-01 was already closed in v1.0 and BENCH-01 retires those numbers this milestone (STATE.md); BASELINE.md:241's citation of PERF-01 alongside the repointed job id was left as-is per the plan's narrowly-scoped instruction (job-name repoint only, not a PERF-01 sweep)."
  - "bench_workflow_shape_test.go declares its own bespoke YAML struct set (benchStep/benchJob/benchWorkflowDoc) rather than extending the package's existing workflowRunStep/fullWorkflowJob types — widening a shared parsing type for one file's bespoke needs (uses:/if:/runs-on: together) risks silently changing what every OTHER guard in this package sees."

requirements-completed: [BENCH-03]

coverage:
  - id: D06.1
    description: "TestBenchPublishJobShape asserts the publish job's runs-on and CODEGRAPH_BENCH_RUNNER env value are both exactly namespace-profile-linux-amd64-4x8"
    requirement: BENCH-03
    verification:
      - kind: unit
        ref: "internal/upgrade/bench_workflow_shape_test.go#TestBenchPublishJobShape"
        status: pass
    human_judgment: false
  - id: D06.4
    description: "Publish job's upload-artifact step (its OWN with: map, not rebless's) carries name/path/if-no-files-found, and exactly one run body references GITHUB_STEP_SUMMARY"
    requirement: BENCH-03
    verification:
      - kind: unit
        ref: "internal/upgrade/bench_workflow_shape_test.go#TestBenchPublishJobShape"
        status: pass
    human_judgment: false
  - id: D06.3
    description: "Concatenation of all publish-job run bodies is non-empty and contains no gate invocation, no regression-mode flag, no -rebless flag, no task bench call, no npm/npx/node"
    requirement: BENCH-03
    verification:
      - kind: unit
        ref: "internal/upgrade/bench_workflow_shape_test.go#TestBenchPublishJobShape"
        status: pass
    human_judgment: false
  - id: D06.2
    description: "Taskfile bench-target set equals exactly {bench:regression}; no task bench* call inside the publish job's run bodies"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "TASKFILE_BENCH_TARGETS=bench:regression:, TASKFILE_TARGETS_TOTAL=47"
        status: pass
    human_judgment: false
  - id: RebRebless
    description: "rebless job proven untouched by a parsed-subtree fixture (runs-on, env, if, ordered 6-step names, upload with: map, no permissions:) plus a SHA-256 digest of its three run bodies"
    requirement: BENCH-03
    verification:
      - kind: unit
        ref: "internal/upgrade/bench_workflow_shape_test.go#TestBenchReblessJobShape"
        status: pass
    human_judgment: false
  - id: PublishScoping
    description: "Publish-job scoping demonstrated (not just designed) by a post-GREEN perturbation: deleting only the publish job's if-no-files-found: error line reddens TestBenchPublishJobShape by name while rebless's identical line survives untouched"
    requirement: BENCH-03
    verification:
      - kind: other
        ref: "See 'Post-GREEN discriminating perturbation' below — verbatim RED naming the publish job's upload assertion, byte-clean revert confirmed"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-08-16
status: complete
---

# Phase 6 Plan 2: Bench.yml Publish Job + D-06 Shape Verifier Summary

**Replaced `.github/workflows/bench.yml`'s two-subject `headtohead` job with a
`publish` job that measures only the freshly-built Go binary, and added
`internal/upgrade/bench_workflow_shape_test.go` — a Go test that parses the
workflow YAML and asserts every D-06 property against the parsed
`jobs.publish` subtree, never a file-wide grep another job could
pre-satisfy.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-16
- **Tasks:** 2
- **Files modified:** 5 (2 created: `bench_workflow_shape_test.go`,
  `06-02-PREPLAN-SHA.txt`; 3 modified: `bench.yml`, `BASELINE.md`,
  `taskfile_shape_test.go`)

## Accomplishments

- Deleted the `headtohead` job's two-subject scaffolding from `bench.yml`:
  the `Set up Node (for the installed comparison binary)` step, the
  `Install the comparison binary @1.3.1` step (`npm install -g
  @colbymchenry/codegraph@1.3.1`), and the `-ts-binary` flag on the runner
  invocation are all gone. The `publish` job that replaces it measures only
  the Go binary via `go run ./tools/bench/runner -mode publish`.
- Carried forward all four D-06 properties verbatim into `publish`: exact
  `runs-on: namespace-profile-linux-amd64-4x8` and matching
  `CODEGRAPH_BENCH_RUNNER` env value; the inline (never Taskfile-wrapped)
  runner invocation, with a comment noting `publish` mode has no
  baseline-overwriting flag at all so the D-13/D-01 exception is inherited
  context here, not a live risk; the non-blocking publish-not-gate contract
  restated in the run step's own comment; and the `GITHUB_STEP_SUMMARY`
  block plus `actions/upload-artifact` step (renamed `headtohead-results` ->
  `publish-results`) with `if-no-files-found: error`.
- Moved the identifier family together: the job's YAML key
  (`headtohead:` -> `publish:`), `workflow_dispatch.inputs.job`'s `default:`
  and `options:` entry, and the job's own `if:` conditional — while keeping
  the `both` dispatch option and its `|| inputs.job == 'both'` clause intact
  (cycle-2 LOW 2), so `both` still reaches this job.
- Repointed the one dangling `headtohead` reference outside every edited
  region: `bench.yml:295`'s `cpu-diag-github` preamble comment
  (`# NOT a gate and NOT part of the rebless/headtohead flow`) now names
  `publish` — the only word changed; the diagnostic jobs themselves are
  byte-untouched (`DIAG_JOB_DIFF_LINES=0`, `DIAG_JOB_MENTIONS=34`).
- Added `internal/upgrade/bench_workflow_shape_test.go` with its own
  bespoke YAML struct set (`benchStep`/`benchJob`/`benchWorkflowDoc`) and
  two tests:
  - `TestBenchPublishJobShape` selects `jobs.publish` and asserts, against
    that subtree only: exact `runs-on`/env, all three `if:` dispatch terms,
    exactly one inline `-mode publish` runner invocation, non-empty run
    bodies containing none of `task bench`/`CheckRegression`/`-mode
    regression`/`-rebless`/`npm `/`npx `/`node `, exactly one
    `GITHUB_STEP_SUMMARY` reference, exactly one upload-artifact step whose
    OWN `with:` map carries `name: publish-results`/`path:
    publish-results.json`/`if-no-files-found: error`, no Node-setup action,
    the four-element ORDERED (not set) SHA-pinned action list, no job-level
    `permissions:`, and the workflow-level `permissions:` map exactly
    `{contents: read}`.
  - `TestBenchReblessJobShape` compares `jobs.rebless` against a literal
    fixture transcribed from the pre-edit file: `runs-on`, env, `if:`
    verbatim, the ordered six-step name sequence, the upload `with:` map,
    no `permissions:`, and a SHA-256 digest of the normalised concatenation
    of its three run bodies (cycle-2 MEDIUM 2) — proving `rebless` is
    untouched by content, not just by shape.
- Recorded the pre-plan SHA (`c49ef77251d8cc65866f697fdf2b41a6ed7020e7`) in
  `06-02-PREPLAN-SHA.txt` before any edit, so every tamper guard in this
  plan compares against it rather than HEAD.
- Swept `tools/bench/BASELINE.md`'s two remaining `headtohead` references
  (line 64's dangling capture-file citation, re-authored on its own terms
  since the finding is about platform spread, not binary comparison; line
  241's job-name reference, repointed to `publish`) and
  `taskfile_shape_test.go:102`'s `inScopeJobs` doc comment (comment-only —
  the fixture literals themselves are untouched).

## Task Commits

1. **Task 1: Author the publish-job structural verifier, then the publish
   job it describes** - `bd98d47` (feat)
2. **Task 2: Sweep the benchmark investigation record and the
   workflow-exception comment** - `e8cab5d` (docs)

_This plan's SUMMARY commit follows as the metadata commit (worktree mode —
STATE.md/ROADMAP.md excluded; orchestrator updates those centrally after
merge)._

## Files Created/Modified

- `.github/workflows/bench.yml` - `headtohead:` job deleted in full and
  replaced by `publish:`; dispatch `default:`/`options:` and the job's
  `if:` repointed; header/runner-profile comment paragraphs re-authored to
  describe absolute single-subject numbers; line 295's dangling
  `headtohead` word repointed to `publish`
- `internal/upgrade/bench_workflow_shape_test.go` - **new**:
  `TestBenchPublishJobShape`, `TestBenchReblessJobShape`, and their shared
  YAML decode/helper infrastructure
- `tools/bench/BASELINE.md` - lines 64 and 241 repointed; every other
  `comparison` use (5, all describing `CheckRegression`) and every measured
  figure left byte-unchanged
- `internal/upgrade/taskfile_shape_test.go` - `inScopeJobs` doc comment at
  line 102 repointed; fixture literals untouched
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PREPLAN-SHA.txt` -
  **new**: pre-plan HEAD SHA anchor (`c49ef77251d8cc65866f697fdf2b41a6ed7020e7`)

## RED Output (TestBenchPublishJobShape, before the publish job existed)

Captured by temporarily reverting `bench.yml` to the pre-plan-SHA commit
(`git checkout -- .github/workflows/bench.yml`, since HEAD at that point was
still `c49ef77`) and re-running the newly-authored test, then restoring the
edit from a backup copy:

```
=== RUN   TestBenchPublishJobShape
    bench_workflow_shape_test.go:219: bench.yml declares no job "publish"
--- FAIL: TestBenchPublishJobShape (0.00s)
=== RUN   TestBenchReblessJobShape
--- PASS: TestBenchReblessJobShape (0.00s)
FAIL
```

(`TestBenchReblessJobShape` passes even pre-edit, since `rebless` already
existed unchanged — only the publish-job anchor is absent at this point.)

## Post-GREEN Discriminating Perturbation (cycle-2 MEDIUM 3)

The initial RED above comes from `jobs.publish` being wholly absent, which
reddens every assertion at once from a single missing anchor — evidence the
test runs, not evidence any individual assertion is scoped to the publish
job rather than reading the file globally. Ran the one-observation FIXT-07
perturbation the plan specifies:

1. Confirmed `git diff --quiet -- .github/workflows/bench.yml` (clean tree,
   committed state).
2. Deleted **only** the publish job's `if-no-files-found: error` line
   (line 154 of the post-edit file), leaving `rebless`'s identical line
   (now at line 270) untouched. Confirmed a non-empty `git diff --stat`
   (`1 deletion(-)`).
3. Re-ran `TestBenchPublishJobShape` — verbatim failure:

```
=== RUN   TestBenchPublishJobShape
    bench_workflow_shape_test.go:304: publish job's upload-artifact step with.if-no-files-found = "", want "error"
--- FAIL: TestBenchPublishJobShape (0.00s)
```

   The failure names the publish job's own upload assertion, and it fires
   even though the identical string is still present elsewhere in the file
   (rebless's upload step) — proving the subtree anchoring is real, not a
   file-wide check in disguise.
4. `git checkout -- .github/workflows/bench.yml` reverted byte-cleanly:
   `git diff --stat -- .github/workflows/bench.yml` empty,
   `git status --porcelain .github/` empty.
5. Re-ran both tests — both PASS (see "Accomplishments" GREEN state above).

## `uses:` Pinning Counts (T-06-04)

- `USES_TOTAL=21`, `USES_LOCAL=2` (`./.github/actions/install-task`,
  referenced twice), `USES_PINNED=19` (`19 + 2 = 21`).
- Derived: pre-edit file had 22 `uses:` lines (20 pinned, 2 local); this
  edit deletes exactly one pinned action (the Node setup step), leaving 19.

## `TestBenchReblessJobShape` Fixture Source and Digest

Transcribed from the file at the recorded pre-plan SHA
(`c49ef77251d8cc65866f697fdf2b41a6ed7020e7`, before any edit in this plan) —
verified by computing the digest with the exact decode/normalise logic that
ships in the committed test, run once against the pre-edit file to derive
the literal, then re-run against the post-edit file to confirm it still
matches (`rebless` is byte-unchanged by this plan).

- **Normalisation:** trim trailing whitespace from every line of each of
  the three run bodies ("Record candidate baseline on this runner",
  "Publish candidate to job summary", "Verify the candidate survives the
  gate on this runner class"), drop trailing blank lines, join the three
  normalised bodies with `\n`, SHA-256 the result.
- **Digest:**
  `edbd1b7f634aef3f85c812d7b7c0ccecc303a85f7d37c698e118173f43a9a1ae`

## Verbatim Verification Totals

- `BENCH_YML_RETIRED_TERMS_TOTAL=0` (Task 1 census: `\bheadtohead\b`,
  `\bts-binary\b`, `\bcolbymchenry\b`, `setup-node`, `npm install`,
  `\bcomparator\b`, the bounded `comparison` pattern — re-measured pre-edit
  at **25** hits total, not the plan's stated 26; all 25 fell inside
  regions this task deletes or rewrites, including line 295, which the
  plan's own prose separately called out as the "26th, out-of-region" hit —
  the discrepancy is in the plan's arithmetic, not in scope: the same
  companion gate below (`DIAG_JOB_*`) still confirms line 295 was the only
  edit outside the deleted-job/header/dispatch regions.)
- `DIAG_JOB_DIFF_LINES=0 CPUDIAG_NOT_A_GATE_CLAUSE=1 CPUDIAG_NO_BASELINE_CLAUSE=1 DIAG_JOB_MENTIONS=34`
- `USES_TOTAL=21 USES_LOCAL=2 USES_PINNED=19` (`ALL_REMOTE_USES_SHA_PINNED=true`)
- `PERMISSION_DIFF_LINES=0 PERMISSION_LINES_PRESENT=2`
- `TASKFILE_BENCH_TARGETS=bench:regression:, TASKFILE_TARGETS_TOTAL=47`
- `BASELINE_AND_SHAPETEST_RETIRED_TERMS_TOTAL=0`
- `rg -c '\bcomparison\b' tools/bench/BASELINE.md` = `5` (unchanged — legitimate `CheckRegression` uses survive)
- `rg -o 'publish' tools/bench/BASELINE.md | wc -l` = `2`
- `SHAPETEST_FIXTURE_DIFF_LINES=0 SHAPETEST_FIXTURE_ENTRIES=10 SHAPETEST_FIXTURE_ENTRIES_PREPLAN=10`
- `BASELINE_FIGURE_DIFF_LINES=0 BASELINE_FIGURES_PRESENT=38`
- `task lint:actions` — clean (actionlint, no findings)
- `go build ./...`, `go vet ./...`, `go test -count=1 ./internal/upgrade/...` — all pass, including `TestWorkflowRunBodiesInvokeTask`

## Decisions Made

- **BASELINE.md:64 re-authored, not dropped.** The plan offered either
  option depending on whether the observation "depended entirely on a
  second subject." The darwin/arm64-vs-linux/amd64 platform-spread finding
  is about hardware/OS class (measured via whichever subject the deleted
  captures recorded), not a two-binary comparison — so it survives on its
  own terms once the dangling `tools/bench/headtohead-*.json` file citation
  is removed.
- **Job `name:` retitled from PERF-01 to BENCH-01/BENCH-03.** Per the
  plan's Task 1 action item 4 ("retitle the summary heading to name
  absolute per-corpus numbers under BENCH-01/BENCH-03 rather than a
  head-to-head result under PERF-01"). `BASELINE.md:241`'s own `PERF-01`
  citation was left untouched — Task 2's action item for that line was
  scoped narrowly to the job-name repoint, and `PERF-01` is not part of
  either task's census pattern set.
- **New bespoke YAML structs, not an extension of the package's existing
  ones.** `workflowRunStep`/`fullWorkflowJob` (used elsewhere in this
  package) don't carry `runs-on:`/`if:`/typed `with:` together; widening
  either shared type for this file's needs risked changing what every
  OTHER guard in `internal/upgrade` sees. `benchStep.With` is
  `map[string]any` (not `map[string]string`) because `Set up Go`'s
  `cache: false` is a YAML bool sitting next to a string sibling — decoding
  the whole `with:` block into `map[string]string` would fail the entire
  document's `Unmarshal`, not just that one field.

## Deviations from Plan

None — plan executed exactly as written. One measurement correction
recorded above (Task 1's pre-edit retired-term census returns 25 hits, not
the plan's stated 26; all in-scope conclusions the plan draws from that
count — the line-295 dangling reference, the authorised single-word
repoint, the companion diagnostic-job-untouched gate — hold regardless of
whether the total is 25 or 26, since line 295 is confirmed the sole
out-of-region hit either way).

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required. BENCH-03's remaining
half (a live `workflow_dispatch` of the `publish` job, per the plan's
`<success_criteria>`) is explicitly deferred to 06-04 as a blocking
decision gate, not a manual step in this plan.

## Next Phase Readiness

- `bench.yml`'s `publish` job and `internal/upgrade/bench_workflow_shape_test.go`
  are ready for 06-04, which makes a live dispatch of this job a blocking
  decision gate rather than a conditional check.
- No blockers. `rebless` and every diagnostic job in `bench.yml` are
  structurally proven untouched.

## Self-Check: PASSED

- FOUND: `.github/workflows/bench.yml`
- FOUND: `internal/upgrade/bench_workflow_shape_test.go`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PREPLAN-SHA.txt`
- FOUND: `tools/bench/BASELINE.md`
- FOUND: `internal/upgrade/taskfile_shape_test.go`
- FOUND commit `bd98d47` (Task 1)
- FOUND commit `e8cab5d` (Task 2)

---
*Phase: 06-benchmark-de-coupling-memory-sweep*
*Completed: 2026-08-16*
