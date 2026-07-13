package migrate

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// TestSaveProgressLoadProgress_RoundTrip proves saveProgress/loadProgress
// carry a Progress cursor byte-faithfully through the real graphstore
// PutMigration/GetMigration boundary (D-06 resumability).
func TestSaveProgressLoadProgress_RoundTrip(t *testing.T) {
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	want := Progress{
		SourceSchemaVersion: 7,
		TargetSchemaVersion: 1,
		LastTable:           "edges",
		LastRowID:           5000,
		Status:              StatusInProgress,
	}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := saveProgress(w, want); err != nil {
		t.Fatalf("saveProgress: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	got, ok, err := loadProgress(snap)
	if err != nil {
		t.Fatalf("loadProgress: %v", err)
	}
	if !ok {
		t.Fatalf("loadProgress: got absent=false, want true")
	}
	if got != want {
		t.Fatalf("loadProgress = %+v, want %+v", got, want)
	}
}

// TestLoadProgress_AbsentReportsCleanly proves a fresh store with no cursor
// ever committed reports absent (false, nil error) — not an error, not a
// crash — so a first migration run starts from the top.
func TestLoadProgress_AbsentReportsCleanly(t *testing.T) {
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	got, ok, err := loadProgress(snap)
	if err != nil {
		t.Fatalf("loadProgress on fresh store: unexpected error %v", err)
	}
	if ok {
		t.Fatalf("loadProgress on fresh store: got absent=true (ok=%v), want ok=false", ok)
	}
	if got != (Progress{}) {
		t.Fatalf("loadProgress on fresh store: got %+v, want zero Progress", got)
	}
}

// TestSaveProgressLoadProgress_StatusComplete proves a saved
// Status=StatusComplete round-trips and is distinguishable from
// StatusInProgress.
func TestSaveProgressLoadProgress_StatusComplete(t *testing.T) {
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	want := Progress{
		SourceSchemaVersion: 7,
		TargetSchemaVersion: 1,
		LastTable:           "edges",
		LastRowID:           99999,
		Status:              StatusComplete,
	}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := saveProgress(w, want); err != nil {
		t.Fatalf("saveProgress: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	got, ok, err := loadProgress(snap)
	if err != nil {
		t.Fatalf("loadProgress: %v", err)
	}
	if !ok {
		t.Fatalf("loadProgress: got absent=false, want true")
	}
	if got.Status != StatusComplete {
		t.Fatalf("loadProgress.Status = %q, want %q (StatusComplete)", got.Status, StatusComplete)
	}
	if got.Status == StatusInProgress {
		t.Fatalf("loadProgress.Status must NOT equal StatusInProgress after saving StatusComplete")
	}
}

// TestLoadProgress_CorruptFailsLoud proves loadProgress on non-JSON cursor
// bytes returns a wrapped error rather than silently treating garbled bytes
// as "start clean" (which would risk a double-migration).
func TestLoadProgress_CorruptFailsLoud(t *testing.T) {
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMigration([]byte("not-json-at-all-{{{")); err != nil {
		t.Fatalf("PutMigration: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	_, ok, err := loadProgress(snap)
	if err == nil {
		t.Fatalf("loadProgress on corrupt bytes: want error, got nil (ok=%v)", ok)
	}
}
