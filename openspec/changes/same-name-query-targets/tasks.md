# Tasks

## 1. Start nodes for every definition

- [ ] 1.1 In `internal/query/overloads.go`, make `symbolTargets` build its start nodes from `enumerateSymbolDefs` plus the set nodes, in the order of design D1. Add `targets.spans`, computed by one function on each return path (design D2). Remove the branch that calls `resolveSymbolNode`. Verify with a new `TestCalleesOfANameWithTwoDefinitions` in `internal/query/overloads_test.go`. It covers the scenario "Two functions of one name in two files" and fails on the current code.
- [ ] 1.2 Add `TestCallersOfAStubAndTheRealFunction` and `TestCalleesOfAMethodAndATestHelper`. They index small Python fixtures through the real indexer. They cover the scenarios "A typing stub and the real function" and "A method and a test helper of one name". Verify that both fail before 1.1 and pass after it.
- [ ] 1.3 Add `TestImpactOverTwoDefinitions` and `TestCallersOfASetAndADefinitionOutsideIt` for the scenarios "Impact over two definitions" and "An overload set and a definition outside it". Verify that both pass.
- [ ] 1.4 Update `TestImpact_DispatchTraversal` in `internal/query/traverse_test.go` and the comment of its fixture. `Handle` has two definitions, so `NodeCount` is 3 and both `Handle` methods are in `Affected`. Update the comment of `TestCallers_DispatchTraversal`, which says the query resolves to the lower node id. Verify that `go test ./internal/query -run DispatchTraversal` passes.
- [ ] 1.5 Verify that `go test ./internal/query -run 'Overload|Cap|Sorted|Dangling'` passes. The only assertion that can change there is a row order of an overloaded query (design D4). Record each such change in the pull request body.

## 2. Row order without node ids

- [ ] 2.1 Add the definition label and the signature as the last two keys of `sortLocations` (design D4). Verify with `TestOneCallerOfTwoDefinitionsIsOrderedByDefinition`, which covers the scenario "One caller of two definitions" and asserts the order of the two rows.
- [ ] 2.2 Add a property test `TestQueryResultDoesNotDependOnNodeIDs` with `pgregory.net/rapid`. Draw a graph with two to four same-named functions, drawn callers and callees, and at least one caller of two definitions. Build it twice in the fake reader of `traverse_test.go`, with node ids drawn at random and all other fields equal. Require equal `CallersResult`, `CalleesResult` and `ImpactResult` values, with row order, labels and the `Definitions` list included, also under a drawn `limit`. Positive assertion: a run whose two builds have the same ids fails the test.

## 3. Definition labels in the results

- [ ] 3.1 Add `Location.Definition`, the label function and the `Definitions` list of `CallersResult`, `CalleesResult` and `ImpactResult` (design D3). If `spans` is false, leave them empty. Verify with `TestCallerNamesTheDefinitionItCalls` and `TestImpactReachesOneSymbolFromTwoDefinitions`.
- [ ] 3.2 Add `TestSignatureThatMatchesInTwoFilesSpansDefinitions` and `TestAmbiguousCallerInASpanningQueryHasNoDefinition`. They cover the two scenarios of those names. Verify that both pass.
- [ ] 3.3 Add `TestAFunctionAndAPackageOfOneName` for the requirement "A definition is named by its file and line". It uses a node with an empty `FilePath` and asserts the label `package example.com/app/cache`. Verify that it passes.
- [ ] 3.4 Add `TestJSONOfANameWithTwoDefinitionsHasDefinitions`, `TestJSONOfANameWithOneDefinitionHasNoNewKeys` and `TestJSONOfOneOverloadSetHasNoDefinitionKeys`. They marshal the results and compare the exact key sets. They cover "JSON output over several definitions", "One definition in JSON" and "One overload set alone". Verify that all three pass.

## 4. Text and table rendering

- [ ] 4.1 Add the two shared functions of design D5 beside `OverloadSuffix`: one for the definitions line, with the cap of `nodeMultiDefListCap`, and one for the row suffix. Use them in the plain branches and the styled renderers of `callers`, `callees` and `impact`. Verify with `TestCallersTextListsTheDefinitions` and `TestCallersTextOfNoResultListsTheDefinitions` in `internal/cli/query_cli_test.go`. The second covers "A query that spans definitions and finds no result".
- [ ] 4.2 Make `renderLocationTable` build its header and cells from two flags. Print the definitions line in the three markdown renderers, also for a result with no row. Verify with new cases in `TestRenderCallersMarkdown`, `TestRenderCalleesMarkdown` and `TestRenderImpactMarkdown`: two definitions with rows, two definitions with no row, and an impact row with an empty Definition cell.
- [ ] 4.3 Add a case to `TestCallersTextListsTheDefinitions` with 25 definitions. Assert that the line holds 20 labels and `+5 more`.
- [ ] 4.4 Verify the scenario "One definition in text" with `go test ./internal/cli -run 'TestPlainGolden$' -count=1` and no update flag. Add an assertion to `TestCallersTextListsTheDefinitions` that the output for a name with one definition holds no `definitions named`.

## 5. Selection by file

- [ ] 5.1 Add the file parameter to `CallersSelected`, `CalleesSelected` and `ImpactSelected` and the hard filter in `symbolTargets` (design D6). Verify with `TestCalleesWithFileSelectsOneDefinition`, `TestCalleesWithTheEndOfAFilePath`, `TestCallersWithAFileThatMatchesNothingListsEachFileOnce` and `TestCallersWithFileAndSignature`, one for each scenario of the requirement "A file selects the definitions in it".
- [ ] 5.2 Add a property test `TestFileSelectionPartitionsTheUnion` with `pgregory.net/rapid`. Draw a graph and run a query with no file. Run it again with each file that holds a definition. The rows of the first run equal the union of the rows of the other runs. Positive assertion: a run with fewer than two files that hold a definition fails the test.
- [ ] 5.3 Add one helper in `internal/cli` that registers `--signature` and `--file` with the short form `-f`. Use it in `callers`, `callees` and `impact`. Verify with `TestCalleesFileFlag` in `internal/cli/query_cli_test.go`. It runs the real command tree and asserts the non-zero exit status of the no-match case.
- [ ] 5.4 Run `task docs:cli` and commit `docs/CLI-REFERENCE.md`. Verify that `task docs:cli:drift` passes.

## 6. MCP tools

- [ ] 6.1 Add the optional `file` input to `CallersArgs`, `CalleesArgs` and `ImpactArgs` in `internal/mcp/tools.go`, in the order of the JSON names. Pass it to the traversal. Verify with `TestCallersToolOnANameWithTwoDefinitions` and `TestCalleesToolWithAFile` in `internal/mcp/overloads_test.go`.
- [ ] 6.2 Add `TestDefinitionToolsDeclareOptionalFile` in `internal/mcp/tools_schema_drift_test.go`. It asserts an optional string property `file` on the three input schemas and covers "The input schema of the three tools". Verify that it passes.
- [ ] 6.3 Add `TestCallersToolTableHasADefinitionColumn`, `TestCallersToolOfNoCallerListsTheDefinitions`, `TestImpactToolRowReachedByTheWalkHasNoDefinition` and `TestCallersToolTableOfOneDefinitionHasThreeColumns`. They cover the four table scenarios of the `mcp-server` delta. Verify that all four pass.
- [ ] 6.4 Update `internal/mcp/resources/callers.md`, `callees.md` and `impact.md` with the `file` input and the behavior for several definitions. Verify that `go test ./internal/mcp` passes.

## 7. Frozen outputs and docs

- [ ] 7.1 Capture the plain CLI goldens with `go test ./internal/cli -run 'TestPlainGolden$' -update-plain-goldens -count=1` and the wire oracle transcripts with the capture command under `test/wireoracle/cmd/wireoracle`. For each file that changes, record the reason in the pull request body. Verify that `git diff --stat` shows no change under `testdata/golden/corpus` after `task golden:regen` (design D7).
- [ ] 7.2 Widen the purpose line of `openspec/specs/graph-queries/spec.md` so it covers a name with several definitions and not only overloads. Verify that `openspec validate --strict` passes.
- [ ] 7.3 Add one paragraph on several definitions and `--file` beside the `--signature` text in `docs/LANGUAGE-CAPABILITY-MATRIX.md`. Verify that `go test ./internal/indexer/capability` passes.
- [ ] 7.4 Run the two queries of issue #318 on the locked `psf/requests` corpus: `callers to_key_val_list` and `callees prepare_url`. Paste the output in the pull request body. Verify that the first lists a caller and the second lists `requote_uri`.
- [ ] 7.5 Run `task test`, `task eval:semantic` and `task lint`. Verify that all three pass.
- [ ] 7.6 Add the changelog fragment: `task changie -- new --interactive=false -k Fixes -b "callers, callees and impact answer for every definition of a name and say which definitions exist. The new --file flag selects the definitions in one file."`

## Workflow follow-up

- File an issue for the definition label and the file selector in the graph UI (design D8).
- File an issue for the `--line` conflict between the `graph-queries` spec and the three commands.
- Run the implementation review with `/sb:openspec-review same-name-query-targets implementation`.
- Archive the change with `/opsx:archive` in the implementation pull request.
