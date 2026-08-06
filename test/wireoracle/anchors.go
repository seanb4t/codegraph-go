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
