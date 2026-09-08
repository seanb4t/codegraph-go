package uiserver

import (
	"context"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// watchGraphSender is the minimal interface WatchGraph's send loop needs
// from *connect.ServerStream[uiv1.WatchGraphEvent]. connect.ServerStream
// has no exported constructor (its own doc comment says so explicitly),
// so extracting this interface is what lets runWatchGraphLoop's error-
// classification branch be exercised directly against a fake sender,
// rather than only reachable through a full real HTTP/Connect round
// trip.
type watchGraphSender interface {
	Send(*uiv1.WatchGraphEvent) error
}

// WatchGraph is the real, functioning body for the service's only
// streaming rpc (RPC-04): subscribe to the publisher, loop selecting
// between the subscriber channel and the request context's Done
// channel, send each event, and deregister on the way out via a
// deferred unsubscribe.
//
// This handler holds no store handle of any kind (SRV-04): it never
// calls withEngine, openEngine, or query.OpenAt. The publisher already
// performs the one-shot open/read/close per check (livepublish.go);
// this file is a thin adapter over Subscribe/unsubscribe, which is
// exactly what keeps criterion 5 (concurrent multi-process access)
// satisfiable — a stream held open here pins no Pebble snapshot.
//
// Structural note for LIV-03: this handler's per-request state (one
// subscriber registration, released by the deferred unsubscribe) and
// the publisher's server-initiated send accounting (liveRegistry's
// sendCount, livepublish.go) are two separate values that never share
// storage. Nothing on this path increments or reads sendCount at all —
// the same counter-corruption shape internal/mcp's pendingWriter
// (server.go:466) already fixed once cannot recur here, because the two
// kinds of state never touch the same field.
//
// The request's since_generation is deliberately unread (IN-02,
// 06-REVIEW.md): the registry has no per-generation history to serve
// from, only a single current value, so a resuming client's prior
// generation cannot change what this subscription gets seeded with.
// Resumption is entirely a client-side concern (live-store.ts's
// epoch-scoped admission gate) — this mirrors the equally deliberate
// doc comments already present on the TypeScript side.
func (s *uiService) WatchGraph(ctx context.Context, _ *connect.Request[uiv1.WatchGraphRequest], stream *connect.ServerStream[uiv1.WatchGraphEvent]) error {
	ch, unsubscribe := s.publisher.Subscribe(ctx)
	defer unsubscribe()

	return runWatchGraphLoop(ctx, ch, stream)
}

// runWatchGraphLoop is WatchGraph's send loop, extracted so its error
// classification can be exercised directly against a fake sender and a
// plain channel — connect.ServerStream has no exported constructor to
// build a real one outside a live HTTP round trip.
//
// A send failure BRANCHES on ctx.Err() FIRST rather than passing every
// failure through mapContextError uniformly: stream.Send can return a
// broken-pipe or connection-reset error BEFORE req.Context().Err() has
// become visible, and treating every such failure as a cancellation
// would silently swallow a genuine transport fault as an ordinary closed
// tab.
func runWatchGraphLoop(ctx context.Context, ch <-chan *uiv1.WatchGraphEvent, sender watchGraphSender) error {
	for {
		select {
		case <-ctx.Done():
			// A closed tab, an aborted fetch, or the client's own
			// disconnect — reported as a cancellation, never an
			// internal error.
			return mapContextError(ctx.Err())
		case ev, ok := <-ch:
			if !ok {
				// The publisher stopped (server shutdown): the
				// registry closed every subscriber channel. The
				// client sees a normal stream end — its own backoff
				// handles the reconnect — never an error here.
				return nil
			}
			if err := sender.Send(ev); err != nil {
				if ctx.Err() != nil {
					return mapContextError(ctx.Err())
				}
				// A genuine transport failure with a still-live
				// context: return it unclassified rather than
				// mislabelling it a cancellation.
				return err
			}
		}
	}
}
