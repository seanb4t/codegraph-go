---
phase: 07-migration-tool
plan: 06
subsystem: migration
tags: [orchestration, resumability, atomic-swap, graphstore, pebble, crash-safety]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 03)
    provides: "internal/migrate.OpenSource/DetectTS/SchemaVersion/ScanTable/FindDBFile (read-only reader) and nodeFromRow/edgeFromRow/fileFromRow (row -> proto translation)"
  - phase: 07-migration-tool (plan 04)
    provides: "internal/migrate.Progress/saveProgress/loadProgress (durable resumable cursor) and siblingTempDir/atomicSwapDir/checkWritableDir (atomic directory swap)"
  - phase: 07-migration-tool (plan 05)
    provides: "internal/migrate.validate/Options/Report/isFileEndpoint/countNodes/countEdges/countFiles (D-09 invariant pass, D-10 healthy gate)"
provides:
  - "internal/migrate.Run(from, to string, opts Options) (Result, error) — the single-call MIGR-01/MIGR-02 orchestration entry point"
  - "internal/migrate.Result{Nodes,Edges,Files,Resumed,HealthMessage,Report}"
  - "deterministic .codegraph.migrate-partial/ sibling partial-store convention + resumable files->nodes->edges write loop with durable per-batch cursor checkpoints"
affects: [07-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Deterministic (not randomly-named) sibling partial-store directory (.codegraph.migrate-partial/) so a re-run after a crash finds the SAME partial store and resumes, unlike swap.go's randomly-named siblingTempDir which is used only for the final swap's own temp-then-rename mechanics"
    - "seedNodeFilePath scans the target store's already-committed nodes at the start of every Run (a no-op on a fresh run) so PutEdge's ownerPath lookup has full coverage regardless of resume state, without needing to persist nodeFilePath across process invocations"
    - "recomputeFileEdgeCounts reads the x/ file index (post-write source of truth) rather than in-memory bookkeeping to set File.edge_count — correct across resumes for free, no cursor-side accounting needed"
    - "batchWriter.commitData/advanceCursor split: commitData flushes staged puts (no-op if empty); advanceCursor always calls commitData then durably commits the progress cursor in its own small batch, unifying the mid-table batchSize checkpoint and the end-of-table flush into one code path"
    - "testStopAfterBatch — an unexported, test-only package-level hook fired after every durable cursor checkpoint, letting migrate_test.go simulate a crash between batches without a real process kill"

key-files:
  created:
    - internal/migrate/migrate.go
    - internal/migrate/migrate_test.go
  modified: []

key-decisions:
  - "partialDir uses a FIXED name (.codegraph.migrate-partial) under the target's parent, not swap.go's siblingTempDir (which mints a fresh random name every call) — resumability requires the SAME path across process invocations; siblingTempDir remains reserved for future randomly-named use cases, unused by Run"
  - "checkTargetOverwrite (D-08) opens <target>/store (not <target> directly) to probe for a prior healthy migration, matching internal/cli's established .codegraph/store/ subdirectory convention (storeDirName) confirmed via internal/cli/init.go — an earlier draft that opened <target> directly was a bug caught before commit"
  - "Table order is files -> nodes -> edges (not nodes -> edges -> files as informally paraphrased in some planning docs); files has no ownerPath dependency so its position doesn't affect D-04 correctness, and the plan's own <action> text and must_haves both specify files first"
  - "A found Progress with Status=complete (process died between saveProgress(complete) and atomicSwapDir) short-circuits straight to finishFromComplete: no re-write, no re-validate — just re-read counts/Meta and swap, since D-09 already passed in the interrupted prior invocation"
  - "recomputeFileEdgeCounts runs unconditionally after the edges phase (not gated on resume state) since it derives edge_count from the x/ index's actual current contents (Pitfall 7), which is correct whether those x/ entries were written in this invocation or a prior interrupted one"

patterns-established:
  - "internal/migrate error-wrap convention extended to migrate.go: every error prefixed 'migrate: <verb>: %w'"

requirements-completed: [MIGR-01, MIGR-02]

coverage:
  - id: D1
    description: "One Run(from, to, opts) call converts a happy-path TS SQLite index into a healthy new-format store: detect -> guard -> write (files, then nodes, then edges) -> validate -> atomic swap, with counts reconciling against the source (edge count de-dup aware) and Meta stamped SchemaVersion/HasFileIndex=true/Healthy=true/HealthMessage documenting the D-01 first-sync full-reindex behavior"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_Happy"
        status: pass
    human_judgment: false
  - id: D2
    description: "Edges are written nodes-before-edges so PutEdge's ownerPath resolves to the source node's file (D-04 x/ index population); a file:-prefixed edge source passes ownerPath=\"\" and migrates without error (isFileEndpoint exemption, no fabricated owner)"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_NodesBeforeEdgesOwnerPath"
        status: pass
    human_judgment: false
  - id: D3
    description: "The source SQLite file is byte-identical (sha256 before==after) and no -wal/-shm sidecar exists after a full Run — the source is never mutated (D-08)"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_SourceUnmodified"
        status: pass
    human_judgment: false
  - id: D4
    description: "An interrupted run (simulated via the test-only testStopAfterBatch seam, firing at the first durable cursor checkpoint) leaves no swapped target and no source mutation; a second Run call against the same target resumes from the durable in_progress cursor and completes with the same final counts an uninterrupted run produces, reporting Resumed=true"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_Resume"
        status: pass
    human_judgment: false
  - id: D5
    description: "A source missing later-added columns (nodes.return_type, edges.provenance) migrates to a healthy store, with migrated records carrying the proto zero value for the absent columns (D-09.4 aged-DB tolerance)"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_AgedDB"
        status: pass
    human_judgment: false
  - id: D6
    description: "A source with a genuinely dangling (non-file:) edge fails Run loudly by default, performs no atomic swap (target absent), leaves the source untouched, and leaves the partial store present for inspection/resume (D-09.2 default policy, D-10 healthy gate)"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_DanglingFailsLoud"
        status: pass
    human_judgment: false
  - id: D7
    description: "Options{DropDangling:true} completes healthy, the dangling edge is dropped (re-scan of the swapped-in store finds zero dangling), and Result.Report records the drop count"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/migrate/migrate_test.go#TestRun_DropDangling"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-07-13
status: complete
---

# Phase 7 Plan 6: Migration Orchestration (migrate.Run) Summary

**`internal/migrate.Run(from, to, opts)` wires the reader, translation layer, resumable progress cursor, D-09 validation, and atomic directory swap into the single-call MIGR-01/MIGR-02 entry point: detect+guard the TS source, write files->nodes->edges into a deterministic sibling partial store with durably-checkpointed bounded batches, recompute per-file edge_count from the written x/ index, gate Meta.healthy on validate passing, and atomically swap into place — resuming cleanly from a durable cursor after a simulated mid-run crash.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-07-13
- **Tasks:** 1
- **Files modified:** 2 (migrate.go, migrate_test.go — both new)

## Accomplishments
- `Run(from, to string, opts Options) (Result, error)` composes 07-03/07-04/07-05's pieces into one call: `resolveSourceDB` (accepts a `.codegraph/` dir or a direct `*.db` path) -> `OpenSource`/`DetectTS`/`SchemaVersion` -> `checkWritableDir`/`checkTargetOverwrite` (D-08) -> a **deterministic** sibling partial store (`.codegraph.migrate-partial/`, distinct from swap.go's randomly-named `siblingTempDir`) -> resume-aware files->nodes->edges write loop -> `recomputeFileEdgeCounts` -> `validate` -> `schema.NewMeta()` stamped `Healthy=true` only after validate passes -> `atomicSwapDir`
- `seedNodeFilePath` + the nodes-table callback together guarantee `PutEdge`'s ownerPath lookup has full `id -> FilePath` coverage regardless of whether nodes were written in this invocation or a prior interrupted one — no cross-process bookkeeping required
- `batchWriter` bounds Puts at `batchSize` (5000) per Commit and durably persists the progress cursor (`saveProgress`) in its own small batch immediately after each data commit (D-06); the unexported `testStopAfterBatch` seam lets `migrate_test.go` simulate an interruption at the first real checkpoint without a process kill
- `recomputeFileEdgeCounts` sets `File.edge_count` from the actual `x/` file-index contents post-write (Pitfall 7) — correct across resumes for free, since it reads the store's current state rather than tracking counts in memory
- `finishFromComplete` handles the narrow "died between marking complete and swapping" resume case: no re-write, no re-validate, just re-read the already-final counts/Meta and swap
- All 7 required behaviors (happy path, nodes-before-edges/ownerPath/x-index, source-byte-identity, resume-to-identical-counts, aged-DB tolerance, dangling-fails-loud-with-no-swap, drop-dangling) pass, including under `-race`

## Task Commits

Each task was committed atomically:

1. **Task 1: migrate.Run — detect/guard/write/resume/validate/swap orchestration** - `ca7fbb9` (feat)

_No plan-metadata commit needed beyond this SUMMARY/STATE/ROADMAP commit (below)._

## Files Created/Modified
- `internal/migrate/migrate.go` - `Run`, `Result`, `Options` (reused from validate.go, not redeclared), `resolveSourceDB`, `partialDir`, `checkTargetOverwrite`, `readProgress`, `resumePosition`, `finishFromComplete`, `getStoreMeta`, `seedNodeFilePath`, `batchWriter`/`newBatchWriter`/`commitData`/`advanceCursor`, `advanceCursorChecked`, `scanAndWriteTable`, `recomputeFileEdgeCounts`, `countFileIndexEdges`, `healthMessage`, `batchSize`/`partialStoreName`/`migrateTableOrder` constants, `testStopAfterBatch`/`errTestInterrupted` (test-only seam)
- `internal/migrate/migrate_test.go` - `runTarget`/`openTargetStore` helpers plus `TestRun_Happy`, `TestRun_NodesBeforeEdgesOwnerPath`, `TestRun_SourceUnmodified`, `TestRun_Resume`, `TestRun_AgedDB`, `TestRun_DanglingFailsLoud`, `TestRun_DropDangling` — reuses `hashFile` (reader_test.go), `exists` (swap_test.go), `scanDangling`/`Report` (validate.go) rather than redefining them

## Decisions Made
- `partialDir` mints a **fixed** name (`.codegraph.migrate-partial`) under the target's parent directory rather than reusing swap.go's `siblingTempDir` (which mints a fresh random name every call) — resumability across process invocations requires the SAME path be found on a re-run; `siblingTempDir` remains available for other randomly-named use cases but is not called from `Run`. This is a deliberate divergence from a literal reading of "reuse siblingTempDir," required by the plan's own explicit resumability instruction ("a re-run after a crash finds the partial store").
- `checkTargetOverwrite` (D-08's "recognizably a prior healthy migration" check) opens `<target>/store`, not `<target>` directly — confirmed against `internal/cli/init.go`'s `storeDirName = "store"` convention (`.codegraph/store/` is the established Pebble-store subdirectory layout used by `init`/`index`/`sync`/`serve`). An earlier draft opened `<target>` directly, which the test suite caught immediately (`GetMeta: graphstore: not found`) since the swapped-in store actually lives one level deeper.
- Table processing order is `files -> nodes -> edges`, matching the plan's `<action>` text and `must_haves.truths` verbatim; `files` has no `ownerPath` dependency on `nodes`, so its position relative to `nodes` doesn't affect D-04 correctness — it's simply the first table scanned.
- A `Progress` found with `Status=complete` short-circuits to `finishFromComplete`: this covers the case where a prior `Run` finished writing, validating, and stamping `Meta.healthy=true`, but the process died before `atomicSwapDir` ran. No re-write or re-validate is needed or performed.
- `recomputeFileEdgeCounts` runs unconditionally after the edges write phase completes (never gated on resume state), because it derives `edge_count` from the `x/` file-index's actual on-disk contents rather than any per-run in-memory accumulator — this is correct whether those `x/` entries were written in the current invocation or a prior interrupted one, without needing to persist or reconstruct edge-count bookkeeping across a resume boundary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `checkTargetOverwrite` opened the wrong directory level**
- **Found during:** Task 1 (initial test run against `TestRun_Happy`)
- **Issue:** The first implementation pass's `checkTargetOverwrite` (D-08's non-empty-target guard) and the test helper `openTargetStore` both opened `<target>` directly via `graphstore.Open`, rather than `<target>/store`. This mismatched the established `.codegraph/store/` subdirectory convention (`internal/cli/init.go`'s `storeDirName`) that `atomicSwapDir(tmpDir, target)` actually produces — `tmpDir` contains a `store/` subdir per the plan's own step-3 instruction, and swapping `tmpDir` into `target` means the real Pebble data ends up at `target/store`, not `target` itself.
- **Fix:** Corrected both `checkTargetOverwrite`'s prior-migration probe and the test's `openTargetStore` helper to open `filepath.Join(target, "store")`.
- **Files modified:** internal/migrate/migrate.go, internal/migrate/migrate_test.go
- **Verification:** `TestRun_Happy` and the other five `TestRun_*` tests failing with `GetMeta: graphstore: not found` before the fix, passing after
- **Committed in:** ca7fbb9 (part of the single task commit — caught and fixed before committing, not a separate commit)

**2. [Rule 1 - Bug] Unchecked `Close()` return values flagged by golangci-lint's errcheck**
- **Found during:** Task 1 (post-implementation lint pass)
- **Issue:** Several non-`defer` `Close()` calls on early-return error paths (`mw.Close()`, `pw.Close()`, `r.Close()`, `store.Close()` inside `checkTargetOverwrite`, plus a few in `recomputeFileEdgeCounts`, and `nit.Close()`/`fit.Close()`/`eit.Close()` in the test file) had their error return values silently discarded, which `errcheck` flags.
- **Fix:** Made the discard explicit (`_ = x.Close()`) at each non-`defer` call site, matching the codebase's existing convention where discards are intentional best-effort cleanup on an already-erroring path. `defer x.Close()` call sites were left as unchecked-`defer` — this exact pattern is already established throughout `reader.go`/`validate.go`/`swap.go` from prior 07-xx plans and is out of this task's scope to change.
- **Files modified:** internal/migrate/migrate.go, internal/migrate/migrate_test.go
- **Verification:** `golangci-lint run ./internal/migrate/` shows zero remaining non-`defer` errcheck findings in migrate.go/migrate_test.go after the fix (only pre-existing `defer`-pattern findings in other 07-xx files remain, out of scope)
- **Committed in:** ca7fbb9 (part of the single task commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs, both caught and corrected before the task commit)
**Impact on plan:** Both fixes were necessary for correctness (D1 was a real store-path bug that would have made every downstream reader/CLI consumer unable to find the migrated data) and code quality (D2). No scope creep — no pre-existing lint findings in other 07-xx files were touched, per the scope-boundary rule.

## Issues Encountered
None beyond the two auto-fixes documented above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `internal/migrate.Run`/`Result`/`Options` are ready for 07-07's CLI wiring (`codegraph migrate [--from][--to][--force][--drop-dangling]`) to call directly — `Run` already resolves both a `.codegraph/` source directory and a direct `*.db` path, and already enforces D-08's non-empty-target-without---force guard, so the CLI layer's job is thin: flags -> `Options{}` -> `Run` -> print `Result`/`Report`.
- The `.codegraph/store/` two-level layout convention (`target/store`, not `target` directly) is now load-bearing in `migrate.go` and confirmed against `internal/cli/init.go` — 07-07 should reuse `target` as the CLI's own `--to` default resolution (repo-root `.codegraph/`), not `target/store`.
- No blockers for 07-07.

---
*Phase: 07-migration-tool*
*Completed: 2026-07-13*

## Self-Check: PASSED

All created files and task commit hashes verified present on disk / in git log.
