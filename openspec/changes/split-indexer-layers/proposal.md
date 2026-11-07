# Proposal: Split the indexer into layers with one job each

Issue: #270

## Why

`internal/indexer` holds discovery, the extraction worker pool, symbol lookup, resolution, the graph write and sync in one package. `Resolve` takes a `graphstore.GraphStore`, `writeGraph` sits beside the resolver, and state that one file needs lives in maps that last the whole run. That is where the memory overhead of #250 hid (#265). The bounded parallel pipeline of #271 needs clean seams between these jobs. A split that changes no behavior comes first, so that the concurrency change stays small and testable.

## What Changes

- Discovery moves to `internal/indexer/discover`. Per-file extraction and the language registry move to `internal/indexer/extract`. Both are synchronous and never open the store.
- The symbol index moves to `internal/indexer/symbols`. A `Builder` collects declarations. `Freeze` returns a separate read-only `View` type, so the compiler rejects a write after the barrier. A second frozen type holds the type hierarchy and the overload sets, which resolution needs before it can resolve calls.
- Resolution moves to `internal/indexer/resolve`. It resolves one file at a time against the frozen views, as a pure function. A separate whole-graph `Link` step adds the edges that need every file: synthesized `implements` and `overrides`, the conformance retry, and the overload-set rewiring. Neither step opens the store or starts a goroutine.
- `internal/indexer` keeps the orchestration: `Run`, `Sync`, the extraction worker pool, the reading of the stored graph for a sync, and route detection wiring. It defines a `Sink` interface for staging and committing records. `graphstore.Writer` implements it. The full index and sync stage records through one shared function, in place of the two copies they have today.
- A `depclosure` test enforces the import direction. Each layer imports only the layers below it, and no layer below orchestration reaches `internal/graphstore`, even through another package.
- `ShouldSkipDir` moves with discovery. Its callers in `internal/watch`, `internal/daemon` and `internal/query` take the new import path. `Run`, `Sync`, `Options`, `Stats`, `NeedsFullIndex` and `ErrTooManyChanges` keep their names and signatures in `internal/indexer`.
- After the split, `docs/` gets an Archify architecture diagram of the indexer layers, drawn from the code after the split.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `graph-index`: adds the requirement that a full index of the same tree gives the same graph (INDX-02). The planning archive records it, and `TestDeterministicRebuild` covers it, but no spec states it. The split must keep it, and #271 changes the order in which files are processed. So the spec names it before either change.

No other spec changes. The stored graph stays byte-identical, and no CLI output or MCP result changes.

## Out of scope

- Concurrency, streaming or any memory bound. Those belong to #271. Orchestration keeps today's order: all of pass 1, then all of pass 2, then one commit.
- Route detection in sync. `Sync` does not run route detection today. Adding it changes the graph.
- Moving `goextract.FileResult` and the other shared extraction types out of `goextract`. The language packages and resolution keep importing them from there.
- Changes to the language extractors, node ids, edge collapse rules or the store format.
- Trend tracking of benchmark results (#269).

## Impact

- Code: every non-test file of `internal/indexer` moves or changes. The language packages under `internal/indexer` keep their import paths. So do `nodeid`, `dispatch`, `capability` and `routes`.
- Tests: white-box tests move with the code they cover. Tests that drive `Run` or `Sync` stay in `internal/indexer`.
- Callers: `internal/watch`, `internal/daemon` and `internal/query` change one import each for `ShouldSkipDir`.
- Gates: one new `depclosure` test. No CI job is renamed.
- No changelog fragment. The PR body carries a `changelog-exempt` marker, because users see no change.
