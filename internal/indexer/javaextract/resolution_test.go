// Package javaextract_test is an EXTERNAL test package (not
// `package javaextract`, the internal test package javaextract_test.go
// uses) deliberately: it drives the real internal/indexer.Run pipeline
// end-to-end, and internal/indexer itself imports javaextract
// (languages_java.go) — a same-package (internal) test file importing
// internal/indexer would create an import cycle. The external-test-package
// pattern is the standard Go mechanism for exercising a package's consumer
// from its own test tree without that cycle.
package javaextract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// javaFixtureFile is one file this test writes into a temp repo root.
type javaFixtureFile struct {
	relPath, src string
}

func writeJavaFixture(t *testing.T, files []javaFixtureFile) string {
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

// buildJavaGraph runs the real indexer.Run pipeline against root into a
// fresh temp store and returns every committed node (indexed by nodeKey)
// and edge, for direct inspection.
func buildJavaGraph(t *testing.T, root string) (map[nodeKey]*schema.Node, []*schema.Edge) {
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

// TestResolve_SamePackageCrossFileCall proves a same-package, cross-file
// static-class-qualified call (`Helper.assist()`, no import needed within
// the same package) resolves into a real "calls" edge through the Java
// package-name module key (Task 2's first must_have truth).
func TestResolve_SamePackageCrossFileCall(t *testing.T) {
	root := writeJavaFixture(t, []javaFixtureFile{
		{relPath: "com/example/Helper.java", src: "package com.example;\n\npublic class Helper {\n\tpublic static void assist() {}\n}\n"},
		{relPath: "com/example/Caller.java", src: "package com.example;\n\npublic class Caller {\n\tpublic void run() {\n\t\tHelper.assist();\n\t}\n}\n"},
	})

	nodes, edges := buildJavaGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "method", name: "assist", filePath: "com/example/Helper.java"}]
	if !ok {
		t.Fatal("Helper.assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "run", filePath: "com/example/Caller.java"}]
	if !ok {
		t.Fatal("Caller.run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Caller.run -> Helper.assist (same-package cross-file), got edges: %+v", edges)
	}
}

// TestResolve_ImportedCrossPackageCall proves a call to a class imported
// from a DIFFERENT package resolves into a real "calls" edge via the
// import -> ModuleKey mapping (Task 2's second must_have truth).
func TestResolve_ImportedCrossPackageCall(t *testing.T) {
	root := writeJavaFixture(t, []javaFixtureFile{
		{relPath: "com/example/Helper.java", src: "package com.example;\n\npublic class Helper {\n\tpublic static void assist() {}\n}\n"},
		{relPath: "com/other/Consumer.java", src: "package com.other;\n\nimport com.example.Helper;\n\npublic class Consumer {\n\tpublic void run() {\n\t\tHelper.assist();\n\t}\n}\n"},
	})

	nodes, edges := buildJavaGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "method", name: "assist", filePath: "com/example/Helper.java"}]
	if !ok {
		t.Fatal("Helper.assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "run", filePath: "com/other/Consumer.java"}]
	if !ok {
		t.Fatal("Consumer.run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Consumer.run -> Helper.assist (cross-package, imported), got edges: %+v", edges)
	}
}

// TestResolve_ImportedCrossPackageInheritance proves a class that extends
// a superclass imported from a different package resolves into a real
// "embeds" edge (the RefKindEmbeds shape RESEARCH Pattern 2 promotes to
// "implements" only at Wave 6's resolve-time pass, out of this plan's
// scope) — the "inheritance" leg of Task 2's cross-file resolution truth.
func TestResolve_ImportedCrossPackageInheritance(t *testing.T) {
	root := writeJavaFixture(t, []javaFixtureFile{
		{relPath: "com/example/Base.java", src: "package com.example;\n\npublic class Base {\n\tpublic void baseMethod() {}\n}\n"},
		{relPath: "com/other/Derived.java", src: "package com.other;\n\nimport com.example.Base;\n\npublic class Derived extends Base {}\n"},
	})

	nodes, edges := buildJavaGraph(t, root)

	base, ok := nodes[nodeKey{kind: "struct", name: "Base", filePath: "com/example/Base.java"}]
	if !ok {
		t.Fatal("Base node not found in committed graph")
	}
	derived, ok := nodes[nodeKey{kind: "struct", name: "Derived", filePath: "com/other/Derived.java"}]
	if !ok {
		t.Fatal("Derived node not found in committed graph")
	}

	if !hasEdge(edges, derived.Id, base.Id, "embeds") {
		t.Errorf("expected an embeds edge Derived -> Base (cross-package, imported inheritance), got edges: %+v", edges)
	}
}
