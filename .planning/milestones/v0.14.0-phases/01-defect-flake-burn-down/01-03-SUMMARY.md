---
phase: 01-defect-flake-burn-down
plan: 03
subsystem: testing
tags: [go, bench, regression-gate, category-error-guard, tdd]

# Dependency graph
requires:
  - phase: 01-defect-flake-burn-down (plan 01-02)
    provides: watchdog per-instance seam and flake burn-down groundwork for this phase's testing discipline
provides:
  - "CheckRegression's fourth category-error guard: baseline/current Repo (corpus identity) mismatch refuses with a rebless-pointing message"
  - "repoString degrade-empty helper, sibling of runnerString/scratchFSString"
  - "GH #16 closed"
affects: [tools/bench, internal/bench, perf-gate]

# Actuals (#2632)
actuals:
  tokens: 1773
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Category-error guard template (GOOS/GOARCH, Runner, ScratchFS, now Repo): plain != comparison, degrade-empty helper for messages, empty-vs-empty passes, empty-vs-non-empty refuses in both directions, no normalisation"

key-files:
  created: []
  modified:
    - internal/bench/regression_test.go
    - internal/bench/regression.go

key-decisions:
  - "D-16 (locked, from 01-CONTEXT.md/01-RESEARCH.md): strict equality on Repo, no normalisation of any kind — the only Repo values ever reaching CheckRegression are the deterministic synthetic-seed{N}-count{M} ids synthesized at tools/bench/runner/main.go:691, per the write-site audit in 01-RESEARCH.md § Investigation 5."
  - "GREEN commit uses `fix(01-03)` rather than the generic TDD template's `feat(01-03)`, per the plan's own <action> instruction — this closes GH #16, a bug (missing guard), not a new feature. See 'TDD Gate Compliance' below."

patterns-established: []

requirements-completed: [FIX-09]

coverage:
  - id: D1
    description: "CheckRegression refuses a baseline/current Repo (corpus identity) mismatch, in both empty-vs-non-empty directions, while matching and both-empty pairs still pass"
    requirement: "FIX-09"
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/repo_mismatch_between_baseline_and_current_fails_even_when_runner,_scratch_fs_and_GOOS/GOARCH_match"
        status: pass
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/matching_repo_on_both_sides_passes"
        status: pass
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/empty_baseline_repo_against_non-empty_current_repo_fails"
        status: pass
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/non-empty_baseline_repo_against_empty_current_repo_fails"
        status: pass
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression/both_repos_empty_passes"
        status: pass
    human_judgment: false

duration: ~7min
completed: 2026-09-15
status: complete
---

# Phase 01 Plan 03: Repo Corpus-Identity Guard in CheckRegression Summary

**`CheckRegression` gains a fourth category-error guard — baseline/current `Repo` mismatch now refuses with a message pointing at `tools/bench/BASELINE.md`, closing GH #16.**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-09-15T04:32:49Z (approx, from STATE.md's last plan completion timestamp)
- **Completed:** 2026-09-15T04:39:28Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `CheckRegression` now compares `baseline.Repo` vs `current.Repo` — the one measurement-frame dimension (GOOS/GOARCH, Runner, ScratchFS, and now Repo) that was previously unguarded, so a throughput comparison across two different synthetic corpora refuses instead of producing a meaningless verdict.
- `repoString`, a `runnerString`/`scratchFSString` twin, degrades an empty `Repo` to `"(not recorded)"` for error messages.
- The guard's doc comment names the exact write sites (`tools/bench/runner/main.go:691`, `:759`) and sole call site (`:627`), spelling out that publish-mode `Metrics` (human corpus names) never reach `CheckRegression` — the Pitfall 16 assumption is now written where a future reader stands, not only in a planning artifact.
- Five new table-driven subtests pin the contract in both directions plus both controls (mismatch refuses, matching passes, empty-vs-non-empty refuses both ways, both-empty passes) — watched fail (3 of 5) against the unguarded code before the guard existed.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): four Repo subtests that fail against the unguarded comparison** - `44316f25` (test)
2. **Task 2 (GREEN): the Repo guard and its degrade-empty helper** - `e2b3bc54` (fix — see TDD Gate Compliance below)

**Plan metadata:** commit pending (this SUMMARY + STATE/ROADMAP/REQUIREMENTS update)

_Note: No REFACTOR commit — the GREEN implementation needed no cleanup; it mirrors the existing ScratchFS guard's shape exactly._

## RED Transcript (Task 1)

Command: `GOTOOLCHAIN=go1.26.6 go test ./internal/bench/ -run 'TestCheckRegression' -count=1 -v`, run against unmodified `regression.go` (commit `44316f25`, which touched only `regression_test.go`):

```
--- PASS: TestCheckRegressionAgainstCommittedBaseline (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_in_frame:_no_regression_passes (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_throughput_11%_slower:_exceeds_band_fails (0.00s)
    --- PASS: TestCheckRegressionAgainstCommittedBaseline/committed_baseline_peak_RSS_16%_larger:_exceeds_band_fails (0.00s)
--- FAIL: TestCheckRegression (0.00s)
    ... (25 pre-existing subtests PASS) ...
    --- FAIL: TestCheckRegression/repo_mismatch_between_baseline_and_current_fails_even_when_runner,_scratch_fs_and_GOOS/GOARCH_match (0.00s)
        regression_test.go:682: CheckRegression() = nil, want error
    --- PASS: TestCheckRegression/matching_repo_on_both_sides_passes (0.00s)
    --- FAIL: TestCheckRegression/empty_baseline_repo_against_non-empty_current_repo_fails (0.00s)
        regression_test.go:682: CheckRegression() = nil, want error
    --- FAIL: TestCheckRegression/non-empty_baseline_repo_against_empty_current_repo_fails (0.00s)
        regression_test.go:682: CheckRegression() = nil, want error
    --- PASS: TestCheckRegression/both_repos_empty_passes (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/bench	0.060s
```

3 of 5 new subtests FAIL (the three refusing cases: full mismatch, empty-baseline, empty-current) exactly as `<behavior>` specified; the two passing controls (`matching repo on both sides passes`, `both repos empty passes`) already pass, proving the guard cannot be written as an unconditional refusal. `git show --stat --format= HEAD -- internal/bench/regression.go` on this commit is empty — `regression.go` was byte-unchanged.

## GREEN Confirmation (Task 2)

After adding the guard and `repoString`, `GOTOOLCHAIN=go1.26.6 go test ./internal/bench/ -run TestCheckRegression -count=1 -v` shows **37/37 PASS, 0 FAIL** (diffed line-by-line against the RED transcript: exactly the 3 previously-failing subtests plus the parent aggregate flipped to PASS; zero subtests disappeared or newly broke). `GOTOOLCHAIN=go1.26.6 go test ./internal/bench/... -count=1` is green; `go vet ./internal/bench/...` and `gofmt -l internal/bench/regression.go` are both clean.

## TDD Gate Compliance

- **RED gate:** present — `test(01-03): add failing Repo corpus-identity subtests to TestCheckRegression` (`44316f25`).
- **GREEN gate:** present in substance but committed as `fix(01-03)` rather than the generic TDD template's `feat(01-03)` — `fix(01-03): refuse a baseline/current corpus-identity mismatch in CheckRegression` (`e2b3bc54`). The plan's own `<action>` text for Task 2 explicitly specified this exact commit message; `fix` was the correct classification because this closes GH #16, a missing-guard bug (Rule 1/2 territory), not new feature surface. This is a deliberate, plan-directed choice, not an omission — flagged here because the mechanical `git log --grep="^feat(01-03):"` gate-check would otherwise misreport GREEN as missing. (That same grep, run naively without a milestone/phase-directory anchor, also collides with an unrelated historical `feat(01-03)` commit from a different milestone's phase-3 work — `01-03` phase/plan numbers are reused across milestones per this repo's own CLAUDE.md note on `#4459`, another reason a bare commit-message grep is not a reliable oracle here.)
- **REFACTOR gate:** not applicable — no cleanup needed after GREEN.

## Files Created/Modified
- `internal/bench/regression_test.go` - five new `Repo` subtests in `TestCheckRegression`'s table, grouped after the `scratch_fs` block
- `internal/bench/regression.go` - new `Repo` category-error guard (after ScratchFS, before `FilesPerSec <= 0`) and `repoString` helper

## Decisions Made
- Followed D-16 exactly: strict equality, no normalisation, `Repo` guard placed after ScratchFS and before the validity checks, per the file's existing ordering convention.
- Used `fix(01-03)` for the GREEN commit per the plan's explicit instruction rather than the generic TDD `feat(...)` template (see TDD Gate Compliance above).

## Deviations from Plan

None — plan executed exactly as written. All `must_haves.truths`, `artifacts`, `key_links`, and `prohibitions` were satisfied without needing an auto-fix: no normalisation was added, `Metrics`' schema and `tools/bench/runner/main.go` were left byte-unchanged (confirmed via `git diff --stat`), and the doc comment states the Pitfall-16 assumption as required.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- GH #16 closed: `CheckRegression` now guards all four measurement-frame dimensions (GOOS/GOARCH, Runner, ScratchFS, Repo) with identical semantics.
- `internal/bench` package is fully green; no follow-up work identified by this plan.
- EDGE-FIX-09-01 (flagged assumption in PLAN.md frontmatter, re: whitespace-only Repo differences) remains genuinely open per the plan's own note — strict equality refuses on a trailing-whitespace difference exactly as it would on any other difference, consistent with D-16's "no normalisation" rule, but no acceptance criterion pinned this specific sub-case. Not blocking; carried forward for human review if it ever surfaces in practice.

## Self-Check: PASSED

- FOUND: internal/bench/regression_test.go
- FOUND: internal/bench/regression.go
- FOUND: .planning/phases/01-defect-flake-burn-down/01-03-SUMMARY.md
- FOUND: commit 44316f25 in `git log --oneline --all`
- FOUND: commit e2b3bc5 in `git log --oneline --all`
- Re-ran all acceptance criteria from both tasks: all PASS (see RED/GREEN transcripts above).
- Re-ran plan-level `<verification>`: 3/5 new subtests fail against unmodified `regression.go` (Task 1), all 5 pass after Task 2 with the whole `internal/bench` package green (Task 2), `repoString` exists and the message names the rebless mechanism with no normalisation call on any `Repo` line — all confirmed.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
