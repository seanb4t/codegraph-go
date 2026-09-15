package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestWatchdogCancelsOnReparent drives a synthetic reparent through
// startWatchdog's injected ppid func and tick channel (no forking, no real
// ticker required) and asserts cancel() fires within a bounded time, and
// that stop() returns promptly afterward — proving the poll goroutine
// actually joined (RESEARCH Pitfall 4).
func TestWatchdogCancelsOnReparent(t *testing.T) {
	const original = 12345
	var current int32 = original
	ppid := func() int { return int(atomic.LoadInt32(&current)) }
	ticks := make(chan time.Time)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := startWatchdog(ctx, cancel, 5*time.Millisecond, ppid, ticks)
	stopErr := make(chan error, 1)
	go func() {
		stop()
		stopErr <- nil
		close(stopErr)
	}()
	joinDaemonRun(t, cancel, stopErr)

	// Simulate a reparent, then drive the watchdog's next poll directly —
	// no real ticker is involved anywhere in this path.
	atomic.StoreInt32(&current, original+1)
	select {
	case ticks <- time.Now():
	case <-time.After(testBudget(2 * time.Second)):
		t.Fatal("could not send a tick to the watchdog goroutine")
	}

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
	const original = 54321
	ppid := func() int { return original } // parent never changes
	ticks := make(chan time.Time)

	ctx, cancel := context.WithCancel(context.Background())

	var cancelCalls int32
	wrappedCancel := func() {
		atomic.AddInt32(&cancelCalls, 1)
		cancel()
	}

	stop := startWatchdog(ctx, wrappedCancel, 5*time.Millisecond, ppid, ticks)
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
