// spa.go implements codegraph ui's SPA app-shell handler (D-12): it
// serves the embedded, committed SvelteKit build (web.BuildFS, BLD-02)
// at "/" on the same mux the Connect RPC handler is already mounted on
// (D-09 — Go 1.26 http.ServeMux's most-specific-pattern-wins resolves
// "/codegraph.ui.v1.UIService/" ahead of "/" with no allowlist to keep
// in sync). D-10's asset-vs-route discrimination is implemented as a
// three-way rule: a miss under the immutable-asset prefix is a 404 —
// deliberately, by design, never a fallback, because serving HTML for a
// missing hashed chunk produces a browser MIME/module error that points
// nowhere near the real cause; a hit anywhere else that resolves to a
// real file is served as itself; anything else falls back to
// index.html so SvelteKit's client router can take over. D-11's
// two-class Cache-Control policy and a same-origin
// Content-Security-Policy (with script-src hashes derived from the
// embedded index.html itself) are set unconditionally on every
// response.
//
// Package-level doc comment lives in originguard.go — not repeated here.
package uiserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
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

	// spaCacheControlImmutable is D-11's long-lived class: served on
	// every real file resolved under spaAssetPrefix. Content-hashed
	// paths never change their bytes under a fixed name — a rebuild
	// emits a new hash, not a mutated file at the old one — so a
	// one-year max-age plus "immutable" (skip revalidation entirely) is
	// safe by construction.
	spaCacheControlImmutable = "public, max-age=31536000, immutable"

	// spaCacheControlNoStore is D-11's no-store class: served on
	// index.html only. This is the header that makes the post-upgrade
	// stale-shell failure impossible — a cached old shell must never be
	// replayed against a new RPC surface after a binary upgrade.
	spaCacheControlNoStore = "no-store"

	// spaCacheControlDefault is served on a real file that resolves
	// outside spaAssetPrefix (e.g. a static/ passthrough file such as a
	// favicon or robots.txt, which is not content-hashed). Claude's
	// Discretion under D-11: revalidation ("no-cache", not "no-store")
	// is the value that keeps the post-upgrade stale-shell failure
	// impossible for this class too, without forcing a full re-fetch on
	// every request the way "no-store" would.
	spaCacheControlDefault = "no-cache"

	// spaCSPBaseDirectives is the FIXED half of the Content-Security-Policy
	// this handler serves on every response — non-negotiable and
	// asserted by TestSPASetsCSPOnEveryResponse. It deliberately excludes
	// script-src and style-src: those two are composed separately by
	// buildSPACSPPolicy from hash sources derived at handler-construction
	// time, so the served policy is always the COMPOSED result, never
	// this const on its own.
	//
	// default-src/connect-src 'self' close T-02-02-06 (a compromised or
	// injected embedded asset must not exfiltrate to, or load code from,
	// a third-party origin) — a boundary neither nosniff nor
	// originHostGuard covers, since the origin guard protects the server
	// boundary and says nothing about what a served asset may fetch. The
	// Connect client (02-03) targets baseUrl "/", so same-origin is the
	// whole connect-src requirement; widening it would permit a
	// capability nothing in this milestone uses. object-src/base-uri/
	// form-action/frame-ancestors close the remaining classic injection
	// and embedding vectors this HTML-serving surface newly opens.
	spaCSPBaseDirectives = "default-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
)

// spaScriptBlockRE and spaStyleBlockRE are the purpose-built scan
// buildSPACSPPolicy uses to extract inline <script>/<style> block content
// from the embedded index.html, in place of pulling in an HTML parser:
// the input is this repository's own committed, reviewed, drift-guarded
// build output, not adversarial third-party HTML. Each match captures the
// tag's attribute text (group 1) — checked by spaExternalSrcAttrRE for a
// `src` ATTRIBUTE, which marks an EXTERNAL reference with nothing inline
// to hash — and the verbatim inner content (group 2), hashed byte-exact.
var (
	spaScriptBlockRE = regexp.MustCompile(`(?is)<script(\s[^>]*)?>(.*?)</script>`)
	spaStyleBlockRE  = regexp.MustCompile(`(?is)<style(\s[^>]*)?>(.*?)</style>`)
)

// spaExternalSrcAttrRE matches a `src` ATTRIBUTE in a tag's attribute
// text, anchored to an attribute boundary: `src` must be preceded by
// whitespace (or start of the attribute run) and followed by optional
// whitespace and `=`.
//
// The boundary is the whole point. An unanchored `src=` substring search
// also matches the tail of any attribute NAME ending in "src" —
// `data-hydrate-src="1"`, `data-img-src="…"` — and would then classify an
// inline <script> as external, silently dropping its CSP hash from
// script-src. The served policy would still be strict, so nothing fails
// loudly; the script simply stops executing in the browser, which is a
// blank app shell diagnosed nowhere near this function.
var spaExternalSrcAttrRE = regexp.MustCompile(`(?i)(^|\s)src\s*=`)

// spaHasInlineStyleAttrRE detects a `style="..."` HTML attribute anywhere
// in the document — distinct from a <style> element. CSP hash sources
// apply only to <style> element content, never to attribute values, so
// this is what justifies buildSPACSPPolicy's style-src fallback below
// rather than triggering it unconditionally whenever no <style> element
// is found.
var spaHasInlineStyleAttrRE = regexp.MustCompile(`(?i)\sstyle\s*=\s*['"]`)

// cspHashSource SHA-256s content byte-exact and renders it as a CSP
// hash-source expression ('sha256-<base64>'), the wire format every
// browser's CSP engine expects for a script-src/style-src hash source.
func cspHashSource(content []byte) string {
	sum := sha256.Sum256(content)
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// spaInlineBlockHashes extracts the byte-exact content of every inline
// (non-"src="-attributed) <script> and <style> block in html and returns
// each one's CSP hash-source expression, scripts and styles kept
// separate since they populate different directives. Order matches
// document order; a document with N inline <script> blocks yields exactly
// N script hash sources — TestSPACSPHashesCoverEmbeddedInlineScripts
// derives its own expectation by calling this same function directly
// against the embedded bytes, independently of the handler's composed
// policy.
func spaInlineBlockHashes(html []byte) (scriptSources, styleSources []string) {
	for _, m := range spaScriptBlockRE.FindAllSubmatch(html, -1) {
		attrs, content := m[1], m[2]
		if spaExternalSrcAttrRE.Match(attrs) {
			continue // external script — nothing inline to hash
		}
		scriptSources = append(scriptSources, cspHashSource(content))
	}
	for _, m := range spaStyleBlockRE.FindAllSubmatch(html, -1) {
		attrs, content := m[1], m[2]
		if spaExternalSrcAttrRE.Match(attrs) {
			continue
		}
		styleSources = append(styleSources, cspHashSource(content))
	}
	return scriptSources, styleSources
}

// buildSPACSPPolicy composes the served Content-Security-Policy from
// spaCSPBaseDirectives (the fixed half) plus a script-src and style-src
// derived from html — SvelteKit's adapter-static fallback page at the
// time this was written. script-src is NEVER hand-written and NEVER
// loosened: it is always 'self' plus one 'sha256-…' source per inline
// <script> block spaInlineBlockHashes finds, so the policy re-derives
// itself for free when 02-05 rebuilds the tree — there is no
// hand-maintained hash to go stale.
//
// style-src takes one of three branches, decided empirically against the
// real embedded index.html:
//  1. Inline <style> ELEMENT blocks exist -> hash each one, same as
//     script-src. (Not this build: none found.)
//  2. No <style> elements, but an inline style="..." ATTRIBUTE exists
//     (this build: the SPA root <div>'s "display: contents") -> CSP hash
//     sources apply only to <style> element content, never to attribute
//     values (that requires the separate 'unsafe-hashes' keyword), so
//     there is nothing to hash. Per this task's Claude's-Discretion
//     fallback, style-src alone gets 'unsafe-inline' appended — recorded
//     in the SUMMARY, and script-src carries no such fallback.
//  3. Neither exists -> style-src stays 'self' with no relaxation.
func buildSPACSPPolicy(html []byte) string {
	scriptSources, styleSources := spaInlineBlockHashes(html)

	scriptSrc := "script-src 'self'"
	for _, s := range scriptSources {
		scriptSrc += " " + s
	}

	var styleSrc string
	switch {
	case len(styleSources) > 0:
		styleSrc = "style-src 'self'"
		for _, s := range styleSources {
			styleSrc += " " + s
		}
	case spaHasInlineStyleAttrRE.Match(html):
		styleSrc = "style-src 'self' 'unsafe-inline'"
	default:
		styleSrc = "style-src 'self'"
	}

	return spaCSPBaseDirectives + "; " + scriptSrc + "; " + styleSrc
}

// newSPAHandler returns the SPA app-shell http.Handler, serving out of
// fsys (the caller passes fs.Sub(web.BuildFS, spaSubdirName), so fsys's
// root already IS the build output root). The Content-Security-Policy is
// computed exactly once here, from the embedded index.html, and stored on
// the handler — never recomputed per request, and never hand-maintained
// as a literal.
func newSPAHandler(fsys fs.FS) http.Handler {
	csp := spaCSPBaseDirectives + "; script-src 'self'; style-src 'self'"
	if indexHTML, err := fs.ReadFile(fsys, spaFallbackFile); err == nil {
		csp = buildSPACSPPolicy(indexHTML)
	}
	return &spaHandler{fsys: fsys, csp: csp}
}

type spaHandler struct {
	fsys fs.FS
	csp  string
}

// ServeHTTP implements D-10's complete three-way rule and D-11's
// two-class cache policy. X-Content-Type-Options and Content-Security-Policy
// are set FIRST, unconditionally, before the method check or any routing
// branch — including on the 405 path — so no branch, present or future,
// can forget either header (T-02-02-01, T-02-02-06).
func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", h.csp)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Normalize the request path: strip the single leading slash, then
	// path.Clean. An embed.FS cannot be escaped by "..", so fs.ValidPath
	// here is defense in depth, not the only barrier.
	p := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if p == "." || p == "" || !fs.ValidPath(p) {
		h.serveFallback(w, r)
		return
	}

	if strings.HasPrefix(p, spaAssetPrefix) {
		fi, statErr := fs.Stat(h.fsys, p)
		if statErr != nil || fi.IsDir() {
			// A miss under the immutable-asset prefix is a 404 BY
			// DESIGN (D-10) — never a fallback to index.html. Serving
			// HTML for a missing hashed chunk would produce a browser
			// MIME/module error that points nowhere near the real
			// cause; a 404 at least names the actual missing path.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			if r.Method == http.MethodGet {
				fmt.Fprintf(w, "asset not found: %s\n", p)
			}
			return
		}
		h.serveFile(w, r, p, spaCacheControlImmutable)
		return
	}

	if fi, statErr := fs.Stat(h.fsys, p); statErr == nil && !fi.IsDir() {
		h.serveFile(w, r, p, spaCacheControlDefault)
		return
	}

	h.serveFallback(w, r)
}

// serveFile byte-serves the already-resolved regular file at p with the
// given Cache-Control class. Content-Type is resolved deliberately via
// mime.TypeByExtension on p's own extension — NOT via ServeContent's
// content-sniffing fallback, since nosniff is this handler's own contract
// — falling back to application/octet-stream when the extension is
// unregistered. http.ServeContent (not http.FileServer/FileServerFS/
// ServeFileFS — see the file's package doc comment) does the actual byte
// serving once the branch and every header are already decided, handling
// HEAD and Range requests correctly without any of FileServer's
// implicit index.html-redirect/directory-listing machinery.
func (h *spaHandler) serveFile(w http.ResponseWriter, r *http.Request, p string, cacheControl string) {
	data, err := fs.ReadFile(h.fsys, p)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if r.Method == http.MethodGet {
			fmt.Fprintf(w, "asset not found: %s\n", p)
		}
		return
	}

	ct := mime.TypeByExtension(path.Ext(p))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, p, time.Time{}, bytes.NewReader(data))
}

// serveFallback serves spaFallbackFile (index.html) with 200 and
// spaCacheControlNoStore (D-11, D-04) — the branch every path that is not
// a real file, or is empty, takes so SvelteKit's client router can
// resolve the actual client-side route.
func (h *spaHandler) serveFallback(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(h.fsys, spaFallbackFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", spaCacheControlNoStore)
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
