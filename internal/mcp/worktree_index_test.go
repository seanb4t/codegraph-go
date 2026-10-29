package mcp

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/wtindex"
)

// TestWorktreeIndexAppearsMidSession: a server started on a sibling
// worktree's borrowed roots answers from the main checkout's index with
// the notice, and from the worktree's own index, without the notice, as
// soon as one is placed. The session and the tool list do not change.
func TestWorktreeIndexAppearsMidSession(t *testing.T) {
	wt, main := mcpSiblingWorktreeFixture(t)
	if err := os.MkdirAll(filepath.Join(wt, "onlyhere"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wt, "onlyhere", "onlyhere.go"), []byte("package onlyhere\n\nfunc OnlyHere() {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	r, err := query.ResolveIndex(wt)
	if err != nil {
		t.Fatalf("ResolveIndex: %v", err)
	}
	if r.ConfineRoot != wt {
		t.Fatalf("confinement root = %q; want the worktree %q", r.ConfineRoot, wt)
	}

	s := BuildServer(true, map[string]bool{"status": true}, r.ConfineRoot, wt)
	session, cleanup := newTestSession(t, s)
	defer cleanup()
	explore := func() string {
		t.Helper()
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "codegraph_explore",
			Arguments: map[string]any{"query": "OnlyHere"},
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if result.IsError {
			t.Fatalf("codegraph_explore returned an error result: %q", resultText(t, result))
		}
		return resultText(t, result)
	}
	tools := func() []string {
		t.Helper()
		result, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		var names []string
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		slices.Sort(names)
		return names
	}

	toolsBefore := tools()
	before := explore()
	if !strings.HasPrefix(before, worktreeNoticeText) {
		t.Fatalf("before the worktree index: output does not start with the worktree notice, got: %q", before)
	}
	if strings.Contains(before, "onlyhere/onlyhere.go") {
		t.Fatalf("before the worktree index: the main checkout's index found OnlyHere: %q", before)
	}

	b, err := wtindex.Prepare(context.Background(), wt, main, wtindex.DefaultLimits)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !b.Ready {
		t.Fatalf("Prepare left the build pending (%s)", b.Reason)
	}

	after := explore()
	if strings.Contains(after, worktreeNoticeText) {
		t.Errorf("after the worktree index: output still carries the worktree notice: %q", after)
	}
	if !strings.Contains(after, "onlyhere/onlyhere.go") {
		t.Errorf("after the worktree index: OnlyHere not found: %q", after)
	}
	if toolsAfter := tools(); !slices.Equal(toolsBefore, toolsAfter) {
		t.Errorf("tools/list changed: %v -> %v", toolsBefore, toolsAfter)
	}
}
