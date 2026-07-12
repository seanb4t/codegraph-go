package phpextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo PHP parser (php/src grammar accessor,
// not php_only), closed automatically at the end of the test.
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewPHPParser()
	if err != nil {
		t.Fatalf("cgo.NewPHPParser: %v", err)
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
// shape: one case per tree-sitter node type this extractor maps onto the
// shared codegraph vocabulary (LANG-06, D-06).
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
			src:               "<?php\nclass Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
		},
		{
			name:              "interface maps to interface",
			src:               "<?php\ninterface Reader {\n    public function read(): int;\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
		},
		{
			name:              "trait maps to struct (documented)",
			src:               "<?php\ntrait Helpers {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helpers",
			wantQualifiedName: "Helpers",
		},
		{
			name:              "top-level function",
			src:               "<?php\nfunction helper() {\n    return 1;\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "helper",
			wantQualifiedName: "helper",
		},
		{
			name:              "class method",
			src:               "<?php\nclass Widget {\n    public function size(): int {\n        return 1;\n    }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "widget", "Widget.php", []byte(tt.src))
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
			if found.Node.Language != "php" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "php")
			}

			fileNode := findNode(result, goextract.KindFile, "Widget.php")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's method produces a
// same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "<?php\nclass Widget {\n    public function size(): int {\n        return 1;\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "Widget.php", []byte(src))
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

// TestExtract_NoPropertyNodes proves class properties never become their
// own node (mirrors goextract's/pyextract's ratified "no field node" skip).
func TestExtract_NoPropertyNodes(t *testing.T) {
	src := "<?php\nclass Widget {\n    private int $size = 1;\n    public string $name;\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Name == "size" || n.Node.Name == "name" {
			t.Fatalf("unexpected property node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_NamespaceUse proves namespace_use_declaration (simple,
// grouped/braced, and aliased forms) produces RefKindImports unresolved
// refs and populates Imports keyed by the alias-or-simple-name.
func TestExtract_NamespaceUse(t *testing.T) {
	src := "<?php\nnamespace App\\Models;\n\nuse App\\Contracts\\Shape;\nuse App\\Utils\\{Helper, Logger};\nuse App\\Utils\\Other as Alias;\n\nclass Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "app/Models/Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, want := range []string{`App\Contracts\Shape`, `App\Utils\Helper`, `App\Utils\Logger`, `App\Utils\Other`} {
		if !hasUnresolved(result, goextract.RefKindImports, want, "") {
			t.Errorf("expected imports ref to %s, got %+v", want, result.Unresolved)
		}
	}
	if result.Imports["Shape"] != `App\Contracts\Shape` {
		t.Errorf("Imports[Shape] = %q, want %q", result.Imports["Shape"], `App\Contracts\Shape`)
	}
	if result.Imports["Helper"] != `App\Utils\Helper` {
		t.Errorf("Imports[Helper] = %q, want %q (grouped/braced use)", result.Imports["Helper"], `App\Utils\Helper`)
	}
	if result.Imports["Alias"] != `App\Utils\Other` {
		t.Errorf("Imports[Alias] = %q, want %q (aliased use)", result.Imports["Alias"], `App\Utils\Other`)
	}
}

// TestExtract_SupertypeClauses proves a class's base_clause (extends) and
// class_interface_clause (implements), and an interface's own base_clause
// (extends), each produce RefKindEmbeds unresolved refs (Pattern 2 —
// extends/implements undistinguished at parse time).
func TestExtract_SupertypeClauses(t *testing.T) {
	src := "<?php\ninterface Shape {}\n\ninterface Reader {}\n\ninterface ReadableShape extends Shape, Reader {}\n\nclass Base {}\n\nclass Widget extends Base implements Shape, Reader {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, want := range []string{"Base", "Shape", "Reader"} {
		if !hasUnresolved(result, goextract.RefKindEmbeds, want, "") {
			t.Errorf("expected embeds ref to %s, got %+v", want, result.Unresolved)
		}
	}
}

// TestExtract_Calls proves function_call_expression, member_call_expression,
// and scoped_call_expression each produce RefKindCalls unresolved refs,
// distinguishing an unqualified function call, an implicit $this-> call, a
// same-module Type::method() static call, and a local-variable-receiver
// call (never mis-resolved as same-module, mirroring goextract's WR-02
// fix).
func TestExtract_Calls(t *testing.T) {
	src := `<?php
class Widget {
    public function run() {
        helper();
        Widget::build();
        $this->size();
        $w = new Widget();
        $w->size();
    }

    public function size(): int {
        return 1;
    }

    public static function build(): Widget {
        return new Widget();
    }
}

function helper() {}
`
	p := newTestParser(t)
	result, err := Extract(p, "widget", "Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "build", "") {
		t.Errorf("expected same-module static calls ref to Widget::build with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "size", "") {
		t.Errorf("expected implicit $this-> calls ref to size with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "size", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to $w->size with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_NamespaceOverridesModuleKey proves a file's own `namespace
// Foo\Bar;` declaration overrides the discovery-time path-based moduleKey
// placeholder passed into Extract, mirroring csharpextract's identical
// parse-time-override pattern.
func TestExtract_NamespaceOverridesModuleKey(t *testing.T) {
	src := "<?php\nnamespace App\\Models;\n\nclass Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.php", "sub/Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != `App\Models` {
		t.Errorf("ImportPath = %q, want %q (declared namespace overrides the path-based moduleKey)", result.ImportPath, `App\Models`)
	}
}

// TestExtract_NoNamespaceKeepsModuleKey proves a file with no namespace
// declaration keeps the passed-in path-based moduleKey placeholder (D-03's
// path-identity fallback).
func TestExtract_NoNamespaceKeepsModuleKey(t *testing.T) {
	src := "<?php\nclass Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.php", "sub/Widget.php", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "sub/Widget.php" {
		t.Errorf("ImportPath = %q, want %q (no namespace declaration, keeps the path-based fallback)", result.ImportPath, "sub/Widget.php")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error — skip-not-fatal (T-05-DoS), the front-line mitigation for PHP's
// external tag-switching (`<?php ?>`) C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "big", "Big.php", []byte("<?php\nclass Big {}\n"))
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
