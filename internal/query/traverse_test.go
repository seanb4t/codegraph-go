package query

import (
	"os"
	"path/filepath"
	"testing"
)

// traverseFixtureTestFile is seeded into the *copied* gofixture's pkga
// package before indexing (the checked-in testdata tree has no _test.go
// file, and this plan's Wave-3 isolation forbids editing engine_test.go
// or the checked-in fixture — 03-04-PLAN.md "Wave-3 test isolation").
// It gives TestAffected a real test->symbol calls edge to derive
// structural correctness from (D-07): Widget.Rename is otherwise
// uncalled anywhere in the base fixture, so TestWidgetRename is the only
// caller of Rename once seeded, keeping this fixture addition isolated
// from TestCallersCallees/TestImpact's assertions (those exercise
// Alpha/helper/Run, never Rename).
const traverseFixtureTestFile = `package pkga

import "testing"

// TestWidgetRename exercises Widget.Rename so TestAffected has a known
// test->symbol calls edge to derive impacted-test-file structural
// correctness from (D-07) — Rename has no other caller in the fixture.
func TestWidgetRename(t *testing.T) {
	w := &Widget{}
	w.Rename("x")
}
`

// traverseFixture copies gofixture, seeds pkga_test.go (above), indexes
// the result via the shared indexFixture harness (engine_test.go,
// reused at runtime, not modified), and opens an Engine on it. Every
// traverse_test.go case shares this deterministic call topology:
//
//	main.main -> pkgb.Run -> pkga.Alpha -> pkga.helper
//	                       -> pkga.Widget.Describe
//	pkga_test.TestWidgetRename -> pkga.Widget.Rename
func traverseFixture(t *testing.T) *Engine {
	t.Helper()

	dir := copyFixture(t)
	testFile := filepath.Join(dir, "pkga", "pkga_test.go")
	if err := os.WriteFile(testFile, []byte(traverseFixtureTestFile), 0o644); err != nil {
		t.Fatalf("seed pkga_test.go: %v", err)
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
// vs. reverse (buildReverseAdjacency, D-04) traversal semantics against
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
}

// TestAffected pins the D-07 query-time derivation: no persisted
// test-coverage edge, just reverse `calls` edges from a changed file's
// symbols filtered by the _test.go/Test*/Benchmark* heuristic. There is
// no golden oracle for this command (D-07a) — assert structural
// correctness against the seeded TestWidgetRename -> Widget.Rename call.
func TestAffected(t *testing.T) {
	engine := traverseFixture(t)

	got, err := engine.Affected([]string{"pkga/pkga.go"})
	if err != nil {
		t.Fatalf("Affected: unexpected error: %v", err)
	}

	if len(got.Files) != 1 || got.Files[0] != "pkga/pkga.go" {
		t.Fatalf("Files: got %+v, want [pkga/pkga.go]", got.Files)
	}

	var found bool
	for _, loc := range got.AffectedTests {
		if loc.Name == "TestWidgetRename" {
			found = true
			if loc.FilePath != "pkga/pkga_test.go" {
				t.Fatalf("TestWidgetRename.FilePath: got %q, want pkga/pkga_test.go", loc.FilePath)
			}
		}
		// Non-test callers (e.g. Run, which calls Widget.Describe — also
		// defined via pkga.go's Widget methods) must never leak into the
		// heuristic-filtered result.
		if loc.Name == "Run" {
			t.Fatalf("AffectedTests unexpectedly includes non-test caller %q", loc.Name)
		}
	}
	if !found {
		t.Fatalf("AffectedTests: got %+v, want TestWidgetRename present", got.AffectedTests)
	}
}
