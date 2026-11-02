package indexer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// fooDeclResult builds a single-file FileResult declaring one KindFunction
// symbol named "Foo" in the shared module "example.com/pkg" — used to
// simulate two DIFFERENT files in the same module both declaring a
// same-named symbol (the WR-01 collision shape).
func fooDeclResult(relPath string) goextract.FileResult {
	id := nodeid.DeclID(goextract.KindFunction, "Foo", relPath, "()")
	return goextract.FileResult{
		ImportPath: "example.com/pkg",
		RelPath:    relPath,
		Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
			Id:            id,
			Kind:          goextract.KindFunction,
			Name:          "Foo",
			QualifiedName: "Foo",
			FilePath:      relPath,
		}}},
	}
}

// TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent proves a
// same-(moduleKey, name) collision between two files' declarations resolves
// to a stable, order-independent target — the lowest node Id wins,
// regardless of which file was processed first — and the collision is
// counted, never silently dropped (WR-01, D-06a).
func TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent(t *testing.T) {
	a := fooDeclResult("pkg/a.go")
	b := fooDeclResult("pkg/b.go")

	wantID := a.Nodes[0].Node.Id
	if b.Nodes[0].Node.Id < wantID {
		wantID = b.Nodes[0].Node.Id
	}

	for _, order := range [][]goextract.FileResult{
		{a, b},
		{b, a},
	} {
		idx := newSymbolIndex(order)

		gotID, ok := idx.resolveUnqualified("example.com/pkg", "Foo")
		if !ok {
			t.Fatalf("resolveUnqualified(Foo) not found for order %v", order)
		}
		if gotID != wantID {
			t.Errorf("resolveUnqualified(Foo) = %s, want lowest-Id winner %s (order %v)", gotID, wantID, order)
		}
		if idx.Collisions != 1 {
			t.Errorf("Collisions = %d, want 1 (order %v)", idx.Collisions, order)
		}
	}
}

// TestSymbolIndex_NoCollisionForDistinctNames proves declaring two
// distinctly-named symbols in the same module never increments Collisions
// — only a genuine same-(moduleKey, name) clash does.
func TestSymbolIndex_NoCollisionForDistinctNames(t *testing.T) {
	a := fooDeclResult("pkg/a.go")
	barID := nodeid.DeclID(goextract.KindFunction, "Bar", "pkg/b.go", "()")
	b := goextract.FileResult{
		ImportPath: "example.com/pkg",
		RelPath:    "pkg/b.go",
		Nodes: []goextract.ExtractedNode{{Node: &schema.Node{
			Id: barID, Kind: goextract.KindFunction, Name: "Bar",
			QualifiedName: "Bar", FilePath: "pkg/b.go",
		}}},
	}

	idx := newSymbolIndex([]goextract.FileResult{a, b})
	if idx.Collisions != 0 {
		t.Errorf("Collisions = %d, want 0 for distinct names", idx.Collisions)
	}
	if _, ok := idx.resolveUnqualified("example.com/pkg", "Foo"); !ok {
		t.Error("expected Foo to resolve")
	}
	if _, ok := idx.resolveUnqualified("example.com/pkg", "Bar"); !ok {
		t.Error("expected Bar to resolve")
	}
}

// TestSymbolIndex_ReOverlaySameIDNotACollision proves overlaying the exact
// same (moduleKey, name, id) twice — e.g. Sync re-overlaying an unchanged
// file's own symbols on top of a store-seeded index that already contains
// them — does not increment Collisions.
func TestSymbolIndex_ReOverlaySameIDNotACollision(t *testing.T) {
	a := fooDeclResult("pkg/a.go")
	idx := newSymbolIndex([]goextract.FileResult{a})
	idx.overlay([]goextract.FileResult{a})

	if idx.Collisions != 0 {
		t.Errorf("Collisions = %d, want 0 for re-overlay of identical id", idx.Collisions)
	}
}

// pickTrio returns a function name X whose three candidate ids sort A2,
// B, A1: A1 and A2 are the first and second copy of X in a.go, and B is
// X in b.go. With these hashes, a pairwise rule depends on the order in
// which the candidates arrive.
func pickTrio(t *testing.T) (name string, a1, a2, b *schema.Node) {
	t.Helper()
	node := func(id, name, path string) *schema.Node {
		return &schema.Node{Id: id, Kind: goextract.KindFunction, Name: name, QualifiedName: name, FilePath: path}
	}
	for i := 0; i < 1000; i++ {
		name = fmt.Sprintf("Pick%d", i)
		a1 = node(nodeid.DeclID(goextract.KindFunction, name, "a.go", "()"), name, "a.go")
		a2 = node(nodeid.OrdinalID(goextract.KindFunction, name, "a.go", "()", 2), name, "a.go")
		b = node(nodeid.DeclID(goextract.KindFunction, name, "b.go", "()"), name, "b.go")
		if a2.Id < b.Id && b.Id < a1.Id {
			return name, a1, a2, b
		}
	}
	t.Fatal("found no name whose ids sort A2, B, A1")
	return
}

// TestSymbolPickIgnoresInsertionOrder checks that a name resolves to the
// lower id of the two group winners, A1 and B, for all six insertion
// orders of A1, its copy A2, and B, and through a full index and the
// store-seeded index of an incremental Sync. A2 never wins.
func TestSymbolPickIgnoresInsertionOrder(t *testing.T) {
	name, a1, a2, b := pickTrio(t)
	orders := [][]*schema.Node{
		{a1, a2, b}, {a1, b, a2}, {a2, a1, b}, {a2, b, a1}, {b, a1, a2}, {b, a2, a1},
	}
	label := map[string]string{a1.Id: "A1", a2.Id: "A2", b.Id: "B"}
	for _, order := range orders {
		idx := newEmptySymbolIndex()
		for _, n := range order {
			idx.add("example.com/pick", n)
		}
		if got, _ := idx.resolveUnqualified("example.com/pick", name); got != b.Id {
			t.Errorf("insertion order %s %s %s: got %s, want B", label[order[0].Id], label[order[1].Id], label[order[2].Id], label[got])
		}
	}

	root := writeFixture(t, map[string]string{
		"a.go": fmt.Sprintf("package pick\n\nfunc %s() {}\n\nfunc %s() {}\n", name, name),
		"b.go": fmt.Sprintf("package pick\n\nfunc %s() {}\n", name),
		"c.go": fmt.Sprintf("package pick\n\nfunc Caller() { %s() }\n", name),
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	requireCallTarget := func(stage string) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		callers := declaredNodes(t, r, "c.go", goextract.KindFunction, "Caller")
		if len(callers) != 1 {
			t.Fatalf("%s: got %d Caller nodes, want 1", stage, len(callers))
		}
		for _, n := range []*schema.Node{a1, a2, b} {
			want := n == b
			if got := hasEdge(t, r, callers[0].Id, goextract.RefKindCalls, n.Id); got != want {
				t.Errorf("%s: Caller calls %s in %s (id %s) = %v, want %v", stage, n.Name, n.FilePath, n.Id, got, want)
			}
		}
	}
	requireCallTarget("index")

	writeFile(t, filepath.Join(root, "c.go"), fmt.Sprintf("package pick\n\n// Caller calls %s.\nfunc Caller() { %s() }\n", name, name))
	stats, err := Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if stats.FilesReparsed != 1 {
		t.Fatalf("Sync reparsed %d files, want only c.go", stats.FilesReparsed)
	}
	requireCallTarget("sync")
}
