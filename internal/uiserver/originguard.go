// Package uiserver implements codegraph ui's local, loopback-only Connect
// RPC server. originguard.go is the first piece to exist in this package
// (SRV-02): the exact-match Origin/Host control that must run before any
// RPC handler is reachable at all — the ROADMAP's blocking order names
// this file, not a handler, as this phase's Task 1.
package uiserver

import "net/http"

// originHostGuard is the SRV-02 mitigation for DNS-rebinding attacks
// against codegraph ui's loopback listener — the same class documented in
// CVE-2024-28224 (Ollama's loopback API was reachable from any webpage via
// a rebound Host header) and CVE-2025-66414/66416 (recurring in MCP SDK
// loopback servers). Loopback binding (127.0.0.1) is NOT itself a
// security boundary: a browser can be induced to send a request whose
// Host header names an attacker-controlled DNS entry that resolves to
// 127.0.0.1, and the TCP connection really is local even though the
// request originated from an arbitrary web page. The Host header is what
// reveals this — it is mandatory on every HTTP/1.1 request — which is why
// Host, not the bind address, is the primary check here.
//
// Matching is exact map-key equality only: no substring, prefix, suffix
// or case-insensitive comparison of any kind, and no normalisation
// between the three loopback spellings this package admits. "127.0.0.1",
// "localhost" and "[::1]" are three different names, not three renderings
// of one name — admitting a request on one must never admit a request
// that merely resembles another.
//
// A GET request legitimately carries no Origin header at all on a
// same-origin navigation — browsers reliably send Origin only on
// "unsafe" methods (POST, PUT, DELETE, ...) and on genuinely
// cross-origin requests, not on same-origin GET — so Origin's ABSENCE is
// never itself a rejection reason; only its PRESENCE with a disallowed
// value is (01-RESEARCH.md Pitfall 5). When Origin IS present it must
// belong to the SAME spelling as the admitted Host: membership in the
// origin allowlist alone is not enough, because two independently
// checked allowlists would admit `Host: 127.0.0.1:<port>` paired with
// `Origin: http://localhost:<port>` — both individually admitted
// literals, but a cross-spelling pair that mixes two different loopback
// identities into one request. Pairing (origin == "http://"+r.Host)
// closes exactly that gap.
func originHostGuard(port string, next http.Handler) http.Handler {
	hosts := allowedHosts(port)
	origins := allowedOrigins(port)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := hosts[r.Host]; !ok {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			// Both conditions are required: membership (the enumerated
			// contract a reader audits) and pairing with THIS request's
			// Host (the per-request tie that rejects a cross-spelling
			// pair such as Host: 127.0.0.1:<port> carrying
			// Origin: http://localhost:<port>, which membership alone
			// would admit).
			if _, ok := origins[origin]; !ok || origin != "http://"+r.Host {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// allowedHosts returns the exact set of Host header values admitted for a
// server bound to the given port: the three loopback spellings this
// project treats as equally legitimate — each its own literal, never
// derived from another by normalisation (SRV-02, D-07). Exactly these
// three and no others.
func allowedHosts(port string) map[string]struct{} {
	return map[string]struct{}{
		"127.0.0.1:" + port: {},
		"localhost:" + port: {},
		"[::1]:" + port:     {},
	}
}

// allowedOrigins returns the exact set of Origin header values admitted
// for a server bound to the given port — the same three spellings as
// allowedHosts, each rendered as a full "http://" origin. Exactly these
// three and no others.
func allowedOrigins(port string) map[string]struct{} {
	return map[string]struct{}{
		"http://127.0.0.1:" + port: {},
		"http://localhost:" + port: {},
		"http://[::1]:" + port:     {},
	}
}
