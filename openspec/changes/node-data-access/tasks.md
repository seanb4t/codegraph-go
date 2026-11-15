# Tasks

## 1. Query detail

- [ ] 1.1 Add `DataAccess` and the fields `AccessedBy` and `Binds` to `DefinitionDetail` in `internal/query/detail.go` (D1). Add `buildAccessIndex` to `internal/query/traverse.go` behind a package variable, as `buildReverseAdjacency` is (D2). Fill `AccessedBy` in `buildSingleDefDetail` and `buildMultiDefDetail` (D3). `TestNodeDataAccess` in `internal/query/detail_test.go` covers these cases:
  - a source with one `read_write` edge
  - a source with a `read` edge and a `write` edge, which merge to `read_write`
  - two sources in `b.go` and `a.go`, which come out `a.go` first
  - a missing source, which is skipped
  - the subtest "a function scans no access index". It counts the calls of the package variable. It expects 0 for a function, and 1 for a multi-definition name with two data candidates.
- [ ] 1.2 Fill `Binds` from the node's own `references` edges with `binding=init` (D4). `TestNodeDataAccess` adds two cases:
  - a variable whose initializer edge targets a function, which gives that function
  - a variable with a `references` edge without `binding=init`, which gives no entry.
- [ ] 1.3 Print `**Accessed by ←**` and `**Binds →**` in `RenderNode` and `renderNodeSection` (D5). `TestRenderNodeDataLines` in `internal/query/render_markdown_test.go` compares the full output in four cases:
  - a single-definition field with entries
  - a single-definition constant with two empty lists
  - a multi-definition section with only an access list, which prints the trail header
  - a multi-definition data section with four empty lists, which prints no trail header.

  The existing render tests for functions pass unchanged.

## 2. CLI and MCP

- [ ] 2.1 Print the same lines in `writeSingleDef` and `writeNodeSection` in `internal/cli/present/node.go` (D5). `TestRenderNodeStrippedEqualsMarkdownContract` gets the subtests "SingleDef data" and "MultiDef data". They compare the stripped styled output with `query.RenderNode` and `query.RenderNodeMultiDef`.
- [ ] 2.2 Add `TestNodeShowsDataAccess` to `internal/cli/data_symbol_queries_test.go`. It runs `codegraph node` on the `p/Account.java` fixture and on a Go fixture. It has one subtest for each scenario of the spec delta, except the graph UI scenario:
  - a method reads and writes a field
  - a variable holds a function
  - nothing accesses a constant
  - readers in two files
  - a field and a method share a name
  - a function keeps its output.
- [ ] 2.3 Add to `internal/mcp/data_symbols_test.go` a check that `codegraph_node` for `balance` returns the line `**Accessed by ←** deposit (p/Account.java:7, read_write)`.

## 3. Graph UI

- [ ] 3.1 Add `DataAccess`, `NodeDefinition.accessed_by` (6), `NodeDefinition.binds` (7), `GetNodeDetailResponse.accessed_by` (10) and `GetNodeDetailResponse.binds` (11) to `internal/uiproto/uiv1/ui.proto` (D6). Run `task proto:gen`. `task proto:drift` passes.
- [ ] 3.2 Map the new fields in `nodeDefinitionToProto` and `nodeDetailToProto` in `internal/uiserver/handlers.go`. `TestGetNodeDetailDataAccess` in `internal/uiserver/handlers_test.go` covers the scenario "The graph UI asks for a field". It expects one `accessed_by` entry for `deposit` with `read_write`, and an empty `binds`.
- [ ] 3.3 Add a helper in `web/src/lib/data-access.ts` that turns `accessed_by` and `binds` into panel rows. Show the rows in `NeighborsPanel.svelte` for a data symbol. `web/src/lib/data-access.test.ts` checks the rows for one entry of each role and for empty lists. `pnpm --dir web test` and `pnpm --dir web check` pass.

## 4. Goldens

- [ ] 4.1 Run `task golden:regen`. The diff of `testdata/golden/corpus/` adds only `**Accessed by ←**` and `**Binds →**` lines, all in hugo field sections. `task test:golden` passes. `internal/cli/testdata/plain/node-symbol.golden` and the wire oracle transcripts do not change, because they show functions.

## 5. Release

- [ ] 5.1 Add a changelog fragment: `task changie -- new --interactive=false -k Features -b "node, codegraph_node and the graph UI list the readers and writers of a data symbol with their access role, and the value that its initializer binds."`
