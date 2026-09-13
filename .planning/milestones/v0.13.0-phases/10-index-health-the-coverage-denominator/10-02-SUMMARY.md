---
phase: 10-index-health-the-coverage-denominator
plan: 02
subsystem: indexer
tags: [discovery, walkdir, exclusion-reasons, tdd, fixture]

# Dependency graph
requires:
  - phase: 10-index-health-the-coverage-denominator
    provides: "Plan 01's DiscoverAll/Discovery, discoverexclusion.go, newExcludedFile/buildTagDetail, the c/ graphstore namespace, Engine.CoverageSummary/CoverageRows — this plan extends the same walk and read-back path"
provides:
  - "internal/indexer/discoverexclusion.go: dirExclusionReason, exceedsSizeLimit, unsupportedExtensionDetail, sizeLimitDetail — pure, unit-tested classification helpers for the three remaining exclusion reasons plus the size pre-check"
  - "DiscoverAll now records all four pre-extraction exclusion reasons (DIR_VENDOR, DIR_DOTPREFIX, UNSUPPORTED_EXTENSION, BUILD_TAG, SIZE_LIMIT) from the SAME walk, in the decision-point order research verified"
  - "internal/indexer/testdata/coverage/: the committed D-13 fixture module (go.mod, main.go, vendor/x.go, .hidden/y.go, notes.md, tagged.go) — one file of every discovery-time exclusion kind"
  - "internal/indexer/coverage_fixture_test.go: the verified dangling-symlink extraction-failure technique, the exact-counts end-to-end test, and the D-14a disk-mutation-without-reindex test"
affects: [10-03-sync-diff-and-prune, 10-04-health-page-coverage-section, 10-05-coverage-rows-pagination-and-security, 10-06-mutation-log-and-security-doc]

# Actuals (#2632)
actuals:
  tokens: 11980
  tasks: 2
  commits: 4
plan_head_before: d94a7e70

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "dirExclusionReason never calls ShouldSkipDir (and vice versa is asserted, not assumed) — the walk keeps ShouldSkipDir as the sole prune authority shared with the fsnotify watcher, while a separate, independently-tested helper only classifies WHICH reason a prune gets. The agreement test (TestDirExclusionReasonAgreesWithShouldSkipDir) is the guard against the two silently diverging, not a tautological self-check."
    - "Fixtures that index into repoRoot/.codegraph/store must never pre-create that directory before calling indexer.Run — DiscoverAll's walk runs BEFORE graphstore.Open creates the store dir on a genuine first index, and pre-creating it makes the walk record a phantom DIR_DOTPREFIX exclusion for \".codegraph\" that production never sees on a fresh repo."

key-files:
  created:
    - internal/indexer/coverage_fixture_test.go
    - internal/indexer/testdata/coverage/go.mod
    - internal/indexer/testdata/coverage/main.go
    - internal/indexer/testdata/coverage/vendor/x.go
    - internal/indexer/testdata/coverage/.hidden/y.go
    - internal/indexer/testdata/coverage/notes.md
    - internal/indexer/testdata/coverage/tagged.go
  modified:
    - internal/indexer/discoverexclusion.go
    - internal/indexer/discover.go
    - internal/indexer/discoverexclusion_test.go
    - internal/indexer/discover_test.go
    - internal/query/coverage_test.go
    - internal/uiserver/coverage_test.go

key-decisions:
  - "Assumption A1 (the dangling-symlink extraction-failure technique) VERIFIED on first probe run (darwin/arm64) — no fallback (chmod-unreadable file) was needed."
  - "sizeLimitDetail's exact string format ('<n> bytes > 4194304') is Claude's own invention (no wire/UI contract pins it yet) — documented inline so a later plan formatting the coverage page can rely on it."
  - "internal/query and internal/uiserver's coverage_test.go fixtures were fixed to stop pre-creating the store directory before indexer.Run (see Deviations) — the correct, production-faithful fixture shape this plan's Task 2 fixture also follows (store OUTSIDE the copy, per the plan's own instruction)."

patterns-established:
  - "A coverage-index test fixture must either (a) let indexer.Run's own graphstore.Open create the store directory after the walk (matches a real first index), or (b) place the store entirely outside the tree being walked. Pre-creating repoRoot/.codegraph/store before Run is the one shape that silently corrupts exclusion counts — avoid it in every future coverage fixture."

requirements-completed: [HLT-04, HLT-05]

coverage:
  - id: D1
    description: "All four pre-extraction exclusion reasons (DIR_VENDOR, DIR_DOTPREFIX, UNSUPPORTED_EXTENSION, BUILD_TAG, SIZE_LIMIT) are decided inside DiscoverAll's single WalkDir pass, with the SIZE_LIMIT boundary strict at parser.MaxSourceBytes"
    requirement: "HLT-05"
    verification:
      - kind: unit
        ref: "internal/indexer/discoverexclusion_test.go#TestDiscoverAll_RecordsAllFourReasons"
        status: pass
      - kind: unit
        ref: "internal/indexer/discoverexclusion_test.go#TestExceedsSizeLimitIsStrictlyGreaterThan"
        status: pass
      - kind: unit
        ref: "internal/indexer/discoverexclusion_test.go#TestDiscoverAll_ExactSizeLimitFileIsNotExcluded"
        status: pass
      - kind: unit
        ref: "internal/indexer/discoverexclusion_test.go#TestDirExclusionReasonAgreesWithShouldSkipDir"
        status: pass
    human_judgment: false
  - id: D2
    description: "A pruned directory (vendor/, dot-prefixed) yields exactly ONE directory-level record, never phantom per-file rows for its contents"
    requirement: "HLT-05"
    verification:
      - kind: unit
        ref: "internal/indexer/discoverexclusion_test.go#TestDiscoverAll_RecordsAllFourReasons"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscoverAll_SkipsVendorAndDotDirsWithOneRecordEach"
        status: pass
    human_judgment: false
  - id: D3
    description: "Against the committed testdata/coverage/ fixture plus its two test-time-generated files, indexer.Run yields exactly indexed=1, extraction_failed=1, excluded=6, file_level=4, discovered=6 — counts reported by the test, not implied"
    requirement: "HLT-04"
    verification:
      - kind: integration
        ref: "internal/indexer/coverage_fixture_test.go#TestCoverageFixtureCountsAndReasons"
        status: pass
    human_judgment: false
  - id: D4
    description: "The dangling-.py-symlink extraction-failure technique is verified (not assumed): DiscoverAll discovers it, Extract fails it with a wrapped fs.ErrNotExist"
    verification:
      - kind: unit
        ref: "internal/indexer/coverage_fixture_test.go#TestDanglingSymlinkIsDiscoveredAndFailsExtraction"
        status: pass
    human_judgment: false
  - id: D5
    description: "Exclusion reasons survive a disk mutation without re-indexing (D-14a) — the store's stale reasons for tagged.go/notes.md persist while a fresh DiscoverAll of the same mutated disk provably disagrees"
    requirement: "HLT-05"
    verification:
      - kind: integration
        ref: "internal/indexer/coverage_fixture_test.go#TestCoverageReasonsSurviveDiskMutationWithoutReindex"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-09-13
status: complete
---

# Phase 10 Plan 2: All Four Exclusion Reasons + the Committed Coverage Fixture Summary

DiscoverAll now decides all four pre-extraction exclusion reasons (DIR_VENDOR, DIR_DOTPREFIX, UNSUPPORTED_EXTENSION, BUILD_TAG, SIZE_LIMIT) in one walk, proven end-to-end against a committed fixture module with exact counts and a disk-mutation-without-reindex guard.

## Performance

- **Duration:** ~50 min (not precisely timestamped at session start; approximate)
- **Completed:** 2026-09-13T02:18:28Z
- **Tasks:** 2 (both `tdd="true"`)
- **Files modified:** 13 (7 created, 6 modified)

## Accomplishments

- `discoverexclusion.go` gained four pure, independently-tested helpers (`dirExclusionReason`, `exceedsSizeLimit`, `unsupportedExtensionDetail`, `sizeLimitDetail`) and `DiscoverAll`'s walk now wires all four decision points in the order research verified — directory prune (1) → extension miss (2) → build-tag miss (3) → size pre-check (4) — with `ShouldSkipDir` kept as the sole prune authority the fsnotify watcher also shares
- HLT-05's boundary is pinned exactly: a file of `parser.MaxSourceBytes` bytes is discovered normally; `+1` byte is `SIZE_LIMIT`, bytes never read
- The `internal/indexer/testdata/coverage/` fixture module is committed — exactly 6 tracked files, no symlink, no oversize file (both generated per-test in a temp copy)
- Assumption A1 (dangling `.py` symlink as an extraction-failure trigger) is now a VERIFIED technique, not an `[ASSUMED]` one — the probe passed on its first run
- `TestCoverageFixtureCountsAndReasons` proves the full pipeline end-to-end: `indexed=1 extraction_failed=1 excluded=6 file_level=4 discovered=6`, reported by the test itself, not implied
- `TestCoverageReasonsSurviveDiskMutationWithoutReindex` proves D-14a: stripping `tagged.go`'s build tag and deleting `notes.md` on disk, without re-indexing, leaves the store's reported reasons unchanged — while a fresh `DiscoverAll` of the same mutated disk provably disagrees

## Task Commits

1. **Task 1: RED — failing tests for the other three exclusion reasons and the size pre-check** - `1a45a5d6` (test)
2. **Task 1: GREEN — wire DIR_VENDOR, DIR_DOTPREFIX, UNSUPPORTED_EXTENSION, SIZE_LIMIT into DiscoverAll** - `90e4ff0f` (feat, includes the Rule-1 fixture-ordering fix to two Plan 01 precedent test files)
3. **Task 2: RED — coverage fixture proving tests + verified symlink probe** - `fdde2cb5` (test)
4. **Task 2: GREEN — commit the coverage fixture module** - `bc10004d` (feat)

_TDD tasks: 2 commits each (test → feat); no REFACTOR commit needed for either task._

## TDD Gate Compliance

| Unit | RED commit | GREEN commit | REFACTOR | Status |
|---|---|---|---|---|
| Task 1 — four decision points + size pre-check | `1a45a5d6` | `90e4ff0f` | none needed | Pass |
| Task 2 — fixture + proving tests | `fdde2cb5` | `bc10004d` | none needed | Pass |

GSD's TAP-only `tdd-red-evidence` check was not run — this repo's own `<verify>` gates (Go-native, with `fails_when`) are the authority here, per the same commit-discipline note Plan 01 recorded. Every RED phase below was verified INTENTIONAL before its GREEN commit landed.

### RED transcript — Task 1 (`discoverexclusion_test.go`, `discover_test.go` new tests, against Plan-01-only implementation)

```
internal/indexer/discoverexclusion_test.go:135:17: undefined: dirExclusionReason
internal/indexer/discoverexclusion_test.go:158:5: undefined: exceedsSizeLimit
internal/indexer/discoverexclusion_test.go:161:5: undefined: exceedsSizeLimit
internal/indexer/discoverexclusion_test.go:164:6: undefined: exceedsSizeLimit
internal/indexer/discoverexclusion_test.go:167:5: undefined: exceedsSizeLimit
FAIL	github.com/seanb4t/codegraph-go/internal/indexer [build failed]
```

Captured by temporarily reverting `discover.go`/`discoverexclusion.go` to their Plan-01 state (git checkout) with only the new test file staged, running the named-test gate, then restoring the Task 1 implementation. `RED_EVIDENCE_OK`-equivalent: the target tests (`dirExclusionReason`, `exceedsSizeLimit` undefined) are exactly the symbols Task 1 was about to add — not an unrelated build break.

### GREEN transcript — Task 1

```
    discoverexclusion_test.go:152: dirExclusionReason/ShouldSkipDir agreement rows checked: 7
--- PASS: TestDirExclusionReasonAgreesWithShouldSkipDir (0.00s)
--- PASS: TestExceedsSizeLimitIsStrictlyGreaterThan (0.00s)
    discoverexclusion_test.go:213: excluded records: 7
--- PASS: TestDiscoverAll_RecordsAllFourReasons (0.00s)
--- PASS: TestDiscoverAll_ExactSizeLimitFileIsNotExcluded (0.00s)
--- PASS: TestDiscoverAll_ExcludedSortedAndDeterministic (0.00s)
--- PASS: TestDiscover_SkipsVendorAndDotDirs (0.00s)
--- PASS: TestDiscover_MixedLanguage_ExtensionRegistry (0.00s)
--- PASS: TestExtractPool_OversizedFileContained (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/indexer	0.370s
```

8/8 named PASS, 0 FAIL, no vacuous run, `excluded records: 7` logged (the positive count). `task test:golden` green afterward (`ok testdata/golden 23.172s`, goldens unmoved).

### RED transcript — Task 2 (`coverage_fixture_test.go`, before the fixture module was committed)

```
coverage_fixture_test.go:189: excluded records: 3
    coverage_fixture_test.go:191: Excluded rows = [{path:vendor ...} {path:.hidden ...} {path:huge.go ...}], want exactly 6
--- FAIL: TestCoverageFixtureCountsAndReasons (0.10s)
FAIL	github.com/seanb4t/codegraph-go/internal/indexer	0.452s
```

Genuine RED: only the two test-time-generated files (`huge.go`, `broken.py`) plus the pre-existing empty `vendor/`/`.hidden/` directories existed on disk; `go.mod`/`main.go`/`notes.md`/`tagged.go` were the next commit.

### GREEN transcript — Task 2

```
coverage_fixture_test.go:189: excluded records: 6
    coverage_fixture_test.go:257: indexed=1 extraction_failed=1 excluded=6 file_level=4 discovered=6
--- PASS: TestCoverageFixtureCountsAndReasons (0.11s)
--- PASS: TestCoverageReasonsSurviveDiskMutationWithoutReindex (0.09s)
ok  	github.com/seanb4t/codegraph-go/internal/indexer	0.558s
```

### Probe verdict (Assumption A1)

```
--- PASS: TestDanglingSymlinkIsDiscoveredAndFailsExtraction (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/indexer	0.356s
```

PASSED on the first run (darwin/arm64) — `DiscoverAll` returns the dangling `.py` symlink as a `DiscoveredFile` (no walk error; `d.Info()` is lstat-based and Go's build-tag `MatchFile` is never reached for a non-Go extension), and `Extract` records `FileResult.Err` wrapping `fs.ErrNotExist`. The `[ASSUMED]` tag in 10-RESEARCH.md Pitfall 4 is resolved: **confirmed valid**. The chmod-0o000-unreadable-file fallback was not needed.

## Files Created/Modified

- `internal/indexer/discoverexclusion.go` — `dirExclusionReason`, `exceedsSizeLimit`, `unsupportedExtensionDetail`, `sizeLimitDetail`; the walk callback now decides all four reasons
- `internal/indexer/discover.go` — `Discover`'s doc comment now states it returns the extraction-bound subset of `DiscoverAll`
- `internal/indexer/discoverexclusion_test.go`, `internal/indexer/discover_test.go` — new/updated tests (see Task Commits)
- `internal/indexer/coverage_fixture_test.go` — the fixture-proving tests and the symlink probe
- `internal/indexer/testdata/coverage/` — the committed 6-file fixture module
- `internal/query/coverage_test.go`, `internal/uiserver/coverage_test.go` — Rule 1 fix to `indexCoverageFixture`/`uiserverBuildTagFixture` (see Deviations)

## Decisions Made

See `key-decisions` in frontmatter. In short: Assumption A1 verified (no fallback needed); `sizeLimitDetail`'s format is a new, undocumented-elsewhere convention; two Plan 01 precedent test fixtures were corrected rather than the new decision points weakened.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan 01's `indexCoverageFixture`/`uiserverBuildTagFixture` pre-created the store directory before indexing, producing a phantom DIR_DOTPREFIX exclusion once decision point 1 existed**
- **Found during:** Task 1's full-package verify gate (`go test ./internal/indexer/... ./internal/query/... ./internal/uiserver/...`)
- **Issue:** `internal/query/coverage_test.go`'s `indexCoverageFixture` and `internal/uiserver/coverage_test.go`'s `uiserverBuildTagFixture` both called `os.MkdirAll(storeDir, ...)` (`storeDir = root/.codegraph/store`) BEFORE calling `indexer.Run`. Before this plan, `DiscoverAll` only ever recorded `BUILD_TAG`, so a directory-level exclusion for `.codegraph` was structurally impossible and this latent issue was invisible. Once decision point 1 (directory pruning) was wired, the pre-created `.codegraph` directory was itself walked and recorded as `DIR_DOTPREFIX` — a record a genuine from-scratch index never produces, since `indexer.Run`'s own `DiscoverAll` walk runs BEFORE `graphstore.Open` creates the store directory. This broke `TestCoverageSummaryAfterRunOnBuildTagRepo`'s invariant assertion (`Discovered != Indexed+ExtractionFailed+Excluded`, since `Excluded` now included one directory-level record its formula didn't account for) and made `TestCoverageRowsUnknownGraphAndSingleRow`/`TestGetHealthCarriesCoverageForBuildTagExclusion` see 3 rows instead of the expected 1.
- **Fix:** Removed the premature `os.MkdirAll` calls in both fixture helpers, letting `indexer.Run`'s own `graphstore.Open` create the store directory AFTER the walk — exactly mirroring a real first-time index. This is a fixture-correctness fix, not a production-code change: `internal/query/coverage.go`'s own `Discovered = indexed + extractionFailed + fileLevel` math (excluding directory-level records) was already correct per Plan 01's Pitfall-1 mitigation.
- **Also fixed:** both tests' row-count/name assertions to expect the now-permanently-present `go.mod` `UNSUPPORTED_EXTENSION` record (`.mod` was never a registered extension; decision point 2 now correctly records it) alongside `tagged.go`'s `BUILD_TAG` record.
- **Files modified:** `internal/query/coverage_test.go`, `internal/uiserver/coverage_test.go`
- **Verification:** `TestCoverageSummaryAfterRunOnBuildTagRepo`, `TestCoverageRowsUnknownGraphAndSingleRow`, `TestGetHealthCarriesCoverageForBuildTagExclusion` all PASS; full `internal/indexer`/`internal/query`/`internal/uiserver` suites green; `task test:golden` green.
- **Committed in:** `90e4ff0f` (part of Task 1's GREEN commit, since the fix was a direct, unavoidable consequence of Task 1's own decision-point-1 wiring).

---

**Total deviations:** 1 auto-fixed (Rule 1 — a pre-existing test-fixture ordering bug, latent until this plan's new decision points exposed it). **Impact:** test-fixture-only; no production code (`internal/query/coverage.go`, `internal/uiserver/handlers.go`) was touched or needed touching — its counting logic already correctly excluded directory-level records from the `Discovered` invariant.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All four exclusion reasons are now live in `DiscoverAll`, proven end-to-end against a committed fixture with exact, test-reported counts and a disk-mutation-without-reindex guard (D-14a's behavioural half).
- Plan 03 can now build `Sync`'s real upsert/prune diff against the `c/` namespace using the same `DiscoverAll` result this plan hardened, with all four reasons available (not just `BUILD_TAG`).
- Plan 04 (health page) has the full backend denominator/reason surface to render against.
- D-14's structural half (the file-scoped source scan proving `internal/query/coverage.go` never walks disk, `TestCoverageSourceNeverWalksDisk`) already existed from Plan 01 and remains untouched/green — not this plan's job.
- No blockers.

## Self-Check: PASSED

All key files present on disk: `internal/indexer/discoverexclusion.go`, `internal/indexer/discover.go`, `internal/indexer/discoverexclusion_test.go`, `internal/indexer/discover_test.go`, `internal/indexer/coverage_fixture_test.go`, `internal/indexer/testdata/coverage/{go.mod,main.go,vendor/x.go,.hidden/y.go,notes.md,tagged.go}`, `internal/query/coverage_test.go`, `internal/uiserver/coverage_test.go`. All four commits (`1a45a5d6`, `90e4ff0f`, `fdde2cb5`, `bc10004d`) found in `git log`.

---
*Phase: 10-index-health-the-coverage-denominator*
*Completed: 2026-09-13*
