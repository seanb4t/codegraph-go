package query

import (
	"os"
	"sort"
	"strings"
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

	// mutation: per-call seed counter
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

// TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture is D-04's RED
// control: it proves the fixed seed is LOAD-BEARING by finding a seed
// offset that flips the modularity-tying tie fixture to the other of
// its two symmetric partitions, then shows the well-separated structured
// fixture is unaffected by that same offset — a well-separated graph is
// seed-invariant, while a modularity tie is not, which is exactly why
// the fixed seed matters.
func TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture(t *testing.T) {
	tieNodes, tieEdges := communityTieFixture()
	baseline := AssignCommunities(tieNodes, tieEdges)

	var flipOffset int
	for k := 1; k <= 32; k++ {
		perturbed := assignCommunitiesWith(tieNodes, tieEdges, communityOptions{
			seed1: communitySeed1 + uint64(k),
			seed2: communitySeed2,
		})
		if !communityMapsEqual(baseline, perturbed) {
			flipOffset = k
			t.Logf("seed offset %d flipped the tie fixture", k)
			break
		}
	}
	if flipOffset == 0 {
		t.Fatalf("no seed offset in [1, 32] flipped the tie fixture — the fixed seed may not be load-bearing: baseline = %v", baseline)
	}

	structuredNodes, structuredEdges := communityStructuredFixture()
	structuredBaseline := AssignCommunities(structuredNodes, structuredEdges)
	structuredPerturbed := assignCommunitiesWith(structuredNodes, structuredEdges, communityOptions{
		seed1: communitySeed1 + uint64(flipOffset),
		seed2: communitySeed2,
	})
	if !communityMapsEqual(structuredBaseline, structuredPerturbed) {
		t.Fatalf("structured fixture changed under the flipping offset %d: baseline = %v, perturbed = %v — a well-separated graph should be seed-invariant", flipOffset, structuredBaseline, structuredPerturbed)
	}
	t.Logf("structured fixture invariant under the flipping offset")
}

// TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder is
// D-02c's positive control: canonical relabeling, not gonum itself, is
// what neutralises node-insertion order. With the relabel step ON,
// reversing insertion order produces the SAME result as the default
// (forward) order. With the relabel step turned OFF (test-only seam),
// reversing insertion order produces a DIFFERENT result — proving the
// relabel step is load-bearing, not cosmetic.
func TestAssignCommunitiesCanonicalRelabelNeutralisesInsertionOrder(t *testing.T) {
	nodes, edges := communityStructuredFixture()

	forward := AssignCommunities(nodes, edges)
	reversedRelabeled := assignCommunitiesWith(nodes, edges, communityOptions{
		seed1:            communitySeed1,
		seed2:            communitySeed2,
		reverseInsertion: true,
	})
	if !communityMapsEqual(forward, reversedRelabeled) {
		t.Fatalf("reverseInsertion with canonical relabel ON = %v, want it to equal the default result %v — canonical relabeling should neutralise insertion order", reversedRelabeled, forward)
	}

	forwardRaw := assignCommunitiesWith(nodes, edges, communityOptions{
		seed1:                communitySeed1,
		seed2:                communitySeed2,
		skipCanonicalRelabel: true,
	})
	reversedRaw := assignCommunitiesWith(nodes, edges, communityOptions{
		seed1:                communitySeed1,
		seed2:                communitySeed2,
		reverseInsertion:     true,
		skipCanonicalRelabel: true,
	})
	if communityMapsEqual(forwardRaw, reversedRaw) {
		t.Fatalf("with canonical relabel OFF, forward and reversed insertion order produced the SAME labels %v — this positive control requires them to differ, otherwise the relabel step cannot be shown load-bearing", forwardRaw)
	}
}

// TestAssignCommunitiesDegenerate pins the degenerate and encoding
// boundary cases: zero nodes, a single node, zero edges over several
// nodes, and byte-wise (never case-folded) path ordering.
func TestAssignCommunitiesDegenerate(t *testing.T) {
	t.Run("zero nodes", func(t *testing.T) {
		got := AssignCommunities(nil, nil)
		if got == nil {
			t.Fatal("AssignCommunities(nil, nil) returned a nil map, want a non-nil empty map")
		}
		if len(got) != 0 {
			t.Fatalf("AssignCommunities(nil, nil) = %v, want empty", got)
		}

		e := New(&traverseFakeReader{nodes: map[string]*schema.Node{}, edges: nil})
		fg, err := e.FileGraph()
		if err != nil {
			t.Fatalf("FileGraph over an empty reader: unexpected error: %v", err)
		}
		if fg.CommunityCount != 0 {
			t.Fatalf("FileGraph over an empty reader: CommunityCount = %d, want 0", fg.CommunityCount)
		}
	})

	t.Run("one node", func(t *testing.T) {
		got := AssignCommunities([]FileGraphNode{{Path: "only.go"}}, nil)
		want := map[string]int{"only.go": 1}
		if !communityMapsEqual(got, want) {
			t.Fatalf("AssignCommunities(one node) = %v, want %v", got, want)
		}
	})

	t.Run("zero edges", func(t *testing.T) {
		nodes := []FileGraphNode{{Path: "a.go"}, {Path: "b.go"}, {Path: "c.go"}, {Path: "d.go"}}
		got := AssignCommunities(nodes, nil)
		want := map[string]int{"a.go": 1, "b.go": 2, "c.go": 3, "d.go": 4}
		if !communityMapsEqual(got, want) {
			t.Fatalf("AssignCommunities(zero edges) = %v, want %v (ascending path order)", got, want)
		}
	})

	t.Run("byte order encoding", func(t *testing.T) {
		nodes := []FileGraphNode{{Path: "B.go"}, {Path: "a.go"}}
		got := AssignCommunities(nodes, nil)
		want := map[string]int{"B.go": 1, "a.go": 2}
		if !communityMapsEqual(got, want) {
			t.Fatalf("AssignCommunities(B.go, a.go) = %v, want %v — byte-wise order (uppercase sorts first), never case-folded", got, want)
		}
	})

	t.Run("every id dense and >= 1", func(t *testing.T) {
		nodes, edges := communityStructuredFixture()
		got := AssignCommunities(nodes, edges)
		maxID := 0
		distinct := map[int]struct{}{}
		for _, id := range got {
			if id < 1 {
				t.Fatalf("AssignCommunities: id %d < 1", id)
			}
			distinct[id] = struct{}{}
			if id > maxID {
				maxID = id
			}
		}
		if maxID != len(distinct) {
			t.Fatalf("AssignCommunities: max id %d != distinct count %d, want dense 1..N", maxID, len(distinct))
		}
	})
}

// TestUndirectedPairWeightsSumBothDirections proves undirectedPairWeights
// SUMS both directions of a file pair's TotalCount into one undirected
// weight (measured this session: simple.WeightedUndirectedGraph.
// SetWeightedEdge REPLACES rather than accumulates), while skipping
// self-edges, non-positive counts, and edges naming an id absent from
// idByPath.
func TestUndirectedPairWeightsSumBothDirections(t *testing.T) {
	idByPath := map[string]int64{"a": 0, "b": 1, "c": 2}
	edges := []FileGraphEdge{
		{SourceFile: "a", TargetFile: "b", TotalCount: 3},
		{SourceFile: "b", TargetFile: "a", TotalCount: 2},
		{SourceFile: "a", TargetFile: "c", TotalCount: 1},
		{SourceFile: "c", TargetFile: "c", TotalCount: 9},   // self edge, skipped
		{SourceFile: "a", TargetFile: "zzz", TotalCount: 4}, // unknown endpoint, skipped
		{SourceFile: "b", TargetFile: "c", TotalCount: 0},   // zero, skipped
	}
	got := undirectedPairWeights(idByPath, edges)
	want := map[[2]int64]float64{
		{0, 1}: 5,
		{0, 2}: 1,
	}
	if len(got) != len(want) {
		t.Fatalf("undirectedPairWeights = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("undirectedPairWeights[%v] = %v, want %v (full map: %v)", k, got[k], v, got)
		}
	}
}

// forbiddenCommunitySourceSubstrings are the schema Node field-50
// accessor shapes and persistence/caching calls the assumption-delta
// invariant forbids on the fresh-compute path (D-15). Finding any of
// these in community.go or traverse.go means the fresh-compute-only
// assumption has been silently replaced by index-time persistence —
// exactly the D-07 fallback, which is sanctioned ONLY as 11-02 Task 3's
// deliberate, documented inversion on a FAIL verdict.
//
// mutation: read Node field 50 instead of computing
var forbiddenCommunitySourceSubstrings = []string{
	"GetCommunityId(",
	".CommunityId",
	"sync.Once",
	"NewWriter(",
	"PutNode(",
}

// stripCommentLines removes every line-comment line (a line whose
// trimmed content starts with "//") from src, so a forbidden-substring
// scan checks actual code, never prose that legitimately DISCUSSES a
// forbidden shape (e.g. a doc comment explaining what is NOT present).
// This is a line-level strip, not a full Go tokenizer — sufficient here
// because every forbidden substring this file scans for is checked
// against whole lines of source, not embedded in a multi-line string
// literal.
func stripCommentLines(src string) string {
	lines := strings.Split(src, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// TestFileGraphCommunitySourceIsFreshComputeOnly is the assumption-delta
// invariant test (orchestrator decision, 11-01-PLAN.md): community
// assignment's sole primary is fresh computation inside FileGraph()
// (D-15). It asserts community.go and traverse.go contain none of
// forbiddenCommunitySourceSubstrings, with a positive control proving
// the scan actually inspects the files it claims to (AssignCommunities(
// result.Nodes, result.Edges) present exactly once in traverse.go,
// community.Modularize( present exactly once in community.go).
// 11-02 Task 3 is the ONLY sanctioned inversion of this test, on a FAIL
// verdict.
func TestFileGraphCommunitySourceIsFreshComputeOnly(t *testing.T) {
	communitySrc, err := os.ReadFile("community.go")
	if err != nil {
		t.Fatalf("read community.go: %v", err)
	}
	traverseSrc, err := os.ReadFile("traverse.go")
	if err != nil {
		t.Fatalf("read traverse.go: %v", err)
	}
	if len(communitySrc) == 0 {
		t.Fatal("community.go is empty — this guard would pass vacuously")
	}
	if len(traverseSrc) == 0 {
		t.Fatal("traverse.go is empty — this guard would pass vacuously")
	}

	for _, name := range []struct {
		file string
		text string
	}{
		{"community.go", stripCommentLines(string(communitySrc))},
		{"traverse.go", stripCommentLines(string(traverseSrc))},
	} {
		for _, forbidden := range forbiddenCommunitySourceSubstrings {
			if strings.Contains(name.text, forbidden) {
				t.Fatalf("%s contains %q outside a comment — D-15 forbids the fresh-compute path from reading or writing the persisted Node field-50 community_id, or from caching/persisting an assignment", name.file, forbidden)
			}
		}
	}

	callSiteCount := strings.Count(string(traverseSrc), "AssignCommunities(result.Nodes, result.Edges)")
	if callSiteCount != 1 {
		t.Fatalf("traverse.go: AssignCommunities(result.Nodes, result.Edges) appears %d times, want exactly 1 (positive control)", callSiteCount)
	}
	modularizeCount := strings.Count(string(communitySrc), "community.Modularize(")
	if modularizeCount != 1 {
		t.Fatalf("community.go: community.Modularize( appears %d times, want exactly 1 (positive control)", modularizeCount)
	}

	t.Logf("inspected %d bytes (community.go) + %d bytes (traverse.go), 0 forbidden substrings, 2 positive controls", len(communitySrc), len(traverseSrc))
}
