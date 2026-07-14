# Phase 4: Incremental Sync & File Watcher - Context

**Gathered:** 2026-07-11
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase makes the **frozen from-scratch graph self-maintaining**. Phases 2–3 build a correct graph once and read it; Phase 4 turns the from-scratch indexer into an **incremental** one and keeps the graph current automatically as files change — without a full re-index and without orphaned/dangling graph state.

**In scope:**
1. **`codegraph sync`** (INDX-03) — incrementally reparse only content-hash-changed files (stat pre-filter → `File.content_hash` confirm), pruning stale symbols/edges and recomputing dependent-file edges. **Not** a full re-index of a subset.
2. **Correct subgraph pruning** (INDX-04) — file renames, deletes, and moves prune stale symbols and edges with **no orphaned nodes or dangling edges**, verified by a fixture suite. This requires closing the deliberately-deferred file→subgraph keying gap (`keys.go:113-118`).
3. **Native debounced watching** (SYNC-01, SYNC-02) — per-OS watchers (FSEvents/inotify/ReadDirectoryChangesW) via `fsnotify`, debounced (default 2000ms, env-tunable), coalescing edit bursts into one sync; a staleness banner in agent-facing output while a sync is pending.
4. **Shared daemon + lifecycle** (SYNC-04, SYNC-05, SYNC-06) — `codegraph daemon` (shared watch/index server, in-process fallback where unsupported), `codegraph unlock` (clears stale locks after a crash), and a **goroutine-leak-free** watcher/daemon proven by soak tests.
5. **MCP reconnect reconciliation** (SYNC-03) — on MCP server (re)connect, reconcile offline filesystem changes via stat comparison + content hashing before serving tools.

**Out of scope (belongs to later phases):**
- **Interface→impl dispatch synthesis, `provenance: heuristic` edges, framework `route` nodes, non-Go languages** — Phase 5 (RES-02/03, LANG-02+). Phase 4 syncs the **Go** ground-truth graph the current extractor produces; the two known Go resolution gaps (WR-01 same-package func/method collision, WR-02 selector-on-non-identifier) are **Phase 5** and MUST NOT be worked around inside sync.
- **Persisted reverse-edge index** — Phase 8 scale work. Phase 4 reuses Phase-3's **query-time** reverse-adjacency scan (D-04) for dependent detection; a persisted reverse index is adopted only if profiling on the 100k-file monorepo shows the scan dominates.
- **100k-file scale / peak-RSS bounding / performance-regression gates** — Phase 8 (INDX-06, PERF-*). Phase 4 targets **correctness** of incremental update and pruning on a real mid-size Go repo, plus leak-freedom; it does not tune monorepo-scale memory.
- **`install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry`** — Phase 6 (AGNT-*, CLI-*). Phase 4 ships `sync`, `daemon`, `unlock` only.
- **HTTP/SSE MCP transport, remote/multi-user daemon** — v2 (SERVER-01). The daemon here is a **local, single-user** shared process.
- **Migration of TS SQLite indexes** — Phase 7.

</domain>

<decisions>
## Implementation Decisions

> Auto-resolved in `--auto` mode (single pass). Each decision takes the recommended default grounded in: the Phase 1–3 substrate (`GraphStore`/`Writer`/`Reader`, `n/`·`e/`·`f/`·`m/` keyspace, content-hashed node ids, forward-only edges + query-time reverse scan), the schema seams already planted for this phase (`File.content_hash`, `Meta.last_sync_unix_ms`, `Writer.DeleteFileSubgraph`, `keys.fileSubgraphPrefix`), the ROADMAP Phase-4 success criteria + requirements (INDX-03/04, SYNC-01…06), and the technology-stack research in `.claude/CLAUDE.md` (fsnotify, Pebble range-deletes/snapshots, no-polling-default).

### Incremental Sync Engine (INDX-03)
- **D-01:** Add **`codegraph sync`** as a thin Cobra command (mirroring `init`/`index`/`uninit` in `internal/cli`) delegating to a new **`internal/indexer.Sync()`** incremental entry that **reuses the existing `extract`/`resolve`/`symbolindex`/`discover` internals** — not a second pipeline. `sync` resolves the nearest `.codegraph/` (same resolution idiom as the query commands, D-01a from Phase 3) and errors clearly if uninitialized.
  *[auto] Q: sync as reuse-of-two-pass-internals vs a standalone incremental path → Selected: reuse existing extract/resolve internals (recommended — keeps one indexing code path, preserves Phase-2 determinism).*
- **D-01a:** **Change detection = stat pre-filter → content-hash confirm.** Compare on-disk file `mtime`/`size` against the graph's per-file record to cheaply shortlist candidates, then confirm an actual change by recomputing the SHA-256 into `File.content_hash` and comparing. Only files whose **hash changed** are reparsed. This is exactly the "stat comparison + content hashing" SYNC-03 also needs, so the same reconcile routine backs both `sync` and MCP-reconnect (D-07).
  *[auto] Q: content-hash-only vs stat-then-hash → Selected: stat pre-filter then content-hash confirm (recommended — avoids hashing every file every sync; matches SYNC-03 wording).*
- **D-01b:** **`sync` flags mirror `index`** — `--quiet`/`--verbose`, and a concise default summary (files reparsed, files pruned, nodes/edges added/removed, dependents recomputed, duration); a `--json` machine summary is executor's discretion. `sync` writes through the **single coordinated `GraphStore.Writer`** in batched windows (never per-symbol), stamping `Meta.last_sync_unix_ms` on commit.

### Subgraph Pruning & Keyspace — the load-bearing decision (INDX-04)
- **D-02:** **Add a file-owned secondary index** so a file's nodes + outgoing edges can be pruned by range-delete. The store today keys nodes by content-hash (`n/<id>`) and edges forward-only (`e/<src>/<kind>/<dst>`) with **no file→subgraph mapping** — `keys.go:113-118` documents `DeleteFileSubgraph` as pruning only the `f/` record and names extending it to node/edge records as *the Phase-4 hook*. Introduce an **additive namespace** (e.g. `fx/<path>/…` reverse index, or owning-file-prefixed node/edge keys — exact shape executor's discretion) under the reserved-namespace mechanism (`keys.go:21`), populated at `init`/`index`/`sync` time, so `DeleteFileSubgraph(path)` becomes an O(subgraph) `DeleteRange`. **Additive keyspace only — `SchemaVersion` stays `1`** (a new key namespace is not a record-format break, per `internal/schema/meta.go` additive rule).
  *[auto] Q: file-owned secondary index (O(subgraph) DeleteRange) vs query-time full node/edge scan filtered by file_path → Selected: file-owned secondary index (recommended — it is the explicitly-designed `DeleteFileSubgraph`/`fileSubgraphPrefix` hook; makes prune tractable and avoids re-scanning the whole graph per sync).*
- **D-02a:** **Dangling-edge prevention via dependent recomputation, NOT a persisted reverse index.** Deleting file B's subgraph does not touch edges *into* B (e.g. `e/funcA/calls/funcB`) because they live under A's source key. Instead, each sync builds the **Phase-3 D-04 query-time reverse-adjacency map** once (`IterateEdges("")` over the snapshot) to find files that had an edge **targeting a symbol in a changed/deleted file**, adds them to the recompute set, and **re-resolves** them — so any edge to a now-gone symbol simply isn't regenerated (becomes an unresolved ref, no edge). This keeps dangling-edge cleanup a property of the authoritative two-pass resolve, not a bespoke reverse-delete. A **persisted** reverse index stays **Phase 8**.
  *[auto] Q: recompute dependents via query-time reverse scan vs maintain a persisted reverse-edge index to delete in-edges → Selected: recompute dependents (recommended — reuses Phase-3 D-04, no frozen-writer change, correctness-first for mid-size corpus).*
- **D-02b:** **Backfill / fallback for pre-Phase-4 graphs.** Graphs built by Phase 2/3 lack the D-02 secondary index. A `sync` that detects its absence (Meta flag or missing `fx/` namespace) performs a **one-time full re-index** to populate the index, then subsequent syncs are incremental. Detect-and-backfill is silent-safe and logged; it never produces a silently-wrong partial prune.
  *[auto] Q: fail-hard on un-indexed old graphs vs one-time full-reindex backfill → Selected: one-time backfill then incremental (recommended — no user-visible breakage for existing indexes).*

### Rename / Move Handling (INDX-04)
- **D-03:** **Treat rename/move as delete(old path) + add(new path)** at the file level. Content-hashed node ids (Phase-2 D-02 / `internal/indexer/nodeid`) mean a symbol whose **content is unchanged** keeps a stable `n/<id>`, so inbound edges to unmoved-content symbols regenerate identically and the graph doesn't churn on a pure move. The fixture suite (success-criterion 2) asserts **no orphaned nodes and no dangling edges** after each of: create, modify, delete, rename, and move — this fixture suite is the acceptance gate for INDX-04.
  *[auto] Q: rename detection heuristic vs delete-old + add-new → Selected: delete-old + add-new leaning on content-hash id stability (recommended — simplest correct model; nodeid scheme already makes "same symbol moved" stable).*

### Native Watcher & Debounce (SYNC-01, SYNC-02)
- **D-04:** **Build on `github.com/fsnotify/fsnotify`** (new direct dependency; per `.claude/CLAUDE.md` §fsnotify) with an **explicit recursive-add-on-Create loop** (fsnotify does not recurse). Coalesce a burst of events into **one `Sync`** over the **union of changed paths**; debounce **default 2000ms**, tunable via env (e.g. `CODEGRAPH_DEBOUNCE_MS`, per SYNC-01). Always ignore `.codegraph/` and honor `.gitignore`/the same discovery exclusions the indexer's `discover.go` already applies, so watcher and indexer agree on the file set. **No polling as default** — polling is only an explicit degraded fallback where native events are unsupported (REQUIREMENTS "Out of Scope: Polling as default").
  *[auto] Q: fsnotify native + recursive-add vs polling walker → Selected: fsnotify native with recursive-add-on-Create (recommended — mandated by CLAUDE.md; polling rejected by requirements).*
- **D-04a:** **Staleness banner** (SYNC-02): while a sync is pending/debouncing, agent-facing output shows a staleness warning — **prepended to `explore` markdown output** and reflected as a `stale`/`pendingSync` signal in `status`. The pending state is set by the watcher/daemon on first event and cleared on sync commit; a stat/`last_sync_unix_ms` comparison is the fallback signal when no daemon is running. The same banner threads through the **MCP** `explore` tool output (one formatter, per Phase-3 D-08b).
  *[auto] Q: banner in explore+status vs status-only → Selected: explore (primary agent surface) + status signal (recommended — explore is the flagship agent round-trip; SYNC-02 says "agent-facing output").*

### Daemon, Lock & unlock (SYNC-04, SYNC-05)
- **D-05:** **`codegraph daemon`** = a long-lived local process owning the watcher + the single `GraphStore.Writer`, so multiple agent sessions share one indexer. Guard it with a **lockfile in `.codegraph/`** carrying pid + start timestamp (single-writer invariant, INDX-05). Where a separate daemon is unsupported/undesired, provide an **in-process fallback**: the watcher runs inside `serve` (or a CLI invocation) at the same single-writer discipline. **`codegraph unlock`** removes a **stale** lock after a crash — it checks pid liveness (and staleness of the timestamp) before clearing, so it won't stomp a live daemon.
  *[auto] Q: dedicated daemon process + lockfile with in-process fallback vs watcher only inside serve → Selected: daemon + lockfile + in-process fallback (recommended — SYNC-04 requires a shared server with documented fallback; SYNC-05 requires unlock).*

### MCP Reconnect Reconciliation (SYNC-03)
- **D-06:** On **MCP server (re)connect / init**, run the **D-01a stat+content-hash reconciliation** (a `Sync` pass) **before serving tools**, so offline changes made while the watcher/daemon was down are caught and the first `explore` reads a current graph. Cheap no-op when nothing changed. Reuses the exact `sync` reconcile routine — no second reconciliation code path.
  *[auto] Q: reconcile on MCP reconnect vs rely solely on the live watcher → Selected: reconcile on reconnect (recommended — catches changes made while offline; SYNC-03 is explicit).*

### Leak-Free Lifecycle & Soak (SYNC-06)
- **D-07:** **Lifecycle via `context` cancellation + `sync.WaitGroup` join on shutdown** — every goroutine (watcher loop, debounce timer, sync worker) is owned by a cancelable context and joined before the daemon/watcher returns. A **soak test** exercises many watch→debounce→sync cycles and asserts **goroutine-leak-free** (via `go.uber.org/goleak` or a `runtime.NumGoroutine()` delta gate; tool choice executor's discretion). This soak gate is the acceptance evidence for SYNC-06.
  *[auto] Q: goleak-based soak vs manual NumGoroutine delta → Selected: context+WaitGroup lifecycle with a goleak/NumGoroutine soak gate (recommended — standard Go leak-detection; exact library is discretion).*

### Claude's Discretion
- Exact secondary-index key shape for D-02 (`fx/<path>/<id>` reverse map vs owning-file-prefixed node/edge keys) and its backfill-detection mechanism (Meta flag vs namespace probe).
- Sync writer batch granularity (per-file vs per-N) so long as no per-symbol commits and Phase-2 determinism holds.
- Debounce data structure / timer reset strategy; whether to leverage the CGo parser's **incremental-reparse** path (`internal/parser/cgo`, present since Phase 1/2) or full-reparse changed files.
- Lockfile format/contents beyond pid+timestamp; `daemon` foreground vs background and its IPC (if any) with `serve`.
- Exact staleness banner text/placement and the `status` field name; `--json` shape of the `sync` summary.
- Soak-test iteration counts and the leak-detection library.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents (researcher, planner, executor) MUST read these before planning or implementing.**

### Storage substrate — the pruning & write seams (bind directly; do not re-derive)
- `internal/graphstore/store.go` — `GraphStore` (`Snapshot`/`NewWriter`), `Writer` (`PutNode`/`PutEdge`/`PutFile`/`PutMeta`/**`DeleteFileSubgraph`**/`Commit`/`Close`), `Reader` (`IterateEdges`/`IterateNodes`/`IterateFiles`). The single-writer + snapshot-reader concurrency contract (INDX-05) the daemon must respect.
- `internal/graphstore/keys.go` — the `n/`·`e/`·`f/`·`m/` keyspace; **`fileSubgraphPrefix`/`rangeUpperBound` + the `keys.go:104-118` note that `DeleteFileSubgraph` currently prunes only the `f/` record and that extending it to node/edge records is the Phase-4 hook (D-02)**; the reserved additive-namespace mechanism (`keys.go:21`); the forward-only edge ordering behind D-02a's reverse scan.
- `internal/graphstore/pebble_store.go` — the `DeleteFileSubgraph`/`DeleteRange` implementation D-02 extends to prune the file-owned subgraph.
- `internal/schema/graph.proto` + `graph.pb.go` — **`File.content_hash` (field 2, SHA-256)** backing D-01a content-hash diffing; **`Meta.last_sync_unix_ms` (field 4)** backing D-01b staleness stamping + D-04a/D-06.
- `internal/schema/meta.go` — `SchemaVersion = 1` and the additive-only-no-bump rule D-02 relies on (a new namespace is additive).

### Indexer substrate — reused by the incremental path
- `internal/indexer/pipeline.go` — `Run(repoRoot, storeDir, opts)`, `Options`, `Stats`; the from-scratch orchestrator whose extract→resolve stages D-01's `Sync()` reuses.
- `internal/indexer/extract.go` + `resolve.go` + `symbolindex.go` + `discover.go` — per-file parallel extract, sequential resolve (single writer), the global symbol index, and the file-discovery/exclusion seam the watcher must agree with (D-04).
- `internal/indexer/nodeid/` — content-hashed `<kind>:<sha256-trunc>` node ids (Phase-2 D-02/D-02a); the stability property behind D-03 rename/move handling.
- `internal/parser/cgo/parser_cgo.go` — the CGo parser + its incremental-reparse path (present since Phase 1/2); one-parser-per-worker rule; `parser.MaxSourceBytes` ceiling.

### Query/MCP surface — where dependent-detection and staleness thread through
- `internal/query/` — the Phase-3 read engine incl. the **D-04 query-time reverse-adjacency scan** D-02a reuses for dependent detection; the `explore`/`status` formatters D-04a's banner threads through.
- `internal/mcp/` + `internal/cli/serve.go` — the stdio MCP server where D-06 reconnect reconciliation and the D-05 in-process watcher fallback wire in (one formatter path, Phase-3 D-08b).
- `internal/cli/root.go` + `status.go` + `init.go`/`index.go`/`uninit.go` — the Cobra tree D-01/D-05 extend with `sync`/`daemon`/`unlock`; `status` is where the staleness signal surfaces; the thin-CLI-delegation pattern to mirror.

### Project planning & prior decisions
- `.planning/ROADMAP.md` §"Phase 4: Incremental Sync & File Watcher" — the 5 success criteria this CONTEXT must satisfy.
- `.planning/REQUIREMENTS.md` — **INDX-03, INDX-04, SYNC-01…SYNC-06** are the locked contract; INDX-05/RES-01/LANG-01/QRY-*/MCP-* (Phases 1–3, complete) are the substrate. Note "Out of Scope: Polling as default file watching."
- `.planning/phases/02-go-indexing-pipeline/02-CONTEXT.md` — D-02 (content-hashed node id — the sync/rename seam), **D-05 (edge collapse)**, D-06 (Go node/edge vocabulary) the sync must preserve; the deferred note that the `f/`-namespace range-delete hook is what Phase-4 pruning binds to.
- `.planning/phases/03-query-engine-mcp-server/03-CONTEXT.md` — **D-04 (query-time reverse adjacency)** D-02a reuses; D-08b (one engine, two front-ends) the staleness banner/reconcile must not fork.
- `.claude/CLAUDE.md` §"Supporting Libraries"/fsnotify (`NewWatcher`/`NewBufferedWatcher`, recursive-add-on-Create), §"What NOT to Use" (no polling default), §Storage (Pebble range-deletes + snapshots for consistent reads during background writes).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`Writer.DeleteFileSubgraph(path)` + `keys.fileSubgraphPrefix`/`rangeUpperBound`** — the range-delete prune mechanism was **built in Phase 1 specifically as the Phase-4 hook**; today it prunes only the `f/` record. D-02 extends it to node/edge records via a file-owned secondary index — the intended extension point, not new architecture.
- **`File.content_hash` (SHA-256) + `Meta.last_sync_unix_ms`** — the change-detection and staleness fields already exist in the schema; no schema bump needed (D-01a, D-01b).
- **Two-pass indexer internals (`internal/indexer`)** — `extract`/`resolve`/`symbolindex`/`discover` are directly reusable by `Sync()`; the incremental path is a new orchestration over existing stages, keeping one indexing code path (D-01).
- **Phase-3 query engine reverse scan (`internal/query`, D-04)** — reused for dependent-file detection (D-02a) with no persisted-index change.
- **CGo parser incremental-reparse path (`internal/parser/cgo`)** — present since Phase 1/2; available if the executor wants sub-file reparse (discretion).

### Established Patterns
- **Interface-boundary discipline (`internal/graphstore/archtest`)** — only `internal/graphstore` imports Pebble; the sync/watcher/daemon code MUST depend only on the `GraphStore`/`Reader`/`Writer` interface. D-02's secondary index lives **inside** `internal/graphstore`.
- **Single coordinated writer** — the resolve pass owns the only writer (INDX-05); the daemon (D-05) preserves this — one writer even with many agent sessions.
- **RED→GREEN atomic commits + determinism-as-a-gate** — write the failing pruning-fixture / soak test before the implementing code; a `sync` that reparses everything must land the same graph a from-scratch `index` would (the Phase-2 byte-identical property must survive incremental update).
- **Thin CLI → engine delegation** — `sync`/`daemon`/`unlock` resolve paths+flags and delegate; no logic in the Cobra command (mirrors `init`/`index`/`serve`).

### Integration Points
- **Watcher → debounce → `indexer.Sync()` → `GraphStore.Writer`** — the write path; the daemon owns it end-to-end under the single-writer lock (D-05).
- **`indexer.Sync()` ↔ `internal/query` reverse scan** — dependent detection reads a snapshot while sync writes; Pebble snapshots give the consistent point-in-time read (D-02a).
- **`serve`/MCP ↔ reconcile** — D-06 runs `Sync` on reconnect; D-04a surfaces staleness through the shared `explore`/`status` formatters.
- **`go.mod`** — adds `github.com/fsnotify/fsnotify` (pure-Go; the only new direct dependency this phase introduces).

</code_context>

<specifics>
## Specific Ideas

- **The pruning gap IS the phase's technical spine.** `keys.go:113-118` names it explicitly: nodes/edges aren't keyed by owning file yet, and extending `DeleteFileSubgraph` is *the Phase-4 hook*. Surface D-02 (secondary index) + D-02a (dependent recomputation, not a reverse-delete) to the planner first — nothing else in the phase is correct until pruning is correct.
- **Dangling edges are prevented by recomputation, not deletion.** The forward-only keyspace can't find edges *into* a deleted symbol; reusing Phase-3's reverse scan to recompute dependents is the correctness argument. Don't build a bespoke reverse-delete — that pulls Phase-8 work forward.
- **Determinism must survive incremental update.** An incremental `sync` that touches every file must yield the same graph as a from-scratch `index` (Phase-2's byte-identical gate). Make a "sync-equals-reindex" property test a first-class acceptance check, not an afterthought.
- **Existing indexes must not break.** Phase-2/3 graphs predate the D-02 secondary index; D-02b's one-time backfill keeps a `sync` on an old `.codegraph/` correct rather than silently mis-pruning.
- **Don't compensate for Phase-5 resolution gaps inside sync.** WR-01/WR-02 (same-package collision, selector-on-non-identifier) are Phase-5 fixes; sync should reproduce the current extractor's output faithfully, not paper over them.

</specifics>

<deferred>
## Deferred Ideas

- **Persisted reverse-edge index** — Phase 8 scale work; adopt only if the D-02a query-time reverse scan profiles as a bottleneck on the 100k-file monorepo.
- **Interface→impl dispatch synthesis, `provenance: heuristic` tagging, framework `route` nodes, non-Go languages** — Phase 5 (RES-02/03, LANG-02+). Also the Phase-5-deferred Go resolution gaps WR-01 (same-package func/method collision) and WR-02 (selector-on-non-identifier) — sync must not work around them.
- **100k-file monorepo scale, peak-RSS bounding, performance-regression gates** — Phase 8 (INDX-06, PERF-*). Phase 4 proves correctness + leak-freedom on a mid-size repo.
- **HTTP/SSE MCP transport, remote/multi-user daemon** — v2 (SERVER-01). The Phase-4 daemon is local single-user.
- **`install`/`uninstall`/`upgrade`/`help`/`version`/`telemetry`** — Phase 6 (AGNT-*, CLI-*). Phase 4 ships `sync`/`daemon`/`unlock`.
- **Migration from TS SQLite `.codegraph/`** — Phase 7 (MIGR-*).

None of the above are scope creep into Phase 4 — they are correctly-placed future work, recorded so nothing is lost.

</deferred>

---

*Phase: 4-Incremental Sync & File Watcher*
*Context gathered: 2026-07-11*
