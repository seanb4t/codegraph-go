package graphstore

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cockroachdb/pebble/v2"
)

// quietLogger implements pebble.Logger (D-01): Infof is the WAL/compaction/
// memtable chatter pebble emits directly (open.go's "Found %d WALs",
// compaction_picker.go's "pickAuto: ..."), discarded unconditionally (D-02).
// Errorf is real diagnostic signal (background-error/metrics-error paths)
// and is preserved via the diagWriter seam with a provenance prefix. Fatalf
// preserves pebble's own semantics: message out, then exit — pebble only
// calls Fatalf on invariant violations where continuing is unsafe (e.g.
// version_set.go's "MANIFEST not locked for writing"), so softening it
// would hide exactly the corruption signal this phase must never hide
// (ROADMAP guardrail, D-02).
type quietLogger struct{}

var _ pebble.Logger = quietLogger{}

// errorPrefix / fatalPrefix are the provenance prefixes writeDiagLine
// prepends for Errorf and Fatalf respectively (D-04). Exact wording is
// Claude's discretion per CONTEXT.md; the "codegraph: pebble: " root is
// kept so every pebble-originated diagnostic line is unambiguously
// attributable.
const (
	errorPrefix = "codegraph: pebble: "
	fatalPrefix = "codegraph: pebble: fatal: "
)

func (quietLogger) Infof(format string, args ...any) {}

func (quietLogger) Errorf(format string, args ...any) {
	writeDiagLine(errorPrefix, format, args...)
}

// Fatalf calls the shared formatting helper THEN os.Exit(1) as a final,
// reviewed-but-untested statement — do NOT soften this (D-02, RESEARCH
// Pitfall 3). No test invokes Fatalf directly; logger_test.go exercises
// writeDiagLine's formatting behavior instead, since a direct call here
// would terminate the test binary via os.Exit rather than fail an
// assertion.
func (quietLogger) Fatalf(format string, args ...any) {
	writeDiagLine(fatalPrefix, format, args...)
	os.Exit(1)
}

// writeDiagLine formats prefix+format+args as one line and writes it to
// diagWriter. Shared by Errorf and Fatalf so their formatting behavior is
// tested identically without ever invoking Fatalf's os.Exit(1) call.
func writeDiagLine(prefix, format string, args ...any) {
	fmt.Fprintf(getDiagWriter(), prefix+format+"\n", args...)
}

// diagWriter is the test-only-seam convention already established by
// openLockRetrySleep in pebble_store.go: an unexported package-level var,
// defaulting to the production value (os.Stderr, per the repo-wide
// diagnostics rule, T-03-07-Leak / internal/mcp/server.go:63-66), with no
// exported setter. Tests reassign this var (via captureDiagWriter, through
// the setDiagWriter accessor below) to capture output; production behavior
// is unchanged.
//
// diagWriterMu guards diagWriter itself (not the io.Writer's own internal
// state — a *bytes.Buffer written from only one goroutine at a time is
// still safe). This is a latent-footgun fix (WR-03): today no test in this
// package calls t.Parallel() and quietLogger.Errorf/Fatalf never fires
// concurrently with a capture window, but the mu RWMutex convention
// pebbleStore already uses elsewhere in this package is applied here too so
// the seam stays race-safe if either assumption is ever violated.
var (
	diagWriterMu sync.RWMutex
	diagWriter   io.Writer = os.Stderr
)

// getDiagWriter returns the current diagWriter under a read lock.
func getDiagWriter() io.Writer {
	diagWriterMu.RLock()
	defer diagWriterMu.RUnlock()
	return diagWriter
}

// setDiagWriter installs w as the current diagWriter under a write lock and
// returns the previous value, so callers (captureDiagWriter) can restore it
// on t.Cleanup.
func setDiagWriter(w io.Writer) io.Writer {
	diagWriterMu.Lock()
	defer diagWriterMu.Unlock()
	prev := diagWriter
	diagWriter = w
	return prev
}
