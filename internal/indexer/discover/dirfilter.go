package discover

import (
	"fmt"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
)

// DirFilter is the path rule of a walk under one repository root. It skips
// the directories that ShouldSkipDir names, and every part of the root's
// index that indexdir.Filter names, whatever its name. A zero DirFilter
// skips directories by name only.
type DirFilter struct {
	index indexdir.Filter
}

// NewDirFilter returns the DirFilter of root. Pass root as the walk will
// spell it: the filter compares the paths of the walk to root. On an error
// it returns the zero DirFilter and an error that wraps the error of
// indexdir.NewFilter, such as an *indexdir.OutsideRootError for an index
// outside root.
func NewDirFilter(root string) (DirFilter, error) {
	index, err := indexdir.NewFilter(root)
	if err != nil {
		return DirFilter{}, fmt.Errorf("discover: %w", err)
	}
	return DirFilter{index: index}, nil
}

// SkipDir reports whether a walk skips the directory at path, a path that
// the walk reached from the root of f. A walk does not call it for the
// root, which the name rule would skip when its name starts with a dot.
func (f DirFilter) SkipDir(path string) bool {
	return ShouldSkipDir(filepath.Base(path)) || f.index.Skip(path)
}

// SkipFile reports whether path, a path under the root of f, is a part of
// the index, such as a lock file beside the store. It applies no name rule,
// so a watcher can call it for an event path of any kind.
func (f DirFilter) SkipFile(path string) bool {
	return f.index.Skip(path)
}
