package query

import (
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// TestIndexDirsIsStableOnAnUnchangedTree is the property the watcher's
// index check depends on (#375): on a tree that does not change, two calls
// of IndexDirs agree. If they did not, a watcher would stop on an index
// that no pull changed.
func TestIndexDirsIsStableOnAnUnchangedTree(t *testing.T) {
	base := t.TempDir()
	resolved, refused := 0, 0
	rapid.Check(t, func(rt *rapid.T) {
		top, err := os.MkdirTemp(base, "tree")
		if err != nil {
			rt.Fatal(err)
		}
		root := filepath.Join(top, "repo")
		outside := filepath.Join(top, "outside")
		for _, dir := range []string{outside, filepath.Join(root, ".cg-a"), filepath.Join(root, ".cg-b")} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				rt.Fatal(err)
			}
		}
		index := filepath.Join(root, IndexDirName)
		drawIndexEntry(rt, "index", index, root, outside)
		if info, err := os.Stat(index); err == nil && info.IsDir() {
			drawIndexEntry(rt, "store", filepath.Join(index, StoreDirName), root, outside)
		}

		first, err1 := IndexDirs(root)
		second, err2 := IndexDirs(root)
		switch {
		case (err1 == nil) != (err2 == nil):
			rt.Fatalf("IndexDirs = %v then %v", err1, err2)
		case err1 != nil && err1.Error() != err2.Error():
			rt.Fatalf("IndexDirs errors differ: %q then %q", err1, err2)
		case first != second:
			rt.Fatalf("IndexDirs = %+v then %+v", first, second)
		}
		if err1 == nil {
			resolved++
		} else {
			refused++
		}
	})
	if resolved == 0 || refused == 0 {
		t.Fatalf("compared %d resolved and %d refused trees, want both above zero", resolved, refused)
	}
}

// drawIndexEntry makes path one of the states a pull can leave it in.
func drawIndexEntry(rt *rapid.T, label, path, root, outside string) {
	var err error
	switch rapid.SampledFrom([]string{"absent", "dir", "inside", "outside", "dangling", "self"}).Draw(rt, label) {
	case "absent":
	case "dir":
		err = os.MkdirAll(path, 0o755)
	case "inside":
		target := rapid.SampledFrom([]string{".cg-a", ".cg-b"}).Draw(rt, label+" target")
		err = os.Symlink(filepath.Join(root, target), path)
	case "outside":
		err = os.Symlink(outside, path)
	case "dangling":
		err = os.Symlink(filepath.Join(outside, "missing"), path)
	case "self":
		err = os.Symlink(filepath.Base(path), path)
	}
	if err != nil {
		rt.Fatal(err)
	}
}
