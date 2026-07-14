---
phase: 03-query-engine-mcp-server
plan: 08
subsystem: cli
tags: [cobra, mcp, mark3labs-mcp-go, query-engine, stdio]

# Dependency graph
requires:
  - phase: 03-query-engine-mcp-server (03-03..03-06)
    provides: internal/query.Engine (Query/Search/Callers/Callees/Impact/Affected/Files/Status/Node/Explore) over a read-only GraphStore.Snapshot()
  - phase: 03-query-engine-mcp-server (03-07)
    provides: internal/mcp.BuildServer + ParseAllowlist/WarnUnknownToolsTo — the stdio MCP server with startup-time tool gating
provides:
  - All 11 user-facing commands (query, node, search, callers, callees, impact, affected, files, status, explore, serve) wired into the codegraph Cobra tree
  - codegraph serve --mcp launching the stdio MCP server with CODEGRAPH_MCP_TOOLS allowlist gating and MCP-03 zero-tools-when-uninitialized behavior
  - Black-box execCmd integration tests for every command against a real indexed fixture
affects: [phase-4 (sync/status reconciliation extends status), phase-6 (install/uninstall agent wiring extends serve), golden-parity-test (future plan diffs these commands' --json output against testdata/golden)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Thin Cobra command: resolve -p/--path (default cwd) via resolveStartPath, query.OpenAt for a fresh snapshot+closer, delegate to Engine method, render via cmd.OutOrStdout() — never os.Stdout directly"
    - "--json toggles a dedicated query.Marshal*JSON helper (or raw json.Marshal for Search, which has no dedicated marshaler) on the eight structured commands; node/explore always emit markdown (D-01b)"
    - "serve resolves .codegraph/ via query.ResolveCodegraphDir but tolerates ErrNotInitialized (unlike every other command) so the MCP server still starts with zero tools (MCP-03)"

key-files:
  created:
    - internal/cli/query.go
    - internal/cli/search.go
    - internal/cli/callers.go
    - internal/cli/callees.go
    - internal/cli/impact.go
    - internal/cli/affected.go
    - internal/cli/files.go
    - internal/cli/status.go
    - internal/cli/node.go
    - internal/cli/explore.go
    - internal/cli/serve.go
    - internal/cli/query_cli_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "serve requires an explicit --mcp flag (errors if omitted) even though stdio is the only v1 transport — makes the transport selection explicit for the future HTTP/SSE (v2 SERVER-01) addition"
  - "search's --json uses a direct encoding/json.Marshal([]query.Location) call, matching internal/mcp's companionHandler 'search' branch, since no dedicated MarshalSearchJSON helper exists in internal/query"
  - "files' non-JSON default output renders the tree format as indented plain text (printFileTree) when --format tree is set, flat one-line-per-file otherwise — no golden oracle exists for files' human output (D-07a)"
  - "affected requires at least one positional file argument (cobra.MinimumNArgs(1)) — an empty changed-file set has no useful query-time-derivation output"

requirements-completed: [QRY-01, QRY-02, QRY-03, QRY-04, QRY-05, QRY-06, QRY-07, QRY-08, QRY-09, MCP-01]

coverage:
  - id: D1
    description: "query/search/callers/callees/impact/affected/files/status commands wired into root.go with -p/--path, --json, and their per-command flags (--kind/--limit/--depth/--pattern/--filter/--format)"
    requirement: "QRY-01, QRY-04, QRY-05, QRY-06, QRY-07, QRY-09"
    verification:
      - kind: integration
        ref: "internal/cli/query_cli_test.go#TestQueryCmd,TestSearchCmd,TestCallersCalleesCmd,TestImpactCmd,TestAffectedCmd,TestFilesCmd,TestStatusCmd"
        status: pass
    human_judgment: false
  - id: D2
    description: "node/explore markdown commands (QRY-02, QRY-08) — symbol detail / file-mode read, and the flagship one-round-trip verbatim-source + blast-radius explore"
    requirement: "QRY-02, QRY-03, QRY-08"
    verification:
      - kind: integration
        ref: "internal/cli/query_cli_test.go#TestNodeCmd,TestExploreCmd"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph serve --mcp runs the stdio MCP server with default/allowlist/zero-tool visibility, verified live end-to-end (raw JSON-RPC over stdio) against the MCP-01/02/03 scenarios"
    requirement: "MCP-01"
    verification:
      - kind: manual_procedural
        ref: "Live smoke test: initialize + tools/list against `codegraph serve --mcp` (default -> [codegraph_explore]; CODEGRAPH_MCP_TOOLS=node,status,bogus -> [codegraph_explore,codegraph_node,codegraph_status] + stderr warning for bogus; no .codegraph/ -> []); tools/call codegraph_explore returned the same markdown shape as CLI explore"
        status: pass
    human_judgment: true
    rationale: "Checkpoint 3 (blocking-human-verify) is pre-approved under --auto per the executor's checkpoint_autonomy_note. The live stdio MCP handshake itself was exercised end-to-end via raw JSON-RPC (not simulated) and all four how-to-verify scenarios passed, but the plan designates this a human-verify gate — a human connecting a real MCP client (e.g. Claude Code) remains the recommended manual smoke test before relying on this in production."

# Metrics
duration: 25min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 8: Query & MCP Command Surface Summary

**Wired all 11 query/serve commands into the codegraph Cobra tree (query, node, search, callers, callees, impact, affected, files, status, explore, serve --mcp), each a thin delegate to the 03-03..03-07 internal/query.Engine and internal/mcp server, with black-box execCmd tests and a live-verified stdio MCP handshake.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-07-11T14:59:45Z
- **Completed:** 2026-07-11T15:07:01Z
- **Tasks:** 3 (2 execute + 1 pre-approved checkpoint)
- **Files modified:** 13 (11 new command files + query_cli_test.go + root.go)

## Accomplishments

- Eight structured query commands (`query`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`) support `-p/--path` and `--json`, plus their documented per-command flags (`--kind`, `--limit`, `--depth`, `--pattern`, `--filter`, `--format`)
- Two flagship markdown commands (`node`, `explore`) reproduce the golden `node.json`/`explore.json` templates byte-for-byte via `internal/query.RenderNode`/`RenderExplore` — no `--json` flag, per D-01b
- `serve --mcp` runs the 03-07 stdio MCP server: resolves `.codegraph/` via `query.ResolveCodegraphDir` but does NOT error when absent (MCP-03's zero-tools case), parses `CODEGRAPH_MCP_TOOLS` via `mcp.ParseAllowlist`, warns on unknown names via `cmd.ErrOrStderr()` (never stdout — the JSON-RPC transport), and delegates to `server.ServeStdio`
- All 11 commands verified present in `codegraph --help`
- Live end-to-end MCP verification (raw JSON-RPC over stdio, not a unit test): default visibility (`codegraph_explore` only), allowlist visibility (`+codegraph_node`, `+codegraph_status`, `bogus` warned-not-crashed), zero-tools with no index, and `codegraph_explore` tool-call output matching the CLI's `explore` markdown

## Task Commits

Each task was committed atomically:

1. **Task 1: Structured query commands + root wiring + integration tests** - `6b7388c` (feat)
2. **Task 2: node/explore markdown commands + serve command** - `54daf48` (feat)
3. **Task 3: Human verifies live MCP tool visibility over stdio** - pre-approved under `--auto` (checkpoint_autonomy_note); live-verified via raw JSON-RPC smoke test, no code change/commit for this task

**Plan metadata:** (this commit) `docs(03-08): complete query & MCP command surface plan`

## Files Created/Modified

- `internal/cli/query.go` - `codegraph query <term>`; also hosts `resolveStartPath`/`writeJSONLine`, the shared `-p/--path` resolution and `--json` write helpers every other command file uses
- `internal/cli/search.go` - `codegraph search <term>` (locations-only projection)
- `internal/cli/callers.go` - `codegraph callers <symbol>`
- `internal/cli/callees.go` - `codegraph callees <symbol>`
- `internal/cli/impact.go` - `codegraph impact <symbol> --depth`
- `internal/cli/affected.go` - `codegraph affected <files...>`
- `internal/cli/files.go` - `codegraph files --pattern --filter --depth --format`
- `internal/cli/status.go` - `codegraph status`
- `internal/cli/node.go` - `codegraph node [symbol] -f <file>`, markdown-only
- `internal/cli/explore.go` - `codegraph explore <query> --max-files`, markdown-only, the flagship one-round-trip command
- `internal/cli/serve.go` - `codegraph serve --mcp`, wires `internal/mcp.BuildServer` + `server.ServeStdio`
- `internal/cli/query_cli_test.go` - black-box `execCmd`-driven integration tests for all 11 commands against a real indexed `gofixture` (main -> pkgb.Run -> pkga.Alpha -> pkga.helper call chain)
- `internal/cli/root.go` - registers all 11 new commands on the root Cobra tree

## Decisions Made

- `serve` requires an explicit `--mcp` flag (errors with a clear message if omitted), even though stdio is the only transport v1 ships — makes the future HTTP/SSE (v2, SERVER-01) transport selection explicit from day one rather than retrofitting a flag later
- `search --json` marshals `[]query.Location` directly via `encoding/json.Marshal` (no dedicated `MarshalSearchJSON` exists in `internal/query` — matches `internal/mcp`'s own `companionHandler` "search" branch, keeping both front-ends' JSON-shaping code paths identical)
- `files`' human-readable (non-`--json`) output renders `--format tree` as indented plain text via a small `printFileTree` helper; flat format is one line per file — no golden oracle constrains this shape (D-07a)
- `affected` requires at least one positional file argument (`cobra.MinimumNArgs(1)`) since an empty changed-file set produces no useful query-time-derived output

## Deviations from Plan

None — plan executed exactly as written. All 11 commands, root wiring, and integration tests match the plan's task descriptions and acceptance criteria.

## Auth Gates Encountered

None.

## Issues Encountered

None. `go build ./...`, `go test ./internal/cli/... -count=1`, and `go test ./...` (whole suite) were all green after each task; `go run ./cmd/codegraph --help` confirmed all 11 commands are registered.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 3's full success-criteria set (all documented commands runnable, `explore` one-round-trip, `serve --mcp` MCP-01/02/03 tool-visibility behavior) is now demonstrably wired end-to-end
- Remaining Phase 3 work (per ROADMAP): a golden-corpus parity test (`testdata/golden/golden_parity_test.go`) diffing these commands' `--json`/markdown output against `testdata/golden/corpus/weft-go/*.json` (MCP-04) — not part of this plan's scope, next in the phase's plan sequence
- No blockers for that follow-on parity work: every command here already delegates through the same `internal/query.Engine` methods and `Marshal*JSON`/`Render*` formatters the golden shapes were designed against (D-05/D-05a/D-05b)

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

All 12 created/modified files verified present on disk; all 3 commit hashes (6b7388c, 54daf48, 20ce806) verified present in git log.
