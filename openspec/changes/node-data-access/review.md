---
verdict: approve
phase: implementation
reviewed_at: 2026-11-15T17:41:13+00:00
head: e07fed0dc6b893547b1124f1862605fb98b3696f
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
| fit-1 | fit | should-fix | beyond-scope | The new reverseEdgeIndex loop in internal/query is the same full-scan, key-by-target loop as the indexer's buildReverseAdjacency. design.md names the copy, and issue #274 tracks its collapse outside this change. | internal/query/traverse.go:37 — "func reverseEdgeIndex(r graphstore.Reader, keep func(*schema.Edge) bool) (map[string][]*schema.Edge, error) {"; scope basis: proposal.md > Out of scope — "The indexer's own copy of the reverse edge index in `internal/indexer/sync.go`, which issue #274 tracks." | Leave it to #274. That issue moves reverseEdgeIndex to a package both import, and the indexer then passes a kinds predicate. This change does nothing more. |
| inv-1 | invariants | should-fix | deviates-design | D1 moves IsDataKind to goextract to keep production internal/query off every indexer package but goextract and nodeid; the depclosure test only forbids the internal/indexer root, so an import of internal/indexer/resolve from internal/query would pass. | design.md > D1. Two new fields on DefinitionDetail — "The test enforces only the part of that rule that forbids the `internal/indexer` root. This change keeps to the whole ru"; scope basis: design.md > D1. Two new fields on DefinitionDetail — "This change does not extend the test." | In the T-01-18 production subtest, fail for any dep under internal/indexer other than internal/indexer/goextract and internal/indexer/nodeid, and add a fixture check that the rule can match. |

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
| The graph UI asks for a field | internal/uiserver/handlers_test.go: TestGetNodeDetailDataAccess | covered | pass |

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
- exit 0: `go test -count=1 -v -run 'TestGetNodeDetailDataAccess' ./internal/uiserver`

## Dropped

None.

## Lens notes

- approach: All four requirements have producing decisions: R1 by D1-D3 (internal/query/detail.go fillDataLines, traverse.go buildAccessIndex), R2 by D4 (node.go fetchTargets with goextract.IsInitBinding), R3 by D5 (render_markdown.go TrailRows, RenderNode, renderNodeSection), R4 by D6 (ui.proto fields 6, 7, 10, 11, handlers.go dataAccessToProto, browse-state.ts, NeighborsPanel.svelte).
- approach: Every scenario THEN is producible at head: file nodes carry Name = relPath and StartLine 1 in every extractor, so a module-level write prints as main.go (main.go:1, write); UnknownWrite in goextract/types.go gives the write role; withMergedAccess in resolve/collapse.go gives read_write.
- approach: Cited facts hold at head: T-01-18 in internal/depclosure/dependency_closure_test.go allows goextract and nodeid; IsDataKind and IsInitBinding live in goextract/types.go; the five inline binding=init checks in resolve/bindings.go, resolve/collapse.go and sync_bindings.go now call IsInitBinding; the indexer keeps buildReverseAdjacency(r, kinds...) in sync.go; transportSendMaxBytes is 16 MiB and uiMultiDefCap is 20.
- approach: No code departs from a decision: BuildReverseAdjacency and BuildImplementsIndex route through reverseEdgeIndex, BuildImplementsIndex keeps its sort by source, fetchCalledBy is gone in favor of fetchSources, accessLess sorts by locationLess with the signature set and then by role, writeNodeSection takes *query.DefinitionDetail, and the styled loop keeps its own budget constants and strippedLen budget.
- approach: The multi-definition browse view renders DefinitionPicker, which shows no neighbor list of any kind, so the data lists appear only after a definition is picked. D6 scopes the browse state to single-definition mode and the requirement text does not name the mode; this matches how Callers and Callees behave today, so it is recorded here and not as a finding.
- approach: The design has no open questions and no migration plan, so the answered-open-question and no-rollback classes do not apply.
- approach: Issue #335, cited by D5 for the rejected two-list option, was not read; the rejection reason was not checked against it.
- fit: D1: IsDataKind now exists once, in internal/indexer/goextract/types.go:303. No copy is left in resolve, and internal/query reaches only goextract.
- fit: D4: every binding=init check outside tests now goes through goextract.IsInitBinding (resolve/bindings.go:30 and :78, resolve/collapse.go:102, sync_bindings.go:98 and :110, query/detail.go:208). The remaining MetaBinding writes in resolver.go:469 and collapse.go:103 set the value and do not test it.
- fit: D2: BuildReverseAdjacency, BuildImplementsIndex and buildAccessIndex all route through reverseEdgeIndex. buildAccessIndex follows the package-var seam pattern of buildReverseAdjacency (traverse.go:67).
- fit: D3: accessLess reuses locationLess and nodeLocation instead of writing a new comparator. sourceEdge is the only node-plus-edge pair type in internal.
- fit: D5: the markdown and styled views both loop over query.TrailRows and query.TrailHeader, so neither view keeps its own copy of the labels or the data-kind check. writeNodeRefs stays the sanctioned styled mirror of joinNodeRefs and formatNodeRef, and both put a non-empty suffix inside the parentheses the same way.
- fit: D6: the DataAccess message follows the BlastEntry pattern of a Node plus annotation fields. dataAccessToProto reuses nodeToProto, as nodesToProto does. NeighborsPanel now draws callers, callees and both data lists through one snippet.
- fit: The Account.java fixture is defined once, as testtree.AccountJava, and the cli, mcp and uiserver tests share it.
- fit: Not a finding (pre-existing, and the change adds no copy): web/tests keeps separate singleDefState and node builders in source-pane, source-pane-breadcrumb, source-pane-editor-link and degrade-states. This change had to add accessedBy and initializedTo to each one, plus the inline states in browse-page, browse-tracer and live-route-refetch. A shared builder under web/tests/support would make the next field addition a single edit.
- fit: Generated files (ui.pb.go, ui_pb.ts, web/build/**) and the hugo goldens were left out of the fit reading, as the exclusions direct.
- fit: fit-1 is should-fix only because the gate allows note severity just for third-copy. The rulebook's tracked-ticket downgrade applies to third copies alone, so a tracked change-vs-repo copy stays should-fix, with scope beyond-scope.
- invariants: I1: AccessedBy and InitializedTo are always nil for a kind that IsDataKind rejects \| enforced at internal/query/detail.go gatherDefinition (IsDataKind check) and render_markdown.go TrailRows \| boundary test TestNodeDataAccess 'a function candidate gets no data lines', TestGetNodeDetailDataAccess 'a field and a method named size'
- invariants: I2: A data symbol always prints both data lines in the single-definition view, also when empty \| enforced at render_markdown.go RenderNode via TrailRows \| boundary test TestNodeShowsDataAccess 'nothing accesses a constant' (HasSuffix on both empty lines), styled 'SingleDef constant'
- invariants: I3: A multi-definition section never prints an empty data line, and prints the trail header iff one of four lists has an entry \| enforced at renderNodeSection and writeNodeSection \| boundary test TestRenderNodeDataLines cases 3 to 5 and the styled MultiDef subtests
- invariants: I4: AccessedBy holds one entry per present source and never depends on edge order or node id \| enforced at detail.go accessLess (locationLess then role) and graphstore keys.go edgeKey (one edge per source, kind, target) \| boundary test TestAccessOrderIgnoresEdgeOrder (rapid, twins with same and different roles, dropped sources) plus the file, name, line and role tie subtests
- invariants: I5: Only a references edge with a non-empty access role enters the access index \| enforced at traverse.go buildAccessIndex \| boundary test 'a references edge without access is not an entry' and 'a calls edge with access is not an entry'
- invariants: I6: Only a references edge with binding=init enters InitializedTo, and a dangling target is skipped \| enforced at detail.go fillDataLines with goextract.IsInitBinding, node.go fetchTargets (WR-04) \| boundary test the three binding=init subtests of TestNodeDataAccess
- invariants: I7: A dangling access source is skipped, never an error \| enforced at node.go fetchSources \| boundary test 'a missing source is skipped' and the dropped sources of the rapid test
- invariants: I8: The access index is built at most once per request, and never when no candidate is a data kind \| enforced at detail.go buildSingleDefDetail and buildMultiDefDetail (slices.ContainsFunc) \| boundary test 'only a data symbol scans the access index' with 0 and 1 on both sides
- invariants: I9: Module-level code is listed as the file path at line 1 \| enforced at the extractors' file node (goextract.go StartLine: 1, Name: relPath) \| boundary test 'module-level code is listed by the file path' and the CLI scenario
- invariants: I10: The IsInitBinding refactor keeps the indexer checks byte-equal \| enforced at goextract/types.go IsInitBinding, the same expression as the removed inline checks \| boundary test none needed; no inline MetaBinding check remains (rg)
- invariants: I11: Production internal/query reaches no internal/indexer package but goextract and nodeid \| enforced at none for subpackages (depclosure checks only the root) \| boundary test none; holds at head, see inv-1
- invariants: I12: UI proto field numbers are new and unique \| enforced at readonly_test.go uiProtoFieldNumbers with the +6 length constant \| boundary test TestUIProtoFieldNumbersAreStableAndUnique
- invariants: I13: Only gathered multi-definition candidates carry data lists on the wire \| enforced at handlers.go nodeDefinitionToProto (gathered branch) \| boundary test 'only gathered candidates carry their entries' at uiMultiDefCap and uiMultiDefCap+1
- invariants: I14: Svelte keys of a data list are unique \| enforced at NeighborsPanel.svelte idKey plus graphstore edgeKey uniqueness \| boundary test neighbors-panel 'two accessors that share file, line and name both render'
- invariants: I15: A data section's entries count against the plain body budget \| enforced at RenderNodeMultiDef \| boundary test 'data entries move a later section to the other definitions' (fills the budget exactly, then one past)
- invariants: No enforcement point of this change is a script, so no edge-input probe ran. A section-by-section comparison of the three hugo goldens against main showed that only field sections changed, each gaining the trail header and an Accessed by line.
- invariants: InitializedTo keeps edge order (store key order, so by target id); resolver.go dataRefMetadata sets binding=init only when the ref is the whole initializer, so the list holds at most one entry and no ordering finding follows.
- test-coverage: Assertions read at the CLI test and the uiserver handler test; query, render, present, MCP and web tests were located by name and grep counts only.
- test-honesty: TestNodeDataAccess, TestAccessOrderIgnoresEdgeOrder, TestNodeShowsDataAccess, TestGetNodeDetailDataAccess and the stripped-equals-markdown property test assert specific values against the spec THEN clauses and can fail for the named reason.
- test-honesty: web browse-state test passes a stub response through loadBrowseTarget and checks the lists survive; it is a thin pass-through but matches task 3.3 and does exercise the mapping, so not reported.
- test-honesty gap: render_markdown_test.go TestRenderNodeDataLines, internal/mcp check, readonly_test.go and web tests other than neighbors-panel/browse-state were only grepped, not read in full
- test-honesty gap: Skips found in the grep are in files outside this change's added tests and were not judged as new

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
