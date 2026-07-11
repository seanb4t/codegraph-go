---
phase: 02-go-indexing-pipeline
plan: 04
subsystem: indexer
tags: [resolve, symbol-index, cross-file, determinism, graphstore, pass-2]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline (plans 01-03)
    provides: internal/indexer/nodeid.NodeID, additive graph.proto/graph.pb.go, internal/indexer.Discover/DiscoveredFile, internal/indexer.Extract (Pass 1 worker pool), internal/indexer/goextract.FileResult (nodes/intra-edges/unresolved refs/imports map), the committed example.com/gofixture multi-package test fixture
  - phase: 01 (all plans)
    provides: internal/graphstore.GraphStore/Writer (batched single-writer interface), internal/schema.Node/Edge/File/Meta, schema.NewMeta
provides:
  - internal/indexer.symbolIndex (+ resolveSelector/resolveUnqualified) — the global (importPath, declaredName) -> nodeID index Pass 2 builds over every Pass-1 result
  - internal/indexer.resolveRefs(results, modulePath) — resolves calls/imports/embeds/contains UnresolvedRefs into ground-truth calls/imports/embeds/contains edges, plus synthetic package pseudo-nodes for intra-module imports
  - internal/indexer.collapseEdges — deterministic (filePath,line,col)-sorted duplicate-edge collapse (D-05)
  - internal/indexer.writeGraph / internal/indexer.Resolve — single batched GraphStore.Writer commit of the whole resolved graph (D-04a), with a version-stamped Meta record
affects: [pipeline-orchestration, cli-index-command, phase-3-mcp-queries]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "resolveSelector's own alias-membership check against the file's Imports map doubles as RQ-2's narrowest-safe-set boundary — no separate local-variable-type-tracking logic needed, since goextract's UnresolvedRef data model gives Pass 2 no operand-type information to track in the first place"
    - "Edge-collapse representative chosen from a total order (filePath via a nodeID->FilePath map, then line, then col), never processing/map/goroutine order — verified order-independent under repeated shuffled trials, not just a single fixed-order test"
    - "Synthetic package pseudo-node id computed via the SAME nodeid.NodeID hasher as every other node kind (kind=\"package\"), so it participates in the same sorted-by-id staging order with no special-casing at write time"

key-files:
  created:
    - internal/indexer/symbolindex.go
    - internal/indexer/resolve.go
    - internal/indexer/resolve_test.go
  modified: []

key-decisions:
  - "RQ-1 ratified: synthetic package pseudo-node (kind \"package\", id via nodeid.NodeID(\"package\", importPath, importPath), name = last path segment, qualified_name = full import path) for intra-module imports only; external/stdlib imports produce no node and no edge"
  - "RQ-2 ratified: resolve.go performs NO local-variable type tracking at all. A selector call's PkgAlias is checked against the file's own Imports map; if it is not a real import alias (e.g. a local variable receiver in w.Describe()), the call is left unresolved. This happens to fall directly out of goextract's existing data model (UnresolvedRef carries no operand-type information), so RQ-2's 'freshly-constructed literal' extension described in RESEARCH is not implemented — it would require Pass 1 to also track local declarations, which is out of this plan's scope and not required by any test"
  - "Cross-file method-receiver containment (goextract's 4th UnresolvedRef kind, RefKindContains, from 02-03) is resolved into a type -> method 'contains' edge via an unqualified same-package symbol-index lookup — added beyond the plan's illustrative calls/imports/embeds vocabulary (Rule 2: Pass 1 already computes this relationship: a method whose receiver type lives in a different file has NO IntraEdge fallback at all, so leaving it unresolved in Pass 2 would silently orphan the method node from its type in the graph)"
  - "schema.File.NodeCount/EdgeCount are computed PRE-collapse per file (len(r.Nodes), len(r.IntraEdges) + this file's own successfully-resolved reference count) rather than re-deriving them from the final collapsed graph — a defensible, simple per-file bookkeeping metric; the authoritative post-collapse totals live on the Meta record instead"
  - "The symbol index is keyed only by a node's bare Name (not QualifiedName) within an importPath — this means a package-level function and a method sharing the same bare identifier in the same package could theoretically collide; not exercised by any fixture or behavior in this plan, and not a scenario RESEARCH's Pattern 5/Open Question 2 flagged, so left as documented in code rather than pre-emptively engineered around"

requirements-completed: [RES-01, LANG-01]

coverage:
  - id: D1
    description: "A cross-package pkg.Fn() call resolves to the correct callee node id via import-alias -> import-path -> symbol-index lookup"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_CrossPackageCall"
        status: pass
    human_judgment: false
  - id: D2
    description: "An intra-package unqualified call resolves against the calling file's own package symbols"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_IntraPackageCall"
        status: pass
    human_judgment: false
  - id: D3
    description: "Struct and interface embedding produce embeds edges to the embedded type's node when it is in-repo"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_StructEmbeds"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_InterfaceEmbeds"
        status: pass
    human_judgment: false
  - id: D4
    description: "An intra-module import produces an imports edge to a synthetic package pseudo-node; an external/stdlib import produces no edge"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_IntraModuleImport"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_ExternalImportNoEdge"
        status: pass
    human_judgment: false
  - id: D5
    description: "A method call whose receiver type requires interface/inference is left unresolved (D-06a) — never emitted as an edge, but counted"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_UnresolvedMethodCall"
        status: pass
    human_judgment: false
  - id: D6
    description: "Cross-file method-receiver containment resolves into a type -> method contains edge"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_CrossFileMethodContainment"
        status: pass
    human_judgment: false
  - id: D7
    description: "Multiple call sites sharing (source, kind, target) collapse to one edge whose representative line/col is chosen by a deterministic total order, invariant under input shuffling"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestEdgeCollapse_Deterministic"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestEdgeCollapse_OrderIndependent"
        status: pass
    human_judgment: false
  - id: D8
    description: "The whole resolved graph is written through exactly one GraphStore.Writer with a single Commit(); a staging error calls Close() instead; Meta is stamped with correct counts and schema_version == 1"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestSingleWriter_CommitsOnce"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestSingleWriter_CloseOnStagingError"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_EndToEnd"
        status: pass
      - kind: other
        ref: "go test ./internal/graphstore/archtest/... -count=1 (indexer still bypasses no interface)"
        status: pass

duration: 20min
completed: 2026-07-11
status: complete
---

# Phase 2 Plan 04: Pass 2 Resolve — Global Symbol Index and Deterministic Cross-File Resolution Summary

**Pass 2 (`internal/indexer.Resolve`) builds a global (importPath, name) -> nodeID symbol index over Pass 1's results, resolves calls/imports/embeds/contains references into ground-truth edges, deterministically collapses duplicate call sites by a (filePath,line,col) total order, and commits the whole graph through one batched `GraphStore.Writer`**

## Performance

- **Duration:** 20 min
- **Completed:** 2026-07-11T02:12:27Z
- **Tasks:** 2
- **Files created:** 3 (`internal/indexer/symbolindex.go`, `internal/indexer/resolve.go`, `internal/indexer/resolve_test.go`)

## Accomplishments
- `symbolIndex` builds a `map[importPath]map[declaredName]nodeID` over every Pass-1 `FileResult` (skipping skipped/errored files and file nodes), with `resolveSelector` (alias -> import path -> name) and `resolveUnqualified` (own import path -> name) lookups matching RESEARCH's Pattern 5 exactly.
- `resolveRefs` settles every file's `Unresolved` references:
  - Cross-package (`pkga.Alpha()`) and intra-package (`helper()`) calls resolve to the correct callee node id.
  - Struct (`Derived` embeds `Base`) and interface (`ReadWriter` embeds `Reader`) embedding resolve to `embeds` edges.
  - Intra-module imports (`example.com/gofixture/pkga`) mint a synthetic `package` pseudo-node (id via `nodeid.NodeID("package", importPath, importPath)`) and an `imports` edge file -> package node; stdlib/external imports (`"fmt"`) produce neither.
  - A method call through a local variable (`w.Describe()`) is left unresolved — `resolveSelector`'s own import-alias check is what implements RQ-2's narrowest-safe-set boundary, since `PkgAlias` for a local variable is never a key in the file's `Imports` map.
  - Cross-file method-receiver containment (goextract's 4th `UnresolvedRef` kind, `"contains"`, from 02-03) resolves into a type -> method edge — added beyond the plan's illustrative vocabulary so a cross-file method is never left orphaned in the graph.
  - Every emitted edge carries `provenance: "ast"` — no heuristic/dispatch edge is ever synthesized (confirmed via `grep '"heuristic"'` returning nothing in `resolve.go`/`symbolindex.go`).
- `collapseEdges` aggregates every candidate for a `(source, kind, target)` triple and picks the representative by sorting on `(filePath via a source->FilePath map, line, col)` — verified order-independent across 10 shuffled trials, not just a single fixed-order assertion.
- `writeGraph`/`Resolve` stage package pseudo-nodes + symbol nodes (sorted by id), files (sorted by path), and collapsed edges (sorted by source/kind/target) through exactly one `GraphStore.Writer`, committing once; any staging error calls `Close()` instead of `Commit()`. A `Meta` record is stamped with `schema_version` and final node/edge counts.
- Verified via TDD: RED (`test`) commits for both task's test additions precede their GREEN (`feat`) implementation commits.
- `go build ./...`, `go vet ./...`, `go test ./... -race -count=1`, and the `archtest` boundary check all pass.

## Task Commits

Each task was committed atomically:

1. **Task 1: Global symbol index + cross-file reference resolution — RED** - `1766da7` (test)
1. **Task 1: Global symbol index + cross-file reference resolution — GREEN** - `64ec7ed` (feat)
2. **Task 2: Deterministic edge collapse + single-writer batched commit — RED** - `338100f` (test)
2. **Task 2: Deterministic edge collapse + single-writer batched commit — GREEN** - `db7c27f` (feat)

**Plan metadata:** (pending, this commit)

_Note: Both tasks are TDD — RED (`test`) commit precedes GREEN (`feat`) commit._

## Files Created/Modified
- `internal/indexer/symbolindex.go` - `symbolIndex`, `newSymbolIndex`, `resolveSelector`, `resolveUnqualified`
- `internal/indexer/resolve.go` - `Resolve`, `resolveRefs`, `resolveNameRef`, `isIntraModule`, `lastPathSegment`, `collapseEdges`, `writeGraph`, `edgeTriple`, `kindPackage`
- `internal/indexer/resolve_test.go` - `TestResolve_CrossPackageCall`, `TestResolve_IntraPackageCall`, `TestResolve_StructEmbeds`, `TestResolve_InterfaceEmbeds`, `TestResolve_IntraModuleImport`, `TestResolve_ExternalImportNoEdge`, `TestResolve_UnresolvedMethodCall`, `TestResolve_CrossFileMethodContainment`, `TestEdgeCollapse_Deterministic`, `TestEdgeCollapse_OrderIndependent`, `TestSingleWriter_CommitsOnce`, `TestSingleWriter_CloseOnStagingError`, `TestResolve_EndToEnd`, plus `stubWriter`/`stubStore` test doubles and a local `newTestParser` helper

## Decisions Made
- RQ-1 ratified as RESEARCH recommended: a minimal synthetic `package` pseudo-node for intra-module imports only; external/stdlib imports get no node/edge.
- RQ-2 ratified via the data model itself, not new tracking logic: resolve.go never attempts local-variable type inference; the alias-membership check against a file's `Imports` map is both the selector-resolution mechanism AND the "narrowest safe set" boundary, since goextract's `UnresolvedRef` carries no operand-type information to track in the first place.
- Cross-file method-receiver containment (`RefKindContains`) is resolved in this plan even though the plan's interface block names only calls/imports/embeds as illustrative — Rule 2 (auto-add missing critical functionality): leaving it unresolved would silently orphan every cross-file method from its owning type in the graph, since Pass 1 emits no IntraEdge fallback for that case at all.
- `schema.File.NodeCount`/`EdgeCount` are computed pre-collapse, per file; the Meta record carries the authoritative post-collapse totals.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Cross-file method-receiver containment (`RefKindContains`) was not in the plan's illustrative calls/imports/embeds vocabulary but is real Pass-1 output that would otherwise be silently dropped**
- **Found during:** Task 1, while enumerating every `UnresolvedRef.Kind` value goextract (02-03) actually emits
- **Issue:** A method whose receiver type is declared in a different file gets `UnresolvedRef{Kind: RefKindContains}` from Pass 1 with NO fallback `IntraEdge` at all (unlike the same-file case, which always gets a `type->method contains` edge). If Pass 2 only handled calls/imports/embeds, such a method would have zero incoming edge from its own type — an orphaned node in the graph, a real completeness gap for RES-01's "cross-file resolution" headline capability.
- **Fix:** Extended `resolveRefs`'s switch to handle `goextract.RefKindContains`, resolving the receiver type name via an unqualified same-package symbol-index lookup and emitting a `type -> method` `contains` edge.
- **Files modified:** `internal/indexer/resolve.go`
- **Verification:** `TestResolve_CrossFileMethodContainment` (two-file fixture: type in one file, method in another)
- **Committed in:** `64ec7ed` (Task 1 GREEN commit)

## Issues Encountered
None beyond the one auto-fixed completeness gap documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- `internal/indexer.Resolve(store, results, modulePath)` is the complete Pass 2 entry point: given `graphstore.Open`'s store and Pass 1's `[]goextract.FileResult` + `modulePath` (both already available from `Discover`/`Extract`), it commits a fully-resolved graph in one batched write.
- The next plan (pipeline orchestration / CLI `index` command) wires `Discover` -> `Extract` -> `Resolve` together and surfaces `Resolve`'s returned unresolved-reference count via `--verbose`.
- No blockers for subsequent Phase 2 plans.

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-11*

## Self-Check: PASSED
