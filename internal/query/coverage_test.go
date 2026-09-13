package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
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
