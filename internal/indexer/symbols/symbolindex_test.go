package symbols

import (
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

		gotID, ok := winnerOf(idx, "example.com/pkg", "Foo")
		if !ok {
			t.Fatalf("winnerOf(Foo) not found for order %v", order)
		}
		if gotID != wantID {
			t.Errorf("winnerOf(Foo) = %s, want lowest-Id winner %s (order %v)", gotID, wantID, order)
		}
		if idx.Collisions() != 1 {
			t.Errorf("Collisions = %d, want 1 (order %v)", idx.Collisions(), order)
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
	if idx.Collisions() != 0 {
		t.Errorf("Collisions = %d, want 0 for distinct names", idx.Collisions())
	}
	if _, ok := winnerOf(idx, "example.com/pkg", "Foo"); !ok {
		t.Error("expected Foo to resolve")
	}
	if _, ok := winnerOf(idx, "example.com/pkg", "Bar"); !ok {
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
	idx.AddResults(declsOf([]goextract.FileResult{a}))

	if idx.Collisions() != 0 {
		t.Errorf("Collisions = %d, want 0 for re-overlay of identical id", idx.Collisions())
	}
}

// newSymbolIndex returns a Builder that holds the declarations of
// results, in the order given.
func newSymbolIndex(results []goextract.FileResult) *Builder {
	b := NewBuilder()
	b.AddResults(declsOf(results))
	return b
}

// declsOf returns the declarations of each result, with no default
// visibility.
func declsOf(results []goextract.FileResult) []Declarations {
	decls := make([]Declarations, len(results))
	for i, r := range results {
		decls[i] = DeclarationsOf(r, "")
	}
	return decls
}

// winnerOf returns the id that name resolves to in module, as Pick
// chooses it from the candidates the index holds.
func winnerOf(idx *Builder, module, name string) (string, bool) {
	cands := idx.d.candidates[module][name]
	if len(cands) == 0 {
		return "", false
	}
	return Pick(cands), true
}
