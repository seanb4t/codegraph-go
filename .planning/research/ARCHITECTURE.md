# Architecture Research: v0.12.0 Local Graph UI

**Domain:** Adding a ConnectRPC-served, `go:embed`'d Svelte SPA to an existing mature Go CLI/MCP product
**Researched:** 2026-08-22
**Confidence:** HIGH for integration points and the Pebble multi-process finding (verified against real code + upstream issue tracker); MEDIUM for exact protobuf message shapes and live-push fan-out sizing (design recommendations, not yet built)

## Standard Architecture

### System Overview

```
                    ┌─────────────────────────────────────────────┐
                    │              Browser (localhost)             │
                    │   Svelte + shadcn-svelte SPA (connect-es)    │
                    └───────────────┬───────────────┬─────────────┘
                                    │ unary RPC      │ server-stream
                                    │ (Connect proto)│ (Connect proto,
                                    │                │  HTTP/1.1 chunked,
                                    │                │  no h2c needed)
┌───────────────────────────────────▼───────────────▼─────────────┐
│  `codegraph ui` process (own PID, loopback-only net/http.Server) │
│  ┌──────────────┐  ┌─────────────────┐  ┌──────────────────────┐│
│  │ internal/ui  │  │ internal/ui      │  │ internal/ui           ││
│  │ /assets.go   │  │ /rpc.go          │  │ /stream.go            ││
│  │ go:embed     │  │ Connect service  │  │ fsnotify on           ││
│  │ dist/ SPA,   │  │ impls: open a    │  │ .codegraph/           ││
│  │ SPA fallback │  │ fresh query.     │  │ .sync-pending,        ││
│  │ to index.html│  │ OpenAt PER CALL  │  │ fan-out to N tabs     ││
│  └──────────────┘  └────────┬─────────┘  └──────────┬───────────┘│
│         mounted on one http.ServeMux, Origin/Host-validated       │
└──────────────────────────────┼────────────────────────┼──────────┘
                                │                        │ (signal only,
                                ▼                        │  no store read)
                    ┌───────────────────────┐            │
                    │  internal/query.Engine │◄───────────┘ (push triggers a
                    │  (UNCHANGED seam;      │               fresh OpenAt on
                    │   3rd consumer, not a  │               next client poll/
                    │   2nd implementation)  │               query, not inside
                    └───────────┬───────────┘               the stream handler)
                                │ graphstore.Open(dir) — PER CALL, closed after
                                ▼
                    ┌───────────────────────┐        ┌─────────────────────┐
                    │ internal/graphstore    │◄──────►│ `serve --mcp` /      │
                    │ (Pebble, D-04a's sole  │  same  │ `codegraph daemon`   │
                    │  door)                 │  dir,  │ (separate process,   │
                    │  EXCLUSIVE directory   │  never │  its own daemon      │
                    │  LOCK held only while  │  held  │  lockfile, writes    │
                    │  a GraphStore handle   │  open  │  .sync-pending       │
                    │  is open — see the     │  long- │  before/after each   │
                    │  Pebble finding below  │  term  │  debounced flush)    │
                    └───────────────────────┘        └─────────────────────┘
```

The important structural fact this diagram encodes: **`internal/ui` never talks to Pebble.** It is a pure transport/translation layer over `internal/query.Engine`, exactly like `internal/mcp` today — confirmed by the existing `internal/graphstore/archtest.TestNoPackageBypassesGraphStore`, which already scans the *whole module* for direct `pebble/v2` imports outside `internal/graphstore`. `internal/ui` inherits that enforcement for free the moment it exists, with zero new archtest code required, as long as it is written to the same discipline `internal/mcp` and `internal/cli` already follow.

### Component Responsibilities

| Component | Responsibility | New / Modified |
|-----------|-----------------|-----------------|
| `internal/cli/ui.go` (`newUICmd`) | Cobra command: resolve start path, require an index (UI has nothing useful to show without one — unlike `serve --mcp`'s MCP-03 zero-tools-without-index tolerance), build `internal/ui.Server`, bind loopback, print URL, optional `--open` browser launch | **New** |
| `internal/ui/server.go` | Constructs one `*http.Server`/`http.ServeMux` combining Connect handlers + static asset handler + Origin/Host validation middleware; loopback-only listener | **New** |
| `internal/ui/rpc.go` | Implements the generated Connect service interface(s); each RPC method calls `query.OpenAt` (or a UI-scoped equivalent) fresh, never holds a Reader across calls | **New** |
| `internal/ui/stream.go` | The server-streaming live-push method: fsnotify-watches `.codegraph/.sync-pending`, fans out "index changed" events to connected stream handlers with bounded/non-blocking send | **New** |
| `internal/ui/assets.go` | `//go:embed` the committed `dist/` SPA build; SPA fallback-to-`index.html` for client-side routes | **New** |
| `internal/ui/archtest/` | Confinement guard (extends D-04a's existing whole-module scan automatically) + drift guards (see Build-Time Architecture) | **New** |
| `proto/codegraph/ui/v1/*.proto` | The wire-API protobuf schema (distinct package from `internal/schema`'s storage-format proto) | **New** |
| `internal/query.Engine` (`node.go`, `explore.go`) | Gains two new structured-result methods (`NodeDetail`, `ExploreResult` — names illustrative) that reuse existing fetch helpers; **zero change** to `Node()`/`Explore()`'s markdown output | **Modified (additive only)** |
| `internal/query` (new file, e.g. `filegraph.go`) | New `Engine.FileGraph()`-shaped method computing the file/package rollup from a fresh full scan, mirroring `BuildReverseAdjacency`'s fresh-per-call discipline | **New method on an existing package** |
| `internal/mcp/server.go` | CR-01 fix: `pendingWriter`'s counter corruption from server-initiated notifications | **Modified (bug fix, unrelated surface but folded into this milestone)** |
| `internal/graphstore`, `internal/daemon`, `internal/watch` | **Untouched.** The UI's read path reuses `graphstore.Open`'s existing bounded-retry/`ErrStoreLocked` mechanism; its push signal reuses the existing `.sync-pending` sidecar `internal/daemon` already writes | **Unmodified** |

## Recommended Project Structure

```
internal/
├── ui/
│   ├── server.go          # http.Server assembly, loopback bind, Origin/Host guard
│   ├── rpc.go              # Connect service implementations (query translation)
│   ├── filegraph_rpc.go    # (or folded into rpc.go) rollup RPC handler
│   ├── stream.go           # live-push server-streaming handler + fan-out registry
│   ├── assets.go           # go:embed dist/*, SPA fallback handler
│   ├── dist/                # COMMITTED built SPA assets (git-tracked, not .gitignore'd)
│   └── archtest/
│       ├── graphstore_confinement_test.go   # thin, mostly free via D-04a's existing scope
│       ├── proto_drift_test.go              # .proto ↔ generated codegen, non-vacuous
│       └── dist_drift_test.go               # SPA source ↔ committed dist/, non-vacuous
├── query/
│   ├── node.go              # MODIFIED (additive): extract fetch from render
│   ├── explore.go           # MODIFIED (additive): extract fetch from render
│   └── filegraph.go         # NEW: file/package rollup, fresh-per-call scan
├── mcp/
│   └── server.go            # MODIFIED: CR-01 pendingWriter fix
proto/
└── codegraph/ui/v1/
    ├── graph.proto           # Node/Edge/File/Package wire messages, spans, pagination
    ├── query.proto            # unary RPCs: search, node detail, callers/callees/impact/affected, files, status, explore
    └── stream.proto           # server-streaming live-push RPC
web/                            # SPA SOURCE (not embedded directly — dist/ is)
├── src/
├── package.json
└── ...
```

### Structure Rationale

- **`internal/ui/` mirrors `internal/mcp/`'s shape** (`server.go` for construction, a handlers file, an assets/resources file) — this is a deliberate consistency choice, not a new pattern: the codebase already has exactly one precedent for "a second front-end over `internal/query.Engine`," and the UI should look structurally like it, not invent a new shape.
- **`dist/` lives under `internal/ui/`, not the repo root**, so the `//go:embed dist/*` directive in `assets.go` needs no `..` traversal — the same constraint `claudeassets.go`'s doc comment already documents for this exact reason (Go embed patterns cannot cross into a sibling directory via `..`; embedding only works at-or-below the directory containing the source file with the directive). Unlike `claudeassets.go` (which had to move to the repo root because its source lived at `.claude/`, a sibling of `internal/`), `internal/ui/dist/` can live directly under the package that embeds it, so no root-level indirection package is needed here.
- **`proto/codegraph/ui/v1/` is a separate protobuf package from `internal/schema`'s `codegraph.v1`** (see Protobuf Schema Design below) — the storage format and the wire API are different concerns with different evolution cadences and different consumers (Pebble records vs. a browser client), and this repo's own storage schema doc comment already frames `codegraph.v1` as specifically the "record format for the codegraph-go graph store," not a general-purpose API contract.
- **`web/` (SPA source) is a sibling of `internal/`, not nested inside it** — keeps the Go module's `go build ./...` and `go list ./...` machinery (which already special-cases `testdata/`, per `GOLDEN-01`'s documented gotcha) from ever needing to reason about a `node_modules/` tree, and matches this repo's existing convention of keeping non-Go build inputs (proto source, in a project with one) at predictable top-level locations.

## Engine Reuse Without Forking Query Logic (the markdown-vs-structured seam)

This is the single most consequential integration decision, and the codebase already answers most of it by precedent — **8 of the Engine's 10 read methods are already structured, not markdown**:

| Engine method | Return shape today | UI can consume directly? |
|---|---|---|
| `Query`, `Search` | `[]*schema.Node` / `[]Location` | Yes, as-is |
| `Callers`, `Callees` | `CallersResult` / `CalleesResult` (JSON-tagged) | Yes, as-is |
| `Impact`, `Affected` | `ImpactResult` / `AffectedResult` (JSON-tagged) | Yes, as-is |
| `Files` | `FilesResult` (JSON-tagged) | Yes, as-is |
| `Status` | `StatusResult` (JSON-tagged) | Yes, as-is |
| `Node` | `(string, error)` — markdown only, via `RenderNode`/`RenderNodeMultiDef` | **No — needs a new structured variant** |
| `Explore` | `(string, error)` — markdown only, via `RenderExplore` | **No — needs a new structured variant** |

These already-structured six exist precisely because `internal/cli`'s `--json` flags and `internal/mcp`'s SURF-06 JSON→markdown conversion (v1.0 Phase 2) both needed a non-string shape to work from — this is a well-worn seam, not a novel one. The UI's Connect handlers for `Callers`/`Callees`/`Impact`/`Affected`/`Search`/`Files`/`Status` are a **thin, mechanical field-mapping translation** from these existing `Marshal*JSON`-adjacent Go structs into the corresponding protobuf response messages. Zero risk to the frozen golden tests, because none of this touches `internal/query/render_markdown.go` or the `--json`/MCP paths at all.

### The real gap: `Node` and `Explore`

Both already compute structured intermediate data internally before rendering to markdown — they are not monolithic string-builders:

- `Node`'s single-def path (`renderSingleDefNode`) already separates fetch from render: `fetchCalls(node)` and `fetchCalledBy(node, rev)` return `[]*schema.Node`, then `RenderNode(node, calls, calledBy)` renders. The multi-def path (`renderMultiDefNode`) does the same per-candidate via a `fetch` closure returning `(source []byte, calls, calledBy []*schema.Node, error)`, then `RenderNodeMultiDef(symbol, matches, fetch)` renders.
- `Explore` already builds `groups []exploreFileGroup`, `blasts []exploreBlast`, and a `sources map[string][]byte` before calling `RenderExplore(query, fileCount, symbolCount, groups, blasts, sources, stale, skeletonFiles)`.

**The least-invasive fix is a pure extraction, not a rewrite:**

1. Add `Engine.NodeDetail(symbol, file string, line *int) (NodeDetailResult, error)` that runs the *exact same* resolution/fetch pipeline `Node()` already runs (`resolveNodeForDetail` / `enumerateSymbolDefs` + `narrowNodeMatches`, then `fetchCalls`/`fetchCalledBy`/`readSourceFile` per candidate) and returns the raw pieces as a new JSON/proto-friendly struct, **stopping before the `RenderNode`/`RenderNodeMultiDef` call**. `Node()` itself is untouched — it still calls the same fetch helpers and the same render functions in the same order, so its output is byte-identical and the frozen golden suite never sees a diff.
2. Add `Engine.ExploreResult(query string, maxFiles int) (ExploreResult, error)` the same way: run the identical ranking/gathering pipeline that produces `groups`/`blasts`/`sources`/`stale`/`skeletonFiles`, return them as an exported struct, stop before `RenderExplore`. `Explore()` is untouched.
3. Both new methods are **purely additive** — no existing exported signature changes, no existing test needs to change, and the CLI/MCP markdown consumers (protected by frozen goldens) never execute this new code path at all.

This is a small, mechanical refactor (the fetch/render split already exists inside both functions; it just isn't exposed as two separate exported steps yet) and should be scoped as its own early phase precisely because everything downstream (the UI's node-detail and graph-explore views) depends on it.

## Protobuf Schema Design

### Two separate `.proto` packages, deliberately

`internal/schema/graph.proto` (`package codegraph.v1`) is the **on-disk Pebble record format** — its evolution is governed by `D-02a` (additive-only, reserved field ranges for embedding vectors and community assignments) and is a private implementation detail of `internal/graphstore`. The **wire API to the browser** is a different contract with different consumers and a different, faster-moving cadence (a UI iteration can add a field to a response message without touching the storage format at all). Model it as a new package (e.g. `codegraph.ui.v1`) under `proto/codegraph/ui/v1/`, generated into `internal/ui/uiv1/` (or similar) — never reuse `schema.Node`/`schema.Edge` directly as wire types. Translate at the RPC-handler boundary in `internal/ui/rpc.go`, the same way `query.Location`/`query.CallersResult` already re-project `schema.Node` into smaller, JSON-shaped structs rather than serializing `schema.Node` verbatim.

### Message shapes

- **Node** (wire): `id`, `kind`, `name`, `qualified_name`, `file_path`, `language`, `start_line`, `end_line` — a thin subset of `schema.Node`'s fields, matching what `query.Location` already exposes plus whatever `NodeDetail` needs (signature, docstring, visibility). Do **not** wire-expose `schema.Node`'s reserved-for-future fields (embedding vectors, community assignment) until they have a real producer — reserve field numbers in the new proto the same way `graph.proto` already does, so a later addition is additive.
- **Edge** (wire): `source`, `target`, `kind` — same shape as the storage record, since callers/callees/impact/affected already expose this via `Location` pairs.
- **File** / **Package** (wire, for the rollup view): `path`, `language`, `symbol_count`, aggregated **edge counts by kind** between file/package pairs (not full edge lists) — this is a rollup, not a re-export of every edge; keep it small. Package is derived from `file_path`'s directory (or, for Go, its declared package — the extractor's existing `ModuleKey` concept may already carry this; verify against `internal/indexer/goextract` at implementation time rather than assuming).
- **SourceSpan**: `file_path`, `start_line`, `end_line`, `start_col`, `end_col` — a small reusable message referenced by Node and by any "jump to definition" response, rather than four loose scalar fields repeated in every message that needs a location.
- **Bounded traversals** (`impact`/`affected` depth, `callers`/`callees`/search limit): mirror the Engine's *already-enforced* server-side bounds — `validateDepth`/`clampDepth`/`clampAffectedDepth` and `validateLimit`/`MaxLimit` already exist in `internal/query`. The proto request messages should carry `depth`/`limit` as plain `int32` fields with **no client-trusted upper bound of their own** — the Connect handler must call the *existing* Engine methods, which already reject or clamp out-of-range values server-side. Do not re-implement bounds-checking in `internal/ui`; it would be a second, driftable copy of a rule the Engine already owns.
- **Pagination**: none of the Engine's current methods are cursor-paginated — they are limit-capped, full-result-in-one-call methods (`MaxLimit`). For v1 of the UI, mirror that: a bounded `limit` per request, no cursor/continuation token. This matches the CLI's own capability today and avoids inventing a pagination contract the CLI/MCP surfaces don't have and can't validate against.

### Verbatim source blobs — keep them out of the graph messages

`Node`'s multi-def render path already reads verbatim file source fresh from disk per candidate (`readSourceFile`, confined by `resolveSourcePath`'s repo-root confinement) — this can be large (a whole file). Putting a `bytes` or `string` verbatim-source field directly on the `Node` message that a *rollup* or *search* response returns would mean every symbol-list response silently carries megabytes of duplicate source text. Two established pitfalls this must avoid:

1. **Don't attach source to list-shaped responses at all.** `Search`/`Files`/rollup responses should carry only `SourceSpan` (path + line range) — the client fetches source lazily via a dedicated `GetSource(file_path, [start_line, end_line])`-shaped unary RPC only when a symbol is actually opened, mirroring how `Node`'s file-mode already does an on-demand `readSourceFile` rather than the graph store carrying source inline (the store itself never persists source text — `readSourceFile` reads fresh from disk on every call, D-05a).
2. **Cap what a single `NodeDetail`/source-fetch response can return.** protobuf itself has no built-in message-size ceiling beyond gRPC's (and Connect's) default max-receive-message-size — a pathologically large generated/vendored file opened through this path could produce a multi-MB response. `connect-go`'s handler options include a `WithReadMaxBytes`/`WithSendMaxBytes`-shaped configuration surface (verify exact option name against the pinned `connect-go` version at implementation time); set an explicit ceiling rather than relying on the default, and have `NodeDetail`/`GetSource` truncate (with a `truncated: bool` flag in the response) rather than fail outright on an oversized file, matching this codebase's existing "degrade rather than abort" convention (`WR-04`'s dangling-edge-skip pattern, applied here to oversized-source rather than missing-node).

## Live Push Architecture

### The event source: a signal that already exists

`internal/daemon`'s `flush` already touches `.codegraph/.sync-pending` before every debounced sync and removes it after a successful commit (`staleSidecarName`, referenced identically in `internal/query/status.go`'s `computeStale` — "true when `.codegraph/.sync-pending` exists (watcher/daemon signal)"). This file is written by **whichever process is actually running the watcher** — `serve --mcp`'s own default-on watcher, a standalone `codegraph daemon`, or a plain `codegraph sync` invocation — regardless of which process that is.

**`codegraph ui` should not run its own daemon/watcher at all.** It is read-only by construction (a stated milestone constraint); spawning a second competing writer process would be scope creep and would collide with `internal/daemon`'s existing single-writer lockfile the moment both `serve --mcp` and a hypothetical `codegraph ui`-embedded daemon tried to hold it. Instead:

- `internal/ui/stream.go` opens **one lightweight `fsnotify.Watcher`** scoped to the single file `.codegraph/.sync-pending` (or, more robustly, the containing `.codegraph/` directory filtered to that one filename — `fsnotify` doesn't watch non-existent files, so watching the directory and filtering `event.Name` is the correct pattern for a sidecar that is created/removed rather than always-present).
- On sidecar **removed** (sync completed successfully), broadcast an "index changed" event to every connected stream.
- On sidecar **created** (sync started), optionally broadcast a lighter "syncing" status event — nice-to-have, not required for the milestone's stated scope.
- This is completely decoupled from *who* is syncing, requires no new daemon/lockfile interaction, and costs nothing when no other process is actively syncing (the watch is idle).

### Event granularity: whole-index-changed, not per-file deltas

The watcher already debounces bursts into one flush per `internal/watch.DebounceDuration()` window (default tunable via `CODEGRAPH_DEBOUNCE_MS`) — by the time `.sync-pending` clears, an arbitrary number of files may have changed in that window. Computing and pushing a precise per-node/per-edge diff would require either (a) diffing two full graph snapshots (expensive, and the store has no changelog/CDC mechanism today) or (b) threading fine-grained change info out of `indexer.Sync` through the sidecar file itself (a real but much larger change to `internal/daemon`/`internal/indexer`, out of scope for this milestone). The stated requirement — "views update in place rather than going quietly stale" — is satisfied by the coarser signal: push a single `IndexChanged{ meta: <fresh Status()-shaped summary> }` event, and let the browser's already-open views re-issue their normal unary queries (which each do their own fresh `OpenAt`) to refresh. This is simpler, cheaper, and correctness-equivalent to a fine-grained diff for a "view updates" UX (as opposed to an "animate exactly what changed" UX, which is explicitly not what was scoped).

### Fan-out, backpressource, clean shutdown, reconnect

- **Fan-out**: maintain a small in-memory registry (`map[streamID]chan Event` guarded by a mutex, or a `sync.Map`) in `internal/ui/stream.go`. Each `StreamUpdates` RPC call registers a channel on entry and deregisters (via `defer`) on exit — standard Go server-streaming teardown, and `connect-go`'s `ServerStreamForHandler` already ties stream lifetime to the request context, so a client disconnect (browser tab closed) cancels `ctx` and the handler's `Send` loop should select on `ctx.Done()` to exit promptly.
- **Backpressure**: each per-client channel should be small and buffered (e.g. capacity 1) with a **non-blocking send** (`select { case ch <- event: default: /* drop; the next event supersedes it anyway */ }`) from the fan-out publisher. Because the event payload itself is coarse ("index changed," not a queue of deltas), dropping a stale "changed" notification in favor of a newer one is lossless in effect — the client's next unary query reads current state regardless of how many "changed" events it actually received. This sidesteps the harder general pub/sub backpressure problem entirely: **the event stream carries no state, so at-least-one-eventually-delivered is sufficient**, unlike a delta-stream design which would need to guarantee ordered, lossless delivery.
- **Reconnect/resume**: because events are stateless notifications (not deltas), there is no sequence-number/resume-token semantics to design. A browser reconnecting after a network blip simply re-issues its normal set of unary queries on connect (or the SPA proactively refetches on stream-open, treating "stream just (re)connected" the same as "an update arrived") — no server-side session state to reconstruct.
- **Clean shutdown**: `internal/ui/server.go`'s `http.Server` shutdown (triggered by the CLI command's signal handling, mirroring `serve --mcp`'s existing pattern of deferred watcher cancel + drain) should call `http.Server.Shutdown(ctx)`, which lets in-flight streams observe context cancellation and exit their `Send` loops; the fsnotify watcher on `.sync-pending` should be closed in the same shutdown path.

## Concurrency and Consistency — the Pebble multi-process question, VERIFIED

**Claim to verify:** does a second OS process opening the same Pebble directory a `serve --mcp`/daemon process already holds open cause a lock conflict, and does `Options.ReadOnly` (if it exists) offer an escape hatch?

**Verified finding (upstream, current as of the version pinned in `go.mod`, `github.com/cockroachdb/pebble/v2 v2.1.6`):** Pebble's `Open` acquires an **exclusive** directory lock unconditionally — a read-only open still takes the exclusive lock. This is a documented, still-open upstream limitation: [cockroachdb/pebble#1583](https://github.com/cockroachdb/pebble/issues/1583) states directly, quoting the maintainers' own framing of the gap, *"the pebble.Open method tries to acquire an exclusive lock regardless whether or not the DB is being opened in read-only mode"* — the issue asks for a proper 1-writer/N-reader multi-process mode and it does not exist; Pebble's own recommended path for concurrent multi-process access is to put a server (like CockroachDB itself) in front of it, which is exactly what this project's own `internal/graphstore.GraphStore` interface already is, in miniature. **There is no `Options.ReadOnly` escape hatch for the cross-process case** — read-only intent does not relax the lock.

**Why this is not a blocker, and requires zero new mechanism:** this codebase already discovered and solved exactly this problem, twice, before the UI was ever conceived:

1. `internal/graphstore/pebble_store.go`'s `Open()` already wraps every `pebble.Open` call in a bounded retry (`openLockRetryAttempts=5`, `openLockRetryBackoff=100ms`) and classifies a lock-held failure into the exported `ErrStoreLocked` sentinel — built for exactly the CLI-vs-daemon-vs-another-CLI-invocation collision this project already has today (documented in `Open`'s doc comment: "every open site in the module... collides on Pebble's exclusive directory LOCK by design").
2. `internal/graphstore` never holds a store open longer than one logical operation: `query.OpenAt` opens, takes one `Snapshot()`, and the caller `Close()`s when done — "one snapshot per invocation, never reused across calls" (Engine's own doc comment). `internal/indexer.Sync` does the identical open→write→close per debounced flush. **Nothing in this codebase holds a `GraphStore` handle open for a whole process's lifetime today** — which is precisely why the exclusive-lock limitation has never bitten it: collisions are always brief (a snapshot read or a batch commit), never permanent.
3. `internal/mcp`'s `openEngine` (referenced in `Engine.UseDetector`'s doc comment) already builds a **fresh `Engine`** — meaning a fresh `graphstore.Open` — **on every single MCP tool call**, inside a long-lived server process. This is the direct precedent: a long-lived server process that must serve many requests over its lifetime already pays the "reopen Pebble per request" cost today, by design, specifically to avoid holding the exclusive lock for the process's whole lifetime.

**The concrete implication for `internal/ui/rpc.go`: every RPC handler must open a fresh `query.OpenAt`-equivalent snapshot per call and close it before returning — exactly like `internal/mcp`'s `openEngine`, never once per server startup.** Get this wrong (cache one `*query.Engine`/`GraphStore` for the UI server's whole lifetime, to save the reopen cost) and the UI process will hold Pebble's exclusive lock continuously, and `serve --mcp`'s watcher flushes will exhaust their 5-attempt/400ms retry budget and start hard-failing syncs the entire time the UI is running — a severe, easy-to-miss regression that would only show up under real concurrent use, not in an isolated UI-only test. This should be called out explicitly in the phase plan and covered by an integration test that runs a real `codegraph daemon` (or `serve --mcp`) alongside a real `codegraph ui` against the same store and asserts sync flushes keep succeeding.

**Snapshot consistency for the UI specifically:** each unary RPC call gets one `Snapshot()` — a single, internally-consistent point-in-time view for that one request (Pebble snapshots are lock-free with respect to an in-flight writer, per `graphstore.Reader`'s own doc comment: "Multiple snapshots may be open concurrently with an in-flight writer; Pebble coordinates this without pinning memtables or blocking readers"). A UI request that fans out into several Engine calls internally (e.g. a rollup view that calls `Files()` then a new `FileGraph()`-shaped aggregation) should take **one** `OpenAt`/snapshot and reuse it across those calls within the single request — not reopen per sub-call — so the whole response reflects one consistent point in time, matching how `Node`'s multi-def render path already reuses one `BuildReverseAdjacency` scan across every candidate rather than rebuilding it per candidate.

## The File/Package Rollup Projection

**Computed on demand, not precomputed** — this matches the maintainer's own stated rationale in `PROJECT.md` ("the rollup is largely reading edges that already exist rather than computing a new projection") and the codebase's existing performance posture.

**Cost shape:** this codebase already pays full-graph-scan costs on every single `Callers`/`Impact`/`Affected`/`Status --json` (`edgesByKind`) call, by explicit design — `BuildReverseAdjacency`, `BuildImplementsIndex`, and `buildContainsIndex` are each a full `IterateEdges("")` scan, built **fresh on every call, with no package-level cache and no `sync.Once`** ("a long-lived process (the future MCP server) must never serve a stale point-in-time reverse view across multiple calls"). A file/package rollup is the same shape of work — one full `IterateNodes()` scan (group by `FilePath`, derive package from path/module-key) plus one full `IterateEdges("")` scan (aggregate `(source-file, target-file, kind)` triples into counts) — and should follow the identical fresh-per-call discipline as a new `Engine.FileGraph()` method in `internal/query`, not a new cached/precomputed projection living in the store.

At "a few thousand files" (the milestone's own stated scale, and in range of already-benchmarked corpora — `temporal/sdk-java` at 1223 files, `ccstatusline` at 13k files for TS/JS), this is the same O(nodes + edges) single-pass cost class this codebase's benchmarks already report favorably against TS CodeGraph (indexing throughput and query latency both measured, per `docs/BENCHMARKS.md`) — a *read* scan over an already-built graph is cheaper than the indexing pass itself. No new caching layer is warranted for v1 of this feature; if profiling after implementation shows the rollup view is measurably slower than the CLI's existing full-scan queries at comparable corpus size, the correct fix is the same one already used elsewhere in this codebase for *write*-side costs (batch writes into one `Writer`/`Commit`) — not a store-level precomputed projection, which would need its own invalidation-on-sync logic and reopen the "stale point-in-time view" risk `BuildReverseAdjacency`'s doc comment explicitly warns against.

**If a cache is added later** (e.g. because a specific UI interaction pattern proves the full scan too slow to re-run on every rollup-view open), it should be invalidated by the exact same `.sync-pending` signal `internal/ui/stream.go` already watches for live push — "cache goes stale exactly when the graph does" is a free correctness property of reusing that signal, and inventing a second staleness mechanism would be worth avoiding.

## Build-Time Architecture

### Two independent tool-generation pipelines, both following an existing repo pattern

This repository already has a working precedent for "pin an external code-generation tool without polluting the main module's dependency graph": `go.tool.mod` (task, goreleaser) and `go.tool-lint.mod` (actionlint) are isolated `-modfile`s specifically because co-locating unrelated tool dependencies in the root `go.mod` measurably bloats the main module's build list, `govulncheck` scope, and release SBOM. `protoc-gen-go`/`protoc-gen-connect-go` (and, for the TS side, `protoc-gen-es`/`protoc-gen-connect-es`, or `buf generate` driving both) should follow the **same isolation pattern** — a new tool-modfile (e.g. `go.tool-proto.mod`) rather than a root `go.mod` addition, keeping the release binary's dependency closure exactly as minimal as the "Minimal, audited dependencies" constraint requires. Note `internal/schema/graph.pb.go` was already generated once by `protoc v7.35.1` + `protoc-gen-go v1.36.11`, evidently outside any committed Taskfile target today — this milestone is a natural point to formalize that gap for both the existing storage proto and the new wire-API proto in one pass, rather than leaving proto codegen as an uncodified, manually-run step.

Proposed Taskfile additions (mirroring the `vuln`/`vuln:selftest` blocking/self-test pairing already in `Taskfile.yml`):

- `task generate:proto` — regenerates `internal/schema/graph.pb.go` and the new `internal/ui/uiv1/*.pb.go` + `*_connect.go` (and drives the TS generator for `web/src/gen/`) from the committed `.proto` sources, via the isolated tool-modfile.
- `task generate:ui` — runs the SPA's own build (`npm run build` or equivalent) producing `internal/ui/dist/`.
- `task check:proto-drift` and `task check:dist-drift` — the two drift guards this milestone explicitly requires (see below), run in CI on every push, never as a build-time regeneration step (regeneration is a local/CI-triggered developer action; the *check* is what gates merges).

### Drift guards that satisfy rule `84d1gfpywd` (positive assertion, not "nothing bad appeared")

This repository already has two directly-analogous, battle-tested precedents to copy the *shape* of, not just the intent:

1. **`resources_schema_drift_test.go`** (`GUARD-01/02`) — derives every claim from its real source (never hand-typed), and — critically — the phase that shipped it was explicit that "ungated resource content is worse than none," with a companion non-vacuity requirement: the guard must be demonstrated red against a real mutation before being trusted, and the review process for it found and fixed a real doc-drift bug the first time it ran for real.
2. **`TestGoldenScenarioCountIsExact` / `TestReFrozenGoldensValid`** (`FIXT-03/07`) — "sound only as a *pair*": one guard pins enumeration↔constant, the other pins enumeration↔filesystem; **neither alone suffices**, and both are individually zero-guarded (i.e., each asserts a nonzero/expected count before trusting the comparison, so an empty enumeration can't make the check pass vacuously).
3. **`import_graph_test.go`'s `foundGraphstoreImporter` sanity check** — after scanning for a forbidden import and finding none, the test does **not** stop there; it asserts that the scan *itself* found at least the one known-legitimate importer (`internal/graphstore` importing `pebble/v2`), so a scan that silently stopped resolving the target import path entirely (e.g. after a refactor) fails loudly instead of passing for the wrong reason.

Both of this milestone's required guards should be built to the same shape:

**`dist_drift_test.go`** (SPA source ↔ committed `dist/`):
- Compute a deterministic content hash over the SPA source tree (`web/src/**`, `web/package.json`, lockfile — explicitly excluding `node_modules/` and `dist/` itself).
- Compare against a hash recorded in a small sidecar the build writes alongside `dist/` (e.g. `internal/ui/dist/.source-sha256`).
- **Positive assertion, not just equality:** before trusting a match, assert the computed source-file count is above a floor (e.g. `> 0`, or a known minimum matching the SPA's expected file count) — a glob that silently matched zero files (e.g. after a directory rename) must not produce a vacuous "hash of nothing equals hash of nothing" pass.
- **Non-vacuity proof required before trusting it green:** mutate one SPA source file without regenerating `dist/`, run the guard, confirm it goes red naming the specific drift, then revert — the same "demonstrated red against a confirmed-applied mutation" standing rule this repo already applies to every other gate (per `PROJECT.md`'s Key Decisions table).

**`proto_drift_test.go`** (Protobuf codegen ↔ `.proto` definitions):
- Regenerate `*.pb.go`/`*_connect.go` (and, for full coverage, the TS output too, if CI has the toolchain for it) into a temp directory from the committed `.proto` files, using the pinned tool-modfile toolchain.
- Byte-diff (or a normalized diff tolerant only of a generator-version comment line, matching the existing `graph.pb.go` header's own `protoc-gen-go v1.36.11` / `protoc v7.35.1` version-stamp convention) against the committed generated files.
- **Positive assertion:** before diffing, assert `len(regeneratedFiles) == len(committedFiles) && len(regeneratedFiles) > 0` — a broken tool invocation that silently emits zero files (wrong working directory, missing plugin on `PATH`) must fail loudly as "generated nothing," never pass as "no diff found."
- **Non-vacuity proof:** hand-edit one committed generated file (a field name, a comment) without touching the `.proto`, confirm the guard goes red, revert.

## New vs. Modified Components — Explicit

**New:**
- `internal/ui/` package (server, RPC handlers, stream handler, asset embed, archtest)
- `internal/cli/ui.go` (`newUICmd`, registered in `root.go`'s `AddCommand`)
- `proto/codegraph/ui/v1/*.proto` + generated `internal/ui/uiv1/*.pb.go`/`*_connect.go`
- `web/` (SPA source, Svelte + shadcn-svelte) + `internal/ui/dist/` (committed build output)
- `internal/query/filegraph.go` (new `Engine.FileGraph()`-shaped method)
- A new isolated tool-modfile for protobuf/connect codegen tooling (mirroring `go.tool.mod`)
- Two Taskfile drift-guard targets + their CI wiring

**Modified (additive only — no existing exported behavior changes):**
- `internal/query/node.go` — new `NodeDetail`-shaped exported method, extracted from `Node()`'s existing fetch pipeline
- `internal/query/explore.go` — new `ExploreResult`-shaped exported method, extracted from `Explore()`'s existing gather pipeline
- `internal/cli/root.go` — one new line adding `newUICmd()` to `AddCommand(...)`
- `Taskfile.yml`, `.github/workflows/ci.yml` — new generate/check targets and jobs
- `internal/mcp/server.go` — CR-01 `pendingWriter` fix (unrelated bug, folded into this milestone per `PROJECT.md`'s stated rationale: "this milestone adds a second long-lived connection-bearing surface, so the existing one being correct matters more, not less")

**Explicitly unmodified:**
- `internal/graphstore` (no new methods, no `ReadOnly` mode — not needed, see the Pebble finding)
- `internal/daemon`, `internal/watch` (the UI reuses the existing `.sync-pending` signal rather than adding a hook)
- `internal/schema`'s `graph.proto`/`graph.pb.go` (the storage format; the UI gets its own separate proto package)
- Every existing markdown-rendering path (`render_markdown.go`, `RenderNode`, `RenderNodeMultiDef`, `RenderExplore`) and every frozen golden test that depends on it

## Suggested Build Order

1. **Engine seam extraction** (`NodeDetail`, `ExploreResult` in `internal/query`) — no dependency on anything else in this milestone; unblocks all UI read paths; lowest risk (pure extraction of already-existing logic) and highest leverage (everything downstream needs it). Should land and be independently tested (including a regression check that `Node()`/`Explore()`'s markdown output is unchanged) before any proto/UI work starts.
2. **Protobuf schema + codegen tooling** (`proto/codegraph/ui/v1/*.proto`, the isolated tool-modfile, `task generate:proto`, and the `proto_drift_test.go` guard) — can proceed in parallel with step 1 once the Engine's structured shapes from step 1 are known well enough to model the wire messages against them; the drift guard itself has no dependency on step 1 and can be built and proven non-vacuous immediately against the *existing* `internal/schema/graph.proto` before the new UI proto even exists, de-risking the guard mechanism itself early.
3. **`internal/ui` server skeleton** (`server.go`, `rpc.go` for the already-structured 7 methods, loopback bind, Origin/Host validation, `newUICmd`) — depends on 1 and 2. This is the first point at which `codegraph ui` does anything real (serves Connect RPCs for search/callers/callees/impact/affected/files/status) and is a natural first UAT checkpoint, even before the SPA or `NodeDetail`/`FileGraph`/streaming exist.
4. **`NodeDetail` + `ExploreResult` RPC wiring** — depends on step 1's structs and step 3's server skeleton.
5. **File/package rollup** (`Engine.FileGraph()` + its RPC handler) — depends on step 3; can proceed in parallel with step 4 since it's a new, independent Engine method with no shared state.
6. **Live push** (`stream.go`, fsnotify-on-sidecar, fan-out registry, streaming RPC + proto) — depends on step 3's server skeleton and step 2's proto tooling; should be built and integration-tested **specifically alongside a real running `serve --mcp`/`codegraph daemon`** (per the Pebble-lock finding above) rather than in isolation, since the property it must not violate (never holding the store open) is only observable under real concurrent multi-process use.
7. **SPA build + `dist/` embed + `dist_drift_test.go`** — depends on 2 (needs the generated `connect-es` TS client) and benefits from 3-6 being far enough along to build real UI screens against; the drift guard itself, like the proto one, can be built and proven non-vacuous as soon as *any* committed `dist/` exists, even a placeholder.
8. **CR-01 fix** (`internal/mcp/server.go` `pendingWriter`) — no dependency on any of the above; can be done at any point, but is lowest-risk to land *early* (it's an isolated, well-understood bug in an unrelated file) so it doesn't get rushed at the end of the milestone alongside integration-heavy UI work.

Step 8 aside, the critical path is **1 → 2 → 3 → {4, 5} → 6 → 7**, with 2 and 1 parallelizable, and 4/5 parallelizable once 3 lands.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Caching a long-lived `*query.Engine`/`GraphStore` in the UI server
**What people do:** hold one `Engine`/`GraphStore` handle open for the whole `codegraph ui` process lifetime, to avoid the per-request `pebble.Open` cost.
**Why it's wrong:** Pebble's exclusive directory lock is held for as long as the handle is open (verified: [cockroachdb/pebble#1583](https://github.com/cockroachdb/pebble/issues/1583)), so this would permanently starve any concurrently-running `serve --mcp`/`codegraph daemon` sync flush, exhausting `graphstore.Open`'s existing 5-attempt retry budget and turning transient collisions into permanent failures for as long as the UI runs.
**Instead:** open fresh per RPC call (per logical request, reusing one snapshot across sub-calls within that request), exactly like `internal/mcp`'s existing `openEngine` convention.

### Anti-Pattern 2: Re-implementing depth/limit validation in `internal/ui`
**What people do:** add client-request bounds-checking in the Connect handler before calling into the Engine, "for defense in depth."
**Why it's wrong:** `internal/query` already owns and enforces `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit`/`MaxLimit` server-side; a second copy in `internal/ui` is a driftable duplicate of a rule the Engine already guarantees for every caller (CLI, MCP, and now UI).
**Instead:** pass the client-supplied `depth`/`limit` straight through to the existing Engine methods and trust their existing validation; the UI's proto request messages need no bespoke bounds beyond what protobuf's own scalar types provide.

### Anti-Pattern 3: Threading verbatim source text through list/rollup responses
**What people do:** attach full file source to every `Node` message so the client "has everything in one round trip."
**Why it's wrong:** balloons response size for search/rollup views that never need source, and duplicates the exact anti-pattern this project's own storage design already rejected — the store itself never persists source text; it's read fresh from disk on demand (`D-05a`).
**Instead:** carry only `SourceSpan` in list-shaped responses; fetch verbatim source lazily via a dedicated call when a symbol is actually opened.

### Anti-Pattern 4: A precomputed/cached rollup projection with its own invalidation logic
**What people do:** pre-compute the file/package graph at sync time and store it as a new record kind, to make the rollup view "instant."
**Why it's wrong:** invents a second staleness/invalidation mechanism the project doesn't need yet, contradicts the maintainer's own stated rationale for choosing the rollup approach ("reading edges that already exist rather than computing a new projection"), and repeats the exact anti-pattern `BuildReverseAdjacency`'s doc comment already warns against for a long-lived process ("must never serve a stale point-in-time... view across multiple calls").
**Instead:** compute on demand, fresh per call, following the same discipline every other multi-edge-scan Engine method already uses; revisit only with a measured performance problem, and invalidate via the same `.sync-pending` signal already in use for live push if a cache is ever added.

## Sources

- `internal/query/engine.go`, `node.go`, `traverse.go` (this repo, read directly, HIGH confidence) — `OpenAt`/`Engine` seam, fresh-per-call discipline, `BuildReverseAdjacency`/`BuildImplementsIndex` conventions
- `internal/mcp/server.go`, `resources.go` (this repo, read directly, HIGH confidence) — `openEngine`-per-call precedent, `go:embed` precedent, `pendingWriter`/CR-01 bug location
- `internal/graphstore/store.go`, `pebble_store.go`, `open_lock_test.go` (this repo, read directly, HIGH confidence) — `GraphStore`/`Reader`/`Writer` interfaces, `Open`'s bounded-retry/`ErrStoreLocked` mechanism, snapshot semantics
- `internal/daemon/daemon.go`, `internal/query/status.go` (this repo, read directly, HIGH confidence) — `.sync-pending` sidecar mechanism (`staleSidecarName`), `flush`'s open/close-per-invocation pattern
- `internal/schema/graph.proto`, `graph.pb.go` (this repo, read directly, HIGH confidence) — existing protobuf conventions (additive-only, reserved ranges), confirms `protoc`/`protoc-gen-go` codegen is already in use but not yet Taskfile/CI-formalized
- `go.tool.mod`, `go.tool-lint.mod`, `Taskfile.yml` (this repo, read directly, HIGH confidence) — isolated tool-modfile pattern for external codegen/lint tools, `vuln`/`vuln:selftest` blocking/self-test pairing precedent
- `internal/graphstore/archtest/import_graph_test.go`, `internal/mcp/resources_schema_drift_test.go`, `internal/cli/present/archtest/charm_cgo_test.go` (this repo, read directly, HIGH confidence) — drift-guard and confinement-guard shapes, non-vacuity/self-test conventions satisfying rule `84d1gfpywd`
- `.planning/PROJECT.md` (this repo, read directly, HIGH confidence) — milestone scope, maintainer directives on ConnectRPC/Svelte/embed/loopback-only, rollup rationale
- [cockroachdb/pebble#1583](https://github.com/cockroachdb/pebble/issues/1583) (web, MEDIUM-HIGH confidence — direct GitHub issue text, cross-checked against this repo's own observed lock-retry behavior) — **VERIFIED**: `pebble.Open` acquires an exclusive lock regardless of `ReadOnly`; no multi-process 1-writer/N-reader mode exists
- `connectrpc/connect-go` (Context7, MEDIUM confidence, official docs) — `NewXHandler` returns `(path, http.Handler)` mountable on a plain `http.ServeMux`; `NewServerStreamHandler` shape
- General Go `http.ServeMux`/SPA-fallback pattern (web, LOW-MEDIUM confidence, several independent tutorial sources agreeing on the same shape) — longest-pattern-match naturally prefers a registered RPC path prefix over a catch-all SPA fallback with no extra ordering logic required
