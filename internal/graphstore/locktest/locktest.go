// Package locktest holds the store lock for a test, the way another
// process holds it. It imports Pebble and must not import graphstore, so
// that the tests of package graphstore can use it.
package locktest

import (
	"errors"
	"io"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
)

// HoldStoreLock takes the Pebble directory lock of the store at dir and
// fails the test if the lock is not free. The directory must exist.
//
// Pebble keeps its locks of one process in a registry keyed by the cleaned
// path it was given, and its file lock does not exclude a second lock from
// the same process. HoldStoreLock therefore locks three names of dir, where
// they differ: the cleaned path, the absolute path, and the absolute path
// with symbolic links resolved. An open through any other spelling, for
// example through a second symbolic link, does not meet the lock.
//
// The returned function releases the lock. It is safe to call more than
// once. The test also releases the lock when it ends.
func HoldStoreLock(t testing.TB, dir string) (release func()) {
	t.Helper()

	names := []string{filepath.Clean(dir)}
	if abs, err := filepath.Abs(dir); err == nil {
		names = append(names, abs)
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			names = append(names, resolved)
		}
	}
	slices.Sort(names)
	names = slices.Compact(names)

	locks := make([]io.Closer, 0, len(names))
	releaseAll := func() error {
		var errs []error
		for _, l := range locks {
			errs = append(errs, l.Close())
		}
		return errors.Join(errs...)
	}
	for _, name := range names {
		l, err := pebble.LockDirectory(name, vfs.Default)
		if err != nil {
			_ = releaseAll()
			t.Fatalf("locktest: the store lock of %s is not free: %v", name, err)
		}
		locks = append(locks, l)
	}

	release = sync.OnceFunc(func() {
		if err := releaseAll(); err != nil {
			t.Errorf("locktest: releasing the store lock of %s: %v", dir, err)
		}
	})
	t.Cleanup(release)
	return release
}
