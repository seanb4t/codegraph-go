package graphstore

import (
	"fmt"
	"sync"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestConcurrentReadersSingleWriter is the INDX-05 concurrency proof: many
// reader goroutines, each on its own Pebble snapshot, run lock-free
// alongside one writer goroutine committing a single batch, with no data
// race (run this test with `-race`) and no reader observing a torn/partial
// write — a snapshot only ever sees none of the batch's nodes (taken
// before commit) or all of them (taken after), never a subset.
//
// Per RESEARCH Assumption A4: this uses a manual goroutine +
// sync.WaitGroup + `-race` harness rather than testing/synctest. Pebble's
// snapshot/commit-pipeline timing is driven by its own internal
// sequence-number bookkeeping, not by synctest's fake clock, so a manual
// harness is the more direct fit for asserting "no torn write is ever
// observed" rather than a specific interleaving order.
func TestConcurrentReadersSingleWriter(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	// A snapshot taken before any writes must see nothing yet.
	preSnap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot (pre): %v", err)
	}
	if _, err := preSnap.GetNode("node-0"); err != ErrNotFound {
		t.Fatalf("pre-write snapshot GetNode: want ErrNotFound, got %v", err)
	}
	if err := preSnap.Close(); err != nil {
		t.Fatalf("preSnap.Close: %v", err)
	}

	const numNodes = 200
	const numReaders = 16

	var wg sync.WaitGroup

	// Barrier forcing genuine overlap between the readers' Snapshot() calls
	// and the writer's Commit() call (WR-03): without this, Go's scheduler
	// could serialize the whole run (all readers finish before the writer
	// is even scheduled, or vice versa) and the "no torn write" assertion
	// below would pass vacuously, without ever observing the interleaved
	// window it claims to prove is safe. readersReady tracks every reader
	// goroutine reaching the start barrier; the writer stages its batch,
	// waits for all readers to be poised at the barrier, then releases them
	// and calls Commit() immediately after — so the readers' Snapshot()
	// calls and the writer's Commit() race in earnest.
	var readersReady sync.WaitGroup
	readersReady.Add(numReaders)
	start := make(chan struct{})

	// Single coordinated writer: stages every node on one Batch and
	// commits once (D-04) — not one Pebble write per symbol.
	wg.Add(1)
	go func() {
		defer wg.Done()
		w, err := store.NewWriter()
		if err != nil {
			t.Errorf("NewWriter: %v", err)
			return
		}
		for i := 0; i < numNodes; i++ {
			n := &schema.Node{
				Id:   fmt.Sprintf("node-%d", i),
				Kind: "function",
				Name: fmt.Sprintf("Fn%d", i),
			}
			if err := w.PutNode(n); err != nil {
				t.Errorf("PutNode: %v", err)
				return
			}
		}
		readersReady.Wait()
		close(start)
		if err := w.Commit(); err != nil {
			t.Errorf("Commit: %v", err)
		}
	}()

	// N reader goroutines, each on its own snapshot, run concurrently with
	// the writer above. Each snapshot must see either NONE of the numNodes
	// nodes (its point-in-time predates the commit) or ALL of them (it
	// postdates the commit) — never a partial subset, since the writer
	// commits the whole batch atomically.
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			readersReady.Done()
			<-start

			snap, err := store.Snapshot()
			if err != nil {
				t.Errorf("Snapshot: %v", err)
				return
			}
			defer snap.Close()

			seen := 0
			for i := 0; i < numNodes; i++ {
				_, err := snap.GetNode(fmt.Sprintf("node-%d", i))
				switch {
				case err == nil:
					seen++
				case err == ErrNotFound:
					// Fine: this snapshot's point-in-time predates the commit.
				default:
					t.Errorf("GetNode: %v", err)
					return
				}
			}
			if seen != 0 && seen != numNodes {
				t.Errorf("snapshot observed a torn write: %d/%d nodes visible", seen, numNodes)
			}
		}()
	}

	wg.Wait()

	// A snapshot taken after Wait must see every node (final, post-commit
	// state).
	postSnap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot (post): %v", err)
	}
	defer postSnap.Close()
	for i := 0; i < numNodes; i++ {
		if _, err := postSnap.GetNode(fmt.Sprintf("node-%d", i)); err != nil {
			t.Fatalf("post-commit snapshot missing node-%d: %v", i, err)
		}
	}
}

// TestDeleteFileSubgraphPrunesOnlyThatFile proves DeleteFileSubgraph's
// single range-delete removes exactly the target file's own record and
// leaves a lexicographically adjacent sibling file (one whose path is a
// naive prefix/suffix of the target's) untouched — the D-03 key-encoding
// guarantee that fileSubgraphPrefix relies on.
func TestDeleteFileSubgraphPrunesOnlyThatFile(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutFile(&schema.File{Path: "pkg/foo.go", Language: "go"}); err != nil {
		t.Fatalf("PutFile foo: %v", err)
	}
	if err := w.PutFile(&schema.File{Path: "pkg/foo.go.bak", Language: "go"}); err != nil {
		t.Fatalf("PutFile foo.go.bak: %v", err)
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
	if _, err := snap.GetFile("pkg/foo.go.bak"); err != nil {
		t.Fatalf("GetFile(pkg/foo.go.bak): want no error (sibling must survive the delete), got %v", err)
	}
}
