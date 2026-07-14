---
phase: 03-query-engine-mcp-server
plan: 04
subsystem: database
tags: [query-engine, call-graph, reverse-adjacency, bfs, pebble, graphstore, go]

# Dependency graph
requires:
  - phase: 03-query-engine-mcp-server
    provides: "03-02's Engine/OpenAt/clampDepth/validateLimit seam and copyFixture/indexFixture harness; 03-03's exported Location shape reused for callers/callees/impact/affected"
provides:
  - "internal/query.Engine.Callees — forward call-graph traversal via direct IterateEdges(srcID), no scan"
  - "internal/query.Engine.Callers — reverse call-graph traversal via the D-04 in-memory reverse-adjacency map"
  - "internal/query.Engine.Impact — depth-bounded reverse BFS blast radius (nodeCount/edgeCount arithmetic cross-checked against impact.json)"
  - "internal/query.Engine.Affected — D-07 query-time test-impact derivation over reverse calls edges + a _test.go/Test*/Benchmark* heuristic, no persisted test-coverage edge"
  - "internal/query.buildReverseAdjacency — the shared reverse-adjacency builder every reverse-traversal verb calls fresh (no cache)"
  - "internal/graphstore.Reader.IterateEdges(\"\") now correctly scans the whole e/ namespace (bug fix, prerequisite for D-04)"
affects: [03-05, 03-06, mcp-server, cli-callers-callees-impact-affected-commands]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "buildReverseAdjacency: one IterateEdges(\"\") scan filtered to goextract.RefKindCalls, keyed by edge.Target — built fresh inside every caller (Callers/Impact/Affected), never cached (T-03-04-Stale)"
    - "resolveSymbolNode: full IterateNodes() scan for an exact Name match, deterministic lowest-Id tie-break when multiple nodes share a name"
    - "Impact BFS: frontier-by-frontier expansion bounded by clampDepth(depth); nodeCount = distinct visited nodes including the symbol itself, edgeCount = reverse edges inspected while expanding each depth's frontier (golden-verified counting rule)"
    - "Affected: seed node ids from all symbols whose FilePath is in the changed-files set, then filter reverse-callers by isTestSymbol (_test.go suffix OR Test*/Benchmark* name prefix)"

key-files:
  created:
    - internal/query/traverse.go
    - internal/query/traverse_test.go
  modified:
    - internal/graphstore/pebble_store.go
    - internal/graphstore/iter_test.go

key-decisions:
  - "D-07 auto-approved under --auto: affected derives impacted test files at query time (reverse calls + test-file heuristic) rather than persisting a new test-coverage edge type. Persisting an edge would require reindexing the frozen Phase-2 graph and pulling Phase-5 (RES-02/RES-03) provenance-tagged-edge work forward into Phase 3 — both explicitly out of scope. The query-time derivation delivers identical user-facing behavior at none of that cost."
  - "buildReverseAdjacency filters to goextract.RefKindCalls only (not contains/embeds/imports edges) so callers/callees/impact/affected reflect only the call graph, matching the golden fixture's locations-only shape which contains nothing but call targets"
  - "Impact's edgeCount counts every reverse edge inspected while expanding each depth's frontier (not just edges leading to newly-discovered nodes) — this counting rule was cross-checked against testdata/golden/corpus/weft-go/impact.json's own arithmetic (symbol=mergeStyle depth=2 nodeCount=5 edgeCount=4 decomposes as 3 direct callers + 1 second-hop caller = 4 edges, 5 nodes including the symbol itself) before being applied to this plan's own fixture topology"
  - "AffectedResult's JSON shape ({files, affectedTests}) is this plan's own design — D-07a confirms no golden oracle exists for affected, so parity is structural/best-effort, not corpus-diffed"

patterns-established:
  - "Any future reverse-traversal query verb reuses buildReverseAdjacency directly; any future verb needing symbol->node resolution reuses resolveSymbolNode"

requirements-completed: [QRY-04, QRY-05, QRY-06]

coverage:
  - id: D1
    description: "callers/callees traverse the call graph correctly — callees via direct forward IterateEdges(srcID), callers via the D-04 reverse-adjacency map"
    requirement: "QRY-04"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestCallersCallees"
        status: pass
    human_judgment: false
  - id: D2
    description: "impact --depth returns a depth-bounded reverse blast radius with nodeCount/edgeCount arithmetic matching the golden counting rule, and an absurdly large --depth is clamped rather than unbounded"
    requirement: "QRY-05"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestImpact"
        status: pass
    human_judgment: false
  - id: D3
    description: "affected derives impacted test files from reverse calls edges + a test-file heuristic at query time — no persisted test-coverage edge (D-07)"
    requirement: "QRY-06"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffected"
        status: pass
    human_judgment: false
  - id: D4
    description: "IterateEdges(\"\") scans the whole e/ namespace (not just empty-src edges) — the prerequisite bug fix that makes D-04's reverse-adjacency scan possible at all"
    verification:
      - kind: unit
        ref: "internal/graphstore/iter_test.go#TestIterateEdgesEmptyPrefixScansEveryEdge"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 4: Call-Graph Traversal (Callers/Callees/Impact/Affected) Summary

**`Engine.Callers`/`Callees`/`Impact`/`Affected` over a fresh-per-call in-memory reverse-adjacency map (D-04), plus a prerequisite bug fix making `Reader.IterateEdges("")` actually scan the whole edge namespace instead of only empty-source edges.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-11T10:01:21-04:00 (RED commit)
- **Completed:** 2026-07-11T10:06:29-04:00 (GREEN commit)
- **Tasks:** 3 (RED + GREEN + non-blocking checkpoint)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `Engine.Callees(symbol, limit)` — forward call targets via a direct `IterateEdges(srcID)` range scan, filtered to `calls`-kind edges, no reverse-adjacency scan needed
- `Engine.Callers(symbol, limit)` — reverse callers via the D-04 in-memory reverse-adjacency map, built fresh per call
- `Engine.Impact(symbol, depth)` — depth-bounded reverse BFS blast radius, `clampDepth` (03-02) bounding the traversal (T-03-04-DoS); `nodeCount`/`edgeCount` counting rule cross-checked against `impact.json`'s arithmetic before being applied to this plan's own fixture
- `Engine.Affected(files)` — D-07 query-time test-impact derivation: walks reverse `calls` edges from every symbol defined in the changed files, keeps targets matching the `_test.go`/`Test*`/`Benchmark*` heuristic — no persisted test-coverage edge
- `buildReverseAdjacency` — the shared one-scan-per-call reverse-adjacency builder (filtered to `calls`-kind edges), never cached (T-03-04-Stale — a future long-lived MCP server process must never serve a stale point-in-time reverse view)
- `resolveSymbolNode` — deterministic symbol->node resolution (full `IterateNodes()` scan, lowest-Id tie-break on name collisions)
- **Prerequisite bug fix:** `internal/graphstore.pebbleReader.IterateEdges("")` was scanning only edges whose source is the literal empty string, not the whole `e/` namespace as `store.go`'s doc comment and D-04's design require — this plan is the first caller of `IterateEdges("")` in the codebase, and the bug would have silently made every reverse-traversal command (callers/impact/affected) return empty results

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): TestCallersCallees/TestImpact/TestAffected pin traversal + golden arithmetic** - `ceee4c8` (test)
2. **Prerequisite fix, discovered during GREEN: IterateEdges("") whole-namespace scan** - `59b8e1c` (fix)
3. **Task 2 (GREEN): Implement buildReverseAdjacency + Callers/Callees/Impact/Affected** - `60488d3` (feat)
4. **Task 3: D-07 confirmation checkpoint** - non-blocking, auto-approved under `--auto` (see Deviations)

_TDD gate sequence confirmed: `test(03-04)` commit (`ceee4c8`) precedes `feat(03-04)` commit (`60488d3`); the `fix(03-04)` commit (`59b8e1c`) landed between RED and GREEN because GREEN's first test run surfaced the `IterateEdges("")` bug as a blocking issue (Rule 3) before the traversal logic itself could be proven correct — no REFACTOR commit needed._

## Files Created/Modified
- `internal/query/traverse.go` - `buildReverseAdjacency`, `resolveSymbolNode`, `nodeLocation`, `CalleesResult`/`CallersResult`/`ImpactResult`/`AffectedResult`, `Engine.Callees`/`Callers`/`Impact`/`Affected`, `isTestSymbol`, `MarshalCalleesJSON`/`MarshalCallersJSON`/`MarshalImpactJSON`/`MarshalAffectedJSON` (golden JSON shaping, colocated per 03-03's established convention)
- `internal/query/traverse_test.go` - `traverseFixture` (seeds `pkga/target.go`+`pkga/target_test.go` into a copied gofixture, reusing `copyFixture`/`indexFixture` from `engine_test.go` at runtime, never editing that shared file), `TestCallersCallees`/`TestImpact`/`TestAffected`
- `internal/graphstore/pebble_store.go` - `IterateEdges` now special-cases `srcPrefix == ""` to scan the whole `e/` namespace, mirroring `IterateNodes`/`IterateFiles`'s existing whole-namespace-prefix pattern
- `internal/graphstore/iter_test.go` - `TestIterateEdgesEmptyPrefixScansEveryEdge` regression coverage for the fix above

## Decisions Made
- **D-07 (auto-approved under `--auto`, Task 3's non-blocking checkpoint):** `affected` derives impacted test files at query time from reverse `calls` edges + the `_test.go`/`Test*`/`Benchmark*` heuristic, rather than reading a persisted "test-coverage edge type" as QRY-06's literal wording implies. Rationale: persisting a test-coverage edge would require reindexing the frozen Phase-2 graph and pulling Phase-5 (RES-02/RES-03) synthesized/provenance-tagged-edge work forward into Phase 3 — both explicitly out of scope for this phase. The query-time derivation delivers identical user-facing behavior (list impacted test files for changed files) at none of that cost. Recorded here per the checkpoint's own instruction to make this decision visible.
- `buildReverseAdjacency` filters to `goextract.RefKindCalls` only — `contains`/`embeds`/`imports` edges are excluded from the reverse map so callers/callees/impact/affected reflect only the call graph, matching the golden `callers.json`/`callees.json`/`impact.json` shapes (locations-only, call targets exclusively)
- Impact's `edgeCount` counts every reverse edge inspected while expanding each depth's frontier (not only edges leading to a newly-discovered node) — this counting rule was derived by decomposing `impact.json`'s own arithmetic (`mergeStyle` depth=2: 3 direct callers + 1 second-hop caller = 4 edges traversed, 5 nodes visited including `mergeStyle` itself) before being encoded as the assertion in this plan's own fixture-based `TestImpact`
- `AffectedResult`'s JSON shape (`{files, affectedTests}`) is this plan's own design choice — D-07a confirms no golden oracle exists for `affected`, so its shape and parity are structural/best-effort rather than corpus-diffed
- `traverse_test.go`'s `TestAffected` fixture seeds a direct intra-package call (`Target()`/`TestTarget`) rather than a method call on a locally-constructed receiver (the RED test's original design used `Widget.Rename` via `w.Rename(...)`) — GREEN's first run surfaced that `internal/indexer`'s resolver does not track local-variable types (a Phase-2 decision recorded in STATE.md: "no local-variable-type-tracking logic implemented"), so `w.Rename(...)` produces no resolved `calls` edge at all; a direct unqualified call (the same shape `Alpha->helper` already proves works) was substituted

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `Reader.IterateEdges("")` was not scanning the whole edge namespace**
- **Found during:** Task 2 (GREEN) — the first test run against real traversal logic returned empty `Callers`/`Impact`/`Affected` results despite `Callees` (a direct, non-empty-prefix `IterateEdges(srcID)` call) working correctly.
- **Issue:** `edgeSrcPrefix("")` length-prefixes an empty `src` segment as a real, addressable (if never-written) key range in the `appendSegment` encoding — so `IterateEdges("")` was scanning only edges whose source happens to be the literal empty string (none exist), not the whole `e/` namespace as `store.go`'s own doc comment ("every edge whose source is srcPrefix") and D-04's design require. This is the first place in the codebase that calls `IterateEdges("")`, so the gap had gone unexercised until this plan.
- **Fix:** `pebbleReader.IterateEdges` now special-cases `srcPrefix == ""` to use `lower := []byte{prefixEdge}` (the same whole-namespace-prefix pattern already used by `IterateNodes`/`IterateFiles`), leaving all non-empty-`srcPrefix` behavior (used by `Callees`) unchanged.
- **Files modified:** `internal/graphstore/pebble_store.go`, `internal/graphstore/iter_test.go`
- **Verification:** New `TestIterateEdgesEmptyPrefixScansEveryEdge` regression test proves a 3-edge store returns all 3 edges from `IterateEdges("")`; full `internal/graphstore` and `internal/graphstore/archtest` suites pass; `TestCallersCallees`/`TestImpact`/`TestAffected` all pass after the fix.
- **Committed in:** `59b8e1c` (separate `fix(03-04)` commit, landed between the RED and GREEN commits)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix was necessary for any reverse-traversal command to function at all — without it, `callers`/`impact`/`affected` would silently return empty results for every symbol. No scope creep: the fix is scoped exactly to the `IterateEdges("")` empty-prefix case this plan is the first to exercise, and the archtest boundary (`internal/query`/`internal/mcp` importing only `internal/graphstore`'s interfaces, never Pebble directly) is unaffected.

## Issues Encountered

- The RED test's original `TestAffected` fixture design (a `Test*` function calling `Widget.Rename` via a locally-constructed `&Widget{}` receiver) produced no `calls` edge at all once indexed, because `internal/indexer`'s resolver does not track local-variable types (a Phase-2 design decision, not a bug in this plan's scope). Diagnosed by dumping the fixture's actual node/edge set during GREEN and cross-referencing against `internal/indexer/resolve.go`'s narrowest-safe-set design. Resolved by seeding a direct, unqualified intra-package call instead (`Target()`/`TestTarget`), the same resolvable call shape already proven by the fixture's existing `Alpha -> helper` edge.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/query.Engine.Callers`/`Callees`/`Impact`/`Affected` and their `MarshalXxxJSON` helpers are ready for 03-06's CLI command wiring (`callers`/`callees`/`impact`/`affected` Cobra commands) and the MCP companion tools
- `buildReverseAdjacency`/`resolveSymbolNode` are reusable by any future query verb needing symbol resolution or reverse traversal (e.g. `explore`'s blast-radius section in 03-05)
- `Reader.IterateEdges("")` is now correctly whole-namespace-scanning for any other package that adopts the same D-04 pattern later (Phase 8's persisted reverse-edge index, if ever built, does not need this fix, but any other v1 code relying on a full edge scan now works correctly)
- No blockers or concerns carried forward

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created/modified files (`internal/query/traverse.go`, `internal/query/traverse_test.go`, `internal/graphstore/pebble_store.go`, `internal/graphstore/iter_test.go`, this SUMMARY.md) and all three task commit hashes (`ceee4c8`, `59b8e1c`, `60488d3`) verified present on disk / in `git log`.
