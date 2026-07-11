# Phase 4: Incremental Sync & File Watcher - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-11
**Phase:** 4-Incremental Sync & File Watcher
**Mode:** `--auto` (fully autonomous — recommended default selected for every gray area, no interactive prompts)
**Areas discussed:** Incremental sync engine, Subgraph pruning & keyspace, Rename/move handling, Native watcher & debounce, Daemon/lock/unlock, Staleness surface, MCP reconnect reconciliation, Leak-free verification

---

## Incremental Sync Engine (INDX-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse two-pass internals | `sync` delegates to `indexer.Sync()` reusing extract/resolve/symbolindex/discover | ✓ |
| Standalone incremental path | Separate incremental indexer independent of the from-scratch pipeline | |

| Option | Description | Selected |
|--------|-------------|----------|
| stat pre-filter → content-hash confirm | mtime/size shortlist, then SHA-256 `File.content_hash` confirm; reparse only hash-changed | ✓ |
| content-hash only | Hash every file every sync | |

**Selected:** Reuse internals + stat-then-hash change detection (D-01/D-01a/D-01b).
**Notes:** Keeps one indexing code path (preserves Phase-2 determinism); stat pre-filter avoids hashing every file; the same reconcile routine backs SYNC-03.

---

## Subgraph Pruning & Keyspace (INDX-04) — load-bearing

| Option | Description | Selected |
|--------|-------------|----------|
| File-owned secondary index | Additive namespace so `DeleteFileSubgraph` range-deletes a file's nodes+edges (O(subgraph)) | ✓ |
| Query-time full scan | Scan all nodes filtered by `file_path` + scan all edges each sync | |

| Option | Description | Selected |
|--------|-------------|----------|
| Recompute dependents (query-time reverse scan) | Reuse Phase-3 D-04 to find files with edges into changed symbols; re-resolve them | ✓ |
| Persisted reverse-edge index | Maintain reverse index to delete in-edges on node deletion | |

| Option | Description | Selected |
|--------|-------------|----------|
| One-time backfill for old graphs | `sync` on a pre-Phase-4 index full-reindexes once to populate the secondary index | ✓ |
| Fail-hard on un-indexed graphs | Refuse to sync graphs built before the secondary index | |

**Selected:** Additive file-owned secondary index + dependent recomputation + backfill fallback (D-02/D-02a/D-02b).
**Notes:** `keys.go:113-118` names extending `DeleteFileSubgraph` as *the Phase-4 hook*; SchemaVersion stays 1 (additive namespace). Dangling edges prevented by authoritative re-resolve, not a bespoke reverse-delete. Persisted reverse index stays Phase 8.

---

## Rename / Move Handling (INDX-04)

| Option | Description | Selected |
|--------|-------------|----------|
| delete-old + add-new (content-hash id stability) | Rename = prune old path + index new path; stable `n/<id>` keeps unmoved-content edges | ✓ |
| Explicit rename-detection heuristic | Detect renames via similarity and rewrite keys in place | |

**Selected:** delete-old + add-new leaning on content-hashed node ids (D-03).
**Notes:** nodeid scheme already makes "same symbol moved" stable; fixture suite asserts no orphaned nodes / dangling edges across create/modify/delete/rename/move.

---

## Native Watcher & Debounce (SYNC-01, SYNC-02)

| Option | Description | Selected |
|--------|-------------|----------|
| fsnotify native + recursive-add-on-Create | Per-OS events, coalesce burst → one sync over union of paths, 2000ms env-tunable debounce | ✓ |
| Polling walker | Periodic full-tree stat scan | |

| Option | Description | Selected |
|--------|-------------|----------|
| Banner in explore + status signal | Staleness warning prepended to `explore`, `stale`/`pendingSync` in `status` | ✓ |
| status-only | Staleness only in `status` | |

**Selected:** fsnotify + recursive-add + debounce; banner in explore + status (D-04/D-04a).
**Notes:** Polling rejected by REQUIREMENTS "Out of Scope: Polling as default"; explore is the flagship agent surface; same formatter feeds MCP.

---

## Daemon, Lock & unlock (SYNC-04, SYNC-05)

| Option | Description | Selected |
|--------|-------------|----------|
| Daemon + lockfile + in-process fallback | `codegraph daemon` owns watcher+writer under a pid+timestamp lock; watcher-in-`serve` fallback; `unlock` clears stale locks | ✓ |
| Watcher only inside serve | No dedicated daemon process | |

**Selected:** Daemon + lockfile + in-process fallback (D-05).
**Notes:** SYNC-04 requires a shared server with documented fallback; SYNC-05 requires `unlock`; `unlock` checks pid liveness before clearing.

---

## MCP Reconnect Reconciliation (SYNC-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Reconcile on reconnect | Run stat+content-hash `Sync` before serving tools on (re)connect/init | ✓ |
| Rely solely on live watcher | Trust the watcher to have caught everything | |

**Selected:** Reconcile on reconnect (D-06).
**Notes:** Catches offline changes made while the watcher/daemon was down; reuses the `sync` reconcile routine.

---

## Leak-Free Verification (SYNC-06)

| Option | Description | Selected |
|--------|-------------|----------|
| context+WaitGroup lifecycle + soak gate | Cancelable-context-owned goroutines joined on shutdown; goleak/NumGoroutine soak test | ✓ |
| Manual NumGoroutine delta only | Ad-hoc goroutine count check | |

**Selected:** context+WaitGroup lifecycle with a goleak/NumGoroutine soak gate (D-07).
**Notes:** Standard Go leak-detection; exact library is executor's discretion.

---

## Claude's Discretion

- Secondary-index key shape (`fx/<path>/<id>` vs owning-file-prefixed keys) and backfill-detection mechanism.
- Sync writer batch granularity; whether to use the CGo incremental-reparse path vs full reparse.
- Debounce data structure / timer reset; lockfile format; daemon foreground/background + IPC.
- Staleness banner text/placement and `status` field name; `sync --json` summary shape.
- Soak-test iteration counts and leak-detection library.

## Deferred Ideas

- Persisted reverse-edge index — Phase 8.
- Interface→impl dispatch synthesis, `provenance: heuristic`, framework routes, non-Go languages, and the Phase-5 Go resolution gaps WR-01/WR-02 — Phase 5.
- 100k-file scale / peak-RSS / perf-regression gates — Phase 8.
- HTTP/SSE MCP transport, remote/multi-user daemon — v2 (SERVER-01).
- `install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry` — Phase 6.
- TS SQLite migration — Phase 7.
