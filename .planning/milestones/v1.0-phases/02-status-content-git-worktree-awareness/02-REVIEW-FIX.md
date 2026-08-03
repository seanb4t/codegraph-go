---
phase: 02-status-content-git-worktree-awareness
fixed_at: 2026-07-16T01:23:21Z
review_path: .planning/phases/02-status-content-git-worktree-awareness/02-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 2: Code Review Fix Report

**Fixed at:** 2026-07-16T01:23:21Z
**Source review:** .planning/phases/02-status-content-git-worktree-awareness/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (2 Critical, 5 Warning — Info findings out of scope per fix instructions)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: The MCP worktree notice is unreachable through `codegraph serve --mcp`

**Files modified:** `internal/cli/serve.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go`, `internal/mcp/reconnect_test.go`, `internal/mcp/markdown_test.go`
**Commit:** dc6ddd5
**Applied fix:** `BuildServer` now takes two distinct paths — `repoPath` (the resolved index root, used as the confinement boundary) and `startPath` (the caller's actual cwd, used as every handler's default `path`). `internal/cli/serve.go` no longer collapses both into the post-`ResolveCodegraphDir` value; it passes the pre-overwrite `start` through as `startPath`. `openEngine`/`exploreHandler`/`companionHandler` were split accordingly, confining the resolved path to `repoPath` while defaulting to `startPath`. Because `repoPath` is always `startPath` itself or an ancestor of it (by construction of `ResolveCodegraphDir`'s upward walk), the default path always passes confinement — no security weakening. `internal/mcp/markdown_test.go` was rewritten to add a `deriveServeRepoPath` helper that replicates `serve.go`'s exact derivation, replacing every hand-picked `wt`-as-repoPath call site that had hidden the bug.

**Reachability proof (beyond the test suite):** built the real `codegraph` binary, created a real git repo + `git worktree add` linked worktree under `.claude/worktrees/probe`, indexed the main checkout via `codegraph init`, then drove `codegraph serve --mcp -p <worktree>` as a real subprocess over stdio JSON-RPC (via `mcpclient.NewStdioMCPClient`) and called `codegraph_callers`. The response's text content began with the `⚠ CodeGraph results below come from a different git worktree` notice — confirming the fix is reachable through the actual production entry point, not just a unit test. The probe worktree, binary, and throwaway driver code were all cleaned up afterward (`git worktree remove --force`, `rm -rf`); nothing from the probe was committed.

`TestOpenEnginePathConfinedToRepoRoot` (the CR-02/Phase-1-era confinement guard) was re-verified passing after the change — confinement was not weakened.

### CR-02: `TestGoldenParity/status`'s `dbSizeBytes` assertion measures the wrong directory

**Files modified:** `testdata/golden/golden_parity_test.go`
**Commit:** 1cfd5b3
**Applied fix:** `buildEngineAt` now copies `sourceDir` into a fresh temp dir and indexes it on disk at `<dst>/.codegraph/store` via the existing `buildIndexedFixture` helper, then opens it via `query.OpenAt(dst)` — mirroring what `TestExploreCLIMatchesMCP`/`TestNodeCLIMatchesMCP` already do against the same weft-go corpus, so this is not a new performance cost class. `Status()`'s `dbSizeBytes` now measures the exact store the test built, with no dependency on filesystem pollution to pass. Removed the now-unused `graphstore` import.

**Verification:** reproduced the reviewer's failure first (fresh `git clone` of weft at the pinned commit `f89ae3ea4e4c37509f7302fd4e37986212a72079`, confirmed no `.codegraph/store` present), then confirmed `go test ./testdata/golden/ -run TestGoldenParity/status` passes against that clean clone after the fix (`CODEGRAPH_WEFT_CORPUS=<fresh-clone>`). Also re-ran against the polluted `../weft` checkout to confirm no regression. Full `go test ./testdata/golden/...` passes.

### WR-01: `Engine.WorktreeMismatch()` hardcoded `context.Background()`

**Files modified:** `internal/query/engine.go`, `internal/query/status.go`, `internal/cli/callees.go`, `internal/cli/callers.go`, `internal/cli/explore.go`, `internal/cli/files.go`, `internal/cli/impact.go`, `internal/cli/node.go`, `internal/cli/search.go`, `internal/cli/status.go`, `internal/mcp/tools.go`, `internal/query/engine_worktree_test.go`, `internal/query/files_status_test.go`, `internal/query/status_staleness_test.go`, `testdata/golden/golden_parity_test.go`
**Commit:** 9cfbb4f
**Applied fix:** `Engine.WorktreeMismatch` and `Engine.Status` now take a `context.Context` parameter and thread it through to `gitmeta.CachingDetector.Detect` instead of discarding the caller's context. All 6 MCP handlers now pass their real `ctx`; all 8 CLI commands pass `cmd.Context()` (mirroring the existing `cmd.Context()` idiom already used in `daemon.go`). Test call sites updated to pass `context.Background()`.

### WR-02: `CachingDetector`'s map grows without bound

**Files modified:** `internal/gitmeta/cache.go`, `internal/gitmeta/cache_test.go`
**Commit:** c09982d
**Applied fix:** Added a `maxCacheEntries` (1024) bound — once reached, the next `Detect` call resets the map rather than growing further. Also added an `os.Stat` short-circuit: a `startPath` that isn't an existing, statable directory returns `nil` immediately, before touching the cache at all (a pure short-circuit — gate 1 of `DetectIndexMismatch` would return `""` for it anyway). Added two new regression tests: `TestCachingDetectorRejectsNonexistentStartPath` and `TestCachingDetectorBounded`.

### WR-03: Gate 4 failed open on a degraded git call

**Files modified:** `internal/gitmeta/detect.go`
**Commit:** ad51fd6
**Applied fix:** Verified TS's actual source (`sync/worktree.js`) genuinely has this same fail-open bug in its own gate 4. Rather than replicate it, gate 4's condition was flipped to fail closed: `worktreeCommon == "" || indexCommon == "" || worktreeCommon != indexCommon` now degrades to "no mismatch" whenever either `CommonDir` call fails, consistent with gates 1-3's and `worktree.go`'s own documented "best-effort, never a false positive" contract. Documented as a deliberate, intentional D-02 divergence from TS in the code comment, with rationale. All existing fixture tests (`TestFixtureVerdicts`, including the submodule/nested-clone rows that exercise gate 4's normal successful path) still pass unchanged.

### WR-04: `codegraph query`/`affected` had no worktree notice

**Files modified:** `internal/cli/query.go`, `internal/cli/affected.go`, `internal/cli/notice_test.go`
**Commit:** 1dd8fb4
**Applied fix:** Added the same gated compact-notice print used by `search.go` (strictly after the `--json` early return, inside the human-output branch) to both `query.go` and `affected.go`. Extended `noticeCommandCases` from 7 to 9 rows and added `query`/`affected` to `TestNoticeSuppressedInJSON`'s case list.

### WR-05: CLI `explore`/`node` printed the notice before the query ran

**Files modified:** `internal/cli/explore.go`, `internal/cli/node.go`, `internal/cli/notice_test.go`
**Commit:** 0410710
**Applied fix:** Moved the notice print in both commands to after the engine call succeeds, matching the other 6 non-status CLI commands. Added a new regression test, `TestNoticeNotEmittedOnQueryFailure`, that drives a failing `node`/`explore` call against a real worktree-mismatch fixture and asserts no bare notice glyph reaches stdout.

## Skipped Issues

None — all 7 in-scope findings were fixed.

---

_Fixed: 2026-07-16T01:23:21Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
