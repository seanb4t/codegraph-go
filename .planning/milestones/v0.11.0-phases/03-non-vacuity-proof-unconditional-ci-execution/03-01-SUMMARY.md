---
phase: 03-non-vacuity-proof-unconditional-ci-execution
plan: 01
subsystem: testing
tags: [ci, golden-suite, testdata, corpora, go-test-cache, exact-equality, github-actions]

# Dependency graph
requires:
  - phase: 02-golden-harness-re-authoring-re-freeze
    provides: the re-frozen golden suite (26 goldens) + the CASES.json-driven behavioral assertions
  - phase: 01-corpus-selection-by-measurement
    provides: the corpus-aware corpora.yml workflow, nscloud cache wiring, and the corpora:fetch/corpora:assert targets
provides:
  - executed-scenario-count self-assertion (TestGoldenScenarioCountIsExact, exact verified guard, executedCases accumulator)
  - unconditional golden CI job (fetch -> assert -> test:golden) in corpora.yml, never cache-gated
  - widened transitive-closure path filter with a maintenance-rule comment
  - ci.yml reconciliation (corpus-dependent golden step removed, D-04)
  - inScopeJobs guard entry binding the new job's run bodies
affects: [phase 05 process/CI sweep, FIXT-03, FIXT-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 2622   # chars/4 over the realized diff (10488 content chars added+removed)
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "exact-equality scenario-count self-assertion keyed to loop-executed counters (wire-oracle TestScenarioCountIsExact precedent)"
    - "unconditional CI job never gated on cache-hit, with -count=1 defeating the go test cache"
    - "transitive-closure path filter naming only packages that exist, with a maintenance-rule comment"

key-files:
  created: []
  modified:
    - testdata/golden/golden_test.go
    - testdata/golden/behavioral_test.go
    - .github/workflows/corpora.yml
    - .github/workflows/ci.yml
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "ExpectedGoldenScenarioCount = 30 placed beside the expectedGoCaptures derivation and proven by TestGoldenScenarioCountIsExact (26 goldens + 4 CASES cases = 30, exact equality, zero-guards on both legs, individual 26/4 adjacency checks)"
  - "TestReFrozenGoldensValid guard upgraded from verified < expectedTotal (lower bound) to verified != expectedTotal (exact), and TestCorpusBehaviorSynthetic asserts executedCases == case total with the counter incremented inside the per-case closure — the positive claim measures EXECUTION, not inventory"
  - "The golden suite runs ONLY in corpora.yml's new golden job (D-04); ci.yml's corpus-dependent test:golden step is removed, not skipped, and no fetch/cache wiring is duplicated into ci.yml"
  - "No needs: entry between the corpora and golden jobs — the fetch driver's claim-lock + staged-promote serializes concurrent fetchers; first real parallel dispatch is the verification instrument, with recorded fallback needs: [corpora]"

patterns-established:
  - "A zero-executed golden run is RED by construction: Fatal on expectedTotal == 0, Fatal on goldenTotal == 0, Fatal on caseTotal == 0, exact executed == total guards on both the golden and behavioral loops"
  - "Every go test invocation in the golden path carries -count=1 (Taskfile test:golden target and the verification commands)"

requirements-completed: [FIXT-03]

coverage:
  - id: D1
    description: "Executed-scenario-count self-assertion for the golden suite — TestGoldenScenarioCountIsExact proves the derived 30 (26 goldens from expectedGoCaptures + 4 CASES.json cases) with exact equality; TestReFrozenGoldensValid's executed-verified guard is exact (verified == expectedTotal); TestCorpusBehaviorSynthetic asserts executedCases == case total. Three RED directions proven and byte-cleanly reverted."
    requirement: FIXT-03
    verification:
      - kind: unit
        ref: "testdata/golden/golden_test.go#TestGoldenScenarioCountIsExact"
        status: pass
      - kind: unit
        ref: "testdata/golden/golden_test.go#TestReFrozenGoldensValid"
        status: pass
      - kind: unit
        ref: "testdata/golden/behavioral_test.go#TestCorpusBehaviorSynthetic"
        status: pass
    human_judgment: false
  - id: D2
    description: "Unconditional golden job in corpora.yml (fetch -> assert -> test:golden, never cache-gated), widened transitive-closure path filter with maintenance-rule comment, -count=1 on the test:golden Taskfile target, {corpora.yml, golden} in inScopeJobs, and ci.yml's corpus-dependent golden step removed (D-04)."
    requirement: FIXT-03
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: integration
        ref: "task test:golden (local, after task corpora:fetch + task corpora:assert)"
        status: pass
      - kind: integration
        ref: "task lint:actions"
        status: pass
      - kind: other
        ref: "rg test:golden .github/workflows/ci.yml (must be empty)"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-08-15
status: complete
---

# Phase 03 Plan 01: Unconditional CI Execution & Executed-Scenario-Count Self-Assertion Summary

**The golden suite now proves its own execution: a table-derived, loop-keyed exact count (26 goldens + 4 CASES cases = 30) plus an unconditional corpora.yml golden job (fetch → assert → `task test:golden` with -count=1) that CI cannot silently stop running.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-08-15T14:00:00Z (approx)
- **Completed:** 2026-08-15T14:30:10Z
- **Tasks:** 2 (1 tracer + 1 auto)
- **Files modified:** 6

## Accomplishments

- **TestGoldenScenarioCountIsExact** (new): derives the scenario total from the authoritative tables — 26 goldens summed across `expectedGoCaptures` file lists plus `len(loadBehavioralCases(t))` = 4 from the committed CASES.json — and asserts `goldenTotal+caseTotal == ExpectedGoldenScenarioCount` (30) with EXACT equality, mirroring the wire-oracle `TestScenarioCountIsExact` precedent. Individual `goldenTotal != 26`, `caseTotal != 4`, and zero-guards on both legs are enforced before the combined check (FIXT-03 adjacency/empty).
- **Execution-keyed counters**: `TestReFrozenGoldensValid`'s guard upgraded from the lower bound `verified < expectedTotal` to EXACT `verified != expectedTotal` (the count of goldens actually executed inside subtest closures must equal the enumerated 26); `TestCorpusBehaviorSynthetic` adds an `executedCases` accumulator incremented inside the case-loop closure, asserted `!= len(cases)` after the loop — a loop that never runs leaves the executed counters at zero and fails the exact assertions (review finding #1).
- **Three RED directions proven and byte-cleanly reverted**: (1) deleting CASES.json case "d" → `caseTotal = 3, want exactly 4`; (2) deleting `corpus/hugo/go-explore.json` → subtest Fatal + `verified 25 of 26` exact-guard failure; (3) early return before `executedCases++` → `executed 0 of 4` exact-guard failure. All reverted, `git status` clean for the mutated paths.
- **Unconditional `golden` job** added to `.github/workflows/corpora.yml` (D-03): copies the corpora job's first four steps verbatim, declares the same job-level `CODEGRAPH_CORPUS_DIR`, then three unconditional `run:` steps invoking exactly `task corpora:fetch` → `task corpora:assert` → `task test:golden`. NOT gated on `steps.cache-corpora.hit`; a cache miss falls through to a real fetch and a failing fetch/assert fails the job loudly (T-03-P1-02/03).
- **Path filter widened to the real transitive closure** (both `pull_request` and `push` blocks): adds `testdata/golden/**`, `corpus/**`, `internal/query/**`, `internal/cli/**`, `internal/mcp/**`, `cmd/**`, `internal/graphstore/**`, `internal/parser/**`, `internal/schema/**`, `go.mod`, `go.sum` (NOT the non-existent `internal/store/**`) with a maintenance-rule comment naming the real golden-pipeline packages (T-03-P1-05).
- **ci.yml reconciliation (D-04)**: the corpus-dependent "Test golden parity suite" step is removed and replaced by a pointer comment explaining the suite now runs only in corpora.yml's golden job where pinned corpora are fetched; `rg "test:golden" .github/workflows/ci.yml` is empty.
- **Single-definition guard bound**: `{Workflow: "corpora.yml", JobID: "golden"}` added as the last `inScopeJobs` entry; `TestWorkflowRunBodiesInvokeTask` green.
- **`test:golden` Taskfile target** now carries `-count=1` (constraint 3), defeating the go test cache so every CI invocation executes against the current corpus tree (T-03-P1-06).

## Task Commits

Each task was committed atomically:

1. **Task 1: Executed-scenario-count self-assertion (tracer)** - `70dd07c` (test: golden count self-assertion)
2. **Task 2: Unconditional golden job + widened filter + guard + ci.yml reconciliation** - `d9f343d` (ci: unconditional golden job in corpora.yml)

**Plan metadata:** (no separate docs commit — executed as a solo wave in a worktree; orchestrator owns shared state writes)

## Files Created/Modified

- `testdata/golden/golden_test.go` - Added `ExpectedGoldenScenarioCount = 30` constant beside `expectedGoCaptures`; new `TestGoldenScenarioCountIsExact`; upgraded `TestReFrozenGoldensValid` guard to exact `verified != expectedTotal`.
- `testdata/golden/behavioral_test.go` - Added `executedCases` accumulator incremented inside the per-case closure and exact post-loop assertion.
- `.github/workflows/corpora.yml` - Added unconditional `golden` job; widened both path filters to the transitive closure with a maintenance-rule comment; documented the parallel-volume (no needs:) decision.
- `.github/workflows/ci.yml` - Removed the corpus-dependent `test:golden` step (D-04); replaced with a pointer comment.
- `Taskfile.yml` - `test:golden` target now runs `go test -count=1 ./testdata/golden/...`.
- `internal/upgrade/taskfile_shape_test.go` - Added `{Workflow: "corpora.yml", JobID: "golden"}` to `inScopeJobs`.

## Decisions Made

- Placed `ExpectedGoldenScenarioCount = 30` beside the authoritative derivation (not a detached literal), mirroring `scenarios.go`'s `ExpectedScenarioCount = 42` placement.
- Exact equality is the only accepted shape on every executed-count guard — never a lower bound (FIXT-03 boundary/adjacency).
- The golden suite runs only in the corpus-aware workflow; no skip and no duplicated fetch/cache wiring in ci.yml (FIXT-03 forbids skips).
- No `needs:` coupling between the corpora and golden jobs — documented claim-lock rationale with `needs: [corpora]` as the recorded fallback if the first parallel dispatch shows a lock leak.

## Deviations from Plan

None of substance — the plan executed as written. One acceptance-criterion note:

- The literal acceptance command `rg -A2 "test:golden:" Taskfile.yml | rg "count=1"` does not match because the target's multi-line `desc:` block (extended in this change to document the -count=1 rationale) pushes the `cmds` line beyond the 2-line `-A2` window. The intent — the `test:golden` target body carries `-count=1` — is satisfied and verified directly (`go test -count=1 ./testdata/golden/...` in the target's `cmds`). This is a documentation-shape note, not a behavioral gap.

## Issues Encountered

- None. The corpus precondition (`task corpora:fetch` → `task corpora:assert`) confirmed all 4 locked corpora already cached at `/Users/sean/.cache/codegraph/corpora/` and verified 4/4; `task test:golden` ran green locally against them.

## Known Stubs

None. The changes are test assertions, a Taskfile flag, workflow job/step wiring, and a guard-fixture entry — no placeholder values, no empty data paths, no TODOs.

## Threat Flags

None. The `golden` job's surface (new CI job reading the nscloud cache volume and the widened path filter) is exactly the surface the plan's `<threat_model>` specifies (T-03-P1-01..08); no endpoint, auth path, file-access pattern, or schema change beyond the modeled register.

## Self-Check: PASSED

- Created/modified files exist: golden_test.go, behavioral_test.go, corpora.yml, ci.yml, Taskfile.yml, taskfile_shape_test.go (verified via git status clean + diff).
- Commits exist: `70dd07c`, `d9f343d` (verified via git log).
- `go test -count=1 ./testdata/golden/... -run 'TestReFrozenGoldensValid|TestGoldenScenarioCountIsExact|TestCorpusBehaviorSynthetic'` — ok (26/26 + 30/30 + 4/4, exact counts).
- `go test -count=1 ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask` — ok.
- `rg "test:golden" .github/workflows/ci.yml` — empty (exit 1 = no matches).
- `task lint:actions` — exit 0.
- `task test:golden` locally after `task corpora:fetch` + `task corpora:assert` — ok.

## Next Phase Readiness

- Plan 03-02 (FIXT-07 mutation-revert matrix) can proceed: the per-family mutation rhythm is already specified (D-01) and this plan's RED demonstrations establish the mutation → RED → byte-clean revert discipline the log records.
- The corpus-aware `golden` job is the single place the golden suite runs; the `inScopeJobs` guard will bind any future workflow edit touching its run bodies.
- Phase 5's in-tree comment/process sweep (PROC) remains deferred as planned — including the `test:golden` desc vocabulary note recorded in the plan's "Deferred this cycle".

---
*Phase: 03-non-vacuity-proof-unconditional-ci-execution*
*Plan: 01*
*Completed: 2026-08-15*
