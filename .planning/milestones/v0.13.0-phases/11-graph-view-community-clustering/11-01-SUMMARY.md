---
phase: 11-graph-view-community-clustering
plan: 01
subsystem: query-engine
tags: [gonum, louvain, community-detection, graph, connectrpc, proto, tdd]

requires:
  - phase: 05-file-package-graph-view
    provides: "FileGraph() rollup, CycleID/CycleCount precedent, uiv1.FileGraphNode/FileGraphResponse wire messages, the ELK layered layout this phase colours"
  - phase: 10-index-health-the-coverage-denominator
    provides: "the uiProtoFieldFixtureLenAtPlanNNNN chained-extension convention, house-format planning artifacts"
provides:
  - "corpora/graph-cluster-threshold.json — GRF-09's pass condition, committed alone as the phase's first commit"
  - "gonum.org/v1/gonum v0.17.0 promoted to a direct go.mod require"
  - "internal/query.AssignCommunities — deterministic Louvain community assignment over FileGraph()'s rollup"
  - "FileGraphNode.CommunityID / FileGraphResult.CommunityCount, computed fresh on every FileGraph() call"
  - "FileGraphNode.community_id = 5 / FileGraphResponse.community_count = 7 on the wire, field-for-field projected"
affects: [11-02-graphcluster-harness, 11-03-supply-chain-check, 11-04-ui-colouring, 11-05-mutation-log]

actuals:
  tokens: 17806
  tasks: 3
  commits: 5

tech-stack:
  added: [gonum.org/v1/gonum v0.17.0]
  patterns:
    - "Fresh-per-call graph algorithm computed inside FileGraph(), mirroring the CycleID/stronglyConnectedCycles precedent — no cache, no package-level state"
    - "Determinism via three mechanisms: sorted-path node ids, fixed math/rand/v2 PCG seed, canonical relabel by smallest member path"
    - "Test-only behavior seam (communityOptions) unreachable from production, used to demonstrate load-bearing determinism mechanisms via RED controls"
    - "Threshold-committed-alone, verdict-never-writes-threshold protocol (GRF-01 lineage)"

key-files:
  created:
    - corpora/graph-cluster-threshold.json
    - internal/query/community.go
    - internal/query/community_test.go
  modified:
    - go.mod
    - internal/query/traverse.go
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go
    - internal/uiserver/filegraph_test.go
    - web/src/lib/gen/ui_pb.ts
    - web/build/** (rebuilt)

key-decisions:
  - "Threshold JSON's measurementProtocol carries no duplicate `runs` field — the binding run count lives once, at metrics.clusteringTimeMs.runs, so there's exactly one number to keep in sync (deviation from the plan's literal field list, made to satisfy the plan's own acceptance gate — see Deviations)"
  - "Task 2's Commit A was split into a test(11-01) RED commit and a feat(11-01) GREEN commit, matching the tdd.md commit-scope contract, rather than one combined commit"
  - "TestFileGraphCommunitySourceIsFreshComputeOnly strips comment-only lines before its forbidden-substring scan, so pre-existing doc-comment prose (traverse.go's unrelated 'no sync.Once' commentary on BuildReverseAdjacency) cannot false-positive the assumption-delta invariant"

requirements-completed: [GRF-09, GRF-08, GRF-06]

coverage:
  - id: D1
    description: "corpora/graph-cluster-threshold.json committed alone, first, with GRF-09's pass condition and the D-07 fallback written down before any measurement exists"
    requirement: "GRF-09"
    verification:
      - kind: unit
        ref: "git show --name-only --format= <threshold commit> | wc -l == 1"
        status: pass
      - kind: unit
        ref: "node -e (threshold key/value assertions: max=500, statistic=median, runs=3, minNodes=2, guava pin, onFailure names community_id/field 50/PROMOTED, purpose says forbidden, prohibitions.length==2)"
        status: pass
    human_judgment: false
  - id: D2
    description: "AssignCommunities computes a deterministic, 1-based, canonical community id for every file node — sorted-path ids, fixed PCG seed, canonical relabel, summed undirected pair weights"
    requirement: "GRF-08"
    verification:
      - kind: unit
        ref: "internal/query/community_test.go#TestAssignCommunitiesDeterministic (structured + tie, 5 runs each)"
        status: pass
      - kind: unit
        ref: "internal/query/community_test.go#TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture (RED control: seed offset 1 flips the tie fixture; structured fixture invariant under the same offset)"
        status: pass
      - kind: unit
        ref: "internal/query/community_test.go#TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder"
        status: pass
      - kind: unit
        ref: "internal/query/community_test.go#TestAssignCommunitiesDegenerate"
        status: pass
      - kind: unit
        ref: "internal/query/community_test.go#TestUndirectedPairWeightsSumBothDirections"
        status: pass
      - kind: unit
        ref: "internal/query/community_test.go#TestFileGraphCommunitySourceIsFreshComputeOnly"
        status: pass
    human_judgment: false
  - id: D3
    description: "FileGraph() populates CommunityID/CommunityCount fresh on every call (no cache, no drift across repeated calls), and the values reach the wire field-for-field as community_id=5 / community_count=7, with the method set unchanged at 16"
    requirement: "GRF-06"
    verification:
      - kind: unit
        ref: "internal/query/community_test.go#TestFileGraphPopulatesCommunityFields"
        status: pass
      - kind: integration
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphProjectsEngineResult (extended: community_id/community_count compared against an independent eng.FileGraph() call)"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique, #TestUIServiceMethodSetIsExactlyTheReadSet"
        status: pass
      - kind: other
        ref: "task proto:drift (MATCH, 4 files) && task -s web:drift (both halves MATCH, web/build committed)"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-13
status: complete
---

# Phase 11 Plan 01: Threshold, Deterministic Louvain Clustering, and Wire Fields Summary

**Deterministic Louvain community detection (gonum v0.17.0) computed fresh inside `FileGraph()`, surfaced as `community_id`/`community_count` on the wire, with GRF-09's clustering-time threshold locked first and GRF-08's determinism proof hardened by a seed-perturbation RED control.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-13T00:00:00Z (approx, see git log)
- **Completed:** 2026-09-13
- **Tasks:** 3
- **Files modified:** 31 (across 5 commits)

## Accomplishments

- `corpora/graph-cluster-threshold.json` locked as the phase's sole, first commit — GRF-09's pass condition (median-of-3, 500ms max, guava pin) and the D-07 index-time-persistence fallback are both written down before any clustering code exists.
- `gonum.org/v1/gonum v0.17.0` promoted to a direct `go.mod` require — a one-line diff, `go.sum` unchanged, already pinned transitively.
- `internal/query/community.go`: `AssignCommunities` computes Louvain modularity over an undirected weighted graph built from `FileGraph()`'s rollup, with three determinism mechanisms (sorted node ids, fixed `rand.NewPCG` seed, canonical relabel by smallest member path) and a test-only seam (`communityOptions`) that never reaches production.
- `FileGraphNode.CommunityID` / `FileGraphResult.CommunityCount` populated fresh on every `FileGraph()` call, mirroring the `CycleID`/`CycleCount` precedent exactly — no cache, no store write.
- `FileGraphNode.community_id = 5` / `FileGraphResponse.community_count = 7` shipped on the wire in one commit (proto + regenerated Go/TS + fixture + handler + rebuilt `web/build`), with the method set unchanged at 16 rpcs.
- GRF-08's determinism proof hardened: a seed-perturbation RED control demonstrably flips a modularity-tying tie fixture (offset 1) while leaving a well-separated fixture unchanged; canonical relabeling is shown load-bearing (not cosmetic); degenerate/encoding cases pinned; summed undirected pair weights proven; a persisted, positive-controlled assumption-delta invariant (`TestFileGraphCommunitySourceIsFreshComputeOnly`) guards the fresh-compute-only contract for 11-05's mutation log.

## Task Commits

Each task was committed atomically (Task 2 split into a RED test commit and a GREEN implementation commit, per the tdd.md commit-scope contract):

1. **Task 1: Commit `corpora/graph-cluster-threshold.json` ALONE** — `698235a2` (feat)
2. **Task 2a: RED — failing tests for `AssignCommunities`/`CommunityID`** — `6e3b46bc` (test)
2. **Task 2b: GREEN — deterministic Louvain assignment fresh in `FileGraph()`** — `895ae50d` (feat)
2. **Task 2c: GREEN — wire slice (proto + gen + fixture + handler + web/build), ONE commit** — `8da21387` (feat)
3. **Task 3: GRF-08 hardening — RED-control seam, canonical relabel, degenerate cases, assumption-delta invariant** — `b2c9ae25` (test)

_thresholdCommit: `698235a2bb29dc5187940c43e4b51a08b3947c0a`_

No separate plan-metadata commit beyond this SUMMARY's own commit (per orchestrator instruction: STATE.md/ROADMAP.md are NOT updated by this plan — the orchestrator owns those writes).

## RED/GREEN Transcripts

### Engine (Task 2, Commit A)

RED (before `community.go`/`traverse.go` changes):

```
internal/query/community_test.go:96:11: undefined: AssignCommunities
internal/query/community_test.go:114:11: undefined: AssignCommunities
internal/query/community_test.go:174:8: n.CommunityID undefined (type FileGraphNode has no field or method CommunityID)
internal/query/community_test.go:179:9: got.CommunityCount undefined (type FileGraphResult has no field or method CommunityCount)
internal/query/community_test.go:190:12: second.CommunityCount undefined (type FileGraphResult has no field or method CommunityCount)
FAIL	github.com/seanb4t/codegraph-go/internal/query [build failed]
```

GREEN:

```
community_test.go:106: compared 5 runs
    community_test.go:136: compared 5 runs
--- PASS: TestAssignCommunitiesDeterministic (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/structured (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/tie (0.00s)
--- PASS: TestFileGraphPopulatesCommunityFields (0.00s)
--- PASS: TestFileGraphCyclesDeterministicIds (0.00s)
--- PASS: TestFileGraphPopulatesCycleFields (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/query	0.329s
```

The structured fixture's exact expected assignment (`a/*=1, b/*=2, lone.go=3, s/*=4`) matched the planning session's measured facts precisely on first implementation.

### Wire (Task 2, Commit B)

RED (proto regenerated, handler mapping not yet added):

```
filegraph_test.go:66: community_count = 0, want 2
filegraph_test.go:69: community_count = 0, want >= 1 (positive control: the gofixture yields at least one community)
filegraph_test.go:94: nodes[0].community_id = 0, want 1
--- FAIL: TestFileGraphProjectsEngineResult (0.18s)
```

GREEN:

```
filegraph_test.go:71: communities on the wire: 2 over 4 nodes
--- PASS: TestFileGraphProjectsEngineResult (0.23s)
--- PASS: TestFileGraphKindCountsAreSparse (0.17s)
--- PASS: TestUIServiceMethodSetIsExactlyTheReadSet (0.00s)
--- PASS: TestUIServiceDeclaresNoMutatingMethod (0.00s)
    readonly_test.go:643: inspected 49 messages and 202 fields in the generated uiv1 descriptor
--- PASS: TestUIProtoFieldNumbersAreStableAndUnique (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.820s
```

### GRF-08 hardening (Task 3)

RED (genuine, unplanned — see Deviations):

```
--- FAIL: TestFileGraphCommunitySourceIsFreshComputeOnly (0.00s)
    community_test.go:430: traverse.go contains "sync.Once" — D-15 forbids ...
```

GREEN:

```
community_test.go:222: seed offset 1 flipped the tie fixture
community_test.go:239: structured fixture invariant under the flipping offset
--- PASS: TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture (0.00s)
--- PASS: TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder (0.00s)
--- PASS: TestAssignCommunitiesDegenerate (0.00s)
--- PASS: TestUndirectedPairWeightsSumBothDirections (0.00s)
    community_test.go:464: inspected 8440 bytes (community.go) + 33897 bytes (traverse.go), 0 forbidden substrings, 2 positive controls
--- PASS: TestFileGraphCommunitySourceIsFreshComputeOnly (0.00s)
```

**Flipping seed offset found: 1.** Structured fixture confirmed invariant under that same offset.

## Files Created/Modified

- `corpora/graph-cluster-threshold.json` — GRF-09's pass condition, locked alone
- `go.mod` — `gonum.org/v1/gonum v0.17.0` direct require (+1 line)
- `internal/query/community.go` — `AssignCommunities`, `assignCommunitiesWith`, `undirectedPairWeights`, `canonicalCommunityLabels`, `communityOptions`/`communityDefaults`
- `internal/query/community_test.go` — determinism, RED-control, canonical-relabel, degenerate, pair-weight, and assumption-delta tests
- `internal/query/traverse.go` — `FileGraphNode.CommunityID`, `FileGraphResult.CommunityCount`, the `AssignCommunities` call site
- `internal/uiproto/uiv1/ui.proto` / `ui.pb.go` — `community_id = 5`, `community_count = 7`
- `web/src/lib/gen/ui_pb.ts` — regenerated TS client
- `internal/uiserver/handlers.go` — `CommunityId`/`CommunityCount` field mapping
- `internal/uiserver/readonly_test.go` — `uiProtoFieldFixtureLenAtPlan1101`, two fixture rows
- `internal/uiserver/filegraph_test.go` — `TestFileGraphProjectsEngineResult` extended
- `web/build/**` — rebuilt, drift-clean

## Decisions Made

- Split Task 2's "Commit A" into a `test(11-01)` RED commit and a `feat(11-01)` GREEN commit rather than one combined commit, following the canonical TDD commit-scope contract (`tdd.md`) — the plan's own wording ("test(11-01): … RED transcript pasted, then feat(11-01): …") supports this reading.
- Threshold JSON's `measurementProtocol` does not repeat `runs: 3` (only `metrics.clusteringTimeMs.runs` carries it) — see Deviations for why.
- `TestFileGraphCommunitySourceIsFreshComputeOnly` strips comment-only lines before scanning for forbidden substrings, so legitimate pre-existing doc-comment prose in `traverse.go` cannot false-positive the invariant.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan verification arithmetic bug] Threshold JSON's duplicate `"runs": 3` broke its own acceptance criterion**
- **Found during:** Task 1
- **Issue:** The plan's `<action>` text specified `runs: 3` in BOTH `metrics.clusteringTimeMs` and `measurementProtocol`, but the plan's own `<acceptance_criteria>` asserts `rg -o '"runs": 3' ... | wc -l` prints exactly 1 (while explicitly expecting `"statistic": "median"` to print 2 — metrics + measurementProtocol). Following the action literally produces 2 matches, failing the plan's own acceptance gate.
- **Fix:** Kept `runs: 3` only in `metrics.clusteringTimeMs` (the metric the harness reads per `<verify>`'s node-script assertions); added a `runsNote` in `measurementProtocol` documenting that the run count lives once. All `<verify>` and `<acceptance_criteria>` checks pass.
- **Files modified:** `corpora/graph-cluster-threshold.json`
- **Verification:** `rg -o '"runs": 3' corpora/graph-cluster-threshold.json | wc -l` = 1; `rg -o '"statistic": "median"' ...` = 2; the node-script key-assertion check passes.
- **Committed in:** `698235a2` (part of Task 1's commit)

**2. [Rule 1 - Test scan false positive] `TestFileGraphCommunitySourceIsFreshComputeOnly` flagged pre-existing, unrelated doc-comment prose**
- **Found during:** Task 3
- **Issue:** The forbidden-substring scan (per the plan's `<behavior>` spec) checked whole-file text including comments. `traverse.go`'s pre-existing doc comments for `BuildReverseAdjacency`/`BuildImplementsIndex` legitimately discuss "no package-level cache, no `sync.Once`" in prose — unrelated to the community assignment path — which the literal substring `sync.Once` matched, producing a false RED.
- **Fix:** Added `stripCommentLines` to remove `//`-prefixed lines before scanning, so the invariant checks actual code, never prose describing what is absent.
- **Files modified:** `internal/query/community_test.go`
- **Verification:** `TestFileGraphCommunitySourceIsFreshComputeOnly` passes; the two positive controls (`AssignCommunities(result.Nodes, result.Edges)` once in `traverse.go`, `community.Modularize(` once in `community.go`) still hold.
- **Committed in:** `b2c9ae25` (part of Task 3's commit)

**3. [Rule 1 - Plan verification arithmetic bug] Several `<verify>` regex counts did not account for `t.Run` subtest line substring matches**
- **Found during:** Tasks 2 and 3
- **Issue:** Multiple `<verify>` blocks assert an exact count of `--- PASS: Test<Name>` lines via unanchored `rg -o` patterns (e.g. expecting 4 for Task 2's engine gate, 7 for Task 3's hardening gate). Because `TestAssignCommunitiesDeterministic` and `TestAssignCommunitiesDegenerate` use `t.Run` subtests, `go test -v` prints additional lines like `    --- PASS: TestAssignCommunitiesDeterministic/structured (0.00s)`, and the unanchored regex (no `^` anchor, no trailing ` (`) matches the parent test name as a substring of each subtest line too — producing 6 matches instead of 4, and 14 instead of 7. The `<behavior>` spec explicitly requires these tests to use `t.Run` sub-tests, so removing them to satisfy the literal count would violate the test-content contract, which is more authoritative.
- **Fix:** Verified the actual intent (distinct top-level test names, all PASS, zero FAIL) with an anchored equivalent (`^--- PASS: Test(...) \(`), which returns the expected counts (4 and 7 respectively) in both cases. No test or implementation code was changed for this — it is a plan-authored verification imprecision, not a defect in the shipped tests.
- **Files modified:** None (verification-only; documented here per the "log as deviation with reason" instruction after 2 fix attempts on an unsatisfiable literal check)
- **Verification:** Anchored counts: 4/4 (Task 2) and 7/7 (Task 3); zero FAIL in both runs; all named PASS lines present.
- **Committed in:** N/A (no code change)

**4. [Rule 1 - Plan verification arithmetic bug] `./internal/query/...` and combined package runs always include `internal/query/archtest`, which the plan's exact `ok`-line and "no tests to run" counts did not account for**
- **Found during:** Tasks 2 and 3
- **Issue:** Several `<verify>`/`<acceptance_criteria>` commands run `go test ... ./internal/query/...` with a `-run` filter scoped to `internal/query` package test names, or assert an exact `ok`-line count (e.g. "two ok lines"). `./internal/query/...` always expands to include the `internal/query/archtest` subpackage. When a `-run` filter matches nothing in `archtest`, Go prints `testing: warning: no tests to run` for that package (which the plan's `! rg -q 'no tests to run'` check would then fail on) and an additional `ok ... archtest ... [no tests to run]` line (making 3 `ok` lines where the plan expected 2).
- **Fix:** None applied to code — this is inherent, expected Go test-runner behavior for a `-run`-filtered run over a package tree that includes a sibling subpackage with no matching tests. Confirmed via `go test -count=1 ./internal/query/... ./internal/uiserver/...` (no `-run` filter) that all three packages (`internal/query`, `internal/query/archtest`, `internal/uiserver`) report `ok` with zero `FAIL`, which is the actual correctness signal these gates exist to protect.
- **Files modified:** None
- **Verification:** `go test -count=1 ./internal/query/... ./internal/query/archtest/... ./internal/uiserver/...` → 3 `ok` lines, 0 `FAIL` (Task 3's own final gate, which correctly expects 3 `ok` lines, confirms this is the accurate shape)
- **Committed in:** N/A (no code change)

---

**Total deviations:** 4 (2 auto-fixed in code/data, 2 documented verification-only findings). **Impact on plan:** No scope creep. Deviations 1 and 2 are genuine correctness fixes (a data-file self-consistency bug and a test false-positive) required for the plan's own gates to pass meaningfully. Deviations 3 and 4 are pre-existing imprecision in the plan's literal verification commands that does not reflect any defect in the shipped code — the underlying tests all pass, and the true intent of each check (all named tests PASS, zero FAIL, positive controls present) was independently confirmed.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Ready for **11-02** (the `tools/graphcluster` GRF-09 measurement harness and its ancestry test): the threshold artifact, the engine (`AssignCommunities`), and the wire fields all exist and are proven. `internal/schema/graph.proto`'s reserved `50-59` range remains untouched, and the assumption-delta invariant test (`TestFileGraphCommunitySourceIsFreshComputeOnly`) is in place for 11-02 Task 3 to invert on a FAIL verdict, and for 11-05's mutation log to watch RED.

No blockers.

---
*Phase: 11-graph-view-community-clustering*
*Completed: 2026-09-13*

## Self-Check: PASSED

All key files confirmed present on disk (`corpora/graph-cluster-threshold.json`, `internal/query/community.go`, `internal/query/community_test.go`, `internal/query/traverse.go`, `internal/uiproto/uiv1/ui.proto`, `internal/uiserver/handlers.go`, `internal/uiserver/readonly_test.go`, `internal/uiserver/filegraph_test.go`, `web/src/lib/gen/ui_pb.ts`). All 5 task commits confirmed present in `git log` (`698235a2`, `6e3b46bc`, `895ae50d`, `8da21387`, `b2c9ae25`).
