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

// starSubgraph builds a small deterministic fixture: seed node "a" at the
// center of a star (a-b, a-c), plus a distant node "z" reachable only
// through a long chain (c-d-e-z), and a dangling node "iso" (degree 0).
func starSubgraph() ([]string, []*schema.Edge) {
	nodeIDs := []string{"a", "b", "c", "d", "e", "z", "iso"}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "c", Kind: goextract.RefKindCalls},
		{Source: "c", Target: "d", Kind: goextract.RefKindCalls},
		{Source: "d", Target: "e", Kind: goextract.RefKindCalls},
		{Source: "e", Target: "z", Kind: goextract.RefKindCalls},
	}
	return nodeIDs, edges
}

// TestComputeGraphRelevance_SeedOutranksDistant asserts a seed node ends
// with higher mass than a node many hops away (RESEARCH §3's core claim:
// structurally-connected symbols must outrank distant/lexical matches).
func TestComputeGraphRelevance_SeedOutranksDistant(t *testing.T) {
	nodeIDs, edges := starSubgraph()
	seeds := map[string]bool{"a": true}
	scores := computeGraphRelevance(nodeIDs, edges, seeds)
	if scores["a"] <= scores["z"] {
		t.Fatalf("expected seed a's score (%v) > distant z's score (%v)", scores["a"], scores["z"])
	}
}

// TestComputeGraphRelevance_NoSeedUniformRestart asserts that when no seed
// lands in the candidate set, the restart vector falls back to
// uniform-over-all and every node ends up with non-zero mass.
func TestComputeGraphRelevance_NoSeedUniformRestart(t *testing.T) {
	nodeIDs, edges := starSubgraph()
	seeds := map[string]bool{"not-in-graph": true}
	scores := computeGraphRelevance(nodeIDs, edges, seeds)
	for _, id := range nodeIDs {
		if scores[id] <= 0 {
			t.Errorf("expected non-zero mass for %q under uniform restart fallback, got %v", id, scores[id])
		}
	}
}

// TestComputeGraphRelevance_DanglingNodeRetainsMass asserts a degree-0
// node keeps its own mass across all 25 iterations rather than losing it
// (RESEARCH §3: "dangling: keep its mass").
func TestComputeGraphRelevance_DanglingNodeRetainsMass(t *testing.T) {
	nodeIDs, edges := starSubgraph()
	seeds := map[string]bool{"iso": true}
	scores := computeGraphRelevance(nodeIDs, edges, seeds)
	// iso is seeded and has no edges, so its restart mass never leaks
	// away to any neighbor and never receives redistribution from
	// elsewhere; its final score must still be positive.
	if scores["iso"] <= 0 {
		t.Fatalf("expected dangling seeded node iso to retain positive mass, got %v", scores["iso"])
	}
}

// TestComputeGraphRelevance_EmptyNodeIDs asserts n==0 returns an empty map
// without panicking.
func TestComputeGraphRelevance_EmptyNodeIDs(t *testing.T) {
	scores := computeGraphRelevance(nil, nil, nil)
	if len(scores) != 0 {
		t.Fatalf("expected empty map for empty nodeIDs, got %v", scores)
	}
}

// TestRWRDeterminism_RepeatedRunsIdentical asserts two (or more, via
// -count=5) runs over the identical subgraph produce bit-identical
// (post-rounding) score maps — the golden-corpus determinism contract
// (D-04).
func TestRWRDeterminism_RepeatedRunsIdentical(t *testing.T) {
	nodeIDs, edges := starSubgraph()
	seeds := map[string]bool{"a": true, "iso": true}

	first := computeGraphRelevance(nodeIDs, edges, seeds)
	for run := 0; run < 10; run++ {
		got := computeGraphRelevance(nodeIDs, edges, seeds)
		if len(got) != len(first) {
			t.Fatalf("run %d: score map size %d != first run's %d", run, len(got), len(first))
		}
		for id, want := range first {
			if got[id] != want {
				t.Fatalf("run %d: score for %q = %v, want %v (first run)", run, id, got[id], want)
			}
		}
	}
}

// TestSortRWRScores_TieBreakScoreDescIdAsc asserts equal scores resolve
// score-desc-then-Id-asc (D-04, the codebase's lowest-Id convention).
func TestSortRWRScores_TieBreakScoreDescIdAsc(t *testing.T) {
	scores := map[string]float64{
		"z": 0.5,
		"a": 0.5,
		"m": 0.9,
	}
	got := sortRWRScores(scores)
	want := []string{"m", "a", "z"}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("position %d: got %q, want %q (full: %v)", i, got[i].ID, id, got)
		}
	}
}
