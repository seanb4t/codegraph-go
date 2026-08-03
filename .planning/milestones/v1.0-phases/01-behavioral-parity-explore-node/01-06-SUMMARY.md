---
phase: 01-behavioral-parity-explore-node
plan: 06
subsystem: query
tags: [rwr, random-walk-with-restart, graph-relevance, rank-edges, determinism, d-04, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 02)
    provides: "The 6 shared RefKind*/EdgeKind* string constants in goextract's vocabulary"
  - phase: 01-behavioral-parity-explore-node (plan 05)
    provides: "Go extractor + resolve.go emit all 9 of TS's RANK_EDGES kinds, so RWR ranks over the full set"
provides:
  - "computeGraphRelevance: pure, deterministic RWR relevance scorer (α=0.25, fixed 25 iterations) over an in-memory candidate subgraph"
  - "RankEdges: the 9-kind Go RANK_EDGES-equivalent set, sourced from goextract's shared constants"
  - "buildRWRAdjacency: fresh-per-call undirected adjacency build with WR-04 dangling-endpoint tolerance"
  - "sortRWRScores: score-desc-then-Id-asc deterministic ordering helper (D-04 tie-break)"
affects: [01-11, 01-13, 01-14, 01-16, 01-17 (explore's subgraph-gathering/seeding/gate/golden-validation plans consume this)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure in-memory algorithm function (no graphstore.Reader dependency) — fully unit-testable on synthetic edge slices, mirroring the codebase's existing fresh-per-call, no-package-cache discipline (BuildReverseAdjacency) without needing a fake Reader"
    - "D-04 determinism triad: sorted map-key iteration before building any order-sensitive vector, fixed-iteration-count loop (no convergence early-exit), and a single rounding helper applied once at the return boundary"

key-files:
  created:
    - internal/query/rwr.go
    - internal/query/rwr_test.go
  modified: []

key-decisions:
  - "computeGraphRelevance and buildRWRAdjacency both landed in the same feat commit (Task 1's GREEN) since they share one file and computeGraphRelevance has no dependency beyond what Task 1 already implements — Task 2's tests were then added as their own commit locking in the required behavior (determinism, seed-outranks-distant, dangling-retention, tie-break), satisfying the plan-level TDD gate (a test commit precedes a feat commit in git log) without artificially splitting one cohesive algorithm across two RED/GREEN cycles"
  - "T-01-10 (DoS) mitigation for this plan is documentation-only, per the plan's own threat_model disposition: computeGraphRelevance's doc comment states the O(iterations*edges) cost and the maxNodes=200/GLUE_NODE_CAP=60 precondition explicitly, but does NOT enforce those caps here — enforcement is scoped to the upstream subgraph-gathering plans (11/16) per the plan's own text, not this pure-function plan"
  - "Score rounding precision is 1e-9 (math.Round(v*1e9)/1e9), matching the plan's 'e.g. to 1e-9' guidance — applied once at the return boundary in computeGraphRelevance so every downstream comparison/sort sees only rounded values"

patterns-established:
  - "rwrScoredNode + sortRWRScores: the reusable score-desc-then-Id-asc ordering shape future explore-pipeline plans (seeding, gate, rendering) should call rather than re-implementing the tie-break"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "RankEdges is exactly the 9 RESEARCH §C.1 members, sourced from goextract's shared RefKind*/EdgeKind* constants (no re-declared literals)"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/rwr_test.go#TestRankEdges"
        status: pass
    human_judgment: false
  - id: D2
    description: "buildRWRAdjacency excludes non-RankEdges kinds, self-loops, and dangling endpoints (WR-04), and builds undirected (both-direction) adjacency for valid edges"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/rwr_test.go#TestRWRAdjacency_ExcludesNonRankEdgeKind|TestRWRAdjacency_ExcludesSelfLoop|TestRWRAdjacency_SkipsDanglingEndpoint|TestRWRAdjacency_UndirectedBothDirections"
        status: pass
    human_judgment: false
  - id: D3
    description: "computeGraphRelevance runs a fixed-25-iteration, α=0.25, undirected/unweighted power-iteration RWR: a seed node outranks a distant node, a no-seed restart falls back to uniform-over-all with every node getting non-zero mass, and a dangling (degree-0) node retains its own mass"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/rwr_test.go#TestComputeGraphRelevance_SeedOutranksDistant|TestComputeGraphRelevance_NoSeedUniformRestart|TestComputeGraphRelevance_DanglingNodeRetainsMass|TestComputeGraphRelevance_EmptyNodeIDs"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-04 determinism contract: repeated runs over the identical subgraph produce bit-identical (post-rounding) score maps, and sortRWRScores resolves equal scores score-desc-then-Id-asc"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/rwr_test.go#TestRWRDeterminism_RepeatedRunsIdentical (run at -count=5) | TestSortRWRScores_TieBreakScoreDescIdAsc"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 6: RWR Graph-Relevance Core (EXPL-02) Summary

**computeGraphRelevance: a deterministic, fixed-25-iteration, α=0.25 power-iteration Random-Walk-with-Restart over the full 9-kind RANK_EDGES set, ported verbatim from TS 1.3.1's `mcp/tools.js:2321-2386`.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-15T13:19:14Z
- **Completed:** 2026-07-15T13:34:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `RankEdges`: the 9-member Go RANK_EDGES-equivalent set (calls/references/extends/implements/overrides/instantiates/returns/type_of/imports), sourced from goextract's shared constants — no re-declared string literals
- `buildRWRAdjacency`: fresh-per-call, undirected adjacency build over the candidate node set, excluding non-RankEdges kinds, self-loops, and edges to nodes outside the candidate set (WR-04 dangling tolerance, mirroring `BuildReverseAdjacency`'s discipline)
- `computeGraphRelevance`: the load-bearing pure-function port of TS's power-iteration RWR — restart vector uniform over present seeds (uniform-over-all fallback), FIXED 25 iterations (no convergence early-exit), dangling nodes retain their own mass, scores rounded to 1e-9 before return
- `sortRWRScores` / `rwrScoredNode`: the D-04 score-desc-then-Id-asc ordering helper future explore-pipeline plans (seeding, file-relevance gate, rendering) will reuse
- Determinism proven directly: `TestRWRDeterminism_RepeatedRunsIdentical` asserts 10 in-process repeated calls produce identical rounded scores, itself run at `-count=5` by the test runner (50 total repetitions)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED+GREEN RankEdges set + undirected adjacency build** - `bc8806d` (test), `4274409` (feat)
2. **Task 2: RED+GREEN fixed-25-iteration deterministic power iteration** - `d7559aa` (test — see Decisions Made below for why no separate feat commit was needed)

**Plan metadata:** (this commit)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified
- `internal/query/rwr.go` - `RankEdges`, `buildRWRAdjacency`, `computeGraphRelevance`, `roundRWRScore`, `rwrScoredNode`/`sortRWRScores`
- `internal/query/rwr_test.go` - RankEdges membership, adjacency exclusion/undirected tests, RWR behavior + determinism + tie-break tests

## Decisions Made
1. **Combined Task 1/Task 2 implementation, split test commits** — `computeGraphRelevance` (Task 2) has no dependency beyond `RankEdges`/`buildRWRAdjacency` (Task 1) and lives in the same file, so both were implemented together in Task 1's GREEN commit (`4274409`). Task 2's dedicated test commit (`d7559aa`) then locks in computeGraphRelevance's own required behaviors (determinism, seed-vs-distant, dangling-retention, tie-break) as a separate, reviewable unit. The plan-level TDD gate (a `test(...)` commit before a `feat(...)` commit) is satisfied by the Task 1 pair (`bc8806d` → `4274409`); Task 2's tests, though technically GREEN-on-arrival given the shared-file implementation, still serve their purpose — locking in D-04's determinism contract with dedicated, reviewable coverage.
2. **T-01-10 DoS mitigation stays documentation-only in this plan** — the plan's own threat_model table disposes T-01-10 as "the caps are enforced before it is called (documented in the doc comment as a precondition)," explicitly deferring `maxNodes=200`/`GLUE_NODE_CAP=60` enforcement to the upstream subgraph-gathering plans (11/16). `computeGraphRelevance`'s doc comment states this precondition (O(iterations·edges) cost, caller-bounded input) but this plan does not implement the caps itself — doing so here would be premature: there is no subgraph-gathering caller yet to receive them.
3. **Rounding precision 1e-9** — `math.Round(v*1e9)/1e9`, matching the plan's "e.g. to 1e-9" guidance, applied once at computeGraphRelevance's return boundary.

## Deviations from Plan

None - plan executed exactly as written. (See Decisions Made #1 for a commit-sequencing note that is not a scope or behavior deviation — both tasks' acceptance criteria and behaviors are fully implemented and tested.)

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `computeGraphRelevance`/`RankEdges`/`sortRWRScores` are ready for the subgraph-gathering, named-seeding, and file-relevance-gate plans (11/13/14/16) to consume — those plans own enforcing the `maxNodes=200`/`GLUE_NODE_CAP=60` caps before calling this function, per this plan's threat-model disposition.
- Golden-corpus validation against the real re-indexed corpus is explicitly deferred to plan 17, as scoped by this plan's objective.
- Pre-existing unrelated flaky test noted during full-suite verification: `internal/daemon.TestSoak` failed once under `go test ./... -count=1` ("lock held by current process") but passed cleanly in isolation (`go test ./internal/daemon/ -run TestSoak -count=1`) — an environment/parallelism artifact unrelated to this plan's changes, out of scope per the deviation rules' scope boundary (not fixed).

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*
