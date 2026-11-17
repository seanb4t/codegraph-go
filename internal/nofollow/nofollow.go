//go:build !windows

// Package nofollow opens files without following a symbolic link in the
// last element of the path, and finds symbolic links in a directory.
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
var ErrSymlink = errors.New("a file in the index is a symbolic link")

// SymlinkError reports that Link, a file that codegraph opens in the index,
// is a symbolic link to Target.
type SymlinkError struct {
	Link, Target string
}

func (e *SymlinkError) Error() string {
	return fmt.Sprintf("%q is a symbolic link to %q, and codegraph does not follow links inside the index: remove the link", e.Link, e.Target)
}

// Is reports whether target is ErrSymlink.
func (e *SymlinkError) Is(target error) bool {
	return target == ErrSymlink
}

// OpenFile is os.OpenFile with O_NOFOLLOW. When the last element of path is
// a symbolic link, it returns a *SymlinkError.
func OpenFile(path string, flag int, perm fs.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag|unix.O_NOFOLLOW, perm)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			if target, linkErr := os.Readlink(path); linkErr == nil {
				return nil, &SymlinkError{Link: path, Target: target}
			}
		}
		return nil, err
	}
	return f, nil
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
// symbolic link. A missing dir has none.
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
		target, err := os.Readlink(link)
		if err != nil {
			return err
		}
		return &SymlinkError{Link: link, Target: target}
	}
	return nil
}
