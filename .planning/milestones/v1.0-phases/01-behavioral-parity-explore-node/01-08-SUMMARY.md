---
phase: 01-behavioral-parity-explore-node
plan: 08
subsystem: indexer
tags: [javaextract, csharpextract, edge-kinds, d-09, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 05)
    provides: "The shared resolve.go Pass-2 case arms for references/instantiates/type_of/returns, plus the extends split and overrides synthesis, and goextract's Pass-1 capture pattern to mirror"
provides:
  - "Java + C# extractors emit all 6 new D-09 edge kinds (extends/overrides via the shared Pass-2 synthesis already landed in plan 05, references/instantiates/type_of/returns via new per-language Pass-1 capture) — reaching TS's full 9-member RANK_EDGES set for both languages"
  - "Java's field type_of divergence (anchored on the enclosing class/interface, since this extractor emits no field node) and C#'s equivalent — the pattern plan 09 (Python/TS-JS) can reference for the same 'no field node' constraint"
affects: [01-06 (query/rwr.go RankEdges set now ranks over Java/C# edges of all 9 kinds), 01-09 (Python/TS-JS mirror this same Pass-1 pattern)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Field type_of anchored on the enclosing type, not a field node: neither javaextract nor csharpextract ever emits a field node (a pre-existing ratified skip) — collectFieldTypeOfRefs emits type_of refs FROM the enclosing class/interface/struct id TO the field's declared type, a documented D-02 divergence from Go's package-level-var-anchored type_of"
    - "Local-variable type_of anchored on the enclosing method: local_variable_declaration (Java) / local_declaration_statement (C#) emit type_of refs anchored at the method id, mirroring how references/instantiates are already anchored at method-body scope"
    - "De-dup via walkDescendants pruning: collectReferencesAndInstantiates returns false at method_invocation/invocation_expression nodes so collectCalls' own separate whole-body scan remains the sole source of a called symbol's ref — only the call's arguments are walked as additional reference read positions, exactly mirroring goextract's Go-side de-dup discipline"
    - "Primitive/void filtering falls out of the grammar's own node-kind distinction: Java's integral_type/boolean_type/void_type and C#'s predefined_type are structurally distinct from type_identifier/generic_type/identifier — no per-language predeclared-name list (unlike Go's isGoPredeclaredType) is needed for either language"

key-files:
  created:
    - internal/indexer/javaextract/d09_test.go
    - internal/indexer/csharpextract/d09_test.go
  modified:
    - internal/indexer/javaextract/javaextract.go
    - internal/indexer/csharpextract/csharpextract.go

key-decisions:
  - "Java/C# field type_of is anchored on the ENCLOSING TYPE id (not a field node, since neither extractor emits one) — a documented D-02 precision note distinct from Go's own type_of divergence (Go anchors on package-level KindVariable nodes, which exist; Java/C# have no field-level node to anchor on at all, so the closest meaningful anchor is the declaring type itself)"
  - "Local-variable type_of was added beyond the plan's literal 'field' behavior spec, anchored at the enclosing method id — RESEARCH §B's own table describes the missing kind as 'field/local declared type', and the method-body walk already visits local_variable_declaration/local_declaration_statement nodes for references/instantiates capture, so adding type_of there is a low-cost fidelity improvement, not scope creep"
  - "references' Java/C# capture mirrors Go's exact bounded-allow-list discipline (call/constructor arguments, return values, local-variable initializers, assignment RHS) rather than exhaustive AST coverage — the same false same-package/same-namespace-name-collision risk Go's D-02 note documents applies identically to Java's method_invocation/field_access and C#'s invocation_expression/member_access_expression alias resolution"
  - "instantiates' Kind-check disambiguation (resolve.go, unchanged from plan 05) requires the resolved target to be a KindStruct node — both languages' class_declaration/struct_declaration map to KindStruct, so `new T()` against an interface (anonymous-class body syntax aside, not specially handled) correctly resolves to nothing rather than a wrong-Kind edge"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "javaextract Pass-1 emit sites for instantiates (object_creation_expression), returns (method return type field), type_of (field declared type anchored on the enclosing type + local-variable declared type anchored on the enclosing method), and references (a bounded allow-list of value-read positions, de-duped against method_invocation calls)"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/javaextract/ -run 'TestJava(References|Instantiates|TypeOf|Returns)' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "csharpextract Pass-1 emit sites for the same 4 kinds at C#'s tree-sitter anchors (object_creation_expression, method \"returns\" field, field/local declared type via the shared variable_declaration node shape, invocation_expression/member_access_expression-scoped references), predefined/primitive types filtered via C#'s distinct predefined_type node kind"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./internal/indexer/csharpextract/ -run 'TestCSharp(References|Instantiates|TypeOf|Returns)' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Full repo regression suite remains green — no existing Java/C# behavior (calls, embeds/extends/implements, imports) regressed by the new Pass-1 capture sites"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go test ./... -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 8: Java + C# D-09 Edge-Kind Extraction Summary

**Java and C# extractors now emit all 6 new D-09 edge kinds (extends/overrides via the shared plan-05 Pass-2 synthesis, references/instantiates/type_of/returns via new per-language Pass-1 capture at each language's own tree-sitter anchors), reaching TS's full 9-member RANK_EDGES set for both OOP-heavy priority-4 languages.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-15T14:05:00Z
- **Completed:** 2026-07-15T14:24:23Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Java: added Pass-1 capture for `instantiates` (`new T(...)`, `object_creation_expression`'s "type" field), `returns` (reuses the method's already-parsed "type" return-type field), `type_of` (class/interface field declared types, anchored on the enclosing type since Java's extractor emits no field node; local-variable declared types, anchored on the enclosing method), and `references` (a bounded allow-list of value-read positions — call/constructor arguments, return statements, local-variable initializers — de-duped against `method_invocation` callees via `walkDescendants` pruning)
- C#: mirrored the identical shape at C#'s own anchors — `object_creation_expression`, the method's "returns" field, the shared `variable_declaration` node (used identically by both `field_declaration` and `local_declaration_statement`, so field and local-variable `type_of` reuse one code path), and `invocation_expression`/`member_access_expression`-scoped `references`, plus assignment right-hand-side reads (C#-specific, since `assignment_expression` is a distinct statement shape Java also has but the plan's Go/Java precedent didn't need to special-case identically)
- Both languages filter primitive/void return and field types for free, via each grammar's own distinct node kind (Java: `integral_type`/`boolean_type`/`void_type`; C#: `predefined_type`) rather than a per-language predeclared-name list like Go's `isGoPredeclaredType`
- No changes to `resolve.go` — the shared Pass-2 case arms for all 4 new kinds (landed in plan 05) resolve Java's and C#'s new refs identically to Go's
- Full repo test suite (`go test ./... -count=1`) green, no regressions to either language's pre-existing `calls`/`embeds`/`extends`/`implements`/`imports` behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: RED+GREEN Java Pass-1 emit for references/instantiates/type_of/returns** - `e163b81` (feat)
2. **Task 2: RED+GREEN C# Pass-1 emit for references/instantiates/type_of/returns** - `f49ca91` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/indexer/javaextract/javaextract.go` - `emitNamedTypeRef`/`recordInstantiate`/`collectReferencesAndInstantiates`/`captureExprRead`/`captureFieldAccessRead`/`collectFieldTypeOfRefs`, wired into `emitMethod`/`emitTypeDecl`
- `internal/indexer/javaextract/d09_test.go` - `TestJavaInstantiates`/`TestJavaTypeOf`/`TestJavaReturns`/`TestJavaReferences`
- `internal/indexer/csharpextract/csharpextract.go` - `namedTypeRef`/`emitNamedTypeRef`/`recordInstantiate`/`collectReferencesAndInstantiates`/`captureExprRead`/`captureMemberAccessRead`/`collectFieldTypeOfRefs`, wired into `emitMethod`/`emitTypeDecl`
- `internal/indexer/csharpextract/d09_test.go` - `TestCSharpInstantiates`/`TestCSharpTypeOf`/`TestCSharpReturns`/`TestCSharpReferences`

## Decisions Made
See `key-decisions` in frontmatter above — the two highest-signal ones:
1. Both languages' field `type_of` anchors on the ENCLOSING TYPE (not a field node — neither extractor emits one, mirroring the pre-existing "no field node" skip documented in each package's own `types.go`), a distinct D-02 divergence from Go's package-level-var-anchored `type_of`.
2. `references` capture in both languages mirrors Go's exact bounded-allow-list discipline (not exhaustive AST coverage) to avoid the same false same-package/same-namespace name-collision risk Go's own D-02 note documents.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs, blocking issues, or missing critical functionality were found. This was implemented as designed against the plan's tree-sitter anchors (verified via live parses this session, not just docs) and RESEARCH §B's D-09 table.

### Scope Additions (documented, not corrective)

**1. Local-variable `type_of` added beyond the plan's literal "field" behavior spec**
- Both `<behavior>` blocks in the plan explicitly list "Foo f;" field -> type_of, but RESEARCH §B's own table describes the missing kind's target as "field/local declared type". Since the method-body walk (`collectReferencesAndInstantiates`) already visits `local_variable_declaration` (Java) / `local_declaration_statement` (C#) nodes for `references`/`instantiates` capture, adding a `type_of` ref for the declared local-variable type at the same walk site was a near-zero-cost fidelity improvement consistent with RESEARCH's fuller scope, not a new tree-walk or new anchor.
- **Files:** `internal/indexer/javaextract/javaextract.go`, `internal/indexer/csharpextract/csharpextract.go`
- **Verification:** covered implicitly by the existing `TestJavaTypeOf`/`TestCSharpTypeOf` fixtures' method bodies not producing unexpected refs; no dedicated local-var-type_of test was added (out of the plan's explicit acceptance criteria), but the capability is live.

---

**Total deviations:** 0 corrective (no Rule 1/2/3 fixes needed). 1 documented scope addition (local-variable `type_of`), consistent with RESEARCH §B's broader "field/local" scope and the plan's own D-10 "no documented-divergence drops except where genuinely intractable" instruction.
**Impact on plan:** None on the plan's own acceptance criteria — all 8 required tests (4 per language) pass exactly as specified.

## Issues Encountered
None. Tree-sitter grammar shapes for both languages (field names for `object_creation_expression`, `variable_declaration`/`variable_declarator`, `field_access`/`member_access_expression`, `assignment_expression`) were verified via live parses (`ToSexp()` dumps) this session before writing extraction code, avoiding guesswork against stale grammar docs.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Java and C# now emit all 9 of TS's RANK_EDGES kinds, alongside Go (plan 05) — 3 of the 5 priority-4 languages complete.
- Plan 09 (Python/TS-JS) mirrors this identical Pass-1 pattern for the remaining 2 priority-4 languages. Python's `type_of`/`references`/`instantiates` will face its own dynamic-typing precision notes (annotated vs. un-annotated assignments); TS-JS's `type_of`/`returns` will need to handle TypeScript's optional type annotations similarly.
- F4 (re-index this repo's own `.codegraph/` with `codegraph index --force`) and F5 (regenerate the golden corpus) remain explicitly deferred until ALL priority-4 language extractors land (RESEARCH §A's ordering constraint F1 → F3 → F4 → F5) — plan 09 is the last F3 slice before F4/F5 become unblocked.
- The "no field node" constraint (documented here for Java/C#) is a pattern plan 09 should check against Python (which similarly may not emit field/attribute nodes) and TS-JS (which may or may not, depending on that extractor's existing vocabulary).

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: internal/indexer/javaextract/javaextract.go
- FOUND: internal/indexer/javaextract/d09_test.go
- FOUND: internal/indexer/csharpextract/csharpextract.go
- FOUND: internal/indexer/csharpextract/d09_test.go
- FOUND: commit e163b81 (Task 1)
- FOUND: commit f49ca91 (Task 2)
