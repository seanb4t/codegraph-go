---
phase: 03-query-engine-mcp-server
plan: 03
subsystem: database
tags: [query-engine, lexical-search, pebble, graphstore, go]

# Dependency graph
requires:
  - phase: 03-query-engine-mcp-server
    provides: "03-02's Engine/OpenAt/ValidateKind/validateLimit construction seam and copyFixture/indexFixture test harness this plan builds Query/Search on top of"
provides:
  - "internal/query.Engine.Query — full node-record lexical search (D-06), --kind filtered, --limit capped, deterministically ranked"
  - "internal/query.Engine.Search — locations-only projection of the same matcher (D-06), for search/callers/callees/impact-style output"
  - "internal/query.Location — the exported name/kind/filePath/startLine shape 03-04's callers/callees/impact commands can reuse directly"
  - "internal/query.MarshalQueryJSON — the golden query.json {\"node\": {...}} envelope shape (D-05), isAsync/isStatic/isAbstract:false, no score key"
affects: [03-04, 03-06, mcp-server, cli-query-command]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "matchNodes: one Reader.IterateNodes() scan (D-03, no per-kind key scan) + in-memory --kind filter + lexicalMatchTier (exact > prefix > substring) + stable sort by (tier, QualifiedName, Id) for deterministic tie-break"
    - "Query/Search both call ValidateKind then validateLimit before any scan runs, so an unknown --kind or out-of-range --limit never touches the store (V5)"
    - "JSON shaping colocated in search.go (queryNodeJSON/queryNodeEnvelope/MarshalQueryJSON) per this plan's explicit instruction, not a shared render_json.go, to avoid Wave-3 file conflicts with 03-04/03-05"

key-files:
  created:
    - internal/query/search.go
    - internal/query/search_test.go
  modified: []

key-decisions:
  - "Location and its JSON tags (name/kind/filePath/startLine) are exported from search.go specifically so 03-04's callers/callees/impact commands can reuse the identical shape instead of redefining it"
  - "Visibility renders as *string (nil when the underlying schema.Node.Visibility is empty) so query --json emits \"visibility\": null for every current record, matching the golden fixture's literal value rather than omitting the key"
  - "matchNodes does not itself call ValidateKind — Query and Search each call ValidateKind + validateLimit explicitly before invoking matchNodes, so the RED test's spy-reader assertion (zero IterateNodes calls on an invalid --kind) holds for both entry points independently"
  - "Test-double type names (searchFakeReader, searchFakeNodeIterator) and JSON-shaping type names (queryNodeJSON, queryNodeEnvelope, renderQueryNodeJSON) are prefixed to avoid identifier collisions with 03-04/03-05's same-package test/render code landing in the same Wave-3 window"

patterns-established:
  - "Any future query verb needing lexical ranking reuses lexicalMatchTier/matchNodes directly; any future verb needing the locations-only shape reuses the exported Location type"

requirements-completed: [QRY-01, QRY-03]

coverage:
  - id: D1
    description: "query <term> returns full node records matching name or qualifiedName, filtered by --kind, capped by --limit"
    requirement: "QRY-01"
    verification:
      - kind: unit
        ref: "internal/query/search_test.go#TestQuery"
        status: pass
      - kind: unit
        ref: "internal/query/search_test.go#TestQueryRankingTieBreak"
        status: pass
      - kind: unit
        ref: "internal/query/search_test.go#TestQueryLimitCapsAfterRanking"
        status: pass
    human_judgment: false
  - id: D2
    description: "search <term> returns the lightweight locations-only projection (name/kind/filePath/startLine) for the same matches"
    requirement: "QRY-03"
    verification:
      - kind: unit
        ref: "internal/query/search_test.go#TestSearch"
        status: pass
    human_judgment: false
  - id: D3
    description: "Ranking is deterministic (exact-name > prefix > substring, stable tie-break) so --json output is byte-reproducible with no score field"
    requirement: "QRY-01"
    verification:
      - kind: unit
        ref: "internal/query/search_test.go#TestQueryRankingTieBreak"
        status: pass
      - kind: unit
        ref: "internal/query/search_test.go#TestQueryJSONShape"
        status: pass
    human_judgment: false
  - id: D4
    description: "An unknown --kind is rejected via ValidateKind before any node scan runs (V5, T-03-03-Kind)"
    verification:
      - kind: unit
        ref: "internal/query/search_test.go#TestQueryUnknownKindRejectedBeforeScan"
        status: pass
      - kind: unit
        ref: "internal/query/search_test.go#TestSearchUnknownKindRejectedBeforeScan"
        status: pass
    human_judgment: false

duration: 7min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 3: Deterministic Lexical Query/Search Summary

**`Engine.Query`/`Engine.Search` (D-06 lexical matcher, exact>prefix>substring tie-break) plus the golden `query.json` `{"node": {...}}` JSON shaping — one `IterateNodes()` scan shared by both, no embeddings, no score field.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-11T09:52:22-04:00 (RED commit)
- **Completed:** 2026-07-11T09:53:21-04:00 (GREEN commit)
- **Tasks:** 2 (RED + GREEN)
- **Files modified:** 2 (both created)

## Accomplishments
- `Engine.Query(term, kind, limit)` — full `*schema.Node` records, `--kind` filtered, `--limit` capped, ranked by `lexicalMatchTier` (exact-name > prefix > substring) with a stable secondary sort on `(QualifiedName, Id)` for deterministic ties
- `Engine.Search(term, kind, limit)` — the same match set projected to the exported `Location` (name/kind/filePath/startLine) shape, no source body/signature
- `matchNodes` does exactly one `reader.IterateNodes()` full scan per call (D-03, no per-kind key scan — RESEARCH Pitfall 1) with the `--kind` filter applied in memory
- Both `Query` and `Search` call `ValidateKind` then `validateLimit` before `matchNodes` ever runs, closing the T-03-03-Kind/DoS gaps — proved with a hand-built `searchFakeReader` whose `IterateNodes` increments a call counter, asserted zero on an unknown `--kind`
- `MarshalQueryJSON` reproduces the golden `query.json` shape: top-level array of `{"node": {...}}` envelopes, `isAsync`/`isStatic`/`isAbstract` present-and-`false`, `visibility` rendered as JSON `null` when unset, no `score` key, and byte-identical across two independent marshals of the same query

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): TestQuery/TestSearch pin the D-06 matcher, ranking, and JSON shapes** - `10cdc66` (test)
2. **Task 2 (GREEN): Implement Engine.Query/Engine.Search + matcher + JSON shaping** - `3bafa9b` (feat)

_TDD gate sequence confirmed: `test(03-03)` commit (`10cdc66`) precedes `feat(03-03)` commit (`3bafa9b`); no REFACTOR commit needed — GREEN bodies matched the already-designed shape from RED, no cleanup pass required._

## Files Created/Modified
- `internal/query/search.go` - `Location`, `lexicalMatchTier`/`matchNodes` (D-06 matcher + tie-break), `Engine.Query`/`Engine.Search`, `queryNodeJSON`/`queryNodeEnvelope`/`renderQueryNodeJSON`/`MarshalQueryJSON` (golden JSON shaping, colocated per this plan's instruction)
- `internal/query/search_test.go` - `searchFakeReader`/`searchFakeNodeIterator` (hand-built reader test double, own file, `engine_test.go` untouched), `TestQueryRankingTieBreak`/`TestQueryKindFilterOnFakeReader`/`TestQueryLimitCapsAfterRanking`/`TestQueryUnknownKindRejectedBeforeScan`/`TestSearchUnknownKindRejectedBeforeScan`, `TestQuery`/`TestSearch`/`TestQueryJSONShape` against the real indexed `gofixture` via `copyFixture`/`indexFixture`

## Decisions Made
- `Location` is exported from `search.go` (not `render_json.go` — no such shared file exists per this plan's explicit instruction) specifically so 03-04's `callers`/`callees`/`impact` commands can reuse the identical locations-only shape instead of redefining it
- `Visibility` renders as `*string`, nil when `schema.Node.Visibility == ""`, so `query --json` currently always emits `"visibility": null` — matching the golden fixture's literal value for every captured record (the extractor never sets `Visibility` today) rather than omitting the key or emitting `""`
- The tie-break ranking test (`TestQueryRankingTieBreak`) uses a hand-built `searchFakeReader` with four purpose-crafted nodes rather than the shared `gofixture`, because the fixture's real symbol set (main/helper/Alpha/Widget/Describe/Rename/Run/Base/Derived/Reader/ReadWriter/ID/Version/Registry) does not naturally contain a three-tier (exact/prefix/substring) match set for any single term, and `gofixture` is out of this plan's `files_modified` (shared across 03-04/03-05 in the same Wave-3 window)
- Test-double and JSON-shaping identifiers (`searchFakeReader`, `searchFakeNodeIterator`, `queryNodeJSON`, `queryNodeEnvelope`, `renderQueryNodeJSON`) are deliberately prefixed/specific rather than generic (`fakeReader`, `nodeJSON`) to reduce the risk of package-level identifier collisions with 03-04/03-05's own new files landing in the same package during the same Wave-3 parallel-execution window

## Deviations from Plan

None - plan executed exactly as written. `matchNodes` does not itself call `ValidateKind` (each of `Query`/`Search` calls it explicitly before delegating) — this is an implementation-detail choice within the plan's stated contract ("Both first call `ValidateKind(kind)`... and return its error before touching the store"), not a deviation from it.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/query.Engine.Query`/`Engine.Search`/`Location`/`MarshalQueryJSON` are ready for 03-04 (`callers`/`callees`/`impact`) to build reverse/forward traversal on top of, reusing `Location` for their own JSON shapes
- `internal/cli`'s future `query`/`search` command files (a later plan) can delegate directly to these two methods plus `MarshalQueryJSON` for `--json` output
- No blockers or concerns carried forward

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created files (`internal/query/search.go`, `internal/query/search_test.go`) and both task commit hashes (`10cdc66`, `3bafa9b`) verified present on disk / in `git log`.
