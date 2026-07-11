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
	d := newDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
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
	// again — proving the debouncer resets after firing.
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
		if got := debounceDuration(); got != defaultDebounceMs*time.Millisecond {
			t.Fatalf("debounceDuration() = %v, want %v", got, defaultDebounceMs*time.Millisecond)
		}
	})

	t.Run("positive override honored", func(t *testing.T) {
		t.Setenv("CODEGRAPH_DEBOUNCE_MS", "500")
		if got, want := debounceDuration(), 500*time.Millisecond; got != want {
			t.Fatalf("debounceDuration() = %v, want %v", got, want)
		}
	})

	badValues := []string{"0", "-100", "not-a-number", "  ", "3.5"}
	for _, v := range badValues {
		t.Run("falls back to default for "+v, func(t *testing.T) {
			t.Setenv("CODEGRAPH_DEBOUNCE_MS", v)
			if got := debounceDuration(); got != defaultDebounceMs*time.Millisecond {
				t.Fatalf("debounceDuration() with CODEGRAPH_DEBOUNCE_MS=%q = %v, want default %v", v, got, defaultDebounceMs*time.Millisecond)
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
		d := newDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
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
		d := newDebouncer(ctx, 30*time.Millisecond, func(paths map[string]struct{}) {
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
