package indexer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

	alphaID := nodeid.DeclID(goextract.KindFunction, "Alpha", "pkga/pkga.go", "()")
	if _, err := r.GetNode(alphaID); err != nil {
		t.Errorf("GetNode(pkga.Alpha) = %v, want nil (node present in committed store)", err)
	}

	runID := nodeid.DeclID(goextract.KindFunction, "Run", "pkgb/pkgb.go", "()")
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

// errInjectedCommitFailure is the error that a test's onCommit hook
// returns to fail a run at a batch commit.
var errInjectedCommitFailure = errors.New("indexer: injected commit failure for test")

// setHook sets a hook variable of the package for the life of the test.
func setHook[T any](t *testing.T, hook *T, value T) {
	t.Helper()
	orig := *hook
	*hook = value
	t.Cleanup(func() { *hook = orig })
}

// TestPipelineRun_ClosesStoreOnCommitError checks that a run that fails
// at a commit closes what it opened and leaves no store behind
// (T-02-11): a later open of the same path works at once.
func TestPipelineRun_ClosesStoreOnCommitError(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "store")
	commits := 0
	setHook(t, &onCommit, func(graphstore.Commit) error {
		commits++
		return errInjectedCommitFailure
	})

	_, err := run(context.Background(), fixtureRoot, storeDir, Options{})
	if !errors.Is(err, errInjectedCommitFailure) {
		t.Fatalf("run error = %v, want %v", err, errInjectedCommitFailure)
	}
	// Positive assertion: the run reached a commit.
	if commits == 0 {
		t.Fatal("the run failed before any commit, so the hook did not cause the failure")
	}
	if _, err := os.Lstat(storeDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Lstat(%s) = %v; want no store after a first index that failed", storeDir, err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open after the failed run: %v (the run left a lock)", err)
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
