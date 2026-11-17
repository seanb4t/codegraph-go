//go:build !windows

// Package nofollow opens files without following a symbolic link in the
// last element of the path, and finds symbolic links among the entries of
// a directory.
package nofollow

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// ErrSymlink is the sentinel a *SymlinkError matches with errors.Is.
var ErrSymlink = errors.New("the file is a symbolic link")

// SymlinkError reports that Link is a symbolic link to Target. Target is
// empty when the link could not be read.
type SymlinkError struct {
	Link, Target string
}

func (e *SymlinkError) Error() string {
	if e.Target == "" {
		return fmt.Sprintf("%q is or goes through a symbolic link, and codegraph does not follow it here: remove the link", e.Link)
	}
	return fmt.Sprintf("%q is a symbolic link to %q, and codegraph does not follow it here: remove the link", e.Link, e.Target)
}

// Is reports whether target is ErrSymlink.
func (e *SymlinkError) Is(target error) bool {
	return target == ErrSymlink
}

// OpenFile is os.OpenFile with O_NOFOLLOW. When the last element of path is
// a symbolic link, it returns a *SymlinkError: the open fails with ELOOP,
// or with EEXIST when flag holds O_CREATE and O_EXCL. A path whose
// directories hold a loop of links also fails with ELOOP and gives a
// *SymlinkError. The directories of path are followed.
func OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag|unix.O_NOFOLLOW, perm)
	if err == nil {
		return f, nil
	}
	switch {
	case errors.Is(err, unix.ELOOP):
		return nil, symlinkError(path)
	case errors.Is(err, fs.ErrExist) && flag&os.O_EXCL != 0:
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&fs.ModeSymlink != 0 {
			return nil, symlinkError(path)
		}
	}
	return nil, err
}

// WriteFile is os.WriteFile through OpenFile.
func WriteFile(path string, data []byte, perm fs.FileMode) error {
	f, err := OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Close())
}

// CheckDir returns a *SymlinkError for the first entry of dir that is a
// symbolic link. A missing dir has none. It reads dir once, at the time of
// the call, and looks only at its entries, not into subdirectories. It
// follows dir itself when dir is a link. An entry that is gone by the time
// CheckDir looks at it again is skipped.
func CheckDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink == 0 {
			continue
		}
		link := filepath.Join(dir, e.Name())
		if _, err := os.Lstat(link); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return symlinkError(link)
	}
	return nil
}

func symlinkError(path string) *SymlinkError {
	target, _ := os.Readlink(path)
	return &SymlinkError{Link: path, Target: target}
}
