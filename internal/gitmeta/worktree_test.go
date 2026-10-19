package gitmeta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newBareWorktreeFixture builds a bare clone with one linked worktree. A bare
// repository has no main worktree, so LinkedWorktreeMain reports "" for it.
func newBareWorktreeFixture(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	src := initRepo(t, filepath.Join(tmp, "src"))
	bare := filepath.Join(tmp, "bare.git")
	runGit(t, tmp, "clone", "--bare", src, bare)
	wt := filepath.Join(tmp, "wt")
	runGit(t, bare, "worktree", "add", wt, "main")
	return wt
}

// plantedGitFile creates a directory outside any repository whose .git file
// names gitdir, the way a hand-written or downloaded .git file would.
func plantedGitFile(t *testing.T, gitdir string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: "+gitdir+"\n"), 0o644); err != nil {
		t.Fatalf("write planted .git: %v", err)
	}
	return dir
}

func TestLinkedWorktreeMain(t *testing.T) {
	sibling, siblingMain := newLinkedWorktreeFixture(t)
	nested, nestedMain := newClaudeWorktreeFixture(t)
	submodule, _ := newSubmoduleFixture(t)
	nonGit, _ := newNonGitFixture(t)
	adminDir := filepath.Join(siblingMain, ".git", "worktrees", filepath.Base(sibling))
	if _, err := os.Stat(filepath.Join(adminDir, "gitdir")); err != nil {
		t.Fatalf("fixture: the sibling worktree's admin dir %s has no gitdir file: %v", adminDir, err)
	}

	cases := []struct {
		name string
		dir  string
		want string
	}{
		{"sibling worktree", sibling, resolveExpected(t, siblingMain)},
		{"nested worktree", nested, resolveExpected(t, nestedMain)},
		{"main checkout", siblingMain, ""},
		{"subdirectory of a worktree", mkdir(t, filepath.Join(sibling, "pkg")), ""},
		{"bare repository worktree", newBareWorktreeFixture(t), ""},
		{"submodule", submodule, ""},
		{"outside git", nonGit, ""},
		{"planted .git naming a main git dir", plantedGitFile(t, filepath.Join(siblingMain, ".git")), ""},
		{"planted .git naming another worktree's admin dir", plantedGitFile(t, adminDir), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LinkedWorktreeMain(context.Background(), tc.dir); got != tc.want {
				t.Fatalf("LinkedWorktreeMain(%q) = %q, want %q", tc.dir, got, tc.want)
			}
		})
	}
}

// TestLinkedWorktreeMainRelativePaths covers a worktree whose links git
// wrote as relative paths (`git worktree add --relative-paths`, git 2.48
// and later); runGit skips the test on an older git.
func TestLinkedWorktreeMainRelativePaths(t *testing.T) {
	tmp := t.TempDir()
	main := initRepo(t, filepath.Join(tmp, "main"))
	wt := filepath.Join(tmp, "wt")
	runGit(t, main, "worktree", "add", "--relative-paths", "-b", "rel", wt)

	if got, want := LinkedWorktreeMain(context.Background(), wt), resolveExpected(t, main); got != want {
		t.Fatalf("LinkedWorktreeMain(%q) = %q, want %q", wt, got, want)
	}
}

func TestLinkedWorktreeMainCachesAnswer(t *testing.T) {
	wt, main := newLinkedWorktreeFixture(t)
	want := resolveExpected(t, main)
	if got := LinkedWorktreeMain(context.Background(), wt); got != want {
		t.Fatalf("first LinkedWorktreeMain = %q, want %q", got, want)
	}

	t.Setenv("PATH", t.TempDir())
	if got := LinkedWorktreeMain(context.Background(), wt); got != want {
		t.Fatalf("LinkedWorktreeMain with git unreachable = %q, want the cached %q", got, want)
	}
}

func TestLinkedWorktreeMainDoesNotCacheFailure(t *testing.T) {
	wt, main := newLinkedWorktreeFixture(t)
	path := os.Getenv("PATH")

	t.Setenv("PATH", t.TempDir())
	if got := LinkedWorktreeMain(context.Background(), wt); got != "" {
		t.Fatalf("LinkedWorktreeMain with git unreachable = %q, want \"\"", got)
	}

	t.Setenv("PATH", path)
	if got, want := LinkedWorktreeMain(context.Background(), wt), resolveExpected(t, main); got != want {
		t.Fatalf("LinkedWorktreeMain after git is back = %q, want %q (the failure must not be cached)", got, want)
	}
}

func mkdir(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	return dir
}
