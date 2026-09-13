---
phase: 10-index-health-the-coverage-denominator
plan: 03
subsystem: indexer
tags: [sync, incremental, exclusion-diff, graphstore, tdd]

# Dependency graph
requires:
  - phase: 10-index-health-the-coverage-denominator
    provides: "Plan 01's PutExcludedFile/DeleteExcludedFile/IterateExcludedFiles store primitives and Meta.has_coverage field; Plan 02's all-four-reasons DiscoverAll (this plan's tests use ONLY the BUILD_TAG reason, per the plan's own wave-2 scoping note, to stay order-independent of Plan 02's landing)"
provides:
  - "internal/indexer/synccoverage.go: loadStoredExclusions, diffExclusions, stageExclusionDiff — the pure D-07 incremental lifecycle for the c/ namespace"
  - "Sync's coverageDirty gate: an exclusion-only change (new excluded file appears, excluded file disappears, or has_coverage was never recorded) is never mistaken for a no-op"
  - "Both of Sync's meta-write sites now stamp HasCoverage=true unconditionally, replacing the tracer's carry-forward placeholder"
affects: [10-04-health-page-coverage-section, 10-05-coverage-rows-pagination-and-security, 10-06-mutation-log-and-security-doc]

# Actuals (#2632)
actuals:
  tokens: 6783
  tasks: 2
  commits: 2
plan_head_before: 91702a1e

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "diffExclusions upserts only absent-or-proto.Equal-changed records and prunes only stored paths absent from the fresh set, mirroring Sync's own pre-existing `deleted` diff over r0.IterateFiles() — a second namespace can adopt the same per-path incremental-lifecycle shape without inventing a new technique."
    - "Every schema.NewMeta() write site in an incremental (non-from-scratch) code path must stamp EVERY additive Meta boolean flag independently and unconditionally when its own commit batch makes that flag true — never carry a prior value forward 'until a later plan fills it in'. A carry-forward placeholder is the exact tracer anti-pattern this plan replaced (Pitfall 3)."

key-files:
  created:
    - internal/indexer/synccoverage.go
    - internal/indexer/synccoverage_test.go
    - internal/indexer/sync_coverage_test.go
  modified:
    - internal/indexer/sync.go

key-decisions:
  - "fabricatePreCoverageStore takes no boolean parameter and always forces HasCoverage=false directly (a literal assignment, not a variable) — this satisfies the plan's own structural verify gate (>=2 literal 'HasCoverage = false' occurrences across the two fabrication helpers) and is also the only value the helper is ever called with in practice."
  - "stageExclusionDiff stages deletes before puts on the same Writer, sharing exactly Sync's existing two NewWriter call sites (WR-03 small-commit path and the main incremental path) — never a third writer, per T-10-10 and the plan's own NewWriter-count-stays-2 prohibition."

patterns-established:
  - "A per-path incremental-lifecycle diff for a new graphstore namespace (upsert absent/changed, prune gone) belongs in its own small file (synccoverage.go) with pure, independently unit-tested functions, wired into Sync's existing write sites via one stage-on-caller's-Writer helper — never a bespoke inline diff at each call site."

requirements-completed: []

coverage:
  - id: D1
    description: "Sync upserts every currently-excluded path whose stored record is absent or differs (proto.Equal) and prunes every stored c/ record whose path is no longer excluded, inside the SAME Writer commit as File/Node/Edge mutations, never a second writer"
    requirement: "HLT-05"
    verification:
      - kind: unit
        ref: "internal/indexer/synccoverage_test.go#TestDiffExclusionsUpsertsAbsentAndChangedOnly"
        status: pass
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncPrunesAnExclusionThatBecameIndexable"
        status: pass
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncPrunesAnExclusionWhoseFileWasDeleted"
        status: pass
    human_judgment: false
  - id: D2
    description: "A Sync whose only on-disk change is an exclusion change is NOT a no-op: the new record is committed and Meta.has_coverage is stamped true — the fully-no-op early return fires only when files, mtime refreshes AND exclusions are all unchanged AND coverage is already recorded"
    requirement: "HLT-05"
    verification:
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncRecordsANewBuildTagExclusionWithoutAnyIndexedFileChange"
        status: pass
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncIsANoOpWhenNothingChangedAndCoverageIsRecorded"
        status: pass
    human_judgment: false
  - id: D3
    description: "A graph with has_file_index==true but has_coverage unset (a post-Phase-4, pre-Phase-10 store) is backfilled by ONE incremental Sync; a graph with has_file_index unset still takes the existing full-run backfill, which stamps both"
    requirement: "HLT-05"
    verification:
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncBackfillsCoverageOnAGraphThatHasFileIndexButNoCoverage"
        status: pass
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncFullBackfillStampsBothFlags"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both schema.NewMeta() sites in sync.go stamp HasCoverage=true unconditionally after their commit includes the exclusion diff — the tracer's carry-forward placeholder is gone; the mtime-refresh-only path never drops previously recorded coverage"
    requirement: "HLT-05"
    verification:
      - kind: integration
        ref: "internal/indexer/sync_coverage_test.go#TestSyncMtimeRefreshOnlyPathKeepsCoverageRecorded"
        status: pass
      - kind: other
        ref: "rg -o 'HasCoverage = true' internal/indexer/sync.go | wc -l -> 2; rg -o 'HasCoverage = meta\\.GetHasCoverage\\(\\)' internal/indexer/sync.go | wc -l -> 0"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-09-13
status: complete
---

# Phase 10 Plan 3: Sync's Incremental c/ Namespace Lifecycle Summary

`Sync` now diffs the freshly-walked exclusion set against the stored `c/` namespace on every call — upserting new/changed records and pruning stale ones inside its existing single-Writer commit — and both of its meta-write sites unconditionally stamp `has_coverage`, so an exclusion-only change is never mistaken for a no-op and a pre-Phase-10 graph is backfilled by the very next incremental sync.

## Performance

- **Duration:** ~35 min
- **Started:** 2026-09-13T02:13:00Z (approx.)
- **Completed:** 2026-09-13T02:48:10Z
- **Tasks:** 2 (both `tdd="true"`)
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments

- `synccoverage.go` adds three pure, independently unit-tested functions — `loadStoredExclusions`, `diffExclusions`, `stageExclusionDiff` — giving the `c/` namespace the exact per-path incremental lifecycle (upsert absent/changed, prune gone) that `f/` File records already had, using `proto.Equal` so an unchanged record is never re-written
- `Sync`'s no-op gate now also fires on `coverageDirty` (`len(exclUpserts) > 0 || len(exclPrunes) > 0 || !meta.GetHasCoverage()`), so a new build-tag-excluded file appearing with zero other disk changes is no longer silently dropped by the fully-no-op early return — this is the exact scenario the tracer's carry-forward left broken, proven RED before the fix existed
- Both `schema.NewMeta()` write sites in `sync.go` (the WR-03 small-commit path and the main incremental path) now stamp `HasCoverage = true` unconditionally, replacing the "carry the prior value forward" placeholder comment Plan 01/02's tracer left behind (RESEARCH Pitfall 3)
- A pre-Phase-10 graph (`has_file_index` true, `has_coverage` false/unset, empty `c/` namespace) is proven to become "known" after exactly ONE incremental `Sync` with no disk change — via the small-commit path (`FilesReparsed == 0`), not a full re-index
- The from-scratch backfill path (`has_file_index` false, delegating to `run()`/`writeGraph`) and the mtime-refresh-only path are both proven to leave the `c/` namespace and `has_coverage` exactly as D-06/D-07 require: no duplicate record, no coverage regression
- `NewWriter(` call-site count in `sync.go` stayed at exactly 2 throughout (T-10-10); `sync.go` never calls `DeleteAllExcludedFiles` (that range-delete discipline stays exclusive to the from-scratch `writeGraph` path)

## Task Commits

1. **Task 1: Pure exclusion diff + the two Sync write sites** - `3883b18a` (feat, includes `synccoverage.go`, `synccoverage_test.go`, and the first 5 tests in `sync_coverage_test.go`)
2. **Task 2: Backfill and preservation tests** - `be615003` (test, adds the remaining 3 tests + 2 fabrication helpers to `sync_coverage_test.go`; no production code change needed)

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `internal/indexer/synccoverage.go` - `loadStoredExclusions`/`diffExclusions`/`stageExclusionDiff`, the D-07 incremental lifecycle for the `c/` namespace
- `internal/indexer/synccoverage_test.go` - `TestDiffExclusionsUpsertsAbsentAndChangedOnly` (pure diff unit test)
- `internal/indexer/sync_coverage_test.go` - 7 Sync-lifecycle tests through a real store: 2 new-exclusion/no-op tests, 2 prune tests, 1 true-no-op test, 1 backfill test, 1 mtime-preservation test, 1 full-backfill test, plus `findExcluded`/`fabricatePreCoverageStore`/`fabricateNoFileIndexStore` helpers
- `internal/indexer/sync.go` - wired `loadStoredExclusions`/`diffExclusions` after the `deleted` diff, extended the no-op gate with `coverageDirty`, called `stageExclusionDiff` at both write sites before `PutMeta`, replaced both `HasCoverage = meta.GetHasCoverage()` carry-forwards with `HasCoverage = true`

## Decisions Made

- `fabricatePreCoverageStore` forces `HasCoverage = false` as a direct literal (no boolean parameter) so the fabrication is unambiguous in the test source and satisfies the plan's structural verify gate expecting the literal pattern to appear at least twice across the two fabrication helpers.
- Task 2's three tests were written directly against Task 1's already-correct implementation (no RED phase was required or claimed for Task 2 — the plan itself frames Task 2 as "if it fails, the fix is confined to sync.go's gate/stamp lines from Task 1"). All three passed on first run, confirming Task 1's `coverageDirty` gate and Plan 01's `writeGraph` range-delete are both correct without further changes.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

One run of `go test ./internal/indexer/... ./internal/daemon/...`, executed concurrently with a separate foreground test invocation while both competed for CPU, hit a single flaky failure in `TestRunWatchdogCancelsRunOnSimulatedReparent` (`internal/daemon`) — a test whose own source comment explicitly documents this exact failure mode as a known load-induced flake under heavy concurrent full-suite load (unrelated to `internal/indexer`). Confirmed not a regression: (1) the same test passes in isolation in 1.08s, (2) three other full-suite runs of `./internal/indexer/... ./internal/daemon/...` — two run serially before this incident and one run independently afterward — all passed cleanly (~66s each, daemon package included), and (3) this plan's changes never touch `internal/daemon`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The `c/` namespace now has a complete incremental lifecycle (upsert/prune per path) matching `f/`'s, so a long-running daemon never reports a stale or unknown coverage denominator after its first sync — Plan 04 (health page) and Plan 05 (coverage rows pagination) can rely on `has_coverage`/the `c/` namespace being kept honest across every future `Sync` call, not just a from-scratch index.
- `HLT-05` is NOT marked complete in REQUIREMENTS.md yet: it is also declared by sibling plans 10-01 (already summarized), 10-02 (already summarized), 10-05 and 10-06 (not yet summarized) — `requirements.ready-ids` correctly reports it `blocked` until every declaring plan finishes. No action needed here; it will flip automatically when the last sibling plan completes.
- No blockers.

## Self-Check: PASSED

All key files present on disk: `internal/indexer/synccoverage.go`, `internal/indexer/synccoverage_test.go`, `internal/indexer/sync_coverage_test.go`, `internal/indexer/sync.go`. Both commits (`3883b18a`, `be615003`) found in `git log --oneline --all`.

---
*Phase: 10-index-health-the-coverage-denominator*
*Completed: 2026-09-13*
