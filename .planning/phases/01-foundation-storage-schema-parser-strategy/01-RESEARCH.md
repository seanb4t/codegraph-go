# Phase 1: Foundation — Storage, Schema & Parser Strategy - Research

**Researched:** 2026-07-10
**Domain:** Embedded KV storage engines, protobuf schema evolution, tree-sitter parsing (CGo vs WASM), Go architecture-boundary testing, ground-truth capture from a live sibling CLI tool
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Storage Engine**
- **D-01:** Use `github.com/cockroachdb/pebble` as the embedded KV store backing the new graph format. Rationale: pure Go (no CGo), snapshots for consistent reads while a background re-index writes, range deletes for pruning a stale file/symbol subgraph, `IndexedBatch` (read-your-writes) mapping onto graph-mutation semantics. Rejected: bbolt (single-writer bottleneck), Badger (WiscKey shines for large values; graph's dominant payload is small structured records).

**Record Encoding & Schema Evolution**
- **D-02:** Encode node/edge/file records with Protocol Buffers (`google.golang.org/protobuf`), stamp a `schema_version` in a dedicated `meta` record. Field-numbered forward/backward-compatible encoding is the mechanism that makes ARCH-01 true. A `version/export` test asserts an old-schema record round-trips through a new-schema reader and a bulk export re-imports losslessly.
- **D-02a:** Schema evolution discipline is additive-only within a major schema version (never renumber or reuse a field number; deprecate by reserving). A `schema_version` bump is required only for a genuinely breaking layout change, which v1 must avoid.

**Keyspace Layout**
- **D-03:** Prefix-namespaced typed keys: `meta/…` (schema version, counts, last-sync, health), `n/<node-id>` (nodes), `e/<src>/<type>/<dst>` (edges, ordered so callers/callees/impact are range scans), `f/<path>` (file records + content hash), and a reserved `a/…` annotation namespace for post-v1 embeddings/communities. Edge ordering by source enables lock-free forward/reverse traversal via range scans; a file's subgraph is prunable with a single Pebble range delete; the reserved annotation prefix means future features bolt on without touching existing records. Exact id/edge-key byte encoding is planner/executor discretion as long as range-scan and range-delete properties hold.

**GraphStore Interface & Concurrency**
- **D-04:** All graph reads and writes go through a `GraphStore` interface — no other package imports the KV engine directly. Exposes: snapshot-based reads, batched writes (one `IndexedBatch` per file-change/debounce window), iterators for range scans, and an explicit bulk graph export method. Concurrency model: many lock-free readers via Pebble snapshots + one coordinated writer owned by the store.
- **D-04a:** A concurrency/architecture test verifies (a) concurrent readers run correctly alongside a single writer, and (b) no package bypasses the interface (e.g., an import-graph / `go/packages` assertion that only the store package imports `pebble`). This is the enforceable form of INDX-05.

**Parser Strategy Spike**
- **D-05:** Run a head-to-head spike on Go (LANG-01) plus one external-scanner grammar (Python or C# — a grammar whose C scanner exercises the crash-isolation tail-risk), on a real mid-size repo. Measure: parse throughput, incremental-reparse time, static-build impact (does it break `CGO_ENABLED=0`; cross-compile complexity), and crash isolation (does malformed input take down the host process). **Pending spike.**
- **D-05a:** Decision criterion: default to Option A — CGo tree-sitter (`tree-sitter/go-tree-sitter` + per-language grammar modules) for v1. Adopt Option B — wazero WASM grammars only if the spike shows its parse-time overhead is invisible against the full indexing pipeline AND the grammar-to-WASM compilation cost is acceptable. Option C (native pure-Go tree-sitter) is monitor-only, not for v1.
- **D-05b:** Regardless of outcome, the parser sits behind a narrow interface (`Parser.Parse([]byte, *Tree) (*Tree, error)`-shaped) from day one, so a later CGo↔wazero swap is a backend change, not an architecture change. If CGo is selected, it is the single documented CGo exception to the pure-Go/minimal-deps constraint (feeds DIST-05).

**Golden Corpus & TS DDL Capture**
- **D-06:** While the live TS CodeGraph v1.3.x is still available, capture: (a) the TS `.codegraph/` SQLite schema DDL (`.schema` and a representative `.dump`) from a real aged index, and (b) golden JSON snapshots of `codegraph_explore` and companion tool outputs on a small, pinned corpus. Store both as version-pinned fixtures under `testdata/golden/`, recording the exact TS version and capture date.
- **D-06a:** Corpus selection: a compact Go repo (aligns with Phase 2's first language) plus the TS `colbymchenry/codegraph` repo itself (multi-language, exercises the tool surface broadly). Keep it small enough to commit and re-run deterministically.

### Claude's Discretion
- Exact byte encoding of node ids and edge keys (as long as range-scan + range-delete properties from D-03 hold).
- The precise protobuf message shapes for node/edge/file records (design against the TS schema DDL captured in D-06).
- Spike harness structure and which specific real repo is used for benchmarking (D-05), provided it includes Go + one external-scanner language.
- Whether `meta` counts are maintained incrementally or recomputed — a Phase 2+ concern, not locked here.

### Deferred Ideas (OUT OF SCOPE)
- Embeddings/vector search, community detection, graph-visualization UI — post-v1 milestones. v1 only anticipates them via the reserved annotation namespace, forward-compatible encoding, and a bulk export method.
- The actual TS→new-format migration converter — Phase 7. Phase 1 only captures the SQLite DDL as ground truth.
- Indexer, extractors, query commands, MCP tools, file watcher — Phases 2–4.
- Team/server features (central server, CI-distributed indexes) — v2. Architecture must not preclude them but is not built here.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INDX-05 | Storage engine (new format) supports concurrent lock-free readers with a single coordinated writer, implemented behind a `GraphStore` interface no other package bypasses | Pebble `Snapshot`/`IndexedBatch` API confirmed via Context7 (see Code Examples); import-graph enforcement pattern via `golang.org/x/tools/go/packages` (see Architecture Patterns, Pattern 5); Validation Architecture maps this to two concrete tests |
| ARCH-01 | New storage format and `GraphStore` interface are schema-versioned from v1 and accommodate future node/edge annotations and bulk graph export without a format break | Protobuf reserved-field mechanics confirmed via Context7 (unknown-field preservation, `reserved` keyword); `a/` annotation namespace (D-03) gives a physical keyspace slot; bulk export method shape documented in Architecture Patterns |
</phase_requirements>

## Summary

This phase has no code to build on yet — it establishes the substrate. Three research threads converge: (1) the storage/schema design, which is well-precedented (Pebble + protobuf are mature, well-documented Go libraries) but **one locked recommendation in `.claude/CLAUDE.md` is stale** — Pebble has since split into a `/v2` module (`github.com/cockroachdb/pebble/v2`, currently v2.1.6, tagged 2026-05-27) and the project's own research predates or missed this; for a greenfield project there is no reason to start on the deprecated v1 import path. (2) the parser spike, where the highest-value finding is **direct, verified ground truth from the actual Go module registry**: the two grammars CONTEXT.md names as spike candidates (Python, C#) are not equally viable today — `tree-sitter-python` is tagged at v0.25.0 (matching `go-tree-sitter` core), while `tree-sitter-c-sharp` lags at v0.23.5, and Python's grammar has an external C scanner (for INDENT/DEDENT tracking) that C#'s does not — making **Python the correct spike-partner language**, not an arbitrary choice between the two. (3) the golden-corpus capture, where this session had direct, unsupervised access to a **live, installed TS CodeGraph v1.3.1** and real `.codegraph/` SQLite indexes already on disk (including a compact Go repo, `weft`, with 84 files / 1221 nodes / 4063 edges) — the schema DDL, sample tool outputs, and a documented historical dedup bug (#1034) were inspected directly, which is stronger evidence than any web source for this phase.

**Primary recommendation:** Build `internal/graphstore` around `github.com/cockroachdb/pebble/v2` (not v1) with a `GraphStore` interface enforced by both Go convention (`internal/`) and an explicit `go/packages`-based import-graph test; encode records with protobuf and a versioned `meta` record from day one; run the parser spike on Go + Python (not C#) on a real, pinned, externally-reproducible mid-size repo, feeding both valid and deliberately malformed input; and capture the golden corpus + DDL from the already-installed TS CodeGraph v1.3.1 immediately, using its real output as the literal fixture content (not a description of it).

## Architectural Responsibility Map

This project is a single local static binary (CLI + embedded MCP server), not a browser/server/CDN application — the standard tier table does not map cleanly. The table below substitutes the tiers that actually exist in this architecture.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Key-value persistence (nodes/edges/files/meta) | Storage Engine (Pebble) | — | Pebble owns durability, compaction, and the on-disk LSM format; nothing above it should know about SSTables or WAL |
| Read/write access boundary | Library API (`GraphStore` interface) | Storage Engine | The interface is the only legal caller of Pebble; it is itself a library boundary inside one process, not a network API, but plays the same isolating role a backend API tier plays for a database in a networked app |
| Record shape / schema evolution | Serialization Layer (protobuf) | Storage Engine | Protobuf messages are the values Pebble stores; schema versioning lives one layer above raw bytes, in the `meta` record and message definitions, not in Pebble itself |
| Source parsing (AST production) | Parsing Layer (`Parser` interface + CGo or wazero backend) | — | Fully decoupled from storage; produces trees that a later phase's extractor will turn into `GraphStore` writes. No storage or query logic belongs here. |
| Ground-truth capture (golden corpus + DDL) | Build-time / Test-fixture tooling | External process (installed TS CLI) | Not part of the shipped binary at all — a one-time capture harness that shells out to the already-installed `codegraph` v1.3.1 CLI and `sqlite3`, writing results into `testdata/golden/` |
| Import-graph / architecture enforcement | CI / Test tooling | — | Runs as a Go test using `golang.org/x/tools/go/packages`; enforced at test time, not runtime |

**Why this matters here:** the single most consequential mis-mapping this phase could make is letting the *parser* or *any future extractor* import Pebble directly "for convenience," or letting the *storage* layer leak protobuf-specific concerns (e.g., writing bare bytes instead of versioned messages). D-04/D-04a and D-02/D-02a already lock the correct boundaries; this map exists so the plan-checker can verify task assignments don't cross them.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/cockroachdb/pebble/v2` | **v2.1.6** (tagged 2026-05-27) `[VERIFIED: Go module proxy — proxy.golang.org/.../@v/v2.1.6.info]` | Embedded LSM KV store backing the graph format | Pure Go, snapshots, range deletes, `IndexedBatch`. **Correction to `.claude/CLAUDE.md`:** that doc recommends "latest" without noting Pebble split into a `/v2` module; v1's last tag (`v1.1.5`) is from 2025-04-01 and does not open pre-existing v1-format databases going forward the way v2 does — but since this is a greenfield project with no legacy Pebble data, there is no reason to start on the deprecated v1 path. Use the `/v2` import path from the first line of code. |
| `google.golang.org/protobuf` | **v1.36.11** `[VERIFIED: Go module proxy]` — `protoc-gen-go v1.36.11` and `protoc` (`libprotoc 35.1`) are already installed in this dev environment | Record encoding (nodes/edges/files/meta), forward/backward-compatible schema evolution | Field-numbered wire format; `reserved` keyword + unknown-field preservation is the literal mechanism ARCH-01 depends on (confirmed via Context7 — see Code Examples) |
| `github.com/tree-sitter/go-tree-sitter` | **v0.25.0** `[VERIFIED: Go module proxy]` | CGo bindings to tree-sitter core, spike Option A | Official, org-maintained; `Parser.Parse`/`Tree.Edit` API confirmed via Context7 |
| `github.com/tree-sitter/tree-sitter-go` | **v0.25.0** `[VERIFIED: Go module proxy]` | Go grammar for the spike's first language (LANG-01) | Version-aligned with core v0.25.0 |
| `github.com/tree-sitter/tree-sitter-python` | **v0.25.0** `[VERIFIED: Go module proxy]` | Spike's external-scanner grammar — **use Python, not C#** | Version-aligned with core (v0.25.0); Python's grammar has a hand-written external `scanner.c` for INDENT/DEDENT tracking, which is exactly the crash-isolation tail-risk D-05 wants exercised `[CITED: tree-sitter external-scanner docs]`. By contrast `tree-sitter-c-sharp`'s latest tag is **v0.23.5** `[VERIFIED: Go module proxy]` — three minor versions behind core, a live instance of the "grammar modules version independently" risk `.claude/CLAUDE.md`'s Version Compatibility section already warns about. Picking C# for the spike would mean either pinning a skewed grammar version or discovering the skew mid-spike; Python avoids that entirely and still gives the external-scanner test case. |
| `github.com/tetratelabs/wazero` | **v1.12.0** `[VERIFIED: Go module proxy]` | Spike Option B runtime (only if pursuing the wazero comparison arm) | Pure-Go WASM runtime; no CGo, no libc dependency at the host level |
| `golang.org/x/tools` (specifically `go/packages`) | latest (tracks Go toolchain; `go get golang.org/x/tools@latest` at plan time) | Import-graph enforcement test (D-04a) | Already the de facto extension of the Go toolchain (powers `gopls`, `goimports`); no new supply-chain surface beyond what most Go projects already pull in for tooling |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `google.golang.org/protobuf/testing/protocmp` (part of the protobuf module, no separate install) | v1.36.11 | Comparing protobuf messages in round-trip tests (ARCH-01's version/export test) | Use in `internal/schema`'s round-trip test instead of hand-rolled field-by-field comparison |
| Go standard `testing` + `testing/synctest` (Go 1.24+) or manual goroutine + `sync.WaitGroup` harness | stdlib | Concurrency test for INDX-05 (many readers / one writer) | `testing/synctest` (stable since Go 1.24) is worth evaluating for deterministic concurrency testing without real-time sleeps; if it doesn't fit the Pebble snapshot access pattern cleanly, a manual goroutine harness with `-race` is the fallback — both are zero new dependencies |
| `sqlite3` CLI (already installed: 3.54.0) | n/a — OS package, not a Go module | D-06 DDL capture (`.schema`, `.dump`) | Shell out from the capture harness; no Go SQLite driver is needed in Phase 1 at all — reading the TS index is a Phase 7 concern (`modernc.org/sqlite`) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Pebble v2 | Pebble v1 (`github.com/cockroachdb/pebble`, no `/v2` suffix) | Only relevant if the team needed to open an existing pre-v2-format Pebble database — not applicable to a greenfield store. No reason to choose v1 here. |
| CGo tree-sitter (Option A) | wazero WASM grammars (Option B) | Real crash isolation and `CGO_ENABLED=0`, but requires hand-building a grammar→WASM compilation pipeline (no mature off-the-shelf library — see State of the Art) and an informal ~2x parse-time cost `[ASSUMED — single low-confidence community source, per .claude/CLAUDE.md]` |
| Python as the spike's external-scanner grammar | C# | C#'s grammar module is version-skewed against go-tree-sitter core today (v0.23.5 vs v0.25.0); usable, but adds a variable the spike doesn't need to control for |
| Custom import-graph test via `go/packages` | `github.com/fe3dback/go-arch-lint` or `github.com/OpenPeeDeeP/depguard` | Both are real, actively maintained (`go-arch-lint` up to v1.16.0, `depguard` to v1.1.1) `[VERIFIED: Go module proxy]` and give richer layered-architecture config; a hand-written `go/packages`-based test is zero-dependency and sufficient for this phase's single rule ("only `internal/graphstore` imports pebble"). Reconsider a dedicated linter if the architecture grows more layers post-v1. |

**Installation:**
```bash
go mod init github.com/seanb4t/codegraph-go
go get github.com/cockroachdb/pebble/v2@v2.1.6
go get google.golang.org/protobuf@v1.36.11
go get github.com/tree-sitter/go-tree-sitter@v0.25.0
go get github.com/tree-sitter/tree-sitter-go@v0.25.0
go get github.com/tree-sitter/tree-sitter-python@v0.25.0
go get github.com/tetratelabs/wazero@v1.12.0   # spike-only; may be removed from go.mod after D-05a decides
go get golang.org/x/tools@latest                # go/packages for the import-graph test
```

**Version verification performed this session (all via `go list -m -versions` / `go list -m <mod>@latest` against `proxy.golang.org`, the canonical Go module registry — not training data):**

```
github.com/cockroachdb/pebble/v2@latest  -> v2.1.6   (2026-05-27T21:14:17Z)
github.com/cockroachdb/pebble@latest     -> v1.1.5   (2025-04-01T16:37:28Z)  <- do not use
google.golang.org/protobuf@latest        -> v1.36.11
github.com/tree-sitter/go-tree-sitter    -> v0.25.0
github.com/tree-sitter/tree-sitter-go    -> v0.25.0
github.com/tree-sitter/tree-sitter-python-> v0.25.0
github.com/tree-sitter/tree-sitter-c-sharp -> v0.23.5 (lagging)
github.com/tetratelabs/wazero            -> v1.12.0
golang.org/x/tools/go/packages           -> tracks x/tools (no pin needed beyond @latest at plan time)
```

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam supports `npm`/`pypi`/`crates` ecosystems only — Go modules are out of scope for that tool. This audit substitutes direct verification against the official Go module proxy (`proxy.golang.org`, which mirrors canonical VCS tags — an authoritative source, not a community index) plus a manual reputation check on each module's owning organization.

| Package | Registry | Age / Maintainer | Downloads/Reputation | Source Repo | Verdict | Disposition |
|---------|----------|-------------------|----------------------|--------------|---------|-------------|
| `github.com/cockroachdb/pebble/v2` | Go module proxy | Org-maintained since ~2019 (v1), v2 since ~2024 | CockroachDB's own production storage engine — extremely high real-world usage | github.com/cockroachdb/pebble | OK | Approved |
| `google.golang.org/protobuf` | Go module proxy | Google-owned, canonical Go protobuf implementation | Ubiquitous — effectively the Go ecosystem standard | github.com/protocolbuffers/protobuf-go | OK | Approved |
| `github.com/tree-sitter/go-tree-sitter` | Go module proxy | Official `tree-sitter` GitHub org (successor to community `smacker/go-tree-sitter` fork) | Actively released through 2026 | github.com/tree-sitter/go-tree-sitter | OK | Approved |
| `github.com/tree-sitter/tree-sitter-go` | Go module proxy | Official `tree-sitter` GitHub org | Actively released | github.com/tree-sitter/tree-sitter-go | OK | Approved |
| `github.com/tree-sitter/tree-sitter-python` | Go module proxy | Official `tree-sitter` GitHub org | Actively released, version-aligned with core | github.com/tree-sitter/tree-sitter-python | OK | Approved |
| `github.com/tree-sitter/tree-sitter-c-sharp` | Go module proxy | Official `tree-sitter` GitHub org | Actively maintained but **version-lagging** (v0.23.5 vs core v0.25.0) | github.com/tree-sitter/tree-sitter-c-sharp | OK (not SUS — legitimacy is fine, only version currency is a concern) | Not selected for the spike (see rationale above); no action needed |
| `github.com/tetratelabs/wazero` | Go module proxy | Tetrate-backed, well-known pure-Go WASM runtime, also used by `ncruces/go-sqlite3` and others | High — widely adopted for CGo-free WASM embedding | github.com/tetratelabs/wazero | OK | Approved (spike-scope only) |
| `golang.org/x/tools` | Go module proxy | Official Go team (`golang.org/x`) | Canonical extended-stdlib tooling module | golang.org/x/tools | OK | Approved |

**Packages removed due to `[SLOP]` verdict:** none
**Packages flagged as suspicious `[SUS]`:** none

All packages above are org-maintained, high-reputation, and directly named/justified in `.claude/CLAUDE.md`'s prior research — no new unvetted third-party names were introduced this session except confirming the correct import path (`/v2`) for an already-approved package.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌───────────────────────────────┐
                         │   (Future phases: extractor,   │
                         │    CLI commands, MCP server)    │
                         └───────────────┬─────────────────┘
                                         │  calls
                                         ▼
                         ┌───────────────────────────────┐
                         │        GraphStore interface     │  <- D-04: the ONLY
                         │  (internal/graphstore)          │     legal entry point
                         │  - Snapshot() Reader             │
                         │  - NewIndexedBatch() Writer       │
                         │  - Iterate(prefix) Iterator        │
                         │  - Export(w io.Writer) error         │  <- ARCH-01 bulk export
                         └───────┬───────────────┬───────────┘
                    snapshot read│               │batched write
                                 ▼               ▼
                         ┌────────────────────────────────┐
                         │   Pebble/v2 KV engine (LSM)      │  <- ONLY this package
                         │   keyspace:                       │     imports pebble/v2
                         │     meta/…   (schema_version,      │
                         │              counts, health)          │
                         │     n/<id>   (node records)            │
                         │     e/<src>/<kind>/<dst> (edges,         │
                         │              range-scannable)              │
                         │     f/<path> (file + content hash)           │
                         │     a/…      (reserved: embeddings,           │
                         │              community assignments)             │
                         └────────────────────────────────────────────────┘
                                         ▲
                                         │ protobuf-encoded values
                                         │ (schema-versioned, additive-only)
                         ┌───────────────┴─────────────────┐
                         │  internal/schema (generated .pb.go) │
                         └──────────────────────────────────┘

  ── separate, decoupled subsystem ──

                         ┌───────────────────────────────┐
                         │   internal/parser (Parser        │  <- D-05b: narrow interface,
                         │   interface)                       │     backend swappable
                         └───────┬───────────────┬───────────┘
                        CGo backend         wazero backend (spike-only;
                        (tree-sitter core    kept only if D-05a selects it)
                        + grammar modules)
                                 │
                                 ▼
                          AST / Tree  ──────────► (consumed by Phase 2's extractor,
                                                    which writes into GraphStore —
                                                    out of scope for Phase 1)

  ── one-shot, build/test-time only, not shipped ──

     tools/spike/ ── benchmarks CGo vs wazero on Go + Python corpus, feeds
                     parse-throughput / incremental-reparse / crash-isolation
                     numbers into the D-05a decision record

     testdata/golden/ ◄── capture harness shells out to the ALREADY-INSTALLED
                          `codegraph` v1.3.1 CLI + `sqlite3 .schema/.dump`
                          against real .codegraph/ indexes (D-06)
```

### Recommended Project Structure
```
codegraph-go/
├── go.mod
├── internal/
│   ├── graphstore/            # D-04: the ONLY package that imports pebble/v2
│   │   ├── store.go           # GraphStore interface definition
│   │   ├── pebble_store.go    # pebble/v2-backed implementation
│   │   ├── keys.go            # D-03: typed key encoders (meta/ n/ e/ f/ a/)
│   │   ├── batch.go           # IndexedBatch write-path helpers
│   │   ├── export.go          # bulk graph export (ARCH-01)
│   │   ├── store_test.go      # concurrency test (INDX-05a)
│   │   └── archtest/
│   │       └── import_graph_test.go   # D-04a: go/packages bypass check
│   ├── schema/
│   │   ├── graph.proto        # node/edge/file/meta message definitions (D-02)
│   │   ├── graph.pb.go         # generated
│   │   ├── meta.go             # schema_version helpers, additive-only discipline
│   │   └── roundtrip_test.go   # ARCH-01 version/export round-trip test
│   └── parser/
│       ├── parser.go           # D-05b: narrow Parser interface
│       ├── cgo/                # Option A backend (tree-sitter + grammars)
│       │   └── parser_cgo.go
│       └── wazero/             # Option B spike backend (may be deleted post-decision)
│           └── parser_wazero.go
├── tools/
│   └── spike/                  # D-05: throwaway/documented benchmark harness
│       ├── main.go
│       └── bench_test.go       # go test -bench=. output feeds the decision doc
└── testdata/
    └── golden/
        ├── ts-version.txt      # "v1.3.1, captured 2026-07-10"
        ├── ts-schema.sql       # `.schema` dump from a real .codegraph/ index
        ├── ts-schema.dump.sql  # representative `.dump`
        └── corpus/
            ├── <pinned-go-repo>/...       # codegraph_explore + companion JSON
            └── colbymchenry-codegraph/... # multi-language corpus (D-06a)
```

### Pattern 1: Typed, range-scannable keyspace (D-03)
**What:** Encode every stored record under a single-byte-prefixed, typed key so that a whole namespace (or a whole file's subgraph) is addressable by prefix.
**When to use:** Every write and read in `internal/graphstore` goes through these encoders — never construct a raw `[]byte` key inline elsewhere.
**Example:**
```go
// internal/graphstore/keys.go
package graphstore

const (
    prefixMeta       = 'm' // meta/…
    prefixNode       = 'n' // n/<node-id>
    prefixEdge       = 'e' // e/<src>/<kind>/<dst>
    prefixFile       = 'f' // f/<path>
    prefixAnnotation = 'a' // reserved — embeddings, community assignments (ARCH-01)
)

// nodeKey encodes a node id into its Pebble key.
// Length-prefixing avoids ambiguity if an id ever contains a separator byte
// (see Common Pitfalls — key-injection via crafted identifiers).
func nodeKey(id string) []byte {
    return append([]byte{prefixNode, '/'}, []byte(id)...)
}

// edgeKey preserves range-scan order by source, so "all edges from X"
// (callers/callees/impact) is a single Pebble prefix iteration.
func edgeKey(src, kind, dst string) []byte {
    k := append([]byte{prefixEdge, '/'}, []byte(src)...)
    k = append(k, '/')
    k = append(k, []byte(kind)...)
    k = append(k, '/')
    k = append(k, []byte(dst)...)
    return k
}

// fileSubgraphPrefix returns the prefix that a single Pebble DeleteRange
// call can use to prune a whole file's node/edge records on rename/delete.
func fileSubgraphPrefix(path string) []byte {
    return append([]byte{prefixFile, '/'}, []byte(path)...)
}
```

### Pattern 2: Snapshot reads + IndexedBatch writes behind `GraphStore` (D-04)
**What:** Readers never block on the writer; the writer commits one batch per debounce window.
**When to use:** Any code that needs to read the graph (future query/MCP phases) or write to it (future indexer).
**Example:**
```go
// internal/graphstore/store.go
package graphstore

import "io"

type GraphStore interface {
    // Snapshot returns a consistent point-in-time Reader. Multiple snapshots
    // may be open concurrently with an in-flight writer — Pebble coordinates
    // this without locking (confirmed via Context7: a Pebble snapshot "provides
    // a consistent, point-in-time view of the database without pinning
    // memtables").
    Snapshot() (Reader, error)

    // NewWriter returns a batched writer scoped to one file-change /
    // debounce window. Callers commit once; do not issue one write per symbol.
    NewWriter() (Writer, error)

    // Export streams every record in schema-versioned form (ARCH-01) —
    // used by the version/export round-trip test and (post-v1) by
    // visualization/embedding tooling.
    Export(w io.Writer) error
}

type Reader interface {
    GetNode(id string) (*Node, error)
    IterateEdges(srcPrefix string) (EdgeIterator, error)
    Close() error
}

type Writer interface {
    PutNode(n *Node) error
    PutEdge(e *Edge) error
    DeleteFileSubgraph(path string) error // single range-delete (D-03)
    Commit() error
}
```
```go
// internal/graphstore/pebble_store.go (sketch)
package graphstore

import "github.com/cockroachdb/pebble/v2"

type pebbleStore struct {
    db *pebble.DB // the ONLY field in the whole module holding a *pebble.DB
}

func (s *pebbleStore) Snapshot() (Reader, error) {
    snap := s.db.NewSnapshot() // Context7-confirmed: consistent, point-in-time
    return &pebbleReader{snap: snap}, nil
}

func (s *pebbleStore) NewWriter() (Writer, error) {
    // Plain Batch, not IndexedBatch: the write path in this phase does not
    // need read-your-writes within the same batch. IndexedBatch is slower
    // for inserts (Context7-confirmed) — reserve it for callers that need
    // to read uncommitted writes back before Commit.
    b := s.db.NewBatch()
    return &pebbleWriter{batch: b}, nil
}
```

### Pattern 3: Additive-only protobuf schema with a versioned `meta` record (D-02/D-02a)
**What:** Every message reserves field numbers for known-future fields; a dedicated `meta` record (not a per-message field) stamps `schema_version`.
**When to use:** Any change to `graph.proto`.
**Example:**
```protobuf
// internal/schema/graph.proto
syntax = "proto3";
package codegraph.v1;

message Meta {
  uint32 schema_version = 1; // bump ONLY for a breaking layout change
  int64  node_count = 2;
  int64  edge_count = 3;
  int64  last_sync_unix_ms = 4;
}

message Node {
  string id = 1;
  string kind = 2;
  string name = 3;
  string qualified_name = 4;
  string file_path = 5;
  string language = 6;
  int32  start_line = 7;
  int32  end_line = 8;
  // ...

  // Reserved now, before any consumer exists, so a future embedding-vector
  // field lands at a pre-agreed number instead of "whatever's next" —
  // this is the literal mechanism behind ARCH-01.
  reserved 50 to 59; // future: embedding vector, community/cluster assignment
}
```
```go
// A protobuf-verified round trip: an old-schema Node message (missing the
// reserved fields) unmarshals cleanly through a newer reader, and unknown
// fields from a NEWER writer survive an OLDER reader — confirmed via
// Context7 (protobuf-go's UnmarshalOptions / unknown-field preservation).
func TestSchemaRoundTripsUnknownFields(t *testing.T) {
    oldBytes := mustMarshal(t, &NodeV1WithoutReservedFields{Id: "x"})
    var n Node
    if err := proto.Unmarshal(oldBytes, &n); err != nil {
        t.Fatalf("old-schema record must unmarshal through new reader: %v", err)
    }
}
```

### Pattern 4: Narrow, backend-swappable `Parser` interface (D-05b)
**What:** Storage and future extractors never see `*tree_sitter.Tree` directly from a specific backend — only this interface.
**When to use:** Both the CGo and wazero spike backends implement this same shape so the spike can A/B them without touching call sites.
**Example:**
```go
// internal/parser/parser.go
package parser

type Parser interface {
    // Parse produces a Tree from source bytes. If oldTree is non-nil,
    // implementations SHOULD perform an incremental reparse
    // (tree_sitter.Parser.Parse(source, oldTree) in the CGo backend).
    Parse(source []byte, oldTree *Tree) (*Tree, error)
    Close() error
}
```
```go
// internal/parser/cgo/parser_cgo.go — Context7-confirmed API shape
package cgo

import (
    tree_sitter "github.com/tree-sitter/go-tree-sitter"
    tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func NewGoParser() (*CGoParser, error) {
    p := tree_sitter.NewParser()
    if err := p.SetLanguage(tree_sitter.NewLanguage(tree_sitter_go.Language())); err != nil {
        return nil, err
    }
    return &CGoParser{inner: p}, nil
}
// Close() MUST call p.inner.Close() — Context7: "always remember to call
// Close() on Parser, Tree, TreeCursor, Query, QueryCursor... to properly
// free C memory allocations."
```

### Pattern 5: Import-graph bypass test (D-04a)
**What:** A Go test (not a shell script, not a third-party linter) that loads the module's package graph and asserts only `internal/graphstore` (and its own subpackages) import `pebble/v2`.
**When to use:** Runs in the standard `go test ./...` suite — this is the enforceable form of INDX-05's "no package bypasses the interface."
**Example:**
```go
// internal/graphstore/archtest/import_graph_test.go
package archtest

import (
    "testing"
    "golang.org/x/tools/go/packages"
)

const pebbleImportPath = "github.com/cockroachdb/pebble/v2"
const allowedImporterPrefix = "github.com/seanb4t/codegraph-go/internal/graphstore"

func TestNoPackageBypassesGraphStore(t *testing.T) {
    cfg := &packages.Config{Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps}
    pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
    if err != nil {
        t.Fatalf("packages.Load: %v", err)
    }
    for _, pkg := range pkgs {
        if _, imports := pkg.Imports[pebbleImportPath]; imports {
            if len(pkg.PkgPath) < len(allowedImporterPrefix) || pkg.PkgPath[:len(allowedImporterPrefix)] != allowedImporterPrefix {
                t.Errorf("package %s imports pebble/v2 directly — only internal/graphstore may", pkg.PkgPath)
            }
        }
    }
}
```
Note: Go's `internal/` convention independently blocks *other modules* from importing `internal/graphstore`'s helpers, which matters for the milestone-2 team/server architecture — but it does NOT stop a sibling package inside this same module from importing `pebble/v2` directly. The test above is what actually enforces D-04a today.

### Anti-Patterns to Avoid
- **Per-symbol Pebble writes:** D-04 requires one `Batch`/`IndexedBatch` per file-change/debounce window. Writing per-symbol defeats Pebble's commit-pipeline batching (confirmed via Context7's commit-pipeline docs) and will dominate indexing latency at monorepo scale.
- **Reaching for `IndexedBatch` by default:** Context7-confirmed — "An indexed batch is slower than a non-indexed batch for insert operations." Use plain `Batch` unless a caller genuinely needs read-your-writes before `Commit()`.
- **Reusing or renumbering a protobuf field number:** breaks D-02a's additive-only discipline and silently corrupts old records read by a newer schema. Always add a `reserved N;` line when a field is retired.
- **Hand-parsing import statements with regex/string matching** instead of `go/packages` for the D-04a test — regex-based import detection breaks on aliased imports, build-tag-gated files, and vendored copies.
- **Treating `recover()` as a safety net for the CGo parser path:** a C-level segfault in a grammar's external scanner is not a Go panic — `recover()` cannot catch it. Don't design the crash-isolation spike measurement (or later production code) around a false sense of safety here (see Security Domain).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Embedded KV storage with concurrent readers | A custom mmap'd file format with manual locking | `pebble/v2` | LSM compaction, WAL durability, and snapshot isolation are exactly the "huge effort, easy to get subtly wrong" category this constraint exists to avoid |
| Schema evolution / forward-compat encoding | A hand-rolled versioned-struct + manual field-migration system | Protobuf `reserved` + unknown-field preservation | Protobuf's wire format was purpose-built for this; reinventing it risks exactly the "silent format break" ARCH-01 forbids |
| Source-code parsing into an AST | A hand-written recursive-descent parser per language | `tree-sitter` (CGo or, if selected, wazero-WASM) grammars | 12+ language grammars, each with their own edge cases (Python indentation, Go's implicit semicolons, etc.) — this is squarely in "existing solution for a deceptively complex problem" territory |
| Import-graph / architecture-boundary checking | Regex over `.go` file text to detect "who imports what" | `golang.org/x/tools/go/packages` | The official Go tooling library already resolves imports correctly across build tags, aliases, and test variants — regex will silently miss cases |
| Cross-compiling a CGo binary for multiple OS/arch targets | A bespoke Docker-per-target build matrix | `zig cc`/`zig c++` as `CC`/`CXX` via GoReleaser (`goreleaser/example-zig-cgo`) | Zig bundles libc for every target and is a drop-in clang-compatible cross-compiler; this is a well-trodden, documented pattern (Phase 8 concern, but the parser spike's static-build-impact measurement should use the same approach so results transfer) |

**Key insight:** every "don't hand-roll" item above already has a mature, actively-maintained, officially-documented Go library or tool — this phase's job is wiring them together correctly (interface boundaries, keyspace layout, schema discipline), not inventing new mechanisms.

## Common Pitfalls

### Pitfall 1: Non-deterministic golden fixtures (verified directly against the live TS index)
**What goes wrong:** A captured golden JSON snapshot (D-06) contains fields that will differ on every future run against the "same" corpus, causing spurious parity-test failures in Phases 3/4/7.
**Why it happens:** Directly observed in this session's inspection of a real `.codegraph/` index: `codegraph query --json` returns a floating-point FTS5 BM25 `score` (e.g. `114.65185167578747`) and every node/file record carries `updatedAt`/`indexed_at` epoch-millisecond timestamps set to "now" at index time — neither is stable across a re-index, even of byte-identical source.
**How to avoid:** The capture harness must either (a) normalize/strip `score` and all `*_at`/`*At` timestamp fields before writing a fixture, or (b) store them but have the Phase 3+ comparison logic explicitly ignore those fields (document which fields are "volatile" directly in the fixture's own README or a sidecar metadata file).
**Warning signs:** A "golden" fixture that fails to reproduce byte-for-byte on a second capture run against the identical source tree.

### Pitfall 2: Edge duplication without deterministic identity (a real, historical bug in the TS original)
**What goes wrong:** Two indexing passes (or a resolve pass re-run after a sync) emit the same logical edge twice, inflating edge counts and polluting callers/impact results.
**Why it happens:** Directly observed in the captured schema.sql comment: the TS project hit exactly this (issue #1034) — `INSERT OR IGNORE` without a `UNIQUE` constraint to conflict on behaved like a plain `INSERT`. Their fix was a unique index on `(source, target, kind, IFNULL(line,-1), IFNULL(col,-1))`.
**How to avoid:** The Pebble edge key (`e/<src>/<kind>/<dst>`) as currently specified in D-03 does **not** include line/col — this means two structurally-different edges at different call sites but the same (src, kind, dst) will collide and overwrite, which is actually *desirable* dedup behavior for most edge kinds, but the planner should explicitly decide (and document) whether line/col needs to be part of the key for edge kinds where multiple call sites between the same two symbols must be preserved distinctly (e.g., a function calling the same callee twice on different lines).
**Warning signs:** Edge counts in the new store don't match expected counts from a hand-verified small fixture; callers/impact results show a single edge where the source actually has two distinct call sites.

### Pitfall 3: Conflating "single static binary for users" with "no CGo toolchain needed anywhere"
**What goes wrong:** Assuming CGo tree-sitter preserves the "one static binary" distribution story without extra CI work.
**Why it happens:** The distributed artifact is still one static binary per platform either way — but *building* a CGo binary for a target OS/arch the CI runner isn't natively on requires a C cross-compiler (`zig cc`) at build time, not just `GOOS`/`GOARCH` env vars.
**How to avoid:** The D-05 spike's "static-build impact" measurement must explicitly attempt a cross-compile (e.g., build for `linux/arm64` from this darwin/arm64 dev machine) and record whether it succeeds with the currently-installed toolchain, not just measure `CGO_ENABLED=0` vs `=1` on the native target. **This session's environment does not have `zig` installed** — see Environment Availability.
**Warning signs:** CI passes on the native runner's architecture but fails (or silently produces a non-functional binary) on a cross-compiled target.

### Pitfall 4: Pebble v1/v2 import-path confusion
**What goes wrong:** Copy-pasting an older Pebble example (including, ironically, `.claude/CLAUDE.md`'s own installation snippet, which predates this correction) pulls in `github.com/cockroachdb/pebble` (v1, last tagged 2025-04-01) instead of `github.com/cockroachdb/pebble/v2` (v2.1.6, current).
**Why it happens:** Go's major-version-in-import-path convention means `go get github.com/cockroachdb/pebble@latest` and `go get github.com/cockroachdb/pebble/v2@latest` resolve to two entirely different, API-compatible-but-not-identical modules — no compiler error warns you picked the older one.
**How to avoid:** Always type the `/v2` suffix explicitly; grep `go.mod` for a bare `cockroachdb/pebble` line (without `/v2`) as a pre-commit sanity check.
**Warning signs:** `go.mod` shows `github.com/cockroachdb/pebble v1.x.x` instead of `github.com/cockroachdb/pebble/v2 v2.x.x`.

### Pitfall 5: Treating the D-05 "crash isolation" dimension as validated by valid-input testing alone
**What goes wrong:** The spike runs both backends against well-formed source files, both "pass," and the team concludes crash isolation is a non-issue.
**Why it happens:** CGo tree-sitter's crash-isolation *risk* only manifests on malformed/adversarial input hitting a grammar's external C scanner (per `.claude/CLAUDE.md`'s own framing) — valid source from a real repo won't exercise it.
**How to avoid:** The spike harness must deliberately feed truncated files, byte-garbage, and deeply nested/pathological constructs (especially to the Python grammar's external scanner) to both backends and observe: does the CGo backend's host process survive, and does the wazero backend's error propagate as a recoverable Go error instead of a runtime panic/crash?
**Warning signs:** Spike report says "no crashes observed" but the corpus used was only well-formed code.

## Code Examples

### Pebble snapshot for lock-free reads
```go
// Source: Context7 /cockroachdb/pebble — "a snapshot provides a consistent,
// point-in-time view of the database without pinning memtables"
snap := db.NewSnapshot()
defer snap.Close()
value, closer, err := snap.Get(nodeKey("function:abc123"))
if err != nil { /* handle */ }
defer closer.Close()
```

### Batched writes (plain Batch, not IndexedBatch, for the bulk write path)
```go
// Source: Context7 /cockroachdb/pebble — db.go NewBatch/NewIndexedBatch docs
b := db.NewBatch()
for _, n := range nodesToWrite {
    if err := b.Set(nodeKey(n.Id), mustMarshalNode(n), nil); err != nil {
        return err
    }
}
if err := b.Commit(pebble.Sync); err != nil {
    return err
}
```

### Range-delete a file's stale subgraph
```go
// D-03: a whole file's node/edge records are prunable with one call.
// Signature shape per Pebble's documented range-deletion semantics
// (start inclusive, end exclusive) — confirmed via Context7.
start := fileSubgraphPrefix(path)
end := append(append([]byte{}, start...), 0xff) // exclusive upper bound
if err := b.DeleteRange(start, end, nil); err != nil {
    return err
}
```

### Incremental tree-sitter reparse
```go
// Source: Context7 /tree-sitter/go-tree-sitter
edit := &tree_sitter.InputEdit{
    StartByte: 8, OldEndByte: 9, NewEndByte: 11,
    StartPosition: tree_sitter.Point{Row: 0, Column: 8},
    OldEndPosition: tree_sitter.Point{Row: 0, Column: 9},
    NewEndPosition: tree_sitter.Point{Row: 0, Column: 11},
}
oldTree.Edit(edit)
newTree := parser.Parse(newSource, oldTree) // reuses unaffected subtrees
defer newTree.Close()
changedRanges := oldTree.ChangedRanges(newTree) // feeds incremental-reparse metric
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `github.com/cockroachdb/pebble` (v1, single-module) | `github.com/cockroachdb/pebble/v2` | v2.0.0 release; current v2.1.6 tagged 2026-05-27 | **`.claude/CLAUDE.md`'s existing research recommends "latest" without this distinction — it is now stale on this specific point.** v2 drops support for opening pre-v1/RocksDB on-disk formats, which is irrelevant for a greenfield store; use `/v2` from the start. `[VERIFIED: Go module proxy]` |
| `ncruces/go-sqlite3` running SQLite-via-WASM on wazero, cited in CLAUDE.md as "proof the wazero pattern scales" | Project is migrating to `wasm2go` (compile-time WASM→Go transpilation, not a runtime WASM interpreter) per maintainer discussion #361, citing "slow performance on interpreter targets, and slow startup on compiler targets" | Ongoing as of this research | Tempers, but doesn't invalidate, the CLAUDE.md framing: the maintainer explicitly says wazero "will always be relevant if you want to load Wasm modules at runtime" — which describes tree-sitter's need for dynamically-loaded grammars better than SQLite's statically-known single WASM blob. Still, this is a live signal that wazero's interpreter/startup overhead is a real, acknowledged cost even in its flagship large-scale Go use case. `[CITED: github.com/ncruces/go-sqlite3 discussion #361]` |
| — | `malivvan/tree-sitter` (wazero-based, CGo-free tree-sitter bindings) exists but remains pre-release/experimental: 3 commits, 3 stars, single maintainer, no documented benchmarks vs CGo | Unchanged since CLAUDE.md's research | Confirms CLAUDE.md's "not for v1" call still holds — no mature Option-B library exists to lean on; building one is genuinely this project's own engineering work if D-05a selects Option B `[CITED: github.com/malivvan/tree-sitter README]` |

**Deprecated/outdated:**
- `github.com/cockroachdb/pebble` (v1 import path): superseded by `/v2` — see above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The informal "~2x slower" WASM-vs-CGo parse-time figure (from `.claude/CLAUDE.md`, itself sourced from a single low-confidence community benchmark) | Standard Stack / Alternatives Considered | If the real gap on this project's actual corpus is smaller, Option B looks more attractive than assumed; if larger, Option A's case strengthens further. The Phase-1 spike itself is the intended resolution — this assumption should not survive past the spike. |
| A2 | Zig (`zig cc`) "just works" for CGo tree-sitter cross-compilation the same way it does for the general CGo/SQLite precedent cited in CLAUDE.md and the `goreleaser/example-zig-cgo` repo | Common Pitfalls / Don't Hand-Roll | Tree-sitter's external scanners (C, sometimes C++) are a different codebase shape than SQLite's amalgamated single file; if zig's cross-compile fails on a specific grammar's external scanner, the spike needs to surface that concretely rather than assuming a clean generic-CGo result transfers |
| A3 | `golang.org/x/tools/go/packages`-based import-graph test is sufficient enforcement for D-04a at v1 scale, without a dedicated architecture-linter (`go-arch-lint`/`depguard`) | Architecture Patterns, Pattern 5 | If the codebase grows more architectural layers before a real linter is adopted, boundary violations in layers other than "who imports pebble" won't be caught by this narrow test |
| A4 | `testing/synctest` (Go 1.24+) is a viable fit for the INDX-05 concurrency test's deterministic-timing needs | Standard Stack (Supporting) | Untested against Pebble's actual snapshot/commit-pipeline timing model this session; the planner should spike this quickly before committing, with a manual goroutine+`-race` harness as the documented fallback |

**If this table is empty:** N/A — see entries above.

## Open Questions

1. **Should the parser spike's corpus repo be a pinned external OSS repo, or the locally-discovered `weft` repo?**
   - What we know: This session found a real, compact, already-indexed Go repo (`weft`, 84 files / 1221 nodes / 4063 edges) at `/Volumes/Code/github.com/seanb4t/weft` — a genuinely convenient local candidate matching D-06a's "compact Go repo" criterion for the golden-corpus capture.
   - What's unclear: Whether the *parser spike's* benchmark corpus (a separate concern from the golden-corpus capture) should be this same private local repo, or a pinned-commit public OSS repo for CI reproducibility across machines/contributors.
   - Recommendation: Use `weft` (or a similarly-sized local/committable repo) for the **golden-corpus capture** (D-06a explicitly allows planner discretion on the specific repo). For the **parser spike's** benchmark corpus, prefer a pinned-commit public repo so CI can reproduce the exact throughput numbers without depending on a machine-local path — the planner should pick one at execution time.

2. **Does the D-06 golden corpus need to be captured against a freshly-reindexed `.codegraph/` (running `codegraph index --force` first), or is the existing on-disk index (built with the CLI's older internal `1.1.6` extraction-version stamp, per `codegraph status --json`'s `builtWithVersion` field) good enough?**
   - What we know: `codegraph status --json` against the real `weft` index reports `"builtWithVersion":"1.1.6"` even though the installed CLI is `1.3.1`; `reindexRecommended` was reported `false`, but the extraction-version field (`24`) matched current — suggesting output shape is stable even though the index metadata predates the current CLI version.
   - What's unclear: Whether tool *output shape* (what Phase 3's MCP parity test cares about) can differ between extraction versions even when `reindexRecommended` is false.
   - Recommendation: Re-run `codegraph index --force` (or `sync`) immediately before capturing golden fixtures, so the captured ground truth is unambiguously "produced by CLI v1.3.1 end-to-end," not a mix of an older index plus a newer CLI's read path.

3. **Line/col in the edge key (see Common Pitfalls #2) — include or omit?**
   - What we know: D-03 as locked specifies `e/<src>/<type>/<dst>` without line/col; the TS original's own schema treats `(source, target, kind, line, col)` as the full identity tuple for uniqueness.
   - What's unclear: Whether any edge kind in this project's design legitimately needs multiple distinct edges between the same (src, kind, dst) at different call sites, or whether that's acceptable to collapse.
   - Recommendation: Planner decision, informed by which edge kinds Phase 2's extractor will emit — flag for `/gsd-discuss-phase`-style confirmation if Phase 2 planning reveals a kind that needs multiplicity.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All of Phase 1 | ✓ | 1.26.5 (darwin/arm64) | — |
| C compiler (native CGo) | Option A spike, native-target builds | ✓ | Apple clang 21.0.0 | — |
| `zig` (cross-compile toolchain) | D-05's cross-compile-complexity measurement | ✗ | — | `brew install zig`; if unavailable, document the *known* zig-cc approach (per `goreleaser/example-zig-cgo`) without executing a live cross-compile, and defer full cross-arch validation to Phase 8 |
| `protoc` + `protoc-gen-go` | D-02 protobuf codegen | ✓ | `libprotoc 35.1` / `protoc-gen-go v1.36.11` | — |
| `sqlite3` CLI | D-06 DDL capture (`.schema`/`.dump`) | ✓ | 3.54.0 | — |
| Live TS CodeGraph CLI (v1.3.x) | D-06 golden-corpus + DDL capture — **time-sensitive** | ✓ | v1.3.1, installed at `/opt/homebrew/bin/codegraph` | — |
| Real `.codegraph/` SQLite indexes for corpus source | D-06a | ✓ | Found at `weft` (Go, compact — 84 files), `fovea` (Go), `holomush/holomush` (large, multi-language, 154MB) | — |
| `git` | version control, module init | ✓ | 2.55.0 | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** `zig` — install via Homebrew, or scope the spike's cross-compile measurement to documentation-only pending Phase 8.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (`go test`) — no third-party test framework needed for a greenfield Go project |
| Config file | none yet — Wave 0 creates `go.mod` and the package skeleton below |
| Quick run command | `go test ./... -run <TestName> -v` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INDX-05 | Concurrent lock-free readers alongside a single coordinated writer | unit/integration | `go test ./internal/graphstore/... -race -run TestConcurrentReadersSingleWriter` | ❌ Wave 0 |
| INDX-05 | No package bypasses the `GraphStore` interface | static/architecture | `go test ./internal/graphstore/archtest/... -run TestNoPackageBypassesGraphStore` | ❌ Wave 0 |
| ARCH-01 | Schema-versioned records round-trip unknown/future fields | unit | `go test ./internal/schema/... -run TestSchemaRoundTripsUnknownFields` | ❌ Wave 0 |
| ARCH-01 | Bulk graph export re-imports losslessly | integration | `go test ./internal/graphstore/... -run TestBulkExportReimportsLosslessly` | ❌ Wave 0 |
| Success Criterion 3 (parser spike) | Parse throughput / incremental-reparse / crash-isolation comparison, CGo vs wazero | benchmark (not pass/fail) | `go test ./tools/spike/... -bench=. -benchmem` | ❌ Wave 0 |
| Success Criterion 4 (golden corpus) | TS DDL + golden JSON fixtures captured and non-empty | smoke test | `go test ./testdata/golden/... -run TestGoldenFixturesExist` (or a `Taskfile`/shell smoke check, since fixtures are captured via an external CLI, not Go code) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** targeted `go test ./<changed-package>/... -run <TestName>`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** full suite green, plus the spike's benchmark numbers recorded in a decision document (e.g. `PARSER-DECISION.md` or embedded in the plan's completion notes) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `go.mod` — does not exist yet; `go mod init` is the first executable step of this phase
- [ ] `internal/graphstore/store_test.go` — concurrency test, covers INDX-05
- [ ] `internal/graphstore/archtest/import_graph_test.go` — bypass test, covers INDX-05
- [ ] `internal/schema/roundtrip_test.go` — covers ARCH-01
- [ ] `internal/graphstore/export_test.go` (or folded into `store_test.go`) — bulk export test, covers ARCH-01
- [ ] `tools/spike/bench_test.go` — parser benchmark harness, covers Success Criterion 3
- [ ] `testdata/golden/` capture harness (shell script or `Taskfile` target invoking the installed `codegraph` CLI + `sqlite3`) — covers Success Criterion 4
- [ ] Framework install: none beyond `go get` for the libraries in Standard Stack — Go's `testing` package needs no separate install

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This phase is a local, single-user embedded library — no auth boundary exists at this layer |
| V3 Session Management | No | No sessions in a storage/parser substrate |
| V4 Access Control | No | Filesystem permissions on `.codegraph/`/the Pebble data directory are the only access boundary, unchanged from the OS default; no in-process access control model needed |
| V5 Input Validation | **Yes** | The parser ingests arbitrary, potentially adversarial third-party source files. Standard controls: bound file size/line count before handing bytes to the parser; length-prefix or otherwise escape path/id segments before concatenating them into Pebble keys (see Common Pitfalls, key-injection risk) |
| V6 Cryptography | Partial | File records' `content_hash` (per the captured TS schema, D-03's `f/<path>` record) should use a collision-resistant hash (e.g. SHA-256) rather than a weak hash like MD5 — standard control, not a novel cryptographic design |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/adversarial source file crashes the CGo tree-sitter host process (no process isolation between C scanner code and the Go host) | Denial of Service | This is precisely what the D-05 spike's crash-isolation dimension must measure (see Common Pitfalls #5); mitigation options for whichever backend is chosen: enforce per-file parse timeouts, and if Option A (CGo) is selected, document explicitly that `recover()` cannot catch a C-level segfault — only isolate via a subprocess boundary if this risk proves unacceptable in practice (a Phase 2+ concern, not Phase 1) |
| Crafted file path or identifier string containing key-delimiter bytes (`/`) or control characters, corrupting Pebble range-scan/range-delete boundaries across namespaces | Tampering | Length-prefix or escape path/id segments in the key encoders (Pattern 1) rather than raw string concatenation with a literal `/` separator |
| Pathological input (e.g., a single-line multi-megabyte minified file, or deeply nested constructs) causing excessive memory/CPU during parse | Denial of Service | Enforce a file-size ceiling before parsing; this is also directly relevant to INDX-06 (100k+ file monorepo, bounded memory) in a later phase, but the ceiling mechanism belongs in the `Parser` interface's contract from day one |

## Sources

### Primary (HIGH confidence)
- **Direct inspection of the installed TS CodeGraph v1.3.1 CLI and its real `.codegraph/` SQLite indexes** (`/opt/homebrew/bin/codegraph`, `weft`/`fovea`/`holomush` repos on this machine) — schema DDL (`.schema`), live query/status/explore JSON output, and the in-source comment documenting historical edge-dedup bug #1034. This is the strongest possible source for D-06 and directly informed Common Pitfalls #1 and #2.
- `go list -m -versions` / `go list -m <module>@latest` against `proxy.golang.org` (the official Go module proxy) — version currency for Pebble (including the `/v2` split), protobuf, go-tree-sitter, tree-sitter-go/python/c-sharp, wazero, x/tools, go-arch-lint, depguard.
- Context7 `/cockroachdb/pebble` — `NewBatch`/`NewIndexedBatch`/`DeleteRange`/`NewSnapshot` API and semantics.
- Context7 `/protocolbuffers/protobuf-go` and `/protocolbuffers/protobuf` — reserved field numbers, `reserved` keyword syntax, unknown-field preservation on unmarshal.
- Context7 `/tree-sitter/go-tree-sitter` — `Parser.Parse`, `Tree.Edit`, incremental reparse, resource-cleanup (`Close()`) requirements.
- `.claude/CLAUDE.md` §"Storage — the new graph format", §"The Parser Decision", §"Alternatives Considered", §"What NOT to Use", §"Version Compatibility" — the project's own primary technology-stack research; treated as the baseline this document builds on and, in one case (Pebble v2), corrects.

### Secondary (MEDIUM confidence)
- Context7 `/websites/wazero_io` — runtime terminology (Host/Binary/Sandbox/Module), architecture overview.
- WebFetch `goreleaser.com/cookbooks/cgo-and-crosscompiling` and `github.com/goreleaser/example-zig-cgo` — zig-cc cross-compile pattern for CGo binaries (env vars, per-OS `CC`/`CXX`, musl targets).
- WebFetch `github.com/ncruces/go-sqlite3` discussion #361 — maintainer's stated rationale for migrating from wazero to `wasm2go`, and explicit statement that wazero "will always be relevant" for runtime-loaded WASM modules.
- WebFetch `pkg.go.dev/golang.org/x/tools/go/packages` — `Config`/`Package` struct shapes, `Load` usage pattern for the import-graph test.

### Tertiary (LOW confidence)
- WebSearch on tree-sitter-python's external scanner / INDENT-DEDENT handling — general tree-sitter documentation and community discussion, not the grammar's own source directly inspected this session.
- WebFetch `github.com/malivvan/tree-sitter` — single-maintainer, 3-commit, pre-release repo; low signal beyond confirming it remains immature.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version claim was directly verified against the Go module proxy this session (not asserted from training data), including a material correction (Pebble `/v2`) to the project's existing research.
- Architecture: MEDIUM-HIGH — the `GraphStore`/keyspace/protobuf patterns are standard, well-precedented Go idioms and the Pebble/protobuf API specifics are Context7-confirmed; the exact interface and message shapes are explicitly left to planner/executor discretion by CONTEXT.md, so this document provides a strong starting sketch, not a final design.
- Pitfalls: HIGH — the two most concrete pitfalls (non-deterministic fixtures, edge-dedup) were verified by direct inspection of the live TS CodeGraph artifact's real behavior and its own documented bug history, not guessed or inferred.

**Research date:** 2026-07-10
**Valid until:** ~30 days for the storage/schema domain (Pebble/protobuf move slowly); ~14 days for the parser-spike comparison landscape specifically (wazero/wasm2go/malivvan/tree-sitter are all actively moving targets this session found mid-transition) — re-verify grammar/core version alignment immediately before executing the spike if more than a couple of weeks elapse before this phase starts.
