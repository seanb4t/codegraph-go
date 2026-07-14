---
phase: 05-language-coverage-resolution-breadth
plan: 07
subsystem: indexing
tags: [tree-sitter, typescript, tsx, javascript, multi-language, extraction, resolution, cross-file-resolution, tsconfig, golden-parity]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 01
    provides: LanguageSpec registry, ProjectDescriptor interface, cgo.NewTypeScriptParser/NewTSXParser/NewJavaScriptParser + tree-sitter-typescript@v0.23.2/tree-sitter-javascript@v0.25.0 pins
  - phase: 05-language-coverage-resolution-breadth
    plan: 02
    provides: extension->language registry walker, per-worker language-keyed parser cache, per-language Extract dispatch
  - phase: 05-language-coverage-resolution-breadth
    plan: 03
    provides: per-language ModuleKey-keyed symbolIndex (byModuleKeyAndName), addSymbol WR-01 collision handling, resolveSelector's alias-membership boundary
  - phase: 05-language-coverage-resolution-breadth
    plan: 04
    provides: javaextract as a structural analog (PascalCase/camelCase call-qualifier heuristic, WR-02-style synthetic non-matching alias pattern reused)
  - phase: 05-language-coverage-resolution-breadth
    plan: 06
    provides: pyextract's discovery-time-only ModuleKey pattern (directory-structure-derived identity, no parse-time override) as the closest prior analog for TS/JS's own per-file, path-derived identity
provides:
  - internal/indexer/tsextract package — ONE shared extractor serving three grammars (typescript/.ts, tsx/.tsx, javascript/.js/.jsx/.mjs/.cjs); full structural extraction (class/interface/function/method/type_alias, exported-const-as-function/constant) into the shared goextract vocabulary
  - Three LanguageSpec registrations (languages_typescript.go) sharing one Extract function and one tsDescriptor/tsModuleKey pair
  - tsconfig-aware module-specifier resolution: relative-specifier (priority tier, pure path arithmetic) + tsconfig.json paths/baseUrl best-effort (via a package-level Config/SetConfig singleton, since Extract's shared cross-language signature carries no descriptor parameter) + external/node_modules specifiers left unresolved (documented gap)
  - Named-import call/heritage resolution mechanism (namedImportOrigin table) — the ES-modules-specific pattern needed because a named import binds directly into local scope, unlike every other priority-4 language's own qualifier-vs-symbol-name split
  - Cross-file TS/JS resolution proven end-to-end through the real indexer.Run pipeline — relative-specifier calls, tsconfig paths-aliased calls, namespace-import calls, and imported-base-class inheritance all resolve into real committed edges
  - TestGoldenParity_TSJS — a self-skipping D-12 validation harness, smoke-tested against a real 13,464-file TypeScript/TSX/JavaScript repo (ccstatusline) this session
affects: [dispatch synthesis (Wave 6 — extends/implements promotion consumes the RefKindEmbeds shape this plan emits for TS's declared implements), docs/LANGUAGE-CAPABILITY-MATRIX.md's D-11 TS/JS row (tsconfig paths/baseUrl best-effort scope, renamed-default-import gap, node_modules/exports-map gap, directory-style-import gap must be recorded there)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Unconditional (descriptor-independent) per-file ModuleKey: unlike every other priority-4 sibling's 'nil descriptor -> raw relPath' fallback convention, languages_typescript.go's ModuleKey is ALWAYS tsextract.NormalizeModuleKey(relPath) (extension-stripped, repo-root-relative) regardless of whether a tsconfig.json/package.json descriptor was resolved — required because tsextract.Extract's relative-specifier resolution is baseUrl-invariant path arithmetic that must match the target file's own key by construction in every repo, descriptor or not."
    - "Package-level Config/SetConfig singleton as the descriptor-to-Extract side channel: Extract's shared cross-language signature (established 05-01, unchanged by four prior languages) carries no descriptor parameter, so tsconfig.json's paths/baseUrl table — needed only for resolving a NON-relative import specifier — is threaded through a small, mutex-guarded package-level Config installed once per repo root by languages_typescript.go's Descriptor hook (itself called once per Discover run, always before Extract's worker pool starts for that run). A bounded, documented, single-repo-per-process pragmatic choice, not a general multi-root-safe design."
    - "Named-import call/heritage resolution (namedImportOrigin): a bare identifier call/heritage reference first checks a local-alias -> target-module's-own-declared-name table (populated for default/named imports only); a hit emits UnresolvedRef{PkgAlias: localAlias, Name: originName}, satisfying resolveSelector's existing (PkgAlias must be a literal Imports key) contract without any resolve.go change. The ES-modules-specific mechanism every other priority-4 language's own pkg.Symbol()/qualifier-differs-from-symbol-name shape never needed."

key-files:
  created:
    - internal/indexer/tsextract/tsextract.go
    - internal/indexer/tsextract/types.go
    - internal/indexer/tsextract/tsextract_test.go
    - internal/indexer/tsextract/resolution_test.go
    - internal/indexer/languages_typescript.go
    - testdata/golden/parity_tsjs_test.go
  modified:
    - internal/indexer/languages_test.go
    - testdata/golden/golden_parity_test.go

key-decisions:
  - "ModuleKey is unconditionally NormalizeModuleKey(relPath) (extension-stripped, repo-root-relative), regardless of descriptor presence — a deliberate divergence from every sibling's nil-descriptor-fallback convention, required for relative-specifier resolution to work correctly even in a repo with no tsconfig.json/package.json at all."
  - "tsconfig.json paths/baseUrl config is threaded from languages_typescript.go's Descriptor hook to tsextract.Extract via a package-level Config/SetConfig singleton — Extract's shared signature has no descriptor parameter, and generalizing it is outside this plan's file scope (languages.go/extract.go untouched)."
  - "A named/default import's local binding is tracked in a second, TS/JS-specific extractor.namedImportOrigin map (local alias -> target module's own declared name), used only for BARE identifier call/heritage references — a qualified member-access reference (NS.Foo()) resolves through the ordinary Imports-membership check alone, identical to every priority-4 sibling."
  - "A default import resolves correctly only when the local binding text coincides with the target's own declared symbol name (the dominant `export default class Foo{}` + `import Foo from './foo'` idiom) — a renamed default import is an explicit, documented, accepted gap (no per-module 'default export identity' is tracked)."
  - "class_declaration/abstract_class_declaration -> KindStruct (not a new 'class' kind), mirroring every priority-4 sibling's identical decision."
  - "An exported top-level const whose value is an arrow_function/function_expression/generator_function emits a KindFunction node (the dominant modern TS/JS exported-function idiom); any other exported const emits KindConstant; a NON-exported top-level const/let/var emits no node at all (this plan's own bounded 'exported consts as appropriate' scope)."
  - "TestGoldenParity_TSJS implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency) instead of a byte/shape diff against captured TS output — no live TS CodeGraph v1.3.x CLI was available in this environment (identical finding to 05-04/05-05/05-06's Java/C#/Python); self-skips cleanly via CODEGRAPH_TSJS_CORPUS when no corpus is configured, and was smoke-tested this session against a real ccstatusline checkout (13,464 files, all three typescript/tsx/javascript languages present, 83,184 nodes, 123,682 edges, 48,809 calls + 7,205 embeds resolved, deterministic second-pass rebuild)."

patterns-established:
  - "Unconditional descriptor-independent ModuleKey (see tech-stack.patterns) — the shape any future path-derived-identity language whose specifier-resolution mechanism is itself descriptor-invariant should follow."
  - "Package-level Config/SetConfig side channel (see tech-stack.patterns) — the shape any future language needing extra per-repo-root config beyond what ModuleKey's own (descriptor, relPath) signature carries into Extract should reuse, without generalizing the shared LanguageSpec.Extract signature itself."

requirements-completed: [LANG-05]

coverage:
  - id: D1
    description: "tsextract package extracts class/interface/function/method/type_alias declarations (across all three TS/TSX/JS grammars, one shared extractor) into the shared codegraph vocabulary, with field/property definitions correctly emitting no node (mirroring goextract's/every sibling's ratified skip)"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_NoFieldNodes"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_MethodContainsEdge"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_ExportedConsts"
        status: pass
    human_judgment: false
  - id: D2
    description: "A .tsx file with JSX elements parses via the tsx grammar and extracts its surrounding function declaration without error — JSX syntax does not break extraction"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_TSXJSXFixtureParsesWithoutError"
        status: pass
    human_judgment: false
  - id: D3
    description: "import_statement/export-from produce RefKindImports unresolved refs; class_heritage's TS extends_clause+implements_clause (and JS's single direct-expression shape) and an interface's extends_type_clause each produce RefKindEmbeds refs, undistinguished at parse time (Pattern 2); call_expression produces RefKindCalls, correctly disambiguating a PascalCase same-module attempt, a camelCase forced-unresolved local-variable receiver, and a named/default/namespace import"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_ExtendsImplements"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_Imports"
        status: pass
    human_judgment: false
  - id: D4
    description: "typescript/tsx/javascript are all registered in the LanguageSpec registry (resolvable by ID and by .ts/.tsx/.js/.jsx/.mjs/.cjs extensions respectively), sharing one Extract function and one tsconfig.json+package.json Descriptor, with ModuleKey unconditionally NormalizeModuleKey(relPath) regardless of descriptor presence"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_TypeScript"
        status: pass
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_ModuleKeyPassedThroughUnchanged"
        status: pass
    human_judgment: false
  - id: D5
    description: "tsconfig-aware module-specifier resolution resolves both a relative specifier and a tsconfig.json paths-aliased specifier to the correct target moduleKey, and leaves an external/node_modules bare specifier unresolved (documented gap)"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_ModuleSpecifierResolution"
        status: pass
    human_judgment: false
  - id: D6
    description: "Cross-file TS/JS resolution works end-to-end through the real indexer.Run pipeline: relative-specifier calls, tsconfig paths-aliased calls, namespace-import calls, and imported-base-class inheritance each land as real committed calls/embeds graph edges"
    requirement: "LANG-05"
    verification:
      - kind: integration
        ref: "internal/indexer/tsextract/resolution_test.go#TestResolve_RelativeSpecifierCrossFileCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/tsextract/resolution_test.go#TestResolve_TSConfigPathsAliasedCrossFileCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/tsextract/resolution_test.go#TestResolve_NamespaceImportCrossFileCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/tsextract/resolution_test.go#TestResolve_CrossFileInheritance"
        status: pass
    human_judgment: false
  - id: D7
    description: "The parse-failure skip contract matches goextract's/every sibling's (parser.MaxSourceBytes enforced before any backend-specific parsing, across all three grammars); D-12 TS/JS validation harness (shape-not-byte, self-skipping when no corpus is configured), smoke-tested against a real 13,464-file TypeScript/TSX/JavaScript repo; the full repo suite (including the pre-existing Go/Java/C#/Python golden-parity fixtures) remains green under -race with TS/TSX/JS registered"
    requirement: "LANG-05"
    verification:
      - kind: unit
        ref: "internal/indexer/tsextract/tsextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: integration
        ref: "testdata/golden/parity_tsjs_test.go#TestGoldenParity_TSJS (self-skips absent CODEGRAPH_TSJS_CORPUS in the committed repo; manually smoke-tested this session against a real ccstatusline checkout — 13464 files, 83184 nodes, 123682 edges, 48809 calls + 7205 embeds resolved, deterministic rebuild confirmed — see Issues Encountered)"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./... && go test -race ./... ./testdata/golden/... -count=1 (all 18 packages pass, including testdata/golden's pre-existing Go/Java/C#/Python golden-parity fixtures)"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 07: TypeScript/JavaScript Extraction + Cross-File Resolution (LANG-05) Summary

**Full TypeScript/TSX/JavaScript structural extraction via ONE shared `tsextract` package serving three grammars, a tsconfig-aware module-specifier resolver (relative-specifier priority tier + paths/baseUrl best-effort via a package-level Config side channel), an ES-modules-specific named-import call/heritage resolution mechanism, and cross-file resolution (relative, paths-aliased, namespace-import, inheritance) proven end-to-end through the real `indexer.Run` pipeline plus a 13,464-file real-repo smoke test.**

## Performance

- **Duration:** ~30 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 8 (6 created, 2 modified)

## Accomplishments
- Created `internal/indexer/tsextract`: ONE `Extract(p, moduleKey, relPath, src) (goextract.FileResult, error)` serving all three registered grammars (typescript/.ts, tsx/.tsx, javascript/.js/.jsx/.mjs/.cjs) — self-derives which grammar produced a file purely from its own relPath extension, reproducing goextract's exact per-file skip/error contract
- Node-kind mapping (verified directly against each grammar module's own `node-types.json`/`grammar.js`, not assumed): `class_declaration`/`abstract_class_declaration`→`KindStruct`, `interface_declaration`→`KindInterface` (TS/TSX only), `type_alias_declaration`→`KindTypeAlias` (TS/TSX only), `function_declaration`/`generator_function_declaration`→`KindFunction`, `method_definition`→`KindMethod`; an exported `const NAME = (...) => {...}`/function-expression→`KindFunction` (the dominant modern exported-function idiom), any other exported const→`KindConstant`, a non-exported top-level const→no node; `public_field_definition`/`field_definition`→no node (field-skip precedent)
- `class_heritage`'s two distinct grammar shapes (TS/TSX: separate `extends_clause`+`implements_clause` children; JS: a single direct expression child, no `implements` keyword) and an interface's `extends_type_clause` all emit undistinguished `RefKindEmbeds` refs (Pattern 2, extends/implements promotion deferred to Wave 6)
- tsconfig-aware module-specifier resolution in RESEARCH's documented priority order: relative specifier (`./foo`, `../bar`) via pure path arithmetic against the importing file's own relPath — the PRIORITY tier; tsconfig.json `paths`-aliased specifiers via a package-level `Config`/`SetConfig` singleton (Extract's shared cross-language signature carries no descriptor parameter, an architectural constraint of this plan's file scope); best-effort `baseUrl`-relative bare specifiers; external/`node_modules` specifiers explicitly left unresolved (documented gap)
- The ES-modules-specific named-import resolution mechanism this plan had to design: a `namedImportOrigin` table (local alias → target module's own declared symbol name) lets a BARE identifier call/heritage reference (`Foo()`, `class X extends Foo`) resolve through `resolveSelector`'s existing `(PkgAlias must be a literal Imports key)` contract with zero `resolve.go` changes — the mechanism every other priority-4 language's own `pkg.Symbol()`-shaped qualifier never needed, since ES named imports bind directly into local scope
- Registered all three languages in the `LanguageSpec` registry (`languages_typescript.go`): `cgo.NewTypeScriptParser`/`NewTSXParser`/`NewJavaScriptParser` (all pre-pinned since 05-01), a shared `tsconfig.json`+`package.json` `Descriptor` that also installs `tsextract`'s `Config` side channel, path-identity fallback when absent (D-03) — `ModuleKey` is UNCONDITIONALLY `tsextract.NormalizeModuleKey(relPath)` regardless of descriptor presence, a deliberate divergence from every sibling's nil-descriptor-fallback convention (required for relative-specifier resolution correctness even with no tsconfig.json/package.json at all)
- Proved cross-file resolution end-to-end through the REAL `indexer.Run` pipeline: relative-specifier calls, tsconfig paths-aliased calls, namespace-import calls, and imported-base-class inheritance all land as real committed `calls`/`embeds` edges
- `TestGoldenParity_TSJS` implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency), self-skipping cleanly via `CODEGRAPH_TSJS_CORPUS` when unconfigured — and was smoke-tested this session against a real `ccstatusline` checkout (13,464 files, all three `typescript`/`tsx`/`javascript` languages genuinely present: 83,184 nodes, 123,682 edges, 48,809 `calls` + 7,205 `embeds` resolved, byte-identical second-pass rebuild)
- Discovered and fixed, via the pinned `../weft` corpus, that `testdata/golden/golden_parity_test.go`'s `TestGoldenParity/status` subtest hardcoded `status.Languages == ["go","python"]` — weft's three committed `.js`/`.mjs`/`.cjs` files are now genuinely extracted under the shared `"javascript"` `LanguageSpec`, so the assertion needed updating to `["go","javascript","python"]`

## Task Commits

Each task was committed with a RED (test) then GREEN (feat) pair:

1. **Task 1: shared tsextract package + TS/TSX/JS LanguageSpec registrations**
   - `455840e` (test) — `tsextract_test.go` alone (no implementation) confirmed RED via a genuine compile failure (`undefined: Extract`), verified by temporarily removing tsextract.go/types.go
   - `9604e9a` (feat) — tsextract.go/types.go/languages_typescript.go + `TestLanguageRegistry_TypeScript` in languages_test.go, confirmed GREEN
2. **Task 2: cross-file resolution + golden parity** - `8d44f83` (test) — no production code change was needed (Task 1's implementation already resolved correctly); this commit proves it end-to-end through the real pipeline, adds the D-12 golden-parity harness, and fixes the weft `status.Languages` golden assertion

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/indexer/tsextract/tsextract.go` - Extract, extractor tree-walk, node-kind mapping, imports/calls/extends-implements ref emission, module-specifier resolution
- `internal/indexer/tsextract/types.go` - the full documented rationale for node-kind mapping, tsconfig-aware module resolution, and named-import call/heritage resolution (package doc comment)
- `internal/indexer/tsextract/tsextract_test.go` - table-driven node-kind mapping (across all three grammars), TSX JSX fixture, field-skip, extends/implements, calls (3 disambiguation shapes), imports (default/named/namespace), module-specifier resolution (relative/paths-alias/external), exported consts, moduleKey pass-through, oversized-file skip
- `internal/indexer/tsextract/resolution_test.go` - external test package (`tsextract_test`) driving `indexer.Run` end-to-end for 4 cross-file resolution scenarios
- `internal/indexer/languages_typescript.go` - three `LanguageSpec` registrations, `readTSDescriptor` (tsconfig.json/package.json + `tsextract.SetConfig` installation), shared `tsModuleKey`/`tsDescriptor`
- `testdata/golden/parity_tsjs_test.go` - `TestGoldenParity_TSJS`, self-skipping D-12 harness
- `internal/indexer/languages_test.go` - `TestLanguageRegistry_TypeScript`, covering all three IDs/extension sets
- `testdata/golden/golden_parity_test.go` - `TestGoldenParity/status` now asserts `["go","javascript","python"]` against the pinned weft corpus

## Decisions Made
See `key-decisions` in frontmatter for the full list. Highlights:
- `ModuleKey` is unconditionally extension-stripped `relPath` (never descriptor-dependent) — the one deliberate divergence from every priority-4 sibling's own nil-descriptor-fallback convention, required for relative-specifier-resolution correctness
- tsconfig.json's `paths`/`baseUrl` table reaches `Extract` via a package-level `Config`/`SetConfig` singleton rather than a signature change to the shared cross-language `Extract` hook — a bounded, documented, single-repo-per-process pragmatic choice
- Named/default imports need a second internal table (`namedImportOrigin`) beyond `Imports` itself, since ES modules bind an imported symbol's OWN name directly into local scope — the one genuinely new resolution mechanism this plan had to invent (no other priority-4 language needed it)
- A renamed default import (`import Renamed from './foo'`) is an explicit, accepted gap; so is directory-style import resolution (`./utils` → `utils/index.ts`) and `node_modules`/`package.json` `main`/`exports`-map resolution — all documented in `tsextract/types.go`'s package doc rather than guessed at

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stale `status.Languages == ["go","python"]` golden assertion**
- **Found during:** Post-Task-2 full-suite verification against the pinned `../weft` corpus
- **Issue:** weft's committed tree includes three real `.js`/`.mjs`/`.cjs` files that, before this plan, were discovered but never extracted (no JavaScript `LanguageSpec` existed) — `TestGoldenParity/status` hardcoded `status.Languages == ["go","python"]`. Now that tsextract genuinely fires on them (this plan's entire purpose), the old assertion is directly contradicted by correct new behavior, not a regression.
- **Fix:** Updated the assertion to `["go","javascript","python"]` (Languages is sorted ascending).
- **Files modified:** testdata/golden/golden_parity_test.go
- **Verification:** `CODEGRAPH_WEFT_CORPUS=../weft go test ./testdata/golden/... -run TestGoldenParity -v` — all 7 subtests (including status) pass
- **Committed in:** `8d44f83`

---

**Total deviations:** 1 auto-fixed (1 stale-assertion bug, identical class of fix to 05-06's Python plan). 0 scope-creep.
**Impact on plan:** Necessary correction directly caused by this plan's real JavaScript registration reaching the pinned weft corpus's own committed files — no files outside this plan's natural surface touched beyond the one pre-existing golden test that genuinely observes the new behavior.

## Issues Encountered

- **No live TS CodeGraph v1.3.x CLI or curated TS/JS golden corpus committed to this repo** — identical finding to 05-04/05-05/05-06's Java/C#/Python. `TestGoldenParity_TSJS` implements the same documented fallback and self-skips via `CODEGRAPH_TSJS_CORPUS` when unconfigured. UNLIKE Java/C# (manual smoke test only), this plan's environment had a real, representative, license-clean, large TypeScript/TSX/JavaScript repo available as a sibling checkout (`ccstatusline`) — the harness was smoke-tested against its full tree (13,464 files, tsconfig.json present) and confirmed working at real-world scale: 83,184 nodes, 123,682 edges, all three `typescript`/`tsx`/`javascript` languages genuinely present, 48,809 `calls` + 7,205 `embeds` edges resolved, deterministic second-pass rebuild. The test is left in its self-skipping state in the committed repo (no corpus configured by default), matching Java/C#/Python's precedent — selecting and committing a permanent TS/JS validation corpus reference remains a Wave F/D-11-closeout follow-up, same as the other priority-4 languages' equivalent open item.
- **The pre-existing cross-language `resolve.go` `isIntraModule` gap (Go-specific `modulePath`) affects TS "imports" edges identically to every other non-Go language** — a `RefKindImports` unresolved ref never becomes a committed "imports" edge for TS/JS (or Java/C#/Python) today, since `isIntraModule` is gated on `descriptors["go"].ModulePath()`. This does NOT affect `calls`/`embeds` resolution via `Imports`, which is this extractor's own, fully independent mechanism (proven working in `resolution_test.go`) — only the "imports" EDGE itself. Noted per the plan's own critical_constraints; not fixed here (pre-existing, outside this plan's `files_modified` scope, identical finding already documented by 05-05's C# SUMMARY).

## User Setup Required

None - no external service configuration required. Optionally, to exercise `TestGoldenParity_TSJS` against a real repo: `CODEGRAPH_TSJS_CORPUS=/path/to/a/real/ts-or-js/repo go test ./testdata/golden/... -run TestGoldenParity_TSJS -v`.

## Next Phase Readiness
- `internal/indexer/tsextract` and all three TS/TSX/JS `LanguageSpec` registrations are complete and proven through the real pipeline (including a real 13,464-file corpus, not just synthetic fixtures); Wave 6's dispatch synthesis (RES-02) can consume the `RefKindEmbeds` shape this plan emits for `class X extends Y implements Z` to promote qualifying edges to `"implements"`
- The unconditional descriptor-independent `ModuleKey` pattern and the package-level `Config`/`SetConfig` side-channel pattern are both available for any future language needing similar shapes to reuse directly
- Documented, accepted gaps carried forward to Wave F/D-11 closeout: (1) renamed default imports, (2) directory-style imports (`./utils` → `utils/index.ts`), (3) `node_modules`/`package.json` `main`/`exports`-map resolution, (4) a bare cross-namespace reference reached only through TS's `paths` when no `paths` entry matches and `baseUrl` is unset — all should be recorded as named gaps in the D-11 TS/JS capability matrix entry, not silently left implicit
- `go build ./...`, `go vet ./...`, and `go test -race ./... ./testdata/golden/... -count=1` all pass across the full repo (18 packages including the new `tsextract`), with the pre-existing Go/Java/C#/Python golden-parity fixtures remaining green — TS/TSX/JS registration does not regress any prior language

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk; all three commits (455840e, 9604e9a, 8d44f83) confirmed in git log.
