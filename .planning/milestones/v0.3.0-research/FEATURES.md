# Feature Research: MCP `2026-07-28` Protocol Currency

**Domain:** MCP protocol compliance for a STDIO, tools-only server (`codegraph-go`'s `serve --mcp`)
**Researched:** 2026-08-03
**Confidence:** Mixed — spec text is HIGH (primary source, directly quoted, cross-referenced across the official specification's own pages); empirical client-adoption claims (Q6) are explicitly LOW/UNKNOWN, per the classify-confidence seam's generic per-provider heuristic and the quality gate's instruction to prefer UNKNOWN over a plausible guess.

**Existing surface this milestone touches** (from `internal/mcp/server.go` and `internal/mcp/tools.go`): stdio-only transport via `mark3labs/mcp-go` (`server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))`); tools-only capability; a dynamic catalog gated by `hasIndex` (zero tools registered at `BuildServer` construction time when no `.codegraph/` resolves) and a `CODEGRAPH_MCP_TOOLS` allowlist over 7 companion tools plus an always-on `codegraph_explore`. `mark3labs/mcp-go` currently pins `LATEST_PROTOCOL_VERSION = "2025-11-25"` (confirmed via `pkg.go.dev`, v0.57.0, 2026-07-23) — this is a **Legacy** (initialize-handshake) implementation under the new spec's own terminology, not a **Modern** one.

---

## Q1 — What the spec REQUIRES of stdio vs. Streamable HTTP

**This is the load-bearing finding for the whole milestone.**

### The spec is now genuinely transport-agnostic at the message-semantics layer

> "Protocol semantics are identical on every transport. A transport is a **binding**: it defines how messages are framed and delivered, how request metadata is carried, and how cancellation and termination are signaled. It does not define what the messages mean." — [Transports overview](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports)

> "All implementations **MUST** support the base protocol, versioning, and the message patterns." — [Basic overview](https://modelcontextprotocol.io/specification/2026-07-28/specification/2026-07-28/basic)

So statelessness, `_meta`-carried version/capabilities, `resultType`, MRTR, and `server/discover` are **core-protocol requirements that bind stdio exactly as much as Streamable HTTP.** There is no "stdio gets a lighter-weight core" carve-out.

### `initialize`/`initialized` is genuinely gone — on every transport, including stdio

> "Make MCP stateless: remove the `initialize`/`notifications/initialized` handshake. Every request now carries its protocol version and client capabilities in `_meta`." — [Changelog, major change #2](https://modelcontextprotocol.io/specification/2026-07-28/changelog)

> "Servers **MUST NOT** rely on prior requests over the same connection to establish context (e.g., capabilities, protocol version, client identity). Every request supplies this metadata in its `_meta` field." — [Statelessness](https://modelcontextprotocol.io/specification/2026-07-28/basic)

> "This implies that an open connection, such as a STDIO process, is not a conversation or session: clients may interleave unrelated requests on the same transport, and a server must not treat connection or process identity as a proxy for conversation or session continuity." — same section, explicit Note

There is **no replacement handshake on stdio.** A stdio client learns the server's protocol version and capabilities either (a) implicitly, by sending any request and reading its result, or (b) explicitly via `server/discover` (Q3) — which servers **MUST** implement but clients are only encouraged, not required, to call. Nothing about stdio specifically requires a startup round-trip anymore; the daemon can, in principle, accept `codegraph_explore` as its very first message with no prior negotiation.

### What genuinely IS Streamable-HTTP-only

Everything the spec removes or adds that is HTTP-specific has no stdio analogue and is **irrelevant to us**:
- `Mcp-Session-Id` header removal (HTTP-only concept to begin with — SEP-2567)
- `Mcp-Method`/`Mcp-Name` mandatory headers, `x-mcp-header` custom-header mirroring (SEP-2243) — HTTP transport mirrors `_meta`/argument values into headers for intermediary routing; stdio has "no header layer" at all (explicit stdio-page text below)
- `MCP-Protocol-Version` HTTP header (stdio carries version only in `_meta`, no header equivalent)
- SSE stream resumability / `Last-Event-ID` removal — SSE-specific
- HTTP `400 Bad Request` status-code semantics for malformed `_meta` or version mismatch — stdio has no status codes, only JSON-RPC error objects

> "All request metadata for the stdio transport is carried inline in the JSON-RPC message body... There is no header layer." — [stdio transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio)

### What stdio retains that IS new and applies to us regardless of transport

- Framing rules are **unchanged and already satisfied**: newline-delimited JSON-RPC, no embedded newlines, `stdout` reserved for valid MCP messages only, `stderr` free for logging (already the shape of our `internal/mcp` design — see HYG-02's archtest per `PROJECT.md`).
- Shutdown/unexpected-termination semantics are explicitly restated as stateless-compatible: "If the server process exits unexpectedly, the client **SHOULD** restart it. Because the protocol is stateless, any in-flight requests are simply lost and the client can retry them against the fresh process." This validates our existing daemon-restart model rather than requiring anything new.
- Cancellation via `notifications/cancelled` is unchanged.

### Verdict for Q1

**Table stakes, transport-agnostic, apply to codegraph-go's stdio server exactly as they would to an HTTP server:** per-request `_meta` version/capabilities (Q2), `server/discover` (Q3), `resultType` on every result, `UnsupportedProtocolVersionError` on version mismatch, `CacheableResult` hints on list/read results (Q4), the new error-code partition (`-32020`..`-32099` reserved for MCP).

**Streamable-HTTP-only, N/A to us because we ship no HTTP surface (confirmed anti-feature, see below):** session headers, `Mcp-Method`/`Mcp-Name`/`x-mcp-header`, `MCP-Protocol-Version` header, SSE resumability, HTTP status-code-driven backward-compat detection.

---

## Q2 — `_meta` requirements

Exact reserved keys and cardinality, from [Basic protocol — `_meta`](https://modelcontextprotocol.io/specification/2026-07-28/basic):

| Key | Direction | Required? | Type |
|---|---|---|---|
| `io.modelcontextprotocol/protocolVersion` | client → server, every request | **Yes** | `string` |
| `io.modelcontextprotocol/clientCapabilities` | client → server, every request | **Yes** | `ClientCapabilities` |
| `io.modelcontextprotocol/clientInfo` | client → server | No (but client **SHOULD** send it) | `Implementation` |
| `io.modelcontextprotocol/logLevel` | client → server | No | `LoggingLevel` |
| `io.modelcontextprotocol/serverInfo` | server → client, every result | No (but server **SHOULD** send it) | `Implementation` |

**What a server MUST do:**
- Reject a request missing either required field: `"A request missing any required field is malformed; the server **MUST** reject it with JSON-RPC error code `-32602` (Invalid params)."` (On HTTP this additionally means `400`; stdio has no HTTP status layer, so only the JSON-RPC error applies.)
- Never infer capabilities the client didn't declare: `"A server **MUST NOT** rely on capabilities the client has not declared. If processing a request requires a capability the client did not include in `clientCapabilities`, the server **MUST** return a `MissingRequiredClientCapabilityError` (`-32021`)..."`
- Respond to an unsupported `protocolVersion` with `UnsupportedProtocolVersionError` (`-32022`) listing `supported` versions (Q5).

**What a server is NOT obligated to do:** validate or act on `clientInfo` beyond acceptance — it's explicitly self-reported, unverified, display/debug-only: `"clientInfo and serverInfo are self-reported by the sender and are not verified by the protocol... Implementations **SHOULD NOT** use them to change the behavior of the client or server, and **SHOULD NOT** rely on them for security decisions."` A server MAY ignore `clientInfo` entirely and MAY ignore `logLevel` if it doesn't implement per-request log-level filtering.

**What a server MUST return:** nothing is strictly required in response `_meta` — `serverInfo` is a **SHOULD**, not a MUST. Practically: since every `tools/list`/`server/discover` result also carries `CacheableResult` fields (Q4), those ARE a MUST on the listed operations, and that's a distinct requirement from `_meta` per se.

**Complexity for us:** LOW-MEDIUM. `mark3labs/mcp-go` currently has no concept of per-request `_meta.io.modelcontextprotocol/*` fields at all (it does an `initialize`-based handshake per its own `LATEST_PROTOCOL_VERSION`); adopting this requires either an SDK that implements it or hand-rolling `_meta` parsing/validation in our own JSON-RPC layer — this is exactly the "SDK decision" the milestone already names as a required deliverable.

---

## Q3 — `server/discover`

**Servers MUST implement it — this is unambiguous and settles the milestone's central open question.**

> "Servers **MUST** implement `server/discover`." — [Discovery](https://modelcontextprotocol.io/specification/2026-07-28/server/discover)
> "Add `server/discover`: servers MUST implement this RPC to advertise their supported protocol versions, capabilities, and identity. Clients MAY call it before any other request..." — [Changelog, major change #3](https://modelcontextprotocol.io/specification/2026-07-28/changelog)

It is optional **for clients to call**, never optional for a server to expose. For a dynamic-catalog server like ours this fully resolves the ambiguity flagged in the question: if codegraph-go adopts `2026-07-28` at all, `server/discover` is not a nice-to-have discovery convenience — it is as mandatory as answering `tools/list`.

**Exact request shape** (no body params beyond standard `_meta`):
```json
{
  "jsonrpc": "2.0", "id": "discover-1", "method": "server/discover",
  "params": { "_meta": {
    "io.modelcontextprotocol/protocolVersion": "2026-07-28",
    "io.modelcontextprotocol/clientInfo": {"name": "ExampleClient", "version": "1.0.0"},
    "io.modelcontextprotocol/clientCapabilities": {}
  }}
}
```

**Exact response shape** (`DiscoverResult`):
```json
{
  "jsonrpc": "2.0", "id": "discover-1",
  "result": {
    "resultType": "complete",
    "supportedVersions": ["2026-07-28"],
    "capabilities": {"tools": {}, "resources": {}},
    "_meta": {"io.modelcontextprotocol/serverInfo": {"name": "ExampleServer", "version": "1.0.0"}},
    "instructions": "This server provides weather and resource utilities.",
    "ttlMs": 3600000,
    "cacheScope": "public"
  }
}
```
Fields: `supportedVersions` (array of version strings the client should choose from), `capabilities` (object — for us this would just be `{"tools": {}}`, no `resources`/`prompts`/`roots`/`sampling`/`logging`), `_meta.serverInfo` (SHOULD, not MUST), `instructions` (optional natural-language guidance for the LLM), plus the `CacheableResult` fields `ttlMs`/`cacheScope` (Q4 — `server/discover` "supports caching" per spec).

**Complexity:** LOW once the underlying SDK/transport supports per-request `_meta` parsing at all — it's one more RPC method with a small, static-shaped response (our supported-version list is a 1-element array either way; see Q5's adopt-or-defer framing).

---

## Q4 — `ttlMs` / `cacheScope` (SEP-2549) — the dynamic-catalog collision

This is the sharpest concrete risk in the whole milestone, and the spec resolves it cleanly if applied correctly.

**Where it's required:** `CacheableResult` fields are **mandatory** (not optional) on any `resultType: "complete"` result from `server/discover`, `tools/list`, `prompts/list`, `resources/list`, `resources/templates/list`, `resources/read`. [Caching](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching): `"Servers MUST include caching hints on results with resultType: 'complete' returned by [these operations]."` `"Servers **MUST** provide a `ttlMs` value that is `>= 0`."`

**`ttlMs` exact semantics** (HTTP `Cache-Control: max-age` analogue):
- `ttlMs == 0` → `"the response **SHOULD** be considered immediately stale. The client MAY re-fetch every time the result is needed."`
- `ttlMs > 0` → client **SHOULD** consider fresh for that many ms after receipt.
- `ttlMs` absent → client **SHOULD** assume `0` — but spec flags `"This should only occur in older server versions"`, i.e. a `2026-07-28`-compliant server must never omit it.
- `ttlMs < 0` → client **SHOULD** treat as `0`.
- Freshness math: `now < t_received + ttlMs`.

**`cacheScope` exact values:**
| Value | Meaning |
|---|---|
| `"public"` | No user-specific data; any client, shared gateway, or caching proxy MAY store and reuse across users. Appropriate for tool/prompt/resource-template lists identical for everyone. |
| `"private"` | User/authorization-scoped; MUST NOT be shared across authorization contexts. Appropriate for `resources/read` or per-user filtered lists. |

**The specific collision this milestone must solve:** our tool catalog legitimately changes — zero tools pre-`codegraph init`, then `codegraph_explore` (+ allowlisted companions) post-init, and `hasIndex` is currently evaluated **once**, at `BuildServer` construction time (`internal/mcp/server.go`), which itself is a session-scoped assumption the stateless model no longer licenses (see the Statelessness quote in Q1 — a stdio process is explicitly *not* a session, so nothing stops a client from calling `tools/list` both before and after `codegraph init` against the same long-lived daemon process). If we return, say, `ttlMs: 300000, cacheScope: "public"` on an empty catalog and a client honors that TTL, the client will not re-`tools/list` for five minutes after `codegraph init` makes tools real — an entirely spec-legal but user-visible "my tools didn't show up" bug.

**The spec's answer, explicit and unambiguous:** `ttlMs: 0` is exactly "do not cache" / "immediately stale" — there is no separate sentinel needed. The spec's own guidance:

> "A server **MAY** provide `ttlMs` without advertising `listChanged: true` in its capabilities. In this case, the client relies entirely on TTL-based freshness." — [Interaction with Notifications](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching)

> "A server **MAY** advertise `listChanged: true` **and** provide `ttlMs`. In this case, the client can use the TTL to avoid unnecessary refetches between notifications, and the notification acts as an immediate invalidation signal." — same section

> "When a relevant notification is received while a cached response is still fresh, the notification **invalidates** the cached response and it should be considered immediately stale." — same section

So there are two independently sufficient correctness fixes, of different complexity:

1. **LOW complexity, correct by itself:** always return `ttlMs: 0, cacheScope: "public"` on `tools/list` (and `server/discover`). This is fully spec-compliant (`ttlMs=0` legally means "our catalog is volatile, re-check every time") and requires zero new infrastructure — just correct field population once the SDK/transport speaks `CacheableResult` shapes at all. This alone eliminates the stale-empty-catalog risk: a spec-compliant client that honors `ttlMs=0` will re-`tools/list` on its next need for the tool list, by definition.
2. **MEDIUM-HIGH complexity, a genuine differentiator on top:** additionally advertise `tools.listChanged: true` and push `notifications/tools/list_changed` the moment `codegraph init`/`uninit` flips `hasIndex`. This requires implementing the new `subscriptions/listen` long-lived stream mechanism (SEP-2575's replacement for the old GET/subscribe endpoints) — a materially bigger lift than (1), since our current `mark3labs/mcp-go`-based server has no open-stream/subscription concept at all. Buys instant invalidation instead of relying on the client's next natural need-driven fetch, but (1) already closes the correctness gap on its own.

**Recommendation for requirements-definition:** (1) is TABLE STAKES if the milestone adopts `2026-07-28` at all — it is a MUST-level spec field, and getting its *value* right (0, not some plausible-looking 300000) is the one line-item that directly prevents the exact bug the milestone context worries about. (2) is a DIFFERENTIATOR — worth scoping only after (1) ships and only if the SDK decision lands on something that makes `subscriptions/listen` tractable.

One more correctness note surfaced by this research, independent of caching: `BuildServer`'s current `hasIndex bool` parameter is evaluated once at construction. Under the stateless model this is now a genuine spec-adjacent risk regardless of `ttlMs` — if the daemon process outlives a `codegraph init` (plausible: watcher-on-MCP is already default per `PROJECT.md`), a spec-compliant client that correctly treats `ttlMs=0` as "re-fetch" will still get a **stale in-process** answer unless `tools/list`'s handler re-checks `hasIndex` per-call rather than trusting the value captured at server startup. This is a codegraph-go-specific implementation detail the spec doesn't and can't mandate (it only governs the wire contract), but it's the other half of making `ttlMs=0` actually true rather than a lie about a value we compute once and never revisit.

---

## Q5 — Version negotiation and backward compatibility

This is the second primary requirements-driver, and the finding is more severe than the milestone context's framing ("the roster failure mode is quiet — tools silently not advertised") suggests for the naive case.

**Negotiation mechanics (Modern↔Modern):** every request self-declares its version in `_meta`; if unsupported, the server **MUST** respond `UnsupportedProtocolVersionError` (`-32022`) with a `data.supported` array; the client **SHOULD** retry with a mutually supported version or surface an error. There is no separate negotiation phase — negotiation *is* per-request error/retry.

**The compatibility matrix** ([Versioning: Backward Compatibility](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning#backward-compatibility-with-initialization-based-versions)), condensed to the outcomes that matter for our 8-agent roster:

| Client era | Server era | Outcome |
|---|---|---|
| Modern | Modern | Works — `UnsupportedProtocolVersionError` + retry if versions differ. |
| Modern | Legacy | **Fails.** Server may reject with an implementation-defined error, stay silent, or process the request under legacy semantics anyway. |
| Dual-era | Modern | Works — probe (`server/discover` on stdio) returns a `DiscoverResult`, client stays modern. |
| Dual-era | Legacy | Works — probe fails/times out, client falls back to `initialize`. |
| **Legacy** | **Modern** | **Fails, hard.** `"stdio: the server rejects `initialize` with a JSON-RPC error... Legacy clients have no fall-forward mechanism."` |
| Legacy | Dual-era | Works — server answers `initialize`, serves legacy semantics. |
| Legacy | Legacy | Works (out of scope for the new spec). |

**This is the row that matters for codegraph-go:** our server is currently **Legacy** (mark3labs/mcp-go caps at `2025-11-25`, an `initialize`-handshake era per the spec's own terminology). Every one of the 8 roster clients is, per the empirical audit below, either confirmed-Legacy or unconfirmed/UNKNOWN — **none is confirmed-Modern**. That means today's actual risk direction is the opposite of "our server silently drops tools for a client that moved on" — it is symmetric and currently moot (Legacy↔Legacy works fine, out of scope for the new spec). The real fork is **what happens if codegraph-go's server becomes Modern-only** (e.g., by fully migrating to `modelcontextprotocol/go-sdk` and dropping legacy `initialize` support) while the roster clients are still Legacy: per the matrix, that is **"Legacy client, Modern server: Fails, hard"** — every currently-Legacy agent in the roster would outright fail to connect, not silently lose tools. This is strictly worse than "quiet degradation" and is the single strongest argument in this research for treating **dual-era server support as a hard prerequisite of adoption**, not an optional nicety:

> "A server that wishes to support both legacy clients... and modern clients... **MAY** implement both behaviors... A dual-era server **MAY** serve both eras concurrently on the same endpoint or process."

**Concrete guidance a `2026-07-28`-only server MUST still give legacy callers**, if adoption proceeds without dual-era support (i.e., an explicit, accepted breaking change):
> "A server that supports only modern versions **SHOULD** name the protocol versions it supports in any error it returns to an `initialize` request, on any transport: legacy clients have no fall-forward mechanism, and this message may be the only diagnostic they can surface to users."

**Where the milestone context's "quiet failure" framing DOES apply:** a client that is itself Dual-era (probes with `server/discover`, falls back to `initialize` on any non-modern error) should degrade gracefully against our current Legacy server — this is the "Dual-era client, Legacy server: Works" row. The quiet-degradation risk materializes specifically for a *naive* Dual-era client implementation that doesn't correctly implement the fallback-on-any-error rule (the spec explicitly warns fallback "**MUST NOT** be keyed to one specific error code" because legacy servers return varied, implementation-defined errors) — such a client could misclassify our Legacy server's response and give up rather than falling back, which would look like "tools not advertised" with no error surfaced to the user. This is a client-side bug class the spec anticipates and warns against, not something codegraph-go's server can fully defend against from the server side — but it is exactly the scenario the milestone's planned "real-client MCP verification" harness needs to probe.

**Table stakes vs. differentiator split for Q5:**
- TABLE STAKES (if adopting at all): dual-era server support (serve both `initialize`-based Legacy and per-request-`_meta` Modern on the same stdio process), `UnsupportedProtocolVersionError` on any Modern request with a version we don't support, naming supported versions in `initialize`-rejection errors.
- ANTI-FEATURE: dropping Legacy (`initialize`) support entirely on the strength of the new spec alone, before empirical evidence (Q6) that the 8-agent roster has actually moved. Per the matrix, this converts an invisible risk into a certain, hard breakage for every still-Legacy client in the roster.

---

## Q6 — Empirical: what revision do the 8 roster clients actually negotiate?

**Methodology note:** primary sources are `canimcp.dev` (a caniuse-style, dated, sourced MCP compatibility matrix — 41 clients, 34 features, per-cell verification dates and provenance labels) plus targeted web search. `canimcp.dev`'s own "Core Protocol" category — covering exactly "Stateless requests & per-request negotiation," "Server discovery," and "Multi round-trip requests," i.e. the three `2026-07-28`-defining features — is marked **`Unknown`** for every client checked below, and every "Verified" date found is `2026-07-01` or `2026-02-02`, both **before** the spec finalized on `2026-07-28`. Per the quality gate, UNKNOWN is preferred over inference here, and that is what the evidence actually supports for every one of the 8 roster clients — none is confirmed either way.

| Client | SDK/version evidence found | Spec revision confirmed? | Confidence | Source |
|---|---|---|---|---|
| **Claude Code** | Supports stdio + Streamable HTTP; `tools/list` + `list_changed` supported. Anthropic's own blog frames `2026-07-28` support for Claude products as "rolling out ... soon" (future tense as of the post). | **UNKNOWN** (Core Protocol row explicitly `Unknown`, verified 2026-07-01 — pre-dates the spec) | LOW | [canimcp.dev/client/claude-code](https://canimcp.dev/client/claude-code/); [Claude blog: "MCP 2026-07-28 spec: stateless core, coming to Claude"](https://claude.com/blog/bringing-mcp-2026-07-28-to-claude) |
| **Cursor** | Supports stdio, HTTP+SSE (deprecated), Streamable HTTP; `tools/list_changed` explicitly **Not supported** as of v0.47 (conformance-tested 2026-06-28). | **UNKNOWN** | LOW | [canimcp.dev/client/cursor-vscode](https://canimcp.dev/client/cursor-vscode/) |
| **Codex CLI (OpenAI)** | `tools/call`/`tools/list` supported per an automated crawl dated 2026-02-02; transport support fields all `Unknown`. A secondary web summary claimed "Codex CLI supports MCP protocol version 2025-06-18" but this could not be traced to a primary, dated source and is not corroborated by canimcp.dev's own per-feature data. | **UNKNOWN** — treat the "2025-06-18" claim as unverified, not a fact | LOW | [canimcp.dev/client/codex](https://canimcp.dev/client/codex/) |
| **opencode** | `tools/call`/`tools/list`/`list_changed` all supported per a 2026-02-02 automated crawl. A GitHub issue notes the SDK "dynamically negotiates MCP-Protocol-Version correctly during normal operation" but that its own `mcp debug` subcommand hardcodes `protocolVersion: "2024-11-05"` — i.e. even opencode's own tooling doesn't agree internally on what it speaks. | **UNKNOWN** | LOW | [canimcp.dev/client/opencode](https://canimcp.dev/client/opencode/); GitHub issue [anomalyco/opencode#28567](https://github.com/anomalyco/opencode/issues/28567) |
| **Gemini CLI** | Documented as supporting stdio, SSE, and Streamable HTTP transports generally; no protocol-revision string found in official docs or GitHub issues surfaced. | **UNKNOWN** | LOW | [google-gemini/gemini-cli docs: MCP servers](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html) |
| **Hermes Agent** (NousResearch — the roster's actual "Hermes," confirmed via `github.com/colbymchenry/codegraph`'s own agent list) | Built-in MCP client since v0.2.0, MCP server since v0.6.0. **Caution:** an unrelated Elixir package also named "Hermes MCP" (`hexdocs.pm/hermes_mcp`) documents conformance to `2024-11-05` — this is a different codebase and must not be attributed to NousResearch's Hermes Agent. No protocol-revision evidence for the actual roster client was found. | **UNKNOWN** | LOW | [hermes-agent.ai MCP integration guide](https://hermes-agent.ai/blog/hermes-mcp-integration-guide); [github.com/colbymchenry/codegraph](https://github.com/colbymchenry/codegraph) (confirms roster identity) |
| **Antigravity (Google)** | Confirmed to support MCP generally (MCP Store, shared config with Gemini CLI/IDE at `~/.gemini/config/mcp_config.json`); no protocol-revision string found. | **UNKNOWN** | LOW | [antigravity.google/docs/mcp](https://antigravity.google/docs/mcp) |
| **Kiro** | Confirmed to support MCP generally (IDE + CLI docs, remote MCP servers). Kiro CLI's "protocol version 1" reference found in search results is for **ACP (Agent Client Protocol)**, an unrelated protocol from Zed — not MCP. No MCP protocol-revision string found. | **UNKNOWN** | LOW | [kiro.dev/docs/mcp](https://kiro.dev/docs/mcp/) |

**Bottom line for the compatibility risk assessment:** zero of the 8 roster clients have a *confirmed* `2026-07-28` implementation as of this research (2026-08-03, five days post-finalization) — entirely consistent with a brand-new spec revision. But equally, **none can be confirmed as staying on `2025-11-25`/Legacy either** with primary-source certainty for every client; the strongest evidence (Claude Code, Cursor via canimcp.dev) predates the spec and is silent specifically on the "Core Protocol" (statelessness/discovery/MRTR) category that defines this revision. **Do not plan phase work assuming either direction is settled — the milestone's own "real-client MCP verification" harness is the correct mechanism to resolve this, not further desk research.** The official Go SDK (`modelcontextprotocol/go-sdk`) does confirm `2026-07-28` support in v1.7.0 pre-releases as of mid-2026 (per `.claude/CLAUDE.md`'s Alternatives Considered table, cross-checked against the MCP blog's "Beta SDKs for the 2026-07-28 MCP Spec Release Candidate" post) — but that is *our own potential server SDK*, not evidence about any client in the roster.

---

## Feature Landscape

### Table Stakes (spec-required for a stdio, tools-only server, if `2026-07-28` is adopted at all)

| Feature | Why Required | Complexity | Notes |
|---|---|---|---|
| Per-request `_meta.protocolVersion` + `clientCapabilities` parsing/validation | Spec MUST: malformed/missing → `-32602` | MEDIUM | No handshake state to hang this off of; must be a per-call check in the new SDK/transport layer, not a one-time startup check |
| `resultType: "complete"` on every non-MRTR result | Spec MUST field on every result | LOW | Mechanical; every existing tool-result builder needs the field added |
| `UnsupportedProtocolVersionError` (`-32022`) with `data.supported` on version mismatch | Spec MUST | LOW | Single small error path |
| `server/discover` implementation | Spec MUST for servers (Q3) | LOW-MEDIUM | Our `capabilities` response is trivially `{"tools": {}}`; the work is mostly in the transport layer supporting it at all |
| `ttlMs`/`cacheScope` on `tools/list` and `server/discover`, correctly `ttlMs: 0` for our volatile catalog | Spec MUST field; wrong *value* directly causes the milestone's named failure mode | LOW | The dominant fix for Q4's dynamic-catalog risk; must ship alongside a per-call (not per-process) `hasIndex` re-check |
| Dual-era (Legacy `initialize` + Modern per-request) server support during the transition | Not spec-mandated per se, but the compatibility matrix (Q5) shows Modern-only breaks every still-Legacy roster client outright | HIGH | The single biggest architectural lift this research identifies; directly gates the milestone's "adopt or dated-defer" decision |
| Deterministic `tools/list` ordering | New minor-change SHOULD, enables client-side caching + LLM prompt-cache hits | LOW | Our tool set is already small and statically ordered (`companionNames` fixed slice + `codegraph_explore` first) — likely already compliant, verify only |

### Differentiators (worth adopting, not spec-forced)

| Feature | Value Proposition | Complexity | Notes |
|---|---|---|---|
| `tools.listChanged: true` + `notifications/tools/list_changed` on `codegraph init`/`uninit` | Instant catalog-refresh signal instead of relying on the client's next natural `tools/list` call after a `ttlMs:0` cache miss | HIGH | Requires implementing `subscriptions/listen`, a new long-lived-stream mechanism our current SDK has no concept of at all — defer until after the LOW-complexity `ttlMs:0` fix ships and is proven sufficient |
| `instructions` field on `server/discover` response | Natural-language guidance surfaced to the LLM about how to use the server (e.g., "call `codegraph_explore` before `codegraph_node` for unfamiliar symbols") | LOW | Pure value-add once `server/discover` exists at all |
| `io.modelcontextprotocol/serverInfo` in every result `_meta` | Self-identification for debugging/logging across the roster | LOW | SHOULD, not MUST; cheap once the SDK plumbs `_meta` at all |
| OpenTelemetry trace-context passthrough (`traceparent`/`tracestate`/`baggage` in `_meta`) | Standardized tracing if/when codegraph-go grows observability tooling | LOW | Only relevant if OTel is ever adopted elsewhere in the project; not currently in scope |

### Anti-Features (explicitly do NOT adopt for this milestone)

| Feature | Why It Might Seem Relevant | Why Problematic For Us | Alternative |
|---|---|---|---|
| Streamable HTTP transport | The stateless-core motivation is explicitly HTTP-scaling (serverless/edge/load-balancer deployment) — easy to conflate "adopt the new spec" with "add HTTP" | codegraph-go is deliberately stdio-only, one static binary, no daemon-over-network story; none of the HTTP-specific requirements (`Mcp-Method`/`Mcp-Name` headers, session-header removal, SSE resumability removal) apply to us at all | Stay stdio-only; the stateless core's benefits (per-request `_meta`, no handshake) apply to stdio too, independent of transport choice |
| Roots, Sampling, Logging capabilities | These are core MCP features that existed pre-`2026-07-28` and might look like a completeness gap | All three are **Deprecated** as of this exact revision (SEP-2577), with migrations explicitly pointed elsewhere: "pass directories or files via tool parameters... instead of Roots; integrate directly with LLM provider APIs instead of Sampling; log to stderr (stdio)... instead of Logging." We already do all three replacements natively (tool args carry paths, we have no sampling need, we already log to stderr per HYG-01) | No action — this is already our existing, spec-endorsed posture; do not add these deprecated features |
| HTTP+SSE transport | Might appear as a lower-effort HTTP option than full Streamable HTTP | Reclassified Deprecated under the new feature-lifecycle policy (SEP-2596), on top of already being irrelevant since we ship no HTTP surface at all | N/A — stdio only |
| `x-mcp-header` tool-parameter annotations | New tool-schema feature that sounds broadly useful | HTTP-transport-only by definition (mirrors args into HTTP headers for intermediaries); spec explicitly says stdio clients "MAY ignore `x-mcp-header` annotations entirely" | Ignore entirely |
| Dropping Legacy (`initialize`) support immediately, on spec-currency grounds alone | "We should be spec-current" is a reasonable-sounding instinct | Per Q5's compatibility matrix, this converts every still-Legacy roster client (confirmed-or-unknown for all 8, per Q6) from "works" to "fails, hard, with no client-side fallback" | Dual-era support during a measured transition, or an explicit dated defer per the milestone's own stated acceptable-outcome framing |

## Feature Dependencies

```
Per-request _meta parsing/validation (table stakes)
    └──requires──> SDK/transport decision (mark3labs/mcp-go legacy-only vs modelcontextprotocol/go-sdk)
                       └──requires──> Real-client verification harness (must precede any SDK swap,
                                       per PROJECT.md: "mcp-go's own client silently skips malformed
                                       lines and cannot fail a purity test... validating a new SDK
                                       with that SDK's client is circular")

server/discover (table stakes)
    └──requires──> Per-request _meta parsing/validation (same transport-layer prerequisite)

ttlMs=0 / cacheScope fix on tools/list (table stakes)
    └──requires──> CacheableResult shape support in whatever SDK/transport is chosen
    └──enhances──> per-call (not per-process) hasIndex re-evaluation in BuildServer/tools/list handler
                    (a codegraph-go-specific correctness fix the spec doesn't mandate but that makes
                    ttlMs=0 actually TRUE rather than a lie about a value computed once at startup)

tools.listChanged + notifications/tools/list_changed (differentiator)
    └──requires──> subscriptions/listen stream mechanism (new, not present in mark3labs/mcp-go today)
    └──enhances──> ttlMs=0 fix (notification is a complement to TTL-based freshness, not a replacement —
                    spec: "TTL and server-push notifications are complementary")

Dual-era server support (table stakes IF adopting) ──conflicts──> Modern-only migration
    (per Q5's compatibility matrix: Modern-only + any still-Legacy roster client = hard failure,
     not the "quiet" degradation the milestone context worried about)
```

### Dependency Notes

- **Per-request `_meta` parsing requires the SDK decision, and the SDK decision requires the verification harness first.** This is not this research's invention — it's already recorded in `PROJECT.md`'s milestone framing ("Verification precedes migration") and this research's findings reinforce exactly why: `mark3labs/mcp-go` has no `_meta`/`server/discover`/`CacheableResult`/dual-era concept at all today, so any of Q1-Q5's table-stakes items forces the SDK question before they can be implemented.
- **`ttlMs=0` enhances, and is enhanced by, a per-call `hasIndex` check** — they are two halves of the same correctness property (a client that honors `ttlMs=0` will re-ask, but only a per-call `hasIndex` check gives it a different, correct answer when it does).
- **`listChanged` conflicts with nothing but is strictly additive on top of `ttlMs=0`**, not a substitute for it — do not scope `listChanged` as a way to *avoid* fixing `ttlMs`; the spec treats them as complementary, and `ttlMs=0` alone already closes the correctness gap at much lower cost.
- **Dual-era support conflicts with a clean Modern-only migration** — this is the crux of the milestone's "adopt or dated-defer" framing. A dated defer sidesteps the conflict entirely (stay Legacy, revisit within the 12-month window); adopting without dual-era support resolves the conflict by accepting breakage for any still-Legacy roster client, which per Q6 cannot currently be ruled out for any of the 8.

## MVP Definition

### Launch With (if the milestone's decision is ADOPT)

- [ ] Real-client verification harness (non-circular, does not use the SDK-under-test as its own oracle) — **must land before any of the below**, per `PROJECT.md`
- [ ] SDK decision recorded (`mark3labs/mcp-go` vs `modelcontextprotocol/go-sdk`), with an explicit pinned/asserted protocol version (no more `mcp.LATEST_PROTOCOL_VERSION` floating pin)
- [ ] Dual-era serving on the stdio process (Legacy `initialize` fallback + Modern per-request `_meta`) — essential, per Q5, to avoid hard-breaking any still-Legacy roster client
- [ ] `server/discover` implemented (spec MUST for servers)
- [ ] Per-request `_meta.protocolVersion`/`clientCapabilities` validation with correct `-32602`/`UnsupportedProtocolVersionError` error paths
- [ ] `resultType: "complete"` on all existing tool results
- [ ] `ttlMs: 0, cacheScope: "public"` on `tools/list` and `server/discover`, paired with a per-call (not per-process) `hasIndex` re-check in the tools/list handler

### Add After Validation (if ADOPT, next iteration)

- [ ] `tools.listChanged: true` + `notifications/tools/list_changed` via `subscriptions/listen` — once the SDK/transport supports long-lived streams and the LOW-complexity `ttlMs=0` fix is proven sufficient in practice
- [ ] `instructions` field on `server/discover` populated with codegraph-usage guidance
- [ ] `io.modelcontextprotocol/serverInfo` in result `_meta`

### Future Consideration / Explicit Non-Goals

- [ ] Streamable HTTP transport — no current product reason; would only become relevant if codegraph-go ever grows a remote/team-server story (the deferred Team Scale milestone), and even then is a separate, additive transport, not a replacement for stdio
- [ ] Roots/Sampling/Logging — deprecated by the spec itself; do not add
- [ ] `x-mcp-header` — HTTP-only, N/A on stdio

### If the milestone's decision is DEFER

- [ ] Record an explicit, dated defer decision naming the 12-month minimum deprecation window (per `SEP-2596`'s feature-lifecycle policy) as the re-evaluation trigger
- [ ] No code changes required this milestone beyond the decision record itself
- [ ] Re-run Q6's empirical client audit at the next milestone boundary — it is the fastest-decaying finding in this research and the one most likely to flip the adopt/defer calculus

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---|---|---|---|
| SDK decision + verification harness | HIGH (gates everything else) | HIGH | P1 |
| Dual-era server support | HIGH (prevents hard breakage across the roster) | HIGH | P1 |
| `server/discover` | MEDIUM (spec-mandated; low direct user visibility) | LOW-MEDIUM | P1 |
| `ttlMs`/`cacheScope` correctness on `tools/list` | HIGH (directly prevents the named "tools vanish after init" bug) | LOW | P1 |
| Per-call `hasIndex` re-check | HIGH (makes the `ttlMs=0` fix actually true) | LOW | P1 |
| `_meta` validation + `UnsupportedProtocolVersionError` | MEDIUM (spec-mandated correctness) | MEDIUM | P1 |
| `tools.listChanged` + `subscriptions/listen` | LOW-MEDIUM (nice UX, not correctness-critical given `ttlMs=0`) | HIGH | P2 |
| `server/discover` `instructions` field | LOW | LOW | P3 |
| Streamable HTTP | N/A to current product | HIGH | Out of scope |

**Priority key:** P1: table stakes if adopting; P2: differentiator, second wave; P3: nice-to-have polish.

## Sources

**Primary (official specification — HIGH confidence, directly quoted, cross-referenced across pages):**
- [modelcontextprotocol.io/specification/2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28) — overview, security principles
- [.../changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog) — full major/minor/deprecated change list with SEP references
- [.../basic](https://modelcontextprotocol.io/specification/2026-07-28/basic) — JSON-RPC message rules, statelessness, `_meta` field table and per-request/per-response requirements, error-code partition, JSON Schema rules, icons
- [.../basic/transports](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports) — transport-agnostic message-semantics statement, backward-compat pointer
- [.../basic/transports/stdio](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio) — framing, shutdown, cancellation, stdio-specific backward-compatibility probe rules
- [.../basic/versioning](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning) — Modern/Legacy/Dual-era terminology, negotiation flow, full compatibility matrix, `server/discover` MUST-implement statement
- [.../server/discover](https://modelcontextprotocol.io/specification/2026-07-28/server/discover) — exact request/response schema, `DiscoverResult` fields, when-to-call guidance
- [.../server/tools](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) — tools capability, `tools/list`/`tools/call`, MRTR `InputRequiredResult`, `listChanged` notification, tool-name rules, `x-mcp-header`, security considerations
- [.../server/utilities/caching](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching) — full `CacheableResult`/`ttlMs`/`cacheScope` specification, pagination interaction, notification interaction, security considerations

**SDK/dependency evidence (MEDIUM confidence, official package documentation):**
- [pkg.go.dev/github.com/mark3labs/mcp-go/mcp](https://pkg.go.dev/github.com/mark3labs/mcp-go/mcp) — confirms `LATEST_PROTOCOL_VERSION = "2025-11-25"`, `ValidProtocolVersions` list, current release v0.57.0 (2026-07-23); no `2026-07-28` support
- [blog.modelcontextprotocol.io/posts/sdk-betas-2026-07-28](https://blog.modelcontextprotocol.io/posts/sdk-betas-2026-07-28/) — official TypeScript/Python/Go/C# SDKs confirmed shipping `2026-07-28` beta support

**Empirical client-adoption evidence (LOW confidence per the classify-confidence seam; UNKNOWN is the honest answer for all 8 roster clients on the specific `2026-07-28` question — see Q6 table for full per-client sourcing):**
- [canimcp.dev](https://canimcp.dev/) and per-client pages (`/client/claude-code/`, `/client/cursor-vscode/`, `/client/codex/`, `/client/opencode/`) — dated, sourced, per-feature compatibility matrix; all "Core Protocol" rows `Unknown` as of last verification (2026-07-01 / 2026-02-02), which predates the spec's 2026-07-28 finalization
- [claude.com/blog/bringing-mcp-2026-07-28-to-claude](https://claude.com/blog/bringing-mcp-2026-07-28-to-claude) — Anthropic's own announcement frames Claude-product support as forthcoming, not yet shipped, as of the post
- [google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html), [antigravity.google/docs/mcp](https://antigravity.google/docs/mcp), [kiro.dev/docs/mcp](https://kiro.dev/docs/mcp/), [hermes-agent.ai](https://hermes-agent.ai/blog/hermes-mcp-integration-guide) — general MCP-support confirmation for Gemini CLI, Antigravity, Kiro, Hermes Agent; no protocol-revision string found for any
- [github.com/anomalyco/opencode/issues/28567](https://github.com/anomalyco/opencode/issues/28567) — internal inconsistency evidence (opencode's own debug tooling hardcodes an older version string than its SDK negotiates)
- [github.com/colbymchenry/codegraph](https://github.com/colbymchenry/codegraph) — confirms roster identity (disambiguates NousResearch's "Hermes Agent" from an unrelated Elixir "hermes_mcp" package)

---
*Feature research for: MCP protocol currency, codegraph-go v0.3.0*
*Researched: 2026-08-03*
