package graphstore

import (
	"bytes"
	"strings"
	"testing"
)

// captureDiagWriter swaps diagWriter for a fresh bytes.Buffer for the
// duration of the calling test, restoring the previous value via
// t.Cleanup — the same test-only-seam capture pattern openLockRetrySleep
// already established in pebble_store.go.
func captureDiagWriter(t *testing.T) *bytes.Buffer {
	t.Helper()
	prev := diagWriter
	buf := &bytes.Buffer{}
	diagWriter = buf
	t.Cleanup(func() { diagWriter = prev })
	return buf
}

// TestQuietLoggerInfofDiscards asserts quietLogger.Infof (D-02: the
// WAL/compaction/memtable chatter pebble emits directly) writes nothing to
// the diagWriter seam — the unconditional discard this phase exists to add.
func TestQuietLoggerInfofDiscards(t *testing.T) {
	buf := captureDiagWriter(t)

	var l quietLogger
	l.Infof("Found %d WALs", 3)
	l.Infof("  - %s", "000001.log")

	if buf.Len() != 0 {
		t.Fatalf("quietLogger.Infof wrote %d bytes to diagWriter, want 0: %q", buf.Len(), buf.String())
	}
}

// TestQuietLoggerErrorfWritesProvenance asserts quietLogger.Errorf (D-04:
// preserved real diagnostic signal) writes exactly one provenance-prefixed
// line to diagWriter.
func TestQuietLoggerErrorfWritesProvenance(t *testing.T) {
	buf := captureDiagWriter(t)

	var l quietLogger
	l.Errorf("boom %d", 1)

	want := "codegraph: pebble: boom 1\n"
	if got := buf.String(); got != want {
		t.Fatalf("quietLogger.Errorf wrote %q, want %q", got, want)
	}
}

// TestQuietLoggerFatalfFormattingHelper asserts the Fatalf message-
// formatting path (factored into the same shared helper Errorf uses)
// produces a "codegraph: pebble: fatal: " prefixed line. This deliberately
// exercises the shared formatting helper directly and NEVER calls
// quietLogger.Fatalf itself — per RESEARCH.md Pitfall 3, Fatalf calls
// os.Exit(1) after formatting, and invoking it directly here would
// terminate the whole test binary rather than fail an assertion. This
// mirrors pebble's own base.InMemLogger.Fatalf test-double shape (formats
// and logs, never exits).
func TestQuietLoggerFatalfFormattingHelper(t *testing.T) {
	buf := captureDiagWriter(t)

	writeDiagLine(fatalPrefix, "invariant violated: %s", "MANIFEST not locked for writing")

	got := buf.String()
	if !strings.HasPrefix(got, "codegraph: pebble: fatal: ") {
		t.Fatalf("fatal-path formatting helper wrote %q, want prefix %q", got, "codegraph: pebble: fatal: ")
	}
	want := "codegraph: pebble: fatal: invariant violated: MANIFEST not locked for writing\n"
	if got != want {
		t.Fatalf("fatal-path formatting helper wrote %q, want %q", got, want)
	}
}
