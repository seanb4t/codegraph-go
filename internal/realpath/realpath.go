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
func Resolve(path string) (string, error) {
	if path == "" {
		return "", errors.New("realpath: empty path")
	}
	sep := string(filepath.Separator)
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = wd + sep + path
	}
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		return real, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
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
	for len(rest) > 0 {
		el := rest[0]
		rest = rest[1:]
		switch el {
		case "", ".":
			continue
		case "..":
			if missing {
				return "", fmt.Errorf("realpath: %s: \"..\" follows a path element that does not exist", path)
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
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			cur = next
			continue
		}
		links++
		if links > maxLinks {
			return "", fmt.Errorf("realpath: %s: more than %d symlinks to a path that does not exist, possibly a loop", path, maxLinks)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(target) {
			cur = filepath.VolumeName(target) + sep
			target = strings.TrimPrefix(target, cur)
		}
		rest = append(strings.Split(target, sep), rest...)
	}
	return cur, nil
}
