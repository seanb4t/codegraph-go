package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// runAndSnapshot runs the full from-scratch pipeline against repoRoot and
// returns an open graphstore.Reader snapshot, closed automatically at test
// end — the shared setup every route-detection integration test below
// uses (mirrors pipeline_test.go's TestPipelineRun shape).
func runAndSnapshot(t *testing.T, repoRoot string) (graphstore.Reader, Stats) {
	t.Helper()
	storeDir := t.TempDir()

	stats, err := Run(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Run(%s): %v", repoRoot, err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Cleanup(func() { r.Close() })

	return r, stats
}

// collectRouteNodes scans r for every committed goextract.KindRoute node.
func collectRouteNodes(t *testing.T, r graphstore.Reader) []*schema.Node {
	t.Helper()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()

	var routes []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.Kind == goextract.KindRoute {
			routes = append(routes, n)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	return routes
}

// findRouteEdge returns the single "calls" edge sourced from a route node
// whose Metadata["httpMethod"]/["routePath"] match method/path, or nil.
func findRouteEdge(t *testing.T, r graphstore.Reader, routeNodes []*schema.Node, method, path string) *schema.Edge {
	t.Helper()
	for _, rn := range routeNodes {
		it, err := r.IterateEdges(rn.Id)
		if err != nil {
			t.Fatalf("IterateEdges(%s): %v", rn.Id, err)
		}
		for it.Next() {
			e := it.Edge()
			if e.Kind != goextract.RefKindCalls {
				continue
			}
			if e.Metadata["httpMethod"] == method && e.Metadata["routePath"] == path {
				it.Close()
				return e
			}
		}
		it.Close()
	}
	return nil
}

// TestGin_RouteDetectionEndToEnd proves a full Run() over the Gin fixture
// commits route nodes + heuristic "calls" edges to their handlers, opt-in
// gated by the gin-gonic/gin go.mod dependency (D-08/D-09), and that those
// edges surface via query.Callers — the unchanged BuildReverseAdjacency
// traversal (05-PATTERNS.md's zero-query-engine-change requirement).
func TestGin_RouteDetectionEndToEnd(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/gin")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 2 {
		t.Fatalf("route node count = %d, want 2 (GET+POST); got %+v", len(routeNodes), routeNodes)
	}

	getEdge := findRouteEdge(t, r, routeNodes, "GET", "/users/:id")
	if getEdge == nil {
		t.Fatal("no route->handler edge found for GET /users/:id")
	}
	if getEdge.Provenance != "heuristic" {
		t.Errorf("GET route edge Provenance = %q, want heuristic", getEdge.Provenance)
	}
	if getEdge.Metadata["synthesizedBy"] != "gin-route" {
		t.Errorf("GET route edge synthesizedBy = %q, want gin-route", getEdge.Metadata["synthesizedBy"])
	}

	handler, err := r.GetNode(getEdge.Target)
	if err != nil {
		t.Fatalf("GetNode(handler): %v", err)
	}
	if handler.Name != "getUserHandler" {
		t.Errorf("resolved handler Name = %q, want getUserHandler", handler.Name)
	}

	// The route->handler edge is a "calls" edge, so it must surface via
	// the UNCHANGED BuildReverseAdjacency-backed Callers traversal with no
	// query-engine change (05-PATTERNS.md's explicit requirement).
	engine := query.New(r)
	callers, err := engine.Callers("getUserHandler", 0)
	if err != nil {
		t.Fatalf("Callers(getUserHandler): %v", err)
	}
	foundRouteCaller := false
	for _, c := range callers.Callers {
		if c.Kind == goextract.KindRoute {
			foundRouteCaller = true
		}
	}
	if !foundRouteCaller {
		t.Errorf("Callers(getUserHandler) = %+v, want a route-kind caller", callers.Callers)
	}
}

// TestGin_OptInNoDependencyNoRoutes proves an identical Gin-shaped
// call-site pattern produces ZERO route nodes when the go.mod manifest
// does not mention gin-gonic/gin (D-09's opt-in gate — no false-positive
// routes in a non-Gin repo).
func TestGin_OptInNoDependencyNoRoutes(t *testing.T) {
	r, _ := runAndSnapshot(t, "testdata/routesfixture/gonogin")

	routeNodes := collectRouteNodes(t, r)
	if len(routeNodes) != 0 {
		t.Fatalf("route node count = %d, want 0 (no gin dependency in go.mod)", len(routeNodes))
	}
}

// TestRoute_DeterministicRebuild proves indexing the Gin route fixture
// twice from scratch yields byte-identical Export() streams — route
// nodes/edges flow through the SAME collapseEdges/writeGraph path as
// every other edge, so this reuses determinism_test.go's own
// indexAndExport helper unchanged (05-12-PLAN.md's determinism
// acceptance criterion).
func TestRoute_DeterministicRebuild(t *testing.T) {
	first := indexAndExport(t, "testdata/routesfixture/gin")
	second := indexAndExport(t, "testdata/routesfixture/gin")

	if len(first) == 0 {
		t.Fatal("indexAndExport produced an empty stream")
	}
	if string(first) != string(second) {
		reportFirstDiff(t, first, second)
	}
}
