package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestManCmd_CreatesMissingDirectory_AndWritesFullTree asserts `codegraph
// man <dir>` creates every missing parent of a target directory that does
// not exist yet — doc.GenManTree does not create its own destination (see
// internal/cli/man.go's doc comment) — and writes the full command-tree
// page set into it, including the root codegraph.1 page. This is the
// behavior the cask hook depends on: it targets Homebrew's man1 directory,
// which is absent on a prefix where nothing has yet installed a man page.
func TestManCmd_CreatesMissingDirectory_AndWritesFullTree(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "several", "missing", "parents", "man1")

	if _, _, err := execCmd("man", dir); err != nil {
		t.Fatalf("man %s: unexpected error: %v", dir, err)
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("ReadDir(%s) after man: %v", dir, readErr)
	}
	if len(entries) <= 1 {
		t.Fatalf("expected strictly more than one man page written, got %d", len(entries))
	}
	if _, statErr := os.Stat(filepath.Join(dir, "codegraph.1")); statErr != nil {
		t.Fatalf("expected codegraph.1 to exist in %s: %v", dir, statErr)
	}
}

// TestManCmd_UnwritablePath_ReturnsNonNilErrorNamingPath asserts `man`
// returns a non-nil error naming the target directory when a write inside
// it cannot succeed, rather than printing a friendly line and returning
// nil (internal/cli/man.go's RunE doc comment: a hidden command invoked by
// a Ruby postflight hook has no interactive user to address). Mirrors
// githooks_test.go's TestGithooksInstall_AllHooksUnwritable_ShowsSyncFallbackMessage
// convention: pre-seed a directory at the exact path GenManTree will try to
// os.Create, so the write deterministically fails with EISDIR — this
// avoids a chmod-based permission test, which no-ops when run as root.
func TestManCmd_UnwritablePath_ReturnsNonNilErrorNamingPath(t *testing.T) {
	dir := t.TempDir()
	// doc.GenManTree writes every child command's page before the command's
	// own page, so the root's page ("codegraph.1") is written last —
	// pre-seeding a directory at that name collides only after every other
	// page in the tree has already written successfully.
	if err := os.MkdirAll(filepath.Join(dir, "codegraph.1"), 0o755); err != nil {
		t.Fatalf("MkdirAll conflicting entry: %v", err)
	}

	_, _, err := execCmd("man", dir)
	if err == nil {
		t.Fatalf("man %s: expected a non-nil error, got nil", dir)
	}
	if !strings.Contains(err.Error(), dir) {
		t.Fatalf("expected error to name the offending path %s, got %q", dir, err.Error())
	}
}

// TestManCmd_WritesFullCommandTree_NotJustRootPage asserts the man command
// generates a page for the full command tree — one file per registered
// command/subcommand, not a single root page — the D-04 behavior this
// command exists to provide.
func TestManCmd_WritesFullCommandTree_NotJustRootPage(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := execCmd("man", dir); err != nil {
		t.Fatalf("man %s: unexpected error: %v", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	if len(entries) <= 1 {
		t.Fatalf("expected strictly more than one man page, got %d: %v", len(entries), entries)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "codegraph.1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected codegraph.1 among the written pages, got %v", entries)
	}
}

// TestManCmd_HiddenOnRootCommand asserts the man command's Hidden field is
// true by reading the command object located in newRootCmd().Commands(),
// not a substring of rendered help output — a negative-only assertion
// against help text would pass vacuously the moment the help template
// changes (repository rule 84d1gfpywd).
func TestManCmd_HiddenOnRootCommand(t *testing.T) {
	root := newRootCmd()
	var man *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "man" {
			man = c
			break
		}
	}
	if man == nil {
		t.Fatalf(`expected a "man" command registered on newRootCmd()`)
	}
	if !man.Hidden {
		t.Fatalf("expected the man command's Hidden field to be true, got false")
	}
}

// TestManCmd_ArgsExactlyOne asserts `man` rejects zero and two arguments
// and accepts exactly one.
func TestManCmd_ArgsExactlyOne(t *testing.T) {
	if _, _, err := execCmd("man"); err == nil {
		t.Fatalf("man (0 args): expected a non-nil error, got nil")
	}
	if _, _, err := execCmd("man", "a", "b"); err == nil {
		t.Fatalf("man a b (2 args): expected a non-nil error, got nil")
	}

	dir := t.TempDir()
	if _, _, err := execCmd("man", dir); err != nil {
		t.Fatalf("man <dir> (1 arg): expected nil error, got %v", err)
	}
}
