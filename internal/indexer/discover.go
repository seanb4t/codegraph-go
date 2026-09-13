package indexer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// DiscoveredFile is one discovered source file of any registered language,
// the shape Pass 1 (extract) and Pass 2 (resolve) consume.
type DiscoveredFile struct {
	// AbsPath is the absolute path on disk, for reading bytes.
	AbsPath string
	// RelPath is the slash-normalized path relative to the repo root, the
	// file_path stored on Node/File records.
	RelPath string
	// ImportPath is this file's cross-file symbol-index key, computed by
	// its language's LanguageSpec.ModuleKey (D-03/Pitfall 2). For Go this
	// is the module path joined with the file's relative directory (""
	// relative directory means the module root package) — byte-identical
	// to the pre-Phase-5 behavior. A file whose language has no resolvable
	// project descriptor still gets a value here (its LanguageSpec's
	// path-based fallback), never a dropped file.
	ImportPath string
	// Language is the registered LanguageSpec.ID this file's extension
	// resolved to ("go", "java", "python", ...) — the key extract.go's
	// worker pool uses to select the correct parser + extractor per file
	// (Pitfall 1).
	Language string

	// MtimeUnixNs and SizeBytes are the file's on-disk stat info at
	// discovery time (Phase 4 D-01a) — carried through Extract into the
	// committed File record so Sync's stat pre-filter has something cheap
	// to compare against on the next invocation, without hashing every
	// file every sync.
	MtimeUnixNs int64
	SizeBytes   int64
}

// ShouldSkipDir reports whether a directory named name should be excluded
// from traversal — Discover's own WalkDir callback and, per Phase 4 D-04,
// the native filesystem watcher's recursive-add loop both call this exact
// predicate so the two never silently diverge on which paths they cover.
// vendor/ and any dot-prefixed directory (.git, .codegraph, etc.) are
// excluded.
func ShouldSkipDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}

// pendingFile is Discover's own walk-time intermediate, before each
// language's ModuleKey has been computed (that requires the language's
// repo-root descriptor to be resolved first, which happens once per
// language AFTER the walk completes, not per file during it).
type pendingFile struct {
	abs, relPath, language string
	mtimeUnixNs, sizeBytes int64
}

// Discover walks root and returns every file whose extension is claimed by
// a registered LanguageSpec (D-03), sorted by RelPath in ascending byte
// order. It is a thin wrapper around DiscoverAll (Phase 10 D-01,
// discoverexclusion.go) for the many existing callers that need only the
// discovered-file list and module path, not the exclusion-reason list —
// see DiscoverAll's own doc comment for the full walk contract.
func Discover(root string) ([]DiscoveredFile, string, error) {
	d, err := DiscoverAll(root)
	if err != nil {
		return nil, "", err
	}
	return d.Files, d.ModulePath, nil
}

// readModulePath parses root/go.mod and returns its declared module path.
// This is Go's own LanguageSpec.Descriptor implementation (languages_go.go)
// and is also consulted directly by symbolindex.go's store-seeded index.
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
// root package). This is Go's own LanguageSpec.ModuleKey implementation
// (languages_go.go) and is also consulted directly by symbolindex.go's
// store-seeded index.
func importPathFor(modulePath, relPath string) string {
	relDir := path.Dir(relPath)
	if relDir == "." {
		return modulePath
	}
	return modulePath + "/" + relDir
}
