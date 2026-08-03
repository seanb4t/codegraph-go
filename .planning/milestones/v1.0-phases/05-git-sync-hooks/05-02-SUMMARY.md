---
phase: 05-git-sync-hooks
plan: 02
subsystem: infra
tags: [go, git, exec, gitmeta]

# Dependency graph
requires:
  - phase: 05-git-sync-hooks
    provides: "05-01's internal/gitmeta package tone/exec contract (worktree.go) this plan extends"
provides:
  - "gitmeta.IsGitRepo(ctx, dir) bool — git-repo probe, degrades to false on any failure"
  - "gitmeta.HooksDir(ctx, projectRoot) string — git hooks dir resolution honoring core.hooksPath and linked worktrees, degrades to \"\" on any failure"
affects: [05-03]

# Tech tracking
tech-stack:
  added: []
  patterns: ["gitmeta exec contract reused verbatim for two new git-exec seams (CommandContext + gitTimeout + cmd.Dir + cmd.Stdin=nil + degrade-on-error)"]

key-files:
  created: [internal/gitmeta/githooks.go, internal/gitmeta/githooks_test.go]
  modified: []

key-decisions:
  - "Reused the package-level gitTimeout constant (worktree.go:28) — no redeclaration"
  - "HooksDir deliberately omits realpath (D-04): relative output is joined against projectRoot, absolute output is passed through unchanged, never symlink-resolved"
  - "Linked-worktree HooksDir test compares via filepath.EvalSymlinks rather than raw string equality — git's own --git-path output is internally realpath-resolved for a linked worktree while the plain-repo comparison path (built by joining the caller-supplied projectRoot) is not, so on hosts with a symlinked TMPDIR (macOS /var -> /private/var) the raw strings differ by that hop alone; EvalSymlinks isolates the true functional property (same underlying directory) without smuggling realpath into HooksDir's own implementation"

patterns-established:
  - "internal/gitmeta/githooks.go is the single new git-exec seam this phase introduces — internal/githooks (05-03) consumes IsGitRepo/HooksDir and never shells out to git directly"

requirements-completed: [HOOK-01, HOOK-02, HOOK-03]

coverage:
  - id: D1
    description: "gitmeta.IsGitRepo probes whether a directory is inside a git working tree, following the worktree.go exec contract and degrading to false on any failure"
    requirement: "HOOK-03"
    verification:
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestIsGitRepo_TrueForInitializedRepo"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestIsGitRepo_FalseForNonGitDir"
        status: pass
    human_judgment: false
  - id: D2
    description: "gitmeta.HooksDir resolves the git hooks directory via --git-path hooks, correctly handling relative resolution, absolute passthrough, core.hooksPath, and linked worktrees, degrading to empty string on any failure"
    requirement: "HOOK-01"
    verification:
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestHooksDir_PlainRepoResolvesUnderProjectRoot"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestHooksDir_HonorsCoreHooksPath"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestHooksDir_LinkedWorktreeResolvesToSharedCommonHooksDir"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/githooks_test.go#TestHooksDir_EmptyForNonGitDir"
        status: pass
    human_judgment: false

# Metrics
duration: 3min
completed: 2026-07-16
status: complete
---

# Phase 5 Plan 2: gitmeta IsGitRepo & HooksDir Summary

**Added `gitmeta.IsGitRepo` and `gitmeta.HooksDir` as the two new single-seam git-exec probes internal/githooks (05-03) will consume — HooksDir correctly honors `core.hooksPath` and resolves linked worktrees to the shared common hooks dir, both following worktree.go's existing exec contract verbatim.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-16T21:22:00-04:00
- **Completed:** 2026-07-16T21:24:51-04:00
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `gitmeta.IsGitRepo(ctx, dir) bool` — `git rev-parse --is-inside-work-tree` probe, degrades to `false` on any exec failure
- `gitmeta.HooksDir(ctx, projectRoot) string` — `git rev-parse --git-path hooks` resolution: relative output joined against `projectRoot`, absolute output passed through, no `.git/hooks` hand-joining anywhere
- Real-git fixture tests cover plain repo, `core.hooksPath`-honoring, linked-worktree shared-common-dir, and non-git-dir cases for both functions

## Task Commits

Each task was committed atomically:

1. **Task 1: IsGitRepo probe with real-git fixture tests** — `359eb96` (test, RED) + `adee75f` (feat, GREEN)
2. **Task 2: HooksDir resolution honoring core.hooksPath and linked worktrees** — `b19eae3` (test, RED) + `29d7c12` (feat, GREEN, includes test-comparison fix)

**Plan metadata:** pending (docs: complete plan)

_Both TDD tasks landed as two commits each (RED test → GREEN implementation); no refactor commit needed (implementation matched the target shape from PATTERNS.md)._

## Files Created/Modified
- `internal/gitmeta/githooks.go` - `IsGitRepo` and `HooksDir`, following `worktree.go`'s `WorktreeRoot`/`CommonDir` exec contract exactly; reuses the existing package `gitTimeout` constant
- `internal/gitmeta/githooks_test.go` - Real-git fixture tests for both functions, reusing `runGit`/`initRepo`/`newLinkedWorktreeFixture` from `fixtures_test.go`

## Decisions Made
- HooksDir's linked-worktree test compares hooks-dir paths through `filepath.EvalSymlinks` rather than raw string equality, because on macOS the TMPDIR root (`/var`) is itself a symlink to `/private/var`: git's own `--git-path hooks` output for a linked worktree is internally realpath-resolved to `/private/var/...`, while the plain-repo comparison path (built in the test by joining the caller-supplied `projectRoot`, which is `/var/...`) is not. This is a test-fixture artifact of the host's symlinked temp directory, not a functional divergence in `HooksDir` itself — the function still correctly omits `realpath` per D-04 (verified separately: `TestHooksDir_PlainRepoResolvesUnderProjectRoot` asserts an exact, non-symlink-resolved path).

## Deviations from Plan

None - plan executed exactly as written. The one test-comparison detail above (EvalSymlinks normalization) is a test-correctness fix within Task 2's own TDD cycle, not a deviation from the plan's specified behavior — `HooksDir`'s implementation matches the plan's `<action>` and `<acceptance_criteria>` verbatim (no realpath call, relative-join/absolute-passthrough only).

## Issues Encountered
- Task 2's linked-worktree test initially failed on a raw string comparison due to the macOS `/var` → `/private/var` symlink hop described above. Root-caused via systematic debugging (confirmed both paths point at the same real directory, confirmed `HooksDir`'s implementation has no realpath call per the acceptance criteria), then fixed by normalizing the test's comparison through `filepath.EvalSymlinks` rather than altering `HooksDir` itself.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `gitmeta.IsGitRepo` and `gitmeta.HooksDir` are ready for `internal/githooks` (05-03) to consume as the sole git-introspection seam — no new git-exec call sites needed there.
- No hand-joined `.git/hooks` path exists anywhere in the codebase (verified via search).
- No blockers for 05-03/05-04/05-05.

---
*Phase: 05-git-sync-hooks*
*Completed: 2026-07-16*

## Self-Check: PASSED
