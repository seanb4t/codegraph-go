// Package parser defines a narrow, backend-swappable boundary for turning
// source bytes into a syntax tree, independent of storage concerns.
package parser

import "errors"

// MaxSourceBytes is the default file-size ceiling enforced before any bytes
// reach a backend's underlying parser. This is the mitigation for the
// pathological-input denial-of-service threat (Security Domain V5): a
// single multi-megabyte minified line handed to a tree-sitter grammar's
// external C scanner can consume unbounded time/memory. Implementations
// MUST reject input over this ceiling (or a caller-configured ceiling)
// with ErrSourceTooLarge BEFORE invoking the underlying C/WASM parser.
const MaxSourceBytes = 4 * 1024 * 1024 // 4 MiB

// ErrSourceTooLarge is returned by Parse when len(source) exceeds the
// configured ceiling. Implementations MUST return this sentinel (checked
// via errors.Is) without calling into the underlying C/WASM parser.
var ErrSourceTooLarge = errors.New("parser: source exceeds size ceiling")

// Tree is a backend-neutral syntax tree handle. Call sites never touch a
// backend-specific tree type directly (e.g. *tree_sitter.Tree) — every
// backend wraps its native tree behind this opaque type so storage and
// future extractors can be written against a single shape regardless of
// which Parser implementation produced the tree.
type Tree struct {
	// inner holds the backend-specific tree value. It is intentionally
	// untyped (any) here in the shared package; each backend package is
	// responsible for storing and retrieving its own concrete tree type,
	// typically by embedding/wrapping this Tree rather than reaching into
	// this field directly from outside the backend package.
	inner any
}

// NewTree wraps a backend-specific tree value in the shared, opaque Tree
// type. Backend packages call this after a successful parse.
func NewTree(inner any) *Tree {
	return &Tree{inner: inner}
}

// Inner returns the backend-specific tree value that was wrapped by
// NewTree. Callers that need to unwrap it must type-assert against the
// concrete type their own backend produced.
func (t *Tree) Inner() any {
	if t == nil {
		return nil
	}
	return t.inner
}

// Parser is the narrow, backend-swappable seam between arbitrary source
// bytes and a syntax tree (D-05b). Both the CGo tree-sitter backend
// (internal/parser/cgo) and any future wazero-WASM backend implement this
// exact shape so callers (storage, extractors) never depend on a specific
// backend's types.
//
// Security contract (Security Domain V5 / threat T-01-03): implementations
// MUST reject source input larger than their configured ceiling (see
// MaxSourceBytes) by returning ErrSourceTooLarge, and MUST perform that
// check BEFORE handing the bytes to the underlying C or WASM parser. This
// bounds the pathological-input denial-of-service surface (e.g. a single
// multi-megabyte minified line) at the seam, before any backend-specific
// parsing cost is paid.
//
// Crash-isolation contract (threat T-01-01, accepted for Phase 1): a
// C-level segfault inside a tree-sitter grammar's external scanner (for
// example Python's INDENT/DEDENT scanner) runs in-process with no memory
// isolation from the Go host. This is NOT a Go panic and is NOT
// recoverable via Go's recover() — callers and implementations MUST NOT
// design any crash-isolation logic around recover() for this path. The
// CGo backend's crash-isolation behavior under adversarial input is
// measured (not eliminated) by the Plan 01-07 spike; a subprocess
// isolation boundary remains a Phase 2+ option if that risk proves
// unacceptable in practice.
//
// Resource contract: implementations allocate C or WASM-backed memory for
// parsers/trees. Callers MUST call Close() exactly once when done with a
// Parser to free those resources; Close() must be safe to call even if no
// Parse call was made.
type Parser interface {
	// Parse produces a Tree from source bytes. If oldTree is non-nil,
	// implementations SHOULD perform an incremental reparse using it as a
	// starting point. Implementations MUST enforce the size ceiling
	// (returning ErrSourceTooLarge) before doing any backend-specific
	// parsing work.
	Parse(source []byte, oldTree *Tree) (*Tree, error)

	// Close releases any resources (C memory, WASM instances) held by the
	// parser. Safe to call once; implementations should make repeat calls
	// a no-op rather than double-free.
	Close() error
}
