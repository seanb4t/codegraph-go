---
phase: 07-migration-tool
plan: 02
subsystem: database
tags: [pebble, protobuf, graphstore, meta-record, additive-schema]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 01)
    provides: modernc.org/sqlite fixture harness + confinement archtest (unused by this plan directly, but establishes internal/migrate as the eventual consumer of PutMigration/GetMigration)
  - phase: 01-foundation-storage-schema-parser-strategy
    provides: internal/graphstore Writer/Reader interfaces, metaKey/appendSegment length-prefixed key encoder, deterministicMarshal, additive-namespace precedent (prefixFileIndex doc comment)
provides:
  - Writer.PutMigration([]byte) error / Reader.GetMigration() ([]byte, error) on the graphstore interfaces
  - pebbleWriter.PutMigration (raw batch.Set, no proto marshal) and pebbleReader.GetMigration (via new getRaw helper) implementations
  - migrationRecordName = "migration" constant — a second, distinct m/ meta-namespace name sitting alongside metaRecordName = "schema"
  - getRaw(getter, key) ([]byte, error) — a general raw-bytes-with-copy Pebble getter, sibling of getProto, reusable by any future raw-blob meta record
affects: [07-03, 07-04, 07-05, 07-06, 07-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive meta-namespace record: a second fixed name under the existing m/ prefix via metaKey(name), zero key-format change, no SchemaVersion bump — same pattern usable for any future store-wide scalar/blob record"
    - "getRaw mirrors getProto's ErrNotFound-mapping + copy-before-closer.Close discipline, but skips proto.Unmarshal for opaque []byte payloads owned entirely by the caller package"

key-files:
  created:
    - internal/graphstore/migration_record_test.go
  modified:
    - internal/graphstore/store.go
    - internal/graphstore/batch.go
    - internal/graphstore/pebble_store.go
    - internal/graphstore/keys.go
    - internal/indexer/resolve_test.go
    - internal/query/traverse_test.go
    - internal/query/search_test.go

key-decisions:
  - "PutMigration writes via a raw batch.Set (no deterministicMarshal) — the payload is an opaque []byte owned and encoded by internal/migrate/progress.go (not yet built), not a proto message"
  - "getRaw copies the value into a caller-owned slice before closer.Close, exactly mirroring the RESEARCH-flagged buffer-lifetime rule (T-07-02-02) rather than trusting Pebble's Get to return a stable slice"
  - "Updated three existing in-package Writer/Reader test doubles (stubWriter in internal/indexer, traverseFakeReader and searchFakeReader in internal/query) to keep them compiling against the widened interfaces — no test behavior changed, purely mechanical interface-satisfaction stubs"

patterns-established:
  - "Second meta-namespace record via metaKey(distinctName): confirmed as the established, zero-format-change way to add a new store-wide scalar/blob record without touching SchemaVersion or the key namespace prefixes"

requirements-completed: [MIGR-02]

coverage:
  - id: D1
    description: "Writer.PutMigration/Reader.GetMigration round-trip a migration progress blob byte-identically through the GraphStore boundary"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/graphstore/migration_record_test.go#TestMigrationRecord_RoundTrip"
        status: pass
    human_judgment: false
  - id: D2
    description: "GetMigration on a store that never had a migration record returns graphstore.ErrNotFound, never a panic or nil-success"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/graphstore/migration_record_test.go#TestMigrationRecord_AbsentReturnsErrNotFound"
        status: pass
    human_judgment: false
  - id: D3
    description: "The m/migration record and the real m/schema Meta record coexist without clobbering each other; SchemaVersion stays 1"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/graphstore/migration_record_test.go#TestMigrationRecord_IsolatedFromMeta"
        status: pass
    human_judgment: false
  - id: D4
    description: "PutMigration is last-write-wins on the single m/migration key (overwrite idempotency)"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/graphstore/migration_record_test.go#TestMigrationRecord_OverwriteIdempotent"
        status: pass
    human_judgment: false
  - id: D5
    description: "D-04a sole-storage-door archtest (TestNoPackageBypassesGraphStore) remains green after the additive interface change"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest/import_graph_test.go#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-07-12
status: complete
---

# Phase 7 Plan 2: Migration Progress Record Summary

**Additive `Writer.PutMigration([]byte)` / `Reader.GetMigration() ([]byte, error)` pair backed by a new `m/migration` meta-record, kept byte-isolated from the real `m/schema` Meta via a distinct fixed key name — the storage door for D-06's resumable migration progress cursor.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-07-12T20:27:14-04:00 (approx, after 07-01 completion commit)
- **Completed:** 2026-07-12T20:31:13-04:00
- **Tasks:** 1
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments
- `Writer.PutMigration`/`Reader.GetMigration` added to the `graphstore` interfaces, backed by real `pebbleWriter`/`pebbleReader` implementations
- New `getRaw` helper (sibling of `getProto`) does a raw-bytes Pebble `Get`, copying the value before `closer.Close` (buffer-lifetime safety) and mapping `pebble.ErrNotFound` → `graphstore.ErrNotFound`
- `migrationRecordName = "migration"` constant added to `keys.go`, documented as the additive second meta-namespace name — zero key-format change, `SchemaVersion` unchanged at 1
- Round-trip, absent-record, Meta-isolation, and overwrite-idempotency proven by a new `migration_record_test.go`
- `TestNoPackageBypassesGraphStore` (D-04a) confirmed still green — the sole-storage-door boundary is unaffected

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: failing test for PutMigration/GetMigration** - `f95df0e` (test)
2. **Task 1 GREEN: additive PutMigration/GetMigration implementation** - `7f5513f` (feat)

_No REFACTOR commit — the GREEN implementation was already minimal; no cleanup needed._

## Files Created/Modified
- `internal/graphstore/migration_record_test.go` - round-trip, absent-ErrNotFound, Meta-isolation, overwrite-idempotency tests
- `internal/graphstore/store.go` - `Writer.PutMigration`/`Reader.GetMigration` interface methods
- `internal/graphstore/batch.go` - `pebbleWriter.PutMigration` (raw `batch.Set`, no `deterministicMarshal`)
- `internal/graphstore/pebble_store.go` - `getRaw` helper + `pebbleReader.GetMigration`
- `internal/graphstore/keys.go` - `migrationRecordName = "migration"` constant with additive-namespace doc comment
- `internal/indexer/resolve_test.go` - `stubWriter.PutMigration` no-op stub (interface-satisfaction only)
- `internal/query/traverse_test.go` - `traverseFakeReader.GetMigration` not-implemented stub (interface-satisfaction only)
- `internal/query/search_test.go` - `searchFakeReader.GetMigration` not-implemented stub (interface-satisfaction only)

## Decisions Made
- `PutMigration` bypasses `deterministicMarshal` entirely — unlike every other `PutX` in `batch.go`, the payload is an opaque `[]byte` owned by the future `internal/migrate/progress.go`, not a proto message, so a raw `batch.Set` is correct and there is no map-ordering determinism concern to guard against.
- `getRaw` was written as a small, general, reusable raw-bytes getter (not `GetMigration`-specific inline code) so any future raw-blob meta record can reuse it without duplicating the copy-before-close / ErrNotFound-mapping logic.
- The three pre-existing in-package Writer/Reader test doubles (`stubWriter`, `traverseFakeReader`, `searchFakeReader`) were updated with minimal not-implemented/no-op stubs to keep them satisfying the widened interfaces — this is Rule 3 (blocking-issue auto-fix): the plan's own action explicitly called for keeping "any in-package test doubles / mock Writer-Reader implementations in graphstore tests compiling," and the fix is purely mechanical (no behavior change to the tests that use them).

## Deviations from Plan

None — plan executed exactly as written. The three test-double updates were explicitly anticipated and directed by the plan's own `<action>` text ("Keep any in-package test doubles ... compiling by adding the new methods to them"), not an unplanned discovery.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `graphstore.Writer.PutMigration`/`Reader.GetMigration` are ready for `internal/migrate/progress.go` (a later 07-xx plan) to persist and read back the D-06 resumable migration progress cursor
- The `m/migration` key is proven isolated from `m/schema` — a later plan building the real progress-record encoding (source `schema_versions` max, target `SchemaVersion`, cursor, status) can build directly on this storage door without re-verifying the boundary
- No blockers for subsequent 07-xx plans (reader, translate, migrate orchestration, swap, CLI wiring)

---
*Phase: 07-migration-tool*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files and both task commit hashes (f95df0e, 7f5513f) verified present on disk / in git log.
