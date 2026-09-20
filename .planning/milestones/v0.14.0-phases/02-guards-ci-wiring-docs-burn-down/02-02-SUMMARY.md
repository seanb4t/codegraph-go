---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 02
subsystem: ci-cd
tags: [github-actions, syft, sbom, govulncheck, gonum, cytoscape-layout-guard, dependency-bump]

# Dependency graph
requires:
  - phase: 02-01
    provides: "02-MUTATION-LOG.md's Family (a) shape (pre-mutation gate / mutation / observed failure / revert / byte-clean proof / green control) that Family (b) here follows"
provides:
  - "Two new `test`-job steps in `.github/workflows/ci.yml`: `Install syft` (SHA-pinned, same as release.yml/linux-cross-canary.yml) and `Gonum dependency-shape guard (GRD-11/GRF-10)` / `No-force-layout guard (GRD-11/GRF-06)`, both calling their existing Taskfile targets — no ruleset edit needed since `test` is already a required context"
  - "`02-MUTATION-LOG.md` Family (b): a planted forbidden-layout literal proven to fail the wired `check:no-force-layout` step, reverted byte-clean, re-verified green"
  - "A pre-existing, unrelated grpc vulnerability (GO-2026-6348) fixed via an indirect dependency bump, discovered only because this plan required `check:gonum` to actually pass locally"
affects: ["02-03 (Ruleset drift check, placed immediately after these steps in wave 1)", "02-04 (Family c appends to the same 02-MUTATION-LOG.md)"]

# Actuals (#2632)
actuals:
  tokens: 3607
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: ["anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 (GitHub Action, reused pin from release.yml)"]
  patterns: ["CI step run: body is a bare `task <target>` call, never re-derived guard logic (D-04/Phase 7 D-07)", "rationale comment above every ci.yml step (house style)"]

key-files:
  created: []
  modified:
    - .github/workflows/ci.yml
    - go.mod
    - go.sum
    - .planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md

key-decisions:
  - "Bumped google.golang.org/grpc (indirect) v1.82.1 -> v1.83.2 in its own commit before the ci.yml commit, so the CI-wiring commit's diff stays scoped to ci.yml only (satisfies the plan's own file-scope verify) while still fixing the blocking vulnerability check:gonum's govulncheck half found on the clean tree."
  - "Documented two plan-authoring assumption gaps in the automated <verify> scripts (a whole-file, non-diff-scoped `continue-on-error` grep in Task 1, and a self-test-must-stay-PASS assumption in Task 2 that does not hold because the scanner's self-test scans the real tree jointly with its injected fixtures) rather than treating either as an implementation defect — the acceptance_criteria for both tasks, which are diff-scoped and outcome-scoped respectively, are met exactly as written."

requirements-completed: [GRD-11]

coverage:
  - id: D1
    description: "syft + check:gonum + check:no-force-layout wired into ci.yml's test job in house style, after Set up Node and in the correct precondition order, with shape tests and actionlint green and both target commands passing locally with their positive-control lines printed"
    requirement: GRD-11
    verification:
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: other
        ref: "task lint:actions (actionlint over .github/workflows/*.yml)"
        status: pass
      - kind: other
        ref: "task check:gonum (govulncheck source-mode + SBOM + cgo-closure halves, all three positive controls firing)"
        status: pass
      - kind: other
        ref: "task check:no-force-layout (--self-test then real scan)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Planted forbidden-layout literal makes the exact wired step command (task check:no-force-layout) fail naming the planted file; byte-clean revert; green re-run; recorded as Family (b) in 02-MUTATION-LOG.md"
    requirement: GRD-11
    verification:
      - kind: other
        ref: "task check:no-force-layout with web/src/lib/__planted-no-force-layout__.ts present (exit 201, names the file) then removed (exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-09-16
status: complete
---

# Phase 2 Plan 2: CI Wiring for check:gonum and check:no-force-layout Summary

**Wired `check:gonum` and `check:no-force-layout` into `ci.yml`'s already-required `test` job via one SHA-pinned `syft` install and two bare `task <target>` steps, then proved the wiring can fail with a planted forbidden-layout literal — discovering and fixing a real, unrelated grpc CVE along the way.**

## Performance

- **Duration:** 55 min
- **Completed:** 2026-09-16T02:32:41Z
- **Tasks:** 2/2 completed
- **Files modified:** 4 (.github/workflows/ci.yml, go.mod, go.sum, 02-MUTATION-LOG.md)

## Accomplishments
- `ci.yml`'s `test` job now runs `Install syft`, `Gonum dependency-shape guard (GRD-11/GRF-10)`, and `No-force-layout guard (GRD-11/GRF-06)` — in that order, after `Set up Node` and before the `Ruleset drift check (GRD-12)` step 02-03 placed in wave 1. Both guards were previously local-only Taskfile targets (STATE.md's Phase-11 blocker); since `test` is already a required status check, they now block merges with no ruleset edit and no fixture change (D-04).
- Both `run:` bodies are bare `task <target>` calls — `TestWorkflowRunBodiesInvokeTask` accepts them with no `runBodyExceptions` entry needed, and `task lint:actions` (actionlint) passes on the edited workflow.
- Proved the wiring is not vacuous: planted `const layout = { name: 'cose' };` as an untracked file under `web/src/lib/`, ran the exact command the CI step runs, watched it fail naming the planted file (and, as a bonus, watched the scanner's own `--self-test` also correctly fail because it scans the real tree jointly with its injected fixtures), reverted byte-clean, and re-ran green. Recorded as Family (b) in `02-MUTATION-LOG.md`.
- Discovered and fixed a real, pre-existing vulnerability (GO-2026-6348, HTTP/2 DATA frame fragmentation heap exhaustion in `google.golang.org/grpc`, reachable via the `sigstore-go` dependency chain used by `internal/upgrade`) that would have made the newly-wired `check:gonum` step fail on `main` from the moment it merged, for a reason unrelated to this plan's guards.

## Task Commits

Each task was committed atomically (plus one deviation-fix commit ahead of Task 1's commit):

1. **[Deviation] Bump google.golang.org/grpc to v1.83.2 for GO-2026-6348** - `0e9ffb5b` (fix)
2. **Task 1: Wire syft + both guards into the `test` job** - `e1aba9ad` (ci)
3. **Task 2: Planted-violation positive control, Family (b)** - `ea1adc96` (docs)

## Files Created/Modified
- `.github/workflows/ci.yml` - three new `test`-job steps (`Install syft`, `Gonum dependency-shape guard (GRD-11/GRF-10)`, `No-force-layout guard (GRD-11/GRF-06)`)
- `go.mod` / `go.sum` - `google.golang.org/grpc` v1.82.1 -> v1.83.2 (indirect), plus its transitively-moved `golang.org/x/crypto`, `golang.org/x/net`, `golang.org/x/text`, `google.golang.org/genproto/googleapis/rpc`
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` - appended `## Family (b) — GRD-11` after 02-01's `## Family (a) — GRD-09`, unchanged

## Decisions Made
- The grpc CVE fix was committed separately from the `ci.yml` wiring commit, specifically so Task 1's own file-scope `<verify>` gate (`git show --format= --name-only HEAD` must equal exactly `.github/workflows/ci.yml`) still passes. The fix commit precedes the wiring commit in history.
- Two automated `<verify>` scripts in the plan carried assumptions that didn't hold against the real tree/file state (see Deviations below); both are documented in `02-MUTATION-LOG.md` and here rather than worked around by changing scanner or workflow behavior, since the acceptance_criteria (the actual gate) were met exactly as written in both cases.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed pre-existing grpc vulnerability (GO-2026-6348) blocking `check:gonum`'s local PASS**
- **Found during:** Task 1 (proving `task check:gonum` passes locally on the clean tree, a hard requirement of the plan's own acceptance criteria)
- **Issue:** `govulncheck` (source mode, main module) found GO-2026-6348 reachable from `internal/upgrade` via the `sigstore-go` -> `rekor-tiles` -> `google.golang.org/grpc` dependency chain. This is entirely unrelated to the CI-wiring change itself, but wiring an already-red check onto every PR's required `test` job would have broken CI for every subsequent PR for a reason this phase's guards were never meant to catch.
- **Fix:** `go get google.golang.org/grpc@v1.83.2` (patched release), which also moved `golang.org/x/crypto`, `golang.org/x/net`, `golang.org/x/text`, and `google.golang.org/genproto/googleapis/rpc` forward. No source code changed. `go mod tidy` could not complete cleanly (unrelated pre-existing `tree-sitter-swift` test-dependency resolution issue on `go.mod`'s `retract`/replace shape), so the targeted `go get` result was kept as-is rather than forcing a full tidy.
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `go build ./...` clean; `go test ./internal/upgrade/... -count=1` passes (26 tests, no regressions); re-ran `task check:gonum` — `check:gonum: PASS` with all three positive-control lines printed (govulncheck clean, SBOM gonum+pebble present exactly once each, cgo closure 0/0 plus tree-sitter positive control).
- **Committed in:** `0e9ffb5b` (separate commit, ahead of the Task 1 `ci.yml` commit)

**2. [Rule 3 - Blocking] Freed critical disk space (`go clean -cache`) to unblock local builds**
- **Found during:** Task 1, immediately after the grpc bump — `go build ./...` failed with "no space left on device" (the machine's Go build cache had grown to 181GB against ~900MB free disk).
- **Issue:** Not caused by this plan's changes; a pre-existing environment condition that blocked any further local verification.
- **Fix:** `go clean -cache` (safe, rebuildable cache only — no project or git-tracked files touched), freeing the disk to ~120GB available. Waited for the cache-clean process to fully exit before re-attempting the build (an initial retry raced against the still-running clean and produced transient "no such file" cache errors, resolved once the clean process was confirmed finished).
- **Files modified:** none (build cache only)
- **Verification:** `go build ./...` clean on retry after the clean process exited.
- **Committed in:** N/A (no git-tracked change)

---

**Total deviations:** 2 auto-fixed (2 blocking). **Impact on plan:** Both were necessary to satisfy Task 1's own acceptance criteria (`task check:gonum` must PASS locally on the clean tree); neither touched `.github/workflows/ci.yml`, `Taskfile.yml`, or any scanner script. No scope creep into the guard logic itself.

## Issues Encountered

- **Task 1's third `<automated>` verify command is not diff-scoped.** It asserts `! rg -q 'continue-on-error' .github/workflows/ci.yml` against the WHOLE file, but `ci.yml` already carries a pre-existing, unrelated `continue-on-error: true` (for a documented macOS/arm64 cross-compile leg, predating this plan). `git diff -- .github/workflows/ci.yml | rg '^\+' | rg 'continue-on-error'` confirms this plan's diff adds none. The task's own `<acceptance_criteria>` ("No `continue-on-error`... change anywhere in the diff") is correctly diff-scoped and is met; only the literal automated command as written cannot pass regardless of implementation. Treated as a plan-authoring gap, not fixed (fixing it would mean editing the PLAN.md, out of an executor's scope) — noted here for the phase verifier.
- **Task 2's first `<automated>` verify command asserts `self-test: PASS` even with the planted violation present.** `check-no-force-layout.mjs`'s `selfTest()` scans the real `web/src` tree jointly with its own injected fixtures and asserts exactly one forbidden match; with the planted file also present, it correctly reports `self-test: FAIL` (two matches, not one) — a stronger, not weaker, demonstration that the scan is sensitive to real-tree state. The full literal command was still run and its other three sub-assertions (byte-clean before/after, non-zero exit, planted path named) all passed; only the `self-test: PASS` substring check fails, which is a plan-authoring assumption about self-test isolation that does not hold given the script's actual (correct) design. Recorded verbatim in `02-MUTATION-LOG.md` Family (b) rather than glossed over.

Neither issue required any change to `ci.yml`, `Taskfile.yml`, or `web/scripts/check-no-force-layout.mjs` — both are documentation-only observations about the plan's own verify-script literal text versus its acceptance_criteria, which are the true gate and were met in full.

## User Setup Required

None - no external service configuration required.

## TDD Note

This plan is `type: execute`, not `type: tdd`. Its RED/GREEN shape is the planted-violation
demonstration in Task 2 (the wired step's command failing on a real forbidden-layout literal),
not a unit-test RED commit — there is no application source code being test-driven here, only a
CI workflow wiring and a mutation-log demonstration. No RED test commit was made or expected.

## Next Phase Readiness
- `check:gonum` and `check:no-force-layout` now gate every pull request through the existing required `test` context; a PR that breaks either guard fails CI with no further setup.
- Plan 02-03 (Ruleset drift check, GRD-12) already placed its step immediately after these three in wave 1 — line order in `ci.yml` is `Set up Node` < `Install syft` < `check:gonum` < `check:no-force-layout` < `Ruleset drift check (GRD-12)` < `Test subprocess integration harness`, verified.
- Plan 02-04 will append Family (c) to `02-MUTATION-LOG.md` after this plan's Family (b) — no conflict, the file's tail marker was updated to point at 02-04 only.
- The grpc CVE fix is a pure dependency bump with no source changes; no blocker for downstream plans.

---
*Phase: 02-guards-ci-wiring-docs-burn-down*
*Completed: 2026-09-16*

## Self-Check: PASSED
- FOUND: `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-02-SUMMARY.md`
- FOUND commit `0e9ffb5b` (grpc bump)
- FOUND commit `e1aba9ad` (ci.yml wiring)
- FOUND commit `ea1adc96` (Family b mutation log)
- Re-ran acceptance criteria: step-name uniqueness (3/3), line ordering (Set up Node < Install syft < check:gonum < check:no-force-layout < Ruleset drift check < integration harness), syft SHA pin count 1, `go test ./internal/upgrade/` shape guards PASS, `task lint:actions` clean, `task check:gonum` PASS, `task check:no-force-layout` PASS, Task 1 commit touches only `ci.yml`, Family (b) present in `02-MUTATION-LOG.md` alongside untouched Family (a), no CI-skip marker in any commit message, planted file never tracked.
