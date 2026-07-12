package goextract

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo Go parser, closed automatically at the
// end of the test (mirrors the "one Parser per worker" lifecycle, scoped
// here to one Parser per test).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewGoParser()
	if err != nil {
		t.Fatalf("cgo.NewGoParser: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func findNode(result FileResult, kind, name string) *ExtractedNode {
	for i := range result.Nodes {
		if result.Nodes[i].Node.Kind == kind && result.Nodes[i].Node.Name == name {
			return &result.Nodes[i]
		}
	}
	return nil
}

func hasIntraEdge(result FileResult, source, target, kind string) bool {
	for _, e := range result.IntraEdges {
		if e.Edge.Source == source && e.Edge.Target == target && e.Edge.Kind == kind {
			return true
		}
	}
	return false
}

func hasUnresolved(result FileResult, kind, name, pkgAlias string) bool {
	for _, u := range result.Unresolved {
		if u.Kind == kind && u.Name == name && u.PkgAlias == pkgAlias {
			return true
		}
	}
	return false
}

// TestExtract_NodeKinds is table-driven: one case per tree-sitter node type
// this extractor must map onto the codegraph vocabulary (LANG-01, D-06).
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
			name:              "top-level exported function",
			src:               "package p\n\nfunc Alpha() int { return 1 }\n",
			wantKind:          KindFunction,
			wantName:          "Alpha",
			wantQualifiedName: "Alpha",
			wantExported:      true,
		},
		{
			name:              "top-level unexported function",
			src:               "package p\n\nfunc helper() int { return 1 }\n",
			wantKind:          KindFunction,
			wantName:          "helper",
			wantQualifiedName: "helper",
			wantExported:      false,
		},
		{
			name:              "struct declaration",
			src:               "package p\n\ntype Widget struct {\n\tName string\n}\n",
			wantKind:          KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "interface declaration",
			src:               "package p\n\ntype Reader interface {\n\tRead() int\n}\n",
			wantKind:          KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
			wantExported:      true,
		},
		{
			name:              "type definition maps to type_alias",
			src:               "package p\n\ntype Celsius float64\n",
			wantKind:          KindTypeAlias,
			wantName:          "Celsius",
			wantQualifiedName: "Celsius",
			wantExported:      true,
		},
		{
			name:              "true type alias maps to type_alias",
			src:               "package p\n\ntype MyAlias = int\n",
			wantKind:          KindTypeAlias,
			wantName:          "MyAlias",
			wantQualifiedName: "MyAlias",
			wantExported:      true,
		},
		{
			name:              "constant declaration",
			src:               "package p\n\nconst Version = \"1\"\n",
			wantKind:          KindConstant,
			wantName:          "Version",
			wantQualifiedName: "Version",
			wantExported:      true,
		},
		{
			name:              "variable declaration",
			src:               "package p\n\nvar Registry = map[string]int{}\n",
			wantKind:          KindVariable,
			wantName:          "Registry",
			wantQualifiedName: "Registry",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t)
			result, err := Extract(p, "example.com/p", "p.go", []byte(tt.src))
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
			if found.Node.FilePath != "p.go" {
				t.Errorf("FilePath = %q, want %q", found.Node.FilePath, "p.go")
			}
			if found.Node.Language != "go" {
				t.Errorf("Language = %q, want %q", found.Node.Language, "go")
			}

			fileNode := findNode(result, KindFile, "p.go")
			if fileNode == nil {
				t.Fatalf("no file node found in %+v", result.Nodes)
			}
			if !hasIntraEdge(result, fileNode.Node.Id, found.Node.Id, "contains") {
				t.Errorf("expected file->%s contains edge, got %+v", tt.wantKind, result.IntraEdges)
			}
		})
	}
}

// TestExtract_MethodReceivers proves value- and pointer-receiver methods
// both resolve to the same receiver type, get the "Recv.Method" qualified
// name, and produce a same-file type->method contains edge (LANG-01).
func TestExtract_MethodReceivers(t *testing.T) {
	src := `package p

type Widget struct {
	Name string
}

func (w Widget) Describe() string {
	return w.Name
}

func (w *Widget) Rename(n string) {
	w.Name = n
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.Err != nil {
		t.Fatalf("FileResult.Err = %v, want nil", result.Err)
	}

	widget := findNode(result, KindStruct, "Widget")
	if widget == nil {
		t.Fatalf("no struct node Widget found in %+v", result.Nodes)
	}

	describe := findNode(result, KindMethod, "Describe")
	if describe == nil {
		t.Fatalf("no method node Describe found in %+v", result.Nodes)
	}
	if describe.Node.QualifiedName != "Widget.Describe" {
		t.Errorf("Describe QualifiedName = %q, want %q", describe.Node.QualifiedName, "Widget.Describe")
	}
	if !hasIntraEdge(result, widget.Node.Id, describe.Node.Id, "contains") {
		t.Errorf("expected Widget->Describe contains edge, got %+v", result.IntraEdges)
	}

	rename := findNode(result, KindMethod, "Rename")
	if rename == nil {
		t.Fatalf("no method node Rename found in %+v", result.Nodes)
	}
	if rename.Node.QualifiedName != "Widget.Rename" {
		t.Errorf("Rename (pointer receiver) QualifiedName = %q, want %q", rename.Node.QualifiedName, "Widget.Rename")
	}
	if !hasIntraEdge(result, widget.Node.Id, rename.Node.Id, "contains") {
		t.Errorf("expected Widget->Rename contains edge, got %+v", result.IntraEdges)
	}
}

// TestExtract_StructEmbedding proves an embedded field (a type child with
// no field-name child) is recorded as an unresolved "embeds" ref, and a
// named field of the same underlying type is NOT.
func TestExtract_StructEmbedding(t *testing.T) {
	src := `package p

type Base struct {
	ID int
}

type Derived struct {
	Base
	Extra string
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	derived := findNode(result, KindStruct, "Derived")
	if derived == nil {
		t.Fatalf("no struct node Derived found in %+v", result.Nodes)
	}
	if !hasUnresolved(result, RefKindEmbeds, "Base", "") {
		t.Errorf("expected embeds ref to Base from Derived, got %+v", result.Unresolved)
	}
	if hasUnresolved(result, RefKindEmbeds, "Extra", "") {
		t.Errorf("named field Extra must NOT be recorded as an embeds ref, got %+v", result.Unresolved)
	}
	_ = derived
}

// TestExtract_InterfaceEmbedding proves interface embedding (a type_list
// entry, distinct from the interface's own method set) is recorded as an
// unresolved "embeds" ref.
func TestExtract_InterfaceEmbedding(t *testing.T) {
	src := `package p

type Reader interface {
	Read() int
}

type ReadWriter interface {
	Reader
	Write(int)
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, RefKindEmbeds, "Reader", "") {
		t.Errorf("expected embeds ref to Reader from ReadWriter, got %+v", result.Unresolved)
	}
}

// TestExtract_Calls proves a cross-package qualified call and an
// intra-package unqualified call both yield an unresolved "calls" ref
// carrying name, kind, and call-site line/col (RES-01).
func TestExtract_Calls(t *testing.T) {
	src := `package pkgb

import "example.com/gofixture/pkga"

func Run() {
	_ = pkga.Alpha()
	_ = helper()
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/pkgb", "pkgb.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, RefKindCalls, "Alpha", "pkga") {
		t.Errorf("expected calls ref to pkga.Alpha, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, RefKindCalls, "helper", "") {
		t.Errorf("expected calls ref to unqualified helper, got %+v", result.Unresolved)
	}

	for _, u := range result.Unresolved {
		if u.Kind == RefKindCalls && u.Name == "Alpha" {
			if u.Line == 0 {
				t.Errorf("calls ref to Alpha has zero Line, want a real call-site line")
			}
		}
	}
}

// TestExtract_CallAsArgument locks in that a call passed as an argument to
// another call (`outer(inner())`) is extracted as TWO separate calls refs —
// one for the outer call, one for the inner call (D-05's call-as-argument
// item). Investigation for this plan confirmed walkDescendants already
// visits argument-position call_expression nodes correctly (it continues
// descending into every node's children regardless of kind); this test
// locks that behavior in as a regression guard.
func TestExtract_CallAsArgument(t *testing.T) {
	src := `package p

func Run() {
	outer(inner())
}
func outer(x int) int { return x }
func inner() int { return 1 }
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, RefKindCalls, "outer", "") {
		t.Errorf("expected calls ref to outer, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, RefKindCalls, "inner", "") {
		t.Errorf("expected calls ref to inner (call-as-argument), got %+v", result.Unresolved)
	}
}

// TestExtract_SelectorNonIdentifierOperandNeverAliasQualified proves WR-02:
// a selector call whose operand is a non-identifier expression
// (`foo().Bar()`, operand is a call_expression) never carries a PkgAlias
// that would route resolveNameRef to a same-package unqualified match —
// its PkgAlias must be non-empty (so resolveNameRef routes it through
// resolveSelector, not resolveUnqualified) and must never equal a real
// import alias present in the file.
func TestExtract_SelectorNonIdentifierOperandNeverAliasQualified(t *testing.T) {
	src := `package p

type Widget struct{}

func (w Widget) Bar() int { return 1 }

func foo() Widget { return Widget{} }

func Run() {
	foo().Bar()
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	var found *UnresolvedRef
	for i := range result.Unresolved {
		u := &result.Unresolved[i]
		if u.Kind == RefKindCalls && u.Name == "Bar" {
			found = u
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a calls ref for Bar, got %+v", result.Unresolved)
	}
	if found.PkgAlias == "" {
		t.Fatalf("WR-02: Bar's calls ref has empty PkgAlias, which resolveNameRef treats as unqualified and would mis-resolve to a same-package Bar symbol")
	}
}

// TestExtract_LocalVariableReceiverCallUnchanged proves an identifier
// operand that is a local variable (not a real import alias) keeps its
// existing behavior: PkgAlias is set to the identifier text, later falling
// through to "unresolved" via resolveSelector's own alias-membership
// boundary — this plan does not touch that path.
func TestExtract_LocalVariableReceiverCallUnchanged(t *testing.T) {
	src := `package p

type Widget struct{}

func (w Widget) Describe() string { return "" }

func F() {
	var w Widget
	_ = w.Describe()
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, RefKindCalls, "Describe", "w") {
		t.Errorf("expected calls ref to Describe with PkgAlias %q (local-var receiver, unchanged), got %+v", "w", result.Unresolved)
	}
}

// TestExtract_Imports proves the file's import_spec list is parsed into
// Imports keyed by local alias (default, explicit alias).
func TestExtract_Imports(t *testing.T) {
	src := `package p

import (
	"fmt"
	x "example.com/gofixture/pkga"
)

func F() {
	fmt.Println(x.Version)
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.Imports["fmt"] != "fmt" {
		t.Errorf("Imports[fmt] = %q, want %q", result.Imports["fmt"], "fmt")
	}
	if result.Imports["x"] != "example.com/gofixture/pkga" {
		t.Errorf("Imports[x] = %q, want %q", result.Imports["x"], "example.com/gofixture/pkga")
	}
}

// TestExtract_NoFieldNodes proves struct fields never become their own
// "field" node (ratified skip, D-06 / Open Question 3).
func TestExtract_NoFieldNodes(t *testing.T) {
	src := `package p

type Widget struct {
	Name string
	Age  int
}
`
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, n := range result.Nodes {
		if n.Node.Kind == "field" {
			t.Fatalf("unexpected field node emitted: %+v", n.Node)
		}
	}
}

// TestExtract_ContentHashIsSHA256 proves ContentHash is computed via
// crypto/sha256 (never crypto/md5), per D-02a.
func TestExtract_ContentHashIsSHA256(t *testing.T) {
	src := []byte("package p\n\nfunc Alpha() int { return 1 }\n")
	p := newTestParser(t)
	result, err := Extract(p, "example.com/p", "p.go", src)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	want := sha256.Sum256(src)
	wantHex := hex.EncodeToString(want[:])
	if result.ContentHash != wantHex {
		t.Errorf("ContentHash = %q, want sha256 %q", result.ContentHash, wantHex)
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file
// that trips parser.MaxSourceBytes, without needing to allocate an actual
// >4 MiB source string in this test.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error — skip-not-fatal (RESEARCH Pitfall 4, threat T-02-03).
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "example.com/p", "big.go", []byte("package p\n"))
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

// TestExtract_SharedFixture runs the extractor against the committed,
// larger multi-construct fixture (internal/indexer/testdata/gofixture),
// not just small inline sources, proving the mapper holds up on real,
// multi-declaration files.
func TestExtract_SharedFixture(t *testing.T) {
	src, err := os.ReadFile("../testdata/gofixture/pkga/pkga.go")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	p := newTestParser(t)
	result, err := Extract(p, "example.com/gofixture/pkga", "pkga/pkga.go", src)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.Err != nil {
		t.Fatalf("FileResult.Err = %v, want nil", result.Err)
	}

	for _, want := range []struct{ kind, name string }{
		{KindFunction, "Alpha"},
		{KindFunction, "helper"},
		{KindConstant, "Version"},
		{KindVariable, "Registry"},
		{KindStruct, "Widget"},
		{KindMethod, "Describe"},
		{KindMethod, "Rename"},
	} {
		if findNode(result, want.kind, want.name) == nil {
			t.Errorf("fixture: expected %s node %q, not found in %+v", want.kind, want.name, result.Nodes)
		}
	}

	if !hasUnresolved(result, RefKindCalls, "helper", "") {
		t.Errorf("fixture: expected intra-package calls ref to helper, got %+v", result.Unresolved)
	}

	embedSrc, err := os.ReadFile("../testdata/gofixture/pkga/embed.go")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	embedResult, err := Extract(p, "example.com/gofixture/pkga", "pkga/embed.go", embedSrc)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(embedResult, RefKindEmbeds, "Base", "") {
		t.Errorf("fixture: expected embeds ref to Base from Derived, got %+v", embedResult.Unresolved)
	}
	if !hasUnresolved(embedResult, RefKindEmbeds, "Reader", "") {
		t.Errorf("fixture: expected embeds ref to Reader from ReadWriter, got %+v", embedResult.Unresolved)
	}
}
