package uiserver

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// diagPrefix is the provenance prefix every diagnostic line this package
// emits carries, mirroring internal/graphstore/logger.go's errorPrefix
// discipline: the "codegraph: " root keeps a uiserver-originated line
// unambiguously attributable, and the second segment names the package
// that produced it.
const diagPrefix = "codegraph: uiserver: "

// diagWriter is this package's server-side diagnostic sink. It is the
// unexported package-level test-seam convention already established by
// internal/graphstore's diagWriter (and openLockRetrySleep before it): a
// var defaulting to the production value with NO exported setter, so
// production behavior is fixed while tests can capture what was written.
//
// os.Stderr, never os.Stdout, per the repo-wide diagnostics rule
// (T-03-07-Leak): stdout is reserved for the MCP JSON-RPC transport.
// uiserver is not one of internal/graphstore/archtest's six
// serve-reachable guarded packages, but the rule is a repo-wide one and
// is followed here for the same reason.
//
// diagWriterMu guards diagWriter itself. Unlike graphstore's, this seam
// is written from HTTP handler goroutines, which genuinely do run
// concurrently — the lock is load-bearing here, not merely conventional.
var (
	diagWriterMu sync.RWMutex
	diagWriter   io.Writer = os.Stderr
)

// writeDiagLine formats diagPrefix+format+args as one line and writes it
// to the current diagWriter. It is the ONLY way this package emits a
// server-side diagnostic, so a test that captures the seam captures
// everything.
func writeDiagLine(format string, args ...any) {
	diagWriterMu.RLock()
	w := diagWriter
	diagWriterMu.RUnlock()
	fmt.Fprintf(w, diagPrefix+format+"\n", args...)
}

// setDiagWriter installs w as the current diagWriter and returns the
// previous value, so a test can restore it on t.Cleanup. Unexported, and
// used only from this package's own tests.
func setDiagWriter(w io.Writer) io.Writer {
	diagWriterMu.Lock()
	defer diagWriterMu.Unlock()
	prev := diagWriter
	diagWriter = w
	return prev
}
