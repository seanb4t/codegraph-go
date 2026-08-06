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

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

// newTestSession builds an in-memory client/server session pair for s
// (Phase 2, RESEARCH Testing Architecture): mcp.NewInMemoryTransports()
// returns two *InMemoryTransport halves; s.Run is started on one in a
// goroutine, and a client connects to the other via Connect, which
// performs the MCP initialize handshake itself — go-sdk's Client has no
// separate, repeatable Initialize call the way mark3labs' client did
// (initClient no longer exists; this is the PROVEN-ABSENT finding
// 02-RESEARCH.md Q1 asked to be confirmed rather than assumed: no
// ServerOptions field or client method lets a caller inject a
// ProtocolVersion — the repo-owned ProtocolVersion constant keeps its
// role as the asserted pin). Every other test file in this package uses
// this one helper rather than repeating the construction.
//
// The returned cleanup func closes the session; callers are responsible
// for calling it (directly or via defer).
func newTestSession(t *testing.T, s *mcp.Server) (*mcp.ClientSession, func()) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-mcp-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}

	return session, func() { _ = session.Close() }
}

// listToolNames connects to s via newTestSession and returns the sorted
// set of registered tool names.
func listToolNames(t *testing.T, s *mcp.Server) []string {
	t.Helper()

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
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

	result := callTool(t, s, "codegraph_explore", map[string]any{
		"query": "main",
		"path":  dir,
	})
	if result.IsError {
		t.Fatalf("codegraph_explore returned an error result: %+v", result)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "**Exploration:") {
		t.Fatalf("codegraph_explore output missing the Engine.Explore markdown header, got: %q", text)
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

	result := callTool(t, s, "codegraph_status", map[string]any{"path": outside})
	if !result.IsError {
		t.Fatal("codegraph_status with a path outside the server's repo root: expected an error result, got success")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "outside") {
		t.Fatalf("codegraph_status error message = %q, want it to explain the path is outside the repo root", text)
	}
}

// TestConfinementAnchoredOnRepoRootNotStartPath is WR-02's required test
// (02-REVIEW-2.md): CR-01 split openEngine's single confinement parameter
// into two adjacent, same-typed strings (defaultPath/startPath and
// repoPath). TestOpenEnginePathConfinedToRepoRoot above builds its server
// with startPath == repoPath (BuildServer(..., dir, dir)) — a degenerate
// configuration in which the two anchors are indistinguishable, so that
// test passes identically whichever one confineToRepoRoot actually
// anchors on. This test runs the SAME confinement assertion in the
// configuration production actually uses (startPath != repoPath, the
// linked-worktree shape serve.go produces via serveServerPaths), with a
// path chosen specifically to distinguish the two anchors: INSIDE repoPath
// (main) but OUTSIDE startPath (wt).
func TestConfinementAnchoredOnRepoRootNotStartPath(t *testing.T) {
	wt, main := mcpWorktreeMismatchFixture(t) // startPath=wt, repoPath=main

	s := BuildServer(true, map[string]bool{"status": true}, deriveServeRepoPath(t, wt), wt)

	// A sibling of wt is INSIDE repoPath (main) but OUTSIDE startPath (wt).
	// Anchored on repoPath (correct) this resolves; anchored on startPath
	// it would be rejected — so this call distinguishes the two anchors.
	sibling := filepath.Join(main, "pkga")
	result := callTool(t, s, "codegraph_status", map[string]any{"path": sibling})
	if result.IsError {
		t.Fatalf("a path inside repoPath (%q) was rejected — confinement is anchored on startPath, not repoPath: %+v", sibling, result)
	}

	// A genuinely outside, separately-indexed path must still be rejected
	// — confining on repoPath must not be confused with "accept anything".
	outside := copyFixture(t)
	indexFixture(t, outside)
	result = callTool(t, s, "codegraph_status", map[string]any{"path": outside})
	if !result.IsError {
		t.Fatal("a path outside repoPath was accepted — confinement is not enforced")
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
