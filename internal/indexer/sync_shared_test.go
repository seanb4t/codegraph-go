package indexer

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/invariant"
)

// setSyncBeforeCommitHook sets syncBeforeCommitHook for the test.
func setSyncBeforeCommitHook(t interface{ Cleanup(func()) }, hook func()) {
	syncBeforeCommitHook = hook
	t.Cleanup(func() { syncBeforeCommitHook = nil })
}

// filesIn reports, for each path, whether a new snapshot of store holds
// its file record.
func filesIn(store graphstore.GraphStore, paths ...string) ([]bool, error) {
	r, err := store.Snapshot()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	held := make([]bool, len(paths))
	for i, p := range paths {
		_, err := r.GetFile(p)
		if err != nil && !errors.Is(err, graphstore.ErrNotFound) {
			return nil, err
		}
		held[i] = err == nil
	}
	return held, nil
}

// TestSyncReaderSeesWholeSync checks that a reader of the same process
// gets the store while a sync has it open, and that each of its snapshots
// holds every file that the sync adds or none of them.
func TestSyncReaderSeesWholeSync(t *testing.T) {
	dir := t.TempDir()
	root, storeDir := filepath.Join(dir, "repo"), filepath.Join(dir, "store")
	writeFixtureFile(t, root, "go.mod", "module example.com/shared\n\ngo 1.26\n")
	writeFixtureFile(t, root, "a.go", "package shared\n\nfunc Alpha() {}\n")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The reader is open before the sync starts and stays open after it.
	reader, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer reader.Close()
	added := []string{"gen_a.go", "gen_b.go"}
	writeFixtureFile(t, root, added[0], "package shared\n\nfunc GenA() {}\n")
	writeFixtureFile(t, root, added[1], "package shared\n\nfunc GenB() {}\n")

	// One goroutine takes snapshots through the reader from before the
	// sync starts until after it returns.
	var (
		wg        sync.WaitGroup
		stop      = make(chan struct{})
		snapshots [][]bool
		pollErr   error
	)
	wg.Go(func() {
		for {
			held, err := filesIn(reader, added...)
			if err != nil {
				pollErr = err
				return
			}
			snapshots = append(snapshots, held)
			select {
			case <-stop:
				return
			default:
			}
		}
	})

	// The hook runs while the sync has the store open, before its commit.
	// It reads through the reader, and through an open made at that moment.
	var beforeCommit [][]bool
	setSyncBeforeCommitHook(t, func() {
		late, err := graphstore.Open(storeDir)
		if err != nil {
			t.Errorf("Open while the sync has the store open: %v", err)
			return
		}
		defer late.Close()
		for _, s := range []graphstore.GraphStore{reader, late} {
			held, err := filesIn(s, added...)
			if err != nil {
				t.Errorf("a read while the sync has the store open: %v", err)
				return
			}
			beforeCommit = append(beforeCommit, held)
		}
	})
	_, syncErr := Sync(root, storeDir, Options{})
	close(stop)
	wg.Wait()
	if syncErr != nil {
		t.Fatalf("Sync: %v", syncErr)
	}
	if pollErr != nil {
		t.Fatalf("a snapshot during the sync: %v", pollErr)
	}

	// Positive assertion: the hook ran, so two snapshots were taken while
	// the sync had the store open and before its commit.
	if len(beforeCommit) != 2 {
		t.Fatalf("%d snapshots were taken before the commit, want 2", len(beforeCommit))
	}
	for i, held := range beforeCommit {
		if held[0] || held[1] {
			t.Errorf("snapshot %d, taken before the commit, holds gen_a.go=%v gen_b.go=%v; want neither", i, held[0], held[1])
		}
	}
	for i, held := range snapshots {
		if held[0] != held[1] {
			t.Errorf("snapshot %d of %d holds gen_a.go=%v gen_b.go=%v; want both or neither", i, len(snapshots), held[0], held[1])
		}
	}
	after, err := filesIn(reader, added...)
	if err != nil {
		t.Fatalf("a snapshot after the sync: %v", err)
	}
	if !after[0] || !after[1] {
		t.Errorf("after the sync, the reader holds gen_a.go=%v gen_b.go=%v; want both", after[0], after[1])
	}
}

// TestSyncSnapshotsAreWholeSyncs checks, for a drawn edit of a drawn tree,
// that a snapshot taken through the shared handle while Sync has the store
// open equals the graph from before the sync, and that a snapshot after
// the sync equals the graph that a new open reads.
func TestSyncSnapshotsAreWholeSyncs(t *testing.T) {
	base := t.TempDir()
	comparisons := 0
	defer func() {
		// Positive assertion: some drawn sync reached its commit, so the
		// property compared a snapshot taken during a sync.
		if !t.Failed() && comparisons == 0 {
			t.Errorf("no drawn sync reached its commit, so the property compared nothing")
		}
	}()
	rapid.Check(t, func(t *rapid.T) {
		dir, root, m := newPropRun(t, base)
		storeDir := filepath.Join(dir, "store")
		written := make(map[string]string)
		writeModel(t, root, m, written)
		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}

		reader, err := graphstore.Open(storeDir)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer reader.Close()
		snapshot := func() graphstore.Reader {
			r, err := reader.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot: %v", err)
			}
			return r
		}

		steps := rapid.IntRange(1, 4).Draw(t, "steps")
		for i := range steps {
			op := rapid.IntRange(0, len(propOps)-1).Draw(t, "op")
			desc := propOps[op](t, m)
			if desc == "" {
				continue
			}
			t.Logf("step %d: %s", i+1, desc)
			writeModel(t, root, m, written)

			before := snapshot()
			hookRuns := 0
			setSyncBeforeCommitHook(t, func() {
				hookRuns++
				comparisons++
				during := snapshot()
				defer during.Close()
				if err := invariant.Equal(during, before); err != nil {
					t.Errorf("a snapshot taken before the commit of the sync after %q differs from the graph before the sync:\n%v", desc, err)
				}
			})
			_, err := Sync(root, storeDir, Options{})
			before.Close()
			if err != nil {
				t.Fatalf("Sync after %q: %v", desc, err)
			}
			// A sync that finds no change to commit does not reach the
			// hook. Every other sync commits one time.
			if hookRuns > 1 {
				t.Fatalf("the sync after %q reached its commit %d times, want at most 1", desc, hookRuns)
			}

			after := snapshot()
			fresh, closeFresh := openRapidSnapshot(t, storeDir)
			err = invariant.Equal(after, fresh)
			after.Close()
			closeFresh()
			if err != nil {
				t.Fatalf("after the sync after %q, the shared handle differs from a new open:\n%v", desc, err)
			}
		}
	})
}
