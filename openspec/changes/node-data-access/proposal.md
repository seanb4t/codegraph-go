# Node shows the access and the binding of a data symbol

Issue: #335

## Why

`codegraph node` and `codegraph_node` show a data symbol's location, its owner for a field, and its calls. They do not show which functions read or write the symbol, or the value that its initializer binds. The index already stores both facts. A `references` edge from a reader or writer to the symbol records the access as `read`, `write` or `read_write`. A `references` edge with `binding=init` goes from the symbol to the name that its initializer holds. An agent that asks `node` about a field or a variable must now read source files to find these facts.

## What Changes

- For a data symbol (a variable, a constant or a field), `node` adds two lines after `**Called by ←**`:
  - `**Accessed by ←**` lists each definition that reads or writes the symbol, once, with its access role: `deposit (p/Account.java:7, read_write)`.
  - `**Binds →**` lists the value that the symbol's initializer holds: `start (run.go:5)`.
- The single-definition view always prints both lines for a data symbol, as it always prints `**Calls →**` and `**Called by ←**`. In a section of the multi-definition view, an empty list prints no line. If all four lists of a section are empty, the section prints no trail header.
- Entries of `**Accessed by ←**` are ordered by file, then name, then line, the order that `callers` uses.
- The styled terminal view of `codegraph node` shows the same two lines.
- The graph UI gets the same facts. `NodeDefinition` and `GetNodeDetailResponse` in `ui.proto` gain the fields `accessed_by` and `binds`, and the browse view's neighbors panel shows them.
- The output for every other kind does not change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `data-symbols`: adds four requirements for the access and binding lines of `node` and the graph UI.

## Impact

- Code: `internal/query` (`DefinitionDetail`, `detail.go`, `render_markdown.go`), `internal/cli/present/node.go`, `internal/uiserver/handlers.go`, `internal/uiproto/uiv1/ui.proto` and its generated Go and TypeScript code, and `web/src/lib/components/browse/NeighborsPanel.svelte`.
- Output: additive lines in CLI and MCP markdown for data symbols, and additive fields on the UI wire. Field names, types and the order of existing lines do not change.
- Goldens: the hugo `Page` captures under `testdata/golden/corpus/hugo/` show field sections, so they gain `**Accessed by ←**` lines and are regenerated. The wire oracle's node scenario asks for a function and does not change.
- Cost: a `node` call on a data symbol scans the edges once more, as the owner lookup of a field already does. Other kinds pay nothing.

## Out of scope

- The function that calls through a binding resolve to. That target depends on the analysis context (`go-production` or `go-test`) and on competing writes, and `callers` already shows the calls. `**Binds →**` shows only the stored initializer value.
- The values that writes assign. A write is listed in `**Accessed by ←**` with its role, without its value.
- A `--json` output for `node`, which has none today.
- The owner of a field in the graph UI, which the UI does not show today.
