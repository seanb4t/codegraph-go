package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

func TestNewRefusesIndexLinkedOutside(t *testing.T) {
	for _, name := range []string{codegraphDirName, filepath.Join(codegraphDirName, storeDirName)} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), filepath.Join(root, name)); err != nil {
				t.Fatal(err)
			}

			_, err := New(root, indexer.Options{Quiet: true})
			if !errors.Is(err, indexdir.ErrOutsideRoot) {
				t.Fatalf("New = %v, want indexdir.ErrOutsideRoot", err)
			}
		})
	}
}

func TestNewUsesResolvedIndexPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".cg-real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".cg-real", filepath.Join(root, codegraphDirName)); err != nil {
		t.Fatal(err)
	}

	d, err := New(root, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(real, ".cg-real"); d.start.Dir != want {
		t.Fatalf("start.Dir = %q, want %q", d.start.Dir, want)
	}
	if want := filepath.Join(real, ".cg-real", storeDirName); d.start.Store != want {
		t.Fatalf("start.Store = %q, want %q", d.start.Store, want)
	}
}

func TestIndexChangedErrorText(t *testing.T) {
	for _, tc := range []struct {
		name string
		was  indexdir.Paths
		now  indexdir.Paths
		want []string
	}{
		{
			name: "dir",
			was:  indexdir.Paths{Dir: "/r/.codegraph", Store: "/r/.codegraph/store"},
			now:  indexdir.Paths{Dir: "/r/.cg-real", Store: "/r/.cg-real/store"},
			want: []string{`"/r/.codegraph"`, `"/r/.cg-real"`, `"/r/.codegraph/store"`, `"/r/.cg-real/store"`},
		},
		{
			name: "store only",
			was:  indexdir.Paths{Dir: "/r/.codegraph", Store: "/r/.codegraph/store"},
			now:  indexdir.Paths{Dir: "/r/.codegraph", Store: "/r/.store-real"},
			want: []string{`"/r/.codegraph/store"`, `"/r/.store-real"`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := error(&IndexChangedError{Was: tc.was, Now: tc.now})
			if !errors.Is(err, ErrIndexChanged) {
				t.Fatalf("errors.Is(%v, ErrIndexChanged) = false", err)
			}
			for _, part := range append(tc.want, "restart the watcher") {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("message %q does not contain %s", err.Error(), part)
				}
			}
		})
	}
}

// indexChange changes the index of root after New resolved it. Each
// change models a pull that replaces .codegraph or its store. apply
// returns the directory the changed link now leads to, or "" for none.
type indexChange struct {
	name  string
	apply func(t *testing.T, root string) string
}

func replaceWithLink(t *testing.T, path, target string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func moveAndLink(t *testing.T, path, dest, target string) {
	t.Helper()
	if err := os.Rename(path, dest); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

var (
	indexLinkedOutside = indexChange{"index linked outside", func(t *testing.T, root string) string {
		outside := t.TempDir()
		replaceWithLink(t, filepath.Join(root, codegraphDirName), outside)
		return outside
	}}
	storeLinkedOutside = indexChange{"store linked outside", func(t *testing.T, root string) string {
		outside := t.TempDir()
		replaceWithLink(t, filepath.Join(root, codegraphDirName, storeDirName), outside)
		return outside
	}}
	indexLinkedToItself = indexChange{"index linked to itself", func(t *testing.T, root string) string {
		replaceWithLink(t, filepath.Join(root, codegraphDirName), codegraphDirName)
		return ""
	}}
	indexMovedInside = indexChange{"index moved inside", func(t *testing.T, root string) string {
		dest := filepath.Join(root, ".cg-real")
		moveAndLink(t, filepath.Join(root, codegraphDirName), dest, ".cg-real")
		return dest
	}}
	storeMovedInside = indexChange{"store moved inside", func(t *testing.T, root string) string {
		dest := filepath.Join(root, ".store-real")
		moveAndLink(t, filepath.Join(root, codegraphDirName, storeDirName), dest, filepath.Join("..", ".store-real"))
		return dest
	}}
	indexUnchanged = indexChange{"unchanged", func(*testing.T, string) string { return "" }}
)

func mkIndexDirs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, codegraphDirName, storeDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckIndexReportsEveryChange(t *testing.T) {
	for _, tc := range []struct {
		change indexChange
		check  func(t *testing.T, err error)
	}{
		{indexLinkedOutside, wantOutside},
		{storeLinkedOutside, wantOutside},
		{indexMovedInside, func(t *testing.T, err error) {
			changed := wantChanged(t, err)
			if changed.Was.Dir == changed.Now.Dir || changed.Was.Store == changed.Now.Store {
				t.Fatalf("Dir and Store did not both change: %+v", changed)
			}
		}},
		{storeMovedInside, func(t *testing.T, err error) {
			changed := wantChanged(t, err)
			if changed.Was.Dir != changed.Now.Dir || changed.Was.Store == changed.Now.Store {
				t.Fatalf("want an equal Dir and a different Store: %+v", changed)
			}
		}},
		{indexLinkedToItself, func(t *testing.T, err error) {
			if err == nil || errors.Is(err, indexdir.ErrOutsideRoot) || errors.Is(err, ErrIndexChanged) {
				t.Fatalf("checkIndex = %v, want an error that matches neither sentinel", err)
			}
		}},
		{indexUnchanged, func(t *testing.T, err error) {
			if err != nil {
				t.Fatalf("checkIndex = %v, want nil", err)
			}
		}},
	} {
		t.Run(tc.change.name, func(t *testing.T) {
			root := mkIndexDirs(t)
			d, err := New(root, indexer.Options{Quiet: true})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			tc.change.apply(t, root)
			tc.check(t, d.checkIndex())
		})
	}
}

// TestRunSyncsWhenIndexUnchanged is the positive control for the index
// check in Run: with the index unchanged, an edit still reaches the store
// through the real indexer.Sync, and Run still returns nil on a cancel.
func TestRunSyncsWhenIndexUnchanged(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	withRegistryDir(t)
	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	synced := make(chan error, 4)
	d.onSync = func(_ indexer.Stats, err error) { synced <- err }
	opened := watcherOpened(d)

	ctx, cancel := context.WithCancel(context.Background())
	runErr := startRun(t, ctx, cancel, d)
	waitForLock(t, codegraphDir)
	waitWatcherOpened(t, opened)
	if err := waitCatchUpSync(t, synced); err != nil {
		t.Fatalf("catch-up sync: %v", err)
	}

	writeFixtureFile(t, root, "added.go", "package main\n\nfunc added() {}\n")
	if err := receive(t, synced, "the sync for added.go"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	stopRun(t, cancel, runErr)

	engine, closer, err := query.OpenAt(root)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	defer closer.Close()
	found, err := engine.Search("added", "", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, loc := range found {
		if loc.Name == "added" {
			return
		}
	}
	t.Fatalf("Search(added) = %+v, want the function added", found)
}

func wantOutside(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, indexdir.ErrOutsideRoot) {
		t.Fatalf("err = %v, want indexdir.ErrOutsideRoot", err)
	}
}

func wantChanged(t *testing.T, err error) *IndexChangedError {
	t.Helper()
	changed, ok := errors.AsType[*IndexChangedError](err)
	if !ok {
		t.Fatalf("err = %v, want an *IndexChangedError", err)
	}
	return changed
}
