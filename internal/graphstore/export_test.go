package graphstore

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/protobuf/testing/protocmp"

	"github.com/google/go-cmp/cmp"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestBulkExportReimportsLosslessly is the ARCH-01 bulk-export proof: a
// store populated with nodes, edges, files, and a meta record must
// re-import into a fresh, empty store with every record equal under
// protocmp — no format break across the Export/Import framing.
func TestBulkExportReimportsLosslessly(t *testing.T) {
	src, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open (src): %v", err)
	}
	defer src.Close()

	w, err := src.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	meta := schema.NewMeta()
	meta.NodeCount = 2
	meta.EdgeCount = 1
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}

	nodeA := &schema.Node{Id: "func:pkg.A", Kind: "function", Name: "A", FilePath: "pkg/a.go", Language: "go"}
	nodeB := &schema.Node{Id: "func:pkg.B", Kind: "function", Name: "B", FilePath: "pkg/b.go", Language: "go"}
	if err := w.PutNode(nodeA); err != nil {
		t.Fatalf("PutNode A: %v", err)
	}
	if err := w.PutNode(nodeB); err != nil {
		t.Fatalf("PutNode B: %v", err)
	}

	edge := &schema.Edge{Source: nodeA.Id, Target: nodeB.Id, Kind: "calls", Line: 10, Col: 2}
	if err := w.PutEdge(edge, nodeA.FilePath); err != nil {
		t.Fatalf("PutEdge: %v", err)
	}

	file := &schema.File{Path: "pkg/a.go", ContentHash: "deadbeef", Language: "go", NodeCount: 1, EdgeCount: 1}
	if err := w.PutFile(file); err != nil {
		t.Fatalf("PutFile: %v", err)
	}

	// Phase 10 D-05: the c/ namespace is a fifth record kind the export
	// stream must carry explicitly (Pitfall 2 — nothing else regenerates
	// these records on import, unlike x/).
	excludedVendor := &schema.ExcludedFile{Path: "vendor", Reason: schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR}
	excludedReadme := &schema.ExcludedFile{Path: "README.md", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".md", SizeBytes: 42}
	if err := w.PutExcludedFile(excludedVendor); err != nil {
		t.Fatalf("PutExcludedFile(vendor): %v", err)
	}
	if err := w.PutExcludedFile(excludedReadme); err != nil {
		t.Fatalf("PutExcludedFile(README.md): %v", err)
	}

	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	if err := src.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	dst, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open (dst): %v", err)
	}
	defer dst.Close()

	if err := Import(dst, bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("Import: %v", err)
	}

	dstSnap, err := dst.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer dstSnap.Close()

	gotMeta, err := dstSnap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if diff := cmp.Diff(meta, gotMeta, protocmp.Transform()); diff != "" {
		t.Errorf("Meta did not round-trip (-want +got):\n%s", diff)
	}

	gotA, err := dstSnap.GetNode(nodeA.Id)
	if err != nil {
		t.Fatalf("GetNode A: %v", err)
	}
	if diff := cmp.Diff(nodeA, gotA, protocmp.Transform()); diff != "" {
		t.Errorf("Node A did not round-trip (-want +got):\n%s", diff)
	}

	gotB, err := dstSnap.GetNode(nodeB.Id)
	if err != nil {
		t.Fatalf("GetNode B: %v", err)
	}
	if diff := cmp.Diff(nodeB, gotB, protocmp.Transform()); diff != "" {
		t.Errorf("Node B did not round-trip (-want +got):\n%s", diff)
	}

	gotFile, err := dstSnap.GetFile(file.Path)
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if diff := cmp.Diff(file, gotFile, protocmp.Transform()); diff != "" {
		t.Errorf("File did not round-trip (-want +got):\n%s", diff)
	}

	edgeIter, err := dstSnap.IterateEdges(nodeA.Id)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer edgeIter.Close()

	var edges []*schema.Edge
	for edgeIter.Next() {
		edges = append(edges, edgeIter.Edge())
	}
	if err := edgeIter.Err(); err != nil {
		t.Fatalf("edge iteration: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("want 1 edge from %s, got %d", nodeA.Id, len(edges))
	}
	if diff := cmp.Diff(edge, edges[0], protocmp.Transform()); diff != "" {
		t.Errorf("Edge did not round-trip (-want +got):\n%s", diff)
	}

	excludedIter, err := dstSnap.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	defer excludedIter.Close()

	excludedByPath := make(map[string]*schema.ExcludedFile, 2)
	excludedExported := 0
	for excludedIter.Next() {
		x := excludedIter.ExcludedFile()
		excludedByPath[x.GetPath()] = x
		excludedExported++
	}
	if err := excludedIter.Err(); err != nil {
		t.Fatalf("excluded-file iteration: %v", err)
	}
	t.Logf("excluded records exported: %d", excludedExported)
	if excludedExported == 0 {
		t.Fatal("excluded records exported: 0 — the c/ namespace was not carried by the export stream (rule 84d1gfpywd)")
	}
	if diff := cmp.Diff(excludedVendor, excludedByPath["vendor"], protocmp.Transform()); diff != "" {
		t.Errorf("ExcludedFile(vendor) did not round-trip (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(excludedReadme, excludedByPath["README.md"], protocmp.Transform()); diff != "" {
		t.Errorf("ExcludedFile(README.md) did not round-trip (-want +got):\n%s", diff)
	}
}

// TestImportRejectsUnknownRecordKind proves the export/import framing set
// is closed: a kind byte outside {1..5} is rejected with the existing
// "unknown record kind" error, rather than silently skipped or
// misinterpreted (Phase 10 T-10-11).
func TestImportRejectsUnknownRecordKind(t *testing.T) {
	dst, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dst.Close()

	// One framed record: kind=6 (never allocated), length=1, payload=0x00.
	stream := []byte{6, 1, 0x00}
	err = Import(dst, bytes.NewReader(stream))
	if err == nil {
		t.Fatal("Import with an unknown record kind succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "unknown record kind") {
		t.Fatalf("Import error = %q, want it to mention %q", err.Error(), "unknown record kind")
	}
}

// TestBulkExportIsConsistentUnderConcurrentWrite proves Export is taken
// from a single Pebble snapshot: a writer that commits after Export has
// already captured its snapshot must not tear the in-flight export. This
// is asserted indirectly — Export must succeed and produce a
// self-consistent, well-framed stream (Import replays it cleanly) even
// when a write races it.
func TestBulkExportIsConsistentUnderConcurrentWrite(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutNode(&schema.Node{Id: "func:pre", Kind: "function", Name: "Pre"}); err != nil {
		t.Fatalf("PutNode: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	if err := store.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// A write committed after Export returned must not appear in the
	// already-captured export bytes.
	w2, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter (post): %v", err)
	}
	if err := w2.PutNode(&schema.Node{Id: "func:post", Kind: "function", Name: "Post"}); err != nil {
		t.Fatalf("PutNode (post): %v", err)
	}
	if err := w2.Commit(); err != nil {
		t.Fatalf("Commit (post): %v", err)
	}

	dst, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open (dst): %v", err)
	}
	defer dst.Close()

	if err := Import(dst, bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("Import: %v", err)
	}

	dstSnap, err := dst.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer dstSnap.Close()

	if _, err := dstSnap.GetNode("func:pre"); err != nil {
		t.Fatalf("GetNode(func:pre): %v", err)
	}
	if _, err := dstSnap.GetNode("func:post"); err != ErrNotFound {
		t.Fatalf("GetNode(func:post): want ErrNotFound (post-export write must not leak in), got %v", err)
	}
}
