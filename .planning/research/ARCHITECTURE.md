# Architecture Research — MCP `2026-07-28` Protocol Currency

**Domain:** MCP server integration architecture (stdio, tools-only), codegraph-go v0.3.0
**Researched:** 2026-08-03 (corrected 2026-08-03 — see amendment note below)
**Confidence:** MEDIUM overall — code-derived findings (file/symbol enumeration) are HIGH confidence (read directly); protocol-spec findings are sourced from the official `modelcontextprotocol.io` spec/changelog and blog, which is the spec's own primary source, but the research seam's generic `classify-confidence` tool has no tier above LOW for raw `websearch`/`webfetch` providers — see Sources for the honest breakdown.

> **Correction (2026-08-03, same day):** the original version of this file understated `modelcontextprotocol/go-sdk`'s maturity — it characterized `v1.7.0-pre.1` (the first pre-release) as the current state of the art. The coordinator verified directly against the Go module proxy (`go list -m -json ... @latest`) and the module source (`go-sdk@v1.7.0/mcp/shared.go`) that **`v1.7.0` is a STABLE release, published 2026-07-27T15:20:53Z — the day before the spec's public announcement** — shipping five-era protocol negotiation (`2026-07-28`, `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`) and the SEP-2575-numbered `CodeUnsupportedProtocolVersion = -32022`. This is now corrected throughout, most consequentially in Q4 and Q5. The mark3labs finding (`mcp-go@v0.57.0` pins `LATEST_PROTOCOL_VERSION = "2025-11-25"`, no `2026-07-28` support) was independently re-verified and stands unchanged.

This is a **subsequent-milestone** research file. It does not re-litigate the stack (see `.claude/CLAUDE.md`'s Technology Stack section, already shipped) — it integrates MCP `2026-07-28` into the existing, shipped Go architecture described in `internal/mcp/`, `internal/cli/serve.go`, and `test/integration/`.

## Standard Architecture — Current State (as shipped, v1.0)

### System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│  Agent client (Claude Code, Cursor, ...) — spawns ONE subprocess      │
│  per session via the stdio invocation internal/agents wrote           │
└───────────────────────────────┬────────────────────────────────────┘
                                 │ stdio (stdin/stdout pipes)
┌────────────────────────────────▼───────────────────────────────────┐
│  internal/cli/serve.go — newServeCmd().RunE                         │
│   1. resolveStartPath(path)              → start                    │
│   2. serveServerPaths(start)             → repoPath, hasIndex        │
│   3. indexer.Sync(repoPath, storeDir,…)  → reconnect reconcile (D-06)│
│   4. serveWatchStart(repoPath, hasIndex,…) → background watcher      │
│   5. mcp.BuildServer(hasIndex, allowlist, repoPath, start) → *server.MCPServer │
│   6. server.ServeStdio(s)                → blocks on stdin/stdout    │
└────────────────────────────────┬───────────────────────────────────┘
                                 │ *server.MCPServer (mark3labs concrete type)
┌────────────────────────────────▼───────────────────────────────────┐
│  internal/mcp (server.go + tools.go) — D-08 seam                    │
│   BuildServer: registers codegraph_explore (always) + 7 companion   │
│   tools (allowlist-gated), closes over ONE gitmeta.CachingDetector  │
│   per server, and over (repoPath, startPath) — both process-start-  │
│   time constants baked into every handler closure                   │
│   Each handler: resolvePath → confineToRepoRoot (CR-02) →           │
│   query.OpenAt (FRESH snapshot, D-02/D-08b) → Engine method →       │
│   Render*Markdown → mcp.NewToolResultText                            │
└────────────────────────────────┬───────────────────────────────────┘
                                 │ imports only internal/query
┌────────────────────────────────▼───────────────────────────────────┐
│  internal/query.Engine — shared read-only engine (CLI + MCP)         │
│  → internal/graphstore (Pebble) — single-writer, snapshot reads      │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Current implementation |
|-----------|----------------|-------------------------|
| `internal/cli/serve.go` (`newServeCmd`, `serveServerPaths`, `serveWatchStart`) | Process bootstrap: resolve repo, reconcile offline changes, start watcher, construct server, block on transport | Owns the ONE call to `mcp.BuildServer` and the ONE call to `server.ServeStdio` — both mark3labs-typed |
| `internal/mcp/server.go` (`BuildServer`, `ParseAllowlist`, `WarnUnknownToolsTo`) | Startup-time conditional tool registration; one `gitmeta.CachingDetector` per server | Returns `*server.MCPServer` (mark3labs concrete type) — see Q2 leak finding |
| `internal/mcp/tools.go` (`exploreHandler`, `companionHandler`, `openEngine`, `confineToRepoRoot`, `resolvePath`) | Per-call arg parsing, path confinement, fresh-engine delegation, markdown rendering | Every handler is `server.ToolHandlerFunc` (mark3labs type); tool schemas built with `mcp.NewTool`/`mcp.With*` (mark3labs builder DSL) |
| `internal/query.Engine` | Read-only query surface shared byte-for-byte by CLI and MCP | Untouched by this milestone — the seam this milestone must not disturb |
| `test/integration/*_test.go`, `internal/mcp/*_test.go`, `testdata/golden/golden_parity_test.go` | Behavioral + protocol verification | Nearly all use `mcpclient` (mark3labs client) as the driving client — see Q3 |

## Q1 — Team Scale Strategic Assessment (read-out, not a build order)

**Verdict: YES, materially more tractable — for a specific, narrow reason, not a blanket one.**

### What actually changes

The prior (pre-`2026-07-28`) stateful protocol tied a client to a server-held, in-memory session established during `initialize`/`notifications/initialized` and addressed by `Mcp-Session-Id`. Any multi-instance deployment of that protocol needed one of two things codegraph-go's deferred milestone-2 never designed: **sticky routing** (an LB pins a session to the instance that negotiated it) or a **shared session store** (any instance can serve any session id by reading shared state). `2026-07-28` removes the session concept from the protocol entirely (SEP-2567, SEP-2575): `initialize`/`initialized` is gone, every request self-describes via `_meta` (`io.modelcontextprotocol/protocolVersion`, `io.modelcontextprotocol/clientCapabilities`, `io.modelcontextprotocol/clientInfo`), and `server/discover` (a servers-MUST-implement RPC advertising supported versions/capabilities/identity) is answerable identically by any instance. The practical result: **a plain round-robin LB with zero shared session infrastructure is now a valid production topology for an MCP server** — an entire category of infrastructure work milestone-2 would otherwise have had to build is simply gone.

### Why codegraph-go specifically benefits more than a typical MCP server would

Three decisions made in v1.0 for LOCAL, single-process reasons turn out to already match the shape `2026-07-28` recommends for stateless cross-call state:

1. **`path` is already an explicit per-call tool argument, not implicit session state.** Every one of the 8 tool schemas (`tools.go` `exploreTool`/`companionTool`) declares `mcp.WithString("path", ...)`, and every handler resolves it per-call via `resolvePath`/`confineToRepoRoot` (CR-01/CR-02) rather than trusting a session-bound value. This is *exactly* the "explicit, server-minted handle passed as ordinary tool arguments" pattern the `2026-07-28` blog names as the replacement for protocol-level session state. codegraph-go didn't need to invent this for team-scale readiness — it already exists, because CR-02's trust boundary ("an MCP client may be an AI agent processing attacker-influenced content") demanded it independently.
2. **Every handler opens a FRESH `query.Engine` snapshot per call** (`openEngine`, D-02/D-08b — deliberately never a cached engine held across calls). There is no session-scoped read cache to reconcile across instances; each call is already independently satisfiable by any process that can reach the right physical store.
3. **`gitmeta.CachingDetector` is the only genuinely session-scoped state today**, and it is explicitly documented as "one per SERVER... bounds [git subprocess] cost to once per (startPath, indexRoot) pair for this server's entire lifetime" — a pure performance cache, not a correctness dependency. It generalizes to a keyed cache (by tenant/repo) sitting above per-request server construction; nothing about its existence forecloses a stateless design.

### What a remote/team codegraph MCP server would look like under `2026-07-28` (future milestone, sketch only)

| Concern | Current (stdio, 1 repo/process) | Remote/team shape under `2026-07-28` |
|---|---|---|
| Transport | stdio, one subprocess per session | Streamable HTTP (mandatory `Mcp-Method`/`Mcp-Name` routing headers per SEP-2243; no `Mcp-Session-Id`; no SSE resumability — a broken stream means the client re-issues as a new request, which is trivially safe for codegraph-go's read-only, idempotent queries) |
| Auth | None — process-spawn trust boundary | OAuth 2.1 resource-server pattern; **Client ID Metadata Documents (CIMD)** preferred over per-AS Dynamic Client Registration (good fit for an 8-agent roster — each client publishes one CIMD document instead of registering per install); **RFC 9207** `iss` validation on the client side; **RFC 8707** Resource Indicators binding a token to this specific tenant's codegraph server so a token can't be replayed against a different tenant's index |
| Which repo/index a request targets | Baked into `BuildServer`'s closure at process construction (`repoPath`, `startPath`) | Must become **request-scoped**, resolved from (authenticated tenant/token audience) × (an identifier analogous to today's `path` argument, reinterpreted as a repo/org id, not a raw filesystem path) — looked up through a registry that resolves to a physical index location (local disk cache fed by CI-distributed index artifacts — the already-deferred backlog item) |
| Tool catalog construction | `BuildServer` runs ONCE at process start; `hasIndex`/`allowlist` are startup-time snapshots | Must move from constructor-time to **per-request-resolved** (a given HTTP request's tenant may or may not have an index yet) — this is the one clear, bounded refactor forced by statelessness, not by the transport change alone |
| `gitmeta.CachingDetector` | One per server process | Promote to a cache keyed by (tenant, repo), held by a router/connection-pool layer above per-request server construction |
| Store access | `query.OpenAt` opens a LOCAL Pebble path per call | Unchanged in shape — Pebble was chosen specifically for concurrent-read/snapshot semantics under exactly this kind of many-reader load; `query.OpenAt` already just takes a resolved path, so the resolution layer is new, not `query.OpenAt` itself |
| `tools/list` caching | N/A today (no `ttlMs`/`cacheScope` field exists in the current SDK) | `cacheScope` must be `"private"` (catalog varies per tenant/repo — never safe for a shared intermediary to cache across tenants) with a short `ttlMs` (catalog can change the moment CI publishes a tenant's first index) — contrast with the stdio case (Q4) where the registered tool set is mechanically fixed for a process's life, but see Q4's reconciliation on why "mechanically fixed" is not the same as "safe to advertise as long-lived" |

### Does anything in the CURRENT design foreclose this path?

**No.** The one real structural gap is `mcp.BuildServer(hasIndex, allowlist, repoPath, startPath)` being a constructor-time-only API — every one of these four parameters would need a request-scoped resolution path for a multi-tenant server. That is a bounded, already-anticipated refactor (the standing constraint literally says "must accommodate milestone-2 team features... without a rewrite") — not a rewrite, because the actual query-and-render pipeline underneath (`openEngine` → `Engine` method → `Render*Markdown`) needs no change at all. The auth/tenant-resolution/CI-index-distribution layers are entirely NEW components with no existing analog to conflict with.

**Recommendation:** Record this assessment in `PROJECT.md`'s Key Decisions or Context section as the deferred milestone-2 read-out (a decision-record entry, not a phase). Do not open an HTTP transport in this milestone (explicitly out of scope per the quality gate) — but the phase that eventually does should budget for exactly the four rows above, in this order: (1) promote `BuildServer`'s four parameters to per-request resolution behind an interface (this is also what Q2's narrow seam recommendation sets up for free), (2) add the auth/tenant-resolution layer, (3) add the CI-index-distribution/pull-cache layer feeding `query.OpenAt`, (4) add the Streamable HTTP transport itself last, once (1)-(3) exist and can be exercised over stdio in a test harness first.

## Q2 — The SDK Seam

### Every file/symbol touching mark3labs types

**Production code (non-test):**

| File | Symbols |
|---|---|
| `internal/mcp/server.go` | `server.NewMCPServer`, `server.WithToolCapabilities`, `server.MCPServer` (return type of `BuildServer`), `server.ToolHandlerFunc` (parameter type of `exploreHandler`/`companionHandler`), `s.AddTool` |
| `internal/mcp/tools.go` | `mcp.CallToolRequest`, `mcp.CallToolResult`, `mcp.NewTool`, `mcp.WithDescription`, `mcp.WithString`, `mcp.WithNumber`, `mcp.Required`, `mcp.NewToolResultText`, `mcp.NewToolResultError`, `server.ToolHandlerFunc` |
| `internal/cli/serve.go` | `server.ServeStdio(s)` — takes the `*server.MCPServer` `BuildServer` returned |

**Test code (all import `mcpclient "github.com/mark3labs/mcp-go/client"` and/or `mcp-go/mcp`, using the SDK's client as the driving/asserting client):**

`internal/mcp/server_test.go`, `internal/mcp/reconnect_test.go`, `internal/mcp/markdown_test.go`, `internal/mcp/tools_schema_drift_test.go` (AST-parses source, imports `mcp-go/mcp` incidentally), `test/integration/mcp_stdout_purity_test.go` (imports only for the `mcp.LATEST_PROTOCOL_VERSION` constant — the actual frame encode/decode is 100% hand-rolled, see Q3), `test/integration/watch_default_test.go`, `test/integration/watch_live_sync_test.go`, `test/integration/worktree_notice_test.go`, `testdata/golden/golden_parity_test.go`.

### Is `internal/mcp` a genuine seam?

**Partially.** The package boundary correctly holds for the *query* dependency (`internal/mcp` imports only `internal/query`, never `internal/graphstore` directly — the existing archtest-enforced boundary is untouched by this analysis). But the SDK itself leaks across the `internal/mcp` → `internal/cli` boundary: `BuildServer` returns `*server.MCPServer`, a concrete mark3labs struct, and `internal/cli/serve.go` must import `github.com/mark3labs/mcp-go/server` directly just to call `server.ServeStdio(s)` on that returned value. That is a real, avoidable leak — `internal/cli` should never need to name an SDK type.

The much larger leak is in the **test surface**: every test that wants to drive the server end-to-end (in-process or real-stdio) constructs an `mcpclient.Client` and calls `mcpclient.NewInProcessClient`/`NewStdioMCPClientWithOptions`/`c.CallTool` — meaning mark3labs' client implementation is the de facto verification oracle across nearly the entire test suite, not an implementation detail hidden behind `internal/mcp`.

### Narrowest interface worth introducing

```go
// internal/mcp
type Server interface {
    ServeStdio() error
}
```

`BuildServer` would return this interface (backed today by a thin wrapper holding the `*server.MCPServer`), and `internal/cli/serve.go` would depend only on `internal/mcp.Server` — never importing mark3labs directly. This is small, surgical, and worth doing **now, independent of the SDK decision**: it removes the one real non-test leak for near-zero cost and closes exactly the gap Q1's Team Scale refactor will need anyway (a `BuildServer`-shaped construction point that `internal/cli` treats opaquely).

**What this narrow interface does NOT buy you, and it would be dishonest to claim otherwise:**

1. **Tool schema construction is not abstracted, and should not be.** `tools.go`'s `mcp.NewTool`/`mcp.With*` builder calls are SDK-specific by nature — building a codegraph-owned schema DSL to hide this would itself be a shadow SDK, disproportionate for 8 tools. An SDK swap means **porting `tools.go` directly**, not swapping an implementation behind an interface.
2. **It does nothing for the test-surface leak.** The interface only covers the one production call site. The verification harness (Q3) is the actual fix for test-side SDK coupling, and it is a materially larger effort than the interface itself.

**Honest recommendation:** introduce the `internal/mcp.Server` interface now (cheap, unambiguously worth it, and it is literally the same seam Q1's Team Scale refactor needs). Do **not** attempt to make the SDK swap "just a backend change" at the tool-schema level — budget the swap as a direct, file-by-file port of `tools.go` + `server.go`, plus a full rewrite of every test's client-side calls (Q3 exists to make that rewrite verifiable without circularity).

## Q3 — Real-Client Verification Harness

### What already exists and why it matters

`test/integration/mcp_stdout_purity_test.go` (`TestServeMCPStdoutIsPureJSONRPC`) already IS the SDK-independent raw-stdio harness the project's history required in v1.0 Phase 4: it spawns the real binary via plain `os/exec`, writes hand-framed JSON-RPC requests (`map[string]any` → `json.Marshal`) directly to `cmd.StdinPipe()`, and reads `cmd.StdoutPipe()` through a raw `bufio.Scanner`, decoding each line into an **anonymous struct** (`jsonrpc`/`id`/`error` fields only) — never through `mcpclient`. Its only mark3labs import is `mcp.LATEST_PROTOCOL_VERSION`, used solely to embed a valid version string in a hand-built map; that one reference is trivially replaceable with a literal string constant. This file is the correct precedent to generalize, not a new pattern to invent.

### Harness design for `2026-07-28`

**Location:** `test/integration/` — extend the existing raw-stdio pattern rather than adding a parallel mechanism. Concretely: factor the hand-framing/raw-scanning helpers (`writeLine`, the scanner-goroutine-plus-channel pattern) out of `mcp_stdout_purity_test.go` into a small shared internal helper (e.g. `test/integration/rawmcp_helper_test.go`) so new assertions don't re-implement the raw reader, and add a new test file for the `2026-07-28`-specific assertions (e.g. `test/integration/mcp_protocol_2026_test.go`).

**What it must assert**, each as a literal byte-level check on the raw JSON frame — never via any SDK's typed result:

1. **`server/discover` is implemented** and returns supported protocol versions + capabilities + identity — spec-mandatory (`SEP-2575`). Send the raw request, parse the raw response into a local anonymous struct, assert the expected fields are present.
2. **Statelessness is real, not just documented:** issue a `tools/call` (or `tools/list`) as the very first request on a fresh connection, with `_meta["io.modelcontextprotocol/protocolVersion"]` and `_meta["io.modelcontextprotocol/clientCapabilities"]` set, and NO prior `initialize`/`notifications/initialized` exchange — assert a normal, non-error response. This is the single most important new assertion: it is the one behavior that, if wrong, silently breaks every client that has moved to the new spec (the "quiet failure mode" `PROJECT.md` names).
3. **`tools/list` carries `ttlMs`/`cacheScope`** (`SEP-2549`) — raw JSON field-presence check.
4. **Version-mismatch handling:** send a request with an unsupported/malformed protocol version in `_meta`, assert the response is a JSON-RPC error with the renumbered `UnsupportedProtocolVersion` code (`-32022` under the new error-code allocation policy — independently confirmed as `CodeUnsupportedProtocolVersion = -32022` in `go-sdk@v1.7.0/mcp/shared.go:394`, not the old `-32004`).
5. **Deterministic `tools/list` ordering** (SHOULD, minor change #3): call `tools/list` twice, diff the raw tool-name order.
6. **Stdout purity, re-run against whichever SDK is in place** — this is the existing `TestServeMCPStdoutIsPureJSONRPC` test, kept as a permanent regression guard for the exact historical failure mode (an SDK's own client silently skipping malformed lines).

### Avoiding circularity — the actual answer to "which oracle"

Three candidate oracles were considered:

- **Captured real-client traffic** (record actual Claude Code/Cursor/etc. byte streams against `2026-07-28`, replay as fixtures): the highest-fidelity option, but `2026-07-28` is brand new (release-candidate as of this research date) — no real fixtures exist yet for most of the 8-agent roster. Defer; revisit once agent clients actually ship `2026-07-28` support to capture against.
- **A hand-rolled JSON-RPC client** (generalizing `mcp_stdout_purity_test.go`'s existing pattern): the primary mechanism, and the one with zero new infrastructure cost — it is precisely what the project already proved necessary and already has running.
- **A conformance corpus validated against the official spec's own JSON Schema** (vendor the `2026-07-28` `schema.json` published at `modelcontextprotocol.io`, validate captured frames against it with a generic, oracle-neutral JSON-Schema validator such as `github.com/santhosh-tekuri/jsonschema`): recommended as a **complementary second layer**, not a replacement for the hand-rolled client. This resolves the philosophical bind directly — the official schema *is* the spec itself, not a competing SDK's interpretation of it, so validating against it is not circular the way asserting against mark3labs' or the official SDK's own client would be. Use it for the fields that matter most (`resultType`, `ttlMs`, `cacheScope`, the `_meta` keys) where a hand-written struct-shape check would otherwise have to duplicate the schema by hand.

**The harness must exist and pass against the CURRENT (mark3labs) server before any SDK code changes** — this both discharges backlog 999.6's empirical-audit requirement (run it against what ships today to see exactly what `2026-07-28` behaviors are missing) and gives the SDK-swap phase (if any) a fixed, SDK-independent acceptance gate: the same test file, unmodified, must stay green against whatever server exists after the swap.

## Q4 — Dynamic Catalog vs. Cacheable Lists

### The registered tool set is mechanically fixed within a process — but that is not the same claim as "safe to cache"

Re-reading `internal/cli/serve.go` and `internal/mcp/server.go` closely: `hasIndex` and `allowlist` are **both computed once, at process startup**, and closed over for the server's entire lifetime. `serveServerPaths`'s doc comment states this explicitly for `hasIndex` (IN-09: "An index created mid-session... does NOT retroactively start the watcher... auto-sync begins on the next serve --mcp session"), and `CODEGRAPH_MCP_TOOLS` is read once via `os.Getenv` in `newServeCmd`'s `RunE`. `BuildServer` calls `s.AddTool` only during its own single invocation and never again — mechanically, **the registered tool set cannot change for the life of one `serve --mcp` process**, because nothing in the codebase calls `AddTool`/removes a tool afterward.

### Reconciling with the FEATURES researcher's finding — which reading is right

The FEATURES research independently found the same underlying mechanism (`hasIndex` is a construction-time snapshot) and concluded that `ttlMs: 0` would be a lie unless `tools/list` re-checks `hasIndex` per call, because a user can run `codegraph init` while a long-lived MCP server is already running. **On reconciliation, the FEATURES reading is the one that should drive the actual value chosen, and my original "large ttlMs is accurate" framing was the wrong product conclusion even though it was mechanically true.** Here is why both are correct at their own layer, and why that resolves in FEATURES' favor:

- My claim is a true statement about the **mechanism**: given today's register-once code, the tool set genuinely will not change without a process restart, so a long `ttlMs` would not be contradicted by any event this server can produce.
- FEATURES' claim is the correct statement about **what the field is for**: `ttlMs` is a promise to the *client* that it can stop checking and trust the cached list. The scenario that breaks this is exactly the one FEATURES names — a server started before `.codegraph/` existed (`hasIndex=false`, zero tools ever registered, including `codegraph_explore`) stays permanently toolless for that connection's whole life even after `codegraph init` runs, and a client that has cached `tools/list` for a long `ttlMs` now has **less** chance of ever re-checking and recovering than a client that was never told it could cache at all. Advertising a long `ttlMs` doesn't just describe the existing MCP-03 gap — it actively encourages clients to stop working around it.

**Conclusion: `ttlMs` should be short/conservative (effectively near-zero) as the honest default under the current mechanism**, not the large value the mechanical fact alone would justify. `cacheScope` should still be `"private"` (unchanged reasoning — this server's catalog is specific to its own `repoPath`/`allowlist` configuration, and Q1's remote-server sketch shows why no shared intermediary should ever conflate two tenants' catalogs). The more complete fix — making `hasIndex` genuinely dynamic (re-evaluated per `tools/list` call rather than snapshotted once) — is a legitimate scope candidate for this milestone precisely because the SDK-swap work (if adopted) already touches this exact code path in `internal/mcp/server.go`; landing it removes the tension entirely rather than merely picking a conservative `ttlMs` around it. This should be raised explicitly when this milestone's phases are planned, not silently deferred.

### Implementation caveat (corrected)

mark3labs `mcp-go@v0.57.0` (protocol `2025-11-25`) has no `ttlMs`/`cacheScope` field on its `ListToolsResult` type at all today — this is a `2026-07-28`-only `CacheableResult` interface (`SEP-2549`). **Landing this concretely is gated on the SDK decision (Q5), but that gate is far less restrictive than originally stated:** the official `modelcontextprotocol/go-sdk` shipped `2026-07-28` support, including `CacheableResult`-shaped fields, in the **stable** `v1.7.0` release (published 2026-07-27) — it is available today, not pending. The gate is the SDK *decision*, not SDK *availability*. The **impact assessment can and should document the target values now** (near-zero `ttlMs`, `"private"` `cacheScope`, per the reconciliation above); the code lands as part of whichever SDK-swap phase Q5 produces.

### Interaction with `internal/mcp/reconnect_test.go`

`TestReconnectReconcile` covers a **completely different staleness axis** and must not be conflated with catalog caching. It pins D-06/SYNC-03: the `indexer.Sync` reconcile pass `serve.go`'s `RunE` runs before `BuildServer` is even called, catching *content* drift (a file edited while no watcher/daemon was running) so the first `codegraph_explore` call after a reconnect reads current graph *data*. Nothing about `ttlMs`/`cacheScope` touches this path — the catalog (which tools exist) and the graph content (what those tools return) are orthogonal freshness concerns, and no future change should let a `tools/list` cache-invalidation mechanism reach into the graph store, or vice versa. A `tools/list` handler must never need to consult `query.Engine`/`graphstore` to decide `ttlMs` — `hasIndex`/`allowlist` are already known, static-for-the-process facts by the time `BuildServer` runs (or would be re-evaluated directly from the filesystem check `serveServerPaths` already performs, if `hasIndex` is made dynamic per the reconciliation above — either way, no dependency on the graph store itself).

## Q5 — Suggested Build Order

Dependencies, not narrative convenience, drive this ordering. The standing constraint (verification harness precedes any SDK swap) is structural, not sequencing preference: Phase C's decision cannot be trusted, and Phase D's swap cannot be verified as non-regressive, without Phase B existing first.

**Corrected input to this ordering:** `mark3labs/mcp-go@v0.57.0` pins `LATEST_PROTOCOL_VERSION = "2025-11-25"` with no `2026-07-28` support and no announced timeline (unchanged finding). `modelcontextprotocol/go-sdk@v1.7.0` is a **stable** release (published 2026-07-27, one day before the spec's public announcement) shipping five-era negotiation (`2026-07-28` down to `2024-11-05`) and the SEP-2575 error codes out of the box. This is a materially different starting position for Phase C than "no SDK is ready yet," and it changes how much weight the dated-defer branch can honestly carry.

```
Phase A: MCP 2026-07-28 impact assessment (999.6)
  │  Research + document what reaches a stdio, tools-only server;
  │  audit across the 8-agent roster; produce the Team Scale read-out (Q1).
  │  Depends on: nothing new (this research file is largely Phase A's output).
  ▼
Phase B: Real-client verification harness (Q3)
  │  Extend test/integration's raw-stdio pattern with 2026-07-28 assertions
  │  (server/discover, stateless-first-request, ttlMs/cacheScope presence,
  │  renumbered error codes, deterministic tools/list order).
  │  MUST run green against the CURRENT mark3labs server first — this is
  │  the empirical half of Phase A's audit, and the fixed acceptance gate
  │  for whatever Phase C decides.
  │  Depends on: Phase A (to know exactly which behaviors need assertions).
  ▼
Phase C: SDK decision (mark3labs v0.57.0 vs modelcontextprotocol/go-sdk v1.7.0)
  │  A recorded decision (adopt-now / dated-defer) discharging the standing
  │  re-evaluation commitment in .claude/CLAUDE.md's Alternatives table.
  │  mark3labs does not support 2026-07-28 today and has no announced
  │  timeline; the official SDK's 2026-07-28 support is STABLE (v1.7.0,
  │  released 2026-07-27) with built-in five-era negotiation — the
  │  "wait for the SDK to mature" rationale for deferring no longer
  │  applies (see the dated-defer note below).
  │  Depends on: Phase A (impact) + Phase B (a harness that can actually
  │  prove whichever choice is made).
  ▼
        ┌───────────────────────┴────────────────────────┐
        ▼                                                 ▼
Phase D: SDK swap (if ADOPT)                    Phase D': Dated defer (if DEFER)
  Port internal/mcp/tools.go + server.go to        WEAKENED by the corrected
  the new SDK's API. Because go-sdk's own          finding — see note below.
  supportedProtocolVersions/negotiatedMe-          Record the decision with the
  chanism already implements five-era              spec's 12-month minimum
  negotiation, this phase does NOT need to         deprecation window named
  build any bespoke dual-era compatibility          explicitly. No code change
  shim of its own — that work is inherited          required beyond the decision
  from the SDK, not built by this project.          record itself.
  Fix the Q2 leak at the same call site being
  touched anyway (internal/mcp.Server interface
  into serve.go). Replace mcp.LATEST_PROTOCOL_
  VERSION with an explicit literal asserted
  version (wire behavior stops moving silently
  on a dependency bump). Migrate every test call
  site (server_test.go, markdown_test.go,
  reconnect_test.go, golden_parity_test.go,
  watch_*_test.go, worktree_notice_test.go,
  tools_schema_drift_test.go) off mark3labs'
  client. Land the near-zero ttlMs / "private"
  cacheScope values from Q4 once the new SDK's
  CacheableResult fields are wired up — consider
  also making hasIndex dynamic per Q4's
  reconciliation while this file is already open.
  Acceptance gate: Phase B's harness, UNMODIFIED,
  stays green against the new server.
  Depends on: Phase C = adopt, Phase B (harness).
        └───────────────────────┬────────────────────────┘
                                 ▼
Phase E: Tool-modfile vulnerability scanning (999.3)
  │  Closes the ~400-module credentialed-CI-tooling gap over whichever
  │  dependency closure Phase C/D produced (new SDK module, or an
  │  unchanged mark3labs closure on defer).
  │  Depends on: Phase C (needs the final dependency set to be meaningful —
  │  running it before C is decided means re-running it after).

Phase F: Daemon test-seam fixes (#13 getppid race, #17 watchdog flakes)
  │  Independent bug fixes; PROJECT.md notes they sit "on the substrate
  │  this milestone modifies." Bundle with Phase D if adopting (both touch
  │  serve.go-adjacent code, reducing churn on the same files); otherwise
  │  schedule standalone. No hard dependency on A-D.

Phase G: GoReleaser pin reconciliation (ci.yml v2.17.1 vs release.yml v2.17.0)
  │  Pure housekeeping. No dependency on anything above — schedule
  │  wherever convenient (first or last).
```

**Critical path:** A → B → C → (D or D′) → E. F and G float freely; F is cheapest to bundle with D when D happens.

**On the weakened dated-defer branch (D′):** the original framing implicitly justified deferring as "waiting for tooling to mature" — that justification is now false; a stable, spec-authored, multi-era-capable SDK is available today. The only argument for D′ that survives the correction is **production-track-record caution**: `v1.7.0` is roughly a week old as of this research (published 2026-07-27), versus mark3labs' longer field history on a now-superseded protocol revision. If Phase C still lands on defer, the decision record must name the actual tradeoff honestly — "we chose a longer track record on a stale protocol revision over a newer stable release of the current one, and we accept the 12-month-window clock this starts" — rather than "no SDK supports this yet," which is no longer true. Given `go-sdk` also eliminates the need for this project to build its own dual-era negotiation logic (Phase D inherits `supportedProtocolVersions`/`negotiatedVersion` for free), the balance of evidence assembled in this research favors adopt over defer more than the original build order implied — though Phase C remains the actual decision point, not this document.

## Anti-Patterns to Avoid in This Milestone

### Anti-Pattern 1: Abstracting the tool-schema builder DSL "for portability"

**What people do:** wrap `mcp.NewTool`/`mcp.With*` in a codegraph-owned schema builder so an SDK swap "only touches one file."
**Why it's wrong:** for 8 tools, this is a shadow SDK — more code to maintain than the port it's meant to avoid, and it still doesn't solve the test-surface coupling (Q3), which is the actual expensive part of a swap.
**Instead:** treat `tools.go`'s schema definitions as a direct-port surface. Spend the abstraction budget on the `internal/mcp.Server` interface at the `ServeStdio` call site instead (Q2) — that one is cheap and genuinely reusable for Team Scale.

### Anti-Pattern 2: Validating the new SDK with the new SDK's own client

**What people do:** after a swap, write "it works" tests using the new SDK's client library.
**Why it's wrong:** this is exactly the failure mode `PROJECT.md` names as already proven in v1.0 Phase 4 — an SDK's client can silently skip or misrepresent malformed wire output, so a test built on it can pass while the actual bytes on stdout are wrong.
**Instead:** the Phase B raw-stdio/JSON-Schema harness (Q3) is the only trusted oracle, and it must predate and outlive any specific SDK choice.

### Anti-Pattern 3: Conflating catalog cache-control with content-freshness reconciliation

**What people do:** wire `ttlMs`/`cacheScope` invalidation to the same reconcile path that keeps graph content fresh (`indexer.Sync`, the reconnect path `reconnect_test.go` covers).
**Why it's wrong:** they are orthogonal (Q4) — the registered tool set is mechanically fixed per-process; content freshness is a live, per-call concern. Coupling them adds a graph-store dependency to `tools/list` construction for no correctness benefit and makes the two staleness models harder to reason about independently.
**Instead:** keep `ttlMs`/`cacheScope` decided from already-known `hasIndex`/`allowlist` values (static today, or re-evaluated directly against the filesystem if `hasIndex` becomes dynamic per Q4) — no new dependency on `query.Engine`.

## Integration Points

### Internal Boundaries

| Boundary | Current coupling | Notes |
|---|---|---|
| `internal/cli/serve.go` ↔ `internal/mcp` | `internal/cli` imports `mark3labs/mcp-go/server` directly for `server.ServeStdio(s)` | Real leak (Q2) — fixable now with a narrow `internal/mcp.Server` interface, independent of the SDK decision |
| `internal/mcp` ↔ `internal/query` | Clean — `internal/mcp` never imports `internal/graphstore` directly (archtest-enforced) | Untouched by this milestone; do not let any MCP protocol work weaken this |
| `internal/mcp` (production) ↔ test suite | Nearly every test drives the server via `mcpclient` (mark3labs) | The actual expensive coupling; Phase B's raw-stdio harness is the fix, not a production-code interface |
| `internal/agents` ↔ MCP transport | None at all — `internal/agents` only writes a `serve --mcp` stdio invocation string into each agent's config; it never imports `mcp-go` | Confirms the 8-agent installer surface is fully decoupled from the SDK question — an SDK swap changes nothing here |

## Sources

- `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/reconnect_test.go`, `internal/mcp/server_test.go`, `internal/mcp/markdown_test.go`, `internal/mcp/tools_schema_drift_test.go`, `internal/cli/serve.go`, `test/integration/mcp_stdout_purity_test.go`, `test/integration/watch_default_test.go`, `test/integration/watch_live_sync_test.go`, `test/integration/worktree_notice_test.go`, `testdata/golden/golden_parity_test.go`, `internal/agents/*.go` — read directly, HIGH confidence (primary source, this repository).
- `.planning/PROJECT.md` — read directly, HIGH confidence (primary source, this repository).
- [`https://modelcontextprotocol.io/specification/2026-07-28/changelog`](https://modelcontextprotocol.io/specification/2026-07-28/changelog) — official spec changelog, fetched directly. The research seam's `classify-confidence` tool has no tier above LOW for the generic `webfetch` provider id, but this is the specification's own primary-source document (SEP-2567, SEP-2575, SEP-2549, SEP-2243, SEP-2468, SEP-2596 all cited directly from this page) — treat the *content* as authoritative, the *tier label* as a tooling gap, not a signal about the source's reliability.
- [`https://blog.modelcontextprotocol.io/posts/2026-07-28/`](https://blog.modelcontextprotocol.io/posts/2026-07-28/) — official MCP blog announcement, fetched directly. Same tooling-tier caveat as above; same primary-source status.
- **`go list -m -json github.com/modelcontextprotocol/go-sdk@latest`** (module proxy) and **`go-sdk@v1.7.0/mcp/shared.go:45-65,68+,394`** (module source) — verified directly by the coordinator, not by this research pass originally; this is the correction basis for the `v1.7.0` stable-release, five-era-negotiation, and `CodeUnsupportedProtocolVersion = -32022` findings now reflected throughout. HIGH confidence — primary source (module proxy + actual dependency source code), the same evidentiary standard as this document's own repository-code findings, and strictly better sourcing than the WebSearch-derived claim it corrects.
- `mark3labs/mcp-go@v0.57.0/mcp/types.go:163` (`LATEST_PROTOCOL_VERSION = "2025-11-25"`) — independently re-verified alongside the correction above; HIGH confidence, primary source. Supersedes this document's earlier `v0.56.0` version reference to the version actually re-checked during the correction.
- [`https://mer.vin/2026/07/stateful-vs-stateless-mcp-sticky-sessions-gone/`](https://mer.vin/2026/07/stateful-vs-stateless-mcp-sticky-sessions-gone/), [`https://appwrite.io/blog/post/mcp-goes-stateless-in-the-2026-07-28-specification`](https://appwrite.io/blog/post/mcp-goes-stateless-in-the-2026-07-28-specification), [`https://4sysops.com/archives/2026-07-28-model-context-protocol-mcp-stateless-multi-round-trip-routable-headers-authorization-hardening/`](https://4sysops.com/archives/2026-07-28-model-context-protocol-mcp-stateless-multi-round-trip-routable-headers-authorization-hardening/), [`https://workos.com/blog/mcp-2026-spec-agent-authentication`](https://workos.com/blog/mcp-2026-spec-agent-authentication) — third-party commentary, WebSearch results only (not individually fetched), used only to corroborate/cross-check the official changelog's own wording on stateless core, MRTR, routing headers, and CIMD/RFC 9207 — LOW confidence per the seam, used for corroboration only, never as the sole source for a claim in this document.
- WebSearch on mark3labs/mcp-go and `modelcontextprotocol/go-sdk` protocol-version support — LOW confidence per the seam (generic `websearch` provider), and this specific search is what produced the now-corrected "pre-release only" claim about `go-sdk` (it surfaced only the first pre-release, `v1.7.0-pre.1`, and missed the subsequent stable `v1.7.0`). Retained here as a documented example of why the module-proxy/source-level verification above is the standard this file now holds itself to for SDK-maturity claims, not as a source still being relied on for that claim.

---
*Architecture research for: MCP 2026-07-28 protocol currency integration, codegraph-go v0.3.0*
*Researched: 2026-08-03; corrected 2026-08-03*
