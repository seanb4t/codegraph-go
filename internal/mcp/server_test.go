package mcp

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// copyFixture copies internal/indexer/testdata/gofixture into a fresh
// t.TempDir(), mirroring internal/query/engine_test.go's helper of the
// same name (03-PATTERNS.md §"Test scaffolding") so this package's tests
// run against a normal directory with its own go.mod rather than
// mutating the checked-in testdata tree.
func copyFixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "gofixture"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

// indexFixture builds a real graph from the fixture rooted at dir into
// <dir>/.codegraph/store via indexer.Run, mirroring
// internal/query/engine_test.go's helper of the same name — a real
// Pebble-backed store, not a mock, so the handler-delegation test
// exercises the actual engine read path.
func indexFixture(t *testing.T, dir string) {
	t.Helper()

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

// initClient runs the MCP initialize handshake against c with a minimal
// valid InitializeRequest, failing the test on error.
func initClient(t *testing.T, ctx context.Context, c *mcpclient.Client) {
	t.Helper()

	req := mcp.InitializeRequest{}
	req.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	req.Params.ClientInfo = mcp.Implementation{Name: "codegraph-mcp-test", Version: "0.0.0"}

	if _, err := c.Initialize(ctx, req); err != nil {
		t.Fatalf("client Initialize: %v", err)
	}
}

// listToolNames builds an in-process mcp-go client against s (no live
// stdio transport, per RESEARCH's Testing Architecture — the
// client/server pair talk to each other in-process) and returns the
// sorted set of registered tool names.
func listToolNames(t *testing.T, s *server.MCPServer) []string {
	t.Helper()

	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	initClient(t, ctx, c)

	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	sort.Strings(names)
	return names
}

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

func TestAllowlist(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	allowed, unknown := ParseAllowlist("node,status,bogus")

	if !allowed["node"] || !allowed["status"] {
		t.Fatalf("ParseAllowlist allowed = %v, want node+status set", allowed)
	}
	if allowed["bogus"] {
		t.Fatalf("ParseAllowlist allowed contains unknown name %q", "bogus")
	}
	if len(unknown) != 1 || unknown[0] != "bogus" {
		t.Fatalf("ParseAllowlist unknown = %v, want [bogus]", unknown)
	}

	var stderr bytes.Buffer
	WarnUnknownToolsTo(&stderr, unknown)
	if !strings.Contains(stderr.String(), "bogus") {
		t.Fatalf("WarnUnknownToolsTo did not mention %q, got %q", "bogus", stderr.String())
	}

	s := BuildServer(true, allowed, dir, dir)
	got := listToolNames(t, s)
	want := []string{"codegraph_explore", "codegraph_node", "codegraph_status"}
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}
}

func TestNoIndexZeroTools(t *testing.T) {
	dir := t.TempDir()

	s := BuildServer(false, map[string]bool{"node": true, "status": true}, dir, dir)

	got := listToolNames(t, s)
	if len(got) != 0 {
		t.Fatalf("registered tools = %v, want none (MCP-03: no .codegraph/ means zero tools)", got)
	}
}

// TestExploreHandlerDelegatesToEngine proves D-08b: the codegraph_explore
// tool call resolves its path/query args, opens a fresh engine snapshot
// via query.OpenAt, and returns Engine.Explore's markdown — not a
// second, MCP-only rendering path.
func TestExploreHandlerDelegatesToEngine(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{}, dir, dir)

	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	initClient(t, ctx, c)

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "codegraph_explore",
			Arguments: map[string]any{
				"query": "main",
				"path":  dir,
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_explore: %v", err)
	}
	if result.IsError {
		t.Fatalf("codegraph_explore returned an error result: %+v", result)
	}
	if len(result.Content) == 0 {
		t.Fatal("codegraph_explore returned no content")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("codegraph_explore content[0] is not text: %+v", result.Content[0])
	}
	if !strings.Contains(text.Text, "**Exploration:") {
		t.Fatalf("codegraph_explore output missing the Engine.Explore markdown header, got: %q", text.Text)
	}
}

// TestOpenEnginePathConfinedToRepoRoot pins CR-02: a client-supplied
// "path" argument that resolves outside the server's configured repo
// root must be rejected with an MCP tool error, never opened as an
// engine — even when that outside path is itself a validly-indexed
// .codegraph/ project (proving this is a trust-boundary confinement
// check, not just an "index not found" failure).
func TestOpenEnginePathConfinedToRepoRoot(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	outside := copyFixture(t)
	indexFixture(t, outside)

	s := BuildServer(true, map[string]bool{"status": true}, dir, dir)

	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	initClient(t, ctx, c)

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "codegraph_status",
			Arguments: map[string]any{
				"path": outside,
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_status: %v", err)
	}
	if !result.IsError {
		t.Fatal("codegraph_status with a path outside the server's repo root: expected an error result, got success")
	}
	if len(result.Content) == 0 {
		t.Fatal("codegraph_status error result has no content")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("codegraph_status error content[0] is not text: %+v", result.Content[0])
	}
	if !strings.Contains(text.Text, "outside") {
		t.Fatalf("codegraph_status error message = %q, want it to explain the path is outside the repo root", text.Text)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
