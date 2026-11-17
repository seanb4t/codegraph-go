package query

import (
	"fmt"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/realpath"
)

// IndexOutsideRootError reports that Link, the .codegraph directory of
// the repository at Root or the store inside it, resolves to Target,
// outside Root. Root and Target are resolved paths.
type IndexOutsideRootError struct {
	Link, Target, Root string
}

func (e *IndexOutsideRootError) Error() string {
	return fmt.Sprintf("query: %s leads to %s, outside the repository at %s; codegraph does not open or create an index there", e.Link, e.Target, e.Root)
}

// CheckIndexInside returns an *IndexOutsideRootError when root's
// .codegraph, or the store inside it, resolves outside root. A real
// directory, a symlink whose target stays inside root, and a path that
// does not exist yet all pass. Every command that opens or creates the
// index at root calls it first: query.ResolveIndex for readers, and the
// commands that build the store path themselves for writers.
func CheckIndexInside(root string) error {
	rootReal, err := realpath.Resolve(root)
	if err != nil {
		return err
	}
	index := filepath.Join(root, codegraphDirName)
	for _, p := range []string{index, filepath.Join(index, storeSubdir)} {
		real, err := realpath.Resolve(p)
		if err != nil {
			return err
		}
		if !realpath.Within(rootReal, real) {
			return &IndexOutsideRootError{Link: p, Target: real, Root: rootReal}
		}
	}
	return nil
}
