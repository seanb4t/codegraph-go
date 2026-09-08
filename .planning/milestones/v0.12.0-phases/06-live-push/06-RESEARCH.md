# Phase 6: Live Push - Research

**Researched:** 2026-09-01
**Domain:** Connect RPC server-streaming over HTTP/1.1 (Go + browser fetch), fsnotify-based store watching, ELK/cytoscape-elk layout stability, Go fan-out/backpressure patterns
**Confidence:** HIGH on the wire-protocol mechanics and the FIX-01 analogue verdict; MEDIUM on the ELK pinning mechanism (upstream-maintainer-confirmed limitation, not directly testable in this session); MEDIUM on fsnotify/Pebble event-shape specifics (reasoned from Pebble's own commit-pipeline docs, not captured live in this session).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** The UI learns of a re-index by watching `.codegraph/store/` with fsnotify as a
  **wake-up**, then treats `GetMeta()` as the **authoritative** signal, emitting an event only
  when `last_sync_unix_ms` actually changed. Pebble's directory is noisy (compactions, WAL
  rotation, MANIFEST churn, OPTIONS rewrites); the fsnotify event is never itself the signal.
  No IPC, no daemon subscribe socket — `SRV-01` stays intact.
- **D-02:** The 60s absolute `writeTimeout` is cleared **per-request, streaming path only**, via
  `http.NewResponseController(w).SetWriteDeadline(time.Time{})` in a thin middleware. The 60s
  bound stays fully intact for all 13 unary rpcs and the SPA handler.
- **D-03:** LIV-04 is served by pinning existing node positions and laying out only new nodes,
  plus a no-layout fast path when the node set is unchanged — and the result **must be measured
  at guava scale (134 collapsed / 3,233 expanded), not fixture scale**.
- **D-04:** Backpressure is a bounded per-subscriber channel that **coalesces** — a newer event
  replaces an unsent older one, rather than being dropped or blocking the publisher.
- **D-05:** The event carries the `Meta` delta plus a monotonic generation counter: `generation`,
  `node_count`, `edge_count`, `healthy`, `health_message`, `last_sync_unix_ms`. Views displaying
  exactly those values render directly from the event; views needing full data re-fetch through
  their existing rpcs.

### Naming constraint

`WatchIndex`, `IndexEvents`, `StreamIndex` are BANNED (each contains a `mutatingVerbs` substring
match in `internal/uiserver/readonly_test.go`: `Index`). Full banned list: `Create, Update,
Delete, Remove, Set, Put, Post, Write, Add, Insert, Mutate, Patch, Modify, Sync, Reindex, Index,
Clear, Reset, Save`. Verified CLEAN this session (by the orchestrator, prior to this research):
`WatchGraph`, `GraphEvents`, `WatchRepo`, `StreamEvents`, `Subscribe`, `WatchStore`, `LiveEvents`,
`Events`, `GraphStream`. **Re-run `TestUIServiceMethodSetIsExactlyTheReadSet`'s sibling
`mutatingVerbs` check against the live fixture before the proto freeze regardless.**

### Claude's Discretion

- The rpc's final name (from the verified-clean set above) and its request message shape. The
  proto freeze earns a `blocking-human` checkpoint regardless.
- The store-watcher debounce window, sized so Pebble's compaction churn cannot produce an event
  burst. The `Meta` comparison in D-01 is the real filter; debounce is an efficiency measure.
- Client reconnect: exponential backoff with jitter, resuming from the last seen `generation`.
- Which routes subscribe, and how the graph route's subscription interacts with D-03's two
  layout paths.

### Deferred Ideas (OUT OF SCOPE)

- A daemon subscribe socket — revisit only if store-watching proves insufficient.
- Making `PendingChanges` live (Phase 4 D-06 placeholder; explicitly out of scope).
- Streaming per-view payloads to avoid re-fetches — revisit only if re-fetch latency is measured
  as a real problem.

### Carried-forward open items (from Phase 5, none blocking)

- **WINDOWS 26** — non-fatal `cytoscape-elk` adapter error (`pageErrorCount: 0` in every recorded
  run this phase's own sessions produced; not reproduced, not treated as gone).
- **WINDOWS 28** — guava-scale cytoscape layout density/legibility observation, pre-existing,
  non-blocking.
- **WINDOWS 29** — `web:drift`'s output half enumerates the filesystem (`Taskfile.yml:60`) while
  its source half uses `git ls-files` (`:42`), so it cannot detect an incompletely-staged bundle.
  **This phase rebuilds and re-commits the bundle** — stage with `git add web/build` and verify
  `git status --porcelain web/build` is empty; do not rely on the gate to catch a staging error.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RPC-04 | A Connect server-streaming method carries re-index events to the browser over plain HTTP/1.1 | Q1 below: `connect.NewServerStreamHandler` + `createConnectTransport`'s `stream()` are both confirmed, by reading the installed source, to work over plain HTTP/1.1 with no h2c/HTTP-2 requirement — flush-per-message is mandatory and already the library's default behavior. |
| LIV-01 | Watcher re-index events feed the streaming RPC | Q2 below: fsnotify on `.codegraph/store/` as wake-up, `(*query.Engine).IndexMeta()` as the authoritative read — same open/close-per-check discipline as every other rpc, never a held handle. |
| LIV-02 | Open views update in place when the index changes | Q1 delivery mechanics + the D-05 event-shape gap documented under Pitfalls — `StatusBanner`/`classifyStatus` cannot be driven by the event's five fields alone as currently shaped. |
| LIV-03 | Streaming survives multiple tabs, applies backpressure to slow clients, and shuts down/reconnects cleanly — verdict against `pendingWriter`'s root cause recorded | Q4 below: **no analogous counter exists in `internal/uiserver` today** (grep + read, zero hits) — verdict recorded as a genuine finding, not asserted. Q5 below: fan-out pattern, connect-web disconnect surfacing, Page Lifecycle "frozen" state. |
| LIV-04 | Graph layout stays stable across live updates — nodes do not jump on re-render | Q3 below: ELK's own maintainer states literal position fixing is **not supported** by the layered algorithm; only the interactive/semi-interactive strategies "somewhat preserve" topology. This materially affects what D-03 can actually deliver — flagged as a Pitfall, not resolved here. |
</phase_requirements>

## Summary

Every one of the five locked decisions is implementable with the currently-installed dependency
versions and no new packages. The two highest-risk unknowns both resolved cleanly by reading
installed source rather than trusting training data: (1) Connect's server-streaming transport
genuinely streams message-by-message on both ends — connect-go's handler calls
`http.Flusher.Flush()` after every `Send()`, and `createConnectTransport`'s browser client reads
the response body incrementally via `ReadableStreamDefaultReader` rather than buffering it — so
no h2c/HTTP-2 upgrade is needed for RPC-04's "plain HTTP/1.1" requirement; and (2) ELK's own
maintainer has stated, on the record, that the layered algorithm **cannot precisely fix node
positions** — only "somewhat preserve" them via `interactive`/`semiInteractive` strategy flags —
which means D-03's "pin surviving nodes" language is achievable as an *approximation*, not a
guarantee, and the guava-scale measurement D-03 already mandates is the only way to know if that
approximation is good enough.

The FIX-01 analogue check (criterion 3) resolves cleanly: `internal/uiserver` holds **no**
counter, in-flight tracker, or lifecycle bookkeeping of any kind today (confirmed by both a
targeted grep and a full read of every non-test file) — so there is nothing for a server-initiated
stream write to corrupt. This is a genuine, checkable "no analogue exists" finding, not a
reassurance.

The one real design gap this research surfaces and does **not** resolve — because resolving it is
plan-phase's job, not research's — is that D-05's event fields (`generation`, `node_count`,
`edge_count`, `healthy`, `health_message`, `last_sync_unix_ms`) do not match the field set either
existing "chrome" component actually consumes: `StatusBanner`/`classifyStatus` need `initialized`,
`stale`, `store_exists`, `indexing_in_progress` (none of which are on the event), and the full
Health view's staleness verdict needs `stale`/`index_health`/`worktree_mismatch` (also absent from
the event). Nothing in the shipped UI today renders `healthy`/`health_message` at all — a
targeted search found zero matches outside doc comments. This is flagged under Pitfalls with a
concrete recommendation.

**Primary recommendation:** Implement the streaming rpc as an ordinary `connect.NewServerStreamHandler`
registered through the same `NewUIServiceHandler` constructor codegen already produces (no new mux
wiring); drive it from a small in-process publisher that runs its own `.codegraph/store/`
fsnotify watcher + debounce + `IndexMeta()` poll-on-wakeup loop, opening and closing the store on
every check exactly like `withEngine` does today; fan out via one bounded, coalescing channel per
subscriber (D-04); and treat D-03's "pin nodes" language as "best-effort via ELK's interactive
strategies, verified by guava-scale measurement" rather than a literal guarantee — because ELK's
own maintainers say a literal guarantee is not on offer.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Detecting a re-index happened | Backend (Go, `internal/uiserver`) | — | fsnotify + `IndexMeta()` must run server-side; the browser has no filesystem access (D-01). |
| Delivering the event to open tabs | Backend (Go, Connect handler) → Browser (fetch stream reader) | — | Connect server-streaming is the transport; both ends already exist in the installed dependency graph. |
| Coalescing/backpressure per subscriber | Backend (Go, in-process publisher) | — | D-04's bounded-coalescing channel is server-side state; the browser only ever sees the latest un-dropped message. |
| Re-fetching full view data after an event | Browser (SvelteKit routes) | Backend (existing 13 unary rpcs) | D-05: views needing full data re-fetch through rpcs that already exist — no new data-shape duplication. |
| Health/staleness chrome rendering | Browser (`StatusBanner`, health view) | Backend (`GetStatus`/`GetHealth`, unchanged) | See Pitfalls: the event alone is not sufficient for the FULL verdict; a re-fetch trigger (bypassing `notifyNavigated`'s identity guard) is likely still required for `stale`/`index_health`/`worktree_mismatch`. |
| Graph layout stability across an update | Browser (`GraphCanvas.svelte`, `cytoscape-elk`) | Backend (unchanged — `FileGraph`/`FileSymbols` already exist) | D-03 is a pure client-side rendering concern; the wire payload for graph data does not change in this phase. |
| Reconnect/backoff policy | Browser (Connect client wrapper) | — | The server has no reconnect state to track; every new stream request is a fresh subscriber registration. |

## Standard Stack

### Core

| Library | Version (installed) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `connectrpc.com/connect` | v1.20.0 [VERIFIED: go.mod:42] | Go server-streaming handler (`connect.NewServerStreamHandler`, `*connect.ServerStreamForHandler[Res]`) | Already the project's RPC layer for all 13 existing rpcs; server-streaming is a first-class method kind in the same library, confirmed by reading `handler_stream.go`/`protocol_connect.go` in the installed module. |
| `@connectrpc/connect-web` | 2.1.2 [VERIFIED: web/package.json:45] | Browser Connect transport, `createConnectTransport(...).stream(...)` | Already the project's client transport; its `stream()` method is confirmed (by reading the installed `connect-transport.js`) to use `fetch` + incremental `ReadableStream` reads, not a buffered response. |
| `github.com/fsnotify/fsnotify` | v1.10.1 [VERIFIED: go.mod:12] | Watch `.codegraph/store/` as the wake-up primitive (D-01) | Already the project's mandated cross-platform watch primitive (`internal/watch` package uses it for source files); no second watching library is introduced. |
| `cytoscape-elk` | 2.3.0 [VERIFIED: web/pnpm-lock.yaml, resolved] | Cytoscape ↔ ELK layout adapter | Already the project's graph-layout adapter (Phase 5); D-03 works within its existing `nodeLayoutOptions`/`elk` config surface, no swap. |
| `elkjs` | **0.9.3** [VERIFIED: web/pnpm-lock.yaml:720-721 `elkjs@0.9.3:` and `web/node_modules/.pnpm/elkjs@0.9.3`] | Layout engine cytoscape-elk wraps | Transitive dependency of `cytoscape-elk`; note the resolved version is **0.9.3, not 0.12.0** — Phase 5's research measured a different version than what is actually installed. This affects nothing measured in Phase 5 (the API surface used is stable across that range) but the plan should not assume 0.12.0's changelog applies. |

No new package is introduced by this phase. Every library the five decisions need is already a
direct or transitive dependency.

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.uber.org/goleak` | v1.3.0 [VERIFIED: go.mod:32] | Goroutine-leak detection for the publisher/fan-out subsystem | The exact same discipline `internal/watch`'s `TestSoak` already uses (`main_test.go`'s `goleak.VerifyTestMain`) — the new stream publisher needs the same soak/leak test shape criterion 5 demands. |
| `@playwright/test` | 1.62.1 [pre-installed, per task prompt] | Real-browser, trusted-input multi-tab verification | Criterion 3 needs 3+ real tabs — jsdom has no networking/streaming fidelity for this; Phase 5's own `graph-collapse-affordance-check.mjs` is the precedent pattern for a real-browser Playwright check committed alongside its diagnostic JSON record. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Connect server-streaming over HTTP/1.1 | Server-Sent Events (SSE) or a raw WebSocket | Both explicitly banned by RPC-04's own text ("no separate SSE or WebSocket transport") — not evaluated further; recorded here only because Connect's own docs sometimes present SSE as an alternative for streaming and a planner might otherwise wonder why it wasn't considered. |
| fsnotify wake-up + `Meta` compare (D-01) | Polling `GetMeta()` on an interval | Already rejected in CONTEXT.md D-01 — weakens "as they happen" (criterion 2). Not re-litigated here. |
| ELK `interactive`/`semiInteractive` strategy flags | A hand-rolled "restore survivor positions after layout" post-process | The ELK maintainer's own guidance (`eclipse-elk/elk#355`) says post-hoc position restoration risks new nodes overlapping survivors — exactly what D-03 already rejected as "full re-layout then restoring survivor positions." The interactive flags are the only avenue ELK itself offers toward the requested behavior; they are an approximation, not a guarantee, and must be measured (D-03 already mandates this). |

**Installation:** None — no new packages for this phase.

**Version verification performed this session:**
```
$ rg -n "connectrpc.com/connect" go.mod
42:	connectrpc.com/connect v1.20.0 // indirect
```
```
$ rg -n "fsnotify" go.mod
12:	github.com/fsnotify/fsnotify v1.10.1
```
```
$ rg -n "^  elkjs@" web/pnpm-lock.yaml
720:  elkjs@0.9.3:
```

## Package Legitimacy Audit

Not applicable — this phase introduces zero new packages. All libraries used (`connectrpc.com/connect`,
`@connectrpc/connect-web`, `github.com/fsnotify/fsnotify`, `cytoscape-elk`, `elkjs`, `go.uber.org/goleak`,
`@playwright/test`) were already audited and installed in Phases 1, 2, and 5 of this milestone.

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

One pre-existing item worth flagging, not a legitimacy issue but a go.mod hygiene one:
`connectrpc.com/connect` is marked `// indirect` in `go.mod` [VERIFIED: go.mod:42], yet
`internal/uiserver/server.go` already imports it **directly** today (`connect.WithSendMaxBytes`,
`connect.WithReadMaxBytes` — confirmed by reading the file's import block this session). This
marking is stale, not newly introduced by this phase, and — per this repo's own documented fact —
**cannot be corrected by `go mod tidy`**, which is broken on this branch for an unrelated
`tree-sitter-swift` reason (task prompt's own "Repo facts"). The plan should not attempt a manual
edit of the `// indirect` comment (that is tool-owned structure); it should either leave it as-is
with a comment noting the known cause, or defer the fix to whenever the `tree-sitter-swift`
blocker clears.

## Architecture Patterns

### System Architecture Diagram

```
 Pebble writes (WAL append on every PutMeta commit,
 MANIFEST/OPTIONS/.sst churn on background compaction)
              |
              v
   .codegraph/store/  (flat directory, fsnotify Add — no recursion needed)
              |
              v (Create/Write/Rename events — noisy, not filtered by content)
    +-----------------------------+
    | store watcher (new, Go)     |  D-01 wake-up
    | debounce (reuse Debouncer)  |
    +-----------------------------+
              |
              v (on debounced flush)
    +-----------------------------+
    | query.OpenAt -> IndexMeta() |  open-snapshot-close, same
    | -> close                    |  discipline as withEngine
    +-----------------------------+
              |
              v (only if last_sync_unix_ms changed since last known value)
    +-----------------------------+
    | publisher: fan-out registry |  one goroutine, holds no store handle
    | map[subscriberID]chan Event |  D-04: bounded, coalescing per-subscriber
    +-----------------------------+
              |         |         |
              v         v         v
        subscriber  subscriber  subscriber   (one per open browser tab's
         channel     channel     channel      stream RPC call)
              |         |         |
              v         v         v
    +-----------------------------+
    | connect.ServerStreamForHandler[Event].Send()  |  flush after every Send
    +-----------------------------+  (connect-go default; confirmed source-read)
              |  chunked HTTP/1.1 response body, no h2c
              v
    +-----------------------------+
    | createConnectTransport().stream()  |  fetch() + incremental
    | -> AsyncIterable<Event>            |  ReadableStream reader,
    +-----------------------------+      |  confirmed NOT buffered
              |
      +-------+--------+
      v                v
  StatusBanner/    Route-specific re-fetch
  health chrome    (Browse/Workbench/Graph call
  (needs a field-  their existing unary rpcs on
  set decision —   event arrival — D-05's "no
  see Pitfalls)     wire-shape duplication")
                         |
                         v
                   Graph route specifically:
                   GraphCanvas.svelte re-layout
                   via D-03's two paths (no-op
                   fast path, or ELK interactive
                   re-layout of new nodes only)
```

### Recommended Project Structure

```
internal/uiserver/
├── server.go            # existing — mount the new streaming handler here,
│                         #   same mux.Handle(uiv1connect.NewUIServiceHandler(...))
├── watchtimeout.go       # NEW — D-02's thin middleware, ResponseController.SetWriteDeadline
├── livepublish.go        # NEW — D-01's watcher+debounce+IndexMeta loop, D-04's fan-out registry
├── livehandler.go        # NEW — the streaming rpc handler itself (Send loop, subscriber lifecycle)
└── livepublish_test.go   # NEW — soak/leak test in TestSoak's shape (goleak-guarded)

web/src/lib/
├── live/
│   ├── live-client.ts    # NEW — wraps createConnectTransport's stream() call,
│   │                     #   reconnect/backoff (Q5), generation tracking
│   └── live-store.ts     # NEW — Svelte store/context distributing events to subscribed routes
└── status.ts             # MODIFIED — resolve the D-05 field-set gap (see Pitfalls) —
                           #   either extended to accept event-driven updates, or left as the
                           #   round-trip path with the event as its trigger
```

### Pattern 1: Server-streaming handler with a flush-per-message guarantee

**What:** connect-go's `ServerStreamForHandler.Send()` calls `defer flushResponseWriter(...)`
after every marshal — confirmed by reading the installed module.
**When to use:** Every `Send()` call in the new handler; no manual `Flush()` call is needed
because connect-go already does it.
**Example:**
```go
// Source: /Users/sean/go/pkg/mod/connectrpc.com/connect@v1.20.0/protocol_connect.go:809-814
// (read this session — connect-go v1.20.0, the version this repo has installed)
func (hc *connectStreamingHandlerConn) Send(msg any) error {
	defer flushResponseWriter(hc.responseWriter)
	if err := hc.marshaler.Marshal(msg); err != nil {
		return err
	}
	return nil
}
```
connect-go also gates entry to any server-streaming handler on the response writer actually
implementing `http.Flusher` (`protocol.go:344-352`, invoked at `protocol_connect.go:175`) — Go's
standard `net/http` server always satisfies this for the writer it hands to a handler, so no
special server configuration (h2c or otherwise) is required.

### Pattern 2: Browser-side incremental stream consumption

**What:** `createConnectTransport(...).stream(...)` returns an `AsyncIterable` whose underlying
implementation reads the fetch response body chunk-by-chunk via `ReadableStreamDefaultReader`,
decoding one Connect envelope at a time and yielding as soon as a complete envelope is available —
never waiting for the full body.
**When to use:** The generated client method for the new streaming rpc; consume it with
`for await (const event of stream)`, never by collecting the whole iterable into an array first
(that would reintroduce "arrives in a burst at the end" at the *application* layer even though the
transport itself streams correctly — see Pitfalls).
**Example:**
```typescript
// Source: web/node_modules/.pnpm/@connectrpc+connect@2.1.2.../protocol/envelope.js (read this
// session — the pull() loop reader.read()s the underlying stream and enqueues each decoded
// envelope as soon as it is complete, never buffering the whole body)
async pull(controller) {
  let enqueuedOnce = false;
  while (!enqueuedOnce) {
    const result = await reader.read();
    if (result.done) { /* ...close/error... */ }
    else {
      for (const env of buffer.decode(result.value)) {
        controller.enqueue(env);
        enqueuedOnce = true;
      }
    }
  }
}
```

### Pattern 3: Open-snapshot-close on every Meta check (SRV-04's discipline, extended)

**What:** `withEngine` (`internal/uiserver/handlers.go`) opens via `query.OpenAt`, defers `Close`,
runs one operation, and never retains a handle. The new watcher's `IndexMeta()` check must follow
the identical shape — open, read `Meta`, close — on every debounced wake-up, never holding the
store open across the stream's lifetime.
**When to use:** Every fsnotify-triggered check in the new publisher.
**Example:**
```go
// Modeled on internal/uiserver/handlers.go:47-61 (withEngine) — read this session.
// The new watcher reuses the SAME openEngine seam, not a second one.
func checkForChange(repoPath string, lastKnownSyncMs int64) (*schema.Meta, bool, error) {
	eng, closer, err := openEngine(repoPath) // query.OpenAt, same seam withEngine uses
	if err != nil {
		return nil, false, err
	}
	defer closer.Close()
	meta, err := eng.IndexMeta() // internal/query/engine.go:158-167, VERIFIED this session
	if err != nil {
		return nil, false, err
	}
	if meta == nil || meta.LastSyncUnixMs == lastKnownSyncMs {
		return meta, false, nil
	}
	return meta, true, nil
}
```

### Anti-Patterns to Avoid

- **Buffering the whole stream client-side before rendering:** e.g. `Array.fromAsync(stream)` or
  awaiting the entire `for await` loop before updating any UI. The transport streams correctly;
  application code that accumulates before acting defeats criterion 2 anyway.
- **Wrapping the streaming `http.ResponseWriter` in a custom writer that doesn't forward
  `http.Flusher`/`http.ResponseController`'s optional interfaces:** `originHostGuard` today passes
  `w` straight through unwrapped (confirmed by reading `originguard.go`) — any new middleware
  (D-02's write-timeout clearer included) must do the same, or `checkServerStreamsCanFlush` will
  reject the handler at connect-go's own gate.
- **Treating a literal "pin the survivor node's (x,y)" as achievable through ELK's layered
  algorithm:** per the ELK maintainer directly (`eclipse-elk/elk#355`), it is not — see Pitfall
  below.
- **Holding a `*query.Engine`/`graphstore.Reader` across the stream's lifetime** to avoid repeated
  opens: this violates SRV-04's "never caches an Engine/GraphStore handle" and directly risks
  criterion 5 (starving a concurrent `daemon`/`serve --mcp` sync). Re-open on every debounced
  check; the cost is bounded by the debounce window, not the stream's lifetime.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Server-streaming transport | A custom SSE/chunked-response writer | `connect.NewServerStreamHandler` + `ServerStreamForHandler.Send()` | Already handles flush-per-message, envelope framing, and error/trailer encoding; a hand-rolled writer would have to reimplement the Connect wire protocol client-side parsing already expects. |
| Client-side stream parsing | A manual `fetch` + raw `ReadableStream` reader | `createConnectTransport(...).stream(...)` | Already does envelope decoding, compression rejection, and end-of-stream/trailer handling — confirmed correct and incremental by reading its source this session. |
| Recursive directory watching | A new `filepath.WalkDir` + `fsnotify.Add` loop mirroring `internal/watch/watcher.go` | A single non-recursive `fsw.Add(storeDir)` | `.codegraph/store/` is confirmed flat (`ls .codegraph/store/` shows 11 entries, zero subdirectories) — the recursive-walk machinery `internal/watch` needs for a source tree is unneeded complexity here. |
| Debounce coalescing | A new timer/burst-coalescing implementation | `internal/watch.Debouncer` (unexported internals, but the package is already in this module — `NewDebouncer(ctx, window, flush)`) | Already proven leak-free under `TestSoak`'s goleak-guarded soak; reusing it means the new publisher inherits that proof surface rather than needing a parallel one. Note it currently lives in `internal/watch`, a package whose own doc comment restricts it to depending only on `internal/indexer` — confirm during planning whether `Debouncer` can be imported as-is from `internal/uiserver` or needs to move to a shared location; this is a real open item, not resolved here. |
| Fan-out/pub-sub to multiple subscribers | A message-queue library or a channel-of-channels broadcast package | A plain `map[uint64]chan Event` guarded by a `sync.Mutex`, with per-subscriber bounded+coalescing send | Standard idiomatic Go for this scale (a handful of browser tabs, not thousands); no external dependency justifies itself at this scale, and D-04's coalescing semantics are a few lines of custom logic regardless of library choice. |

**Key insight:** Nothing in this phase's five decisions requires a new dependency. The two
libraries doing the actual work — `connectrpc.com/connect` and `@connectrpc/connect-web` — already
implement exactly the message-by-message streaming behavior RPC-04 and criterion 2 require; the
risk was never "does the library support this" but "does this project's *usage* of the library
(middleware wrapping, client consumption pattern) accidentally defeat what the library already
does correctly." Both `originHostGuard`'s pass-through style and the correct `for await` client
pattern avoid that.

## Common Pitfalls

### Pitfall 1: D-05's event field set does not match what any existing "chrome" component consumes

**What goes wrong:** The plan assumes `StatusBanner`/`classifyStatus` (or the full Health view) can
render "directly from the event with no round trip," per D-05's own text. Reading the actual code
shows this is not mechanically true as the components are shaped today.
**Why it happens:** `classifyStatus` (`web/src/lib/status.ts`) computes its five-member
`StatusVerdict` from `GetStatusResponse`'s `initialized`/`stale`/`store_exists`/
`indexing_in_progress` fields — **none of which are in the event's field set**
(`generation`/`node_count`/`edge_count`/`healthy`/`health_message`/`last_sync_unix_ms`). The full
Health view's staleness verdict additionally needs `stale`/`index_health`/`worktree_mismatch`
(`GetHealthResponse` fields 13-15), also absent from the event. A targeted search
(`rg -n "healthy\b" web/src -g '*.svelte' -g '*.ts'`, excluding generated code) found **zero**
components rendering `healthy`/`health_message` today — this UI element does not exist yet; D-05's
"health chrome" is new work, not a rewiring of something already shipped.
**How to avoid:** The plan needs to make one of two explicit calls, and should record which:
(a) accept that the event's *arrival* (not its exact field values) is what triggers a
`GetStatus`/`GetHealth` re-fetch for `StatusBanner`/the Health view — which is a "round trip," just
a cheap, already-open one, not a violation of D-05's *intent* (avoid duplicating rpc wire shapes)
even if it doesn't literally satisfy "no round trip" for every consumer; or (b) build a genuinely
new, minimal chrome element (e.g., a small pill showing live node/edge counts and last-sync
recency) whose field set is deliberately restricted to exactly what the event carries, leaving
`StatusBanner`'s fuller verdict on its existing `GetStatus`-driven path unchanged. Either is
defensible; silently assuming the existing `StatusBanner` "just works" from the event is not.
**Warning signs:** A task that wires the event straight into `IndexStatus`/`StatusVerdict` without
first reconciling the field mismatch above.

### Pitfall 2: ELK's layered algorithm cannot literally pin node positions

**What goes wrong:** D-03's "pin surviving nodes" language reads as a hard guarantee. It is not
one ELK's layered algorithm can give.
**Why it happens:** Directly from an ELK maintainer (`uruuru`), on the record, in response to
exactly this question: *"it cannot be used to precisely fix the positions of the nodes during
layered layout. You can use the various interactive strategies to somewhat preserve the topology
of the positions you already have."* [CITED: github.com/eclipse-elk/elk#355] A second maintainer
thread (`kieler/elkjs#100`, opened by the author of a diagramming library asking this exact
question) confirms: *"this is being requested/desired regularly... it's out of scope and not
well-suited for ELK layered."* [CITED: github.com/kieler/elkjs#100] `cytoscape-elk`'s own adapter
(`web/node_modules/cytoscape-elk/src/layout.js`, read this session) already sends every node's
CURRENT canvas position to ELK as its initial `x`/`y` on every layout call — but ELK's layered
algorithm ignores that hint unless `elk.interactive: true` plus
`elk.layered.layering.strategy: INTERACTIVE`, `elk.layered.cycleBreaking.strategy: INTERACTIVE`,
and `elk.layered.crossingMinimization.semiInteractive: true` are set — and even then the result is
an *approximation*, not an exact hold.
**How to avoid:** Set the four interactive/semi-interactive layout options above via
`cytoscape-elk`'s `elk` config object (its `defaults.js`, read this session, confirms `elk` is an
opaque pass-through object merged straight into ELK's own `layoutOptions`). Then **measure**
displacement at guava scale exactly as D-03 already mandates — the fixture-scale 0.00px result
from `05-07-SUMMARY.md` is not evidence this approach holds at real scale, and Pattern precedent in
this project (`GRF-01`) is to gate on a real measurement, not an assumption.
**Warning signs:** A task marked complete because a small fixture showed 0px displacement, with no
guava-scale re-measurement — this is the exact failure mode `05-07-SUMMARY.md` itself flagged as
"not necessarily representative."

### Pitfall 3: `web/node_modules`-resolved `elkjs` is 0.9.3, not 0.12.0

**What goes wrong:** Planning against elkjs 0.12.0's changelog or API surface (what Phase 5's
research apparently assumed) when the actually-resolved version is 0.9.3.
**Why it happens:** `pnpm-lock.yaml` resolves `elkjs@0.9.3` (a transitive dependency of
`cytoscape-elk@2.3.0`) — confirmed by `rg -n "^  elkjs@" web/pnpm-lock.yaml` this session and by
the presence of `web/node_modules/.pnpm/elkjs@0.9.3` on disk. No `elkjs` direct entry exists in
`web/package.json` to override this.
**How to avoid:** Verify any ELK option name or behavior against 0.9.3's actual shipped
`main.d.ts`/`elk-api.d.ts` (present at
`web/node_modules/.pnpm/elkjs@0.9.3/node_modules/elkjs/lib/`), not against a newer version's docs.
The three layout-option strings above (`elk.interactive`, `elk.layered.layering.strategy`,
`elk.layered.crossingMinimization.semiInteractive`) are stable, long-standing ELK core options
(referenced in ELK's own reference docs going back years), so they are not expected to be
version-sensitive — but this should be re-confirmed if the plan reaches for anything more exotic.

### Pitfall 4: The `internal/watch.Debouncer`'s package boundary

**What goes wrong:** Reusing `internal/watch.Debouncer` from `internal/uiserver` without checking
whether that import is architecturally sanctioned.
**Why it happens:** `internal/watch`'s own package doc comment (read this session) states:
*"internal/watch depends only on internal/indexer's exported ShouldSkipDir predicate — never on
internal/graphstore or pebble directly (D-04a archtest boundary; this package has no storage
concerns of its own)."* This describes what `internal/watch` may depend ON, not who may depend on
`internal/watch` — but an archtest boundary existing at all is a signal to check for a symmetric
rule before assuming the import is free. `Debouncer`'s exported API (`NewDebouncer`, `Add`, `Stop`,
`Wait`) has no store/indexer coupling itself, so a straight import is plausible, but this was not
verified against an archtest allow-list this session.
**How to avoid:** Check for an archtest (likely `internal/archtest` or similar, per this project's
established pattern of import-boundary tests) before assuming `internal/uiserver -> internal/watch`
is permitted; if it is not, either request an exception or duplicate the small, well-proven
`Debouncer` type rather than fighting the boundary.
**Warning signs:** A CI failure on an archtest package-dependency check after wiring this import.

### Pitfall 5: Pebble's own noise is a genuine efficiency concern, not just a theoretical one

**What goes wrong:** Sizing the debounce window too small, letting every MANIFEST/OPTIONS/`.sst`
churn event from a background compaction re-trigger the (cheap but non-zero) `IndexMeta()`
open/read/close cycle.
**Why it happens:** Pebble's own documentation confirms every committed batch (including every
`PutMeta` call) writes to the current WAL (`.log`) file, and *separately* that "compactions,
memtable flushes, and ingestions append version edits to the manifest" [CITED: cockroachdb/pebble
wiki Glossary, via Context7] — i.e., background compaction activity (unrelated to any actual
`PutMeta`/re-index event) legitimately touches `MANIFEST-*`/`.sst`/`marker.manifest.*` files on its
own schedule, independent of indexing. This matches the CONTEXT.md's own "Grounding measured this
session" table showing several of these files with current timestamps in the live store.
**How to avoid:** D-01 already establishes `Meta` comparison as the correctness filter, so a
too-small debounce window is an efficiency problem, not a correctness one — but every unnecessary
wake-up still means an unnecessary `query.OpenAt`/`IndexMeta`/`Close` cycle. `internal/watch`'s own
default debounce (`defaultDebounceMs = 2000`, `internal/watch/debounce.go:14` — read this session)
is a reasonable starting point for this second, independent debounce instance, but this is
Claude's Discretion per CONTEXT.md — no single "correct" value is asserted here.
**Warning signs:** A CPU/goroutine-churn regression under a soak test with an idle repo that still
undergoes periodic Pebble compaction.

## Code Examples

### Registering the streaming rpc (Go, once `.proto` is regenerated)

```go
// Source: pattern confirmed against internal/uiproto/uiv1/uiv1connect/ui.connect.go's existing
// NewUIServiceHandler (read this session) — adding a `rpc WatchGraph(...) returns (stream ...)`
// to ui.proto regenerates this constructor with an additional block shaped like:
uIServiceWatchGraphHandler := connect.NewServerStreamHandler(
	UIServiceWatchGraphProcedure,
	svc.WatchGraph,
	connect.WithSchema(uIServiceMethods.ByName("WatchGraph")),
	connect.WithHandlerOptions(opts...),
)
// ...folded into the same mux pattern the existing return value already builds. No separate
// mux.Handle call is needed in server.go beyond what already exists.
```

### Browser-side reconnect-with-backoff (idiomatic shape, connect-es's own test suite)

```typescript
// Source: web/node_modules/.../@connectrpc/connect/dist/esm/../connect-node/src/connect-transport.spec.ts
// (read via Context7 this session — this is connect-es's OWN documented retry idiom, not an
// invented pattern)
for (let backoff = 1; ; backoff++) {
  try {
    for await (const event of stream()) {
      // handle event; reset backoff to 1 on any successful message
    }
    break; // stream ended cleanly
  } catch (reason) {
    if (ConnectError.from(reason).code !== Code.Unavailable) throw reason;
    await new Promise<void>((resolve) => setTimeout(resolve, backoff * 1000 + Math.random() * 500));
  }
}
```
Jitter (`Math.random() * 500`) is added here per CONTEXT.md's discretion note ("exponential backoff
with jitter") — not present in the connect-es example verbatim, which uses linear backoff; this
project's own decision calls for exponential+jitter specifically.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gRPC-Web / WebSocket for browser streaming | Connect protocol's own server-streaming over `fetch` + chunked HTTP/1.1 | Connect protocol's original design goal (not a recent change) | This is *why* RPC-04 can require "plain HTTP/1.1... no separate SSE or WebSocket transport" — Connect was built precisely to make gRPC-shaped streaming work over ordinary `fetch`, which this project's already-installed dependency versions (connect-go 1.20.0, connect-web 2.1.2) both implement correctly, confirmed by source read. |

**Deprecated/outdated:** Nothing in this phase's dependency set is deprecated. `elkjs`'s own
README explicitly lists "how to consider previous layout results, including dynamic layout and
incrementally adding nodes" as a **recurring, still-unsolved FAQ topic** (issues #100, referencing
eclipse/elk#315, #355, #627) rather than a solved-and-since-changed problem — this is a standing
limitation, not a stale one.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `internal/watch.Debouncer` is importable from `internal/uiserver` without violating an archtest boundary | Don't Hand-Roll, Pitfall 4 | Low-medium: if blocked, the fallback (duplicate the ~140-line `Debouncer` type) is cheap and well-understood, just not DRY. |
| A2 | `defaultDebounceMs = 2000` is a reasonable starting point for the store-watcher's own debounce window | Pitfall 5 | Low: this is explicitly Claude's Discretion per CONTEXT.md; any reasonable value in the low seconds is defensible, and D-01's Meta-comparison correctness filter means a wrong value only costs efficiency, never correctness. |
| A3 | The reconnect backoff base (1s, exponential) and jitter magnitude in the Code Examples section are reasonable defaults | Code Examples | Low: CONTEXT.md only mandates the *shape* (exponential + jitter, resume from last-seen generation), not specific constants — this is illustrative, not prescriptive. |

## Open Questions

1. **How should `StatusBanner`/the Health view actually consume the live event?**
   - What we know: the event's five fields do not map onto either component's current input
     shape (Pitfall 1).
   - What's unclear: whether the plan should extend the event schema, add a bypass-identity
     re-fetch trigger, or build a new minimal chrome element.
   - Recommendation: surface this explicitly as a plan-phase decision point (likely its own
     `checkpoint:decision`), not something research should resolve on the maintainer's behalf —
     CONTEXT.md's own D-05 text leaves room for either interpretation.

2. **Does an archtest forbid `internal/uiserver -> internal/watch`?**
   - What we know: `internal/watch`'s own doc comment names an archtest boundary for *its own*
     outbound dependencies; nothing was found this session describing inbound restrictions.
   - What's unclear: whether such a test exists and what it currently allows.
   - Recommendation: check `internal/archtest` (or equivalent) at plan time before committing to
     reusing `Debouncer` directly.

3. **What is the actual measured guava-scale displacement under the ELK interactive strategies?**
   - What we know: the fixture-scale (8-node) measurement showed 0.00px displacement; the ELK
     maintainer's own statement is that this is only an approximation at any scale.
   - What's unclear: whether guava scale (134 collapsed / 3,233 expanded nodes) shows materially
     more displacement — this is precisely what D-03 mandates measuring, and this research
     session did not (and could not, without executing the actual implementation) measure it.
   - Recommendation: this is the phase's own required measurement (D-03), not a research gap to
     close beforehand — flagged here only so the plan does not skip it on the strength of the
     fixture-scale number.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `connectrpc.com/connect` (Go) | RPC-04 server handler | ✓ | v1.20.0 | — |
| `@connectrpc/connect-web` | RPC-04 browser client | ✓ | 2.1.2 | — |
| `github.com/fsnotify/fsnotify` | LIV-01 | ✓ | v1.10.1 | — |
| `cytoscape-elk` / `elkjs` | LIV-04 | ✓ | 2.3.0 / 0.9.3 | — |
| `go.uber.org/goleak` | Criterion 5's leak-free soak test | ✓ | v1.3.0 | — |
| `@playwright/test` + chromium | Criterion 3's real multi-tab verification | ✓ (per task prompt; chromium install not re-verified this session) | 1.62.1 | If chromium is not actually installed locally, `pnpm exec playwright install chromium` per the task prompt's own repo facts. |
| `codegraph daemon` / `serve --mcp` (this repo's own binary) | Criterion 5's real concurrent-process verification | ✓ (buildable from this repo) | — | — |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** chromium's actual local install state was not
independently re-verified this session (relies on the task prompt's own stated fact).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | stdlib `testing` + `go.uber.org/goleak` (soak/leak discipline, per `internal/watch/soak_test.go` and `internal/watch/main_test.go`'s `goleak.VerifyTestMain`) |
| Go config file | none — `go test` flags only |
| JS framework | vitest 4.1.11 + `@testing-library/svelte` (jsdom — no layout engine, per task prompt's own repo facts) |
| JS config file | `web/vitest` config implied by `web/package.json`'s `test`/`test:watch` scripts (not independently located this session; existing convention per `web/tests/*.test.ts`) |
| Real-browser framework | `@playwright/test` 1.62.1, following `web/scripts/graph-collapse-affordance-check.mjs`'s established pattern (trusted mouse input, never `dispatchEvent`) |
| Quick run command | `go test ./internal/uiserver/... ; task web:test` |
| Full suite command | `task test:unit` (excludes `internal/daemon` deliberately) + `task web:test` + a new Playwright multi-tab/reconnect script |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RPC-04 | Streaming rpc delivers messages incrementally over HTTP/1.1 | integration (Go, real `httptest.Server` + real client) | `go test ./internal/uiserver/... -run TestStream` | ❌ Wave 0 |
| LIV-01 | Watcher-driven event fires only on real `Meta` change | unit (Go) | `go test ./internal/uiserver/... -run TestWatcherPublish` | ❌ Wave 0 |
| LIV-02 | Open view updates in place on event | unit/component (vitest) | `pnpm --dir web test -- live-store` | ❌ Wave 0 |
| LIV-03 | Multi-tab, backpressure, reconnect, `pendingWriter`-analogue verdict | automated_ui (Playwright, real multi-tab) + Go soak test | `node web/scripts/live-push-multitab-check.mjs` (new) + `go test ./internal/uiserver/... -run TestSoak` | ❌ Wave 0 |
| LIV-04 | Layout stability at guava scale | e2e (Playwright, real corpus) — extends the existing `graph-expand-check.mjs` pattern | `node web/scripts/graph-live-update-check.mjs` (new) | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/uiserver/...` and the relevant `pnpm --dir web test -- <touched-file>` subset.
- **Per wave merge:** `task test:unit` + `task web:test` full suites.
- **Phase gate:** Full suite green, plus the criterion-5 real-process (`daemon`/`serve --mcp` concurrently running) verification and the guava-scale D-03 measurement, both before `/gsd-verify-work`.

### Wave 0 Gaps

- [ ] `internal/uiserver/livepublish_test.go` — covers LIV-01, and the goleak-guarded soak shape criterion 5 needs.
- [ ] `internal/uiserver/livehandler_test.go` — covers RPC-04's streaming-handler-level behavior (message-by-message `httptest` assertions, not just "it all arrived").
- [ ] `web/tests/live-client.test.ts` — covers the browser-side reconnect/backoff and generation-tracking logic at the unit level (vitest can mock `fetch`'s streaming body).
- [ ] `web/scripts/live-push-multitab-check.mjs` — new real-browser Playwright script, 3+ tabs, following the `graph-collapse-affordance-check.mjs` precedent (real input, committed diagnostic JSON record).
- [ ] `web/scripts/graph-live-update-check.mjs` — new real-browser Playwright script measuring guava-scale displacement across a live update, the D-03-mandated measurement.
- [ ] Framework install: none — vitest, Playwright, and goleak are all already present.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Out of scope — v1 has no auth surface (SRV-03), unchanged by this phase. |
| V3 Session Management | no | No session concept introduced; each stream is a stateless subscriber registration. |
| V4 Access Control | no | Read-only surface unchanged; the streaming rpc carries no new privilege. |
| V5 Input Validation | yes (minimal) | The streaming rpc's request message (if it carries any parameters — e.g. a `since_generation` resume field) must go through the same typed-request validation discipline the 13 existing rpcs use; no new hand-rolled parsing. |
| V6 Cryptography | no | No new crypto surface. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A held connection (many open streaming subscribers) exhausting server file descriptors / goroutines | Denial of Service | D-04's bounded-per-subscriber channel already bounds per-subscriber memory; the publisher itself must bound total subscriber count or rely on the existing loopback-only, single-user threat model (SRV-01/SRV-02 already restrict who can even reach the listener — DNS-rebinding is the only remote vector, already closed by `originHostGuard`, unchanged by this phase). |
| A slow/malicious client never reading its stream, holding a goroutine open indefinitely | Denial of Service | D-02's cleared write-deadline is *per-request*, not unbounded-forever — the request's own `ctx` (derived from the underlying `http.Request`) is cancelled when the client disconnects; the publisher's subscriber-cleanup path must remove the subscriber's channel from the fan-out map on that cancellation, or its bounded channel alone is not sufficient to bound goroutine count. This is a genuine implementation-correctness requirement for the plan, not a new ASVS category. |
| Reconnect storm from many tabs reconnecting simultaneously after a server restart | Denial of Service (self-inflicted) | CONTEXT.md's own discretion note (exponential backoff with jitter) is the standard mitigation; jitter specifically prevents synchronized reconnect waves. |

No new ASVS category is introduced by this phase beyond what Phase 1's `SRV-02`/`SRV-03`
established; the origin/host guard and the read-only method-set guard both already cover the new
rpc automatically (it mounts through the same `NewUIServiceHandler`/`originHostGuard` wrapping),
though `readonly_test.go`'s `wantUIServiceMethods` fixture must be updated to include the new rpc's
name (currently 13 entries; will become 14) — an update, not a new control.

## Sources

### Primary (HIGH confidence — read directly this session)

- `internal/uiserver/server.go`, `handlers.go`, `originguard.go`, `readonly_test.go`, `spa.go` — this repo's own current transport/handler/guard code
- `internal/mcp/server.go` (pendingWriter, lines ~430-541) and `pending_writer_test.go` — FIX-01's actual shape
- `internal/watch/watcher.go`, `debounce.go`, `policy.go`, `soak_test.go` — the existing watch subsystem
- `internal/query/engine.go` (`IndexMeta`, `OpenAt`, `storeSubdir`), `internal/query/resolve.go` (`ResolveCodegraphDir`, `codegraphDirName`), `internal/query/status.go` (`Stale` computation)
- `internal/schema/graph.proto` (Meta message, lines 137-166)
- `internal/uiproto/uiv1/ui.proto` (GetStatusResponse, GetHealthResponse, service UIService block)
- `internal/uiproto/uiv1/uiv1connect/ui.connect.go` (`NewUIServiceHandler` generated shape)
- `web/node_modules/.pnpm/@connectrpc+connect-web@2.1.2.../connect-transport.js` — the browser stream() implementation
- `web/node_modules/.pnpm/@connectrpc+connect@2.1.2.../protocol/envelope.js` — the incremental envelope decoder
- `web/node_modules/cytoscape-elk/src/layout.js`, `defaults.js`, `assign.js` — the cytoscape↔ELK adapter
- `web/pnpm-lock.yaml` — resolved `elkjs@0.9.3`
- `web/src/lib/status.ts`, `web/src/routes/+layout.svelte`, `web/src/routes/+page.svelte` — the current status/chrome components
- `/Users/sean/go/pkg/mod/connectrpc.com/connect@v1.20.0/protocol.go`, `protocol_connect.go` — `checkServerStreamsCanFlush`, `flushResponseWriter`, `connectStreamingHandlerConn.Send`
- `.planning/phases/05-file-package-graph-view/05-07-SUMMARY.md`, `05-08-SUMMARY.md` — the inherited displacement/hit-testing findings

### Secondary (MEDIUM confidence — official docs/maintainer statements, via Context7/WebSearch, cross-checked against primary source where possible)

- `eclipse-elk/elk#355` (GitHub issue, ELK maintainer `uruuru`'s direct answer) — "cannot precisely fix positions... interactive strategies... somewhat preserve"
- `kieler/elkjs#100` (GitHub issue) — "out of scope and not well-suited for ELK layered"
- Eclipse ELK reference docs (`org-eclipse-elk-interactive`, `org-eclipse-elk-layered-crossingMinimization-semiInteractive`), via Context7 `/websites/eclipse_dev_elk`
- `cockroachdb/pebble` wiki Glossary + `docs/rocksdb.md` (Commit Pipeline, Manifest), via Context7 `/cockroachdb/pebble`
- `fsnotify/fsnotify` docs (parent-directory watch pattern, debounce idiom), via Context7 `/fsnotify/fsnotify`
- `connectrpc/connect-es` README + test suite (retry-with-backoff idiom, `createConnectTransport` JSDoc), via Context7 `/connectrpc/connect-es`
- Chrome for Developers, Page Lifecycle API docs (frozen-state network behavior), via WebSearch

### Tertiary (LOW confidence)

- None used as load-bearing for any claim in this document.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version number was confirmed against `go.mod`/`pnpm-lock.yaml` directly, and no new package is introduced.
- Architecture (streaming transport): HIGH — both the Go handler and browser client's incremental-delivery behavior were confirmed by reading the actual installed source, not inferred from docs.
- Architecture (layout stability): MEDIUM — the *limitation* is HIGH confidence (direct maintainer statement, cross-checked across two independent issue threads), but the *magnitude* of the limitation at this project's actual scale is unmeasured (that measurement is D-03's own mandated deliverable, not something this research session could produce).
- Pitfalls: HIGH for Pitfalls 1, 3, 5 (each confirmed by direct code/config read); MEDIUM for Pitfall 2 (maintainer statement, not independently reproduced against 0.9.3 in this session); MEDIUM for Pitfall 4 (the archtest boundary's existence was not confirmed, only its plausibility from a doc comment).

**Research date:** 2026-09-01
**Valid until:** 2026-10-01 (30 days — the dependency versions are stable and unlikely to move within this milestone; the ELK-maintainer findings are unlikely to change since they describe a long-standing structural limitation, not a recent regression)
