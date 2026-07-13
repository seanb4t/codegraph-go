---
phase: 07-migration-tool
plan: 05
subsystem: migration
tags: [validation, referential-integrity, graphstore, reconciliation]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 03)
    provides: "internal/migrate.OpenSource/ScanTable/CountRows/CountDistinctEdges (the read-only reader) and nodeFromRow/edgeFromRow/fileFromRow (the row->proto translation layer)"
provides:
  - "internal/migrate.validate(src, store, opts) (Report, error) — the D-09 post-migration invariant pass"
  - "internal/migrate.Options{Force, DropDangling bool} — the package's single shared options struct (07-06 reconciles with this, does not redeclare)"
  - "internal/migrate.Report{Nodes,Files,Edges TableCounts; Dangling []DanglingEdge; Dropped int} — the reconciliation/dangling report shape"
  - "internal/migrate.isFileEndpoint(id) bool — the file:-prefix exemption predicate, reused wherever endpoint resolution matters"
affects: [07-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "validate() reads the migrated store back exclusively through the existing graphstore.Reader surface (Snapshot/IterateNodes/IterateEdges/IterateFiles/GetNode) — no new read machinery, matching every other internal/migrate file's reuse-the-storage-layer discipline"
    - "Edge count reconciliation compares against Source.CountDistinctEdges(), never CountRows(\"edges\") — the de-dup-aware comparison required because the Pebble edgeKey omits line/col and collapses same-(source,kind,target) rows"
    - "file:-prefixed edge endpoints are exempt from the dangling-edge check (isFileEndpoint), not synthesized as pseudo-nodes — TS models files as edge endpoints, the new schema does not"

key-files:
  created:
    - internal/migrate/validate.go
    - internal/migrate/validate_test.go
  modified: []

key-decisions:
  - "Options{Force, DropDangling bool} defined in validate.go (not migrate.go, which does not exist yet) per the plan's artifacts_produced discretion — 07-06's migrate.go will reconcile with this single definition rather than redeclaring it"
  - "dropDanglingEdges performs a best-effort x/ file-index cleanup: when a dangling edge's SOURCE resolves to a node, its owning file's DeleteFileIndexEdge entry is also removed so IterateFileIndex stays consistent; when the source itself is the missing endpoint, no owner was ever recorded (ownerPath=\"\" per the documented D-04 write convention) so there is nothing to clean up — avoids fabricating an owner for an edge that never had one"
  - "validate_test.go builds its own store by hand (ScanTable + nodeFromRow/edgeFromRow/fileFromRow + PutNode/PutEdge with ownerPath tracking) rather than depending on 07-06's not-yet-written migrate.Run orchestration, since this plan depends only on 07-03 — the helper (buildStoreFromSource) mirrors the D-04 nodes-before-edges/ownerPath convention closely enough to be a faithful stand-in"

patterns-established:
  - "internal/migrate error-wrap convention extended to validate.go: every error prefixed 'migrate: validate: <verb>: %w'"

requirements-completed: [MIGR-02]

coverage:
  - id: D1
    description: "reconcileCounts compares migrated Node/File counts against source row counts (exact equality) and migrated Edge count against the source's DISTINCT(source,kind,target) count (de-dup aware), returning the full Report"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/validate_test.go#TestValidate_HappyReconcile"
        status: pass
    human_judgment: false
  - id: D2
    description: "A source edge duplicated only in (line,col) collapses to one stored edge in the target format, and reconcile passes by comparing against CountDistinctEdges rather than the raw source row count — proving a correctly-collapsing migration does not fail its own check"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/validate_test.go#TestValidate_EdgeDedupTolerance"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real migrated shortfall (a deleted node) is a fail-loud error naming the mismatched table and both counts, never silent"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/validate_test.go#TestReconcileCounts_MismatchFailsLoud"
        status: pass
    human_judgment: false
  - id: D4
    description: "scanDangling builds a node-id set and flags every non-file: edge endpoint that fails to resolve; file:-prefixed endpoints are exempt and never reported as dangling, proven both directly (isFileEndpoint) and via the happy path's file:-source contains edges passing with zero dangling"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/validate_test.go#TestIsFileEndpoint, TestValidate_HappyReconcile"
        status: pass
    human_judgment: false
  - id: D5
    description: "A dangling (non-file:) edge fails validate loud by default and leaves the store unchanged; --drop-dangling (Options.DropDangling) instead deletes it, records the count in Report.Dropped, returns nil error, and a re-scan finds zero dangling"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/validate_test.go#TestValidate_DanglingFailsLoudByDefault, TestValidate_DropDangling"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-12
status: complete
---

# Phase 7 Plan 5: Migration Validation (D-09 Invariant Pass) Summary

**`internal/migrate/validate.go` implements the D-09 post-migration invariant pass: de-dup-aware count reconciliation (migrated edges compared against the source's DISTINCT triple count, not raw rows) plus a zero-dangling-edges referential-integrity scan that exempts `file:`-prefixed endpoints, gated by a fail-loud-vs-`--drop-dangling` policy that is the D-10 gate for `Meta.healthy`.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-07-12 (session start)
- **Completed:** 2026-07-12
- **Tasks:** 1
- **Files modified:** 2 (validate.go, validate_test.go — both new)

## Accomplishments
- `reconcileCounts` reads the migrated store back via `graphstore.Reader` (`IterateNodes`/`IterateEdges`/`IterateFiles`) and compares against `Source.CountRows("nodes")`, `CountRows("files")`, and — critically — `Source.CountDistinctEdges()` for edges, never `CountRows("edges")`, so a migration that correctly collapses duplicate-triple edge rows (the Pebble `edgeKey` omits line/col) passes its own check instead of failing it
- `scanDangling` builds an in-memory node-id set, then scans every edge's source/target for referential integrity; `isFileEndpoint` exempts `file:`-prefixed endpoints (TS's synthetic file-node ids) rather than requiring pseudo-nodes to be synthesized
- Default policy: a non-empty dangling list is a fail-loud error naming the count and an example triple, and the store is left unchanged
- `Options.DropDangling=true` opts into deleting each dangling edge (`DeleteEdge`, plus a best-effort `DeleteFileIndexEdge` when the edge's source resolved to an owning file) and records the count in `Report.Dropped`; a re-scan afterward finds zero dangling
- `validate(src, store, opts)` orchestrates both phases and is the function 07-06's `migrate.Run` will call to gate `Meta.healthy=true` (D-10) on a nil error return
- `Options{Force, DropDangling bool}` is defined here as the package's single shared options struct — 07-06's `migrate.go` (not yet written) will reconcile with this definition rather than redeclaring it, per the plan's own coordination note

## Task Commits

Each task was committed atomically (TDD RED → GREEN):

1. **Task 1 RED: add failing test for validate.go** - `ea7b6bd` (test) — verified the package fails to build (undefined `isFileEndpoint`) with `validate.go` absent, confirming a genuine RED before implementation existed
2. **Task 1 GREEN: implement validate.go** - `f01efa6` (feat) — all 6 tests pass on first run against the implementation

_No plan-metadata commit needed beyond this SUMMARY/STATE/ROADMAP commit (below)._

## Files Created/Modified
- `internal/migrate/validate.go` - `Options`, `TableCounts`, `DanglingEdge`, `Report`, `isFileEndpoint`, `validate`, `reconcileCounts`, `countNodes`/`countEdges`/`countFiles`, `scanDangling`, `nodeIDSet`, `findDangling`, `dropDanglingEdges`
- `internal/migrate/validate_test.go` - `buildStoreFromSource` (hand-rolled node-then-edge-with-ownerPath store builder mirroring the future 07-06 write loop), `openHappySource`, and 6 tests: `TestIsFileEndpoint`, `TestValidate_HappyReconcile`, `TestValidate_EdgeDedupTolerance`, `TestReconcileCounts_MismatchFailsLoud`, `TestValidate_DanglingFailsLoudByDefault`, `TestValidate_DropDangling`

## Decisions Made
- `Options{Force, DropDangling bool}` lives in `validate.go` (not `migrate.go`, which 07-06 has not yet created) per the plan's explicit discretion ("defined here or in migrate.go — coordinate a single definition")
- `dropDanglingEdges` best-effort-cleans the `x/` file-index entry when a dangling edge's source resolves to a node (only the target is dangling): it looks up the source's `FilePath` via `GetNode` and calls `DeleteFileIndexEdge`. When the source itself is the missing endpoint, no owner was ever recorded (the documented D-04 convention is `ownerPath=""` for an unresolved source), so there is nothing to clean up — this avoids fabricating an owner for an edge that never had one
- `validate_test.go` builds its own graphstore by hand (`buildStoreFromSource`, using `Source.ScanTable` + `nodeFromRow`/`edgeFromRow`/`fileFromRow` from 07-03, writing via `PutNode`/`PutEdge`/`PutFile`) instead of depending on 07-06's not-yet-written `migrate.Run` — this plan's `depends_on` is only `["07-03"]`, and the helper faithfully mirrors the nodes-before-edges/ownerPath convention 07-06 will implement, so the tests exercise `validate` against a realistic store

## Deviations from Plan

None — plan executed exactly as written. All 6 tests passed on the first GREEN run with no auto-fixes needed.

## Issues Encountered
None.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `internal/migrate.validate`/`Options`/`Report` are ready for 07-06's `migrate.Run` orchestration to call directly (`validate(src, store, opts.DropDangling)` per 07-06's own plan text — note 07-06 must pass `opts` itself since `validate` takes `Options`, not a bare bool; confirm at 07-06 implementation time)
- No blockers for 07-06

---
*Phase: 07-migration-tool*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created files and task commit hashes verified present on disk / in git log.
