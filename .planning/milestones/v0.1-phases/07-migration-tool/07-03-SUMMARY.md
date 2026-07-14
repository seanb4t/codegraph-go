---
phase: 07-migration-tool
plan: 03
subsystem: migration
tags: [database/sql, modernc.org/sqlite, sqlite, field-mapping, translation]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 01)
    provides: modernc.org/sqlite v1.53.0 dependency + migratetest.BuildTSIndex(t, Variant) fixture harness (happy/aged/dangling), archtest confinement guard
  - phase: 07-migration-tool (plan 02)
    provides: Writer.PutMigration/Reader.GetMigration meta-record pair (unused by this plan directly; establishes the resumability write door for a later plan)
provides:
  - internal/migrate.OpenSource/DetectTS/SchemaVersion/ScanTable/CountRows/CountDistinctEdges/FindDBFile — the read-only migration reader
  - internal/migrate.ErrNotATSSource / ErrUnsupportedSchemaVersion sentinels
  - internal/migrate's private nodeFromRow/edgeFromRow/fileFromRow/flattenMetadata/msToNs/normalizeFilePath/parseErrorsJSON translation layer (row -> schema.Node/Edge/File)
affects: [07-04, 07-05, 07-06, 07-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PRAGMA table_info-driven defensive column SELECT: build the query from the intersection of wanted-columns ∩ present-columns per table, so an aged source missing a later-added column reads instead of crashing"
    - "rows.Err() checked after every read loop (mirrors internal/graphstore/export.go's iter.Error() idiom) — the package-wide fail-loud discipline for internal/migrate"
    - "map[string]any row -> proto record translation, keyed by column name, so a missing map key (aged-tolerant SELECT) reads as the Go zero value rather than requiring per-table struct variants"

key-files:
  created:
    - internal/migrate/reader.go
    - internal/migrate/reader_test.go
    - internal/migrate/translate.go
    - internal/migrate/translate_test.go
  modified: []

key-decisions:
  - "D-05 start_col/end_col correction (from RESEARCH, overriding CONTEXT's literal wording) implemented as CARRIED: nodes.start_column/end_column map to Node.StartCol(9)/EndCol(10); only is_async/is_static/is_abstract/decorators/type_parameters are dropped"
  - "msToNs branches on the concrete driver type (int64 direct multiply vs float64 fractional multiply) instead of always round-tripping through float64, to avoid float64's ~2^53 integer-precision ceiling silently corrupting a large integer-ms*1e6 nanosecond value"
  - "rows.Err() truncated-read test simulated via a mid-file-truncated fixture copy (valid SQLite header, corrupted b-tree pages) rather than closing the DB handle mid-scan — database/sql keeps a connection alive for an in-flight Rows even after DB.Close(), so the close-mid-scan approach from the plan's example did not actually reproduce a failure; the truncated-file approach does"

patterns-established:
  - "internal/migrate error-wrap convention confirmed and extended to reader.go/translate.go: every error prefixed 'migrate: <verb>: %w'"

requirements-completed: [MIGR-01]

coverage:
  - id: D1
    description: "OpenSource opens the TS source read-only and the source .db stays byte-identical (no -wal/-shm sidecar) after a full ScanTable pass"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/reader_test.go#TestSource_ByteIdentity"
        status: pass
    human_judgment: false
  - id: D2
    description: "DetectTS/SchemaVersion fail loud on a non-TS source or an out-of-range schema_versions max (supported [1,7])"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/reader_test.go#TestDetectTS_NotATSSource, TestSchemaVersion_TooNewRejected, TestSchemaVersion_TooOldRejected"
        status: pass
    human_judgment: false
  - id: D3
    description: "ScanTable restricts reads to the files/nodes/edges allow-list, tolerates an aged source missing later-added columns, supports a rowid resume cursor, and surfaces a truncated/corrupted read as an error rather than a silent short scan"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/reader_test.go#TestScanTable_AllowListRejectsDisallowedTables, TestScanTable_AgedToleratesMissingColumns, TestScanTable_ResumeCursor, TestScanTable_SurfacesErrorOnCorruptedSource"
        status: pass
    human_judgment: false
  - id: D4
    description: "nodeFromRow/edgeFromRow/fileFromRow map a scanned row to the target proto record per the D-02/D-05 field mapping (verbatim ids, start_col/end_col carried, ms->ns, metadata flattened, TS-only attrs dropped), reproducing the golden dump's constant:0f0ec020... node and file:cmd/weft/main.go contains edge verbatim"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/translate_test.go#TestNodeFromRow_GoldenSpotCheck, TestEdgeFromRow_GoldenFileSourceContains, TestNodeFromRow_StartColEndColCarried"
        status: pass
    human_judgment: false
  - id: D5
    description: "Malformed edges.metadata / files.errors JSON fails loud (wrapped error), never swallowed"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/translate_test.go#TestEdgeFromRow_MetadataMalformedFailsLoud, TestFileFromRow_ErrorsMalformedFailsLoud, TestFlattenMetadata"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-07-13
status: complete
---

# Phase 7 Plan 3: Migration Reader + Translation Summary

**Read-only `internal/migrate` SQLite reader (`OpenSource`/`DetectTS`/`SchemaVersion`/`ScanTable`) plus the row-to-proto translation layer (`nodeFromRow`/`edgeFromRow`/`fileFromRow`) implementing the D-02/D-05 field mapping with the RESEARCH-confirmed start_col/end_col CARRIED correction.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-07-13 (session start)
- **Completed:** 2026-07-13
- **Tasks:** 2
- **Files modified:** 4 (reader.go, reader_test.go, translate.go, translate_test.go — all new)

## Accomplishments
- `internal/migrate.OpenSource` opens the TS SQLite DB via a read-only URI DSN (`mode=ro&_pragma=query_only(1)`), proven byte-identical (no `-wal`/`-shm` sidecar) after a full read pass — the D-08 non-destructive-to-source guarantee is test-verified, not just documented
- `DetectTS`/`SchemaVersion` fail loud: a DB missing `schema_versions`/`nodes`/`edges` returns `ErrNotATSSource`; a schema version outside the observed-safe `[1,7]` range returns `ErrUnsupportedSchemaVersion`
- `ScanTable` restricts reads to the `{files,nodes,edges}` allow-list (rejecting `nodes_fts`, `unresolved_refs`, `sqlite_master`, `schema_versions`, `name_segment_vocab` before ever querying them), builds its `SELECT` from `PRAGMA table_info`'s present-columns intersection (aged-DB tolerance, D-09.4), supports a `rowid` resume cursor (`afterRowID`), and checks `rows.Err()` after every loop
- `nodeFromRow`/`edgeFromRow`/`fileFromRow` translate a scanned row to `schema.Node`/`Edge`/`File`: verbatim ids (D-01), `start_col`/`end_col` **CARRIED** (the RESEARCH correction to CONTEXT's D-05 wording — they are modeled proto fields 9/10, not dropped), `is_async`/`is_static`/`is_abstract`/`decorators`/`type_parameters` and both timestamp-only columns (`nodes.updated_at`, `files.indexed_at`) dropped, `modified_at` ms→ns via `msToNs`, `edges.metadata` JSON flattened via `flattenMetadata`, `files.errors` JSON array via `parseErrorsJSON`
- Golden spot-checks reproduce `testdata/golden/ts-schema.dump.sql`'s `constant:0f0ec02010b45f3735f3f6e3367ec872` node and `file:cmd/weft/main.go` → `import:daa6c015...` `contains` edge verbatim, field-for-field against the real dump

## Task Commits

Each task was committed atomically:

1. **Task 1: reader.go — read-only source open, TS detection, schema guard, defensive row scan** - `5a24417` (feat)
2. **Task 2: translate.go — SQLite row → schema.Node/Edge/File (field mapping D-02/D-05)** - `ae58826` (feat)

_No plan-metadata commit needed beyond this SUMMARY/STATE/ROADMAP commit (below)._

## Files Created/Modified
- `internal/migrate/reader.go` - `Source`, `OpenSource`, `DetectTS`, `SchemaVersion`, `presentColumns`, `ScanTable`, `CountRows`, `CountDistinctEdges`, `FindDBFile`; `ErrNotATSSource`, `ErrUnsupportedSchemaVersion`
- `internal/migrate/reader_test.go` - open/detect/version/scan/allow-list/resume-cursor/byte-identity/corrupted-source/count/find-db-file coverage against `migratetest.BuildTSIndex` fixtures
- `internal/migrate/translate.go` - `nodeFromRow`, `edgeFromRow`, `fileFromRow`, `flattenMetadata`, `parseErrorsJSON`, `msToNs`, `normalizeFilePath`, plus `asString`/`asInt64`/`asFloat64`/`asBool` driver-value coercion helpers
- `internal/migrate/translate_test.go` - table-driven translation tests including two golden dump spot-checks and malformed-JSON fail-loud cases

## Decisions Made
- Confirmed and implemented the RESEARCH §Field Mapping correction to CONTEXT.md's D-05 wording: `start_column`/`end_column` **are** modeled (`Node.StartCol`=9, `Node.EndCol`=10) and are therefore carried, not dropped — only `is_async`/`is_static`/`is_abstract`/`decorators`/`type_parameters` are the genuine unconditional drops
- `msToNs` branches on the concrete Go type the driver returns (`int64` → direct multiply; `float64`/string → float path) rather than always converting through `float64`, because a naive always-`float64` conversion silently loses precision on large integer millisecond values once multiplied by `1e6` (exceeds float64's ~2^53 exact-integer range) — caught by a table-driven test comparing the direct-multiply path against a float-forced expectation
- The plan's suggested "close the DB mid-scan" technique for proving `rows.Err()` surfaces a truncated read does not actually work with `database/sql`: `DB.Close()` does not forcibly close a connection still serving an open `*sql.Rows`, so the callback-triggered close never produced an error. Substituted a mid-file-truncated fixture copy (valid header, corrupted b-tree pages) which reliably reproduces a fail-loud scan error — functionally equivalent proof of the Pitfall 2 discipline, different mechanism than the plan's illustrative example

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `rows.Err()` truncated-read test technique didn't reproduce a failure as written**
- **Found during:** Task 1 (writing reader_test.go's negative case for the mandatory `rows.Err()` check)
- **Issue:** The plan's illustrative example ("closing the DB mid-scan surfaces an error rather than silent truncation") does not actually trigger an error with Go's `database/sql`: `sql.DB.Close()` only prevents new connections and closes idle ones — a connection actively serving an open `*sql.Rows` stays alive until that `Rows` is closed/exhausted, so the originally-written test passed a `nil` error and failed its own assertion.
- **Fix:** Replaced with a fixture that copies the built `.db`, truncates it to half its length (keeping the valid SQLite header but corrupting the b-tree pages a full table scan must walk), and asserts `ScanTable` returns an error (either at `OpenSource`/`Ping` or during the scan itself — both are acceptable fail-loud outcomes). Verified it reliably fails as expected.
- **Files modified:** internal/migrate/reader_test.go
- **Commit:** 5a24417

**2. [Rule 1 - Bug] `msToNs` float64 round-trip test asserted against constant-folded arithmetic, not runtime arithmetic**
- **Found during:** Task 2 (translate_test.go's `TestMsToNs` fractional-ms case)
- **Issue:** The test's `want` value was written as an untyped Go constant expression (`int64(1783108606938.7 * 1e6)`), which the Go compiler evaluates at arbitrary precision — this silently disagreed with `msToNs`'s actual runtime `float64` multiplication in the least-significant digits (`...938700000` constant-folded vs `...938700032` at runtime).
- **Fix:** Bound the fractional value to a typed `float64` variable used identically as both the test input and the `want` computation, so both go through the same runtime IEEE-754 rounding. This is a test-only fix; `msToNs`'s production logic was already correct (verified once the test's own arithmetic matched reality).
- **Files modified:** internal/migrate/translate_test.go
- **Commit:** ae58826

No auth gates encountered. No architectural changes required — every task fit the plan's declared API surface exactly.

## Issues Encountered
None beyond the two test-technique fixes documented above (Deviations).

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `internal/migrate.OpenSource`/`ScanTable`/`nodeFromRow`/`edgeFromRow`/`fileFromRow` are ready for 07-04 (write-side orchestration: `migrate.go`'s node-then-edge-with-ownerPath loop through `graphstore.Writer`)
- `FindDBFile` is ready for CLI wiring (07-06) to autodetect the source `*.db` inside a `.codegraph/` dir
- `CountRows`/`CountDistinctEdges` are ready for 07-05's D-09.1 count-reconciliation validation pass
- No blockers for any downstream 07-04 through 07-07 plan

---
*Phase: 07-migration-tool*
*Completed: 2026-07-13*

## Self-Check: PASSED

All created files and task commit hashes verified present on disk / in git log.
