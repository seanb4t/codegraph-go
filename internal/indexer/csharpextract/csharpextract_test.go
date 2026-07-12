package csharpextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo C# parser, closed automatically at the
// end of the test (mirrors javaextract_test.go's newTestParser, one Parser
// per test).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewCSharpParser()
	if err != nil {
		t.Fatalf("cgo.NewCSharpParser: %v", err)
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

// TestExtract_NodeKinds is table-driven, mirroring javaextract_test.go's
// TestExtract_NodeKinds: one case per tree-sitter node type this extractor
// must map onto the shared codegraph vocabulary (LANG-03, D-06).
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
			src:               "namespace P;\n\npublic class Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "internal-default class",
			src:               "namespace P;\n\nclass Helper {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helper",
			wantQualifiedName: "Helper",
			wantExported:      false,
		},
		{
			name:              "public struct",
			src:               "namespace P;\n\npublic struct Point {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Point",
			wantQualifiedName: "Point",
			wantExported:      true,
		},
		{
			name:              "public record",
			src:               "namespace P;\n\npublic record Pair(int X, int Y);\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Pair",
			wantQualifiedName: "Pair",
			wantExported:      true,
		},
		{
			name:              "public interface",
			src:               "namespace P;\n\npublic interface Reader {\n\tint Read();\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
			wantExported:      true,
		},
		{
			name:              "public method",
			src:               "namespace P;\n\npublic class Widget {\n\tpublic int Size() { return 1; }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "Size",
			wantQualifiedName: "Widget.Size",
			wantExported:      true,
		},
		{
			name:              "private-default method",
			src:               "namespace P;\n\npublic class Widget {\n\tint Helper() { return 1; }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "Helper",
			wantQualifiedName: "Widget.Helper",
			wantExported:      false,
		},
		{
			name:              "constructor maps to method",
			src:               "namespace P;\n\npublic class Widget {\n\tpublic Widget() {}\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "Widget",
			wantQualifiedName: "Widget.Widget",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "p", "Widget.cs", []byte(tt.src))
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
			if found.Node.Language != "csharp" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "csharp")
			}

			fileNode := findNode(result, goextract.KindFile, "Widget.cs")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's method produces a
// same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "namespace P;\n\npublic class Widget {\n\tpublic int Size() { return 1; }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	widget := findNode(result, goextract.KindStruct, "Widget")
	if widget == nil {
		t.Fatalf("no struct node Widget found in %+v", result.Nodes)
	}
	size := findNode(result, goextract.KindMethod, "Size")
	if size == nil {
		t.Fatalf("no method node Size found in %+v", result.Nodes)
	}
	if !hasIntraEdge(result, widget.Node.Id, size.Node.Id, "contains") {
		t.Errorf("expected Widget->Size contains edge, got %+v", result.IntraEdges)
	}
}

// TestExtract_NoPropertyOrFieldNodes proves property_declaration and
// field_declaration never become their own node (mirrors goextract's/
// javaextract's ratified skip).
func TestExtract_NoPropertyOrFieldNodes(t *testing.T) {
	src := "namespace P;\n\npublic class Widget {\n\tprivate string _name;\n\tpublic string Name { get; set; }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Name == "_name" || n.Node.Name == "Name" {
			t.Fatalf("unexpected property/field node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_Usings proves a plain using_directive produces a
// RefKindImports unresolved ref WITHOUT populating Imports (a namespace
// names no single simple type), while the alias form DOES populate
// Imports, exactly like a Go/Java import alias.
func TestExtract_Usings(t *testing.T) {
	src := "using System.Collections.Generic;\nusing Json = System.Text.Json;\n\nnamespace P;\n\npublic class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindImports, "System.Collections.Generic", "") {
		t.Errorf("expected imports ref to System.Collections.Generic, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "System.Text.Json", "") {
		t.Errorf("expected imports ref to System.Text.Json, got %+v", result.Unresolved)
	}
	if _, ok := result.Imports["System.Collections.Generic"]; ok {
		t.Errorf("plain using should not populate Imports, got %+v", result.Imports)
	}
	if result.Imports["Json"] != "System.Text.Json" {
		t.Errorf("Imports[Json] = %q, want %q (alias form)", result.Imports["Json"], "System.Text.Json")
	}
}

// TestExtract_BaseList proves a class's base_list (both a base class and
// interfaces, undistinguished — RESEARCH Pattern 2) and an interface's own
// base_list each produce RefKindEmbeds unresolved refs.
func TestExtract_BaseList(t *testing.T) {
	src := "namespace P;\n\npublic class Widget : BaseWidget, IReader, IWriter {}\n\npublic interface IReadWriter : IReader, IWriter {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, want := range []string{"BaseWidget", "IReader", "IWriter"} {
		if !hasUnresolved(result, goextract.RefKindEmbeds, want, "") {
			t.Errorf("expected embeds ref to %s, got %+v", want, result.Unresolved)
		}
	}
}

// TestExtract_Calls proves invocation_expression produces RefKindCalls
// unresolved refs, distinguishing an implicit same-class call, a
// same-namespace PascalCase-qualified call, a local-variable-receiver call
// (never mis-resolvable as same-namespace, mirroring goextract's WR-02
// fix), a fully-qualified cross-namespace call (resolved via its own
// AST-spelled-out namespace prefix, no `using` needed), and a
// local-variable CHAINED field-access call (`w.Field.Compute()` — never
// mistaken for a namespace-shaped chain merely because it contains dots).
func TestExtract_Calls(t *testing.T) {
	src := `namespace P;

public class Widget {
	public void Run() {
		Helper();
		SameNamespaceHelper.Assist();
		Widget w = new Widget();
		w.Helper();
		Other.Namespace.Helper.Assist();
		w.Field.Compute();
	}

	private void Helper() {}
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "Helper", "") {
		t.Errorf("expected unqualified calls ref to Helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "Assist", "") {
		t.Errorf("expected same-namespace calls ref to SameNamespaceHelper.Assist with empty PkgAlias (same-namespace attempt), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "Helper", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.Helper with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "Assist", "Other.Namespace") {
		t.Errorf("expected fully-qualified cross-namespace calls ref to Other.Namespace.Helper.Assist with PkgAlias %q, got %+v", "Other.Namespace", result.Unresolved)
	}
	if result.Imports["Other.Namespace"] != "Other.Namespace" {
		t.Errorf("expected fully-qualified call to self-map Imports[Other.Namespace], got %+v", result.Imports)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "Compute", "<local:w>") {
		t.Errorf("expected local-variable chained field-access calls ref w.Field.Compute with a synthetic non-matching alias (never a namespace), got %+v", result.Unresolved)
	}
}

// TestExtract_NamespaceDeclarationOverridesModuleKey proves a file's own
// block-form `namespace Foo.Bar { ... }` declaration overrides the
// discovery-time path-based moduleKey placeholder passed into Extract.
func TestExtract_NamespaceDeclarationOverridesModuleKey(t *testing.T) {
	src := "namespace Company.Widgets {\n\tpublic class Widget {}\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.cs", "sub/Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "Company.Widgets" {
		t.Errorf("ImportPath = %q, want %q (declared namespace overrides the path-based moduleKey)", result.ImportPath, "Company.Widgets")
	}
}

// TestExtract_FileScopedNamespaceOverridesModuleKey proves the file-scoped
// namespace form (`namespace Foo.Bar;`, no braces) is parsed identically to
// the block form.
func TestExtract_FileScopedNamespaceOverridesModuleKey(t *testing.T) {
	src := "namespace Company.Widgets;\n\npublic class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.cs", "sub/Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "Company.Widgets" {
		t.Errorf("ImportPath = %q, want %q (file-scoped namespace overrides the path-based moduleKey)", result.ImportPath, "Company.Widgets")
	}
}

// TestExtract_NoNamespaceKeepsModuleKey proves a file with no namespace
// declaration keeps the passed-in path-based moduleKey placeholder (D-03's
// path-identity fallback).
func TestExtract_NoNamespaceKeepsModuleKey(t *testing.T) {
	src := "public class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "sub/Widget.cs", "sub/Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "sub/Widget.cs" {
		t.Errorf("ImportPath = %q, want %q (no namespace declaration, keeps the path-based fallback)", result.ImportPath, "sub/Widget.cs")
	}
}

// TestExtract_PartialClass_SharedNodeIdentity proves the Pitfall 5 scheme
// (b) node-identity decision documented in types.go: two `partial class
// Foo` fragments declared in DIFFERENT files, sharing the SAME namespace,
// produce the exact SAME node id (keyed by qualifiedName+namespace, not
// filePath), with a deterministic sentinel FilePath ("") — and each
// fragment's own method is still `contains`-edged to that ONE shared node
// (no data loss across fragments).
func TestExtract_PartialClass_SharedNodeIdentity(t *testing.T) {
	p := newTestParser(t)

	src1 := "namespace P;\n\npublic partial class Widget {\n\tpublic void FromFragmentOne() {}\n}\n"
	result1, err := Extract(p, "p", "Widget.cs", []byte(src1))
	if err != nil {
		t.Fatalf("Extract (fragment 1) returned error: %v", err)
	}

	src2 := "namespace P;\n\npublic partial class Widget {\n\tpublic void FromFragmentTwo() {}\n}\n"
	result2, err := Extract(p, "p", "Widget.Designer.cs", []byte(src2))
	if err != nil {
		t.Fatalf("Extract (fragment 2) returned error: %v", err)
	}

	widget1 := findNode(result1, goextract.KindStruct, "Widget")
	widget2 := findNode(result2, goextract.KindStruct, "Widget")
	if widget1 == nil || widget2 == nil {
		t.Fatalf("expected a Widget struct node in both fragments, got fragment1=%+v fragment2=%+v", result1.Nodes, result2.Nodes)
	}
	if widget1.Node.Id != widget2.Node.Id {
		t.Errorf("partial class fragments computed DIFFERENT node ids: %q vs %q, want identical (Pitfall 5 scheme b)", widget1.Node.Id, widget2.Node.Id)
	}
	if widget1.Node.FilePath != "" {
		t.Errorf("partial class shared node FilePath = %q, want \"\" (deterministic sentinel — see types.go doc)", widget1.Node.FilePath)
	}
	if widget2.Node.FilePath != "" {
		t.Errorf("partial class shared node FilePath = %q, want \"\" (deterministic sentinel — see types.go doc)", widget2.Node.FilePath)
	}

	fromOne := findNode(result1, goextract.KindMethod, "FromFragmentOne")
	fromTwo := findNode(result2, goextract.KindMethod, "FromFragmentTwo")
	if fromOne == nil || fromTwo == nil {
		t.Fatalf("expected each fragment's own method to be extracted, got fragment1=%+v fragment2=%+v", result1.Nodes, result2.Nodes)
	}
	if !hasIntraEdge(result1, widget1.Node.Id, fromOne.Node.Id, "contains") {
		t.Errorf("expected shared Widget node -> FromFragmentOne contains edge in fragment 1, got %+v", result1.IntraEdges)
	}
	if !hasIntraEdge(result2, widget2.Node.Id, fromTwo.Node.Id, "contains") {
		t.Errorf("expected shared Widget node -> FromFragmentTwo contains edge in fragment 2, got %+v", result2.IntraEdges)
	}
}

// TestExtract_NonPartialClassKeepsFilePathIdentity proves an ORDINARY
// (non-partial) class keeps the normal relPath-keyed node identity —
// Pitfall 5's scheme only applies to `partial` types.
func TestExtract_NonPartialClassKeepsFilePathIdentity(t *testing.T) {
	src := "namespace P;\n\npublic class Widget {}\n"
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	widget := findNode(result, goextract.KindStruct, "Widget")
	if widget == nil {
		t.Fatalf("no struct node Widget found in %+v", result.Nodes)
	}
	if widget.Node.FilePath != "Widget.cs" {
		t.Errorf("FilePath = %q, want %q (non-partial types keep normal relPath identity)", widget.Node.FilePath, "Widget.cs")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes, mirroring javaextract_test.go's stub of the
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
	result, err := Extract(stubOversizedParser{}, "p", "Big.cs", []byte("public class Big {}\n"))
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
