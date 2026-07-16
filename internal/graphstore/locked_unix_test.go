//go:build !windows

package graphstore

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"syscall"
	"testing"
)

// TestIsLockHeldOSUnix pins the unix classifier's exact match set
// (03-REVIEW-2.md WR-02): both real pebble lock forms match — including
// through the fs.PathError wrapping pebble's fcntl path produces, which
// was previously verified only by a manual source read — and the shapes
// that must NOT match (EACCES above all, per WR-01) stay excluded.
func TestIsLockHeldOSUnix(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			// Cross-process form: fcntl(F_SETLK) conflict. Pebble's vfs
			// surfaces the errno wrapped; errors.Is must traverse.
			name: "fcntl EAGAIN wrapped in PathError",
			err:  fmt.Errorf("pebble: open: %w", &fs.PathError{Op: "fcntl", Path: "/repo/.codegraph/store/LOCK", Err: syscall.EAGAIN}),
			want: true,
		},
		{
			name: "bare EWOULDBLOCK",
			err:  syscall.EWOULDBLOCK,
			want: true,
		},
		{
			// Same-process form: pebble's in-process map message
			// (vfs/file_lock_unix.go, unexported — string-matched).
			name: "in-process message wrapped",
			err:  fmt.Errorf("pebble: %w", errors.New("lock held by current process")),
			want: true,
		},
		{
			// WR-01: EACCES is a permission failure on every release
			// target, never a lock conflict — must stay fatal.
			name: "EACCES is not a lock",
			err:  &fs.PathError{Op: "open", Path: "/repo/.codegraph/store/LOCK", Err: syscall.EACCES},
			want: false,
		},
		{
			name: "unrelated sentinel",
			err:  ErrNotFound,
			want: false,
		},
		{
			name: "arbitrary error",
			err:  errors.New("pebble: corruption detected"),
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

// TestClassifyOpenErrorWrapsUnixLockForms pins that classifyOpenError
// attaches the sentinel to both unix lock forms while preserving the
// original error text for diagnostics.
func TestClassifyOpenErrorWrapsUnixLockForms(t *testing.T) {
	raw := &fs.PathError{Op: "fcntl", Path: "/repo/.codegraph/store/LOCK", Err: syscall.EAGAIN}
	got := classifyOpenError(raw)
	if !errors.Is(got, ErrStoreLocked) {
		t.Fatalf("classifyOpenError(fcntl EAGAIN) = %v; want ErrStoreLocked wrap", got)
	}
	if want := raw.Error(); !strings.Contains(got.Error(), want) {
		t.Fatalf("wrapped error %q does not preserve original text %q", got.Error(), want)
	}
}
