# Stack Research: MCP 2026-07-28 Protocol Currency

**Domain:** MCP server SDK currency + tool-manifest vulnerability scanning (subsequent-milestone stack decision, v0.3.0)
**Researched:** 2026-08-03
**Confidence:** HIGH — every version number and code claim below was fetched live (GitHub REST API, raw source at pinned tags, or executed locally in this repo) on 2026-08-03, not recalled from training data or inferred from the MCP blog post.

## Recommended Stack

### Core Decision: DEFER the SDK swap and the 2026-07-28 protocol adoption — with a dated re-check

| Decision point | Verdict | Confidence |
|---|---|---|
| Adopt `2026-07-28` on `mark3labs/mcp-go` now | **Not possible.** The dependency does not support it in any shipped release. | HIGH (verified: latest release inspected directly) |
| Swap to `modelcontextprotocol/go-sdk` now | **Not recommended for this milestone.** SDK is ready; our verification harness (a prerequisite this milestone also owns) is not. Migrating before it exists is exactly the circularity the milestone's own "Key context" warns against. | HIGH on SDK readiness; this is a judgment call on sequencing, not a hard fact |
| Explicit dated defer | **Recommended outcome.** SEP-2577 guarantees a ≥12-month deprecation window (until at least 2027-07-28) on the initialize handshake, roots/sampling/logging, and list-changed notifications our server currently exercises none of beyond the handshake — so nothing breaks by waiting. Re-check at the next milestone boundary or when `mark3labs/mcp-go` ships `server/discover` (issue #928), whichever comes first. | HIGH on the deprecation-window fact (from the spec); the recommendation itself is this research's judgment, flagged as such |

This is a **decision recommendation, not a technology table** — the "stack" question this milestone poses is binary (which SDK, which protocol revision), not a list of libraries to add. The supporting evidence for each numbered question follows.

---

## Q1 — Does `mark3labs/mcp-go` support `2026-07-28`?

**VERIFIED, not inferred:** No.

| Fact | Value | Source |
|---|---|---|
| Our pinned version | `v0.56.0` | `go.mod:13` |
| Latest published release | `v0.57.0`, published `2026-07-23T09:18:40Z` | `gh api repos/mark3labs/mcp-go/releases` |
| `mcp.LATEST_PROTOCOL_VERSION` in `v0.57.0` source | `"2025-11-25"` | fetched `mcp/types.go` at tag `v0.57.0` directly |
| `mcp.ValidProtocolVersions` in `v0.57.0` | `["2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"]` | same file |
| `2026-07-28` mentioned anywhere in `v0.57.0`'s changelog | No | `gh api repos/mark3labs/mcp-go/releases/tags/v0.57.0` |

**What changed between our pinned `v0.56.0` and latest `v0.57.0`** (full changelog, none of it protocol-revision-related): atomic `initialized` state fix, `jsonschema_description`/enum struct-tag support, streamable-HTTP session-close-before-shutdown fix, `Content-Type` parameter tolerance on the GET listening stream, per-method JSON-RPC error labeling on streamable HTTP, embedded-resource-contents unmarshal fix, a new **streamable HTTP stream resumability** feature via pluggable `EventStore`, a multi-line SSE data-field concatenation fix, and in-process-transport notification delivery fix. Bumping to `v0.57.0` is safe and worthwhile on its own merits but does not touch spec currency at all — it is not a substitute for the milestone's protocol question.

**Open issue, not a roadmap statement:** [`mark3labs/mcp-go#928`](https://github.com/mark3labs/mcp-go/issues/928) — "feat(server): support SEP-2575 stateless MCP (`server/discover` + `Mcp-Method`/`Mcp-Name` headers)" — filed 2026-07-14, **still open**, zero maintainer comments (the one comment is an automated internal-tracker link-bot), no linked PR, no milestone/label indicating a target release. The issue author explicitly scopes out SEP-2322 (MRTR), SEP-2549 (cache hints), and SEP-2567 (sessionless) as "arguably deserve their own issues" — meaning even a merged #928 would not be full `2026-07-28` currency, just the minimal `server/discover` + header surface. **There is no committed timeline of any kind.** Treat "mark3labs ships 2026-07-28" as an unscheduled community contribution, not a plan.

**No `WithProtocolVersion`-style constructor option exists** in `mark3labs/mcp-go` (`v0.57.0`) — every exported `With*` `ServerOption` was enumerated directly from `server/server.go`; none configures protocol version. The server always echoes back either the client's requested version (if it appears in `ValidProtocolVersions`) or `LATEST_PROTOCOL_VERSION` — see Q4.

## Q2 — State of `modelcontextprotocol/go-sdk` (the official Go SDK)

**VERIFIED:** Mature, stable, and already current with `2026-07-28` on release day.

| Fact | Value | Source |
|---|---|---|
| Latest version | `v1.7.0`, published `2026-07-28T13:09:53Z` — the **same calendar day** as the spec revision | `gh api repos/modelcontextprotocol/go-sdk/releases` |
| Stability | Well past 1.0 (`v1.0.0` shipped earlier; the module has been on the `v1.x` line through 6+ minor releases: 1.1 → 1.7, released roughly monthly since 2025-10-30) | release list |
| `go.mod` minimum Go version | `go 1.25.0` — compatible with this repo's `go 1.26.5` | fetched `go.mod` at tag `v1.7.0` |
| Spec revisions supported | `2026-07-28` (new, default for new clients), with full backward compatibility to `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05` — negotiated automatically, highest mutually-supported version wins | `v1.7.0` release notes (read directly, not the MCP blog) |
| Adoption signal (verified via API) | 4,926 stars, 490 forks, 81 open issues, pushed same day as this research (`2026-08-03`) | `gh api repos/modelcontextprotocol/go-sdk` |
| Adoption signal (production use, cited in the SDK's own release notes) | "`v1.7.0-pre.3` is already successfully used by GitHub, serving more than half a million users" (linking to a GitHub Changelog post about GitHub's own MCP server) | `v1.7.0` release notes |
| Comparison: `mark3labs/mcp-go` | 8,970 stars, 864 forks, 32 open issues — still the larger community by raw star count, but that gap does not track the protocol-currency question this milestone is actually asking | `gh api repos/mark3labs/mcp-go` |

**Server construction / tool registration / stdio serve loop** — read directly from the SDK's own `examples/server/basic/main.go` at tag `v1.7.0`:

```go
type SayHiParams struct {
    Name string `json:"name"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, args SayHiParams) (*mcp.CallToolResult, any, error) {
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "Hi " + args.Name}},
    }, nil, nil
}

server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v0.0.1"}, nil)
mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
serverSession, err := server.Connect(ctx, &mcp.StdioTransport{}, nil)
// or, for the simple single-session CLI case: server.Run(ctx, &mcp.StdioTransport{})
```

This is the **idiomatic, typed** path (`mcp.AddTool[In, Out]`, generic, JSON schema auto-derived from `In`'s struct tags via reflection). A **non-generic path also exists** — `(*Server).AddTool(t *Tool, h ToolHandler)` where `ToolHandler = func(context.Context, *CallToolRequest) (*CallToolResult, error)` — but it **panics if `t.InputSchema` is nil**, i.e. it requires you to hand-author or hand-generate the JSON schema yourself; it does not offer `mark3labs`-style `mcp.WithString(...)` schema-builder sugar. The typed generic path is the SDK's own recommended shape (it's what every example uses).

## Q3 — Concrete migration-cost comparison

Side-by-side at the exact granularity requested — server construction, tool definition/registration, handler signature, result construction — verified against both SDKs' actual source, not summarized from memory:

| Aspect | `mark3labs/mcp-go` (current, `v0.56.0`/`v0.57.0`) | `modelcontextprotocol/go-sdk` (`v1.7.0`) |
|---|---|---|
| **Server construction** | `server.NewMCPServer(name, version string, opts ...ServerOption) *server.MCPServer` — our `BuildServer` calls `server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))` | `mcp.NewServer(impl *mcp.Implementation, opts *mcp.ServerOptions) *mcp.Server` — `impl` bundles name+version into one struct; `opts` is a single struct pointer, not variadic functional options |
| **Tool schema definition** | `mcp.NewTool(name string, opts ...ToolOption) mcp.Tool`, builder-style: `mcp.WithString("query", mcp.Required(), mcp.Description(...))` per field. Our `exploreTool()`/`companionTool()` use this extensively (8 tool schemas, ~25 field declarations total across `tools.go`) | Idiomatic path: a plain Go struct per tool (`type ExploreParams struct { Query string \`json:"query"\`; Path string \`json:"path,omitempty"\`; ... }`), schema derived by reflection at `AddTool` time. Non-generic path: hand-build a `*jsonschema.Schema` or a `map[string]any` yourself — strictly more code than today's builder DSL, not less |
| **Registration** | `s.AddTool(exploreTool(), exploreHandler(...))` — one call per tool, `ServerTool`-agnostic | `mcp.AddTool(server, &mcp.Tool{Name, Description}, handlerFunc)` (generic, type-inferred from `handlerFunc`'s signature) or `server.AddTool(t *mcp.Tool, h mcp.ToolHandler)` (non-generic) |
| **Handler signature** | `server.ToolHandlerFunc = func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)` — `req` by **value** | Generic: `mcp.ToolHandlerFor[In, Out] = func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error)` — **3 params, 3 returns**, `req` by **pointer**, typed `input` instead of dynamic getters. Non-generic: `mcp.ToolHandler = func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)` — same 2-and-2 shape as today, but still loses the `GetString`/`GetInt`/`RequireString` convenience helpers (see next row) |
| **Argument extraction** | `req.GetString("path", defaultPath)`, `req.GetInt("max_files", 0)`, `req.RequireString("query")` — dynamic, no per-tool struct needed. Every handler in `tools.go` (8 of them) uses this pattern | No equivalent dynamic accessor found on `*CallToolRequest`/`CallToolParamsRaw` in the SDK source. The typed path gets args for free via `input In` (struct field access); the non-generic path requires you to unmarshal `req.Params.Arguments` (raw JSON) by hand per call |
| **Result construction (success)** | `mcp.NewToolResultText(s string) *mcp.CallToolResult` — one-line helper. Every handler in `tools.go` ends with this | **No equivalent helper exists.** Must construct manually: `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}` — verified against the SDK's own canonical example, which does exactly this by hand |
| **Result construction (error)** | `mcp.NewToolResultError(msg string) *mcp.CallToolResult` — one-line helper, used in every handler's error branch (multiple per handler) | No equivalent helper found. `CallToolResult.IsError bool` field exists (doc comment: "errors that originate from the tool should be reported inside Content, with IsError set to true") — manual construction: `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}` |
| **Serve loop** | `server.ServeStdio(s)` (not shown in our `tools.go`/`server.go` excerpt but is the standard mcp-go stdio entry point) | `server.Run(ctx, &mcp.StdioTransport{})` or `server.Connect(ctx, &mcp.StdioTransport{}, nil)` for session-level control |

**Honest port estimate for this codebase specifically:** `internal/mcp/tools.go` has exactly 2 handler *shapes* named in the milestone (`exploreHandler`, `companionHandler`) but `companionHandler` is a `switch` producing **7 distinct closures** (one per companion tool name) plus `exploreHandler` itself — **8 total handler bodies**, each currently ~10-25 lines built on `mcp.NewToolResultText`/`NewToolResultError` and `req.GetString`/`GetInt`/`RequireString`. Porting to the official SDK's idiomatic generic path means: (1) defining 8 typed parameter structs (mechanical, low-risk — every field already has a name, type, and description in the existing `mcp.With*` calls, so this is a direct transcription), (2) rewriting every `req.GetX(...)` call site to a struct field read (mechanical), (3) rewriting every `mcp.NewToolResultText(...)`/`NewToolResultError(...)` call site to manual `&mcp.CallToolResult{...}` construction (mechanical, more verbose), and (4) changing `BuildServer`'s construction call and the two `server.AddTool`/`mcp.AddTool` call sites. **This is a bounded, mechanical refactor confined to `internal/mcp/` — it does not reshape the package's architecture** (the seam this package already has — `openEngine`, `confineToRepoRoot`, `WorktreeNotice` prefixing, the `internal/query.Engine` delegation pattern — is SDK-agnostic and untouched by either SDK's API shape). The `server_test.go` and `mcptest`/in-process client usage would need a parallel rewrite against the new SDK's client API (not audited line-by-line here — flagged as the concrete scope of the "real-client MCP verification" work item, which this milestone already schedules separately and *before* any swap).

## Q4 — Can either SDK declare an explicit protocol version instead of inheriting a `LATEST_PROTOCOL_VERSION`-style constant?

**VERIFIED: No, neither SDK exposes a server-construction-time "pin this exact version" option.**

- **`mark3labs/mcp-go`**: every exported `With*` `ServerOption` in `server/server.go` (29 functions, enumerated directly from source) was checked — none configures a fixed protocol version. The server's `protocolVersion(clientVersion string) string` method (also read directly) always **echoes the client's requested version if valid, else falls back to `mcp.LATEST_PROTOCOL_VERSION`** — negotiation is automatic and un-overridable at the `NewMCPServer`/`ServerOption` layer.
- **`modelcontextprotocol/go-sdk`**: `ServerOptions` (the full struct, read directly from `mcp/server.go` at `v1.7.0`) has no protocol-version field either. Its own internal version constants (`latestProtocolVersion`, `protocolVersion20260728`, etc., in `mcp/shared.go`) are **unexported** — this SDK doesn't even give you a public constant to reference, let alone override. `StreamableHTTPOptions.Stateless` exists but is an HTTP-transport-only knob (irrelevant to our stdio-only server) that gates whether `2026-07-28` is *accepted* over that transport, not a version-pin mechanism.

**Implication for the milestone's stated need** ("assert our wire version in a test so a dependency bump cannot silently move it"): this cannot be satisfied by finding a constructor option on either SDK. It must be satisfied by **decoupling the test's expected value from any SDK-owned symbol** — replace `server_test.go`'s `req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION` (confirmed still present at `internal/mcp/server_test.go:81`, referencing the SDK's own moving constant) with a **repo-owned literal** (e.g. `const codegraphMCPProtocolVersion = "2025-11-25"`), sent as the request version, with the test asserting the **actual wire response** — not the SDK's compile-time constant — equals that same literal. This is SDK-agnostic guidance: it applies whether the milestone ultimately adopts or defers.

## Q5 — Correct `govulncheck` invocation for `go.tool.mod`

**VERIFIED by direct execution in this repo, not by reading docs alone** — the naive approach was reproduced as vacuous, and a working alternative was confirmed live.

### The modes that exist (from the installed `govulncheck v1.6.0`'s own `-h`, matching `go.tool.mod`'s pinned `golang.org/x/vuln v1.6.0`)

```
-mode value   supports 'source', 'binary', and 'extract' (default 'source')
-scan value   one of 'module', 'package', or 'symbol' (default 'symbol')
```

- **`source`** (default): builds an SSA call graph from real source and reports only symbols *reachable* from your code's entry points — the most precise, lowest-noise mode, but it requires the target packages' source to be loadable as ordinary Go packages under the active module.
- **`binary`**: analyzes a *compiled* binary's symbol table. Per the official docs (fetched from `pkg.go.dev/golang.org/x/vuln/cmd/govulncheck`): **not call-graph aware** — it "cannot show call graphs" and "may report false positives for code that is in the binary but unreachable." It is still meaningfully more precise than a naive dependency-list scan, because the Go linker's dead-code elimination means a vulnerable symbol usually only survives into the binary if something actually references it.
- **`extract`**: pulls a minimal analysis blob out of a binary for later `-mode=binary` use (e.g., cross-machine); not relevant here.

### What the current `task vuln` target actually does (reproduced live, not assumed)

`Taskfile.yml`'s `vuln` target runs `GOWORK=off go tool -modfile=go.tool.mod govulncheck ./...`. Running this exact command and asking for `-show verbose` output proves it **scans the 146 modules of the main `go.mod`** (`internal/...`, `cmd/codegraph`, etc.) — **not** `go.tool.mod`'s ~368 third-party tool modules. The `-modfile=go.tool.mod` flag is consumed by `go tool` to select *which modfile's binary of `govulncheck` to run* — it does not propagate into govulncheck's own package-pattern resolution, which still defaults to the ambient `go.mod` at the repo root. **This target is currently a no-op duplicate of the CI gate, not an audit of the tool closure** — directly confirming the milestone's "999.3 closes the ~400-module credentialed-CI-tooling gap" framing is a real, currently-open gap, not a hypothetical one.

**Attempted fix that does NOT currently work** (documented so it isn't re-attempted): pointing `govulncheck -mode=source` at the tool packages' import paths directly (`govulncheck github.com/go-task/task/v3/cmd/task ...`) while forcing module context via `GOFLAGS=-modfile=go.tool.mod` fails with `govulncheck: loading packages: err: exit status 1: stderr: build flag -modfile only valid when using modules` — reproduced live against the installed `govulncheck v1.6.0`. `go list -modfile=go.tool.mod <pkg>` works fine standalone; the failure is specific to how `govulncheck`'s internal `go/packages` driver invokes `go list` for source-mode loading, and no `golang/vuln` issue or doc confirms a supported workaround (issue search on `golang/vuln` for "modfile" returned nothing). **UNRESOLVED as a source-mode path** — flagging this precisely rather than guessing further, since it would need `x/vuln` maintainer confirmation or a newer `govulncheck` release to settle definitively.

### What DOES work — verified live, this is the recommended invocation shape

Build each tool's binary from `go.tool.mod` (no symbol stripping — do **not** pass `-ldflags="-s -w"` for this audit build, only for release artifacts), then run `govulncheck -mode=binary` per binary:

```bash
GOWORK=off go build -modfile=go.tool.mod -o /tmp/toolbins/task       github.com/go-task/task/v3/cmd/task
GOWORK=off go build -modfile=go.tool.mod -o /tmp/toolbins/goreleaser github.com/goreleaser/goreleaser/v2
GOWORK=off go build -modfile=go.tool.mod -o /tmp/toolbins/govulncheck golang.org/x/vuln/cmd/govulncheck

govulncheck -mode=binary /tmp/toolbins/task
govulncheck -mode=binary /tmp/toolbins/goreleaser
govulncheck -mode=binary /tmp/toolbins/govulncheck
```

Executed live against the `task` binary: this produced **real, non-vacuous, distinct-from-main-module results** — `GO-2026-5932` (`golang.org/x/crypto/openpgp` unmaintained/unsafe-by-design) reported as present in a *required* module but not reachable from `task`'s own code paths ("Module Results", not "Symbol Results" — i.e. correctly down-ranked as lower-severity than a reachable finding). This is exactly the shape of finding a per-tool binary-mode scan should surface, and it is **not** a finding that appears in the main module's `govulncheck` output (verified: the main-module run's earlier output showed a *different* single module-level advisory, confirming the two scans cover genuinely disjoint dependency graphs, as expected).

`go.tool-lint.mod` (the `actionlint` tool, ~13 modules including its own `require` block) needs the identical treatment — build `actionlint`'s binary from `-modfile=go.tool-lint.mod`, scan with `-mode=binary`. Smaller closure, same pattern, same fix.

**Recommendation for the Taskfile `vuln` target and/or a new CI gate:** replace (or add alongside) `govulncheck ./...` with a per-tool `-mode=binary` loop over binaries built fresh from each isolated modfile. This trades call-graph precision (accepted: binary mode's own docs concede it can't show call graphs and may over-report unreachable code) for actually covering the ~380 modules (`go.tool.mod` + `go.tool-lint.mod` combined) that presently have zero vulnerability-scanning coverage despite running with full CI credentials (cosign signing, GitHub token, cloud-storage backends for GoReleaser). A `-scan=package` or `-scan=module` binary-mode run is not needed as a fallback — the default `-scan=symbol` behavior worked without adjustment in the live test above.

---

## What NOT to Use / What NOT to Do

| Avoid | Why | Use Instead |
|---|---|---|
| Swapping to `modelcontextprotocol/go-sdk` *this milestone*, before a real-client verification harness exists | The milestone's own stated precondition: mcp-go's client silently skips malformed lines and cannot fail a purity test (established in v1.0 Phase 4) — validating a *new* SDK using that SDK's own client is the same circularity, just with a different SDK. Verification must precede migration, not follow it | Build the harness first (already a separate milestone work item); revisit the swap decision only once it exists |
| Treating `mark3labs/mcp-go`'s eventual `2026-07-28` support as scheduled | Issue #928 is open, unclaimed, has no maintainer commitment, and explicitly scopes out most of the spec revision's actual content (MRTR, cache hints, sessionless) even if merged | Track #928; re-evaluate at the next milestone boundary, not on an assumed timeline |
| `govulncheck ./...` (no `-mode`/pattern change) as "coverage" for `go.tool.mod` | Empirically proven vacuous in this exact repo today — it silently re-scans the main module under a different modfile name, giving zero new coverage while looking like a real gate | `govulncheck -mode=binary` over binaries built from each isolated tool modfile (see Q5) |
| `GOFLAGS=-modfile=go.tool.mod` + `govulncheck -mode=source <pkg>` as a fix for the above | Reproduced failure: `build flag -modfile only valid when using modules`, from govulncheck's internal package-loading driver, not a transient issue | `-mode=binary` (see Q5); revisit source-mode only if a future `govulncheck`/`x/vuln` release documents modfile support |
| Referencing `mcp.LATEST_PROTOCOL_VERSION` (or the official SDK's — nonexistent — public equivalent) inside a test meant to catch silent protocol drift | The whole point of the assertion is defeated if the expected value is itself supplied by the dependency being tested; a version bump moves both sides of the comparison together | A repo-owned literal constant, asserted against the actual wire-negotiated version from a live handshake (SDK-agnostic; do this regardless of the adopt/defer outcome) |

## Version Compatibility

| Package A | Compatible With | Notes |
|---|---|---|
| `github.com/mark3labs/mcp-go` | Go — no explicit floor found beyond what recent releases require; not a blocker either way | Repo currently pins `v0.56.0`; `v0.57.0` is available and safe to take independently of the protocol-currency decision (changelog reviewed — no breaking changes relevant to `internal/mcp/`) |
| `github.com/modelcontextprotocol/go-sdk` | `go 1.25.0` minimum (verified from `go.mod` at `v1.7.0`) | Compatible with this repo's `go 1.26.5` — no toolchain blocker if/when the swap happens |
| `golang.org/x/vuln/cmd/govulncheck` (pinned `v1.6.0` in `go.tool.mod`) | Requires the target packages to be reachable via ordinary `go list`/`go/packages` loading in `-mode=source`; `-mode=binary` has no such constraint | Use `-mode=binary` for `go.tool.mod`/`go.tool-lint.mod` per Q5 — sidesteps the `-modfile` loading failure entirely |

## Sources

- `gh api repos/mark3labs/mcp-go/releases` (GitHub REST API, HIGH confidence — primary source, not inferred) — full release/date list, confirms `v0.57.0` latest as of 2026-08-03
- `gh api repos/mark3labs/mcp-go/releases/tags/v0.57.0` (HIGH confidence) — full changelog body, confirms zero `2026-07-28` content
- Raw `mcp/types.go` and `server/server.go` fetched at tag `v0.57.0` (HIGH confidence, primary source) — `LATEST_PROTOCOL_VERSION` value, `ValidProtocolVersions` list, `protocolVersion()` negotiation method, full `With*` `ServerOption` enumeration
- `gh api repos/mark3labs/mcp-go/issues/928` + its comments (HIGH confidence, primary source) — open SEP-2575 feature request, no maintainer timeline
- `gh api repos/modelcontextprotocol/go-sdk/releases` + `releases/tags/v1.7.0` (HIGH confidence, primary source) — `v1.7.0` published 2026-07-28, full SEP-by-SEP changelog read directly (not summarized from the MCP blog)
- Raw `mcp/server.go`, `mcp/shared.go`, `mcp/protocol.go`, `mcp/requests.go`, `examples/server/basic/main.go`, `mcp/mcp_example_test.go` fetched at tag `v1.7.0` (HIGH confidence, primary source) — `NewServer`/`ServerOptions`/`AddTool`/`ToolHandler`/`ToolHandlerFor`/`CallToolResult`/`CallToolRequest` signatures, canonical usage pattern, unexported internal protocol-version constants
- `gh api repos/mark3labs/mcp-go` and `repos/modelcontextprotocol/go-sdk` (HIGH confidence) — star/fork/issue counts for adoption comparison
- Live local execution in this repo (HIGH confidence, empirical, reproduced 2026-08-03): `GOWORK=off go tool -modfile=go.tool.mod govulncheck -show verbose ./...` (proves current `task vuln` scans the main module, not `go.tool.mod`); `GOFLAGS=-modfile=go.tool.mod ... govulncheck -mode=source <pkg>` (reproduces the `-modfile only valid when using modules` failure); `go build -modfile=go.tool.mod -o ... <tool-pkg>` + `govulncheck -mode=binary <binary>` (proves the working alternative, with real distinct findings)
- `pkg.go.dev/golang.org/x/vuln/cmd/govulncheck` (MEDIUM confidence, official docs fetched via WebFetch — some detail on `-scan` was not present in the fetched page and is marked UNRESOLVED rather than guessed) — `-mode` semantics, binary-mode limitations
- Local `govulncheck -h` / `-version` (HIGH confidence, primary source, matches `go.tool.mod`'s pinned `golang.org/x/vuln v1.6.0`) — authoritative current-release flag set
- `.github/workflows/ci.yml:167-168` (HIGH confidence, this repo) — confirms `golang/govulncheck-action@v1.1.0` scans only the main module in CI today, corroborating the "credentialed-CI-tooling gap" framing in `PROJECT.md`
- `gh api "search/issues?q=repo:golang/vuln+modfile"` (HIGH confidence on absence — search returned zero results, cited as the evidence for the Q5 UNRESOLVED note rather than a guess)

---
*Stack research for: codegraph-go v0.3.0 "MCP Protocol Currency" milestone*
*Researched: 2026-08-03*
