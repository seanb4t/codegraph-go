package goextract

import "github.com/seanb4t/codegraph-go/internal/schema"

// Node kinds this extractor emits (D-06). No other kind is ever produced —
// in particular, no "field" node is emitted (ratified skip, 02-RESEARCH.md
// Open Question 3).
const (
	KindFile      = "file"
	KindFunction  = "function"
	KindMethod    = "method"
	KindStruct    = "struct"
	KindInterface = "interface"
	KindTypeAlias = "type_alias"
	KindConstant  = "constant"
	KindVariable  = "variable"

	// KindRoute is a Phase 5 addition (LANG-07, D-08): a framework-detected
	// HTTP route (path + method), linked to its handler symbol via a
	// heuristic-provenance edge. Additive within SchemaVersion 1 — no
	// pre-existing Kind* constant above is renamed or removed.
	KindRoute = "route"
)

// UnresolvedRef.Kind values. "calls" and "imports" and "embeds" are the
// three cross-file reference kinds named in RES-01/D-04. "contains" is a
// fourth, narrower kind this extractor also emits: a method whose receiver
// type is declared in a DIFFERENT file than the method itself cannot yield
// an IntraEdge (both ids must be known within a single file's walk), so the
// containment is instead recorded here for Pass 2 to resolve once it has
// seen every file's type declarations.
const (
	RefKindCalls    = "calls"
	RefKindImports  = "imports"
	RefKindEmbeds   = "embeds"
	RefKindContains = "contains"
)

// EdgeKindImplements is a Phase 5 addition (RES-02, D-06): resolve.go sets
// this literal on schema.Edge.Kind for two distinct synthesis paths — the
// declared-implements promotion (Pattern 2: a RefKindEmbeds-shaped
// unresolved ref whose resolved target is an interface node) and Go's
// structural method-set match (dispatch.SynthesizeImplements, Pattern 3).
// Named here (not left as a bare string literal, unlike "calls"/"embeds"/
// "contains"/"imports" today) so dispatch.go, resolve.go's promotion
// check, and query/traverse.go's dispatch-index filter share ONE
// definition rather than three copies of "implements". Distinct from
// RefKindEmbeds — do not repurpose that constant (D-06/RES-03: additive
// only, no existing edge kind is renamed).
const EdgeKindImplements = "implements"

// MethodSpec is a bare (name, parameter-count) signature captured during
// extraction. Go's implicit interface-satisfaction synthesis (RES-02,
// Phase 5 Pattern 3) compares a struct's method set against an
// interface's method-spec set by (Name, Arity), not full type identity —
// D-06's bounded-matching discipline (name+arity), never a full Go type
// checker.
type MethodSpec struct {
	Name  string
	Arity int32
}

// ExtractedNode wraps a fully-formed schema.Node produced during the
// tree-walk. Its id is already computed via nodeid.NodeID, so Pass 2 never
// needs to recompute or rename it.
type ExtractedNode struct {
	Node *schema.Node
}

// IntraEdge wraps a schema.Edge whose source and target ids are BOTH known
// at extraction time — a same-file relationship (file->symbol contains,
// or a method's type->method contains edge when the receiver type is
// declared in the same file). Cross-file relationships are never recorded
// here; they become UnresolvedRefs instead.
type IntraEdge struct {
	Edge *schema.Edge
}

// UnresolvedRef is a reference this file's extraction could not resolve to
// a target node id on its own, because doing so requires information from
// other files (a global symbol index) that Pass 1 — being embarrassingly
// parallel over single files — does not have. Pass 2 (resolve, a later
// plan) builds that global index and turns these into calls/imports/embeds
// (or type->method contains) edges.
type UnresolvedRef struct {
	// FromID is the node id of the enclosing symbol (the calling
	// function/method, the embedding struct/interface, the containing
	// file for an import, or the method itself for a cross-file
	// "contains" ref) that this reference originates from.
	FromID string
	// Name is the referenced identifier (a callee name, an embedded
	// type's name, an import path for "imports" refs, or a receiver
	// type's name for "contains" refs).
	Name string
	// PkgAlias is the local package alias for a qualified reference
	// (pkg.Fn(), pkg.Type embedding) — empty for an unqualified
	// reference.
	PkgAlias string
	// Kind is one of RefKindCalls, RefKindImports, RefKindEmbeds, or
	// RefKindContains.
	Kind string
	// Line and Col are the 1-based line / 0-based column of the
	// reference's call/use site (D-04).
	Line, Col int32
}

// FileResult is the complete Pass-1 intermediate for one Go source file:
// every node it declares, every intra-file edge those nodes participate
// in, and every cross-file reference Pass 2 must resolve.
//
// Err is non-nil when this file was SKIPPED (e.g. parser.ErrSourceTooLarge
// or any other single-file extraction failure) — a skip is recorded here,
// never surfaced as Extract's own return error, so one bad file never
// aborts the run (RESEARCH Pitfall 4 / threat T-02-03).
type FileResult struct {
	ImportPath string
	RelPath    string
	Language   string

	Nodes      []ExtractedNode
	IntraEdges []IntraEdge
	Unresolved []UnresolvedRef

	// Imports maps each import_spec's effective local alias (explicit
	// alias, "." for dot imports, "_" for blank imports, or the default
	// last-path-segment name) to its import path.
	Imports map[string]string

	// InterfaceMethods records, per interface node this file declares, the
	// method specs declared directly in that interface's own body (Phase 5
	// RES-02/Pattern 3). Embedded interfaces (`type A interface { B }`)
	// are NOT flattened here — dispatch.SynthesizeImplements composes an
	// interface's transitive method set itself, via the "embeds" edges
	// Pass 2 already resolves between two interface nodes.
	InterfaceMethods map[string][]MethodSpec

	// MethodArity maps a method node's id to its declared parameter count
	// (Phase 5 RES-02/Pattern 3). Keyed by the method's OWN node id (not
	// scoped by struct or file) because Pass 2 resolves a struct's full,
	// possibly cross-file, method set via "contains" edges — exactly like
	// TestResolve_CrossFileMethodContainment — so dispatch synthesis looks
	// arity up post-resolution, by method id, not at extraction time.
	MethodArity map[string]int32

	// ContentHash is hex(sha256(source bytes)) — SHA-256 per D-02a, never
	// MD5 — computed regardless of whether extraction below it succeeds.
	ContentHash string

	// MtimeUnixNs and SizeBytes carry the on-disk file's stat info at
	// extraction time (Phase 4 D-01a), copied verbatim from the
	// DiscoveredFile that produced this result. Pass 2 stamps these onto
	// the committed schema.File record so Sync's stat pre-filter has
	// something cheap to compare against on a later invocation.
	MtimeUnixNs int64
	SizeBytes   int64

	// Err is set instead of returning a non-nil error from Extract when
	// this specific file could not be parsed/extracted.
	Err error
}
