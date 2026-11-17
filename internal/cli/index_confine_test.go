package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
)

func symlinkT(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

func wantIndexOutsideRoot(t *testing.T, err error) {
	t.Helper()
	var outside *query.IndexOutsideRootError
	if !errors.As(err, &outside) {
		t.Fatalf("err = %v, want an *query.IndexOutsideRootError", err)
	}
}

func TestInitRefusesIndexLinkedOutside(t *testing.T) {
	t.Run("to a missing target", func(t *testing.T) {
		dir := copyFixture(t)
		target := filepath.Join(t.TempDir(), "victim")
		symlinkT(t, target, filepath.Join(dir, codegraphDirName))

		_, _, err := execCmd("init", dir)
		wantIndexOutsideRoot(t, err)
		if _, statErr := os.Lstat(target); !os.IsNotExist(statErr) {
			t.Fatalf("init created %s through the link (stat err %v)", target, statErr)
		}
	})

	t.Run("to an existing directory", func(t *testing.T) {
		dir := copyFixture(t)
		outside := t.TempDir()
		symlinkT(t, outside, filepath.Join(dir, codegraphDirName))

		_, _, err := execCmd("init", dir)
		wantIndexOutsideRoot(t, err)
		entries, readErr := os.ReadDir(outside)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("outside directory holds %d entries (err %v), want none", len(entries), readErr)
		}
	})
}

func TestIndexCommandsRefuseStoreLinkedOutside(t *testing.T) {
	dir := copyFixture(t)
	if err := os.Mkdir(filepath.Join(dir, codegraphDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	symlinkT(t, outside, filepath.Join(dir, codegraphDirName, storeDirName))

	for _, args := range [][]string{{"sync", dir}, {"index", "--force", dir}, {"status", "-p", dir}} {
		_, _, err := execCmd(args...)
		wantIndexOutsideRoot(t, err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside directory holds %d entries (err %v), want none", len(entries), err)
	}
}

func TestIndexCommandsAcceptIndexLinkedInside(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.Rename(filepath.Join(dir, codegraphDirName), filepath.Join(dir, "cg-real")); err != nil {
		t.Fatal(err)
	}
	symlinkT(t, "cg-real", filepath.Join(dir, codegraphDirName))

	for _, args := range [][]string{{"sync", dir}, {"index", "--force", dir}, {"status", "-p", dir}} {
		if _, _, err := execCmd(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}
