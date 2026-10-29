package graphstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

func storeContents(t *testing.T, dir string) (nodeIDs, edges []string) {
	t.Helper()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open %s: %v", dir, err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()

	nit, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		nodeIDs = append(nodeIDs, nit.Node().Id)
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("node iteration: %v", err)
	}

	eit, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		edges = append(edges, e.Source+" "+e.Kind+" "+e.Target)
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("edge iteration: %v", err)
	}
	slices.Sort(nodeIDs)
	slices.Sort(edges)
	return nodeIDs, edges
}

func TestCheckpointCopiesGraph(t *testing.T) {
	src := t.TempDir()
	store, err := Open(src)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for i := range 10 {
		if err := w.PutNode(&schema.Node{Id: fmt.Sprintf("function:n%d", i), Kind: "function"}); err != nil {
			t.Fatalf("PutNode: %v", err)
		}
		if i > 0 {
			e := &schema.Edge{Source: fmt.Sprintf("function:n%d", i-1), Kind: "calls", Target: fmt.Sprintf("function:n%d", i)}
			if err := w.PutEdge(e, "a.go"); err != nil {
				t.Fatalf("PutEdge: %v", err)
			}
		}
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "seed", "store")
	if err := Checkpoint(src, dest); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}

	wantNodes, wantEdges := storeContents(t, src)
	gotNodes, gotEdges := storeContents(t, dest)
	if len(wantNodes) != 10 || len(wantEdges) != 9 {
		t.Fatalf("source holds %d nodes and %d edges; want 10 and 9", len(wantNodes), len(wantEdges))
	}
	if !slices.Equal(gotNodes, wantNodes) {
		t.Errorf("checkpoint nodes = %v; want %v", gotNodes, wantNodes)
	}
	if !slices.Equal(gotEdges, wantEdges) {
		t.Errorf("checkpoint edges = %v; want %v", gotEdges, wantEdges)
	}
}

func TestCheckpointLockedSource(t *testing.T) {
	src := t.TempDir()
	holder, err := Open(src)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer holder.Close()

	orig := openLockRetrySleep
	openLockRetrySleep = func(time.Duration) {}
	t.Cleanup(func() { openLockRetrySleep = orig })

	dest := filepath.Join(t.TempDir(), "store")
	err = Checkpoint(src, dest)
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("Checkpoint error = %v; want errors.Is(err, ErrStoreLocked)", err)
	}
	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("destination after a failed checkpoint: stat error = %v; want not exist", statErr)
	}
}

func TestCheckpointMissingSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "store")
	if err := Checkpoint(src, filepath.Join(t.TempDir(), "dest")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Checkpoint error = %v; want errors.Is(err, os.ErrNotExist)", err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("source after the checkpoint: stat error = %v; want not exist", err)
	}
}
