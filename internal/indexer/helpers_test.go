package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureRoot is the shared Go fixture module the pipeline tests index.
const fixtureRoot = "testdata/gofixture"

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// writeFixtureFile writes contents to root/relPath, creating parent
// directories as needed, failing the test on any error.
func writeFixtureFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}

// writeExactSizeGoFile writes a syntactically valid, build-tag-free Go
// source file at exactly size bytes: a package clause and one trivial
// function (so go/build.Context.MatchFile — decision point 3 — finds no
// import section to scan and stops immediately), padded to the exact
// requested length with a single trailing line comment.
func writeExactSizeGoFile(t *testing.T, path string, size int64) {
	t.Helper()
	header := []byte("package tmp\n\nfunc Placeholder() {}\n\n// ")
	if int64(len(header))+1 > size {
		t.Fatalf("requested size %d too small for header+trailer (%d)", size, len(header)+1)
	}
	buf := make([]byte, size)
	copy(buf, header)
	for i := len(header); i < len(buf)-1; i++ {
		buf[i] = 'x'
	}
	buf[len(buf)-1] = '\n'
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
