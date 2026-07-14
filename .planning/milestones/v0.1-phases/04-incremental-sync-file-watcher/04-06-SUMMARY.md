---
phase: 04-incremental-sync-file-watcher
plan: 06
subsystem: query/mcp
tags: [staleness, banner, mcp-reconnect, reconcile, sync-02, sync-03, go]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "internal/indexer.Sync(repoRoot, storeDir, opts) from Plan 04-03 (the D-06 reconcile entry) and File.mtime_unix_ns / Meta.last_sync_unix_ms from Plans 04-01/04-02"
provides:
  - "internal/query StatusResult.Stale live staleness signal (sidecar OR newest-mtime-vs-last_sync fallback)"
  - "RenderExplore staleness banner prepended to explore markdown when a sync is pending (D-04a)"
  - "MCP serve startup reconcile: indexer.Sync runs before mcp.BuildServer when an index exists (D-06 / SYNC-03)"
affects: [04-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "one formatter path: the staleness banner threads through RenderExplore so the MCP explore tool shows it too (Phase-3 D-08b: CLI and MCP share the formatter)"
    - "reconcile-before-serve: the same indexer.Sync entry used by codegraph sync/daemon runs once on MCP serve startup, catching offline changes before the first tool is served; no second reconcile code path"
    - ".codegraph/.sync-pending sidecar (daemon-set) with a newest-source-mtime > Meta.last_sync_unix_ms fallback when no daemon is running (RESEARCH Pattern 4 / Open Question 2)"

key-files:
  created:
    - internal/mcp/reconnect_test.go
  modified:
    - internal/query/status.go
    - internal/query/render_markdown.go
    - internal/query/explore.go
    - internal/query/status_staleness_test.go
    - internal/query/render_markdown_test.go
    - internal/indexer/resolve.go
    - internal/cli/serve.go
    - internal/cli/status.go

key-decisions:
  - "Staleness is a boolean/one-line signal only (StatusResult.Stale + a single bolded banner line) — the information-disclosure threat (T-04-06-02) is accepted as low because status already renders projectPath/indexPath empty and the fallback reads only file metadata within the confined repo root"
  - "An orphaned .codegraph/.sync-pending sidecar (daemon crash) at worst renders a harmless stale banner until the next successful sync removes it; the mtime-vs-last_sync fallback self-corrects (accepted DoS T-04-06-03)"
  - "MCP-03 absent-index branch is unchanged: the reconcile Sync is skipped when no .codegraph/ resolves, so the zero-tools fallback still holds"

patterns-established:
  - "Pattern: agent-facing staleness surfaces through the single query formatter (explore markdown + status --json/text), never a bespoke MCP-only path"

requirements-completed: [SYNC-02, SYNC-03]

coverage:
  - id: D1
    description: "status reports a live Stale/PendingSync signal: true when the .codegraph/.sync-pending sidecar exists OR (no-daemon fallback) newest source-file mtime > Meta.last_sync_unix_ms; false otherwise"
    requirement: "SYNC-02"
    verification:
      - kind: unit
        ref: "internal/query/status_staleness_test.go (sidecar-present, mtime-fallback, fresh-not-stale)"
        status: pass
    human_judgment: false
  - id: D2
    description: "explore markdown prepends a single bolded staleness banner before the Exploration header when stale, and nothing when current (D-04a); MCP explore tool shows it via the shared formatter"
    requirement: "SYNC-02"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go (banner present/absent)"
        status: pass
    human_judgment: false
  - id: D3
    description: "On MCP serve startup, indexer.Sync reconciles offline changes (stat + content hash) before the first tool is served; a no-change reconcile reparses zero files"
    requirement: "SYNC-03"
    verification:
      - kind: integration
        ref: "internal/mcp/reconnect_test.go#TestReconnectReconcile"
        status: pass
    human_judgment: false

## Self-Check: PASSED

`go build ./...` clean; `go test ./internal/query/... ./internal/cli/... ./internal/mcp/... -race -count=1` all green.

## Notes

Commits: `9a12ec6` (Task 1 — signed) and `ff86d64` (Task 2 — unsigned). Task 2's signed commit was blocked mid-run by 1Password's `op-ssh-sign` helper refusing git's sign requests (`1Password: agent returned an error`), an environment/auth condition, not a code issue. Per an explicit user decision (recorded as engram rule `xmz3xknbj0`), agent/pipeline commits use a repo-local `commit.gpgsign=false` override for the remainder of Phase 4; the override is unset at phase end so the user's global signed-commit default is restored. Code was complete and tests-green before the commit unblock.
