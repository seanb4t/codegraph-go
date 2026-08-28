package wireoracle

import (
	"io"
	"testing"
	"time"
)

// TestCaptureArrivalLedgerPreservesWireOrder is 03-03-PLAN.md Task 1(b)'s
// evidence test: it drives scanArrivalLines — the same scan-and-timestamp
// primitive Capture's own stdout-reading goroutine uses (capture.go) —
// with a synthetic writer emitting a known line sequence, and asserts the
// returned ledger reproduces that emission order EXACTLY.
//
// The emitted sequence is deliberately NOT ascending by its embedded
// JSON-RPC id (1, 3, 2, notification, 5): a ledger that silently sorted or
// bucketed lines (by id, by "is this a response", or by anything else)
// would fail this test, while a ledger that does nothing but scan and
// timestamp in arrival order passes it. This is what turns "I looked at
// the capture layer and it doesn't reorder anything" from a one-time
// manual read into a durable, checkable claim — the single test this
// plan's own <verify> block binds to, rather than the pre-existing
// TestFrozenTranscriptsMatch/toolslist-repeat subtest, whose PASS or FAIL
// both satisfied the cycle-1 review's guard without demanding any
// investigation at all.
func TestCaptureArrivalLedgerPreservesWireOrder(t *testing.T) {
	emitted := []string{
		`{"jsonrpc":"2.0","id":1,"result":"a"}`,
		`{"jsonrpc":"2.0","id":3,"result":"c"}`,
		`{"jsonrpc":"2.0","id":2,"result":"b"}`,
		`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`,
		`{"jsonrpc":"2.0","id":5,"result":"e"}`,
	}

	// A synthetic writer, not a synthetic in-memory buffer already
	// containing every line: pr/pw is an io.Pipe, and the writer goroutine
	// below emits one line at a time with a small forced delay between
	// writes, mirroring the real shape Capture's own scanner goroutine
	// reads against (a subprocess's stdout pipe, written to
	// incrementally, not all at once before the first byte is read).
	pr, pw := io.Pipe()
	go func() {
		for _, line := range emitted {
			// Small delay so consecutive writes cannot land in the same
			// bufio.Scanner.Scan() call by coincidence — each line must
			// genuinely arrive as its own scan, exercising the same
			// one-line-per-channel-send shape drainUntil relies on.
			time.Sleep(2 * time.Millisecond)
			if _, err := pw.Write([]byte(line + "\n")); err != nil {
				return
			}
		}
		_ = pw.Close()
	}()

	got, err := scanArrivalLines(pr)
	if err != nil {
		t.Fatalf("scanArrivalLines: %v", err)
	}
	if len(got) != len(emitted) {
		t.Fatalf("scanArrivalLines returned %d lines, want %d: %+v", len(got), len(emitted), got)
	}
	for i, line := range got {
		if string(line.Raw) != emitted[i] {
			t.Errorf("line %d: got %q, want %q — emission order was NOT preserved", i, line.Raw, emitted[i])
		}
		if line.Arrived.IsZero() {
			t.Errorf("line %d: Arrived timestamp is zero", i)
		}
		if i > 0 && line.Arrived.Before(got[i-1].Arrived) {
			t.Errorf("line %d: Arrived %v is before line %d's Arrived %v — timestamps must be non-decreasing in scan order",
				i, line.Arrived, i-1, got[i-1].Arrived)
		}
	}
}
