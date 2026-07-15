package query

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// scoringFakeReader is a minimal in-memory graphstore.Reader driving
// H14-H16 against a fully-controlled synthetic node/edge set (mirrors
// expand_test.go's/gather_test.go's own "each plan may need its own
// reader test double" convention). GetNode returns graphstore.ErrNotFound
// for any id not present in nodes, matching a real Reader's dangling-
// reference contract (WR-04). IterateEdges(prefix) filters to edges whose
// Source equals prefix, exactly like expandFakeReader — the same
// direct-outgoing-edge contract signatureTypeTargets (Task 2, H16) will
// rely on.
type scoringFakeReader struct {
	nodes map[string]*schema.Node
	edges []*schema.Edge
}

func (f *scoringFakeReader) GetNode(id string) (*schema.Node, error) {
	n, ok := f.nodes[id]
	if !ok {
		return nil, graphstore.ErrNotFound
	}
	return n, nil
}
func (f *scoringFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("scoringFakeReader: GetFile not implemented")
}
func (f *scoringFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("scoringFakeReader: GetMeta not implemented")
}
func (f *scoringFakeReader) GetMigration() ([]byte, error) {
	return nil, errors.New("scoringFakeReader: GetMigration not implemented")
}
func (f *scoringFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("scoringFakeReader: IterateFiles not implemented")
}
func (f *scoringFakeReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("scoringFakeReader: IterateFileIndex not implemented")
}
func (f *scoringFakeReader) Close() error { return nil }

func (f *scoringFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	nodes := make([]*schema.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		nodes = append(nodes, n)
	}
	return &scoringFakeNodeIterator{nodes: nodes}, nil
}

func (f *scoringFakeReader) IterateEdges(prefix string) (graphstore.EdgeIterator, error) {
	var filtered []*schema.Edge
	for _, e := range f.edges {
		if prefix == "" || e.Source == prefix {
			filtered = append(filtered, e)
		}
	}
	return &scoringFakeEdgeIterator{edges: filtered}, nil
}

type scoringFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *scoringFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}
func (it *scoringFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *scoringFakeNodeIterator) Err() error         { return nil }
func (it *scoringFakeNodeIterator) Close() error       { return nil }

type scoringFakeEdgeIterator struct {
	edges []*schema.Edge
	i     int
}

func (it *scoringFakeEdgeIterator) Next() bool {
	if it.i >= len(it.edges) {
		return false
	}
	it.i++
	return true
}
func (it *scoringFakeEdgeIterator) Edge() *schema.Edge { return it.edges[it.i-1] }
func (it *scoringFakeEdgeIterator) Err() error         { return nil }
func (it *scoringFakeEdgeIterator) Close() error       { return nil }

// --- H14: TestFileScoreTiers ---

// TestFileScoreTiers_NamedSeedFileScores50 pins H14's top tier: a file
// containing a named-seed node scores exactly 50, regardless of any other
// node it also holds.
func TestFileScoreTiers_NamedSeedFileScores50(t *testing.T) {
	nodes := map[string]*schema.Node{
		"seed":  {Id: "seed", Name: "Seed", FilePath: "a.go"},
		"other": {Id: "other", Name: "Other", FilePath: "a.go"},
	}
	r := &scoringFakeReader{nodes: nodes}
	namedSeedIDs := map[string]bool{"seed": true}

	got, err := computeFileScoreTiers(r, []string{"seed", "other"}, nil, namedSeedIDs, nil)
	if err != nil {
		t.Fatalf("computeFileScoreTiers: unexpected error: %v", err)
	}
	if got["a.go"] != fileScoreNamedSeed {
		t.Errorf("expected a.go score %v (named-seed), got %v", fileScoreNamedSeed, got["a.go"])
	}
}

// TestFileScoreTiers_EntryFileScores10 pins H14's entry tier.
func TestFileScoreTiers_EntryFileScores10(t *testing.T) {
	nodes := map[string]*schema.Node{
		"entry": {Id: "entry", Name: "Entry", FilePath: "b.go"},
	}
	r := &scoringFakeReader{nodes: nodes}
	entryIDs := map[string]bool{"entry": true}

	got, err := computeFileScoreTiers(r, []string{"entry"}, nil, nil, entryIDs)
	if err != nil {
		t.Fatalf("computeFileScoreTiers: unexpected error: %v", err)
	}
	if got["b.go"] != fileScoreEntry {
		t.Errorf("expected b.go score %v (entry), got %v", fileScoreEntry, got["b.go"])
	}
}

// TestFileScoreTiers_ConnectedToEntryScores3 pins H14's connected-to-entry
// tier: a node with a direct RankEdges edge to an entry node, but not
// itself named-seed/entry, scores its file at 3.
func TestFileScoreTiers_ConnectedToEntryScores3(t *testing.T) {
	nodes := map[string]*schema.Node{
		"entry":     {Id: "entry", Name: "Entry", FilePath: "b.go"},
		"connected": {Id: "connected", Name: "Connected", FilePath: "c.go"},
	}
	edges := []*schema.Edge{
		{Source: "entry", Target: "connected", Kind: goextract.RefKindCalls},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}
	entryIDs := map[string]bool{"entry": true}

	got, err := computeFileScoreTiers(r, []string{"entry", "connected"}, edges, nil, entryIDs)
	if err != nil {
		t.Fatalf("computeFileScoreTiers: unexpected error: %v", err)
	}
	if got["c.go"] != fileScoreConnected {
		t.Errorf("expected c.go score %v (connected-to-entry), got %v", fileScoreConnected, got["c.go"])
	}
}

// TestFileScoreTiers_OtherFileDroppedBelowThreshold pins H14's cutoff:
// a file whose every node is "other"-tier (score 1, below the 3 keep
// threshold) is dropped from the returned map entirely.
func TestFileScoreTiers_OtherFileDroppedBelowThreshold(t *testing.T) {
	nodes := map[string]*schema.Node{
		"lonely": {Id: "lonely", Name: "Lonely", FilePath: "z.go"},
	}
	r := &scoringFakeReader{nodes: nodes}

	got, err := computeFileScoreTiers(r, []string{"lonely"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("computeFileScoreTiers: unexpected error: %v", err)
	}
	if _, ok := got["z.go"]; ok {
		t.Errorf("expected z.go (other-tier, score 1 < 3) to be dropped, got present: %v", got)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

// TestFileScoreTiers_FileRollsUpToHighestNodeTier pins H14's per-file (not
// per-node) rollup: a file holding both a named-seed node and an
// "other"-tier node scores 50 (the max), not some blended/summed value.
func TestFileScoreTiers_FileRollsUpToHighestNodeTier(t *testing.T) {
	nodes := map[string]*schema.Node{
		"seed":  {Id: "seed", Name: "Seed", FilePath: "mixed.go"},
		"other": {Id: "other", Name: "Other", FilePath: "mixed.go"},
	}
	r := &scoringFakeReader{nodes: nodes}
	namedSeedIDs := map[string]bool{"seed": true}

	got, err := computeFileScoreTiers(r, []string{"seed", "other"}, nil, namedSeedIDs, nil)
	if err != nil {
		t.Fatalf("computeFileScoreTiers: unexpected error: %v", err)
	}
	if got["mixed.go"] != fileScoreNamedSeed {
		t.Errorf("expected mixed.go to roll up to named-seed tier %v, got %v", fileScoreNamedSeed, got["mixed.go"])
	}
}

// --- H14 fileGraphScore aggregation: TestFileGraphScoreAgg ---

// TestFileGraphScoreAgg_SumsMassPerFile pins RESEARCH §4: fileGraphScore
// is the SUM of node-level RWR mass over every node in that file.
func TestFileGraphScoreAgg_SumsMassPerFile(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a1": {Id: "a1", Name: "A1", FilePath: "a.go"},
		"a2": {Id: "a2", Name: "A2", FilePath: "a.go"},
		"b1": {Id: "b1", Name: "B1", FilePath: "b.go"},
	}
	r := &scoringFakeReader{nodes: nodes}
	rwrScores := map[string]float64{"a1": 0.3, "a2": 0.2, "b1": 0.1}

	got, err := aggregateFileGraphScore(r, []string{"a1", "a2", "b1"}, rwrScores)
	if err != nil {
		t.Fatalf("aggregateFileGraphScore: unexpected error: %v", err)
	}
	if diff := got["a.go"] - 0.5; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("expected a.go fileGraphScore ~0.5 (0.3+0.2), got %v", got["a.go"])
	}
	if diff := got["b.go"] - 0.1; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("expected b.go fileGraphScore ~0.1, got %v", got["b.go"])
	}
}

// TestFileGraphScoreAgg_SkipsDanglingNode pins WR-04: a nodeIDs entry
// that no longer resolves is skipped, not an error.
func TestFileGraphScoreAgg_SkipsDanglingNode(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a1": {Id: "a1", Name: "A1", FilePath: "a.go"},
	}
	r := &scoringFakeReader{nodes: nodes}
	rwrScores := map[string]float64{"a1": 0.4, "ghost": 0.9}

	got, err := aggregateFileGraphScore(r, []string{"a1", "ghost"}, rwrScores)
	if err != nil {
		t.Fatalf("aggregateFileGraphScore: unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected only a.go in result (ghost dangling), got %v", got)
	}
	if diff := got["a.go"] - 0.4; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("expected a.go fileGraphScore ~0.4, got %v", got["a.go"])
	}
}
