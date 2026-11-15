# Design

## Context

`Engine.NodeSelected` builds a `DefinitionDetail` (`internal/query/detail.go`) for one definition, or a `MultiDefDetail` that builds one per candidate on demand. `RenderNode` and `renderNodeSection` in `internal/query/render_markdown.go` turn it into the markdown that the CLI prints and `codegraph_node` returns. `writeSingleDef` and `writeNodeSection` in `internal/cli/present/node.go` print the same lines with styles. `TestRenderNodeStrippedEqualsMarkdownContract` checks that the two agree. `nodeDefinitionToProto` and `nodeDetailToProto` in `internal/uiserver/handlers.go` map a detail onto `ui.proto`. In the web app, `loadBrowseTarget` in `web/src/lib/browse-state.ts` keeps `calls` and `calledBy` of the response in `BrowseTargetState`. `web/src/routes/browse/+page.svelte` passes them to `NeighborsPanel.svelte`.

The stored facts:

- A read or a write is a `references` edge from the reader or writer to the data node. Its `access` metadata (`goextract.MetaAccess`) is `read`, `write` or `read_write`.
- The store keys an edge by source, kind and target, so it holds at most one `references` edge per source and target. `resolve.withMergedAccess` merges a read and a write of one source into one `read_write` edge before the store sees them.
- Module-level code has the file node as its source. The Go, Python and TypeScript extractors write such edges, for example `UnknownWrite(fileID, id)` for `var x = f()`.
- An initializer binding is a `references` edge from the data node to the name that the initializer holds, with `binding=init` (`goextract.MetaBinding`, `goextract.BindingInit`). Such an edge also carries the `read` access of that name.
- `graphstore.Reader` reads only outgoing edges. In `internal/query/traverse.go`, `BuildReverseAdjacency` and `BuildImplementsIndex` each scan every edge once and key one edge kind by target, with the same loop. If the request names a field, `buildContainsIndex` does one more scan. Sync keeps its own copy of the reverse index (`internal/indexer/sync.go`), because `internal/indexer` cannot import `internal/query`. Issue #274 tracks one shared builder for both packages.

## Goals / Non-Goals

**Goals:**

- Show the readers and writers of a data symbol with their role, and its initializer value, in CLI, MCP and the graph UI.
- Keep every existing line and field as it is, and keep the output of every other kind byte for byte.

**Non-Goals:**

- Compute the function that calls through the binding resolve to.
- Show the values that writes assign.
- Add `--json` to `node`.
- Merge the indexer's reverse index with the query one (#274).

## Decisions

### D1. Two new fields on DefinitionDetail

`DefinitionDetail` gains `AccessedBy []DataAccess` and `InitializedTo []*schema.Node`, after `CalledBy`. `DataAccess` is `{Node *schema.Node; Access string}`. Both stay nil for any kind that `IsDataKind` rejects. This follows `Owner`, which only a field fills. The rule T-01-18 in `internal/depclosure/dependency_closure_test.go` names `goextract` and `nodeid` as the only indexer packages that production `internal/query` may reach. The test enforces only the part of that rule that forbids the `internal/indexer` root. This change keeps to the whole rule, so `IsDataKind` moves from `internal/indexer/resolve` to `internal/indexer/goextract`, next to `IsWriteAccess` and the kind constants, and its callers in `resolve` and `internal/indexer/sync_bindings.go` call `goextract.IsDataKind`. The T-01-18 subtest is extended to enforce the whole rule.

### D2. One shared reverse index builder, one more scan only for a data symbol

A new unexported `reverseEdgeIndex(r graphstore.Reader, keep func(*schema.Edge) bool)` in `traverse.go` holds the scan loop: it keys each kept edge by its target. `BuildReverseAdjacency` and `BuildImplementsIndex` call it with their kind filters, and `BuildImplementsIndex` keeps its sort by source. A new `buildAccessIndex` calls it and keeps each `references` edge whose `access` metadata is not empty. `buildAccessIndex` sits behind a package variable, as `buildReverseAdjacency` does, so a test can count its calls. Without the shared loop, `buildAccessIndex` is a third copy of the same scan in `traverse.go`, and the repository shares a loop once it has a third copy. `internal/indexer` cannot import `internal/query`, so issue #274 must move `reverseEdgeIndex` to a package that both import. It stays in `internal/query` until then, because #274 chooses that shared home for all the full-scan indexes.

If the node is a data kind, `buildSingleDefDetail` calls `buildAccessIndex`. If any candidate is a data kind, `buildMultiDefDetail` calls it once and shares it across candidates, as it does with the contains index. Otherwise neither calls it.

The alternative was to widen `BuildReverseAdjacency` to collect `calls` and `references` edges in one scan. Its callers in `internal/query` (callers, impact, affected, seeding, expand and explore) need only `calls` edges, and each of them then pays for the wider map. The owner lookup of a field already pays one more scan, so one more scan for a data symbol fits the existing cost model.

### D3. Accessed by: one entry per source, stored role, stable order

`fetchCalledBy` becomes `fetchSources(node, idx)`, the one place that reads the sources of a node from a reverse index. It returns each source node with its edge, and skips a missing source as `fetchCalledBy` does today. `CalledBy` takes the source nodes from the calls index. `AccessedBy` takes the source nodes and the roles of the edges from the access index.

A source of kind `goextract.KindFile` is module-level code. It is kept, and prints as the file's path at line 1, because file nodes have the path as their name. An empty line then means that no stored code reads or writes the symbol. The store holds one edge per source and target, so each source gives one entry with the role of its edge.

The less function of `sortLocations` moves into `locationLess(a, b Location) bool`, and `sortLocations` calls it. `AccessedBy` is sorted with `sort.SliceStable` by `locationLess` over `nodeLocation` of each source with its signature set, then by role. Every field that the line prints is then a sort key, and no node id is one. Two entries that tie on every key print the same text, so their order does not change the output. `**Called by ←**` keeps its current order.

### D4. Initialized to: the initializer edges of the node itself

`fetchCalls` in `internal/query/node.go` becomes `fetchTargets(node, keep func(*schema.Edge) bool)`, the one place that reads a node's outgoing edges and their targets. `fetchCalls` calls it with the `calls` filter. `InitializedTo` calls it with a filter for `references` edges that `goextract.IsInitBinding` accepts. `IsInitBinding(e *schema.Edge) bool` is new, next to `IsWriteAccess`. It holds the check `e.Metadata[MetaBinding] == BindingInit`, which `resolve/bindings.go`, `resolve/collapse.go` and `internal/indexer/sync_bindings.go` now write inline, and those places call it instead. The new filter is otherwise one more copy of a check that the indexer already repeats in five places. A missing target is skipped, as it is for `calls`. The list keeps edge order. An initializer holds one name, so the list has at most one entry in practice.

### D5. Markdown and styled lines

For a data kind only, `RenderNode` adds `**Accessed by ←** <entries>` and `**Initialized to →** <refs>` after `**Called by ←**`. `formatNodeRef` takes an optional suffix that goes inside the parentheses, so an access entry is `name (path:line, role)`. `joinNodeRefs` takes an optional `func(i int) string` that gives the suffix of entry `i`, so each entry carries its own role. It starts with the same `name (path:line` as the existing references, so a reader that parses `**Called by ←**` can parse it. `**Initialized to →**` has no suffix. In `renderNodeSection`, an empty list prints no line. If any of the four lists is not empty, the section prints the trail header.

The label says "Initialized to" and not "Binds". The line shows the stored initializer value. It is not the call target that `callers` shows. If another write competes, that call target is absent.

`writeSingleDef` and `writeNodeSection` print the same lines with the label and path styles, and `writeNodeRefs` takes the same per-entry suffix. `writeNodeSection` takes the `*query.DefinitionDetail`, as `renderNodeSection` and `writeSingleDef` do, in place of its positional lists. They do not change otherwise. In particular, they keep leaving out the owner and language lines of a field, which only the plain view prints today. So `TestRenderNodeStrippedEqualsMarkdownContract` gets its new subtests for a variable and a constant, which have no such lines.

The styled multi-definition loop keeps its own copy of the budget constants, and it budgets on the stripped length. The plain loop budgets on the markdown length. This change does not unify the two loops, because a shared length would also move function sections near the budget in the terminal view. The plain view is the one that agents and MCP read, so its budget decides the contract.

The issue offered two lists, one for reads and one for writes. One list with a role keeps a `read_write` source on one line, the same as the stored edge.

### D6. UI wire, state and panel

`ui.proto` gains `message DataAccess { Node node = 1; string access = 2; }`. `NodeDefinition` gains `repeated DataAccess accessed_by = 6; repeated Node initialized_to = 7;`. As `calls` and `called_by` are, they are set only for a candidate whose detail was gathered. `GetNodeDetailResponse` gains `repeated DataAccess accessed_by = 10; repeated Node initialized_to = 11;`, set only in `NODE_DETAIL_MODE_SINGLE_DEF`. The numbers are new, so an old client ignores them. `task proto:gen` regenerates the Go and TypeScript code, and `task proto:drift` checks it.

`loadBrowseTarget` copies `accessedBy` and `initializedTo` into the single-definition `BrowseTargetState`, next to `calledBy`. `+page.svelte` passes them to `NeighborsPanel.svelte`. If a list has an entry, the panel shows it: "Accessed by" with each role, and "Initialized to". The server sets the lists only for a data kind, so the panel needs no list of data kinds.

The panel repeats one labeled list block for callers, callees and blast radius today, and the two new lists would add two more copies. So one Svelte snippet renders a labeled neighbor list. Its parameters are the heading, the empty text, the test-id prefix, whether an empty list hides the whole block, an optional suffix for the role, and a key function. Callers, callees and blast radius keep their current test ids and empty text. The data lists hide when they are empty. Callers, callees and blast radius keep their current keys. The two data lists key by node id, because two sources can share file, line and name, and Svelte rejects a keyed list with a duplicate key. The store holds one edge per source and target, so a node id is unique within a data list. The existing `web/tests/neighbors-panel.test.ts` and `web/tests/browse-state.test.ts` get the new cases.

## Risks / Trade-offs

- A data symbol with many readers gives a long line. `**Called by ←**` has the same property today, and the multi-definition budget still bounds the full output.
- The hugo goldens change. `task golden:regen` regenerates them. In the diff, each field section that gains an entry also gains the trail header line. If a longer data section pushes a later section past the body budget, that section moves to the list of other definitions, and the diff shows that move.
- The UI wire has no cap on the entries of a list. `called_by` has none either. A `Node` message holds a name, a path and a signature. The 16 MiB send limit then allows tens of thousands of entries across the 20 gathered definitions, next to their source blobs. A cap for every neighbor list is a separate change.
- A reader that has no stored node, for example a source from a file that sync removed, is skipped. The list can then be shorter than the stored edges.
- The data lines add bytes to both multi-definition loops. Near the budget, the styled view can then put a data section in full while the plain view lists it, or the reverse. The loops already differ in the same way for the markdown markers. The styled view is only for a terminal.
