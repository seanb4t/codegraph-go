// Package realpathtest draws random trees of directories, files and symlinks
// for property tests of a path resolver. It must not import the package
// under test.
package realpathtest

import (
	"os"
	"path/filepath"

	"pgregory.net/rapid"
)

var names = []string{"a", "b", "c", "d"}

// Tree is a random tree under one or more root directories.
type Tree struct {
	// Dirs holds the roots and every directory made under them.
	Dirs []string
	// Paths holds every path the generator made, plus paths under links
	// and paths that do not exist. A resolver is tried on each one.
	Paths []string
}

// Draw makes a random tree under roots, which must exist and be real
// locations. It makes directories, files, and symlinks. A link target is
// absolute or relative, can hold ".." after a linked directory, can name a
// path in another root, and can name a path in a directory that does not
// exist. Links can form a loop.
func Draw(t *rapid.T, roots ...string) Tree {
	tree := Tree{Dirs: append([]string(nil), roots...)}
	taken := map[string]bool{}
	free := func(label string) (string, bool) {
		p := filepath.Join(rapid.SampledFrom(tree.Dirs).Draw(t, label+" parent"), rapid.SampledFrom(names).Draw(t, label+" name"))
		if taken[p] {
			return "", false
		}
		taken[p] = true
		return p, true
	}

	steps := rapid.IntRange(1, 12).Draw(t, "steps")
	for range steps {
		switch rapid.SampledFrom([]string{"dir", "file", "link", "link"}).Draw(t, "op") {
		case "dir":
			if p, ok := free("dir"); ok {
				must(t, os.Mkdir(p, 0o755))
				tree.Dirs = append(tree.Dirs, p)
				tree.Paths = append(tree.Paths, p)
			}
		case "file":
			if p, ok := free("file"); ok {
				must(t, os.WriteFile(p, []byte("x"), 0o644))
				tree.Paths = append(tree.Paths, p)
			}
		case "link":
			p, ok := free("link")
			if !ok {
				continue
			}
			must(t, os.Symlink(drawTarget(t, tree, filepath.Dir(p)), p))
			tree.Paths = append(tree.Paths, p)
		}
	}

	// Paths below what exists: under a link, and under a missing directory.
	for _, p := range append([]string(nil), tree.Paths...) {
		tree.Paths = append(tree.Paths, filepath.Join(p, "new"), p+string(filepath.Separator)+".."+string(filepath.Separator)+"up")
	}
	return tree
}

// drawTarget draws the target of a link that lives in dir.
func drawTarget(t *rapid.T, tree Tree, dir string) string {
	sep := string(filepath.Separator)
	base := rapid.SampledFrom(tree.Dirs).Draw(t, "target dir")
	name := rapid.SampledFrom(names).Draw(t, "target name")
	switch rapid.SampledFrom([]string{"abs", "rel", "dotdot", "missing"}).Draw(t, "target kind") {
	case "abs":
		return filepath.Join(base, name)
	case "rel":
		rel, err := filepath.Rel(dir, filepath.Join(base, name))
		must(t, err)
		return rel
	case "dotdot":
		// A name in this directory, then "..", then a name. If the first
		// name is a linked directory, ".." leaves through its target.
		return rapid.SampledFrom(names).Draw(t, "via") + sep + ".." + sep + name
	default:
		return filepath.Join(base, "missing", name)
	}
}

func must(t *rapid.T, err error) {
	if err != nil {
		t.Fatalf("realpathtest: %v", err)
	}
}
