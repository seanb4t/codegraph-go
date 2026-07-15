package query

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestRankEdges asserts RankEdges has exactly the 9 §C.1 members, sourced
// from goextract's shared RefKind*/EdgeKind* constants (one definition,
// never re-declared literals).
func TestRankEdges(t *testing.T) {
	want := map[string]bool{
		goextract.RefKindCalls:        true,
		goextract.RefKindReferences:   true,
		goextract.EdgeKindExtends:     true,
		goextract.EdgeKindImplements:  true,
		goextract.EdgeKindOverrides:   true,
		goextract.RefKindInstantiates: true,
		goextract.RefKindReturns:      true,
		goextract.RefKindTypeOf:       true,
		goextract.RefKindImports:      true,
	}
	if len(RankEdges) != 9 {
		t.Fatalf("RankEdges has %d members, want 9: %v", len(RankEdges), RankEdges)
	}
	for k := range want {
		if !RankEdges[k] {
			t.Errorf("RankEdges missing expected member %q", k)
		}
	}
	for k := range RankEdges {
		if !want[k] {
			t.Errorf("RankEdges has unexpected member %q", k)
		}
	}
}

// TestRWRAdjacency_ExcludesNonRankEdgeKind asserts an edge whose Kind is
// not in RankEdges is excluded from the adjacency build (e.g. "contains").
func TestRWRAdjacency_ExcludesNonRankEdgeKind(t *testing.T) {
	nodeIDs := []string{"a", "b"}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: "contains"},
	}
	adj := buildRWRAdjacency(nodeIDs, edges)
	if len(adj[0]) != 0 || len(adj[1]) != 0 {
		t.Fatalf("expected no adjacency for non-RankEdges kind, got %v", adj)
	}
}

// TestRWRAdjacency_ExcludesSelfLoop asserts a self-loop (i==j) is excluded.
func TestRWRAdjacency_ExcludesSelfLoop(t *testing.T) {
	nodeIDs := []string{"a"}
	edges := []*schema.Edge{
		{Source: "a", Target: "a", Kind: goextract.RefKindCalls},
	}
	adj := buildRWRAdjacency(nodeIDs, edges)
	if len(adj[0]) != 0 {
		t.Fatalf("expected no self-loop adjacency, got %v", adj)
	}
}

// TestRWRAdjacency_SkipsDanglingEndpoint asserts an edge whose endpoint is
// absent from the candidate node set is skipped, not an error (WR-04).
func TestRWRAdjacency_SkipsDanglingEndpoint(t *testing.T) {
	nodeIDs := []string{"a"}
	edges := []*schema.Edge{
		{Source: "a", Target: "missing", Kind: goextract.RefKindCalls},
		{Source: "missing", Target: "a", Kind: goextract.RefKindCalls},
	}
	adj := buildRWRAdjacency(nodeIDs, edges)
	if len(adj[0]) != 0 {
		t.Fatalf("expected dangling edges to be skipped, got %v", adj)
	}
}

// TestRWRAdjacency_UndirectedBothDirections asserts a valid RankEdges edge
// pushes both directions onto the adjacency (undirected).
func TestRWRAdjacency_UndirectedBothDirections(t *testing.T) {
	nodeIDs := []string{"a", "b"}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
	}
	adj := buildRWRAdjacency(nodeIDs, edges)
	if len(adj[0]) != 1 || adj[0][0] != 1 {
		t.Errorf("expected a->b adjacency, got adj[0]=%v", adj[0])
	}
	if len(adj[1]) != 1 || adj[1][0] != 0 {
		t.Errorf("expected b->a adjacency (undirected), got adj[1]=%v", adj[1])
	}
}
