---
phase: 05-release-chain-release-please-retirement
plan: 02
subsystem: infra
tags: [changie, github-actions, github-app, taskfile, release, tag, guard-tests]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "05-01: release-pr.yml with the pretag-gate job to copy, release:batch, the releaseChain* test helpers, 05-MUTATION-LOG.md"
provides:
  - ".github/workflows/release-tag.yml: push to main that changes CHANGELOG.md, pretag-gate then a tag job that mints the App token, checks out with it and runs task release:tag"
  - "Taskfile release:tag: tags changie latest, skips with success when the tag exists on origin, pushes only that ref, refuses to run outside GitHub Actions (C8)"
  - "TestReleaseTagWorkflowShape, TestReleaseTagTargetShape, TestTagCreationHasSingleAppTokenSite, TestTagCreationScanFlagsPlantedSites"
  - "scanTagCreationSites and validateTagCreationScan: a GRD-16 scan over every workflow run body, local action step and executed Taskfile command"
  - "REL-11 and ROADMAP Phase 5 SC1 amended to the CHANGELOG.md path trigger (D-06)"
  - "05-MUTATION-LOG.md: 05-02 RED/GREEN transcripts, the release:tag rehearsal, Families 05-02/a-i"
affects: [05-03, 05-05, 05-07, phase-06-live-run]

actuals:
  tokens: 18763
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "a Taskfile target is a human-runnable entry point, so the tag-authority scan covers every executed Taskfile command, not only the targets workflows reach"
    - "scanner discrimination is held by named subtests over literal sources, each seen RED against a mutation of the scanner itself"
    - "a live-effect target rehearsed against a scratch clone with a bare local origin, behind an origin guard, with the real repository's tag list and refs hashed before and after"

key-files:
  created:
    - .github/workflows/release-tag.yml
  modified:
    - Taskfile.yml
    - internal/upgrade/changie_release_shape_test.go
    - internal/upgrade/taskfile_shape_test.go
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md

key-decisions:
  - "C8 (new custom logic): release:tag carries a precondition that GITHUB_ACTIONS is true. Forcing constraint: D-06R's standing no-human-git-tag rule. Without it task release:tag on a laptop would push a tag to origin with the maintainer's own credentials"
  - "Lightweight tag (git tag without -a): matches every historical tag and needs no committer identity in CI; post-release-verify resolves the peeled commit either way"
  - "No version-shape validation and no notes-file check in release:tag (D-01): changie latest derives the tag from .changes/v*.md filenames and set -euo pipefail fails on a changie error"
  - "GRD-16 scan lives in changie_release_shape_test.go; tag_authority_test.go is untouched (TestGsdTagCreationIsDisabled passes unchanged)"
  - "Stated scan boundary: a script invoked by path is not opened, a non-sh Taskfile var value is not scanned, and a backslash-continued command is read line by line (T-05-10, accepted)"

patterns-established:
  - "tagCreation prefix for the GRD-16 scan identifiers in package upgrade"
  - "ls-remote status is captured and only exit 2 means absent; any other status fails (T-05-11)"

requirements-completed: [REL-11, GRD-16]

coverage:
  - id: D1
    description: "release-tag.yml: CHANGELOG.md path trigger on main, contents: read, group release-tag with cancel-in-progress false, pretag-gate decoded equal to release-pr.yml's, tag job with App-token mint before the checkout whose token is the app-token output, three SHA pins, no default-token or persist-credentials line"
    requirement: REL-11
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleaseTagWorkflowShape"
        status: pass
      - kind: other
        ref: "task lint:actions"
        status: pass
    human_judgment: false
  - id: D2
    description: "Taskfile release:tag: silent, GITHUB_ACTIONS precondition, ordered tokens (changie latest, ls-remote, 0/2/* arms, git tag, single-ref push), one changie call through GO_TOOL_CHANGIE"
    requirement: REL-11
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleaseTagTargetShape"
        status: pass
      - kind: integration
        ref: "release:tag in a scratch clone with a bare local origin: pushed, already exists, ls-remote error, precondition refusal (05-MUTATION-LOG.md)"
        status: pass
    human_judgment: false
  - id: D3
    description: "GRD-16: exactly one tag-creating unit (task:release:tag) across all workflow run bodies, local action steps and executed Taskfile commands; invoked only from release-tag.yml/tag/Tag the release; its push names refs/tags/ and no widening flag; the tag job's checkout credential is the app-token output"
    requirement: GRD-16
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestTagCreationHasSingleAppTokenSite"
        status: pass
      - kind: unit
        ref: "internal/upgrade/tag_authority_test.go#TestGsdTagCreationIsDisabled"
        status: pass
    human_judgment: false
  - id: D4
    description: "The scanner rejects a planted second tag push, git tag, gh release create, second release:tag invocation and a widened release:tag push, and ignores a commented git tag and a desc naming git tag"
    requirement: GRD-16
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestTagCreationScanFlagsPlantedSites"
        status: pass
      - kind: other
        ref: "Families 05-02/a-i in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D5
    description: "Shape-test registry: inScopeJobs and inScopeWorkflowFiles hold release-tag.yml; check:cross desc names the pretag-gate jobs of release-pr.yml and release-tag.yml (GRD-15, registry only)"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask, TestWorkflowFilePopulationMatchesDisk, TestInScopeJobsPopulationMatchesDisk"
        status: pass
    human_judgment: false
  - id: D6
    description: "REL-11 and ROADMAP Phase 5 SC1 name the CHANGELOG.md path trigger and the skip-if-exists rule, each marked amended by D-06, heading counts unchanged"
    requirement: REL-11
    verification:
      - kind: other
        ref: "05-02-PLAN.md Task 2 verify (rg over REQUIREMENTS.md, ROADMAP.md and heading-count equality against 5a8603f5)"
        status: pass
    human_judgment: false
  - id: D7
    description: "A real release-PR merge fires the CHANGELOG.md path filter, the App-token tag push starts release.yml, and the milestone merge is a no-op"
    requirement: REL-11
    verification: []
    human_judgment: true
    rationale: "Live GitHub behavior (path filter on a squash-merge push, an App-authored tag push starting release.yml) needs the App secrets and a real push to main; it is Phase 6. This plan shows only the repository's own wiring and release:tag's four branches against a local bare origin."

duration: 6min
completed: 2026-10-14
status: complete
plan_head_before: 187cc9588643ddb6f73e7c6785b36d65bf6bee46
plan_head_after: d9030c5b850b040bc844379b153538c120e99202
commits: 6
---

# Phase 5 Plan 02: Tag half of the changie release chain Summary

**A push to main that changes CHANGELOG.md now runs pretag-gate and then `task release:tag`, which pushes tag `changie latest` with the GitHub App token (or skips with success when the tag exists), and a scan over every workflow run body, local action step and executed Taskfile command proves release:tag is the repository's only tag-creating command, with each guard seen RED against a planted second tag push or a mutated scanner.**

This plan changes CI and release tooling only. No codegraph user-visible behavior changes. Nothing was pushed to the real remote, no tag was created in this repository, no PR was opened.

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-14T21:27:02Z
- **Completed:** 2026-10-14T21:33:28Z (last task commit)
- **Tasks:** 3
- **Files modified:** 7 (1 created, 6 modified)

## Accomplishments

- `release-tag.yml` (`name: release-tag`, `on: push: branches: [main], paths: [CHANGELOG.md]`, `contents: read`, concurrency group `release-tag` with `cancel-in-progress: false`): job `pretag-gate` is the same job as release-pr.yml's (decoded equal), and job `tag` needs it, mints the App token (id `app-token`), checks out with `token: ${{ steps.app-token.outputs.token }}` (the credential `git push` uses, `persist-credentials` untouched), sets up Go, installs Task and runs `task release:tag`.
- `task release:tag` (silent): precondition `GITHUB_ACTIONS` is `true`, tag from `changie latest`, `git ls-remote --exit-code --tags origin` with its status captured, a `case` (0 prints `already exists on origin` and exits 0, 2 continues, anything else prints `::error::release:tag: git ls-remote exited` and exits 1), then `git tag` and `git push origin "refs/tags/${tag}"`.
- GRD-16 scan (`scanTagCreationSites`, `validateTagCreationScan`): decodes workflow files with `parseWorkflowJobsAllSteps`, local actions' `runs.steps[].run`, and the Taskfile as a `yaml.Node` tree (every `cmds` string, `cmd:`, `defer`, `sh:`; `task:` references are invocations), drops full-line comments, never reads `desc:` or `msg:`. Exactly one unit holds a tag-creating command, `task:release:tag`, invoked only from `release-tag.yml/tag/Tag the release`.
- Population counts logged by `TestTagCreationHasSingleAppTokenSite` at GREEN: 18 workflow files (18 on disk), 85 run bodies, 1 local action, 72 Taskfile targets, 244 executed Taskfile scalars, 2 tag-creating lines (`git tag`, `git push`) in 1 unit, `release:tag` invoked from `["release-tag.yml/tag/Tag the release"]`. At RED the same scan logged 17 files, 83 run bodies, 71 targets, 241 scalars and 0 tag-creating lines.
- The rehearsal ran `release:tag` for real against a scratch clone with a bare local origin: all four branches behaved as written, and the real repository's tags are unchanged.
- REL-11 and ROADMAP Phase 5 SC1 now name changie's documented release trigger and the skip-if-exists rule, each marked amended by D-06, with unchanged heading counts.
- Families 05-02/a to 05-02/i are recorded; every guard went RED against a confirmed-applied, byte-cleanly-reverted mutation.

## TDD Gate Compliance

RED evidence is the `test(05-02)` commit that landed before the implementation plus the pasted `--- FAIL:` transcript. `gsd-tools check tdd-red-evidence` was not used: it parses TAP only and always reports INVALID_RED for `go test` (rule x1cjy9vyhq).

| Gate | Commit | Subject |
|------|--------|---------|
| RED | `a7908d2a` | test(05-02): expect an App-token tag workflow and a single tag-creating site (REL-11, GRD-16) |
| GREEN | `179aacb2` | ci(05-02): tag the release through the GitHub App when a release PR merges (REL-11) |
| follow-up | `59fc881d` | test(05-02): skip the block-scalar marker when reading release:tag's body |

The RED commit holds exactly `internal/upgrade/changie_release_shape_test.go` and `internal/upgrade/taskfile_shape_test.go`; the GREEN commit holds exactly `Taskfile.yml` and `.github/workflows/release-tag.yml`. Over `release-tag.yml` and `changie_release_shape_test.go` since `5a8603f5`, the 05-02 commits are ordered test, ci, test; the first two are test(05-02) then ci(05-02). `tag_authority_test.go`, `release.yml` and `verify.go` are byte-identical to `5a8603f5`.

`go test ./internal/upgrade/ -count=1 -v` at `a7908d2a` (assertion and read failures on the missing subject, no build error; exactly these six fail, the other tests including `TestTagCreationScanFlagsPlantedSites` and `TestGsdTagCreationIsDisabled` pass):

```
    changie_release_shape_test.go:601: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestReleaseTagWorkflowShape (0.00s)
    changie_release_shape_test.go:774: ../../Taskfile.yml: no top-level "release:tag" task
--- FAIL: TestReleaseTagTargetShape (0.00s)
    changie_release_shape_test.go:1333: found 0 tag-creating units map[], want exactly 1 (task:release:tag)
    changie_release_shape_test.go:1337: GRD-16: found zero tag-creating commands in task:release:tag, want exactly that one unit; task release:tag is invoked from [], want exactly ["release-tag.yml/tag/Tag the release"]
    changie_release_shape_test.go:1342: release-tag.yml is not among the 17 workflow files on disk
--- FAIL: TestTagCreationHasSingleAppTokenSite (0.02s)
    taskfile_shape_test.go:1400: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestWorkflowRunBodiesInvokeTask (0.01s)
    taskfile_shape_test.go:1613: inScopeWorkflowFiles names "release-tag.yml", which no longer exists on disk
--- FAIL: TestWorkflowFilePopulationMatchesDisk (0.00s)
    taskfile_shape_test.go:1686: read ../../.github/workflows/release-tag.yml: open ../../.github/workflows/release-tag.yml: no such file or directory
--- FAIL: TestInScopeJobsPopulationMatchesDisk (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.515s
```

`TestTagCreationScanFlagsPlantedSites` passes at RED by design (literal inputs only). Its RED evidence is Families 05-02/g, h and i, which mutate the scanner (see below).

GREEN (`59fc881d` content): the eight named tests pass with 7 passing scanner subtests plus the baseline (8 subtests), `task lint:actions` exits 0, the whole `internal/upgrade` package passes.

## Rehearsal of release:tag (scratch bare origin)

Evidence, not a committed test (rule eyva2svrcz). `T` is `/private/tmp/claude-501/-Volumes-Code-github-com-seanb4t-codegraph-go/d5fbaa99-ed39-4562-985b-aad98b49b210/scratchpad/exec-05-02/rehearsal`: a bare `origin.git` and a `--no-tags` clone of this checkout whose origin was set to `$T/origin.git`. Every run was `(cd "$T/clone" && GITHUB_ACTIONS=true task release:tag)` (Run 4 without the assignment); the calling shell had `GITHUB_ACTIONS` unset. The tag is `changie latest` in the clone, `v0.14.1`.

The four origin-guard lines (fetch and push URLs expanded by `git remote get-url`, each under `$T/`, none naming github.com):

```
origin guard: fetch=<T>/origin.git push=<T>/origin.git ok        (Run 1)
origin guard: fetch=<T>/origin.git push=<T>/origin.git ok        (Run 2)
origin guard: fetch=<T>/missing.git push=<T>/missing.git ok      (Run 3)
origin guard: fetch=<T>/origin.git push=<T>/origin.git ok        (Run 4)
```

(`<T>` stands for the path above; the log holds the literal lines.)

```
Run 1  GITHUB_ACTIONS=true task release:tag   (tag absent)
To <T>/origin.git
 * [new tag]           v0.14.1 -> v0.14.1
release:tag: pushed v0.14.1 at e6850a48813e7c9f8ad6ca372bb92e7a750f08d4
exit=0     origin tags: v0.14.1

Run 2  same command   (tag present on origin)
release:tag: v0.14.1 already exists on origin — nothing to do
exit=0

Run 3  origin set to <T>/missing.git
fatal: '<T>/missing.git' does not appear to be a git repository
fatal: Could not read from remote repository.
::error::release:tag: git ls-remote exited 128; refusing to guess whether v0.14.1 exists on origin
task: Failed to run task "release:tag": exit status 1
exit=201

Run 4  task release:tag with GITHUB_ACTIONS unset
task: release:tag pushes a tag to origin and runs only in release-tag.yml (D-06R: no human git tag). Rehearse it against a scratch clone whose origin is a scratch bare repository, with GITHUB_ACTIONS=true set for that one command.
task: Failed to run task "release:tag": task: precondition not met
exit=201
```

The two real-repo lines, captured before and after the four runs:

```
real-repo tag --list 'v0.15*' before=[] after=[] equal
real-repo refs/tags sha256 before=6d900e5f187b3d9c27c888e29b8e62b624fd5b9d164be219af6730a41f2bc8fd after=6d900e5f187b3d9c27c888e29b8e62b624fd5b9d164be219af6730a41f2bc8fd equal
```

`git tag -l | sha256sum` of this repository is `34f371adaa8cf683d6c62bd2167023a31716aa70b4faed7a119f9eaa3386e1d2` before and after the whole plan. `git remote -v` is unchanged (https github.com origin). `gh api repos/seanb4t/codegraph-go/git/matching-refs/tags/v0.15 --jq length` (read-only) returned 0. The scratch directory was removed.

## Mutation families

All in `05-MUTATION-LOG.md` (`## Family 05-02/a` to `## Family 05-02/i`), each with a pre-mutation gate, applied-confirmation hunk, verbatim RED transcript, byte-clean revert proof and GREEN re-run.

| Family | Mutation | RED |
|---|---|---|
| a | tag job checkout token becomes the default token context | TestReleaseTagWorkflowShape, TestTagCreationHasSingleAppTokenSite |
| b | planted `git push origin --tags` step in ci.yml `test` | TestTagCreationHasSingleAppTokenSite names `workflow:ci.yml/test/planted`; TestWorkflowRunBodiesInvokeTask also RED |
| c | planted `git tag v9.9.9` in check:cross | TestTagCreationHasSingleAppTokenSite names `task:check:cross` |
| d | `paths: [CHANGELOG.md]` deleted | TestReleaseTagWorkflowShape |
| e | `0)` skip-if-exists arm deleted | TestReleaseTagTargetShape |
| f | push widened to `git push origin --tags` | TestReleaseTagTargetShape, TestTagCreationHasSingleAppTokenSite |
| g | scanner git push pattern disabled | `reject_second_workflow_push_tags` (baseline and both `ignore_*` subtests also RED: release:tag then has no push line to check) |
| h | scanner git tag and gh release create patterns disabled | `reject_second_taskfile_git_tag`, `reject_gh_release_create`; baseline and TestTagCreationHasSingleAppTokenSite stay green |
| i | validator accepts any non-empty release:tag invocation list and `--tags` leaves the forbidden push flags | `reject_second_release_tag_invocation`, `reject_release_tag_push_widened`; baseline and TestTagCreationHasSingleAppTokenSite stay green |

## Task Commits

1. **Task 1: tag expansion, RED then GREEN** - `a7908d2a` (test), `179aacb2` (ci), `59fc881d` (test, block-scalar-marker fix), `e6850a48` (docs, RED/GREEN transcripts)
2. **Task 2: rehearsal and D-06 amendment** - `67df2d1d` (docs)
3. **Task 3: mutation families 05-02/a-i** - `d9030c5b` (docs)

**Plan metadata:** the `docs(05-02): complete tag-half plan` commit that follows this file.

## Files Created/Modified

- `.github/workflows/release-tag.yml` - the tag workflow
- `Taskfile.yml` - `release:tag` target; `check:cross` desc names the two pretag-gate jobs
- `internal/upgrade/changie_release_shape_test.go` - four tests, the GRD-16 scanner and validator, `tagCreation*` and `releaseChainTag*` identifiers
- `internal/upgrade/taskfile_shape_test.go` - `inScopeJobs` entries for release-tag.yml's two jobs, `inScopeWorkflowFiles` element
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` - D-06 value edits to REL-11 and Phase 5 SC1
- `.planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md` - transcripts, rehearsal, Families a-i

## Decisions Made

- **C8, new custom logic, named:** the `GITHUB_ACTIONS` precondition on `release:tag`. Forcing constraint: D-06R's standing "no human `git tag`" rule (REQUIREMENTS standing rules). One precondition line; run 4 shows the refusal.
- Lightweight tag, no version-shape or notes-file validation in `release:tag` (D-01: not forced by a locked constraint).
- The scan covers every executed Taskfile command rather than resolving which targets workflows reach: a Taskfile target is itself a human-runnable entry point, so this is strictly stronger.
- GRD-15 is partial: the registry entries for release-tag.yml and the check:cross desc only. The release-please entries stay until 05-07 deletes their files.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestReleaseTagTargetShape read the block-scalar marker as the first shell line**
- **Found during:** Task 1 (GREEN run)
- **Issue:** the `cmds:` text handed to the test begins with the `- |` line that opens the block scalar, so the `set -euo pipefail` first-line check failed. RED could not show it because release:tag was absent.
- **Fix:** skip the `- |` line when building the body lines.
- **Files modified:** `internal/upgrade/changie_release_shape_test.go`
- **Verification:** the eight named tests pass.
- **Committed in:** `59fc881d`, a separate `test(05-02)` commit after the ci commit, because the plan requires the GREEN commit to hold exactly Taskfile.yml and release-tag.yml (the 05-01 precedent). At `179aacb2` alone TestReleaseTagTargetShape fails on that one assertion; it passes from `59fc881d` on. The order check (first two commits are test then ci) still holds.

**2. [Environment note] A stale plan commit ledger from an earlier milestone**
- **Found during:** commit-ledger setup
- **Issue:** `.git/gsd-plan-head-before-05-02` already existed and held `44fafa27`, a commit that is not an ancestor of this branch (a phase-5 plan 02 from a previous milestone). Reusing it would have made `commits:` count the wrong range.
- **Fix:** overwrote it with this plan's real base, `187cc958`. `commits: 6` is measured from that base.

---

**Total deviations:** 2 (1 Rule 1 bug, 1 environment note)
**Impact on plan:** None on scope. The test fix is a two-line correction; the ledger overwrite only affects measurement.

## Issues Encountered

None beyond the deviations. `internal/daemon` TestSoak was not run (package-scoped runs only). `task check:fragment-policy` reports `executed 36 of 36 defined test methods (floor 36)` and `OK`.

## Known Stubs

None.

## Threat Flags

None. The only new trust surface (the App token as push credential, a tag-pushing Taskfile target) is the plan's own T-05-06, T-05-07, T-05-08 and T-05-11, each mitigated and tested above.

## User Setup Required

None - no external service configuration required in this plan. The live run needs the existing `APP_ID` and `APP_PRIVATE_KEY` secrets and is Phase 6's.

## Next Phase Readiness

- 05-03 can extend `changie_release_shape_test.go` with the release:goreleaser and release:dry-run guards; the GRD-16 scan will cover any command it adds.
- 05-05 rewords only `tag_authority_test.go`'s prose (D-09); GRD-16 requirement text says that file stays unchanged here, and it does.
- Requirements: REL-11 and GRD-16 are complete. GRD-15 is also declared by 05-03, 05-04, 05-05 and 05-07, so it stays open until they finish.
- Risk named in the plan, not planned around: release-please stays live on `main` until the milestone merges, so a release cut on `main` in the meantime needs a verbatim byte-slice seed `.changes/vX.Y.Z.md` in the next main-to-milestone merge.
- Unchanged from `5a8603f5`: `release.yml`, `internal/upgrade/verify.go`, `internal/upgrade/tag_authority_test.go`, CHANGELOG.md, every `.changes/v*.md`.

## Self-Check: PASSED

Created files exist (release-tag.yml, changie_release_shape_test.go, 05-MUTATION-LOG.md); commits `a7908d2a`, `179aacb2`, `59fc881d`, `e6850a48`, `67df2d1d` and `d9030c5b` are ancestors of HEAD; the plan's acceptance criteria and all verify commands were re-run and pass; the main checkout carries no change under Taskfile.yml, .github or internal; no scratch worktree remains.

---
*Phase: 05-release-chain-release-please-retirement*
*Completed: 2026-10-14*
