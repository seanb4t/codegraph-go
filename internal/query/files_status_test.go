package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	t.Run("dir narrows by path prefix", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		unfiltered, err := engine.Files(FilesOptions{})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}

		narrowed, err := engine.Files(FilesOptions{Dir: "pkga/"})
		if err != nil {
			t.Fatalf("Files with dir=pkga/: unexpected error: %v", err)
		}
		if len(narrowed.Files) == 0 {
			t.Fatal("Files with dir=pkga/: got zero entries, want at least pkga/pkga.go and pkga/embed.go")
		}
		if len(narrowed.Files) >= len(unfiltered.Files) {
			t.Fatalf("Files with dir=pkga/: got %d entries, want fewer than unfiltered %d", len(narrowed.Files), len(unfiltered.Files))
		}
		for _, f := range narrowed.Files {
			if !strings.HasPrefix(f.Path, "pkga/") {
				t.Fatalf("Files with dir=pkga/: got non-matching path %q", f.Path)
			}
		}

		excluded, err := engine.Files(FilesOptions{Dir: "cmd/"})
		if err != nil {
			t.Fatalf("Files with dir=cmd/: unexpected error: %v", err)
		}
		if len(excluded.Files) != 0 {
			t.Fatalf("Files with dir=cmd/ (no matches in fixture): got %d entries, want 0", len(excluded.Files))
		}
	})

	t.Run("dir with zero matches returns empty result, not an error", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		got, err := engine.Files(FilesOptions{Dir: "does-not-exist/"})
		if err != nil {
			t.Fatalf("Files with dir=does-not-exist/: unexpected error: %v", err)
		}
		if len(got.Files) != 0 {
			t.Fatalf("Files with dir=does-not-exist/: got %d entries, want 0", len(got.Files))
		}
	})

	t.Run("dir composes with the language filter (AND)", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		got, err := engine.Files(FilesOptions{Dir: "pkga/", Filter: "go"})
		if err != nil {
			t.Fatalf("Files with dir=pkga/ filter=go: unexpected error: %v", err)
		}
		if len(got.Files) == 0 {
			t.Fatal("Files with dir=pkga/ filter=go: got zero entries, want pkga's Go files")
		}
		for _, f := range got.Files {
			if !strings.HasPrefix(f.Path, "pkga/") || f.Language != "go" {
				t.Fatalf("Files with dir=pkga/ filter=go: got entry Path=%q Language=%q that fails one predicate", f.Path, f.Language)
			}
		}

		none, err := engine.Files(FilesOptions{Dir: "pkga/", Filter: "nonexistent-language"})
		if err != nil {
			t.Fatalf("Files with dir=pkga/ filter=nonexistent-language: unexpected error: %v", err)
		}
		if len(none.Files) != 0 {
			t.Fatalf("Files with dir=pkga/ filter=nonexistent-language: got %d entries, want 0 (must satisfy both predicates)", len(none.Files))
		}
	})

	t.Run("dirPrefixMatches: plain prefix semantics, not a glob", func(t *testing.T) {
		cases := []struct {
			name string
			path string
			dir  string
			want bool
		}{
			{"empty dir is a no-op", "internal/query/files.go", "", true},
			{"direct prefix match", "internal/query/files.go", "internal/", true},
			{"./-prefixed path matches an un-prefixed dir", "./internal/query/files.go", "internal/query", true},
			{"non-matching prefix is excluded", "internal/query/files.go", "cmd/", false},
			{"dir is not treated as a glob", "internal/query/files.go", "internal/q*", false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := dirPrefixMatches(tc.path, tc.dir); got != tc.want {
					t.Fatalf("dirPrefixMatches(%q, %q) = %v, want %v", tc.path, tc.dir, got, tc.want)
				}
			})
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

	// WR-01: a zero-match flat-format result's "files" array field must
	// never marshal as JSON null (FilesResult.Files carries `omitempty`,
	// by design, for the flat-vs-tree mutually-exclusive shape — so a
	// zero-match result omits the key entirely rather than emitting
	// null; either way a JSON consumer must never observe a literal
	// null there).
	t.Run("zero-match files JSON never marshals \"files\" as null", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)

		got, err := engine.Files(FilesOptions{Filter: "nonexistent-language"})
		if err != nil {
			t.Fatalf("Files with unknown filter: unexpected error: %v", err)
		}
		if len(got.Files) != 0 {
			t.Fatalf("Files with filter=nonexistent-language: got %d entries, want 0", len(got.Files))
		}

		data, err := MarshalFilesJSON(got)
		if err != nil {
			t.Fatalf("MarshalFilesJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal files JSON: %v\n%s", err, data)
		}
		if raw, present := m["files"]; present && string(raw) == "null" {
			t.Fatalf(`Files JSON "files" key marshaled as null, want omitted or a valid []: %s`, data)
		}
	})
}

func TestStatus(t *testing.T) {
	engine, _ := filesStatusFixture(t)

	got, err := engine.Status(context.Background())
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

	t.Run("PendingChanges stays an inert placeholder; WorktreeMismatch is live and genuinely nil here", func(t *testing.T) {
		// PendingChanges remains the Phase-4 sync placeholder (D-06,
		// explicit REQUIREMENTS out-of-scope row) — unchanged rationale.
		if got.PendingChanges.Added != 0 || got.PendingChanges.Modified != 0 || got.PendingChanges.Removed != 0 {
			t.Fatalf("Status.PendingChanges: got %+v, want all-zero placeholder", got.PendingChanges)
		}
		// WorktreeMismatch is now LIVE (D-14/WORK-01), computed from
		// gitmeta.DetectIndexMismatch via Engine.WorktreeMismatch(). It is
		// nil here because this fixture is an ordinary indexed directory
		// with no borrowed worktree — a genuine "no mismatch" verdict, not
		// an inert placeholder. See engine_worktree_test.go's
		// TestEngineWorktreeMismatchViaOpenAt for the live, non-nil case.
		if got.WorktreeMismatch != nil {
			t.Fatalf("Status.WorktreeMismatch: got %v, want nil (no borrowed worktree in this fixture)", got.WorktreeMismatch)
		}
	})

	t.Run("no volatile keys leak into the JSON shape", func(t *testing.T) {
		// dbSizeBytes is deliberately EXCLUDED from this forbidden-key list
		// (D-08): our own status --json intentionally DOES emit it now,
		// asserted for presence-and-plausibility in the
		// "dbSizeBytes is present and plausible" subtest below. Only the
		// frozen TS golden oracle keeps it stripped (testdata/golden's
		// shared volatileKeys map, untouched by this plan).
		raw, err := MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal status JSON: %v", err)
		}
		for _, volatile := range []string{"lastIndexed", "createdAt", "updatedAt"} {
			if _, present := m[volatile]; present {
				t.Fatalf("Status JSON unexpectedly contains volatile key %q", volatile)
			}
		}
	})

	t.Run("dbSizeBytes is present and plausible", func(t *testing.T) {
		// D-08: byte-for-byte stability across reindexes is deliberately
		// NOT asserted here — Pebble's LSM compaction makes the on-disk
		// byte total genuinely nondeterministic across identical
		// reindexes, a STRONGER version of the SQLite WAL/page-
		// fragmentation rationale that made the frozen TS golden strip
		// this key in the first place. We assert only presence, integer
		// type, and plausibility (> 0), never a fixed or prior-run value.
		if got.DbSizeBytes <= 0 {
			t.Fatalf("Status.DbSizeBytes: got %d, want > 0 for a real indexed Pebble store", got.DbSizeBytes)
		}

		raw, err := MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal status JSON: %v", err)
		}
		rawSize, present := m["dbSizeBytes"]
		if !present {
			t.Fatal(`Status JSON missing "dbSizeBytes" key (STAT-01)`)
		}
		var size int64
		if err := json.Unmarshal(rawSize, &size); err != nil {
			t.Fatalf("dbSizeBytes did not decode as an integer: %v (%s)", err, rawSize)
		}
		if size <= 0 {
			t.Fatalf("Status JSON dbSizeBytes = %d, want > 0", size)
		}
	})

	t.Run("filesByLanguage counts files per language, languages derived from it", func(t *testing.T) {
		if len(got.FilesByLanguage) == 0 {
			t.Fatal("Status.FilesByLanguage: got empty map, want at least one language")
		}
		if got.FilesByLanguage["go"] != got.FileCount {
			t.Fatalf(`Status.FilesByLanguage["go"] = %d, want %d (gofixture is Go-only, equal to FileCount)`, got.FilesByLanguage["go"], got.FileCount)
		}
		for lang, count := range got.FilesByLanguage {
			if count <= 0 {
				t.Fatalf("Status.FilesByLanguage[%q] = %d, want > 0 for a present key", lang, count)
			}
		}

		// D-05: Languages must stay derived from FilesByLanguage (count >
		// 0, sorted) — same order/shape as before this plan — so the
		// golden JSON shape stays parity-stable.
		var wantLanguages []string
		for lang, count := range got.FilesByLanguage {
			if count > 0 {
				wantLanguages = append(wantLanguages, lang)
			}
		}
		sort.Strings(wantLanguages)
		if strings.Join(got.Languages, ",") != strings.Join(wantLanguages, ",") {
			t.Fatalf("Status.Languages = %v, want %v (derived from FilesByLanguage per D-05)", got.Languages, wantLanguages)
		}
	})

	t.Run("filesByLanguage is internal-only and absent from the JSON shape", func(t *testing.T) {
		// D-05: TS's --json derives `languages` from filesByLanguage and
		// discards the counts entirely — emitting a filesByLanguage key
		// here would be a NEW Go-vs-TS divergence in the exact shape the
		// golden oracle guards. The counts exist only to feed renderers.
		raw, err := MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal status JSON: %v", err)
		}
		if _, present := m["filesByLanguage"]; present {
			t.Fatal(`Status JSON unexpectedly contains "filesByLanguage" key (D-05: internal-only, json:"-")`)
		}
	})
}

// TestDbSizeBytes exercises the D-07 best-effort dbSizeBytes helper
// directly against a nonexistent directory, and confirms Status() as a
// whole degrades to DbSizeBytes == 0 (never erroring) when no repoRoot is
// configured — mirroring computeStale's e.repoRoot == "" degrade-safely
// contract (D-07/T-02-07: a missing/unreadable store dir must never fail
// the whole status call).
func TestDbSizeBytes(t *testing.T) {
	t.Run("nonexistent store dir returns 0 and an error, never panics", func(t *testing.T) {
		got, err := dbSizeBytes(filepath.Join(t.TempDir(), "does-not-exist"))
		if err == nil {
			t.Fatal("dbSizeBytes: expected error for a nonexistent directory, got nil")
		}
		if got != 0 {
			t.Fatalf("dbSizeBytes: got %d, want 0 for a nonexistent directory", got)
		}
	})

	t.Run("Status degrades DbSizeBytes to 0 when repoRoot is unset (New, not OpenAt)", func(t *testing.T) {
		engine, _ := filesStatusFixture(t)
		noRootEngine := New(engine.reader)

		got, err := noRootEngine.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: unexpected error with no repoRoot configured: %v", err)
		}
		if got.DbSizeBytes != 0 {
			t.Fatalf("Status.DbSizeBytes: got %d, want 0 when Engine has no repoRoot (D-07 best-effort degrade)", got.DbSizeBytes)
		}
	})
}
