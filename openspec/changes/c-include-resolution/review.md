---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-15T09:57:13+00:00
head: ed26f7a1c905a7da61dc841643affb05f739d966
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
| app-1 | approach | should-fix | in-change | D5 drops each identifier at the declarator field of a declarator and names parenthesized_declarator and reference_declarator as such declarators, but at the pinned grammars neither has a declarator field. So the name in a C++ reference parameter such as int &r, and cb in void (&cb)(int) that task 3.0 asserts equal to void (&)(int), sit in an unnamed child and stay in the key, against the requirement that the match ignores parameter names. | design.md > D5. A view index of C and C++ definitions by name — "through `pointer_declarator`, `array_declarator`, `function_declarator`, `parenthesized_declarator` and `reference_decla" | State in D5 that the key drops the identifier that a declarator names, whether it sits at the declarator field or is the unnamed declarator child of reference_declarator or parenthesized_declarator, as cextract.innermostIdentifier already falls back to the first identifier child. Add int &r against int & to the TestParamKey table of task 3.0. |
| inv-2 | invariants | should-fix | in-change | paramKey treats a lone `(void)` as empty, but no TestParamKey case checks the other side of that rule: `(void *)` must not key as empty. | design.md > D5. A view index of C and C++ definitions by name — "A list of `(void)` keys as empty, as `collectParams` reads it." | In task 3.0, add TestParamKey cases that assert `(void *)` and `(void)` have unequal keys, and that `void (*)(void)` and `void (*)()` keep their inner lists apart. |
| fit-1 | fit | note | in-change | storeIndex.includers adds one more full-scan reverse adjacency of stored edges, next to the sync's buildReverseAdjacency and query.BuildReverseAdjacency. design.md names the copy and issue #274 tracks one shared builder, and task 4.1 puts includers in the scope of #274, so this is a note. | design.md > D6. Sync dependents — "It is one more reverse adjacency of stored edges, next to the cached `reverse()` of `sync.go` and `query.BuildReverseAdj" | Keep the 4.1 step that names storeIndex.includers in the PR body as part of #274, and add includers to the list in the #274 issue body when the PR merges. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A header beside the source file | 2.3 | covered |  |
| A path from the repository root | 2.3 | covered |  |
| A sibling include directory | 2.3 | covered |  |
| An absolute path | 2.3 | covered |  |
| A system header | 2.3 | covered |  |
| An inner dot segment | 2.3 | covered |  |
| A header that includes itself | 2.3 | covered |  |
| A function defined in a header | 3.4 | covered |  |
| An include of an include | 3.4 | covered |  |
| A struct tag in a nearer header | 3.4 | covered |  |
| Include guards that form a cycle | 3.4 | covered |  |
| The nearest header wins | 3.4 | covered |  |
| Two headers at one depth | 3.4 | covered |  |
| A header at depth 1 and depth 2 | 3.4 | covered |  |
| The nearest header has the wrong arity | 3.4 | covered |  |
| Every candidate at the first depth has the wrong arity | 3.4 | covered |  |
| A name the file does not include | 3.4 | covered |  |
| A static helper | 1.2 | covered |  |
| A static forward declaration | 1.2 | covered |  |
| A static declaration outside the definition's #if | 1.2 | covered |  |
| A C++ static member function | 1.2 | covered |  |
| The repro from #259 | 3.4, 3.5 | covered |  |
| A library prototype | 3.4 | covered |  |
| One definition for each platform | 3.4 | covered |  |
| A static definition of the same name | 3.4 | covered |  |
| A prototype under an include guard | 3.4 | covered |  |
| Pointer spacing and unnamed parameters | 3.4 | covered |  |
| Two definitions under #if in one file | 3.4 | covered |  |
| A C++ default argument | 3.4 | covered |  |
| A static prototype | 3.4 | covered |  |
| A definition in the prototype's own file | 3.4 | covered |  |
| A C++ method of the same name | 3.4 | covered |  |
| Another signature | 3.4 | covered |  |
| An array parameter with and without a name | 3.4, 3.0 | covered |  |
| A function-pointer parameter with and without names | 3.4, 3.0 | covered |  |
| Pointer spacing in one file | 3.4, 3.0 | covered |  |
| A default argument in one file | 3.4, 3.0 | covered |  |
| A definition appears after the call | 4.3 | covered |  |
| A header loses its prototype | 4.3 | covered |  |
| An include is removed | 4.3 | covered |  |
| A header appears after its include | 4.3 | covered |  |
| An include cycle in a sync | 4.3 | covered |  |
| A header is deleted | 4.3 | covered |  |
| A header moves to the repository root | 4.3 | covered |  |
| An indirect header is deleted | 4.3 | covered |  |
| A header appears at the root | 4.3 | covered |  |
| A header's own include moves | 4.3 | covered |  |
| A nearer header appears | 4.3 | covered |  |
| A definition loses its body | 4.3 | covered |  |
| A prototype gains its body | 4.3 | covered |  |
| A header with no symbols | 4.3 | covered |  |
| A definition loses static | 4.3 | covered |  |
| A second definition appears | 4.3 | covered |  |
| The first sync after an upgrade | 4.5 | covered |  |
| The syncs after the rebuild | 4.5 | covered |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

- fit-2 (fit, confidence 0.6): D5 has cextract.paramKey walk the leaf tokens of each parameter subtree, and skip comments, the default_value subtree and declarator names. It does not name cextract's existing iterative walkDescendants (the T-02-04 depth guard), whose visitor returns false to prune a subtree. A hand-written recursive walk would be one more tree walk in the package.
- inv-1 (invariants, confidence 0.6): A sync can miss a cross-file prototype redirect when only CParamKey changes. The node id stays the same, and D6.2 flags only IsExported and Prototype changes.

## Lens notes

- approach: Verified at head: C and C++ ModuleKey is relPath; resolveCall ends in CandidatesIn(r.ImportPath, ref.Name); emitInclude trims quotes and brackets into one RefKindImports ref; isIntraModule with an empty module path drops a relative include and keeps one that starts with /; GlobalEdges keeps only structural and binding edges.
- approach: Verified at head: collectFunctions builds definedKeys from sigKey, which appends ConditionalBlockKey; SignatureKeyOfNode has no conditional part; SignatureKeyOf marks a parameter with a default as optional; collectParams stores a size identifier as the name of an abstract array parameter.
- approach: Verified at head: graph.proto Node ends at field 20 and Meta at field 16, so 21, 22 and 17 are free; stampCapabilities and needsFileIndexBackfill are the flag pair D7 names; tsextract.visibilityFor writes module for an unexported symbol.
- approach: Verified at head: WalkSupertypes and query.transitiveWalk are the two BFS walks over map[string][]string that D3 rebuilds; storeIndex.load scans edges once and calls IsBindingEdge, as does loadStoreView; the wave loop calls bindingDependents per wave while filesWaitingOn and filesResolvingToChangedSets run once before it.
- approach: Verified at head: Reachable keeps every non-nested C and C++ node a candidate, so an IsExported false static function in a header stays reachable for the include walk.
- approach: Verified the rejected alternatives: gonum traverse.BreadthFirst works on int64 node ids, and the full index passes only GlobalEdges to later passes. No rejection is contradicted.
- approach: Each risk mitigation names a task (3.0, 3.2, 3.6, 5.2), the migration plan has a rollback step, and the design has no open question.
- approach: Outside this lens: topLevelDecls does not descend into a C++ linkage_specification, so prototypes inside extern "C" { } in a .hpp get no node and the include walk cannot reach them. The proposal does not list this limit.
- fit: openspec/config.yaml has no library-preference rows. A search for library, prefer and helper there finds nothing, so only the generic rows of the rulebook apply.
- fit: D3 follows the rulebook for the third visited-set walk: bfs.Levels replaces WalkSupertypes (symbols/hierarchy.go:338) and query.transitiveWalk (query/expand.go:128), and task 3.2 does the collapse. It has more than one call site: the two rebuilt walks, the include walk in resolveCall and includeDependents. Impact, Affected and expandBFS keep their own loops, and D3 gives the reason.
- fit: D5 adds a third C signature key, c_param_key, beside sigKey and SignatureKeyOfNode, which feeds Candidate.Key and the DeclID group in symbols.Pick. D5 names the alternatives of changing Params or sigKey and gives the reason that node ids would change, so this is not reported as a concept duplication.
- fit: D1, D4, D5 and D7 follow existing patterns: LinksIncludes sits next to LastWins and InheritsMembers, IsIncludeEdge and MetaInclude follow IsBindingEdge and MetaBinding, AddStoredInclude sits beside AddStoredEdge, cDefinitionsByName fills in the goFieldsByName loop of Freeze, module visibility matches tsextract.visibilityFor, and has_c_prototypes follows stampCapabilities and needsFileIndexBackfill.
- fit: includeDependents copies the shape of ownerDependents (sync_owners.go:84), which walks idx.supers with WalkSupertypes, and joins the wave loop beside bindingDependents (sync.go:556).
- fit: The fit-lens rulebook gives forward-misfit and parameterization-candidate the severity suggestion, but the envelope allows only blocker, should-fix and note. This review would report such a finding as note. None was found: the out-of-scope include directories from build files only extend the candidate list of IncludeCandidates and the waits of D2.
- invariants: I1: A file never links to itself through an include \| enforced at planned Freeze and symbols.IncludeCandidates (D1) \| boundary test TestIncludesOf cases a.h from a.h, src/a.h from src/a.h, a.h from src/a.h with root a.h (task 2.1), plus the rapid property in task 2.1
- invariants: I2: An include path that starts with /, leaves the root, or has an inner ., .. or empty segment links nothing \| enforced at planned IncludeCandidates with fs.ValidPath \| boundary test TestIncludesOf cases ../../x.h, add.h/, add.h/., nodir/../add.h, ../sub/../add.h (task 2.1). A probe of the D1 rule under the handoff probe directory found no crash. Only written paths . and .. pass the segment rule, and they give directory candidates such as src or ., which no held C file can equal.
- invariants: I3: A result with Err set is never held \| enforced at internal/indexer/goextract/types.go:476 (FinalizeIDs drops every node on error) and the parse-error returns in cextract.go before the file node \| boundary test TestIncludesOf failed-extraction case (task 2.1)
- invariants: I4: Two directives that match one file give one imports edge \| enforced at resolve.CollapseEdges keyed by EdgeTriple (internal/indexer/resolve/collapse.go:41) \| boundary test TestCIncludeLinks case add.h plus ./add.h (task 2.3)
- invariants: I5: A stored include edge is recognized by its MetaInclude mark alone \| enforced at planned resolve.IsIncludeEdge, shaped like IsBindingEdge (internal/indexer/resolve/bindings.go:26) \| boundary test table test in task 2.2, both sides
- invariants: I6: An include key under IncludeModule never meets a symbol wait \| enforced at none: D2 says a Go import of the NUL key can equal it, and treats a meeting as one harmless extra re-resolve \| boundary test table case in task 2.3
- invariants: I7: Each reachable include file is visited once, at its least depth \| enforced at planned bfs.Levels \| boundary test TestLevels diamond and shortcut, plus a rapid property (task 3.2)
- invariants: I8: The first depth with a candidate of that name decides, and a lone candidate wins whatever its arity \| enforced at resolve.choose (internal/indexer/resolve/resolver.go:742, arity filter only when len(callables) > 1) \| boundary test scenarios The nearest header has the wrong arity and Every candidate at the first depth has the wrong arity (task 3.4)
- invariants: I9: A redirect happens only for exactly one matching non-static definition in another file \| enforced at planned resolver redirect and View.CDefinitionsNamed (D5) \| boundary test scenarios with zero (A library prototype), one (The repro from #259) and two definitions (One definition for each platform, Two definitions under #if in one file)
- invariants: I10: A static change, a body change or a CParamKey change keeps the node id, because the id hashes kind, qualified name, file and signature key (internal/indexer/nodeid/allocator.go DeclID) \| enforced at the planned changed-key set for IsExported and Prototype only \| boundary test 4.3 steps A definition loses static, loses its body, gains its body. CParamKey is not covered (inv-1)
- invariants: I11: A store without has_c_prototypes is rebuilt once \| enforced at needsFileIndexBackfill (internal/indexer/sync.go:986) and stampCapabilities \| boundary test TestCPrototypeRebuildOnce asserts the file count on the first sync and 1 on the second and third (task 4.5). Every sync builds Meta with schema.NewMeta (sync.go:760), so a sync by an older binary drops the flag, and a later upgrade rebuilds again.
- invariants: I12: Proto field numbers Node 21 and 22 and Meta 17 are free \| enforced at protoc and task proto:drift \| Node currently ends at nested = 20, Meta at has_data_symbols = 16, and the reserved ranges are 50 to 59, so there is no collision
- invariants: I13: A C node stays a candidate in another file whatever its IsExported value \| enforced at symbols.Reachable, which returns true for C (internal/indexer/symbols/candidate.go:56) \| boundary test scenario A function defined in a header (static inline twice)
- invariants: I14: The result does not depend on indexing order \| enforced at symbols.Pick lowest id \| property TestCIncludeOrder with pgregory.net/rapid v1.3.0 (task 4.4)
- invariants: No output path serializes a raw schema.Node, so the new Prototype and CParamKey fields stay out of CLI and MCP output. query/search.go and uiserver/handlers.go copy fields one at a time.
- invariants: No weakened finding: no existing test relies on a C absolute include making a package node, or on a C call targeting a struct.
- spec-lint: Form check of proposal.md, specs/graph-index/spec.md, design.md and tasks.md found no should-fix defect in the classes of this lens.
- spec-lint: All 7 requirements of the delta spec have at least one scenario, and every scenario has a WHEN or GIVEN line and a THEN line.
- spec-lint: Every THEN step names an observable: an imports or calls edge, a node with a visibility, a store equal to a fresh index, or a FilesReparsed count.
- spec-lint: Every requirement heading is named in a task line, or its scenarios are named in a task line (tasks 1.2, 2.3, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 4.5).
- spec-lint: Build targets named in tasks.md exist in Taskfile.yml: proto:gen, proto:drift, golden:regen, corpora:measure, changie, test, lint. The package path go test ./internal/indexer/capability exists.
- spec-lint: BREAKING appears in neither proposal.md nor the delta spec, so there is no breaking-mismatch. This lens does not judge whether the struct-tag and static-visibility changes are breaking.
- spec-lint: design.md Open Questions reads None, so no assumed-answer check applies.
- spec-lint: A Grep for BREAKING matched a line in the change's review.md. That file was not read and was not used.
- test-coverage: No requirement is marked BREAKING, so no breaking-unproven finding.
- test-coverage: Tasks name tests, not literal go test commands; 5.4 runs task test. Graded covered on the named test plus Verify clause.
