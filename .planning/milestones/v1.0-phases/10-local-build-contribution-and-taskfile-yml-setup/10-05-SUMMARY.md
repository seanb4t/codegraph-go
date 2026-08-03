---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 05
subsystem: infra
tags: [go-task, goreleaser, github-actions, release-please, yaml, ci]

requires:
  - phase: 10-local-build-contribution-and-taskfile-yml-setup
    provides: "Taskfile.yml (GO_TOOL/GO_TOOL_LINT vars, go.tool.mod, ./.github/actions/install-task) from plans 10-01/10-02"
provides:
  - "task check:cross — the pre-tag 6-target go list -mod=readonly sweep, now defined once in Taskfile.yml"
  - "release-please.yml's pretag-gate job routed through task check:cross instead of an inline copy of the sweep"
  - "TestCheckCrossMatchesGoreleaserTargets — a real-YAML-parsed set-equality guard between check:cross and .goreleaser.yaml's builds: targets"
  - "docs/RELEASE-PROCEDURES.md section 1 naming task check:cross as the local invocation"
affects: [release-procedures, ci-taskfile, goreleaser-config]

actuals:
  tokens: 3614
  tasks: 3
  commits: 3

tech-stack:
  added: ["go.yaml.in/yaml/v3 (already an indirect dependency, now imported directly by a test file for real YAML decoding)"]
  patterns:
    - "Port CI shell verbatim into a Taskfile target rather than reimplementing 'more cleanly' in Taskfile syntax, to avoid silently changing gate behavior (D-15)"
    - "Set-equality guards compared as one sorted-string equality (sortedPairSet) rather than count/exit-status chaining, so omission and addition both fail in one assertion"
    - "Parse structured config (YAML) with a real decoder, never a raw-text regex tuned to one syntax variant — proven with an explicit inline-flow-sequence fixture test"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - .github/workflows/release-please.yml
    - internal/upgrade/taskfile_shape_test.go
    - docs/RELEASE-PROCEDURES.md

key-decisions:
  - "check:cross's command body is byte-for-byte the loop from release-please.yml lines 47-54 — not reimplemented in cleaner Taskfile syntax — per D-15's explicit prohibition on silently changing the target list."
  - "pretag-gate stays on ubuntu-latest (D-06's Namespace migration explicitly excludes this file) but gains an Install Task step; no nscloud-cache-action was added."
  - "Install Task step duration was measured locally (not on the real ubuntu-latest CI runner) at ~2.8s warm, ~9.7s with GOCACHE cleared, ~20.2s fully cold (cleared GOCACHE and GOMODCACHE, real network download of go.tool.mod's module graph) — all well under the plan's 60s threshold, so no actions/cache step was added. The real CI-runner number is unverified; see Known Gaps."
  - "TestCheckCrossMatchesGoreleaserTargets parses .goreleaser.yaml with go.yaml.in/yaml/v3 (a real YAML decoder), not a text regex, specifically to handle the file's inline flow-sequence goos:/goarch: form — proven directly by TestParseGoreleaserCrossPairs_InlineFlowSequence."
  - "go.mod/go.sum left unmodified: go.yaml.in/yaml/v3 was already required indirectly and resolves without a go mod tidy edit. go mod tidy itself was not run because it network-faults on an unrelated pre-existing tree-sitter-swift test dependency, out of this task's scope (SCOPE BOUNDARY)."

patterns-established:
  - "Non-vacuity proof method for a moved/added gate: mutate the real file, confirm the mutation landed via git diff, run the check and capture its exit code from the command itself (never a pipe tail), observe the specific failure naming the offending item, then restore and confirm green — applied identically in Task 1 (Go build-constraint mutation) and Task 3 (Taskfile pair deletion + .goreleaser.yaml extra build entry)."

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "task check:cross exists in Taskfile.yml, ported verbatim from release-please.yml's inline sweep, and rejects a genuinely unresolvable cross-target"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "GOWORK=off go tool -modfile=go.tool.mod task check:cross (manual RED/GREEN mutation proof, see Task 1 notes)"
        status: pass
    human_judgment: false
  - id: D2
    description: "release-please.yml's pretag-gate job invokes task check:cross via an Install Task + Run check:cross sweep step pair, job name and runs-on unchanged, run: body is exactly one line"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestRequiredCheckNamesPreserved"
        status: pass
      - kind: unit
        ref: "GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml (task lint:actions)"
        status: pass
    human_judgment: true
    rationale: "release-please.yml triggers only on push: branches: [main] — a real pushed run showing pretag-gate green with Install Task + Run check:cross sweep steps, and the gate genuinely failing on a pushed mutation, cannot be confirmed from this feature branch. See Known Gaps."
  - id: D3
    description: "TestCheckCrossMatchesGoreleaserTargets enforces set equality between check:cross's pairs and .goreleaser.yaml's builds: pairs using a real YAML decoder, proven non-vacuous in both directions plus edge cases (empty builds, missing check:cross target, inline-flow-sequence parsing, empty-input error contract)"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run TestCheckCrossMatchesGoreleaserTargets -v"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... (full package, including TestParseGoreleaserCrossPairs_InlineFlowSequence and TestCheckCrossParsersFailLoudly)"
        status: pass
    human_judgment: false

duration: 41min
completed: 2026-08-02
status: complete
---

# Phase 10 Plan 05: One-Definition Pre-Tag Cross-Target Sweep Summary

**`task check:cross` now the single definition of the 6-target `go list -mod=readonly` sweep, with `release-please.yml`'s `pretag-gate` routed through it and a real-YAML set-equality guard against `.goreleaser.yaml`'s build matrix.**

## Performance

- **Duration:** 41 min
- **Started:** 2026-08-02T08:40:31-04:00 (first task commit)
- **Completed:** 2026-08-02T09:21:45-04:00 (last task commit)
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Ported `release-please.yml`'s inline 6-target `go list -mod=readonly` sweep into `Taskfile.yml`'s `check:cross`, byte-for-byte, so a contributor can run the exact pre-tag gate locally.
- Routed `release-please.yml`'s `pretag-gate` job through `task check:cross` (Install Task + one-line `run:`), keeping the job on `ubuntu-latest` outside D-06's Namespace scope, with the job `name:` and required-check fixture unchanged.
- Added `TestCheckCrossMatchesGoreleaserTargets`, a real-YAML-decoder-based set-equality guard between `check:cross`'s swept pairs and `.goreleaser.yaml`'s `builds:` pairs, proven non-vacuous in both directions via live mutation.
- Updated `docs/RELEASE-PROCEDURES.md` section 1 to name `task check:cross` as the local invocation.

## Task Commits

Each task was committed atomically:

1. **Task 1: Port the six-target sweep into a check:cross target, verbatim** - `b760f86` (feat)
2. **Task 2: Route release-please.yml's pretag-gate through task check:cross** - `c42a2dd` (feat)
3. **Task 3: Guard that check:cross and .goreleaser.yaml enumerate the same six targets** - `356af3f` (test)

_Note: Task 3 is `tdd="true"` but all its declared files are test-only (`internal/upgrade/taskfile_shape_test.go`) — the parsers and test were authored together, then RED/GREEN discipline was applied as the plan's own required non-vacuity mutation proofs (see TDD Gate Compliance below) rather than a separate feat commit, since there is no separate production file for this task to implement against._

## Files Created/Modified
- `Taskfile.yml` - Added `check:cross` target (6-target `go list -mod=readonly` sweep, verbatim from release-please.yml)
- `.github/workflows/release-please.yml` - `pretag-gate` job: added `Install Task` step, replaced the 8-line inline sweep step with `Run check:cross sweep: task check:cross`, added a comment on the D-06-scope/D-15-bootstrap intersection
- `internal/upgrade/taskfile_shape_test.go` - Added `parseGoreleaserCrossPairs`/`mustParseGoreleaserCrossPairs`, `parseCheckCrossPairs`/`mustParseCheckCrossPairs`, `sortedPairSet`, and 6 new test functions
- `docs/RELEASE-PROCEDURES.md` - Section 1's automation callout now names `task check:cross` as the invocation

## Decisions Made
- Ported the sweep loop character-for-character rather than rewriting it in "cleaner" Taskfile syntax — D-15 explicitly warns this is the one way to silently change the target list.
- Used `go.yaml.in/yaml/v3` (already an indirect dependency, already in `go.sum`) for real YAML decoding in the new guard test, rather than a raw-text regex — required to correctly handle `.goreleaser.yaml`'s inline flow-sequence `goos: [linux]` form, and proven directly by a dedicated fixture test (`TestParseGoreleaserCrossPairs_InlineFlowSequence`).
- Did not run `go mod tidy` to promote `go.yaml.in/yaml/v3` from indirect to direct in `go.mod` — the import resolves and all tests/builds pass without it, and `go mod tidy` itself network-faults on an unrelated pre-existing `tree-sitter-swift` test dependency elsewhere in the repo (out of this task's scope per the SCOPE BOUNDARY rule).
- Measured the `Install Task` step's cold-build cost locally (not on the real CI runner) across three cache states — see TDD/Verification Gaps below — and, per RESEARCH.md's Open Question 2 measure-then-decide instruction, did not add `actions/cache` since all three measurements were well under the plan's 60-second threshold.

## Deviations from Plan

None — plan executed exactly as written. No Rule 1-4 auto-fixes were needed; the only judgment call was how to satisfy Task 2's "measure the workflow on the branch" instruction given `release-please.yml` cannot be triggered from a feature branch (documented below, not a deviation from the plan's own text, which anticipates exactly this constraint via RESEARCH.md's Open Question 2 framing).

## Issues Encountered

**`go list -mod=readonly ./internal/upgrade/...` (single-package scope) does not surface a `DepsErrors` failure for an unresolvable import in one file of that package, even though `go list -mod=readonly ./...` (whole-repo scope, what `check:cross` actually runs) does.** Discovered while designing the Task 1 non-vacuity mutation: an initial test with a `//go:build windows` file importing a nonexistent package showed `go list ./internal/upgrade/...` exiting 0, which looked like the sweep couldn't fail. Re-running the exact `task check:cross` invocation (whole-repo `./...`) against the same mutation correctly failed with exit 201, naming `GOOS=windows GOARCH=amd64`. Resolved by always testing the real command scope (`./...`), not a package-scoped stand-in — documented here because it's a subtlety worth remembering if this guard is ever debugged again.

## Known Gaps (recorded per reporting-honesty, not left silent)

- **`release-please.yml` triggers only on `push: branches: [main]`.** This executor ran on a feature branch, so the following Task 2 acceptance criteria could not be verified from a real CI run and are marked `human_judgment: true` in the `coverage:` block above:
  - "A pushed run shows `pre-tag 6-target go list sanity sweep` green, containing an `Install Task` step and a `Run check:cross sweep` step."
  - "The gate still gates (RED then GREEN): push a branch carrying the same unresolvable-target mutation used in Task 1... confirm `pretag-gate` fails at the `Run check:cross sweep` step; revert and confirm green."
  - What WAS verified locally as a substitute: `task check:cross` (the exact command `pretag-gate` now runs) was proven RED-then-GREEN against a genuinely unresolvable target in Task 1, and `task lint:actions` passes against the edited workflow file. This proves the Taskfile-side behavior is correct; it does not prove the GitHub Actions execution environment (network access to the module proxy, `./.github/actions/install-task`'s composite-action resolution, runner permissions) behaves identically.
- **The `Install Task` step's real ubuntu-latest CI duration is unverified.** Local approximations (macOS, this session's network) were: ~2.8s with a warm `GOCACHE`, ~9.7s with `GOCACHE` cleared and `GOMODCACHE` warm, ~20.2s fully cold (both caches cleared, real network fetch of go.tool.mod's module graph). These corroborate RESEARCH.md's own prior figure (9.9s on "a fast M-series machine"). All three are comfortably under the plan's 60-second caching threshold, supporting the "add nothing" decision, but none is the actual `ubuntu-latest` runner measurement the plan's Task 2 text asks for ("run the workflow on the branch and read the Install Task step's duration from the run's timing"). A maintainer should confirm the real duration on the next push to `main` and add `actions/cache` keyed on `go.tool.sum` if it exceeds 60s.

**Recommended next step for the maintainer:** after this branch merges to `main` (or via a manual `workflow_dispatch` test if one is ever added to `release-please.yml`), confirm `pretag-gate` is green with the new step names, and read the actual `Install Task` step duration from the run's timing to close the two gaps above.

## TDD Gate Compliance

Task 3 carries `tdd="true"`, but its declared `<files>` is exactly one test file (`internal/upgrade/taskfile_shape_test.go`) — per this project's own `IS_BEHAVIOR_ADDING` predicate (tdd + `<behavior>` block + non-test source files), a test-only file set does not have a separate production file to implement against; the parsers (`parseGoreleaserCrossPairs`, `parseCheckCrossPairs`) ARE the guard's test infrastructure. RED/GREEN discipline was applied as the task's own explicitly-required non-vacuity mutation proofs instead of a synthetic test→feat commit split:
- Implemented the test + parsers together, ran the full suite: GREEN (the guarded behavior — `check:cross`'s target list — already existed correctly from Task 1's commit).
- Mutation (a): dropped `darwin/arm64` from `check:cross`'s `for pair in` line, confirmed via `git diff` the mutation landed, ran `TestCheckCrossMatchesGoreleaserTargets`: **RED**, failure named the missing pair exactly (`darwin/arm64`). Restored via the pre-edit backup, re-ran: **GREEN**.
- Mutation (b): added a 7th `builds:` entry (`freebsd/amd64`) to `.goreleaser.yaml`, confirmed via `git diff` the mutation landed, ran the test: **RED**, failure named the extra pair exactly (`freebsd/amd64`). Restored, re-ran the full `internal/upgrade` package: **GREEN**.
- Both `no test(...)→feat(...)` git commit split and both required-by-acceptance-criteria non-vacuity directions are satisfied; only one commit (`356af3f`, typed `test`) was needed since no separate production code changed in this task.

No RED/GREEN gate commits were skipped; this is a documented, plan-consistent interpretation for a test-only task, not a compliance gap.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `check:cross` is live and locally runnable (`GOWORK=off go tool -modfile=go.tool.mod task check:cross`), and `release-please.yml`'s `pretag-gate` job calls it — the sweep now has exactly one definition site.
- The set-equality guard (`TestCheckCrossMatchesGoreleaserTargets`) will fail loudly if a future release target is added to `.goreleaser.yaml` without also being added to `check:cross`, or vice versa.
- Blocker for full confidence: the real CI-runner behavior of `pretag-gate` (green run, Install Task duration, gate-still-gates under a real push) needs a maintainer to confirm on the next push to `main` — see Known Gaps above. This does not block merging this plan's changes; it is a follow-up verification step outside this executor's reach.
- No files under `internal/bench/`, `tools/bench/`, or `.github/workflows/bench.yml` were touched — confirmed disjoint from the concurrent 10-04 executor's territory.

## Self-Check: PASSED

- FOUND: Taskfile.yml
- FOUND: .github/workflows/release-please.yml
- FOUND: internal/upgrade/taskfile_shape_test.go
- FOUND: docs/RELEASE-PROCEDURES.md
- FOUND commit: b760f86
- FOUND commit: c42a2dd
- FOUND commit: 356af3f

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-02*
