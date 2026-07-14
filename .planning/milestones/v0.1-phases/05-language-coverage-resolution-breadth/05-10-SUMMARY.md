---
phase: 05-language-coverage-resolution-breadth
plan: 10
subsystem: indexing
tags: [tree-sitter, rust, ruby, php, mainstream-tier, extraction, resolution, documented-partial]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 08
    provides: cgo.NewRustParser, NewRubyParser, NewPHPParser (php/src grammar accessor) — the three mainstream-tier grammar constructors this plan consumes
  - phase: 05-language-coverage-resolution-breadth
    plan: 06
    provides: pyextract's discovery-time-only ModuleKey pattern (closest structural analog for Rust/Ruby's own path-derived identity) and the shared goextract vocabulary reuse discipline every extractor follows
  - phase: 05-language-coverage-resolution-breadth
    plan: 04
    provides: csharpextract's parse-time-namespace-override pattern (reused verbatim for PHP's own in-source `namespace` declaration)
provides:
  - "internal/indexer/mainstream/rustextract, rubyextract, phpextract packages — Extract(p, moduleKey, relPath, src) for each, reproducing goextract's exact skip/error contract"
  - "Three LanguageSpec registrations (languages_rust.go/languages_ruby.go/languages_php.go): Rust (Cargo.toml crate-name descriptor), Ruby (Gemfile-presence descriptor, unconditional path-based ModuleKey), PHP (composer.json PSR-4 descriptor + parse-time namespace override)"
  - "A phpextract self-consistency test (TestSelfConsistency_DeterministicRebuild) proving two from-scratch indexes of a real fixture produce a byte-identical Export() stream — the mainstream-tier D-12 validation bar"
affects: [05-13 (D-11 capability matrix consumes this plan's per-language coverage assessment below), any future mainstream-tier language following this same "extraction + best-effort same-file/same-module resolution" shape]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mainstream-tier extractors intentionally do LESS ordering/scope work than priority-4: top-level-declaration-only scanning (no descent into Rust `mod` blocks or PHP anonymous classes), and — for Rust specifically — a documented decision to NEVER populate FileResult.Imports from a `use` statement, because Extract's shared cross-language signature has no way to know the enclosing crate name needed to compute a moduleKey-matching target (the same architectural gap tsextract's Config/SetConfig side-channel solved for TS/JS, deliberately not re-solved here — out of scope for a documented-partial tier)."
    - "PHP reuses csharpextract's exact parse-time-ModuleKey-override pattern (a file's own `namespace Foo\\Bar;`/`namespace Foo\\Bar { ... }` declaration overrides the discovery-time placeholder) since PHP's namespace, like C#'s, is declared IN the source and independent of directory layout."
    - "Ruby's ModuleKey is UNCONDITIONALLY the extension-stripped relPath (mirrors tsextract's descriptor-independent pattern), because require_relative's own path arithmetic must match the target file's key by construction regardless of Gemfile presence."
    - "A grammar-level ambiguity is treated as a real, documented gap rather than worked around: tree-sitter-ruby parses a bare, no-parens, no-argument method invocation (`helper`) as a plain `identifier` node, not a `call` node — indistinguishable from a local-variable reference without scope tracking this tier does not implement. `helper()` (with parens) extracts correctly."

key-files:
  created:
    - internal/indexer/mainstream/rustextract/rustextract.go
    - internal/indexer/mainstream/rustextract/types.go
    - internal/indexer/mainstream/rustextract/rustextract_test.go
    - internal/indexer/mainstream/rubyextract/rubyextract.go
    - internal/indexer/mainstream/rubyextract/types.go
    - internal/indexer/mainstream/rubyextract/rubyextract_test.go
    - internal/indexer/mainstream/phpextract/phpextract.go
    - internal/indexer/mainstream/phpextract/types.go
    - internal/indexer/mainstream/phpextract/phpextract_test.go
    - internal/indexer/mainstream/phpextract/resolution_test.go
    - internal/indexer/languages_rust.go
    - internal/indexer/languages_ruby.go
    - internal/indexer/languages_php.go
  modified:
    - internal/indexer/languages_test.go

key-decisions:
  - "Rust's `use` declarations are extracted (RefKindImports refs, every leaf path of plain/aliased/grouped-brace/wildcard forms) but deliberately NEVER populate FileResult.Imports — computing a matching target moduleKey requires the enclosing crate name, which Extract's shared cross-language signature does not carry. Cross-file `use`-qualified call/trait resolution is therefore out of scope for this tier; only same-file (and, incidentally, any other file sharing the identical computed rustModulePath) resolution via an empty PkgAlias works."
  - "Ruby's `require`/`require_relative` are recognized via CALL-SHAPE detection (a receiverless call whose method name is exactly 'require'/'require_relative' with a static string argument), since Ruby's grammar has no dedicated import-statement node kind — every other call node is walked normally, with this shape explicitly excluded from calls-edge emission to avoid double-counting."
  - "PHP's composer.json PSR-4 autoload map is parsed (encoding/json, no new dependency) into languages_php.go's descriptor, with a longest-directory-prefix-match ModuleKey fallback (deterministic tie-break via sorted namespace keys) — but this fallback is only ever load-bearing for a PHP file with NO explicit `namespace` declaration, since phpextract.Extract's parse-time override (mirroring csharpextract) takes priority whenever one exists."
  - "All three extractors map their language's 'lesser' structural concept onto an EXISTING shared-vocabulary kind rather than minting a new one: Rust enum -> KindStruct, Ruby module -> KindStruct, PHP trait -> KindStruct — consistent with every prior extractor's 'reuse, don't fork the vocabulary' discipline (D-01)."

patterns-established:
  - "Documented-partial resolution bar (D-04, this plan's own concrete instances): extraction is always complete and structurally correct; cross-file resolution is explicitly bounded to same-file/same-module, with every specific gap named in each package's types.go doc comment for the D-11 capability matrix to consume verbatim — not discovered ad hoc later."

requirements-completed: [LANG-06]

coverage:
  - id: D1
    description: "Rust extracts struct/enum (documented -> KindStruct)/trait (-> KindInterface)/impl-block methods/top-level functions into the shared vocabulary; `use` dependency refs recorded (no Imports population, documented gap); `impl Trait for Type` emits a RefKindEmbeds ref (auto-promotes to \"implements\" at resolve time); registered via Cargo.toml crate-name descriptor + path-identity fallback"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/rustextract/rustextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rustextract/rustextract_test.go#TestExtract_Uses"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rustextract/rustextract_test.go#TestExtract_ImplTraitEmbeds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rustextract/rustextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Rust"
        status: pass
    human_judgment: false
  - id: D2
    description: "Ruby extracts class/module (documented -> KindStruct)/method/singleton_method into the shared vocabulary; require/require_relative recognized via call-shape detection (require_relative resolves to a directory-relative moduleKey); superclass -> RefKindEmbeds; registered via Gemfile-presence descriptor + unconditional path-based ModuleKey"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/rubyextract/rubyextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rubyextract/rubyextract_test.go#TestExtract_Requires"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rubyextract/rubyextract_test.go#TestExtract_Superclass"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rubyextract/rubyextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Ruby"
        status: pass
    human_judgment: false
  - id: D3
    description: "PHP extracts class/trait (documented -> KindStruct)/interface/methods/functions into the shared vocabulary using the php/src grammar; namespace_use_declaration (plain/grouped-brace/aliased) populates Imports; base_clause/class_interface_clause -> RefKindEmbeds; function/member/scoped call expressions -> RefKindCalls; a declared `namespace` overrides the composer.json-PSR4-or-path moduleKey placeholder (parse-time override, mirrors csharpextract)"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_NamespaceUse"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_SupertypeClauses"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_NamespaceOverridesModuleKey"
        status: pass
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_PHP"
        status: pass
    human_judgment: false
  - id: D4
    description: "PHP self-consistency (mainstream-tier D-12 bar): two from-scratch indexes of a real composer.json + namespaced interface + class + function fixture produce a byte-identical Export() stream after normalizing the one volatile Meta field, and the committed graph contains PHP's full node-kind taxonomy"
    requirement: "LANG-06"
    verification:
      - kind: integration
        ref: "internal/indexer/mainstream/phpextract/resolution_test.go#TestSelfConsistency_DeterministicRebuild"
        status: pass
      - kind: integration
        ref: "internal/indexer/mainstream/phpextract/resolution_test.go#TestSelfConsistency_ExpectedStructure"
        status: pass
    human_judgment: false
  - id: D5
    description: "parser.MaxSourceBytes is enforced before any backend-specific parsing runs for all three grammars (each carries an external C scanner — Rust raw strings, Ruby heredocs, PHP tag-switching); a parse failure is a per-file skip (FileResult.Err), never a fatal batch error; Go/priority-4 fixtures unaffected by the three new registrations"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/mainstream/rustextract/rustextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/rubyextract/rubyextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: unit
        ref: "internal/indexer/mainstream/phpextract/phpextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./... && go test ./internal/indexer/... ./internal/parser/... -count=1 (all packages pass, including the pre-existing Go/Java/C#/Python/TS-JS fixtures)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 10: Mainstream-Tier Extraction — Rust, Ruby, PHP (LANG-06) Summary

**Three mainstream-tier extractor packages (rustextract, rubyextract, phpextract) delivering documented-partial coverage per D-04 — full structural extraction into the shared codegraph vocabulary, best-effort same-file/same-module cross-reference resolution, every gap named for the D-11 capability matrix — registered through the language registry with a Cargo.toml/Gemfile/composer.json project-descriptor hook each, plus a PHP self-consistency (deterministic-rebuild) test as the mainstream-tier D-12 validation bar.**

## Performance

- **Duration:** ~50 min
- **Completed:** 2026-07-12
- **Tasks:** 2 (Task 1: Rust + Ruby; Task 2: PHP)
- **Files modified:** 14 (13 created, 1 modified)

## Accomplishments

- **Rust** (`rustextract`): `struct_item`/`enum_item` (documented) -> `KindStruct`, `trait_item` -> `KindInterface`, `impl_item`'s own `function_item` children -> `KindMethod` (same-file `contains` edge or cross-file `RefKindContains` unresolved ref, mirroring goextract's receiver-type handling), top-level `function_item` -> `KindFunction`, `use_declaration` -> `RefKindImports` refs across every shape (plain/aliased/grouped-brace/wildcard) with a deliberate, documented decision to NEVER populate `Imports` (the crate name needed to compute a matching moduleKey is not available at `Extract` time — the same architectural gap TS/JS's Config side-channel solved, not re-solved here), `impl Trait for Type` -> a `RefKindEmbeds` ref that auto-promotes to `"implements"` at resolve time via the existing generic promotion logic. Calls disambiguate a bare identifier, a PascalCase same-module `Type::method()`/field-receiver attempt, and a synthetic-alias-forced local-variable receiver (WR-02 pattern).
- **Ruby** (`rubyextract`): `class`/`module` (documented) -> `KindStruct`, `method`/`singleton_method` inside a class/module body -> `KindMethod`, top-level -> `KindFunction`. `require`/`require_relative` are ordinary `call` nodes in Ruby's own grammar — recognized via call-shape detection (receiverless, method name exactly `require`/`require_relative`, static string argument) and excluded from calls-edge emission to avoid double-counting; `require_relative` resolves to a directory-relative moduleKey and populates `Imports`. A class's `superclass` -> `RefKindEmbeds`. Discovered and documented a REAL, verified grammar-level limitation: a bare, no-parens, no-argument method call (`helper`) parses as a plain `identifier`, indistinguishable from a local-variable reference without scope tracking — `helper()` (with parens) extracts correctly.
- **PHP** (`phpextract`, `php/src` grammar accessor): `class_declaration` -> `KindStruct`, `interface_declaration` -> `KindInterface`, `trait_declaration` (documented) -> `KindStruct`, `method_declaration`/`function_definition` -> `KindMethod`/`KindFunction`. `namespace_use_declaration` handles plain/grouped-brace (`use Prefix\{A, B}`)/aliased forms and DOES populate `Imports` (unlike Rust, a PHP `use` target is unambiguously a class/interface/trait/function name). `base_clause`/`class_interface_clause` -> `RefKindEmbeds`. `function_call_expression`/`member_call_expression`/`nullsafe_member_call_expression`/`scoped_call_expression` -> `RefKindCalls`, distinguishing `$this->`, `Type::method()`/`self::`/`parent::`, and local-variable receivers (WR-02 pattern). A file's own declared `namespace Foo\Bar;`/`{ ... }` OVERRIDES the discovery-time moduleKey placeholder (parse-time override, reusing csharpextract's exact pattern) — `languages_php.go`'s composer.json PSR-4 longest-prefix-match descriptor is only load-bearing as the fallback for a namespace-less file.
- Registered all three languages (`languages_rust.go`/`languages_ruby.go`/`languages_php.go`): Rust via a bounded line-scan Cargo.toml `[package] name` reader (no TOML dependency, mirroring pyextract's presence-not-parse discipline) + Cargo-convention module-path computation (`src/` stripped, `mod`/`lib`/`main` filenames collapse to their directory, `/` -> `::`); Ruby via a Gemfile-presence-only descriptor (informational, since ModuleKey is unconditionally path-based, mirroring tsextract's descriptor-independent pattern); PHP via a real `encoding/json` composer.json PSR-4 parse with deterministic longest-match tie-breaking (sorted keys, since Go map iteration order is non-deterministic — D-01a).
- `internal/indexer/mainstream/phpextract/resolution_test.go`'s `TestSelfConsistency_DeterministicRebuild` proves two from-scratch `indexer.Run` passes over the same PHP fixture (composer.json + namespaced interface + class-implementing-interface + top-level function) produce a byte-identical `Export()` stream after normalizing the one volatile `Meta.LastSyncUnixMs` field — the mainstream-tier D-12 validation bar (lighter than priority-4's golden-parity harness, per 05-RESEARCH.md).
- `go build ./...`, `go vet ./...`, and `go test ./internal/indexer/... ./internal/parser/... -count=1` all pass — 13 packages green including the pre-existing Go/Java/C#/Python/TS-JS golden-parity and registry fixtures, unaffected by the three new registrations.

## Per-Language Coverage Assessment (for the D-11 capability matrix, Wave F / plan 05-13)

| Language | Extraction | Same-file resolution | Same-module resolution | Cross-module resolution | Named gaps |
|---|---|---|---|---|---|
| **Rust** | Full (struct/enum/trait/impl-methods/top-level fns) | Yes (empty-PkgAlias calls/embeds) | Yes (files sharing the identical computed `crate::mod::path` moduleKey) | **No** — `use` never populates `Imports` (crate name unknown at `Extract` time) | Nested `mod { }` blocks not descended into; macro-generated items invisible; `generic_function`/`macro_invocation` callees unresolved; trait default-method inheritance not resolved (no conformance-retry pass) |
| **Ruby** | Full (class/module/method/singleton_method) | Yes (empty-PkgAlias `self.`/constant-qualified calls) | No (Ruby's moduleKey is per-file, like Python's) | **No** — `require` binds no local alias; `require_relative` populates `Imports` but only helps when the required file's basename happens to textually match the referenced constant (rare, an accepted limitation) | Bare no-parens no-arg method calls are grammatically ambiguous with local-variable references (a verified, real grammar limitation, not an oversight); `include`/`extend`/`prepend` mixins not extracted; dynamic method definition (`define_method`, `method_missing`) invisible |
| **PHP** | Full (class/interface/trait/methods/functions) | Yes (empty-PkgAlias `$this->`/`self::`/`Type::` calls) | Yes (a `use`-imported simple name resolving against a target file whose own declared `namespace` — or composer.json PSR-4 fallback — computes the matching moduleKey) | **Partial** — works when the `use` target's moduleKey matches exactly; a fully-qualified inline reference (`extends \App\Base`) is recorded by simple name only, not as an alternate lookup key | Trait inclusion (`use TraitName;` inside a class body) not extracted as an edge; `object_creation_expression` (`new Widget()`) not tracked as a calls edge; magic methods (`__call`, `__get`) invisible; a namespace-less file falls back to a less-precise PSR-4/path-derived moduleKey |

All three share the project-wide `T-05-DoS` caveat: `parser.MaxSourceBytes` bounds every parse before any backend-specific parsing runs, but a crash inside any of the three external C scanners (Rust raw strings, Ruby heredocs, PHP tag-switching) is NOT `recover()`-able — the accepted Phase-1 mitigation contract, unchanged by this plan.

## Task Commits

Each task was committed with a RED (test) then GREEN (feat) pair:

1. **Task 1: Rust + Ruby extractors + registrations**
   - `24a1b4e` (test) — `rustextract_test.go`/`rubyextract_test.go` alone (no implementation) confirmed RED via `go vet` failure (`undefined: Extract`)
   - `90902a6` (feat) — `rustextract.go`/`types.go`, `rubyextract.go`/`types.go`, `languages_rust.go`, `languages_ruby.go` + registry tests, confirmed GREEN
2. **Task 2: PHP extractor + registration + self-consistency**
   - `816dd2f` (test) — `phpextract_test.go`/`resolution_test.go` alone confirmed RED via `go vet` failure
   - `72f6c30` (feat) — `phpextract.go`/`types.go`, `languages_php.go` + registry test, confirmed GREEN (including the self-consistency integration test)

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified

- `internal/indexer/mainstream/rustextract/rustextract.go` — Extract, extractor tree-walk, struct/enum/trait/impl/fn node-kind mapping, use-tree traversal, calls
- `internal/indexer/mainstream/rustextract/types.go` — node-kind mapping decisions + full documented rationale for Rust's resolution boundaries (package doc comment)
- `internal/indexer/mainstream/rustextract/rustextract_test.go` — table-driven node-kind mapping, uses, impl-trait embeds, calls, moduleKey pass-through, oversized-file skip
- `internal/indexer/mainstream/rubyextract/rubyextract.go` — Extract, extractor tree-walk, class/module/method node-kind mapping, require/require_relative call-shape detection, superclass, calls
- `internal/indexer/mainstream/rubyextract/types.go` — node-kind mapping decisions + full documented rationale, including the verified bare-call grammar ambiguity
- `internal/indexer/mainstream/rubyextract/rubyextract_test.go` — table-driven node-kind mapping, requires, superclass, calls, moduleKey pass-through, oversized-file skip
- `internal/indexer/mainstream/phpextract/phpextract.go` — Extract, extractor tree-walk, class/interface/trait/method/function node-kind mapping, namespace_use_declaration (plain/grouped/aliased), supertype clauses, three call-expression shapes, namespace parse-time override
- `internal/indexer/mainstream/phpextract/types.go` — node-kind mapping decisions + full documented rationale for PHP's resolution boundaries
- `internal/indexer/mainstream/phpextract/phpextract_test.go` — table-driven node-kind mapping, namespace use, supertype clauses, calls, namespace override, oversized-file skip
- `internal/indexer/mainstream/phpextract/resolution_test.go` — external test package (`phpextract_test`) driving `indexer.Run` end-to-end for the D-12 self-consistency/deterministic-rebuild gate
- `internal/indexer/languages_rust.go` — Rust `LanguageSpec` registration, `readRustDescriptor` (bounded Cargo.toml `[package] name` line-scan), `rustModulePath`
- `internal/indexer/languages_ruby.go` — Ruby `LanguageSpec` registration, `readRubyDescriptor` (Gemfile-presence-only), `rubyModuleKey` (unconditional, extension-stripped)
- `internal/indexer/languages_php.go` — PHP `LanguageSpec` registration, `readPHPDescriptor` (composer.json PSR-4 JSON parse), `phpNamespaceFor` (deterministic longest-prefix match)
- `internal/indexer/languages_test.go` — `TestLanguageRegistry_Rust`/`_Ruby`/`_PHP`, `TestPHPNamespaceFor`

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:

- Rust's `use` resolution is intentionally shallow (dependency-ref tracking only, no `Imports` population) because computing a crate-qualified moduleKey requires information (`Extract`'s caller's crate name) this plan's shared cross-language signature does not carry — documented as an explicit, accepted gap rather than a half-working heuristic.
- Ruby's `require`/`require_relative` needed call-SHAPE recognition (not a dedicated AST node) since Ruby's grammar treats them as ordinary method calls — the extractor must positively identify and exclude this shape from normal calls-edge emission to avoid double-counting the same source location as both an import and a call.
- PHP reuses C#'s exact parse-time-namespace-override architecture verbatim (both languages declare their cross-file identity IN the source, independent of directory layout) rather than inventing a new pattern.
- All three "lesser" structural concepts (Rust enum, Ruby module, PHP trait) map onto the EXISTING `KindStruct` rather than a new kind — consistent with every prior extractor's discipline of extending the vocabulary additively only when a genuinely new semantic exists (D-01).

## Deviations from Plan

**None — plan executed exactly as written.** No Rule 1/2/3 auto-fixes were required; the one real discovery (Ruby's bare-no-parens-call grammar ambiguity) was caught and documented DURING initial test authoring (via a live parse-tree dump), before any implementation was written, so it never manifested as a bug needing a fix — it shaped the test/doc from the start.

## Issues Encountered

- **Ruby's tree-sitter grammar parses a bare, no-parens, no-argument method call (`helper`) as a plain `identifier` node, not a `call` node** — verified via a live parse-tree dump during test authoring (not assumed). This is a genuine grammar-level ambiguity (indistinguishable from a local-variable reference without scope tracking), not an extractor bug. Documented explicitly in `rubyextract/types.go` and reflected in the test suite (`helper()`, with parens, is the form asserted as extracting correctly).
- No live TS CodeGraph CLI or curated Rust/Ruby/PHP corpus was needed for this plan — D-12's mainstream-tier bar is self-consistency + spot-check (not golden-parity), fully satisfied by `TestSelfConsistency_DeterministicRebuild`/`_ExpectedStructure` against a synthetic-but-realistic PHP fixture (composer.json + namespaced interface/class/function). Rust and Ruby rely on their own table-driven extraction tests plus the shared `TestDeterministicRebuild`-class discipline already proven at the indexer level for every registered language.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `internal/indexer/mainstream/{rustextract,rubyextract,phpextract}` and all three `LanguageSpec` registrations are complete, tested, and registered — Rust/Ruby/PHP source files in any indexed repo are now extracted (previously silently skipped as unregistered extensions).
- The per-language coverage assessment table above is ready for direct consumption by 05-13's `docs/LANGUAGE-CAPABILITY-MATRIX.md` (D-11) — every gap named, no silent omissions.
- `go build ./...`, `go vet ./...`, and `go test ./internal/indexer/... ./internal/parser/... -count=1` all pass across 13 packages (the three new mainstream packages plus every pre-existing indexer/parser package), confirming Priority-4 and Go fixtures are unaffected by the new registrations.

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 13 created source files plus this SUMMARY.md confirmed present on disk; all four commits (24a1b4e, 90902a6, 816dd2f, 72f6c30) confirmed in git log.
