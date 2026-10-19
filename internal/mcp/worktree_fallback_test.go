package mcp

import (
	"path/filepath"
	"strings"
	"testing"
)

// mcpSiblingWorktreeFixture builds an indexed main checkout and a linked
// worktree in a different temp directory, outside it. Returns both
// absolute paths.
func mcpSiblingWorktreeFixture(t *testing.T) (worktree, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitM(t, main, "init")
	runGitM(t, main, "add", "-A")
	runGitM(t, main, "commit", "-m", "init")

	wt := filepath.Join(t.TempDir(), "sibling")
	runGitM(t, main, "worktree", "add", "-b", "sibling", wt)

	indexFixture(t, main)
	return wt, main
}

// TestSiblingWorktreeToolCalls: a server started in a sibling worktree
// confines tool calls to the worktree, answers them from the main
// checkout's index with the worktree notice, and rejects a path in the
// main checkout.
func TestSiblingWorktreeToolCalls(t *testing.T) {
	wt, main := mcpSiblingWorktreeFixture(t)
	start := filepath.Join(wt, "pkga")
	if got := deriveServeRepoPath(t, start); got != wt {
		t.Fatalf("confinement root for %s = %q, want the worktree %q", start, got, wt)
	}

	s := BuildServer(true, map[string]bool{"status": true}, wt, start)

	t.Run("codegraph_explore without path", func(t *testing.T) {
		result := callTool(t, s, "codegraph_explore", map[string]any{"query": "main"})
		if result.IsError {
			t.Fatalf("codegraph_explore returned an error result: %q", resultText(t, result))
		}
		if text := resultText(t, result); !strings.HasPrefix(text, worktreeNoticeText) {
			t.Fatalf("codegraph_explore output does not start with the worktree notice %q, got: %q", worktreeNoticeText, text)
		}
	})

	t.Run("codegraph_status", func(t *testing.T) {
		result := callTool(t, s, "codegraph_status", map[string]any{})
		if result.IsError {
			t.Fatalf("codegraph_status returned an error result: %q", resultText(t, result))
		}
		if text := resultText(t, result); !strings.Contains(text, worktreeBlockquotePrefix) {
			t.Fatalf("codegraph_status output missing the worktree warning %q, got: %q", worktreeBlockquotePrefix, text)
		}
	})

	t.Run("a path in the main checkout", func(t *testing.T) {
		result := callTool(t, s, "codegraph_explore", map[string]any{"query": "main", "path": main})
		if !result.IsError {
			t.Fatalf("codegraph_explore with path %s: expected an error result, got: %q", main, resultText(t, result))
		}
		if text := resultText(t, result); !strings.Contains(text, "outside the server's configured repo root") {
			t.Fatalf("codegraph_explore with path %s: error does not name the confinement root, got: %q", main, text)
		}
	})
}
