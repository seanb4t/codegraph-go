package daemon

import (
	"context"
	"os"
	"time"
)

// watchdogInterval is the poll interval between parent-liveness checks.
// ~1s, per D-07's discretion within the 1-2s range: frequent enough to
// notice a dead supervising host/agent process promptly, infrequent enough
// to be a negligible background cost for the lifetime of a daemon.
const watchdogInterval = 1 * time.Second

// getppid is an unexported package-level test seam: watchdog_test.go
// overrides it to drive a synthetic reparent deterministically without
// forking, mirroring daemon.go's onSyncStart/onWatchOpen test-seam
// convention. The POSIX parentChanged predicate (watchdog_posix.go) reads
// through this var rather than calling os.Getppid() directly so the same
// injection point covers it.
var getppid = os.Getppid

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
// Per D-08 this does nothing but call cancel() — it adds no new shutdown
// path; clean teardown (release lock, deregister, join) already flows
// from ctx cancellation via daemon.Run's existing paths.
func startWatchdog(ctx context.Context, cancel context.CancelFunc, interval time.Duration) (stop func()) {
	original := getppid()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if parentChanged(original) {
					cancel()
					return
				}
			}
		}
	}()
	return func() { <-done }
}
