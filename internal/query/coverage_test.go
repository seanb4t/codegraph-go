package query

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// buildTagRepoFixture writes a temp repo with go.mod, main.go, and
// tagged.go (a //go:build ignore file) and returns its root — the
// tracer's own minimal fixture, one file of exactly the ONE reason this
// plan records (BUILD_TAG).
func buildTagRepoFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "go.mod", "module example.com/coveragefixture\n\ngo 1.24\n")
	writeFixture(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeFixture(t, root, "tagged.go", "//go:build ignore\n\npackage main\n\nfunc Tagged() {}\n")
	return root
}

func writeFixture(t *testing.T, root, relPath, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	if err := os.WriteFile(abs, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}

// indexCoverageFixture indexes root via a real indexer.Run into
// root/.codegraph/store and returns an opened Reader over that store, its
// own close func, and an *Engine wrapping it. The store directory is
// deliberately NOT pre-created (Phase 10 Plan 2: indexer.Run's own
// DiscoverAll walk runs BEFORE graphstore.Open creates storeDir, exactly
// mirroring production's first-index ordering — pre-creating it here
// would make decision point 1 record a phantom DIR_DOTPREFIX exclusion
// for ".codegraph" that a real from-scratch index never sees).
func indexCoverageFixture(t *testing.T, root string) (*Engine, func()) {
	t.Helper()
	storeDir := filepath.Join(root, ".codegraph", "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	closer := func() {
		_ = snap.Close()
		_ = store.Close()
	}
	return NewWithRoot(snap, root), closer
}

// TestCoverageSummaryAfterRunOnBuildTagRepo indexes a temp repo with one
// //go:build ignore file via a real indexer.Run and proves
// (*Engine).CoverageSummary reports it (Phase 10 HLT-05).
func TestCoverageSummaryAfterRunOnBuildTagRepo(t *testing.T) {
	root := buildTagRepoFixture(t)
	eng, closer := indexCoverageFixture(t, root)
	defer closer()

	summary, err := eng.CoverageSummary()
	if err != nil {
		t.Fatalf("CoverageSummary: %v", err)
	}
	if !summary.Known {
		t.Fatal("Known = false, want true")
	}
	if summary.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1", summary.Indexed)
	}
	if summary.ExtractionFailed != 0 {
		t.Errorf("ExtractionFailed = %d, want 0", summary.ExtractionFailed)
	}
	if got := summary.ExcludedByReason["EXCLUSION_REASON_BUILD_TAG"]; got != 1 {
		t.Errorf("ExcludedByReason[BUILD_TAG] = %d, want 1", got)
	}
	// Invariant assertion (never a hard-coded total, since later plans
	// add more reasons to the same walk): Discovered == Indexed +
	// ExtractionFailed + (file-level excluded).
	if summary.Discovered != summary.Indexed+summary.ExtractionFailed+summary.Excluded {
		t.Errorf("Discovered (%d) != Indexed (%d) + ExtractionFailed (%d) + Excluded (%d)", summary.Discovered, summary.Indexed, summary.ExtractionFailed, summary.Excluded)
	}
}

// TestCoverageSummaryOnOldGraphIsUnknown proves a store whose Meta was
// built WITHOUT HasCoverage (D-15) reports Known == false with every
// count zero and a nil by-reason map — never an error, never 0/0 read as
// "known".
//
// mutation: treat unset as zero
func TestCoverageSummaryOnOldGraphIsUnknown(t *testing.T) {
	t.Run("meta present but has_coverage unset", func(t *testing.T) {
		store, err := graphstore.Open(t.TempDir())
		if err != nil {
			t.Fatalf("graphstore.Open: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })

		w, err := store.NewWriter()
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		meta := schema.NewMeta()
		meta.NodeCount = 3
		meta.HasFileIndex = true
		if err := w.PutFile(&schema.File{Path: "a.go", Language: "go"}); err != nil {
			t.Fatalf("PutFile: %v", err)
		}
		if err := w.PutMeta(meta); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}
		if err := w.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}

		snap, err := store.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		t.Cleanup(func() { _ = snap.Close() })

		summary, err := New(snap).CoverageSummary()
		if err != nil {
			t.Fatalf("CoverageSummary: %v", err)
		}
		assertUnknownCoverageSummary(t, summary)
	})

	t.Run("no meta record at all", func(t *testing.T) {
		store, err := graphstore.Open(t.TempDir())
		if err != nil {
			t.Fatalf("graphstore.Open: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })

		snap, err := store.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		t.Cleanup(func() { _ = snap.Close() })

		summary, err := New(snap).CoverageSummary()
		if err != nil {
			t.Fatalf("CoverageSummary: %v", err)
		}
		assertUnknownCoverageSummary(t, summary)
	})
}

func assertUnknownCoverageSummary(t *testing.T, summary CoverageSummary) {
	t.Helper()
	if summary.Known {
		t.Error("Known = true, want false")
	}
	if summary.Discovered != 0 || summary.Indexed != 0 || summary.Excluded != 0 || summary.ExtractionFailed != 0 {
		t.Errorf("counts = %+v, want all zero", summary)
	}
	if summary.ExcludedByReason != nil {
		t.Errorf("ExcludedByReason = %v, want nil", summary.ExcludedByReason)
	}
}

// TestCoverageRowsUnknownGraphAndSingleRow proves CoverageRows follows
// the same Known contract as CoverageSummary, and returns exactly one row
// for the tagged repo.
func TestCoverageRowsUnknownGraphAndSingleRow(t *testing.T) {
	t.Run("old graph", func(t *testing.T) {
		store, err := graphstore.Open(t.TempDir())
		if err != nil {
			t.Fatalf("graphstore.Open: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })

		snap, err := store.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		t.Cleanup(func() { _ = snap.Close() })

		page, err := New(snap).CoverageRows(CoverageRowsOptions{})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if page.Known {
			t.Error("Known = true, want false")
		}
		if len(page.Rows) != 0 {
			t.Errorf("Rows = %+v, want empty", page.Rows)
		}
		if page.NextPageToken != "" {
			t.Errorf("NextPageToken = %q, want empty", page.NextPageToken)
		}
	})

	t.Run("tagged repo two rows", func(t *testing.T) {
		root := buildTagRepoFixture(t)
		eng, closer := indexCoverageFixture(t, root)
		defer closer()

		page, err := eng.CoverageRows(CoverageRowsOptions{})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if !page.Known {
			t.Fatal("Known = false, want true")
		}
		// Phase 10 Plan 2 also records go.mod itself as an
		// UNSUPPORTED_EXTENSION exclusion (decision point 2) — this
		// fixture now yields both records, not just tagged.go's.
		if len(page.Rows) != 2 {
			t.Fatalf("Rows = %+v, want exactly 2", page.Rows)
		}
		byPath := make(map[string]CoverageRow, len(page.Rows))
		for _, r := range page.Rows {
			byPath[r.Path] = r
		}
		taggedRow, ok := byPath["tagged.go"]
		if !ok {
			t.Fatalf("Rows = %+v, want a tagged.go row", page.Rows)
		}
		if taggedRow.Kind != CoverageRowExcluded {
			t.Errorf("tagged.go Kind = %v, want CoverageRowExcluded", taggedRow.Kind)
		}
		if taggedRow.Reason != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
			t.Errorf("tagged.go Reason = %v, want EXCLUSION_REASON_BUILD_TAG", taggedRow.Reason)
		}
		goModRow, ok := byPath["go.mod"]
		if !ok {
			t.Fatalf("Rows = %+v, want a go.mod row", page.Rows)
		}
		if goModRow.Reason != schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION {
			t.Errorf("go.mod Reason = %v, want EXCLUSION_REASON_UNSUPPORTED_EXTENSION", goModRow.Reason)
		}
		if page.NextPageToken != "" {
			t.Errorf("NextPageToken = %q, want empty", page.NextPageToken)
		}
	})
}

// TestCoverageSourceNeverWalksDisk is the D-14b structural guard: reads
// coverage.go's own bytes and asserts zero occurrences of any
// filesystem-walking/reading call, with a positive control
// (IterateExcludedFiles, GetHasCoverage) proving the scan actually
// inspected the expected source (rule 84d1gfpywd) — mirroring
// internal/cli/editordiscovery_test.go's
// TestEditorDiscoverySourceNeverSpawnsAProcess shape.
func TestCoverageSourceNeverWalksDisk(t *testing.T) {
	src, err := os.ReadFile("coverage.go")
	if err != nil {
		t.Fatalf("read coverage.go: %v", err)
	}
	text := string(src)

	forbidden := []string{"filepath.WalkDir(", "os.ReadDir(", "os.Stat(", "os.Lstat(", "os.ReadFile(", "os.Open("}
	for _, f := range forbidden {
		if strings.Contains(text, f) {
			t.Fatalf("coverage.go contains %q — D-14 forbids any filesystem call: reasons are read back through graphstore, never reconstructed by a query-time walk", f)
		}
	}

	required := []string{"IterateExcludedFiles", "GetHasCoverage"}
	inspected := 0
	for _, r := range required {
		count := strings.Count(text, r)
		if count == 0 {
			t.Fatalf("coverage.go does not contain %q — positive control failed, meaning this scan is not actually inspecting the expected source", r)
		}
		inspected += count
	}
	t.Logf("inspected coverage.go: 0 forbidden filesystem calls, %d positive-control occurrences", inspected)
	if inspected == 0 {
		t.Fatal("inspected 0 positive-control occurrences — this guard is vacuous")
	}
}

// copyCoverageFixtureForQuery materializes an independent copy of the
// Phase 10 D-13 committed fixture (internal/indexer/testdata/coverage)
// under a fresh t.TempDir() — reproduced from
// internal/indexer/coverage_fixture_test.go's copyCoverageFixture (which
// itself thin-wraps copyFixture in that package) rather than imported,
// since it is unexported test code in another package. Mirrors this
// file's own copyFixture/indexFixture convention (engine_test.go) of
// reproducing rather than importing across package boundaries.
func copyCoverageFixtureForQuery(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "coverage"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
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
		t.Fatalf("copy coverage fixture: %v", err)
	}
	return dst
}

// writeOversizeGoFileForQuery writes a syntactically valid,
// build-tag-free Go source file at exactly parser.MaxSourceBytes+1 bytes
// into dir/name — the SIZE_LIMIT trigger, generated fresh per test.
// Reproduced from internal/indexer/coverage_fixture_test.go's
// writeOversizeGoFile / discoverexclusion_test.go's writeExactSizeGoFile
// (unexported test code in another package).
func writeOversizeGoFileForQuery(t *testing.T, dir, name string) int64 {
	t.Helper()
	size := int64(parser.MaxSourceBytes + 1)
	header := []byte("package tmp\n\nfunc Placeholder() {}\n\n// ")
	if int64(len(header))+1 > size {
		t.Fatalf("requested size %d too small for header+trailer (%d)", size, len(header)+1)
	}
	buf := make([]byte, size)
	copy(buf, header)
	for i := len(header); i < len(buf)-1; i++ {
		buf[i] = 'x'
	}
	buf[len(buf)-1] = '\n'
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return size
}

// writeDanglingSymlinkForQuery creates a symlink at dir/name pointing at
// a nonexistent target — the extraction-failure trigger (RESEARCH.md
// Pitfall 4 / Assumption A1), reproduced from
// internal/indexer/coverage_fixture_test.go's writeDanglingSymlink.
func writeDanglingSymlinkForQuery(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Symlink("missing-target.py", filepath.Join(dir, name)); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}
}

// indexCoverageFixtureExternal indexes root (already carrying the
// generated huge.go/broken.py on top of the committed fixture) into a
// store OUTSIDE root — nesting it inside root would make the ".codegraph"
// directory itself a phantom DIR_DOTPREFIX exclusion, corrupting the
// exact discovered=6/excluded=6 counts this fixture is pinned to (see
// TestCoverageFixtureCountsAndReasons's identical rationale in the
// indexer package). Returns an *Engine over a fresh snapshot and a
// closer.
func indexCoverageFixtureExternal(t *testing.T, root string) (*Engine, func()) {
	t.Helper()
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	closer := func() {
		_ = snap.Close()
		_ = store.Close()
	}
	return NewWithRoot(snap, root), closer
}

// TestCoverageSummaryMatchesFixture is HLT-05 pinned against the full
// D-13 fixture (Plan 02's six committed files plus the generated
// oversize file and dangling symlink), store OUTSIDE the repo copy: the
// exact discovered/indexed/excluded/extraction_failed counts and the
// full by-reason breakdown, plus the additive invariant re-derived from
// the reported values (never hard-coded).
func TestCoverageSummaryMatchesFixture(t *testing.T) {
	root := copyCoverageFixtureForQuery(t)
	writeOversizeGoFileForQuery(t, root, "huge.go")
	writeDanglingSymlinkForQuery(t, root, "broken.py")

	eng, closer := indexCoverageFixtureExternal(t, root)
	defer closer()

	summary, err := eng.CoverageSummary()
	if err != nil {
		t.Fatalf("CoverageSummary: %v", err)
	}
	if !summary.Known {
		t.Fatal("Known = false, want true")
	}
	t.Logf("discovered=%d indexed=%d excluded=%d extraction_failed=%d", summary.Discovered, summary.Indexed, summary.Excluded, summary.ExtractionFailed)
	if summary.Discovered != 6 || summary.Indexed != 1 || summary.Excluded != 6 || summary.ExtractionFailed != 1 {
		t.Fatalf("summary = %+v, want discovered=6 indexed=1 excluded=6 extraction_failed=1", summary)
	}
	wantByReason := map[string]int64{
		"EXCLUSION_REASON_UNSUPPORTED_EXTENSION": 2,
		"EXCLUSION_REASON_DIR_DOTPREFIX":         1,
		"EXCLUSION_REASON_DIR_VENDOR":            1,
		"EXCLUSION_REASON_BUILD_TAG":             1,
		"EXCLUSION_REASON_SIZE_LIMIT":            1,
	}
	if len(summary.ExcludedByReason) != len(wantByReason) {
		t.Errorf("ExcludedByReason has %d keys, want %d: %+v", len(summary.ExcludedByReason), len(wantByReason), summary.ExcludedByReason)
	}
	for reason, want := range wantByReason {
		if got := summary.ExcludedByReason[reason]; got != want {
			t.Errorf("ExcludedByReason[%s] = %d, want %d", reason, got, want)
		}
	}
	dirLevel := summary.ExcludedByReason["EXCLUSION_REASON_DIR_VENDOR"] + summary.ExcludedByReason["EXCLUSION_REASON_DIR_DOTPREFIX"]
	if want := summary.Indexed + summary.ExtractionFailed + (summary.Excluded - dirLevel); summary.Discovered != want {
		t.Errorf("invariant broken: Discovered (%d) != Indexed+ExtractionFailed+(Excluded-dirLevel) (%d)", summary.Discovered, want)
	}
}

// TestCoverageRowsOrderingAndPagingAreStable is HLT-06's ordering
// criterion: CoverageRows walks two segments — every extraction-failed
// File record, then every ExcludedFile record — each in the STORE's own
// key order (a length-prefixed encoding, never plain lexical path
// order; see coverage.go's CoverageRows doc comment). A single
// PageSize:1000 call is compared byte-for-byte against the concatenation
// of PageSize:3 pages, and replaying a page token twice must yield
// reflect.DeepEqual results — the property this test actually needs,
// independent of which literal order the store happens to produce.
func TestCoverageRowsOrderingAndPagingAreStable(t *testing.T) {
	root := copyCoverageFixtureForQuery(t)
	writeOversizeGoFileForQuery(t, root, "huge.go")
	writeDanglingSymlinkForQuery(t, root, "broken.py")

	eng, closer := indexCoverageFixtureExternal(t, root)
	defer closer()

	full, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 1000})
	if err != nil {
		t.Fatalf("CoverageRows (single page): %v", err)
	}
	if !full.Known {
		t.Fatal("Known = false, want true")
	}
	if full.NextPageToken != "" {
		t.Errorf("single-page NextPageToken = %q, want empty", full.NextPageToken)
	}
	if len(full.Rows) != 7 {
		t.Fatalf("single-page Rows has %d entries, want exactly 7: %+v", len(full.Rows), full.Rows)
	}
	if full.Rows[0].Kind != CoverageRowExtractionFailed || full.Rows[0].Path != "broken.py" {
		t.Fatalf("Rows[0] = %+v, want broken.py (CoverageRowExtractionFailed) first", full.Rows[0])
	}
	for _, r := range full.Rows[1:] {
		if r.Kind != CoverageRowExcluded {
			t.Fatalf("Rows = %+v, want every row after index 0 to be CoverageRowExcluded", full.Rows)
		}
	}
	t.Logf("single page order: %v", pathsOf(full.Rows))

	// Paginate in chunks of 3 and concatenate.
	var paged []CoverageRow
	var tok string
	seenTokens := map[string]bool{}
	for {
		page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 3, PageToken: tok})
		if err != nil {
			t.Fatalf("CoverageRows (page, token=%q): %v", tok, err)
		}
		paged = append(paged, page.Rows...)
		if page.NextPageToken != "" {
			if seenTokens[page.NextPageToken] {
				t.Fatalf("token %q repeated — paging is not making forward progress", page.NextPageToken)
			}
			seenTokens[page.NextPageToken] = true
			for _, r := range page.Rows {
				if strings.Contains(page.NextPageToken, r.Path) {
					t.Errorf("token %q contains row path %q verbatim — tokens must be opaque", page.NextPageToken, r.Path)
				}
			}
		}
		if page.NextPageToken == "" {
			break
		}
		tok = page.NextPageToken
	}
	if !reflect.DeepEqual(paged, full.Rows) {
		t.Fatalf("paged concatenation = %v, want equal to single-page result %v", pathsOf(paged), pathsOf(full.Rows))
	}

	// Replaying the second page's token twice yields identical pages.
	page2First, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 3})
	if err != nil {
		t.Fatalf("CoverageRows (page 1 for replay): %v", err)
	}
	replayA, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 3, PageToken: page2First.NextPageToken})
	if err != nil {
		t.Fatalf("CoverageRows (replay A): %v", err)
	}
	replayB, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 3, PageToken: page2First.NextPageToken})
	if err != nil {
		t.Fatalf("CoverageRows (replay B): %v", err)
	}
	if !reflect.DeepEqual(replayA, replayB) {
		t.Fatalf("replaying the same token twice produced different pages: %+v vs %+v", replayA, replayB)
	}
}

// pathsOf extracts each row's Path, for compact log/failure output.
func pathsOf(rows []CoverageRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Path
	}
	return out
}

// TestCoverageRowsReasonFilter proves a Reason filter restricts
// CoverageRows to ExcludedFile records with exactly that reason —
// extraction-failed rows never leak into a filtered listing.
func TestCoverageRowsReasonFilter(t *testing.T) {
	root := copyCoverageFixtureForQuery(t)
	writeOversizeGoFileForQuery(t, root, "huge.go")
	writeDanglingSymlinkForQuery(t, root, "broken.py")

	eng, closer := indexCoverageFixtureExternal(t, root)
	defer closer()

	t.Run("UNSUPPORTED_EXTENSION", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		got := pathsOf(page.Rows)
		want := []string{"go.mod", "notes.md"}
		if !reflect.DeepEqual(sortedCopy(got), sortedCopy(want)) {
			t.Fatalf("Rows = %v, want exactly %v (order-independent)", got, want)
		}
		for _, r := range page.Rows {
			if r.Kind != CoverageRowExcluded {
				t.Errorf("row %+v has Kind %v, want CoverageRowExcluded (no extraction-failed leakage)", r, r.Kind)
			}
		}
	})

	t.Run("DIR_VENDOR", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{Reason: schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if got := pathsOf(page.Rows); !reflect.DeepEqual(got, []string{"vendor"}) {
			t.Fatalf("Rows = %v, want exactly [vendor]", got)
		}
	})

	t.Run("SIZE_LIMIT", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{Reason: schema.ExclusionReason_EXCLUSION_REASON_SIZE_LIMIT})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if len(page.Rows) != 1 || page.Rows[0].Path != "huge.go" {
			t.Fatalf("Rows = %+v, want exactly one huge.go row", page.Rows)
		}
		if !strings.HasPrefix(page.Rows[0].Detail, "4194305 bytes") {
			t.Errorf("huge.go Detail = %q, want prefix %q", page.Rows[0].Detail, "4194305 bytes")
		}
	})
}

// sortedCopy returns a sorted copy of ss for order-independent slice
// comparison.
func sortedCopy(ss []string) []string {
	out := append([]string(nil), ss...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// TestCoverageRowsRejectsMalformedTokens is T-10-04's closed refusal
// table: every malformed page-token shape is rejected with
// ErrInvalidArgument, never a panic and never a silent restart from page
// one — plus a positive control proving a valid token IS accepted.
func TestCoverageRowsRejectsMalformedTokens(t *testing.T) {
	root := copyCoverageFixtureForQuery(t)
	writeOversizeGoFileForQuery(t, root, "huge.go")
	writeDanglingSymlinkForQuery(t, root, "broken.py")

	eng, closer := indexCoverageFixtureExternal(t, root)
	defer closer()

	validXToken := encodeCoverageToken('x', "go.mod")

	cases := []struct {
		name  string
		token string
		opts  CoverageRowsOptions
	}{
		// "not base64" — characters outside the base64url alphabet, so
		// decoding itself fails.
		{"not base64", "not*base64*", CoverageRowsOptions{}},
		// A valid raw-url payload corrupted with trailing '=' padding:
		// RawURLEncoding.DecodeString rejects '=' outright, so decoding
		// fails via a DIFFERENT path than the invalid-character case
		// above — note this is NOT "base64url of the empty string":
		// encoding zero bytes always round-trips to "", which
		// decodeCoverageToken deliberately special-cases as "first page"
		// (its own doc comment), so that literal shape is valid input,
		// not malformed.
		{"padded (non-raw-url) base64", base64URLNoPad("x"+"go.mod") + "==", CoverageRowsOptions{}},
		{"unknown kind byte", base64URLNoPad("q" + "path"), CoverageRowsOptions{}},
		{"oversized path", base64URLNoPad("x" + strings.Repeat("a", 4097)), CoverageRowsOptions{}},
		{"invalid utf8", base64URLNoPad("x\xff\xfe"), CoverageRowsOptions{}},
		{"f-token with reason filter", encodeCoverageToken('f', "broken.py"), CoverageRowsOptions{Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION}},
	}
	inspected := 0
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inspected++
			opts := c.opts
			opts.PageToken = c.token
			_, err := eng.CoverageRows(opts)
			if err == nil {
				t.Fatalf("CoverageRows(token=%q): got nil error, want ErrInvalidArgument", c.token)
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("CoverageRows(token=%q): err = %v, want errors.Is(err, ErrInvalidArgument)", c.token, err)
			}
		})
	}
	t.Logf("malformed tokens inspected: %d", inspected)
	if inspected < 6 {
		t.Fatalf("inspected %d malformed token shapes, want at least 6", inspected)
	}

	// Positive control: a valid token is accepted with nil error.
	page, err := eng.CoverageRows(CoverageRowsOptions{PageToken: validXToken})
	if err != nil {
		t.Fatalf("CoverageRows (positive control, valid token): %v", err)
	}
	if !page.Known {
		t.Fatal("positive control: Known = false, want true")
	}
	for _, r := range page.Rows {
		if r.Path == "go.mod" {
			t.Errorf("positive control: rows still contain go.mod (the cursor row itself), want it excluded: %v", pathsOf(page.Rows))
		}
	}
}

// base64URLNoPad base64url-encodes s without padding, matching
// encodeCoverageToken's own encoding (base64.RawURLEncoding) — a test
// helper for constructing malformed token payloads by hand.
func base64URLNoPad(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// TestCoverageRowsPageSizeClamp is T-10-03's DoS mitigation: a
// caller-supplied PageSize is clamped into [1, CoverageMaxPageSize], and
// a non-positive value falls back to CoverageDefaultPageSize.
func TestCoverageRowsPageSizeClamp(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "main.go", "package main\n\nfunc main() {}\n")
	for i := 0; i < 1100; i++ {
		writeFixture(t, root, fmtPaddedName(i), "# generated\n")
	}

	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
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
	eng := NewWithRoot(snap, root)

	t.Run("oversized clamps to max", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 5000})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if len(page.Rows) != 1000 {
			t.Errorf("Rows = %d, want exactly 1000", len(page.Rows))
		}
		if page.NextPageToken == "" {
			t.Error("NextPageToken is empty, want non-empty (1100 > 1000)")
		}
	})

	t.Run("zero falls back to default", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 0})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if len(page.Rows) != 200 {
			t.Errorf("Rows = %d, want exactly 200", len(page.Rows))
		}
	})

	t.Run("negative falls back to default", func(t *testing.T) {
		page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: -7})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if len(page.Rows) != 200 {
			t.Errorf("Rows = %d, want exactly 200", len(page.Rows))
		}
	})
}

// fmtPaddedName returns a deterministic, zero-padded markdown filename
// (n0000.md .. n1099.md) so the fixture's 1100 generated files sort and
// print predictably.
func fmtPaddedName(i int) string {
	digits := "0123456789"
	b := []byte("n0000.md")
	// b[1:5] holds the 4 zero-padded digits; overwrite from the least
	// significant digit outward.
	for pos := 4; pos >= 1; pos-- {
		b[pos] = digits[i%10]
		i /= 10
	}
	return string(b)
}

// TestCoverageRowsEmptyPageOnKnownGraphWithZeroExclusions is D-15's
// contrast case: a KNOWN graph (has_coverage true) with zero exclusions
// answers Known true, zero rows, empty token — never confused with the
// old-graph Known-false case already pinned by
// TestCoverageRowsUnknownGraphAndSingleRow's "old graph" subtest.
func TestCoverageRowsEmptyPageOnKnownGraphWithZeroExclusions(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "main.go", "package main\n\nfunc main() {}\n")

	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
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
	eng := NewWithRoot(snap, root)

	summary, err := eng.CoverageSummary()
	if err != nil {
		t.Fatalf("CoverageSummary: %v", err)
	}
	if !summary.Known {
		t.Fatal("Known = false, want true")
	}
	if summary.Discovered != 1 || summary.Indexed != 1 || summary.Excluded != 0 || summary.ExtractionFailed != 0 {
		t.Fatalf("summary = %+v, want discovered=1 indexed=1 excluded=0 extraction_failed=0", summary)
	}
	if len(summary.ExcludedByReason) != 0 {
		t.Errorf("ExcludedByReason = %v, want empty", summary.ExcludedByReason)
	}

	page, err := eng.CoverageRows(CoverageRowsOptions{})
	if err != nil {
		t.Fatalf("CoverageRows: %v", err)
	}
	if !page.Known {
		t.Fatal("CoverageRows Known = false, want true")
	}
	if len(page.Rows) != 0 {
		t.Errorf("Rows = %+v, want empty", page.Rows)
	}
	if page.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty", page.NextPageToken)
	}
}

// TestCoverageRowsDetailIsScrubbedAndBounded is T-10-05: an
// extraction-failed row's Detail never carries the absolute repository
// root, and every row's Detail is at most coverageDetailMaxBytes bytes
// of valid UTF-8.
func TestCoverageRowsDetailIsScrubbedAndBounded(t *testing.T) {
	t.Run("fixture broken.py: absolute path scrubbed to a relative form", func(t *testing.T) {
		root := copyCoverageFixtureForQuery(t)
		writeOversizeGoFileForQuery(t, root, "huge.go")
		writeDanglingSymlinkForQuery(t, root, "broken.py")

		eng, closer := indexCoverageFixtureExternal(t, root)
		defer closer()

		page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 1000})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		var brokenRow *CoverageRow
		for i := range page.Rows {
			if page.Rows[i].Path == "broken.py" {
				brokenRow = &page.Rows[i]
			}
		}
		if brokenRow == nil {
			t.Fatalf("Rows = %+v, want a broken.py row", page.Rows)
		}
		if strings.Contains(brokenRow.Detail, root) {
			t.Errorf("broken.py Detail = %q, must not contain the absolute repo root %q", brokenRow.Detail, root)
		}
		if !strings.Contains(brokenRow.Detail, "./broken.py") {
			t.Errorf("broken.py Detail = %q, want it to contain %q", brokenRow.Detail, "./broken.py")
		}
	})

	t.Run("fabricated long detail: bounded and valid UTF-8", func(t *testing.T) {
		store, err := graphstore.Open(t.TempDir())
		if err != nil {
			t.Fatalf("graphstore.Open: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })

		w, err := store.NewWriter()
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		if err := w.PutFile(&schema.File{Path: "long.py", Language: "python", Errors: []string{strings.Repeat("é", 600)}}); err != nil {
			t.Fatalf("PutFile: %v", err)
		}
		meta := schema.NewMeta()
		meta.HasCoverage = true
		if err := w.PutMeta(meta); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}
		if err := w.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}

		snap, err := store.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		t.Cleanup(func() { _ = snap.Close() })

		page, err := New(snap).CoverageRows(CoverageRowsOptions{})
		if err != nil {
			t.Fatalf("CoverageRows: %v", err)
		}
		if len(page.Rows) != 1 {
			t.Fatalf("Rows = %+v, want exactly 1", page.Rows)
		}
		detail := page.Rows[0].Detail
		if len(detail) > 256 {
			t.Errorf("Detail is %d bytes, want <= 256", len(detail))
		}
		if !utf8.ValidString(detail) {
			t.Errorf("Detail %q is not valid UTF-8", detail)
		}
	})
}

// TestCoverageRowsSurviveDiskMutationWithoutReindex is the Engine-level
// twin of Plan 02's indexer-level D-14a test (D-14's behavioural guard):
// after indexing the fixture, the disk is mutated WITHOUT re-indexing —
// mutation: filter rows by present-tense disk state → RED — and the
// STORED reasons for both affected paths must still be reported
// unchanged by CoverageRows/CoverageSummary. A reader that re-derived
// reasons from a fresh walk at query time would report tagged.go as no
// longer excluded and drop notes.md entirely.
func TestCoverageRowsSurviveDiskMutationWithoutReindex(t *testing.T) {
	root := copyCoverageFixtureForQuery(t)
	writeOversizeGoFileForQuery(t, root, "huge.go")
	writeDanglingSymlinkForQuery(t, root, "broken.py")

	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "tagged.go"), []byte("package main\n\nfunc Tagged() {}\n"), 0o644); err != nil {
		t.Fatalf("rewrite tagged.go: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "notes.md")); err != nil {
		t.Fatalf("remove notes.md: %v", err)
	}

	// Re-open the store as a fresh handle — never touching the indexer
	// again after the disk mutation above.
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

	eng := NewWithRoot(snap2, root)

	page, err := eng.CoverageRows(CoverageRowsOptions{PageSize: 1000})
	if err != nil {
		t.Fatalf("CoverageRows (post-mutation): %v", err)
	}
	byPath := make(map[string]CoverageRow, len(page.Rows))
	for _, r := range page.Rows {
		byPath[r.Path] = r
	}
	taggedRow, ok := byPath["tagged.go"]
	if !ok {
		t.Fatalf("Rows = %+v, want a stale tagged.go row despite the on-disk mutation (D-14a)", page.Rows)
	}
	if taggedRow.Reason != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Errorf("stale tagged.go Reason = %v, want BUILD_TAG to persist", taggedRow.Reason)
	}
	notesRow, ok := byPath["notes.md"]
	if !ok {
		t.Fatalf("Rows = %+v, want a stale notes.md row despite deletion from disk (D-14a)", page.Rows)
	}
	if notesRow.Reason != schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION {
		t.Errorf("stale notes.md Reason = %v, want UNSUPPORTED_EXTENSION to persist", notesRow.Reason)
	}

	summary, err := eng.CoverageSummary()
	if err != nil {
		t.Fatalf("CoverageSummary (post-mutation): %v", err)
	}
	if summary.Excluded != 6 {
		t.Errorf("post-mutation Excluded = %d, want 6 (stale count unchanged by disk mutation)", summary.Excluded)
	}
}
