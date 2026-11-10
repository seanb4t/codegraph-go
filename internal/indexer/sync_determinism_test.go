package indexer

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
)

// touchEveryFile appends a no-op trailing comment to every discovered .go
// file under root. This changes each file's content hash — forcing Sync's
// D-01a stat+content-hash diff to classify EVERY file "modified" and drive
// it through the real prune+re-extract+re-resolve+write path — without
// altering any extracted symbol's identity tuple (kind, qualifiedName,
// filePath), since nodeid.NodeID's whole preimage (Phase-2 D-02a) is built
// from that tuple alone, never source content. A byte-for-byte-identical
// rewrite would instead be filtered out entirely by the content-hash
// confirm step (D-01a) and Sync would see it as unchanged.
func touchEveryFile(t *testing.T, root string) {
	t.Helper()
	files, _, err := discover.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, f := range files {
		data, err := os.ReadFile(f.AbsPath)
		if err != nil {
			t.Fatalf("read %s: %v", f.AbsPath, err)
		}
		writeFile(t, f.AbsPath, string(data)+"\n// touch\n")
	}
}

// TestSyncEqualsReindex proves INDX-03's determinism bar survives
// incremental update (04-CONTEXT.md Specific Ideas): an incremental Sync
// that reparses every file lands a graph byte-identical to a from-scratch
// Run over the same final tree, after normalizing Meta.last_sync_unix_ms —
// the one volatile field (INDX-02 precedent). GOMAXPROCS is forced high,
// matching TestDeterministicRebuild's rigor, so residual goroutine/map-order
// nondeterminism in Sync's own write path fails loudly here.
func TestSyncEqualsReindex(t *testing.T) {
	prev := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

	repoRoot := copyFixture(t, fixtureRoot)

	storeDirA := t.TempDir()
	if _, err := Sync(repoRoot, storeDirA, Options{}); err != nil {
		t.Fatalf("Sync (seed/backfill): %v", err)
	}

	touchEveryFile(t, repoRoot)

	files, _, err := discover.Discover(repoRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	statsTouch, err := Sync(repoRoot, storeDirA, Options{})
	if err != nil {
		t.Fatalf("Sync (touch-all): %v", err)
	}
	if statsTouch.FilesReparsed != len(files) {
		t.Fatalf("Stats.FilesReparsed = %d, want %d (every file touched)", statsTouch.FilesReparsed, len(files))
	}

	syncExport := exportNormalized(t, storeDirA, true)

	storeDirB := t.TempDir()
	if _, err := Run(repoRoot, storeDirB, Options{}); err != nil {
		t.Fatalf("Run (from-scratch, final tree): %v", err)
	}
	reindexExport := exportNormalized(t, storeDirB, true)

	if !bytes.Equal(syncExport, reindexExport) {
		reportFirstDiff(t, syncExport, reindexExport)
	}
}
