package query

import (
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// communityStructuredFixture returns two dense triangles (a/*, b/*)
// joined by a thin bridge, an isolated singleton (lone.go), and a
// separate 4-cycle (s/*) — a well-separated graph where Louvain's
// modularity optimization has one clear answer, in contrast to
// communityTieFixture's engineered ambiguity (D-04, GRF-08).
func communityStructuredFixture() ([]FileGraphNode, []FileGraphEdge) {
	paths := []string{
		"a/x.go", "a/y.go", "a/z.go",
		"b/p.go", "b/q.go", "b/r.go",
		"lone.go",
		"s/1.go", "s/2.go", "s/3.go", "s/4.go",
	}
	nodes := make([]FileGraphNode, len(paths))
	for i, p := range paths {
		nodes[i] = FileGraphNode{Path: p}
	}
	edges := []FileGraphEdge{
		{SourceFile: "a/x.go", TargetFile: "a/y.go", TotalCount: 3},
		{SourceFile: "a/y.go", TargetFile: "a/z.go", TotalCount: 3},
		{SourceFile: "a/x.go", TargetFile: "a/z.go", TotalCount: 3},
		{SourceFile: "b/p.go", TargetFile: "b/q.go", TotalCount: 3},
		{SourceFile: "b/q.go", TargetFile: "b/r.go", TotalCount: 3},
		{SourceFile: "b/p.go", TargetFile: "b/r.go", TotalCount: 3},
		{SourceFile: "a/z.go", TargetFile: "b/p.go", TotalCount: 1},
		{SourceFile: "s/1.go", TargetFile: "s/2.go", TotalCount: 1},
		{SourceFile: "s/2.go", TargetFile: "s/3.go", TotalCount: 1},
		{SourceFile: "s/3.go", TargetFile: "s/4.go", TotalCount: 1},
		{SourceFile: "s/4.go", TargetFile: "s/1.go", TotalCount: 1},
	}
	return nodes, edges
}

// communityTieFixture returns a standalone 4-cycle (a-b-c-d-a, unit
// weight) with two perfectly symmetric 2+2 modularity-tying partitions
// — {a,b}{c,d} and {a,d}{b,c} — so which partition wins is decided
// entirely by tie-break order (node id, hence sorted path), not by
// structure. This is exactly what Task 3's RED-control seam perturbs.
func communityTieFixture() ([]FileGraphNode, []FileGraphEdge) {
	paths := []string{"a.go", "b.go", "c.go", "d.go"}
	nodes := make([]FileGraphNode, len(paths))
	for i, p := range paths {
		nodes[i] = FileGraphNode{Path: p}
	}
	edges := []FileGraphEdge{
		{SourceFile: "a.go", TargetFile: "b.go", TotalCount: 1},
		{SourceFile: "b.go", TargetFile: "c.go", TotalCount: 1},
		{SourceFile: "c.go", TargetFile: "d.go", TotalCount: 1},
		{SourceFile: "d.go", TargetFile: "a.go", TotalCount: 1},
	}
	return nodes, edges
}

// communityMapsEqual reports whether two community assignment maps are
// element-for-element equal.
func communityMapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// TestAssignCommunitiesDeterministic is GRF-08/D-04's positive proof: the
// clustering is run at least 3 times on identical input and every run
// must agree element-for-element, on both a well-separated fixture (an
// exact expected assignment) and a modularity-tie fixture (run-to-run
// equality plus the canonical-relabel invariant, without asserting which
// of the two symmetric partitions wins — that is what Task 3's RED
// control demonstrates is seed-dependent).
func TestAssignCommunitiesDeterministic(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		nodes, edges := communityStructuredFixture()
		want := map[string]int{
			"a/x.go": 1, "a/y.go": 1, "a/z.go": 1,
			"b/p.go": 2, "b/q.go": 2, "b/r.go": 2,
			"lone.go": 3,
			"s/1.go": 4, "s/2.go": 4, "s/3.go": 4, "s/4.go": 4,
		}
		const runs = 5
		var first map[string]int
		for i := 0; i < runs; i++ {
			got := AssignCommunities(nodes, edges)
			if i == 0 {
				first = got
			} else if !communityMapsEqual(first, got) {
				t.Fatalf("run %d: AssignCommunities = %v, want run 0's result %v", i, got, first)
			}
			if !communityMapsEqual(got, want) {
				t.Fatalf("run %d: AssignCommunities = %v, want %v", i, got, want)
			}
		}
		t.Logf("compared %d runs", runs)
	})

	t.Run("tie", func(t *testing.T) {
		nodes, edges := communityTieFixture()
		const runs = 5
		var first map[string]int
		for i := 0; i < runs; i++ {
			got := AssignCommunities(nodes, edges)
			if i == 0 {
				first = got
			} else if !communityMapsEqual(first, got) {
				t.Fatalf("run %d: AssignCommunities = %v, want run 0's result %v", i, got, first)
			}
		}
		sizes := map[int]int{}
		for _, id := range first {
			sizes[id]++
		}
		if len(sizes) != 2 {
			t.Fatalf("tie fixture: got %d distinct communities, want 2: %v", len(sizes), first)
		}
		for id, n := range sizes {
			if n != 2 {
				t.Fatalf("tie fixture: community %d has %d members, want 2 each: %v", id, n, sizes)
			}
		}
		if first["a.go"] != 1 {
			t.Fatalf("tie fixture: a.go = %d, want 1 (canonical relabel puts the lexically-smallest member's community first)", first["a.go"])
		}
		t.Logf("compared %d runs", runs)
	})
}

// TestFileGraphPopulatesCommunityFields is GRF-06/D-15's integration
// proof: a full Engine.FileGraph() call over a fake reader populates
// CommunityID (1-based, never 0 once computed) on every node and
// CommunityCount on the result, and a SECOND call over the same fixture
// returns identical values — proving the community pass is computed
// fresh per call, with no drift and no cache (D-15).
func TestFileGraphPopulatesCommunityFields(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
		"c": {Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "b", Target: "a", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "c", Kind: goextract.RefKindCalls},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}

	paths := make([]string, len(got.Nodes))
	for i, n := range got.Nodes {
		paths[i] = n.Path
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatalf("FileGraph: nodes not sorted ascending by path: %v", paths)
	}

	distinct := map[int]struct{}{}
	for _, n := range got.Nodes {
		if n.CommunityID < 1 {
			t.Fatalf("FileGraph: node %q has CommunityID %d, want >= 1", n.Path, n.CommunityID)
		}
		distinct[n.CommunityID] = struct{}{}
	}
	if got.CommunityCount != len(distinct) {
		t.Fatalf("FileGraph: CommunityCount = %d, want %d (distinct ids seen)", got.CommunityCount, len(distinct))
	}
	if got.CommunityCount < 1 {
		t.Fatalf("FileGraph: CommunityCount = %d, want >= 1", got.CommunityCount)
	}

	second, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph (second call): unexpected error: %v", err)
	}
	if second.CommunityCount != got.CommunityCount {
		t.Fatalf("FileGraph: second call CommunityCount = %d, want %d — fresh compute must not drift", second.CommunityCount, got.CommunityCount)
	}
	for i := range got.Nodes {
		if second.Nodes[i].CommunityID != got.Nodes[i].CommunityID {
			t.Fatalf("FileGraph: node %q CommunityID drifted between calls: %d vs %d", got.Nodes[i].Path, second.Nodes[i].CommunityID, got.Nodes[i].CommunityID)
		}
	}
}
