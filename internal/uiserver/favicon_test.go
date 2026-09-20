package uiserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFaviconSVGContent verifies FIX-02's favicon correctness:
// GET /favicon.svg returns 200 with SVG content-type and a body containing
// the brand fill, correct viewBox, and expected path count. Critically, it
// must NOT contain "svelte" (case-insensitive) — the Svelte scaffold icon
// this fix replaced.
func TestFaviconSVGContent(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)

	// Read the expected favicon content to validate against
	favicon, err := fs.ReadFile(buildFS, "favicon.svg")
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, \"favicon.svg\"): %v", err)
	}

	// GET /favicon.svg
	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// (1) Status must be 200
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /favicon.svg status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// (2) Content-Type must be SVG
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "image/svg") {
		t.Fatalf("GET /favicon.svg Content-Type = %q, want to contain %q", contentType, "image/svg")
	}

	body := rec.Body.Bytes()

	// (3) Body must match embedded favicon byte-for-byte
	if string(body) != string(favicon) {
		t.Fatalf("GET /favicon.svg body does not match embedded favicon.svg byte-for-byte (got %d bytes, want %d bytes)", len(body), len(favicon))
	}

	// (4) Body must contain viewBox="0 0 2048 2048"
	if !strings.Contains(string(body), `viewBox="0 0 2048 2048"`) {
		t.Fatalf("GET /favicon.svg body does not contain viewBox=\"0 0 2048 2048\"")
	}

	// (5) Body must contain the brand fill rgb(29,78,216)
	if !strings.Contains(string(body), `fill="rgb(29,78,216)"`) {
		t.Fatalf("GET /favicon.svg body does not contain brand fill rgb(29,78,216)")
	}

	// (6) Body must contain at least 7 <path elements
	pathCount := strings.Count(string(body), "<path")
	if pathCount < 7 {
		t.Fatalf("GET /favicon.svg body has %d <path elements, want at least 7", pathCount)
	}

	// (7) Body must NOT contain "svelte" (case-insensitive) — the replaced icon
	if strings.Contains(strings.ToLower(string(body)), "svelte") {
		t.Fatalf("GET /favicon.svg body contains \"svelte\" (case-insensitive) — the Svelte scaffold icon this fix replaced should not be present")
	}
}

// TestFaviconRasterContentType verifies FIX-02's raster favicon assets
// return 200 with image/png content-type.
func TestFaviconRasterContentType(t *testing.T) {
	handler, _ := newTestSPAHandler(t)

	// Test both raster assets: favicon-32.png and apple-touch-icon.png
	cases := []struct {
		name string
		path string
	}{
		{"favicon-32.png", "/favicon-32.png"},
		{"apple-touch-icon.png", "/apple-touch-icon.png"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: GET %s status = %d, want %d; body: %s", tc.name, tc.path, rec.Code, http.StatusOK, rec.Body.String())
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "image/png" {
			t.Fatalf("%s: GET %s Content-Type = %q, want %q", tc.name, tc.path, contentType, "image/png")
		}
	}
}

// TestFaviconUnderUnchangedCSP verifies FIX-02's threat T-01-06-01:
// the favicon loads as a same-origin static file under the unchanged
// default-src 'self' CSP. The CSP must carry no img-src directive
// (per D-08/FIX-02 invariant: never widen the policy for a cosmetic asset).
func TestFaviconUnderUnchangedCSP(t *testing.T) {
	handler, _ := newTestSPAHandler(t)

	// GET / to capture the CSP on a client-side route (per the root layout)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}

	policy := rec.Header().Get("Content-Security-Policy")
	if policy == "" {
		t.Fatal("Content-Security-Policy is empty")
	}

	directives := parseCSP(t, policy)

	// (1) default-src must be exactly 'self' (no relaxation for img-src)
	if got := directives["default-src"]; len(got) != 1 || got[0] != "'self'" {
		t.Fatalf("default-src = %v, want exactly ['self']; any widening violates D-08/FIX-02 invariant (never widen CSP for a cosmetic asset)", got)
	}

	// (2) img-src directive must NOT exist (that would be a widening)
	if got := directives["img-src"]; len(got) > 0 {
		t.Fatalf("img-src = %v, want no img-src directive; its presence violates D-08/FIX-02 invariant", got)
	}

	// (3) Sanity check: connect-src is still 'self' (unchanged from pre-fix)
	if got := directives["connect-src"]; len(got) != 1 || got[0] != "'self'" {
		t.Fatalf("connect-src = %v, want exactly ['self']", got)
	}
}

// TestFaviconSameSrcNotDataURI verifies FIX-02's delivery model:
// the favicon.svg link serves as a root-relative static file path,
// not as a data: URI. A data: URI would require img-src directive
// relaxation, defeating the point of the fix.
//
// This test proves the SPA handler serves /favicon.svg as a real
// file (by checking the response is non-empty and matches the
// embedded asset), never inlines it, and the CSP on the same
// handler response forbids data: URIs by leaving img-src at
// default-src 'self'.
func TestFaviconSameSrcNotDataURI(t *testing.T) {
	handler, buildFS := newTestSPAHandler(t)

	favicon, err := fs.ReadFile(buildFS, "favicon.svg")
	if err != nil {
		t.Fatalf("fs.ReadFile(buildFS, \"favicon.svg\"): %v", err)
	}

	// GET /favicon.svg must return the actual SVG bytes, not a data: URI wrapper
	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /favicon.svg status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("GET /favicon.svg body is empty — the file is not being served")
	}

	// Body must be the real SVG bytes, starting with "<svg" (characteristic of SVG files)
	if !strings.HasPrefix(strings.ToLower(string(body)), "<svg") {
		t.Fatalf("GET /favicon.svg body does not start with '<svg' — it looks like a data: URI wrapper or placeholder, not the real SVG asset")
	}

	// The CSP on this same response must still forbid data: URIs
	// (img-src inherits from default-src 'self', which forbids data:)
	policy := rec.Header().Get("Content-Security-Policy")
	directives := parseCSP(t, policy)

	// Verify default-src does NOT contain 'data:' (it should only contain 'self')
	for _, source := range directives["default-src"] {
		if strings.Contains(source, "data:") {
			t.Fatalf("default-src contains data: source %q — the CSP was widened to permit data: URIs, defeating FIX-02", source)
		}
	}

	// Similarly, confirm the favicon bytes match the embedded asset exactly
	// (same size, same content — no inlining or transformation)
	if string(body) != string(favicon) {
		t.Fatalf("GET /favicon.svg body does not match embedded favicon.svg byte-for-byte (got %d bytes, want %d bytes)", len(body), len(favicon))
	}
}
