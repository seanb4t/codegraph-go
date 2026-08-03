---
phase: 09-release-please-and-goreleaser
plan: 02
subsystem: infra
tags: [release-please, github-actions, gh-cli, cosign, tdd, shell-testing]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-01's mustWorkflowStepRunBlock parse-core helper (release_workflow_shape_test.go) — consumed unchanged to extract the publish step's literal shell"
provides:
  - "internal/upgrade/release_publish_step_test.go — TestPublishReleaseStepBranches, a stubbed-gh test proving all 5 D-04 create-vs-upload publish-step behaviors against the shipped shell, not the YAML text"
  - "release.yml's Publish GitHub release step made idempotent against a release-please-created Release (D-04): create-if-absent-else-upload-clobber, decided on gh release view's exit status"
  - "A new repo testing idiom — Go-hosted stubbed-binary-on-PATH tests for CI shell logic — for future CI-logic tests to follow (no prior in-repo precedent existed)"
affects: [09-03, 09-04, 09-05, 09-06, 09-07, 09-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Stubbed-gh-on-PATH Go test: a POSIX-shell stub script (Go string-template, env-var-supplied argv-log path) plus exec.Command(\"bash\", extractedScriptPath) with a PATH override — proves CI shell branching by executing the shipped bytes, never by reading YAML"
    - "Asset-presence guard via POSIX positional-parameter idiom (set -- glob; [ ! -e \"$1\" ]) to fail loudly on an unmatched glob instead of passing the literal glob word to a CLI"

key-files:
  created:
    - internal/upgrade/release_publish_step_test.go
  modified:
    - .github/workflows/release.yml

key-decisions:
  - "Chose the Go os/exec stubbed-binary shape (RESEARCH.md's option 2) over a standalone bats/shell script — matches the repo's Go-test-centric convention (internal/upgrade already hosts release_workflow_shape_test.go) and needed no new CI tooling"
  - "gh stub distinguishes release view/create/upload purely by argv[1]/argv[2] ('release' 'view'|'create'), recording every invocation's full argv to a log file whose path is passed via env var (not baked into the generated script) — keeps the stub script content identical across all 5 subtests, varying only the two exit codes baked in per stubGHDir call"
  - "Asset-presence guard implemented with the POSIX positional-parameter idiom (set -- codegraph_*; [ ! -e \"$1\" ]) rather than nullglob/shopt, since the step runs under plain sh-compatible bash without nullglob enabled and this idiom needs no shell-option changes"

patterns-established:
  - "Break-observe-restore non-vacuity proof for the extraction helper itself: stripping the step's run: block from a live copy of release.yml turns every subtest into a loud mustWorkflowStepRunBlock failure ('has no run: key'), not a silently-passing empty script — recorded verbatim below, following 09-01's precedent"

requirements-completed: []

coverage:
  - id: D1
    description: "Release exists -> gh release upload --clobber fires, gh release create never fires, and the upload argv carries neither --generate-notes nor --prerelease"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_publish_step_test.go#TestPublishReleaseStepBranches/release_exists_uploads_with_clobber"
        status: pass
    human_judgment: false
  - id: D2
    description: "Release absent + dash-suffixed (prerelease-shaped) tag -> gh release create fires with --prerelease and --generate-notes, upload never fires"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_publish_step_test.go#TestPublishReleaseStepBranches/release_absent_prerelease_tag_creates_with_prerelease_and_notes"
        status: pass
    human_judgment: false
  - id: D3
    description: "Release absent + stable tag -> gh release create fires without --prerelease"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_publish_step_test.go#TestPublishReleaseStepBranches/release_absent_stable_tag_creates_without_prerelease"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero release assets present -> step exits non-zero with a diagnostic annotation, invoking neither create nor upload"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_publish_step_test.go#TestPublishReleaseStepBranches/zero_assets_fails_loud_invokes_neither_branch"
        status: pass
    human_judgment: false
  - id: D5
    description: "Release absent + create call fails -> step exits non-zero rather than silently falling back to upload"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_publish_step_test.go#TestPublishReleaseStepBranches/release_absent_create_fails_no_upload_fallback"
        status: pass
    human_judgment: false
  - id: D6
    description: "The diff is confined to the Publish GitHub release step's body only — release.yml's LOCKED name/trigger/cosign-step identity, internal/upgrade/verify.go, and .goreleaser.yaml are byte-unchanged"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestReleaseWorkflowFileMatchesPattern (pass), #TestReleaseWorkflowTriggerIsTagPushOnly (pass)"
        status: pass
      - kind: other
        ref: "git diff -U0 -- .github/workflows/release.yml (hunks only inside Publish GitHub release step); git diff -- internal/upgrade/verify.go .goreleaser.yaml (empty)"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-28
status: complete
---

# Phase 9 Plan 2: Idempotent release.yml publish step (D-04) Summary

**Made release.yml's asset-publish step idempotent against a release-please-created Release — create-if-absent-else-upload-clobber, decided by `gh release view`'s exit status, with all 5 branches proven by executing the shipped shell against a recording `gh` stub before the workflow was ever edited.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2
- **Files modified:** 2 (1 new test file, 1 modified workflow file)

## Accomplishments
- `internal/upgrade/release_publish_step_test.go`: `TestPublishReleaseStepBranches`, 5 subtests, plus `stubGHDir`/`runExtractedStep` helpers — a new repo testing idiom (Go-hosted stubbed-binary-on-PATH) proving CI shell behavior by executing it, not reading the YAML.
- `.github/workflows/release.yml`'s `Publish GitHub release` step rewritten per D-04: asset-presence guard, then `gh release view`-decided create-vs-upload branch, leaving release-please's changelog body and prerelease flag untouched on the upload path.
- RED confirmed against the unmodified workflow (release-exists and zero-assets subtests failed as expected; create-fails already passed), then GREEN confirmed after the edit — the mandated TDD cycle, not just an assertion that it happened.
- Both 09-01 workflow-shape guards (`TestReleaseWorkflowFileMatchesPattern`, `TestReleaseWorkflowTriggerIsTagPushOnly`) still pass; `actionlint .github/workflows/release.yml` exits 0; `internal/upgrade/verify.go` and `.goreleaser.yaml` are byte-unchanged.

## Task Commits

1. **Task 1: RED — stubbed-`gh` test covering five publish-step cases** - `521bd58` (test)
2. **Task 2: GREEN — create-if-absent-else-upload-clobber in release.yml (D-04)** - `122d80c` (feat)

**Plan metadata:** (this commit, docs)

## Files Created/Modified
- `internal/upgrade/release_publish_step_test.go` - `TestPublishReleaseStepBranches` (5 subtests) + `stubGHDir`/`runExtractedStep`/`writePublishStepFixtureAssets`/`argvLineContainsAll` helpers
- `.github/workflows/release.yml` - `assemble` job's `Publish GitHub release` step body only: asset-presence guard + `gh release view`-decided create-vs-upload branch

## Decisions Made
- Picked the Go `os/exec` stubbed-binary test shape over a standalone bats/shell harness (RESEARCH.md flagged both as valid, no in-repo precedent for either) — stays consistent with `internal/upgrade`'s existing Go-test-centric package and needs no new CI tooling dependency.
- `gh` stub's argv-log path is supplied via an environment variable (`CODEGRAPH_TEST_GH_ARGV_LOG`) rather than baked into the generated script text, so the same script template serves every subtest — only the two exit codes (`viewExit`/`createExit`) vary per `stubGHDir` call.
- Asset-presence guard uses the POSIX positional-parameter idiom (`set -- codegraph_*; [ ! -e "$1" ]`) rather than `shopt -s nullglob`, matching the step's existing plain-`sh`-compatible style and needing no shell-option changes.

## Deviations from Plan

None - plan executed exactly as written. All 5 behaviors, both existing workflow-shape guards, `actionlint`, and the LOCKED-file zero-diff checks were satisfied without requiring an architectural change or an out-of-scope fix.

## Non-Vacuity: Break-Observe-Restore (mandatory, recorded verbatim)

### RED — Task 1, against the unmodified release.yml

```
=== RUN   TestPublishReleaseStepBranches/release_exists_uploads_with_clobber
    release_publish_step_test.go:185: expected an upload invocation carrying --clobber, got recorded argv:
        release create v1.2.3 --repo seanb4t/codegraph-go --title v1.2.3 --generate-notes codegraph_v1.2.3_checksums.txt codegraph_v1.2.3_darwin_arm64 codegraph_v1.2.3_linux_amd64 codegraph_v1.2.3_linux_amd64.sigstore.json
--- FAIL: TestPublishReleaseStepBranches (0.17s)
    --- FAIL: TestPublishReleaseStepBranches/release_exists_uploads_with_clobber (0.04s)
    --- PASS: TestPublishReleaseStepBranches/release_absent_prerelease_tag_creates_with_prerelease_and_notes (0.03s)
    --- PASS: TestPublishReleaseStepBranches/release_absent_stable_tag_creates_without_prerelease (0.04s)
    --- FAIL: TestPublishReleaseStepBranches/zero_assets_fails_loud_invokes_neither_branch (0.03s)
        release_publish_step_test.go:242: exit code = 0, want non-zero for a zero-asset working directory
            recorded argv:
            release create v2.0.0 --repo seanb4t/codegraph-go --title v2.0.0 --generate-notes codegraph_*
    --- PASS: TestPublishReleaseStepBranches/release_absent_create_fails_no_upload_fallback (0.03s)
FAIL
```
Matches the plan's exact prediction: release-exists fails (today's step always creates), zero-assets fails (today's step has no guard), create-fails already passes (today's step already exits non-zero when `gh release create` itself fails, since `set -euo pipefail` propagates that).

### Extraction-guard non-vacuity (Task 1 acceptance criterion)

Temporarily stripped the `Publish GitHub release` step's entire `run:` block from a live copy of `release.yml` (script-removed lines 271–285) and re-ran the suite:

```
=== RUN   TestPublishReleaseStepBranches/release_exists_uploads_with_clobber
    release_publish_step_test.go:180: mustWorkflowStepRunBlock("Publish GitHub release"): parseWorkflowStepRunBlock: step "Publish GitHub release" has no run: key
--- FAIL: TestPublishReleaseStepBranches (0.00s)
    --- FAIL: TestPublishReleaseStepBranches/release_exists_uploads_with_clobber (0.00s)
    --- FAIL: TestPublishReleaseStepBranches/release_absent_prerelease_tag_creates_with_prerelease_and_notes (0.00s)
    --- FAIL: TestPublishReleaseStepBranches/release_absent_stable_tag_creates_without_prerelease (0.00s)
    --- FAIL: TestPublishReleaseStepBranches/zero_assets_fails_loud_invokes_neither_branch (0.00s)
    --- FAIL: TestPublishReleaseStepBranches/release_absent_create_fails_no_upload_fallback (0.00s)
FAIL
```
All five subtests failed loudly via `mustWorkflowStepRunBlock`'s `t.Fatalf` — never a silently-passing empty script (the exact `WR-02` defect class this plan's `<critical_constraints>` names). Restored `release.yml` from the pre-mutation backup; `diff` confirmed byte-identical restoration; `git diff --stat -- .github/workflows/release.yml` was empty before Task 2's real edit began.

### GREEN — Task 2, against the edited release.yml

```
=== RUN   TestPublishReleaseStepBranches/zero_assets_fails_loud_invokes_neither_branch
    release_publish_step_test.go:240: extracted step output:
        ::error::no release assets found matching codegraph_* in /private/var/.../TestPublishReleaseStepBrancheszero_assets_fails_loud_invokes_ne.../001
--- PASS: TestPublishReleaseStepBranches (0.17s)
    --- PASS: TestPublishReleaseStepBranches/release_exists_uploads_with_clobber (0.04s)
    --- PASS: TestPublishReleaseStepBranches/release_absent_prerelease_tag_creates_with_prerelease_and_notes (0.03s)
    --- PASS: TestPublishReleaseStepBranches/release_absent_stable_tag_creates_without_prerelease (0.04s)
    --- PASS: TestPublishReleaseStepBranches/zero_assets_fails_loud_invokes_neither_branch (0.01s)
    --- PASS: TestPublishReleaseStepBranches/release_absent_create_fails_no_upload_fallback (0.04s)
PASS
```

Companion checks, all green after the edit:
```
$ go test ./internal/upgrade/ -run 'TestPublishReleaseStepBranches|TestReleaseWorkflow' -count=1 -v
--- PASS: TestPublishReleaseStepBranches (+ 5 subtests)
--- PASS: TestReleaseWorkflowFileMatchesPattern (+ accept/reject_renamed_workflow/reject_branch_ref)
--- PASS: TestReleaseWorkflowTriggerIsTagPushOnly
--- PASS: TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo
--- PASS: TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef
PASS

$ actionlint .github/workflows/release.yml
(exit 0, no output)

$ git diff -- internal/upgrade/verify.go .goreleaser.yaml
(empty)

$ git diff -U0 -- .github/workflows/release.yml
# both hunks confined to the Publish GitHub release step's leading comment
# and its run: body — no hunk touches on:, name:, the build matrix, the
# cosign step, the syft step, or the provenance job
```

## Issues Encountered
None.

## User Setup Required
None for this plan.

## Next Phase Readiness
- D-04's highest-risk edit is complete and proven both ways (create and upload), with the empty-asset and create-failure edge cases closing the loop the plan's `<must_haves>` demanded.
- `release.yml`'s LOCKED identity (name, trigger, cosign step location) is confirmed unaffected — 09-01's two drift guards and this plan's own diff-scope check both pass.
- No blockers for 09-03 onward. 09-05's GitHub App provisioning remains the phase's one hard external dependency before a real App-token-authored tag push can exercise this step in production.

## Self-Check: PASSED

`internal/upgrade/release_publish_step_test.go` confirmed present on disk; `.github/workflows/release.yml` confirmed modified only in the `Publish GitHub release` step region; commit hashes `521bd58` and `122d80c` confirmed present in `git log --oneline --all`.

---
*Phase: 09-release-please-and-goreleaser*
*Completed: 2026-07-28*
