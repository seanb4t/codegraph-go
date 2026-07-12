package javaextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Java parser, closed automatically at the
// end of the test (mirrors goextract_test.go's newTestParser, one Parser
// per test).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewJavaParser()
	if err != nil {
		t.Fatalf("cgo.NewJavaParser: %v", err)
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

// TestExtract_NodeKinds is table-driven, mirroring goextract_test.go's
// TestExtract_NodeKinds: one case per tree-sitter node type this extractor
// must map onto the shared codegraph vocabulary (LANG-02, D-06).
func TestExtract_NodeKinds(t *testing.T) {
	tests := []struct {
		name              string
		src               string
		wantKind          string
		wantName          string
		wantQualifiedName string
		wantExported      bool
	}{
		{
			name:              "public class",
			src:               "package p;\n\npublic class Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "package-private class",
			src:               "package p;\n\nclass Helper {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helper",
			wantQualifiedName: "Helper",
			wantExported:      false,
		},
		{
			name:              "public interface",
			src:               "package p;\n\npublic interface Reader {\n\tint read();\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
			wantExported:      true,
		},
		{
			name:              "public method",
			src:               "package p;\n\npublic class Widget {\n\tpublic int size() { return 1; }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
			wantExported:      true,
		},
		{
			name:              "private method",
			src:               "package p;\n\npublic class Widget {\n\tprivate int helper() { return 1; }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "helper",
			wantQualifiedName: "Widget.helper",
			wantExported:      false,
		},
		{
			name:              "constructor maps to method",
			src:               "package p;\n\npublic class Widget {\n\tpublic Widget() {}\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "Widget",
			wantQualifiedName: "Widget.Widget",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "p", "Widget.java", []byte(tt.src))
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
			if found.Node.IsExported != tt.wantExported {
				t.Errorf("IsExported = %v, want %v", found.Node.IsExported, tt.wantExported)
			}
			if found.Node.Language != "java" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "java")
			}

			fileNode := findNode(result, goextract.KindFile, "Widget.java")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's method produces a
// same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "package p;\n\npublic class Widget {\n\tpublic int size() { return 1; }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
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

// TestExtract_NoFieldNodes proves field_declaration never becomes its own
// "field" node (mirrors goextract's ratified skip).
func TestExtract_NoFieldNodes(t *testing.T) {
	src := "package p;\n\npublic class Widget {\n\tprivate String name;\n\tprivate int age;\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Kind == "field" {
			t.Fatalf("unexpected field node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_Imports proves import_declaration produces a RefKindImports
// unresolved ref and populates Imports keyed by the imported simple name.
func TestExtract_Imports(t *testing.T) {
	src := "package p;\n\nimport com.example.Helper;\n\npublic class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.Imports["Helper"] != "com.example" {
		t.Errorf("Imports[Helper] = %q, want %q", result.Imports["Helper"], "com.example")
	}
	if !hasUnresolved(result, goextract.RefKindImports, "com.example.Helper", "") {
		t.Errorf("expected imports ref to com.example.Helper, got %+v", result.Unresolved)
	}
}

// TestExtract_ExtendsImplements proves a class's extends/implements clause
// each produce a RefKindEmbeds unresolved ref (RESEARCH Pattern 2 — not
// distinguished from one another at parse time).
func TestExtract_ExtendsImplements(t *testing.T) {
	src := "package p;\n\npublic class Widget extends Base implements Reader, Writer {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, want := range []string{"Base", "Reader", "Writer"} {
		if !hasUnresolved(result, goextract.RefKindEmbeds, want, "") {
			t.Errorf("expected embeds ref to %s, got %+v", want, result.Unresolved)
		}
	}
}

// TestExtract_InterfaceExtends proves an interface's own extends clause
// (extends_interfaces, a differently-shaped grammar node than a class's
// superclass/super_interfaces fields) also produces RefKindEmbeds refs.
func TestExtract_InterfaceExtends(t *testing.T) {
	src := "package p;\n\npublic interface ReadWriter extends Reader, Writer {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "ReadWriter.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, want := range []string{"Reader", "Writer"} {
		if !hasUnresolved(result, goextract.RefKindEmbeds, want, "") {
			t.Errorf("expected embeds ref to %s, got %+v", want, result.Unresolved)
		}
	}
}

// TestExtract_Calls proves method_invocation produces RefKindCalls
// unresolved refs, distinguishing an implicit same-class call, an imported
// static-class call, and a local-variable-receiver call (never
// mis-resolvable as same-package, mirroring goextract's WR-02 fix).
func TestExtract_Calls(t *testing.T) {
	src := `package p;

import com.example.Helper;

public class Widget {
	public void run() {
		helper();
		Helper.assist();
		Widget w = new Widget();
		w.helper();
	}

	private void helper() {}
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "Helper") {
		t.Errorf("expected calls ref to Helper.assist with PkgAlias %q, got %+v", "Helper", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.helper with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_SamePackageQualifiedCall proves an uppercase-leading
// identifier receiver that is NOT a real import resolves as a same-package
// candidate (empty PkgAlias, routes through resolveUnqualified) rather than
// a doomed selector lookup — the Java-specific counterpart to Go's WR-02
// fix, since Java allows `ClassName.method()` for a same-package class with
// no import statement at all.
func TestExtract_SamePackageQualifiedCall(t *testing.T) {
	src := `package p;

public class Caller {
	public void run() {
		Helper.assist();
	}
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Caller.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "") {
		t.Errorf("expected same-package calls ref to Helper.assist with empty PkgAlias, got %+v", result.Unresolved)
	}
}

// TestExtract_PackageDeclarationOverridesModuleKey proves a file's own
// `package a.b.c;` declaration overrides the discovery-time path-based
// moduleKey placeholder passed into Extract.
func TestExtract_PackageDeclarationOverridesModuleKey(t *testing.T) {
	src := "package com.example.widgets;\n\npublic class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.java", "sub/Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "com.example.widgets" {
		t.Errorf("ImportPath = %q, want %q (declared package overrides the path-based moduleKey)", result.ImportPath, "com.example.widgets")
	}
}

// TestExtract_NoPackageDeclarationKeepsModuleKey proves a file with no
// package declaration keeps the passed-in path-based moduleKey placeholder
// (D-03's path-identity fallback).
func TestExtract_NoPackageDeclarationKeepsModuleKey(t *testing.T) {
	src := "public class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.java", "sub/Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "sub/Widget.java" {
		t.Errorf("ImportPath = %q, want %q (no package declaration, keeps the path-based fallback)", result.ImportPath, "sub/Widget.java")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes, mirroring goextract_test.go's stub of the
// same name.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error — skip-not-fatal (RESEARCH Pitfall 4, threat T-05-DoS).
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "p", "Big.java", []byte("public class Big {}\n"))
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
