---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 04
subsystem: infra
tags: [ci, github-actions, benchmarking, perf-gate, namespace, taskfile]

# Dependency graph
requires:
  - phase: 10-01
    provides: "Namespace runner pattern (cache:false + nscloud-cache-action), install-task composite action, .github/actionlint.yaml Namespace label allow-list"
provides:
  - "internal/bench.Metrics.Runner and .ScratchFS fields — the CI runner identity and scratch-filesystem class a Metrics was measured on, alongside goos/goarch"
  - "tools/bench/runner -runner and -scratch-fs (auto/tmpfs/disk, default disk) flags, plus resolveScratchDirForClass/resolveRegressionScratchDir as reusable, tested primitives for future runner/storage investigations"
  - "Taskfile.yml diag:cpu and diag:storage-fit — permanent, non-gating CPU/memory/storage diagnostic targets"
  - "A fresh, non-stale tools/bench/baseline.json (ubuntu-latest, disk, 2026-08-02) and a fully documented investigation in tools/bench/BASELINE.md"
affects: [10-06]

actuals:
  tokens: 38577
  tasks: 3
  commits: 17

tech-stack:
  added: []
  patterns:
    - "Runner/storage frame descriptors recorded in Metrics but deliberately NOT wired into CheckRegression's comparison in the same plan that adds them (explicit handoff to a later plan, avoiding a half-tested gate change)"
    - "Read-only 'measure and report to a throwaway /tmp path' pattern for same-day controls, using the sanctioned -rebless-to-scratch-path idiom already established for local investigation, never committing to the real baseline"
    - "Class-pin flags (-scratch-fs auto/tmpfs/disk) as first-class, permanently-kept capabilities rather than throwaway debugging code, specifically so a future same-day control is cheap"

key-files:
  created:
    - tools/bench/cpudiag/main.go
  modified:
    - internal/bench/metrics.go
    - tools/bench/runner/main.go
    - tools/bench/runner/main_test.go
    - .github/workflows/bench.yml
    - Taskfile.yml
    - tools/bench/baseline.json
    - tools/bench/BASELINE.md

key-decisions:
  - "Namespace migration for the perf-regression gate is NOT adopted. A same-day, same-methodology control across all four (runner x scratch_fs) combinations found ubuntu-latest+disk measurably the most stable (0.35% session disagreement, 28.6x headroom vs 10% tolerance), beating Namespace+disk (4.36%/2.30x), ubuntu+tmpfs (5.75%/1.74x), and Namespace+tmpfs (12.46%/0.80x, worst overall). bench.yml's rebless job returned to ubuntu-latest; headtohead (non-gating) deliberately stays on Namespace."
  - "DefaultThroughputTolerance is UNCHANGED. At 28.6x headroom the existing 10% budget needs no widening; the adopt-and-widen path explored mid-investigation was abandoned entirely once the control ruled it out."
  - "-scratch-fs defaults to disk, not auto/tmpfs. A tmpfs experiment (motivated by a real I/O-bound-workload hypothesis) was implemented, measured, and refuted — tmpfs made session-to-session variance WORSE on both runner classes tested, not better. The tmpfs code path (-scratch-fs tmpfs/auto) is kept, not reverted, as a first-class capability for the deferred Namespace cache-volume follow-up."
  - "The baseline this plan replaced (files_per_sec: 11279.59, committed only two days before this plan started) was itself stale by +51.5% against a same-runner-class, same-scratch-fs, same-day remeasurement. The cause is explicitly NOT established (could be a genuine code speedup or a GitHub-hosted fleet hardware change) and is documented as such, not asserted."
  - "Runner and ScratchFS frame descriptors are recorded in Metrics/baseline.json but deliberately NOT wired into internal/bench.CheckRegression's comparison logic in this plan — internal/bench/regression.go has zero diff across all 17 commits. Explicit handoff to plan 10-06, which already owns that file."

patterns-established:
  - "Any new benchmark-measurement dimension (runner class, storage filesystem) gets a Metrics field, is recorded but not yet gated, and ships with an explicit handoff note for whichever plan next touches CheckRegression — never half-wire a gate change."
  - "Before concluding a same-day comparison contradicts a historical reference number, re-measure the historical side under identical methodology first (the 'missing control' this investigation caught itself making, after already having documented the exact same failure mode in tools/bench/BASELINE.md from a prior incident)."

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "internal/bench.Metrics gains Runner and ScratchFS fields (JSON keys runner, scratch_fs), populated end-to-end through tools/bench/runner in both headtohead and regression modes, round-tripping correctly against legacy baseline.json files with no runner/scratch_fs key"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "tools/bench/runner/main_test.go#TestParseFlags_RunnerFromEnv,TestMetricsRunner_MarshalsToRunnerKey,TestReadBaseline_LegacyFileWithoutRunnerKeyYieldsEmptyString,TestMetricsScratchFS_MarshalsToScratchFSKey,TestReadBaseline_LegacyFileWithoutScratchFSKeyYieldsEmptyString"
        status: pass
      - kind: other
        ref: "committed tools/bench/baseline.json carries non-empty runner:\"ubuntu-latest\" and scratch_fs:\"disk\", confirmed via downloaded rebless artifact diff"
        status: pass
    human_judgment: false
  - id: D2
    description: "Mixed-runner and mixed-scratch-fs trial aggregation is refused as a category error rather than silently resolved, mirroring CheckRegression's existing GOOS/GOARCH refusal"
    requirement: DEV-01
    verification:
      - kind: unit
        ref: "tools/bench/runner/main_test.go#TestMedianMetrics_RejectsMixedRunner"
        status: pass
    human_judgment: false
  - id: D3
    description: "No task target exposes the bench runner's -rebless flag (D-13) — it stays reachable only through bench.yml's manually-dispatched, no-write-token rebless job, across every commit in this plan"
    requirement: DEV-01
    verification:
      - kind: other
        ref: "rg -n rebless Taskfile.yml (empty match) re-verified after every commit touching Taskfile.yml or bench.yml in this plan"
        status: pass
    human_judgment: false
  - id: D4
    description: "The Namespace-vs-ubuntu-latest runner-class question for the perf-regression gate is resolved with real, same-day, same-methodology measurements across four configurations, not assumption — including a CPU-oversubscription hypothesis (refuted), an I/O-bound-workload/tmpfs hypothesis (implemented and refuted), and a missing-control check before drawing a final conclusion"
    requirement: DEV-01
    verification: []
    human_judgment: true
    rationale: "This is an empirical investigation into infrastructure behavior (runner placement variance, storage subsystem performance) rather than a testable code property — its correctness was established through five rounds of maintainer-reviewed checkpoints reading real CI measurement data, not through a unit test. The full evidence trail is in tools/bench/BASELINE.md's '10-04-PLAN' section for a human to re-verify the reasoning."
  - id: D5
    description: "A ~51.5% staleness in the immediately-prior committed baseline is discovered, its consequence for gate detection power is quantified (a 37.8% regression would have passed green), and its cause is explicitly recorded as unmeasured rather than asserted"
    requirement: DEV-01
    verification: []
    human_judgment: true
    rationale: "Whether the documented staleness finding is communicated clearly enough, and whether 'cause not established' is the right epistemic stance to leave for future readers, is an editorial judgment call best made by a human reading tools/bench/BASELINE.md directly."

duration: ~16h (wall-clock across 5 maintainer checkpoints; CI-dispatch-wait-dominated — cumulative benchmark-dispatch wait time across 11 workflow runs was several hours by itself)
completed: 2026-08-02
status: complete
---

# Phase 10 Plan 04: Runner Identity, Storage-Frame Investigation & Baseline Staleness Fix Summary

**Investigated a real 4.36% Namespace throughput-variance regression through five maintainer-reviewed checkpoints — refuted CPU oversubscription, implemented and then refuted a tmpfs fix, caught a missing-control gap in its own reasoning, and landed on keeping the perf gate on ubuntu-latest while discovering and fixing a separate 51.5% baseline-staleness bug along the way.**

## Performance

- **Duration:** ~16h wall-clock (5 maintainer checkpoints across the investigation; the code changes themselves are small — see commit list below — most elapsed time was CI dispatch/measurement wait across 11 `bench.yml` workflow runs, several of them 14-trial ~19-minute reblesses)
- **Started:** 2026-08-01T21:56:18-04:00
- **Completed:** 2026-08-02T13:55:46-04:00
- **Tasks:** 3 (plan tasks) + 1 blocking checkpoint that resolved through 5 maintainer-directed investigation rounds
- **Files modified:** 7 modified, 1 created (net; one workflow file was created and later removed within the investigation — see Deviations)

## Accomplishments

- Added `internal/bench.Metrics.Runner` and `.ScratchFS` — CI runner identity and scratch-filesystem-class frame descriptors, closing two structural blind spots in the perf-regression gate's platform guard (a runner-class change and a storage-frame change can both hold `GOOS`/`GOARCH` constant while changing the actual measurement environment).
- Investigated a real, reproducible 4.36% session-to-session throughput variance on `namespace-profile-linux-amd64-4x8` through a disciplined refute-before-concluding process: a CPU-oversubscription hypothesis (measured `runtime.NumCPU()`, `nproc`, cgroup quota — all agreed, refuted), an I/O-bound-workload/overlayfs hypothesis (real CPU/storage A/B, `dd` probes, a `du -sh` sizing measurement, a tmpfs implementation — measured worse on BOTH runner classes, refuted), and a same-day disk-scratch control that caught the investigation's own missing-control gap before it could repeat this repo's original fictitious-10.6%-regression mistake in a new location.
- **Resolution: stayed on `ubuntu-latest` for the perf-regression gate.** Every alternative configuration measured (Namespace+disk, ubuntu+tmpfs, Namespace+tmpfs) was worse than the incumbent (0.35% disagreement / 28.6x headroom). `DefaultThroughputTolerance` is unchanged — no widening was needed once the actual cause of the observed variance turned out not to be fixable by a runner or storage-frame change.
- Discovered, along the way, that the baseline this plan replaced — committed only two days before this plan started — was itself **51.5% stale**, meaning the gate had quietly lost most of its detection power (a 37.8% real regression would have passed green). Reblessed a fresh baseline on `ubuntu-latest`+disk and documented the staleness finding prominently, explicitly declining to assert an unmeasured cause (genuine speedup vs. GitHub fleet hardware change).
- Added `-scratch-fs` (`auto`/`tmpfs`/`disk`, default `disk`) as a permanent, first-class capability — not debugging scaffolding — specifically so a future same-day control (e.g. testing Namespace cache volumes, the deferred follow-up) is cheap to run without reverting anything.
- Added `Taskfile.yml diag:cpu` and `diag:storage-fit` as permanent, non-gating diagnostic targets, plus `tools/bench/cpudiag` — kept for future infrastructure investigations, not removed after use.

## Task Commits

Each task was committed atomically; this plan's blocking checkpoint (originally a single `checkpoint:decision`) resolved through an extended, maintainer-directed investigation spanning many additional commits beyond the plan's original 3-task shape:

1. **Task 1: Record the runner identity alongside goos/goarch (TDD)** — RED `d44987d`, GREEN `acd14bf`
2. **Task 2: Move bench.yml to Namespace and record a candidate baseline** — `7111b8a` (later partially reverted — see Deviations)
3. **Checkpoint investigation (5 rounds, maintainer-directed):**
   - CPU-topology diagnostic (Rule 2 deviation) — `600c3e9`, `eff4bde` (dispatch-mechanics fix)
   - A/B CPU/memory/storage diagnostic across both runner classes — `0279367`
   - tmpfs sizing diagnostic — `70f2acc`
   - Regression scratch-dir filesystem class (TDD) — RED `01319e0`, GREEN `47eff33`
   - Post-tmpfs-fix re-measurement jobs — `18debd2`
   - `-scratch-fs` class-pin capability (TDD) — RED `b677f78`, GREEN `d680f71`
   - Disk-scratch control re-measurement job — `4c3e21e`
   - `-scratch-fs` default flip to disk (TDD) — RED `ddbaf3e`, GREEN `3b69ec0`
   - Return `rebless` job to `ubuntu-latest`, keep `headtohead` on Namespace — `72d404a`
4. **Task 3: Commit the reviewed baseline with its provenance** — `335a88f` (adopts the `ubuntu-latest`+disk baseline byte-for-byte, documents the full investigation in `BASELINE.md`)

_TDD tasks show RED (failing test, confirmed via compile error) then GREEN (implementation) commits, per this plan's own established discipline from Task 1 onward._

## Files Created/Modified

- `internal/bench/metrics.go` - `Metrics.Runner` and `.ScratchFS` fields (JSON `runner`, `scratch_fs`)
- `tools/bench/runner/main.go` - `-runner`, `-scratch-fs` flags; `resolveRegressionScratchDir`, `resolveScratchDirForClass`; mixed-runner aggregation refusal; `medianMetrics` returns `(Metrics, error)`
- `tools/bench/runner/main_test.go` - full test coverage for all of the above (RED→GREEN pairs throughout)
- `tools/bench/cpudiag/main.go` - new: minimal diagnostic printing `runtime.NumCPU()`/`GOMAXPROCS(0)`
- `.github/workflows/bench.yml` - `rebless` back on `ubuntu-latest`; `headtohead` stays on Namespace; four new non-gating diagnostic/comparison jobs (`cpu-diag-github`, `cpu-diag-namespace`, `scratch-fs-compare-github`, `scratch-fs-compare-namespace`, `disk-control-github`)
- `Taskfile.yml` - `diag:cpu`, `diag:storage-fit` targets (Linux-only preconditions, fail loud on other hosts)
- `tools/bench/baseline.json` - reblessed, `ubuntu-latest`+disk, `runner`/`scratch_fs` populated, byte-identical to the downloaded CI artifact (SHA-256 verified)
- `tools/bench/BASELINE.md` - full investigation record (4 rounds + staleness finding + deferred cache-volume follow-up), ~300 lines added

## Decisions Made

See `key-decisions` in frontmatter for the load-bearing summary. In full, chronologically:

1. Runner and ScratchFS are recorded but not wired into `CheckRegression` — explicit handoff to plan 10-06, `internal/bench/regression.go` has zero diff across all 17 commits (verified via `git diff` after every commit).
2. CPU-oversubscription hypothesis refuted by direct measurement (not assumed) before proceeding to any code change.
3. The tmpfs fix was implemented as a real, tested code change (not a quick hack) specifically because refuting a half-implemented hypothesis is weaker evidence than refuting a fully-implemented one.
4. Before accepting "tmpfs helps Namespace but hurts ubuntu" as a conclusion, a missing same-day control for `ubuntu-latest`+disk was identified and run — this changed the conclusion from "runner-dependent" to "disk beats tmpfs everywhere."
5. `DefaultThroughputTolerance` is explicitly untouched. The adopt-and-widen path considered mid-investigation is recorded as abandoned, not silently dropped.
6. The 51.5% staleness finding's cause is explicitly NOT asserted — recorded as one of two possibilities (code speedup vs. fleet hardware change) with a suggested `git bisect` path if it recurs, rather than picking one to make the writeup feel more complete.

## Deviations from Plan

### Auto-fixed / maintainer-directed issues

**1. [Rule 4 - Architectural, maintainer-approved at each step] The plan's original Namespace-adoption premise was overturned by measurement**

- **Found during:** the blocking checkpoint, across 5 rounds
- **Issue:** the plan's `must_haves.truths` assumed `bench.yml`'s jobs would end up on Namespace with a Namespace-recorded baseline. Real measurement showed this would make the perf gate materially less reliable (worse session-to-session headroom on every alternative configuration tested).
- **Fix:** `rebless` returned to `ubuntu-latest`; `headtohead` (non-gating) stays on Namespace as a deliberate, documented asymmetric choice.
- **Files modified:** `.github/workflows/bench.yml` (commit `72d404a`)
- **Verification:** `task lint:actions` passes; `rg rebless Taskfile.yml` still empty; a fresh `-trials 7` rebless dispatch on `ubuntu-latest` confirmed `regression gate passed` with 0.62% session disagreement.
- **Committed in:** `72d404a`, `335a88f`
- **Note:** this is a plan-level architectural pivot, not a bug fix — it happened through 5 explicit maintainer checkpoint decisions (Rule 4's own "ask about architectural changes" protocol), not autonomous judgment. Recorded under Rule 4 for traceability, not because it went unauthorized.

**2. [Rule 3 - Blocking] GitHub will not dispatch `workflow_dispatch` against a brand-new workflow file on a non-default branch**

- **Found during:** the CPU-topology diagnostic round
- **Issue:** a standalone `.github/workflows/bench-cpu-diag.yml` was added and immediately failed to dispatch (`HTTP 404: workflow bench-cpu-diag.yml not found on the default branch`) — a structural GitHub API constraint, not a bug in the new file.
- **Fix:** removed the standalone file; folded the same diagnostic into `bench.yml` (already registered and dispatchable from any branch) as new `job:` dispatch options instead.
- **Files modified:** `.github/workflows/bench-cpu-diag.yml` (removed), `.github/workflows/bench.yml` (gained the job)
- **Verification:** re-dispatched successfully via `bench.yml`'s new `cpu-diag` option; both jobs went green.
- **Committed in:** `eff4bde`

---

**Total deviations:** 1 architectural (maintainer-directed across the whole checkpoint investigation) + 1 blocking (GitHub API constraint).
**Impact on plan:** The architectural pivot is the plan's actual, correct outcome — the original single-checkpoint shape assumed the investigation would confirm Namespace adoption; it instead disproved it with real data, which is what the checkpoint mechanism exists to allow. No scope creep: every added file/target (diagnostics, `-scratch-fs`) is either load-bearing for the final resolution or a documented, permanently-useful capability for the deferred follow-up.

## Issues Encountered

- Sandbox DNS/network connectivity to `api.github.com` was intermittently flaky during long CI-wait periods (several `error connecting to api.github.com` failures, all transient and resolved on retry within seconds to a few minutes). No data loss; all affected `gh` calls were simple status polls, safely retried.
- An early attempt at a session-to-session disagreement calculation used the wrong pair of numbers (mislabeled Namespace-vs-ubuntu figures) while drafting a `BASELINE.md` passage — caught and corrected before committing, by recomputing from source numbers rather than trusting a prior mental note.

## User Setup Required

None — no external service configuration required. (Namespace GitHub integration was already verified present in plan 10-01.)

## Next Phase Readiness

**Blocker for plan 10-06, flagged explicitly:** 10-06's own plan file states its purpose as "moves the perf and reproducibility jobs onto Namespace... now that a Namespace-recorded baseline is committed." That premise is now false — the committed baseline is `ubuntu-latest`+disk, not Namespace, and this investigation's evidence is that moving the gate to Namespace would make it materially less reliable. **10-06 needs re-scoping before execution**, not blind execution against its current plan text: the runner-aware `CheckRegression` comparison logic it adds (wiring `Runner`/`ScratchFS` in as category-error checks) is still valid and needed work, but the "move `ci.yml`'s perf gate to Namespace" portion of its `must_haves` should be dropped or explicitly re-confirmed with the maintainer given this plan's findings.

- `internal/bench/regression.go` is untouched and ready for 10-06 to wire `Runner`/`ScratchFS` into `CheckRegression` as category-error checks (mirroring the existing `GOOS`/`GOARCH` refusal) — both fields are now populated in the committed baseline for the first time, which 10-06 needs.
- The deferred Namespace cache-volume follow-up (untested candidate storage backend) is documented in `BASELINE.md` with an explicit caveat that it would only address the storage-latency hypothesis, not the host-placement-variance hypothesis that stayed live throughout this investigation — worth a dedicated future plan, not a quick add-on.
- `ci.yml`'s perf-regression gate needs no transitional handling: it was never moved off `ubuntu-latest`, so there is no red/unreliable window to manage, unlike what Task 3's original plan text anticipated for a Namespace adoption.

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 8 files created/modified this plan verified present on disk (`internal/bench/metrics.go`, `tools/bench/runner/main.go`, `tools/bench/runner/main_test.go`, `tools/bench/cpudiag/main.go`, `.github/workflows/bench.yml`, `Taskfile.yml`, `tools/bench/baseline.json`, `tools/bench/BASELINE.md`). All 17 commit hashes listed in Task Commits verified present in `git log --all` (`d44987d`, `acd14bf`, `7111b8a`, `600c3e9`, `eff4bde`, `0279367`, `70f2acc`, `01319e0`, `47eff33`, `18debd2`, `b677f78`, `d680f71`, `4c3e21e`, `ddbaf3e`, `3b69ec0`, `72d404a`, `335a88f`).
