---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-14T03:59:43+00:00
head: adcff4c8e30ac8511155a51fd22985f956331057
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
| app-1 | approach | should-fix | The Python extractor detects a descriptor only when its class is in the same file, so a class attribute that holds an imported descriptor becomes an ordinary field and its reads resolve to it. design.md says to leave descriptors unresolved and records no deviation; only the capability doc names the limit. | design.md > Decisions — "Leave descriptors, dynamic method resolution, and ambiguous class/instance storage unresolved." | Record the same-file descriptor limit as a deviation in design.md beside the Python resolution decision, or give no definitive field target to an attribute whose initializer instantiates an imported or unresolved class. |
| app-2+app-3 | approach | should-fix | app-2: The definition detail behind CLI and MCP node carries only the owner, calls and callers. It holds no data references and no binding path, so a definition shows neither a field's readers and writers nor a variable's bound function, which the design says definitions expose.; app-3: Explore drops every field as a gather candidate and leaves every read or write of a field out of the ranking graph. design.md has no decision for either choice, and it says explore exposes the data-reference path. | design.md > Decisions — "Definitions and explore expose the binding and data-reference path." | app-2: Add the data references with their access role, and the binding reference of a data symbol, to the definition detail. Otherwise record in design.md that definitions expose only owner and call trail, and name where the reference path is visible.; app-3: Record the exclusion of fields from gather and ranking as a decision in design.md with its ranking reason, and state how explore still shows a field's references. |
| app-4+inv-2 | approach,invariants | should-fix | app-4: The keyed value of a Go composite literal is captured only as a read, not as a write to the struct field. A write through a receiver of unknown type stays unresolved and leaves no binding edge, and Go fields are never treated as open. A Go field with one known write can then establish a definitive call target despite these competing writes.; inv-2: The design says an augmented assignment withholds alias resolution. The resolver binds no value for a read_write access, but no test applies an augmented assignment to a callable binding and expects the call to be withheld. | design.md > Decisions — "A competing write, augmented assignment, unknown target assignment, or relevant parse failure prevents definitive alias " | app-4: Record a keyed composite-literal element as a write of its value to the field. Treat a Go field with an unresolved write of the same name as holding an unknown value, or record the limit as a deviation in design.md.; inv-2: In TestCallableBindingResolution or TestPythonDataReferences, bind a module variable to a function, apply an augmented assignment to it (Python `handler += other`, TS `f ??= g` or `f \|\|= g`), call it and assert the caller has a read reference and no calls edge. |
| app-5 | approach | should-fix | The TypeScript extractor declares no field for a quoted class field name, which is a static name. The design matrix includes class fields with static names and excludes only computed property expressions. | design.md > Decisions — "Module `var`, `let`, `const`, identifier destructuring, class fields with static names, private fields" | Emit a field for a string-literal class field name, or add quoted names to the excluded forms in the design matrix. |
| app-6 | approach | should-fix | The design Context says the Go extractor does not expose struct fields as data nodes. At head, goextract emits a field node for each named struct member. | design.md > Context — "emits package variables but does not expose struct fields as data nodes" | Reword the Context sentence as the state before this change, for example 'before this change, the Go extractor did not expose struct fields'. |
| app-7 | approach | should-fix | The design Context says the TypeScript, Python and Java extractors do not provide the declaration contract. At head they emit fields and module bindings, for example the Java extractor's field nodes. | design.md > Context — "do not provide the full proposed declaration contract." | Reword the Context sentence as the state before this change. |
| fit-1 | fit | should-fix | ownerDependents hand-writes a visited-guarded ancestor walk over idx.supers, which symbols.WalkSupertypes already does over the same map[string][]string shape and which the resolver uses for every other supertype walk. | internal/indexer/sync_owners.go:77 — "for len(queue) > 0 {" | For each seed type call symbols.WalkSupertypes(id, idx.supers, func(s string) bool { types[s] = true; return false }) and drop the queue loop. |
| fit-2 | fit | should-fix | bindingDependents and ownerDependents each end with a hand-written collect-and-sort of a map[string]bool, which the same package already provides as sortedBoolKeys. | internal/indexer/sync_bindings.go:133 — "out := make([]string, 0, len(set))" | Return sortedBoolKeys(set) from both functions. |
| fit-3 | fit | should-fix | The sync keys the syntax-error scope of a file by its directory, while the resolver keys the same rule by the file's module key; the design names the module as the Go package and the Python, TypeScript or JavaScript file, so two mechanisms define one concept. | internal/indexer/sync_bindings.go:39 — "dirs[filepath.Dir(r.RelPath)] = true" | Key the damaged set in bindingDependents by the module key (FileResult.ImportPath for new results, the declared or discovered module key that storeview.go computes for stored files), or name in design.md why the sync uses the directory instead. |
| inv-1 | invariants | should-fix | The spec requires a JavaScript or TypeScript binding named _ to remain a symbol, and no test declares one. | specs/data-symbols/spec.md > Requirement: Module and package data declarations — "Python, JavaScript, and TypeScript `_` identifiers within this declaration scope MUST remain symbols." | Add `const _ = 1;` and a destructured `_` to the ts, js and tsx sources of TestTypeScriptDataDeclarations and expect a variable or constant named _. |
| inv-3 | invariants | should-fix | Owner facts compare unequal for a class whose supertype is not indexed (Exception, a JDK or node_modules base). The stored side has no edge for an unresolved supertype, but the new side records its name. Every edit of such a file then adds owner dependents, and no test covers that case. | design.md > Decisions — "A change that leaves the types, supertypes, fields and field visibility of a file the same adds no owner dependent." | Make storedOwnerFacts and newOwnerFacts agree for an unresolved supertype: either record only supertypes that resolve, or read the stored unresolved names. Then add a sync test that edits only a method body of `class E(Exception)` or `class D extends Unindexed` and asserts that ownerDependents returns no path. |
| ctl-not-run | test-coverage | should-fix | covered row 'Grouped and destructured bindings' has no verification run | review > not-run — "internal/indexer/goextract/datadecl_test.go: TestGoDataDeclarations; internal/indexer/tsextract/datadecl_test.go: TestTy" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A Python module binding has only an annotation' has no verification run | review > not-run — "internal/indexer/pyextract/datadecl_test.go: TestPythonDataDeclarations (pending: list)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Multiple fields share a declaration' has no verification run | review > not-run — "internal/indexer/goextract/datadecl_test.go: TestGoDataDeclarations; internal/indexer/javaextract/datadecl_test.go: Test" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Python instance attribute repeats' has no verification run | review > not-run — "internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Shape/Other width)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A Python instance write shadows a base class attribute' has no verification run | review > not-run — "internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Child.__init__)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A field name is computed dynamically' has no verification run | review > not-run — "internal/indexer/tsextract/datadecl_test.go: TestTypeScriptDataDeclarations; internal/indexer/tsdatarefs_test.go: TestTy" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A field and method share a name' has no verification run | review > not-run — "internal/indexer/dataidentity_test.go: TestDataSymbolIdentity (run and sync)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Function value assignment' has no verification run | review > not-run — "internal/indexer/datarefs_test.go: TestGoDataReferences (daemonList)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Initializer invokes a function' has no verification run | review > not-run — "internal/indexer/datarefs_test.go: TestGoDataReferences (initial); internal/indexer/javadatarefs_test.go: TestJavaDataRe" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A local binding shadows a module variable' has no verification run | review > not-run — "internal/indexer/datarefs_test.go: TestGoDataReferences (shadow); tsdatarefs_test.go (scoped); pydatarefs_test.go (shado" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'One function reads and writes a field' has no verification run | review > not-run — "internal/indexer/datarefs_test.go: TestGoDataReferences (Move/both); internal/indexer/resolve/metadata_test.go: TestData" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'An inherited Java, TypeScript, JavaScript, or TSX field is read' has no verification run | review > not-run — "internal/indexer/javadatarefs_test.go: TestJavaDataReferences; internal/indexer/tsdatarefs_test.go: TestTypeScriptDataRe" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A Python descriptor controls an attribute' has no verification run | review > not-run — "internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Model.show, Model.label)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A unique imported function binding is invoked' has no verification run | review > not-run — "internal/indexer/bindings_test.go: TestCallableBindingResolution (callImported); TestBindingsWithheldBySyntaxErrors (cle" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A mutable binding has competing targets' has no verification run | review > not-run — "internal/indexer/bindings_test.go: TestCallableBindingResolution (callSwapped, callLiteral)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A mutable function-kind binding changes value' has no verification run | review > not-run — "internal/indexer/tsdatarefs_test.go: TestTypeScriptDataReferences (callSwap)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A data binding is invoked' has no verification run | review > not-run — "internal/indexer/bindings_test.go: TestCallableBindingResolution (callsInto variables, run -> daemonList read)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Direct and alias calls share endpoints' has no verification run | review > not-run — "internal/indexer/bindings_test.go: TestCallableBindingResolution (both); internal/indexer/resolve/metadata_test.go: Test" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Tests replace the daemon function binding' has no verification run | review > not-run — "internal/cli/function_variable_test.go: TestFunctionVariableQueries (production binding); internal/indexer/bindings_test" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Production code replaces the binding' has no verification run | review > not-run — "internal/cli/function_variable_test.go: TestFunctionVariableQueries (production reassignment)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A user requests a field definition' has no verification run | review > not-run — "internal/cli/data_symbol_queries_test.go: TestDataSymbolPublicQueries; internal/mcp/data_symbols_test.go: TestDataSymbol" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A user asks for callers of a referenced function' has no verification run | review > not-run — "internal/cli/function_variable_test.go: TestFunctionVariableQueries (callers List excludes daemonList)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A field moves to another owner' has no verification run | review > not-run — "internal/indexer/datasync_inherit_test.go: TestDataSymbolSyncEquivalence_Inheritance (typescript, the field moves to ano" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Another file adds then removes a competing write' has no verification run | review > not-run — "internal/indexer/datasync_test.go: TestDataSymbolSyncEquivalence (add/remove a competing write)" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'A subclass gains a shadowing field' has no verification run | review > not-run — "internal/indexer/datasync_inherit_test.go: TestDataSymbolSyncEquivalence_Inheritance (subclass gains a shadowing field, " | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Unsupported forms appear beside supported fields' has no verification run | review > not-run — "testdata/semantic/development/java/fields-beside-enum/case.json (known-gap with exception) run by tools/semantic-eval" | run the covering test and record it in verification-runs.txt |
| tes-1 | test-coverage | should-fix | No test asserts that an exported TypeScript arrow function gains no duplicate data node; the declaration test exports only a function expression. | specs/data-symbols/spec.md > Scenario: Existing exported arrow function — "its existing callable identity remains the single declaration identity" | In internal/indexer/tsextract/datadecl_test.go TestTypeScriptDataDeclarations, add `export const exportedArrow = () => 1;` and assert count of function exportedArrow is 1 and of constant/variable exportedArrow is 0. |
| tes-2 | test-coverage | should-fix | The nonexported function-expression case asserts one function node, not the absence of a duplicate constant node for the same name. | specs/data-symbols/spec.md > Scenario: A module callable is not exported — "that declaration has one function identity" | In TestDataReferenceMetadata_KindsAndStorage assert declaredNodes(m.js, KindConstant, "expr") and KindVariable return zero nodes. |
| hon-1 | test-honesty | should-fix | TestDataSymbolIdentity computes the expected field ids with nodeid.DeclID, the helper that production uses to allocate them, so a wrong id scheme still passes. | internal/indexer/dataidentity_test.go:46 — "fieldID := nodeid.DeclID(goextract.KindField, "Box.size", box, "")" | Assert id stability independent of the helper: compare the field ids before and after the line-moving sync, and assert field and method ids differ. Or pin literal expected ids. |
| hon-2 | test-honesty | should-fix | Several inheritance sync steps have no check, so only fresh-vs-sync equality is tested. The private-field and subclass-replaces-callable outcomes are never asserted. | internal/indexer/datasync_inherit_test.go:212 — "{"the base field becomes private", map[string]*string{"p/Base.java": text("package p;\\n\\npublic class Base {\\n    privat" | Give these steps a check, for example readsCount(javaReader, "", "") after the field becomes private, and an assertion that the callable-field call edge is absent or restored. |
| fit-4 | fit | note | captureInitializer, mayHoldFunction, valueRef and captureWrite repeat across the Go, TypeScript and Python extractors with the same control flow; they differ in the grammar node kinds and the shadow lookup only. | internal/indexer/tsextract/data.go:195 — "goextract.BindInitializer(ex.result, ex.fileID, fromID, before, whole, ex.valueRef(value) != nil, mayHoldFunction(value)" | If a further language joins, move the walk into goextract with the parameters: the set of non-running body kinds, the call kind, the set of literal kinds that cannot hold a function, and the bare-name shadow lookup. |
| inv-4 | invariants | note | Collapse merges access, binds, via and context as sets, but the initializer marker binding=init stays only on the representative. The order-free property test neither draws nor compares that key, so it does not check that the marker survives a merge with a tie on line. | design.md > Decisions — "Preserve initializer binding evidence with an additive metadata key." | With pgregory.net/rapid, as in TestCollapseEdges_MergeIsOrderFree (internal/indexer/resolve/metadata_test.go) and TestSpillRoundTripProperty (internal/indexer/spill_roundtrip_test.go), draw MetaBinding on references candidates and add it to the compared keys. Merge it in withMergedAccess if the property fails. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Grouped and destructured bindings | internal/indexer/goextract/datadecl_test.go: TestGoDataDeclarations; internal/indexer/tsextract/datadecl_test.go: TestTypeScriptDataDeclarations; testdata/semantic/development/go/grouped-bindings/case.json | covered | not-run |
| Existing exported arrow function | internal/indexer/tsextract/datadecl_test.go: TestTypeScriptDataDeclarations (function expression only) | partial | not-run |
| Modifiers do not invent new declaration kinds | internal/indexer/datakinds_test.go: TestDataReferenceMetadata_KindsAndStorage | covered | pass |
| A module callable is not exported | internal/indexer/datakinds_test.go: TestDataReferenceMetadata_KindsAndStorage | partial | pass |
| A Python module binding has only an annotation | internal/indexer/pyextract/datadecl_test.go: TestPythonDataDeclarations (pending: list) | covered | not-run |
| Multiple fields share a declaration | internal/indexer/goextract/datadecl_test.go: TestGoDataDeclarations; internal/indexer/javaextract/datadecl_test.go: TestJavaDataDeclarations | covered | not-run |
| Python instance attribute repeats | internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Shape/Other width) | covered | not-run |
| A Python instance write shadows a base class attribute | internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Child.__init__) | covered | not-run |
| A field name is computed dynamically | internal/indexer/tsextract/datadecl_test.go: TestTypeScriptDataDeclarations; internal/indexer/tsdatarefs_test.go: TestTypeScriptDataReferences | covered | not-run |
| A field and method share a name | internal/indexer/dataidentity_test.go: TestDataSymbolIdentity (run and sync) | covered | not-run |
| Function value assignment | internal/indexer/datarefs_test.go: TestGoDataReferences (daemonList) | covered | not-run |
| Initializer invokes a function | internal/indexer/datarefs_test.go: TestGoDataReferences (initial); internal/indexer/javadatarefs_test.go: TestJavaDataReferences (Box.size initializer) | covered | not-run |
| A local binding shadows a module variable | internal/indexer/datarefs_test.go: TestGoDataReferences (shadow); tsdatarefs_test.go (scoped); pydatarefs_test.go (shadow) | covered | not-run |
| One function reads and writes a field | internal/indexer/datarefs_test.go: TestGoDataReferences (Move/both); internal/indexer/resolve/metadata_test.go: TestDataReferenceMetadata | covered | not-run |
| An inherited Java, TypeScript, JavaScript, or TSX field is read | internal/indexer/javadatarefs_test.go: TestJavaDataReferences; internal/indexer/tsdatarefs_test.go: TestTypeScriptDataReferences (ts, js, tsx) | covered | not-run |
| A Python descriptor controls an attribute | internal/indexer/pydatarefs_test.go: TestPythonDataReferences (Model.show, Model.label) | covered | not-run |
| A unique imported function binding is invoked | internal/indexer/bindings_test.go: TestCallableBindingResolution (callImported); TestBindingsWithheldBySyntaxErrors (clean.ts) | covered | not-run |
| A mutable binding has competing targets | internal/indexer/bindings_test.go: TestCallableBindingResolution (callSwapped, callLiteral) | covered | not-run |
| A mutable function-kind binding changes value | internal/indexer/tsdatarefs_test.go: TestTypeScriptDataReferences (callSwap) | covered | not-run |
| A data binding is invoked | internal/indexer/bindings_test.go: TestCallableBindingResolution (callsInto variables, run -> daemonList read) | covered | not-run |
| Direct and alias calls share endpoints | internal/indexer/bindings_test.go: TestCallableBindingResolution (both); internal/indexer/resolve/metadata_test.go: TestCollapseEdges_MergeIsOrderFree | covered | not-run |
| Tests replace the daemon function binding | internal/cli/function_variable_test.go: TestFunctionVariableQueries (production binding); internal/indexer/bindings_test.go: TestCallableBindingResolution (TestRun) | covered | not-run |
| Production code replaces the binding | internal/cli/function_variable_test.go: TestFunctionVariableQueries (production reassignment) | covered | not-run |
| A user requests a field definition | internal/cli/data_symbol_queries_test.go: TestDataSymbolPublicQueries; internal/mcp/data_symbols_test.go: TestDataSymbolPublicQueries | covered | not-run |
| A user asks for callers of a referenced function | internal/cli/function_variable_test.go: TestFunctionVariableQueries (callers List excludes daemonList) | covered | not-run |
| An unchanged source file gains extraction support | internal/indexer/datarebuild_test.go: TestDataSymbolRebuild | covered | pass |
| A field moves to another owner | internal/indexer/datasync_inherit_test.go: TestDataSymbolSyncEquivalence_Inheritance (typescript, the field moves to another class); datasync_test.go (move the struct) | covered | not-run |
| Another file adds then removes a competing write | internal/indexer/datasync_test.go: TestDataSymbolSyncEquivalence (add/remove a competing write) | covered | not-run |
| A subclass gains a shadowing field | internal/indexer/datasync_inherit_test.go: TestDataSymbolSyncEquivalence_Inheritance (subclass gains a shadowing field, ts, python, java) | covered | not-run |
| Unsupported forms appear beside supported fields | testdata/semantic/development/java/fields-beside-enum/case.json (known-gap with exception) run by tools/semantic-eval | covered | not-run |

### Not run

- Grouped and destructured bindings: no verification run names this row
- Existing exported arrow function: no exported arrow, no duplicate-absence assertion
- A Python module binding has only an annotation: no verification run names this row
- Multiple fields share a declaration: no verification run names this row
- Python instance attribute repeats: no verification run names this row
- A Python instance write shadows a base class attribute: no verification run names this row
- A field name is computed dynamically: no verification run names this row
- A field and method share a name: no verification run names this row
- Function value assignment: no verification run names this row
- Initializer invokes a function: no verification run names this row
- A local binding shadows a module variable: no verification run names this row
- One function reads and writes a field: no verification run names this row
- An inherited Java, TypeScript, JavaScript, or TSX field is read: no verification run names this row
- A Python descriptor controls an attribute: no verification run names this row
- A unique imported function binding is invoked: no verification run names this row
- A mutable binding has competing targets: no verification run names this row
- A mutable function-kind binding changes value: no verification run names this row
- A data binding is invoked: no verification run names this row
- Direct and alias calls share endpoints: no verification run names this row
- Tests replace the daemon function binding: no verification run names this row
- Production code replaces the binding: no verification run names this row
- A user requests a field definition: no verification run names this row
- A user asks for callers of a referenced function: no verification run names this row
- A field moves to another owner: no verification run names this row
- Another file adds then removes a competing write: no verification run names this row
- A subclass gains a shadowing field: no verification run names this row
- Unsupported forms appear beside supported fields: no verification run names this row

## Verification runs

- exit 0: `go test -count=1 -run ^TestDataSymbolPublicQueries$ ./internal/cli/`
- exit 0: `go test -count=1 -run ^TestDataSymbolPublicQueries$ ./internal/mcp/`
- exit 0: `go test -count=1 -run ^TestFunctionVariableQueries$ ./internal/cli/`
- exit 0: `go test -count=1 -run ^TestCallableBindingResolution$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestBindingsWithheldBySyntaxErrors$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestDataSymbolIdentity$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestDataReferenceMetadata_KindsAndStorage$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestDataSymbolRebuild$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestGoDataReferences$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestJavaDataReferences$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestTypeScriptDataReferences$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestPythonDataReferences$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestDataSymbolSyncEquivalence_Inheritance$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestDataSymbolSyncEquivalence$ ./internal/indexer/`
- exit 0: `go test -count=1 -run ^TestCollapseEdges_MergeIsOrderFree$ ./internal/indexer/resolve/`
- exit 0: `go test -count=1 -run ^TestDataReferenceMetadata$ ./internal/indexer/resolve/`
- exit 0: `go test -count=1 -run ^TestGoDataDeclarations$ ./internal/indexer/goextract/`
- exit 0: `go test -count=1 -run ^TestJavaDataDeclarations$ ./internal/indexer/javaextract/`
- exit 0: `go test -count=1 -run ^TestTypeScriptDataDeclarations$ ./internal/indexer/tsextract/`
- exit 0: `go test -count=1 -run ^TestPythonDataDeclarations$ ./internal/indexer/pyextract/`
- exit 0: `mise exec -- task eval:semantic`

## Dropped

None.

## Lens notes

- approach: tasks.md 6.2 says dependencies persist through schema sources and generators. design.md says the schema gains no field for them and that File.syntax_errors is the only new persisted fact. graph.proto at head matches design.md (File.syntax_errors = 11, Meta.has_data_symbols = 16).
- approach: Cited facts that hold at head: nodeid.Allocator.Finalize, has_unique_decl_ids = 15, the has_data_symbols marker that resolve.go sets and sync.go checks, the daemonList binding in internal/cli/daemon.go and its stub in daemon_test.go, the access, binding_context and via metadata keys, and the go-production and go-test contexts.
- approach: The bindings pass also withholds a target for a field in an inheriting language whose owner has subtypes (bindings.go open). This is conservative and does not conflict with a decision, but design.md does not name it.
- approach: The rollback decision names the prior binary and a fresh index, so no no-rollback finding applies.
- fit: No repository library-preference path was named in the prompt; the generic rows of the rulebook were applied.
- fit: The capability descriptor in internal/indexer/capability/matrix.go and docs/LANGUAGE-CAPABILITY-MATRIX.md are a sanctioned mirror held by TestMatrix_DocMirrorsDescriptor; the data-symbol gap lines read match.
- fit: Components that fit their closest pattern: the field kind joins knownKinds in query/validate.go; BareNameCandidate keeps fields out of both the fresh and the store-seeded bare-name index; data declarations take ids through the shared nodeid allocator (alloc.Declare); access and evidence merge inside the existing CollapseEdges seam; has_data_symbols follows the has_unique_decl_ids rebuild path; field owners in query/detail.go reuse buildContainsIndex; ownerDependents reuses filesWithUnresolved.
- fit: The proposal names no follow-up change, so no forward-misfit was assessed.
- invariants: I1: A binding path holds at most maxBindingChain=8 data symbols \| enforced at internal/indexer/resolve/bindings.go:118 \| boundary test TestCallableBindingResolution eight1 and nine1 chains
- invariants: I2: A binding cycle establishes no target \| enforced at internal/indexer/resolve/bindings.go:119 \| boundary test TestCallableBindingResolution loopA/loopB
- invariants: I3: A write of an unknown value, or a write from a damaged module, withholds the target \| enforced at internal/indexer/resolve/bindings.go:98 and :131 \| boundary test TestBindingsWithheldBySyntaxErrors
- invariants: I4: A field never enters the bare-name index, fresh or store-seeded \| enforced at internal/indexer/symbols/candidate.go BareNameCandidate, used in builder.go addSymbols and storeview.go \| boundary test TestDataSymbolIdentity
- invariants: I5: Access roles merge to read_write and evidence sets merge order-free \| enforced at internal/indexer/resolve/collapse.go withMergedAccess and goextract MergeAccess \| boundary test TestCollapseEdges_MergeIsOrderFree
- invariants: I6: has_data_symbols false or absent forces a rebuild \| enforced at internal/indexer/sync.go:987 needsFileIndexBackfill \| boundary test TestDataSymbolRebuild
- invariants: I7: Proto field numbers File.syntax_errors=11 and Meta.has_data_symbols=16 are the next free numbers on main \| enforced at internal/schema/graph.proto \| boundary test none needed
- invariants: I8: Two supertype paths with different suppliers resolve nothing, a diamond to one supplier resolves \| enforced at internal/indexer/resolve/resolver.go supplier \| boundary test TestPythonInheritedFieldOwnership
- invariants: I9: Go _ never declares a symbol \| enforced at internal/indexer/goextract/goextract.go:271,527,1097 \| boundary test TestGoDataDeclarations const c, _
- invariants: I10: Python _ remains a symbol \| enforced by absence of a filter in pyextract \| boundary test TestPythonDataDeclarations
- invariants: I11: A function-kind binding with any foreign write yields a read, not a call \| enforced at internal/indexer/resolve/bindings.go holdsOnlyItself \| boundary test tsdatarefs_test.go reassigned let function (line 83 comment)
- invariants: I12: A field of a class with a subclass is open and establishes no target \| enforced at internal/indexer/resolve/bindings.go open via Hierarchy.HasSubtypes \| boundary test TestDataSymbolSyncEquivalence_Inheritance python 'a subclass can replace the callable field'
- invariants: I13: Go production context ignores _test.go writes, and a test caller uses go-test only \| enforced at internal/indexer/resolve/bindings.go derive \| boundary test TestCallableBindingResolution check(run, go-production) and check(TestStable, go-test)
- invariants: I14: Binding-dependent seed growth and sync waves terminate \| enforced by monotone seeds in sync_bindings.go:104 and the inBatch set in sync.go wave loop \| boundary test TestDataSymbolSyncEquivalence add/remove competing write
- invariants: I15: Fields never rank on their own \| enforced at internal/query/rwr.go isRankEdge \| boundary test not checked
- invariants: I16: A provisional occurrence id never reaches the store \| enforced at internal/indexer/nodeid/allocator.go Finalize scan \| boundary test TestDataSymbolIdentity
- invariants: I17: Spilled results keep SyntaxErrors and the new UnresolvedRef fields \| enforced by the gob codec in internal/indexer/spill/codec.go \| boundary test TestSpillRoundTrip positive field assertion plus TestSpillRoundTripProperty
- invariants: No script enforcement point exists in this change; every enforcement point is Go code, so no edge-input probe was run.
- invariants: Java results set no SyntaxErrors; the design limits the damaged-module rule to Go, Python, TypeScript and JavaScript, and a call through a Java field derives a call only in code that does not compile.
- test-coverage: Semantic cases are covered by the evaluation harness (tools/semantic-eval); I did not run it and saw no recorded result, so the controller must run it.
- test-coverage: Task 5.3 web case exists: web/tests/definition-picker.test.ts renders a field beside a method of the same name.
- test-coverage: TestPythonInheritedFieldOwnership and TestBindingsWithheldBySyntaxErrors have no task of their own but follow the bounded-inheritance and syntax-error rules of the spec; not orphans.
- test-coverage: No BREAKING requirement in the delta spec.
- test-honesty: No new skipped or disabled tests in the diff files.
- test-honesty: Tests are mostly end-to-end through the indexer and store, with independent literal expectations.
- test-honesty gap: Did not read goextract_test.go, tsextract_test.go, pyextract_test.go, javaextract_test.go, detail_test.go, render_markdown_test.go, errors_test.go, present/node_test.go diffs in full, nor the semantic case.json fixtures and tools/semantic-eval; no diff tool available.
