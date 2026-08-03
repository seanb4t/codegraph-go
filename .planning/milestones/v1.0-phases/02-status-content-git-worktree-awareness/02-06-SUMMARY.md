---
phase: 02-status-content-git-worktree-awareness
plan: 06
subsystem: mcp
tags: [mcp, markdown, worktree, gitmeta, surf-06, tdd]

requires:
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-03's 5 Render*Markdown functions (RenderCallersMarkdown/RenderCalleesMarkdown/RenderImpactMarkdown/RenderSearchMarkdown/RenderFilesMarkdown) — this plan's 5 tools.go call-site swaps"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-04's Engine.WorktreeMismatch()/UseDetector()/gitmeta.CachingDetector and query.WorktreeNotice/WorktreeWarningBlockquote — this plan's server-scoped cache and notice injection"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-05's RenderStatusMarkdown (already embeds the blockquote via r.WorktreeMismatch) — this plan's 6th call-site swap"
provides:
  - "internal/mcp/tools.go: all 8 MCP read tools (explore/node/search/callers/callees/impact/files/status) return markdown"
  - "internal/mcp: one gitmeta.CachingDetector constructed per server (BuildServer) and shared by every handler's fresh per-call Engine"
  - "7 non-status MCP read tools carry the compact worktree notice on mismatch, nothing on a clean tree or an error result"
affects: []

tech-stack:
  added: []
  patterns:
    - "Server-lifetime closure parameter (gitmeta.CachingDetector) added alongside the existing defaultPath closure convention in exploreHandler/companionHandler"
    - "Success-path-only string prefix (query.WorktreeNotice(eng.WorktreeMismatch()) + out) applied after every branch's error returns already resolved — no redundant isError check needed"

key-files:
  created:
    - internal/mcp/markdown_test.go
  modified:
    - internal/mcp/tools.go
    - internal/mcp/server.go

key-decisions:
  - "server_test.go needed no edits: BuildServer's public 3-arg signature (hasIndex, allowlist, repoPath) is unchanged — only exploreHandler/companionHandler's internal signatures gained a detector parameter, which is constructed and closed over entirely inside BuildServer"
  - "codegraph_status's compact-notice exclusion required no extra code: RenderStatusMarkdown (02-05) already consumes StatusResult.WorktreeMismatch and embeds WorktreeWarningBlockquote internally, so the status branch simply never calls query.WorktreeNotice — mirroring TS's withWorktreeNotice exclusion by construction, not by a special-case guard"
  - "search's inline json.Marshal(locs) was removed outright (not replaced with an error-returning wrapper) since query.RenderSearchMarkdown returns a plain string with no error to handle — the now-dead err-handling branch was deleted, and the resulting unused encoding/json import was dropped"

requirements-completed: [SURF-06, WORK-02]

coverage:
  - id: D1
    description: "Each of the 5 SURF-06 MCP tools (callers/callees/impact/search/files), driven through the real CallTool path against an indexed fixture, returns text that both fails json.Unmarshal and contains its expected markdown marker"
    requirement: SURF-06
    verification:
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestMarkdownOutput"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph_status returns markdown (fails json.Unmarshal, contains **CodeGraph Status**) — D-17, the 6th JSON-to-markdown call site"
    requirement: SURF-06
    verification:
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestStatusMarkdownOutput"
        status: pass
    human_judgment: false
  - id: D3
    description: "All 7 non-status MCP read tools prefix the compact worktree notice on a real .claude/worktrees/ mismatch fixture; codegraph_status carries the blockquoted verbose warning instead, never the compact form; a clean (non-worktree) fixture adds no notice to any of the 8 tools; an error result (missing required arg) is never prefixed"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestWorktreeNoticeOnMismatch"
        status: pass
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestNoWorktreeNoticeOnCleanTree"
        status: pass
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestWorktreeNoticeNotAppliedOnError"
        status: pass
    human_judgment: false
  - id: D4
    description: "Exactly one gitmeta.CachingDetector is constructed per server (BuildServer) and shared by every handler's fresh per-call Engine via UseDetector, so repeated tool calls against one server produce a consistent (cached) worktree verdict"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/mcp/markdown_test.go#TestWorktreeNoticeConsistentAcrossCalls"
        status: pass
      - kind: unit
        ref: "rg -v '^\\s*//' internal/mcp/server.go | rg -c NewCachingDetector (returns 1)"
        status: pass
    human_judgment: false
  - id: D5
    description: "No Marshal*JSON function body was modified; the CLI --json path, its regression-guard tests, and the golden parity oracle are all unaffected"
    requirement: SURF-06
    verification:
      - kind: unit
        ref: "git diff internal/query/traverse.go internal/query/files.go internal/query/search.go internal/query/status.go internal/cli/query_cli_test.go go.mod go.sum (all empty)"
        status: pass
      - kind: integration
        ref: "go test ./internal/cli/... -run \"TestSearchCmd|TestCallersCalleesCmd|TestImpactCmd|TestFilesCmd\" (pass); go test ./testdata/golden/... (pass)"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 6: MCP Markdown Wiring + Server-Scoped Worktree Notice Summary

**Closed the phase's single biggest blind spot — zero prior tests asserted the MCP success payload of callers/callees/impact/search/files — with a mandatory RED test, then swapped all 6 remaining JSON call sites in `internal/mcp/tools.go` to the sibling `Render*Markdown` functions and wired one server-scoped `gitmeta.CachingDetector` so 7 of 8 MCP read tools carry the compact worktree notice.**

## Performance

- **Duration:** 25 min
- **Tasks:** 3 (TDD RED / GREEN / GREEN)
- **Files modified:** 3 (1 new, 2 modified)

## Accomplishments

- `internal/mcp/markdown_test.go` (new): closes the total pre-phase MCP-surface blind spot for `callers`/`callees`/`impact`/`search`/`files`, and the error-path-only gap for `status`. Every SURF-06 row asserts BOTH `json.Unmarshal` failure AND a positive markdown marker, driven through the real `CallTool` path against an indexed fixture — not the renderer in isolation.
- `internal/mcp/tools.go`: all six remaining JSON call sites (`search`'s inline `json.Marshal`, `callers`/`callees`/`impact`/`files`'s `Marshal*JSON`, and `status`'s `MarshalStatusJSON`) now call the matching `query.Render*Markdown` / `query.RenderStatusMarkdown` function. Zero `Marshal*JSON` bodies were touched.
- `internal/mcp/server.go`: `BuildServer` constructs exactly one `gitmeta.NewCachingDetector()` and closes it over `exploreHandler`/`companionHandler`, following the existing `defaultPath` closure-parameter convention. `openEngine` now calls `eng.UseDetector(detector)` before returning, so every fresh per-call `Engine` shares the one server-scoped cache (D-13).
- `explore`/`node`/`search`/`callers`/`callees`/`impact`/`files` now prefix `query.WorktreeNotice(eng.WorktreeMismatch())` on their success return only. `codegraph_status` is excluded by construction: `RenderStatusMarkdown` (02-05) already embeds the verbose blockquote from `StatusResult.WorktreeMismatch`.
- `confineToRepoRoot` (CR-02) is unmoved and still runs before `query.OpenAt` inside `openEngine` — verified both by inspection and by `TestWorktreeNoticeOnMismatch`'s fixture, which roots the server at the worktree specifically to satisfy (not bypass) that check.

## Task Commits

Each task was committed atomically (TDD RED/GREEN):

1. **Task 1: RED — MCP success-payload markdown assertions** — `b654385` (test)
2. **Task 2: GREEN — six call-site swaps to markdown (SURF-06, D-17)** — `d6204cd` (feat)
3. **Task 3: GREEN — server-scoped CachingDetector + worktree notice on 7 tools (WORK-02, D-12, D-13)** — `13892fd` (feat)

No REFACTOR commit was needed — each GREEN implementation matched Task 1's target contract with no follow-up cleanup.

## Files Created/Modified

- `internal/mcp/markdown_test.go` — 6 new tests: `TestMarkdownOutput` (table over the 5 SURF-06 tools), `TestStatusMarkdownOutput` (D-17), `TestNoWorktreeNoticeOnCleanTree`, `TestWorktreeNoticeOnMismatch`, `TestWorktreeNoticeNotAppliedOnError`, `TestWorktreeNoticeConsistentAcrossCalls`. Includes a local `mcpWorktreeMismatchFixture` (a real linked worktree at `.claude/worktrees/probe`, built via `os/exec` git, mirroring `internal/query/engine_worktree_test.go`'s fixture since Go test helpers are not importable across packages) and comments codifying why `TestExploreCLIMatchesMCP`/`TestNodeCLIMatchesMCP` must not be extended to the 5 SURF-06 tools.
- `internal/mcp/tools.go` — `openEngine`/`exploreHandler`/`companionHandler` all gained a `*gitmeta.CachingDetector` parameter; the six call-site swaps; `query.WorktreeNotice(...)` prefix on 6 companion branches plus `exploreHandler`; `companionHandler`'s doc comment rewritten to record the `Marshal*JSON`-CLI-only / `Render*Markdown`-MCP-only asymmetry; dropped the now-unused `encoding/json` import.
- `internal/mcp/server.go` — `BuildServer` constructs one `gitmeta.CachingDetector`, documents why it cannot live on `Engine` (`openEngine` rebuilds fresh per call), and passes it to every handler constructor.

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:
- `server_test.go` required no edits — `BuildServer`'s public signature is unchanged; the detector is entirely internal plumbing.
- `codegraph_status`'s exclusion from the compact notice needed no special-case guard — it falls out structurally from `RenderStatusMarkdown` already consuming the live `WorktreeMismatch` field and never calling `query.WorktreeNotice`.
- `search`'s dead `err` variable from the removed `json.Marshal` call was deleted along with the marshal call itself, not left as a vestigial always-nil check.

## Reachability Verification

The plan's manual reachability check ("from a `.claude/worktrees/<name>/` worktree ... a `codegraph_explore` call's text starts with the compact notice, and `codegraph_status` contains `**Database size:**`") is covered by `TestWorktreeNoticeOnMismatch`'s automated fixture, which builds a real linked worktree via `git worktree add` and drives both `codegraph_explore` and `codegraph_status` through the in-process `CallTool` path against it — the same code path a live MCP client would exercise. No separate manual step was run against this repo's own working tree, to avoid mutating its live `.codegraph/` index or git worktree state; the automated fixture is behaviorally equivalent and is the mechanism the plan's own Task 1 required (a real `.claude/worktrees/` layout, not a faked one).

## Deviations from Plan

None — plan executed exactly as written. `confineToRepoRoot`'s interaction with the mismatch fixture required no workaround: rooting the test server at the worktree path (rather than the main checkout) was sufficient and is what the plan's Task 1 `<action>` anticipated.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Final Signatures

- `openEngine(req mcp.CallToolRequest, defaultPath string, detector *gitmeta.CachingDetector) (*query.Engine, func() error, error)`
- `exploreHandler(defaultPath string, detector *gitmeta.CachingDetector) server.ToolHandlerFunc`
- `companionHandler(name, defaultPath string, detector *gitmeta.CachingDetector) server.ToolHandlerFunc`
- `BuildServer(hasIndex bool, allowlist map[string]bool, repoPath string) *server.MCPServer` — unchanged public signature; constructs and closes over one `*gitmeta.CachingDetector` internally.

## Next Phase Readiness

- SURF-06 and WORK-02 are both fully wired end-to-end on the MCP surface; D-13's server-scoped cache and D-17's status-specific renderer are both live.
- `go test ./...` and `go test ./testdata/golden/...` are fully green.
- `go.mod`/`go.sum` are byte-identical to before this plan (no new dependencies).
- No blockers for plan 02-07 (CLI wiring, out of this plan's scope).

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*

## Self-Check: PASSED

All 4 referenced files verified present on disk; all 4 commits (b654385, d6204cd, 13892fd, ebe6e77) verified present in git log.
