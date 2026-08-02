---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 01
subsystem: infra
tags: [taskfile, go-tool, goreleaser, actionlint, github-actions, namespace-runners, ci]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: ci.yml's existing goreleaser-check and actionlint jobs (the two jobs this plan rewires), and the six required-status-check job names this plan's guard test locks down
provides:
  - go.tool.mod / go.tool.sum — isolated tool modfile pinning task v3.52.0 + goreleaser v2.17.1 via `go get -tool`, built via `-modfile`, never merged into root go.mod
  - go.tool-lint.mod / go.tool-lint.sum — second isolated modfile pinning actionlint v1.7.12 (D-03 MVS-conflict split, re-demonstrated live)
  - .github/actions/install-task composite action — builds `task` from the module proxy only, no marketplace action or install script
  - Taskfile.yml — the single local/CI command-definition site (default, check:goreleaser, lint:actions targets so far)
  - .github/actionlint.yaml — self-hosted-runner label config for the two namespace-profile-* labels (Rule 3 deviation, needed for Task 2's actionlint job to pass against its own change)
  - ci.yml's goreleaser-check and actionlint jobs rewired onto Namespace runners + nscloud-cache-action + install-task, invoking `task <target>` instead of inline `go install`/`run:` bodies
  - internal/upgrade/taskfile_shape_test.go — TestRequiredCheckNamesPreserved and TestToolModfilesRemainIsolated shape guards
affects: [10-02, 10-03, 10-04, 10-05, 10-06, 10-07]

# Actuals (#2632)
actuals:
  tokens: 10952
  tasks: 3
  commits: 3

tech-stack:
  added:
    - github.com/go-task/task/v3 v3.52.0 (isolated go.tool.mod, not the root module)
    - github.com/goreleaser/goreleaser/v2 v2.17.1 (isolated go.tool.mod, matches ci.yml's pre-existing pin)
    - github.com/rhysd/actionlint v1.7.12 (isolated go.tool-lint.mod, same version ci.yml already pinned)
    - namespacelabs/nscloud-cache-action@c5f8dab7560444c4bf8dbc64f1b203431873c547 (# v1.6.1)
  patterns:
    - Isolated `-modfile` tool bootstrap (GOWORK=off go tool -modfile=<file> <cmd>), two separate modfiles because a shared one does not compile (MVS version-bid conflict)
    - Composite action builds a pinned tool from the Go module proxy only, never a marketplace action or install script (D-04)
    - Taskfile as the single command-definition site; CI `run:` steps become `task <target>` invocations, not reimplementations
    - parseX/mustX shape-guard idiom (from release_workflow_shape_test.go / pr_title_lint_test.go) extended to a new file

key-files:
  created:
    - go.tool.mod
    - go.tool.sum
    - go.tool-lint.mod
    - go.tool-lint.sum
    - .github/actions/install-task/action.yml
    - .github/actionlint.yaml
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
  modified:
    - .github/workflows/ci.yml

key-decisions:
  - "go.tool.mod pins goreleaser v2.17.1 (matching ci.yml's pre-existing pin); release.yml's GORELEASER_VERSION: v2.17.0 mismatch is a pre-existing, deliberately untouched discrepancy per the plan's A1 flagged assumption — filed as a follow-up, not resolved here"
  - "Added .github/actionlint.yaml (not in the plan's files_modified) declaring namespace-profile-linux-amd64-{2x4,4x8} as self-hosted-runner labels — Rule 3 blocking-issue fix, discovered running actionlint against Task 1's own ci.yml change: actionlint's built-in runner-label list only knows GitHub-hosted names, so every namespace-profile-* runs-on: reported as unknown, which would have failed Task 2's actionlint job (one of the six required status checks) on its own diff"
  - "Updated one sentence of the actionlint job's pre-existing header comment (the 'installed via go install / no new pinned Action' claim) rather than keeping the whole block byte-verbatim, because this exact task's own diff makes that sentence false (it adds namespacelabs/nscloud-cache-action, a new pinned Action, and switches to go.tool-lint.mod). The plan's D-02 verbatim instruction governs run: command bodies; this is a factual-accuracy fix to a comment describing a mechanism this same commit removes, not a stylistic cleanup"

patterns-established:
  - "GO_TOOL / GO_TOOL_LINT Taskfile var indirection: every task invoking task/goreleaser/actionlint goes through {{.GO_TOOL}} or {{.GO_TOOL_LINT}}, never a bare binary name"
  - "Every CI job moved to a namespace-profile-* runner gets the same three-step tool-bootstrap prelude: Set up Go (cache: false) -> Cache Go modules and build (nscloud-cache-action, cache: go) -> Install Task (./.github/actions/install-task)"

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "Tracer: goreleaser-check CI job runs `task check:goreleaser`, built from go.tool.mod on a Namespace runner; job name unchanged; go list ./... unaffected"
    requirement: "DEV-01"
    verification:
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task check:goreleaser (exit 0, local)"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity: goreleaser check --config against a scratch .goreleaser.yaml with an added unknown top-level key (exit 1, error names the key) then restored (exit 0)"
        status: pass
    human_judgment: true
    rationale: "The plan's acceptance criteria also require observing a real pushed CI run on a namespace-profile-linux-amd64-2x4 runner (via gh api .../actions/jobs/<id> -> labels). This plan executed entirely on the local working tree with no push/PR — that criterion is UNVERIFIED, not passing. A human (or the next push/PR) must confirm the job actually schedules and completes on a Namespace runner before this deliverable is fully proven."
  - id: D2
    description: "actionlint CI job (one of six required status checks) moved to go.tool-lint.mod + task lint:actions on a Namespace runner; D-03's MVS-conflict split re-demonstrated live; required-check job name unchanged"
    requirement: "DEV-01"
    verification:
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task lint:actions (exit 0, local, against the real .github/workflows/*.yml)"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity: scratch workflow file with an undefined needs.<job>.outputs reference (exit 201, names the file) then deleted (exit 0)"
        status: pass
      - kind: other
        ref: "D-03 re-demonstration: scratch copy of go.tool.mod with the actionlint tool directive added reproduces the exact action_metadata.go:273:22: te.Errors[0].Error undefined compile error"
        status: pass
    human_judgment: true
    rationale: "Same live-CI-run gap as D1, with higher stakes: actionlint is one of the six branch-protection required checks, so a broken rewire blocks every PR until reverted. Needs confirmation on a real push/PR before being trusted as the merge gate."
  - id: D3
    description: "Two shape guards (TestRequiredCheckNamesPreserved, TestToolModfilesRemainIsolated) lock in the required-check-name set and the go.tool.mod/go.tool-lint.mod isolation, each demonstrated RED then GREEN against the real tree"
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestRequiredCheckNamesPreserved"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestToolModfilesRemainIsolated"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... (full package, count=1)"
        status: pass
    human_judgment: false

duration: 45min (estimated — session start time not captured by the record_start_time step; based on first-verification-to-last-commit span plus preceding exploration)
completed: 2026-08-01
status: complete
---

# Phase 10 Plan 1: Tracer — go.tool.mod, install-task, Taskfile, and Namespace runners proven on one real CI job Summary

**Proved the whole Phase-10 tool-bootstrap chain (isolated `-modfile` -> composite action -> Taskfile -> Namespace runner -> `task <target>`) end-to-end on `goreleaser-check`, then extended it to the required-check `actionlint` job, and locked both required-check names plus modfile isolation behind a new shape-guard test.**

## Performance

- **Duration:** ~45 min (estimated)
- **Completed:** 2026-08-01T20:26:11-04:00 (last task commit)
- **Tasks:** 3 / 3
- **Files modified:** 8 (7 created, 1 modified)

## Precondition Verification

This plan's `user_setup` block required the Namespace GitHub integration on `seanb4t/codegraph-go`. **Verified SATISFIED before dispatch, not re-checked during execution per the dispatch instructions:** the maintainer confirmed from the Namespace dashboard (2026-08-01, status "All good") that the `seanb4t` organization is enabled with "All repositories," corroborated independently by the sibling repo `holomush/holomush` running both `namespace-profile-linux-amd64-2x4` and `namespace-profile-linux-amd64-4x8` with green CI as recently as 2026-07-31. Task 1's own `<precondition>` gate (confirm a job requesting the label starts rather than queuing) was therefore not re-run as a local check — it can only be observed on a real push, which is exactly the gap flagged under D1/D2 above.

## Accomplishments

- `go.tool.mod` pins `task@v3.52.0` and `goreleaser@v2.17.1`, isolated from the root module (measured live: merging into root go.mod would add 237 net-new modules, 511 -> 748)
- `go.tool-lint.mod` pins `actionlint@v1.7.12` in a second isolated modfile; the D-03 MVS conflict that forces the split was re-demonstrated live in a scratch repro (verbatim `action_metadata.go:273:22` error reproduced), not merely cited
- `.github/actions/install-task` builds `task` from `go.tool.mod` via the Go module proxy only — no `arduino/setup-task`, no `taskfile.dev` install script
- `Taskfile.yml` carries `GO_TOOL`/`GO_TOOL_LINT` var indirection, a `default` task that only lists targets, `check:goreleaser`, and `lint:actions`
- `ci.yml`'s `goreleaser-check` (non-required) and `actionlint` (one of six required checks) jobs both rewired onto `namespace-profile-linux-amd64-2x4` with `nscloud-cache-action` + `install-task`, invoking `task <target>` instead of inline `go install`/shell bodies; both job `name:` fields verified byte-identical and set-equal to the live ruleset `20157557` required contexts
- `internal/upgrade/taskfile_shape_test.go` adds `TestRequiredCheckNamesPreserved` and `TestToolModfilesRemainIsolated`, each demonstrated RED (via a temporary mutation) then GREEN (restored) against the real tree, following the package's existing `parseX`/`mustX` idiom
- All live-verifiable acceptance criteria confirmed: `go list ./...` package set byte-identical before/after both modfile additions; `task` with no args lists targets and starts no build/test/lint process; both `task check:goreleaser` and `task lint:actions` exit 0 against the real repo state

## Task Commits

1. **Task 1: End-to-end "one CI job runs a task target" — goreleaser-check only** - `286f4aa` (feat)
2. **Task 2: Split the lint toolchain into go.tool-lint.mod and move the actionlint job** - `c53f011` (feat)
3. **Task 3: Shape guards — required-check names and tool-modfile isolation** - `8665c00` (test)

## Files Created/Modified

- `go.tool.mod` / `go.tool.sum` - isolated tool modfile pinning task + goreleaser
- `go.tool-lint.mod` / `go.tool-lint.sum` - isolated tool modfile pinning actionlint
- `.github/actions/install-task/action.yml` - composite action, builds `task` from the module proxy, adds it to PATH
- `.github/actionlint.yaml` - self-hosted-runner labels for the two namespace-profile-* runner classes (deviation, see below)
- `Taskfile.yml` - `default`, `check:goreleaser`, `lint:actions` targets
- `.github/workflows/ci.yml` - `goreleaser-check` and `actionlint` jobs rewired onto Namespace runners + `task`
- `internal/upgrade/taskfile_shape_test.go` - `TestRequiredCheckNamesPreserved`, `TestToolModfilesRemainIsolated`, plus edge-case and repudiation-guard tests

## Decisions Made

- Pinned `go.tool.mod`'s goreleaser to `v2.17.1` (matches `ci.yml`'s existing pin); left `release.yml`'s pre-existing `v2.17.0` mismatch untouched and recorded as a follow-up, per the plan's A1 flagged assumption
- Re-verified all four tool/action versions live against `proxy.golang.org` and the GitHub API before pinning (task v3.52.0, goreleaser v2.17.1, actionlint v1.7.12, `nscloud-cache-action` SHA `c5f8dab7...` -> tag `v1.6.1`) — all matched the plan's stated pins exactly, no drift found
- Added `.github/actionlint.yaml` (Rule 3) — see Deviations
- Updated one sentence of the actionlint job's pre-existing comment block rather than a full byte-verbatim carry-over — see Deviations

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `.github/actionlint.yaml` self-hosted-runner label config**
- **Found during:** Task 1, while verifying Task 1's own `ci.yml` diff against actionlint (ahead of Task 2 formally wiring the `actionlint` job, but the file this task changed is exactly what that job would lint)
- **Issue:** `actionlint` was run against `.github/workflows/ci.yml` as a sanity check on Task 1's diff and reported `namespace-profile-linux-amd64-2x4` as an unknown runner label — actionlint's built-in label list only knows GitHub-hosted runner names. Without a config entry, Task 2's `actionlint` job (one of the six branch-protection required checks) would fail against its own change, and `task lint:actions` would never exit 0 against the real repo state, contradicting Task 2's own acceptance criterion.
- **Fix:** Added `.github/actionlint.yaml` with `self-hosted-runner.labels: [namespace-profile-linux-amd64-2x4, namespace-profile-linux-amd64-4x8]` (both labels this phase's plans use, not just the one Task 1 introduces, so Task 2+ don't hit the same gap again).
- **Files modified:** `.github/actionlint.yaml` (new)
- **Verification:** `GOWORK=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/ci.yml` exits 0 after the config is added (was exit 1 with `[runner-label]` before)
- **Committed in:** `286f4aa` (Task 1 commit)

**2. [Rule 1 - Bug] Corrected a comment sentence in the actionlint job that this same task's diff would otherwise make false**
- **Found during:** Task 2
- **Issue:** The plan instructs keeping the actionlint job's pre-existing comment block "verbatim." That block's last sentence claimed actionlint is "Installed via `go install`... no new pinned Action in the supply chain" — but Task 2's own diff switches actionlint to `go.tool-lint.mod` (not `go install`) and adds `namespacelabs/nscloud-cache-action`, a new pinned Action, to this exact job. Keeping the sentence byte-identical would ship a comment the same commit falsifies.
- **Fix:** Rewrote only that final sentence to describe the new `go.tool-lint.mod`/install-task bootstrap; left the rest of the block (actionlint's necessity, the four-workflow-file scope, the CR-01/WR-02 citation) untouched.
- **Files modified:** `.github/workflows/ci.yml`
- **Verification:** Manual read-through; the D-02 "verbatim" instruction is read as governing `run:` command bodies (its stated purpose), not a comment describing a mechanism this task is actively changing.
- **Committed in:** `c53f011` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug/doc-accuracy)
**Impact on plan:** Both were necessary for the stated acceptance criteria to actually hold (Task 2's `task lint:actions exits 0` and the required-check `actionlint` job staying green). No scope creep beyond what those criteria required.

## Non-Vacuity Demonstrations (RED-then-GREEN), as performed

All five demonstrations the plan's acceptance criteria call for were actually run, not asserted:

1. **`task check:goreleaser` schema violation:** added `not_a_real_top_level_key: true` to a scratch copy of `.goreleaser.yaml`, confirmed the mutation landed via `grep`, ran `goreleaser check --config <scratch>` -> exit 1, error named the unknown key (`field not_a_real_top_level_key not found in type config.Project`); reverted -> exit 0.
2. **`task lint:actions` real finding:** added a scratch workflow file at `.github/workflows/zzz-actionlint-mutation-test.yml` with `${{ needs.nonexistent_job.outputs.foo }}` (valid YAML, real actionlint expression-type finding), confirmed via `rg`, ran `task lint:actions` -> exit 201, error named the file and the undefined property; deleted the file -> exit 0.
3. **D-03 MVS conflict re-demonstration:** copied `go.tool.mod`+`go.tool.sum` to a scratch directory, added the `actionlint` tool directive, ran `go build`. Reproduced the **exact** cited error (`action_metadata.go:273:22: te.Errors[0].Error undefined (type string has no field or method Error)` plus three more errors in `parse.go`) — did not unexpectedly succeed.
4. **`TestRequiredCheckNamesPreserved`:** shortened `ci.yml`'s goreleaser-check job name to `goreleaser check`, confirmed via `rg`, ran the test -> FAIL naming `goreleaser check (config validation, DIST-01)` as missing; restored via backup -> PASS.
5. **`TestToolModfilesRemainIsolated`:** added a `tool (...)` block for `github.com/go-task/task/v3/cmd/task` to a backed-up copy of the root `go.mod`, confirmed via `rg`, ran the test -> FAIL (root go.mod declares a tool directive). Note: the `sed` mutation used inserted the block at two `require (` occurrences in `go.mod` rather than one, so the failure message's reported package list was garbled (`[( (]`) rather than clean — the test still correctly detected and rejected the presence of an unwanted tool directive either way; the garbling is an artifact of the throwaway mutation script, not the parser. Restored from the `go.mod.bak` backup (`git diff go.mod` confirmed zero residual diff) -> PASS.

**Not performed — explicitly flagged, not implied passing:** the plan's acceptance criteria for both Task 1 and Task 2 also require observing **a real pushed CI run** (`gh api .../actions/jobs/<id>` showing the job executed on a `namespace-profile-linux-amd64-*` runner with conclusion `success`, containing an `Install Task` step). This plan executed entirely against the local working tree with no push or PR opened — that specific criterion is **unverified**, not passing. See `coverage.D1`/`D2` `human_judgment: true` above.

## Issues Encountered

- **Pre-existing, out-of-scope linter finding (not introduced by this plan):** `mustWorkflowTopLevelName` at `internal/upgrade/release_workflow_shape_test.go:44` is defined but has zero call sites anywhere in the package (only `parseWorkflowTopLevelName`, its non-`must` counterpart, is actually called, at line 315). This function predates this plan — it landed in commit `7f60822` ("feat(09-01): release-please spine + non-vacuous workflow-shape drift guard"), untouched by any of this plan's three commits (`git log 286f4aa~1..HEAD -- internal/upgrade/release_workflow_shape_test.go` is empty). No plan file under this phase (`10-02` through `10-07`) references it by name, so there is no direct evidence it is deliberate forward-looking scaffolding for a later Phase-10 plan; equally, nothing here confirms it is safe to delete. Left untouched per the scope-boundary rule (only auto-fix issues directly caused by the current task's changes) — flagged here for whoever plans/executes the next 10-0x plan to decide.
- **Modernization hints in the new test file, consciously not applied:** a `gopls`/`modernize`-class analyzer flags `strings.Split` at `taskfile_shape_test.go:70` and `:139` as candidates for `strings.SplitSeq` (Go 1.24+ iterator form) and the classic `for i := 0; i < len(lines); i++` at `:108` as a candidate for `for i := range len(lines)`. Not applied: this file's explicit design goal (per the plan's `read_first`) was to match the existing `parseX`/`mustX` idiom in `release_workflow_shape_test.go` and `pr_title_lint_test.go`, both of which use classic `strings.Split` + index/range loops throughout. Adopting the newer idiom here would make this file diverge in style from the two files it was modeled on, for a purely cosmetic gain. No functional or lint-blocking issue — these are informational hints, not build-breaking warnings.

## User Setup Required

None during execution — the plan's `user_setup` block (Namespace GitHub integration) was verified satisfied by the maintainer before this plan was dispatched (see Precondition Verification above).

## Next Phase Readiness

- The proven chain (`go.tool.mod` -> `install-task` -> `Taskfile.yml` -> Namespace runner -> `task <target>`) is ready for 10-02 onward to extend to the remaining `ci.yml` jobs, `bench.yml`, `release-please.yml`'s `pretag-gate`, and `release.yml`.
- **Blocker/concern carried forward:** neither of this plan's two rewired jobs has been observed running on a real pushed CI event. The next action against this branch (a push or PR) should be watched for both jobs actually scheduling on `namespace-profile-linux-amd64-2x4` and completing successfully — if the Namespace integration turns out not to serve the label despite the pre-verified dashboard state, both jobs will queue indefinitely per Task 1's own `<precondition>` warning, and that failure mode is silent by construction (no error, just a queue).
- `.github/actionlint.yaml` now exists at repo root-adjacent `.github/` and will need its `self-hosted-runner.labels` list extended if a future plan introduces a new Namespace runner-profile label beyond the two already declared.

## Self-Check: PASSED

All 8 created/modified deliverable files confirmed present on disk; all 3 task commits (`286f4aa`, `c53f011`, `8665c00`) plus this summary's own commit (`dcddafb`) confirmed in `git log`.

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-01*
