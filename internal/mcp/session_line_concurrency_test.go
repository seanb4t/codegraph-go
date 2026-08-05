package mcp

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
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
// one BuildServer call owns exactly one mutex: sessionLineClients clients
// initialize concurrently, and each initializes sessionLineInitsPerClient
// times, since AddAfterInitialize fires again on re-initialize.
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

			c, err := mcpclient.NewInProcessClient(s)
			if err != nil {
				t.Errorf("NewInProcessClient: %v", err)
				return
			}
			defer c.Close()

			ctx := context.Background()
			for j := 0; j < sessionLineInitsPerClient; j++ {
				initClient(t, ctx, c)
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
