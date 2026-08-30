package query

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestFileGraphCyclesTwoNodeCycle pins the base case: adjacency A->B and
// B->A yields both A and B with the same non-zero cycle id, and exactly
// one component.
func TestFileGraphCyclesTwoNodeCycle(t *testing.T) {
	adj := map[string][]string{
		"a.go": {"b.go"},
		"b.go": {"a.go"},
	}
	got := stronglyConnectedCycles(adj)

	if len(got) != 2 {
		t.Fatalf("stronglyConnectedCycles: got %d entries, want 2 (a.go and b.go)", len(got))
	}
	idA, ok := got["a.go"]
	if !ok || idA == 0 {
		t.Fatalf("stronglyConnectedCycles: a.go missing or has zero cycle id: %v", got)
	}
	idB, ok := got["b.go"]
	if !ok || idB == 0 {
		t.Fatalf("stronglyConnectedCycles: b.go missing or has zero cycle id: %v", got)
	}
	if idA != idB {
		t.Fatalf("stronglyConnectedCycles: a.go id %d != b.go id %d, want the same component", idA, idB)
	}

	distinct := map[int]bool{}
	for _, id := range got {
		distinct[id] = true
	}
	if len(distinct) != 1 {
		t.Fatalf("stronglyConnectedCycles: got %d distinct components, want exactly 1", len(distinct))
	}
}

// TestFileGraphCyclesSingleNodeIsNotACycle is the negative control paired
// with TestFileGraphCyclesTwoNodeCycle: a graph of three nodes and no
// back edge yields an empty result. Together the two tests prove the
// detector DISTINGUISHES a cycle from no cycle, rather than always
// returning nothing.
func TestFileGraphCyclesSingleNodeIsNotACycle(t *testing.T) {
	adj := map[string][]string{
		"a.go": {"b.go"},
		"b.go": {"c.go"},
	}
	got := stronglyConnectedCycles(adj)
	if len(got) != 0 {
		t.Fatalf("stronglyConnectedCycles: got %v, want empty result for a DAG with no back edge", got)
	}
}

// TestFileGraphCyclesTwoDisjointComponents: two independent cycles yield
// two distinct non-zero cycle ids, and no node appears in both.
func TestFileGraphCyclesTwoDisjointComponents(t *testing.T) {
	adj := map[string][]string{
		"a.go": {"b.go"},
		"b.go": {"a.go"},
		"c.go": {"d.go"},
		"d.go": {"c.go"},
	}
	got := stronglyConnectedCycles(adj)
	if len(got) != 4 {
		t.Fatalf("stronglyConnectedCycles: got %d entries, want 4", len(got))
	}
	idA, idB, idC, idD := got["a.go"], got["b.go"], got["c.go"], got["d.go"]
	if idA == 0 || idB == 0 || idC == 0 || idD == 0 {
		t.Fatalf("stronglyConnectedCycles: expected all four non-zero: a=%d b=%d c=%d d=%d", idA, idB, idC, idD)
	}
	if idA != idB {
		t.Fatalf("stronglyConnectedCycles: a.go and b.go should share a component: %d vs %d", idA, idB)
	}
	if idC != idD {
		t.Fatalf("stronglyConnectedCycles: c.go and d.go should share a component: %d vs %d", idC, idD)
	}
	if idA == idC {
		t.Fatalf("stronglyConnectedCycles: the two disjoint cycles were assigned the SAME component id %d, want distinct", idA)
	}
}

// TestFileGraphCyclesDeterministicIds: two runs over the same adjacency
// assign the same cycle id to the same node, so the wire output does not
// churn between requests.
func TestFileGraphCyclesDeterministicIds(t *testing.T) {
	adj := map[string][]string{
		"a.go": {"b.go"},
		"b.go": {"a.go"},
		"c.go": {"d.go"},
		"d.go": {"c.go"},
	}
	first := stronglyConnectedCycles(adj)
	second := stronglyConnectedCycles(adj)

	if len(first) != len(second) {
		t.Fatalf("stronglyConnectedCycles: entry count differs between runs: %d vs %d", len(first), len(second))
	}
	for k, v := range first {
		if second[k] != v {
			t.Fatalf("stronglyConnectedCycles: %q got id %d on first run, %d on second run", k, v, second[k])
		}
	}
}

// TestFileGraphCyclesDeepChainDoesNotRecurse: a synthetic 10,000-node
// linear chain with one back edge from the last node to the first
// returns a single component of 10,000 members and does not overflow the
// goroutine stack. A recursive implementation fails this test — that is
// the point of the test.
func TestFileGraphCyclesDeepChainDoesNotRecurse(t *testing.T) {
	const n = 10000
	names := make([]string, n)
	for i := 0; i < n; i++ {
		names[i] = filePathForIndex(i)
	}
	adj := make(map[string][]string, n)
	for i := 0; i < n; i++ {
		next := (i + 1) % n // last node's edge wraps to the first, closing the cycle
		adj[names[i]] = []string{names[next]}
	}

	got := stronglyConnectedCycles(adj)
	if len(got) != n {
		t.Fatalf("stronglyConnectedCycles: got %d entries, want %d (one component spanning the whole chain)", len(got), n)
	}
	firstID := got[names[0]]
	if firstID == 0 {
		t.Fatalf("stronglyConnectedCycles: names[0] has zero cycle id")
	}
	for _, name := range names {
		if got[name] != firstID {
			t.Fatalf("stronglyConnectedCycles: %q has id %d, want %d (single component)", name, got[name], firstID)
		}
	}
}

// filePathForIndex generates a deterministic, sortable synthetic file
// path for TestFileGraphCyclesDeepChainDoesNotRecurse's 10,000-node chain.
func filePathForIndex(i int) string {
	digits := "0123456789"
	buf := make([]byte, 5)
	for pos := 4; pos >= 0; pos-- {
		buf[pos] = digits[i%10]
		i /= 10
	}
	return "chain/" + string(buf) + ".go"
}

// TestFileGraphPopulatesCycleFields: an Engine.FileGraph() call over a
// fixture containing a two-file cycle returns CycleCount equal to 1,
// exactly two nodes with a non-zero cycle id, and both edges of the
// cycle marked in-cycle, while a third non-cycle edge in the same
// fixture is not marked. Both the marked and the unmarked side are
// asserted.
func TestFileGraphPopulatesCycleFields(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
		"c": {Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls}, // cycle edge 1: a.go -> b.go
		{Source: "b", Target: "a", Kind: goextract.RefKindCalls}, // cycle edge 2: b.go -> a.go
		{Source: "a", Target: "c", Kind: goextract.RefKindCalls}, // non-cycle edge: a.go -> c.go
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if got.CycleCount != 1 {
		t.Fatalf("FileGraph: CycleCount = %d, want 1", got.CycleCount)
	}
	nonZero := 0
	for _, n := range got.Nodes {
		if n.CycleID != 0 {
			nonZero++
		}
	}
	if nonZero != 2 {
		t.Fatalf("FileGraph: %d nodes with non-zero CycleID, want exactly 2", nonZero)
	}

	var marked, unmarked int
	for _, edge := range got.Edges {
		isCycleEdge := (edge.SourceFile == "a.go" && edge.TargetFile == "b.go") || (edge.SourceFile == "b.go" && edge.TargetFile == "a.go")
		if isCycleEdge {
			if !edge.InCycle {
				t.Fatalf("FileGraph: cycle edge %s -> %s not marked InCycle", edge.SourceFile, edge.TargetFile)
			}
			marked++
		} else {
			if edge.InCycle {
				t.Fatalf("FileGraph: non-cycle edge %s -> %s incorrectly marked InCycle", edge.SourceFile, edge.TargetFile)
			}
			unmarked++
		}
	}
	if marked != 2 {
		t.Fatalf("FileGraph: %d cycle edges marked InCycle, want 2", marked)
	}
	if unmarked != 1 {
		t.Fatalf("FileGraph: %d non-cycle edges checked, want exactly 1 (a.go -> c.go)", unmarked)
	}
}
