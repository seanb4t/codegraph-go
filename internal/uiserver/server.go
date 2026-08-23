package uiserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// DefaultAddr is the ephemeral loopback bind address D-07 mandates: with
// a ":0" port, no concrete port exists until net.Listen actually
// returns, so Listen always resolves an empty Options.Addr to this value
// rather than a fixed, potentially-colliding port. A second `codegraph
// ui` simply gets a different port.
const DefaultAddr = "127.0.0.1:0"

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
	mux.Handle(uiv1connect.NewUIServiceHandler(&uiService{repoPath: o.RepoPath}))

	guarded := originHostGuard(port, mux)

	return &Server{
		ln:  ln,
		srv: &http.Server{Handler: guarded},
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
