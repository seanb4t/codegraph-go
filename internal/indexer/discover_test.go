package indexer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
)

const fixtureRoot = "testdata/gofixture"
const mixedLangFixtureRoot = "testdata/mixedlangfixture"

// init registers one test-only LanguageSpec — "python" (Descriptor is nil
// entirely, simulating a language with no descriptor hook at all) — solely
// to prove Discover's extension->language-registry generalization (D-03)
// mechanically, without depending on a real Python extractor landing in a
// later Wave-B plan. NewParser/Extract are never invoked by any test in
// this file (Discover never parses file content), so they are inert stubs.
//
// A matching "java" test-only spec previously lived here (05-02) to prove
// the erroring-Descriptor fallback shape; it was removed once a REAL "java"
// LanguageSpec landed (languages_java.go, 05-04) — registerLanguage keys
// its registry/extToLang maps by a single string ID, so a second "java"
// registration here would silently collide with (and, depending on Go's
// package-file init order, potentially override) the real one. The
// erroring-Descriptor fallback shape this used to prove is now exercised
// against the real java spec directly, in
// TestDiscover_MixedLanguage_DescriptorAbsentFallback below (the
// mixedlangfixture root has no pom.xml/build.gradle, so
// javaextract's own Descriptor genuinely errors the same way).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "python",
		Extensions: []string{".py"},
		NewParser: func() (parser.Parser, error) {
			return nil, fmt.Errorf("discover_test: python parser not implemented")
		},
		Extract: func(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
			return goextract.FileResult{}, fmt.Errorf("discover_test: python extractor not implemented")
		},
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// D-03 path-identity fallback, same discipline as "java"
			// above, exercised via a nil Descriptor hook instead of an
			// erroring one — both shapes must degrade the same way.
			return path.Dir(relPath)
		},
		// Descriptor intentionally left nil: this language has no
		// project-descriptor hook implementation at all yet.
	})
}

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
		if f.Language != "go" {
			t.Errorf("Language(%s) = %q, want %q", f.RelPath, f.Language, "go")
		}
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
// module-path + relative-dir import path — Go behavior byte-identical to
// pre-Phase-5, now routed through the LanguageSpec.ModuleKey hook instead
// of being computed inline.
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

// TestDiscover_MissingGoMod_FallsBackToPathIdentity asserts a root with no
// go.mod no longer hard-fails Discover (D-03's relaxation of the
// pre-Phase-5 all-or-nothing readModulePath error): the Go file is still
// discovered, with Go's own nil-descriptor ModuleKey fallback (bare
// relPath, languages_go.go) rather than a module-path-joined import path.
func TestDiscover_MissingGoMod_FallsBackToPathIdentity(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package main\n")

	files, modulePath, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: unexpected error for missing go.mod: %v", err)
	}
	if modulePath != "" {
		t.Errorf("modulePath = %q, want \"\" (no go.mod to resolve)", modulePath)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1 (descriptor-absent file must not be dropped)", len(files))
	}
	if files[0].RelPath != "a.go" {
		t.Fatalf("RelPath = %q, want %q", files[0].RelPath, "a.go")
	}
	if files[0].Language != "go" {
		t.Fatalf("Language = %q, want %q", files[0].Language, "go")
	}
	if files[0].ImportPath != "a.go" {
		t.Fatalf("ImportPath = %q, want %q (Go's own nil-descriptor path fallback)", files[0].ImportPath, "a.go")
	}
}

// TestDiscover_MixedLanguage_ExtensionRegistry asserts a mixed-tree fixture
// (.go + .java + .py) yields one DiscoveredFile per supported extension,
// each carrying the correct Language resolved via lookupLanguageByExt, and
// that an unsupported extension (.md, .json) is never discovered.
func TestDiscover_MixedLanguage_ExtensionRegistry(t *testing.T) {
	files, modulePath, err := Discover(mixedLangFixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if modulePath != "example.com/mixed" {
		t.Fatalf("modulePath = %q, want %q", modulePath, "example.com/mixed")
	}

	got := make(map[string]string, len(files))
	for _, f := range files {
		got[f.RelPath] = f.Language
	}

	want := map[string]string{
		"main.go":          "go",
		"sub/Greeter.java": "java",
		"app.py":           "python",
	}
	if len(got) != len(want) {
		t.Fatalf("discovered %d files %v, want exactly %v (unsupported extensions must be excluded)", len(got), got, want)
	}
	for relPath, wantLang := range want {
		if got[relPath] != wantLang {
			t.Errorf("Language(%s) = %q, want %q", relPath, got[relPath], wantLang)
		}
	}
	for relPath := range got {
		if relPath == "README.md" || relPath == "config.json" {
			t.Errorf("unsupported extension %s was discovered, must be excluded", relPath)
		}
	}
}

// TestDiscover_MixedLanguage_DescriptorAbsentFallback asserts a file whose
// language has no resolvable project descriptor (here: "java" — the real
// javaextract LanguageSpec's own Descriptor genuinely errors against
// mixedlangfixture, which has no pom.xml/build.gradle — and "python", whose
// Descriptor hook is nil entirely) is still returned with a path-based
// ModuleKey — never dropped (D-03's central guarantee).
func TestDiscover_MixedLanguage_DescriptorAbsentFallback(t *testing.T) {
	files, _, err := Discover(mixedLangFixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	byRelPath := make(map[string]DiscoveredFile, len(files))
	for _, f := range files {
		byRelPath[f.RelPath] = f
	}

	javaFile, ok := byRelPath["sub/Greeter.java"]
	if !ok {
		t.Fatal("sub/Greeter.java was dropped, want present with path-based fallback identity")
	}
	if want := "sub/Greeter.java"; javaFile.ImportPath != want {
		t.Errorf("java ImportPath = %q, want %q (path-based fallback: real javaextract Descriptor errors on no pom.xml/build.gradle)", javaFile.ImportPath, want)
	}

	pyFile, ok := byRelPath["app.py"]
	if !ok {
		t.Fatal("app.py was dropped, want present with path-based fallback identity")
	}
	if want := "."; pyFile.ImportPath != want {
		t.Errorf("python ImportPath = %q, want %q (path-based fallback: nil Descriptor hook)", pyFile.ImportPath, want)
	}
}

// TestDiscover_SortedByRelPath asserts Discover's output is always sorted
// ascending by RelPath, using a fixture whose files are named so that
// neither directory-lexical walk order nor declaration order already
// happens to match the required RelPath order — the sort.Slice call itself
// must be doing the work, not incidental filesystem ordering.
func TestDiscover_SortedByRelPath(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/shuffled\n\ngo 1.26\n")
	mustWrite(t, filepath.Join(root, "zzz.go"), "package shuffled\n")
	mustWrite(t, filepath.Join(root, "aaa", "yyy.go"), "package aaa\n")
	mustWrite(t, filepath.Join(root, "mmm.go"), "package shuffled\n")
	mustWrite(t, filepath.Join(root, "bbb", "xxx.go"), "package bbb\n")

	files, _, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	var got []string
	for _, f := range files {
		got = append(got, f.RelPath)
	}

	want := make([]string, len(got))
	copy(want, got)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RelPaths = %v, not sorted ascending (want %v)", got, want)
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
