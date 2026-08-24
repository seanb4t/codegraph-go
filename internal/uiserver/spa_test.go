package uiserver

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
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

// newTestSPAHandler builds a handler over the real embedded build tree
// (never a synthetic in-memory FS — 02-02's rule is only meaningful
// against the actual committed SvelteKit output) and returns both the
// handler and the sub-filesystem it serves from, so a test can read an
// expected file's bytes independently of the handler under test.
func newTestSPAHandler(t *testing.T) (http.Handler, fs.FS) {
	t.Helper()
	buildFS, err := fs.Sub(web.BuildFS, spaSubdirName)
	if err != nil {
		t.Fatalf("fs.Sub(web.BuildFS, %q): %v", spaSubdirName, err)
	}
	return newSPAHandler(buildFS), buildFS
}

// findImmutableAssetPath walks the embedded build tree under
// spaAssetPrefix and returns the first regular file path it finds — never
// a hardcoded content hash, since the hash changes on every dependency
// bump (task instruction).
func findImmutableAssetPath(t *testing.T, buildFS fs.FS) string {
	t.Helper()
	root := strings.TrimSuffix(spaAssetPrefix, "/")
	var found string
	err := fs.WalkDir(buildFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && found == "" {
			found = p
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %q under embedded build tree: %v", root, err)
	}
	if found == "" {
		t.Fatalf("no file found under %q in embedded build tree", spaAssetPrefix)
	}
	return found
}

// parseCSP splits a Content-Security-Policy header value into its
// directive -> source-list form, so a test can assert on parsed values
// rather than substring-matching the whole policy string (a substring
// match would pass on a policy that also carried a wildcard source).
func parseCSP(t *testing.T, policy string) map[string][]string {
	t.Helper()
	directives := make(map[string][]string)
	for _, part := range strings.Split(policy, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		directives[fields[0]] = fields[1:]
	}
	return directives
}

// TestSPAClientRouteFallsBackToIndex is D-10's fallback half: any path
// that is not a real file resolves to the embedded index.html, including
// a path segment shaped like a source file name — the exact case the
// rejected "any path with a dot is an asset" heuristic would have broken
// (Phase 3's deep links carry dotted file paths).
func TestSPAClientRouteFallsBackToIndex(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)
	want, err := fs.ReadFile(buildFS, spaFallbackFile)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", spaFallbackFile, err)
	}

	// Deliberately NOT t.Run subtests: this project's own PLAN-level
	// verify command counts exactly one "--- PASS: <TestName>" line per
	// named test via substring grep, and a subtest line
	// ("--- PASS: TestName/case") contains that same substring, which
	// would double- or triple-count. A single top-level PASS covering
	// both cases is what the verify command actually requires.
	cases := []struct {
		name string
		path string
	}{
		{"deep client route", "/browse/some/deep/route"},
		{"dotted path segment (D-10's rejected dot-heuristic)", "/browse/internal/uiserver/spa.go"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: GET %s status = %d, want %d; body: %s", tc.name, tc.path, rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Body.Bytes(); string(got) != string(want) {
			t.Fatalf("%s: GET %s body does not match embedded index.html byte-for-byte (got %d bytes, want %d bytes)", tc.name, tc.path, len(got), len(want))
		}
	}
}

// TestSPAImmutableAssetMissReturns404 is D-10's teeth: a miss under the
// immutable-asset prefix must 404, never fall back to index.html — serving
// HTML for a missing hashed chunk produces a browser MIME/module error
// that points nowhere near the real cause.
func TestSPAImmutableAssetMissReturns404(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)
	fallback, err := fs.ReadFile(buildFS, spaFallbackFile)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", spaFallbackFile, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+spaAssetPrefix+"does-not-exist.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if got := rec.Body.Bytes(); string(got) == string(fallback) {
		t.Fatal("404 body equals embedded index.html — a missing hashed asset must not arrive as HTML")
	}
}

// TestSPAImmutableAssetHitIsCachedImmutable proves D-11's long-lived
// class: a real path under the immutable prefix (discovered by walking
// the embedded FS, never a hardcoded hash) is served with a Cache-Control
// carrying "immutable" and a max-age of at least one year.
func TestSPAImmutableAssetHitIsCachedImmutable(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)
	assetPath := findImmutableAssetPath(t, buildFS)

	req := httptest.NewRequest(http.MethodGet, "/"+assetPath, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /%s status = %d, want %d; body: %s", assetPath, rec.Code, http.StatusOK, rec.Body.String())
	}
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Fatalf("GET /%s Cache-Control = %q, want it to contain %q", assetPath, cc, "immutable")
	}
	if !strings.Contains(cc, "max-age=31536000") {
		t.Fatalf("GET /%s Cache-Control = %q, want it to contain a max-age of at least one year (31536000 seconds)", assetPath, cc)
	}
}

// TestSPAIndexIsNotCached proves D-11's no-store class: index.html itself
// must never be cacheable, or a binary upgrade could boot a cached old
// shell against a new RPC surface.
func TestSPAIndexIsNotCached(t *testing.T) {
	handler, _ := newTestSPAHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != spaCacheControlNoStore {
		t.Fatalf("GET / Cache-Control = %q, want %q", got, spaCacheControlNoStore)
	}
}

// spaRequestShape is one row of the request-shape table
// TestSPASetsNoSniffOnEveryResponse and TestSPASetsCSPOnEveryResponse
// share, so a new branch that forgets a header fails the table rather
// than a hand-duplicated case list.
type spaRequestShape struct {
	name   string
	method string
	path   string
}

// spaBaseRequestShapes returns the four request shapes
// TestSPASetsNoSniffOnEveryResponse and TestSPASetsCSPOnEveryResponse are
// table-driven over: client-route fallback, immutable-prefix miss,
// immutable-prefix hit, and index.
func spaBaseRequestShapes(t *testing.T, buildFS fs.FS) []spaRequestShape {
	t.Helper()
	assetPath := findImmutableAssetPath(t, buildFS)
	return []spaRequestShape{
		{"client route fallback", http.MethodGet, "/browse/some/deep/route"},
		{"immutable asset miss", http.MethodGet, "/" + spaAssetPrefix + "does-not-exist.js"},
		{"immutable asset hit", http.MethodGet, "/" + assetPath},
		{"index", http.MethodGet, "/"},
	}
}

// TestSPASetsNoSniffOnEveryResponse is T-02-02-01's regression test:
// X-Content-Type-Options: nosniff must be set unconditionally, before any
// routing branch, so no branch can forget it. Deliberately NOT t.Run
// subtests — see TestSPAClientRouteFallsBackToIndex's comment on why a
// single top-level PASS line is what this project's verify command
// requires.
func TestSPASetsNoSniffOnEveryResponse(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)
	for _, tc := range spaBaseRequestShapes(t, buildFS) {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s: %s %s: X-Content-Type-Options = %q, want %q", tc.name, tc.method, tc.path, got, "nosniff")
		}
	}
}

// TestSPASetsCSPOnEveryResponse is T-02-02-06's regression test, extended
// beyond the base four shapes to include the 405 (non-read method): every
// response this handler emits carries a non-empty Content-Security-Policy
// whose parsed default-src and connect-src are each exactly 'self'.
// Deliberately NOT t.Run subtests — see TestSPAClientRouteFallsBackToIndex's
// comment.
func TestSPASetsCSPOnEveryResponse(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)
	shapes := spaBaseRequestShapes(t, buildFS)
	shapes = append(shapes, spaRequestShape{"non-read method", http.MethodPost, "/"})

	for _, tc := range shapes {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s: %s %s: X-Content-Type-Options = %q, want %q (nosniff must survive on the 405 path too)", tc.name, tc.method, tc.path, got, "nosniff")
		}

		policy := rec.Header().Get("Content-Security-Policy")
		if policy == "" {
			t.Fatalf("%s: %s %s: Content-Security-Policy is empty", tc.name, tc.method, tc.path)
		}
		directives := parseCSP(t, policy)
		if got := directives["default-src"]; len(got) != 1 || got[0] != "'self'" {
			t.Fatalf("%s: %s %s: default-src = %v, want exactly ['self']", tc.name, tc.method, tc.path, got)
		}
		if got := directives["connect-src"]; len(got) != 1 || got[0] != "'self'" {
			t.Fatalf("%s: %s %s: connect-src = %v, want exactly ['self']", tc.name, tc.method, tc.path, got)
		}
	}
}

// cspForbidsUnsafe reports whether policy contains neither 'unsafe-eval'
// in any directive nor 'unsafe-inline' in script-src — the predicate
// TestSPACSPForbidsUnsafeDirectives asserts against both the real served
// policy and, as a positive control, a deliberately unsafe one.
func cspForbidsUnsafe(t *testing.T, policy string) bool {
	t.Helper()
	directives := parseCSP(t, policy)
	for _, sources := range directives {
		for _, s := range sources {
			if s == "'unsafe-eval'" {
				return false
			}
		}
	}
	for _, s := range directives["script-src"] {
		if s == "'unsafe-inline'" {
			return false
		}
	}
	return true
}

// TestSPACSPForbidsUnsafeDirectives is T-02-02-07's regression test for a
// future "make the CSP stop complaining" edit. It carries its own positive
// control: a deliberately-constructed policy carrying 'unsafe-eval' or
// script-src 'unsafe-inline' IS rejected by the same predicate, so the
// check cannot pass by never examining anything.
func TestSPACSPForbidsUnsafeDirectives(t *testing.T) {
	handler, _ := newTestSPAHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	policy := rec.Header().Get("Content-Security-Policy")
	if !cspForbidsUnsafe(t, policy) {
		t.Fatalf("served policy %q contains 'unsafe-eval' or script-src 'unsafe-inline'", policy)
	}

	// Positive control: these deliberately-unsafe policies MUST be
	// rejected, proving the predicate actually examines its input.
	if cspForbidsUnsafe(t, "default-src 'self'; script-src 'self' 'unsafe-eval'") {
		t.Fatal("positive control failed: a policy carrying 'unsafe-eval' was NOT rejected — the predicate examines nothing")
	}
	if cspForbidsUnsafe(t, "default-src 'self'; script-src 'self' 'unsafe-inline'") {
		t.Fatal("positive control failed: a policy carrying script-src 'unsafe-inline' was NOT rejected — the predicate examines nothing")
	}
}

// TestSPACSPHashesCoverEmbeddedInlineScripts is the guard-the-guard for
// TestSPASetsCSPOnEveryResponse's default-src 'self' assertion: it derives
// the expected hash sources INDEPENDENTLY from the embedded index.html
// (never from the handler's own policy) and requires every one to appear
// in the served script-src, with the extracted count asserted greater
// than zero so this cannot pass by finding nothing.
func TestSPACSPHashesCoverEmbeddedInlineScripts(t *testing.T) {
	_, buildFS := newTestSPAHandler(t)
	indexHTML, err := fs.ReadFile(buildFS, spaFallbackFile)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", spaFallbackFile, err)
	}

	wantScriptHashes, _ := spaInlineBlockHashes(indexHTML)
	if len(wantScriptHashes) == 0 {
		t.Fatal("extracted zero inline <script> blocks from the embedded index.html — the extraction broke, or SvelteKit's adapter-static fallback page no longer carries a bootstrap script")
	}

	handler := newSPAHandler(buildFS)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	policy := rec.Header().Get("Content-Security-Policy")
	directives := parseCSP(t, policy)
	gotScriptSrc := directives["script-src"]

	for _, want := range wantScriptHashes {
		found := false
		for _, got := range gotScriptSrc {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("script-src %v does not carry expected hash source %q (derived independently from the embedded index.html)", gotScriptSrc, want)
		}
	}

	wantCount := len(wantScriptHashes) + 1 // +1 for 'self'
	if len(gotScriptSrc) != wantCount {
		t.Fatalf("script-src has %d sources %v, want exactly %d ('self' plus one sha256 source per extracted block)", len(gotScriptSrc), gotScriptSrc, wantCount)
	}
}

// TestSPARejectsNonReadMethods proves the method gate: a POST is rejected
// with 405 and an Allow header naming the two admitted methods.
func TestSPARejectsNonReadMethods(t *testing.T) {
	handler, _ := newTestSPAHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST / status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	allow := rec.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodHead) {
		t.Fatalf("Allow header = %q, want it to name %s and %s", allow, http.MethodGet, http.MethodHead)
	}
}

// TestSPAServesRootStaticFileWithoutFallback proves the middle branch of
// D-10's rule exists and is not swallowed by the fallback: a real embedded
// file that is neither under the immutable prefix nor index.html is served
// as itself.
func TestSPAServesRootStaticFileWithoutFallback(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)

	const staticPath = "robots.txt"
	want, err := fs.ReadFile(buildFS, staticPath)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v — this test requires a real static file outside the immutable prefix and outside index.html", staticPath, err)
	}
	fallback, err := fs.ReadFile(buildFS, spaFallbackFile)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", spaFallbackFile, err)
	}
	if string(want) == string(fallback) {
		t.Fatalf("%q is byte-identical to %q — this test cannot distinguish the middle branch from the fallback", staticPath, spaFallbackFile)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+staticPath, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /%s status = %d, want %d; body: %s", staticPath, rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Body.Bytes(); string(got) != string(want) {
		t.Fatalf("GET /%s body does not match the embedded file byte-for-byte", staticPath)
	}
	if got := rec.Header().Get("Cache-Control"); got != spaCacheControlDefault {
		t.Fatalf("GET /%s Cache-Control = %q, want %q", staticPath, got, spaCacheControlDefault)
	}
}

// TestSPARPCPathReachesConnectHandler proves D-09's precedence in BOTH
// directions against a REAL running server (not a bare mux): a Connect
// JSON POST to GetStatus reaches the Connect handler and gets a JSON
// Connect response, never the SPA index.html; and, on the same server
// instance, a GET on a client-side route returns the SPA index.html. The
// RPC target uses the repository under test as-is (deliberately NOT
// indexed) — GetStatus answers even in Phase 1's D-16 degraded state,
// which is still a Connect response and still discriminates correctly.
func TestSPARPCPathReachesConnectHandler(t *testing.T) {
	_, buildFS := newTestSPAHandler(t)
	fallback, err := fs.ReadFile(buildFS, spaFallbackFile)
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, %q): %v", spaFallbackFile, err)
	}

	dir := t.TempDir()
	srv := mustListen(t, dir)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ctx)
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	// The Connect protocol's unary mode is plain HTTP POST with a JSON
	// body and Content-Type: application/json — no generated client
	// needed, and no procedure-path literal written here: the path comes
	// from uiv1connect's own exported constant (task instruction).
	rpcResp, err := http.Post(srv.URL()+uiv1connect.UIServiceGetStatusProcedure, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST %s: %v", uiv1connect.UIServiceGetStatusProcedure, err)
	}
	defer rpcResp.Body.Close()
	rpcBody, err := io.ReadAll(rpcResp.Body)
	if err != nil {
		t.Fatalf("read RPC response body: %v", err)
	}

	if rpcResp.StatusCode != http.StatusOK {
		t.Fatalf("GetStatus status = %d, want %d; body: %s", rpcResp.StatusCode, http.StatusOK, rpcBody)
	}
	contentType := rpcResp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("GetStatus response Content-Type = %q, want prefix %q — the SHARPEST single discriminator between the SPA handler and the Connect handler", contentType, "application/json")
	}
	if bytes.Equal(rpcBody, fallback) {
		t.Fatal("GetStatus response body equals the SPA index.html — the RPC path was answered by the SPA handler, not the Connect handler")
	}

	// The converse, against the SAME server instance: a GET on a
	// client-side route returns the SPA index.html — proving both
	// handlers coexist on one mux rather than that one of them is simply
	// absent.
	getResp, err := http.Get(srv.URL() + "/browse/some/deep/route")
	if err != nil {
		t.Fatalf("GET /browse/some/deep/route: %v", err)
	}
	defer getResp.Body.Close()
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, fallback) {
		t.Fatal("GET /browse/some/deep/route did not return the SPA index.html on the same server instance that correctly answered the RPC path")
	}
}

// TestSPAInheritsOriginHostGuard is the regression test for "the SPA
// handler was mounted outside the guard": a client-side route request
// carrying a foreign Host is rejected by originHostGuard before the SPA
// handler ever runs — the same rejection Phase 1 already proves for RPC
// paths.
func TestSPAInheritsOriginHostGuard(t *testing.T) {
	dir := t.TempDir()

	srv := mustListen(t, dir)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ctx)
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	req, err := http.NewRequest(http.MethodGet, srv.URL()+"/browse/some/deep/route", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Host = "evil.com"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /browse/some/deep/route with Host: evil.com: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d — the SPA handler must not be reachable with a foreign Host", resp.StatusCode, http.StatusForbidden)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if bytes.Contains(body, []byte("kit.start")) {
		t.Fatal("response body looks like the SPA index.html bootstrap — the SPA handler was reached despite the foreign Host, meaning it is mounted outside originHostGuard")
	}
}
