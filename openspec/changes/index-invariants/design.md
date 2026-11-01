# Design

## Context

`nodeid.NodeID(kind, qualifiedName, filePath)` (`internal/indexer/nodeid/nodeid.go`) hashes three length-prefixed segments. Each extractor computes the id once per declaration and then passes it to the calls that emit the declaration's edges. For example, `tsextract.emitFunction` computes `id` and then calls `collectCalls(id, body)` and `collectReferencesAndInstantiates(id, body)`. So an id that the extractor chooses at declaration time also reaches every edge that starts in that declaration's body.

Other files reach a declaration through the resolver (RES-01). The resolver does not recompute ids. It looks a name up in a symbol index that holds the extracted ids (`internal/indexer/symbolindex.go:67`). If two symbols share a name, it keeps the lexically lowest id (`:82-94`). A reference names a name, not a copy.

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

### D1. Ids are final only after the whole file is extracted

The extractors do not visit declarations in source order. For example, `tsextract` collects types, then methods, then functions, then exported const functions (`tsextract.go:93-97`). So an ordinal cannot come from the order of the calls.

Add `nodeid.Allocator`, created once per extracted file, with two steps:

1. `Declare(kind, qualifiedName string, startByte uint) string` returns a provisional id that is unique for each declaration occurrence. It is `NodeID(kind, qualifiedName, filePath)` plus `@` plus the start byte. The extractor uses this id for the node and for every edge that the declaration's body emits, as it does with today's id.
2. `Finalize(result)` runs once at the end of the file. For each `(kind, qualifiedName)` pair, it sorts the occurrences by start byte. The first one gets `NodeID(kind, qualifiedName, filePath)`, which is today's id. The n-th one (n of 2 or more) gets the hash of four segments: the three of today, plus the decimal ordinal `n`. Then it rewrites each provisional id to its final id in every field of the file's `FileResult` that holds a node id. The fields are `Node.Id`, the `Source` and `Target` of each `IntraEdge`, `UnresolvedRef.FromID`, and the keys of `InterfaceMethods` and `MethodArity` (`goextract/types.go:118-185`). The resolver reads both maps by node id (`resolve.go:94-104`), so a stale key stops dispatch from matching the node.

A provisional id never reaches the store. After the rewrite, `Finalize` scans the same fields. If a provisional id is left in any of them, it fails the extraction. Ids that name another file's symbol, file nodes and package nodes keep calling `NodeID`.

Alternatives considered:

- A counter in `Declare`, with no second step. It numbers the copies in visit order, not in source order.
- A pass after extraction that renumbers repeated ids. Both copies emit edges with the same source id, so such a pass cannot tell which edges belong to which copy. The provisional id removes that problem.
- The start line in the final hash. Then every edit above a declaration changes its id, and a sync deletes and recreates its node and edges.
- A name from the enclosing call, for example the `it(...)` description. Then the id depends on call arguments, and minified code has no such names.

### D2. Same-file parent lookups use the occurrence, not the name

Some extractors find a declaration's parent in the same file through a map from name to id. For example, `tsextract` stores `typeNodesByName[name] = id` (`tsextract.go:411`) and reads it for each class body (`:725`). With two classes of one name, the methods of both attach to the last class. The Python, Java, C#, PHP and Ruby extractors have the same pattern.

Each such map changes its key from the name to the declaration's start byte. Where a map does not fit, the lookup walks up the syntax tree to the enclosing declaration. A name-keyed lookup stays only where the language merges same-named parts on purpose. That is the C# partial type, which keeps one node for all its parts (`csharpextract/types.go:81-99`), as `docs/LANGUAGE-CAPABILITY-MATRIX.md` documents.

### D3. The resolver prefers the first copy

If two symbols share a name, `symbolIndex.addSymbol` (`internal/indexer/symbolindex.go:67-105`) keeps the lexically lowest id. With D1, the lowest hash can be any copy. A pairwise rule that prefers the first copy is not enough, because another declaration group can share the lookup name. With hashes in the order A2, B, A1, the insertion order A1, B, A2 gives A2, and the order A2, A1, B gives B.

So the symbol index keeps every candidate for a lookup name, with its kind, qualified name and file, and picks the winner as a pure function of that set, in two steps. First, within each group of candidates with the same kind, qualified name and file, it keeps the one whose id equals `NodeID(kind, qualifiedName, filePath)`, which is the first copy. If no candidate in a group has that id, the group keeps its lowest id. Second, across groups, it keeps today's rule: the lowest id. Insertion order cannot change the result. The full index and the store-seeded symbol index of `Sync` both build through the same code, so both get the rule.

### D4. Counts come from distinct ids

`writeGraph` sets `NodeCount` to the number of distinct ids in `allNodes`. `Sync` counts a new id once. With D1 there are no repeats, but the count does not depend on that. `EdgeCount` does not drift (the #188 replay showed this), so it does not change.

### D5. A new `Meta` flag forces one rebuild, and a rebuild starts from an empty graph

Add `bool has_unique_decl_ids = 15;` to `Meta`. Three places write `Meta` today: `writeGraph` and the two incremental writers in `Sync` (`sync.go:207` and `:531`). Each builds a fresh `schema.NewMeta()` and sets each capability flag by hand. A flag set in only one place drops on the next incremental sync. So a new function, `stampCapabilities(meta)`, sets every capability flag, and all three writers call it. If the flag is false, `needsFileIndexBackfill` returns true. `SchemaVersion` stays 1, because the field is additive, as with `has_grouped_vars`.

The backfill path calls `run` over the existing store (`sync.go:48`). `writeGraph` then puts the new graph but does not remove the old one. Only `codegraph index` clears the store first, by removing its directory (`internal/cli/index.go:141`). So old merged edges and the records of files deleted while no watcher ran survive a backfill. A new writer method, `DeleteGraph`, range-deletes the node, edge, file and file-index namespaces. `writeGraph` calls it first, in the same batch, so the delete and the new graph commit together. A reader on an older snapshot sees the old graph or the new one, never a mix. This also fixes every earlier backfill flag, not only the new one.

### D6. One invariant checker, used by every test

Add a test helper package, `internal/indexer/invariant`, with `Check(store) (Counts, error)`. It asserts four things. Each edge endpoint is a stored node. `Meta.NodeCount` equals a scan of the store. Each `contains` edge between two declarations has the child's line range inside the parent's range, which catches the wrong parent from D2. It also returns the counts it scanned, and each caller asserts that they are above zero. A second helper, `Equal(a, b)`, compares two stores node by node and edge by edge, and reports the first difference. The helpers get no tests of their own. Each failing test in tasks 2.2 to 2.5 proves them on `main`.

### D7. Sync equals a fresh index, as a model-based `rapid` property

`TestSyncEqualsFreshIndex` uses `pgregory.net/rapid`. The generator picks a sequence of operations over a small tree of Go and TypeScript files in `t.TempDir()`. The generator uses nine operations. Three change files: add, delete and rename. Six change declarations: add, remove, rename, add a call, add a repeated declaration, and add a repeated class with a method. After each operation, the test runs `Sync`. At the end, it builds a fresh index of the same tree in a second directory and calls `invariant.Equal`. `rapid` shrinks a failing sequence to a short one. The default check count runs on every push. A higher count (`-rapid.checks`) runs in the weekly workflow from D10.

### D8. One node per declaration, checked against tree-sitter

`TestEveryDeclarationHasOneNode` parses each fixture file with tree-sitter and counts the declaration nodes with a small table of node kinds per language. It then compares the count with the extracted nodes of those kinds, and asserts that no two extracted nodes share an id. The table has rows for the languages of the fixtures and of the four pinned corpora:

| language | declaration node kinds |
|---|---|
| Go | `function_declaration`, `method_declaration`, `type_spec`, each name in a `var_spec` and a `const_spec` (the #214 case) |
| TypeScript, JavaScript | `function_declaration`, `class_declaration`, `method_definition`, `interface_declaration` |
| Java | `class_declaration`, `interface_declaration`, `enum_declaration`, `method_declaration`, `constructor_declaration` |
| Python | `function_definition`, `class_definition` |
| C# | `class_declaration`, `struct_declaration`, `interface_declaration`, `record_declaration`, `method_declaration`. The parts of one partial type count as one declaration |

It runs over `internal/indexer/testdata/` and a new fixture with the #198 shape. If `CODEGRAPH_CORPUS_DIR` is set, the same test also runs over hugo (Go), guava (Java), requests (Python) and serilog (C#). `corpora.yml` sets it and already fetches them. If a row disagrees with an extractor on purpose, the row gets a named exception, and the test lists every exception in its failure message.

### D9. Watcher coverage after any sequence of directory changes, as a stateful `rapid` property

`TestWatcherCoversEveryDirectory` in `internal/watch` generates directory creates, renames and deletes under `t.TempDir()`. The two backends need different checks:

- Linux (inotify): after each step, the test compares the watcher's watch list with a `filepath.WalkDir` of the tree, without excluded directories.
- macOS (FSEvents): one recursive stream watches the root (`watcher_darwin.go:26-55`), so there is no per-directory set. After each step, the test writes one file in each directory that the step created or renamed. Then it asserts that the watcher delivers an event for each file.

The macOS leg runs through the existing `daemon-darwin.yml`.

### D10. Key segments get native fuzz targets, and a weekly workflow runs the fuzzers

`appendSegment` puts a varint length before the bytes (`internal/graphstore/keys.go:69-74`). So the byte order of encoded keys is not the order of the strings: `b` encodes before `aa`. The fuzz targets check the properties the encoding has:

- `FuzzSegmentRoundTrip`: `decodeSegment(appendSegment(nil, s))` gives `s` back and consumes the whole buffer.
- `FuzzSegmentBoundaries`: two different lists of segments never give the same concatenated encoding. This is the T-02-01 property: a crafted segment cannot move a boundary.

`go test` runs the seed corpus as unit tests on every push. A new workflow, `fuzz.yml`, runs on a weekly cron and on dispatch. It runs each fuzz target for five minutes and the `rapid` properties with a high check count. If it finds a failure, the job fails and uploads the failing input.

## Risks / Trade-offs

- If a copy is added or removed, the ids of the later copies change. A sync then deletes and recreates those nodes and their edges. Repeated declarations are rare outside minified bundles, so the cost is small.
- References to a repeated name resolve to the first copy (D3). This is a known gap, recorded in the proposal's scope.
- The first sync after an upgrade rebuilds the whole index. On a large repository this takes as long as `codegraph index`. The release notes say so.
- The tree-sitter kind table in D8 is a second description of what a declaration is. Its rows can disagree with an extractor on purpose. Each such case needs a named exception, and the test lists every exception in its failure message, so an exception cannot hide.
- The range delete in D5 makes one commit larger on a large repository. It runs only on a full rebuild, which writes the whole graph in the same commit anyway.
- `rapid` is a new dependency. `go mod tidy` fails on `main` (memory `ws0xjepcy7`), so it goes in through `go get pgregory.net/rapid@<version>` and a hand edit of `go.mod`.
