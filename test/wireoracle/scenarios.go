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

// legacyOmittedVersionCoercion is what mark3labs v0.56.0 negotiates when a
// client's initialize params carry NO protocolVersion key at all: the
// server's own backwards-compat default, applied BEFORE the
// ValidProtocolVersions check runs
// [VERIFIED: github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1196-1198].
// A THIRD, structurally distinct coercion path from the
// unsupported-version case above — not the server's own latest, and not
// the value the client (didn't) offer.
const legacyOmittedVersionCoercion = "2025-03-26"

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
// handshake baseline) = 23. A shrinking count is the failure mode this
// constant exists to catch.
const ExpectedScenarioCount = 23

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

		// --- one statelessness edge (counted separately from the four
		// error shapes above, never as a fifth error: RESEARCH Pitfall 2 —
		// mark3labs never gates tools/list/tools/call on Initialized(), so
		// this is a currently-passing behavior being locked in, not an
		// error) ---

		{
			// A tools/call sent as the very first message, with NO prior
			// initialize. Today's server tolerates this — an accidental
			// but real asset for Phase 3's statelessness work (RESEARCH
			// Pitfall 2, verified by hand against the built binary this
			// session: the explore call succeeds with no initialize
			// handshake at all). Do not add server code for this — only
			// the scenario (01-04-PLAN Task 2).
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
