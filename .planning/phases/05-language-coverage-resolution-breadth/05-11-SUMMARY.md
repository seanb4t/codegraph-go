---
phase: 05-language-coverage-resolution-breadth
plan: 11
subsystem: indexing
tags: [tree-sitter, c, cpp, swift, kotlin, mainstream-tier, extraction, resolution, documented-partial, sus-grammar]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 08
    provides: cgo.NewCParser, NewCppParser, NewSwiftParser, NewKotlinParser — the four mainstream-tier grammar constructors this plan consumes; NewSwiftParser/NewKotlinParser exist only because 05-08's blocking human-verify checkpoint approved both [SUS] community grammar pins
  - phase: 05-language-coverage-resolution-breadth
    plan: 10
    provides: the sibling mainstream/{rust,ruby,php}extract package layout and Extract(p, moduleKey, relPath, src) contract this plan's four extractors reproduce verbatim
provides:
  - "internal/indexer/mainstream/cextract — ONE Extract shared by two grammars (C and C++), determining language from relPath's own extension"
  - "internal/indexer/mainstream/swiftextract, kotlinextract — Extract(p, moduleKey, relPath, src) for each, reproducing goextract's exact skip/error contract"
  - "Four LanguageSpec registrations: languages_c.go (CMakeLists.txt-presence descriptor, path-identity ModuleKey), languages_cpp.go (shares cextract.Extract + descriptor with C), languages_swift.go (Package.swift-presence descriptor, SPM Sources/<Target>/... ModuleKey convention), languages_kotlin.go (build.gradle(.kts)-presence descriptor, path-identity fallback + parse-time package override)"
  - "Live-parse-tree-verified node-kind mappings for all four grammars (not assumed from node-types.json alone) — the C/C++/Swift/Kotlin per-language coverage assessment below, ready for 05-13's D-11 capability matrix"
affects: [05-13 (D-11 capability matrix consumes this plan's per-language coverage assessment below), any future mainstream-tier language following this same "extraction + best-effort same-file resolution" shape]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A SHARED extractor package across two grammars (cextract for C+C++) determines its emitted Language field from relPath's own file extension, since LanguageSpec.Extract's cross-language signature carries no explicit language parameter — the parser.Parser handed in is already the correct language-specific one, so a C-grammar tree simply never contains a C++-only node kind and vice versa, making one shared switch-based walker safe."
    - "When a [SUS] community grammar's node-kind names differ from the plan's assumption (Swift's unified class_declaration covering class/struct/enum/actor/extension, distinguished only by an anonymous keyword token; a function_declaration field bound TWICE to the same field name — a verified grammar rough edge), the extractor is adapted and the adaptation documented in the package's own types.go, rather than filed as a blocker — the plan's own explicit guidance for this exact scenario."
    - "Every node-kind/field-name claim in this plan's four extractors was verified against a LIVE parse-tree dump (a throwaway cmd/ program using the pinned cgo.New*Parser constructors + Node.ToSexp()/ChildByFieldName probes) before being encoded into the extractor, not assumed from node-types.json's field declarations alone — node-types.json documents ALLOWED shapes, not which one a real construct actually takes (e.g. Kotlin's class_declaration name field vs. Swift's TWO node-types.json field entries sharing the literal string \"name\")."
    - "A C++ out-of-line method definition (`RetType Type::method() {}`, the dominant idiom for a class's own method BODIES) is extracted as a KindMethod with the SAME cross-file RefKindContains fallback rustextract's impl-block pattern already established — this tier does not require the qualifying type to be declared in the same file."

key-files:
  created:
    - internal/indexer/mainstream/cextract/cextract.go
    - internal/indexer/mainstream/cextract/types.go
    - internal/indexer/mainstream/cextract/cextract_test.go
    - internal/indexer/mainstream/swiftextract/swiftextract.go
    - internal/indexer/mainstream/swiftextract/types.go
    - internal/indexer/mainstream/swiftextract/swiftextract_test.go
    - internal/indexer/mainstream/kotlinextract/kotlinextract.go
    - internal/indexer/mainstream/kotlinextract/types.go
    - internal/indexer/mainstream/kotlinextract/kotlinextract_test.go
    - internal/indexer/languages_c.go
    - internal/indexer/languages_cpp.go
    - internal/indexer/languages_swift.go
    - internal/indexer/languages_kotlin.go
  modified:
    - internal/indexer/languages_test.go

key-decisions:
  - "C and C++ share ONE cextract package/Extract function across TWO LanguageSpec registrations (languages_c.go, languages_cpp.go) — Extract determines 'c' vs 'cpp' purely from relPath's own extension (languageForExt), since its shared cross-language signature carries no language field. '.h' is claimed by C only (the documented default header-ambiguity disposition); C++ claims only unambiguous extensions (.cpp/.cc/.cxx/.hpp/.hh)."
  - "C++ namespace bodies are transparently flattened (recursively) into the enclosing file's own top-level scope — no namespace node is ever emitted (this vocabulary has no Kind that fits one), so namespace-qualified names are never disambiguated by namespace at this tier."
  - "A C++ out-of-line method definition (`Circle::area() {}`) is extracted as KindMethod via the SAME cross-file RefKindContains fallback pattern rustextract's impl-block handling already established, rather than being silently dropped — a deliberate scope EXTENSION over the plan's minimum bar, since this is the dominant real-world C++ idiom (header declares, .cpp defines)."
  - "Swift's function_declaration binds its own 'name' field TWICE in this grammar (once to the function's simple_identifier name, once to its return-type node) — a verified real grammar rough edge, not an assumption. ChildByFieldName('name') reliably returns the FIRST bound value, which this extractor relies on directly."
  - "Swift extension declarations are recognized (never misextracted as a fresh empty type) but their own member declarations are never walked — the plan's own explicitly-named gap, since merging an extension's members into its (possibly cross-file) extended type's method set requires a cross-declaration merge this tier does not implement."
  - "Kotlin's package declaration parse-time-OVERRIDES the discovery-time ModuleKey placeholder (mirrors csharpextract/phpextract's identical pattern); Kotlin's import DOES populate FileResult.Imports (mirrors PHP's decision, unlike Rust's — a Kotlin import target is unambiguously a simple class/function name, not an ambiguous crate-relative path)."
  - "Kotlin 'object' singleton declarations, extension functions, and companion objects are all explicitly out of scope for this tier — named gaps, not silent omissions."
  - "self/this-qualified calls (Swift's `self.method()`, Kotlin's `this.method()`) are deliberately NOT special-cased as an implicit same-instance call in either extractor — both are lowercase identifiers routed through the same WR-02 synthetic-non-matching-alias pattern as any other local binding, a conservative choice that never risks a false same-module match."

patterns-established:
  - "Live-parse-tree verification BEFORE encoding a node-kind mapping into an extractor, for any grammar whose exact field/node shapes are not already proven elsewhere in this codebase — especially load-bearing for [SUS]-tier community grammars (Swift, Kotlin) getting their FIRST extraction exercise in this project."

requirements-completed: [LANG-06]

coverage:
  - id: D1
    description: "C and C++ extract struct/class (documented -> KindStruct)/typedef (-> KindTypeAlias)/top-level functions and prototypes/inline AND out-of-line methods (-> KindMethod, cross-file contains fallback) into the shared vocabulary through ONE shared cextract package; #include -> RefKindImports (best-effort, no Imports population); base_class_clause -> RefKindEmbeds; calls distinguish bare/this->/qualified shapes; registered via a CMakeLists.txt-presence descriptor + path-identity ModuleKey, with '.h' claimed by C only (documented ambiguity disposition)"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_OutOfLineMethodContainsEdge"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_OutOfLineMethodCrossFileContains"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_Includes"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_Supertypes"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_LanguageDeterminedByExtension"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_C"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Cpp"
        status: pass
    human_judgment: false
  - id: D2
    description: "Swift extracts class/struct/actor/enum (documented, distinguished by an anonymous keyword token -> KindStruct)/protocol (-> KindInterface)/methods/top-level functions into the shared vocabulary using the [SUS] alex-pinkus/tree-sitter-swift grammar (05-08-approved pin); import -> RefKindImports (no Imports population); inheritance_specifier -> RefKindEmbeds; calls distinguish bare/navigation-expression shapes; extension bodies recognized but never walked (named gap); registered via a Package.swift-presence descriptor + SPM Sources/<Target>/... ModuleKey convention"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_MethodContainsEdge"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_ExtensionNotExtracted"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_Import"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_Inheritance"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Swift"
        status: pass
    human_judgment: false
  - id: D3
    description: "Kotlin extracts class/enum (documented, distinguished by an anonymous keyword token -> KindStruct)/interface (-> KindInterface)/methods (including bodyless interface requirements)/top-level functions into the shared vocabulary using the [SUS] tree-sitter-grammars/tree-sitter-kotlin@v1.1.0 grammar (05-08-approved, replacing an unbuildable original pin); package parse-time-overrides ModuleKey; import populates Imports (unlike Rust); delegation_specifier -> RefKindEmbeds; calls distinguish bare/navigation-expression shapes; object declarations never extracted (named gap); registered via a build.gradle(.kts)-presence descriptor + path-identity ModuleKey fallback"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_MethodContainsEdge"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_PackageOverridesModuleKey"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_Import"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_Delegation"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_ObjectNotExtracted"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Kotlin"
        status: pass
    human_judgment: false
  - id: D4
    description: "parser.MaxSourceBytes is enforced before any backend-specific parsing runs for all four grammars (C++/Swift/Kotlin carry external C scanners; C does not); a parse failure is a per-file skip (FileResult.Err), never a fatal batch error; Go/Priority-4/Rust/Ruby/PHP fixtures unaffected by the four new registrations"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/cextract/cextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/swiftextract/swiftextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/kotlinextract/kotlinextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./... && go test ./internal/indexer/... ./internal/parser/... -count=1 (all 15 packages pass, including the pre-existing Go/Java/C#/Python/TS-JS/Rust/Ruby/PHP fixtures)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 11: Mainstream Grammar Layer — C/C++, Swift, Kotlin (LANG-06) Summary

**Second half of the mainstream-6 tier: ONE shared cextract package driving both C and C++ grammars (out-of-line method definitions resolved via rustextract's own cross-file contains pattern), plus swiftextract and kotlinextract — the FIRST extraction exercise for both 05-08-approved `[SUS]` community grammars, with every node-kind/field-name mapping verified against a live parse-tree dump rather than assumed, including a genuine grammar rough edge discovered and documented in Swift's own `function_declaration`.**

## Performance

- **Duration:** ~50 min
- **Completed:** 2026-07-12
- **Tasks:** 2 (Task 1: shared C/C++ cextract + registrations; Task 2: Swift + Kotlin extractors + registrations)
- **Files modified:** 14 (13 created, 1 modified)

## Accomplishments

- **C + C++** (`cextract`, ONE package shared by two grammars): `struct_specifier`/`class_specifier` (documented) -> `KindStruct`, `type_definition` -> `KindTypeAlias`, a root-level `function_definition` -> `KindFunction`, a root-level bodyless `declaration` resolving to a function prototype -> `KindFunction` too (useful header-declared symbols, no calls collected), an inline class-body `function_definition` -> `KindMethod`, and — a deliberate scope extension over the plan's minimum bar — a C++ out-of-line `Type::method() {}` definition -> `KindMethod` via the SAME cross-file `RefKindContains` fallback pattern rustextract's `impl` block handling already established. `preproc_include` -> `RefKindImports` (best-effort, never populates `Imports`). `base_class_clause` -> `RefKindEmbeds`. Calls distinguish a bare identifier, `this->method()`, and `Type::staticMethod()` shapes. Namespace bodies are transparently (recursively) flattened — no namespace node exists in this vocabulary. `Extract` determines "c" vs "cpp" purely from `relPath`'s own extension since its shared cross-language signature carries no language field. Registered via `languages_c.go` (`.c`/`.h`, CMakeLists.txt-presence descriptor) and `languages_cpp.go` (`.cpp`/`.cc`/`.cxx`/`.hpp`/`.hh`) — `.h` is claimed by C only, the documented default disposition for the header ambiguity.
- **Swift** (`swiftextract`, `alex-pinkus/tree-sitter-swift`, `[SUS]`): a LIVE parse-tree dump (a throwaway `cmd/` probe using the pinned `cgo.NewSwiftParser`) revealed this grammar has NO dedicated `class_declaration`/`struct_declaration` node pair — class/struct/enum/actor/extension all share ONE `class_declaration` node kind distinguished only by an anonymous keyword token child, adapted into `swiftDeclKind`. It also revealed a genuine grammar rough edge: `function_declaration`'s own "name" field is bound TWICE (once to the function's real `simple_identifier` name, once to its return-type node) — `ChildByFieldName("name")` reliably returns the first-bound value, which this extractor relies on directly rather than working around. class/struct/actor/enum -> `KindStruct`, `protocol_declaration` -> `KindInterface`, `function_declaration` -> `KindMethod`/`KindFunction`, `import_declaration` -> `RefKindImports` (no `Imports` population), `inheritance_specifier` -> `RefKindEmbeds` (one per conformance, verified against a live multi-conformance `class Foo: A, B` dump). Extension declarations are recognized but their own members are never walked — the plan's own explicitly-named gap. Registered via `languages_swift.go` (`.swift`, Package.swift-presence descriptor, an SPM `Sources/<Target>/...`-convention `ModuleKey`).
- **Kotlin** (`kotlinextract`, `tree-sitter-grammars/tree-sitter-kotlin@v1.1.0`, `[SUS]`): a live parse-tree dump found this grammar CLEANER than Swift's — no verified rough edge. `class_declaration` similarly covers class/interface/enum, distinguished by an anonymous keyword token (`kotlinDeclKind`). class/enum -> `KindStruct`, interface -> `KindInterface`, `function_declaration` -> `KindMethod`/`KindFunction` (including an interface's own bodyless method requirement, since Kotlin uses no separate node kind for one the way Swift's `protocol_function_declaration` does). `package` parse-time-OVERRIDES the discovery-time `ModuleKey` placeholder (mirrors PHP/C#'s identical pattern); `import` DOES populate `Imports` (mirrors PHP, unlike Rust — a Kotlin import target is unambiguously a simple name). `delegation_specifier` -> `RefKindEmbeds` (both plain interface conformance and constructor-invoked superclass shapes, both verified via live dumps). `object` singleton declarations are never extracted — a named gap. Registered via `languages_kotlin.go` (`.kt`/`.kts`, build.gradle(.kts)-presence descriptor, path-identity `ModuleKey` fallback).
- `go build ./...`, `go vet ./...`, and `go test ./internal/indexer/... ./internal/parser/... -count=1` all pass — 15 packages green including the pre-existing Go/Java/C#/Python/TS-JS/Rust/Ruby/PHP fixtures and registry tests, unaffected by the four new registrations.

## Per-Language Coverage Assessment (for the D-11 capability matrix, Wave F / plan 05-13)

| Language | Extraction | Same-file resolution | Same-module resolution | Cross-module resolution | Named gaps |
|---|---|---|---|---|---|
| **C** | Full (struct/typedef/top-level fns and prototypes) | Yes (empty-PkgAlias unqualified calls) | N/A (path-identity ModuleKey, no module concept beyond the file itself) | **No** — `#include` never populates `Imports` | Anonymous structs; preprocessor-macro-generated symbols; multi-declarator `declaration` lines only consider the first declarator |
| **C++** | Full (class/struct/typedef/inline AND out-of-line methods/top-level fns and prototypes) | Yes (empty-PkgAlias `this->`/`Type::`/same-module calls) | Yes (an out-of-line `Type::method()` resolves against a same-file type via `typeNodesByName`) | **Partial** — an out-of-line method whose qualifying type lives in a DIFFERENT file (the common header/.cpp split) is recorded via `RefKindContains` for Pass 2 to resolve, same mechanism as rustextract's `impl` blocks; `#include` never populates `Imports` | Template declarations/instantiations; pure-virtual/bodyless method prototypes; operator overloads; namespace-qualified name disambiguation (namespaces are fully flattened); qualified-identifier base classes (`class A : ns::Base`); `.h` files default to the C grammar (documented ambiguity disposition) |
| **Swift** | Full (class/struct/actor/enum/protocol/methods/top-level fns) | Yes (empty-PkgAlias PascalCase-receiver calls) | No (no module-scoped symbol index exists at this tier; SPM target-dir ModuleKey is a discovery-time placeholder only) | **No** — `import` never populates `Imports` | Extension member extraction/merge (the plan's own named gap); protocol-witness resolution; `self`-qualified calls not specially recognized (routed through the generic local-alias path); enum cases/associated values; property declarations; generic constraints; the grammar's own verified `function_declaration` double-"name"-field rough edge (worked around, not a functional gap) |
| **Kotlin** | Full (class/interface/enum/methods, including bodyless interface requirements/top-level fns) | Yes (empty-PkgAlias PascalCase-receiver calls) | Yes (a `package`-declared file's own computed key matches another file sharing the identical dotted package path) | **Partial** — an `import`-qualified simple name resolves only when the imported target's own `package` computes a matching key; no fully-qualified-inline-reference alternate lookup | Extension functions (the plan's own named gap); companion objects (the plan's own named gap); `object` singleton declarations; `this`-qualified calls not specially recognized; data-class auto-generated members; property declarations |

All four share the project-wide `T-05-DoS` caveat: `parser.MaxSourceBytes` bounds every parse before any backend-specific parsing runs, but a crash inside C++/Swift/Kotlin's external C scanners is NOT `recover()`-able (C carries no external scanner) — the accepted Phase-1 mitigation contract, unchanged by this plan. Swift and Kotlin additionally carry `T-05-SC` (05-08's already-discharged supply-chain gate): both grammars are `[SUS]`-tier, pinned by exact commit/semver only after a blocking human-verify checkpoint — this plan only builds extractors on top of those already-approved pins.

## Task Commits

Each task was committed with a RED (test) then GREEN (feat) pair:

1. **Task 1: shared C/C++ cextract + registrations**
   - `0f3cfc3` (test) — `cextract_test.go` alone (no implementation) confirmed RED via `go vet` failure (`undefined: Extract`)
   - `06cdb65` (feat) — `cextract.go`/`types.go`, `languages_c.go`, `languages_cpp.go` + registry tests, confirmed GREEN
2. **Task 2: Swift + Kotlin extractors + registrations**
   - `7385069` (test) — `swiftextract_test.go`/`kotlinextract_test.go` alone confirmed RED via `go vet` failure
   - `6d4320d` (feat) — `swiftextract.go`/`types.go`, `kotlinextract.go`/`types.go`, `languages_swift.go`, `languages_kotlin.go` + registry tests, confirmed GREEN

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified

- `internal/indexer/mainstream/cextract/cextract.go` — shared Extract, extractor tree-walk, struct/class/typedef/function/prototype/out-of-line-method node-kind mapping, transparent namespace flattening, includes, supertypes, calls
- `internal/indexer/mainstream/cextract/types.go` — node-kind mapping decisions + full documented rationale for C/C++'s resolution boundaries and the `.h` ambiguity disposition (package doc comment)
- `internal/indexer/mainstream/cextract/cextract_test.go` — table-driven node-kind mapping across BOTH a C and a C++ source, out-of-line method contains-edge (same-file and cross-file), no-field-nodes, includes, supertypes, calls, language-by-extension, moduleKey pass-through, oversized-file skip
- `internal/indexer/mainstream/swiftextract/swiftextract.go` — Extract, extractor tree-walk, `swiftDeclKind` keyword-token scan, class-like/protocol node-kind mapping, imports, inheritance, methods/top-level functions, calls
- `internal/indexer/mainstream/swiftextract/types.go` — node-kind mapping decisions + full documented rationale, including the verified `function_declaration` double-"name"-field grammar rough edge
- `internal/indexer/mainstream/swiftextract/swiftextract_test.go` — table-driven node-kind mapping, contains edge, extension-not-extracted, import, multi-conformance inheritance, calls, moduleKey pass-through, oversized-file skip
- `internal/indexer/mainstream/kotlinextract/kotlinextract.go` — Extract, extractor tree-walk, `kotlinDeclKind` keyword-token scan, class-like node-kind mapping, package override, imports, delegation, methods/top-level functions (including bodyless interface requirements), calls
- `internal/indexer/mainstream/kotlinextract/types.go` — node-kind mapping decisions + full documented rationale for Kotlin's resolution boundaries
- `internal/indexer/mainstream/kotlinextract/kotlinextract_test.go` — table-driven node-kind mapping, contains edge, package override, import, delegation (both shapes), calls, object-not-extracted, oversized-file skip
- `internal/indexer/languages_c.go` — C `LanguageSpec` registration, `readCProjectDescriptor` (CMakeLists.txt presence), shared with C++
- `internal/indexer/languages_cpp.go` — C++ `LanguageSpec` registration, reuses `languages_c.go`'s descriptor
- `internal/indexer/languages_swift.go` — Swift `LanguageSpec` registration, `readSwiftDescriptor` (Package.swift presence), `swiftModuleKey` (SPM `Sources/<Target>/...` convention)
- `internal/indexer/languages_kotlin.go` — Kotlin `LanguageSpec` registration, `readKotlinDescriptor` (build.gradle(.kts) presence)
- `internal/indexer/languages_test.go` — `TestLanguageRegistry_C`/`_Cpp`/`_Swift`/`_Kotlin`

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:

- C and C++ deliberately share ONE extractor package across two `LanguageSpec` registrations, with `Extract` inferring the language from `relPath`'s own extension — the cleanest way to satisfy "two grammars, one package" (05-RESEARCH.md's own recommended structure) without threading an extra parameter through the shared cross-language `Extract` signature.
- A C++ out-of-line method definition is resolved through the SAME cross-file `RefKindContains` mechanism rustextract's `impl` blocks already established, rather than inventing a new pattern or dropping the (extremely common) header/.cpp-split idiom entirely.
- Both `[SUS]` grammars' exact node-kind/field shapes were verified via live parse-tree dumps BEFORE any extractor code was written — this caught Swift's real double-"name"-field rough edge and both grammars' unified class-declaration-with-anonymous-keyword shape, neither of which matched the plan's original node-kind naming assumption. Per the plan's own guidance, both were adapted and documented rather than treated as blockers.
- `self`/`this`-qualified calls are deliberately NOT special-cased in either Swift or Kotlin — both route through the same generic local-alias heuristic as any other lowercase receiver, a conservative choice consistent with this tier's "never risk a false same-module match" discipline.

## Deviations from Plan

**None — plan executed exactly as written**, with one deliberate, documented SCOPE EXTENSION beyond the plan's stated minimum:

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical, scope extension] C++ out-of-line method definitions extracted as KindMethod**
- **Found during:** Task 1 (cextract design, before any test was written — surfaced by grammar exploration, not a bug fix)
- **Issue:** The plan's own task description only explicitly calls out "C++ methods -> KindMethod" without distinguishing inline-vs-out-of-line definitions. Out-of-line method BODIES (`RetType Type::method() { ... }`) are the dominant real-world C++ idiom (declare in header, define in .cpp) — silently dropping them would leave the vast majority of real C++ method bodies unextracted, a correctness gap far more severe than the plan's other named, accepted gaps (templates, pure-virtual, operators).
- **Fix:** Extended `emitFreeFunctionLike` to detect a top-level `function_definition`/`declaration` whose declarator resolves to a `qualified_identifier` (`Type::method`) and route it through the SAME cross-file `RefKindContains` fallback pattern rustextract's `impl` block handling already established (an existing, proven pattern — not a new architecture).
- **Files modified:** internal/indexer/mainstream/cextract/cextract.go, types.go
- **Verification:** `TestExtract_NodeKinds/C++_out-of-line_method`, `TestExtract_OutOfLineMethodContainsEdge`, `TestExtract_OutOfLineMethodCrossFileContains` all pass
- **Committed in:** `06cdb65` (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 missing-critical scope extension, reusing an existing proven pattern). 0 architectural changes, 0 scope creep beyond what LANG-06/D-04's own "full structural extraction" floor already implies for C++'s single most common method-definition idiom.
**Impact on plan:** Materially improves real-world C++ coverage without introducing any new resolution mechanism — reuses rustextract's own cross-file contains pattern verbatim.

## Issues Encountered

- **Swift's `function_declaration` binds its own "name" field to TWO different child nodes** (the function's real `simple_identifier` name, AND separately a return-type-shaped node) — discovered via a live parse-tree probe (`ChildByFieldName("name")` explicitly tested against multiple real function declarations), not assumed from `node-types.json`'s field-type list alone (which only documents the UNION of allowed types, not which one wins when a field is multiply-bound). Verified that `ChildByFieldName` reliably returns the first-bound value in tree-sitter's own Go bindings, so this extractor relies on that behavior directly rather than working around it — documented as a real, verified grammar rough edge in `swiftextract/types.go`, consistent with this `[SUS]` grammar's "may be rougher" floor set by 05-CONTEXT.md/05-RESEARCH.md.
- **Neither the plan's original "class_declaration/struct_declaration" (Swift) nor "class_declaration" (Kotlin, matched, but with an undocumented anonymous-keyword-only distinction) node-kind naming matched the real grammars exactly** — both `alex-pinkus/tree-sitter-swift` and `tree-sitter-grammars/tree-sitter-kotlin` unify class/struct/enum/(actor/extension for Swift; interface/object for Kotlin) declarations under ONE node kind, distinguished only by an anonymous keyword token child with no dedicated field. Both were verified via live parse-tree dumps before any extractor code was written, then adapted (`swiftDeclKind`/`kotlinDeclKind` keyword-token scans) and documented — the plan's own explicit guidance for exactly this scenario ("If a grammar's node-kind names differ from expectation ... adapt the extractor and DOCUMENT it").
- No live TS CodeGraph CLI or curated C/C++/Swift/Kotlin corpus was needed for this plan — D-12's mainstream-tier bar is self-consistency + spot-check (not golden-parity, per 05-RESEARCH.md), fully satisfied by each package's own table-driven extraction tests plus the shared `TestLanguageRegistry_*` registry-level proof already established by every prior mainstream-tier plan.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `internal/indexer/mainstream/{cextract,swiftextract,kotlinextract}` and all four `LanguageSpec` registrations (`c`, `cpp`, `swift`, `kotlin`) are complete, tested, and registered — completing the full mainstream-6 tier (Rust/Ruby/PHP from 05-10, C/C++/Swift/Kotlin from this plan) for LANG-06.
- The per-language coverage assessment table above is ready for direct consumption by 05-13's `docs/LANGUAGE-CAPABILITY-MATRIX.md` (D-11) — every gap named, no silent omissions, and Swift/Kotlin's outcome is explicitly consistent with 05-08's approval decision (both approved, both shipped at documented-partial coverage).
- `go build ./...`, `go vet ./...`, and `go test ./internal/indexer/... ./internal/parser/... -count=1` all pass across 15 packages (the four new mainstream packages plus every pre-existing indexer/parser package), confirming Priority-4/Go/Rust/Ruby/PHP fixtures are unaffected by the new registrations.

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 13 created source files plus this SUMMARY.md confirmed present on disk; all four commits (0f3cfc3, 06cdb65, 7385069, 6d4320d) confirmed in git log.
