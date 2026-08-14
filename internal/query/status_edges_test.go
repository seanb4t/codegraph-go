package query

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// seedGraphEngine opens a fresh Pebble store in t.TempDir(), writes files
// and edges through the real Writer/Commit path (graphstore/iter_test.go's
// TestIterateNodes precedent), takes a snapshot, and returns an Engine
// wrapping it via New(reader) directly — the same no-repoRoot
// construction files_status_test.go's TestDbSizeBytes subtest "Status
// degrades DbSizeBytes to 0 when repoRoot is unset" already exercises.
// t.Cleanup closes both the snapshot and the underlying store, in that
// order (the snapshot must release before the store it was taken from).
func seedGraphEngine(t *testing.T, files []*schema.File, edges []*schema.Edge) *Engine {
	t.Helper()

	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for _, f := range files {
		if err := w.PutFile(f); err != nil {
			t.Fatalf("PutFile(%+v): %v", f, err)
		}
	}
	for _, e := range edges {
		if err := w.PutEdge(e, ""); err != nil {
			t.Fatalf("PutEdge(%+v): %v", e, err)
		}
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = snap.Close() })

	return New(snap)
}

// asymmetricEdgeSet is the deliberately unbalanced multiset the tally
// tests below seed: three "calls" edges (three distinct (source, target)
// pairs, since PutEdge dedups on the (source, kind, target) triple per
// its own doc comment, internal/graphstore/batch.go), one "imports" edge,
// one "contains" edge (a real Kind this repo's own indexer emits,
// internal/indexer/resolve.go, but NOT a RankEdges member — RefKindCalls
// through RefKindImports below are, per rwr_test.go's TestRankEdges), and
// deliberately ZERO "overrides"/"type_of" edges. A tally bug that returns
// a uniform or transposed map fails against this shape rather than
// coincidentally passing a symmetric one.
func asymmetricEdgeSet() []*schema.Edge {
	return []*schema.Edge{
		{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "c", Kind: goextract.RefKindCalls},
		{Source: "b", Target: "c", Kind: goextract.RefKindCalls},
		{Source: "a", Target: "d", Kind: goextract.RefKindImports},
		{Source: "a", Target: "e", Kind: goextract.RefKindContains},
	}
}

// TestStatusEdgesByKindTally pins EdgesByKind's per-kind counts against a
// known, deliberately asymmetric seeded multiset (D-04's dense-mode
// closes the absent-vs-measured-zero ambiguity this sparse test half
// exercises; FIXT-01 criterion 3).
func TestStatusEdgesByKindTally(t *testing.T) {
	engine := seedGraphEngine(t, nil, asymmetricEdgeSet())

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}

	want := map[string]int64{
		goextract.RefKindCalls:    3,
		goextract.RefKindImports:  1,
		goextract.RefKindContains: 1,
	}
	if len(got.EdgesByKind) != len(want) {
		t.Fatalf("EdgesByKind has %d keys, want %d: %+v", len(got.EdgesByKind), len(want), got.EdgesByKind)
	}
	for kind, count := range want {
		if got.EdgesByKind[kind] != count {
			t.Errorf("EdgesByKind[%q] = %d, want %d (full map: %+v)", kind, got.EdgesByKind[kind], count, got.EdgesByKind)
		}
	}
}

// TestStatusEdgesByKindEmptyStore asserts a zero-edge store yields a
// non-nil, empty EdgesByKind map — never nil, so a JSON consumer sees {}
// rather than null.
func TestStatusEdgesByKindEmptyStore(t *testing.T) {
	engine := seedGraphEngine(t, nil, nil)

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if got.EdgesByKind == nil {
		t.Fatal("EdgesByKind is nil, want a non-nil empty map (so --json encodes {} rather than null)")
	}
	if len(got.EdgesByKind) != 0 {
		t.Fatalf("EdgesByKind has %d entries, want 0 for a zero-edge store: %+v", len(got.EdgesByKind), got.EdgesByKind)
	}
}

// TestStatusEdgesByKindSparseOmitsAbsentKind asserts a RankEdges kind the
// seeded store contains zero of (overrides, type_of) is ABSENT from
// EdgesByKind, not present with value 0 — the comma-ok form distinguishes
// a missing key from a key holding zero, which a plain value comparison
// would not (D-04's sparse contract).
func TestStatusEdgesByKindSparseOmitsAbsentKind(t *testing.T) {
	engine := seedGraphEngine(t, nil, asymmetricEdgeSet())

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}

	for _, absentKind := range []string{goextract.EdgeKindOverrides, goextract.RefKindTypeOf} {
		if _, ok := got.EdgesByKind[absentKind]; ok {
			t.Fatalf("EdgesByKind unexpectedly contains zero-count kind %q (want absent, not present-with-0): %+v", absentKind, got.EdgesByKind)
		}
	}
}

// TestStatusEdgesByKindKeepsUnrankedKind asserts an edge kind that is NOT
// a RankEdges member ("contains") is still tallied and present — the
// data layer never filters by RankEdges, so a future 10th edge kind
// cannot go silently unmeasured.
func TestStatusEdgesByKindKeepsUnrankedKind(t *testing.T) {
	if RankEdges[goextract.RefKindContains] {
		t.Fatal("test assumption violated: RefKindContains is now a RankEdges member; this test needs a different unranked kind")
	}

	engine := seedGraphEngine(t, nil, asymmetricEdgeSet())

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}

	count, ok := got.EdgesByKind[goextract.RefKindContains]
	if !ok {
		t.Fatalf(`EdgesByKind missing %q — an unranked kind must still be tallied, not silently discarded: %+v`, goextract.RefKindContains, got.EdgesByKind)
	}
	if count != 1 {
		t.Fatalf("EdgesByKind[%q] = %d, want 1", goextract.RefKindContains, count)
	}
}

// TestStatusFilesByLanguageSurvivesJSONRoundTrip asserts FilesByLanguage
// (un-suppressed as of v0.11.0 Phase 1, D-03) marshals through
// MarshalStatusJSON with its counts intact.
func TestStatusFilesByLanguageSurvivesJSONRoundTrip(t *testing.T) {
	files := []*schema.File{
		{Path: "a.go", Language: "go"},
		{Path: "b.go", Language: "go"},
		{Path: "c.py", Language: "python"},
	}
	engine := seedGraphEngine(t, files, nil)

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}

	raw, err := MarshalStatusJSON(got)
	if err != nil {
		t.Fatalf("MarshalStatusJSON: unexpected error: %v", err)
	}

	var decoded struct {
		FilesByLanguage map[string]int64 `json:"filesByLanguage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal status JSON: %v\n%s", err, raw)
	}

	want := map[string]int64{"go": 2, "python": 1}
	if len(decoded.FilesByLanguage) != len(want) {
		t.Fatalf("round-tripped filesByLanguage has %d keys, want %d: %+v", len(decoded.FilesByLanguage), len(want), decoded.FilesByLanguage)
	}
	for lang, count := range want {
		if decoded.FilesByLanguage[lang] != count {
			t.Errorf("round-tripped filesByLanguage[%q] = %d, want %d", lang, decoded.FilesByLanguage[lang], count)
		}
	}
}
