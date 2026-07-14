---
phase: 05-language-coverage-resolution-breadth
plan: 09
subsystem: indexing
tags: [resolution, dispatch, interfaces, provenance, tdd, query-engine]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 03
    provides: per-language ModuleKey-keyed symbolIndex, resolveRefsWithIndex's Pass-2 resolution loop shape
  - phase: 05-language-coverage-resolution-breadth
    plan: 04
    provides: javaextract, RefKindEmbeds-shaped extends/implements refs (undistinguished at parse time)
  - phase: 05-language-coverage-resolution-breadth
    plan: 05
    provides: csharpextract, the same RefKindEmbeds-shaped base_list refs
provides:
  - internal/indexer/dispatch package — SynthesizeImplements (Go structural method-set matching, bounded via an inverted methodName->[]interfaceID index, transitive interface-embeds composition)
  - Java/C# declared-implements promotion in resolve.go's RefKindEmbeds branch (Pattern 2) — target-is-interface detection promotes "embeds" to "implements"
  - Two-pass conformance retry (Pitfall 3) closing the inherited-method-call resolution gap for cross-module-key inheritance
  - query.BuildImplementsIndex + dispatchSiblingIDs composed into Callers/Impact — query-time dynamic-dispatch traversal
  - goextract additions: MethodSpec, FileResult.InterfaceMethods, FileResult.MethodArity, EdgeKindImplements constant
affects: [any future phase touching resolve.go's Pass-2 loop, query.Callers/Impact, or LANG-06 mainstream-tier resolvers that may want the same dispatch synthesis]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Bounded structural matching via inverted index: build methodName->[]interfaceID BEFORE any struct is compared, so a struct is only ever compared against interfaces sharing at least one method name — never O(structs x interfaces)"
    - "Provenance-tagged edge synthesis: every heuristic edge sets Provenance=\"heuristic\" + Metadata[\"synthesizedBy\"], flows through the SAME collapseEdges determinism path as ground-truth edges — no parallel dedup"
    - "Deferred-then-retried resolution: a RefKindCalls failure is NOT counted unresolved on pass 1 — it's stashed with a pointer to its owning *schema.File, retried in a second pass once implements/embeds edges exist, and only counted unresolved if BOTH passes fail"
    - "Query-time dispatch composition via sibling-method lookup: BuildImplementsIndex + buildContainsIndex (both fresh-per-call, no caching) let Callers/Impact find, for a queried method, every OTHER type implementing a shared interface with a same-named method — union their reverse-adjacency results in, dedupe by caller"

key-files:
  created:
    - internal/indexer/dispatch/implements.go
    - internal/indexer/dispatch/implements_test.go
  modified:
    - internal/indexer/goextract/types.go
    - internal/indexer/goextract/goextract.go
    - internal/indexer/goextract/goextract_test.go
    - internal/indexer/resolve.go
    - internal/indexer/resolve_test.go
    - internal/indexer/javaextract/resolution_test.go
    - internal/query/traverse.go
    - internal/query/traverse_test.go

key-decisions:
  - "Declared-implements promotion (Pattern 2) is NOT gated by r.Language — a Go struct embedding an interface value genuinely does promote that interface's methods too (real Go semantics), so the same structural rule (target is interface, source is not) applies uniformly across Go/Java/C#"
  - "Go structural implements edges anchor Line at the implementing struct's OWN declaration (no single syntactic reference exists for implicit satisfaction, unlike a declared extends/implements clause which anchors at its own source location)"
  - "An interface with zero method specs (even after composing embeds) is never a synthesis target — Go's interface{} is trivially satisfied by everything; synthesizing that edge for every struct would be pure noise for a dispatch-traversal consumer"
  - "The conformance retry only walks a call's OWN type's supertypes, never the type itself — pass 1's resolveUnqualified already covers same-moduleKey (including same-package inherited) methods, so the retry is scoped exactly to the cross-module-key inheritance gap Pitfall 3 describes"
  - "query.BuildImplementsIndex is a SEPARATE index from BuildReverseAdjacency, not a widened filter — dispatch traversal is name-joined (interface method -> concrete impls' same-named method), structurally different from calls' identity-followed traversal"

patterns-established:
  - "Bounded structural matching via inverted index (see tech-stack.patterns) — the shape any future many-to-many heuristic-edge synthesis pass should follow to avoid O(n x m) blowup"
  - "Deferred-then-retried resolution (see tech-stack.patterns) — the shape for any future multi-pass resolution requirement within Pass 2"

requirements-completed: [RES-02, RES-03]

coverage:
  - id: D1
    description: "Go structural interface satisfaction is synthesized as implements edges, bounded by an inverted methodName->[]interfaceID index (never O(structs x interfaces)), with transitive interface-embeds composition"
    requirement: "RES-02"
    verification:
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#TestSynthesizeImplements_Superset"
        status: pass
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#TestSynthesizeImplements_NonSuperset"
        status: pass
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#TestSynthesizeImplements_EmbeddedInterfaceComposed"
        status: pass
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#TestSynthesizeImplements_BoundedNotQuadratic"
        status: pass
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#TestSynthesizeImplements_Deterministic"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_GoStructuralImplements"
        status: pass
    human_judgment: false
  - id: D2
    description: "Java/C# declared extends/implements (RefKindEmbeds-shaped, undistinguished at parse time) promote to an implements edge iff the resolved target is an interface and the source is not itself an interface; a class-extends-class stays a plain embeds edge"
    requirement: "RES-02"
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_DeclaredImplementsPromotion"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every synthesized edge carries Provenance=\"heuristic\" + Metadata[\"synthesizedBy\"], distinct from ground-truth ast edges; declared promotions carry a non-zero Line anchored at the extends/implements clause; no schema-version bump"
    requirement: "RES-03"
    verification:
      - kind: unit
        ref: "internal/indexer/dispatch/implements_test.go#findImplements (Provenance/Metadata assertion embedded in every SynthesizeImplements test)"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_GoStructuralImplements (Provenance/Metadata/Line assertions)"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_DeclaredImplementsPromotion (Provenance/Metadata/Line assertions)"
        status: pass
      - kind: other
        ref: "grep SchemaVersion internal/schema/meta.go — const SchemaVersion uint32 = 1, unchanged"
        status: pass
    human_judgment: false
  - id: D4
    description: "Two-pass conformance retry: an unqualified call to a method declared only on a supertype in a different module-key scope resolves via a second pass walking the extends/implements chain, closing the inherited-method-call resolution gap"
    requirement: "RES-02"
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_ConformanceRetryResolvesInheritedCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/javaextract/resolution_test.go#TestResolve_InheritedMethodCallResolvesViaConformanceRetry"
        status: pass
    human_judgment: false
  - id: D5
    description: "Callers/Impact traverse dynamic dispatch at query time via a fresh-per-call BuildImplementsIndex composed with a contains index — a call through an interface reaches every concrete implementation's same-named method, without widening BuildReverseAdjacency's RefKindCalls-only filter"
    requirement: "RES-02"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestCallers_DispatchTraversal"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestImpact_DispatchTraversal"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected"
        status: pass
      - kind: other
        ref: "grep 'e.Kind != goextract.RefKindCalls' internal/query/traverse.go — exactly 2 occurrences (BuildReverseAdjacency, Callees), filter unchanged"
        status: pass
    human_judgment: false
  - id: D6
    description: "Full determinism/regression gate: go build/vet, go test -race across every package (including the daemon, watch, cli, mcp, graphstore packages this plan never touched), determinism/golden-parity fixtures green"
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test ./... -race -count=1 (all 17 packages pass)"
        status: pass
      - kind: unit
        ref: "internal/indexer/determinism_test.go#TestDeterministicRebuild"
        status: pass
      - kind: integration
        ref: "testdata/golden/... -run TestGoldenParity(_Java|_CSharp)?"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 09: Interface→Implementation Dispatch Synthesis (RES-02/RES-03) Summary

**Synthesizes `implements` edges (Go structural method-set matching bounded via an inverted index, Java/C# declared-implements promotion) with heuristic provenance, closes the inherited-method-call resolution gap with a two-pass conformance retry, and makes `query.Callers`/`Impact` traverse dynamic dispatch at query time via a fresh-per-call implements index — all additive within SchemaVersion 1.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-07-12
- **Tasks:** 3
- **Files modified:** 9 (2 created, 7 modified)

## Accomplishments
- **Go structural interface satisfaction (Pattern 3):** `internal/indexer/dispatch.SynthesizeImplements` matches a struct's method set against an interface's method-spec set by (name, arity), bounded by an inverted `methodName->[]interfaceID` pre-filter built before any struct is compared — never an O(structs × interfaces) nested loop. Interface-embeds-interface composition is bounded by a visited-set-guarded walk over interfaces alone (cycle-safe, independent of struct count). Proven bounded at n=400 structs/interfaces under a 2s wall-clock ceiling, correctness proven via disjoint-method-name pairing (exactly one match per pair, out of n² candidates).
- **Interface method-spec extraction:** `goextract` now records each interface's own `method_elem` specs (`FileResult.InterfaceMethods`) and every method's declared parameter count (`FileResult.MethodArity`, correctly counting a grouped multi-identifier parameter like `a, b int` as 2, not 1 node) — additive vocabulary, no existing node/edge/ref kind renamed.
- **Java/C# declared-implements promotion (Pattern 2):** `resolve.go`'s `RefKindEmbeds` branch now promotes to a new `EdgeKindImplements` ("implements") whenever the resolved target is an interface node and the source is not itself an interface — applies uniformly across languages (not gated by `r.Language`) since a Go struct embedding an interface value genuinely does satisfy that interface too.
- **Two-pass conformance retry (Pitfall 3):** a `RefKindCalls` ref that fails pass 1 is deferred (not yet counted unresolved), then retried once every `contains`/`embeds`/`implements` edge exists for the whole graph — walking the calling method's owning type's supertype chain (visited-set-guarded BFS) for a same-named method. Closes the cross-module-key inherited-method-call gap without reopening the `RefKindCalls`-only callees scoping or the local-variable-receiver limitation.
- **Query-time dispatch traversal (RES-02):** `query.BuildImplementsIndex` mirrors `BuildReverseAdjacency`'s shape exactly (one `IterateEdges("")` scan, fresh per call) but is a SEPARATE index (not a widened `RefKindCalls`-only filter) since dispatch traversal is name-joined, not identity-followed. `Callers`/`Impact` now compose dispatch siblings (every other implementer's same-named method, reached via a shared `implements` edge) into their traversal — a call through an interface reaches every concrete implementation's callers.
- **Provenance/RES-03 compliance:** every synthesized edge carries `Provenance="heuristic"` + `Metadata["synthesizedBy"]` (`"go-structural-methodset"` or `"declared-implements"`), distinct from ground-truth `"ast"` edges; declared promotions anchor `Line`/`Col` at the extends/implements clause's own source location, Go structural edges anchor `Line` at the implementing struct's own declaration. Zero schema-version bump — `SchemaVersion` stays `1`.

## Task Commits

Each task was committed with a RED (test) then GREEN (feat) pair:

1. **Task 1: implements-edge synthesis (Go structural + Java/C# declared promotion)**
   - `6a20bce` (test) — confirmed RED via a genuine compile failure (removed `dispatch/implements.go`, reverted `resolve.go`/`goextract.go`/`types.go`)
   - `c0c6b51` (feat) — `dispatch.SynthesizeImplements`, `goextract` interface method-spec/arity extraction, `resolve.go`'s Pattern 2 promotion + wiring, confirmed GREEN
2. **Task 2: two-pass conformance retry for inherited-method calls**
   - `7da0a17` (test) — confirmed RED (assertion failure against pre-retry `resolve.go`, both a controlled fixture and a real Java cross-package inheritance fixture through the full `indexer.Run` pipeline)
   - `a4d2a67` (feat) — deferred-pending-calls mechanism + `retryConformanceCalls`/`walkSupertypesForMethod`, confirmed GREEN
3. **Task 3: query-time implements traversal in Callers/Impact**
   - `a8325b3` (test) — confirmed RED (real assertion failures against pre-composition `traverse.go`, via the public `Callers`/`Impact` API)
   - `197c53f` (feat) — `BuildImplementsIndex`, `buildContainsIndex`, `dispatchSiblingIDs`, composed into `Callers`/`Impact`, confirmed GREEN

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/indexer/dispatch/implements.go` - `SynthesizeImplements`, bounded inverted-index matching, transitive interface-embeds composition
- `internal/indexer/dispatch/implements_test.go` - superset/non-superset, embedded-interface composition, empty-interface exclusion, bounded-not-quadratic stress test, determinism
- `internal/indexer/goextract/types.go` - `MethodSpec`, `EdgeKindImplements`, `FileResult.InterfaceMethods`, `FileResult.MethodArity`
- `internal/indexer/goextract/goextract.go` - `collectInterfaceMethods`, `countParams`, `MethodArity` wiring in `emitMethod`
- `internal/indexer/goextract/goextract_test.go` - `TestExtract_InterfaceMethodSpecs`, `TestExtract_MethodArity`
- `internal/indexer/resolve.go` - Pattern 2 promotion in the `RefKindEmbeds` branch, `synthesizeGoImplements` wiring, deferred-pending-calls mechanism, `retryConformanceCalls`/`walkSupertypesForMethod`
- `internal/indexer/resolve_test.go` - `TestResolve_GoStructuralImplements(_NonSuperset)`, `TestResolve_DeclaredImplementsPromotion`, `TestResolve_ConformanceRetryResolvesInheritedCall`
- `internal/indexer/javaextract/resolution_test.go` - `TestResolve_InheritedMethodCallResolvesViaConformanceRetry` (real cross-package inheritance fixture through `indexer.Run`)
- `internal/query/traverse.go` - `BuildImplementsIndex`, `buildContainsIndex`, `implementedInterfaces`, `dispatchSiblingIDs`, composed into `Callers`/`Impact`
- `internal/query/traverse_test.go` - `TestCallers_DispatchTraversal`, `TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected`, `TestImpact_DispatchTraversal`

## Decisions Made
See `key-decisions` in frontmatter for the full list. Highlights:
- Declared-implements promotion applies uniformly across languages (not `r.Language`-gated) — Go's own struct-embeds-interface case genuinely satisfies that interface too
- An empty interface (`interface{}`, even after composing embeds) is never a synthesis target — avoids noise, not a missed requirement
- The conformance retry walks only a call's OWN type's supertypes (never the type itself, which pass 1 already covers) — precisely scoped to the cross-module-key inheritance gap
- `query.BuildImplementsIndex` is genuinely separate code from `BuildReverseAdjacency`, per the plan's explicit instruction — dispatch traversal is name-joined, not identity-followed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Extended goextract to extract interface method-spec names/arity**
- **Found during:** Task 1 planning, before writing dispatch.go
- **Issue:** The plan's `<action>` assumed "the Go extractor already extracts interface method specs as nodes contained by the interface" and directed extending it additively if a gap existed. Investigation confirmed `goextract.go`'s `collectInterfaceEmbeds` only recorded embedded-type references — interface's own `method_elem` specs (needed for structural matching) were never captured anywhere.
- **Fix:** Added `collectInterfaceMethods` (interface's own method_elem specs, NOT flattening embedded interfaces — `dispatch.SynthesizeImplements` composes those itself) and a shared `countParams` helper (used for both interface method-spec arity and `emitMethod`'s arity), storing results in two new additive `FileResult` fields (`InterfaceMethods`, `MethodArity`) rather than new graph nodes — avoids inflating node/edge counts in every existing golden fixture while still giving dispatch synthesis the data it needs.
- **Files modified:** internal/indexer/goextract/types.go, internal/indexer/goextract/goextract.go, internal/indexer/goextract/goextract_test.go
- **Verification:** `TestExtract_InterfaceMethodSpecs`, `TestExtract_MethodArity` pass; full existing goextract suite remains green (no existing test's node/edge count changed)
- **Committed in:** c0c6b51 (Task 1 feat commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical functionality).
**Impact on plan:** Necessary to make Task 1's Go structural matching possible at all — the plan's assumption about existing extraction coverage didn't hold, and this was the smallest-footprint fix (new FileResult fields, not new graph nodes) that unblocked the rest of the task without touching any other language's extraction or any golden fixture's expected shape.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `dispatch.SynthesizeImplements` and the declared-implements promotion pattern are available for any future language's extractor to reuse without further resolve.go changes — a new language only needs to emit `RefKindEmbeds`-shaped unresolved refs (already the shared vocabulary every extractor uses for extends/implements/base_list) to get both Pattern 2 promotion and (if it declares structural interface satisfaction like Go) can wire into `SynthesizeImplements` directly
- `query.BuildImplementsIndex`/`dispatchSiblingIDs` are exported/reusable building blocks any future traversal (e.g. a future `Affected` extension, explicitly out of this plan's scope) can compose the same way Callers/Impact do
- `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` all pass across the full repo (17 packages, including `internal/daemon` — a pre-existing timing-sensitive test flaked once during this session but passed on 3 subsequent runs both with and without this plan's changes present, confirming it's unrelated pre-existing flakiness, not a regression)
- `SchemaVersion` remains `1`; `testdata/golden/...` (Go/Java/C#) and `TestDeterministicRebuild` remain green

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk; all six commits (6a20bce, c0c6b51, 7da0a17, a4d2a67, a8325b3, 197c53f) confirmed in git log.
