package uiserver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
	"github.com/seanb4t/codegraph-go/web"
)

// DefaultAddr is the ephemeral loopback bind address D-07 mandates: with
// a ":0" port, no concrete port exists until net.Listen actually
// returns, so Listen always resolves an empty Options.Addr to this value
// rather than a fixed, potentially-colliding port. A second `codegraph
// ui` simply gets a different port.
const DefaultAddr = "127.0.0.1:0"

// The four http.Server timeouts Listen sets (WR-03). Named here rather
// than written as bare durations at the construction site, per this
// package's discipline of never repeating a limit as a literal, and
// asserted as a set by TestListenSetsEveryServerTimeout.
//
// Every request this service answers is a local, in-process query
// against an on-disk index — there is no upstream to wait on — so these
// are sized for "a legitimate request finishes in well under a second,
// with generous slack" rather than for network variance:
//
//   - readHeaderTimeout is the load-bearing one: it is the ONLY bound on
//     a connection that has been accepted but whose header block never
//     completes, which is precisely what originHostGuard cannot see.
//   - writeTimeout must exceed the slowest legitimate response. The
//     slowest is an Explore carrying up to uiExploreSourceGroupCap
//     source blobs (handlers.go); 60s is roughly two orders of
//     magnitude above that on any machine this runs on.
//   - idleTimeout bounds a kept-alive connection between requests, so an
//     abandoned tab's socket is reclaimed rather than held for the
//     process's lifetime.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

// Options configures Listen. RepoPath and Addr are the ONLY two fields —
// asserted by TestUIServiceHoldsNoStoreTypedField's sibling assertion in
// server_test.go, a reflected field-set equality with a length-2 check —
// so neither an unwired option nor a second browser-launch owner can be
// reintroduced. Browser-launch policy and the launch itself live
// entirely in internal/cli, never here: one owner means neither an
// unused option nor a double launch (review MEDIUM 01-06).
type Options struct {
	// RepoPath is the single repository uiService answers every rpc
	// for.
	RepoPath string

	// Addr is the bind address net.Listen receives, defaulting to
	// DefaultAddr when empty. Nothing outside this package writes this
	// field in v1 — no flag, no environment variable, no configuration
	// file reads it (D-08). A later `--host`/`--port` is therefore an
	// override of a field that already exists, not a restructuring
	// (SRV-03).
	Addr string
}

// Server is a bound codegraph ui listener: bound the instant Listen
// returns, serving only once Serve is called. The split exists because
// the port cannot be published before the bind, and the bind cannot live
// inside the blocking call — see Listen's own doc comment.
type Server struct {
	ln  net.Listener
	srv *http.Server
	url string
}

// Listen binds addr (defaulting to DefaultAddr), extracts the concrete
// bound port, builds the http.ServeMux with the UIService Connect
// handler mounted on it, wraps the WHOLE mux in originHostGuard — the
// guard is the outermost http.Handler, never a connect interceptor,
// because an interceptor runs after routing and message parsing have
// already begun and would not protect Phase 2's static SPA routes
// either — and returns. The bind happens here, inside Listen, and never
// inside Serve: with an ephemeral ":0" bind the port does not exist
// until net.Listen returns, and (*Server).URL must already be real the
// instant Listen returns, which is only possible because Listen itself
// never blocks.
func Listen(o Options) (*Server, error) {
	addr := o.Addr
	if addr == "" {
		addr = DefaultAddr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("uiserver: Listen bound a non-TCP address: %v", ln.Addr())
	}
	port := strconv.Itoa(tcpAddr.Port)

	mux := http.NewServeMux()
	// D-13's transport backstop: connect-go defaults to UNLIMITED on both
	// send and receive, so RPC-05 is satisfied by no default.
	// transportSendMaxBytes/transportReadMaxBytes (truncate.go) sit
	// strictly above the application caps, asserted by
	// TestTransportBackstopSitsAboveApplicationCap, so this backstop only
	// ever fires when application truncation was somehow missed — never
	// on correctly-truncated output.
	mux.Handle(uiv1connect.NewUIServiceHandler(
		&uiService{repoPath: o.RepoPath},
		connect.WithSendMaxBytes(transportSendMaxBytes),
		connect.WithReadMaxBytes(transportReadMaxBytes),
	))

	// D-09: the SPA handler registers at "/" on the SAME mux, BEFORE
	// originHostGuard wraps it — Go 1.26 http.ServeMux resolves
	// most-specific-pattern-wins, so the Connect prefix above stays
	// preferred over "/" with no allowlist to maintain. Registering the
	// SPA handler here (inside the mux, before the guard) rather than
	// as a separate outer layer is what lets it inherit SRV-02's
	// Origin/Host protection with no extra code (BLD-02).
	buildFS, err := fs.Sub(web.BuildFS, spaSubdirName)
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("uiserver: Listen could not derive SPA build sub-filesystem: %w", err)
	}
	mux.Handle("/", newSPAHandler(buildFS))

	guarded := originHostGuard(port, mux)

	return &Server{
		ln: ln,
		srv: &http.Server{
			Handler: guarded,
			// WR-03 (gosec G112): net/http's own defaults are all "no
			// limit". originHostGuard runs at the HANDLER level, which
			// is AFTER the header block has been read, so it is no
			// defence at all against a client that connects and simply
			// never finishes its headers — and any local process, plus
			// any web page (which can open a socket to
			// localhost:<port> even though the guard will reject the
			// eventual request), can hold such connections open, each
			// pinning a goroutine and a file descriptor until the
			// process hits its fd limit and stops serving the
			// legitimate UI.
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		url: "http://127.0.0.1:" + port,
	}, nil
}

// URL returns the server's connectable base URL, valid the instant
// Listen returns — before Serve is ever called.
func (s *Server) URL() string {
	return s.url
}

// Serve blocks until ctx is cancelled or the underlying http.Server
// stops on its own, normalizing http.ErrServerClosed to nil at every
// return path — codegraph ui returns this value straight from RunE, so a
// normalized nil is the difference between a clean Ctrl-C and a spurious
// non-zero exit code.
//
// http.Server.Serve blocks, so a literal "serve, then on ctx.Done() shut
// down" would never reach the cancellation branch until serving had
// already stopped by itself. Serve therefore runs the blocking Serve
// call in a goroutine over a BUFFERED capacity-1 error channel and
// selects on that channel against ctx.Done(): buffered, so the goroutine
// can always deposit its result and exit even if nobody is left
// reading — an unbuffered channel here would leak the goroutine on the
// cancellation path.
func (s *Server) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.srv.Serve(s.ln)
	}()

	select {
	case err := <-errCh:
		return normalizeServeErr(err)
	case <-ctx.Done():
		// Derived from context.Background(), NOT from the
		// already-cancelled ctx — Shutdown against an already-done
		// context returns immediately without draining any in-flight
		// connections.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownErr := s.srv.Shutdown(shutdownCtx)
		<-errCh // Drain: the serve goroutine is known to have finished before Serve returns, so it can never leak.
		if shutdownErr != nil {
			return shutdownErr
		}
		return nil
	}
}

// normalizeServeErr maps http.ErrServerClosed — what Serve returns after
// a clean Shutdown or Close — to nil. Any other error is returned
// unchanged.
func normalizeServeErr(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Close closes the listener without ever calling Serve — for a caller
// that binds and then decides not to serve.
func (s *Server) Close() error {
	return s.ln.Close()
}
