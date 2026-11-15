# Design

## Context

`Engine.NodeSelected` builds a `DefinitionDetail` (`internal/query/detail.go`) for one definition, or a `MultiDefDetail` that builds one per candidate on demand. `RenderNode` and `renderNodeSection` in `internal/query/render_markdown.go` turn it into the markdown that the CLI prints and `codegraph_node` returns. `writeSingleDef` and `writeNodeSection` in `internal/cli/present/node.go` print the same lines with styles. `TestRenderNodeStrippedEqualsMarkdownContract` checks that the two agree. `nodeDefinitionToProto` and `nodeDetailToProto` in `internal/uiserver/handlers.go` map a detail onto `ui.proto`.

The stored facts:

- A read or a write is a `references` edge from the reader or writer to the data node. Its `access` metadata (`goextract.MetaAccess`) is `read`, `write` or `read_write`. `resolve.withMergedAccess` merges the edges of one source into one edge.
- An initializer binding is a `references` edge from the data node to the name that the initializer holds, with `binding=init` (`goextract.MetaBinding`, `goextract.BindingInit`). Such an edge also carries the `read` access of that name.
- `graphstore.Reader` reads only outgoing edges. `query.BuildReverseAdjacency` keeps only `calls` edges. If the request names a field, `buildContainsIndex` scans every edge once to find its owner.

## Goals / Non-Goals

**Goals:**

- Show the readers and writers of a data symbol with their role, and its initializer binding, in CLI, MCP and the graph UI.
- Keep every existing line and field as it is, and keep the output of every other kind byte for byte.

**Non-Goals:**

- Compute the function that calls through the binding resolve to.
- Show the values that writes assign.
- Add `--json` to `node`.

## Decisions

### D1. Two new fields on DefinitionDetail

`DefinitionDetail` gains `AccessedBy []DataAccess` and `Binds []*schema.Node`, after `CalledBy`. `DataAccess` is `{Node *schema.Node; Access string}`. Both stay nil for any kind that `resolve.IsDataKind` rejects. This follows `Owner`, which only a field fills.

### D2. One edge scan, only for a data symbol

A new `buildAccessIndex(r graphstore.Reader)` in `internal/query/traverse.go` scans every edge once. It keeps each `references` edge whose `access` metadata is not empty, keyed by its target. If the node is a data kind, `buildSingleDefDetail` calls it. If any candidate is a data kind, `buildMultiDefDetail` calls it once and shares it across candidates, as it does with the contains index. Otherwise neither calls it. `BuildReverseAdjacency` does not change, because `internal/indexer` sync uses it and expects only `calls` edges.

The alternative was one scan that collects `calls`, `contains` and `references` edges together. It saves a scan for a data symbol, but changes `BuildReverseAdjacency`, which sync shares. The scan that the owner lookup already pays shows that one more scan for a data symbol is acceptable.

### D3. Accessed by: one entry per source, merged role, stable order

For each edge in the index entry of the node, the source node is read with `GetNode`. A missing source is skipped, as `fetchCalledBy` skips one. Edges of one source merge into one entry with `goextract.MergeAccess`, so a source with a `read` edge and a `write` edge shows `read_write`. The entries are sorted by file, then name, then line, then id, so the order does not depend on the store's key order. `**Called by ←**` keeps its current order.

### D4. Binds: the initializer edges of the node itself

`Binds` takes the targets of the node's own outgoing `references` edges with `binding=init`, from `IterateEdges(node.Id)`. The read is the same as the one in `fetchCalls`, filtered on the other kind. A missing target is skipped. The list keeps edge order. An initializer holds one name, so the list has at most one entry in practice.

### D5. Markdown and styled lines

`RenderNode` adds, for a data kind only, `**Accessed by ←** <entries>` and `**Binds →** <refs>` after `**Called by ←**`. An access entry is `name (path:line, role)`. It starts with the same `name (path:line` as the existing references, so a reader that parses `**Called by ←**` can parse it. `Binds` uses `joinNodeRefs`. In `renderNodeSection`, an empty list prints no line. If any of the four lists is not empty, the section prints the trail header. `writeSingleDef` and `writeNodeSection` print the same lines with the label and path styles. They keep `TestRenderNodeStrippedEqualsMarkdownContract` passing, with new subtests for a data node.

The issue offered two lists, one for reads and one for writes. One list with a role keeps a `read_write` source on one line, the same as the stored edge.

### D6. UI wire and panel

`ui.proto` gains `message DataAccess { Node node = 1; string access = 2; }`. `NodeDefinition` gains `repeated DataAccess accessed_by = 6; repeated Node binds = 7;`. `GetNodeDetailResponse` gains `repeated DataAccess accessed_by = 10; repeated Node binds = 11;`, filled only in `NODE_DETAIL_MODE_SINGLE_DEF`. The numbers are new, so an old client ignores them. `task proto:gen` regenerates the Go and TypeScript code, and `task proto:drift` checks it. If the target is a data symbol, `NeighborsPanel.svelte` shows an "Accessed by" list with each role and a "Binds" list. A helper in `web/src/lib/` builds the rows, so a vitest test can check them without rendering the panel.

## Risks / Trade-offs

- A data symbol with many readers gives a long line. `**Called by ←**` has the same property today, and the multi-definition budget still bounds the full output.
- The hugo `Page` goldens change. `task golden:regen` regenerates them, and the diff must show only new `**Accessed by ←**` and `**Binds →**` lines.
- A reader that has no stored node, for example a source from a file that sync removed, is skipped. The list can then be shorter than the stored edges.
