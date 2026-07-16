//go:build !windows

package graphstore

import (
	"errors"
	"strings"
	"syscall"
)

// isLockHeldOS reports whether err is one of pebble's unix "directory LOCK
// already held" open-failure forms. Verified against the pinned
// pebble/v2@v2.1.6 vfs/file_lock_unix.go:
//
//   - Same-process: pebble's vfs tracks in-process locks in a map and fails
//     with the literal message "lock held by current process" before ever
//     touching the filesystem. String-matched — pebble exports no sentinel
//     for it, so a pebble version bump that rewords the message silently
//     disables this arm; TestOpenSecondOpenInProcessReturnsErrStoreLocked
//     (open_lock_test.go) pins it against exactly that.
//   - Cross-process: the LOCK file is acquired via a non-blocking
//     fcntl(F_SETLK); a conflicting holder in another process surfaces as
//     EAGAIN (EWOULDBLOCK == EAGAIN on linux/darwin; matched separately for
//     any platform where they differ).
//
// EACCES is deliberately NOT matched (03-REVIEW-2.md WR-01): POSIX permits
// it for an F_SETLK conflict only on systems this project does not ship
// to, while on every actual release target an EACCES from pebble.Open
// means a genuine permission failure (e.g. vfs's os.Create(LOCK) against
// an unwritable store dir) that retrying can never fix — it must stay
// fatal, not degrade into the lock retry/requeue paths.
//
// Callers: classifyOpenError only — this matching is valid solely for
// errors whose provenance is pebble.Open.
func isLockHeldOS(err error) bool {
	if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
		return true
	}
	return strings.Contains(err.Error(), "lock held by current process")
}
