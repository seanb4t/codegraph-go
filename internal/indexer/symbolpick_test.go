package indexer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

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
		idx := symbols.NewBuilder()
		for _, n := range order {
			idx.Add("example.com/pick", n)
		}
		if got, _ := winnerOf(idx, "example.com/pick", name); got != b.Id {
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

// winnerOf returns the id that name resolves to in module, as
// symbols.Pick chooses it from the candidates that idx holds. It freezes
// idx.
func winnerOf(idx *symbols.Builder, module, name string) (string, bool) {
	cands := idx.Freeze().CandidatesIn(module, name)
	if len(cands) == 0 {
		return "", false
	}
	return symbols.Pick(cands), true
}
