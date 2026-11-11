package indexer

import (
	"errors"
	"os"
	"path/filepath"
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
func filesIn(t *testing.T, store graphstore.GraphStore, paths ...string) []bool {
	t.Helper()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	held := make([]bool, len(paths))
	for i, p := range paths {
		_, err := r.GetFile(p)
		if err != nil && !errors.Is(err, graphstore.ErrNotFound) {
			t.Fatalf("GetFile(%s): %v", p, err)
		}
		held[i] = err == nil
	}
	return held
}

// TestSyncReaderSeesWholeSync checks that a reader of the same process
// gets the store while a sync has it open, and that its snapshot holds
// every file that the sync adds or none of them.
func TestSyncReaderSeesWholeSync(t *testing.T) {
	dir := t.TempDir()
	root, storeDir := filepath.Join(dir, "repo"), filepath.Join(dir, "store")
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/shared\n\ngo 1.26\n")
	write("a.go", "package shared\n\nfunc Alpha() {}\n")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The reader is open before the sync starts and stays open after it.
	reader, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer reader.Close()

	write("gen_a.go", "package shared\n\nfunc GenA() {}\n")
	write("gen_b.go", "package shared\n\nfunc GenB() {}\n")

	var duringSync [][]bool
	setSyncBeforeCommitHook(t, func() {
		// One snapshot through the reader that was open before the sync,
		// and one through an open made while the sync has the store.
		duringSync = append(duringSync, filesIn(t, reader, "gen_a.go", "gen_b.go"))
		late, err := graphstore.Open(storeDir)
		if err != nil {
			t.Errorf("Open while the sync has the store open: %v", err)
			return
		}
		defer late.Close()
		duringSync = append(duringSync, filesIn(t, late, "gen_a.go", "gen_b.go"))
	})
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Positive assertion: the hook ran, so two snapshots were taken while
	// the sync had the store open and before its commit.
	if len(duringSync) != 2 {
		t.Fatalf("%d snapshots were taken during the sync, want 2", len(duringSync))
	}
	for i, held := range duringSync {
		if held[0] || held[1] {
			t.Errorf("snapshot %d, taken before the commit, holds gen_a.go=%v gen_b.go=%v; want neither", i, held[0], held[1])
		}
	}
	if held := filesIn(t, reader, "gen_a.go", "gen_b.go"); !held[0] || !held[1] {
		t.Errorf("after the sync, the reader holds gen_a.go=%v gen_b.go=%v; want both", held[0], held[1])
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
		dir, err := os.MkdirTemp(base, "run")
		if err != nil {
			t.Fatalf("MkdirTemp: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		root, storeDir := filepath.Join(dir, "repo"), filepath.Join(dir, "store")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/prop\n\ngo 1.26\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		m := &propModel{files: map[string][]propDecl{
			"a.go": {{Name: "alpha"}},
			"x.ts": {{Name: "alpha"}},
		}}
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
