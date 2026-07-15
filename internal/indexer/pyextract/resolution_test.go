// Package pyextract_test is an EXTERNAL test package (not `package
// pyextract`, the internal test package pyextract_test.go uses)
// deliberately: it drives the real internal/indexer.Run pipeline
// end-to-end, and internal/indexer itself imports pyextract
// (languages_python.go) — a same-package (internal) test file importing
// internal/indexer would create an import cycle. The external-test-package
// pattern is the standard Go mechanism for exercising a package's consumer
// from its own test tree without that cycle — mirrors javaextract/
// csharpextract's own resolution_test.go shape exactly.
package pyextract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// pyFixtureFile is one file this test writes into a temp repo root.
type pyFixtureFile struct {
	relPath, src string
}

// writePyFixture always includes a minimal pyproject.toml so the real
// pythonProjectDescriptor resolves (flat layout, repo root as the package
// root) — without it, ModuleKey would degrade to bare-relPath path-identity
// and none of these cross-module import statements (which name real dotted
// module paths) could ever match a file's own moduleKey.
func writePyFixture(t *testing.T, files []pyFixtureFile) string {
	t.Helper()
	root := t.TempDir()
	all := append([]pyFixtureFile{{relPath: "pyproject.toml", src: "[project]\nname = \"demo\"\n"}}, files...)
	for _, f := range all {
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

// buildPyGraph runs the real indexer.Run pipeline against root into a fresh
// temp store and returns every committed node (indexed by nodeKey) and
// edge, for direct inspection.
func buildPyGraph(t *testing.T, root string) (map[nodeKey]*schema.Node, []*schema.Edge) {
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

// TestResolve_CrossModuleImportedCall proves a `from pkg.helper import
// Helper` + `Helper.assist()` call resolves into a real "calls" edge via
// the dotted-module-path ModuleKey (Task 2's first must_have truth).
func TestResolve_CrossModuleImportedCall(t *testing.T) {
	root := writePyFixture(t, []pyFixtureFile{
		{relPath: "pkg/helper.py", src: "class Helper:\n    @staticmethod\n    def assist():\n        pass\n"},
		{relPath: "pkg/caller.py", src: "from pkg.helper import Helper\n\nclass Caller:\n    def run(self):\n        Helper.assist()\n"},
	})

	nodes, edges := buildPyGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "method", name: "assist", filePath: "pkg/helper.py"}]
	if !ok {
		t.Fatal("Helper.assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "run", filePath: "pkg/caller.py"}]
	if !ok {
		t.Fatal("Caller.run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Caller.run -> Helper.assist (cross-module, from-import), got edges: %+v", edges)
	}
}

// TestResolve_AliasedImportCall proves an aliased plain import (`import
// pkg.util as u`) + a call through the alias (`u.assist()`) resolves into a
// real "calls" edge — the OTHER import shape pyextract supports (Task 2's
// first must_have truth, exercised via the aliased-import path).
func TestResolve_AliasedImportCall(t *testing.T) {
	root := writePyFixture(t, []pyFixtureFile{
		{relPath: "pkg/util.py", src: "def assist():\n    pass\n"},
		{relPath: "pkg/caller.py", src: "import pkg.util as u\n\nclass Caller:\n    def run(self):\n        u.assist()\n"},
	})

	nodes, edges := buildPyGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "function", name: "assist", filePath: "pkg/util.py"}]
	if !ok {
		t.Fatal("pkg/util.assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "method", name: "run", filePath: "pkg/caller.py"}]
	if !ok {
		t.Fatal("Caller.run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge Caller.run -> pkg.util.assist (aliased import), got edges: %+v", edges)
	}
}

// TestResolve_CrossModuleInheritance proves a class that subclasses a base
// class imported from a different module (`from pkg.base import Base`)
// resolves into a real "extends" edge (the RefKindEmbeds shape RESEARCH
// Pattern 2 promotes to "implements" when the target is an interface, and
// — since 01-05's D-09 split — to "extends" when the target is a
// class/struct, as here) — the "inheritance" leg of Task 2's cross-file
// resolution truth.
func TestResolve_CrossModuleInheritance(t *testing.T) {
	root := writePyFixture(t, []pyFixtureFile{
		{relPath: "pkg/base.py", src: "class Base:\n    def base_method(self):\n        pass\n"},
		{relPath: "pkg2/derived.py", src: "from pkg.base import Base\n\nclass Derived(Base):\n    pass\n"},
	})

	nodes, edges := buildPyGraph(t, root)

	base, ok := nodes[nodeKey{kind: "struct", name: "Base", filePath: "pkg/base.py"}]
	if !ok {
		t.Fatal("Base node not found in committed graph")
	}
	derived, ok := nodes[nodeKey{kind: "struct", name: "Derived", filePath: "pkg2/derived.py"}]
	if !ok {
		t.Fatal("Derived node not found in committed graph")
	}

	if !hasEdge(edges, derived.Id, base.Id, "extends") {
		t.Errorf("expected an extends edge Derived -> Base (cross-module, imported inheritance), got edges: %+v", edges)
	}
}
