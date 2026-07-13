---
phase: 07-migration-tool
verified: 2026-07-12T22:05:00Z
status: passed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 7: Migration Tool Verification Report

**Phase Goal:** An existing TS CodeGraph user can convert their aged `.codegraph/` SQLite index to the new format in one resumable, validated step.
**Verified:** 2026-07-12T22:05:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A single migration command converts an existing TS `.codegraph/` SQLite index to the new format | ✓ VERIFIED | `internal/cli/migrate.go:newMigrateCmd` registered in `internal/cli/root.go:50`; `RunE` resolves `--from`/`--to` (default: cwd `.codegraph/`), delegates to `migrate.Run`, prints reconciliation report. `TestMigrateCmdEndToEnd` passes. |
| 2 | Migration is resumable after interruption and version-stamped so partial runs recover correctly | ✓ VERIFIED (behavioral test) | `internal/migrate/progress.go` (`Progress{SourceSchemaVersion,TargetSchemaVersion,LastTable,LastRowID,Status}` persisted via `graphstore.Writer.PutMigration`/`Reader.GetMigration`); `migrate.go`'s `advanceCursor`/`resumePosition`/`recoverInterruptedSwap`. `TestRun_Resume` interrupts mid-migration via the `testStopAfterBatch` seam, asserts no swap occurred, re-runs, and asserts `Result.Resumed=true` with counts matching an uninterrupted run and `Meta.Healthy=true`. `TestRun_RecoversInterruptedSwap` and `TestRun_RecoveryLeavesInProgressPartialAlone` cover the swap-crash-window case (WR-04). All pass. |
| 3 | Migration is validated against real aged `.codegraph/` directories AND runs structural-invariant checks on the result, failing loudly on corruption rather than producing a silently-wrong graph | ✓ VERIFIED | `internal/migrate/validate.go` (`reconcileCounts` de-dup-aware edge comparison via `CountDistinctEdges`; `scanDangling` referential-integrity scan with `file:` exemption; fail-loud default, `--drop-dangling` opt-in). `Meta.Healthy` set true only after `validate` returns nil (`migrate.go:244`). Aged-schema fixture (`migratetest.VariantAged`, built from the pre-existing captured `testdata/golden/ts-schema.sql`/`ts-schema.dump.sql`) drives `TestRun_AgedDB` — migrates a source missing `nodes.return_type`/`edges.provenance` to a healthy store. `TestRun_DanglingFailsLoud` and `TestRun_DropDangling` cover both policy branches. All pass. |

**Score:** 3/3 roadmap success criteria verified (all behavior-dependent truths carry a passing behavioral test, not just presence).

### Plan-Level Must-Haves (07-01 through 07-07)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 4 | `modernc.org/sqlite v1.53.0` available to `internal/migrate` only, confined by an architecture test | ✓ VERIFIED | `go.mod:32` direct require; `internal/migrate/archtest/modernc_confinement_test.go` uses `go/packages` import-graph inspection (not regex) with a self-check that fails if it can't detect a real importer. `go test ./internal/migrate/archtest/... -run TestModerncSQLiteConfinedToMigrate -v` → PASS. |
| 5 | In-Go fixture harness builds a real TS-shaped SQLite `.db` with 3 variants (happy/aged/dangling), no external tools | ✓ VERIFIED | `internal/migrate/migratetest/fixture.go`: `BuildTSIndex(t, Variant)`, `VariantHappy`/`VariantAged`/`VariantDangling` consts, reconstructs from committed `testdata/golden/ts-schema.sql` + `.dump.sql` via the modernc driver — no `sqlite3` CLI, no live TS CLI. |
| 6 | Additive `PutMigration`/`GetMigration` on graphstore, isolated `m/migration` key, `ErrNotFound` when absent | ✓ VERIFIED | `internal/graphstore/store.go` interfaces; `pebble_store.go:198` (`getRaw`, not proto unmarshal); `batch.go:118` (raw `Set`, not `deterministicMarshal`); `keys.go:204` (`migrationRecordName = "migration"`, distinct from `metaRecordName`). `TestMigrationRecord_RoundTrip`, `_AbsentReturnsErrNotFound`, `_IsolatedFromMeta`, `_OverwriteIdempotent` all pass. |
| 7 | Reader opens source read-only, detects non-TS DBs, defensive column scan (PRAGMA table_info), allow-listed tables only, `rows.Err()` checked | ✓ VERIFIED | `internal/migrate/reader.go`: `sourceDSN` (`mode=ro&_pragma=query_only(1)`, URL-escaped per WR-05 fix), `DetectTS` (`ErrNotATSSource`), `allowedTables` fixed map (never DB-derived), `presentColumns`/`ScanTable` intersect wanted vs. present columns, `return rows.Err()` after every loop. |
| 8 | Row→proto translation: verbatim TS ids (D-01), start_col/end_col carried (D-05 correction), ms→ns, metadata flatten, correct drops | ✓ VERIFIED | `internal/migrate/translate.go`: `Id: asString(row["id"])` (no `nodeid.NodeID` recompute), `StartCol`/`EndCol` mapped from `start_column`/`end_column`, `msToNs` (exact-int path + float fallback), `flattenMetadata`, `is_async`/`is_static`/`is_abstract`/`decorators`/`type_parameters` absent from `nodeFromRow`. |
| 9 | Durable resumable progress cursor; absent cursor reports cleanly; atomic same-filesystem directory swap, restore-on-failure | ✓ VERIFIED | `internal/migrate/progress.go` (`loadProgress` maps `graphstore.ErrNotFound`→`absent=false`); `internal/migrate/swap.go` (`siblingTempDir` — same parent as target, never `os.TempDir()`; `atomicSwapDir` 3-step rename with restore-on-failure). `TestRun_Resume`/`swap_test.go` pass. |
| 10 | D-09 validation: de-dup-aware edge count reconciliation, `file:`-exempt dangling scan, fail-loud default / `--drop-dangling` opt-in, `Meta.healthy` gated | ✓ VERIFIED | `internal/migrate/validate.go` as detailed in Truth 3 above. |
| 11 | `codegraph migrate` cobra command: flags, defaults, non-destructive refusal, report, non-TS error surfacing | ✓ VERIFIED | `internal/cli/migrate.go` + `root.go:50` registration. `TestMigrateCmdRegisteredAndFlags`, `TestMigrateCmdEndToEnd`, `TestMigrateCmdFailsLoudOnNonTSSource`, `TestMigrateCmdRefusesUnrecognizedTargetWithoutForce` all pass. |

**Score:** 11/11 must-haves verified (roadmap + plan-level, deduplicated).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/migrate/migratetest/fixture.go` | `BuildTSIndex`, 3 variants | ✓ VERIFIED | Exists, substantive, wired into `reader_test.go`, `translate_test.go`, `validate_test.go`, `migrate_test.go`, `cli/migrate_test.go` |
| `internal/migrate/archtest/modernc_confinement_test.go` | `TestModerncSQLiteConfinedToMigrate` | ✓ VERIFIED | Exists, passes, self-checking (fails if it can't detect a real importer) |
| `internal/graphstore/store.go` / `batch.go` / `pebble_store.go` / `keys.go` | `PutMigration`/`GetMigration` additive pair | ✓ VERIFIED | All 4 files updated consistently; round-trip + isolation tests pass |
| `internal/migrate/reader.go` | `OpenSource`, `DetectTS`, `SchemaVersion`, `CountRows`, `CountDistinctEdges`, `ScanTable`, `FindDBFile` | ✓ VERIFIED | All present, all exercised by `reader_test.go` |
| `internal/migrate/translate.go` | `nodeFromRow`/`edgeFromRow`/`fileFromRow` + helpers | ✓ VERIFIED | All present, exercised by `translate_test.go` |
| `internal/migrate/progress.go` | `Progress`, `saveProgress`, `loadProgress` | ✓ VERIFIED | Exists, exercised by `progress_test.go` and end-to-end by `migrate_test.go` |
| `internal/migrate/swap.go` | `siblingTempDir`, `atomicSwapDir`, `checkWritableDir` | ✓ VERIFIED | Exists, exercised by `swap_test.go` and end-to-end |
| `internal/migrate/validate.go` | `Report`, `validate`, `reconcileCounts`, `scanDangling`, `isFileEndpoint` | ✓ VERIFIED | Exists, exercised by `validate_test.go` and `migrate_test.go` |
| `internal/migrate/migrate.go` | `Run`, `Options`, `Result` | ✓ VERIFIED | Exists, orchestrates full pipeline, 11 integration tests in `migrate_test.go` |
| `internal/cli/migrate.go` | `newMigrateCmd` | ✓ VERIFIED | Exists, registered, tested |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `migratetest/fixture.go` | `testdata/golden/ts-schema.sql`/`.dump.sql` | in-Go DDL+dump reconstruction | ✓ WIRED | Uses pre-existing (Phase-6-era) captured golden fixtures — real TS-CodeGraph-captured schema/data, not fabricated |
| `reader.go` | `translate.go` | `map[string]any` row → schema record | ✓ WIRED | `scanAndWriteTable` in `migrate.go` calls `src.ScanTable` with a closure invoking `nodeFromRow`/`edgeFromRow`/`fileFromRow` |
| `reader.go` | `modernc.org/sqlite` | `sql.Open("sqlite", ...)` blank import | ✓ WIRED | `reader.go:20` blank import; `sourceDSN` builds the DSN |
| `progress.go` | `internal/graphstore` | `PutMigration`/`GetMigration` | ✓ WIRED | `saveProgress`/`loadProgress` call through directly |
| `swap.go` | `internal/upgrade/swap.go` pattern | sibling-temp + rename-aside/rename-in/remove-old | ✓ WIRED | Same discipline extended to a directory target, with restore-on-failure (WR-04-hardened) |
| `migrate.go` | `internal/graphstore/export.go` | nodes-before-edges `nodeFilePath`→`ownerPath` | ✓ WIRED | `scanAndWriteTable`'s "edges" case; `file:`-source edges pass `ownerPath=""` |
| `migrate.go` | `validate.go` | `validate(src, store, opts)` gates `Meta.Healthy` | ✓ WIRED | `migrate.go:203-244` — `Healthy=true` set only after nil error from `validate` |
| `migrate.go` | `swap.go` | `atomicSwapDir(tmpDir, target)` only on success | ✓ WIRED | Swap call is unreachable on the validation-error return path (line 209 returns before reaching line 271) |
| `cli/migrate.go` | `internal/migrate` | `migrate.Run(...)` | ✓ WIRED | `RunE` calls `migrate.Run` and prints `Result` |
| `cli/root.go` | `cli/migrate.go` | `root.AddCommand(..., newMigrateCmd())` | ✓ WIRED | `root.go:50` |

### Behavioral Spot-Checks / Targeted Test Runs

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Archtest confinement | `go test ./internal/migrate/archtest/... -run TestModerncSQLiteConfinedToMigrate -v` | PASS | ✓ PASS |
| Migration-record round-trip/isolation | `go test ./internal/graphstore/... -run TestMigration -v` | 4/4 PASS | ✓ PASS |
| Resume after simulated interruption | `go test ./internal/migrate/... -run TestRun_Resume -v` | PASS (Resumed=true, counts match, Healthy=true) | ✓ PASS |
| Interrupted-swap self-heal | `go test ./internal/migrate/... -run TestRun_RecoversInterruptedSwap -v` | PASS | ✓ PASS |
| Aged-DB migration (missing later-added columns) | `go test ./internal/migrate/... -run TestRun_AgedDB -v` | PASS | ✓ PASS |
| Dangling edge fail-loud / --drop-dangling | `go test ./internal/migrate/... -run "TestRun_DanglingFailsLoud|TestRun_DropDangling" -v` | PASS | ✓ PASS |
| Full workspace build | `go build ./...` | clean, no errors | ✓ PASS |
| Full workspace test suite (serial, `-p 1`) | `go test ./... -count=1 -p 1` | All 29 packages pass, including `internal/daemon` (documented flake did not reproduce this run) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| MIGR-01 | 07-01, 07-03, 07-06, 07-07 | User can run a migration command converting an existing TS `.codegraph/` SQLite index to the new format in one step | ✓ SATISFIED | `codegraph migrate` command, `migrate.Run` orchestration, verified above |
| MIGR-02 | 07-01, 07-02, 07-03, 07-04, 07-05, 07-06, 07-07 | Migration is resumable, version-stamped, validated against real aged `.codegraph/` directories, and runs structural-invariant checks on the result | ✓ SATISFIED | `progress.go`/`swap.go` resumability, `validate.go` structural checks, `TestRun_AgedDB` against real captured golden schema, verified above |

REQUIREMENTS.md marks both MIGR-01 and MIGR-02 `[x]` and Phase 7 = "Complete" — consistent with code-level evidence, not merely asserted.

No orphaned requirements: REQUIREMENTS.md's "Migration" section lists exactly MIGR-01/MIGR-02, both claimed by phase plans.

### Anti-Patterns Found

None. Scanned all modified/created files in `internal/migrate/`, `internal/migrate/migratetest/`, `internal/migrate/archtest/`, `internal/cli/migrate.go`, and the 4 modified `internal/graphstore/*.go` files for `TODO|FIXME|XXX|TBD|HACK|PLACEHOLDER` and "not yet implemented"/"coming soon" style markers — zero hits. No debt-marker blockers.

The 5 Info-level findings from `07-REVIEW.md` (IN-01 `msToNs` float precision, IN-02 `checkWritableDir` dropped `Remove` error, IN-03 drive-letter heuristic, IN-04 no resume source-fingerprint check, IN-05 iterator `Close()` errors dropped) remain open by explicit scope decision in `07-REVIEW-FIX.md` (Info-level, not required for phase completion) — noted here for visibility, not treated as gaps.

### Human Verification Required

None. All roadmap success criteria and plan must-haves resolved to VERIFIED via direct source reading, targeted test execution, and a full serial `go test ./...` run. No visual, real-time, or external-service-dependent behavior in this phase.

### Gaps Summary

No gaps. All 3 roadmap success criteria and all 11 plan-level must-haves are implemented, wired, and behaviorally proven by passing tests (not just symbol presence). The 5 code-review Warnings (WR-01..WR-05) found in `07-REVIEW.md` were all fixed with regression tests per `07-REVIEW-FIX.md`, independently confirmed by re-reading the fixed code (`migrate.go`'s post-drop `recomputeFileEdgeCounts` re-run, `batchWriter.Close`, `checkTargetOverwrite`'s stat-before-open guard, `recoverInterruptedSwap`, and `sourceDSN`'s URL-escaped DSN construction).

---

_Verified: 2026-07-12T22:05:00Z_
_Verifier: Claude (gsd-verifier)_
