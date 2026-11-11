package graphstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/pebble/v2"

	"github.com/seanb4t/codegraph-go/internal/flock"
)

// handleState is the state of a handle in the table of shared handles.
type handleState int

const (
	// handleOpening means that one caller is in pebble.Open for the handle.
	handleOpening handleState = iota
	// handleReady means that the handle is open and can take more users.
	handleReady
	// handleClosing means that the last user is closing the handle. The
	// Pebble lock is still held, so the handle takes no new user.
	handleClosing
)

// dbHandle owns one open Pebble database and the swap lock of its store.
// Every pebbleStore points at one dbHandle.
//
// A handle from Open, OpenExisting or Checkpoint is in the table of shared
// handles, and every open of the same store in this process uses it. A
// handle from OpenStaging is outside the table and has one user.
type dbHandle struct {
	db      *pebble.DB
	swap    *os.File
	staging bool

	// key is the key of the handle in the table. It is empty for a handle
	// outside the table.
	key string

	// The fields below are guarded by sharedHandles.mu.
	state handleState
	users int
	// opened is closed when the open of the handle ends, and openErr then
	// holds the error of that open.
	opened  chan struct{}
	openErr error
}

// sharedHandles is the table of shared handles of this process. The key is
// the absolute path of the store.
//
// Pebble allows one open of a store directory at a time, also inside one
// process. The table lets a tool call and a sync of one process use the
// store at the same time: the second open takes the handle of the first.
// The process closes a handle when its last user closes it, so another
// process and ReplaceStore get the store as they do without the table.
var sharedHandles struct {
	mu      sync.Mutex
	entries map[string]*dbHandle
}

// sharedOpenHook, if not nil, runs while a handle is in the opening state.
// sharedCloseHook, if not nil, runs while a handle is in the closing
// state, before the Pebble close. sharedWaitHook, if not nil, runs two
// times in a caller that waits for the open of another caller: with false
// before the wait, and with true after that open ended and before the
// caller looks at the handle again. A test sets them to hold that moment.
var (
	sharedOpenHook  func()
	sharedCloseHook func()
	sharedWaitHook  func(woke bool)
)

// errHandleClosing is the error of an attempt that meets a handle in the
// closing state. It wraps ErrStoreLocked, because the Pebble lock of this
// process is still held, and the caller retries as for any held lock.
var errHandleClosing = fmt.Errorf("%w: this process is closing the store", ErrStoreLocked)

// alreadyClassified is the classifier of retryLocked for an attempt that
// classifies its own errors at the call that made them.
func alreadyClassified(err error) error { return err }

// openShared returns the shared handle of the store at dir with one more
// user, and opens the store if this process does not have it open.
//
// It is one retryLocked loop, so a caller has one retry budget for every
// lock that the open can meet: the swap lock, the Pebble lock of another
// holder, and a handle of this process in the closing state.
func openShared(dir string) (*dbHandle, error) {
	key, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return retryLocked(func() (*dbHandle, error) { return acquireShared(key, dir) }, alreadyClassified)
}

// acquireShared is one attempt of openShared.
func acquireShared(key, dir string) (*dbHandle, error) {
	sharedHandles.mu.Lock()
	h := sharedHandles.entries[key]
	if h == nil {
		h = &dbHandle{key: key, state: handleOpening, opened: make(chan struct{})}
		if sharedHandles.entries == nil {
			sharedHandles.entries = make(map[string]*dbHandle)
		}
		sharedHandles.entries[key] = h
		sharedHandles.mu.Unlock()

		// The table lock does not cover the open, so an open of another
		// store does not wait for this one.
		err := h.open(dir)

		sharedHandles.mu.Lock()
		if err != nil {
			h.openErr = err
			delete(sharedHandles.entries, key)
		} else {
			h.state = handleReady
			h.users = 1
		}
		sharedHandles.mu.Unlock()
		close(h.opened)
		if err != nil {
			return nil, err
		}
		return h, nil
	}

	if h.state == handleOpening {
		sharedHandles.mu.Unlock()
		if sharedWaitHook != nil {
			sharedWaitHook(false)
		}
		<-h.opened
		if sharedWaitHook != nil {
			sharedWaitHook(true)
		}
		sharedHandles.mu.Lock()
	}
	defer sharedHandles.mu.Unlock()
	switch {
	case h.openErr != nil:
		return nil, h.openErr
	case h.state != handleReady:
		// The handle is closing. A caller that waited for the open can
		// also arrive here, when the opener closed the handle first.
		return nil, errHandleClosing
	}
	h.users++
	return h, nil
}

// open takes the shared swap lock of the store at dir and opens Pebble, one
// time each. It classifies a held lock as ErrStoreLocked.
//
// The swap lock comes first, before pebble.Open and before any directory
// of the store exists: ReplaceStore looks for the store under the
// exclusive swap lock, and that look is only right if no open can create
// the store meanwhile.
func (h *dbHandle) open(dir string) error {
	swap, err := lockBesideOnce(dir, swapLockSuffix, flock.Shared)
	if err != nil {
		return err
	}
	if sharedOpenHook != nil && h.key != "" {
		sharedOpenHook()
	}
	db, err := pebble.Open(dir, &pebble.Options{Logger: quietLogger{}, DisableWAL: h.staging})
	if err != nil {
		return errors.Join(classifyOpenError(err), swap.Close())
	}
	h.db, h.swap = db, swap
	return nil
}

// release removes one user of the handle. It closes the handle when that
// user was the last one.
func (h *dbHandle) release() error {
	if h.key != "" {
		sharedHandles.mu.Lock()
		h.users--
		if h.users > 0 {
			sharedHandles.mu.Unlock()
			return nil
		}
		h.state = handleClosing
		sharedHandles.mu.Unlock()

		// The handle leaves the table on every return path, also when the
		// Pebble close returns an error. A handle that stays in the
		// closing state keeps every later open of the store out.
		defer func() {
			sharedHandles.mu.Lock()
			delete(sharedHandles.entries, h.key)
			sharedHandles.mu.Unlock()
		}()
		if sharedCloseHook != nil {
			sharedCloseHook()
		}
	}

	// A staging store has no write-ahead log, so its memtables are
	// flushed before the close.
	var flushErr error
	if h.staging {
		flushErr = h.db.Flush()
	}
	return errors.Join(flushErr, h.db.Close(), h.swap.Close())
}
