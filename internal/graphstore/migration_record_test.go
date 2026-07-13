package graphstore

import (
	"bytes"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestMigrationRecord_RoundTrip proves PutMigration/GetMigration carry an
// opaque []byte payload byte-identically through the GraphStore boundary
// (07-02 must_haves: "A migration progress blob can be written and read
// back byte-identically through the GraphStore boundary").
func TestMigrationRecord_RoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	want := []byte{0x00, 0x01, 0xFF, 'c', 'u', 'r', 's', 'o', 'r'}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMigration(want); err != nil {
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

	got, err := snap.GetMigration()
	if err != nil {
		t.Fatalf("GetMigration: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("GetMigration = %x, want %x", got, want)
	}
}

// TestMigrationRecord_AbsentReturnsErrNotFound proves a fresh store (no
// PutMigration ever committed) returns the graphstore.ErrNotFound sentinel,
// never a nil-success or panic (07-02 must_haves).
func TestMigrationRecord_AbsentReturnsErrNotFound(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	_, err = snap.GetMigration()
	if err != ErrNotFound {
		t.Fatalf("GetMigration on fresh store = %v, want ErrNotFound", err)
	}
}

// TestMigrationRecord_IsolatedFromMeta proves the m/migration record and
// the real m/schema Meta record coexist without clobbering each other
// (07-02 must_haves; T-07-02-01).
func TestMigrationRecord_IsolatedFromMeta(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	meta := schema.NewMeta()
	meta.NodeCount = 42

	cursor := []byte("in-progress:nodes:1000")

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.PutMigration(cursor); err != nil {
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

	gotMeta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if gotMeta.GetNodeCount() != 42 {
		t.Fatalf("GetMeta.NodeCount = %d, want 42 (Meta must be unaffected by PutMigration)", gotMeta.GetNodeCount())
	}
	if gotMeta.GetSchemaVersion() != schema.SchemaVersion {
		t.Fatalf("GetMeta.SchemaVersion = %d, want %d (SchemaVersion must NOT be bumped by the additive migration record)", gotMeta.GetSchemaVersion(), schema.SchemaVersion)
	}

	gotCursor, err := snap.GetMigration()
	if err != nil {
		t.Fatalf("GetMigration: %v", err)
	}
	if !bytes.Equal(gotCursor, cursor) {
		t.Fatalf("GetMigration = %q, want %q", gotCursor, cursor)
	}
}

// TestMigrationRecord_OverwriteIdempotent proves PutMigration twice with
// different bytes leaves the last-written bytes readable back — last-write-
// wins on the single m/migration key, mirroring PutMeta's own overwrite
// semantics.
func TestMigrationRecord_OverwriteIdempotent(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	first := []byte("cursor-v1")
	second := []byte("cursor-v2-longer-payload")

	w1, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter (1): %v", err)
	}
	if err := w1.PutMigration(first); err != nil {
		t.Fatalf("PutMigration (1): %v", err)
	}
	if err := w1.Commit(); err != nil {
		t.Fatalf("Commit (1): %v", err)
	}

	w2, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter (2): %v", err)
	}
	if err := w2.PutMigration(second); err != nil {
		t.Fatalf("PutMigration (2): %v", err)
	}
	if err := w2.Commit(); err != nil {
		t.Fatalf("Commit (2): %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer snap.Close()

	got, err := snap.GetMigration()
	if err != nil {
		t.Fatalf("GetMigration: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Fatalf("GetMigration = %q, want last-written %q", got, second)
	}
}
