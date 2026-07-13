---
phase: 07-migration-tool
plan: 01
subsystem: migration
tags: [modernc.org/sqlite, database/sql, testing-infrastructure, archtest, go/packages]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline
    provides: stable internal/schema record shapes and internal/graphstore Writer API the migrator will write through
  - phase: 01-foundation-storage-schema-parser-strategy
    provides: testdata/golden/ts-schema.sql (TS DDL) and ts-schema.dump.sql (representative dump) captured by the 01-04 capture harness
provides:
  - modernc.org/sqlite v1.53.0 as a direct, confined dependency for the read-only migration reader
  - internal/migrate/migratetest.BuildTSIndex(t, Variant) — in-Go reconstructed real SQLite fixtures (happy/aged/dangling) for every downstream migrate test
  - internal/migrate/archtest.TestModerncSQLiteConfinedToMigrate — enforced confinement boundary
affects: [07-02, 07-03, 07-04, 07-05, 07-06, 07-07]

# Tech tracking
tech-stack:
  added: [modernc.org/sqlite v1.53.0]
  patterns:
    - "go/packages import-graph archtest (mirrors internal/graphstore/archtest's D-04a pebble-confinement pattern) to enforce a dependency-confinement boundary at build/test time, not just by directory convention"
    - "In-Go SQLite fixture reconstruction from committed DDL+dump text (no external sqlite3 binary, no live upstream CLI dependency)"

key-files:
  created:
    - internal/migrate/migratetest/fixture.go
    - internal/migrate/migratetest/fixture_test.go
    - internal/migrate/archtest/modernc_confinement_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "modernc.org/sqlite v1.53.0 added via `go get` then manually promoted from the indirect to the direct require block (no `go mod tidy`, per established project convention of not stripping other deliberately pre-pinned deps)"
  - "ts-schema.sql's CREATE TABLE statements for sqlite_sequence, sqlite_stat1, and the nodes_fts FTS5 shadow tables (nodes_fts_data/idx/docsize/config) must be stripped before Exec — SQLite creates/manages these itself (AUTOINCREMENT, ANALYZE, and the virtual table declaration respectively) and rejects explicit re-creation with 'object name reserved for internal use' / 'already exists' errors"
  - "ts-schema.dump.sql's 5 file:-source contains edges reference import: node ids the representative excerpt does not itself seed as node rows; closeReferentialGaps synthesizes minimal node rows for exactly those ids so VariantHappy is referentially self-consistent and VariantDangling's one deliberately-added bad edge is the only true dangling reference (otherwise the dangling-edge count would already be nonzero on the happy path, making the check meaningless)"
  - "VariantAged must DROP INDEX idx_edges_provenance before ALTER TABLE edges DROP COLUMN provenance — SQLite refuses to drop a column an index still references"

patterns-established:
  - "internal/migrate error-wrap convention: every fixture harness error prefixed 'migratetest: <verb>: %v' via t.Fatalf/t.Errorf, matching the project's fail-loud discipline from prior phases"

requirements-completed: [MIGR-01, MIGR-02]

coverage:
  - id: D1
    description: "modernc.org/sqlite v1.53.0 is a direct go.mod require; module builds with CGO_ENABLED default (pure Go, no C toolchain needed)"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "go build ./..."
        status: pass
    human_judgment: false
  - id: D2
    description: "BuildTSIndex(t, Variant) produces real, readable happy/aged/dangling SQLite .db fixtures with zero external tools"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/migratetest/fixture_test.go#TestBuildTSIndex_Variants"
        status: pass
    human_judgment: false
  - id: D3
    description: "Any modernc.org/sqlite import outside internal/migrate fails the archtest guard"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/archtest/modernc_confinement_test.go#TestModerncSQLiteConfinedToMigrate"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-07-13
status: complete
---

# Phase 7 Plan 1: Migration Tool Wave-0 Infrastructure Summary

**Added modernc.org/sqlite v1.53.0 as a confinement-guarded migration reader dependency and built an in-Go SQLite fixture-reconstruction harness (happy/aged/dangling variants) so every downstream migrate task has a real TS-shaped `.db` to run against.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-07-13T00:00:00Z (approx, session start)
- **Completed:** 2026-07-13T00:22:00Z
- **Tasks:** 3
- **Files modified:** 5 (go.mod, go.sum, fixture.go, fixture_test.go, modernc_confinement_test.go)

## Accomplishments
- `modernc.org/sqlite v1.53.0` added as a direct require, confined to `internal/migrate` (and subpackages) by both convention and an enforced archtest guard
- `internal/migrate/migratetest.BuildTSIndex(t, Variant)` reconstructs a real, queryable SQLite database entirely in-Go from the committed `testdata/golden/ts-schema.sql` DDL + `ts-schema.dump.sql` seed rows — no `sqlite3` binary, no live TS CodeGraph CLI dependency
- Three fixture variants proven by self-test: `VariantHappy` (full schema/rows, file:-source `contains` edges, `unistr()`-decoded docstrings), `VariantAged` (drops `nodes.return_type`/`edges.provenance` to simulate a pre-column-addition index), `VariantDangling` (one edge with a non-existent target — the corruption fixture)
- `internal/migrate/archtest.TestModerncSQLiteConfinedToMigrate` enforces the read-only-reader confinement boundary via `go/packages` import-graph inspection (mirrors `internal/graphstore/archtest`'s established D-04a pattern); manually verified it actually fails against a real outside importer, not just vacuously passes

## Task Commits

Each task was committed atomically:

1. **Task 1: Add modernc.org/sqlite v1.53.0** - `c581efc` (chore)
2. **Task 2: In-Go fixture-reconstruction harness** - `ec0275b` (test)
3. **Task 3: Architecture guard confining modernc.org/sqlite** - `0e106f6` (test)

_No plan-metadata commit needed beyond this SUMMARY/STATE/ROADMAP commit (below)._

## Files Created/Modified
- `go.mod` / `go.sum` - `modernc.org/sqlite v1.53.0` added as direct require (manually promoted from `go get`'s indirect placement, no `go mod tidy`)
- `internal/migrate/migratetest/fixture.go` - `BuildTSIndex`, `Variant`/`VariantHappy`/`VariantAged`/`VariantDangling`, DDL sanitization, referential-gap closure, aging, dangling-edge injection, repo-root discovery
- `internal/migrate/migratetest/fixture_test.go` - `TestBuildTSIndex_Variants` self-test asserting each variant's distinguishing property
- `internal/migrate/archtest/modernc_confinement_test.go` - `TestModerncSQLiteConfinedToMigrate` confinement guard

## Decisions Made
- Reconstructed the fixture entirely via `database/sql` + the same `modernc.org/sqlite` driver the real migrator will use, rather than shelling out to `sqlite3` — matches RESEARCH's recommendation and keeps CI hermetic (no external tool dependency, no live TS CLI dependency which the research flagged as possibly gone)
- Discovered and documented two fixture-reconstruction-specific gotchas not called out explicitly in RESEARCH/PATTERNS (both fixed inline, both scoped to the test harness only, neither affects the real migration reader's design): (1) `ts-schema.sql`'s captured-via-`.schema` explicit `CREATE TABLE` statements for `sqlite_sequence`/`sqlite_stat1`/FTS5 shadow tables conflict with SQLite's own implicit creation of those same tables; (2) the dump's `file:`-source `contains` edges reference `import:` node ids the representative excerpt doesn't itself seed, requiring a `closeReferentialGaps` synthesis step so `VariantHappy` is referentially self-consistent and `VariantDangling`'s single injected bad edge is unambiguous

## Deviations from Plan

None — plan executed as written. The two fixture-reconstruction gotchas above were implementation details necessary to make the plan's own described harness work correctly (Rule 1/Rule 3 auto-fixes: the fixture wouldn't build/wouldn't produce a meaningful dangling-edge signal without them), not scope changes — both are internal to `fixture.go` and don't touch the plan's declared artifacts, files, or acceptance criteria.

## Issues Encountered
- `db.Exec` of the raw `ts-schema.sql` DDL initially failed with `SQL logic error: object name reserved for internal use: sqlite_sequence` and, after stripping that, `table 'nodes_fts_data' already exists` — resolved by identifying that `.schema`-style dumps include auto-generated internal/shadow table definitions that must be filtered before replay (see Decisions Made above). Verified via a throwaway smoke-test program before committing the real implementation.
- `VariantDangling`'s self-test initially reported 6 dangling edges instead of 1 — traced to the dump's 5 seeded `contains` edges pointing at `import:` ids not present among the dump's 5 seeded nodes (the dump is a small representative excerpt, not an exhaustive graph). Resolved with `closeReferentialGaps`.
- `ALTER TABLE edges DROP COLUMN provenance` initially failed with `error in index idx_edges_provenance after drop column` — resolved by dropping the index first.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/migrate/migratetest.BuildTSIndex` is ready for 07-02 (reader/translate) and every subsequent migrate plan to build real, queryable TS-shaped fixtures without any external tooling
- The confinement archtest will immediately flag any future plan that accidentally imports `modernc.org/sqlite` from outside `internal/migrate`
- No blockers for 07-02 (field-mapping/translation) or later plans in this phase

---
*Phase: 07-migration-tool*
*Completed: 2026-07-13*

## Self-Check: PASSED

All created files and task commit hashes verified present on disk / in git log.
