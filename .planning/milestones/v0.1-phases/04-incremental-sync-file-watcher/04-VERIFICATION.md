---
phase: 04-incremental-sync-file-watcher
verified: 2026-07-11T19:50:00Z
status: passed
score: 8/8 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 4: Incremental Sync & File Watcher Verification Report

**Phase Goal:** The graph stays current automatically as files change — correct pruning on any file operation, debounced native watching, a shared daemon, and no goroutine leaks.
**Verified:** 2026-07-11T19:50:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `codegraph sync` reparses only content-hash-changed files, with dependent-edge recomputation (not full re-index) | ✓ VERIFIED | `internal/indexer/sync.go` D-01a stat-prefilter→hash-confirm; `TestSyncResolvesAcrossUnchangedFiles`, `TestSyncEqualsReindex` pass |
| 2 | File create/modify/delete/rename/move prune with no orphaned nodes or dangling edges | ✓ VERIFIED | `TestPruneFixtures` (create/modify/delete/rename/move subtests) all PASS; CR-03/CR-04 cross-file/dependent-edge gaps fixed and regression-tested (`TestSyncIndexesCrossFileContainsEdgeUnderOwnerFile`, `TestSyncPruneOwnedEdgesRemovesStaleFileIndexEntry`) |
| 3 | Native per-OS debounced watcher (fsnotify) consolidates edit bursts into one sync; agent output shows a staleness banner while pending | ✓ VERIFIED | `internal/watch/watcher.go`+`debounce.go` (fsnotify, recursive add-on-Create, `DebounceDuration()` env-tunable 2000ms default); `TestDebounceCoalescesBurst`, `TestDebounceEnvTunable` pass. Banner: `internal/query/render_markdown.go:staleBanner`, wired through `RenderExplore`/`Explore`; `internal/query/status.go:computeStale` (sidecar + mtime fallback); tests pass |
| 4 | On MCP (re)connect, offline changes are reconciled via stat+content-hash before serving | ✓ VERIFIED | `internal/cli/serve.go` calls `indexer.Sync(...)` when `hasIndex` before `mcp.BuildServer`/`ServeStdio`; `TestReconnectReconcile` PASS |
| 5 | `codegraph daemon` runs a shared watch/index server (in-process fallback via `serve --watch`); single-writer invariant holds even under concurrent acquire | ✓ VERIFIED | `internal/daemon/daemon.go` (Run/flush/syncMu), `internal/cli/daemon.go`, `internal/cli/serve.go --watch`; CR-01 (untracked flush goroutine outliving lock release) fixed via `Debouncer.Wait()` + regression test `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` PASS; CR-02 (acquire() TOCTOU) fixed via atomic `os.Link` exclusive-create + regression test `TestAcquireConcurrentRaceOnlyOneWinner` PASS |
| 6 | `codegraph unlock` clears only genuinely-stale locks (pid liveness), refuses a live daemon's lock | ✓ VERIFIED | `internal/daemon/lock.go:Unlock`/`isStale`/`isProcessLive`; `internal/cli/unlock.go` wired in `root.go`; `lock_test.go` suite passes |
| 7 | Long-running watcher/daemon is goroutine-leak-free (soak-verified) | ✓ VERIFIED | `internal/watch/soak_test.go`, `internal/daemon/soak_test.go` both use `goleak.VerifyTestMain`; `TestSoak` in both packages PASS within timeout (well under 120s) |
| 8 | `sync`/`daemon`/`unlock` are real, wired CLI commands (not stubs) | ✓ VERIFIED | `internal/cli/{sync,daemon,unlock}.go` implement `newSyncCmd`/`newDaemonCmd`/`newUnlockCmd`, registered in `root.go:46-47` |

**Score:** 8/8 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/graphstore/keys.go`, `store.go`, `batch.go` | `x/` file-owned secondary index namespace + `DeleteFileIndexEdge` | ✓ VERIFIED | `Writer.DeleteFileIndexEdge` implemented and declared on interface (store.go:214, batch.go:75) |
| `internal/indexer/sync.go` | Incremental `Sync()` engine, dependent recomputation, `ownerPathFor` full-graph fallback | ✓ VERIFIED | 574+ lines, substantive; CR-03 fix present (line 574-...) |
| `internal/indexer/prune_fixtures_test.go` + `testdata/gofixture` | create/modify/delete/rename/move fixture suite | ✓ VERIFIED | `TestPruneFixtures` with 5 subtests, all pass |
| `internal/watch/watcher.go`, `debounce.go` | fsnotify recursive watcher + debounce | ✓ VERIFIED | Wired, tested (`TestDebounce*`) |
| `internal/query/status.go`, `render_markdown.go` | live staleness signal + banner | ✓ VERIFIED | `computeStale`, `staleBanner`, threaded through `Explore` |
| `internal/mcp/reconnect_test.go` | reconnect reconciliation test | ✓ VERIFIED | `TestReconnectReconcile` PASS |
| `internal/daemon/lock.go`, `daemon.go` | lockfile + shared single-writer daemon | ✓ VERIFIED | CR-01/CR-02/WR-02 fixes present with regression tests |
| `internal/cli/sync.go`, `daemon.go`, `unlock.go` | thin CLI commands | ✓ VERIFIED | All registered in `root.go` |
| `internal/watch/soak_test.go`, `internal/daemon/soak_test.go` | goleak soak gate | ✓ VERIFIED | `TestSoak` present, `goleak.VerifyTestMain`, passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/serve.go` | `internal/indexer.Sync` | reconcile before `mcp.BuildServer`/`ServeStdio` | ✓ WIRED | `serve.go:70` calls `indexer.Sync` guarded by `hasIndex` |
| `internal/cli/status.go` | `internal/query/status.go` | live `Stale` field surfaced text+JSON | ✓ WIRED | `Stale bool` on `StatusResult`, `MarshalStatusJSON` includes it |
| `internal/query/explore.go` | `internal/query/render_markdown.go` | `stale` param threaded to `RenderExplore` | ✓ WIRED | `explore.go:158` passes `stale` |
| `internal/daemon/daemon.go` | `internal/watch/debounce.go` | `deb.Wait()` joins in-flight flush before lock release (CR-01) | ✓ WIRED | `daemon.go:154` |
| `internal/indexer/sync.go` | `internal/graphstore` `Writer.DeleteFileIndexEdge` | dependent-edge prune also clears `x/` entry (CR-04) | ✓ WIRED | `sync.go:554` |
| `internal/indexer/sync.go` | `graphstore.Reader.GetNode` | cross-file `contains`-edge owner fallback (CR-03) | ✓ WIRED | `sync.go:574` `ownerPathFor` |
| `internal/cli/root.go` | `newSyncCmd`/`newDaemonCmd`/`newUnlockCmd` | command registration | ✓ WIRED | `root.go:46-47` |

### Behavioral Spot-Checks / Targeted Tests

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full suite green (race) | `go test ./... -race -count=1` | all packages `ok` | ✓ PASS |
| CR-01 regression | `go test ./internal/daemon/... -run TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock -race -count=1` | PASS | ✓ PASS |
| CR-02 regression | `go test ./internal/daemon/... -run TestAcquireConcurrentRaceOnlyOneWinner -race -count=1` | PASS | ✓ PASS |
| CR-03 regression | `go test ./internal/indexer/... -run TestSyncIndexesCrossFileContainsEdgeUnderOwnerFile -race -count=1` | PASS | ✓ PASS |
| CR-04 regression | `go test ./internal/indexer/... -run TestSyncPruneOwnedEdgesRemovesStaleFileIndexEntry -race -count=1` | PASS | ✓ PASS |
| INDX-04 fixture suite | `go test ./internal/indexer/... -run TestPruneFixtures -count=1 -v` | 5/5 subtests PASS | ✓ PASS |
| Sync-equals-reindex determinism | `go test ./internal/indexer/... -run TestSyncEqualsReindex -count=1` | PASS | ✓ PASS |
| Debounce coalescing | `go test ./internal/watch/... -run TestDebounce -count=1 -v` | 4/4 tests PASS | ✓ PASS |
| MCP reconnect reconcile | `go test ./internal/mcp/... -run TestReconnectReconcile -count=1 -v` | PASS | ✓ PASS |
| Soak (goleak) | `go test ./internal/watch/... ./internal/daemon/... -run TestSoak -timeout 120s -count=1 -v` | 2/2 PASS, ~2s each | ✓ PASS |
| Daemon flake re-check | `go test ./internal/daemon/... -race -count=3` | ok, no reproduction | ✓ PASS (matches orchestrator's low-risk note) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| INDX-03 | 04-02, 04-03, 04-04, 04-08 | Incremental sync via content-hash diffing + dependent-edge recomputation | ✓ SATISFIED | `sync.go`, `TestSyncResolvesAcrossUnchangedFiles`, `TestSyncEqualsReindex` |
| INDX-04 | 04-01, 04-04 | Rename/delete/move prune, no orphans/dangling edges | ✓ SATISFIED | `x/` index (keys.go/batch.go), `TestPruneFixtures`, CR-03/CR-04 fixes |
| SYNC-01 | 04-05 | Native debounced watcher, default 2000ms tunable | ✓ SATISFIED | `internal/watch`, `TestDebounce*` |
| SYNC-02 | 04-06 | Staleness banner in agent-facing output | ✓ SATISFIED | `status.go`/`render_markdown.go`, tests pass. Note: `REQUIREMENTS.md` traceability table still shows this row as "Pending" — a doc-sync lag, not a code gap (see Gaps Summary) |
| SYNC-03 | 04-06 | MCP reconnect reconciles offline changes | ✓ SATISFIED | `serve.go` reconcile, `TestReconnectReconcile`. Same doc-sync note as SYNC-02 |
| SYNC-04 | 04-07, 04-08 | `codegraph daemon` shared watch/index server, in-process fallback | ✓ SATISFIED | `daemon.go`, `cli/daemon.go`, `serve.go --watch` |
| SYNC-05 | 04-07, 04-08 | `codegraph unlock` clears only stale locks | ✓ SATISFIED | `lock.go:Unlock`, `cli/unlock.go` |
| SYNC-06 | 04-09 | Goroutine-leak-free soak | ✓ SATISFIED | `soak_test.go` x2, `goleak.VerifyTestMain` |

No orphaned requirements — all 8 requirement IDs declared in the phase directive are claimed by at least one plan and cross-referenced in ROADMAP.md's Phase 4 success criteria.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the 26 files this phase touched (per `04-REVIEW.md`'s file list). No empty stub implementations found in CLI commands, daemon, watcher, or sync engine.

### Deep Code Review Follow-Up (04-REVIEW.md)

`04-REVIEW.md` (deep review, 26 files) found 4 Critical + 5 Warning + 2 Info findings that the green test suite at the time did not catch (concurrency races and multi-generation incremental-sync edge cases). All 4 Criticals and all 5 Warnings were fixed at root cause in follow-up commits, each with a fail-before/pass-after regression test:

| ID | Issue | Fix Commit | Regression Test | Verified |
|----|-------|-----------|------------------|----------|
| CR-01 | `Daemon.Run` could release lock while a debounced flush's `Sync` was still writing | `9ffbfe1` | `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` | ✓ PASS |
| CR-02 | `lock.go acquire()` TOCTOU — two daemons could both "win" | `8ce9497` | `TestAcquireConcurrentRaceOnlyOneWinner` | ✓ PASS |
| CR-03 | Cross-file `contains` edge dropped its `x/` owner entry, becoming unprunable | `d69f905` | `TestSyncIndexesCrossFileContainsEdgeUnderOwnerFile` | ✓ PASS |
| CR-04 | Dependent-edge prune deleted `e/` records but not matching `x/` entries — index drift | `d8b1957` | `TestSyncPruneOwnedEdgesRemovesStaleFileIndexEntry` | ✓ PASS |
| WR-01 | `pebbleStore` closed-check TOCTOU vs `Close()` | `bcb4e4d` | RWMutex guard added, existing suite green | ✓ VERIFIED (code) |
| WR-02 | `lockInfo.StartedAt` never consulted (PID-reuse risk) | `8ce9497` | `isStale` now cross-checks `processStartTime` | ✓ VERIFIED (code) |
| WR-03 | Hash-equal-but-stat-differs files never refresh stored mtime | `2bf79da` | fix present, suite green | ✓ VERIFIED (code) |
| WR-04 | `Daemon.opts` never populated | `29b8d1a` | `daemon.New(repoRoot, opts)` now threaded from `cli/daemon.go` | ✓ VERIFIED (code) |
| WR-05 | `dependentPaths` could include already-deleted files | `e8c087a` | fix present | ✓ VERIFIED (code) |
| IN-01 | dead `Watcher.root` field | `bf339b6` | field removed (`grep` confirms no `.root` reference remains) | ✓ VERIFIED (code) |

All fixes confirmed present in the current codebase (not just claimed in commit messages) and the full race-enabled test suite (`go test ./... -race -count=1`) is green after all fixes.

### Human Verification Required

None. All must-haves resolved to VERIFIED via automated tests and direct code inspection; no visual/UX/external-service-dependent behavior in this phase's scope.

### Gaps Summary

No blocking gaps. One informational note:

- `REQUIREMENTS.md`'s per-requirement traceability table (lines ~152-153) still marks `SYNC-02` and `SYNC-03` as "Pending" even though the phase's checklist section (lines 43-45) has them checked `[x]` and the code/tests fully satisfy both (verified above: `status.go`/`render_markdown.go` staleness signal+banner, `serve.go` MCP reconnect reconcile, both test-covered and passing). This is a stale traceability-table row, not a functional gap — recommend updating `REQUIREMENTS.md`'s table to "Complete" for SYNC-02/SYNC-03 as routine housekeeping, but it does not block phase completion.

- One low-risk watch item carried forward from the orchestrator's notes: a single, non-reproducing timing flake was observed once in `internal/daemon` under full load. Re-run here (`-race -count=3`) did not reproduce it and produced no DATA RACE/panic signature. Not a blocker; worth keeping an eye on in CI over time.

---

_Verified: 2026-07-11T19:50:00Z_
_Verifier: Claude (gsd-verifier)_
