package migrate

import (
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/migrate/migratetest"
)

// runTarget returns a not-yet-existing .codegraph target path inside a
// fresh t.TempDir(), so checkWritableDir's parent-writability probe and
// partialDir's same-parent placement both have a real, writable directory
// to work with.
func runTarget(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), ".codegraph")
}

// openTargetStore opens the migrated store's Pebble subdirectory
// (target/store — the same .codegraph/store/ layout convention
// internal/cli's storeDirName establishes, D-01b) for read-only
// assertions, closing it on test cleanup.
func openTargetStore(t *testing.T, target string) graphstore.GraphStore {
	t.Helper()
	storeDir := filepath.Join(target, "store")
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open(%s): %v", storeDir, err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	return store
}

// TestRun_Happy proves MIGR-01's core promise: one Run call converts a
// happy-path TS index into a healthy new-format store whose counts
// reconcile against the source, with the x/ file index populated (D-04)
// and a version-stamped, healthy Meta carrying the D-01 first-sync note.
func TestRun_Happy(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	wantNodes, err := src.CountRows("nodes")
	if err != nil {
		t.Fatalf("CountRows(nodes): %v", err)
	}
	wantFiles, err := src.CountRows("files")
	if err != nil {
		t.Fatalf("CountRows(files): %v", err)
	}
	wantEdges, err := src.CountDistinctEdges()
	if err != nil {
		t.Fatalf("CountDistinctEdges: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("src.Close: %v", err)
	}

	result, err := Run(dbPath, target, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Resumed {
		t.Error("a fresh run must not report Resumed=true")
	}
	if result.Nodes != wantNodes {
		t.Errorf("Result.Nodes = %d, want %d", result.Nodes, wantNodes)
	}
	if result.Files != wantFiles {
		t.Errorf("Result.Files = %d, want %d", result.Files, wantFiles)
	}
	if result.Edges != wantEdges {
		t.Errorf("Result.Edges = %d, want %d", result.Edges, wantEdges)
	}
	if result.HealthMessage == "" {
		t.Error("Result.HealthMessage must document the first-sync full-reindex behavior")
	}

	store := openTargetStore(t, target)
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHealthy() {
		t.Error("Meta.Healthy = false, want true (validate passed)")
	}
	if !meta.GetHasFileIndex() {
		t.Error("Meta.HasFileIndex = false, want true (D-04)")
	}
	if meta.GetSchemaVersion() == 0 {
		t.Error("Meta.SchemaVersion must be stamped via schema.NewMeta()")
	}
	if meta.GetNodeCount() != wantNodes || meta.GetEdgeCount() != wantEdges {
		t.Errorf("Meta counts = (nodes=%d, edges=%d), want (%d, %d)", meta.GetNodeCount(), meta.GetEdgeCount(), wantNodes, wantEdges)
	}
}

// TestRun_NodesBeforeEdgesOwnerPath proves D-04: after migration, a node's
// owning file enumerates that node via IterateFileIndex, and a
// file:-source contains edge migrated without error despite carrying an
// empty ownerPath (isFileEndpoint exemption — no x/ entry, no dangling
// false-positive).
func TestRun_NodesBeforeEdgesOwnerPath(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	if _, err := Run(dbPath, target, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store := openTargetStore(t, target)
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	nit, err := snap.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	var samplePath string
	for nit.Next() {
		if p := nit.Node().GetFilePath(); p != "" {
			samplePath = p
			break
		}
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("iterate nodes: %v", err)
	}
	_ = nit.Close()
	if samplePath == "" {
		t.Fatal("expected at least one node with a non-empty file_path in the fixture")
	}

	fit, err := snap.IterateFileIndex(samplePath)
	if err != nil {
		t.Fatalf("IterateFileIndex(%s): %v", samplePath, err)
	}
	found := false
	for fit.Next() {
		if fit.Entry().IsNode {
			found = true
		}
	}
	if err := fit.Err(); err != nil {
		t.Fatalf("iterate file index: %v", err)
	}
	_ = fit.Close()
	if !found {
		t.Errorf("expected at least one node entry in %s's x/ file index", samplePath)
	}
}

// TestRun_SourceUnmodified proves D-08: the source .db's bytes are
// byte-identical, and no -wal/-shm sidecar exists, after a full Run.
func TestRun_SourceUnmodified(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	before, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash before: %v", err)
	}

	if _, err := Run(dbPath, target, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	after, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash after: %v", err)
	}
	if before != after {
		t.Error("source db bytes changed after Run")
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if exists(dbPath + suffix) {
			t.Errorf("unexpected sidecar file %s%s after Run", dbPath, suffix)
		}
	}
}

// TestRun_Resume proves MIGR-02/D-06: an interruption (simulated via the
// test-only testStopAfterBatch seam) right after the files table commits
// leaves NO swapped target; calling Run again against the same target
// resumes from the durable cursor (skipping the already-committed files
// table) and completes with the same final counts an uninterrupted run
// would produce.
func TestRun_Resume(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	wantNodes, err := src.CountRows("nodes")
	if err != nil {
		t.Fatalf("CountRows(nodes): %v", err)
	}
	wantFiles, err := src.CountRows("files")
	if err != nil {
		t.Fatalf("CountRows(files): %v", err)
	}
	wantEdges, err := src.CountDistinctEdges()
	if err != nil {
		t.Fatalf("CountDistinctEdges: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("src.Close: %v", err)
	}

	stopped := false
	testStopAfterBatch = func(table string, _ int64) bool {
		if stopped {
			return false
		}
		stopped = true
		return true // interrupt at the very first durable cursor checkpoint
	}
	t.Cleanup(func() { testStopAfterBatch = nil })

	_, err = Run(dbPath, target, Options{})
	if err == nil {
		t.Fatal("expected the first Run to return the injected interruption error")
	}

	if exists(target) {
		t.Fatalf("target %s must not exist after an interrupted run (no swap should have happened)", target)
	}

	testStopAfterBatch = nil // second call runs to completion

	result, err := Run(dbPath, target, Options{})
	if err != nil {
		t.Fatalf("resume Run: %v", err)
	}
	if !result.Resumed {
		t.Error("Result.Resumed = false, want true (second call resumed the in_progress cursor)")
	}
	if result.Nodes != wantNodes {
		t.Errorf("resumed Result.Nodes = %d, want %d", result.Nodes, wantNodes)
	}
	if result.Files != wantFiles {
		t.Errorf("resumed Result.Files = %d, want %d", result.Files, wantFiles)
	}
	if result.Edges != wantEdges {
		t.Errorf("resumed Result.Edges = %d, want %d", result.Edges, wantEdges)
	}

	store := openTargetStore(t, target)
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()
	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHealthy() {
		t.Error("resumed Meta.Healthy = false, want true")
	}
}

// TestRun_AgedDB proves D-09.4: migrating a source missing later-added
// columns (nodes.return_type, edges.provenance) completes healthy, with
// the migrated records carrying the proto zero value for the absent
// columns.
func TestRun_AgedDB(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantAged)
	target := runTarget(t)

	if _, err := Run(dbPath, target, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store := openTargetStore(t, target)
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHealthy() {
		t.Error("aged-DB Meta.Healthy = false, want true")
	}

	nit, err := snap.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	sawNode := false
	for nit.Next() {
		sawNode = true
		if got := nit.Node().GetReturnType(); got != "" {
			t.Errorf("aged node %s ReturnType = %q, want \"\" (column absent)", nit.Node().GetId(), got)
		}
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("iterate nodes: %v", err)
	}
	_ = nit.Close()
	if !sawNode {
		t.Fatal("expected at least one migrated node")
	}

	eit, err := snap.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	sawEdge := false
	for eit.Next() {
		sawEdge = true
		if got := eit.Edge().GetProvenance(); got != "" {
			t.Errorf("aged edge %s->%s Provenance = %q, want \"\" (column absent)", eit.Edge().GetSource(), eit.Edge().GetTarget(), got)
		}
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("iterate edges: %v", err)
	}
	_ = eit.Close()
	if !sawEdge {
		t.Fatal("expected at least one migrated edge")
	}
}

// TestRun_DanglingFailsLoud proves D-09.2's default policy: a source with a
// genuinely dangling (non-file:) edge fails Run loudly, performs no swap
// (the target is absent), and leaves the source untouched.
func TestRun_DanglingFailsLoud(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantDangling)
	target := runTarget(t)

	before, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash before: %v", err)
	}

	_, err = Run(dbPath, target, Options{})
	if err == nil {
		t.Fatal("expected Run to fail loud on a dangling edge by default")
	}

	if exists(target) {
		t.Errorf("target %s must not exist after a failed (unvalidated) run", target)
	}

	after, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash after: %v", err)
	}
	if before != after {
		t.Error("source db bytes changed after a failed run")
	}

	partial := filepath.Join(filepath.Dir(mustAbs(t, target)), partialStoreName)
	if !exists(partial) {
		t.Errorf("expected the partial store %s to remain present for inspection/resume after a validation failure", partial)
	}
}

// TestRun_DropDangling proves the opt-in --drop-dangling path: Run
// completes healthy, the dangling edge is absent from the migrated store,
// and the Report records the drop.
func TestRun_DropDangling(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantDangling)
	target := runTarget(t)

	result, err := Run(dbPath, target, Options{DropDangling: true})
	if err != nil {
		t.Fatalf("Run with DropDangling: %v", err)
	}
	if result.Report.Dropped == 0 {
		t.Error("Result.Report.Dropped = 0, want at least 1")
	}
	if len(result.Report.Dangling) == 0 {
		t.Error("Result.Report.Dangling should record the edge(s) found before dropping")
	}

	store := openTargetStore(t, target)

	// Re-scan the swapped-in store directly (mirrors validate_test.go's
	// TestValidate_DropDangling pattern) to prove the drop actually took:
	// zero dangling edges remain.
	var rescan Report
	if err := scanDangling(store, false, &rescan); err != nil {
		t.Fatalf("re-scan after drop: %v", err)
	}
	if len(rescan.Dangling) != 0 {
		t.Errorf("re-scan found %d dangling edges after drop, want 0: %+v", len(rescan.Dangling), rescan.Dangling)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHealthy() {
		t.Error("drop-dangling Meta.Healthy = false, want true")
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("filepath.Abs(%s): %v", p, err)
	}
	return abs
}
