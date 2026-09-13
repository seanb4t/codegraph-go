package indexer

import (
	"errors"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// coverageFixtureRoot is the committed Phase 10 D-13 fixture: one file of
// every kind — vendor/x.go (DIR_VENDOR), .hidden/y.go (DIR_DOTPREFIX),
// notes.md (UNSUPPORTED_EXTENSION), tagged.go (BUILD_TAG). The oversize
// SIZE_LIMIT file and the dangling-symlink extraction-failure file are
// generated at test time (never committed — D-13's own prohibition).
const coverageFixtureRoot = "testdata/coverage"

// copyCoverageFixture materializes an independent copy of coverageFixtureRoot
// under a fresh temp directory (dot-dirs and vendor/ included — this WalkDir
// copy never calls ShouldSkipDir), so each test can freely add generated
// files without disturbing the shared testdata tree. Thin wrapper around
// copyFixture (prune_fixtures_test.go), which already has this exact shape.
func copyCoverageFixture(t *testing.T) string {
	t.Helper()
	return copyFixture(t, coverageFixtureRoot)
}

// writeOversizeGoFile writes a syntactically valid, build-tag-free Go
// source file at exactly parser.MaxSourceBytes+1 bytes into dir/name —
// the SIZE_LIMIT trigger, generated fresh per test (D-13: never
// committed). Delegates to writeExactSizeGoFile (discoverexclusion_test.go).
func writeOversizeGoFile(t *testing.T, dir, name string) int64 {
	t.Helper()
	size := int64(parser.MaxSourceBytes + 1)
	writeExactSizeGoFile(t, filepath.Join(dir, name), size)
	return size
}

// writeDanglingSymlink creates a symlink at dir/name pointing at a
// nonexistent target — the extraction-failure trigger verified by
// TestDanglingSymlinkIsDiscoveredAndFailsExtraction below (RESEARCH.md
// Pitfall 4 / Assumption A1). Never committed (D-13).
func writeDanglingSymlink(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Symlink("missing-target.py", filepath.Join(dir, name)); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}
}

// TestDanglingSymlinkIsDiscoveredAndFailsExtraction is the RED gate on
// research assumption A1 (10-RESEARCH.md Pitfall 4): it must run and pass
// BEFORE the committed coverage fixture depends on the dangling-symlink
// technique for its extraction-failure file. A dangling symlink with a
// non-Go extension (.py) is discovered normally (d.Info() is lstat-based,
// so DiscoverAll never follows the link at walk time, and Go's build-tag
// MatchFile is only consulted for the "go" language) but fails at
// Extract's os.ReadFile step.
//
// This probe PASSED on its first run (verified 2026-09-13, darwin/arm64) —
// the dangling-symlink technique is confirmed valid; the chmod-0o000
// fallback RESEARCH.md names was not needed.
func TestDanglingSymlinkIsDiscoveredAndFailsExtraction(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module example.com/tmp\n\ngo 1.26\n")
	writeFixtureFile(t, root, "main.go", "package tmp\n\nfunc Main() {}\n")
	writeDanglingSymlink(t, root, "broken.py")

	d, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	var found *DiscoveredFile
	for i := range d.Files {
		if d.Files[i].RelPath == "broken.py" {
			found = &d.Files[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("Files = %+v, want broken.py present (not dropped)", d.Files)
	}
	if found.Language != "python" {
		t.Errorf("broken.py Language = %q, want %q", found.Language, "python")
	}
	for _, x := range d.Excluded {
		if x.GetPath() == "broken.py" {
			t.Fatalf("broken.py was excluded (%v), want it discovered — the dangling symlink must reach Extract, not be pre-filtered", x)
		}
	}

	results, err := Extract(d.Files, 1)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	found2 := false
	for _, r := range results {
		if r.RelPath != "broken.py" {
			continue
		}
		found2 = true
		if r.Err == nil {
			t.Fatal("broken.py FileResult.Err = nil, want a read error")
		}
		if !errors.Is(r.Err, fs.ErrNotExist) {
			t.Errorf("broken.py FileResult.Err = %v, want wrapping fs.ErrNotExist", r.Err)
		}
	}
	if !found2 {
		t.Fatalf("Extract results = %+v, want a broken.py result", results)
	}
}

// excludedRow is a (path, reason, detail) tuple read back through
// graphstore — the plain comparable shape both fixture tests assert
// against, independent of the proto message's other fields.
type excludedRow struct {
	path, detail string
	reason       schema.ExclusionReason
}

// readExcludedRows drains r's ExcludedFileIterator in its own iteration
// order (Pebble key order — length-prefixed segment encoding, NOT raw
// lexical path order; see keys.go's appendSegment and Plan 01's own
// TestExcludedFileNamespaceRoundTrip precedent for this exact
// distinction).
func readExcludedRows(t *testing.T, r graphstore.Reader) []excludedRow {
	t.Helper()
	xit, err := r.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	defer xit.Close()
	var rows []excludedRow
	for xit.Next() {
		x := xit.ExcludedFile()
		rows = append(rows, excludedRow{path: x.GetPath(), reason: x.GetReason(), detail: x.GetDetail()})
	}
	if err := xit.Err(); err != nil {
		t.Fatalf("IterateExcludedFiles iteration: %v", err)
	}
	return rows
}

// TestCoverageFixtureCountsAndReasons is D-13 / HLT-04 criterion 1: index
// the committed coverage fixture plus its two test-time-generated files
// through the real indexer.Run pipeline, then read back through
// graphstore ONLY (never re-deriving from the copy) and assert the exact
// counts and per-file reasons the test itself computes and logs.
func TestCoverageFixtureCountsAndReasons(t *testing.T) {
	dir := copyCoverageFixture(t)
	hugeSize := writeOversizeGoFile(t, dir, "huge.go")
	writeDanglingSymlink(t, dir, "broken.py")

	// storeDir lives OUTSIDE the copy — nesting it inside dir would make
	// decision point 1 record a phantom DIR_DOTPREFIX exclusion for
	// ".codegraph" itself, corrupting this test's exact 6-record count.
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(dir, storeDir, Options{Quiet: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHasCoverage() {
		t.Fatal("GetMeta().GetHasCoverage() = false, want true")
	}

	rows := readExcludedRows(t, snap)
	t.Logf("excluded records: %d", len(rows))
	if len(rows) != 6 {
		t.Fatalf("Excluded rows = %+v, want exactly 6", rows)
	}

	wantSizeDetail := strconv.FormatInt(hugeSize, 10) + " bytes > " + strconv.Itoa(parser.MaxSourceBytes)
	wantByPath := map[string]excludedRow{
		".hidden":   {path: ".hidden", reason: schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX, detail: ".hidden"},
		"go.mod":    {path: "go.mod", reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, detail: ".mod"},
		"huge.go":   {path: "huge.go", reason: schema.ExclusionReason_EXCLUSION_REASON_SIZE_LIMIT, detail: wantSizeDetail},
		"notes.md":  {path: "notes.md", reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, detail: ".md"},
		"tagged.go": {path: "tagged.go", reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, detail: buildTagDetail(build.Default)},
		"vendor":    {path: "vendor", reason: schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR, detail: "vendor"},
	}
	seen := make(map[string]bool, len(rows))
	for _, got := range rows {
		want, ok := wantByPath[got.path]
		if !ok {
			t.Errorf("unexpected excluded row for path %q: %+v", got.path, got)
			continue
		}
		seen[got.path] = true
		if got.reason != want.reason {
			t.Errorf("%s reason = %v, want %v", got.path, got.reason, want.reason)
		}
		if got.detail != want.detail {
			t.Errorf("%s detail = %q, want %q", got.path, got.detail, want.detail)
		}
	}
	for path := range wantByPath {
		if !seen[path] {
			t.Errorf("missing excluded row for path %q", path)
		}
	}

	fit, err := snap.IterateFiles()
	if err != nil {
		t.Fatalf("IterateFiles: %v", err)
	}
	var indexed, extractionFailed int64
	var filePaths []string
	for fit.Next() {
		f := fit.File()
		filePaths = append(filePaths, f.GetPath())
		if len(f.GetErrors()) == 0 {
			indexed++
		} else {
			extractionFailed++
		}
	}
	if err := fit.Err(); err != nil {
		t.Fatalf("IterateFiles iteration: %v", err)
	}
	fit.Close()

	if len(filePaths) != 2 {
		t.Fatalf("File records = %v, want exactly 2 (main.go, broken.py)", filePaths)
	}

	var dirLevel int64
	for _, row := range rows {
		if schema.IsDirectoryExclusion(row.reason) {
			dirLevel++
		}
	}
	fileLevel := int64(len(rows)) - dirLevel
	discovered := indexed + extractionFailed + fileLevel

	t.Logf("indexed=%d extraction_failed=%d excluded=%d file_level=%d discovered=%d", indexed, extractionFailed, int64(len(rows)), fileLevel, discovered)

	if indexed != 1 {
		t.Errorf("indexed = %d, want 1", indexed)
	}
	if extractionFailed != 1 {
		t.Errorf("extraction_failed = %d, want 1", extractionFailed)
	}
	if len(rows) != 6 {
		t.Errorf("excluded = %d, want 6", len(rows))
	}
	if fileLevel != 4 {
		t.Errorf("file_level = %d, want 4", fileLevel)
	}
	if discovered != 6 {
		t.Errorf("discovered = %d, want 6", discovered)
	}
	if discovered != indexed+extractionFailed+fileLevel {
		t.Errorf("invariant broken: discovered (%d) != indexed (%d) + extraction_failed (%d) + file_level (%d)", discovered, indexed, extractionFailed, fileLevel)
	}

	for _, p := range filePaths {
		if p == "vendor/x.go" || p == ".hidden/y.go" {
			t.Errorf("phantom File record for pruned-directory content: %s", p)
		}
	}
	for _, row := range rows {
		if row.path == "vendor/x.go" || row.path == ".hidden/y.go" {
			t.Errorf("phantom ExcludedFile record for pruned-directory content: %s", row.path)
		}
	}
}

// TestCoverageReasonsSurviveDiskMutationWithoutReindex is D-14a / HLT-05
// criterion 2: after indexing the fixture, mutate the disk WITHOUT
// re-indexing (strip tagged.go's build tag, delete notes.md) and assert
// the STORED reasons for both paths are unchanged — a query-time re-walk
// would report neither. This is the behavioural half of D-14's two
// guards (the structural half, the file-scoped source scan, lives in
// internal/query/coverage_test.go per D-14b).
func TestCoverageReasonsSurviveDiskMutationWithoutReindex(t *testing.T) {
	dir := copyCoverageFixture(t)
	writeOversizeGoFile(t, dir, "huge.go")
	writeDanglingSymlink(t, dir, "broken.py")

	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(dir, storeDir, Options{Quiet: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// mutation: re-derive rows from a walk → RED
	// Strip tagged.go's build tag and delete notes.md WITHOUT re-indexing
	// — if the store re-derived reasons from a fresh walk instead of
	// reading its own committed records, this mutation would make it
	// report tagged.go as no longer excluded and notes.md as absent.
	taggedPath := filepath.Join(dir, "tagged.go")
	if err := os.WriteFile(taggedPath, []byte("package main\n\nfunc Tagged() {}\n"), 0o644); err != nil {
		t.Fatalf("rewrite tagged.go: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "notes.md")); err != nil {
		t.Fatalf("remove notes.md: %v", err)
	}

	// Re-open the store as a fresh handle (a fresh process-equivalent) —
	// never touching the indexer again.
	store2, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open (post-mutation): %v", err)
	}
	defer store2.Close()
	snap2, err := store2.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot (post-mutation): %v", err)
	}
	defer snap2.Close()

	xit, err := snap2.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles (post-mutation): %v", err)
	}
	byPath := make(map[string]schema.ExclusionReason)
	for xit.Next() {
		x := xit.ExcludedFile()
		byPath[x.GetPath()] = x.GetReason()
	}
	if err := xit.Err(); err != nil {
		t.Fatalf("IterateExcludedFiles iteration (post-mutation): %v", err)
	}
	xit.Close()

	if got := byPath["tagged.go"]; got != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Errorf("stale tagged.go reason = %v, want BUILD_TAG to persist despite the on-disk mutation (D-14a)", got)
	}
	if got := byPath["notes.md"]; got != schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION {
		t.Errorf("stale notes.md reason = %v, want UNSUPPORTED_EXTENSION to persist despite deletion (D-14a)", got)
	}

	// The explicit contrast: a FRESH walk of the mutated disk now
	// disagrees with the stored (stale) reasons above — proving the
	// store is not silently re-deriving them at read time.
	fresh, err := DiscoverAll(dir)
	if err != nil {
		t.Fatalf("DiscoverAll (fresh, post-mutation): %v", err)
	}
	var taggedNowDiscovered bool
	for _, f := range fresh.Files {
		if f.RelPath == "tagged.go" {
			taggedNowDiscovered = true
		}
	}
	if !taggedNowDiscovered {
		t.Error("fresh DiscoverAll: tagged.go not in Files, want it discovered now that its build tag is gone")
	}
	for _, x := range fresh.Excluded {
		if x.GetPath() == "tagged.go" {
			t.Errorf("fresh DiscoverAll still excludes tagged.go (%v), want it gone from Excluded now", x)
		}
		if x.GetPath() == "notes.md" {
			t.Errorf("fresh DiscoverAll still reports notes.md (%v), want it absent (deleted from disk)", x)
		}
	}
}
