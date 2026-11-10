package graphstore

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// noRetrySleep makes retryLocked's backoff return at once for the test.
func noRetrySleep(t *testing.T) {
	t.Helper()
	orig := openLockRetrySleep
	openLockRetrySleep = func(time.Duration) {}
	t.Cleanup(func() { openLockRetrySleep = orig })
}

// TestLockRetryClassifies checks that each classifier turns only the
// lock-held error of its own lock into ErrStoreLocked, and that the helper
// retries only on that error.
func TestLockRetryClassifies(t *testing.T) {
	noRetrySleep(t)
	pebbleHeld := errors.New("lock held by current process")
	for _, tc := range []struct {
		name       string
		errs       []error // one per attempt; the last one repeats
		classify   func(error) error
		wantLocked bool
		wantErr    error
		wantCalls  int
	}{
		{"Pebble lock held, through classifyOpenError", []error{pebbleHeld}, classifyOpenError, true, nil, openLockRetryAttempts},
		{"EWOULDBLOCK, through the flock classifier", []error{syscall.EWOULDBLOCK}, classifyFlockError, true, nil, openLockRetryAttempts},
		{"EACCES, through classifyOpenError", []error{syscall.EACCES}, classifyOpenError, false, syscall.EACCES, 1},
		{"EACCES, through the flock classifier", []error{syscall.EACCES}, classifyFlockError, false, syscall.EACCES, 1},
		{"held once, then free", []error{syscall.EWOULDBLOCK, nil}, classifyFlockError, false, nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got, err := retryLocked(func() (int, error) {
				e := tc.errs[min(calls, len(tc.errs)-1)]
				calls++
				if e != nil {
					return 0, e
				}
				return 42, nil
			}, tc.classify)

			if calls != tc.wantCalls {
				t.Errorf("attempt ran %d times, want %d", calls, tc.wantCalls)
			}
			switch {
			case tc.wantLocked:
				if !errors.Is(err, ErrStoreLocked) {
					t.Errorf("error %v, want ErrStoreLocked", err)
				}
			case tc.wantErr != nil:
				if err != tc.wantErr {
					t.Errorf("error %v, want %v unchanged", err, tc.wantErr)
				}
			default:
				// Positive assertion: a second attempt that succeeds gives its value.
				if err != nil || got != 42 {
					t.Errorf("got %d, %v; want 42 and no error", got, err)
				}
			}
		})
	}
}

func TestOpenExistingMissingStore(t *testing.T) {
	t.Run("a missing directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "store")
		if _, err := OpenExisting(dir); err != ErrNotFound {
			t.Fatalf("OpenExisting: %v, want ErrNotFound", err)
		}
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Lstat(%s) = %v, want the directory still missing", dir, err)
		}
	})
	t.Run("an empty directory", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := OpenExisting(dir); err != ErrNotFound {
			t.Fatalf("OpenExisting: %v, want ErrNotFound", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("the directory holds %d entries, %v; want none, so no LOCK file", len(entries), err)
		}
	})
	// Positive assertion: an existing store opens, and its metadata reads.
	t.Run("an existing store", func(t *testing.T) {
		dir := t.TempDir()
		created, err := Open(dir)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		w, err := created.NewWriter()
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		if err := w.PutMeta(&schema.Meta{NodeCount: 7}); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}
		if err := w.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
		if err := created.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		s, err := OpenExisting(dir)
		if err != nil {
			t.Fatalf("OpenExisting: %v", err)
		}
		defer s.Close()
		r, err := s.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		defer r.Close()
		meta, err := r.GetMeta()
		if err != nil || meta.GetNodeCount() != 7 {
			t.Fatalf("GetMeta = %v, %v; want node count 7", meta, err)
		}
	})
}
