package query

import (
	"os"
	"path/filepath"
	"testing"
)

// indexEntryStates are the shapes IndexDirs resolves for .codegraph or its
// store: no entry, a directory, a file, absolute and relative links inside
// and outside root, a chain of two links, a dangling link and a loop.
var indexEntryStates = []string{
	"absent", "dir", "file", "inside", "relative", "chain",
	"outside", "dotdot", "dangling", "self",
}

// TestIndexDirsIsStableOnAnUnchangedTree is the property the watcher's
// index check depends on (#375, design D6): on a tree that does not
// change, two calls of IndexDirs agree. If they did not, a watcher would
// stop on an index that no pull changed. The test builds every pair of
// states.
func TestIndexDirsIsStableOnAnUnchangedTree(t *testing.T) {
	resolved, refused := 0, 0
	for _, index := range indexEntryStates {
		for _, store := range indexEntryStates {
			t.Run(index+"/"+store, func(t *testing.T) {
				top := t.TempDir()
				root := filepath.Join(top, "repo")
				outside := filepath.Join(top, "outside")
				for _, dir := range []string{outside, filepath.Join(root, ".cg-real")} {
					if err := os.MkdirAll(dir, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				indexPath := filepath.Join(root, IndexDirName)
				makeIndexEntry(t, index, indexPath, root, outside)
				if info, err := os.Stat(indexPath); err == nil && info.IsDir() {
					makeIndexEntry(t, store, filepath.Join(indexPath, StoreDirName), root, outside)
				}

				first, err1 := IndexDirs(root)
				second, err2 := IndexDirs(root)
				switch {
				case (err1 == nil) != (err2 == nil):
					t.Fatalf("IndexDirs = %v then %v", err1, err2)
				case err1 != nil && err1.Error() != err2.Error():
					t.Fatalf("IndexDirs errors differ: %q then %q", err1, err2)
				case first != second:
					t.Fatalf("IndexDirs = %+v then %+v", first, second)
				}
				if err1 == nil {
					resolved++
				} else {
					refused++
				}
			})
		}
	}
	if resolved == 0 || refused == 0 {
		t.Fatalf("compared %d resolved and %d refused trees, want both above zero", resolved, refused)
	}
}

// makeIndexEntry makes path one of indexEntryStates.
func makeIndexEntry(t *testing.T, state, path, root, outside string) {
	t.Helper()
	var err error
	switch state {
	case "absent":
	case "dir":
		err = os.MkdirAll(path, 0o755)
	case "file":
		err = os.WriteFile(path, nil, 0o644)
	case "inside":
		err = os.Symlink(filepath.Join(root, ".cg-real"), path)
	case "relative":
		err = os.Symlink(relTo(t, path, filepath.Join(root, ".cg-real")), path)
	case "chain":
		hop := filepath.Join(root, ".chain-"+filepath.Base(path))
		if err = os.Symlink(filepath.Join(root, ".cg-real"), hop); err == nil {
			err = os.Symlink(hop, path)
		}
	case "outside":
		err = os.Symlink(outside, path)
	case "dotdot":
		err = os.Symlink(relTo(t, path, outside), path)
	case "dangling":
		err = os.Symlink(filepath.Join(outside, "missing"), path)
	case "self":
		err = os.Symlink(filepath.Base(path), path)
	default:
		t.Fatalf("unknown state %q", state)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// relTo returns target relative to the directory that holds link.
func relTo(t *testing.T, link, target string) string {
	t.Helper()
	rel, err := filepath.Rel(filepath.Dir(link), target)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}
