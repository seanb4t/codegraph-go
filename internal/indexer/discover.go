package indexer

import (
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

// DiscoveredFile is one discovered Go source file, the shape Pass 1
// (extract) and Pass 2 (resolve) consume.
type DiscoveredFile struct {
	// AbsPath is the absolute path on disk, for reading bytes.
	AbsPath string
	// RelPath is the slash-normalized path relative to the repo root, the
	// file_path stored on Node/File records.
	RelPath string
	// ImportPath is the module path joined with the file's relative
	// directory ("" relative directory means the module root package).
	ImportPath string
}

// Discover walks root and returns every *.go file the default build
// context includes, sorted by RelPath in ascending byte order. This
// stable order is determinism's first line of defense: the same input
// tree always yields the same output order, regardless of filesystem
// walk order.
//
// vendor/ directories and any dot-prefixed directory (.git, .codegraph,
// etc.) are skipped entirely. A *.go candidate is included iff
// go/build.Context.MatchFile reports it belongs to the default build
// context (GOOS/GOARCH, build tags) — the same primitive the go toolchain
// itself uses, so no compile-error-free module resolution is required and
// no //go:build/+build parsing is hand-rolled.
//
// Discover also parses root/go.mod (via golang.org/x/mod/modfile) once to
// resolve the module's base import path, returned alongside the files. A
// root with no go.mod is an error.
func Discover(root string) ([]DiscoveredFile, string, error) {
	modulePath, err := readModulePath(root)
	if err != nil {
		return nil, "", err
	}

	ctx := build.Default
	var files []DiscoveredFile

	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) != ".go" {
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}

		// Pitfall 5: MatchFile must be given the file's OWN parent
		// directory, never a hoisted/cached value, or build-tag
		// evaluation silently mis-fires.
		match, err := ctx.MatchFile(filepath.Dir(abs), filepath.Base(abs))
		if err != nil {
			return err
		}
		if !match {
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		files = append(files, DiscoveredFile{
			AbsPath:    abs,
			RelPath:    relPath,
			ImportPath: importPathFor(modulePath, relPath),
		})
		return nil
	})
	if walkErr != nil {
		return nil, "", walkErr
	}

	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, modulePath, nil
}

// readModulePath parses root/go.mod and returns its declared module path.
func readModulePath(root string) (string, error) {
	goModPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("indexer: reading %s: %w", goModPath, err)
	}

	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("indexer: parsing %s: %w", goModPath, err)
	}
	if f.Module == nil {
		return "", fmt.Errorf("indexer: %s has no module directive", goModPath)
	}
	return f.Module.Mod.Path, nil
}

// importPathFor computes a file's Go import path from the module's base
// import path and the file's slash-normalized path relative to the module
// root (Pattern 5: modulePath + "/" + relDir; modulePath alone for the
// root package).
func importPathFor(modulePath, relPath string) string {
	relDir := path.Dir(relPath)
	if relDir == "." {
		return modulePath
	}
	return modulePath + "/" + relDir
}
