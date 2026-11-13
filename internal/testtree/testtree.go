// Package testtree writes a tree of source files for a test that indexes
// it. It holds no index step: each package that uses it indexes the tree
// its own way.
package testtree

import (
	"os"
	"path/filepath"
	"testing"
)

// Write writes files under a new temporary directory and returns that
// directory. A key is a slash-separated path relative to the directory,
// and its value is the content of the file. Write makes the parent
// directories of each file.
func Write(t testing.TB, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}
