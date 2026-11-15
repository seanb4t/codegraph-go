# Node shows the access and the initializer value of a data symbol

Issue: #335

## Why

`codegraph node` and `codegraph_node` show a data symbol's location, its owner for a field, and its calls. They do not show which functions read or write the symbol, or the value that its initializer holds. The index already stores both facts. A `references` edge from a reader or writer to the symbol records the access as `read`, `write` or `read_write`. A `references` edge with `binding=init` goes from the symbol to the name that its initializer holds. An agent that asks `node` about a field or a variable must now read source files to find these facts.

## What Changes

- For a data symbol (a variable, a constant or a field), `node` adds two lines after `**Called by ←**`:
  - `**Accessed by ←**` lists each definition that reads or writes the symbol, once, with its access role: `deposit (p/Account.java:7, read_write)`.
  - `**Initialized to →**` lists the value that the symbol's initializer holds: `start (main.go:3)`. The name says that the line shows the initializer, not the function that calls through the symbol resolve to.
- The single-definition view always prints both lines for a data symbol, as it always prints `**Calls →**` and `**Called by ←**`. In a section of the multi-definition view, an empty list prints no line. If all four lists of a section are empty, the section prints no trail header.
- Entries of `**Accessed by ←**` are ordered by file, then name, then line, the order that `callers` uses.
- Module-level code that reads or writes a symbol has the file as its source, not a definition. Such an access is not listed.
- The styled terminal view of `codegraph node` shows the same two lines. It also starts to show the owner and language lines of a field, which the plain view already shows.
- The graph UI gets the same facts. `NodeDefinition` and `GetNodeDetailResponse` in `ui.proto` gain the fields `accessed_by` and `initialized_to`. The browse state carries them, and the neighbors panel shows each list that has an entry.
- The output for every other kind does not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `data-symbols`: adds four requirements for the access and initializer lines of `node` and the graph UI.

## Impact

- Go code: `internal/query` (`detail.go`, `node.go`, `traverse.go`, `render_markdown.go`), `internal/cli/present/node.go` and `internal/uiserver/handlers.go`.
- Wire: `internal/uiproto/uiv1/ui.proto` and its generated Go and TypeScript code.
- Web code: `web/src/lib/browse-state.ts`, `web/src/routes/browse/+page.svelte` and `web/src/lib/components/browse/NeighborsPanel.svelte`.
- Output: additive lines in CLI and MCP markdown for data symbols, and additive fields on the UI wire. Field names, types and the order of existing lines do not change.
- Goldens: the hugo captures `go-node.json` and `go-node-mcp.json` (`Page`) and `go-node-multi.json` (`Site`) under `testdata/golden/corpus/hugo/` show field sections. A field section with an accessor gains the trail header and an `**Accessed by ←**` line, so these goldens are regenerated. The wire oracle's node scenario asks for a function and does not change.
- Cost: a `node` call on a data symbol scans the edges once more, as the owner lookup of a field already does. Other kinds pay nothing.

## Out of scope

- The function that calls through a binding resolve to. That target depends on the analysis context (`go-production` or `go-test`) and on competing writes, and `callers` already shows the calls.
- The values that writes assign. A write is listed in `**Accessed by ←**` with its role, without its value.
- Access from module-level code, whose source is the file node.
- A `--json` output for `node`, which has none today.
- The owner of a field in the graph UI, which the UI does not show today.
- The indexer's own copy of the reverse edge index in `internal/indexer/sync.go`, which issue #274 tracks.
