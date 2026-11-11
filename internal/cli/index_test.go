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

// TestIndexForceRebuildBumpsCoverageGeneration checks that a full index
// over an indexed store raises Meta.CoverageGeneration (CR-01 of
// 10-REVIEW.md, iteration 3). A full index builds a new store and puts it
// in the place of the old one, so the new store does not hold the old
// generation by itself: the run reads it from the old store first. If it
// started again at 1, the generation would match the one in a page token
// that a client still holds.
//
// The test drives the real CLI command: `init`, a page token against the
// resulting store, then `index --force`. It asserts that (a) the
// generation rose and (b) the store refuses the old token with
// query.ErrAborted.
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

	// A full index through the actual CLI command.
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
