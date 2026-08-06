# Phase 2: SDK Migration — Pattern Map

**Mapped:** 2026-08-05
**Files analyzed:** 8 (5 modified, ~1-2 new sibling implementation files, 2 archtest guards to verify/update)
**Analogs found:** 8 / 8 — every file has an in-repo analog because this phase modifies existing files behind an existing seam. No "no analog found" entries.

**Search methodology note (per known-failure-mode warning):** all occurrence counts below were produced with `rg -n` (line-numbered) or by counting matched `func Test...` lines, never `rg -l` file-path counts. Two searches were run per claimed convention (see "Shared Patterns" below) before concluding a pattern's shape.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/mcp/server.go` (add `goSDKServer` sibling + update `BuildServer`/`NewStdioServer`) | service / adapter | request-response (stdio JSON-RPC) | itself — `mark3labsServer` adapter (lines 84-90) is the direct sibling template | exact (self-analog, new type alongside existing) |
| `internal/mcp/tools.go` (rewrite tool schemas + handlers for go-sdk `AddTool`) | controller (RPC handler set) | request-response | itself — same file, same responsibilities, backend swapped | exact |
| `internal/mcp/protocol_version.go` (doc-comment update only per D-06 finding; constant value unchanged) | config / constant | n/a | itself | exact |
| `internal/mcp/server_test.go` (adapt in-process client construction + set-equality assertions to go-sdk client) | test | request-response | itself | exact |
| `internal/mcp/archtest/protocol_version_test.go` (verify guard still fires against go-sdk's unexported `negotiatedVersion`/version consts) | test (archtest guard) | static analysis | itself | exact |
| `internal/mcp/archtest/protocol_version_selftest_test.go` (update overlay's planted import from mark3labs to go-sdk equivalent, or add a second overlay) | test (archtest self-test) | static analysis | itself | exact |
| `internal/cli/archtest/mcp_sdk_confinement_test.go` (already forward-declares `github.com/modelcontextprotocol/go-sdk` in `forbiddenMCPSDKPrefixes` — verify only, likely no edit needed) | test (archtest guard) | static analysis | itself | exact — already written for this migration |
| `internal/cli/archtest/mcp_sdk_selftest_test.go` (planted-import self-test — may need a second planted-import case for go-sdk's import path, mirroring the mark3labs one) | test (archtest self-test) | static analysis | itself | exact |
| `go.mod` / `go.sum` (SDK-03: remove mark3labs, add go-sdk) | config | n/a | n/a — mechanical `go get`/`go mod tidy` per Q9 | n/a |

No net-new production package is implied by CONTEXT.md/RESEARCH.md — every file in scope already exists and is modified in place, which is why every analog is "itself, pre-swap" rather than a sibling elsewhere in the tree. This is expected: SDK-02 (prior phase) already built the seam specifically so Phase 2 is a backend swap, not new architecture.

## Pattern Assignments

### `internal/mcp/server.go` — the `Server` interface + adapter sibling pattern

**Analog:** the existing `mark3labsServer` adapter, same file, lines 76-90.

**Interface + adapter pattern to replicate for go-sdk** (verbatim, lines 76-90):
```go
type Server interface {
	ServeStdio() error
}

// mark3labsServer is Server's only implementation today: a thin adapter
// over the mark3labs/mcp-go-backed *server.MCPServer BuildServer
// constructs. A future SDK swap (Phase 2) adds a sibling implementation
// behind this same interface; internal/cli never needs to change.
type mark3labsServer struct{ inner *server.MCPServer }

func (m *mark3labsServer) ServeStdio() error { return server.ServeStdio(m.inner) }
```

The go-sdk sibling should follow the same one-line-body adapter shape. Per RESEARCH Q8, the natural go-sdk equivalent is:
```go
type goSDKServer struct{ inner *mcp.Server }
func (g *goSDKServer) ServeStdio() error {
	return g.inner.Run(context.Background(), &mcp.StdioTransport{})
}
```
Note the doc comment on `Server` (line 76-82) and the doc comment on `mark3labsServer` (84-87) should both be updated once the swap lands — they currently describe mark3labs as the only implementation and predict this exact swap; after the swap, either delete `mark3labsServer` entirely (SDK-03 requires mark3labs gone from `go.mod` by phase end — CONTEXT.md Claude's Discretion leans "sibling first, then delete", so `mark3labsServer` is transitional, not a permanent second implementation) or replace it in place if the "replace in place" discretion option is taken.

**Functional-options pattern to preserve unchanged** (lines 92-115, `buildConfig`/`Option`/`WithSessionLog`):
```go
type buildConfig struct {
	sessionLog io.Writer
}

type Option func(*buildConfig)

func WithSessionLog(w io.Writer) Option {
	return func(c *buildConfig) { c.sessionLog = w }
}
```
This absorbs the go-sdk swap's new configuration (e.g. explicit `ServerOptions.Capabilities` per RESEARCH Q3's divergence-2 fix) without touching any of the 17 positional `BuildServer` call sites in tests. Do not widen `BuildServer`'s signature — add new construction-time knobs as new `Option` values if any are needed, mirroring `WithSessionLog`'s shape exactly.

**Session-line hook — needs real redesign, not mechanical translation** (current code, lines 178-200, `AddAfterInitialize`):
```go
serverOpts := []server.ServerOption{server.WithToolCapabilities(true)}
if cfg.sessionLog != nil {
	var mu sync.Mutex
	hooks := &server.Hooks{}
	hooks.AddAfterInitialize(func(_ context.Context, _ any, req *mcp.InitializeRequest, res *mcp.InitializeResult) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprint(cfg.sessionLog, formatSessionLine(
			req.Params.ProtocolVersion,
			res.ProtocolVersion,
			req.Params.ClientInfo.Name,
			req.Params.ClientInfo.Version,
			toolCount,
		))
	})
	serverOpts = append(serverOpts, server.WithHooks(hooks))
}
```
RESEARCH Q6 (verified against real go-sdk source) gives the exact replacement shape — `Server.AddReceivingMiddleware`, wrapping the whole receive pipeline and reading `res.(*mcp.InitializeResult)` after `next()` returns:
```go
s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		if method == "initialize" && err == nil {
			if initRes, ok := res.(*mcp.InitializeResult); ok {
				if params, ok := req.GetParams().(*mcp.InitializeParams); ok {
					// params.ProtocolVersion = requested, initRes.ProtocolVersion = negotiated,
					// params.ClientInfo.Name/Version = client identity, toolCount = closure var
				}
			}
		}
		return res, err
	}
})
```
Preserve the mutex-guarded single-write invariant (the existing doc comment at lines 183-188 explaining why a bare `fmt.Fprint` isn't atomicity-safe on an arbitrary `io.Writer`) and the session-line format function `formatSessionLine` itself unchanged — it is a one-way additive-only wire contract per Phase 1 D-16 (CONTEXT.md Integration Points).

**Capabilities fix required (RESEARCH Q3, D-11):** the go-sdk sibling's `mcp.NewServer(...)` call must pass an explicit `ServerOptions.Capabilities` (e.g. `&mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{ListChanged: true}}`) to replicate today's unconditional `server.WithToolCapabilities(true)` (line 178) — otherwise the `"tools"` key silently vanishes at zero tools (MCP-03's no-index path), which is this migration's named failure mode to actively counter, not merely document.

### `internal/mcp/tools.go` — typed-struct schema + handler pattern (D-07)

**Analog:** the existing `exploreTool`/`exploreHandler` pair (lines 79-123) and the `companionTool`/`companionHandler` switch (lines 129-381), same file.

**Current builder-chain schema pattern** (lines 79-86), to be replaced by struct-tag inference:
```go
func exploreTool() mcp.Tool {
	return mcp.NewTool("codegraph_explore",
		mcp.WithDescription("Explore relevant symbols: verbatim source, call paths, blast radius"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural-language or symbol/file query")),
		mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		mcp.WithNumber("max_files", mcp.Description("Cap on distinct files returned (default 5)")),
	)
}
```

**Target shape per D-07 + RESEARCH Q7's empirically-confirmed struct**, matching field order to the CLI flag order already documented in this file's comments:
```go
type ExploreArgs struct {
	Query    string `json:"query" jsonschema:"Natural-language or symbol/file query"`
	Path     string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	MaxFiles int    `json:"max_files,omitempty" jsonschema:"Cap on distinct files returned (default 5)"`
}
```
`required` derives from the absence of `omitempty`/`omitzero` (RESEARCH Q7, source-confirmed) — no separate `Required()` call needed. `int` fields become `"type":"integer"` (not `"number"` as today) — a confirmed, deliberate divergence to name once per RESEARCH's 7-occurrence list (D-08), not per tool.

**Current handler + manual-extraction + error-wrap pattern** (lines 103-123), the shape all 8 handlers share:
```go
func exploreHandler(repoPath, defaultPath string, detector *gitmeta.CachingDetector) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxFiles := req.GetInt("max_files", 0)

		eng, close, err := openEngine(req, defaultPath, repoPath, detector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer close()

		out, err := eng.Explore(q, maxFiles)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out), nil
	}
}
```
**Every one of the 8 handlers' `if err != nil { return mcp.NewToolResultError(err.Error()), nil }` occurrences (confirmed via RESEARCH Q5: zero bare `return nil, err` paths exist today) can be simplified** under `AddTool`'s typed `ToolHandlerFor` contract (RESEARCH Q5, source-confirmed at `mcp/server.go:377-392` — `toolForErr` auto-wraps a returned Go `error` into the same `IsError:true`/`Content` shape via `CallToolResult.SetError`). Target shape:
```go
func exploreHandler(repoPath, defaultPath string, detector *gitmeta.CachingDetector) func(context.Context, *mcp.CallToolRequest, ExploreArgs) (*mcp.CallToolResult, ExploreOut, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ExploreArgs) (*mcp.CallToolResult, ExploreOut, error) {
		eng, close, err := openEngine(args, defaultPath, repoPath, detector)
		if err != nil {
			return nil, ExploreOut{}, err
		}
		defer close()

		out, err := eng.Explore(args.Query, args.MaxFiles)
		if err != nil {
			return nil, ExploreOut{}, err
		}
		return nil, ExploreOut{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}, nil
	}
}
```
(Exact `ToolHandlerFor` signature — `(context.Context, *CallToolRequest, In) (*CallToolResult, Out, error)` — must be confirmed against the pinned go-sdk version at implementation time; sketch only, per RESEARCH Q5/Q8's own "sketch, not final code" caveat.) `openEngine`'s `req mcp.CallToolRequest`-typed first parameter (`internal/mcp/tools.go:63`) must change to accept the typed args struct instead of doing `req.GetString`/`req.RequireString` extraction — this is the one signature change that ripples through all 8 call sites uniformly, since `resolvePath` (line 20-22) and `openEngine` (line 63-75) are both shared across every handler.

**Companion switch structure to preserve exactly** (the `switch name { case "node": ... case "search": ... }` shape at lines 129-185 for schemas and 221-381 for handlers) — only the per-case schema/handler bodies change to the typed-struct shape; the dispatch-by-name structure, the `panic("mcp: companionTool: unknown tool name " + name)` default (line 183), and the doc comments explaining the SURF-06/D-16 markdown-vs-JSON asymmetry (lines 195-220) all carry forward unchanged.

### `internal/mcp/protocol_version.go` — asserted-pin doc comment, mechanism updates only

**Analog:** itself (25 lines, entire file already read — no partial reads needed, well under the 2,000-line threshold).

The `const ProtocolVersion = "2025-11-25"` value is unchanged by this phase (D-04: take whatever the SDK negotiates, no pinning/injection — the constant remains an *asserted compatibility pin*, not an injected value, exactly as documented today). Only the doc comment (lines 9-22) needs updating: it currently states "mark3labs/mcp-go v0.56.0 decides the negotiated revision inside the unexported `(*MCPServer).protocolVersion` method and exposes no `WithProtocolVersion`-style server option" — replace this sentence with the RESEARCH Q1-confirmed equivalent for go-sdk (`negotiatedVersion()` in `mcp/shared.go:44-79`, unexported, no `ProtocolVersion` field in `ServerOptions`'s 17 fields — RESEARCH Q1 enumerates the full struct). The closing sentence "The stricter ROADMAP phrasing ('reads from') lands in Phase 2, when the SDK swap supplies a backend whose protocol revision a caller can actually supply" is now **proven false by RESEARCH Q1/Q6** and must be corrected, not carried forward — Q1's "One caveat" section documents the only reachable mechanism (`AddReceivingMiddleware` reading the *result* after `next()` returns) and explicitly notes this is not a blessed setter API.

### `internal/mcp/server_test.go` — set-equality test pattern (preserve unchanged in intent)

**Analog:** itself — `TestDefaultToolVisibility` (122-133), `TestAllowlist` (135-163), `TestNoIndexZeroTools` (165-174).

**Set-equality pattern to preserve exactly** (lines 122-133):
```go
func TestDefaultToolVisibility(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{}, dir, dir)

	got := listToolNames(t, s)
	want := []string{"codegraph_explore"}
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}
}
```
`equalStrings` (lines 308-318) is a hand-rolled exact-slice-equality helper — not `len(got) > 0`. This must survive the migration unchanged in intent per CONTEXT.md's explicit instruction ("These must survive the migration unchanged in intent").

**What must change mechanically:** `listToolNames` (lines 93-120) and `initClient` (75-91) currently construct an in-process client via `mcpclient.NewInProcessClient(s)` where `s` is `*server.MCPServer` (mark3labs' concrete type). go-sdk's in-process testing equivalent must be substituted — RESEARCH does not document this SDK-side test helper explicitly; at implementation time, confirm go-sdk's client package offers an equivalent in-process transport (its `Run(ctx, transport)` signature from Q8 suggests an in-memory `Transport` pair is the likely mechanism — check `mcp/transport.go` for an in-process pipe type analogous to mark3labs' `NewInProcessClient`). `TestExploreHandlerDelegatesToEngine` (180-220) and `TestOpenEnginePathConfinedToRepoRoot` (228-270) both depend on the same `mcp.AsTextContent`/`result.IsError`/`result.Content` shape — RESEARCH Q5 confirms this wire shape is unchanged (`IsError:true` + `Content` text), so these assertions' *logic* carries forward; only the client-construction plumbing changes.

### `internal/mcp/archtest/protocol_version_test.go` — VRFY-02 guard, verify not rewrite

**Analog:** itself. This guard is deliberately **identity-agnostic** by design (doc comment lines 7-14): it matches any external `*types.Const` whose name matches `(?i)protocol.?version`, not a mark3labs-specific spelling. Per its own doc comment: "it is deliberately identity-agnostic so it keeps firing after Phase 2 replaces mark3labs/mcp-go with a different SDK, without allowlist maintenance."

**Action for this phase:** confirm — don't rewrite — that go-sdk's unexported identifiers (`protocolVersion20260728`, `latestProtocolVersion`, etc., per RESEARCH Q1) don't accidentally trip a false positive, and more importantly that the guard's core claim holds: go-sdk exposes **no exported constant matching the pattern** at all (RESEARCH Q1 enumerated every `Version`/`Protocol`/`Negotiat`-named func/method and found exactly one, a read-only getter `ServerRequest[P].ProtocolVersion()`, which is a `*types.Func`/method, not a `*types.Const` — `isExternalProtocolVersionConstant`'s type-switch (lines 130-133) already excludes non-const objects by design, so this should pass without modification). Verify by running `go test ./internal/mcp/archtest/... -count=1` after the swap rather than by inspection alone.

### `internal/mcp/archtest/protocol_version_selftest_test.go` and `internal/cli/archtest/mcp_sdk_selftest_test.go` — non-vacuity self-tests, planted-import overlay pattern

**Analog:** itself, both files — identical technique: resolve the real on-disk path via `packages.Load`, build an in-memory `Overlay` inserting a real, syntactically-used (not merely imported) reference to the forbidden identifier immediately after the package clause, reload with the overlay, and assert the production guard's predicate fires on the overlaid package. Verbatim shared shape (from `mcp_sdk_selftest_test.go:55-107`):
```go
func TestInternalCLIImportsNoMCPSDK_PlantedImportIsError(t *testing.T) {
	path := serveGoPath(t)
	original, err := os.ReadFile(path)
	// ... insert "package cli\n" marker, splice in a real import + a
	// package-level var referencing it, so an unused-import compile
	// failure can never masquerade as "guard caught it" ...
	violated := content[:insertAt] +
		"\nimport mcpsdkserver \"github.com/mark3labs/mcp-go/server\"\n" +
		content[insertAt:] +
		"\nvar mcpSDKConfinementSelfTestProbe = mcpsdkserver.ServeStdio\n"
	// ... packages.Load with Overlay, assert zero TypeErrors, assert the
	// forbidden-prefix predicate actually fired ...
}
```
`internal/cli/archtest/mcp_sdk_confinement_test.go`'s `forbiddenMCPSDKPrefixes` (lines 39-42) **already lists both** `github.com/mark3labs/mcp-go` and `github.com/modelcontextprotocol/go-sdk` — this guard was pre-written for this exact migration and its self-test (`mcp_sdk_selftest_test.go`) only plants the mark3labs case today. **Consider adding a second `_IsError`-suffixed planted-import test case for the go-sdk import path** (mirroring the existing test's exact structure with `github.com/modelcontextprotocol/go-sdk/mcp` substituted), so the self-test proves detection of *both* forbidden prefixes, not just the one being removed — this closes a real gap: today's self-test only proves the guard catches mark3labs, never proves it would catch a stray go-sdk import in `internal/cli` after the swap (which is exactly the failure mode SDK-02 exists to prevent going forward).

## Shared Patterns

### Error-to-wire mapping (SDK-04)
**Source:** `internal/mcp/tools.go` — every one of the 8 handlers' `if err != nil { return mcp.NewToolResultError(err.Error()), nil }` sites (e.g. lines 106-108, 112-114, 118-120, and the equivalent in every companion case, 239-245 etc.)
**Apply to:** every migrated tool handler.
**Confirmed replacement (RESEARCH Q5, source-verified against `mcp/server.go:377-392` and `mcp/protocol.go:347-353`):** a returned Go `error` under `ToolHandlerFor`/`AddTool` is automatically converted into the identical `{"content":[{"type":"text","text":"<err.Error()>"}],"isError":true}` wire shape via `toolForErr`'s wrapper calling `CallToolResult.SetError`. Handlers may (and per D-07 discretion, should) simplify to `return nil, zeroOut, err` instead of manually constructing `mcp.NewToolResultError(...)`.

### Set-equality assertion convention
**Source:** `internal/mcp/server_test.go:308-318` (`equalStrings`), used by `TestDefaultToolVisibility`, `TestAllowlist`, `TestNoIndexZeroTools`.
**Apply to:** any new or migrated test asserting a tool-name set or capability set — never assert non-empty or `len() > 0`; always assert the exact expected slice/set.

### Fail-loudly-on-malformed-input convention (distinct from `_IsError`)
**Source:** `internal/mcp/session_line_test.go:51` (`TestSanitizeClientFieldFailLoudly`) and `:129` (`TestParseSessionLineFieldsFailLoudly`); also `internal/mcp/archtest/protocol_version_test.go:398` (`TestProtocolVersionGuardHelpersFailLoudly`).
**What it proves:** a pure parser/predicate **helper function** returns a documented zero-value/false on malformed or incomplete input, never panics. Confirmed via `rg -n "FailLoudly"` — 8 files repo-wide, 2 in `internal/mcp/`.
**Apply to:** any new pure helper this phase introduces (e.g. a typed-args decode helper, if one is hand-written rather than left to `AddTool`'s built-in `applySchema` validation).

### `_IsError`-suffixed mutation-proof convention (distinct from FailLoudly)
**Source:** 13 occurrences confirmed via `rg -n "func Test.*IsError"` (not `rg -l`) across the repo, including `internal/cli/archtest/mcp_sdk_selftest_test.go:55` (`TestInternalCLIImportsNoMCPSDK_PlantedImportIsError`) and `internal/upgrade/*_test.go` (11 more).
**What it proves:** a **guard/assertion** (usually an archtest or shape-check) is demonstrated RED against a real, planted defect — never merely inspected as "looks correct." The convention is a test-name suffix marking "this test's job is to prove the guard fires," distinct from `FailLoudly`'s "this helper never panics."
**Apply to:** `internal/mcp/archtest/protocol_version_selftest_test.go` (already follows this convention via `TestProtocolVersionGuardCatchesOverlaidViolation` — note: does NOT carry the `_IsError` suffix despite matching the convention's intent; naming is not perfectly consistent repo-wide, don't assume the suffix is mandatory, only that the demonstrated-RED technique is) and any new archtest guard SDK-04/SDK-05 introduces.

### Typed-decode / struct-tag pattern for D-07
**Source:** `internal/query/files.go:57-79`, `internal/query/traverse.go:257-283`, `internal/query/search.go:18-21,196-...` — the existing convention for typed result structs with `json:"..."` tags (used for the CLI's `--json` output path, SURF-06/D-16's counterpart to the markdown-render path).
**Apply to:** the new `ExploreArgs`/companion `*Args` input structs (D-07) — this repo already has an established `json:"name,omitempty"` tagging convention for structured data; extend it with `jsonschema:"..."` tags per RESEARCH Q7's confirmed syntax (whole-string description, no enum mini-language) rather than inventing a new tagging style.
**Note:** these are output/result structs (JSON marshaling direction), not input/argument structs — there is no pre-existing "request-args-as-typed-struct-with-validation" convention elsewhere in the repo (CLI flags use `spf13/cobra`'s flag binding, not struct-tag reflection) — D-07's input-struct pattern is genuinely new to this codebase, and `internal/query`'s output-struct tagging is the closest available convention to imitate for tag *style*, not for input-decoding *mechanism* (that mechanism is entirely `jsonschema-go`'s, external to this repo).

## No Analog Found

None. Every file in scope is an existing file being modified behind the pre-built `Server` seam; there is no genuinely new architectural surface in this phase.

## Metadata

**Analog search scope:** `internal/mcp/`, `internal/mcp/archtest/`, `internal/cli/archtest/`, `internal/cli/serve.go`, `internal/query/` (for typed-struct tagging convention).
**Files scanned directly:** `internal/mcp/server.go` (236 lines, full read), `internal/mcp/tools.go` (381 lines, full read), `internal/mcp/protocol_version.go` (25 lines, full read), `internal/mcp/server_test.go` (318 lines, full read), `internal/mcp/archtest/protocol_version_test.go` (438 lines, full read), `internal/mcp/archtest/protocol_version_selftest_test.go` (111 lines, full read), `internal/cli/archtest/mcp_sdk_confinement_test.go` (121 lines, full read), `internal/cli/archtest/mcp_sdk_selftest_test.go` (107 lines, full read), plus targeted `rg -n` greps over `internal/query/*.go` and repo-wide `FailLoudly`/`_IsError` occurrence counts.
**Searches run to rule out false "no analog":** `rg -n "FailLoudly" -tgo -l` (8 files) cross-checked against `rg -n "func Test.*IsError" -tgo` (13 occurrences, counted by line not by file) — confirming these are two distinct, both-real conventions per the known-failure-mode warning.
**Pattern extraction date:** 2026-08-05.
