package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// readCoverageGeneration opens the store under codegraphDir and reads
// back the committed Meta record's CoverageGeneration — mirroring
// cli_test.go's readGraphCounts helper.
func readCoverageGeneration(t *testing.T, codegraphDir string) int64 {
	t.Helper()

	store, err := graphstore.Open(filepath.Join(codegraphDir, storeDirName))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	return meta.GetCoverageGeneration()
}

// TestIndexForceRebuildBumpsCoverageGeneration pins CR-01's fix
// (10-REVIEW.md iteration 3): `codegraph index`'s from-scratch rebuild
// wipes storeDir (RemoveAll + MkdirAll) before calling indexer.Run, which
// previously made writeGraph's own prior-generation read see a genuinely
// empty store and reset Meta.CoverageGeneration to 1 on EVERY rebuild —
// aliasing onto whatever generation a still-live client's page token
// already carried.
//
// This drives that exact sequence through the real CLI command (not two
// writeGraph calls against a store that was never wiped): `init`, mint a
// real page token against the resulting store, `index --force` (the
// actual RemoveAll+MkdirAll+indexer.Run path), then assert (a) the
// generation strictly increased across the wipe and (b) the pre-wipe
// token is rejected against the rebuilt store with query.ErrAborted
// rather than silently accepted.
func TestIndexForceRebuildBumpsCoverageGeneration(t *testing.T) {
	dir := copyFixture(t)

	// gofixture alone may not carry two coverage-gap rows on every
	// platform (skip_linux.go is only excluded on non-linux runners), so
	// add an OS-independent second one: a vendor/ directory is always
	// recorded as one EXCLUSION_REASON_DIR_VENDOR row regardless of
	// platform, and .codegraph/ itself (created by init below) is always
	// recorded as one EXCLUSION_REASON_DIR_DOTPREFIX row — together that
	// guarantees >= 2 coverage rows, so PageSize:1 always yields a real
	// NextPageToken.
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatalf("mkdir vendor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "dep.go"), []byte("package dep\n"), 0o644); err != nil {
		t.Fatalf("write vendor fixture file: %v", err)
	}

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	codegraphDir := filepath.Join(dir, codegraphDirName)

	gen1 := readCoverageGeneration(t, codegraphDir)

	eng1, closer1, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt (1st): %v", err)
	}
	page1, err := eng1.CoverageRows(query.CoverageRowsOptions{PageSize: 1})
	closer1.Close()
	if err != nil {
		t.Fatalf("CoverageRows (1st): %v", err)
	}
	if !page1.Known {
		t.Fatalf("expected Known coverage after init")
	}
	if page1.NextPageToken == "" {
		t.Fatalf("expected a NextPageToken with PageSize=1 against >= 2 coverage rows (got %d rows on this page)", len(page1.Rows))
	}
	staleToken := page1.NextPageToken

	// Mirror `codegraph index`'s real RemoveAll + MkdirAll + indexer.Run
	// sequence via the actual CLI command.
	if _, _, err := execCmd("index", "--force", dir); err != nil {
		t.Fatalf("index --force: %v", err)
	}

	gen2 := readCoverageGeneration(t, codegraphDir)
	if gen2 <= gen1 {
		t.Fatalf("CoverageGeneration did not strictly increase across a full re-index: gen1=%d gen2=%d", gen1, gen2)
	}

	eng2, closer2, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt (2nd): %v", err)
	}
	defer closer2.Close()

	_, err = eng2.CoverageRows(query.CoverageRowsOptions{PageSize: 1, PageToken: staleToken})
	if !errors.Is(err, query.ErrAborted) {
		t.Fatalf("CoverageRows with a pre-wipe page token = %v, want errors.Is(err, query.ErrAborted)", err)
	}
}
