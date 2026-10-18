package agents

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGitA mirrors internal/gitmeta/fixtures_test.go's runGit, a
// package-local copy since test helpers are not importable across
// packages: hermetic flags, and any git failure skips the calling test.
func runGitA(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
	}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git %v failed (git missing or fixture unsupported here): %v: %s", args, err, out)
	}
}

func gitRepoA(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitA(t, dir, "init")
	writeFile(t, filepath.Join(dir, "a.go"), "package a\n")
	runGitA(t, dir, "add", "-A")
	runGitA(t, dir, "commit", "-m", "init")
	return dir
}

// TestSessionNudgeLinkedWorktree (#139): in a linked worktree root the
// nudge fires when the main checkout is indexed, and stays silent
// otherwise. A git stub first on PATH counts git runs: one in a linked
// worktree (the positive control), none in an ordinary repository.
func TestSessionNudgeLinkedWorktree(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not on PATH: %v", err)
	}

	worktreeOf := func(main string) string {
		wt := filepath.Join(t.TempDir(), "wt")
		runGitA(t, main, "worktree", "add", "-b", "wt", wt)
		return wt
	}

	indexedMain := gitRepoA(t, filepath.Join(t.TempDir(), "main"))
	if err := os.Mkdir(filepath.Join(indexedMain, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexedWt := worktreeOf(indexedMain)
	unindexedWt := worktreeOf(gitRepoA(t, filepath.Join(t.TempDir(), "main")))
	homeMain := gitRepoA(t, filepath.Join(t.TempDir(), "main"))
	if err := os.Mkdir(filepath.Join(homeMain, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	homeWt := worktreeOf(homeMain)

	tmp := t.TempDir()
	src := gitRepoA(t, filepath.Join(tmp, "src"))
	bare := filepath.Join(tmp, "bare.git")
	runGitA(t, tmp, "clone", "--bare", src, bare)
	if err := os.Mkdir(filepath.Join(bare, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	bareWt := filepath.Join(tmp, "bare-wt")
	runGitA(t, bare, "worktree", "add", bareWt, "main")

	plain := gitRepoA(t, filepath.Join(t.TempDir(), "plain"))

	bin := t.TempDir()
	runs := filepath.Join(t.TempDir(), "git-runs")
	stub := "#!/bin/sh\necho x >> '" + runs + "'\nexec '" + realGit + "' \"$@\"\n"
	writeStub(t, filepath.Join(bin, "git"), stub, 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cases := []struct {
		name     string
		dir      string
		home     string
		want     string
		wantRuns int
	}{
		{name: "worktree of an indexed main checkout", dir: indexedWt, want: nudgeLine + "\n", wantRuns: 1},
		{name: "worktree of an unindexed main checkout", dir: unindexedWt, wantRuns: 1},
		{name: "worktree of a bare repository", dir: bareWt, wantRuns: 1},
		{name: "main checkout is the home directory", dir: homeWt, home: homeMain, wantRuns: 1},
		{name: "ordinary unindexed repository", dir: plain, wantRuns: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.home != "" {
				t.Setenv("HOME", tc.home)
			}
			if err := os.Remove(runs); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			stdout, stderr, exit, err := runSessionNudge(t, tc.dir, true)
			if err != nil {
				t.Fatalf("runSessionNudge: %v", err)
			}
			if stdout != tc.want || stderr != "" || exit != 0 {
				t.Fatalf("stdout = %q, stderr = %q, exit = %d; want stdout %q, no stderr, exit 0", stdout, stderr, exit, tc.want)
			}
			data, err := os.ReadFile(runs)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if got := strings.Count(string(data), "x\n"); got != tc.wantRuns {
				t.Fatalf("git ran %d times, want %d", got, tc.wantRuns)
			}
		})
	}
}
