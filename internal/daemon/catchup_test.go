package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// runCatchUpSync runs a Daemon on root until its catch-up sync returns, then
// stops it. The watcher is a fakeWatcher that sends no event, so the
// catch-up sync is the only sync that can run, and the helper fails the
// test when the count is not one. beforeOpen runs inside openWatcher, before
// the watcher exists. The sync is the real indexer.Sync, so root needs a
// store, and the caller must not be in a synctest bubble.
func runCatchUpSync(t *testing.T, root string, beforeOpen func() error) indexer.Stats {
	t.Helper()
	d, err := New(root, indexer.Options{Quiet: true},
		WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	fw := newFakeWatcher()
	d.openWatcher = func(string) (watcher, error) {
		if err := beforeOpen(); err != nil {
			return nil, err
		}
		return fw, nil
	}
	var syncs atomic.Int32
	synced := make(chan indexer.Stats, 4)
	d.onSync = func(stats indexer.Stats, err error) {
		syncs.Add(1)
		if err != nil {
			t.Errorf("catch-up sync failed: %v", err)
		}
		synced <- stats
	}

	ctx, cancel := context.WithCancel(t.Context())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	stats := waitCatchUpSync(t, synced)

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}
	if got := syncs.Load(); got != 1 {
		t.Fatalf("%d syncs ran with no event, want 1: the catch-up sync", got)
	}
	return stats
}

// TestRunSyncsEditMadeBeforeWatchOpen proves that Run syncs an edit that no
// watcher event reports, because the edit was made before the watcher
// existed.
func TestRunSyncsEditMadeBeforeWatchOpen(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	t.Run("new file", func(t *testing.T) {
		root, _, storeDir := initFixture(t)

		stats := runCatchUpSync(t, root, func() error {
			return os.WriteFile(filepath.Join(root, "early.go"), []byte("package main\n\nfunc Early() {}\n"), 0o644)
		})

		if stats.FilesReparsed != 1 {
			t.Fatalf("catch-up sync reparsed %d files, want 1: early.go", stats.FilesReparsed)
		}
		if names := nodeNames(t, storeDir); !names["Early"] {
			t.Fatalf("graph = %v, want it to contain Early", names)
		}
	})

	// The new content has the length of the indexed content, so only the
	// modification time tells the sync that main.go changed.
	t.Run("rewrite of the same length", func(t *testing.T) {
		root, _, storeDir := initFixture(t)
		mainPath := filepath.Join(root, "main.go")
		before, err := os.Stat(mainPath)
		if err != nil {
			t.Fatalf("Stat(main.go): %v", err)
		}
		const rewritten = "package main\n\nfunc mane() {}\n"
		if int64(len(rewritten)) != before.Size() {
			t.Fatalf("the rewrite has %d bytes and the indexed main.go has %d, want the same length", len(rewritten), before.Size())
		}

		stats := runCatchUpSync(t, root, func() error {
			if err := os.WriteFile(mainPath, []byte(rewritten), 0o644); err != nil {
				return err
			}
			mtime := before.ModTime().Add(2 * time.Second)
			return os.Chtimes(mainPath, mtime, mtime)
		})

		if stats.FilesReparsed != 1 {
			t.Fatalf("catch-up sync reparsed %d files, want 1: main.go", stats.FilesReparsed)
		}
		if names := nodeNames(t, storeDir); !names["mane"] {
			t.Fatalf("graph = %v, want it to contain mane", names)
		}
	})
}

// TestRunCatchUpSyncWithNoChange proves that the catch-up sync of an
// unchanged tree reparses nothing and leaves the graph marked fresh.
func TestRunCatchUpSyncWithNoChange(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, _ := initFixture(t)

	stats := runCatchUpSync(t, root, func() error { return nil })

	if stats.FilesReparsed != 0 {
		t.Fatalf("catch-up sync reparsed %d files with no edit, want 0", stats.FilesReparsed)
	}
	if _, err := os.Stat(filepath.Join(codegraphDir, staleSidecarName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(%s) = %v, want it absent after a catch-up sync that committed", staleSidecarName, err)
	}
}

// TestRunCatchUpSyncOnHeldStore proves that the catch-up flush has the lock
// requeue budget of every other flush. syncFn injects ErrStoreLocked, as
// TestDaemonFlushLockRequeueGivesUpPerEpisode does, and the Run is in a
// synctest bubble, so the quiet gap after the last attempt is fake time.
// The store is real but nothing opens it inside the bubble.
func TestRunCatchUpSyncOnHeldStore(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "10")

	// runHeld runs a Daemon whose first lockedSyncs syncs find the store
	// locked. It receives want sync results, checks that no further sync
	// follows, and stops the Daemon. No event is sent.
	runHeld := func(t *testing.T, root string, lockedSyncs, want int) {
		t.Helper()
		synctest.Test(t, func(t *testing.T) {
			fw := newFakeWatcher()
			d := newFakeDaemon(t, root, fw)
			var syncs atomic.Int32
			d.syncFn = func(string, string, indexer.Options) (indexer.Stats, error) {
				if int(syncs.Add(1)) <= lockedSyncs {
					return indexer.Stats{}, fmt.Errorf("%w: injected contention", graphstore.ErrStoreLocked)
				}
				return indexer.Stats{}, nil
			}
			synced := make(chan error, 32)
			d.onSync = func(_ indexer.Stats, err error) { synced <- err }

			ctx, cancel := context.WithCancel(t.Context())
			runErr := startFakeRun(t, ctx, cancel, d)

			for i := 1; i <= want; i++ {
				select {
				case err := <-synced:
					if wantLocked := i <= lockedSyncs; errors.Is(err, graphstore.ErrStoreLocked) != wantLocked {
						t.Fatalf("sync %d of %d: error = %v, want store locked = %v", i, want, err, wantLocked)
					}
				case <-time.After(10 * time.Second):
					t.Fatalf("timed out waiting for sync %d of %d", i, want)
				}
			}
			// A further sync would follow one debounce window after the
			// last one, well inside this gap.
			select {
			case err := <-synced:
				t.Fatalf("sync %d ran (err=%v), want no sync after the first %d", want+1, err, want)
			case <-time.After(time.Second):
			}

			cancel()
			select {
			case err := <-runErr:
				if err != nil {
					t.Fatalf("Run: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Run did not return after cancel")
			}
		})
	}

	t.Run("held through every requeue", func(t *testing.T) {
		root, codegraphDir, _ := initFixture(t)

		// The catch-up flush and each of its requeues find the store locked.
		attempts := maxFlushLockRequeues + 1
		runHeld(t, root, attempts, attempts)

		if _, err := os.Stat(filepath.Join(codegraphDir, staleSidecarName)); err != nil {
			t.Fatalf("Stat(%s) = %v, want it present after the catch-up flush gave up", staleSidecarName, err)
		}
		eng, closer, err := query.OpenAt(root)
		if err != nil {
			t.Fatalf("query.OpenAt: %v", err)
		}
		defer closer.Close()
		status, err := eng.Status(t.Context())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if !status.Stale {
			t.Fatal("status reports a fresh graph after the catch-up flush gave up, want stale")
		}
	})

	t.Run("held for one attempt", func(t *testing.T) {
		root, codegraphDir, _ := initFixture(t)

		runHeld(t, root, 1, 2)

		if _, err := os.Stat(filepath.Join(codegraphDir, staleSidecarName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(%s) = %v, want it absent after the requeued sync committed", staleSidecarName, err)
		}
	})
}
