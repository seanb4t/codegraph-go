package daemon

import (
	"errors"
	"fmt"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
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
