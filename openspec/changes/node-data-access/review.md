---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-15T13:38:07+00:00
head: 9e8d8d3dfd8299d0a79e3afc4c32281b670c4fda
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
| app-1 | approach | should-fix | in-change | D4 rewrites the inline binding=init checks in resolve/bindings.go, resolve/collapse.go and internal/indexer/sync_bindings.go to call IsInitBinding. No requirement, non-goal or risk of this change needs the indexer call sites to change. Only the query side needs the predicate. | design.md > D4. Initialized to: the initializer edges of the node itself — "which `resolve/bindings.go`, `resolve/collapse.go` and `internal/indexer/sync_bindings.go` now write inline, and those p" | In D4, name the requirement or risk that needs the indexer call sites to change. If none does, drop the rewrite of resolve and sync_bindings from D4 and task 1.1 and keep IsInitBinding for the query side. |
| app-2 | approach | should-fix | in-change | D2 routes the existing BuildReverseAdjacency and BuildImplementsIndex through the new reverseEdgeIndex. No requirement needs the existing builders to change. Its stated motive is the merge with the indexer copy (#274), which the design lists as a non-goal. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "`BuildReverseAdjacency` and `BuildImplementsIndex` call it with their kind filters, and `BuildImplementsIndex` keeps its" | In D2, state which requirement or risk needs the existing builders rerouted. If none does, limit D2 and task 1.1 to the new buildAccessIndex and leave BuildReverseAdjacency and BuildImplementsIndex unchanged. |
| inv-1 | invariants | should-fix | in-change | D1 moves IsDataKind to keep the whole T-01-18 rule (only goextract and nodeid reachable from production query), but no test enforces that rule and no task creates one. | design.md > D1. Two new fields on DefinitionDetail — "The test enforces only the part of that rule that forbids the `internal/indexer` root. This change keeps to the whole ru" | Add a task that extends the T-01-18 subtest to fail when production internal/query depends on any internal/indexer package other than goextract and nodeid, with internal/indexer/resolve as the edge case. |
| inv-2 | invariants | should-fix | in-change | No planned test discriminates the file key from the name key: the two-file case uses names whose order agrees with the file order. | specs/data-symbols/spec.md > Scenario: Readers in two files — "`b.go` declares `func second()` and `a.go` declares `func first()`" | In TestNodeDataAccess, name the b.go source so it sorts before the a.go source by name (for example `alpha` in b.go, `zeta` in a.go) and assert a.go first. |
| inv-3 | invariants | should-fix | in-change | buildAccessIndex keeps only references edges with a non-empty access, and references edges without access exist, but no planned case passes one to a data symbol. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "A new `buildAccessIndex` calls it and keeps each `references` edge whose `access` metadata is not empty." | Add a TestNodeDataAccess case: a references edge with no access metadata to the data symbol gives no AccessedBy entry, next to a read edge from another source that does. |
| fit-1 | fit | note | in-change | D5 gives formatNodeRef, joinNodeRefs and writeNodeRefs one optional suffix, but joinNodeRefs and writeNodeRefs take a []*schema.Node and AccessedBy is a []DataAccess whose role differs per entry, so one suffix value cannot carry the roles. | design.md > D5. Markdown and styled lines — "`formatNodeRef` and `joinNodeRefs` take an optional suffix that goes inside the parentheses, so an access entry is `name" | Name the parameter in D5: a per-entry suffix (for example a slice of suffixes parallel to the nodes, or a func(i int) string) for joinNodeRefs and writeNodeRefs, and a single string suffix for formatNodeRef. |
| fit-2 | fit | note | in-change | writeNodeSection takes the calls and callers as positional lists while its plain mirror renderNodeSection and its sibling writeSingleDef take the DefinitionDetail; D5 adds two more lists of a new type to it without saying how they arrive. | internal/cli/present/node.go:102 — "func writeNodeSection(pal Palette, n *schema.Node, source []byte, calls, calledBy []*schema.Node) string {" | Say in D5 that writeNodeSection takes the *query.DefinitionDetail (pal, dd) as renderNodeSection does, instead of growing to six positional arguments. |
| fit-3 | fit | note | in-change | D2 places the new shared reverseEdgeIndex unexported in internal/query, where internal/indexer cannot reach it, so #274 must move it; internal/graphstore is already imported by both packages and could hold it now. | design.md > D2. One shared reverse index builder, one more scan only for a data symbol — "`internal/indexer` cannot import `internal/query`, so issue #274 must move `reverseEdgeIndex` to a package that both imp" | Either state in D2 why the builder stays in internal/query until #274, or put it in internal/graphstore now and let internal/indexer's buildReverseAdjacency call it with a kinds filter, which leaves #274 only the other full-scan indexes. |
| fit-4 | fit | note | in-change | The D6 snippet's parameter list omits two things that vary between its five uses: the test-id prefix of the entry buttons and empty text, which the existing tests pin, and whether an empty list shows its empty text or hides the whole block. | design.md > D6. UI wire, state and panel — "the heading, the empty text, the entry buttons, an optional suffix for the role, and a key function." | Add the test-id prefix and the hide-when-empty choice to the snippet parameters in D6, so callers, callees and blast radius keep their current test ids and empty text, and the data lists hide when empty. |
| inv-4 | invariants | note | in-change | The stripped styled view equals RenderNode for a data symbol over any lists of entries and roles, an unbounded domain that the plan checks with fixed subtests only. | tasks.md > 2. CLI and MCP — "`TestRenderNodeStrippedEqualsMarkdownContract` gets the subtests "SingleDef variable", "SingleDef constant" and "MultiDe" | Optionally add a pgregory.net/rapid property after internal/indexer/spill_roundtrip_test.go that generates a variable or constant DefinitionDetail with random AccessedBy and InitializedTo lists and asserts the stripped styled output equals RenderNode. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A method reads and writes a field | 2.2 | covered |  |
| Readers in two files | 2.2 | covered |  |
| Module-level code writes a variable | 2.2 | covered |  |
| A variable holds a function | 2.2 | covered |  |
| Nothing accesses a constant | 2.2 | covered |  |
| A field and a method share a name | 2.2 | covered |  |
| A function keeps its output | 2.2 | covered |  |
| The graph UI asks for a field | 3.2 | covered |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: internal/indexer/sync_bindings.go line 16 also calls resolve.IsDataKind. D1 and task 1.1 name only resolve as the caller to update when IsDataKind moves to goextract.
- approach: D3 says fetchCalledBy becomes fetchSources, but internal/query/render_markdown_test.go line 495 calls engine.fetchCalledBy directly, and no task names that test.
- approach: D5 gives joinNodeRefs one optional suffix, but AccessedBy entries each carry their own role. The design does not say how a per-entry role reaches joinNodeRefs, which takes []*schema.Node today.
- approach: Every other cited path, name and behavior checked holds at head: edgeKey keys by source, kind and target; CollapseEdges merges access per triple; UnknownWrite comes from the file node, whose StartLine is 1; ui.proto field numbers 6, 7, 10 and 11 are free; uiMultiDefCap is 20; transportSendMaxBytes is 16 MiB; only the hugo node goldens hold data sections; the wire oracle node call shows a function.
- fit: The fit rulebook gives forward-misfit and parameterization-candidate the severity suggestion, which the envelope enum does not have; those findings are written as note.
- fit: D1 moves IsDataKind to goextract, and internal/indexer/sync_bindings.go:16 calls resolve.IsDataKind; task 1.1 does not name that call site. If resolve keeps a forwarding IsDataKind instead, the repo has two names for one predicate.
- fit: D4's IsInitBinding call sites in design and task 1.1 (resolve/bindings.go, resolve/collapse.go, internal/indexer/sync_bindings.go) cover every inline MetaBinding == BindingInit check found in production code.
- fit: D3 reuses the sortLocations comparator through locationLess, D2 follows the buildReverseAdjacency package-variable seam, D6 follows the existing NodeDefinition calls and called_by gathering and the Svelte snippet precedent in workbench/+page.svelte; no deviation found there.
- fit: The styled and plain node renderers are a sanctioned mirror (present/node.go budget constants comment) and the design names the remaining difference; not reported as duplication.
- invariants: I1: The store holds at most one references edge per source and target \| enforced at internal/graphstore/keys.go:104 edgeKey(src, kind, dst) and resolve.CollapseEdges (units.go:436) \| boundary test tasks 3.4 (two accessors sharing file, line and name) and 1.2 (distinct sources tying on every key).
- invariants: I2: A read and a write of one source merge into read_write \| enforced at goextract.MergeAccess (types.go:334) via resolve.withMergedAccess \| boundary test existing resolve tests; task 1.2 read_write case and scenario 'A method reads and writes a field'.
- invariants: I3: The access role is always read, write or read_write \| enforced at goextract constants (types.go:285-287) and MergeAccess closure; no validation at render \| boundary test none needed for a closed set; the styled suffix is not passed through sanitizeControl in the plan, safe only while the set stays closed.
- invariants: I4: A module-level access has the file node as source, whose name is its path and StartLine is 1 \| enforced at every extractor's file node (goextract.go:61-69 and the other extractors, all StartLine 1) \| boundary test task 2.2 'module-level code writes a variable'.
- invariants: I5: var x = f() stores a write from the file and const limit = 3 stores none \| enforced at goextract.BindInitializer and mayHoldFunction (goextract.go:1052, int_literal false) \| boundary test task 2.2 both scenarios.
- invariants: I6: An empty Accessed by line means no stored code reads or writes the symbol \| holds only for stored edges: a literal initializer (var y = 3) and a named initializer (var a = b) write the symbol with no stored write edge \| boundary test task 2.2 constant scenario; the wording relies on 'stored'.
- invariants: I7: Only a data kind (variable, constant, field) gets the lines and the access scan \| enforced at IsDataKind (resolve/bindings.go:50, moving to goextract) \| boundary test tasks 1.3 (counts 0 and 1 on both sides) and 1.6, 2.2.
- invariants: I8: Accessed by is ordered by file, then name, then line, then role \| enforced at planned locationLess plus role key \| boundary test tasks 1.2 and 1.4; file-versus-name side missing (inv-2).
- invariants: I9: Multi-def section prints the trail header iff one of four lists is non-empty \| enforced at planned renderNodeSection \| boundary test task 1.6 cases 3, 4 and 5 (both sides).
- invariants: I10: UI gathers detail for exactly the first uiMultiDefCap candidates \| enforced at internal/uiserver/handlers.go:740 i < uiMultiDefCap \| boundary test task 3.2 case 3 (cap and cap+1).
- invariants: I11: New proto field numbers do not collide \| enforced at protoc and buf; NodeDefinition uses 1-5, GetNodeDetailResponse 1-9, no reserved range on either \| boundary test task 3.1 proto:drift.
- invariants: I12: Production internal/query reaches only goextract and nodeid of the indexer packages \| enforced at none for non-root packages (inv-1) \| boundary test none.
- invariants: I13: IsInitBinding is exactly the metadata check MetaBinding == BindingInit \| enforced at the planned helper replacing inline checks at resolve/bindings.go:30,88, collapse.go:102, sync_bindings.go:98,110 \| boundary test task 1.5 cases 1 and 2 (both sides).
- invariants: No enforcement point is a script, so no probe was run.
- spec-lint: Delta spec, requirement 'The graph UI carries the data lines': the MUST clause 'The browse view MUST show each of the two lists that has an entry' has no scenario. The only scenario checks the GetNodeDetail response fields. Form is valid (one scenario with WHEN and THEN); the browse-view clause is untested by any scenario.
- spec-lint: Delta spec, requirement 'The data lines appear only for data symbols': the MUST clause that an empty list prints no line in the multi-definition view has no scenario of its own. The requirement has three other scenarios.
- spec-lint: Delta spec, requirement 'Node lists the readers and writers of a data symbol': the MUST clause 'once' (each source listed once) has no scenario. The scenario 'Readers in two files' uses a '...' placeholder for the line numbers.
- spec-lint: Checked and clean: every requirement has a scenario with WHEN or GIVEN and THEN; every task names a command, test, file or observable; the targets task proto:gen, proto:drift, golden:regen, test:golden and changie exist in Taskfile.yml; pnpm scripts test and check exist in web/package.json; no BREAKING marker in proposal.md or the delta spec; design.md has no open-questions section, so no assumed-answer check applied.
- test-coverage: Spec has 8 scenarios; all matrix rows are named by a task with a specific test. Tasks 2.2 and 3.2 name the test (TestNodeShowsDataAccess, TestGetNodeDetailDataAccess) but not a go test command line; the test name is treated as the verification command.
- test-coverage: design.md was not read; the matrix needs only the spec and tasks.
- test-coverage: No BREAKING requirement in the proposal; no breaking-unproven finding.
