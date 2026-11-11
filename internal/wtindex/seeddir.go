package wtindex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/scratchdir"
)

const (
	indexDirName = ".codegraph"
	storeDirName = "store"
	seedPrefix   = ".codegraph.seed-"
	lockName     = scratchdir.LockName
)

// ErrIndexExists is returned by place when the root already holds an index.
// The seed directory is gone by then.
var ErrIndexExists = errors.New("wtindex: the worktree already has an index")

// seedDir is a temporary index directory under a worktree root. The build
// holds the lock of its scratch directory for as long as the directory
// exists, so a sweep never removes a build in progress.
type seedDir struct {
	*scratchdir.Dir
}

// newSeedDir creates <root>/.codegraph.seed-<random>/ with its lock held.
// It asks for the .gitignore, because the directory sits in the worktree
// root, where git would otherwise see it.
func newSeedDir(root string) (*seedDir, error) {
	d, err := scratchdir.New(root, seedPrefix, true)
	if err != nil {
		return nil, err
	}
	return &seedDir{d}, nil
}

func (d *seedDir) storePath() string {
	return filepath.Join(d.Path(), storeDirName)
}

// remove deletes the directory, then releases its lock.
func (d *seedDir) remove() error {
	return d.Remove()
}

// sweep removes each seed directory under root that no build holds.
func sweep(root string) {
	scratchdir.Sweep(root, seedPrefix)
}

// place renames d to <root>/.codegraph. os.Rename refuses a target that is
// an existing directory, so when root already holds an index, place
// removes d and returns ErrIndexExists. Any other rename failure removes d
// and returns the error.
func place(d *seedDir, root string) error {
	target := filepath.Join(root, indexDirName)
	if err := os.Rename(d.Path(), target); err != nil {
		_ = d.remove()
		if _, statErr := os.Lstat(target); statErr == nil {
			return ErrIndexExists
		}
		return fmt.Errorf("wtindex: placing the index: %w", err)
	}
	os.Remove(filepath.Join(target, lockName))
	return d.Unlock()
}
