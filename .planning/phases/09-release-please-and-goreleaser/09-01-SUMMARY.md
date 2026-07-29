---
phase: 09-release-please-and-goreleaser
plan: 01
subsystem: infra
tags: [release-please, github-actions, ci, cosign, sigstore, conventional-commits]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: "release.yml's proven tag-triggered build/sign/SBOM/provenance pipeline and internal/upgrade/verify.go's LOCKED releaseWorkflowRefPattern"
provides:
  - "internal/upgrade/release_workflow_shape_test.go — non-vacuous drift guard mechanically pinning release.yml's shape to releaseWorkflowRefPattern"
  - "release-please-config.json + .release-please-manifest.json seeded to the repo's real 0.1.0 baseline"
  - ".github/workflows/release-please.yml — App-token release-please workflow with a blocking 6-target pretag-gate job"
  - "Live proof that release-please resolves this repo's real commit history end-to-end (verified against the pushed branch via --target-branch)"
affects: [09-02, 09-03, 09-04, 09-05, 09-06, 09-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-rolled parseX/mustX workflow-YAML parse-core pairs (error-returning, never a zero value on miss) — internal/upgrade test-only style, consumed unchanged by 09-02/09-03"
    - "Break-observe-restore non-vacuity proof recorded verbatim in SUMMARY for every new CI/test guard"

key-files:
  created:
    - internal/upgrade/release_workflow_shape_test.go
    - release-please-config.json
    - .release-please-manifest.json
    - .github/workflows/release-please.yml
  modified: []

key-decisions:
  - "Both third-party action SHAs re-resolved live via gh api before writing: googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 (v5.0.0) and actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 (v3.2.0) — RESEARCH.md's cached release-please-action SHA (8b8fd2cc...) was confirmed a phantom/nonexistent commit (HTTP 422), matching the orchestrator's independent verification"
  - "Dry-run spine proof executed against the real pushed feature branch via release-please's --target-branch flag, not against main — proves the mechanism without touching main or requiring the still-pending GitHub App"

patterns-established:
  - "Non-vacuous guard proof: mutate the premise, run the test, capture the failure output containing diagnosable detail (e.g. the reconstructed SAN), restore, re-run green — recorded verbatim in the plan SUMMARY, not just asserted"

requirements-completed: [REL-02]

coverage:
  - id: D1
    description: "TestReleaseWorkflowFileMatchesPattern reads release.yml off disk, reconstructs the SAN a tag push would produce, and proves releaseWorkflowRefPattern accepts it and rejects a renamed-workflow SAN and a branch-ref SAN"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestReleaseWorkflowFileMatchesPattern"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestReleaseWorkflowTriggerIsTagPushOnly mechanically pins release.yml's on: block to exactly one push/tags trigger (v[0-9]*), enforcing the header comment's claim"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestReleaseWorkflowTriggerIsTagPushOnly"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestWorkflowSourceHelpersFailLoudly proves every parse core returns a non-nil error (never a zero value) on 6 synthetic missing-target cases across all 4 helper pairs"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestWorkflowSourceHelpersFailLoudly"
        status: pass
    human_judgment: false
  - id: D4
    description: "release-please-config.json / .release-please-manifest.json resolve the repo's real 0.1.0 baseline with no bootstrap PR, no release-as, no extra-files"
    requirement: "REL-02"
    verification:
      - kind: integration
        ref: "npx release-please@latest release-pr --dry-run --target-branch=gsd/v1.0-drop-in-parity-human-ux (see this SUMMARY's Spine Proof section)"
        status: pass
    human_judgment: false
  - id: D5
    description: "6-target pretag-gate job blocks the release-please job via needs:, and its set -euo pipefail failure path is proven to abort on a rejecting GOOS/GOARCH pair"
    requirement: "REL-02"
    verification:
      - kind: unit
        ref: "shell loop substitution, observed exit status 1 (see Deviations/Non-Vacuity section)"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-07-29
status: complete
---

# Phase 9 Plan 1: release-please spine + non-vacuous drift guard Summary

**Stood up the release-please mechanical spine (config, manifest, App-token workflow, blocking pre-tag gate) and proved — via a live dry-run against the real pushed branch, not a synthetic fixture — that it resolves this repo's real 0.1.0 baseline while a new Go test mechanically pins release.yml's on-disk shape to `internal/upgrade`'s LOCKED cosign SAN pattern.**

## Performance

- **Duration:** ~40 min (includes a ~15 min live `release-please --dry-run` against 490 real commits)
- **Tasks:** 2
- **Files modified:** 4 (all new)

## Accomplishments
- `internal/upgrade/release_workflow_shape_test.go`: 4 hand-rolled workflow-YAML parse cores (`parseWorkflowTopLevelName`/`OnKeys`/`PushTagPatterns`/`StepRunBlock`) as error-returning `parseX`/`mustX` pairs, plus `TestReleaseWorkflowFileMatchesPattern` (accept + 2 reject subtests), `TestReleaseWorkflowTriggerIsTagPushOnly`, and `TestWorkflowSourceHelpersFailLoudly` (6 synthetic missing-target cases).
- `release-please-config.json` + `.release-please-manifest.json` seeded to the repo's real `0.1.0` baseline (D-06/D-07) — no `release-as`, no `extra-files`, no `bootstrap-sha`.
- `.github/workflows/release-please.yml`: App-token workflow (`actions/create-github-app-token` + `googleapis/release-please-action`) plus a blocking `pretag-gate` job running the 6-target `go list -mod=readonly` sweep from `docs/RELEASE-PROCEDURES.md` §1.
- Live end-to-end spine proof: `release-please --dry-run --target-branch=gsd/v1.0-drop-in-parity-human-ux` against the real, pushed branch resolved baseline `0.1.0` with no bootstrap PR and computed candidate `0.2.0` (expected — the `Release-As: 1.0.0` one-shot footer is plan 09-07's job, not this plan's).
- `release.yml` and `internal/upgrade/verify.go` confirmed byte-identical throughout (`git diff` empty at every checkpoint).

## Task Commits

1. **Task 1: Drift guard + release-please config/manifest/workflow** - `7f60822` (feat)
2. **Task 2: Blocking 6-target pre-tag gate in release-please.yml** - `ce403dc` (feat)

**Plan metadata:** (this commit, docs)

## Files Created/Modified
- `internal/upgrade/release_workflow_shape_test.go` - workflow-shape drift guard (Task 1)
- `release-please-config.json` - release-please strategy config (Task 1)
- `.release-please-manifest.json` - version baseline seed (Task 1)
- `.github/workflows/release-please.yml` - App-token workflow + pretag-gate job (Tasks 1+2)

## Decisions Made
- Re-resolved both action SHAs live via `gh api repos/<org>/<repo>/git/ref/tags/<tag>` rather than trusting RESEARCH.md's cached values — confirmed the orchestrator's independently-verified SHAs (`googleapis/release-please-action@45996ed1...`, `actions/create-github-app-token@bcd2ba49...`) and confirmed RESEARCH.md's cached release-please-action SHA (`8b8fd2cc...`) does not exist (`gh api` HTTP 422).
- Ran the mandatory `release-please --dry-run` spine proof against the real pushed feature branch (`--target-branch=gsd/v1.0-drop-in-parity-human-ux`) rather than against `main`, since the App/secrets are not yet provisioned (plan 09-05) and pushing to `main` is out of scope for this plan. This is a legitimate substitute: release-please's manifest-mode discovery reads committed repo state via the GitHub API regardless of which branch is targeted, so this genuinely exercises the config/manifest resolution logic end-to-end, not a synthetic fixture.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Non-vacuity failure message didn't print the reconstructed SAN**
- **Found during:** Task 1, drift-guard non-vacuity demonstration
- **Issue:** The initial `TestReleaseWorkflowFileMatchesPattern` implementation used `mustWorkflowTopLevelName`'s `t.Fatalf` wrapper for the `name:` equality check, which aborted the test before the SAN was computed/printed — violating the plan's explicit acceptance criterion ("failure message contains the reconstructed SAN string").
- **Fix:** Restructured the test to build the SAN first (from the path constant's basename, independent of the parsed `name:` value per the plan's own design), then report a `name:` mismatch non-fatally via `t.Errorf` with the SAN included in the message, so the `accept`/`reject` subtests still run and the SAN is always visible in a failure.
- **Files modified:** `internal/upgrade/release_workflow_shape_test.go`
- **Verification:** Re-ran the mutate/observe/restore cycle; failure output now reads `release.yml's top-level name: = "release-renamed", want "release" (reconstructed SAN: "https://github.com/seanb4t/codegraph-go/.github/workflows/release.yml@refs/tags/v1.2.3")`.
- **Committed in:** `7f60822` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, found and corrected before the task commit — not a separate follow-up commit)
**Impact on plan:** No scope creep; the fix was required to actually satisfy the plan's own acceptance criterion.

## Non-Vacuity: Observed Break-Observe-Restore Output (mandatory, recorded verbatim)

### Guard 1 — `TestReleaseWorkflowFileMatchesPattern` (drift guard)

Mutated `.github/workflows/release.yml`'s `name: release` to `name: release-renamed`:

```
--- FAIL: TestReleaseWorkflowFileMatchesPattern (0.00s)
    release_workflow_shape_test.go:318: release.yml's top-level name: = "release-renamed", want "release" (reconstructed SAN: "https://github.com/seanb4t/codegraph-go/.github/workflows/release.yml@refs/tags/v1.2.3")
```
(accept/reject_renamed_workflow/reject_branch_ref subtests still ran and PASSed, since the SAN itself is built from the filename, not the `name:` value — proving the two checks are independent and both load-bearing.)

Restored `release.yml` from a pre-mutation backup; re-ran:
```
--- PASS: TestReleaseWorkflowFileMatchesPattern (0.00s)
    --- PASS: TestReleaseWorkflowFileMatchesPattern/accept (0.00s)
    --- PASS: TestReleaseWorkflowFileMatchesPattern/reject_renamed_workflow (0.00s)
    --- PASS: TestReleaseWorkflowFileMatchesPattern/reject_branch_ref (0.00s)
```
`diff` against the pre-mutation backup confirmed byte-identical restoration.

### Guard 2 — `TestReleaseWorkflowTriggerIsTagPushOnly` (trigger guard)

Added a second top-level key (`pull_request:`) under release.yml's `on:` block:

```
--- FAIL: TestReleaseWorkflowTriggerIsTagPushOnly (0.00s)
    release_workflow_shape_test.go:356: release.yml's on: block top-level keys = [pull_request push], want exactly [push]
```

Restored `release.yml`; re-ran:
```
--- PASS: TestReleaseWorkflowTriggerIsTagPushOnly (0.00s)
```
`diff` against the pre-mutation backup confirmed byte-identical restoration. `git diff -- .github/workflows/release.yml internal/upgrade/verify.go` was empty at every checkpoint throughout both demonstrations.

### Guard 3 — pretag-gate loop (`set -euo pipefail` propagation, Task 2)

All 6 real target pairs pass locally (no output, exit 0). A single rejecting pair in isolation:
```
$ GOOS=linux GOARCH=bogusarch go list -mod=readonly ./... > /dev/null
[build-constraint errors printed]
exit status of bogus pair alone: 1
```
Substituting that pair for the 6th real one inside the actual loop shape (`set -euo pipefail; for pair in ...; do GOOS=... GOARCH=... go list -mod=readonly ./... > /dev/null; done`):
```
checking linux/amd64
checking linux/arm64
checking windows/amd64
checking windows/arm64
checking darwin/amd64
checking bogusos/bogusarch
[build-constraint errors printed]
observed exit status: 1
```
The loop aborted at the rejecting pair and never printed "loop completed without aborting" — confirming `set -euo pipefail` propagates the failure rather than logging and continuing.

## End-to-End Spine Proof (mandatory acceptance criterion, recorded verbatim)

Ran `npx -y release-please@latest release-pr --dry-run --debug --repo-url=seanb4t/codegraph-go --config-file=release-please-config.json --manifest-file=.release-please-manifest.json --target-branch=gsd/v1.0-drop-in-parity-human-ux --token="$(gh auth token)"` against the real, pushed `gsd/v1.0-drop-in-parity-human-ux` branch (both task commits pushed to `origin` first, so the manifest/config files exist in the branch release-please reads via the GitHub API).

Observed result:
```
⚠ No latest release pull request found.
✔ Considering: 538 commits
Would open 1 pull requests
title: chore(gsd/v1.0-drop-in-parity-human-ux): release 0.2.0
## [0.2.0](https://github.com/seanb4t/codegraph-go/compare/v0.1.0...v0.2.0) (2026-07-29)
```

- **Resolved baseline: `0.1.0`** — confirmed by the changelog compare link starting from `v0.1.0` and by no bootstrap-PR messaging appearing anywhere in the run.
- **Candidate next version: `0.2.0`** — the expected default-strategy bump from the accumulated `feat`/`fix`/`perf` commits on this branch; this is correct and expected for this plan, since the one-shot `Release-As: 1.0.0` footer that forces `1.0.0` is plan 09-07's deliverable, not this one.
- Three commit messages triggered a harmless conventional-commits-parser warning (`commit could not be parsed`) on footer-adjacent parentheses in the subject line; release-please logged and skipped them without failing the run (`commits: 538` still counted correctly). Not a defect in this plan's deliverables — noted for awareness only.
- This satisfies the plan's acceptance criterion in full: baseline discovered from the existing `v0.1.0` GitHub Release with no bootstrap PR, and a candidate version was printed.

## Issues Encountered
- The dry-run took ~15 minutes because release-please's manifest-mode SHA discovery walks the target branch's commit history backward, 10 commits per API page with a per-commit file-list backfill call, until it finds the last release's commit SHA (490 commits deep on this D-09 fast-forward branch). This is an inherent cost of proving the spine against a real 477+-commit branch rather than a synthetic fixture — expected, not a defect, and it completed successfully.

## User Setup Required
None for this plan. The GitHub App creation/installation (`APP_ID`/`APP_PRIVATE_KEY` secrets) referenced by `.github/workflows/release-please.yml` is plan 09-05's blocking human checkpoint — declared in this plan's frontmatter `user_setup` only because this file references those secret names.

## Next Phase Readiness
- The release-please spine is proven correct in isolation: config/manifest resolve the real baseline, the pretag-gate blocks on a real failure mode, and both new guards are demonstrated non-vacuous.
- `release.yml`'s `Publish GitHub release` step (D-04's create-vs-upload branch) is untouched — that is plan 09-02's single highest-risk edit, ready to proceed.
- No blockers for 09-02/09-03. 09-05's GitHub App setup remains the phase's one hard external dependency before a real (App-token-authored) run of `.github/workflows/release-please.yml` can be exercised in CI.

---
*Phase: 09-release-please-and-goreleaser*
*Completed: 2026-07-29*
