package wireoracle

import "path/filepath"

// fixtureFuncAlpha and fixtureFuncBeta name real exported functions defined
// in testdata/wireoracle/fixture/pkga/pkga.go: Alpha calls Beta
// (intra-package) and pkgb.Helper (cross-package) — real call/callee/caller
// edges the tool-coverage scenarios below query against, rather than
// inventing identifiers that would freeze an error result as the scenario's
// one-way pre-migration baseline (01-04-PLAN Task 2, D-05/VRFY-04).
const (
	fixtureFuncAlpha = "Alpha"
	fixtureFuncBeta  = "Beta"
)

// Allowlist env strings this plan's scenarios drive CODEGRAPH_MCP_TOOLS
// with (internal/mcp/server.go's companionNames, D-08a/MCP-02).
const (
	envAllowlistNodeStatus    = "CODEGRAPH_MCP_TOOLS=node,status"
	envAllowlistAllCompanions = "CODEGRAPH_MCP_TOOLS=node,search,callers,callees,impact,files,status"
	envAllowlistNodeOnly      = "CODEGRAPH_MCP_TOOLS=node"
)

// outsideRepoRootPath is a portable, host-independent literal — never a
// filepath.Join(t.TempDir(), "..", "elsewhere") sibling of the capture
// directory. confineToRepoRoot (internal/mcp/tools.go:42) embeds the
// rejected path verbatim via %q in its error, so whatever value is sent
// here lands byte-for-byte in the frozen transcript; this value is
// identical on every machine, cannot exist as a real directory, and shares
// no path prefix with the fixture directory, so it can never accidentally
// resolve inside repoPath (01-04-PLAN Task 2, review concern "the
// outside-path scenario can leak an unnormalized host path").
const outsideRepoRootPath = "/codegraph-wire-oracle-outside-root"

// unimplementedMethod is a JSON-RPC method name the server has never
// implemented and never will collide with a real MCP method — used by
// error-unknown-method to exercise request_handler.go's default case
// (mcp.METHOD_NOT_FOUND, -32601).
const unimplementedMethod = "wireoracle/unimplemented-method"

// initializeRequest returns the standard hand-authored initialize request
// every new scenario below opens with (except edge-call-before-initialize,
// which sends no initialize at all) — the same protocolVersion,
// capabilities, and clientInfo literals as the tracer's handshake-explore
// scenario, id parameterized so each scenario's own request-id sequence
// stays sequential and readable. handshake-explore itself is left
// hand-inlined and untouched (must_haves: "keeping the tracer's
// handshake-explore unchanged").
func initializeRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": handshakeExploreProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    handshakeExploreClientName,
				"version": handshakeExploreClientVersion,
			},
		},
	}
}

// toolsListRequest returns a bare "tools/list" request with the given id.
func toolsListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
	}
}

// toolCallRequest returns a "tools/call" request naming a tool and its
// arguments. arguments is typed any (not map[string]any) so malformed-shape
// scenarios can pass a non-object value (e.g. a bare string) — the exact
// shape a real malformed client request would carry on the wire.
func toolCallRequest(id int, name string, arguments any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}
}

// modernProtocolVersion is a package-local, hand-authored literal
// (D-06/VRFY-01) holding the `2026-07-28` revision — never read from an
// SDK constant, mirroring handshakeExploreProtocolVersion below and
// legacyEraVersions further down this file.
const modernProtocolVersion = "2026-07-28"

// modernMetaParams returns the SEP-2575 per-request `_meta` object a
// Modern (2026-07-28) client attaches to a sessionless request: exactly
// the three `io.modelcontextprotocol/*` keys the working probe recipe
// uses (03-CONTEXT.md D-01), all hand-authored literals in this file,
// never imported from the SDK under test.
//
// Load-bearing finding (03-CONTEXT.md D-01): SEP-2575 signals the
// protocol version through `_meta`, never through
// `params.protocolVersion` — a discover request carrying
// `params.protocolVersion` is rejected -32601 in a way that looks
// exactly like "the server does not implement discover", a wrong
// conclusion that already survived two messages once in this project's
// history. Every request built via discoverRequest/modernToolCallRequest
// below carries the version in `_meta` only.
func modernMetaParams() map[string]any {
	return map[string]any{
		"io.modelcontextprotocol/protocolVersion": modernProtocolVersion,
		"io.modelcontextprotocol/clientInfo": map[string]any{
			"name":    handshakeExploreClientName,
			"version": handshakeExploreClientVersion,
		},
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}
}

// discoverRequest returns a "server/discover" request whose params carry
// only `_meta` — SEP-2575 sessionless dispatch, no prior "initialize".
func discoverRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "server/discover",
		"params": map[string]any{
			"_meta": modernMetaParams(),
		},
	}
}

// modernToolCallRequest returns a "tools/call" request carrying the same
// Modern per-request `_meta` as discoverRequest, alongside name/arguments
// — SEP-2575 sessionless dispatch again, not a session-ordering
// violation (see modern-discover-explore's own doc comment below).
func modernToolCallRequest(id int, name string, arguments any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
			"_meta":     modernMetaParams(),
		},
	}
}

// modernMetaMissingCapabilities returns modernMetaParams() with the
// io.modelcontextprotocol/clientCapabilities key removed — the well-formed
// half of SEP-2575's `_meta` that is nonetheless incomplete, which SPEC-02's
// `-32602` half must reject. Built on modernMetaParams() rather than
// re-typing its other two key literals, so the two helpers can never drift
// apart.
func modernMetaMissingCapabilities() map[string]any {
	meta := modernMetaParams()
	delete(meta, "io.modelcontextprotocol/clientCapabilities")
	return meta
}

// modernMetaWithVersion returns modernMetaParams() with the
// io.modelcontextprotocol/protocolVersion key replaced by version — used to
// offer a well-formed-but-unsupported version (modernUnsupportedVersion
// below) without re-typing modernMetaParams' other two key literals.
func modernMetaWithVersion(version string) map[string]any {
	meta := modernMetaParams()
	meta["io.modelcontextprotocol/protocolVersion"] = version
	return meta
}

// discoverRequestWithMeta returns a "server/discover" request whose params
// carry the given `_meta` object verbatim — the variation-parameterized
// counterpart to discoverRequest, which always sends modernMetaParams()
// unmodified.
func discoverRequestWithMeta(id int, meta map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "server/discover",
		"params": map[string]any{
			"_meta": meta,
		},
	}
}

// modernUnsupportedVersion is a well-formed, supported-SHAPE protocol
// version literal that is NOT one of the five versions this server
// recognizes — 03-RESEARCH.md's Code Examples case 2, confirmed empirically
// this phase to answer `-32022`.
//
// Its lexical relationship to "2026-07-28" is LOAD-BEARING, not cosmetic:
// go-sdk's unexported validateRequestMeta (mcp/shared.go:543-576) compares
// version strings with a plain lexical comparison and reclassifies any
// `_meta.protocolVersion` value that sorts lexically SMALLER than
// "2026-07-28" as not-using-the-new-protocol BEFORE the unsupported-version
// check ever runs. A smaller literal therefore lands on the Modern-only
// method-availability gate instead of the version check, and answers
// `-32601` with a message the JSON-RPC transport's generic
// errors.Is(ErrMethodNotFound) rewrite (internal/jsonrpc2/conn.go:694-695)
// has already overwritten. This is a property of go-sdk itself, with no
// codegraph-go seam to change it: validateRequestMeta runs inside the SDK's
// own unexported ss.handle, before internal/mcp's AddReceivingMiddleware
// chain is ever invoked (03-RESEARCH.md Q1, PROVEN-ABSENT).
//
// 03-CONTEXT.md's "SPEC-02 is a real gap ... the least understood item in
// the phase" framing described exactly this `-32601` observation (its probe
// offered "1999-01-01", which sorts lexically before "2026-07-28") — that
// framing is RETRACTED here, superseded by 03-RESEARCH.md's source trace and
// its eleven empirically captured request/response pairs. The `-32601`
// answer is go-sdk's own lexical-comparison classification quirk on an
// old-looking literal, not a codegraph-go defect, and it is not asserted by
// any scenario in this file (see anchors.go's Anchors() doc comment for the
// matching statement on the anchor side). "2099-01-01" sorts lexically
// AFTER "2026-07-28" and is not one of the five supported era strings, so it
// reaches the unsupported-version check and answers `-32022` — the only
// shape a real Modern client would ever construct, since the entire point of
// a client sending `_meta.protocolVersion` at all is that it believes it
// speaks >= 2026-07-28.
const modernUnsupportedVersion = "2099-01-01"

// handshakeExploreProtocolVersion is a package-local, hand-authored
// literal (D-06) — never an SDK constant — matching what mark3labs
// v0.56.0's server negotiates today (its own LATEST_PROTOCOL_VERSION).
const handshakeExploreProtocolVersion = "2025-11-25"

// handshakeExploreQuery names a real exported symbol defined in
// testdata/wireoracle/fixture/pkga/pkga.go. The tools/call request below
// MUST carry a real, non-empty query argument: exploreTool() declares
// query as mcp.Required(), and exploreHandler returns an error result when
// it is absent — an empty arguments object would freeze an ERROR as the
// flagship tool's one-way pre-migration baseline (D-05, VRFY-04).
const handshakeExploreQuery = "Alpha"

// handshakeExploreClientName and handshakeExploreClientVersion are the
// clientInfo values the oracle's own scripted session reports — asserted
// against literally by TestFrozenTranscriptsMatch's stderr session-line
// check.
const (
	handshakeExploreClientName    = "codegraph-wire-oracle"
	handshakeExploreClientVersion = "0.0.0"
)

// legacyEraPrefix names the deliberate, frozen six-scenario Legacy
// multi-era handshake baseline (01-05-PLAN Task 1 checkpoint: six-era) —
// the four protocol revisions today's server recognizes, plus one
// unsupported revision, plus one omitted-version request. Every scenario
// name below begins with this prefix so oracle_test.go's
// TestLegacyEraBaselineIsDocumented can select the set structurally
// rather than by an enumerated name list.
const legacyEraPrefix = "legacy-"

// legacyEraVersions is the hand-authored, frozen four-revision matrix
// (D-06): exactly the four strings mark3labs v0.56.0's own
// ValidProtocolVersions declares
// [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:166-171],
// transcribed here as package-local literals — never read from any SDK
// constant (the plan 03 archtest forbids that; VRFY-01 forbids using the
// SDK under test as the source of its own expected values). Each of these
// four negotiates to itself
// [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1196-1210].
var legacyEraVersions = []string{
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// legacyUnsupportedVersion is the revision Phase 3 will implement
// (2026-07-28) — the most informative unsupported-version probe
// available, per plan 05's Task 1 checkpoint rationale. Today's server
// silently coerces it to its own latest rather than rejecting it
// [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1196-1210,
// 01-RESEARCH.md Pitfall 1] — no error-code anchor exists for this
// scenario (anchors.go) because no error fires on this path in the
// pre-migration server.
const legacyUnsupportedVersion = "2026-07-28"

// legacyOmittedVersionCoercion is what the server negotiates when a
// client's initialize params carry NO protocolVersion key at all.
//
// Post-migration (go-sdk@v1.7.0): negotiatedVersion("") checks
// slices.Contains(supportedProtocolVersions, "") — false, since the empty
// string is never one of the five listed versions — and falls through to
// the function's fallback, protocolVersion20251125
// [VERIFIED: github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/shared.go:44-79,
// 02-RESEARCH.md Q4, empirically confirmed by driving the real SDK with an
// omitted protocolVersion]. Under go-sdk this is no longer a structurally
// distinct THIRD coercion path: the omitted-version and
// unsupported-version cases now land on the exact same fallback, for
// different reasons (this one because "" never matches; unsupported
// because negotiatedVersion caps every offer strictly below
// protocolVersion20260728 — see legacyUnsupportedVersion above).
//
// Pre-migration (mark3labs v0.56.0) this was "2025-03-26" — the server's
// own backwards-compat default, applied BEFORE the ValidProtocolVersions
// check ran [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1196-1198].
// That value is superseded; do not resurrect it as a comparison target.
const legacyOmittedVersionCoercion = "2025-11-25"

// initializeRequestWithVersion returns an initialize request offering the
// given protocolVersion literal — the four-supported-plus-one-unsupported
// era scenarios' request, sharing every other field with
// initializeRequest's tracer-matching shape (capabilities, clientInfo)
// deliberately, so the ONLY variable between era scenarios is the offered
// version.
func initializeRequestWithVersion(id int, version string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    handshakeExploreClientName,
				"version": handshakeExploreClientVersion,
			},
		},
	}
}

// initializeRequestOmittingVersion returns an initialize request whose
// params carry NO "protocolVersion" key at all — a structurally distinct
// wire shape from initializeRequestWithVersion(id, "") (an EMPTY-STRING
// literal), even though mark3labs v0.56.0's
// InitializeParams.ProtocolVersion is a plain string field where both
// unmarshal to the same Go value
// [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:521-527] —
// the omission is captured here as the actual absent-key wire shape a
// real legacy client would send, not simulated via an empty string.
func initializeRequestOmittingVersion(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"capabilities": map[string]any{},
			"clientInfo": map[string]any{
				"name":    handshakeExploreClientName,
				"version": handshakeExploreClientVersion,
			},
		},
	}
}

// Concurrency ordering constraint (discovered this session, verified by hand
// against the real binary via repeated captures): mark3labs v0.56.0's stdio
// transport (server/stdio.go processMessage) dispatches every "tools/call"
// message onto a worker-pool queue and returns immediately WITHOUT waiting
// for it — every OTHER method (initialize, tools/list, and any unrecognized
// method) is handled synchronously inline before the next stdin line is even
// read. Two consequences that matter for byte-reproducible capture:
//  1. A synchronous request queued AFTER an async tools/call can complete
//     and be WRITTEN TO STDOUT BEFORE that earlier tools/call's response —
//     observed directly: a scenario sending tools/call, tools/call,
//     unknown-method in that order printed the unknown-method response
//     first, and the two tools/call responses in a racy, run-to-run-varying
//     relative order (confirmed non-deterministic across 5 repeated runs of
//     an otherwise-identical request script).
//  2. Two tools/call requests in the SAME scenario race each other in the
//     worker pool with no ordering guarantee between them.
//
// Every scenario below therefore carries AT MOST ONE tools/call request,
// and when present it is always the LAST request in Requests — exactly
// mirroring handshake-explore's own proven shape (initialize, tools/list,
// tools/call). This is a load-bearing invariant for any future scenario
// added to this list (plan 05, plan 07): violating it reintroduces
// intermittent CI flakes in TestFrozenTranscriptsMatch's byte comparison,
// not a real regression in the server under test.
//
// Scenarios returns the oracle's full scripted scenario list. Phase 1
// scripts exactly one scenario, handshake-explore, proving the oracle
// architecture end-to-end; plan 04 adds the 16 scenarios that bring the
// suite to exactly 17 — the D-05 full coverage bar approved at plan 04's
// Task 1 blocking checkpoint (full-bar, no additional scenarios); plan 05
// (this plan) adds the six-era Legacy handshake baseline below, bringing
// the suite to exactly 23 — the six-era selection approved at plan 05's
// own Task 1 blocking checkpoint — this same file, same function, no
// phase-conditional branch (must_haves).
// ExpectedScenarioCount is the exact size Scenarios() must return (D-07),
// declared immediately beside Scenarios() itself so the single source of
// truth for "how many scenarios exist" lives next to the function that
// produces them. Changing this value is a deliberate act that must land in
// the same commit as a matching change to Scenarios() and to the frozen
// transcripts directory — TestScenarioCountIsExact (oracle_test.go)
// compares len(Scenarios()) against this constant with EXACT equality,
// never a lower bound, because a lower bound cannot detect a scenario
// silently disappearing. TestTranscriptSetMatchesScenarioSet separately
// enforces the transcripts-directory half via two-way set equality. Value:
// 1 tracer scenario (plan 01) + 16 scenarios (plan 04's full D-05 coverage
// bar, the "full-bar" blocking-checkpoint selection: 4 tools/list variants,
// 7 tools/call, 4 error shapes, 1 statelessness edge) + 6 scenarios (plan
// 05's "six-era" blocking-checkpoint selection: the multi-era Legacy
// handshake baseline) + 1 scenario (phase 3 plan 01's tracer,
// modern-discover-explore: the Modern 2026-07-28 server/discover +
// sessionless tools/call proof) + 2 scenarios (phase 3 plan 03's SPEC-02
// proof: modern-meta-invalid-params and modern-meta-unsupported-version,
// freezing the -32602 and -32022 halves of per-request `_meta` validation)
// = 26. A shrinking count is the failure mode this constant exists to
// catch.
const ExpectedScenarioCount = 26

func Scenarios() []Scenario {
	return []Scenario{
		{
			Name:  "handshake-explore",
			Index: true,
			Requests: []map[string]any{
				{
					"jsonrpc": "2.0",
					"id":      1,
					"method":  "initialize",
					"params": map[string]any{
						"protocolVersion": handshakeExploreProtocolVersion,
						"capabilities":    map[string]any{},
						"clientInfo": map[string]any{
							"name":    handshakeExploreClientName,
							"version": handshakeExploreClientVersion,
						},
					},
				},
				{
					"jsonrpc": "2.0",
					"id":      2,
					"method":  "tools/list",
				},
				{
					"jsonrpc": "2.0",
					"id":      3,
					"method":  "tools/call",
					"params": map[string]any{
						"name": "codegraph_explore",
						"arguments": map[string]any{
							"query": handshakeExploreQuery,
						},
					},
				},
			},
			ExpectTools: 1,
		},

		// --- tools/list variants (D-05's three, plus a determinism probe) ---

		{
			Name:  "toolslist-default",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 1, // explore-only default (MCP-01)
		},
		{
			Name:  "toolslist-allowlist",
			Index: true,
			Env:   []string{envAllowlistNodeStatus},
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 3, // explore + node + status
		},
		{
			// Index: false — fixture copied but never indexed (MCP-03: no
			// .codegraph/ means zero tools). initialize still succeeds;
			// only tools/list is affected.
			Name:  "toolslist-no-index",
			Index: false,
			Env:   []string{envAllowlistNodeStatus},
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 0,
		},
		{
			// Two consecutive tools/list requests (ids 2 and 3) in one
			// session — the deterministic-ordering probe the 2026-07-28
			// changelog's minor change #3 makes relevant
			// (TestToolsListOrderIsDeterministic). Both are "tools/list",
			// not "tools/call", so both are handled synchronously in
			// request order — no worker-pool race (see the concurrency
			// ordering constraint documented above Scenarios()).
			Name:  "toolslist-repeat",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
				toolsListRequest(3),
			},
			ExpectTools: 8, // explore + all 7 companions
		},

		// --- one tools/call per remaining tool (codegraph_explore is
		// covered by handshake-explore above; TestEveryRegisteredTool-
		// HasASuccessfulCallScenario asserts that coverage structurally
		// rather than inheriting it by prose) ---

		{
			Name:  "call-node",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_node", map[string]any{"symbol": fixtureFuncAlpha}),
			},
			ExpectTools: 8,
		},
		{
			Name:  "call-search",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_search", map[string]any{"query": fixtureFuncAlpha}),
			},
			ExpectTools: 8,
		},
		{
			// Beta is called by Alpha (pkga.go) — a non-empty callers
			// result.
			Name:  "call-callers",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_callers", map[string]any{"symbol": fixtureFuncBeta}),
			},
			ExpectTools: 8,
		},
		{
			// Alpha calls Beta (intra-package) and Helper (cross-package)
			// — a non-empty, two-entry callees result.
			Name:  "call-callees",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_callees", map[string]any{"symbol": fixtureFuncAlpha}),
			},
			ExpectTools: 8,
		},
		{
			Name:  "call-impact",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_impact", map[string]any{"symbol": fixtureFuncBeta}),
			},
			ExpectTools: 8,
		},
		{
			// pattern/filter/depth/format are all optional — an empty
			// arguments object is a real, successful call, not a degraded
			// one.
			Name:  "call-files",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_files", map[string]any{}),
			},
			ExpectTools: 8,
		},
		{
			// codegraph_status takes no required arguments.
			Name:  "call-status",
			Index: true,
			Env:   []string{envAllowlistAllCompanions},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_status", map[string]any{}),
			},
			ExpectTools: 8,
		},

		// --- four error shapes ---

		{
			Name:  "error-unknown-method",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				{
					"jsonrpc": "2.0",
					"id":      2,
					"method":  unimplementedMethod,
				},
			},
			ExpectTools: 1,
		},
		{
			// codegraph_bogus_tool is never registered under any allowlist
			// — handleToolCall's "tool not found" path
			// (server.go:1936-1942), mcp.INVALID_PARAMS (-32602). No
			// hand-authored anchor exists for this scenario (only
			// error-unknown-method and error-malformed-args are anchored,
			// per Task 3) — it is captured-and-frozen only.
			Name:  "error-unknown-tool",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_bogus_tool", map[string]any{}),
			},
			ExpectTools: 1,
		},
		{
			// error-malformed-args exercises a genuinely malformed
			// tools/call request and freezes whatever mark3labs v0.56.0
			// actually returns for it — NOT a "wrong-type argument value on
			// a registered tool". That shape (verified by hand against the
			// built binary this session: e.g. codegraph_search with
			// {"query":123}, or a non-object "arguments" value sent
			// against a real registered tool name) never reaches a
			// JSON-RPC-level error at all — every companion/explore
			// handler converts its own RequireString/RequireInt failure
			// into mcp.NewToolResultError, a SUCCESSFUL JSON-RPC response
			// with result.isError=true, because BuildServer never enables
			// WithInputSchemaValidation (server.go:1990's opt-in-only
			// schema validator; confirmed nil here). The ONLY
			// request_handler.go path that returns mcp.INVALID_PARAMS for
			// tools/call is "tool '%s' not found" (handleToolCall,
			// server.go:1936-1942) — the SAME underlying mechanism
			// error-unknown-tool exercises. This scenario reaches that
			// path via a distinct, still genuinely malformed shape:
			// params.name omitted (empty string) AND params.arguments sent
			// as a bare JSON string instead of an object — a structurally
			// malformed tools/call request that resolves to "tool '' not
			// found" => mcp.INVALID_PARAMS (-32602), matching Task 3's
			// hand-authored anchor. Deviation from this plan's original
			// "wrong JSON type argument on a registered tool" framing
			// (Rule 1 — the assumed code path does not exist in mark3labs
			// v0.56.0 as this server configures it; fixed to the real path
			// that actually produces the required -32602, verified
			// empirically this session and documented in
			// 01-04-SUMMARY.md).
			Name:  "error-malformed-args",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "", "not-an-object"),
			},
			ExpectTools: 1,
		},
		{
			// codegraph_node with a "path" argument resolving outside the
			// index root — exercises confineToRepoRoot
			// (internal/mcp/tools.go:24-45), the CR-02 trust boundary.
			Name:  "error-confinement-reject",
			Index: true,
			Env:   []string{envAllowlistNodeOnly},
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_node", map[string]any{"path": outsideRepoRootPath}),
			},
			ExpectTools: 2, // explore + node
		},

		// --- one session-ordering edge (counted separately from the four
		// error shapes above, never as a fifth error) ---

		{
			// A tools/call sent as the very first message, with NO prior
			// initialize.
			//
			// RETRACTED (02-05, Task 1 checkpoint): this scenario's
			// original doc comment (01-04-PLAN Task 2, RESEARCH Pitfall 2)
			// called mark3labs' non-gating "a currently-passing behavior
			// being locked in, not an error" and "an accidental but real
			// asset for Phase 3's statelessness work." That framing is
			// false post-migration and is not carried forward.
			//
			// Post-migration (go-sdk@v1.7.0): the server now REJECTS this
			// request — {"code":0,"message":"method \"tools/call\" is
			// invalid during session initialization"} — because go-sdk
			// enforces MCP's own ordering requirement that initialize
			// precede other requests. mark3labs' permissiveness was the
			// spec deviation, not this. The maintainer accepted this as
			// cause #9 of 02-05's re-freeze (semantic, not cosmetic):
			// spec-correct, zero known blast radius against the 8-agent
			// roster (VRFY-05 audit surfaced no client calling tools
			// before initializing). This scenario now freezes go-sdk's
			// session-ordering enforcement, not mark3labs' absence of it.
			//
			// The error code itself (0) is not a defined JSON-RPC integer
			// error code — tracked as an upstream go-sdk defect
			// (github.com/modelcontextprotocol/go-sdk#976), not
			// codegraph-go's to fix. No anchor exists for it in
			// anchors.go and none should be added; do not work around it.
			// Do not add server code for this — only the scenario changed
			// (01-04-PLAN Task 2 origin; 02-05 Task 2 retraction).
			Name:         "edge-call-before-initialize",
			Index:        true,
			NoInitialize: true,
			Requests: []map[string]any{
				toolCallRequest(1, "codegraph_explore", map[string]any{"query": fixtureFuncAlpha}),
			},
			// No initialize means the VRFY-03 AddAfterInitialize hook
			// never fires — no session line is ever emitted for this
			// scenario (oracle_test.go gates the assertion on
			// NoInitialize rather than reading this field as a real tool
			// count).
			ExpectTools: 0,
		},

		// --- six-era Legacy handshake baseline (01-05-PLAN Task 1
		// checkpoint: six-era) — what today's pre-migration
		// mark3labs-backed server answers at each protocol revision it
		// recognizes, plus the revision Phase 3 will implement
		// (unsupported today), plus a request that omits protocolVersion
		// entirely. This is the phase's second and last one-way capture:
		// once Phase 2 removes mark3labs/mcp-go from go.mod, none of
		// these six handshakes can ever be replayed again. Every
		// scenario here sends exactly one initialize followed by one
		// tools/list, so the frozen transcript records both the
		// negotiated version and whether the session stayed usable at
		// that revision. ---

		{
			Name:                 legacyEraPrefix + "2025-11-25",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyEraVersions[0],
			EraNegotiatedVersion: legacyEraVersions[0],
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyEraVersions[0]),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},
		{
			Name:                 legacyEraPrefix + "2025-06-18",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyEraVersions[1],
			EraNegotiatedVersion: legacyEraVersions[1],
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyEraVersions[1]),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},
		{
			Name:                 legacyEraPrefix + "2025-03-26",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyEraVersions[2],
			EraNegotiatedVersion: legacyEraVersions[2],
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyEraVersions[2]),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},
		{
			Name:                 legacyEraPrefix + "2024-11-05",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyEraVersions[3],
			EraNegotiatedVersion: legacyEraVersions[3],
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyEraVersions[3]),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},
		{
			// Silent coercion, not rejection (RESEARCH Pitfall 1): today's
			// server has no version-rejection path. This is a SUCCESSFUL
			// initialize result negotiating the server's own latest —
			// never a JSON-RPC error. No error-code anchor exists for
			// this scenario (anchors.go).
			Name:                 legacyEraPrefix + "unsupported-2026-07-28",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyUnsupportedVersion,
			EraNegotiatedVersion: legacyEraVersions[0], // server's own latest
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyUnsupportedVersion),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},
		{
			// A distinct, third coercion path (RESEARCH Pitfall 1, D-06's
			// discretionary sixth scenario): an initialize whose params
			// carry NO protocolVersion key negotiates the server's older
			// backwards-compat default, not its latest. Also a SUCCESS,
			// not an error.
			Name:                 legacyEraPrefix + "omitted-version",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    "",
			EraNegotiatedVersion: legacyOmittedVersionCoercion,
			Requests: []map[string]any{
				initializeRequestOmittingVersion(1),
				toolsListRequest(2),
			},
			ExpectTools: 1,
		},

		// --- Modern (2026-07-28) tracer: server/discover then a
		// sessionless tools/call, both proved in one capture (phase 3
		// plan 01's tracer) ---

		{
			// One capture, two proofs, deliberately kept as a single
			// scenario rather than two: the discover response carries
			// SPEC-01's capabilities-without-a-tool-call, SPEC-03's
			// resultType, SPEC-04's cacheScope/ttlMs, and SPEC-08's
			// serverInfo in _meta; the following tools/call proves
			// SPEC-03 and SPEC-08 for a TOOL result, which no
			// pre-existing transcript covers because both fields are
			// per-request Modern-gated and every pre-existing scenario
			// negotiates a Legacy era.
			//
			// NoInitialize: true here means something different from
			// edge-call-before-initialize: a Modern _meta-bearing
			// request is spec-sanctioned sessionless dispatch (SEP-2575),
			// not a session-ordering violation — the server never
			// rejects it, unlike edge-call-before-initialize's classic
			// tools/call with no _meta at all.
			Name:         "modern-discover-explore",
			Index:        true,
			NoInitialize: true,
			Requests: []map[string]any{
				discoverRequest(1),
				modernToolCallRequest(2, "codegraph_explore", map[string]any{"query": handshakeExploreQuery}),
			},
			// ExpectTools is unread for a NoInitialize scenario — it
			// asserts the absence of a session line instead
			// (assertNoSessionLine), never a real tool count.
			ExpectTools: 0,
		},

		// --- SPEC-02 proof (phase 3 plan 03): the two per-request `_meta`
		// failure answers the requirement actually demands, frozen and
		// independently anchored. Both are sessionless (SEP-2575), like
		// modern-discover-explore above — NOT session-ordering violations.
		// See modernUnsupportedVersion's doc comment above for why no
		// `-32601` scenario is added here (that observation is go-sdk's own
		// lexical-comparison classification quirk, retracted from
		// 03-CONTEXT.md's "real gap" framing, not a shape SPEC-02 asks this
		// suite to prove). ---

		{
			// A well-formed Modern `_meta` whose
			// io.modelcontextprotocol/clientCapabilities key is absent —
			// SPEC-02's `-32602` half.
			Name:         "modern-meta-invalid-params",
			Index:        true,
			NoInitialize: true,
			Requests: []map[string]any{
				discoverRequestWithMeta(1, modernMetaMissingCapabilities()),
			},
			ExpectTools: 0,
		},
		{
			// A well-formed Modern `_meta` offering modernUnsupportedVersion,
			// a supported-SHAPE but unrecognized protocol version that sorts
			// lexically AFTER "2026-07-28" — SPEC-02's `-32022` half.
			Name:         "modern-meta-unsupported-version",
			Index:        true,
			NoInitialize: true,
			Requests: []map[string]any{
				discoverRequestWithMeta(1, modernMetaWithVersion(modernUnsupportedVersion)),
			},
			ExpectTools: 0,
		},
	}
}

// ScenarioByName returns the named scenario from Scenarios(), or
// ok=false if no scenario has that name.
func ScenarioByName(name string) (Scenario, bool) {
	for _, sc := range Scenarios() {
		if sc.Name == name {
			return sc, true
		}
	}
	return Scenario{}, false
}

// TranscriptPath returns the path to name's frozen golden transcript,
// relative to this package's own directory.
func TranscriptPath(name string) string {
	return filepath.Join("..", "..", "testdata", "wireoracle", "transcripts", name+".golden")
}
