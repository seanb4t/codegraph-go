# Tasks

Run every target as `mise exec -- task <target>` in a shell without `mise activate`. Run `task test` and `task lint` one after the other, not at the same time. Each group ends in one commit that builds and passes `task test`. No group changes a test assertion. A test that moves keeps its name and its assertions.

## 1. Discovery and extraction

- [x] 1.1 Move `discover.go`, `discoverexclusion.go` and their tests to `internal/indexer/discover`. Export `Discover`, `DiscoverAll`, `DiscoveredFile`, `Discovery` and `ShouldSkipDir`. Change the `ShouldSkipDir` callers in `internal/watch`, `internal/daemon` and `internal/query` to the new import path. Verify that `go build ./...` and `task test` pass.
- [x] 1.2 Move `languages.go`, `languages_*.go` and `languages_test.go` to `internal/indexer/extract`, with `LanguageSpec` and the registry. Add `extract.File(spec, parser, moduleKey, relPath, src)`, the per-file step of today's worker loop, without the file read. The worker pool stays in `internal/indexer` and calls `extract.File`. Move the per-file cases of `extract_test.go` with it. Verify that `task test` passes and that `extract` imports no package that starts a goroutine or opens the store.

## 2. The symbol layer

- [x] 2.1 Add `internal/indexer/symbols` with `Builder`, `Builder.Freeze() View` and the lookups of today's `symbolIndex` on `View` (design.md D3). `Builder` takes declarations from pass 1 and from the stored graph through the same add methods (D7). `View` has no exported field and no method that writes. Move `symbolindex_test.go` with it. Add the `rapid` property `TestFreezeIgnoresInsertionOrder`: two builders fed one set of declarations in two generated orders answer every `candidatesIn`, `typeIn` and `members` lookup the same. Verify that `task test` passes.
- [x] 2.2 Add `symbols.NewHierarchy(view, typeEdges) Hierarchy` with the work of today's `learnEdges` and `declareSets`: supertypes, member owners and the declared overload sets (D3). `Hierarchy` has no method that writes. Verify that `TestOverloadsResolveAsTheLanguageDoes` and the overload-set tests pass unchanged.

## 3. The resolution layer

- [x] 3.1 Add `internal/indexer/resolve` with `Types(view, result) TypeOutput` and `Calls(view, hierarchy, result) CallOutput` (design.md D2). `Calls` keeps `callTargets` and the same-file declaration cache in local variables. `CallOutput` keeps the calls without a deferred type apart from the deferred-call edges and their sort keys (D6). Verify that `resolve` builds and that its closure holds no `graphstore`, with `go list -deps ./internal/indexer/resolve`.
- [x] 3.2 Add `resolve.Link(view, hierarchy, types, calls) Graph` with the synthesized `implements` and `overrides` edges, the conformance retry, the set rewiring, the `File` records and `collapseEdges`. Assemble edges and pending calls in the order of D6. Add the `rapid` property `TestLinkIgnoresResultOrder`: over a generated permutation of the results of `testdata/overloadfixture` and `testdata/repeatfixture`, `Link` gives an equal `Graph`. Verify that it passes.
- [x] 3.3 Make `Run` and `Sync` call `Types`, `NewHierarchy`, `Calls` and `Link`. Delete `resolveRefsWithIndex`, `resolveRefs` and the `resolver` struct. Move `resolve_test.go`, `resolution_test.go`, `samefile_test.go` and `deferredpos_test.go` to `resolve`, or make them drive `Run` where they read the store. Verify that `TestDeterministicRebuild`, `TestSyncEqualsReindex`, the invariant tests and `task test:golden` pass.

## 4. The sink and the store reads of a sync

- [x] 4.1 Declare `Sink` in `internal/indexer` with the `graphstore.Writer` methods that staging calls (design.md D5). Add `stage(sink, graph, ownerPath)`, which sorts and puts nodes, files and edges. Make `writeGraph` and the tail of `Sync` call it. Add `TestStagePutsRecordsInKeyOrder`: a recording `Sink` receives nodes by id, files by path and edges by source, kind and target, and it receives more than zero of each. Verify that `task test` passes.
- [x] 4.2 Make `loadStoreView` feed a `symbols.Builder` in place of building a `symbolIndex` (D7). Verify that `TestSyncEqualsReindex` and the sync property tests pass.

## 5. Orchestration tests and the layer gate

- [x] 5.1 Add `TestRebuildIgnoresWorkerCount` for the `graph-index` scenario "A different worker count". It indexes the shared fixture with `Workers: 1` and with `Workers: 8`, and compares the two `Export()` streams with the normalization of `TestDeterministicRebuild`. It asserts that each stream holds more than zero nodes and edges. Verify that it passes on `main` before the split and after group 3.
- [x] 5.2 Add `goleak.VerifyTestMain` to the `TestMain` of `internal/indexer`, after its `rapid.checks` default. Add `TestExtractStopsEveryWorkerOnError` under `testing/synctest`: a parser factory that fails on the third file ends `Extract` with that error, and no worker goroutine remains. Verify that both pass.
- [x] 5.3 Add `TestIndexerLayers` to `internal/depclosure` (design.md D1). Positive assertion: the `go list -deps` output lists each of the four layer packages. Also, the closure of `resolve` holds `symbols`, and the closure of `discover` holds `extract`. Negative assertion: no layer below `internal/indexer` has `internal/graphstore` or `internal/indexer` itself in its closure. Verify that it fails with a temporary `graphstore` import in `resolve`, then remove that import.

## 6. Equality and documentation

- [ ] 6.1 Build `main` and the branch. Run `codegraph index --force` with each on the synthetic corpus, this repository and Pebble. Hash the node and edge records of each store in key order, without `Meta`, with a throwaway program in the scratch directory. Record the counts and hashes in the PR body, as #268 did. Verify that every pair of hashes is equal.
- [x] 6.2 Rewrite the package comment in `doc.go` of `internal/indexer` and add one to each new package. Each states the job of its layer and what it does not import. Verify that `go doc ./internal/indexer/...` prints each one.
- [x] 6.3 Draw an Archify architecture diagram of the indexer layers from the code after the split, at `docs/architecture/indexer-layers.html`. It shows the five layers, the two freeze barriers, `Link` and the `Sink`. Verify that the page opens and that each package it names exists.
- [ ] 6.4 Record `<!-- changelog-exempt: internal refactor, the stored graph and every output stay the same -->` in the PR body.

## Workflow follow-up

- Open the proposal-only PR with `Refs #270`.
- The implementation PR says `Closes #270` and carries the code and the `openspec archive split-indexer-layers` result.
- After #270 merges, #271 builds the bounded pipeline on these seams.
