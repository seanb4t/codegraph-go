# Design: Split the indexer into layers with one job each

## Context

See proposal.md for the reason. This section records the facts in the code that shape the split.

- `resolveRefsWithIndex` in `resolve.go` is one function of about 350 lines over all files at once. It runs in four steps. First, every reference that is not a call becomes an edge: embeds, contains, imports, references, instantiates, returns and type-of. Second, `learnEdges` and `declareSets` read those edges to build the type hierarchy and the overload sets. Third, calls resolve: first the calls without a deferred type, then the deferred ones, sorted by file and end position. Fourth, whole-graph passes add `implements`, `overrides` and conformance-retry edges, and rewire member edges onto overload sets.
- The `resolver` struct carries two kinds of mutable state during call resolution. `callTargets` records where each call resolved, and `fileNodes` caches one file's declarations. `fillDeferred` always reads `callTargets` with the caller's own file path. So both are file-scoped in practice, although they live for the whole run.
- `collapseEdges` sorts the duplicates of one edge by file, line and column with `sort.Slice`, which is not stable. Two duplicates that tie on all three keys keep whichever came first in the input. So the order in which resolution emits edges is part of the stored graph.
- `Sync` reads the stored graph into a `storeView` and overlays the reparsed files on its symbol index. Then it calls the same `resolveRefsWithIndex` and stages its own writes. `writeGraph` and the tail of `Sync` each sort and stage nodes, files and edges, in two copies.
- `Extract` is a worker pool around a per-file step: read the file, pick the parser for its language, call `LanguageSpec.Extract`. The language packages are already synchronous and never open the store.
- `internal/depclosure` already holds architecture rules over the transitive import closure, from `go list -deps`. `.golangci.yml` holds the direct-import rules through `depguard`.

## Goals / Non-Goals

**Goals:**

- Each layer has one job and its own package. The compiler and one gate enforce the boundaries.
- Resolution of one file depends only on that file and on frozen, read-only views.
- The full index and sync share the resolution and staging code.
- The stored graph is byte-identical to `main` by construction, not only by test.

**Non-Goals:**

- Changes to memory use or speed. The pass order, the result slices and the single commit stay as they are.
- New public API. The new packages are internal, and each exports only what the layer above it calls.

## Decisions

The first three decisions are the ones most likely to change in review.

### D1. One `depclosure` test enforces the import direction, not `depguard`

The issue asks for a `depguard` rule. This design uses a `depclosure` test instead. The test lists each layer package and its closure with `go list -deps`. It makes two assertions:

- Positive: each layer package is in the list, and its closure contains the layer below it. If a package moves or a pattern stops matching, the test fails. It does not pass on nothing.
- Negative: no forbidden package is in the closure of a layer.

A `depguard` rule matches files by glob. If a glob stops matching, the rule passes in silence. It also sees only direct imports. So a layer that reaches `graphstore` through a helper package passes. "Each layer depends only on the layers below it" is a rule about the closure. This repository keeps closure rules in `depclosure`.

Alternative: a `depguard` rule per layer, as the issue says. It is cheaper to write but has the two gaps above. If review prefers it, the `depclosure` test is dropped. Both gates together give two places for one rule.

### D2. Resolution is a per-file pure function plus one whole-graph `Link` step

The issue describes resolution as a pure function of one file. Three of today's steps need every file: the synthesized `implements` and `overrides` edges, the conformance retry, and the overload-set rewiring. They cannot run per file without a change in output. The hierarchy barrier of D3 also falls between the type-level edges and the calls of one file. So `internal/indexer/resolve` has three entry points:

- `resolve.Types(view, result) TypeOutput`: the nodes and the type-level edges of one file, with its unresolved names.
- `resolve.Calls(view, hierarchy, result) CallOutput`: the call edges and the pending calls of one file. It keeps `callTargets` and the same-file declaration cache in local variables, so no state outlives the file.
- `resolve.Link(view, hierarchy, types, calls) Graph`: the whole-graph passes, the `File` records and the assembly of the final node, edge and file lists.

Both are pure. Neither opens the store or starts a goroutine. `Link` is the seam that #271 keeps as a barrier after the per-file work.

Alternative: keep the whole-graph passes inside orchestration. That puts resolution logic in the one layer meant to hold only scheduling, and it splits one algorithm across two packages.

### D3. Two frozen views, one per barrier

The type-level edges need the declarations of every file. The call edges also need the type hierarchy and the overload sets, which come from the type-level edges of every file. So there are two barriers, and each gets its own frozen type:

1. `symbols.Builder` collects declarations from pass 1 and, for a sync, from the stored graph. `Builder.Freeze()` returns a `symbols.View`, the read-only lookups of today's `symbolIndex`.
2. Resolution produces the type-level edges of every file against the `View`. `symbols.NewHierarchy(view, typeEdges)` returns a `symbols.Hierarchy`: the supertypes, the member owners and the declared overload sets. Its constructor does all the work of today's `learnEdges` and `declareSets`, and the type has no method that writes.

Neither frozen type has an exported field or a method that writes. A write after the barrier is a compile error, because the method to do it exists only on `Builder`.

Alternative: one `View` with a `Freeze` after the hierarchy. Then type-level resolution reads a type that still has write methods. The issue wants the compiler to reject exactly that.

### D4. Package layout

| Package | Job | Imports |
| --- | --- | --- |
| `internal/indexer/discover` | Walk the tree, apply exclusions, name each file's language and module key | `extract` (for the language registry) |
| `internal/indexer/extract` | The language registry and the per-file step, from bytes to `goextract.FileResult`. The file read stays in the worker pool. | the language packages, `parser` |
| `internal/indexer/symbols` | `Builder`, `View`, `Hierarchy` | `goextract`, `nodeid`, `schema` |
| `internal/indexer/resolve` | `File`, `Link`, edge collapse | `symbols`, `dispatch`, `nodeid`, `goextract`, `schema` |
| `internal/indexer` | `Run`, `Sync`, the worker pool, the store reads of a sync, route detection wiring, the `Sink` interface and the staging function | everything above, `graphstore`, `routes`, `capability` |

The language packages, `nodeid`, `dispatch`, `capability` and `routes` keep their paths. `discover` sits beside `extract` and not above it in the order of the run. It imports the registry only to know which extensions to keep.

### D5. The sink is the `graphstore.Writer` seen through a narrow interface

`internal/indexer` declares `Sink` with the methods that staging calls: the puts, the deletes, `Commit` and `Close`. `graphstore.Writer` already has all of them, so `graphstore` changes nothing and `Import` keeps using its `Writer`. One function, `stage(sink, graph, ownerPath)`, sorts and puts nodes, files and edges for both `Run` and `Sync`. The parts that differ stay with their callers. The full index deletes the old graph first and stamps a fresh `Meta`. Sync deletes pruned records and keeps synthesized edges that are already stored.

Alternative: a `Sink` with record-level methods such as `StageGraph(Graph)`, implemented in `graphstore`. That moves graph knowledge, such as edge owners and sort order, into the store. It also adds a second write path beside `Writer` that `Import` does not use.

### D6. `Link` assembles edges in today's order

`collapseEdges` breaks a tie by input order (see Context), so equality with `main` needs the same order. `CallOutput` keeps two lists apart: calls resolved without a deferred type, and deferred-call edges with their sort keys. `Link` concatenates the lists in this order:

1. The type-level edges of every file, in result order.
2. The calls without a deferred type, in result order.
3. The deferred-call edges, after the stable sort by file and end position that runs today.

The pending list for the conformance retry follows the same rule. So equality holds by construction for every input order. This includes a sync, whose result slice is not sorted.

Alternative: make `collapseEdges` stable and order-free with a full key. That changes which duplicate is kept in a tie, so it changes the stored graph. If that is wanted, it is a separate change.

### D7. The store reads of a sync stay in orchestration

`loadStoreView` and the invalidation planning (`filesWaitingOn`, `filesResolvingToChangedSets`, the reverse adjacency) read `graphstore.Reader`. They stay in `internal/indexer`. `loadStoreView` feeds a `symbols.Builder` through its add methods, in place of building a `symbolIndex` itself. The maps it copies today (kinds, names, start lines, arity, interface methods, stored sets, module of file) become inputs to the `Builder`. So the stored graph and pass 1 reach the views through one door.

### D8. Tests follow the code, and each layer gets the test that fits it

White-box tests move with the code they cover. Tests that drive `Run` or `Sync` stay in `internal/indexer` and keep their assertions as they are. The new tests are:

- `symbols`: a `rapid` property: two `Builder`s fed the same declarations in different orders freeze to views that answer every lookup the same.
- `resolve`: a `rapid` property: `Link` over the outputs of a corpus gives the same `Graph` for any permutation of the result slice.
- Orchestration: `goleak` in the `TestMain` of `internal/indexer`. Also a `testing/synctest` test: if one file fails, every worker of the extraction pool stops.

## Risks / Trade-offs

- [The diff is large and mostly moves, which hides small edits] → The work lands as one commit per layer, in the order of tasks.md. Each commit builds and passes `task test`. So a reviewer reads one layer at a time.
- [A moved test loses an unexported helper] → The test moves to the package of the helper. Or it drives the exported entry point. No helper is exported only for a test.
- [The equality argument of D6 misses an order dependence] → Before the PR, index three corpora on `main` and on the branch: the synthetic corpus, this repository and Pebble. The node and edge hashes must be equal. The PR records the counts and hashes, as #268 did. The goldens, `TestDeterministicRebuild` and `TestSyncEqualsReindex` also run on every commit.
- [`go list` adds seconds to `task test`] → The test joins the `depclosure` package, which already pays that cost once per run.

## Migration Plan

No data or format migration. A store from `main` stays valid, and the binary output does not change. To roll back, revert the PR.

## Open Questions

- Where the Archify diagram lives in `docs/`. The task names `docs/architecture/indexer-layers.html`. Review can move it without a change to the code.
