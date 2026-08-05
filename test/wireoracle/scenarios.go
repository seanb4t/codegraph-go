package wireoracle

import "path/filepath"

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

// Scenarios returns the oracle's full scripted scenario list. Phase 1
// scripts exactly one scenario, handshake-explore, proving the oracle
// architecture end-to-end; plans 03/04/05 expand this list — this same
// file, same function, no phase-conditional branch (must_haves).
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
