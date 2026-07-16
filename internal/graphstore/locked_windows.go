//go:build windows

package graphstore

import (
	"errors"
	"syscall"
)

// errSharingViolation is windows' ERROR_SHARING_VIOLATION (winerror 32).
// Declared as syscall.Errno directly — golang.org/x/sys/windows returns
// plain syscall.Errno values from its syscall wrappers, so errors.Is
// against this constant matches without importing x/sys/windows here.
const errSharingViolation = syscall.Errno(32)

// isLockHeldOS reports whether err is pebble's windows "directory LOCK
// already held" open-failure form. Verified against the pinned
// pebble/v2@v2.1.6 vfs/file_lock_windows.go: the LOCK is acquired via
// windows.CreateFile(..., shareMode=0, CREATE_ALWAYS, ...), so EVERY
// collision — same-process and cross-process alike (windows' vfs has no
// in-process tracking map, unlike unix) — surfaces as
// ERROR_SHARING_VIOLATION. The unix forms (the "lock held by current
// process" message, fcntl EAGAIN) never occur here (03-REVIEW-2.md CR-01).
//
// Callers: classifyOpenError only — this matching is valid solely for
// errors whose provenance is pebble.Open.
func isLockHeldOS(err error) bool {
	return errors.Is(err, errSharingViolation)
}
