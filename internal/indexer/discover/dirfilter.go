package discover

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// DirFilter is the directory rule of a walk under one repository root. It
// skips the directories that ShouldSkipDir names, and the directories that
// hold the root's index and its store, whatever their names. A zero
// DirFilter skips by name only.
type DirFilter struct {
	root  string
	index []string
}

// NewDirFilter returns the DirFilter of root. It resolves the root's
// .codegraph directory and store with indexdir.Resolve, and returns its
// error. An index directory that a name rule already skips adds nothing.
func NewDirFilter(root string) (DirFilter, error) {
	paths, err := indexdir.Resolve(root)
	if err != nil {
		return DirFilter{}, fmt.Errorf("discover: resolve the index of %s: %w", root, err)
	}
	rootReal, err := realpath.Resolve(root)
	if err != nil {
		return DirFilter{}, fmt.Errorf("discover: resolve %s: %w", root, err)
	}
	f := DirFilter{root: root}
	for _, dir := range []string{paths.Dir, paths.Store} {
		rel, err := filepath.Rel(rootReal, dir)
		if err != nil || rel == "." || nameSkipped(rel) || slices.Contains(f.index, rel) {
			continue
		}
		f.index = append(f.index, rel)
	}
	return f, nil
}

// nameSkipped reports whether a directory of rel names one that
// ShouldSkipDir skips, so a walk never reaches rel.
func nameSkipped(rel string) bool {
	return slices.ContainsFunc(strings.Split(rel, string(filepath.Separator)), ShouldSkipDir)
}

// SkipDir reports whether a walk skips the directory at path, a path that
// the walk reached from the root of f. The root itself is not checked here.
func (f DirFilter) SkipDir(path string) bool {
	return ShouldSkipDir(filepath.Base(path)) || f.isIndex(path)
}

// isIndex reports whether path is a directory that holds the index. A
// path that differs from an index directory only in letter case counts
// when it opens the same directory, as it does on a case-insensitive file
// system.
func (f DirFilter) isIndex(path string) bool {
	if len(f.index) == 0 {
		return false
	}
	rel, err := filepath.Rel(f.root, path)
	if err != nil {
		return false
	}
	for _, ix := range f.index {
		if rel == ix || (strings.EqualFold(rel, ix) && sameFile(path, filepath.Join(f.root, ix))) {
			return true
		}
	}
	return false
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && os.SameFile(ai, bi)
}
