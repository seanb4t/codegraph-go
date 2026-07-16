package graphstore

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// captureDiagWriter swaps diagWriter for a fresh bytes.Buffer for the
// duration of the calling test, restoring the previous value via
// t.Cleanup — the same test-only-seam capture pattern openLockRetrySleep
// already established in pebble_store.go. Swap/restore go through the
// setDiagWriter accessor (not a bare assignment) so the seam stays
// race-safe (WR-03) if t.Parallel() is ever added to this package.
func captureDiagWriter(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := setDiagWriter(buf)
	t.Cleanup(func() { setDiagWriter(prev) })
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

// TestOpenInjectsQuietLogger is the D-08 mutation-proof wiring test: it
// opens a real store at a t.TempDir() directory, drives a real write/flush/
// close cycle (which provokes pebble's own Infof surface — open.go's
// "Found %d WALs" fires on EVERY Open, even a brand-new empty store, per
// RESEARCH.md), and asserts zero bytes of that noise reached a captured
// buffer.
//
// Why this is genuinely mutation-proof (not just re-asserting a private
// replica): capturing diagWriter alone would NOT detect a revert of
// pebble_store.go:147 back to &pebble.Options{} — pebble's base.
// DefaultLogger (installed by EnsureDefaults when Options.Logger is left
// nil) never touches diagWriter at all; it writes via stdlib log.Output(2,
// ...) instead. So this test ALSO redirects the stdlib `log` package's
// default output (log.SetOutput) into the SAME buffer diagWriter is
// pointed at. With both seams draining into one buffer: if line 147 is
// reverted, pebble's real "Found %d WALs" Infof call lands in this buffer
// via log.SetOutput and the zero-bytes assertion below fails; with
// quietLogger correctly wired, Infof is discarded at the source and the
// buffer stays empty. This is what closes the mutation-proof gap (D-08) —
// manually reverting line 147 and re-running this test is the documented
// verification step (see 04-01-SUMMARY.md).
func TestOpenInjectsQuietLogger(t *testing.T) {
	buf := captureDiagWriter(t)

	prevFlags := log.Flags()
	log.SetOutput(buf)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(prevFlags)
	})

	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(&schema.Meta{SchemaVersion: 1}); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if buf.Len() != 0 {
		t.Fatalf("real Open/write/flush/close cycle produced %d bytes of pebble log noise — quiet logger not wired at the Open seam?: %q", buf.Len(), buf.String())
	}
}

// TestQuietLoggerSilencesStoreActivity is the D-08 control paired with
// TestOpenInjectsQuietLogger above: it proves the capture mechanism itself
// works — a directly-invoked quietLogger.Errorf still reaches diagWriter —
// so the zero-bytes assertion in the wiring test reflects genuine
// silencing of pebble's Infof surface, not an unobserved/miswired buffer
// (D-02: preservation, not blanket silence).
func TestQuietLoggerSilencesStoreActivity(t *testing.T) {
	buf := captureDiagWriter(t)

	var l quietLogger
	l.Errorf("control-check")

	if !strings.Contains(buf.String(), "control-check") {
		t.Fatalf("direct quietLogger.Errorf did not reach diagWriter — capture seam broken: %q", buf.String())
	}
}
