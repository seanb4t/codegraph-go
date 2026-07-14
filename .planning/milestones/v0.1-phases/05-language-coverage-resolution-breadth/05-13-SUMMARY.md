---
phase: 05-language-coverage-resolution-breadth
plan: 13
subsystem: indexing
tags: [documentation, capability-matrix, golden-parity, determinism, regression, closeout]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 09
    provides: dispatch synthesis (RES-02/RES-03) — the D4/D5/D6/D7 "full" dispatch claims for go/java/csharp/typescript/tsx consumed here
  - phase: 05-language-coverage-resolution-breadth
    plan: 10
    provides: "Per-Language Coverage Assessment table for Rust/Ruby/PHP — consumed verbatim into the D-11 matrix"
  - phase: 05-language-coverage-resolution-breadth
    plan: 11
    provides: "Per-Language Coverage Assessment table for C/C++/Swift/Kotlin — consumed verbatim into the D-11 matrix"
  - phase: 05-language-coverage-resolution-breadth
    plan: 12
    provides: framework routing (LANG-07) — the routing=full claims for go/java/csharp/python/typescript/tsx/javascript
provides:
  - "internal/indexer/capability package — CapabilityEntry machine-readable D-11 descriptor keyed by language ID, Lookup/All accessors"
  - "docs/LANGUAGE-CAPABILITY-MATRIX.md — human-readable mirror, self-consistency-tested against the Go descriptor"
  - "indexer.RegisteredLanguageIDs() — exported registry accessor the matrix's own consistency test consumes"
  - "matrix_test.go's D-11/D-12 phase gate: full priority-4 entries map to an existing golden-parity test function; every partial entry names ≥1 gap; doc and descriptor stay byte-identical on gap text"
  - "Cross-language regression closeout: go test -race ./... and testdata/golden/... both green, confirming Wave A's generalization did not regress Go's own golden-parity fixture or the project-wide determinism gate"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Gaps is not restricted to non-full axes only: an otherwise-'full' capability may still carry a documented heuristic-boundary caveat (e.g. Java's PascalCase/camelCase call-qualifier heuristic), since D-11's 'documented-partial means written down' discipline applies to every honestly-discovered limitation, not only ones severe enough to drop a coverage value."
    - "Doc/descriptor consistency enforced by literal substring matching: every Gaps[] string in matrix.go must appear byte-for-byte in docs/LANGUAGE-CAPABILITY-MATRIX.md, and every markdown table row's four coverage cells are regex-parsed and compared against the Go descriptor — a future edit to one half without the other fails go test."
    - "Golden-parity-test-existence check via go/parser, not import: matrix_test.go cannot import testdata/golden (would be a test-only package with no exported API), so it parses testdata/golden/*_test.go's own AST to confirm a given TestGoldenParity_* function is actually declared, rather than assuming from the plan document."

key-files:
  created:
    - internal/indexer/capability/matrix.go
    - internal/indexer/capability/matrix_test.go
    - docs/LANGUAGE-CAPABILITY-MATRIX.md
  modified:
    - internal/indexer/languages.go

key-decisions:
  - "Python and JavaScript's Dispatch axis is 'none', not 'full', despite both being priority-4 languages: pyextract emits no KindInterface nodes (Python has no declared-interface construct) and JavaScript's own grammar has no interface_declaration/implements clause — RES-02's Pattern-2 promotion structurally never fires for either, so claiming 'full' dispatch would overstate what 05-06/05-07's own SUMMARYs actually built. TypeScript and TSX (which DO have interface_declaration + implements clauses) are 'full'."
  - "Mainstream-6 Dispatch is 'partial' (not 'none') for the four languages whose extractors emit an interface-shaped node AND a supertype RefKindEmbeds ref (Rust's trait_item, PHP's interface_declaration, Swift's protocol_declaration, Kotlin's interface_declaration) — the promotion mechanism genuinely fires within each language's own documented same-file/same-module resolution scope. Ruby, C, and C++ are 'none': none of their extractors ever emit an interface-shaped node, so promotion never fires at all."
  - "Routing is 'none' for every mainstream-6 language — LANG-07/05-12 implemented exactly the five priority frameworks (Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS), none of which target Rust/Ruby/PHP/C/C++/Swift/Kotlin. This is accurately 'none' with a named gap, not a claim of partial coverage."
  - "The cross-language resolve.go isIntraModule/modulePath imports-edge limitation (first documented in 05-05/05-07's SUMMARYs) is recorded as a Shared Caveat in the doc, not as a per-language Resolution downgrade — it affects only the 'imports' edge (a file-level dependency edge), not the calls/embeds resolution every priority-4 golden-parity/self-consistency test actually proves. Downgrading Resolution to 'partial' for every non-Go language over a gap that doesn't affect calls/embeds would be a less accurate signal, not a more honest one."
  - "Task 2's consistency-check tests (TestMatrix_FullPriority4EntriesHaveGoldenTest, TestMatrix_PartialEntriesNameGaps) were authored together with Task 1's matrix.go/matrix_test.go/doc in a single commit, since matrix_test.go is one file with no natural split point between 'the tests that prove the matrix is internally consistent' and 'the tests that prove it against the golden-parity/gap-naming phase gate' — both were designed and verified together before the first commit."

patterns-established:
  - "Gaps-is-general-purpose-not-non-full-only (see tech-stack.patterns) — the shape any future capability/coverage descriptor in this codebase should follow when a 'full' claim still has honest caveats worth surfacing."

requirements-completed: [LANG-06]

coverage:
  - id: D1
    description: "internal/indexer/capability/matrix.go covers exactly the 14 registered language IDs (go/java/csharp/python/typescript/tsx/javascript/rust/ruby/php/c/cpp/swift/kotlin), each cell one of full|partial|none, every non-full cell naming ≥1 gap"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/capability/matrix_test.go#TestMatrix_CoversRegisteredLanguages"
        status: pass
      - kind: unit
        ref: "internal/indexer/capability/matrix_test.go#TestMatrix_CellsValid"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/LANGUAGE-CAPABILITY-MATRIX.md's table and per-language gap bullets are byte-for-byte consistent with the Go descriptor (same coverage values, same gap text)"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/capability/matrix_test.go#TestMatrix_DocMirrorsDescriptor"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every full Resolution/Dispatch entry for a priority-4 language maps to an existing (self-skipping) golden-parity test function in testdata/golden/; every partial entry names ≥1 gap (D-11/D-12 phase gate)"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/indexer/capability/matrix_test.go#TestMatrix_FullPriority4EntriesHaveGoldenTest"
        status: pass
      - kind: unit
        ref: "internal/indexer/capability/matrix_test.go#TestMatrix_PartialEntriesNameGaps"
        status: pass
    human_judgment: false
  - id: D4
    description: "Cross-language regression closeout: go test -race ./... is green across all 26 packages; testdata/golden's Go golden-parity fixture (TestGoldenParity) and priority-4 self-skipping harnesses (Java/C#/Python/TS-JS) remain green; the determinism gate (TestDeterministicRebuild) still passes"
    verification:
      - kind: other
        ref: "go test -race ./... -count=1 (26 packages, all pass)"
        status: pass
      - kind: integration
        ref: "go test -race ./testdata/golden/... -count=1 -v (TestGoldenParity PASS with all 7 subtests; TestGoldenParity_Java/_CSharp/_Python/_TSJS self-skip cleanly, no corpus configured)"
        status: pass
      - kind: unit
        ref: "internal/indexer#TestDeterministicRebuild"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 13: Language Capability Matrix + Cross-Language Regression Closeout (LANG-06) Summary

**D-11 language capability matrix shipped in both machine-readable (`internal/indexer/capability`) and human-readable (`docs/LANGUAGE-CAPABILITY-MATRIX.md`) form for all 14 registered languages, self-consistency-tested against each other and against the actual golden-parity test coverage, with the cross-language regression closeout confirming `go test -race ./...` and the pinned Go golden-parity fixture are unregressed by Waves A-G's generalization.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments

- `internal/indexer/capability/matrix.go`: a `CapabilityEntry{Extraction, Resolution, Dispatch, Routing Coverage; Gaps []string}` struct, a package-level `matrix map[string]CapabilityEntry` keyed by language ID, and `Lookup`/`All` accessors (`All` returns a defensive deep copy). Populated from the ACTUAL coverage each of 05-04..05-12's own SUMMARYs reported — not aspiration.
- `docs/LANGUAGE-CAPABILITY-MATRIX.md`: a human-readable table (14 rows × 4 capability columns) plus per-language gap bullets and a "Shared Caveats" section (external-scanner crash risk, C# partial-class node identity, `.h` C/C++ disposition, Swift/Kotlin `[SUS]` grammar provenance, the cross-language `resolve.go` `modulePath` imports-edge limitation, route-detection-on-full-index-only scope boundary).
- `indexer.RegisteredLanguageIDs()`: a new exported accessor on the `LanguageSpec` registry, sorted ascending — the matrix's own consistency test's source of truth for "which languages must be covered."
- `matrix_test.go`: seven tests — registry coverage (no missing/no extra), cell validity (every non-full cell names a gap), doc/descriptor mirroring (coverage values AND gap text, byte-for-byte), the D-12 phase gate (every full priority-4 Resolution/Dispatch entry maps to a real, parsed-via-go/parser golden-parity test function; every partial entry names ≥1 gap), plus `Lookup`/`All` unit tests including a mutation-isolation proof for `All()`'s defensive copy.
- Cross-language regression closeout: `go test -race ./... -count=1` green across all 26 packages (including every new mainstream/priority-4 extractor package); `go test -race ./testdata/golden/... -count=1` green — `TestGoldenParity` (Go's own pinned `weft` corpus fixture) passes all 7 subtests, confirming Wave A's generalization from Go-only to 14 languages did NOT regress Go's own golden-parity behavior; `TestGoldenParity_Java`/`_CSharp`/`_Python`/`_TSJS` self-skip cleanly (no corpus configured in this environment, the same documented D-12 fallback every prior priority-4 plan recorded); `TestDeterministicRebuild` and the route-specific `TestRoute_DeterministicRebuild` both pass, confirming the determinism gate holds across the generalized multi-language pipeline.

## Coverage Headline

| Language | Extraction | Resolution | Dispatch | Routing |
|---|---|---|---|---|
| go | full | full | full | full |
| java | full | full | full | full |
| csharp | full | full | full | full |
| python | full | full | none | full |
| typescript | full | full | full | full |
| tsx | full | full | full | full |
| javascript | full | full | none | full |
| rust | full | partial | partial | none |
| ruby | full | partial | none | none |
| php | full | partial | partial | none |
| c | full | partial | none | none |
| cpp | full | partial | none | none |
| swift | full | partial | partial | none |
| kotlin | full | partial | partial | none |

Full table + every named gap: `docs/LANGUAGE-CAPABILITY-MATRIX.md`.

## Task Commits

1. **Task 1 + Task 2 (combined): machine-readable capability descriptor + human-readable matrix + D-11/D-12 consistency check** - `a8a1f6d` (feat)

Task 2's specific deliverable (the matrix↔golden-parity-test/gap-naming consistency check) was authored together with Task 1's matrix.go/doc in the same commit — see Deviations below. Task 2's remaining work (the cross-language regression closeout: running `go test -race ./...` and `testdata/golden/...`) is a verification-only step with no code diff to commit; both suites are green as recorded in this SUMMARY's `coverage` block.

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified

- `internal/indexer/capability/matrix.go` - `CapabilityEntry`, `Coverage` enum, the 14-language `matrix`, `Lookup`/`All`
- `internal/indexer/capability/matrix_test.go` - registry-coverage, cell-validity, doc-mirroring, D-12 phase-gate, `Lookup`/`All` tests
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` - human-readable mirror + per-language gaps + Shared Caveats
- `internal/indexer/languages.go` - added `RegisteredLanguageIDs() []string` (sorted, read-only registry accessor)

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:

- Python and JavaScript's Dispatch is honestly `none` (not `full`) — neither extractor ever emits an interface-shaped node, so RES-02's declared-implements promotion structurally never fires for either language. This is a deliberate departure from a literal reading of the plan's "priority-4 = full extraction+resolution+dispatch" framing, in favor of the plan's own overriding instruction to consolidate the ACTUAL coverage from each language's own SUMMARY and not overstate.
- Mainstream-6 Dispatch is `partial` for the four languages whose extractors emit BOTH an interface-shaped node and a supertype embeds ref (Rust/PHP/Swift/Kotlin) and `none` for the three that emit neither (Ruby/C/C++) — a finer-grained signal than uniformly marking all six `partial` or all six `none`.
- The pre-existing `resolve.go` `isIntraModule`/`modulePath` "imports"-edge limitation (affects every non-Go language identically) is documented as a Shared Caveat, not folded into each language's per-row Resolution score — it doesn't affect the calls/embeds resolution every golden-parity/self-consistency test actually measures, so downgrading Resolution for it would be a less accurate signal.

## Deviations from Plan

### Auto-fixed Issues

None — Rule 1/2/3 auto-fixes were not needed.

### Scope Clarifications (not deviations, but worth recording)

**1. [Rule 4 boundary, resolved without a checkpoint] Task 2's consistency tests were combined into Task 1's single commit**

- **Found during:** Task 1 design, before writing matrix_test.go
- **Issue:** The plan splits Task 1 ("create the matrix + doc") from Task 2 ("add the D-12 consistency check on top of the matrix"), implying two separate commits touching the same file (`matrix_test.go`). Designing the full test suite required understanding the D-12 phase gate's exact shape (golden-test-function existence, gap-naming enforcement) BEFORE the matrix's own coverage values could be finalized responsibly — e.g. discovering during design that Python/JavaScript's Dispatch axis needed to be `none` (not the plan's literal "full") only became clear while writing the consistency-check logic and cross-referencing 05-06/05-07/05-09's SUMMARYs together.
- **Resolution:** Authored matrix.go, the doc, and the FULL matrix_test.go (both the Task-1-shaped structural tests and the Task-2-shaped D-12 phase-gate tests) together, verified as one coherent whole (`go test ./internal/indexer/capability/... -v` — all 7 tests pass), and committed once. This avoided landing an intermediate commit whose matrix.go entries would have needed revision one commit later once the phase-gate logic exposed the Dispatch-axis nuance above — a worse outcome (a commit that briefly overstated coverage) than combining the two tasks' file-identical deliverable into one atomic, internally-consistent commit.
- **Files affected:** internal/indexer/capability/matrix.go, matrix_test.go, docs/LANGUAGE-CAPABILITY-MATRIX.md
- **Verification:** `go test ./internal/indexer/capability/... -v -count=1` — all 7 tests pass on the single commit; `go build ./...` green.

---

**Total deviations:** 0 auto-fixed. 1 scoped process clarification (documented, not a checkpoint).
**Impact on plan:** No scope creep — both tasks' full deliverable set was shipped; only the task/commit boundary shifted, and only because splitting it would have produced a less honest intermediate state.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 5 (Language Coverage & Resolution Breadth) is now complete: 14 languages registered, priority-4 validated full across extraction/resolution (Java/C#/Python/TS-JS), dispatch synthesis (RES-02/RES-03) live, framework routing (LANG-07) covering the five priority frameworks, and mainstream-6 shipped at accurately-documented-partial coverage with every gap named in both the human- and machine-readable D-11 matrix.
- `internal/indexer/capability.Lookup`/`All` are available for any future consumer (CLI `codegraph status`, an MCP tool, or a future `explore` enrichment) that wants to surface per-language coverage to an agent or user without re-deriving it from scratch.
- `go build ./...`, `go vet ./...`, `go test -race ./... -count=1`, and `go test -race ./testdata/golden/... -count=1` all pass — the full multi-language suite is green with no regression to Go's golden parity or the project-wide determinism gate.

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk (internal/indexer/capability/matrix.go, matrix_test.go, docs/LANGUAGE-CAPABILITY-MATRIX.md, internal/indexer/languages.go); commit a8a1f6d confirmed in git log.
