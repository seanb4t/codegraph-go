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

`DefinitionDetail` gains `AccessedBy []DataAccess` and `InitializedTo []*schema.Node`, after `CalledBy`. `DataAccess` is `{Node *schema.Node; Access string}`. Both stay nil for any kind that `resolve.IsDataKind` rejects. This follows `Owner`, which only a field fills. `internal/query` can import `internal/indexer/resolve`, because neither package depends on the other.

### D2. One shared reverse index builder, one more scan only for a data symbol

A new unexported `reverseEdgeIndex(r graphstore.Reader, keep func(*schema.Edge) bool)` in `traverse.go` holds the scan loop: it keys each kept edge by its target. `BuildReverseAdjacency` and `BuildImplementsIndex` call it with their kind filters, and `BuildImplementsIndex` keeps its sort by source. A new `buildAccessIndex` calls it and keeps each `references` edge whose `access` metadata is not empty. `buildAccessIndex` sits behind a package variable, as `buildReverseAdjacency` does, so a test can count its calls.

If the node is a data kind, `buildSingleDefDetail` calls `buildAccessIndex`. If any candidate is a data kind, `buildMultiDefDetail` calls it once and shares it across candidates, as it does with the contains index. Otherwise neither calls it.

The alternative was to widen `BuildReverseAdjacency` to collect `calls` and `references` edges in one scan. Its callers in `internal/query` (callers, impact, affected, seeding, expand and explore) need only `calls` edges, and each of them then pays for the wider map. The owner lookup of a field already pays one more scan, so one more scan for a data symbol fits the existing cost model.

### D3. Accessed by: one entry per source, stored role, stable order

For each edge in the index entry of the node, the source node is read with `GetNode`. A missing source is skipped, as `fetchCalledBy` skips one. A source of kind `goextract.KindFile` is skipped, because module-level code is not a definition. The store holds one edge per source and target, so each source gives one entry with the role of its edge.

The entries are sorted by file, then name, then line, then signature, then kind, with `sort.SliceStable`. These are the keys of `sortLocations` that a node carries. Like `sortLocations`, the sort never uses a node id. Two entries that tie on every key print the same text, so their order does not change the output. `**Called by ←**` keeps its current order.

### D4. Initialized to: the initializer edges of the node itself

`fetchCalls` in `internal/query/node.go` becomes `fetchTargets(node, keep func(*schema.Edge) bool)`, the one place that reads a node's outgoing edges and their targets. `fetchCalls` calls it with the `calls` filter. `InitializedTo` calls it with a filter for `references` edges with `binding=init`. A missing target is skipped, as it is for `calls`. The list keeps edge order. An initializer holds one name, so the list has at most one entry in practice.

### D5. Markdown and styled lines

For a data kind only, `RenderNode` adds `**Accessed by ←** <entries>` and `**Initialized to →** <refs>` after `**Called by ←**`. An access entry is `name (path:line, role)`. It starts with the same `name (path:line` as the existing references, so a reader that parses `**Called by ←**` can parse it. `**Initialized to →**` uses `joinNodeRefs`. In `renderNodeSection`, an empty list prints no line. If any of the four lists is not empty, the section prints the trail header.

The label says "Initialized to" and not "Binds". The line shows the stored initializer value. It is not the call target that `callers` shows. If another write competes, that call target is absent.

`writeSingleDef` and `writeNodeSection` print the same lines with the label and path styles. Today they leave out the owner and language lines that `fieldFacts` gives the plain view. They start to print those lines too, so the styled view of a field mirrors the plain view. `TestRenderNodeStrippedEqualsMarkdownContract` gets subtests for a field and for a variable.

The issue offered two lists, one for reads and one for writes. One list with a role keeps a `read_write` source on one line, the same as the stored edge.

### D6. UI wire, state and panel

`ui.proto` gains `message DataAccess { Node node = 1; string access = 2; }`. `NodeDefinition` gains `repeated DataAccess accessed_by = 6; repeated Node initialized_to = 7;`. As `calls` and `called_by` are, they are set only for a candidate whose detail was gathered. `GetNodeDetailResponse` gains `repeated DataAccess accessed_by = 10; repeated Node initialized_to = 11;`, set only in `NODE_DETAIL_MODE_SINGLE_DEF`. The numbers are new, so an old client ignores them. `task proto:gen` regenerates the Go and TypeScript code, and `task proto:drift` checks it.

`loadBrowseTarget` copies `accessedBy` and `initializedTo` into the single-definition `BrowseTargetState`, next to `calledBy`. `+page.svelte` passes them to `NeighborsPanel.svelte`. If a list has an entry, the panel shows it: "Accessed by" with each role, and "Initialized to". The server sets the lists only for a data kind, so the panel needs no list of data kinds. One Svelte snippet renders a labeled neighbor list: the heading, the empty text, the entry buttons and an optional suffix for the role. Callers, callees, blast radius and the two new lists all use it. The existing `web/tests/neighbors-panel.test.ts` and `web/tests/browse-state.test.ts` get the new cases.

## Risks / Trade-offs

- A data symbol with many readers gives a long line. `**Called by ←**` has the same property today, and the multi-definition budget still bounds the full output.
- The hugo goldens change. `task golden:regen` regenerates them. In the diff, each field section that gains an entry also gains the trail header line, and nothing else changes.
- A reader that has no stored node, for example a source from a file that sync removed, is skipped. The list can then be shorter than the stored edges.
- The styled view of a field gains its owner and language lines. That changes the terminal output of `codegraph node` for a field. The styled view is only for a terminal. The plain view, which agents and MCP read, does not change for this.
