package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestDataSymbolRebuild covers a store written before data symbols: its
// Meta record lacks has_data_symbols and it holds no call through a
// function-valued variable. The first sync, with no source changed,
// rebuilds every file once and gives what a fresh index gives.
func TestDataSymbolRebuild(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod":           "module example.com/m\n\ngo 1.22\n",
		"daemon/daemon.go": "package daemon\n\nfunc List() {}\n",
		"cli/cli.go":       "package cli\n\nimport \"example.com/m/daemon\"\n\nvar fn = daemon.List\n\nfunc run() { fn() }\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, closeFn := openSnapshot(t, storeDir)
	run := declaredNodes(t, r, "cli/cli.go", goextract.KindFunction, "run")[0].Id
	list := declaredNodes(t, r, "daemon/daemon.go", goextract.KindFunction, "List")[0].Id
	closeFn()

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	meta, err := snap.GetMeta()
	snap.Close()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.DeleteEdge(run, goextract.RefKindCalls, list); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	meta.HasDataSymbols = false
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	store.Close()

	stats, err := Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if stats.FilesReparsed != stats.Files || stats.Files != 2 {
		t.Errorf("Sync reparsed %d of %d files, want a full rebuild of 2", stats.FilesReparsed, stats.Files)
	}
	r, closeFn = openSnapshot(t, storeDir)
	if !hasEdge(t, r, run, goextract.RefKindCalls, list) {
		t.Error("after the rebuild, run does not call List through fn")
	}
	if m, err := r.GetMeta(); err != nil || !m.GetHasDataSymbols() {
		t.Errorf("after the rebuild, has_data_symbols = %v (%v), want true", m.GetHasDataSymbols(), err)
	}
	closeFn()
	requireSyncEqualsRun(t, root, storeDir)

	stats, err = Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if stats.FilesReparsed != 0 {
		t.Errorf("Sync 2 reparsed %d files, want none: the rebuild happens once", stats.FilesReparsed)
	}
}
