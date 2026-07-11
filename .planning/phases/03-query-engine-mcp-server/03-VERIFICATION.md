---
phase: 03-query-engine-mcp-server
verified: 2026-07-11T16:36:13Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 3: Query Engine & MCP Server Verification Report

**Phase Goal:** Agents and users can interrogate the frozen Phase-2 Go graph through the full read-only query command suite (query, node, search, callers, callees, impact, affected, files, status, explore) and a parity stdio MCP server (`codegraph serve --mcp`) whose output shapes match TS CodeGraph v1.3.1.
**Verified:** 2026-07-11T16:36:13Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All 9 query commands (`query`/`node`/`search`/`callers`/`callees`/`impact`/`affected`/`files`/`status`) run with documented flags (`--kind`/`--limit`/`--json`/`--depth`/`--pattern`/`--filter`/`--format`/`--max-files`) and return correct results | ✓ VERIFIED | All 11 commands registered in `internal/cli/root.go:38-40`; per-command flags confirmed by direct inspection of each `internal/cli/*.go` file (grep dump above); `internal/cli/query_cli_test.go` black-box `execCmd` tests pass (`go test ./internal/cli/...` green); engine methods (`Query`/`Search`/`Callers`/`Callees`/`Impact`/`Affected`/`Files`/`Status`) exist in `internal/query/{search,traverse,files,status}.go` and are unit-tested |
| 2 | `codegraph explore <query>` returns verbatim line-numbered source grouped by file + call paths + blast-radius summary in one round trip | ✓ VERIFIED | Live-executed: built the binary, indexed a fresh fixture repo, ran `codegraph explore main` via raw stdio MCP JSON-RPC — response contained `**Exploration: main**`, `Found 2 symbols across 1 file`, `**Blast radius**` bullets, and the `**Source Code**` verbatim-source disclaimer paragraph, matching `testdata/golden/corpus/weft-go/explore.json`'s template (D-05a). `internal/query/render_markdown_test.go` (`TestExplore`) pins the byte-shape |
| 3 | An agent connecting to `codegraph serve --mcp` sees `codegraph_explore` as the only default tool; the 7 companions appear only via `CODEGRAPH_MCP_TOOLS`; zero tools when no `.codegraph/` exists | ✓ VERIFIED | Live-executed three scenarios via raw stdio JSON-RPC against a real built binary (not simulated): (a) default → `tools/list` returned exactly `[codegraph_explore]`; (b) `CODEGRAPH_MCP_TOOLS=node,status,bogus` → returned `[codegraph_explore, codegraph_node, codegraph_status]` plus a stderr warning `unknown tool name "bogus"...ignoring`; (c) no `.codegraph/` present → server completed `initialize` and returned `tools: []`. All three match `internal/mcp/server.go`'s `BuildServer`/`ParseAllowlist` logic and are additionally pinned by `TestDefaultToolVisibility`/`TestAllowlist`/`TestNoIndexZeroTools` (all pass) |
| 4 | MCP/CLI output shapes match TS v1.3.1, verified against the golden-output corpus | ✓ VERIFIED | `testdata/golden/golden_parity_test.go` exists, resolves the real pinned `../weft` checkout at commit `f89ae3ea4e4c37509f7302fd4e37986212a72079` (present on this machine), indexes it via the real `indexer.Run` pipeline, and diffs `status/query/callers/callees/impact/explore/node` against the golden fixtures — `go test ./testdata/golden/ -run TestGoldenParity -v` PASSED for all 7 subtests (one documented informational `t.Logf` on `impact(mergeStyle, depth=2)` node/edge-count divergence, not a failure, per 03-REVIEW-FIX.md verification notes) |
| 5 | CR-01 (unbounded `--limit` DoS) is fixed | ✓ VERIFIED | `internal/query/search.go:140-141,173-174` and `internal/query/traverse.go:168-169,208-209` apply an unconditional `if n/len(...) > MaxLimit { ... = MaxLimit }` cap independent of whether an explicit `--limit` was given. `TestQueryDefaultCapAtMaxLimit`/`TestSearchDefaultCapAtMaxLimit`/`TestCalleesDefaultCapAtMaxLimit`/`TestCallersDefaultCapAtMaxLimit` all pass. Commit `fdc41ea` present in `git log` |
| 6 | CR-02 (MCP path confinement) is fixed | ✓ VERIFIED | `internal/mcp/tools.go` `confineToRepoRoot`/`openEngine` reject any client-supplied `path` outside the server's configured repo root. Live-verified: an MCP `tools/call codegraph_explore` with `path` pointed at a sibling project (`/Volumes/Code/github.com/seanb4t/codegraph-go`) returned `isError: true` with message `mcp: path "..." is outside the server's configured repo root`; a call with no path override succeeded normally. `TestOpenEnginePathConfinedToRepoRoot` also passes. Commit `6dd5c7b` present in `git log` |
| 7 | Requirements QRY-01…09, MCP-01…04 are all satisfied with no orphans | ✓ VERIFIED | Every plan's `requirements:` frontmatter (03-01…03-09) sums to exactly QRY-01..09 + MCP-01..04; `.planning/REQUIREMENTS.md` lists all 13 IDs as "Phase 3 / Complete" with matching descriptions; no additional Phase-3-mapped ID appears in REQUIREMENTS.md that isn't claimed by a plan |
| 8 | `go build ./...` and `go test ./...` are green | ✓ VERIFIED | Ran directly: `go build ./...` clean, `go vet ./...` clean, `go test ./...` all packages `ok`. `testdata/golden` package (ignored by `./...` per Go's `testdata` convention) tested explicitly: `go test ./testdata/golden/` — `ok` |
| 9 | `internal/query`/`internal/mcp` never bypass the `GraphStore` interface (archtest boundary holds) | ✓ VERIFIED | `go test ./internal/graphstore/archtest/...` — `TestNoPackageBypassesGraphStore` PASSED; confirmed no `internal/query/*.go` or `internal/mcp/*.go` file directly imports `github.com/cockroachdb/pebble/v2` |

**Score:** 9/9 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/graphstore/store.go` + `pebble_store.go` + `iter_test.go` | D-03 `IterateNodes`/`IterateFiles` additive Reader extension | ✓ VERIFIED | Exists, wired, tested; `archtest` still passes |
| `internal/query/engine.go`, `resolve.go`, `validate.go`, `engine_test.go` | Engine foundation, resolver, validation clamps | ✓ VERIFIED | Present; `ValidateKind`/clamp helpers exist and are unit-tested |
| `internal/query/search.go` + `search_test.go` | `Query`/`Search` (QRY-01/03) | ✓ VERIFIED | Present, tested, CR-01/WR-05 fixes applied |
| `internal/query/traverse.go` + `traverse_test.go` | `Callers`/`Callees`/`Impact`/`Affected` (QRY-04/05/06) | ✓ VERIFIED | Present, tested, CR-01/WR-01/WR-04 fixes applied |
| `internal/query/files.go`, `status.go`, `files_status_test.go` | `Files`/`Status` (QRY-07/09) | ✓ VERIFIED | Present, tested; D-05 status key-mapping table documented in-code |
| `internal/query/render_markdown.go`, `node.go`, `explore.go`, `render_markdown_test.go` | `Node`/`Explore` markdown templates (QRY-02/08) | ✓ VERIFIED | Present, byte-shape tested, WR-03 symlink-confinement fix applied |
| `internal/mcp/server.go`, `tools.go`, `server_test.go` | Stdio MCP server + tool gating (MCP-01/02/03) | ✓ VERIFIED | Present, tested, CR-02 fix applied, live-verified |
| `internal/cli/{query,node,search,callers,callees,impact,affected,files,status,explore,serve}.go` + `root.go` + `query_cli_test.go` | 11 Cobra commands wired | ✓ VERIFIED | All 11 registered in root.go; black-box integration tests pass |
| `testdata/golden/golden_parity_test.go` | Golden parity harness (MCP-04) | ✓ VERIFIED | Exists, resolves real weft corpus at pinned commit, runs and passes |
| `go.mod` — `github.com/mark3labs/mcp-go v0.56.0` | MCP dependency | ✓ VERIFIED | Present in go.mod |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/*.go` | `internal/query.Engine` | `query.OpenAt` fresh-snapshot-per-invocation | ✓ WIRED | Confirmed by reading each CLI command file and by passing `query_cli_test.go` |
| `internal/mcp/tools.go` | `internal/query.Engine` | `openEngine` (fresh `query.OpenAt` per tool call, now path-confined) | ✓ WIRED | Confirmed by code inspection + live JSON-RPC test |
| `internal/mcp/server.go` | `github.com/mark3labs/mcp-go/server` | `server.NewMCPServer` + `AddTool` conditional gating | ✓ WIRED | Live-verified: 3/3 visibility scenarios behaved exactly as documented |
| `internal/query` / `internal/mcp` | `internal/graphstore.Reader` interface only | archtest boundary | ✓ WIRED | `TestNoPackageBypassesGraphStore` passes; no direct pebble import found |
| `testdata/golden/golden_parity_test.go` | real `weft` corpus + `internal/indexer.Run` + `internal/query.Engine` | corpus resolver + engine invocation + diff | ✓ WIRED | Ran and passed against the actual pinned commit present on this machine |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| MCP default tool visibility | Built binary, `initialize` + `tools/list` over stdio (no `CODEGRAPH_MCP_TOOLS`) | `tools: [codegraph_explore]` only | ✓ PASS |
| MCP allowlist visibility + unknown-name warning | Same, with `CODEGRAPH_MCP_TOOLS=node,status,bogus` | `tools: [codegraph_explore, codegraph_node, codegraph_status]`; stderr: `unknown tool name "bogus" in CODEGRAPH_MCP_TOOLS, ignoring` | ✓ PASS |
| MCP zero-tools when uninitialized | Same, in a directory with no `.codegraph/` | `initialize` succeeds; `tools: []` | ✓ PASS |
| MCP `codegraph_explore` tool call | `tools/call codegraph_explore {"query":"main"}` | Markdown response with Exploration header, blast radius, verbatim-source disclaimer, matching golden template | ✓ PASS |
| MCP CR-02 path confinement | `tools/call codegraph_explore {"query":"main","path":"<other repo>"}` | `isError: true`, `"mcp: path ... is outside the server's configured repo root"` | ✓ PASS |
| `go build ./...` | full build | clean, no errors | ✓ PASS |
| `go vet ./...` | full vet | clean, no warnings | ✓ PASS |
| `go test ./...` | full unit suite | all packages `ok` | ✓ PASS |
| `go test ./testdata/golden/ -run TestGoldenParity -v` | golden parity against real pinned corpus | all 7 subtests PASS (1 documented informational divergence in impact arithmetic, not a failure) | ✓ PASS |
| `go test ./internal/graphstore/archtest/...` | import-boundary enforcement | PASS | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| QRY-01 | 03-01, 03-02, 03-03, 03-08 | `query <search>` with `--kind`/`--limit`/`--json` | ✓ SATISFIED | `internal/cli/query.go`, `internal/query/search.go`, tested |
| QRY-02 | 03-06, 03-08 | `node <symbol\|file>` detail/file read | ✓ SATISFIED | `internal/cli/node.go`, `internal/query/node.go`, tested |
| QRY-03 | 03-01, 03-03, 03-08 | `search` locations-only | ✓ SATISFIED | `internal/cli/search.go`, `internal/query/search.go`, tested |
| QRY-04 | 03-04, 03-08 | `callers`/`callees` traversal | ✓ SATISFIED | `internal/cli/{callers,callees}.go`, `internal/query/traverse.go`, tested |
| QRY-05 | 03-04, 03-08 | `impact --depth` | ✓ SATISFIED | `internal/cli/impact.go`, `internal/query/traverse.go`, tested |
| QRY-06 | 03-04, 03-08 | `affected [files...]` (D-07 query-time derivation, documented divergence from literal "test-coverage edge type" wording, confirmed via checkpoint) | ✓ SATISFIED | `internal/cli/affected.go`, `internal/query/traverse.go` `Affected`, tested |
| QRY-07 | 03-01, 03-05, 03-08 | `files` browse (format/filter/pattern/depth) | ✓ SATISFIED | `internal/cli/files.go`, `internal/query/files.go`, tested |
| QRY-08 | 03-06, 03-08 | `explore <query>` one-round-trip | ✓ SATISFIED | `internal/cli/explore.go`, `internal/query/explore.go`, live-verified |
| QRY-09 | 03-01, 03-05, 03-08 | `status --json` health/counts | ✓ SATISFIED | `internal/cli/status.go`, `internal/query/status.go`, tested |
| MCP-01 | 03-07, 03-08 | `codegraph_explore` only default tool | ✓ SATISFIED | Live-verified + `TestDefaultToolVisibility` |
| MCP-02 | 03-07 | `CODEGRAPH_MCP_TOOLS` allowlist | ✓ SATISFIED | Live-verified + `TestAllowlist` |
| MCP-03 | 03-07 | Zero tools when no `.codegraph/` | ✓ SATISFIED | Live-verified + `TestNoIndexZeroTools` |
| MCP-04 | 03-09 | Golden-corpus parity | ✓ SATISFIED | `TestGoldenParity` ran and passed against the real pinned corpus |

No orphaned requirements — all 13 Phase-3 IDs in REQUIREMENTS.md are claimed by a plan and satisfied.

### Anti-Patterns Found

None blocking. Scanned every non-test `.go` file under `internal/query`, `internal/mcp`, `internal/cli`, `internal/graphstore`, `testdata/golden` for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/stub-return patterns:
- Zero `TBD`/`FIXME`/`XXX`/`TODO`/`HACK` markers found.
- `internal/query/status.go` contains the word "placeholder" 4 times, but these are documented, intentional D-05 design decisions (Phase-4 sync concepts — `pendingChanges`/`worktreeMismatch`/`pendingRefs` — rendered as inert, all-zero/null values because Phase 3 is explicitly read-only and does not reconcile drift). Not a code stub; the surrounding comment block is an explicit per-key TS→Go mapping table, and the values are the intentional/correct Phase-3 behavior, not unfinished work.
- `internal/mcp/server.go`'s one "placeholder" comment refers to the MCP implementation-version string constant, which is documented as intentionally arbitrary since no release-version concept exists yet — cosmetic, not functional.

One pre-existing item was explicitly deferred, not silently dropped: `IN-01` (`DeleteFileSubgraph` naming/doc mismatch) was scoped out of the review-fix pass per `03-REVIEW-FIX.md`'s own note ("tracked as Phase-4 debt") — this is a Phase-1 artifact unrelated to Phase-3's `internal/query`/`internal/mcp` surface and does not block this phase's goal.

### Human Verification Required

None. The plan (03-08) designated the live MCP stdio handshake as a human-verify checkpoint, but this verifier independently built the binary and exercised all three MCP-01/02/03 visibility scenarios plus the CR-02 path-confinement defense via raw JSON-RPC over stdio against the real server — not a simulation, not trusting the SUMMARY's claim. All behaviors observed matched the documented contract.

### Gaps Summary

No gaps. All four ROADMAP Phase-3 success criteria are observably true in the codebase: the full 9-command query suite is wired with documented flags and passes both unit and black-box CLI tests; `explore` produces the golden markdown shape in a live round trip; the MCP server's three-tier tool-visibility contract (default/allowlist/zero-tools) was independently exercised live and matches spec; and the golden parity harness runs against the real pinned `weft` corpus and passes for all 7 captured commands. The two Critical code-review findings (CR-01 unbounded-limit DoS, CR-02 MCP path confinement) are fixed, present in the code, covered by dedicated regression tests, and — for CR-02 — independently reproduced live by this verifier. `go build`/`go vet`/`go test ./...` are all clean, and the `internal/graphstore/archtest` import-boundary enforcement (which automatically covers the new `internal/query`/`internal/mcp` packages via whole-module `go/packages` loading) passes.

---

_Verified: 2026-07-11T16:36:13Z_
_Verifier: Claude (gsd-verifier)_
