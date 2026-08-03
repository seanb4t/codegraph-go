---
phase: 01-behavioral-parity-explore-node
plan: 05
subsystem: indexer
tags: [goextract, resolve, edge-kinds, rank-edges, d-09, go-extractor, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 02)
    provides: "The 6 shared RefKind*/EdgeKind* string constants (references/instantiates/returns/type_of/extends/overrides) in goextract's vocabulary"
provides:
  - "Go extractor + resolve.go Pass-2 emit all 6 new D-09 edge kinds — the reference-language slice plans 08/09 mirror for Java/C#/Python/TS-JS"
  - "resolve.go's extends split (Pass-2 reclassification) and overrides synthesis (Pass-2 derivation) — SHARED code every language extractor's tests now assert against"
  - "goextract's Pass-1 capture pattern for references/instantiates/type_of/returns — the extract-time shape plans 08/09 mirror per-language"
affects: [01-06 (query/rwr.go RankEdges set consumes all 9 kinds), 01-08, 01-09 (Java/C#/Python/TS-JS mirror this Pass-1 pattern)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pass-2 branch-split synthesis: resolve.go's existing RefKindEmbeds promotion branch splits three ways (implements/extends/embeds) by target+source Kind, mirroring the pre-existing implements-promotion discipline"
    - "Pass-2 derivation-from-already-resolved-edges synthesis: overrides is computed purely from contains + extends/implements/embeds edges already built, via a shared walkSupertypes BFS primitive (no new tree-walk)"
    - "Pass-1 bounded-allow-list capture for high-volume/ambiguous kinds: references' Go implementation scopes to a documented allow-list of unambiguous read positions rather than exhaustive AST coverage, to avoid false same-package-name-collision resolutions"
    - "Predeclared-type filtering at Pass-1: Go's tree-sitter grammar has no primitive_type node, so returns/type_of filter Go's ~20 predeclared type identifiers (int/string/error/...) before emitting a ref, avoiding unresolvedCount noise"

key-files:
  created: []
  modified:
    - internal/indexer/resolve.go
    - internal/indexer/resolve_test.go
    - internal/indexer/goextract/goextract.go
    - internal/indexer/goextract/goextract_test.go
    - internal/indexer/determinism_test.go
    - internal/indexer/csharpextract/resolution_test.go
    - internal/indexer/javaextract/resolution_test.go
    - internal/indexer/pyextract/resolution_test.go
    - internal/indexer/tsextract/resolution_test.go

key-decisions:
  - "extends is a 3-way split of the pre-existing embeds-promotion branch by (target Kind, source Kind): interface target + non-interface source -> implements (unchanged); non-interface target -> extends (NEW, replaces the old unconditional embeds fallback); interface target + interface source (interface-embeds-interface) -> stays embeds (unchanged) — this is resolve.go's SHARED Pass-2 code, so the split ripples into every language extractor's class-extends-class regression test, not just Go's"
  - "overrides is Go-structural (name+arity) matching, documented as a precision note per RESEARCH §B rather than a drop — Go has no override keyword"
  - "overrides only ever fires across embeds/extends supertype edges in practice, never through implements: an interface's own method signatures never get a real contains (type->method) edge (collectInterfaceMethods only records MethodSpecs for structural implements matching), so an implements-only supertype chain never yields a match"
  - "instantiates' Pass-2 Kind-check disambiguation restricts to KindStruct targets only (Go has no class kind, interfaces can't be composite-literal-constructed, and type_alias underlying types are not resolved here) — a resolved-but-wrong-Kind target counts as unresolved, never silently mis-typed"
  - "references' Go Pass-1 capture is scoped to a bounded allow-list of unambiguous read positions (call arguments, return values, assignment/short-var-decl right-hand sides, composite-literal element values, plus common compound-expression wrappers) rather than exhaustive AST coverage — a documented Go D-02 precision note that avoids the false same-package-name-collision risk an over-broad walk would introduce"
  - "type_of applies only to package-level var declarations with an explicit type (KindVariable nodes, which exist) — struct-field and local-var type_of are an explicit per-language divergence, since this Go extractor emits no field node at all (pre-existing ratified skip) and no local-var nodes, so there is no FromID to anchor either ref on"
  - "returns/type_of filter Go's predeclared type identifiers (bool/byte/int/string/error/any/comparable/...) at Pass-1, since tree-sitter-go has no primitive_type node kind distinguishing them from user-defined type_identifiers — filtering here avoids inflating unresolvedCount with spurious noise for every primitive-typed var/return"

patterns-established:
  - "walkSupertypes: a shared BFS primitive over a type->supertype adjacency map, refactored out of walkSupertypesForMethod so both the calls conformance retry and overrides synthesis reuse the identical traversal shape instead of each writing its own BFS"
  - "namedTypeRef: a single (name, pkgAlias, ok) resolver for 'is this a single named type reference' (type_identifier/qualified_type, optionally one pointer_type level, filtered against Go's predeclared types) — shared by returns/type_of/instantiates instead of three separate unwrap-and-check blocks"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "resolve.go's extends split: a class/struct-extends-class/struct RefKindEmbeds ref promotes to a NEW EdgeKindExtends edge (heuristic provenance, synthesizedBy=declared-extends), while the interface-target implements promotion and interface-embeds-interface cases stay unregressed"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/ -run 'TestResolveExtends' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "resolve.go's overrides Pass-2 synthesis: a method sharing a supertype method's name+arity emits an EdgeKindOverrides edge (heuristic provenance), with an arity mismatch correctly producing no edge"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/ -run 'TestResolveOverrides' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "goextract Pass-1 capture for instantiates (composite_literal/&T{}), references (bounded read-position allow-list, de-duped against calls), type_of (var declarations), and returns (reused ReturnType), each mirroring RefKindCalls' exact extract-time shape"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/goextract/ -run 'TestExtract_Instantiates|TestExtract_InstantiatesUnnamedTypeNoRef|TestExtract_References|TestExtract_ReferencesQualifiedSelector|TestNewKindDedup|TestExtract_TypeOf|TestExtract_Returns' -count=1"
        status: pass
    human_judgment: false
  - id: D4
    description: "resolve.go Pass-2 case arms resolve references/instantiates/type_of/returns into edges, with instantiates' Kind-check disambiguation (target must be a struct node) correctly rejecting a non-struct target"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/ -run 'TestReferences_ResolvesEdge|TestInstantiates_ResolvesToStructKind|TestInstantiates_NonStructTargetUnresolved|TestTypeOf_ResolvesEdge|TestReturns_ResolvesEdge' -count=1"
        status: pass
    human_judgment: false
  - id: D5
    description: "Full regression suite green across the whole repo — including the shared resolve.go extends split rippling correctly into all 4 other language extractors' class-extends-class inheritance tests (csharp/java/py/ts)"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./... -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 19min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 5: Go Reference-Language D-09 Edge-Kind Extraction Summary

**Go extractor + resolve.go now emit all 6 new D-09 edge kinds (extends, overrides, references, instantiates, returns, type_of) alongside the existing 3 (calls, implements, imports) — reaching TS's full 9-member RANK_EDGES set for the first time, and establishing the exact Pass-1/Pass-2 pattern plans 08/09 mirror for Java/C#/Python/TS-JS.**

## Performance

- **Duration:** 19 min
- **Started:** 2026-07-15T12:57:00Z
- **Completed:** 2026-07-15T13:16:17Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Split resolve.go's existing `RefKindEmbeds` promotion branch three ways: interface target + non-interface source → `implements` (unchanged), non-interface target → new `extends`, interface-embeds-interface → stays `embeds` (unchanged) — a pure Pass-2 reclassification of data already captured, no new tree-walking
- Added `synthesizeOverrides`: Pass-2 synthesis deriving `overrides` edges from already-resolved `contains` + `extends`/`implements`/`embeds` edges, matching same-named supertype methods by name+arity (Go's structural precision note per RESEARCH §B)
- Refactored `walkSupertypesForMethod` into a shared `walkSupertypes` BFS primitive, reused by both the calls conformance retry and overrides synthesis
- Added Go Pass-1 capture for `instantiates` (composite_literal/&T{}), `references` (a bounded allow-list of unambiguous value-read positions, de-duped against `calls` by construction), `type_of` (package-level `var` declarations), and `returns` (reusing the already-parsed return-type field)
- Added matching Pass-2 case arms in resolve.go for all four new Pass-1 kinds, including `instantiates`' Kind-check disambiguation (target must resolve to a struct node)
- Updated regression tests across `resolve_test.go` and all 4 other language extractors' `resolution_test.go` files (csharp/java/py/ts) whose class-extends-class fixtures now correctly assert `extends` instead of the pre-D-09 `embeds` edge — a necessary consequence of resolve.go's shared Pass-2 code, not a scope violation
- Full repo test suite (`go test ./... -count=1`) green

## Task Commits

Each task was committed atomically:

1. **Task 1: RED+GREEN Pass-2 extends split + overrides synthesis** - `cc20305` (feat)
2. **Task 2: RED+GREEN Pass-1 capture + Pass-2 resolution for references/instantiates/type_of/returns (Go)** - `2a475d4` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/indexer/resolve.go` - extends split (3-way), overrides synthesis (`synthesizeOverrides`, `walkSupertypes`), Pass-2 case arms for references/instantiates/type_of/returns, extended `retryConformanceCalls`'s supertype-kind switch to recognize `extends`
- `internal/indexer/resolve_test.go` - new extends/overrides tests, new references/instantiates/type_of/returns resolve-level tests, updated the two pre-existing tests whose expectations changed (`TestResolveExtends_StructTarget` renamed from `TestResolve_StructEmbeds`, `TestResolve_DeclaredImplementsPromotion`'s Derived→Base assertion)
- `internal/indexer/goextract/goextract.go` - `namedTypeRef`/`isGoPredeclaredType` helpers, `emitTypeOfRef`, `collectReturnTypeRef`, `recordInstantiate`, `collectReferencesAndInstantiates` + `captureExprRead`/`captureCompositeElement`/`captureSelectorRead`, wired into `emitFunction`/`emitMethod`/`emitConstVarSpec`
- `internal/indexer/goextract/goextract_test.go` - new RED tests for instantiates/references/type_of/returns/dedup
- `internal/indexer/determinism_test.go` - updated Derived→Base assertion (embeds → extends)
- `internal/indexer/{csharpextract,javaextract,pyextract,tsextract}/resolution_test.go` - updated each language's class-extends-class inheritance test assertion (embeds → extends), since resolve.go's Pass-2 split is shared code

## Decisions Made
See `key-decisions` in frontmatter above — the two highest-signal ones:
1. `extends`/`overrides`/`instantiates` all follow the exact Pass-2-synthesis-vs-Pass-1-capture split RESEARCH §B specified, with provenance discipline (`heuristic` + `synthesizedBy`) matching the pre-existing `implements` promotion pattern exactly.
2. Go's `references` capture is deliberately scoped narrower than "every identifier use" — a bounded allow-list documented as a per-language precision note, to avoid the false same-package-name-collision risk an exhaustive walk would introduce (the same risk class the codebase's existing WR-02 discipline already guards against for `calls`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated 4 other language extractors' resolution_test.go regression assertions**
- **Found during:** Task 1 verification (`go test ./internal/indexer/... -count=1`)
- **Issue:** resolve.go's Pass-2 code is SHARED across every language extractor (Pass 1 in each `*extract` package emits the same `RefKindEmbeds`-shaped ref; Pass 2 in `internal/indexer/resolve.go` is the one place that resolves it). Splitting the embeds branch to add `extends` therefore changed behavior for csharpextract/javaextract/pyextract/tsextract's own class-extends-class inheritance tests too, not just Go's — each asserted the pre-D-09 `embeds` edge.
- **Fix:** Updated each language's `TestResolve_*Inheritance` test (csharp: `TestResolve_FullyQualifiedCrossNamespaceInheritance`, java: `TestResolve_ImportedCrossPackageInheritance`, python: `TestResolve_CrossModuleInheritance`, ts: `TestResolve_CrossFileInheritance`) to assert the new `extends` edge instead of `embeds`, plus one Go-level determinism test (`TestRealRepoStructure` in `determinism_test.go`).
- **Files modified:** `internal/indexer/csharpextract/resolution_test.go`, `internal/indexer/javaextract/resolution_test.go`, `internal/indexer/pyextract/resolution_test.go`, `internal/indexer/tsextract/resolution_test.go`, `internal/indexer/determinism_test.go`
- **Verification:** `go test ./... -count=1` green across the whole repo.
- **Committed in:** `cc20305` (Task 1 commit)

**2. [Rule 1 - Bug] Extended retryConformanceCalls' supertype-kind switch to recognize the new "extends" kind**
- **Found during:** Task 1 implementation (before running the test suite, reasoning through the change's ripple effects)
- **Issue:** The conformance retry's supertype-walk (`retryConformanceCalls`, used to resolve an inherited method call through a struct's supertype chain) only recognized `"embeds"` and `EdgeKindImplements` as type→supertype edges. Since a class/struct-extends-class/struct ref now resolves to `"extends"` instead of `"embeds"`, an inherited call through an extends chain would have silently stopped resolving.
- **Fix:** Added `goextract.EdgeKindExtends` to the switch case alongside `"embeds"`/`EdgeKindImplements`.
- **Files modified:** `internal/indexer/resolve.go`
- **Verification:** `TestResolve_ConformanceRetryResolvesInheritedCall` (an existing regression test exercising exactly this path) still passes.
- **Committed in:** `cc20305` (Task 1 commit)

**3. [Rule 1 - Bug] Filtered Go's predeclared type identifiers from returns/type_of capture**
- **Found during:** Task 2 RED test (`TestExtract_Returns`) — a primitive return type (`func Count() int`) unexpectedly produced a `returns` ref, since tree-sitter-go has no `primitive_type` node kind distinguishing `int` from a user-defined `type_identifier`.
- **Issue:** Without filtering, every primitive-typed var/return anywhere in the indexed tree would emit a spurious Pass-1 ref that Pass 2 could never resolve (no such node named `"int"` exists), inflating `unresolvedCount` with noise unrelated to any real gap.
- **Fix:** Added `isGoPredeclaredType` (the Go spec's ~20 predeclared type identifiers) and filtered them inside `namedTypeRef`, so `returns`/`type_of` never emit a ref for a primitive type at all.
- **Files modified:** `internal/indexer/goextract/goextract.go`
- **Verification:** `TestExtract_Returns` (asserts no ref for `int`/`error`) and `TestExtract_TypeOf` pass.
- **Committed in:** `2a475d4` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bugs/necessary consistency fixes surfaced by implementing the plan correctly). No scope creep: all three are direct, unavoidable consequences of resolve.go/goextract being shared code, not independent additions.
**Impact on plan:** None on scope or the plan's own acceptance criteria — all three fixes were required to keep the full regression suite green while implementing exactly what the plan specified.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Go now emits all 9 of TS's RANK_EDGES kinds, ready for plan 06's `query/rwr.go` RankEdges set to consume.
- The extract→resolve Pass-1/Pass-2 pattern for `references`/`instantiates`/`type_of`/`returns` (mirroring `RefKindCalls`' exact shape) and the Pass-2 synthesis pattern for `extends`/`overrides` (mirroring `synthesizeGoImplements`'s composition-from-already-resolved-edges pattern) are both established and ready for plans 08/09 to mirror for Java/C#/Python/TS-JS.
- **Not yet done, deferred to later plans per the phase's F-task ordering (RESEARCH §A):** re-indexing this repo's own `.codegraph/` with `codegraph index --force` (F4) and regenerating the golden corpus (F5) — both explicitly ordered AFTER all priority-4 language extractors land (F3 for every language), not after Go alone. This plan is the Go slice of F3 only.
- Struct-field and local-variable `type_of` remain an explicit, documented Go-side gap (no field/local-var nodes exist in this extractor's vocabulary to anchor those refs on) — any future plan choosing to add field/local-var nodes would need to revisit this.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: internal/indexer/resolve.go
- FOUND: internal/indexer/resolve_test.go
- FOUND: internal/indexer/goextract/goextract.go
- FOUND: internal/indexer/goextract/goextract_test.go
- FOUND: internal/indexer/determinism_test.go
- FOUND: internal/indexer/csharpextract/resolution_test.go
- FOUND: internal/indexer/javaextract/resolution_test.go
- FOUND: internal/indexer/pyextract/resolution_test.go
- FOUND: internal/indexer/tsextract/resolution_test.go
- FOUND: commit cc20305 (Task 1)
- FOUND: commit 2a475d4 (Task 2)
