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

// TestResolveExtends_StructTarget proves `type Derived struct { Base }`
// (a class/struct-extends-class/struct RefKindEmbeds ref) yields an
// EdgeKindExtends edge Derived -> Base, NOT "embeds" and NOT "implements"
// — D-09's split of the old unconditional "embeds" fallback (formerly
// TestResolve_StructEmbeds, which asserted the pre-D-09 "embeds" edge;
// this is the intentional behavior change RESEARCH §B documents).
func TestResolveExtends_StructTarget(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "pkga/embed.go")
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "pkga/embed.go")

	if findEdge(edges, derivedID, "embeds", baseID) {
		t.Fatalf("expected NO plain embeds edge %s -> %s (Derived -> Base) — D-09 reclassifies class/struct targets as extends", derivedID, baseID)
	}
	if findEdge(edges, derivedID, goextract.EdgeKindImplements, baseID) {
		t.Fatalf("expected NO implements edge %s -> %s (Derived -> Base) — Base is not an interface", derivedID, baseID)
	}
	e := findEdgeFull(edges, derivedID, goextract.EdgeKindExtends, baseID)
	if e == nil {
		t.Fatalf("expected extends edge %s -> %s (Derived -> Base), got %+v", derivedID, baseID, edges)
	}
	if e.Provenance != "heuristic" {
		t.Errorf("Provenance = %q, want %q (D-09 synthesis discipline)", e.Provenance, "heuristic")
	}
	if e.Metadata["synthesizedBy"] != "declared-extends" {
		t.Errorf("Metadata[synthesizedBy] = %q, want %q", e.Metadata["synthesizedBy"], "declared-extends")
	}
}

// TestResolveExtends_StructEmbedsInterfaceStaysImplements proves the
// interface-target promotion path is unregressed by the D-09 split: a
// struct embedding an interface value still promotes to "implements", not
// "extends".
func TestResolveExtends_StructEmbedsInterfaceStaysImplements(t *testing.T) {
	src := `package p

type Reader interface {
	Read() int
}

type Wrapper struct {
	Reader
}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	wrapperID := nodeid.NodeID(goextract.KindStruct, "Wrapper", "p.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "p.go")

	if !findEdge(edges, wrapperID, goextract.EdgeKindImplements, readerID) {
		t.Fatalf("expected implements edge %s -> %s (Wrapper embeds Reader interface, unregressed), got %+v", wrapperID, readerID, edges)
	}
	if findEdge(edges, wrapperID, goextract.EdgeKindExtends, readerID) {
		t.Fatalf("expected NO extends edge %s -> %s (target is an interface), got %+v", wrapperID, readerID, edges)
	}
}

// TestResolveExtends_InterfaceEmbedsInterfaceUnregressed proves
// interface-embeds-interface (both ends are interfaces) stays a plain
// "embeds" edge, unregressed by the D-09 split — neither the implements
// promotion condition (source must NOT be an interface) nor the new
// extends condition (target must NOT be an interface) matches this case.
func TestResolveExtends_InterfaceEmbedsInterfaceUnregressed(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	rwID := nodeid.NodeID(goextract.KindInterface, "ReadWriter", "pkga/embed.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "pkga/embed.go")

	if !findEdge(edges, rwID, "embeds", readerID) {
		t.Fatalf("expected embeds edge %s -> %s (ReadWriter -> Reader, unregressed), got %+v", rwID, readerID, edges)
	}
	if findEdge(edges, rwID, goextract.EdgeKindExtends, readerID) {
		t.Fatalf("expected NO extends edge %s -> %s (target is an interface), got %+v", rwID, readerID, edges)
	}
}

// TestResolveOverrides_StructMethodOverridesSupertypeMethod proves a
// method on a struct that extends another struct, sharing a supertype
// method's name+arity, synthesizes an EdgeKindOverrides edge
// method -> supertype-method carrying heuristic provenance + a
// synthesizedBy tag (D-09, mirroring synthesizeGoImplements's
// composition-from-already-resolved-edges pattern).
func TestResolveOverrides_StructMethodOverridesSupertypeMethod(t *testing.T) {
	src := `package p

type A struct{}

func (a A) Greet() string { return "hi" }

type B struct {
	A
}

func (b B) Greet() string { return "hello" }
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	aGreetID := nodeid.NodeID(goextract.KindMethod, "A.Greet", "p.go")
	bGreetID := nodeid.NodeID(goextract.KindMethod, "B.Greet", "p.go")

	e := findEdgeFull(edges, bGreetID, goextract.EdgeKindOverrides, aGreetID)
	if e == nil {
		t.Fatalf("expected overrides edge %s -> %s (B.Greet overrides A.Greet), got %+v", bGreetID, aGreetID, edges)
	}
	if e.Provenance != "heuristic" {
		t.Errorf("Provenance = %q, want %q (D-09 synthesis discipline)", e.Provenance, "heuristic")
	}
	if e.Metadata["synthesizedBy"] == "" {
		t.Errorf("Metadata[synthesizedBy] is empty, want a non-empty synthesizedBy tag")
	}
}

// TestResolveOverrides_ArityMismatchNoEdge proves Go's structural
// name+arity match (RESEARCH §B's documented precision note) does NOT
// synthesize an overrides edge when a subtype method shares its
// supertype's name but NOT its arity — a genuinely different method, not
// an override.
func TestResolveOverrides_ArityMismatchNoEdge(t *testing.T) {
	src := `package p

type A struct{}

func (a A) Greet() string { return "hi" }

type B struct {
	A
}

func (b B) Greet(name string) string { return "hello " + name }
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	aGreetID := nodeid.NodeID(goextract.KindMethod, "A.Greet", "p.go")
	bGreetID := nodeid.NodeID(goextract.KindMethod, "B.Greet", "p.go")

	if e := findEdgeFull(edges, bGreetID, goextract.EdgeKindOverrides, aGreetID); e != nil {
		t.Fatalf("expected NO overrides edge for an arity-mismatched same-named method, got %+v", e)
	}
}

// TestResolveOverrides_OverloadedMethodsBothSynthesized proves CR-01's
// fix: when a supertype declares two overloads of the same method name
// (legal in Java/C#, impossible in Go — hence this drives
// synthesizeOverrides directly with hand-built edges rather than through
// the Go extractor) and a subtype overrides both, synthesizeOverrides
// must emit BOTH overrides edges to their respective same-arity targets,
// not collapse the name->methodID map to a single arbitrary overload on
// either the subtype or supertype side.
//
// Edge order is deliberately arranged so the pre-fix map[string]string
// (methodName -> last-written methodID) keeps a MISMATCHED arity pair on
// each side (Base collapses to its arity-2 overload, Derived collapses
// to its arity-1 overload) — under the old code this yields ZERO
// overrides edges even though both real overrides exist, proving the
// bug isn't merely "misses one edge" but can silently drop all of them.
func TestResolveOverrides_OverloadedMethodsBothSynthesized(t *testing.T) {
	const (
		baseType    = "Base"
		derivedType = "Derived"
		baseFoo1    = "base.Foo/1" // Foo(x) — arity 1
		baseFoo2    = "base.Foo/2" // Foo(x, y) — arity 2
		derivedFoo1 = "derived.Foo/1"
		derivedFoo2 = "derived.Foo/2"
	)

	edges := []*schema.Edge{
		{Source: baseType, Target: baseFoo1, Kind: "contains"},
		{Source: baseType, Target: baseFoo2, Kind: "contains"},
		{Source: derivedType, Target: derivedFoo2, Kind: "contains"},
		{Source: derivedType, Target: derivedFoo1, Kind: "contains"},
		{Source: derivedType, Target: baseType, Kind: goextract.EdgeKindExtends},
	}
	nodeKindByID := map[string]string{
		baseFoo1: goextract.KindMethod, baseFoo2: goextract.KindMethod,
		derivedFoo1: goextract.KindMethod, derivedFoo2: goextract.KindMethod,
	}
	nodeNameByID := map[string]string{
		baseFoo1: "Foo", baseFoo2: "Foo", derivedFoo1: "Foo", derivedFoo2: "Foo",
	}
	methodArity := map[string]int32{
		baseFoo1: 1, baseFoo2: 2, derivedFoo1: 1, derivedFoo2: 2,
	}
	nodeStartLineByID := map[string]int32{}

	synthesized := synthesizeOverrides(edges, nodeKindByID, nodeNameByID, methodArity, nodeStartLineByID)

	if e := findEdgeFull(synthesized, derivedFoo1, goextract.EdgeKindOverrides, baseFoo1); e == nil {
		t.Errorf("expected overrides edge %s -> %s (arity-1 overload), got %+v", derivedFoo1, baseFoo1, synthesized)
	}
	if e := findEdgeFull(synthesized, derivedFoo2, goextract.EdgeKindOverrides, baseFoo2); e == nil {
		t.Errorf("expected overrides edge %s -> %s (arity-2 overload), got %+v", derivedFoo2, baseFoo2, synthesized)
	}
	if len(synthesized) != 2 {
		t.Errorf("len(synthesized) = %d, want 2 (both overloads' overrides edges, no cross-arity misattribution): %+v", len(synthesized), synthesized)
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

// findEdgeFull returns the first edge matching (source, kind, target)
// exactly, or nil — like findEdge, but returns the edge itself so a test
// can assert on Provenance/Metadata/Line (RES-03).
func findEdgeFull(edges []*schema.Edge, source, kind, target string) *schema.Edge {
	for _, e := range edges {
		if e.Source == source && e.Kind == kind && e.Target == target {
			return e
		}
	}
	return nil
}

// TestResolve_GoStructuralImplements proves RES-02 Pattern 3: a struct
// whose method set structurally satisfies an interface's method-spec set
// is synthesized as an "implements" edge, carrying heuristic provenance +
// a synthesizedBy tag + a non-zero Line anchored at the struct's own
// declaration (RES-03) — distinct from every ground-truth "ast" edge.
func TestResolve_GoStructuralImplements(t *testing.T) {
	src := `package p

type Reader interface {
	Read() int
}

type File struct{}

func (f File) Read() int { return 0 }
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	fileStructID := nodeid.NodeID(goextract.KindStruct, "File", "p.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "p.go")

	e := findEdgeFull(edges, fileStructID, "implements", readerID)
	if e == nil {
		t.Fatalf("expected implements edge %s -> %s (File implements Reader), got %+v", fileStructID, readerID, edges)
	}
	if e.Provenance != "heuristic" {
		t.Errorf("Provenance = %q, want %q (RES-03)", e.Provenance, "heuristic")
	}
	if e.Metadata["synthesizedBy"] != "go-structural-methodset" {
		t.Errorf("Metadata[synthesizedBy] = %q, want %q", e.Metadata["synthesizedBy"], "go-structural-methodset")
	}
	if e.Line == 0 {
		t.Errorf("Line = 0, want a non-zero anchor at File's own declaration (RES-03)")
	}
}

// TestResolve_GoStructuralImplements_NonSuperset proves a struct whose
// method set does NOT satisfy an interface's method spec produces no
// implements edge.
func TestResolve_GoStructuralImplements_NonSuperset(t *testing.T) {
	src := `package p

type Reader interface {
	Read() int
}

type Empty struct{}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	emptyID := nodeid.NodeID(goextract.KindStruct, "Empty", "p.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "p.go")

	if e := findEdgeFull(edges, emptyID, "implements", readerID); e != nil {
		t.Fatalf("expected no implements edge for Empty (no Read method), got %+v", e)
	}
}

// TestResolve_DeclaredImplementsPromotion proves Pattern 2: an
// UnresolvedRef.Kind==RefKindEmbeds ref whose resolved target is an
// interface node (and whose source is not itself an interface) promotes
// to an "implements" edge carrying heuristic provenance + a
// synthesizedBy tag — the shape Java's extends/implements and C#'s
// base_list both emit at extraction time (undistinguished, per 05-04/
// 05-05's javaextract/csharpextract). A class-extends-class reference
// (target is NOT an interface) now promotes to an "extends" edge (D-09
// split of the branch this doc comment used to describe as "stays a
// plain embeds edge" — see TestResolveExtends_StructTarget for the
// dedicated coverage of that case).
func TestResolve_DeclaredImplementsPromotion(t *testing.T) {
	interfaceID := nodeid.NodeID(goextract.KindInterface, "Iface", "iface.go")
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "base.go")
	implID := nodeid.NodeID(goextract.KindStruct, "Impl", "impl.go")
	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "derived.go")

	results := []goextract.FileResult{
		{
			ImportPath: "example.com/p", RelPath: "iface.go", Language: "csharp",
			Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
				Id: interfaceID, Kind: goextract.KindInterface, Name: "Iface", QualifiedName: "Iface", FilePath: "iface.go",
			}}},
		},
		{
			ImportPath: "example.com/p", RelPath: "base.go", Language: "csharp",
			Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
				Id: baseID, Kind: goextract.KindStruct, Name: "Base", QualifiedName: "Base", FilePath: "base.go",
			}}},
		},
		{
			ImportPath: "example.com/p", RelPath: "impl.go", Language: "csharp",
			Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
				Id: implID, Kind: goextract.KindStruct, Name: "Impl", QualifiedName: "Impl", FilePath: "impl.go",
			}}},
			Unresolved: []goextract.UnresolvedRef{
				{FromID: implID, Name: "Iface", Kind: goextract.RefKindEmbeds, Line: 5, Col: 10},
			},
		},
		{
			ImportPath: "example.com/p", RelPath: "derived.go", Language: "csharp",
			Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
				Id: derivedID, Kind: goextract.KindStruct, Name: "Derived", QualifiedName: "Derived", FilePath: "derived.go",
			}}},
			Unresolved: []goextract.UnresolvedRef{
				{FromID: derivedID, Name: "Base", Kind: goextract.RefKindEmbeds, Line: 7, Col: 3},
			},
		},
	}

	_, _, edges, _, _ := resolveRefs(results, "example.com/p")

	implEdge := findEdgeFull(edges, implID, "implements", interfaceID)
	if implEdge == nil {
		t.Fatalf("expected implements edge %s -> %s (Impl -> Iface, promoted), got %+v", implID, interfaceID, edges)
	}
	if implEdge.Provenance != "heuristic" {
		t.Errorf("Provenance = %q, want %q (RES-03)", implEdge.Provenance, "heuristic")
	}
	if implEdge.Metadata["synthesizedBy"] != "declared-implements" {
		t.Errorf("Metadata[synthesizedBy] = %q, want %q", implEdge.Metadata["synthesizedBy"], "declared-implements")
	}
	if implEdge.Line != 5 {
		t.Errorf("Line = %d, want 5 (ref.Line — the extends/implements clause's own source location)", implEdge.Line)
	}

	if e := findEdgeFull(edges, implID, "embeds", interfaceID); e != nil {
		t.Fatalf("expected NO plain embeds edge once promoted to implements, got %+v", e)
	}

	derivedEdge := findEdgeFull(edges, derivedID, goextract.EdgeKindExtends, baseID)
	if derivedEdge == nil {
		t.Fatalf("expected extends edge %s -> %s (Derived -> Base, class extends class, D-09 split), got %+v", derivedID, baseID, edges)
	}
	if derivedEdge.Provenance != "heuristic" {
		t.Errorf("Derived->Base Provenance = %q, want %q (D-09 synthesis discipline)", derivedEdge.Provenance, "heuristic")
	}
	if findEdge(edges, derivedID, "embeds", baseID) {
		t.Fatalf("expected NO plain embeds edge %s -> %s once reclassified as extends, got %+v", derivedID, baseID, edges)
	}
}

// TestResolve_ConformanceRetryResolvesInheritedCall proves Pitfall 3's
// two-pass conformance retry (Task 2): an unqualified call
// (`baseMethod()`) whose target is declared ONLY on a supertype in a
// DIFFERENT module-key scope cannot resolve via pass 1's own
// resolveUnqualified (same-moduleKey-only lookup) — it resolves only once
// the retry walks Derived's own "embeds" edge to Base and finds
// baseMethod there. unresolvedCount ends at 0; a single-pass-only run
// (i.e. this fixture minus the retry step) would leave it at 1 — this is
// exactly the "unresolvedCount drops" acceptance criterion, verified by
// controlling the fixture precisely enough that the retry is the ONLY
// path that could resolve this call.
func TestResolve_ConformanceRetryResolvesInheritedCall(t *testing.T) {
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "base.java")
	baseMethodID := nodeid.NodeID(goextract.KindMethod, "Base.baseMethod", "base.java")
	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "derived.java")
	runID := nodeid.NodeID(goextract.KindMethod, "Derived.run", "derived.java")

	results := []goextract.FileResult{
		{
			ImportPath: "pkgA", RelPath: "base.java", Language: "java",
			Nodes: []goextract.ExtractedNode{
				{Node: &schema.Node{Id: baseID, Kind: goextract.KindStruct, Name: "Base", QualifiedName: "Base", FilePath: "base.java"}},
				{Node: &schema.Node{Id: baseMethodID, Kind: goextract.KindMethod, Name: "baseMethod", QualifiedName: "Base.baseMethod", FilePath: "base.java"}},
			},
			IntraEdges: []goextract.IntraEdge{
				{Edge: &schema.Edge{Source: baseID, Target: baseMethodID, Kind: "contains", Provenance: "ast"}},
			},
		},
		{
			ImportPath: "pkgB", RelPath: "derived.java", Language: "java",
			Nodes: []goextract.ExtractedNode{
				{Node: &schema.Node{Id: derivedID, Kind: goextract.KindStruct, Name: "Derived", QualifiedName: "Derived", FilePath: "derived.java"}},
				{Node: &schema.Node{Id: runID, Kind: goextract.KindMethod, Name: "run", QualifiedName: "Derived.run", FilePath: "derived.java"}},
			},
			IntraEdges: []goextract.IntraEdge{
				{Edge: &schema.Edge{Source: derivedID, Target: runID, Kind: "contains", Provenance: "ast"}},
			},
			Imports: map[string]string{"Base": "pkgA"},
			Unresolved: []goextract.UnresolvedRef{
				{FromID: derivedID, Name: "Base", PkgAlias: "Base", Kind: goextract.RefKindEmbeds, Line: 3, Col: 1},
				{FromID: runID, Name: "baseMethod", Kind: goextract.RefKindCalls, Line: 5, Col: 3},
			},
		},
	}

	_, _, edges, _, unresolved := resolveRefs(results, "example.com/root")

	if !findEdge(edges, runID, "calls", baseMethodID) {
		t.Fatalf("expected calls edge %s -> %s (Derived.run -> Base.baseMethod, resolved via conformance retry), got %+v", runID, baseMethodID, edges)
	}
	if unresolved != 0 {
		t.Errorf("unresolvedCount = %d, want 0 (the retry must resolve the inherited call)", unresolved)
	}
}

// TestResolve_UnresolvedMethodCall (above) already proves the retry does
// NOT introduce a false resolution for a plain function's local-variable-
// receiver call — F() has no enclosing type at all, so methodOwner never
// has an entry for it, and the retry correctly leaves it unresolved
// exactly like pass 1 did.

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

// TestReferences_ResolvesEdge proves a package-level identifier read as a
// value (not called) resolves into a "references" edge with "ast"
// provenance — D-09 Pass-1 capture + Pass-2 resolution (01-RESEARCH.md
// §B), mirroring RefKindCalls' resolve shape exactly.
func TestReferences_ResolvesEdge(t *testing.T) {
	src := `package p

var Shared int

func Run() {
	Consume(Shared)
}
func Consume(x int) {}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "p.go")
	sharedID := nodeid.NodeID(goextract.KindVariable, "Shared", "p.go")

	e := findEdgeFull(edges, runID, goextract.RefKindReferences, sharedID)
	if e == nil {
		t.Fatalf("expected references edge %s -> %s (Run -> Shared), got %+v", runID, sharedID, edges)
	}
	if e.Provenance != "ast" {
		t.Errorf("Provenance = %q, want %q (ground truth, not synthesized)", e.Provenance, "ast")
	}
}

// TestInstantiates_ResolvesToStructKind proves `Widget{}` resolves into an
// "instantiates" edge to Widget's struct node — D-09 Pass-1 capture +
// Pass-2 Kind-check disambiguation (01-RESEARCH.md §B).
func TestInstantiates_ResolvesToStructKind(t *testing.T) {
	src := `package p

type Widget struct{}

func Run() {
	_ = Widget{}
}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "p.go")
	widgetID := nodeid.NodeID(goextract.KindStruct, "Widget", "p.go")

	if !findEdge(edges, runID, goextract.RefKindInstantiates, widgetID) {
		t.Fatalf("expected instantiates edge %s -> %s (Run -> Widget), got %+v", runID, widgetID, edges)
	}
}

// TestInstantiates_NonStructTargetUnresolved proves Pass 2's Kind-check
// disambiguation (RESEARCH §B): an instantiates ref whose resolved target
// is NOT a struct-Kind node (here, an interface — syntactically
// impossible to construct via valid Go composite-literal syntax, so this
// FileResult is hand-built like TestResolve_DeclaredImplementsPromotion)
// produces no edge — counted as unresolved, never silently mis-typed.
func TestInstantiates_NonStructTargetUnresolved(t *testing.T) {
	ifaceID := nodeid.NodeID(goextract.KindInterface, "Iface", "p.go")
	runID := nodeid.NodeID(goextract.KindFunction, "Run", "p.go")

	results := []goextract.FileResult{
		{
			ImportPath: "example.com/p", RelPath: "p.go", Language: "go",
			Nodes: []goextract.ExtractedNode{
				{Node: &schema.Node{Id: ifaceID, Kind: goextract.KindInterface, Name: "Iface", QualifiedName: "Iface", FilePath: "p.go"}},
				{Node: &schema.Node{Id: runID, Kind: goextract.KindFunction, Name: "Run", QualifiedName: "Run", FilePath: "p.go"}},
			},
			Unresolved: []goextract.UnresolvedRef{
				{FromID: runID, Name: "Iface", Kind: goextract.RefKindInstantiates, Line: 3, Col: 5},
			},
		},
	}

	_, _, edges, _, unresolved := resolveRefs(results, "example.com/p")

	if e := findEdgeFull(edges, runID, goextract.RefKindInstantiates, ifaceID); e != nil {
		t.Fatalf("expected no instantiates edge for a non-struct target, got %+v", e)
	}
	if unresolved != 1 {
		t.Errorf("unresolvedCount = %d, want 1 (the Kind-check-rejected ref)", unresolved)
	}
}

// TestTypeOf_ResolvesEdge proves `var W Widget` resolves into a "type_of"
// edge W -> Widget — D-09 Pass-1 capture + Pass-2 resolution
// (01-RESEARCH.md §B).
func TestTypeOf_ResolvesEdge(t *testing.T) {
	src := `package p

type Widget struct{}

var W Widget
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	wID := nodeid.NodeID(goextract.KindVariable, "W", "p.go")
	widgetID := nodeid.NodeID(goextract.KindStruct, "Widget", "p.go")

	if !findEdge(edges, wID, goextract.RefKindTypeOf, widgetID) {
		t.Fatalf("expected type_of edge %s -> %s (W -> Widget), got %+v", wID, widgetID, edges)
	}
}

// TestReturns_ResolvesEdge proves `func MakeWidget() Widget` resolves
// into a "returns" edge MakeWidget -> Widget — D-09 Pass-1 capture (reuse
// of the already-parsed return type) + Pass-2 resolution (01-RESEARCH.md
// §B).
func TestReturns_ResolvesEdge(t *testing.T) {
	src := `package p

type Widget struct{}

func MakeWidget() Widget { return Widget{} }
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	makeWidgetID := nodeid.NodeID(goextract.KindFunction, "MakeWidget", "p.go")
	widgetID := nodeid.NodeID(goextract.KindStruct, "Widget", "p.go")

	if !findEdge(edges, makeWidgetID, goextract.RefKindReturns, widgetID) {
		t.Fatalf("expected returns edge %s -> %s (MakeWidget -> Widget), got %+v", makeWidgetID, widgetID, edges)
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

// TestResolve_SelectorNonIdentifierOperandNeverMisresolvesSamePackage
// proves WR-02: a selector call whose operand is a non-identifier
// expression (`foo().Bar()`, where the operand is itself a call_expression)
// never resolves as a same-package unqualified reference to an unrelated
// symbol that happens to share the bare name "Bar" (here, Widget.Bar) —
// it must stay unresolved, exactly like a local-variable-receiver call.
func TestResolve_SelectorNonIdentifierOperandNeverMisresolvesSamePackage(t *testing.T) {
	src := `package p

type Widget struct{}

func (w Widget) Bar() int { return 1 }

func foo() Widget { return Widget{} }

func Run() {
	foo().Bar()
}
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, unresolved := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "p.go")
	barID := nodeid.NodeID(goextract.KindMethod, "Widget.Bar", "p.go")

	if findEdge(edges, runID, "calls", barID) {
		t.Fatalf("WR-02: foo().Bar() must NOT mis-resolve to Widget.Bar as a same-package unqualified call, got edges %+v", edges)
	}
	if unresolved == 0 {
		t.Errorf("expected the foo().Bar() call ref to be counted as unresolved, got 0")
	}
}

// TestResolve_CallAsArgument locks in that a call passed as an argument to
// another call (`outer(inner())`) resolves into calls edges for BOTH the
// outer and the inner call (D-05's call-as-argument item) — a regression
// test proving this stays true going forward.
func TestResolve_CallAsArgument(t *testing.T) {
	src := `package p

func Run() {
	outer(inner())
}
func outer(x int) int { return x }
func inner() int { return 1 }
`
	p := newTestParser(t)
	result, err := goextract.Extract(p, "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "p.go")
	outerID := nodeid.NodeID(goextract.KindFunction, "outer", "p.go")
	innerID := nodeid.NodeID(goextract.KindFunction, "inner", "p.go")

	if !findEdge(edges, runID, "calls", outerID) {
		t.Errorf("expected calls edge Run -> outer, got %+v", edges)
	}
	if !findEdge(edges, runID, "calls", innerID) {
		t.Errorf("expected calls edge Run -> inner (call-as-argument), got %+v", edges)
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

func (w *stubWriter) PutMigration(data []byte) error { return nil }

func (w *stubWriter) DeleteFileSubgraph(path string) error { return nil }

func (w *stubWriter) DeleteNode(id string) error { return nil }

func (w *stubWriter) DeleteEdge(source, kind, target string) error { return nil }

func (w *stubWriter) DeleteFileIndexEdge(ownerPath, source, kind, target string) error { return nil }

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
