---
phase: 05-language-coverage-resolution-breadth
plan: 01
subsystem: indexing
tags: [tree-sitter, cgo, go-tree-sitter, language-registry, multi-language, java, csharp, javascript, typescript]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline
    provides: goextract.Extract, FileResult/ExtractedNode/IntraEdge/UnresolvedRef vocabulary, Kind*/RefKind* constants
  - phase: 01-foundation
    provides: parser.Parser seam, internal/parser/cgo (NewGoParser/NewPythonParser), MaxSourceBytes ceiling
provides:
  - Five priority-4 CGo grammar parser constructors (Java, C#, JavaScript, TypeScript, TSX)
  - Four grammar go.mod pins at exact researched versions
  - LanguageSpec registry (internal/indexer/languages.go) keyed by language ID and extension
  - Go registered as the first LanguageSpec, proving the seam end-to-end
  - KindRoute additive vocabulary constant for Wave D (framework routing)
affects: [05-02, 05-03, 05-04, 05-05, discovery generalization, extract.go worker-pool fix, resolve.go per-language dispatch]

# Tech tracking
tech-stack:
  added:
    - github.com/tree-sitter/tree-sitter-java@v0.23.5
    - github.com/tree-sitter/tree-sitter-c-sharp@v0.23.5
    - github.com/tree-sitter/tree-sitter-javascript@v0.25.0
    - github.com/tree-sitter/tree-sitter-typescript@v0.23.2
  patterns:
    - "Per-language CGo parser constructor: New<Lang>Parser() routes through the single newCGoParser(languagePtr)/Parse seam — no bypass of the MaxSourceBytes ceiling"
    - "Registry-keyed-by-ID: package-level var registry = map[string]T{}, populated via each language's own init(), looked up by stable string ID/extension, never rebuilt per call"

key-files:
  created:
    - internal/indexer/languages.go
    - internal/indexer/languages_go.go
    - internal/indexer/languages_test.go
  modified:
    - internal/parser/cgo/parser_cgo.go
    - internal/parser/cgo/parser_cgo_test.go
    - internal/indexer/goextract/types.go
    - go.mod
    - go.sum

key-decisions:
  - "TypeScript module exposes two accessors in one bindings/go package (LanguageTypescript/LanguageTSX) — confirmed directly against the module's own binding_test.go before wiring, per plan instruction not to guess"
  - "ProjectDescriptor declared as a minimal interface (ModulePath() string) in languages.go, with goProjectDescriptor as Go's first concrete implementation wrapping the existing readModulePath/importPathFor logic — kept forward-compatible for Wave 2's discover.go generalization without redesigning it now"
  - "Manually promoted the four new grammar go.mod requires from indirect to direct (go get leaves them indirect since parser_cgo.go wasn't yet importing them) — no go mod tidy run, per project convention"

patterns-established:
  - "Registry-keyed-by-ID (languages.go): every subsequent language registers itself via its own init() calling registerLanguage, mirroring the parser/cgo per-constructor pattern"

# requirements-completed intentionally empty: LANG-02..05 (listed in this plan's
# frontmatter requirements field) are NOT fully satisfied by this plan — it lands
# only the seam (registry + parser constructors). Full Java/C#/Python/TS/JS
# extraction+resolution lands in later Phase 5 plans; REQUIREMENTS.md checkboxes
# must stay unchecked until each language's extractor actually ships.
requirements-completed: []

coverage:
  - id: D1
    description: "Five priority-4 CGo grammar parser constructors (NewJavaParser, NewCSharpParser, NewJavaScriptParser, NewTypeScriptParser, NewTSXParser) all routed through the existing newCGoParser/Parse size-ceiling seam — a prerequisite for LANG-02/03/05, not their full completion"
    verification:
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoParsesPriority4Sources"
        status: pass
    human_judgment: false
  - id: D2
    description: "LanguageSpec registry resolves Go by both ID and extension, returning a working parser + extractor — the seam every priority-4/mainstream language's extractor will register against"
    verification:
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry"
        status: pass
    human_judgment: false
  - id: D3
    description: "KindRoute added additively to the shared vocabulary — no pre-existing Kind* constant renamed or removed"
    verification:
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestKindRouteAdditive"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 01: Multi-Language Seam Foundation Summary

**LanguageSpec registry keyed by language ID + five priority-4 CGo grammar parser constructors (Java, C#, JavaScript, TypeScript, TSX), proven against Go's existing extractor before any new language lands.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 8 (3 created, 5 modified)

## Accomplishments
- Added `NewJavaParser`, `NewCSharpParser`, `NewJavaScriptParser`, `NewTypeScriptParser`, `NewTSXParser` to `internal/parser/cgo/parser_cgo.go`, each a one-liner through the existing `newCGoParser`/`Parse` seam (MaxSourceBytes ceiling enforced uniformly, T-05-DoS mitigated)
- Pinned the four priority-4 grammar modules at the exact researched versions (`tree-sitter-java@v0.23.5`, `tree-sitter-c-sharp@v0.23.5`, `tree-sitter-javascript@v0.25.0`, `tree-sitter-typescript@v0.23.2`) as direct `go.mod` requires
- Created `internal/indexer/languages.go`: the `LanguageSpec` struct, `ProjectDescriptor` interface, package-level `registry`/`extToLang` maps, `registerLanguage`/`lookupLanguageByID`/`lookupLanguageByExt`
- Created `internal/indexer/languages_go.go`: registers Go as the first `LanguageSpec` via `init()`, delegating `ModuleKey`/`Descriptor` to the existing `importPathFor`/`readModulePath` logic and reusing `goextract.Extract` verbatim (signature already matches)
- Added `KindRoute = "route"` to `goextract/types.go`, additive — every pre-existing `Kind*`/`RefKind*` constant unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Priority-4 CGo parser constructors + grammar pins** - `3ef9aa2` (feat)
2. **Task 2: LanguageSpec registry + additive vocabulary + Go registration** - `13a7704` (feat)

_No plan-metadata commit is separate from these — this SUMMARY's own commit closes out the plan._

## Files Created/Modified
- `internal/parser/cgo/parser_cgo.go` - five new priority-4 `New<Lang>Parser` constructors + grammar accessor imports
- `internal/parser/cgo/parser_cgo_test.go` - `TestCGoParsesPriority4Sources` table-driven test parsing a trivial valid snippet per new language
- `internal/indexer/goextract/types.go` - `KindRoute = "route"` added additively
- `internal/indexer/languages.go` - new `LanguageSpec`/`ProjectDescriptor` types + registry
- `internal/indexer/languages_go.go` - Go's `LanguageSpec` registration
- `internal/indexer/languages_test.go` - `TestLanguageRegistry`, `TestKindRouteAdditive`
- `go.mod` / `go.sum` - four new direct grammar requires

## Decisions Made
- TypeScript's two grammars (typescript/tsx) confirmed via the module's own `bindings/go/binding_test.go` (`LanguageTypescript()`, `LanguageTSX()`) rather than assumed from the plan's illustrative naming
- `ProjectDescriptor` declared as a minimal `ModulePath() string` interface, forward-compatible for Wave 2's discover.go generalization (pom.xml/csproj/pyproject/tsconfig descriptors) without redesigning the seam now
- Four new grammar `go.mod` requires manually promoted from `// indirect` to direct (go get alone left them indirect since parser_cgo.go's new imports came after the go get run) — no `go mod tidy`, per established project convention (Phase 1/2/3/4 precedent)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Did not mark LANG-02..05 complete in REQUIREMENTS.md despite the plan's frontmatter `requirements` field listing them**
- **Found during:** state_updates step (after both tasks committed)
- **Issue:** This plan's frontmatter declares `requirements: [LANG-02, LANG-03, LANG-04, LANG-05]`, and the standard state-update step runs `requirements mark-complete` against that list. But this plan only lands the seam (registry + parser constructors) — zero Java/C#/Python/TS/JS extraction or resolution logic exists yet (that is Wave B+ per RESEARCH's plan-structure recommendation, later plans in this same phase). Blindly running `requirements mark-complete LANG-02 LANG-03 LANG-04 LANG-05` checked off "full extraction + resolution" for four languages that have no extractor at all, which would misrepresent phase progress to every downstream planner/executor reading REQUIREMENTS.md.
- **Fix:** Reverted the four checkboxes and the traceability-table rows in REQUIREMENTS.md back to unchecked/Pending; left `requirements-completed: []` in this SUMMARY's frontmatter with an explanatory comment instead of listing the four IDs.
- **Files modified:** .planning/REQUIREMENTS.md
- **Verification:** REQUIREMENTS.md now shows LANG-02..05 as `[ ]` Pending, consistent with the actual shipped scope (registry + parser seam only)
- **Committed in:** plan-metadata commit (this SUMMARY's own commit)

## Issues Encountered

None. `gofmt` flagged the initial parser_cgo_test.go table literal alignment; reformatted with `gofmt -w` before commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The `LanguageSpec` registry and priority-4 parser constructors are in place; Wave 2 (discover.go generalization, extract.go worker-pool per-file language dispatch, resolve.go/symbolindex.go per-language ModuleKey) can now build directly on this seam
- `internal/indexer/extract.go`'s worker pool still hardcodes `defaultParserFactory`/`goextract.Extract` (Pitfall 1) — unchanged by this plan, deliberately deferred to the next plan in Wave A per the plan's stated scope (registry-first, before touching the dispatch points that consume it)
- `go build ./...` and `go test ./...` pass across the full repo (all 13 packages), confirming zero regression to existing Go extraction/resolution/query/CLI behavior

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created files confirmed present on disk; both task commits (3ef9aa2, 13a7704) confirmed in git log.
