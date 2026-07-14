---
phase: 04-incremental-sync-file-watcher
plan: 03
subsystem: database
tags: [pebble, graphstore, incremental-index, sync, tdd, go]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "Plan 04-01's x/ file-owned secondary index (IterateFileIndex, DeleteNode/DeleteEdge, extended DeleteFileSubgraph, PutEdge(e, ownerPath))"
  - phase: 04-incremental-sync-file-watcher
    provides: "Plan 04-02's File.mtime_unix_ns/size_bytes + Meta.has_file_index additive fields, exported query.BuildReverseAdjacency"
provides:
  - "internal/indexer.Sync(repoRoot, storeDir, opts) (Stats, error) — the incremental indexing entry (INDX-03): one routine for `codegraph sync`, the debounced daemon cycle, and MCP-reconnect reconcile"
  - "internal/indexer.newSymbolIndexFromStore(r, modulePath, exclude) — store-seeded symbol index seam (symbolIndex.overlay shared with newSymbolIndex)"
  - "internal/indexer.resolveRefsWithIndex(results, modulePath, idx) — resolveRefs' idx-injectable implementation"
  - "Exported internal/indexer.ShouldSkipDir(name) — the watcher's future exclusion rule"
  - "DiscoveredFile.MtimeUnixNs/SizeBytes; File.mtime_unix_ns/size_bytes stamped on every PutFile (index and sync alike)"
  - "Extended indexer.Stats: FilesReparsed, FilesPruned, NodesRemoved, EdgesRemoved, DependentsRecomputed"
affects: [04-05, 04-06, 04-07, 04-08, watcher, daemon, mcp-reconnect, cli-sync]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Store-seeded resolve seam: newSymbolIndexFromStore(r, modulePath, exclude) + idx.overlay(batchResults), so cross-file resolution survives touching only a bounded batch"
    - "Prune-then-write single Writer/single Commit: IterateFileIndex(path) drives DeleteNode/DeleteEdge before DeleteFileSubgraph(path); PutNode/PutFile/PutEdge for the batch land on the SAME Writer"
    - "Delta Meta counting: NodeCount = base - nodesRemoved + nodesAdded (nodesAdded excludes ids already present-and-unpruned, so a re-staged dependent/package node never double-counts); EdgeCount = base - edgesRemoved + len(collapsedEdges) (safe unconditionally, since every owned edge of a pruned/dependent file is deleted before the batch regenerates it)"
    - "Backfill probed in isolation (needsFileIndexBackfill opens/closes its own store handle) before Sync's own graphstore.Open, so the D-02b backfill's delegated `run()` never double-opens the same Pebble directory"

key-files:
  created:
    - internal/indexer/sync.go
    - internal/indexer/sync_test.go
  modified:
    - internal/indexer/symbolindex.go
    - internal/indexer/resolve.go
    - internal/indexer/discover.go
    - internal/indexer/extract.go
    - internal/indexer/goextract/types.go
    - internal/indexer/pipeline.go

key-decisions:
  - "internal/indexer does NOT import internal/query for BuildReverseAdjacency, despite 04-RESEARCH.md and 04-02-SUMMARY.md both asserting that edge is safe — internal/query's own white-box test files (engine_test.go et al., package query) import internal/indexer to seed fixtures via indexer.Run, so indexer->query is a real Go-toolchain-enforced 'import cycle not allowed in test' once indexer also imports query. Fixed by duplicating the ~15-line reverse-adjacency scan as a package-private internal/indexer.buildReverseAdjacency, rather than refactoring Phase 3's entire (8-file) white-box test suite to black-box `query_test` just to unblock this edge."
  - "The store-seeded symbol index's exclude set is (reparse batch paths) UNION (deleted paths) — RESEARCH's own text only says 'the reparse batch, about to be superseded'. Deleted files are never re-extracted (nothing to overlay back in), so if the store-seeded scan didn't also exclude them, a deleted symbol would remain incorrectly resolvable from the stale store scan taken before the prune Writer's deletes are committed."
  - "writeGraph (shared by full-index Resolve/Run AND used as the model for Sync's own write step) now unconditionally stamps Meta.HasFileIndex=true: 04-01 already made every PutNode/PutEdge populate the x/ index regardless of caller, so any graph committed through the current code genuinely has a complete x/ index by commit time — the flag being false is now only possible for a genuinely pre-Phase-4 store."
  - "resolveRefs(results, modulePath) is preserved byte-for-byte as a public call signature — no existing call site (including resolve_test.go's ~9 sites) needed updating — by having it delegate to the new resolveRefsWithIndex(results, modulePath, idx) rather than adding an idx parameter directly to resolveRefs itself."

patterns-established:
  - "Sync's Extract batch is bounded by (added ∪ modified ∪ direct dependents) — never widened to a second-order transitive closure; a symbol pruned during a modified/deleted file's wholesale subgraph delete conservatively marks EVERY file with an edge into it as a dependent, even when that symbol's node id (content-hash of kind+qualifiedName+filePath) turns out to be stable across the edit."

requirements-completed: [INDX-03]

coverage:
  - id: D1
    description: "Sync reparses only content-hash-changed files (stat pre-filter -> content-hash confirm), not the whole repo"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncReparsesOnlyChangedFiles"
        status: pass
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncNoOpWhenNothingChanged"
        status: pass
    human_judgment: false
  - id: D2
    description: "A reference from an unchanged file to a changed file still resolves after sync (store-seeded symbol index, Pitfall 1)"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncResolvesAcrossUnchangedFiles"
        status: pass
    human_judgment: false
  - id: D3
    description: "A dependent file whose call target moved/changed is re-extracted so its edge regenerates against the new node id (Pitfall 2)"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncReExtractsDependents"
        status: pass
    human_judgment: false
  - id: D4
    description: "Sync prunes changed/deleted files' subgraphs via the x/ index and commits prune+write in one atomic Commit"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncPrunesDeletedFile"
        status: pass
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncDetectsAddedFile"
        status: pass
      - kind: other
        ref: "grep -c 'NewWriter()\\|\\.Commit()' internal/indexer/sync.go == 2 (one of each in Sync's non-backfill path)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A pre-Phase-4 graph (Meta.has_file_index false/absent) triggers a one-time full backfill, then goes incremental (D-02b)"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_test.go#TestSyncBackfillsPrePhase4Graph"
        status: pass
    human_judgment: false
  - id: D6
    description: "Whole-repo build/vet/race-test suite and the archtest Pebble-import boundary stay green after Sync's introduction"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && go test ./... -race -count=1"
        status: pass
      - kind: unit
        ref: "internal/graphstore/archtest#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 3: Incremental Sync Engine (internal/indexer.Sync) Summary

**Store-seeded incremental `Sync()` — stat/content-hash diff, x/-index subgraph prune, dependent Pass-1 re-extraction via a package-local reverse-adjacency scan, and a D-02b one-time backfill for pre-Phase-4 graphs, all landing in one atomic Writer/Commit.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-11T19:14:10Z
- **Completed:** 2026-07-11T19:19:51Z
- **Tasks:** 3 (RED, GREEN, GREEN)
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments
- `internal/indexer.Sync(repoRoot, storeDir, opts) (Stats, error)` — store-seeded incremental resolve reusing `Discover`/`Extract`/`resolveRefsWithIndex`/`writeGraph`'s building blocks, not a second pipeline (D-01)
- Stat pre-filter (`mtime`/`size`) then content-hash confirm classifies every discovered file into added/modified/unchanged; diffing stored `IterateFiles()` against the discovered set finds deletions (D-01a)
- Prune step walks each changed/deleted file's `x/` index (`IterateFileIndex`) and stages `DeleteNode`/`DeleteEdge` before `DeleteFileSubgraph`, per 04-01's documented ordering constraint
- Dependent detection via a package-local reverse-adjacency scan finds every file with an edge into a just-pruned node id, narrow-prunes only their owned edges (nodes/File record survive), and folds them into the Extract batch (D-02a, Pitfall 2)
- Store-seeded symbol index (`newSymbolIndexFromStore` + `symbolIndex.overlay`) so a new/modified file's reference into an untouched file still resolves (Pitfall 1)
- D-02b backfill: `needsFileIndexBackfill` probes `Meta.has_file_index` in an isolated store open/close, delegating to the from-scratch `run()` path when absent/false — the very next Sync is then incremental
- Exactly one `NewWriter()`/one `Commit()` in Sync's non-backfill path (grep-verified); `Stats` gains `FilesReparsed`/`FilesPruned`/`NodesRemoved`/`EdgesRemoved`/`DependentsRecomputed`

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing tests for store-seeded incremental Sync** - `a296848` (test)
2. **Task 2 (GREEN): store-seeded index, resolve idx-injection, shouldSkipDir, mtime/size stamping** - `61aea4d` (feat)
3. **Task 3 (GREEN): Sync() orchestration — diff, prune, dependents, backfill, atomic commit** - `d640568` (feat)

_TDD plan: RED commit precedes both GREEN commits; the full test/vet/build gate could only pass once Task 3 landed, since sync_test.go (Task 1) references `Sync` — inherent RED-state carryover for a multi-file TDD plan, not a separate deviation._

## Files Created/Modified
- `internal/indexer/sync.go` - New: `Sync`, `needsFileIndexBackfill`, `pruneFileSubgraph`, `pruneOwnedEdgesOnly`, `buildReverseAdjacency` (package-local, see Decisions), `contentHash`
- `internal/indexer/sync_test.go` - New: 7 `TestSync*` tests plus `writeFixture`/`writeFile`/`openSnapshot` test helpers
- `internal/indexer/symbolindex.go` - `newSymbolIndexFromStore`; `newSymbolIndex` refactored onto a shared `symbolIndex.overlay(results)` method
- `internal/indexer/resolve.go` - `resolveRefs` delegates to new `resolveRefsWithIndex(results, modulePath, idx)`; `File` records carry `MtimeUnixNs`/`SizeBytes`; `writeGraph` stamps `Meta.HasFileIndex=true`
- `internal/indexer/discover.go` - Exported `ShouldSkipDir(name)`; `DiscoveredFile` carries `MtimeUnixNs`/`SizeBytes` from `WalkDir`'s `DirEntry.Info()`
- `internal/indexer/extract.go` - Threads `DiscoveredFile.MtimeUnixNs`/`SizeBytes` onto every `goextract.FileResult` (success and read-failure paths)
- `internal/indexer/goextract/types.go` - `FileResult.MtimeUnixNs`/`SizeBytes` fields
- `internal/indexer/pipeline.go` - `Stats` gains `FilesReparsed`/`FilesPruned`/`NodesRemoved`/`EdgesRemoved`/`DependentsRecomputed` (additive, zero-valued for `Run`)

## Decisions Made
- **Broke the plan's stated `indexer -> query` import** (see key-decisions above for the full cycle explanation) — a package-local `buildReverseAdjacency` duplicate in `sync.go` instead. `query.BuildReverseAdjacency` remains exported and unchanged, still used by `Callers`/`Impact`/`Affected`.
- **Exclude set for the store-seeded index is batch-paths UNION deleted-paths**, not just "the reparse batch" as RESEARCH's prose literally says — necessary so a deleted file's symbols don't remain spuriously resolvable via the pre-prune snapshot `r0`.
- **`writeGraph` now always stamps `Meta.HasFileIndex=true`** — true for every commit made through current code, since 04-01's `PutNode`/`PutEdge` unconditionally populate `x/`. This makes D-02b's flag meaningful: only a genuinely pre-04-01 store can ever have it false.
- **Meta.NodeCount delta arithmetic excludes already-present-and-unpruned ids from the "added" count** — otherwise a dependent file's own (content-unchanged, never-pruned) nodes, or a re-minted `kindPackage` pseudo-node with a stable id, would double-count on every sync that touches them. This required one `r0.GetNode` existence check per batch node (bounded by batch size, not full-graph) — acceptable given 04-03's scope explicitly excludes 100k-file/peak-RSS performance tuning (Phase 8).
- **`resolveRefs`'s public signature is untouched** — the idx-injection seam is a new `resolveRefsWithIndex` function `resolveRefs` delegates to, avoiding any edit to `resolve_test.go`'s ~9 existing call sites (a smaller blast radius than the alternative the plan explicitly offered).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `internal/indexer` importing `internal/query.BuildReverseAdjacency` creates a real Go test-compilation cycle**
- **Found during:** Task 3 (`go vet ./...` after wiring `Sync()`'s dependent-detection step)
- **Issue:** 04-RESEARCH.md and 04-02-SUMMARY.md both asserted `internal/indexer -> internal/query` was a "new, safe, non-circular edge," checked only against `internal/query`'s PRODUCTION imports of `internal/indexer/goextract`. `internal/query`'s own white-box test files (`engine_test.go`, `explore_test.go`, `node_test.go`, `search_test.go`, `traverse_test.go`, `render_markdown_test.go`, `files_status_test.go` — all `package query`) import `internal/indexer` directly (to seed real fixtures via `indexer.Run`). Once `internal/indexer` (production `sync.go`) also imports `internal/query`, `go vet`/`go test` on `internal/query` fails: `"import cycle not allowed in test"`.
- **Fix:** Duplicated `query.BuildReverseAdjacency`'s ~15-line algorithm as a package-private `internal/indexer.buildReverseAdjacency`, called from `Sync()` instead of the exported `query` function. `query.BuildReverseAdjacency` itself is untouched — still exported, still the implementation `Callers`/`Impact`/`Affected` use. Converting all 7 of `internal/query`'s white-box test files to black-box `query_test` (the "true" architectural fix restoring a single implementation) was judged disproportionate blast radius for this plan — it would touch Phase 3's entire, already-shipped test suite to unblock a single new production import.
- **Files modified:** `internal/indexer/sync.go` (import swap, new unexported function)
- **Verification:** `go build ./... && go vet ./... && go test ./... -race -count=1` all green; `internal/graphstore/archtest` boundary test still green
- **Committed in:** `d640568` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — corrected an inaccurate cross-package-safety claim inherited from 04-RESEARCH.md/04-02-SUMMARY.md)
**Impact on plan:** Necessary for the whole repository's test suite (not just this package) to build/vet/test cleanly. No scope creep — the fix is contained entirely within `sync.go`; `internal/query` was not touched.

## Issues Encountered
None beyond the import-cycle deviation documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/indexer.Sync` is the single incremental entry point Plans 04-05 (watcher), 04-06 (MCP reconnect), 04-07 (daemon), and 04-08 (`codegraph sync` CLI) all call — no further indexer-side plumbing is needed for those plans to invoke incremental sync.
- `ShouldSkipDir` is exported specifically for Plan 04-05's watcher to reuse the identical directory-exclusion rule Discover already applies.
- `Stats`' new fields (`FilesReparsed`/`FilesPruned`/`NodesRemoved`/`EdgesRemoved`/`DependentsRecomputed`) are ready for `codegraph sync`'s summary line (D-01b, Plan 04-08) without further Stats changes.
- The `sync-equals-reindex` byte-identical determinism property (Phase-2's determinism gate surviving incremental update) is explicitly OUT of this plan's own `<verification>`/`<success_criteria>` — 04-RESEARCH.md's Validation Architecture names a separate `TestSyncEqualsReindex`/`sync_determinism_test.go` as its own Wave-0 gap; whichever later plan closes that gap should be aware Meta.NodeCount/EdgeCount here are computed via delta arithmetic (base -removed +added) rather than a full post-commit recount, and should verify that arithmetic holds under a real byte-identical comparison, not just the functional graph-state assertions this plan's tests make.
- No behavior for `internal/watch`, `internal/daemon`, or the `sync`/`daemon`/`unlock` CLI commands was built in this plan — those remain for their own dedicated plans per 04-CONTEXT.md's Claude's Discretion / canonical_refs sections.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED

All key files and all three task commits (a296848 test/RED, 61aea4d feat/GREEN, d640568 feat/GREEN) verified present.
