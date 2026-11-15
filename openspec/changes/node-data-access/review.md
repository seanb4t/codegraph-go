---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-15T13:45:10+00:00
head: 71c3398493cfb12602fd332424a018c9a45bb76a
lenses:
  spec-lint: ok
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: not-applicable
  invariants: ok
---
# Review

## Findings

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| app-1 | approach | should-fix | in-change | D6 says the panel repeats one labeled list block for callers, callees and blast radius. At head the blast radius section also holds a depth input and loading, failed and loaded states, its empty text names the depth, and its section, empty and entry test ids do not share one prefix pattern with the callers block. A snippet with only heading, empty text, test-id prefix, hide flag, suffix and key cannot render that section and keep its test ids. | design.md > D6. UI wire, state and panel — "The panel repeats one labeled list block for callers, callees and blast radius today" | Say in D6 that the snippet renders only the entry list inside the blast radius section, whose heading, depth input and load states stay outside it, or keep blast radius out of the snippet, and make task 3.4 say the same. |
| fit-1 | fit | should-fix | in-change | The styled node view is a sanctioned mirror of RenderNode that claims the same sections, but it omits the owner and language lines of a field. D5 edits writeSingleDef and writeNodeSection and keeps that difference without a reason, so a field's styled view gains the data lines and still lacks the owner lines. | internal/cli/present/node.go:38 — "// RenderNode writes a lipgloss-styled rendering of d — the SAME sections," | Since writeNodeSection now takes the *query.DefinitionDetail, print the owner and language lines of a field in writeSingleDef and writeNodeSection too, and add a field subtest to TestRenderNodeStrippedEqualsMarkdownContract. Otherwise D5 should say why the mirror may differ for a field. |
| fit-3 | fit | should-fix | in-change | Task 1.4 models the order property test on a serialization round-trip test in internal/indexer. internal/query already has a rapid property test of result order, TestQueryResultDoesNotDependOnNodeIDs, with its own defGraph fixture builder. That is the closest pattern, and the task neither names it nor says why it uses a different one. | tasks.md > 1. Query detail — "Add the property test `TestAccessOrderIgnoresEdgeOrder` with `pgregory.net/rapid`, after the pattern of `internal/indexe" | In task 1.4, name TestQueryResultDoesNotDependOnNodeIDs and defGraph in internal/query/definitions_test.go as the pattern. Build the drawn sources with defGraph.node, and add a references-edge helper next to defGraph.call rather than a second fixture builder. |
| inv-1 | invariants | should-fix | in-change | The D-13 headroom invariant of the UI transport counts neighbor lists as overhead, nothing bounds them, and D6 adds two more unbounded lists to each gathered definition and to the single-definition response. | internal/uiserver/truncate.go:62 — "// Plus the surrounding message's own overhead (node metadata," | Either cap accessed_by and initialized_to on the wire in task 3.2 with a test on both sides of the cap, or add a task that asserts the response stays under transportSendMaxBytes for a data symbol with many accessors. Also correct the per-entry premise: nodeToProto copies all 15 Node fields (id, qualified name, return type and docstring included), not a name, a path and a signature. |
| fit-2 | fit | note | in-change | Issue #274, which the proposal names as the follow-up, must move reverseEdgeIndex. Its predicate signature also differs from the indexer copy, which takes edge kinds, so #274 must reconcile the two signatures too. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "so issue #274 must move `reverseEdgeIndex` to a package that both import" | Record on #274 that the shared builder takes an edge predicate, because the access index filters on metadata as well as kind, so the indexer's kinds variadic becomes a predicate there. |
| inv-2 | invariants | note | in-change | AccessedBy holds exactly one entry per present source (a size bound over an unbounded edge set), and the planned property test does not generate missing sources, so the skip path and the order are only checked together by one example. | tasks.md > 1. Query detail — "Its generator also makes sources that tie on file, name, line, signature and kind, with the same role or with different " | In TestAccessOrderIgnoresEdgeOrder (pgregory.net/rapid, after internal/indexer/spill_roundtrip_test.go), let the generator drop some source nodes from the reader, and assert that len(AccessedBy) equals the number of present sources. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A method reads and writes a field | 2.2, 2.3 | covered |  |
| Readers in two files | 2.2, 1.2 | covered |  |
| Module-level code writes a variable | 2.2, 1.2 | covered |  |
| A variable holds a function | 2.2, 1.5 | covered |  |
| Nothing accesses a constant | 2.2, 1.6 | covered |  |
| A field and a method share a name | 2.2, 1.3, 1.6 | covered |  |
| A function keeps its output | 2.2, 1.3, 4.1 | covered |  |
| The graph UI asks for a field | 3.2 | covered |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: Checked and held at head: DefinitionDetail fields and Owner (detail.go), fetchCalls and fetchCalledBy (node.go), BuildReverseAdjacency, BuildImplementsIndex, buildContainsIndex, sortLocations and nodeLocation (traverse.go), RenderNode and renderNodeSection (render_markdown.go), writeSingleDef, writeNodeSection, writeNodeRefs and the stripped-length budget (cli/present/node.go), TestRenderNodeStrippedEqualsMarkdownContract, the T-01-18 subtest text, IsDataKind callers, the five inline BindingInit checks, withMergedAccess, UnknownWrite, edgeKey by source, kind and target, file nodes named by path at line 1, ui.proto next free numbers 6, 7, 10 and 11, uiMultiDefCap 20, transportSendMaxBytes 16 MiB, loadBrowseTarget, the Taskfile targets, the hugo field sections and the function-only wire oracle node transcript.
- approach: Task 1.1 names only resolve as the caller of the moved IsDataKind, while D1 also names internal/indexer/sync_bindings.go, which calls resolve.IsDataKind at line 16. The compiler forces that edit, so this is no finding of this lens.
- fit: openspec/config.yaml has no library-preference rows; only the generic rows of the rulebook apply.
- fit: D1: the data-kind set exists only in resolve.IsDataKind, so moving it to goextract leaves one copy; production internal/query already imports only goextract and nodeid.
- fit: D2: reverseEdgeIndex collapses the two same-shape scans BuildReverseAdjacency and BuildImplementsIndex. buildContainsIndex, buildTypeHierarchyIndex and setsOfMembers build other shapes and are rightly left out.
- fit: D4: the five inline binding=init checks that IsInitBinding replaces are at sync_bindings.go:98 and :110, bindings.go:30 and :88, and collapse.go:102, which matches the design's count.
- fit: D6: the new DataAccess message has no existing wire equivalent in ui.proto, and Svelte snippets are an existing pattern in the web app (workbench/+page.svelte, SourcePane.svelte).
- invariants: I1: The store holds at most one edge per (source, kind, target), so a source appears once in an access list and a node id is a unique Svelte key \| enforced at internal/graphstore/batch.go:46 (edgeKey) and resolve.CollapseEdges \| boundary test internal/graphstore/keyenc_test.go collision test; task 3.4 case 'two accessors that share file, line and name both render'.
- invariants: I2: A read and a write of one source to one target merge into one read_write edge \| enforced at goextract.MergeAccess (types.go:334) via resolve.withMergedAccess \| boundary test task 1.2 read_write case and spec scenario 'A method reads and writes a field' (task 2.2).
- invariants: I3: A file node has its path as Name and StartLine 1, so module-level access prints 'path (path:1, role)' \| enforced at goextract.go:64-69 and pyextract.go:65-73 \| boundary test task 1.2 file-source case and task 2.2 'module-level code writes a variable'.
- invariants: I4: Only a variable, a constant or a field gets the data lines and the access scan \| enforced at IsDataKind (moving to goextract) plus the kind gate in RenderNode \| boundary test task 1.3 scan counts 0/1, task 2.2 function and size pair, hugo goldens with struct and interface sections unchanged (task 4.1).
- invariants: I5: An outgoing binding=init edge has a data symbol as its source \| enforced at the captureInitializer helpers of goextract, tsextract and pyextract, which take the data symbol id as fromID \| boundary test none needed for the gate; task 1.5 covers present, absent and dangling init edges.
- invariants: I6: AccessedBy order is file, name, line, then signature, kind and role, independent of edge order \| enforced at the planned locationLess plus role key (D3) \| boundary test task 1.2 key-by-key cases and the rapid property test of task 1.4.
- invariants: I7: Production internal/query reaches only goextract and nodeid among indexer packages \| enforced at the T-01-18 subtest, which today forbids only the internal/indexer root; task 1.1 extends it \| go list -deps ./internal/query at head shows exactly nodeid and goextract, so the extended rule holds before the move.
- invariants: I8: New proto field numbers 6, 7 (NodeDefinition) and 10, 11 (GetNodeDetailResponse) are unused \| enforced at convention only; buf.yaml configures no breaking or lint gate \| verified at head: NodeDefinition uses 1-5, GetNodeDetailResponse uses 1-9, no DataAccess message and no reserved range covers them.
- invariants: I9: The UI gathers detail only for the first uiMultiDefCap candidates \| enforced at handlers.go:740 (i < uiMultiDefCap) \| boundary test task 3.2 case 3 tests both sides, read as the 1-based uiMultiDefCap-th candidate; a 0-based reading would name the last, ungathered candidate, so the task wording can mislead.
- invariants: I10: Top-level accessed_by and initialized_to are set only in single-definition mode \| enforced at the per-mode switch of nodeDetailToProto \| no planned test asserts them empty in multi-definition or file mode.
- invariants: No enforcement point in this change is a script, so no probe inputs were run.
- spec-lint: Checked all four requirements and their scenarios: each requirement has at least one scenario, and each scenario has WHEN or GIVEN and THEN lines with concrete inputs and observables.
- spec-lint: Every tracked task in sections 1 to 5 names a file, test, command or observable. No BREAKING marker appears in proposal.md or the delta spec.
- spec-lint: Build targets task proto:gen, proto:drift, golden:regen, test:golden and changie exist in Taskfile.yml. The pnpm scripts test and check exist in web/package.json.
- spec-lint: design.md has no open-questions section, so the assumed-answer check had nothing to test.
- spec-lint: A grep over the change directory printed a line of review.md. That line was not used, and no finding relies on it.
- test-coverage: Every task names its tests but only 1.1, 3.1, 3.4, 4.1 and 5.1 give a literal command. The tests in 1.2 to 3.3 are runnable by name.
- test-coverage: No BREAKING requirement exists, so no breaking-unproven finding.
- test-coverage: The spec has 8 scenario headings; the matrix has 8 rows.
