package indexer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

const buildTagFixture = "//go:build ignore\n\npackage tmp\n\nfunc Tagged() {}\n"

// TestSyncRecordsANewBuildTagExclusionWithoutAnyIndexedFileChange proves
// the coverageDirty gate: a Sync whose ONLY on-disk change is a new
// build-tag-excluded file must not take the fully-no-op early return —
// the tracer's carry-forward left this RED (the no-op path returned
// before any write ever staged the new record).
func TestSyncRecordsANewBuildTagExclusionWithoutAnyIndexedFileChange(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	rSeed, closeSeed := openSnapshot(t, storeDir)
	seedMeta, err := rSeed.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (seed): %v", err)
	}
	seedLastSync := seedMeta.GetLastSyncUnixMs()
	closeSeed()

	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	found := findExcluded(t, r, "tagged.go")
	if found == nil {
		t.Fatalf("expected a c/ record for tagged.go after Sync")
	}
	if found.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Fatalf("tagged.go reason = %v, want BUILD_TAG", found.GetReason())
	}

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHasCoverage() {
		t.Fatalf("expected HasCoverage true after recording a new exclusion")
	}
	if meta.GetLastSyncUnixMs() < seedLastSync {
		t.Fatalf("LastSyncUnixMs did not advance: seed=%d after=%d", seedLastSync, meta.GetLastSyncUnixMs())
	}
	if stats.Files != 1 {
		t.Fatalf("Stats.Files = %d, want 1 (tagged.go is excluded, not extraction-bound)", stats.Files)
	}
}

// TestSyncPrunesAnExclusionThatBecameIndexable proves the per-path prune
// side of the diff: a file that loses its build tag stops being excluded
// and its stale c/ record is removed in the same Sync that indexes it.
func TestSyncPrunesAnExclusionThatBecameIndexable(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	// Rewrite tagged.go WITHOUT the build tag: it is now indexable.
	writeFixtureFile(t, repoRoot, "tagged.go", "package tmp\n\nfunc Tagged() {}\n")

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	if found := findExcluded(t, r, "tagged.go"); found != nil {
		t.Fatalf("expected tagged.go's c/ record to be pruned, found %v", found)
	}
	if _, err := r.GetFile("tagged.go"); err != nil {
		t.Fatalf("expected a File record for tagged.go now that it is indexable: %v", err)
	}
	if stats.FilesReparsed < 1 {
		t.Fatalf("Stats.FilesReparsed = %d, want >= 1", stats.FilesReparsed)
	}

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHasCoverage() {
		t.Fatalf("expected HasCoverage true")
	}
}

// TestSyncPrunesAnExclusionWhoseFileWasDeleted proves the other prune
// trigger: the excluded file disappears from disk entirely.
func TestSyncPrunesAnExclusionWhoseFileWasDeleted(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	if err := os.Remove(filepath.Join(repoRoot, "tagged.go")); err != nil {
		t.Fatalf("os.Remove: %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	if found := findExcluded(t, r, "tagged.go"); found != nil {
		t.Fatalf("expected tagged.go's c/ record to be gone after deletion, found %v", found)
	}
	if stats.FilesPruned != 0 {
		t.Fatalf("Stats.FilesPruned = %d, want 0 (tagged.go was never a File record)", stats.FilesPruned)
	}
}

// TestSyncIsANoOpWhenNothingChangedAndCoverageIsRecorded proves the
// no-op gate still holds: a truly clean Sync with coverage already
// recorded writes nothing — same LastSyncUnixMs, same HasCoverage.
func TestSyncIsANoOpWhenNothingChangedAndCoverageIsRecorded(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	rBefore, closeBefore := openSnapshot(t, storeDir)
	before, err := rBefore.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (before): %v", err)
	}
	beforeLastSync := before.GetLastSyncUnixMs()
	beforeHasCoverage := before.GetHasCoverage()
	closeBefore()

	if !beforeHasCoverage {
		t.Fatalf("expected HasCoverage already true after Run")
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()
	after, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (after): %v", err)
	}

	if after.GetLastSyncUnixMs() != beforeLastSync {
		t.Fatalf("LastSyncUnixMs changed on a true no-op sync: before=%d after=%d", beforeLastSync, after.GetLastSyncUnixMs())
	}
	if after.GetHasCoverage() != beforeHasCoverage {
		t.Fatalf("HasCoverage changed on a true no-op sync")
	}
	if stats.FilesReparsed != 0 {
		t.Fatalf("Stats.FilesReparsed = %d, want 0", stats.FilesReparsed)
	}
}

// TestSyncBackfillsCoverageOnAGraphThatHasFileIndexButNoCoverage covers
// D-06's "known" transition for a fabricated pre-Phase-10 graph: HasFileIndex
// true, HasCoverage false/unset, empty c/ namespace. One incremental Sync
// (no disk change) must record every exclusion and stamp HasCoverage true
// via the small-commit path (no file is reparsed).
func TestSyncBackfillsCoverageOnAGraphThatHasFileIndexButNoCoverage(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	fabricatePreCoverageStore(t, storeDir)

	// Assert the fabrication is real BEFORE calling Sync.
	rPre, closePre := openSnapshot(t, storeDir)
	preMeta, err := rPre.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (pre): %v", err)
	}
	if preMeta.GetHasCoverage() {
		t.Fatalf("fabrication failed: HasCoverage already true")
	}
	if found := findExcluded(t, rPre, "tagged.go"); found != nil {
		t.Fatalf("fabrication failed: c/ namespace not empty, found %v", found)
	}
	closePre()

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	postMeta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (post): %v", err)
	}
	if !postMeta.GetHasCoverage() {
		t.Fatalf("expected HasCoverage true after backfill Sync")
	}

	found := findExcluded(t, r, "tagged.go")
	if found == nil || found.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Fatalf("expected tagged.go BUILD_TAG record after backfill, got %v", found)
	}

	if stats.FilesReparsed != 0 {
		t.Fatalf("Stats.FilesReparsed = %d, want 0 (small-commit path, no file changed)", stats.FilesReparsed)
	}
}

// TestSyncMtimeRefreshOnlyPathKeepsCoverageRecorded proves the
// mtime-refresh-only path (WR-03) never drops previously recorded
// coverage: an mtime-only touch reparses nothing, and coverage stays
// exactly as it was.
func TestSyncMtimeRefreshOnlyPathKeepsCoverageRecorded(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	future := time.Now().Add(1 * time.Hour)
	if err := os.Chtimes(filepath.Join(repoRoot, "main.go"), future, future); err != nil {
		t.Fatalf("os.Chtimes: %v", err)
	}

	stats, err := Sync(repoRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if !meta.GetHasCoverage() {
		t.Fatalf("expected HasCoverage true after mtime-only refresh")
	}
	if found := findExcluded(t, r, "tagged.go"); found == nil {
		t.Fatalf("expected tagged.go's c/ record to still be present")
	}
	if stats.FilesReparsed != 0 {
		t.Fatalf("Stats.FilesReparsed = %d, want 0 (mtime-only refresh reparses nothing)", stats.FilesReparsed)
	}
}

// TestSyncFullBackfillStampsBothFlags covers the OTHER backfill trigger
// (needsFileIndexBackfill: HasFileIndex false) — Sync delegates to the
// from-scratch run()/writeGraph path, which range-deletes then rewrites
// the c/ namespace cleanly (no duplicate, no leftover) and stamps both
// flags in the same commit.
func TestSyncFullBackfillStampsBothFlags(t *testing.T) {
	repoRoot := writeFixture(t, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})
	writeFixtureFile(t, repoRoot, "tagged.go", buildTagFixture)
	storeDir := t.TempDir()

	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run (seed): %v", err)
	}

	fabricateNoFileIndexStore(t, storeDir)

	if _, err := Sync(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	r, closeAll := openSnapshot(t, storeDir)
	defer closeAll()

	postMeta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta (post): %v", err)
	}
	if !postMeta.GetHasFileIndex() {
		t.Fatalf("expected HasFileIndex true after full backfill")
	}
	if !postMeta.GetHasCoverage() {
		t.Fatalf("expected HasCoverage true after full backfill")
	}

	count := 0
	it, err := r.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	defer it.Close()
	for it.Next() {
		if it.ExcludedFile().GetPath() == "tagged.go" {
			count++
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterate err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one tagged.go exclusion record after full backfill, got %d", count)
	}
}

// findExcluded returns the c/ record at path, or nil if absent.
func findExcluded(t *testing.T, r graphstore.Reader, path string) *schema.ExcludedFile {
	t.Helper()
	it, err := r.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	defer it.Close()
	var found *schema.ExcludedFile
	for it.Next() {
		if it.ExcludedFile().GetPath() == path {
			found = it.ExcludedFile()
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterate err: %v", err)
	}
	return found
}

// fabricatePreCoverageStore rewrites storeDir's Meta with HasCoverage
// forced false and clears the WHOLE c/ namespace, mimicking a
// pre-Phase-10 graph that already has a file index (HasFileIndex stays
// whatever the seed Run left it, i.e. true) but no exclusion records —
// Task 2's fabrication step, sharing Task 1's scaffolding.
func fabricatePreCoverageStore(t *testing.T, storeDir string) {
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
	seedMeta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	r.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.DeleteAllExcludedFiles(); err != nil {
		w.Close()
		t.Fatalf("DeleteAllExcludedFiles: %v", err)
	}
	fabricated := proto.Clone(seedMeta).(*schema.Meta)
	fabricated.HasCoverage = false
	if err := w.PutMeta(fabricated); err != nil {
		w.Close()
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// fabricateNoFileIndexStore rewrites storeDir's Meta with HasFileIndex and
// HasCoverage both false, WITHOUT clearing the c/ namespace — this is the
// OTHER backfill trigger (needsFileIndexBackfill), which delegates to the
// from-scratch run()/writeGraph path; that path range-deletes the c/
// namespace itself, so a leftover pre-fabrication record here proves the
// from-scratch rewrite is genuinely clean (no duplicate survives).
func fabricateNoFileIndexStore(t *testing.T, storeDir string) {
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
	seedMeta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	r.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	fabricated := proto.Clone(seedMeta).(*schema.Meta)
	fabricated.HasFileIndex = false
	fabricated.HasCoverage = false
	if err := w.PutMeta(fabricated); err != nil {
		w.Close()
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}
