package query

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// ErrIndexOutsideRoot is the sentinel an *IndexOutsideRootError matches
// with errors.Is.
var ErrIndexOutsideRoot = errors.New("the index leads outside the directory that holds it")

// IndexOutsideRootError reports that Link, a .codegraph directory or the
// store inside it as written below Root, resolves to Target, which is not
// Root or a descendant of it. Root is the resolved directory that holds
// the .codegraph; Link is not resolved; Target is.
type IndexOutsideRootError struct {
	Link, Target, Root string
}

func (e *IndexOutsideRootError) Error() string {
	return fmt.Sprintf("%q leads to %q, outside %q, the directory that holds the index: remove the link or point it inside that directory", e.Link, e.Target, e.Root)
}

// Is reports whether target is ErrIndexOutsideRoot.
func (e *IndexOutsideRootError) Is(target error) bool {
	return target == ErrIndexOutsideRoot
}

// IndexPaths are the resolved locations of an index: its .codegraph
// directory and the store inside it.
type IndexPaths struct {
	Dir, Store string
}

// IndexDirs resolves root's .codegraph directory and the store inside it.
// It returns an *IndexOutsideRootError when either resolves outside root.
// A path that does not exist yet resolves to where it would be created,
// through any link to a missing target. When the resolved .codegraph
// exists and is not a directory, Store is joined to it without resolving.
func IndexDirs(root string) (IndexPaths, error) {
	rootReal, dir, err := indexDir(root)
	if err != nil {
		return IndexPaths{}, err
	}
	index := filepath.Join(root, IndexDirName)
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return IndexPaths{Dir: dir, Store: filepath.Join(dir, StoreDirName)}, nil
	}
	store, err := resolveInside(rootReal, filepath.Join(index, StoreDirName))
	if err != nil {
		return IndexPaths{}, err
	}
	return IndexPaths{Dir: dir, Store: store}, nil
}

// IndexDir resolves root's .codegraph directory alone, as the first half
// of IndexDirs does. It returns an *IndexOutsideRootError when it resolves
// outside root.
func IndexDir(root string) (string, error) {
	_, dir, err := indexDir(root)
	return dir, err
}

func indexDir(root string) (rootReal, dir string, err error) {
	rootReal, err = realpath.Resolve(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve %q: %w", root, err)
	}
	dir, err = resolveInside(rootReal, filepath.Join(root, IndexDirName))
	if err != nil {
		return "", "", err
	}
	return rootReal, dir, nil
}

func resolveInside(rootReal, link string) (string, error) {
	real, err := realpath.Resolve(link)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", link, err)
	}
	if !realpath.Within(rootReal, real) {
		return "", &IndexOutsideRootError{Link: link, Target: real, Root: rootReal}
	}
	return real, nil
}
