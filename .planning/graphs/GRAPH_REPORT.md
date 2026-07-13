# Graph Report - codegraph-go  (2026-07-10)

## Corpus Check
- 76 files · ~184,861 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 674 nodes · 755 edges · 60 communities (47 shown, 13 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 46 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `44b52c8a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Phase 1: Foundation — Storage, Schema & Parser Strategy - Research
- Architecture Research
- Critical Pitfalls
- Implementation Decisions
- v1 Requirements
- Feature Research
- Technology Stack
- goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed
- Phase Details
- CodeGraph Go
- CLAUDE.md
- Project State
- Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log
- Architecture Patterns
- Project Research Summary
- Common Pitfalls
- Code Examples
- Node
- Meta
- parser_test.go
- File
- Edge
- graph.pb.go
- TestMetaRoundTripsSchemaVersion
- Phase 1 — Validation Strategy
- Message
- 01-01-PLAN.md
- 01-02-PLAN.md
- 01-03-PLAN.md
- 01-04-PLAN.md
- 01-05-PLAN.md
- 01-06-PLAN.md
- 01-07-PLAN.md
- github.com/seanb4t/codegraph-go
- Graph Report - codegraph-go  (2026-07-10)
- Architecture Patterns
- Phase 01 Plan 03: Parser Interface + CGo Tree-sitter Backend Summary
- keys.go
- Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06)
- Common Pitfalls
- capture.sh
- Phase 01 Plan 05: Typed Keyspace & Key-Injection Guard Summary
- Common Pitfalls
- golden_test.go
- Code Examples
- Validation Architecture
- store_test.go
- TestNoPackageBypassesGraphStore
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- Security Domain
- Critical Issues
- Phase 01 Plan 06: GraphStore Interface + Pebble/v2 Implementation Summary
- Parser Strategy Decision (D-05 / D-05a)
- Goal Achievement
- Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero)

## God Nodes (most connected - your core abstractions)
1. `Communities (60 total, 13 thin omitted)` - 45 edges
2. `Node` - 21 edges
3. `Meta` - 19 edges
4. `Phase 1: Foundation — Storage, Schema & Parser Strategy - Research` - 19 edges
5. `Edge` - 17 edges
6. `File` - 16 edges
7. `v1 Requirements` - 13 edges
8. `goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed` - 12 edges
9. `Phase 1 Plan 2: Versioned Protobuf Schema Summary` - 12 edges
10. `Graph Report - codegraph-go  (2026-07-10)` - 11 edges

## Surprising Connections (you probably didn't know these)
- `exportNamespace()` --calls--> `rangeUpperBound()`  [INFERRED]
  internal/graphstore/export.go → internal/graphstore/keys.go
- `TestBulkExportReimportsLosslessly()` --calls--> `NewMeta()`  [INFERRED]
  internal/graphstore/export_test.go → internal/schema/meta.go
- `TestBulkExportIsConsistentUnderConcurrentWrite()` --calls--> `Import()`  [INFERRED]
  internal/graphstore/export_test.go → internal/graphstore/export.go
- `TestBulkExportReimportsLosslessly()` --calls--> `Import()`  [INFERRED]
  internal/graphstore/export_test.go → internal/graphstore/export.go
- `TestBulkExportReimportsLosslessly()` --calls--> `Open()`  [INFERRED]
  internal/graphstore/export_test.go → internal/graphstore/pebble_store.go

## Import Cycles
- None detected.

## Communities (60 total, 13 thin omitted)

### Community 0 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Research"
Cohesion: 0.17
Nodes (11): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 1: Foundation — Storage, Schema & Parser Strategy - Research (+3 more)

### Community 1 - "Architecture Research"
Cohesion: 0.07
Nodes (29): Anti-Pattern 1: SQL (or any storage detail) leaking outside `internal/graph`, Anti-Pattern 2: Parser implementation baked directly into the extractor, Anti-Pattern 3: Full reindex as the only incremental strategy ("just re-run init"), Anti-Pattern 4: Multiple uncoordinated writers to the same SQLite file, Anti-Pattern 5: Growing installer if/else chain instead of a per-agent adapter registry, Anti-Patterns, Architectural Patterns, Architecture Research (+21 more)

### Community 2 - "Critical Pitfalls"
Cohesion: 0.09
Nodes (21): Critical Pitfalls, Integration Gotchas, "Looks Done But Isn't" Checklist, Performance Traps, Pitfall 10: Supply-chain "theater" — signing, SBOM, and reproducible-build claims that don't actually verify anything, Pitfall 1: CGo tree-sitter bindings silently break the "single static binary, all platforms" promise, Pitfall 2: SQLite "WAL mode" is mistaken for "concurrent writes," causing silent data loss or `database is locked` storms under multi-agent-session load, Pitfall 3: Naive incremental re-indexing silently corrupts the graph — stale nodes after renames, orphaned edges after deletes, dropped inbound edges (+13 more)

### Community 3 - "Implementation Decisions"
Cohesion: 0.10
Nodes (20): Canonical References, Claude's Discretion, Deferred Ideas, Established Patterns, Existing Code Insights, External Ground Truth, Golden Corpus & TS DDL Capture, GraphStore Interface & Concurrency (+12 more)

### Community 4 - "v1 Requirements"
Cohesion: 0.10
Nodes (20): Agent Integrations, Architecture Future-Proofing, Auto-Sync & Daemon, CLI & Lifecycle, Distribution & Supply Chain, Graph Resolution, Indexing Core, Language Support (+12 more)

### Community 5 - "Feature Research"
Cohesion: 0.12
Nodes (15): Add After Validation (v1.x), Anti-Features (Deliberately Not Building — v1), Competitor Feature Analysis, Dependency Notes, Differentiators (Competitive Advantage for the Go Port), Feature Dependencies, Feature Landscape, Feature Prioritization Matrix (+7 more)

### Community 6 - "Technology Stack"
Cohesion: 0.12
Nodes (15): Alternatives Considered, Core Technologies, Installation, Option A — CGo tree-sitter bindings (`tree-sitter/go-tree-sitter`) — RECOMMENDED for v1, Option B — tree-sitter grammars compiled to WASM, run via `wazero` — WORTH A PHASE-1 SPIKE, not v1-default, Option C — Native pure-Go tree-sitter reimplementation — NOT RECOMMENDED for v1, monitor only, Recommended Stack, Sources (+7 more)

### Community 7 - "goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed"
Cohesion: 0.08
Nodes (25): Alternatives Considered, Architecture, Constraints, Conventions, Core, Core Technologies, Dev / CI tooling (not go.mod deps — install as CLI tools in CI), Developer Profile (+17 more)

### Community 8 - "Phase Details"
Cohesion: 0.14
Nodes (13): Overview, Phase 1: Foundation — Storage, Schema & Parser Strategy, Phase 2: Go Indexing Pipeline, Phase 3: Query Engine & MCP Server, Phase 4: Incremental Sync & File Watcher, Phase 5: Language Coverage & Resolution Breadth, Phase 6: Agent Integrations & CLI Lifecycle, Phase 7: Migration Tool (+5 more)

### Community 9 - "CodeGraph Go"
Cohesion: 0.17
Nodes (11): Active, CodeGraph Go, Constraints, Context, Core Value, Evolution, Key Decisions, Out of Scope (+3 more)

### Community 10 - "CLAUDE.md"
Cohesion: 0.04
Nodes (45): Communities (60 total, 13 thin omitted), Community 0 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Research", Community 10 - "CLAUDE.md", Community 11 - "Project State", Community 12 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log", Community 13 - "Architecture Patterns", Community 14 - "Project Research Summary", Community 15 - "Common Pitfalls" (+37 more)

### Community 11 - "Project State"
Cohesion: 0.18
Nodes (10): Accumulated Context, Blockers/Concerns, Current Position, Decisions, Deferred Items, Pending Todos, Performance Metrics, Project Reference (+2 more)

### Community 12 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log"
Cohesion: 0.20
Nodes (9): Claude's Discretion, Deferred Ideas, Golden-Corpus & TS DDL Capture, GraphStore Interface & Concurrency, Keyspace Layout, Parser Spike Methodology, Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log, Record Encoding & Schema Evolution (+1 more)

### Community 13 - "Architecture Patterns"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 14 - "Project Research Summary"
Cohesion: 0.29
Nodes (6): Confidence Assessment, Executive Summary, Implications for Roadmap, Key Findings, Project Research Summary, Sources

### Community 15 - "Common Pitfalls"
Cohesion: 0.21
Nodes (12): CGoParser, newCGoParser(), NewGoParser(), NewPythonParser(), T, TestCGoCloseIsSafeToCallOnce(), TestCGoParseRejectsOversizeInput(), TestCGoParsesGoSource() (+4 more)

### Community 16 - "Code Examples"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 19 - "parser_test.go"
Cohesion: 0.47
Nodes (6): T, newStubParser(), TestParserAcceptsInputUnderCeiling(), TestParserCloseIsCallable(), TestParserRejectsOversizeInput(), stubParser

### Community 22 - "graph.pb.go"
Cohesion: 0.29
Nodes (3): file_graph_proto_init(), file_graph_proto_rawDescGZIP(), init()

### Community 23 - "TestMetaRoundTripsSchemaVersion"
Cohesion: 0.36
Nodes (6): IsCurrentSchemaVersion(), NewMeta(), T, TestMetaRoundTripsSchemaVersion(), TestNodeRoundTripsEqual(), TestSchemaRoundTripsUnknownFields()

### Community 24 - "Phase 1 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 1 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 37 - "Graph Report - codegraph-go  (2026-07-10)"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 38 - "Architecture Patterns"
Cohesion: 0.22
Nodes (9): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Typed, range-scannable keyspace (D-03), Pattern 2: Snapshot reads + IndexedBatch writes behind `GraphStore` (D-04), Pattern 3: Additive-only protobuf schema with a versioned `meta` record (D-02/D-02a), Pattern 4: Narrow, backend-swappable `Parser` interface (D-05b), Pattern 5: Import-graph bypass test (D-04a), Recommended Project Structure (+1 more)

### Community 39 - "Phase 01 Plan 03: Parser Interface + CGo Tree-sitter Backend Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 40 - "keys.go"
Cohesion: 0.16
Nodes (14): Batch, pebbleWriter, T, TestEdgeKeyRangeScanOrdering(), TestFileSubgraphRangeDeleteBoundary(), TestKeyEncodingRejectsDelimiterInjection(), appendSegment(), edgeKey() (+6 more)

### Community 41 - "Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06)"
Cohesion: 0.29
Nodes (6): Capture provenance, Corpus (D-06a), Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06), Historical bug note (Pitfall 2) — edge-dedup, issue #1034, Re-running the capture, Volatile fields (Pitfall 1) — stripped for byte-for-byte reproducibility

### Community 42 - "Common Pitfalls"
Cohesion: 0.18
Nodes (10): Community Hubs (Navigation), Corpus Check, God Nodes (most connected - your core abstractions), Graph Freshness, Graph Report - codegraph-go  (2026-07-10), Import Cycles, Knowledge Gaps, Suggested Questions (+2 more)

### Community 43 - "capture.sh"
Cohesion: 0.60
Nodes (5): capture_repo(), capture.sh script, strip_json(), strip_sql_timestamps(), wrap_text()

### Community 44 - "Phase 01 Plan 05: Typed Keyspace & Key-Injection Guard Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 45 - "Common Pitfalls"
Cohesion: 0.33
Nodes (6): Common Pitfalls, Pitfall 1: Non-deterministic golden fixtures (verified directly against the live TS index), Pitfall 2: Edge duplication without deterministic identity (a real, historical bug in the TS original), Pitfall 3: Conflating "single static binary for users" with "no CGo toolchain needed anywhere", Pitfall 4: Pebble v1/v2 import-path confusion, Pitfall 5: Treating the D-05 "crash isolation" dimension as validated by valid-input testing alone

### Community 46 - "golden_test.go"
Cohesion: 0.60
Nodes (4): findVolatileKeys(), T, isVolatileKey(), TestGoldenFixturesExist()

### Community 47 - "Code Examples"
Cohesion: 0.40
Nodes (5): Batched writes (plain Batch, not IndexedBatch, for the bulk write path), Code Examples, Incremental tree-sitter reparse, Pebble snapshot for lock-free reads, Range-delete a file's stale subgraph

### Community 48 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 49 - "store_test.go"
Cohesion: 0.07
Nodes (27): DB, EdgeIterator, getter, GraphStore, pebbleEdgeIterator, pebbleReader, Reader, Writer (+19 more)

### Community 50 - "TestNoPackageBypassesGraphStore"
Cohesion: 0.67
Nodes (3): T, isAllowedImporter(), TestNoPackageBypassesGraphStore()

### Community 51 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 52 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 53 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 54 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 55 - "Critical Issues"
Cohesion: 0.12
Nodes (15): CR-01: Every parsed tree-sitter Tree leaks its underlying C memory, CR-02: CGo backend can return a "successful" Tree wrapping a nil native tree, CR-03: Import performs an unbounded allocation driven by an untrusted length prefix, CR-04: Close() races with Snapshot()/NewWriter()/Commit() panic instead of returning errors, Critical Issues, IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match, IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`, IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns (+7 more)

### Community 56 - "Phase 01 Plan 06: GraphStore Interface + Pebble/v2 Implementation Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 57 - "Parser Strategy Decision (D-05 / D-05a)"
Cohesion: 0.15
Nodes (12): Crash Isolation (RESEARCH Pitfall 5), D-05b Note, Decision Rationale (against D-05a's criterion), Disposition of Spike Artifacts, DIST-05 Consequence, Full-parse throughput (entire corpus, one pass per iteration), Incremental reparse (single-keystroke append-at-EOF edit on the corpus's largest file), Measured Numbers (+4 more)

### Community 58 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths, Phase 1: Foundation — Storage, Schema & Parser Strategy Verification Report (+2 more)

### Community 59 - "Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero)"
Cohesion: 0.20
Nodes (9): Decision (human-ratified 2026-07-10), Decisions & deviations, Dependency graph, Measured results, Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero), Requirements, Self-Check: PASSED, Tech tracking (+1 more)

## Knowledge Gaps
- **394 isolated node(s):** `github.com/seanb4t/codegraph-go`, `pebbleStore`, `Constraints`, `Technology Stack`, `Core Technologies` (+389 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **13 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Meta` connect `Meta` to `keys.go`, `store_test.go`, `File`, `Edge`, `graph.pb.go`, `TestMetaRoundTripsSchemaVersion`, `Message`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **Why does `newCGoParser()` connect `Common Pitfalls` to `Message`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `Node` connect `Node` to `keys.go`, `store_test.go`, `Meta`, `File`, `Edge`, `graph.pb.go`, `Message`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **What connects `github.com/seanb4t/codegraph-go`, `pebbleStore`, `Constraints` to the rest of the system?**
  _394 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Architecture Research` be split into smaller, more focused modules?**
  _Cohesion score 0.06666666666666667 - nodes in this community are weakly interconnected._
- **Should `Critical Pitfalls` be split into smaller, more focused modules?**
  _Cohesion score 0.09090909090909091 - nodes in this community are weakly interconnected._
- **Should `Implementation Decisions` be split into smaller, more focused modules?**
  _Cohesion score 0.09523809523809523 - nodes in this community are weakly interconnected._