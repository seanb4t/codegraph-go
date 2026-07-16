//go:build windows

package graphstore

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"
)

// TestIsLockHeldOSWindows pins the windows classifier (03-REVIEW-2.md
// CR-01/WR-02): pebble's vfs/file_lock_windows.go acquires the LOCK via
// CreateFile(share=0), so every collision — same-process and
// cross-process alike — surfaces as ERROR_SHARING_VIOLATION (errno 32),
// and nothing else may match. This file compiles only under GOOS=windows;
// on other platforms the classifier's cross-GOOS integrity is held by
// `GOOS=windows go vet ./internal/graphstore/` plus the platform-neutral
// tests in open_lock_test.go.
func TestIsLockHeldOSWindows(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "bare ERROR_SHARING_VIOLATION",
			err:  syscall.Errno(32),
			want: true,
		},
		{
			name: "ERROR_SHARING_VIOLATION wrapped in PathError",
			err:  fmt.Errorf("pebble: open: %w", &fs.PathError{Op: "CreateFile", Path: `C:\repo\.codegraph\store\LOCK`, Err: syscall.Errno(32)}),
			want: true,
		},
		{
			// ERROR_ACCESS_DENIED (5) is windows' permission failure —
			// the analogue of unix's WR-01 EACCES pin: must stay fatal,
			// never a retryable lock.
			name: "ERROR_ACCESS_DENIED is not a lock",
			err:  &fs.PathError{Op: "CreateFile", Path: `C:\repo\.codegraph\store\LOCK`, Err: syscall.Errno(5)},
			want: false,
		},
		{
			// The unix in-process message never occurs on windows (no
			// in-process tracking map in pebble's windows vfs) and must
			// not be matched here.
			name: "unix in-process message is not matched on windows",
			err:  errors.New("lock held by current process"),
			want: false,
		},
		{
			name: "unrelated sentinel",
			err:  ErrNotFound,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLockHeldOS(tc.err); got != tc.want {
				t.Fatalf("isLockHeldOS(%v) = %v; want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestClassifyOpenErrorWrapsSharingViolation pins the sentinel wrap on
// the windows lock form.
func TestClassifyOpenErrorWrapsSharingViolation(t *testing.T) {
	raw := &fs.PathError{Op: "CreateFile", Path: `C:\repo\.codegraph\store\LOCK`, Err: syscall.Errno(32)}
	got := classifyOpenError(raw)
	if !errors.Is(got, ErrStoreLocked) {
		t.Fatalf("classifyOpenError(ERROR_SHARING_VIOLATION) = %v; want ErrStoreLocked wrap", got)
	}
}
