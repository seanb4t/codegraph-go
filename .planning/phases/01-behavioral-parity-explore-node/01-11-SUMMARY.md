---
phase: 01-behavioral-parity-explore-node
plan: 11
subsystem: query
tags: [explore, subgraph-construction, type-hierarchy, bfs, glue-nodes, dos-mitigation, d-10, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 03)
    provides: "Tokenizers feeding explore's query -> symbols/terms pipeline that upstream gather channels (H3-H6, plan 07) consume"
  - phase: 01-behavioral-parity-explore-node (plan 06)
    provides: "computeGraphRelevance (RWR) + RankEdges — the ranker this plan's bounded subgraph feeds"
  - phase: 01-behavioral-parity-explore-node (plan 07)
    provides: "gatherCandidate shape + sortGatherCandidates (D-04 score-desc-then-Id-asc tie-break) that expandBFS's root pruning/trimming reuses"
provides:
  - "expandTypeHierarchy (H10): 2-pass extends/implements ancestor+descendant+sibling expansion, bounded to ceil(maxNodes/4)"
  - "expandBFS (H11): candidate-pruning + bounded BFS subgraph construction (maxNodes=200/traversalDepth=3/minScore=0.2/searchLimit=8)"
  - "expandGlueNodes (H12): same-file-only caller/callee injection capped at GLUE_NODE_CAP=60"
  - "subgraphFileSet: node-id-list -> distinct FilePath set helper H12 (and future wire-time callers) consume"
affects: [01-13, 01-14, 01-16, 01-17 (seeding/gate/wire-time/golden-validation plans consume these three primitives)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure graphstore.Reader-driven primitives (no package-level cache, fresh IterateEdges/IterateNodes scan per call) — mirrors BuildReverseAdjacency's/BuildImplementsIndex's/gather.go's fresh-per-call discipline throughout"
    - "Sorted-frontier BFS determinism: every BFS level (transitiveWalk, expandBFS) sorts its frontier/neighbor set before admission, so a cap binding mid-traversal always keeps the identical subset across repeated runs"
    - "WR-04 dangling-tolerance extended to subgraph construction: expandBFS validates each BFS-discovered neighbor via GetNode before admitting it (a dangling edge target is skipped, not an error), matching Impact's/Callers'/Callees' existing convention"

key-files:
  created:
    - internal/query/expand.go
    - internal/query/expand_test.go
  modified: []

key-decisions:
  - "expandHierarchyKinds reuses gather.go's definitionKinds var rather than re-declaring an identical map — H10's TS wording (class/interface/struct/trait/protocol) and H4's TS wording (...+enum/type_alias) both collapse onto the identical 3 Go Kind values (KindStruct/KindInterface/KindTypeAlias) under this codebase's existing D-02 consolidation, so a second map would be a duplicate, not a distinct filter"
  - "H11's BFS traverses along RankEdges (rwr.go's 9-kind set, undirected) rather than a narrower or wider edge set — this makes the subgraph expandBFS returns exactly the edge universe computeGraphRelevance itself walks (buildExpandAdjacency's RankEdges filter mirrors buildRWRAdjacency's), so RWR never silently ranks over edges the subgraph never surfaced or vice versa"
  - "No verbatim TS source exists for H10/H11/H12 in the frozen RESEARCH capture (unlike H1-H9's Code Examples §1-9/§11) — only the constants + one-line rule descriptions in RESEARCH §C.2's table survive, since the live TS dist is no longer readable on this machine (per gather.go's package doc comment). Every constant (ceil(maxNodes/4), maxNodes=200/traversalDepth=3/minScore=0.2/searchLimit=8, GLUE_NODE_CAP=60) is ported verbatim from that table; the specific traversal/admission mechanics (which edge kinds BFS walks, tie-break order when a cap binds) are this plan's own documented design filling that gap, using the codebase's existing D-04 sorted-Id determinism convention throughout rather than inventing a new one"
  - "GLUE_NODE_CAP's tie-break when the cap binds uses sorted-Id order (lowest-Id-first kept) — RESEARCH §C.2/H12's own admission order (whatever TS's per-root iteration produced) is not recoverable from the frozen capture, so D-04's existing lowest-Id-first convention (resolveSymbolNode, sortRWRScores) is reused rather than inventing score- or discovery-order-based tie-breaking with no source to validate against"
  - "expandBFS's minScore=0.2 prunes CANDIDATE roots (gather.go's gatherCandidate.Score) before traversal, not BFS-discovered nodes — BFS-discovered nodes carry no gather-channel score to prune by, so H11's 'prunes candidates with score < 0.2' rule is implemented as a pre-traversal seed filter, consistent with the plan's own <behavior> spec phrasing ('prunes candidates', not 'prunes discovered nodes')"

patterns-established:
  - "ExpandBFSBounds / DefaultExploreBFSBounds: the reusable bounds-struct shape future wire-time (plan 16) and any other future BFS-bounded subgraph caller should pass, rather than four positional args"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "H10 type-hierarchy expansion: a focal type node's extends/implements ancestors AND descendants are added (Pass 1), then newly-found parents' other direct children are added (Pass 2 siblings), bounded to ceil(maxNodes/4); a non-type-kind focal id is ignored"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/expand_test.go#TestTypeHierarchyExpansion_AncestorsAndDescendants|TestTypeHierarchyExpansion_SecondPassSiblings|TestTypeHierarchyExpansion_BudgetCap|TestTypeHierarchyExpansion_NonTypeFocalIgnored"
        status: pass
    human_judgment: false
  - id: D2
    description: "H11 BFS traversal bounds: maxNodes=200/traversalDepth=3/minScore=0.2/searchLimit=8 all enforced — depth-bounded expansion, total-node cap, low-score root pruning, and root-count trim to searchLimit (highest score first)"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/expand_test.go#TestBFSBounds_DepthLimit|TestBFSBounds_NodeCap|TestBFSBounds_MinScorePrune|TestBFSBounds_SearchLimit"
        status: pass
    human_judgment: false
  - id: D3
    description: "H12 glue-node injection: a root's caller/callee is injected only if its file is already surfaced by the subgraph, never a root re-injecting itself, capped at GLUE_NODE_CAP=60 with deterministic sorted-Id admission when the cap binds"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/expand_test.go#TestGlueNodeInjection_SameFileOnly|TestGlueNodeCap_Binds|TestGlueNodeInjection_RootNeverInjectedAsOwnGlue|TestGlueNodeInjection_SubgraphFileSetHelper"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 11: Subgraph-Construction Heuristics H10-H12 (EXPL-02) Summary

**expandTypeHierarchy/expandBFS/expandGlueNodes: the three DoS-bounded subgraph-construction primitives (2-pass extends/implements expansion, bounded BFS, same-file glue-node injection) that turn explore's gathered candidates into the maxNodes=200/GLUE_NODE_CAP=60-capped subgraph RWR ranks over.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-15T13:50:41Z
- **Completed:** 2026-07-15T13:53:32Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `expandTypeHierarchy` (H10): 2-pass type-hierarchy expansion — Pass 1 walks the FULL transitive extends/implements ancestor chain and descendant tree of each focal type-kind node; Pass 2 adds every newly-found ancestor's OTHER direct children (siblings the focal chain didn't already reach) — bounded to `ceil(maxNodes/4)` newly-added node ids, verbatim per RESEARCH §C.2/H10
- `expandBFS` (H11): prunes candidate roots scoring below `minScore=0.2`, trims survivors to `searchLimit=8` (highest score first, D-04 tie-break), then BFS-expands along the same 9-kind `RankEdges` set RWR itself walks, bounded to `traversalDepth=3` hops and `maxNodes=200` total nodes — WR-04 dangling-target skip mirrors Impact's BFS discipline
- `expandGlueNodes` (H12): injects a root's callers/callees ONLY when they live in a file the subgraph already surfaces, capped at `GLUE_NODE_CAP=60`, deterministic sorted-Id admission when the cap binds; a root is never re-injected as its own glue node
- `subgraphFileSet`: the node-id-list → distinct `FilePath` set helper H12 (and future wire-time callers, plan 16) use to build the "files already surfaced" input H12's same-file-only constraint checks against
- 12 unit tests exercising every cited constant (ceil(200/4)=50 budget, depth/node/score/searchLimit bounds, GLUE_NODE_CAP=60) plus edge cases (non-type focal ignored, root never self-injects, dangling-id tolerance)

## Task Commits

Each task was committed atomically (RED then GREEN):

1. **Task 1: RED+GREEN type-hierarchy expansion (H10) + BFS bounds (H11)** - `686df8b` (test), `a9da595` (feat)
2. **Task 2: RED+GREEN glue-node injection (H12) with GLUE_NODE_CAP** - `23ea747` (test), `022b2dd` (feat)

**Plan metadata:** (this commit)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified
- `internal/query/expand.go` - `expandTypeHierarchy`, `expandHierarchyBudget`, `buildTypeHierarchyIndex`, `transitiveWalk` (H10); `ExpandBFSBounds`, `DefaultExploreBFSBounds`, `buildExpandAdjacency`, `expandBFS` (H11); `subgraphFileSet`, `calleeCallEdges`, `admitGlueCandidate`, `expandGlueNodes` (H12); `ExpandMaxNodes`/`ExpandTraversalDepth`/`ExpandMinScore`/`ExpandSearchLimit`/`GlueNodeCap` constants
- `internal/query/expand_test.go` - `expandFakeReader` synthetic in-memory `graphstore.Reader` + 12 tests covering H10/H11/H12 behaviors and bound constants

## Decisions Made
1. **`expandHierarchyKinds` reuses `gather.go`'s `definitionKinds`** rather than re-declaring a near-identical map — see key-decisions above.
2. **H11's BFS traverses `RankEdges`** (the same 9-kind set `computeGraphRelevance` walks) — see key-decisions above; keeps the subgraph and the ranker's edge universe in lockstep.
3. **No verbatim TS source for H10/H11/H12** exists in the frozen RESEARCH capture — every cited constant is ported verbatim, but the traversal/admission mechanics are this plan's own documented design using the codebase's existing D-04 determinism conventions. See key-decisions for full reasoning.
4. **`GLUE_NODE_CAP` tie-break = sorted-Id order** (lowest-Id-first kept) — no recoverable TS admission order to match, so D-04's existing convention is reused.
5. **`minScore=0.2` prunes candidate ROOTS pre-traversal**, not BFS-discovered nodes (which carry no gather-channel score to prune by) — matches the plan's own `<behavior>` phrasing ("prunes candidates").

## Deviations from Plan

None - plan executed exactly as written. All three heuristics (H10/H11/H12) and every cited constant (`ceil(maxNodes/4)`, `maxNodes=200`, `traversalDepth=3`, `minScore=0.2`, `searchLimit=8`, `GLUE_NODE_CAP=60`) are implemented and tested per the plan's `must_haves.truths` and `threat_model` T-01-18 disposition.

## Issues Encountered
None. Full repo test suite (`go test ./... -count=1`) passes after both tasks; `go vet ./...` and `gofmt -l` clean.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `expandTypeHierarchy`/`expandBFS`/`expandGlueNodes` are ready for plans 13 (named-symbol seeding), 14 (relevance gate), and 16 (Explore() wire-time composition) to consume — those plans own composing gather.go's candidates → this plan's bounded subgraph → rwr.go's `computeGraphRelevance`, in that pipeline order.
- T-01-18 (DoS via unbounded subgraph growth) is now FULLY mitigated at this stage: `maxNodes=200`, `traversalDepth=3`, `minScore=0.2`, the `ceil(maxNodes/4)` hierarchy budget, and `GLUE_NODE_CAP=60` are all enforced here, closing the gap plan 06's RWR core deliberately left open (01-06-SUMMARY.md Decision 2).
- Golden-corpus validation against the real re-indexed corpus remains deferred to plan 17, as scoped by the phase's overall objective.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: internal/query/expand.go
- FOUND: internal/query/expand_test.go
- FOUND: .planning/phases/01-behavioral-parity-explore-node/01-11-SUMMARY.md
- FOUND commit: 686df8b (test H10+H11)
- FOUND commit: a9da595 (feat H10+H11)
- FOUND commit: 23ea747 (test H12)
- FOUND commit: 022b2dd (feat H12)
