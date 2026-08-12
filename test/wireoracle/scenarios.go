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

// Narrowing-filter env strings the scenarios below drive
// CODEGRAPH_MCP_TOOLS with (internal/mcp/server.go's companionNames).
//
// The variable is a NARROWING filter, not an allowlist: unset means all
// eight tools register, and setting it removes every companion it does not
// name. There is deliberately no "all companions" constant any more — under
// the old opt-in semantics that value was how a scenario asked for the full
// surface, and keeping it would now be an elaborate way of spelling "the
// default", hiding whether a scenario exercises the default path or an
// explicit filter that happens to select everything.
//
// envFilterExploreOnly is the SET-but-EMPTY value. It is not a degenerate
// case to be tidied away: it is the only wire shape that distinguishes an
// unset variable from a set one, and it is how an operator restores the
// pre-inversion explore-only surface.
const (
	envFilterNodeStatus  = "CODEGRAPH_MCP_TOOLS=node,status"
	envFilterNodeOnly    = "CODEGRAPH_MCP_TOOLS=node"
	envFilterExploreOnly = "CODEGRAPH_MCP_TOOLS="
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

// resourcesListRequest returns a bare "resources/list" request with the
// given id — resourcesListRequest/toolsListRequest's sibling.
func resourcesListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "resources/list",
	}
}

// resourceReadRequest returns a "resources/read" request naming a URI —
// toolCallRequest's sibling for the resources method family.
func resourceReadRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "resources/read",
		"params": map[string]any{
			"uri": uri,
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

// notificationSubscriptionsAcknowledgedMethod and
// notificationToolsListChangedMethod are hand-authored, package-local
// JSON-RPC method-name literals (VRFY-01) — never imported from the SDK
// under test. They carry the same wire values as go-sdk@v1.7.0's own
// unexported notificationSubscriptionsAck and notificationToolListChanged
// constants [VERIFIED: github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:2350-2353],
// transcribed here independently, matching this file's existing pattern for
// every other spec-pinned literal (codeMethodNotFound, modernProtocolVersion,
// etc.).
const (
	notificationSubscriptionsAcknowledgedMethod = "notifications/subscriptions/acknowledged"
	notificationToolsListChangedMethod          = "notifications/tools/list_changed"
)

// notificationsOptInFieldName is the PLURAL JSON key a Modern
// subscriptions/listen request's params.notifications object uses to opt
// in to tool-catalog-change notifications — go-sdk's
// NotificationSubscriptions.ToolsListChanged field,
// `json:"toolsListChanged,omitempty"`
// [VERIFIED: github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:2071-2073].
//
// The plural spelling is load-bearing, not cosmetic (05-CONTEXT.md D-02): a
// client that sends the SINGULAR "toolListChanged" is silently ACCEPTED by
// the server — answered with a subscriptionId and no error anywhere — and
// acknowledged with an EMPTY subscription set, `"notifications":{}`,
// because go-sdk's NotificationSubscriptions struct simply has no field
// that spelling ever matches, so nothing is ever granted. A client that
// types the singular by mistake therefore gets a successful-looking
// subscription that will never deliver a notification, and the
// acknowledgment echo (assertSubscriptionAckEcho, anchors.go) is the ONLY
// discriminator available between that dead subscription and a live one —
// Task 3's mutation 6 (MUTATION-PROOF.md) reproduces this exact shape
// deliberately, by removing the AwaitAfterRequest wait rather than by
// misspelling this constant, so the constant itself always stays correct.
const notificationsOptInFieldName = "toolsListChanged"

// modernToolsListRequest returns a "tools/list" request whose params carry
// modernMetaParams() — the Modern (SEP-2575) sessionless counterpart to
// toolsListRequest, used by modern-listen-catalog-change so every request
// in that scenario carries Modern `_meta` (see that scenario's own doc
// comment for why this is required, not stylistic).
func modernToolsListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": modernMetaParams(),
		},
	}
}

// subscriptionsListenRequest returns a "subscriptions/listen" request whose
// params carry modernMetaParams() plus a "notifications" object opting
// into optIn. optIn is parameterized — rather than hardcoding
// notificationsOptInFieldName inline — specifically so Task 3's dead-opt-in
// mutation (MUTATION-PROOF.md mutation 6) is a one-token edit at a single
// call site.
func subscriptionsListenRequest(id int, optIn string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "subscriptions/listen",
		"params": map[string]any{
			"_meta": modernMetaParams(),
			"notifications": map[string]any{
				optIn: true,
			},
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
// + 1 scenario (phase 3 plan 04's SPEC-05 proof: index-appears-mid-session,
// a real `codegraph init` run mid-session against the same connected
// server, driving an empty tools/list to a one-tool tools/list on the same
// connection) + 1 scenario (phase 5 plan 01's SPEC-09 proof:
// modern-listen-catalog-change, an opted-in Modern subscriptions/listen
// stream observing notifications/tools/list_changed on the same live
// connection after a real mid-session `codegraph init`) + 1 scenario
// (toolslist-filter-empty, added when CODEGRAPH_MCP_TOOLS inverted from an
// opt-in allowlist to an opt-out narrowing filter: the SET-but-EMPTY value,
// the only wire shape that distinguishes a set variable from an unset one,
// and therefore the frozen proof that the server reads os.LookupEnv rather
// than os.Getenv) + 2 scenarios (phase 5 plan 01's RSRC-01/02/03 wire
// proof: resources-list and resources-read-explore, the first
// resources/list and resources/read scenarios in this corpus, anchored
// by assertResourceCacheControl's cacheScope:"private" check) = 31. A
// shrinking count is the failure mode this constant exists to catch.
//
// Note that toolslist-allowlist was RENAMED to toolslist-narrowed in the
// same change, not removed — a rename moves a .golden file's name without
// moving this count.
const ExpectedScenarioCount = 31

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
			ExpectTools: 8,
		},

		// --- tools/list variants: the four points of the narrowing-filter
		// matrix (unset / named subset / set-empty / no index), plus a
		// determinism probe. Re-derived when CODEGRAPH_MCP_TOOLS inverted
		// from an opt-in allowlist to an opt-out narrowing filter — the
		// scenario NAMES moved with the semantics rather than the values
		// being patched under names that no longer described them. ---

		{
			// The default surface: CODEGRAPH_MCP_TOOLS unset, index
			// present, all eight tools. This scenario existed before the
			// inversion and froze the opposite answer (one tool) under the
			// same name — it is the single case the reported "the mcp
			// server is only showing one tool" bug landed on, and the one
			// whose frozen bytes prove the fix on the wire rather than in a
			// unit test.
			Name:  "toolslist-default",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 8, // explore + all 7 companions, no env set
		},
		{
			// Renamed from toolslist-allowlist: the value and the answer
			// are unchanged (explore + node + status), but the MECHANISM
			// reversed — these two companions now survive a filter instead
			// of being opted into an empty set. The observable set is
			// identical under both contracts, which is precisely why this
			// scenario cannot detect the inversion and the two below can.
			Name:  "toolslist-narrowed",
			Index: true,
			Env:   []string{envFilterNodeStatus},
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 3, // explore + node + status
		},
		{
			// CODEGRAPH_MCP_TOOLS SET to the empty string — the operator
			// escape hatch back to the pre-inversion explore-only surface,
			// and the only wire shape that distinguishes a set variable
			// from an unset one. Paired with toolslist-default above, these
			// two transcripts are the frozen proof that the server reads
			// the variable's PRESENCE (os.LookupEnv) and not merely its
			// value: an implementation that used os.Getenv would produce
			// byte-identical tools arrays for both, and this pair is what
			// makes that indistinguishable state a loud failure.
			Name:  "toolslist-filter-empty",
			Index: true,
			Env:   []string{envFilterExploreOnly},
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
			},
			ExpectTools: 1, // explore alone
		},
		{
			// Index: false — fixture copied but never indexed (MCP-03: no
			// .codegraph/ means zero tools). initialize still succeeds;
			// only tools/list is affected. The filter is set here so this
			// scenario proves the index gate dominates the filter gate:
			// neither a narrowed nor a default surface survives a missing
			// index.
			Name:  "toolslist-no-index",
			Index: false,
			Env:   []string{envFilterNodeStatus},
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
			//
			// It carries no Env: the full eight-tool surface it needs to
			// make an ordering probe meaningful is now the default. It used
			// to name all seven companions explicitly, which after the
			// inversion would have been a filter selecting everything — a
			// second, silently divergent way of spelling the default.
			Name:  "toolslist-repeat",
			Index: true,
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
		// rather than inheriting it by prose).
		//
		// None of these carries an Env any more: each used to name all
		// seven companions to opt the tool it calls INTO the surface, and
		// after the inversion the tool it calls is registered by default.
		// Keeping the old value would have turned every one of these into a
		// filter that selects everything — a second spelling of the default
		// that would keep passing if the default silently regressed. ---

		{
			Name:  "call-node",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_node", map[string]any{"symbol": fixtureFuncAlpha}),
			},
			ExpectTools: 8,
		},
		{
			Name:  "call-search",
			Index: true,
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
			Requests: []map[string]any{
				initializeRequest(1),
				toolCallRequest(2, "codegraph_callees", map[string]any{"symbol": fixtureFuncAlpha}),
			},
			ExpectTools: 8,
		},
		{
			Name:  "call-impact",
			Index: true,
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
			ExpectTools: 8,
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
			ExpectTools: 8,
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
			ExpectTools: 8,
		},
		{
			// codegraph_node with a "path" argument resolving outside the
			// index root — exercises confineToRepoRoot
			// (internal/mcp/tools.go:24-45), the CR-02 trust boundary.
			Name:  "error-confinement-reject",
			Index: true,
			Env:   []string{envFilterNodeOnly},
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
			ExpectTools: 8,
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
			ExpectTools: 8,
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
			ExpectTools: 8,
		},
		{
			// SPEC-06 (03-05-PLAN Task 1): a trailing tools/call proves this
			// scenario — the OLDEST Legacy era, maximal distance from
			// Modern — completes a session AND successfully calls a tool,
			// not merely negotiates. handshake-explore already proves the
			// same at 2025-11-25, the NEWEST Legacy era. go-sdk's
			// callTool/toolForErr never consult the request's protocol
			// version once initialize has succeeded, so one success at
			// each end of the Legacy range is judged sufficient evidence
			// for the three eras between (2025-06-18, 2025-03-26, and the
			// two coercion scenarios below, which stay negotiation-only).
			// This is the judgment call 03-RESEARCH.md's "SPEC-06
			// recommendation" flagged LOW confidence, not a proof —
			// recorded as such deliberately, not overstated.
			Name:                 legacyEraPrefix + "2024-11-05",
			Index:                true,
			EraScenario:          true,
			EraOfferedVersion:    legacyEraVersions[3],
			EraNegotiatedVersion: legacyEraVersions[3],
			Requests: []map[string]any{
				initializeRequestWithVersion(1, legacyEraVersions[3]),
				toolsListRequest(2),
				toolCallRequest(3, "codegraph_explore", map[string]any{"query": handshakeExploreQuery}),
			},
			ExpectTools: 8,
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
			ExpectTools: 8,
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
			ExpectTools: 8,
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

		// --- SPEC-05 proof (phase 3 plan 04): the live tool catalog. A
		// real `codegraph init` runs mid-session, against the SAME
		// already-running server process, and the transcript proves the
		// tool set follows it in both request order — no restart, no
		// reconnect. ---

		{
			// Index: false — the fixture is copied but never indexed
			// before the server starts (mirroring toolslist-no-index
			// above), so the id-2 tools/list proves the pre-index empty
			// catalog. InitAfterRequest: 2 then runs `codegraph init`
			// against this exact server's working directory once (and
			// only once) the id-2 response has actually been observed on
			// stdout — never a sleep, never a fixed delay — before the
			// id-3 request is even written to stdin. The id-3 tools/list,
			// answered by the SAME connected server process, proves the
			// re-check (internal/mcp/server.go's recheckCatalog,
			// 03-04-PLAN.md Task 1) reacted to a REAL index appearing on
			// disk, not to a test-only hook: the plan's own discretion
			// item (03-CONTEXT.md "Whether the index-appears scenario
			// drives real codegraph init in a fixture or simulates the
			// transition") is resolved in favor of the real form, because
			// a simulated transition would only prove the mechanism
			// reacts to whatever the simulation calls, not to the
			// filesystem state SPEC-05 actually promises to track.
			//
			// It carries no tools/call, so the at-most-one-and-last
			// worker-pool ordering constraint documented above Scenarios()
			// is trivially satisfied — all three requests (initialize,
			// tools/list, tools/list) are handled synchronously in
			// request order.
			//
			// ExpectTools: 0 is deliberate, not a placeholder: the VRFY-03
			// session line is written during the id-1 initialize, BEFORE
			// codegraph init has run (InitAfterRequest fires after id-2,
			// not id-1) — so 0 is the honest, correct expectation, and
			// asserting it pins tools=N's now-point-in-time meaning
			// (T-03-17) rather than letting a stale expectation quietly
			// paper over what changed.
			Name:             "index-appears-mid-session",
			Index:            false,
			InitAfterRequest: 2,
			Requests: []map[string]any{
				initializeRequest(1),
				toolsListRequest(2),
				toolsListRequest(3),
			},
			ExpectTools: 0,
		},

		// --- SPEC-09 proof (phase 5 plan 01): live tool-catalog change
		// notification. An opted-in Modern subscriptions/listen stream
		// receives notifications/tools/list_changed on the SAME live stdio
		// connection after a real mid-session `codegraph init` mutates the
		// catalog — Phase 3's per-request re-check (recheckCatalog) and
		// this phase's own subscription plumbing observed CONNECTED on the
		// wire for the first time. 05-CONTEXT.md D-01 establishes both
		// halves already worked independently; nobody had observed them
		// wired together until this scenario. ---

		{
			// NoInitialize with Modern `_meta` on EVERY request is
			// REQUIRED, not stylistic — this is the single easiest way to
			// destroy this proof (05-CONTEXT.md D-01/key_links). go-sdk's
			// notifySessions routes any session whose InitializeParams are
			// nil OR whose negotiated version is below "2026-07-28" onto
			// the Legacy shared channel, which receives
			// notifications/tools/list_changed with NO opt-in at all. A
			// scenario opening with a classic `initialize` would therefore
			// receive the notification regardless of whether its
			// subscription is actually live — a vacuous proof that would
			// pass even with a dead opt-in. Modern `_meta`-bearing
			// sessionless dispatch on every request is what sets this
			// session's InitializeParams to "2026-07-28" and routes it
			// onto the subscriber-only path, making the opt-in the SOLE
			// delivery mechanism this scenario can ever observe — Task 3's
			// mutations 5 and 6 (MUTATION-PROOF.md) prove that path is
			// live, not merely present.
			//
			// Index: false plus an EXPLICIT explore-only narrowing filter is
			// deliberate: exactly one tool (codegraph_explore) registers on
			// the mid-session transition, so changeAndNotify's 10ms debounce
			// coalesces to exactly one notification frame — a count that does
			// not depend on how many AddTool calls happen to land inside that
			// debounce window.
			//
			// The filter is what preserves that property, and it was added
			// when CODEGRAPH_MCP_TOOLS inverted. Before the inversion this
			// scenario got its single AddTool call for free from the
			// explore-only default and named no env at all; under
			// default-all the same scenario would issue eight AddTool calls
			// and the frame count would become a bet on the debounce window
			// swallowing all eight. Setting envFilterExploreOnly keeps the
			// one-frame count a structural property of the scenario rather
			// than a timing observation that happens to hold today.
			//
			// Request 1 (subscriptions/listen) is answered with an
			// acknowledgment NOTIFICATION, never an id-matched result
			// frame, on this transport (05-CONTEXT.md D-01, MEASURED 5/5
			// isolated runs): go-sdk's SubscriptionsListenResult is sent
			// only on graceful subscription teardown, and stdin EOF is not
			// that path — hence NoResponseRequests: []int{1}, asserted as
			// an exactly-zero claim by assertFramingInvariant
			// (oracle_test.go), never a blind spot. AwaitAfterRequest maps
			// index 1 to the acknowledgment method (go-sdk calls
			// jsonrpc2.Async for every call except "initialize", so the ack
			// frame races whatever is written next) and index 3 — the LAST
			// request — to the tools-list-changed method, which keeps
			// stdin open long enough for changeAndNotify's debounce timer
			// to fire before Capture closes it: closing stdin immediately
			// after the mutating request would let process shutdown race
			// that timer and silently drop the notification from the
			// transcript, a quietly weaker proof rather than a loud
			// failure.
			//
			// Measured deterministic line order (05-01-PLAN, 5/5 runs):
			// acknowledgment, id-2 result (tools:[]), id-3 result
			// (tools:[codegraph_explore]), notifications/tools/list_changed.
			//
			// It carries no tools/call, so the at-most-one-and-last
			// worker-pool ordering constraint documented above Scenarios()
			// is trivially satisfied.
			Name:               "modern-listen-catalog-change",
			Index:              false,
			Env:                []string{envFilterExploreOnly},
			NoInitialize:       true,
			NoResponseRequests: []int{1},
			InitAfterRequest:   2,
			AwaitAfterRequest: map[int]string{
				1: notificationSubscriptionsAcknowledgedMethod,
				3: notificationToolsListChangedMethod,
			},
			Requests: []map[string]any{
				subscriptionsListenRequest(1, notificationsOptInFieldName),
				modernToolsListRequest(2),
				modernToolsListRequest(3),
			},
			// ExpectTools is unread for a NoInitialize scenario (see
			// modern-discover-explore above) — it asserts the absence of a
			// session line instead (assertNoSessionLine), never a real
			// tool count.
			ExpectTools: 0,
		},

		// --- MCP Resources (2) — plan 05-01. Both scenarios carry
		// exactly one async request and it is last — a VERIFIED
		// constraint, not the [ASSUMED] one 05-RESEARCH.md A2 recorded:
		// go-sdk dispatches every call except "initialize" through
		// jsonrpc2.Async(ctx) (server.go:1910-1915), so resources/list
		// and resources/read race each other and race any request queued
		// after them exactly as tools/call does — the same worker-pool
		// ordering constraint documented above Scenarios() for
		// tools/call, and the same class of flake already open as a
		// MAJOR todo in STATE.md for toolslist-repeat. ---
		{
			Name:  "resources-list",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				resourcesListRequest(2),
			},
			ExpectTools: 8,
		},
		{
			Name:  "resources-read-explore",
			Index: true,
			Requests: []map[string]any{
				initializeRequest(1),
				resourceReadRequest(2, "codegraph://tools/explore"),
			},
			ExpectTools: 8,
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
