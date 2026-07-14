---
phase: 02-go-indexing-pipeline
plan: 05
subsystem: indexer
tags: [pipeline, orchestration, determinism, graphstore, export, pass-1, pass-2]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline (plans 01-04)
    provides: internal/indexer.Discover (file walk + modulePath), internal/indexer.Extract (Pass 1 worker pool), internal/indexer.Resolve (Pass 2 single-writer commit), internal/indexer/goextract node-kind taxonomy, the committed example.com/gofixture multi-package test fixture
  - phase: 01 (all plans)
    provides: internal/graphstore.GraphStore.Open/Close/Snapshot/Export, internal/schema.Meta/Node/Edge
provides:
  - internal/indexer.Run(repoRoot, storeDir string, opts Options) (Stats, error) — the single entry point wiring Discover -> Extract -> Resolve into one from-scratch, committed, version-stamped graph
  - internal/indexer.Options / internal/indexer.Stats — CLI-facing configuration and result summary
  - Automated determinism gate (INDX-02): two from-scratch indexes of the fixture produce byte-identical GraphStore.Export() streams after normalizing Meta.last_sync_unix_ms
  - Automated structural fixture-diff (RES-01/LANG-01, success-criterion-4 stand-in): full node-kind taxonomy plus four named cross-file edges verified against the committed graph
affects: [cli-index-command, phase-3-mcp-queries, phase-8-reproducibility-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "run() takes resolveFunc as a parameter (never hard-codes package-level Resolve) purely as a testing seam, mirroring extract.go's parserFactory injection pattern — lets a test simulate a Pass-2 failure and assert Close-once discipline without depending on a real Resolve error condition"
    - "Stats.Nodes/Edges are read back from the just-committed Meta record via a fresh Snapshot (readGraphCounts), never separately re-derived, so they can never drift from what writeGraph actually stamped"
    - "The determinism diff always targets GraphStore.Export() output (frame-decoded and Meta.last_sync_unix_ms zeroed), never raw Pebble .sst/.log files — LSM internals aren't byte-stable across independently-built stores"

key-files:
  created:
    - internal/indexer/pipeline.go
    - internal/indexer/pipeline_test.go
    - internal/indexer/determinism_test.go
  modified:
    - internal/indexer/testdata/gofixture/pkga/embed.go

key-decisions:
  - "Options.Workers <= 0 defaults to runtime.NumCPU() inside run(), mirroring Extract's own default, so Options{} and Options{Workers: runtime.NumCPU()} are observably identical to callers"
  - "The gofixture gained a `type ID = int` declaration (type_alias) so the structural fixture-diff can assert every node kind goextract's taxonomy produces, not just the ones the pre-existing fixture happened to exercise"
  - "TestDeterministicRebuild forces GOMAXPROCS(8) for the duration of the test and is run under -race, specifically to surface any residual goroutine/map-order nondeterminism in Pass 1/2 rather than let it hide behind a low-concurrency default test run"

requirements-completed: [INDX-02, RES-01]

coverage:
  - id: D1
    description: "Run(repoRoot, storeDir, opts) orchestrates Discover -> Extract -> Resolve into a single committed graph with Stats matching Meta's stamped node/edge counts"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/pipeline_test.go#TestPipelineRun"
        status: pass
    human_judgment: false
  - id: D2
    description: "The GraphStore is opened exactly once and Closed on every path, including when Pass 2 fails (T-02-11 resource-leak mitigation)"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/pipeline_test.go#TestPipelineRun_ClosesStoreOnResolveError"
        status: pass
    human_judgment: false
  - id: D3
    description: "Options.Workers <= 0 defaults the Pass-1 pool to runtime.NumCPU() without panicking or short-circuiting"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/pipeline_test.go#TestPipelineRun_DefaultsWorkersToNumCPU"
        status: pass
    human_judgment: false
  - id: D4
    description: "Indexing the fixture twice from scratch into separate temp stores yields byte-identical GraphStore.Export() streams after normalizing Meta.last_sync_unix_ms, stable under -race and GOMAXPROCS(8)"
    requirement: INDX-02
    verification:
      - kind: unit
        ref: "internal/indexer/determinism_test.go#TestDeterministicRebuild"
        status: pass
    human_judgment: false
  - id: D5
    description: "The committed graph contains every expected node kind (file/function/method/struct/interface/type_alias/constant/variable/package) and the four named cross-file edges (calls/embeds x2/imports)"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/determinism_test.go#TestRealRepoStructure"
        status: pass
    human_judgment: false
  - id: D6
    description: "Real-repo (weft-go corpus) node-kind and call/contains-edge parity spot-check per 02-VALIDATION.md"
    requirement: RES-01
    verification: []
    human_judgment: true
    rationale: "Manual corpus spot-check documented as the human-verification path in 02-VALIDATION.md; not automatable within this plan's scope (requires an external checked-in corpus repo comparison, deferred to phase validation)"

# Metrics
duration: 25min
completed: 2026-07-11
status: complete
---

# Phase 2 Plan 05: Pipeline Orchestration and Determinism Gate Summary

**`indexer.Run` wires Discover -> Extract -> Resolve into one from-scratch committed graph, proven byte-identical across independent rebuilds via an automated Export()-diff gate (INDX-02) plus a structural fixture-diff of node kinds and cross-file edges (RES-01/LANG-01)**

## Performance

- **Duration:** 25 min
- **Completed:** 2026-07-11T02:31:22Z
- **Tasks:** 2
- **Files created:** 3 (`internal/indexer/pipeline.go`, `internal/indexer/pipeline_test.go`, `internal/indexer/determinism_test.go`)
- **Files modified:** 1 (`internal/indexer/testdata/gofixture/pkga/embed.go`)

## Accomplishments
- `indexer.Run(repoRoot, storeDir string, opts Options) (Stats, error)` opens the GraphStore exactly once (`defer store.Close()`), runs `Discover -> Extract -> Resolve`, and returns `Stats` (Files/Nodes/Edges/Unresolved/Skipped/Duration) read back from the just-committed `Meta` record so counts can never drift from what was actually stamped.
- The store is Closed on every return path, including an injected Pass-2 failure (`TestPipelineRun_ClosesStoreOnResolveError` proves the directory is cleanly reopenable afterward — no leaked Pebble lock).
- `Options.Workers <= 0` defaults to `runtime.NumCPU()`, matching `Extract`'s own default.
- `TestDeterministicRebuild` (INDX-02): indexes the shared multi-package fixture twice from scratch into two separate `t.TempDir()` stores, `Export()`s both, normalizes the one genuinely volatile field (`Meta.last_sync_unix_ms`, zeroed in the decoded frame) and asserts the byte streams are identical — run under `-race` with `GOMAXPROCS` forced to 8 to surface any residual goroutine/map-order nondeterminism. Passed on first run against the already-implemented Pass 1/2 code (no determinism bug found).
- `TestRealRepoStructure`: indexes the fixture once and asserts the committed graph contains the full node-kind taxonomy (file/function/method/struct/interface/type_alias/constant/variable/package) plus the four named cross-file edges (`pkgb.Run -calls-> pkga.Alpha`, `Derived -embeds-> Base`, `ReadWriter -embeds-> Reader`, `pkgb/pkgb.go -imports-> package example.com/gofixture/pkga`) — the automated stand-in for a real-repo structural spot-check.
- Extended the shared `gofixture` with `type ID = int` (`internal/indexer/pkga/embed.go`) so the `type_alias` node kind (D-06) is actually exercised by the structural fixture-diff rather than asserted against a kind the fixture never produced.
- `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all pass; the determinism + structural tests also pass under `-race`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pipeline orchestration — RED** - `3fbdf1d` (test)
1. **Task 1: Pipeline orchestration — GREEN** - `e5efe8f` (feat)
2. **Task 2: Determinism gate + structural fixture-diff** - `4ef5e4d` (test)

**Plan metadata:** (pending, this commit)

_Note: Task 1 is TDD (RED then GREEN). Task 2 is a test-only property/structural gate against already-implemented code — both tests passed on first run, so no separate RED->GREEN cycle applies; committed as a single `test(02-05):` commit per the plan's own instruction._

## Files Created/Modified
- `internal/indexer/pipeline.go` - `Run`, `run`, `Options`, `Stats`, `resolveFunc`, `readGraphCounts`
- `internal/indexer/pipeline_test.go` - `TestPipelineRun`, `TestPipelineRun_ClosesStoreOnResolveError`, `TestPipelineRun_DefaultsWorkersToNumCPU`
- `internal/indexer/determinism_test.go` - `TestDeterministicRebuild`, `TestRealRepoStructure`, `indexAndExport`, `decodeExportFrames`/`encodeExportFrames`, `reportFirstDiff`, `hasEdge`, `expectedNodeKinds`
- `internal/indexer/testdata/gofixture/pkga/embed.go` - added `type ID = int` to exercise the `type_alias` node kind

## Decisions Made
- Task 1 followed the plan's TDD instruction exactly: RED (`3fbdf1d`) then GREEN (`e5efe8f`).
- Task 2's tests were written to exercise already-shipped Pass 1/2 code (per the plan's own text: "these tests should PASS on first run"); committed as a single `test(02-05):` commit rather than forcing an artificial RED phase.
- Kept the fixture change (`type ID = int`) rather than reverting it — it's additive, doesn't alter any existing fixture symbol, and closes a real coverage gap (the `type_alias` kind was in `expectedNodeKinds` but previously unexercised by the fixture).

## Deviations from Plan

None - plan executed exactly as written. (Task 2's uncommitted work from the interrupted prior executor was reviewed, found complete and correct — including the fixture addition explicitly called for by the plan's `type_alias` note — and committed as-is.)

## Issues Encountered

None. The determinism gate passed on first run under `-race` with `GOMAXPROCS(8)`, confirming Pass 1/2 (02-03/02-04) have no residual ordering nondeterminism to fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `indexer.Run` is the complete pipeline entry point the CLI `index` command drives directly.
- INDX-02 (byte-identical rebuild) and the structural cross-file correctness gate are locked behind automated tests that will fail loudly on any future regression (map/goroutine-order nondeterminism, resolve logic changes).
- Remaining manual verification (per `02-VALIDATION.md`): indexing the `weft-go` corpus repo and spot-checking node kinds/edges against `testdata/golden/corpus/weft-go/` — tracked as coverage item D6 (`human_judgment: true`), not a blocker for subsequent Phase 2 work.
- No blockers for CLI wiring or Phase 3.

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-11*

## Self-Check: PASSED
