package graphstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

// outsideFile writes a file outside every store and returns its path and
// contents.
func outsideFile(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "outside")
	data := []byte("keep me\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func assertFileUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func TestOpenRefusesSymlinkInStore(t *testing.T) {
	for _, name := range []string{"LOCK", "OPTIONS-000001", "MANIFEST-000001"} {
		t.Run(name, func(t *testing.T) {
			outside, data := outsideFile(t)
			dir := filepath.Join(t.TempDir(), "store")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}

			for _, opener := range []struct {
				name string
				open func(string) (GraphStore, error)
			}{{"Open", Open}, {"OpenStaging", OpenStaging}} {
				s, err := opener.open(dir)
				if err == nil {
					s.Close()
				}
				assertFileUnchanged(t, outside, data)
				if !errors.Is(err, nofollow.ErrSymlink) {
					t.Fatalf("%s = %v, want nofollow.ErrSymlink", opener.name, err)
				}
			}
		})
	}
}

func TestOpenExistingRefusesSymlinkInStore(t *testing.T) {
	outside, data := outsideFile(t)
	dir := filepath.Join(t.TempDir(), "store")
	buildStore(t, dir, "n")
	if err := os.Remove(filepath.Join(dir, "LOCK")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "LOCK")); err != nil {
		t.Fatal(err)
	}

	if _, err := OpenExisting(dir); !errors.Is(err, nofollow.ErrSymlink) {
		t.Fatalf("OpenExisting = %v, want nofollow.ErrSymlink", err)
	}
	assertFileUnchanged(t, outside, data)
}

func TestReplaceStoreRefusesSymlinkInLiveStore(t *testing.T) {
	outside, data := outsideFile(t)
	parent := t.TempDir()
	live, staging := filepath.Join(parent, "store"), filepath.Join(parent, "staging")
	buildStore(t, staging, "new")
	if err := os.Mkdir(live, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(live, "LOCK")); err != nil {
		t.Fatal(err)
	}

	err := ReplaceStore(staging, live)
	assertFileUnchanged(t, outside, data)
	if !errors.Is(err, nofollow.ErrSymlink) {
		t.Fatalf("ReplaceStore = %v, want nofollow.ErrSymlink", err)
	}
}

func TestLockFilesBesideStoreRefuseSymlink(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		run    func(dir string) error
	}{
		{swapLockSuffix, func(dir string) error {
			s, err := Open(dir)
			if err == nil {
				s.Close()
			}
			return err
		}},
		{writerLockSuffix, func(dir string) error {
			l, err := LockWriter(dir)
			if err == nil {
				l.Close()
			}
			return err
		}},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			outside, data := outsideFile(t)
			dir := filepath.Join(t.TempDir(), "store")
			if err := os.Symlink(outside, dir+tc.suffix); err != nil {
				t.Fatal(err)
			}

			if err := tc.run(dir); !errors.Is(err, nofollow.ErrSymlink) {
				t.Fatalf("got %v, want nofollow.ErrSymlink", err)
			}
			assertFileUnchanged(t, outside, data)
		})
	}
}

func TestOpenAcceptsStoreWithoutSymlinks(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	buildStore(t, dir, "n")
	s, err := OpenExisting(dir)
	if err != nil {
		t.Fatalf("OpenExisting: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
