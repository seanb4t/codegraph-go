package indexer

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

const fixtureRoot = "testdata/gofixture"

// TestDiscover_Fixture asserts Discover returns the fixture's .go files in a
// stable, deterministic RelPath-sorted order, with skip_linux.go included
// or excluded depending on the current build context's GOOS (Pitfall 5:
// MatchFile is consulted per-file with its own parent directory).
func TestDiscover_Fixture(t *testing.T) {
	files, modulePath, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if modulePath != "example.com/gofixture" {
		t.Fatalf("modulePath = %q, want %q", modulePath, "example.com/gofixture")
	}

	want := []string{"main.go", "pkga/embed.go", "pkga/pkga.go", "pkgb/pkgb.go"}
	if runtime.GOOS == "linux" {
		want = append(want, "skip_linux.go")
	}
	sort.Strings(want)

	var got []string
	for _, f := range files {
		got = append(got, f.RelPath)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RelPaths = %v, want %v", got, want)
	}
}

// TestDiscover_Deterministic asserts repeated calls over the same tree
// yield byte-identical results, independent of filesystem walk order.
func TestDiscover_Deterministic(t *testing.T) {
	first, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	second, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Discover is not deterministic:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// TestDiscover_ImportPaths asserts each DiscoveredFile carries the correct
// module-path + relative-dir import path.
func TestDiscover_ImportPaths(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	want := map[string]string{
		"main.go":       "example.com/gofixture",
		"pkga/embed.go": "example.com/gofixture/pkga",
		"pkga/pkga.go":  "example.com/gofixture/pkga",
		"pkgb/pkgb.go":  "example.com/gofixture/pkgb",
	}
	got := make(map[string]string, len(files))
	for _, f := range files {
		got[f.RelPath] = f.ImportPath
	}
	for relPath, wantImportPath := range want {
		if got[relPath] != wantImportPath {
			t.Errorf("ImportPath(%s) = %q, want %q", relPath, got[relPath], wantImportPath)
		}
	}
}

// TestDiscover_SkipsVendorAndDotDirs asserts files under vendor/ or any
// dot-prefixed directory are never returned.
func TestDiscover_SkipsVendorAndDotDirs(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/tmp\n\ngo 1.26\n")
	mustWrite(t, filepath.Join(root, "real.go"), "package tmp\n")
	mustWrite(t, filepath.Join(root, "vendor", "ignored.go"), "package vendored\n")
	mustWrite(t, filepath.Join(root, ".hidden", "x.go"), "package hidden\n")

	files, _, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	var got []string
	for _, f := range files {
		got = append(got, f.RelPath)
	}
	want := []string{"real.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RelPaths = %v, want %v (vendor/.hidden must be excluded)", got, want)
	}
}

// TestDiscover_MissingGoMod asserts a directory with no go.mod returns a
// clear error rather than panicking.
func TestDiscover_MissingGoMod(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package main\n")

	_, _, err := Discover(root)
	if err == nil {
		t.Fatal("Discover: expected error for missing go.mod, got nil")
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
