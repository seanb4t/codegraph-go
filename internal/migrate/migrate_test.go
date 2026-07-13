package migrate

import (
	"database/sql"
	"os"
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

// TestRun_ClosesSourceBeforeSwap proves CR-01: the read-only source DB handle
// is released BEFORE atomicSwapDir runs. On Windows a directory containing an
// open file handle cannot be renamed, so holding the source (which, for the
// default in-place migration, lives inside the swapped directory) open across
// the swap deterministically breaks the default command. This can't reproduce
// a real Windows sharing violation on the CI host, so it proves the ordering
// invariant structurally via the testBeforeSwap seam.
func TestRun_ClosesSourceBeforeSwap(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	var sawSwap, srcOpenAtSwap bool
	testBeforeSwap = func(src *Source) {
		sawSwap = true
		if src != nil && !src.Closed() {
			srcOpenAtSwap = true
		}
	}
	t.Cleanup(func() { testBeforeSwap = nil })

	if _, err := Run(dbPath, target, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sawSwap {
		t.Fatal("expected the before-swap seam to fire (the happy path must reach atomicSwapDir)")
	}
	if srcOpenAtSwap {
		t.Error("source DB handle was still open when atomicSwapDir ran (CR-01: Windows cannot rename a directory that still contains an open handle)")
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

// TestRun_RefusesWithoutCreatingStore proves WR-03: calling Run with
// Options{Force:false} against a non-empty target that is NOT a prior healthy
// migration (e.g. an in-place .codegraph/ still holding the TS source db)
// refuses loudly AND does not create a pebble store/ directory in the target
// while refusing. Before the fix, the "is this a prior migration?" probe
// called graphstore.Open, which pebble.Open-creates the store/ dir even as it
// refuses — mutating the target (and, for in-place from==to, the source),
// brushing against the D-08 non-destructive guarantee.
func TestRun_RefusesWithoutCreatingStore(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)

	target := runTarget(t)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", target, err)
	}
	// A stray file makes the target non-empty and NOT a recognizable store.
	if err := os.WriteFile(filepath.Join(target, "index.db"), []byte("not a store"), 0o600); err != nil {
		t.Fatalf("seed non-migration target: %v", err)
	}

	if _, err := Run(dbPath, target, Options{}); err == nil {
		t.Fatal("expected Run to refuse overwriting a non-empty, non-migration target")
	}

	storeDir := filepath.Join(target, "store")
	if _, statErr := os.Stat(storeDir); statErr == nil {
		t.Errorf("refusal check created a pebble store at %s (the probe must be read-only, D-08)", storeDir)
	}
}

// buildDanglingUnderFileDB writes a minimal but complete TS-shaped SQLite
// source in which a File record's OWN symbol owns a dangling edge — the exact
// shape WR-01 needs but the shared migratetest fixture cannot express (its
// seeded node file_paths do not correspond to any files-table row, so no File
// record ever carries an x/ edge entry). One file (pkg/a.go), one node
// (func:a in pkg/a.go), and one edge (func:a -> func:missing) whose source
// resolves but whose target is absent: dropping it removes the x/ entry under
// pkg/a.go, so pkg/a.go's File.edge_count must be recomputed post-drop.
func buildDanglingUnderFileDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "dangling-under-file.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	defer db.Close()

	const ddl = `
CREATE TABLE schema_versions (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL, description TEXT);
CREATE TABLE files (path TEXT PRIMARY KEY, content_hash TEXT, language TEXT, size INTEGER, modified_at INTEGER, node_count INTEGER, errors TEXT);
CREATE TABLE nodes (id TEXT PRIMARY KEY, kind TEXT, name TEXT, qualified_name TEXT, file_path TEXT, language TEXT, start_line INTEGER, end_line INTEGER, start_column INTEGER, end_column INTEGER, docstring TEXT, signature TEXT, visibility TEXT, is_exported INTEGER, return_type TEXT);
CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, source TEXT, target TEXT, kind TEXT, line INTEGER, col INTEGER, provenance TEXT, metadata TEXT);
`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("exec ddl: %v", err)
	}
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO schema_versions(version, applied_at, description) VALUES (7, 0, '')`, nil},
		{`INSERT INTO files(path, content_hash, language, size, modified_at, node_count, errors) VALUES ('pkg/a.go', 'h', 'go', 10, 1700000000000, 1, NULL)`, nil},
		{`INSERT INTO nodes(id, kind, name, qualified_name, file_path, language, start_line, end_line, start_column, end_column, docstring, signature, visibility, is_exported, return_type) VALUES ('func:a', 'function', 'A', 'pkg.A', 'pkg/a.go', 'go', 1, 2, 0, 0, NULL, 'func A()', 'public', 1, NULL)`, nil},
		{`INSERT INTO edges(source, target, kind, line, col, provenance, metadata) VALUES ('func:a', 'func:missing', 'calls', 1, 1, NULL, NULL)`, nil},
	}
	for _, s := range stmts {
		if _, err := db.Exec(s.q, s.args...); err != nil {
			t.Fatalf("exec %q: %v", s.q, err)
		}
	}
	return dbPath
}

// TestRun_DropDangling_FileEdgeCountReconciled proves WR-01: after a
// --drop-dangling migration deletes a dangling edge whose source resolved but
// target did not (so its x/ file-index entry under the owning file WAS written
// and then deleted), the owning file's persisted File.edge_count equals the
// actual number of edge entries remaining in that file's x/ index.
// recomputeFileEdgeCounts originally ran only BEFORE validate's drop, leaving
// the owning file's count over-counted by one; the post-drop recompute
// reconciles it.
func TestRun_DropDangling_FileEdgeCountReconciled(t *testing.T) {
	dbPath := buildDanglingUnderFileDB(t)
	target := runTarget(t)

	result, err := Run(dbPath, target, Options{DropDangling: true})
	if err != nil {
		t.Fatalf("Run with DropDangling: %v", err)
	}
	if result.Report.Dropped != 1 {
		t.Fatalf("Report.Dropped = %d, want 1 (the single dangling edge)", result.Report.Dropped)
	}

	store := openTargetStore(t, target)
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	const owner = "pkg/a.go"
	f, err := snap.GetFile(owner)
	if err != nil {
		t.Fatalf("GetFile(%s): %v", owner, err)
	}
	want, err := countFileIndexEdges(snap, owner)
	if err != nil {
		t.Fatalf("countFileIndexEdges(%s): %v", owner, err)
	}
	if want != 0 {
		t.Fatalf("owner %s x/ edge count = %d, want 0 after the only edge was dropped", owner, want)
	}
	if f.GetEdgeCount() != want {
		t.Errorf("file %s: persisted edge_count = %d, want %d (live x/ index count) — recompute did not run after --drop-dangling", owner, f.GetEdgeCount(), want)
	}
}

// TestRun_RecoversInterruptedSwap proves WR-04: a crash between
// atomicSwapDir's two renames leaves the target absent, the validated
// StatusComplete store at the deterministic partial path, and the
// pre-migration original aside at <target>.old. For an in-place (from==to)
// migration the source is now gone (it became <target>.old), so before the fix
// resolveSourceDB hard-errored and the fully-validated migration required a
// manual mv. Run must instead detect the orphaned swap and finish it.
func TestRun_RecoversInterruptedSwap(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	// A full clean migration first, giving us a real validated, StatusComplete
	// store to stand in for the partial an interrupted swap leaves behind.
	want, err := Run(dbPath, target, Options{})
	if err != nil {
		t.Fatalf("initial Run: %v", err)
	}

	// Simulate a crash BETWEEN atomicSwapDir's two renames.
	partial := partialDir(target)
	if err := os.Rename(target, partial); err != nil {
		t.Fatalf("stage partial store: %v", err)
	}
	aside := target + ".old"
	if err := os.MkdirAll(aside, 0o755); err != nil {
		t.Fatalf("stage %s: %v", aside, err)
	}
	if err := os.WriteFile(filepath.Join(aside, "original-marker"), []byte("orig"), 0o600); err != nil {
		t.Fatalf("stage .old marker: %v", err)
	}
	if exists(target) {
		t.Fatalf("precondition: target %s must be absent to simulate the interrupted swap", target)
	}

	// In-place semantics: from == to, and the source directory is now gone
	// (it became <target>.old).
	res, err := Run(target, target, Options{})
	if err != nil {
		t.Fatalf("recovery Run: %v", err)
	}
	if !res.Resumed {
		t.Error("recovery Run should report Resumed=true")
	}
	if res.Nodes != want.Nodes || res.Edges != want.Edges || res.Files != want.Files {
		t.Errorf("recovered counts (n=%d e=%d f=%d) != original (n=%d e=%d f=%d)",
			res.Nodes, res.Edges, res.Files, want.Nodes, want.Edges, want.Files)
	}

	if !exists(target) {
		t.Fatalf("target %s must exist after recovery", target)
	}
	if exists(aside) {
		t.Errorf("stale %s must be removed after recovery", aside)
	}
	if exists(partial) {
		t.Errorf("partial %s must be gone (renamed into place) after recovery", partial)
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
		t.Error("recovered Meta.Healthy = false, want true")
	}
}

// TestRun_RecoveredSwapReportsSourceCounts proves WR-01: a resumed/recovered
// migration (which returns through finishFromComplete) reports the real source
// counts in Result.Report, not the zero value. Before the fix
// finishFromComplete left Report zero-valued, so the CLI printed
// "files=N/0 nodes=N/0 edges=N/0", making a fully-validated migration look
// empty. This drives the same interrupted-swap recovery path as
// TestRun_RecoversInterruptedSwap and asserts the reconciliation denominators.
func TestRun_RecoveredSwapReportsSourceCounts(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	// A full clean migration first, capturing the real source counts.
	want, err := Run(dbPath, target, Options{})
	if err != nil {
		t.Fatalf("initial Run: %v", err)
	}
	if want.Report.Nodes.Source == 0 || want.Report.Files.Source == 0 || want.Report.Edges.Source == 0 {
		t.Fatalf("fixture must have non-zero source counts: %+v", want.Report)
	}

	// Simulate a crash between atomicSwapDir's two renames: the validated,
	// StatusComplete store sits at the deterministic partial path and the
	// target is absent.
	partial := partialDir(target)
	if err := os.Rename(target, partial); err != nil {
		t.Fatalf("stage partial store: %v", err)
	}
	if exists(target) {
		t.Fatalf("precondition: target %s must be absent to simulate the interrupted swap", target)
	}

	// In-place semantics: from == to; recovery finishes via finishFromComplete.
	res, err := Run(target, target, Options{})
	if err != nil {
		t.Fatalf("recovery Run: %v", err)
	}
	if !res.Resumed {
		t.Fatal("recovery Run should report Resumed=true")
	}

	if res.Report.Files.Source != want.Report.Files.Source ||
		res.Report.Nodes.Source != want.Report.Nodes.Source ||
		res.Report.Edges.Source != want.Report.Edges.Source {
		t.Errorf("recovered Report source counts = files=%d nodes=%d edges=%d, want files=%d nodes=%d edges=%d (WR-01: finishFromComplete must not return a zero Report)",
			res.Report.Files.Source, res.Report.Nodes.Source, res.Report.Edges.Source,
			want.Report.Files.Source, want.Report.Nodes.Source, want.Report.Edges.Source)
	}
	if res.Report.Files.Source == 0 || res.Report.Nodes.Source == 0 || res.Report.Edges.Source == 0 {
		t.Errorf("recovered Report source counts must be non-zero (WR-01): %+v", res.Report)
	}
	// Migrated side must reconcile with the returned counts too.
	if res.Report.Nodes.Migrated != res.Nodes || res.Report.Files.Migrated != res.Files || res.Report.Edges.Migrated != res.Edges {
		t.Errorf("recovered Report migrated counts %+v disagree with Result (n=%d f=%d e=%d)", res.Report, res.Nodes, res.Files, res.Edges)
	}
}

// TestRun_RecoveryLeavesInProgressPartialAlone proves the recovery guard is
// conservative: an in_progress (NOT StatusComplete) partial with an absent
// target must NOT be swapped in by recoverInterruptedSwap — that store is
// unvalidated. With a live source present, the normal resume path handles it;
// recovery must step aside.
func TestRun_RecoveryLeavesInProgressPartialAlone(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	target := runTarget(t)

	// Interrupt the first run at its first durable checkpoint, leaving an
	// in_progress partial and no swapped target.
	stopped := false
	testStopAfterBatch = func(string, int64) bool {
		if stopped {
			return false
		}
		stopped = true
		return true
	}
	t.Cleanup(func() { testStopAfterBatch = nil })

	if _, err := Run(dbPath, target, Options{}); err == nil {
		t.Fatal("expected the interrupted first Run to error")
	}
	if exists(target) {
		t.Fatalf("target %s must not exist after an interrupted (pre-swap) run", target)
	}
	testStopAfterBatch = nil

	// Recovery must decline the in_progress partial; the normal resume path
	// (with the live source) then completes it.
	res, err := Run(dbPath, target, Options{})
	if err != nil {
		t.Fatalf("resume Run: %v", err)
	}
	if !res.Resumed {
		t.Error("expected the resume Run to report Resumed=true")
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

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("filepath.Abs(%s): %v", p, err)
	}
	return abs
}
