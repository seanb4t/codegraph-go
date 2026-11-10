//go:build !windows

// Package flock takes advisory flock(2) locks on lock files. The kernel
// releases a lock when its file handle closes, and so when the process
// exits, however it exits.
package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// Mode says whether a lock excludes every other holder or only the
// exclusive ones.
type Mode int

const (
	// Exclusive excludes every other lock on the file.
	Exclusive Mode = iota
	// Shared excludes only an exclusive lock.
	Shared
)

// Open opens the lock file at path, creating it if needed, and takes a
// lock on it without waiting. Close the returned file to release the lock.
func Open(path string, mode Mode) (*os.File, error) {
	return open(path, os.O_RDWR|os.O_CREATE, mode)
}

// OpenExisting is Open for a lock file that must exist already. It never
// creates the file.
func OpenExisting(path string, mode Mode) (*os.File, error) {
	return open(path, os.O_RDWR, mode)
}

// Create creates the lock file at path, which must not exist, and takes a
// lock on it without waiting.
func Create(path string, mode Mode) (*os.File, error) {
	return open(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
}

func open(path string, flag int, mode Mode) (*os.File, error) {
	f, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		return nil, err
	}
	how := unix.LOCK_EX
	if mode == Shared {
		how = unix.LOCK_SH
	}
	if err := unix.Flock(int(f.Fd()), how|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// IsHeld reports whether err, from Open, OpenExisting or Create, means that
// another handle holds a lock that excludes the one asked for. It matches
// only the errno of a non-blocking flock call, so it is valid only for the
// errors of those three functions.
func IsHeld(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK)
}
