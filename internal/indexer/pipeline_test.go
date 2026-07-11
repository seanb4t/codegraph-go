package indexer

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
)

// TestPipelineRun proves Run orchestrates Discover -> Extract -> Resolve
// into a single from-scratch, committed graph, with Stats matching the
// counts stamped onto Meta (RES-01, D-04).
func TestPipelineRun(t *testing.T) {
	storeDir := t.TempDir()

	stats, err := Run(fixtureRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Files == 0 {
		t.Error("Stats.Files = 0, want > 0")
	}
	if stats.Nodes == 0 {
		t.Error("Stats.Nodes = 0, want > 0")
	}
	if stats.Edges == 0 {
		t.Error("Stats.Edges = 0, want > 0")
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()

	alphaID := nodeid.NodeID(goextract.KindFunction, "Alpha", "pkga/pkga.go")
	if _, err := r.GetNode(alphaID); err != nil {
		t.Errorf("GetNode(pkga.Alpha) = %v, want nil (node present in committed store)", err)
	}

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "pkgb/pkgb.go")
	it, err := r.IterateEdges(runID)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()

	found := false
	for it.Next() {
		if it.Edge().Kind == "calls" && it.Edge().Target == alphaID {
			found = true
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateEdges error: %v", err)
	}
	if !found {
		t.Error("expected calls edge pkgb.Run -> pkga.Alpha in the committed store")
	}

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if meta.SchemaVersion != 1 {
		t.Errorf("Meta.SchemaVersion = %d, want 1", meta.SchemaVersion)
	}
	if meta.NodeCount != int64(stats.Nodes) {
		t.Errorf("Meta.NodeCount = %d, want %d (Stats.Nodes)", meta.NodeCount, stats.Nodes)
	}
	if meta.EdgeCount != int64(stats.Edges) {
		t.Errorf("Meta.EdgeCount = %d, want %d (Stats.Edges)", meta.EdgeCount, stats.Edges)
	}
}

// errInjectedResolveFailure is the sentinel a stub resolveFunc returns to
// simulate a Pass-2 failure without depending on a real Resolve error
// condition.
var errInjectedResolveFailure = errors.New("indexer: injected resolve failure for test")

// TestPipelineRun_ClosesStoreOnResolveError proves the GraphStore is
// opened exactly once and Closed even when Pass 2 fails (T-02-11): if Run
// leaked the open store, reopening the same directory afterward would fail
// with a Pebble lock-contention error.
func TestPipelineRun_ClosesStoreOnResolveError(t *testing.T) {
	storeDir := t.TempDir()
	failingResolve := func(store graphstore.GraphStore, results []goextract.FileResult, modulePath string) (int, error) {
		return 0, errInjectedResolveFailure
	}

	_, err := run(fixtureRoot, storeDir, Options{}, failingResolve)
	if !errors.Is(err, errInjectedResolveFailure) {
		t.Fatalf("run error = %v, want %v", err, errInjectedResolveFailure)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open after failed run: %v (store was not Closed)", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestPipelineRun_DefaultsWorkersToNumCPU proves Options.Workers <= 0
// still produces a correct run (Extract's own runtime.NumCPU() default,
// D-04) — Run neither panics nor short-circuits on the zero value.
func TestPipelineRun_DefaultsWorkersToNumCPU(t *testing.T) {
	storeDir := t.TempDir()

	stats, err := Run(fixtureRoot, storeDir, Options{Workers: 0})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Files == 0 {
		t.Error("Stats.Files = 0, want > 0")
	}
}
