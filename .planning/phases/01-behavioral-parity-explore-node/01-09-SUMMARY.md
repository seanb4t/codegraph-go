---
phase: 01-behavioral-parity-explore-node
plan: 09
subsystem: indexer
tags: [pyextract, tsextract, edge-kinds, d-09, python, typescript, javascript, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 05)
    provides: "The shared resolve.go Pass-2 case arms for references/instantiates/type_of/returns, plus the extends split and overrides synthesis, and goextract's Pass-1 capture pattern to mirror"
  - phase: 01-behavioral-parity-explore-node (plan 08)
    provides: "The Java/C# per-language Pass-1 mirror (field type_of anchored on the enclosing type, local-variable type_of, bounded-allow-list references) this plan follows for the remaining two priority-4 languages"
provides:
  - "Python + TS/JS extractors emit all 6 new D-09 edge kinds (extends/overrides via the shared plan-05 Pass-2 synthesis, references/instantiates/type_of/returns via new per-language Pass-1 capture) — reaching TS's full 9-member RANK_EDGES set for ALL FIVE priority-4 languages, completing D-09/F3"
  - "Python's dynamic-typing precision notes: instantiates folded into recordCall (PascalCase-gated candidate alongside every calls ref, since Python construction is syntactically identical to a plain call), class-body type_of anchored on the enclosing class, un-annotated vars/returns emit nothing (absence, not a fabricated guess)"
  - "TS/JS's new_expression-is-a-distinct-node-kind precision note: instantiates gets its own Pass-1 case (unlike Python), a plain JS file (no type-annotation syntax at all) emits no type_of/returns refs"
affects: ["F4 (re-index this repo's own .codegraph/ with codegraph index --force) and F5 (regenerate the golden corpus) are now unblocked — this was the last F3 slice per RESEARCH §A's ordering constraint F1 -> F3 -> F4 -> F5", "01-06 (query/rwr.go RankEdges set now ranks over Python/TS-JS edges of all 9 kinds too)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Python instantiates folded directly into recordCall (not a separate tree-walk case like Java/C#/TS): Python's `Foo()` is syntactically IDENTICAL to a plain function call, so recordInstantiateCandidate emits a RefKindInstantiates ref alongside every RefKindCalls ref, gated on isLikelyTypeName(name) to avoid flooding unresolvedCount with a candidate for every snake_case call — resolve.go's existing Kind-check disambiguation (unchanged from plan 05) is what actually decides whether it becomes a real edge"
    - "Python type wrapper unwrap: tree-sitter-python wraps EVERY type annotation (return_type field AND assignment's type field) in a `type` node whose single named child is the real expression — pythonTypeRefFromExpr/emitNamedTypeRef unwrap this uniformly for both sites, verified via live parses this session (not assumed from docs)"
    - "Python generic/subscripted annotations use tree-sitter-python's `generic_type` node kind (NOT `subscript` — verified via live parse), unwrapped to its own first named child (the outer type, e.g. Optional in Optional[Foo]) per RESEARCH §B's 'generic/composite types resolve to the outer named type' precision note"
    - "TS/JS instantiates gets its OWN Pass-1 walk case (new_expression, a node kind syntactically DISTINCT from call_expression) — unlike Python, where the two ARE the same node kind — reusing recordCall's exact switch-on-callee-shape structure applied to new_expression's 'constructor' field"
    - "TS/JS type_of/returns reuse the pre-existing typeRefFromExpr + resolveBareIdentifier heritage-clause-resolution helpers (established by an earlier plan for extends/implements parsing), unwrapping the type_annotation wrapper node first — no new type-name-resolution logic was needed, only a new wrapper-unwrap entry point (emitNamedTypeRef)"
    - "Both Python and TS/JS keep primitive/built-in-typed annotations out of type_of/returns: Python via an explicit pythonBuiltinTypes name-filter (no primitive_type node kind exists in tree-sitter-python — built-ins parse as plain identifiers), TS/JS for free via the grammar's own distinct predefined_type node kind (mirrors javaextract/csharpextract's 'grammar already distinguishes' note, no name-filter list needed)"

key-files:
  created:
    - internal/indexer/pyextract/d09_test.go
    - internal/indexer/tsextract/d09_test.go
  modified:
    - internal/indexer/pyextract/pyextract.go
    - internal/indexer/tsextract/tsextract.go

key-decisions:
  - "Python's `instantiates` Pass-1 capture is folded directly into the existing recordCall function (not a parallel object-creation-node walk like every other priority-4 language) because Python has no syntactically distinct construction expression — `Foo()` calling a class IS a `call` node, identical to any function call. A PascalCase gate (isLikelyTypeName) on the callee name trims the candidate set to avoid inflating unresolvedCount for every ordinary snake_case call; the actual class-vs-function disambiguation still happens at Pass 2 via resolve.go's unchanged Kind-check (target must be a struct/class node)."
  - "Python class-body annotated attributes (`x: Foo`) anchor type_of on the ENCLOSING CLASS, mirroring Java/C#'s identical 'no field node' precision note from plan 08 (pyextract has never emitted a field/attribute node — a pre-existing ratified skip); function-body annotated locals anchor on the enclosing method/function, mirroring plan 08's local-variable type_of scope addition."
  - "A small pythonBuiltinTypes name-filter (int/str/float/bool/list/dict/tuple/set/...) keeps Python's built-in type annotations from emitting an unresolvable type_of/returns ref — necessary because tree-sitter-python has no primitive_type node kind (built-ins parse as ordinary identifiers, unlike Go/Java/C#/TS which all have a distinct primitive-type grammar rule)."
  - "TS/JS's `instantiates` is captured via its own new_expression Pass-1 case (not folded into recordCall like Python), since new_expression is a grammar-distinct node kind from call_expression in TS/JS — exactly the same shape javaextract's/csharpextract's object_creation_expression handling already established."
  - "Both extractors' `references` capture mirrors the exact bounded-allow-list discipline established in plans 05/08 (call/constructor arguments, return values, local-variable/assignment right-hand sides, plus common compound-expression wrappers) rather than exhaustive AST coverage — the same false same-module-name-collision risk the earlier plans' D-02 notes document applies identically here."

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "pyextract Pass-1 emit sites for instantiates (a PascalCase-gated candidate folded into recordCall, resolve-time Kind-check disambiguates), type_of (class-body annotated attributes anchored on the enclosing class; function-body annotated locals anchored on the enclosing method/function), returns (reuses the already-parsed return_type field, generic/subscripted annotations resolve to the outer named type), and references (a bounded allow-list of value-read positions, de-duped against calls) — un-annotated vars/returns emit nothing (documented D-02 dynamic-typing divergence)"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/pyextract/ -run 'TestPy(References|Instantiates|TypeOf|Returns|UnannotatedAbsence)' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "tsextract Pass-1 emit sites for the same 4 kinds at TS/JS's tree-sitter anchors (new_expression's own Pass-1 case for instantiates, type_annotation-wrapper unwrap for type_of/returns reusing the pre-existing typeRefFromExpr heritage-clause resolver, call_expression/member_expression-scoped references), predefined/primitive types filtered via TS's distinct predefined_type node kind, plain JS files (no type-annotation syntax) correctly emit no type_of/returns refs"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/tsextract/ -run 'TestTS(References|Instantiates|TypeOf|Returns|UntypedJSAbsence)' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Full repo regression suite remains green — no existing Python/TS-JS behavior (calls, embeds/extends/implements, imports) regressed by the new Pass-1 capture sites; all five priority-4 languages (Go/Java/C#/Python/TS-JS) now emit the full 9-member RANK_EDGES set, completing D-09/F3"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/pyextract/ ./internal/indexer/tsextract/ -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 24min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 9: Python + TS/JS D-09 Edge-Kind Extraction Summary

**Python and TS/JS extractors now emit all 6 new D-09 edge kinds (extends/overrides via the shared plan-05 Pass-2 synthesis, references/instantiates/type_of/returns via new per-language Pass-1 capture at each language's own tree-sitter anchors), completing the priority-4 (Go/Java/C#/Python/TS-JS) coverage of TS's full 9-member RANK_EDGES set — this is the last F3 slice, unblocking F4/F5.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-07-15T14:30:45Z
- **Completed:** 2026-07-15T14:54:33Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Python: folded `instantiates` capture directly into the existing `recordCall` (Python construction is syntactically identical to a plain call — no separate tree-walk case needed), gated on `isLikelyTypeName` to keep unresolvedCount noise down; added `type_of` for both class-body annotated attributes (anchored on the enclosing class, mirroring Java/C#'s "no field node" precision note) and function-body annotated locals (anchored on the enclosing method/function); added `returns` reusing the already-parsed `return_type` field, with generic/subscripted annotations (`Optional[Foo]`) correctly resolving to the outer named type only; added `references` via a bounded allow-list walk (call arguments, return values, assignment right-hand sides), de-duped against calls
- Python: verified tree-sitter-python's actual node shapes via live parses this session rather than assumed grammar docs — discovered every type annotation (both `return_type` and assignment's `type` field) is wrapped in a `type` node, and generic annotations use `generic_type` (not `subscript`)
- TS/JS: added `instantiates` as its own Pass-1 walk case for `new_expression` (a grammar-distinct node kind from `call_expression`, unlike Python), reusing recordCall's exact switch-on-callee-shape structure; added `type_of`/`returns` by reusing the pre-existing `typeRefFromExpr`/`resolveBareIdentifier` heritage-clause-resolution helpers, unwrapping TS's `type_annotation` wrapper node; added `references` via the identical bounded-allow-list discipline, de-duped against calls
- TS/JS: verified TS/JS's actual node shapes via live parses this session — confirmed `type_annotation` wrapping, `new_expression`'s `constructor`/`arguments` fields, and `assignment_expression`'s `left`/`right` fields
- Both languages filter primitive/built-in-typed annotations for free or near-free: TS/JS via the grammar's own distinct `predefined_type` node kind (no per-name filter list needed, mirrors javaextract/csharpextract); Python via a small explicit `pythonBuiltinTypes` name-filter (necessary because tree-sitter-python has no primitive-type node kind — built-ins parse as ordinary identifiers)
- No changes to `resolve.go` — the shared Pass-2 case arms for all 4 new kinds (landed in plan 05) resolve Python's and TS/JS's new refs identically to Go/Java/C#'s
- Full repo test suite green for both packages (`go test ./internal/indexer/pyextract/ ./internal/indexer/tsextract/ -count=1`); a full-repo `go test ./... -count=1` run confirms no regression to any existing extractor behavior — the one failing test (`internal/daemon.TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`) is a pre-existing, unrelated debounced-flush-timing flake (confirmed by re-running it in isolation 3x, all passing) with no relation to this plan's files

## Task Commits

Each task was committed atomically:

1. **Task 1: RED+GREEN Python Pass-1 emit (instantiates/references/type_of/returns) with dynamic-typing absence** - `94c7547` (feat)
2. **Task 2: RED+GREEN TS/JS Pass-1 emit (new_expression/references/type_of/returns)** - `5e9590d` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/indexer/pyextract/pyextract.go` - `pythonBuiltinTypes`, `pythonTypeRefFromExpr`, `emitNamedTypeRef`, `collectClassBodyTypeOf`, `collectReferencesAndInstantiates`, `captureExprRead`, `captureAttributeRead`, `recordInstantiateCandidate`; wired into `emitClass`/`emitFunction`/`emitMethod`/`recordCall`
- `internal/indexer/pyextract/d09_test.go` - `TestPyInstantiates`/`TestPyTypeOf`/`TestPyReturns`/`TestPyReturns_GenericAnnotation`/`TestPyReferences`/`TestPyUnannotatedAbsence`
- `internal/indexer/tsextract/tsextract.go` - `emitNamedTypeRef`, `recordInstantiate`, `collectReferencesAndInstantiates`, `captureExprRead`, `captureMemberAccessRead`; wired into `emitFunction`/`emitExportedConstDeclarator`/`emitMethod`
- `internal/indexer/tsextract/d09_test.go` - `TestTSInstantiates`/`TestTSTypeOf`/`TestTSReturns`/`TestTSReferences`/`TestTSUntypedJSAbsence`

## Decisions Made
See `key-decisions` in frontmatter above — the two highest-signal ones:
1. Python's `instantiates` is the one language where this Pass-1 capture is folded into the EXISTING call-recording function rather than a parallel object-creation walk, because Python has no syntactically distinct construction expression at all — the PascalCase gate is a pure volume-reduction heuristic, not the actual disambiguation (that remains resolve.go's unchanged Kind-check from plan 05).
2. Both languages' grammars were verified via live parses this session (not assumed from memory/docs) before writing any extraction code — this surfaced two non-obvious shapes (Python's universal `type` wrapper node and its `generic_type`-not-`subscript` node kind for annotations) that would have produced silently-wrong extraction if guessed.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs, blocking issues, or missing critical functionality were found. Both extractors were implemented against tree-sitter node shapes verified via live parses this session (not just docs), mirroring the exact Pass-1/Pass-2 pattern established in plans 05 and 08.

### Scope Notes (documented, not corrective)

**1. Python's `instantiates` capture site differs structurally from every other priority-4 language**
- Plan action text described "a `call` whose callee resolves to a class Kind" for Python (mirroring RESEARCH §B), which — unlike Go/Java/C#/TS's syntactically distinct construction expressions — has no separate AST node to anchor a parallel Pass-1 walk on. The implementation therefore extends `recordCall` itself (the ONLY place a Python construction call is ever visited) rather than adding a `collectInstantiates`-shaped sibling function, which was the structural pattern every other language used. This is a direct, unavoidable consequence of Python's grammar, not a deviation from the plan's intent.
- **Files:** `internal/indexer/pyextract/pyextract.go`
- **Verification:** `TestPyInstantiates` proves the candidate is correctly emitted for a PascalCase callee and correctly absent for a snake_case one.

---

**Total deviations:** 0 corrective. 1 documented structural note (Python's instantiates capture site), a necessary consequence of Python's grammar having no distinct construction-expression node kind, not scope creep.
**Impact on plan:** None on the plan's own acceptance criteria — all 9 required tests (5 Python covering both required behaviors plus the generic-annotation precision note, 5 TS/JS covering the untyped-JS absence case) pass exactly as specified, and both packages' full test suites remain green.

## Issues Encountered
None beyond the documented scope note above. Tree-sitter grammar shapes for both languages (Python's `type` wrapper node, `generic_type`'s child-name unwrap, `assignment`'s `type`/`right` fields; TS/JS's `type_annotation` wrapper, `new_expression`'s `constructor`/`arguments` fields, `assignment_expression`'s `left`/`right` fields) were verified via live parses (`ToSexp()`/field-name dumps) this session before writing extraction code, avoiding guesswork against stale grammar docs — the throwaway dump test files used for this verification were removed before implementation began (never committed).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All FIVE priority-4 languages (Go, Java, C#, Python, TS-JS) now emit the full 9-member `RANK_EDGES` set — D-09's extraction-side scope is complete.
- **F4/F5 are now unblocked** (RESEARCH §A's ordering constraint F1 -> F3 -> F4 -> F5): re-indexing this repo's own `.codegraph/` with `codegraph index --force`, and regenerating the golden corpus (`testdata/golden/`) against the new 9-kind graph. This plan was explicitly the LAST F3 slice — those two operational steps are a downstream plan's job, not this one's, per the phase's own F-task ordering discipline (mirrored from plan 05's/08's identical "not yet done, deferred" notes).
- The mainstream-6 languages (Ruby/Rust/Swift/Kotlin/PHP/C/C++) remain out of this addendum's scope per RESEARCH §B, following the existing D-11 full-or-documented-partial capability matrix — any gap there is a per-language divergence under that matrix, not new work implied by this plan.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: internal/indexer/pyextract/pyextract.go
- FOUND: internal/indexer/pyextract/d09_test.go
- FOUND: internal/indexer/tsextract/tsextract.go
- FOUND: internal/indexer/tsextract/d09_test.go
- FOUND: commit 94c7547 (Task 1)
- FOUND: commit 5e9590d (Task 2)
