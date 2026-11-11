package graphstore

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore/locktest"
)

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
	release := locktest.HoldStoreLock(t, dir)

	sleeps := make(chan struct{})
	released := make(chan struct{})
	orig := openLockRetrySleep
	openLockRetrySleep = func(time.Duration) { sleeps <- struct{}{} }
	t.Cleanup(func() { openLockRetrySleep = orig })

	go func() {
		<-sleeps // attempt 2's sleep began: attempt 1 has failed lock-held
		release()
		close(released)
		for range sleeps {
			// Drain later attempts' sleep signals (each send happens-after
			// the release above) so Open never blocks on the seam.
		}
	}()

	second, err := Open(dir)
	close(sleeps) // Open has returned; end the drain loop
	if err != nil {
		t.Fatalf("Open did not converge after the holder released: %v", err)
	}
	second.Close()
	<-released
}

// TestOpenSucceedsOnTheFinalAttempt pins the FINAL-ATTEMPT boundary of
// Open's bounded retry loop — the property TestOpenConvergesWhenHolderCloses
// above does NOT cover. That test's own doc comment states the holder is
// released once "attempt 2's backoff sleep began" (some remaining
// attempt out of budget 5), which is release-BETWEEN-attempts, not the
// final-attempt boundary. internal/uiserver's degrade tests delegate
// this specific boundary here, because openLockRetrySleep is unexported
// and unreachable from that package.
//
// Deterministic and wall-clock-free, built on the SAME openLockRetrySleep
// seam TestOpenConvergesWhenHolderCloses already uses, with one addition:
// an acknowledgement channel (resume). TestOpenConvergesWhenHolderCloses's
// drain-loop trick works because a LATER sleep blocks until Close
// returns; on the FINAL sleep there is no later sleep, so without an
// acknowledgement the holder's Close would race the final pebble.Open.
// With the acknowledgement, Close happens-before the final attempt,
// deterministically.
//
// Sleep ordinals are counted 1-INDEXED (the Nth call to
// openLockRetrySleep), matching "the fourth and LAST sleep" for
// openLockRetryAttempts=5 (4 sleeps occur, before attempts 2 through 5 in
// 1-indexed human terms — attempt 1 never sleeps). The LAST ordinal is
// therefore openLockRetryAttempts-1, which is both the release condition
// below and the assertion that the observed release ordinal really was
// the final one, not an earlier one.
func TestOpenSucceedsOnTheFinalAttempt(t *testing.T) {
	dir := t.TempDir()
	release := locktest.HoldStoreLock(t, dir)

	sleeps := make(chan int)
	resume := make(chan struct{})
	orig := openLockRetrySleep
	ordinal := 0
	openLockRetrySleep = func(time.Duration) {
		ordinal++
		sleeps <- ordinal
		<-resume
	}
	t.Cleanup(func() { openLockRetrySleep = orig })

	var observedOrdinal int
	go func() {
		for o := range sleeps {
			observedOrdinal = o
			if o == openLockRetryAttempts-1 {
				// The fourth and LAST sleep, immediately preceding the
				// fifth and final attempt: release the holder NOW, before
				// acknowledging, so the release happens-before the final
				// pebble.Open rather than racing it.
				release()
			}
			resume <- struct{}{}
		}
	}()

	second, err := Open(dir)
	close(sleeps) // Open has returned; end the goroutine's range loop.
	if err != nil {
		t.Fatalf("Open did not succeed on the final attempt: %v", err)
	}
	second.Close()

	if observedOrdinal != openLockRetryAttempts-1 {
		t.Fatalf("observed release sleep ordinal = %d, want %d (openLockRetryAttempts-1) — success must be proven to occur on the FINAL attempt, not an earlier one (that is TestOpenConvergesWhenHolderCloses's case)", observedOrdinal, openLockRetryAttempts-1)
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
