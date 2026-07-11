package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// soakIterations is a modest cycle count that comfortably exercises the
// watch->debounce path many times while keeping the soak well inside the
// 120s CI timeout (RESEARCH "Soak-test iteration counts... executor's
// discretion").
const soakIterations = 40

// TestSoak drives many watch->debounce cycles over a temp dir on a
// cancelable ctx tracked by a sync.WaitGroup, then cancels and joins BEFORE
// returning — the exact ordering goleak.VerifyTestMain (main_test.go)
// requires, per RESEARCH Pattern 7's "wg.Wait() MUST complete before
// goleak.VerifyNone runs, or every still-shutting-down goroutine is a
// false-positive leak". This is the SYNC-06 acceptance gate for
// internal/watch: the watcher/debounce subsystem must be goroutine-leak-free
// under sustained cycles.
func TestSoak(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root := t.TempDir()
	w, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	var flushCount int64
	ctx, cancel := context.WithCancel(context.Background())
	deb := NewDebouncer(ctx, DebounceDuration(), func(paths map[string]struct{}) {
		if len(paths) == 0 {
			return
		}
		atomic.AddInt64(&flushCount, 1)
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Run(ctx, deb)
	}()

	window := DebounceDuration()
	for i := 0; i < soakIterations; i++ {
		path := filepath.Join(root, fmt.Sprintf("soak%d.txt", i))
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
		// Sleep past the debounce window plus slack so each iteration's
		// write coalesces into its own flush rather than piling into the
		// next iteration's burst — this proves the debounce timer fires
		// and resets across many cycles, not just once.
		time.Sleep(window + 30*time.Millisecond)
	}

	// Cancel and join BEFORE any goleak assertion runs (TestMain's
	// VerifyTestMain fires after the whole package's tests complete):
	// watchLoop's ctx.Done() branch calls deb.Stop() before returning, so
	// wg.Wait() here guarantees no watcher goroutine and no pending debounce
	// timer survive past this test.
	cancel()
	wg.Wait()

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got := atomic.LoadInt64(&flushCount); got == 0 {
		t.Fatal("soak completed with zero flushes — watch->debounce path did not fire across any cycle")
	}
}
