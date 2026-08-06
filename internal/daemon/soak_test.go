package daemon

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
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
		close(runErr)
	}()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)

	lastSymbol := ""
	for i := 0; i < soakIterations; i++ {
		lastSymbol = fmt.Sprintf("soak%d", i)
		writeFixtureFile(t, root, fmt.Sprintf("soak%d.go", i), fmt.Sprintf("package main\n\nfunc %s() {}\n", lastSymbol))

		select {
		case <-synced:
		case <-time.After(testBudget(5 * time.Second)):
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

// TestConvergenceTwoSessions proves WATCH-04's two invariants
// deterministically and goleak-clean, in this SAME package's existing
// TestMain (no second goleak harness, per RESEARCH's "don't hand-roll"
// warning): (a) while session A holds the daemon lock, session B's
// RunWithRetry never becomes a second writer — at most one writer always;
// (b) once A exits, B's RunWithRetry converges and becomes the sole
// writer — exactly one writer eventually. Both sessions share one
// initFixture root and use an injected sub-second retry interval (D-14's
// production ~30s cadence would make this soak impractically slow) plus a
// Probe pinning IsWSL false so the soak never depends on the host's real
// /proc/version.
func TestConvergenceTwoSessions(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, storeDir := initFixture(t)

	noWSL := watch.Probe{IsWSL: func() bool { return false }}

	dA, err := New(root, indexer.Options{Quiet: true}, WithProbe(noWSL))
	if err != nil {
		t.Fatalf("New (session A): %v", err)
	}
	dB, err := New(root, indexer.Options{Quiet: true}, WithProbe(noWSL))
	if err != nil {
		t.Fatalf("New (session B): %v", err)
	}

	syncedA := make(chan indexer.Stats, 8)
	dA.onSync = func(stats indexer.Stats, err error) {
		if err != nil {
			t.Errorf("session A sync failed: %v", err)
			return
		}
		syncedA <- stats
	}
	syncedB := make(chan indexer.Stats, 8)
	dB.onSync = func(stats indexer.Stats, err error) {
		if err != nil {
			t.Errorf("session B sync failed: %v", err)
			return
		}
		syncedB <- stats
	}

	const retryInterval = 30 * time.Millisecond // injected short (RESEARCH Pattern 3), not the ~30s production cadence

	var deferredBCount int32
	onDeferredB := func() { atomic.AddInt32(&deferredBCount, 1) }

	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	runErrA := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErrA <- RunWithRetry(ctxA, dA, retryInterval, func() {
			t.Error("session A (the first to start) should never observe ErrLockLive")
		})
		close(runErrA)
	}()
	joinDaemonRun(t, cancelA, runErrA)

	waitForLock(t, codegraphDir)

	runErrB := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErrB <- RunWithRetry(ctxB, dB, retryInterval, onDeferredB)
		close(runErrB)
	}()
	joinDaemonRun(t, cancelB, runErrB)

	// Session B must observe ErrLockLive (via onDeferred) while A holds
	// the lock — proof it is retrying, not silently giving up (today's
	// defer-once bug) nor somehow acquiring alongside A.
	deadline := time.Now().Add(testBudget(2 * time.Second))
	for time.Now().Before(deadline) && atomic.LoadInt32(&deferredBCount) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt32(&deferredBCount) == 0 {
		t.Fatal("session B never observed ErrLockLive while session A held the lock")
	}

	// Invariant (a) — at most one writer always: an edit while both
	// sessions are live is synced by A only, never B.
	writeFixtureFile(t, root, "convA.go", "package main\n\nfunc convA() {}\n")
	select {
	case <-syncedA:
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("timed out waiting for session A's sync while it held the lock")
	}
	select {
	case <-syncedB:
		t.Fatal("session B synced while session A held the lock — at most one writer always violated")
	case <-time.After(testBudget(200 * time.Millisecond)):
	}

	// A exits; invariant (b) — exactly one writer eventually — requires B
	// to converge and become the sole writer.
	cancelA()
	select {
	case err := <-runErrA:
		if err != nil {
			t.Fatalf("session A RunWithRetry: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("session A did not shut down")
	}

	// Wait for B to have actually re-acquired the lock (and therefore
	// opened its own watcher) before writing — otherwise the file create
	// could land before B's fsnotify watch is registered and never be
	// observed at all, a race distinct from the convergence property this
	// test is proving.
	waitForLock(t, codegraphDir)

	writeFixtureFile(t, root, "convB.go", "package main\n\nfunc convB() {}\n")
	select {
	case <-syncedB:
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("session B did not converge and become the sole writer after session A exited")
	}

	if names := nodeNames(t, storeDir); !names["convA"] || !names["convB"] {
		t.Fatalf("committed graph = %v, want it to contain both convA (session A) and convB (session B, post-convergence)", names)
	}

	// Terminal discipline (D-15): all cancels + wg.Waits complete before
	// this function returns, so TestMain's VerifyTestMain sees zero leaked
	// goroutines.
	cancelB()
	select {
	case err := <-runErrB:
		if err != nil {
			t.Fatalf("session B RunWithRetry: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("session B did not shut down")
	}
	wg.Wait()

	if _, ok, err := readLock(codegraphDir); err != nil || ok {
		t.Fatalf("lockfile still present after both sessions shut down (ok=%v err=%v)", ok, err)
	}

	// A fresh acquire/release pair succeeding proves the lock was
	// genuinely released, not merely absent for an unrelated reason.
	if err := acquire(codegraphDir); err != nil {
		t.Fatalf("acquire after convergence soak shutdown: %v", err)
	}
	if err := release(codegraphDir); err != nil {
		t.Fatalf("release: %v", err)
	}
}
