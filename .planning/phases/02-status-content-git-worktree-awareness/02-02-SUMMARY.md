---
phase: 02-status-content-git-worktree-awareness
plan: 02
subsystem: query
tags: [status, pebble, dbSizeBytes, filesByLanguage, golden-parity, tdd]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: internal/query read seam (Engine, MarshalStatusJSON, StatusResult decision-table convention)
provides:
  - StatusResult.DbSizeBytes int64 (json "dbSizeBytes") — best-effort Pebble store-dir byte sum (STAT-01)
  - StatusResult.FilesByLanguage map[string]int64 (json "-", internal-only) — per-language file counts feeding future renderers (STAT-02)
  - Languages now derived from FilesByLanguage (count > 0, sorted) instead of a separate node-scan languageSet
  - dbSizeBytes(storeDir) unexported helper — best-effort filepath.WalkDir byte sum, root-missing surfaces an error, per-entry errors deeper in the tree are skipped
  - golden_parity_test.go findVolatileKeysExcept — narrow named exemption for dbSizeBytes at the "our own output" call site only
affects: [02-05, 02-06, 02-07]

tech-stack:
  added: []
  patterns:
    - "Best-effort filepath.WalkDir sum mirroring newestSourceMtime's shape, with one refinement: root-stat failure propagates as the helper's own error (caller degrades), per-entry failures deeper in the walk are skipped (never abort)"
    - "FilesByLanguage computed inside the pre-existing IterateFiles() scan (one field read, fileIt.File().Language) rather than a second store scan"
    - "Golden-oracle volatility exemption applied at the call site (findVolatileKeysExcept), never by mutating the shared volatileKeys map that governs frozen TS fixtures"

key-files:
  created: []
  modified:
    - internal/query/status.go
    - internal/query/files_status_test.go
    - testdata/golden/golden_parity_test.go
    - testdata/golden/README.md

key-decisions:
  - "dbSizeBytes' WalkDir callback distinguishes root-level failure (missing/unreadable storeDir itself, propagated as the function's own error) from per-entry failure deeper in the tree (e.g. an SSTable vanishing mid-walk during live Pebble compaction, silently skipped) — a small refinement over RESEARCH's literal code sample (which swallowed all errors unconditionally), needed so the RED test's 'dbSizeBytes(nonexistent) returns (0, err)' contract and Status()'s own degrade-to-0-without-erroring contract can both hold simultaneously"
  - "Languages re-derivation from FilesByLanguage (D-05) produced NO value-set change on the weft golden corpus — TestGoldenParity/status's wantLanguages assertion (go/javascript/python) still passes unmodified, so no file in that corpus is stored with zero extracted nodes under a language no prior file already contributed"
  - "findVolatileKeysExcept lives in golden_parity_test.go (this plan's own file), not golden_test.go — the shared volatileKeys map and TestGoldenFixturesExist stay byte-identical (git diff confirmed empty), continuing to guard the frozen TS oracle fixtures"

requirements-completed: [STAT-01, STAT-02]

coverage:
  - id: D1
    description: "status --json emits dbSizeBytes as a positive integer computed from the real Pebble store directory"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestStatus/dbSizeBytes_is_present_and_plausible"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestDbSizeBytes"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity/status (plausibility + MB-shape assertions)"
        status: pass
      - kind: manual
        ref: "codegraph status --json --path . | python3 -c \"assert d['dbSizeBytes']>0 and 'filesByLanguage' not in d\" — observed dbSizeBytes=1237966 on this repo's own real index"
        status: pass
    human_judgment: false
  - id: D2
    description: "StatusResult.FilesByLanguage carries real per-language file counts computed in the existing file scan; Languages derives from it"
    requirement: STAT-02
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestStatus/filesByLanguage_counts_files_per_language,_languages_derived_from_it"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestStatus/filesByLanguage_is_internal-only_and_absent_from_the_JSON_shape"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity/status (wantLanguages unchanged)"
        status: pass
    human_judgment: false

duration: 28min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 2: Status content — dbSizeBytes + FilesByLanguage Summary

**Added Pebble on-disk `dbSizeBytes` (best-effort `filepath.WalkDir` byte sum) and a genuinely new `FilesByLanguage` aggregation to `StatusResult`, with `Languages` re-derived from it; reversed the golden-corpus `dbSizeBytes` strip for Go's own output only, via a narrowly-scoped exemption that leaves the shared TS-oracle volatility map untouched.**

## Performance

- **Duration:** 28 min
- **Tasks:** 3 (TDD RED / GREEN / golden exemption)
- **Files modified:** 4

## Accomplishments

- `StatusResult.DbSizeBytes int64` (`json:"dbSizeBytes"`) — real Pebble store-directory byte sum via a best-effort `filepath.WalkDir`, mirroring `newestSourceMtime`'s shape. Degrades to 0 (never errors `Status()`) when the Engine has no `repoRoot` (`New`, not `OpenAt`) or the store dir is missing/unreadable (STAT-01).
- `StatusResult.FilesByLanguage map[string]int64` (`json:"-"`) — computed inside the pre-existing `IterateFiles()` scan (one field read, `fileIt.File().Language`), not a second store scan. Deliberately excluded from `--json` output, matching TS's own behavior of deriving `languages` from this map and discarding the counts (STAT-02, D-05).
- `Languages []string` now derived from `FilesByLanguage` (`count > 0`, sorted) instead of a separate node-scan `languageSet` — reflects files the indexer stored, including any that yield zero extracted nodes.
- Golden-parity oracle updated: `dbSizeBytes` is exempted from the "our own output must have no volatile fields" check at exactly one call site (`findVolatileKeysExcept`), replaced by presence/integer/`>0`/MB-shape plausibility assertions. The shared `volatileKeys` map governing the frozen TS oracle fixtures is untouched (`git diff testdata/golden/golden_test.go` confirmed empty).
- `testdata/golden/README.md`'s volatile-fields table records the Go-side reversal and its rationale.

## Task Commits

1. **Task 1: RED — DbSizeBytes + FilesByLanguage assertions** — `92e5e85` (test)
2. **Task 2: GREEN — dbSizeBytes walk + FilesByLanguage aggregation + decision-table rows** — `f93403d` (feat)
3. **Task 3: Golden volatility exemption for dbSizeBytes + README amendment** — `d3ea0dd` (test)

No REFACTOR commit was needed.

## Files Created/Modified

- `internal/query/status.go` — `StatusResult.DbSizeBytes`/`FilesByLanguage` fields, `dbSizeBytes()` helper, `Status()` extended to fill both, decision-table doc comment gains two rows and corrects the stale "no dbSizeBytes key" row
- `internal/query/files_status_test.go` — new `TestStatus` subtests for both fields' presence/shape/JSON-exclusion contracts, new `TestDbSizeBytes` for the degrade-safely contracts, existing "no volatile keys" subtest updated to stop forbidding `dbSizeBytes`
- `testdata/golden/golden_parity_test.go` — `findVolatileKeysExcept` + `mbShapeRE`, status subtest gains the D-08 plausibility assertions and an explanatory comment on the golden-vs-our-output asymmetry
- `testdata/golden/README.md` — volatile-fields table's `dbSizeBytes` row amended to record the Go-side reversal

## Final `--json` Key Set

Verified end-to-end against this repo's own real index (`codegraph status --json --path .`):

```
initialized, version, projectPath, indexPath, fileCount, nodeCount, edgeCount,
dbSizeBytes, backend, nodesByKind, languages, pendingChanges, worktreeMismatch,
stale, index
```

`filesByLanguage` is confirmed absent (D-05, `json:"-"`). Observed `dbSizeBytes` on this repo's own index: `1237966` bytes (~1.18 MB). On the gofixture test corpus (4 Go files): `8409` bytes, `FilesByLanguage={"go":4}`.

## `TestGoldenParity/status` `wantLanguages` — unchanged

The regression watch item from Task 2's acceptance criteria: switching `Languages`' derivation from node-scan `languageSet` to file-scan `FilesByLanguage` **did not** change the weft corpus's expected value set. `wantLanguages = []string{"go", "javascript", "python"}` still matches exactly — no file in the weft corpus is stored with a language that had zero prior node-derived representation. `TestGoldenParity/status` passes with no assertion changes needed for this key.

## Decisions Made

- `dbSizeBytes`'s `WalkDir` callback distinguishes a root-level walk failure (surfaced as the function's own error, so `dbSizeBytes(nonexistentDir)` returns `(0, err)` as the RED test requires) from a per-entry failure deeper in the tree (silently skipped, walk continues) — a small, deliberate refinement over RESEARCH's Pattern 2 code sample (which swallowed all `WalkDir` errors unconditionally, including the root-missing case). The literal sample would have made `Status()`'s own degrade-to-0 contract correct but broken the RED test's explicit "(0, err)" requirement for the direct helper call; this refinement satisfies both simultaneously and still matches the threat register's T-02-07 disposition ("a failed walk leaves DbSizeBytes at 0 without erroring Status()").
- `findVolatileKeysExcept` lives in `golden_parity_test.go` (this plan's file), not `golden_test.go` — confirmed via `git diff --stat testdata/golden/golden_test.go go.mod go.sum` returning empty, so the shared `volatileKeys` map and `TestGoldenFixturesExist` continue to guard the frozen TS oracle fixtures unmodified.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] RESEARCH's literal `dbSizeBytes` code sample would have broken the RED test's error-return contract**
- **Found during:** Task 2 (GREEN)
- **Issue:** RESEARCH's Pattern 2 code sample returns `nil` unconditionally on any `WalkDir` callback error (including the root-directory-missing case), which would make `dbSizeBytes(nonexistentDir)` return `(0, nil)` — contradicting Task 1's RED test asserting `(0, err)` for a nonexistent directory, and contradicting the plan's own Test 7 acceptance language ("returns (0, err)").
- **Fix:** Added a `p == storeDir` check in the callback: root-level failures propagate as the helper's own error; only per-entry failures deeper in the tree are skipped. `Status()` still swallows the error and degrades `DbSizeBytes` to 0, matching T-02-07's disposition.
- **Files modified:** `internal/query/status.go`
- **Commit:** `f93403d`

### None Other

All other plan instructions were followed exactly as written, including the field ordering, `json:"-"` tag, decision-table row format, and the golden-exemption call-site scoping.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `StatusResult.DbSizeBytes`/`FilesByLanguage` are ready for wave 2's CLI (`RenderStatus`, D-09) and MCP (`RenderStatusMarkdown`, D-17) renderers — both fields are live, tested, and documented in the decision table.
- `MarshalStatusJSON` itself was never touched (only struct fields added) — the CLI `--json` contract and golden oracle remain intact.
- Zero new dependencies; `go.mod`/`go.sum` are byte-identical to before this plan (confirmed via the same `git diff --stat` check used for `golden_test.go`).

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*

## Self-Check: PASSED

All 5 referenced files verified present on disk; all 3 task commits (92e5e85, f93403d, d3ea0dd) verified present in git log.
