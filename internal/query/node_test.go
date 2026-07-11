package query

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveSourcePathAllowsRegularFileInRepo is the control case for
// TestResolveSourcePathRejectsSymlinkEscape: a plain, non-symlinked
// in-repo file must still be readable via Node's file mode.
func TestResolveSourcePathAllowsRegularFileInRepo(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	out, err := engine.Node("", "main.go")
	if err != nil {
		t.Fatalf("Node: unexpected error reading a regular in-repo file: %v", err)
	}
	if out == "" {
		t.Fatal("Node: expected non-empty rendered source for main.go")
	}
}

// TestResolveSourcePathRejectsSymlinkEscape pins WR-03: a symlink inside
// the repo root that points at a file outside it must be rejected, not
// silently followed. The string-level Clean/Rel check alone cannot catch
// this — the symlink's own path text is entirely inside the repo — so
// this specifically exercises the filepath.EvalSymlinks re-verification
// step.
func TestResolveSourcePathRejectsSymlinkEscape(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	// Create a directory OUTSIDE the repo root with a secret file, then a
	// symlink INSIDE the repo root pointing at it.
	outside := t.TempDir()
	secretPath := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("outside-repo-secret"), 0o644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	linkPath := filepath.Join(dir, "escape-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err = engine.Node("", "escape-link/secret.txt")
	if err == nil {
		t.Fatal("Node: expected an error for a path escaping the repo root via a symlink, got nil")
	}
}
