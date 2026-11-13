// Package realpath answers two questions about a path: where the operating
// system reaches for it, and whether that location is inside a directory.
// It imports only the standard library.
package realpath

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxLinks bounds the links that Resolve reads on its own walk, so that a
// loop is an error and not a hang.
const maxLinks = 8

// Within reports whether path is root itself or a descendant of it. Both
// must be absolute.
func Within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Resolve returns the location that the operating system reaches for path,
// with every symlink resolved. A relative path is taken from the working
// directory. Unlike filepath.EvalSymlinks, it also resolves a path whose
// last parts do not exist yet, for example a link to a file that the caller
// is about to create.
//
// Resolve never cleans a path as text. ".." after a link names the parent
// of the link's target, and a relative link target starts at the real
// directory of the link.
//
// Resolve suits a directory that the caller makes with os.MkdirAll. For a
// file that the caller makes, use ResolveFile.
func Resolve(path string) (string, error) {
	real, _, err := resolve(path)
	return real, err
}

// ResolveFile is Resolve for a path where the caller makes a regular file.
// It returns an error for a missing location that the path names with a
// trailing separator or a trailing ".". The operating system accepts only
// a directory that exists there, so no file can be made through the path.
func ResolveFile(path string) (string, error) {
	real, dirOnly, err := resolve(path)
	if err != nil {
		return "", err
	}
	if dirOnly {
		return "", fmt.Errorf("realpath: %s: a trailing separator names a directory that does not exist", path)
	}
	return real, nil
}

// resolve is the walk behind Resolve and ResolveFile. dirOnly reports a
// missing location that the path names as a directory.
func resolve(path string) (real string, dirOnly bool, err error) {
	if path == "" {
		return "", false, errors.New("realpath: empty path")
	}
	sep := string(filepath.Separator)
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", false, err
		}
		path = wd + sep + path
	}
	real, err = filepath.EvalSymlinks(path)
	if err == nil {
		return real, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", false, err
	}

	// Some part of path does not exist. Start at the longest leading part
	// that the operating system resolves, then walk the rest one element at
	// a time.
	root := filepath.VolumeName(path) + sep
	elems := strings.Split(strings.TrimPrefix(path, root), sep)
	cur, rest := root, elems
	for i := len(elems) - 1; i > 0; i-- {
		if real, err := filepath.EvalSymlinks(root + strings.Join(elems[:i], sep)); err == nil {
			cur, rest = real, elems[i:]
			break
		}
	}

	links := 0
	missing := false
	// needsDir is set when the path, or the target of its last link, ends
	// with a separator or with ".". The operating system then accepts only
	// a directory that exists.
	needsDir := false
	for len(rest) > 0 {
		el := rest[0]
		rest = rest[1:]
		switch el {
		case "", ".":
			needsDir = needsDir || len(rest) == 0
			continue
		case "..":
			if missing {
				return "", false, fmt.Errorf("realpath: %s: \"..\" follows a path element that does not exist", path)
			}
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, el)
		if missing {
			cur = next
			continue
		}
		info, err := os.Lstat(next)
		if errors.Is(err, fs.ErrNotExist) {
			missing = true
			cur = next
			continue
		}
		if err != nil {
			return "", false, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			cur = next
			continue
		}
		links++
		if links > maxLinks {
			return "", false, fmt.Errorf("realpath: %s: more than %d symlinks to a path that does not exist, possibly a loop", path, maxLinks)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", false, err
		}
		if filepath.IsAbs(target) {
			cur = filepath.VolumeName(target) + sep
			target = strings.TrimPrefix(target, cur)
		}
		rest = append(strings.Split(target, sep), rest...)
	}
	return cur, missing && needsDir, nil
}
