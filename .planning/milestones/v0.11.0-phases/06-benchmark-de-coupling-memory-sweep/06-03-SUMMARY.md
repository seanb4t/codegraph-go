---
phase: 06-benchmark-de-coupling-memory-sweep
plan: 03
subsystem: testing
tags: [go, benchmark, regression-gate, mutation-testing, fixt-07]

# Dependency graph
requires:
  - phase: 03-non-vacuity-proof-unconditional-ci-execution
    provides: the FIXT-07 five-step mutation-rehearsal template (03-MUTATION-LOG.md), reused verbatim per D-08
provides:
  - internal/bench comment-only sweep removing head-to-head/Go-vs-TS framing (BENCH-02 half 1)
  - internal/bench/baseline_gate_test.go — TestCheckRegressionAgainstCommittedBaseline, the first Go test to load the committed tools/bench/baseline.json
  - 06-MUTATION-LOG.md — FIXT-07 two-family rehearsal proving CheckRegression still fires against the committed baseline (BENCH-02 half 2)
affects: [06-01-comparison-runner-removal, 06-04-bench-yml-publish-job, phase-06-verification]

# Actuals (#2632)
actuals:
  tokens: 7213
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "FIXT-07 five-step mutation rehearsal (mutate/confirm/observe/revert/reconfirm) reused verbatim per D-08, extended here to require RED in TWO independent oracles per family"
    - "Comment-only sweep verified by code-portion equality (strip each changed diff line at its first //, compare removed vs added code prefixes) rather than a naive comment-line-shape regex"

key-files:
  created:
    - internal/bench/baseline_gate_test.go
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-MUTATION-LOG.md
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PREPLAN-SHA.txt
  modified:
    - internal/bench/metrics.go
    - internal/bench/regression.go
    - internal/bench/regression_test.go
    - internal/bench/rss.go

key-decisions:
  - "internal/corpora/manifest.go's cross-reference to tools/bench/realcorpus was verified accurate and left unchanged — 06-01 had not yet run in this worktree, so the referenced package's current prose still matches the cross-reference verbatim"
  - "corpora/manifest.json's line-2 note required no edit — census showed it already clean of every retired-framing pattern in this task's scope"
  - "Task 2's test derives its 'current' record from the loaded baseline's frame fields (GOOS/GOARCH/Runner/ScratchFS) rather than hard-coding them, so a future re-bless on a different runner class cannot silently turn a passing gate test into a category-error false-positive"

requirements-completed: [BENCH-02]

coverage:
  - id: D1
    description: "internal/bench package doc/comment prose describes absolute, externally-observed measurement on its own terms, citing no deleted capture file and no second runtime"
    requirement: BENCH-02
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/bench/... ./internal/corpora/..."
        status: pass
      - kind: other
        ref: "retired-term census (bounded TS pattern) over internal/bench/, internal/corpora/manifest.go, corpora/manifest.json — 0 hits"
        status: pass
    human_judgment: false
  - id: D2
    description: "internal/bench/baseline_gate_test.go's TestCheckRegressionAgainstCommittedBaseline loads the COMMITTED tools/bench/baseline.json and drives CheckRegression with it (review HIGH 4)"
    requirement: BENCH-02
    verification:
      - kind: unit
        ref: "internal/bench/baseline_gate_test.go#TestCheckRegressionAgainstCommittedBaseline (3 subtests: in-frame control, throughput 11% slower, peak RSS 16% larger)"
        status: pass
    human_judgment: false
  - id: D3
    description: "CheckRegression still fires against the committed baseline.json — demonstrated RED against two confirmed-applied, byte-cleanly-reverted mutations, each reddening both the pre-existing table AND the committed-baseline test"
    requirement: BENCH-02
    verification:
      - kind: other
        ref: ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-MUTATION-LOG.md — Family A (DefaultThroughputTolerance) and Family B (DefaultRSSTolerance), each with verbatim RED/revert/green-rerun"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-08-16
status: complete
---

# Phase 6 Plan 3: Benchmark comparison-framing sweep + committed-baseline mutation proof Summary

**Swept retired head-to-head/Go-vs-TS framing out of `internal/bench`'s comments (zero behavior
change, verified by code-portion-equality diff), added the first Go test to load the committed
`tools/bench/baseline.json`, and proved `CheckRegression` still fires by demonstrating it RED
against two independently mutated tolerance constants — each reddening both the pre-existing test
table and the new committed-baseline test, then reverted byte-cleanly.**

## Performance

- **Duration:** ~6 min (span between first and last task commit)
- **Tasks:** 3
- **Files modified:** 7 (4 comment-only edits, 1 new test file, 1 new mutation log, 1 new preplan-SHA anchor)

## Accomplishments

- Comment-only framing sweep across `internal/bench/{metrics,regression,regression_test,rss}.go`:
  removed every retired-framing term (head-to-head, Go-vs-TS, TS Node, bare `"ts"` example) and
  replaced the citations to 06-01's deleted `tools/bench/headtohead-*.json` capture files with
  citations to `tools/bench/BASELINE.md`'s recorded ~10.6% fictitious-regression investigation and
  the synthetic-corpus control numbers already quoted in `regression.go`. Zero tolerance constant,
  guard, error string, json tag or test case value changed — verified by code-portion-equality diff
  (strip each changed line at its first `//`, compare removed vs added code prefixes: identical).
- `internal/corpora/manifest.go` and `corpora/manifest.json`'s line-2 note were both verified
  accurate against `tools/bench/realcorpus/manifest.go` and left byte-unchanged (see Decisions).
- New `internal/bench/baseline_gate_test.go` with `TestCheckRegressionAgainstCommittedBaseline`:
  the first Go test in the repository to load `tools/bench/baseline.json` off disk, unmarshal it
  into `bench.Metrics`, and drive `CheckRegression` with it — closing cross-AI review HIGH 4
  (ROADMAP success criterion 2 names this exact file as the mechanism to prove).
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-MUTATION-LOG.md`: two independently
  mutated tolerance constants (`DefaultThroughputTolerance`, `DefaultRSSTolerance`, each widened to
  `1.0`), each confirmed applied, each reddening BOTH oracles (the pre-existing table subtest AND
  the new committed-baseline subtest), each reverted byte-cleanly with an empty post-revert
  `git diff --stat` and a verbatim green re-run.

## Task Commits

Each task was committed atomically:

1. **Task 1: Comment-only framing sweep across internal/bench and internal/corpora** - `763a438` (docs)
2. **Task 2: Drive the gate with the COMMITTED baseline.json, not an in-memory stand-in** - `b4c78e0` (test)
3. **Task 3: FIXT-07 mutation rehearsal — demonstrate the gate RED, revert byte-cleanly, record it** - `dfe86ae` (docs)

_This plan's SUMMARY commit follows as the metadata commit (worktree mode — STATE.md/ROADMAP.md
excluded; orchestrator updates those centrally after merge)._

## Files Created/Modified

- `internal/bench/rss.go` - package doc re-authored to state the peak-RSS-must-be-external rule on its own terms (no TS Node citation)
- `internal/bench/metrics.go` - struct/field doc re-authored to name the committed-baseline gate + absolute-numbers publisher; `Subject` example reduced to the surviving value
- `internal/bench/regression.go` - platform-mismatch rationale (`:37-48`) re-cites `tools/bench/BASELINE.md`'s ~10.6% investigation instead of the deleted headtohead capture glob
- `internal/bench/regression_test.go` - matching comment re-author at the GOOS/GOARCH-mismatch test case (`:117-125`); zero test case/table values touched
- `internal/bench/baseline_gate_test.go` - **new**: `TestCheckRegressionAgainstCommittedBaseline`
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-MUTATION-LOG.md` - **new**: FIXT-07 two-family rehearsal record
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PREPLAN-SHA.txt` - **new**: pre-plan HEAD SHA anchor (`575e610b0c03db21120c4a139ab1681899a0b681`)

## Before/After Measurements (recorded per plan's `<output>` spec)

**Subtest count, `TestCheckRegression` (must be unchanged by the comment sweep):**
- Before: 25 (`rg -c '^\s*name: '` over the pre-plan-SHA copy of `regression_test.go`)
- After: 25 (`go test -run TestCheckRegression -v | rg -c '^=== RUN   TestCheckRegression/'`)

**Per-file comment-line counts (`rg -c '^\s*//'`), before → after — all within the ±25% band, none below 90%:**
- `internal/bench/metrics.go`: 51 → 51
- `internal/bench/regression.go`: 70 → 72
- `internal/bench/regression_test.go`: 70 → 73
- `internal/bench/rss.go`: 18 → 21

**Retired-term census (bounded `TS` pattern), `internal/bench/` + `internal/corpora/manifest.go` +
`corpora/manifest.json`:** `BENCH_PKG_RETIRED_TERMS_TOTAL=0` (measured before edit: 7 hits, all in
`internal/bench/`, 0 in `corpora/manifest.json` — matching the plan's pre-measured figure exactly).

**`internal/corpora` verification statement:** All cross-references in `internal/corpora/manifest.go`
(package doc's non-merge paragraph, both `Note` field declarations) were read against
`tools/bench/realcorpus/manifest.go` and `corpora/manifest.json`'s committed top-level `note`, and
found accurate — **left unchanged**. `06-01` had not yet executed in this worktree at the time of
this verification (parallel wave-1 execution, `depends_on: []`), so `tools/bench/realcorpus/manifest.go`'s
prose was still in its pre-06-01 form and the existing cross-reference matched it verbatim.
`corpora/manifest.json`'s line-2 top-level `note` was independently censused and found already
clean of every retired-framing pattern in this task's scope — **no edit made**.

**`TestCheckRegressionAgainstCommittedBaseline` subtest names and derivation:**
- `committed baseline in frame: no regression passes` — `current := baseline` (full struct copy), proving the four category guards are quiet.
- `committed baseline throughput 11% slower: exceeds band fails` — current derived as `baseline.FilesPerSec * 0.89` with `PeakRSSBytes` and all frame fields (`Repo`, `Subject`, `GOOS`, `GOARCH`, `Runner`, `ScratchFS`) copied verbatim from the loaded baseline.
- `committed baseline peak RSS 16% larger: exceeds band fails` — current derived as `int64(float64(baseline.PeakRSSBytes) * 1.16)` with `FilesPerSec` and all frame fields copied verbatim.
- No absolute figure from `tools/bench/baseline.json` (17090.87…, 845950976, 2588799.96…, 258.199) appears as a literal anywhere in the test file — census confirms `0` hits.
- `ceilingBytes` is `0` in every case, disabling the absolute bounded-memory branch so the peak-RSS case can only fail through the relative tolerance band.

**Both mutation families' RED subtest names from BOTH oracles (`06-MUTATION-LOG.md`):**

| Family | Constant mutated | Table oracle RED | Committed-baseline oracle RED |
|---|---|---|---|
| A — throughput | `DefaultThroughputTolerance` 0.10 → 1.0 | `TestCheckRegression/throughput_11%_slower:_exceeds_band_fails` | `TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails` |
| B — peak RSS | `DefaultRSSTolerance` 0.15 → 1.0 | `TestCheckRegression/peak_RSS_16%_larger:_exceeds_band_fails` | `TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails` |

Both REDs are `CheckRegression() = nil, want error` / `CheckRegression(committed baseline) = nil,
want error` — the assertion the tolerance-widening mutation actually produces (the gate's own
tolerance-violation error strings are suppressed by the mutation, not reachable). No
category-mismatch phrase (platform/runner/scratch-filesystem) appears in either family's observed
output — neither RED was a category error.

## Decisions Made

- **`internal/corpora/manifest.go` and `corpora/manifest.json` line 2 required no edit.** Both were
  in scope per the frontmatter and the `<action>`'s permitted-scope note, but the pre-execution
  census (bounded retired-term pattern) returned 0 hits in both files, and manual review of
  `internal/corpora/manifest.go`'s cross-references against `tools/bench/realcorpus/manifest.go`
  found them still accurate — 06-01 (which rewrites `realcorpus`'s package doc) had not yet run in
  this parallel worktree. Recorded here rather than left silently inferred, per the plan's
  instruction.
- **Derived, not literal, "current" records in the committed-baseline test.** Copying the loaded
  baseline's frame fields verbatim (rather than hard-coding `GOOS: "linux"` etc.) keeps the test
  correct across a future re-bless on a different runner class — a literal would drift out of frame
  and fail with a category-mismatch error that looks like a firing gate while proving nothing
  (`06-RESEARCH.md` Pitfall 2).

## Deviations from Plan

None - plan executed exactly as written. One in-flight correction during Task 3 authoring: the
first draft of `06-MUTATION-LOG.md`'s "Failure-mode discipline" section quoted the literal phrase
"a runner mismatch" in its own prose while explaining what a wrong-RED would look like — the plan's
`<action>` explicitly prohibits this because Task 3's own acceptance gate negative-greps those exact
phrases over the whole log file to disqualify a wrong-RED, and a correct rehearsal's explanatory
prose would otherwise trip that gate on itself. Caught by re-running the gate before considering the
task done (`CATEGORY_ERROR_LINES` read `1`, not `0`); reworded to "a category-error RED" per the
plan's own prescribed escape, re-ran the gate, confirmed `0`. Not logged as a Rule 1-4 deviation
since it never left the log file in a state committed to git — the fix landed before the task
commit, inside the same authoring pass the plan's `<action>` describes.

## Issues Encountered

None beyond the self-caught gate issue above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BENCH-02 is fully closed: 06-01 removes the comparison runner, this plan proves the gate still
  fires against the committed baseline by demonstration.
- `internal/bench` and `internal/corpora/manifest.go` are ready for 06-04's `bench.yml` publish-job
  work with no outstanding framing cleanup in the files this plan owns.
- Depends on nothing from this wave; nothing in this wave depends on it (`depends_on: []`,
  `wave: 1`). Safe to merge independently of 06-01/06-02's outcomes.

## Self-Check: PASSED

- FOUND: `internal/bench/baseline_gate_test.go`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-MUTATION-LOG.md`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PREPLAN-SHA.txt`
- FOUND commit `763a438` (Task 1)
- FOUND commit `b4c78e0` (Task 2)
- FOUND commit `dfe86ae` (Task 3)

---
*Phase: 06-benchmark-de-coupling-memory-sweep*
*Completed: 2026-08-16*
