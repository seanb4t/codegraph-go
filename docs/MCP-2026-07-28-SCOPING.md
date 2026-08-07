# MCP spec revision `2026-07-28`: stdio, tools-only applicability (backlog 999.6)

**Compiled:** 2026-08-05. Source: [CITED:
`modelcontextprotocol.io/specification/2026-07-28/changelog`, fetched
2026-08-04] — every row below transcribes that changelog's own wording for
"what it does"; this document adds only the applicability verdict and
reason for codegraph-go's specific shape: a **stdio-only, tools-only** MCP
server with no `resources`, no `prompts`, no sampling, no roots, and no
HTTP transport.

## Why this table exists

PITFALLS Pitfall 1 is the specific failure this document is built to
prevent: the `2026-07-28` revision is overwhelmingly an HTTP-scaling
release (session removal, header requirements, SSE reclassification,
OAuth hardening), and a plan that reads its headline changes without
checking each one against codegraph-go's actual transport risks importing
HTTP-scaling guidance onto a transport that was never sessioned,
never load-balanced, and never had a header layer to begin with. Every row
below states explicitly whether the change is HTTP-only (and therefore
inert here), transport-agnostic core protocol (and therefore real work),
or already satisfied by decisions this codebase made for unrelated,
pre-existing reasons.

## SEP-by-SEP applicability

| SEP / Change | What it does (changelog's own words) | Applies to codegraph-go (stdio, tools-only)? | Reason |
|---|---|---|---|
| **SEP-2567** — remove protocol-level sessions, `Mcp-Session-Id` header | "Remove protocol-level sessions and the `Mcp-Session-Id` header from the Streamable HTTP transport." | **N/A — HTTP-only** | stdio never had `Mcp-Session-Id`; there is no header layer on stdio at all |
| **SEP-2575** — stateless core: `_meta` version/capabilities, remove `initialize` handshake | "Make MCP stateless: remove the `initialize`/`notifications/initialized` handshake. Every request now carries its protocol version and client capabilities in `_meta`." | **Applicable** (Phase 3 work) | Core protocol, transport-agnostic — stdio binds the same message semantics as any transport |
| **SEP-2575** — `server/discover` | "Add `server/discover`: servers MUST implement this RPC... Clients MAY call it before any other request for up-front version selection, or use it as a backward-compatibility probe on STDIO." | **Applicable, MUST** (Phase 3 work) | Explicitly named as a stdio-relevant probe mechanism by the spec's own text; maps to SPEC-01 and SPEC-07 |
| **SEP-2575** — `subscriptions/listen` (replaces GET/subscribe) | "Replace the HTTP GET endpoint and `resources/subscribe`/`resources/unsubscribe` with `subscriptions/listen`..." | **Applicable in principle, differentiator not table-stakes** | Transport-agnostic mechanism; codegraph-go has no `resources` capability today, and `tools.listChanged` (the one subscription this server needs) is explicitly a Phase-5, second-wave item — maps to SPEC-09 |
| **SEP-2575** — remove `ping`, `logging/setLevel`, `notifications/roots/list_changed` | "Remove `ping`, `logging/setLevel`, and `notifications/roots/list_changed`. Log level is now set per-request via `io.modelcontextprotocol/logLevel`..." | **Applicable, already-compliant** | codegraph-go implements none of roots/sampling/logging capabilities; nothing to remove |
| **SEP-2663** — tasks extension redesign | "Move experimental tasks out of the core protocol and into an official extension (`io.modelcontextprotocol/tasks`)..." | **N/A today** | codegraph's 8 tools are fast synchronous reads; TASK-01 is explicitly deferred (not protocol currency) |
| **SEP-2322** — MRTR pattern, `resultType` field | "All results now carry a required `resultType` field: `\"complete\"`... Clients MUST treat results from earlier-protocol servers that omit the field as `\"complete\"`." | **`resultType` applicable (Phase 3, SPEC-03); the MRTR `input_required` interaction itself deferred (MRTR-01)** | `resultType` is a transport-agnostic MUST; the mid-call-elicitation interaction pattern is new product behavior, not currency work |
| **SEP-2243** — `Mcp-Method`/`Mcp-Name` headers, `x-mcp-header` | "Require standard MCP request headers... on Streamable HTTP POST requests, and add support for custom headers from tool parameters via `x-mcp-header`." | **N/A — HTTP-only** | stdio has no header layer (spec text, quoted in FEATURES.md); `x-mcp-header` is explicitly ignorable on stdio per spec |
| **SEP-2549** — `CacheableResult` (`ttlMs`/`cacheScope`) | "Require `ttlMs` and `cacheScope` fields on results returned by `tools/list`... via a new `CacheableResult` interface." | **Applicable, MUST** (Phase 3, SPEC-04) | Transport-agnostic; `ttlMs: 0`/`cacheScope: "private"` is the correctness-critical value pairing this milestone's decisions record — a long `ttlMs` would be a true statement about the mechanism and a dangerous one about the promise |
| Resource-not-found code `-32002` → `-32602` | "Change resource not found error code from `-32002` to `-32602`..." | **N/A** | codegraph-go implements no `resources` capability |
| **RFC 9207 `iss`** (SEP-2468) | "Authorization servers SHOULD include the `iss` parameter... MCP clients MUST validate..." | **N/A** | stdio subprocess has no OAuth surface (REQUIREMENTS.md Out of Scope) |
| **DCR → CIMD** (PR #2858) | "Deprecate the OAuth 2.0 Dynamic Client Registration Protocol... in favor of Client ID Metadata Documents." | **N/A** | same — no auth surface |
| **SEP-837** — `application_type` in DCR | OAuth redirect URI detail | **N/A** | same |
| **SEP-2352** — credential/issuer binding | OAuth detail | **N/A** | same |
| **SEP-2106** — loosen `inputSchema`/`outputSchema`, `structuredContent` | "Loosen `inputSchema` and `outputSchema` to allow any JSON Schema 2020-12 keywords..." | **Applicable, low-impact, verify-only** | codegraph-go's 8 tool schemas are simple (`mcp.WithString`/`WithNumber`); worth a quick check whether any new keyword capability is wanted, not required |
| **SEP-2577** — deprecate Roots/Sampling/Logging | "These features remain fully functional... but new implementations should not add support for them." | **N/A / already-compliant** | codegraph-go never declared these capabilities; already uses the spec-endorsed replacements (tool-arg paths, no sampling, stderr logging) |
| **SEP-2596** — HTTP+SSE reclassified Deprecated; `includeContext` values | Governance + specific value deprecations | **N/A** | stdio-only, no HTTP+SSE transport, no elicitation `includeContext` usage |
| **SEP-1850** — SEP process formalization | Governance/process only | **N/A** | not a technical requirement |
| **Error code allocation policy** (unnumbered, minor change) | "`-32000`-`-32019` remains implementation-defined... `-32020`-`-32099` reserved for MCP... `UnsupportedProtocolVersion` `-32004` → `-32022`." | **Applicable when Phase 3 lands Modern `_meta` validation** | Phase 1 stays on mark3labs (Legacy, emits neither code); Phase 3 must use `-32022`, not an old `-32004`-style value — maps to SPEC-02 |
| Deterministic `tools/list` ordering (minor change) | "Servers SHOULD return tools from `tools/list` in a deterministic order..." | **Applicable, believed already-compliant, verify via oracle** | `companionNames` (`internal/mcp/server.go:32` — `var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}`) is a fixed slice, and `codegraph_explore` is always registered first; the wire oracle's "call `tools/list` twice, diff order" scenario (D-05) confirms this rather than assuming it |
| OTel `_meta` conventions (SEP-414) | Documents `traceparent`/`tracestate`/`baggage` conventions | **N/A / optional** | No OTel adoption in codegraph-go currently |
| Elicitation `notifications/elicitation/complete` removal | Removed under MRTR | **N/A** | codegraph-go never implemented elicitation |

## Requirement index — which rows map to which v0.3.0 SPEC requirement

| Requirement | Rows it draws on |
|---|---|
| **SPEC-01** — `server/discover` without a prior tool call | SEP-2575 `server/discover` row |
| **SPEC-02** — per-request `_meta` validation, `-32602`/`-32022` | SEP-2575 stateless-core row; error code allocation policy row |
| **SPEC-03** — every result carries `resultType: "complete"` | SEP-2322 row |
| **SPEC-04** — `ttlMs: 0` / `cacheScope: "private"` | SEP-2549 `CacheableResult` row |
| **SPEC-05** — `hasIndex` re-checked per call, not snapshotted | Not a SEP row — this is a codegraph-go-specific consequence of SEP-2575's statelessness, not a spec text itself |
| **SPEC-06** — Legacy clients keep working | SEP-2575 stateless-core row (Legacy compatibility matrix), REQUIREMENTS.md Out of Scope's "never drop Legacy `initialize` support" |
| **SPEC-07** — `server/discover.instructions` carries usage guidance | SEP-2575 `server/discover` row |
| **SPEC-08** — `io.modelcontextprotocol/serverInfo` in tool result `_meta` | Not independently listed above (a `_meta` convention, not its own SEP row); follows from the same stateless-core `_meta` mechanism SEP-2575 introduces |
| **SPEC-09** — `subscriptions/listen` + `notifications/tools/list_changed` | SEP-2575 `subscriptions/listen` row |

## The wire-oracle anti-regeneration trigger set is a floor, not a proof of innocence.

Plan 06's CI guard fires only when a frozen transcript changes together
with (a) `internal/mcp/*.go` or (b) the MCP dependency line in `go.mod`.
CONTEXT.md D-03 locks that narrow trigger set deliberately — it is
specifically "a frozen transcript **plus** (`go.mod`'s MCP dependency **or**
`internal/mcp/*.go`)", not a blanket "no golden changes" gate.

Transcript bytes also legitimately depend on packages the trigger set does
**not** name: `internal/query` (explore ranking output shapes what
`codegraph_explore` returns), `internal/indexer` (what gets indexed shapes
node/edge counts and ordering in tool results), and the tree-sitter
grammars (parse output feeds the indexer). A query-engine change that moves
`handshake-explore.golden` and regenerates it in the same PR would pass the
guard untouched — the guard only ever looks at `internal/mcp/*.go` and
`go.mod`'s MCP line, never at `internal/query` or `internal/indexer`.

That is a **recorded residual risk, not a defect to fix by widening the
trigger set**. Widening it is explicitly out of scope here: split-PR
discipline for `internal/query` and `internal/indexer` changes that touch
frozen transcripts remains a convention, enforced by review, not by the
gate. Do not widen the trigger set to compensate.

## The repo-owned protocol version is an asserted pin in v0.3.0 Phase 1.

`internal/mcp.ProtocolVersion` is **asserted against the negotiated value**,
not injected into negotiation. This is a deliberate consequence of the SDK
this phase still runs on, not an oversight: `mark3labs/mcp-go v0.56.0`
decides the negotiated revision internally, in the unexported
`(*MCPServer).protocolVersion(clientVersion string) string` method
[VERIFIED: `github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1197-1210`],
and exposes no `WithProtocolVersion` server option anywhere in its public
API — there is no supported way to make this SDK negotiate a value this
repo chooses rather than one baked into the library itself.

What this pin buys **today**: a dependency bump that moves the SDK's own
negotiated value — for example a future `mark3labs/mcp-go` release adding a
newer `ValidProtocolVersions` entry — turns `internal/mcp.ProtocolVersion`'s
assertion red immediately, giving this milestone's VRFY-02 guard a spec
anchor that fails loudly on drift instead of silently tracking whatever the
SDK happens to do this week.

What lands in **Phase 2**: the SDK migration (`modelcontextprotocol/go-sdk`)
replaces the backend this pin asserts against with one whose negotiated
protocol revision this codebase can actually supply as an input, not merely
observe as an output. Phase 1's pin is the honest interim state — a repo-
owned literal that watches the current SDK's behavior, not one that
controls it — and this document is where a reader asking "what revision
does codegraph declare, and who decides it" should land.

## Measured pre-migration behavior

**Measured:** 2026-08-05, against `mark3labs/mcp-go v0.56.0` (the pinned
dependency in `go.mod` at measurement time). Source: the six frozen
transcripts under `testdata/wireoracle/transcripts/legacy-*.golden`,
captured by `test/wireoracle/scenarios.go`'s six-era Legacy handshake
baseline (`01-05-PLAN.md` Task 1 checkpoint selection: six-era) and
verified positively by `TestLegacyEraBaselineIsDocumented`
(`test/wireoracle/oracle_test.go`).

Today's server does not reject an unrecognized `protocolVersion` — it
**silently coerces** it, and does the same (via a distinct code path) when
`protocolVersion` is omitted entirely:

| Client offers | Server negotiates | Golden transcript | Behavior |
|---|---|---|---|
| `2025-11-25` | `2025-11-25` | `legacy-2025-11-25.golden` | supported, echoed back |
| `2025-06-18` | `2025-06-18` | `legacy-2025-06-18.golden` | supported, echoed back |
| `2025-03-26` | `2025-03-26` | `legacy-2025-03-26.golden` | supported, echoed back |
| `2024-11-05` | `2024-11-05` | `legacy-2024-11-05.golden` | supported, echoed back |
| `2026-07-28` (unsupported) | `2025-11-25` | `legacy-unsupported-2026-07-28.golden` | **silent coercion to server's own latest — a SUCCESSFUL `initialize` result, no `error` object** |
| *(omitted)* | `2025-03-26` | `legacy-omitted-version.golden` | **silent coercion to the server's older backwards-compat default — also a SUCCESS, a structurally distinct branch from the unsupported-version case above** |

Both silent-coercion rows are frozen and documented as such per RESEARCH
Pitfall 1: a green oracle on either of these two scenarios is evidence of
today's Legacy tolerance, never evidence that the server validates
versions.

**Contrast with Phase 3's obligation:** the *Error code allocation policy*
row in the SEP-by-SEP table above records that `UnsupportedProtocolVersion`
moves to `-32022` under the `2026-07-28` revision. Phase 1 (this
measurement) stays on the Legacy `mark3labs` backend, which emits neither
`-32004` nor `-32022` for an unsupported version — it emits no error at
all. Phase 3, once it lands Modern `_meta` validation on the migrated SDK,
must reject an unsupported version explicitly with `-32022`; this table's
silent-coercion baseline is what Phase 3's SPEC-02/SPEC-06 tests compare
against to prove that transition actually happened, rather than assuming
it from the SDK's own documentation.

## Sources

- [CITED: `modelcontextprotocol.io/specification/2026-07-28/changelog`, fetched 2026-08-04] — every "what it does" cell above
- `github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1197-1210` [VERIFIED, module cache] — the asserted-pin section
- `.planning/REQUIREMENTS.md` § Out of Scope — Streamable HTTP, authorization work, and Legacy-support exclusions this table agrees with
- `.planning/phases/01-protocol-scoping-the-sdk-independent-wire-oracle/01-RESEARCH.md` § "SEP-by-SEP Applicability Table" and § "PITFALLS Pitfall 1" — the researched source of this table's content
- `testdata/wireoracle/transcripts/legacy-*.golden` [VERIFIED, six frozen transcripts, measured 2026-08-05] — the source of the "Measured pre-migration behavior" table above
