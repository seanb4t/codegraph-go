package graphstore

import (
	"fmt"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestIterateNodes proves IterateNodes visits exactly every node record
// written through the real Writer/Commit path, each unmarshaled with its
// id/kind/name populated — a single contiguous range scan over the n/
// namespace (D-03), mirroring IterateEdges' contract.
func TestIterateNodes(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	const numNodes = 25
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	want := make(map[string]string, numNodes) // id -> kind
	for i := 0; i < numNodes; i++ {
		id := fmt.Sprintf("node-%d", i)
		kind := "function"
		if i%2 == 0 {
			kind = "struct"
		}
		if err := w.PutNode(&schema.Node{Id: id, Kind: kind, Name: fmt.Sprintf("Fn%d", i)}); err != nil {
			t.Fatalf("PutNode: %v", err)
		}
		want[id] = kind
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	it, err := snap.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()

	got := make(map[string]string, numNodes)
	count := 0
	for it.Next() {
		n := it.Node()
		if n == nil {
			t.Fatalf("Node() returned nil after Next() == true")
		}
		if n.Id == "" {
			t.Fatalf("Node() returned record with empty id")
		}
		got[n.Id] = n.Kind
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err after iteration: %v", err)
	}

	if count != numNodes {
		t.Fatalf("visited %d nodes, want %d", count, numNodes)
	}
	for id, kind := range want {
		gotKind, ok := got[id]
		if !ok {
			t.Fatalf("missing node %q in iteration results", id)
		}
		if gotKind != kind {
			t.Fatalf("node %q: kind = %q, want %q", id, gotKind, kind)
		}
	}
}

// TestIterateFiles proves IterateFiles visits exactly every file record
// written through the real Writer/Commit path, each unmarshaled with its
// path populated.
func TestIterateFiles(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	const numFiles = 10
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	want := make(map[string]bool, numFiles)
	for i := 0; i < numFiles; i++ {
		path := fmt.Sprintf("pkg/file%d.go", i)
		if err := w.PutFile(&schema.File{Path: path, Language: "go"}); err != nil {
			t.Fatalf("PutFile: %v", err)
		}
		want[path] = true
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	it, err := snap.IterateFiles()
	if err != nil {
		t.Fatalf("IterateFiles: %v", err)
	}
	defer it.Close()

	got := make(map[string]bool, numFiles)
	count := 0
	for it.Next() {
		f := it.File()
		if f == nil {
			t.Fatalf("File() returned nil after Next() == true")
		}
		if f.Path == "" {
			t.Fatalf("File() returned record with empty path")
		}
		got[f.Path] = true
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err after iteration: %v", err)
	}

	if count != numFiles {
		t.Fatalf("visited %d files, want %d", count, numFiles)
	}
	for path := range want {
		if !got[path] {
			t.Fatalf("missing file %q in iteration results", path)
		}
	}
}

// TestIterateNodesEmptyStore proves an empty store yields zero iterations
// and a nil Err() — the same "no records, no error" contract IterateEdges
// already guarantees.
func TestIterateNodesEmptyStore(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	it, err := snap.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()

	if it.Next() {
		t.Fatalf("Next() on empty store: want false, got true")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err() on empty store: want nil, got %v", err)
	}
}

// TestIterateEdgesEmptyPrefixScansEveryEdge proves IterateEdges("") scans
// the whole e/ namespace — every edge regardless of source — rather than
// only edges whose source happens to be the literal empty string. D-04
// (Phase 3's reverse-adjacency builder, internal/query/traverse.go) does
// exactly one IterateEdges("") scan per invocation and relies on this
// contract; edgeSrcPrefix("") alone would otherwise bound the scan to
// just the (never-written) empty-src slice of the keyspace, since
// appendSegment length-prefixes "" as a real, addressable zero-length
// segment rather than "no segment restriction".
func TestIterateEdgesEmptyPrefixScansEveryEdge(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	want := map[string]bool{
		"a->b": true,
		"b->c": true,
		"c->a": true,
	}
	if err := w.PutEdge(&schema.Edge{Source: "a", Target: "b", Kind: "calls"}, ""); err != nil {
		t.Fatalf("PutEdge: %v", err)
	}
	if err := w.PutEdge(&schema.Edge{Source: "b", Target: "c", Kind: "calls"}, ""); err != nil {
		t.Fatalf("PutEdge: %v", err)
	}
	if err := w.PutEdge(&schema.Edge{Source: "c", Target: "a", Kind: "calls"}, ""); err != nil {
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

	it, err := snap.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges(\"\"): %v", err)
	}
	defer it.Close()

	got := make(map[string]bool)
	count := 0
	for it.Next() {
		e := it.Edge()
		got[e.Source+"->"+e.Target] = true
		count++
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err after iteration: %v", err)
	}

	if count != len(want) {
		t.Fatalf("IterateEdges(\"\"): visited %d edges, want %d", count, len(want))
	}
	for k := range want {
		if !got[k] {
			t.Fatalf("IterateEdges(\"\"): missing edge %q in results", k)
		}
	}
}

// TestIterateFilesEmptyStore proves an empty store yields zero iterations
// and a nil Err() for IterateFiles.
func TestIterateFilesEmptyStore(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	it, err := snap.IterateFiles()
	if err != nil {
		t.Fatalf("IterateFiles: %v", err)
	}
	defer it.Close()

	if it.Next() {
		t.Fatalf("Next() on empty store: want false, got true")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err() on empty store: want nil, got %v", err)
	}
}
