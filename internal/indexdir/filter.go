package indexdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/realpath"
	"github.com/seanb4t/codegraph-go/internal/scratchdir"
)

// The entries that codegraph keeps beside a store, in the parent directory
// of the store. Each name is the base name of the store and a suffix.
const (
	// SwapLockSuffix names the swap lock file, <store>.swap.
	SwapLockSuffix = ".swap"
	// WriterLockSuffix names the writer lock file, <store>.write.
	WriterLockSuffix = ".write"
	// ScratchSuffix starts the names of the scratch directories of a full
	// index, <store>.scratch-<random>. A scratch directory that is still
	// being made is named <store>.scratch.pending-<random>, after
	// scratchdir.PendingPrefix. The suffix ends in "-", as scratchdir
	// requires of a prefix.
	ScratchSuffix = ".scratch-"
)

// SkipDirName is the name rule of every walk of a repository: it reports
// whether the walk skips a directory named name. It skips vendor/ and every
// dot-prefixed directory, such as .git and .codegraph.
func SkipDirName(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}

// Filter tells a walk under one repository root which paths belong to the
// index of that root: the .codegraph directory, the store, and the lock
// files and scratch directories beside the store. A link can give each of
// them a name that no name rule of the walk skips. A zero Filter skips
// nothing.
type Filter struct {
	root string
	// dirs are the .codegraph directory and the store, relative to the
	// resolved root.
	dirs []string
	// parent is the parent directory of the store, relative to the
	// resolved root, and store is the base name of the store.
	parent, store string
}

// NewFilter returns the Filter of root. Pass root as the walk will spell
// it: Skip compares the paths of the walk to root as text. The error wraps
// the error of Resolve, such as an *OutsideRootError, or the error of
// resolving root itself.
func NewFilter(root string) (Filter, error) {
	paths, err := Resolve(root)
	if err != nil {
		return Filter{}, fmt.Errorf("resolve the index of %s: %w", root, err)
	}
	rootReal, err := realpath.Resolve(root)
	if err != nil {
		return Filter{}, fmt.Errorf("resolve %s: %w", root, err)
	}
	f := Filter{root: root}
	for _, dir := range []string{paths.Dir, paths.Store} {
		if rel, err := filepath.Rel(rootReal, dir); err == nil && rel != "." {
			f.dirs = append(f.dirs, rel)
		}
	}
	if rel, err := filepath.Rel(rootReal, filepath.Dir(paths.Store)); err == nil {
		f.parent, f.store = rel, filepath.Base(paths.Store)
	}
	return f, nil
}

// Skip reports whether path, a path under the root of f, is the top path of
// a part of the index: the .codegraph directory, the store, or an entry
// beside the store. It does not match a path below one of them, so a walk
// must prune each path that Skip matches. A name that differs from the
// index only in letter case matches when both name the same file, as on a
// case-insensitive file system.
func (f Filter) Skip(path string) bool {
	if f.root == "" {
		return false
	}
	rel, err := filepath.Rel(f.root, path)
	if err != nil {
		return false
	}
	for _, dir := range f.dirs {
		if f.same(path, rel, dir) {
			return true
		}
	}
	return f.store != "" && besideStore(f.store, filepath.Base(rel)) &&
		f.same(filepath.Dir(path), filepath.Dir(rel), f.parent)
}

// same reports whether path, at rel under the root of f, is the entry at
// want under that root.
func (f Filter) same(path, rel, want string) bool {
	if rel == want {
		return true
	}
	return strings.EqualFold(rel, want) && sameFile(path, filepath.Join(f.root, want))
}

// besideStore reports whether name is a lock file or a scratch directory
// of the store named store.
func besideStore(store, name string) bool {
	if name == store+SwapLockSuffix || name == store+WriterLockSuffix {
		return true
	}
	scratch := store + ScratchSuffix
	return strings.HasPrefix(name, scratch) || strings.HasPrefix(name, scratchdir.PendingPrefix(scratch))
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}
