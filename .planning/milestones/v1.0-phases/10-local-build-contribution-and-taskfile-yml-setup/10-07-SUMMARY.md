---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 07
subsystem: infra
tags: [taskfile, ci, contributing, workflow-guard, go-yaml]

# Dependency graph
requires:
  - phase: 10-local-build-contribution-and-taskfile-yml-setup (plans 01-06)
    provides: Taskfile.yml's full target set, go.tool.mod/go.tool-lint.mod isolated tool modfiles, ci.yml and release-please.yml rewired to call `task <target>` for every step
provides:
  - "CONTRIBUTING.md pointer paragraph naming the task targets a contributor actually types (build/test/lint/check:cross/vet:daemon-windows/check:reproducibility:arm64), without rewriting the already-correct CGo/zig/mingw-w64 prose"
  - "TestWorkflowRunBodiesInvokeTask: an enforced, non-vacuous single-definition guard over ci.yml's test/actionlint/goreleaser-check/reproducibility/perf-regression jobs and release-please.yml's pretag-gate — every run: body must be `task <target>`, exceptions require a matched, reasoned fixture entry"
  - "A recorded, real clean-checkout proof that `task build`, `task test`, and `task lint` succeed from a fresh clone with only Go + a C toolchain on PATH (no task/zig/mingw-w64/goreleaser/actionlint pre-installed)"
affects: [ci, release-tooling, contributing-docs]

# Actuals (#2632)
actuals:
  tokens: 4468
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Real YAML-decoded workflow-step guard (yaml.Unmarshal into workflowFileYAML{Jobs map[string]workflowJobYAML{Steps []workflowRunStep}}) rather than a line-regex scan, so multi-line run: block scalars and step ordering decode correctly"
    - "Composite exception-key fixture (workflow\\x00job\\x00step -> reason) with a mandatory match-and-reason double-check: every exception must both carry a non-empty reason AND be observed matching a real step, closing the 'stale allowlist entry' failure mode in both directions"

key-files:
  created: []
  modified:
    - CONTRIBUTING.md
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "ROADMAP criterion 2 recorded as pre-satisfied on evidence (CONTRIBUTING.md lines 57-71, 156-line file, pre-existing before this phase) rather than rewritten — per D-00, only a pointer paragraph was added after the existing actionlint sentence"
  - "bench.yml and release.yml deliberately excluded from TestWorkflowRunBodiesInvokeTask's inScopeJobs fixture — both already carry their own documented, in-file D-01 exception comments (rebless/headtohead/diagnostic jobs' inline `go run ./tools/bench/runner`; release.yml's native build matrix, D-08) decided in earlier plans of this phase; including them would fail the guard for reasons the project already settled"
  - "task build's own desc is 'compile check only, no binary retained' — the plan's literal '`./codegraph` appears after `task build`' acceptance criterion does not hold for that target by design, so the binary-appears proof was additionally run against `task build:release` (which does produce ./codegraph) rather than silently skipped or the criterion silently reinterpreted without comment"

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "CONTRIBUTING.md points contributors at the task targets (task/task build/task test/task lint/cross-toolchain targets/check:cross) without rewriting the pre-existing, already-correct CGo/zig/mingw-w64 prose"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "git diff CONTRIBUTING.md shows only added lines in the Building section; target-name subset check against Taskfile.yml (comm -23) empty; task lint exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestWorkflowRunBodiesInvokeTask enforces the D-01/D-02 single-definition property over ci.yml's test/actionlint/goreleaser-check/reproducibility/perf-regression jobs and release-please.yml's pretag-gate, with a reasoned, matched exception fixture for the two legitimate non-task steps"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestParseWorkflowJobSteps_MissingJobIsError, TestParseWorkflowJobSteps_NoJobsIsError, TestParseWorkflowJobSteps_ZeroStepsIsError, TestCheckStepInvokesTask_ForbiddenGoInvocationNamesVerb, TestCheckStepInvokesTask_ExceptedStepPasses, TestCheckStepInvokesTask_UsesStepPasses, TestStripRunBodyNoise_KeepsHashInsideCommand, TestRunBodyExceptionsHaveReasons_EmptyReasonIsError"
        status: pass
    human_judgment: false
  - id: D3
    description: "Non-vacuity of the guard, demonstrated three ways: RED against the phase's base commit 82ffd60, RED-then-GREEN against an injected regression (re-inlined actionlint step), RED-then-GREEN against a stale exception entry"
    requirement: DEV-01
    verification:
      - kind: other
        ref: "manual proof, not committed to the tree — see 'Non-Vacuity Proofs' section below for the exact commands and captured output"
        status: pass
    human_judgment: false
  - id: D4
    description: "ROADMAP criterion 3 (a clean checkout can build, test, and lint via task targets alone) proven by executing task build/task test/task lint from a real scratch clone with task/zig/mingw-w64/goreleaser/actionlint absent from PATH"
    requirement: DEV-01
    verification:
      - kind: other
        ref: "manual scratch-clone run, not committed to the tree — see 'Clean-Checkout Proof' section below for commands, exit codes, and durations"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-08-02
status: complete
---

# Phase 10 Plan 07: Close the Phase — CONTRIBUTING Pointer, Single-Definition Guard, Clean-Checkout Proof Summary

**CONTRIBUTING.md now points contributors at the task targets, `TestWorkflowRunBodiesInvokeTask` makes the single-definition property an enforced (and demonstrably non-vacuous) invariant over the rewired CI jobs, and a real scratch-clone run proves `task build`/`task test`/`task lint` succeed with only Go and a C toolchain on `PATH`.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-02 (session start)
- **Completed:** 2026-08-02T19:13:29Z
- **Tasks:** 3
- **Files modified:** 2 (CONTRIBUTING.md, internal/upgrade/taskfile_shape_test.go)

## Accomplishments
- Added a pointer paragraph to `CONTRIBUTING.md`'s Building section naming every task target a contributor needs (`task`, `task build`, `task test`, `task lint`, `task vet:daemon-windows`, `task check:reproducibility:arm64`, `task check:cross`), leaving the pre-existing CGo/zig/mingw-w64 prose and the "Never do these" section byte-identical.
- Added `TestWorkflowRunBodiesInvokeTask` plus supporting YAML-decoded parsers (`parseWorkflowJobSteps`, `checkStepInvokesTask`, `stripRunBodyNoise`, `validateRunBodyExceptions`) and 8 edge-case tests to `internal/upgrade/taskfile_shape_test.go`, enforcing that every step's `run:` body across 6 in-scope jobs is a single `task <target>` call unless matched to a reasoned exception fixture entry.
- Demonstrated the guard non-vacuous three separate ways (see below), all reverted cleanly with the working tree confirmed clean between proofs.
- Ran a real scratch-clone proof: cloned the branch outside the repo, bootstrapped `task` from `go.tool.mod` with `task`/`zig`/`goreleaser`/`actionlint`/mingw-w64 absent from `PATH`, and ran `task build`, `task test`, `task lint` (plus a supplementary `task build:release`) to completion.

## Task Commits

Each task was committed atomically:

1. **Task 1: Point CONTRIBUTING.md at the task targets** - `5ee053f` (docs)
2. **Task 2: Enforce the single-definition property** - `3c716ff` (test)
3. **Task 3: Prove a clean checkout builds, tests, and lints through task alone** - no commit (verification-only; no repository files modified per the plan's own file list)

**Plan metadata:** (this commit, following)

## Files Created/Modified
- `CONTRIBUTING.md` - Added a 23-line pointer paragraph after the existing actionlint sentence in the Building section, naming the task targets; no existing line touched.
- `internal/upgrade/taskfile_shape_test.go` - Added `TestWorkflowRunBodiesInvokeTask`, its fixtures (`inScopeJobs`, `runBodyExceptions`), its supporting parsers (`parseWorkflowJobSteps`, `checkStepInvokesTask`, `stripRunBodyNoise`, `validateRunBodyExceptions`, `runBodyExceptionKey`), and 8 edge-case tests (+331 lines).

## ROADMAP Criterion 2 — Pre-Satisfaction Record (D-00)

**Criterion 2 was satisfied before this phase began and was NOT redone.** `CONTRIBUTING.md`'s `## Building` section (lines 57-71 in the pre-phase file, unmodified by this plan) already documented the CGo requirement, linked `PARSER-DECISION.md`, and named both `zig` (cross-builds) and `mingw-w64` (Windows vet) — the exact content the criterion asks for. It landed during OSS-readiness work outside any phase plan. This plan's only CONTRIBUTING.md change is the 23-line pointer paragraph directing contributors from that existing prose to the concrete task target names; the CGo/zig/mingw-w64 content itself is byte-identical before and after (confirmed via `git diff` showing zero removed/modified lines).

## Non-Vacuity Proofs (Task 2)

All three proofs were run against the real working tree, captured, and reverted — confirmed via `git status --short` returning empty after each. None of these edits are committed; they exist only as this record.

### 1. RED against the phase's base commit (82ffd60)

Temporarily replaced `.github/workflows/ci.yml` and `.github/workflows/release-please.yml` with their content at commit `82ffd60` ("build(taskfile): add Taskfile.yml + isolated tool modfiles, route CI through task (#19)" — the commit that merged plans 10-01/02/03), then ran:

```
go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask -v
```

Result: **FAIL**, naming all four offending steps that predated this phase's later task-ification work (plans 10-04/05/06):

```
taskfile_shape_test.go:1073: ci.yml job "reproducibility" step "linux/amd64 double build (BLOCKING — canonical target, D-03)" invokes "go build" directly instead of a task target — single-definition property violated (D-01)
taskfile_shape_test.go:1073: ci.yml job "reproducibility" step "linux/arm64 double build (REPORTED ONLY, non-blocking — D-03 cross-target)" invokes "go build" directly instead of a task target — single-definition property violated (D-01)
taskfile_shape_test.go:1073: ci.yml job "perf-regression" step "Perf-regression gate (offline synthetic 120k-file corpus)" invokes "go run" directly instead of a task target — single-definition property violated (D-01)
taskfile_shape_test.go:1073: release-please.yml job "pretag-gate" step "6-target go list -mod=readonly sweep"'s run: body is not a single 'task <target>' call after stripping comments/blanks (D-01): "set -euo pipefail\nfor pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do\n..."
```

Restored the current files and confirmed **PASS** again.

### 2. RED-then-GREEN against an injected regression

Temporarily edited `ci.yml`'s `actionlint` job, replacing `run: task lint:actions` with the raw inline command (`GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml`). Ran the guard:

```
taskfile_shape_test.go:1073: ci.yml job "actionlint" step "Run actionlint"'s run: body is not a single 'task <target>' call after stripping comments/blanks (D-01): "GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml"
```

**FAIL**, naming the exact workflow/job/step. Reverted the edit and confirmed `git diff .github/workflows/ci.yml` was empty, then re-ran and confirmed **PASS**.

### 3. RED-then-GREEN against a stale exception entry

Temporarily added a bogus `runBodyException` entry (`ci.yml/test/"This step does not exist in any workflow (stale-exception proof)"`, with a non-empty reason) to the test file's fixture. Ran the guard:

```
taskfile_shape_test.go:1087: exception ci.yml/test/"This step does not exist in any workflow (stale-exception proof)" was never matched against a real step in an in-scope job — a stale exception silently widens the allowlist (T-10-07-01); fix the step name or remove the entry
```

**FAIL**. Removed the bogus entry, confirmed `git diff internal/upgrade/taskfile_shape_test.go` showed only the permanent +331-line addition (no stray fixture entry left behind), and re-ran the full package suite (`go test ./internal/upgrade/...`) — **all PASS**.

## Clean-Checkout Proof (Task 3, ROADMAP criterion 3)

Cloned the current branch (`gsd/v1.0-drop-in-parity-human-ux` at commit `3c716ff`) into a scratch directory outside the working tree via `git clone --branch ... file:///Volumes/Code/github.com/seanb4t/codegraph-go <scratch>`. Constructed an isolated `PATH` containing only a symlink to the real `go` binary (`/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go`) plus `/usr/bin:/bin:/usr/sbin:/sbin` — confirmed via `command -v` that `task`, `zig`, `goreleaser`, `actionlint`, and `x86_64-w64-mingw32-gcc` all resolved to nothing on this `PATH`, while `clang` (`/usr/bin/clang`, the system C toolchain) and `git` (`/usr/bin/git`) remained available. Used isolated `GOCACHE`/`GOPATH` directories (not the developer machine's warm caches) so every download and compile in this proof was genuinely cold.

| Step | Command | Exit | Duration |
|---|---|---|---|
| Bootstrap `task` | `GOWORK=off go build -modfile=go.tool.mod -o <scratch>/bin/task github.com/go-task/task/v3/cmd/task` | 0 | 18s |
| List targets | `task` (no args) | 0 | <1s |
| Build (compile check) | `task build` | 0 | 20s |
| Test (5 host-only legs) | `task test` | 0 | 115s |
| Lint | `task lint` | 0 | 5s |
| Supplementary binary proof | `task build:release` | 0 | 12s |

`task` with no arguments listed all targets and started no build/test/lint process (confirmed by `./codegraph` still absent immediately after). `task test` ran all five host-only legs (`test:unit`, `test:golden`, `test:integration`, `test:daemon`, `test:race`) to green, including the `-race` leg. No `zig` or `mingw-w64` was ever reachable during any of these steps, and none of the cross-toolchain targets (`vet:daemon-windows`, `check:reproducibility:arm64`) were invoked — proving the host-only target set genuinely needs nothing beyond Go and a C compiler, exactly as D-10 designed.

**`task build` does not produce a binary by design** — its own `desc:` states "compile check only, no binary retained" (`go build ./...` with no `-o`, which Go discards the object for on a multi-package build). The plan's acceptance criterion literally asking for `./codegraph` to appear after `task build` does not hold for that target's actual, deliberately-chosen shape. Rather than silently reinterpret the criterion, this proof additionally ran `task build:release` (the target that does produce `./codegraph` with version-stamped `ldflags`) against the same scratch clone: confirmed `./codegraph` absent beforehand, present afterward (62,682,194 bytes, executable), and `./codegraph --version` reported `codegraph version v0.2.0-38-g3c716ff (commit 3c716ffaeb9f98c98241275c98a259dc7ce46493, built 2026-08-02T18:56:56Z)` — a real, working binary, not a no-op.

After the proof, the entire scratch clone, the isolated `PATH` bin directory, and the isolated `GOCACHE`/`GOPATH` were deleted (`rm -rf`, with `chmod -R u+w` first to clear the Go module cache's read-only file permissions). Confirmed the working tree of the actual repository was unaffected: `git status --porcelain` returned empty and `git log --oneline -2` showed only this plan's two task commits.

**ROADMAP criterion 3 ("a clean checkout can build, test, and lint via task targets alone") is satisfied, proven by this execution — not inferred from workflow text.**

## Decisions Made
- ROADMAP criterion 2 recorded as pre-satisfied on evidence rather than rewritten (D-00) — see the dedicated section above.
- `bench.yml` and `release.yml` excluded from the guard's scope entirely, matching the plan's explicit instruction — both already carry documented D-01 exceptions from earlier plans.
- Used a real YAML decoder (`yaml.Unmarshal` into typed structs) for workflow step parsing rather than extending the file's existing line-regex approach, because `run:` bodies are frequently multi-line block scalars that a line-oriented regex cannot reliably isolate per-step.
- Supplemented the `task build` no-binary reality with a `task build:release` run in the same scratch clone, rather than silently treating the "binary appears" acceptance criterion as satisfied by a target that by design produces no binary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical / criterion mismatch] `task build`'s own design (compile-check only, no binary) does not satisfy the plan's literal "`./codegraph` appears after `task build`" acceptance criterion**
- **Found during:** Task 3
- **Issue:** The plan's acceptance criteria for Task 3 state "`./codegraph` (or the build output) does not exist in the scratch tree before `task build` and does after — proving the target did work rather than no-opping." `task build`'s `desc:` (set in an earlier plan, 10-01/02) explicitly documents it as "compile check only, no binary retained" — `go build ./...` across multiple packages discards the compiled object rather than writing a binary to disk. This is a pre-existing, deliberate design decision this plan did not create and is not in scope to change (D-01: "Changing what any command does" is explicitly out of scope for this phase).
- **Fix:** Ran the proof exactly as specified against `task build` (confirming its real exit-0 success and duration) and additionally ran `task build:release` — the target that does produce `./codegraph` with real ldflags — in the same scratch clone, confirming the binary's absence-then-presence and that it executes correctly (`--version` reports the right commit). This closes the spirit of the criterion (proving the compile pipeline is not a no-op) without silently reinterpreting or skipping the literal criterion.
- **Files modified:** None — this was a verification-methodology adjustment, not a code change.
- **Verification:** Recorded in the "Clean-Checkout Proof" section above with exit codes, durations, and the binary's reported version string.
- **Committed in:** N/A (Task 3 makes no repository commit per the plan's own file list)

---

**Total deviations:** 1 auto-fixed (verification-methodology adjustment, Rule 2 classification)
**Impact on plan:** No scope creep — the underlying `task build` target's behavior is unchanged and correctly documented; only the proof methodology for one acceptance criterion was supplemented to remain honest about what that specific target does and does not produce.

## Issues Encountered
- The developer machine's normal `PATH` (`/opt/homebrew/bin`) contains `go`, `zig`, `mingw-w64`, `goreleaser`, `actionlint`, and `task` all in the same directory, so a naive `PATH` restriction to exclude the latter five would also have excluded `go` itself. Resolved by symlinking only the real `go` binary (resolved via its actual Cellar path, not the Homebrew symlink chain) into a dedicated clean-bin directory, keeping `/usr/bin`/`/bin` (which contain the system `clang` and `git`, but none of the other five tools) as the rest of `PATH`.
- The first `task test` run inside the scratch clone timed out at the tool harness's default 2-minute Bash limit (`go test -race` over three packages plus four other legs genuinely takes longer than 2 minutes from a cold build cache). Re-ran without a shortened timeout; completed in 115s with all legs green.
- Deleting the scratch clone's isolated `GOPATH`/module cache initially failed with `Permission denied` — Go marks `pkg/mod` contents read-only by design to prevent accidental modification of cached module sources. Resolved with `chmod -R u+w` before `rm -rf`.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 10 is now fully closed: `Taskfile.yml`'s complete target set is in place, `ci.yml`/`release-please.yml` are rewired through it with an enforced non-vacuous single-definition guard, `CONTRIBUTING.md` points contributors at it, and a real clean-checkout run proves the ROADMAP's build/test/lint promise.
- No blockers for subsequent milestone work. This was the final plan of Phase 10 — the phase's own SUMMARY/verification rollup can now proceed against all seven plan summaries.

## Self-Check: PASSED

- FOUND: `CONTRIBUTING.md`
- FOUND: `internal/upgrade/taskfile_shape_test.go`
- FOUND: `.planning/phases/10-local-build-contribution-and-taskfile-yml-setup/10-07-SUMMARY.md`
- FOUND commit: `5ee053f` (Task 1)
- FOUND commit: `3c716ff` (Task 2)
- FOUND commit: `83aff93` (this SUMMARY)

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-02*
