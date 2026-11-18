package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// ErrIndexChanged is the sentinel an *IndexChangedError matches with
// errors.Is.
var ErrIndexChanged = errors.New("the index resolves to other paths than when the watcher started")

// IndexChangedError reports that the index directory or its store now
// resolves to other paths than Was, the paths the watcher started with.
type IndexChangedError struct {
	Was, Now query.IndexPaths
}

func (e *IndexChangedError) Error() string {
	var changes []string
	if e.Was.Dir != e.Now.Dir {
		changes = append(changes, fmt.Sprintf("the index directory %q is now %q", e.Was.Dir, e.Now.Dir))
	}
	if e.Was.Store != e.Now.Store {
		changes = append(changes, fmt.Sprintf("the store %q is now %q", e.Was.Store, e.Now.Store))
	}
	return strings.Join(changes, ", and ") + ": the watcher wrote nothing, restart the watcher"
}

// Is reports whether target is ErrIndexChanged.
func (e *IndexChangedError) Is(target error) bool {
	return target == ErrIndexChanged
}

// releaseOwnLock removes the lock file of a stopping watcher (#375). A
// pull can change .codegraph while the watcher runs, so it resolves
// .codegraph again and removes daemon.lock only when .codegraph resolves
// inside the repository and the lock names this process. In every other
// case it leaves the file and returns an error that names it. A directory
// other than the start one is accepted: when .codegraph moved to a link
// inside the repository, the watcher's own lock moved with it.
func (d *Daemon) releaseOwnLock() error {
	rootReal, err := realpath.Resolve(d.repoRoot)
	if err != nil {
		return err
	}
	dir, err := realpath.Resolve(filepath.Join(d.repoRoot, codegraphDirName))
	if err != nil {
		return err
	}
	if !realpath.Within(rootReal, dir) {
		return fmt.Errorf("left %s in place: %s now leads outside %s", lockPath(dir), filepath.Join(d.repoRoot, codegraphDirName), rootReal)
	}
	info, ok, err := readLock(dir)
	if err != nil || !ok {
		return err
	}
	if info.PID != os.Getpid() {
		return fmt.Errorf("left %s in place: it names pid %d, not this process", lockPath(dir), info.PID)
	}
	return release(dir)
}

// checkIndex resolves the index paths again. It returns the error of
// query.IndexDirs unchanged, an *IndexChangedError when the paths differ
// from d.start, or nil.
func (d *Daemon) checkIndex() error {
	now, err := query.IndexDirs(d.repoRoot)
	if err != nil {
		return err
	}
	if now != d.start {
		return &IndexChangedError{Was: d.start, Now: now}
	}
	return nil
}
