package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUninit_RemovesGitHooks drives `codegraph init`, `githooks install`,
// then `uninit --force` through the real cobra command tree against a
// real-git fixture, and asserts the D-06 best-effort cleanup strips
// codegraph's marker blocks from the hook files and reports a "Removed
// git ... sync hook" line. Reverting uninit.go's githooks.Remove wiring
// turns this test red (D-13).
func TestUninit_RemovesGitHooks(t *testing.T) {
	dir := copyFixture(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	if _, _, err := execCmd("githooks", "install", dir); err != nil {
		t.Fatalf("githooks install: unexpected error: %v", err)
	}

	out, _, err := execCmd("uninit", "--force", dir)
	if err != nil {
		t.Fatalf("uninit --force: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Removed git") {
		t.Fatalf("expected stdout to report hook removal, got %q", out)
	}

	for _, hook := range []string{"post-commit", "post-merge", "post-checkout"} {
		hookPath := filepath.Join(dir, ".git", "hooks", hook)
		content, readErr := os.ReadFile(hookPath)
		if readErr == nil && strings.Contains(string(content), markerBeginBytes) {
			t.Fatalf("expected %s to no longer contain the begin marker after uninit --force, got %q", hook, string(content))
		}
		// os.IsNotExist is also an acceptable outcome — the effectively-empty
		// gate deletes the file entirely when only our block was present.
	}
}

// TestUninit_NoHooksInstalled_NoRemovalLine asserts uninit's D-06 cleanup
// is a silent no-op — no "Removed git" line — when no hooks were ever
// installed, never blocking or erroring the primary removal.
func TestUninit_NoHooksInstalled_NoRemovalLine(t *testing.T) {
	dir := copyFixture(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}

	out, _, err := execCmd("uninit", "--force", dir)
	if err != nil {
		t.Fatalf("uninit --force: unexpected error: %v", err)
	}
	if strings.Contains(out, "Removed git") {
		t.Fatalf("expected no hook-removal line when no hooks were installed, got %q", out)
	}
	if !strings.Contains(out, "removed "+filepath.Join(dir, codegraphDirName)) {
		t.Fatalf("expected the standard .codegraph removal line, got %q", out)
	}
}
