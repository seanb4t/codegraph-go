---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 06
subsystem: infra
tags: [ci, github-actions, benchmarking, perf-gate, reproducibility, namespace, taskfile]

# Dependency graph
requires:
  - phase: 10-04
    provides: "internal/bench.Metrics.Runner/.ScratchFS fields, a fresh non-stale tools/bench/baseline.json (runner: ubuntu-latest, scratch_fs: disk), and the explicit handoff that internal/bench/regression.go still needed runner (and, per BASELINE.md, scratch_fs) wired into CheckRegression"
  - phase: 10-01
    provides: "Namespace runner pattern (cache:false + nscloud-cache-action), install-task composite action, Taskfile.yml/go.tool.mod bootstrap"
provides:
  - "internal/bench.CheckRegression refuses a runner OR scratch_fs mismatch as a category error, same treatment and same ordering as the pre-existing GOOS/GOARCH refusal, firing before any numeric tolerance check; an empty value on either side is refused too (never recorded, never a wildcard)"
  - "task bench:regression — D-14's legible local wrapper around the perf gate, explaining a platform/runner/scratch_fs category-error refusal in prose without remapping or swallowing the underlying exit status; no -rebless flag reachable (D-13)"
  - "task check:reproducibility / task check:reproducibility:arm64 — the two double-build hash-diff legs ported character-for-character from ci.yml, runnable on a contributor checkout"
  - "ci.yml's perf-regression job now invokes task bench:regression (stays on ubuntu-latest — see Deviations) with CODEGRAPH_BENCH_RUNNER=ubuntu-latest so its own measurement satisfies the new runner check"
  - "ci.yml's reproducibility job moved to namespace-profile-linux-amd64-4x8, both double-build steps now invoke the new task targets, blocking/non-blocking split preserved"
affects: []

actuals:
  tokens: 7842
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Frame-descriptor category-error refusal: a new measurement-environment field (Runner, ScratchFS) gets the identical treatment as the pre-existing GOOS/GOARCH guard — equality check before any numeric arithmetic, empty-vs-non-empty asymmetry refused, both-empty allowed — rather than inventing new comparison semantics per field"
    - "go-task's CLI always reports process exit 201 for ANY failing task regardless of the wrapped command's real exit code (verified via an isolated `exit 5` test task) — a target's own internal exit-status handling is proven correct by testing the script logic directly (echoed prose, captured $STATUS), not by comparing task's own outer process exit code against the bare command's"
    - "Splitting a multi-target Taskfile/workflow edit into per-task commits after the fact: reconstruct each intermediate state from git show HEAD:<file> + the new blocks by line range, verify with a diff against the final state, commit, then restore — used here to keep Task 2 (bench:regression) and Task 3 (check:reproducibility) as separate atomic commits despite having authored both files' changes together"

key-files:
  created: []
  modified:
    - internal/bench/regression.go
    - internal/bench/regression_test.go
    - tools/bench/BASELINE.md
    - Taskfile.yml
    - .github/workflows/ci.yml

key-decisions:
  - "Task 1 wires BOTH runner and scratch_fs into CheckRegression, not runner alone. PLAN.md's Task 1 text and BASELINE.md's own handoff note only asked for runner; the orchestrator's changed_premise instruction explicitly extended scope to scratch_fs too (10-04 had already flagged scratch_fs as the identical failure class and left it as an open handoff to whichever plan next touched this file — this plan closes both in one pass rather than leaving scratch_fs for a third plan)."
  - "ci.yml's perf-regression job stays on ubuntu-latest — PLAN.md's premise (move it to Namespace now that a Namespace baseline is committed) is false. 10-04's own five-round investigation found ubuntu-latest+disk the most stable configuration measured (0.35% session disagreement, 28.6x headroom) against every Namespace alternative tested, and the committed baseline.json is ubuntu-latest+disk. Moving the gate would recreate the exact runner-vs-baseline frame mismatch that produced this repo's retracted fictitious 10.6% regression. DefaultThroughputTolerance and baseline.json are both untouched."
  - "ci.yml's reproducibility job DID move to Namespace (namespace-profile-linux-amd64-4x8) — this part of D-06/the original plan was never superseded. Unlike the perf gate, this job self-compares two builds within one job run; it carries no baseline-frame comparison risk, so 10-04's findings don't apply to it."
  - "CODEGRAPH_BENCH_RUNNER=ubuntu-latest added at perf-regression job level. Without it the job's own measurement would carry an empty Runner field, which the new category-error check refuses just as hard as a genuine mismatch (empty is never a wildcard) — this would have broken the job's own required check on the very next PR had it been omitted."
  - "check:reproducibility's script is ported byte-for-byte from ci.yml including its GNU `date -u -d` usage — verbatim per D-15-style byte-fidelity, not restructured for macOS/BSD portability. Documented explicitly in the target's own desc: a Linux host (or GNU coreutils on PATH) is required locally."

patterns-established:
  - "A Metrics-comparison guard's empty-side handling is proven with FOUR cases per field (mismatch, match, baseline-empty, current-empty) plus a fifth (both-empty passes) — established by the pre-existing GOOS/GOARCH tests, extended identically for Runner and ScratchFS rather than inventing a lighter-weight test shape for the newer fields."

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "internal/bench.CheckRegression refuses a runner mismatch (non-empty vs non-empty, differing) as a category error naming both values, before any numeric comparison — proven in a Go unit table AND against the real committed tools/bench/baseline.json inside a linux/amd64 Docker container (CODEGRAPH_BENCH_RUNNER=deliberately-wrong-runner -> exit 1, error names both 'ubuntu-latest' and 'deliberately-wrong-runner'; the correct label then proceeds past the runner check to the numeric verdict)"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression (cases: runner mismatch..., runner mismatch refused before numeric comparison...)"
        status: pass
      - kind: other
        ref: "docker run --platform linux/amd64 golang:1.26 go run ./tools/bench/runner -mode regression -baseline tools/bench/baseline.json against the real committed baseline; CODEGRAPH_BENCH_RUNNER=deliberately-wrong-runner exits 1 naming both runner values, CODEGRAPH_BENCH_RUNNER=ubuntu-latest exits 1 for an unrelated throughput reason (proceeded past the category-error check)"
        status: pass
    human_judgment: false
  - id: D2
    description: "An empty runner OR scratch_fs value on either side is refused, never treated as a wildcard match against a recorded value; both sides empty still passes (unattributed callers, pre-attribution baselines unaffected)"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression (cases: empty baseline runner..., non-empty baseline runner against empty current runner..., unattributed runner on both sides passes, and the four scratch_fs equivalents)"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/bench.CheckRegression refuses a scratch_fs mismatch with the identical category-error treatment as runner/GOOS/GOARCH, before any numeric comparison — proven in the same unit table plus a real Docker run (-scratch-fs tmpfs against the disk-recorded baseline exits 1, names both scratch_fs values)"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "internal/bench/regression_test.go#TestCheckRegression (scratch_fs mismatch..., matching scratch_fs..., empty baseline/current scratch_fs..., unattributed scratch_fs on both sides passes)"
        status: pass
      - kind: other
        ref: "docker run --platform linux/amd64 golang:1.26 go run ./tools/bench/runner -mode regression -scratch-fs tmpfs against the real committed baseline (scratch_fs: disk) — exit 1, names both scratch_fs values"
        status: pass
    human_judgment: false
  - id: D4
    description: "task bench:regression explains a platform/runner/scratch_fs category-error refusal in prose and still exits non-zero on a developer machine, never remapping or swallowing the underlying exit status; no -rebless flag reachable from any task target (D-13)"
    requirement: DEV-01
    verification:
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task bench:regression on this darwin/arm64 dev machine: prints the platform-mismatch error plus the added prose explanation, go-task's own CLI reports exit 201 (its fixed generic-failure code for ANY failing task, verified via an isolated exit-5 test task); the bare `go run ./tools/bench/runner -mode regression ...` invocation for the identical input exits 1 (source: tools/bench/runner/main.go's single os.Exit(1) error path) — the target's own internal $STATUS (echoed correctly, never overwritten before `exit \"$STATUS\"`) matches the bare command's real exit code; go-task's outer 201 is a pre-existing constant across every target in this Taskfile, not something this target introduces or masks. rg -n -- '-rebless' Taskfile.yml shows only prose mentions, no reachable flag."
        status: pass
    human_judgment: false
  - id: D5
    description: "ci.yml's reproducibility job runs on Namespace and invokes the new task targets for both double-build legs; the byte-for-byte comparison logic (build flags, hash comparison, blocking-vs-reported-only disposition) is unchanged from the pre-move script"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml (clean); internal/upgrade TestRequiredCheckNamesPreserved (job name unchanged)"
        status: pass
      - kind: other
        ref: "task check:reproducibility / task check:reproducibility:arm64 run for real inside a linux/amd64 Docker container with Go 1.26.5 + zig 0.15.1 (matching ci.yml's pin): both GREEN on the real repo (double-build hashes matched). RED/GREEN mutation proof on each: injected a build-time-varying ldflag into the second build only, confirmed via git diff the mutation landed, re-ran — linux/amd64 leg: exit 201 (task's generic-failure code), named the two differing hashes; arm64 leg: emitted ::warning:: naming the two differing hashes and exited 0 (non-blocking split survived the move). Reverted both, re-ran: GREEN. zig-absent precondition: with zig removed from the container's PATH (its initial state), check:reproducibility:arm64 exited non-zero (201) with the install instruction, captured directly from the command."
        status: pass
    human_judgment: false
  - id: D6
    description: "task check:reproducibility is runnable locally, deriving the same determinism inputs (SOURCE_DATE_EPOCH, COMMIT) ci.yml's Compute determinism inputs step computes, when they are not already set in the environment"
    requirement: DEV-01
    verification:
      - kind: other
        ref: "check:reproducibility's SOURCE_DATE_EPOCH/COMMIT defaulting (git log -1 --format=%ct / git rev-parse HEAD) matches ci.yml's own step verbatim; exercised for real inside the Docker verification above with no env vars pre-set"
        status: pass
    human_judgment: false
  - id: D7
    description: "A real pushed CI run shows both rewired jobs (perf-regression on ubuntu-latest, reproducibility on Namespace) green with the new step names, and a deliberately-mismatched CODEGRAPH_BENCH_RUNNER / injected non-determinism fails the respective job in real CI before being reverted"
    verification: []
    human_judgment: true
    rationale: "This executor ran on a feature branch with no open PR; ci.yml triggers on pull_request/push, so this IS verifiable in principle (unlike release.yml/release-please.yml, which are tag/main-push-only) — but a full push + observe-green + push-mutation + observe-red + revert-and-observe-green round trip for two separate jobs was judged out of scope for a single execute-plan session, consistent with 10-01/10-03/10-05's identical documented gap for their own Namespace-migrated or newly-task-invoked jobs. Locally-runnable substitutes were used instead (see D1/D3/D5): real linux/amd64 Docker double-builds and regression-gate runs against the actual committed baseline.json and actionlint, all producing the exact script bodies ci.yml now invokes via `task <target>`. A maintainer should confirm on the next push."

duration: ~24min (task commit span; local + Docker-based verification work extended wall-clock well beyond this)
completed: 2026-08-02
status: complete
---

# Phase 10 Plan 06: Runner/ScratchFS Category-Error Gate + Reproducibility Task Targets Summary

**`internal/bench.CheckRegression` now refuses a runner OR scratch_fs mismatch the same way it already refused GOOS/GOARCH — closing the exact blind spot that produced this repo's retracted fictitious 10.6% "regression" — while the perf gate itself stays on `ubuntu-latest` (10-04 disproved the Namespace-migration premise this plan was originally written against) and the reproducibility job's two double-builds move to Namespace behind new, locally-runnable `task check:reproducibility(:arm64)` targets.**

## Performance

- **Duration:** ~24 min (span between the first and last task commit; substantial additional wall-clock went into Docker-based real double-build and regression-gate verification not reflected in the commit-to-commit span)
- **Started:** 2026-08-02T14:12:27-04:00
- **Completed:** 2026-08-02T14:35:52-04:00
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- `internal/bench.CheckRegression` refuses a runner mismatch AND a scratch_fs mismatch as category errors — same shape, same ordering (before any numeric tolerance check), same empty-means-never-recorded semantics as the pre-existing GOOS/GOARCH guard. Demonstrated in a 12-case unit table extension AND against the real committed `tools/bench/baseline.json` inside a linux/amd64 Docker container (this dev machine is darwin/arm64, so the platform guard would otherwise mask both new checks).
- Added `task bench:regression` (D-14): wraps the perf gate's exact CI invocation, explains a category-error refusal in prose, never remaps or swallows the real exit status, and exposes no `-rebless` flag (D-13).
- Added `task check:reproducibility` / `task check:reproducibility:arm64`, porting `ci.yml`'s two double-build scripts character-for-character (D-15-style byte fidelity — no restructuring, no shared helper factored across the two targets). Both proven real inside a linux/amd64 Docker container with Go 1.26.5 and zig 0.15.1: GREEN on the actual repo, RED-then-GREEN under an injected non-determinism mutation, and the arm64 leg's `::warning::`/exit-0 non-blocking disposition confirmed to survive the port under the identical mutation that failed the blocking leg.
- Rewired `ci.yml`'s `perf-regression` job to call `task bench:regression`, keeping `runs-on: ubuntu-latest` unchanged (the plan's Namespace-migration premise for this specific job is false — see Deviations) and adding `CODEGRAPH_BENCH_RUNNER: ubuntu-latest` so the job's own measurement satisfies the new runner check against the committed baseline.
- Rewired `ci.yml`'s `reproducibility` job onto `namespace-profile-linux-amd64-4x8` with the standard `Cache Go modules and build` + `Install Task` steps, both double-build steps now one-line `task <target>` calls, blocking/non-blocking split and all step names/comments preserved.
- Updated `tools/bench/BASELINE.md`'s two "does NOT yet participate" paragraphs to record that both `runner` and `scratch_fs` now gate `CheckRegression`.

## Task Commits

Each task was committed atomically (Task 1 as TDD RED→GREEN→docs, Tasks 2/3 as single `feat` commits each, reconstructed from the joint edit into two atomic commits — see Deviations for the reconstruction method):

1. **Task 1: Refuse a cross-runner (and cross-scratch_fs) regression comparison (TDD)** — RED `c466801`, GREEN `ad56f4b`, docs `360bed5`
2. **Task 2: bench:regression target; perf gate wired through task (stays on ubuntu-latest)** — `d3aad20` (feat)
3. **Task 3: check:reproducibility(:arm64) targets; reproducibility job moved to Namespace** — `112810c` (feat)

_TDD Task 1 shows RED (12 new failing cases, confirmed via `go test -v` before any implementation change) then GREEN (implementation) then a docs commit updating BASELINE.md's handoff note, per this project's established TDD discipline._

## Files Created/Modified

- `internal/bench/regression.go` - Runner/ScratchFS category-error checks added after the GOOS/GOARCH guard, before any numeric comparison; `runnerString`/`scratchFSString` render-helpers mirroring `platformString`
- `internal/bench/regression_test.go` - 12 new `TestCheckRegression` cases covering runner and scratch_fs mismatch/match/empty-asymmetry/both-empty/fires-before-numeric
- `tools/bench/BASELINE.md` - the two "does NOT yet participate" paragraphs rewritten to state both fields now gate the comparison
- `Taskfile.yml` - `bench:regression`, `check:reproducibility`, `check:reproducibility:arm64` targets added
- `.github/workflows/ci.yml` - `perf-regression` job: `Install Task` step added, `CODEGRAPH_BENCH_RUNNER: ubuntu-latest` job env added, run body -> `task bench:regression` (runs-on unchanged). `reproducibility` job: `runs-on` -> `namespace-profile-linux-amd64-4x8`, `cache: false` + `Cache Go modules and build` + `Install Task` steps added, both double-build run bodies -> `task check:reproducibility` / `task check:reproducibility:arm64` (env:/continue-on-error:/step names unchanged)

## Decisions Made

See `key-decisions` in frontmatter for the load-bearing summary. In full:

1. Task 1 wires both `runner` and `scratch_fs` in one pass, not `runner` alone as PLAN.md's Task 1 text literally specified — the orchestrator's `<changed_premise>` explicitly extended the scope, and `BASELINE.md` had already flagged `scratch_fs` as the identical failure class with an open handoff to "whichever plan next touches `CheckRegression`" (this plan).
2. `perf-regression` stays on `ubuntu-latest` — the plan's central premise (a Namespace baseline would be committed, so the gate should follow) was disproved by 10-04's own five-round, four-configuration investigation. Moving the gate now would recompare Namespace throughput against the ubuntu-latest-recorded committed baseline, i.e. deliberately reproduce the exact failure class this whole gate exists to prevent.
3. `reproducibility` DID move to Namespace — this half of D-06 was never in question; the changed premise is scoped to the perf gate specifically (a self-comparing double-build carries no baseline-frame risk).
4. `CODEGRAPH_BENCH_RUNNER=ubuntu-latest` added at the perf-regression job level — without it, the job's own next run would carry an empty `Runner` and get refused by the very check this plan just added, against the committed baseline's non-empty `"ubuntu-latest"`. This is a Rule 2/3 addition: omitting it would have broken a required status check on the very next PR.
5. `check:reproducibility`'s script keeps `date -u -d` (GNU-only) verbatim from `ci.yml`, not the `TZ=UTC git log ... --date=format-local` idiom `build:release` already uses for cross-platform compatibility — D-15's byte-fidelity requirement (character-for-character, no restructuring) takes precedence here, and the target's own `desc:` documents the Linux/GNU-coreutils requirement explicitly rather than leaving it a silent surprise.
6. Split the joint Taskfile.yml/ci.yml edit into two atomic commits (Task 2, Task 3) by reconstructing each intermediate state via `git show HEAD:<file>` plus the relevant new block(s), verified against the final state with `diff` before each commit — preserves the plan's task-level commit granularity despite authoring both files' full set of changes together for efficiency.

## Deviations from Plan

### Auto-fixed / orchestrator-directed issues

**1. [Rule 4-equivalent, orchestrator-directed — not autonomous] Task 2's Namespace migration for `perf-regression` dropped; the job stays on `ubuntu-latest`**

- **Found during:** Before Task 2 began — flagged explicitly in the spawning prompt's `<changed_premise>`, itself sourced from 10-04-SUMMARY.md's own "Next Phase Readiness" blocker note and `tools/bench/BASELINE.md`'s full investigation record.
- **Issue:** PLAN.md's objective, Task 2's action text, and its `must_haves.truths` all assumed 10-04 would commit a Namespace-recorded baseline and this plan would move `ci.yml`'s perf gate to follow it. 10-04's actual five-round investigation found `ubuntu-latest`+disk the most stable configuration measured (0.35% session disagreement, 28.6x headroom) against every Namespace alternative (Namespace+disk 4.36%/2.30x, ubuntu+tmpfs 5.75%/1.74x, Namespace+tmpfs 12.46%/0.80x — the worst), and committed the baseline on `ubuntu-latest`+disk instead.
- **Fix:** Dropped the `runs-on` migration and the "replace ubuntu-latest with the Namespace profile" comment edit from Task 2's action text (nothing needed replacing — the comment's runner-class assertion was already correct). Kept everything else Task 2 asked for: the `bench:regression` target, wiring `ci.yml`'s run body through it, and a job-level `CODEGRAPH_BENCH_RUNNER` env var (value changed from a Namespace profile label to `ubuntu-latest`, matching the actual committed baseline).
- **Files modified:** `.github/workflows/ci.yml` (commit `d3aad20`)
- **Verification:** `GOWORK=off go tool -modfile=go.tool-lint.mod actionlint .github/workflows/*.yml` clean; `internal/upgrade` shape tests (job name unchanged) pass; real Docker-run regression gate against the committed baseline with `CODEGRAPH_BENCH_RUNNER=ubuntu-latest` proceeds past the new runner check to its numeric verdict.
- **Committed in:** `d3aad20`

**2. [Rule 2 - Missing critical functionality] `must_haves.truths` #4 in PLAN.md's frontmatter is only half-satisfiable as written; delivered the still-valid half plus its supersession**

- **Found during:** Reading `<changed_premise>` before Task 1.
- **Issue:** The frontmatter truth "`ci.yml`'s perf gate and reproducibility jobs run on Namespace and invoke task targets" bundles two independent claims. The perf-gate half is now false (see deviation 1); the reproducibility-job half was never in question and is delivered in full (Task 3, unchanged from PLAN.md).
- **Fix:** Delivered the reproducibility half exactly as planned; recorded the perf-gate half's supersession here and in `coverage:` D5/D7 rather than silently declaring the truth satisfied or silently dropping it.
- **Files modified:** none beyond what Tasks 1-3 already touch; this is a documentation/tracking deviation.
- **Committed in:** captured in this SUMMARY, not a separate code commit.

---

**Total deviations:** 2 (1 orchestrator-directed architectural supersession, carried through Task 2's actual implementation; 1 documentation reconciliation of a frontmatter truth that bundled a since-disproved premise with a still-valid one).
**Impact on plan:** Every still-valid piece of PLAN.md's work landed in full (the runner/scratch_fs category-error wiring, `bench:regression`, both `check:reproducibility` targets, the reproducibility job's Namespace move). Only the perf-gate runner migration — the part 10-04's own investigation disproved — was dropped, per explicit instruction from the spawning context rather than autonomous judgment. No scope creep: `scratch_fs` wiring was likewise an explicit instruction, not an autonomous addition, and was already flagged as this plan's responsibility by 10-04's own handoff note.

## Issues Encountered

- **This dev machine is darwin/arm64; `tools/bench/baseline.json`'s committed frame is linux/amd64/ubuntu-latest/disk.** Every real (non-Docker) local run of `tools/bench/runner -mode regression` or `task bench:regression` on this machine hits the pre-existing GOOS/GOARCH platform guard first, which would have masked the new runner/scratch_fs checks entirely if relied on alone. Resolved by running the actual `go run ./tools/bench/runner` / `task check:reproducibility(:arm64)` commands for real inside a `--platform linux/amd64` Docker container (`golang:1.26`, matching `go.mod`'s pinned `go 1.26.5`, plus `gcc` for CGo and `zig 0.15.1` matching `ci.yml`'s pin) — proving the refusals and the double-builds against the actual runtime environment class the baseline and `ci.yml` both use, not merely in a Go unit table.
- **`go-task`'s CLI always reports process exit 201 for any failing task**, regardless of the wrapped command's own real exit code (confirmed via an isolated `exit 5` test-task: `task` itself exits 201, not 5). This is a pre-existing, uniform go-task behavior affecting every target in this Taskfile already, not something introduced by `bench:regression` or `check:reproducibility`. Task 2's acceptance criterion ("assert the exit status is the same non-zero value the bare `go run` invocation produces") is satisfied at the level go-task's own CLI wrapping allows: the target's internal `$STATUS` variable is proven to carry the real underlying exit code (1, matching the bare invocation, confirmed by inspecting `tools/bench/runner/main.go`'s single `os.Exit(1)` error path and the target's echoed prose only firing on that non-zero branch) and is passed unmodified to the shell's `exit` builtin — nothing in this target's own script remaps or swallows it. Documented here rather than silently treated as satisfied by a numeric coincidence that go-task's design does not actually provide for any target.
- **`ziglang.org`'s download URL naming changed between the older `zig-linux-<arch>-<ver>.tar.xz` pattern and the current `zig-<arch>-linux-<ver>.tar.xz`** — encountered while installing zig 0.15.1 inside the Docker verification container. Resolved by querying `https://ziglang.org/download/index.json` for the exact tarball URL rather than guessing the pattern. Not a repo issue — `ci.yml` uses the `mlugg/setup-zig` action, which handles this internally.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `internal/bench/regression.go` now checks GOOS/GOARCH, runner, AND scratch_fs — open issue #16 (`Metrics.Repo`/corpus identity never compared) remains the one documented gap in `CheckRegression`'s coverage, unchanged by this plan (flagged, not addressed — out of this plan's scope per its own `<flagged_assumptions>`).
- **Watch the next push/PR** for both rewired `ci.yml` jobs actually scheduling and reporting green: `perf-regression` on `ubuntu-latest` (should behave identically to before this plan, now invoking `task bench:regression`) and `reproducibility` on `namespace-profile-linux-amd64-4x8` (a genuinely new runner class for this job — confirm it schedules rather than queuing indefinitely, matching the same watch-item pattern 10-01/10-03 already left open for their own Namespace migrations).
- This plan's coverage item D7 (real pushed-CI RED/GREEN proof for both jobs) is `human_judgment: true` and unverified from this feature branch — see its `rationale` for what was verified as a substitute (real linux/amd64 Docker double-builds and regression-gate runs against the actual committed `baseline.json`, `actionlint` clean).
- No files under `tools/bench/baseline.json` were touched — `git diff` confirms it, and this was a hard constraint (D-09, "do not re-rebless, do not touch baseline.json").

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 5 files modified this plan verified present on disk (`internal/bench/regression.go`, `internal/bench/regression_test.go`, `tools/bench/BASELINE.md`, `Taskfile.yml`, `.github/workflows/ci.yml`). All 5 commit hashes listed in Task Commits verified present in `git log --oneline --all` (`c466801`, `ad56f4b`, `360bed5`, `d3aad20`, `112810c`). Frontmatter YAML validated parseable (Ruby's `YAML.safe_load`, 17 keys, 7 `coverage` entries).
