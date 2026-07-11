---
phase: 03-query-engine-mcp-server
plan: 01
subsystem: database
tags: [pebble, graphstore, iterator, go]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: pebbleStore/pebbleReader, EdgeIterator + IterateEdges range-scan pattern, keys.go namespace prefixes
  - phase: 02-go-indexing-pipeline
    provides: schema.Node/schema.File records populated by the indexer, so there is real data to enumerate
provides:
  - "graphstore.Reader.IterateNodes() — whole-namespace range scan over n/ returning every Node record"
  - "graphstore.Reader.IterateFiles() — whole-namespace range scan over f/ returning every File record"
  - "NodeIterator/FileIterator interfaces mirroring EdgeIterator's Next/accessor/Err/Close contract"
  - "pebbleNodeIterator/pebbleFileIterator implementations"
affects: [03-02, 03-03, 03-05, query-engine, mcp-server]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Whole-namespace range scan: lower=[]byte{prefixX}, upper=rangeUpperBound(lower) — no appendSegment call needed when there is no segment to bound"
    - "Iterator struct triplet (iter/started/cur/err) copied byte-for-byte per record type, unmarshaling into the type-specific proto message"

key-files:
  created:
    - internal/graphstore/iter_test.go
  modified:
    - internal/graphstore/store.go
    - internal/graphstore/pebble_store.go

key-decisions:
  - "No kind-scoped variant added for IterateNodes — full-scan-with-in-memory-filter is the v1 posture per D-03/RESEARCH Pitfall 1 (n/ key length-prefixes the whole id, so a byte-prefix kind scan isn't cleanly supported)"
  - "No new keys.go helpers added — the whole-namespace prefix is a literal single-byte slice ([]byte{prefixNode}/[]byte{prefixFile}), inlined at the two call sites per the plan's executor-discretion note (a helper would add no clarity for a one-byte literal)"

patterns-established:
  - "Reader enumeration seam: any future whole-namespace scan (e.g. a hypothetical prefixAnnotation iterator) follows the same three-line lower/upper/NewIter shape plus a copied iterator struct"

requirements-completed: [QRY-01, QRY-03, QRY-07, QRY-09]

coverage:
  - id: D1
    description: "Reader.IterateNodes() enumerates every node record in the store via a single contiguous range scan over n/"
    requirement: "QRY-01"
    verification:
      - kind: unit
        ref: "internal/graphstore/iter_test.go#TestIterateNodes"
        status: pass
      - kind: unit
        ref: "internal/graphstore/iter_test.go#TestIterateNodesEmptyStore"
        status: pass
    human_judgment: false
  - id: D2
    description: "Reader.IterateFiles() enumerates every file record in the store via a single contiguous range scan over f/"
    requirement: "QRY-03"
    verification:
      - kind: unit
        ref: "internal/graphstore/iter_test.go#TestIterateFiles"
        status: pass
      - kind: unit
        ref: "internal/graphstore/iter_test.go#TestIterateFilesEmptyStore"
        status: pass
    human_judgment: false
  - id: D3
    description: "graphstore import boundary (archtest) remains intact after the additive Reader extension — no new package reaches pebble/v2 directly"
    requirement: "QRY-07"
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 1: Reader Node/File Enumeration Summary

**Extended `graphstore.Reader` with `IterateNodes()`/`IterateFiles()` — contiguous range scans over the existing `n/`/`f/` keyspaces, byte-for-byte mirroring `IterateEdges`/`pebbleEdgeIterator`, no schema change.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-11T09:24:47-04:00
- **Completed:** 2026-07-11T09:37:10-04:00
- **Tasks:** 2 (RED + GREEN)
- **Files modified:** 3 (2 modified, 1 created)

## Accomplishments
- `Reader` interface gained `IterateNodes() (NodeIterator, error)` and `IterateFiles() (FileIterator, error)`, documented in the same register as `IterateEdges`, citing D-03 and RESEARCH Pitfall 1 for why no kind-scoped variant exists
- `NodeIterator`/`FileIterator` interfaces declared with the same four-method shape as `EdgeIterator` (`Next`/typed accessor/`Err`/`Close`)
- `pebbleNodeIterator`/`pebbleFileIterator` implement the whole-namespace scan: `lower := []byte{prefixNode}` (or `prefixFile`), `upper := rangeUpperBound(lower)`, unmarshaling into `schema.Node`/`schema.File`
- Behavior tests seed a real store through `Writer`/`Commit`, read through `Snapshot()`, and assert exact id/kind (nodes) and path (files) set equality, plus empty-store zero-iteration/nil-Err() contract for both iterators

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Declare IterateNodes/IterateFiles + NodeIterator/FileIterator and write failing enumeration tests** - `12bc01d` (test)
2. **Task 2 (GREEN): Implement pebbleNodeIterator/pebbleFileIterator + keys helpers** - `07f378b` (feat)

_TDD gate sequence confirmed: `test(03-01)` commit precedes `feat(03-01)` commit; no REFACTOR commit needed (implementation was a clean copy of the existing `pebbleEdgeIterator` shape)._

## Files Created/Modified
- `internal/graphstore/store.go` - Added `IterateNodes`/`IterateFiles` to the `Reader` interface; added `NodeIterator`/`FileIterator` interface declarations
- `internal/graphstore/pebble_store.go` - Added `(*pebbleReader).IterateNodes`/`IterateFiles` plus `pebbleNodeIterator`/`pebbleFileIterator` struct+method implementations
- `internal/graphstore/iter_test.go` - New behavior tests: `TestIterateNodes`, `TestIterateFiles`, `TestIterateNodesEmptyStore`, `TestIterateFilesEmptyStore`

## Decisions Made
- Did not add `nodeNamespacePrefix()`/`fileNamespacePrefix()` helpers to `keys.go` — the plan explicitly left this to executor discretion ("inline the byte slice if a helper adds no clarity"); a single-byte literal slice at the two call sites is clearer than a one-line wrapper function
- Did not add a kind-scoped `IterateNodes(kind string)` variant — explicitly excluded by the plan (D-03/Pitfall 1: the `n/` key's length-prefixed id makes a byte-prefix kind scan structurally unsupported; kind filtering belongs in the query engine, in-memory, over the full scan)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- The local 1Password SSH-signing agent (`op-ssh-sign`) intermittently failed (`failed to fill whole buffer` / `agent returned an error`) immediately after a 1Password app auto-update/restart, blocking `git commit` signing for both task commits. Not a code or plan issue — retried after the agent finished re-initializing (confirmed via a standalone `op-ssh-sign -Y sign` smoke test) and both commits succeeded with valid signatures on the next attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/query` (Plans 03-02/03-03/03-05) can now build `Engine.Query`/`Engine.Search`/`Engine.Files`/`Engine.Status` directly against `IterateNodes`/`IterateFiles` — this was the blocking read-path gap for every listing/counting query verb in the phase
- No blockers or concerns carried forward

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*
