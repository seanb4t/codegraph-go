package rustextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Rust parser, closed automatically at the
// end of the test (mirrors csharpextract_test.go/pyextract's own
// newTestParser).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewRustParser()
	if err != nil {
		t.Fatalf("cgo.NewRustParser: %v", err)
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

// TestExtract_NodeKinds is table-driven, mirroring csharpextract_test.go's
// TestExtract_NodeKinds: one case per tree-sitter node type this extractor
// maps onto the shared codegraph vocabulary (LANG-06, D-06).
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
			name:              "public struct",
			src:               "pub struct Widget {\n    pub size: i32,\n}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "private-default struct",
			src:               "struct Helper;\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helper",
			wantQualifiedName: "Helper",
			wantExported:      false,
		},
		{
			name:              "enum maps to struct (documented)",
			src:               "pub enum Shape {\n    Circle,\n    Square,\n}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Shape",
			wantQualifiedName: "Shape",
			wantExported:      true,
		},
		{
			name:              "public trait maps to interface",
			src:               "pub trait Reader {\n    fn read(&self) -> i32;\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
			wantExported:      true,
		},
		{
			name:              "top-level function",
			src:               "pub fn helper() -> i32 {\n    1\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "helper",
			wantQualifiedName: "helper",
			wantExported:      true,
		},
		{
			name:              "impl method",
			src:               "pub struct Widget;\n\nimpl Widget {\n    pub fn size(&self) -> i32 {\n        1\n    }\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "mycrate", "widget.rs", []byte(tt.src))
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
			if found.Node.Language != "rust" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "rust")
			}

			fileNode := findNode(result, goextract.KindFile, "widget.rs")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_ImplMethodContainsEdge proves a struct's impl-block method
// produces a same-file type->method contains IntraEdge.
func TestExtract_ImplMethodContainsEdge(t *testing.T) {
	src := "pub struct Widget;\n\nimpl Widget {\n    pub fn size(&self) -> i32 {\n        1\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "mycrate", "widget.rs", []byte(src))
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

// TestExtract_NoFieldNodes proves struct fields never become their own
// node (mirrors goextract's/pyextract's ratified "no field node" skip).
func TestExtract_NoFieldNodes(t *testing.T) {
	src := "pub struct Widget {\n    pub size: i32,\n    name: String,\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "mycrate", "widget.rs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Name == "size" || n.Node.Name == "name" {
			t.Fatalf("unexpected field node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_Uses proves a use_declaration (plain path, aliased, and
// grouped/braced form) produces a RefKindImports unresolved ref per leaf
// path — Rust's `use` resolution needs the enclosing crate name at Extract
// time to compute a matching moduleKey (not available via this signature,
// see types.go), so this extractor deliberately never populates Imports
// from a `use` statement (a documented, accepted gap).
func TestExtract_Uses(t *testing.T) {
	src := "use std::fmt::Display;\nuse std::collections::{HashMap, HashSet};\nuse serde::Serialize as Ser;\n\npub struct Widget;\n"
	p := newTestParser(t)
	result, err := Extract(p, "mycrate", "widget.rs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, want := range []string{"std::fmt::Display", "std::collections::HashMap", "std::collections::HashSet", "serde::Serialize"} {
		if !hasUnresolved(result, goextract.RefKindImports, want, "") {
			t.Errorf("expected imports ref to %s, got %+v", want, result.Unresolved)
		}
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected no Imports entries from `use` (documented gap), got %+v", result.Imports)
	}
}

// TestExtract_ImplTraitEmbeds proves `impl Trait for Type` produces a
// RefKindEmbeds unresolved ref (Pattern 2 — extends/implements
// undistinguished at parse time; resolve.go promotes it to "implements" if
// the target resolves to an interface node).
func TestExtract_ImplTraitEmbeds(t *testing.T) {
	src := "pub trait Shape {\n    fn area(&self) -> f64;\n}\n\npub struct Circle;\n\nimpl Shape for Circle {\n    fn area(&self) -> f64 {\n        0.0\n    }\n}\n"
	p := newTestParser(t)
	result, err := Extract(p, "mycrate", "shape.rs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindEmbeds, "Shape", "") {
		t.Errorf("expected embeds ref to Shape, got %+v", result.Unresolved)
	}
}

// TestExtract_Calls proves call_expression produces RefKindCalls unresolved
// refs, distinguishing a bare function call, a same-module PascalCase
// associated-function call (Type::new(), no import needed — naming
// heuristic mirroring pyextract/javaextract), a method call on a local
// variable (never mis-resolved as same-module, mirroring goextract's WR-02
// fix), and a same-module method call.
func TestExtract_Calls(t *testing.T) {
	src := `pub struct Widget;

impl Widget {
    pub fn new() -> Widget {
        Widget
    }

    pub fn run(&self) {
        helper();
        let w = Widget::new();
        w.size();
    }
}

fn helper() {}
`
	p := newTestParser(t)
	result, err := Extract(p, "mycrate", "widget.rs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "new", "") {
		t.Errorf("expected same-module associated-function calls ref to Widget::new with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "size", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.size with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves moduleKey flows
// straight through to FileResult.ImportPath — Rust's crate+module identity
// is computed entirely at discovery time (languages_rust.go's ModuleKey),
// unlike Java/C#'s in-source-declared identity.
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	p := newTestParser(t)
	result, err := Extract(p, "mycrate::foo", "src/foo.rs", []byte("pub struct Widget;\n"))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "mycrate::foo" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "mycrate::foo")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes, mirroring csharpextract_test.go's stub of
// the same name.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error — skip-not-fatal (T-05-DoS), the front-line mitigation for Rust's
// external raw-string C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "mycrate", "big.rs", []byte("pub struct Big;\n"))
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
