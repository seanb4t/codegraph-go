package fsatomic

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// recordSyncs wraps syncDir for the life of the test and returns the list
// of directories that get an fsync.
func recordSyncs(t *testing.T) *[]string {
	t.Helper()
	var dirs []string
	orig := syncDir
	syncDir = func(dir string) error {
		dirs = append(dirs, dir)
		return orig(dir)
	}
	t.Cleanup(func() { syncDir = orig })
	return &dirs
}

// twoDirs makes the directories a and b, each with one file named after it.
// They share a parent, or each has its own.
func twoDirs(t *testing.T, sameParent bool) (a, b string) {
	t.Helper()
	root := t.TempDir()
	a, b = filepath.Join(root, "a"), filepath.Join(root, "b")
	if !sameParent {
		a, b = filepath.Join(root, "pa", "a"), filepath.Join(root, "pb", "b")
	}
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(d, filepath.Base(d)+".txt"), nil, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return a, b
}

func holds(t *testing.T, dir, file string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, file))
	return err == nil
}

// requireParentSyncs is the positive assertion of both tables: the parents
// of the two paths got an fsync, each one exactly once.
func requireParentSyncs(t *testing.T, got []string, a, b string) {
	t.Helper()
	want := []string{filepath.Dir(a)}
	if filepath.Dir(b) != filepath.Dir(a) {
		want = append(want, filepath.Dir(b))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("fsync ran on %v, want %v", got, want)
	}
}

var parentCases = []struct {
	name       string
	sameParent bool
}{
	{"one parent", true},
	{"two parents", false},
}

func TestExchangeDirs(t *testing.T) {
	for _, tc := range parentCases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := twoDirs(t, tc.sameParent)
			synced := recordSyncs(t)

			if err := ExchangeDirs(a, b); err != nil {
				t.Fatalf("ExchangeDirs: %v", err)
			}
			if !holds(t, a, "b.txt") || holds(t, a, "a.txt") {
				t.Errorf("%s does not hold exactly the files of the other directory", a)
			}
			if !holds(t, b, "a.txt") || holds(t, b, "b.txt") {
				t.Errorf("%s does not hold exactly the files of the other directory", b)
			}
			requireParentSyncs(t, *synced, a, b)
		})
	}
}

func TestRenameDir(t *testing.T) {
	for _, tc := range parentCases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := twoDirs(t, tc.sameParent)
			if err := os.RemoveAll(b); err != nil {
				t.Fatalf("remove: %v", err)
			}
			synced := recordSyncs(t)

			if err := RenameDir(a, b); err != nil {
				t.Fatalf("RenameDir: %v", err)
			}
			if !holds(t, b, "a.txt") {
				t.Errorf("%s does not hold the files of the old path", b)
			}
			if _, err := os.Lstat(a); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Lstat(%s) = %v, want not-exist", a, err)
			}
			requireParentSyncs(t, *synced, a, b)
		})
	}
}

func TestExchangeDirsUnsupported(t *testing.T) {
	for _, tc := range []struct {
		name  string
		errno syscall.Errno
	}{
		{"EINVAL, as Linux returns it", syscall.EINVAL},
		{"ENOTSUP, as macOS returns it", syscall.ENOTSUP},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := twoDirs(t, true)
			orig := exchange
			exchange = func(string, string) error { return tc.errno }
			t.Cleanup(func() { exchange = orig })

			err := ExchangeDirs(a, b)
			if !errors.Is(err, ErrExchangeUnsupported) || !errors.Is(err, tc.errno) {
				t.Fatalf("error %v, want one that wraps ErrExchangeUnsupported and %v", err, tc.errno)
			}
			if !strings.Contains(err.Error(), "does not support an atomic exchange") {
				t.Errorf("error %q does not name the cause", err)
			}
			if !holds(t, a, "a.txt") || !holds(t, b, "b.txt") {
				t.Errorf("a directory lost its files after a failed exchange")
			}
		})
	}
}
