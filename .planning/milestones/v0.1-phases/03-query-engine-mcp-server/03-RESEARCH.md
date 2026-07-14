# Phase 3: Query Engine & MCP Server - Research

**Researched:** 2026-07-11
**Domain:** Read-only graph query engine (Go) + stdio MCP server, over a frozen Pebble-backed graph store
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Command Surface & CLI Wiring (QRY-01…09, MCP-01)**
- **D-01:** Add the query subcommands and `serve` to the existing Cobra tree in `internal/cli/root.go` (alongside `init`/`index`/`uninit`). Follow Phase 2's thin-CLI pattern: commands resolve paths + flags and delegate all logic to a new read-only `internal/query` engine; the MCP server lives in a new `internal/mcp` package that calls the same engine/formatters. Commands: `query`, `node`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, `explore`, `serve`.
- **D-01a:** Every query command accepts `-p`/`--path` (default cwd) and resolves the nearest `.codegraph/` at or above that path (matches the MCP server's documented resolution behavior). No `.codegraph/` → a clear "not initialized" error (reuse the `internal/cli` `ErrNotInitialized` idiom); the MCP server handles the no-index case differently (D-08a).
- **D-01b:** Output flags mirror the TS capture surface: `--json` on the structured commands (`query`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`); `explore` and `node` emit markdown text only. Per-command flags per requirements: `--kind` (query/search), `--limit` (query/search/callers/callees), `--depth` (impact), plus format/filter/pattern/depth for `files` (QRY-07).

**Read Path Over the Frozen Graph (INDX-05 contract; the load-bearing decisions)**
- **D-02:** The query engine is strictly read-only over `GraphStore.Snapshot()` — one `Reader` snapshot per CLI invocation; the MCP server takes a fresh snapshot per tool call. No writes, no reindex, no interface bypass — the `internal/graphstore/archtest` boundary still holds.
- **D-03:** Additively extend the `Reader` interface with node/file enumeration — add iterator methods over the existing `n/` (optionally kind-scoped) and `f/` prefixes (e.g. `IterateNodes(kindPrefix string)`, `IterateFiles()`), reusing the range-scan keyspace design from Phase 1. Additive to the interface only — no key change, no schema bump, no reindex, archtest unaffected. Exact signatures are executor's discretion.
- **D-04:** Reverse traversal (`callers`, `impact`, `affected`) is computed at query time: build an in-memory reverse-adjacency map by scanning all forward edges once (`IterateEdges("")`) per invocation, then traverse it. Forward traversal (`callees`) uses `IterateEdges(src)` directly. No persisted reverse-edge index in Phase 3 (deferred to Phase 8 if profiling shows the scan dominates).

**Parity Model (MCP-04)**
- **D-05:** Parity = output-shape / key-name / semantic-structure parity against the golden corpus — NOT byte-identical values. Documented, expected divergences: node ids differ (compare on stable fields, not `id`); edge dedup (multiplicity may differ — documented, not a bug); `status` backend fields get analogous Go/Pebble-truthful values (drop or remap keys with no Go analog); `query` `score` is omitted; `languages`/`nodesByKind` reflect Go-only extraction until Phase 5.
- **D-05a:** `explore` markdown template reproduces the golden `explore.json` output structure verbatim in shape, including the verbatim-source disclaimer paragraph copied from the golden — reproduce, don't paraphrase.
- **D-05b:** `node` markdown template reproduces the golden `node.json` shape exactly (Location/Signature/Trail/Calls→/Called by←).

**Search / Explore Matching — no embeddings (QRY-01, QRY-03, QRY-08)**
- **D-06:** `query`/`search`/`explore` matching is deterministic lexical matching over `name` + `qualifiedName` (substring/token match, `--kind` filter, `--limit` cap) — no embeddings/vector search. TS's FTS5/BM25 `score` is not reproduced; ranking uses a deterministic tie-break (exact-name > prefix > substring, then stable sort by qualifiedName/id) so `--json` output is byte-reproducible. `query` returns full node records; `search` returns locations-only.

**`affected` / `files` — no golden oracle (QRY-06, QRY-07)**
- **D-07:** `affected [files...]` derives impacted test files at query time from existing `calls` edges + a test-file heuristic (`_test.go` suffix / `Test*`/`Benchmark*` naming), using the reverse adjacency from D-04. Does NOT persist a new "test-coverage edge type" — documented divergence from QRY-06's literal wording. The user-facing behavior is fully preserved; only the mechanism is query-time derivation. The planner/researcher should confirm this is acceptable rather than blocking on a persisted edge. **This research confirms D-07 as architecturally sound and implementable over the frozen graph (see Summary and Don't Hand-Roll).**
- **D-07a:** `affected` and `files` have no golden fixture. Parity is best-effort against TS CLI behavior/documentation, not corpus-diffed. Do not fabricate a golden fixture; assert on structural correctness instead.

**MCP Server (MCP-01…MCP-04)**
- **D-08:** Build the stdio MCP server on `github.com/mark3labs/mcp-go` (`server.NewMCPServer` + `mcp.NewTool` + `server.ServeStdio`) — pure-Go, adds no CGo. Add it to `go.mod`. Re-evaluate the official `modelcontextprotocol/go-sdk` at the next milestone boundary.
- **D-08a:** `codegraph serve --mcp` runs the stdio server (stdio only in v1). Tool visibility: `codegraph_explore` is the only default-visible tool; the 7 companions register only when named in `CODEGRAPH_MCP_TOOLS` (comma-separated allowlist; unknown names ignored with a stderr warning). When no `.codegraph/` resolves, the server still starts and completes MCP init but advertises zero tools.
- **D-08b:** MCP tools reuse the exact CLI query engine + formatters so tool output shapes match the golden corpus with no second code path. Each tool call takes a fresh `Snapshot()`. Tool arg schemas mirror the CLI flags.

### Claude's Discretion
- Exact `IterateNodes`/`IterateFiles` signatures and whether kind-filtering is a key-prefix scan or a full-node scan with in-memory filter.
- The lexical ranking algorithm's precise scoring/tie-break, and the in-memory adjacency data structures.
- How `status.nodesByKind` / `languages` are computed (node scan vs. an added Meta breakdown — prefer scan to avoid touching the writer).
- Markdown rendering helper structure; where the `.codegraph/`-resolution helper lives (shared `internal/cli` vs `internal/query`).
- Whether `serve` is a top-level command with `--mcp` or the sole mode; concrete MCP tool names beyond the fixed `codegraph_explore`.

### Deferred Ideas (OUT OF SCOPE)
- Incremental `sync`, native file watchers, rename/delete pruning, daemon, staleness banner, MCP reconnect reconciliation — Phase 4 (INDX-03/04, SYNC-*). Phase 3's `status` reports health but does not reconcile drift.
- Persisted reverse-edge index — Phase 8 scale work; adopt only if the D-04 query-time scan profiles as a bottleneck on the 100k-file monorepo.
- Synthesized dispatch edges, `provenance: heuristic`, a persisted test-coverage edge type, framework `route` nodes — Phase 5 (RES-02/RES-03, LANG-07). `affected` derives test impact at query time instead (D-07).
- Non-Go language query output (multi-language `status.languages`/`nodesByKind`) — Phase 5 (LANG-02+). Phase 3 proves the query surface on the Go graph.
- Embeddings / semantic search over `explore`/`query` — future milestone (EMBED-01); v1 stays lexical/structural (D-06).
- MCP HTTP/SSE transport, remote/multi-user queries — v2 (SERVER-01). Phase 3 ships stdio only.
- `install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry` — Phase 6 (AGNT-*, CLI-*). Phase 3 ships `serve` + query commands.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| QRY-01 | `codegraph query <search>` symbol search with `--kind`, `--limit`, `--json` | D-06 deterministic matcher (Architecture Pattern in Standard Stack); golden `query.json` shape confirmed verbatim (Code Examples) |
| QRY-02 | `codegraph node <symbol\|file>` symbol detail or line-numbered file read | D-05b markdown template confirmed verbatim against `node.json` (Code Examples); forward edges via `IterateEdges(src)`, reverse via D-04 map |
| QRY-03 | Lightweight locations-only `search` | D-06 — `search` shares the D-06 matcher, differs only in projection (locations-only vs. full record), confirmed against `callers.json`/`callees.json` location shape |
| QRY-04 | `callers`/`callees` reverse/forward call-graph traversal | Pattern 2 (D-04 reverse adjacency) for `callers`; direct `IterateEdges(src)` for `callees`; both shapes confirmed against golden fixtures |
| QRY-05 | `impact <symbol> --depth` transitive blast-radius | Pattern 2 BFS over reverse adjacency, depth-bounded; `impact.json` shape confirmed, semantics flagged as Open Question 1 |
| QRY-06 | `affected [files...]` impacted test files | D-07 query-time derivation over reverse `calls` + test-file heuristic — confirmed implementable over the frozen graph with no new edge kind (Summary, Don't Hand-Roll) |
| QRY-07 | `files` browse indexed structure with format/filter/pattern/depth | D-03's `IterateFiles()` additive iterator; no golden oracle (D-07a) — structural-correctness testing only |
| QRY-08 | `explore <query>` verbatim source + call paths + blast radius in one round trip | D-05a markdown template confirmed byte-for-byte against `explore.json` including the disclaimer paragraph (Code Examples) |
| QRY-09 | `status --json` index health/counts/staleness | Golden `status.json` field-by-field mapping analysis (Code Examples); TS-only field remapping flagged as Open Question 2 |
| MCP-01 | `codegraph serve --mcp` with `codegraph_explore` as only default tool | Pattern 3 (mcp-go conditional `AddTool` at startup), confirmed against current mcp-go v0.56.0 API via Context7 |
| MCP-02 | 7 additional tools via `CODEGRAPH_MCP_TOOLS` allowlist | Pattern 3 — allowlist parsing + conditional registration; unknown-name stderr-warning behavior specified |
| MCP-03 | Zero tools when no `.codegraph/` exists | Pattern 3 — `hasIndex` check gates all `AddTool` calls including `codegraph_explore` itself |
| MCP-04 | MCP tool output shapes match TS CodeGraph v1.3.x golden corpus | D-08b one-engine-two-front-ends architecture (Architectural Responsibility Map); Validation Architecture section maps this to `testdata/golden/golden_parity_test.go` |
</phase_requirements>

## Summary

Phase 3 is an internal-integration problem, not a new-technology problem. The two hard technical questions — "how do I enumerate nodes I currently can't enumerate" (D-03) and "how do I traverse edges backward when the store is forward-only" (D-04) — are both solvable entirely inside `internal/graphstore`'s existing Pebble/range-scan idioms, with no schema change and no reindex. The one genuinely new external dependency is `github.com/mark3labs/mcp-go` (D-08), whose stdio-server API is small, stable, and directly confirmed against its current (v0.56.0) documentation: `server.NewMCPServer` + `mcp.NewTool` + `s.AddTool` + `server.ServeStdio`, with typed `Require*`/`Get*` argument helpers. Conditional tool visibility (MCP-01/02/03) requires no dynamic/session-tool machinery — it is simply conditional `AddTool` calls made once at process startup, based on the `CODEGRAPH_MCP_TOOLS` env var and whether `.codegraph/` resolved.

The golden corpus (`testdata/golden/corpus/weft-go/*.json`) is the literal spec for every JSON key name and every markdown template byte, including the verbatim-source disclaimer paragraph in `explore.json`. This research confirms the exact shapes for `query`/`callers`/`callees`/`impact`/`status`/`explore`/`node` by reading the fixtures directly (not by memory), and cross-references the TS SQLite DDL (`ts-schema.sql`) for the full `nodes` column vocabulary so `query --json`'s full-record output can be reproduced field-for-field from the existing `schema.Node` proto (no schema change needed — every field TS's `nodes` table has, `schema.Node` already carries).

D-07's divergence (deriving `affected` from `calls` edges at query time instead of persisting a new test-coverage edge kind) is architecturally sound: Phase 2 froze the edge vocabulary to ground-truth AST kinds, D-04's reverse-adjacency map already gives `affected` everything it needs (walk reverse-`calls` from each changed file's symbols, filter targets by test-file heuristic), and no new key, no writer change, and no reindex is required. This research recommends confirming D-07 as-is.

**Primary recommendation:** Build one `internal/query` engine (`Engine` type wrapping a `graphstore.Reader`, one method per command) and one set of formatters (JSON + markdown), consumed identically by `internal/cli`'s ten Cobra commands and `internal/mcp`'s stdio tool handlers — this is what makes MCP-04 parity hold without a second code path (D-08b). Extend `Reader` additively with `IterateNodes(kindPrefix string)` and `IterateFiles()` (D-03) reusing the exact `appendSegment`/range-scan pattern already in `keys.go`, and build the callers/impact/affected reverse-adjacency map once per invocation via a single `IterateEdges("")` scan (D-04).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Symbol/lexical search (`query`, `search`, `explore`) | Query Engine (`internal/query`) | Storage (`GraphStore.Reader`) | Matching logic (D-06 tie-break) is engine-owned; storage only supplies the node iterator to scan |
| Forward call traversal (`callees`) | Storage (`Reader.IterateEdges`) | Query Engine | Already a direct range scan — no new storage capability needed |
| Reverse call traversal (`callers`, `impact`, `affected`) | Query Engine (in-memory reverse-adjacency, D-04) | Storage (`Reader.IterateEdges("")` full scan) | Storage stays forward-only (frozen Phase-2 keyspace); the engine builds the reverse view per-invocation |
| Node/file enumeration (`files`, `status.nodesByKind`, `search` fallback) | Storage (`Reader.IterateNodes`/`IterateFiles`, D-03 additive) | Query Engine | New capability lives at the storage boundary since it's a raw range scan over `n/`/`f/` prefixes, same pattern as existing `IterateEdges` |
| Markdown rendering (`explore`, `node`) | Query Engine (`internal/query` formatters) | — | Template fidelity to the golden corpus is a formatting concern, not a storage or transport concern |
| CLI argument/flag handling, path resolution | CLI (`internal/cli`) | Query Engine (`.codegraph/` resolution helper, shared) | Thin-CLI pattern (D-01) — commands parse flags/paths and delegate; the nearest-`.codegraph/` walk is shared logic, callable from both CLI and MCP |
| MCP transport (stdio JSON-RPC, tool schema) | MCP Server (`internal/mcp`) | Query Engine | `mcp-go` owns protocol framing; `internal/mcp` handlers call the same `internal/query.Engine` methods CLI commands call (D-08b) |
| Tool visibility gating (default-only / allowlist / zero-tools) | MCP Server (`internal/mcp`) | — | Pure startup-time registration logic; no engine involvement |
| Source-file rendering (verbatim, line-numbered, tab-indented) | Query Engine (`explore`/`node` formatters) | Filesystem (`os.ReadFile`) | "Read fresh from disk" (D-05a) means the formatter reads the file directly, not from the stored `Node`/`File` record |
| Reader/GraphStore interface boundary | Storage (`internal/graphstore`) | — | `internal/query`/`internal/mcp` depend only on the `Reader` interface — `archtest.TestNoPackageBypassesGraphStore` must continue to pass with these two new packages added |

## Package Legitimacy Audit

Only one new external dependency is introduced this phase.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/mark3labs/mcp-go` | Go module proxy | v0.1.0 first tagged 2024; 56 minor releases through v0.56.0, actively maintained | High real-world adoption (broadest of Go MCP servers per project's own CLAUDE.md tech-stack research) | `github.com/mark3labs/mcp-go` (public, active) | OK | Approved — already a locked decision (CONTEXT D-08), re-confirmed here |

**Verification performed:** `go list -m github.com/mark3labs/mcp-go@v0.56.0` resolves via the Go module proxy `[VERIFIED: proxy.golang.org]`. `go.mod` of v0.56.0 declares `go 1.25.5` as its minimum — compatible with this project's `go 1.26.5` `[VERIFIED: go module proxy]`. Its direct dependencies (`google/jsonschema-go`, `google/uuid`, `santhosh-tekuri/jsonschema/v6`, `spf13/cast`, `stretchr/testify`) are all established pure-Go libraries with no CGo — adding `mcp-go` does not compromise the project's CGo-only-for-tree-sitter constraint `[CITED: v0.56.0 go.mod]`. API surface (`NewMCPServer`, `NewTool`, `AddTool`, `ServeStdio`) is confirmed current against Context7-fetched official docs (2026), not training-data memory `[VERIFIED: Context7 /mark3labs/mcp-go]`.

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** none.

## Standard Stack

### Core (already in `go.mod` — no new pinning needed except mcp-go)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/mark3labs/mcp-go` | v0.56.0 (latest as of research; requires go ≥1.25.5) | Stdio MCP server, tool registration | Locked decision D-08; pure Go, broadest Go MCP-server adoption per project's own stack research `[VERIFIED: go module proxy + Context7 docs]` |
| `github.com/spf13/cobra` | v1.10.2 (already pinned) | Query subcommand tree | Existing convention (D-01) — no new library, extend `newRootCmd`'s `AddCommand` calls |
| `github.com/cockroachdb/pebble/v2` | v2.1.6 (already pinned) | Underlying store the `Reader` extensions range-scan over | No new usage pattern — `IterateNodes`/`IterateFiles` reuse `pebbleReader.IterateEdges`'s exact `NewIter(&pebble.IterOptions{LowerBound, UpperBound})` shape `[VERIFIED: internal/graphstore/pebble_store.go]` |
| `google.golang.org/protobuf` | v1.36.11 (already pinned) | `schema.Node`/`Edge`/`File`/`Meta` marshal/unmarshal | No schema change this phase — every field `query --json`/`status --json` need already exists on `schema.Node`/`schema.Meta` `[VERIFIED: internal/schema/graph.proto]` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` (stdlib) | go1.26.5 | `--json` output marshaling | All structured commands (`query`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`) |
| `strings`/`sort` (stdlib) | go1.26.5 | D-06 deterministic lexical matching + tie-break | `internal/query`'s search/query matcher — no third-party fuzzy-match library needed; TS's FTS5/BM25 is explicitly not reproduced (score stripped) |
| `text/tabwriter` or manual `strings.Builder` (stdlib) | go1.26.5 | `explore`/`node` markdown rendering, tab-indented fenced source blocks | Formatter layer — golden fixture uses literal `\t` (tab) indentation inside fenced code blocks, confirmed by inspecting `explore.json`'s raw `\n1\t// cmd/weft/main.go\n...` output |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `mark3labs/mcp-go` | `modelcontextprotocol/go-sdk` (official) | Already evaluated and deferred in CLAUDE.md/D-08 — newer, less Go-server production mileage today; re-evaluate at next milestone boundary, not this phase |
| Query-time reverse-adjacency scan (D-04) | A persisted reverse-edge index (`e_rev/<dst>/<kind>/<src>`) | Rejected for v1 — would require touching Phase 2's frozen writer and a reindex; deferred to Phase 8 if the 100k-file monorepo profiles the scan as a bottleneck (explicitly noted in CONTEXT D-04) |
| Deterministic lexical tie-break (D-06) | FTS5/BM25-equivalent scoring | Rejected — `score` is explicitly stripped from the golden fixtures as volatile (README "Volatile fields" table); a deterministic tie-break is required for `--json` byte-reproducibility, which BM25 floating-point scores cannot provide |

**Installation:**
```bash
go get github.com/mark3labs/mcp-go@v0.56.0
```
No other new dependencies. Do **not** run a bare `go mod tidy` — this project's established convention (see STATE.md Phase 1 decisions) is to promote/add dependencies explicitly and let intentionally-pre-pinned-but-unimported deps alone; a tidy pass here is safe only after this phase's new imports (`internal/query`, `internal/mcp`) actually compile and reference `mcp-go`.

**Version verification:** `go list -m github.com/mark3labs/mcp-go@v0.56.0` resolved successfully against `proxy.golang.org` on 2026-07-11; `go.mod` of that tag declares `go 1.25.5` (project is on `go 1.26.5`, compatible) `[VERIFIED: go module proxy]`.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │           Two front-ends,                │
                    │        one engine (D-08b)                │
                    └─────────────────────────────────────────┘

  CLI invocation                          MCP stdio connection
  `codegraph query "foo" -p .`            (agent sends CallTool "codegraph_explore")
        │                                          │
        ▼                                          ▼
  internal/cli/*.go                      internal/mcp/server.go
  (Cobra RunE: parse flags,               (mcp-go: NewMCPServer,
   resolve -p/--path)                      conditional AddTool per
        │                                  CODEGRAPH_MCP_TOOLS env,
        │                                  parse tool args via
        │                                  req.RequireString/GetInt)
        │                                          │
        └──────────────┬───────────────────────────┘
                        ▼
         resolveCodegraphDir(startPath)   (D-01a: walk filepath.Dir()
                        │                  upward until .codegraph/ found
                        │                  or filesystem root reached)
                        ▼
              graphstore.Open(storeDir) → GraphStore.Snapshot() → Reader
                        │                  (one fresh snapshot per CLI
                        │                   invocation OR per MCP tool call,
                        │                   D-02 — never reused across calls)
                        ▼
              internal/query.Engine{reader}
        ┌───────────────┼────────────────────────────────┐
        ▼               ▼                                ▼
  Forward path     Reverse path (D-04)              Enumeration path (D-03)
  reader.IterateEdges(src)   buildReverseAdjacency()      reader.IterateNodes(kindPrefix)
  → callees                  (one full IterateEdges("")   reader.IterateFiles()
                              scan → map[dst][]Edge)       → query/search/files/
                              → callers, impact (BFS       status.nodesByKind
                              bounded by --depth),
                              affected (test-file filter
                              over reverse calls)
        │               │                                │
        └───────────────┴────────────────┬───────────────┘
                                          ▼
                          Formatters (internal/query/render*.go)
                     ┌────────────────────┴────────────────────┐
                     ▼                                          ▼
              JSON encoder                          Markdown renderer (D-05a/b)
              (query/search/callers/callees/         (explore/node: os.ReadFile
               impact/affected/files/status)          the file fresh, tab-indent,
                     │                                 line-number, verbatim-
                     │                                 source disclaimer)
                     └────────────────────┬────────────────────┘
                                          ▼
                          cmd.OutOrStdout() (CLI)  OR
                          mcp.NewToolResultText()   (MCP)
```

### Recommended Project Structure
```
internal/
├── query/
│   ├── engine.go          # Engine{reader graphstore.Reader}; one method per command
│   ├── resolve.go         # resolveCodegraphDir(startPath) — shared D-01a walk-up helper
│   ├── search.go          # D-06 deterministic lexical matcher (query + search share the matcher, differ in projection)
│   ├── traverse.go        # D-04 reverse-adjacency builder + BFS (callers, impact, affected)
│   ├── render_json.go     # struct → golden-shaped JSON (query/callers/callees/impact/affected/files/status)
│   ├── render_markdown.go # explore/node markdown templates (D-05a/D-05b), byte-exact to golden
│   └── engine_test.go
├── mcp/
│   ├── server.go          # newMCPServer(reader-provider, allowlist) — D-08a tool visibility gating
│   ├── tools.go           # tool schema defs (mirror CLI flags) + handlers delegating to internal/query.Engine
│   └── server_test.go
└── cli/
    ├── root.go            # AddCommand(newQueryCmd(), newNodeCmd(), ..., newServeCmd()) — extend existing tree
    ├── query.go, node.go, search.go, callers.go, callees.go, impact.go, affected.go, files.go, status.go, explore.go, serve.go
    └── (existing init.go/index.go/uninit.go untouched)
```

### Pattern 1: Additive Reader extension via range-scan (D-03)
**What:** Two new `Reader` methods, `IterateNodes(kindPrefix string) (NodeIterator, error)` and `IterateFiles() (FileIterator, error)`, implemented in `pebbleReader` exactly like the existing `IterateEdges`.
**When to use:** Any query needing "all nodes" or "all files" rather than a single lookup (`query`, `search` fallback, `files`, `status.nodesByKind`/`languages`).
**Example:**
```go
// Source: internal/graphstore/pebble_store.go (existing IterateEdges, the pattern to mirror)
func (r *pebbleReader) IterateEdges(srcPrefix string) (EdgeIterator, error) {
	lower := edgeSrcPrefix(srcPrefix)
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleEdgeIterator{iter: iter}, nil
}

// New (D-03) — same shape, over the n/ namespace. kindPrefix == "" scans all nodes;
// non-empty scans only that kind IF node ids embed kind as their first segment
// (they do: NodeID returns "kind:sha256hex" — see nodeid.go), so a kind-scoped
// scan is a prefix-range over nodeKey(kind + ":") rather than a full scan +
// filter. Exact signature/kind-scoping strategy is executor's discretion (CONTEXT).
func (r *pebbleReader) IterateNodes(kindPrefix string) (NodeIterator, error) {
	lower := nodeKeyPrefix(kindPrefix) // new keys.go helper, mirrors edgeSrcPrefix
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleNodeIterator{iter: iter}, nil
}
```
**Caution:** `nodeKey(id)` currently length-prefixes the *entire* id (`kind:hash`) as one opaque segment via `appendSegment` — it does NOT expose `kind` as a separately-scannable key segment today. A kind-scoped prefix scan therefore requires either (a) a full `n/` scan with in-memory kind filtering (simplest, correctness-first, matches the phase's stated "un-optimized, correctness-first" posture), or (b) a byte-prefix match against `nodeKey(kind+":")`'s literal prefix bytes exploiting the fact that `NodeID` always emits `"kind:" + hex` verbatim before hashing is irrelevant to the *stored key* — the stored key is `appendSegment(id)` where `id` itself starts with the literal string `"kind:"`. Since `appendSegment` is a length-then-bytes encoding, the length prefix varies with `len(id)`, so a raw byte-prefix match on `nodeKey("")` still won't cleanly restrict to one kind without decoding the length varint first. **Recommendation: start with (a) full scan + in-memory filter for v1 correctness; this is explicitly the same posture CONTEXT takes for D-04's edge scan** ("scan is deliberately un-optimized... correctness-first for the golden corpus"). Confirm this with the planner rather than attempting a byte-level prefix trick that the current `appendSegment` encoding does not cleanly support.

### Pattern 2: Query-time reverse adjacency (D-04)
**What:** One `IterateEdges("")` full scan per CLI invocation / MCP tool call, building `map[dst][]*schema.Edge` in memory, then BFS/lookup against that map for `callers`/`impact`/`affected`.
**When to use:** Any reverse-traversal command. Forward traversal (`callees`) does NOT need this — use `reader.IterateEdges(src)` directly.
**Example:**
```go
// Pattern (no direct golden-fixture source — this is new engine code, not
// reproduced from an official doc). Cross-checked against store.go's
// documented IterateEdges contract and CONTEXT D-04's explicit design.
func buildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("") // "" == every edge, D-04
	if err != nil {
		return nil, err
	}
	defer it.Close()
	rev := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		rev[e.Target] = append(rev[e.Target], e)
	}
	return rev, it.Err()
}

// impact --depth N: BFS outward from the target node's id through rev,
// bounding hop count at N; nodeCount/edgeCount in impact.json count the
// distinct visited nodes/edges across the whole traversal (confirmed by
// reading impact.json: symbol="mergeStyle", depth=2, nodeCount=5 (the 5
// "affected" entries including mergeStyle itself), edgeCount=4 (mergeStyle
// has exactly 3 direct callers per callers.json, plus 1 second-hop caller
// of newFinishReconcileCmd — newFinishCmd — giving 4 total edges traversed
// at depth 2). Verify this arithmetic precisely against the fixture during
// planning/implementation, not just this research summary.
```
**Caution — memory/perf note for the planner:** D-04 explicitly defers optimization to Phase 8; do not add caching or a persisted reverse index in this phase.

### Pattern 3: MCP stdio server with startup-time conditional tool registration (D-08a)
**What:** `codegraph serve --mcp` builds one `*server.MCPServer`, then conditionally calls `AddTool` based on (a) whether `.codegraph/` resolved and (b) the `CODEGRAPH_MCP_TOOLS` allowlist — all decided once before `ServeStdio` blocks. No per-session dynamic tool add/remove is needed (that mcp-go feature — `AddSessionTool`/`DeleteSessionTools` — is for genuinely per-client-session tool sets, which this phase does not need; visibility here is process-global, decided once at startup from env + filesystem state).
**When to use:** `internal/mcp/server.go`'s `NewServer` / `Serve` entrypoint, called from `internal/cli/serve.go`.
**Example:**
```go
// Source: Context7 /mark3labs/mcp-go (v0.56.0 official docs), adapted to
// this phase's conditional-registration requirement (MCP-01/02/03).
func buildServer(hasIndex bool, allowlist map[string]bool) *server.MCPServer {
	s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))
	if !hasIndex {
		return s // MCP-03: zero tools when no .codegraph/ resolves
	}
	// codegraph_explore is the only default-visible tool (MCP-01)
	s.AddTool(exploreTool(), exploreHandler)
	for _, name := range []string{"node", "search", "callers", "callees", "impact", "files", "status"} {
		if allowlist[name] { // parsed from CODEGRAPH_MCP_TOOLS, comma-separated
			s.AddTool(companionTool(name), companionHandler(name))
		}
	}
	return s
}

func exploreTool() mcp.Tool {
	return mcp.NewTool("codegraph_explore",
		mcp.WithDescription("Explore relevant symbols: verbatim source, call paths, blast radius"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural-language or symbol/file query")),
		mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		mcp.WithNumber("max_files", mcp.Description("Cap on files returned")),
	)
}
```
**Unknown-name handling (MCP-02):** parse `CODEGRAPH_MCP_TOOLS` into a set; for any name in the set not in the 7 known companions, log a stderr warning (never stdout — stdout is the JSON-RPC transport) and ignore it, per CONTEXT D-08a.

### Pattern 4: Nearest `.codegraph/` resolution (D-01a)
**What:** Walk `filepath.Dir()` upward from the given `-p`/`--path` (or cwd) checking for `.codegraph/` at each level, stopping when found or when `filepath.Dir(p) == p` (filesystem root).
**When to use:** Every query command and the MCP server's path-resolution logic (documented MCP behavior CONTEXT references) — should be one shared helper (executor's discretion on package location per CONTEXT).
**Example:**
```go
// Pattern confirmed as the standard Go/git idiom via WebSearch cross-check
// (no single stdlib call performs this walk; git rev-parse --show-toplevel
// and community libraries like go-findroot implement exactly this loop).
func resolveCodegraphDir(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ".codegraph")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInitialized // reuse existing cli.ErrNotInitialized idiom
		}
		dir = parent
	}
}
```

### Anti-Patterns to Avoid
- **A second rendering/matching code path for MCP:** D-08b is explicit — MCP tools must call the *same* `internal/query.Engine` methods and formatters the CLI uses. Building MCP-specific JSON shaping "for convenience" silently breaks MCP-04 parity even if the CLI's own golden test still passes.
- **Reusing one `Reader`/`Snapshot()` across multiple CLI commands or MCP tool calls:** D-02 requires a *fresh* snapshot per invocation/call. A long-lived MCP server process must call `store.Snapshot()` inside each tool handler, not once at server startup — otherwise the server serves a stale point-in-time view forever (irrelevant until Phase 4 sync exists, but the interface discipline should be correct from day one).
- **Persisting a new edge kind for `affected` (QRY-06's literal wording):** CONTEXT D-07 already flags and accepts this divergence — do not "fix" it by touching the frozen Phase-2 writer; that pulls Phase-5 provenance work forward out of order.
- **Byte-prefix-matching node ids to fake a kind-scoped scan:** `appendSegment`'s length-prefix encoding (see Pattern 1 caution) means a naive `bytes.HasPrefix` on `nodeKey(kind+":")` is not safe without first decoding the varint length. Prefer full-scan-with-filter for v1 correctness.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| MCP JSON-RPC framing, tool schema/capability negotiation | A custom stdio JSON-RPC loop | `mark3labs/mcp-go`'s `server.ServeStdio` | Protocol correctness (initialize handshake, capability advertisement, error envelopes) is exactly the kind of "deceptively complex" wire protocol this library already gets right; D-08 already locked this choice |
| Relevance-ranked full-text search (BM25/FTS) | A custom scoring function to *mimic* TS's FTS5 score | Nothing — D-06 explicitly drops score entirely | The golden fixtures already strip `score` as non-deterministic; building a scorer to match a value the oracle itself doesn't preserve is wasted, unverifiable effort |
| A persisted reverse-edge index | A new Pebble keyspace + writer changes for `e_rev/` | The D-04 in-memory query-time scan | Touches the frozen Phase-2 writer and forces a reindex — explicitly deferred to Phase 8, only if profiling proves the scan is a bottleneck |
| Directory-upward-walk libraries (`go-findroot`, `find-up` ports) | A third-party dependency for the `.codegraph/` resolution walk | ~10 lines of `filepath.Dir` loop (Pattern 4) | The loop is trivial and dependency-free; pulling in a library for it would violate the project's "minimal, audited dependencies" constraint for zero benefit |

**Key insight:** Nearly everything this phase needs already exists as an established pattern inside `internal/graphstore` (the `IterateEdges`/range-scan idiom) or as ~10-line idiomatic Go (the directory walk). The only place a real external library earns its keep is the MCP wire protocol itself.

## Common Pitfalls

### Pitfall 1: Treating `nodeKey`'s id encoding as kind-prefix-scannable
**What goes wrong:** Assuming `IterateNodes("function")` can be a cheap byte-prefix Pebble range scan because ids look like `"function:abc123..."`.
**Why it happens:** `appendSegment` length-prefixes the *whole* id as one opaque segment (see `keys.go`), so the stored key bytes are `[prefixNode][varint(len(id))][id bytes]` — the varint length prefix sits *before* the id string, meaning a literal-byte match against `nodeKey("function:")`'s prefix does not correspond to a valid Pebble lower-bound for "all ids starting with function:" without also matching on the length varint, which varies per id.
**How to avoid:** For v1, implement `IterateNodes(kindPrefix)` as a full `n/` range scan (`nodeKeyPrefix("")` == just `prefixNode`) with an in-memory `strings.HasPrefix(node.Id, kindPrefix+":")` or `node.Kind == kindPrefix` filter. This matches the project's existing "correctness-first" posture for D-04's edge scan.
**Warning signs:** A "kind-scoped iterator" that silently returns zero or wrong results because the byte-range math didn't account for the varint length prefix.

### Pitfall 2: Building the reverse-adjacency map per-command-invocation but then calling multiple commands in one process (MCP server case)
**What goes wrong:** In the CLI, one process = one command = one scan, so a package-level or long-lived reverse-adjacency cache is harmless. In the MCP server (one long-lived process serving many tool calls over stdio), a naively cached reverse-adjacency map built once at server startup would go stale the moment Phase 4 introduces background writes — even though Phase 3 itself is read-only and the graph is frozen during a single server's lifetime today.
**Why it happens:** Optimizing away the "obviously wasteful" repeated full-edge-scan looks free right now, but bakes in an assumption (immutable graph for the server's whole lifetime) that Phase 4 breaks.
**How to avoid:** Build the reverse-adjacency map fresh inside each tool handler (from a fresh `Snapshot()`, per D-02/D-08b), not once at server construction. This costs one edge-count-sized scan per relevant tool call (`callers`/`impact`/`affected`/`explore`'s blast-radius section) — acceptable per CONTEXT's stated "≈4k edges, trivial" scale target for this phase.
**Warning signs:** A `sync.Once` or package-level cache wrapping the reverse-adjacency build inside `internal/mcp`.

### Pitfall 3: Markdown template drift from the golden fixture's exact bytes
**What goes wrong:** Reproducing `explore`/`node` output "close enough" — e.g. paraphrasing the verbatim-source disclaimer paragraph, using spaces instead of a literal tab for source-line indentation, or reordering the `Calls →` / `Called by ←` sections — passes a casual read but fails MCP-04's byte-shape parity intent.
**Why it happens:** The golden fixture's `output` field is a single JSON string with embedded `\n`/`\t` — easy to skim past exact whitespace when reading it as rendered markdown instead of as a raw string.
**How to avoid:** Diff against `explore.json`/`node.json`'s raw string content (as read in this research — see Code Examples below) character-for-character where the template is fixed text; only the interpolated fields (symbol name, file paths, counts, source lines) should vary.
**Warning signs:** A parity test that string-trims/normalizes whitespace to make an explore/node comparison "pass" — that defeats the point of D-05a/D-05b.

### Pitfall 4: Unbounded `impact --depth` / large monorepo BFS as a resource-exhaustion vector
**What goes wrong:** A caller (human or, more dangerously, an untrusted/compromised MCP client) passes an unreasonably large `--depth`, causing the BFS over the in-memory reverse-adjacency map to visit a very large fraction of the graph, consuming excessive CPU/memory in what should be a fast, bounded local query.
**Why it happens:** `--depth` is user/agent-supplied input with no documented ceiling in the requirements.
**How to avoid:** Clamp `--depth` to a sane maximum (e.g. the graph's actual node count, or a fixed ceiling like 50) before starting BFS; document the clamp. This is a V5 Input Validation concern (see Security Domain below), not just a UX nicety.
**Warning signs:** No upper bound check on `--depth`/`--limit` flags in the CLI or MCP tool schema.

## Code Examples

Verified patterns from official/golden sources:

### mcp-go: minimal stdio server with a typed tool
```go
// Source: Context7 /mark3labs/mcp-go (README.md, v0.56.0 docs, fetched 2026-07-11)
package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("codegraph", "0.1.0", server.WithToolCapabilities(true))

	tool := mcp.NewTool("codegraph_explore",
		mcp.WithDescription("Explore relevant symbols with verbatim source and blast radius"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural-language or symbol/file query")),
	)
	s.AddTool(tool, exploreHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func exploreHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// delegate to internal/query.Engine.Explore(query, ...) — D-08b
	return mcp.NewToolResultText("..."), nil
}
```

### Golden `explore.json` — the literal markdown contract (D-05a)
```json
// Source: testdata/golden/corpus/weft-go/explore.json (read verbatim in this research)
{
  "command": "explore \"main function\" -p weft-go --max-files 1",
  "output": "**Exploration: main function**\n\nFound 1 symbol across 1 file.\n\n**Blast radius — what depends on these (update/verify before editing)**\n\n- `mergeStyle` (internal/cli/finish.go:378) — 3 callers in `internal/cli/finish.go`; tests: `internal/cli/finish_test.go`\n\n**Source Code**\n\n> The code below is the **verbatim, current on-disk source** of these files — re-read from disk on this call and line-numbered, byte-for-byte identical to what the Read tool returns. It is NOT a summary, outline, or stale cache. Treat each block as a Read you have already performed: do not Read a file shown here.\n\n**`cmd/weft/main.go`** — main(function)\n\n```go\n1\t// cmd/weft/main.go\n...\n```\n\n"
}
```
Note the exact structure to reproduce: `**Exploration: <query>**` blank-line `Found N symbol(s) across M file(s).` blank-line `**Blast radius — what depends on these (update/verify before editing)**` blank-line, then one bullet per blast-radius symbol: `` - `name` (path:line) — N callers in `path`; tests: `path` `` blank-line `**Source Code**` blank-line, the disclaimer blockquote (`>` prefixed, exact wording), blank-line, then per-file `` **`path`** — sym(kind) `` header followed by a fenced ` ```go ` block with tab-separated `<line>\t<source>` per line.

### Golden `node.json` — the literal markdown contract (D-05b)
```json
// Source: testdata/golden/corpus/weft-go/node.json (read verbatim in this research)
{
  "command": "node \"mergeStyle\" -p weft-go -f internal/cli/finish.go",
  "output": "**mergeStyle** (function)\n\n**Location:** internal/cli/finish.go:378\n**Signature:** `(r run.Runner, epic string) (string, error)`\n**Trail — codegraph_node any of these to follow it (no Read needed)**\n**Calls →** JJ (internal/run/run.go:50), Hardf (internal/exit/exit.go:50), mergeStyleSquashOrRebase (internal/cli/finish.go:348), mergeStyleMergeCommit (internal/cli/finish.go:347), Runner (internal/run/run.go:25)\n**Called by ←** newFinishReconcileCmd (internal/cli/finish.go:443), TestMergeStyleDetectsTrueMergeVsSquash (internal/cli/finish_test.go:384), TestMergeStyleSurfacesNonZeroExit (internal/cli/finish_test.go:666)\n"
}
```
Section order is fixed: `**name** (kind)` blank-line `**Location:** path:line` `**Signature:** \`sig\`` `**Trail — codegraph_node any of these to follow it (no Read needed)**` `**Calls →**` (forward edges, `name (path:line)` comma-joined) `**Called by ←**` (reverse edges, same format). Note **`Calls →` uses `IterateEdges(src)` directly (forward, no scan needed); `Called by ←` uses the D-04 reverse-adjacency map.**

### Golden `query.json` — full-record JSON wrapper shape
```json
// Source: testdata/golden/corpus/weft-go/query.json (first two entries, read verbatim)
[
  { "node": { "id": "function:fdfb6d7395c1fce8d245a491ea26bdac", "kind": "function", "name": "main", "qualifiedName": "main", "filePath": "cmd/weft/main.go", "language": "go", "startLine": 18, "endLine": 29, "startColumn": 0, "endColumn": 1, "signature": "()", "visibility": null, "isExported": false, "isAsync": false, "isStatic": false, "isAbstract": false } },
  { "node": { "id": "file:cmd/weft/main.go", "kind": "file", "name": "main.go", "qualifiedName": "cmd/weft/main.go", "filePath": "cmd/weft/main.go", "language": "go", "startLine": 1, "endLine": 30, "startColumn": 0, "endColumn": 0, "visibility": null, "isExported": false, "isAsync": false, "isStatic": false, "isAbstract": false } }
]
```
`query --json` is a **top-level array**, each element `{"node": {...}}` — not a bare array of node objects. `score` is absent (already stripped). Fields present but *not* on `schema.Node` today: `isAsync`, `isStatic`, `isAbstract` — TS-only concepts with no Go analog (Go has no async/static/abstract modifiers); render these as `false` literals for a Go graph rather than omitting the keys, to keep JSON *shape* parity (D-05's "keep key names/shape where meaningful" principle) even though the underlying `schema.Node` proto has no such fields. Confirm this rendering choice with the planner — it is this research's judgment call, not an explicit CONTEXT decision.

### Golden `callers.json` / `callees.json` / `impact.json` — locations-only shape
```json
// Source: testdata/golden/corpus/weft-go/callers.json (read verbatim)
{
  "symbol": "mergeStyle",
  "callers": [
    { "name": "newFinishReconcileCmd", "kind": "method", "filePath": "internal/cli/finish.go", "startLine": 443 }
  ]
}
```
```json
// Source: testdata/golden/corpus/weft-go/impact.json (read verbatim)
{
  "symbol": "mergeStyle", "depth": 2, "nodeCount": 5, "edgeCount": 4,
  "affected": [ { "name": "mergeStyle", "kind": "function", "filePath": "internal/cli/finish.go", "startLine": 378 }, "..." ]
}
```
`callers`/`callees` wrap in `{"symbol": ..., "callers"|"callees": [locations]}`; `impact` wraps in `{"symbol", "depth", "nodeCount", "edgeCount", "affected": [locations]}`. All location entries are `{name, kind, filePath, startLine}` — no `id`, no `endLine`, no signature. This is the "search returns locations-only" shape (D-06) applied consistently across callers/callees/impact, not just `search` itself.

### Golden `status.json` — field mapping for D-05's Go/Pebble-truthful values
```json
// Source: testdata/golden/corpus/weft-go/status.json (read verbatim)
{
  "initialized": true, "version": "1.3.1", "projectPath": "<CORPUS_PATH>", "indexPath": "<CORPUS_PATH>",
  "fileCount": 84, "nodeCount": 1223, "edgeCount": 4212,
  "backend": "node-sqlite", "journalMode": "wal",
  "nodesByKind": { "class": 1, "constant": 47, "file": 77, "function": 572, "import": 366, "interface": 1, "method": 85, "struct": 52, "type_alias": 2, "variable": 20 },
  "languages": ["go", "javascript", "python", "yaml"],
  "pendingChanges": { "added": 0, "modified": 0, "removed": 0 },
  "worktreeMismatch": null,
  "index": { "builtWithVersion": "1.3.1", "builtWithExtractionVersion": 24, "currentExtractionVersion": 24, "reindexRecommended": false, "state": "complete", "pendingRefs": 0 }
}
```
D-05's re-mapping guidance applied concretely: `backend: "node-sqlite"` → e.g. `"pebble"`; `journalMode: "wal"` → drop (no Pebble analog — Pebble's WAL is not user-facing config the same way) or map to Pebble's own durability descriptor if one is easily surfaced; `builtWithVersion`/`builtWithExtractionVersion`/`currentExtractionVersion` → `schema.Meta.SchemaVersion` is the one Go analog (`schema.SchemaVersion` const, currently `1`) — `reindexRecommended` can derive from `schema.IsCurrentSchemaVersion(meta)`. `pendingChanges`/`worktreeMismatch` are Phase-4 sync concepts with no meaning yet on a frozen graph — CONTEXT's own scope boundary says `status` "reports counts/health but does not reconcile filesystem drift" this phase, so these likely render as zero-value/`null` placeholders rather than being computed. **Flag for planner confirmation:** whether to keep these keys present-but-null/zero (shape parity) or omit them until Phase 4 gives them real meaning — this research recommends keeping them present-but-inert per D-05's "keep key names/shape where meaningful" language, but it is a judgment call, not an explicit CONTEXT decision.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| TS CodeGraph: SQLite FTS5 + BM25 relevance scoring for `query`/`search` | Deterministic lexical substring/token match, no score (D-06) | This phase (Phase 3 decision, not an upstream TS change) | `--json` output is byte-reproducible across runs; loses BM25's fuzzy-relevance ranking, which is an accepted v1 tradeoff per CONTEXT — embeddings/semantic search is the eventual replacement (EMBED-01, deferred) |
| TS CodeGraph: persisted `(source,target,kind,line,col)` edge uniqueness, distinct call sites kept as distinct edges | Pebble `e/<src>/<kind>/<dst>` key (no line/col in key) — multiple call sites between the same pair collapse to one edge | Phase 2 (D-05, already shipped) | `callers`/`impact` result multiplicity may differ from TS on repos with multiple call sites between the same two symbols — documented, accepted divergence (D-05), not something Phase 3 fixes |
| mcp-go early versions (`server.NewServer`, manual JSON-RPC dispatch, pre-v0.20) | `server.NewMCPServer` + `mcp.NewTool`(functional options) + `server.ServeStdio`, typed `Require*`/`Get*` request helpers, `NewStructuredToolHandler` for schema-validated typed I/O | Stabilized well before v0.56.0 (current) | The API surface fetched via Context7 in this research (2026) reflects the current, stable shape — do not reference older `mcp-go` examples that predate functional-options tool construction |

**Deprecated/outdated:**
- Nothing in this phase's dependency set is itself deprecated. The one thing to watch: `mark3labs/mcp-go` is explicitly a "re-evaluate at next milestone" choice per CLAUDE.md/D-08, not a permanent one — Phase 3 should keep the MCP tool-registration code isolated in `internal/mcp` so a later swap to `modelcontextprotocol/go-sdk` stays a bounded refactor, per the project's own stated intent.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `query --json`'s TS-only boolean fields (`isAsync`, `isStatic`, `isAbstract`) should render as literal `false` for Go nodes to preserve JSON shape, rather than being omitted | Code Examples — `query.json` shape | Low — a shape/key-presence choice; if wrong, the parity test's key-diffing logic just needs `--omit-ts-only-bools` normalization instead, an easy fix during planning |
| A2 | `status.json`'s Phase-4-only fields (`pendingChanges`, `worktreeMismatch`, `journalMode`) should render present-but-inert (zero/null) rather than be omitted entirely in Phase 3's `status --json` | Code Examples — `status.json` field mapping | Low-Medium — affects whether the parity test's normalization list needs an "omitted key" allowance vs. a "present-but-zero" allowance; a wrong guess here just means adjusting the golden-diff normalizer, not re-architecting `status` |
| A3 | `impact.json`'s `nodeCount`/`edgeCount` semantics (nodes/edges *visited during the depth-bounded BFS*, not the whole graph) were inferred from one fixture's arithmetic (`depth:2, nodeCount:5, edgeCount:4` matching `mergeStyle` + its 3 direct callers + 1 second-hop caller of `newFinishReconcileCmd`) | Architecture Patterns — Pattern 2 example comment | Medium — if the actual TS semantics differ (e.g. edgeCount counts something else), the `impact` JSON renderer would need correction; verify this arithmetic precisely against the fixture during planning, ideally by also computing it from `callers.json` cross-referenced with a would-be `impact` at depth 1 |
| A4 | The MCP server should call `store.Snapshot()` fresh inside each tool handler rather than once at server construction, even though Phase 3's graph is read-only/frozen for the server's lifetime | Common Pitfalls — Pitfall 2 | Low — this is a forward-compatibility recommendation (Phase 4 will add writes), not a Phase-3 correctness requirement; either choice passes Phase 3's own tests, but caching now creates rework later |

**If this table is empty:** N/A — see entries above. All four are judgment calls this research made to fill gaps the golden fixtures don't fully specify (no `affected`/`files` golden exists per D-07a, and some `status`/`query` fields have no exact TS-to-Go mapping documented anywhere). None are compliance/security/retention-policy claims — all are output-shape judgment calls the planner and/or a parity-test author should confirm against the fixture arithmetic and CONTEXT's stated D-05 principle ("keep key names/shape where meaningful; drop or clearly re-map keys that have no Go analog") before locking the renderer implementation.

## Open Questions

1. **Exact `impact.json` `nodeCount`/`edgeCount` semantics**
   - What we know: The one available fixture (`mergeStyle`, depth 2) gives `nodeCount:5, edgeCount:4`, consistent with "5 distinct nodes visited (mergeStyle + 4 affected), 4 edges traversed to reach them" — but this is inferred from a single data point, not documented in `testdata/golden/README.md` or `ts-schema.sql`.
   - What's unclear: Whether `edgeCount` counts traversed edges, or something else (e.g. total edges among the visited node set, which could differ if visited nodes have edges to each other not on the BFS tree).
   - Recommendation: The planner/executor should derive the exact TS semantics by manually recomputing from `callers.json`+`callees.json`+`impact.json` together (all three exist for `mergeStyle` in this same corpus) before finalizing the `impact` renderer, since re-capturing from a live TS install may not be possible (README: "time-sensitive, one-shot capture").

2. **`status --json` field mapping for TS-only keys with no clean Go/Pebble analog**
   - What we know: `backend`, `journalMode`, `builtWithExtractionVersion`/`currentExtractionVersion` are explicitly called out in CONTEXT D-05 as needing "analogous Go/Pebble-truthful values... drop or clearly re-map keys that have no Go analog" — but CONTEXT does not specify which of drop-vs-remap applies to which key.
   - What's unclear: Concretely, does `journalMode` get dropped (no Pebble user-facing analog) or remapped to something Pebble-specific (e.g. compaction/WAL state)? Does `pendingChanges`/`worktreeMismatch` (Phase-4 concepts) appear as zero-valued placeholders or get omitted in Phase 3?
   - Recommendation: Planner should make this an explicit per-key decision table in the plan (not left to executor improvisation), since it directly determines what the Phase-3 parity test's normalization rules look like. This research's Assumptions Log A2 records a recommended default (present-but-inert) but flags it as non-authoritative.

3. **Whether `IterateNodes(kindPrefix)` should attempt a byte-prefix scan or a full-scan-with-filter (D-03 executor's discretion)**
   - What we know: The current `nodeKey` encoding (length-prefix-then-bytes, `appendSegment`) does not cleanly support a byte-prefix Pebble range scan for "all ids of kind X" without decoding the varint length first (see Pitfall 1).
   - What's unclear: Whether it's worth a small `keys.go` addition (e.g. a `nodeKindIndex` secondary key, or restructuring `nodeKey` to length-prefix `kind` and `hash` as two separate segments so a genuine kind-scoped prefix scan becomes possible) versus just doing the full scan.
   - Recommendation: Full-scan-with-filter for v1 (this research's recommendation, consistent with D-04's stated correctness-first posture for the ≈4k-edge/1223-node golden corpus scale). Revisit only alongside Phase 8's persisted-reverse-index work if profiling on the 100k-file monorepo shows node enumeration is also a bottleneck.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing `internal/query`, `internal/mcp`, and 10 new Cobra commands | ✓ | go1.26.5 darwin/arm64 | — |
| `github.com/mark3labs/mcp-go` (module) | `internal/mcp` stdio server | ✓ (resolves via proxy.golang.org) | v0.56.0 | — |
| `.codegraph/` Pebble store (golden corpus or any indexed repo) | All query commands' manual/integration testing | ✓ — `internal/indexer` + `codegraph init`/`index` from Phase 2 already build one | — | — |
| An MCP client (Claude Code, etc.) for live stdio server testing | Manual verification of MCP-01/02/03 tool-visibility behavior | Available in this dev environment (Claude Code) but not scriptable as an automated Go test | — | Automated tests exercise `internal/mcp`'s `buildServer`/tool-registration logic directly (unit-level, no live client needed); a `checkpoint:human-verify` step is appropriate for actually connecting an agent and observing tool visibility end-to-end |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** live end-to-end MCP client connection (covered by unit-level tests on tool-registration logic plus a human-verify checkpoint for the real stdio handshake).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no testify in project code — `cli_test.go` confirms stdlib-only convention `[VERIFIED: internal/cli/cli_test.go]`) |
| Config file | none — plain `go test` |
| Quick run command | `go test ./internal/query/... ./internal/mcp/... ./internal/cli/...` |
| Full suite command | `go test ./...` (includes `testdata/golden/golden_test.go` and `internal/graphstore/archtest`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| QRY-01 | `query <search>` with `--kind`/`--limit`/`--json` | unit + golden-diff | `go test ./internal/query/... -run TestQuery` | ❌ Wave 0 |
| QRY-02 | `node <symbol\|file>` detail / line-numbered file read | unit + golden-diff | `go test ./internal/query/... -run TestNode` | ❌ Wave 0 |
| QRY-03 | `search` locations-only | unit | `go test ./internal/query/... -run TestSearch` | ❌ Wave 0 |
| QRY-04 | `callers`/`callees` traversal | unit + golden-diff | `go test ./internal/query/... -run TestCallersCallees` | ❌ Wave 0 |
| QRY-05 | `impact --depth` | unit + golden-diff | `go test ./internal/query/... -run TestImpact` | ❌ Wave 0 |
| QRY-06 | `affected [files...]` (D-07 query-time derivation) | unit (no golden — D-07a) | `go test ./internal/query/... -run TestAffected` | ❌ Wave 0 |
| QRY-07 | `files` browse with format/filter/pattern/depth | unit (no golden — D-07a) | `go test ./internal/query/... -run TestFiles` | ❌ Wave 0 |
| QRY-08 | `explore <query>` one-round-trip verbatim source + blast radius | unit + golden-diff (byte-exact markdown) | `go test ./internal/query/... -run TestExplore` | ❌ Wave 0 |
| QRY-09 | `status --json` health/counts | unit + golden-diff (shape-normalized) | `go test ./internal/query/... -run TestStatus` | ❌ Wave 0 |
| MCP-01 | `codegraph_explore` only default-visible tool | unit (server construction, no live client) | `go test ./internal/mcp/... -run TestDefaultToolVisibility` | ❌ Wave 0 |
| MCP-02 | `CODEGRAPH_MCP_TOOLS` allowlist exposes companions | unit | `go test ./internal/mcp/... -run TestAllowlist` | ❌ Wave 0 |
| MCP-03 | Zero tools when no `.codegraph/` | unit | `go test ./internal/mcp/... -run TestNoIndexZeroTools` | ❌ Wave 0 |
| MCP-04 | Tool output shapes match golden corpus | golden-diff (extends `testdata/golden/golden_test.go`) | `go test ./testdata/golden/... -run TestGoldenParity` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/query/... ./internal/mcp/... ./internal/cli/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green (including `archtest` and `golden_test.go`) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/query/engine_test.go` — covers QRY-01…QRY-09's engine-level logic against a small in-repo fixture graph (reuse `internal/indexer/testdata/gofixture` per `cli_test.go`'s existing `copyFixture` pattern)
- [ ] `internal/mcp/server_test.go` — covers MCP-01/02/03 tool-registration logic (construct `*server.MCPServer`, introspect registered tool names — no live stdio transport needed for this)
- [ ] `testdata/golden/golden_parity_test.go` (new file, extends the existing `golden` package) — covers MCP-04: runs the actual CLI commands against the `weft-go` corpus (needs the corpus source tree available locally, same as the existing golden capture assumes) and diffs against `corpus/weft-go/*.json` with the documented D-05 normalizations (id fields ignored, edge-multiplicity tolerance, status field remapping, no `score` key expected)
- [ ] A small local copy/checkout mechanism for `seanb4t/weft` (the golden corpus's source repo) reachable by the parity test — confirm during planning whether `weft` is already vendored somewhere in this repo's testdata or needs to be a submodule/fetch step; `testdata/golden/README.md` says "`weft` is cloned/available separately" — this needs a concrete plan for CI-reproducibility

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Local CLI + local stdio MCP transport; no network-facing auth surface this phase (HTTP/SSE transport is v2/SERVER-01, out of scope) |
| V3 Session Management | No | Stdio is a single-process, single-client transport; no session tokens |
| V4 Access Control | No | OS filesystem permissions gate `.codegraph/` access; no additional app-level ACL this phase |
| V5 Input Validation | Yes | Bound `--depth`/`--limit`/`--max-files` to sane maxima before traversal/allocation (Pitfall 4); validate/clean `-p`/`--path` and `-f`/file args via `filepath.Clean`+confinement rather than passing raw user strings into filesystem reads |
| V6 Cryptography | No | No new crypto surface — node ids already use SHA-256 (Phase 2, D-02a); this phase only reads existing records |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Resource exhaustion via unbounded `impact --depth` BFS or unbounded `--limit`/`--max-files` | Denial of Service | Clamp all numeric flags to documented maxima (e.g. `--depth` ≤ graph node count or a fixed ceiling; `--limit`/`--max-files` ≤ a fixed cap like 1000) before allocating/traversing — reject with a clear error above the ceiling rather than silently truncating |
| Path traversal via `-f`/file args or a crafted node's `filePath` reaching `os.ReadFile` outside the repo root | Tampering / Information Disclosure | Node `filePath` values originate from the trusted, already-indexed graph (not raw user input) — low risk in practice, but the `explore`/`node` "read fresh from disk" step should still resolve the path relative to the resolved `.codegraph/`'s parent repo root and reject any resolved path that escapes it (defense in depth, consistent with `keys.go`'s existing delimiter-injection-guard discipline for ids/paths) |
| MCP tool argument injection (a malicious/compromised MCP client passing malformed `path`/`query` args) | Tampering | `mcp-go`'s schema-validated `Required()`/typed `WithString`/`WithNumber` args plus the same input-validation bounds as the CLI path — MCP handlers must apply identical validation to CLI flag parsing, not looser validation, since D-08b already shares the engine |
| Information disclosure via error messages leaking absolute host filesystem paths back to an MCP client | Information Disclosure | Follow the existing `status.json` convention of normalizing `projectPath`/`indexPath`-shaped values before they'd ever be echoed in an error string reaching an MCP client; low severity for a local dev tool but worth a consistent convention |

## Sources

### Primary (HIGH confidence)
- `internal/graphstore/store.go`, `keys.go`, `pebble_store.go` — `Reader`/`Writer`/`EdgeIterator` interfaces, `n/`/`e/`/`f/`/`m/` keyspace, forward-only edge key design, existing `IterateEdges` implementation pattern to mirror for D-03
- `internal/schema/graph.proto`, `meta.go` — `Node`/`Edge`/`File`/`Meta` field inventory (confirms no schema change needed this phase)
- `internal/cli/root.go`, `init.go`, `index.go`, `uninit.go`, `cli_test.go` — thin-CLI delegation pattern, `targetRoot`/`confirm` helpers, `ErrNotInitialized`/`ErrAlreadyInitialized` idioms, stdlib-testing + `execCmd`/`copyFixture` test convention
- `internal/indexer/nodeid/nodeid.go` — `<kind>:<32-hex-sha256>` id scheme, `appendSegment` encoding (the source of Pitfall 1's kind-prefix-scan caveat)
- `internal/graphstore/archtest/import_graph_test.go` — the `go/packages`-based boundary enforcement `internal/query`/`internal/mcp` must respect (Reader-interface-only, never import `pebble/v2` directly)
- `testdata/golden/README.md`, `golden_test.go`, `ts-schema.sql`, `corpus/weft-go/{status,query,callers,callees,impact,explore,node}.json` — read verbatim in this research session; the literal parity oracle for every JSON/markdown shape claim above
- Context7 `/mark3labs/mcp-go` (fetched 2026-07-11, v0.56.0 current) — `NewMCPServer`/`NewTool`/`AddTool`/`ServeStdio`/`Require*`/`Get*` API surface
- `go list -m github.com/mark3labs/mcp-go@v0.56.0` + `go mod download` against `proxy.golang.org` (executed in this research session) — confirms the module resolves, its `go.mod` declares `go 1.25.5` (compatible with project's `go 1.26.5`)

### Secondary (MEDIUM confidence)
- WebSearch cross-check on the nearest-directory-upward-walk idiom (git `rev-parse --show-toplevel`, `go-findroot`, `go/repo_root` community libraries) — confirms Pattern 4's `filepath.Dir` loop is the standard approach; no official Go stdlib source for this specific idiom exists, so it remains a cross-referenced community pattern rather than an authoritative doc

### Tertiary (LOW confidence)
- This research's own arithmetic inference of `impact.json`'s `nodeCount`/`edgeCount` semantics from a single fixture data point (flagged as Open Question 1 / Assumption A3) — needs manual re-verification against the fixture during planning, not treated as settled
- The `status.json` TS-key-to-Go-analog mapping recommendations (Assumption A2 / Open Question 2) — CONTEXT gives principles ("keep shape where meaningful, drop/remap where not") but not a concrete per-key table; this research's suggested defaults are a starting point for planner confirmation, not a locked mapping

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — one new dependency (`mark3labs/mcp-go`), version-verified against the live module proxy and current Context7 docs; everything else is already-pinned, already-used project infrastructure
- Architecture: HIGH — every pattern (D-03 additive iterator, D-04 in-memory reverse adjacency, D-08a conditional tool registration, D-01a directory walk) is either directly grounded in existing `internal/graphstore` code read in this session, or confirmed against current official `mcp-go` docs
- Pitfalls: HIGH for storage/traversal pitfalls (grounded in reading `keys.go`'s actual encoding); MEDIUM for the two open `status.json`/`impact.json` field-semantics questions, which are honestly flagged as inferred-from-one-fixture rather than documented

**Research date:** 2026-07-11
**Valid until:** 2026-08-10 (30 days — stable domain: internal storage code is frozen/Phase-2-complete, and `mark3labs/mcp-go`'s stdio-server API has been stable across many minor releases; re-verify the pinned `mcp-go` version only if a new release lands before planning executes)
