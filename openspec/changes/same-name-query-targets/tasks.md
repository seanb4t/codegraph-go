# Tasks

## 1. Start nodes for every definition

- [ ] 1.1 In `internal/query/overloads.go`, make `symbolTargets` return every node with the name and set the new `targets.spans` field (design D1, D2). Remove the branch that calls `resolveSymbolNode`. Verify with a new `TestCalleesOfANameWithTwoDefinitions` in `internal/query/overloads_test.go`. It covers the scenario "Two functions of one name in two files" and fails on the current code.
- [ ] 1.2 Add `TestCallersOfAStubAndTheRealFunction` and `TestCalleesOfAMethodAndATestHelper`. They index small Python fixtures through the real indexer. They cover the scenarios "A typing stub and the real function" and "A method and a test helper of one name". Verify that both fail before 1.1 and pass after it.
- [ ] 1.3 Add `TestImpactOverTwoDefinitions` and `TestCallersOfASetAndADefinitionOutsideIt` for the scenarios "Impact over two definitions" and "An overload set and a definition outside it". Verify that both pass.
- [ ] 1.4 Add a property test `TestQueryAnswerDoesNotDependOnNodeIDs` with `pgregory.net/rapid`. Draw two to four same-named functions with drawn callers and callees. Build the store twice with different file paths, so the ids differ. Require the same `callers` and `callees` rows by name. Positive assertion: a run that draws no caller fails the test.
- [ ] 1.5 Verify that `go test ./internal/query -run 'Overload|Dispatch|Cap|Sorted'` passes with no assertion changed, so the overload, dispatch and limit behavior is the same (design D6).

## 2. Definition labels in the results

- [ ] 2.1 Add `Location.Definition` and the `Definitions` list of `CallersResult`, `CalleesResult` and `ImpactResult` (design D3). If `spans` is false, leave them empty. Verify with `TestCallerNamesTheDefinitionItCalls`, `TestOneCallerOfTwoDefinitionsIsListedTwice` and `TestImpactReachesOneSymbolFromTwoDefinitions`. They cover the three scenarios of the requirement "A row of a query that spans definitions names its definition".
- [ ] 2.2 Add `TestJSONOfANameWithOneDefinitionHasNoNewKeys` and `TestJSONOfOneOverloadSetHasNoDefinitionKeys`, which marshal the results and compare the exact key sets. They cover "One definition in JSON" and "One overload set alone". Verify that both pass.
- [ ] 2.3 Render the definitions line and the row suffix in the CLI text of `callers`, `callees` and `impact` (`internal/cli/present/results.go` and the plain branches). Render the definitions line and the Definition column in `renderLocationTable` (design D4). Verify with new cases in `TestRenderCallersMarkdown`, `TestRenderCalleesMarkdown` and `TestRenderImpactMarkdown`. Verify the CLI text with a new `TestCallersTextListsTheDefinitions` in `internal/cli/query_cli_test.go`.
- [ ] 2.4 Verify that the existing plain goldens `internal/cli/testdata/plain/callers.golden` and `callees.golden` are unchanged, which covers "One definition in text".

## 3. Selection by file

- [ ] 3.1 Add `Selector.File` and the filter in `symbolTargets`, with the not-found error that lists the files (design D5). Verify with `TestCalleesWithFileSelectsOneDefinition`, `TestCallersWithAFileThatHasNoDefinitionListsTheFiles` and `TestCallersWithFileAndSignature`, one for each scenario of the requirement "A file selects the definitions in it".
- [ ] 3.2 Add `--file` with the short form `-f` to `callers`, `callees` and `impact` in `internal/cli`. Verify with `TestCalleesFileFlag` in `internal/cli/query_cli_test.go`. It runs the real command tree and asserts the non-zero exit status of the no-definition case.
- [ ] 3.3 Run `task docs:cli` and commit `docs/CLI-REFERENCE.md`. Verify that `task docs:cli:drift` passes.

## 4. MCP tools

- [ ] 4.1 Add the optional `file` input to `CallersArgs`, `CalleesArgs` and `ImpactArgs` in `internal/mcp/tools.go`, in the order of the JSON names. Pass it to the selector. Verify with `TestCallersToolOnANameWithTwoDefinitions` and `TestCalleesToolWithAFile` in `internal/mcp/overloads_test.go`. Verify the scenario "The input schema of the three tools" with a schema assertion.
- [ ] 4.2 Add `TestCallersToolTableHasADefinitionColumn` and `TestCallersToolTableOfOneDefinitionHasThreeColumns` for the two table scenarios of the `mcp-server` delta. Verify that both pass.
- [ ] 4.3 Update `internal/mcp/resources/callers.md`, `callees.md` and `impact.md` with the `file` input and the behavior for several definitions. Verify that `go test ./internal/mcp` passes, including `tools_schema_drift_test.go` and `instructions_contract_test.go`.

## 5. Frozen outputs and docs

- [ ] 5.1 Run `task golden:regen` and `go test -count=1 ./test/wireoracle/...`. For each capture or transcript that changes, record the queried name and the reason in the pull request body. Verify that no capture for a name with one definition changes (design D7).
- [ ] 5.2 Widen the purpose line of `openspec/specs/graph-queries/spec.md` so it covers a name with several definitions and not only overloads. Verify that `openspec validate --strict` passes.
- [ ] 5.3 Add one paragraph on several definitions and `--file` beside the `--signature` text in `docs/LANGUAGE-CAPABILITY-MATRIX.md`. Verify that `go test ./internal/indexer/capability` passes.
- [ ] 5.4 Run the two queries of issue #318 on the locked `psf/requests` corpus: `callers to_key_val_list` and `callees prepare_url`. Paste the output in the pull request body. Verify that the first lists a caller and the second lists `requote_uri`.
- [ ] 5.5 Run `task test`, `task eval:semantic` and `task lint`. Verify that all three pass.
- [ ] 5.6 Add the changelog fragment: `task changie -- new --interactive=false -k Fixes -b "callers, callees and impact answer for every definition of a name and say which definitions exist. The new --file flag selects the definitions in one file."`

## Workflow follow-up

- Run the implementation review with `/sb:openspec-review same-name-query-targets implementation`.
- Archive the change with `/opsx:archive` in the implementation pull request.
