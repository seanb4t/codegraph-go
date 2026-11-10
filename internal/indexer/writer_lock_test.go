package indexer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// TestSyncNeedsWriterLock checks that a sync is a writer of the store: it
// fails with ErrStoreLocked while another writer holds the writer lock,
// and it changes nothing.
func TestSyncNeedsWriterLock(t *testing.T) {
	root := copyFixture(t, prunefixtureRoot)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	before := exportNormalized(t, storeDir)

	lock, err := graphstore.LockWriter(storeDir)
	if err != nil {
		t.Fatalf("LockWriter: %v", err)
	}
	_, err = Sync(root, storeDir, Options{})
	if !errors.Is(err, graphstore.ErrStoreLocked) {
		lock.Close()
		t.Fatalf("Sync under a held writer lock: %v, want ErrStoreLocked", err)
	}
	if !bytes.Equal(exportNormalized(t, storeDir), before) {
		t.Errorf("the store changed in a Sync that could not take the writer lock")
	}

	// Positive assertion: the same sync works once the lock is free.
	if err := lock.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync after the writer lock is free: %v", err)
	}
}

// TestBackfillCheckCreatesNoStore checks that the backfill check of a sync
// reads the store without creating one: a missing store needs a backfill,
// and the check leaves its path missing.
func TestBackfillCheckCreatesNoStore(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "codegraph", "store")
	need, err := needsFileIndexBackfill(storeDir)
	if err != nil || !need {
		t.Fatalf("needsFileIndexBackfill of a missing store = %v, %v; want true", need, err)
	}
	if _, err := os.Lstat(filepath.Dir(storeDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Lstat(%s) = %v; want the check to create nothing", filepath.Dir(storeDir), err)
	}

	// Positive assertion: an indexed store needs no backfill.
	if _, err := Run(fixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	need, err = needsFileIndexBackfill(storeDir)
	if err != nil || need {
		t.Fatalf("needsFileIndexBackfill of an indexed store = %v, %v; want false", need, err)
	}
}
