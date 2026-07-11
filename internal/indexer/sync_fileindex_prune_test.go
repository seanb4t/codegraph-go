package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
)

// TestSyncPruneOwnedEdgesRemovesStaleFileIndexEntry is the CR-04
// regression: when a dependent file's owned edge is discarded via
// pruneOwnedEdgesOnly (the narrow path — the file's own nodes and File
// record survive; only its outgoing edges are pruned before
// re-resolution), the matching x/ file-index entry for that edge must be
// removed too, not just the e/ edge record. Otherwise the x/ index
// accumulates a stale entry with no matching e/ record, which a later
// direct prune of that same file would enumerate and phantom-count as a
// real removal (corrupting Meta.EdgeCount's arithmetic) — a scenario that
// requires at least two sync generations with a changing dependent-edge
// target, which a single reindex-vs-sync comparison never exercises.
func TestSyncPruneOwnedEdgesRemovesStaleFileIndexEntry(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
		"pkg/b.go": "package pkg\n\nfunc UseFoo() int { return Foo() }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	fooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")
	useFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/b.go")

	assertFileIndexHasEdge(t, storeDir, "pkg/b.go", useFooID, "calls", fooID)

	// Delete pkg/a.go: Foo is pruned outright (pruneFileSubgraph +
	// DeleteFileSubgraph). pkg/b.go, a DEPENDENT (not itself
	// modified/deleted), goes through the narrow pruneOwnedEdgesOnly path —
	// its own UseFoo node and File record survive; only its owned outgoing
	// edges are discarded, to be regenerated once it is re-extracted. Since
	// Foo no longer exists anywhere in the graph, UseFoo's call becomes
	// unresolved: no new calls edge is regenerated at all this cycle.
	if err := os.Remove(filepath.Join(repoRoot, "pkg", "a.go")); err != nil {
		t.Fatalf("os.Remove: %v", err)
	}

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (delete dependency): %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	if hasEdge(t, r, useFooID, "calls", fooID) {
		t.Fatalf("expected UseFoo -> Foo edge to be gone (Foo no longer resolves anywhere)")
	}
	if _, err := r.GetNode(useFooID); err != nil {
		t.Fatalf("expected UseFoo's own node to survive as a content-unchanged dependent: %v", err)
	}

	// The CR-04 assertion: pkg/b.go's x/ index must NOT retain a stale
	// edge entry pointing at the now-removed (UseFoo -calls-> Foo) triple
	// — pruneOwnedEdgesOnly must delete the x/ entry alongside the e/
	// record, not just the e/ record alone.
	xit, err := r.IterateFileIndex("pkg/b.go")
	if err != nil {
		t.Fatalf("IterateFileIndex(pkg/b.go): %v", err)
	}
	defer xit.Close()
	for xit.Next() {
		e := xit.Entry()
		if !e.IsNode && e.Source == useFooID && e.Kind == "calls" && e.Target == fooID {
			t.Fatalf("stale x/ index entry survives for removed edge (%s -calls-> %s) — CR-04 regression", useFooID, fooID)
		}
	}
	if err := xit.Err(); err != nil {
		t.Fatalf("IterateFileIndex(pkg/b.go) error: %v", err)
	}

	assertNoOrphansOrDangling(t, r)
}
