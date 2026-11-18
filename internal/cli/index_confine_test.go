package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexdir"
)

func symlinkT(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

func wantIndexOutsideRoot(t *testing.T, err error) {
	t.Helper()
	var outside *indexdir.OutsideRootError
	if !errors.As(err, &outside) {
		t.Fatalf("err = %v, want an *indexdir.OutsideRootError", err)
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

func TestIndexCommandsRefuseIndexLinkedOutside(t *testing.T) {
	for _, tc := range []struct {
		name string
		link string
	}{
		{"store", filepath.Join(codegraphDirName, storeDirName)},
		{"index", codegraphDirName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyFixture(t)
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, tc.link)), 0o755); err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			symlinkT(t, outside, filepath.Join(dir, tc.link))

			for _, args := range [][]string{{"sync", dir}, {"index", "--force", dir}, {"status", "-p", dir}} {
				_, _, err := execCmd(args...)
				wantIndexOutsideRoot(t, err)
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatalf("outside directory holds %d entries (err %v), want none", len(entries), err)
			}
		})
	}
}

func TestIndexCommandsAcceptIndexLinkedInside(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.Rename(filepath.Join(dir, codegraphDirName), filepath.Join(dir, ".cg-real")); err != nil {
		t.Fatal(err)
	}
	symlinkT(t, ".cg-real", filepath.Join(dir, codegraphDirName))

	for _, args := range [][]string{{"sync", dir}, {"index", "--force", dir}, {"status", "-p", dir}} {
		if _, _, err := execCmd(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

// linkStoreInside moves the store of an initialized dir to dir/store-real
// and links .codegraph/store to it. It returns the path of store-real.
func linkStoreInside(t *testing.T, dir string) string {
	t.Helper()
	real := filepath.Join(dir, "store-real")
	if err := os.Rename(filepath.Join(dir, codegraphDirName, storeDirName), real); err != nil {
		t.Fatal(err)
	}
	symlinkT(t, filepath.Join("..", "store-real"), filepath.Join(dir, codegraphDirName, storeDirName))
	return real
}

func wantStoreLinkKept(t *testing.T, dir, real string) {
	t.Helper()
	target, err := os.Readlink(filepath.Join(dir, codegraphDirName, storeDirName))
	if err != nil || target != filepath.Join("..", "store-real") {
		t.Fatalf("store link = %q (err %v), want ../store-real", target, err)
	}
	if err := graphstore.CheckLiveStore(real); err != nil {
		t.Fatalf("store-real does not hold the store: %v", err)
	}
}

func TestIndexCommandsAcceptAStoreLinkedInside(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	real := linkStoreInside(t, dir)

	for _, args := range [][]string{{"index", "--force", dir}, {"sync", dir}} {
		if _, _, err := execCmd(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	wantStoreLinkKept(t, dir, real)

	out, _, err := execCmd("status", "-p", dir, "--json")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var status struct {
		FileCount int64 `json:"fileCount"`
		Stale     bool  `json:"stale"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("status output %q: %v", out, err)
	}
	if status.FileCount == 0 || status.Stale {
		t.Fatalf("status = %+v, want indexed files and a graph that is not stale", status)
	}
}

func TestServeReconcileBuildsTheStoreBehindALinkInside(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	real := linkStoreInside(t, dir)
	if err := os.RemoveAll(real); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	if err := serveReconcile(dir, &stderr); err != nil {
		t.Fatalf("serveReconcile: %v (stderr %q)", err, stderr.String())
	}
	wantStoreLinkKept(t, dir, real)
}

func TestInitKeepsAlreadyExistsForIndexThatIsARegularFile(t *testing.T) {
	dir := copyFixture(t)
	if err := os.WriteFile(filepath.Join(dir, codegraphDirName), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := execCmd("init", dir)
	if !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("init = %v, want ErrAlreadyInitialized", err)
	}
}

func TestDaemonUnlockRefusesIndexLinkedOutside(t *testing.T) {
	dir := copyFixture(t)
	outside := t.TempDir()
	lock := filepath.Join(outside, "daemon.lock")
	if err := os.WriteFile(lock, []byte("1 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkT(t, outside, filepath.Join(dir, codegraphDirName))

	_, _, err := execCmd("daemon", "unlock", dir)
	wantIndexOutsideRoot(t, err)
	if _, statErr := os.Stat(lock); statErr != nil {
		t.Fatalf("the lock outside the repository is gone: %v", statErr)
	}
}
