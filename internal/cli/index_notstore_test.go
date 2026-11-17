package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

func entryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestIndexCommandsRefuseStoreLinkedToRepositoryDirectory(t *testing.T) {
	for name, target := range map[string]string{"source directory": "src", "git directory": "gitdir", "repository root": "."} {
		t.Run(name, func(t *testing.T) {
			dir := copyFixture(t)
			for path, data := range map[string]string{
				filepath.Join(dir, "src", "keep.txt"):  "keep me\n",
				filepath.Join(dir, "gitdir", "HEAD"):   "ref: refs/heads/main\n",
				filepath.Join(dir, "gitdir", "config"): "[core]\n",
			} {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(filepath.Join(dir, codegraphDirName), 0o755); err != nil {
				t.Fatal(err)
			}
			symlinkT(t, filepath.Join("..", target), filepath.Join(dir, codegraphDirName, storeDirName))
			targetDir := filepath.Join(dir, target)
			before := entryNames(t, targetDir)
			parentBefore := entryNames(t, filepath.Dir(dir))

			for _, args := range [][]string{{"sync", dir}, {"index", "--force", dir}} {
				_, _, err := execCmd(args...)
				if !errors.Is(err, graphstore.ErrNotStore) {
					t.Fatalf("%v = %v, want graphstore.ErrNotStore", args, err)
				}
			}
			if after := entryNames(t, targetDir); !slices.Equal(before, after) {
				t.Fatalf("%s changed: before %q, after %q", targetDir, before, after)
			}
			if after := entryNames(t, filepath.Dir(dir)); !slices.Equal(parentBefore, after) {
				t.Fatalf("the parent of the repository changed: before %q, after %q", parentBefore, after)
			}
			if data, err := os.ReadFile(filepath.Join(dir, "src", "keep.txt")); err != nil || string(data) != "keep me\n" {
				t.Fatalf("src/keep.txt = %q, %v, want it unchanged", data, err)
			}
		})
	}
}
