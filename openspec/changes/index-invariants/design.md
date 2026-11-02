# Design

## Context

`nodeid.NodeID(kind, qualifiedName, filePath)` (`internal/indexer/nodeid/nodeid.go`) hashes three length-prefixed segments. Each extractor computes the id once per declaration and then passes it to the calls that emit the declaration's edges. For example, `tsextract.emitFunction` computes `id` and then calls `collectCalls(id, body)` and `collectReferencesAndInstantiates(id, body)`. So an id that the extractor chooses at declaration time also reaches every edge that starts in that declaration's body.

Other files reach a declaration through the resolver (RES-01). The resolver does not recompute ids. It looks a name up in a symbol index that holds the extracted ids (`internal/indexer/symbolindex.go`). If two symbols share a name, it keeps the lexically lowest id. A reference names a name, not a declaration. A call records its name, an import alias, a kind and a position. It records nothing about its arguments. A method node has `Signature` and `ReturnType` as display strings, and `Arity` is filled only by the Go extractor.

A call through a receiver, `logger.log(x)`, goes through `tsextract.memberAccessAlias`, `csharpextract.isLikelyTypeName` and `goextract`'s WR-02 rule. Each one reads the receiver's first letter. An uppercase receiver is tried as a type, a lowercase one is forced unresolved as a local variable. So a call through a variable never resolves in Java, C#, Kotlin, Swift or TypeScript.

`writeGraph` (`internal/indexer/resolve.go`) sets `Meta.NodeCount = len(allNodes)`, repeats included. `Sync` (`internal/indexer/sync.go`) counts `nodesAdded` once for each entry in `allNodes`, so two new nodes with one id count twice. `PutNode` writes an id once, so the store holds fewer nodes than the count.

If a `Meta` flag is false, `needsFileIndexBackfill` forces a full rebuild. #217 added `has_grouped_vars` (field 14 of `Meta` in `internal/schema/graph.proto`) the same way.

The key encoding is `appendSegment` and `decodeSegment` in `internal/graphstore/keys.go`, with example tests in `keyenc_test.go`.

The in-repository fixtures are under `internal/indexer/testdata/`. The four pinned corpora (hugo, guava, requests, serilog) are fetched only in `corpora.yml`, which runs `task test:golden`. `testdata/golden/corpus/` holds their golden outputs, not their sources. Indexed with unique ids but without overload resolution, guava gained 3,897 nodes and serilog 375. `callers` on a bare name went empty for 189 of 3,468 guava names and for 11 of 107 serilog names.

## Goals / Non-Goals

**Goals:**

- Each declaration gets its own node, and a declaration's id depends on that declaration alone.
- A reference resolves to the declaration the language selects, or to the set of declarations it cannot choose between. It never resolves to a guess.
- One resolution pipeline for every language. The per-language parts are the syntax walkers and a type compatibility table.
- The stored node count is always the number of stored nodes.
- Properties and a fuzz target cover the invariant classes behind #187, #188, #193, #194, #197, #198, #208 and #214.
- Every new test has a positive assertion: if it inspects nothing, it fails.

**Non-Goals:**

- A type checker, generic inference, flow analysis or inherited fields.
- References by path across languages. The model leaves room for them (D13).
- New qualified names for anonymous scopes.
- A mutation tool (#222) or a BDD runner.

## Decisions

### D1. Ids are keyed by the signature, and final only after the whole file is extracted

The extractors do not visit declarations in source order. For example, `tsextract` collects types, then methods, then functions, then exported const functions. So an ordinal cannot come from the order of the calls.

A function or method id is the hash of four segments: kind, qualified name, file path and signature key. The signature key is the ordered list of parameter types as written. An untyped parameter contributes `_`. An optional parameter or a parameter with a default gets a `?` suffix, and a variadic one gets `...`. Parameter names are left out, because no language makes them part of a function's identity. Python `def f(a, b)` keys as `(_,_)`, Go `func f(a, b int)` as `(int,int)`, Java `foo(int, String...)` as `(int,String...)`, and a shell function as `()`. The key is a pure function of the declaration's own text, so adding or removing an overload changes no other id. A type, constant or variable has an empty key and keeps today's id. Ids that name another file's symbol, file nodes and package nodes keep calling `NodeID`.

Add `nodeid.Allocator`, created once per extracted file, with two steps:

1. `Declare(kind, qualifiedName, signatureKey string, startByte uint) string` returns a provisional id that is unique for each declaration occurrence. It is the final hash plus `@` plus the start byte. The extractor uses this id for the node and for every edge that the declaration's body emits.
2. `Finalize(result)` runs once at the end of the file. For each (kind, qualified name, signature key), it sorts the occurrences by start byte. The first gets the hash of the four segments. The n-th one (n of 2 or more) gets the hash of five segments: the four plus the decimal ordinal `n`. Then it rewrites each provisional id to its final id in every field of the file's `FileResult` that holds a node id: `Node.Id`, the `Source` and `Target` of each `IntraEdge`, `UnresolvedRef.FromID`, and the keys of `InterfaceMethods` and `MethodArity`.

A provisional id never reaches the store. After the rewrite, `Finalize` scans the same fields. If a provisional id is left in any of them, it fails the extraction.

Alternatives considered:

- An ordinal for every repeat, with no signature key. It depends on sibling order: inserting `foo(long)` above `foo(int)` renumbers `foo(int)`, and a sync deletes and recreates its node and every edge into it. It also violates the requirement that an id depends only on the declaration.
- A signature key only in languages with overloading. It splits the resolver by language for no gain. A signature edit in Go then changes the id and the sync reparses the callers, which is what a fresh index does and is correct.
- The start line in the hash. Then every edit above a declaration changes its id.
- A name from the enclosing call, for example the `it(...)` description. Then the id depends on call arguments, and minified code has no such names.

### D2. Same-file parent lookups use the occurrence, not the name

Some extractors find a declaration's parent in the same file through a map from name to id. For example, `tsextract` stores `typeNodesByName[name] = id` and reads it for each class body. With two classes of one name, the methods of both attach to the last class. The Python, Java, C#, PHP and Ruby extractors have the same pattern.

Each such map changes its key from the name to the declaration's start byte. A name-keyed lookup stays only where the language merges same-named parts on purpose: the C# partial type, which keeps one node for all its parts (`csharpextract/types.go`), and the reopened Ruby class or module (D16).

### D3. The resolver works on candidate sets and selects as the language does

The symbol index keeps every candidate for an owner and a name, with its kind, qualified name, file and structured signature. A candidate set is keyed by owner and name, not by language. Resolution of a call runs in three steps, as a pure function of the set and the call's recorded arguments:

1. Arity. If its parameter count fits the argument count, a candidate stays. The count is exact for fixed parameters. With optional, default or variadic parameters, the count is a range. A spread argument makes the argument count a lower bound.
2. Argument types. If several candidates stay, each argument with a known type keeps the candidates whose parameter type at that position is compatible. A per-language table defines compatibility: exact text first, then boxing and widening for Java, C# and Kotlin. An unknown type, `null`, `nil` or a lambda matches any parameter. If no candidate survives this step, the arity survivors stay, because a misread type must not drop a caller.
3. Outcome. One candidate gets the edge. Several candidates mean the call targets the set node (D14) with `ambiguous` in the edge metadata. No candidate means the reference stays unresolved and counts as today.

A reference that is not a call, such as `embeds`, `references` or `type_of`, names a type. Its set holds one candidate per declared type, so the steps reduce to a lookup. The full index and the store-seeded index of `Sync` build through the same code, so both get the same rules. Insertion order cannot change a result.

### D4. Counts come from distinct ids

`writeGraph` sets `NodeCount` to the number of distinct ids in `allNodes`. `Sync` counts a new id once. With D1 there are no repeats, but the count does not depend on that. `EdgeCount` does not drift (the #188 replay showed this), so it does not change.

### D5. A new `Meta` flag forces one rebuild, and a rebuild starts from an empty graph

Add `bool has_unique_decl_ids = 15;` to `Meta`. Three places write `Meta` today: `writeGraph` and the two incremental writers in `Sync`. Each builds a fresh `schema.NewMeta()` and sets each capability flag by hand. A flag set in only one place drops on the next incremental sync. So a new function, `stampCapabilities(meta)`, sets every capability flag, and all three writers call it. If the flag is false, `needsFileIndexBackfill` returns true. `SchemaVersion` stays 1, because the field is additive, as with `has_grouped_vars`.

The backfill path calls `run` over the existing store. `writeGraph` then puts the new graph but does not remove the old one. Only `codegraph index` clears the store first, by removing its directory. So old merged edges and the records of files deleted while no watcher ran survive a backfill. A new writer method, `DeleteGraph`, range-deletes the node, edge, file and file-index namespaces. `writeGraph` calls it first, in the same batch, so the delete and the new graph commit together. A reader on an older snapshot sees the old graph or the new one, never a mix.

### D6. One invariant checker, used by every test

Add a test helper package, `internal/indexer/invariant`, with `Check(store) (Counts, error)`. It asserts four things. Each edge endpoint is a stored node. `Meta.NodeCount` equals a scan of the store. Each `contains` edge that stands for lexical nesting has the child's line range inside the parent's range, which catches the wrong parent from D2. Lexical nesting means that the parent and the child are in the same file. The parent is a class, interface, struct, enum or overload set. The languages are TypeScript, JavaScript, Java, Python, C#, PHP and Ruby. Two kinds of `contains` edge are ownership, not nesting, and the rule leaves them out. A Go method belongs to its receiver type, but it is declared outside the type body. A merged C# partial type has a sentinel path, so the same-file condition already leaves it out. It also returns the counts it scanned, and each caller asserts that they are above zero. A second helper, `Equal(a, b)`, compares two stores node by node and edge by edge, and reports the first difference. The helpers get no tests of their own. Each failing test in tasks 2.2 to 2.5 proves them on `main`.

### D7. Sync equals a fresh index, as a model-based `rapid` property

`TestSyncEqualsFreshIndex` uses `pgregory.net/rapid`. The generator picks a sequence of operations over a small tree of Go, TypeScript and Java files in `t.TempDir()`. The generator uses eleven operations. Three change files: add, delete and rename. Eight change declarations: add, remove, rename and add a call. The other four add a repeated declaration, a repeated class with a method, an overload with a typed call to it, and a call through a local typed from a call into another Java file. After each operation, the test runs `Sync` and then `invariant.Check`. So an invalid graph after one step cannot hide behind a later step that repairs it. At the end, it builds a fresh index of the same tree in a second directory and calls `invariant.Equal`. The comparison leaves out route nodes and their edges, because route detection runs only in a full index. That is a documented limitation (`docs/LANGUAGE-CAPABILITY-MATRIX.md`), and this change does not alter it. `rapid` shrinks a failing sequence to a short one. The default check count runs on every push. A higher count (`-rapid.checks`) runs in the weekly workflow from D10.

### D8. One node per declaration, checked against tree-sitter

`TestEveryDeclarationHasOneNode` parses each fixture file with tree-sitter and lists the declaration occurrences with a small table of node kinds per language. It compares occurrences, not totals, because one missing declaration and one extra node can give equal totals. Each expected occurrence must match exactly one extracted node. The key is the name, the kind, and the start line and start column, which every stored node has. The line alone is not enough: minified code and one-line callbacks put two same-named declarations on one line. Each extracted node of a listed kind must match one expected occurrence. An `overload_set` node is not a declaration and is left out. It also asserts that no two extracted nodes share an id. The table has rows for the languages of the fixtures and of the four pinned corpora:

| language | declaration node kinds |
|---|---|
| Go | `function_declaration`, `method_declaration`, `type_spec`, each name in a `var_spec` and a `const_spec` (the #214 case) |
| TypeScript, JavaScript | `function_declaration`, `class_declaration`, `method_definition` and `interface_declaration`. Also an `arrow_function` or `function_expression` in an exported `const`, which `tsextract` emits as a function |
| Java | `class_declaration`, `interface_declaration`, `enum_declaration`, `method_declaration`, `constructor_declaration` |
| Python | `function_definition`, `class_definition` |
| C# | `class_declaration`, `struct_declaration`, `interface_declaration`, `record_declaration`, `method_declaration`. The parts of one partial type count as one declaration |

It runs over `internal/indexer/testdata/` and a new fixture with the #198 shape. If `CODEGRAPH_CORPUS_DIR` is set, the same test also runs over hugo (Go), guava (Java), requests (Python) and serilog (C#). `corpora.yml` sets it and already fetches them. If a row disagrees with an extractor on purpose, the row gets a named exception. The test logs every exception on each run, not only on a failure.

### D9. Watcher coverage after any sequence of directory changes, as a stateful `rapid` property

`TestWatcherCoversEveryDirectory` in `internal/watch` generates directory creates, renames and deletes under `t.TempDir()`. The two backends need different checks:

- Linux (inotify): after each step, the test compares the watcher's watch list with a `filepath.WalkDir` of the tree, without excluded directories.
- macOS (FSEvents): one recursive stream watches the root, so there is no per-directory set. After each step, the test writes one file in each directory that the step created or renamed. Then it asserts that the watcher delivers an event for each file.

The macOS leg runs through the existing `daemon-darwin.yml`.

### D10. Key segments get native fuzz targets, and a weekly workflow runs the fuzzers

`appendSegment` puts a varint length before the bytes. So the byte order of encoded keys is not the order of the strings: `b` encodes before `aa`. The fuzz targets check the properties the encoding has:

- `FuzzSegmentRoundTrip`: `decodeSegment(appendSegment(nil, s))` gives `s` back and consumes the whole buffer.
- `FuzzSegmentBoundaries`: two different lists of segments never give the same concatenated encoding. This is the T-02-01 property: a crafted segment cannot move a boundary.

`go test` runs the seed corpus as unit tests on every push. A new workflow, `fuzz.yml`, runs on a weekly cron and on dispatch. It runs each fuzz target for five minutes and the `rapid` properties with a high check count. If it finds a failure, the job fails and uploads the failing input.

### D11. A sync reparses the files whose candidate set changes

The 2,000-case run of `TestSyncEqualsFreshIndex` found a gap that `main` also has. A sync can add a candidate to a set that already resolves, and the outcome of D3 for an existing call can change. A fresh index then resolves the existing references with the new set. But `Sync` reparses only changed files, their dependents and the files waiting on an unresolved name, so the old edges stay.

A stored edge does not record the name it was resolved from. So `Sync` finds each (module key, name) whose candidates it adds to or removes from. For each one, it collects the unchanged files that own an edge into any candidate of that set, or into the set node. It reparses them the same way as a file waiting on a new name. A `contains` edge is resolved by its source's name, because a method names its receiver type, so for that kind the source counts. A same-file `contains` edge is made by the extractor itself and is left out. When no set changes, the step scans nothing. When a set changes, it scans the nodes once and the edges of the changed sets.

### D12. The inotify watcher drops the watches of a moved directory

`TestWatcherCoversEveryDirectory` found a #208-class bug on Linux. After a watched directory moves, the watch loop re-adds it on the `Create` event of the new path. inotify returns the watch that the moved directory still has, and fsnotify then handles the directory's `IN_MOVE_SELF` event by removing that watch. The moved directory goes unwatched, and its subdirectories stay in the watch list under their old paths.

fsnotify reports the `Rename` of the old path before the `Create` of the new one. On the `Rename` event, the loop now removes the watches of the old path and of every watched path below it. The re-add then starts from fresh watches. fsnotify's recursive mode handles moves itself, but it is enabled only in fsnotify's own tests.

### D13. Signatures and arguments are data in the schema

`Node` gains `repeated Param params` and `return_type_name`, the type name without decorations. `Param` has `name`, `type` as written, `optional`, `has_default` and `variadic`. `Signature` and `ReturnType` stay as display strings. `Arity` stays and equals `len(params)`, so the existing dispatch code keeps working.

`UnresolvedRef` and `Edge` gain the call's data: `receiver_type`, `arg_count`, `has_spread`, `repeated string arg_types` with `""` for an unknown type, and `repeated string arg_labels` for Swift. A receiver or argument type can be deferred. `UnresolvedRef` then records the position of the call whose return type it is. Pass 2 replaces it with the resolved return type or `""`. The stored edge holds only concrete types. `UnresolvedRef` also gains `addressing`: `name` for every reference today, with `path` reserved for a later change that resolves `subprocess.run(["./deploy.sh"])` to a file node in another language. The resolver pipeline branches on `addressing` once, so a path reference needs a lookup, not a second pipeline. Every field is additive, and `SchemaVersion` stays 1. The edge carries the argument data for two reasons. `callers --json` can show how a call was decided, and the set's ambiguous edges list what was known.

### D14. Overloads get a set node

Two or more overloads of one name under one owner get an `overload_set` node. A name with one declaration gets none, so the node count does not double. The set's id is `NodeID("overload_set", name, ownerID)`, where the owner is the type for a method and the file for a free function. So the parts of a C# partial type get one set across files. Its `Name` and `QualifiedName` are those of the overloads. Its location spans the first to the last overload in the file. When the overloads span files, as in a partial type, it takes the partial type's sentinel location. The owner contains the set, and the set contains each overload. A method with one declaration stays directly under its type.

The set is the target of an ambiguous call (D3). `callers`, `callees` and `impact` treat a bare name that is a set as the set (`graph-queries` spec).

Alternatives considered:

- Links between the overloads. A union query then traverses N nodes, and an ambiguous call has no honest target.
- An edge to every remaining candidate for an ambiguous call. It asserts N calls where one happened, and `callers` on each overload over-reports.
- A set node for every method. It doubles the method count and changes every golden for no gain.

### D15. A same-file reference resolves by lexical scope, and only reachable names cross files

A function declared inside a callback or another function is visible only in its enclosing scope. The extractor keeps, for each declaration, the byte range of its enclosing scope. A same-file call resolves to the innermost declaration of its name whose scope contains the call. Failing that, it resolves to a module-level declaration in the file, and then through the cross-file index. The cross-file index holds only what the language exports or declares at module level: exported declarations in TypeScript and JavaScript, module-level declarations in Python and Ruby, package-level declarations in Go, Java and Kotlin, and namespace-level declarations in C#. A nested or unexported declaration never enters it.

### D16. Each kind of repeat follows its language

| language | rule for two declarations with one name and owner |
|---|---|
| Java, C#, Kotlin, C++, Swift | overloads, resolved by D3 |
| Python, Ruby, JavaScript | at module level the last definition wins, as at runtime |
| C, C++, C# under `#if` or `#ifdef`, PHP under a `function_exists` guard | alternatives: a call targets their set as ambiguous |
| Ruby | a reopened class or module is one node, like a C# partial type |
| Swift | the qualified name includes the argument labels, so `foo(x:)` and `foo(y:)` are different functions |
| Go | a repeated `init` is never callable, and any other repeat is an error, so each gets an ordinal id |
| shell | a redefinition is last-wins, with an empty signature key |

The extractors mark a declaration under a conditional-compilation block, and the resolver treats the copies under different blocks as alternatives.

### D17. A receiver resolves through a type environment, not through its name

Each extractor builds, for each function body, a type environment: the declared types of the parameters, of the locals, of the fields of the enclosing type read from the type body, and `this`. When the syntax shows the type of its initializer, a local without a declared type takes that type. The initializer is a literal, a `new T(...)` or a cast. When the initializer is a call, the extractor cannot know its return type, because pass 1 reads one file at a time. So it records the local's type as deferred: the return type of the call at that position. Everything else is unknown.

When the environment knows the receiver's static type, a call `recv.m(args)` records it. A receiver that is a call, as in `make().add(s)`, records a deferred type the same way. Pass 2 first resolves the calls with no deferred type. Then it fills each deferred type from a call that resolved to one declaration with a recorded return type, and resolves the calls that waited. A deferred type that depends on a call still waiting stays unknown. That is the one level of propagation. The resolver then looks `m` up in the candidate set of the receiver's type, found through the file's imports or its own module as today. A receiver that names a type or a module directly resolves as today. A receiver whose type is unknown stays unresolved, as today. The same environment supplies the argument types of D3. A receiver of an interface type resolves to the interface's method, and the existing `implements` and `overrides` edges lead on from there.

### D18. No casing heuristics

`tsextract.memberAccessAlias`, `csharpextract.isLikelyTypeName` and `goextract`'s WR-02 rule read the receiver's first letter to decide what it is. The type environment of D17 decides instead, in the language's lookup order: a local, a parameter, a field, a type declared or imported in the file, a module or package, or unknown. Capitalization is never read. A Go package named in lowercase and a Python module then resolve like any other receiver. Code that does not follow a naming convention resolves the same way as code that does.

## Risks / Trade-offs

- Every function and method id changes once, because the signature key enters the hash. → The `has_unique_decl_ids` flag forces one rebuild, which this change needs anyway.
- The ids of identical-key repeats still depend on source order. → Identical keys occur only for redefinitions and alternatives, which are rare outside tests and conditional compilation. Overloads, which are common in Java and C#, never share a key.
- The first sync after an upgrade rebuilds the whole index. → The release notes say so.
- Argument and receiver types are syntactic, so some calls to overloaded names resolve to the set, not to one overload. → An ambiguous edge is visible and never wrong. The limits are listed in the proposal.
- `explore` lists each overload, so its output grows on overload-heavy code. → See the open question.
- The tree-sitter kind table in D8 is a second description of what a declaration is. → Each disagreement needs a named exception, and the test lists every exception on every run.
- The range delete in D5 makes one commit larger on a large repository. → It runs only on a full rebuild, which writes the whole graph in the same commit anyway.
- `rapid` is a new dependency, and `go mod tidy` fails on `main`. → It goes in through `go get pgregory.net/rapid@<version>` and a hand edit of `go.mod`.

## Migration Plan

A store from before this change lacks `has_unique_decl_ids`, so its first `codegraph sync` rebuilds it from an empty graph. No other step is needed. Every schema field is additive, so a rollback to the previous version reads the new store. It rebuilds the store the same way on its first sync.

## Open Questions

- Does `explore` fold an overload set into one entry by default and expand it on request? This touches only the renderer. The review of the golden outputs is the time to decide it.
