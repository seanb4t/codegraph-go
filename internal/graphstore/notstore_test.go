package graphstore

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

// treeOf returns the relative path and contents of every file under dir,
// and every directory as a path ending in a separator.
func treeOf(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			out = append(out, rel+string(filepath.Separator))
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, rel+"="+string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// nonStoreDirs builds directories that hold files and no store: a source
// directory and a git directory.
func nonStoreDirs(t *testing.T, parent string) map[string]string {
	t.Helper()
	src := filepath.Join(parent, "src")
	git := filepath.Join(parent, ".git")
	for path, data := range map[string]string{
		filepath.Join(src, "main.go"):               "package main\n",
		filepath.Join(git, "HEAD"):                  "ref: refs/heads/main\n",
		filepath.Join(git, "objects", "ab", "cdef"): "blob",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return map[string]string{"src": src, ".git": git}
}

func TestReplaceStoreRefusesDirectoryThatIsNotAStore(t *testing.T) {
	for _, viaLink := range []bool{false, true} {
		for _, name := range []string{"src", ".git"} {
			t.Run(name+map[bool]string{false: "", true: " through a link"}[viaLink], func(t *testing.T) {
				parent := t.TempDir()
				target := nonStoreDirs(t, parent)[name]
				live := target
				if viaLink {
					live = filepath.Join(parent, "store")
					if err := os.Symlink(target, live); err != nil {
						t.Fatal(err)
					}
				}
				staging := filepath.Join(parent, "staging")
				stagingExport := buildStore(t, staging, "new")
				before := treeOf(t, target)

				err := ReplaceStore(staging, live)
				var notStore *NotStoreError
				if !errors.Is(err, ErrNotStore) || !errors.As(err, &notStore) {
					t.Fatalf("ReplaceStore = %v, want a *NotStoreError", err)
				}
				if after := treeOf(t, target); !slices.Equal(before, after) {
					t.Fatalf("%s changed:\nbefore %q\nafter  %q", target, before, after)
				}
				if !bytes.Equal(exportAt(t, staging), stagingExport) {
					t.Fatalf("the staging store changed in a ReplaceStore that failed")
				}
			})
		}
	}
}

func TestReplaceStoreAcceptsEmptyLiveDirectory(t *testing.T) {
	parent := t.TempDir()
	live, staging := filepath.Join(parent, "store"), filepath.Join(parent, "staging")
	if err := os.Mkdir(live, 0o755); err != nil {
		t.Fatal(err)
	}
	newExport := buildStore(t, staging, "new")

	if err := ReplaceStore(staging, live); err != nil {
		t.Fatalf("ReplaceStore: %v", err)
	}
	if !bytes.Equal(exportAt(t, live), newExport) {
		t.Fatalf("the live path does not hold the new store")
	}
}

func TestOpenRefusesDirectoryThatIsNotAStore(t *testing.T) {
	for name, dir := range nonStoreDirs(t, t.TempDir()) {
		t.Run(name, func(t *testing.T) {
			before := treeOf(t, dir)
			for _, opener := range []struct {
				name string
				open func(string) (GraphStore, error)
			}{{"Open", Open}, {"OpenStaging", OpenStaging}} {
				s, err := opener.open(dir)
				if err == nil {
					s.Close()
				}
				if after := treeOf(t, dir); !slices.Equal(before, after) {
					t.Fatalf("%s changed %s:\nbefore %q\nafter  %q", opener.name, dir, before, after)
				}
				if !errors.Is(err, ErrNotStore) {
					t.Fatalf("%s = %v, want ErrNotStore", opener.name, err)
				}
			}
			if _, err := OpenExisting(dir); !errors.Is(err, ErrNotFound) {
				t.Fatalf("OpenExisting = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestCheckLiveStore(t *testing.T) {
	parent := t.TempDir()
	dirs := nonStoreDirs(t, parent)
	if err := CheckLiveStore(dirs["src"]); !errors.Is(err, ErrNotStore) {
		t.Fatalf("CheckLiveStore(src) = %v, want ErrNotStore", err)
	}

	withLink := filepath.Join(parent, "linked")
	buildStore(t, withLink, "n")
	outside, _ := outsideFile(t)
	if err := os.Symlink(outside, filepath.Join(withLink, "extra")); err != nil {
		t.Fatal(err)
	}
	if err := CheckLiveStore(withLink); !errors.Is(err, nofollow.ErrSymlink) {
		t.Fatalf("CheckLiveStore(store with a link) = %v, want nofollow.ErrSymlink", err)
	}

	for _, ok := range []string{filepath.Join(parent, "missing"), t.TempDir()} {
		if err := CheckLiveStore(ok); err != nil {
			t.Fatalf("CheckLiveStore(%s) = %v, want nil", ok, err)
		}
	}
	store := filepath.Join(parent, "store")
	buildStore(t, store, "n")
	if err := CheckLiveStore(store); err != nil {
		t.Fatalf("CheckLiveStore(store) = %v, want nil", err)
	}
}
