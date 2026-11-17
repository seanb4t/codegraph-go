package graphstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

// ErrNotStore is the sentinel a *NotStoreError matches with errors.Is.
var ErrNotStore = errors.New("graphstore: the directory holds files but no store")

// NotStoreError reports that Dir, the directory of a store, holds files
// but no store.
type NotStoreError struct {
	Dir string
}

func (e *NotStoreError) Error() string {
	return fmt.Sprintf("%q holds files but no codegraph store, so codegraph does not use it as the store: if a link leads the store there, remove the link", e.Dir)
}

// Is reports whether target is ErrNotStore.
func (e *NotStoreError) Is(target error) bool {
	return target == ErrNotStore
}

// CheckLiveStore reports whether a store can be opened or replaced at dir.
// A missing dir passes, and so does a dir that is empty, holds only a
// LOCK file, or holds a store. A dir with an entry that is a symbolic link
// gives a *nofollow.SymlinkError. A dir that holds other files and no
// store gives a *NotStoreError.
//
// The check reads dir once, at the time of the call. It looks only at the
// entries of dir, and it follows dir itself when dir is a link.
func CheckLiveStore(dir string) error {
	if err := nofollow.CheckDir(dir); err != nil {
		if errors.Is(err, nofollow.ErrSymlink) {
			return err
		}
		return fmt.Errorf("graphstore: check the store %s: %w", dir, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("graphstore: check the store %s: %w", dir, err)
	}
	empty := true
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), manifestMarkerPrefix) {
			return nil
		}
		if e.Name() != "LOCK" {
			empty = false
		}
	}
	if empty {
		return nil
	}
	return &NotStoreError{Dir: dir}
}
