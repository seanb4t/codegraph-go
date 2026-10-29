package wtindex

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
	}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, out)
	}
	return string(out)
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	return err == nil
}

func TestSeedDirGitignoreFirst(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")

	d, err := newSeedDir(root)
	if err != nil {
		t.Fatalf("newSeedDir: %v", err)
	}
	defer func() {
		if err := d.remove(); err != nil {
			t.Errorf("remove the seed directory: %v", err)
		}
	}()
	if err := os.WriteFile(filepath.Join(d.path, "probe"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write probe: %v", err)
	}

	if !strings.HasPrefix(filepath.Base(d.path), seedPrefix) || filepath.Dir(d.path) != root {
		t.Fatalf("seed directory = %s; want %s/%s*", d.path, root, seedPrefix)
	}
	got, err := os.ReadFile(filepath.Join(d.path, ".gitignore"))
	if err != nil || string(got) != "*\n" {
		t.Fatalf(".gitignore = %q, %v; want %q", got, err, "*\n")
	}
	if out := runGit(t, root, "status", "--porcelain", "--untracked-files=all"); out != "" {
		t.Errorf("git status lists the seed directory:\n%s", out)
	}
	if lock, err := lockFile(filepath.Join(d.path, lockName)); err == nil {
		lock.Close()
		t.Errorf("took the seed directory's lock while newSeedDir's holder is open")
	}
}

func TestSweepRemovesOnlyUnlockedLeftovers(t *testing.T) {
	root := t.TempDir()
	leftover := filepath.Join(root, seedPrefix+"crashed")
	held := filepath.Join(root, seedPrefix+"building")
	other := filepath.Join(root, "codegraph.seed-not-ours")
	for _, dir := range []string{leftover, held, other} {
		if err := os.MkdirAll(filepath.Join(dir, storeDirName), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(leftover, lockName), nil, 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	lock, err := lockFile(filepath.Join(held, lockName))
	if err != nil {
		t.Fatalf("lockFile: %v", err)
	}
	defer lock.Close()

	sweep(root)

	if exists(t, leftover) {
		t.Errorf("sweep kept %s, whose lock no one holds", leftover)
	}
	if !exists(t, filepath.Join(held, storeDirName)) {
		t.Errorf("sweep removed %s, whose lock a build holds", held)
	}
	if !exists(t, other) {
		t.Errorf("sweep removed %s, which is not a seed directory", other)
	}
}

func TestPlaceMovesSeedIntoPlace(t *testing.T) {
	root := t.TempDir()
	d, err := newSeedDir(root)
	if err != nil {
		t.Fatalf("newSeedDir: %v", err)
	}
	if err := os.MkdirAll(d.storePath(), 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}

	if err := place(d, root); err != nil {
		t.Fatalf("place: %v", err)
	}
	target := filepath.Join(root, indexDirName)
	if !exists(t, filepath.Join(target, storeDirName)) || !exists(t, filepath.Join(target, ".gitignore")) {
		t.Errorf("%s lacks the seed's store or .gitignore", target)
	}
	if exists(t, filepath.Join(target, lockName)) {
		t.Errorf("%s still holds the build's lock file", target)
	}
	if exists(t, d.path) {
		t.Errorf("seed directory %s still exists", d.path)
	}
}

func TestPlaceYieldsToExistingIndex(t *testing.T) {
	for _, tc := range []struct {
		name   string
		marker bool
	}{
		{"an index", true},
		{"an empty directory", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, indexDirName)
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			marker := filepath.Join(target, "marker")
			if tc.marker {
				if err := os.WriteFile(marker, []byte("theirs"), 0o644); err != nil {
					t.Fatalf("write marker: %v", err)
				}
			}
			d, err := newSeedDir(root)
			if err != nil {
				t.Fatalf("newSeedDir: %v", err)
			}
			if err := os.MkdirAll(d.storePath(), 0o755); err != nil {
				t.Fatalf("mkdir store: %v", err)
			}

			if err := place(d, root); !errors.Is(err, ErrIndexExists) {
				t.Fatalf("place error = %v; want ErrIndexExists", err)
			}
			if tc.marker {
				if got, err := os.ReadFile(marker); err != nil || string(got) != "theirs" {
					t.Errorf("existing index's marker = %q, %v; want it unchanged", got, err)
				}
			}
			if exists(t, filepath.Join(target, storeDirName)) {
				t.Errorf("place moved the seed's store into the existing index")
			}
			if exists(t, d.path) {
				t.Errorf("seed directory %s still exists", d.path)
			}
		})
	}
}
