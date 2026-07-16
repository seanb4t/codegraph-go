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
func TestOpenConvergesWhenHolderCloses(t *testing.T) {
	dir := t.TempDir()

	holder, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	closeErr := make(chan error, 1)
	go func() {
		// Release mid-retry: after attempt 1 (t=0) and attempt 2
		// (t≈100ms) have failed, before the budget (4 sleeps, ~400ms
		// of waiting) runs out.
		time.Sleep(150 * time.Millisecond)
		closeErr <- holder.Close()
	}()

	second, err := Open(dir)
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
