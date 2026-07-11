// Package mcp implements the codegraph stdio MCP server (D-08): a
// startup-time conditional tool registration surface built on
// github.com/mark3labs/mcp-go. It imports only internal/query (the
// read-only engine + formatters) — never internal/graphstore's Pebble
// implementation directly — so the internal/graphstore/archtest boundary
// holds (D-08b).
package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// version is this server's reported MCP implementation version. There is
// no project release-version concept yet (Phase 6 territory); a literal
// placeholder is fine here since MCP clients don't gate behavior on it.
const version = "0.1.0"

// BuildServer constructs the stdio MCP server with startup-time
// conditional tool registration (D-08a, Pattern 3): hasIndex gates
// whether ANY tool is registered at all (MCP-03 — zero tools when no
// .codegraph/ resolves), allowlist gates which of the 7 companion tools
// register beyond the always-visible codegraph_explore (MCP-01/02), and
// repoPath is the default repo root every tool handler resolves against
// when the caller does not supply its own "path" argument.
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath string) *server.MCPServer {
	s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))
	// TODO(03-07 Task 2): conditional AddTool registration.
	_ = allowlist
	_ = repoPath
	_ = query.OpenAt
	return s
}
