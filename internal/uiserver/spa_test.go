package uiserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/web"
)

// TestSPAServesEmbeddedIndexAtRoot is Criterion-1's tracer proof (BLD-02):
// a GET / against the handler newSPAHandler returns must serve the real,
// embedded SvelteKit index.html byte-for-byte — closing Phase 1's UAT gap
// G-01-1 (GET / -> 404).
func TestSPAServesEmbeddedIndexAtRoot(t *testing.T) {
	buildFS, err := fs.Sub(web.BuildFS, spaSubdirName)
	if err != nil {
		t.Fatalf("fs.Sub(web.BuildFS, %q): %v", spaSubdirName, err)
	}

	handler := newSPAHandler(buildFS)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want prefix %q", contentType, "text/html")
	}

	want, err := fs.ReadFile(buildFS, "index.html")
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", "index.html", err)
	}

	got := rec.Body.Bytes()
	if string(got) != string(want) {
		t.Fatalf("GET / body does not match embedded index.html byte-for-byte (got %d bytes, want %d bytes)", len(got), len(want))
	}

	// An empty or placeholder file must not pass: the body must actually
	// be SvelteKit's real compiled shell, not merely "some bytes".
	if len(got) == 0 {
		t.Fatal("GET / body is empty")
	}
	const bootstrapMarker = "kit.start(app, element)"
	if !strings.Contains(string(got), bootstrapMarker) {
		t.Fatalf("GET / body does not contain the SvelteKit client bootstrap marker %q — got:\n%s", bootstrapMarker, got)
	}
}

// spaPathSetDiff reports, given an expected and an actual set of
// relative build-tree paths, which paths are missing (present in
// expected, absent from actual) and which are orphaned (present in
// actual, never expected). Adapted in shape from
// internal/mcp/resources_schema_drift_test.go's resourceStemSetDiff.
// Both returned slices are sorted for a stable failure message and are
// both empty exactly when the two sets agree — asserting only one
// direction would miss an orphaned embedded file, which is drift too.
func spaPathSetDiff(expected, actual []string) (missing, orphaned []string) {
	expectedSet := make(map[string]bool, len(expected))
	for _, s := range expected {
		expectedSet[s] = true
	}
	actualSet := make(map[string]bool, len(actual))
	for _, s := range actual {
		actualSet[s] = true
	}
	for s := range expectedSet {
		if !actualSet[s] {
			missing = append(missing, s)
		}
	}
	for s := range actualSet {
		if !expectedSet[s] {
			orphaned = append(orphaned, s)
		}
	}
	sort.Strings(missing)
	sort.Strings(orphaned)
	return missing, orphaned
}

// onDiskBuildTreePaths walks web/build/ on disk (relative to this test
// file's own package directory — internal/uiserver/../../web/build,
// following claude_skillpackage_test.go's established
// filepath.Join("..", "..", ...) convention for reaching the repository
// root from a package two levels down) and returns every regular file's
// path relative to that root, e.g. "index.html", "_app/immutable/...".
func onDiskBuildTreePaths(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..", "web", "build")
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk on-disk build tree %s: %v", root, err)
	}
	return paths
}

// embeddedBuildTreePaths walks web.BuildFS's build/ subtree (via
// fs.Sub, so paths come back relative to the build root, matching
// onDiskBuildTreePaths's rooting exactly once — never by string-trimming
// the "build/" prefix) and returns every regular file's path.
func embeddedBuildTreePaths(t *testing.T) []string {
	t.Helper()
	buildFS, err := fs.Sub(web.BuildFS, spaSubdirName)
	if err != nil {
		t.Fatalf("fs.Sub(web.BuildFS, %q): %v", spaSubdirName, err)
	}
	var paths []string
	err = fs.WalkDir(buildFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded build tree: %v", err)
	}
	return paths
}

// TestEmbeddedFSMatchesOnDiskBuildTree is ROADMAP criterion 1's actual
// verification: the set of file paths inside the embedded web.BuildFS
// must be exactly the set of file paths on disk under web/build/,
// _app/-prefixed entries included — not merely "the build succeeded".
// This goes RED by construction if //go:embed all:build in web/embed.go
// is ever weakened to //go:embed build (dropping the "all:" prefix):
// Go's default embed walk excludes dot/underscore-prefixed paths, and
// SvelteKit's entire hashed-asset tree lives under the
// underscore-prefixed build/_app/ directory, so every _app/-prefixed
// path would show up in `missing`.
func TestEmbeddedFSMatchesOnDiskBuildTree(t *testing.T) {
	onDisk := onDiskBuildTreePaths(t)
	embedded := embeddedBuildTreePaths(t)

	missing, orphaned := spaPathSetDiff(onDisk, embedded)
	if len(missing) != 0 {
		t.Errorf("paths present on disk but missing from web.BuildFS: %v", missing)
	}
	if len(orphaned) != 0 {
		t.Errorf("paths present in web.BuildFS but not on disk: %v", orphaned)
	}
}

// TestEmbeddedBuildTreeIsNonTrivial is the guard-the-guard for
// TestEmbeddedFSMatchesOnDiskBuildTree: without it, two empty walks
// (e.g. against an accidentally-empty web/build/) would agree with each
// other and the diff test above would pass vacuously. It asserts the
// on-disk walk actually found files, and specifically found at least one
// path under the immutable-asset prefix — the exact subtree an "all:"-less
// embed directive would silently drop.
func TestEmbeddedBuildTreeIsNonTrivial(t *testing.T) {
	onDisk := onDiskBuildTreePaths(t)
	if len(onDisk) == 0 {
		t.Fatal("on-disk web/build/ walk found zero files — TestEmbeddedFSMatchesOnDiskBuildTree would pass vacuously against an empty tree")
	}

	foundImmutableAsset := false
	for _, p := range onDisk {
		if strings.HasPrefix(p, spaAssetPrefix) {
			foundImmutableAsset = true
			break
		}
	}
	if !foundImmutableAsset {
		t.Fatalf("on-disk web/build/ walk found no path under %q — got: %v", spaAssetPrefix, onDisk)
	}
}
