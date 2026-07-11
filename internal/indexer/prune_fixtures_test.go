package indexer

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
)

// prunefixtureRoot is a dedicated subfixture (distinct from fixtureRoot's
// shared discover_test.go/determinism_test.go tree, which several tests
// assert an exact file list against) purpose-built for the INDX-04 prune
// invariant: a self-contained cross-file call pair (pkg/a.go's Foo <-
// pkg/b.go's UseFoo) plus a cross-file receiver-method case (pkg/types.go's
// Widget <- pkg/methods.go's Describe <- pkg/caller.go's CallDescribe).
const prunefixtureRoot = "testdata/prunefixture"

// copyFixture materializes an independent copy of srcRoot under a fresh
// temp directory, so each subtest can freely create/modify/delete/rename/
// move files without disturbing the shared testdata tree or any other
// subtest running (t.Run subtests share the parent's working directory,
// never the source fixture itself).
func copyFixture(t *testing.T, srcRoot string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyFixture(%s): %v", srcRoot, err)
	}
	return dst
}

// assertNoOrphansOrDangling is the shared INDX-04 acceptance invariant: for
// every edge in the committed graph, both endpoints must resolve to a live
// node (no dangling edge); for every node id recorded in a live file's x/
// index, GetNode must succeed (no orphaned x/ entry pointing at a node that
// was deleted without also clearing its owning file's secondary index).
func assertNoOrphansOrDangling(t *testing.T, r graphstore.Reader) {
	t.Helper()

	eit, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges(\"\"): %v", err)
	}
	for eit.Next() {
		e := eit.Edge()
		if _, err := r.GetNode(e.Source); err != nil {
			t.Errorf("dangling edge: source %s (-%s-> %s) has no node: %v", e.Source, e.Kind, e.Target, err)
		}
		if _, err := r.GetNode(e.Target); err != nil {
			t.Errorf("dangling edge: target %s (%s -%s->) has no node: %v", e.Target, e.Source, e.Kind, err)
		}
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges(\"\") error: %v", err)
	}
	eit.Close()

	fit, err := r.IterateFiles()
	if err != nil {
		t.Fatalf("IterateFiles: %v", err)
	}
	var paths []string
	for fit.Next() {
		paths = append(paths, fit.File().GetPath())
	}
	if err := fit.Err(); err != nil {
		t.Fatalf("IterateFiles error: %v", err)
	}
	fit.Close()

	for _, path := range paths {
		xit, err := r.IterateFileIndex(path)
		if err != nil {
			t.Fatalf("IterateFileIndex(%s): %v", path, err)
		}
		for xit.Next() {
			entry := xit.Entry()
			if !entry.IsNode {
				continue
			}
			if _, err := r.GetNode(entry.NodeID); err != nil {
				t.Errorf("orphaned x/ index entry: node %s (owned by %s) has no node record: %v", entry.NodeID, path, err)
			}
		}
		if err := xit.Err(); err != nil {
			t.Fatalf("IterateFileIndex(%s) error: %v", path, err)
		}
		xit.Close()
	}
}

// assertFileIndexEmpty proves no x/<path>/... entry survives for path — the
// explicit "no x/ index entry keyed to the old path survives" check for a
// deleted/renamed/moved file, independent of assertNoOrphansOrDangling
// (which only walks entries owned by files still present in f/).
func assertFileIndexEmpty(t *testing.T, r graphstore.Reader, path string) {
	t.Helper()
	xit, err := r.IterateFileIndex(path)
	if err != nil {
		t.Fatalf("IterateFileIndex(%s): %v", path, err)
	}
	defer xit.Close()
	if xit.Next() {
		t.Errorf("expected no x/ index entries for %s, found at least one: %+v", path, xit.Entry())
	}
	if err := xit.Err(); err != nil {
		t.Fatalf("IterateFileIndex(%s) error: %v", path, err)
	}
}

// TestPruneFixtures is the INDX-04 acceptance gate (ROADMAP success
// criterion 2): create, modify, delete, rename, and move each leave the
// graph with no orphaned nodes and no dangling edges, verified against
// prunefixtureRoot's cross-file call pair (Foo/UseFoo).
func TestPruneFixtures(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repoRoot := copyFixture(t, prunefixtureRoot)
		storeDir := t.TempDir()
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (seed): %v", err)
		}

		writeFile(t, filepath.Join(repoRoot, "pkg", "extra.go"),
			"package pkg\n\nfunc UseFooToo() int { return Foo() }\n")

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (create): %v", err)
		}

		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()

		newID := nodeid.NodeID(goextract.KindFunction, "UseFooToo", "pkg/extra.go")
		fooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")
		if _, err := r.GetNode(newID); err != nil {
			t.Fatalf("GetNode(UseFooToo): %v", err)
		}
		if !hasEdge(t, r, newID, "calls", fooID) {
			t.Fatalf("expected UseFooToo -> Foo edge after create")
		}
		assertNoOrphansOrDangling(t, r)
	})

	t.Run("modify", func(t *testing.T) {
		repoRoot := copyFixture(t, prunefixtureRoot)
		storeDir := t.TempDir()
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (seed): %v", err)
		}

		oldFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")

		// MODIFY: rename Foo -> Foo2 within pkg/a.go. nodeid.NodeID (Phase-2
		// D-02a) hashes (kind, qualifiedName, filePath) — not source
		// content — so a body-only edit leaving the name unchanged would
		// leave Foo's id (and thus nothing to prune) untouched. Renaming
		// the symbol is the modify that genuinely changes its identity, so
		// this subtest exercises "the old id has no node and nothing
		// targets it" for real.
		writeFile(t, filepath.Join(repoRoot, "pkg", "a.go"),
			"package pkg\n\nfunc Foo2() int { return 42 }\n")

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (modify): %v", err)
		}

		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()

		if _, err := r.GetNode(oldFooID); err == nil {
			t.Error("expected old Foo node to be pruned after modify")
		}
		newFoo2ID := nodeid.NodeID(goextract.KindFunction, "Foo2", "pkg/a.go")
		if _, err := r.GetNode(newFoo2ID); err != nil {
			t.Fatalf("GetNode(Foo2): %v", err)
		}

		// UseFoo (pkg/b.go) had an edge into the now-gone Foo id; D-02a's
		// dependent detection re-extracts it, and since Foo no longer
		// resolves under that name, no edge may still target oldFooID.
		useFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/b.go")
		eit, err := r.IterateEdges(useFooID)
		if err != nil {
			t.Fatalf("IterateEdges(UseFoo): %v", err)
		}
		for eit.Next() {
			if eit.Edge().Target == oldFooID {
				t.Errorf("dangling edge: UseFoo still targets pruned Foo id %s", oldFooID)
			}
		}
		if err := eit.Err(); err != nil {
			t.Fatalf("IterateEdges(UseFoo) error: %v", err)
		}
		eit.Close()

		assertNoOrphansOrDangling(t, r)
	})

	t.Run("delete", func(t *testing.T) {
		repoRoot := copyFixture(t, prunefixtureRoot)
		storeDir := t.TempDir()
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (seed): %v", err)
		}

		fooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")

		if err := os.Remove(filepath.Join(repoRoot, "pkg", "a.go")); err != nil {
			t.Fatalf("os.Remove: %v", err)
		}

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (delete): %v", err)
		}

		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()

		if _, err := r.GetNode(fooID); err == nil {
			t.Error("expected Foo node pruned after pkg/a.go deleted")
		}
		if _, err := r.GetFile("pkg/a.go"); err == nil {
			t.Error("expected pkg/a.go File record pruned after deletion")
		}
		assertFileIndexEmpty(t, r, "pkg/a.go")

		assertNoOrphansOrDangling(t, r)
	})

	t.Run("rename", func(t *testing.T) {
		repoRoot := copyFixture(t, prunefixtureRoot)
		storeDir := t.TempDir()
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (seed): %v", err)
		}

		oldFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")

		oldPath := filepath.Join(repoRoot, "pkg", "a.go")
		newPath := filepath.Join(repoRoot, "pkg", "a2.go")
		data, err := os.ReadFile(oldPath)
		if err != nil {
			t.Fatalf("read pkg/a.go: %v", err)
		}
		if err := os.Remove(oldPath); err != nil {
			t.Fatalf("remove pkg/a.go: %v", err)
		}
		writeFile(t, newPath, string(data))

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (rename): %v", err)
		}

		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()

		if _, err := r.GetNode(oldFooID); err == nil {
			t.Error("expected old Foo node (pkg/a.go) pruned after rename")
		}
		if _, err := r.GetFile("pkg/a.go"); err == nil {
			t.Error("expected pkg/a.go File record pruned after rename")
		}
		assertFileIndexEmpty(t, r, "pkg/a.go")

		// Rename keeps the file in the same directory (same import path),
		// so UseFoo's unqualified call continues to resolve at the
		// content-preserving symbol's new (path-qualified) id — inbound
		// edges regenerate identically, no churn in the resulting graph
		// shape (D-03), even though the literal node id itself changes
		// because nodeid.NodeID's preimage includes filePath (see
		// SUMMARY.md for the full note on this).
		newFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a2.go")
		if _, err := r.GetNode(newFooID); err != nil {
			t.Fatalf("GetNode(Foo at pkg/a2.go): %v", err)
		}

		useFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/b.go")
		if !hasEdge(t, r, useFooID, "calls", newFooID) {
			t.Fatalf("expected UseFoo -> Foo(pkg/a2.go) edge to regenerate after rename")
		}

		assertNoOrphansOrDangling(t, r)
	})

	t.Run("move", func(t *testing.T) {
		repoRoot := copyFixture(t, prunefixtureRoot)
		storeDir := t.TempDir()
		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (seed): %v", err)
		}

		oldFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/a.go")
		oldUseFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/b.go")

		// MOVE relocates the Foo/UseFoo pair TOGETHER into pkg/sub/ — Go's
		// import path is directory-derived (RESEARCH/resolve.go), so
		// moving only the callee across a directory boundary would cross
		// a real package boundary and correctly leave the caller's
		// unqualified reference unresolved (not a prune bug — real Go
		// would refuse to compile that too). Moving the pair together
		// keeps them in the same (relocated) package, so the edge between
		// them regenerates cleanly, proving the invariant on a genuine
		// cross-directory move rather than a same-directory rename.
		if err := os.MkdirAll(filepath.Join(repoRoot, "pkg", "sub"), 0o755); err != nil {
			t.Fatalf("mkdir pkg/sub: %v", err)
		}
		for _, name := range []string{"a.go", "b.go"} {
			oldPath := filepath.Join(repoRoot, "pkg", name)
			newPath := filepath.Join(repoRoot, "pkg", "sub", name)
			data, err := os.ReadFile(oldPath)
			if err != nil {
				t.Fatalf("read pkg/%s: %v", name, err)
			}
			if err := os.Remove(oldPath); err != nil {
				t.Fatalf("remove pkg/%s: %v", name, err)
			}
			writeFile(t, newPath, string(data))
		}

		if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Sync (move): %v", err)
		}

		r, closeAll := openSnapshot(t, storeDir)
		defer closeAll()

		for _, old := range []string{"pkg/a.go", "pkg/b.go"} {
			if _, err := r.GetFile(old); err == nil {
				t.Errorf("expected %s File record pruned after move", old)
			}
			assertFileIndexEmpty(t, r, old)
		}
		if _, err := r.GetNode(oldFooID); err == nil {
			t.Error("expected old Foo node (pkg/a.go) pruned after move")
		}
		if _, err := r.GetNode(oldUseFooID); err == nil {
			t.Error("expected old UseFoo node (pkg/b.go) pruned after move")
		}

		newFooID := nodeid.NodeID(goextract.KindFunction, "Foo", "pkg/sub/a.go")
		newUseFooID := nodeid.NodeID(goextract.KindFunction, "UseFoo", "pkg/sub/b.go")
		if _, err := r.GetNode(newFooID); err != nil {
			t.Fatalf("GetNode(Foo at pkg/sub/a.go): %v", err)
		}
		if _, err := r.GetNode(newUseFooID); err != nil {
			t.Fatalf("GetNode(UseFoo at pkg/sub/b.go): %v", err)
		}
		if !hasEdge(t, r, newUseFooID, "calls", newFooID) {
			t.Fatalf("expected UseFoo -> Foo edge to regenerate at the moved pair's new ids")
		}

		assertNoOrphansOrDangling(t, r)
	})
}
