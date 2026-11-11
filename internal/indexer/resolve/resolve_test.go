package resolve

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
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

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "pkgb/pkgb.go", "()")
	alphaID := nodeid.DeclID(goextract.KindFunction, "Alpha", "pkga/pkga.go", "()")

	if !findEdge(edges, runID, "calls", alphaID) {
		t.Fatalf("expected calls edge %s -> %s (pkgb.Run -> pkga.Alpha), got %+v", runID, alphaID, edges)
	}
}

// TestResolve_IntraPackageCall proves an unqualified call resolves against
// the calling file's own package symbols.
func TestResolve_IntraPackageCall(t *testing.T) {
	results, modulePath := fixtureResults(t)
	_, _, edges, _, _ := resolveRefs(results, modulePath)

	alphaID := nodeid.DeclID(goextract.KindFunction, "Alpha", "pkga/pkga.go", "()")
	helperID := nodeid.DeclID(goextract.KindFunction, "helper", "pkga/pkga.go", "()")

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
	result := extractGo(t, p, "p.go", src)

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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	aGreetID := nodeid.DeclID(goextract.KindMethod, "A.Greet", "p.go", "()")
	bGreetID := nodeid.DeclID(goextract.KindMethod, "B.Greet", "p.go", "()")

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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	aGreetID := nodeid.DeclID(goextract.KindMethod, "A.Greet", "p.go", "()")
	bGreetID := nodeid.DeclID(goextract.KindMethod, "B.Greet", "p.go", "()")

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
	b := symbols.NewBuilder()
	for id, arity := range map[string]int32{baseFoo1: 1, baseFoo2: 2, derivedFoo1: 1, derivedFoo2: 2} {
		b.AddNode(&schema.Node{Id: id, Kind: goextract.KindMethod, Name: "Foo"})
		b.SetArity(id, arity)
	}

	tb, _ := newTables(edges, nil)
	synthesized := synthesizeOverrides(b.Freeze(), tb, tb.supertypes(nil, nil), nil)

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
	result := extractGo(t, p, "p.go", src)

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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	emptyID := nodeid.NodeID(goextract.KindStruct, "Empty", "p.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "p.go")

	// Positive control: the view holds the method specs of Reader, so the
	// absence of the edge below comes from Empty having no Read method and
	// not from an interface that looks empty.
	b := symbols.NewBuilder()
	b.AddResults(declarationsOf([]goextract.FileResult{result}))
	if specs := b.Freeze().InterfaceMethods()[readerID]; len(specs) != 1 || specs[0].Name != "Read" {
		t.Fatalf("the view holds the method specs %+v for Reader, want [Read]", specs)
	}

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
// the same-module lookup — it resolves only once
// the retry walks Derived's own "embeds" edge to Base and finds
// baseMethod there. unresolvedCount ends at 0; a single-pass-only run
// (i.e. this fixture minus the retry step) would leave it at 1 — this is
// exactly the "unresolvedCount drops" acceptance criterion, verified by
// controlling the fixture precisely enough that the retry is the ONLY
// path that could resolve this call.
func TestResolve_ConformanceRetryResolvesInheritedCall(t *testing.T) {
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "base.java")
	baseMethodID := nodeid.DeclID(goextract.KindMethod, "Base.baseMethod", "base.java", "()")
	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "derived.java")
	runID := nodeid.DeclID(goextract.KindMethod, "Derived.run", "derived.java", "()")

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
	result := extractGo(t, p, "p.go", src)

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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "p.go", "()")
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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "p.go", "()")
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
	runID := nodeid.DeclID(goextract.KindFunction, "Run", "p.go", "()")

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
	result := extractGo(t, p, "p.go", src)

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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	makeWidgetID := nodeid.DeclID(goextract.KindFunction, "MakeWidget", "p.go", "()")
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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, unresolved := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	fID := nodeid.DeclID(goextract.KindFunction, "F", "p.go", "()")
	describeID := nodeid.DeclID(goextract.KindMethod, "Widget.Describe", "p.go", "()")

	// design.md D17: the local w has the declared type Widget, so the call
	// resolves to Widget.Describe through the receiver type.
	if !findEdge(edges, fID, "calls", describeID) {
		t.Fatalf("expected the call w.Describe() through `var w Widget` to resolve to Widget.Describe, got %+v", edges)
	}
	if unresolved != 0 {
		t.Errorf("expected no unresolved reference, got %d", unresolved)
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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, unresolved := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "p.go", "()")
	barID := nodeid.DeclID(goextract.KindMethod, "Widget.Bar", "p.go", "()")

	// design.md D17: foo() returns Widget, which pass 2 fills into the
	// deferred receiver type, so the call resolves to Widget.Bar.
	if !findEdge(edges, runID, "calls", barID) {
		t.Fatalf("expected foo().Bar() to resolve to Widget.Bar through foo's return type, got edges %+v", edges)
	}
	if unresolved != 0 {
		t.Errorf("expected no unresolved reference, got %d", unresolved)
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
	result := extractGo(t, p, "p.go", src)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{result}, "example.com/p")

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "p.go", "()")
	outerID := nodeid.DeclID(goextract.KindFunction, "outer", "p.go", "(int)")
	innerID := nodeid.DeclID(goextract.KindFunction, "inner", "p.go", "()")

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
	typeResult := extractGo(t, p, "type.go", typeSrc)
	methodResult := extractGo(t, p, "method.go", methodSrc)

	_, _, edges, _, _ := resolveRefs([]goextract.FileResult{typeResult, methodResult}, "example.com/p")

	widgetID := nodeid.NodeID(goextract.KindStruct, "Widget", "type.go")
	describeID := nodeid.DeclID(goextract.KindMethod, "Widget.Describe", "method.go", "()")

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

// TestEdgeCollapse_Deterministic proves duplicate (source, kind, target)
// candidates collapse to ONE edge whose representative line/col is chosen
// by sorting candidates by (line, col) and taking the first —
// never processing order (D-05, RESEARCH Pitfall 1).
func TestEdgeCollapse_Deterministic(t *testing.T) {
	collapsed := CollapseEdges(candidateEdges())

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
	rng := rand.New(rand.NewSource(1))

	for trial := 0; trial < 10; trial++ {
		edges := candidateEdges()
		rng.Shuffle(len(edges), func(i, j int) { edges[i], edges[j] = edges[j], edges[i] })

		collapsed := CollapseEdges(edges)
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

// TestResolve_RecordsUnresolvedNames proves a File record lists the names
// its references could not resolve, for every reference kind that
// resolves by name: an unqualified name under the file's own module key,
// an import-qualified name under the imported path, sorted and listed
// once. A call through a local variable is not listed, since no symbol
// can resolve it by name.
func TestResolve_RecordsUnresolvedNames(t *testing.T) {
	src := `package p

import "example.com/m/q"

type S struct {
	Base
}

func (r Recv) M() {}

func G() q.Thing {
	q.Absent()
	Missing()
	Missing()
	_ = Gone
	_ = Lit{}
	var w q.Widget
	w.Describe()
	return q.Thing{}
}
`
	p := newTestParser(t)
	result := extractGoIn(t, p, "example.com/m/p", "p/p.go", src)

	_, _, _, files, _ := resolveRefs([]goextract.FileResult{result}, "example.com/m")
	if len(files) != 1 {
		t.Fatalf("got %d File records, want 1", len(files))
	}
	got := unresolvedNameStrings(files[0])
	want := []string{
		"example.com/m/p Base",
		"example.com/m/p Gone",
		"example.com/m/p Lit",
		"example.com/m/p Missing",
		"example.com/m/p Recv",
		"example.com/m/q Absent",
		// w has the type q.Widget, so the call w.Describe() waits on
		// Describe in q, not on a Describe in p (design.md D17).
		"example.com/m/q Describe",
		"example.com/m/q Thing",
	}
	if !slices.Equal(got, want) {
		t.Errorf("unresolved names = %q, want %q", got, want)
	}
}
