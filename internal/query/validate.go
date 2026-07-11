package query

// Documented ceilings for the numeric flags every query command exposes
// (--depth/--limit/--max-files). These bound BFS/scan/allocation work
// before it starts (V5 Input Validation, RESEARCH Pitfall 4,
// T-03-02-DoS): a caller — human or an untrusted/compromised MCP client —
// cannot force an unbounded traversal or allocation just by passing a
// large number.
const (
	// MaxDepth bounds impact/affected BFS depth. 50 comfortably exceeds
	// any realistic call-chain depth in a real codebase while keeping a
	// worst-case traversal small relative to the ≈4k-edge golden corpus
	// scale this phase targets (CONTEXT D-04).
	MaxDepth = 50

	// MaxLimit bounds how many result rows query/search/callers/callees
	// may return in one call.
	MaxLimit = 1000

	// MaxFiles bounds explore's per-call file-read fan-out.
	MaxFiles = 1000

	// defaultDepth is applied when a caller passes a non-positive depth
	// (clampDepth's "0 means default" convention, matching the CLI flags'
	// zero-value default).
	defaultDepth = 5
)

// clampDepth returns min(n, MaxDepth), treating n<=0 as defaultDepth.
//
// TODO(03-02 GREEN): unimplemented — RED stub.
func clampDepth(n int) int {
	return n
}

// validateLimit rejects n above MaxLimit with a clear error.
//
// TODO(03-02 GREEN): unimplemented — RED stub.
func validateLimit(n int) error {
	return nil
}

// validateMaxFiles rejects n above MaxFiles with a clear error.
//
// TODO(03-02 GREEN): unimplemented — RED stub.
func validateMaxFiles(n int) error {
	return nil
}

// ValidateKind rejects an unknown --kind value against the known
// node-kind set before any node scan (T-03-02-Kind, V5).
//
// TODO(03-02 GREEN): unimplemented — RED stub.
func ValidateKind(kind string) error {
	return nil
}
