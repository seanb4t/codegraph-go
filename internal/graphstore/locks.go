package graphstore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"

	"github.com/seanb4t/codegraph-go/internal/flock"
	"github.com/seanb4t/codegraph-go/internal/fsatomic"
)

// Two lock files sit beside each store, in its parent directory. Both are
// flock locks, which the kernel releases when their holder exits.
//
// The swap lock, <store>.swap, keeps a store from being replaced under an
// open handle. Every open holds it shared for the life of the handle, and
// ReplaceStore takes it exclusive. Without it, a process that has the
// store open during an exchange keeps the old directory but creates its
// new files by path, so it writes into the new store.
//
// The writer lock, <store>.write, keeps one writer per store at a time. A
// full index holds it for its whole run, and so does a sync. A query does
// not take it, so a query reads the previous graph while a full index
// builds the next one.
const (
	swapLockSuffix   = ".swap"
	writerLockSuffix = ".write"
)

// ErrAfterReplace reports that ReplaceStore put the new store in place and
// then met an error. The store at live is the new one.
var ErrAfterReplace = errors.New("graphstore: the store was replaced, but a later step failed")

// exchangeDirs is the exchange that ReplaceStore calls. A test replaces it
// to fail the exchange.
var exchangeDirs = fsatomic.ExchangeDirs

// lockBeside takes the flock of the lock file beside dir, with the bounded
// retry of retryLocked. A lock that another handle still holds after the
// retries gives ErrStoreLocked. The lock file lives in the parent of dir,
// and lockBeside creates that parent if it is missing.
func lockBeside(dir, suffix string, mode flock.Mode) (*os.File, error) {
	path := filepath.Clean(dir) + suffix
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return retryLocked(func() (*os.File, error) { return flock.Open(path, mode) }, classifyFlockError)
}

// LockWriter takes the writer lock of the store at dir and returns its
// holder. Close the holder to release the lock. If another writer holds
// the lock after the bounded retry, LockWriter returns ErrStoreLocked.
func LockWriter(dir string) (io.Closer, error) {
	return lockBeside(dir, writerLockSuffix, flock.Exclusive)
}

// ReplaceStore puts the store at staging in the place of the store at
// live, in one atomic step. A reader of live sees the old store or the new
// one, never a part of either, and a crash leaves one of the two.
//
// If live exists, whatever it holds, ReplaceStore exchanges the two
// directories, and staging then holds what live held. If live does not
// exist, it renames staging to live. Both paths must be on one
// filesystem. A filesystem that cannot exchange two directories gives an
// error that wraps fsatomic.ErrExchangeUnsupported, and live stays as it
// was.
//
// An error that wraps ErrAfterReplace means that the new store is in place:
// the fsync of a parent directory failed, or the release of a lock did.
// Every other error means that live stays as it was.
//
// ReplaceStore takes the swap lock exclusive, and only then looks for
// live, so no open can create live between the look and the exchange. It
// also takes the Pebble lock of live, because a process of an earlier
// release opens a store without the swap lock. A holder of either lock
// that stays past the bounded retry gives ErrStoreLocked. ReplaceStore
// releases both locks on every return path: Pebble keeps its directory
// locks in a registry of the process, and a lock that leaks there blocks
// every later open of the store until the process ends.
func ReplaceStore(staging, live string) (err error) {
	// replaced turns true once staging is in the place of live. An error
	// after that moment, also one of the deferred releases below, tells
	// the caller so.
	replaced := false
	defer func() {
		if replaced && err != nil {
			err = fmt.Errorf("%w: %w", ErrAfterReplace, err)
		}
	}()
	swap, err := lockBeside(live, swapLockSuffix, flock.Exclusive)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, swap.Close()) }()

	if _, statErr := os.Lstat(live); statErr != nil {
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		err = fsatomic.RenameDir(staging, live)
		replaced = err == nil || errors.Is(err, fsatomic.ErrNotDurable)
		return err
	}
	lock, err := retryLocked(func() (*pebble.Lock, error) { return pebble.LockDirectory(live, vfs.Default) }, classifyOpenError)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	err = exchangeDirs(staging, live)
	replaced = err == nil || errors.Is(err, fsatomic.ErrNotDurable)
	return err
}
