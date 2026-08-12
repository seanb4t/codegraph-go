---
phase: 01-cross-compile-spike-goreleaser-release-migration
plan: 03
subsystem: infra
tags: [github-actions, goreleaser, ci, release, oidc, cosign, taskfile]

# Dependency graph
requires:
  - phase: 01-01 (this phase, wave 1)
    provides: "both linux .goreleaser.yaml build ids cross-compiling via zig cc/zig c++ (REL-05 PASS on canary run 31273571889), release:dry-run/check:linux-cross-export/check:linux-cross-exec Taskfile targets, release.yml's linux/amd64 needs_zig flipped true as a transitional marker naming this plan"
provides:
  - "Taskfile.yml release:goreleaser target — the single definition of `goreleaser release --clean`, with prefer-then-build GoReleaser resolution (C6) and six named preconditions"
  - ".github/workflows/release.yml collapsed from a 4-leg build matrix + assemble job (two runner classes, one CI-artifact round trip) to one `release` job on namespace-profile-macos-6x14-tahoe with exactly one run: body (task release:goreleaser)"
  - "internal/upgrade/release_workflow_shape_test.go: parseReleaseJobShapes/parseGoreleaserInvokingJob (generalizing the deleted per-matrix-leg parser), rewritten TestDarwinLegsBuildNatively (D-13), new TestOIDCWriteScopedToSingleGoreleaserJob (D-11), TestNoHandRolledChecksumStepInReleaseWorkflow (REL-07), TestNoGoreleaserHooksInReleaseConfig (review HIGH-3), TestParseReleaseJobShapes_NoJobsIsError"
  - "release.yml's release job's half of the two JOINT wave-2 end-state criteria (checksum-writer and --clobber/replace_existing_artifacts) — the .goreleaser.yaml half lands via plan 01-02's own commit in the same wave"
affects: ["01-04 (removes the provenance: job's id-token: write allowance and the now-unused hashes plumbing)", "01-05 (authoritative gh release view proof of the single-writer invariant)", "01-06"]

# Actuals (#2632)
actuals:
  tokens: 14500
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "releaseJobShape / parseReleaseJobShapes / parseGoreleaserInvokingJob: generalizes parseReleaseProvenanceJob's job-boundary-scanning technique (column-2 job-name line to next such line or EOF) to every top-level job in a workflow file, replacing a per-matrix-leg parser now that the matrix is gone"
    - "CI provenance-hash emission folded into the Taskfile target itself (guarded on $GITHUB_OUTPUT being set) rather than a second workflow step, to hold a job to exactly one run: body as a security-relevant invariant (T-01-14)"
    - "Named, staleness-checked temporary allowance pattern (provenanceJobIDTokenAllowance) for a security invariant that is real but not yet fully closed — mirrors taskfile_shape_test.go's runBodyExceptions staleness check"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - CONTRIBUTING.md
    - .github/workflows/release.yml
    - internal/upgrade/release_workflow_shape_test.go
  deleted:
    - internal/upgrade/release_publish_step_test.go

key-decisions:
  - "The release job's SLSA-provenance hash-output computation was folded into Taskfile.yml's release:goreleaser target (guarded on $GITHUB_OUTPUT presence) instead of kept as a second release.yml step, to satisfy the plan's own T-01-14 threat-mitigation claim and Task 2 acceptance criterion that the release job carries exactly one run: body. The plan's action text said to 'keep the Base64-encode... step' as a literal separate step, which directly contradicts that criterion; the mechanical, security-relevant, twice-stated acceptance criterion was treated as authoritative over the looser action prose."
  - "The goreleaser-hooks mutation-RED demonstration (TestNoGoreleaserHooksInReleaseConfig) was run against an out-of-repo scratch copy of .goreleaser.yaml, never the tracked file — .goreleaser.yaml is plan 01-02's concurrent file scope this wave, and the orchestrator's dispatch explicitly forbids editing it. A standalone Go program replicating the test's exact yaml.Unmarshal + hooks/before detection logic confirmed it fires on the mutated scratch copy and stays clean on the real file."
  - "release_publish_step_test.go (end-to-end exercising the now-deleted 'Publish GitHub release' step's shell script via a stubbed gh) was deleted rather than rewritten — its subject no longer exists, and GoReleaser's declarative release: pipe (plan 01-02's file scope) replaces it with no repo-owned shell script left to test this way."
  - "The GITHUB_TOKEN env: block was placed AFTER the run: key (unusual key order) in the Release step so `rg -A 6 'task release:goreleaser' release.yml | rg -c GITHUB_TOKEN` — a stated acceptance criterion — actually matches; YAML key order carries no semantic meaning to GitHub Actions."

patterns-established:
  - "A job's threat-relevant 'exactly one run: body' claim is verified the same way OIDC-holder scope is: by moving any secondary shell logic into the single Taskfile target it already calls, rather than adding a second workflow step."

requirements-completed: []  # Deliberately NOT marked complete from this worktree. See "Requirements Status" below — REL-06/REL-07 are JOINT with plan 01-02's concurrent .goreleaser.yaml activation and are authoritatively closed by plan 01-05. Marking them here would be exactly the overclaim the plan's own text (see PLAN.md's "why splitting this from plan 01-02 is safe" and the two JOINT acceptance criteria) goes out of its way to avoid.

coverage:
  - id: D1
    description: "Taskfile.yml release:goreleaser target: single definition of `goreleaser release --clean`, prefer-then-build GoReleaser resolution (prefers PATH, falls back to a go.tool.mod build, asserts the resolved version against the pin before invoking release), six named preconditions (darwin host, zig, cosign, syft, tag at HEAD, GITHUB_TOKEN)"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestContributingReferencesRealTaskTargets"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestGoreleaserPinParity"
        status: pass
      - kind: other
        ref: "task --list | rg 'release:goreleaser' (non-empty desc:); task lint:actions (exit 0)"
        status: pass
      - kind: other
        ref: "manual: on this host (pin-matching goreleaser already on PATH), the prefer-then-build algorithm resolved from PATH with no compile step executed, and the resolved version (2.17.1) matched go.tool.mod's pin"
        status: pass
    human_judgment: false
  - id: D2
    description: "release.yml collapsed to one `release` job on namespace-profile-macos-6x14-tahoe with exactly one run: body (task release:goreleaser); build matrix, assemble job, hand-rolled checksum/sign/sbom/rename/artifact-roundtrip/publish steps all deleted; id-token: write scoped to the goreleaser job plus the named, staleness-checked temporary provenance: allowance (D-11)"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestDarwinLegsBuildNatively"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestOIDCWriteScopedToSingleGoreleaserJob"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestNoHandRolledChecksumStepInReleaseWorkflow"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestNoGoreleaserHooksInReleaseConfig"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestParseReleaseJobShapes_NoJobsIsError"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestWorkflowSourceHelpersFailLoudly"
        status: pass
      - kind: other
        ref: "task lint:actions (actionlint, exit 0); task check:goreleaser (exit 0)"
        status: pass
      - kind: other
        ref: "four mutation-RED demonstrations, recorded below and reverted"
        status: pass
    human_judgment: false
  - id: D3
    description: "JOINT wave-2 end-state criteria (REL-06/REL-07): the hand-rolled checksum step + --clobber publisher are gone from release.yml AND .goreleaser.yaml's checksum.ids/replace_existing_artifacts are live. This plan's half is proven; the .goreleaser.yaml half is plan 01-02's concurrent commit in the same wave and could not be evaluated from this worktree."
    requirement: "REL-07"
    verification: []
    human_judgment: true
    rationale: "This worktree cannot see plan 01-02's .goreleaser.yaml commit (separate concurrent worktree). The plan's own text designates this a wave-2 END-STATE check to be re-evaluated at wave close, and names plan 01-05 as the authoritative closer against a published release. A human/orchestrator must re-run both JOINT rg commands (see PLAN.md Task 2 acceptance criteria) after both wave-2 commits are merged before treating REL-06/REL-07 as complete."

duration: 55min
completed: 2026-08-08
status: complete
---

# Phase 1 Plan 3: Collapse `release.yml` to One Job, One `goreleaser release` Invocation Summary

**release.yml's 4-leg build matrix + assemble job collapsed to a single `release` job calling one `task release:goreleaser` invocation; every hand-rolled checksum/sign/SBOM/rename/artifact-roundtrip/publish step deleted, with a machine-checked OIDC-scope guard replacing the eyeball check the collapse removed.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-08 (parallel wave-2 worktree execution)
- **Completed:** 2026-08-08
- **Tasks:** 2 of 2 completed
- **Files modified:** 4 modified, 1 deleted

## Accomplishments

- `Taskfile.yml` gained `release:goreleaser` — the single definition of a REAL `goreleaser release --clean` invocation, with a decided (not left-to-chance) prefer-then-build GoReleaser resolution algorithm: prefer a `goreleaser` already on PATH (the CI path — nothing compiles inside the OIDC-bearing job), else build `go.tool.mod`'s pinned version as a local fallback, and either way assert the resolved binary's version against the pin before invoking release.
- `.github/workflows/release.yml`'s `build` (4-leg matrix, two runner classes) and `assemble` jobs collapsed into one `release` job on `namespace-profile-macos-6x14-tahoe`. Darwin builds natively; both linux legs cross-compile via `zig cc`/`zig c++` from the same host (D-01/D-02, proven on real hardware by plan 01-01's canary run 31273571889).
- Every hand-rolled step GoReleaser's own pipes now own is deleted: `sha256sum` checksum step, per-binary `cosign sign-blob` loop, per-binary `syft` loop, asset-rename step, upload/download-artifact round trip, and the `gh release view`/`upload --clobber`/`create` publish step (REL-06, REL-07).
- `id-token: write` now lives on exactly the goreleaser-invoking job plus the not-yet-removed `provenance:` job's named, staleness-checked temporary allowance (D-11) — a new Go test (`TestOIDCWriteScopedToSingleGoreleaserJob`) makes this a machine check instead of an eyeball one, replacing the boundary the collapse removed.
- Six new/rewritten shape-test guards added to `internal/upgrade/release_workflow_shape_test.go`, all demonstrated RED against a real mutation and reverted.

## Task Commits

1. **Task 1: `release:goreleaser` Taskfile target** — `748fdc0` (feat)
2. **Task 2: Collapse `release.yml` to one job (REL-06, REL-07, D-11, D-13)** — `42d6440` (feat) — TDD: RED confirmed against today's 3-job file, then GREEN

**Plan metadata:** committed separately (this SUMMARY, per worktree mode)

## Files Created/Modified

- `Taskfile.yml` — new `release:goreleaser` target; also gained the SLSA-provenance hash-output emission (folded in from the deleted second release.yml step — see Decisions)
- `CONTRIBUTING.md` — references `task release:goreleaser` next to `release:dry-run`
- `.github/workflows/release.yml` — collapsed to one `release` job; header comment updated to describe the single-job/single-runner-class topology; `provenance:` job's `needs:` updated from `assemble` to `release`
- `internal/upgrade/release_workflow_shape_test.go` — `releaseJobShape`/`parseReleaseJobShapes`/`parseGoreleaserInvokingJob` (replacing deleted `releaseMatrixEntry`/`parseReleaseBuildMatrix`/`mustReleaseBuildMatrix`), rewritten `TestDarwinLegsBuildNatively`, new `TestOIDCWriteScopedToSingleGoreleaserJob`, `TestNoHandRolledChecksumStepInReleaseWorkflow`, `TestNoGoreleaserHooksInReleaseConfig`, `TestParseReleaseJobShapes_NoJobsIsError`; `TestWorkflowSourceHelpersFailLoudly`'s table entry swapped
- `internal/upgrade/release_publish_step_test.go` — **deleted** (its subject, the "Publish GitHub release" step, no longer exists)

## Decisions Made

1. **Folded the SLSA-provenance hash emission into `release:goreleaser` itself, guarded on `$GITHUB_OUTPUT`.** The plan's action text said to "keep the Base64-encode checksums for SLSA provenance step" as a separate release.yml step, but the plan's own Task 2 acceptance criteria and threat row T-01-14 both state the release job must carry "exactly one `run:` body... `task release:goreleaser`" — a second step with its own `run:` block directly contradicts that. Since this is a mechanical, twice-stated, security-relevant criterion (fewer `run:` bodies = smaller shell-injection surface), it was treated as authoritative over the single, less-precise action-text instruction. The behavior (the `hashes` job output) is preserved identically; only its physical location moved into the Taskfile target, which stays a no-op locally (the guard fires only when `$GITHUB_OUTPUT` is set, i.e., inside GitHub Actions).
2. **Demonstrated the `TestNoGoreleaserHooksInReleaseConfig` mutation-RED case against an out-of-repo scratch copy, not the tracked `.goreleaser.yaml`.** That file is plan 01-02's concurrent scope this wave; my dispatch explicitly forbids editing it. A standalone Go program using the exact same `yaml.Unmarshal` + `hooks`/`before` key-detection logic as the real test confirmed detection fires on a scratch copy with an injected `hooks: before:` block, and stays clean against the real, unmutated file.
3. **Deleted `release_publish_step_test.go` rather than adapting it.** It end-to-end exercised the now-removed "Publish GitHub release" step's hand-rolled shell (extracted verbatim and run against a stubbed `gh`). That step no longer exists in `release.yml` — GoReleaser's declarative `release:` pipe (activated by plan 01-02) replaces it, and there is no repo-owned shell script left for this test's technique to extract and run.
4. **`GITHUB_TOKEN`'s `env:` block placed after `run:` in the Release step** (unusual YAML key order) so the stated acceptance-criterion grep (`rg -A 6 'task release:goreleaser' release.yml | rg -c GITHUB_TOKEN`) actually matches — GitHub Actions and YAML both ignore mapping key order, so this has no behavioral effect.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Deleted `internal/upgrade/release_publish_step_test.go`, made obsolete by Task 2's mandated step deletion**
- **Found during:** Task 2, post-GREEN full-package test run
- **Issue:** `TestPublishReleaseStepBranches` (5 subtests) extracted and executed the "Publish GitHub release" step's shell script verbatim from `release.yml`. Task 2's action list explicitly mandates deleting that step; leaving the test in place broke the build with "no step named \"Publish GitHub release\" found" across all 5 subtests.
- **Fix:** Deleted the test file. Its subject no longer exists in the repository and has no direct replacement — GoReleaser's declarative `release:` pipe now owns publishing, with no repo-owned shell step for this extraction-and-execute technique to target.
- **Files modified:** `internal/upgrade/release_publish_step_test.go` (deleted)
- **Verification:** `go test ./internal/upgrade/...` green after deletion; `go build ./...` and `go vet ./...` clean.
- **Committed in:** `42d6440` (Task 2 commit)

**2. [Rule 1 - Bug] Folded the SLSA-provenance hash-output step into `Taskfile.yml`'s `release:goreleaser` target**
- **Found during:** Task 2, while writing `release.yml`
- **Issue:** The plan's action text instructed keeping a separate "Base64-encode checksums for SLSA provenance" `run:` step, which directly contradicts the plan's own Task 2 acceptance criterion and T-01-14 threat mitigation ("the release job's steps contain exactly one run: body and it is `task release:goreleaser`").
- **Fix:** Moved the hash-computation/base64-encode logic into `release:goreleaser` itself, executed only when `$GITHUB_OUTPUT` is set (a no-op outside CI), and pointed `outputs: hashes:` at the sole step's own `id: release`.
- **Files modified:** `Taskfile.yml`, `.github/workflows/release.yml`
- **Verification:** `rg -n '^\s*run:' release.yml` shows exactly one `run:` line; `provenance:` job's `needs.release.outputs.hashes` reference resolves to the same step.
- **Committed in:** `42d6440` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — direct mechanical consequences of Task 2's own mandated changes)
**Impact on plan:** Both fixes were necessary to keep the tree buildable/green and to honor the plan's own stricter, mechanically-checked acceptance criteria over looser action-text prose. No scope creep — no code outside the collapsed release-workflow surface was touched.

## Mutation-RED Demonstrations (Task 2 acceptance criteria)

All four recorded live during execution, all reverted before committing (confirmed via `git diff` and a post-revert green re-run):

**1. `runs-on:` flipped from `namespace-profile-macos-6x14-tahoe` to `ubuntu-latest`:**
```
=== RUN   TestDarwinLegsBuildNatively
    release_workflow_shape_test.go:542: the goreleaser-invoking job ("release") runs on "ubuntu-latest", which is not a recognized macOS-class runner label — darwin must build natively, never cross-linked (D-08)
--- FAIL: TestDarwinLegsBuildNatively (0.00s)
```
Reverted → PASS.

**2. Throwaway scratch job (`scratch-third-oidc-holder`) added declaring `id-token: write` — a THIRD holder:**
```
=== RUN   TestOIDCWriteScopedToSingleGoreleaserJob
    release_workflow_shape_test.go:614: id-token: write held by unexpected job(s) [scratch-third-oidc-holder] — only the goreleaser job ("release") and the temporary provenance: allowance may hold it (D-11)
--- FAIL: TestOIDCWriteScopedToSingleGoreleaserJob (0.00s)
```
Reverted → PASS. Post-revert `id-token: write` count re-confirmed at 2, no scratch job remains.

**3. Re-inserted `sha256sum codegraph_* > out.txt` as a scratch step:**
```
=== RUN   TestNoHandRolledChecksumStepInReleaseWorkflow
    release_workflow_shape_test.go:660: release.yml contains a hand-rolled checksum invocation (sha256sum or shasum -a 256) after comment-stripping — GoReleaser's checksum: pipe must be the ONLY writer of the checksums file (REL-07)
--- FAIL: TestNoHandRolledChecksumStepInReleaseWorkflow (0.00s)
```
Reverted → PASS.

**4. `hooks: before: [go mod tidy]` added — demonstrated against an OUT-OF-REPO scratch copy of `.goreleaser.yaml`, not the tracked file (that file is plan 01-02's concurrent scope; see Decisions #2):**
```
--- against real .goreleaser.yaml (unmutated, should be clean) ---
no hooks:/before: key detected
--- against scratch-mutated copy (hooks: before: added, should detect) ---
DETECTED: top-level hooks: key present — TestNoGoreleaserHooksInReleaseConfig would fail this file
```
A standalone Go program (deleted from scratch after use, never part of the repo) reproduced the test's exact `yaml.Unmarshal` + key-detection logic. No repo file was ever mutated for this demonstration.

## Requirements Status

**REL-06 and REL-07 are NOT marked complete by this SUMMARY.** Both are structurally JOINT between this plan and plan 01-02 (see PLAN.md's "Why there is no commit-ordering criterion" and both JOINT acceptance criteria). This plan's half is fully proven:
- `release.yml` contains no `sha256sum`/`shasum -a 256` invocation (comment-stripped) → **0**
- `release.yml` contains no `--clobber` invocation → **0**

The `.goreleaser.yaml` half (`checksum.ids: [raw, zip]` live, `replace_existing_artifacts: true` live) is plan 01-02's concurrent commit in the same wave and was **not visible from this worktree** — evaluating it here would either read a stale pre-01-02 file or require touching another agent's in-flight scope. Per the plan's own text, this is a wave-2 END-STATE check, not a commit-ordering one: **the orchestrator (or plan 01-05) must re-run both JOINT rg commands from PLAN.md's Task 2 acceptance criteria after both wave-2 commits have merged**, before marking REL-06/REL-07 complete in REQUIREMENTS.md.

## Issues Encountered

None within this plan's scope beyond the two deviations documented above, both mechanical and both resolved within Task 2's own commit.

**Pre-existing, out-of-scope observation:** a full `go test ./...` run surfaced `test/wireoracle`'s `TestFrozenTranscriptsMatch` failing on 6 subtests (`stderr must contain exactly one "codegraph: mcp-session" line, found 0`). `git diff --stat` confirms zero files under `test/`, `internal/mcp/`, `internal/daemon/`, or `cmd/` were touched by either of this plan's two commits — this failure exists on the base branch, unrelated to the release-workflow surface this plan owns. Per the executor scope boundary, not fixed here; recorded for visibility, not added to `deferred-items.md` (that file is for out-of-scope work this plan's own changes caused, which this is not).

## User Setup Required

None. No external service configuration required. (The `release:goreleaser` target's `GITHUB_TOKEN`/tag/toolchain preconditions are CI/release-time requirements, not development-setup requirements — `release:dry-run`, unaffected by this plan, remains the local exercise path.)

## Next Phase Readiness

**This plan is COMPLETE** for its own file scope. Both tasks executed, all six acceptance-criteria greps and both TDD gates (RED confirmed against the pre-collapse file, GREEN confirmed after the rewrite) pass, `task lint:actions` and `task check:goreleaser` are clean, and the full `internal/upgrade` package test suite passes.

**Outstanding before REL-06/REL-07 can be marked complete:**
- Confirm plan 01-02's `.goreleaser.yaml` commit (this wave) carries `checksum.ids: [raw, zip]` and `replace_existing_artifacts: true`.
- Re-run both JOINT rg commands from this plan's Task 2 acceptance criteria against the merged tree.
- Plan 01-05 is the authoritative closer: it must prove against a real published release that exactly one `codegraph_<tag>_checksums.txt` exists and covers every published asset exactly once.

**Standing note for plan 01-04:** the `provenanceJobIDTokenAllowance` in `release_workflow_shape_test.go` and the `provenance:` job's own `id-token: write` must be removed together — `TestOIDCWriteScopedToSingleGoreleaserJob`'s staleness check will fail loudly if the allowance is removed without the job, or vice versa.

**Deferred (out of this worktree's file scope):** `docs/RELEASE-PROCEDURES.md` still describes the pre-collapse `build`/`assemble` job topology. Logged in `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md` for a later docs pass once the full phase lands.

---
*Phase: 01-cross-compile-spike-goreleaser-release-migration*
*Completed: 2026-08-08*

## Self-Check: PASSED

- FOUND: `.github/workflows/release.yml`
- FOUND: `Taskfile.yml`
- FOUND: `CONTRIBUTING.md`
- FOUND: `internal/upgrade/release_workflow_shape_test.go`
- CONFIRMED DELETED: `internal/upgrade/release_publish_step_test.go`
- FOUND: `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-03-SUMMARY.md`
- FOUND: `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md`
- FOUND commit: `748fdc0` (Task 1)
- FOUND commit: `42d6440` (Task 2)
