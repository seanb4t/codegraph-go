package watch

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

// defaultDebounceMs is the debounce window used when CODEGRAPH_DEBOUNCE_MS
// is unset or invalid (D-04).
const defaultDebounceMs = 2000

// DebounceDuration returns the debounce window: CODEGRAPH_DEBOUNCE_MS (a
// positive integer number of milliseconds) overrides the 2000ms default; a
// missing, zero, negative, or non-numeric value falls back to the default.
func DebounceDuration() time.Duration {
	if v := os.Getenv("CODEGRAPH_DEBOUNCE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultDebounceMs * time.Millisecond
}

// Debouncer coalesces a burst of Add calls within window into a single
// flush call over the deduplicated union of changed paths (Pattern 3). A
// quiet gap longer than window flushes and resets; a subsequent burst
// flushes again.
type Debouncer struct {
	ctx    context.Context
	window time.Duration
	flush  func(paths map[string]struct{})

	mu      sync.Mutex
	pending map[string]struct{}
	timer   *time.Timer
}

// NewDebouncer returns a Debouncer bound to ctx: once ctx is cancelled, no
// further flush fires (fire checks ctx.Err(), and Stop cancels any pending
// timer — Pattern 7's two-part guarantee against a late-firing timer
// goroutine). flush is invoked from the timer's own goroutine
// (time.AfterFunc) and must be safe to call from an arbitrary goroutine.
func NewDebouncer(ctx context.Context, window time.Duration, flush func(paths map[string]struct{})) *Debouncer {
	return &Debouncer{
		ctx:     ctx,
		window:  window,
		flush:   flush,
		pending: make(map[string]struct{}),
	}
}

// Add records path as changed and (re)starts the debounce timer so a burst
// of Adds within window coalesces into one flush.
func (d *Debouncer) Add(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending[path] = struct{}{}
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.window, d.fire)
}

// fire runs in its own goroutine (time.AfterFunc's — outside the caller's
// context tree by default, Pattern 7). It MUST check ctx.Err() before
// flushing: Stop() cannot retroactively cancel a callback that has already
// been scheduled to run, so this check is the belt to Stop's suspenders
// against a late flush racing shutdown.
func (d *Debouncer) fire() {
	if d.ctx.Err() != nil {
		return
	}
	d.mu.Lock()
	if len(d.pending) == 0 {
		d.mu.Unlock()
		return
	}
	paths := d.pending
	d.pending = make(map[string]struct{})
	d.mu.Unlock()
	d.flush(paths)
}

// Stop cancels any pending timer so no late flush fires after shutdown
// (Pattern 7) — required for Plan 04-09's leak-free soak gate: an
// unstopped timer.AfterFunc callback goroutine, still scheduled to fire,
// is exactly what goleak.VerifyNone would catch as a leak.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
