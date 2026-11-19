// Package indexdir locates a repository's index: the .codegraph directory
// below a root and the store inside it, with every symlink resolved. It
// also names the parts of the index that a walk of the repository skips.
// Packages on both sides of the query and indexer boundary import it, so
// it must not import internal/query or internal/indexer.
package indexdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// Name is the directory that holds a repository's index.
const Name = ".codegraph"

// StoreName is the Pebble store's subdirectory under Name.
const StoreName = "store"

// ErrOutsideRoot is the sentinel an *OutsideRootError matches with
// errors.Is.
var ErrOutsideRoot = errors.New("the index leads outside the directory that holds it")

// OutsideRootError reports that Link resolves to Target, which is not Root
// or a descendant of it. Link is the .codegraph directory, or the store
// inside it, joined to the root as the caller spelled it. Link is not
// resolved. Root is the resolved root, and Target is the resolved Link.
type OutsideRootError struct {
	Link, Target, Root string
}

func (e *OutsideRootError) Error() string {
	return fmt.Sprintf("%q leads to %q, outside %q, the directory that holds the index: remove the link or point it inside that directory", e.Link, e.Target, e.Root)
}

// Is reports whether target is ErrOutsideRoot.
func (e *OutsideRootError) Is(target error) bool {
	return target == ErrOutsideRoot
}

// Paths are the resolved locations of an index: its .codegraph directory
// and the store inside it.
type Paths struct {
	Dir, Store string
}

// Resolve resolves root's .codegraph directory and the store inside it.
// It returns an *OutsideRootError when either resolves outside root. A
// path that does not exist yet resolves to where it would be created,
// through any link to a missing target. When the resolved .codegraph
// exists and is not a directory, Store is joined to it without resolving.
func Resolve(root string) (Paths, error) {
	rootReal, dir, err := resolveDir(root)
	if err != nil {
		return Paths{}, err
	}
	index := filepath.Join(root, Name)
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return Paths{Dir: dir, Store: filepath.Join(dir, StoreName)}, nil
	}
	store, err := resolveInside(rootReal, filepath.Join(index, StoreName))
	if err != nil {
		return Paths{}, err
	}
	return Paths{Dir: dir, Store: store}, nil
}

// ResolveDir resolves root's .codegraph directory alone, as the first half
// of Resolve does. It returns an *OutsideRootError when it resolves
// outside root.
func ResolveDir(root string) (string, error) {
	_, dir, err := resolveDir(root)
	return dir, err
}

func resolveDir(root string) (rootReal, dir string, err error) {
	rootReal, err = realpath.Resolve(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve %q: %w", root, err)
	}
	dir, err = resolveInside(rootReal, filepath.Join(root, Name))
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
		return "", &OutsideRootError{Link: link, Target: real, Root: rootReal}
	}
	return real, nil
}
