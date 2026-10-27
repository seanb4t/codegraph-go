package indexer

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

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

// requireSyncEqualsRun fails t unless the store at storeDir, built by
// Sync, exports byte-identical to a fresh Run over the same tree, and
// returns the fresh store's directory.
func requireSyncEqualsRun(t *testing.T, repoRoot, storeDir string) string {
	t.Helper()
	freshDir := t.TempDir()
	if _, err := Run(repoRoot, freshDir, Options{}); err != nil {
		t.Fatalf("Run (fresh): %v", err)
	}
	synced, fresh := exportNormalized(t, storeDir), exportNormalized(t, freshDir)
	if !bytes.Equal(synced, fresh) {
		reportFirstDiff(t, synced, fresh)
	}
	return freshDir
}

// crossFileEdgeCount counts the edges in the store at storeDir whose
// source and target symbols are declared in different files.
func crossFileEdgeCount(t *testing.T, storeDir string) int {
	t.Helper()
	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()
	it, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()
	n := 0
	for it.Next() {
		e := it.Edge()
		src, err := r.GetNode(e.Source)
		if err != nil || src.Kind == goextract.KindFile {
			continue
		}
		dst, err := r.GetNode(e.Target)
		if err != nil || dst.Kind == kindPackage {
			continue
		}
		if src.FilePath != dst.FilePath {
			n++
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	return n
}

// TestSyncKeepsCrossFileEdgesOfEditedFile proves that editing a file and
// running Sync keeps its edges into files that did not change, in each
// language whose fresh index resolves a reference across files. Each
// language keys its symbols its own way, so the store-seeded index must
// use the key of each stored symbol's file.
func TestSyncKeepsCrossFileEdgesOfEditedFile(t *testing.T) {
	cases := []struct {
		lang  string
		files map[string]string
		edit  string
	}{
		{"typescript", map[string]string{
			"web/a.ts": "export function helper(): number {\n  return 1;\n}\n",
			"web/b.ts": "import { helper } from './a';\n\nexport function caller(): number {\n  return helper();\n}\n",
		}, "web/b.ts"},
		{"tsx", map[string]string{
			"web/a.tsx": "export function helper(): number {\n  return 1;\n}\n",
			"web/b.tsx": "import { helper } from './a';\n\nexport function caller(): number {\n  return helper();\n}\n",
		}, "web/b.tsx"},
		{"javascript", map[string]string{
			"web/a.js": "export function helper() {\n  return 1;\n}\n",
			"web/b.js": "import { helper } from './a.js';\n\nexport function caller() {\n  return helper();\n}\n",
		}, "web/b.js"},
		{"java", map[string]string{
			"src/com/ex/A.java": "package com.ex;\n\npublic class A {\n    public static int helper() { return 1; }\n}\n",
			"src/com/ex/B.java": "package com.ex;\n\npublic class B extends A {\n    A field;\n    int caller() { return helper(); }\n}\n",
		}, "src/com/ex/B.java"},
		{"csharp", map[string]string{
			"Ex/A.cs": "namespace Ex {\n    public class A {\n        public static int Helper() { return 1; }\n    }\n}\n",
			"Ex/B.cs": "namespace Ex {\n    public class B : A {\n        A field;\n        int Caller() { return Helper(); }\n    }\n}\n",
		}, "Ex/B.cs"},
		{"kotlin", map[string]string{
			"src/ex/A.kt": "package ex\n\nopen class A\n\nfun helper(): Int = 1\n",
			"src/ex/B.kt": "package ex\n\nclass B : A() {\n    fun caller(): Int = helper()\n}\n",
		}, "src/ex/B.kt"},
		{"php", map[string]string{
			"src/A.php": "<?php\nnamespace Ex;\n\nclass A {\n    public function helper() { return 1; }\n}\n",
			"src/B.php": "<?php\nnamespace Ex;\n\nclass B extends A {\n    public function caller() { return helper(); }\n}\n",
		}, "src/B.php"},
		{"swift", map[string]string{
			"Sources/Ex/A.swift": "func helper() -> Int {\n    return 1\n}\n",
			"Sources/Ex/B.swift": "func caller() -> Int {\n    return helper()\n}\n",
		}, "Sources/Ex/B.swift"},
	}
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			repoRoot := writeFixture(t, tc.files)
			storeDir := t.TempDir()
			if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
				t.Fatalf("Sync (seed): %v", err)
			}

			path := filepath.Join(repoRoot, tc.edit)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.edit, err)
			}
			writeFile(t, path, string(data)+"\n")
			stats, err := Sync(repoRoot, storeDir, Options{})
			if err != nil {
				t.Fatalf("Sync (edit): %v", err)
			}
			if stats.FilesReparsed != 1 {
				t.Errorf("Stats.FilesReparsed = %d, want 1 (only %s changed)", stats.FilesReparsed, tc.edit)
			}

			freshDir := requireSyncEqualsRun(t, repoRoot, storeDir)
			if n := crossFileEdgeCount(t, freshDir); n == 0 {
				t.Fatalf("a fresh index of the %s fixture has no cross-file edge, so this case proves nothing", tc.lang)
			}
		})
	}
}

// TestSyncResolvesSymbolAddedAfterCaller proves a reference to a symbol
// that does not exist yet gets its edge when a later Sync adds the
// symbol, although the referring file does not change again.
func TestSyncResolvesSymbolAddedAfterCaller(t *testing.T) {
	cases := []struct {
		name      string
		caller    string
		calleeRel string
		callee    string
	}{
		{
			name:      "same package",
			caller:    "package pkg\n\nfunc A() { B() }\n",
			calleeRel: "pkg/b.go",
			callee:    "package pkg\n\nfunc B() {}\n",
		},
		{
			name:      "imported package",
			caller:    "package pkg\n\nimport \"example.com/syncfixture/q\"\n\nfunc A() { q.B() }\n",
			calleeRel: "q/b.go",
			callee:    "package q\n\nfunc B() {}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := writeFixture(t, map[string]string{
				"pkg/a.go": "package pkg\n\nfunc A() {}\n",
			})
			storeDir := t.TempDir()
			if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
				t.Fatalf("Sync (seed): %v", err)
			}

			writeFile(t, filepath.Join(repoRoot, "pkg", "a.go"), tc.caller)
			if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
				t.Fatalf("Sync (caller): %v", err)
			}

			calleePath := filepath.Join(repoRoot, tc.calleeRel)
			if err := os.MkdirAll(filepath.Dir(calleePath), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			writeFile(t, calleePath, tc.callee)
			stats, err := Sync(repoRoot, storeDir, Options{})
			if err != nil {
				t.Fatalf("Sync (callee): %v", err)
			}
			if stats.DependentsRecomputed != 1 {
				t.Errorf("Stats.DependentsRecomputed = %d, want 1 (pkg/a.go waits on B)", stats.DependentsRecomputed)
			}

			func() {
				r, closeAll := openSnapshot(t, storeDir)
				defer closeAll()
				aID := nodeid.NodeID(goextract.KindFunction, "A", "pkg/a.go")
				bID := nodeid.NodeID(goextract.KindFunction, "B", tc.calleeRel)
				if !hasEdge(t, r, aID, "calls", bID) {
					t.Fatalf("no calls edge A -> B after the Sync that added B")
				}
			}()
			requireSyncEqualsRun(t, repoRoot, storeDir)
		})
	}
}

// TestSyncRestoresEdgesToRestoredFile proves that deleting a file and
// putting it back, as git stash and git stash pop do, leaves the edges
// into it that a fresh index has.
func TestSyncRestoresEdgesToRestoredFile(t *testing.T) {
	const ySrc = "package pkg\n\nfunc Y() int { return 1 }\n"
	repoRoot := writeFixture(t, map[string]string{
		"pkg/x.go": "package pkg\n\nfunc X() int { return Y() }\n",
		"pkg/y.go": ySrc,
	})
	storeDir := t.TempDir()
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	yPath := filepath.Join(repoRoot, "pkg", "y.go")
	if err := os.Remove(yPath); err != nil {
		t.Fatalf("os.Remove(y.go): %v", err)
	}
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (delete): %v", err)
	}
	writeFile(t, yPath, ySrc)
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (restore): %v", err)
	}

	func() {
		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()
		xID := nodeid.NodeID(goextract.KindFunction, "X", "pkg/x.go")
		yID := nodeid.NodeID(goextract.KindFunction, "Y", "pkg/y.go")
		if !hasEdge(t, r, xID, "calls", yID) {
			t.Fatalf("no calls edge X -> Y after y.go was restored")
		}
	}()
	requireSyncEqualsRun(t, repoRoot, storeDir)
}

// TestSyncWaitingFileReparsedOnce proves a file that waits on a name this
// sync adds, and is already reparsed for another reason, is reparsed and
// counted once: when it changed itself, and when it depends on a symbol
// this sync removes.
func TestSyncWaitingFileReparsedOnce(t *testing.T) {
	cases := []struct {
		name           string
		edits          map[string]string
		wantDependents int
	}{
		{
			name: "changed",
			edits: map[string]string{
				"pkg/x.go": "package pkg\n\nfunc X() { Y(); Z() }\n\n// edited\n",
				"pkg/z.go": "package pkg\n\nfunc Z() {}\n",
			},
			wantDependents: 0,
		},
		{
			name: "dependent",
			edits: map[string]string{
				"pkg/y.go": "package pkg\n\nfunc Z() {}\n",
			},
			wantDependents: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := writeFixture(t, map[string]string{
				"pkg/x.go": "package pkg\n\nfunc X() { Y(); Z() }\n",
				"pkg/y.go": "package pkg\n\nfunc Y() {}\n",
			})
			storeDir := t.TempDir()
			if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
				t.Fatalf("Sync (seed): %v", err)
			}

			for rel, src := range tc.edits {
				writeFile(t, filepath.Join(repoRoot, rel), src)
			}
			stats, err := Sync(repoRoot, storeDir, Options{})
			if err != nil {
				t.Fatalf("Sync (edit): %v", err)
			}
			if stats.FilesReparsed != 2 {
				t.Errorf("Stats.FilesReparsed = %d, want 2", stats.FilesReparsed)
			}
			if stats.DependentsRecomputed != tc.wantDependents {
				t.Errorf("Stats.DependentsRecomputed = %d, want %d", stats.DependentsRecomputed, tc.wantDependents)
			}

			func() {
				r, closeAll := openSnapshot(t, storeDir)
				defer closeAll()
				xID := nodeid.NodeID(goextract.KindFunction, "X", "pkg/x.go")
				zPath := "pkg/z.go"
				if _, ok := tc.edits[zPath]; !ok {
					zPath = "pkg/y.go"
				}
				zID := nodeid.NodeID(goextract.KindFunction, "Z", zPath)
				if !hasEdge(t, r, xID, "calls", zID) {
					t.Fatalf("no calls edge X -> Z after the Sync that added Z")
				}
			}()
			requireSyncEqualsRun(t, repoRoot, storeDir)
		})
	}
}

// TestSyncSkipsWaitingFilesWhenNoSymbolIsNew proves a Sync that adds no
// new symbol re-extracts no waiting file, even one waiting on a name the
// changed file declares: Set{} does not resolve to a map type, so x.go
// waits on Set, but Set is not new when y.go changes.
func TestSyncSkipsWaitingFilesWhenNoSymbolIsNew(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/x.go": "package pkg\n\nfunc X() { _ = Set{} }\n",
		"pkg/y.go": "package pkg\n\ntype Set map[string]bool\n",
	})
	storeDir := t.TempDir()
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	func() {
		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()
		f, err := r.GetFile("pkg/x.go")
		if err != nil {
			t.Fatalf("GetFile(pkg/x.go): %v", err)
		}
		want := []string{"example.com/syncfixture/pkg Set"}
		if got := unresolvedNameStrings(f); !slices.Equal(got, want) {
			t.Fatalf("pkg/x.go unresolved names = %q, want %q", got, want)
		}
	}()

	writeFile(t, filepath.Join(repoRoot, "pkg", "y.go"), "package pkg\n\ntype Set map[string]bool\n\n// edited\n")
	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (edit): %v", err)
	}
	if stats.FilesReparsed != 1 {
		t.Errorf("Stats.FilesReparsed = %d, want 1 (only pkg/y.go)", stats.FilesReparsed)
	}
	if stats.DependentsRecomputed != 0 {
		t.Errorf("Stats.DependentsRecomputed = %d, want 0 (Set is not new)", stats.DependentsRecomputed)
	}
}

// TestSyncBackfillsGraphWithoutUnresolvedNames proves a store whose Meta
// lacks has_unresolved_names is rebuilt from scratch by the next Sync,
// which records each file's unresolved names and stamps the flag, so the
// Sync after it is incremental again.
func TestSyncBackfillsGraphWithoutUnresolvedNames(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc A() { B() }\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	clearMetaFlag(t, storeDir, func(m *schema.Meta) { m.HasUnresolvedNames = false })

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}
	if stats.FilesReparsed != stats.Files {
		t.Errorf("Stats.FilesReparsed = %d, want %d (a full rebuild)", stats.FilesReparsed, stats.Files)
	}

	func() {
		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()
		meta, err := r.GetMeta()
		if err != nil {
			t.Fatalf("GetMeta: %v", err)
		}
		if !meta.GetHasUnresolvedNames() {
			t.Error("Meta.HasUnresolvedNames = false after backfill, want true")
		}
		f, err := r.GetFile("pkg/a.go")
		if err != nil {
			t.Fatalf("GetFile(pkg/a.go): %v", err)
		}
		want := []string{"example.com/syncfixture/pkg B"}
		if got := unresolvedNameStrings(f); !slices.Equal(got, want) {
			t.Errorf("pkg/a.go unresolved names = %q, want %q", got, want)
		}
	}()

	stats2, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (after backfill): %v", err)
	}
	if stats2.FilesReparsed != 0 {
		t.Errorf("Stats.FilesReparsed = %d on the Sync after the backfill, want 0", stats2.FilesReparsed)
	}
}

// clearMetaFlag rewrites the Meta record of the store at storeDir with
// clear applied, as a store written by an older build has it.
func clearMetaFlag(t *testing.T, storeDir string, clear func(*schema.Meta)) {
	t.Helper()
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	meta, err := r.GetMeta()
	r.Close()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	old := proto.Clone(meta).(*schema.Meta)
	clear(old)
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(old); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// TestSyncBackfillsGraphWithoutDeclaredModuleKeys proves a store whose
// Meta lacks has_declared_module_keys is rebuilt from scratch by the next
// Sync, which records the module a Java file declares, none for a Go
// file, and stamps the flag.
func TestSyncBackfillsGraphWithoutDeclaredModuleKeys(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go":          "package pkg\n\nfunc A() {}\n",
		"src/com/ex/A.java": "package com.ex;\n\npublic class A {}\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	clearMetaFlag(t, storeDir, func(m *schema.Meta) { m.HasDeclaredModuleKeys = false })

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync (backfill): %v", err)
	}
	if stats.FilesReparsed != stats.Files {
		t.Errorf("Stats.FilesReparsed = %d, want %d (a full rebuild)", stats.FilesReparsed, stats.Files)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()
	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHasDeclaredModuleKeys() {
		t.Error("Meta.HasDeclaredModuleKeys = false after backfill, want true")
	}
	for path, want := range map[string]string{"src/com/ex/A.java": "com.ex", "pkg/a.go": ""} {
		f, err := r.GetFile(path)
		if err != nil {
			t.Fatalf("GetFile(%s): %v", path, err)
		}
		if got := f.GetDeclaredModuleKey(); got != want {
			t.Errorf("%s declared module key = %q, want %q", path, got, want)
		}
	}
}

// TestSyncFollowsModulePathRename proves Sync files an unchanged file's
// symbols under the module key discovery computes on this run, not one
// stored earlier: after go.mod renames the module and the caller's
// import follows, its call into the unchanged package still resolves.
func TestSyncFollowsModulePathRename(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc A() {}\n",
		"cmd/b.go": "package cmd\n\nimport \"example.com/syncfixture/pkg\"\n\nfunc B() { pkg.A() }\n",
	})
	storeDir := t.TempDir()
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (seed): %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, "go.mod"), "module example.com/renamed\n\ngo 1.26\n")
	writeFile(t, filepath.Join(repoRoot, "cmd", "b.go"), "package cmd\n\nimport \"example.com/renamed/pkg\"\n\nfunc B() { pkg.A() }\n")
	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync (rename): %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()
	aID := nodeid.NodeID(goextract.KindFunction, "A", "pkg/a.go")
	bID := nodeid.NodeID(goextract.KindFunction, "B", "cmd/b.go")
	if !hasEdge(t, r, bID, "calls", aID) {
		t.Fatalf("no calls edge B -> A after the module rename")
	}
}

// unresolvedNameStrings renders f's unresolved names as "moduleKey name"
// strings, in stored order.
func unresolvedNameStrings(f *schema.File) []string {
	var out []string
	for _, u := range f.GetUnresolvedNames() {
		out = append(out, u.GetModuleKey()+" "+u.GetName())
	}
	return out
}
