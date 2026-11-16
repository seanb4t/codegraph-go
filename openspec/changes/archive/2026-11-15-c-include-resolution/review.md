---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-16T02:38:33+00:00
head: e77bfdd308ceede68b87521c1703471e2354cfa9
round: 6
next_id: 70
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

| id | lens | severity | scope | claim | evidence | action | mark | judge |
|---|---|---|---|---|---|---|---|---|
| R-69 | invariants | suggestion | in-change | mergeSortedPaths claims a sorted union with no path twice over any number of path lists, an unbounded domain, and TestMergeSortedPaths pins it with five fixed cases only. | internal/indexer/mergepaths_test.go:18 — "{"a path in several lists", []string{"b"}, [][]string{{"a", "b"}, {"b", "c"}, {"a"}}, []string{"a", "b", "c"}}," | Add a pgregory.net/rapid property in internal/indexer/mergepaths_test.go, in the shape of internal/indexer/symbols/freeze_test.go: draw a sorted duplicate-free a and up to four lists of paths from a small alphabet, and assert that the result equals the sorted set of all inputs and is unchanged when more is permuted. |  |  |

## Repeats

| id | lens | severity | claim | evidence |
|---|---|---|---|---|
| R-18 | test-coverage | should-fix | covered row 'A function defined in a header' has no verification run | review > not-run — "A function defined in a header" |
| R-32 | test-coverage | should-fix | covered row 'The repro from #259' has no verification run | review > not-run — "The repro from #259" |

## Out of bound

None.

## Accepted

| id | severity | disposition | reason | ticket |
|---|---|---|---|---|
| R-63 | should-fix | accepted | The round 5 diff since 7a4f0eb1 changes no test file, only sync.go and two comments, so the unread test files are the ones rounds 1 to 4 reviewed. |  |
| R-65 | should-fix | accepted | Rounds 1 and 2 reviewed those six test files for fit; round 3 only reviewed the follow-up diff since aa99aaac, which touches test helpers. |  |

## Rounds

| round | phase | kind | head | digest | verdict | new | repeats | regressions | out-of-bound | decision |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | implementation | full | 88a919de6b0ef9339cf9c05946f6f769af4de0d6 | 87512adec56fc61d6676b8817b1e860e055b0eb8ff2887613bee94733819d4df | approve-with-notes | 13 | 0 | 0 |  |  |
| 2 | implementation | follow-up | aa99aaac2a962bfcbcd6063c9d429decfa148d0c | 515c8eb3d6bf509c81419b8d7fc5da2887d94d21097b2657fe011cf33e825f13 | approve-with-notes | 0 | 2 | 0 | 6 |  |
| 3 | implementation | follow-up | a58b098da52bdd70fc9ba9b8706ed53c70d8557a | 02cd17bedaf6e447dcf94a6e6b81eb84fd235ada47a7d777ce6c916c73496161 | approve-with-notes | 2 | 2 | 0 | 1 |  |
| 4 | implementation | follow-up | 7a4f0eb1752a6a00ee81f3baef44c147407e52f5 | 02cd17bedaf6e447dcf94a6e6b81eb84fd235ada47a7d777ce6c916c73496161 | approve-with-notes | 0 | 2 | 0 | 1 | operator decision |
| 5 | implementation | follow-up | a6be85a668932ab0f2f3e7152ca462ce8370cec5 | 02cd17bedaf6e447dcf94a6e6b81eb84fd235ada47a7d777ce6c916c73496161 | approve-with-notes | 2 | 3 | 0 | 0 | operator decision |
| 6 | implementation | follow-up | e77bfdd308ceede68b87521c1703471e2354cfa9 | 02cd17bedaf6e447dcf94a6e6b81eb84fd235ada47a7d777ce6c916c73496161 | approve-with-notes | 1 | 2 | 0 | 0 | operator decision |

## Judge

judge: 2 questions, 0.000 USD

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A header beside the source file | internal/indexer/cinclude_test.go: TestCIncludeLinks/A header beside the source file | covered | pass |
| A path from the repository root | internal/indexer/cinclude_test.go: TestCIncludeLinks/A path from the repository root | covered | pass |
| A sibling include directory | internal/indexer/cinclude_test.go: TestCIncludeLinks/A sibling include directory | covered | pass |
| An absolute path | internal/indexer/cinclude_test.go: TestCIncludeLinks/An absolute path | covered | pass |
| A system header | internal/indexer/cinclude_test.go: TestCIncludeLinks/A system header | covered | pass |
| An inner dot segment | internal/indexer/cinclude_test.go: TestCIncludeLinks/An inner dot segment | covered | pass |
| A header that includes itself | internal/indexer/cinclude_test.go: TestCIncludeLinks/A header that includes itself | covered | pass |
| A function defined in a header | internal/indexer/cinclude_test.go: TestCIncludeCalls/A function defined in a header; test/integration/testdata/c_header_prototype.txtar: callers twice | covered | not-run |
| An include of an include | internal/indexer/cinclude_test.go: TestCIncludeCalls/An include of an include | covered | pass |
| A struct tag in a nearer header | internal/indexer/cinclude_test.go: TestCIncludeCalls/A struct tag in a nearer header | covered | pass |
| Include guards that form a cycle | internal/indexer/cinclude_test.go: TestCIncludeCalls/Include guards that form a cycle | covered | pass |
| The nearest header wins | internal/indexer/cinclude_test.go: TestCIncludeCalls/The nearest header wins | covered | pass |
| Two headers at one depth | internal/indexer/cinclude_test.go: TestCIncludeCalls/Two headers at one depth | covered | pass |
| A header at depth 1 and depth 2 | internal/indexer/cinclude_test.go: TestCIncludeCalls/A header at depth 1 and depth 2 | covered | pass |
| The nearest header has the wrong arity | internal/indexer/cinclude_test.go: TestCIncludeCalls/The nearest header has the wrong arity | covered | pass |
| Every candidate at the first depth has the wrong arity | internal/indexer/cinclude_test.go: TestCIncludeCalls/Every candidate at the first depth has the wrong arity | covered | pass |
| A name the file does not include | internal/indexer/cinclude_test.go: TestCIncludeCalls/A name the file does not include | covered | pass |
| A static helper | internal/indexer/mainstream/cextract/marks_test.go: TestCFunctionMarks/A static helper | covered | pass |
| A static forward declaration | internal/indexer/mainstream/cextract/marks_test.go: TestCFunctionMarks/A static forward declaration | covered | pass |
| A static declaration outside the definition's #if | internal/indexer/mainstream/cextract/marks_test.go: TestCFunctionMarks/A static declaration outside the definition's #if | covered | pass |
| A C++ static member function | internal/indexer/mainstream/cextract/marks_test.go: TestCFunctionMarks/A C++ static member function | covered | pass |
| The repro from #259 | internal/indexer/cinclude_test.go: TestCIncludeCalls/The repro from #259; test/integration/testdata/c_header_prototype.txtar: callers add | covered | not-run |
| A library prototype | internal/indexer/cinclude_test.go: TestCIncludeCalls/A library prototype | covered | pass |
| One definition for each platform | internal/indexer/cinclude_test.go: TestCIncludeCalls/One definition for each platform | covered | pass |
| A static definition of the same name | internal/indexer/cinclude_test.go: TestCIncludeCalls/A static definition of the same name | covered | pass |
| A prototype under an include guard | internal/indexer/cinclude_test.go: TestCIncludeCalls/A prototype under an include guard | covered | pass |
| Pointer spacing and unnamed parameters | internal/indexer/cinclude_test.go: TestCIncludeCalls/Pointer spacing and unnamed parameters | covered | pass |
| Two definitions under #if in one file | internal/indexer/cinclude_test.go: TestCIncludeCalls/Two definitions under #if in one file | covered | pass |
| A C++ default argument | internal/indexer/cinclude_test.go: TestCIncludeCalls/A C++ default argument | covered | pass |
| A static prototype | internal/indexer/cinclude_test.go: TestCIncludeCalls/A static prototype | covered | pass |
| A definition in the prototype's own file | internal/indexer/cinclude_test.go: TestCIncludeCalls/A definition in the prototype's own file | covered | pass |
| A C++ method of the same name | internal/indexer/cinclude_test.go: TestCIncludeCalls/A C++ method of the same name | covered | pass |
| Another signature | internal/indexer/cinclude_test.go: TestCIncludeCalls/Another signature | covered | pass |
| An array parameter with and without a name | internal/indexer/cinclude_test.go: TestCIncludeCalls/An array parameter with and without a name | covered | pass |
| A function-pointer parameter with and without names | internal/indexer/cinclude_test.go: TestCIncludeCalls/A function-pointer parameter with and without names | covered | pass |
| Pointer spacing in one file | internal/indexer/cinclude_test.go: TestCIncludeCalls/Pointer spacing in one file | covered | pass |
| A default argument in one file | internal/indexer/cinclude_test.go: TestCIncludeCalls/A default argument in one file | covered | pass |
| A definition appears after the call | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A definition appears after the call | covered | pass |
| A header loses its prototype | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header loses its prototype | covered | pass |
| An include is removed | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/An include is removed | covered | pass |
| A header appears after its include | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header appears after its include | covered | pass |
| An include cycle in a sync | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/An include cycle in a sync | covered | pass |
| A header is deleted | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header is deleted | covered | pass |
| A header moves to the repository root | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header moves to the repository root | covered | pass |
| An indirect header is deleted | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/An indirect header is deleted | covered | pass |
| A header appears at the root | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header appears at the root | covered | pass |
| A header's own include moves | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header's own include moves | covered | pass |
| A nearer header appears | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A nearer header appears | covered | pass |
| A definition loses its body | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A definition loses its body | covered | pass |
| A prototype gains its body | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A prototype gains its body | covered | pass |
| A header with no symbols | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A header with no symbols | covered | pass |
| A definition loses static | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A definition loses static | covered | pass |
| A second definition appears | internal/indexer/sync_cinclude_test.go: TestCIncludeSyncEquivalence/A second definition appears | covered | pass |
| The first sync after an upgrade | internal/indexer/cprototype_rebuild_test.go: TestCPrototypeRebuildOnce | covered | pass |
| The syncs after the rebuild | internal/indexer/cprototype_rebuild_test.go: TestCPrototypeRebuildOnce | covered | pass |

### Not run

- A function defined in a header: no verification run names this row
- The repro from #259: no verification run names this row

## Verification runs

- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/A header beside the source file" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/A path from the repository root" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/A sibling include directory" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/An absolute path" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/A system header" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/An inner dot segment" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeLinks/A header that includes itself" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A function defined in a header" ./internal/indexer && go test -count=1 -v -run "TestScripts/c_header_prototype" ./test/integration`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/An include of an include" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A struct tag in a nearer header" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Include guards that form a cycle" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/The nearest header wins" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Two headers at one depth" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A header at depth 1 and depth 2" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/The nearest header has the wrong arity" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Every candidate at the first depth has the wrong arity" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A name the file does not include" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCFunctionMarks/A static helper" ./internal/indexer/mainstream/cextract`
- exit 0: `go test -count=1 -v -run "TestCFunctionMarks/A static forward declaration" ./internal/indexer/mainstream/cextract`
- exit 0: `go test -count=1 -v -run "TestCFunctionMarks/A static declaration outside the definition's #if" ./internal/indexer/mainstream/cextract`
- exit 0: `go test -count=1 -v -run "$(printf %s 'TestCFunctionMarks/A C++ static member function' | sed 's/+/\\+/g')" ./internal/indexer/mainstream/cextract`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/The repro from #259" ./internal/indexer && go test -count=1 -v -run "TestScripts/c_header_prototype" ./test/integration`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A library prototype" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/One definition for each platform" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A static definition of the same name" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A prototype under an include guard" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Pointer spacing and unnamed parameters" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Two definitions under #if in one file" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "$(printf %s 'TestCIncludeCalls/A C++ default argument' | sed 's/+/\\+/g')" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A static prototype" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A definition in the prototype's own file" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "$(printf %s 'TestCIncludeCalls/A C++ method of the same name' | sed 's/+/\\+/g')" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Another signature" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/An array parameter with and without a name" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A function-pointer parameter with and without names" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/Pointer spacing in one file" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeCalls/A default argument in one file" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A definition appears after the call" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header loses its prototype" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/An include is removed" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header appears after its include" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/An include cycle in a sync" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header is deleted" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header moves to the repository root" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/An indirect header is deleted" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header appears at the root" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header's own include moves" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A nearer header appears" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A definition loses its body" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A prototype gains its body" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A header with no symbols" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A definition loses static" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCIncludeSyncEquivalence/A second definition appears" ./internal/indexer`
- exit 0: `go test -count=1 -v -run "TestCPrototypeRebuildOnce" ./internal/indexer`

## Dropped

None.

## Lens notes

- approach: Follow-up round since a6be85a6: the changed files are internal/indexer/mergepaths_test.go, internal/indexer/sync.go and internal/query/expand.go. No change artifact changed, so no artifact-cited finding can be inside the bound.
- approach: sync.go: the new waveDependents runs bindingDependents and includeDependents together on the first wave (reparsed plus deleted paths) and on each later wave's paths, which matches D6 item 1: includeDependents runs in each wave next to bindingDependents.
- approach: sync.go: the first wave still merges the filesWaitingOn, filesResolvingToChangedSets, emitter, reader and owner lists before the loop, so their files join the first wave and their includers re-resolve in the next one, as D6 says.
- approach: sync.go: factsOf compares IsExported, Prototype and CParamKey, the three fields that D6 item 2 names for the changed-key set.
- approach: expand.go: transitiveWalk is built on bfs.Levels and sorts a copy of each level, which matches D3 and the bfs.Levels contract that visit must not change its slice. expandBFS keeps its own loop, as D3 says.
- approach: Prior rows R-4, R-67 and R-68 (fixed) were checked at head: includeDependents returns sortedBoolKeys, TestMergeSortedPaths covers the all-empty, nil-first and repeated-path edges, and the two wave passes are written once in waveDependents. None of these defects came back.
- fit: Follow-up round since a6be85a6: read the three changed files, internal/indexer/sync.go (waveDependents, both wave call sites, mergeSortedPaths), internal/indexer/mergepaths_test.go and internal/query/expand.go (transitiveWalk).
- fit: waveDependents runs bindingDependents and includeDependents once for the first wave and once per later wave, so the double write-out of R-68 is gone; the first wave still adds outbid, emitters and ownerDependents, which only that wave computes.
- fit: mergeSortedPaths builds on the existing sortedKeys helper of sync.go. internal/cli/daemon.go unionPaths also unions path lists but keeps input order on purpose, so it is not a copy of the sorted merge.
- fit: transitiveWalk in expand.go stays on bfs.Levels and sorts each level with slices.Sorted, as design D3 says; no new walk copy is in the changed text.
- fit: The prompt named no path for repository library-preference rows, so only the generic rows of the rulebook were applied.
- invariants: Follow-up round since a6be85a6: the changed text is internal/indexer/mergepaths_test.go (new), the waveDependents refactor in internal/indexer/sync.go, and a doc comment of transitiveWalk in internal/query/expand.go. No changed text touches a requirement other than D6 and D3; the other requirements hold no changed text and are outside the bound.
- invariants: I1: mergeSortedPaths returns the sorted union of its inputs with no path twice \| enforced at internal/indexer/sync.go:1338 (set, then sortedKeys) \| boundary test TestMergeSortedPaths cases no other list, only empty lists, an empty first list, a path in several lists.
- invariants: I2: the first argument a of mergeSortedPaths is sorted and holds no path twice, since the early return gives it back unchanged \| enforced by each caller: filesWaitingOn output at sync.go:519, waveDependents output at sync.go:569, sortedBoolKeys of bindingDependents at sync.go:1333 \| boundary test TestMergeSortedPaths no other list and only empty lists (both sides of n == len(a)).
- invariants: I3: waveDependents returns a sorted list with no path twice and no path of inBatch \| enforced at sync_bindings.go:130 and sync_includes.go (the !inBatch filter, then sortedBoolKeys) and mergeSortedPaths \| boundary test none direct; reached through TestCIncludeSyncEquivalence and TestSyncEqualsFreshIndex.
- invariants: I4: no dependency pass writes inBatch, so running includeDependents before ownerDependents in the first wave keeps the same union as before \| enforced at sync_bindings.go:130, sync_owners.go:93 and 105, sync_includes.go (reads only) \| boundary test the 4.1 steps of TestCIncludeSyncEquivalence.
- invariants: I5: every wave, the first included, runs includeDependents on its own paths (design D6) \| enforced at sync.go:506 and sync.go:559 through waveDependents \| boundary test the 4.1 step A header's own include moves, for a C++ struct.
- invariants: I6: transitiveWalk never returns root, as its changed doc comment now says \| enforced at internal/bfs/bfs.go:12 (seen[start] set before the walk) \| boundary test internal/query/transitive_walk_test.go a cycle back to the root, and TestLevels a cycle back to the start.
- invariants: Prior R-67 (fixed): TestMergeSortedPaths now passes the all-empty edge, a nil a with a non-empty more, and a path repeated across slices of more; the defect is gone.
- invariants: Prior R-68 (fixed): both waves now call waveDependents, and the later wave no longer merges in two calls; the defect is gone.
- invariants: No weakened check: the first-wave union before the diff (waiting, outbid, more, readers, owners, includers) equals the one after, since readers now carries includers.
- test-coverage: Matrix graded by reading test names and the assertions of the call, link, mark and rebuild tests; no test was run.
- test-honesty: TestMergeSortedPaths has literal expected values and covers the early return, nil first list and cross-list duplicates; it can fail for those reasons. No other test changed in the follow-up.
- test-coverage gap: Follow-up round: sync.go and expand.go changes were not read line by line; mergepaths_test.go covers the all-empty, nil-first and repeated-path cases of mergeSortedPaths.

## Ledger

| id | first | last | severity | scope | location | disposition | basis | override | claim |
|---|---|---|---|---|---|---|---|---|---|
| R-1 | 1 | 1 | should-fix | in-change | ["artifact",["design.md","D1. The include graph lives in the symbol view"],"`Freeze`, the `resolve.Types` branch, `resolveCall`, `storeview.go`, `storeIndex`, `newNames` and `filesResolvingToChangedSets` call it"] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | D1 lists storeview.go and storeIndex among the callers of symbols.LinksIncludes, but neither calls it at head; both recognize includes only through resolve.IsIncludeEdge, and design.md records no deviation. DeclarationsOf, which does call it, is not in the list. |
| R-2 | 1 | 1 | should-fix | in-change | ["artifact",["design.md","D5. A view index of C and C++ definitions by name"],"It leaves out a `comment` node, the `default_value` subtree, and the name of each declarator."] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | D5 says paramKey leaves out the default_value subtree, but in an optional_parameter_declaration the `=` token is a sibling of default_value, not inside it. The code also drops that `=` token. D5 does not record this, and without it `int b = 0` would key as `int =`, so the default-argument scenarios would not follow from D5 as written. |
| R-3 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/mainstream/cextract/params.go",123] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | paramTokens keys its skip set by a hand-built nodeSpan (start, end, kind), where go-tree-sitter already gives each node a unique Id(), which the repository already uses as a map key. |
| R-4 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/sync_includes.go",33] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | includeDependents collects its set into a slice and sorts it by hand, which is the body of sortedBoolKeys that bindingDependents and ownerDependents, its neighbours, call for the same return. |
| R-5 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/resolve/resolver.go",700] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | The C struct-tag rule of D3 is written twice in resolveCall and throughIncludes, as two inline language-and-kind tests, while its sibling rule LinksIncludes is one named predicate in symbols. |
| R-6 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/mainstream/cextract/cextract.go",426] | refuted | freeFunctionName and its declarator chain came from #250 (commit ccd01870); this change only calls it, so a shared helper is outside its scope. |  | freeFunctionName repeats the declarator, findFunctionDeclarator and name-declarator steps that emitFreeFunctionLike runs on the same declaration a loop later, and that collectInlineMethods also runs. |
| R-7 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/cinclude_test.go",16] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | cinclude_test.go adds writeTree and a TempDir root in indexTree, which is what testtree.Write does, and the same change calls testtree.Write in sync_cinclude_test.go and cprototype_rebuild_test.go. |
| R-8 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/mainstream/cextract/params.go",104] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | In C++ paramKey reads only named children of parameter_list, where the grammar leaves '...' unnamed. So int f(int a, ...) keys equal to int f(int a), and a C++ variadic prototype can redirect to a non-variadic definition in another file. No TestParamKey case passes an ellipsis. |
| R-9 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/resolve/types.go",35] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | IsIncludeEdge compares the include mark to the value quoted, but its table test never passes an imports edge whose MetaInclude key is present with another value, the other side of that equality. |
| R-10 | 1 | 1 | should-fix | in-change | ["path","internal/indexer/cinclude_test.go",355] | fixed | aa99aaac2a962bfcbcd6063c9d429decfa148d0c |  | The overload_set scenario asserts only the target node, not that its members are the prototype and the definition in add.h. |
| R-18 | 1 | 6 | should-fix | in-change | ["artifact",["review","not-run"],"A function defined in a header"] | refuted | Round 6 ran both tests of the row and they passed: the cinclude_test.go subtest and TestScripts/c_header_prototype share one && run line in verification-runs.txt; the merge matches one key per row, so it cannot see a run of two tests. |  | covered row 'A function defined in a header' has no verification run |
| R-32 | 1 | 6 | should-fix | in-change | ["artifact",["review","not-run"],"The repro from #259"] | refuted | Round 6 ran both tests of the row and they passed: the cinclude_test.go subtest and TestScripts/c_header_prototype share one && run line in verification-runs.txt; the merge matches one key per row, so it cannot see a run of two tests. |  | covered row 'The repro from #259' has no verification run |
| R-63 | 1 | 5 | should-fix | in-change | ["artifact",["review","lens-status"],"test-honesty partial"] | accepted | ticket: none; reason: The round 5 diff since 7a4f0eb1 changes no test file, only sync.go and two comments, so the unread test files are the ones rounds 1 to 4 reviewed. |  | lens test-honesty reported partial, gaps: 1 |
| R-64 | 3 | 3 | should-fix | in-change | ["path","internal/testtree/testtree.go",12] | fixed | 7a4f0eb1752a6a00ee81f3baef44c147407e52f5 |  | The change adds testtree.T, the Helper and Fatalf subset of testing.TB and rapid.T, while its own paramkey_test.go declares the same two-method interface as fataler, which graphstore tests also declare. |
| R-65 | 3 | 3 | should-fix | in-change | ["artifact",["review","lens-status"],"fit partial"] | accepted | ticket: none; reason: Rounds 1 and 2 reviewed those six test files for fit; round 3 only reviewed the follow-up diff since aa99aaac, which touches test helpers. |  | lens fit reported partial, gaps: 1 |
| R-67 | 5 | 5 | should-fix | in-change | ["path","internal/indexer/sync.go",1337] | fixed | e77bfdd308ceede68b87521c1703471e2354cfa9 |  | The variadic mergeSortedPaths has a new early return when every slice of more is empty, and it dedupes across several slices, but no test calls it. No test passes the all-empty edge, a nil a with a non-empty more, or a path repeated across two slices of more. Only sync tests reach it, through a mid-range happy path. |
| R-68 | 5 | 5 | suggestion | in-change | ["path","internal/indexer/sync.go",524] | fixed | e77bfdd308ceede68b87521c1703471e2354cfa9 |  | The first wave and each later wave both run bindingDependents and then includeDependents with the same error handling and merge, and the later wave still merges in two calls. Only the owners, outbid and emitter lists differ, so D6's rule that includeDependents runs next to bindingDependents in each wave is written out twice instead of once. |
| R-69 | 6 | 6 | suggestion | in-change | ["path","internal/indexer/mergepaths_test.go",18] | refuted | The five table cases reach each branch of mergeSortedPaths, and the rapid sync property test drives the merge end to end through Sync against a fresh Run. |  | mergeSortedPaths claims a sorted union with no path twice over any number of path lists, an unbounded domain, and TestMergeSortedPaths pins it with five fixed cases only. |

## Plan review (2026-11-15)

Verdict approve-with-notes at head ed26f7a1c905a7da61dc841643affb05f739d966.

### Findings

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| app-1 | approach | should-fix | in-change | D5 drops each identifier at the declarator field of a declarator and names parenthesized_declarator and reference_declarator as such declarators, but at the pinned grammars neither has a declarator field. So the name in a C++ reference parameter such as int &r, and cb in void (&cb)(int) that task 3.0 asserts equal to void (&)(int), sit in an unnamed child and stay in the key, against the requirement that the match ignores parameter names. | design.md > D5. A view index of C and C++ definitions by name — "through `pointer_declarator`, `array_declarator`, `function_declarator`, `parenthesized_declarator` and `reference_decla" | State in D5 that the key drops the identifier that a declarator names, whether it sits at the declarator field or is the unnamed declarator child of reference_declarator or parenthesized_declarator, as cextract.innermostIdentifier already falls back to the first identifier child. Add int &r against int & to the TestParamKey table of task 3.0. |
| inv-2 | invariants | should-fix | in-change | paramKey treats a lone `(void)` as empty, but no TestParamKey case checks the other side of that rule: `(void *)` must not key as empty. | design.md > D5. A view index of C and C++ definitions by name — "A list of `(void)` keys as empty, as `collectParams` reads it." | In task 3.0, add TestParamKey cases that assert `(void *)` and `(void)` have unequal keys, and that `void (*)(void)` and `void (*)()` keep their inner lists apart. |
| fit-1 | fit | note | in-change | storeIndex.includers adds one more full-scan reverse adjacency of stored edges, next to the sync's buildReverseAdjacency and query.BuildReverseAdjacency. design.md names the copy and issue #274 tracks one shared builder, and task 4.1 puts includers in the scope of #274, so this is a note. | design.md > D6. Sync dependents — "It is one more reverse adjacency of stored edges, next to the cached `reverse()` of `sync.go` and `query.BuildReverseAdj" | Keep the 4.1 step that names storeIndex.includers in the PR body as part of #274, and add includers to the list in the #274 issue body when the PR merges. |

### Coverage matrix

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
