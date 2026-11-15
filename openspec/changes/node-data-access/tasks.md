# Tasks

## 1. Query detail

- [ ] 1.1 Add `reverseEdgeIndex` to `internal/query/traverse.go`, and route `BuildReverseAdjacency` and `BuildImplementsIndex` through it (D2). The existing tests of callers, impact and dispatch pass unchanged.
- [ ] 1.2 Add `DataAccess` and the fields `AccessedBy` and `InitializedTo` to `DefinitionDetail` in `internal/query/detail.go` (D1). Add `buildAccessIndex` behind a package variable (D2). Fill `AccessedBy` in `buildSingleDefDetail` and `buildMultiDefDetail` (D3). `TestNodeDataAccess` in `internal/query/detail_test.go` covers these cases:
  - a source with a `read_write` edge, which gives one entry with `read_write`
  - a data symbol `a` whose initializer reads the variable `b`, which lists `a` with `read` for `b`
  - a source of kind `file` (`var x = f()`), which gives no entry
  - a missing source, which is skipped
  - two sources in `b.go` and `a.go`, which come out `a.go` first
  - two sources in one file whose names sort in the reverse of their lines, which come out by name
  - two sources with one name in one file at lines 9 and 3, which come out line 3 first
- [ ] 1.3 Add the subtest "only a data symbol scans the access index" to `TestNodeDataAccess`. It counts the calls of the `buildAccessIndex` package variable. It expects these counts:
  - 0 for a function
  - 1 for a single-definition variable
  - 1 for the `size` field and method pair
  - 1 for a multi-definition name with two data candidates
  - 0 for a multi-definition name with only functions
- [ ] 1.4 Add the property test `TestAccessOrderIgnoresEdgeOrder` with `pgregory.net/rapid`, after the pattern of `internal/indexer/spill_roundtrip_test.go`. It permutes the edges of one index entry and asserts the same `AccessedBy` slice (D3).
- [ ] 1.5 Turn `fetchCalls` into `fetchTargets` with an edge filter, and fill `InitializedTo` with it (D4). `TestNodeDataAccess` adds three cases:
  - a variable whose `binding=init` edge targets a function, which gives that function
  - a variable with a `references` edge without `binding=init`, which gives no entry
  - a `binding=init` edge to an id that has no node, which gives an empty list and no error
- [ ] 1.6 Print `**Accessed by ←**` and `**Initialized to →**` in `RenderNode` and `renderNodeSection` (D5). `TestRenderNodeDataLines` in `internal/query/render_markdown_test.go` covers the requirement "The data lines appear only for data symbols". It compares the full output in five cases:
  - a single-definition field with entries
  - a single-definition constant with two empty lists
  - a multi-definition section with only an access list, which prints the trail header
  - a multi-definition variable section with only an initializer entry, which prints the trail header
  - a multi-definition data section with four empty lists, which prints no trail header

  The existing render tests for functions pass unchanged.

## 2. CLI and MCP

- [ ] 2.1 Print the field owner and language lines, and the two data lines, in `writeSingleDef` and `writeNodeSection` in `internal/cli/present/node.go` (D5). `TestRenderNodeStrippedEqualsMarkdownContract` gets the subtests "SingleDef field", "SingleDef variable" and "MultiDef data". They compare the stripped styled output with `query.RenderNode` and `query.RenderNodeMultiDef`.
- [ ] 2.2 Add `TestNodeShowsDataAccess` to `internal/cli/data_symbol_queries_test.go`. It runs `codegraph node` on the `p/Account.java` fixture and on Go fixtures. It has one subtest for each scenario of the spec delta, except the graph UI scenario:
  - a method reads and writes a field
  - readers in two files
  - module-level code writes a variable
  - a variable holds a function
  - nothing accesses a constant
  - a field and a method share a name, so the data lines appear only for data symbols
  - a function keeps its output
- [ ] 2.3 Add to `internal/mcp/data_symbols_test.go` a check that `codegraph_node` for `balance` returns the line `**Accessed by ←** deposit (p/Account.java:7, read_write)`.

## 3. Graph UI

- [ ] 3.1 Add `DataAccess`, `NodeDefinition.accessed_by` (6), `NodeDefinition.initialized_to` (7), `GetNodeDetailResponse.accessed_by` (10) and `GetNodeDetailResponse.initialized_to` (11) to `internal/uiproto/uiv1/ui.proto` (D6). Run `task proto:gen`. `task proto:drift` passes.
- [ ] 3.2 Map the new fields in `nodeDefinitionToProto` and `nodeDetailToProto` in `internal/uiserver/handlers.go`. `TestGetNodeDetailDataAccess` in `internal/uiserver/handlers_test.go` has two cases:
  - the scenario "The graph UI asks for a field", with one `accessed_by` entry for `deposit` with `read_write` and an empty `initialized_to`
  - `size` as a multi-definition name, where the gathered field definition carries its entry, and a candidate past `uiMultiDefCap` carries empty lists
- [ ] 3.3 Carry `accessedBy` and `initializedTo` through `BrowseTargetState` in `web/src/lib/browse-state.ts` and the props in `web/src/routes/browse/+page.svelte`. `web/tests/browse-state.test.ts` checks that `loadBrowseTarget` keeps both lists of a single-definition response.
- [ ] 3.4 Add one snippet for a labeled neighbor list to `NeighborsPanel.svelte` (D6). Render callers, callees, blast radius, "Accessed by" and "Initialized to" through it. `web/tests/neighbors-panel.test.ts` checks that a list with entries shows its heading and each role, and that an empty data list shows no heading. The existing panel cases pass unchanged. `pnpm --dir web test` and `pnpm --dir web check` pass.

## 4. Goldens

- [ ] 4.1 Run `task golden:regen`. In the diff of `testdata/golden/corpus/`, the only changes are in hugo field sections of `go-node.json`, `go-node-mcp.json` and `go-node-multi.json`. Each such section gains an `**Accessed by ←**` line or an `**Initialized to →**` line. If the section had no trail header, it also gains the trail header. `task test:golden` passes. `internal/cli/testdata/plain/node-symbol.golden` and the wire oracle transcripts do not change, because they show functions.

## 5. Release

- [ ] 5.1 Add a changelog fragment: `task changie -- new --interactive=false -k Features -b "node, codegraph_node and the graph UI list the readers and writers of a data symbol with their access role, and the value that its initializer holds."`
