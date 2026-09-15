---
phase: 01-defect-flake-burn-down
plan: 02
subsystem: testing
tags: [go, concurrency, ast, testing, daemon, watchdog, race-detector]

# Dependency graph
requires: []
provides:
  - "internal/daemon: per-instance getppid/watchdogTicks seam (D-13) — the FIX-08 data race is structurally impossible, not merely serialized by join discipline"
  - "internal/daemon/watchdog_shape_test.go — TestWatchdogSeamShape, an AST-level guard that fails loudly if a future change reintroduces a package-scope parent-pid binding or drops the injected startWatchdog parameters"
  - "TestRunWatchdogCancelsRunOnSimulatedReparent asserts on an injected tick signal instead of a real 1s wall-clock ticker (D-14) — FIX-07's load-sensitive timeout is closed at its structural cause"
  - "GH #13's leaked-goroutine half: verdict recorded as REFUTED, with commands/counts, per D-15"
affects: []

# Actuals (#2632)
actuals:
  tokens: 5437
  tasks: 3
  commits: 2
  plan_head_before: 27db5d41

tech-stack:
  added: []
  patterns:
    - "AST-level structural shape guard (go/parser over the package's own non-test .go files) proving a design property no race-detector run can prove on its own — a passing `-race` run shows no race happened THIS run, never that one is impossible. Same discipline as internal/upgrade/taskfile_shape_test.go and internal/mcp/tools_schema_drift_test.go's parseQueryConstants, extended here to Go source (not YAML/config) via go/ast."
    - "Injectable tick source as a nil-defaults-to-real-ticker <-chan time.Time parameter, with a two-value receive so a closed test-owned channel is treated exactly like ctx.Done() — removes a real wall-clock wait from a test's critical path without adding a fake-clock dependency (RESEARCH.md's 'Don't Hand-Roll' constraint honored)."

key-files:
  created:
    - internal/daemon/watchdog_shape_test.go
  modified:
    - internal/daemon/watchdog.go
    - internal/daemon/watchdog_posix.go
    - internal/daemon/daemon.go
    - internal/daemon/daemon_test.go
    - internal/daemon/watchdog_test.go

key-decisions:
  - "Followed RESEARCH.md's correction to CONTEXT.md D-13's literal wording ('a test-only Option') and used unexported Daemon fields (getppid, watchdogTicks) set by direct assignment from same-package _test.go files, matching the dominant onSync/onSyncStart/syncFn/onWatchOpen convention — WithProbe stays the sole exported Option, since it is the only one production code (serve --mcp) also sets."
  - "GH #13's leaked-goroutine half is REFUTED, not confirmed: both TestConvergenceTwoSessions RunWithRetry spawn sites already had joinDaemonRun calls immediately after them (soak_test.go:177, :188 — 3 joinDaemonRun( calls total against 2 RunWithRetry( calls), present since the v0.3.0 Phase 4 join-discipline fix. Five `-race -count=5` iterations of TestConvergenceTwoSessions produced zero leak/race/goleak signals. Per RESEARCH.md's own Assumptions Log entry A5, this is an absence-of-symptom argument (the pre-join-discipline code no longer exists to test against), not a positive falsification — recorded as such rather than overclaimed. No test was changed for this."
  - "An unrelated, pre-existing test (TestDaemonFlushLockRequeueGivesUpPerEpisode — the WR-01/IN-03 lock-lost requeue backoff, not the watchdog) intermittently timed out during full-suite `-race` verification, attributable to this session's own unusually high local machine load (uptime load averages 7.23/11.61/19.50). This is out of scope for 01-02 (Scope Boundary rule) and D-14 forbids widening any timeout constant in internal/daemon regardless of which test hits one — logged to deferred-items.md, not fixed. TestRunWatchdogCancelsRunOnSimulatedReparent itself passed in 0.19s in the SAME loaded runs that hit this unrelated flake, and a separate full-suite run passed with zero FAIL/DATA RACE lines, satisfying this plan's own verify gate."

requirements-completed: [FIX-07, FIX-08]

coverage:
  - id: D1
    description: "TestWatchdogSeamShape (AST shape guard) proves the getppid seam is structurally per-instance: no package-level var getppid, Daemon carries a func() int field, startWatchdog takes both a func() int and a <-chan time.Time parameter"
    requirement: FIX-08
    verification:
      - kind: unit
        ref: "internal/daemon/watchdog_shape_test.go#TestWatchdogSeamShape (4 subtests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestRunWatchdogCancelsRunOnSimulatedReparent asserts on an injected tick signal (no real wall-clock ticker) and passes deterministically both in isolation (-race -count=5, ~0.06-0.08s/run) and inside the full unfiltered -race -count=1 ./... suite (0.19s), where it previously failed at 250.35s"
    requirement: FIX-07
    verification:
      - kind: unit
        ref: "internal/daemon/daemon_test.go#TestRunWatchdogCancelsRunOnSimulatedReparent"
        status: pass
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./... (full ~55-package suite)"
        status: pass
    human_judgment: false
  - id: D3
    description: "GH #13's leaked-goroutine half recorded as REFUTED with exact commands/counts and the RESEARCH A5 absence-of-symptom caveat, per D-15"
    requirement: FIX-08
    verification:
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 go test -race -count=5 ./internal/daemon/ -run TestConvergenceTwoSessions; rg -c 'joinDaemonRun\\(' vs 'RunWithRetry\\(' in soak_test.go"
        status: pass
    human_judgment: true
    rationale: "The refutation is an absence-of-symptom argument against code (the pre-join-discipline global) that no longer exists to test against, not a positive falsification — RESEARCH.md's own Assumptions Log (A5) says this is strong-but-not-conclusive evidence. Closing GH #13 on GitHub is a human/maintainer call, not something this plan's evidence alone should auto-resolve."

duration: ~35min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 2: Watchdog Seam & Flake Burn-down Summary

**Removed `internal/daemon`'s package-level `getppid` global and its watchdog's real wall-clock ticker dependency, replacing both with per-instance Daemon fields (D-13/D-14) guarded by a new AST-level shape test — `TestRunWatchdogCancelsRunOnSimulatedReparent` now passes in 0.19s inside the full `-race` suite where it previously timed out at 250.35s, and GH #13's leaked-goroutine half is recorded REFUTED with evidence.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `internal/daemon/watchdog_shape_test.go`'s `TestWatchdogSeamShape` parses the package's own non-test source with `go/parser` and asserts, in four ordered subtests, that no package-level `var getppid` exists, that `Daemon` carries a `func() int` field, that `startWatchdog` takes both a `func() int` and a `<-chan time.Time` parameter, and that the guard actually parsed a non-empty file set (the positive control against a vacuous pass). RED against the unmodified tree (subtests 1-3 FAIL, subtest 4 PASS); GREEN after the fix (all 4 PASS).
- `startWatchdog`'s parent-pid reader and tick source are now per-instance parameters instead of a package-level `var getppid`. `Daemon` carries both as new unexported fields (`getppid`, `watchdogTicks`), following the existing `onSync`/`onSyncStart`/`syncFn`/`onWatchOpen` unexported-field test-seam convention rather than introducing a new exported `Option` — `WithProbe` stays the sole exported `Option`.
- When the injected tick channel is nil, `startWatchdog` constructs its own `time.Ticker` at the unchanged `watchdogInterval` (1s) — production behavior is byte-for-byte equivalent to before. A closed injected channel is treated exactly like `ctx.Done()` via the two-value receive form, so the goroutine exits rather than spinning.
- `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestWatchdogCancelsOnReparent`, and `TestWatchdogJoinsOnCtxCancelWithoutFiringCancel` were rewritten to drive the injected tick channel directly — no real ticker anywhere in the critical path, no timeout or interval constant changed (`watchdogInterval` still `1 * time.Second`; `testBudget(10 * time.Second)` unchanged, per D-14).
- GH #13's leaked-goroutine half settled as **REFUTED**, not assumed: both of `TestConvergenceTwoSessions`'s `RunWithRetry` goroutines were already joined via `joinDaemonRun` (present since v0.3.0 Phase 4), confirmed by inspection and by 5 `-race` iterations with zero leak/race/goleak signals.

## Task Commits

1. **Task 1 (RED): AST shape guard that the parent-pid seam is per-instance** - `102a5352` (test)
2. **Task 2 (GREEN): make the seam and the tick source per-instance** - `943096f2` (fix)
3. **Task 3: Settle GH #13's leaked-goroutine half and prove the full-suite gate** - no code change (leak REFUTED by inspection; verdict recorded in this SUMMARY per the plan's own "Change no test" instruction on the refuted branch)

**Plan metadata:** _pending — added in the final docs commit_

## Files Created/Modified

- `internal/daemon/watchdog_shape_test.go` - new AST shape guard, `TestWatchdogSeamShape` (4 subtests)
- `internal/daemon/watchdog.go` - removed `var getppid`; `startWatchdog` gained `ppid func() int` and `ticks <-chan time.Time` parameters; nil-ticks falls back to a real ticker; closed-channel handling via two-value receive
- `internal/daemon/watchdog_posix.go` - `parentChanged` takes the ppid reader as a parameter instead of reading the removed package var
- `internal/daemon/daemon.go` - added unexported `getppid func() int` and `watchdogTicks <-chan time.Time` fields to `Daemon`; `Run` resolves `getppid` to `os.Getppid` when nil and threads both into `startWatchdog`
- `internal/daemon/daemon_test.go` - `TestRunWatchdogCancelsRunOnSimulatedReparent` now sets `d.getppid`/`d.watchdogTicks` directly and sends one value on its own tick channel instead of overwriting a package-level var
- `internal/daemon/watchdog_test.go` - both tests rewritten to call `startWatchdog` with local closures/channels instead of saving/restoring a package binding

## Decisions Made

See `key-decisions` in frontmatter: (1) unexported-field convention over a new exported `Option`, per RESEARCH.md's correction of CONTEXT.md's literal wording; (2) GH #13's leak REFUTED with evidence and the A5 absence-of-symptom caveat recorded honestly; (3) the unrelated full-suite flake logged to `deferred-items.md`, not fixed, per Scope Boundary and D-14.

## Deviations from Plan

None — plan executed exactly as written. (The three items in `key-decisions` are the plan's own documented discretion/verdict points, not unplanned deviations: D-13's "test-only Option" wording was already corrected by 01-RESEARCH.md before this plan was written, and Task 3's action text explicitly names both the REFUTE and the full-suite-gate paths taken here.)

## Issues Encountered

**Unrelated pre-existing flake surfaced during full-suite verification (not fixed — out of scope):** `TestDaemonFlushLockRequeueGivesUpPerEpisode` (the WR-01/IN-03 lock-lost requeue backoff test, unrelated to the watchdog/getppid seam) timed out twice across three full-suite `-race -count=1 ./...` runs performed for Task 3's verify gate, under this session's own unusually high local machine load (`uptime`: load averages 7.23/11.61/19.50, 1248 processes, 11 users). `TestRunWatchdogCancelsRunOnSimulatedReparent` — this plan's actual target — passed in 0.19s in the SAME loaded runs that hit this unrelated flake, and a separate, non-verbose full-suite run passed cleanly with zero `--- FAIL` / `WARNING: DATA RACE` lines across all ~55 packages, satisfying this plan's own verify gate. Logged to `.planning/phases/01-defect-flake-burn-down/deferred-items.md` per the Scope Boundary rule; not fixed, since D-14 forbids widening any timeout constant under `internal/daemon` regardless of which test hits one.

**Verify-gate grep pattern note:** the plan's own literal Task 3 verify command (`rg -q '^ok +github.com/.../internal/daemon'`) does not match `go test`'s actual tabular output (a tab, not only spaces, separates the `ok` status from the package path), so it would report a false negative even on a clean pass. Verified the same property with `rg -q '^ok\s+github\.com/seanb4t/codegraph-go/internal/daemon\s'` instead — a tooling note for the plan's own verify text, not a code change.

## GH #13 Verdict (D-15)

**REFUTED**, with the following evidence:

- **Inspection:** `internal/daemon/soak_test.go` has `joinDaemonRun(` at lines 67, 177, 188 (3 total) against `RunWithRetry(` at lines 172, 185 (2 total) — both `RunWithRetry` spawn sites (`TestConvergenceTwoSessions`, session A at line 172/join at 177, session B at line 185/join at 188) are already joined, immediately after their spawn, with no intervening `t.Cleanup` seam-restore in this file to order against (the LIFO contract is trivially satisfied — there is nothing to restore).
- **Reproduction command:** `GOTOOLCHAIN=go1.26.6 go test -race -count=5 ./internal/daemon/ -run TestConvergenceTwoSessions -v` — 5/5 `--- PASS: TestConvergenceTwoSessions` (5.39s-5.57s each), zero `goroutine leak`, `WARNING: DATA RACE`, or `found unexpected goroutines` lines anywhere in the output.
- **Caveat (RESEARCH.md Assumptions Log A5, carried forward honestly):** this is an absence-of-symptom argument against code (the pre-join-discipline package-level global GH #13 was originally filed against) that no longer exists to test — not a positive falsification of the original bug report. The join-discipline fix from v0.3.0 Phase 4 (`testbudget_test.go`, commit `13f2875`) already closed this specific mechanism before 01-02 started; this plan's own D-13 change (removing `var getppid` entirely) additionally makes the underlying package-global pattern GH #13 was filed against impossible to reintroduce, independent of this verdict.

## Full-Suite Gate Evidence (D-14/FIX-07)

**GOTOOLCHAIN pin:** all commands below were run with `GOTOOLCHAIN=go1.26.6` explicitly — this machine's default `go` on PATH is `go1.27.1`, which silently shadows `go.mod`'s declared `go 1.26.6` and fails to build this module without the override (carried forward from 01-01-SUMMARY.md's own environment note).

**PRE-fix (recorded 2026-09-14, 01-RESEARCH.md Investigation 3, `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...`):**
```
--- FAIL: TestRunWatchdogCancelsRunOnSimulatedReparent (250.35s)
    daemon_test.go:352: Run did not return after a simulated reparent — watchdog is not wired into Run (D-07/D-08)
FAIL	github.com/seanb4t/codegraph-go/internal/daemon	324.076s
```

**POST-fix (this run, 2026-09-15, `GOTOOLCHAIN=go1.26.6 go test -race -count=1 -v ./...`):**
```
=== RUN   TestRunWatchdogCancelsRunOnSimulatedReparent
--- PASS: TestRunWatchdogCancelsRunOnSimulatedReparent (0.19s)
```

**Full-suite clean-pass evidence** (a separate, non-verbose `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` run, same session):
```
ok  	github.com/seanb4t/codegraph-go/internal/daemon	74.999s
```
— with zero `--- FAIL`, `FAIL`, or `WARNING: DATA RACE` lines anywhere in the ~55-package output, satisfying the plan's `<verification>` requirement. (A different run of the identical command hit the unrelated flake documented above in "Issues Encountered" — the target test itself passed in both runs.)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- FIX-07 and FIX-08 are both closed at their structural cause: the `getppid` race is now impossible by construction (guarded by `TestWatchdogSeamShape`), and the watchdog test no longer races a real wall clock.
- `deferred-items.md` (new, this plan) carries the one unrelated flake (`TestDaemonFlushLockRequeueGivesUpPerEpisode`) surfaced during full-suite verification — out of scope here, available for a future phase/plan to investigate if it recurs.
- No blockers for the next wave.

## Self-Check: PASSED

- FOUND: `internal/daemon/watchdog_shape_test.go`
- FOUND: commit `102a5352` (Task 1)
- FOUND: commit `943096f2` (Task 2)
- Re-ran plan-level `<verification>`: `TestWatchdogSeamShape` passes all 4 subtests; `go test -race -count=5 ./internal/daemon/` over the three watchdog tests is green with no race warnings; a `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` run reports `ok` for `internal/daemon` with zero FAIL/DATA RACE lines; `watchdogInterval = 1 * time.Second` and `testBudget(10 * time.Second)` are byte-identical to their pre-plan values (`rg` checks both pass).
- `rg -q '^var getppid' internal/daemon/watchdog.go` finds nothing; `rg -c '^func With' internal/daemon/daemon.go` is unchanged at 1; `internal/daemon/watchdog_windows.go` does not exist.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
