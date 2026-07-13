package migrate

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/migrate/migratetest"
)

// buildStoreFromSource walks src's files/nodes/edges tables (in that order,
// mirroring D-04's nodes-before-edges convention) and writes each
// translated record into a fresh graphstore at t.TempDir(), tracking each
// node's FilePath so PutEdge gets the correct ownerPath (the source node's
// file); a file:-prefixed edge source gets ownerPath="" (07-06's migrate.go
// will do the same — the dangling-edge check exempts file: endpoints
// rather than a synthesized owner catching them). This plan depends only
// on 07-03 (reader/translate), so validate_test.go builds its own store
// by hand rather than depending on 07-06's not-yet-written orchestration.
func buildStoreFromSource(t *testing.T, src *Source) graphstore.GraphStore {
	t.Helper()

	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	nodeFilePath := make(map[string]string)

	if err := src.ScanTable("files", 0, func(_ int64, row map[string]any) error {
		f, err := fileFromRow(row)
		if err != nil {
			return err
		}
		return w.PutFile(f)
	}); err != nil {
		t.Fatalf("scan files: %v", err)
	}

	if err := src.ScanTable("nodes", 0, func(_ int64, row map[string]any) error {
		n, err := nodeFromRow(row)
		if err != nil {
			return err
		}
		nodeFilePath[n.GetId()] = n.GetFilePath()
		return w.PutNode(n)
	}); err != nil {
		t.Fatalf("scan nodes: %v", err)
	}

	if err := src.ScanTable("edges", 0, func(_ int64, row map[string]any) error {
		e, err := edgeFromRow(row)
		if err != nil {
			return err
		}
		ownerPath := ""
		if !isFileEndpoint(e.GetSource()) {
			ownerPath = nodeFilePath[e.GetSource()]
		}
		return w.PutEdge(e, ownerPath)
	}); err != nil {
		t.Fatalf("scan edges: %v", err)
	}

	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	return store
}

func openHappySource(t *testing.T) *Source {
	t.Helper()
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	t.Cleanup(func() {
		if err := src.Close(); err != nil {
			t.Errorf("src.Close: %v", err)
		}
	})
	return src
}

func TestIsFileEndpoint(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"file:cmd/weft/main.go", true},
		{"class:1aa9ad9ada394f639ed0f8104462aef5", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isFileEndpoint(c.id); got != c.want {
			t.Errorf("isFileEndpoint(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

// TestValidate_HappyReconcile proves the happy-path invariant: migrated
// Node/File counts equal source row counts, migrated edges equal the
// source's DISTINCT(source,kind,target) count, and referential integrity
// is clean (VariantHappy's only "dangling-looking" sources are the
// file:-prefixed contains-edge sources, which are exempt).
func TestValidate_HappyReconcile(t *testing.T) {
	src := openHappySource(t)
	store := buildStoreFromSource(t, src)

	report, err := validate(src, store, Options{})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if report.Nodes.Migrated != report.Nodes.Source {
		t.Errorf("nodes: migrated %d != source %d", report.Nodes.Migrated, report.Nodes.Source)
	}
	if report.Files.Migrated != report.Files.Source {
		t.Errorf("files: migrated %d != source %d", report.Files.Migrated, report.Files.Source)
	}
	if report.Edges.Migrated != report.Edges.Source {
		t.Errorf("edges: migrated %d != source %d", report.Edges.Migrated, report.Edges.Source)
	}
	if len(report.Dangling) != 0 {
		t.Errorf("expected zero dangling, got %d: %+v", len(report.Dangling), report.Dangling)
	}
}

// TestValidate_EdgeDedupTolerance proves the D-09.1 de-dup-aware edge
// comparison: two source edge rows sharing (source,kind,target) but
// differing in (line,col) collapse to one stored edge (the Pebble edge key
// omits line/col), and reconcile must pass by comparing against
// CountDistinctEdges — a raw-row comparison would wrongly fail this
// correctly-collapsing migration.
func TestValidate_EdgeDedupTolerance(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)

	rw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open rw: %v", err)
	}
	defer func() {
		if err := rw.Close(); err != nil {
			t.Errorf("rw.Close: %v", err)
		}
	}()

	var source, kind, target string
	if err := rw.QueryRow(`SELECT source, kind, target FROM edges LIMIT 1`).Scan(&source, &kind, &target); err != nil {
		t.Fatalf("select existing edge: %v", err)
	}
	if _, err := rw.Exec(`INSERT INTO edges (source, kind, target, line, col) VALUES (?, ?, ?, 999, 999)`, source, kind, target); err != nil {
		t.Fatalf("insert duplicate-triple edge: %v", err)
	}

	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer func() {
		if err := src.Close(); err != nil {
			t.Errorf("src.Close: %v", err)
		}
	}()

	rawEdgeRows, err := src.CountRows("edges")
	if err != nil {
		t.Fatalf("CountRows(edges): %v", err)
	}
	distinctEdges, err := src.CountDistinctEdges()
	if err != nil {
		t.Fatalf("CountDistinctEdges: %v", err)
	}
	if rawEdgeRows == distinctEdges {
		t.Fatalf("test setup: expected raw row count (%d) > distinct triple count (%d)", rawEdgeRows, distinctEdges)
	}

	store := buildStoreFromSource(t, src)

	report, err := validate(src, store, Options{})
	if err != nil {
		t.Fatalf("validate: %v (a correctly-collapsing migration must pass its own check)", err)
	}
	if report.Edges.Source != distinctEdges {
		t.Errorf("report.Edges.Source = %d, want the DISTINCT count %d", report.Edges.Source, distinctEdges)
	}
	if report.Edges.Migrated != distinctEdges {
		t.Errorf("report.Edges.Migrated = %d, want %d (collapsed)", report.Edges.Migrated, distinctEdges)
	}
}

// TestReconcileCounts_MismatchFailsLoud proves a real migrated shortfall
// (simulated by deleting one migrated node after an otherwise-faithful
// build) returns a loud error naming the mismatched table and both counts,
// rather than silently succeeding.
func TestReconcileCounts_MismatchFailsLoud(t *testing.T) {
	src := openHappySource(t)
	store := buildStoreFromSource(t, src)

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	it, err := snap.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	if !it.Next() {
		t.Fatal("expected at least one node in the fixture")
	}
	victim := it.Node().GetId()
	if err := it.Close(); err != nil {
		t.Fatalf("close iterator: %v", err)
	}
	if err := snap.Close(); err != nil {
		t.Fatalf("close snapshot: %v", err)
	}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.DeleteNode(victim); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	_, err = reconcileCounts(src, store)
	if err == nil {
		t.Fatal("expected a fail-loud error on node count shortfall, got nil")
	}
	if !strings.Contains(err.Error(), "nodes") {
		t.Errorf("error %q does not name the mismatched table", err.Error())
	}
}

// TestValidate_DanglingFailsLoudByDefault proves a non-file: dangling edge
// (VariantDangling's extra edge, whose target resolves to no node) fails
// validate loud by default and leaves the store unchanged.
func TestValidate_DanglingFailsLoudByDefault(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantDangling)
	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer func() {
		if err := src.Close(); err != nil {
			t.Errorf("src.Close: %v", err)
		}
	}()

	store := buildStoreFromSource(t, src)

	edgesBefore, err := countEdges(store)
	if err != nil {
		t.Fatalf("countEdges: %v", err)
	}

	report, err := validate(src, store, Options{DropDangling: false})
	if err == nil {
		t.Fatal("expected a fail-loud error for a dangling non-file: edge")
	}
	if len(report.Dangling) != 1 {
		t.Fatalf("report.Dangling = %d entries, want 1: %+v", len(report.Dangling), report.Dangling)
	}
	if report.Dangling[0].MissingSource {
		t.Errorf("dangling edge's source should resolve (only target is missing): %+v", report.Dangling[0])
	}
	if !report.Dangling[0].MissingTarget {
		t.Errorf("dangling edge's target should be missing: %+v", report.Dangling[0])
	}

	edgesAfter, err := countEdges(store)
	if err != nil {
		t.Fatalf("countEdges: %v", err)
	}
	if edgesAfter != edgesBefore {
		t.Errorf("store changed on fail-loud path: before %d, after %d", edgesBefore, edgesAfter)
	}
}

// TestValidate_DropDangling proves --drop-dangling (Options.DropDangling)
// deletes the dangling edge, records it in Report.Dropped, and returns a
// nil error; a re-scan afterward finds zero dangling edges.
func TestValidate_DropDangling(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantDangling)
	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer func() {
		if err := src.Close(); err != nil {
			t.Errorf("src.Close: %v", err)
		}
	}()

	store := buildStoreFromSource(t, src)

	report, err := validate(src, store, Options{DropDangling: true})
	if err != nil {
		t.Fatalf("validate with DropDangling=true: %v", err)
	}
	if len(report.Dangling) != 1 {
		t.Fatalf("report.Dangling = %d, want 1", len(report.Dangling))
	}
	if report.Dropped != 1 {
		t.Errorf("report.Dropped = %d, want 1", report.Dropped)
	}

	var rescan Report
	if err := scanDangling(store, false, &rescan); err != nil {
		t.Fatalf("re-scan after drop: %v", err)
	}
	if len(rescan.Dangling) != 0 {
		t.Errorf("re-scan found %d dangling after drop, want 0: %+v", len(rescan.Dangling), rescan.Dangling)
	}
}
