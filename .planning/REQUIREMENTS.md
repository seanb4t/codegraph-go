# Requirements: CodeGraph Go — v0.12.0 Local Graph UI

**Defined:** 2026-08-22
**Core Value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.

**Milestone goal:** Give the graph a human face — a local, read-only web UI served by the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent.

**Consumes:** SEED-001 (local Svelte/shadcn graph-browsing UI, planted 2026-07-14).
**Note on SEED-001's trigger:** the seed fired on "CLI-surface parity with TS CodeGraph," a gate v0.11.0 retired when Compatibility stopped being a constraint. The warrant is now that the query surface is stable *on its own terms*.

## v1 Requirements

### Command & Transport

- [x] **SRV-01**: `codegraph ui` binds loopback, prints and opens the URL, and runs as its own process — never sharing a lifecycle with `serve --mcp`
- [x] **SRV-02**: Origin/Host exact-match validation rejects DNS-rebinding requests (CVE-2024-28224 Ollama, CVE-2025-66414/66416 MCP SDK class); loopback binding alone is insufficient and is not treated as sufficient
- [x] **SRV-03**: Read-only by construction; bind address and auth exist as explicit seams so a later `--host` is a change, not a rewrite — neither is exposed in v1
- [x] **SRV-04**: Every RPC opens-snapshots-closes the store and never caches an `Engine`/`GraphStore` handle; an `ErrStoreLocked` that outlives `graphstore.Open`'s bounded retry renders as "indexing in progress", never as an error
- [x] **SRV-05**: Verbatim source serving reuses the existing MCP path-confinement fix rather than reimplementing it

### Wire Protocol

- [x] **RPC-01**: Protobuf schema for the UI API, inheriting `internal/schema/graph.proto`'s additive-only evolution discipline (D-02a — field numbers never renumbered or reused; retired fields `reserved`)
- [x] **RPC-02**: `connect-go` handlers mount on `net/http` alongside the `go:embed`'d SPA
- [x] **RPC-03**: SPA fallback routing — client-side routes resolve to `index.html`; RPC paths and hashed assets do not
- [ ] **RPC-04**: A Connect server-streaming method carries re-index events to the browser over plain HTTP/1.1
- [x] **RPC-05**: Message sizes are bounded; verbatim source blobs are handled without unbounded response growth

### Engine Seam

- [x] **ENG-01**: A structured `NodeDetail` variant is exposed; `Node()` and every frozen golden covering it remain byte-identical
- [x] **ENG-02**: A structured `ExploreResult` variant is exposed; `Explore()` and every frozen golden covering it remain byte-identical
- [x] **ENG-03**: `Engine.FileGraph()` returns aggregated file/package adjacency with per-kind edge counts, computed fresh per call following `BuildReverseAdjacency`'s full-scan discipline
- [x] **ENG-04**: `schema.Meta` records the indexed commit SHA as an additive field, following the `HasFileIndex` precedent (absent ⇒ pre-upgrade graph, degrades gracefully)

### Browse & Inspect

- [x] **BRW-01**: User can search symbols and files as they type
- [x] **BRW-02**: User can open a node and see verbatim source, callers, callees, and blast radius
- [x] **BRW-03**: User can click any neighbor and continue navigating from there
- [x] **BRW-04**: User can jump from a symbol reference in rendered source to its definition
- [x] **BRW-05**: User is offered a disambiguation picker when a bare symbol name resolves to multiple definitions
- [x] **BRW-06**: Source is syntax-highlighted using a lightweight highlighter with only the indexed languages registered
- [x] **BRW-07**: User can copy a file path or symbol name in one action
- [x] **BRW-08**: User can enter a natural-language query and get `Explore`'s relevance-selected results, alongside exact-name search
- [x] **BRW-09**: User can open the current file/line on GitHub, permalinked to the **indexed** commit so the remote view matches what the UI showed

### Navigation

- [x] **NAV-01**: Every view, symbol, and query state is addressable by a shareable URL encoding view, target, depth and limit
- [x] **NAV-02**: Browser back and forward navigate view history correctly
- [x] **NAV-03**: User can drive search and result selection from the keyboard, including a focus shortcut and `Esc` to dismiss
- [x] **NAV-04**: No-index, stale-index, and symbol-not-found each render an explicit state rather than an empty pane

### Query Workbench

- [x] **WRK-01**: User can run `Impact` on a symbol and adjust traversal depth interactively
- [x] **WRK-02**: User can select multiple files and see what they affect
- [x] **WRK-03**: User can run `Callers`/`Callees` with an adjustable result limit
- [x] **WRK-04**: Workbench results render as structured, sortable tables

### Graph View

- [x] **GRF-01**: A spike measures real file/package rollup node and edge counts against indexed repos, with its pass condition locked **before** dispatch, and its result selects the renderer (Cytoscape.js vs Sigma.js + graphology)
- [ ] **GRF-02**: User can view a whole-repo file/package graph with aggregated edges, laid out hierarchically with directory-structural grouping — never a force-directed whole-graph view
- [ ] **GRF-03**: User can drill into a file to see its symbols
- [ ] **GRF-04**: Dependency cycles are visually distinguished
- [x] **GRF-05**: The rendering library sits behind a narrow component seam so it can be swapped without an architecture change

### Index Health

- [x] **HLT-01**: User can see index freshness, coverage, per-language file counts, and node/edge counts
- [x] **HLT-02**: Staleness renders as a trust verdict above the raw numbers, not as a figure buried among them
- [x] **HLT-03**: Worktree mismatch renders as a loud, first-class visual warning

### Live Push

- [ ] **LIV-01**: Watcher re-index events feed the streaming RPC
- [ ] **LIV-02**: Open views update in place when the index changes
- [ ] **LIV-03**: Streaming survives multiple tabs, applies backpressure to slow clients, and shuts down and reconnects cleanly
- [ ] **LIV-04**: Graph layout stays stable across live updates — nodes do not jump on re-render

### Build & Supply Chain

- [x] **BLD-01**: The frontend toolchain is `pnpm` — `pnpm-lock.yaml` committed, pnpm version pinned via Corepack's `packageManager`, `pnpm install --frozen-lockfile` in CI
- [x] **BLD-02**: The built SPA is committed to the repo and `go:embed`'d into the binary
- [x] **BLD-03**: A drift guard proves committed `dist/` matches its SPA source, carries a positive assertion that it inspected something, and is demonstrated RED before being trusted green
- [x] **BLD-04**: A drift guard proves committed protobuf codegen matches its `.proto` source, covering **both** the new UI schema **and** the pre-existing `internal/schema/graph.proto` (which has no such guard today)
- [x] **BLD-05**: pnpm build-script approvals are committed, and CI asserts no new "Ignored build scripts" warning appears — a blocked lifecycle script must not silently change `dist/`
- [x] **BLD-06**: A `pnpm audit` gate covers the JS dependency tree that `govulncheck` and Syft cannot see, with a sibling assertion proving non-vacuity independent of `pnpm audit`'s own exit code
- [x] **BLD-07**: No Node or pnpm invocation appears anywhere in the signed release path — `.goreleaser.yaml` and `release.yml` build steps stay pure Go

### Carried Fix

- [x] **FIX-01**: `internal/mcp/server.go`'s `pendingWriter` counter is not corrupted by server-initiated notifications (CR-01)

## v2 Requirements

Deferred. Tracked, not in this roadmap.

### Index Health

- **HLT-04**: Coverage denominator — report files discovered but NOT indexed, answering "why is my file missing". Needs new Engine surface; deliberately out of v1 scope
- **BRW-10**: "Where am I" breadcrumb showing the containing symbol while scrolling a long file

### Graph View

- **GRF-06**: Community-detection clustering — the schema reserves annotation space for it; directory-structural grouping is the correct v1 substitute
- **GRF-07**: Opt-in whole-symbol graph, scoped within a single already-drilled-into package

### Browse & Inspect

- **BRW-11**: Editor handoff (`vscode://file/...`-style) with a configurable URI scheme

## Out of Scope

Explicitly excluded. Each is a trap specifically because this UI is local, read-only, single-binary and cloud-free.

| Feature | Reason |
|---------|--------|
| In-browser code editing | Read-only by construction; editing needs write-conflict handling and file locking against the user's real editor, and competes with an IDE this project has no reason to displace |
| Multi-user auth / accounts | "Local-first, no cloud" is a standing project boundary; bind/auth are deliberately seams, not features. Building auth now is speculative work against a future that may not arrive |
| Server-persisted bookmarks / saved views | Requires a write path and user-identity concept on a strictly read-only tool, duplicating what deep-linkable URLs (NAV-01) already provide — a saved URL *is* a bookmark |
| Force-directed whole-graph landing view | The best-documented failure mode in the surveyed literature: dependency-cruiser's FAQ, the MSR "Trimming the Hairball" paper, and `aryx/codegraph` abandoning node-link entirely. SourceTrail shipped it in v1 and retreated from it in v2 after user studies |
| Collaborative presence / cursors | No multi-user concept exists or is wanted; loopback bind means one machine, and "who else is viewing this" has no answer by construction |
| Free-form graph query console | A new query planner and a new unbounded-traversal cost surface on an unauthenticated local server; the four structured Engine-backed forms are the scoped design |
| Churn / complexity / hotspot dashboard | That data is not in the schema — `Node`/`Edge`/`Meta` carry structural graph facts, not git history. Adding it is an indexer-extraction milestone, and drifts the product from graph browser to code-quality dashboard |
| Hosted platform features | Different product; permanently out of scope per PROJECT.md |

## Traceability

Populated during roadmap creation (2026-08-22). Phase assignments come from `ROADMAP.md` → Phase Details; every v1 requirement maps to exactly one phase.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SRV-01 | Phase 1 | Complete |
| SRV-02 | Phase 1 | Complete |
| SRV-03 | Phase 1 | Complete |
| SRV-04 | Phase 1 | Complete |
| SRV-05 | Phase 3 | Complete |
| RPC-01 | Phase 1 | Complete |
| RPC-02 | Phase 1 | Complete |
| RPC-03 | Phase 2 | Complete |
| RPC-04 | Phase 6 | Pending |
| RPC-05 | Phase 1 | Complete |
| ENG-01 | Phase 1 | Complete |
| ENG-02 | Phase 1 | Complete |
| ENG-03 | Phase 5 | Complete |
| ENG-04 | Phase 1 | Complete |
| BRW-01 | Phase 3 | Complete |
| BRW-02 | Phase 3 | Complete |
| BRW-03 | Phase 3 | Complete |
| BRW-04 | Phase 3 | Complete |
| BRW-05 | Phase 3 | Complete |
| BRW-06 | Phase 3 | Complete |
| BRW-07 | Phase 3 | Complete |
| BRW-08 | Phase 3 | Complete |
| BRW-09 | Phase 3 | Complete |
| NAV-01 | Phase 3 | Complete |
| NAV-02 | Phase 3 | Complete |
| NAV-03 | Phase 3 | Complete |
| NAV-04 | Phase 3 | Complete |
| WRK-01 | Phase 4 | Complete |
| WRK-02 | Phase 4 | Complete |
| WRK-03 | Phase 4 | Complete |
| WRK-04 | Phase 4 | Complete |
| GRF-01 | Phase 5 | Complete |
| GRF-02 | Phase 5 | Pending |
| GRF-03 | Phase 5 | Pending |
| GRF-04 | Phase 5 | Pending |
| GRF-05 | Phase 5 | Complete |
| HLT-01 | Phase 4 | Complete |
| HLT-02 | Phase 4 | Complete |
| HLT-03 | Phase 4 | Complete |
| LIV-01 | Phase 6 | Pending |
| LIV-02 | Phase 6 | Pending |
| LIV-03 | Phase 6 | Pending |
| LIV-04 | Phase 6 | Pending |
| BLD-01 | Phase 2 | Complete |
| BLD-02 | Phase 2 | Complete |
| BLD-03 | Phase 2 | Complete |
| BLD-04 | Phase 1 | Complete |
| BLD-05 | Phase 2 | Complete |
| BLD-06 | Phase 2 | Complete |
| BLD-07 | Phase 2 | Complete |
| FIX-01 | Phase 1 | Complete |

**Coverage:**

- v1 requirements: 51 total
- Mapped to phases: 51
- Unmapped: 0 ✓

**By phase:**

| Phase | Name | Requirements |
|-------|------|--------------|
| 1 | Engine Seam, Wire Protocol & Secure Transport | 12 |
| 2 | SPA Toolchain, Embedded App Shell & JS Supply Chain | 7 |
| 3 | Browse, Inspect & Navigation | 14 |
| 4 | Query Workbench & Index Health | 7 |
| 5 | File/Package Graph View | 6 |
| 6 | Live Push | 5 |

---
*Requirements defined: 2026-08-22*
*Last updated: 2026-08-22 after roadmap creation (51/51 requirements mapped across 6 phases)*
