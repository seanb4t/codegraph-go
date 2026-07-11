package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// newTestParser returns a real CGo Go parser, closed automatically at the
// end of the test — this package's own copy of goextract_test.go's helper
// of the same name (unexported, package-scoped, no cross-package export
// warranted for a two-line test utility).
func newTestParser(t *testing.T) parser.Parser {
	t.Helper()
	p, err := cgo.NewGoParser()
	if err != nil {
		t.Fatalf("cgo.NewGoParser: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

// fixtureResults returns Pass 1's results and module path for the shared
// multi-package testdata/gofixture tree, the same fixture 02-03's own
// tests exercise.
func fixtureResults(t *testing.T) ([]goextract.FileResult, string) {
	t.Helper()
	files, modulePath, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	results, err := Extract(files, 0)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return results, modulePath
}

// findEdge reports whether edges contains one matching (source, kind,
// target) exactly.
func findEdge(edges []*schema.Edge, source, kind, target string) bool {
	for _, e := range edges {
		if e.Source == source && e.Kind == kind && e.Target == target {
			return true
		}
	}
	return false
}

// TestResolve_CrossPackageCall proves pkgb.Run's call to pkga.Alpha()
// resolves via alias -> import path -> symbol-index lookup (RES-01).
func TestResolve_CrossPackageCall(t *testing.T) {
	results, modulePath := fixtureResults(t)
	nodes, packageNodes, edges, files, unresolved := resolveRefs(results, modulePath)
	_ = nodes
	_ = packageNodes
	_ = files
	_ = unresolved

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "pkgb/pkgb.go")
	alphaID := nodeid.NodeID(goextract.KindFunction, "Alpha", "pkga/pkga.go")

	if !findEdge(edges, runID, "calls", alphaID) {
		t.Fatalf("expected calls edge %s -> %s (pkgb.Run -> pkga.Alpha), got %+v", runID, alphaID, edges)
	}
}

// TestResolve_IntraPackageCall proves an unqualified call resolves against
// the calling file's own package symbols.
func TestResolve_IntraPackageCall(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	alphaID := nodeid.NodeID(goextract.KindFunction, "Alpha", "pkga/pkga.go")
	helperID := nodeid.NodeID(goextract.KindFunction, "helper", "pkga/pkga.go")

	if !findEdge(edges, alphaID, "calls", helperID) {
		t.Fatalf("expected calls edge %s -> %s (pkga.Alpha -> pkga.helper), got %+v", alphaID, helperID, edges)
	}
}

// TestResolve_StructEmbeds proves `type Derived struct { Base }` yields an
// embeds edge Derived -> Base when both are in-repo.
func TestResolve_StructEmbeds(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "pkga/embed.go")
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "pkga/embed.go")

	if !findEdge(edges, derivedID, "embeds", baseID) {
		t.Fatalf("expected embeds edge %s -> %s (Derived -> Base), got %+v", derivedID, baseID, edges)
	}
}

// TestResolve_InterfaceEmbeds proves `type ReadWriter interface { Reader }`
// yields an embeds edge ReadWriter -> Reader.
func TestResolve_InterfaceEmbeds(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	rwID := nodeid.NodeID(goextract.KindInterface, "ReadWriter", "pkga/embed.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "pkga/embed.go")

	if !findEdge(edges, rwID, "embeds", readerID) {
		t.Fatalf("expected embeds edge %s -> %s (ReadWriter -> Reader), got %+v", rwID, readerID, edges)
	}
}

// TestResolve_IntraModuleImport proves an intra-module import produces a
// synthetic package pseudo-node and a file -> package "imports" edge
// (RQ-1 recommendation (a)).
func TestResolve_IntraModuleImport(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, packageNodes, edges, _, _ := resolveRefs(results, modulePath)

	importPath := "example.com/gofixture/pkga"
	pkgID := nodeid.NodeID("package", importPath, importPath)

	var found *schema.Node
	for _, n := range packageNodes {
		if n.Id == pkgID {
			found = n
			break
		}
	}
	if found == nil {
		t.Fatalf("expected synthetic package node %s for %s, got %+v", pkgID, importPath, packageNodes)
	}
	if found.Name != "pkga" {
		t.Errorf("package node Name = %q, want %q", found.Name, "pkga")
	}
	if found.QualifiedName != importPath {
		t.Errorf("package node QualifiedName = %q, want %q", found.QualifiedName, importPath)
	}

	pkgbFileID := nodeid.NodeID(goextract.KindFile, "pkgb/pkgb.go", "pkgb/pkgb.go")
	if !findEdge(edges, pkgbFileID, "imports", pkgID) {
		t.Fatalf("expected imports edge %s -> %s (pkgb.go -> pkga package node), got %+v", pkgbFileID, pkgID, edges)
	}
}

// TestResolve_ExternalImportNoEdge proves a stdlib/external import produces
// NO package node and NO imports edge.
func TestResolve_ExternalImportNoEdge(t *testing.T) {
	src := `package p

import "fmt"

func F() {
	fmt.Println("hi")
}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, packageNodes, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	if len(packageNodes) != 0 {
		t.Errorf("expected no package nodes for a stdlib-only import, got %+v", packageNodes)
	}
	fileID := nodeid.NodeID(goextract.KindFile, "p.go", "p.go")
	for _, e := range edges {
		if e.Kind == "imports" && e.Source == fileID {
			t.Errorf("expected no imports edge for stdlib \"fmt\", got %+v", e)
		}
	}
}

// TestResolve_UnresolvedMethodCall proves a method call whose receiver
// type cannot be determined without interface/inference is left
// unresolved (D-06a) — never emitted as an edge, but counted.
func TestResolve_UnresolvedMethodCall(t *testing.T) {
	src := `package p

type Widget struct{}

func (w Widget) Describe() string { return "" }

func F() {
	var w Widget
	_ = w.Describe()
}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, unresolved := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	fID := nodeid.NodeID(goextract.KindFunction, "F", "p.go")
	describeID := nodeid.NodeID(goextract.KindMethod, "Widget.Describe", "p.go")

	if findEdge(edges, fID, "calls", describeID) {
		t.Fatalf("did not expect a resolved calls edge for a local-variable receiver call, got %+v", edges)
	}
	if unresolved == 0 {
		t.Errorf("expected unresolved count > 0 for the unresolvable method call, got 0")
	}
}

// TestResolve_CrossFileMethodContainment proves a method whose receiver
// type is declared in a DIFFERENT file resolves into a type -> method
// "contains" edge once Pass 2 has the global symbol index (extends the
// plan's illustrative calls/imports/embeds vocabulary with the 4th kind
// goextract already emits, per 02-03's decision).
func TestResolve_CrossFileMethodContainment(t *testing.T) {
	typeSrc := `package p

type Widget struct{}
`
	methodSrc := `package p

func (w Widget) Describe() string { return "" }
`
	p := newTestParser(t)
	typeResult, err := goextract.Extract(p, "example.com/p", "type.go", []byte(typeSrc))
	if err != nil {
		t.Fatalf("Extract(type.go): %v", err)
	}
	methodResult, err := goextract.Extract(p, "example.com/p", "method.go", []byte(methodSrc))
	if err != nil {
		t.Fatalf("Extract(method.go): %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{typeResult, methodResult}, "example.com/p")

	widgetID := nodeid.NodeID(goextract.KindStruct, "Widget", "type.go")
	describeID := nodeid.NodeID(goextract.KindMethod, "Widget.Describe", "method.go")

	if !findEdge(edges, widgetID, "contains", describeID) {
		t.Fatalf("expected contains edge %s -> %s (Widget -> cross-file Describe), got %+v", widgetID, describeID, edges)
	}
}
