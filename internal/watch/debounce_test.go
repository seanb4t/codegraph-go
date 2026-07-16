package watch

import (
	"context"
	"testing"
	"time"
)

// TestDebounceCoalescesBurst proves N rapid Add calls within the debounce
// window collapse into exactly one flush, whose argument is the
// deduplicated union of the added paths.
func TestDebounceCoalescesBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flushed := make(chan map[string]struct{}, 8)
	d := NewDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
		flushed <- paths
	})

	// A rapid burst, including a duplicate path, well inside the window.
	d.Add("a.go")
	d.Add("b.go")
	d.Add("a.go")
	d.Add("c.go")

	select {
	case paths := <-flushed:
		want := map[string]struct{}{"a.go": {}, "b.go": {}, "c.go": {}}
		if len(paths) != len(want) {
			t.Fatalf("flush paths = %v, want union %v", paths, want)
		}
		for p := range want {
			if _, ok := paths[p]; !ok {
				t.Fatalf("flush paths %v missing %q", paths, p)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the coalesced flush")
	}

	// No second flush should follow from the same burst.
	select {
	case paths := <-flushed:
		t.Fatalf("unexpected second flush: %v", paths)
	case <-time.After(100 * time.Millisecond):
	}

	// A quiet gap longer than the window, then a fresh burst, flushes
	// again — proving the Debouncer resets after firing.
	d.Add("d.go")
	select {
	case paths := <-flushed:
		if _, ok := paths["d.go"]; !ok || len(paths) != 1 {
			t.Fatalf("second-burst flush paths = %v, want {d.go}", paths)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the second burst's flush")
	}
}

// TestDebounceEnvTunable proves the 2000ms default, the
// CODEGRAPH_DEBOUNCE_MS override, and the fallback-to-default behavior for
// zero/negative/non-numeric values.
func TestDebounceEnvTunable(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		t.Setenv("CODEGRAPH_DEBOUNCE_MS", "")
		if got := DebounceDuration(); got != defaultDebounceMs*time.Millisecond {
			t.Fatalf("DebounceDuration() = %v, want %v", got, defaultDebounceMs*time.Millisecond)
		}
	})

	t.Run("positive override honored", func(t *testing.T) {
		t.Setenv("CODEGRAPH_DEBOUNCE_MS", "500")
		if got, want := DebounceDuration(), 500*time.Millisecond; got != want {
			t.Fatalf("DebounceDuration() = %v, want %v", got, want)
		}
	})

	badValues := []string{"0", "-100", "not-a-number", "  ", "3.5"}
	for _, v := range badValues {
		t.Run("falls back to default for "+v, func(t *testing.T) {
			t.Setenv("CODEGRAPH_DEBOUNCE_MS", v)
			if got := DebounceDuration(); got != defaultDebounceMs*time.Millisecond {
				t.Fatalf("DebounceDuration() with CODEGRAPH_DEBOUNCE_MS=%q = %v, want default %v", v, got, defaultDebounceMs*time.Millisecond)
			}
		})
	}
}

// TestDebounceNoFlushAfterCancel proves no flush fires after ctx
// cancellation. Two independent guarantees are exercised: fire's own
// ctx.Err() check (Pattern 7's belt), and Stop's explicit timer.Stop()
// (Pattern 7's suspenders) — both must hold for Plan 04-09's leak-free
// soak gate.
func TestDebounceNoFlushAfterCancel(t *testing.T) {
	t.Run("ctx cancelled before window elapses, Stop not called", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		flushed := make(chan map[string]struct{}, 1)
		d := NewDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
			flushed <- paths
		})

		d.Add("a.go")
		cancel() // cancel before the window elapses; do not call Stop

		select {
		case paths := <-flushed:
			t.Fatalf("flush fired after ctx cancellation: %v", paths)
		case <-time.After(150 * time.Millisecond):
			// no flush — correct.
		}
	})

	t.Run("Stop cancels the pending timer outright", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		flushed := make(chan map[string]struct{}, 1)
		d := NewDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
			flushed <- paths
		})

		d.Add("a.go")
		d.Stop()

		select {
		case paths := <-flushed:
			t.Fatalf("flush fired after Stop: %v", paths)
		case <-time.After(150 * time.Millisecond):
			// no flush — correct.
		}
	})
}

// TestDebounceAddAfterCancelIsNoOp pins IN-04's structural fix (WR-01
// coverage gap): Add called AFTER ctx cancellation must be a complete
// no-op — no path recorded, no timer armed, no fireWG count taken — so a
// caller's Wait returns immediately instead of riding out a dead debounce
// window. TestDebounceNoFlushAfterCancel covers fire's own ctx check and
// Stop; this is the third guarantee: the daemon's requeue-vs-shutdown
// TOCTOU where a requeue Add lands after cancellation.
func TestDebounceAddAfterCancelIsNoOp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	flushed := make(chan map[string]struct{}, 1)
	d := NewDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
		flushed <- paths
	})

	cancel()
	d.Add("x.go")

	// Structural pin (same-package access): the ctx gate must return
	// before recording the path or arming a timer — deterministic, no
	// timing involved.
	d.mu.Lock()
	timerArmed := d.timer != nil
	pendingLen := len(d.pending)
	d.mu.Unlock()
	if timerArmed {
		t.Fatal("Add after ctx cancellation armed a timer — IN-04 regression")
	}
	if pendingLen != 0 {
		t.Fatalf("Add after ctx cancellation recorded %d pending path(s), want 0", pendingLen)
	}

	// Wait must return immediately: the early return happens before
	// fireWG.Add(1), so there is nothing to join.
	waitReturned := make(chan struct{})
	go func() {
		d.Wait()
		close(waitReturned)
	}()
	select {
	case <-waitReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait blocked after a post-cancel Add — the no-op Add touched fireWG accounting")
	}

	// Behavioral belt: no flush fires within ~2 debounce windows.
	select {
	case paths := <-flushed:
		t.Fatalf("flush fired after a post-cancel Add: %v", paths)
	case <-time.After(75 * time.Millisecond):
	}
}

// TestDebounceWaitJoinsInFlightFire is the CR-01 regression: proves Wait
// blocks until a fire() that had already started running — one Stop() can
// no longer cancel, because the timer already fired — has fully completed,
// including the flush(...) call it makes. Before CR-01's fix, Debouncer had
// no Wait method at all, so a caller's only join primitive (Stop) could not
// distinguish "timer never fired" from "timer fired and its callback is
// still running" — exactly the gap that let Daemon.Run release the daemon
// lock while a debounced indexer.Sync was still writing.
func TestDebounceWaitJoinsInFlightFire(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	d := NewDebouncer(ctx, 10*time.Millisecond, func(paths map[string]struct{}) {
		close(started)
		<-release
		close(finished)
	})

	d.Add("a.go")

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fire() to start running flush")
	}

	// Simulate the daemon's shutdown sequence: cancel ctx, then Stop — but
	// the timer has already fired, so Stop can only be a no-op against
	// this in-flight invocation.
	cancel()
	d.Stop()

	waitReturned := make(chan struct{})
	go func() {
		d.Wait()
		close(waitReturned)
	}()

	select {
	case <-waitReturned:
		t.Fatal("Debouncer.Wait returned before the in-flight flush finished — CR-01 regression")
	case <-time.After(150 * time.Millisecond):
		// correct: Wait must still be blocked.
	}

	close(release)

	select {
	case <-waitReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("Debouncer.Wait did not return after the in-flight flush finished")
	}

	select {
	case <-finished:
	default:
		t.Fatal("Wait returned before flush's own body completed")
	}
}
