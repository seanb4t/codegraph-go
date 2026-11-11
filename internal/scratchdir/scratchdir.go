// Package scratchdir makes temporary directories that a later run can
// clean up safely. The owner of a directory holds an exclusive flock on a
// lock file inside it for as long as the directory exists. Sweep removes
// the directories whose lock it can take: the remains of a process that
// stopped. It never removes the directory of a live owner.
package scratchdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/flock"
	"github.com/seanb4t/codegraph-go/internal/fsatomic"
)

const (
	// LockName is the name of the lock file inside a directory.
	LockName = ".lock"

	// PendingMaxAge is how long Sweep leaves a pending directory that has
	// no lock file. New takes far less time than this to lock a directory
	// it just made, so an older one belongs to a process that stopped.
	PendingMaxAge = time.Minute

	// tmpLockName is the name under which New creates and locks the lock
	// file, before the file gets the name that Sweep looks for.
	tmpLockName = ".lock.tmp"
)

// Dir is a temporary directory and the lock that its owner holds.
type Dir struct {
	path string
	lock *os.File
}

// checkPrefix returns an error for a prefix that New and Sweep cannot
// work with. A prefix must end in "-": PendingPrefix replaces that dash,
// so a pending name never starts with the prefix. An empty prefix would
// also make Sweep take every directory of the parent for its own.
func checkPrefix(prefix string) error {
	if len(prefix) < 2 || !strings.HasSuffix(prefix, "-") {
		return fmt.Errorf("scratchdir: the prefix %q must have a name and end in \"-\"", prefix)
	}
	return nil
}

// PendingPrefix is the prefix of a directory of prefix that New did not
// finish yet. It never has prefix itself as a prefix, so a pending
// directory does not look like a finished one. That holds because prefix
// ends in "-", which New and Sweep check.
func PendingPrefix(prefix string) string {
	return strings.TrimSuffix(prefix, "-") + ".pending-"
}

// New creates the directory <parent>/<prefix><random> and returns it with
// its lock held. With gitignore set, the directory holds a .gitignore that
// hides everything in it from git.
//
// New builds the directory under a pending name. It creates the lock file
// under a temporary name, locks it, and only then gives it its name. So a
// lock file that has its name always had its lock, and Sweep cannot take
// the lock of a directory that New still builds. The directory gets its
// final name last.
func New(parent, prefix string, gitignore bool) (*Dir, error) {
	if err := checkPrefix(prefix); err != nil {
		return nil, err
	}
	pending, err := os.MkdirTemp(parent, PendingPrefix(prefix)+"*")
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Dir, error) {
		os.RemoveAll(pending)
		return nil, err
	}
	if err := os.Chmod(pending, 0o755); err != nil {
		return fail(err)
	}
	if gitignore {
		if err := os.WriteFile(filepath.Join(pending, ".gitignore"), []byte("*\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	lock, err := flock.Create(filepath.Join(pending, tmpLockName), flock.Exclusive)
	if err != nil {
		return fail(err)
	}
	if err := os.Rename(filepath.Join(pending, tmpLockName), filepath.Join(pending, LockName)); err != nil {
		lock.Close()
		return fail(err)
	}
	suffix := strings.TrimPrefix(filepath.Base(pending), PendingPrefix(prefix))
	path := filepath.Join(parent, prefix+suffix)
	if err := fsatomic.RenameDir(pending, path); err != nil {
		lock.Close()
		return fail(err)
	}
	return &Dir{path: path, lock: lock}, nil
}

// Path is the path of the directory.
func (d *Dir) Path() string {
	return d.path
}

// Remove deletes the directory, then releases its lock.
func (d *Dir) Remove() error {
	err := os.RemoveAll(d.path)
	return errors.Join(err, d.lock.Close())
}

// Unlock releases the lock and leaves the directory. A caller that moves
// the directory to a permanent place calls it after the move.
func (d *Dir) Unlock() error {
	return d.lock.Close()
}

// Sweep removes each leftover directory of prefix under parent. It never
// creates a lock file.
//
//   - A directory whose lock file Sweep can lock goes away. Its owner
//     stopped.
//   - A directory whose lock another handle holds stays.
//   - A finished directory with no lock file goes away. New names the lock
//     file before the directory, so this is an owner that a kill stopped
//     during its own Remove.
//   - A pending directory with no lock file is an owner in its first
//     instants, or one that a kill stopped then. It goes away once it is
//     older than PendingMaxAge.
func Sweep(parent, prefix string) {
	if checkPrefix(prefix) != nil {
		return
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	pendingPfx := PendingPrefix(prefix)
	for _, e := range entries {
		pending := strings.HasPrefix(e.Name(), pendingPfx)
		if !e.IsDir() || (!pending && !strings.HasPrefix(e.Name(), prefix)) {
			continue
		}
		path := filepath.Join(parent, e.Name())
		lock, err := flock.OpenExisting(filepath.Join(path, LockName), flock.Exclusive)
		switch {
		case err == nil:
			_ = (&Dir{path: path, lock: lock}).Remove()
		case !errors.Is(err, os.ErrNotExist):
			// Another handle holds the lock, or the file cannot be read.
		case !pending:
			_ = os.RemoveAll(path)
		default:
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > PendingMaxAge {
				_ = os.RemoveAll(path)
			}
		}
	}
}
