# Tasks

## 1. Shared data-symbol contract

- [ ] 1.1 Integrate with the identity finalizer after the `index-invariants` implementation proposed in PR #237 lands. Cover new field identities and remapped references in `TestDataSymbolIdentity`. Document the shared result contract.
- [ ] 1.2 Add the field kind and deterministic read/write metadata merging. Cover serialization, accepted kinds, and combined access in `TestDataReferenceMetadata`.
- [ ] 1.3 Add independently annotated cross-language fixtures to #238's evaluation inputs. Review the declaration and reference inventory against `specs/data-symbols/spec.md`.

## 2. Go data declarations and references

- [ ] 2.1 Extend `internal/indexer/goextract` for named struct fields and complete package declaration lists. Cover grouped declarations, blank identifiers, and owners in `TestGoDataDeclarations`.
- [ ] 2.2 Add initializer references and scoped data reads and writes. Cover receiver types, local shadowing, and actual initializer calls in `TestGoDataReferences`.
- [ ] 2.3 Update the Go capability evidence and documented limits. Run the Go reference cases from #238 and make sure that supported facts match exactly.

## 3. TypeScript and JavaScript data declarations

- [ ] 3.1 Extend `internal/indexer/tsextract` for module bindings, identifier destructuring, and static class field names. Cover TypeScript, JavaScript, TSX, and private fields in `TestTypeScriptDataDeclarations`.
- [ ] 3.2 Preserve existing callable-constant identities and add scoped data references. Cover duplicate prevention, computed names, and block shadowing in `TestTypeScriptDataReferences`.
- [ ] 3.3 Update separate TypeScript, JavaScript, and TSX capability evidence. Run the corresponding #238 cases and document remaining dynamic forms.

## 4. Python and Java data declarations

- [ ] 4.1 Extend `internal/indexer/pyextract` for module assignments, annotations, and class attributes. Cover destructuring and repeated assignments in `TestPythonDataDeclarations`.
- [ ] 4.2 Add declared-receiver attributes and scoped Python data references. Cover renamed receivers, class methods, static methods, shadowing, and dynamic names in `TestPythonDataReferences`.
- [ ] 4.3 Extend `internal/indexer/javaextract` for class and interface field lists and their references. Cover owners, static fields, initializer calls, and local shadowing in `TestJavaDataDeclarations` and `TestJavaDataReferences`.
- [ ] 4.4 Update Python and Java capability evidence. Run their #238 acceptance cases and retain record components, enum-specific forms, and dynamic ownership as visible gaps.

## 5. Callable bindings and public queries

- [ ] 5.1 Add conservative callable-binding resolution with bounded chains and cycle detection. Cover imports, competing writes, unknown assignments, and parse failures in `TestCallableBindingResolution`.
- [ ] 5.2 Add the #175 declaration-to-call regression and its reassignment variant. Cover binding visibility and correct callers, callees, and impact in `TestFunctionVariableQueries`.
- [ ] 5.3 Expose fields through existing kind filters and graph presentation. Cover CLI and MCP definitions and explore in `TestDataSymbolPublicQueries`. Add a field-rendering case to the existing web test suite.
- [ ] 5.4 Document binding evidence and query limits beside the capability matrix. Make sure that an uncalled function initializer never appears as a caller in `TestFunctionVariableQueries`.

## 6. Migration and integration acceptance

- [ ] 6.1 Add an additive rebuild marker through the existing metadata source and generator. Cover old indexes with unchanged sources in `TestDataSymbolRebuild`. Document the one-time rebuild behavior.
- [ ] 6.2 Compare fresh indexes with edits, renames, deletion, restoration, and moved fields. Cover stale nodes and alias invalidation in `TestDataSymbolSyncEquivalence`.
- [ ] 6.3 Run the implemented `index-invariants` tests and #238 semantic acceptance for all affected languages. Remove fixed known-failure entries and record remaining gaps without weakening supported cases.
- [ ] 6.4 Add the release note with `task changie -- new --interactive=false -k Features -b "Index core-language variables and fields with scoped references and conservative callable bindings."`. Make sure that the generated fragment describes the shipped contract.
