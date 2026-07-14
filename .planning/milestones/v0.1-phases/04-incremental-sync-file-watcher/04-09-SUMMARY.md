---
phase: 04-incremental-sync-file-watcher
plan: 09
subsystem: testing
tags: [goleak, soak-test, goroutine-leak, watch, daemon, sync-06, go]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/watch.Watcher/Debouncer (Plan 04-05) — context+WaitGroup-compatible lifecycle primitives (Watcher.Close idempotent, Debouncer.Stop)"
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/daemon.Daemon.Run (Plan 04-07) — context+sync.WaitGroup shutdown discipline, onSync test observability seam, lockfile acquire/release"
provides:
  - "internal/watch/main_test.go: TestMain(goleak.VerifyTestMain) — package-wide leak guard for internal/watch"
  - "internal/watch/soak_test.go: TestSoak — 40 watch->debounce cycles proving the watcher/debounce subsystem is goroutine-leak-free"
  - "internal/daemon/soak_test.go: TestMain(goleak.VerifyTestMain) + TestSoak — 15 edit->sync cycles proving the daemon's full watch->debounce->sync lifecycle is goroutine-leak-free and lock-clean on shutdown"
  - "go.uber.org/goleak v1.3.0 as a direct test-only dependency"
affects: []

# Tech tracking
tech-stack:
  added: ["go.uber.org/goleak v1.3.0 (test-only)"]
  patterns:
    - "TestMain(m) calling goleak.VerifyTestMain(m) for package-wide leak detection (avoids per-test defer goleak.VerifyNone(t) false positives from in-flight timer callbacks)"
    - "cancel(); wg.Wait() BEFORE any goleak assertion — every soak test joins its spawned goroutine(s) prior to returning, so TestMain's end-of-package VerifyTestMain sees zero in-flight goroutines"
    - "onSync observability seam (internal/daemon, pre-existing from Plan 04-07) reused to synchronize the soak loop on real sync completion instead of sleep-and-hope timing"

key-files:
  created:
    - internal/watch/main_test.go
    - internal/watch/soak_test.go
    - internal/daemon/soak_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "go.uber.org/goleak promoted to go.mod's direct require block by manual edit (not go mod tidy), per the established project convention (STATE.md Phase 1 decisions) — go get -t added it as indirect first; the require line was moved manually to preserve alphabetical placement without disturbing other pre-pinned deps"
  - "internal/daemon/soak_test.go carries the package's TestMain (internal/daemon had none before this plan) rather than adding a separate main_test.go — kept in soak_test.go since it exists solely to gate the soak test's leak assertion"
  - "Watch-package soak asserts flush-fired (not a full indexer.Sync) — Task 1's action text scopes the watch-package soak to the watch->debounce path; the daemon-package soak (Task 2) is where the real watch->debounce->sync path is exercised end-to-end, matching the plan's tiered soak design"

patterns-established:
  - "Pattern: goleak.VerifyTestMain lives in the package whose lifecycle-owning type (Watcher/Daemon) is under test, gating the whole test package rather than one test — every existing test in internal/watch and internal/daemon already cancels+joins before returning, so no other test needed rework to pass under the new goleak gate"

requirements-completed: [SYNC-06]

coverage:
  - id: D1
    description: "Running many watch->debounce cycles leaves zero leaked goroutines (goleak clean) in internal/watch"
    requirement: "SYNC-06"
    verification:
      - kind: unit
        ref: "internal/watch/soak_test.go#TestSoak"
        status: pass
    human_judgment: false
  - id: D2
    description: "The daemon's full lifecycle (start, edit bursts, cancel, join) is goroutine-leak-free"
    requirement: "SYNC-06"
    verification:
      - kind: unit
        ref: "internal/daemon/soak_test.go#TestSoak"
        status: pass
    human_judgment: false
  - id: D3
    description: "The soak suite runs well within the 120s CI timeout, including under -race for the daemon soak"
    requirement: "SYNC-06"
    verification:
      - kind: unit
        ref: "go test ./internal/watch/... ./internal/daemon/... -race -run 'TestSoak' -timeout 120s -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 9: Goroutine-Leak-Free Watcher/Daemon Soak (goleak) Summary

**A goleak-gated soak suite (`go.uber.org/goleak` v1.3.0, test-only) proving 40 watch->debounce cycles in `internal/watch` and 15 full edit->sync cycles through `internal/daemon.Daemon.Run` leave zero leaked goroutines — the SYNC-06 acceptance gate turning the Plan 04-05/04-07 context+WaitGroup lifecycle discipline into a repeatable, CI-runnable assertion.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-11T21:12:25Z
- **Completed:** 2026-07-11T21:16:47Z
- **Tasks:** 2
- **Files modified:** 5 (3 created, `go.mod`/`go.sum` updated)

## Accomplishments
- `internal/watch/main_test.go`'s `TestMain` gates the whole package on `goleak.VerifyTestMain(m)` — no other test in the package needed rework, since every existing test already cancels its context and joins its goroutine(s) before returning
- `internal/watch/soak_test.go#TestSoak` drives 40 file-write cycles through a live `Watcher.Run`+`Debouncer`, sleeping `DebounceDuration()+30ms` between each so every cycle produces its own flush, then `cancel(); wg.Wait()` before asserting at least one flush fired — proving the flush path is exercised, not a no-op
- `internal/daemon/soak_test.go` adds the package's first `TestMain` (also `goleak.VerifyTestMain`) and `TestSoak`, which drives `Daemon.Run(ctx)` through 15 real edit->sync cycles via the pre-existing `onSync` observability seam (no sleep-and-hope timing), then asserts the final edit's symbol landed in the committed graph and that a subsequent `acquire` succeeds after shutdown (proving the lockfile was genuinely released, not merely absent)
- Both soak tests join (`wg.Wait()`) strictly before returning — the ordering `goleak.VerifyTestMain` requires (RESEARCH Pattern 7): a still-shutting-down goroutine at assertion time is a false-positive leak, not a real one
- `go.uber.org/goleak v1.3.0` added as a direct test-only dependency by manual `go.mod` edit (no broad `go mod tidy`), matching the project's established Phase 1 convention

## Task Commits

Each task was committed atomically:

1. **Task 1: goleak dependency + watch soak with VerifyTestMain** - `54ea614` (test)
2. **Task 2: daemon full-lifecycle soak** - `a4437e2` (test)

**Plan metadata:** (this commit) - `docs(04-09): complete plan`

## Files Created/Modified
- `internal/watch/main_test.go` - `TestMain` calling `goleak.VerifyTestMain(m)`
- `internal/watch/soak_test.go` - `TestSoak`: 40 watch->debounce cycles, asserts flush fired, leak-free
- `internal/daemon/soak_test.go` - `TestMain` (package's first) + `TestSoak`: 15 edit->sync cycles through `Daemon.Run`, asserts final graph state and post-shutdown lock release
- `go.mod` / `go.sum` - `go.uber.org/goleak v1.3.0` added as a direct (test-only) dependency

## Decisions Made
- `go.uber.org/goleak` was promoted from `go get -t`'s initial `// indirect` placement to the direct require block by manual edit, inserted alphabetically between `tree-sitter-python` and `golang.org/x/mod` (`go.uber.org` sorts before `golang.org` lexically) — kept consistent with the project's manual-promotion convention rather than running a broad `go mod tidy`.
- `internal/daemon`'s `TestMain` lives in `soak_test.go` rather than a separate `main_test.go` (unlike `internal/watch`, which already had a natural place for one) — it exists solely to gate this plan's soak assertion and there was no prior `internal/daemon` test-package-wide file to extend.
- The `internal/watch` soak asserts "flush fired" rather than invoking a real `indexer.Sync` — the plan's Task 1 action text scopes this soak to the watch->debounce path specifically; Task 2's daemon soak is the one that exercises the full watch->debounce->sync path end-to-end (a deliberate two-tier soak design, not a shortcut).

## Deviations from Plan

None - plan executed exactly as written. No real goroutine leaks were found in `internal/watch` or `internal/daemon` — the Plan 04-05/04-07 context+WaitGroup lifecycle discipline held under both soaks on the first run, with no `IgnoreTopFunction`/`IgnoreAnyFunction` suppressions needed.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. `go.uber.org/goleak` is a test-only dependency; it does not ship in the release binary.

## Next Phase Readiness
- SYNC-06 is proven: `go test ./internal/watch/... ./internal/daemon/... -run TestSoak -race -timeout 120s -count=1` passes with zero leaked goroutines.
- `go test ./... -race -count=1` is green across the whole repo (all packages, not just the two touched by this plan).
- This is the last plan (04-09 of 9) in Phase 4 — Incremental Sync & File Watcher's success criteria (INDX-03, INDX-04, SYNC-01…06) are now all implemented and verified; ready for phase transition.
- No blockers.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*
