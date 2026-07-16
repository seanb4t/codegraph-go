---
phase: 03-watcher-on-mcp-default
plan: 01
subsystem: infra
tags: [watcher, wsl2, policy, fsnotify, tdd]

# Dependency graph
requires: []
provides:
  - "internal/watch/policy.go — WatchDisabledReason(projectRoot, Probe) string, the WATCH-03 decision function"
  - "watch.DetectWSL() bool — cached WSL2 detection"
  - "watch.ErrWatchDisabled sentinel error for downstream enforcement/messaging"
affects: [03-02, 03-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Explicit-input pure policy function (Probe struct), never mutates process env — D-05"
    - "sync.Once-cached probe with unexported test-reset hook (resetWSLCacheForTests), mirroring daemon.onSyncStart's test-only-seam convention"

key-files:
  created: [internal/watch/policy.go, internal/watch/policy_test.go]
  modified: []

key-decisions:
  - "Ported TS 1.3.1's watch-policy.js precedence and reason strings verbatim, with the documented fs.watch->file watching wording divergence (D-13)"
  - "--no-watch flag and CODEGRAPH_NO_WATCH=1 env produce the identical reason string (matches TS's env-routing behavior without mutating process env, per D-05/bin/codegraph.js finding)"
  - "Fixed filepath.ToSlash-vs-strings.ReplaceAll bug found during GREEN: ToSlash is a host-OS no-op for backslashes on Linux (the only GOOS DetectWSL ever returns true for), unlike TS's unconditional normalizePath regex-replace"

patterns-established:
  - "Verbatim TS reason strings with an explicitly commented, documented allowed divergence (D-02-style) for any Node-API-naming wording"

requirements-completed: [WATCH-03]

coverage:
  - id: D1
    description: "WatchDisabledReason implements the full D-04 precedence (NoWatch -> ForceWatch -> WSL2+/mnt -> default) with verbatim TS reason strings and strict ==\"1\" env parsing"
    requirement: "WATCH-03"
    verification:
      - kind: unit
        ref: "internal/watch/policy_test.go#TestWatchDisabledReason"
        status: pass
    human_judgment: false
  - id: D2
    description: "DetectWSL is cached (sync.Once) and returns false off-linux / on any /proc/version read failure, with a test-only reset hook"
    requirement: "WATCH-03"
    verification:
      - kind: unit
        ref: "internal/watch/policy_test.go#TestDetectWSL"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-07-16
status: complete
---

# Phase 3 Plan 1: Watch-Policy Port Summary

**Ported TS 1.3.1's `watch-policy.js` verbatim into `internal/watch/policy.go` — a pure, injectable, table-tested WSL2/env watch-decision function with `ErrWatchDisabled` exported for downstream enforcement.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-16T09:52:56-04:00
- **Completed:** 2026-07-16T09:56:11-04:00
- **Tasks:** 2 (RED + GREEN)
- **Files modified:** 2

## Accomplishments
- `WatchDisabledReason(projectRoot string, p Probe) string` implementing the exact TS precedence: NoWatch(flag|env) → off, ForceWatch(flag|env) → on, WSL2+`/mnt/[a-z]` → off, default → on
- `DetectWSL() bool`, cached via `sync.Once`, false off-linux and on any `/proc/version` read failure, with an unexported `resetWSLCacheForTests` reset hook
- `ErrWatchDisabled` sentinel exported from `internal/watch` so both `internal/daemon` (03-02) and `internal/cli` (03-03) can consume it without an import cycle
- Full table-driven RED→GREEN TDD cycle: 14 precedence/strict-env/WSL cases plus a caching-property test, all green

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — table-driven policy_test.go** - `92484fc` (test)
2. **Task 2: GREEN — implement internal/watch/policy.go** - `44f83c3` (feat)

**Plan metadata:** (this commit)

## TDD Gate Compliance

- RED gate: `92484fc` `test(03-01): add failing table-driven test...` — confirmed build failure (`undefined: Probe`) before any implementation existed.
- GREEN gate: `44f83c3` `feat(03-01): implement internal/watch/policy.go...` — full table green, `go vet` clean.
- No REFACTOR commit needed (implementation was correct-shaped on first pass except the one bug below, which was fixed within the GREEN commit before it landed).

## Files Created/Modified
- `internal/watch/policy.go` - `Probe`, `WatchDisabledReason`, `DetectWSL`, `isWindowsDriveMount`, `resetWSLCacheForTests`, `ErrWatchDisabled`
- `internal/watch/policy_test.go` - table-driven precedence test + `DetectWSL` caching test

## Decisions Made
- Ported TS 1.3.1's `watch-policy.js` verbatim precedence and reason strings; documented the one intentional wording divergence (`fs.watch` → `file watching`, D-13 — human-facing stderr only, never parsed, since naming a Node API in a Go binary is wrong).
- `--no-watch` flag and `CODEGRAPH_NO_WATCH=1` env are two inputs to the same tier-1 check, both producing the identical reason string `"CODEGRAPH_NO_WATCH=1 is set"` — matches TS's actual behavior (its own `--no-watch` flag routes through `process.env.CODEGRAPH_NO_WATCH='1'` before the check runs) without our port mutating process env (D-05).
- `ErrWatchDisabled` placed in `internal/watch` (not `internal/daemon`) so both downstream consumers (03-02 enforcement, 03-03 human message) import it without a cycle, per the plan's explicit `key_links`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `filepath.ToSlash` does not normalize backslashes on non-Windows hosts**
- **Found during:** Task 2 (GREEN) — the backslash-normalization test case (`\mnt\c\repo`) failed against the initial implementation
- **Issue:** The plan's `read_first`/PATTERNS.md guidance cited `filepath.ToSlash` as the established `normalizePath` equivalent elsewhere in this codebase. That equivalence does not hold here: `filepath.ToSlash` only replaces `filepath.Separator`, which is already `/` on Linux — the only `GOOS` `DetectWSL` ever reports `true` for — so it is a no-op for backslashes on the one platform this code path actually runs on. TS's `normalizePath` does an unconditional `replace(/\\/g, '/')` regardless of host OS.
- **Fix:** Replaced `filepath.ToSlash(projectRoot)` with `strings.ReplaceAll(projectRoot, `\`, "/")` in `isWindowsDriveMount`, matching TS's unconditional behavior. Removed the now-unused `path/filepath` import.
- **Files modified:** `internal/watch/policy.go`
- **Verification:** `TestWatchDisabledReason/WSL_+_backslash-normalized_project_root_still_matches_the_/mnt/_mount` passes; full `go test ./internal/watch/...` green.
- **Committed in:** `44f83c3` (Task 2/GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary correctness fix caught by the plan's own test table before any downstream code could depend on the wrong behavior. No scope creep.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `watch.WatchDisabledReason`, `watch.DetectWSL`, and `watch.ErrWatchDisabled` are ready for 03-02 (`internal/daemon.Run` enforcement) and 03-03 (`internal/cli/serve.go` human-visible message and `--no-watch`/`--watch` flag wiring).
- No blockers. WSL2 real-hardware validation remains a documented follow-up (RESEARCH.md Environment Availability) — the injectable-probe unit tests are sufficient for this plan's completion per the phase's own research recommendation.

---
*Phase: 03-watcher-on-mcp-default*
*Completed: 2026-07-16*
