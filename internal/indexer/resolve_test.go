package indexer

import (
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
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

// --- Task 2: deterministic edge collapse + single-writer batched commit ---

// candidateEdges builds the shared set of duplicate-triple call-site
// candidates TestEdgeCollapse_Deterministic and
// TestEdgeCollapse_OrderIndependent both exercise: three call sites from
// the same (source, "calls", target) triple, at lines 30, 12, and 21, plus
// one unrelated edge that must survive collapse untouched.
func candidateEdges() []*schema.Edge {
	return []*schema.Edge{
		{Source: "fn:caller", Kind: "calls", Target: "fn:callee", Line: 30, Col: 5, Provenance: "ast"},
		{Source: "fn:caller", Kind: "calls", Target: "fn:callee", Line: 12, Col: 2, Provenance: "ast"},
		{Source: "fn:caller", Kind: "calls", Target: "fn:callee", Line: 21, Col: 9, Provenance: "ast"},
		{Source: "fn:other", Kind: "calls", Target: "fn:unrelated", Line: 1, Col: 0, Provenance: "ast"},
	}
}

func candidateNodeFilePath() map[string]string {
	return map[string]string{
		"fn:caller":    "pkg/caller.go",
		"fn:other":     "pkg/caller.go",
		"fn:callee":    "pkg/callee.go",
		"fn:unrelated": "pkg/other.go",
	}
}

// TestEdgeCollapse_Deterministic proves duplicate (source, kind, target)
// candidates collapse to ONE edge whose representative line/col is chosen
// by sorting candidates by (filePath, line, col) and taking the first —
// never processing order (D-05, RESEARCH Pitfall 1).
func TestEdgeCollapse_Deterministic(t *testing.T) {
	collapsed := collapseEdges(candidateEdges(), candidateNodeFilePath())

	var got *schema.Edge
	for _, e := range collapsed {
		if e.Source == "fn:caller" && e.Kind == "calls" && e.Target == "fn:callee" {
			got = e
		}
	}
	if got == nil {
		t.Fatalf("expected exactly one collapsed edge fn:caller -calls-> fn:callee, got %+v", collapsed)
	}
	if got.Line != 12 || got.Col != 2 {
		t.Errorf("representative = (line %d, col %d), want (line 12, col 2) — the (filePath,line,col)-sorted first candidate", got.Line, got.Col)
	}

	// The unrelated edge must survive collapse untouched.
	foundUnrelated := false
	for _, e := range collapsed {
		if e.Source == "fn:other" && e.Kind == "calls" && e.Target == "fn:unrelated" {
			foundUnrelated = true
		}
	}
	if !foundUnrelated {
		t.Errorf("expected the unrelated edge to survive collapse, got %+v", collapsed)
	}
}

// TestEdgeCollapse_OrderIndependent proves shuffling the candidate input
// order never changes the chosen representative (order-independent total
// order, not processing/goroutine/map order).
func TestEdgeCollapse_OrderIndependent(t *testing.T) {
	nodeFilePath := candidateNodeFilePath()
	rng := rand.New(rand.NewSource(1))

	for trial := 0; trial < 10; trial++ {
		edges := candidateEdges()
		rng.Shuffle(len(edges), func(i, j int) { edges[i], edges[j] = edges[j], edges[i] })

		collapsed := collapseEdges(edges, nodeFilePath)
		var got *schema.Edge
		for _, e := range collapsed {
			if e.Source == "fn:caller" && e.Kind == "calls" && e.Target == "fn:callee" {
				got = e
			}
		}
		if got == nil {
			t.Fatalf("trial %d: expected collapsed edge, got %+v", trial, collapsed)
		}
		if got.Line != 12 || got.Col != 2 {
			t.Errorf("trial %d: representative = (line %d, col %d), want (line 12, col 2) regardless of input shuffle", trial, got.Line, got.Col)
		}
	}
}

// errStubWrite is the sentinel error stubWriter returns for whichever
// Put* method its failOn field names.
var errStubWrite = errors.New("stub write error")

// stubWriter is an in-memory graphstore.Writer stand-in so writeGraph's
// single-writer/batched-commit contract can be tested without a real
// Pebble-backed store. failOn, when non-empty, names the Put* method that
// should fail (simulating a mid-write staging error).
type stubWriter struct {
	nodes []*schema.Node
	edges []*schema.Edge
	files []*schema.File
	meta  *schema.Meta

	commitCalls int
	closeCalls  int

	failOn string
}

func (w *stubWriter) PutNode(n *schema.Node) error {
	if w.failOn == "PutNode" {
		return errStubWrite
	}
	w.nodes = append(w.nodes, n)
	return nil
}

func (w *stubWriter) PutEdge(e *schema.Edge, ownerPath string) error {
	if w.failOn == "PutEdge" {
		return errStubWrite
	}
	w.edges = append(w.edges, e)
	return nil
}

func (w *stubWriter) PutFile(f *schema.File) error {
	if w.failOn == "PutFile" {
		return errStubWrite
	}
	w.files = append(w.files, f)
	return nil
}

func (w *stubWriter) PutMeta(m *schema.Meta) error {
	if w.failOn == "PutMeta" {
		return errStubWrite
	}
	w.meta = m
	return nil
}

func (w *stubWriter) DeleteFileSubgraph(path string) error { return nil }

func (w *stubWriter) DeleteNode(id string) error { return nil }

func (w *stubWriter) DeleteEdge(source, kind, target string) error { return nil }

func (w *stubWriter) Commit() error {
	w.commitCalls++
	return nil
}

func (w *stubWriter) Close() error {
	w.closeCalls++
	return nil
}

// stubStore is a graphstore.GraphStore stand-in whose NewWriter always
// returns the same injected stubWriter, so a test can inspect it after
// writeGraph returns.
type stubStore struct {
	writer *stubWriter
}

func (s *stubStore) Snapshot() (graphstore.Reader, error)  { return nil, nil }
func (s *stubStore) NewWriter() (graphstore.Writer, error) { return s.writer, nil }
func (s *stubStore) Export(w io.Writer) error              { return nil }
func (s *stubStore) Close() error                          { return nil }

// TestSingleWriter_CommitsOnce proves the entire resolved graph is written
// through exactly ONE GraphStore.Writer with a single Commit() — no
// per-symbol commits (D-04a) — and that Meta is stamped with correct
// node/edge counts and schema_version == 1.
func TestSingleWriter_CommitsOnce(t *testing.T) {
	w := &stubWriter{}
	store := &stubStore{writer: w}

	nodes := []*schema.Node{
		{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"},
		{Id: "fn:b", Kind: "function", Name: "b", FilePath: "pkg/a.go"},
	}
	packageNodes := []*schema.Node{
		{Id: "package:x", Kind: "package", Name: "x", QualifiedName: "example.com/x"},
	}
	edges := []*schema.Edge{
		{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 30, Col: 1, Provenance: "ast"},
		{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 12, Col: 1, Provenance: "ast"},
	}
	files := []*schema.File{
		{Path: "pkg/a.go", ContentHash: "deadbeef", Language: "go", NodeCount: 2, EdgeCount: 1},
	}

	if err := writeGraph(store, nodes, packageNodes, edges, files); err != nil {
		t.Fatalf("writeGraph returned error: %v", err)
	}

	if w.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", w.commitCalls)
	}
	if w.closeCalls != 0 {
		t.Errorf("closeCalls = %d, want 0 (Close is only for the error path)", w.closeCalls)
	}
	if len(w.nodes) != len(nodes)+len(packageNodes) {
		t.Errorf("staged %d nodes, want %d (symbol + package pseudo-nodes)", len(w.nodes), len(nodes)+len(packageNodes))
	}
	if len(w.files) != len(files) {
		t.Errorf("staged %d files, want %d", len(w.files), len(files))
	}
	if len(w.edges) != 1 {
		t.Fatalf("staged %d edges, want 1 (the duplicate (fn:a,calls,fn:b) pair must collapse)", len(w.edges))
	}
	if w.edges[0].Line != 12 {
		t.Errorf("collapsed edge Line = %d, want 12 (the (filePath,line,col)-sorted first candidate)", w.edges[0].Line)
	}
	if w.meta == nil {
		t.Fatal("PutMeta was never called")
	}
	if w.meta.SchemaVersion != schema.SchemaVersion {
		t.Errorf("Meta.SchemaVersion = %d, want %d", w.meta.SchemaVersion, schema.SchemaVersion)
	}
	if w.meta.NodeCount != int64(len(nodes)+len(packageNodes)) {
		t.Errorf("Meta.NodeCount = %d, want %d", w.meta.NodeCount, len(nodes)+len(packageNodes))
	}
	if w.meta.EdgeCount != 1 {
		t.Errorf("Meta.EdgeCount = %d, want 1", w.meta.EdgeCount)
	}
}

// TestSingleWriter_CloseOnStagingError proves a staging error mid-write
// calls Writer.Close() (releasing the batch) instead of Commit(), and
// writeGraph returns the error — never a partial commit.
func TestSingleWriter_CloseOnStagingError(t *testing.T) {
	w := &stubWriter{failOn: "PutEdge"}
	store := &stubStore{writer: w}

	nodes := []*schema.Node{{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"}}
	edges := []*schema.Edge{{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 1, Provenance: "ast"}}
	files := []*schema.File{{Path: "pkg/a.go"}}

	err := writeGraph(store, nodes, nil, edges, files)
	if err == nil {
		t.Fatal("expected writeGraph to return the staging error, got nil")
	}
	if w.commitCalls != 0 {
		t.Errorf("commitCalls = %d, want 0 (must not commit after a staging error)", w.commitCalls)
	}
	if w.closeCalls != 1 {
		t.Errorf("closeCalls = %d, want 1 (must release the abandoned batch)", w.closeCalls)
	}
}

// TestResolve_EndToEnd proves the exported Resolve entry point runs the
// full Pass 2 pipeline against a REAL GraphStore (not a stub), over the
// shared multi-package fixture.
func TestResolve_EndToEnd(t *testing.T) {
	results, modulePath := fixtureResults(t)

	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	if _, err := Resolve(store, results, modulePath); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	meta, err := (func() (*schema.Meta, error) {
		r, err := store.Snapshot()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return r.GetMeta()
	})()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if meta.SchemaVersion != schema.SchemaVersion {
		t.Errorf("Meta.SchemaVersion = %d, want %d", meta.SchemaVersion, schema.SchemaVersion)
	}
	if meta.NodeCount == 0 {
		t.Error("Meta.NodeCount = 0, want > 0")
	}
	if meta.EdgeCount == 0 {
		t.Error("Meta.EdgeCount = 0, want > 0")
	}
}
