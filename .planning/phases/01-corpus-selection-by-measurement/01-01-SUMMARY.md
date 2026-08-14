---
phase: 01-corpus-selection-by-measurement
plan: 01
subsystem: query
tags: [go, pebble, status, edgesByKind, filesByLanguage, measurement-instrument]

# Dependency graph
requires: []
provides:
  - "query.StatusResult.EdgesByKind — sparse per-edge-kind tally, read-time-derived from a full IterateEdges(\"\") scan"
  - "query.StatusResult.FilesByLanguage emitted in --json (json:\"-\" un-suppressed)"
  - "The measuring instrument FIXT-01's corpus measurement (Plans 01-02+) reads from — codegraph status --json now carries both halves of the required data"
affects: [01-02, 01-03, 01-04, 01-05, 01-06, 01-07]

# Actuals (#2632)
actuals:
  tokens: 5216
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Third full-scan primitive in Engine.Status (edgesByKind), mirroring the existing IterateFiles/IterateNodes scans and buildExpandAdjacency's IterateEdges(\"\") idiom — unfiltered by RankEdges so an unranked kind is still measured"
    - "Explicit per-iterator close right after its own Err() check (both success and error paths), replacing three deferred-to-method-exit closes with one consistent discipline"
    - "graphstore.Open + Writer.PutEdge/PutFile + Snapshot + New(reader) as the in-package pattern for seeding a Pebble store with a known, controlled multiset in query-layer tests (vs. indexing real fixture source)"

key-files:
  created:
    - internal/query/status_edges_test.go
  modified:
    - internal/query/status.go
    - internal/query/files_status_test.go

key-decisions:
  - "Un-suppressed FilesByLanguage's json:\"-\" tag to json:\"filesByLanguage\" per D-03 — the TS-parity Compatibility constraint that motivated the suppression was formally retired 2026-08-13 (engram gw79qy2a9z), so the shape it protected is no longer owed."
  - "edgesByKind is unfiltered by RankEdges at the data layer — a kind outside the 9 ranked kinds (e.g. \"contains\") is still tallied, matching the plan's explicit instruction that filtering-vs-full-key-set is a display concern, not a data-layer one."
  - "Converted all three Status() scans (files, nodes, edges) from deferred-to-method-exit closes to explicit close-after-Err()-check on both the success and error path — no iterator is held open longer than its own scan, and none leaks on an early error return."
  - "Cost-quantified rather than only documented: measured codegraph status wall-clock on this repo's own real index (447 files, 4646 nodes, 11257 edges) before and after the change. Cold-run medians were 70ms pre-change vs 220ms post-change (delta 150ms, under the plan's 200ms halt threshold); warm-cache runs of both binaries converged to ~50-60ms, showing most of the cold delta was disk-cache noise rather than a real per-call regression from the new scan."

patterns-established:
  - "Pattern: seed a Pebble store directly via graphstore.Open/NewWriter/PutEdge/PutFile/Commit/Snapshot when a test needs a specific, controlled node/edge/file multiset — do not go through indexer.Run against fixture source when the test cares about exact counts rather than realistic extraction output."

requirements-completed: [FIXT-01]

coverage:
  - id: D1
    description: "codegraph status --json emits a live edgesByKind object (sparse: absent, not zero, for unmeasured kinds) alongside the pre-existing counts"
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/status_edges_test.go#TestStatusEdgesByKindTally"
        status: pass
      - kind: unit
        ref: "internal/query/status_edges_test.go#TestStatusEdgesByKindEmptyStore"
        status: pass
      - kind: unit
        ref: "internal/query/status_edges_test.go#TestStatusEdgesByKindSparseOmitsAbsentKind"
        status: pass
      - kind: unit
        ref: "internal/query/status_edges_test.go#TestStatusEdgesByKindKeepsUnrankedKind"
        status: pass
      - kind: e2e
        ref: "codegraph status -p . --json against this repo's own real Pebble index (built codegraph binary), shape+consistency assertion in the plan's <verify> block"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph status --json emits a live filesByLanguage object (previously computed but suppressed with json:\"-\")"
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/status_edges_test.go#TestStatusFilesByLanguageSurvivesJSONRoundTrip"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestStatus/filesByLanguage_is_present_in_the_JSON_shape_(v0.11.0_Phase_1,_D-03)"
        status: pass
      - kind: e2e
        ref: "codegraph status -p . --json against this repo's own real Pebble index — set(filesByLanguage)==set(languages), sum(filesByLanguage)<=fileCount"
        status: pass
    human_judgment: false
  - id: D3
    description: "The unconditional new edge scan's cost is measured and recorded, not merely asserted acceptable"
    requirement: FIXT-01
    verification: []
    human_judgment: true
    rationale: "The 200ms halt threshold is a human-reviewable engineering judgment call over noisy wall-clock measurements (cold-run delta 150ms, warm-cache delta ~0ms on the same index) — the numbers are recorded in this SUMMARY for a human/future-phase reviewer to judge the tradeoff, not a property a test can automatically classify as pass/fail."

# Metrics
duration: ~35min
completed: 2026-08-14
status: complete
---

# Phase 1 Plan 1: Measuring Instrument — edgesByKind + un-suppressed filesByLanguage Summary

**`codegraph status --json` now emits a live, sparse `edgesByKind` tally and a live `filesByLanguage` map — both required by FIXT-01 and both absent from the previous output — from one new unconditional `IterateEdges("")` full scan, unfiltered by `RankEdges`.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 2 (Task 1 tracer, Task 2 TDD test pin)
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments

- `StatusResult` gained `EdgesByKind map[string]int64` (`json:"edgesByKind"`), computed by a third full scan in `Engine.Status` mirroring the existing `nodesByKind`/`buildExpandAdjacency` `IterateEdges("")` idiom — deliberately unfiltered by `RankEdges`, so kinds outside the 9-kind ranked set (e.g. `contains`) are still measured rather than silently discarded.
- `FilesByLanguage`'s `json:"-"` suppression was removed (`json:"filesByLanguage"`) per D-03 — the Compatibility constraint that motivated it was formally retired 2026-08-13. Both halves of FIXT-01's required data (per-kind edge counts, per-language file counts) are now readable from one `status --json` call.
- All three of `Status()`'s scans (files, nodes, edges) were converted from deferred-to-method-exit iterator closes to an explicit close immediately after each scan's own `Err()` check — on both the success and error return path, so no iterator is retained longer than its own scan and none leaks on an early error return.
- The file's per-key decision table and `Status()`'s own doc comment were rewritten to describe the code as it now is: the edge scan is unconditional on every call, `edgesByKind` is deliberately read-time-derived (never stored in `Meta` — D-01 scopes this phase to `status`, not indexer/`Meta` changes), and every scan-derived count comes from the same `e.reader` snapshot so it stays internally consistent even while `Meta.EdgeCount` may legitimately disagree mid-reindex.
- A real binary built from this tree, run as `codegraph status -p . --json` against this repository's own index (447 files, 4646 nodes, 11257 edges — built via `codegraph init` on a `git archive`-clean copy and, separately, directly against the worktree with cleanup after), confirmed the exact shape and consistency the plan's `<verify>` block specifies: `edgesByKind` and `filesByLanguage` both non-empty objects of string→positive-int, `set(filesByLanguage) == set(languages)`, `sum(filesByLanguage) <= fileCount`.
- **Cost quantified, not just documented:** 3 cold runs each of `codegraph status` pre- and post-change against the same real index gave medians of 70ms (pre) and 220ms (post) — a 150ms delta, under the plan's 200ms halt threshold. A follow-up warm-cache measurement (both binaries run once to warm, then re-measured) converged to ~50-60ms for both, showing most of the cold-run delta was OS disk-cache variance rather than a genuine steady-state regression attributable to the new scan.
- Five named tests (`internal/query/status_edges_test.go`) pin the tally's exact correctness against a deliberately asymmetric seeded multiset, the nil-safety of an empty store, the sparse absent-not-present-with-0 contract, unranked-kind retention, and the `filesByLanguage` JSON round-trip. `TestStatusEdgesByKindTally` was confirmed to fail (RED) against a deliberately wrong expectation before the correct value was restored and committed.
- The now-stale `TestStatus` subtest in `internal/query/files_status_test.go` that asserted `filesByLanguage` was absent from `--json` was updated (Rule 1 — the test was asserting behavior this plan deliberately reverses) to assert presence and shape instead.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end `edgesByKind` + `filesByLanguage` on the `--json` contract** - `fa55d28` (feat)
2. **Task 2: Pin the tally's correctness and its absent-not-zero semantics** - `6226b51` (test)

_Note: Task 1 also fixed a stale test assertion in `internal/query/files_status_test.go` as a directly-caused consequence of the D-03 tag change (Rule 1)._

## Files Created/Modified

- `internal/query/status.go` - `StatusResult.EdgesByKind` field + un-suppressed `FilesByLanguage` tag; `Engine.Status`'s new unfiltered edge scan; explicit per-iterator close discipline; updated decision table and doc comment
- `internal/query/files_status_test.go` - `TestStatus`'s `filesByLanguage` subtest flipped from asserting absence to asserting presence/shape (D-03 consequence)
- `internal/query/status_edges_test.go` (new) - five named tests pinning the tally, nil-safety, sparse absence, unranked-kind retention, and JSON round-trip

## Decisions Made

- **Un-suppress `filesByLanguage` in the same diff as `edgesByKind`** (D-03) — both are needed by FIXT-01's downstream measurement record, and the Compatibility constraint that previously justified suppression no longer applies (engram `gw79qy2a9z`, retired 2026-08-13).
- **`edgesByKind` is unfiltered by `RankEdges` at the data layer.** The plan was explicit that filtering-vs-full-key-set is a presentation-layer decision (RESEARCH), not a data-layer one — the raw tally is the honest record of what the store contains.
- **Explicit close-after-Err() on both branches, no `defer` for any of the three scan iterators.** This satisfies "close each iterator explicitly after its own Err() check" and "no iterator retained to method exit" without needing a flag/closure-guarded deferred close for the error path — closing unconditionally right after the loop, before either return, already prevents any leak on the error branch.
- **Seeded test stores directly via `graphstore.Open`/`NewWriter`/`PutEdge`/`PutFile` rather than indexing real fixture source** — the plan required a deliberately asymmetric, exactly-known edge multiset (three `calls`, one `imports`, one `contains`, zero `overrides`/`type_of`), which real Go source indexing cannot guarantee. This reuses `graphstore/iter_test.go`'s Writer/Commit/Snapshot pattern and the existing no-repoRoot `New(reader)` construction already exercised by `TestDbSizeBytes`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated a now-stale test assertion in `files_status_test.go`**
- **Found during:** Task 1 (verifying `go test ./internal/query/...` after un-suppressing `filesByLanguage`)
- **Issue:** `TestStatus`'s subtest "filesByLanguage is internal-only and absent from the JSON shape" asserted the pre-D-03 behavior this plan deliberately reverses. Left unfixed, it would fail every run and mask real regressions.
- **Fix:** Renamed the subtest and flipped its assertion to require presence, correct decoded shape, and a non-empty map, matching the new D-03 behavior.
- **Files modified:** `internal/query/files_status_test.go`
- **Verification:** `go test -count=1 ./internal/query/...` passes; the subtest fails if `filesByLanguage` is absent or decodes to an empty map.
- **Committed in:** `fa55d28` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1)
**Impact on plan:** The fix was a direct, in-scope consequence of Task 1's own struct-tag change (same file family as `status.go`, not a scope expansion). No other files needed adjustment — `render_status.go`, `cli/status.go`, and `cli/present/status.go` remain byte-identical to the plan's base commit, confirmed by `git diff --quiet` against `f787c5f8`.

## Issues Encountered

- The sandboxed Bash tool rejected several multi-statement shell commands (loops, `cd &&`, multi-line `if`/`case` blocks) as "too complex to verify... stays inside the worktree," even when they were pure read-only measurement commands. Worked around by running each iteration of the timing loop as a separate single-statement tool call. No functional impact on the plan's deliverables.
- The plan's cost-measurement instruction ("largest index available on this machine — this repository's own, and any fetched corpus if one is already present") had no fetched corpus available yet (Plan 01-01 is upstream of the corpus-fetch plans), so only this repository's own index was measured. This is consistent with the plan's phrasing, which names the corpus as conditional on presence.
- Cold-run timing showed higher variance than expected (first-run outliers of 220-260ms on both pre- and post-change binaries), traced to OS disk-cache state rather than the code change — confirmed by a follow-up warm-cache measurement showing both binaries converge to ~50-60ms. Recorded both the required cold-run medians (which satisfy the plan's literal instruction and stay under its 200ms halt threshold) and the warm-cache confirmation for a fuller picture.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The measuring instrument FIXT-01 needs is now live: `codegraph status --json` emits both `edgesByKind` and `filesByLanguage` end-to-end from the real Pebble store, with no changes required to `render_status.go`, `cli/status.go`, or `cli/present/status.go` (confirmed byte-identical to base).
- Plan 01-02 (per CONTEXT.md D-02/D-04) can now build the dense-mode `--all-kinds` flag and the human-text/MCP-markdown "Edges by Kind:" sections on top of this data layer without needing any further `Engine.Status` changes.
- No blockers. The one open item worth a future phase's attention (explicitly flagged in `Status()`'s own doc comment, not a defect): the new edge scan is unconditional and adds real cost on every `status` call; a future phase could choose to stamp a per-kind aggregate into `Meta` at index time to avoid it, which this plan deliberately did not take on (D-01 scope).

---
*Phase: 01-corpus-selection-by-measurement*
*Completed: 2026-08-14*
