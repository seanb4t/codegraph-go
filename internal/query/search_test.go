package query

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// searchFakeReader is a minimal in-memory graphstore.Reader used only to
// prove Query/Search's ranking tie-break (D-06) and the "ValidateKind
// before any scan" ordering (T-03-03-Kind) — independent of a real
// Pebble-backed store so the exact node set under test is fully
// controlled. Named with a search-specific prefix (rather than a generic
// "fakeReader") because 03-04/03-05 execute in this same package in
// parallel (Wave 3) and may need their own reader test double; only
// IterateNodes is meaningful here, every other Reader method is unused by
// Query/Search.
type searchFakeReader struct {
	nodes       []*schema.Node
	scanCounter *int // incremented each time IterateNodes is invoked
}

func (f *searchFakeReader) GetNode(string) (*schema.Node, error) {
	return nil, errors.New("searchFakeReader: GetNode not implemented")
}

func (f *searchFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("searchFakeReader: GetFile not implemented")
}

func (f *searchFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("searchFakeReader: GetMeta not implemented")
}

func (f *searchFakeReader) IterateEdges(string) (graphstore.EdgeIterator, error) {
	return nil, errors.New("searchFakeReader: IterateEdges not implemented")
}

func (f *searchFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("searchFakeReader: IterateFiles not implemented")
}

func (f *searchFakeReader) Close() error { return nil }

func (f *searchFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	if f.scanCounter != nil {
		*f.scanCounter++
	}
	return &searchFakeNodeIterator{nodes: f.nodes}, nil
}

type searchFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *searchFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}

func (it *searchFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *searchFakeNodeIterator) Err() error         { return nil }
func (it *searchFakeNodeIterator) Close() error       { return nil }

// TestQueryRankingTieBreak pins the D-06 tie-break: exact-name beats
// prefix beats substring/token, with a stable secondary sort by
// qualifiedName. The fixture is hand-built (not the indexed gofixture)
// so all three tiers are exercised deterministically:
//   - "Widget" (struct)          -> exact  (name == term)
//   - "Widget.Describe" (method) -> prefix (qualifiedName has prefix term)
//   - "Widget.Rename" (method)   -> prefix (qualifiedName has prefix term)
//   - "helperWidget" (function)  -> substring (neither field starts with term)
func TestQueryRankingTieBreak(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "function:zzz", Kind: "function", Name: "helperWidget", QualifiedName: "helperWidget"},
		{Id: "method:bbb", Kind: "method", Name: "Rename", QualifiedName: "Widget.Rename"},
		{Id: "method:aaa", Kind: "method", Name: "Describe", QualifiedName: "Widget.Describe"},
		{Id: "struct:ccc", Kind: "struct", Name: "Widget", QualifiedName: "Widget"},
	}
	e := New(&searchFakeReader{nodes: nodes})

	got, err := e.Query("Widget", "", 0)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}

	wantOrder := []string{"struct:ccc", "method:aaa", "method:bbb", "function:zzz"}
	if len(got) != len(wantOrder) {
		t.Fatalf("Query: got %d results, want %d: %+v", len(got), len(wantOrder), got)
	}
	for i, id := range wantOrder {
		if got[i].Id != id {
			t.Fatalf("Query result[%d] = %q, want %q (order: exact > prefix > substring, tie-break by qualifiedName)", i, got[i].Id, id)
		}
	}
}

func TestQueryKindFilterOnFakeReader(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:ccc", Kind: "struct", Name: "Widget", QualifiedName: "Widget"},
		{Id: "method:aaa", Kind: "method", Name: "Describe", QualifiedName: "Widget.Describe"},
	}
	e := New(&searchFakeReader{nodes: nodes})

	got, err := e.Query("Widget", "struct", 0)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Id != "struct:ccc" {
		t.Fatalf("Query with --kind struct = %+v, want exactly [struct:ccc]", got)
	}
}

func TestQueryLimitCapsAfterRanking(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "function:zzz", Kind: "function", Name: "helperWidget", QualifiedName: "helperWidget"},
		{Id: "method:bbb", Kind: "method", Name: "Rename", QualifiedName: "Widget.Rename"},
		{Id: "method:aaa", Kind: "method", Name: "Describe", QualifiedName: "Widget.Describe"},
		{Id: "struct:ccc", Kind: "struct", Name: "Widget", QualifiedName: "Widget"},
	}
	e := New(&searchFakeReader{nodes: nodes})

	got, err := e.Query("Widget", "", 2)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}
	want := []string{"struct:ccc", "method:aaa"}
	if len(got) != len(want) {
		t.Fatalf("Query with --limit 2: got %d results, want %d: %+v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].Id != id {
			t.Fatalf("Query with --limit 2, result[%d] = %q, want %q", i, got[i].Id, id)
		}
	}
}

// manyMatchingNodes builds n synthetic nodes whose name/qualifiedName all
// share the "widget" prefix, so every one of them matches a "widget" term
// scan — used by TestQueryDefaultCapAtMaxLimit/TestSearchDefaultCapAtMaxLimit
// (CR-01) to prove the MaxLimit ceiling applies even when the caller never
// supplies an explicit --limit (limit==0, the default/most common case).
func manyMatchingNodes(n int) []*schema.Node {
	nodes := make([]*schema.Node, n)
	for i := range nodes {
		name := fmt.Sprintf("widget%04d", i)
		nodes[i] = &schema.Node{
			Id:            fmt.Sprintf("function:%04d", i),
			Kind:          "function",
			Name:          name,
			QualifiedName: name,
		}
	}
	return nodes
}

// TestQueryDefaultCapAtMaxLimit pins CR-01: Query must cap its result set
// at MaxLimit even when limit==0 (the caller did not supply an explicit
// --limit) — previously only an explicit over-large --limit was rejected,
// and the common no-flag invocation returned every match unbounded.
func TestQueryDefaultCapAtMaxLimit(t *testing.T) {
	e := New(&searchFakeReader{nodes: manyMatchingNodes(MaxLimit + 50)})

	got, err := e.Query("widget", "", 0)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}
	if len(got) != MaxLimit {
		t.Fatalf("Query with limit=0 (default): got %d results, want the MaxLimit=%d ceiling to apply", len(got), MaxLimit)
	}
}

// TestSearchDefaultCapAtMaxLimit mirrors TestQueryDefaultCapAtMaxLimit for
// Search (CR-01) — the same unbounded-default gap existed independently in
// Search's own limit-application code.
func TestSearchDefaultCapAtMaxLimit(t *testing.T) {
	e := New(&searchFakeReader{nodes: manyMatchingNodes(MaxLimit + 50)})

	got, err := e.Search("widget", "", 0)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(got) != MaxLimit {
		t.Fatalf("Search with limit=0 (default): got %d results, want the MaxLimit=%d ceiling to apply", len(got), MaxLimit)
	}
}

// TestQueryUnknownKindRejectedBeforeScan proves --kind is validated via
// ValidateKind before any scan runs (V5, T-03-03-Kind, acceptance
// criteria): a fake reader that increments scanCounter on every
// IterateNodes call must see zero calls when --kind is unknown.
func TestQueryUnknownKindRejectedBeforeScan(t *testing.T) {
	var scans int
	e := New(&searchFakeReader{scanCounter: &scans})

	_, err := e.Query("term", "banana", 0)
	if err == nil {
		t.Fatal("Query: expected error for unknown --kind, got nil")
	}
	if scans != 0 {
		t.Fatalf("Query: IterateNodes was called %d time(s) despite an invalid --kind — ValidateKind must run before any scan", scans)
	}
}

func TestSearchUnknownKindRejectedBeforeScan(t *testing.T) {
	var scans int
	e := New(&searchFakeReader{scanCounter: &scans})

	_, err := e.Search("term", "banana", 0)
	if err == nil {
		t.Fatal("Search: expected error for unknown --kind, got nil")
	}
	if scans != 0 {
		t.Fatalf("Search: IterateNodes was called %d time(s) despite an invalid --kind — ValidateKind must run before any scan", scans)
	}
}

// TestQuery exercises Engine.Query against a real Pebble-backed store
// built from the shared gofixture (copyFixture/indexFixture, reused
// at runtime from engine_test.go per this plan's Wave-3 isolation rule).
func TestQuery(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	t.Run("matches a known fixture symbol by name", func(t *testing.T) {
		got, err := engine.Query("Alpha", "", 0)
		if err != nil {
			t.Fatalf("Query: unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Fatal(`Query("Alpha"): expected at least one match, got none`)
		}
		found := false
		for _, n := range got {
			if n.Name == "Alpha" {
				found = true
			}
		}
		if !found {
			t.Fatalf(`Query("Alpha"): no result named "Alpha" in %+v`, got)
		}
	})

	t.Run("kind filter drops non-matching kinds", func(t *testing.T) {
		got, err := engine.Query("Widget", "struct", 0)
		if err != nil {
			t.Fatalf("Query: unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Fatal(`Query("Widget", kind="struct"): expected at least one match, got none`)
		}
		for _, n := range got {
			if n.Kind != "struct" {
				t.Fatalf("Query with --kind struct returned a %q node: %+v", n.Kind, n)
			}
		}
	})

	t.Run("unknown kind is rejected", func(t *testing.T) {
		_, err := engine.Query("Widget", "banana", 0)
		if err == nil {
			t.Fatal("Query: expected error for unknown --kind, got nil")
		}
	})

	t.Run("limit caps the returned count", func(t *testing.T) {
		all, err := engine.Query("e", "", 0)
		if err != nil {
			t.Fatalf("Query: unexpected error: %v", err)
		}
		if len(all) < 2 {
			t.Skip(`fixture has too few matches for "e" to exercise --limit`)
		}
		limited, err := engine.Query("e", "", 1)
		if err != nil {
			t.Fatalf("Query with --limit 1: unexpected error: %v", err)
		}
		if len(limited) != 1 {
			t.Fatalf("Query with --limit 1: got %d results, want 1", len(limited))
		}
	})
}

// TestSearch exercises Engine.Search: the same matches as Query, but
// projected to the locations-only shape (name/kind/filePath/startLine,
// no source body/signature).
func TestSearch(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	t.Run("returns locations for known matches", func(t *testing.T) {
		got, err := engine.Search("Alpha", "", 0)
		if err != nil {
			t.Fatalf("Search: unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Fatal(`Search("Alpha"): expected at least one match, got none`)
		}
		for _, loc := range got {
			if loc.Name == "" || loc.Kind == "" || loc.FilePath == "" {
				t.Fatalf("Search: location missing a required field: %+v", loc)
			}
		}
	})

	t.Run("same match set as Query, locations-only projection", func(t *testing.T) {
		nodes, err := engine.Query("Alpha", "", 0)
		if err != nil {
			t.Fatalf("Query: unexpected error: %v", err)
		}
		locs, err := engine.Search("Alpha", "", 0)
		if err != nil {
			t.Fatalf("Search: unexpected error: %v", err)
		}
		if len(nodes) != len(locs) {
			t.Fatalf("Query returned %d results, Search returned %d — expected the same match set", len(nodes), len(locs))
		}
		for i := range nodes {
			if nodes[i].Name != locs[i].Name || nodes[i].Kind != locs[i].Kind ||
				nodes[i].FilePath != locs[i].FilePath || nodes[i].StartLine != locs[i].StartLine {
				t.Fatalf("Search[%d] = %+v does not match Query[%d] = %+v", i, locs[i], i, nodes[i])
			}
		}
	})
}

// TestQueryJSONShape pins query --json's element shape to the golden
// query.json contract (D-05): a top-level array of {"node": {...}}
// envelopes, no "score" key, and the three TS-only bool keys
// (isAsync/isStatic/isAbstract) present-and-false. Also proves
// byte-reproducibility across two independent marshals (D-06).
func TestQueryJSONShape(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	nodes, err := engine.Query("Alpha", "", 0)
	if err != nil {
		t.Fatalf("Query: unexpected error: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal(`Query("Alpha"): expected at least one match`)
	}

	b1, err := MarshalQueryJSON(nodes)
	if err != nil {
		t.Fatalf("MarshalQueryJSON: unexpected error: %v", err)
	}

	var envelopes []map[string]any
	if err := json.Unmarshal(b1, &envelopes); err != nil {
		t.Fatalf("MarshalQueryJSON output is not a top-level JSON array: %v\n%s", err, b1)
	}

	for i, env := range envelopes {
		nodeVal, ok := env["node"]
		if !ok {
			t.Fatalf(`envelope[%d] missing top-level "node" key: %v`, i, env)
		}
		nodeObj, ok := nodeVal.(map[string]any)
		if !ok {
			t.Fatalf(`envelope[%d]["node"] is not an object: %v`, i, nodeVal)
		}
		if _, present := nodeObj["score"]; present {
			t.Fatalf(`envelope[%d]["node"] must not contain a "score" key (D-06 drops TS's volatile FTS5/BM25 score): %v`, i, nodeObj)
		}
		for _, key := range []string{"isAsync", "isStatic", "isAbstract"} {
			v, present := nodeObj[key]
			if !present {
				t.Fatalf(`envelope[%d]["node"] missing TS-only bool key %q (D-05 shape parity)`, i, key)
			}
			if v != false {
				t.Fatalf(`envelope[%d]["node"][%q] = %v, want false (Go has no async/static/abstract modifiers)`, i, key, v)
			}
		}
	}

	t.Run("byte-reproducible across two marshals", func(t *testing.T) {
		nodes2, err := engine.Query("Alpha", "", 0)
		if err != nil {
			t.Fatalf("Query (2nd run): unexpected error: %v", err)
		}
		b2, err := MarshalQueryJSON(nodes2)
		if err != nil {
			t.Fatalf("MarshalQueryJSON (2nd run): unexpected error: %v", err)
		}
		if !bytes.Equal(b1, b2) {
			t.Fatalf("MarshalQueryJSON is not byte-reproducible:\nrun1: %s\nrun2: %s", b1, b2)
		}
	})
}
