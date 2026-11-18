package graphstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/flock"
	"github.com/seanb4t/codegraph-go/internal/fsatomic"
	"github.com/seanb4t/codegraph-go/internal/graphstore/locktest"
	"github.com/seanb4t/codegraph-go/internal/nofollow"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// buildStore makes a store at dir that holds one node named name, and
// returns its export.
func buildStore(t *testing.T, dir, name string) []byte {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%s): %v", dir, err)
	}
	w, err := s.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutNode(&schema.Node{Id: name, Name: name}); err != nil {
		t.Fatalf("PutNode: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	export := exportOf(t, s)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return export
}

// exportAt opens the store at dir, which must exist, and returns its export.
func exportAt(t *testing.T, dir string) []byte {
	t.Helper()
	s, err := OpenExisting(dir)
	if err != nil {
		t.Fatalf("OpenExisting(%s): %v", dir, err)
	}
	defer s.Close()
	return exportOf(t, s)
}

// storePaths returns a staging path in a scratch directory and a live
// path, both under one root, as a full index lays them out.
func storePaths(t *testing.T) (staging, live string) {
	t.Helper()
	root := t.TempDir()
	scratch := filepath.Join(root, "store.scratch-x")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return filepath.Join(scratch, "store"), filepath.Join(root, "store")
}

func TestOpenWaitsForSwapLock(t *testing.T) {
	noRetrySleep(t)
	dir := filepath.Join(t.TempDir(), "store")
	lock, err := flock.Open(dir+swapLockSuffix, flock.Exclusive)
	if err != nil {
		t.Fatalf("flock.Open: %v", err)
	}

	if _, err := Open(dir); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("Open under an exclusive swap lock: %v, want ErrStoreLocked", err)
	}
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Lstat(%s) = %v; want the store still missing, because open takes the lock before it creates anything", dir, err)
	}

	// Positive assertion: the same open works once the lock is free.
	lock.Close()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open after the swap lock is free: %v", err)
	}
	s.Close()
}

func TestReplaceStoreExchanges(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, live string) (oldExport []byte)
	}{
		{"a real store", func(t *testing.T, live string) []byte { return buildStore(t, live, "old") }},
		{"an empty directory", func(t *testing.T, live string) []byte {
			if err := os.Mkdir(live, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			return nil
		}},
		{"a directory that holds only a LOCK file", func(t *testing.T, live string) []byte {
			if err := os.Mkdir(live, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(live, "LOCK"), nil, 0o644); err != nil {
				t.Fatalf("write LOCK: %v", err)
			}
			return nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			staging, live := storePaths(t)
			oldExport := tc.setup(t, live)
			newExport := buildStore(t, staging, "new")

			if err := ReplaceStore(staging, live); err != nil {
				t.Fatalf("ReplaceStore: %v", err)
			}
			if !bytes.Equal(exportAt(t, live), newExport) {
				t.Errorf("the live path does not hold the export of the staging store")
			}
			if oldExport != nil && !bytes.Equal(exportAt(t, staging), oldExport) {
				t.Errorf("the staging path does not hold the export of the old live store")
			}
		})
	}
}

func TestReplaceStoreRenamesFirstStore(t *testing.T) {
	staging, live := storePaths(t)
	newExport := buildStore(t, staging, "new")

	if err := ReplaceStore(staging, live); err != nil {
		t.Fatalf("ReplaceStore: %v", err)
	}
	if !bytes.Equal(exportAt(t, live), newExport) {
		t.Errorf("the live path does not hold the staging store")
	}
	if _, err := os.Lstat(staging); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Lstat(%s) = %v, want the staging path gone", staging, err)
	}
}

// requireUsable is the positive assertion of the lock tests: after a
// failed ReplaceStore, the swap lock and the Pebble lock of the live store
// are both free, so no lock of the failed call is left in this process.
func requireUsable(t *testing.T, live string) {
	t.Helper()
	resolved, err := resolveStorePath(live)
	if err != nil {
		t.Fatalf("resolveStorePath(%s): %v", live, err)
	}
	swap, err := flock.Open(resolved+swapLockSuffix, flock.Exclusive)
	if err != nil {
		t.Fatalf("the swap lock of %s is not free after the failed ReplaceStore: %v", live, err)
	}
	if err := swap.Close(); err != nil {
		t.Fatalf("releasing the swap lock of %s: %v", live, err)
	}
	locktest.HoldStoreLock(t, live)()
}

func TestReplaceStoreWaitsForOpenHandle(t *testing.T) {
	for _, tc := range []struct {
		name string
		hold func(t *testing.T, live string) (release func())
	}{
		{"a handle from graphstore.Open", func(t *testing.T, live string) func() {
			s, err := Open(live)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			return func() { s.Close() }
		}},
		{"the Pebble lock with no swap lock", func(t *testing.T, live string) func() {
			return locktest.HoldStoreLock(t, live)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			noRetrySleep(t)
			staging, live := storePaths(t)
			oldExport := buildStore(t, live, "old")
			newExport := buildStore(t, staging, "new")
			release := tc.hold(t, live)

			if err := ReplaceStore(staging, live); !errors.Is(err, ErrStoreLocked) {
				release()
				t.Fatalf("ReplaceStore with the live store open: %v, want ErrStoreLocked", err)
			}
			release()
			if !bytes.Equal(exportAt(t, live), oldExport) || !bytes.Equal(exportAt(t, staging), newExport) {
				t.Fatalf("a store changed in a ReplaceStore that failed")
			}

			requireUsable(t, live)
			if err := ReplaceStore(staging, live); err != nil {
				t.Fatalf("a second ReplaceStore, with the handle released: %v", err)
			}
			if !bytes.Equal(exportAt(t, live), newExport) {
				t.Errorf("the live path does not hold the new store after the second ReplaceStore")
			}
		})
	}
}

// TestReplaceStoreErrorAfterTheSwap checks that an error which comes after
// the new store is in place wraps ErrAfterReplace, and that an error before
// it does not.
func TestReplaceStoreErrorAfterTheSwap(t *testing.T) {
	errSync := errors.New("injected fsync failure")
	setExchange := func(t *testing.T, f func(a, b string) error) {
		orig := exchangeDirs
		exchangeDirs = f
		t.Cleanup(func() { exchangeDirs = orig })
	}
	t.Run("the fsync after the exchange fails", func(t *testing.T) {
		staging, live := storePaths(t)
		buildStore(t, live, "old")
		newExport := buildStore(t, staging, "new")
		setExchange(t, func(a, b string) error {
			if err := fsatomic.ExchangeDirs(a, b); err != nil {
				return err
			}
			return fmt.Errorf("%w: %w", fsatomic.ErrNotDurable, errSync)
		})
		err := ReplaceStore(staging, live)
		if !errors.Is(err, ErrAfterReplace) || !errors.Is(err, errSync) {
			t.Fatalf("ReplaceStore error = %v, want one that wraps ErrAfterReplace and the fsync error", err)
		}
		if !bytes.Equal(exportAt(t, live), newExport) {
			t.Errorf("the live path does not hold the new store")
		}
	})
	t.Run("the exchange itself fails", func(t *testing.T) {
		staging, live := storePaths(t)
		oldExport := buildStore(t, live, "old")
		buildStore(t, staging, "new")
		setExchange(t, func(a, b string) error { return errSync })
		err := ReplaceStore(staging, live)
		if !errors.Is(err, errSync) || errors.Is(err, ErrAfterReplace) {
			t.Fatalf("ReplaceStore error = %v, want the exchange error without ErrAfterReplace", err)
		}
		if !bytes.Equal(exportAt(t, live), oldExport) {
			t.Errorf("the live store changed in a ReplaceStore that failed")
		}
	})
}

func TestReplaceStoreReleasesLocksOnFailure(t *testing.T) {
	staging, live := storePaths(t)
	oldExport := buildStore(t, live, "old")
	newExport := buildStore(t, staging, "new")

	orig := exchangeDirs
	exchangeDirs = func(a, b string) error {
		return errors.Join(fsatomic.ErrExchangeUnsupported, syscall.EINVAL)
	}
	err := ReplaceStore(staging, live)
	exchangeDirs = orig

	if !errors.Is(err, fsatomic.ErrExchangeUnsupported) || !strings.Contains(err.Error(), "does not support an atomic exchange") {
		t.Fatalf("ReplaceStore error = %v, want one that names the missing exchange", err)
	}
	if !bytes.Equal(exportAt(t, live), oldExport) {
		t.Fatalf("the live store changed in a ReplaceStore that failed")
	}

	// Positive assertion: the failed call left no lock in this process.
	requireUsable(t, live)
	if err := ReplaceStore(staging, live); err != nil {
		t.Fatalf("a second ReplaceStore, with the exchange restored: %v", err)
	}
	if !bytes.Equal(exportAt(t, live), newExport) {
		t.Errorf("the live path does not hold the new store after the second ReplaceStore")
	}
}

func TestReplaceStoreWaitsForCheckpoint(t *testing.T) {
	noRetrySleep(t)
	staging, live := storePaths(t)
	buildStore(t, live, "old")
	newExport := buildStore(t, staging, "new")

	// The hook holds the source handle of a Checkpoint open while the
	// test calls ReplaceStore.
	entered, release := make(chan struct{}), make(chan struct{})
	checkpointHook = func() {
		close(entered)
		<-release
	}
	defer func() { checkpointHook = nil }()
	done := make(chan error, 1)
	go func() { done <- Checkpoint(live, filepath.Join(t.TempDir(), "copy")) }()
	<-entered

	err := ReplaceStore(staging, live)
	close(release)
	if cpErr := <-done; cpErr != nil {
		t.Fatalf("Checkpoint: %v", cpErr)
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("ReplaceStore during a Checkpoint of the live store: %v, want ErrStoreLocked", err)
	}

	// Positive assertion: with the Checkpoint done, the exchange works.
	checkpointHook = nil
	if err := ReplaceStore(staging, live); err != nil {
		t.Fatalf("ReplaceStore after the Checkpoint: %v", err)
	}
	if !bytes.Equal(exportAt(t, live), newExport) {
		t.Errorf("the live path does not hold the new store")
	}
}

func TestLockWriterExcludesSecondWriter(t *testing.T) {
	noRetrySleep(t)
	dir := filepath.Join(t.TempDir(), "nested", "store")

	first, err := LockWriter(dir)
	if err != nil {
		t.Fatalf("LockWriter: %v", err)
	}
	if _, err := LockWriter(dir); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("a second LockWriter: %v, want ErrStoreLocked", err)
	}
	// The writer lock keeps writers apart, not readers: an open works.
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open under the writer lock: %v", err)
	}
	s.Close()

	// Positive assertion: the lock is free once its holder closes.
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	second, err := LockWriter(dir)
	if err != nil {
		t.Fatalf("LockWriter after the first holder closed: %v", err)
	}
	second.Close()
}

// linkedStore returns the path of a store directory and a symlink to it,
// two names for one store.
func linkedStore(t *testing.T) (real, link string) {
	t.Helper()
	root := t.TempDir()
	real, link = filepath.Join(root, "store"), filepath.Join(root, "link")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return real, link
}

func TestReplaceStoreWaitsForHandleOpenedThroughLink(t *testing.T) {
	noRetrySleep(t)
	real, link := linkedStore(t)
	staging := filepath.Join(t.TempDir(), "store")
	oldExport := buildStore(t, real, "old")
	newExport := buildStore(t, staging, "new")

	s, err := Open(link)
	if err != nil {
		t.Fatalf("Open through the link: %v", err)
	}
	if err := ReplaceStore(staging, real); !errors.Is(err, ErrStoreLocked) {
		s.Close()
		t.Fatalf("ReplaceStore by the real path with a handle open by the link: %v, want ErrStoreLocked", err)
	}
	s.Close()
	if !bytes.Equal(exportAt(t, real), oldExport) {
		t.Fatalf("the live store changed in a ReplaceStore that failed")
	}

	// Positive assertion: the replace goes through once the handle is closed.
	if err := ReplaceStore(staging, real); err != nil {
		t.Fatalf("ReplaceStore after the handle closed: %v", err)
	}
	if !bytes.Equal(exportAt(t, real), newExport) {
		t.Errorf("the live store does not hold the staging store after the replace")
	}
}

func TestReplaceStoreRefusesALinkAtLive(t *testing.T) {
	for _, tc := range []struct {
		name     string
		oldStore bool
	}{
		{"a link to a store", true},
		{"a link to a missing store", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			real, link := linkedStore(t)
			var oldExport []byte
			if tc.oldStore {
				oldExport = buildStore(t, real, "old")
			} else if err := os.Remove(real); err != nil {
				t.Fatalf("remove: %v", err)
			}
			staging := filepath.Join(t.TempDir(), "store")
			newExport := buildStore(t, staging, "new")

			err := ReplaceStore(staging, link)
			if !errors.Is(err, nofollow.ErrSymlink) {
				t.Fatalf("ReplaceStore through a link = %v, want nofollow.ErrSymlink", err)
			}
			if target, err := os.Readlink(link); err != nil || target != real {
				t.Fatalf("the link reads %q, %v after the refusal, want %q", target, err, real)
			}
			if tc.oldStore && !bytes.Equal(exportAt(t, real), oldExport) {
				t.Errorf("the store behind the link changed")
			}
			if _, err := os.Stat(real); !tc.oldStore && !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the missing target of the link exists after the refusal: %v", err)
			}
			if !bytes.Equal(exportAt(t, staging), newExport) {
				t.Errorf("staging does not hold the new store after the refusal")
			}
		})
	}
}

func TestOpenWaitsForSwapLockTakenThroughOtherName(t *testing.T) {
	noRetrySleep(t)
	real, link := linkedStore(t)
	lock, err := flock.Open(real+swapLockSuffix, flock.Exclusive)
	if err != nil {
		t.Fatalf("flock.Open: %v", err)
	}
	defer lock.Close()

	s, err := Open(link)
	if err == nil {
		t.Cleanup(func() { s.Close() })
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("Open by the link under a swap lock taken by the real path: %v, want ErrStoreLocked", err)
	}
}

func TestLockWriterExcludesSecondWriterThroughOtherName(t *testing.T) {
	noRetrySleep(t)
	real, link := linkedStore(t)

	first, err := LockWriter(link)
	if err != nil {
		t.Fatalf("LockWriter: %v", err)
	}
	defer first.Close()
	second, err := LockWriter(real)
	if err == nil {
		t.Cleanup(func() { second.Close() })
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("LockWriter by the real path with the lock held by the link: %v, want ErrStoreLocked", err)
	}

	// Positive assertion: the lock is free once its holder closes.
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	third, err := LockWriter(real)
	if err != nil {
		t.Fatalf("LockWriter by the real path after the first holder closed: %v", err)
	}
	third.Close()
}

func TestLockWriterFailsOnSymlinkLoopWithoutRetry(t *testing.T) {
	sleeps := countRetrySleeps(t)
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	l, err := LockWriter(filepath.Join(a, "store"))
	if err == nil {
		l.Close()
		t.Fatalf("LockWriter through a symlink loop succeeded, want an error")
	}
	if errors.Is(err, ErrStoreLocked) {
		t.Errorf("LockWriter through a symlink loop: %v, want an error that is not ErrStoreLocked", err)
	}
	if n := sleeps.Load(); n != 0 {
		t.Errorf("LockWriter retried %d times on a resolution error, want one attempt", n)
	}
}
