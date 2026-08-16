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

	// MaxAffectedFiles bounds how many changed-file paths `affected
	// --stdin` (an explicitly untrusted input surface, SURF-04/
	// T-08-05-01) may ingest before Engine.Affected ever runs its bounded
	// graph scan (CR-01, T-03-02-DoS). Deliberately its own constant
	// rather than reusing MaxFiles — MaxFiles bounds Explore's per-call
	// file-READ fan-out (an expensive per-file verbatim-source read);
	// affected's stdin path is a cheap string-dedup/BFS-seed operation
	// that can tolerate a much larger ceiling before becoming a DoS
	// concern, so the two must not be silently coupled.
	MaxAffectedFiles = 10000

	// defaultDepth is applied when a caller passes a non-positive depth
	// (clampDepth's "0 means default" convention, matching the CLI flags'
	// zero-value default). SURF-01/D-02: is the documented
	// impact default of 2 — changed here in the shared engine (not a
	// per-surface CLI/MCP default) so codegraph impact and the
	// codegraph_impact MCP tool inherit depth-2 together. MaxDepth's
	// clamp ceiling is unchanged.
	defaultDepth = 2

	// defaultMaxFiles is applied when a caller passes a non-positive
	// max-files (clampMaxFiles's "0 means default" convention, mirroring
	// clampDepth — Explore's per-file verbatim-source read is expensive
	// enough that "unlimited by default" would be a DoS footgun, unlike
	// Files' "0 means unlimited" browse-only convention).
	defaultMaxFiles = 5

	// defaultAffectedDepth is applied when a caller passes a non-positive
	// depth to Affected (clampAffectedDepth's "0 means default"
	// convention). SURF-04/D-05: deliberately distinct from defaultDepth
	// — the documented `affected` command defaults to depth 5,
	// while `impact` defaults to depth 2 (defaultDepth, SURF-01). A naive
	// reuse of clampDepth/defaultDepth for affected would silently apply
	// impact's smaller default. Shares MaxDepth as its ceiling.
	defaultAffectedDepth = 5
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

// clampAffectedDepth mirrors clampDepth's shape but with
// defaultAffectedDepth (5) instead of defaultDepth (2) — SURF-04/D-05
// deliberately does NOT reuse clampDepth, so affected's documented
// default cannot be silently overwritten by impact's smaller one.
// Shares MaxDepth as its ceiling.
func clampAffectedDepth(n int) int {
	if n <= 0 {
		n = defaultAffectedDepth
	}
	if n > MaxDepth {
		return MaxDepth
	}
	return n
}

// validateLimit rejects n above MaxLimit, and (WR-02) a negative n, with a
// clear error instead of silently truncating or falling back to a
// default (V5 — the caller should know its request was out-of-range, not
// receive a silently-smaller result set). n==0 is still accepted as
// "caller did not set a limit"; callers apply their own default
// downstream. Rejecting negatives outright unifies this with
// validateFilesDepth's stricter "reject absurd input" posture — before
// this fix, a negative --limit/--depth/--max-files was silently treated
// as "unset" here and in clampDepth/clampMaxFiles, while files.go's
// --depth explicitly rejected negatives: a genuine cross-command
// consistency gap (WR-02).
func validateLimit(n int) error {
	if n < 0 {
		return fmt.Errorf("query: limit %d must be non-negative", n)
	}
	if n > MaxLimit {
		return fmt.Errorf("query: limit %d exceeds maximum %d", n, MaxLimit)
	}
	return nil
}

// validateMaxFiles rejects n above MaxFiles, and (WR-02) a negative n,
// with a clear error, mirroring validateLimit's contract.
func validateMaxFiles(n int) error {
	if n < 0 {
		return fmt.Errorf("query: max-files %d must be non-negative", n)
	}
	if n > MaxFiles {
		return fmt.Errorf("query: max-files %d exceeds maximum %d", n, MaxFiles)
	}
	return nil
}

// validateDepth rejects a negative depth (WR-02), mirroring
// validateLimit/validateMaxFiles/validateFilesDepth's "reject absurd
// input outright" posture. 0 is accepted and clampDepth treats it as
// "caller did not set a depth" (defaultDepth). An explicit depth above
// MaxDepth is deliberately NOT rejected here — clampDepth silently caps
// it, matching Impact's existing "absurdly large depth is clamped, not
// unbounded" contract (TestImpact); only the negative-value convention is
// unified across depth/max-files/limit by this fix.
func validateDepth(n int) error {
	if n < 0 {
		return fmt.Errorf("query: depth %d must be non-negative", n)
	}
	return nil
}

// ValidateAffectedFiles rejects a files slice longer than MaxAffectedFiles
// with a clear error (CR-01), mirroring validateLimit/validateMaxFiles's
// "reject outright rather than silently truncate" posture — a caller (or
// an untrusted/compromised MCP client, or a hostile CI diff) that tries to
// grow `affected --stdin`'s input past this ceiling gets an explicit
// error instead of an ever-growing, unbounded seen/files/fileSet
// allocation. Exported so internal/cli/affected.go's collectAffectedFiles
// can enforce the cap as it reads stdin, before Engine.Affected is ever
// called.
//
// Unlike its unexported siblings above, this message carries no "query: "
// prefix: those are internal-only, whereas this one is exported precisely
// so the CLI can surface it verbatim under its own "affected: " prefix.
// Prefixing here would render as "affected: query: N files exceeds
// maximum M", leaking an internal package name into user-facing output.
func ValidateAffectedFiles(n int) error {
	if n > MaxAffectedFiles {
		return fmt.Errorf("%d files exceeds maximum %d", n, MaxAffectedFiles)
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
