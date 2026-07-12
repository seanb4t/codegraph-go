---
gsd_state_version: 1.0
milestone: v1.3
milestone_name: milestone
current_phase: 05
current_phase_name: Language Coverage & Resolution Breadth
status: executing
stopped_at: Completed 05-01-PLAN.md
last_updated: "2026-07-12T10:37:36.776Z"
last_activity: 2026-07-12
last_activity_desc: Completed 05-01 (multi-language seam foundation)
progress:
  total_phases: 8
  completed_phases: 4
  total_plans: 44
  completed_plans: 34
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-10)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.
**Current focus:** Phase 05 — Language Coverage & Resolution Breadth

## Current Position

Phase: 05 (Language Coverage & Resolution Breadth) — EXECUTING
Plan: 4 of 13
Status: Ready to execute
Last activity: 2026-07-12 — Completed 05-01 (multi-language seam foundation)

Progress: [███████░░░] 73%

## Performance Metrics

**Velocity:**

- Total plans completed: 31
- Average duration: — min
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 7 | - | - |
| 2 | 6 | - | - |
| 3 | 9 | - | - |
| 4 | 9 | - | - |

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
| Phase 03 P06 | 18min | 2 tasks | 6 files |
| Phase 03-query-engine-mcp-server P07 | 15min | 2 tasks | 5 files |
| Phase 03-query-engine-mcp-server P08 | 25min | 3 tasks | 13 files |
| Phase 03-query-engine-mcp-server P09 | 28min | 2 tasks | 1 files |
| Phase 04 P01 | 20min | 2 tasks | 12 files |
| Phase 04-incremental-sync-file-watcher P02 | 8min | 2 tasks | 6 files |
| Phase 04 P03 | 6min | 3 tasks | 8 files |
| Phase 04-incremental-sync-file-watcher P04 | 12min | 2 tasks | 8 files |
| Phase 04 P05 | 6min | 2 tasks | 6 files |
| Phase 04 P07 | 22min | 2 tasks | 8 files |
| Phase 04 P08 | 6min | 2 tasks | 6 files |
| Phase 04-incremental-sync-file-watcher P09 | 5min | 2 tasks | 5 files |
| Phase 05 P01 | 15min | 2 tasks | 8 files |
| Phase 05 P02 | 20min | 2 tasks | 5 files |
| Phase 05 P03 | 35min | 2 tasks | 5 files |

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
- [Phase 03]: Added Engine.repoRoot + NewWithRoot (engine.go) so Node/Explore have a confinement root for on-disk source reads — New() unchanged for existing Reader-only callers
- [Phase 03]: Blast-radius bullet's caller count groups by the matched symbol's OWN defining file (not the callers' files), confirmed against explore.json/node.json arithmetic
- [Phase 03]: Manually promoted github.com/mark3labs/mcp-go to go.mod's direct require block instead of running go mod tidy, per established project convention
- [Phase 03]: ParseAllowlist (pure) split from WarnUnknownToolsTo (io.Writer) so unknown-name stderr warnings are directly unit-testable
- [Phase 03]: search's MCP companion handler marshals []query.Location via a direct encoding/json.Marshal call — no MarshalSearchJSON exists in internal/query unlike its sibling commands, and Location's tags already own the shape
- [Phase 03]: serve requires an explicit --mcp flag (errors if omitted) even though stdio is the only v1 transport — makes the future HTTP/SSE transport selection explicit
- [Phase 03]: search's CLI --json marshals []query.Location directly via encoding/json.Marshal (no dedicated MarshalSearchJSON exists) — matches internal/mcp's companionHandler search branch
- [Phase 03]: files' non-JSON default output renders --format tree as indented plain text via printFileTree; flat format is one line per file — no golden oracle constrains this shape (D-07a)
- [Phase 03]: affected requires at least one positional file argument (cobra.MinimumNArgs(1)) — an empty changed-file set has no useful query-time-derivation output
- [Phase 03-query-engine-mcp-server]: Golden parity test uses set-based subset comparison for callers/callees/impact-affected (never exact equality), generalizing D-05's edge-dedup tolerance to also cover the discovered callees-scope divergence (TS includes non-call references, RefKindCalls-only scoping excludes them)
- [Phase 03-query-engine-mcp-server]: Explore parity subtest normalizes the golden's literal two-word query term ("main function", 0 matches under D-06's no-FTS lexical matcher) to the single-token "mergeStyle"
- [Phase 03-query-engine-mcp-server]: Impact parity subtest asserts a tolerant (<=) NodeCount/EdgeCount relationship, not exact equality — closes RESEARCH Open Question 1's semantics question while documenting a real internal/indexer extraction gap (method call as call-argument not resolved to a calls edge) as a finding, not silently normalized away
- [Phase 4]: x/ namespace stores no value payload — FileIndexEntry fields decode directly from key bytes, no proto.Marshal for index entries
- [Phase 4]: DeleteFileSubgraph does not itself point-delete a file's scattered n/e records — callers must IterateFileIndex(path) before calling it and stage DeleteNode/DeleteEdge for each entry found (binds Plan 04-03's prune-step ordering)
- [Phase 4]: PutEdge signature-change blast radius was wider than RESEARCH's single-call-site claim — also fixed graphstore.Import (with id->FilePath tracking so migrated stores rebuild the x/ index) and five test-double implementations
- [Phase 4]: Confirmed Meta.has_file_index (field 7) was not already added by 04-01 before adding it — no reconciliation needed
- [Phase 4]: query.buildReverseAdjacency exported as BuildReverseAdjacency (mechanical rename) so internal/indexer.Sync() (04-03) can reuse the D-04 reverse-adjacency scan without a circular import
- [Phase ?]: Dedicated testdata/prunefixture subfixture instead of extending shared testdata/gofixture (avoids breaking discover_test.go's exact file-list assertion)
- [Phase ?]: MODIFY subtest re-scoped to within-file symbol rename since nodeid.NodeID hashes (kind,qualifiedName,filePath) not content
- [Phase ?]: MOVE subtest relocates caller+callee pair together across a directory boundary since unqualified-call resolution is import-path scoped
- [Phase 04]: internal/watch depends only on indexer.ShouldSkipDir — no graphstore import, keeping the archtest boundary clean
- [Phase 04]: fsnotify promoted to go.mod direct require by manual edit, not go mod tidy, per Phase 1 convention
- [Phase 04]: internal/daemon never holds a graphstore.GraphStore/Writer directly — indexer.Sync owns its own store lifecycle; single-writer enforced via lockfile + in-process syncMu
- [Phase 04]: Exported internal/watch's debouncer/newDebouncer/debounceDuration as Debouncer/NewDebouncer/DebounceDuration (Rule 3 fix) so internal/daemon can construct and drive one
- [Phase 04]: root.go's sync/daemon/unlock registration split across two commits so each task's own build-verify step stays green independently
- [Phase 04]: serve --watch gates on hasIndex so MCP-03 absent-index behavior is unaffected; a live standalone daemon's ErrLockLive inside --watch is a graceful defer, not a serve failure
- [Phase ?]: SYNC-06 proven via goleak soak (internal/watch TestMain + TestSoak, internal/daemon TestMain + TestSoak) — no real goroutine leaks found on first run, Plan 04-05/04-07 context+WaitGroup discipline held
- [Phase ?]: [Phase 5-01]: TypeScript module exposes two grammar accessors (LanguageTypescript/LanguageTSX) in one bindings/go package, confirmed against the module's own binding_test.go before wiring
- [Phase ?]: [Phase 5-01]: ProjectDescriptor declared as a minimal ModulePath() string interface in languages.go, forward-compatible for Wave 2's discover.go generalization without redesigning the seam now
- [Phase ?]: [Phase 5-01]: Four new grammar go.mod requires manually promoted from indirect to direct — no go mod tidy, per established project convention
- [Phase 05]: Discover's missing-go.mod behavior relaxed to Go's own nil-descriptor path fallback instead of a hard error, per D-03's never-drop-a-supported-extension guarantee
- [Phase 05]: extract.go's multi-language worker-pool fix proven via a go-dup registry entry (real Go parser/extractor under a second ID) rather than waiting on a real second-language extractor
- [Phase 05]: WR-01 tie-break mirrors query.Engine.resolveSymbolNode's lowest-Id-wins convention; single addSymbol() chokepoint used by both overlay() and newSymbolIndexFromStore()
- [Phase 05]: WR-02 fix lives in goextract.go's recordCall (synthetic non-matching PkgAlias for non-identifier operands), not resolve.go
- [Phase 05]: Call-as-argument extraction gap (D-05 third item, 252e2sav94) investigated and found already-fixed by existing walkDescendants; no code change made, only regression tests added

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

Last session: 2026-07-12T10:36:57.243Z
Stopped at: Completed 05-01-PLAN.md
Resume file: None
