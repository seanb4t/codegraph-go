# Language Capability Matrix

This is the D-11 language capability matrix ([`.planning/milestones/v0.1-phases/05-language-coverage-resolution-breadth/05-CONTEXT.md`](https://github.com/seanb4t/codegraph-go/blob/planning-archive-v0.15.0/.planning/milestones/v0.1-phases/05-language-coverage-resolution-breadth/05-CONTEXT.md))
— the human-readable half of the coverage contract. The machine-readable
half lives in `internal/indexer/capability/matrix.go`, and
`internal/indexer/capability/matrix_test.go` proves the two stay identical:
every coverage value below matches the Go descriptor exactly, and every gap
bullet below is a byte-for-byte copy of a `Gaps` entry in `matrix.go`. If
this table and the Go descriptor ever drift, `go test ./internal/indexer/capability/...`
fails — this document cannot silently overclaim.

**Coverage legend:** `full` = validated end-to-end (priority-4 via a
corresponding behavioral test in `testdata/golden/`, mainstream-tier via
that axis's own dedicated test coverage); `partial` = works within an
explicitly bounded, named scope (see that language's gaps below); `none` =
does not apply or is not implemented (always paired with a named gap
explaining why).

## Coverage Table

| Language | Extraction | Resolution | Dispatch | Routing |
|---|---|---|---|---|
| `go` | full | full | full | full |
| `java` | full | full | full | full |
| `csharp` | full | full | full | full |
| `python` | full | full | none | full |
| `typescript` | full | full | full | full |
| `tsx` | full | full | full | full |
| `javascript` | full | full | none | full |
| `rust` | full | partial | partial | none |
| `ruby` | full | partial | none | none |
| `php` | full | partial | partial | none |
| `c` | full | partial | none | none |
| `cpp` | full | partial | none | none |
| `swift` | full | partial | partial | none |
| `kotlin` | full | partial | partial | none |

This covers exactly the 14 languages registered in
`internal/indexer/languages.go`'s `LanguageSpec` registry (Go +
priority-4 [Java, C#, Python, TypeScript, TSX, JavaScript] + mainstream-6
[Rust, Ruby, PHP, C, C++, Swift, Kotlin]) — `matrix_test.go` asserts this
set is exact, no missing, no extra.

**Requirement mapping:** priority-4 (`java`/`csharp`/`python`/`typescript`/`tsx`/`javascript`)
= LANG-02..05, validated against Phase-1-locked corpora through hermetic
internal/corpora resolution, failing loudly on absence (02-03, D-10).
Mainstream-6 = LANG-06, documented-partial per
D-04/D-11. Dispatch = RES-02/RES-03 (05-09). Routing = LANG-07 (05-12),
covering the five priority frameworks only: Gin (Go), Spring (Java),
ASP.NET (C#), Django/Flask/FastAPI (Python), Express/NestJS
(TypeScript/TSX/JavaScript).

---

## Repeated names and overloads

Each declaration gets one node. Two methods with one name under one type are overloads. The indexer gives them one `overload_set` node. The type contains the set, and the set contains each overload. A name with one declaration gets no set.

A call resolves among the candidates of its name under its owner. The arity filter runs first. An optional, default or variadic parameter widens the range, and a spread argument sets a lower bound. The argument types that the extractor knows filter second. A type mismatch rejects a candidate in two cases. One of the two types is final, such as `int` or `String`. Or both types are in the index. When one candidate remains, the call gets an edge to it. When one candidate is more specific than the others, it wins. For example, `add(String)` wins over `add(Object)` for a `String` argument. When several candidates remain, the call gets an edge to the set, marked `ambiguous`. When none remains, the call stays unresolved.

A receiver resolves through a type environment, not through its name. The extractor records the declared type of each parameter, local, field and receiver word. A call through a local that holds the result of another call takes that call's return type in a second pass. The extractor never reads the case of a name to decide what the name means.

A same-file reference resolves to the innermost declaration of its name whose scope holds the reference. Only module-level declarations enter the cross-file index. In TypeScript, only exported declarations enter it. A JavaScript file with no ESM export keeps every module-level declaration reachable, because the extractor does not read `module.exports`.

The `node`, `callers`, `callees` and `impact` commands accept `--signature`. The MCP tools with the same names accept the `signature` input. The value is the parameter types of one overload, such as `"String, Throwable"`. Without it, a name that has a set reports across every overload, and each row carries the overload it belongs to. An ambiguous caller is marked `ambiguous`. The `explore` command folds a set into one row that names the overloads.

The rules for two declarations with one name under one owner differ by language.

| Language | Rule |
|---|---|
| Java, C#, Kotlin, C++, Swift | The two are overloads. |
| Python, Ruby, JavaScript | At module level the last definition wins, as at run time. |
| C, C++ and C# under `#if`, PHP under a `function_exists` guard | The two are alternatives. A call to the name reaches their set as ambiguous. |
| C, C++ | A prototype whose definition is in the same file gets no node. |
| Ruby | A class or module that a file reopens is one node in that file. |
| Swift | The qualified name carries the argument labels, so `move(to:)` and `move(by:)` are two functions. |
| Go | A repeated `init` and any other repeat get ordinal ids. |

Two limits remain. In Go, TypeScript, Java and C#, a local is visible from the end of its declaration to the end of its block. In the other languages a local is visible in the whole body. There, a local that shadows an import alias hides the alias in the whole body. A reference by path across languages, such as a route handler named in a string, is not resolved. The follow-up is [issue #249](https://github.com/seanb4t/codegraph-go/issues/249).

## Per-Language Gaps

### `go`

No named gaps — Go is fully covered, validated end-to-end against the
pinned golden corpora
(`testdata/golden/behavioral_test.go#TestCorpusBehavior_Go`).

### `java`

- A call through a parameter, a local or a field resolves through the declared type of that name. A static import of a method from another package does not resolve.
- TestCorpusBehavior_Java now resolves the Phase-1-locked guava corpus through hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old source-as-specification self-consistency fallback is retired.

### `csharp`

- A bare using-imported cross-namespace call or inheritance reference — C#'s dominant idiom — is an accepted, documented gap: only fully-qualified cross-namespace references and same-namespace references resolve, since no global symbol table is built at parse time.
- The extractor reads the members under each branch of an `#if`. Two branches that declare one member are alternatives, and a call to that member is ambiguous.
- partial class/struct/record/interface fragments share one node keyed by (qualifiedName, namespace) with a deterministic sentinel FilePath/StartLine, rather than a resolve.go-coordinated first-fragment-by-path tie-break (Pitfall 5, scheme-b variant).
- TestCorpusBehavior_CSharp now resolves the Phase-1-locked serilog corpus through hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old source-as-specification self-consistency fallback is retired.

### `python`

- pyextract emits no KindInterface nodes — Python has no declared-interface construct, so base-class RefKindEmbeds refs never promote to an implements edge; interface->implementation dispatch traversal does not apply to Python.
- A plain unaliased `import foo.bar` populates no Imports entry (Python's own binding semantics bind only the top-level name); only an aliased plain import or a from-import populates Imports.
- Wildcard from-imports (`from x import *`) are not resolved.
- At module level the last definition of a name wins, as at run time.
- TestCorpusBehavior_Python now resolves the Phase-1-locked requests corpus through hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old self-skip source-as-specification fallback is retired.

### `typescript`

- A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.
- Directory-style imports (`./utils` resolving to `utils/index.ts`) and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.
- TestCorpusBehavior_TSJS now resolves through the language map to the Phase-1-locked hugo corpus (whose 25 JS files supply the tsjs leg despite hugo's manifest language being "go") via hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old self-skip source-as-specification fallback is retired.

### `tsx`

- A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.
- Directory-style imports (`./utils` resolving to `utils/index.ts`) and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.
- TestCorpusBehavior_TSJS now resolves through the language map to the Phase-1-locked hugo corpus (whose 25 JS files supply the tsjs leg despite hugo's manifest language being "go") via hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old self-skip source-as-specification fallback is retired.

### `javascript`

- JavaScript's own grammar has no interface_declaration node and no implements clause — class heritage is extends-only, so RefKindEmbeds refs never resolve to an interface target and never promote to an implements edge for pure-JavaScript files.
- A renamed default import (`import Renamed from './foo'`) is an accepted gap — a default import only resolves when the local binding text coincides with the target's own declared symbol name.
- Directory-style imports and node_modules/package.json main/exports-map resolution are not implemented — only relative-specifier and tsconfig paths/baseUrl-aliased specifiers resolve.
- TestCorpusBehavior_TSJS now resolves through the language map to the Phase-1-locked hugo corpus (whose 25 JS files supply the tsjs leg despite hugo's manifest language being "go") via hermetic internal/corpora resolution and FAILS LOUDLY on absence (02-03, D-10) — the old self-skip source-as-specification fallback is retired.

### `rust`

- `use` declarations are extracted as dependency refs but deliberately never populate Imports — the enclosing crate name needed to compute a matching moduleKey is not available at Extract time, so cross-crate/cross-module use-qualified call and trait resolution does not work; only same-file and same-computed-module-path resolution succeeds.
- Nested `mod { }` blocks are not descended into; macro-generated items are invisible; generic_function/macro_invocation callees are unresolved; trait default-method inheritance is not resolved (no conformance-retry pass for Rust).
- No framework route detector exists for Rust — LANG-07 covers only the five priority frameworks (Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS).

### `ruby`

- Ruby's moduleKey is per-file (like Python's), so there is no same-module (directory-shared) resolution tier; require binds no local alias, and require_relative populates Imports but only helps when the required file's basename happens to textually match the referenced constant.
- A bare, no-parens, no-argument method call (`helper`) is grammatically ambiguous with a local-variable reference in tree-sitter-ruby's own grammar — a verified real grammar limitation, not an oversight (`helper()`, with parens, extracts correctly).
- `include`/`extend`/`prepend` mixins are not extracted as edges; dynamic method definition (`define_method`, `method_missing`) is invisible; Ruby has no declared-interface construct in this extractor, so dispatch synthesis never fires.
- No framework route detector exists for Ruby (e.g. Rails) — LANG-07 covers only the five priority frameworks.

### `php`

- Cross-module resolution works only when a use-imported simple name's target moduleKey matches exactly; a fully-qualified inline reference (`extends \App\Base`) is recorded by simple name only, not as an alternate lookup key.
- Trait inclusion (`use TraitName;` inside a class body) is not extracted as an edge; `object_creation_expression` (`new Widget()`) is not tracked as a calls edge; magic methods (`__call`, `__get`) are invisible.
- A namespace-less file falls back to a less-precise composer.json PSR-4/path-derived moduleKey.
- No framework route detector exists for PHP (e.g. Laravel/Symfony) — LANG-07 covers only the five priority frameworks.

### `c`

- #include never populates Imports — there is no cross-file call resolution for C; only same-file (unqualified) calls resolve.
- A prototype whose definition is in the same file gets no node. Two definitions under the branches of one `#if` are alternatives, and a call to them is ambiguous.
- Anonymous structs, preprocessor-macro-generated symbols, and multi-declarator declaration lines (only the first declarator is considered) are not extracted.
- C has no interface/dispatch construct — not applicable to a procedural language; no implements synthesis is attempted.
- No framework route detector exists for C — LANG-07 covers only the five priority frameworks.
- `.h` files default to the C grammar (a documented default disposition for the C/C++ header-extension ambiguity) — a C++-only header using C++-specific syntax will misparse under the C grammar unless renamed .hpp/.hh.

### `cpp`

- #include never populates Imports; an out-of-line method whose qualifying type lives in a different file resolves via the same cross-file RefKindContains fallback rustextract's impl blocks use, but namespace-qualified name disambiguation does not work since namespace bodies are transparently flattened.
- Template declarations/instantiations, pure-virtual/bodyless method prototypes, operator overloads, and qualified-identifier base classes (`class A : ns::Base`) are not extracted/resolved.
- cextract never emits KindInterface for C++ (no interface/pure-virtual distinction) — base_class_clause embeds edges never promote to an implements edge.
- No framework route detector exists for C++ — LANG-07 covers only the five priority frameworks.
- `.h` files default to the C grammar (a documented default disposition for the C/C++ header-extension ambiguity); C++ claims only unambiguous extensions (.cpp/.cc/.cxx/.hpp/.hh).

### `swift`

- No module-scoped symbol index exists at this tier — the SPM Sources/<Target>/... ModuleKey is a discovery-time placeholder only; import never populates Imports, so only same-file resolution succeeds.
- Extension member extraction/merge is not implemented (an extension's own member declarations are recognized but never walked); protocol-witness resolution, enum cases/associated values, property declarations, and generic constraints are not extracted.
- self-qualified calls are not specially recognized — routed through the generic local-alias path like any other lowercase receiver.
- swift is a `[SUS]`-tier community grammar (alex-pinkus/tree-sitter-swift), pinned by exact commit only after a blocking human-verify supply-chain checkpoint (05-08); its function_declaration binds its own "name" field twice (once to the real identifier, once to the return-type node) — a verified grammar rough edge this extractor relies on tree-sitter's first-bound-value behavior to work around.
- No framework route detector exists for Swift (e.g. Vapor) — LANG-07 covers only the five priority frameworks.

### `kotlin`

- Cross-module resolution works only when an import-qualified simple name's target package computes a matching key; there is no fully-qualified-inline-reference alternate lookup.
- Extension functions, companion objects, and `object` singleton declarations are not extracted; data-class auto-generated members and property declarations are not extracted.
- this-qualified calls are not specially recognized — routed through the generic local-alias path like any other lowercase receiver.
- kotlin is a `[SUS]`-tier community grammar (tree-sitter-grammars/tree-sitter-kotlin@v1.1.0), pinned by exact semver only after a blocking human-verify supply-chain checkpoint (05-08), replacing an originally-approved fwcd fork that failed to build.
- No framework route detector exists for Kotlin (e.g. Ktor, or Spring Boot written in Kotlin — the Spring detector's opt-in Signature is gated on the "java" LanguageSpec only) — LANG-07 covers only the five priority frameworks.

---

## Shared Caveats

These apply across multiple languages and are not repeated per-row above:

- **External-scanner in-process crash risk (Pitfall 4, T-05-DoS).** 9 of the
  10 languages added in Phase 5 carry a tree-sitter grammar with an
  external C scanner (Java, C#, Python's INDENT/DEDENT, TypeScript/TSX/
  JavaScript, Rust, Ruby, PHP, C++, Swift, Kotlin — only C's grammar has
  none). A pathological or adversarial input crashing one of these
  scanners in-process is NOT `recover()`-able — `parser.MaxSourceBytes`
  (4 MiB, Phase 1) is the front-line, accepted-risk-with-mitigation bound,
  not an elimination of the risk.
- **C# partial-class node-identity decision (Pitfall 5).** See the
  `csharp` row above — a `partial class`/`struct`/`record`/`interface`
  fragment's node id is keyed by `(qualifiedName, namespace)` with a
  deterministic sentinel location, not a cross-file "first fragment by
  path" tie-break.
- **`.h` C/C++ extension disposition.** `.h` is claimed by the C
  `LanguageSpec` only; C++ claims only unambiguous extensions
  (`.cpp`/`.cc`/`.cxx`/`.hpp`/`.hh`). A C++-only header using C++-specific
  syntax (templates, classes) will misparse under the C grammar unless
  renamed to one of C++'s claimed extensions.
- **Swift/Kotlin `[SUS]` grammar provenance (T-05-SC, 05-08).** Both
  grammars are community-maintained (not `tree-sitter`-org), pinned by
  exact commit/semver only after a blocking human-verify supply-chain
  checkpoint — never `@latest`, never silently substituted. Kotlin's final
  pin (`tree-sitter-grammars/tree-sitter-kotlin@v1.1.0`) differs from
  05-RESEARCH.md's original `fwcd` recommendation because the originally-
  approved source failed to build; the revised source was re-submitted for
  and received explicit human re-approval before being pinned (05-08).
- **The cross-language `resolve.go` `modulePath` imports-edge limitation
  for non-Go repos.** `isIntraModule`'s `modulePath` parameter is
  populated ONLY from the `"go"` `LanguageSpec`'s own descriptor
  (`discover.go`). In a repo with no `go.mod` (or where Go's own
  descriptor did not resolve), a `RefKindImports` unresolved ref never
  becomes a committed `"imports"` edge — regardless of language. This is
  orthogonal to `calls`/`embeds` resolution, which every extractor's own
  `Imports` map handles independently (and which the corresponding
  behavioral/self-consistency tests do prove resolves). First
  documented in 05-05-SUMMARY.md (C#) and re-confirmed identically by
  05-07-SUMMARY.md (TS/JS); not fixed in Phase 5 — pre-existing, outside
  every individual language plan's own file scope (`resolve.go` is a
  shared file no single language plan owns).
- **Route detection runs on full (`codegraph index`) indexing only.**
  Framework routing (LANG-07, 05-12) is wired into `pipeline.go`'s `Run`,
  not `sync.go`'s incremental `Sync` path. A repo whose routes changed
  will not have them re-detected until the next full re-index — an
  accepted, documented v1 scope boundary (05-12-SUMMARY.md).
