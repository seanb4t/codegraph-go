# Design

## Context

`nodeid.NodeID(kind, qualifiedName, filePath)` (`internal/indexer/nodeid/nodeid.go`) hashes three length-prefixed segments. Each extractor computes the id once per declaration and then passes it to the calls that emit the declaration's edges. For example, `tsextract.emitFunction` computes `id` and then calls `collectCalls(id, body)` and `collectReferencesAndInstantiates(id, body)`. So an id that the extractor chooses at declaration time also reaches every edge that starts in that declaration's body.

Other files reach a declaration through the resolver, which computes `NodeID` from a qualified name (RES-01). A reference cannot name a copy.

`writeGraph` (`internal/indexer/resolve.go:924`) sets `Meta.NodeCount = len(allNodes)`, repeats included. `Sync` (`internal/indexer/sync.go:482-494`) counts `nodesAdded` once for each entry in `allNodes`, so two new nodes with one id count twice. `PutNode` writes an id once, so the store holds fewer nodes than the count.

If a `Meta` flag is false, `needsFileIndexBackfill` (`internal/indexer/sync.go:625`) forces a full rebuild. #217 added `has_grouped_vars` (field 14 of `Meta` in `internal/schema/graph.proto`) the same way.

The key encoding is `appendSegment` and `decodeSegment` in `internal/graphstore/keys.go`, with example tests in `keyenc_test.go`.

The in-repository fixtures are under `internal/indexer/testdata/` (`gofixture`, `mixedlangfixture`, `prunefixture`, `routesfixture`). The four pinned corpora (hugo, guava, requests, serilog) are fetched only in `corpora.yml`, which runs `task test:golden`. `testdata/golden/corpus/` holds their golden outputs, not their sources.

## Goals / Non-Goals

**Goals:**

- Each declaration gets its own node, and the first copy keeps today's id.
- The stored node count is always the number of stored nodes.
- Properties and a fuzz target cover the invariant classes behind #187, #188, #193, #194, #197, #198, #208 and #214.
- Every new test has a positive assertion: if it inspects nothing, it fails.

**Non-Goals:**

- Scoped reference resolution for repeated declarations.
- New qualified names for anonymous scopes.
- A mutation tool (#222) or a BDD runner.

## Decisions

### D1. A per-file id allocator numbers repeated declarations

Add `nodeid.Allocator`, created once per extracted file. Its method `Declare(kind, qualifiedName) string` returns `NodeID(kind, qualifiedName, filePath)` the first time it sees a pair. For the n-th time (n of 2 or more), it returns the hash of four segments: the three of today, plus the decimal ordinal `n`. Each extractor calls `Declare` where it calls `NodeID` for a declaration today. Ids that name another file's symbol, file nodes and package nodes keep calling `NodeID`.

The extractors walk the tree in source order, so the ordinal is the source order of the copies. The first copy hashes the same three segments as today, so its id, and every resolved reference to it, does not change.

Alternatives considered:

- A pass after extraction that renumbers repeated ids. Both copies emit edges with the same source id, so such a pass cannot tell which edges belong to which copy.
- The start line in the hash. Then every edit above a declaration changes its id, and a sync deletes and recreates its node and edges.
- A name from the enclosing call, for example the `it(...)` description. Then the id depends on call arguments, and minified code has no such names.

### D2. C# partial types keep one id

`csharpextract` keys a partial type by its qualified name and namespace, with a deterministic sentinel path, so that all parts merge (`csharpextract/types.go:81-99`). That path keeps calling `NodeID`. Each partial part therefore resolves to the same node, as `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.

### D3. Counts come from distinct ids

`writeGraph` sets `NodeCount` to the number of distinct ids in `allNodes`. `Sync` counts a new id once. With D1 there are no repeats, but the count does not depend on that. `EdgeCount` does not drift (the #188 replay showed this), so it does not change.

### D4. A new `Meta` flag forces one rebuild

Add `bool has_unique_decl_ids = 15;` to `Meta`. `writeGraph` sets it. If the flag is false, `needsFileIndexBackfill` returns true. The rebuild is a full index, so it also writes a correct `NodeCount`. `SchemaVersion` stays 1, because the field is additive, as with `has_grouped_vars`.

### D5. One invariant checker, used by every test

Add a test helper package, `internal/indexer/invariant`, with `Check(store) error`. It asserts that each edge endpoint is a stored node and that `Meta.NodeCount` equals a scan of the store. It also returns the counts it scanned, so each caller asserts that the counts are above zero. A second helper, `Equal(a, b)`, compares two stores node by node and edge by edge, and reports the first difference.

### D6. Sync equals a fresh index, as a model-based `rapid` property

`TestSyncEqualsFreshIndex` uses `pgregory.net/rapid`. The generator picks a sequence of operations over a small tree of Go and TypeScript files in `t.TempDir()`. The generator uses eight operations. Three change files: add, delete and rename. Five change declarations: add, remove, rename, add a call, and add a repeated declaration. After each operation, the test runs `Sync`. At the end, it builds a fresh index of the same tree in a second directory and calls `invariant.Equal`. `rapid` shrinks a failing sequence to a short one. The default check count runs on every push. A higher count (`-rapid.checks`) runs in the weekly workflow from D9.

### D7. One node per declaration, checked against tree-sitter

`TestEveryDeclarationHasOneNode` parses each fixture file with tree-sitter and counts the declaration nodes with a small table of node kinds per language. Go uses `function_declaration`, `method_declaration`, `type_spec`, and each name in a `var_spec` and a `const_spec` (the #214 case). TypeScript and JavaScript use `function_declaration`, `class_declaration`, `method_definition` and `interface_declaration`. It then compares the count with the extracted nodes of those kinds, and asserts that no two extracted nodes share an id. It runs over `internal/indexer/testdata/` and a new fixture with the #198 shape. The table covers Go and TypeScript first, because seven of the twelve bugs were in those languages. If a bug or a fixture calls for another language, that language gets a row.

If `CODEGRAPH_CORPUS_DIR` is set, the same test also runs over the four pinned corpora. `corpora.yml` sets it and already fetches them.

### D8. The watch set equals the directory set, as a stateful `rapid` property

`TestWatchSetEqualsDirectorySet` in `internal/watch` generates directory creates, renames and deletes under `t.TempDir()`. After each step, it waits for the watcher to report the step. Then it compares the watched set with a `filepath.WalkDir` of the tree, without excluded directories. It runs on Linux (inotify) and on macOS (FSEvents) through the existing `daemon-darwin.yml`.

### D9. `keyenc` gets a native fuzz target, and a weekly workflow runs the fuzzers

`FuzzSegmentRoundTrip` in `internal/graphstore` checks that `decodeSegment(appendSegment(nil, s))` gives `s` back and consumes the whole buffer. `FuzzSegmentOrder` checks that for two inputs, the byte order of their encoded keys equals the order of the inputs. `go test` runs the seed corpus as unit tests on every push. A new workflow, `fuzz.yml`, runs on a weekly cron and on dispatch. It runs each fuzz target for five minutes and the `rapid` properties with a high check count. If it finds a failure, the job fails and uploads the failing input.

## Risks / Trade-offs

- If a copy is added or removed, the ids of the later copies change. A sync then deletes and recreates those nodes and their edges. Repeated declarations are rare outside minified bundles, so the cost is small.
- References to a repeated name resolve to the first copy. This is a known gap, recorded in the proposal's scope.
- The first sync after an upgrade rebuilds the whole index. On a large repository this takes as long as `codegraph index`. The release notes say so.
- The tree-sitter kind table in D7 is a second description of what a declaration is. If it disagrees with an extractor on purpose, the test needs an exception row with the reason. The test lists every exception in its failure message, so an exception cannot hide.
- `rapid` is a new dependency. `go mod tidy` fails on `main` (memory `ws0xjepcy7`), so it goes in through `go get pgregory.net/rapid@<version>` and a hand edit of `go.mod`.
