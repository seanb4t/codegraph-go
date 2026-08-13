---
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
plan: 04
subsystem: cli
tags: [upgrade, agents, claude-code, install, tdd]
dependency-graph:
  requires: [07-03]
  provides: [upgrade-post-swap-skill-refresh]
  affects: [internal/cli]
tech-stack:
  added: []
  patterns:
    - "package-level injectable func var (refreshInstalledSkillsFunc) mirroring upgradeRunFunc/interactiveAllowed"
key-files:
  created: []
  modified:
    - internal/cli/upgrade.go
    - internal/cli/upgrade_test.go
decisions:
  - "D-06 implemented: refreshInstalledSkills re-invokes Install() only for Claude locations already carrying a manifest (agents.ConfiguredSkillLocations), never widening scope to a location the user never configured"
  - "AutoAllow is always passed false on refresh — it's a per-invocation install-time choice not recorded in the manifest, and Install(AutoAllow:false) is a neutral no-op on that step rather than a deletion"
  - "D-07 implemented: a refresh failure after a successful swap prints a warning naming `codegraph install` and returns nil; a swap failure returns the swap error unchanged with no refresh attempted"
metrics:
  duration: "~35 min"
  completed: 2026-08-13
status: complete
actuals:
  tokens: 3039
  tasks: 2
  commits: 2
---

# Phase 07 Plan 04: Upgrade Post-Swap Skill Refresh (D-06/D-07) Summary

`codegraph upgrade` now refreshes the installed Claude Code skill package at every previously-configured location immediately after a successful binary swap, and reports a refresh failure as a separate warning rather than a failed upgrade.

## What Was Built

**Task 1 — post-swap refresh (D-06).** Added `refreshInstalledSkills(execPath string, out io.Writer) error` to `internal/cli/upgrade.go`, gated behind the injectable seam `refreshInstalledSkillsFunc` (same idiom as `upgradeRunFunc`/`interactiveAllowed`). It calls `agents.ConfiguredSkillLocations(agents.Claude)` to find every location that already carries a manifest, resolves the Claude target via `agents.ResolveTargetFlag`, and calls `Install(loc, agents.InstallOptions{ExecPath: execPath, AutoAllow: false})` for each. `RunE` captures the swap result, returns immediately on swap failure (no refresh attempted), skips the refresh entirely under `--check`, and otherwise invokes the refresh seam with the resolved exec path.

**Task 2 — D-07 reporting split.** When the refresh seam returns an error, `RunE` prints a warning to `cmd.OutOrStdout()` stating the binary upgrade succeeded, that the skill-package refresh did not, and naming `codegraph install` as the command to re-run — then returns `nil`, the value it would have returned had the refresh step not existed. The swap-failure path is untouched: `RunE` returns that error directly, unwrapped, so `errors.Is` against a sentinel still holds.

Both tasks followed RED→GREEN: each behavior set was proven to fail against the pre-implementation code (confirmed by temporarily reverting the relevant `RunE` line and re-running the new tests) before the implementation was restored and the tests turned green.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `<behavior>` tests and `<acceptance_criteria>` are implemented and passing as specified.

## Verification

- `go test ./internal/cli/... -run 'TestUpgradeCommand|TestRefreshInstalledSkills' -v` — all 10 tests pass (4 pre-existing + 6 new).
- `go test ./internal/cli/... ./internal/agents/... ./internal/upgrade/...` — green.
- `go vet ./...` — clean.
- `task test` (full phase gate, including `-race` and golden/wire-oracle suites) — green.
- No test performs a real network download; every `upgradeRunFunc`/`refreshInstalledSkillsFunc` call site in tests is a fake.

The plan's `<human-check>` (live-session smoke check of the global install path with a real Claude Code session) is a phase-level verification item, not scoped to this plan's two tasks, and is not exercised by this executor.

## Self-Check: PASSED

- `internal/cli/upgrade.go` — FOUND
- `internal/cli/upgrade_test.go` — FOUND
- Commit `2f37745` (Task 1) — FOUND in `git log --oneline`
- Commit `060fee4` (Task 2) — FOUND in `git log --oneline`
