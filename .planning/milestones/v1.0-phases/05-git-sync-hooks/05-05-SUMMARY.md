---
phase: 05-git-sync-hooks
plan: 05
subsystem: cli
tags: [git-hooks, watch-policy, init, uninit, cobra]

requires:
  - phase: 05-git-sync-hooks
    provides: "internal/githooks (Install/Remove/Status) and gitmeta.IsGitRepo/HooksDir from 05-02/05-03; the codegraph githooks install/remove/status CLI tree from 05-04"
  - phase: 03-watcher-on-mcp-default
    provides: "watch.WatchDisabledReason + injectable watch.Probe (the D-07 gate)"
provides:
  - "init's success path surfaces a non-interactive plain-text D-07 advisory (port of TS offerWatchFallback) that is silent when the watcher runs normally and points at codegraph githooks install / codegraph sync / an already-installed notice when it is disabled"
  - "uninit --force best-effort strips codegraph's marker blocks from the three git hooks after removing .codegraph/ (D-06), never failing uninit on cleanup"
  - "Mutation-proof reachability tests for both wirings, manually confirmed red on revert"
affects: [phase-07-tui-interactive-hook-select, phase-08-surf-05-divergence-table]

tech-stack:
  added: []
  patterns:
    - "Best-effort, never-block advisory: printWatchFallbackAdvisory never returns an error and is called strictly after the primary command's own error paths"
    - "Test seam via injectable watch.Probe's nil-Env-defaults-to-os.Getenv fallback (t.Setenv(\"CODEGRAPH_NO_WATCH\", \"1\")) rather than a bespoke CLI flag"

key-files:
  created:
    - internal/cli/init_advisory_test.go
    - internal/cli/uninit_test.go
  modified:
    - internal/cli/init.go
    - internal/cli/uninit.go

key-decisions:
  - "watch.Probe{} (the bare zero value) is the correct D-07 gate call, not a hardcoded-unreachable literal — WatchDisabledReason defaults a nil Probe.Env to os.Getenv internally, so CODEGRAPH_NO_WATCH=1 already forces the advisory deterministically from a test without any new CLI flag or plumbing"
  - "Reused the package cli test helpers already added by 05-04 (runGit, initGitRepo, markerBeginBytes) rather than duplicating them a third time"
  - "Manually confirmed both wirings are load-bearing: reverting init.go's advisory call turns 3 tests red; reverting uninit.go's cleanup call fails the build outright (unused githooks import) — a stronger signal than a red test"

patterns-established:
  - "D-07/D-06 call sites live at the tail of their RunE bodies, after the primary operation's own success path, so any advisory/cleanup failure can never mask or block the primary result"

requirements-completed: [HOOK-03]

coverage:
  - id: D1
    description: "init's success path prints nothing when the watcher is enabled (not-always-on guarantee)"
    requirement: HOOK-03
    verification:
      - kind: unit
        ref: "internal/cli/init_advisory_test.go#TestInitAdvisory_WatcherEnabled"
        status: pass
    human_judgment: false
  - id: D2
    description: "init's success path warns + points at codegraph githooks install when the watcher is disabled, the target is a git repo, and no hooks are installed"
    requirement: HOOK-03
    verification:
      - kind: unit
        ref: "internal/cli/init_advisory_test.go#TestInitAdvisory_WatcherDisabled"
        status: pass
    human_judgment: false
  - id: D3
    description: "init's success path points at codegraph sync when the watcher is disabled and the target is not a git repo"
    requirement: HOOK-03
    verification:
      - kind: unit
        ref: "internal/cli/init_advisory_test.go#TestInitAdvisory_WatcherDisabled_NonGitRepo"
        status: pass
    human_judgment: false
  - id: D4
    description: "init's success path reports hooks already installed (no re-pointer) when the watcher is disabled, the target is a git repo, and any hook is already installed"
    requirement: HOOK-03
    verification:
      - kind: unit
        ref: "internal/cli/init_advisory_test.go#TestInitAdvisory_WatcherDisabled_HooksAlreadyInstalled"
        status: pass
    human_judgment: false
  - id: D5
    description: "uninit --force best-effort removes codegraph's marker blocks from the three git hooks after removing .codegraph/, printing a Removed line, without ever failing uninit"
    requirement: HOOK-03
    verification:
      - kind: unit
        ref: "internal/cli/uninit_test.go#TestUninit_RemovesGitHooks"
        status: pass
      - kind: unit
        ref: "internal/cli/uninit_test.go#TestUninit_NoHooksInstalled_NoRemovalLine"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-17
status: complete
---

# Phase 5 Plan 5: Wire HOOK-03's watcher-fallback advisory into init/uninit Summary

**Non-interactive plain-text port of TS `offerWatchFallback` on init's success path, plus best-effort git-hook cleanup on `uninit --force` — both gated exactly as TS, both mutation-proof-tested.**

## Performance

- **Duration:** 12 min
- **Completed:** 2026-07-17
- **Tasks:** 3 completed
- **Files modified:** 4 (2 modified, 2 created)

## Accomplishments

- `internal/cli/init.go`'s success path now calls `printWatchFallbackAdvisory`, a gate-for-gate port of TS `offerWatchFallback` (`installer/index.js` ~476-525): silent when `watch.WatchDisabledReason` returns `""`; otherwise warns + prints the frozen-index line, then branches on `gitmeta.IsGitRepo` (non-repo → manual `codegraph sync` hint) and `githooks.Status` (any hook installed → already-installed notice; otherwise → `codegraph githooks install` pointer). The already-initialized early-return branch is byte-unchanged.
- `internal/cli/uninit.go`'s `RunE` now calls `githooks.Remove` immediately after `os.RemoveAll(codegraphDir)` succeeds, printing `Removed git <hooks> sync hook(s)` only when `len(result.Removed) > 0`. The cleanup's own result is never propagated as an error — a non-repo or no-hooks-installed target is a silent no-op.
- Four new init-advisory tests and two new uninit tests drive the real cobra command tree end-to-end against real-git fixtures (D-13); manually confirmed reverting either wiring breaks things (3 red tests for init.go; a build failure for uninit.go, since removing the call leaves the `githooks` import unused — a stronger mutation-proof signal than a red test alone).

## Task Commits

1. **Task 1: D-07 init success-path advisory** - `be7864a` (feat)
2. **Task 2: D-06 uninit best-effort hook cleanup** - `aee7da1` (feat)
3. **Task 3: Mutation-proof reachability tests** - `78e3bb5` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/cli/init.go` - added `printWatchFallbackAdvisory` and its call site after `printSummary`; new imports `internal/githooks`, `internal/gitmeta`, `internal/watch`
- `internal/cli/uninit.go` - added the D-06 `githooks.Remove` cleanup call and a local `plural` helper; new import `internal/githooks`
- `internal/cli/init_advisory_test.go` - 4 tests covering all 4 D-07 gate outcomes (enabled/silent, disabled+repo+no-hooks, disabled+non-repo, disabled+repo+already-installed)
- `internal/cli/uninit_test.go` - 2 tests covering D-06 cleanup (hooks removed + reported; no hooks → silent no-op)

## Decisions Made

- `watch.Probe{}` (bare zero value) is the correct D-07 gate call — not a test-seam violation. `WatchDisabledReason` defaults a `nil` `Probe.Env` to `os.Getenv` internally, so `t.Setenv("CODEGRAPH_NO_WATCH", "1")` deterministically forces "disabled" through the exact same code path production traffic uses, with no new CLI flag or plumbing needed. This matches 05-PATTERNS.md's documented gate pattern and satisfies D-13's injectable-Probe requirement without adding surface area.
- Reused `runGit`/`initGitRepo`/`markerBeginBytes` from `internal/cli/githooks_test.go` (landed in 05-04) rather than adding a third package-local copy — same package, no cross-package import needed.
- Added an optional fourth already-installed-branch test and a no-hooks-installed uninit test beyond the plan's minimum two tests, to cover all four D-07 gate outcomes and the D-06 never-blocks contract explicitly rather than relying on inference.

## Deviations from Plan

None — plan executed exactly as written. The plan's acceptance criteria (advisory gate order, injectable Probe reachability, byte-unchanged already-initialized branch, uninit cleanup non-propagation, removal-message gating) are all met and verified.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 5 (Git Sync Hooks) is now feature-complete: `internal/fsatomic` extraction (05-01), `gitmeta.IsGitRepo`/`HooksDir` (05-02), `internal/githooks` Install/Remove/Status (05-03), the `codegraph githooks install/remove/status` CLI tree (05-04), and this plan's `init`/`uninit` lifecycle wiring (05-05) together deliver HOOK-01, HOOK-02, and HOOK-03 in full. The HOOK-03 partial-install "some()" semantics edge and the D-08 init-on-already-initialized message residual remain flagged (not silently decided) for Phase 8's SURF-05 divergence table, per this plan's own must_haves. Phase 7's interactive clack-analogue select and the formal TEST-03 byte-invariance harness are the only deferred items, both explicitly out of scope here.

---
*Phase: 05-git-sync-hooks*
*Completed: 2026-07-17*

## Self-Check: PASSED

All created/modified files and task commit hashes verified present on disk and in git history.
