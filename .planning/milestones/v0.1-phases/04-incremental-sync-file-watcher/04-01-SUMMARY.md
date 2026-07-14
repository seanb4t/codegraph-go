---
phase: 04-incremental-sync-file-watcher
plan: 01
subsystem: database
tags: [pebble, graphstore, secondary-index, pruning, tdd]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "Pebble-backed GraphStore/Reader/Writer interfaces, keys.go's appendSegment/rangeUpperBound collision-safety primitives, fileSubgraphPrefix/DeleteFileSubgraph named as the Phase-4 hook"
  - phase: 02-go-indexing-pipeline
    provides: "content-hashed node ids (not file-clustered), resolve.go's writeGraph/nodeFilePath map, additive schema evolution discipline"
provides:
  - "x/ file-owned secondary index namespace (prefixFileIndex 'x') mapping a file path to its owned node ids and outgoing edge triples"
  - "Reader.IterateFileIndex(path) — one contiguous scan enumerating a file's owned n/e records"
  - "Writer.DeleteNode/DeleteEdge point-deletes; DeleteFileSubgraph extended to also range-delete the file's x/ region"
  - "Writer.PutEdge(e, ownerPath) signature — every caller now supplies the edge's owning file"
affects: [04-02, 04-03, sync-engine, prune-step]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive Pebble namespace with fixed-byte markers (fileIndexKindNode/Edge) distinguishing sub-ranges within one file's prefix"
    - "Key-only storage (nil value payload) for reference-only index entries — the key itself encodes the (path, node/edge) relationship"
    - "decodeSegment as the symmetric decode counterpart to appendSegment, enabling iterator decode-from-key-bytes without a stored value"

key-files:
  created:
    - internal/graphstore/fileindex_test.go
  modified:
    - internal/graphstore/keys.go
    - internal/graphstore/store.go
    - internal/graphstore/batch.go
    - internal/graphstore/pebble_store.go
    - internal/graphstore/keyenc_test.go
    - internal/graphstore/export.go
    - internal/graphstore/export_test.go
    - internal/graphstore/iter_test.go
    - internal/indexer/resolve.go
    - internal/indexer/resolve_test.go
    - internal/query/search_test.go
    - internal/query/traverse_test.go

key-decisions:
  - "x/ namespace stores no value payload — every FileIndexEntry field decodes directly from the key bytes (path segment skipped, marker byte, then id or src/kind/dst segments)"
  - "DeleteFileSubgraph issues two DeleteRange calls (f/ then x/) but remains one logical Writer call from the caller's perspective; it does NOT itself point-delete a file's scattered n/e records — callers must IterateFileIndex(path) BEFORE calling it and stage DeleteNode/DeleteEdge for each entry found"
  - "PutEdge's blast radius was wider than the plan's RESEARCH claimed ('exactly one existing call site'): also updated graphstore.Import (export.go), plus three test files' Writer/Reader stand-ins (resolve_test.go's stubWriter, search_test.go/traverse_test.go's fake readers) — all necessary for go build ./... and go vet ./... to pass"
  - "Import tracks id->FilePath across streamed node records (Export's own ordering guarantees nodes precede edges) so a migrated/re-imported store's x/ index is correctly rebuilt, not silently left empty"

patterns-established:
  - "Reference-only secondary index: key-encodes-everything, nil value, decoded via a segment offset walk mirroring the encode-side appendSegment order"

requirements-completed: [INDX-04]

coverage:
  - id: D1
    description: "x/ namespace key builders (fileIndexNodeKey/fileIndexEdgeKey/fileIndexPrefix) never collide or range-overlap across adversarial paths/ids, mirroring T-01-02's collision-safety discipline"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/graphstore/keyenc_test.go#TestKeyEncodingRejectsDelimiterInjection/distinct_file-index_node/edge_entries_never_collide"
        status: pass
      - kind: unit
        ref: "internal/graphstore/fileindex_test.go#TestFileIndexEncodingRejectsCollision"
        status: pass
    human_judgment: false
  - id: D2
    description: "Reader.IterateFileIndex(path) enumerates exactly a file's owned node ids and outgoing edge triples in one contiguous scan, isolated from other files"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/graphstore/fileindex_test.go#TestFileIndexRoundTrip"
        status: pass
    human_judgment: false
  - id: D3
    description: "DeleteFileSubgraph(path) prunes both the file's f/ record and every x/<path>/... entry atomically, leaving a lexicographically adjacent sibling file's own entries untouched"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/graphstore/fileindex_test.go#TestFileIndexDeleteFileSubgraphPrunesXNamespace"
        status: pass
    human_judgment: false
  - id: D4
    description: "Writer.DeleteNode/DeleteEdge point-delete exactly the targeted n/e record without touching an untouched sibling"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/graphstore/fileindex_test.go#TestFileIndexPointDeletes"
        status: pass
    human_judgment: false
  - id: D5
    description: "Whole-module build/vet/race-test suite and the archtest Pebble-import boundary stay green after the PutEdge signature change and its full call-site blast radius"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && go test ./... -race -count=1"
        status: pass
      - kind: unit
        ref: "internal/graphstore/archtest#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 1: File-Owned Secondary Index Summary

**Additive `x/` Pebble namespace mapping file path to owned node ids and outgoing-edge triples, making `DeleteFileSubgraph`-class pruning O(subgraph) instead of a full-graph scan — the load-bearing INDX-04 storage hook Phase 1 explicitly planted.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-11T18:44:56Z
- **Completed:** 2026-07-11T18:51:40Z
- **Tasks:** 2 (RED, GREEN)
- **Files modified:** 12 (1 created, 11 modified)

## Accomplishments
- New `x/` key namespace (`prefixFileIndex = 'x'`) with `fileIndexNodeKey`/`fileIndexEdgeKey`/`fileIndexPrefix` builders following the exact `appendSegment` collision-safety discipline as every other namespace
- `Reader.IterateFileIndex(path)` — a single contiguous scan yielding a file's owned node ids and outgoing edge triples, decoded straight from key bytes (no value payload stored)
- `Writer.DeleteNode`/`DeleteEdge` point-delete primitives, and `DeleteFileSubgraph` extended to range-delete the file's `x/` region alongside its existing `f/` record delete
- `Writer.PutEdge(e, ownerPath)` — every edge now carries its owning file, threaded from `resolve.go`'s already-computed `nodeFilePath` map
- `SchemaVersion` unchanged at 1 — verified additive-only via `grep -c 'SchemaVersion uint32 = 1' internal/schema/meta.go` == 1

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing tests for the x/ file-index namespace** - `0e47d93` (test)
2. **Task 2 (GREEN): implement the x/ namespace, iterator, and Writer methods** - `52c8cc2` (feat)

_TDD plan: RED commit precedes GREEN commit; no separate REFACTOR commit was needed._

## Files Created/Modified
- `internal/graphstore/fileindex_test.go` - New: TestFileIndexRoundTrip, TestFileIndexDeleteFileSubgraphPrunesXNamespace, TestFileIndexPointDeletes, TestFileIndexEncodingRejectsCollision
- `internal/graphstore/keys.go` - `prefixFileIndex`/marker consts, `fileIndex*` builders, `decodeSegment` decode counterpart
- `internal/graphstore/store.go` - `FileIndexEntry`/`FileIndexIterator` types, `Reader.IterateFileIndex`, `Writer.DeleteNode`/`DeleteEdge`, `PutEdge` signature change
- `internal/graphstore/batch.go` - `pebbleWriter` stages `x/` entries in `PutNode`/`PutEdge`; `DeleteFileSubgraph`'s second `DeleteRange`; `DeleteNode`/`DeleteEdge` impls
- `internal/graphstore/pebble_store.go` - `IterateFileIndex` + `pebbleFileIndexIterator`/`decodeFileIndexKey`
- `internal/graphstore/keyenc_test.go` - Extended `TestKeyEncodingRejectsDelimiterInjection` with the `x/` collision subtest
- `internal/graphstore/export.go` - `Import` tracks `id -> FilePath` across streamed nodes to supply `PutEdge`'s `ownerPath`
- `internal/graphstore/export_test.go`, `internal/graphstore/iter_test.go` - Updated `PutEdge` call sites for the new signature
- `internal/indexer/resolve.go` - `writeGraph` threads `nodeFilePath[e.Source]` as `ownerPath`
- `internal/indexer/resolve_test.go` - `stubWriter` updated: `PutEdge(e, ownerPath)`, `DeleteNode`, `DeleteEdge` stubs
- `internal/query/search_test.go`, `internal/query/traverse_test.go` - Fake `Reader` stand-ins gained `IterateFileIndex` stubs

## Decisions Made
- The `x/` namespace stores no value payload — every `FileIndexEntry` field is decoded directly from the key bytes (path segment skipped since the caller already knows it from the scan bound, then a fixed marker byte, then either one `nodeID` segment or three `src/kind/dst` segments)
- `DeleteFileSubgraph` remains one logical "prune this file entirely" Writer call from the caller's perspective, but internally issues two `DeleteRange`s (`f/` then `x/`); it does NOT itself point-delete a file's scattered `n/`/`e/` records — callers must `IterateFileIndex(path)` *before* calling it and stage `DeleteNode`/`DeleteEdge` for each entry found (this ordering constraint will bind Plan 04-03's prune step)
- `Import` (bulk export/import round-trip) tracks `id -> FilePath` across streamed node records — since Export's own namespace ordering guarantees every node precedes its edges — so a migrated/re-imported store's `x/` index rebuilds correctly rather than silently ending up empty

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] PutEdge's actual call-site blast radius was wider than RESEARCH's "exactly one existing call site" claim**
- **Found during:** Task 2 (GREEN implementation), running `go build ./...`/`go vet ./...` after the signature change
- **Issue:** The plan's interfaces block and RESEARCH Pattern 2 stated `PutEdge`'s signature change "blast radius: exactly one existing call site, resolve.go's writeGraph." A repo-wide `rg -n "\.PutEdge\("` found five call sites: `resolve.go` (the one named), plus `graphstore/export.go`'s `Import`, and three `_test.go` files (`iter_test.go`, `export_test.go`) calling `PutEdge` directly, plus two more test files (`resolve_test.go`'s `stubWriter`, `search_test.go`/`traverse_test.go`'s fake `Reader`s) implementing the `Writer`/`Reader` interfaces structurally and therefore needing the new methods too.
- **Fix:** Updated all five real call sites to the new `PutEdge(e, ownerPath)` signature, and updated the three interface-implementing test doubles to add `DeleteNode`/`DeleteEdge`/`IterateFileIndex` stub methods. For `Import`, rather than passing an empty `ownerPath` (which would silently leave every migrated store's `x/` index for edges empty), added `id -> FilePath` tracking across the streamed node records so the rebuilt `x/` index stays correct for re-imported/migrated stores too (Rule 2: missing-critical-functionality, since an incomplete `x/` index after migration would silently break Plan 04-03's pruning on any pre-Phase-4 store that goes through `Export`/`Import`).
- **Files modified:** `internal/graphstore/export.go`, `internal/graphstore/export_test.go`, `internal/graphstore/iter_test.go`, `internal/indexer/resolve_test.go`, `internal/query/search_test.go`, `internal/query/traverse_test.go`
- **Verification:** `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` all pass; `internal/graphstore/archtest` boundary test still green
- **Committed in:** `52c8cc2` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — corrected an inaccurate blast-radius claim in the plan's own RESEARCH context, plus one Rule 2 completeness fix layered onto it)
**Impact on plan:** Necessary for the code to compile and for the `x/` index to stay correct across the bulk export/import migration path; no scope creep beyond what `go build`/`go vet` required plus the one migration-correctness addition.

## Issues Encountered
None beyond the PutEdge blast-radius deviation documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The `x/` file-owned secondary index, `IterateFileIndex`, `DeleteNode`/`DeleteEdge`, and the extended `DeleteFileSubgraph` are all in place and tested — Plan 04-02 (store-seeded symbol index) and Plan 04-03 (the `Sync()` prune step) can now build directly on this storage substrate without further `internal/graphstore` changes.
- `DeleteFileSubgraph`'s ordering constraint (callers must `IterateFileIndex` before calling it) is documented in the `Writer` interface doc comment and this summary — Plan 04-03's prune-step implementation must follow that order.
- No backfill/detection mechanism for pre-Phase-4 graphs (D-02b) was built in this plan — that remains for whichever plan implements `Sync()`'s Meta-flag/namespace-probe backfill path.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED

All key files and both task commits (0e47d93 RED, 52c8cc2 GREEN) verified present.
