// Package csharpextract_test is an EXTERNAL test package (not `package
// csharpextract`, the internal test package csharpextract_test.go uses)
// deliberately: it drives the real internal/indexer.Run pipeline
// end-to-end, and internal/indexer itself imports csharpextract
// (languages_csharp.go) — a same-package (internal) test file importing
// internal/indexer would create an import cycle. The external-test-package
// pattern is the standard Go mechanism for exercising a package's consumer
// from its own test tree without that cycle (mirrors javaextract's own
// resolution_test.go).
package csharpextract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// csharpFixtureFile is one file this test writes into a temp repo root.
type csharpFixtureFile struct {
	relPath, src string
}

func writeCSharpFixture(t *testing.T, files []csharpFixtureFile) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f.relPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir for fixture %s: %v", f.relPath, err)
		}
		if err := os.WriteFile(abs, []byte(f.src), 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", f.relPath, err)
		}
	}
	return root
}

// nodeKey identifies a committed node by its stable, human-legible fields
// (never by its opaque hash id, which this test never needs to compute).
type nodeKey struct {
	kind, name, filePath string
}

// buildCSharpGraph runs the real indexer.Run pipeline against root into a
// fresh temp store and returns every committed node (indexed by nodeKey)
// and edge, for direct inspection.
func buildCSharpGraph(t *testing.T, root string) (map[nodeKey]*schema.Node, []*schema.Edge) {
	t.Helper()

	storeDir := t.TempDir()
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	reader, err := store.Snapshot()
	if err != nil {
		t.Fatalf("store.Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })

	nodes := make(map[nodeKey]*schema.Node)
	nit, err := reader.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		n := nit.Node()
		nodes[nodeKey{kind: n.Kind, name: n.Name, filePath: n.FilePath}] = n
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}

	var edges []*schema.Edge
	eit, err := reader.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		edges = append(edges, eit.Edge())
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}

	return nodes, edges
}

func hasEdge(edges []*schema.Edge, source, target, kind string) bool {
	for _, e := range edges {
		if e.Source == source && e.Target == target && e.Kind == kind {
			return true
		}
	}
	return false
}

// TestResolve_SameNamespaceCrossFileCall proves a same-namespace,
// cross-file PascalCase-qualified call (`Helper.Assist()`, no `using`
// needed within the same namespace) resolves into a real "calls" edge
// through the C# namespace module key (Task 2's first must_have truth).
func TestResolve_SameNamespaceCrossFileCall(t *testing.T) {
	root := writeCSharpFixture(t, []csharpFixtureFile{
		{relPath: "Helper.cs", src: "namespace Example;\n\npublic class Helper {\n\tpublic static void Assist() {}\n}\n"},
		{relPath: "Caller.cs", src: "namespace Example;\n\npublic class Caller {\n\tpublic void Run() {\n\t\tHelper.Assist();\n\t}\n}\n"},
	})

	nodes, edges := buildCSharpGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "method", name: "Assist", filePath: "Helper.cs"}]
	if !ok {
		t.Fatal("Helper.Assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "Run", filePath: "Caller.cs"}]
	if !ok {
		t.Fatal("Caller.Run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Caller.Run -> Helper.Assist (same-namespace cross-file), got edges: %+v", edges)
	}
}

// TestResolve_FullyQualifiedCrossNamespaceCall proves a FULLY-QUALIFIED
// cross-namespace call (`Other.Namespace.Helper.Assist()` — its own AST
// shape spells out the declaring namespace, no `using` needed) resolves
// into a real "calls" edge via the C# namespace module key (Task 2's
// "cross-namespace... resolves into calls edges" truth, csharpextract's
// documented fully-qualified-prefix mechanism — see types.go's package doc
// comment for why a bare `using`-shortened cross-namespace call is NOT
// resolvable without a full symbol table this extractor does not build).
func TestResolve_FullyQualifiedCrossNamespaceCall(t *testing.T) {
	root := writeCSharpFixture(t, []csharpFixtureFile{
		{relPath: "helper/Helper.cs", src: "namespace Other.Namespace;\n\npublic class Helper {\n\tpublic static void Assist() {}\n}\n"},
		{relPath: "caller/Caller.cs", src: "namespace Example;\n\npublic class Caller {\n\tpublic void Run() {\n\t\tOther.Namespace.Helper.Assist();\n\t}\n}\n"},
	})

	nodes, edges := buildCSharpGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "method", name: "Assist", filePath: "helper/Helper.cs"}]
	if !ok {
		t.Fatal("Helper.Assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "Run", filePath: "caller/Caller.cs"}]
	if !ok {
		t.Fatal("Caller.Run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Caller.Run -> Other.Namespace.Helper.Assist (fully-qualified cross-namespace), got edges: %+v", edges)
	}
}

// TestResolve_FullyQualifiedCrossNamespaceInheritance proves a class that
// extends a FULLY-QUALIFIED base type from a different namespace resolves
// into a real "embeds" edge (the RefKindEmbeds shape RESEARCH Pattern 2
// promotes to "implements" only at Wave 6's resolve-time pass, out of this
// plan's scope) — the "inheritance" leg of Task 2's cross-file resolution
// truth.
func TestResolve_FullyQualifiedCrossNamespaceInheritance(t *testing.T) {
	root := writeCSharpFixture(t, []csharpFixtureFile{
		{relPath: "base/Base.cs", src: "namespace Other.Namespace;\n\npublic class Base {\n\tpublic void BaseMethod() {}\n}\n"},
		{relPath: "derived/Derived.cs", src: "namespace Example;\n\npublic class Derived : Other.Namespace.Base {}\n"},
	})

	nodes, edges := buildCSharpGraph(t, root)

	base, ok := nodes[nodeKey{kind: "struct", name: "Base", filePath: "base/Base.cs"}]
	if !ok {
		t.Fatal("Base node not found in committed graph")
	}
	derived, ok := nodes[nodeKey{kind: "struct", name: "Derived", filePath: "derived/Derived.cs"}]
	if !ok {
		t.Fatal("Derived node not found in committed graph")
	}

	if !hasEdge(edges, derived.Id, base.Id, "embeds") {
		t.Errorf("expected an embeds edge Derived -> Base (fully-qualified cross-namespace inheritance), got edges: %+v", edges)
	}
}

// TestResolve_PartialClassBothFragmentsCallable proves the Pitfall 5
// scheme (b) shared node (types.go/csharpextract_test.go's
// TestExtract_PartialClass_SharedNodeIdentity) survives the FULL indexer.Run
// pipeline: a same-namespace caller in a THIRD file can reach a method
// declared in EITHER partial fragment via a single "calls" edge, proving
// no data loss across fragments once the graph is actually committed (not
// just at the Pass-1 FileResult level).
func TestResolve_PartialClassBothFragmentsCallable(t *testing.T) {
	root := writeCSharpFixture(t, []csharpFixtureFile{
		{relPath: "Widget.cs", src: "namespace Example;\n\npublic partial class Widget {\n\tpublic static void FromFragmentOne() {}\n}\n"},
		{relPath: "Widget.Designer.cs", src: "namespace Example;\n\npublic partial class Widget {\n\tpublic static void FromFragmentTwo() {}\n}\n"},
		{relPath: "Caller.cs", src: "namespace Example;\n\npublic class Caller {\n\tpublic void Run() {\n\t\tWidget.FromFragmentOne();\n\t\tWidget.FromFragmentTwo();\n\t}\n}\n"},
	})

	nodes, edges := buildCSharpGraph(t, root)

	fromOne, ok := nodes[nodeKey{kind: "method", name: "FromFragmentOne", filePath: "Widget.cs"}]
	if !ok {
		t.Fatal("Widget.FromFragmentOne node not found in committed graph")
	}
	fromTwo, ok := nodes[nodeKey{kind: "method", name: "FromFragmentTwo", filePath: "Widget.Designer.cs"}]
	if !ok {
		t.Fatal("Widget.FromFragmentTwo node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "Run", filePath: "Caller.cs"}]
	if !ok {
		t.Fatal("Caller.Run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, fromOne.Id, "calls") {
		t.Errorf("expected a calls edge Caller.Run -> Widget.FromFragmentOne (fragment 1), got edges: %+v", edges)
	}
	if !hasEdge(edges, run.Id, fromTwo.Id, "calls") {
		t.Errorf("expected a calls edge Caller.Run -> Widget.FromFragmentTwo (fragment 2), got edges: %+v", edges)
	}
}
