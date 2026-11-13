package graphstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore/locktest"
	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

// requireLocked fails the test unless Open(dir) returns ErrStoreLocked.
func requireLocked(t *testing.T, dir string) {
	t.Helper()
	s, err := Open(dir)
	if err == nil {
		s.Close()
		t.Fatalf("Open(%s) succeeded while the store lock is held", dir)
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("Open(%s) = %v, want ErrStoreLocked", dir, err)
	}
}

// requireOpens fails the test unless Open(dir) succeeds.
func requireOpens(t *testing.T, dir string) {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%s) with the store lock released: %v", dir, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestHoldStoreLockExcludesOpen(t *testing.T) {
	t.Run("held through a symbolic link", func(t *testing.T) {
		noRetrySleep(t)
		store := filepath.Join(realpathtest.TempDir(t), "store")
		if err := os.Mkdir(store, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(realpathtest.TempDir(t), "link")
		if err := os.Symlink(store, link); err != nil {
			t.Fatal(err)
		}

		release := locktest.HoldStoreLock(t, link)
		requireLocked(t, store)
		requireLocked(t, link)

		release()
		requireOpens(t, store)
		requireOpens(t, link)
		// The cleanup of the test releases again. A second release of a
		// Pebble lock panics, so the helper must make it a no-op.
		release()
	})

	t.Run("held through a relative spelling", func(t *testing.T) {
		noRetrySleep(t)
		parent := realpathtest.TempDir(t)
		store := filepath.Join(parent, "store")
		if err := os.Mkdir(store, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(parent)

		release := locktest.HoldStoreLock(t, "./store/")
		requireLocked(t, "store")
		requireLocked(t, store)

		release()
		requireOpens(t, "store")
		requireOpens(t, store)
	})
}
