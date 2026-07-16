// Package mcp implements the codegraph stdio MCP server (D-08): a
// startup-time conditional tool registration surface built on
// github.com/mark3labs/mcp-go. It imports only internal/query (the
// read-only engine + formatters) — never internal/graphstore's Pebble
// implementation directly — so the internal/graphstore/archtest boundary
// holds (D-08b). Every tool handler in tools.go delegates to the same
// internal/query.Engine methods and formatters the CLI uses, taking a
// fresh query.OpenAt snapshot per call (D-02/D-08b, RESEARCH Pitfall 2)
// — one engine, two front-ends, so MCP output shapes stay byte-identical
// to the CLI's without a second rendering path.
package mcp

import (
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// version is this server's reported MCP implementation version. There is
// no project release-version concept yet (Phase 6 territory); a literal
// placeholder is fine here since MCP clients don't gate behavior on it.
const version = "0.1.0"

// companionNames is the fixed vocabulary of the 7 tools CODEGRAPH_MCP_TOOLS
// may allowlist (D-08a, MCP-02) — codegraph_explore is not in this list
// because it is always visible when hasIndex is true (MCP-01) and is
// never gated by the allowlist.
var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}

// ParseAllowlist splits the CODEGRAPH_MCP_TOOLS env value on commas,
// trims whitespace around each entry, and classifies each non-empty
// name against companionNames (D-08a/MCP-02). Recognized names are
// returned in allowed (allowed[name] == true); unrecognized names are
// returned in unknown, in the order they were seen, for the caller to
// warn about via WarnUnknownToolsTo — ParseAllowlist itself never writes
// output or aborts, so an unknown name can never fail startup (MCP-02:
// "unknown names ignored with a stderr warning").
func ParseAllowlist(env string) (allowed map[string]bool, unknown []string) {
	allowed = make(map[string]bool)
	known := make(map[string]bool, len(companionNames))
	for _, n := range companionNames {
		known[n] = true
	}

	for _, raw := range strings.Split(env, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if known[name] {
			allowed[name] = true
			continue
		}
		unknown = append(unknown, name)
	}
	return allowed, unknown
}

// WarnUnknownToolsTo writes one stderr-style warning line per unknown
// allowlist name to w. Diagnostics never go to stdout — stdout is
// reserved for the MCP JSON-RPC transport (T-03-07-Leak) — so callers
// must pass os.Stderr, never os.Stdout, in production.
func WarnUnknownToolsTo(w io.Writer, unknown []string) {
	for _, name := range unknown {
		fmt.Fprintf(w, "codegraph mcp: unknown tool name %q in CODEGRAPH_MCP_TOOLS, ignoring\n", name)
	}
}

// BuildServer constructs the stdio MCP server with startup-time
// conditional tool registration (D-08a, Pattern 3): hasIndex gates
// whether ANY tool is registered at all (MCP-03 — zero tools when no
// .codegraph/ resolves, though MCP init still completes successfully),
// allowlist gates which of the 7 companion tools register beyond the
// always-visible codegraph_explore (MCP-01/02), and repoPath is the
// default repo root every tool handler resolves against when the caller
// does not supply its own "path" argument.
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath string) *server.MCPServer {
	s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))
	if !hasIndex {
		return s
	}

	// One gitmeta.CachingDetector per SERVER, not per handler or per call
	// (D-13, corrected). openEngine builds a FRESH query.Engine on every
	// single tool call by design (its own doc comment says so), so an
	// Engine-scoped cache alone would give ZERO cross-call benefit on this
	// server's long-lived process — the exact surface the cache exists to
	// help. Detection costs up to four git subprocesses per verdict;
	// constructing exactly one detector here and closing it over every
	// handler bounds that cost to once per (startPath, indexRoot) pair for
	// this server's entire lifetime, however many tool calls follow.
	detector := gitmeta.NewCachingDetector()

	s.AddTool(exploreTool(), exploreHandler(repoPath, detector))
	for _, name := range companionNames {
		if allowlist[name] {
			s.AddTool(companionTool(name), companionHandler(name, repoPath, detector))
		}
	}
	return s
}
