---
phase: 05-language-coverage-resolution-breadth
plan: 04
subsystem: indexing
tags: [tree-sitter, java, multi-language, extraction, resolution, cross-file-resolution, golden-parity]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 01
    provides: LanguageSpec registry, ProjectDescriptor interface, cgo.NewJavaParser + tree-sitter-java@v0.23.5 pin
  - phase: 05-language-coverage-resolution-breadth
    plan: 02
    provides: extension->language registry walker, per-worker language-keyed parser cache, per-language Extract dispatch
  - phase: 05-language-coverage-resolution-breadth
    plan: 03
    provides: per-language ModuleKey-keyed symbolIndex (byModuleKeyAndName), addSymbol WR-01 collision handling, resolveSelector's alias-membership boundary
provides:
  - internal/indexer/javaextract package — full Java structural extraction (class/interface/method/constructor) into the shared goextract vocabulary
  - Java LanguageSpec registration (ID "java", .java extension, cgo.NewJavaParser, pom.xml/build.gradle Descriptor, path-based ModuleKey fallback overridden at parse time by the file's own `package` declaration)
  - Cross-file Java resolution proven end-to-end through the real indexer.Run pipeline — same-package calls, cross-package imported calls, and cross-package inheritance all resolve into real committed edges
  - TestGoldenParity_Java — a self-skipping D-12 validation harness implementing RESEARCH's documented source-as-specification + self-consistency fallback (no live TS CodeGraph CLI was available to capture a byte-comparable golden fixture)
affects: [05-05, 05-06, 05-07, dispatch synthesis (Wave 6 — extends/implements promotion consumes the RefKindEmbeds shape this plan emits), any future plan touching discover_test.go's language-registry test fixtures]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Parse-time ModuleKey override: a LanguageSpec.ModuleKey function that cannot see file content (its signature is (descriptor, relPath) only) computes a path-based PLACEHOLDER; the language's own Extract function then overrides FileResult.ImportPath once it has parsed an in-source declaration of the language's real cross-file identity (Java's `package` statement) — the pattern every future in-source-declared-identity language (C#'s `namespace`) should follow rather than trying to force identity resolution into the pre-parse ModuleKey hook."
    - "Naming-convention disambiguation for declared-import-optional languages: when a language's own grammar (unlike Go's `pkg.Func()`) allows the SAME `Identifier.method()` syntax to mean either 'a same-package class needing no import' or 'a local variable/field receiver', and the extractor tracks no local-variable type table, Java's PascalCase-vs-camelCase convention is used as the disambiguator — an uppercase-leading, non-imported identifier routes through resolveUnqualified (same-package candidate); a lowercase-leading one routes through the WR-02 synthetic-non-matching-alias pattern (deterministically unresolved, never a same-package false match)."

key-files:
  created:
    - internal/indexer/javaextract/javaextract.go
    - internal/indexer/javaextract/types.go
    - internal/indexer/javaextract/javaextract_test.go
    - internal/indexer/javaextract/resolution_test.go
    - internal/indexer/languages_java.go
    - testdata/golden/parity_java_test.go
  modified:
    - internal/indexer/discover_test.go
    - internal/indexer/languages_test.go

key-decisions:
  - "class_declaration maps to goextract.KindStruct (not a new 'class' kind) — Java has no struct keyword, and reusing KindStruct keeps struct/class-shaped downstream consumers (Wave 6's implements synthesis) working unchanged across languages; documented in javaextract/types.go"
  - "extends/implements are NOT distinguished at parse time (RESEARCH Pattern 2) — both emit a single RefKindEmbeds unresolved ref; promotion to 'implements' is Wave 6's resolve-time job, out of this plan's scope"
  - "Java's real cross-file identity is the file's own declared package (parsed inside javaextract.Extract), overriding the discovery-time path-based ModuleKey placeholder languages_java.go computes — necessary because LanguageSpec.ModuleKey's (descriptor, relPath) signature cannot see file content"
  - "Same-package qualified calls (Helper.assist(), no import needed within Java's own package) are disambiguated from local-variable receivers via PascalCase-vs-camelCase naming convention, since this extractor tracks no local-variable type table — an explicit, documented heuristic rather than a silent gap"
  - "TestGoldenParity_Java implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency) instead of a byte/shape diff against captured TS output — no live TS CodeGraph v1.3.x CLI was available in this environment to capture a NEW golden fixture (05-RESEARCH.md Environment Availability); it self-skips cleanly via CODEGRAPH_JAVA_CORPUS when no corpus is configured"

patterns-established:
  - "Parse-time ModuleKey override (see tech-stack.patterns) — the shape every future in-source-declared-identity language's extractor follows"

requirements-completed: [LANG-02]

coverage:
  - id: D1
    description: "javaextract package extracts class/interface/method/constructor declarations into the shared codegraph vocabulary, with field_declaration correctly emitting no node (mirroring goextract's ratified skip)"
    requirement: "LANG-02"
    verification:
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_NoFieldNodes"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_MethodContainsEdge"
        status: pass
    human_judgment: false
  - id: D2
    description: "import_declaration produces RefKindImports unresolved refs and populates the alias->moduleKey Imports map; extends/implements produce RefKindEmbeds unresolved refs (undistinguished at parse time, Pattern 2)"
    requirement: "LANG-02"
    verification:
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_Imports"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_ExtendsImplements"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_InterfaceExtends"
        status: pass
    human_judgment: false
  - id: D3
    description: "method_invocation calls resolve correctly across three shapes: implicit same-class, imported cross-package static call, and local-variable-receiver (never mis-resolved as same-package, mirroring WR-02)"
    requirement: "LANG-02"
    verification:
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_Calls"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_SamePackageQualifiedCall"
        status: pass
    human_judgment: false
  - id: D4
    description: "Java is registered in the LanguageSpec registry (resolvable by ID and .java extension), with a pom.xml/build.gradle Descriptor and a path-based ModuleKey fallback that a parsed `package` declaration overrides"
    requirement: "LANG-02"
    verification:
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Java"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_PackageDeclarationOverridesModuleKey"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_NoPackageDeclarationKeepsModuleKey"
        status: pass
    human_judgment: false
  - id: D5
    description: "Cross-file Java resolution works end-to-end through the real indexer.Run pipeline: same-package calls, cross-package imported calls, and cross-package inheritance each land as real committed graph edges"
    requirement: "LANG-02"
    verification:
      - kind: integration
        ref: "internal/indexer/javaextract/resolution_test.go#TestResolve_SamePackageCrossFileCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/javaextract/resolution_test.go#TestResolve_ImportedCrossPackageCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/javaextract/resolution_test.go#TestResolve_ImportedCrossPackageInheritance"
        status: pass
    human_judgment: false
  - id: D6
    description: "D-12 Java validation harness (shape-not-byte, self-skipping when no corpus is configured); a parse-failure skip contract matching goextract's; the full repo suite (including the pre-existing Go golden-parity fixture) remains green under -race with Java registered"
    requirement: "LANG-02"
    verification:
      - kind: integration
        ref: "testdata/golden/parity_java_test.go#TestGoldenParity_Java (self-skips absent CODEGRAPH_JAVA_CORPUS; manually smoke-tested against a synthetic corpus during this session — see Issues Encountered)"
        status: pass
      - kind: unit
        ref: "internal/indexer/javaextract/javaextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
      - kind: other
        ref: "go test -race ./... ./testdata/golden/... -count=1 (all packages pass, including testdata/golden/TestGoldenParity for Go)"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 04: Java Extraction + Cross-File Resolution (LANG-02) Summary

**Full Java structural extraction (class/interface/method/constructor) via a `javaextract` package mirroring `goextract`'s shape, registered through the multi-language seam, with cross-file resolution (same-package, cross-package import, cross-package inheritance) proven end-to-end through the real indexer.Run pipeline.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 8 (6 created, 2 modified)

## Accomplishments
- Created `internal/indexer/javaextract`: `Extract(p, moduleKey, relPath, src) (goextract.FileResult, error)` reproducing goextract's exact per-file skip/error contract, reusing the shared `FileResult`/`Kind*`/`RefKind*` vocabulary unchanged (no new kinds needed for this plan's scope)
- Node-kind mapping: `class_declaration`→`KindStruct`, `interface_declaration`→`KindInterface`, `method_declaration`/`constructor_declaration`→`KindMethod`, `field_declaration`→ no node (mirrors Go's ratified skip); `import_declaration`→`RefKindImports`; `superclass`/`super_interfaces`/`extends_interfaces`→`RefKindEmbeds` (extends/implements undistinguished at parse time, Pattern 2); `method_invocation`→`RefKindCalls`
- Java's real cross-file identity (its declared `package a.b.c;` statement) is parsed inside `Extract` and overrides the discovery-time path-based `ModuleKey` placeholder — necessary because `LanguageSpec.ModuleKey`'s `(descriptor, relPath)` signature cannot see file content
- Same-package qualified calls (`Helper.assist()`, legal in Java without an import) are disambiguated from local-variable receivers via PascalCase-vs-camelCase naming convention — an explicit, documented heuristic since this extractor tracks no local-variable type table
- Registered Java in the `LanguageSpec` registry (`languages_java.go`): `cgo.NewJavaParser`, a `pom.xml`/`build.gradle`/`build.gradle.kts` `Descriptor` resolving the Maven/Gradle group identity, path-identity fallback when absent (D-03)
- Proved cross-file resolution end-to-end through the REAL `indexer.Run` pipeline (not unit-level `UnresolvedRef` shape assertions alone): same-package cross-file calls, cross-package calls resolved via the import→ModuleKey mapping, and cross-package inheritance all land as real committed `calls`/`embeds` edges
- `TestGoldenParity_Java` implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency) since no live TS CodeGraph v1.3.x CLI was available in this environment to capture a byte-comparable golden fixture — self-skips cleanly via `CODEGRAPH_JAVA_CORPUS` when unconfigured, and was manually smoke-tested against a synthetic multi-file corpus this session to confirm its own logic (see Issues Encountered)

## Task Commits

Each task was committed with a RED (test) then GREEN (fix/feat) pair, plus one auto-fix commit for a blocking registry collision discovered before Task 1's implementation:

1. **Pre-Task-1 fix: resolve inert "java" LanguageSpec collision** - `9b0485b` (fix, Rule 3 blocking-issue auto-fix)
2. **Task 1: javaextract package + Java LanguageSpec registration**
   - `40be68d` (test) — failing tests confirmed RED (`undefined: Extract`) by temporarily removing the implementation files and observing a genuine compile failure
   - `ac996f0` (feat) — javaextract.go/types.go/languages_java.go, confirmed GREEN
3. **Task 2: cross-file resolution + golden parity** - `a4a3f7b` (test) — no production code change was needed (Task 1's implementation already resolved correctly); this commit proves it end-to-end through the real pipeline

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/indexer/javaextract/javaextract.go` - Extract, extractor tree-walk, node-kind mapping, imports/calls/extends-implements ref emission
- `internal/indexer/javaextract/types.go` - node-kind mapping decisions documented (package doc comment only, no new types — D-01 vocabulary reuse)
- `internal/indexer/javaextract/javaextract_test.go` - table-driven node-kind mapping, imports, extends/implements, calls (3 disambiguation shapes), ModuleKey override, oversized-file skip
- `internal/indexer/javaextract/resolution_test.go` - external test package (`javaextract_test`) driving `indexer.Run` end-to-end for 3 cross-file resolution scenarios
- `internal/indexer/languages_java.go` - Java `LanguageSpec` registration, `readJavaDescriptor` (pom.xml/build.gradle group identity)
- `testdata/golden/parity_java_test.go` - `TestGoldenParity_Java`, self-skipping D-12 harness
- `internal/indexer/discover_test.go` - removed the inert test-only "java" `LanguageSpec` (05-02) that would have collided with the real one; updated the mixed-language fallback assertion
- `internal/indexer/languages_test.go` - `TestLanguageRegistry_Java`, mirroring `TestLanguageRegistry`'s shape for Go

## Decisions Made
- `class_declaration`→`KindStruct` (not a new "class" kind) — keeps struct/class-shaped downstream consumers (Wave 6's implements synthesis) language-agnostic
- Extends/implements are NOT distinguished at parse time (Pattern 2) — a single `RefKindEmbeds` unresolved ref for both; promotion to `"implements"` is Wave 6's resolve-time job
- Parse-time `ModuleKey` override pattern: `languages_java.go`'s `ModuleKey` computes only a path-based placeholder (its signature cannot see source); `javaextract.Extract` overrides `FileResult.ImportPath` with the parsed `package` declaration whenever one exists
- PascalCase-vs-camelCase naming-convention heuristic disambiguates a same-package qualified call from a local-variable receiver, since no local-variable type table is tracked — documented explicitly as a heuristic, not silently assumed
- `resolution_test.go` uses an external test package (`javaextract_test`) rather than the internal `javaextract` package, because driving `internal/indexer.Run` end-to-end from inside `javaextract`'s own package would be an import cycle (`indexer` imports `javaextract` via `languages_java.go`)
- `TestGoldenParity_Java` implements RESEARCH's explicitly-sanctioned source-as-specification + self-consistency fallback (shape coverage + resolved-edge presence + deterministic-rebuild check against a user-configured real corpus) instead of a byte/shape diff against captured TS output, since no live TS CodeGraph v1.3.x CLI was available in this environment to capture a NEW per-language golden fixture

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved inert "java" LanguageSpec registry collision in discover_test.go**
- **Found during:** Pre-Task-1 investigation (reading discover_test.go before writing languages_java.go)
- **Issue:** 05-02 registered a test-only, permanently-erroring "java" `LanguageSpec` in `discover_test.go`'s `init()` (proving the D-03 descriptor-error fallback shape ahead of a real Java extractor landing). Adding the real "java" `LanguageSpec` in `languages_java.go` this plan would register a SECOND entry under the same registry key — `registerLanguage` unconditionally overwrites on a same-ID collision, and Go's package-file `init()` order (alphabetical by filename: `discover_test.go` before `languages_java.go`) meant the real spec would silently win, breaking `TestDiscover_MixedLanguage_DescriptorAbsentFallback`'s assertion (which expected the OLD inert spec's directory-based fallback shape, not the real spec's bare-relPath fallback).
- **Fix:** Removed the inert "java" registration from `discover_test.go`'s `init()` (the "python" one, still needed since no real Python extractor exists yet, stays). Updated the doc comment to explain why, and updated `TestDiscover_MixedLanguage_DescriptorAbsentFallback`'s java assertion to the real `languages_java.go` spec's actual fallback behavior (`sub/Greeter.java`, bare relPath — mirroring Go's own nil-descriptor convention) instead of the old inert spec's `path.Dir(relPath)` shape.
- **Files modified:** internal/indexer/discover_test.go
- **Verification:** `go test ./internal/indexer/... -run TestDiscover -v` — all pass, including the updated fallback assertion
- **Committed in:** `9b0485b` (separate fix commit, landed before Task 1's test/feat pair)

---

**Total deviations:** 1 auto-fixed (1 blocking).
**Impact on plan:** Necessary to land the real Java LanguageSpec without silently breaking a pre-existing, previously-passing test via an undocumented registry collision. No scope creep — the fix is scoped exactly to the collision itself.

## Issues Encountered

- **No live TS CodeGraph v1.3.x CLI or curated Java golden corpus available in this environment.** Per 05-RESEARCH.md's "Environment Availability" table, this was an anticipated possibility with a documented fallback (source-as-specification + self-consistency), which `TestGoldenParity_Java` implements. The test was manually smoke-tested this session by pointing `CODEGRAPH_JAVA_CORPUS` at a synthetic 3-file Java tree (a class with inheritance, a static helper, and a subclass exercising both same-package inheritance and cross-file calls) — confirmed the harness correctly reports shape coverage (`nodeKinds`/`edgeKinds` non-empty, `calls` edges > 0) and the deterministic-rebuild check, then was left in its self-skipping state (no corpus configured in the committed repo) for CI. Selecting and committing a real, license-clean Java validation corpus reference (or capturing a genuine TS-CLI golden fixture, if that CLI becomes available) is a natural follow-up for Wave F (D-11/D-12 closeout), not required by this plan's `must_haves`.

## User Setup Required

None - no external service configuration required. Optionally, to exercise `TestGoldenParity_Java` against a real repo: `CODEGRAPH_JAVA_CORPUS=/path/to/a/real/java/repo go test ./testdata/golden/... -run TestGoldenParity_Java -v`.

## Next Phase Readiness
- `internal/indexer/javaextract` and Java's `LanguageSpec` registration are complete and proven through the real pipeline; Wave 6's dispatch synthesis (RES-02) can consume the `RefKindEmbeds` shape this plan emits for extends/implements to promote qualifying edges to `"implements"`
- The parse-time `ModuleKey`-override pattern (place a path-based placeholder in `LanguageSpec.ModuleKey`, override it inside `Extract` once the language's real in-source identity is parsed) is available for C#'s `namespace` (05-05/whichever plan lands C#) to reuse directly
- `go build ./...`, `go vet ./...`, and `go test -race ./... ./testdata/golden/... -count=1` all pass across the full repo (14 packages including the new `javaextract`), with the pre-existing Go golden-parity fixture (`testdata/golden.TestGoldenParity`) remaining green — Java registration does not regress Go

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk; all four commits (9b0485b, 40be68d, ac996f0, a4a3f7b) confirmed in git log.
