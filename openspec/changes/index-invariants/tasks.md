# Tasks

Run every target as `mise exec -- task <target>` in a shell without `mise activate`. Run `task test` and `task lint` one after the other, not at the same time. Write each test first and watch it fail on `main` before the fix.

## 1. The invariant helpers

- [ ] 1.1 Add `pgregory.net/rapid` at its latest release with `go get pgregory.net/rapid@<version>`. Move its `go.mod` line to the direct block by hand, because `go mod tidy` fails on `main`. Record the version and its license in the PR body for the minimal-dependency audit.
- [ ] 1.2 Add the test helper package `internal/indexer/invariant` with `Check(store) (Counts, error)` and `Equal(a, b) error`, as design.md D6 describes. The helpers get no tests of their own. The failing tests in section 2 prove them.

## 2. The failing invariant tests

- [ ] 2.1 Add the fixture `internal/indexer/testdata/repeatfixture/`. Its TypeScript file declares `function nextGeometry()` in three `it(...)` callbacks. It also declares `export const e = () => {}` above a nested `function e()`. It also declares `class Model` in two callbacks, each with its own method. One line holds two callbacks that each declare `function helper()`. A second TypeScript file imports the first `e` and calls it. Its Go file has a grouped `var` block.
- [ ] 2.2 Add `TestEveryDeclarationHasOneNode` with the table of design.md D8. It matches occurrences exactly, by name, kind, start line and start column, and it asserts that it matched more than zero declarations in each fixture. It logs each named exception on every run. Verify that it fails on `main` for `repeatfixture`.
- [ ] 2.3 Add `TestRepeatedClassesOwnTheirMethods`: index `repeatfixture` and assert that `invariant.Check` passes, including its `contains` range rule for lexical nesting. Assert the exact expected pairs: the first `Model` contains only its own method, and the second `Model` contains only its own method. Verify that `invariant.Check` also passes on the existing Go fixture with methods outside the type body. Verify that it fails on `main`.
- [ ] 2.4 Add `TestReferenceResolvesToFirstCopy`: the call in the second file targets the first `e` in source order. Verify that it fails on `main`, or record in the PR body why it passes there.
- [ ] 2.5 Add `TestNodeCountMatchesStoreAfterSyncs`: index `repeatfixture`, edit and sync the TypeScript file ten times, and assert that `invariant.Check` passes after each sync. Verify that it fails on `main` (#188).
- [ ] 2.6 Add `TestSyncEqualsFreshIndex` (design.md D7). Verify that it fails on `main`, and record the shrunk failing sequence in the PR body.

## 3. Unique declaration ids (#198)

- [ ] 3.1 Add `nodeid.Allocator` with `Declare(kind, qualifiedName, startByte)` and `Finalize(result)` (design.md D1). Add `TestFinalizeOrdersBySourcePosition`: declare three copies out of source order. The copy with the lowest start byte gets `NodeID`, and the other two get two different ordinal ids. A second run gives the same three ids. Add `TestFinalizeRewritesEveryIDField`: a result whose nodes, edges, `Unresolved`, `InterfaceMethods` and `MethodArity` hold provisional ids comes out with only final ids. A result with a provisional id in a field that `Finalize` does not know fails. Verify `go test ./internal/indexer/nodeid -count=1` passes.
- [ ] 3.2 Use `Declare` at each declaration site, and call `Finalize` once per file, in each extractor. Add `TestFinalizedIDsReachResolution` over `repeatfixture`. A call from the second `nextGeometry` gets an edge whose source is that copy's final id. A repeated declaration that records method metadata keeps that metadata under its final id. The extractors are `goextract`, `tsextract`, `pyextract`, `javaextract`, `csharpextract`, and the `mainstream` extractors for C, Kotlin, PHP, Ruby, Rust and Swift. The C# partial-type path keeps `NodeID` (design.md D2). Verify that `TestEveryDeclarationHasOneNode` passes and that the existing C# partial-class tests pass unchanged.
- [ ] 3.3 Change each name-keyed parent map in the extractors to the occurrence (design.md D2). Start with `typeNodesByName` in `tsextract`, then the Python, Java, C#, PHP and Ruby extractors. Verify that `TestRepeatedClassesOwnTheirMethods` passes.
- [ ] 3.4 Make the symbol index pick the first copy in each group, then the lowest id across groups (design.md D3). Add `TestSymbolPickIgnoresInsertionOrder` with three candidates: A1, its copy A2, and B. The test picks names whose hashes sort A2, B, A1. All six insertion orders give the same winner: B, the lower id of the two group winners A1 and B. A2 never wins. Run it through the full index and through the store-seeded `Sync` path. Verify that `TestReferenceResolvesToFirstCopy` also passes after both `codegraph index` and an incremental `Sync`.
- [ ] 3.5 Run `task test:golden` with the corpora present, or let `corpora.yml` run it on the PR. If a golden changes, make sure that each change is a new node for a repeated declaration, and regenerate it. Note each changed golden in the PR body.

## 4. Counts, the rebuild flag and a clean rebuild (#188)

- [ ] 4.1 Count distinct ids in `writeGraph` and in the `nodesAdded` loop of `Sync` (design.md D4). Verify that `TestNodeCountMatchesStoreAfterSyncs` passes.
- [ ] 4.2 Add `bool has_unique_decl_ids = 15;` to `Meta` in `internal/schema/graph.proto`. Install the web dependencies with `task web:deps`, regenerate with `task proto:gen`, and verify that `task proto:drift` passes.
- [ ] 4.3 Add `stampCapabilities(meta)` and call it from `writeGraph` and from both incremental `Meta` writers in `Sync` (design.md D5). Read the new flag in `needsFileIndexBackfill`.
- [ ] 4.4 Add `DeleteGraph` to the graph writer, and call it first in `writeGraph`, in the same batch (design.md D5). Add `TestBackfillReplacesOldGraph`. It writes a store, adds a node and an edge that no source file declares, clears the flag and sets `NodeCount` to a wrong value. Then it runs `Sync` three times, with one file changed before the second and third runs. The first sync is a full rebuild, and `invariant.Equal` finds no difference between its result and a fresh index. The second and third syncs each reparse only the changed file.
- [ ] 4.5 Verify that `TestSyncEqualsFreshIndex` passes with the default number of runs, and with `-rapid.checks=2000` once on a local run.

## 5. The watcher and key properties

- [ ] 5.1 Add `TestWatcherCoversEveryDirectory` to `internal/watch`, with the Linux and macOS assertions of design.md D9. It asserts that each run made at least one step. Verify that it passes on macOS locally and on Linux in CI.
- [ ] 5.2 Add `FuzzSegmentRoundTrip` and `FuzzSegmentBoundaries` to `internal/graphstore` (design.md D10). The seeds are the empty string, a string with a zero byte, and strings of 127 and 128 bytes (the varint boundary). Verify that `go test ./internal/graphstore -run 'Fuzz' -count=1` runs the seeds, and that `go test ./internal/graphstore -fuzz FuzzSegmentRoundTrip -fuzztime 60s` finds nothing.
- [ ] 5.3 Add `.github/workflows/fuzz.yml` with a weekly cron and `workflow_dispatch`. It runs each fuzz target for five minutes and the `rapid` properties with `-rapid.checks=10000`. It uploads `testdata/fuzz/` and the `rapid` failure file on a failure. Pin each action by SHA. Verify `task lint:actions` passes. Then run it once with `gh workflow run fuzz.yml --ref <branch>`, and link the run in the PR body.
- [ ] 5.4 Run `TestEveryDeclarationHasOneNode` over the four pinned corpora in `corpora.yml` through `CODEGRAPH_CORPUS_DIR` (design.md D8). Verify on the PR that the job runs it and reports more than zero declarations for each of hugo, guava, requests and serilog.

## 6. Whole-suite checks and the changelog

- [ ] 6.1 Verify `task test` and `task lint` pass.
- [ ] 6.2 Add the fragment: `task changie -- new --interactive=false -k Fixes -b "Each repeated declaration in a file now gets its own node, the node count in the sync summary matches the store, and an existing index rebuilds once on its first sync."`
