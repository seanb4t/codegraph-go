# Phase 3: `2026-07-28` Spec Compliance - Research

**Researched:** 2026-08-06
**Domain:** MCP `2026-07-28` wire contract (SEP-2575 `server/discover` + per-request `_meta`) on `modelcontextprotocol/go-sdk@v1.7.0`, for a stdio, tools-only, already-migrated server
**Confidence:** HIGH — every SDK-behavior claim below is `[VERIFIED]` against the real `v1.7.0` source in `$GOMODCACHE` read this session, cross-confirmed by building `bin/codegraph` from this branch's HEAD and driving it over real stdio with the exact byte sequences quoted. The one place this research could not get a clean, unambiguous answer (SPEC-06's era-vs-tool-call coverage) is flagged LOW and left as an explicit recommendation rather than an assumption.

## Summary

This phase inherited more than the milestone expected (D-01/D-02) and needs to build less than REQUIREMENTS.md's language implies for SPEC-02. The three genuine build items are exactly the three CONTEXT.md's D-03/D-04/D-05 named: `cacheScope` on `server/discover` (one line, same seam as D-09's `tools/list` fix), the `instructions` string (one field, same seam), and SPEC-05's dynamic tool catalog (a real design decision, now fully de-risked by source evidence: `AddTool`/`RemoveTools` are `s.mu`-protected and safe to call from any goroutine, including from inside `AddReceivingMiddleware`, while requests are in flight).

**The single most important finding this session adds:** SPEC-02's `-32601`-instead-of-`-32022` behavior CONTEXT.md flagged as "the least understood item" is now fully traced to its root cause, and it is **not a codegraph-go gap** — it is a documented, source-verified property of go-sdk's own unexported `validateRequestMeta` (`shared.go:543-576`), which uses a **plain lexical string comparison** (`protocolVersion < protocolVersion20260728`) to decide whether an incoming request "uses the new protocol" at all. Any `_meta.protocolVersion` value that sorts lexically *before* `"2026-07-28"` — including nonsense like `"1999-01-01"`, and including every genuinely-supported Legacy version string — is silently reclassified as "not modern," never reaches the `-32022` check, and (for a Modern-only method like `server/discover`) falls into the ordinary method-availability gate instead, which answers `-32601` with a **message that is itself further overwritten** by the JSON-RPC transport's generic `errors.Is`-based rewriter (`internal/jsonrpc2/conn.go:694-695`) before it reaches the wire. Both steps were read in source and reproduced byte-for-byte against the real binary. codegraph-go has **no seam** to intervene earlier — `validateRequestMeta` runs inside `ss.handle` *before* `AddReceivingMiddleware`'s chain is ever invoked (PROVEN-ABSENT, matching the exact pattern Phase 2's Q1 established for protocol-version injection). For a *well-formed* unsupported version — the only shape a real Modern client would ever construct — `-32022` already fires correctly, verified empirically. SPEC-02's `-32602` half is also already fully correct and applies uniformly to every method, not just `initialize`/`discover`, also verified empirically.

**Second finding:** adopting `AddTool`/`RemoveTools` for SPEC-05 (D-05's already-locked mechanism) is *safe* — every mutator and every reader (`listTools`, `getServerTool`, `capabilities()`) is guarded by the same `s.mu`, and none of them hold that lock across handler execution, so calling `AddTool` from inside the existing `AddReceivingMiddleware` closure cannot deadlock and cannot race a concurrent `tools/call`. It also does **more than asked**: `changeAndNotify` unconditionally fires `notifications/tools/list_changed` to every Legacy session the moment new tools are registered (no opt-in required) — a real, free, in-scope improvement for Legacy clients that does **not** pull SPEC-09 forward, because Modern sessions only receive that notification if they hold an active `subscriptions/listen` stream, which Phase 3 does not build.

**Third finding, confirmed by actually running the freeze gate:** the SDK-01 exemption genuinely does not cover a normal Phase 3 diff. Simulated against a post-Phase-2 base with a synthetic `internal/mcp` + transcript change and no `go.mod` line touched, `task check:transcript-freeze` exits 1 with a hard Violation. This must be planned for explicitly (see Q6 below); it is not a hypothetical.

**Primary recommendation:** implement SPEC-04/SPEC-07 as one-line additions to the existing `AddReceivingMiddleware` switch in `BuildServer` (exactly mirroring D-09's `tools/list` pattern); implement SPEC-05 as a per-request pre-check inside that same middleware that calls the existing tool-registration logic (factored out) via `AddTool` when `hasIndex` flips false→true; treat SPEC-02 as **already satisfied for every realistic client shape** and spend the phase's SPEC-02 budget on *proving* that with new wire-oracle scenarios, not on writing validation code that has no seam to attach to.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Per-request `_meta` validation (SPEC-02) | API/Backend (in-process go-sdk library) | — | Wholly owned by go-sdk's unexported `validateRequestMeta`/`ss.handle`; codegraph-go has no server-side control point (Q1, PROVEN-ABSENT, same class as Phase 2's protocol-version finding) |
| `server/discover` response shape (SPEC-01/03/04/07/08) | API/Backend | — | `Server.discover` + `AddReceivingMiddleware`, in-process, no client/runtime component |
| Dynamic tool catalog (SPEC-05) | API/Backend | — | `mcp.AddTool`/`RemoveTools` + `internal/mcp`'s own per-request middleware; `internal/watch`'s fsnotify watcher is explicitly NOT this tier's mechanism (see Q2) |
| Legacy tool-call coverage (SPEC-06) | Database/Storage (test fixture) + API/Backend (driver) | — | Frozen `.golden` transcripts + the wire-oracle capture CLI; no client tier involved |
| CI regeneration gate (Q6) | Build tooling | — | `tools/transcriptfreeze` + `.github` CI job; not a runtime tier, but load-bearing for whether the phase's own PR can merge |

This phase touches exactly one runtime tier — the in-process Go server library — consistent with Phase 2's finding and the milestone's stdio-only framing. No browser, SSR, CDN, or persistent-storage surface is in scope.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPEC-01 | `server/discover` answers with capabilities, no tool call first | Already inherited (D-01); Q3/Q4 confirm the exact response fields and that nothing in this phase's changes can regress it |
| SPEC-02 | Per-request `_meta` validated: `-32602` malformed/missing, `-32022` unsupported version | Q1 — full trace of both codes' actual firing conditions, root-caused the `-32601` observation, empirically confirmed 6 distinct `_meta` shapes against the real binary |
| SPEC-03 | Every tool result carries `resultType:"complete"` | Q1 — confirmed this is gated on `validatedMeta.usesNewProtocol` (a per-request property), not a global server setting; already correct for every Modern-shaped call, empirically confirmed |
| SPEC-04 | `tools/list` + `server/discover` carry `ttlMs:0`, `cacheScope:"private"` | Q4 — confirms the exact one-line fix and that `AddReceivingMiddleware` reaches discover exactly like it already reaches `tools/list` |
| SPEC-05 | `hasIndex` re-checked per call, not snapshotted at construction | Q2 — full concurrency-safety trace of `AddTool`/`RemoveTools`, the existing watcher's explicit non-applicability (documented in `serve.go`), and the recommended per-request-middleware mechanism |
| SPEC-06 | A Legacy client (2025-11-25 and earlier) completes a session and calls a tool | Q5 — confirms which existing scenario already proves this and which eras still lack tool-call coverage |
| SPEC-07 | `server/discover`'s `instructions` carries usage guidance | Q3 — confirms `ServerOptions.Instructions` reaches both `initialize` and `discover` via the identical field, no length/format constraint in the type system |
| SPEC-08 | Tool results carry `io.modelcontextprotocol/serverInfo` in `_meta` | Already inherited (D-01); Q1 confirms it is stamped by `annotateServerInfo`, gated the same way as SPEC-03 |
</phase_requirements>

## Standard Stack

No new external packages are required for this phase. `modelcontextprotocol/go-sdk@v1.7.0` is already the sole MCP dependency (landed in Phase 2); every SPEC-01…08 item is implemented against APIs that dependency already exposes. `go.mod` is not expected to change.

## Package Legitimacy Audit

**Skipped — this phase installs no new packages.** `go.mod`'s only MCP-related line (`github.com/modelcontextprotocol/go-sdk v1.7.0`, `[VERIFIED: go.mod:14]`) is unchanged from Phase 2's audited state.

## Architecture Patterns

### System Architecture Diagram

```
                     stdio (JSON-RPC, newline-delimited)
                              │
                              ▼
                   internal/cli/serve.go
                              │  mcp.NewStdioServer(...)
                              ▼
                internal/mcp.Server (goSDKServer)
                              │  s.Run(ctx, IOTransport{...})
                              ▼
     ┌────────────────────────────────────────────────────────┐
     │ go-sdk internal dispatch (ss.handle, server.go:1852)     │
     │                                                          │
     │  1. validateRequestMeta(req)   ◄── SPEC-02's whole gate  │
     │     ├─ no _meta at all              → Legacy, no error   │
     │     ├─ _meta.protocolVersion         │                   │
     │     │     < "2026-07-28" (lexical)  → Legacy, no error   │
     │     │     (catches BOTH real Legacy  │  (THIS is the     │
     │     │      versions AND garbage!)    │   -32601 trap)    │
     │     ├─ _meta.protocolVersion         │                   │
     │     │     >= "2026-07-28",           │                   │
     │     │     clientCapabilities         │                   │
     │     │     missing/invalid           → -32602 (correct)   │
     │     └─ _meta.protocolVersion         │                   │
     │           >= "2026-07-28", valid,    │                   │
     │           not in supportedVersions  → -32022 (correct)   │
     │                                                          │
     │  2. method-availability switch (server.go:1879)          │
     │     "server/discover" + !usesNewProtocol → -32601        │
     │        (message THEN overwritten by conn.go:694-695's    │
     │         generic errors.Is(ErrMethodNotFound) rewrite —   │
     │         WireError.Is compares by Code only, wire.go:88)  │
     │                                                          │
     │  3. handleReceive → s.receivingMethodHandler_             │
     │     (AddReceivingMiddleware's chain — internal/mcp's      │
     │      OWN code starts having a say HERE, never earlier)    │
     │        │                                                  │
     │        ▼                                                  │
     │  4. Server.discover / .initialize / .listTools / .callTool│
     │     (all s.mu-guarded; AddTool/RemoveTools also s.mu-     │
     │      guarded, safe to call concurrently — Q2)             │
     │                                                          │
     │  5. usesNewProtocol? → setCompleteResultType +             │
     │     annotateServerInfo (SPEC-03/08, per-request gated)    │
     └────────────────────────────────────────────────────────┘
                              │
                              ▼
                 internal/query.Engine (UNCHANGED)
```

A reader can trace SPEC-02's entire failure surface without leaving step 1, and can trace exactly why "server/discover" with an old-looking `_meta.protocolVersion` produces a misleading `-32601` without leaving steps 1–2. `internal/mcp`'s own code (`AddReceivingMiddleware`) never sees a request that step 1 or step 2 rejected — this is why there is no seam to "fix" the `-32601` case from codegraph-go's side.

### Recommended Project Structure

No new files are required. This phase modifies:
```
internal/mcp/
├── server.go     # BuildServer: add ServerOptions.Instructions; extend the
│                  # existing AddReceivingMiddleware switch with a
│                  # "server/discover" case (cacheScope:"private", mirroring
│                  # the existing "tools/list" case); factor tool
│                  # registration into a reusable func; add a per-request
│                  # hasIndex re-check that calls it when the index appears
│                  # mid-session; widen the session-line toolCount capture
│                  # from a plain int to something mutation-safe (see
│                  # Common Pitfalls)
test/wireoracle/
└── scenarios.go   # new scenarios (Q7); ExpectedScenarioCount moves off 23
tools/transcriptfreeze/
└── (no code change expected — see Q6: this is a workflow/sequencing
    question, not a classifier-logic gap)
```

### Pattern 1: Extending the existing per-request middleware switch (SPEC-04/SPEC-05/SPEC-07's seam)

**What:** `internal/mcp/server.go`'s `BuildServer` already registers one `AddReceivingMiddleware` closure with a `switch method` handling `"initialize"` (session line) and `"tools/list"` (`cacheScope` correction, D-09). This is the exact, already-proven seam for `"server/discover"`'s `cacheScope` fix — `discover()` calls the identical `res.setDefaultCacheableValues()` method `listTools()` calls (`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:903-909]`):
```go
res := &DiscoverResult{
    SupportedVersions: versions,
    Capabilities:      s.capabilities(),
    Instructions:      s.opts.Instructions,
}
res.setDefaultCacheableValues()
return res, nil
```
and `DiscoverResult` embeds the same `Cacheable` struct `ListToolsResult` does (`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:1138-1148]`: `type DiscoverResult struct { completeResultWithType; Meta \`json:"_meta,omitempty"\`; Cacheable; SupportedVersions []string \`json:"supportedVersions"\`; Capabilities *ServerCapabilities \`json:"capabilities"\`; Instructions string \`json:"instructions,omitempty"\` }`), so the fix is one new `case "server/discover":` branch doing `if discoverRes, ok := res.(*mcp.DiscoverResult); ok { discoverRes.CacheScope = "private" }` — byte-identical in shape to the existing `"tools/list"` case already in `internal/mcp/server.go:452-464` `[VERIFIED: internal/mcp/server.go:452-464, read this session]`.

**When to use:** SPEC-04's discover half, and (if the plan wants a single check site) SPEC-05's per-request `hasIndex` re-check.

### Pattern 2: Dynamic tool registration via `AddTool`/`RemoveTools`, triggered from the same middleware

**What:** D-05 already locked `AddTool`/`RemoveTools` mutation over per-call filtering. Both are `s.mu`-guarded through `changeAndNotify` (`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:705-724]`):
```go
func (s *Server) changeAndNotify(notification string, change func() bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if change() && s.shouldSendListChangedNotification(notification) {
        ...
    }
}
```
and every reader that a concurrent `tools/call`/`tools/list` dispatches through — `listTools` (`s.mu.Lock(); defer s.mu.Unlock()`, `[VERIFIED: server.go:930-931]`), `getServerTool` (`s.mu.Lock(); defer s.mu.Unlock()`, `[VERIFIED: server.go:950-952]`), `capabilities()` (`s.mu.Lock(); defer s.mu.Unlock()`, `[VERIFIED: server.go:616-617]`) — acquires and releases the *same* `s.mu` briefly, never holding it across handler execution. `featureSet[T]` itself (the type backing `s.tools`) has **no internal lock of its own** (`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/features.go:23-27]`: `type featureSet[T any] struct { uniqueID func(T) string; features map[string]T; sortedKeys []string }`) — it relies entirely on `Server`'s own `s.mu` discipline, which every exported mutator/reader already follows. Calling `mcp.AddTool` from inside `AddReceivingMiddleware`'s closure body is therefore safe: the dispatch path that invokes middleware does not hold `s.mu` while doing so (`receivingMethodHandler()` itself only holds the lock long enough to read the field, `[VERIFIED: server.go:1841-1846]`), so there is no self-deadlock and no race with a concurrently-executing tool handler.

**When to use:** SPEC-05's dynamic catalog. Trigger point: inside the existing middleware, **before** calling `next(...)` (so the same call that observes the index — e.g. a `tools/list` sent right after `codegraph init` completes — already reflects the new registration), re-resolve `hasIndex` cheaply (the same `query.ResolveCodegraphDir` check `serveServerPaths` already does once at startup, `[VERIFIED: internal/cli/serve.go:42-50]`) and call the (factored-out) registration loop if it flipped false→true.

### Anti-Patterns to Avoid

- **Importing `mcp.MetaKeyProtocolVersion` or `mcp.CodeUnsupportedProtocolVersion` anywhere in the module.** Both are external `*types.Const` values whose names match `internal/mcp/archtest`'s VRFY-02 name heuristic (`(?i)protocol.?version`) even though neither is a protocol-version *value* pin — the guard's own doc comment names exactly this pair as a known false-positive-shaped surface for "Phase 3's SPEC-02/SPEC-08 work" (`[VERIFIED: internal/mcp/archtest/protocol_version_test.go:31-42, read this session]`, doc comment text quoted verbatim: *"Referencing either directly (e.g. Phase 3's SPEC-02/SPEC-08 work on the 2026-07-28 \`_meta\` obligations) will read as a VRFY-02 violation under this guard"*). Critically, **there is no code-level allowlist** — `scanForProtocolVersionRefs` (`[VERIFIED: internal/mcp/archtest/protocol_version_test.go:169-188, read this session]`) is a pure structural walk with no exception table; the doc comment's "record a reviewed exception in this file's doc comment" is prose-only and does **not** suppress `TestNoExternalProtocolVersionConstantReferences`. The correct avoidance is the wire oracle's own established convention (D-02/D-06): hand-author the literal `-32022` and the literal string `"io.modelcontextprotocol/protocolVersion"` wherever needed (scenarios.go, a new anchor), exactly as `anchors.go` already hand-authors `codeMethodNotFound = -32601` and `codeInvalidParams = -32602` (`[VERIFIED: test/wireoracle/anchors.go:19-22, read this session]`) rather than importing `mcp.CodeMethodNotFound`. This sidesteps the guard entirely — no exception, no allowlist edit, no risk of a real false-positive-vs-real-regression ambiguity.
- **Attempting to intercept `_meta` validation earlier than go-sdk's own dispatch.** There is no seam (Q1) — `validateRequestMeta` and the method-availability switch both run inside the unexported `ss.handle` before `AddReceivingMiddleware`'s chain is invoked. Do not add defensive re-validation in codegraph-go's own middleware "just in case" — it cannot see rejected requests, and building code to duplicate go-sdk's own validation is exactly the kind of assurance-on-top-of-assurance the phase's calibration note forbids.
- **Re-deriving `hasIndex` via a full filesystem walk on every single request.** `query.ResolveCodegraphDir` is already the cheap check `serveServerPaths` uses once at startup (a bounded upward directory walk, not a recursive scan) — reuse it, do not build a new detection mechanism.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| `_meta.protocolVersion`/`clientCapabilities` validation | A codegraph-go-side `_meta` parser/validator | go-sdk's own `validateRequestMeta` | No seam exists to run codegraph-go code before it (Q1); it already produces the exact `-32602`/`-32022` codes SPEC-02 wants for every well-formed client shape |
| `notifications/tools/list_changed` delivery to Legacy sessions | A manual notification-send loop after `AddTool` | `changeAndNotify`'s built-in, debounced (`10ms`) broadcast | Already wired, already correct, and already scoped to NOT reach Modern sessions without an active `subscriptions/listen` (so it cannot accidentally pull SPEC-09 forward) |
| Tool-registry concurrency control | A new mutex around codegraph-go's own tool-registration call sites | `s.mu` (already held by every `AddTool`/`RemoveTools`/`listTools`/`getServerTool` call) | Building a second lock around calls into an API that's already internally synchronized is redundant and a deadlock risk if the two locks are ever acquired in different orders |

**Key insight:** this phase's actual engineering surface is smaller than REQUIREMENTS.md's SPEC-02 wording implies — the validation logic already exists and is already correct for every input shape a real Modern client produces. The work is (a) two one-line additions to an existing middleware switch, (b) one new per-request check plus a factored-out registration loop, and (c) proving all of the above with wire-oracle scenarios, not writing new validation code.

## Runtime State Inventory

Not applicable — this is an implementation phase, not a rename/refactor/migration. No stored data, live service config, OS-registered state, secrets, or build artifacts carry a name this phase changes.

## Common Pitfalls

### Pitfall 1: Mistaking a well-formed-but-old `_meta.protocolVersion` for an SDK bug

**What goes wrong:** Testing SPEC-02's `-32022` path with a version string that sorts lexically *before* `"2026-07-28"` (e.g. the milestone's own earlier probe, `"1999-01-01"`) produces `-32601`, not `-32022` — and the message is the generic `"method not found: \"server/discover\""`, not even the friendlier message go-sdk's own source writes at that rejection site.
**Why it happens:** `validateRequestMeta`'s classification (`shared.go:549`: `if !ok || protocolVersion < protocolVersion20260728`) is a plain Go string comparison — any `_meta.protocolVersion` lexically less than `"2026-07-28"` is reclassified as "not using the new protocol" *before* the unsupported-version check ever runs, and `server/discover` is Modern-only, so the method-availability switch (`server.go:1888-1897`) rejects it. The message is then further overwritten by `internal/jsonrpc2/conn.go:694-695`'s generic `errors.Is(err, ErrMethodNotFound)` rewrite, because `jsonrpc.Error = jsonrpc2.WireError` and `WireError.Is` compares by `Code` only (`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/internal/jsonrpc2/wire.go:88-94]`: `func (err *WireError) Is(other error) bool { w, ok := other.(*WireError); if !ok { return false }; return err.Code == w.Code }`).
**How to avoid:** Test SPEC-02's `-32022` path only with a version string that is lexically *greater than or equal to* `"2026-07-28"` and not one of the five supported values (e.g. `"2099-01-01"`, `"2026-07-29"`) — this is also the only shape a real Modern client would ever construct, since the whole point of a client sending `_meta.protocolVersion` at all is that it believes it speaks `>= 2026-07-28`. Empirically confirmed this session: `"2099-01-01"` → `{"code":-32022,"message":"unsupported protocol version","data":{"supported":[...],"requested":"2099-01-01"}}`.
**Warning signs:** A plan task phrased as "verify an unsupported `_meta.protocolVersion` returns `-32022`" without specifying the version's lexical relationship to `"2026-07-28"` will produce a flaky or wrong result depending on which literal the implementer picks.

### Pitfall 2: `toolCount`'s plain-`int` capture goes stale once SPEC-05 lands

**What goes wrong:** `internal/mcp/server.go`'s session-line middleware closes over `var toolCount int` (`[VERIFIED: internal/mcp/server.go:356-397, read this session]`), incremented once during `BuildServer`'s construction-time registration loop and never touched again. VRFY-03's session line (`codegraph: mcp-session ... tools=N`) reads this captured value on every `initialize`. Once SPEC-05 adds a mid-session registration path, a session that starts with `hasIndex=false` (`toolCount=0`) and later has tools appear will keep reporting `tools=0` in any *subsequent* session-line-relevant event, because nothing updates the closed-over `int`.
**Why it happens:** The variable was correctly designed for a startup-time-only registration model; SPEC-05 changes that model without touching every consumer of the count.
**How to avoid:** When adding SPEC-05's per-request registration path, also make `toolCount` mutation-safe (e.g. `atomic.Int64`, or read live off the newly-factored registration-count return value under the same `mu sync.Mutex` the session-line write already uses) and update it at the exact point new tools are registered. There is no SDK-exposed `Server` method to read the current tool count (`Server.Sessions()` is the only such iterator exposed; no `Server.Tools()`/count accessor exists, `[VERIFIED: rg over $GOMODCACHE/.../mcp/server.go "^func (s \*Server) [A-Z]", read this session — Sessions is the only public feature-count-adjacent accessor]`) — codegraph-go's own `toolCount` remains the only source of truth and must be kept in sync manually.
**Warning signs:** A wire-oracle transcript for "index appears mid-session" whose stderr session line still reports the pre-registration tool count after a tool clearly became callable.

### Pitfall 3: Reaching for `internal/watch`'s fsnotify watcher for SPEC-05

**What goes wrong:** SPEC-05's requirement text ("a running server sees the tools appear") sounds like a watcher problem, and `internal/watch`/`internal/cli/serve.go` already has an elaborate fsnotify-backed watcher (`serveWatchStart`). Reaching for it is the more "complete" option and therefore tempting.
**Why it happens:** `serveWatchStart`'s own doc comment states, verbatim, the opposite of what SPEC-05 needs: *"A no-op (cancel, already-closed done) pair is returned when !hasIndex (MCP-03: no watcher, but RunE's control flow stays uniform — the caller always defers cancel()/<-done unconditionally). hasIndex is a startup-time snapshot (IN-09, deliberate): an index created mid-session (\`codegraph init\` while serve is running) is served live by per-call query resolution but does NOT retroactively start the watcher — auto-sync begins on the next serve --mcp session"* (`[VERIFIED: internal/cli/serve.go:71-78, read this session]`). Reusing it for SPEC-05 would require re-architecting the watcher's own start condition (a v1.0-era decision, IN-09) and threading a new cross-package callback from `internal/watch` into `internal/mcp` — a materially larger and riskier change than the per-request check.
**How to avoid:** Use the per-request middleware check (Pattern 2 above), which needs no new goroutine, no fsnotify wiring, and matches SPEC-05's own literal wording ("re-checked per call").
**Warning signs:** A plan task that touches `internal/watch/policy.go` or `serveWatchStart`'s `!hasIndex` early-return for SPEC-05 — that is IN-09's decision boundary, not this phase's.

### Pitfall 4: Assuming the SDK-01 freeze-gate exemption covers this phase's transcript changes

**What goes wrong:** `task check:transcript-freeze` currently exits 0 (EXEMPTED) when run right now, because this branch's diff against `origin/main` still contains the Phase 2 `mark3labs → go-sdk` `go.mod` transition. A plan that assumes this state persists will be surprised when the actual Phase 3 PR (whenever it is opened, against whatever `main` looks like at that time) fails.
**Why it happens:** `sdkSwapExemption` (`[VERIFIED: tools/transcriptfreeze/classify.go:174-189, read this session]`) fires only when the diff contains a `-` line naming `github.com/mark3labs/mcp-go` AND a `+` line naming `github.com/modelcontextprotocol/go-sdk` — a shape that can only exist once, and specifically does not exist in a diff that starts from a base where the swap already landed.
**How to avoid:** See Q6 below — this is a sequencing/workflow question, not a code fix, and must be planned rather than discovered at PR time (CONTEXT.md's own words).
**Warning signs:** `task check:transcript-freeze` returning EXEMPTED during Phase 3 development against a stale local `origin/main` — always re-run against the actual PR base at merge time.

## Code Examples

### `_meta` validation matrix, empirically captured against `bin/codegraph` built from this branch's HEAD (Phase 2 complete, Phase 3 not started)

```
# Source: this session's own probes, piped into bin/codegraph serve --mcp over real stdio,
# run from the repo root (a real .codegraph/ index present, hasIndex=true)

1. server/discover, _meta.protocolVersion="1999-01-01" (lexically OLD), full valid _meta:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found: \"server/discover\""}}

2. server/discover, _meta.protocolVersion="2099-01-01" (lexically >= 2026-07-28, unsupported), full valid _meta:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32022,"message":"unsupported protocol version",
    "data":{"supported":["2026-07-28","2025-11-25","2025-06-18","2025-03-26","2024-11-05"],"requested":"2099-01-01"}}}

3. server/discover, protocolVersion="2026-07-28", clientCapabilities MISSING:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"missing or invalid _meta field \"io.modelcontextprotocol/clientCapabilities\""}}

4. server/discover, protocolVersion="2026-07-28", clientCapabilities malformed (a string, not object):
   {"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"missing or invalid _meta field \"io.modelcontextprotocol/clientCapabilities\""}}

5. server/discover, _meta present but NO protocolVersion key:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found: \"server/discover\""}}

6. server/discover, no _meta at all:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found: \"server/discover\""}}

7. Classic Legacy initialize(2025-11-25) session, then tools/call with per-request
   _meta.protocolVersion="2026-07-28" but clientCapabilities missing (mixed-protocol request
   mid an already-Legacy-negotiated session — proves _meta validation is per-request, not
   per-session):
   codegraph: mcp-session requested=2025-11-25 negotiated=2025-11-25 client=probe/1 tools=1
   {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":true}},"protocolVersion":"2025-11-25","serverInfo":{"name":"codegraph","version":"0.1.0"}}}
   {"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"missing or invalid _meta field \"io.modelcontextprotocol/clientCapabilities\""}}

8. server/discover, clientInfo present but malformed (a string, not object):
   {"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"invalid _meta field \"io.modelcontextprotocol/clientInfo\""}}

9. Sessionless (no prior initialize) tools/call, valid Modern _meta, unregistered tool name:
   {"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"unknown tool \"codegraph_status\""}}

10. Sessionless Modern tools/call on codegraph_explore (registered by default), valid Modern _meta:
    {"jsonrpc":"2.0","id":1,"result":{"resultType":"complete",
     "_meta":{"io.modelcontextprotocol/serverInfo":{"name":"codegraph","version":"0.1.0"}},
     "content":[{"type":"text","text":"**Exploration: Alpha** ..."}]}}

11. Classic Legacy initialize + classic tools/call, NO per-request _meta at all:
    {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"**Exploration: Alpha** ..."}]}}
    (no "resultType", no "_meta" — confirms SPEC-03/08 are per-request-Modern-gated, not global)
```

Case 1/5/6 vs case 2 is Pitfall 1's exact distinction. Case 3/4/8 confirm the `-32602` half of SPEC-02 is already fully correct and covers both missing and malformed shapes for both `clientCapabilities` and `clientInfo`. Case 7 confirms per-request `_meta` validation applies uniformly to any method, not just `initialize`/`discover`. Case 9/10 confirm SPEC-05's underlying statelessness already works today (a sessionless `tools/call` reaches codegraph-go's own tool registry) — Phase 3 does not need to build session-independence, only the dynamic-catalog re-check.

### Verified: `server/discover`'s handler body (unchanged by this phase, but every field Phase 3 touches is visible here)

```go
// Source: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:884-910
func (s *Server) discover(_ context.Context, req *ServerRequest[*DiscoverParams]) (*DiscoverResult, error) {
    ...
    res := &DiscoverResult{
        SupportedVersions: versions,
        Capabilities:      s.capabilities(),
        Instructions:      s.opts.Instructions,
    }
    res.setDefaultCacheableValues()
    return res, nil
}
```

### The existing middleware switch this phase extends

```go
// Source: internal/mcp/server.go:409-468, read this session (current state, Phase 2 complete)
s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
        res, err := next(ctx, method, req)
        if err != nil {
            return res, err
        }
        switch method {
        case "initialize":
            // ... VRFY-03 session line ...
        case "tools/list":
            // D-09: cacheScope "public" -> "private"
            if listRes, ok := res.(*mcp.ListToolsResult); ok {
                listRes.CacheScope = "private"
            }
        }
        return res, err
    }
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| CONTEXT.md's assumption that the `-32601` observation is a codegraph-go gap to close | Fully traced to go-sdk's own lexical version-string classification + the JSON-RPC transport's generic error-message rewrite; no codegraph-go seam exists at that stage | This session | Phase 3 should NOT plan a "fix" for the `1999-01-01`-shaped case; it should document the finding and test the shape a real client actually produces |
| CONTEXT.md's framing of SPEC-05's watch question as open between "existing watcher" and "per-call check" | Existing watcher is explicitly documented (`serve.go:71-78`) to be a no-op for the exact scenario SPEC-05 targets; per-call check is the only mechanism that matches both SPEC-05's wording and the watcher's own IN-09 boundary | This session | De-risks D-05's stated open risk; the concurrency half (`AddTool`/`RemoveTools` safety) is also now fully evidenced, not assumed |

**Deprecated/outdated:** none — Phase 2's research (`02-RESEARCH.md`) remains accurate for everything it covered; this document only extends it into `server/discover` and `_meta` territory Phase 2 explicitly left out of scope.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No real MCP client constructs a `_meta.protocolVersion` value that sorts lexically before `"2026-07-28"` while also setting `_meta.clientCapabilities` (i.e., Pitfall 1's edge case is not a realistic client shape) | Common Pitfalls, Q1 | Low — even if a client did this, the resulting `-32601` is a defensible "not modern" classification, not silent data corruption; worst case is a confusing error message for a malformed client, not a spec violation on codegraph-go's part |
| A2 | `query.ResolveCodegraphDir`'s cost is negligible per MCP request (a few bounded stat calls, not a recursive scan) | Q2, Pattern 2 | Low — `serveServerPaths` already calls this once per process at startup and every tool handler already does comparable path-resolution work (`confineToRepoRoot`) per call; re-confirm with a benchmark only if profiling later shows otherwise |

**If this table is empty:** N/A — two low-risk assumptions remain, both flagged for cheap re-confirmation during execution rather than blocking planning.

## Open Questions

1. **Exact scenario shape/count for the SPEC-02 `-32602`/`-32022` wire-oracle proof (Q7).**
   - What we know: the six shapes tested empirically in Code Examples above cover the full decision surface (`usesNewProtocol` false/true × valid/invalid `clientCapabilities`/`clientInfo` × supported/unsupported version).
   - What's unclear: how many of these six the plan chooses to freeze as scenarios vs. cover with a smaller representative subset (a `-32602` case + a `-32022` case is probably sufficient; the phase's own calibration note argues against over-coverage).
   - Recommendation: freeze exactly two new `_meta`-failure scenarios (one `-32602`, one `-32022`, using the "2099-01-01"-shaped literal for the latter per Pitfall 1) plus one Modern `server/discover` success scenario; that's the minimum that proves SPEC-02/01/03/04/07/08's discover half without redundant coverage.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing | ✓ | matches `go.mod`'s declared version, confirmed by this session's own `go build` | — |
| `modelcontextprotocol/go-sdk@v1.7.0` (module cache) | Every finding in this document | ✓ | already resolved in `$GOMODCACHE`, read directly this session | — |
| Existing wire oracle (`test/wireoracle`) | Q6/Q7's mechanism | ✓ | unmodified, 23 scenarios, confirmed runnable this session (`task check:transcript-freeze` executed) | — |
| `task check:transcript-freeze` / `tools/transcriptfreeze` | Q6 | ✓ | run twice this session against real and simulated diffs | — |

No missing dependencies.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `test/wireoracle`'s raw-stdio harness (unchanged from Phase 2) |
| Config file | `Taskfile.yml` (`test:wireoracle`, `check:transcript-freeze`, `test`) |
| Quick run command | `task test:wireoracle` |
| Full suite command | `task test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SPEC-01/03/08 | Discover/tool-result shape already inherited | integration (wire) | `go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch` | ✅ (existing transcripts already exercise the classic-`initialize` half; the Modern-`discover` half needs a new scenario) |
| SPEC-02 | `-32602` malformed/missing `_meta`; `-32022` unsupported version | integration (wire) + hand-authored anchor | new scenarios + a new `Anchor` in `anchors.go` mirroring `codeMethodNotFound`/`codeInvalidParams`'s existing pattern (`[VERIFIED: test/wireoracle/anchors.go:73-92]`) | ❌ Wave 0 gap |
| SPEC-04 | `cacheScope:"private"` on discover | integration (wire) | new discover-success scenario | ❌ Wave 0 gap |
| SPEC-05 | Tools appear mid-session | integration (wire) | new "index appears mid-session" scenario, either driving real `codegraph init` in a fixture or simulating the transition (CONTEXT.md's own open discretion item — unresolved by this research, plan-level call) | ❌ Wave 0 gap |
| SPEC-06 | Legacy client completes session AND calls a tool | integration (wire) | `handshake-explore` already proves this at `2025-11-25` (`[VERIFIED: test/wireoracle/scenarios.go:93, 270-304]` — `handshakeExploreProtocolVersion = "2025-11-25"`, and the scenario's `Requests` include `initialize`, `tools/list`, and a `tools/call` for `codegraph_explore`); the four other legacy-era scenarios (`2025-06-18`, `2025-03-26`, `2024-11-05`, plus omitted/unsupported) prove negotiation only, no `tools/call` | ⚠️ partial — see recommendation below |
| SPEC-07 | `instructions` on discover | integration (wire) | folds into the same new discover-success scenario as SPEC-04 | ❌ Wave 0 gap |

### SPEC-06 recommendation (LOW confidence — a judgment call, not a proven gap)

`handshake-explore` already proves a `2025-11-25` Legacy session completes AND successfully calls `codegraph_explore` (`[VERIFIED: test/wireoracle/scenarios.go:270-304, read this session]` — its `Requests` slice literally contains an `initialize` at id 1, a `tools/list` at id 2, and a `tools/call` for `"codegraph_explore"` at id 3). Since tool-call dispatch in go-sdk has no per-era branching once `initialize` succeeds (`callTool`/`toolForErr` never consult `req.ProtocolVersion()`, confirmed by reading both this session — the only per-era gating is the `usesNewProtocol` response-annotation logic covered under SPEC-03/08, which is orthogonal to whether the call itself succeeds), one successful call at the *newest* Legacy era is reasonable evidence the mechanism works across all Legacy eras. The four other legacy-era scenarios prove negotiation-only. **Recommendation:** add a `tools/call` step to the existing `legacy-2024-11-05` scenario (the *oldest* era, maximal distance from Modern) rather than adding a new scenario — cheaper than a new scenario and closes the widest remaining gap. This is a plan-level decision, not a research finding; flagged LOW confidence because "one era plus the newest is enough" is a judgment call CONTEXT.md left open, not something this session could prove is sufficient.

### Sampling Rate

- **Per task commit:** `go build ./... && go vet ./...` plus `go test ./internal/mcp/...`
- **Per wave merge:** `task test:wireoracle`
- **Phase gate:** `task test` full suite green, plus a human-read transcript diff (D-01/D-03's mechanism, unchanged), plus `task check:transcript-freeze` run against the **actual PR base** (see Q6 — do not assume EXEMPTED)

### Wire Oracle Gate: how to prove it non-vacuous for THIS phase

Per the repo's standing rule, re-run at least one of Phase 1's confirmed mutations (e.g. Mutation 1, stray stdout line) against the Phase-3-complete binary and confirm `TestFrozenTranscriptsMatch` still goes RED, then green after revert. Additionally, the new `-32022`/`-32602` anchors must themselves be demonstrated RED: temporarily mutate the offered `_meta.protocolVersion` literal in the new scenario's request construction (e.g. swap `"2099-01-01"` for a supported value) and confirm the anchor assertion fails before reverting — mirroring exactly how `codeMethodNotFound`/`codeInvalidParams`'s existing anchors were proven (D-02's convention).

### Wave 0 Gaps

- [ ] A new `Anchor` in `test/wireoracle/anchors.go` for `-32022` (hand-authored `const codeUnsupportedProtocolVersion = -32022`, never `mcp.CodeUnsupportedProtocolVersion` — see Anti-Patterns)
- [ ] New scenario(s): Modern `server/discover` success (covers SPEC-01/03/04/07/08's discover half in one capture); `_meta` malformed/missing (`-32602`); `_meta` unsupported-but-well-formed version (`-32022`); index-appears-mid-session (SPEC-05)
- [ ] `ExpectedScenarioCount` (`test/wireoracle/scenarios.go:265`, currently `23`) bumped to match, in the same commit as the new scenarios per its own doc comment's rule

*(No framework installation needed — `test/wireoracle` and `go test` already exist and are proven.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | stdio subprocess, no auth surface (unchanged from Phase 2's determination) |
| V3 Session Management | Marginal | `_meta`-based per-request "sessionless" dispatch (SEP-2575) is new to this phase — a Modern client can call tools without ever completing `initialize` (empirically confirmed, case 9/10 above); this is spec-sanctioned statelessness, not a session-fixation risk, since every handler still re-resolves and re-confines its own path per call (`confineToRepoRoot`, unchanged) |
| V4 Access Control | Yes (unchanged) | `confineToRepoRoot` (`internal/mcp/tools.go:33-47`) — this phase does not touch it and must not "helpfully improve" it in the same change, per the standing project rule carried from Phase 2's research |
| V5 Input Validation | Yes | `_meta` validation is now go-sdk's own responsibility (Q1) — an improvement over anything codegraph-go could hand-roll, since it runs before any handler code executes |
| V6 Cryptography | No | not applicable |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A client bypassing `initialize` entirely via sessionless Modern `_meta` calls to probe tool availability without establishing identity | Information Disclosure (minor) | Not a new risk relative to today: `tools/list`/`tools/call` results carry no secrets, and `confineToRepoRoot` already gates every path argument regardless of session state |
| A malformed `_meta` object crafted to trigger a panic in codegraph-go's own code | Denial of Service | Not applicable — `_meta` parsing happens entirely inside go-sdk before any codegraph-go code runs; codegraph-go's middleware only ever sees already-validated `_meta` (Q1) |

## Sources

### Primary (HIGH confidence — real source read and/or real code built and run this session)

- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go` — `discover` (:884-910), `ServerOptions` (:69-172), `NewServer` (:182-233), `AddTool`/`RemoveTools`/`changeAndNotify`/`notifySessions`/`shouldSendListChangedNotification` (:273-826), `capabilities()` (:615-663), `ss.handle` (:1852-1934), `AddReceivingMiddleware`/`serverMethodInfos` (:1760-1801)
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/shared.go` — `validateRequestMeta`/`validatedMeta` (:528-576), `ServerRequest.ProtocolVersion()` (:640-658)
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go` — `DiscoverResult` (:1138-1150), `Cacheable`/`setDefaultCacheableValues` (:1158-1196)
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/features.go` — `featureSet[T]` (:23-27, no internal lock)
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/internal/jsonrpc2/conn.go:694-695` — generic `ErrMethodNotFound` message rewrite
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/internal/jsonrpc2/wire.go:64-94` — `WireError`/`WireError.Is` (code-only comparison)
- **This session's own probe runs**, built via `go build -o bin/codegraph ./cmd/codegraph` from this branch's HEAD (Phase 2 complete) and driven with raw JSON-RPC over real stdio — 11 distinct request shapes captured verbatim in Code Examples
- `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/cli/serve.go` — read in full this session
- `internal/mcp/archtest/protocol_version_test.go` — read this session (VRFY-02 guard, its documented false-positive surface, and the absence of a code-level exception mechanism)
- `test/wireoracle/scenarios.go`, `test/wireoracle/anchors.go` — read in full this session
- `tools/transcriptfreeze/classify.go` — read in full this session; `task check:transcript-freeze` run twice — once against the real branch state (EXEMPTED, exit 0), once against a git-simulated post-Phase-2-merge base with a synthetic Phase-3-shaped diff (Violation, exit 1)

### Secondary (from prior-phase artifacts, cross-checked against primary sources above)

- `.planning/phases/03-2026-07-28-spec-compliance/03-CONTEXT.md` — D-01 through D-07, canonical refs, the probe recipe this session's Code Examples build on
- `.planning/phases/02-sdk-migration-official-go-sdk-on-the-existing-surface/02-RESEARCH.md` — Q1 (PROVEN-ABSENT pattern this session's Q1 extends to `_meta` validation), Q3 (initialize result shape, cross-checked and still accurate), Q6 (`AddReceivingMiddleware` mechanics)
- `.planning/phases/02-sdk-migration-official-go-sdk-on-the-existing-surface/02-05-SUMMARY.md` — the nine named causes and the reviewed-diff mechanism this phase's own diff review will reuse
- `.planning/research/PITFALLS.md` — Pitfall 8 (`server/discover` probe-spawn cost, informative background, not directly load-bearing for this phase's scope)

### Tertiary

None — every claim traces to primary source or this session's own empirical, built-and-run verification.

## Metadata

**Confidence breakdown:**
- SPEC-02 `_meta` validation surface: HIGH — full source trace plus 11 empirically-captured request/response pairs
- SPEC-04/07 discover fields: HIGH — source read confirms identical mechanism to the already-shipped D-09 `tools/list` fix
- SPEC-05 concurrency safety: HIGH — every mutator/reader's locking read directly from source
- SPEC-05 mechanism choice (per-request check vs. watcher): HIGH on the "watcher doesn't apply" half (documented in the codebase itself); MEDIUM on "per-request check is the right replacement" (a reasonable, evidenced recommendation, but the plan retains discretion here)
- SPEC-06 coverage sufficiency: LOW — a judgment call about how much Legacy-era tool-call coverage is "enough," explicitly flagged as such
- Q6 (freeze-gate sequencing): HIGH — reproduced with a real, git-verified simulation, not inferred

**Research date:** 2026-08-06
**Valid until:** pinned to `go-sdk@v1.7.0` specifically — any version bump invalidates the `validateRequestMeta`/`WireError.Is`/`changeAndNotify` findings above and would need re-verification. Not calendar-based.
