package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// fakeWatcher stands in for *watch.Watcher inside a synctest bubble. The
// test sends paths on events instead of writing files, so no goroutine in
// the bubble waits on file-system events and the bubble's fake clock can
// advance through every debounce window. Deadlines inside a bubble are
// fake time, so tests that use fakeWatcher do not scale them with
// testBudget.
type fakeWatcher struct {
	events chan string
	done   chan struct{}
	closed atomic.Bool
}

// newFakeWatcher must be called inside the bubble that runs the Daemon:
// its channels belong to the bubble they are made in.
func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{events: make(chan string), done: make(chan struct{})}
}

func (f *fakeWatcher) Run(ctx context.Context, deb *watch.Debouncer) {
	defer deb.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.done:
			return
		case p := <-f.events:
			deb.Add(p)
		}
	}
}

func (f *fakeWatcher) Close() error {
	if !f.closed.Swap(true) {
		close(f.done)
	}
	return nil
}

// fakeFixture returns a repo root that holds only an empty .codegraph/,
// which is all New needs. Tests that use it replace indexer.Sync through
// syncFn, so no store is built.
func fakeFixture(t *testing.T) (root, codegraphDir string) {
	t.Helper()
	root = t.TempDir()
	codegraphDir = filepath.Join(root, codegraphDirName)
	if err := os.MkdirAll(codegraphDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", codegraphDir, err)
	}
	return root, codegraphDir
}

// newFakeDaemon returns a Daemon on root whose Run opens fw.
func newFakeDaemon(t *testing.T, root string, fw *fakeWatcher) *Daemon {
	t.Helper()
	d, err := New(root, indexer.Options{Quiet: true},
		WithProbe(watch.Probe{IsWSL: func() bool { return false }}),
		WithWatcher(func(string) (Watcher, error) { return fw, nil }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

// startFakeRun starts d.Run inside the bubble and returns its result
// channel once Run holds the lock and every goroutine it started is
// waiting.
func startFakeRun(t *testing.T, ctx context.Context, cancel context.CancelFunc, d *Daemon) <-chan error {
	t.Helper()
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)
	synctest.Wait()
	if _, ok, err := readLock(d.codegraphDir); err != nil || !ok {
		t.Fatalf("Run does not hold the lock after it started (ok=%v err=%v)", ok, err)
	}
	return runErr
}

// TestRunUsesInjectedWatcher proves Run drives the watcher that WithWatcher
// opens: one event from it produces exactly one sync, and Run closes it on
// shutdown.
func TestRunUsesInjectedWatcher(t *testing.T) {
	root, _ := fakeFixture(t)
	synctest.Test(t, func(t *testing.T) {
		fw := newFakeWatcher()
		d := newFakeDaemon(t, root, fw)
		var syncs atomic.Int32
		d.syncFn = func(string, string, indexer.Options) (indexer.Stats, error) {
			syncs.Add(1)
			return indexer.Stats{}, nil
		}
		synced := make(chan error, 4)
		d.onSync = func(_ indexer.Stats, err error) { synced <- err }

		ctx, cancel := context.WithCancel(context.Background())
		runErr := startFakeRun(t, ctx, cancel, d)

		fw.events <- "main.go"
		select {
		case err := <-synced:
			if err != nil {
				t.Fatalf("sync: %v", err)
			}
		case <-time.After(time.Minute):
			t.Fatal("no sync after an event from the injected watcher")
		}
		select {
		case err := <-synced:
			t.Fatalf("a second sync ran (err=%v) after one event", err)
		case <-time.After(10 * watch.DebounceDuration()):
		}

		cancel()
		select {
		case err := <-runErr:
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
		case <-time.After(time.Minute):
			t.Fatal("Run did not return after cancel")
		}
		if got := syncs.Load(); got != 1 {
			t.Fatalf("syncFn ran %d times, want 1", got)
		}
		if !fw.closed.Load() {
			t.Fatal("Run returned without closing the injected watcher")
		}
	})
}
