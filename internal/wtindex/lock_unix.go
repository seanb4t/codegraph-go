//go:build !windows

package wtindex

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// lockFile opens path, creating it if needed, and takes an exclusive flock
// on it without waiting. The kernel releases the lock when the process
// exits.
func lockFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// sameDevice reports whether paths a and b are on one filesystem, where a
// checkpoint can hard-link instead of copying. A test can replace it.
var sameDevice = func(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return ai.Sys().(*syscall.Stat_t).Dev == bi.Sys().(*syscall.Stat_t).Dev, nil
}
