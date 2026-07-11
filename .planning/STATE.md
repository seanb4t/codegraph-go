---
gsd_state_version: 1.0
milestone: v1.3
milestone_name: milestone
current_phase: 3
current_phase_name: Query Engine & MCP Server
status: executing
stopped_at: Completed 03-05-PLAN.md
last_updated: "2026-07-11T14:24:24.214Z"
last_activity: 2026-07-11
last_activity_desc: Phase 3 execution started
progress:
  total_phases: 8
  completed_phases: 2
  total_plans: 22
  completed_plans: 18
  percent: 25
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-10)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.
**Current focus:** Phase 3 — Query Engine & MCP Server

## Current Position

Phase: 3 (Query Engine & MCP Server) — EXECUTING
Plan: 6 of 9
Status: Ready to execute
Last activity: 2026-07-11 — Phase 3 execution started

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 13
- Average duration: — min
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 7 | - | - |
| 2 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 01 P01 | 6min | 2 tasks | 6 files |
| Phase 01 P02 | 3min | 2 tasks | 6 files |
| Phase 01 P03 | 20min | 2 tasks | 5 files |
| Phase 01 P04 | 25min | 2 tasks | 19 files |
| Phase 01 P05 | 5min | 2 tasks | 2 files |
| Phase 01 P06 | 25min | 3 tasks | 9 files |
| Phase 02-go-indexing-pipeline P01 | 12min | 2 tasks | 6 files |
| Phase 02-go-indexing-pipeline P02 | 12min | 2 tasks | 9 files |
| Phase 02-go-indexing-pipeline P03 | 10min | 2 tasks | 6 files |
| Phase 02-go-indexing-pipeline P04 | 20min | 2 tasks | 3 files |
| Phase 02 P05 | 25min | 2 tasks | 4 files |
| Phase 02-go-indexing-pipeline P06 | 4min | 2 tasks | 8 files |
| Phase 03-query-engine-mcp-server P01 | 13min | 2 tasks | 3 files |
| Phase 03-query-engine-mcp-server P02 | 6min | 2 tasks | 4 files |
| Phase 03 P03 | 7min | 2 tasks | 2 files |
| Phase 03 P04 | 6min | 3 tasks | 4 files |
| Phase 03-query-engine-mcp-server P05 | 12min | 2 tasks | 3 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Parser strategy (CGo tree-sitter vs wazero WASM) resolved by a benchmarked spike in Phase 1 before architecture locks
- [Roadmap]: Golden-output corpus + TS schema DDL captured in Phase 1 (while live TS version is available) to measure MCP parity against ground truth
- [Roadmap]: Migration tool (Phase 7) waits on Phase 2 schema stability; interface-dispatch synthesis + provenance land with language breadth in Phase 5
- [Phase 01]: Used github.com/cockroachdb/pebble/v2 (not deprecated bare v1 path) per RESEARCH Pitfall 4
- [Phase 01]: Did not run go mod tidy in 01-01 — deps pinned but unimported until Wave 2 code lands
- [Phase 01]: Edge record carries optional line/col fields even though the D-03 Pebble edge key omits them; key-identity multiplicity deferred to Phase 2 extractor design — Preserves call-site data at extraction time per RESEARCH Pitfall 2, without prejudging the key-shape decision
- [Phase 01]: Manually promoted google.golang.org/protobuf and github.com/google/go-cmp from indirect to direct in go.mod instead of running go mod tidy — A full tidy would have stripped 01-01's deliberately pre-pinned, still-unimported deps (pebble/v2, tree-sitter, wazero, x/tools)
- [Phase 01]: Manually promoted go-tree-sitter + grammar modules from indirect to direct requires in go.mod instead of running go mod tidy (would strip 01-01 pre-pinned pebble/v2, wazero, x/tools)
- [Phase 01]: Tree wraps backend value as an unexported any field (NewTree/Inner) rather than generics, keeping the Parser seam simple for the CGo/wazero A/B in 01-07
- [Phase 01]: Used seanb4t/weft (public) + colbymchenry/codegraph (temp clone) as the D-06a golden-fixture corpus; only captured JSON outputs committed
- [Phase 01]: Extended volatile-field strip beyond score/*_at/*At to dbSizeBytes + projectPath/indexPath normalization for byte-for-byte reproducibility
- [Phase 01]: fileSubgraphPrefix scopes only the file's own f/ record in v1; extending to node/edge records deferred to Plan 01-06
- [Phase 01]: rangeUpperBound implemented once as a namespace-agnostic byte-successor helper reused by edge range-scans and file range-deletes
- [Phase 01]: Task 1+2 GREEN commits landed together in 01-06 (GraphStore.Export is part of the interface from the start, D-04) — both RED test commits still preceded implementation
- [Phase 01]: Added golang.org/x/sync and golang.org/x/mod via go get (not go mod tidy) to satisfy go/packages transitive deps for the D-04a archtest; wazero left untouched
- [Phase 02]: NodeID uses SHA-256 (never MD5) truncated to 32 hex chars for TS-parity id shape while retaining collision resistance (D-02a)
- [Phase 02]: graph.proto extended additively only: new field numbers below reserved 50-59, SchemaVersion stayed at 1
- [Phase 02]: Discover returns (files, modulePath, err) rather than embedding modulePath per-file, keeping DiscoveredFile to the three interfaces-block fields
- [Phase 02]: Fixture keeps skip_linux.go and main.go both as package main at fixture root — no declaration collision, lets the discovery test assert GOOS-conditional inclusion without a second fixture dir
- [Phase 02]: Both true type aliases and non-struct/interface type definitions map to the single type_alias node kind (D-06)
- [Phase 02]: Cross-file method receiver containment recorded as a 4th UnresolvedRef.Kind value, contains, beyond the plan's illustrative calls/imports/embeds list
- [Phase 02]: Pass 1 worker pool uses fixed persistent workers pulling file indices from a shared atomic counter, not the errgroup.Go-per-file pattern shown in RESEARCH/PATTERNS, to actually bound parser construction to limit
- [Phase 02]: resolveSelector's alias-membership check against the file's Imports map doubles as RQ-2's narrowest-safe-set boundary — no local-variable-type-tracking logic implemented, since goextract's UnresolvedRef data model gives Pass 2 no operand-type information to track
- [Phase 02]: Cross-file method-receiver containment (RefKindContains) resolved into type->method edges beyond the plan's illustrative calls/imports/embeds vocabulary — Pass 1 emits no fallback edge for this case, so leaving it unresolved would orphan the method node
- [Phase 02]: RQ-1 ratified — synthetic package pseudo-node for intra-module imports only; external/stdlib imports produce no node/edge
- [Phase 02]: INDX-02 determinism gate: two from-scratch fixture rebuilds must produce byte-identical GraphStore.Export() streams after normalizing Meta.last_sync_unix_ms; verified under -race with GOMAXPROCS(8)
- [Phase 02-go-indexing-pipeline]: index without --force prompts interactively via a shared confirm() helper (also used by uninit) rather than hard-requiring --force
- [Phase 02-go-indexing-pipeline]: .codegraph/store/ as the Pebble store subdirectory (D-01b) plus a self-contained .codegraph/.gitignore (*) rather than editing the repo's root .gitignore
- [Phase 03]: No kind-scoped IterateNodes variant added — full-scan-with-in-memory-filter is v1 posture per D-03/RESEARCH Pitfall 1
- [Phase 03]: No new keys.go namespace-prefix helpers — whole-namespace prefix inlined as []byte{prefixNode}/[]byte{prefixFile} literals
- [Phase 03]: internal/query re-declares codegraphDirName/storeSubdir locally rather than importing internal/cli, keeping the CLI-depends-on-query dependency direction
- [Phase 03]: ValidateKind's package kind is a documented string literal, not an import of internal/indexer's unexported kindPackage const
- [Phase 03]: Location (name/kind/filePath/startLine) exported from search.go so 03-04's callers/callees/impact reuse the same shape instead of redefining it
- [Phase 03]: query --json renders Visibility as *string (null when unset) to match the golden fixture's literal visibility:null rather than omitting the key
- [Phase 03]: D-07 auto-approved under --auto: affected derives impacted test files at query time (reverse calls + test-file heuristic) rather than persisting a new test-coverage edge type — persisting would require reindexing the frozen Phase-2 graph and pulling Phase-5 provenance work forward
- [Phase 03]: Fixed Reader.IterateEdges("") to scan the whole e/ namespace (was scanning only empty-src edges) — prerequisite bug for D-04's reverse-adjacency scan, first exercised by 03-04
- [Phase 03]: buildReverseAdjacency filters to goextract.RefKindCalls only — contains/embeds/imports edges excluded from callers/callees/impact/affected
- [Phase 03]: status.go's StatusResult doc comment is the authoritative TS-to-Go/Pebble status.json remapping table (D-05, RESEARCH Open Question 2): backend=pebble, journalMode dropped, version/extraction fields derive from schema.SchemaVersion, pendingChanges/worktreeMismatch present-but-inert
- [Phase 03]: files' Depth=0 means unlimited (diverges from clampDepth's 0-means-default-5 convention used by impact/callers/callees) — negative or above-MaxDepth values are rejected rather than silently clamped

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

None yet.

### Blockers/Concerns

[Issues that affect future work]

- Parser strategy is unresolved until the Phase 1 spike; it gates static-build guarantees (DIST) and the CGo dependency exception (DIST-05)

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-07-11T14:24:24.208Z
Stopped at: Completed 03-05-PLAN.md
Resume file: None
