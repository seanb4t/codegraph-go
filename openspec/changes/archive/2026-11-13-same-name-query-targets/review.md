---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-13T19:56:30+00:00
head: 46748ae2699fbff0bac9e3677c48c354f2f7457a
lenses:
  spec-lint: not-applicable
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: ok
  invariants: ok
---
# Review

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | D4 says the three new sort keys apply only to a result over several start nodes and that the affected command keeps its old row order byte for byte. sortLocations applies the definition, signature and kind keys to every result, and Affected sorts its affectedTests with it, so affected rows that tie on file, name and line are now ordered by kind. design.md records no deviation for affected, and the same paragraph later says a query over one definition can change its order, which contradicts its own first claim. | design.md > D4: The order of the rows — "They apply only to a result over several start nodes, one that spans definitions or overloads. A query over one definiti" | Either keep Affected (and one-start-node results, if intended) on the old three-key comparator, or rewrite the D4 sentence to say the keys apply to every result, affected included, and name the affected tie order as a changed output. |
| fit-1 | fit | should-fix | The change adds the shared test helper testtree.Write, which writes a map of files with MkdirAll and WriteFile, while the repository already has the same loop in wtindex writeFiles and in the wtindex benchmark fixture, which the change leaves as they are. | internal/testtree/testtree.go:16 — "func Write(t testing.TB, files map[string]string) string {" | Give testtree a root-taking form (for example WriteAt(t, root, files), with Write calling it on t.TempDir()) and replace wtindex writeFiles and the loop of the benchmark fixture with it. |
| inv-1 | invariants | should-fix | The row tie-break compares definition labels as strings, so a row for lib.py:12 sorts before a row for lib.py:5, while the definitions list sorts the same two by numeric line. No test passes two definitions of one file whose lines differ in digit count; the property test draws lines 1 and 2 only. | internal/query/traverse.go:593 — "return locs[i].Definition < locs[j].Definition" | Add a test where one caller calls two definitions in one file at lines 5 and 12 and assert the row order. If rows should follow the definitions list, compare the definition by file and numeric line (for example by the index of the definition in tg.definitions) instead of by the label string, and widen the property test's line range past 9. |
| inv-2 | invariants | should-fix | Impact still composes dispatch siblings into its walk, but TestImpact_DispatchTraversal now starts from both Handle methods, so B.Handle reaches Caller by its own edge and the test passes with the dispatch composition removed. No other test exercises impact dispatch. | internal/query/traverse_test.go:345 — "got, err := e.Impact("Handle", 1)" | Give A.Handle and B.Handle a FilePath in dispatchFixtureNodesEdges and call ImpactSelected with the file of A.Handle, so the walk starts from A.Handle alone and Caller is reachable only through the dispatch composition. Keep the union case as a second assertion. |
| inv-3 | invariants | should-fix | The spec requires --file to ignore letter case. filterByFile lowercases both sides, but no test of callers, callees, impact, the MCP tools or node passes a value whose case differs from the stored path. | specs/graph-queries/spec.md > Requirement: A file selects the definitions in it — "The comparison SHALL ignore letter case and SHALL treat `\\` as `/`, as `node --file` does." | Extend TestCalleesWithABackslashFilePath, or add a sibling, with a value such as `APP\\Models.PY` and assert that only the callees of src/app/models.py are listed. |
| inv-4 | invariants | should-fix | D4 states that the affected command keeps its old row order byte for byte, but sortLocations now breaks every file, name and line tie by kind, and Affected sorts its rows with sortLocations. Tied affected rows of two kinds move from discovery order to kind order, and no test pins either order. | design.md > D4: The order of the rows — "A query over one definition and the `affected` command keep their old row order byte for byte." | Either restrict the three new keys to results over several start nodes, as D4 first says, or correct D4 to name affected under the same one-time tie exception, and add a test that pins the order of two tied affected rows. |
| inv-6 | invariants | should-fix | D7 states that the row limit never shortens the list of definitions. The result takes Definitions from the targets and caps only the rows, but no test runs a query that spans definitions with a limit below the row count and asserts that every definition is still listed. | design.md > D7: Limits and frozen outputs — "The definitions line is not subject to the row limit, so the user still sees how many definitions exist and can select w" | Add a test in internal/query/definitions_test.go that calls CallersSelected on two definitions with three caller rows and limit 1, and asserts one row and two entries in Definitions. |
| fit-2 | fit | note | definitionsParagraph and emptyDefinitions are the same wrapper of DefinitionsLine and differ only in the text around the line. | internal/query/render_results.go:176 — "return line + "\\n\\n"" | Keep one function that takes the prefix and the suffix around the line, or leave both and accept the two names. |
| inv-5 | invariants | note | The file selection partitions the rows of the union for callees and impact as well as for callers, an invariant over an unbounded graph domain; TestFileSelectionPartitionsTheUnion checks Callers only. | internal/query/definitions_test.go:953 — "part, err := e.CallersSelected(Selector{Symbol: "target"}, f, 0)" | Extend TestFileSelectionPartitionsTheUnion in internal/query/definitions_test.go (pgregory.net/rapid) to compare CalleesSelected rows the same way. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two functions of one name in two files | internal/query/definitions_test.go: TestCalleesOfANameWithTwoDefinitions | covered | pass |
| A typing stub and the real function | internal/query/definitions_test.go: TestCallersOfAStubAndTheRealFunction | covered | pass |
| A method and a test helper of one name | internal/query/definitions_test.go: TestCalleesOfAMethodAndATestHelper | covered | pass |
| Impact over two definitions | internal/query/definitions_test.go: TestImpactOverTwoDefinitions | covered | pass |
| An overload set and a definition outside it | internal/query/definitions_test.go: TestCallersOfASetAndADefinitionOutsideIt | covered | pass |
| The definitions are listed | internal/cli/query_cli_test.go: TestCallersTextListsTheDefinitions | covered | pass |
| A query that spans definitions and finds no result | internal/cli/query_cli_test.go: TestCallersTextOfNoResultListsTheDefinitions | covered | pass |
| JSON output over several definitions | internal/query/definitions_test.go: TestJSONOfANameWithTwoDefinitionsHasDefinitions | covered | pass |
| One overload set alone | internal/query/definitions_test.go: TestJSONOfOneOverloadSetHasNoDefinitionKeys | covered | pass |
| A function and a package of one name | internal/query/definitions_test.go: TestAFunctionAndAPackageOfOneName | covered | pass |
| A caller names the definition it calls | internal/query/definitions_test.go: TestCallerNamesTheDefinitionItCalls | covered | pass |
| A signature that matches in two files | internal/query/definitions_test.go: TestSignatureThatMatchesInTwoFilesSpansDefinitions | covered | pass |
| An ambiguous callee in a query that spans definitions | internal/query/definitions_test.go: TestAmbiguousCalleeInASpanningQueryNamesItsDefinition | covered | pass |
| An ambiguous caller in a query that spans definitions | internal/query/definitions_test.go: TestAmbiguousCallerInASpanningQueryHasNoDefinition | covered | pass |
| Impact reaches one symbol from two definitions | internal/query/definitions_test.go: TestImpactReachesOneSymbolFromTwoDefinitions | covered | pass |
| One caller of two definitions | internal/query/definitions_test.go: TestOneCallerOfTwoDefinitionsIsOrderedByDefinition | covered | pass |
| Tied results of one definition | internal/query/definitions_test.go: TestTiedResultsOfOneDefinition | covered | pass |
| One definition in JSON | internal/query/definitions_test.go: TestJSONOfANameWithOneDefinitionHasNoNewKeys | covered | pass |
| One definition in text | internal/cli/query_cli_test.go: TestCallersTextListsTheDefinitions (exact output of callers load) | covered | pass |
| Callees of the definition in one file | internal/query/definitions_test.go: TestCalleesWithFileSelectsOneDefinition; internal/cli/query_cli_test.go: TestCalleesFileFlag | covered | pass |
| A file named by the end of its path | internal/query/definitions_test.go: TestCalleesWithTheEndOfAFilePath | covered | pass |
| A file with no definition of the name | internal/query/definitions_test.go: TestCallersWithAFileThatMatchesNothingListsEachFileOnce; internal/cli/query_cli_test.go: TestCalleesFileFlag | covered | pass |
| A file and a signature | internal/query/definitions_test.go: TestCallersWithFileAndSignature | covered | pass |
| An MCP callers call on a name with two definitions | internal/mcp/definitions_test.go: TestCallersToolOnANameWithTwoDefinitions | covered | pass |
| An MCP callees call with a file | internal/mcp/definitions_test.go: TestCalleesToolWithAFile | covered | pass |
| Callers of a name with two definitions | internal/mcp/definitions_test.go: TestCallersToolTableHasADefinitionColumn | covered | pass |
| No caller of a name with two definitions | internal/mcp/definitions_test.go: TestCallersToolOfNoCallerListsTheDefinitions | covered | pass |
| An impact row that the walk reached | internal/mcp/definitions_test.go: TestImpactToolRowReachedByTheWalkHasNoDefinition | covered | pass |
| Callers of a name with one definition | internal/mcp/definitions_test.go: TestCallersToolTableOfOneDefinitionHasThreeColumns | covered | pass |
| The input schema of the three tools | internal/mcp/tools_schema_drift_test.go: TestDefinitionToolsDeclareOptionalFile | covered | pass |

### Not run

None.

## Verification runs

- exit 0: `go test -count=1 -run '^(TestCallersTextListsTheDefinitions)$' ./internal/cli`
- exit 0: `go test -count=1 -run '^(TestCallersTextListsTheDefinitions)$' ./internal/cli # covers: internal/cli/query_cli_test.go: TestCallersTextListsTheDefinitions (exact output of callers load)`
- exit 0: `go test -count=1 -run '^(TestCallersTextOfNoResultListsTheDefinitions)$' ./internal/cli`
- exit 0: `go test -count=1 -run '^(TestCalleesToolWithAFile)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestCallersToolOfNoCallerListsTheDefinitions)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestCallersToolOnANameWithTwoDefinitions)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestCallersToolTableHasADefinitionColumn)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestCallersToolTableOfOneDefinitionHasThreeColumns)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestImpactToolRowReachedByTheWalkHasNoDefinition)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestDefinitionToolsDeclareOptionalFile)$' ./internal/mcp`
- exit 0: `go test -count=1 -run '^(TestAFunctionAndAPackageOfOneName)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestAmbiguousCalleeInASpanningQueryNamesItsDefinition)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestAmbiguousCallerInASpanningQueryHasNoDefinition)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCalleesOfAMethodAndATestHelper)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCalleesOfANameWithTwoDefinitions)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCalleesWithFileSelectsOneDefinition|TestCalleesFileFlag)$' ./internal/cli ./internal/query # covers: internal/query/definitions_test.go: TestCalleesWithFileSelectsOneDefinition; internal/cli/query_cli_test.go: TestCalleesFileFlag`
- exit 0: `go test -count=1 -run '^(TestCalleesWithTheEndOfAFilePath)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCallerNamesTheDefinitionItCalls)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCallersOfASetAndADefinitionOutsideIt)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCallersOfAStubAndTheRealFunction)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestCallersWithAFileThatMatchesNothingListsEachFileOnce|TestCalleesFileFlag)$' ./internal/cli ./internal/query # covers: internal/query/definitions_test.go: TestCallersWithAFileThatMatchesNothingListsEachFileOnce; internal/cli/query_cli_test.go: TestCalleesFileFlag`
- exit 0: `go test -count=1 -run '^(TestCallersWithFileAndSignature)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestImpactOverTwoDefinitions)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestImpactReachesOneSymbolFromTwoDefinitions)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestJSONOfANameWithOneDefinitionHasNoNewKeys)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestJSONOfANameWithTwoDefinitionsHasDefinitions)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestJSONOfOneOverloadSetHasNoDefinitionKeys)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestOneCallerOfTwoDefinitionsIsOrderedByDefinition)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestSignatureThatMatchesInTwoFilesSpansDefinitions)$' ./internal/query`
- exit 0: `go test -count=1 -run '^(TestTiedResultsOfOneDefinition)$' ./internal/query`

## Dropped

None.

## Lens notes

- approach: Every requirement of both spec deltas maps to a decision that the code at head produces: symbolTargets (internal/query/overloads.go) unions definitionsOf with the overload_set nodes, computes spans after both filters, builds Definitions and labels via DefinitionLabel; CallersSelected/CalleesSelected/ImpactSelected take the file parameter; the CLI and MCP surfaces pass it; the markdown table takes the Definition flag from the result.
- approach: Cited facts checked and holding at head: definitionsOf/nodesNamed/enumerateSymbolDefs, filterByFile shared with narrowNodeMatches, nodeMultiDefListCap = 20, addWorkersFlag pattern, -l on callers/callees and -d on impact, uiv1.Location has no definition field, UI handlers call the unselected wrappers, gocapture captures explore and node only, -update-plain-goldens flag, test/wireoracle/cmd/wireoracle exists, TestImpact_DispatchTraversal asserts NodeCount 3, package node Name is the last path segment with QualifiedName the import path.
- approach: resolveSymbolNode and the empty-file branch of resolveNodeForDetail are gone at head, as design Context says the implementation removes them.
- approach: diff-files.txt lists internal/indexer/symbols/candidate.go and internal/indexer/sync_test.go although the proposal puts the index out of scope. Without git I could not see the hunk; candidate.go at head shows no behavior tied to this change, so it is likely a comment edit. The controller may want to confirm.
- approach: Tied rows are ordered by the definition label as a string, so a.go:10 sorts before a.go:9, while the definitions list orders by numeric line. This matches the spec wording 'by the name of their definition' and is not a finding of this lens.
- fit: D1 to D6 fit their named patterns: symbolTargets reuses nodesNamed and definitionsOf, filterByFile is the one file test that narrowNodeMatches and symbolTargets share, addSignatureFlag and addFileFlag follow addWorkersFlag in their own file, and DefinitionsHeading and CapDefinitions are shared with the node renderers.
- fit: The MCP args repeat the file and signature help text in struct tags; Go struct tags cannot reference a constant, so this is not reported.
- fit: The plain and styled renderers of callers, callees and impact keep their sanctioned plain/styled mirror; each calls query.RowSuffix and query.DefinitionsLine, so the new text has one source.
- fit: D8 holds: internal/uiserver/handlers.go calls Callers, Callees and Impact with no file and needs no change.
- fit: The prompt names no path for repository library-preference rows; only the generic rows were applied.
- invariants: I1: The start nodes of callers, callees and impact are every non-set node of the name plus each overload_set of the name, never one picked node \| enforced at symbolTargets, internal/query/overloads.go:56 \| boundary test TestCalleesOfANameWithTwoDefinitions (two), TestJSONOfANameWithOneDefinitionHasNoNewKeys (one), existing not-found path (zero)
- invariants: I2: spans is true exactly when the kept definitions are not all members of one set \| enforced at spansDefinitions, internal/query/overloads.go:133 (len<2 guard) \| boundary test TestJSONOfOneOverloadSetHasNoDefinitionKeys (one set), TestCallersOfASetAndADefinitionOutsideIt, TestTwoOverloadSetsSpanDefinitions, TestCalleesWithFileSelectsOneDefinition (filter down to one)
- invariants: I3: A name with one definition emits no definitions and no definition key \| enforced at omitempty on Location.Definition and the three Definitions fields, set only when spans \| boundary test TestJSONOfANameWithOneDefinitionHasNoNewKeys, plain goldens
- invariants: I4: Row order never depends on node ids \| enforced at sortLocations (traverse.go:579) and sortDefinitions (overloads.go:151) \| boundary test TestQueryResultDoesNotDependOnNodeIDs (rapid, two opposite id schemes, drawn limit); lines drawn 1..2 only, see inv-1
- invariants: I5: The definitions line shows at most 20 labels and then +K more \| enforced at CapDefinitions, render_results.go \| boundary test TestCallersTextListsTheDefinitions and TestRenderCallersMarkdown at 20 and 21
- invariants: I6: --file is a hard filter; no match fails and lists each file once, also with no file at all \| enforced at symbolTargets file branch and definitionFiles (slices.Compact after sort) \| boundary test TestCallersWithAFileThatMatchesNothingListsEachFileOnce, TestFileWhenNoDefinitionHasAFile, TestFileThatMatchesNothingWithASignatureListsEveryFile; letter case untested, see inv-3
- invariants: I7: A set whose members are all filtered out is not a start node \| enforced at symbolTargets keptSets check \| boundary test TestFileThatExcludesASetDropsItsAmbiguousCallers
- invariants: I8: A caller of two definitions appears once per definition; an ambiguous caller of two sets appears once \| enforced at the callers row key, traverse.go CallersSelected \| boundary test TestOneCallerOfTwoDefinitionsIsOrderedByDefinition, TestAmbiguousCallerOfTwoSetsIsListedOnce, TestTwoCallersWithEqualRowsAreBothListed
- invariants: I9: The table of one definition or one set is byte-identical to before \| enforced at renderLocationTableColumns, read line by line against the removed branches: header, separator and row bytes match \| boundary test TestCallersToolTableOfOneDefinitionHasThreeColumns, wire oracle transcripts
- invariants: I10: The --file and --signature flags register without a shorthand clash \| enforced at addFileFlag (-f); no persistent -f flag exists on the root \| boundary test TestCalleesFileFlag runs the real command tree
- invariants: I11: The three MCP tool inputs declare an optional string file \| enforced at CallersArgs, CalleesArgs, ImpactArgs omitempty tags \| boundary test TestDefinitionToolsDeclareOptionalFile
- invariants: No enforcement point of this change is a script; all are Go functions in internal packages, so no probe was run. The string order claim of inv-1 follows from Go byte-wise string comparison ('1' < '5').
- invariants: resolveNodeForDetail lost its empty-file branch; its one production caller (detail.go:207) guards with file != "", so no check that a caller relies on is dropped.
- test-coverage: Not read: the rapid property tests TestQueryResultDoesNotDependOnNodeIDs and TestFileSelectionPartitionsTheUnion (task-explained, not orphans), and the markdown renderer tests of tasks 4.2 and 4.3.
- test-coverage: The MCP file input is exercised only through codegraph_callees; the pass-through in callers and impact is covered only by the schema test.
- test-honesty: Read internal/query/definitions_test.go in full, the MCP definitions_test.go, the schema drift test and the CLI definition tests. No skips, no missing assertions, no mock-echo or tautology found; id-independence and file-partition property tests carry positive assertions.
- test-honesty: Not read in depth: edits to engine_test, explore_test, files_status_test, render_*_test, sync_test, overloads tests, traverse_test edits, and the regenerated golden files.

## Plan review (2026-11-13)

Verdict approve-with-notes at head 3c126455a9adecc1dd5634896f478793613fb280.

### Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | The Context says resolveNodeForDetail calls resolveSymbolNode for `node` with no file. At head buildNodeDetail calls resolveNodeForDetail only when a file is given, so `node` with no file goes through enumerateSymbolDefs, and the resolveSymbolNode branch of resolveNodeForDetail has no production caller. After D1 removes the symbolTargets branch, resolveSymbolNode is unreachable from production code. | design.md > Context — "`resolveSymbolNode` has one more caller: `resolveNodeForDetail` in `internal/query/node.go`, for `node` with no file. Th" | Correct the Context bullet. State that resolveSymbolNode is reached after D1 only through the file=="" branch of resolveNodeForDetail, which no production path takes, and decide whether that code is removed or kept. Restate the non-goal on resolveSymbolNode to match. |
| app-2 | approach | should-fix | D7 says a one-definition capture may change only for the D4 row order and the schema text. The wire oracle also captures resources-read transcripts of the callers, callees and impact resource docs, which task 6.4 edits, so those transcripts change too and the stated exceptions do not cover them. | design.md > D7: Limits and frozen outputs — "A capture for a name with one definition must not change, except for the row order of an overloaded query (D4) and the s" | Add the resources-read-callers, resources-read-callees and resources-read-impact transcripts, and the resource text, to the D7 list of expected changes. |
| app-3 | approach | should-fix | D4 claims rows tie on file, name and line only inside an overloaded query. An overload_set node copies Name, FilePath and StartLine from its first member. So a callees query whose start node calls both a member (resolved) and the set (ambiguous) gets two rows that tie on all three keys in any query. They also tie on the new definition and signature keys, because both rows take those values from the same start node. Their order still follows the edge scan by target id, so the order of the results still depends on node ids. | design.md > D4: The order of the rows — "Rows that tie today tie only inside an overloaded query, where their order follows the node ids." | Correct the claim in D4. Add a final key that separates a set row from its member row, for example Ambiguous or Kind, so no tie is left to the id order. Either name that key in the ordering requirement or record the residual tie as accepted. |
| fit-1 | fit | should-fix | D6's hard --file filter copies the file test inside narrowNodeMatches instead of sharing it. The --file flags of sibling commands would then keep two copies of one match rule. | design.md > D6: `--file` filters the start nodes — "`symbolTargets` keeps the definitions whose `FilePath` contains the value, compared through `normalizeNarrowPath`. This " | In D6, extract the loop at node.go:254-261 into one function, for example filterByFile(matches, hint) []*schema.Node, which returns the hard result. narrowNodeMatches keeps its fallback around that function, and symbolTargets calls it directly. |
| fit-2 | fit | should-fix | After D1, the signature branch of symbolTargets filters the output of enumerateSymbolDefs just as narrowBySignature does. D6 rewrites that branch but does not name narrowBySignature, so the two copies of the filter and its not-found error stay. | internal/query/overloads.go:93 — "if !matchesSignature(n, sel.Signature) {" | In D1/D6, state that symbolTargets calls narrowBySignature on the definitions and then adds the sets of the kept members. Note in the design that the error then lists compacted signatures, as node already does, and treat that as a frozen-output diff in D7. |
| fit-3 | fit | should-fix | The D6 flag helper covers callers, callees and impact only. node keeps a fourth copy of the identical --signature text, so the design's claim that the flag text exists once is false. | design.md > D6: `--file` filters the start nodes — "One helper in `internal/cli` registers `--signature` and `--file` (short form `-f`) on the three commands, so the flag t" | Follow the addWorkersFlag pattern (one flag, its own file in internal/cli). Add addSignatureFlag, used by node, callers, callees and impact, and register --file for the three traversals beside it, since node's --file text differs on purpose. Otherwise, name node's copy in D6. |
| fit-4 | fit | should-fix | D6 passes the file as a new positional parameter, copying NodeSelected. Selector already exists to name what a traversal starts from, and Signature lives there. The design does not weigh a Selector.File field, which would leave the three signatures and their six call sites unchanged and would give the --line follow-up a place to go. | design.md > D6: `--file` filters the start nodes — "`CallersSelected`, `CalleesSelected` and `ImpactSelected` take the file as a parameter beside the `Selector`, as `NodeSe" | In D6, either add File to Selector (the selection type of the traversals) or give the reason for following NodeSelected. NodeSelected's file is a separate parameter because it also selects file mode, which the traversals do not have. |
| fit-5 | fit | should-fix | D6 runs the file filter before the signature filter. node, the closest pattern with both selectors, runs the signature filter first. The design names neither the pattern nor a reason, so on the same input the sibling commands report a different not-found error. | design.md > D6: `--file` filters the start nodes — "The filter runs before the signature filter, and `spans` is computed after both." | In D6, either use node's order (signature, then file) or name node's order and give the reason for the opposite one, including which error a user sees when both selectors match nothing. |
| fit-6 | fit | should-fix | D5 adds a second row-suffix function that the CLI text appends after OverloadSuffix. Each of the six row printers that call OverloadSuffix today would then concatenate the two suffixes in the same order. | design.md > D5: Text and table rendering — "One function builds the row suffix of a definition. In the CLI text it follows the overload suffix." | In D5, name one function, for example query.RowSuffix(l), that returns the overload suffix followed by the definition suffix. The six printers call only that function, so the order rule exists once. |
| inv-1 | invariants | should-fix | Rows of callees can still tie on every planned sort key and then keep the target-id order of the edge scan. An overload_set callee takes the file, name and start line of its first member. Within one start node both rows carry the same signature and label, so D4's keys do not separate them and the stable sort keeps edgeKey order, which is by target id. | specs/graph-queries/spec.md > Requirement: The order of the results does not depend on node ids — "Results that are equal in all three SHALL be ordered by the name of their definition, then by their signature." | Add Kind and then Ambiguous as the last keys of sortLocations in D4 and in the requirement text, so rows that differ in any output field never tie. Draw an overload_set callee beside its first member in TestQueryResultDoesNotDependOnNodeIDs (task 2.2). |
| inv-2 | invariants | should-fix | D7 relies on the definitions line telling the user how many definitions exist, but D3 caps Definitions at MaxLimit and D5 builds the line from that list. A name with more than 1000 definitions (__init__, String, New in a large repository) then prints 1000 as its count. No field and no task carries the uncapped total. | design.md > D3: The definition label — "The list is in the order of D1, and its length is capped at `MaxLimit`." | Either count the definitions before the cap and carry the total (for example a definitionCount field, omitted when spans is false), or state in D3 and the line that the count is capped. Add a test with MaxLimit and MaxLimit+1 definitions that asserts the printed count. |
| inv-3 | invariants | should-fix | The cap of the definitions line is tested only at 25 definitions. No test passes exactly 20 labels (no +0 more) or 21 labels (+1 more), the two sides of the cap. | tasks.md > 4. Text and table rendering — "Add a case to `TestCallersTextListsTheDefinitions` with 25 definitions. Assert that the line holds 20 labels and `+5 mor" | In task 4.3 use 20 and 21 definitions: assert 20 labels and no `more` for 20, and 20 labels and `+1 more` for 21. Add the same pair to a markdown renderer case in task 4.2. |
| inv-4 | invariants | should-fix | D2 names two overload sets as a case where spans is true, but no task passes two sets with no other definition. Every planned spans case has a plain definition or a signature match in two files. | design.md > D2: "Spans definitions" is a second mode beside "overloaded" — "The cases are two or more definitions with no set, a set plus a definition outside it, and two sets." | Add a test with two classes that each declare two overloads of `log` and nothing else named `log`. Assert that spans is true, the Definitions list has four entries, and each set's ambiguous caller has no definition field. |
| inv-5 | invariants | should-fix | D6 says a set whose members are all filtered out by --file is not a start node, but no planned test runs --file that excludes every member of a set. The file-and-signature test keeps the set, so a leak of the excluded set's ambiguous callers would pass. | design.md > D6: `--file` filters the start nodes — "If no member of a set stays, the set is not a start node." | Add a test where Logger.java holds log(String), log(String, Throwable) and an ambiguous caller of the set, and Audit.java holds log(String). Run callers log --file Audit.java and assert that the ambiguous caller is absent. |
| inv-6 | invariants | should-fix | The mcp-server delta requires a Definition column whenever a query spans definitions. Today renderLocationTable picks its columns from the rows. If the new Definition flag also comes from the rows, a spanning result whose rows are all ambiguous (no labels) has no Definition column. No planned table test passes that edge. | specs/mcp-server/spec.md > Requirement: A table over several definitions names the definition of each row — "The table SHALL have a Definition column with the name of each row's definition." | State in D5 that the Definition flag is len(result.Definitions) > 0, and add a markdown case where the only row is the ambiguous caller of a set beside an unrelated definition (the fixture of TestAmbiguousCallerInASpanningQueryHasNoDefinition). Assert the Definition column with an empty cell. |
| inv-7 | invariants | should-fix | The design claims that the definitions line appears only where today's answer is incomplete, and that a query over overloads keeps its output. A Java constructor has the class name, so a class with overloaded constructors has the set plus the class node. spans then turns true for every such class, although new T() is an instantiates edge and the class node adds no calls row. | design.md > Risks / Trade-offs — "The line appears only where today's answer is incomplete." | Decide in D1 or D2 how a type and its constructors of one name count (for example, a type whose only same-named nodes are its own constructors does not make spans true), and add a test with a Java class with two constructors that asserts the chosen output. |
| fit-7 | fit | note | The D5 definitions line repeats the header and the capped list with a +K more tail from RenderNodeMultiDef. Only the format of one entry differs: a label on one line here, a bullet with name and kind in node. | design.md > D5: Text and table rendering — "The line lists at most `nodeMultiDefListCap` labels and then `+K more`." | Parameter: the formatter of one entry. A shared heading function, for example definitionsHeading(n, symbol), and the cap with its tail can serve both node and the D5 line, so the wording that D5 says follows node stays the same. |
| inv-8 | invariants | note | The partition property of task 5.2 does not hold as written. A per-file run over one definition has spans false and carries no Definition label, while the no-file run labels every row. A definition with no file appears in no per-file run. | tasks.md > 5. Selection by file — "The rows of the first run equal the union of the rows of the other runs." | Keep the property with pgregory.net/rapid (as in internal/indexer/spill_roundtrip_test.go), but compare rows with Definition removed, or keyed by start definition, and give every drawn definition a file. Say so in task 5.2. |

### Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two functions of one name in two files | 1.1 | covered |  |
| A typing stub and the real function | 1.2 | covered |  |
| A method and a test helper of one name | 1.2 | covered |  |
| Impact over two definitions | 1.3 | covered |  |
| An overload set and a definition outside it | 1.3 | covered |  |
| The definitions are listed | 4.1 | covered |  |
| A query that spans definitions and finds no result | 4.1 | covered |  |
| JSON output over several definitions | 3.4 | covered |  |
| One overload set alone | 3.4 | covered |  |
| A function and a package of one name | 3.3 | covered |  |
| A caller names the definition it calls | 3.1 | covered |  |
| A signature that matches in two files | 3.2 | covered |  |
| An ambiguous caller in a query that spans definitions | 3.2 | covered |  |
| Impact reaches one symbol from two definitions | 3.1 | covered |  |
| One caller of two definitions | 2.1 | covered |  |
| One definition in JSON | 3.4 | covered |  |
| One definition in text | 4.4 | covered |  |
| Callees of the definition in one file | 5.1 | covered |  |
| A file named by the end of its path | 5.1 | covered |  |
| A file with no definition of the name | 5.1, 5.3 | covered |  |
| A file and a signature | 5.1 | covered |  |
| An MCP callers call on a name with two definitions | 6.1 | covered |  |
| An MCP callees call with a file | 6.1 | covered |  |
| Callers of a name with two definitions | 6.3 | covered |  |
| No caller of a name with two definitions | 6.3 | covered |  |
| An impact row that the walk reached | 6.3 | covered |  |
| Callers of a name with one definition | 6.3 | covered |  |
| The input schema of the three tools | 6.2 | covered |  |

### Not run

Plan phase: no verification runs.
