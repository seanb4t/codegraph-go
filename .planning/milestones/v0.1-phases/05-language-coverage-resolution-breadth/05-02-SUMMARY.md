---
phase: 05-language-coverage-resolution-breadth
plan: 02
subsystem: indexing
tags: [tree-sitter, multi-language, discovery, worker-pool, determinism, language-registry]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 01
    provides: LanguageSpec registry (registerLanguage/lookupLanguageByID/lookupLanguageByExt), ProjectDescriptor interface, Go registered as the first LanguageSpec
provides:
  - Generalized Discover() — extension->language registry walker + per-language ProjectDescriptor/ModuleKey hook, replacing the hardcoded .go filter and all-or-nothing go.mod error
  - DiscoveredFile.Language field, threaded through Extract
  - Per-worker language-keyed parser cache in extract.go's worker pool (Pitfall 1 fix) — a single Extract() call can now correctly parse files of more than one language
  - Path-based ModuleKey fallback (D-03) for any language whose project descriptor cannot be resolved
affects: [05-03, 05-04, 05-05, javaextract, csharpextract, pyextract, tsextract, mainstream-tier extractors, resolve.go/symbolindex.go generalization]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Discover's two-phase walk: collect (abs,relPath,language) during filepath.WalkDir, then resolve each present language's ProjectDescriptor exactly once per repo root and compute ModuleKey per file — never per-file descriptor resolution"
    - "Per-worker language-keyed parser cache: map[string]parser.Parser inside each extractWithFactory worker goroutine, lazily populated via the registry's NewParser, Close()d for every entry at worker exit"
    - "parserFactory generalized to func(languageID string) (parser.Parser, error) — the testing seam now proves a workers*distinct-languages bound, not just a workers bound"

key-files:
  created:
    - internal/indexer/testdata/mixedlangfixture/ (go.mod, main.go, sub/Greeter.java, app.py, README.md, config.json)
  modified:
    - internal/indexer/discover.go
    - internal/indexer/discover_test.go
    - internal/indexer/extract.go
    - internal/indexer/extract_test.go
    - internal/indexer/pipeline.go

key-decisions:
  - "Relaxed Discover's missing-go.mod behavior from a hard error to Go's own nil-descriptor path fallback (bare relPath, already implemented in languages_go.go from 05-01) — required by D-03's 'never silently drop a supported extension' guarantee; TestDiscover_MissingGoMod (asserted an error) was replaced with TestDiscover_MissingGoMod_FallsBackToPathIdentity (asserts success + fallback ImportPath)"
  - "Discover's second return value stays the Go-specific module path string (not a generalized map) — every existing caller (Sync, Resolve, symbolindex.go) is outside this plan's file scope and consumes it unchanged; it is empty string when no go.mod resolves"
  - "extract_test.go proves the multi-language fix using a 'go-dup' registry entry (same real cgo.NewGoParser/goextract.Extract under a second ID) rather than waiting for a real second-language extractor — this proves the worker-pool selection mechanism is keyed correctly by registry ID without depending on Wave-B plans that haven't landed yet"
  - "discover_test.go registers two inert test-only LanguageSpecs ('java': erroring Descriptor, 'python': nil Descriptor hook) to exercise both descriptor-absent shapes the plan's acceptance criteria require, ahead of real Java/Python extractors"

patterns-established:
  - "Two-phase discovery (collect-then-resolve-descriptors-then-compute-ModuleKey) is the shape every future descriptor hook (pom.xml, *.csproj, pyproject.toml, package.json+tsconfig.json) plugs into without touching the walk itself"

requirements-completed: []
# LANG-02..05 remain unchecked in REQUIREMENTS.md: this plan lands only the
# discovery/extraction seam generalization (Wave A per 05-RESEARCH.md's
# recommendation), not any actual Java/C#/Python/TS extraction+resolution.
# Full per-language requirement completion happens in the Wave B plans that
# consume this seam.

coverage:
  - id: D1
    description: "Discover walks every supported extension via the extension->language registry (not a hardcoded .go filter), reusing ShouldSkipDir verbatim"
    verification:
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MixedLanguage_ExtensionRegistry"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_Fixture"
        status: pass
    human_judgment: false
  - id: D2
    description: "A file whose language has no resolvable project descriptor is still discovered with path-based ModuleKey identity — never dropped"
    verification:
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MixedLanguage_DescriptorAbsentFallback"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MissingGoMod_FallsBackToPathIdentity"
        status: pass
    human_judgment: false
  - id: D3
    description: "Discovery output remains stably sorted by RelPath, and Go's own go.mod ModuleKey/MatchFile behavior is byte-identical to pre-Phase-5"
    verification:
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_SortedByRelPath"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_ImportPaths"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_Deterministic"
        status: pass
    human_judgment: false
  - id: D4
    description: "extract.go's worker pool selects a parser+extractor per file's own language (not once per worker lifetime) — a single Extract() call correctly handles a genuinely mixed-language batch"
    verification:
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_MultiLanguage"
        status: pass
    human_judgment: false
  - id: D5
    description: "Constructed-parser count stays bounded to workers * distinct languages, never one-per-file, and every cached parser is Close()d at worker exit; -race clean"
    verification:
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_MultiLanguage_ParserCountBounded"
        status: pass
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_BoundedNotPerFile"
        status: pass
    human_judgment: false
  - id: D6
    description: "Go's existing single-language extraction behavior is byte-identical after generalization — self-index of this repo is stable across repeated rebuilds, and the golden-parity fixture is unaffected"
    verification:
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_OrderStable"
        status: pass
      - kind: unit
        ref: "internal/indexer/extract_test.go#TestExtractPool_OversizedFileContained"
        status: pass
      - kind: other
        ref: "manual: `codegraph index . -f` run three times, files=115 nodes=921 edges=1722 identical each run"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-11
status: complete
---

# Phase 5 Plan 02: Multi-Language Discovery + Extraction Worker-Pool Fix Summary

**Generalized Discover() to an extension->language registry walker with a two-phase descriptor-resolution/ModuleKey hook, and fixed extract.go's worker pool to select a parser+extractor per file's own language instead of once per worker's whole lifetime — Go's byte-identical behavior preserved throughout.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-11
- **Tasks:** 2
- **Files modified:** 5 (3 test/impl pairs + pipeline.go doc comment), 1 new fixture directory (6 files)

## Accomplishments
- `discover.go`: replaced the hardcoded `filepath.Ext(d.Name()) != ".go"` filter with a `lookupLanguageByExt` registry lookup; added `DiscoveredFile.Language`; `go/build.Context.MatchFile`'s build-tag filtering is now gated to `Language=="go"` only
- Implemented the two-phase walk the plan called for: collect candidate files during `filepath.WalkDir`, then resolve each present language's `ProjectDescriptor` exactly once per repo root and compute `ImportPath` via `LanguageSpec.ModuleKey` — a missing/erroring manifest degrades to that language's own path-based fallback instead of hard-failing `Discover` (D-03)
- Relaxed the pre-Phase-5 "no go.mod = hard error" contract: a root with only Go files and no go.mod now succeeds via Go's existing nil-descriptor `ModuleKey` fallback (bare `relPath`, already implemented in `languages_go.go` from 05-01)
- `extract.go`: fixed the Pitfall-1 worker-pool bug — each worker now owns a `map[string]parser.Parser` cache keyed by `DiscoveredFile.Language`, lazily populated via the registry's `NewParser` and `Close()`d for every entry at worker exit, replacing the single per-worker-lifetime parser
- `parserFactory` generalized from `func() (parser.Parser, error)` to `func(languageID string) (parser.Parser, error)`; `defaultParserFactory` now routes through `lookupLanguageByID` instead of calling `cgo.NewGoParser()` directly
- Extraction dispatch now calls `spec.Extract` (registry lookup by `f.Language`) instead of the hardcoded `goextract.Extract`; a read-failure `FileResult` stamps `f.Language` instead of a hardcoded `"go"`
- `registerLanguage`/`lookupLanguageByExt` from 05-01 are now genuinely consumed by production code (Discover + Extract), not just proven in isolation by `languages_test.go`

## Task Commits

Each task was committed atomically (TDD RED->GREEN per task):

1. **Task 1: Generalize discovery to extension->language registry + descriptor hooks** - `bc76dd6` (test)
2. **Task 2: Fix the worker-pool per-language parser selection (Pitfall 1) + pipeline dispatch** - `b0be033` (fix)

_No plan-metadata commit is separate from these — this SUMMARY's own commit closes out the plan._

## Files Created/Modified
- `internal/indexer/discover.go` - extension->language registry walk, two-phase descriptor resolution, `DiscoveredFile.Language`, Go-gated `MatchFile`
- `internal/indexer/discover_test.go` - mixed-language registry test, descriptor-absent-fallback tests, missing-go.mod fallback test (replaces the old hard-error test), sorted-output test, two inert test-only LanguageSpec registrations
- `internal/indexer/testdata/mixedlangfixture/` - new fixture: `.go`/`.java`/`.py` + two unsupported extensions (`.md`/`.json`) to prove exclusion
- `internal/indexer/extract.go` - per-worker language-keyed parser cache, generalized `parserFactory`, registry-dispatched `Extract`
- `internal/indexer/extract_test.go` - `go-dup` registry entry (real Go parser/extractor under a second ID) driving two new multi-language tests; existing tests updated for the new factory signature and `DiscoveredFile.Language`
- `internal/indexer/pipeline.go` - doc-comment update only (Run's description now reflects registry-driven discovery/extraction; no functional change)

## Decisions Made
- Discover's missing-go.mod behavior changed from a hard error to a path-based fallback — this is the exact behavior D-03 locks in ("a file whose language has no descriptor still gets extracted with path-based identity"), not a scope deviation; the old `TestDiscover_MissingGoMod` test was replaced rather than kept alongside a contradictory new test
- Discover's second return value (`modulePath`) stays Go-specific (a plain string, not a per-language map) since every consumer of it (`sync.go`, `resolve.go`, `symbolindex.go`) is outside this plan's file scope and is deliberately deferred to the resolve.go/symbolindex.go generalization plan
- Proved the extract.go multi-language fix with a `go-dup` registry entry (the real Go parser+extractor registered under a second ID) instead of waiting on a real second-language extractor — this isolates "does the worker pool select the correct cache entry per file's Language" from "is a specific language's extractor correct," which is exactly what this plan's scope is (and is not)
- Two distinct descriptor-absent shapes were exercised in `discover_test.go` (an erroring `Descriptor` func vs. a `nil` `Descriptor` field entirely) since both are legitimate ways a future language's `LanguageSpec` might indicate "no manifest resolution available," and both must degrade identically per D-03

## Deviations from Plan

None — plan executed exactly as written. Both tasks' TDD RED phases were confirmed by temporarily reverting the implementation file and observing genuine compile/test failures before restoring the fix (GREEN).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The extension->language registry, the two-phase descriptor/ModuleKey hook, and the per-worker language-keyed parser cache are now load-bearing in production code — the next Wave-B plans (Java/C#/Python/TS extractors) register real `LanguageSpec`s and the discovery/extraction pipeline picks them up with zero further changes to `discover.go`/`extract.go`
- `resolve.go`/`symbolindex.go` still hardcode Go's `importPath`-shaped `byImportAndName` index (RESEARCH Pitfall 2) — deliberately untouched by this plan's file scope; a mixed-language repo can now be **discovered and extracted** correctly, but Pass 2 resolution still assumes Go semantics until that generalization lands
- `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` all pass across the full repo; `testdata/golden/... -run TestGoldenParity` remains green; a three-run self-index of this repo (`files=115 nodes=921 edges=1722`) confirms Go's extraction/resolution behavior is unchanged

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created/modified files confirmed present on disk; both task commits (bc76dd6, b0be033) confirmed in git log.
