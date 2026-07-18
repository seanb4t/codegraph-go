---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 04
subsystem: daemon
tags: [go, signals, sigterm, process-lifecycle, daemon-registry]

# Dependency graph
requires:
  - phase: 07-02
    provides: "Global daemon registry (Record, List/Register/Deregister, isStale re-check plumbing)"
provides:
  - "sendStop(pid) platform split: POSIX real SIGTERM, Windows documented hard-kill (TerminateProcess)"
  - "StopMatching(repoRoot) and StopAll() orchestration over registry.List() with per-target isStale re-corroboration"
  - "Injectable stopSignal test seam for safe unit testing of stop orchestration"
affects: [07-07 (daemon stop / stop --all CLI command), 07-07 (picker's stop action)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sendStop platform split (stop_posix.go !windows / stop_windows.go windows) mirroring lock.go's isProcessLive os.FindProcess+Signal call shape"
    - "Package-level injectable func var (stopSignal = sendStop) as a test seam, same convention as registryDir/onSync*/onWatchOpen"
    - "Re-corroborate via isStale immediately before signaling — defense-in-depth against the List()-scan-to-signal TOCTOU window"

key-files:
  created:
    - internal/daemon/stop.go
    - internal/daemon/stop_posix.go
    - internal/daemon/stop_windows.go
    - internal/daemon/stop_test.go
  modified: []

key-decisions:
  - "Windows sendStop uses stdlib os.Process.Kill only (no golang.org/x/sys import) — TerminateProcess via the Go runtime, matching the plan's preference for minimal Windows surface"
  - "stopTargets re-derives isStale on every candidate from List() even though List() itself already self-heals — deliberate belt-and-suspenders re-corroboration per the plan's stated safety invariant (T-07-04-01), not redundant given the TOCTOU window between the two calls"
  - "RepoRoot matching falls back to plain string comparison when filepath.EvalSymlinks errors (e.g. path no longer exists), so a StopMatching call against a torn-down repo path still resolves deterministically instead of silently matching nothing"
  - "stop_test.go is POSIX-only (//go:build !windows), covering both Task 1 (sendStop) and Task 2 (StopAll/StopMatching) — matches the plan's file list and verify commands, which only require compile-clean (go vet) on Windows, not test execution"

patterns-established:
  - "Live-process test double via self-exec (GO_WANT_HELPER_PROCESS + TestHelperProcess blocking on select{}), extending the existing deadPID() self-exec idiom in lock_test.go to the 'need a real live pid' case"

requirements-completed: [DMON-02]

coverage:
  - id: D1
    description: "sendStop delivers a real SIGTERM on POSIX (terminates a live process) and errors on a dead pid"
    requirement: DMON-02
    verification:
      - kind: unit
        ref: "internal/daemon/stop_test.go#TestSendStop"
        status: pass
    human_judgment: false
  - id: D2
    description: "stop_windows.go sendStop hard-kill divergence compiles under GOOS=windows and documents the divergence"
    requirement: DMON-02
    verification:
      - kind: unit
        ref: "go build ./internal/daemon/ (native); GOOS=windows go build succeeds for stop_windows.go itself — see Deviations for the pre-existing unrelated tree_sitter vet noise"
        status: pass
    human_judgment: false
  - id: D3
    description: "StopAll/StopMatching signal only corroborated-live records, skip stale ones, de-duplicate by pid, and no-op cleanly when empty/no-match"
    requirement: DMON-02
    verification:
      - kind: unit
        ref: "internal/daemon/stop_test.go#TestStopAll"
        status: pass
      - kind: unit
        ref: "internal/daemon/stop_test.go#TestStopMatching"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-18
status: complete
---

# Phase 07 Plan 04: Daemon Stop Signaling Summary

**Graceful POSIX SIGTERM / documented Windows hard-kill sendStop, plus corroboration-guarded StopMatching/StopAll orchestration over the 07-02 registry.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-18T19:43:00Z
- **Completed:** 2026-07-18T19:47:38Z
- **Tasks:** 2
- **Files modified:** 4 (all new)

## Accomplishments
- `sendStop(pid)` platform split: POSIX delivers a real `SIGTERM` via the exact `os.FindProcess`+`Signal` call shape `lock.go`'s `isProcessLive` already uses; Windows performs a documented hard-kill (`os.Process.Kill`/`TerminateProcess`) since Windows has no cross-process `SIGTERM` delivery (RESEARCH Open Q#1, Assumption A4).
- `StopAll()` and `StopMatching(repoRoot)` resolve targets from the self-healing registry (`List()`, 07-02) and re-corroborate each candidate via `isStale` immediately before signaling — a stale/forged/reused-pid record is never passed to `sendStop`.
- Targets de-duplicated by pid; per-target `sendStop` errors aggregated via `errors.Join` (never swallowed); empty registry / no matching repoRoot are clean `(nil, nil)` no-ops.
- `internal/daemon` stays charm-free (`rg charm.land internal/daemon` — no hits) and the whole package's `-race` tests (including the pre-existing `goleak`-gated `TestMain`) pass green.

## Task Commits

Each task was committed atomically as a TDD RED/GREEN pair:

1. **Task 1: sendStop platform split** — RED `836cd3b` (test), GREEN `4deab40` (feat)
2. **Task 2: StopMatching / StopAll orchestration** — RED `4a6affc` (test), GREEN `dc2e91e` (feat)

**Plan metadata:** _pending (this commit)_

## Files Created/Modified
- `internal/daemon/stop_posix.go` - `//go:build !windows` — `sendStop` via real `SIGTERM`
- `internal/daemon/stop_windows.go` - `//go:build windows` — `sendStop` hard-kill divergence (stdlib `os.Process.Kill` only, no x/sys)
- `internal/daemon/stop.go` - `StopMatching`/`StopAll` orchestration, injectable `stopSignal` seam, `resolveRepoRoot` symlink-normalized comparison
- `internal/daemon/stop_test.go` - `TestSendStop`, `TestStopAll`, `TestStopMatching` (POSIX-only, `//go:build !windows`)

## Decisions Made
- Windows `sendStop` uses only stdlib `os.Process.Kill` — no `golang.org/x/sys` import needed, matching the plan's preference and `locked_windows.go`'s minimal-direct-wrapper style.
- `stopTargets` re-derives `isStale` on every `List()` candidate even though `List()` itself already self-heals on the same call — this is deliberate defense-in-depth against the (narrow) TOCTOU window between `List()`'s scan and the actual signal attempt, per the plan's explicit safety-invariant language, not redundant dead code.
- `resolveRepoRoot` falls back to plain string comparison when `filepath.EvalSymlinks` errors (nonexistent path), so `StopMatching` against a repo whose directory has since been removed still resolves deterministically.

## Deviations from Plan

None — plan executed exactly as written. One clarifying note: `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` (the plan's stated Task 1 verify command) surfaces pre-existing `tree_sitter.*` undefined-symbol noise from `internal/daemon`'s transitive import of `internal/indexer` → CGo tree-sitter route extractors — confirmed present in the codebase before this plan's changes (unrelated to `stop_posix.go`/`stop_windows.go`), and explicitly flagged as a known artifact in the executor's operating instructions ("trust `go build ./...` + `go test`, not editor/vet cross-compile diagnostics"). Verified `stop_windows.go` itself is sound by isolating a `GOOS=windows go build ./internal/daemon/` run — the failure surface is entirely within `internal/indexer/routes` and `internal/indexer/goextract`, never `stop_windows.go`.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/daemon.StopMatching`/`StopAll` are ready for 07-07's `daemon stop`/`daemon stop --all` CLI command and the picker's stop action to call directly.
- The daemon-stop safety invariant (never signal an uncorroborated pid) is unit-tested and holds; no further work needed in this package for DMON-02's signaling half.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED
All created files and referenced commit hashes verified present on disk / in git log.
