package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPendingWriterDrainsEarlyWithoutFix is FIX-01's committed reproduction
// (ROADMAP success criterion 5: the fix must be "proven against a
// reproduction that showed the loss"). It demonstrates a PREMATURE DRAIN —
// the pending count reaching zero while a response is still owed, so
// waitForDrain stops waiting and the reader propagates EOF early — NOT a
// proven abandoned response. Whether a particular in-flight response is
// then actually lost depends on the surrounding go-sdk session and is not
// what this primitive-level reproduction observes; a full-stdio test
// confirming an abandoned response would be stronger evidence and is out
// of scope here.
//
// Reproduction: one client call is accepted (pending=1, mirroring
// stdinLingerReader.Read's increment for an accepted call whose response
// has not yet been written). A server-initiated notification — never
// counted on the increment side, since stdinLingerReader only increments
// for lines looksLikeJSONRPCCall classifies as calls — is written through
// pendingWriter BEFORE the call's own response is written. Before the fix,
// pendingWriter.Write unconditionally decrements on every Write, so this
// single notification alone drains pending to zero and waitForDrain
// returns immediately even though the call's response has not been
// written. After the fix, the notification does not decrement (it carries
// a "method" and no "id", so looksLikeJSONRPCResponse classifies it as not
// a response), pending stays at 1, and waitForDrain keeps blocking until
// the real response write lands.
func TestPendingWriterDrainsEarlyWithoutFix(t *testing.T) {
	var pending atomic.Int64
	pending.Store(1) // one accepted client call, response not yet written

	var buf bytes.Buffer
	pw := &pendingWriter{w: &buf, pending: &pending}

	notification := []byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n")
	if _, err := pw.Write(notification); err != nil {
		t.Fatalf("Write notification: %v", err)
	}

	// The reproduction: waitForDrain must still be blocked here, because
	// the accepted call's response has not been written yet. If it
	// unblocks after only the notification was written, that is the
	// premature drain — the pending count reached zero while a response
	// was still owed.
	s := &stdinLingerReader{pending: &pending}
	drained := make(chan struct{})
	go func() {
		s.waitForDrain()
		close(drained)
	}()

	select {
	case <-drained:
		t.Fatalf("waitForDrain returned after a notification write alone (pending=%d) — premature drain: the accepted call's response was never written, so drain should still be blocking", pending.Load())
	case <-time.After(20 * time.Millisecond):
		// Expected on the fixed code: still blocked, because the
		// notification write did not decrement pending.
	}

	// Now the accepted call's actual response lands.
	response := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n")
	if _, err := pw.Write(response); err != nil {
		t.Fatalf("Write response: %v", err)
	}

	select {
	case <-drained:
		// Expected: drain unblocks once the real response is written.
	case <-time.After(stdinLingerGrace + time.Second):
		t.Fatalf("waitForDrain did not unblock after the accepted call's response was written")
	}
}

// TestPendingWriterDecrementsOnlyResponses covers all seven outbound line
// shapes looksLikeJSONRPCResponse must classify, asserting the pending
// counter delta each produces. Exactly seven t.Run subtests, not seven
// sequential assertions in one body — this test's floor-derivation
// contribution is 1 + 7 = 8 PASS lines.
func TestPendingWriterDecrementsOnlyResponses(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		delta int64 // pending delta this line should produce, starting from 1
	}{
		{
			name:  "response with id and result",
			line:  `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n",
			delta: -1,
		},
		{
			name:  "response with id and error",
			line:  `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"boom"}}` + "\n",
			delta: -1,
		},
		{
			name:  "notification (method, no id)",
			line:  `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n",
			delta: 0,
		},
		{
			name:  "server-initiated request (method AND id)",
			line:  `{"jsonrpc":"2.0","method":"sampling/createMessage","id":3}` + "\n",
			delta: 0,
		},
		{
			name:  "id: null and no method (JSON-RPC parse-error response shape)",
			line:  `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}` + "\n",
			delta: 0,
		},
		{
			name:  "unparseable line",
			line:  "this is not json\n",
			delta: 0,
		},
		{
			name:  "empty line",
			line:  "\n",
			delta: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var pending atomic.Int64
			pending.Store(1)

			var buf bytes.Buffer
			pw := &pendingWriter{w: &buf, pending: &pending}

			if _, err := pw.Write([]byte(c.line)); err != nil {
				t.Fatalf("Write(%q): %v", c.line, err)
			}

			got := pending.Load() - 1
			if got != c.delta {
				t.Fatalf("pending delta = %d, want %d for line %q", got, c.delta, c.line)
			}
		})
	}
}

// partialWriter is a test double for the underlying stdout writer that
// accepts at most n bytes of any Write call and then reports err —
// simulating io.Writer's documented "n < len(b) together with an error"
// short-write contract.
type partialWriter struct {
	n   int
	err error
}

func (p *partialWriter) Write(b []byte) (int, error) {
	n := p.n
	if n > len(b) {
		n = len(b)
	}
	return n, p.err
}

// TestPendingWriterClassifiesOnlyBytesActuallyWritten asserts the
// classifier's input is b[:n], never b: a short write must decrement only
// for the complete response lines actually inside b[:n]. Exactly two
// t.Run subtests, contributing 1 + 2 = 3 PASS lines.
func TestPendingWriterClassifiesOnlyBytesActuallyWritten(t *testing.T) {
	t.Run("short-write", func(t *testing.T) {
		line1 := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n")
		line2 := []byte(`{"jsonrpc":"2.0","id":2,"result":{}}` + "\n")
		full := append(append([]byte{}, line1...), line2...)

		var pending atomic.Int64
		pending.Store(2)

		underlying := &partialWriter{n: len(line1), err: io.ErrShortWrite}
		pw := &pendingWriter{w: underlying, pending: &pending}

		n, err := pw.Write(full)
		if n != len(line1) {
			t.Fatalf("Write returned n=%d, want %d (only line1's bytes were accepted)", n, len(line1))
		}
		if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("Write returned err=%v, want io.ErrShortWrite", err)
		}
		if got := pending.Load(); got != 1 {
			t.Fatalf("pending = %d, want 1 — exactly one decrement for the ONE complete response actually inside b[:n], not two", got)
		}
	})

	t.Run("zero-bytes-written", func(t *testing.T) {
		line := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n")

		var pending atomic.Int64
		pending.Store(1)

		underlying := &partialWriter{n: 0, err: io.ErrClosedPipe}
		pw := &pendingWriter{w: underlying, pending: &pending}

		if _, err := pw.Write(line); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("Write returned err=%v, want io.ErrClosedPipe", err)
		}
		if got := pending.Load(); got != 1 {
			t.Fatalf("pending = %d, want 1 — zero bytes actually written means zero decrements, even though the buffer contained a complete response line", got)
		}
	})
}

// TestPendingWriterDecrementsPerLineAcrossWriteBoundaries covers the two
// write-boundary cases: multiple responses in one Write, and one response
// split across two Writes. Exactly two t.Run subtests, contributing
// 1 + 2 = 3 PASS lines.
func TestPendingWriterDecrementsPerLineAcrossWriteBoundaries(t *testing.T) {
	t.Run("two-responses-one-write", func(t *testing.T) {
		var pending atomic.Int64
		pending.Store(2)

		var buf bytes.Buffer
		pw := &pendingWriter{w: &buf, pending: &pending}

		payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{}}` + "\n")
		if _, err := pw.Write(payload); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if got := pending.Load(); got != 0 {
			t.Fatalf("pending = %d, want 0 — a single Write carrying two responses must decrement twice", got)
		}
	})

	t.Run("one-response-split-across-two-writes", func(t *testing.T) {
		var pending atomic.Int64
		pending.Store(1)

		var buf bytes.Buffer
		pw := &pendingWriter{w: &buf, pending: &pending}

		part1 := []byte(`{"jsonrpc":"2.0","id":1,"resu`)
		part2 := []byte(`lt":{}}` + "\n")

		if _, err := pw.Write(part1); err != nil {
			t.Fatalf("Write part1: %v", err)
		}
		if got := pending.Load(); got != 1 {
			t.Fatalf("pending after mid-line Write = %d, want 1 — no complete line yet, so zero decrements", got)
		}

		if _, err := pw.Write(part2); err != nil {
			t.Fatalf("Write part2: %v", err)
		}
		if got := pending.Load(); got != 0 {
			t.Fatalf("pending after completing the split line = %d, want 0 — decremented exactly once, not zero and not twice", got)
		}
	})
}

// recordingWriter is a test double that records the exact byte sequence
// it receives, across every Write call, in order — used to prove
// pendingWriter never buffers a byte before forwarding it.
type recordingWriter struct {
	mu  sync.Mutex
	got []byte
}

func (r *recordingWriter) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, b...)
	return len(b), nil
}

// TestPendingWriterForwardsBeforeClassifying asserts the underlying writer
// receives every byte of every Write, in order, including a Write that
// ends mid-line — the classification buffer must never become a
// forwarding buffer.
func TestPendingWriterForwardsBeforeClassifying(t *testing.T) {
	var pending atomic.Int64
	rec := &recordingWriter{}
	pw := &pendingWriter{w: rec, pending: &pending}

	writes := [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"notifications/x"}` + "\n"),
		[]byte(`{"jsonrpc":"2.0","id":1,"resu`), // ends mid-line
		[]byte(`lt":{}}` + "\n"),
	}

	var want []byte
	for _, w := range writes {
		want = append(want, w...)
		if _, err := pw.Write(w); err != nil {
			t.Fatalf("Write(%q): %v", w, err)
		}
	}

	if !bytes.Equal(rec.got, want) {
		t.Fatalf("underlying writer received %q, want %q — a byte was held back or reordered before forwarding", rec.got, want)
	}
}

// chunkyWriter is a test double for the underlying stdout writer that is
// deliberately UNSAFE for concurrent Write calls: it appends to a plain,
// unguarded []byte in small chunks with a scheduling yield between them.
// If pendingWriter fails to serialize the ENTIRE underlying Write() call
// (not just its own classification buffer) inside one lock, two
// concurrent Write calls can interleave their chunk appends to this
// writer's unguarded state — go test -race reports a data race on it, and
// the recorded output can be corrupted (a line reassembled from two
// different writers' fragments). Under the fixed pendingWriter, only one
// goroutine is ever inside this Write() method at a time by construction,
// so this writer never needs its own lock — the absence of one here is
// what makes it a meaningful witness for
// TestPendingWriterSerializesWriteAndClassification.
type chunkyWriter struct {
	got []byte
}

func (c *chunkyWriter) Write(b []byte) (int, error) {
	const chunkSize = 3
	total := len(b)
	for len(b) > 0 {
		n := chunkSize
		if n > len(b) {
			n = len(b)
		}
		c.got = append(c.got, b[:n]...)
		b = b[n:]
	}
	return total, nil
}

// assertMutexPrecedesWriteInSource is a source-order check, not an
// eyeballed review: it reads server.go, strips comment-only lines, and
// asserts textually that pendingWriter.Write's p.mu.Lock() call precedes
// its call to the underlying p.w.Write(b) — the property that makes the
// mutex cover the underlying write, not just the classification buffer.
// It is positive-controlled: both anchor strings must actually be found
// inside pendingWriter.Write before their relative order means anything,
// so a rename of either that broke this check would fail loudly rather
// than passing vacuously.
func assertMutexPrecedesWriteInSource(t *testing.T) {
	t.Helper()

	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}

	var stripped strings.Builder
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		stripped.WriteString(line)
		stripped.WriteByte('\n')
	}
	body := stripped.String()

	const funcAnchor = "func (p *pendingWriter) Write(b []byte) (int, error) {"
	funcIdx := strings.Index(body, funcAnchor)
	if funcIdx < 0 {
		t.Fatalf("positive control failed: could not find %q in server.go", funcAnchor)
	}
	rest := body[funcIdx:]

	const lockAnchor = "p.mu.Lock()"
	const writeAnchor = "p.w.Write(b)"
	lockIdx := strings.Index(rest, lockAnchor)
	writeIdx := strings.Index(rest, writeAnchor)

	if lockIdx < 0 {
		t.Fatalf("positive control failed: could not find %q inside pendingWriter.Write", lockAnchor)
	}
	if writeIdx < 0 {
		t.Fatalf("positive control failed: could not find %q inside pendingWriter.Write", writeAnchor)
	}
	if lockIdx >= writeIdx {
		t.Fatalf("%s (offset %d) does not precede %s (offset %d) inside pendingWriter.Write — the mutex must cover the underlying write, not only the classification buffer", lockAnchor, lockIdx, writeAnchor, writeIdx)
	}
}

// TestPendingWriterSerializesWriteAndClassification runs concurrent Write
// calls under -race against chunkyWriter, an instrumented underlying
// writer that is itself unsafe for concurrent use. It asserts that every
// concurrently-written response line arrives at the underlying writer
// intact and exactly once — the property a buffer-only mutex does not
// provide, because it would let concurrent Write calls reach chunkyWriter
// unserialized, corrupting its unguarded state (caught by -race) and
// potentially reassembling a line from two different writers' fragments.
func TestPendingWriterSerializesWriteAndClassification(t *testing.T) {
	assertMutexPrecedesWriteInSource(t)

	const workers = 20

	var pending atomic.Int64
	pending.Store(workers)

	cw := &chunkyWriter{}
	pw := &pendingWriter{w: cw, pending: &pending}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			line := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":{}}`, i) + "\n")
			if _, err := pw.Write(line); err != nil {
				t.Errorf("Write: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := pending.Load(); got != 0 {
		t.Fatalf("pending = %d after %d concurrent responses, want 0 — each must decrement exactly once", got, workers)
	}

	lines := bytes.Split(bytes.TrimRight(cw.got, "\n"), []byte("\n"))
	if len(lines) != workers {
		t.Fatalf("underlying writer recorded %d lines, want %d — a buffer-only mutex would let concurrent Write calls interleave bytes at the sink, corrupting the line count", len(lines), workers)
	}

	seen := make(map[int]bool, workers)
	for _, line := range lines {
		var msg struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			t.Fatalf("underlying writer's line %q did not parse as valid JSON — a response was reassembled from two different writers' fragments: %v", line, err)
		}
		if seen[msg.ID] {
			t.Fatalf("id %d appeared twice in the underlying writer's output — a line was duplicated or corrupted by concurrent unsynchronized writes", msg.ID)
		}
		seen[msg.ID] = true
	}
	if len(seen) != workers {
		t.Fatalf("saw %d distinct ids, want %d", len(seen), workers)
	}
}

// TestPendingWriterNeverGoesNegative writes more responses than there were
// accepted calls and asserts the counter never goes negative, under
// -race, and that pendingUnderflows records every refused decrement.
func TestPendingWriterNeverGoesNegative(t *testing.T) {
	pendingUnderflows.Store(0) // isolate this test's count from any prior test in the same binary run

	const accepted = 5
	const attempts = 12 // strictly more decrement attempts than accepted calls

	var pending atomic.Int64
	pending.Store(accepted)

	var buf bytes.Buffer
	pw := &pendingWriter{w: &buf, pending: &pending}

	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		i := i
		go func() {
			defer wg.Done()
			line := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":{}}`, i) + "\n")
			if _, err := pw.Write(line); err != nil {
				t.Errorf("Write: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := pending.Load(); got != 0 {
		t.Fatalf("pending = %d after %d attempted decrements against %d accepted calls, want 0 (never negative)", got, attempts, accepted)
	}

	wantUnderflows := int64(attempts - accepted)
	if got := pendingUnderflows.Load(); got != wantUnderflows {
		t.Fatalf("pendingUnderflows = %d, want %d refused decrements (attempts=%d - accepted=%d)", got, wantUnderflows, attempts, accepted)
	}
}
