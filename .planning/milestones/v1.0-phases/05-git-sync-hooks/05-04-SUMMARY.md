---
phase: 05-git-sync-hooks
plan: 04
subsystem: cli
tags: [go, git-hooks, cobra, cli]

# Dependency graph
requires:
  - phase: 05-git-sync-hooks
    provides: "05-03's internal/githooks.Install/Remove/Status (the core package this plan wraps)"
provides:
  - "codegraph githooks install/remove/status [path] — the user-facing CLI surface for HOOK-01/HOOK-02"
affects: [05-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "githooks command tree mirrors init/uninit/sync's single-subcommand shape (targetRoot + MaximumNArgs(1) + RunE closure)"
    - "install exits 0 (never errors) on both the friendly non-repo skip and the could-not-install degrade case"

key-files:
  created: [internal/cli/githooks.go, internal/cli/githooks_test.go]
  modified: [internal/cli/root.go]

key-decisions:
  - "install/remove/status RunE bodies write directly to cmd.OutOrStdout() and return nil on every degrade path (Skipped != \"\" or zero installed/removed) per D-11's exit-0 contract — only targetRoot's own path-resolution error propagates as a real error"
  - "No shared `plural` helper existed in internal/cli (uninit.go doesn't define one despite the plan's read_first note) — inlined a 2-line singular/plural branch in newGithooksRemoveCmd rather than introducing a new package-level helper for one call site"
  - "githooks_test.go uses local runGit/initGitRepo helpers (package cli, distinct from internal/gitmeta and internal/githooks's own local copies) per the plan's read_first note — no cross-package test-helper import introduced"

patterns-established:
  - "internal/cli/githooks.go is the sole CLI-layer consumer of internal/githooks — no other command file imports it"

requirements-completed: [HOOK-01, HOOK-02]

coverage:
  - id: D1
    description: "newGithooksCmd + 3 subcommands exist, each with Args cobra.MaximumNArgs(1) and targetRoot(args); root.go registers newGithooksCmd() appended after newMigrateCmd() with all prior entries retained; module builds and vets clean"
    requirement: "HOOK-01"
    verification:
      - kind: command
        ref: "go build ./... && go vet ./internal/cli/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "githooks install/status/remove drive the real cobra tree end-to-end against a real-git fixture: install writes mode-0755 marker-fenced hooks and stdout names them; status reports all 3 hooks installed plus a hooks-dir line; remove reports removal and strips the marker from disk; install against a non-git directory exits 0 with a friendly skip message"
    requirement: "HOOK-01, HOOK-02"
    verification:
      - kind: unit
        ref: "internal/cli/githooks_test.go#TestGithooksInstall_RealCobraTree"
        status: pass
      - kind: unit
        ref: "internal/cli/githooks_test.go#TestGithooksStatus_AfterInstall_ReportsAllThreeInstalled"
        status: pass
      - kind: unit
        ref: "internal/cli/githooks_test.go#TestGithooksRemove_AfterInstall_StripsMarkerFromAllHooks"
        status: pass
      - kind: unit
        ref: "internal/cli/githooks_test.go#TestGithooksInstall_NonGitDirectory_SkipsCleanlyWithExitZero"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-07-16
status: complete
---

# Phase 5 Plan 4: codegraph githooks CLI command tree Summary

**Added `codegraph githooks install|remove|status [path]` — a Go-only cobra command tree wrapping 05-03's internal/githooks core, registered in root.go, mirroring init/uninit/sync's targetRoot + MaximumNArgs(1) shape, with mutation-proof reachability tests driving the real command tree against real-git fixtures.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-16T21:36:00-04:00 (approx)
- **Completed:** 2026-07-16T21:37:52-04:00
- **Tasks:** 2
- **Files modified:** 3 (2 new, 1 modified)

## Accomplishments
- `newGithooksCmd()` — parent `githooks` cobra command with `install`/`remove`/`status` subcommands, each `[path]` resolved via the existing `targetRoot` helper
- `install` reports the installed hooks and a background-sync reminder on success, a friendly skip line when `result.Skipped != ""`, and a could-not-install degrade message otherwise — always exit 0
- `remove` reports removed hooks or a nothing-to-remove line, always exit 0 on the skip path
- `status` prints a `hooks dir: <path>` line then one `<hook>: installed|not installed` line per hook, exit 0 regardless of install state
- root.go's `AddCommand` list gained `newGithooksCmd()` appended after `newMigrateCmd()` — no reordering of the existing 22 entries
- `internal/cli/githooks_test.go` — 4 mutation-proof reachability tests driving `execCmd` (the real cobra tree) against real-git `t.TempDir()` fixtures

## Task Commits

Each task was committed atomically:

1. **Task 1: githooks command tree + output formatting + root registration** — `3eebfda` (feat)
2. **Task 2: Mutation-proof CLI reachability tests (real cobra tree, real-git fixtures)** — `0f8bfed` (test)

## Files Created/Modified
- `internal/cli/githooks.go` - `newGithooksCmd` + `newGithooksInstallCmd`/`newGithooksRemoveCmd`/`newGithooksStatusCmd`, each wired to `internal/githooks.Install`/`Remove`/`Status`
- `internal/cli/githooks_test.go` - Real-git fixture helpers (`runGit`/`initGitRepo`, local to package `cli`) + 4 reachability tests covering install/status/remove/non-repo-skip
- `internal/cli/root.go` - `AddCommand` list gains `newGithooksCmd()`, appended, all prior entries retained

## Decisions Made
- Followed D-11 literally: fixed hook trio, no hook-selection flags, `[path]` via `targetRoot`, `status` exits 0 unconditionally.
- Install/remove RunE bodies never return the `Skipped`/zero-count cases as errors — only a `targetRoot` path-resolution failure (e.g. inaccessible cwd) propagates as a real cobra error, matching the plan's acceptance criterion "install RunE returns nil when result.Skipped != \"\"".
- No shared `plural` helper existed in `internal/cli` — the plan's read_first note referenced one via `uninit.go`, but reading `uninit.go` showed no such helper defined there. Inlined a 2-line singular/plural branch in `newGithooksRemoveCmd` instead of introducing a new package-level helper for a single call site (deviation documented below).
- `githooks_test.go` uses locally-defined `runGit`/`initGitRepo` (package `cli`) rather than importing `internal/gitmeta`'s or `internal/githooks`'s test helpers — matches the established per-package fixture-helper convention from 05-03's `githooks_test.go` and `internal/gitmeta/fixtures_test.go`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `plural` helper referenced in read_first did not exist**
- **Found during:** Task 1, writing `newGithooksRemoveCmd`
- **Issue:** PATTERNS.md's D-06 uninit-cleanup snippet showed `plural(len(result.Removed))` as if a shared helper already existed in `internal/cli`. Reading `internal/cli/uninit.go` in full (the plan's own read_first target) confirmed no such function is defined anywhere in the package.
- **Fix:** Inlined a 2-line singular/plural branch (`suffix := "s"; if len == 1 { suffix = "" }`) directly in `newGithooksRemoveCmd` rather than adding a new unused-elsewhere package-level helper.
- **Files modified:** `internal/cli/githooks.go`
- **Commit:** `3eebfda`

Everything else — plan executed exactly as written, no other deviations.

## Issues Encountered
None beyond the `plural` helper correction above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `codegraph githooks install|remove|status` is a complete, tested, registered CLI surface — ready for 05-05 to wire `init`'s D-07 watch-fallback advisory and `uninit`'s D-06 best-effort cleanup on top of the same `internal/githooks` package this plan already imports.
- No new git-exec call sites or write primitives were introduced at the CLI layer — this plan is a thin wrapper, consistent with the single-seam-confinement pattern established in 05-01/05-02/05-03.
- No blockers for 05-05.

---
*Phase: 05-git-sync-hooks*
*Completed: 2026-07-16*

## Self-Check: PASSED

All created/modified files found on disk (internal/cli/githooks.go, internal/cli/githooks_test.go, internal/cli/root.go, 05-04-SUMMARY.md); both task commits (3eebfda, 0f8bfed) found in git log.
