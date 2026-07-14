---
phase: 4
slug: incremental-sync-file-watcher
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-11
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `04-RESEARCH.md` §Validation Architecture. The planner maps each
> task to a requirement row below; the nyquist-auditor fills per-task-ID rows.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` (`go test`) — matches every Phase 1–3 `*_test.go` file; no external framework |
| **Config file** | none — `go test ./...` is the project convention |
| **Quick run command** | `go test ./internal/indexer/... ./internal/graphstore/... -run TestSync -count=1` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~30–120 s (soak test dominates; `-timeout 120s`) |

---

## Sampling Rate

- **After every task commit:** Run the targeted package test (e.g. `go test ./internal/indexer/... -run TestSync -count=1`)
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite green, including the soak test
- **Max feedback latency:** ~120 s (soak); ~5 s for quick per-task runs

---

## Per-Requirement Verification Map

*(Task-ID granularity filled by the planner/nyquist-auditor once PLAN.md tasks exist. Each row is a success-criterion-level contract from 04-RESEARCH.md.)*

| Requirement | Behavior | Test Type | Automated Command | File (Wave 0) | Status |
|-------------|----------|-----------|-------------------|---------------|--------|
| INDX-03 | `sync` reparses only changed files + dependent-edge recomputation (store-seeded symbol index) | unit + property | `go test ./internal/indexer/... -run TestSync -count=1` | `internal/indexer/sync_test.go` ❌ | ⬜ pending |
| INDX-03 | sync-equals-reindex determinism (byte-identical `Export()` after normalizing `last_sync_unix_ms`) | property | `go test ./internal/indexer/... -run TestSyncEqualsReindex -count=1` | `internal/indexer/sync_determinism_test.go` ❌ | ⬜ pending |
| INDX-04 | rename/delete/move fixture suite — no orphaned nodes / dangling edges | fixture integration | `go test ./internal/indexer/... -run TestPruneFixtures -count=1` | `internal/indexer/prune_fixtures_test.go` + `testdata/gofixture` ❌ | ⬜ pending |
| INDX-04 | `x/` file-index namespace key encoding + `IterateFileIndex`/`DeleteNode`/`DeleteEdge` | unit | `go test ./internal/graphstore/... -run TestFileIndex -count=1` | `internal/graphstore/fileindex_test.go` ❌ | ⬜ pending |
| SYNC-01 | debounced watcher coalesces an edit burst into one sync | integration (real fsnotify, temp dir) | `go test ./internal/watch/... -run TestDebounce -count=1` | `internal/watch/debounce_test.go` ❌ | ⬜ pending |
| SYNC-02 | staleness banner in `explore`/`status` while a sync is pending | unit (formatter-level) | `go test ./internal/query/... -run TestStalenessBanner -count=1` | `internal/query/status_staleness_test.go` ❌ | ⬜ pending |
| SYNC-03 | MCP reconnect reconciles offline changes before serving | integration | `go test ./internal/mcp/... -run TestReconnectReconcile -count=1` | `internal/mcp/reconnect_test.go` ❌ | ⬜ pending |
| SYNC-04 | `daemon` shared watch/index server + in-process fallback (single writer) | integration | `go test ./internal/daemon/... -run TestDaemonSharedWriter -count=1` | `internal/daemon/daemon_test.go` ❌ | ⬜ pending |
| SYNC-05 | `unlock` clears only genuinely-stale locks (pid liveness) | unit | `go test ./internal/daemon/... -run TestUnlockStaleOnly -count=1` | `internal/daemon/lock_test.go` ❌ | ⬜ pending |
| SYNC-06 | goroutine-leak-free soak | soak (`goleak.VerifyTestMain`) | `go test ./internal/watch/... ./internal/daemon/... -run TestSoak -timeout 120s -count=1` | `internal/watch/soak_test.go` ❌ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/indexer/sync_test.go` — INDX-03 (Sync algorithm, store-seeded index, dependent recomputation)
- [ ] `internal/indexer/sync_determinism_test.go` — INDX-03 sync-equals-reindex property gate
- [ ] `internal/indexer/prune_fixtures_test.go` + extended `testdata/gofixture` (create/modify/delete/rename/move) — INDX-04
- [ ] `internal/graphstore/fileindex_test.go` — the new `x/` namespace key encoding + `IterateFileIndex`/`DeleteNode`/`DeleteEdge` (mirror `keyenc_test.go` collision-safety style)
- [ ] `internal/watch/watcher_test.go`, `internal/watch/debounce_test.go` — new package, SYNC-01
- [ ] `internal/query/status_staleness_test.go`, extend `render_markdown_test.go` — SYNC-02
- [ ] `internal/mcp/reconnect_test.go` — SYNC-03
- [ ] `internal/daemon/lock_test.go`, `internal/daemon/daemon_test.go` — new package, SYNC-04/SYNC-05
- [ ] `internal/watch/soak_test.go` (or shared `internal/daemon/soak_test.go`) with `goleak.VerifyTestMain` — SYNC-06
- [ ] Dependency install: `go get github.com/fsnotify/fsnotify@v1.10.1 && go get -t go.uber.org/goleak@v1.3.0`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cross-OS native watcher backends (FSEvents/inotify/ReadDirectoryChangesW) | SYNC-01 | CI runs one OS per job; per-OS event semantics can't all be asserted in one `go test` run | Run `go test ./internal/watch/...` on macOS, Linux, and Windows CI matrix legs; assert debounce+coalesce on each |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
