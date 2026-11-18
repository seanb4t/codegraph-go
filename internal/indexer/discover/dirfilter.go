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
// it returns the zero DirFilter and the error of indexdir.NewFilter, which
// wraps an *indexdir.OutsideRootError for an index outside root.
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

// SkipFile reports whether a walk skips the file at path, a path under the
// root of f: a lock file beside the store, or another part of the index.
func (f DirFilter) SkipFile(path string) bool {
	return f.index.Skip(path)
}
