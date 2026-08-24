// spa.go implements codegraph ui's SPA app-shell handler (D-12): it
// serves the embedded, committed SvelteKit build (web.BuildFS, BLD-02)
// at "/" on the same mux the Connect RPC handler is already mounted on
// (D-09 — Go 1.26 http.ServeMux's most-specific-pattern-wins resolves
// "/codegraph.ui.v1.UIService/" ahead of "/" with no allowlist to keep
// in sync). D-10's asset-vs-route discrimination (a miss under the
// immutable-asset prefix is a 404; a miss anywhere else falls back to
// index.html) and D-11's two-class Cache-Control policy are declared
// here as named constants but not yet implemented — this tracer task
// (02-01) proves exactly one path end to end: GET / serves the real
// embedded index.html. 02-02 fills in the fallback/asset rules that
// spaFallbackFile and spaAssetPrefix exist for.
//
// Package-level doc comment lives in originguard.go — not repeated here.
package uiserver

import (
	"io/fs"
	"net/http"
)

const (
	// spaSubdirName is the directory inside web.BuildFS every lookup is
	// rooted under — adapter-static's own default output directory name
	// (D-03), stripped exactly once via fs.Sub so lookups read
	// "index.html", not "build/index.html".
	spaSubdirName = "build"

	// spaFallbackFile is the SPA fallback SvelteKit's client router
	// takes over from (D-04) — deliberately index.html, not SvelteKit's
	// own recommended 200.html: this app is served by our own Go mux,
	// which never special-cases a "200.html" filename the way a static
	// host does.
	spaFallbackFile = "index.html"

	// spaAssetPrefix is D-10's asset boundary: a miss under this prefix
	// is a 404, never a fallback, because serving HTML for a missing
	// hashed chunk produces a browser MIME/module error that points
	// nowhere near the real cause. Deliberately a prefix, not a
	// file-extension heuristic — Phase 3's deep links will carry file
	// paths containing dots, which a dot-based heuristic would wrongly
	// 404.
	spaAssetPrefix = "_app/immutable/"
)

// newSPAHandler returns the SPA app-shell http.Handler, serving out of
// fsys (the caller passes fs.Sub(web.BuildFS, spaSubdirName), so fsys's
// root already IS the build output root). In this tracer it implements
// exactly one path: GET / (and HEAD /) return the embedded index.html
// verbatim with a text/html Content-Type. Any other method is rejected
// with 405; 02-02 adds the asset-prefix and fallback-routing rules for
// every other path.
func newSPAHandler(fsys fs.FS) http.Handler {
	return &spaHandler{fsys: fsys}
}

type spaHandler struct {
	fsys fs.FS
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := fs.ReadFile(h.fsys, spaFallbackFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
