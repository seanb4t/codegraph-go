package discover

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// linkFixture builds a repository with main.go at its root, runs layout on
// the root to add an index layout, and returns the root. A file that a
// layout plants is a Go file, so discovery would index it if the walk
// reached it.
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

func storeLinkLayout(t *testing.T, root string) {
	writeFixtureFile(t, root, "store-real/planted.go", "package store\n")
	writeFixtureFile(t, root, "store-real.swap", "")
	writeFixtureFile(t, root, "store-real.write", "")
	writeFixtureFile(t, root, "store-real.scratch-abc/store/planted.go", "package store\n")
	writeFixtureFile(t, root, "store-real.scratch.pending-def/planted.go", "package store\n")
	if err := os.Mkdir(filepath.Join(root, ".codegraph"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	symlink(t, filepath.Join("..", "store-real"), filepath.Join(root, ".codegraph", "store"))
}

func TestDiscoverAllSkipsTheIndexBehindALink(t *testing.T) {
	for _, tc := range []struct {
		name   string
		layout func(t *testing.T, root string)
		// index is the name that every path of the index starts with.
		index string
		// viaLink walks the root through a link to its parent.
		viaLink bool
	}{
		{"a .codegraph link to a directory with a plain name", func(t *testing.T, root string) {
			writeFixtureFile(t, root, "cg-real/store/planted.go", "package store\n")
			writeFixtureFile(t, root, "cg-real/store.scratch-x/store/planted.go", "package store\n")
			writeFixtureFile(t, root, "cg-real/store/OPTIONS-000008", "")
			symlink(t, "cg-real", filepath.Join(root, ".codegraph"))
		}, "cg-real", false},
		{"a store link to a directory with a plain name", storeLinkLayout, "store-real", false},
		{"a store link, with a link to the parent of the root", storeLinkLayout, "store-real", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := linkFixture(t, func(root string) { tc.layout(t, root) })
			if tc.viaLink {
				link := filepath.Join(t.TempDir(), "parent")
				symlink(t, filepath.Dir(root), link)
				root = filepath.Join(link, filepath.Base(root))
			}

			d, err := DiscoverAll(root)
			if err != nil {
				t.Fatalf("DiscoverAll: %v", err)
			}
			found := false
			for _, f := range d.Files {
				if strings.HasPrefix(f.RelPath, tc.index) {
					t.Errorf("discovered %s, a file of the index", f.RelPath)
				}
				found = found || f.RelPath == "main.go"
			}
			if !found {
				t.Fatalf("main.go not discovered: %+v", d.Files)
			}
			for _, e := range d.Excluded {
				if strings.HasPrefix(e.GetPath(), tc.index) {
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

// TestDirFilterKeepsADirectoryThatDiffersFromTheIndexInCase covers a
// case-sensitive file system, where a name that differs from the index in
// letter case is another directory.
func TestDirFilterKeepsADirectoryThatDiffersFromTheIndexInCase(t *testing.T) {
	root := linkFixture(t, func(root string) {
		writeFixtureFile(t, root, "cg-real/store/planted.go", "package store\n")
	})
	if _, err := os.Stat(filepath.Join(root, "CG-REAL")); err == nil {
		t.Skip("the file system is case-insensitive")
	}
	writeFixtureFile(t, root, "CG-REAL/kept.go", "package kept\n")
	symlink(t, "cg-real", filepath.Join(root, ".codegraph"))

	f, err := NewDirFilter(root)
	if err != nil {
		t.Fatalf("NewDirFilter: %v", err)
	}
	if !f.SkipDir(filepath.Join(root, "cg-real")) {
		t.Errorf("SkipDir(cg-real) = false for the index")
	}
	if f.SkipDir(filepath.Join(root, "CG-REAL")) {
		t.Errorf("SkipDir(CG-REAL) = true for a directory that is not the index")
	}
}

func TestDiscoverAllRecordsAnIndexThatANameRuleSkips(t *testing.T) {
	root := linkFixture(t, func(root string) {
		writeFixtureFile(t, root, ".codegraph/store/planted.go", "package store\n")
	})
	d, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	var got []string
	for _, e := range d.Excluded {
		got = append(got, e.GetPath())
		if e.GetPath() == ".codegraph" && e.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX {
			t.Errorf("the exclusion of .codegraph has reason %v, want DIR_DOTPREFIX", e.GetReason())
		}
	}
	if len(got) != 1 || got[0] != ".codegraph" {
		t.Fatalf("exclusions = %q, want one for .codegraph", got)
	}
}

func TestNewDirFilterRefusesAnIndexOutsideTheRoot(t *testing.T) {
	root := linkFixture(t, func(root string) {
		symlink(t, t.TempDir(), filepath.Join(root, ".codegraph"))
	})
	if _, err := NewDirFilter(root); !errors.Is(err, indexdir.ErrOutsideRoot) {
		t.Fatalf("NewDirFilter = %v, want indexdir.ErrOutsideRoot", err)
	}
	if _, err := DiscoverAll(root); !errors.Is(err, indexdir.ErrOutsideRoot) {
		t.Fatalf("DiscoverAll = %v, want indexdir.ErrOutsideRoot", err)
	}
}
