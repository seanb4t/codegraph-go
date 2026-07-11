package indexer

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// writeFixture materializes a fresh go.mod + the given relative-path ->
// source-text map under a new temp directory, mirroring pipeline_test's
// testdata/gofixture convention but built per-test so each Sync test can
// control its own call graph precisely (INDX-03's dependent-recomputation
// tests need this precision — the shared testdata/gofixture tree is too
// interconnected to isolate a single-hop dependency).
func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/syncfixture\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

// writeFileSeq guarantees every writeFile call in this test binary stamps
// a strictly increasing, always-in-the-future mtime — filesystems with
// coarse mtime resolution (or two writes within the same test) could
// otherwise collide with a prior stat, silently defeating Sync's D-01a
// stat pre-filter regardless of content change.
var writeFileSeq atomic.Int64

// writeFile overwrites path's content and forces a distinct mtime, so
// Sync's stat pre-filter (mtime/size compare) always sees a change before
// falling through to the content-hash confirm.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s): %v", path, err)
	}
	seq := writeFileSeq.Add(1)
	stamp := time.Now().Add(time.Duration(seq) * time.Second)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("os.Chtimes(%s): %v", path, err)
	}
}

// openSnapshot opens storeDir and returns a Reader snapshot plus a close
// func that releases both the Reader and the GraphStore, so every Sync
// test can assert on the committed graph state with one call.
func openSnapshot(t *testing.T, storeDir string) (graphstore.Reader, func()) {
	t.Helper()
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	r, err := store.Snapshot()
	if err != nil {
		store.Close()
		t.Fatalf("Snapshot: %v", err)
	}
	return r, func() {
		r.Close()
		store.Close()
	}
}

// TestSyncReparsesOnlyChangedFiles proves a Sync after modifying exactly
// one file's body reparses only that file (D-01a) — not the whole repo.
func TestSyncReparsesOnlyChangedFiles(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Standalone() int { return 1 }\n",
		"pkg/b.go": "package pkg\n\nfunc Other() int { return 2 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, "pkg", "a.go"), "package pkg\n\nfunc Standalone() int { return 99 }\n")

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (incremental): %v", err)
	}
	if stats.FilesReparsed != 1 {
		t.Errorf("Stats.FilesReparsed = %d, want 1 (only pkg/a.go changed)", stats.FilesReparsed)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	id := nodeid.NodeID(goextract.KindFunction, "Standalone", "pkg/a.go")
	if _, err := r.GetNode(id); err != nil {
		t.Fatalf("GetNode(Standalone): %v", err)
	}
}

// TestSyncResolvesAcrossUnchangedFiles proves RESEARCH Pitfall 1: a
// modified file's NEW reference to a symbol declared in a completely
// unchanged file still resolves after Sync, because the resolve step is
// seeded from the store (not just the reparse batch).
func TestSyncResolvesAcrossUnchangedFiles(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/unchanged.go": "package pkg\n\nfunc Standalone() int { return 1 }\n",
		"pkg/caller.go":    "package pkg\n\nfunc Caller() int { return 0 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, "pkg", "caller.go"),
		"package pkg\n\nfunc Caller() int { return Standalone() }\n")

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (incremental): %v", err)
	}
	if stats.Unresolved != 0 {
		t.Errorf("Stats.Unresolved = %d, want 0 (Standalone must resolve via the store-seeded index — Pitfall 1)", stats.Unresolved)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	callerID := nodeid.NodeID(goextract.KindFunction, "Caller", "pkg/caller.go")
	standaloneID := nodeid.NodeID(goextract.KindFunction, "Standalone", "pkg/unchanged.go")
	if !hasEdge(t, r, callerID, "calls", standaloneID) {
		t.Fatalf("expected calls edge Caller -> Standalone across an unchanged file (Pitfall 1: store-seeded index required)")
	}
}

// TestSyncReExtractsDependents proves RESEARCH Pitfall 2: moving a symbol
// from one file to another (D-03's delete-old+add-new rename model) forces
// re-extraction of every caller referencing it, even though the caller's
// own on-disk content never changed — because Unresolved refs are never
// persisted, only re-extraction regenerates them for re-resolution.
func TestSyncReExtractsDependents(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
		"pkg/b.go": "package pkg\n\nfunc UseFoo() int { return Foo() }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	oldFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")
	useFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/b.go")

	// Move Foo out of a.go into a2.go. b.go's own source is left byte-for-
	// byte untouched — its content hash does not change.
	writeFile(t, filepath.Join(repoRoot, "pkg", "a.go"), "package pkg\n")
	writeFile(t, filepath.Join(repoRoot, "pkg", "a2.go"), "package pkg\n\nfunc Foo() int { return 1 }\n")

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (incremental): %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	if _, err := r.GetNode(oldFooID); err == nil {
		t.Errorf("expected old Foo node (pkg/a.go) to be pruned after the move")
	}

	newFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a2.go")
	if _, err := r.GetNode(newFooID); err != nil {
		t.Fatalf("GetNode(new Foo in a2.go): %v", err)
	}

	if !hasEdge(t, r, useFooID, "calls", newFooID) {
		t.Fatalf("expected UseFoo -> Foo(a2.go) edge to regenerate against the moved symbol's new node id (Pitfall 2: b.go must be re-extracted even though its own content is unchanged)")
	}
}

// TestSyncPrunesDeletedFile proves removing a file from disk prunes its
// owned nodes and File record via the x/ index (D-02/D-02a) — no orphaned
// nodes survive.
func TestSyncPrunesDeletedFile(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
		"pkg/b.go": "package pkg\n\nfunc Bar() int { return 2 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	fooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")

	if err := os.Remove(filepath.Join(repoRoot, "pkg", "a.go")); err != nil {
		t.Fatalf("os.Remove: %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (incremental): %v", err)
	}
	if stats.FilesPruned == 0 {
		t.Error("Stats.FilesPruned = 0, want > 0 after deleting a file")
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	if _, err := r.GetNode(fooID); err == nil {
		t.Error("expected Foo's node to be pruned after pkg/a.go was deleted")
	}
	if _, err := r.GetFile("pkg/a.go"); err == nil {
		t.Error("expected pkg/a.go's File record to be pruned after deletion")
	}
}

// TestSyncDetectsAddedFile proves a brand-new file's symbols appear after
// Sync, and that only the new file is reparsed.
func TestSyncDetectsAddedFile(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, "pkg", "b.go"), "package pkg\n\nfunc Bar() int { return 2 }\n")

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (incremental): %v", err)
	}
	if stats.FilesReparsed != 1 {
		t.Errorf("Stats.FilesReparsed = %d, want 1 (only the added file)", stats.FilesReparsed)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	barID := nodeid.NodeID(goextract.KindFunction, "Bar", "pkg/b.go")
	if _, err := r.GetNode(barID); err != nil {
		t.Fatalf("GetNode(Bar): %v", err)
	}
}

// TestSyncBackfillsPrePhase4Graph proves a store whose Meta.has_file_index
// is false triggers a one-time full re-index backfill (D-02b), after which
// a second Sync is a cheap no-op.
func TestSyncBackfillsPrePhase4Graph(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
	})
	storeDir := t.TempDir()

	// Simulate a pre-Phase-4 graph: a store with a Meta record present but
	// has_file_index explicitly false (as opposed to every other test's
	// totally-fresh store, which exercises the ErrNotFound branch of the
	// same backfill detection instead).
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	m := schema.NewMeta()
	m.HasFileIndex = false
	if err := w.PutMeta(m); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}
	if stats.Files == 0 {
		t.Error("Stats.Files = 0, want > 0 after backfill")
	}

	func() {
		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()
		meta, err := r.GetMeta()
		if err != nil {
			t.Fatalf("GetMeta: %v", err)
		}
		if !meta.GetHasFileIndex() {
			t.Error("Meta.HasFileIndex = false after backfill, want true")
		}
	}()

	stats2, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (second, should be no-op): %v", err)
	}
	if stats2.FilesReparsed != 0 {
		t.Errorf("Stats.FilesReparsed = %d on second sync, want 0 (no-op)", stats2.FilesReparsed)
	}
}

// TestSyncNoOpWhenNothingChanged proves a Sync with nothing changed on
// disk reparses zero files and prunes nothing.
func TestSyncNoOpWhenNothingChanged(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (no-op): %v", err)
	}
	if stats.FilesReparsed != 0 {
		t.Errorf("Stats.FilesReparsed = %d, want 0 (nothing changed)", stats.FilesReparsed)
	}
	if stats.FilesPruned != 0 {
		t.Errorf("Stats.FilesPruned = %d, want 0 (nothing changed)", stats.FilesPruned)
	}
}

// TestSyncRefreshesStaleMtimeOnHashEqualSkip is the WR-03 regression: a
// file whose on-disk mtime/size differs from the stored File record, but
// whose recomputed content hash still matches (a touch, or a git checkout
// that doesn't change content), must have its stored File.MtimeUnixNs/
// SizeBytes refreshed to the current on-disk values — otherwise every
// subsequent Sync re-fails the cheap stat pre-filter for this file forever
// and pays the full content-hash cost again each time.
func TestSyncRefreshesStaleMtimeOnHashEqualSkip(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	stored := func() *schema.File {
		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()
		f, err := r.GetFile("pkg/a.go")
		if err != nil {
			t.Fatalf("GetFile(pkg/a.go): %v", err)
		}
		return f
	}
	before := stored()

	// Touch the file (new mtime, byte-identical content) — the stat
	// pre-filter must see mtime differ, recompute the hash, find it
	// matches, and (per WR-03) still refresh the stored mtime/size.
	full := filepath.Join(repoRoot, "pkg", "a.go")
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	writeFile(t, full, string(data))

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (touch-only): %v", err)
	}
	if stats.FilesReparsed != 0 {
		t.Errorf("Stats.FilesReparsed = %d, want 0 (content unchanged, must not be treated as modified)", stats.FilesReparsed)
	}

	after := stored()
	if after.GetContentHash() != before.GetContentHash() {
		t.Fatalf("ContentHash changed across a touch-only sync: before=%s after=%s", before.GetContentHash(), after.GetContentHash())
	}
	if after.GetMtimeUnixNs() == before.GetMtimeUnixNs() {
		t.Fatalf("stored MtimeUnixNs unchanged after a touch-only sync (%d) — WR-03 regression: the stale stat pre-filter metadata was never refreshed", after.GetMtimeUnixNs())
	}

	fileInfo, err := os.Stat(full)
	if err != nil {
		t.Fatalf("os.Stat: %v", err)
	}
	if after.GetMtimeUnixNs() != fileInfo.ModTime().UnixNano() {
		t.Fatalf("stored MtimeUnixNs = %d, want it to match the current on-disk mtime %d", after.GetMtimeUnixNs(), fileInfo.ModTime().UnixNano())
	}

	// A further Sync with truly nothing changed must now be a clean no-op
	// — proving the refreshed metadata restored the stat pre-filter's fast
	// path (before WR-03's fix, this file would keep failing the cheap
	// mtime/size comparison on every subsequent Sync, forever).
	stats2, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (second, should be a clean no-op): %v", err)
	}
	if stats2.FilesReparsed != 0 {
		t.Errorf("Stats.FilesReparsed on the follow-up sync = %d, want 0", stats2.FilesReparsed)
	}
}

// TestSyncDeletedDependentNotDoubleCounted is the WR-05 regression:
// deleting a file that is ITSELF a dependent of another pruned symbol
// (i.e. it both goes through the direct pruneFileSubgraph path AND would
// otherwise be discovered via the reverse-adjacency dependent scan) must
// not also be recomputed as a "dependent" — dependentPaths must exclude
// paths already present in `deleted`.
func TestSyncDeletedDependentNotDoubleCounted(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc Foo() int { return 1 }\n",
		"pkg/b.go": "package pkg\n\nfunc UseFoo() int { return Foo() }\n",
	})
	storeDir := t.TempDir()

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	// Delete BOTH a.go (Foo's own file) AND b.go (a caller of Foo) in the
	// same cycle: b.go is directly deleted, but its edge into the now-gone
	// Foo would ALSO make it match the reverse-adjacency dependent scan —
	// exactly the WR-05 double-classification.
	if err := os.Remove(filepath.Join(repoRoot, "pkg", "a.go")); err != nil {
		t.Fatalf("os.Remove(a.go): %v", err)
	}
	if err := os.Remove(filepath.Join(repoRoot, "pkg", "b.go")); err != nil {
		t.Fatalf("os.Remove(b.go): %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (delete both): %v", err)
	}
	if stats.DependentsRecomputed != 0 {
		t.Errorf("Stats.DependentsRecomputed = %d, want 0 (pkg/b.go was itself deleted, not merely a dependent) — WR-05 regression", stats.DependentsRecomputed)
	}
	if stats.FilesPruned != 2 {
		t.Errorf("Stats.FilesPruned = %d, want 2 (both a.go and b.go directly deleted)", stats.FilesPruned)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()
	if _, err := r.GetFile("pkg/a.go"); err == nil {
		t.Error("expected pkg/a.go File record pruned after deletion")
	}
	if _, err := r.GetFile("pkg/b.go"); err == nil {
		t.Error("expected pkg/b.go File record pruned after deletion")
	}
	assertNoOrphansOrDangling(t, r)
}
