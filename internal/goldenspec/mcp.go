package goldenspec

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	internalmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// CallExploreViaMCP drives codegraph_explore through a real, in-process MCP
// server (internalmcp.BuildServer) over in-memory transports. This is the
// exact call shape the go-explore-mcp.json goldens were captured through,
// so the byte-identity oracle must drive its MCP-Explore cases through
// this function rather than a re-derived approximation of it.
//
// Moved verbatim (body unchanged, including the empty tool allowlist,
// which is part of what the -mcp goldens captured) from
// testdata/golden/gocapture/main.go's unexported callExploreViaMCP
// (2026-08-22).
func CallExploreViaMCP(repoDir, query string) (string, error) {
	s := internalmcp.BuildServer(true, map[string]bool{}, repoDir, repoDir)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-gocapture", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("client Connect: %w", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_explore",
		Arguments: map[string]any{"query": query},
	})
	if err != nil {
		return "", fmt.Errorf("CallTool codegraph_explore(%q): %w", query, err)
	}
	if result.IsError {
		return "", fmt.Errorf("codegraph_explore(%q) returned an error result", query)
	}
	return MCPResultText(result)
}

// CallNodeViaMCP drives codegraph_node the same way CallExploreViaMCP
// drives codegraph_explore, with no file/line narrowing.
//
// Moved verbatim from testdata/golden/gocapture/main.go's unexported
// callNodeViaMCP (2026-08-22), re-expressed as a call to
// CallNodeViaMCPWithArgs so the (symbol, file, line) call shape exists in
// exactly one place.
func CallNodeViaMCP(repoDir, symbol string) (string, error) {
	return CallNodeViaMCPWithArgs(repoDir, symbol, "", nil)
}

// CallNodeViaMCPWithArgs is CallNodeViaMCP's fuller sibling (CR-02): it
// additionally accepts the "file" and "line" args codegraph_node's schema
// exposes, so a caller can drive the SAME codegraph_node MCP call the
// CLI's --line flag reaches.
//
// Moved from testdata/golden/behavioral_test.go's unexported
// callNodeViaMCPWithArgs (2026-08-22), converted from a *testing.T/
// t.Fatalf shape to an error-returning shape — gocapture is package main
// and has no *testing.T to call — with the tool allowlist
// (map[string]bool{"node": true}) and call semantics otherwise unchanged
// from gocapture's own pre-move callNodeViaMCP, which this function also
// replaces.
func CallNodeViaMCPWithArgs(repoDir, symbol, file string, line *int) (string, error) {
	s := internalmcp.BuildServer(true, map[string]bool{"node": true}, repoDir, repoDir)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-gocapture", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("client Connect: %w", err)
	}
	defer session.Close()

	args := map[string]any{"symbol": symbol}
	if file != "" {
		args["file"] = file
	}
	if line != nil {
		args["line"] = float64(*line)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_node",
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("CallTool codegraph_node(%q): %w", symbol, err)
	}
	if result.IsError {
		return "", fmt.Errorf("codegraph_node(%q) returned an error result", symbol)
	}
	return MCPResultText(result)
}

// MCPResultText extracts the first text content block from a successful
// CallTool result.
//
// Moved verbatim from testdata/golden/gocapture/main.go's unexported
// mcpResultText (2026-08-22).
func MCPResultText(result *mcp.CallToolResult) (string, error) {
	if len(result.Content) == 0 {
		return "", fmt.Errorf("CallTool result has no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text, nil
}
