---
phase: 01-behavioral-parity-explore-node
plan: 13
subsystem: query
tags: [explore, relevance-gate, file-scoring, rwr, rank-edges, determinism, d-04, d-09, d-10, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 06)
    provides: "computeGraphRelevance's per-node RWR mass + RankEdges (rwr.go) — the input this plan aggregates into fileGraphScore"
  - phase: 01-behavioral-parity-explore-node (plan 07)
    provides: "the shared isTestFile path predicate (gather.go) H15's low-value-file exclusion reuses"
provides:
  - "computeFileScoreTiers: H14 per-file score tiers (named-seed +50/entry +10/connected-to-entry +3/other +1), keep threshold score>=3"
  - "aggregateFileGraphScore: RESEARCH §4's fileGraphScore = sum of per-node RWR mass per file"
  - "applyHardTestExclusion: H15 hard test/spec/icon/i18n exclusion with the query-mentions-test+>=2-non-test exemption"
  - "applyBuriedRescue: H16 change-surface buried-rescue over the new D-09 references/type_of/returns signature-type edges, score=max(score,45) force-keep"
affects: [01-14 (the H17 relevance gate consumes fileScores/fileGraphScore/the rescued set as its 5-way OR gate input)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-file score is a MAX rollup over its nodes' tier value (not a sum) — a file with even one named-seed node scores 50 regardless of how many other-tier nodes it also holds"
    - "H14-H16 accept caller-supplied node-id sets (namedSeedIDs/entryIDs/tierSeedIDs) and maps (fileTermHits) rather than deriving them internally — mirrors expand.go's/seeding.go's own 'primitives now, wiring later' discipline; the actual wiring is a later plan's job"
    - "In-place map mutation for file-score pipelines (applyHardTestExclusion deletes keys, applyBuriedRescue adds/raises keys) — same pattern as gather.go's H7-H9 post-merge rerankers"

key-files:
  created:
    - internal/query/scoring.go
    - internal/query/scoring_test.go
  modified: []

key-decisions:
  - "Per-file H14 score is the MAX tier value among a file's nodes, not a sum — a documented D-02 design choice since RESEARCH's frozen citation pins the four tier constants and the >=3 keep threshold but not the exact per-file rollup arithmetic; MAX is the simplest reading consistent with 'keep files with score>=3' cleanly excluding only pure-other-tier files"
  - "namedSeedIDs/entryIDs (H14) and tierSeedIDs/fileTermHits (H16) are accepted as caller-supplied parameters, not derived inside scoring.go — RESEARCH's citations pin H14-H16's constants/rule structure but not the exact upstream wiring of which node ids count as 'entry' or how fileTermHits is computed; that wiring is scoped to a later plan (mirrors expand.go's own H10-H12 'primitives now, wiring later' precedent)"
  - "isIconOrI18nFile (H15's icon/i18n low-value component) is a documented D-02 substitute: RESEARCH's H15 row cites the RULE ('test/spec/icon/i18n') but no further source detail on the icon/i18n predicate's exact patterns survives the frozen capture (the live TS dist JS is unreadable on this machine, per gather.go's package doc comment) — implemented as a conservative path-based heuristic (icon/i18n/locale/translation directory segments + filename prefixes) mirroring isTestFile's own shape"
  - "queryMentionsTest (H15) is a separate function from gather.go's queryMentionsTestOrSpec (H7) — RESEARCH's frozen citations give the two heuristics DIFFERENT query-substring wording ('mentions test' for H15 vs 'mentions test/spec' for H7), so they are not the same check despite the surface similarity"
  - "H16's applyBuriedRescue uses Reader.IterateEdges(srcID) to walk a tier-seed callable's signature-type edges directly (mirroring expand.go's calleeCallEdges), rather than filtering a pre-built subgraph edge list — buried-rescue's entire purpose is to reach files OUTSIDE the bounded subgraph, so it cannot be limited to edges the subgraph traversal already surfaced"

patterns-established:
  - "Reader-driven pure functions accepting caller-supplied entry/seed sets as the seam between 'this plan's algorithm' and 'a later plan's wiring' — the third plan in this phase (after expand.go/seeding.go) to use this shape for a heuristic whose exact upstream inputs are pinned only at the constant/structure level by RESEARCH"

requirements-completed: [EXPL-02, EXPL-03]

coverage:
  - id: D1
    description: "H14 per-file score tiers: named-seed +50, entry +10, connected-to-entry +3, other +1, per-file MAX rollup, keep threshold score>=3"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/scoring_test.go#TestFileScoreTiers_NamedSeedFileScores50|TestFileScoreTiers_EntryFileScores10|TestFileScoreTiers_ConnectedToEntryScores3|TestFileScoreTiers_OtherFileDroppedBelowThreshold|TestFileScoreTiers_FileRollsUpToHighestNodeTier"
        status: pass
    human_judgment: false
  - id: D2
    description: "fileGraphScore aggregation: sum of per-node RWR mass per file (RESEARCH §4), WR-04 dangling-node tolerance"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/scoring_test.go#TestFileGraphScoreAgg_SumsMassPerFile|TestFileGraphScoreAgg_SkipsDanglingNode"
        status: pass
    human_judgment: false
  - id: D3
    description: "H15 hard test/spec/icon/i18n exclusion, with the query-mentions-test AND >=2-non-test-remain exemption (all-or-nothing, not partial)"
    requirement: "EXPL-03"
    verification:
      - kind: unit
        ref: "internal/query/scoring_test.go#TestHardTestExclusion_DropsTestFileByDefault|TestHardTestExclusion_DropsIconI18nFileByDefault|TestHardTestExclusion_ExemptWhenQueryMentionsTestAndTwoNonTestRemain|TestHardTestExclusion_StillDropsWhenFewerThanTwoNonTestRemain"
        status: pass
    human_judgment: false
  - id: D4
    description: "H16 change-surface buried-rescue: rescues ONLY genuinely-buried (fileGraphScore<maxGraph*0.06 AND termHits<2) signature-type files reachable via a tier-seed's references/type_of/returns edges, force-kept at score=max(score,45); non-buried signature-type files and non-signature edge kinds are never rescued"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/scoring_test.go#TestBuriedRescue_RescuesGenuinelyBuriedSignatureTypeFile|TestBuriedRescue_ForcedScoreIsMaxNotOverwrite|TestBuriedRescue_NotBuriedNotRescued|TestBuriedRescue_TermHitsAboveThresholdNotRescued|TestBuriedRescue_NonSignatureEdgeKindIgnored"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 13: Per-File Scoring, Hard Exclusion & Buried-Rescue (H14-H16) Summary

**scoring.go: H14's per-file score tiers (+50/+10/+3/+1, keep>=3), H15's hard test/spec/icon/i18n exclusion (with its query-mentions-test+>=2-non-test exemption), and H16's change-surface buried-rescue exercising the new D-09 references/type_of/returns edges — the gate's (plan 14) three-stage input pipeline.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-07-15T14:47:00Z
- **Completed:** 2026-07-15T15:12:32Z
- **Tasks:** 2
- **Files modified:** 2 (both new)

## Accomplishments
- `computeFileScoreTiers` (H14): assigns every candidate node one of four tiers (named-seed/entry/connected-to-entry/other) via `classifyNodeTier`, rolls each file's score up to the MAX tier among its nodes, and drops any file scoring below the `>=3` keep threshold
- `computeConnectedToEntry`: the one-hop, RankEdges-filtered adjacency check that identifies "connected-to-entry" nodes without re-running RWR
- `aggregateFileGraphScore`: sums `computeGraphRelevance`'s (plan 06) per-node RWR mass into RESEARCH §4's `fileGraphScore` per file, deterministic sorted-Id summation order
- `applyHardTestExclusion` (H15): drops every low-value file (`isLowValueFile` = `isTestFile` reused from plan 07, OR `isIconOrI18nFile`) UNLESS the query mentions "test" AND >=2 non-low-value candidates remain — an all-or-nothing exemption, matching TS's short-circuit
- `applyBuriedRescue` (H16): for each tier-seed callable, walks its `references`/`type_of`/`returns` edges (the D-09 signature-type edge kinds) directly via `Reader.IterateEdges`, rescuing ONLY genuinely-buried targets (`fileGraphScore < maxGraph*0.06 AND termHits < 2`) at `score = max(existing, 45)` — reaching outside the bounded subgraph is the entire point of this heuristic, so it cannot be limited to a pre-built edge list
- All functions deterministic (sorted-Id iteration order throughout, D-04) and independently unit-testable against a synthetic `scoringFakeReader`, mirroring expand.go's/seeding.go's/gather.go's own reader-double conventions

## Task Commits

Each task's RED and GREEN steps were committed atomically:

1. **Task 1: H14 per-file score tiers + fileGraphScore aggregation** — RED `1034ed2` (test), GREEN `698132e` (feat)
2. **Task 2: H15 hard test exclusion + H16 buried-rescue** — RED `b5c39bc` (test), GREEN `90e2121` (feat)

_TDD gate sequence verified: `test(...)` commits precede their `feat(...)` commits in git log for both tasks (confirmed via `git log --oneline`)._

## Files Created/Modified
- `internal/query/scoring.go` - `computeFileScoreTiers`, `computeConnectedToEntry`, `classifyNodeTier`, `aggregateFileGraphScore` (H14); `isLowValueFile`, `isIconOrI18nFile`, `queryMentionsTest`, `applyHardTestExclusion` (H15); `signatureTypeTargets`, `isGenuinelyBuried`, `applyBuriedRescue` (H16) + their constant blocks
- `internal/query/scoring_test.go` - `scoringFakeReader` test double (mirrors expand_test.go's `expandFakeReader`) + scenario tests for every tier, the exclusion exemption, and the buried-rescue trigger/non-trigger cases

## Decisions Made
1. **Per-file H14 score = MAX tier, not sum** — RESEARCH's frozen citation pins the four constants and the `>=3` threshold but not the per-file rollup arithmetic. MAX is the design that makes "keep files with score>=3" cleanly mean "drop only pure-other-tier files," and avoids an unbounded-accumulation DoS surface the plan's own threat model flags as a concern for rescue-driven expansion.
2. **namedSeedIDs/entryIDs/tierSeedIDs/fileTermHits are caller-supplied, not derived here** — RESEARCH pins H14-H16's rule structure but not which upstream node-id sets feed them (H11's BFS roots, H13's seed tiers, H5's term-hit tracking). Consistent with expand.go's own "primitives now, wiring later" precedent — a later wiring plan composes these from plans 06/07/10/11/12's actual outputs.
3. **isIconOrI18nFile is a documented D-02 substitute** — no source detail on TS's exact icon/i18n patterns survives the frozen RESEARCH capture (the live dist JS is unreadable on this machine, a constraint every plan since 01-07 has hit). Implemented conservatively, mirroring `isTestFile`'s own directory-segment + filename-pattern shape.
4. **queryMentionsTest kept separate from gather.go's queryMentionsTestOrSpec** — RESEARCH cites different exact wording for H7 ("test/spec") vs H15 ("test" only); reusing H7's function would silently widen H15's exemption trigger beyond what's cited.
5. **applyBuriedRescue reads edges directly via `Reader.IterateEdges(srcID)`** rather than filtering a passed-in subgraph edge list — buried-rescue exists specifically to reach files the bounded subgraph traversal (H11) did NOT already surface, so limiting it to subgraph-internal edges would defeat its purpose.

## Deviations from Plan

None - plan executed exactly as written, including the H16 buried-rescue exercising the new D-09 `references`/`type_of`/`returns` edge kinds as the plan's objective required. The commit-splitting decisions above (per-file MAX rollup, caller-supplied entry/seed sets, icon/i18n substitute) are documented D-02 design choices within the plan's own explicit allowance ("Where a heuristic depends on data Go genuinely cannot produce, document it as an explicit D-02 allowed divergence"), not scope or behavior deviations from the plan's stated tasks/acceptance criteria.

## Issues Encountered
None. `go build ./...` and `go test ./internal/query/... -count=1` both clean. Full-repo `go test ./... -count=1` shows only the pre-existing, previously-documented flaky `internal/daemon.TestSoak` (unrelated to this plan — see 01-06-SUMMARY.md's Next-Phase-Readiness note; out of scope per the deviation rules' scope boundary, not fixed here).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `computeFileScoreTiers`, `aggregateFileGraphScore`, `applyHardTestExclusion`, and `applyBuriedRescue` are ready for plan 14's H17 relevance gate to compose: the gate's 5-way OR (graph mass >= 6% of max, central, entry/named-file, change-surface-rescued, distinctTermHits >= 2) consumes this plan's `fileScores`/`fileGraphScore` maps and the `rescued` set directly as its `changeSurfaceFiles` input.
- The caller-supplied `namedSeedIDs`/`entryIDs`/`tierSeedIDs` parameters are NOT yet wired to plans 06/11/12's actual outputs (computeGraphRelevance's seedIDs, expandBFS's roots, seedNamedSymbols' Primary tier) — that wiring is explicitly deferred to a later "wire time" plan, consistent with expand.go's own precedent for H10-H12.
- `fileTermHits` (H16's second buried-condition input) is likewise not yet computed anywhere in the codebase — this plan documents it as an opaque caller-supplied map; a later plan (likely 14, alongside H17's own `distinctTermHits >= 2` condition) owns computing it from the gather channels' term-hit data.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

All created files (internal/query/scoring.go, internal/query/scoring_test.go) and commit hashes (1034ed2, 698132e, b5c39bc, 90e2121) verified present.
