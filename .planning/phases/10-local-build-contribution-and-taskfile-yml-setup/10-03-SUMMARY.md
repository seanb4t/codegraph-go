---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 03
subsystem: infra
tags: [release-yml, namespace-runners, slsa, goreleaser, github-actions, darwin, ci]

# Dependency graph
requires:
  - phase: 10-local-build-contribution-and-taskfile-yml-setup
    provides: "10-01's proven Namespace-runner pattern (namespace-profile-linux-amd64-{2x4,4x8}, setup-go cache:false + nscloud-cache-action) and the .github/actionlint.yaml self-hosted-runner-label config this plan extends"
provides:
  - "release.yml's build matrix (linux/windows legs) and assemble job moved to namespace-profile-linux-amd64-{4x8,2x4}"
  - "release.yml's build matrix darwin legs moved to namespace-profile-macos-6x14-tahoe per an explicit maintainer checkpoint decision — UNPROVEN in real CI, see Known Unknowns below"
  - "release.yml's provenance job left unchanged (D-07, structural) with an in-file comment explaining why"
  - "internal/upgrade/release_workflow_shape_test.go TestDarwinLegsBuildNatively and TestProvenanceJobUsesTaggedSLSAGenerator — machine-enforced D-08/D-07, each demonstrated RED then GREEN against real mutations"
affects: [10-04, 10-05, 10-06, 10-07]

# Actuals (#2632)
actuals:
  tokens: 3643
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Runner-only edits to a locked-contract release pipeline: every run: body, job name, and version pin verified byte-identical/set-equal before vs after via a scripted diff, not eyeballed"
    - "Allow-set (pattern-family) assertions for runner labels, not literal-string equality — TestDarwinLegsBuildNatively accepts macos-*/namespace-profile-macos-* so a future native-profile move stays green while a move to any linux profile still fails"

key-files:
  created: []
  modified:
    - .github/workflows/release.yml
    - .github/actionlint.yaml
    - internal/upgrade/release_workflow_shape_test.go

key-decisions:
  - "Maintainer checkpoint decision (blocking, human): darwin legs move from macos-latest to namespace-profile-macos-6x14-tahoe (maintainer-attested native Apple-Silicon macOS profile, 6 CPU/14GB, macOS Tahoe), not stay-github-hosted. Recorded verbatim from the coordinator's resume message; this executor did not select the option."
  - "No run: body, job name:, or version pin (GORELEASER_VERSION: v2.17.0) changed anywhere in release.yml — verified by scripted before/after comparison, not by inspection alone"
  - "TestDarwinLegsBuildNatively asserts runner-label FAMILY membership (macos-*, namespace-profile-macos-*) rather than one literal string, per the plan's explicit instruction, so a later deliberate move to a different native macOS profile does not go red for the wrong reason"

patterns-established: []

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "release.yml's build matrix (4 linux/windows legs -> namespace-profile-linux-amd64-4x8, 2 darwin legs -> namespace-profile-macos-6x14-tahoe) and assemble job (-> namespace-profile-linux-amd64-2x4) moved to Namespace runners; provenance job left unchanged (D-07); no run: body, job name, or version pin touched"
    requirement: "DEV-01"
    verification:
      - kind: other
        ref: "scripted extraction+comparison of every run: body in release.yml, before vs after this plan's commit — all 7 byte-identical"
        status: pass
      - kind: other
        ref: "sorted-set diff of every job name: value, before vs after — identical"
        status: pass
      - kind: other
        ref: "grep-equivalent checks: env.GORELEASER_VERSION still v2.17.0; provenance job's uses: still the v2.1.0-tagged SLSA generic generator; both darwin entries still needs_zig: false"
        status: pass
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task lint:actions (exit 0, local, against the edited file)"
        status: pass
    human_judgment: true
    rationale: "release.yml triggers ONLY on a v[0-9]* tag push (on: push: tags:) — it does not run on pull_request, so PR #19 will not exercise ANY part of this change. namespace-profile-linux-amd64-{4x8,2x4} are the same labels waves 1/2 already proved green in real CI on this repo, but that proof does not extend to this file's own trigger path. namespace-profile-macos-6x14-tahoe has NEVER been exercised in this repository and cannot be until an actual v* tag is pushed — see Known Unknowns below. A human (a real tag push, watched end-to-end) must confirm this before it is trusted as the release pipeline."
  - id: D2
    description: "TestDarwinLegsBuildNatively (D-08: darwin legs never needs_zig:true, runner drawn from a macOS-class allow-set) and TestProvenanceJobUsesTaggedSLSAGenerator (D-07: provenance job tag-referenced, declares no runs-on:) added to internal/upgrade/release_workflow_shape_test.go following the file's parseX/mustX convention; both empty-input table cases added"
    requirement: "DEV-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestDarwinLegsBuildNatively"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestProvenanceJobUsesTaggedSLSAGenerator"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... (full package, count=1)"
        status: pass
    human_judgment: false

duration: unknown — record_start_time was not run before file-reading began; commit-to-commit span across both task commits is a few minutes, excluding preceding checkpoint evidence-gathering and the maintainer's decision turnaround
completed: 2026-08-01
status: complete
---

# Phase 10 Plan 3: release.yml onto Namespace runners — darwin leg moved by maintainer decision, D-07/D-08 machine-enforced Summary

**Moved release.yml's linux/windows build legs to `namespace-profile-linux-amd64-4x8`, the `assemble` job to `namespace-profile-linux-amd64-2x4`, and — per an explicit blocking maintainer checkpoint decision — the darwin legs to `namespace-profile-macos-6x14-tahoe`; left the SLSA `provenance` job untouched (D-07, structural); locked both D-07 and D-08 into machine-enforced tests, each proven RED then GREEN against a real mutation; zero `run:` bodies, job names, or version pins changed.**

## Performance

- **Duration:** unknown (not captured — see frontmatter `duration` note)
- **Completed:** 2026-08-01
- **Tasks:** 2 / 2 (plus the blocking checkpoint, decided by the maintainer)
- **Files modified:** 3 (0 created, 3 modified: `.github/workflows/release.yml`, `.github/actionlint.yaml`, `internal/upgrade/release_workflow_shape_test.go`)

## Checkpoint Decision (blocking, human)

This plan opened with a blocking `checkpoint:decision`: whether `release.yml`'s darwin build leg moves to a Namespace macOS runner profile, or stays on GitHub-hosted `macos-latest`. This executor gathered evidence (the live `release.yml` state, and — via `gh api` — the sibling reference-implementation repo `holomush/holomush`'s `release.yaml`, which has **no darwin/macOS build leg at all**, so it provides no precedent either way) and returned the decision to the coordinator without selecting an option, per the plan's explicit instruction that a human was waiting on it.

**The maintainer decided: `namespace-macos`.** Profile label `namespace-profile-macos-6x14-tahoe` (6 CPU / 14 GB, macOS Tahoe), maintainer-attested as their native Apple-Silicon macOS profile from the Namespace dashboard. This decision is recorded here verbatim from the coordinator's resume message — it was not re-derived or second-guessed by this executor.

## Accomplishments

- `release.yml`'s `build` matrix: 4 linux/windows legs move `runner: ubuntu-latest` -> `namespace-profile-linux-amd64-4x8` (needs_zig values unchanged); 2 darwin legs move `runner: macos-latest` -> `namespace-profile-macos-6x14-tahoe` (`needs_zig: false` unchanged on both — still a native Xcode-clang build, never zig)
- `assemble` job: `runs-on: ubuntu-latest` -> `namespace-profile-linux-amd64-2x4`
- `provenance` job: unchanged, with a new in-file comment explaining that a reusable workflow declares its own `runs-on` and a caller cannot override it (D-07) — the documented sole exception
- The `build` job's `Set up Go` step gains `cache: false` + a new `Cache Go modules and build` step (`namespacelabs/nscloud-cache-action@c5f8dab7...` / `v1.6.1`, same pin as 10-01), matching the pattern already proven in `ci.yml`; `assemble` has no `setup-go` step, so no caching change there
- Updated the matrix-item and locked-contract-header comments to name the new runner profiles while preserving the darwin-DNS-resolver rationale (Finding 2 / Pitfall 2) — the reason the leg exists is not deleted, only the runner name it currently points at
- `internal/upgrade/release_workflow_shape_test.go` gains `TestDarwinLegsBuildNatively` and `TestProvenanceJobUsesTaggedSLSAGenerator`, following the file's existing `parseX`/`mustX` idiom, plus empty-input table cases added to `TestWorkflowSourceHelpersFailLoudly`
- Verified, not eyeballed: all 7 `run:` bodies in `release.yml` are byte-identical before vs after (scripted extraction+comparison); the set of job `name:` values is identical; `env.GORELEASER_VERSION` still reads `v2.17.0`; the `provenance` job's `uses:` still resolves to the `v2.1.0`-tagged SLSA generic generator; both darwin entries still carry `needs_zig: false`
- `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` exits 0 against the edited file (required a Rule 3 deviation — see below)

## Task Commits

1. **Task 1: Move release.yml's movable jobs to Namespace runners** - `36416e0` (feat)
2. **Task 2: Lock the two hard release-pipeline constraints into the existing shape test** - `df20fbb` (test)

## Files Created/Modified

- `.github/workflows/release.yml` - runner classes and cache mechanics moved to Namespace; darwin legs moved per the maintainer's checkpoint decision; provenance job unchanged plus explanatory comment
- `.github/actionlint.yaml` - added `namespace-profile-macos-6x14-tahoe` to `self-hosted-runner.labels` (deviation, see below)
- `internal/upgrade/release_workflow_shape_test.go` - `TestDarwinLegsBuildNatively`, `TestProvenanceJobUsesTaggedSLSAGenerator`, their shared parse helpers (`parseReleaseBuildMatrix`, `parseReleaseProvenanceJob`, `isMacOSClassRunner`), and two new empty-input table cases

## Decisions Made

- Darwin runner profile: `namespace-macos` / `namespace-profile-macos-6x14-tahoe` — maintainer decision, recorded above, not made by this executor
- `TestDarwinLegsBuildNatively` asserts membership in a runner-label-family allow-set (`macos-*`, `namespace-profile-macos-*`), not one literal string, per the plan's explicit instruction — a later deliberate move to a different native macOS profile stays green
- No `run:` body in `release.yml` was touched anywhere in this plan, including the `goreleaser build --single-target --clean` step and every shell block in `assemble` — only `runner:`/`runs-on:`/`cache:` values, the two new cache steps, and comments changed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `namespace-profile-macos-6x14-tahoe` to `.github/actionlint.yaml`**
- **Found during:** Task 1, running `task lint:actions` against the edited `release.yml`
- **Issue:** `actionlint`'s built-in runner-label list only knows GitHub-hosted labels; without a config entry it reported both darwin matrix entries' `runner: namespace-profile-macos-6x14-tahoe` as unknown (`[runner-label]`), which would have failed this task's own `<verify>` and, downstream, the `actionlint` required-check job in `ci.yml`. Same root cause as 10-01's Rule 3 deviation for the two linux profile labels, now recurring for the new macOS label the maintainer's checkpoint decision introduced.
- **Fix:** Added `namespace-profile-macos-6x14-tahoe` to `.github/actionlint.yaml`'s `self-hosted-runner.labels` list, alongside the two existing linux labels.
- **Files modified:** `.github/actionlint.yaml`
- **Verification:** `GOWORK=off go tool -modfile=go.tool.mod task lint:actions` — exit 1 with `[runner-label]` before the config addition, exit 0 after.
- **Committed in:** `36416e0` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3, blocking)
**Impact on plan:** Necessary for Task 1's own stated `<verify>` (`task lint:actions` exit 0) to hold at all. No scope creep — same class of fix 10-01 already established a precedent for.

## Non-Vacuity Demonstrations (RED-then-GREEN), as performed

1. **`TestDarwinLegsBuildNatively`, needs_zig direction:** flipped one darwin/amd64 matrix entry's `needs_zig` from `false` to `true` via a scripted, asserted-single-substitution edit, confirmed the mutation landed via `rg`, ran the test -> FAIL naming `darwin/amd64`; restored from a full-file backup, confirmed `git diff --stat` empty -> PASS.
2. **`TestDarwinLegsBuildNatively`, runner-label direction:** swapped that same darwin/amd64 entry's `runner:` from `namespace-profile-macos-6x14-tahoe` to `namespace-profile-linux-amd64-4x8` (asserted single substitution), ran the test -> FAIL naming the exact wrong label; restored from backup, confirmed zero diff -> PASS.
3. **`TestProvenanceJobUsesTaggedSLSAGenerator`:** swapped the `provenance` job's `uses:` tag suffix (`@v2.1.0`) for a fabricated 40-char SHA (asserted single substitution), ran the test -> FAIL naming the SHA-form reference; restored from backup, confirmed zero diff -> PASS.

All three mutations were applied to the real, on-disk `release.yml` (not a scratch copy) and reverted from a full-file backup taken before the first mutation, with `git diff --stat` confirming zero residual diff after each restore — the working tree was clean before Task 1's actual (permanent) edit was committed.

## Issues Encountered

None beyond the Rule 3 deviation documented above.

## Known Unknowns — read before treating this leg as validated

**`release.yml` triggers ONLY on `on: push: tags: ["v[0-9]*"]`. It does not run on `pull_request`.** PR #19 (or any PR) will not exercise any part of this plan's change — not the linux/windows runner move, not the darwin runner move, not the cache-mechanics change. The only way to observe this workflow run at all is a real `v*` tag push.

Unlike waves 1 and 2, where `namespace-profile-linux-amd64-2x4` and `namespace-profile-linux-amd64-4x8` were empirically proven green in real CI on PR #19, **`namespace-profile-macos-6x14-tahoe` has never been exercised in this repository, in this plan, or in the sibling reference repo `holomush/holomush` (which has no darwin build leg at all).** It cannot be exercised until an actual release tag is pushed. The label is maintainer-attested from their Namespace dashboard, not machine-observed in a workflow run.

Specific unknowns a real release will settle, in order of how loudly they'd fail if wrong:

1. **Whether the profile is served at all for this repo.** An unserved label queues the job indefinitely rather than failing — silent by construction, same failure mode 10-01/10-02 flagged for the linux profiles before their first real CI run confirmed them.
2. **Whether the image ships Xcode / Command Line Tools, so `clang` exists.** If absent, the CGo darwin build fails loudly at the `goreleaser build --single-target --clean` step — this is the *safe* failure, since a red job blocks the release before anything is signed or published.
3. **Whether the macOS SDK version on a Tahoe image shifts the effective minimum deployment target of the produced binaries relative to what `macos-latest` produced.** This is the quiet one: a binary can build, get signed by cosign, get an SBOM, and pass SLSA provenance cleanly on this runner, and still refuse to launch on an older macOS a user is running. No gate in this pipeline currently checks a built binary's minimum-OS-version metadata.

**What IS true by construction, and what is NOT yet verified:**
- Preserved by construction: `needs_zig: false` on both darwin entries (still a native Xcode-clang build, never a zig cross-link), so the D-08 libresolv/DNS-resolver mitigation's *structural precondition* holds regardless of which macOS host runs it.
- NOT verified: that `namespace-profile-macos-6x14-tahoe` is actually a native (not emulated, not cross-compiled) darwin host, that it produces working binaries, or that those binaries' DNS resolution behaves identically to what `macos-latest` produced. `TestDarwinLegsBuildNatively` (this plan's Task 2) enforces the *matrix declaration* (`needs_zig: false` + a macOS-class label) — it cannot and does not enforce anything about what the runner actually does at build time.

Do not treat this leg as validated until a real `v*` tag push has been watched end-to-end: the `build` job scheduling on `namespace-profile-macos-6x14-tahoe` rather than queuing, both darwin binaries building successfully, and (ideally) a `codegraph upgrade` smoke test against the resulting macOS binary on a real macOS machine.

## User Setup Required

None during execution beyond what 10-01 already established (Namespace GitHub integration, verified satisfied for that plan and unchanged here). The new `namespace-profile-macos-6x14-tahoe` label's actual availability is exactly the open item recorded above — nothing further for the maintainer to configure locally; the confirmation can only come from a real tag push.

## Next Phase Readiness

- `release.yml` is now on Namespace runners everywhere it structurally can be, with both hard exceptions (D-07 provenance, and — as directed by the maintainer's checkpoint decision — no exception on darwin, which also moved) documented in-file and machine-enforced by tests.
- **Blocker carried forward, higher stakes than 10-01/10-02's equivalent gap:** this plan's entire diff is invisible to every PR (tag-push-only trigger), so the darwin runner change specifically must be watched on the next real release tag push, not the next PR. If `namespace-profile-macos-6x14-tahoe` is not actually served, is not a native host, or produces binaries with the DNS-resolution regression D-08 exists to prevent, no gate in this repository will catch it before a broken `codegraph upgrade` binary reaches a user's disk — recovery would mean cutting a new release. Recorded as a STATE.md blocker (see below); this needs a maintainer-observed real release before it can be trusted.
- 10-04 onward (per `10-CONTEXT.md` D-09/D-15) can proceed independently — `bench.yml`'s Namespace re-bless ordering and `release-please.yml`'s `check:cross` replacement don't depend on this plan's darwin-runner decision.

## Self-Check: PASSED

Both modified deliverable files (`.github/workflows/release.yml`, `internal/upgrade/release_workflow_shape_test.go`) plus the deviation file (`.github/actionlint.yaml`) confirmed present on disk with the expected content; both task commits (`36416e0`, `df20fbb`) confirmed in `git log`.

**Self-check verification performed:**
```
FOUND: .github/workflows/release.yml
FOUND: .github/actionlint.yaml
FOUND: release_workflow_shape_test.go
FOUND: SUMMARY.md
FOUND: 36416e0
FOUND: df20fbb
```

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-01*
