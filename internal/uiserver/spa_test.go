package uiserver

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	web "github.com/seanb4t/codegraph-go/web"
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
