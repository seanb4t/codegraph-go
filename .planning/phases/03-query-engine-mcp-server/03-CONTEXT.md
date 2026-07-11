# Phase 3: Query Engine & MCP Server - Context

**Gathered:** 2026-07-11
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase opens the **frozen Phase-2 Go graph** to agents and users through the full **read-only query command suite** and a **parity stdio MCP server**. Nothing here writes to or re-indexes the graph — Phase 2 produced the node/edge/id shapes; Phase 3 only *reads* them and renders agent-facing output whose shapes match TS CodeGraph v1.3.1.

**In scope:**
1. **Query command suite** (QRY-01…QRY-09) — `query`, `node`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, and the flagship `explore`, wired into the existing Cobra tree (`internal/cli/root.go`) and delegating to a new read-only `internal/query` engine over `GraphStore.Snapshot()`.
2. **MCP stdio server** (MCP-01…MCP-04) — `codegraph serve --mcp` exposing `codegraph_explore` by default, the 7 companion tools behind the `CODEGRAPH_MCP_TOOLS` allowlist, and zero tools when no `.codegraph/` resolves.
3. **Output-shape parity** verified against the Phase-1 golden corpus (`testdata/golden/corpus/`) for the seven captured tools, with divergences documented and normalized in the parity test.
4. **Two additive `GraphStore.Reader` extensions** the query surface requires (node/file enumeration; the reverse-adjacency read path) — additive to the interface, no keyspace change, no reindex.

**Out of scope (belongs to later phases):**
- Any graph **write, `sync`, incremental reparse, file watching, rename/delete pruning, daemon, staleness banner** — Phase 4 (INDX-03/04, SYNC-*). Phase 3 reads a static graph; a `status` command reports counts/health but does not reconcile filesystem drift.
- **Synthesized / provenance-tagged edges** (interface→impl dispatch, a persisted test-coverage edge type, framework `route` nodes) — Phase 5 (RES-02/RES-03, LANG-07). Phase 3 traverses only the ground-truth AST edges Phase 2 emitted; `affected` *derives* test impact at query time (see D-07) rather than reading a persisted edge.
- **Non-Go languages** — Phase 5. A Go repo's `status.languages` will read `["go"]`; multi-language golden fixtures (colbymchenry-codegraph) are shape references, not Go-extraction targets.
- **`install`/`uninstall` agent wiring, `upgrade`, `help`/`version` ergonomics, `telemetry`** — Phase 6 (AGNT-*, CLI-*). Phase 3 ships `serve` and the query commands only.
- **Embeddings / semantic (vector) search** — permanently deferred for v1 (EMBED-01); `explore`/`query` matching is purely lexical/structural (D-06).
- **100k-file scale, peak-RSS bounding, a persistent reverse-edge index** — Phase 8 (INDX-06, PERF-*). Phase 3 targets correctness + parity on the mid-size golden corpus; the query-time reverse scan (D-04) is deliberately un-optimized.

</domain>

<decisions>
## Implementation Decisions

> Auto-resolved in `--auto` mode. Each decision is the recommended default grounded in: the Phase-1/2 substrate (`GraphStore`/`Reader`, `n/`/`e/`/`f/` keyspace, node-id/edge-dedup shapes), the captured golden corpus (`testdata/golden/`), the Phase-3 ROADMAP success criteria, and the technology-stack research in `.claude/CLAUDE.md`.

### Command Surface & CLI Wiring (QRY-01…09, MCP-01)
- **D-01:** Add the query subcommands and `serve` to the **existing Cobra tree** in `internal/cli/root.go` (alongside `init`/`index`/`uninit`). Follow Phase 2's **thin-CLI pattern**: commands resolve paths + flags and delegate all logic to a new read-only **`internal/query`** engine; the MCP server lives in a new **`internal/mcp`** package that calls the same engine/formatters. Commands: `query`, `node`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, `explore`, `serve`.
- **D-01a:** Every query command accepts **`-p`/`--path`** (default cwd) and resolves the **nearest `.codegraph/` at or above** that path (matches the MCP server's documented resolution behavior). No `.codegraph/` → a clear "not initialized" error (reuse the `internal/cli` `ErrNotInitialized` idiom); the MCP server handles the no-index case differently (D-08a).
- **D-01b:** **Output flags mirror the TS capture surface** (parity is measured against how the golden was captured): `--json` on the *structured* commands (`query`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`); `explore` and `node` emit **markdown text only** (the golden wraps them as `{command, output}` precisely because TS has no native `--json` for them). Per-command flags per requirements: `--kind` (query/search), `--limit` (query/search/callers/callees), `--depth` (impact), plus format/filter/pattern/depth for `files` (QRY-07).

### Read Path Over the Frozen Graph (INDX-05 contract; the load-bearing decisions)
- **D-02:** The query engine is **strictly read-only over `GraphStore.Snapshot()`** — one `Reader` snapshot per CLI invocation; the MCP server takes a **fresh snapshot per tool call** (consistent point-in-time read, and forward-compatible with Phase-4 background writes). No writes, no reindex, no interface bypass — the `internal/graphstore/archtest` boundary still holds.
- **D-03:** **Additively extend the `Reader` interface** with node/file enumeration — the current `Reader` (`GetNode`/`GetFile`/`GetMeta`/`IterateEdges`) has **no way to list nodes**, which `query`/`search`/`files`/`status.nodesByKind`/`status.languages` all require. Add iterator methods over the existing **`n/` (optionally kind-scoped) and `f/`** prefixes (e.g. `IterateNodes(kindPrefix string)`, `IterateFiles()`), reusing the D-03 range-scan keyspace design from Phase 1. This is **additive to the interface only** — no key change, no schema bump, no reindex, archtest unaffected. Exact signatures are executor's discretion.
- **D-04:** **Reverse traversal (`callers`, `impact`, `affected`)** is computed at query time: the store indexes edges **forward only** (`e/<src>/<kind>/<dst>`), so build an **in-memory reverse-adjacency map by scanning all forward edges once** (`IterateEdges("")`) per invocation, then traverse it. Forward traversal (`callees`) uses `IterateEdges(src)` directly — no scan. **No persisted reverse-edge index** in Phase 3 (that would touch Phase 2's frozen writer + require a reindex); a persistent reverse index is deferred to **Phase 8** if profiling on the 100k-file monorepo shows the scan dominates. Correctness-first: for the golden corpus (≈4k edges) the scan is trivial.

### Parity Model (MCP-04)
- **D-05:** **Parity = output-shape / key-name / semantic-structure parity against the golden corpus — NOT byte-identical values.** The parity test normalizes/ignores these **documented, expected divergences** from TS:
  - **Node ids differ** — Phase-2 D-02a ids are `<kind>:<sha256-trunc>` vs TS `<kind>:<md5>`; compare records on **stable fields** (name, kind, filePath, line), not `id`. (`file:` nodes use `file:<path>` in both — those match.)
  - **Edge dedup** — Phase-2 D-05 keeps one representative call site per `(src,kind,dst)`; callers/impact lists may differ in multiplicity where TS's `(source,target,kind,line,col)` index kept distinct sites. Documented, not a bug.
  - **`status` backend fields** — `backend`/`journalMode`/`builtWithExtractionVersion`/`currentExtractionVersion` are TS-SQLite-isms. Emit **analogous Go/Pebble-truthful values** (e.g. a pebble backend identifier, schema version from `schema.Meta`), keeping key *names/shape* where meaningful; drop or clearly re-map keys that have no Go analog.
  - **`query` `score`** — omitted (already stripped from the golden as volatile; our lexical ranking emits no score).
  - **`languages`/`nodesByKind`** — reflect **Go-only** extraction until Phase 5 (a Go repo reads `["go"]`).
- **D-05a:** **`explore` markdown template** reproduces the golden `explore.json` output structure in shape: `**Exploration: <query>**` → `Found N symbol(s) across M file(s)` → `**Blast radius — what depends on these (update/verify before editing)**` bullets (callers count grouped by file + `tests:` list) → the `**Source Code**` **verbatim-source disclaimer paragraph copied from the golden** → per-file `` **`path`** — sym(kind) `` blocks with fenced, **tab-indented, line-numbered** source read fresh from disk. The disclaimer text is an agent-facing contract — reproduce it, don't paraphrase.
- **D-05b:** **`node` markdown template** reproduces the golden `node.json` shape: `**name** (kind)` → `**Location:**` → `**Signature:**` → `**Trail — codegraph_node any of these to follow it (no Read needed)**` → `**Calls →**` (forward edges) / `**Called by ←**` (reverse edges) with `name (file:line)` entries. `node <symbol>` → this detail; `node <file>` → line-numbered file read (QRY-02).

### Search / Explore Matching — no embeddings (QRY-01, QRY-03, QRY-08)
- **D-06:** `query`/`search`/`explore` matching is **deterministic lexical matching** over `name` + `qualifiedName` (substring/token match, `--kind` filter, `--limit` cap) — **no embeddings / vector search** (EMBED-01 is a deferred future milestone; v1 is pure structural). TS's FTS5/BM25 `score` is **not** reproduced (stripped as volatile); ranking uses a **deterministic tie-break** (e.g. exact-name > prefix > substring, then stable sort by qualifiedName/id) so `--json` output is byte-reproducible. `query` returns full node records (with `signature`, `visibility`, etc.); **`search` returns locations-only** (name/kind/filePath/startLine — no source body), the lightweight variant.

### `affected` / `files` — no golden oracle (QRY-06, QRY-07)
- **D-07:** **`affected [files...]` derives impacted test files at query time** from existing `calls` edges + a **test-file heuristic** (`_test.go` suffix / `Test*`/`Benchmark*` naming), using the reverse adjacency from D-04. It **does NOT persist a new "test-coverage edge type."** ⚠ **Documented divergence from QRY-06's literal wording** ("via a dedicated test-coverage edge type"): Phase 2 froze the edge vocabulary to ground-truth AST edges, and synthesized/provenance-tagged edges are **Phase 5** (RES-02/RES-03) — persisting a test-coverage edge here would require reindexing the frozen graph and pulling Phase-5 work forward. The **user-facing behavior** (list impacted test files for changed files) is fully preserved; only the *mechanism* is query-time derivation. **The planner/researcher should confirm this is acceptable** rather than blocking on a persisted edge.
- **D-07a:** **`affected` and `files` have no golden fixture** — the corpus captured only `explore/node/query/callers/callees/impact/status`. Their parity is **best-effort against TS CLI behavior/documentation**, not corpus-diffed. Do not fabricate a golden fixture for them; assert on structural correctness (right files identified, right browse output) instead.

### MCP Server (MCP-01…MCP-04)
- **D-08:** Build the stdio MCP server on **`github.com/mark3labs/mcp-go`** (`server.NewMCPServer` + `mcp.NewTool` + `server.ServeStdio`), per `.claude/CLAUDE.md` §MCP — pure-Go, adds no CGo. Add it to `go.mod`. Re-evaluate the official `modelcontextprotocol/go-sdk` at the next milestone boundary; the tool-registration seam keeps that a bounded swap.
- **D-08a:** **`codegraph serve --mcp`** runs the stdio server (stdio only in v1; HTTP/SSE is v2 SERVER-01). **Tool visibility:** `codegraph_explore` is the **only default-visible tool**; the 7 companions (`node`, `search`, `callers`, `callees`, `impact`, `files`, `status`) register **only when named in `CODEGRAPH_MCP_TOOLS`** (comma-separated allowlist; unknown names ignored with a stderr warning). When **no `.codegraph/` resolves**, the server still starts and completes MCP init but advertises **zero tools** (MCP-03 — agents fall back to built-ins gracefully; the server does not crash or refuse the connection).
- **D-08b:** MCP tools **reuse the exact CLI query engine + formatters** so tool output shapes match the golden corpus (MCP-04) with no second code path. Each tool call takes a **fresh `Snapshot()`** (D-02). Tool arg schemas mirror the CLI flags (query string, path, limit, depth, kind, `max-files` for explore).

### Claude's Discretion
- Exact `IterateNodes`/`IterateFiles` signatures and whether kind-filtering is a key-prefix scan or a full-node scan with in-memory filter.
- The lexical ranking algorithm's precise scoring/tie-break, and the in-memory adjacency data structures.
- How `status.nodesByKind` / `languages` are computed (node scan vs. an added Meta breakdown — prefer scan to avoid touching the writer).
- Markdown rendering helper structure; where the `.codegraph/`-resolution helper lives (shared `internal/cli` vs `internal/query`).
- Whether `serve` is a top-level command with `--mcp` or the sole mode; concrete MCP tool names beyond the fixed `codegraph_explore`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher, planner, executor) MUST read these before planning or implementing.**

### Phase 1/2 Substrate (bind directly against these — do not re-derive)
- `internal/graphstore/store.go` — the `GraphStore`/`Reader`/`EdgeIterator`/`Writer` interfaces. Phase 3 reads via `Snapshot()`→`Reader`; **note the Reader has GetNode/GetFile/GetMeta/IterateEdges only — D-03 adds node/file enumeration here.**
- `internal/graphstore/keys.go` — the `n/<id>`, `e/<src>/<kind>/<dst>` (forward-only, deduped), `f/<path>`, `m/` keyspace. The forward-only edge ordering is why D-04 scans for reverse traversal; the `n/`/`f/` prefixes are what D-03's iterators range-scan.
- `internal/schema/graph.proto` + `internal/schema/graph.pb.go` — `Node`/`Edge`/`File`/`Meta` record shapes the query output is rendered from (`signature`, `docstring`, `visibility`, `is_exported`, `return_type`, edge `provenance`).
- `internal/schema/meta.go` — `Meta` (schema version + counts + health) that `status` reports from.
- `internal/cli/root.go` + `internal/cli/{init,index,uninit}.go` — the Cobra tree D-01 extends and the thin-CLI-delegation pattern to mirror.
- `internal/indexer/nodeid/nodeid.go` — the `<kind>:<sha256-trunc>` id scheme (D-02/D-02a) behind the D-05 id divergence.

### Golden Corpus — the parity oracle (MCP-04)
- `testdata/golden/README.md` — corpus provenance, the **volatile-field stripping rules** (score, `*_at`, `dbSizeBytes`, paths) the parity test must respect, and the edge-dedup bug note (#1034) behind D-05.
- `testdata/golden/corpus/weft-go/{explore,node,query,callers,callees,impact,status}.json` — the **exact output shapes** Phase 3 must reproduce (primary Go-repo oracle for D-05/D-05a/D-05b).
- `testdata/golden/corpus/colbymchenry-codegraph/*.json` — multi-language shape reference (shape only; not a Go-extraction target).
- `testdata/golden/golden_test.go` — the existing fixture-invariant test; the Phase-3 parity test extends this harness.
- `testdata/golden/ts-schema.sql` — TS `nodes`/`edges`/`status` column vocabulary; the field-name reference for `status`/`query` output keys.

### Project Planning & Decisions
- `.planning/ROADMAP.md` §"Phase 3: Query Engine & MCP Server" — the four success criteria this CONTEXT must satisfy.
- `.planning/REQUIREMENTS.md` — **QRY-01…QRY-09**, **MCP-01…MCP-03** are the locked contract; **MCP-04** (corpus captured) and INDX-05/RES-01/LANG-01 (Phase 1/2, complete) are the substrate.
- `.planning/phases/02-go-indexing-pipeline/02-CONTEXT.md` — Phase-2 decisions D-02 (node id), **D-05 (edge dedup)**, D-06 (node/edge vocabulary) that Phase-3 queries traverse and D-05-parity documents as divergences.
- `.planning/phases/01-foundation-storage-schema-parser-strategy/01-CONTEXT.md` — Phase-1 D-03 (keyspace range-scan design D-03 reuses) and D-04 (snapshot/reader concurrency model D-02 relies on).
- `.claude/CLAUDE.md` §"MCP" / §"Alternatives Considered" — the `mark3labs/mcp-go` selection (D-08) and the official-SDK re-evaluation note.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`GraphStore` + Pebble impl (`internal/graphstore`)** — fully built, concurrency-tested. Phase 3 opens a `Snapshot()` per query and reads; `Export()` is not needed. **Gap:** the `Reader` cannot enumerate nodes or traverse edges in reverse — D-03/D-04 close these within the read path (D-03 additively, D-04 in-memory).
- **Cobra CLI (`internal/cli`)** — `root.go` wires subcommands; `init/index/uninit` model the thin command → engine delegation, path resolution, and `ErrNotInitialized`/`ErrAlreadyInitialized` error idioms Phase 3 reuses.
- **Schema records (`internal/schema`)** — `Node.signature/visibility/is_exported/return_type` and `File`/`Meta` carry everything `query`/`node`/`status` render; no schema change needed.
- **Golden harness (`testdata/golden/golden_test.go` + `capture.sh`)** — the parity test extends this; volatile-field rules already encoded.

### Established Patterns
- **Interface-boundary discipline** (`internal/graphstore/archtest`) — only `internal/graphstore` imports Pebble. `internal/query` and `internal/mcp` MUST depend only on the `GraphStore`/`Reader` interface; D-03's additions stay inside that boundary.
- **RED→GREEN atomic commits** — write the failing parity/golden-diff test before the implementing code (Phase 1/2 convention the planner should carry forward).
- **Determinism as a first-class gate** — Phase 2 made from-scratch indexing byte-identical; Phase 3 keeps `--json` output deterministic (D-06 ranking tie-break) so the parity diff is stable.

### Integration Points
- **CLI/MCP → `internal/query` → `GraphStore.Snapshot()`** — the single read seam; MCP tools and CLI commands are two front-ends over one engine + one set of formatters (D-08b), which is what makes MCP-04 parity hold without a second code path.
- **Query output → golden corpus** — the parity test is the acceptance gate for success-criterion 4; every command with a fixture diffs against it (shape, per D-05).
- **`go.mod`** — adds `github.com/mark3labs/mcp-go` (pure-Go; the only new direct dependency this phase introduces).

</code_context>

<specifics>
## Specific Ideas

- **The golden corpus IS the spec for output shape.** `explore.json`/`node.json` embed the exact markdown templates (headers, the blast-radius bullet format, the verbatim-source disclaimer paragraph) — treat them as the literal contract, not inspiration. `status.json`/`query.json`/`callers.json`/`callees.json`/`impact.json` are the JSON key/shape contract.
- **Two Reader gaps surfaced during scouting are the phase's technical spine** (D-03 node enumeration, D-04 reverse adjacency). Surface them to the planner first — the query commands can't be built until the read path exists, and both are deliberately kept additive/in-memory to avoid touching the frozen Phase-2 graph.
- **One engine, two front-ends.** CLI and MCP must share the query engine and formatters (D-08b) so parity is proved once. Resist a separate MCP-only rendering path.
- **`affected`/`files` are the loosest-specified commands** (no golden, D-07 divergence on QRY-06). Flag D-07's query-time-derivation choice for explicit confirmation during planning rather than silently persisting a new edge type.

</specifics>

<deferred>
## Deferred Ideas

- **Incremental `sync`, native file watchers, rename/delete pruning, daemon, staleness banner, MCP reconnect reconciliation** — Phase 4 (INDX-03/04, SYNC-*). Phase 3's `status` reports health but does not reconcile drift.
- **Persisted reverse-edge index** — Phase 8 scale work; adopt only if the D-04 query-time scan profiles as a bottleneck on the 100k-file monorepo.
- **Synthesized dispatch edges, `provenance: heuristic`, a persisted test-coverage edge type, framework `route` nodes** — Phase 5 (RES-02/RES-03, LANG-07). `affected` derives test impact at query time instead (D-07).
- **Non-Go language query output** (multi-language `status.languages`/`nodesByKind`) — Phase 5 (LANG-02+). Phase 3 proves the query surface on the Go graph.
- **Embeddings / semantic search over `explore`/`query`** — future milestone (EMBED-01); v1 stays lexical/structural (D-06).
- **MCP HTTP/SSE transport, remote/multi-user queries** — v2 (SERVER-01). Phase 3 ships stdio only.
- **`install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry`** — Phase 6 (AGNT-*, CLI-*). Phase 3 ships `serve` + query commands.

None of the above are scope creep into Phase 3 — they are correctly-placed future work, recorded so nothing is lost.

</deferred>

---

*Phase: 3-Query Engine & MCP Server*
*Context gathered: 2026-07-11*
