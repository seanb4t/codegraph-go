package wireoracle

import (
	"bytes"
	"encoding/json"
	"testing"
)

// codeMethodNotFound and codeInvalidParams are spec-pinned JSON-RPC error
// codes (D-02): properties of the JSON-RPC/MCP spec itself, not of
// whichever SDK happens to be serving the wire — an anchor asserting
// against these constants fails independently of the captured bulk if a
// migration ever moved something the spec pins.
//
// Verified against github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:455,458
// (01-RESEARCH.md "Code Examples", "these are the two codes D-02 can safely
// hand-author as spec anchors independent of capture"). Hand-authored here,
// never imported from the SDK under test.
const (
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// codeUnsupportedProtocolVersion is a third spec-pinned JSON-RPC error code
// (D-02's pattern, extended by phase 3 plan 03): a property of SEP-2575 and
// the MCP spec itself, not of whichever SDK happens to be serving the wire —
// an anchor asserting against this constant fails independently of the
// captured bulk if a migration ever moved something the spec pins.
//
// Verified against github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:1872-1877
// (03-RESEARCH.md "Code Examples", case 2's empirically captured response).
// Hand-authored here, never imported from the SDK under test — the SDK's
// own equivalently-named exported error-code constant matches
// internal/mcp/archtest's VRFY-02 `(?i)protocol.?version` name heuristic
// (which has no code-level allowlist), so referencing it directly would
// trip that guard in addition to breaking VRFY-01 (never use the SDK under
// test as the source of its own expected values).
const codeUnsupportedProtocolVersion = -32022

// Anchor is one hand-authored, spec-pinned assertion, run against a named
// scenario's freshly captured stdout independently of the frozen transcript
// (an anchor read from the golden file would be circular — TestSpecAnchorsHold
// re-captures rather than reading testdata/wireoracle/transcripts/*.golden).
type Anchor struct {
	// Scenario is the name of the Scenarios() entry this anchor checks.
	Scenario string
	// Name describes what this anchor asserts, for -v test output and
	// failure messages.
	Name string
	// Assert decodes only the one field this anchor names out of the raw
	// captured stdout — never by unmarshalling into an SDK type — and
	// fails the test if it does not hold.
	Assert func(t *testing.T, stdout []byte)
}

// Anchors returns the hand-authored spec anchor set (D-02) that needs
// nothing beyond a plain int comparison: the two JSON-RPC error codes.
//
// The THIRD spec anchor this plan's Task 3 also requires — handshake-explore's
// initialize result.protocolVersion == internal/mcp.ProtocolVersion — is
// deliberately NOT modeled as an Anchor value here. internal/mcp is package
// mcp, and that package's OTHER files (server.go, tools.go) import
// github.com/mark3labs/mcp-go; Go resolves imports at the package level, so
// even a reference to the single SDK-free symbol
// internal/mcp.ProtocolVersion would transitively pull the SDK under test
// into this package's own dependency graph — breaking this same task's own
// acceptance criterion ("go list -deps ... test/wireoracle contains no line
// under the MCP SDK's module path", VRFY-01). assertProtocolVersionAnchor
// therefore stays exactly where the tracer (plan 01) put it — oracle_test.go,
// a _test.go file, whose imports are invisible to `go list -deps` on the
// plain (non -test) package — and TestSpecAnchorsHold calls it directly for
// handshake-explore rather than through this Anchors() slice.
//
// No anchor exists for the six-era Legacy baseline's
// "legacy-unsupported-2026-07-28" scenario (plan 05's D-06 multi-era
// baseline), and this omission is STILL correct today, unlike the
// paragraph this one replaces: that scenario is a classic `initialize`
// (no `_meta` at all) driven against a client offering "2026-07-28" as
// `params.protocolVersion`, and today's go-sdk@v1.7.0 server silently
// coerces that unrecognized value to its own latest supported revision
// rather than returning an error
// [VERIFIED: 01-RESEARCH.md Pitfall 1, reconfirmed unchanged post-migration
// by 02-RESEARCH.md] — asserting an error code there would assert a
// behavior that never fires on that scenario's own wire shape. It remains
// captured-and-frozen as a SUCCESS; do not "fix" this omission by adding an
// anchor to it.
//
// This is a DIFFERENT scenario, and a DIFFERENT outcome, from
// modern-meta-unsupported-version below (phase 3 plan 03): that scenario
// sends a well-formed Modern `_meta` object whose
// io.modelcontextprotocol/protocolVersion is modernUnsupportedVersion
// ("2099-01-01", chosen because it sorts lexically AFTER "2026-07-28" — see
// its doc comment in scenarios.go), which DOES return an error
// (`-32022`) under go-sdk@v1.7.0, and IS anchored, below. A prior version
// of this comment described no unsupported-version anchor existing at all
// — that was true only for mark3labs v0.56.0's classic-initialize path and
// is now retracted for the Modern `_meta` path; do not re-merge the two
// scenarios' outcomes into one paragraph, they are structurally distinct
// (see modernUnsupportedVersion's own doc comment for the full trace of
// why offering a lexically-SMALLER version instead would land back on
// legacy-unsupported-2026-07-28's silent-coercion territory by a totally
// different mechanism — a `-32601` availability-gate rejection, not a
// version-negotiation coercion — and must still never gain an anchor
// either).
//
// The framing invariant (every stdout line is jsonrpc:"2.0", every request
// id has exactly one response line) is also checked outside this slice, in
// TestSpecAnchorsHold, across every scenario in Scenarios() — it is not
// "the one field a scenario's own captured line carries," it is a
// cross-cutting structural check that applies uniformly to every scenario.
//
// modern-discover-explore's discover cache-control check (SPEC-04) exists
// alongside the frozen transcript for a reason distinct from redundancy:
// the transcript proves the captured bytes did not change, while the
// anchor proves the specific spec-pinned property still holds against a
// FRESH capture, so a wholesale transcript regeneration (which replaces
// the golden file's bytes wholesale, D-06) cannot launder a regression in
// this property past the byte-comparison test. The same reasoning applies
// to the two `_meta`-failure anchors below.
//
// The four SPEC-09 anchors below (phase 5 plan 01) exist for the identical
// reason. assertToolsListChangedCapability is registered TWICE — once
// against handshake-explore's Legacy `initialize` response, once against
// modern-discover-explore's Modern `server/discover` response — because
// SPEC-09 criterion 1 says the server advertises `tools.listChanged: true`,
// not that one negotiation path does; a regression that only broke ONE
// path would be invisible to an anchor checking only the other.
// assertSubscriptionAckEcho and assertToolsListChangedNotification exist
// for the same "fresh capture, not frozen bytes" reason as the cache-control
// anchor, plus a second one specific to this plan: they are the ONLY
// mechanism proving the D-02 acknowledgment-echo discriminator and the T-05-02
// content-free property stay true going forward — a byte-for-byte transcript
// match alone cannot distinguish "this property holds" from "this property
// happened to hold in the one transcript that was frozen."
//
// NOTE (05-01-PLAN Task 2 deviation, Rule 1): the plan's own action text
// says "Register five Anchor entries" but then describes exactly four —
// the acknowledgment echo, the notification delivery, and the capability
// check registered twice (once per negotiation path). Four Anchor entries
// is what those four descriptions actually produce; a fifth was not
// separately specified anywhere in the plan (must_haves, behavior, or
// action), and inventing one would violate this plan's own calibration
// note ("one wire proof plus its assertions... do not inflate"). Recorded
// here and in 05-01-SUMMARY.md rather than silently reconciled.
func Anchors() []Anchor {
	return []Anchor{
		{
			Scenario: "error-unknown-method",
			Name:     "error.code == method-not-found (-32601)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertErrorCode(t, "error-unknown-method", stdout, 2, codeMethodNotFound)
			},
		},
		{
			Scenario: "error-malformed-args",
			Name:     "error.code == invalid-params (-32602)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertErrorCode(t, "error-malformed-args", stdout, 2, codeInvalidParams)
			},
		},
		{
			Scenario: "modern-discover-explore",
			Name:     "discover cache control: cacheScope == \"private\", ttlMs == 0 (SPEC-04)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertDiscoverCacheControl(t, "modern-discover-explore", stdout)
			},
		},
		{
			Scenario: "modern-meta-invalid-params",
			Name:     "error.code == invalid-params (-32602), SPEC-02",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertErrorCode(t, "modern-meta-invalid-params", stdout, 1, codeInvalidParams)
			},
		},
		{
			Scenario: "modern-meta-unsupported-version",
			Name:     "error.code == unsupported-protocol-version (-32022), SPEC-02",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertErrorCode(t, "modern-meta-unsupported-version", stdout, 1, codeUnsupportedProtocolVersion)
			},
		},
		{
			Scenario: "modern-listen-catalog-change",
			Name:     "acknowledgment echo == {toolsListChanged:true} by set equality, never non-empty (SPEC-09, D-02)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertSubscriptionAckEcho(t, "modern-listen-catalog-change", stdout)
			},
		},
		{
			Scenario: "modern-listen-catalog-change",
			Name:     "exactly one correlated, content-free notifications/tools/list_changed frame (SPEC-09)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertToolsListChangedNotification(t, "modern-listen-catalog-change", stdout)
			},
		},
		{
			Scenario: "handshake-explore",
			Name:     "capabilities.tools.listChanged == true on the Legacy initialize path (SPEC-09 criterion 1)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertToolsListChangedCapability(t, "handshake-explore", stdout, 1)
			},
		},
		{
			Scenario: "modern-discover-explore",
			Name:     "capabilities.tools.listChanged == true on the Modern server/discover path (SPEC-09 criterion 1)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertToolsListChangedCapability(t, "modern-discover-explore", stdout, 1)
			},
		},
		{
			Scenario: "resources-list",
			Name:     "resources/list cache control: cacheScope == \"private\", ttlMs == 0",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertResourceCacheControl(t, "resources-list", stdout)
			},
		},
		{
			Scenario: "resources-read-explore",
			Name:     "resources/read cache control: cacheScope == \"private\", ttlMs == 0",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertResourceCacheControl(t, "resources-read-explore", stdout)
			},
		},
		{
			// -32602, not -32002: SEP-2164 made invalid-params the default
			// error code for a resource-not-found in go-sdk v1.7.0, this
			// repository's other two unknown-target error scenarios
			// (error-unknown-tool, error-malformed-args) already freeze
			// -32602, and a transcript showing -32002 would mean the
			// deprecated MCPGODEBUG=customresnotfounderrcode=1 compatibility
			// flag leaked into the capture environment. The SDK also
			// deliberately returns this SAME error for an unregistered URI
			// and for a registered one whose handler could not find it
			// (server.go:1020-1025) — the client learns nothing about server
			// configuration from the failure. This is T-05-02's mitigation
			// observed on the wire, not asserted from source.
			Scenario: "resources-read-unknown",
			Name:     "error.code == invalid-params (-32602), unregistered resource URI (T-05-02)",
			Assert: func(t *testing.T, stdout []byte) {
				t.Helper()
				assertErrorCode(t, "resources-read-unknown", stdout, 2, codeInvalidParams)
			},
		},
	}
}

// discoverCacheScopePrivate and discoverCacheScopeTTLMs are SPEC-04's two
// hand-authored, spec-pinned expected values (D-02/D-06/VRFY-01) — never
// imported from the SDK under test, exactly as codeMethodNotFound and
// codeInvalidParams above are hand-authored rather than read from
// mcp.CodeMethodNotFound/mcp.CodeInvalidParams.
const (
	discoverCacheScopePrivate = "private"
	discoverCacheScopeTTLMs   = 0
)

// assertDiscoverCacheControl decodes only response id=1's result.cacheScope
// and result.ttlMs fields, field-by-field in assertErrorCode's style, and
// fails naming the scenario and quoting the observed line if cacheScope is
// not discoverCacheScopePrivate or ttlMs is not discoverCacheScopeTTLMs.
// Fails if no id=1 result line is found at all — a missing response must
// never read as a pass.
func assertDiscoverCacheControl(t *testing.T, scenario string, stdout []byte) {
	t.Helper()

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any `json:"id"`
			Result *struct {
				CacheScope string `json:"cacheScope"`
				TTLMs      int    `json:"ttlMs"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != 1 || frame.Result == nil {
			continue
		}
		if frame.Result.CacheScope != discoverCacheScopePrivate {
			t.Fatalf("scenario %q: result.cacheScope = %q, want %q: %q", scenario, frame.Result.CacheScope, discoverCacheScopePrivate, line)
		}
		if frame.Result.TTLMs != discoverCacheScopeTTLMs {
			t.Fatalf("scenario %q: result.ttlMs = %d, want %d: %q", scenario, frame.Result.TTLMs, discoverCacheScopeTTLMs, line)
		}
		return
	}
	t.Fatalf("scenario %q: no id=1 result found in captured stdout — a missing response must never read as a pass", scenario)
}

// assertResourceCacheControl is assertDiscoverCacheControl's direct
// sibling for the resources method family (plan 05-01 Task 3): it decodes
// only the id=2 response frame's result.cacheScope and result.ttlMs — id=2
// because both resources-list and resources-read-explore send
// initializeRequest(1) then their one async request as id 2, unlike
// modern-discover-explore's sessionless id=1 discover call — and fails
// naming the scenario and quoting the observed line unless cacheScope is
// "private" and ttlMs is absent or zero. This pins server.go's
// "resources/list", "resources/read" cacheScope case (Task 1) on the wire,
// so it cannot silently revert to the SDK's "public" default. Fails if no
// id=2 result line is found at all — a missing response must never read
// as a pass.
func assertResourceCacheControl(t *testing.T, scenario string, stdout []byte) {
	t.Helper()

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any `json:"id"`
			Result *struct {
				CacheScope string `json:"cacheScope"`
				TTLMs      int    `json:"ttlMs"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != 2 || frame.Result == nil {
			continue
		}
		if frame.Result.CacheScope != discoverCacheScopePrivate {
			t.Fatalf("scenario %q: result.cacheScope = %q, want %q: %q", scenario, frame.Result.CacheScope, discoverCacheScopePrivate, line)
		}
		if frame.Result.TTLMs != discoverCacheScopeTTLMs {
			t.Fatalf("scenario %q: result.ttlMs = %d, want %d: %q", scenario, frame.Result.TTLMs, discoverCacheScopeTTLMs, line)
		}
		return
	}
	t.Fatalf("scenario %q: no id=2 result found in captured stdout — a missing response must never read as a pass", scenario)
}

// assertErrorCode decodes only the top-level error.code field of the
// response bearing wantID, and fails naming the scenario and quoting the
// observed value if it does not equal want. wantID is explicit rather than
// hardcoded because it varies by scenario shape: every pre-existing
// anchored error scenario opens with an id=1 initialize and carries its
// error at id=2, but modern-meta-invalid-params and
// modern-meta-unsupported-version are NoInitialize sessionless scenarios
// (SEP-2575) whose single request — and therefore their error response —
// is id=1. Fails if no response bearing wantID is found at all — a missing
// response must never read as a pass.
func assertErrorCode(t *testing.T, scenario string, stdout []byte, wantID float64, want int) {
	t.Helper()

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID    any `json:"id"`
			Error *struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != wantID || frame.Error == nil {
			continue
		}
		if frame.Error.Code != want {
			t.Fatalf("scenario %q: error.code = %d, want %d: %q", scenario, frame.Error.Code, want, line)
		}
		return
	}
	t.Fatalf("scenario %q: no error response (id=%v) found in captured stdout — a missing response must never read as a pass", scenario, wantID)
}

// metaSubscriptionIDKey is a hand-authored, package-local literal (VRFY-01)
// — never imported from the SDK under test — carrying the same wire value
// as go-sdk@v1.7.0's own exported MetaKeySubscriptionID constant
// [VERIFIED: github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:2377-2379].
// The correlation field a subscriptions/listen stream's acknowledgment and
// every notification it receives both carry, under `_meta`, so a client can
// demultiplex frames belonging to different concurrent listens.
const metaSubscriptionIDKey = "io.modelcontextprotocol/subscriptionId"

// modernListenCatalogChangeSubscriptionID is the listen request's own id in
// the modern-listen-catalog-change scenario
// (scenarios.go: subscriptionsListenRequest(1, ...)) — the value go-sdk
// stamps into that stream's acknowledgment and notification frames' `_meta`
// under metaSubscriptionIDKey. Hardcoded here rather than looked up from
// Scenarios() because exactly one scenario in this package opens a listen
// stream, and its id is a fixed part of its own definition, not a value
// that varies by call site the way assertErrorCode's wantID does.
const modernListenCatalogChangeSubscriptionID = 1

// assertSubscriptionAckEcho decodes the acknowledgment frame's
// params.notifications object into a plain map[string]bool and requires it
// to equal, by EXACT set equality (length compared first, then every key),
// the single-entry map naming notificationsOptInFieldName — never a
// "present" or "non-empty" check (05-CONTEXT.md D-02). The server answers a
// MISSPELLED opt-in with a subscriptionId and an EMPTY subscription set —
// `"notifications":{}` — and no error anywhere, so a client that typos its
// opt-in gets a successful-looking subscription that will never deliver
// anything. This echo is the ONLY discriminator between that dead
// subscription and a live one; Task 3's mutation 6 (MUTATION-PROOF.md)
// reproduces this exact failure shape, deliberately, to prove this
// assertion actually catches it. Also requires the frame's own `_meta`
// subscription id to equal the listen request's own id — the correlation
// field exists to demultiplex concurrent listens, and codegraph opens
// exactly one, so equality to that one id is the whole claim. Fails,
// quoting the observed line, when either check does not hold; fails
// separately, naming the scenario, when no acknowledgment frame is present
// at all — a missing frame must never read as a pass.
func assertSubscriptionAckEcho(t *testing.T, scenario string, stdout []byte) {
	t.Helper()

	want := map[string]bool{notificationsOptInFieldName: true}

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			Method string `json:"method"`
			Params struct {
				Meta          map[string]any  `json:"_meta"`
				Notifications map[string]bool `json:"notifications"`
			} `json:"params"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		if frame.Method != notificationSubscriptionsAcknowledgedMethod {
			continue
		}

		if len(frame.Params.Notifications) != len(want) {
			t.Fatalf("scenario %q: acknowledgment params.notifications = %v, want exactly %v: %q", scenario, frame.Params.Notifications, want, line)
		}
		for k, v := range want {
			if frame.Params.Notifications[k] != v {
				t.Fatalf("scenario %q: acknowledgment params.notifications = %v, want exactly %v: %q", scenario, frame.Params.Notifications, want, line)
			}
		}

		rawSubID, ok := frame.Params.Meta[metaSubscriptionIDKey]
		if !ok {
			t.Fatalf("scenario %q: acknowledgment params._meta missing %q: %q", scenario, metaSubscriptionIDKey, line)
		}
		subID, ok := idAsFloat64(rawSubID)
		if !ok || subID != modernListenCatalogChangeSubscriptionID {
			t.Fatalf("scenario %q: acknowledgment params._meta[%q] = %v, want %v: %q", scenario, metaSubscriptionIDKey, rawSubID, modernListenCatalogChangeSubscriptionID, line)
		}
		return
	}
	t.Fatalf("scenario %q: no acknowledgment (%s) frame found in captured stdout — a missing frame must never read as a pass", scenario, notificationSubscriptionsAcknowledgedMethod)
}

// assertToolsListChangedNotification requires EXACTLY ONE
// notifications/tools/list_changed frame in stdout, correlated to the
// listen request's own id via `_meta`, and content-free: its params key set
// is exactly {"_meta"} — decoded into a map[string]json.RawMessage and
// checked by length-then-key, never assumed — so T-05-02's property ("a
// subscriber learns that the catalog moved, never what is in it") is
// asserted rather than assumed. Fails naming the observed count on zero
// frames (the opted-in stream received nothing) and on two-or-more
// (changeAndNotify's debounce stopped coalescing).
func assertToolsListChangedNotification(t *testing.T, scenario string, stdout []byte) {
	t.Helper()

	var matches [][]byte
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		if frame.Method == notificationToolsListChangedMethod {
			matches = append(matches, append([]byte(nil), line...))
		}
	}

	if len(matches) != 1 {
		t.Fatalf("scenario %q: found %d %s frames, want exactly 1: %q", scenario, len(matches), notificationToolsListChangedMethod, matches)
	}
	line := matches[0]

	var frame struct {
		Params map[string]json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &frame); err != nil {
		t.Fatalf("scenario %q: decode %s params: %v; raw line: %q", scenario, notificationToolsListChangedMethod, err, line)
	}
	if _, ok := frame.Params["_meta"]; !ok || len(frame.Params) != 1 {
		t.Fatalf("scenario %q: %s params has %d keys, want exactly 1 (%q): %q", scenario, notificationToolsListChangedMethod, len(frame.Params), "_meta", line)
	}

	var metaFrame struct {
		Params struct {
			Meta map[string]any `json:"_meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(line, &metaFrame); err != nil {
		t.Fatalf("scenario %q: decode %s params._meta: %v; raw line: %q", scenario, notificationToolsListChangedMethod, err, line)
	}
	rawSubID, ok := metaFrame.Params.Meta[metaSubscriptionIDKey]
	if !ok {
		t.Fatalf("scenario %q: %s params._meta missing %q: %q", scenario, notificationToolsListChangedMethod, metaSubscriptionIDKey, line)
	}
	subID, ok := idAsFloat64(rawSubID)
	if !ok || subID != modernListenCatalogChangeSubscriptionID {
		t.Fatalf("scenario %q: %s params._meta[%q] = %v, want %v: %q", scenario, notificationToolsListChangedMethod, metaSubscriptionIDKey, rawSubID, modernListenCatalogChangeSubscriptionID, line)
	}
}

// assertToolsListChangedCapability decodes only the response bearing
// wantID's result.capabilities.tools.listChanged field and requires it to
// be true — SPEC-09 criterion 1, asserted against a FRESH capture (never
// the golden file) so a wholesale transcript regeneration cannot launder a
// regression past the byte comparison. wantID is explicit, matching
// assertErrorCode's pattern, because it varies by scenario shape: this
// anchor is registered against both handshake-explore's Legacy `initialize`
// response and modern-discover-explore's Modern `server/discover`
// response, two structurally different result shapes that both carry
// capabilities.tools.listChanged at the same JSON path. Fails when the
// field is false or absent (a plain bool zero value cannot distinguish the
// two, and the acceptance criterion treats them identically), and fails
// separately when no response bearing wantID is found at all.
func assertToolsListChangedCapability(t *testing.T, scenario string, stdout []byte, wantID float64) {
	t.Helper()

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any `json:"id"`
			Result *struct {
				Capabilities struct {
					Tools struct {
						ListChanged bool `json:"listChanged"`
					} `json:"tools"`
				} `json:"capabilities"`
			} `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != wantID || frame.Result == nil {
			continue
		}
		if !frame.Result.Capabilities.Tools.ListChanged {
			t.Fatalf("scenario %q: result.capabilities.tools.listChanged = false or absent, want true: %q", scenario, line)
		}
		return
	}
	t.Fatalf("scenario %q: no response (id=%v) found in captured stdout — a missing response must never read as a pass", scenario, wantID)
}
