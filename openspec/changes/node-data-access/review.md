---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-15T13:52:49+00:00
head: 2bc14b359a1790cf495db684261a1a98e2296617
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
| app-1 | approach | should-fix | beyond-scope | D1 widens the T-01-18 subtest so that it forbids every internal/indexer package except goextract and nodeid. No requirement, non-goal or risk of this change needs that wider check. Moving IsDataKind already keeps production internal/query off resolve. | design.md > D1. Two new fields on DefinitionDetail — "The T-01-18 subtest is extended to enforce the whole rule."; scope basis: proposal.md > Impact — "`IsDataKind` moves from `internal/indexer/resolve` to `internal/indexer/goextract`, which also gains `IsInitBinding`." | Either name in design.md the requirement, non-goal or risk that the wider T-01-18 check serves, or drop the test extension from D1 and task 1.1 and leave it to a separate change. |
| app-2 | approach | should-fix | in-change | D2 justifies reverseEdgeIndex with a repository convention that a loop is shared once it has a third copy. No such rule is written in openspec/config.yaml or the repository docs. traverse.go already holds more than two IterateEdges("") scan loops with no shared loop (buildContainsIndex, setsOfMembers, FileGraph's second scan). | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "the repository shares a loop once it has a third copy" | Replace the cited convention in D2 with the actual reason for the shared loop, or cite the place where the rule is recorded. |
| fit-2 | fit | should-fix | in-change | D6 gives the shared neighbor-list snippet a key function so that callers and callees keep the file:line:name key while the two data lists key by node id. Both kinds of list hold Node messages that carry an id, so one panel gets two key mechanisms for the same entry type, and the old key keeps the duplicate-key fault that D6 names for the data lists. | design.md > D6. UI wire, state and panel — "The two data lists key by node id, because two sources can share file, line and name, and Svelte rejects a keyed list wi" | Let the snippet key every Node entry by node.id and drop the key-function parameter. Keep entryKey for the test ids and for the blast-radius Location list, which has no id. |
| fit-3 | fit | should-fix | in-change | writeSingleDef and writeNodeSection mirror RenderNode and renderNodeSection minus markdown syntax, but they leave out the owner and language lines of a field. D5 edits both mirrors and their contract test and keeps that drift, and task 2.1 picks contract subtests that use no field. | internal/query/render_markdown.go:148 — "for _, line := range fieldFacts(d) {" | Print the owner and language lines of a field in writeSingleDef and writeNodeSection, after the location, as fieldFacts does, and add a field subtest to TestRenderNodeStrippedEqualsMarkdownContract. |
| inv-1 | invariants | should-fix | in-change | The rule that a non-data section never changes is enforced per candidate in buildMultiDefDetail once the shared access index exists. No planned test passes a function that is itself the target of an access edge next to a data candidate. The init binding gives a function exactly such an incoming read edge, and the size method in the fixture has none. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "If any candidate is a data kind, `buildMultiDefDetail` calls it once and shares it across candidates" | Add a case to TestNodeDataAccess. It builds a multi-definition name that has a data candidate and a function candidate, and the function is the target of a references edge with access read, for example from a `binding=init` edge. Assert that the function's DefinitionDetail has nil AccessedBy and InitializedTo, and that its rendered section has no data lines. |
| inv-2 | invariants | should-fix | in-change | Every ui.proto field number is pinned by the fixture of TestUIProtoFieldNumbersAreStableAndUnique, which fails on a declared field that has no entry. No task adds entries for the six new fields (DataAccess.node and access, NodeDefinition 6 and 7, GetNodeDetailResponse 10 and 11) or raises the pinned fixture length. Until it does, nothing pins the numbers that D6 promises old clients. | internal/uiserver/readonly_test.go:682 — "t.Fatalf("descriptor declares field %s, but no fixture entry covers it — the fixture must cover every field of every mes" | In task 3.1, add the six entries to uiProtoFieldNumbers in internal/uiserver/readonly_test.go. Add a chained length constant that is the previous one plus 6, and state that TestUIProtoFieldNumbersAreStableAndUnique passes. |
| inv-3 | invariants | should-fix | in-change | The extended T-01-18 subtest has two sides, allowed (goextract, nodeid) and forbidden (any other internal/indexer package). The plan keeps only the inspected-package count. Nothing shows that the matcher fires on the forbidden side, or that the dependency list holds an allowed indexer package, so a matcher that never matches an indexer subpackage would pass. | internal/depclosure/dependency_closure_test.go:207 — "if depSet(p)[indexerRoot] {" | In task 1.1, put the allowlist predicate in a function. Give it a table with internal/indexer/resolve and internal/indexer/goextractx (rejected), and goextract and nodeid (accepted). Also assert that production internal/query depends on internal/indexer/goextract, so the dependency list holds an indexer package that the rule can match. |
| inv-4 | invariants | should-fix | in-change | The styled view decides the trail header with its own condition, separate from renderNodeSection. Task 2.1 gives that condition one 'MultiDef data' subtest, while the plain view gets three edge cases. A styled condition that leaves out InitializedTo passes a case with access entries. | internal/cli/present/node.go:119 — "if len(calls) > 0 \|\| len(calledBy) > 0 {" | Split 'MultiDef data' in task 2.1 into three sections that mirror cases 3 to 5 of task 1.6: only an access entry, only an initializer entry, and four empty lists. Compare each with query.RenderNodeMultiDef. |
| fit-1 | fit | note | in-change | D2 puts the new shared scan loop reverseEdgeIndex in internal/query, so issue #274, the follow-up the proposal names, must move it again, while the indexer keeps its own copy of the same loop. internal/graphstore, which holds Reader and is imported by both packages, could hold it today. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "so issue #274 must move `reverseEdgeIndex` to a package that both import. It stays in `internal/query` until then" | Either place reverseEdgeIndex in a package both import (internal/graphstore takes a Reader and returns schema edges) and route internal/indexer/sync.go buildReverseAdjacency through it in this change, or keep D2 as is and record in tasks.md that #274 moves reverseEdgeIndex. |

## Coverage matrix

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

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: Each of the four requirements has a producing decision: Accessed by (D2, D3, D5), Initialized to (D4, D5), data-only lines and the multi-definition rules (D1, D5), and the UI wire and panel (D6).
- approach: Verified at head: fetchCalls and fetchCalledBy in internal/query/node.go, BuildReverseAdjacency, BuildImplementsIndex, buildContainsIndex and sortLocations in traverse.go, and resolve.IsDataKind in resolve/bindings.go with callers in resolver.go and sync_bindings.go. Also verified: the five inline BindingInit checks, withMergedAccess, UnknownWrite and BindInitializer, file nodes named by path at StartLine 1, the free field numbers 6/7 and 10/11 in ui.proto, uiMultiDefCap 20, the 16 MiB transportSendMaxBytes, and the positional lists of writeNodeSection.
- approach: The scenario fixtures match the cited lines. In p/Account.java, deposit is at line 7 and size() at line 12. In Go, var x = f() is a call_expression, so mayHoldFunction holds and BindInitializer writes UnknownWrite from the file node.
- approach: Only the hugo node goldens hold field, variable or constant sections. That matches task 4.1.
- fit: openspec/config.yaml context names no library-preference rows; only the generic rows of the rulebook apply.
- fit: The forward-misfit category carries severity suggestion in the rulebook; the envelope records it as note, the closest severity the envelope allows.
- fit: D1 (IsDataKind to goextract), D3 (fetchSources, locationLess), D4 (fetchTargets, IsInitBinding over the five inline checks) and the uiserver mapping through nodesToProto each reuse or collapse the closest existing helper; no fit finding there.
- fit: The other full edge scans in internal/query (buildContainsIndex, buildTypeHierarchyIndex, buildExpandAdjacency, FileGraph) have other shapes than the key-by-target edge list, so D2's reverseEdgeIndex leaves no third copy of its own loop in internal/query.
- invariants: I1: The store holds at most one edge per (source, kind, target) \| enforced at internal/graphstore/batch.go:46 edgeKey and internal/indexer/resolve/collapse.go CollapseEdges \| boundary test tasks 1.2 (two distinct sources that tie on every key) and 3.4 (two accessors that share file, line and name).
- invariants: I2: A read and a write of one source merge into one read_write edge \| enforced at internal/indexer/resolve/collapse.go:108 goextract.MergeAccess \| boundary test tasks 1.2 read_write case and 2.2 'a method reads and writes a field'.
- invariants: I3: The access metadata is only read, write or read_write \| enforced at the extractors' use of the goextract.Access* constants and MergeAccess, with no check on the query side \| boundary test none needed for the change, which prints the stored value as the spec asks.
- invariants: I4: A references edge without access metadata is not in the access index \| enforced at the planned buildAccessIndex filter \| boundary test task 1.2 case of an edge without access next to a read edge (both sides covered).
- invariants: I5: A file node has its path as Name and StartLine 1 \| enforced at the file node construction of every extractor (goextract.go:69, tsextract.go:73, pyextract.go:72, javaextract.go:66 and the rest) \| boundary test tasks 1.2 file-source case and 2.2 module-level scenario.
- invariants: I6: The data lines appear only for variable, constant and field \| enforced at the planned IsDataKind gate in buildSingleDefDetail and per candidate in buildMultiDefDetail \| boundary test task 1.3 counts for single-definition, and none for a function with an incoming access edge in a mixed multi-definition (inv-1).
- invariants: I7: A data symbol has at most one binding=init edge \| enforced structurally: nodeid.Allocator.Declare gives each declaration occurrence its own id, and each declaration has one initializer \| boundary test none. With more than one, InitializedTo would follow store key (target id) order, so it is a note only.
- invariants: I8: The new proto field numbers are unused and stay stable \| free at head (NodeDefinition uses 1-5, GetNodeDetailResponse uses 1-9, no reserved range there) and pinned by the readonly_test.go fixture \| boundary test none planned (inv-2).
- invariants: I9: Production internal/query reaches only goextract and nodeid among indexer packages \| holds at head (go list -deps shows only those two) and is enforced at dependency_closure_test.go T-01-18 \| boundary test none for the allowlist predicate (inv-3).
- invariants: I10: Accessed by is independent of edge order and node ids \| enforced at the planned locationLess-then-role sort \| property test task 1.4 TestAccessOrderIgnoresEdgeOrder with pgregory.net/rapid, so no property-candidate finding.
- invariants: I11: The multi-definition body budget is used+len(section) <= 12000 \| enforced at render_markdown.go RenderNodeMultiDef and present/node.go:154 \| boundary test task 1.6 case 6 for the plain view. The change does not touch the comparison.
- invariants: I12: A data list in the panel has unique keys (Svelte rejects a duplicate key in a keyed each block) \| enforced by keying on node id together with I1 \| boundary test task 3.4 'two accessors that share file, line and name both render'.
- invariants: Task 3.2 case 3 'candidate uiMultiDefCap carries its entry' is right only when read 1-based (index uiMultiDefCap-1, the last gathered one, since handlers.go:740 gathers i < uiMultiDefCap). Naming the index removes the ambiguity.
- invariants: No enforcement point of this change is a script, so no probe was run. go list -deps was run as a read only.
- spec-lint: All four requirements in specs/data-symbols/spec.md have at least one scenario with WHEN and THEN lines. Every WHEN names a fixture, input or command, and every THEN names a rendered line, field or absence of a line.
- spec-lint: Scenario 'Readers in two files' THEN uses the placeholder 'a.go:...' for the line number. The ordering observable is still named, so no finding is raised.
- spec-lint: Scenario 'A function keeps its output' THEN checks only the absence of the two data lines. It names no baseline output, so 'keeps its output' is checked only in part. No finding is raised because the THEN names an observable.
- spec-lint: Requirement 'Node lists the readers and writers of a data symbol' is matched by task 2.2 (subtests such as 'a method reads and writes a field') and task 2.3 (data_symbols_test.go and the Accessed by line). Match is by word, not exact heading, so it is a weak match.
- spec-lint: BREAKING: neither proposal.md nor specs/data-symbols/spec.md contains the word BREAKING. The proposal describes only additive output and additive wire fields, so no mismatch.
- spec-lint: Build targets named in tasks.md exist: task proto:gen, proto:drift, golden:regen, test:golden, changie, test and lint in Taskfile.yml; pnpm scripts test and check in web/package.json. go test invocations use package paths.
- spec-lint: design.md has no open-questions section, so no assumed-answer check applies.
- spec-lint: A content search matched a line of a change file outside the artifacts I was asked to read. I did not use that text in this review.
- test-coverage: Most tasks name a test function but give no go test command; only 1.1, 3.1 and 3.4 give one. Graded covered because each named test states its assertion.
- test-coverage: No BREAKING requirement in the delta spec.
