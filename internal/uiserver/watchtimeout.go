// Package uiserver: watchtimeout.go implements D-02 — clearing
// http.Server's absolute WriteTimeout for exactly one path, the
// generated WatchGraph streaming procedure, and no other.
//
// writeTimeout (server.go) is an ABSOLUTE deadline on an entire response
// write, sized for a unary rpc's slowest legitimate response (server.go's
// own comment: "roughly two orders of magnitude above" that). A
// server-streaming rpc's response is intentionally long-lived — it has no
// "response finished" moment short of the client disconnecting — so the
// same 60-second bound that protects every unary rpc would kill a
// healthy stream. Clearing it here, for this path only, is the
// deliberate relaxation D-02 chose over its two rejected alternatives:
// WriteTimeout: 0 server-wide (trades away a documented safety property
// across all 13 unary rpcs and the SPA handler to serve one method) and
// forced client reconnects under 60s (churns every open tab at least
// once a minute, close to the reconnect storm LIV-03 exists to prevent).
//
// connectrpc.com/connect v1.20.0's checkServerStreamsCanFlush
// (protocol.go:347-350) gates entry to EVERY server-streaming handler on
// a bare type assertion against http.Flusher, with no Unwrap()
// traversal:
//
//	if _, flushable := responseWriter.(http.Flusher); requiresFlusher && !flushable {
//		return NewError(CodeInternal, fmt.Errorf("%T does not implement http.Flusher", responseWriter))
//	}
//
// A middleware that wraps w in any custom type — even one that itself
// forwards Flush by embedding — hides the assertion's target from that
// check and the handler is refused outright, not merely degraded.
// clearWatchDeadline therefore calls http.NewResponseController(w) for
// its side effect only and passes the ORIGINAL w straight through to
// next, the exact shape originHostGuard (originguard.go) already
// establishes as this package's one other middleware.
package uiserver

import (
	"net/http"
	"time"

	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// clearWatchDeadline matches on the generated
// uiv1connect.UIServiceWatchGraphProcedure constant, never a hand-typed
// path literal, so a future proto rename cannot silently unmatch this
// middleware and reintroduce the 60-second kill on the stream. Every
// other request — all 13 unary rpcs and the SPA handler mounted at "/" —
// passes through completely untouched, keeping writeTimeout's documented
// protection exactly where it already applied.
func clearWatchDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == uiv1connect.UIServiceWatchGraphProcedure {
			// Side effect only: the returned *http.ResponseController is
			// never retained, and w itself is never replaced or
			// wrapped — see this file's own doc comment for why that
			// matters. An error here would mean the underlying
			// connection does not support a write deadline at all
			// (never observed against net/http's own *http.Server,
			// whose per-connection writer always does); there is
			// nothing actionable to do with it, since the alternative
			// is failing a request that would otherwise have worked
			// under the very deadline this call exists to clear.
			_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		}
		next.ServeHTTP(w, r)
	})
}
