---
phase: 05-release-chain-release-please-retirement
plan: 07
subsystem: release-tooling
tags: [release-please-retirement, census, positive-control, guard-tests, changie, github-actions]
status: complete

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "05-05: every pinning guard re-pointed off the retired files (only the three registry entries left); 05-06: 05-CENSUS.md with the D-10 verdicts; 05-01..05-04: the changie release chain, scanners and docs"
provides:
  - "the retired release workflow, config and manifest deleted in one commit together with their three registry entries (ffd84c4a)"
  - "05-CENSUS.md: the recorded REL-14 census instrument, its positive-control transcript, the clean-tree transcript and the phase gate"
  - "05-MUTATION-LOG.md: the registry RED against the replacement and the re-pointed green run"
affects: [phase-6-live-chain, milestone-merge]

plan_head_before: 09bc14daea46d23adad9db9e0f4f72ba41973ffa
plan_head_after: c20e29776cc787e17c5646ea0964e13d20abd54f

actuals:
  tokens: 9000
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "retire a subject by showing its registry guards RED against the bare deletion in a detached scratch worktree, then deleting subject and registry entries in one commit"
    - "a census is trusted only after it finds planted references (content, filename, in-scope .changes/) and skips planted references under each excluded path, each exclusion first confirmed to match under an unexcluded grep"

key-files:
  created: []
  modified:
    - internal/upgrade/taskfile_shape_test.go
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md
    - .planning/phases/05-release-chain-release-please-retirement/05-CENSUS.md
  deleted:
    - .github/workflows/release-please.yml
    - release-please-config.json
    - .release-please-manifest.json

key-decisions:
  - "Census recorded as an instrument in 05-CENSUS.md, not committed as a Taskfile target (plan decision; 04-02 precedent, D-01)"
  - "Exclusion set is exactly REL-14's three clauses, with .changes/unreleased/ as the concrete path of 'this milestone's own notes' (D-09, D-11); .changes/header.tpl.md stays in scope"

patterns-established:
  - "The deletion is the phase's last change outside .planning/: every later commit touches only .planning/"

requirements-completed: [REL-14, GRD-15]

coverage:
  - id: D1
    description: "With only the three retired files deleted, TestWorkflowRunBodiesInvokeTask, TestWorkflowFilePopulationMatchesDisk and TestInScopeJobsPopulationMatchesDisk FAIL; removing their three registry entries makes the whole internal/upgrade package pass"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "internal/upgrade#TestWorkflowRunBodiesInvokeTask, TestWorkflowFilePopulationMatchesDisk, TestInScopeJobsPopulationMatchesDisk (05-MUTATION-LOG.md, 05-07 sections)"
        status: pass
    human_judgment: false
  - id: D2
    description: "One commit deletes the three retired files and removes their inScopeJobs entry, usesOnlyJobExceptions entry and inScopeWorkflowFiles element; it touches exactly those four paths and the package passes at that commit"
    requirement: REL-14
    verification:
      - kind: command
        ref: "git diff-tree on ffd84c4a lists exactly the four paths; GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1; task lint:actions"
        status: pass
    human_judgment: false
  - id: D3
    description: "The census instrument finds planted content, filename and in-scope .changes/header.tpl.md references and skips the .planning/ and .changes/unreleased/ plants (each confirmed to match under an unexcluded grep); on the real tree it prints nothing, and the history files still hold the hits the exclusion skips"
    requirement: REL-14
    verification:
      - kind: command
        ref: "05-CENSUS.md ## Positive control, ## Clean-tree census; Task 2 verify blocks 1 and 2"
        status: pass
    human_judgment: false
  - id: D4
    description: "The deletion is the last change outside .planning/; no CI-skip phrase in any commit since 5a8603f5; no scratch worktree remains; changie latest is the version source"
    requirement: REL-14
    verification:
      - kind: command
        ref: "Task 2 verify block 3; task changie -- latest prints v0.14.1"
        status: pass
    human_judgment: false
  - id: D5
    description: "Phase gate: task test, check:changie 14 of 14, check:fragment-policy 46 of 46, lint:actions, the upgrade package, release.yml one comment-only hunk against 7b12cb92, verify.go / CHANGELOG.md / .changes/v*.md equal to 5a8603f5"
    requirement: GRD-15
    verification:
      - kind: command
        ref: "05-CENSUS.md ## Phase gate; Task 2 verify block 4"
        status: pass
    human_judgment: false

duration: 25 min
completed: 2026-10-14
---

# Phase 5 Plan 07: Retire release-please and prove it with a positive-controlled census Summary

**The release-please workflow, config and manifest are deleted in one commit with their three registry entries (registry guards shown RED against the bare deletion first), and a positive-controlled `git grep` census with exactly REL-14's three exclusions finds nothing on the real tree.**

This phase changes CI, release tooling and docs only. The live chain (changie release PR, tag, goreleaser) is Phase 6, entered by the milestone merge. No push, tag, PR or publishing release was made.

## Performance

- **Duration:** 25 min
- **Started:** 2026-10-14T22:08Z (approximate)
- **Completed:** 2026-10-14T22:33Z
- **Tasks:** 2
- **Files modified:** 1 source test file, 2 planning files, 3 files deleted

## Accomplishments

- **Registry RED against the replacement (GRD-15).** In a detached scratch worktree of `09bc14da`, deleting only the three retired files made exactly three tests fail, each on its own assertion naming the deleted workflow. `TestChangieBreakingBumpsMinorPreOne`, `TestGsdTagCreationIsDisabled` and the rest of the package passed there (05-05 re-pointed them). Removing the three registry entries in the same worktree turned the package green.
- **One deletion commit (REL-14).** `ffd84c4a` deletes `.github/workflows/release-please.yml`, `release-please-config.json` and `.release-please-manifest.json` and removes the `inScopeJobs` entry (`release-please.yml`/`pretag-gate`), the `usesOnlyJobExceptions` entry (`release-please.yml`/`release-please`; `govulncheck` stays) and the `inScopeWorkflowFiles` element. `git diff-tree` lists exactly those four paths. The package passed and `task lint:actions` reported nothing at that commit, so the suite was never red between commits.
- **Census instrument and positive control (D-09, D-11).** Recorded verbatim in `05-CENSUS.md`. Its exclusion set is exactly the requirement's three clauses (`.planning/`; `CHANGELOG.md` and `.changes/v*.md`; `.changes/unreleased/`); `.changes/header.tpl.md` stays in scope. In a scratch worktree of the post-deletion HEAD, five plants were staged. An unexcluded `git grep -l` listed both excluded plants. The instrument printed exactly the four in-scope lines (both `.changes/header.tpl.md` lines, the docs line, the scripts filename) and neither excluded plant. After `git reset --hard HEAD` the status with `--ignored --untracked-files=all` was empty and the worktree was removed.
- **Clean-tree census.** On the real tree the instrument prints nothing; the three retired paths are absent on disk; an unexcluded grep over the history files still lists exactly `.changes/v0.2.0.md` and `CHANGELOG.md`. `task changie -- latest` prints `v0.14.1`, the newest `.changes/v*.md`, and no version manifest remains.

## RED transcript (Go; repo rule x1cjy9vyhq, TAP classifier skipped)

Command, from the scratch worktree with only the three files deleted: `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 -v` (exit 1; exactly three `--- FAIL:`):

```
=== RUN   TestWorkflowRunBodiesInvokeTask
    taskfile_shape_test.go:1400: read ../../.github/workflows/release-please.yml: open ../../.github/workflows/release-please.yml: no such file or directory
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.00s)
=== RUN   TestWorkflowFilePopulationMatchesDisk
    taskfile_shape_test.go:1613: inScopeWorkflowFiles names "release-please.yml", which no longer exists on disk
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
=== RUN   TestInScopeJobsPopulationMatchesDisk
    taskfile_shape_test.go:1669: usesOnlyJobExceptions: release-please.yml/release-please: read ../../.github/workflows/release-please.yml: open ../../.github/workflows/release-please.yml: no such file or directory
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.538s
```

Semantic assessment: each target executed and failed on the assertion it exists to make (the deleted workflow is still named by a registry), not on a compile, load or fixture fault. After the registry removals: `ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.535s`.

## Control and clean-tree census outputs

Positive control (`census "$W"` at `92a4ce15`; full transcript in `05-CENSUS.md`):

```
.changes/header.tpl.md:1:planted Release Please reference
.changes/header.tpl.md:2:planted RELEASEPLEASE reference
docs/__census_control__.md:1:planted release-please reference
scripts/__release_please_control__.txt
```

Clean tree: `x=$(census .; true); printf "[%s]\n" "$x"` prints `[]`.

## Phase gate

- `GOTOOLCHAIN=go1.26.9 task test`: exit 0, no FAIL line (includes `test:golden`, `test:daemon` 64.5 s and `test:race`; the documented internal/daemon flake did not occur).
- `task check:changie`: `14 of 14 checks passed against a scratch copy (source tree byte-unchanged)`.
- `task check:fragment-policy`: `executed 46 of 46 defined test methods (floor 46)`.
- `task lint:actions`: no finding. `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1`: `ok`.
- `release.yml` against `7b12cb92`: one hunk, comment-only.
- `internal/upgrade/verify.go`, `CHANGELOG.md` and `.changes/v*.md` equal `5a8603f5`; `.changes` (header.tpl.md included) is byte-identical outside `unreleased`.
- No `[skip ci]` / `[ci skip]` in any commit since `5a8603f5`; no `p05-scratch-wt` worktree remains.
- Read-only remote checks: `gh api .../git/matching-refs/tags/v0.15 --jq length` is `0`; `git ls-remote origin refs/heads/main` is still `7b12cb92c407ae7e495a62b4ac38d8685e64307c`, so `main` cut no release during the phase and no verbatim seed is needed for the milestone merge (memory jc6h9cf1qm).

## Task Commits

1. **Task 1 deletion:** `ffd84c4a` ci(05-07): delete the retired release workflow, config and manifest (REL-14)
2. **Task 1 log:** `92a4ce15` docs(05-07): record the registry RED against the replacement
3. **Task 2 census and gate:** `c20e2977` docs(05-07): record the positive-controlled census and the phase gate (REL-14)

The plan metadata commit (this SUMMARY, STATE.md, ROADMAP.md, REQUIREMENTS.md) follows. Commits touching a path outside `.planning/` after `ffd84c4a`: none.

## Deviations from Plan

None - plan executed exactly as written.

Notes that are not deviations:
- `.git/gsd-plan-head-before-05-07` held `0f2549f5`, which is not an ancestor of HEAD; per the orchestrator it was reset to the then-current HEAD `09bc14da` before the first commit. `commits: 3` is measured from that base.
- The registry removals were edited on the real branch and the edited file copied byte-for-byte into the scratch worktree for the green run (the RED transcript was recorded first, against the unedited registry). The plan's order of showing RED before editing is preserved.
- The `tdd-red-evidence` classifier was not run (repo rule x1cjy9vyhq: TAP-only, always INVALID_RED for `go test`).
- The first Task 2 verify block was reproduced with a script using an explicit scratch directory (BSD `mktemp -d` ignores TMPDIR) and recorded verbatim in 05-CENSUS.md rather than re-run as one line.

## Authentication Gates

None.

## Known Stubs

None.

## Threat Flags

None. The plan removed workflow surface and added none.

## Requirements

- REL-14: complete (files deleted, census clean and positive-controlled, `changie latest` is the version source).
- GRD-15: complete (every pinning guard re-pointed across 05-01, 05-03, 05-04, 05-05 and 05-07; the last three registry entries leave with the files).

## Next Phase Readiness

Phase 5 plans are complete; ready for phase verification. The live chain is Phase 6, entered by the milestone merge. Out-of-repo state (the GitHub App display name and its Issues permission) is untouched and is not a census subject.

## Self-Check: PASSED

Commits `ffd84c4a`, `92a4ce15` and `c20e2977` are ancestors of HEAD; the edited test file, `05-CENSUS.md` and `05-MUTATION-LOG.md` exist; the three retired paths are absent from disk.
