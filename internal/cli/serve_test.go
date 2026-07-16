package cli

import "testing"

// TestServeKeepsStartPathDistinctFromConfinementRoot is WR-01's required
// test (02-REVIEW-2.md): it exercises serveServerPaths, the EXACT function
// newServeCmd's RunE calls to derive BuildServer's repoPath argument — not
// a hand-built replica living only in a test file (markdown_test.go's
// deriveServeRepoPath, in internal/mcp, is such a replica; it proves the
// mismatch fixture is wired correctly, but nothing there observes whether
// serve.go itself still performs this derivation).
//
// Reproduces the reviewer's mutation directly: if serveServerPaths ever
// collapsed to `return start, hasIndex, nil` unconditionally (the literal
// CR-01 regression — BuildServer(..., repoPath, repoPath)), repoPath would
// equal wt here and this test would fail.
func TestServeKeepsStartPathDistinctFromConfinementRoot(t *testing.T) {
	wt, main := statusWorktreeMismatchFixture(t)

	repoPath, hasIndex, err := serveServerPaths(wt)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", wt, err)
	}
	if !hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = false, want true (main checkout was indexed)", wt)
	}
	if repoPath == wt {
		t.Fatal("serveServerPaths returned the START path as repoPath — CR-01 regression: repoPath must be the RESOLVED index root, distinct from the caller's start path in a worktree, or BuildServer's worktree-mismatch detection silently short-circuits to nil on every production call")
	}
	if repoPath != main {
		t.Fatalf("serveServerPaths(%s) repoPath = %q, want the main checkout %q", wt, repoPath, main)
	}
}

// TestServeServerPathsNoIndex pins MCP-03: when no .codegraph/ resolves
// above start, serveServerPaths reports hasIndex=false (not an error) and
// repoPath falls back to start itself — serve still starts with zero tools
// rather than refusing the connection.
func TestServeServerPathsNoIndex(t *testing.T) {
	dir := t.TempDir()

	repoPath, hasIndex, err := serveServerPaths(dir)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", dir, err)
	}
	if hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = true, want false (no .codegraph/ present)", dir)
	}
	if repoPath != dir {
		t.Fatalf("serveServerPaths(%s) repoPath = %q, want %q (no index found: repoPath falls back to start)", dir, repoPath, dir)
	}
}
