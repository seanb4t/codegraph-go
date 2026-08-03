---
phase: 02-status-content-git-worktree-awareness
plan: 01
subsystem: infra
tags: [git, worktree, os-exec, gitmeta, tdd]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: internal/query read seam (Engine, Marshal*JSON, render_markdown.go patterns) this package will plug into in later waves
provides:
  - internal/gitmeta package — stdlib-only, best-effort git worktree/index-mismatch detection
  - gitmeta.WorktreeRoot / gitmeta.CommonDir git subprocess primitives (5s bound, stderr discarded)
  - gitmeta.DetectIndexMismatch — TS's 4-gate cascade ported verbatim, correct gate-4 polarity
  - gitmeta.Mismatch.Warning() / .Notice() — byte-identical TS strings, nil-safe
  - gitmeta.CachingDetector — server-lifetime memoization of positive AND negative verdicts
  - seven real-git TEST-02 fixtures (linked-worktree, .claude/worktrees, submodule, nested-clone, monorepo-subdir, symlinked, non-git) plus a plain-ancestor gate-3 isolation subtest
affects: [02-02, 02-03, 02-04, 02-05, 02-06, 02-07, phase-5-git-sync-hooks]

tech-stack:
  added: []
  patterns:
    - "Bounded best-effort git subprocess (exec.CommandContext + 5s timeout + stdin=nil + Output() stderr-discard, any error -> safe zero value, never propagated)"
    - "Real-git fixture builder in t.TempDir() (runGit t.Skipf-on-error, never t.Fatalf; initRepo commit helper) — first repo-BUILDING test infra in this codebase (previously only repo-cloning existed)"
    - "Nil-receiver-safe methods returning \"\" (mirrors staleBanner's shape) so callers never need a nil guard"
    - "Package-scoped mutex-guarded cache (not Engine-scoped) for state that must survive across per-call-fresh Engine construction"

key-files:
  created:
    - internal/gitmeta/worktree.go
    - internal/gitmeta/detect.go
    - internal/gitmeta/notice.go
    - internal/gitmeta/cache.go
    - internal/gitmeta/fixtures_test.go
    - internal/gitmeta/detect_test.go
    - internal/gitmeta/notice_test.go
    - internal/gitmeta/cache_test.go
  modified: []

key-decisions:
  - "Ported TS sync/worktree.js's 4-gate cascade verbatim, gate order and polarity preserved exactly (gate 4: differing common dirs SUPPRESS the warning, not trigger it — documented in full on DetectIndexMismatch and on the submodule/nested-clone fixtures)"
  - "CachingDetector lives in internal/gitmeta itself (not on query.Engine) per D-13's correction — internal/mcp's openEngine rebuilds Engine per call, so an Engine-scoped cache would give zero cross-call benefit on the exact surface it exists for"
  - "T-02-04 accepted: Mismatch.WorktreeRoot/IndexRoot intentionally carry absolute host paths in the warning/notice text — a deliberate, scoped exception to this codebase's usual no-host-paths-in-MCP-output stance, because the warning is meaningless without naming the two trees involved and TS emits identical paths"
  - "The monorepo-subdir fixture's primary case (index=repo root) is caught by gate 2 before gate 3 is reached; added a separate 'monorepo-subdir-plain-ancestor' subtest (index=a non-git tmp ancestor) to isolate gate 3 specifically, per D-15's 'assert the plain-ancestor variant too' instruction, without adding an 8th named fixture builder"

patterns-established:
  - "internal/gitmeta stays free of internal/query/internal/mcp/internal/cli imports (verified: only comment mentions, zero actual imports) so Phase 5's git sync hooks can reuse it unchanged"
  - "Zero new dependencies (go.mod/go.sum diff is empty) — stdlib context/os/exec/path/filepath/strings/sync/time only"

requirements-completed: [WORK-01, WORK-03, TEST-02]

coverage:
  - id: D1
    description: "gitmeta.DetectIndexMismatch implements TS's 4-gate cascade and correctly flags a borrowed linked-worktree index (including the motivating .claude/worktrees/<name>/ layout) while suppressing submodule/nested-clone/monorepo-subdir/symlinked/non-git false positives"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/gitmeta/detect_test.go#TestFixtureVerdicts"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every git failure mode (absent git, timeout, non-repo, subprocess error) degrades to nil/empty-string, never an error or panic"
    requirement: "WORK-03"
    verification:
      - kind: unit
        ref: "internal/gitmeta/detect_test.go#TestFixtureVerdicts/non-git"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/cache_test.go#TestCachingDetectorNilReceiverSafety"
        status: pass
    human_judgment: false
  - id: D3
    description: "Six required real-git layouts plus non-git have passing fixtures matching the D-15 verdict matrix"
    requirement: "TEST-02"
    verification:
      - kind: unit
        ref: "internal/gitmeta/fixtures_test.go (seven builders) + internal/gitmeta/detect_test.go#TestFixtureVerdicts"
        status: pass
    human_judgment: false
  - id: D4
    description: "Warning()/Notice() strings are byte-identical to TS 1.3.1's sync/worktree.js, including the U+26A0 glyph with no U+FE0F variation selector, and are nil-safe"
    verification:
      - kind: unit
        ref: "internal/gitmeta/notice_test.go#TestMismatchWarningVerbatim,TestMismatchNoticeVerbatim,TestMismatchNilReceiverSafety"
        status: pass
    human_judgment: false
  - id: D5
    description: "CachingDetector memoizes both positive and negative verdicts on TS's own cache key and is nil-receiver-safe"
    verification:
      - kind: unit
        ref: "internal/gitmeta/cache_test.go#TestCachingDetectorMemoizesPositive,TestCachingDetectorMemoizesNegative,TestCachingDetectorNilReceiverSafety"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 1: internal/gitmeta — Git Worktree Detection Summary

**New `internal/gitmeta` package porting TS 1.3.1's 4-gate borrowed-worktree-index detection verbatim, backed by seven real-git fixtures and a server-lifetime CachingDetector.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-07-15T22:09:00Z (approx.)
- **Completed:** 2026-07-15T22:31:49Z
- **Tasks:** 3
- **Files modified:** 8 (all new)

## Accomplishments
- `gitmeta.WorktreeRoot`/`gitmeta.CommonDir`: 5s-bounded `git rev-parse` subprocess wrappers that degrade to `""` on any error, absent git, or timeout — never block a read query (WORK-03)
- `gitmeta.DetectIndexMismatch`: TS `detectWorktreeIndexMismatch`'s 4-gate cascade ported verbatim, with gate 4's counterintuitive polarity (differing git common dirs SUPPRESS the warning — submodule/embedded clone, not a real mismatch) documented at three separate sites so it can't be silently inverted (WORK-01)
- Seven real-`git` fixtures built via `os/exec` in `t.TempDir()` (no faked `.git` dirs), including the motivating `.claude/worktrees/<name>/` true-positive layout, all passing against the D-15 verdict matrix (TEST-02)
- `(*Mismatch).Warning()`/`.Notice()`: byte-identical to TS's `worktreeMismatchWarning`/`worktreeMismatchNotice`, including the exact U+26A0 glyph bytes with no U+FE0F variation selector, nil-safe like `staleBanner`
- `gitmeta.CachingDetector`: mutex-guarded memoization of both positive and negative verdicts on TS's own `startPath\x00indexRoot` cache key, correctly placed in `internal/gitmeta` (not `Engine`) per D-13's correction, nil-receiver-safe

## Task Commits

Each task was committed atomically (TDD RED/GREEN):

1. **Task 1: Six-layout real-git fixture builder + RED verdict-matrix test** - `5d3aa72` (test)
2. **Task 2: git primitives + the 4-gate cascade** - `3322d0b` (feat) — GREEN
3. **Task 3: Verbatim notice/warning strings + CachingDetector** - `4d08261` (feat)

_No REFACTOR commit was needed — GREEN implementations matched the target shape from Task 1's fixtures with no follow-up cleanup._

## Files Created/Modified
- `internal/gitmeta/worktree.go` - `WorktreeRoot`/`CommonDir`/`realpath`, the bounded git subprocess primitives
- `internal/gitmeta/detect.go` - `Mismatch` struct + `DetectIndexMismatch`'s 4-gate cascade
- `internal/gitmeta/notice.go` - `Warning()`/`Notice()`, verbatim TS strings + the U+26A0 `warnGlyph` constant
- `internal/gitmeta/cache.go` - `CachingDetector`, mutex-guarded `map[string]*Mismatch` memoization
- `internal/gitmeta/fixtures_test.go` - `runGit`/`initRepo` helpers + seven real-git layout builders
- `internal/gitmeta/detect_test.go` - `TestFixtureVerdicts` table-driven test over all seven fixtures + plain-ancestor gate-3 subtest
- `internal/gitmeta/notice_test.go` - nil-safety + byte-exact string assertions
- `internal/gitmeta/cache_test.go` - positive/negative memoization + nil-receiver assertions

## Decisions Made
- Ported the 4-gate cascade, gate order, and gate-4 polarity exactly as read from the live TS 1.3.1 source at `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/worktree.js` (D-01/D-02) — no paraphrasing, no reconstruction from memory.
- `CachingDetector` placed in `internal/gitmeta`, not on `query.Engine`, per D-13's correction: `internal/mcp`'s `openEngine` rebuilds a fresh `Engine` on every tool call, so an Engine-scoped cache would deliver zero cross-call benefit on the exact long-lived surface (the MCP server) the cache exists to help.
- T-02-04 (threat register): `Mismatch.WorktreeRoot`/`IndexRoot` intentionally carry absolute host paths in the rendered warning/notice text. This is a deliberate, scoped exception to the codebase's general "no host paths in MCP output" convention (`StatusResult.ProjectPath`/`IndexPath` render as `""`) — the warning is useless without naming the two trees involved, and TS emits the identical paths. The consumer is the operator's own agent session on the operator's own machine.
- The monorepo-subdir fixture's literal Test 5 case (start=subdir, index=repo root) is short-circuited by gate 2 before gate 3 is reached, since the index root equals `WorktreeRoot(start)`. To genuinely isolate gate 3 per D-15's "assert the plain-ancestor variant too" instruction, added a `monorepo-subdir-plain-ancestor` subtest (index=a non-git ancestor directory) inside `TestFixtureVerdicts` rather than an 8th named `newXFixture` builder — keeps the "exactly seven fixture builders" acceptance criterion literal while still covering gate 3 distinctly.

## Deviations from Plan

None — plan executed exactly as written. All three tasks' acceptance criteria were verified directly (grep counts for `gitTimeout`/`protocol.file.allow=always`, `t.Skipf`-never-`t.Fatalf` on git errors, byte-exact glyph assertions, `go vet`/`go build ./...` clean, `go.mod`/`go.sum` diff empty).

## Issues Encountered

None. One pre-flight check was done manually (`git worktree add -b feature nested/deep/wt` in a scratch dir) to confirm git auto-creates leading directories for the `.claude/worktrees/phase-2` fixture — no code changes resulted, just confirmed the fixture builder didn't need extra `MkdirAll` scaffolding.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/gitmeta`'s full exported surface (`WorktreeRoot`, `CommonDir`, `DetectIndexMismatch`, `Mismatch`, `.Warning()`/`.Notice()`, `CachingDetector`/`NewCachingDetector`) is ready for plan 02-04's `Engine.startPath` (D-14) plumbing and `StatusResult.WorktreeMismatch` wiring.
- No fixture layout had to `t.Skip` on this execution machine (macOS, git present) — all seven fixtures plus the plain-ancestor subtest ran and passed. A CI/Linux run without git on `PATH` would skip every fixture test gracefully (via `runGit`'s `t.Skipf`), never fail the suite.
- Zero new dependencies were introduced; `go.mod`/`go.sum` are byte-identical to before this plan.

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*

## Self-Check: PASSED

All 8 created files verified present on disk; all 3 task commits (5d3aa72, 3322d0b, 4d08261) verified present in git log.
