package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
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
			// WR-01 regression: a sibling directory whose name is a
			// literal string-prefix of dir (or vice versa) must NOT
			// match -- the missing path-separator boundary check was
			// the actual defect, distinct from the "not a glob" design.
			{"WR-01: sibling directory sharing a string prefix is excluded", "pkgab/bar.go", "pkga", false},
			{"WR-01: the same sibling collision without a trailing slash on dir", "pkgab/bar.go", "pkga/", false},
			{"WR-01: the real directory still matches", "pkga/foo.go", "pkga", true},
			{"WR-01: an exact-path match (no trailing separator) is a boundary match", "pkga", "pkga", true},
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
			t.Fatalf("Status.Backend: got %q, want a non-empty, non-node-sqlite value", got.Backend)
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
		// golden JSON shape stays stable under the golden suite.
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

	t.Run("filesByLanguage is present in the JSON shape (v0.11.0 Phase 1, D-03)", func(t *testing.T) {
		// D-03: the Compatibility constraint that suppressed this key
		// (json:"-" — the JSON derives `languages` from this map and
		// discards the counts, the project's own shape) was formally retired
		// 2026-08-13 (engram record gw79qy2a9z). filesByLanguage is now
		// un-suppressed, emitted alongside the new edgesByKind tally
		// (FIXT-01).
		raw, err := MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal status JSON: %v", err)
		}
		raw2, present := m["filesByLanguage"]
		if !present {
			t.Fatal(`Status JSON missing "filesByLanguage" key (D-03: un-suppressed as of v0.11.0 Phase 1)`)
		}
		var decoded map[string]int64
		if err := json.Unmarshal(raw2, &decoded); err != nil {
			t.Fatalf("filesByLanguage did not decode as map[string]int64: %v (%s)", err, raw2)
		}
		if len(decoded) == 0 {
			t.Fatal(`Status JSON "filesByLanguage" decoded to an empty map, want at least one language`)
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

// filesGlobFixture augments the shared gofixture with a root-level file, a
// two-directories-deep file, and a second registered language (typescript)
// at the same nested location -- the corpus TestFilesPatternRecursiveGlob
// needs but the shared fixture does not provide (verified in-tree during
// cycle-1 review: gofixture is one directory level deep and Go-only). The
// extra files are written into the copied temp tree BEFORE indexFixture
// runs, so they are indexed like any other source file.
func filesGlobFixture(t *testing.T) *Engine {
	t.Helper()

	dir := copyFixture(t)

	extra := map[string]string{
		"termroot.go":                 "package main\n\nfunc TermRoot() {}\n",
		"internal/deep/termnested.go": "package deep\n\nfunc TermNested() {}\n",
		"internal/deep/termnested.ts": "export function termNested(): void {}\n",
	}
	for rel, content := range extra {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine
}

// globRefusingReader is a graphstore.Reader whose IterateFiles fails the
// test if it is ever invoked -- the instrumented-reader convention
// search_test.go's searchFakeReader established (D-06's ranking tests),
// reused here to prove refusal of a malformed pattern precedes the store
// scan (T-04-05's actual property), rather than merely producing the same
// error text after paying for a scan that already ran.
type globRefusingReader struct {
	t *testing.T
}

func (r *globRefusingReader) GetNode(string) (*schema.Node, error) {
	return nil, errors.New("globRefusingReader: GetNode not implemented")
}

func (r *globRefusingReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("globRefusingReader: GetFile not implemented")
}

func (r *globRefusingReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("globRefusingReader: GetMeta not implemented")
}

func (r *globRefusingReader) IterateEdges(string) (graphstore.EdgeIterator, error) {
	return nil, errors.New("globRefusingReader: IterateEdges not implemented")
}

func (r *globRefusingReader) IterateFiles() (graphstore.FileIterator, error) {
	r.t.Fatal("globRefusingReader: IterateFiles called -- a malformed pattern must be refused before the store is scanned")
	return nil, nil
}

func (r *globRefusingReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("globRefusingReader: IterateFileIndex not implemented")
}

func (r *globRefusingReader) IterateNodes() (graphstore.NodeIterator, error) {
	return nil, errors.New("globRefusingReader: IterateNodes not implemented")
}

func (r *globRefusingReader) Close() error { return nil }

// filesPathSet extracts the sorted set of Paths from a []FileEntry, for
// exact-set (never count-only) comparison against an expected set.
func filesPathSet(entries []FileEntry) []string {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	sort.Strings(paths)
	return paths
}

func assertFilesPathSet(t *testing.T, got []FileEntry, want []string) {
	t.Helper()
	gotSet := filesPathSet(got)
	wantSet := append([]string(nil), want...)
	sort.Strings(wantSet)
	if strings.Join(gotSet, ",") != strings.Join(wantSet, ",") {
		t.Fatalf("Files: got path set %v, want %v", gotSet, wantSet)
	}
}

// containsPath (explore_gate_test.go, same package) reports whether path
// is present in paths -- reused here rather than redeclared.

// expectedGlobMatches computes the independent-oracle expected result set
// for pattern by filtering the RAW indexed path set (rawFilePaths) through
// doublestar.Match directly -- the same independent-oracle convention this
// file's rawFilePaths already establishes for TestFiles (comparing against
// the graph's own IterateFiles rather than re-implementing Files'
// filtering), applied here so each subtest's expected set is derived from
// what the indexer actually produced (which varies by build tags/platform
// -- e.g. skip_linux.go is excluded on darwin but present on linux) rather
// than a hardcoded, platform-fragile literal list.
func expectedGlobMatches(t *testing.T, raw []string, pattern string) []string {
	t.Helper()
	var want []string
	for _, p := range raw {
		matched, err := doublestar.Match(pattern, p)
		if err != nil {
			t.Fatalf("doublestar.Match(%q, %q): unexpected error: %v", pattern, p, err)
		}
		if matched {
			want = append(want, p)
		}
	}
	sort.Strings(want)
	return want
}

// TestFilesPatternRecursiveGlob is the two-direction regression test for
// D-14: a Pattern of "**/*term*" must return both a nested match
// (internal/deep/termnested.go) and the root-level match (termroot.go)
// that works today, and existing non-recursive patterns must keep
// behaving identically. See 04-02-PLAN.md Task 1/2.
func TestFilesPatternRecursiveGlob(t *testing.T) {
	engine := filesGlobFixture(t)

	raw := rawFilePaths(t, engine)
	sortedRaw := append([]string(nil), raw...)
	sort.Strings(sortedRaw)
	t.Logf("indexed path set: %v", sortedRaw)

	t.Run("nested", func(t *testing.T) {
		got, err := engine.Files(FilesOptions{Pattern: "**/*term*"})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		want := expectedGlobMatches(t, raw, "**/*term*")
		if !containsPath(want, "internal/deep/termnested.go") {
			t.Fatal("fixture setup: expected corpus to include internal/deep/termnested.go, cannot test nested match")
		}
		assertFilesPathSet(t, got.Files, want)
	})

	t.Run("root_level", func(t *testing.T) {
		got, err := engine.Files(FilesOptions{Pattern: "**/*term*"})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		want := expectedGlobMatches(t, raw, "**/*term*")
		if !containsPath(want, "termroot.go") {
			t.Fatal("fixture setup: expected corpus to include termroot.go, cannot test root-level match")
		}
		assertFilesPathSet(t, got.Files, want)
	})

	t.Run("non_recursive_unchanged", func(t *testing.T) {
		got, err := engine.Files(FilesOptions{Pattern: "*term*"})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		want := expectedGlobMatches(t, raw, "*term*")
		assertFilesPathSet(t, got.Files, want)
		for _, f := range got.Files {
			if strings.Contains(f.Path, "/") {
				t.Fatalf("Files with pattern *term*: got nested path %q, want root-level only (non-recursive pattern must not cross '/')", f.Path)
			}
		}
	})

	t.Run("brace_alternation", func(t *testing.T) {
		got, err := engine.Files(FilesOptions{Pattern: "**/*.{go,ts}"})
		if err != nil {
			t.Fatalf("Files: unexpected error: %v", err)
		}
		want := expectedGlobMatches(t, raw, "**/*.{go,ts}")
		if !containsPath(want, "internal/deep/termnested.go") || !containsPath(want, "internal/deep/termnested.ts") {
			t.Fatal("fixture setup: expected corpus to include both a nested .go and .ts file, cannot test brace alternation")
		}
		assertFilesPathSet(t, got.Files, want)
	})

	t.Run("malformed_refused", func(t *testing.T) {
		_, err := engine.Files(FilesOptions{Pattern: "["})
		if err == nil {
			t.Fatal("Files with malformed pattern: expected error, got nil")
		}
		if !strings.Contains(err.Error(), `query: invalid pattern "["`) {
			t.Fatalf(`Files with malformed pattern: got error %q, want it to contain query: invalid pattern "["`, err.Error())
		}
	})

	t.Run("refusal_precedes_scan", func(t *testing.T) {
		refusingEngine := New(&globRefusingReader{t: t})
		_, err := refusingEngine.Files(FilesOptions{Pattern: "["})
		if err == nil {
			t.Fatal("Files with malformed pattern: expected error, got nil")
		}
		if !strings.Contains(err.Error(), `query: invalid pattern "["`) {
			t.Fatalf(`Files with malformed pattern: got error %q, want it to contain query: invalid pattern "["`, err.Error())
		}
	})

	t.Run("escaped_metacharacter_is_literal", func(t *testing.T) {
		matched, err := doublestar.Match(`**/*\[*`, "weird[name.go")
		if err != nil {
			t.Fatalf("doublestar.Match: unexpected error: %v", err)
		}
		if !matched {
			t.Fatal(`doublestar.Match("**/*\[*", "weird[name.go"): got false, want true (escaped metacharacter matches literally)`)
		}

		notMatched, err := doublestar.Match(`**/*\[*`, "plainname.go")
		if err != nil {
			t.Fatalf("doublestar.Match: unexpected error: %v", err)
		}
		if notMatched {
			t.Fatal(`doublestar.Match("**/*\[*", "plainname.go"): got true, want false`)
		}
	})
}
