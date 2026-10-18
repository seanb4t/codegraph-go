package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitRefusesHomeDirectory: `codegraph init <home>` exits with
// ErrHomeDirectoryTarget, names the directory, and creates no .codegraph
// there.
func TestInitRefusesHomeDirectory(t *testing.T) {
	home := fakeHome(t)

	_, _, err := execCmd("init", home)
	if !errors.Is(err, ErrHomeDirectoryTarget) {
		t.Fatalf("init %s: got err %v, want ErrHomeDirectoryTarget", home, err)
	}
	if !strings.Contains(err.Error(), home) {
		t.Fatalf("init %s: error %q does not name the home directory", home, err)
	}
	if _, statErr := os.Stat(filepath.Join(home, codegraphDirName)); !os.IsNotExist(statErr) {
		t.Fatalf("init %s: %s/.codegraph exists after the refusal (stat err %v)", home, home, statErr)
	}
}

// TestUninitRefusesHomeDirectory: `codegraph uninit --force <home>` exits
// with ErrHomeDirectoryTarget and leaves an existing <home>/.codegraph,
// and the file inside it, untouched.
func TestUninitRefusesHomeDirectory(t *testing.T) {
	home := fakeHome(t)
	keep := filepath.Join(home, codegraphDirName, "telemetry.json")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(keep, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, _, err := execCmd("uninit", "--force", home)
	if !errors.Is(err, ErrHomeDirectoryTarget) {
		t.Fatalf("uninit --force %s: got err %v, want ErrHomeDirectoryTarget", home, err)
	}
	got, readErr := os.ReadFile(keep)
	if readErr != nil {
		t.Fatalf("uninit --force %s removed %s: %v", home, keep, readErr)
	}
	if string(got) != "{}\n" {
		t.Fatalf("uninit --force %s changed %s to %q", home, keep, got)
	}
}

// TestInitUnderHomeDirectoryProceeds: a project below the home directory
// is not the home directory, so `codegraph init <home>/x/repo` runs and
// creates the index there.
func TestInitUnderHomeDirectoryProceeds(t *testing.T) {
	home := fakeHome(t)
	repo := filepath.Join(home, "x", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	if _, _, err := execCmd("init", "--quiet", repo); err != nil {
		t.Fatalf("init %s: unexpected error: %v", repo, err)
	}
	if info, statErr := os.Stat(filepath.Join(repo, codegraphDirName, storeDirName)); statErr != nil || !info.IsDir() {
		t.Fatalf("init %s: no store directory afterwards (stat err %v)", repo, statErr)
	}
}
