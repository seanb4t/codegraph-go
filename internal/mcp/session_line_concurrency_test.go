package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// interleaveProneWriter is an io.Writer whose Write is deliberately
// non-atomic: it appends its payload in small chunks and yields the
// scheduler between them. It models the weakest thing an arbitrary
// io.Writer is allowed to be — server.go's session-line hook writes to a
// caller-supplied io.Writer (os.Stderr in production), and io.Writer
// carries no atomicity guarantee whatsoever.
//
// The writer holds its own mutex around each chunk append, so it is
// internally race-free. That is deliberate: this test must measure the
// SERVER's mutex, not the writer's thread-safety. Under -race an
// unsynchronized bytes.Buffer would report a data race for the wrong
// reason and would still pass without -race; chunked interleaving fails
// deterministically under plain `go test` instead.
type interleaveProneWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *interleaveProneWriter) Write(p []byte) (int, error) {
	const chunk = 8
	for i := 0; i < len(p); i += chunk {
		end := i + chunk
		if end > len(p) {
			end = len(p)
		}
		w.mu.Lock()
		w.buf.Write(p[i:end])
		w.mu.Unlock()
		runtime.Gosched()
	}
	return len(p), nil
}

func (w *interleaveProneWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// sendRawInitialize drives a classic JSON-RPC "initialize" handshake
// directly against s over a fresh in-memory transport pair, deliberately
// bypassing mcp.NewClient/Client.Connect.
//
// Why this exists (a load-bearing SDK-01 finding this file's other helper,
// newTestSession, cannot paper over): go-sdk v1.7.0's Client.Connect
// defaults its protocol version to latestProtocolVersion (2026-07-28) and,
// per SEP-2575, tries the stateless "server/discover" RPC FIRST —
// $GOMODCACHE/.../mcp/server.go's Server.discover is wired in
// unconditionally by mcp.NewServer, with no ServerOptions field to opt
// out — so two go-sdk v1.7.0 peers using the SDK's own Client/Server pair
// NEVER negotiate via the classic "initialize" method this package's
// session line is keyed on (confirmed empirically: instrumenting the
// middleware showed method="server/discover", never "initialize", for
// every newTestSession-based test in this package). The one field that
// could force the classic path client-side, ClientSessionOptions.
// protocolVersion, is unexported ("for testing", go-sdk's own internal
// test suite only) and unreachable from a downstream package — confirmed
// by reading its declaration. VRFY-03's session line is explicitly scoped
// to the classic initialize handshake (02-CONTEXT.md's domain boundary:
// "Explicitly NOT in scope: implementing 2026-07-28's obligations —
// server/discover... That is Phase 3" — Phase 2 does not implement that
// revision's semantics), so this helper drives the classic handshake
// directly via the public jsonrpc/mcp.Connection primitives — the same
// wire-level approach test/wireoracle's own hand-rolled driver already
// uses for the real stdio path, just reused here for the in-memory one.
// It never imports test/wireoracle (package boundary) and never re-derives
// go-sdk's own negotiation logic — it only sends the one request VRFY-03
// needs to see.
func sendRawInitialize(t *testing.T, s *mcp.Server, clientName, clientVersion string) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	conn, err := clientTransport.Connect(ctx)
	if err != nil {
		t.Fatalf("clientTransport.Connect: %v", err)
	}
	defer conn.Close()

	params, err := json.Marshal(map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": clientName, "version": clientVersion},
	})
	if err != nil {
		t.Fatalf("marshal initialize params: %v", err)
	}
	id, err := jsonrpc.MakeID(float64(1))
	if err != nil {
		t.Fatalf("jsonrpc.MakeID: %v", err)
	}

	if err := conn.Write(ctx, &jsonrpc.Request{ID: id, Method: "initialize", Params: params}); err != nil {
		t.Fatalf("write initialize request: %v", err)
	}

	msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read initialize response: %v", err)
	}
	if resp, ok := msg.(*jsonrpc.Response); !ok || resp.Error != nil {
		t.Fatalf("initialize response = %+v, want a successful *jsonrpc.Response", msg)
	}
}

// TestSessionLineSurvivesConcurrentAndRepeatedInitialize is the executable
// backstop for the non-interleaving property server.go's session-line hook
// asserts in prose ("the mutex is what makes 'never a partially-written or
// interleaved session line' a construction property rather than an
// assumption").
//
// Before this test, that mutex had no coverage of any kind: every test in
// session_line_test.go drives formatSessionLine directly on a single
// goroutine, so deleting the mutex left the whole package green — and
// internal/mcp was not in test:race's package set either, so the race
// detector was not a fallback net. Both gaps were found during phase-01
// UAT (01-UAT.md test 2) and closed together; this test is the first half,
// adding ./internal/mcp/... to Taskfile.yml's test:race is the second.
//
// The test drives concurrency AND repetition through one server, because
// one BuildServer call owns exactly one mutex: sessionLineClients
// goroutines fire concurrently, and each sends
// sessionLineInitsPerClient separate classic-initialize handshakes (via
// sendRawInitialize, above) in turn.
//
// This differs from the pre-migration shape in two respects, both forced
// by go-sdk rather than chosen: (1) mark3labs allowed calling Initialize
// repeatedly on the SAME client, re-firing AddAfterInitialize each time,
// so the original test looped initClient on one shared client — go-sdk's
// ss.initialize() explicitly rejects a second "initialize" on the SAME
// session ("duplicate %q received"), so each of sessionLineInitsPerClient
// "repeats" is its OWN sendRawInitialize call (its own session, its own
// initialize) rather than a second Initialize on one session; (2) driving
// the handshake at the raw jsonrpc/mcp.Connection level, rather than via
// mcp.NewClient, per sendRawInitialize's own doc comment. Neither changes
// the property under test — this server's ONE mutex surviving many
// concurrent, interleaving-prone session-line writes does not depend on
// how many distinct sessions produced that concurrent load, or which
// client API sent the handshake — so the total session-line count
// (sessionLineClients * sessionLineInitsPerClient) and the
// non-interleaving assertion below are unchanged.
//
// Non-vacuity: with the mutex removed from server.go, chunks from
// concurrent hook invocations interleave in the writer, and the resulting
// malformed lines fail parseSessionLineFields — verified by applying that
// mutation and observing this test go red.
func TestSessionLineSurvivesConcurrentAndRepeatedInitialize(t *testing.T) {
	const (
		sessionLineClients        = 8
		sessionLineInitsPerClient = 4
	)

	dir := t.TempDir()
	w := &interleaveProneWriter{}

	// One server, so all sessions share the single mutex under test.
	s := BuildServer(false, map[string]bool{}, dir, dir, WithSessionLog(w))

	var wg sync.WaitGroup
	for i := 0; i < sessionLineClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < sessionLineInitsPerClient; j++ {
				sendRawInitialize(t, s, "codegraph-mcp-test", "0.0.0")
			}
		}()
	}
	wg.Wait()

	out := w.String()
	if out == "" {
		t.Fatal("no session lines were written at all — the always-on VRFY-03 line did not fire")
	}

	// TrimSuffix then Split, so the final line's trailing newline does not
	// produce a spurious empty element.
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	wantLines := sessionLineClients * sessionLineInitsPerClient
	if len(lines) != wantLines {
		t.Fatalf("got %d session lines, want %d (one per initialize; a mismatch means a line was lost, duplicated, or split)", len(lines), wantLines)
	}

	for i, line := range lines {
		fields, err := parseSessionLineFields(line + "\n")
		if err != nil {
			t.Fatalf("session line %d of %d is malformed: %v\nfull output:\n%s", i+1, len(lines), err, out)
		}
		if got := fields["client"]; got != "codegraph-mcp-test/0.0.0" {
			t.Fatalf("session line %d client field = %q, want %q (a corrupted field means two lines interleaved)", i+1, got, "codegraph-mcp-test/0.0.0")
		}
	}
}
