package query

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// filesStatusFixture copies+indexes the shared gofixture (engine_test.go's
// copyFixture/indexFixture, reused at runtime only — Wave-3 isolation,
// 03-05-PLAN.md) and opens an Engine on it. It returns both the Engine and
// the fixture's root dir so tests can also simulate on-disk drift (e.g.
// deleting a file after indexing) to prove Files reads the frozen graph,
// not a live filesystem scan.
func filesStatusFixture(t *testing.T) (*Engine, string) {
	t.Helper()

	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine, dir
}

// rawFilePaths collects every file path the graph itself reports via a
// direct IterateFiles() scan — the independent oracle TestFiles compares
// Engine.Files' output against, so the test does not just re-implement
// Files' own filtering logic as its own check.
func rawFilePaths(t *testing.T, e *Engine) []string {
	t.Helper()

	it, err := e.reader.IterateFiles()
	if err != nil {
		t.Fatalf("IterateFiles: unexpected error: %v", err)
	}
	defer it.Close()

	var paths []string
	for it.Next() {
		paths = append(paths, it.File().Path)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateFiles: unexpected error: %v", err)
	}
	return paths
}

func TestFiles(t *testing.T) {
	t.Run("no options returns exactly what the graph's IterateFiles yields", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		want := rawFilePaths(t, engine)
		if len(want) == 0 {
			t.Fatal("fixture produced zero indexed files, cannot test Files")
		}

		got, err := engine.Files(FilesOptions{})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		if len(got.Files) != len(want) {
			t.Fatalf("Files: got %d entries, want %d (graph's IterateFiles count)", len(got.Files), len(want))
		}
		gotPaths := make(map[string]bool, len(got.Files))
		for _, f := range got.Files {
			gotPaths[f.Path] = true
		}
		for _, p := range want {
			if !gotPaths[p] {
				t.Fatalf("Files: missing path %q present in graph's IterateFiles", p)
			}
		}
	})

	t.Run("reads from the frozen graph, not a live filesystem scan", func(t *testing.T) {
		engine, dir := filesStatusFixture(t)

		before, err := engine.Files(FilesOptions{})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		var mainWasIndexed bool
		for _, f := range before.Files {
			if f.Path == "main.go" {
				mainWasIndexed = true
			}
		}
		if !mainWasIndexed {
			t.Fatal("fixture setup: expected main.go to be indexed, cannot prove graph-vs-filesystem divergence")
		}

		// Delete main.go from disk *after* indexing. If Files ever falls
		// back to (or supplements with) a live os.ReadDir/filesystem walk,
		// this file would disappear from the result. It must not.
		if err := os.Remove(filepath.Join(dir, "main.go")); err != nil {
			t.Fatalf("os.Remove(main.go): %v", err)
		}

		after, err := engine.Files(FilesOptions{})
		if err != nil {
			t.Fatalf("Files (post-delete): unexpected error: %v", err)
		}
		var stillPresent bool
		for _, f := range after.Files {
			if f.Path == "main.go" {
				stillPresent = true
			}
		}
		if !stillPresent {
			t.Fatal("Files: main.go vanished after on-disk deletion — Files is scanning the filesystem, not the graph")
		}
		if len(after.Files) != len(before.Files) {
			t.Fatalf("Files: entry count changed after on-disk deletion (got %d, want %d) — Files must be filesystem-independent", len(after.Files), len(before.Files))
		}
	})

	t.Run("pattern narrows the set", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		unfiltered, err := engine.Files(FilesOptions{})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}

		narrowed, err := engine.Files(FilesOptions{Pattern: "pkga/*.go"})
		if err != nil {
			t.Fatalf("Files with pattern: unexpected error: %v", err)
		}
		if len(narrowed.Files) == 0 {
			t.Fatal("Files with pattern pkga/*.go: got zero entries, want at least pkga.go/embed.go")
		}
		if len(narrowed.Files) >= len(unfiltered.Files) {
			t.Fatalf("Files with pattern: got %d entries, want fewer than unfiltered %d", len(narrowed.Files), len(unfiltered.Files))
		}
		for _, f := range narrowed.Files {
			if !strings.HasPrefix(f.Path, "pkga/") {
				t.Fatalf("Files with pattern pkga/*.go: got non-matching path %q", f.Path)
			}
		}
	})

	t.Run("filter narrows by language", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		none, err := engine.Files(FilesOptions{Filter: "nonexistent-language"})
		if err != nil {
			t.Fatalf("Files with unknown filter: unexpected error: %v", err)
		}
		if len(none.Files) != 0 {
			t.Fatalf("Files with filter=nonexistent-language: got %d entries, want 0", len(none.Files))
		}

		goOnly, err := engine.Files(FilesOptions{Filter: "go"})
		if err != nil {
			t.Fatalf("Files with filter=go: unexpected error: %v", err)
		}
		if len(goOnly.Files) == 0 {
			t.Fatal("Files with filter=go: got zero entries, want the fixture's Go files")
		}
		for _, f := range goOnly.Files {
			if f.Language != "go" {
				t.Fatalf("Files with filter=go: got entry with Language=%q", f.Language)
			}
		}
	})

	t.Run("depth limits directory nesting", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		rootOnly, err := engine.Files(FilesOptions{Depth: 1})
		if err != nil {
			t.Fatalf("Files with depth=1: unexpected error: %v", err)
		}
		if len(rootOnly.Files) == 0 {
			t.Fatal("Files with depth=1: got zero entries, want root-level files (e.g. main.go)")
		}
		for _, f := range rootOnly.Files {
			if strings.Contains(f.Path, "/") {
				t.Fatalf("Files with depth=1: got nested path %q, want root-level only", f.Path)
			}
		}

		nested, err := engine.Files(FilesOptions{Depth: 2})
		if err != nil {
			t.Fatalf("Files with depth=2: unexpected error: %v", err)
		}
		if len(nested.Files) <= len(rootOnly.Files) {
			t.Fatalf("Files with depth=2: got %d entries, want more than depth=1's %d (should include pkga/pkgb)", len(nested.Files), len(rootOnly.Files))
		}

		unlimited, err := engine.Files(FilesOptions{Depth: 0})
		if err != nil {
			t.Fatalf("Files with depth=0 (unlimited): unexpected error: %v", err)
		}
		full := rawFilePaths(t, engine)
		if len(unlimited.Files) != len(full) {
			t.Fatalf("Files with depth=0: got %d entries, want the full graph's %d (0 means unlimited)", len(unlimited.Files), len(full))
		}
	})

	t.Run("absurd depth is rejected, not silently clamped", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		if _, err := engine.Files(FilesOptions{Depth: -1}); err == nil {
			t.Fatal("Files with depth=-1: expected error, got nil")
		}
		if _, err := engine.Files(FilesOptions{Depth: MaxDepth + 1}); err == nil {
			t.Fatal("Files with depth=MaxDepth+1: expected error, got nil")
		}
	})

	t.Run("format toggles the projection", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		flat, err := engine.Files(FilesOptions{Format: "flat"})
		if err != nil {
			t.Fatalf("Files with format=flat: unexpected error: %v", err)
		}
		if len(flat.Files) == 0 || flat.Tree != nil {
			t.Fatalf("Files with format=flat: got Files=%d Tree=%v, want a populated flat list and nil Tree", len(flat.Files), flat.Tree)
		}

		tree, err := engine.Files(FilesOptions{Format: "tree"})
		if err != nil {
			t.Fatalf("Files with format=tree: unexpected error: %v", err)
		}
		if tree.Files != nil || len(tree.Tree) == 0 {
			t.Fatalf("Files with format=tree: got Files=%v Tree=%d, want nil Files and a populated Tree", tree.Files, len(tree.Tree))
		}

		var pkgaDir *FileTreeNode
		for _, node := range tree.Tree {
			if node.IsDir && node.Name == "pkga" {
				pkgaDir = node
			}
		}
		if pkgaDir == nil {
			t.Fatal("Files with format=tree: expected a pkga directory node at the top level")
		}
		var foundPkgaGo bool
		for _, child := range pkgaDir.Children {
			if !child.IsDir && child.Name == "pkga.go" {
				foundPkgaGo = true
				if child.Language != "go" {
					t.Fatalf("tree pkga/pkga.go: got Language=%q, want go", child.Language)
				}
				if child.Path != "pkga/pkga.go" {
					t.Fatalf("tree pkga/pkga.go: got Path=%q, want pkga/pkga.go", child.Path)
				}
			}
		}
		if !foundPkgaGo {
			t.Fatal("Files with format=tree: expected pkga/pkga.go as a leaf under the pkga directory node")
		}

		if _, err := engine.Files(FilesOptions{Format: "bogus"}); err == nil {
			t.Fatal("Files with format=bogus: expected error, got nil")
		}
	})
}

func TestStatus(t *testing.T) {
	engine, _ := filesStatusFixture(t)

	got, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}

	t.Run("initialized and non-zero counts consistent with the fixture graph", func(t *testing.T) {
		if !got.Initialized {
			t.Fatal("Status.Initialized: got false, want true")
		}
		if got.FileCount == 0 || got.NodeCount == 0 || got.EdgeCount == 0 {
			t.Fatalf("Status counts: got FileCount=%d NodeCount=%d EdgeCount=%d, want all non-zero", got.FileCount, got.NodeCount, got.EdgeCount)
		}

		wantFiles := rawFilePaths(t, engine)
		if got.FileCount != int64(len(wantFiles)) {
			t.Fatalf("Status.FileCount: got %d, want %d (graph's IterateFiles count)", got.FileCount, len(wantFiles))
		}

		var sumByKind int64
		for _, n := range got.NodesByKind {
			sumByKind += n
		}
		if sumByKind != got.NodeCount {
			t.Fatalf("Status.NodesByKind sums to %d, want NodeCount %d", sumByKind, got.NodeCount)
		}
		if got.NodesByKind["function"] == 0 {
			t.Fatalf("Status.NodesByKind: got %+v, want a non-zero function count", got.NodesByKind)
		}
	})

	t.Run("languages reflects Go-only extraction", func(t *testing.T) {
		if len(got.Languages) != 1 || got.Languages[0] != "go" {
			t.Fatalf("Status.Languages: got %v, want [\"go\"]", got.Languages)
		}
	})

	t.Run("backend renders a Pebble-truthful value, not node-sqlite", func(t *testing.T) {
		if got.Backend == "" || got.Backend == "node-sqlite" {
			t.Fatalf("Status.Backend: got %q, want a non-empty, non-TS-SQLite value", got.Backend)
		}
		if !strings.Contains(strings.ToLower(got.Backend), "pebble") {
			t.Fatalf("Status.Backend: got %q, want it to identify Pebble", got.Backend)
		}
	})

	t.Run("version/extraction fields derive from schema.SchemaVersion", func(t *testing.T) {
		wantVersion := fmt.Sprintf("%d", schema.SchemaVersion)
		if got.Version != wantVersion {
			t.Fatalf("Status.Version: got %q, want %q (schema.SchemaVersion-derived)", got.Version, wantVersion)
		}
		if got.Index.BuiltWithExtractionVersion != schema.SchemaVersion {
			t.Fatalf("Status.Index.BuiltWithExtractionVersion: got %d, want schema.SchemaVersion=%d", got.Index.BuiltWithExtractionVersion, schema.SchemaVersion)
		}
		if got.Index.CurrentExtractionVersion != schema.SchemaVersion {
			t.Fatalf("Status.Index.CurrentExtractionVersion: got %d, want schema.SchemaVersion=%d", got.Index.CurrentExtractionVersion, schema.SchemaVersion)
		}
		if got.Index.ReindexRecommended {
			t.Fatal("Status.Index.ReindexRecommended: got true, want false for a freshly-indexed fixture at the current schema version")
		}
	})

	t.Run("Phase-4-only keys render present-but-inert", func(t *testing.T) {
		if got.PendingChanges.Added != 0 || got.PendingChanges.Modified != 0 || got.PendingChanges.Removed != 0 {
			t.Fatalf("Status.PendingChanges: got %+v, want all-zero placeholder", got.PendingChanges)
		}
		if got.WorktreeMismatch != nil {
			t.Fatalf("Status.WorktreeMismatch: got %v, want nil placeholder", got.WorktreeMismatch)
		}
	})

	t.Run("no volatile keys leak into the JSON shape", func(t *testing.T) {
		raw, err := MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal status JSON: %v", err)
		}
		for _, volatile := range []string{"dbSizeBytes", "lastIndexed", "createdAt", "updatedAt"} {
			if _, present := m[volatile]; present {
				t.Fatalf("Status JSON unexpectedly contains volatile key %q", volatile)
			}
		}
	})
}
