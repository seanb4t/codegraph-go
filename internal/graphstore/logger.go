package graphstore

import (
	"fmt"
	"io"
	"os"

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
	fmt.Fprintf(diagWriter, prefix+format+"\n", args...)
}

// diagWriter is the test-only-seam convention already established by
// openLockRetrySleep in pebble_store.go: an unexported package-level var,
// defaulting to the production value (os.Stderr, per the repo-wide
// diagnostics rule, T-03-07-Leak / internal/mcp/server.go:63-66), with no
// exported setter. Tests reassign this var (via captureDiagWriter) to
// capture output; production behavior is unchanged.
var diagWriter io.Writer = os.Stderr
