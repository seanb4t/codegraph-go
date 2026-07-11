package indexer

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// exportNormalized opens the store at storeDir, Exports it, and returns the
// stream with Meta.last_sync_unix_ms zeroed in the leading Meta frame — the
// same normalization determinism_test.go's indexAndExport applies (INDX-02),
// factored out here since this test compares two ALREADY-BUILT stores
// (one via Sync, one via a fresh Run) rather than building both itself.
func exportNormalized(t *testing.T, storeDir string) []byte {
	t.Helper()

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	var buf bytes.Buffer
	if err := store.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	frames := decodeExportFrames(t, buf.Bytes())
	for i, f := range frames {
		if f.kind != frameKindMeta {
			continue
		}
		var m schema.Meta
		if err := proto.Unmarshal(f.data, &m); err != nil {
			t.Fatalf("unmarshal meta frame: %v", err)
		}
		m.LastSyncUnixMs = 0
		normalized, err := proto.Marshal(&m)
		if err != nil {
			t.Fatalf("marshal normalized meta frame: %v", err)
		}
		frames[i].data = normalized
	}
	return encodeExportFrames(frames)
}

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
	files, _, err := Discover(root)
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

	files, _, err := Discover(repoRoot)
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

	syncExport := exportNormalized(t, storeDirA)

	storeDirB := t.TempDir()
	if _, err := Run(repoRoot, storeDirB, Options{}); err != nil {
		t.Fatalf("Run (from-scratch, final tree): %v", err)
	}
	reindexExport := exportNormalized(t, storeDirB)

	if !bytes.Equal(syncExport, reindexExport) {
		reportFirstDiff(t, syncExport, reindexExport)
	}
}
