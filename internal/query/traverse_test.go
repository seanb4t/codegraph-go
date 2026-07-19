package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// traverseFakeReader is a minimal in-memory graphstore.Reader used to
// exercise Callees/Callers/Impact against a fully-controlled node/edge set
// — independent of a real Pebble-backed store — so CR-01's default-cap
// behavior (thousands of synthetic edges) and WR-04's dangling-edge
// graceful-degradation behavior can both be proven deterministically
// without indexing a huge or deliberately-corrupt fixture. GetNode returns
// graphstore.ErrNotFound for any id not present in nodes, matching a real
// Reader's contract for a dangling reference.
type traverseFakeReader struct {
	nodes map[string]*schema.Node
	edges []*schema.Edge
}

func (f *traverseFakeReader) GetNode(id string) (*schema.Node, error) {
	n, ok := f.nodes[id]
	if !ok {
		return nil, graphstore.ErrNotFound
	}
	return n, nil
}

func (f *traverseFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("traverseFakeReader: GetFile not implemented")
}

func (f *traverseFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("traverseFakeReader: GetMeta not implemented")
}

func (f *traverseFakeReader) GetMigration() ([]byte, error) {
	return nil, errors.New("traverseFakeReader: GetMigration not implemented")
}

func (f *traverseFakeReader) IterateEdges(prefix string) (graphstore.EdgeIterator, error) {
	var filtered []*schema.Edge
	for _, e := range f.edges {
		if prefix == "" || e.Source == prefix {
			filtered = append(filtered, e)
		}
	}
	return &traverseFakeEdgeIterator{edges: filtered}, nil
}

func (f *traverseFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	nodes := make([]*schema.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		nodes = append(nodes, n)
	}
	return &traverseFakeNodeIterator{nodes: nodes}, nil
}

func (f *traverseFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("traverseFakeReader: IterateFiles not implemented")
}

func (f *traverseFakeReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("traverseFakeReader: IterateFileIndex not implemented")
}

func (f *traverseFakeReader) Close() error { return nil }

type traverseFakeEdgeIterator struct {
	edges []*schema.Edge
	i     int
}

func (it *traverseFakeEdgeIterator) Next() bool {
	if it.i >= len(it.edges) {
		return false
	}
	it.i++
	return true
}

func (it *traverseFakeEdgeIterator) Edge() *schema.Edge { return it.edges[it.i-1] }
func (it *traverseFakeEdgeIterator) Err() error         { return nil }
func (it *traverseFakeEdgeIterator) Close() error       { return nil }

type traverseFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *traverseFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}

func (it *traverseFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *traverseFakeNodeIterator) Err() error         { return nil }
func (it *traverseFakeNodeIterator) Close() error       { return nil }

// TestCalleesDefaultCapAtMaxLimit pins CR-01 for Callees: the MaxLimit
// ceiling must apply even when limit==0 (no explicit --limit).
func TestCalleesDefaultCapAtMaxLimit(t *testing.T) {
	nodes := map[string]*schema.Node{
		"origin": {Id: "origin", Kind: "function", Name: "Origin", QualifiedName: "Origin"},
	}
	var edges []*schema.Edge
	for i := 0; i < MaxLimit+50; i++ {
		id := fmt.Sprintf("target%04d", i)
		nodes[id] = &schema.Node{Id: id, Kind: "function", Name: id, QualifiedName: id}
		edges = append(edges, &schema.Edge{Source: "origin", Target: id, Kind: goextract.RefKindCalls})
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callees("Origin", 0)
	if err != nil {
		t.Fatalf("Callees: unexpected error: %v", err)
	}
	if len(got.Callees) != MaxLimit {
		t.Fatalf("Callees with limit=0 (default): got %d entries, want the MaxLimit=%d ceiling to apply", len(got.Callees), MaxLimit)
	}
}

// TestCallersDefaultCapAtMaxLimit mirrors TestCalleesDefaultCapAtMaxLimit
// for Callers (CR-01).
func TestCallersDefaultCapAtMaxLimit(t *testing.T) {
	nodes := map[string]*schema.Node{
		"target": {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target"},
	}
	var edges []*schema.Edge
	for i := 0; i < MaxLimit+50; i++ {
		id := fmt.Sprintf("caller%04d", i)
		nodes[id] = &schema.Node{Id: id, Kind: "function", Name: id, QualifiedName: id}
		edges = append(edges, &schema.Edge{Source: id, Target: "target", Kind: goextract.RefKindCalls})
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callers("Target", 0)
	if err != nil {
		t.Fatalf("Callers: unexpected error: %v", err)
	}
	if len(got.Callers) != MaxLimit {
		t.Fatalf("Callers with limit=0 (default): got %d entries, want the MaxLimit=%d ceiling to apply", len(got.Callers), MaxLimit)
	}
}

// TestCalleesSkipsDanglingEdgeInsteadOfFailing pins WR-04: a dangling
// edge (target node missing/pruned) must be skipped, not abort the whole
// Callees call.
func TestCalleesSkipsDanglingEdgeInsteadOfFailing(t *testing.T) {
	nodes := map[string]*schema.Node{
		"origin": {Id: "origin", Kind: "function", Name: "Origin", QualifiedName: "Origin"},
		"live":   {Id: "live", Kind: "function", Name: "Live", QualifiedName: "Live"},
	}
	edges := []*schema.Edge{
		{Source: "origin", Target: "live", Kind: goextract.RefKindCalls},
		{Source: "origin", Target: "missing", Kind: goextract.RefKindCalls}, // dangling
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callees("Origin", 0)
	if err != nil {
		t.Fatalf("Callees: unexpected error from a dangling edge, want graceful skip: %v", err)
	}
	if len(got.Callees) != 1 || got.Callees[0].Name != "Live" {
		t.Fatalf("Callees: got %+v, want exactly [Live] (dangling edge skipped)", got.Callees)
	}
}

// TestCallersSkipsDanglingEdgeInsteadOfFailing mirrors
// TestCalleesSkipsDanglingEdgeInsteadOfFailing for Callers (WR-04).
func TestCallersSkipsDanglingEdgeInsteadOfFailing(t *testing.T) {
	nodes := map[string]*schema.Node{
		"target": {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target"},
		"live":   {Id: "live", Kind: "function", Name: "Live", QualifiedName: "Live"},
	}
	edges := []*schema.Edge{
		{Source: "live", Target: "target", Kind: goextract.RefKindCalls},
		{Source: "missing", Target: "target", Kind: goextract.RefKindCalls}, // dangling
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callers("Target", 0)
	if err != nil {
		t.Fatalf("Callers: unexpected error from a dangling edge, want graceful skip: %v", err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Name != "Live" {
		t.Fatalf("Callers: got %+v, want exactly [Live] (dangling edge skipped)", got.Callers)
	}
}

// TestImpactSkipsDanglingEdgeInsteadOfFailing pins WR-04 for Impact's
// BFS: a dangling reverse-edge source must be skipped, not abort the
// whole traversal, and must not be counted as a resolved node.
func TestImpactSkipsDanglingEdgeInsteadOfFailing(t *testing.T) {
	nodes := map[string]*schema.Node{
		"target": {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target"},
		"live":   {Id: "live", Kind: "function", Name: "Live", QualifiedName: "Live"},
	}
	edges := []*schema.Edge{
		{Source: "live", Target: "target", Kind: goextract.RefKindCalls},
		{Source: "missing", Target: "target", Kind: goextract.RefKindCalls}, // dangling
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Impact("Target", 2)
	if err != nil {
		t.Fatalf("Impact: unexpected error from a dangling edge, want graceful skip: %v", err)
	}
	// target (self) + live (hop 1) = 2 nodes; the dangling edge is
	// skipped entirely, not counted as a resolved node.
	if got.NodeCount != 2 {
		t.Fatalf("Impact NodeCount: got %d, want 2 (target, live) — dangling edge must be skipped, not counted", got.NodeCount)
	}
}

// dispatchFixtureNodesEdges builds a small in-memory graph proving RES-02's
// query-time dispatch traversal: structs A and B both implement interface
// I and both declare a method named "Handle" — A's node id sorts lower
// than B's ("method:a-handle" < "method:b-handle"), so
// resolveSymbolNode("Handle") deterministically picks A.Handle, which has
// NO direct caller of its own. Only B.Handle is actually called (by
// Caller). The dispatch-traversal composition must still surface Caller
// as a "Handle" caller/impact result, since a call dispatched dynamically
// through interface I could have reached either implementer.
func dispatchFixtureNodesEdges() (map[string]*schema.Node, []*schema.Edge) {
	nodes := map[string]*schema.Node{
		"iface:I":         {Id: "iface:I", Kind: "interface", Name: "I", QualifiedName: "I"},
		"struct:A":        {Id: "struct:A", Kind: "struct", Name: "A", QualifiedName: "A"},
		"struct:B":        {Id: "struct:B", Kind: "struct", Name: "B", QualifiedName: "B"},
		"method:a-handle": {Id: "method:a-handle", Kind: "method", Name: "Handle", QualifiedName: "A.Handle"},
		"method:b-handle": {Id: "method:b-handle", Kind: "method", Name: "Handle", QualifiedName: "B.Handle"},
		"func:caller":     {Id: "func:caller", Kind: "function", Name: "Caller", QualifiedName: "Caller"},
	}
	edges := []*schema.Edge{
		{Source: "struct:A", Target: "iface:I", Kind: goextract.EdgeKindImplements, Provenance: "heuristic"},
		{Source: "struct:B", Target: "iface:I", Kind: goextract.EdgeKindImplements, Provenance: "heuristic"},
		{Source: "struct:A", Target: "method:a-handle", Kind: "contains", Provenance: "ast"},
		{Source: "struct:B", Target: "method:b-handle", Kind: "contains", Provenance: "ast"},
		{Source: "func:caller", Target: "method:b-handle", Kind: goextract.RefKindCalls, Provenance: "ast"},
	}
	return nodes, edges
}

// TestCallers_DispatchTraversal proves RES-02: Callers("Handle") resolves
// to A.Handle (lower node id) but must ALSO surface Caller — the caller of
// SIBLING implementer B's same-named Handle method — via the "implements"
// edge composition, even though no edge points at A.Handle directly.
func TestCallers_DispatchTraversal(t *testing.T) {
	nodes, edges := dispatchFixtureNodesEdges()
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callers("Handle", 0)
	if err != nil {
		t.Fatalf("Callers: unexpected error: %v", err)
	}

	var found bool
	for _, loc := range got.Callers {
		if loc.Name == "Caller" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Callers(Handle) = %+v, want Caller present (dispatch traversal through interface I)", got.Callers)
	}
}

// TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected proves the
// dispatch composition is a strict no-op when the graph has no
// "implements" edges at all (the pre-Phase-5 common case) — regression
// guard alongside TestCallersCallees's own real-fixture assertions.
func TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected(t *testing.T) {
	nodes := map[string]*schema.Node{
		"target": {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target"},
		"live":   {Id: "live", Kind: "function", Name: "Live", QualifiedName: "Live"},
	}
	edges := []*schema.Edge{
		{Source: "live", Target: "target", Kind: goextract.RefKindCalls},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Callers("Target", 0)
	if err != nil {
		t.Fatalf("Callers: unexpected error: %v", err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Name != "Live" {
		t.Fatalf("Callers: got %+v, want exactly [Live] (no implements edges, no composition)", got.Callers)
	}
}

// TestImpact_DispatchTraversal proves RES-02: Impact("Handle") at depth 1
// includes Caller in Affected/NodeCount — dispatch traversal composed into
// the BFS frontier expansion, not just Callers' single-hop case.
func TestImpact_DispatchTraversal(t *testing.T) {
	nodes, edges := dispatchFixtureNodesEdges()
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Impact("Handle", 1)
	if err != nil {
		t.Fatalf("Impact: unexpected error: %v", err)
	}

	var found bool
	for _, loc := range got.Affected {
		if loc.Name == "Caller" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Impact(Handle, depth=1).Affected = %+v, want Caller present (dispatch traversal through interface I)", got.Affected)
	}
	// self (A.Handle) + Caller = 2 distinct visited nodes.
	if got.NodeCount != 2 {
		t.Errorf("NodeCount = %d, want 2 (A.Handle self + Caller via dispatch)", got.NodeCount)
	}
}

// traverseFixtureTargetFile and traverseFixtureTargetTestFile are seeded
// into the *copied* gofixture's pkga package before indexing (the
// checked-in testdata tree has no _test.go file, and this plan's Wave-3
// isolation forbids editing engine_test.go or the checked-in fixture —
// 03-04-PLAN.md "Wave-3 test isolation"). They give TestAffected a real
// test->symbol calls edge to derive structural correctness from (D-07).
//
// Target is called directly (an unqualified intra-package call) rather
// than via a method call on a local variable — internal/indexer's
// resolver does not track local-variable types (STATE.md Phase 2
// decision: "no local-variable-type-tracking logic implemented"), so a
// method call like `w.Rename(...)` on a locally-constructed receiver
// produces no resolved `calls` edge at all. A brand-new function in its
// own file also keeps this fixture addition fully isolated from
// TestCallersCallees/TestImpact's Alpha/helper/Run assertions.
const traverseFixtureTargetFile = `package pkga

// Target is a fixture-only function that TestTarget calls directly, so
// TestAffected has a real test->symbol calls edge to derive from (D-07).
func Target() int {
	return 42
}
`

const traverseFixtureTargetTestFile = `package pkga

import "testing"

// TestTarget calls Target directly (unqualified intra-package call, the
// same resolvable call shape as Alpha->helper) so TestAffected has a
// known test->symbol calls edge to derive impacted-test-file structural
// correctness from (D-07).
func TestTarget(t *testing.T) {
	_ = Target()
}
`

// traverseFixture copies gofixture, seeds target.go/target_test.go
// (above), indexes the result via the shared indexFixture harness
// (engine_test.go, reused at runtime, not modified), and opens an
// Engine on it. Every traverse_test.go case shares this deterministic
// call topology:
//
//	main.main -> pkgb.Run -> pkga.Alpha -> pkga.helper
//	                       -> pkga.Widget.Describe (unresolved — local-var method call)
//	pkga.TestTarget -> pkga.Target
func traverseFixture(t *testing.T) *Engine {
	t.Helper()

	dir := copyFixture(t)
	targetFile := filepath.Join(dir, "pkga", "target.go")
	if err := os.WriteFile(targetFile, []byte(traverseFixtureTargetFile), 0o644); err != nil {
		t.Fatalf("seed target.go: %v", err)
	}
	targetTestFile := filepath.Join(dir, "pkga", "target_test.go")
	if err := os.WriteFile(targetTestFile, []byte(traverseFixtureTargetTestFile), 0o644); err != nil {
		t.Fatalf("seed target_test.go: %v", err)
	}
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine
}

// TestCallersCallees pins forward (IterateEdges(src) directly, no scan)
// vs. reverse (BuildReverseAdjacency, D-04) traversal semantics against
// the fixture's deterministic call chain: main -> Run -> Alpha -> helper
// (plus Run -> Widget.Describe).
func TestCallersCallees(t *testing.T) {
	engine := traverseFixture(t)

	t.Run("Callees(Alpha) returns Alpha's forward call target", func(t *testing.T) {
		got, err := engine.Callees("Alpha", 0)
		if err != nil {
			t.Fatalf("Callees: unexpected error: %v", err)
		}
		if got.Symbol != "Alpha" {
			t.Fatalf("Symbol: got %q, want %q", got.Symbol, "Alpha")
		}
		if len(got.Callees) != 1 || got.Callees[0].Name != "helper" {
			t.Fatalf("Callees: got %+v, want a single entry named helper", got.Callees)
		}
		if got.Callees[0].Kind != "function" || got.Callees[0].FilePath != "pkga/pkga.go" {
			t.Fatalf("Callees[0]: got %+v, want kind=function filePath=pkga/pkga.go", got.Callees[0])
		}
	})

	t.Run("Callers(Alpha) returns Alpha's reverse caller", func(t *testing.T) {
		got, err := engine.Callers("Alpha", 0)
		if err != nil {
			t.Fatalf("Callers: unexpected error: %v", err)
		}
		if got.Symbol != "Alpha" {
			t.Fatalf("Symbol: got %q, want %q", got.Symbol, "Alpha")
		}
		if len(got.Callers) != 1 || got.Callers[0].Name != "Run" {
			t.Fatalf("Callers: got %+v, want a single entry named Run", got.Callers)
		}
	})

	t.Run("Callers(helper) returns Alpha only", func(t *testing.T) {
		got, err := engine.Callers("helper", 0)
		if err != nil {
			t.Fatalf("Callers: unexpected error: %v", err)
		}
		if len(got.Callers) != 1 || got.Callers[0].Name != "Alpha" {
			t.Fatalf("Callers: got %+v, want a single entry named Alpha", got.Callers)
		}
	})

	t.Run("limit caps the returned set", func(t *testing.T) {
		got, err := engine.Callees("Alpha", 0)
		if err != nil {
			t.Fatalf("Callees: unexpected error: %v", err)
		}
		full := len(got.Callees)
		if full == 0 {
			t.Fatal("Callees: fixture setup produced zero callees, cannot test limit")
		}
		limited, err := engine.Callees("Alpha", 1)
		if err != nil {
			t.Fatalf("Callees with limit: unexpected error: %v", err)
		}
		if len(limited.Callees) != 1 {
			t.Fatalf("Callees with limit=1: got %d entries, want 1", len(limited.Callees))
		}
	})
}

// TestImpact pins the depth-bounded reverse-BFS arithmetic. The
// fixture's reverse chain from helper is: helper <- Alpha <- Run <- main
// (three hops). The counting rule under test — nodeCount = distinct
// visited nodes including the symbol itself, edgeCount = reverse edges
// inspected while expanding each depth's frontier — is cross-checked
// against testdata/golden/corpus/weft-go/impact.json's own arithmetic:
// symbol="mergeStyle" depth=2 nodeCount=5 edgeCount=4 there decomposes
// as 3 direct callers of mergeStyle (hop 1) + 1 second-hop caller of
// newFinishReconcileCmd (hop 2) = 4 edges traversed, 5 nodes visited
// including mergeStyle itself — the same "frontier expansion" semantics
// verified here against our own deterministic fixture topology (D-07a:
// no golden oracle for this exact custom graph, but the counting *rule*
// is the golden-verified one).
func TestImpact(t *testing.T) {
	engine := traverseFixture(t)

	t.Run("depth=2 walks two hops of reverse calls", func(t *testing.T) {
		got, err := engine.Impact("helper", 2)
		if err != nil {
			t.Fatalf("Impact: unexpected error: %v", err)
		}
		if got.Symbol != "helper" || got.Depth != 2 {
			t.Fatalf("Symbol/Depth: got %q/%d, want helper/2", got.Symbol, got.Depth)
		}
		// helper (self) + Alpha (hop 1) + Run (hop 2) = 3 nodes.
		if got.NodeCount != 3 {
			t.Fatalf("NodeCount: got %d, want 3 (helper, Alpha, Run)", got.NodeCount)
		}
		// Alpha->helper (hop 1) + Run->Alpha (hop 2) = 2 edges.
		if got.EdgeCount != 2 {
			t.Fatalf("EdgeCount: got %d, want 2", got.EdgeCount)
		}
		if len(got.Affected) != 3 || got.Affected[0].Name != "helper" {
			t.Fatalf("Affected: got %+v, want helper first", got.Affected)
		}
	})

	t.Run("depth=1 stops after the first hop", func(t *testing.T) {
		got, err := engine.Impact("helper", 1)
		if err != nil {
			t.Fatalf("Impact: unexpected error: %v", err)
		}
		if got.NodeCount != 2 {
			t.Fatalf("NodeCount: got %d, want 2 (helper, Alpha)", got.NodeCount)
		}
		if got.EdgeCount != 1 {
			t.Fatalf("EdgeCount: got %d, want 1", got.EdgeCount)
		}
	})

	t.Run("absurdly large depth is clamped, not unbounded", func(t *testing.T) {
		got, err := engine.Impact("helper", 999999)
		if err != nil {
			t.Fatalf("Impact: unexpected error: %v", err)
		}
		if got.Depth != MaxDepth {
			t.Fatalf("Depth: got %d, want clamped MaxDepth=%d", got.Depth, MaxDepth)
		}
		// The fixture's whole reverse chain from helper terminates at
		// main (helper<-Alpha<-Run<-main, 3 hops); clamping to
		// MaxDepth=50 must not visit more nodes than the graph actually
		// has on that path — BFS naturally stops once the frontier is
		// empty, well short of the clamp ceiling.
		if got.NodeCount != 4 {
			t.Fatalf("NodeCount: got %d, want 4 (helper, Alpha, Run, main)", got.NodeCount)
		}
		if got.EdgeCount != 3 {
			t.Fatalf("EdgeCount: got %d, want 3", got.EdgeCount)
		}
	})

	// WR-02: a negative --depth is rejected outright, consistent with
	// --limit/--max-files/files' --depth, instead of silently falling
	// back to defaultDepth as clampDepth alone would do.
	t.Run("negative depth is rejected, not silently defaulted", func(t *testing.T) {
		if _, err := engine.Impact("helper", -1); err == nil {
			t.Fatal("Impact with depth=-1: expected error, got nil")
		}
	})
}

// TestClampAffectedDepth pins SURF-04/D-05: affected has its OWN
// default (5), deliberately distinct from impact's clampDepth default
// (2, see TestClampDepth in engine_test.go) — a naive reuse of
// clampDepth would silently apply impact's smaller default to affected.
// Both share the same MaxDepth=50 ceiling.
func TestClampAffectedDepth(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"non-positive uses affected's own default (5, not impact's 2)", 0, 5},
		{"negative uses affected's own default", -5, 5},
		{"explicit small depth preserved", 1, 1},
		{"in-range passes through", 10, 10},
		{"at ceiling passes through", MaxDepth, MaxDepth},
		{"above ceiling clamps to MaxDepth", MaxDepth + 1, MaxDepth},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampAffectedDepth(tc.in); got != tc.want {
				t.Fatalf("clampAffectedDepth(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestAffected pins the D-07 query-time derivation: no persisted
// test-coverage edge, just reverse `calls` edges from a changed file's
// symbols filtered by the _test.go/Test*/Benchmark* heuristic. There is
// no golden oracle for this command (D-07a) — assert structural
// correctness against the seeded TestTarget -> Target call. SURF-04:
// Affected now takes an explicit depth (a single hop is enough to reach
// TestTarget in this fixture's topology).
func TestAffected(t *testing.T) {
	engine := traverseFixture(t)

	got, err := engine.Affected([]string{"pkga/target.go"}, 1)
	if err != nil {
		t.Fatalf("Affected: unexpected error: %v", err)
	}

	if len(got.Files) != 1 || got.Files[0] != "pkga/target.go" {
		t.Fatalf("Files: got %+v, want [pkga/target.go]", got.Files)
	}

	var found bool
	for _, loc := range got.AffectedTests {
		if loc.Name == "TestTarget" {
			found = true
			if loc.FilePath != "pkga/target_test.go" {
				t.Fatalf("TestTarget.FilePath: got %q, want pkga/target_test.go", loc.FilePath)
			}
		}
		// Non-test callers must never leak into the heuristic-filtered
		// result — only names matching the _test.go/Test*/Benchmark*
		// heuristic belong in AffectedTests.
		if loc.Name == "Run" || loc.Name == "Alpha" {
			t.Fatalf("AffectedTests unexpectedly includes non-test caller %q", loc.Name)
		}
	}
	if !found {
		t.Fatalf("AffectedTests: got %+v, want TestTarget present", got.AffectedTests)
	}
}

// TestAffectedNegativeDepthRejected mirrors Impact's WR-02 negative-depth
// rejection contract for Affected.
func TestAffectedNegativeDepthRejected(t *testing.T) {
	engine := traverseFixture(t)
	if _, err := engine.Affected([]string{"pkga/target.go"}, -1); err == nil {
		t.Fatal("Affected with depth=-1: expected error, got nil")
	}
}

// TestAffectedEmptyFilesReturnsEmptyResultNoError pins the empty-input
// contract: an empty files slice seeds no BFS frontier at all — no
// error, just an empty AffectedTests.
func TestAffectedEmptyFilesReturnsEmptyResultNoError(t *testing.T) {
	engine := traverseFixture(t)

	got, err := engine.Affected(nil, 2)
	if err != nil {
		t.Fatalf("Affected with empty files: unexpected error: %v", err)
	}
	if len(got.AffectedTests) != 0 {
		t.Fatalf("Affected with empty files: got %+v, want no affected tests", got.AffectedTests)
	}
}

// affectedDepthFixtureNodesEdges builds a three-hop reverse chain —
// Target (the changed file's symbol) <- NonTestCaller <- TestSomething
// <- TestGrand — to prove Affected's depth-bounded BFS and TS
// test-files-as-leaves pruning (SURF-04/D-05, RESEARCH Pitfall 2):
// TestSomething is reachable only at depth>=2 (through the non-test
// intermediary), and — because a test dependent is a LEAF — its own
// dependent TestGrand must never surface at any depth.
func affectedDepthFixtureNodesEdges() (map[string]*schema.Node, []*schema.Edge) {
	nodes := map[string]*schema.Node{
		"target":        {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target", FilePath: "changed.go"},
		"nonTestCaller": {Id: "nonTestCaller", Kind: "function", Name: "NonTestCaller", QualifiedName: "NonTestCaller", FilePath: "other.go"},
		"testSomething": {Id: "testSomething", Kind: "function", Name: "TestSomething", QualifiedName: "TestSomething", FilePath: "other_test.go"},
		"testGrand":     {Id: "testGrand", Kind: "function", Name: "TestGrand", QualifiedName: "TestGrand", FilePath: "other_test.go"},
	}
	edges := []*schema.Edge{
		{Source: "nonTestCaller", Target: "target", Kind: goextract.RefKindCalls},
		{Source: "testSomething", Target: "nonTestCaller", Kind: goextract.RefKindCalls},
		{Source: "testGrand", Target: "testSomething", Kind: goextract.RefKindCalls},
	}
	return nodes, edges
}

// TestAffectedDepthBFSWithTestLeafPruning is this plan's core behavioral
// proof (SURF-04): depth genuinely bounds BFS expansion, and a test
// dependent is a leaf — recorded but never expanded.
func TestAffectedDepthBFSWithTestLeafPruning(t *testing.T) {
	t.Run("depth=1 finds only the direct non-test dependent, no test yet", func(t *testing.T) {
		nodes, edges := affectedDepthFixtureNodesEdges()
		e := New(&traverseFakeReader{nodes: nodes, edges: edges})

		got, err := e.Affected([]string{"changed.go"}, 1)
		if err != nil {
			t.Fatalf("Affected: unexpected error: %v", err)
		}
		if len(got.AffectedTests) != 0 {
			t.Fatalf("AffectedTests at depth=1: got %+v, want none (TestSomething is 2 hops away)", got.AffectedTests)
		}
	})

	t.Run("depth=2 finds the test reached through the non-test intermediary", func(t *testing.T) {
		nodes, edges := affectedDepthFixtureNodesEdges()
		e := New(&traverseFakeReader{nodes: nodes, edges: edges})

		got, err := e.Affected([]string{"changed.go"}, 2)
		if err != nil {
			t.Fatalf("Affected: unexpected error: %v", err)
		}
		var found bool
		for _, loc := range got.AffectedTests {
			if loc.Name == "TestSomething" {
				found = true
			}
			if loc.Name == "NonTestCaller" {
				t.Fatalf("AffectedTests unexpectedly includes non-test dependent %q", loc.Name)
			}
		}
		if !found {
			t.Fatalf("AffectedTests at depth=2: got %+v, want TestSomething present", got.AffectedTests)
		}
	})

	t.Run("a test dependent is a leaf — its own dependents are never pulled in, even at a much larger depth", func(t *testing.T) {
		nodes, edges := affectedDepthFixtureNodesEdges()
		e := New(&traverseFakeReader{nodes: nodes, edges: edges})

		got, err := e.Affected([]string{"changed.go"}, 10)
		if err != nil {
			t.Fatalf("Affected: unexpected error: %v", err)
		}
		for _, loc := range got.AffectedTests {
			if loc.Name == "TestGrand" {
				t.Fatalf("AffectedTests unexpectedly includes %q — TestSomething (a test leaf) must not be expanded", loc.Name)
			}
		}
	})

	t.Run("depth=0 uses defaultAffectedDepth (5), reaching the test two hops away", func(t *testing.T) {
		nodes, edges := affectedDepthFixtureNodesEdges()
		e := New(&traverseFakeReader{nodes: nodes, edges: edges})

		got, err := e.Affected([]string{"changed.go"}, 0)
		if err != nil {
			t.Fatalf("Affected: unexpected error: %v", err)
		}
		var found bool
		for _, loc := range got.AffectedTests {
			if loc.Name == "TestSomething" {
				found = true
			}
		}
		if !found {
			t.Fatalf("AffectedTests at depth=0 (default 5): got %+v, want TestSomething present", got.AffectedTests)
		}
	})
}

// TestAffectedSkipsDanglingEdgeInsteadOfFailing mirrors
// TestImpactSkipsDanglingEdgeInsteadOfFailing for Affected's BFS (WR-04):
// a dangling reverse-edge source encountered mid-traversal must be
// skipped, not abort the whole call.
func TestAffectedSkipsDanglingEdgeInsteadOfFailing(t *testing.T) {
	nodes := map[string]*schema.Node{
		"target":   {Id: "target", Kind: "function", Name: "Target", QualifiedName: "Target", FilePath: "changed.go"},
		"testLive": {Id: "testLive", Kind: "function", Name: "TestLive", QualifiedName: "TestLive", FilePath: "other_test.go"},
	}
	edges := []*schema.Edge{
		{Source: "testLive", Target: "target", Kind: goextract.RefKindCalls},
		{Source: "missing", Target: "target", Kind: goextract.RefKindCalls}, // dangling
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})

	got, err := e.Affected([]string{"changed.go"}, 2)
	if err != nil {
		t.Fatalf("Affected: unexpected error from a dangling edge, want graceful skip: %v", err)
	}
	if len(got.AffectedTests) != 1 || got.AffectedTests[0].Name != "TestLive" {
		t.Fatalf("Affected: got %+v, want exactly [TestLive] (dangling edge skipped)", got.AffectedTests)
	}
}

// assertJSONArrayNotNull fails t if data's top-level object key marshaled
// as JSON null — WR-01: a zero-match array field must marshal as [], not
// null, so a JSON consumer that assumes an array field is always an array
// (result.callers.map(...), etc.) never crashes on the zero-match case.
func assertJSONArrayNotNull(t *testing.T, data []byte, key string) {
	t.Helper()

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal JSON: %v\n%s", err, data)
	}
	raw, ok := m[key]
	if !ok {
		t.Fatalf("missing top-level key %q in %s", key, data)
	}
	if string(raw) == "null" {
		t.Fatalf("key %q marshaled as null, want []: %s", key, data)
	}
}

// TestZeroMatchJSONShapesAreEmptyArraysNotNull pins WR-01: Callees,
// Callers, and Affected all build their array field via `var x []T` and
// only ever append — on a genuine zero-match result the slice stays nil
// and previously marshaled as JSON null instead of [].
func TestZeroMatchJSONShapesAreEmptyArraysNotNull(t *testing.T) {
	engine := traverseFixture(t)

	t.Run("Callees zero-match", func(t *testing.T) {
		// Target (fixture-seeded, traverse_test.go) calls nothing itself.
		got, err := engine.Callees("Target", 0)
		if err != nil {
			t.Fatalf("Callees: unexpected error: %v", err)
		}
		if len(got.Callees) != 0 {
			t.Fatalf("Callees(Target): got %+v, want zero callees (fixture invariant)", got.Callees)
		}
		data, err := MarshalCalleesJSON(got)
		if err != nil {
			t.Fatalf("MarshalCalleesJSON: unexpected error: %v", err)
		}
		assertJSONArrayNotNull(t, data, "callees")
	})

	t.Run("Callers zero-match", func(t *testing.T) {
		// TestTarget (fixture-seeded) is called by nothing in the fixture.
		got, err := engine.Callers("TestTarget", 0)
		if err != nil {
			t.Fatalf("Callers: unexpected error: %v", err)
		}
		if len(got.Callers) != 0 {
			t.Fatalf("Callers(TestTarget): got %+v, want zero callers (fixture invariant)", got.Callers)
		}
		data, err := MarshalCallersJSON(got)
		if err != nil {
			t.Fatalf("MarshalCallersJSON: unexpected error: %v", err)
		}
		assertJSONArrayNotNull(t, data, "callers")
	})

	t.Run("Affected zero-match", func(t *testing.T) {
		got, err := engine.Affected([]string{"nonexistent-file-with-no-callers.go"}, 2)
		if err != nil {
			t.Fatalf("Affected: unexpected error: %v", err)
		}
		if len(got.AffectedTests) != 0 {
			t.Fatalf("Affected(nonexistent file): got %+v, want zero affected tests", got.AffectedTests)
		}
		data, err := MarshalAffectedJSON(got)
		if err != nil {
			t.Fatalf("MarshalAffectedJSON: unexpected error: %v", err)
		}
		assertJSONArrayNotNull(t, data, "affectedTests")
	})
}
