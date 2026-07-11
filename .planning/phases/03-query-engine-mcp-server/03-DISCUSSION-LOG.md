# Phase 3: Query Engine & MCP Server - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-11
**Phase:** 3-Query Engine & MCP Server
**Mode:** `--auto` (all gray areas auto-selected; recommended default chosen per question)
**Areas discussed:** Command surface & CLI wiring, Read path & Reader additions, Parity model, explore/node markdown templates, Search matching, affected/files behavior, MCP server

---

## Read Path — node enumeration

| Option | Description | Selected |
|--------|-------------|----------|
| Additive Reader iterators | Add `IterateNodes`/`IterateFiles` over the frozen `n/`/`f/` keyspace — no key change, no reindex | ✓ |
| Add persisted per-kind index at index time | Precompute node lists; requires touching Phase-2 writer + reindex | |
| Full-store raw scan bypassing Reader | Violates the archtest interface boundary | |

**Choice:** Additive Reader iterators (recommended default).
**Notes:** Current `Reader` (GetNode/GetFile/GetMeta/IterateEdges) cannot list nodes; `query`/`search`/`files`/`status` require it. Additive-to-interface keeps the archtest boundary and needs no reindex.

---

## Read Path — reverse traversal (callers/impact/affected)

| Option | Description | Selected |
|--------|-------------|----------|
| Query-time in-memory reverse adjacency | Scan forward edges once (`IterateEdges("")`), build reverse map per invocation | ✓ |
| Persisted reverse-edge index | Write `re/<dst>/<kind>/<src>` at index time; requires reindex + keyspace change | |

**Choice:** Query-time in-memory reverse adjacency (recommended default).
**Notes:** Store indexes edges forward-only. Correctness-first; ≈4k edges in the corpus makes the scan trivial. Persistent index deferred to Phase 8 if the 100k-file monorepo profiles it as a bottleneck.

---

## Parity Model

| Option | Description | Selected |
|--------|-------------|----------|
| Shape/key + semantic parity, documented divergences | Compare output shapes vs golden, normalize known value divergences (ids, edge dedup, backend fields, score) | ✓ |
| Byte-identical value parity | Impossible — SHA-256 ids ≠ TS MD5; Pebble ≠ SQLite backend fields | |

**Choice:** Shape/key + semantic parity (recommended default).
**Notes:** Divergences documented in CONTEXT D-05: node-id hash algo, edge dedup multiplicity, `status` backend/journal keys, stripped `score`, Go-only languages.

---

## explore / node markdown templates

| Option | Description | Selected |
|--------|-------------|----------|
| Reproduce golden markdown verbatim-in-shape | Match explore.json/node.json headers, blast-radius bullets, verbatim-source disclaimer | ✓ |
| Design a new/cleaner agent output format | Breaks parity with the captured oracle | |

**Choice:** Reproduce golden markdown (recommended default).
**Notes:** The disclaimer paragraph and trail/blast-radius formatting are the agent-facing contract; copy, don't paraphrase.

---

## Search / Explore matching

| Option | Description | Selected |
|--------|-------------|----------|
| Deterministic lexical matching | name/qualifiedName substring+token match, kind filter, deterministic tie-break | ✓ |
| Reproduce TS FTS5/BM25 scoring | Requires SQLite FTS; score was stripped as volatile anyway | |
| Embedding / vector search | Out of scope for v1 (EMBED-01 deferred) | |

**Choice:** Deterministic lexical matching (recommended default).
**Notes:** `query` returns full node records; `search` returns locations-only. No `score` in output; deterministic ranking keeps `--json` reproducible.

---

## affected / files behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Derive test impact at query time from calls edges + `_test.go` heuristic | No persisted edge; keeps Phase-2 graph frozen | ✓ |
| Persist a dedicated test-coverage edge type (QRY-06 literal) | Requires reindex + edge-vocab change; synthesized edges are Phase 5 | |

**Choice:** Query-time derivation (recommended default). ⚠ Documented divergence from QRY-06's literal wording — flagged for planner confirmation.
**Notes:** `affected`/`files` have no golden fixture; parity is best-effort vs TS CLI behavior, not corpus-diffed.

---

## MCP server

| Option | Description | Selected |
|--------|-------------|----------|
| `mark3labs/mcp-go` stdio | Per stack research; broadest Go adoption; pure-Go, no CGo | ✓ |
| Official `modelcontextprotocol/go-sdk` | Newer, less Go-server mileage; re-evaluate at next milestone | |

**Choice:** `mark3labs/mcp-go` (recommended default).
**Notes:** `serve --mcp` stdio only; `codegraph_explore` default-visible; 7 companions behind `CODEGRAPH_MCP_TOOLS`; zero tools when no `.codegraph/`. Tools reuse the CLI engine/formatters for MCP-04 parity.

## Claude's Discretion

- Exact iterator signatures, lexical ranking scoring, in-memory adjacency structures.
- `status.nodesByKind`/`languages` computation (scan vs Meta breakdown).
- Markdown rendering helpers; `.codegraph/`-resolution helper location.
- Whether `serve` is a top-level command with `--mcp` or the sole mode.

## Deferred Ideas

- Persisted reverse-edge index → Phase 8 (scale).
- Synthesized dispatch edges / persisted test-coverage edge / provenance → Phase 5.
- `sync`/watcher/daemon/staleness/reconnect → Phase 4.
- Embeddings/semantic search → future milestone (EMBED-01).
- MCP HTTP/SSE / remote queries → v2 (SERVER-01).
- `install`/`upgrade`/`help`/`version`/`telemetry` → Phase 6.
