package query

import (
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// TestOpenAtDuringSync checks that OpenAt succeeds while another user of
// the same process has the store open, as a sync does, and that OpenAt and
// its Close leave that user's handle open. The graphstore tests prove that
// such an open runs no retry sleep.
func TestOpenAtDuringSync(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)
	storeDir := filepath.Join(dir, codegraphDirName, storeSubdir)

	held, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer held.Close()

	eng, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt while another user of this process has the store open: %v", err)
	}
	status, err := eng.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.NodeCount == 0 {
		t.Fatalf("Status reports no nodes; want the nodes of the fixture")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// The other user still reads after the Engine closed.
	r, err := held.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot through the other user's handle after the Engine closed: %v", err)
	}
	defer r.Close()
	if meta, err := r.GetMeta(); err != nil || meta.GetNodeCount() == 0 {
		t.Fatalf("GetMeta through the other user's handle = %v, %v; want the metadata of the fixture", meta, err)
	}
}
