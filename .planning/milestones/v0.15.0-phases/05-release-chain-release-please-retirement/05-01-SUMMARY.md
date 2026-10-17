---
phase: 05-release-chain-release-please-retirement
plan: 01
subsystem: infra
tags: [changie, github-actions, create-pull-request, github-app, taskfile, release]

requires:
  - phase: 01-changie-foundation
    provides: ".changie.yaml, the isolated go.tool-changie.mod pin (task changie) and check:changie legs 1-13"
  - phase: 03-fragment-backfill
    provides: "the real Breaking fragment under .changes/unreleased/ used by the tracer rehearsal"
provides:
  - ".github/workflows/release-pr.yml: pretag-gate then release:batch and create-pull-request with the App token"
  - "Taskfile release:batch: fragment-count no-op, batch auto, merge, latest, skip-ci phrase scan, version output"
  - "check:changie leg 14 (14 of 14): batch auto then merge renders a well-formed version file and CHANGELOG.md"
  - ".changie.yaml newlines keys beforeKind, afterKind, endOfVersion (D-13)"
  - "TestReleasePRWorkflowShape, TestReleaseBatchTargetShape and the releaseChain* helpers later plans extend"
  - "05-MUTATION-LOG.md with the RED/GREEN transcripts and Families 05-01/a-h"
affects: [05-02, 05-03, 05-04, 05-05, 05-06, 05-07, phase-06-live-run]

actuals:
  tokens: 23776
  tasks: 2
  commits: 5

tech-stack:
  added: [peter-evans/create-pull-request v8.1.1 (SHA-pinned GitHub Action)]
  patterns:
    - "workflow run bodies are a single `task <target>` call; the version crosses to the PR action only through steps.<id>.outputs in with: inputs"
    - "shape tests read this repository's YAML and Taskfile bodies, with a positive population report before asserting"
    - "mutation families run in a detached p05-scratch-wt worktree with a pre-mutation gate, applied-confirmation and byte-clean revert"

key-files:
  created:
    - .github/workflows/release-pr.yml
    - internal/upgrade/changie_release_shape_test.go
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md
  modified:
    - .changie.yaml
    - Taskfile.yml
    - internal/upgrade/changie_shape_test.go
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "D-13 (maintainer, 2026-10-14) amends Phase 1 D-02: .changie.yaml newlines holds afterChangelogHeader 1, beforeKind 1, afterKind 1, endOfVersion 2, so a batched version file ends in a blank line and changie merge never glues a heading onto the last bullet"
  - "Two workflow files (release-pr.yml here, release-tag.yml in 05-02); the documented path filter costs one duplicated pretag-gate job (C7)"
  - "release:batch is held by a static shape test plus worktree rehearsals, not by check:changie legs, because a Taskfile target always runs in the Taskfile's own directory and would batch the real tree"
  - "App-token permission narrowing is not adopted; the SHA pin is the control (T-05-SC)"

patterns-established:
  - "releaseChain prefix for every new package-level identifier in package upgrade"
  - "check:changie leg 14 evaluates all four sub-checks before failing and prints one ::error:: line per failing sub-check"

requirements-completed: [REL-10]

coverage:
  - id: D1
    description: "release-pr.yml: push to main only, contents: read, pretag-gate then release-pr job, App token, ready-for-review PR on changie/release, four SHA pins"
    requirement: REL-10
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleasePRWorkflowShape"
        status: pass
      - kind: other
        ref: "task lint:actions"
        status: pass
    human_judgment: false
  - id: D2
    description: "Taskfile release:batch: silent, ordered tokens, GO_TOOL_CHANGIE only, empty-set no-op, skip-ci phrase scan, version output"
    requirement: REL-10
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleaseBatchTargetShape"
        status: pass
      - kind: integration
        ref: "task release:batch in a detached worktree of HEAD (tracer and empty-set rehearsals, 05-MUTATION-LOG.md)"
        status: pass
    human_judgment: false
  - id: D3
    description: ".changie.yaml newlines keys so a batched version file ends in a blank line (D-13)"
    requirement: REL-10
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieConfigShape"
        status: pass
      - kind: integration
        ref: "task check:changie (14 of 14)"
        status: pass
    human_judgment: false
  - id: D4
    description: "check:changie leg 14 with four independently reported sub-checks"
    requirement: REL-10
    verification:
      - kind: integration
        ref: "task check:changie; Families 05-01/a, 05-01/g, 05-01/h in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D5
    description: "Shape-test registry: inScopeJobs and inScopeWorkflowFiles hold release-pr.yml (GRD-15, registry only)"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask, TestWorkflowFilePopulationMatchesDisk, TestInScopeJobsPopulationMatchesDisk"
        status: pass
    human_judgment: false
  - id: D6
    description: "The live release PR is opened by the App on main and its required checks report"
    requirement: REL-10
    verification: []
    human_judgment: true
    rationale: "Needs the App secrets and a real push to main; the live run is Phase 6. This plan shows only that the PR's content passes the required test job's changie guards, in a disposable worktree."

duration: 12min
completed: 2026-10-14
status: complete
plan_head_before: 43748a81b6b08500e6ff9d49b28a22b7fd81352f
plan_head_after: a7a8240c121cf53a33e783b850d7146a688faf82
commits: 5
---

# Phase 5 Plan 01: Release-PR half of the changie chain Summary

**A push to main with unreleased fragments now batches them (changie batch auto, merge, latest through the pinned `task changie` path) into one `chore(main): release vX.Y.Z` PR opened with the GitHub App token, behind pretag-gate, with the newlines keys that keep the batched CHANGELOG.md well-formed and eight proven-RED guards.**

This plan changes CI and release tooling only. No codegraph user-visible behavior changes.

## Performance

- **Duration:** 12 min
- **Started:** 2026-10-14T21:06:25Z
- **Completed:** 2026-10-14T21:18:00Z
- **Tasks:** 2
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments

- `release-pr.yml` (`name: release-pr`, push to main only, `contents: read`, concurrency group `release-pr`) runs `pretag-gate` (`task check:cross`), then mints the App token, checks out, installs Go and Task, runs `task release:batch` (step id `batch`) and opens or updates the PR with `peter-evans/create-pull-request` (pinned to a SHA) on branch `changie/release`, title and commit message `chore(main): release vX.Y.Z`, body from `.changes/vX.Y.Z.md`, `draft: false`. The version crosses only through `with:` inputs; no run body holds an expression.
- `task release:batch` counts `*.yaml` fragments (`.gitkeep` never counts), prints a notice and exits 0 when there are none (CHG-03, C1), otherwise runs batch auto, merge and latest, fails on a GitHub skip-ci phrase in the rendered notes (C2), and writes `version=<v>` to GITHUB_OUTPUT.
- `.changie.yaml` gains `beforeKind: 1`, `afterKind: 1`, `endOfVersion: 2` (D-13). Without them a batched version file has no trailing newline and `changie merge` glues the next `## [` heading onto the last bullet; the tracer RED reproduced the lost heading and `TestChangieVersionSeedsMatchChangelog` failing.
- `check:changie` runs 14 legs and prints `14 of 14`. Leg 14 batches and merges the scratch fragments, evaluates all four sub-checks, and prints one `::error::check:changie: [14/14] (i|ii|iii|iv)` line per failing sub-check.
- Tracer, end to end: in a disposable worktree of HEAD, `task release:batch` on the real Phase 3 fragment wrote exactly `version=v0.15.0` (equal to `task changie -- next auto`) and `.changes/v0.15.0.md`; the changie Go guards and `task check:changie` passed there. The empty-set rehearsal exited 0, wrote nothing to GITHUB_OUTPUT and left CHANGELOG.md and every `.changes/v*.md` unchanged. The real tree was not batched, nothing was pushed, no PR was opened.
- Mutation Families 05-01/a to 05-01/h are recorded in `05-MUTATION-LOG.md`, each with a pre-mutation gate, a confirmed-applied mutation, the RED transcript, a byte-clean revert and a GREEN re-run. Leg 14's (ii), (iii) and (iv) were each seen failing on their own line (Families a, g and h); (i) and (ii) in the RED transcript.

## TDD Gate Compliance

RED evidence is the `test(05-01)` commit that landed before the implementation plus the pasted `--- FAIL:` transcript. `gsd-tools check tdd-red-evidence` was not used: it parses TAP only and always reports INVALID_RED for `go test` (rule x1cjy9vyhq).

| Gate | Commit | Subject |
|------|--------|---------|
| RED | `219303c7` | test(05-01): expect a release-PR workflow, its batch body and well-formed batched notes (REL-10) |
| GREEN | `106e6cd6` | ci(05-01): batch unreleased fragments into one release PR through the GitHub App (REL-10) |
| follow-up | `c4a2c122` | test(05-01): sort the expected pin set in TestReleasePRWorkflowShape |

The RED commit holds exactly `Taskfile.yml` and the three test files; the GREEN commit holds exactly `.changie.yaml`, `.github/workflows/release-pr.yml` and `Taskfile.yml`. The first two commits touching `changie_release_shape_test.go` or `release-pr.yml` after `5a8603f5` are test(05-01) then ci(05-01).

`go test ./internal/upgrade/ -count=1 -v` at `219303c7` (assertion and read failures on the missing subject, no build error; exactly these six fail):

```
--- FAIL: TestReleasePRWorkflowShape (0.00s)
--- FAIL: TestReleaseBatchTargetShape (0.00s)
--- FAIL: TestChangieConfigShape (0.00s)
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.00s)
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.508s
```

`task check:changie` at `219303c7` stopped at leg 14 (legs 1-13 ok):

```
::error::check:changie: [14/14] (i) .changes/v0.15.0.md does not end in two newlines (a blank line), so the next merged heading would be glued onto its last bullet
::error::check:changie: [14/14] (ii) CHANGELOG.md holds a '## [' heading that is not at the start of its line: 10:- check:changie collision probe two ([#3](https://github.com/seanb4t/codegraph-go/pull/3))## [0.14.1](...) (2026-10-13)
```

Tracer RED, end to end (scratch worktree at `219303c7`, `task changie -- batch auto` then `task changie -- merge`, then the seed guard):

```
--- FAIL: TestChangieVersionSeedsMatchChangelog (0.00s)
    changie_shape_test.go:399: ../../.changes vs ../../CHANGELOG.md: set mismatch — versions only in seeds: [v0.14.1], versions only in changelog headings: []
```

Tracer GREEN (worktree at `c4a2c122` content): `release:batch: rendered v0.15.0 from 1 fragment(s)`, GITHUB_OUTPUT `version=v0.15.0`, all eight `TestChangie*` tests PASS, `check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)`. The six named tests pass in the main checkout and `task lint:actions` exits 0. Full transcripts: `05-MUTATION-LOG.md` (`## 05-01 RED-first transcript`, `## 05-01 Tracer RED (end to end)`, `## 05-01 Tracer GREEN (end to end)`).

## Task Commits

1. **Task 1: Tracer, RED first** - `219303c7` (test), `106e6cd6` (ci), `c4a2c122` (test, pin-order fix), `c4c19ade` (docs, RED/GREEN transcripts)
2. **Task 2: Prove every 05-01 guard bites** - `a7a8240c` (docs, Families 05-01/a-h)

**Plan metadata:** the `docs(05-01): complete release-PR plan` commit that follows this file.

## Files Created/Modified

- `.github/workflows/release-pr.yml` - the release-PR workflow
- `Taskfile.yml` - `release:batch` target; check:changie leg 14 and the `[n/14]` / `14 of 14` renumbering
- `.changie.yaml` - three newlines keys and the rewritten header comment
- `internal/upgrade/changie_release_shape_test.go` - new: `TestReleasePRWorkflowShape`, `TestReleaseBatchTargetShape`, `releaseChain*` constants, YAML structs and parse helpers
- `internal/upgrade/changie_shape_test.go` - newlines pin changed (D-13), comment reworded
- `internal/upgrade/taskfile_shape_test.go` - `inScopeJobs` and `inScopeWorkflowFiles` entries for release-pr.yml
- `.planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md` - transcripts and Families a-h

## Decisions Made

- D-13 (maintainer, 2026-10-14) supersedes Phase 1 D-02's "afterChangelogHeader is the only newlines override"; `TestChangieConfigShape` pins the four keys and values RED first. Uses changie's own config keys, not custom logic (D-01).
- GRD-15 is partial here: only the registry entries for release-pr.yml. The release-please entries stay until 05-07 deletes their files.
- Followed the plan's layout (two workflow files, fixed branch `changie/release`, default `delete-branch`, no `add-paths`, concurrency group with `cancel-in-progress: false`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Expected pin list in TestReleasePRWorkflowShape was not in sorted order**
- **Found during:** Task 1 (GREEN run)
- **Issue:** The set-equality assertion compared a sorted list of observed pins with an unsorted expected list, so it failed although the workflow carries exactly the four pins. RED could not show it because the workflow file was absent.
- **Fix:** Reordered `releaseChainExpectedPins`.
- **Files modified:** `internal/upgrade/changie_release_shape_test.go`
- **Verification:** the six named tests pass.
- **Committed in:** `c4a2c122`, a separate `test(05-01)` commit after the ci commit. The plan requires the GREEN commit to hold exactly three files, so the fix could not ride in it. Consequence: at `106e6cd6` alone, `TestReleasePRWorkflowShape` fails on that one assertion; it passes from `c4a2c122` on. The order check (first two commits touching the new test or workflow are test then ci) still holds.

**2. [Rule 1 - Bug] release-pr.yml `title:` and `commit-message:` were invalid YAML as plain scalars**
- **Found during:** Task 1 (GREEN run, caught by the shape test and actionlint)
- **Issue:** `chore(main): release ${{ ... }}` contains a colon followed by a space, which a plain YAML scalar cannot hold.
- **Fix:** Double-quoted both values.
- **Files modified:** `.github/workflows/release-pr.yml`
- **Verification:** actionlint clean, all six tests pass.
- **Committed in:** `106e6cd6` (before the commit, so no intermediate bad state)

**3. [Orchestrator note] Some scratch worktrees were created under the system temp dir, not the scratchpad**
- **Found during:** Task 1 (tracer GREEN) and Task 2
- **Issue:** BSD `mktemp -d` ignores `TMPDIR`, so those worktrees landed under `/var/folders/...`. The paths still contain `p05-scratch-wt`.
- **Fix:** none needed; every worktree was removed and pruned, and `git worktree list` shows only the two pre-existing entries.

---

**Total deviations:** 3 (2 Rule 1 bugs, 1 environment note)
**Impact on plan:** None on scope. The pin-order fix is a one-line fixture correction; the quoting fix is required for the workflow to parse.

## Issues Encountered

None beyond the deviations above. `internal/daemon` TestSoak was not run (package-scoped runs only).

## Known Stubs

None.

## User Setup Required

None - no external service configuration required in this plan. The live run needs the existing `APP_ID` and `APP_PRIVATE_KEY` secrets and is Phase 6's.

## Next Phase Readiness

- 05-02 can add `release-tag.yml` and extend `changie_release_shape_test.go` using the `releaseChain*` helpers.
- Risk named in the plan, not planned around: release-please stays live on `main` until the milestone merges; a release cut on `main` in the meantime needs a verbatim byte-slice seed `.changes/vX.Y.Z.md` in the next main-to-milestone merge (quick 261013-esj precedent), or `TestChangieVersionSeedsMatchChangelog` goes red.
- Unchanged from `5a8603f5`: CHANGELOG.md, every `.changes/v*.md`, `.changes/unreleased/`, `release.yml` and `internal/upgrade/verify.go`.

## Self-Check: PASSED

Created files exist on disk (release-pr.yml, changie_release_shape_test.go, 05-MUTATION-LOG.md); commits `219303c7`, `106e6cd6`, `c4a2c122`, `c4c19ade` and `a7a8240c` are ancestors of HEAD; the plan's acceptance criteria and verify commands were re-run and pass.

---
*Phase: 05-release-chain-release-please-retirement*
*Completed: 2026-10-14*
