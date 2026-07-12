package kotlinextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Kotlin parser, closed automatically at
// the end of the test (mirrors rustextract_test.go/phpextract_test.go's own
// newTestParser).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewKotlinParser()
	if err != nil {
		t.Fatalf("cgo.NewKotlinParser: %v", err)
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
			src:               "class Circle(val radius: Double) {\n    fun area(): Double {\n        return 1.0\n    }\n}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Circle",
			wantQualifiedName: "Circle",
		},
		{
			name:              "interface maps to interface",
			src:               "interface Shape {\n    fun area(): Double\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Shape",
			wantQualifiedName: "Shape",
		},
		{
			name:              "top-level function",
			src:               "fun topLevel(): Int {\n    return 1\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "topLevel",
			wantQualifiedName: "topLevel",
		},
		{
			name:              "instance method",
			src:               "class Circle {\n    fun area(): Double {\n        return 1.0\n    }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "area",
			wantQualifiedName: "Circle.area",
		},
		{
			name:              "interface method (bodyless)",
			src:               "interface Shape {\n    fun area(): Double\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "area",
			wantQualifiedName: "Shape.area",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "com.example.widgets", "circle.kt", []byte(tt.src))
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
			if found.Node.Language != "kotlin" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "kotlin")
			}

			fileNode := findNode(result, goextract.KindFile, "circle.kt")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's instance method produces
// a same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "class Circle {\n    fun area(): Double {\n        return 1.0\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "com.example", "circle.kt", []byte(src))
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

// TestExtract_PackageOverridesModuleKey proves a file's own declared
// `package foo.bar` statement overrides the discovery-time moduleKey
// placeholder -- mirrors phpextract's identical parse-time-override
// pattern (types.go).
func TestExtract_PackageOverridesModuleKey(t *testing.T) {
	src := "package com.example.widgets\n\nclass Circle {\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "placeholder", "circle.kt", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "com.example.widgets" {
		t.Errorf("ImportPath = %q, want %q (package override)", result.ImportPath, "com.example.widgets")
	}
}

// TestExtract_Import proves import produces a RefKindImports unresolved ref
// AND populates FileResult.Imports (unlike Rust's `use`, mirroring PHP's
// `use` decision -- types.go).
func TestExtract_Import(t *testing.T) {
	src := "import kotlin.collections.List\n\nfun topLevel(): Int {\n    return 1\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "com.example", "widget.kt", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "kotlin.collections.List", "") {
		t.Errorf("expected imports ref to kotlin.collections.List, got %+v", result.Unresolved)
	}
	if got, want := result.Imports["List"], "kotlin.collections.List"; got != want {
		t.Errorf("Imports[List] = %q, want %q", got, want)
	}
}

// TestExtract_Delegation proves a plain interface-conformance
// delegation_specifier AND a constructor-invoked superclass delegation both
// produce a RefKindEmbeds unresolved ref (Pattern 2 -- extends/implements
// undistinguished at parse time).
func TestExtract_Delegation(t *testing.T) {
	src := "interface Shape {\n}\n\nopen class Base {\n}\n\nclass Circle : Base(), Shape {\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "com.example", "circle.kt", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, want := range []string{"Base", "Shape"} {
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
	src := `class Circle {
    fun run() {
        helper()
        val c = Circle()
        c.area()
    }

    fun area(): Double {
        return 1.0
    }
}

fun helper() {}
`
	p := newTestParser(t)
	result, err := Extract(p, "com.example", "circle.kt", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "area", "<local:c>") {
		t.Errorf("expected local-variable-receiver calls ref to c.area with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_ObjectNotExtracted proves a Kotlin `object` singleton
// declaration is not extracted -- a documented gap (types.go).
func TestExtract_ObjectNotExtracted(t *testing.T) {
	src := "object Singleton {\n    fun helper() {}\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "com.example", "singleton.kt", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if n := findNode(result, goextract.KindStruct, "Singleton"); n != nil {
		t.Fatalf("unexpected struct node for object declaration: %+v", n.Node)
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
// tree-sitter-kotlin's external C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "com.example", "big.kt", []byte("class Big {\n}\n"))
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
