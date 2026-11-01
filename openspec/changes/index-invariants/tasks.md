# Tasks

Run every target as `mise exec -- task <target>` in a shell without `mise activate`. Run `task test` and `task lint` one after the other, not at the same time. Write each test first and watch it fail on `main` before the fix.

## 1. The invariant helpers

- [ ] 1.1 Add `pgregory.net/rapid` at its latest release with `go get pgregory.net/rapid@<version>`. Move its `go.mod` line to the direct block by hand, because `go mod tidy` fails on `main`. Record the version and its license in the PR body for the minimal-dependency audit.
- [ ] 1.2 Add the test helper package `internal/indexer/invariant` with `Check(store) (Counts, error)` and `Equal(a, b) error`, as design.md D5 describes. Add `TestCheckFindsDanglingEdge` and `TestEqualReportsFirstDifference`, which build small stores by hand. Verify `go test ./internal/indexer/invariant -count=1` passes.

## 2. The failing invariant tests

- [ ] 2.1 Add the fixture `internal/indexer/testdata/repeatfixture/`. Its TypeScript file declares `function nextGeometry()` in three `it(...)` callbacks. Its Go file has a grouped `var` block.
- [ ] 2.2 Add `TestEveryDeclarationHasOneNode` (design.md D7) for Go and TypeScript. It asserts that it counted more than zero declarations in each fixture. Verify that it fails on `main` for `repeatfixture`, with three TypeScript declarations against one id.
- [ ] 2.3 Add `TestNodeCountMatchesStoreAfterSyncs`: index `repeatfixture`, edit and sync the TypeScript file ten times, and assert that `invariant.Check` passes after each sync. Verify that it fails on `main` (#188).
- [ ] 2.4 Add `TestSyncEqualsFreshIndex` (design.md D6). Verify that it fails on `main`, and record the shrunk failing sequence in the PR body.

## 3. Unique declaration ids (#198)

- [ ] 3.1 Add `nodeid.Allocator` with `Declare(kind, qualifiedName) string` (design.md D1). Add `TestAllocatorFirstCopyKeepsNodeID`: the first `Declare` equals `NodeID`. Add `TestAllocatorNumbersRepeats`: the second and third calls give two new, different ids. A new allocator over the same calls gives the same three ids. Verify `go test ./internal/indexer/nodeid -count=1` passes.
- [ ] 3.2 Replace the declaration-site `NodeID` calls with `Declare` in each extractor. The extractors are `goextract`, `tsextract`, `pyextract`, `javaextract`, `csharpextract`, and the `mainstream` extractors for C, Kotlin, PHP, Ruby, Rust and Swift. The C# partial-type path keeps `NodeID` (design.md D2). Keep `NodeID` for file nodes, package nodes and ids of symbols in other files. Verify that `TestEveryDeclarationHasOneNode` passes and that the existing C# partial-class tests pass unchanged.
- [ ] 3.3 Run `task test:golden` with the corpora present, or let `corpora.yml` run it on the PR. If a golden changes, make sure that each change is a new node for a repeated declaration, and regenerate it. Note each changed golden in the PR body.

## 4. Counts and the rebuild flag (#188)

- [ ] 4.1 Count distinct ids in `writeGraph` and in the `nodesAdded` loop of `Sync` (design.md D3). Verify that `TestNodeCountMatchesStoreAfterSyncs` passes.
- [ ] 4.2 Add `bool has_unique_decl_ids = 15;` to `Meta` in `internal/schema/graph.proto` and regenerate with `task proto`. Set the flag in `writeGraph`, and read it in `needsFileIndexBackfill` (design.md D4). Add `TestSyncRebuildsStoreWithoutUniqueDeclIDs`: write a store, clear the flag and set `NodeCount` to a wrong value, then run `Sync`. Assert that the sync was a full rebuild and that `invariant.Check` passes. A second `Sync` with one changed file reparses only that file. Verify `task proto:drift` passes.
- [ ] 4.3 Verify that `TestSyncEqualsFreshIndex` passes with the default check count, and with `-rapid.checks=2000` once on a local run.

## 5. The watcher and key properties

- [ ] 5.1 Add `TestWatchSetEqualsDirectorySet` to `internal/watch` (design.md D8). It asserts that each run made at least one step. Verify that it passes on macOS locally and on Linux in CI.
- [ ] 5.2 Add `FuzzSegmentRoundTrip` and `FuzzSegmentOrder` to `internal/graphstore` (design.md D9). The seeds are the empty string, a string with a zero byte, and strings of 127 and 128 bytes (the varint boundary). Verify that `go test ./internal/graphstore -run 'Fuzz' -count=1` runs the seeds, and that `go test ./internal/graphstore -fuzz FuzzSegmentRoundTrip -fuzztime 60s` finds nothing.
- [ ] 5.3 Add `.github/workflows/fuzz.yml` with a weekly cron and `workflow_dispatch`. It runs each fuzz target for five minutes and the `rapid` properties with `-rapid.checks=10000`. It uploads `testdata/fuzz/` and the `rapid` failure file on a failure. Pin each action by SHA. Verify `task lint:actions` passes, then run it once with `gh workflow run fuzz.yml` from the PR branch and link the run in the PR body.
- [ ] 5.4 Run `TestEveryDeclarationHasOneNode` over the four pinned corpora in `corpora.yml` through `CODEGRAPH_CORPUS_DIR` (design.md D7). Verify on the PR that the job runs it and reports more than zero declarations for each corpus.

## 6. Whole-suite checks and the changelog

- [ ] 6.1 Verify `task test` and `task lint` pass.
- [ ] 6.2 Add the fragment: `task changie -- new --interactive=false -k Fixes -b "Each repeated declaration in a file now gets its own node, the node count in the sync summary matches the store, and an existing index rebuilds once on its first sync."`
