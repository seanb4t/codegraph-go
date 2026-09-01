# Phase 6: Live Push - Pattern Map

**Mapped:** 2026-09-01
**Files analyzed:** 9 (5 new Go, 2 new TS, 2 new Playwright scripts) + 3 modified
**Analogs found:** 6 verified / 9 new files. **3 have NO analog in this codebase** — say so
plainly per the repo notes, not a stretched substitute.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/uiserver/watchtimeout.go` | middleware | request-response (deadline-clearing wrapper) | `internal/uiserver/originguard.go` | role-match (only existing middleware in the package) |
| `internal/uiserver/livepublish.go` | service (fan-out publisher) | event-driven / pub-sub | `internal/watch/watcher.go` + `internal/watch/debounce.go` | partial — reusable primitive, no existing fan-out registry to copy |
| `internal/uiserver/livehandler.go` | controller (streaming rpc handler) | streaming | **none** — first streaming rpc in this service | no analog |
| `internal/uiserver/livepublish_test.go` | test (soak/leak) | event-driven | `internal/watch/soak_test.go` | role-match |
| `internal/uiserver/livehandler_test.go` | test (streaming) | streaming | **none** — no existing streaming-handler test in this repo | no analog |
| `web/src/lib/live/live-client.ts` | provider/client wrapper | streaming | `web/src/lib/client.ts` | role-match (same transport, unary today) |
| `web/src/lib/live/live-store.ts` | store | event-driven | `web/src/lib/status.ts` (as a "fetch trigger" seam, explicitly named by its own doc comment as what Phase 6 replaces) | partial |
| `web/src/routes/**/+page.svelte` (graph route wiring) | component | event-driven → CRUD re-fetch | `web/src/lib/components/graph/GraphCanvas.svelte` (`runLayout`, `add`/`replace`/`removeByIds` seam) | role-match |
| `web/scripts/live-push-multitab-check.mjs` | test (real-browser) | event-driven | `web/scripts/graph-collapse-affordance-check.mjs` | exact (established pattern) |
| `web/scripts/graph-live-update-check.mjs` | test (real-browser) | event-driven / measurement | `web/scripts/graph-collapse-affordance-check.mjs` | exact (established pattern) |
| `internal/uiserver/readonly_test.go` (MODIFIED) | test (fixture) | — | itself | n/a — value-only edit, add the 14th name to `wantUIServiceMethods` |
| `internal/uiproto/uiv1/ui.proto` (MODIFIED) | config/schema | — | existing 13-rpc `service UIService` block | exact |

## Pattern Assignments

### `internal/uiserver/watchtimeout.go` (middleware, request-response)

**Analog:** `internal/uiserver/originguard.go` (full file, 51 lines read)

This is the only middleware that already exists in the package, and it establishes the
pass-through-unwrapped discipline the RESEARCH's own Anti-Patterns section calls out as
mandatory for D-02: `originHostGuard` takes `next http.Handler` and returns
`http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {...})` — `w` is passed
straight to `next.ServeHTTP` with no wrapping type. D-02's deadline-clearing middleware
MUST follow the same shape or `checkServerStreamsCanFlush` (connect-go's own gate) will
reject the handler.

**Wrapping pattern** (`internal/uiserver/originguard.go:37-44`):
```go
func originHostGuard(port string, next http.Handler) http.Handler {
	hosts := allowedHosts(port)
	origins := allowedOrigins(port)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := hosts[r.Host]; !ok {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		// ... origin check, then next.ServeHTTP(w, r) unwrapped
```

**Composition site** (`internal/uiserver/server.go:141`):
```go
guarded := originHostGuard(port, mux)
```
D-02's middleware must compose the same way — wrap `mux` (or `guarded`) once, at `Listen`
construction time, not per-handler. Use `http.NewResponseController(w).SetWriteDeadline(time.Time{})`
inside a handler that first checks whether the request targets the new streaming procedure
(match on `r.URL.Path`, the same string the generated `uiv1connect` constants expose) —
apply it ONLY to that one path, per D-02's "streaming path only" text; every other of the
13 unary rpcs plus the SPA handler must keep the 60s bound (`server.go:47-50`) untouched.

**Timeout constants to leave alone** (`internal/uiserver/server.go:47-50`):
```go
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)
```
`TestListenSetsEveryServerTimeout` asserts this set; D-02 does not touch it — it clears the
deadline `http.Server.WriteTimeout` establishes, per-request, only for the stream.

---

### `internal/uiserver/livepublish.go` (service/publisher, event-driven)

**Analog (open-snapshot-close discipline):** `internal/uiserver/handlers.go:26-61` (`openEngine` var + `withEngine`)

This is the binding discipline criterion 5 requires: never hold a store handle across the
stream's lifetime.

```go
// internal/uiserver/handlers.go:26
var openEngine = query.OpenAt

// internal/uiserver/handlers.go:47-61
func withEngine(ctx context.Context, repoPath string, fn func(*query.Engine) error) error {
	if err := ctx.Err(); err != nil {
		return mapContextError(err)
	}
	eng, closer, err := openEngine(repoPath)
	if err != nil {
		return mapEngineError(err)
	}
	defer closer.Close()
	if err := fn(eng); err != nil {
		return mapEngineError(err)
	}
	return nil
}
```
The new watcher's debounced wake-up handler must reuse this exact shape: `openEngine` (or
`query.OpenAt` directly) → `eng.IndexMeta()` → close, on **every** check — never once at
publisher-startup and held. `IndexMeta` itself:

```go
// internal/query/engine.go:158-167
func (e *Engine) IndexMeta() (*schema.Meta, error) {
	meta, err := e.reader.GetMeta()
	if err != nil {
		if errors.Is(err, graphstore.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return meta, nil
}
```

**Analog (debounce reuse):** `internal/watch/debounce.go:1-40` (`Debouncer`, `defaultDebounceMs = 2000`)

```go
// internal/watch/debounce.go:13-24
const defaultDebounceMs = 2000

func DebounceDuration() time.Duration {
	if v := os.Getenv("CODEGRAPH_DEBOUNCE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultDebounceMs * time.Millisecond
}
```
`Debouncer` coalesces a burst of `Add` calls into one `flush(paths map[string]struct{})`
call. **Verified this session: no archtest restricts who may IMPORT `internal/watch`.**
`internal/graphstore/archtest/import_graph_test.go` only restricts importers of
`pebble/v2`; `find internal/uiserver -iname '*archtest*'` returns nothing; no test anywhere
asserts an inbound-dependency allowlist for `internal/watch`. `internal/watch`'s own doc
comment (`watcher.go:9-11`) restricts what `internal/watch` may import OUT (only
`internal/indexer`'s `ShouldSkipDir`), which is the opposite direction — it says nothing
about `internal/uiserver → internal/watch`. **Conclusion: a straight import of
`watch.Debouncer`/`watch.DebounceDuration` from `internal/uiserver` is unblocked as of this
session** — RESEARCH's Open Question 2 is resolved: no blocking archtest exists.

**Analog (recursive-vs-flat watch, and why NOT to copy `Watcher` verbatim):**
`internal/watch/watcher.go:9-12,59` — `Watcher` walks the tree and re-`Add`s on `Create`
because it watches a *source* tree. `.codegraph/store/` is flat (11 entries, zero
subdirectories per RESEARCH's own `ls` check) — the new store-watcher should call
`fsnotify.NewWatcher()` + a single non-recursive `.Add(storeDir)`, NOT reuse `watch.Open`/
`addRecursive`. This is Don't-Hand-Roll guidance already captured in RESEARCH; restated here
because it is a "what NOT to copy" call, which matters for the planner's file boundary.

**No analog exists for:** the fan-out subscriber registry itself (`map[uint64]chan Event`
guarded by `sync.Mutex`, D-04's bounded-coalescing send). Nothing in this codebase currently
holds a live multi-subscriber channel map — say so plainly, this is genuinely new
plumbing, not a copy.

---

### `internal/uiserver/livehandler.go` (controller, streaming) — NO ANALOG

**No existing file in this repo implements a Connect server-streaming handler.** All 13
current `UIServiceHandler` methods are unary (`Search`, `Files`, `FileGraph`, etc., per
`internal/uiserver/readonly_test.go:57-70`'s own `wantUIServiceMethods` fixture). The
closest thing to a shape reference is RESEARCH's own confirmed-by-source-read excerpt of
connect-go's `ServerStreamForHandler.Send()` (`protocol_connect.go:809-814` in the
installed module, not this repo), which flushes after every `Send()` automatically — no
manual `Flush()` needed. Do not force-fit a unary handler (e.g. `GetHealth`,
`FileGraph`/`FileSymbols` from Phase 5) as a structural analog: the request/response
shape, the `withEngine` single-call boilerplate, and the subscriber-registration/cleanup
lifecycle are categorically different for a stream. Reuse `withEngine`'s
open-snapshot-close discipline only for the one-shot `Meta` read inside `livepublish.go`,
not for the handler itself — the handler's job is registering/deregistering a subscriber
channel and looping `Send()`, which has no unary precedent to imitate.

**Explicitly NOT to copy:** `GetStatus` (`internal/uiserver/handlers.go`, per repo notes) —
documented as the one degrade-and-answer exception among the 13 unary rpcs; it is a
special case, not the base shape for anything new.

---

### `web/src/lib/live/live-client.ts` (client wrapper, streaming)

**Analog:** `web/src/lib/client.ts` (full file, 30 lines read)

```typescript
// web/src/lib/client.ts:23-29
export const transport = createConnectTransport({
	baseUrl: "/",
	useBinaryFormat: false,
});

export const uiClient = createClient(UIService, transport);
```
The live client reuses this SAME `transport` (same-origin `baseUrl: "/"`, JSON encoding
per D-08) — do not construct a second transport instance. Call the generated streaming
method on `uiClient` (once `ui.proto` regenerates it) and consume via `for await`, per
RESEARCH's Pattern 2 (`connect-transport.js`'s incremental `ReadableStreamDefaultReader`
read loop) — never collect into an array first.

**No analog for reconnect/backoff** — nothing in this codebase currently reconnects a
Connect stream. RESEARCH's own Code Examples section cites connect-es's own test-suite
idiom (`connect-transport.spec.ts`, not this repo) as the reference shape; treat that as
external prior art, not an in-repo analog.

---

### `web/src/lib/live/live-store.ts` (store, event-driven)

**Analog (partial):** `web/src/lib/status.ts:1-70`

`status.ts`'s own doc comment names itself the seam Phase 6 replaces: *"the ONE fetch
trigger this app has for 'on load, on navigation, nothing else' — Phase 6 replaces this
mechanism with live push."* Its `classifyStatus(response: GetStatusResponse)` pure-function
shape (no I/O, exhaustive field-combination switch, one `UNKNOWN_STATUS` fallback) is the
pattern to carry into whatever function turns a live event into a `StatusVerdict`-shaped
update — **but per D-07, the event's field set now mirrors `GetStatusResponse`'s own names**
(`initialized`, `stale`, `store_exists`, `indexing_in_progress`, `commit_sha`, plus
`generation`), so `classifyStatus` itself may be directly callable against the event with
no translation layer — confirm this at plan time by diffing the final event proto against
`classifyStatus`'s destructured fields (`status.ts:60-68`, read this session).

```typescript
// web/src/lib/status.ts:60-68
export function classifyStatus(response: GetStatusResponse): IndexStatus {
	const commit: CommitKnowledge = response.commitSha ? 'known' : 'unknown';
	const commitSha = response.commitSha;

	if (response.initialized) {
		return { verdict: response.stale ? 'stale' : 'ok', commit, commitSha };
	}
	if (!response.storeExists) {
		return { verdict: 'no-index', commit, commitSha };
	}
	if (response.indexingInProgress) {
```

**No analog for the store/broadcast half** (distributing one event to N subscribed Svelte
routes). No pub-sub Svelte store exists in this codebase today — this is new.

---

### Graph route live-update wiring (component, event-driven → CRUD re-fetch)

**Analog:** `web/src/lib/components/graph/GraphCanvas.svelte:251-` (`runLayout`) and its
`add`/`replace`/`removeByIds` seam (per repo notes).

```typescript
// web/src/lib/components/graph/GraphCanvas.svelte:239-251
let liveAddedElements: any[] = [];

function runLayout(layoutRunStartedAt: number, fit: boolean, resizeAfter = false) {
	opts.cy.one('layoutstop', () => {
		if (resizeAfter && typeof opts.cy.resize === 'function') {
			opts.cy.resize();
		}
		...
```
D-06's write-back (pin every surviving node's prior x/y after ELK's interactive-strategy
layout) integrates here: after a live event triggers a `FileGraph`/`FileSymbols` re-fetch,
diff the returned node set against the currently-rendered one, call `runLayout` only when
nodes were added/removed (D-03's two-path fast-path decision), and write back captured
positions for survivors post-`layoutstop`. This component already distinguishes "replace"
vs "add" element sets — extend that distinction rather than introducing a third code path.

---

### `web/scripts/live-push-multitab-check.mjs` and `web/scripts/graph-live-update-check.mjs` (real-browser tests)

**Analog:** `web/scripts/graph-collapse-affordance-check.mjs` (first 50 lines read; pattern
verified)

Established, must-follow shape for both new scripts:
- `#!/usr/bin/env node` + `import { chromium } from '@playwright/test'`
- `repoRoot()` helper resolving via `import.meta.url`
- `pollUntil(pg, check, timeoutMs, pollMs)` polling helper for async UI settling
- Trusted, real input only (`page.mouse`, real navigation) — never `dispatchEvent`
- **A diagnostic JSON record is written on every run, including failure** — not just on
  success. This is the load-bearing convention: `live-push-multitab-check.mjs` must record
  per-tab timing/backpressure evidence (criterion 2's message-to-message latency, not just
  "arrived"), and `graph-live-update-check.mjs` must record the guava-scale survivor
  displacement measurement D-03/D-06 mandate, in both cases whether the run passes or fails.

```javascript
// web/scripts/graph-collapse-affordance-check.mjs:31-49
function repoRoot() {
	const here = path.dirname(url.fileURLToPath(import.meta.url));
	return path.resolve(here, '..', '..');
}

async function pollUntil(pg, check, timeoutMs, pollMs) {
	const deadlineAt = Date.now() + timeoutMs;
	for (;;) {
		const value = await check();
		if (value) return value;
		if (Date.now() >= deadlineAt) {
			throw new Error(`pollUntil: condition did not become true within ${timeoutMs}ms`);
```

---

### `internal/uiserver/readonly_test.go` (MODIFIED — value-only edit)

**Analog:** itself, `internal/uiserver/readonly_test.go:57-70` and `:118-121`

```go
// wantUIServiceMethods — ADD the 14th entry here (the new rpc name), nothing else
var wantUIServiceMethods = map[string]struct{}{
	"GetStatus": {}, "Search": {}, "Files": {}, "Callers": {}, "Callees": {},
	"Impact": {}, "Affected": {}, "GetNodeDetail": {}, "Explore": {},
	"GetPermalink": {}, "GetHealth": {}, "FileGraph": {}, "FileSymbols": {},
}
```
```go
// mutatingVerbs (internal/uiserver/readonly_test.go:118-121) — MUST NEVER be modified,
// per repo notes. The new rpc name must be re-verified CLEAN against this exact list,
// not assumed from CONTEXT.md's "verified this session" note, before the proto freeze.
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}
```
This is filling in a value in a shape the fixture already declares, per this repo's own
`wantUIServiceMethods`/`mutatingVerbs` structure — not inventing structure.

## Shared Patterns

### Open-snapshot-close discipline (SRV-04)
**Source:** `internal/uiserver/handlers.go:26-61`
**Apply to:** `livepublish.go`'s every debounced wake-up check. Never hold `*query.Engine`
or `graphstore.Reader` across the publisher's lifetime or the stream's lifetime.

### Middleware pass-through (no `http.ResponseWriter` wrapping)
**Source:** `internal/uiserver/originguard.go:37-44`
**Apply to:** `watchtimeout.go`. Any wrapping type that hides `http.Flusher`/
`http.ResponseController`'s optional interfaces breaks connect-go's own
`checkServerStreamsCanFlush` gate.

### Same-origin transport reuse
**Source:** `web/src/lib/client.ts:23-29`
**Apply to:** `live-client.ts` — reuse the existing `transport` const, do not construct a
second `createConnectTransport` instance.

### Real-browser check script shape (committed diagnostic JSON, every run)
**Source:** `web/scripts/graph-collapse-affordance-check.mjs:1-49`
**Apply to:** both new Playwright scripts.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/uiserver/livehandler.go` | controller | streaming | First streaming rpc among 13 unary ones in this service — no in-repo handler shape to copy. Use RESEARCH's Pattern 1 (connect-go source excerpt) instead. |
| `internal/uiserver/livehandler_test.go` | test | streaming | No existing streaming-handler test exists to imitate; must assert message-by-message timing (`httptest`), not "all arrived" — RESEARCH's criterion-2 warning. |
| Fan-out subscriber registry (inside `livepublish.go`) | service (pub-sub core) | event-driven / pub-sub | No multi-subscriber channel map exists anywhere in this codebase today. D-04's bounded-coalescing send logic is new. |
| `live-store.ts`'s broadcast-to-routes half | store | pub-sub | No Svelte pub-sub store exists in this codebase; `status.ts` is a single-fetch-trigger seam, not a broadcast mechanism — reusable only for its `classifyStatus`-shaped pure-function half, not the distribution half. |
| Reconnect/backoff logic (`live-client.ts`) | client wrapper | streaming | No stream-reconnect code exists in this repo; RESEARCH cites connect-es's own external test-suite idiom, not an in-repo analog. |

## Metadata

**Analog search scope:** `internal/uiserver/`, `internal/watch/`, `internal/query/engine.go`,
`internal/graphstore/archtest/`, `web/src/lib/`, `web/src/lib/components/graph/`,
`web/scripts/`
**Files scanned (read this session):** `internal/uiserver/handlers.go`,
`internal/uiserver/server.go`, `internal/uiserver/originguard.go`,
`internal/uiserver/readonly_test.go`, `internal/watch/debounce.go`,
`internal/watch/watcher.go`, `internal/watch/policy.go`,
`internal/graphstore/archtest/import_graph_test.go`, `internal/query/engine.go`,
`web/src/lib/client.ts`, `web/src/lib/status.ts`,
`web/src/lib/components/graph/GraphCanvas.svelte`,
`web/scripts/graph-collapse-affordance-check.mjs`
**Pattern extraction date:** 2026-09-01
