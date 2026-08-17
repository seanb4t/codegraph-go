package query

import (
	"errors"
	"fmt"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// expandFakeReader is a minimal in-memory graphstore.Reader driving H10-
// H12 against a fully-controlled synthetic node/edge set (mirrors
// gather_test.go's/traverse_test.go's own "each plan may need its own
// reader test double" convention). GetNode returns graphstore.ErrNotFound
// for any id not present in nodes, matching a real Reader's dangling-
// reference contract (WR-04).
type expandFakeReader struct {
	nodes map[string]*schema.Node
	edges []*schema.Edge
}

func (f *expandFakeReader) GetNode(id string) (*schema.Node, error) {
	n, ok := f.nodes[id]
	if !ok {
		return nil, graphstore.ErrNotFound
	}
	return n, nil
}
func (f *expandFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("expandFakeReader: GetFile not implemented")
}
func (f *expandFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("expandFakeReader: GetMeta not implemented")
}
func (f *expandFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("expandFakeReader: IterateFiles not implemented")
}
func (f *expandFakeReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("expandFakeReader: IterateFileIndex not implemented")
}
func (f *expandFakeReader) Close() error { return nil }

func (f *expandFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	nodes := make([]*schema.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		nodes = append(nodes, n)
	}
	return &expandFakeNodeIterator{nodes: nodes}, nil
}

func (f *expandFakeReader) IterateEdges(prefix string) (graphstore.EdgeIterator, error) {
	var filtered []*schema.Edge
	for _, e := range f.edges {
		if prefix == "" || e.Source == prefix {
			filtered = append(filtered, e)
		}
	}
	return &expandFakeEdgeIterator{edges: filtered}, nil
}

type expandFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *expandFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}
func (it *expandFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *expandFakeNodeIterator) Err() error         { return nil }
func (it *expandFakeNodeIterator) Close() error       { return nil }

type expandFakeEdgeIterator struct {
	edges []*schema.Edge
	i     int
}

func (it *expandFakeEdgeIterator) Next() bool {
	if it.i >= len(it.edges) {
		return false
	}
	it.i++
	return true
}
func (it *expandFakeEdgeIterator) Edge() *schema.Edge { return it.edges[it.i-1] }
func (it *expandFakeEdgeIterator) Err() error         { return nil }
func (it *expandFakeEdgeIterator) Close() error       { return nil }

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// --- H10: TestTypeHierarchyExpansion ---

// TestTypeHierarchyExpansion_AncestorsAndDescendants pins H10's Pass-1
// rule: a focal class's extends/implements ancestors AND descendants are
// both added.
func TestTypeHierarchyExpansion_AncestorsAndDescendants(t *testing.T) {
	nodes := map[string]*schema.Node{
		"A": {Id: "A", Kind: goextract.KindStruct, Name: "A"},
		"B": {Id: "B", Kind: goextract.KindStruct, Name: "B"}, // focal
		"C": {Id: "C", Kind: goextract.KindStruct, Name: "C"}, // extends B (descendant)
	}
	edges := []*schema.Edge{
		{Source: "B", Target: "A", Kind: goextract.EdgeKindExtends}, // B extends A
		{Source: "C", Target: "B", Kind: goextract.EdgeKindExtends}, // C extends B
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	got, err := expandTypeHierarchy(r, []string{"B"}, 200)
	if err != nil {
		t.Fatalf("expandTypeHierarchy: unexpected error: %v", err)
	}
	if !containsID(got, "A") {
		t.Errorf("expected ancestor A in expansion, got %v", got)
	}
	if !containsID(got, "C") {
		t.Errorf("expected descendant C in expansion, got %v", got)
	}
	if containsID(got, "B") {
		t.Errorf("focal node B must not be re-added to the expansion, got %v", got)
	}
}

// TestTypeHierarchyExpansion_SecondPassSiblings pins H10's Pass-2 rule:
// a newly-found parent's OTHER direct children (siblings of the focal
// node not reachable in Pass 1) are added too.
func TestTypeHierarchyExpansion_SecondPassSiblings(t *testing.T) {
	nodes := map[string]*schema.Node{
		"A": {Id: "A", Kind: goextract.KindStruct, Name: "A"},
		"B": {Id: "B", Kind: goextract.KindStruct, Name: "B"}, // focal
		"D": {Id: "D", Kind: goextract.KindStruct, Name: "D"}, // also extends A (sibling of B)
	}
	edges := []*schema.Edge{
		{Source: "B", Target: "A", Kind: goextract.EdgeKindExtends},
		{Source: "D", Target: "A", Kind: goextract.EdgeKindExtends},
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	got, err := expandTypeHierarchy(r, []string{"B"}, 200)
	if err != nil {
		t.Fatalf("expandTypeHierarchy: unexpected error: %v", err)
	}
	if !containsID(got, "A") {
		t.Fatalf("expected parent A in expansion, got %v", got)
	}
	if !containsID(got, "D") {
		t.Errorf("expected Pass-2 sibling D (also extends A) in expansion, got %v", got)
	}
}

// TestTypeHierarchyExpansion_BudgetCap pins H10's exact, cited constant:
// the expansion is bounded to ceil(maxNodes/4) newly-added node ids,
// even when far more ancestors/descendants exist.
func TestTypeHierarchyExpansion_BudgetCap(t *testing.T) {
	nodes := map[string]*schema.Node{
		"focal": {Id: "focal", Kind: goextract.KindStruct, Name: "Focal"},
	}
	var edges []*schema.Edge
	// 20 direct descendants of focal — far more than ceil(8/4)=2.
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("child%02d", i)
		nodes[id] = &schema.Node{Id: id, Kind: goextract.KindStruct, Name: id}
		edges = append(edges, &schema.Edge{Source: id, Target: "focal", Kind: goextract.EdgeKindExtends})
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	const maxNodes = 8
	wantBudget := expandHierarchyBudget(maxNodes)
	if wantBudget != 2 {
		t.Fatalf("expandHierarchyBudget(%d) = %d, want 2 (ceil(8/4))", maxNodes, wantBudget)
	}

	got, err := expandTypeHierarchy(r, []string{"focal"}, maxNodes)
	if err != nil {
		t.Fatalf("expandTypeHierarchy: unexpected error: %v", err)
	}
	if len(got) != wantBudget {
		t.Fatalf("expandTypeHierarchy budget: got %d nodes, want exactly %d (the ceil(maxNodes/4) budget)", len(got), wantBudget)
	}
}

// TestTypeHierarchyExpansion_NonTypeFocalIgnored asserts H10 only fires
// for class/interface/struct/trait/protocol-shaped focal nodes — a
// function focal id yields no expansion.
func TestTypeHierarchyExpansion_NonTypeFocalIgnored(t *testing.T) {
	nodes := map[string]*schema.Node{
		"fn": {Id: "fn", Kind: goextract.KindFunction, Name: "Fn"},
		"A":  {Id: "A", Kind: goextract.KindStruct, Name: "A"},
	}
	edges := []*schema.Edge{
		{Source: "fn", Target: "A", Kind: goextract.EdgeKindExtends}, // nonsensical but must still be ignored
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	got, err := expandTypeHierarchy(r, []string{"fn"}, 200)
	if err != nil {
		t.Fatalf("expandTypeHierarchy: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expandTypeHierarchy over a non-type focal: got %v, want empty", got)
	}
}

// --- H11: TestBFSBounds ---

func gc(id string, score float64) gatherCandidate {
	return gatherCandidate{Node: &schema.Node{Id: id, Kind: goextract.KindFunction, Name: id}, Score: score}
}

// TestBFSBounds_DepthLimit pins H11's traversalDepth=3: a chain 5 hops
// deep from the seed root only surfaces nodes within 3 hops.
func TestBFSBounds_DepthLimit(t *testing.T) {
	nodes := map[string]*schema.Node{
		"root": {Id: "root", Kind: goextract.KindFunction, Name: "root"},
	}
	var edges []*schema.Edge
	prev := "root"
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("n%d", i)
		nodes[id] = &schema.Node{Id: id, Kind: goextract.KindFunction, Name: id}
		edges = append(edges, &schema.Edge{Source: prev, Target: id, Kind: goextract.RefKindCalls})
		prev = id
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	nodeIDs, _, _, err := expandBFS(r, []gatherCandidate{gc("root", 1.0)}, DefaultExploreBFSBounds)
	if err != nil {
		t.Fatalf("expandBFS: unexpected error: %v", err)
	}
	for _, want := range []string{"root", "n1", "n2", "n3"} {
		if !containsID(nodeIDs, want) {
			t.Errorf("expected %s within traversalDepth=3, got %v", want, nodeIDs)
		}
	}
	for _, notWant := range []string{"n4", "n5"} {
		if containsID(nodeIDs, notWant) {
			t.Errorf("expected %s to be OUTSIDE traversalDepth=3, got %v", notWant, nodeIDs)
		}
	}
}

// TestBFSBounds_NodeCap pins H11's maxNodes=200 override, checked here
// against a small override value so the test stays fast: the subgraph
// never exceeds bounds.MaxNodes total nodes even when far more are
// reachable within traversalDepth.
func TestBFSBounds_NodeCap(t *testing.T) {
	nodes := map[string]*schema.Node{
		"root": {Id: "root", Kind: goextract.KindFunction, Name: "root"},
	}
	var edges []*schema.Edge
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("n%02d", i)
		nodes[id] = &schema.Node{Id: id, Kind: goextract.KindFunction, Name: id}
		edges = append(edges, &schema.Edge{Source: "root", Target: id, Kind: goextract.RefKindCalls})
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	bounds := DefaultExploreBFSBounds
	bounds.MaxNodes = 10
	nodeIDs, _, _, err := expandBFS(r, []gatherCandidate{gc("root", 1.0)}, bounds)
	if err != nil {
		t.Fatalf("expandBFS: unexpected error: %v", err)
	}
	if len(nodeIDs) != bounds.MaxNodes {
		t.Fatalf("expandBFS node cap: got %d nodes, want exactly bounds.MaxNodes=%d", len(nodeIDs), bounds.MaxNodes)
	}
}

// TestBFSBounds_MinScorePrune pins H11's minScore=0.2: a root candidate
// scoring below minScore never seeds the BFS at all.
func TestBFSBounds_MinScorePrune(t *testing.T) {
	nodes := map[string]*schema.Node{
		"lowScoreRoot": {Id: "lowScoreRoot", Kind: goextract.KindFunction, Name: "lowScoreRoot"},
		"reachable":    {Id: "reachable", Kind: goextract.KindFunction, Name: "reachable"},
	}
	edges := []*schema.Edge{
		{Source: "lowScoreRoot", Target: "reachable", Kind: goextract.RefKindCalls},
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	nodeIDs, seedIDs, _, err := expandBFS(r, []gatherCandidate{gc("lowScoreRoot", 0.1)}, DefaultExploreBFSBounds)
	if err != nil {
		t.Fatalf("expandBFS: unexpected error: %v", err)
	}
	if len(seedIDs) != 0 {
		t.Fatalf("a root scoring below minScore=%v must be pruned before seeding, got seedIDs=%v", ExpandMinScore, seedIDs)
	}
	if len(nodeIDs) != 0 {
		t.Fatalf("no seeds survived minScore pruning, so the subgraph must be empty, got %v", nodeIDs)
	}
}

// TestBFSBounds_SearchLimit pins H11's searchLimit=8: more than
// searchLimit surviving root candidates are trimmed to the top
// searchLimit by score (D-04 tie-break).
func TestBFSBounds_SearchLimit(t *testing.T) {
	nodes := map[string]*schema.Node{}
	var candidates []gatherCandidate
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("root%02d", i)
		nodes[id] = &schema.Node{Id: id, Kind: goextract.KindFunction, Name: id}
		candidates = append(candidates, gc(id, float64(i))) // higher i = higher score
	}
	r := &expandFakeReader{nodes: nodes}

	_, seedIDs, _, err := expandBFS(r, candidates, DefaultExploreBFSBounds)
	if err != nil {
		t.Fatalf("expandBFS: unexpected error: %v", err)
	}
	if len(seedIDs) != ExpandSearchLimit {
		t.Fatalf("expandBFS searchLimit: got %d seeds, want exactly ExpandSearchLimit=%d", len(seedIDs), ExpandSearchLimit)
	}
	// The 4 lowest-scored roots (root00..root03) must have been trimmed.
	for _, low := range []string{"root00", "root01", "root02", "root03"} {
		if containsID(seedIDs, low) {
			t.Errorf("expected lowest-scored root %s to be trimmed by searchLimit, got seedIDs=%v", low, seedIDs)
		}
	}
}

// --- H12: TestGlueNodeInjection / TestGlueNodeCap ---

// TestGlueNodeInjection_SameFileOnly pins H12's same-file-only
// constraint: a root's caller living in a file already surfaced by the
// subgraph is injected; a caller in a file NOT yet surfaced is not.
func TestGlueNodeInjection_SameFileOnly(t *testing.T) {
	nodes := map[string]*schema.Node{
		"root":        {Id: "root", Kind: goextract.KindFunction, Name: "root", FilePath: "pkg/root.go"},
		"sameFile":    {Id: "sameFile", Kind: goextract.KindFunction, Name: "sameFile", FilePath: "pkg/root.go"},
		"otherFile":   {Id: "otherFile", Kind: goextract.KindFunction, Name: "otherFile", FilePath: "pkg/other.go"},
		"calleeSame":  {Id: "calleeSame", Kind: goextract.KindFunction, Name: "calleeSame", FilePath: "pkg/root.go"},
		"calleeOther": {Id: "calleeOther", Kind: goextract.KindFunction, Name: "calleeOther", FilePath: "pkg/other.go"},
	}
	edges := []*schema.Edge{
		{Source: "sameFile", Target: "root", Kind: goextract.RefKindCalls},    // caller, same file
		{Source: "otherFile", Target: "root", Kind: goextract.RefKindCalls},   // caller, other file
		{Source: "root", Target: "calleeSame", Kind: goextract.RefKindCalls},  // callee, same file
		{Source: "root", Target: "calleeOther", Kind: goextract.RefKindCalls}, // callee, other file
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	surfaced := map[string]bool{"pkg/root.go": true}
	got, err := expandGlueNodes(r, []string{"root"}, surfaced, GlueNodeCap)
	if err != nil {
		t.Fatalf("expandGlueNodes: unexpected error: %v", err)
	}
	if !containsID(got, "sameFile") {
		t.Errorf("expected same-file caller sameFile injected, got %v", got)
	}
	if !containsID(got, "calleeSame") {
		t.Errorf("expected same-file callee calleeSame injected, got %v", got)
	}
	if containsID(got, "otherFile") {
		t.Errorf("expected other-file caller otherFile EXCLUDED, got %v", got)
	}
	if containsID(got, "calleeOther") {
		t.Errorf("expected other-file callee calleeOther EXCLUDED, got %v", got)
	}
}

// TestGlueNodeCap_Binds pins H12's exact, cited constant: total glue
// nodes across every root are capped at GLUE_NODE_CAP=60, kept in
// deterministic sorted-Id order when the cap binds.
func TestGlueNodeCap_Binds(t *testing.T) {
	nodes := map[string]*schema.Node{
		"root": {Id: "root", Kind: goextract.KindFunction, Name: "root", FilePath: "pkg/root.go"},
	}
	var edges []*schema.Edge
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("caller%03d", i)
		nodes[id] = &schema.Node{Id: id, Kind: goextract.KindFunction, Name: id, FilePath: "pkg/root.go"}
		edges = append(edges, &schema.Edge{Source: id, Target: "root", Kind: goextract.RefKindCalls})
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	surfaced := map[string]bool{"pkg/root.go": true}
	got, err := expandGlueNodes(r, []string{"root"}, surfaced, GlueNodeCap)
	if err != nil {
		t.Fatalf("expandGlueNodes: unexpected error: %v", err)
	}
	if len(got) != GlueNodeCap {
		t.Fatalf("expandGlueNodes cap: got %d glue nodes, want exactly GlueNodeCap=%d", len(got), GlueNodeCap)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("expandGlueNodes cap must keep the lowest-Id candidates in sorted order, got %v", got)
		}
	}
	// Confirm it kept the LOWEST 60 ids (caller000..caller059), not an
	// arbitrary subset.
	if got[0] != "caller000" || got[len(got)-1] != "caller059" {
		t.Fatalf("expandGlueNodes cap: got range [%s, %s], want [caller000, caller059]", got[0], got[len(got)-1])
	}
}

// TestGlueNodeInjection_RootNeverInjectedAsOwnGlue asserts a root itself
// is never re-injected as a glue node even if reachable via another
// root's caller/callee edges.
func TestGlueNodeInjection_RootNeverInjectedAsOwnGlue(t *testing.T) {
	nodes := map[string]*schema.Node{
		"rootA": {Id: "rootA", Kind: goextract.KindFunction, Name: "rootA", FilePath: "pkg/a.go"},
		"rootB": {Id: "rootB", Kind: goextract.KindFunction, Name: "rootB", FilePath: "pkg/a.go"},
	}
	edges := []*schema.Edge{
		{Source: "rootA", Target: "rootB", Kind: goextract.RefKindCalls},
	}
	r := &expandFakeReader{nodes: nodes, edges: edges}

	surfaced := map[string]bool{"pkg/a.go": true}
	got, err := expandGlueNodes(r, []string{"rootA", "rootB"}, surfaced, GlueNodeCap)
	if err != nil {
		t.Fatalf("expandGlueNodes: unexpected error: %v", err)
	}
	if containsID(got, "rootA") || containsID(got, "rootB") {
		t.Fatalf("expandGlueNodes must never inject an existing root as its own glue node, got %v", got)
	}
}

// TestGlueNodeInjection_SubgraphFileSetHelper pins subgraphFileSet's own
// contract: it resolves ids to their distinct, non-empty FilePath set,
// skipping a dangling id (WR-04) rather than erroring.
func TestGlueNodeInjection_SubgraphFileSetHelper(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFunction, Name: "a", FilePath: "pkg/a.go"},
		"b": {Id: "b", Kind: goextract.KindFunction, Name: "b", FilePath: "pkg/b.go"},
	}
	r := &expandFakeReader{nodes: nodes}

	got, err := subgraphFileSet(r, []string{"a", "b", "dangling"})
	if err != nil {
		t.Fatalf("subgraphFileSet: unexpected error: %v", err)
	}
	if !got["pkg/a.go"] || !got["pkg/b.go"] {
		t.Fatalf("subgraphFileSet: got %v, want pkg/a.go and pkg/b.go present", got)
	}
	if len(got) != 2 {
		t.Fatalf("subgraphFileSet: got %d files, want exactly 2 (dangling id must be skipped, not error)", len(got))
	}
}
