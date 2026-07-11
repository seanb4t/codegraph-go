---
phase: 02-go-indexing-pipeline
plan: 03
subsystem: indexer
tags: [tree-sitter, go, ast, errgroup, worker-pool, extraction]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline (plans 01-02)
    provides: internal/indexer/nodeid.NodeID content hasher, additively-extended graph.proto/graph.pb.go, internal/indexer.Discover/DiscoveredFile, the committed example.com/gofixture test fixture
provides:
  - internal/indexer/goextract.Extract(p, importPath, relPath, src) (FileResult, error) — Go tree-sitter CST -> codegraph nodes/intra-file contains edges/unresolved cross-file references (LANG-01, D-06)
  - internal/indexer/goextract types: FileResult, ExtractedNode, IntraEdge, UnresolvedRef (Kind: calls/imports/embeds/contains)
  - internal/indexer.Extract(files, limit) ([]goextract.FileResult, error) — Pass 1 bounded, persistent-worker pool (D-04)
affects: [resolve-pass, symbol-index, pipeline-orchestration, cli-index-command]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Iterative (stack-based) CST walk for call-expression scanning — no unbounded Go recursion over a pathologically deep AST (T-02-04)"
    - "Fixed-count persistent worker pool + shared atomic index counter for Pass 1 concurrency, NOT the naive 'create parser inside each errgroup.Go closure' idiom — the latter creates one parser per file under SetLimit whenever files > limit"
    - "Index-addressed result slice (pre-allocated, written by disjoint index) rather than channel-drained completion order, for goroutine-scheduling-independent determinism"
    - "Types collected before functions/methods within a single file's walk, so same-file method->receiver-type contains edges never depend on declaration order"

key-files:
  created:
    - internal/indexer/goextract/goextract.go
    - internal/indexer/goextract/types.go
    - internal/indexer/goextract/doc.go
    - internal/indexer/goextract/goextract_test.go
    - internal/indexer/extract.go
    - internal/indexer/extract_test.go
  modified: []

key-decisions:
  - "Both true type aliases (`type A = B`) and type definitions of a non-struct/interface underlying type (`type Celsius float64`) map to the single `type_alias` node kind, per D-06 — this extractor does not emit a separate node kind for the two Go-language shapes"
  - "A method whose receiver type is declared in a DIFFERENT file than the method gets an UnresolvedRef{Kind:\"contains\"} (a 4th ref kind beyond the plan interface block's illustrative calls/imports/embeds list) rather than being silently dropped or forced into a same-file edge it cannot prove"
  - "Pass 1's worker pool uses a fixed number of persistent goroutines (min(limit, len(files))) pulling file indices from a shared atomic.Int64 counter, each owning exactly one parser for its whole lifetime — NOT the errgroup.Go-per-file-with-SetLimit pattern shown in 02-RESEARCH.md/02-PATTERNS.md, which would construct one parser per file whenever files exceed the limit (see Deviations)"
  - "Cross-package and intra-package calls are both recorded as UnresolvedRef{Kind:\"calls\"} without disambiguating package-qualified selector calls from local-variable method calls (e.g. `w.Describe()`) — Pass 1 over-records candidates; Pass 2's resolve step (a later plan) filters using the Imports map, per RESEARCH Open Question 2's narrowest-safe-set guidance"

patterns-established:
  - "Grammar facts verified via a throwaway debug program parsing real source through the CGo backend (not assumed from documentation alone) before finalizing node-shape-dependent code — caught two 02-RESEARCH.md inaccuracies (interface embedding shape, type_alias node kind) that would otherwise have shipped silently wrong"

requirements-completed: [LANG-01, RES-01]

coverage:
  - id: D1
    description: "Go tree-sitter node types map to the correct codegraph node kind (function/method/struct/interface/type_alias/constant/variable/file); no field nodes emitted"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_NoFieldNodes"
        status: pass
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_SharedFixture"
        status: pass
    human_judgment: false
  - id: D2
    description: "Method value and pointer receivers both resolve to the correct receiver type and qualified_name (Recv.Method), yielding a type->method contains edge (same file) or an unresolved contains ref (cross-file)"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_MethodReceivers"
        status: pass
    human_judgment: false
  - id: D3
    description: "Struct and interface embedding produce unresolved 'embeds' references; a named field of the same underlying type does not"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_StructEmbedding"
        status: pass
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_InterfaceEmbedding"
        status: pass
    human_judgment: false
  - id: D4
    description: "Cross-package pkg.Fn() and intra-package unqualified calls both produce unresolved 'calls' references carrying name, kind, and call-site line/col"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_Calls"
        status: pass
    human_judgment: false
  - id: D5
    description: "ContentHash uses crypto/sha256 (never md5); parser.ErrSourceTooLarge (or any Parse error) is a per-file skip recorded on FileResult.Err with a nil returned error"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_ContentHashIsSHA256"
        status: pass
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
    human_judgment: false
  - id: D6
    description: "Pass 1 extracts every discovered file in parallel with a bounded pool, one Parser per worker (never per file), results index-addressed (order-stable across runs), and a real oversized file contained without aborting the rest of the batch; -race clean"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_OrderStable"
        status: pass
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_BoundedNotPerFile"
        status: pass
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_OversizedFileContained"
        status: pass
      - kind: other
        ref: "go test ./internal/indexer/goextract/... ./internal/indexer/... -run 'TestExtract|TestExtractPool' -race -count=1"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-07-10
status: complete
---

# Phase 2 Plan 03: Go AST Extraction Mapper and Pass 1 Worker Pool Summary

**Go tree-sitter CST walker (`goextract.Extract`) mapping the LANG-01 node/edge vocabulary plus a bounded, persistent-worker-pool Pass 1 (`indexer.Extract`) that runs one parser per worker across the whole file batch, index-addressed for scheduling-independent determinism**

## Performance

- **Duration:** 10 min
- **Started:** 2026-07-10T21:46:31-04:00
- **Completed:** 2026-07-10T21:55:50-04:00
- **Tasks:** 2
- **Files modified:** 6 (all new)

## Accomplishments
- Implemented `internal/indexer/goextract.Extract`: walks a parsed Go file's tree-sitter CST and emits `function`/`method`/`struct`/`interface`/`type_alias`/`constant`/`variable`/`file` nodes, intra-file `contains` edges, and unresolved `calls`/`imports`/`embeds`/`contains` references — no `field` nodes, no synthesized dispatch edges (D-06/D-06a)
- Method receivers (value and pointer) resolve to the same receiver type via a pointer_type-unwrap-first rule; qualified_name is `Recv.Method`; same-file receiver yields an intra-file `type->method` contains edge, cross-file yields an `UnresolvedRef{Kind:"contains"}` for the later resolve pass
- Struct/interface embedding, cross-/intra-package calls, and `import_spec` parsing (aliased/dot/blank imports) all recorded as `UnresolvedRef`s or `Imports` map entries
- `ContentHash` via `crypto/sha256`; a `parser.ErrSourceTooLarge` (or any `Parse` error) is recorded on `FileResult.Err` with a nil returned error — skip, not fatal
- Implemented `internal/indexer.Extract` (Pass 1): a fixed number of persistent worker goroutines (`min(limit, len(files))`), each owning exactly one `parser.Parser` for its whole lifetime, pulling file indices off a shared `atomic.Int64` counter; results are written to a pre-allocated slice by disjoint index, so output order always equals input order regardless of scheduling
- Verified via TDD: RED (`test`) commits for both `goextract_test.go` and `extract_test.go` precede their GREEN (`feat`) implementation commits
- `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` all pass across the whole module

## Task Commits

Each task was committed atomically:

1. **Task 1: Go AST -> codegraph vocabulary mapper — RED** - `ae06ace` (test)
1. **Task 1: Go AST -> codegraph vocabulary mapper — GREEN** - `67decfa` (feat)
2. **Task 2: Pass 1 bounded worker pool — RED** - `0ae12ca` (test)
2. **Task 2: Pass 1 bounded worker pool — GREEN** - `e5b1173` (feat)

**Plan metadata:** (pending, this commit)

_Note: Both tasks are TDD — RED (`test`) commit precedes GREEN (`feat`) commit._

## Files Created/Modified
- `internal/indexer/goextract/doc.go` - Package doc
- `internal/indexer/goextract/types.go` - `FileResult`/`ExtractedNode`/`IntraEdge`/`UnresolvedRef` + node/ref-kind constants
- `internal/indexer/goextract/goextract.go` - `Extract` implementation: type/const/var/import/func/method collection, embedding/call detection, iterative CST walk
- `internal/indexer/goextract/goextract_test.go` - Table-driven node-kind mapping, method receivers, embedding, calls, imports, no-field-nodes, sha256, oversized-skip, shared-fixture tests
- `internal/indexer/extract.go` - `Extract`/`extractWithFactory`/`defaultParserFactory`: the Pass 1 persistent worker pool
- `internal/indexer/extract_test.go` - Order-stability, bounded-not-per-file (counting factory), oversized-file-containment tests

## Decisions Made
- Both a true type alias (`type A = B`) and a type definition of a non-struct/interface underlying type (`type Celsius float64`) map to the single `type_alias` node kind per D-06 — no separate node kind distinguishes the two Go-language shapes
- A method whose receiver type lives in a different file than the method gets a 4th `UnresolvedRef.Kind` value, `"contains"` (the plan's interfaces block names three kinds — calls/imports/embeds — as illustrative, not exhaustive; this extends it for the one relationship Pass 1 genuinely cannot resolve intra-file)
- Pass 1's worker pool uses a fixed count of persistent goroutines pulling file indices from a shared atomic counter, rather than the `errgroup.Group.SetLimit` + "create a parser inside each `g.Go` closure" pattern shown in `02-RESEARCH.md`/`02-PATTERNS.md` — see Deviations below
- Cross-package (`pkg.Fn()`) and local-variable method calls (`w.Describe()`) are both recorded as `UnresolvedRef{Kind:"calls"}` without disambiguation in Pass 1; Pass 2 filters using each file's `Imports` map, per RESEARCH Open Question 2's "narrowest safe set, leave the rest unresolved" guidance

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Two 02-RESEARCH.md grammar-shape claims were wrong for this exact grammar version and had to be corrected before GREEN**
- **Found during:** Task 1, while getting `TestExtract_NodeKinds/true_type_alias_maps_to_type_alias` and `TestExtract_InterfaceEmbedding` to pass
- **Issue:** (a) RESEARCH claimed `type A = B` parses as a `type_spec` distinguished by an `"="` token child — a live parse showed it is actually its own node kind, `type_alias`, with the same `name`/`type` fields as `type_spec`. (b) RESEARCH claimed interface embedding lives under a nested `type_list` sibling of `method_declaration_list` — a live parse showed each embedded entry is instead its own `type_elem` node, a direct sibling of `method_elem` method-signature nodes, with no intermediate `type_list`/`method_declaration_list` wrapper.
- **Fix:** Wrote a small throwaway debug program (`internal/indexer/goextract/zzdebug`, deleted before committing) that parsed representative source through the real CGo backend and printed the CST with field names, confirming the actual node shapes; updated `collectTypes` to handle both `type_spec` and `type_alias` node kinds identically, and rewrote `collectInterfaceEmbeds` to scan for `type_elem` children directly instead of a `type_list` wrapper
- **Files modified:** `internal/indexer/goextract/goextract.go`
- **Verification:** `TestExtract_NodeKinds/true_type_alias_maps_to_type_alias` and `TestExtract_InterfaceEmbedding` (plus `TestExtract_SharedFixture`'s interface-embedding assertion) pass
- **Committed in:** `67decfa` (Task 1 GREEN commit — caught and fixed before that commit, not a separate follow-up)

**2. [Rule 1 - Bug] The RESEARCH/PATTERNS "create a parser inside each errgroup.Go closure" example does not actually bound parser construction to `limit`**
- **Found during:** Task 2, while designing `extractWithFactory` against the acceptance criterion "at most `limit` parsers are constructed (counting factory proves not-per-file)"
- **Issue:** `errgroup.Group.SetLimit(n)` bounds how many goroutines from the group run *concurrently* — it does not turn separate `g.Go(...)` calls into a fixed set of long-lived, file-reusing workers. The pattern shown in `02-RESEARCH.md`'s Pattern 2 and `02-PATTERNS.md`'s `extract.go` section calls `cgo.NewGoParser()` inside each per-file `g.Go` closure, which constructs ONE PARSER PER FILE (up to the file count) whenever there are more files than `limit` — directly violating the plan's own bound
- **Fix:** Implemented `extractWithFactory` with exactly `min(limit, len(files))` persistent worker goroutines, each constructing one parser up front and then pulling file indices from a shared `atomic.Int64` counter until none remain — the shape that actually delivers "one Parser per worker, at most `limit` total, never per file"
- **Files modified:** `internal/indexer/extract.go`
- **Verification:** `TestExtractPool_BoundedNotPerFile` (injected counting parser factory, `limit=2` over a 5-file fixture, asserts `created <= 2`)
- **Committed in:** `e5b1173` (Task 2 GREEN commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - bug fixes to plan-supplied design assumptions that would have shipped incorrect behavior)
**Impact on plan:** Both fixes were necessary for the plan's own stated acceptance criteria to hold (correct node/edge shapes; genuine parser-count bound). No scope creep — no new files, no new package, no behavior beyond what Task 1/Task 2 already specified.

## Issues Encountered
None beyond the two auto-fixed grammar/concurrency-design issues documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/indexer/goextract.FileResult` (nodes, intra-file edges, unresolved refs, imports map, content hash) is the complete Pass-1 intermediate the next plan's Pass 2 (resolve) consumes to build a global symbol index and settle `calls`/`imports`/`embeds`/`contains` references into `GraphStore`-writable edges
- `internal/indexer.Extract`'s ordered `[]goextract.FileResult` is ready to be handed directly to a sequential resolve pass — no further sorting/reordering needed
- No blockers for subsequent Phase 2 plans

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-10*

## Self-Check: PASSED
