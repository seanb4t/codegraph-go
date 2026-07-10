---
phase: 01-foundation-storage-schema-parser-strategy
plan: 06
subsystem: database
tags: [pebble, graphstore, concurrency, protobuf, export, go-packages, archtest, go]

# Dependency graph
requires:
  - phase: 01-foundation-storage-schema-parser-strategy (plan 02)
    provides: internal/schema Node/Edge/File/Meta protobuf records (D-02)
  - phase: 01-foundation-storage-schema-parser-strategy (plan 05)
    provides: internal/graphstore/keys.go typed key encoders (nodeKey, edgeKey, edgeSrcPrefix, fileKey, fileSubgraphPrefix, metaKey, rangeUpperBound)
provides:
  - internal/graphstore/store.go — GraphStore/Reader/Writer/EdgeIterator interfaces (D-04): snapshot reads, batched writes, edge range-scan iteration, bulk export
  - internal/graphstore/pebble_store.go — pebbleStore, the sole holder of *pebble.DB in the module; Open/Snapshot/NewWriter/Close
  - internal/graphstore/batch.go — pebbleWriter wrapping a plain (non-indexed) Pebble Batch; PutNode/PutEdge/PutFile/PutMeta/DeleteFileSubgraph/Commit
  - internal/graphstore/export.go — Export (snapshot-consistent, length-framed protobuf stream) and Import (ARCH-01 bulk export/import)
  - internal/graphstore/archtest/import_graph_test.go — TestNoPackageBypassesGraphStore, the go/packages-based enforcement of D-04a
affects: [02-indexer (writes through GraphStore.NewWriter), 03-query-mcp (reads through GraphStore.Snapshot), 04-sync (snapshots + DeleteFileSubgraph for rename/delete pruning), 07-migration (writes reconstructed graph through GraphStore)]

# Tech tracking
tech-stack:
  added:
    - "github.com/cockroachdb/pebble/v2 — promoted from indirect to direct (first real import)"
    - "golang.org/x/tools/go/packages — promoted from indirect to direct; used only by archtest"
    - "golang.org/x/sync, golang.org/x/mod — added as transitive deps of go/packages"
  patterns:
    - "Single pebbleStore struct holds the only *pebble.DB in the module; every other type (pebbleReader, pebbleWriter, pebbleEdgeIterator) wraps a Pebble handle obtained FROM pebbleStore, never opens one itself"
    - "getProto(getter, key, msg) helper unifies Get+unmarshal+ErrNotFound-translation across *pebble.DB and *pebble.Snapshot, so Reader methods stay a few lines each"
    - "Plain Batch (not IndexedBatch) for all writes — the write path never needs read-your-writes before Commit"
    - "Self-describing length-framed export stream ([kind byte][uvarint length][protobuf bytes]) makes Import format-detection independent of read order — meta/nodes/edges/files can be interleaved in future without breaking the reader"
    - "go/packages-based import-graph test (not regex) is the actual enforcement of an architectural boundary — codified as archtest.TestNoPackageBypassesGraphStore with a self-check guarding against the test passing vacuously"

key-files:
  created:
    - internal/graphstore/store.go
    - internal/graphstore/pebble_store.go
    - internal/graphstore/batch.go
    - internal/graphstore/export.go
    - internal/graphstore/store_test.go
    - internal/graphstore/export_test.go
    - internal/graphstore/archtest/import_graph_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Task 1 (GraphStore + pebble_store.go + batch.go) and Task 2 (export.go) landed in a single GREEN commit: GraphStore's interface declares Export from the start (D-04/RESEARCH Pattern 2), so pebbleStore cannot satisfy the GraphStore interface — and therefore cannot compile — until Export exists. The plan's two RED test commits (store_test.go, then export_test.go) still preceded the implementation, preserving test-first intent even though the two tasks' GREEN implementations are mutually dependent at the interface level."
  - "Named the edge iteration interface EdgeIterator (not the plan prose's generic 'Iterator') to match RESEARCH Pattern 2's Reader sketch exactly and to leave room for a future NodeIterator/FileIterator without a naming collision."
  - "Added GetFile and GetMeta to Reader beyond the plan's explicit GetNode/IterateEdges, since export.go's round-trip test and Phase 3's future query needs both need a way to read files/meta back through the same consistent-snapshot Reader rather than reaching into pebbleStore internals."
  - "Chose a package-level Import(dst GraphStore, r io.Reader) function rather than a GraphStore.Import method — importing only needs the already-public Writer interface, so it doesn't need privileged access to pebbleStore internals and stays decoupled from the concrete implementation, same as any other GraphStore consumer."
  - "metaRecordName='schema' is the single well-known key name under the m/ namespace for the store-wide Meta record — chosen since D-03 describes meta/... as a namespace, not a single key, and future non-schema meta entries (if any) can share the namespace without colliding."

patterns-established:
  - "Pattern: every Reader/Writer method builds its key exclusively through internal/graphstore/keys.go's encoders (nodeKey, edgeKey, edgeSrcPrefix, fileKey, fileSubgraphPrefix, metaKey) — no raw []byte concatenation anywhere in pebble_store.go/batch.go/export.go"
  - "Pattern: DeleteFileSubgraph and exportNamespace both derive their scan/delete range from rangeUpperBound(prefix) — the same namespace-agnostic byte-successor helper from Plan 01-05, reused rather than reimplemented"
  - "Pattern: archtest is its own subpackage under internal/graphstore/archtest so it is itself an 'allowed importer' of graphstore's namespace prefix without needing a special-case exemption in the enforcement test"

requirements-completed: [INDX-05, ARCH-01]

coverage:
  - id: D1
    description: "GraphStore/Reader/Writer/EdgeIterator interfaces declared, with pebbleStore as the sole holder of *pebble.DB in the module — no other package constructs a raw []byte key or opens the engine directly"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "go build ./internal/graphstore/... && go vet ./internal/graphstore/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "Many lock-free reader goroutines on independent Pebble snapshots run alongside one writer committing a batch, with no data race and no reader observing a torn/partial write"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/store_test.go#TestConcurrentReadersSingleWriter (run with -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "DeleteFileSubgraph issues a single range-delete that removes exactly the target file's own record, leaving a lexicographically adjacent sibling file untouched"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/store_test.go#TestDeleteFileSubgraphPrunesOnlyThatFile"
        status: pass
    human_judgment: false
  - id: D4
    description: "A bulk graph export (meta, nodes, edges, files) streamed as self-describing length-framed protobuf records re-imports losslessly into a fresh store, verified record-for-record via protocmp"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "internal/graphstore/export_test.go#TestBulkExportReimportsLosslessly"
        status: pass
    human_judgment: false
  - id: D5
    description: "Export is taken from a single consistent snapshot — a write committed after Export returns does not leak into the already-captured export stream"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "internal/graphstore/export_test.go#TestBulkExportIsConsistentUnderConcurrentWrite"
        status: pass
    human_judgment: false
  - id: D6
    description: "No package outside internal/graphstore (or its own subpackages) imports cockroachdb/pebble/v2 directly — enforced via a go/packages import-graph inspection, not regex/string matching"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest/import_graph_test.go#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-07-10
status: complete
---

# Phase 01 Plan 06: GraphStore Interface + Pebble/v2 Implementation Summary

**pebble/v2-backed GraphStore with lock-free snapshot reads, single-batch atomic writes, a self-describing bulk export/import stream, and a go/packages-enforced "no bypass" boundary test**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-07-10T17:01Z
- **Completed:** 2026-07-10T17:13Z
- **Tasks:** 3
- **Files modified:** 9 (7 created, 2 modified: go.mod, go.sum)

## Accomplishments
- `internal/graphstore/store.go`: `GraphStore` (Snapshot/NewWriter/Export/Close), `Reader` (GetNode/GetFile/GetMeta/IterateEdges/Close), `Writer` (PutNode/PutEdge/PutFile/PutMeta/DeleteFileSubgraph/Commit), and `EdgeIterator` (Next/Edge/Err/Close) — the single integration seam every later phase's indexer/query/sync/migration binds to (D-04)
- `internal/graphstore/pebble_store.go`: `pebbleStore` is the sole holder of a `*pebble.DB` in the whole module; `Snapshot()` wraps `db.NewSnapshot()` for lock-free, point-in-time reads; all reads translate `pebble.ErrNotFound` into a package-local `ErrNotFound` sentinel so callers never need to import pebble
- `internal/graphstore/batch.go`: `pebbleWriter` wraps a plain `db.NewBatch()` (never `NewIndexedBatch`, per the RESEARCH anti-pattern — this write path needs no read-your-writes); `Commit` applies atomically with `pebble.Sync`; `DeleteFileSubgraph` issues exactly one `DeleteRange` over the file's own key range
- `internal/graphstore/export.go`: `Export` streams Meta first, then Nodes/Edges/Files, as `[kind byte][uvarint length][protobuf bytes]` frames from a single snapshot — a concurrent writer cannot tear the export; `Import(dst, r)` replays that stream into a fresh store's batched `Writer`, committing once
- `internal/graphstore/archtest/import_graph_test.go`: `TestNoPackageBypassesGraphStore` loads the whole module's import graph via `golang.org/x/tools/go/packages` and fails if any package outside `internal/graphstore`'s prefix imports `pebble/v2` — with a self-check guarding against the test silently passing if graphstore itself ever stopped importing pebble
- `go test ./... -race -count=1` is green across all five packages (graphstore, archtest, parser, parser/cgo, schema) — the Wave 3 gate

## Task Commits

Each task was committed atomically (TDD RED-then-GREEN for Tasks 1 and 2, with the two GREEN implementations landing together — see Decisions Made):

1. **Task 1 RED: Concurrency test** - `50fd82c` (test) — `TestConcurrentReadersSingleWriter` + `TestDeleteFileSubgraphPrunesOnlyThatFile`, fails to compile (Open/GraphStore don't exist)
2. **Task 2 RED: Export round-trip test** - `58946a0` (test) — `TestBulkExportReimportsLosslessly` + `TestBulkExportIsConsistentUnderConcurrentWrite`, fails to compile (Export/Import/Open don't exist)
3. **Task 1 + Task 2 GREEN: Implementation** - `002263c` (feat) — store.go, pebble_store.go, batch.go, export.go, plus go.mod/go.sum promotion; both prior RED tests now pass
4. **Task 3: Import-graph boundary test** - `1e0d46c` (test) — `archtest.TestNoPackageBypassesGraphStore`

**Plan metadata:** (this commit)

_Note: per `<tdd_execution>`'s plan-level gate check — a `test(...)` commit precedes a `feat(...)` commit in git log for both TDD tasks; no REFACTOR commit was needed._

## Files Created/Modified
- `internal/graphstore/store.go` - GraphStore/Reader/Writer/EdgeIterator interface declarations (D-04)
- `internal/graphstore/pebble_store.go` - pebbleStore (sole *pebble.DB holder), pebbleReader, pebbleEdgeIterator, getProto helper, ErrNotFound sentinel
- `internal/graphstore/batch.go` - pebbleWriter (plain Batch wrapper): PutNode/PutEdge/PutFile/PutMeta/DeleteFileSubgraph/Commit
- `internal/graphstore/export.go` - Export (snapshot-consistent length-framed stream) + Import (ARCH-01 round trip)
- `internal/graphstore/store_test.go` - TestConcurrentReadersSingleWriter, TestDeleteFileSubgraphPrunesOnlyThatFile
- `internal/graphstore/export_test.go` - TestBulkExportReimportsLosslessly, TestBulkExportIsConsistentUnderConcurrentWrite
- `internal/graphstore/archtest/import_graph_test.go` - TestNoPackageBypassesGraphStore (D-04a)
- `go.mod` / `go.sum` - promoted `cockroachdb/pebble/v2` and `golang.org/x/tools` from indirect to direct; added `golang.org/x/sync` and `golang.org/x/mod` (transitive deps of `go/packages`); `wazero` untouched

## Decisions Made
- Task 1 and Task 2's GREEN implementations landed in one commit (`002263c`) rather than two, because `GraphStore.Export` is part of the interface declared in `store.go` from the start (D-04) — `pebbleStore` cannot satisfy `GraphStore`, and therefore cannot compile, until `export.go` exists. Both RED test commits still preceded this GREEN commit, preserving test-first sequencing at the commit-history level even though the two tasks are mutually dependent at the interface level.
- Named the edge-iteration type `EdgeIterator` (RESEARCH Pattern 2's naming) rather than the plan prose's generic "Iterator", leaving room for a future `NodeIterator`/`FileIterator` without a collision.
- Added `GetFile`/`GetMeta` to `Reader` beyond the plan's explicitly-named `GetNode`/`IterateEdges`, since the export round-trip test (and Phase 3's future query needs) need to read files/meta back through the same consistent-snapshot `Reader`.
- `Import` is a package-level function `Import(dst GraphStore, r io.Reader) error`, not a `GraphStore` method — it only needs the public `Writer` interface, so it stays decoupled from `pebbleStore` internals like any other `GraphStore` consumer would be.
- `golang.org/x/sync` and `golang.org/x/mod` were added via `go get <pkg>@latest` (not `go mod tidy`) to satisfy `go/packages`'s own transitive requirements — confirmed both are official `golang.org/x/*` extended-standard-library modules (same publisher as `x/tools`, `x/exp`, `x/sys`, `x/text` already in `go.mod`), not new/unvetted third-party packages, so this falls outside the package-install exclusion's slopsquatting concern. `wazero` was verified untouched afterward.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added golang.org/x/sync and golang.org/x/mod as transitive dependencies**
- **Found during:** Task 3 (import-graph boundary test)
- **Issue:** `go test ./internal/graphstore/archtest/...` failed at setup with "missing go.sum entry for module providing package golang.org/x/sync/errgroup (imported by golang.org/x/tools/go/packages)", then similarly for `golang.org/x/mod/semver`
- **Fix:** `go get golang.org/x/sync@latest` and `go get golang.org/x/mod@latest` — both are official `golang.org/x/*` modules, already trusted alongside `x/tools` (explicitly required by this plan) and `x/exp`/`x/sys`/`x/text` (already in go.mod); no `go mod tidy` was run, so `wazero` and the pinned-but-still-unimported deps from 01-01 were left untouched
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` all pass; `grep wazero go.mod` still shows it present
- **Committed in:** `002263c` (bundled with the Task 1+2 GREEN commit, since both fixes were needed to get the whole package graph compiling before Task 3's test could even run)

---

**Total deviations:** 1 auto-fixed (1 blocking dependency resolution)
**Impact on plan:** Necessary to satisfy `go/packages`'s own import requirements for Task 3; no scope creep — both added modules are official Go extended-standard-library packages already implicitly trusted by this plan's explicit `golang.org/x/tools` requirement.

## Issues Encountered

None beyond the dependency-resolution deviation above. Pebble/v2's exact API shapes (`Snapshot.Get`, `Snapshot.NewIter`, `Batch.Set/DeleteRange/Commit`, `pebble.Sync`, `pebble.ErrNotFound`) were confirmed against the locally-cached `v2.1.6` module source (not just Context7 snippets) before writing `pebble_store.go`/`batch.go`, so no API-mismatch rework was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/graphstore`'s `GraphStore`/`Reader`/`Writer`/`EdgeIterator` interfaces are the stable integration seam for Phase 2 (indexer writes), Phase 3 (query/MCP reads), Phase 4 (sync writes + snapshots), and Phase 7 (migration writes) — no further design work needed before those phases bind to it
- INDX-05 (concurrent lock-free store behind an interface no package bypasses) and ARCH-01's bulk-export requirement are both closed and proven by automated tests (`-race` clean, `-count=1` clean, `go/packages`-enforced boundary)
- No blockers for Plan 01-07 (parser backend spike) — this plan's work is storage-layer only and does not touch `internal/parser`

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*

## Self-Check: PASSED

- FOUND: internal/graphstore/store.go
- FOUND: internal/graphstore/pebble_store.go
- FOUND: internal/graphstore/batch.go
- FOUND: internal/graphstore/export.go
- FOUND: internal/graphstore/store_test.go
- FOUND: internal/graphstore/export_test.go
- FOUND: internal/graphstore/archtest/import_graph_test.go
- FOUND: commit 50fd82c (test: concurrency RED)
- FOUND: commit 58946a0 (test: export round-trip RED)
- FOUND: commit 002263c (feat: Task 1+2 GREEN)
- FOUND: commit 1e0d46c (test: archtest boundary)
