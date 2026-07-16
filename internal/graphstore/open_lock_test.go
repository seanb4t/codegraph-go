package graphstore

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"
	"time"
)

// TestOpenSecondOpenInProcessReturnsErrStoreLocked pins the load-bearing
// seam of the CR-01 fix (03-REVIEW-2.md WR-02) at unit speed, on every
// platform CI runs: a second Open of the SAME directory while the first
// store is still open must (a) exercise the platform's real lock-held
// failure shape — on unix that is pebble's unexported in-process
// "lock held by current process" message, so a pebble version bump that
// rewords it turns THIS test red instead of silently disabling the
// daemon requeue and serve downgrade — (b) surface wrapped in the
// exported ErrStoreLocked sentinel, and (c) only after Open's full
// bounded retry budget (openLockRetryAttempts−1 sleeps of
// openLockRetryBackoff each — a deterministic LOWER bound; no upper
// bound is asserted, so this cannot flake on a slow machine).
func TestOpenSecondOpenInProcessReturnsErrStoreLocked(t *testing.T) {
	dir := t.TempDir()

	holder, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer holder.Close()

	start := time.Now()
	second, err := Open(dir)
	elapsed := time.Since(start)

	if err == nil {
		second.Close()
		t.Fatal("second Open succeeded while the first store was still open; want lock-held failure")
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("second Open error = %v; want errors.Is(err, ErrStoreLocked)", err)
	}
	if want := (openLockRetryAttempts - 1) * openLockRetryBackoff; elapsed < want {
		t.Fatalf("second Open returned after %v; want >= %v (the full bounded retry budget)", elapsed, want)
	}
}

// TestOpenConvergesWhenHolderCloses pins the retry loop's
// success-after-contention behavior: the holder releasing the LOCK
// between attempts must convert the collision into a successful Open —
// the exact transient-flush-window scenario the CR-01 retry exists for.
//
// 03-REVIEW.md IN-02: the release is event-synchronized on the retry
// loop's own attempt boundaries via the openLockRetrySleep seam rather
// than a wall-clock sleep racing the ~400ms budget under parallel CI
// load. The closer goroutine releases the holder only after observing
// attempt 2's backoff sleep begin (attempt 1 has already failed
// lock-held); any attempt whose sleep starts while the Close is still
// in progress blocks on the unbuffered channel until the goroutine
// re-enters its drain loop — i.e. until Close has returned — so some
// remaining attempt (budget: 5) deterministically finds the LOCK free.
func TestOpenConvergesWhenHolderCloses(t *testing.T) {
	dir := t.TempDir()

	holder, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	sleeps := make(chan struct{})
	closeErr := make(chan error, 1)
	orig := openLockRetrySleep
	openLockRetrySleep = func(time.Duration) { sleeps <- struct{}{} }
	t.Cleanup(func() { openLockRetrySleep = orig })

	go func() {
		<-sleeps // attempt 2's sleep began: attempt 1 has failed lock-held
		closeErr <- holder.Close()
		for range sleeps {
			// Drain later attempts' sleep signals (each send happens-after
			// Close returned above) so Open never blocks on the seam.
		}
	}()

	second, err := Open(dir)
	close(sleeps) // Open has returned; end the drain loop
	if err != nil {
		t.Fatalf("second Open did not converge after the holder released: %v", err)
	}
	second.Close()
	if err := <-closeErr; err != nil {
		t.Fatalf("holder Close: %v", err)
	}
}

// TestClassifyOpenErrorSharedPath tests the platform-neutral shared
// classification path with synthesized errors (03-REVIEW-2.md WR-02 /
// WR-01): only errors matching the running platform's pebble lock shape
// gain the ErrStoreLocked sentinel; everything else — crucially a
// permission-denied fs.PathError like the ones indexer.Sync's
// WalkDir/contentHash chains propagate — passes through unchanged and
// can never be mistaken for lock contention by the errors.Is consumers.
func TestClassifyOpenErrorSharedPath(t *testing.T) {
	if got := classifyOpenError(nil); got != nil {
		t.Fatalf("classifyOpenError(nil) = %v; want nil", got)
	}

	// The WR-01 regression pin: an EACCES-carrying PathError (unreadable
	// file/dir — a permanent permission failure) must NOT classify as
	// lock-held on any platform. syscall.EACCES exists on windows too
	// (WSAEACCES), and is distinct from ERROR_SHARING_VIOLATION there.
	eacces := &fs.PathError{Op: "open", Path: "/repo/unreadable.go", Err: syscall.EACCES}
	if got := classifyOpenError(eacces); errors.Is(got, ErrStoreLocked) {
		t.Fatalf("classifyOpenError(EACCES PathError) = %v; must not carry ErrStoreLocked", got)
	} else if got != eacces { //nolint:errorlint // identity check is the point: non-lock errors pass through unchanged
		t.Fatalf("classifyOpenError(EACCES PathError) = %v; want the error returned unchanged", got)
	}

	// Unrelated sentinels pass through unchanged too.
	if got := classifyOpenError(ErrNotFound); got != ErrNotFound || errors.Is(got, ErrStoreLocked) {
		t.Fatalf("classifyOpenError(ErrNotFound) = %v; want ErrNotFound unchanged", got)
	}
}
