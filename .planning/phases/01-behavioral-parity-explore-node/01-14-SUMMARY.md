---
phase: 01-behavioral-parity-explore-node
plan: 14
subsystem: query
tags: [explore, relevance-gate, file-sort, central-file, rwr, determinism, d-04, d-08, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 06)
    provides: "computeGraphRelevance's per-node RWR mass (rwr.go) — the ultimate source of fileGraphScore this plan gates/sorts on"
  - phase: 01-behavioral-parity-explore-node (plan 07)
    provides: "the shared isTestFile path predicate (gather.go) reused (via scoring.go's isLowValueFile) by the sort's !low-value tier"
  - phase: 01-behavioral-parity-explore-node (plan 13)
    provides: "computeFileScoreTiers/aggregateFileGraphScore/applyBuriedRescue (scoring.go) — the fileScores/fileGraphScore/rescued maps this plan's gate and sort consume directly"
provides:
  - "fileRelevanceGate: the EXPL-03 5-way OR (graph-mass>=6% of max, central, entry/named-file score tier, change-surface-rescued, >=2 distinct term hits), guarded to never prune below 2 files and only apply when maxGraph>0"
  - "centralFileSelection: H19's 1-2 highest-graph-mass files with >=1 term hit — feeds the gate's clause (2) and the larger whole-file render ceiling downstream"
  - "fiveTierFileSort: H18's named-seed > corroborated > graph-mass(1%-epsilon) > term-hits > !low-value tiers, then !generated/score/node-count/path deterministic tail"
affects: [01-16 (the ranked file list this plan produces feeds explore's render stage), 01-17 (golden-corpus validation exercises this gate/sort directly)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure map-based functions over caller-supplied path slices — no graphstore.Reader dependency, mirroring rwr.go's fully-synthetic-testable shape rather than scoring.go's Reader-driven one, since the gate/sort operate purely on plan 13's already-aggregated per-file maps"
    - "Constant reuse across heuristics sharing a cited value (fileRelevanceGateMassFraction = buriedRescueMassFraction) instead of re-declaring the same 0.06 literal — documents that H16 and H17 share one constant per RESEARCH's adjacent source citations"

key-files:
  created:
    - internal/query/explore_gate.go
    - internal/query/explore_gate_test.go
  modified: []

key-decisions:
  - "H17's clause-(3) 'entry/named-file' is derived directly from plan 13's fileScores tier (score >= fileScoreEntry, i.e. entry OR named-seed tier) rather than a separate caller-supplied entryFiles set — RESEARCH's citations describe entryFiles/centralFiles/changeSurfaceFiles as three parallel TS closures, but plan 13's own 'Next Phase Readiness' note names only fileScores/fileGraphScore/rescued as this plan's consumed inputs, not a fourth boolean set; deriving clause 3 from the already-computed H14 tier avoids inventing an unsourced parameter while preserving the clause's independent OR semantics"
  - "Files carry no stable 'Id' (unlike nodes) — the H18 sort's final deterministic tail substitutes ascending file path for the codebase's usual lowest-Id tie-break, documented inline as this plan's D-04 substitute, consistent with node.go's own documented Id-vs-row-order divergence for NODE-01"
  - "Tasks 1 and 2 landed in one feat commit (shared new file, five-tier sort directly reuses the gate's fileScoreEntry/fileScoreNamedSeed/fileRelevanceGateMinTermHits constants and centralFileSelection's output) — mirrors plan 13's own precedent for combining same-file, mutually-referencing tasks into a single GREEN commit after one combined RED test commit covering both tasks' TDD cases"
  - "fileNodeCounts is accepted as a caller-supplied map[string]int (H18's tail-tier input) rather than derived here — no upstream plan yet counts nodes per file; this is the same 'primitives now, wiring later' scoping plan 13 established for fileTermHits, which this plan also still treats as caller-supplied"

patterns-established:
  - "fileRelevanceGateMassFraction/centralFileSelection/fiveTierFileSort as the terminal pure-function stage of the explore pipeline: plan 13's fileScores/fileGraphScore/rescued maps in, a gated+ordered []string out — ready for a later wiring plan to compose with H1-H16's outputs exactly as scoring.go's own README-style doc comment anticipated"

requirements-completed: [EXPL-03, EXPL-02]

coverage:
  - id: D1
    description: "H17 5-way OR relevance gate: each of the 5 clauses (graph-mass>=6%, central, entry/named, change-surface-rescued, >=2 term hits) independently keeps a file; a file failing all 5 is dropped"
    requirement: "EXPL-03"
    verification:
      - kind: unit
        ref: "internal/query/explore_gate_test.go#TestFileRelevanceGate_Clause1GraphMassKeepsFile|TestFileRelevanceGate_Clause2CentralKeepsFile|TestFileRelevanceGate_Clause3EntryNamedKeepsFile|TestFileRelevanceGate_Clause4ChangeSurfaceRescuedKeepsFile|TestFileRelevanceGate_Clause5TermHitsKeepsFile|TestFileRelevanceGate_DropsFileFailingAllClauses"
        status: pass
    human_judgment: false
  - id: D2
    description: "H17 guards: gate never prunes below 2 files, and is skipped entirely when maxGraph==0"
    requirement: "EXPL-03"
    verification:
      - kind: unit
        ref: "internal/query/explore_gate_test.go#TestFileRelevanceGate_NeverPrunesBelowTwoFiles|TestFileRelevanceGate_SkippedWhenMaxGraphZero"
        status: pass
    human_judgment: false
  - id: D3
    description: "H19 central-file selection: the 1-2 highest-graph-mass files that also have >=1 term hit; a higher-mass file with 0 term hits is excluded; result is not padded to 2 when fewer qualify"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/explore_gate_test.go#TestCentralFileSelection_PicksTopTwoByMassWithTermHit|TestCentralFileSelection_ExcludesFilesWithZeroTermHits|TestCentralFileSelection_ReturnsFewerThanTwoWhenNotEnoughQualify"
        status: pass
    human_judgment: false
  - id: D4
    description: "H18 5-tier file sort: named-seed > corroborated > graph-mass(1%-epsilon) > term-hits > !low-value precedence, then !generated/score/node-count/path deterministic tail"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/explore_gate_test.go#TestFiveTierFileSort_NamedSeedFirst|TestFiveTierFileSort_CorroboratedAboveGraphMass|TestFiveTierFileSort_GraphMassEpsilonTie|TestFiveTierFileSort_GraphMassBeyondEpsilonWins|TestFiveTierFileSort_LowValueSortsAfter|TestFiveTierFileSort_GeneratedFileSortsAfterEquivalent|TestFiveTierFileSort_DeterministicTailOnFullTie|TestFiveTierFileSort_UsesSliceStable"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 14: File-Relevance Gate, Central-File Selection & 5-Tier Sort (H17-H19) Summary

**explore_gate.go: the EXPL-03 5-way OR relevance gate — the primary lever stopping weakly-connected Test\* funcs from topping explore results — plus H19's central-file selection and H18's 5-tier file sort, all ported verbatim from RESEARCH §4/§C.2 as pure functions over plan 13's per-file score/mass/rescue maps.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-15T15:14:24Z
- **Completed:** 2026-07-15T15:26:00Z
- **Tasks:** 2
- **Files modified:** 2 (both new)

## Accomplishments
- `fileRelevanceGate` (H17, EXPL-03's core): the 5-way boolean OR — `fileGraphScore[fp] >= maxGraph*6%` OR `centralFiles[fp]` OR `fileScores[fp] >= fileScoreEntry` OR `rescuedFiles[fp]` OR `fileTermHits[fp] >= 2` — each clause independently sufficient, applied only when `maxGraph > 0`, and only replacing the working set when the gated result retains at least 2 files (mirrors `groupMatchesByFile`'s cap-not-truncate discipline, explore.go)
- `centralFileSelection` (H19): among files with >=1 distinct term hit, the top 1-2 by graph mass (deterministic mass-desc/path-asc tie-break) — a file with 0 term hits is excluded outright regardless of mass dominance
- `fiveTierFileSort` (H18): named-seed file first, then corroborated (entry/central AND >=2 terms), then graph mass with a 1%-of-max epsilon tie band, then term hits, then `!low-value` (reusing scoring.go's `isLowValueFile`), then a deterministic tail of `!generated` (reusing node.go's `isGeneratedFile`), score, node count, and finally ascending path
- All three functions pure and map-based (no `graphstore.Reader` dependency), independently unit-testable against synthetic path/score/mass/term-hit fixtures — 19 tests covering every clause, both guards, the central-selection cap/exclusion/under-fill cases, every sort tier, the epsilon boundary (both sides), and the full deterministic tail

## Task Commits

Each task's RED and GREEN steps were committed:

1. **Task 1 + Task 2: H17 gate + H19 central selection + H18 five-tier sort** — RED `cb231ea` (test), GREEN `ed86161` (feat)

_TDD gate sequence verified: `test(01-14)` (cb231ea) precedes `feat(01-14)` (ed86161) in git log._

## Files Created/Modified
- `internal/query/explore_gate.go` - `fileRelevanceGate` (H17) + its constants; `centralFileSelection` (H19) + its constants; `fiveTierFileSort` (H18) + its epsilon constant
- `internal/query/explore_gate_test.go` - 19 tests: 8 for the gate (one per clause + drop case + both guards), 3 for central selection (cap/exclusion/under-fill), 8 for the five-tier sort (each tier, both sides of the epsilon boundary, generated-file tail, full deterministic-tail cascade, order-independence)

## Decisions Made
1. **H17's entry/named-file clause (3) derives from plan 13's `fileScores` tier** (`>= fileScoreEntry`) rather than a separate caller-supplied `entryFiles` set — plan 13's own "Next Phase Readiness" note names only `fileScores`/`fileGraphScore`/`rescued` as this plan's inputs, not a fourth boolean set; reusing the already-computed H14 tier avoids inventing an unsourced parameter.
2. **Path is the H18 sort's deterministic tail**, not an Id — files have no stable Id field in this codebase (unlike nodes); ascending path is the natural, documented substitute, consistent with `node.go`'s own precedent of documenting where TS's implicit ordering has no Go equivalent.
3. **Tasks 1/2 combined into one feat commit** — the five-tier sort directly reuses the gate's `fileScoreEntry`/`fileScoreNamedSeed`/`fileRelevanceGateMinTermHits` constants and would need `centralFileSelection`'s output wired the same way as the gate's clause (2); splitting them into two files/commits would have meant either duplicating constants or introducing a premature cross-file dependency. Mirrors plan 13's own documented precedent for this exact situation.
4. **`fileNodeCounts` is caller-supplied**, not derived in this file — no upstream plan yet counts nodes per file; consistent with the "primitives now, wiring later" scoping plan 13 already established for `fileTermHits`.

## Deviations from Plan

None - plan executed exactly as written. The combined-commit and entry-clause-derivation choices above are documented D-02 design decisions within the plan's own explicit allowance for heuristics whose exact upstream wiring RESEARCH's frozen citations pin only at the constant/structure level, not scope or behavior deviations from the plan's stated tasks/acceptance criteria.

## Issues Encountered
None. `go build ./...`, `go vet ./...`, and `golangci-lint run ./internal/query/...` all clean on the new files (only pre-existing errcheck findings in untouched files, and one staticcheck empty-branch finding in this plan's own first test draft — fixed before the GREEN commit). Full-repo `go test ./... -count=1` shows only the pre-existing, previously-documented flaky `internal/daemon.TestSoak` (unrelated to this plan — see 01-06-SUMMARY.md/01-13-SUMMARY.md's prior notes; out of scope per the deviation rules' scope boundary, not fixed here). `go test ./internal/query/... -count=1` is fully green.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `fileRelevanceGate`, `centralFileSelection`, and `fiveTierFileSort` are ready for a later wiring plan (per this phase's repeated "primitives now, wiring later" precedent) to compose with H1-H16's actual outputs: plan 13's `fileScores`/`fileGraphScore`/`rescued`, `centralFileSelection`'s own output feeding the gate's clause (2), and a still-unwired `fileTermHits`/`fileNodeCounts` pair that a later plan must compute from the gather channels' term-hit data and a per-file node count.
- The ranked, gated `[]string` this plan's `fiveTierFileSort` produces is the direct input to plan 16's explore render stage (per the plan's own `key_links`).
- Plan 17's golden-corpus validation is the first point where this gate/sort's behavior gets checked against real captured TS output — no golden fixture exercises H17-H19 yet.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*
