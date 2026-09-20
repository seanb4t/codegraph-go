package daemon

import (
	"context"
	"time"
)

// watchdogInterval is the poll interval between parent-liveness checks.
// ~1s, per D-07's discretion within the 1-2s range: frequent enough to
// notice a dead supervising host/agent process promptly, infrequent enough
// to be a negligible background cost for the lifetime of a daemon.
const watchdogInterval = 1 * time.Second

// startWatchdog launches a background poll goroutine that cancels ctx when
// the process's original parent goes away — POSIX: getppid() changes away
// from the captured baseline (Pattern 5, robust to subreaper reparenting,
// not just ppid==1); Windows: the captured parent pid stops being alive
// (watchdog_windows.go). It returns a stop func() that blocks until the
// goroutine has actually joined, whether it exited via ctx.Done() or via
// detecting a reparent — mirroring daemon.Run's wg.Wait() join discipline
// (RESEARCH Pitfall 4) so internal/daemon's goleak-gated TestMain stays
// clean.
//
// ppid and ticks are the per-instance test seam (D-13/D-14): production
// callers (daemon.Run) pass os.Getppid and a nil ticks channel, in which
// case startWatchdog constructs its own real time.Ticker at interval and
// stops it on exit — the production path is byte-for-byte equivalent in
// behavior to a hard-coded ticker. A test passes its own func() int and a
// channel it owns, so it can drive "next poll" deterministically instead
// of racing a real wall-clock ticker under load. Because the seam is a
// parameter rather than a package-level binding, two Daemon instances
// (or two tests) injecting different readers can never race each other —
// the FIX-08 data race is structurally impossible, not merely serialized
// by test join discipline. A closed ticks channel is treated the same as
// ctx.Done(): the goroutine returns rather than spinning on a
// perpetually-ready closed channel.
//
// Per D-08 this does nothing but call cancel() — it adds no new shutdown
// path; clean teardown (release lock, deregister, join) already flows
// from ctx cancellation via daemon.Run's existing paths.
func startWatchdog(ctx context.Context, cancel context.CancelFunc, interval time.Duration, ppid func() int, ticks <-chan time.Time) (stop func()) {
	original := ppid()
	done := make(chan struct{})
	go func() {
		defer close(done)
		tickC := ticks
		if tickC == nil {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			tickC = ticker.C
		}
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-tickC:
				if !ok {
					// Closed injected channel: treat exactly like
					// ctx.Done() rather than busy-spinning on a
					// perpetually-ready closed channel.
					return
				}
				if parentChanged(original, ppid) {
					cancel()
					return
				}
			}
		}
	}()
	return func() { <-done }
}
