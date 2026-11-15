---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-15T17:26:37+00:00
head: 335586acea00eb53d960a07abc35c40a82a9a353
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

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| inv-1 | invariants | should-fix | deviates-design | D1 moves IsDataKind to goextract to keep the rule that production internal/query reaches only goextract and nodeid among indexer packages, but the depclosure test forbids only the internal/indexer root, so an import of internal/indexer/resolve from internal/query would pass. | design.md > D1. Two new fields on DefinitionDetail — "The test enforces only the part of that rule that forbids the `internal/indexer` root."; scope basis: design.md > D1. Two new fields on DefinitionDetail — "This change does not extend the test." | In the T-01-18 production subtest, fail for any dependency under internal/indexer/ other than internal/indexer/goextract and internal/indexer/nodeid, and keep the root check. |
| inv-2 | invariants | should-fix | in-change | The Accessed by block hides on the entry count after node-less entries are dropped, and no test passes a list whose only entries lack a node, the edge where the raw list is non-empty and the shown list is empty. | web/src/lib/components/browse/NeighborsPanel.svelte:72 — "access.flatMap((a) => (a.node ? [{ node: a.node, role: a.access }] : []));" | Add a case to web/tests/neighbors-panel.test.ts with accessedBy: [noNode] that expects queryByTestId('neighbors-accessed-by') to be null. |
| fit-1 | fit | suggestion | deviates-design | Issue #274, the follow-up the proposal names, must move the new reverseEdgeIndex out of internal/query, where the indexer cannot reach it. | internal/query/traverse.go:37 — "func reverseEdgeIndex(r graphstore.Reader, keep func(*schema.Edge) bool) (map[string][]*schema.Edge, error) {"; scope basis: design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "so issue #274 must move `reverseEdgeIndex` to a package that both import" | Either place reverseEdgeIndex now in a package that both internal/query and internal/indexer import, or leave it and keep the move on #274. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A method reads and writes a field | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/a method reads and writes a field | covered | pass |
| Readers in two files | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/readers in two files | covered | pass |
| Module-level code writes a variable | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/module-level code writes a variable | covered | pass |
| A variable holds a function | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/a variable holds a function | covered | pass |
| Nothing accesses a constant | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/nothing accesses a constant | covered | pass |
| A field and a method share a name | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/the data lines appear only for data symbols | covered | pass |
| A function keeps its output | internal/cli/data_symbol_queries_test.go: TestNodeShowsDataAccess/a function keeps its output | covered | pass |
| The graph UI asks for a field | internal/uiserver/handlers_test.go: TestGetNodeDetailDataAccess/the graph UI asks for a field | covered | pass |

### Not run

None.

## Verification runs

- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/a method reads and writes a field' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/readers in two files' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/module-level code writes a variable' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/a variable holds a function' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/nothing accesses a constant' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/the data lines appear only for data symbols' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestNodeShowsDataAccess/a function keeps its output' ./internal/cli`
- exit 0: `go test -count=1 -v -run 'TestGetNodeDetailDataAccess/the graph UI asks for a field' ./internal/uiserver`

## Dropped

None.

## Lens notes

- approach: Requirement 'Node lists the readers and writers of a data symbol' is produced by D1, D2 and D3: buildAccessIndex in internal/query/traverse.go, fillDataLines and accessLess in internal/query/detail.go.
- approach: Requirement 'Node lists the initializer value of a data symbol' is produced by D4: fetchTargets with goextract.IsInitBinding in internal/query/node.go and detail.go.
- approach: Requirement 'The data lines appear only for data symbols' is produced by D5: TrailRows gates the data rows on goextract.IsDataKind, RenderNode prints every row, renderNodeSection and writeNodeSection skip empty rows.
- approach: Requirement 'The graph UI carries the data lines' is produced by D6: ui.proto fields 6/7 and 10/11, dataAccessToProto in internal/uiserver/handlers.go, browse-state.ts and the neighborList snippet of NeighborsPanel.svelte.
- approach: Cited facts checked at head: file nodes have Name = relPath and StartLine 1 in every extractor; UnknownWrite carries access=write; edgeKey is (source, kind, target); withMergedAccess merges roles; the five inline binding=init checks now call IsInitBinding; IsDataKind lives only in goextract; T-01-18 allows goextract and nodeid; internal/indexer/sync.go keeps its own reverse index for the test import cycle; uiMultiDefCap is 20; transportSendMaxBytes is 16 MiB; the Taskfile has proto:gen, proto:drift, golden:regen and test:golden.
- approach: The D2 rejected alternative holds: the non-detail callers of BuildReverseAdjacency are callers, impact, affected (traverse.go), seeding.go, expand.go and explore (detail.go), and each reads calls edges only.
- approach: D5's byte-for-byte claim for function sections holds against the unchanged serilog go-node-multi.json golden, where a multi-definition section with calls and no callers prints no Called by line.
- approach: The design has no Open Questions and no migration plan, so the answered-open-question and no-rollback classes do not apply.
- fit: The scan loop of reverseEdgeIndex (internal/query/traverse.go:37) and the indexer's buildReverseAdjacency (internal/indexer/sync.go:823) remain two copies. design.md D2 and proposal.md Out of scope name the copy, and issue #274 tracks its collapse, so it is not raised as a duplication finding.
- fit: The inline binding=init checks are collapsed: goextract.IsInitBinding (internal/indexer/goextract/types.go:297) is the only remaining comparison of MetaBinding with BindingInit in production code. IsDataKind has one definition, in goextract, and internal/query imports no resolve or symbols package.
- fit: TrailRows and TrailHeader are the one source of the trail labels, the data-kind check and the access suffix for both the markdown view and the styled view in internal/cli/present/node.go. The styled view holds no copy of them.
- fit: NeighborsPanel.svelte renders callers, callees and the two data lists through one neighborList snippet. The blast radius section stays separate, as design.md D6 states.
- fit: The codegraph_node tool description (internal/mcp/tools.go:398) and internal/mcp/resources/node.md still say that node shows a symbol's signature, calls, and callers, with no mention of the data lines. This is outside the fit lens and is recorded for the controller.
- fit: Seven web test files each keep their own BrowseTargetState stub, and the change adds accessedBy and initializedTo to each one. These are per-file test fixtures that exist before the change, so they fall under the fixture exclusion of the rulebook.
- fit: The repository adds no library-preference rows. The prompt named no path for them, so only the generic rows of the rulebook were applied.
- invariants: I1: AccessedBy and InitializedTo are nil for every kind that IsDataKind rejects \| enforced at internal/query/detail.go gatherDefinition and render_markdown.go TrailRows \| boundary test internal/query/detail_test.go 'a function candidate gets no data lines', internal/cli/data_symbol_queries_test.go 'a function keeps its output'
- invariants: I2: AccessedBy holds one entry per stored source \| enforced at internal/graphstore/keys.go edgeKey(src, kind, dst) \| boundary test TestAccessOrderIgnoresEdgeOrder (count equals present sources)
- invariants: I3: AccessedBy is ordered by file, name, line, then signature, kind and role, never by node id \| enforced at internal/query/detail.go accessLess via locationLess \| boundary test detail_test.go file-before-name, by-name, by-line, by-role subtests and the rapid property test
- invariants: I4: a module-level source prints at line 1 \| enforced at the extractors (file nodes set StartLine: 1) \| boundary test detail_test.go 'module-level code is listed by the file path' and the CLI subtest
- invariants: I5: a dangling source or initializer target is skipped, never an error \| enforced at node.go fetchSources and fetchTargets (ErrNotFound) \| boundary test detail_test.go 'a missing source is skipped' and 'binding=init edge to a missing node'
- invariants: I6: buildAccessIndex runs only when a data kind is present, at most once per request \| enforced at detail.go buildSingleDefDetail and buildMultiDefDetail \| boundary test 'only a data symbol scans the access index' (0 and 1 on both sides)
- invariants: I7: proto field numbers stay unique and stable \| enforced at TestUIProtoFieldNumbersAreStableAndUnique (descriptor reflection) \| boundary test the six new rows
- invariants: I8: the multi-definition body budget holds with data lines \| enforced at RenderNodeMultiDef \| boundary test TestRenderNodeDataLines exact-budget case on both sides
- invariants: I9: gathered UI candidates carry data lists, others none \| enforced at handlers.go nodeDefinitionToProto \| boundary test TestGetNodeDetailDataAccess at uiMultiDefCap and uiMultiDefCap+1
- invariants: I10: Svelte data list keys (node id) are unique within a list \| enforced at the store edge key and nodeToProto Id \| boundary test neighbors-panel.test.ts 'two accessors that share file, line and name'
- invariants: I11: IsInitBinding is the same check as the inline binding=init comparisons it replaces \| enforced at goextract/types.go IsInitBinding \| boundary test existing resolve and sync tests; diff is a pure substitution, no check weakened
- invariants: No enforcement point of this change is a script, so no probe run was made.
- invariants: I12: a data list hides when it shows no entry, also after node-less entries are dropped \| enforced at NeighborsPanel.svelte neighborList snippet \| boundary test none for the all-node-less list (inv-2)
- invariants: I13: production internal/query reaches only goextract and nodeid among indexer packages \| enforced at depclosure T-01-18 for the root only \| boundary test none for internal/indexer/resolve (inv-1)
- invariants: The stored access role is not validated against read, write and read_write at query time; every writer found (resolver.go, collapse.go, bindings.go, goextract types.go:322) uses the goextract constants, so this is not raised.
- test-coverage: Task 5.1 (changelog fragment) is unchecked in tasks.md, though an untracked fragment file exists; not a scenario matter.
- test-coverage: The browse-view clause of the graph UI requirement is tested in web/tests/neighbors-panel.test.ts and browse-state.test.ts; the web suite was not run by this lens.
- test-honesty: Read TestNodeDataAccess, TestAccessOrderIgnoresEdgeOrder, TestRenderNodeDataLines, TestRenderNodeDataLinesStrippedEqualsMarkdown, TestNodeShowsDataAccess, TestGetNodeDetailDataAccess, the MCP balance check and the new neighbors-panel and browse-state cases. Each asserts the spec's THEN against code under test.
- test-honesty: The present test compares stripped styled output with query.RenderNode through an independent helper, and adds positive checks for the entry lines.
- test-honesty: The skips in the test files are not on lines of this diff.
- test-honesty: The golden JSON and web/build files, and the one-line stub field additions in web tests, were not read as tests.

## Plan review (2026-11-15)

Verdict approve-with-notes at head 2bc14b359a1790cf495db684261a1a98e2296617.

### Findings

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| app-1 | approach | should-fix | beyond-scope | D1 widens the T-01-18 subtest so that it forbids every internal/indexer package except goextract and nodeid. No requirement, non-goal or risk of this change needs that wider check. Moving IsDataKind already keeps production internal/query off resolve. | design.md > D1. Two new fields on DefinitionDetail — "The T-01-18 subtest is extended to enforce the whole rule."; scope basis: proposal.md > Impact — "`IsDataKind` moves from `internal/indexer/resolve` to `internal/indexer/goextract`, which also gains `IsInitBinding`." | Either name in design.md the requirement, non-goal or risk that the wider T-01-18 check serves, or drop the test extension from D1 and task 1.1 and leave it to a separate change. |
| app-2 | approach | should-fix | in-change | D2 justifies reverseEdgeIndex with a repository convention that a loop is shared once it has a third copy. No such rule is written in openspec/config.yaml or the repository docs. traverse.go already holds more than two IterateEdges("") scan loops with no shared loop (buildContainsIndex, setsOfMembers, FileGraph's second scan). | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "the repository shares a loop once it has a third copy" | Replace the cited convention in D2 with the actual reason for the shared loop, or cite the place where the rule is recorded. |
| fit-2 | fit | should-fix | in-change | D6 gives the shared neighbor-list snippet a key function so that callers and callees keep the file:line:name key while the two data lists key by node id. Both kinds of list hold Node messages that carry an id, so one panel gets two key mechanisms for the same entry type, and the old key keeps the duplicate-key fault that D6 names for the data lists. | design.md > D6. UI wire, state and panel — "The two data lists key by node id, because two sources can share file, line and name, and Svelte rejects a keyed list wi" | Let the snippet key every Node entry by node.id and drop the key-function parameter. Keep entryKey for the test ids and for the blast-radius Location list, which has no id. |
| fit-3 | fit | should-fix | in-change | writeSingleDef and writeNodeSection mirror RenderNode and renderNodeSection minus markdown syntax, but they leave out the owner and language lines of a field. D5 edits both mirrors and their contract test and keeps that drift, and task 2.1 picks contract subtests that use no field. | internal/query/render_markdown.go:148 — "for _, line := range fieldFacts(d) {" | Print the owner and language lines of a field in writeSingleDef and writeNodeSection, after the location, as fieldFacts does, and add a field subtest to TestRenderNodeStrippedEqualsMarkdownContract. |
| inv-1 | invariants | should-fix | in-change | The rule that a non-data section never changes is enforced per candidate in buildMultiDefDetail once the shared access index exists. No planned test passes a function that is itself the target of an access edge next to a data candidate. The init binding gives a function exactly such an incoming read edge, and the size method in the fixture has none. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "If any candidate is a data kind, `buildMultiDefDetail` calls it once and shares it across candidates" | Add a case to TestNodeDataAccess. It builds a multi-definition name that has a data candidate and a function candidate, and the function is the target of a references edge with access read, for example from a `binding=init` edge. Assert that the function's DefinitionDetail has nil AccessedBy and InitializedTo, and that its rendered section has no data lines. |
| inv-2 | invariants | should-fix | in-change | Every ui.proto field number is pinned by the fixture of TestUIProtoFieldNumbersAreStableAndUnique, which fails on a declared field that has no entry. No task adds entries for the six new fields (DataAccess.node and access, NodeDefinition 6 and 7, GetNodeDetailResponse 10 and 11) or raises the pinned fixture length. Until it does, nothing pins the numbers that D6 promises old clients. | internal/uiserver/readonly_test.go:682 — "t.Fatalf("descriptor declares field %s, but no fixture entry covers it — the fixture must cover every field of every mes" | In task 3.1, add the six entries to uiProtoFieldNumbers in internal/uiserver/readonly_test.go. Add a chained length constant that is the previous one plus 6, and state that TestUIProtoFieldNumbersAreStableAndUnique passes. |
| inv-3 | invariants | should-fix | in-change | The extended T-01-18 subtest has two sides, allowed (goextract, nodeid) and forbidden (any other internal/indexer package). The plan keeps only the inspected-package count. Nothing shows that the matcher fires on the forbidden side, or that the dependency list holds an allowed indexer package, so a matcher that never matches an indexer subpackage would pass. | internal/depclosure/dependency_closure_test.go:207 — "if depSet(p)[indexerRoot] {" | In task 1.1, put the allowlist predicate in a function. Give it a table with internal/indexer/resolve and internal/indexer/goextractx (rejected), and goextract and nodeid (accepted). Also assert that production internal/query depends on internal/indexer/goextract, so the dependency list holds an indexer package that the rule can match. |
| inv-4 | invariants | should-fix | in-change | The styled view decides the trail header with its own condition, separate from renderNodeSection. Task 2.1 gives that condition one 'MultiDef data' subtest, while the plain view gets three edge cases. A styled condition that leaves out InitializedTo passes a case with access entries. | internal/cli/present/node.go:119 — "if len(calls) > 0 \|\| len(calledBy) > 0 {" | Split 'MultiDef data' in task 2.1 into three sections that mirror cases 3 to 5 of task 1.6: only an access entry, only an initializer entry, and four empty lists. Compare each with query.RenderNodeMultiDef. |
| fit-1 | fit | note | in-change | D2 puts the new shared scan loop reverseEdgeIndex in internal/query, so issue #274, the follow-up the proposal names, must move it again, while the indexer keeps its own copy of the same loop. internal/graphstore, which holds Reader and is imported by both packages, could hold it today. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "so issue #274 must move `reverseEdgeIndex` to a package that both import. It stays in `internal/query` until then" | Either place reverseEdgeIndex in a package both import (internal/graphstore takes a Reader and returns schema edges) and route internal/indexer/sync.go buildReverseAdjacency through it in this change, or keep D2 as is and record in tasks.md that #274 moves reverseEdgeIndex. |

### Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A method reads and writes a field | 2.2, 2.3 | covered |  |
| Readers in two files | 2.2, 1.2 | covered |  |
| Module-level code writes a variable | 2.2, 1.2 | covered |  |
| A variable holds a function | 2.2, 1.5 | covered |  |
| Nothing accesses a constant | 2.2, 1.6 | covered |  |
| A field and a method share a name | 2.2, 1.3, 1.6 | covered |  |
| A function keeps its output | 2.2, 4.1 | covered |  |
| The graph UI asks for a field | 3.2, 3.3, 3.4 | covered |  |

### Not run

Plan phase: no verification runs.
