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
	tree_sitter_c_sharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

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

// NewJavaParser returns a CGoParser configured for the Java grammar
// (priority-4, LANG-02). Java's grammar carries no external C scanner.
func NewJavaParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_java.Language())
}

// NewCSharpParser returns a CGoParser configured for the C# grammar
// (priority-4, LANG-03). C#'s grammar carries an external C scanner —
// see parser.Parser's crash-isolation contract.
func NewCSharpParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_c_sharp.Language())
}

// NewJavaScriptParser returns a CGoParser configured for the JavaScript
// grammar (priority-4, LANG-05). Carries an external C scanner.
func NewJavaScriptParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_javascript.Language())
}

// NewTypeScriptParser returns a CGoParser configured for the TypeScript
// grammar (priority-4, LANG-05). Carries an external C scanner. The
// tree-sitter-typescript module ships two grammars in one repo
// (typescript/ and tsx/ subdirs) — this constructor uses the typescript
// accessor; see NewTSXParser for the sibling .tsx grammar.
func NewTypeScriptParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_typescript.LanguageTypescript())
}

// NewTSXParser returns a CGoParser configured for the TSX grammar
// (priority-4, LANG-05) — the JSX-aware sibling of NewTypeScriptParser,
// both bindings shipped by the same tree-sitter-typescript module.
func NewTSXParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_typescript.LanguageTSX())
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
