package fsatomic

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ErrExchangeUnsupported reports a filesystem that cannot exchange two
// directories in one step.
var ErrExchangeUnsupported = errors.New("fsatomic: the filesystem does not support an atomic exchange of two directories")

// exchange is the system call behind ExchangeDirs. A test replaces it.
var exchange = exchangeOS

// syncDir runs fsync on one directory. A test wraps it to record the
// directories.
var syncDir = func(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ExchangeDirs swaps the directories at a and b in one system call, so no
// moment exists at which either path is missing. It then runs fsync on the
// parent of each path. Both paths must exist, on one filesystem. An error
// that wraps ErrNotDurable means that the swap happened. Every other error
// means that both directories stay as they were.
//
// A filesystem without the operation gives an error that wraps
// ErrExchangeUnsupported, and both directories stay as they were. There is
// no fallback to two renames, because a crash between them leaves one path
// empty.
func ExchangeDirs(a, b string) error {
	if err := exchange(a, b); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, errors.ErrUnsupported) {
			return fmt.Errorf("%w: exchange %s and %s: %w", ErrExchangeUnsupported, a, b, err)
		}
		return fmt.Errorf("fsatomic: exchange %s and %s: %w", a, b, err)
	}
	return syncParents(a, b)
}

// RenameDir renames the directory at oldPath to newPath, then runs fsync on
// the parent of each path. An error that wraps ErrNotDurable means that the
// rename happened.
func RenameDir(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	return syncParents(oldPath, newPath)
}

// ErrNotDurable reports that ExchangeDirs or RenameDir changed the
// directories, but the fsync of a parent failed. The paths hold what the
// call put there. A crash can still undo the change.
var ErrNotDurable = errors.New("fsatomic: the change is in place but not durable")

// syncParents runs fsync on the parent of a and on the parent of b. A
// rename changes an entry in each parent, so both need it for the change to
// be durable. If the two paths share a parent, it runs fsync once. Its
// error wraps ErrNotDurable.
func syncParents(a, b string) error {
	pa, pb := filepath.Dir(a), filepath.Dir(b)
	if err := syncDir(pa); err != nil {
		return fmt.Errorf("%w: fsync %s: %w", ErrNotDurable, pa, err)
	}
	if pb == pa {
		return nil
	}
	if err := syncDir(pb); err != nil {
		return fmt.Errorf("%w: fsync %s: %w", ErrNotDurable, pb, err)
	}
	return nil
}
