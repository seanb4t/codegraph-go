# Phase 1: Foundation — Storage, Schema & Parser Strategy - Context

**Gathered:** 2026-07-10
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the **substrate**, not any user-facing feature. Three things ship:

1. A **schema-versioned graph store** behind a `GraphStore` interface — concurrent lock-free readers alongside a single coordinated writer (INDX-05), with an on-disk format that round-trips future annotation fields and a bulk export without a format break (ARCH-01).
2. A **chosen, benchmarked parser strategy** — CGo tree-sitter vs wazero WASM decided from a head-to-head spike, with parse throughput and static-build impact documented as the basis for the decision.
3. **Captured TS ground-truth** — a golden-output corpus and the TS `.codegraph/` SQLite schema DDL captured from the live TS CodeGraph v1.3.x, so later phases measure parity against ground truth rather than memory.

**In scope:** the storage engine + schema + `GraphStore` boundary; the parser spike + decision + the narrow parser interface; the ground-truth capture harness and its fixtures.

**Out of scope (belongs to later phases):** the indexer/extractors that populate the graph (Phase 2), any query command or MCP tool (Phase 3), the file watcher/daemon (Phase 4), additional languages (Phase 5), and the migration tool that reads TS SQLite (Phase 7 — Phase 1 only *captures* the DDL, it does not convert). Do not build extraction or query logic here; build the store and lock the parser.

</domain>

<decisions>
## Implementation Decisions

> Auto-resolved in `--auto` mode. Each decision takes the recommended default from the technology-stack research in `.claude/CLAUDE.md` and the founding decisions in engram (`repo:codegraph-go`). Decisions marked **pending spike** are deliberately left to be settled by a benchmark inside this phase, not prejudged.

### Storage Engine
- **D-01:** Use **`github.com/cockroachdb/pebble`** as the embedded KV store backing the new graph format. Rationale: pure Go (no CGo — keeps the single-static-binary story), snapshots for consistent reads while a background re-index writes, range deletes for pruning a stale file/symbol subgraph on re-index, and `IndexedBatch` (read-your-writes) that maps onto graph-mutation semantics. Rejected: **bbolt** (single-writer transaction model is a structural bottleneck against "optimized for concurrent access"), **Badger** (WiscKey shines only for large values; the graph's dominant payload is small structured records — revisit only if profiling shows large stored snippets dominate write-amplification). See `.claude/CLAUDE.md` §"Storage — the new graph format" and §"Alternatives Considered".

### Record Encoding & Schema Evolution
- **D-02:** Encode node/edge/file records with **Protocol Buffers** (`google.golang.org/protobuf`), and stamp a **`schema_version`** in a dedicated `meta` record. Rationale: field-numbered, forward/backward-compatible encoding is the mechanism that makes ARCH-01 true — future annotation fields (embedding vectors, community/cluster assignments) get **reserved field numbers** and are added without a format break, and unknown fields from a newer writer survive an older reader. Protobuf is a well-audited pure-Go dependency, acceptable under the "minimal audited deps" constraint. A `version/export` test asserts an old-schema record round-trips through a new-schema reader and that a bulk export re-imports losslessly.
- **D-02a:** Schema evolution discipline is **additive-only within a major schema version** (never renumber or reuse a field number; deprecate by reserving). A `schema_version` bump is required only for a genuinely breaking layout change — which v1 must avoid.

### Keyspace Layout
- **D-03:** Lay out the KV keyspace as **prefix-namespaced typed keys**, e.g. `meta/…` (schema version, node/edge counts, last-sync, index health), `n/<node-id>` (nodes), `e/<src>/<type>/<dst>` (edges, ordered so callers/callees/impact are range scans), `f/<path>` (file records + content hash), and a **reserved `a/…` annotation namespace** for post-v1 embeddings/communities. Rationale: edge ordering by source enables lock-free forward/reverse traversal via range scans; a whole file's subgraph is prunable with a single Pebble range delete (feeds Phase 4's rename/delete pruning); the reserved annotation prefix means future features bolt on without touching existing records (ARCH-01). Exact id/edge-key byte encoding is planner/executor discretion as long as it preserves range-scan and range-delete properties.

### GraphStore Interface & Concurrency
- **D-04:** All graph reads and writes go through a **`GraphStore` interface** — no other package imports the KV engine directly. The interface exposes: **snapshot-based reads** (a consistent view for a query or MCP call even while a re-index writes), **batched writes** (one `IndexedBatch` per file-change / debounce window rather than per-symbol writes), **iterators** for range scans (callers/callees/impact/files), and an explicit **bulk graph export** method (ARCH-01). Concurrency model: **many lock-free readers via Pebble snapshots + one coordinated writer** owned by the store.
- **D-04a:** A **concurrency/architecture test** verifies (a) concurrent readers run correctly alongside a single writer, and (b) **no package bypasses the interface** to reach the engine (e.g., an import-graph / `go/packages` assertion that only the store package imports `pebble`). This is the enforceable form of INDX-05.

### Parser Strategy Spike
- **D-05:** Run a **head-to-head spike** on **Go (LANG-01) plus one external-scanner grammar** (Python or C# — a grammar whose C scanner exercises the crash-isolation tail-risk), on a real mid-size repo. Measure: **parse throughput**, **incremental-reparse** time, **static-build impact** (does it break `CGO_ENABLED=0`; cross-compile complexity across target platforms), and **crash isolation** (does a malformed input take down the host process). **pending spike**
- **D-05a:** **Decision criterion:** default to **Option A — CGo tree-sitter (`tree-sitter/go-tree-sitter` + per-language grammar modules)** for v1 — per research it is the only path with full 12-language coverage *and* top-tier performance. Adopt **Option B — wazero WASM grammars** only if the spike shows its parse-time overhead is invisible against the full indexing pipeline **and** the grammar-to-WASM compilation cost is acceptable. **Option C (native pure-Go tree-sitter)** is monitor-only, not for v1.
- **D-05b:** Regardless of outcome, the parser sits behind a **narrow interface** (`Parser.Parse([]byte, *Tree) (*Tree, error)`-shaped) from day one, so a later CGo↔wazero swap is a backend change, not an architecture change. If CGo is selected, it is the **single documented CGo exception** to the pure-Go / minimal-deps constraint (feeds DIST-05).

### Golden Corpus & TS DDL Capture
- **D-06:** While the live TS CodeGraph v1.3.x is still available, capture two artifacts: **(a)** the TS `.codegraph/` **SQLite schema DDL** (`.schema` and a representative `.dump`) from a real aged index, and **(b)** **golden JSON snapshots** of `codegraph_explore` and the companion tool outputs on a small, **pinned** corpus. Store both as **version-pinned fixtures under `testdata/golden/`**, recording the exact TS version (v1.3.x) and capture date. Rationale: this is the ground truth Phase 3 (MCP output-shape parity, MCP-04), Phase 4 (sync), and Phase 7 (migration reader) diff against — capture must happen now, before the live TS version drifts or is uninstalled.
- **D-06a:** Corpus selection: a **compact Go repo** (aligns with Phase 2's first language) **plus the TS `colbymchenry/codegraph` repo itself** (multi-language, exercises the tool surface broadly). Keep it small enough to commit and re-run deterministically.

### Claude's Discretion
- Exact byte encoding of node ids and edge keys (as long as range-scan + range-delete properties from D-03 hold).
- The precise protobuf message shapes for node/edge/file records (planner/executor to design against the TS schema DDL captured in D-06).
- Spike harness structure and which specific real repo is used for benchmarking (D-05), provided it includes Go + one external-scanner language.
- Whether `meta` counts are maintained incrementally or recomputed — a Phase 2+ concern, not locked here.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher, planner, executor) MUST read these before planning or implementing.**

### Technology-Stack Research (primary — this IS the Phase 1 research doc)
- `.claude/CLAUDE.md` §"Storage — the new graph format" — Pebble selection rationale (concurrency, snapshots, range deletes, `IndexedBatch`).
- `.claude/CLAUDE.md` §"The Parser Decision" — Options A (CGo, recommended v1), B (wazero WASM, Phase-1 spike), C (native, monitor-only); the spike mandate and decision framing behind D-05.
- `.claude/CLAUDE.md` §"Alternatives Considered" and §"What NOT to Use" — rejected engines/drivers/parsers and the conditions under which to reconsider (Badger, bbolt, `smacker/go-tree-sitter`, `mattn/go-sqlite3`).
- `.claude/CLAUDE.md` §"Version Compatibility" — `go-tree-sitter@v0.25.x` ↔ grammar module pinning; Pebble ↔ Go version; `modernc.org/sqlite` ↔ TS `.codegraph/` schema (the DDL captured in D-06 must be readable by that driver in Phase 7).

### Project Planning
- `.planning/ROADMAP.md` §"Phase 1" — phase goal + the four success criteria this CONTEXT must satisfy.
- `.planning/REQUIREMENTS.md` — **INDX-05** (concurrent store behind `GraphStore`) and **ARCH-01** (schema-versioned, annotation/export-ready) are the locked contract for this phase.
- `.planning/PROJECT.md` §"Constraints" + §"Key Decisions" — single-static-binary, minimal-audited-deps, CGo-only-with-justification, parity-first, new-format-plus-migration.

### External Ground Truth
- **TS CodeGraph v1.3.x** — `https://github.com/colbymchenry/codegraph` (live install + source). Source of the SQLite schema DDL and golden tool-output corpus captured in D-06. MIT-licensed; port with attribution.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **None — greenfield repository.** No Go source, no `go.mod`, no `.planning/codebase/` maps exist yet. This phase creates the module and the first packages (`graphstore`, `parser`, and a ground-truth capture harness).

### Established Patterns
- **None in-repo yet.** Patterns for this phase come from the technology-stack research in `.claude/CLAUDE.md`, not from existing code. The planner should treat that document as the pattern source.

### Integration Points
- The `GraphStore` interface (D-04) is the integration seam every later phase binds to: Phase 2 (indexer writes), Phase 3 (query/MCP reads), Phase 4 (sync writes + snapshots), Phase 7 (migration writes). Its surface is the single most consequential design output of this phase.
- The narrow `Parser` interface (D-05b) is the seam that isolates the CGo/wazero decision from the rest of the pipeline.

</code_context>

<specifics>
## Specific Ideas

- **Capture-first for ground truth:** the golden corpus + SQLite DDL (D-06) must be grabbed while Sean's live TS CodeGraph v1.3.x is still running — this is a time-sensitive, one-shot opportunity that gates parity verification in Phases 3/4/7.
- **The parser decision is honestly deferred:** engram founding memory records parser strategy as "unresolved pending research." CONTEXT locks the spike *method and criteria* (D-05/D-05a/D-05b), not the outcome. The planner should schedule the spike early in the phase so its result is available before any code hard-binds to a parser backend.

</specifics>

<deferred>
## Deferred Ideas

- **Embeddings / vector search, community detection, graph-visualization UI** — post-v1 milestones. v1 only *anticipates* them: reserved annotation namespace (D-03), forward-compatible record encoding (D-02), and a bulk export method (D-04). Do not implement them in Phase 1.
- **The actual TS→new-format migration converter** — Phase 7 (MIGR-01/02). Phase 1 captures the SQLite DDL as ground truth (D-06); it does not read or convert TS indexes.
- **Indexer, extractors, query commands, MCP tools, file watcher** — Phases 2–4. Phase 1 provides the store and parser they build on, nothing more.
- **Team/server features (central server, CI-distributed indexes)** — v2. Architecture must *not preclude* them (concurrency-capable store, versioned schema), but they are not built here.

None of the above are scope creep into Phase 1 — they are correctly-placed future work, recorded so nothing is lost.

</deferred>

---

*Phase: 1-Foundation — Storage, Schema & Parser Strategy*
*Context gathered: 2026-07-10*
