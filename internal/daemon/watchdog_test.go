package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestWatchdogCancelsOnReparent drives a synthetic reparent through the
// injectable getppid seam (no forking required) and asserts cancel() fires
// within a bounded time, and that stop() returns promptly afterward —
// proving the poll goroutine actually joined (RESEARCH Pitfall 4).
func TestWatchdogCancelsOnReparent(t *testing.T) {
	origGetppid := getppid
	// t.Cleanup (not defer): a defer positioned here would still run on
	// the runtime.Goexit() unwind a failing t.Fatalf takes, but ONLY a
	// t.Cleanup registered AFTER the join below is guaranteed by LIFO to
	// run AFTER that join's cleanup — see joinDaemonRun's ordering
	// contract (MAINT-01).
	t.Cleanup(func() { getppid = origGetppid })

	const original = 12345
	var current int32 = original
	getppid = func() int { return int(atomic.LoadInt32(&current)) }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := startWatchdog(ctx, cancel, 5*time.Millisecond)
	// joinDaemonRun's t.Cleanup, registered AFTER the getppid restore
	// above, runs FIRST on LIFO unwind: stop() — bounded by the shared
	// budget, not left to block the whole run indefinitely — is guaranteed
	// to complete before getppid is restored, regardless of which
	// statement in this test triggers a Goexit-driven unwind (MAINT-01).
	stopErr := make(chan error, 1)
	go func() {
		stop()
		stopErr <- nil
		close(stopErr)
	}()
	joinDaemonRun(t, cancel, stopErr)

	// Simulate a reparent shortly after start.
	atomic.StoreInt32(&current, original+1)

	select {
	case <-ctx.Done():
	case <-time.After(testBudget(2 * time.Second)):
		t.Fatal("expected ctx to be cancelled after simulated reparent")
	}

	joined := make(chan struct{})
	go func() {
		stop()
		close(joined)
	}()
	select {
	case <-joined:
	case <-time.After(testBudget(2 * time.Second)):
		t.Fatal("stop() did not join the watchdog goroutine promptly")
	}
}

// TestWatchdogJoinsOnCtxCancelWithoutFiringCancel asserts the goroutine
// exits cleanly (and joins via stop()) when ctx is cancelled by something
// OTHER than the watchdog itself, and that in that path the watchdog never
// calls cancel a second time.
func TestWatchdogJoinsOnCtxCancelWithoutFiringCancel(t *testing.T) {
	origGetppid := getppid
	// t.Cleanup, not defer — see TestWatchdogCancelsOnReparent's comment;
	// registered before the join below so LIFO makes the join run first.
	t.Cleanup(func() { getppid = origGetppid })

	const original = 54321
	getppid = func() int { return original } // parent never changes

	ctx, cancel := context.WithCancel(context.Background())

	var cancelCalls int32
	wrappedCancel := func() {
		atomic.AddInt32(&cancelCalls, 1)
		cancel()
	}

	stop := startWatchdog(ctx, wrappedCancel, 5*time.Millisecond)
	stopErr := make(chan error, 1)
	go func() {
		stop()
		stopErr <- nil
		close(stopErr)
	}()
	joinDaemonRun(t, cancel, stopErr)

	cancel() // external cancellation, not a reparent

	joined := make(chan struct{})
	go func() {
		stop()
		close(joined)
	}()
	select {
	case <-joined:
	case <-time.After(testBudget(2 * time.Second)):
		t.Fatal("stop() did not join the watchdog goroutine promptly")
	}

	if got := atomic.LoadInt32(&cancelCalls); got != 0 {
		t.Fatalf("expected watchdog not to call cancel on ctx-cancel exit, got %d calls", got)
	}
}
