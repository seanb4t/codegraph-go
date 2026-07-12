package pyextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Python parser, closed automatically at
// the end of the test (mirrors javaextract_test.go's newTestParser).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewPythonParser()
	if err != nil {
		t.Fatalf("cgo.NewPythonParser: %v", err)
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
// must map onto the shared codegraph vocabulary (LANG-04).
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
			src:               "class Widget:\n    pass\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "private-convention class",
			src:               "class _Helper:\n    pass\n",
			wantKind:          goextract.KindStruct,
			wantName:          "_Helper",
			wantQualifiedName: "_Helper",
			wantExported:      false,
		},
		{
			name:              "public method",
			src:               "class Widget:\n    def size(self):\n        return 1\n",
			wantKind:          goextract.KindMethod,
			wantName:          "size",
			wantQualifiedName: "Widget.size",
			wantExported:      true,
		},
		{
			name:              "private-convention method",
			src:               "class Widget:\n    def _helper(self):\n        return 1\n",
			wantKind:          goextract.KindMethod,
			wantName:          "_helper",
			wantQualifiedName: "Widget._helper",
			wantExported:      false,
		},
		{
			name:              "module-level function",
			src:               "def run():\n    pass\n",
			wantKind:          goextract.KindFunction,
			wantName:          "run",
			wantQualifiedName: "run",
			wantExported:      true,
		},
		{
			name:              "decorated method still extracted",
			src:               "class Widget:\n    @staticmethod\n    def build():\n        pass\n",
			wantKind:          goextract.KindMethod,
			wantName:          "build",
			wantQualifiedName: "Widget.build",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "widget", "widget.py", []byte(tt.src))
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
			if found.Node.Language != "python" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "python")
			}

			fileNode := findNode(result, goextract.KindFile, "widget.py")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
		})
	}
}

// TestExtract_MethodContainsEdge proves a class's method produces a
// same-file class->method contains IntraEdge.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "class Widget:\n    def size(self):\n        return 1\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.py", []byte(src))
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

// TestExtract_NoModuleLevelAssignmentNodes proves a bare module-level (or
// class-body) assignment never becomes its own node (mirrors the "no field
// node" skip goextract/javaextract/csharpextract all already apply).
func TestExtract_NoModuleLevelAssignmentNodes(t *testing.T) {
	src := "CONST = 1\n\nclass Widget:\n    x = 2\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Node.Kind == "constant" || n.Node.Kind == "variable" || n.Node.Kind == "field" {
			t.Fatalf("unexpected %s node emitted: %+v", n.Node.Kind, n.Node)
		}
	}
}

// TestExtract_Imports proves import/from-import statements produce
// RefKindImports unresolved refs, with only an aliased plain import or a
// from-import populating result.Imports (types.go's documented gap: a
// plain `import a.b` binds only the top-level name "a", not the full
// dotted path, so no Imports entry is populated for it).
func TestExtract_Imports(t *testing.T) {
	src := "import pkg.util as u\nimport pkg.raw\nfrom pkg.helpers import Helper\nfrom pkg.helpers import Other as O\nfrom pkg import *\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.Imports["u"] != "pkg.util" {
		t.Errorf("Imports[u] = %q, want %q", result.Imports["u"], "pkg.util")
	}
	if _, ok := result.Imports["pkg"]; ok {
		t.Errorf("expected no Imports entry for a plain unaliased `import pkg.raw`, got Imports[pkg]=%q", result.Imports["pkg"])
	}
	if result.Imports["Helper"] != "pkg.helpers" {
		t.Errorf("Imports[Helper] = %q, want %q", result.Imports["Helper"], "pkg.helpers")
	}
	if result.Imports["O"] != "pkg.helpers" {
		t.Errorf("Imports[O] = %q, want %q", result.Imports["O"], "pkg.helpers")
	}
	if !hasUnresolved(result, goextract.RefKindImports, "pkg.util", "") {
		t.Errorf("expected imports ref to pkg.util, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "pkg.raw", "") {
		t.Errorf("expected imports ref to pkg.raw, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "pkg.helpers", "") {
		t.Errorf("expected imports ref to pkg.helpers, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindImports, "pkg", "") {
		t.Errorf("expected imports ref to pkg (wildcard from-import), got %+v", result.Unresolved)
	}
}

// TestExtract_RelativeImports proves a relative import (`from . import x`,
// `from ..pkg import y`) resolves against the current file's own enclosing
// dotted package, derived from its moduleKey.
func TestExtract_RelativeImports(t *testing.T) {
	src := "from . import sibling\nfrom ..other import Thing\n"
	p := newTestParser(t)
	// moduleKey "pkg.sub.mod" -> ownPackage "pkg.sub" (one dot = current
	// package); two dots strips one more level -> "pkg".
	result, err := Extract(p, "pkg.sub.mod", "pkg/sub/mod.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.Imports["sibling"] != "pkg.sub" {
		t.Errorf("Imports[sibling] = %q, want %q", result.Imports["sibling"], "pkg.sub")
	}
	if result.Imports["Thing"] != "pkg.other" {
		t.Errorf("Imports[Thing] = %q, want %q", result.Imports["Thing"], "pkg.other")
	}
}

// TestExtract_BaseClasses proves a class's base-class list (identifiers and
// a single module.Attr-shaped attribute chain) produces RefKindEmbeds
// unresolved refs, skipping a keyword argument (metaclass=...).
func TestExtract_BaseClasses(t *testing.T) {
	src := "import pkg.zoo as zoo\n\nclass Dog(Animal, zoo.Mammal, metaclass=Meta):\n    pass\n"
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindEmbeds, "Animal", "") {
		t.Errorf("expected embeds ref to Animal (same-module, empty PkgAlias), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindEmbeds, "Mammal", "zoo") {
		t.Errorf("expected embeds ref to zoo.Mammal with PkgAlias %q, got %+v", "zoo", result.Unresolved)
	}
	if hasUnresolved(result, goextract.RefKindEmbeds, "Meta", "") {
		t.Errorf("did not expect an embeds ref for the metaclass= keyword argument, got %+v", result.Unresolved)
	}
}

// TestExtract_Calls proves a call produces RefKindCalls unresolved refs,
// distinguishing an implicit same-class call (self.), an imported call, a
// same-module PascalCase-qualified call, and a local-variable-receiver call
// (never mis-resolvable as same-module, mirroring goextract's WR-02 fix).
func TestExtract_Calls(t *testing.T) {
	src := `import pkg.util as u

class Widget:
    def run(self):
        self.helper()
        u.assist()
        Helper.assist()
        w = Widget()
        w.helper()

    def helper(self):
        pass
`
	p := newTestParser(t)
	result, err := Extract(p, "widget", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected self.helper() to resolve with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "u") {
		t.Errorf("expected calls ref to u.assist with PkgAlias %q, got %+v", "u", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "") {
		t.Errorf("expected Helper.assist() (Helper not a real import, PascalCase convention) to resolve with empty PkgAlias, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "<local:w>") {
		t.Errorf("expected local-variable-receiver calls ref to w.helper with a synthetic non-matching alias, got %+v", result.Unresolved)
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves Python's moduleKey
// (already fully computed by languages_python.go's ModuleKey hook before
// Extract is ever called, since Python has no in-source declared identity
// to parse) is never overridden by Extract — the opposite of javaextract/
// csharpextract's parse-time-override pattern.
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	src := "class Widget:\n    pass\n"
	p := newTestParser(t)
	result, err := Extract(p, "pkg.sub.widget", "pkg/sub/widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "pkg.sub.widget" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "pkg.sub.widget")
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
// (or any Parse error — the INDENT/DEDENT external scanner's front-line
// mitigation, threat T-05-DoS) is recorded on FileResult.Err with a nil
// returned error — skip-not-fatal.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "big", "big.py", []byte("class Big:\n    pass\n"))
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
