package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"go.uber.org/goleak"
)

// TestMain gates the whole internal/daemon test package on goleak: after
// every test in this package has run, VerifyTestMain asserts zero
// goroutines remain that were not present at process start (SYNC-06).
// internal/daemon had no existing TestMain (daemon_test.go/lock_test.go
// define none), so this is the package's first.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// soakIterations is a modest edit->sync cycle count that exercises the
// daemon's full watch->debounce->sync lifecycle repeatedly while keeping the
// soak well inside the 120s CI timeout under -race (RESEARCH "Soak-test
// iteration counts... executor's discretion").
const soakIterations = 15

// TestSoak drives Daemon.Run(ctx) through many edit->sync cycles — the
// daemon repeatedly acquires-writer -> Sync -> releases via the debounced
// flush — then cancels and joins BEFORE returning, proving the full
// watcher+debounce+sync lifecycle (all three goroutine sources: the
// watcher loop, the debounce timer, and the sync worker invoked from it)
// is goroutine-leak-free (SYNC-06) and that the lockfile is released after
// shutdown so a subsequent acquire succeeds.
func TestSoak(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, storeDir := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	synced := make(chan indexer.Stats, soakIterations+1)
	d.onSync = func(stats indexer.Stats, err error) {
		if err != nil {
			t.Errorf("daemon-driven sync failed: %v", err)
			return
		}
		synced <- stats
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	runErr := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr <- d.Run(ctx)
	}()

	waitForLock(t, codegraphDir)

	lastSymbol := ""
	for i := 0; i < soakIterations; i++ {
		lastSymbol = fmt.Sprintf("soak%d", i)
		writeFixtureFile(t, root, fmt.Sprintf("soak%d.go", i), fmt.Sprintf("package main\n\nfunc %s() {}\n", lastSymbol))

		select {
		case <-synced:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for sync #%d of the soak", i)
		}
	}

	if names := nodeNames(t, storeDir); !names[lastSymbol] {
		t.Fatalf("graph missing %q after %d edit->sync cycles — real path not exercised", lastSymbol, soakIterations)
	}

	// Cancel and join BEFORE any goleak assertion runs (TestMain's
	// VerifyTestMain fires after the whole package's tests complete): Run
	// blocks on wg.Wait() internally for its own watcher goroutine (D-07)
	// before returning, so joining here guarantees no daemon-spawned
	// goroutine (watcher loop, debounce timer, sync worker) survives past
	// this test.
	cancel()
	wg.Wait()

	err = <-runErr
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, ok, err := readLock(codegraphDir); err != nil || ok {
		t.Fatalf("lockfile still present after soak shutdown (ok=%v err=%v)", ok, err)
	}

	// A subsequent acquire succeeding proves the lock was genuinely
	// released, not merely absent for an unrelated reason.
	if err := acquire(codegraphDir); err != nil {
		t.Fatalf("acquire after soak shutdown: %v", err)
	}
	if err := release(codegraphDir); err != nil {
		t.Fatalf("release: %v", err)
	}
}
