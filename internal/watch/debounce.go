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

	// fireWG tracks every fire() invocation that has actually started (or
	// is guaranteed to start) running, in its own time.AfterFunc goroutine
	// (CR-01). Wait blocks until all of them — including the flush(...)
	// call each one makes — have completed, closing the race where a
	// caller treats Stop() as a complete join: Stop can only cancel a
	// timer that hasn't fired yet; it cannot wait for one that has already
	// started running.
	fireWG sync.WaitGroup
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
// of Adds within window coalesces into one flush. Once ctx is cancelled,
// Add is a no-op (03-REVIEW.md IN-04): arming a timer post-cancel could
// only ever produce a no-op fire (fire checks ctx.Err()) but would still
// make a caller's Wait block up to a full window on it — the daemon's
// requeue-vs-shutdown TOCTOU. The early return happens before
// fireWG.Add(1), so Wait's accounting is untouched.
func (d *Debouncer) Add(path string) {
	if d.ctx.Err() != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending[path] = struct{}{}
	if d.timer != nil {
		if d.timer.Stop() {
			// Successfully cancelled before firing: that scheduled fire()
			// goroutine will now never run, so undo the fireWG.Add(1) made
			// when it was scheduled (CR-01) — otherwise Wait would block
			// forever on a goroutine that will never call Done.
			d.fireWG.Done()
		}
		// If Stop returns false, fire() has already started running (or
		// already returned) in its own goroutine; it owns its own
		// fireWG.Done() call via defer, so nothing to undo here.
	}
	d.fireWG.Add(1)
	d.timer = time.AfterFunc(d.window, d.fire)
}

// fire runs in its own goroutine (time.AfterFunc's — outside the caller's
// context tree by default, Pattern 7). It MUST check ctx.Err() before
// flushing: Stop() cannot retroactively cancel a callback that has already
// been scheduled to run, so this check is the belt to Stop's suspenders
// against a late flush racing shutdown. The deferred fireWG.Done() (CR-01)
// is what lets Wait join this goroutine — including the flush(...) call
// below — even though Stop can no longer cancel it once it has started.
func (d *Debouncer) fire() {
	defer d.fireWG.Done()
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
// is exactly what goleak.VerifyNone would catch as a leak. Stop does NOT
// wait for a fire() that has already started running — call Wait after
// Stop for that (CR-01).
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		if d.timer.Stop() {
			d.fireWG.Done()
		}
		d.timer = nil
	}
}

// Wait blocks until every fire() invocation that has actually started
// running — i.e. that Stop could not cancel in time — has fully completed,
// including the flush(...) call it makes (CR-01). Callers join the
// Debouncer's lifecycle via Stop (cancel anything not yet running) followed
// by Wait (join anything that is): together these give a caller a genuine
// "no debounce-triggered work is still in flight" guarantee, which Stop
// alone cannot provide since a timer that has already fired is no longer
// cancellable.
func (d *Debouncer) Wait() {
	d.fireWG.Wait()
}
