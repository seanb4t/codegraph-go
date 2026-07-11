package graphstore

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// fileIndexEntrySeen is the decoded shape TestFileIndex* assertions collect
// from a FileIndexIterator: either a node reference (IsNode true, NodeID
// set) or an edge reference (IsNode false, Source/Kind/Target set).
type fileIndexEntrySeen struct {
	isNode               bool
	nodeID               string
	source, kind, target string
}

// collectFileIndex drains a FileIndexIterator into a slice, closing it and
// failing the test on any iteration error.
func collectFileIndex(t *testing.T, it FileIndexIterator) []fileIndexEntrySeen {
	t.Helper()
	defer it.Close()

	var got []fileIndexEntrySeen
	for it.Next() {
		e := it.Entry()
		got = append(got, fileIndexEntrySeen{
			isNode: e.IsNode,
			nodeID: e.NodeID,
			source: e.Source,
			kind:   e.Kind,
			target: e.Target,
		})
	}
	if err := it.Err(); err != nil {
		t.Fatalf("FileIndexIterator.Err: %v", err)
	}
	return got
}

// TestFileIndexRoundTrip proves that PutNode/PutEdge record a file's owned
// node ids and outgoing edge triples under the x/ namespace, and that
// Reader.IterateFileIndex(path) yields exactly that file's entries — no
// other file's entries leak in.
func TestFileIndexRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Two nodes owned by pkg/a.go, one owned by pkg/b.go.
	nodeA1 := &schema.Node{Id: "func:pkg.A1", Kind: "function", Name: "A1", FilePath: "pkg/a.go"}
	nodeA2 := &schema.Node{Id: "func:pkg.A2", Kind: "function", Name: "A2", FilePath: "pkg/a.go"}
	nodeB1 := &schema.Node{Id: "func:pkg.B1", Kind: "function", Name: "B1", FilePath: "pkg/b.go"}
	for _, n := range []*schema.Node{nodeA1, nodeA2, nodeB1} {
		if err := w.PutNode(n); err != nil {
			t.Fatalf("PutNode(%s): %v", n.Id, err)
		}
	}

	// pkg/a.go has one outgoing edge (A1 calls B1); pkg/b.go has none.
	edgeA1B1 := &schema.Edge{Source: nodeA1.Id, Target: nodeB1.Id, Kind: "calls"}
	if err := w.PutEdge(edgeA1B1, "pkg/a.go"); err != nil {
		t.Fatalf("PutEdge: %v", err)
	}

	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	it, err := snap.IterateFileIndex("pkg/a.go")
	if err != nil {
		t.Fatalf("IterateFileIndex(pkg/a.go): %v", err)
	}
	got := collectFileIndex(t, it)

	wantNodeIDs := map[string]bool{nodeA1.Id: true, nodeA2.Id: true}
	gotNodeIDs := map[string]bool{}
	edgeCount := 0
	for _, e := range got {
		if e.isNode {
			gotNodeIDs[e.nodeID] = true
			continue
		}
		edgeCount++
		if e.source != nodeA1.Id || e.kind != "calls" || e.target != nodeB1.Id {
			t.Fatalf("unexpected edge entry: %+v", e)
		}
	}
	if len(gotNodeIDs) != len(wantNodeIDs) {
		t.Fatalf("pkg/a.go node entries = %v, want %v", gotNodeIDs, wantNodeIDs)
	}
	for id := range wantNodeIDs {
		if !gotNodeIDs[id] {
			t.Fatalf("missing node entry %q for pkg/a.go", id)
		}
	}
	if edgeCount != 1 {
		t.Fatalf("pkg/a.go edge entries = %d, want 1", edgeCount)
	}

	// pkg/b.go owns exactly nodeB1 and no edges (it has no outgoing edges).
	itB, err := snap.IterateFileIndex("pkg/b.go")
	if err != nil {
		t.Fatalf("IterateFileIndex(pkg/b.go): %v", err)
	}
	gotB := collectFileIndex(t, itB)
	if len(gotB) != 1 || !gotB[0].isNode || gotB[0].nodeID != nodeB1.Id {
		t.Fatalf("pkg/b.go file-index entries = %+v, want exactly [node %s]", gotB, nodeB1.Id)
	}
}

// TestFileIndexDeleteFileSubgraphPrunesXNamespace proves DeleteFileSubgraph
// removes both a file's f/ record AND every x/<path>/... entry it owns —
// IterateFileIndex on that path afterwards yields nothing — while a
// lexicographically adjacent sibling file's own x/ entries survive.
func TestFileIndexDeleteFileSubgraphPrunesXNamespace(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	nodeFoo := &schema.Node{Id: "func:pkg.Foo", Kind: "function", Name: "Foo", FilePath: "pkg/foo.go"}
	nodeBar := &schema.Node{Id: "func:pkg.Bar", Kind: "function", Name: "Bar", FilePath: "pkg/foo.go.bak"}
	if err := w.PutNode(nodeFoo); err != nil {
		t.Fatalf("PutNode foo: %v", err)
	}
	if err := w.PutNode(nodeBar); err != nil {
		t.Fatalf("PutNode bar: %v", err)
	}
	if err := w.PutFile(&schema.File{Path: "pkg/foo.go", Language: "go"}); err != nil {
		t.Fatalf("PutFile foo: %v", err)
	}
	if err := w.PutFile(&schema.File{Path: "pkg/foo.go.bak", Language: "go"}); err != nil {
		t.Fatalf("PutFile bak: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	w2, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter (delete): %v", err)
	}
	if err := w2.DeleteFileSubgraph("pkg/foo.go"); err != nil {
		t.Fatalf("DeleteFileSubgraph: %v", err)
	}
	if err := w2.Commit(); err != nil {
		t.Fatalf("Commit (delete): %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	if _, err := snap.GetFile("pkg/foo.go"); err != ErrNotFound {
		t.Fatalf("GetFile(pkg/foo.go): want ErrNotFound, got %v", err)
	}

	it, err := snap.IterateFileIndex("pkg/foo.go")
	if err != nil {
		t.Fatalf("IterateFileIndex(pkg/foo.go): %v", err)
	}
	got := collectFileIndex(t, it)
	if len(got) != 0 {
		t.Fatalf("IterateFileIndex(pkg/foo.go) after delete = %+v, want empty", got)
	}

	// Sibling survives.
	if _, err := snap.GetFile("pkg/foo.go.bak"); err != nil {
		t.Fatalf("GetFile(pkg/foo.go.bak): want no error (sibling must survive), got %v", err)
	}
	itBak, err := snap.IterateFileIndex("pkg/foo.go.bak")
	if err != nil {
		t.Fatalf("IterateFileIndex(pkg/foo.go.bak): %v", err)
	}
	gotBak := collectFileIndex(t, itBak)
	if len(gotBak) != 1 || !gotBak[0].isNode || gotBak[0].nodeID != nodeBar.Id {
		t.Fatalf("pkg/foo.go.bak file-index entries = %+v, want exactly [node %s]", gotBak, nodeBar.Id)
	}
}

// TestFileIndexPointDeletes proves Writer.DeleteNode/DeleteEdge remove
// exactly the targeted n//e/ records: GetNode returns ErrNotFound and
// IterateEdges no longer yields the deleted triple, while an untouched
// sibling record survives.
func TestFileIndexPointDeletes(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	nodeA := &schema.Node{Id: "func:pkg.A", Kind: "function", Name: "A", FilePath: "pkg/a.go"}
	nodeB := &schema.Node{Id: "func:pkg.B", Kind: "function", Name: "B", FilePath: "pkg/a.go"}
	nodeC := &schema.Node{Id: "func:pkg.C", Kind: "function", Name: "C", FilePath: "pkg/a.go"}
	for _, n := range []*schema.Node{nodeA, nodeB, nodeC} {
		if err := w.PutNode(n); err != nil {
			t.Fatalf("PutNode(%s): %v", n.Id, err)
		}
	}
	edgeAB := &schema.Edge{Source: nodeA.Id, Target: nodeB.Id, Kind: "calls"}
	edgeAC := &schema.Edge{Source: nodeA.Id, Target: nodeC.Id, Kind: "calls"}
	if err := w.PutEdge(edgeAB, "pkg/a.go"); err != nil {
		t.Fatalf("PutEdge AB: %v", err)
	}
	if err := w.PutEdge(edgeAC, "pkg/a.go"); err != nil {
		t.Fatalf("PutEdge AC: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	w2, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter (delete): %v", err)
	}
	if err := w2.DeleteNode(nodeB.Id); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	if err := w2.DeleteEdge(nodeA.Id, "calls", nodeB.Id); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	if err := w2.Commit(); err != nil {
		t.Fatalf("Commit (delete): %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	if _, err := snap.GetNode(nodeB.Id); err != ErrNotFound {
		t.Fatalf("GetNode(%s): want ErrNotFound, got %v", nodeB.Id, err)
	}
	if _, err := snap.GetNode(nodeC.Id); err != nil {
		t.Fatalf("GetNode(%s): want no error (untouched sibling), got %v", nodeC.Id, err)
	}

	edgeIt, err := snap.IterateEdges(nodeA.Id)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer edgeIt.Close()
	var remaining []*schema.Edge
	for edgeIt.Next() {
		remaining = append(remaining, edgeIt.Edge())
	}
	if err := edgeIt.Err(); err != nil {
		t.Fatalf("edge iteration: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Target != nodeC.Id {
		t.Fatalf("remaining edges from %s = %+v, want exactly [-> %s]", nodeA.Id, remaining, nodeC.Id)
	}
}

// TestFileIndexEncodingRejectsCollision mirrors
// TestKeyEncodingRejectsDelimiterInjection: for every adversarialSegments
// pair, fileIndexNodeKey/fileIndexEdgeKey for distinct (path,id)/(path,
// triple) never produce equal or range-overlapping keys, and no crafted
// path/id key falls inside another file's fileIndexPrefix range.
func TestFileIndexEncodingRejectsCollision(t *testing.T) {
	t.Run("distinct file-index node/edge entries never collide", func(t *testing.T) {
		seen := map[string]string{}
		for _, path := range adversarialSegments {
			for _, id := range adversarialSegments {
				k := fileIndexNodeKey(path, id)
				ks := string(k)
				desc := "node:" + path + "\x00" + id
				if other, ok := seen[ks]; ok && other != desc {
					t.Fatalf("fileIndexNodeKey(%q,%q) collides with %q: both encode to %x", path, id, other, k)
				}
				seen[ks] = desc
			}
		}
		for _, path := range adversarialSegments {
			for _, src := range adversarialSegments {
				k := fileIndexEdgeKey(path, src, "calls", "dst")
				ks := string(k)
				desc := "edge:" + path + "\x00" + src
				if other, ok := seen[ks]; ok && other != desc {
					t.Fatalf("fileIndexEdgeKey(%q,%q,...) collides with %q: both encode to %x", path, src, other, k)
				}
				seen[ks] = desc
			}
		}
	})

	t.Run("no crafted file-index key falls inside another namespace's range", func(t *testing.T) {
		for _, path := range adversarialSegments {
			for _, id := range adversarialSegments {
				k := fileIndexNodeKey(path, id)
				if k[0] != prefixFileIndex {
					t.Fatalf("fileIndexNodeKey(%q,%q) does not start with prefixFileIndex: %x", path, id, k)
				}
				for _, foreign := range []byte{prefixMeta, prefixNode, prefixEdge, prefixFile, prefixAnnotation} {
					if k[0] == foreign {
						t.Fatalf("fileIndexNodeKey(%q,%q) leading byte %q collides with foreign namespace %q", path, id, k[0], foreign)
					}
				}
			}
		}
	})

	t.Run("no file's fileIndexPrefix range bleeds into an adjacent file's", func(t *testing.T) {
		cases := []struct {
			target        string
			adjacentPaths []string
		}{
			{target: "foo", adjacentPaths: []string{"foobar", "foo2", "foo/bar.go"}},
			{target: "src/pkg/a", adjacentPaths: []string{"src/pkg/ab", "src/pkg/a/b.go"}},
		}
		for _, tc := range cases {
			prefix := fileIndexPrefix(tc.target)
			upper := rangeUpperBound(prefix)
			for _, adj := range tc.adjacentPaths {
				for _, id := range adversarialSegments {
					adjKey := fileIndexNodeKey(adj, id)
					within := len(adjKey) >= len(prefix) && string(adjKey[:len(prefix)]) == string(prefix) &&
						(upper == nil || string(adjKey) < string(upper))
					if within {
						t.Fatalf("fileIndexNodeKey(%q,%q) incorrectly falls inside %q's fileIndexPrefix range-delete window", adj, id, tc.target)
					}
				}
			}
		}
	})
}
