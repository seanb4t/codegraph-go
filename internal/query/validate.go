package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

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

	// defaultMaxFiles is applied when a caller passes a non-positive
	// max-files (clampMaxFiles's "0 means default" convention, mirroring
	// clampDepth — Explore's per-file verbatim-source read is expensive
	// enough that "unlimited by default" would be a DoS footgun, unlike
	// Files' "0 means unlimited" browse-only convention).
	defaultMaxFiles = 5
)

// clampDepth returns min(n, MaxDepth), treating n<=0 as defaultDepth
// rather than "unbounded" or "zero traversal" — a caller that omits
// --depth gets a small, useful default instead of an error.
func clampDepth(n int) int {
	if n <= 0 {
		n = defaultDepth
	}
	if n > MaxDepth {
		return MaxDepth
	}
	return n
}

// validateLimit rejects n above MaxLimit with a clear error instead of
// silently truncating (V5 — the caller should know its request was
// out-of-range, not receive a silently-smaller result set). n<=0 is
// accepted as "caller did not set a limit"; callers apply their own
// default downstream.
func validateLimit(n int) error {
	if n > MaxLimit {
		return fmt.Errorf("query: limit %d exceeds maximum %d", n, MaxLimit)
	}
	return nil
}

// validateMaxFiles rejects n above MaxFiles with a clear error, mirroring
// validateLimit's contract.
func validateMaxFiles(n int) error {
	if n > MaxFiles {
		return fmt.Errorf("query: max-files %d exceeds maximum %d", n, MaxFiles)
	}
	return nil
}

// clampMaxFiles returns min(n, MaxFiles), treating n<=0 as defaultMaxFiles
// (03-06) — mirrors clampDepth's "0 means a small useful default" shape.
// Callers must run validateMaxFiles first so an explicit out-of-range
// request is rejected rather than silently clamped (T-03-06-DoS).
func clampMaxFiles(n int) int {
	if n <= 0 {
		n = defaultMaxFiles
	}
	if n > MaxFiles {
		return MaxFiles
	}
	return n
}

// knownKinds is the authoritative --kind allow-list ValidateKind checks
// against: every schema.Node kind the Phase-2 extractor can emit
// (internal/indexer/goextract's Kind* constants, D-06) plus the synthetic
// "package" pseudo-node kind (internal/indexer/resolve.go's unexported
// kindPackage — its value is duplicated here as a string literal, not
// imported, because it is deliberately unexported by internal/indexer and
// this comment is what ties the two together; if kindPackage's value ever
// changes, this literal must change with it). Building the set from the
// extractor's own constants (rather than a hand-typed literal list for the
// other eight kinds) means this set cannot silently drift from the
// vocabulary goextract actually produces (T-03-02-Kind).
var knownKinds = map[string]bool{
	goextract.KindFile:      true,
	goextract.KindFunction:  true,
	goextract.KindMethod:    true,
	goextract.KindStruct:    true,
	goextract.KindInterface: true,
	goextract.KindTypeAlias: true,
	goextract.KindConstant:  true,
	goextract.KindVariable:  true,
	"package":               true, // internal/indexer/resolve.go's kindPackage
}

// ValidateKind rejects an unknown --kind value against knownKinds before
// any node scan (T-03-02-Kind, V5): the empty string passes (no filter —
// query/search's default "match every kind" behavior), any known kind
// passes, and anything else returns an error naming the full allowed set
// so the caller can self-correct instead of silently scanning to an
// always-empty result.
func ValidateKind(kind string) error {
	if kind == "" {
		return nil
	}
	if knownKinds[kind] {
		return nil
	}
	allowed := make([]string, 0, len(knownKinds))
	for k := range knownKinds {
		allowed = append(allowed, k)
	}
	sort.Strings(allowed)
	return fmt.Errorf("query: unknown kind %q — allowed kinds: %s", kind, strings.Join(allowed, ", "))
}
