package gitmeta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newBareWorktreeFixture builds a bare clone with one linked worktree. A bare
// repository has no main worktree, so MainWorktree reports "" from the
// linked one.
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

func TestMainWorktree(t *testing.T) {
	sibling, siblingMain := newLinkedWorktreeFixture(t)
	nested, nestedMain := newClaudeWorktreeFixture(t)
	nonGit, _ := newNonGitFixture(t)

	cases := []struct {
		name string
		dir  string
		want string
	}{
		{"sibling worktree", sibling, resolveExpected(t, siblingMain)},
		{"nested worktree", nested, resolveExpected(t, nestedMain)},
		{"main checkout", siblingMain, resolveExpected(t, siblingMain)},
		{"subdirectory of a worktree", mkdir(t, filepath.Join(sibling, "pkg")), resolveExpected(t, siblingMain)},
		{"bare repository worktree", newBareWorktreeFixture(t), ""},
		{"outside git", nonGit, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MainWorktree(context.Background(), tc.dir); got != tc.want {
				t.Fatalf("MainWorktree(%q) = %q, want %q", tc.dir, got, tc.want)
			}
		})
	}
}

func TestMainWorktreeCachesAnswer(t *testing.T) {
	wt, main := newLinkedWorktreeFixture(t)
	want := resolveExpected(t, main)
	if got := MainWorktree(context.Background(), wt); got != want {
		t.Fatalf("first MainWorktree = %q, want %q", got, want)
	}

	t.Setenv("PATH", t.TempDir())
	if got := MainWorktree(context.Background(), wt); got != want {
		t.Fatalf("MainWorktree with git unreachable = %q, want the cached %q", got, want)
	}
}

func TestMainWorktreeDoesNotCacheFailure(t *testing.T) {
	wt, main := newLinkedWorktreeFixture(t)
	path := os.Getenv("PATH")

	t.Setenv("PATH", t.TempDir())
	if got := MainWorktree(context.Background(), wt); got != "" {
		t.Fatalf("MainWorktree with git unreachable = %q, want \"\"", got)
	}

	t.Setenv("PATH", path)
	if got, want := MainWorktree(context.Background(), wt), resolveExpected(t, main); got != want {
		t.Fatalf("MainWorktree after git is back = %q, want %q (the failure must not be cached)", got, want)
	}
}

func mkdir(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	return dir
}
