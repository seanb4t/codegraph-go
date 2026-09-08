package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestIndexMetaCarriesTheStoredMeta proves (*Engine).IndexMeta returns the
// *schema.Meta the store actually holds, including the commit_sha field
// (ENG-04/D-05) — the data path 01-09's Status RPC needs, which did not
// exist before this plan (Engine.reader is unexported and
// graphstore.Reader.GetMeta was otherwise unreachable from a wire layer).
func TestIndexMetaCarriesTheStoredMeta(t *testing.T) {
	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	want := schema.NewMeta()
	want.NodeCount = 7
	want.EdgeCount = 3
	want.HasFileIndex = true
	want.CommitSha = "e08b1a3b3e9dfea6b06e17df8a2c8e37e3c9d0f1"

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(want); err != nil {
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

	e := New(snap)
	got, err := e.IndexMeta()
	if err != nil {
		t.Fatalf("IndexMeta: %v", err)
	}
	if got == nil {
		t.Fatal("IndexMeta returned (nil, nil) for a store with a committed Meta record")
	}
	if got.GetCommitSha() != want.GetCommitSha() {
		t.Fatalf("IndexMeta().CommitSha = %q, want %q", got.GetCommitSha(), want.GetCommitSha())
	}
	if got.GetNodeCount() != want.GetNodeCount() {
		t.Fatalf("IndexMeta().NodeCount = %d, want %d", got.GetNodeCount(), want.GetNodeCount())
	}
	if got.GetEdgeCount() != want.GetEdgeCount() {
		t.Fatalf("IndexMeta().EdgeCount = %d, want %d", got.GetEdgeCount(), want.GetEdgeCount())
	}
}

// TestIndexMetaOnAGraphWithNoMeta proves a store with no Meta record ever
// written yields (nil, nil) — absent, never an error — so a pre-upgrade
// graph (or a genuinely brand-new empty store) degrades the Status RPC
// rather than failing it.
func TestIndexMetaOnAGraphWithNoMeta(t *testing.T) {
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

	e := New(snap)
	got, err := e.IndexMeta()
	if err != nil {
		t.Fatalf("IndexMeta on a store with no Meta record: err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("IndexMeta on a store with no Meta record = %+v, want nil", got)
	}
}

// statusResultKnownFieldNames is a literal fixture of StatusResult's field
// names, transcribed from internal/query/status.go's struct definition as
// of 2026-08-23 (Phase 1, 01-06). D-06 requires codegraph status's output
// bytes stay byte-identical across this plan — this test is one of three
// independent controls proving it (the others: status.go asserted
// unmodified by diff, and a before/after `codegraph status --json`
// capture recorded in the SUMMARY). A field added to StatusResult by this
// plan (there must be none) fails this test's set-equality and length
// assertions both.
var statusResultKnownFieldNames = []string{
	"Initialized",
	"Version",
	"ProjectPath",
	"IndexPath",
	"FileCount",
	"NodeCount",
	"EdgeCount",
	"DbSizeBytes",
	"Backend",
	"NodesByKind",
	"EdgesByKind",
	"FilesByLanguage",
	"Languages",
	"PendingChanges",
	"WorktreeMismatch",
	"Stale",
	"Index",
}

// TestStatusResultFieldSetIsUnchanged asserts StatusResult's reflected
// field-name set equals statusResultKnownFieldNames exactly — both a set
// equality (no field added, none removed, none renamed) and a length
// assertion (so a fixture accidentally emptied by a bad edit cannot pass
// vacuously).
func TestStatusResultFieldSetIsUnchanged(t *testing.T) {
	typ := reflect.TypeOf(StatusResult{})

	got := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, typ.Field(i).Name)
	}

	want := append([]string(nil), statusResultKnownFieldNames...)

	sort.Strings(got)
	sort.Strings(want)

	if len(want) == 0 {
		t.Fatal("statusResultKnownFieldNames fixture is empty — this guard would be vacuous")
	}
	if len(got) != len(want) {
		t.Fatalf("StatusResult has %d fields %v, want %d fields %v", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("StatusResult field set differs from the recorded fixture (-want +got):\nwant: %v\ngot:  %v", want, got)
		}
	}
}
