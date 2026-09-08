package uiserver

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// streamingChunkHandler is shared test scaffolding for this file only: it
// writes n chunks, flushing after each and sleeping interval between
// writes, and stops early the moment a Write returns an error (e.g.
// because the connection's absolute write deadline fired) rather than
// looping forever against a dead connection.
func streamingChunkHandler(n int, interval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		for i := 0; i < n; i++ {
			if _, err := fmt.Fprintf(w, "chunk %d\n", i); err != nil {
				return
			}
			fl.Flush()
			if i < n-1 {
				time.Sleep(interval)
			}
		}
	}
}

// countChunks reads newline-delimited chunks from body until EOF or any
// other read error (e.g. the connection being cut by a write deadline),
// returning however many complete lines it saw.
func countChunks(body io.Reader) int {
	r := bufio.NewReader(body)
	count := 0
	for {
		if _, err := r.ReadString('\n'); err != nil {
			return count
		}
		count++
	}
}

// TestStreamDeadlineSurvivesPastWriteTimeout is the load-bearing RED/GREEN
// test for D-02: an httptest.Server whose Config.WriteTimeout is
// deliberately tiny (250ms) mounts a handler behind clearWatchDeadline at
// the generated WatchGraph procedure path, writing a chunk every 100ms
// for over a second. Without the middleware this fails at the deadline —
// removing clearWatchDeadline's SetWriteDeadline call and re-running this
// test was performed this session and observed FAILING (received well
// under 8 chunks); see the SUMMARY for the captured output.
func TestStreamDeadlineSurvivesPastWriteTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(uiv1connect.UIServiceWatchGraphProcedure, streamingChunkHandler(12, 100*time.Millisecond))

	srv := httptest.NewUnstartedServer(clearWatchDeadline(mux))
	srv.Config.WriteTimeout = 250 * time.Millisecond
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL + uiv1connect.UIServiceWatchGraphProcedure)
	if err != nil {
		t.Fatalf("GET %s: %v", uiv1connect.UIServiceWatchGraphProcedure, err)
	}
	defer resp.Body.Close()

	got := countChunks(resp.Body)
	if got < 8 {
		t.Fatalf("received %d chunks over a cleared write deadline, want at least 8 (deadline is 250ms, the handler writes 12 chunks 100ms apart — ~1.1s total, so surviving requires the deadline to actually be cleared)", got)
	}
}

// TestStreamDeadlineNonStreamPathStillCutOff is the paired negative test:
// the SAME tiny write deadline, on the SAME server, at a path
// clearWatchDeadline does not match, still cuts the connection off —
// proving the middleware is scoped to one path rather than having
// silently disabled writeTimeout everywhere.
func TestStreamDeadlineNonStreamPathStillCutOff(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/other", streamingChunkHandler(12, 100*time.Millisecond))

	srv := httptest.NewUnstartedServer(clearWatchDeadline(mux))
	srv.Config.WriteTimeout = 250 * time.Millisecond
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/other")
	if err != nil {
		t.Fatalf("GET /other: %v", err)
	}
	defer resp.Body.Close()

	got := countChunks(resp.Body)
	if got >= 8 {
		t.Fatalf("received %d chunks at an unmatched path under a 250ms write deadline, want fewer than 8 — this path's deadline must still fire since clearWatchDeadline never touches it", got)
	}
}

// TestStreamDeadlineHandlerReceivesFlushableWriter asserts the connect-go
// admission gate directly: the inner handler's http.ResponseWriter still
// satisfies http.Flusher after passing through clearWatchDeadline. A
// middleware that wrapped w in a custom type would fail this even if
// that type itself forwarded Flush, because connect-go's own gate is a
// bare, non-traversing type assertion.
func TestStreamDeadlineHandlerReceivesFlushableWriter(t *testing.T) {
	isFlusherCh := make(chan bool, 1)

	mux := http.NewServeMux()
	mux.Handle(uiv1connect.UIServiceWatchGraphProcedure, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, ok := w.(http.Flusher)
		isFlusherCh <- ok
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(clearWatchDeadline(mux))
	defer srv.Close()

	resp, err := http.Get(srv.URL + uiv1connect.UIServiceWatchGraphProcedure)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	select {
	case isFlusher := <-isFlusherCh:
		if !isFlusher {
			t.Fatal("inner handler's response writer does not implement http.Flusher — clearWatchDeadline must pass w through unwrapped")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}
}

// TestStreamDeadlineResponseControllerSetsDeadlineWithoutError asserts
// the second half of the connect-go admission gate: the same writer
// clearWatchDeadline hands to next also supports
// http.NewResponseController(w).SetWriteDeadline, returning no error —
// checked directly rather than inferred from the timeout test alone.
func TestStreamDeadlineResponseControllerSetsDeadlineWithoutError(t *testing.T) {
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.Handle(uiv1connect.UIServiceWatchGraphProcedure, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		errCh <- http.NewResponseController(w).SetWriteDeadline(time.Time{})
		w.WriteHeader(http.StatusOK)
	}))

	srv := httptest.NewServer(clearWatchDeadline(mux))
	defer srv.Close()

	resp, err := http.Get(srv.URL + uiv1connect.UIServiceWatchGraphProcedure)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("http.NewResponseController(w).SetWriteDeadline(time.Time{}) = %v inside the handler clearWatchDeadline passed w to, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}
}
