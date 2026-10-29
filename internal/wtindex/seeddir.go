package wtindex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	indexDirName = ".codegraph"
	storeDirName = "store"
	seedPrefix   = ".codegraph.seed-"
	lockName     = ".lock"
)

// ErrIndexExists is returned by place when the root already holds an index.
// The seed directory is gone by then.
var ErrIndexExists = errors.New("wtindex: the worktree already has an index")

// seedDir is a temporary index directory under a worktree root. The build
// holds an exclusive lock on its lock file for as long as the directory
// exists, so sweep never removes a build in progress.
type seedDir struct {
	path string
	lock *os.File
}

// newSeedDir creates <root>/.codegraph.seed-<random>/. Its .gitignore is the
// first file in it, so git never sees the directory, and its lock is taken
// before it holds anything else.
func newSeedDir(root string) (*seedDir, error) {
	path, err := os.MkdirTemp(root, seedPrefix+"*")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(path, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	lock, err := lockFile(filepath.Join(path, lockName))
	if err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	return &seedDir{path: path, lock: lock}, nil
}

func (d *seedDir) storePath() string {
	return filepath.Join(d.path, storeDirName)
}

// remove deletes the directory, then releases its lock.
func (d *seedDir) remove() error {
	err := os.RemoveAll(d.path)
	return errors.Join(err, d.lock.Close())
}

// sweep removes each seed directory under root whose lock it can take: the
// remains of a build whose process stopped. A directory whose lock another
// build holds is left alone.
func sweep(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), seedPrefix) {
			continue
		}
		path := filepath.Join(root, e.Name())
		lock, err := lockFile(filepath.Join(path, lockName))
		if err != nil {
			continue
		}
		_ = (&seedDir{path: path, lock: lock}).remove()
	}
}

// place renames d to <root>/.codegraph. os.Rename refuses a target that is
// an existing directory, so when root already holds an index, place
// removes d and returns ErrIndexExists. Any other rename failure removes d
// and returns the error.
func place(d *seedDir, root string) error {
	target := filepath.Join(root, indexDirName)
	if err := os.Rename(d.path, target); err != nil {
		_ = d.remove()
		if _, statErr := os.Lstat(target); statErr == nil {
			return ErrIndexExists
		}
		return fmt.Errorf("wtindex: placing the index: %w", err)
	}
	os.Remove(filepath.Join(target, lockName))
	return d.lock.Close()
}
