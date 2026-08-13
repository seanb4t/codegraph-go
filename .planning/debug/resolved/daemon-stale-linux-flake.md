---
status: resolved
resolved_causes: "A (test harness, commit b51fefc) + B (product, commit 326aba7)"
separate_finding: "TestConvergenceTwoSessions soak_test.go:232 watcher-arming race — NOT fixed, NOT bundled"
trigger: "internal/daemon tests fail on Linux CI only — StopAll/StopMatching return [] because isStale's /proc corroboration rejects test records built with StartedAt: time.Now() against the test binary's real start time (procStartTimeSlack=5s). Suspected test-harness bug, not product bug. Need Linux -count=N repro to confirm, then fix all affected tests."
created: 2026-07-31
updated: 2026-08-13
tdd_mode: true
goal: find_and_fix
---

# Debug Session: daemon-stale-linux-flake

## Symptoms

**Expected behavior:**
`go test ./internal/daemon/...` passes on Linux CI, as it does on macOS. `StopAll()` should return exactly the one live record it was given; `StopMatching("/repo/a")` should return exactly the matching record.

**Actual behavior:**
On Linux CI only, `StopAll()` and `StopMatching()` return an empty slice `[]`. The live record is silently skipped. Passes 100% of the time on macOS (local darwin/arm64 verified this session: `go test ./internal/upgrade/...` ok; daemon package green locally).

**Error messages (verbatim, CI run 30604258476, main@3738acc, job "Test internal/daemon (isolated, -count=1)"):**
```
--- FAIL: TestStopAll (0.01s)
    --- FAIL: TestStopAll/signals_only_the_live_record,_skips_the_stale_one (0.00s)
        stop_test.go:116: StopAll: got [], want exactly the live record {PID:14912 StartedAt:2026-07-31 04:33:46.134294393 +0000 UTC RepoRoot:/live/repo}
    --- FAIL: TestStopAll/de-duplicates_targets_by_pid (0.00s)
        stop_test.go:150: StopAll: got 0 records, want exactly 1 (de-duplicated by pid): []
--- FAIL: TestStopMatching (0.01s)
    --- FAIL: TestStopMatching/signals_only_the_matching_repoRoot (0.00s)
        stop_test.go:181: StopMatching: got [], want exactly {PID:14912 StartedAt:2026-07-31 04:33:46.139701765 +0000 UTC RepoRoot:/repo/a}
FAIL	github.com/seanb4t/codegraph-go/internal/daemon	4.409s
```

**Timeline:**
Surfaced at the phase 09-07 blocking gate on main@a1c298f, where **9** daemon tests failed on Linux (TestAcquireRejectsLiveLock, TestStopAll, TestWatchdogCancelsOnReparent, TestIsStale, TestConvergenceTwoSessions, TestRegistryListPrunesStale, TestStopMatching, TestUnlockStaleOnly, TestAcquireConcurrentRaceOnlyOneWinner). On main@3738acc only **2** failed (TestStopAll, TestStopMatching). Recorded historically as "Phase-8 gap #5". Not caused by the grpc v1.82.1 bump (a0cfceb) — internal/daemon is not on the grpc reachability chain and that diff was 2 lines of go.mod/go.sum.

**Reproduction:**
Linux only. Requires the test binary to have been alive longer than the corroboration slack when the subtest runs. Not yet reproduced under a controlled `-count=N` loop — THAT IS THE FIRST TASK.

## Leading Hypothesis (orchestrator pre-analysis — VERIFY, do not assume)

Tests construct a physically impossible record: the *current* process's PID paired with a `StartedAt` of *now*.

`internal/daemon/stop_test.go:99` (also `:137`, `:166`):
```go
live := Record{PID: os.Getpid(), StartedAt: time.Now().UTC(), RepoRoot: "/live/repo"}
```

`Register` (`internal/daemon/registry.go:48`) stores the caller's Record **verbatim** — it does not stamp `StartedAt` itself, so nothing normalizes the bad value.

`stopTargets` (`internal/daemon/stop.go:69`) re-derives staleness immediately before signaling:
```go
if isStale(lockInfo{PID: rec.PID, StartedAt: rec.StartedAt}) {
    continue   // → record skipped → got = []
}
```

`isStale` (`internal/daemon/lock.go:82`):
```go
if actualStart, ok := processStartTime(info.PID); ok {
    if !startTimesCorroborate(info.StartedAt, actualStart) {
        return true
    }
}
```
with `procStartTimeSlack = 5 * time.Second` (`lock.go:103`).

**Why Linux-only:** `procstart_other.go` (build tag `!linux`) returns `(time.Time{}, false)` unconditionally, so the corroboration branch never runs on macOS — `isStale` degrades to liveness-only and the tests pass deterministically. Only `procstart_linux.go` reads `/proc/<pid>/stat` and can produce a mismatch.

**Why flaky rather than always-red:** two compounding factors.
1. Go compiles test files alphabetically, so `stop_test.go` runs 5th — after `daemon_test.go` (30KB, timeouts to 10s), `lock_test.go`, `registry_test.go`, `soak_test.go`. Elapsed binary lifetime when a given subtest runs varies with runner speed. Failing run clocked the package at 4.409s — near the boundary.
2. `procstart_linux.go` computes `time.Unix(bootUnix+ticks/procStatClockTicks, 0)`. `bootUnix` (from `/proc/stat` btime) is whole seconds and `ticks/100` is integer division — two independent downward truncations, so the derived start can be up to ~2s EARLIER than reality, inflating the delta. Effective margin ≈ 3s, not the nominal 5s.

**Prediction this hypothesis makes (use to falsify):** on Linux, injecting an artificial delay of > ~5s before `TestStopAll` runs should make it fail 100% of the time; setting the test record's `StartedAt` to `processStartTime(os.Getpid())` should make it pass 100% of the time. If either prediction fails, the hypothesis is wrong.

**Assessment:** test-harness bug, NOT a product bug. In production a daemon calls `Register` within milliseconds of starting, so recorded ≈ actual and corroboration passes. The 5s slack legitimately defends against PID reuse (WR-02 / T-07-04-01), where the gap is necessarily large.

## Current Focus

hypothesis: CONFIRMED + EXTENDED. Any `lockInfo`/`Record` naming the CURRENT process with `StartedAt: time.Now()` fails `isStale`'s /proc corroboration once the process has been alive longer than the EFFECTIVE slack (~3s, not 5s — /proc truncation eats ~1.6-2.0s). This occurs in TWO independent places, and the observed failure set is the union (AND-gate fired — see rca_branching below).
test: Linux container repro achieved; delta instrumented directly.
expecting: n/a — measured.
next_action: UNBLOCKED — maintainer chose Option A (apply the product fix). Sequence: (1) commit Cause A alone; (2) write the Cause-B product test and demonstrate RED against unmodified product code on Linux; (3) add selfStartedAt() in lock.go, use at acquire() + daemon.go Register(); (4) GREEN at -count=20, full package, ZERO skips on Linux; (5) confirm TestIsStaleStillDetectsPIDReuseOnAgedProcess still green; (6) re-verify darwin; (7) commit Cause B as fix(daemon).

reasoning_checkpoint:
  hypothesis: "Cause A (test-harness): tests build self-referencing records with StartedAt=time.Now(); once the test binary is >~3s old, startTimesCorroborate fails and isStale reports the CURRENT LIVE process as stale. Cause B (product): acquire() (lock.go:152) stamps StartedAt=time.Now() for os.Getpid(), so a lock written by any process >~3s past its own start is stale the instant it is written."
  confirming_evidence:
    - "Standalone /proc probe on linux/arm64: delta(time.Now(), processStartTime(self)) = 1.949s at t=0 and crosses procStartTimeSlack at ~3.05s of process lifetime; corroborate flips true->false between elapsed=3s and elapsed=4s."
    - "In-package probe (aged 6.0s): isStale(self, StartedAt=time.Now()) = TRUE; isStale(self, StartedAt=processStartTime(self)) = FALSE. Fix direction directly validated."
    - "In-package probe (aged 6.0s): acquire(dir) then readLock(dir) -> isStale(info) = TRUE. A second acquire(dir) then SUCCEEDED over that live lock."
    - "Full-package Linux run: 8 daemon tests fail. Same tests run in isolation (young binary): ALL PASS. Elapsed-lifetime dependence proven, not platform incompatibility."
  falsification_test: "If the measured delta stayed under procStartTimeSlack at the elapsed lifetimes at which the tests fail, or if isStale(self, StartedAt=processStartTime) also returned true, the hypothesis would be dead. Neither held."
  fix_rationale: "Cause A is a test-harness defect: a record naming os.Getpid() must carry that process's ACTUAL start time, because that is precisely what isStale corroborates against. Deriving StartedAt from processStartTime(pid) makes the fixture physically possible instead of physically impossible. It does not touch procStartTimeSlack, does not remove startTimesCorroborate, and does not weaken WR-02."
  blind_spots: "Verified on linux/arm64 (docker) only, not linux/amd64 as CI runs; USER_HZ assumed 100 (as the product does). Cause B is left UNFIXED per constraints, so 3 tests stay red on Linux."
  candidate_causes:
    - "code (product): acquire()/Register() stamp time.Now() for the current pid — CONFIRMED (Cause B)"
    - "code (test harness): fixtures pair os.Getpid() with time.Now() — CONFIRMED (Cause A)"
    - "environment: container /proc/stat btime vs host boot time skew — RULED OUT (deltas are consistent and small; corroboration succeeds on a young binary)"
    - "data: malformed /proc parse — RULED OUT (processStartTime returned ok=true with sane values)"
  and_gate: "YES — fired. The 8-test failure set is NOT explained by one cause. Cause A alone explains TestStopAll/TestStopMatching/TestIsStale/TestUnlockStaleOnly/TestAcquireRejectsLiveLock/TestRegistryListPrunesStale (tests that build their own records). Cause B alone explains TestAcquireConcurrentRaceOnlyOneWinner/TestRunWithRetryCtxCancelDuringSleep/TestConvergenceTwoSessions (tests that never build a record — they go through acquire()). Both require the shared enabling condition 'process older than effective slack'."

## Evidence

- timestamp: 2026-07-31 — CI run 30604258476 on main@3738acc: 2 of 5 jobs' worth of daemon tests failed with `got []`; package elapsed 4.409s. govulncheck + perf gate green in the same run.
- timestamp: 2026-07-31 — Failure SET varies across runs (9 → 2) on the same platform, which rules out a deterministic Linux incompatibility and indicates timing dependence.
- timestamp: 2026-07-31 — Local macOS (darwin/arm64): daemon tests pass; `procstart_other.go` makes the corroboration branch unreachable there.
- timestamp: 2026-07-31 — REPRO ACHIEVED. Linux container (docker, golang:1.26, linux/arm64), test binary built from the repo and run directly. Full-package `-count=1` FAILS with 8 tests: TestRunWithRetryCtxCancelDuringSleep, TestUnlockStaleOnly/live_pid_lock_is_refused, TestAcquireRejectsLiveLock, TestIsStale, TestAcquireConcurrentRaceOnlyOneWinner, TestRegistryListPrunesStale, TestConvergenceTwoSessions, TestStopAll (2 subtests), TestStopMatching (1 subtest). Verbatim: `stop_test.go:116: StopAll: got [], want exactly the live record` — byte-identical shape to CI run 30604258476.
- timestamp: 2026-07-31 — Standalone /proc probe (replica of procstart_linux.go) on linux/arm64: `processStartTime(self)` is 1.949s EARLIER than `time.Now()` at process start (the predicted double truncation: whole-second btime + integer ticks/100). Corroboration holds at elapsed 0s/1s/2s/3s and FAILS at 4s/5s/6s — i.e. effective margin is ~3.05s of process lifetime, not the nominal 5s. Hypothesis's quantitative prediction CONFIRMED.
- timestamp: 2026-07-31 — In-package probe, binary aged to 6.0s: `isStale(lockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})` = TRUE (the current, live process reported stale). `isStale(lockInfo{PID: os.Getpid(), StartedAt: processStartTime(os.Getpid())})` = FALSE. CAUSE A confirmed and fix direction validated in one experiment.
- timestamp: 2026-07-31 — Isolation control: every failing test PASSES when run alone with `-test.run` against a freshly-started (young) binary. Rules out platform incompatibility; proves elapsed-lifetime dependence.
- timestamp: 2026-07-31 — SECOND ROOT CAUSE (product). In-package probe, binary aged to 6.0s: `acquire(dir)` succeeds, then `readLock(dir)` -> `isStale(info)` = TRUE. The lockfile is stale the instant acquire() writes it. A second `acquire(dir)` then SUCCEEDED over that live lock. Mechanism: `lock.go:152` stamps `StartedAt: time.Now().UTC()` for `os.Getpid()`, but `startTimesCorroborate` compares that against the OS-reported PROCESS START time. On Linux this defeats the single-writer invariant (INDX-05 / T-04-07-01) for any process that calls acquire more than ~3s after its own start. Explains TestAcquireConcurrentRaceOnlyOneWinner (16/32 racers "won"), TestRunWithRetryCtxCancelDuringSleep, TestConvergenceTwoSessions — none of which construct a Record themselves.
- timestamp: 2026-07-31 — CAUSE B TDD RED ESTABLISHED (post-decision). Two new tests written against UNMODIFIED product code, both forcing the aged-process condition via requireAgedProcess so the failure is deterministic rather than a timing coin flip. Linux container, `-count=1 -v`, verbatim:
  ```
  === RUN   TestRunRegistersRecordThatSurvivesPruningOnAgedProcess
      daemon_test.go:216: timed out waiting for a registry record with pid=7860 repoRoot=/tmp/TestRunRegistersRecordThatSurvivesPruningOnAgedProcess1600740405/002
  --- FAIL: TestRunRegistersRecordThatSurvivesPruningOnAgedProcess (9.43s)
  === RUN   TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck
      lock_test.go:354: acquire wrote a lock that is already stale the instant it was written: recorded StartedAt=2026-07-31 15:54:56.055360298 +0000 UTC, OS-reported start of pid 7860=2026-07-31 15:54:45 +0000 UTC, delta 11.055s exceeds procStartTimeSlack 5s — acquire must record the process's actual start time, not time.Now()
  --- FAIL: TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck (0.00s)
  FAIL	github.com/seanb4t/codegraph-go/internal/daemon	9.439s
  ```
  Determinism confirmed at `-count=3`: 6/6 failures (2 tests x 3 iterations), zero passes. The measured delta (11.055s) is the process's full lifetime at that point, confirming the recorded value tracks wall-clock-at-acquire rather than process start.
- timestamp: 2026-07-31 — CAUSE B GREEN. selfStartedAt() added in lock.go, used at acquire() and at daemon.go's Register(). Same four aged-process tests on Linux `-count=1 -v`: all PASS, no skips. TestIsStaleStillDetectsPIDReuseOnAgedProcess still PASSES all 5 subtests including "recorded exactly at the slack boundary" — WR-02 / T-07-04-01 PID-reuse detection is intact, narrowed only in that both sides now derive from the same clock.
- timestamp: 2026-07-31 — CAUSE B A/B PROOF (same isolated setup, `-run TestConvergenceTwoSessions -count=30`, Linux). PRE-fix: 10 PASS then 20 FAIL — a MONOTONIC transition at ~3.3s of accumulated runtime, exactly the predicted effective window; every failure `soak_test.go:192: session B never observed ErrLockLive while session A held the lock` at 2.02s (the 2s deadline). That message IS the single-writer-invariant violation: A's lock went self-stale, so B acquired ALONGSIDE A instead of deferring. POST-fix: 30 PASS / 0 FAIL. The monotonic aging failure mode is gone.
- timestamp: 2026-07-31 — FULL-PACKAGE LINUX VERIFICATION, post-fix, ZERO skips (the platform skip is Linux-inapplicable). Run 1 `-count=20`: 1178 pass, 2 fail. Run 2 `-count=20`: 1180 pass, 0 fail, exit 0. 40 total iterations; the only 2 failures were TestConvergenceTwoSessions at soak_test.go:232.
- timestamp: 2026-07-31 — SEPARATE FINDING (not bundled, not fixed). The residual 2/40 TestConvergenceTwoSessions failure is a DIFFERENT defect from Cause B, distinguished on four independent axes: different assertion (soak_test.go:232 vs :192), different message ("did not converge and become the sole writer after session A exited" vs "never observed ErrLockLive"), different duration (5.27s / the 5s deadline vs 2.02s / the 2s deadline), and different distribution (scattered at iterations 8 and 14 with passes after, vs monotonic-after-aging). Mechanism: the failure occurs AFTER `waitForLock` at soak_test.go:226 succeeded, so B DID acquire the lock — staleness is not implicated at all. `waitForLock` only proves the lockfile exists, but `Run()` calls `acquire()` (daemon.go:228) well before `watch.Open()` (daemon.go:268), with Register + startWatchdog in between; a convB.go create landing in that window is never delivered by fsnotify and B never syncs. The test's own comment at soak_test.go:221-225 claims waitForLock closes this race — it does not. Post-fix this cannot be a corroboration false-stale: recorded and actual now derive from processStartTime for the same pid, so the delta is exactly 0. Did NOT reproduce on darwin `-count=20` (20/20) nor under `-run 'TestSoak|TestConvergenceTwoSessions' -count=20` (20/20) — needs full-package load. Recommend a separate fix that waits for B's watcher to be armed (the onWatchOpen seam already exists on Daemon) rather than for its lockfile.
- timestamp: 2026-07-31 — TestWatchdogCancelsOnReparent (in the historical 9-test set) does NOT involve isStale/processStartTime/os.Getpid at all (watchdog_test.go drives a synthetic `getppid` seam). It passed in the container repro. Its historical failure is an unrelated 2s-timeout flake under CI load — OUT OF SCOPE for this session.
- timestamp: 2026-08-13 — ARCHIVAL-TIME RE-VERIFICATION (the archival step below never ran on 07-31; the session sat in `.planning/debug/` with `status: resolved` for 13 days). Both claimed commits exist and their file sets match `files_changed` exactly: b51fefc `test(daemon): derive self-referencing fixture StartedAt from the OS, not time.Now()` touches lock_test.go/registry_test.go/stop_test.go (+131/-8); 326aba7 `fix(daemon): record the process's actual start time in locks and registry records` touches daemon.go/daemon_test.go/lock.go/lock_test.go (+135/-11). Both fixes are still live in HEAD, not reverted: `selfStartedAt()` is defined at lock.go:142 and called at lock.go:186 (acquire) and daemon.go:241 (Register); `startedAtFor` at lock_test.go:34; all three guard tests present (lock_test.go:337, lock_test.go:297, daemon_test.go:199) plus `requireAgedProcess` at lock_test.go:236. Prohibitions still honoured in current code: `procStartTimeSlack = 5 * time.Second` (lock.go:108) and `startTimesCorroborate` still called from `isStale` (lock.go:87). darwin/arm64 re-run `go test ./internal/daemon/ -count=1`: ok 64.820s, exit 0.

## Eliminated

- hypothesis: Caused by the google.golang.org/grpc v1.82.0→v1.82.1 bump (a0cfceb). — REASON: internal/daemon is not on the grpc reachability chain (internal/upgrade → sigstore-go/pkg/verify → rekor-tiles/v2 → grpc); the diff was 2 lines in go.mod/go.sum; and the failures predate the bump (present on main@a1c298f).

## Constraints

- TDD mode is ON: a failing test must exist before the fix.
- DO NOT weaken `procStartTimeSlack` or delete the corroboration check to make tests pass — it implements a PID-reuse security mitigation (WR-02, T-07-04-01). Changing product behavior here is a separate decision requiring maintainer sign-off, not part of a test fix.
- Confirm whether the other ~6 historically-failing tests (TestIsStale, TestAcquireRejectsLiveLock, TestConvergenceTwoSessions, TestRegistryListPrunesStale, TestUnlockStaleOnly, TestAcquireConcurrentRaceOnlyOneWinner, TestWatchdogCancelsOnReparent) share the same `os.Getpid()` + `time.Now()` root cause; if so one shared helper should fix all of them.
- A single green run does NOT prove this fixed (prior finding: "one green run is not de-flaked"). Verification requires a Linux `-count=N` loop.

## Resolution

root_cause: |
  TWO independent causes (AND-gate fired), sharing one enabling condition — the process
  being older than the EFFECTIVE corroboration window (~3s, not the nominal 5s: /proc's
  derived start time reads 1.6-2.0s early due to whole-second btime + integer ticks/100).

  CAUSE A (test harness — FIXED here): fixtures paired `os.Getpid()` with
  `StartedAt: time.Now()`, describing a physically impossible process. Once the test
  binary outlived the window, isStale's WR-02 corroboration correctly rejected them as
  PID-reuse forgeries. Affected: TestStopAll, TestStopMatching, TestIsStale,
  TestUnlockStaleOnly, TestAcquireRejectsLiveLock, TestRegistryListPrunesStale.

  CAUSE B (product — NOT fixed, needs maintainer decision): `acquire()` (lock.go:152)
  and `Register()` (daemon.go:241) stamp `StartedAt: time.Now().UTC()` for `os.Getpid()`,
  but `startTimesCorroborate` compares that against the OS-reported PROCESS START time.
  Any process calling acquire/Register more than ~3s after its own start writes a
  lock/record that is stale the instant it is written. Proven impact: a second
  `acquire()` SUCCEEDED over a live lock — the single-writer invariant (INDX-05 /
  T-04-07-01) is defeated on Linux. Affected: TestAcquireConcurrentRaceOnlyOneWinner,
  TestRunWithRetryCtxCancelDuringSleep, TestConvergenceTwoSessions,
  TestRunRegistersRecordAfterAcquire, TestRunDeregistersRecordOnCleanShutdown,
  TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonSharedWriter.

fix: |
  BOTH causes fixed, in two distinct commits.

  CAUSE A — commit b51fefc `test(daemon):`. Added `startedAtFor(pid int) time.Time`
  (lock_test.go) deriving StartedAt from `processStartTime(pid)` with a
  `time.Now().UTC()` fallback (dead pids / non-Linux), and routed every
  self-referencing fixture through it. Added a deterministic aged-process
  regression guard and a WR-02 PID-reuse security guard with boundary neighbours.
  No product file touched.

  CAUSE B — commit 326aba7 `fix(daemon):`, per maintainer decision (Option A).
  Added `selfStartedAt()` in lock.go: returns `processStartTime(os.Getpid())` when
  ok, else `time.Now().UTC()`. Used at `acquire()` (lock.go) and at Run()'s
  `Register(Record{...})` (daemon.go). Corrected the procStartTimeSlack doc comment,
  which described the now-removed time.Now() behaviour as fact. Covered by two new
  deterministic tests (TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck,
  TestRunRegistersRecordThatSurvivesPruningOnAgedProcess).

  Prohibitions honoured: procStartTimeSlack VALUE unchanged (5s); the
  startTimesCorroborate call in isStale unchanged; procstart_linux.go's truncation
  untouched (with both sides now deriving from processStartTime for the same pid the
  delta is exactly 0, so the truncation cancels — measured, not assumed); no
  workflow/goreleaser/release-please/baseline.json files touched; no `go mod tidy`
  (go.mod and go.sum verified unmodified after every container run).

verification: |
  Repro: docker golang:1.26 linux/arm64, test binary built from repo and run directly.
  - RED (pre-fix, new test isolated): 8/8 runs FAIL, deterministic.
  - RED control (pre-fix binary, identical skip set, -count=5): 6 Cause-A tests fail 4/5.
  - GREEN (post-fix, -count=20, Cause-B tests skipped): PASS, zero failures.
  - Security guard TestIsStaleStillDetectsPIDReuseOnAgedProcess: GREEN both pre- and
    post-fix (5 subtests incl. exact slack boundary) — the fix narrows fixtures, not the check.
  - darwin/arm64 host: `go test ./internal/daemon/ -count=1` ok 7.028s; new tests SKIP
    correctly (corroboration branch unreachable). gofmt clean, go vet clean (both OSes).
  CAUSE B (post-decision, committed):
  - TDD RED: both new tests fail deterministically against UNMODIFIED product code,
    6/6 across -count=3. Verbatim failures recorded in Evidence above.
  - TDD GREEN: both pass after the fix.
  - A/B control, same isolated setup, -count=30: PRE-fix 10 PASS / 20 FAIL
    (monotonic at the aging threshold, soak_test.go:192, 2.02s);
    POST-fix 30 PASS / 0 FAIL.
  - Full package Linux, ZERO skips: run 1 -count=20 = 1178 pass / 2 fail;
    run 2 -count=20 = 1180 pass / 0 fail, exit 0. 40 iterations total.
    The only 2 failures were the SEPARATE soak_test.go:232 watcher-arming race
    documented in Evidence — a different assertion, message, duration and
    distribution from Cause B, and reached only AFTER B successfully acquired the
    lock (so staleness is not implicated).
  - TestIsStaleStillDetectsPIDReuseOnAgedProcess GREEN post-fix, all 5 subtests
    including the exact slack boundary — WR-02 / T-07-04-01 narrowed, not disabled.
  - darwin/arm64 host, full package -count=20: ok 139.142s, 1000 pass, 0 fail.
    The 4 aged-process tests SKIP correctly there (20 each), confirming the
    expected processStartTime ok=false -> time.Now() fallback -> unchanged
    behaviour. gofmt clean, go vet clean.

  NOT proven — gaps that must not be dropped from the writeup:
  - linux/amd64, which is CI's ACTUAL architecture, was NOT tested. All Linux
    verification ran on linux/arm64 under Docker. Not available in this environment.
  - USER_HZ != 100 systems remain untested; procStatClockTicks is hardcoded to 100.
    The fix REDUCES exposure here (both sides derive from the same derivation, so a
    wrong USER_HZ cancels for self-owned records) but does not validate it.
  - The residual TestConvergenceTwoSessions race is characterized and argued to be
    pre-existing and independent, but was NOT directly observed on pre-fix code
    (pre-fix the test dies earlier, at :192, before ever reaching :232).

files_changed:
  # commit b51fefc — test(daemon), Cause A
  - internal/daemon/lock_test.go (startedAtFor helper, writeLock, TestIsStale, 2 new tests)
  - internal/daemon/stop_test.go (4 self-referencing fixtures)
  - internal/daemon/registry_test.go (1 self-referencing fixture)
  # commit 326aba7 — fix(daemon), Cause B
  - internal/daemon/lock.go (selfStartedAt(), used at acquire(); procStartTimeSlack doc corrected)
  - internal/daemon/daemon.go (Run's Register uses selfStartedAt())
  - internal/daemon/lock_test.go (TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck)
  - internal/daemon/daemon_test.go (TestRunRegistersRecordThatSurvivesPruningOnAgedProcess)

oracle_type: derived
  # The assertions are derived from the package's own contract: a lock must
  # exclude a second acquirer (ErrLockLive), and a live daemon's record must
  # survive List()'s self-heal pruning. Not implicit (crash-only) and not
  # merely specified by a literal expected value.

prevention: |
  Why not caught: no gate existed for this class. The only Linux execution of
  internal/daemon was CI, which runs -count=1 on a young binary, so the
  aging-dependent branch was sampled at exactly the elapsed lifetime where it
  passes. Locally the branch is unreachable (procstart_other.go), so the whole
  corroboration path had ZERO effective coverage on the developer machine and
  near-zero on CI. The failure was read as "flaky infra" for a full phase
  (recorded as "Phase-8 gap #5") rather than as a real single-writer-invariant
  defect, because the symptom (an empty slice) did not look like a lock bug.

  Guard: TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck and
  TestRunRegistersRecordThatSurvivesPruningOnAgedProcess (both force the aged-process
  condition via requireAgedProcess, so they are deterministic rather than
  timing-sampled), plus TestIsStaleStillDetectsPIDReuseOnAgedProcess as the
  standing WR-02 / T-07-04-01 counter-guard proving detection was narrowed and
  not disabled. Together these convert an elapsed-lifetime coin flip into a
  fixed assertion that fails 100% pre-fix.

  Residual risk accepted knowingly: CI runs linux/amd64 and verification ran only
  on linux/arm64; USER_HZ != 100 is still unvalidated.

open_followups:
  - id: convergence-watcher-arming-race
    summary: |
      TestConvergenceTwoSessions fails ~2/40 iterations at soak_test.go:232 (NOT :192).
      Mechanism: waitForLock only proves the lockfile exists, but Run() calls acquire()
      before watch.Open(); a file created in that window is never delivered by fsnotify.
      The test comment at soak_test.go:221-225 claims waitForLock closes this race — it
      does not. Suggested fix: wait on the existing onWatchOpen seam instead.
    status: NOT FIXED — separate defect, deliberately not bundled into this session.
    confidence: characterized by mechanism + signature separation (assertion, message,
      duration, distribution), NOT by direct pre-fix observation — pre-fix the test dies
      earlier at :192 and never reaches :232.
  - id: procstart-truncation
    summary: |
      procstart_linux.go derives start time via time.Unix(bootUnix+ticks/procStatClockTicks, 0),
      two independent downward truncations that read ~1.6-2.0s early, shrinking the
      EFFECTIVE corroboration window to ~3.05s against a nominal procStartTimeSlack of 5s.
    status: NOT CHANGED — intentionally out of scope. After the Cause B fix both sides of
      the self-comparison derive from processStartTime for the same pid, so the truncation
      cancels to a zero delta for self-owned records. It still applies to cross-process
      PID-reuse comparisons, where a ~2s narrowing of a 5s window is conservative (more
      likely to declare stale, never less). Revisit only if that conservatism bites.
