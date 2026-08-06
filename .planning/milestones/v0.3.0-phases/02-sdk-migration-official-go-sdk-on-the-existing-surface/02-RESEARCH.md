# Phase 2: SDK Migration — official go-sdk on the existing surface - Research

**Researched:** 2026-08-05
**Domain:** Go MCP SDK internals (`modelcontextprotocol/go-sdk@v1.7.0`), wire-protocol behavior of a stdio server
**Confidence:** HIGH — every claim about SDK behavior in this document is `[VERIFIED]` against the real `v1.7.0` module source in `$GOMODCACHE`, and the highest-risk claims (Q1, Q3, Q4, plus three bonus findings) were additionally confirmed by **building and running the actual SDK code** end-to-end over raw stdio, byte-for-byte, using the same driver methodology as `test/wireoracle`.

## Summary

This research answers Q1–Q9 by reading `github.com/modelcontextprotocol/go-sdk@v1.7.0`'s real source in the module cache and, for the highest-risk questions, by compiling a throwaway probe server against the real SDK and driving raw JSON-RPC over its stdio — the exact wire-oracle methodology, run against the actual dependency Phase 2 is migrating to.

**The single most important finding overturns a CONTEXT.md lean:** D-05 predicted `legacy-unsupported-2026-07-28.golden` would move from `"2025-11-25"` to `"2026-07-28"` after the swap, based on Context7 doc evidence ("go-sdk recognizes it"). **This is wrong.** `negotiatedVersion()` — the sole, unexported, deterministic function that decides the classic `initialize` handshake's answer — caps every offer at `< "2026-07-28"` (a strict string comparison) specifically because the classic `initialize` method is itself deprecated as of `2026-07-28`. An offer of `"2026-07-28"` therefore falls through to the same fallback as an unrecognized version and negotiates `"2025-11-25"` — **identical to today's mark3labs output**. This transcript should be expected to move **zero bytes** in its `protocolVersion` field (though the surrounding envelope still changes — see below). D-06's directive ("must be proven, not assumed... source enumeration, not Context7") was correct to insist on this before planning.

The second-most-important finding is new and was not anticipated by any prior-phase document: **every `tools/list` response gains `"ttlMs":0,"cacheScope":"public"` unconditionally**, the moment the SDK swap lands — not gated behind protocol era, not something Phase 3 can defer landing. This is Phase 3's SPEC-04 territory by requirement ID, but it **arrives as a side effect of Phase 2's dependency swap regardless of what Phase 2 intends to implement**. The value `"public"` is also wrong per SPEC-04's eventual target (`"private"`) — Phase 2 does not need to fix this, but the diff review must name it as an expected, inherited, not-yet-corrected divergence, or a reviewer will misdiagnose it as scope creep.

Third: with **zero tools registered** (`hasIndex=false`, MCP-03's no-index path), go-sdk's `capabilities()` omits the `"tools"` key from `initialize`'s capabilities entirely, whereas mark3labs unconditionally advertises `"tools":{"listChanged":true}` via `server.WithToolCapabilities(true)` regardless of registration count. **This is a real behavioral divergence the migration must actively counter** (via `ServerOptions.Capabilities`), not merely document, because it is the one surviving artifact of Pitfall 4's "no tools = ambiguous between not-indexed and protocol-mismatch" risk this milestone is designed to close.

Fourth: struct-tag schema inference (D-07's chosen path) adds `"additionalProperties":false` to **every** tool's `inputSchema` (not present in any tool today), reorders `properties` keys to **Go struct field declaration order** instead of mark3labs' alphabetical order, and reorders the schema's own top-level keys (`type, properties, required, additionalProperties` vs. today's `properties, required, type`). None of these were flagged in CONTEXT.md.

**Primary recommendation:** Proceed with D-07's `AddTool`-generic + struct-tag approach; it is confirmed to route Go errors into the same `IsError:true`/`Content` shape mark3labs used (Q5), confirmed to auto-validate required arguments before the handler runs (closing Phase 1's known coverage gap by construction, with a different error message), and confirmed to have no enum-tag mechanism to lose (Q7 — D-08 was right). Explicitly configure `ServerOptions.Capabilities` to preserve the always-on tools capability and treat the `ttlMs`/`cacheScope`/`additionalProperties`/key-order/annotation-order deltas as expected, explained lines in the D-01 diff review — they are unavoidable consequences of adopting this SDK, not implementation choices Phase 2 can tune away except where noted.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| MCP protocol negotiation (initialize handshake) | API/Backend (in-process library) | — | Wholly owned by the `mcp` package's `negotiatedVersion()`; codegraph-go has no server-side control point (Q1) |
| Tool schema generation | API/Backend | — | `AddTool` + `jsonschema-go` reflection at server-construction time, no client/runtime component |
| Tool execution / query engine | API/Backend | — | Unchanged by this phase — `internal/query.Engine`, delegated to identically pre- and post-swap |
| Session diagnostics (VRFY-03 line) | API/Backend | — | `AddReceivingMiddleware`, in-process, stderr-only — no new tier introduced |
| Wire verification (wire oracle) | Database/Storage (test fixture) + API/Backend (driver) | — | Frozen `.golden` files on disk compared against real subprocess stdio; no client tier involved |

This phase touches exactly one tier: the in-process Go server library. There is no browser, SSR, CDN, or persistent-storage surface in scope — consistent with the milestone's stdio-only framing.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SDK-01 | `internal/mcp` runs on go-sdk v1.7.0, semantically unchanged, every transcript diff explained | Q1–Q4 plus three bonus findings enumerate every wire-visible change the swap causes, with root cause and exact bytes, so every line of every diff has a name before the plan writes a single line of migration code |
| SDK-03 | mark3labs removed from `go.mod`, dependency closure re-audited | Q9 — real `go get`/`go mod tidy` dry run against a scratch copy of the actual `go.mod`/`go.sum`, net module delta enumerated |
| SDK-04 | Error-to-wire mapping audited and asserted | Q5 — `toolForErr`'s wrapper read and cited line-by-line; confirmed empirically that a returned Go `error` produces `IsError:true` with `Content` text, matching mark3labs' `NewToolResultError` shape |
| SDK-05 | Tool input schemas audited for lost constraints | Q7 — `jsonschema-go`'s struct-tag mechanism read in full; confirmed no enum-tag syntax exists in this version (D-08 recalibration was correct), confirmed `int`→`"integer"` (not `"number"`), confirmed `required` derived from absence of `omitempty`/`omitzero` |

</phase_requirements>

## Package Legitimacy Audit

Go's module ecosystem has no seam in `gsd-tools query package-legitimacy check` (npm/PyPI/crates only). Verification below was performed directly against the Go module proxy — `go get`/`go mod download` succeeded and pulled real, buildable source for every package named — and cross-referenced against each package's own `go.mod`.

| Package | Registry | Source Repo | Verdict | Disposition |
|---------|----------|--------------|---------|-------------|
| `github.com/modelcontextprotocol/go-sdk@v1.7.0` | Go module proxy (proxy.golang.org) | `github.com/modelcontextprotocol/go-sdk` — official MCP org, already the maintainer's 2026-08-03 pre-decision in REQUIREMENTS.md | OK | Approved (pre-decided; this research downloaded and read the real v1.7.0 source, confirming it exists, builds, and matches the described API) |
| `github.com/google/jsonschema-go` (v0.4.2→v0.4.3, transitive) | Go module proxy | `github.com/google/jsonschema-go` — official Google org | OK | Approved — **already present** in codegraph-go's `go.mod` today as an indirect dependency of mark3labs itself (`go mod why -m` confirms `internal/mcp → mark3labs/mcp-go/mcp → google/jsonschema-go`), so this is a version bump, not a new supply-chain surface |
| `github.com/segmentio/encoding` + `github.com/segmentio/asm` (transitive, new) | Go module proxy | `github.com/segmentio/encoding` — Segment/Twilio org, widely used fast JSON codec | OK | Approved — pulled in transitively by go-sdk; net-new to codegraph-go's closure |
| `golang.org/x/oauth2`, `golang.org/x/time` (transitive, new) | Go module proxy | `golang.org/x/*` — official Go extended stdlib | OK | Approved — official Go team packages; net-new to codegraph-go's closure but lowest-possible supply-chain risk tier |
| `github.com/yosida95/uritemplate/v3` | Go module proxy | already present | OK | No change — required by **both** mark3labs and go-sdk at the same version (`v3.0.2`); survives the swap unchanged |

**Packages removed due to SLOP verdict:** none.
**Packages flagged as suspicious (SUS):** none — every new transitive dependency is an established, widely-used, actively maintained package from a recognizable org (Segment, the Go team, Google).

**Note on `[ASSUMED]` vs `[VERIFIED]`:** every package-existence claim above is `[VERIFIED: Go module proxy download + go.mod read]`, not `[ASSUMED]` — the packages were actually downloaded to `$GOMODCACHE` and their own `go.mod` files read in this session (see Q9 below for the exact commands and output).

## Q1 — Server-side control over the negotiated protocol version: PROVEN-ABSENT

**Verdict: PROVEN-ABSENT**, by exhaustive enumeration, matching Phase 1's own bar ("enumerate every `func With…`... before concluding absence"), plus **empirically confirmed by running the real code**.

### Every field of `mcp.ServerOptions`

`[VERIFIED: $GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:69-172]` — full struct read in this session:

```
Instructions                 string
Logger                       *slog.Logger
InitializedHandler           func(context.Context, *InitializedRequest)
PageSize                     int
RootsListChangedHandler      func(context.Context, *RootsListChangedRequest)   // deprecated (SEP-2577)
ProgressNotificationHandler  func(context.Context, *ProgressNotificationServerRequest)
CompletionHandler            func(context.Context, *CompleteRequest) (*CompleteResult, error)
KeepAlive                    time.Duration
KeepAliveFailureThreshold    int
SubscribeHandler             func(context.Context, *SubscribeRequest) error
UnsubscribeHandler           func(context.Context, *UnsubscribeRequest) error
Capabilities                 *ServerCapabilities
HasPrompts                   bool   // deprecated
HasResources                 bool   // deprecated
HasTools                     bool   // deprecated
SchemaCache                  *SchemaCache
GetSessionID                 func() string
```

**17 fields. None named `ProtocolVersion`, none accept a version string, none influence version negotiation.** `Capabilities` configures the *capabilities object*, not the negotiated version.

### Every exported func/method whose name mentions Version, Protocol, or Negotiate

`[VERIFIED: rg -n "^func .*\b(Version|Protocol|Negotiat)" over mcp/*.go, non-test]` — exactly one hit:

```go
func (r *ServerRequest[P]) ProtocolVersion() string   // shared.go:646
```

This is a **read-only getter** used inside a request handler. Its doc comment (`shared.go:640-645`) and body (`:646-658`) show it is a *fallback chain reader*, not a setter: for `>= 2026-07-28` requests it reads per-request `_meta`; for older requests it falls back to `r.Session.InitializeParams().ProtocolVersion` — the **client's originally-requested** value, not the negotiated one. There is no companion setter anywhere in the package.

### `negotiatedVersion()` — full body and every call site

`[VERIFIED: $GOMODCACHE/.../mcp/shared.go:44-79]`:

```go
const (
    latestProtocolVersion   = protocolVersion20260728
    protocolVersion20260728 = "2026-07-28"
    protocolVersion20251125 = "2025-11-25"
    protocolVersion20250618 = "2025-06-18"
    protocolVersion20250326 = "2025-03-26"
    protocolVersion20241105 = "2024-11-05"
)

var supportedProtocolVersions = []string{
    protocolVersion20260728, protocolVersion20251125, protocolVersion20250618,
    protocolVersion20250326, protocolVersion20241105,
}

// negotiatedVersion returns the effective protocol version to use, given a
// client version.
func negotiatedVersion(clientVersion string) string {
    // In general, prefer to use the clientVersion, but if we don't support the
    // client's version, use the latest version.
    //
    // Cap the supported versions at the legacy protocolVersion20251125, as this
    // method is used by the initialize method which is deprecated in
    // version protocolVersion20260728.
    if slices.Contains(supportedProtocolVersions, clientVersion) && clientVersion < protocolVersion20260728 {
        return clientVersion
    }
    return protocolVersion20251125
}
```

Every identifier in this function (`latestProtocolVersion`, `protocolVersion20260728`, `supportedProtocolVersions`, `negotiatedVersion` itself) is **unexported (lowercase)** — none of these could ever be referenced from `internal/mcp` even by accident, which is directly relevant to `internal/mcp/archtest`'s VRFY-02 guard (see Common Pitfalls below).

Call sites `[VERIFIED: rg -n negotiatedVersion over mcp/*.go]`:
- `server.go:1985` — `ss.initialize()`, the classic `initialize` RPC handler (the only call site the wire oracle's 23 scenarios exercise)
- `client.go:365-366` — client-side, irrelevant to a server migration
- `transport.go:510` — used inside the `server/discover` response path (out of Phase 2's scope per REQUIREMENTS.md)

### Full body of `ss.initialize()`

`[VERIFIED: $GOMODCACHE/.../mcp/server.go:1965-1990]`:

```go
func (ss *ServerSession) initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
    if params == nil {
        return nil, fmt.Errorf("%w: \"params\" must be be provided", jsonrpc2.ErrInvalidParams)
    }
    var wasInit bool
    ss.updateState(func(state *ServerSessionState) {
        wasInit = state.InitializeParams != nil
        if !wasInit { state.InitializeParams = params }
    })
    if wasInit {
        ss.server.opts.Logger.Error("duplicate initialize request")
        return nil, fmt.Errorf("duplicate %q received", methodInitialize)
    }
    s := ss.server
    return &InitializeResult{
        ProtocolVersion: negotiatedVersion(params.ProtocolVersion),
        Capabilities:    s.capabilities(),
        Instructions:    s.opts.Instructions,
        ServerInfo:      s.impl,
    }, nil
}
```

No `ss.server.opts` field is consulted for `ProtocolVersion`. There is no hook, callback, or interceptor invoked before this struct literal is returned.

### `ServerSessionState` — does the session store the negotiated value anywhere retrievable?

`[VERIFIED: $GOMODCACHE/.../mcp/session.go:17-29]`:

```go
type ServerSessionState struct {
    InitializeParams   *InitializeParams   `json:"initializeParams"`
    InitializedParams  *InitializedParams  `json:"initializedParams"`
    LogLevel           LoggingLevel        `json:"logLevel"`
}
```

**No field stores the negotiated version.** Only the client's *requested* params are kept. The negotiated value is computed once, inline, inside `ss.initialize()`, and never written back to session state — confirming that even `AddReceivingMiddleware` (Q6's mechanism) must read it off the *result* of the call, not off any session accessor.

### Empirical confirmation (built and ran the real code)

A throwaway probe server built against the real `go-sdk@v1.7.0` module, driven with raw JSON-RPC over its actual stdio pipes (methodology identical to `test/wireoracle`):

```
$ ./driver "2026-07-28"
offered=2026-07-28 -> {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-11-25","serverInfo":{"name":"codegraph","version":"0.1.0"}}}
```

`protocolVersion` in the response is **`"2025-11-25"`**, not `"2026-07-28"` — directly refuting the CONTEXT.md D-05 lean. See Q4 for the full empirical matrix.

**One caveat, documented rather than left implicit:** `AddReceivingMiddleware` (Q6) technically *can* mutate the `*InitializeResult` in place after `next()` returns, before it is written to the wire — this is a generic capability of any middleware wrapping any result, not a version-specific API. D-04 already decided against pinning/injecting, so this is noted only so a future reader does not mistake "PROVEN-ABSENT as a supported API" for "PROVEN-IMPOSSIBLE by the type system." It is absent as a *blessed* mechanism; it is technically reachable via a generic escape hatch nobody is proposing to use.

## Q2 — `ToolAnnotations`: does `omitempty` drop the false hints?

**No — the opposite of PITFALLS Pitfall 7's prediction.** `[VERIFIED: $GOMODCACHE/.../mcp/protocol.go:1967-2013]`, struct verbatim:

```go
type ToolAnnotations struct {
    DestructiveHint *bool  `json:"destructiveHint,omitempty"`
    IdempotentHint  bool   `json:"idempotentHint"`        // NO omitempty
    OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
    ReadOnlyHint    bool   `json:"readOnlyHint"`           // NO omitempty
    Title           string `json:"title,omitempty"`
}

// MarshalJSON implements [json.Marshaler] for ToolAnnotations.
//
// To restore the previous behavior where false-valued ReadOnlyHint and
// IdempotentHint were omitted, set MCPGODEBUG=hintomitempty=1.
func (t ToolAnnotations) MarshalJSON() ([]byte, error) {
    if hintomitempty == "1" {
        type compat struct {
            DestructiveHint *bool  `json:"destructiveHint,omitempty"`
            IdempotentHint  bool   `json:"idempotentHint,omitempty"`
            OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
            ReadOnlyHint    bool   `json:"readOnlyHint,omitempty"`
            Title           string `json:"title,omitempty"`
        }
        return json.Marshal(compat(t))
    }
    type nomethod ToolAnnotations
    return json.Marshal(nomethod(t))   // <-- default path: always includes readOnlyHint/idempotentHint
}
```

The SDK's own authors evidently shipped the `omitempty`-drops-false bug at some point, then **fixed it going forward and made the old (dropping) behavior an explicit opt-out** via `MCPGODEBUG=hintomitempty=1`, not the default. **By default (env var unset, which is what CI and a normal `serve --mcp` invocation will have), `readOnlyHint:false` and `idempotentHint:false` survive on the wire exactly as they do today.** This directly reverses PITFALLS Pitfall 7's prediction for `readOnlyHint`/`idempotentHint`; that finding is now stale and should not be repeated as a live risk in the plan.

**What does change (confirmed empirically, see below):** the four keys' *order* on the wire, because the SDK marshals `ToolAnnotations` in Go struct-field-declaration order (`destructiveHint, idempotentHint, openWorldHint, readOnlyHint` — coincidentally alphabetical), while mark3labs' frozen order today is `readOnlyHint, destructiveHint, idempotentHint, openWorldHint`. Empirical proof (built the real server, real annotations, real wire bytes):

```
"annotations":{"destructiveHint":true,"idempotentHint":false,"openWorldHint":true,"readOnlyHint":false}
```

vs. today's frozen (`testdata/wireoracle/transcripts/handshake-explore.golden:2`):

```
"annotations":{"readOnlyHint":false,"destructiveHint":true,"idempotentHint":false,"openWorldHint":true}
```

**Same four key/value pairs, different order.** This is a real, expected, D-01-class semantic-equivalence diff on every one of the 8 tools' `annotations` block in every `tools/list`-bearing transcript (`handshake-explore`, `toolslist-default`, `toolslist-allowlist`, `toolslist-repeat`).

## Q3 — `initialize` response: `InitializeResult`, `Implementation`, `ServerCapabilities`, `Server.capabilities()`

### `InitializeResult` verbatim

`[VERIFIED: $GOMODCACHE/.../mcp/protocol.go:1088-1105]`:

```go
type InitializeResult struct {
    Meta            `json:"_meta,omitempty"`
    Capabilities    *ServerCapabilities `json:"capabilities"`
    Instructions    string              `json:"instructions,omitempty"`
    ProtocolVersion string              `json:"protocolVersion"`
    ServerInfo      *Implementation     `json:"serverInfo"`
}
```

**Field declaration order — and therefore marshaled key order — is `capabilities, instructions(if set), protocolVersion, serverInfo`.** Today's frozen baseline emits `protocolVersion` **first**, then `capabilities`, then `serverInfo` (`handshake-explore.golden:1`: `{"protocolVersion":"...","capabilities":{...},"serverInfo":{...}}`). **This is a real, previously-unflagged key-order divergence on the very first line of every single one of the 23 frozen transcripts.**

### `Implementation` verbatim

`[VERIFIED: $GOMODCACHE/.../mcp/protocol.go:2214-2228]`:

```go
type Implementation struct {
    Name        string `json:"name"`
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Version     string `json:"version"`
    WebsiteURL  string `json:"websiteUrl,omitempty"`
    Icons       []Icon `json:"icons,omitempty"`
}
```

Marshaled order (with only `Name`/`Version` set, as today): `name, version` — **matches today's frozen order exactly.** No diff here.

### `ServerCapabilities` verbatim

`[VERIFIED: $GOMODCACHE/.../mcp/protocol.go:2265-2294]`:

```go
type ServerCapabilities struct {
    Experimental map[string]any            `json:"experimental,omitempty"`
    Extensions   map[string]any            `json:"extensions,omitempty"`
    Completions  *CompletionCapabilities   `json:"completions,omitempty"`
    Logging      *LoggingCapabilities      `json:"logging,omitempty"`
    Prompts      *PromptCapabilities       `json:"prompts,omitempty"`
    Resources    *ResourceCapabilities     `json:"resources,omitempty"`
    Tools        *ToolCapabilities         `json:"tools,omitempty"`
}
```

### `Server.capabilities()` — full body

`[VERIFIED: $GOMODCACHE/.../mcp/server.go:615-663]`:

```go
func (s *Server) capabilities() *ServerCapabilities {
    var caps *ServerCapabilities
    if s.opts.Capabilities != nil {
        caps = s.opts.Capabilities.clone()
    } else {
        // SDK defaults: only logging capability.
        caps = &ServerCapabilities{Logging: &LoggingCapabilities{}}
    }
    if s.opts.HasTools || s.tools.len() > 0 {
        if caps.Tools == nil {
            caps.Tools = &ToolCapabilities{ListChanged: true}
        }
    }
    // ... (Prompts/Resources/Completions, all irrelevant — codegraph-go uses none)
    return caps
}
```

**Two confirmed, material divergences, both empirically reproduced by running the real code:**

1. **`"logging":{}` is ADDED to `capabilities` unless `ServerOptions.Capabilities` is explicitly set to a non-nil value.** Today's frozen baseline is `"capabilities":{"tools":{"listChanged":true}}` — no `logging` key. If `BuildServer`'s migration does not explicitly set `Capabilities`, every `initialize` response gains a `"logging":{}` entry that mark3labs never emitted and codegraph-go never uses (it declares no `LoggingCapabilities` handler). Empirically confirmed:
   ```
   {"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-11-25", ...}
   ```
2. **With zero tools registered (`hasIndex=false`, MCP-03's no-index path), the `"tools"` key is OMITTED entirely.** `s.opts.HasTools` defaults false, and `s.tools.len() == 0` when no `AddTool` call ever fires — so `caps.Tools` stays `nil` and is dropped by `omitempty`. Empirically confirmed by running a probe server with zero tools registered:
   ```
   {"capabilities":{"logging":{}},"protocolVersion":"2025-11-25","serverInfo":{"name":"codegraph","version":"0.1.0"}}
   ```
   **No `"tools"` key at all.** Today's frozen `toolslist-no-index.golden`'s `initialize` line reads `"capabilities":{"tools":{"listChanged":true}}` — mark3labs advertises the tools capability **unconditionally** via `server.WithToolCapabilities(true)` (`internal/mcp/server.go:178`), independent of whether any tool is actually registered. **This is a real behavioral regression risk, not merely a cosmetic diff**: it changes what a client can infer about server capability shape at zero tools, in exactly the ambiguous "no tools = not indexed, or protocol mismatch?" territory PITFALLS Pitfall 4 and VRFY-03 exist to disambiguate. **Action for the plan:** `BuildServer`'s migrated `ServerOptions.Capabilities` should be set explicitly (e.g. `&ServerCapabilities{Tools: &ToolCapabilities{ListChanged: true}}`, with `Logging` left nil to also close divergence 1) so the `"tools"` key survives regardless of registration count, preserving today's contract. This is a one-line fix, not a redesign — flagging it as a **Common Pitfall** below since it is easy to miss.

### Fields ADDED / REMOVED — summary answer to Q3's literal question

| Field | Status | Cause |
|---|---|---|
| `capabilities.logging` | **ADDED** unless suppressed | SDK default when `ServerOptions.Capabilities == nil` |
| `capabilities.tools` (at zero tools) | **REMOVED** unless forced | `capabilities()`'s `HasTools \|\| tools.len()>0` gate — no unconditional escape hatch like mark3labs' `WithToolCapabilities(true)` |
| Key order (`capabilities` before `protocolVersion`) | **REORDERED** | `InitializeResult`'s own field declaration order |
| Everything else (`serverInfo.name/version`, `protocolVersion` value itself for in-range clients) | **UNCHANGED** | confirmed byte-identical in the empirical driver runs above |

## Q4 — Empty/omitted `protocolVersion` in the initialize request

**Empirically confirmed**, driving the real SDK with a request that omits `protocolVersion` entirely:

```
$ ./driver ""
offered= -> {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-11-25","serverInfo":{"name":"codegraph","version":"0.1.0"}}}
```

`negotiatedVersion("")`: `slices.Contains(supportedProtocolVersions, "")` is `false` (the empty string is never one of the five listed versions), so the function falls through to `return protocolVersion20251125` — **`"2025-11-25"`**.

Today's frozen `legacy-omitted-version.golden` records mark3labs' behavior for the same request as negotiating **`"2025-03-26"`** (its older backwards-compat default for an omitted version). **This is a real, confirmed transcript divergence** — `"2025-03-26"` → `"2025-11-25"` — the third predicted candidate from CONTEXT.md, now proven rather than assumed.

**Combined verdict for the two era-boundary scenarios**, both proven, not assumed:

| Scenario | Pre-migration (frozen) | Post-migration (proven) | Moves? |
|---|---|---|---|
| `legacy-unsupported-2026-07-28` (offer `2026-07-28`) | `"2025-11-25"` | `"2025-11-25"` | **No** — same value, different reason (mark3labs didn't recognize it; go-sdk recognizes it but caps the classic-`initialize` path below it) |
| `legacy-omitted-version` (offer omitted) | `"2025-03-26"` | `"2025-11-25"` | **Yes** |

This directly overturns CONTEXT.md D-05's framing ("the single most predictable diff in the phase" — the `2026-07-28` one) and confirms D-06's insistence on source-level proof: **the omitted-version scenario, not the unsupported-version scenario, is the one that actually moves.** `legacy-unsupported-2026-07-28.golden`'s rename (per the `<specifics>` note in CONTEXT.md — "it no longer measures an unsupported version, it measures the newest supported one") is **still semantically appropriate to do** (go-sdk *does* recognize `2026-07-28` as a version, even though the classic-`initialize` code path caps it), but the plan should not expect its `protocolVersion` byte value to change.

## Q5 — Error-to-wire mapping (SDK-04)

**Confirmed: a plain Go `error` returned from a typed `ToolHandlerFor` produces exactly the same wire shape mark3labs used — `IsError:true` with `Content` text — not a protocol error.**

`[VERIFIED: $GOMODCACHE/.../mcp/tool.go:17-30]` — `ToolHandlerFor`'s own doc comment states the contract:

> "An error result is treated as a tool error, rather than a protocol error, and is therefore packed into `CallToolResult.Content`, with `IsError` set."

`[VERIFIED: $GOMODCACHE/.../mcp/server.go:377-392]` — `toolForErr`'s wrapper, the code that actually implements this for every `AddTool[In,Out]`-registered handler:

```go
res, out, err := h(ctx, req, in)
if err != nil {
    // Check if this is already a structured JSON-RPC error
    if wireErr, ok := err.(*jsonrpc.Error); ok {
        return nil, wireErr
    }
    // For regular errors, embed them in the tool result as per MCP spec
    var errRes CallToolResult
    errRes.SetError(err)
    return &errRes, nil
}
```

`[VERIFIED: $GOMODCACHE/.../mcp/protocol.go:347-353]` — `CallToolResult.SetError`:

```go
func (r *CallToolResult) SetError(err error) {
    if len(r.Content) == 0 || seterroroverwrite == "1" {
        r.Content = []Content{&TextContent{Text: err.Error()}}
    }
    r.IsError = true
    r.err = err
}
```

This produces `{"content":[{"type":"text","text":"<err.Error()>"}],"isError":true}` — **the identical shape mark3labs' `mcp.NewToolResultError(err.Error())` produced.**

**Confirming the codebase claim** (CONTEXT.md canonical_refs / Phase 1 code_context): `[VERIFIED: internal/mcp/tools.go — read in full this session]` — every one of the 8 tool handlers' error paths reads `return mcp.NewToolResultError(err.Error()), nil` (e.g. `tools.go:107`, `:113`, `:119`, `:239`, `:245`). **Zero occurrences of a bare `return nil, err`.** This confirms the Phase 1 finding independently: SDK-04's audit surface really is this small for the code that exists today.

**The go-sdk equivalent of `NewToolResultError`:** there is no single named helper function — the pattern is `var r mcp.CallToolResult; r.SetError(err); return &r, someZeroOutputValue, nil` under `ToolHandlerFor`'s three-return-value shape, or more simply, **if the input struct's zero-value `Out` and a `nil` `*CallToolResult` are acceptable, just `return nil, zeroOut, err`** — `toolForErr` (above) converts a plain returned `error` into the same `SetError` shape automatically. This means the D-07 migration can *simplify* every handler from today's four-line `if err != nil { return mcp.NewToolResultError(err.Error()), nil }` down to `if err != nil { return nil, out, err }`, since the wrapper now does the conversion — a genuine code-simplification opportunity, not just a mechanical port.

**Bonus finding — closes part of Phase 1's known coverage gap by construction.** Empirically driving a `tools/call` for `codegraph_explore` with the required `query` argument omitted, against the real SDK using `AddTool`'s typed-struct schema validation:

```
call-missing-query -> {"jsonrpc":"2.0","id":4,"result":{"content":[{"type":"text","text":"validating \"arguments\": validating root: required: missing properties: [\"query\"]"}],"isError":true}}
```

This is `applySchema`'s automatic input validation (`tool.go:75-131`, called from `toolForErr` **before** the handler runs at all — `[VERIFIED: mcp/server.go:353-365]`), not codegraph-go's own `req.RequireString("query")` logic. It already returns a **tool-visible `isError:true` result**, matching the shape MUTATION-PROOF.md's Mutation 3 showed mark3labs' bare-`error` path would have produced as a **protocol** error if codegraph-go's handlers had used `return nil, err` instead of `NewToolResultError`. Under go-sdk + `AddTool`, this class of failure is **automatically** a tool-visible error by construction, closing the "no scenario exercises a handler's own required-argument validation failure" gap MUTATION-PROOF.md flagged — with a different error message text (`"validating \"arguments\": ..."` vs. today's handler-authored `"required argument \"query\" not found"`), which is exactly the kind of explained, expected divergence D-01 exists to accommodate.

## Q6 — Session-line replacement hook (VRFY-03)

**No single ServerOptions callback exposes both the requested and negotiated protocol version simultaneously — `Server.AddReceivingMiddleware` is the mechanism that does, by wrapping the whole request lifecycle rather than being a single-purpose hook.**

Two considered options, evaluated against real source:

1. **`ServerOptions.InitializedHandler`** (`server.go:74-75`, fired from `ss.initialized()` at `server.go:1403-1431`) — fires on `notifications/initialized`, **after** the `initialize` response was already sent. Its `req *InitializedRequest` exposes `req.Session.InitializeParams()` (the client's *requested* version + `ClientInfo`), but **the negotiated value is nowhere on the session** (confirmed by reading `ServerSessionState`'s full field list in Q1 — no negotiated-version field exists). To get the negotiated value here, the handler would have to **re-derive it by duplicating `negotiatedVersion()`'s unexported logic** in `internal/mcp` — a real risk of drift the next time go-sdk's negotiation rule changes (e.g. a hypothetical `2027-xx-xx` release).

2. **`Server.AddReceivingMiddleware(...Middleware)`** (`server.go:1770-1774`, `Middleware = func(MethodHandler) MethodHandler`, `MethodHandler = func(ctx, method string, req Request) (Result, error)`) — wraps **the entire receiving pipeline**, confirmed by reading the actual dispatch chain: `[VERIFIED: mcp/shared.go:186-205]` `handleReceive` calls `mh := session.receivingMethodHandler(); res, err := mh(ctx, jreq.Method, req)` — `mh` **is** `s.receivingMethodHandler_`, the exact field `AddReceivingMiddleware` wraps. A middleware that checks `method == "initialize"`, calls `next(ctx, method, req)`, then inspects the **returned** `res` (type-asserted to `*InitializeResult`) sees the fully-computed `res.ProtocolVersion` — the actual negotiated value — because by the time `next()` returns, `ss.initialize()` has already run to completion. The same middleware invocation can read `req.GetParams().(*InitializeParams)` for the requested version and `ClientInfo`.

**This is the correct mechanism, and it satisfies CONTEXT.md's stated requirement in a single hook** (contrary to the framing "if no single hook sees all of that, describe what combination does" — one hook suffices here):

```go
// sketch, not final code — illustrates the shape only
s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
        res, err := next(ctx, method, req)
        if method == "initialize" && err == nil {
            if initRes, ok := res.(*mcp.InitializeResult); ok {
                if params, ok := req.GetParams().(*mcp.InitializeParams); ok {
                    // params.ProtocolVersion = requested, initRes.ProtocolVersion = negotiated,
                    // params.ClientInfo.Name/Version = client identity, toolCount = closure var, unchanged shape
                }
            }
        }
        return res, err
    }
})
```

**Tool count** — no SDK dependency at all: `toolCount` is derived at server-construction time in `BuildServer`'s existing registration loop, exactly as it is today (`internal/mcp/server.go:176,227,231`). This part of the mechanism carries over completely unchanged; only the "how do I see requested+negotiated" half needed a new answer.

**Registration point:** `AddReceivingMiddleware` is a `*Server` method, called once after `mcp.NewServer(...)` and before `Run` — structurally analogous to today's `server.WithHooks(hooks)` `ServerOption`, i.e. still a construction-time, once-per-process wiring, not a per-call cost.

## Q7 — Struct-tag schema inference (SDK-05)

Answered by reading `github.com/google/jsonschema-go@v0.4.2` (the exact version pinned by go-sdk v1.7.0's own `go.mod`) and confirming with a real `jsonschema.For[T]()` call.

### Does a `jsonschema:"..."` tag become the property's description?

**Yes, and it is the *entire* tag value, verbatim — not a key=value mini-language.** `[VERIFIED: $GOMODCACHE/github.com/google/jsonschema-go@v0.4.2/jsonschema/infer.go:330-338]`:

```go
if tag, ok := field.Tag.Lookup("jsonschema"); ok {
    if tag == "" {
        return nil, fmt.Errorf("empty jsonschema tag on struct field %s.%s", t, field.Name)
    }
    if disallowedPrefixRegexp.MatchString(tag) {
        return nil, fmt.Errorf("tag must not begin with 'WORD=': %q", tag)
    }
    fs.Description = tag
}
```

The `disallowedPrefixRegexp` check (`infer.go:399` doc comment: "Disallow jsonschema tag values beginning `WORD=`, for future expansion") confirms there is **no enum syntax, no key=value syntax of any kind today** — the whole string is the description, and a future SDK version might add `WORD=`-prefixed keywords, but v0.4.2 (the version actually pinned) does not. **D-08's recalibration is correct and now source-confirmed**: there is no enum-tag mechanism to lose, because none exists to use.

### `int` → `"integer"`, not `"number"` — confirmed

`[VERIFIED: $GOMODCACHE/.../jsonschema-go@v0.4.2/jsonschema/infer.go:52-53,158-196]`:

```
//   - Signed and unsigned integer types have schema type "integer".
//   - Floating point types have schema type "number".
case reflect.Int, reflect.Int64: s.Type = "integer"
...
case reflect.Float32, reflect.Float64: s.Type = "number"
```

Every one of today's numeric tool parameters (`max_files`, `line`, `limit`, `depth`) is declared as a Go `int` in the current handlers (`GetInt`/`WithNumber`). Under D-07's typed-struct approach, these become `"type":"integer"` on the wire, whereas mark3labs' `mcp.WithNumber` emits `"type":"number"` today — confirmed by the frozen transcript (`handshake-explore.golden:2`: `"max_files":{"description":"...","type":"number"}`). **This is a real, deliberate, per-D-08 divergence across every numeric parameter of every tool** — worth naming exhaustively in the diff review rather than once, since it recurs on `codegraph_explore.max_files`, `codegraph_node.line`, `codegraph_search.limit`, `codegraph_callers.limit`, `codegraph_callees.limit`, `codegraph_impact.depth`, `codegraph_files.depth` — 7 occurrences.

### `required` determination

`[VERIFIED: $GOMODCACHE/.../jsonschema-go@v0.4.2/jsonschema/infer.go:343-345]`:

```go
if !info.settings["omitempty"] && !info.settings["omitzero"] {
    s.Required = append(s.Required, info.name)
}
```

Confirmed: required = absence of `omitempty`/`omitzero` on the field's `json` tag, exactly as D-07 assumed.

### Can a tag express an enum? — No (empirically confirmed, not just doc-read)

No syntax exists in v0.4.2 (see above). This was independently confirmed by building and running `jsonschema.For[T]()` against representative structs (below) — no enum keyword appeared in any output regardless of tag content.

### Bonus, undocumented-in-CONTEXT.md finding: `additionalProperties:false` is added to every inferred schema, and property order changes from alphabetical to Go struct field order

Empirically confirmed by actually calling `jsonschema.For[T]()` from the real `v0.4.2` module against a struct shaped like `codegraph_explore`'s and `codegraph_status`'s arguments:

```go
type ExploreArgs struct {
    Query    string `json:"query" jsonschema:"Natural-language or symbol/file query"`
    Path     string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
    MaxFiles int    `json:"max_files,omitempty" jsonschema:"Cap on distinct files returned (default 5)"`
}
```

Output (`json.Marshal` of the real `*jsonschema.Schema`, one field per line reformatted for readability — the actual bytes are on one line):

```json
{
  "type": "object",
  "properties": {
    "query":     {"type": "string", "description": "Natural-language or symbol/file query"},
    "path":      {"type": "string", "description": "Repo path (default: server cwd)"},
    "max_files": {"type": "integer", "description": "Cap on distinct files returned (default 5)"}
  },
  "required": ["query"],
  "additionalProperties": false
}
```

Compare to today's frozen `codegraph_explore` schema (`handshake-explore.golden:2`):

```json
{
  "properties": {
    "max_files": {"description": "...", "type": "number"},
    "path":      {"description": "...", "type": "string"},
    "query":     {"description": "...", "type": "string"}
  },
  "required": ["query"],
  "type": "object"
}
```

**Three confirmed, independent divergences per tool's `inputSchema`:**

1. **`"additionalProperties":false` is a new key, present in the go-sdk output, absent from every one of today's 8 tools' frozen schemas.** This is not optional or suppressible via `ForOptions` (`[VERIFIED: jsonschema-go@v0.4.2/jsonschema/infer.go:25-40]` — `ForOptions` has exactly two fields, `IgnoreInvalidTypes` and `TypeSchemas`, neither of which controls this). It is a structural default of struct-derived schemas. **Expect this on all 8 tools, every `tools/list`-bearing transcript.**
2. **`properties` key order is Go struct field declaration order** (`query, path, max_files`), **not alphabetical** — traced to `Schema.MarshalJSON`'s custom `orderedProperties` wrapper (`[VERIFIED: jsonschema-go@v0.4.2/jsonschema/schema.go:276-294,311-357]`), which writes keys from `PropertyOrder` (populated from struct field order during inference, `infer.go:341`) before falling back to any remaining unordered keys. Today's mark3labs order is alphabetical (`file, line, path, symbol` for `codegraph_node`; `max_files, path, query` for `codegraph_explore`). **The plan can choose the Go struct's field declaration order deliberately** — declaring fields in the same order the parameter matters most for reading, or in alphabetical order to minimize the diff — either is a legitimate, reviewable choice under D-01.
3. **The schema object's own top-level key order changes**: go-sdk emits `type, properties, required(if any), additionalProperties` (traced to `Schema`'s custom `MarshalJSON`, `[VERIFIED: jsonschema-go@v0.4.2/jsonschema/schema.go:236-309]` — `Type` is hoisted into an anonymous wrapper struct declared before the embedded `*schemaWithoutMethods`, so it marshals first, followed by `Properties`, then the rest of `schemaWithoutMethods`'s own field order which places `Required` before `AdditionalProperties`), vs. today's alphabetical `properties, required, type`.

**And when a tool has zero required fields** (`codegraph_status`, `codegraph_node`'s two optional-ish variants — actually all companions except `search`/`callers`/`callees`/`impact` which require `query`/`symbol`), `required` is **omitted entirely** (`Required []string \`json:"required,omitempty"\`` — confirmed empty-slice-collapses-under-omitempty via the same empirical run: `codegraph_status`'s schema had no `required` key at all), vs. today's explicit `"required":[]` (confirmed present in `toolslist-allowlist.golden:2`'s `codegraph_status`/`codegraph_node` entries).

## Q8 — Transport/serve API replacing `server.ServeStdio(s)`

`[VERIFIED: $GOMODCACHE/.../mcp/server.go:1285-1312]`:

```go
func (s *Server) Run(ctx context.Context, t Transport) error
```

with `t = &mcp.StdioTransport{}` (`[VERIFIED: mcp/transport.go:112-117]` — `type StdioTransport struct{}`, `func (*StdioTransport) Connect(context.Context) (Connection, error)`).

Server construction: `[VERIFIED: mcp/server.go:182-217]` `func NewServer(impl *Implementation, options *ServerOptions) *Server`.

Mapping onto `internal/mcp`'s existing `Server` interface (`ServeStdio() error`, `internal/mcp/server.go:80-82`):

```go
type goSDKServer struct{ inner *mcp.Server }
func (g *goSDKServer) ServeStdio() error {
    return g.inner.Run(context.Background(), &mcp.StdioTransport{})
}
```

`Run` needs a `context.Context` that today's `Server.ServeStdio() error` signature does not carry — `context.Background()` is the direct equivalent of mark3labs' `server.ServeStdio(s)` (which similarly has no external cancellation hook today; `internal/cli/serve.go:258` calls `s.ServeStdio()` with no context threading currently). This is a mechanical, in-bounds substitution — no interface change to `internal/mcp.Server` is required, confirming the "planner's call" framing in CONTEXT.md's Claude's Discretion section resolves cleanly either way.

## Q9 — `go.mod` impact (SDK-03)

Verified via a real dry-run: copied the actual `go.mod`/`go.sum` into a scratch directory, edited `require` to drop `mark3labs/mcp-go` and add `go-sdk@v1.7.0`, and separately read both dependencies' own `go.mod` files directly from `$GOMODCACHE` (the most authoritative source — the module's own declared requirements, not a summary).

**Modules go-sdk v1.7.0 requires (its own `go.mod`):**

```
require (
    github.com/golang-jwt/jwt/v5 v5.3.1
    github.com/google/go-cmp v0.7.0
    github.com/google/jsonschema-go v0.4.3
    github.com/segmentio/encoding v0.5.4
    github.com/yosida95/uritemplate/v3 v3.0.2
    golang.org/x/oauth2 v0.35.0
    golang.org/x/time v0.15.0
    golang.org/x/tools v0.42.0
)
require ( // indirect
    github.com/segmentio/asm v1.1.3
    golang.org/x/sync v0.20.0
    golang.org/x/sys v0.41.0
)
```

**Modules mark3labs v0.56.0 requires (its own `go.mod`, for comparison):**

```
require (
    github.com/google/jsonschema-go v0.4.2
    github.com/google/uuid v1.6.0
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
    github.com/spf13/cast v1.7.1
    github.com/stretchr/testify v1.11.1
    github.com/yosida95/uritemplate/v3 v3.0.2
)
```

**Net delta, computed against the real closure (a scratch `go get`+`go mod tidy` was run; results below reflect the pruned/actually-reachable set, not the raw declared set — go-sdk's own `golang-jwt/jwt/v5`, `golang.org/x/tools`, and `google/go-cmp` requirements turn out to be unreachable from the `mcp` package alone and would not survive `go mod tidy` in codegraph-go's own tree):**

| Module | Change | Notes |
|---|---|---|
| `github.com/mark3labs/mcp-go` | **REMOVED** | the point of SDK-03 |
| `github.com/modelcontextprotocol/go-sdk` | **ADDED** (`v1.7.0`) | the new backend |
| `github.com/google/jsonschema-go` | **VERSION BUMP** `v0.4.2 → v0.4.3` | already present today as an indirect dep of mark3labs itself (`go mod why -m github.com/google/jsonschema-go` shows `internal/mcp → mark3labs/mcp-go/mcp → google/jsonschema-go`) — not a new supply-chain surface, just a newer pin |
| `github.com/yosida95/uritemplate/v3` | **UNCHANGED** (`v3.0.2`) | required by both SDKs at the identical version |
| `github.com/segmentio/encoding` + `github.com/segmentio/asm` | **ADDED** | net-new transitive dependency of go-sdk, reachable |
| `golang.org/x/oauth2` | **ADDED** | net-new, go-sdk's HTTP-auth code (unused by codegraph-go's stdio-only surface, but present in the closure regardless — Go modules doesn't do conditional requires) |
| `golang.org/x/time` | **ADDED** | net-new, official Go extended package |
| `github.com/google/uuid` | **REMOVED** (assuming `go mod tidy` confirms no other reference) | today reachable **only** via `internal/mcp → mark3labs/mcp-go/server → google/uuid` (`[VERIFIED: go mod why -m github.com/google/uuid]`, path confirmed this session) |
| `github.com/santhosh-tekuri/jsonschema/v6` | **REMOVED** (assuming `go mod tidy` confirms) | today reachable **only** via `internal/mcp → mark3labs/mcp-go/server → santhosh-tekuri/jsonschema/v6` (`[VERIFIED: go mod why -m]`, confirmed this session) — go-sdk uses `google/jsonschema-go` instead, a different package |
| `github.com/spf13/cast` | **REMOVED** (assuming `go mod tidy` confirms) | today reachable **only** via `internal/mcp → mark3labs/mcp-go/mcp → spf13/cast` (`[VERIFIED: go mod why -m]`, confirmed this session) — not used elsewhere in codegraph-go's tree (no `viper`, no other `cast` consumer found) |
| `github.com/stretchr/testify` | likely **REMOVED** from `go.sum`, unconfirmed for `go.mod` | present in `go.sum` today as mark3labs' own test-only dependency; low actual build impact either way |
| `golang.org/x/tools`, `golang.org/x/sync`, `golang.org/x/sys`, `github.com/google/go-cmp` | **UNCHANGED** | already present in codegraph-go's `go.mod` at versions higher than go-sdk requests; Go's MVS keeps the higher pin, no bump needed |

**What `govulncheck`/SBOM should watch for:** the two clearly net-new, non-stdlib-adjacent surfaces are `segmentio/encoding`/`segmentio/asm` (a fast JSON codec, Segment/Twilio org) and `golang-jwt/jwt` — the latter turned out to be **pruned away** in the actual reachable-package analysis (it's part of go-sdk's HTTP transport/OAuth code, not the `mcp` package's stdio surface codegraph-go imports), so it should **not** actually land in codegraph-go's `go.mod` after a real `go mod tidy` — worth confirming this prediction against the real `go mod tidy` output during execution, since a scratch-directory dry run without the actual importing source can differ subtly from the real tree. `govulncheck`'s call-graph-aware reachability analysis (already the house standard per `.claude/CLAUDE.md`) is the correct tool to re-confirm this post-migration, exactly as SDK-03 requires.

## Architecture Patterns

### System Architecture Diagram

```
                    stdio (JSON-RPC, newline-delimited)
                            │
                            ▼
              internal/cli/serve.go (newServeCmd)
                            │
                            │  mcp.NewStdioServer(...)
                            ▼
              internal/mcp.Server interface  ◄── unchanged by this phase
                    (ServeStdio() error)
                            │
        ┌───────────────────┴───────────────────┐
        │ pre-migration                          │ post-migration
        ▼                                        ▼
mark3labsServer{*server.MCPServer}      goSDKServer{*mcp.Server}
        │                                        │
        │ server.ServeStdio(s)                   │ s.Run(ctx, &mcp.StdioTransport{})
        ▼                                        ▼
  mark3labs/mcp-go dispatch                go-sdk dispatch:
  (Hooks.AddAfterInitialize)               (AddReceivingMiddleware, wraps
        │                                   the WHOLE method-dispatch chain,
        │                                   sees pre-negotiation request AND
        │                                   post-negotiation result)
        ▼                                        │
  initialize / tools/list / tools/call    same three RPC methods, same
        │                                  handler bodies (D-07: typed args
        ▼                                  via AddTool, same query.Engine
  internal/query.Engine (UNCHANGED)        delegation, UNCHANGED)
        │
        ▼
  internal/query.Render*Markdown (UNCHANGED)
```

**What changes:** the box between `internal/mcp.Server` and `internal/query.Engine` — the SDK dispatch internals, the schema-generation mechanism, and the session-line hook mechanism. **What does not change:** the seam boundary itself (`internal/mcp.Server`, already closed by SDK-02 in Phase 1), and everything below `internal/mcp` (the query engine, the renderers, `internal/cli`).

### Recommended Project Structure

No new packages or directories are required. This phase modifies:
```
internal/mcp/
├── server.go              # BuildServer: swap server.NewMCPServer → mcp.NewServer,
│                           # explicit ServerOptions.Capabilities, AddReceivingMiddleware
│                           # replaces Hooks.AddAfterInitialize
├── tools.go                # mcp.NewTool/WithString/WithNumber builder chain →
│                           # mcp.AddTool[In,Out] + typed structs (D-07)
├── protocol_version.go     # doc-comment-only update: "reads from" is now
│                           # accurate to describe (still not literally injectable — Q1)
└── archtest/
    └── protocol_version_test.go   # re-run unmodified; verify it still passes
                                     # post-swap (see Common Pitfalls)
```

### Pattern 1: `AddTool` generic + typed struct arguments

**What:** Replace each tool's `mcp.NewTool(name, mcp.WithString(...), ...)` + `server.ToolHandlerFunc` pair with one Go struct (fields tagged `json`+`jsonschema`) and `mcp.AddTool[In, Out](s, tool, handler)` where `handler` has signature `func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)`.

**When to use:** Every one of the 8 existing tools — this is D-07's locked decision.

**Example (verified against real SDK build, not hypothetical):**
```go
// Source: this session's own probe, built and run against go-sdk v1.7.0
type ExploreArgs struct {
    Query    string `json:"query" jsonschema:"Natural-language or symbol/file query"`
    Path     string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
    MaxFiles int    `json:"max_files,omitempty" jsonschema:"Cap on distinct files returned (default 5)"`
}

mcp.AddTool(s, &mcp.Tool{
    Name:        "codegraph_explore",
    Description: "Explore relevant symbols: verbatim source, call paths, blast radius",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint: false, DestructiveHint: &destructiveTrue,
        IdempotentHint: false, OpenWorldHint: &openWorldTrue,
    },
}, func(ctx context.Context, req *mcp.CallToolRequest, in ExploreArgs) (*mcp.CallToolResult, any, error) {
    // in.Query, in.Path, in.MaxFiles — no more req.RequireString/GetInt extraction
    out, err := eng.Explore(in.Query, in.MaxFiles)
    if err != nil {
        return nil, nil, err   // toolForErr converts this to IsError:true automatically (Q5)
    }
    return mcp.NewToolResultText(query.WorktreeNotice(...) + out), nil, nil // or similar
})
```

### Pattern 2: `AddReceivingMiddleware` for the always-on session line

**What:** Replace `server.Hooks{}.AddAfterInitialize(...)` with `Server.AddReceivingMiddleware(...)`, filtering on `method == "initialize"` and reading the *result* (not just the request) — see Q6's full sketch above.

**When to use:** VRFY-03's session line, the one place this migration requires genuine redesign rather than mechanical translation, exactly as CONTEXT.md flagged.

### Anti-Patterns to Avoid

- **Relying on `ServerOptions.Capabilities == nil`'s default:** silently adds `"logging":{}` and silently drops `"tools"` at zero registered tools. Set `Capabilities` explicitly (Q3).
- **Re-deriving `negotiatedVersion()`'s logic by hand** inside `internal/mcp` to feed the session line from `InitializedHandler`: this duplicates unexported SDK logic and will silently drift the next time the SDK's negotiation rule changes. Use `AddReceivingMiddleware` instead (Q6) — it reads the SDK's own actual computed result, never re-implements the computation.
- **Assuming `ForOptions` can suppress `additionalProperties:false`:** it cannot (Q7) — this is a structural default of struct-derived schemas in `jsonschema-go@v0.4.2`, not a toggle.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Deriving the negotiated protocol version for the session line | A hand-ported copy of `negotiatedVersion()`'s comparison logic | `AddReceivingMiddleware` reading the real `*InitializeResult.ProtocolVersion` post-`next()` | The SDK's own negotiation rule is unexported and may change between releases; reading its actual output can never drift, a hand-copy always can (Q6) |
| Tool argument schema generation | Hand-authored `*jsonschema.Schema` values per tool | `mcp.AddTool[In,Out]` struct-tag reflection (D-07, already locked) | D-01 removed the byte-control requirement that was the only argument for hand-authoring; reflection is confirmed to preserve every constraint codegraph-go's tools currently express (D-08, confirmed no enums exist to lose) |
| Error-to-wire conversion for tool failures | Manual `CallToolResult{IsError:true, Content:...}` construction in every handler | `return nil, out, err` under `ToolHandlerFor` — `toolForErr` converts automatically (Q5) | The SDK's own wrapper already does exactly what `mcp.NewToolResultError` did; duplicating it per-handler is pure boilerplate the migration can delete |

**Key insight:** this migration is unusually favorable for deleting code, not just porting it — three of the four things codegraph-go's handlers do by hand today (error-to-result conversion, required-argument validation, and — if desired — the always-on tools capability) have first-class SDK support that either already matches or improves on today's behavior, once `ServerOptions.Capabilities` is set explicitly for the one case (zero-tools) where the SDK's default actually regresses today's contract.

## Common Pitfalls

### Pitfall 1: Silent capability-shape regression at zero registered tools

**What goes wrong:** With no `.codegraph/` index (`hasIndex=false`), `initialize`'s `capabilities` object loses its `"tools"` key entirely under go-sdk's default behavior, whereas mark3labs always advertised it.
**Why it happens:** `Server.capabilities()`'s tools-capability inference is gated on `HasTools || tools.len()>0`; there is no equivalent of mark3labs' unconditional `WithToolCapabilities(true)`.
**How to avoid:** Explicitly set `ServerOptions.Capabilities: &ServerCapabilities{Tools: &ToolCapabilities{ListChanged: true}}` in `BuildServer`, unconditionally (not gated on `hasIndex`).
**Warning signs:** `toolslist-no-index`'s post-migration `initialize` line is missing `"tools"` — this is the scenario to check first after the swap.

### Pitfall 2: Mistaking `ttlMs`/`cacheScope` for Phase 3 scope creep (or missing it as an expected diff)

**What goes wrong:** Every `tools/list` response gains `"ttlMs":0,"cacheScope":"public"` unconditionally, the moment go-sdk is adopted — not because Phase 2 implements anything from SPEC-04, but because `res.setDefaultCacheableValues()` is called unconditionally inside the SDK's own `listTools` handler (`[VERIFIED: mcp/server.go:944]`).
**Why it happens:** go-sdk's `ListToolsResult` embeds `Cacheable` (`CacheableResult` interface machinery, `protocol.go:1158-1197`) as a structural part of the type, not an opt-in feature.
**How to avoid:** Name this explicitly in the D-01 diff review as an inherited, unavoidable side effect of the dependency swap, not a Phase-2 implementation choice and not something Phase 2 needs to "fix" — SPEC-04 (Phase 3) is what will later correct `cacheScope` from `"public"` to `"private"`.
**Warning signs:** A reviewer asks "why is Phase 2 already doing SPEC-04's work" — the answer is it isn't; the SDK does this regardless of what Phase 2's Go code says.

### Pitfall 3: Assuming `legacy-unsupported-2026-07-28.golden`'s `protocolVersion` value changes

**What goes wrong:** CONTEXT.md D-05 predicted this transcript's `protocolVersion` moves to `"2026-07-28"`. It does not — `negotiatedVersion()` caps every offer strictly below `"2026-07-28"` because the classic `initialize` method is itself deprecated at that revision (Q1, Q4).
**How to avoid:** Expect this transcript's `protocolVersion` value to stay `"2025-11-25"` post-migration (same as pre-migration) — only the scenario's *rename* (per CONTEXT's `<specifics>` note) is still appropriate, not a value change. The surrounding envelope (capability key order, `logging` addition) still changes per the other findings above.
**Warning signs:** A plan task that says "verify `legacy-unsupported-2026-07-28` now negotiates `2026-07-28`" will fail against the real SDK — this exact framing must not appear in the plan.

### Pitfall 4: `internal/mcp/archtest`'s VRFY-02 guard has two future false-positive candidates (not a Phase 2 blocker, but worth recording)

**What goes wrong:** go-sdk exports `MetaKeyProtocolVersion = "io.modelcontextprotocol/protocolVersion"` (`protocol.go:2363`) and `CodeUnsupportedProtocolVersion = -32022` (`shared.go:394`) — both are external `*types.Const` whose names match the guard's `(?i)protocol.?version` pattern. Neither is a protocol-version *value* pin (Q1's actual concern); they're a `_meta` key name and a JSON-RPC error code. If Phase 3 (SPEC-02/SPEC-08) ever references either constant directly, `TestNoExternalProtocolVersionConstantReferences` will flag it as a false-positive VRFY-02 violation.
**Why it happens:** the guard's name-heuristic (`internal/mcp/archtest/protocol_version_test.go:82`) is deliberately broad — it distinguishes external `*types.Const` from other kinds of references, but not "a version-value constant" from "a same-named-but-different-purpose constant."
**How to avoid:** the guard's own doc comment already documents its escape hatch ("record a reviewed exception in this file's doc comment") — Phase 2 does not need `_meta` handling and will not trigger this, but Phase 3's planner should know it's coming.
**Confirmed NOT a risk for Phase 2:** the actual protocol-version *value* constants (`latestProtocolVersion`, `protocolVersion20260728`, etc.) are all unexported — Go's own visibility rules make it structurally impossible for `internal/mcp` to ever reference them, so `internal/mcp/protocol_version.go`'s "asserted pin, not injection point" framing (its own doc comment) remains accurate after the swap, unchanged.

### Pitfall 5: `MCPGODEBUG=hintomitempty=1` silently reverting Q2's finding

**What goes wrong:** if this environment variable is ever set in CI or a developer's shell (e.g. copied from an unrelated go-sdk troubleshooting guide), `ToolAnnotations` reverts to dropping `false`-valued hints, re-introducing exactly the divergence PITFALLS Pitfall 7 predicted and this research disproved as the default.
**How to avoid:** confirm this env var is unset in the CI environment that runs `task test:wireoracle` post-migration; do not set it "to match old behavior" — the whole point of D-01's semantic-equivalence bar is that the *default* (always-present) behavior is fine to accept as a documented diff.

## Code Examples

### Verified: what a real `tools/list` response looks like post-migration (before any codegraph-specific fixes)

```json
// Source: this session's own probe server, built against go-sdk v1.7.0, driven over real stdio
{"jsonrpc":"2.0","id":2,"result":{"ttlMs":0,"cacheScope":"public","tools":[{"annotations":{"destructiveHint":true,"idempotentHint":false,"openWorldHint":true,"readOnlyHint":false},"description":"Explore relevant symbols: verbatim source, call paths, blast radius","inputSchema":{"type":"object","properties":{"query":{"type":"string","description":"..."},"path":{"type":"string","description":"..."},"max_files":{"type":"integer","description":"..."}},"required":["query"],"additionalProperties":false},"name":"codegraph_explore"}]}}
```

Every difference from today's frozen `handshake-explore.golden:2` in this output is now named and explained somewhere above: `ttlMs`/`cacheScope` (Pitfall 2), `annotations` key order (Q2), `inputSchema` key order + `additionalProperties` + property order + `type:integer` (Q7).

### Verified: what a real `initialize` response looks like at zero tools (before the Pitfall-1 fix)

```json
{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{}},"protocolVersion":"2025-11-25","serverInfo":{"name":"codegraph","version":"0.1.0"}}}
```

No `"tools"` key — apply Pitfall 1's fix (`ServerOptions.Capabilities` set explicitly) before comparing against `toolslist-no-index.golden`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `server.Hooks{}.AddAfterInitialize(...)` | `Server.AddReceivingMiddleware(...)` filtering `method=="initialize"` | This migration | Middleware sees both the request AND the eventually-computed result, closing the gap mark3labs' after-hook and go-sdk's session-state (no negotiated-version field) both leave open |
| `mcp.WithString`/`WithNumber`/builder-chain schemas | `mcp.AddTool[In,Out]` + struct tags | This migration (D-07) | Type-safe handler args, automatic required-arg validation before the handler runs, at the cost of `additionalProperties:false` and reordered keys |
| `return mcp.NewToolResultError(err.Error()), nil` per handler | `return nil, out, err` (SDK auto-converts) | This migration | 4-line boilerplate per handler collapses to the natural Go error-return idiom |
| `mcp.LATEST_PROTOCOL_VERSION` (exported SDK constant, Phase 1's Pitfall 2) | No exported protocol-version-value constant exists in go-sdk at all | This migration | The failure mode Pitfall 2 warned about (a routine dependency bump silently moving the declared version) becomes structurally impossible post-migration, not just guarded against — there is nothing exported to accidentally reference |

**Deprecated/outdated:**
- PITFALLS Pitfall 7's `omitempty`-drops-false-hints prediction: was accurate for an *earlier* go-sdk release; the pinned `v1.7.0` has already reverted to always-including the hints by default (Q2). Do not carry this prediction into the plan unmodified.
- CONTEXT.md D-05's "the transcript moves to `2026-07-28`" framing: superseded by Q1/Q4's proof. The rename is still appropriate; the value-change is not.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `go mod tidy` against the real codegraph-go tree (not the scratch dry-run) will actually drop `google/uuid`, `santhosh-tekuri/jsonschema/v6`, `spf13/cast`, and not pull in `golang-jwt/jwt/v5`/`golang.org/x/tools` as new direct requirements | Q9 | Low — `go mod tidy`'s own output during Phase 2 execution is authoritative and will be run for real; this research's scratch dry-run used a module with no actual importing source, so its module-graph resolution is directionally correct but must be re-confirmed against the real build |
| A2 | No other part of codegraph-go's tree (outside `internal/mcp`) imports `google/uuid`, `santhosh-tekuri/jsonschema/v6`, or `spf13/cast` | Q9 | Low-medium — `go mod why -m` was run for all three and each showed exactly one path, entirely through mark3labs; a second, unseen import path would keep the module in `go.mod` post-swap without being wrong, just a smaller-than-predicted removal |
| A3 | The plan's chosen Go struct field order for each tool's input struct will determine the final `properties` key order on the wire, and this is an acceptable, reviewable D-01 divergence rather than something the plan must fight to prevent | Q7 | Low — this is a design choice, not a technical risk; flagged so the planner makes it deliberately rather than by accident |

**If this table is empty:** N/A — three low-risk assumptions remain, all pointing at "re-confirm against the real build during execution," not at anything uncertain about SDK behavior itself (which was proven, not assumed, throughout this document).

## Open Questions

1. **Does `AddReceivingMiddleware`'s registration point interact with `mcp.NewServer`'s construction-vs-`Run` lifecycle in a way that matters for `BuildServer`'s existing functional-options pattern (`Option`, `WithSessionLog`)?**
   - What we know: `AddReceivingMiddleware` is a `*Server` method, callable any time before the first request is dispatched; `BuildServer` already builds the server via `opts ...Option` before returning it, so registering middleware inside that same function (after `mcp.NewServer(...)`, before `return`) is structurally identical to today's `server.WithHooks(hooks)` `ServerOption` placement.
   - What's unclear: whether `AddReceivingMiddleware` needs to run before or after `AddTool` calls — untested in this session's probes (all probes registered tools first, middleware was not combined with the session-line sketch in the same run).
   - Recommendation: low risk, mechanical to verify in the first migration plan task — write the probe, run it, confirm ordering doesn't matter (middleware wraps the whole per-request dispatch, not registration-time behavior, so it almost certainly doesn't).

2. **Does the migrated `BuildServer` need `SchemaCache` (`ServerOptions.SchemaCache`)?**
   - What we know: it exists to avoid repeated reflection "for stateless server deployments where a new Server is created for each request" (`server.go:154-158`) — codegraph-go's stdio server constructs exactly one `*Server` per process lifetime, so this does not apply.
   - What's unclear: nothing — this is a non-issue, recorded only so the planner doesn't spend time investigating it.
   - Recommendation: do not set `SchemaCache`; the default (no caching) is correct for a single-construction, long-lived stdio server.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the migration | ✓ | go1.26.5 darwin/arm64 (matches `go.mod`'s `go 1.26.5`) | — |
| `modelcontextprotocol/go-sdk@v1.7.0` (Go module proxy) | The migration itself | ✓ | v1.7.0, downloaded and built successfully this session | — |
| `google/jsonschema-go@v0.4.3` (transitive) | Schema inference | ✓ | resolved automatically via `go get` | — |
| Existing wire oracle (`test/wireoracle`) | SDK-01's diff-review mechanism | ✓ | unmodified, 23 scenarios, ~17.7s wall-clock per Phase 1's own measurement | — |

No missing dependencies. Everything this phase needs is already resolvable from the standard Go module proxy with no network/registry surprises — confirmed by actually downloading and building against the real dependency in this research session.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package + `test/wireoracle`'s custom raw-stdio harness |
| Config file | `Taskfile.yml` (`test:wireoracle`, `test:unit`, `test:integration`, `test:golden`, `test:daemon`, `test:race` targets) |
| Quick run command | `task test:wireoracle` (~17.7s, all 23 scenarios) |
| Full suite command | `task test` (chains `test:unit`, `test:golden`, `test:integration`, `test:wireoracle`, `test:daemon`, `test:race`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SDK-01 | Every transcript diff explained, harness code unmodified | integration (wire) | `go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch -v` | ✅ (`test/wireoracle/oracle_test.go`) |
| SDK-01 | `TestScenarioCountIsExact`/`TestTranscriptSetMatchesScenarioSet` still hold post-swap | integration | `go test ./test/wireoracle/... -run 'TestScenarioCountIsExact|TestTranscriptSetMatchesScenarioSet'` | ✅ |
| SDK-02 (carried) | `internal/cli` still names no MCP SDK package | archtest | `go test ./internal/cli/archtest/... -run TestInternalCLIImportsNoMCPSDK` | ✅ (already forward-declares `modelcontextprotocol/go-sdk` in `forbiddenMCPSDKPrefixes` — no code change needed) |
| VRFY-02 (carried) | No SDK-owned protocol-version constant referenced anywhere | archtest | `go test ./internal/mcp/archtest/... -run TestNoExternalProtocolVersionConstantReferences` | ✅ — will pass unmodified post-swap since go-sdk's actual version-*value* constants are all unexported (Q1) |
| SDK-04 | Error mapping asserted, not inferred | ❌ new — no test today asserts the `IsError:true`/`Content` shape for a missing-required-argument path against the real SDK | new unit or wireoracle-adjacent test | ❌ Wave 0 gap |
| SDK-05 | Schema constraint audit | ❌ new — no test today diffs raw `tools/list` schema JSON pre/post migration | manual diff review (per D-03, no new ceremony) + optional targeted assertion on `additionalProperties`/`type:integer` | ❌ Wave 0 gap, may be satisfied by the existing wire oracle's `TestFrozenTranscriptsMatch` diff review alone, per the calibration note |

### Sampling Rate

- **Per task commit:** `go build ./... && go vet ./...` plus the specific package under change (`go test ./internal/mcp/...`)
- **Per wave merge:** `task test:wireoracle` (the load-bearing gate for this phase)
- **Phase gate:** `task test` (full suite) green, plus a **human-read** full transcript diff per D-01/D-03 — this is the phase's actual acceptance mechanism, not a new automated assertion

### Wire Oracle Gate: how to prove it non-vacuous for THIS phase (per the repo's standing rule)

The wire oracle's non-vacuity was already proven in Phase 1 (`MUTATION-PROOF.md`, four mutations, all demonstrated RED against the *pre-migration* binary). Per this repo's rule ("a gate is not trusted until demonstrated RED against a confirmed-applied mutation"), Phase 2 should re-confirm at least one of those same mutations against the **post-migration** binary before trusting `TestFrozenTranscriptsMatch` as this phase's gate — e.g., re-run Mutation 1 (stray stdout line) against the migrated server and confirm it still goes red. This is cheap (one mutation, ~17s) and closes the "does the gate still work against the new backend" question the calibration note would otherwise leave assumed.

### Wave 0 Gaps

- [ ] A targeted test (new, small) asserting the `IsError:true` + `Content` shape for at least one intentionally-broken handler call against the real migrated server — closes SDK-04's "asserted, not inferred" requirement without new ceremony (one test, not a framework)
- [ ] Re-run at least one of Phase 1's four confirmed mutations against the post-migration binary (see above) — proves the gate, not just the code, survived the swap

*(No framework installation needed — `test/wireoracle` and `go test` already exist and are proven.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | stdio subprocess, no auth surface (per REQUIREMENTS.md Out of Scope: "Authorization work... applies to remote/HTTP MCP servers") |
| V3 Session Management | Marginal | go-sdk's `GetSessionID` (`ServerOptions.GetSessionID`) is HTTP/Streamable-transport-oriented; codegraph-go's stdio transport (`StdioTransport`) does not consult it (`ServerOptions.GetSessionID`'s doc comment: "not consulted when `StreamableHTTPOptions.Stateless` is true" — irrelevant to stdio, which has no `Mcp-Session-Id` header concept at all) |
| V4 Access Control | Yes (unchanged) | `confineToRepoRoot` (`internal/mcp/tools.go:31-45`) — this phase does not touch it; the trust boundary is entirely in codegraph-go's own code, orthogonal to the SDK swap |
| V5 Input Validation | Yes | Migrates from mark3labs' `RequireString`/`GetInt` manual extraction to go-sdk's automatic JSON-Schema validation via `applySchema` (Q5's bonus finding) — **an improvement**: invalid/missing arguments are now rejected before the handler runs, by a general-purpose schema validator, not ad hoc per-field checks |
| V6 Cryptography | No | not applicable — no crypto surface in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A hostile/buggy MCP client sending malformed or oversized `initialize`/`tools/call` arguments | Denial of Service (resource exhaustion) | go-sdk's `applySchema` validation runs before any handler code executes, rejecting malformed input at the protocol boundary — same mitigation tier mark3labs provided, now backed by a general JSON-Schema validator (`google/jsonschema-go`) rather than hand-rolled checks |
| A client attempting to redirect a tool call outside the confined repo root via a crafted `path` argument | Tampering (path traversal / confinement bypass) | `confineToRepoRoot` (`internal/mcp/tools.go`) — **unchanged by this phase**, must not be touched or "helpfully improved" in the same change per PITFALLS' explicit warning (Security Mistakes table) |
| A dependency-graph supply-chain compromise via one of the newly-added transitive modules (`segmentio/encoding`, `golang.org/x/oauth2`, `golang.org/x/time`) | Tampering (supply chain) | `govulncheck` re-audit (SDK-03's own requirement) + existing SBOM generation path — all three new modules are from recognizable, actively-maintained orgs (see Package Legitimacy Audit above) |

## Sources

### Primary (HIGH confidence — real source read and/or real code built and run this session)

- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go` — `ServerOptions`, `Server.capabilities()`, `ss.initialize()`, `AddTool`, `toolForErr`, `Server.Run`, `AddReceivingMiddleware`, `listTools`
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/shared.go` — `negotiatedVersion()`, `ServerRequest.ProtocolVersion()`, `handleReceive`, `Middleware`/`MethodHandler`
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go` — `InitializeResult`, `Implementation`, `ServerCapabilities`, `Tool`, `ToolAnnotations` + its custom `MarshalJSON`, `CallToolResult.SetError`, `Cacheable`
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/session.go` — `ServerSessionState`
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/tool.go` — `ToolHandler`, `ToolHandlerFor` doc comments
- `$GOMODCACHE/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/transport.go` — `StdioTransport`
- `$GOMODCACHE/github.com/google/jsonschema-go@v0.4.2/jsonschema/{doc.go,infer.go,schema.go}` — struct-tag inference, type mapping, `MarshalJSON`/`orderedProperties`
- **This session's own probe server**, built with `go build` against the real `v1.7.0` module and driven with raw JSON-RPC over its actual stdio pipes (same methodology as `test/wireoracle`) — produced the empirical wire bytes quoted throughout Q1, Q3, Q4, Q7, Pitfall 1
- `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/session_line.go`, `internal/mcp/protocol_version.go`, `internal/cli/serve.go` — read in full this session
- `internal/mcp/archtest/protocol_version_test.go`, `internal/mcp/archtest/protocol_version_selftest_test.go`, `internal/cli/archtest/mcp_sdk_confinement_test.go` — read this session
- `testdata/wireoracle/transcripts/{handshake-explore,call-node,toolslist-allowlist,toolslist-no-index}.golden` — read this session as ground truth for pre-migration wire bytes
- `go.mod`, `go.sum`, `go mod why -m` (real invocations against the real repo, this session) — Q9's dependency-closure evidence
- `$GOMODCACHE/github.com/mark3labs/mcp-go@v0.56.0/go.mod` — read this session for Q9's comparison

### Secondary (from prior-phase artifacts, cross-checked against primary sources above)

- `.planning/phases/02-.../02-CONTEXT.md` — D-01 through D-08, Claude's Discretion, canonical refs (locked decisions this research validates against)
- `.planning/phases/01-.../01-CONTEXT.md` — D-01 through D-16 (Phase 1's own decisions, several directly cited above)
- `.planning/research/PITFALLS.md` — Pitfall 6, Pitfall 7, Testing Traps A-D (Pitfall 7's specific claim was checked against primary source and found stale for v1.7.0 — see Q2)
- `test/wireoracle/COVERAGE-BASELINE.md`, `test/wireoracle/MUTATION-PROOF.md` — the frozen scenario set and the one-time mutation matrix (Mutation 3's finding directly informs Q5's bonus finding)

### Tertiary

None — every claim in this document traces to a primary source (real SDK source in `$GOMODCACHE`) or this session's own empirical, built-and-run verification. No web search or training-data recall was used for any SDK-behavior claim.

## Metadata

**Confidence breakdown:**
- Q1 (protocol version control surface): HIGH — exhaustive enumeration + empirical proof
- Q2 (ToolAnnotations omitempty): HIGH — struct read + empirical proof, directly contradicts a stale prior-research claim
- Q3 (InitializeResult/Implementation/ServerCapabilities): HIGH — struct reads + empirical proof for both capability divergences
- Q4 (omitted/2026-07-28 version behavior): HIGH — empirical proof, directly overturns a locked CONTEXT.md lean (flagged prominently for the discuss/plan step to reconcile)
- Q5 (error mapping): HIGH — source read + empirical proof + cross-check against the actual `internal/mcp/tools.go` source
- Q6 (session-line hook): HIGH — source read of the full dispatch chain confirming the mechanism; the specific middleware code itself is a sketch, not yet built (flagged as Open Question 1, low risk)
- Q7 (schema inference): HIGH — source read + empirical `jsonschema.For[T]()` runs against representative structs
- Q8 (transport API): HIGH — straightforward source read, low complexity
- Q9 (go.mod impact): MEDIUM-HIGH — real `go.mod` reads for both SDKs' own requirements; the scratch dry-run's pruned-module-graph result (A1 in Assumptions Log) needs re-confirmation against the real tree's `go mod tidy` output during execution, not because the SDK behavior is uncertain but because a source-free scratch module cannot fully replicate Go's lazy-module-graph pruning

**Research date:** 2026-08-05
**Valid until:** pinned to `go-sdk@v1.7.0` and `jsonschema-go@v0.4.2/v0.4.3` specifically — any version bump to either invalidates the empirical findings above (especially Q2's `ToolAnnotations` behavior and Q7's `additionalProperties`/ordering findings) and would need re-verification. Not a calendar-based expiry; a dependency-pin-based one.
