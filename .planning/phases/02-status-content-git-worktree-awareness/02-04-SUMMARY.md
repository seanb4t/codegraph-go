---
phase: 02-status-content-git-worktree-awareness
plan: 04
subsystem: query
tags: [worktree, gitmeta, engine, status, mcp, tdd]

requires:
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-01's internal/gitmeta package (DetectIndexMismatch, Mismatch.Warning()/Notice(), CachingDetector) — this plan wires it into the shared read seam"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-02's StatusResult.DbSizeBytes/FilesByLanguage and its decision-table convention — this plan adds the worktreeMismatch row to the same table"
provides:
  - "Engine.startPath — the caller's absolutized OpenAt start directory, retained instead of discarded (D-14)"
  - "Engine.WorktreeMismatch() *gitmeta.Mismatch — the single accessor both CLI and MCP surfaces will call, once-per-Engine cached via sync.Once"
  - "Engine.UseDetector(*gitmeta.CachingDetector) — the seam plan 02-06's MCP BuildServer injects a server-scoped detector through"
  - "StatusResult.WorktreeMismatch is now live *gitmeta.Mismatch (was an always-nil *string placeholder)"
  - "query.WorktreeNotice(*gitmeta.Mismatch) string — the compact-notice prepend helper for the 7 non-status read tools"
  - "query.WorktreeWarningBlockquote(*gitmeta.Mismatch) string — the MCP status blockquote wrap of the verbose warning"
affects: [02-05, 02-06, 02-07]

tech-stack:
  added: []
  patterns:
    - "sync.Once-guarded Engine-scoped detection cache, paired with an injectable gitmeta.CachingDetector for cross-call (MCP server-lifetime) caching — Engine alone gives zero cross-call benefit since openEngine rebuilds fresh per tool call"
    - "Nil-safe Render*/Notice* helpers returning \"\" on nil (mirrors staleBanner) so every call site needs no nil guard"
    - "Pebble's per-store-directory exclusive lock means two Engines on the SAME store cannot be open concurrently in tests — close the first before opening the second, mirroring internal/mcp's openEngine fresh-per-call discipline"

key-files:
  created:
    - internal/query/engine_worktree_test.go
    - internal/query/worktree_notice.go
  modified:
    - internal/query/engine.go
    - internal/query/status.go
    - internal/query/files_status_test.go

key-decisions:
  - "OpenAt absolutizes start via filepath.Abs before storing it as Engine.startPath (Pitfall 3) — on a filepath.Abs error, startPath is left empty rather than failing OpenAt, so a status query never dies because a path could not be absolutized (WORK-03)"
  - "WorktreeMismatch() degrades to nil without panicking whenever startPath==\"\" or repoRoot==\"\" — the exact shape computeStale already uses for e.repoRoot==\"\" — so Engines built via New/NewWithRoot (every existing test, and any future caller with no filesystem context) are safe by construction"
  - "UseDetector's cache injection point exists because internal/mcp's openEngine builds a fresh Engine on every tool call by design; an Engine-scoped sync.Once alone gives zero cross-call benefit on the exact long-lived surface (the MCP server process) the cache exists to help — plan 02-06 constructs one CachingDetector per server and calls UseDetector on every Engine it opens"
  - "StatusResult.WorktreeMismatch changed type from *string to *gitmeta.Mismatch (not a widening for its own sake) — TS's own --json branch emits {worktreeRoot,indexRoot} or null, so the *string placeholder was never the parity shape; nil still marshals to JSON null, keeping the frozen golden fixtures green"
  - "T-02-14 accepted and documented in StatusResult's decision table: worktreeMismatch intentionally carries absolute host paths when non-nil, a deliberate scoped exception to the table's own projectPath/indexPath privacy-blanking stance, because the warning is useless without naming the two trees and TS emits identical paths — a clean tree still leaks nothing (nil)"
  - "engine_worktree_test.go builds its own local runGitW/worktreeMismatchFixture helpers rather than importing internal/gitmeta's fixtures_test.go, since Go test files are not importable across packages; reuses engine_test.go's copyFixture/indexFixture for the indexing half per the plan's explicit instruction"
  - "Fixed a latent Pebble exclusive-store-lock conflict discovered while writing the RED tests: opening two Engines concurrently on the SAME store directory deadlocks (\"lock held by current process\"); tests now close the first Engine before opening the second wherever both point at the same store, mirroring internal/mcp's openEngine fresh-per-call discipline rather than internal/mcp's own behavior being at fault"

patterns-established:
  - "internal/query/worktree_notice.go's warnGlyph constant is a local, byte-identical redeclaration of internal/gitmeta's unexported glyph — U+26A0 with no U+FE0F variation selector — since the gitmeta constant is unexported and this file needs it for the MCP status blockquote prefix"

requirements-completed: [WORK-01, WORK-02]

coverage:
  - id: D1
    description: "OpenAt retains the caller's absolutized start path as Engine.startPath instead of discarding it after the upward walk"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestOpenAtAbsolutizesStartPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "Engine.WorktreeMismatch() computes a live verdict from startPath vs repoRoot via gitmeta.DetectIndexMismatch, detecting the motivating .claude/worktrees/ borrowed-index layout, and is cached once per Engine"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestEngineWorktreeMismatchViaOpenAt"
        status: pass
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestWorktreeMismatchCachedPerEngine"
        status: pass
    human_judgment: false
  - id: D3
    description: "Engines built via New/NewWithRoot (no startPath) degrade to nil without panicking"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestEngineWorktreeMismatchDegradesSafely"
        status: pass
    human_judgment: false
  - id: D4
    description: "UseDetector lets a shared gitmeta.CachingDetector serve two independently-opened Engines, proven via *Mismatch pointer identity across engines"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestWorktreeMismatchSharedDetector"
        status: pass
    human_judgment: false
  - id: D5
    description: "StatusResult.WorktreeMismatch is live (*gitmeta.Mismatch) and status --json emits it as {worktreeRoot,indexRoot} or null, matching TS; nil still marshals to null so the frozen golden fixtures stay green"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestStatusWorktreeMismatchLive"
        status: pass
      - kind: unit
        ref: "internal/query/engine_worktree_test.go#TestStatusWorktreeMismatchJSONShape"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity/status"
        status: pass
    human_judgment: false
  - id: D6
    description: "WorktreeNotice/WorktreeWarningBlockquote exist, are nil-safe (\"\" on nil), and carry the verbatim D-11 strings/glyph for plans 02-05/02-06/02-07 to wire in"
    requirement: "WORK-02"
    verification:
      - kind: unit
        ref: "go build ./... (compiles cleanly; both helpers exported from internal/query/worktree_notice.go)"
        status: pass
      - kind: integration
        ref: "go test ./... (full suite green, no regressions)"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 4: Engine.startPath + Live WorktreeMismatch Summary

**`OpenAt` now retains the caller's absolute start path instead of discarding it, `Engine.WorktreeMismatch()` computes a live borrowed-index verdict via `internal/gitmeta`, `StatusResult.WorktreeMismatch` flipped from an always-nil `*string` placeholder to a live `*gitmeta.Mismatch`, and two new nil-safe notice helpers (`WorktreeNotice`/`WorktreeWarningBlockquote`) are ready for the CLI/MCP wiring plans.**

## Performance

- **Duration:** 35 min
- **Tasks:** 3 (TDD RED / GREEN / GREEN)
- **Files modified:** 5 (2 new, 3 modified)

## Accomplishments

- `OpenAt` absolutizes `start` via `filepath.Abs` and stores it as `Engine.startPath` (D-14) — the load-bearing plumbing fix, since `OpenAt` is the one read seam both CLI commands and MCP tool handlers call, delivering CLI+MCP worktree awareness in this single commit
- `Engine.WorktreeMismatch() *gitmeta.Mismatch` — the single accessor both surfaces use, guarded by `sync.Once` so detection runs at most once per Engine; degrades to `nil` without panicking whenever `startPath==""` or `repoRoot==""` (Engines built via `New`/`NewWithRoot`)
- `Engine.UseDetector(*gitmeta.CachingDetector)` — the seam plan 02-06's MCP `BuildServer` will inject a server-scoped detector through, since `openEngine` rebuilds a fresh `Engine` per tool call and an Engine-scoped cache alone gives zero cross-call benefit
- `StatusResult.WorktreeMismatch` is now live, populated from `e.WorktreeMismatch()` in `Status()`; type changed from `*string` to `*gitmeta.Mismatch` (TS's own `{worktreeRoot,indexRoot}`/`null` shape), with nil still marshaling to JSON `null` so the frozen golden fixtures stay green
- `query.WorktreeNotice`/`query.WorktreeWarningBlockquote` — both nil-safe, no ANSI, ready for plans 02-05 (CLI wiring) and 02-06 (MCP wiring)

## Task Commits

Each task was committed atomically (TDD RED/GREEN):

1. **Task 1: RED — Engine.startPath retention, live mismatch, and safe degradation** - `72ceb3a` (test)
2. **Task 2: GREEN — Engine.startPath, WorktreeMismatch(), UseDetector** - `eb9528b` (feat)
3. **Task 3: GREEN — live StatusResult.WorktreeMismatch + the two notice helpers** - `1dd5c39` (feat)

_No REFACTOR commit was needed — both GREEN implementations matched Task 1's target shape with no follow-up cleanup._

## Files Created/Modified

- `internal/query/engine_worktree_test.go` - local `runGitW`/`worktreeMismatchFixture` helpers plus 7 tests covering the D-14/D-13/WORK-01 contract (live mismatch, safe degradation, live `Status()`, JSON shape, relative-path absolutization, once-per-Engine caching, shared-detector caching)
- `internal/query/engine.go` - `Engine.startPath`/`detector`/`mismatchOnce`/`mismatchCache` fields, `WorktreeMismatch()`, `UseDetector()`, `OpenAt` absolutizes and retains `start`, doc comment records the repoRoot-vs-startPath distinction
- `internal/query/status.go` - `StatusResult.WorktreeMismatch` type change to `*gitmeta.Mismatch`, `Status()` sets it from `e.WorktreeMismatch()`, decision-table row updated to record the live shape and the T-02-14 host-path exception
- `internal/query/worktree_notice.go` - new file: `WorktreeNotice`/`WorktreeWarningBlockquote`, local `warnGlyph` constant
- `internal/query/files_status_test.go` - existing placeholder subtest retitled/re-commented: `WorktreeMismatch` nil here is now a genuine live verdict (no borrowed worktree in this fixture), not an inert placeholder; `PendingChanges` all-zero assertion unchanged (D-06)

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:
- `filepath.Abs` errors leave `startPath` empty rather than failing `OpenAt` (WORK-03's best-effort contract extends to the plumbing itself, not just detection)
- The Pebble-store-lock conflict discovered while writing RED tests (two Engines open concurrently on the same store dir deadlock) was fixed in the test file itself, not treated as a product bug — it's a correct reflection of Pebble's real exclusive-lock semantics, and `internal/mcp`'s `openEngine` already follows the fresh-per-call discipline that avoids it in production

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pebble exclusive-store-lock conflict in the shared-detector and dual-engine test cases**
- **Found during:** Task 2 (GREEN — Engine.startPath, WorktreeMismatch(), UseDetector)
- **Issue:** `TestWorktreeMismatchSharedDetector`, `TestStatusWorktreeMismatchLive`, and `TestStatusWorktreeMismatchJSONShape` each opened two `Engine`s that resolve to the SAME `.codegraph/store` directory while both were still held open (deferred via `t.Cleanup`). Pebble's `graphstore.Open` takes an exclusive lock per store directory, so the second `OpenAt` call failed with "lock held by current process".
- **Fix:** Restructured all three tests to close the first Engine (`closer1.Close()`) immediately after extracting the value needed from it, before opening the second Engine on the same store — mirroring `internal/mcp`'s `openEngine`, which already opens and closes a fresh Engine per call rather than holding two open concurrently.
- **Files modified:** `internal/query/engine_worktree_test.go`
- **Verification:** All affected tests pass; full `go test ./...` green.
- **Committed in:** `eb9528b` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug, test-infrastructure only — no product code affected)
**Impact on plan:** No scope creep. The fix is entirely within the RED test file's own fixture-sequencing logic; `Engine`/`OpenAt`/`WorktreeMismatch` behavior is unaffected.

## Issues Encountered

None beyond the Pebble-lock deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `Engine.WorktreeMismatch()`, `Engine.UseDetector()`, `StatusResult.WorktreeMismatch`, `query.WorktreeNotice`, and `query.WorktreeWarningBlockquote` are all live, tested, and exported for plan 02-06 (MCP wiring: constructs one `gitmeta.CachingDetector` per server, calls `UseDetector` on every opened Engine, applies `WorktreeNotice` to the 7 non-status tools and `WorktreeWarningBlockquote` to `codegraph_status`) and plan 02-05/02-07 (CLI wiring: prints the compact notice/verbose warning to stdout in human mode only, per D-12).
- `MarshalStatusJSON` itself was never modified (only the struct it serializes changed) — the CLI `--json` contract and golden-parity oracle remain intact, confirmed via `git diff internal/query/status.go | grep -c '^-.*func MarshalStatusJSON'` returning 0.
- Zero new dependencies; `go.mod`/`go.sum` are byte-identical to before this plan (`git diff --stat go.mod go.sum` empty).
- Manual reachability check confirmed end-to-end: `go run ./cmd/codegraph status --json --path .` on this repo's own (non-worktree) checkout emits `"worktreeMismatch": null`.
- No blockers. `go build ./...`, `go vet ./...`, and `go test ./...` are all clean.

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*

## Self-Check: PASSED

All 6 referenced files verified present on disk; all 3 task commits (72ceb3a, eb9528b, 1dd5c39) verified present in git log.
