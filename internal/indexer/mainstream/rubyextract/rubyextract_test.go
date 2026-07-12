package rubyextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Ruby parser, closed automatically at the
// end of the test.
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewRubyParser()
	if err != nil {
		t.Fatalf("cgo.NewRubyParser: %v", err)
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
			src:               "class Widget\nend\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
		},
		{
			name:              "module maps to struct (documented)",
			src:               "module Helpers\nend\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helpers",
			wantQualifiedName: "Helpers",
		},
		{
			name:              "top-level method maps to function",
			src:               "def helper\n  1\nend\n",
			wantKind:          goextract.KindFunction,
			wantName:          "helper",
			wantQualifiedName: "helper",
		},
		{
			name:              "instance method maps to method",
			src:               "class Widget\n  def size\n    1\n  end\nend\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
		},
		{
			name:              "singleton method maps to method",
			src:               "class Widget\n  def self.create\n    Widget.new\n  end\nend\n",
			wantKind:          goextract.KindMethod,
			wantName:          "create",
			wantQualifiedName: "Widget.create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "widget", "widget.rb", []byte(tt.src))
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
			if found.Node.Language != "ruby" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "ruby")
			}

			fileNode := findNode(result, goextract.KindFile, "widget.rb")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's method produces a
// same-file type->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "class Widget\n  def size\n    1\n  end\nend\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.rb", []byte(src))
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

// TestExtract_NoInstanceVariableNodes proves instance-variable assignments
// never become their own node (mirrors goextract's/pyextract's ratified
// "no field node" skip).
func TestExtract_NoInstanceVariableNodes(t *testing.T) {
	src := "class Widget\n  def initialize\n    @size = 1\n  end\nend\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.rb", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Name == "@size" {
			t.Fatalf("unexpected instance-variable node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_Requires proves require/require_relative produce
// RefKindImports unresolved refs, and require_relative additionally
// resolves to a path-joined target moduleKey (best-effort, per
// languages_ruby.go's directory-relative ModuleKey).
func TestExtract_Requires(t *testing.T) {
	src := "require 'json'\nrequire_relative 'models/base'\n\nclass Widget\nend\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "app/widget.rb", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "json", "") {
		t.Errorf("expected imports ref to json, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "models/base", "") {
		t.Errorf("expected imports ref to models/base, got %+v", result.Unresolved)
	}
	if result.Imports["base"] != "app/models/base" {
		t.Errorf("Imports[base] = %q, want %q (require_relative resolved against this file's own directory)", result.Imports["base"], "app/models/base")
	}
}

// TestExtract_Superclass proves a class's superclass produces a
// RefKindEmbeds unresolved ref (Pattern 2 — extends/implements
// undistinguished at parse time).
func TestExtract_Superclass(t *testing.T) {
	src := "class Widget < Base\nend\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.rb", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindEmbeds, "Base", "") {
		t.Errorf("expected embeds ref to Base, got %+v", result.Unresolved)
	}
}

// TestExtract_Calls proves call produces RefKindCalls unresolved refs,
// distinguishing an implicit self call, a same-module constant-qualified
// call (no import needed, Ruby constants are PascalCase by convention), and
// a local-variable-receiver call (never mis-resolved as same-module,
// mirroring goextract's WR-02 fix). require/require_relative calls are
// never double-emitted as calls edges (see TestExtract_Requires). A bare
// no-parens no-args method invocation (`helper` alone, no receiver) is a
// documented, accepted gap: Ruby's own grammar cannot distinguish it from a
// local-variable reference without scope tracking this extractor does not
// implement, so this test uses the unambiguous `helper()` form instead.
func TestExtract_Calls(t *testing.T) {
	src := `class Widget
  def run
    helper()
    Widget.build
    w = Widget.new
    w.size
  end

  def helper
  end
end
`
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.rb", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected unqualified calls ref to helper, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "build", "") {
		t.Errorf("expected same-module constant-qualified calls ref to Widget.build with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "size", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.size with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindCalls && u.Name == "require" {
			t.Errorf("did not expect a calls ref for %q (require is import-only)", u.Name)
		}
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves moduleKey flows
// straight through to FileResult.ImportPath — Ruby's identity is computed
// entirely at discovery time (languages_ruby.go's ModuleKey, unconditional
// on descriptor presence, mirroring tsextract's own pattern).
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	p := newTestParser(t)
	result, err := Extract(p, "app/widget", "app/widget.rb", []byte("class Widget\nend\n"))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "app/widget" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "app/widget")
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
// error — skip-not-fatal (T-05-DoS), the front-line mitigation for Ruby's
// external heredoc C scanner.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "big", "big.rb", []byte("class Big\nend\n"))
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
