package extract

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

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
