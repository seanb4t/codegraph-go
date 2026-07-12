package cextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newCTestParser returns a real CGo C parser, closed automatically at the
// end of the test (mirrors rustextract_test.go/phpextract_test.go's own
// newTestParser).
func newCTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewCParser()
	if err != nil {
		t.Fatalf("cgo.NewCParser: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

// newCppTestParser returns a real CGo C++ parser, closed automatically at
// the end of the test.
func newCppTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewCppParser()
	if err != nil {
		t.Fatalf("cgo.NewCppParser: %v", err)
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
// TestExtract_NodeKinds: one case per tree-sitter node type this SHARED
// extractor maps onto the shared codegraph vocabulary, across BOTH a C and
// a C++ source (LANG-06, D-06).
func TestExtract_NodeKinds(t *testing.T) {
	tests := []struct {
		name              string
		cpp               bool
		relPath           string
		src               string
		wantKind          string
		wantName          string
		wantQualifiedName string
		wantLanguage      string
	}{
		{
			name:              "C struct",
			relPath:           "widget.c",
			src:               "struct Widget {\n    int size;\n};\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantLanguage:      "c",
		},
		{
			name:              "C typedef",
			relPath:           "widget.c",
			src:               "typedef struct Widget WidgetAlias;\n",
			wantKind:          goextract.KindTypeAlias,
			wantName:          "WidgetAlias",
			wantQualifiedName: "WidgetAlias",
			wantLanguage:      "c",
		},
		{
			name:              "C top-level function",
			relPath:           "widget.c",
			src:               "int add(int a, int b) {\n    return a + b;\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "add",
			wantQualifiedName: "add",
			wantLanguage:      "c",
		},
		{
			name:              "C function prototype (declaration, no body)",
			relPath:           "widget.h",
			src:               "int helper(int x);\n",
			wantKind:          goextract.KindFunction,
			wantName:          "helper",
			wantQualifiedName: "helper",
			wantLanguage:      "c",
		},
		{
			name:              "C++ class maps to struct",
			cpp:               true,
			relPath:           "shape.cpp",
			src:               "class Circle {\npublic:\n    double area() { return 1.0; }\n};\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Circle",
			wantQualifiedName: "Circle",
			wantLanguage:      "cpp",
		},
		{
			name:              "C++ inline method",
			cpp:               true,
			relPath:           "shape.cpp",
			src:               "class Circle {\npublic:\n    double area() { return 1.0; }\n};\n",
			wantKind:          goextract.KindMethod,
			wantName:          "area",
			wantQualifiedName: "Circle.area",
			wantLanguage:      "cpp",
		},
		{
			name:              "C++ out-of-line method",
			cpp:               true,
			relPath:           "shape.cpp",
			src:               "class Circle {\npublic:\n    double area();\n};\n\ndouble Circle::area() {\n    return 1.0;\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "area",
			wantQualifiedName: "Circle.area",
			wantLanguage:      "cpp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p parser.Parser
			if tt.cpp {
				p = newCppTestParser(t)
			} else {
				p = newCTestParser(t)
			}
			result, err := Extract(p, "mymodule", tt.relPath, []byte(tt.src))
			if err != nil {
				t.Fatalf("Extract returned error: %v", err)
			}
			if result.Err != nil {
				t.Fatalf("FileResult.Err = %v, want nil", result.Err)
			}
			if result.Language != tt.wantLanguage {
				t.Errorf("FileResult.Language = %q, want %q", result.Language, tt.wantLanguage)
			}

			found := findNode(result, tt.wantKind, tt.wantName)
			if found == nil {
				t.Fatalf("no %s node named %q found in %+v", tt.wantKind, tt.wantName, result.Nodes)
			}
			if found.Node.QualifiedName != tt.wantQualifiedName {
				t.Errorf("QualifiedName = %q, want %q", found.Node.QualifiedName, tt.wantQualifiedName)
			}
			if found.Node.Language != tt.wantLanguage {
				t.Errorf("Node.Language = %q, want %q", found.Node.Language, tt.wantLanguage)
			}

			fileNode := findNode(result, goextract.KindFile, tt.relPath)
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_OutOfLineMethodContainsEdge proves a C++ out-of-line method
// definition (`RetType Type::method() {}`) produces a same-file
// type->method contains IntraEdge when the qualifying type is declared in
// the same file, mirroring rustextract's identical cross-file impl-block
// pattern.
func TestExtract_OutOfLineMethodContainsEdge(t *testing.T) {
	src := "class Circle {\npublic:\n    double area();\n};\n\ndouble Circle::area() {\n    return 1.0;\n}\n"
	p := newCppTestParser(t)
	result, err := Extract(p, "mymodule", "shape.cpp", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	circle := findNode(result, goextract.KindStruct, "Circle")
	if circle == nil {
		t.Fatalf("no struct node Circle found in %+v", result.Nodes)
	}
	area := findNode(result, goextract.KindMethod, "area")
	if area == nil {
		t.Fatalf("no method node area found in %+v", result.Nodes)
	}
	if !hasIntraEdge(result, circle.Node.Id, area.Node.Id, "contains") {
		t.Errorf("expected Circle->area contains edge, got %+v", result.IntraEdges)
	}
}

// TestExtract_OutOfLineMethodCrossFileContains proves a C++ out-of-line
// method definition whose qualifying type is NOT declared in this same file
// (the common .h/.cpp split) produces a RefKindContains unresolved ref for
// Pass 2 to resolve, instead of silently dropping the method.
func TestExtract_OutOfLineMethodCrossFileContains(t *testing.T) {
	src := "double Circle::area() {\n    return 1.0;\n}\n"
	p := newCppTestParser(t)
	result, err := Extract(p, "mymodule", "shape.cpp", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	area := findNode(result, goextract.KindMethod, "area")
	if area == nil {
		t.Fatalf("no method node area found in %+v", result.Nodes)
	}
	if !hasUnresolved(result, goextract.RefKindContains, "Circle", "") {
		t.Errorf("expected a RefKindContains ref to Circle, got %+v", result.Unresolved)
	}
}

// TestExtract_NoFieldNodes proves a struct's member fields, AND a C++
// class's bodyless/pure-virtual method declarations, never become their own
// node (mirrors goextract's/rustextract's ratified "no field node" skip,
// extended to field_declaration-shaped method prototypes -- types.go).
func TestExtract_NoFieldNodes(t *testing.T) {
	src := "struct Widget {\n    int size;\n    char name;\n};\n"
	p := newCTestParser(t)
	result, err := Extract(p, "mymodule", "widget.c", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Name == "size" || n.Node.Name == "name" {
			t.Fatalf("unexpected field node emitted: %+v", n.Node)
		}
	}

	cppSrc := "class Shape {\npublic:\n    virtual double area() = 0;\n};\n"
	cp := newCppTestParser(t)
	cresult, err := Extract(cp, "mymodule", "shape.cpp", []byte(cppSrc))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if n := findNode(cresult, goextract.KindMethod, "area"); n != nil {
		t.Fatalf("unexpected method node for pure-virtual prototype: %+v", n.Node)
	}
}

// TestExtract_Includes proves preproc_include produces a RefKindImports
// unresolved ref for both quoted and system-lib forms, and never populates
// FileResult.Imports (types.go's documented gap, mirroring Rust's `use`
// decision).
func TestExtract_Includes(t *testing.T) {
	src := "#include \"foo.h\"\n#include <stdio.h>\n\nint main() {\n    return 0;\n}\n"
	p := newCTestParser(t)
	result, err := Extract(p, "mymodule", "main.c", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, want := range []string{"foo.h", "stdio.h"} {
		if !hasUnresolved(result, goextract.RefKindImports, want, "") {
			t.Errorf("expected imports ref to %s, got %+v", want, result.Unresolved)
		}
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected no Imports entries from #include (documented gap), got %+v", result.Imports)
	}
}

// TestExtract_Supertypes proves a C++ base_class_clause produces a
// RefKindEmbeds unresolved ref (Pattern 2 -- extends/implements
// undistinguished at parse time).
func TestExtract_Supertypes(t *testing.T) {
	src := "class Shape {\npublic:\n    virtual double area() = 0;\n};\n\nclass Circle : public Shape {\npublic:\n    double area() { return 1.0; }\n};\n"
	p := newCppTestParser(t)
	result, err := Extract(p, "mymodule", "shape.cpp", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindEmbeds, "Shape", "") {
		t.Errorf("expected embeds ref to Shape, got %+v", result.Unresolved)
	}
}

// TestExtract_Calls proves call_expression produces RefKindCalls unresolved
// refs across C's bare-function-call shape and C++'s this->/qualified-call
// shapes.
func TestExtract_Calls(t *testing.T) {
	src := `int helper(int x) { return x; }

int add(int a, int b) {
    return helper(a) + b;
}
`
	p := newCTestParser(t)
	result, err := Extract(p, "mymodule", "widget.c", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}

	cppSrc := `class Widget {
public:
    void run() {
        this->helper();
        Widget::staticMethod();
        w.size();
    }
    void helper() {}
    static void staticMethod() {}
};
`
	cp := newCppTestParser(t)
	cresult, err := Extract(cp, "mymodule", "widget.cpp", []byte(cppSrc))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(cresult, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected this->helper() calls ref with empty PkgAlias, got %+v", cresult.Unresolved)
	}
	if !hasUnresolved(cresult, goextract.RefKindCalls, "staticMethod", "") {
		t.Errorf("expected Widget::staticMethod() calls ref with empty PkgAlias, got %+v", cresult.Unresolved)
	}
	if !hasUnresolved(cresult, goextract.RefKindCalls, "size", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.size with a synthetic non-matching alias, got %+v", cresult.Unresolved)
	}
}

// TestExtract_LanguageDeterminedByExtension proves cextract.Extract
// determines "c" vs "cpp" from relPath's own extension, since its shared
// cross-language signature carries no explicit language field -- the
// central adaptation this SHARED package makes over every other
// single-language mainstream extractor.
func TestExtract_LanguageDeterminedByExtension(t *testing.T) {
	cases := []struct {
		relPath string
		want    string
	}{
		{"widget.c", "c"},
		{"widget.h", "c"},
		{"widget.cpp", "cpp"},
		{"widget.cc", "cpp"},
		{"widget.cxx", "cpp"},
		{"widget.hpp", "cpp"},
		{"widget.hh", "cpp"},
	}
	for _, c := range cases {
		if got := languageForExt(c.relPath); got != c.want {
			t.Errorf("languageForExt(%q) = %q, want %q", c.relPath, got, c.want)
		}
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves moduleKey flows
// straight through to FileResult.ImportPath -- C/C++ have no in-source
// module declaration to override it with (types.go).
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	p := newCTestParser(t)
	result, err := Extract(p, "src/widget.c", "src/widget.c", []byte("struct Widget {\n    int size;\n};\n"))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "src/widget.c" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "src/widget.c")
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
// tree-sitter-cpp's external C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "mymodule", "big.cpp", []byte("class Big {};\n"))
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
