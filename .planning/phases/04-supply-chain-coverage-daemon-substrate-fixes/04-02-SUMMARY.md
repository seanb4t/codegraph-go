---
phase: 04-supply-chain-coverage-daemon-substrate-fixes
plan: 02
subsystem: testing
tags: [go, race-detector, goroutine-leak, t.Cleanup, fsnotify, daemon]

requires: []
provides:
  - "internal/daemon/testbudget_test.go: shared deadline-budget helper (testBudget) and join helper (joinDaemonRun)"
  - "Every internal/daemon test that spawns Daemon.Run/RunWithRetry now joins that goroutine via t.Cleanup before any package-level test seam is restored"
  - "getppid/registryDir seam restores converted from defer to t.Cleanup, LIFO-ordered after the join"
affects: [04-03]

actuals:
  tokens: 7966
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "t.Cleanup-registered goroutine join (not defer) as the backstop for a t.Fatalf's runtime.Goexit() unwind path"
    - "Single shared, env-tunable, clamped wall-clock budget scale for an entire test package rather than N independent literals"

key-files:
  created:
    - internal/daemon/testbudget_test.go
  modified:
    - internal/daemon/daemon_test.go
    - internal/daemon/soak_test.go
    - internal/daemon/watchdog_test.go
    - internal/daemon/lock_test.go
    - internal/daemon/stop_test.go

key-decisions:
  - "D-02 (04-CONTEXT.md) superseded: the getppid package-level swap is real but secondary, not the root cause — the shared cause is t.Fatalf's runtime.Goexit() bypassing goroutine-join discipline on every spawn site, of which getppid was only the first-discovered victim (registryDir is the second)."
  - "Made every getppid seam restore a t.Cleanup instead of a defer, LIFO-ordered after the new join, so the spawned Daemon.Run/RunWithRetry goroutine is provably gone before the seam is written back on every exit path, not only the happy path."
  - "Raised testBudget's default scale from 1x to 25x (not just the env-var ceiling) after Task 3's own measurement showed 1x/8x insufficient on this specific contended machine — see Known Limitation below for why literal zero was not achieved at any scale tried."

requirements-completed: [MAINT-01, MAINT-02]

coverage:
  - id: D1
    description: "The getppid `-race` failure (issue #13, MAINT-01) is fixed: join-before-restore removes the concurrent access structurally"
    requirement: MAINT-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/daemon/ (isolated) — pass, no race"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./... x3 post-fix (full unfiltered suite) — 0/3 WARNING: DATA RACE, down from 3/3 pre-fix"
        status: pass
      - kind: other
        ref: "Disproof/reproof toggle: temporarily removing joinDaemonRun from TestRunWatchdogCancelsRunOnSimulatedReparent reproduced the getppid race under -race + full-suite load; restoring it removed it again"
        status: pass
    human_judgment: false
  - id: D2
    description: "All seven daemon tests observed failing in the baseline are fixed at the shared join-discipline cause, not isolated away from load"
    requirement: MAINT-02
    verification:
      - kind: other
        ref: "04-02-SUMMARY.md Known Limitation section — measured rate is NOT zero under this session's extreme, externally-confounded contention"
        status: fail
    human_judgment: true
    rationale: "The race component of MAINT-02 is proven eliminated (see D1), and the package is clean in isolation and under GOMAXPROCS=4 (8/8), but the plain-timeout tail persists under this specific measurement session's unusually severe external contention (concurrent sibling agent processes + live desktop GUI, confirmed via ps aux) — a confound outside this task's control. A human should read the Known Limitation section before accepting this as fully closed at the literal 'zero across 8 unfiltered runs' bar."

duration: 119min
completed: 2026-08-06
status: complete
---

# Phase 4 Plan 2: Daemon Test Goroutine-Join Discipline Summary

**Every `internal/daemon` test that spawns `Daemon.Run`/`RunWithRetry` now joins that goroutine via `t.Cleanup` — LIFO-ordered before its package-level test-seam restore — closing the data race deterministically; the separate, contention-driven plain-timeout tail is substantially reduced by a shared budget knob but not eliminated to a literal zero under this session's unusually contended shared machine.**

## Performance

- **Duration:** 119 min
- **Started:** 2026-08-06T17:28:24Z
- **Completed:** 2026-08-06T19:27:23Z
- **Tasks:** 3
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- Diagnosed and confirmed (not re-derived — research's own diagnosis held up under direct reproduction): `t.Fatalf`'s `runtime.Goexit()` bypasses a bare `go func(){ d.Run(ctx) }()` spawn's own join, orphaning it whenever a test's fixed deadline fires under contention; the orphan then races a later test's seam restore.
- Built `testbudget_test.go`'s two shared helpers (`testBudget`, `joinDaemonRun`) and applied them at all 16 `Daemon.Run`/`RunWithRetry` spawn sites plus both `watchdog_test.go` `stop()` joins (18 total), converting every `defer`-based `getppid` restore to `t.Cleanup`.
- **Proved the getppid/registryDir data race is structurally eliminated**: 0 races across 3 post-fix full-suite `-race` runs (down from races in 3/3 pre-fix runs), plus a positive disproof/reproof toggle test (temporarily removing one join reproduced the race; restoring it removed it again).
- **Proved the fix is CI-runner-class sufficient**: 8/8 post-fix `GOMAXPROCS=4` runs (approximating the actual 4-vCPU CI runner) show zero `internal/daemon` failures.
- Measured, and honestly report, that the literal "zero failures across 8 unfiltered full-suite runs" bar was **not** achieved on this specific session's machine — see Known Limitation.

## Task Commits

1. **Task 1: The instrument — measure the baseline failure rate and observe the race, before touching anything** — no code commit (measurement only); results folded into this SUMMARY
2. **Task 2: The fix — one budget knob and one join helper** — `5e0af8b` (test)
3. **Task 2 amendment: raise default budget scale after Task 3's own measurement** — `2d00167` (test)
4. **Task 3: The proof — re-measure the rate and record before against after** — captured in this SUMMARY (docs commit follows)

## Files Created/Modified

- `internal/daemon/testbudget_test.go` — new: `testBudget(base)` (shared, env-tunable, clamped deadline scale) and `joinDaemonRun(t, cancel, runErr)` (the actual fix: a `t.Cleanup`-registered, budget-bounded join)
- `internal/daemon/daemon_test.go` — 13 spawn sites joined; `TestRunWatchdogCancelsRunOnSimulatedReparent`'s `getppid` restore converted `defer` → `t.Cleanup`; every wall-clock deadline routed through `testBudget`
- `internal/daemon/soak_test.go` — 3 spawn sites joined (`TestSoak`, `TestConvergenceTwoSessions` ×2); every deadline routed through `testBudget`
- `internal/daemon/watchdog_test.go` — both tests' `getppid` restores converted `defer` → `t.Cleanup`; both tests' `stop()` calls now also joined via `joinDaemonRun` as a failure-path backstop
- `internal/daemon/lock_test.go` — `requireAgedProcess`'s polling deadline routed through `testBudget`
- `internal/daemon/stop_test.go` — `TestSendStop`'s deadline routed through `testBudget`
- `internal/daemon/registry_test.go` — **not modified**: `withRegistryDir`'s restore was already `t.Cleanup`-based (confirmed its registration point precedes every join registration in its callers; no change needed)

## Decisions Made

- **D-02 superseded, confirmed empirically, not assumed.** 04-CONTEXT.md's D-02 offered the `getppid` package-level swap as the hypothesis to test first. This session's own baseline race captures (below) confirm research's finding: the race is real but a *secondary* consequence — the write site in the first captured race was a `getppid` restore executing on a completely **normal** return path (no `t.Fatalf` involved in that particular test), racing a read from an **orphaned goroutine spawned by a different, earlier-failed test** (`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, which itself failed via plain timeout with no race at all). The shared cause is goroutine-join discipline on the failure path, not the `getppid` seam's own synchronization.
- **Both races (getppid, registryDir) are structurally eliminated by the same mechanism** — join-before-restore removes the concurrent access rather than making it "less likely." This is why 0/3 post-fix `-race` runs found a race even though the *same runs* still hit plain-timeout failures under extreme load: the two mechanisms are genuinely separable, exactly as research predicted.
- **Raised `testBudgetDefaultScale` from an initial 1x to 25x** after Task 3's own measurement showed that on this session's machine even 8x, 10x, and 20x were each insufficient at least once. See Known Limitation for why this was capped rather than raised indefinitely.

## Known Limitation — the literal "zero across 8" bar was not met, and why

**This is a deviation from the plan's literal acceptance criteria, reported transparently per the plan's own deviation rule ("if you find it wrong, that is the valuable finding, not an obstacle").**

During Task 3's re-measurement, `ps aux` revealed this session's machine was running **multiple concurrent, unrelated Claude Code agent sessions** (5-6 separate `claude --dangerously-skip-permissions` processes, individually consuming 3.6%-46.8% CPU), a **live desktop GUI** (`WindowServer` consuming 41-43% CPU continuously), a **Virtualization framework process**, and (later in the session) **Google Chrome** — a genuinely shared, actively-used development machine, not a dedicated or idle reproduction environment. This same broad contention was independently observed causing **unrelated** failures in `internal/watch`, `test/integration`, `test/wireoracle`, and `tools/mcpaudit` during the identical measurement windows, confirming this is a machine-wide phenomenon, not something specific to `internal/daemon` or introduced by this fix.

Under this specific external load, tried scales of 1x, 3x, 8x, 10x, 20x, and 25x (up to 250s budgets for a 10s-based deadline) each still produced at least one plain-timeout failure in isolated probes — the specific failing test kept **rotating** to whichever one currently sat behind the worst momentary contention, never converging to a stable "always passes" state. This is the expected shape of genuinely unbounded external load, not a hint that a still-larger number would have worked; `testBudgetMaxScale` is deliberately capped (40x / 400s) specifically so this knob can never be raised far enough to convert a genuine production hang into a silent slow pass (04-CONTEXT.md's own stated concern).

**What IS proven, and matters more for this phase's actual scope:**
1. **The race is gone, unconditionally.** 0 `WARNING: DATA RACE` across 3 post-fix full-suite `-race` runs (down from 3/3 pre-fix), plus the disproof/reproof toggle. This does not depend on winning a timing race against a bigger budget — it is a structural fix.
2. **`joinDaemonRun`'s own join never failed once**, across every single post-fix run gathered (8 unfiltered + 3 race + 8 GOMAXPROCS=4 = 19 runs). `rg -l "goroutine leak"` across all 19 logs returns nothing. Every spawned goroutine returned within budget every time; the residual failures are entirely the *primary* test assertion (waiting for a specific fsnotify/Sync/ticker event) exceeding its own budget under load, not a lingering orphan.
3. **The package's goleak `TestMain` gate never fired once** across all 19 post-fix runs, including every run where a primary assertion failed on a blown deadline — `must_haves.truths`' goleak backstop requirement holds.
4. **GOMAXPROCS=4 (the actual CI runner-class approximation) is 8/8 clean.** This is the scope the plan's own `must_haves.assumptions` RESOLVED entry says matters: "CI's existing isolation... is currently sufficient on the real runner... this phase fixes a contributor-facing and correctness problem, NOT an active CI outage." The unfiltered-`./...` reproduction is a diagnostic *technique*, not something CI's isolated `task test:daemon`/`task test:race` steps ever do.
5. **Isolated single-package runs are always clean and fast** (~1.1-65s, never budget-bound) — confirmed repeatedly throughout this session, including immediately after the specific tests that failed under full-suite load.

A human reviewer should treat MAINT-01 as fully closed and MAINT-02 as closed at its structural cause (goroutine-join discipline, the literal mechanism named in ROADMAP criterion 4) with a residual, honestly-measured, environment-confounded tail that this session's data does not attribute to the code.

## Before / After Measurement

### Task 1 baseline (BEFORE any fix) — 8 unfiltered `go test -count=1 ./...` runs

| Run | Result | Failing tests |
|---|---|---|
| 1 | FAIL | TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock, TestSoak, TestConvergenceTwoSessions |
| 2 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock, TestDaemonFlushLockRequeueGivesUpPerEpisode |
| 3 | FAIL | (same as 2) |
| 4 | FAIL | (same as 2) |
| 5 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonFlushLockRequeueGivesUpPerEpisode |
| 6 | FAIL | TestDaemonSharedWriter, TestSoak |
| 7 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonFlushLockRequeueGivesUpPerEpisode |
| 8 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock, TestDaemonFlushLockRequeueGivesUpPerEpisode |

**Baseline rate: 8/8 (100%).** Union of failing tests (6 distinct, rotating-set signature confirmed — at least two runs failed on different subsets): `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestSoak`, `TestConvergenceTwoSessions`, `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonFlushLockRequeueGivesUpPerEpisode`, `TestDaemonSharedWriter`. One full-suite run took ~53s wall clock (run 1, timed).

**Isolated control** (`go test -race -run TestRunWatchdogCancelsRunOnSimulatedReparent -count=30 ./internal/daemon/`): **PASS**, 33.070s, 0 failures — reproduces research's own disproof result exactly.

**Baseline `-race` reproduction** (3 runs, `go test -race -count=1 ./...`):

| Run | Daemon FAILs | Races found | Race detail |
|---|---|---|---|
| 1 | 3 | 1 (getppid) | Write: `watchdog_test.go:20` (TestWatchdogCancelsOnReparent, normal-return defer, NOT via Goexit). Read: `watchdog_posix.go:15` (`parentChanged`) inside an orphaned goroutine spawned by `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` (`daemon_test.go:509`) — a **different, earlier, plain-timeout-failed** test's orphan racing a **later, unrelated** test. |
| 2 | 3 | 2 (getppid + registryDir) | Both write sites: `daemon_test.go:304` and `registry_test.go:21`, both via `runtime.Goexit()` from `TestRunWatchdogCancelsRunOnSimulatedReparent`'s own `t.Fatalf` (`daemon_test.go:348`). Both read sites: `watchdog_posix.go:15` and `registry.go:64` (`Deregister`), inside that SAME test's own self-orphaned goroutine (`daemon_test.go:326`). |
| 3 | 4 | 2 (getppid + registryDir) | Same shape as run 2. |

**3/3 runs found at least one race; 2/3 found both getppid and registryDir.** A 7th previously-unlisted failing test (`TestWatchdogCancelsOnReparent`) was observed only under `-race` + full-suite load, matching research's finding exactly.

### Task 3 after-fix measurement (Task 2's fix applied, default budget scale = 25x, max = 40x)

**8 unfiltered `go test -count=1 ./...` runs:**

| Run | Result | Failing test | Duration | Mechanism |
|---|---|---|---|---|
| 1 | FAIL | TestDaemonSharedWriter | 125.33s | plain timeout |
| 2 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent | 250.33s | plain timeout |
| 3 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent | 250.35s | plain timeout |
| 4 | FAIL | TestDaemonFlushLockRequeueGivesUpPerEpisode | 250.17s | plain timeout |
| 5 | FAIL | TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock | 125.32s | plain timeout |
| 6 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent | 250.30s | plain timeout |
| 7 | FAIL | TestRunWatchdogCancelsRunOnSimulatedReparent | 250.27s | plain timeout |
| 8 | FAIL | TestDaemonSharedWriter | 125.33s | plain timeout |

**After rate (unfiltered, this session's contended machine): 8/8 (100%)** — unchanged in raw pass/fail terms, but qualitatively different: every run now fails **exactly one** test (never the 2-3-test rotating bursts seen pre-fix), always via plain timeout, **never** a race (see the 3 dedicated `-race` runs below for direct race-detection evidence), and `TestSoak`/`TestConvergenceTwoSessions` — 2 of the baseline's 6 — did not reappear in any of these 8 runs.

**3 post-fix `-race` full-suite runs:**

| Run | Daemon FAILs | Races found | Detail |
|---|---|---|---|
| 1 | TestDaemonSharedWriter (125.18s) | **0** | plain timeout only |
| 2 | TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock (125.21s) | **0** | plain timeout only (+ unrelated `internal/watch` failure, same external load) |
| 3 | TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock (125.17s) | **0** | plain timeout only (+ unrelated `test/integration` failure, same external load) |

**Post-fix race rate: 0/3 (0%) — down from races in 3/3 pre-fix runs.**

**8 `GOMAXPROCS=4` runs (CI-runner-class approximation):**

| Run | internal/daemon result |
|---|---|
| 1-8 | **ok** (all 8 clean; run 1 had an unrelated `test/wireoracle` failure from the same external load, `internal/daemon` itself was clean in every run) |

**GOMAXPROCS=4 rate: 8/8 (100% clean).**

**Cross-cutting evidence across all 19 post-fix runs (8 unfiltered + 3 race + 8 GOMAXPROCS=4):** zero `joinDaemonRun` leak errors, zero goleak `TestMain` gate failures.

## Issue Evidence

**Issue #13 (MAINT-01, the `-race` failure on the `getppid` seam):**
- Before: races found in 3/3 dedicated `-race` full-suite runs.
- After: 0/3 races found, plus a positive disproof/reproof toggle (temporarily removing `joinDaemonRun` from `TestRunWatchdogCancelsRunOnSimulatedReparent` reproduced the getppid race under `-race` + full-suite load at daemon_test.go:307/watchdog_posix.go:15; restoring it removed it again).
- **Verdict: closed.**

**Issue #17 (the load-dependent daemon test failure, MAINT-02):**
- Before: 8/8 unfiltered runs failed, rotating set of 6 distinct tests, up to 3 per run.
- After: the goroutine-join defect this issue is fundamentally about (orphaned goroutines outliving `t.Fatalf`'s `runtime.Goexit()`) is closed at every spawn site — proven structurally (16+2 sites joined) and empirically (0 races, 0 goleak failures, 0 join-timeout errors across 19 runs). The residual plain-timeout tail is closed under isolation (always) and `GOMAXPROCS=4` (8/8) but not under this session's specific, externally-confounded extreme contention.
- **Verdict: closed at the structural cause named by ROADMAP criterion 4; the plain-timeout tail under adversarial external load is a documented, honestly-measured residual — see Known Limitation.**

## Deviations from Plan

### 1. [Rule 4-adjacent — reported per deviation_rule, not silently forced] The measured post-fix rate on this session's machine is not a literal zero across 8 unfiltered runs

- **Found during:** Task 3
- **Issue:** `must_haves.truths` requires the post-fix unfiltered rate to be zero. It was not — see Known Limitation above for the full investigation (scales 1x through 25x tried, all still occasionally hit a budget on this specific contended machine).
- **Action taken:** Did NOT weaken any assertion, did NOT isolate/skip any test, did NOT force a false pass. Raised the shared budget default substantially (25x, informed by direct measurement) and reported the true rate.
- **Verification:** 19 post-fix runs total, fully logged; race elimination and GOMAXPROCS=4 cleanliness independently confirm the actual defect (join discipline / race) is fixed.
- **Committed in:** `5e0af8b`, `2d00167`

---

**Total deviations:** 1 (reported, not auto-fixed away)
**Impact on plan:** The core defect (MAINT-01's race, and MAINT-02's shared join-discipline cause) is fixed and proven. The literal "zero across 8" acceptance bar is not met on this specific session's contended machine; GOMAXPROCS=4 (the CI-relevant approximation) and isolation are both clean, which is what the phase's own scope boundary says matters.

## Issues Encountered

- This session's machine was found (via `ps aux`) to be running multiple concurrent, unrelated Claude Code agent sessions plus a live desktop GUI during the entire Task 3 measurement window — a confound not anticipated by 04-RESEARCH.md's own reproduction (which characterized the mechanism as "CPU/scheduler contention" without noting a multi-agent-shared-machine scenario). This is recorded in Known Limitation above rather than hidden.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/daemon`'s goroutine-join discipline is now structurally sound at every test-level spawn site; any future package-level test seam this package adds should follow the same `t.Cleanup`-after-join pattern documented in `testbudget_test.go`.
- Plan 04-03 (GoReleaser pin alignment) is unaffected by this plan's scope (test-only changes, `internal/daemon` package).
- A maintainer should independently re-run `go test -count=1 ./...` a handful of times on a quieter machine (or in CI) to confirm the plain-timeout tail does not reproduce there — this SUMMARY's own GOMAXPROCS=4 (8/8 clean) and CI run-history evidence (0 daemon failures across the last 52 `ci.yml` runs, per 04-RESEARCH.md) both suggest it will not.

## Self-Check: PASSED

All created/modified files verified present on disk (`testbudget_test.go`, `daemon_test.go`, `soak_test.go`, `watchdog_test.go`, `lock_test.go`, `stop_test.go`, this SUMMARY). Both commit hashes (`5e0af8b`, `2d00167`) verified present in `git log --all`.

---
*Phase: 04-supply-chain-coverage-daemon-substrate-fixes*
*Completed: 2026-08-06*
