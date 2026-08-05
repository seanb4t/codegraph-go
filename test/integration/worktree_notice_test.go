package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	codegraphmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// newServeClient spawns the real binPath binary as `serve --mcp` with the
// SPAWNED PROCESS's own working directory set to cwd — D-19/D-20's anchor
// requirement: the real cwd->resolveStartPath->serveServerPaths->
// BuildServer wiring seam, not a `-p` flag (which drives the same `start`
// value but never exercises os.Getwd()'s role in resolveStartPath) and
// never an in-process client. transport.WithCommandFunc is mcp-go's
// documented seam for exactly this ("working directories... environment
// control"), used here instead of the plan's discretionary `-p` fallback
// since it lets the anchor drive the REAL process cwd directly.
//
// initializes the MCP session and returns the connected client; the
// caller does not need to call Close (registered via t.Cleanup).
func newServeClient(t *testing.T, ctx context.Context, cwd string) *mcpclient.Client {
	t.Helper()

	c, err := mcpclient.NewStdioMCPClientWithOptions(binPath, nil, []string{"serve", "--mcp"},
		transport.WithCommandFunc(func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(ctx, command, args...)
			cmd.Dir = cwd
			cmd.Env = append(os.Environ(), env...)
			return cmd, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewStdioMCPClientWithOptions(serve --mcp, cwd=%s): %v", cwd, err)
	}
	t.Cleanup(func() { _ = c.Close() })

	req := mcp.InitializeRequest{}
	// codegraphmcp.ProtocolVersion is the repo-owned literal — a
	// dependency bump must not move this value silently (VRFY-02).
	req.Params.ProtocolVersion = codegraphmcp.ProtocolVersion
	req.Params.ClientInfo = mcp.Implementation{Name: "codegraph-integration-test", Version: "0.0.0"}
	if _, err := c.Initialize(ctx, req); err != nil {
		t.Fatalf("Initialize (cwd=%s): %v", cwd, err)
	}
	return c
}

// resultText extracts the first text content block from result, failing
// the test if result has no content, the first block is not text, or the
// tool call itself reported an error — mirroring
// internal/mcp/markdown_test.go's helper of the same name.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if result.IsError {
		t.Fatalf("CallTool result reported an error: %+v", result)
	}
	if len(result.Content) == 0 {
		t.Fatal("CallTool result has no content")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text
}

// exploreAlpha drives codegraph_explore for "Alpha" (pkga.Alpha, the same
// symbol internal/cli/notice_test.go and internal/mcp/markdown_test.go
// already rely on) against c, returning the payload text. codegraph_explore
// needs no CODEGRAPH_MCP_TOOLS allowlist entry — it is always visible
// (D-21 note; internal/mcp/tools.go/server.go).
func exploreAlpha(t *testing.T, ctx context.Context, c *mcpclient.Client) string {
	t.Helper()

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "codegraph_explore",
			Arguments: map[string]any{"query": "Alpha"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_explore: %v", err)
	}
	return resultText(t, result)
}

// TestWorktreeNoticeReachesServeMCPExplore is TEST-04's mandatory D-20
// anchor: a real `codegraph serve --mcp` subprocess, spawned with its
// process cwd inside a real git worktree linked at
// .claude/worktrees/probe, must return a codegraph_explore payload
// carrying the bare U+26A0 worktree notice — while the SAME binary, spawned
// with cwd in the main checkout, must not. This is the exact
// cwd/argv->resolveStartPath->serveServerPaths->mcp.BuildServer wiring
// in-process BuildServer->CallTool tests (internal/mcp/markdown_test.go)
// structurally bypass: those tests construct a *server.MCPServer directly
// and never touch serve.go's RunE at all, so they cannot detect a
// regression in how serve.go derives BuildServer's arguments from a real
// process's cwd.
//
// Mutation-proof (CR-01 root cause, documented per the plan's acceptance
// criteria — verified manually, not committed): temporarily editing
// internal/cli/serve.go's `s := mcp.BuildServer(hasIndex, allowlist,
// repoPath, start)` call to pass repoPath for BOTH the third and fourth
// arguments (mcp.BuildServer(hasIndex, allowlist, repoPath, repoPath)) —
// the literal CR-01 regression, collapsing startPath == repoRoot so
// eng.WorktreeMismatch's cwd-vs-index-root comparison always sees the SAME
// path and silently short-circuits to no-mismatch on every real call —
// must turn this test red, while internal/mcp/markdown_test.go's in-process
// suite (which builds its own BuildServer calls with explicit, correct
// (repoPath, startPath) fixture values regardless of what serve.go itself
// does) stays green. That gap is precisely why TEST-04 exists.
func TestWorktreeNoticeReachesServeMCPExplore(t *testing.T) {
	wt, main := buildWorktreeFixture(t)
	glyph := noticeGlyph(t)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Anchor: cwd inside the borrowed worktree — the explore payload must
	// carry the bare notice.
	wtClient := newServeClient(t, ctx, wt)
	wtText := exploreAlpha(t, ctx, wtClient)
	if !containsBareNoticeGlyph(wtText, glyph) {
		t.Fatalf("worktree-cwd codegraph_explore: expected the bare worktree notice glyph %q, got: %q", glyph, wtText)
	}

	// Control: cwd in the main checkout (not borrowed) — no notice.
	mainClient := newServeClient(t, ctx, main)
	mainText := exploreAlpha(t, ctx, mainClient)
	if containsBareNoticeGlyph(mainText, glyph) {
		t.Fatalf("main-checkout codegraph_explore: expected NO worktree notice, got: %q", mainText)
	}
}
