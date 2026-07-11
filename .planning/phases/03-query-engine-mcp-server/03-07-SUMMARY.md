---
phase: 03-query-engine-mcp-server
plan: 07
subsystem: api
tags: [mcp, mark3labs/mcp-go, stdio, tool-gating]

# Dependency graph
requires:
  - phase: 03-05
    provides: "internal/query.Engine's Files/Status methods + Marshal*JSON formatters"
  - phase: 03-06
    provides: "internal/query.Engine's Node/Explore markdown-rendering methods"
provides:
  - "internal/mcp package: stdio MCP server with startup-time conditional tool registration"
  - "BuildServer(hasIndex, allowlist, repoPath) — the tool-visibility gate MCP-01/02/03 depend on"
  - "ParseAllowlist/WarnUnknownToolsTo — CODEGRAPH_MCP_TOOLS parsing with never-abort unknown-name handling"
  - "8 MCP tool schemas + handlers (codegraph_explore + 7 companions) delegating to the shared query engine"
affects: [03-08, 03-09]

# Tech tracking
tech-stack:
  added: ["github.com/mark3labs/mcp-go v0.56.0"]
  patterns:
    - "Startup-time conditional AddTool registration (no per-session dynamic tool machinery)"
    - "Fresh query.OpenAt snapshot per tool call via a single openEngine() seam (D-02/D-08b)"
    - "In-process mcp-go client (client.NewInProcessClient) for tool-registration tests without a live stdio transport"

key-files:
  created:
    - internal/mcp/server.go
    - internal/mcp/tools.go
    - internal/mcp/server_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Manually promoted github.com/mark3labs/mcp-go to go.mod's direct require block instead of running go mod tidy, per established project convention"
  - "ParseAllowlist and WarnUnknownToolsTo are split: parsing is pure/testable, warning-writing takes an io.Writer (WarnUnknownToolsTo) so tests can assert stderr content without capturing the real os.Stderr"
  - "search's companion handler marshals []query.Location directly via encoding/json.Marshal (no MarshalSearchJSON exists in internal/query, unlike its sibling commands) — Location's own json tags are the shape, so this is not a second rendering path"
  - "companionTool/companionHandler panic on an unrecognized name — the only callers are BuildServer's fixed companionNames loop, so an unknown name reaching them would be a programming error, not user input"

patterns-established:
  - "internal/mcp imports only internal/query + mcp-go — never internal/graphstore/pebble — preserving the archtest boundary"
  - "Every tool handler: parse args -> openEngine (fresh snapshot) -> delegate to Engine method -> delegate to query Marshal*JSON/markdown output -> mcp.NewToolResultText"

requirements-completed: [MCP-01, MCP-02, MCP-03]

coverage:
  - id: D1
    description: "codegraph_explore is the only default-visible tool when an index is present and the allowlist is empty"
    requirement: "MCP-01"
    verification:
      - kind: unit
        ref: "internal/mcp/server_test.go#TestDefaultToolVisibility"
        status: pass
    human_judgment: false
  - id: D2
    description: "The 7 companion tools register only when named in CODEGRAPH_MCP_TOOLS; unknown names are ignored with a stderr warning and never abort startup"
    requirement: "MCP-02"
    verification:
      - kind: unit
        ref: "internal/mcp/server_test.go#TestAllowlist"
        status: pass
    human_judgment: false
  - id: D3
    description: "Zero tools are advertised when no .codegraph/ resolves, and the server still constructs successfully"
    requirement: "MCP-03"
    verification:
      - kind: unit
        ref: "internal/mcp/server_test.go#TestNoIndexZeroTools"
        status: pass
    human_judgment: false
  - id: D4
    description: "Tool handlers delegate to the shared internal/query.Engine + formatters on a fresh snapshot per call (D-08b) — proved for codegraph_explore; the remaining 7 handlers follow the identical openEngine()-delegate-marshal shape"
    verification:
      - kind: unit
        ref: "internal/mcp/server_test.go#TestExploreHandlerDelegatesToEngine"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live end-to-end stdio handshake against a real MCP client (not the in-process test client)"
    verification: []
    human_judgment: true
    rationale: "Not scriptable as a Go unit test (RESEARCH §Environment Availability) — covered by a human-verify checkpoint in 03-08 once `serve --mcp` is wired to a CLI entrypoint"

# Metrics
duration: 15min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 07: Stdio MCP Server + Tool Gating Summary

**Stdio MCP server on mark3labs/mcp-go v0.56.0 with startup-time conditional tool registration: codegraph_explore is the only default-visible tool, 7 companions gate on CODEGRAPH_MCP_TOOLS, and every handler delegates to the shared internal/query.Engine on a fresh snapshot per call.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-11T10:47:00Z (immediately after 03-06's completion commit)
- **Completed:** 2026-07-11T10:55:42Z
- **Tasks:** 2 (Task 1 scaffold, Task 2 RED+GREEN)
- **Files modified:** 5 (go.mod, go.sum, internal/mcp/server.go, internal/mcp/tools.go, internal/mcp/server_test.go)

## Accomplishments

- Added `github.com/mark3labs/mcp-go` v0.56.0 as a direct dependency (pre-audited OK) — the phase's only new external dependency
- `internal/mcp.BuildServer(hasIndex, allowlist, repoPath)` implements D-08a's conditional `AddTool` gating: zero tools when `hasIndex` is false (MCP-03), `codegraph_explore` only when `hasIndex` is true and the allowlist is empty (MCP-01), plus the 7 companions (`node`/`search`/`callers`/`callees`/`impact`/`files`/`status`) when named in `CODEGRAPH_MCP_TOOLS` (MCP-02)
- `ParseAllowlist`/`WarnUnknownToolsTo` parse the comma-separated env var and warn on unrecognized names without ever aborting startup or writing to stdout (reserved for JSON-RPC framing)
- 8 tool schemas (`internal/mcp/tools.go`) mirroring the eventual CLI flags (query/path/max_files, symbol/file, limit/depth/kind, pattern/filter/format), each handler opening a fresh `query.OpenAt` snapshot and delegating to the exact `internal/query.Engine` method + `Marshal*JSON`/markdown formatter the CLI will use — no second rendering path
- Test suite uses mcp-go's in-process client (`client.NewInProcessClient`) to introspect registered tool names and exercise a real tool call end-to-end, without a live stdio transport

## Task Commits

1. **Task 1: Add mcp-go dependency and scaffold internal/mcp** - `fa6bdcd` (build)
2. **Task 2 RED: failing tool visibility gating tests** - `b942173` (test)
2. **Task 2 GREEN: tool gating + engine-delegating handlers** - `0a8305e` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `go.mod` / `go.sum` - `github.com/mark3labs/mcp-go v0.56.0` promoted to the direct require block
- `internal/mcp/server.go` - `BuildServer`, `ParseAllowlist`, `WarnUnknownToolsTo`, `companionNames`
- `internal/mcp/tools.go` - `exploreTool`/`exploreHandler`, `companionTool`/`companionHandler` for all 7 companions, `openEngine` (the shared fresh-snapshot-per-call seam)
- `internal/mcp/server_test.go` - `TestDefaultToolVisibility`, `TestAllowlist`, `TestNoIndexZeroTools`, `TestExploreHandlerDelegatesToEngine`, plus `copyFixture`/`indexFixture` test scaffolding mirroring `internal/query/engine_test.go`

## Decisions Made

- Manually promoted `mark3labs/mcp-go` from indirect to direct in `go.mod` instead of running a bare `go mod tidy` (established project convention since Phase 1)
- Split allowlist parsing (`ParseAllowlist`, pure) from stderr diagnostics (`WarnUnknownToolsTo`, takes an `io.Writer`) so the unknown-name warning behavior is directly unit-testable without capturing the process's real `os.Stderr`
- `search`'s companion handler marshals `[]query.Location` via a direct `encoding/json.Marshal` call rather than a new `query.MarshalSearchJSON` helper — no such helper exists in `internal/query` (unlike its sibling commands), and `Location`'s JSON shape is already fully owned by `internal/query`'s struct tags, so this is a thin stdlib encode, not a second rendering path
- `companionTool`/`companionHandler` panic on an unrecognized name — the only caller is `BuildServer`'s fixed `companionNames` loop, so reaching the default branch would be a programming bug, not attacker-controlled input

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- mcp-go v0.56.0's package-level `go get` initially landed the module as `// indirect` since nothing imported it yet; a follow-up `go get` for the `mcp` and `server` subpackages plus a manual `go.mod` edit (matching this project's existing convention) resolved it to a clean direct dependency.
- Several mcp-go API names differ slightly from the RESEARCH.md example snippets against the pinned v0.56.0 release: `client.NewInProcessClient` returns `(*Client, error)` (not a bare `*Client`), `Client.Initialize` requires an explicit `mcp.InitializeRequest{ProtocolVersion, ClientInfo}` argument, and `CallToolRequest.Params` is typed `mcp.CallToolParams` (not `CallToolRequestParams`). Verified against the vendored module source (`$GOPATH/pkg/mod/github.com/mark3labs/mcp-go@v0.56.0`) rather than guessing from the RESEARCH snippets — all four tests pass against the actual v0.56.0 API.

## Next Phase Readiness

- `internal/mcp.BuildServer` is ready for 03-08 to wire into a `codegraph serve --mcp` Cobra command, reading `CODEGRAPH_MCP_TOOLS` from the environment and resolving `hasIndex`/`repoPath` from the CLI's existing path-resolution idiom
- The live stdio handshake against a real external MCP client (not the in-process test client used here) remains a human-verify checkpoint for 03-08, per RESEARCH §Environment Availability
- 03-09's golden parity harness can call the same `internal/query.Engine` methods `internal/mcp`'s handlers call — no additional MCP-specific parity surface to reconcile

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*
