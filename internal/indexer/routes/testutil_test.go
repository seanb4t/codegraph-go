package routes

import (
	"os"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// stubResolver is a test-only HandlerResolver keyed by name and by line —
// every Walk test builds one directly from the fixture's own known
// declaration lines/names rather than depending on a real per-language
// extractor (this package must not import internal/indexer, which would
// be a cycle: internal/indexer imports internal/indexer/routes, not the
// reverse). Shared by every per-framework *_test.go file in this package.
type stubResolver struct {
	byName map[string]string
	byLine map[int32]string
}

func newStubResolver() *stubResolver {
	return &stubResolver{byName: make(map[string]string), byLine: make(map[int32]string)}
}

func (s *stubResolver) withName(name, id string) *stubResolver {
	s.byName[name] = id
	return s
}

func (s *stubResolver) withLine(line int32, id string) *stubResolver {
	s.byLine[line] = id
	return s
}

func (s *stubResolver) ResolveByName(name string) (string, bool) {
	id, ok := s.byName[name]
	return id, ok
}

func (s *stubResolver) ResolveByLine(line int32) (string, bool) {
	id, ok := s.byLine[line]
	return id, ok
}

// parseFixture parses a real fixture file with lang's real CGo parser
// (never a hand-rolled stub tree) — every Walk function is exercised
// against a genuine tree-sitter AST, matching Pattern 4's own "AST-based,
// not regex" requirement.
func parseFixture(t *testing.T, lang, path string) (*tree_sitter.Node, []byte) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}

	var p parser.Parser
	switch lang {
	case "go":
		p, err = cgo.NewGoParser()
	case "java":
		p, err = cgo.NewJavaParser()
	case "csharp":
		p, err = cgo.NewCSharpParser()
	case "python":
		p, err = cgo.NewPythonParser()
	case "typescript":
		p, err = cgo.NewTypeScriptParser()
	default:
		t.Fatalf("parseFixture: unsupported lang %q", lang)
	}
	if err != nil {
		t.Fatalf("new %s parser: %v", lang, err)
	}
	t.Cleanup(func() { p.Close() })

	tree, err := p.Parse(src, nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	t.Cleanup(func() { tree.Close() })

	native, ok := tree.Inner().(*tree_sitter.Tree)
	if !ok || native == nil {
		t.Fatalf("Parse: unexpected tree type")
	}
	return native.RootNode(), src
}

func findRoute(routes []Route, method, path string) (Route, bool) {
	for _, r := range routes {
		if r.HTTPMethod == method && r.Path == path {
			return r, true
		}
	}
	return Route{}, false
}
