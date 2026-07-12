package swiftextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Swift parser, closed automatically at
// the end of the test (mirrors rustextract_test.go/phpextract_test.go's own
// newTestParser).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewSwiftParser()
	if err != nil {
		t.Fatalf("cgo.NewSwiftParser: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func findNode(result goextract.FileResult, kind, name string) *goextract.ExtractedNode {
	for i := range result.Nodes {
		if result.Nodes[i].Node.Kind == kind && result.Nodes[i].Node.Name == name {
			return &result.Nodes[i]
		}
	}
	return nil
}

func hasIntraEdge(result goextract.FileResult, source, target, kind string) bool {
	for _, e := range result.IntraEdges {
		if e.Edge.Source == source && e.Edge.Target == target && e.Edge.Kind == kind {
			return true
		}
	}
	return false
}

func hasUnresolved(result goextract.FileResult, kind, name, pkgAlias string) bool {
	for _, u := range result.Unresolved {
		if u.Kind == kind && u.Name == name && u.PkgAlias == pkgAlias {
			return true
		}
	}
	return false
}

// TestExtract_NodeKinds is table-driven, mirroring rustextract_test.go's
// TestExtract_NodeKinds: one case per tree-sitter node shape this extractor
// maps onto the shared codegraph vocabulary (LANG-06, D-06).
func TestExtract_NodeKinds(t *testing.T) {
	tests := []struct {
		name              string
		src               string
		wantKind          string
		wantName          string
		wantQualifiedName string
	}{
		{
			name:              "class maps to struct",
			src:               "class Widget {\n    func size() -> Int {\n        return 1\n    }\n}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
		},
		{
			name:              "struct maps to struct",
			src:               "struct Point {\n    func dist() -> Int {\n        return 1\n    }\n}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Point",
			wantQualifiedName: "Point",
		},
		{
			name:              "protocol maps to interface",
			src:               "protocol Shape {\n    func area() -> Double\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Shape",
			wantQualifiedName: "Shape",
		},
		{
			name:              "top-level function",
			src:               "func topLevel() -> Int {\n    return 1\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "topLevel",
			wantQualifiedName: "topLevel",
		},
		{
			name:              "instance method",
			src:               "class Widget {\n    func size() -> Int {\n        return 1\n    }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "MyTarget", "widget.swift", []byte(tt.src))
			if err != nil {
				t.Fatalf("Extract returned error: %v", err)
			}
			if result.Err != nil {
				t.Fatalf("FileResult.Err = %v, want nil", result.Err)
			}

			found := findNode(result, tt.wantKind, tt.wantName)
			if found == nil {
				t.Fatalf("no %s node named %q found in %+v", tt.wantKind, tt.wantName, result.Nodes)
			}
			if found.Node.QualifiedName != tt.wantQualifiedName {
				t.Errorf("QualifiedName = %q, want %q", found.Node.QualifiedName, tt.wantQualifiedName)
			}
			if found.Node.Language != "swift" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "swift")
			}

			fileNode := findNode(result, goextract.KindFile, "widget.swift")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's instance method produces
// a same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "class Widget {\n    func size() -> Int {\n        return 1\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "widget.swift", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	widget := findNode(result, goextract.KindStruct, "Widget")
	if widget == nil {
		t.Fatalf("no struct node Widget found in %+v", result.Nodes)
	}
	size := findNode(result, goextract.KindMethod, "size")
	if size == nil {
		t.Fatalf("no method node size found in %+v", result.Nodes)
	}
	if !hasIntraEdge(result, widget.Node.Id, size.Node.Id, "contains") {
		t.Errorf("expected Widget->size contains edge, got %+v", result.IntraEdges)
	}
}

// TestExtract_ExtensionNotExtracted proves an extension declaration is
// recognized (never misextracted as a fresh type) but its own members are
// never walked -- the plan's own explicitly-named gap (types.go).
func TestExtract_ExtensionNotExtracted(t *testing.T) {
	src := "class Widget {\n}\n\nextension Widget {\n    func extra() -> Int {\n        return 1\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "widget.swift", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if n := findNode(result, goextract.KindMethod, "extra"); n != nil {
		t.Fatalf("unexpected method node for extension member: %+v", n.Node)
	}
	// The class itself is still extracted exactly once.
	found := false
	for _, n := range result.Nodes {
		if n.Node.Kind == goextract.KindStruct && n.Node.Name == "Widget" {
			if found {
				t.Fatalf("Widget extracted more than once: %+v", result.Nodes)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a single Widget struct node, got %+v", result.Nodes)
	}
}

// TestExtract_Import proves import_declaration produces a RefKindImports
// unresolved ref and never populates FileResult.Imports (types.go's
// documented gap).
func TestExtract_Import(t *testing.T) {
	src := "import Foundation\n\nfunc topLevel() -> Int {\n    return 1\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "widget.swift", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "Foundation", "") {
		t.Errorf("expected imports ref to Foundation, got %+v", result.Unresolved)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected no Imports entries from `import` (documented gap), got %+v", result.Imports)
	}
}

// TestExtract_Inheritance proves a multi-conformance class (`class Foo: A,
// B`) produces one RefKindEmbeds unresolved ref per inheritance_specifier
// (Pattern 2 -- extends/implements undistinguished at parse time).
func TestExtract_Inheritance(t *testing.T) {
	src := "protocol A {\n}\nprotocol B {\n}\nclass Foo: A, B {\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "foo.swift", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, want := range []string{"A", "B"} {
		if !hasUnresolved(result, goextract.RefKindEmbeds, want, "") {
			t.Errorf("expected embeds ref to %s, got %+v", want, result.Unresolved)
		}
	}
}

// TestExtract_Calls proves call_expression produces RefKindCalls unresolved
// refs, distinguishing a bare function call and a local-variable-receiver
// method call (never mis-resolved as same-module, mirroring goextract's
// WR-02 fix).
func TestExtract_Calls(t *testing.T) {
	src := `class Widget {
    func run() {
        helper()
        let w = Widget()
        w.size()
    }

    func size() -> Int {
        return 1
    }
}

func helper() {}
`
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "widget.swift", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "size", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.size with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves moduleKey flows
// straight through to FileResult.ImportPath -- Swift has no in-source
// module declaration to override it with (types.go).
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	p := newTestParser(t)
	result, err := Extract(p, "MyTarget", "widget.swift", []byte("class Widget {\n}\n"))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "MyTarget" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "MyTarget")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes, mirroring rustextract_test.go's stub of the
// same name.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error -- skip-not-fatal (T-05-DoS), the front-line mitigation for
// tree-sitter-swift's external C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "MyTarget", "big.swift", []byte("class Big {\n}\n"))
	if err != nil {
		t.Fatalf("Extract returned a non-nil error for a per-file skip: %v", err)
	}
	if result.Err == nil {
		t.Fatalf("FileResult.Err = nil, want parser.ErrSourceTooLarge")
	}
	if !errors.Is(result.Err, parser.ErrSourceTooLarge) {
		t.Errorf("FileResult.Err = %v, want wrapping parser.ErrSourceTooLarge", result.Err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes for a skipped file, got %+v", result.Nodes)
	}
}
