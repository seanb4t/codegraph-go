// Package cgo implements the production CGo tree-sitter backend (Option A)
// for the parser.Parser interface, covering the two spike-partner languages
// (D-05): Go (LANG-01) and Python (carries a real external C scanner). If
// the Plan 01-07 spike selects Option A, this is the single documented CGo
// exception feeding DIST-05.
package cgo

import (
	"errors"
	"sync"
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/seanb4t/codegraph-go/internal/parser"
)

// ErrParseFailed is returned when the underlying tree-sitter C call
// returns no tree (a NULL *TSTree), so callers never receive a
// non-nil *parser.Tree wrapping a nil native tree.
var ErrParseFailed = errors.New("cgo: tree-sitter parse returned no tree")

// CGoParser implements parser.Parser using the CGo tree-sitter bindings
// for a single language grammar.
type CGoParser struct {
	inner     *tree_sitter.Parser
	closeOnce sync.Once
}

// NewGoParser returns a CGoParser configured for the Go grammar.
func NewGoParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_go.Language())
}

// NewPythonParser returns a CGoParser configured for the Python grammar.
// Python exercises the external C scanner path (its INDENT/DEDENT
// tokenizer), the crash-isolation dimension the Plan 01-07 spike measures.
func NewPythonParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_python.Language())
}

func newCGoParser(languagePtr unsafe.Pointer) (*CGoParser, error) {
	p := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(languagePtr)
	if err := p.SetLanguage(lang); err != nil {
		p.Close()
		return nil, err
	}
	return &CGoParser{inner: p}, nil
}

// Parse enforces the shared size ceiling (parser.MaxSourceBytes) before
// handing bytes to the underlying C parser (Security Domain V5 / threat
// T-01-03), then performs an incremental reparse via the tree-sitter
// bindings if oldTree is non-nil.
func (p *CGoParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	if len(source) > parser.MaxSourceBytes {
		return nil, parser.ErrSourceTooLarge
	}

	var native *tree_sitter.Tree
	if oldTree != nil {
		if t, ok := oldTree.Inner().(*tree_sitter.Tree); ok {
			native = t
		}
	}

	result := p.inner.Parse(source, native)
	if result == nil {
		return nil, ErrParseFailed
	}
	return parser.NewTree(result, result.Close), nil
}

// Close frees the underlying C parser's allocations. Safe to call once
// or concurrently; repeat calls are a no-op.
func (p *CGoParser) Close() error {
	p.closeOnce.Do(func() {
		p.inner.Close()
	})
	return nil
}
