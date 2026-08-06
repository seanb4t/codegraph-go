package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newServeClient spawns the real binPath binary as `serve --mcp` with the
// SPAWNED PROCESS's own working directory set to cwd — D-19/D-20's anchor
// requirement: the real cwd->resolveStartPath->serveServerPaths->
// BuildServer wiring seam, not a `-p` flag (which drives the same `start`
// value but never exercises os.Getwd()'s role in resolveStartPath) and
// never an in-process client. exec.CommandContext ties the subprocess's
// lifetime to ctx — the go-sdk equivalent of mark3labs' WithCommandFunc
// seam this test used before the SDK swap (02-04) — and
// mcp.CommandTransport{Command: cmd} plus Client.Connect performs the
// initialize handshake itself, replacing mark3labs' separate
// NewStdioMCPClientWithOptions + Initialize calls.
//
// initializes the MCP session and returns the connected session; the
// caller does not need to call Close (registered via t.Cleanup).
func newServeClient(t *testing.T, ctx context.Context, cwd string) *mcp.ClientSession {
	t.Helper()

	cmd := exec.CommandContext(ctx, binPath, "serve", "--mcp")
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-integration-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("CommandTransport Connect(serve --mcp, cwd=%s): %v", cwd, err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
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
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text
}

// exploreAlpha drives codegraph_explore for "Alpha" (pkga.Alpha, the same
// symbol internal/cli/notice_test.go and internal/mcp/markdown_test.go
// already rely on) against session, returning the payload text.
// codegraph_explore needs no CODEGRAPH_MCP_TOOLS allowlist entry — it is
// always visible (D-21 note; internal/mcp/tools.go/server.go).
func exploreAlpha(t *testing.T, ctx context.Context, session *mcp.ClientSession) string {
	t.Helper()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_explore",
		Arguments: map[string]any{"query": "Alpha"},
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
// structurally bypass: those tests construct a *mcp.Server directly and
// never touch serve.go's RunE at all, so they cannot detect a regression
// in how serve.go derives BuildServer's arguments from a real process's
// cwd.
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
