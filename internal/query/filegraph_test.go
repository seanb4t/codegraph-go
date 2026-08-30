package query

import (
	"sync"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestFileGraphExcludesContainsEdges pins D-03: a "contains" edge rolled
// up to file granularity is excluded from the aggregated rollup, while a
// retained kind ("calls" here) between the same two files still
// contributes. Both are asserted so the absence of "contains" means
// something rather than an empty rollup passing vacuously.
func TestFileGraphExcludesContainsEdges(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "b", Kind: goextract.RefKindContains},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if len(got.Edges) != 1 {
		t.Fatalf("FileGraph: got %d edges, want exactly 1 (a.go -> b.go)", len(got.Edges))
	}
	edge := got.Edges[0]
	if _, ok := edge.KindCounts[goextract.RefKindCalls]; !ok {
		t.Fatalf("FileGraph: KindCounts missing %q, want it present (positive control)", goextract.RefKindCalls)
	}
	if _, ok := edge.KindCounts[goextract.RefKindContains]; ok {
		t.Fatalf("FileGraph: KindCounts contains %q, want it excluded (D-03)", goextract.RefKindContains)
	}
}

// TestFileGraphExcludesSelfEdges pins D-03's self-edge exclusion: two
// symbols in the SAME file with a "calls" edge between them contribute
// zero aggregated edges, while a cross-file "calls" edge in the same
// fixture still contributes one. A zero-only assertion would pass on an
// empty rollup, so the cross-file edge is the positive control.
func TestFileGraphExcludesSelfEdges(t *testing.T) {
	nodes := map[string]*schema.Node{
		"s1": {Id: "s1", Kind: goextract.KindFunction, FilePath: "a.go"},
		"s2": {Id: "s2", Kind: goextract.KindFunction, FilePath: "a.go"},
		"s3": {Id: "s3", Kind: goextract.KindFunction, FilePath: "b.go"},
	}
	edges := []*schema.Edge{
		{Source: "s1", Target: "s2", Kind: goextract.RefKindCalls}, // same file (a.go) — self-edge
		{Source: "s1", Target: "s3", Kind: goextract.RefKindCalls}, // cross-file (a.go -> b.go)
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if len(got.Edges) != 1 {
		t.Fatalf("FileGraph: got %d edges, want exactly 1 (the cross-file edge)", len(got.Edges))
	}
	if got.Edges[0].SourceFile != "a.go" || got.Edges[0].TargetFile != "b.go" {
		t.Fatalf("FileGraph: got edge %s -> %s, want a.go -> b.go", got.Edges[0].SourceFile, got.Edges[0].TargetFile)
	}
	if got.ExcludedSelfEdges < 1 {
		t.Fatalf("FileGraph: ExcludedSelfEdges = %d, want at least 1", got.ExcludedSelfEdges)
	}
}

// TestFileGraphExcludesPackagePseudoNodes pins D-08: a fixture carrying a
// synthetic "package"-kind node with an empty file path, and an "imports"
// edge targeting it, yields no node whose path is empty and no edge
// referencing an empty path — while ExcludedPackageNodes is greater than
// zero, proving the exclusion ran rather than the fixture being empty.
func TestFileGraphExcludesPackagePseudoNodes(t *testing.T) {
	nodes := map[string]*schema.Node{
		"pkg": {Id: "pkg", Kind: "package", Name: "internal/foo", QualifiedName: "internal/foo", FilePath: ""},
		"a":   {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "pkg", Kind: goextract.RefKindImports},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if got.ExcludedPackageNodes < 1 {
		t.Fatalf("FileGraph: ExcludedPackageNodes = %d, want at least 1", got.ExcludedPackageNodes)
	}
	for _, n := range got.Nodes {
		if n.Path == "" {
			t.Fatalf("FileGraph: node with empty path present, want package pseudo-nodes excluded")
		}
	}
	for _, edge := range got.Edges {
		if edge.SourceFile == "" || edge.TargetFile == "" {
			t.Fatalf("FileGraph: edge referencing empty path present: %+v", edge)
		}
	}
	foundA := false
	for _, n := range got.Nodes {
		if n.Path == "a.go" {
			foundA = true
		}
	}
	if !foundA {
		t.Fatalf("FileGraph: a.go missing from node set")
	}
}

// TestFileGraphSparseKindCounts pins the edgesByKind sparse-map
// convention (status.go): no aggregated edge's KindCounts contains a key
// whose value is 0, and at least one map is non-empty.
func TestFileGraphSparseKindCounts(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
		"c": {Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "b", Kind: goextract.RefKindReferences},
		{Source: "b", Target: "c", Kind: goextract.RefKindImports},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	nonEmpty := false
	for _, edge := range got.Edges {
		if len(edge.KindCounts) > 0 {
			nonEmpty = true
		}
		for kind, count := range edge.KindCounts {
			if count == 0 {
				t.Fatalf("FileGraph: KindCounts[%q] = 0 on edge %s -> %s, want absent not zero", kind, edge.SourceFile, edge.TargetFile)
			}
		}
	}
	if !nonEmpty {
		t.Fatalf("FileGraph: no edge carries a non-empty KindCounts map")
	}
}

// TestFileGraphSymbolCountExcludesFileNode pins review M-4: a file
// carrying its own file-kind record PLUS exactly three symbol records
// reports SymbolCount of exactly 3, not 4 — the file-kind record is not a
// declared symbol. A second fixture file with one symbol reports exactly
// 1, so the assertion is not satisfied by a constant. Both files are
// asserted present as nodes so the count cannot be reached by dropping
// the file from the rollup entirely.
func TestFileGraphSymbolCountExcludesFileNode(t *testing.T) {
	nodes := map[string]*schema.Node{
		"fa":  {Id: "fa", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"sa1": {Id: "sa1", Kind: goextract.KindFunction, FilePath: "a.go", Language: "go"},
		"sa2": {Id: "sa2", Kind: goextract.KindFunction, FilePath: "a.go", Language: "go"},
		"sa3": {Id: "sa3", Kind: goextract.KindFunction, FilePath: "a.go", Language: "go"},
		"fb":  {Id: "fb", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
		"sb1": {Id: "sb1", Kind: goextract.KindFunction, FilePath: "b.go", Language: "go"},
	}
	e := New(&traverseFakeReader{nodes: nodes})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	var aNode, bNode *FileGraphNode
	for i := range got.Nodes {
		switch got.Nodes[i].Path {
		case "a.go":
			aNode = &got.Nodes[i]
		case "b.go":
			bNode = &got.Nodes[i]
		}
	}
	if aNode == nil {
		t.Fatalf("FileGraph: a.go missing from node set")
	}
	if bNode == nil {
		t.Fatalf("FileGraph: b.go missing from node set")
	}
	if aNode.SymbolCount != 3 {
		t.Fatalf("FileGraph: a.go SymbolCount = %d, want exactly 3 (file-kind record excluded)", aNode.SymbolCount)
	}
	if bNode.SymbolCount != 1 {
		t.Fatalf("FileGraph: b.go SymbolCount = %d, want exactly 1", bNode.SymbolCount)
	}
}

// TestFileGraphDeterministicOrder pins the ordering contract: two
// consecutive FileGraph() calls on one Engine return node slices equal
// element for element and edge slices equal element for element; nodes
// ascend by path, edges ascend by source file then target file.
func TestFileGraphDeterministicOrder(t *testing.T) {
	nodes := map[string]*schema.Node{
		"c": {Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "c", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "c", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	first, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph (first call): unexpected error: %v", err)
	}
	second, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph (second call): unexpected error: %v", err)
	}

	if len(first.Nodes) != len(second.Nodes) || len(first.Edges) != len(second.Edges) {
		t.Fatalf("FileGraph: call count mismatch: first %d/%d, second %d/%d", len(first.Nodes), len(first.Edges), len(second.Nodes), len(second.Edges))
	}
	for i := range first.Nodes {
		if first.Nodes[i] != second.Nodes[i] {
			t.Fatalf("FileGraph: node[%d] differs between calls: %+v vs %+v", i, first.Nodes[i], second.Nodes[i])
		}
	}
	for i := range first.Edges {
		if first.Edges[i].SourceFile != second.Edges[i].SourceFile || first.Edges[i].TargetFile != second.Edges[i].TargetFile {
			t.Fatalf("FileGraph: edge[%d] differs between calls: %+v vs %+v", i, first.Edges[i], second.Edges[i])
		}
	}

	for i := 1; i < len(first.Nodes); i++ {
		if first.Nodes[i-1].Path >= first.Nodes[i].Path {
			t.Fatalf("FileGraph: nodes not ascending by path at index %d: %q >= %q", i, first.Nodes[i-1].Path, first.Nodes[i].Path)
		}
	}
	for i := 1; i < len(first.Edges); i++ {
		prev, cur := first.Edges[i-1], first.Edges[i]
		if prev.SourceFile > cur.SourceFile || (prev.SourceFile == cur.SourceFile && prev.TargetFile >= cur.TargetFile) {
			t.Fatalf("FileGraph: edges not ascending by (source,target) at index %d: %s/%s then %s/%s", i, prev.SourceFile, prev.TargetFile, cur.SourceFile, cur.TargetFile)
		}
	}
}

// TestFileGraphRepoRelativePaths pins T-05-05: node count is asserted
// greater than zero first (so the path check cannot pass vacuously on an
// empty result), then every returned node path is non-empty, does not
// begin with a forward slash, and does not match a drive-letter prefix.
func TestFileGraphRepoRelativePaths(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "cmd/root.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "internal/query/traverse.go", Language: "go"},
	}
	e := New(&traverseFakeReader{nodes: nodes})

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if len(got.Nodes) == 0 {
		t.Fatalf("FileGraph: node count is zero, want > 0 before checking path shape")
	}
	for _, n := range got.Nodes {
		if n.Path == "" {
			t.Fatalf("FileGraph: node with empty path present")
		}
		if n.Path[0] == '/' {
			t.Fatalf("FileGraph: node path %q begins with a forward slash, want repo-relative", n.Path)
		}
		if len(n.Path) >= 2 && n.Path[1] == ':' {
			t.Fatalf("FileGraph: node path %q matches a drive-letter prefix, want repo-relative", n.Path)
		}
	}
}

// TestFileGraphConcurrentCallsAgree is ENG-03's concurrency guarantee:
// the Engine holds ONE graphstore.Reader snapshot, so four concurrent
// FileGraph() calls on one Engine cannot disagree, and a background
// writer cannot mutate an open snapshot. Run under `go test -race`.
func TestFileGraphConcurrentCallsAgree(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
		"b": {Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
		"c": {Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
	}
	edges := []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "b", Target: "c", Kind: goextract.RefKindImports},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	const workers = 4
	nodeCounts := make([]int, workers)
	edgeCounts := make([]int, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			res, err := e.FileGraph()
			if err != nil {
				errs[i] = err
				return
			}
			nodeCounts[i] = len(res.Nodes)
			edgeCounts[i] = len(res.Edges)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("FileGraph (goroutine %d): unexpected error: %v", i, err)
		}
	}
	if nodeCounts[0] == 0 || edgeCounts[0] == 0 {
		t.Fatalf("FileGraph: baseline counts are zero (nodes=%d edges=%d), want > 0", nodeCounts[0], edgeCounts[0])
	}
	for i := 1; i < workers; i++ {
		if nodeCounts[i] != nodeCounts[0] {
			t.Fatalf("FileGraph: goroutine %d node count %d disagrees with goroutine 0's %d", i, nodeCounts[i], nodeCounts[0])
		}
		if edgeCounts[i] != edgeCounts[0] {
			t.Fatalf("FileGraph: goroutine %d edge count %d disagrees with goroutine 0's %d", i, edgeCounts[i], edgeCounts[0])
		}
	}
}

// TestFileGraphAgainstThisRepositoryIndex is gated on this repository's
// own .codegraph/store being present: it asserts the corrected D-08
// figures (572 nodes, 1,057 distinct source-to-target file pairs).
// google/guava carries zero package pseudo-nodes, so the measurement
// corpus GRF-01 uses structurally cannot catch a regression here — only
// this repository's own index can.
func TestFileGraphAgainstThisRepositoryIndex(t *testing.T) {
	e, closer, err := OpenAt(".")
	if err != nil {
		t.Skipf("this repository's own .codegraph/store is not available: %v", err)
	}
	defer closer.Close()

	got, err := e.FileGraph()
	if err != nil {
		t.Fatalf("FileGraph: unexpected error: %v", err)
	}
	if len(got.Nodes) != 572 {
		t.Fatalf("FileGraph: got %d nodes against this repository's own index, want 572 (the corrected D-08 figure)", len(got.Nodes))
	}
	if len(got.Edges) != 1057 {
		t.Fatalf("FileGraph: got %d distinct source-to-target file pairs against this repository's own index, want 1057 (the corrected D-08 figure)", len(got.Edges))
	}
}
