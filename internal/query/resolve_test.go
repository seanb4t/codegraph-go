package query

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitRepo creates dir as a git repository with one commit, so worktrees
// can branch from it.
func gitRepo(t *testing.T, dir string) string {
	t.Helper()
	mkdirAll(t, dir)
	runGitW(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	runGitW(t, dir, "add", "-A")
	runGitW(t, dir, "commit", "-m", "init")
	return dir
}

// markIndexed creates dir/.codegraph. Resolution only looks for the
// directory, so no index is built.
func markIndexed(t *testing.T, dir string) {
	t.Helper()
	mkdirAll(t, filepath.Join(dir, codegraphDirName))
}

func mkdirAll(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return dir
}

func realpathT(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", p, err)
	}
	return r
}

// siblingWorktree returns a main checkout and a linked worktree beside it,
// outside it.
func siblingWorktree(t *testing.T) (main, wt string) {
	t.Helper()
	tmp := t.TempDir()
	main = gitRepo(t, filepath.Join(tmp, "main"))
	wt = filepath.Join(tmp, "wt")
	runGitW(t, main, "worktree", "add", "-b", "wt", wt)
	return main, wt
}

func TestResolveIndexWorktreeFallback(t *testing.T) {
	wantResolution := func(t *testing.T, start string, want Resolution) {
		t.Helper()
		got, err := ResolveIndex(start)
		if err != nil {
			t.Fatalf("ResolveIndex(%s): unexpected error: %v", start, err)
		}
		if got != want {
			t.Fatalf("ResolveIndex(%s) = %+v, want %+v", start, got, want)
		}
	}
	wantNotInitialized := func(t *testing.T, start string) {
		t.Helper()
		got, err := ResolveIndex(start)
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("ResolveIndex(%s) = %+v, %v, want ErrNotInitialized", start, got, err)
		}
	}

	t.Run("a worktree outside the main checkout", func(t *testing.T) {
		main, wt := siblingWorktree(t)
		markIndexed(t, main)
		start := mkdirAll(t, filepath.Join(wt, "pkg"))
		wantResolution(t, start, Resolution{IndexRoot: realpathT(t, main), ConfineRoot: wt})

		dir, err := ResolveCodegraphDir(start)
		if err != nil || dir != realpathT(t, main) {
			t.Fatalf("ResolveCodegraphDir(%s) = %q, %v, want %q", start, dir, err, realpathT(t, main))
		}
	})

	t.Run("a worktree with its own index", func(t *testing.T) {
		main, wt := siblingWorktree(t)
		markIndexed(t, main)
		markIndexed(t, wt)
		wantResolution(t, wt, Resolution{IndexRoot: wt, ConfineRoot: wt})
	})

	t.Run("a worktree inside the main checkout", func(t *testing.T) {
		main := gitRepo(t, filepath.Join(t.TempDir(), "main"))
		markIndexed(t, main)
		wt := filepath.Join(main, ".worktrees", "wt")
		runGitW(t, main, "worktree", "add", "-b", "wt", wt)
		wantResolution(t, wt, Resolution{IndexRoot: main, ConfineRoot: main})
	})

	t.Run("an indexed ancestor comes first", func(t *testing.T) {
		tmp := t.TempDir()
		main := gitRepo(t, filepath.Join(tmp, "main"))
		markIndexed(t, main)
		ancestor := mkdirAll(t, filepath.Join(tmp, "dir"))
		markIndexed(t, ancestor)
		wt := filepath.Join(ancestor, "wt")
		runGitW(t, main, "worktree", "add", "-b", "wt", wt)
		wantResolution(t, wt, Resolution{IndexRoot: ancestor, ConfineRoot: ancestor})
	})

	t.Run("the main worktree is not indexed", func(t *testing.T) {
		_, wt := siblingWorktree(t)
		wantNotInitialized(t, wt)
	})

	t.Run("a worktree of a bare repository", func(t *testing.T) {
		tmp := t.TempDir()
		src := gitRepo(t, filepath.Join(tmp, "src"))
		bare := filepath.Join(tmp, "bare.git")
		runGitW(t, tmp, "clone", "--bare", src, bare)
		markIndexed(t, bare)
		wt := filepath.Join(tmp, "wt")
		runGitW(t, bare, "worktree", "add", wt, "main")
		wantNotInitialized(t, wt)
	})

	t.Run("the main worktree is the home directory", func(t *testing.T) {
		main, wt := siblingWorktree(t)
		markIndexed(t, main)
		t.Setenv("HOME", main)
		wantNotInitialized(t, wt)
	})

	t.Run("git is unavailable", func(t *testing.T) {
		main, wt := siblingWorktree(t)
		markIndexed(t, main)
		t.Setenv("PATH", t.TempDir())
		wantNotInitialized(t, wt)
	})

	t.Run("a submodule outside any indexed directory", func(t *testing.T) {
		tmp := t.TempDir()
		parent := gitRepo(t, filepath.Join(tmp, "parent"))
		child := gitRepo(t, filepath.Join(tmp, "child"))
		runGitW(t, parent, "submodule", "add", child, "sub")
		wantNotInitialized(t, filepath.Join(parent, "sub"))
	})
}

// TestResolveIndexRunsGitOnlyInLinkedWorktree puts a git stub first on
// PATH that touches a marker and then runs the real git. The sibling
// worktree is the positive control: it must leave the marker, so the two
// cases that must not cannot pass just because the stub never ran.
func TestResolveIndexRunsGitOnlyInLinkedWorktree(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not on PATH: %v", err)
	}

	main, wt := siblingWorktree(t)
	markIndexed(t, main)
	indexed := gitRepo(t, filepath.Join(t.TempDir(), "indexed"))
	markIndexed(t, indexed)
	plain := gitRepo(t, filepath.Join(t.TempDir(), "plain"))

	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "git-ran")
	stub := "#!/bin/sh\n: > '" + marker + "'\nexec '" + realGit + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write git stub: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cases := []struct {
		name    string
		start   string
		wantGit bool
	}{
		{"an unindexed sibling worktree", wt, true},
		{"an indexed repository", indexed, false},
		{"an unindexed repository with a .git directory", plain, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
				t.Fatalf("remove marker: %v", err)
			}
			_, _ = ResolveIndex(tc.start)
			_, statErr := os.Stat(marker)
			if ran := statErr == nil; ran != tc.wantGit {
				t.Fatalf("ResolveIndex(%s) ran git = %v, want %v", tc.start, ran, tc.wantGit)
			}
		})
	}
}

func TestProjectIndexed(t *testing.T) {
	indexedMain, indexedWt := siblingWorktree(t)
	markIndexed(t, indexedMain)
	_, unindexedWt := siblingWorktree(t)
	plain := gitRepo(t, filepath.Join(t.TempDir(), "plain"))
	parent := gitRepo(t, filepath.Join(t.TempDir(), "parent"))
	markIndexed(t, parent)

	cases := []struct {
		name string
		dir  string
		want bool
	}{
		{"an indexed directory", indexedMain, true},
		{"a sibling worktree root of an indexed main", indexedWt, true},
		{"a worktree of an unindexed main", unindexedWt, false},
		{"an unindexed repository", plain, false},
		{"a subdirectory of an indexed repository", mkdirAll(t, filepath.Join(parent, "pkg")), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ProjectIndexed(tc.dir); got != tc.want {
				t.Fatalf("ProjectIndexed(%s) = %v, want %v", tc.dir, got, tc.want)
			}
		})
	}
}
