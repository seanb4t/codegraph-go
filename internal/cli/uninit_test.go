package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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

// TestUninit_UnwritableHooksDir_SurfacesWarning is the WR-01 regression
// test: uninit's D-06 best-effort hook cleanup must surface
// RemoveResult.Errors as stderr warnings via printHookErrors, matching the
// standalone `githooks remove` command, rather than silently discarding a
// per-hook write/delete failure just because cleanup is non-fatal to
// uninit overall. Skipped under root (permission bits don't apply) and on
// Windows (no POSIX permission model), matching the convention in
// internal/githooks's own TestRemove_UnwritableHooksDir_AccumulatesErrors.
func TestUninit_UnwritableHooksDir_SurfacesWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

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

	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.Chmod(hooksDir, 0o500); err != nil {
		t.Fatalf("Chmod(%s, 0500): %v", hooksDir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(hooksDir, 0o755) })

	_, stderr, err := execCmd("uninit", "--force", dir)
	if err != nil {
		t.Fatalf("uninit --force: unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "warning:") {
		t.Fatalf("expected stderr to contain a warning for the failed hook cleanup, got %q", stderr)
	}
}

// TestUninit_MalformedHook_SurfacesWarning is the round-6 WR-01 regression
// test: uninit's D-06 best-effort hook cleanup must surface a warning for a
// hand-damaged hook file (begin marker present, end marker deleted) the
// same way it does for an unwritable hooks dir — not silently remove
// .codegraph/ and report success while leaving a malformed hook behind
// with no signal it exists.
func TestUninit_MalformedHook_SurfacesWarning(t *testing.T) {
	dir := copyFixture(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}

	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"",
		markerBeginBytes,
		"... (block body, end marker line deleted) ...",
		"echo after",
		"echo more-user-content",
	}, "\n") + "\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, stderr, err := execCmd("uninit", "--force", dir)
	if err != nil {
		t.Fatalf("uninit --force: unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "malformed codegraph marker block") {
		t.Fatalf("expected stderr to warn about the malformed marker block, got %q", stderr)
	}
	got, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(got) != malformed {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), malformed)
	}
}

// TestSanitizePathForDisplay shows that the wrapper delegates to
// present.SanitizeDisplay. The rune rules are tested in package present.
func TestSanitizePathForDisplay(t *testing.T) {
	in := "/a\x1b/b" + string(rune(0x202E)) + "c"
	if got, want := sanitizePathForDisplay(in), "/a<U+001B>/b<U+202E>c"; got != want {
		t.Errorf("sanitizePathForDisplay(%q) = %q, want %q", in, got, want)
	}
}

// TestUninitAndIndexSanitizeTheDirectoryName runs the uninit and index
// commands in a directory whose name holds a bidi override and checks every
// printed line and prompt that names it, in plain output.
func TestUninitAndIndexSanitizeTheDirectoryName(t *testing.T) {
	newRoot := func(t *testing.T, withIndex bool) string {
		t.Helper()
		root := filepath.Join(t.TempDir(), "gpj.\u202Eexe")
		dir := root
		if withIndex {
			dir = filepath.Join(root, ".codegraph")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", dir, err)
		}
		return root
	}
	requireEscaped := func(t *testing.T, label, out string) {
		t.Helper()
		if strings.Contains(out, "\u202E") {
			t.Errorf("%s holds a raw U+202E: %q", label, out)
		}
		if !strings.Contains(out, "gpj.<U+202E>exe") {
			t.Errorf("%s lacks the <U+202E> escape: %q", label, out)
		}
	}

	t.Run("uninit with no index", func(t *testing.T) {
		out, _, err := execCmd("uninit", newRoot(t, false))
		if err != nil {
			t.Fatalf("uninit: %v", err)
		}
		requireEscaped(t, "nothing-to-do line", out)
	})

	t.Run("uninit confirmation prompt", func(t *testing.T) {
		out, _, err := execCmdWithInput("n\n", "uninit", newRoot(t, true))
		if err != nil {
			t.Fatalf("uninit: %v", err)
		}
		requireEscaped(t, "prompt", out)
	})

	t.Run("uninit removed line", func(t *testing.T) {
		out, _, err := execCmd("uninit", "--force", newRoot(t, true))
		if err != nil {
			t.Fatalf("uninit --force: %v", err)
		}
		requireEscaped(t, "removed line", out)
	})

	t.Run("sync not initialized error", func(t *testing.T) {
		_, _, err := execCmd("sync", newRoot(t, false))
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("sync: %v; want errors.Is(err, ErrNotInitialized)", err)
		}
		requireEscaped(t, "error", err.Error())
	})

	t.Run("init already initialized error", func(t *testing.T) {
		_, _, err := execCmd("init", newRoot(t, true))
		if !errors.Is(err, ErrAlreadyInitialized) {
			t.Fatalf("init: %v; want errors.Is(err, ErrAlreadyInitialized)", err)
		}
		requireEscaped(t, "error", err.Error())
	})

	t.Run("index not initialized error", func(t *testing.T) {
		_, _, err := execCmd("index", newRoot(t, false))
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("index: %v; want errors.Is(err, ErrNotInitialized)", err)
		}
		requireEscaped(t, "error", err.Error())
	})

	t.Run("index confirmation prompt", func(t *testing.T) {
		out, _, err := execCmdWithInput("n\n", "index", newRoot(t, true))
		if err != nil {
			t.Fatalf("index: %v", err)
		}
		requireEscaped(t, "prompt", out)
	})
}
