# Design

## Context

See proposal.md for the problem. The current state that shapes the approach:

- The C and C++ `ModuleKey` (`internal/indexer/extract/languages_c.go`, `languages_cpp.go`) is the file path. `resolveCall` (`internal/indexer/resolve/resolver.go`) looks up a bare call with `CandidatesIn(r.ImportPath, name)`, so it sees only the calling file.
- `cextract.emitInclude` emits each `#include` as a `RefKindImports` ref with the trimmed path. With an empty module path, `resolve/types.go` drops a relative include. An absolute include such as `/usr/include/foo.h` passes `isIntraModule`, so today it makes a synthetic `package` node and an `imports` edge.
- `resolve.Calls` reads only the file result, the symbol view and the hierarchy. The full index builds the view from `symbols.Declarations` at barrier 1, before any edge exists. A sync builds the view in `storeview.go` from stored nodes, and from the stored edges that `Builder.AddStoredEdge` keeps.
- `cextract.collectFunctions` drops a prototype whose name and signature key match a definition in the same file (#250). Other prototypes get an ordinary function node. The extractor never reads `static`: every free function is `public` and exported.
- The C signature key in the extractor, `sigKey`, appends `ConditionalBlockKey`. `goextract.SignatureKeyOfNode` rebuilds a key from the stored `Params`, which has no conditional part. Both keys mark a parameter with a default as optional, and both keep the spacing around `*`.
- `resolve/types.go` turns an imports ref into an edge to a synthetic `package` node, with the line and column of the ref. The file graph query skips package nodes.
- Sync finds the files to re-resolve in these ways:
  - The dependents of a pruned node, through the cached reverse adjacency `reverse()` in `sync.go`.
  - The files waiting on a new name, through `newNames` and `filesWaitingOn`, with `resolve.AnyModule` for a name in any module.
  - The owners of edges into a changed candidate set, through `filesResolvingToChangedSets`.
  - The emitters of shared ids, `bindingDependents`, `ownerDependents` and `containedMethodFiles`. The wave loop recomputes `bindingDependents` and the emitters for the files that each wave adds.
  - A missing `Meta` flag, which forces a full rebuild (`needsFileIndexBackfill`). `stampCapabilities` sets the flags for every writer of `Meta`.

## Goals / Non-Goals

**Goals:**

- One include graph for each resolve pass, which the full index and a sync fill from the data they already have.
- One resolution rule for a full index and a sync, so the equivalence tests compare one rule with itself.

**Non-Goals:**

- No preprocessor. Macros, `#ifdef` around an `#include`, and include directories from build files stay out (proposal.md, Out of scope).
- No change to how other languages use `Imports`.
- No new output field. The prototype marker is stored and read by the resolver only.

## Decisions

### D1. The include graph lives in the symbol view

One predicate, `symbols.LinksIncludes(lang)` next to `symbols.LastWins` in `internal/indexer/symbols/hierarchy.go`, is true for C and C++. `Freeze`, `DeclarationsOf`, the `resolve.Types` branch, `resolveCall`, `newNames` and `filesResolvingToChangedSets` call it, as the resolver and the sync both call `resolve.InheritsMembers` today. In this design, "a C or C++ file" means a file for which it is true.

`symbols.Declarations` gets `Includes`, the quoted include paths of a C or C++ file as written. The extractor marks a quoted include, whose path node is a `string_literal`, on its `RefKindImports` ref. `DeclarationsOf` keeps only marked refs, so an angle-bracket include and a macro include (`#include HEADER`) give no path.

`Builder.Freeze` resolves each written path against the C and C++ file paths that the builder holds. In a full index those are the extracted files. In a sync they are the stored C and C++ file nodes and the reparsed files, so a header with no symbol node still counts. In both runs a result with `Err` set is not held, because `cextract` writes no file node for it. A candidate equal to the including file matches nothing, so a file never links to itself. Two directives that match the same file give one stored `imports` edge. The C branch emits one edge for each matched directive, and `resolve.CollapseEdges` keeps one, as it does for every other repeated edge. If the near candidate is the including file, the lookup ends with no match, because the compiler opens the includer itself. So `#include "a.h"` in `src/a.h` never links to a root `a.h`. A new `symbols.IncludeCandidates(file, p)` returns the candidates of one written path in order. A written path that starts with `/` has none. After its leading `./` and `../` segments, a written path whose other segments include `.`, `..` or an empty segment also has none. A compiler opens `add.h/`, `add.h/.` or `nodir/../add.h` only when the directories in it exist, and the index holds no directories. A held file under a directory would show that the directory exists, but a rule built on that adds a second lookup for a rare form. Such includes stay unlinked. Otherwise the candidates are `path.Join(dir(file), p)` first, then `path.Clean(p)`. A candidate for which `fs.ValidPath` is false drops out, as in `internal/uiserver/spa.go`, so a candidate outside the repository root matches nothing. So `"../include/add.h"` from `src/main.c` can match `include/add.h`, and `"../../x.h"` from there matches nothing. `Freeze` takes the first candidate that it holds. The waits of D2 use the same function. `View.IncludeTarget(file, p)` returns the match of one written path, and `View.IncludesOf(file)` returns all matches of a file in sorted order.

The full index fills the graph from the declarations of barrier 1. A sync fills it from two sources. Reparsed files give their declarations. Unchanged files give their stored `imports` edges between two C or C++ file nodes, which `storeview.go` passes to a new `Builder.AddStoredInclude`. One predicate, `resolve.IsIncludeEdge(e)` beside the C branch of `resolve.Types`, recognizes such an edge from the edge alone, in the shape of `resolve.IsBindingEdge`. The C branch writes `Metadata[goextract.MetaInclude] = "quoted"` on each include edge, as the binding pass writes `MetaBinding`. `IsIncludeEdge` is true for kind `imports` with that mark. So a later file-to-file `imports` edge of another language never counts as an include. `storeview.go` and `storeIndex.load` both call it, as both call `resolve.IsBindingEdge` today. A stored include is already resolved, so `Freeze` keeps it as it is and does not apply the path rule again. So both runs build the graph with one rule, before pass 2.

`AddStoredInclude` is separate from `Builder.AddStoredEdge`. `View.StoredEdges` feeds the hierarchy tables and the binding pass. They read only their own edge kinds, and an include is neither.

In the `RefKindImports` case of `resolve.Types`, a C or C++ ref takes a new branch and never the package-node branch. So an absolute include no longer makes a package node or an edge. If `IncludeTarget` gives a match, the branch emits one `imports` edge from the file node of `r` to the file node of the match. The edge keeps the line and column of the ref. That edge is the stored form of the include for the next sync. It targets the file node, not a package node, because a C file has no package. So a C include appears in the file graph of the graph UI (`Engine.FileGraph`), and an include of another language does not. A C user expects to see that link.

Alternative: build the graph from the `imports` edges of pass 2. The full index passes only `GlobalEdges` to later passes, and `GlobalEdges` drops `imports` edges. That path needs a new edge channel through every pass. The view is already what every pass reads.

Alternative: resolve the path in the extractor, as `rubyextract.resolveRequireRelative` and `tsextract.resolveModuleSpecifier` do. The root path applies only when the near path names no indexed file, and an extractor does not know the set of indexed files.

### D2. An include waits on the paths that can change it

An include records a wait under the reserved module key `resolve.IncludeModule` on each candidate path that can change its match. An unmatched include waits on both candidates. An include that matched the root candidate waits on the near candidate, because a file at the near path takes precedence. `newNames` adds `{IncludeModule, relPath}` for each C or C++ file node that is new in the sync. `IncludeModule` is the constant `"\x00include"`, next to `AnyModule` (`"*"`). Module keys come from file paths and declared names, and no path holds a NUL byte. A Go import string is unquoted into its module key, so `import "\x00include"` could equal it. Such an import names no package, and a meeting only re-resolves a file once more. So no `ModuleKey` of any language returns it, and an include key never meets a symbol wait of a module. `filesWaitingOn` matches an `AnyModule` wait by name alone, so a new include path such as `"add"` can meet a field wait on `add`. That only re-resolves the waiting file once more, with the same result. If a file then appears at one of those paths, `filesWaitingOn` finds the includer.

### D3. Breadth-first lookup through the include graph

After the same-file rules in `resolveCall` find nothing, a C or C++ bare call walks `IncludesOf` breadth-first from its file. The walk is a new depth-aware `bfs.Levels` in a new package `internal/bfs`, which calls its visitor once for each depth with each node at its least depth, once. It keeps the order of the caller's adjacency within a depth, and the include walk sorts its adjacency by path.

`bfs.Levels` replaces the two breadth-first walks over a `map[string][]string` adjacency, for the risk "A third copy of the visited-set walk" below. Each keeps its output:

- `WalkSupertypes` visits the levels in order and stops when its visitor returns true. A node first enters the queue at its least depth, so the flat order of the levels is the order that the walk gives today.
- `query.transitiveWalk` sorts each level and appends it. Each level is the set of nodes at that least depth, so its sorted form is the frontier that the walk sorts today.

`internal/bfs` imports only the standard library, so `symbols` and `query` can both import it, as `mcp` and `uiserver` share `internal/textutil`. A generic walk does not belong in `symbols`, whose exports are the `Builder`, `View` and `Hierarchy` of a resolve pass.

`query.Impact`, `query.Affected` and `expandBFS` keep their own loops. They start from several seed nodes and stop at a depth or size bound. `Impact` and `Affected` also walk the dispatch siblings of each node, through `dispatchSiblingIDs`. A visitor of `bfs.Levels` would need each of those rules as a parameter. Their output is a compatibility surface of the CLI and MCP tools, and this change does not touch it.

Each depth gives one candidate list: the `CandidatesIn` of each file at that depth, in the order of the walk. `choose` ends in `symbols.Pick`, which keeps the lowest id, so the order within a depth does not change the result. `choose` filters that list as it filters the candidates of one file. So the arity filter runs only among several callables, and a lone candidate wins whatever its arity. The first depth with a candidate of that name decides, even if the arity filter removes all of its candidates. In a C file, the walk first drops the `KindStruct` candidates of the name from each depth, the calling file at depth 0 included. So a struct tag in the calling file does not stop the walk either. For the calling file, the `want` predicate of the `sameFile` call in `resolveCall` leaves out `KindStruct` in a C file, so a tag never takes part in the scope choice. `resolveCall` applies the same drop to `CandidatesIn` and to each depth of the walk. This changes one result of today: a C call whose name the calling file declares only as a struct no longer targets the struct. In C these are `struct` tags, which the extractor emits as `KindStruct`, and C keeps tags in a namespace of their own. So a call never names a tag. In a C++ file a `KindStruct` candidate still counts, because a call of a class name constructs it. The call then waits and does not look deeper, because in C the nearest declaration of a name is the one the compiler sees. A visited set stops include cycles.

Alternative: gonum `graph/traverse.BreadthFirst`. `internal/query` already depends on gonum, through its `community` and `simple` packages. It needs `int64` node ids, so each walk needs a map from path to id and back.

Alternative: one flat list of all reachable headers. In a depth-first or flat search, a far header can shadow a near one. Its result also depends on the order of the includes.

When the walk finds nothing, the call keeps the wait that `choose` gives today. It needs no wait under `AnyModule`. Only a change to a file of the include graph can change the result, and D2 and D6 re-resolve the caller for each such change. A name that no included file declares, such as a libc function, then adds no file to the reparse of each sync.

### D4. A prototype marker on the node, and `static` as visibility

The node schema (`internal/schema/graph.proto`) gets `bool prototype = 21`. `cextract.emitFunction` sets it for a `declaration` with no body. On a free function, a `storage_class_specifier` of `static` sets `Visibility: "module"` and `IsExported: false`. That is the value `tsextract.visibilityFor` writes for a TypeScript top-level symbol that its file does not export. A C file is its own module, so `module` names the same scope. Rust writes `private` for a module-private item, but there the module is not the file, and `private` already means a class member out of reach in `resolve.inheritable`. A C++ static member function keeps its visibility, because there `static` means class scope, not internal linkage. A definition with no storage class after a `static` prototype in the same file has internal linkage in C. So `collectFunctions` records each earlier `static` declaration. In a C file it records the name alone, because C has no overloads, so `static int h();` before `int h(int x)` gives `h` internal linkage. In a C++ file it records the name and the `c_param_key`, without the conditional part, because `h(long)` and `h(int)` are two functions. The carry applies only to a free function, the identifier branch of `emitFreeFunctionLike`, so an out-of-line C++ method `S::f` never takes it. It gives `static` to each later declaration that matches, prototype or definition. This includes a prototype that keeps its own node because its `#if` block differs from the definition's. The resolver reads the stored fields `Prototype`, `IsExported` and `CParamKey`, so a sync's view sees the same facts as a full index.

Alternative: work out a prototype from the node span or from the `signature` text. Neither is reliable, because a one-line definition has the same span shape, and the signature text shows no body.

### D5. A view index of C and C++ definitions by name

`Builder.Freeze` fills `cDefinitionsByName` in the loop that fills `goFieldsByName`. It holds each C and C++ node of kind `KindFunction` with `!Prototype && IsExported`, in id order. A C++ method is `KindMethod`, so an in-class member and an out-of-line `S::f` stay out. `View.CDefinitionsNamed(name)` returns them.

When `choose` returns a prototype with `IsExported`, the resolver filters `CDefinitionsNamed(name)` to the definitions whose match key is the same, in a file other than the file of the prototype. Exactly one match gives the target. Zero or several keep the prototype. The redirect runs only when `choose` returns one prototype. It runs on the same-file result of `choose` as well as on a result from the include walk, so a prototype in the calling file also reaches its definition. A prototype and a definition of one name under different `#if` blocks of one file form an overload set (`overloadGroup`), so `choose` returns the set and the redirect does not run.

The match key is a new stored field, `string c_param_key = 22` on `Node`. When `cextract` emits a C or C++ function, `cextract.paramKey` builds the key from the parse tree of its `parameter_list`. For each parameter, it walks the parameter's subtree with `cextract.walkDescendants`, which visits children left to right, and joins the leaf tokens it keeps with single spaces. Its visitor returns false for a node it leaves out, so the walk skips that subtree. It leaves out a `comment` node, the `default_value` subtree with the `=` token before it, and the name of each declarator. The name is the identifier that the walk reaches from the parameter's `declarator` field. It follows the `declarator` field of a `pointer_declarator`, `array_declarator` or `function_declarator`. A `parenthesized_declarator` and a C++ `reference_declarator` have no such field, so it follows their one named declarator child. It never takes another identifier child, as the fallback of `innermostIdentifier` does, so the size `N` in `int[N]` stays. The names of the inner parameters of a function-pointer type are declarator names too. The grammar marks a qualifier as `type_qualifier` and an array size as the `size` field, so neither is a name and both stay. A list of `(void)` keys as empty, as `collectParams` reads it. The C++ grammar leaves the `...` of a variadic list unnamed, so `paramKey` reads every child of the list and keys that token as `...`, and `f(int, ...)` does not match `f(int)`. The parameter keys join with `,`. So `const char *s` and `const char* s` both key as `const char *`. `int a[10]` and `int[10]` key as `int [ 10 ]`, and `int[N]` and `int a[N]` as `int [ N ]`. `void (*cb)(int x)` and `void (*)(int)` key alike, and `void (*const cb)(int)` keeps its `const`. A comment between a type and its name drops out. A C++ prototype `f(int a, int b = 0)` matches its definition `f(int a, int b)`, which cannot repeat the default. The key has no conditional part, so a prototype under an include guard matches a definition outside any `#if`. A top-level qualifier of the parameter itself, as in `int f(const int x)`, stays in the key, as proposal.md Out of scope says. `collectParams`, the stored `Params` and `sigKey` keep their output, so no node id changes. The resolver reads `GetCParamKey()`. `cextract.collectFunctions` builds `definedKeys` from the same key plus the conditional part, and the lookup in `emitFreeFunctionLike` uses the same key. So the same-file rule of #250 and the cross-file match agree on which prototype declares which definition.

Alternative: merge the prototype node into the definition. This changes node ids, deletes the header declaration from `search` and `node`, and breaks the rule that each declaration has its own node.

Alternative: normalize the stored `Params` text in the resolver. Text needs its own rules for whitespace, comments and function-pointer names, and a hand-written list of the qualifier spellings that the grammar already marks. It also cannot tell the size `N` in `int[N]` from a name, because `collectParams` stores that size as the name.

Alternative: change `Params` or `sigKey` to the tree form. This changes the node id of every C and C++ function with a pointer parameter.

### D6. Sync dependents

A sync extends three mechanisms that it has, and adds no new scan:

1. `storeIndex` (`sync_storeindex.go`) gets `includers`, a map from each C or C++ file path to the paths of the files that include it, filled from `imports` edges in its edge scan. It is one more reverse adjacency of stored edges, next to the cached `reverse()` of `sync.go` and `query.BuildReverseAdjacency`. The design does not read it from `reverse()`, because that map would then hold the `imports` edges of every language, while `includers` holds only the C and C++ include edges from a scan that `storeIndex` already makes. Issue #274 tracks one shared builder for these maps, and `includers` joins its scope. A new `includeDependents` in `sync_includes.go`, next to `bindingDependents` and `ownerDependents`, walks `idx.includers` with `bfs.Levels` from each reparsed, added or deleted C or C++ file, as `ownerDependents` walks `idx.supers` with `WalkSupertypes`. So the closure needs no extra Extract wave for each include depth. It also runs in each wave on the files that the wave adds, next to `bindingDependents`, so a header that a later wave adds re-resolves its includers too. The `inBatch` set of the loop keeps a file from joining twice. `filesWaitingOn` and `filesResolvingToChangedSets` run once, before the wave loop, and their files join the first wave. So a header that they add also re-resolves its includers. A deleted or renamed header then re-resolves every file that includes it, directly or not.
2. `filesResolvingToChangedSets` keys a C or C++ function by `AnyModule` and its name, not by its file. So a prototype and the definitions of one name form one candidate set. If a sync adds or removes one of them, the owners of edges into any of them re-resolve. A reparsed function whose `IsExported`, `Prototype` or `CParamKey` differs from its stored node keeps its node id, so its id sets compare equal. `filesResolvingToChangedSets` therefore keeps a set of the `{AnyModule, name}` keys of such functions. A key in that set counts as changed and skips the `maps.Equal` check.
3. The waits of D2 and D3 go through `filesWaitingOn`.

A sync still reparses the callers of a changed file, as it does today. The nodes of a changed file are gone nodes, and the files that call them are dependents.

### D7. One rebuild

`Meta` gets `bool has_c_prototypes = 17`, with a comment in the shape of `has_data_symbols`. `stampCapabilities` (`internal/indexer/resolve.go`) sets it, so every writer of `Meta` keeps it. If the flag is false, `needsFileIndexBackfill` returns true. Its doc comment lists the flag.

## Risks / Trade-offs

- [The qualified name of a C++ free function has no namespace. So a prototype `a::f(int)` can match a definition `b::f(int)`] → The capability matrix keeps the namespace gap. If both namespaces define `f`, the several-definitions rule keeps the prototype as the target.
- [A third copy of the visited-set walk: the include walk would be a third breadth-first walk over a string adjacency, next to `WalkSupertypes` and `query.transitiveWalk`, and a fix to one copy can miss the others] → D3 builds both walks on `bfs.Levels`. Task 3.2 keeps their tests and the golden files unchanged, so their output does not change.
- [An edit to a header that most files include makes a sync reparse most files] → A fresh index gives the same edges, so a sync cannot do less without a finer dependency record. Task 3.6 also measures a sync after an edit to the most included header, and the PR records the number.
- [In a large project with deep include chains, each bare call walks many files] → The walk stops at the first depth with a candidate. The view holds the include graph once for each resolve pass. Task 3.6 measures the full-index time on a C repository before and after the change, and the PR records both numbers. `tools/bench/runner` measures index time, but the `tools/bench/realcorpus` manifest holds only Go repositories (`weft-go`, `cockroachdb-pebble`). A C entry there would change the published bench set, which is outside this change, so task 3.6 is a one-off timing. The sync timing after a header edit has no runner mode either.
- [A header under `#ifdef` links, and the build can leave it out] → Out of scope. Such a link can only add a candidate. Among several candidates, the arity filter and the several-definitions rule handle it. A lone extra candidate wins, as it does in one file.
- [The key leaves out declarator names and comments, and that can make two types match] → Every other token stays, in order, and C and C++ give no meaning to the spacing between tokens. So the key matches only spellings of one type. The table test and the property of task 3.0 pin this: `int *` and `int`, `char *` and `char **`, and `const char *` and `char *` keep unequal keys.

## Migration Plan

The first `codegraph sync` after the upgrade rebuilds the index (D7). The user does nothing. To roll back, the user installs the older binary and runs `codegraph index --force`. The older binary does not know the new flag, so it does not rebuild the store by itself.

## Open Questions

None.
