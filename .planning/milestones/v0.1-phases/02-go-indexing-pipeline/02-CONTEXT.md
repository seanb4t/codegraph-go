# Phase 2: Go Indexing Pipeline - Context

**Gathered:** 2026-07-10
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the **first end-to-end indexer**: a user runs `codegraph init` (or `index`) in a Go repository and gets a correct, cross-file-resolved, queryable graph built from scratch — proving the two-pass `parallel-extract → sequential-resolve` mechanism on the first language (Go, LANG-01), on top of the Phase 1 `GraphStore` and `Parser` substrate.

**In scope:**
1. **CLI lifecycle** — `codegraph init` / `index` / `uninit` (Cobra), with `--force`, `--quiet`, `--verbose` and the `.codegraph/` directory create/remove semantics (INDX-01, INDX-02).
2. **Go extraction** — a file-walk + tree-sitter (CGo backend) Go extractor emitting the parity node/edge vocabulary (LANG-01).
3. **Two-pass engine** — parallel per-file extract producing nodes/intra-file edges/unresolved refs, then a sequential resolve pass that links imports, call edges, and Go "type inheritance" (embedding/type references) across a multi-package repo, writing through `GraphStore` (RES-01).
4. **Two Phase-1-deferred decisions settled here** — the node-identity scheme and the edge-key multiplicity choice, because the extractor cannot be built without them.

**Out of scope (belongs to later phases):**
- Any query command or `explore`/MCP output surface — **Phase 3** (Phase 2 must produce a graph that *would* answer those queries correctly; it does not build the query/MCP layer or diff golden outputs itself).
- Incremental `sync`, file watching, rename/delete pruning, daemon — **Phase 4**.
- Non-Go languages, **interface→implementation dispatch synthesis**, provenance-tagged heuristic edges, and framework-aware routing — **Phase 5** (RES-02/RES-03/LANG-02+). Phase 2 emits **only ground-truth AST edges** for Go.
- Migration from TS SQLite — **Phase 7** (Phase 1 captured the DDL; Phase 2 only depends on the TS id/vocabulary *shape* for parity, it does not read TS indexes).
- 100k-file scale / peak-RSS gates — **Phase 8** (Phase 2 targets correctness on a real mid-size Go repo, not monorepo-scale memory tuning).

</domain>

<decisions>
## Implementation Decisions

> Auto-resolved in `--auto` mode. Each decision takes the recommended default grounded in the Phase 1 substrate (`GraphStore`/`Parser`/keyspace/schema), the captured TS ground truth (`testdata/golden/`), the ROADMAP Phase 2 success criteria, and the technology-stack research in `.claude/CLAUDE.md`. Where Phase 1 explicitly deferred a decision to "Phase 2 extractor design," it is settled below.

### CLI Lifecycle & Commands (INDX-01, INDX-02)
- **D-01:** Build the CLI on **`spf13/cobra`** with root command `codegraph` and subcommands `init`, `index`, `uninit`. **`init`** = create the `.codegraph/` directory **and** run a full from-scratch index in one step (INDX-01's "in one step"). **`index`** = deterministic from-scratch rebuild against an already-initialized `.codegraph/` (INDX-02). **`uninit`** = remove `.codegraph/` cleanly. Rationale: mirrors the TS parity surface and the command hierarchy Cobra is chosen for (`.claude/CLAUDE.md` §CLI).
- **D-01a:** **Idempotency / safety semantics.** `init` on an existing `.codegraph/` **errors with guidance** ("already initialized — use `codegraph index --force` to rebuild") rather than silently clobbering. `uninit` requires confirmation unless `--force` is passed. `index --force` rebuilds without prompting; `--quiet` suppresses progress output; `--verbose` emits per-file / per-pass detail; the default (no flag) prints a concise end-of-run summary (files, nodes, edges, duration). A from-scratch `index` must be **deterministic** — same input tree ⇒ byte-identical graph (this is what makes Phase 3's golden-diff and Phase 8's reproducibility checks possible).
- **D-01b:** **`.codegraph/` layout.** The Pebble store lives in a subdirectory of `.codegraph/` (name is executor's discretion, e.g. `.codegraph/store/`); the store-wide `Meta` record (via `schema.NewMeta()`) carries schema version + counts + health. No separate on-disk config/JSON file in v1. Whether `init` writes a `.gitignore` hint for `.codegraph/` is executor's discretion.

### Node Identity (RES-01 substrate; parity + migration enabler)
- **D-02:** Node ids follow the **TS-parity shape `<kind>:<hash>`** (the captured TS dump shows `class:1aa9ad9ada394f639ed0f8104462aef5`, `constant:01228593…`), where `<hash>` is a **stable content hash** over identity-defining fields (kind + qualified_name + file_path; exact input tuple is executor's discretion but MUST be deterministic and reproducible across runs). Rationale: (a) content-hashed ids are stable across re-index, so the resolve pass matches a reference to a symbol by *recomputing* its id, and Phase 4 incremental sync can detect "same symbol, moved" instead of churning the graph; (b) the `<kind>:<hash>` shape keeps Phase 7 migration a clean id-preserving map onto the TS ground truth; (c) it drops straight into the existing `n/<node-id>` keyspace (`internal/graphstore/keys.go`) with no key-encoder change.
- **D-02a:** The hash MUST be **collision-resistant (SHA-256 family, hex-truncated to id length)** — never MD5 — per Security Domain V6 (01-RESEARCH.md) and the `graph.proto` File.content_hash note. The `<kind>:<hex>` id *shape* is identical to TS; only the hash algorithm is strengthened, so parity/migration mapping is unaffected.

### Schema Field Parity — additive extension (honors D-02a additive-only)
- **D-03:** **Additively extend `internal/schema/graph.proto`** with the parity fields the Go extractor can populate now: on **`Node`** — `signature`, `docstring`, `visibility`, `is_exported` (bool), `return_type`; on **`Edge`** — `provenance` (string) and optional `metadata`. Defer TS `Node` fields that don't apply to Go yet (`is_async`, `is_static`, `is_abstract`, `decorators`, `type_parameters`) — they are added additively when a Phase-5 language needs them. **All new field numbers stay below the reserved `50..59` annotation range**, honoring D-02a; **`SchemaVersion` stays `1`** (a purely additive change never triggers a bump, per `internal/schema/meta.go`). Regenerate `graph.pb.go` from the edited `.proto`.
- **D-03a:** Phase 2 writes **only ground-truth AST edges** — every Phase-2 Go edge has empty/`ast` provenance. The `provenance` field is added now so the record shape is frozen, but the `provenance: heuristic` tag and synthesized dispatch edges are **Phase 5** (RES-02/RES-03). Do not emit any synthesized edge in Phase 2.

### Two-Pass Indexing Architecture (RES-01)
- **D-04:** Implement the roadmap-mandated two-pass pipeline:
  - **Pass 1 — parallel extract:** a bounded worker pool (default `runtime.NumCPU()`, tunable) parses each discovered Go file through the CGo `parser.Parser` backend (`internal/parser/cgo`), walks the tree-sitter tree, and emits a per-file intermediate = (nodes, intra-file edges, **unresolved cross-file references** with name + kind + call-site line/col). Extract is read-only and embarrassingly parallel.
  - **Pass 2 — sequential resolve:** build a global symbol index (qualified-name / import-path → node id), resolve unresolved references into `calls` / `imports` / type-reference edges, and write everything through `GraphStore`. The resolve pass **owns the single coordinated writer**, aligning with `GraphStore`'s "many lock-free readers + one writer" model (INDX-05) — no external locking, no interface bypass.
- **D-04a:** **Write batching** uses `GraphStore.Writer` (IndexedBatch) in batched windows — never one engine write per symbol (Phase 1 D-04). Unresolved references are held **in memory** for v1; the TS on-disk `unresolved_refs` table is an implementation detail we need not mirror in Phase 2 (revisit only if monorepo-scale memory pressure appears — a Phase 8 concern). Batch granularity (per-file vs. per-N) is executor's discretion so long as no per-symbol commits occur.

### Edge Multiplicity — settling the Phase-1-deferred key-identity question
- **D-05:** **Keep the Phase-1 edge-key shape `e/<src>/<kind>/<dst>` unchanged** (`internal/graphstore/keys.go`): multiple call sites sharing the same `(source, kind, target)` **collapse to one stored edge**, carrying the *first/representative* call site's `line`/`col` on the `Edge` record. Rationale: `keys.go` documents this collapse as the deliberate fix for the TS original's historical edge-duplication issue, and Phase 3's callers/callees/impact traversal does not require per-call-site multiplicity. **Documented divergence:** the captured TS DDL's `idx_edges_identity(source,target,kind,line,col)` keeps line/col-distinct edges; we intentionally diverge toward dedup because it matches `keys.go`'s stated design and yields a simpler graph. The `Edge` record already carries `line`/`col`, so if a Phase-3 golden-diff shows this materially changes an agent-facing output, the key shape can change **without data loss** — but the default is collapse.

### Go Extraction Scope (LANG-01)
- **D-06:** The Phase-2 Go extractor produces the parity **node kinds** seen in the golden corpus / TS DDL: `file`, `function`, `method`, `type` (struct), `interface`, `constant`, `variable` (`field` optional, executor's discretion), and **edge kinds**: `contains` (file→symbol, type→method), `imports` (file→imported package), `calls` (function/method → callee), and **type-reference / embedding** edges (struct embedding, interface embedding) — the concrete, AST-visible Go form of success-criterion-3's "type inheritance." These are exactly the relationships Phase-3 queries traverse, so correctness here is what makes those queries answerable.
- **D-06a:** **Interface→implementation dispatch is NOT synthesized in Phase 2.** Go's implicit-interface satisfaction requires the heuristic synthesizer that lands in **Phase 5** (RES-02, with `provenance: heuristic` per RES-03). Phase-2 "type inheritance" = concrete embedding + type references only. This keeps the two-pass mechanism honest and avoids pulling Phase 5's dynamic-dispatch work forward.

### Claude's Discretion
- The exact tree-sitter node-type → codegraph node-kind mapping and the tree-walk/query design (design against `testdata/golden/`).
- The precise hash input tuple and truncation length for node ids (must be deterministic + collision-resistant per D-02a).
- Worker-pool sizing knobs, resolve-pass batch-commit granularity, and the in-memory intermediate's data structures.
- `.codegraph/` internal subdirectory naming and whether `init` writes a `.gitignore` hint.
- Whether `field` nodes and doc-comment (`docstring`) extraction are full or minimal in Phase 2.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher, planner, executor) MUST read these before planning or implementing.**

### Phase 1 Substrate (bind directly against these interfaces — do not re-derive)
- `internal/graphstore/store.go` — the `GraphStore` / `Reader` / `Writer` / `EdgeIterator` interface the indexer writes through (D-04); the single writer + snapshot-reader concurrency contract (INDX-05).
- `internal/graphstore/keys.go` — keyspace encoders; the `e/<src>/<kind>/<dst>` edge-key doc that D-05 settles and the `n/<node-id>` shape D-02 targets.
- `internal/parser/parser.go` — the narrow `Parser` seam (`Parse([]byte, *Tree)`), the `MaxSourceBytes` (4 MiB) size ceiling contract, and the CGo crash-isolation caveat extractors must respect.
- `internal/parser/cgo/parser_cgo.go` — `NewGoParser()` the extractor calls; the incremental-reparse path (relevant for Phase 4, present now).
- `internal/schema/graph.proto` + `internal/schema/graph.pb.go` — the `Node`/`Edge`/`File`/`Meta` record shapes D-03 extends additively; the reserved `50..59` annotation range that additions must stay below.
- `internal/schema/meta.go` — `SchemaVersion` (=1), `NewMeta()`, and the additive-only-no-bump rule D-03 relies on.

### Captured TS Ground Truth (parity vocabulary + id/edge shapes)
- `testdata/golden/ts-schema.sql` — TS `.codegraph/` DDL: `nodes` columns (the field-parity target for D-03), `edges` columns incl. `provenance` + `idx_edges_identity(source,target,kind,line,col)` (the D-05 divergence reference), `unresolved_refs` + `name_segment_vocab` (the two-pass intermediate D-04a references).
- `testdata/golden/ts-schema.dump.sql` — sample node ids (`class:<hash>`, `constant:<hash>`) grounding D-02's `<kind>:<hash>` convention.
- `testdata/golden/corpus/weft-go/` — golden `explore`/`query`/`node`/`callers`/`callees`/`impact`/`status` JSON for a real Go repo; the node kinds (`function`, `method`, `interface`, `constant`, `type`, `file`) and call/contain relationships D-06 must reproduce so Phase 3 queries would return these shapes.
- `testdata/golden/README.md` — corpus provenance + TS version pin.

### Project Planning & Decisions
- `.planning/ROADMAP.md` §"Phase 2: Go Indexing Pipeline" — the four success criteria this CONTEXT must satisfy.
- `.planning/REQUIREMENTS.md` — **INDX-01**, **INDX-02**, **RES-01**, **LANG-01** are the locked contract for this phase; **INDX-05**/**ARCH-01** (Phase 1, complete) are the substrate constraints.
- `.planning/phases/01-foundation-storage-schema-parser-strategy/01-CONTEXT.md` — Phase 1 decisions D-01…D-06 (storage, schema, keyspace, parser) this phase builds on; note the explicit "Phase 2 extractor design" deferrals (edge multiplicity, file-scoped subgraph pruning) that D-05 and Phase 4 pick up.
- `PARSER-DECISION.md` — the ratified Option A (CGo tree-sitter) decision; the incremental-reparse capability and the DIST-05 CGo-exception consequence.
- `.claude/CLAUDE.md` §"The Parser Decision" / §"Storage" / §CLI — the technology-stack rationale behind Cobra, Pebble, and the CGo parser this phase consumes.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`GraphStore` (`internal/graphstore`)** — fully built and concurrency-tested in Phase 1. The indexer's resolve pass instantiates it and drives `NewWriter()` → `Put{Node,Edge,File,Meta}` → `Commit()`. `Snapshot()`/`Export()` are for later phases; the indexer only writes.
- **`Parser` + CGo backend (`internal/parser`, `internal/parser/cgo`)** — `cgo.NewGoParser()` returns a ready `parser.Parser`. The extractor feeds file bytes to `Parse()` (enforcing `MaxSourceBytes`) and walks `Tree.Inner()` cast to the tree-sitter type inside the extractor package. Remember to `Close()` every `*Tree` and the `Parser`.
- **Schema records (`internal/schema`)** — `Node`/`Edge`/`File`/`Meta` protobuf types + `NewMeta()`. Phase 2 extends the `.proto` additively (D-03) then regenerates.
- **Keyspace encoders (`internal/graphstore/keys.go`)** — already implement `n/`, `e/`, `f/`, `m/` layout; the indexer does not touch keys directly (it goes through `GraphStore`), but the id/edge shapes (D-02/D-05) must be compatible with them.

### Established Patterns
- **Interface-boundary discipline (D-04a, enforced by `internal/graphstore/archtest`):** no package outside `internal/graphstore` imports the KV engine. The indexer package MUST depend only on the `GraphStore` interface — the archtest will fail the build otherwise.
- **RED→GREEN atomic commits** (Phase 1 SUMMARY logs) — the project writes a failing test commit before the implementing commit; the planner should structure Phase 2 plans the same way (e.g. golden-fixture-diff tests before extractor code).
- **Size-ceiling-before-parse** (`parser.MaxSourceBytes`) — any code that reads file bytes and parses MUST respect the 4 MiB ceiling the `Parser` contract enforces.

### Integration Points
- **Indexer → `GraphStore`** (the write seam): the resolve pass is the single coordinated writer.
- **Indexer → `Parser`** (the parse seam): extract workers each hold their own `Parser` (tree-sitter parsers are not goroutine-safe — one per worker).
- **CLI → indexer** (`init`/`index` invoke the pipeline; `uninit` only touches the `.codegraph/` directory).
- **This phase freezes the graph the whole product reads:** Phase 3 (queries/MCP), Phase 4 (sync), Phase 7 (migration) all consume the node/edge shapes and id scheme decided here — hence D-02/D-05 are load-bearing beyond Phase 2.

</code_context>

<specifics>
## Specific Ideas

- **Parity is measured against captured ground truth, not memory:** the Go extractor's node-kind and edge-kind vocabulary comes straight from `testdata/golden/corpus/weft-go/` and `testdata/golden/ts-schema.sql`. The planner should treat those fixtures as the spec for "what a correct Go graph contains."
- **Two Phase-1 deferrals are consciously closed here** (node id `<kind>:<hash>`, edge collapse) — they were left open in Phase 1 precisely because they are extractor-design decisions; the planner should surface them early so extractor code binds to settled shapes.
- **Determinism is a first-class requirement** (D-01a): a from-scratch index must be reproducible byte-for-byte, because Phase 3's golden-diff and Phase 8's double-build reproducibility gate both depend on it. Watch for map-iteration-order and parallel-write-order nondeterminism in the resolve pass.

</specifics>

<deferred>
## Deferred Ideas

- **Query commands, `explore`, MCP server / golden-output diffing** — Phase 3 (QRY-*, MCP-*). Phase 2 produces the graph those queries read; it does not build or verify the query surface.
- **Incremental `sync`, native file watchers, rename/delete subgraph pruning, daemon** — Phase 4 (INDX-03/04, SYNC-*). The content-hashed node ids (D-02) and the `f/`-namespace range-delete hook (`keys.go`) are the seams sync will bind to, but no sync logic is built here.
- **Interface→implementation dispatch synthesis + `provenance: heuristic` tagging + framework-aware routing** — Phase 5 (RES-02, RES-03, LANG-07). Phase 2's `provenance`/`metadata` fields exist (D-03) but carry only ground-truth values.
- **Additional languages (Java, C#, Python, TS/JS, mainstream tier)** — Phase 5 (LANG-02+). Phase 2 proves the pipeline on Go only; the extractor should be shaped so a second language is a new extractor behind the same two-pass engine, not a rewrite.
- **On-disk `unresolved_refs` staging, `name_segment_vocab` search index** — TS implementation details not required for Phase 2 correctness; revisit if monorepo-scale memory (Phase 8) or search ergonomics (Phase 3) demand them.
- **100k-file monorepo scale / peak-RSS bounding** — Phase 8 (INDX-06, PERF-02). Phase 2 targets correctness on a real mid-size Go repo.

None of the above are scope creep into Phase 2 — they are correctly-placed future work, recorded so nothing is lost.

</deferred>

---

*Phase: 2-Go Indexing Pipeline*
*Context gathered: 2026-07-10*
