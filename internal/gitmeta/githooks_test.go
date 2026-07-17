package gitmeta

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitRepo_TrueForInitializedRepo(t *testing.T) {
	dir := initRepo(t, t.TempDir())

	if !IsGitRepo(context.Background(), dir) {
		t.Fatalf("IsGitRepo(%s) = false, want true", dir)
	}
}

func TestIsGitRepo_FalseForNonGitDir(t *testing.T) {
	dir := t.TempDir()

	if IsGitRepo(context.Background(), dir) {
		t.Fatalf("IsGitRepo(%s) = true, want false", dir)
	}
}

func TestHooksDir_PlainRepoResolvesUnderProjectRoot(t *testing.T) {
	dir := initRepo(t, t.TempDir())

	got := HooksDir(context.Background(), dir)
	if got == "" {
		t.Fatalf("HooksDir(%s) = \"\", want non-empty", dir)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("HooksDir(%s) = %q, want absolute path", dir, got)
	}
	want := filepath.Join(dir, ".git", "hooks")
	if got != want {
		t.Fatalf("HooksDir(%s) = %q, want %q", dir, got, want)
	}
}

func TestHooksDir_HonorsCoreHooksPath(t *testing.T) {
	dir := initRepo(t, t.TempDir())
	runGit(t, dir, "config", "core.hooksPath", "myhooks")

	got := HooksDir(context.Background(), dir)
	want := filepath.Join(dir, "myhooks")
	if got != want {
		t.Fatalf("HooksDir(%s) with core.hooksPath=myhooks = %q, want %q", dir, got, want)
	}
}

func TestHooksDir_LinkedWorktreeResolvesToSharedCommonHooksDir(t *testing.T) {
	startPath, indexRoot := newLinkedWorktreeFixture(t)

	mainHooks := HooksDir(context.Background(), indexRoot)
	if mainHooks == "" {
		t.Fatalf("HooksDir(%s) = \"\", want non-empty", indexRoot)
	}

	wtHooks := HooksDir(context.Background(), startPath)
	if wtHooks == "" {
		t.Fatalf("HooksDir(%s) = \"\", want non-empty", startPath)
	}
	if !filepath.IsAbs(wtHooks) {
		t.Fatalf("HooksDir(%s) = %q, want absolute path (git returns absolute --git-path for linked worktrees)", startPath, wtHooks)
	}

	// HooksDir deliberately does not call realpath (D-04: resolve-relative
	// or passthrough-absolute only). git itself internally realpath-resolves
	// the absolute --git-path it returns for a linked worktree, while
	// mainHooks here is built by joining the caller-supplied indexRoot
	// (unresolved). On a host where TMPDIR sits behind a symlink (e.g.
	// macOS /var -> /private/var) the two raw strings can differ only by
	// that symlink hop, so compare through EvalSymlinks to assert the
	// functional property (same underlying directory) without smuggling
	// realpath into HooksDir itself.
	resolvedMain, err := filepath.EvalSymlinks(mainHooks)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", mainHooks, err)
	}
	resolvedWt, err := filepath.EvalSymlinks(wtHooks)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", wtHooks, err)
	}
	if resolvedWt != resolvedMain {
		t.Fatalf("HooksDir(%s) = %q, want the shared common hooks dir %q", startPath, wtHooks, mainHooks)
	}
	resolvedIndexRoot, err := filepath.EvalSymlinks(indexRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", indexRoot, err)
	}
	if !strings.HasPrefix(resolvedWt, resolvedIndexRoot) {
		t.Fatalf("HooksDir(%s) = %q, want it to live under the main checkout %q", startPath, wtHooks, indexRoot)
	}
}

func TestHooksDir_EmptyForNonGitDir(t *testing.T) {
	dir := t.TempDir()

	if got := HooksDir(context.Background(), dir); got != "" {
		t.Fatalf("HooksDir(%s) = %q, want \"\"", dir, got)
	}
}
