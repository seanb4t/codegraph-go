package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linkFixture builds a repository with main.go at its root and the index
// layout that link describes, and returns the root. Each planted file is a
// Go file, so discovery would index it if the walk reached it.
func linkFixture(t *testing.T, layout func(root string)) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, "main.go", "package main\n")
	layout(root)
	return root
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
}

func TestDiscoverAllSkipsTheIndexBehindALink(t *testing.T) {
	for _, tc := range []struct {
		name   string
		layout func(t *testing.T, root string)
		index  string
	}{
		{"a .codegraph link to a directory with a plain name", func(t *testing.T, root string) {
			writeFixtureFile(t, root, "cg-real/store/planted.go", "package store\n")
			writeFixtureFile(t, root, "cg-real/store.scratch-x/store/planted.go", "package store\n")
			writeFixtureFile(t, root, "cg-real/store/OPTIONS-000008", "")
			symlink(t, "cg-real", filepath.Join(root, ".codegraph"))
		}, "cg-real"},
		{"a store link to a directory with a plain name", func(t *testing.T, root string) {
			writeFixtureFile(t, root, "store-real/planted.go", "package store\n")
			if err := os.Mkdir(filepath.Join(root, ".codegraph"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			symlink(t, filepath.Join("..", "store-real"), filepath.Join(root, ".codegraph", "store"))
		}, "store-real"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := linkFixture(t, func(root string) { tc.layout(t, root) })

			d, err := DiscoverAll(root)
			if err != nil {
				t.Fatalf("DiscoverAll: %v", err)
			}
			found := false
			for _, f := range d.Files {
				if strings.HasPrefix(f.RelPath, tc.index+"/") {
					t.Errorf("discovered %s, a file of the index", f.RelPath)
				}
				found = found || f.RelPath == "main.go"
			}
			if !found {
				t.Fatalf("main.go not discovered: %+v", d.Files)
			}
			for _, e := range d.Excluded {
				if e.GetPath() == tc.index || strings.HasPrefix(e.GetPath(), tc.index+"/") {
					t.Errorf("recorded an exclusion for %s, a path of the index", e.GetPath())
				}
			}
		})
	}
}

func TestDirFilterMatchesTheIndexInAnyLetterCase(t *testing.T) {
	root := linkFixture(t, func(root string) {
		writeFixtureFile(t, root, "cg-real/store/planted.go", "package store\n")
	})
	symlink(t, "CG-REAL", filepath.Join(root, ".codegraph"))
	if _, err := os.Stat(filepath.Join(root, "CG-REAL")); err != nil {
		t.Skipf("the file system is case-sensitive: %v", err)
	}

	f, err := NewDirFilter(root)
	if err != nil {
		t.Fatalf("NewDirFilter: %v", err)
	}
	if !f.SkipDir(filepath.Join(root, "cg-real")) {
		t.Errorf("SkipDir(cg-real) = false for an index that the link names CG-REAL")
	}
	if f.SkipDir(filepath.Join(root, "cg-other")) {
		t.Errorf("SkipDir(cg-other) = true, want false")
	}
}

func TestDirFilterAddsNothingForAnIndexThatANameRuleSkips(t *testing.T) {
	root := linkFixture(t, func(root string) {
		if err := os.MkdirAll(filepath.Join(root, ".codegraph", "store"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	})
	f, err := NewDirFilter(root)
	if err != nil {
		t.Fatalf("NewDirFilter: %v", err)
	}
	if len(f.index) != 0 {
		t.Errorf("index = %q, want none for a real .codegraph directory", f.index)
	}
	if !f.SkipDir(filepath.Join(root, ".codegraph")) {
		t.Errorf("SkipDir(.codegraph) = false, want true by name")
	}
}

func TestNewDirFilterRefusesAnIndexOutsideTheRoot(t *testing.T) {
	root := linkFixture(t, func(root string) {
		symlink(t, t.TempDir(), filepath.Join(root, ".codegraph"))
	})
	if _, err := NewDirFilter(root); err == nil {
		t.Fatalf("NewDirFilter = nil error for a .codegraph link outside the root")
	}
	if _, err := DiscoverAll(root); err == nil {
		t.Fatalf("DiscoverAll = nil error for a .codegraph link outside the root")
	}
}
