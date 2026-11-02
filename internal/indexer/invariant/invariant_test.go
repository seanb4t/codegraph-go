package invariant

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// writeStore commits nodes, edges and meta to a fresh store and returns a
// snapshot of it.
func writeStore(t *testing.T, nodes []*schema.Node, edges []*schema.Edge, meta *schema.Meta) graphstore.Reader {
	t.Helper()
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for _, n := range nodes {
		if err := w.PutNode(n); err != nil {
			t.Fatalf("PutNode: %v", err)
		}
	}
	for _, e := range edges {
		if err := w.PutEdge(e, "f.java"); err != nil {
			t.Fatalf("PutEdge: %v", err)
		}
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func typeAndMethod(methodStart, methodEnd int32) []*schema.Node {
	return []*schema.Node{
		{Id: "struct:T", Kind: goextract.KindStruct, Name: "T", QualifiedName: "T", FilePath: "f.java", Language: "java", StartLine: 1, EndLine: 5},
		{Id: "method:m", Kind: goextract.KindMethod, Name: "m", QualifiedName: "T.m", FilePath: "f.java", Language: "java", StartLine: methodStart, EndLine: methodEnd},
	}
}

// TestCheckReportsEachViolation checks that Check names a dangling edge,
// a member outside its type and a wrong meta count, and passes a graph
// that has none.
func TestCheckReportsEachViolation(t *testing.T) {
	contains := &schema.Edge{Source: "struct:T", Target: "method:m", Kind: goextract.RefKindContains, Provenance: "ast"}
	t.Run("valid", func(t *testing.T) {
		r := writeStore(t, typeAndMethod(2, 4), []*schema.Edge{contains}, &schema.Meta{NodeCount: 2, EdgeCount: 1})
		counts, err := Check(r)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if counts.Nodes != 2 || counts.Edges != 1 || counts.Nesting != 1 {
			t.Errorf("counts = %+v, want 2 nodes, 1 edge, 1 nesting edge", counts)
		}
	})
	cases := []struct {
		name  string
		nodes []*schema.Node
		edges []*schema.Edge
		meta  *schema.Meta
		want  string
	}{
		{"dangling target", typeAndMethod(2, 4), []*schema.Edge{{Source: "struct:T", Target: "method:gone", Kind: goextract.RefKindCalls}}, &schema.Meta{NodeCount: 2, EdgeCount: 1}, "target is not a stored node"},
		{"dangling source", typeAndMethod(2, 4), []*schema.Edge{{Source: "func:gone", Target: "method:m", Kind: goextract.RefKindCalls}}, &schema.Meta{NodeCount: 2, EdgeCount: 1}, "source is not a stored node"},
		{"member outside type", typeAndMethod(8, 9), []*schema.Edge{contains}, &schema.Meta{NodeCount: 2, EdgeCount: 1}, "the member is outside the type"},
		{"node count", typeAndMethod(2, 4), []*schema.Edge{contains}, &schema.Meta{NodeCount: 3, EdgeCount: 1}, "Meta.NodeCount = 3"},
		{"edge count", typeAndMethod(2, 4), []*schema.Edge{contains}, &schema.Meta{NodeCount: 2, EdgeCount: 0}, "Meta.EdgeCount = 0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := writeStore(t, c.nodes, c.edges, c.meta)
			_, err := Check(r)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Check = %v, want an error containing %q", err, c.want)
			}
		})
	}
}

// TestEqualNamesTheDifference checks that Equal passes two stores with
// the same graph and names a node or an edge that only one holds.
func TestEqualNamesTheDifference(t *testing.T) {
	contains := &schema.Edge{Source: "struct:T", Target: "method:m", Kind: goextract.RefKindContains, Provenance: "ast"}
	meta := &schema.Meta{NodeCount: 2, EdgeCount: 1}
	a := writeStore(t, typeAndMethod(2, 4), []*schema.Edge{contains}, meta)
	same := writeStore(t, typeAndMethod(2, 4), []*schema.Edge{contains}, meta)
	if err := Equal(a, same); err != nil {
		t.Errorf("Equal(same) = %v, want nil", err)
	}
	moved := writeStore(t, typeAndMethod(3, 4), []*schema.Edge{contains}, meta)
	if err := Equal(a, moved); err == nil || !strings.Contains(err.Error(), "node method:m differs") {
		t.Errorf("Equal(moved) = %v, want the differing node", err)
	}
	fewer := writeStore(t, typeAndMethod(2, 4), nil, &schema.Meta{NodeCount: 2})
	if err := Equal(a, fewer); err == nil || !strings.Contains(err.Error(), "only in the first store") {
		t.Errorf("Equal(fewer) = %v, want the missing edge", err)
	}
}
