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

// --- H15: TestHardTestExclusion ---

// TestHardTestExclusion_DropsTestFileByDefault pins H15's default
// behavior: a test-file candidate is dropped when the query does not
// mention "test".
func TestHardTestExclusion_DropsTestFileByDefault(t *testing.T) {
	fileScores := map[string]float64{
		"pkg/foo_test.go": 10,
		"pkg/foo.go":      10,
		"pkg/bar.go":      10,
	}
	applyHardTestExclusion(fileScores, "widget handler")

	if _, ok := fileScores["pkg/foo_test.go"]; ok {
		t.Errorf("expected pkg/foo_test.go to be dropped, got present: %v", fileScores)
	}
	if len(fileScores) != 2 {
		t.Errorf("expected 2 remaining non-test files, got %v", fileScores)
	}
}

// TestHardTestExclusion_DropsIconI18nFileByDefault pins H15's icon/i18n
// component of the low-value set.
func TestHardTestExclusion_DropsIconI18nFileByDefault(t *testing.T) {
	fileScores := map[string]float64{
		"assets/icons/plus.go": 10,
		"pkg/foo.go":           10,
		"pkg/bar.go":           10,
	}
	applyHardTestExclusion(fileScores, "widget handler")

	if _, ok := fileScores["assets/icons/plus.go"]; ok {
		t.Errorf("expected assets/icons/plus.go to be dropped, got present: %v", fileScores)
	}
}

// TestHardTestExclusion_ExemptWhenQueryMentionsTestAndTwoNonTestRemain
// pins H15's exemption: query mentions "test" AND >=2 non-test candidates
// remain -> NOTHING is dropped, including the low-value files.
func TestHardTestExclusion_ExemptWhenQueryMentionsTestAndTwoNonTestRemain(t *testing.T) {
	fileScores := map[string]float64{
		"pkg/foo_test.go": 10,
		"pkg/foo.go":      10,
		"pkg/bar.go":      10,
	}
	applyHardTestExclusion(fileScores, "test coverage for widget")

	if _, ok := fileScores["pkg/foo_test.go"]; !ok {
		t.Errorf("expected pkg/foo_test.go to be EXEMPT (query mentions test, >=2 non-test remain), got dropped: %v", fileScores)
	}
	if len(fileScores) != 3 {
		t.Errorf("expected all 3 files retained under exemption, got %v", fileScores)
	}
}

// TestHardTestExclusion_StillDropsWhenFewerThanTwoNonTestRemain pins
// H15's safety net: even when the query mentions "test", the exemption
// requires >=2 non-test candidates to remain — with only 1 non-test file,
// the low-value files are still dropped.
func TestHardTestExclusion_StillDropsWhenFewerThanTwoNonTestRemain(t *testing.T) {
	fileScores := map[string]float64{
		"pkg/foo_test.go": 10,
		"pkg/bar.go":      10,
	}
	applyHardTestExclusion(fileScores, "test coverage for widget")

	if _, ok := fileScores["pkg/foo_test.go"]; ok {
		t.Errorf("expected pkg/foo_test.go to be dropped (<2 non-test remain), got present: %v", fileScores)
	}
	if len(fileScores) != 1 {
		t.Errorf("expected 1 remaining file, got %v", fileScores)
	}
}

// --- H16: TestBuriedRescue ---

// TestBuriedRescue_RescuesGenuinelyBuriedSignatureTypeFile pins H16's
// core rule: a tier-seed callable's signature-type file with
// fileGraphScore < maxGraph*0.06 AND termHits < 2 is rescued, forced to
// score = max(existing, 45).
func TestBuriedRescue_RescuesGenuinelyBuriedSignatureTypeFile(t *testing.T) {
	nodes := map[string]*schema.Node{
		"callable": {Id: "callable", Name: "Callable", FilePath: "svc.go"},
		"sigType":  {Id: "sigType", Name: "SigType", FilePath: "buried.go"},
	}
	edges := []*schema.Edge{
		{Source: "callable", Target: "sigType", Kind: goextract.RefKindReturns},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}

	fileGraphScore := map[string]float64{"svc.go": 100.0, "buried.go": 0.5} // 0.5 < 100*0.06=6
	fileTermHits := map[string]int{"buried.go": 0}
	fileScores := map[string]float64{}

	rescued, err := applyBuriedRescue(r, []string{"callable"}, fileGraphScore, 100.0, fileTermHits, fileScores)
	if err != nil {
		t.Fatalf("applyBuriedRescue: unexpected error: %v", err)
	}
	if !rescued["buried.go"] {
		t.Errorf("expected buried.go to be rescued, got %v", rescued)
	}
	if fileScores["buried.go"] != buriedRescueScoreFloor {
		t.Errorf("expected buried.go score forced to %v, got %v", buriedRescueScoreFloor, fileScores["buried.go"])
	}
}

// TestBuriedRescue_ForcedScoreIsMaxNotOverwrite pins the "max(score,45)"
// wording literally: an existing HIGHER score is never lowered by rescue.
func TestBuriedRescue_ForcedScoreIsMaxNotOverwrite(t *testing.T) {
	nodes := map[string]*schema.Node{
		"callable": {Id: "callable", Name: "Callable", FilePath: "svc.go"},
		"sigType":  {Id: "sigType", Name: "SigType", FilePath: "buried.go"},
	}
	edges := []*schema.Edge{
		{Source: "callable", Target: "sigType", Kind: goextract.RefKindTypeOf},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}

	fileGraphScore := map[string]float64{"buried.go": 0.0}
	fileTermHits := map[string]int{}
	fileScores := map[string]float64{"buried.go": 60.0} // already above 45

	_, err := applyBuriedRescue(r, []string{"callable"}, fileGraphScore, 100.0, fileTermHits, fileScores)
	if err != nil {
		t.Fatalf("applyBuriedRescue: unexpected error: %v", err)
	}
	if fileScores["buried.go"] != 60.0 {
		t.Errorf("expected buried.go score to remain 60 (max(60,45)=60), got %v", fileScores["buried.go"])
	}
}

// TestBuriedRescue_NotBuriedNotRescued pins the negative case: a
// signature-type file with enough graph mass (>=6% of max) is NOT
// rescued, even though it is reachable via a tier-seed's signature-type
// edge.
func TestBuriedRescue_NotBuriedNotRescued(t *testing.T) {
	nodes := map[string]*schema.Node{
		"callable": {Id: "callable", Name: "Callable", FilePath: "svc.go"},
		"sigType":  {Id: "sigType", Name: "SigType", FilePath: "notburied.go"},
	}
	edges := []*schema.Edge{
		{Source: "callable", Target: "sigType", Kind: goextract.RefKindReferences},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}

	// 10.0 >= 100*0.06=6, so mass alone disqualifies "buried" regardless of termHits.
	fileGraphScore := map[string]float64{"notburied.go": 10.0}
	fileTermHits := map[string]int{"notburied.go": 0}
	fileScores := map[string]float64{}

	rescued, err := applyBuriedRescue(r, []string{"callable"}, fileGraphScore, 100.0, fileTermHits, fileScores)
	if err != nil {
		t.Fatalf("applyBuriedRescue: unexpected error: %v", err)
	}
	if rescued["notburied.go"] {
		t.Errorf("expected notburied.go NOT rescued (mass >= threshold), got rescued: %v", rescued)
	}
	if _, ok := fileScores["notburied.go"]; ok {
		t.Errorf("expected notburied.go absent from fileScores (never rescued), got %v", fileScores)
	}
}

// TestBuriedRescue_TermHitsAboveThresholdNotRescued pins the OTHER half
// of the buried AND condition: low mass but termHits>=2 is also NOT
// rescued.
func TestBuriedRescue_TermHitsAboveThresholdNotRescued(t *testing.T) {
	nodes := map[string]*schema.Node{
		"callable": {Id: "callable", Name: "Callable", FilePath: "svc.go"},
		"sigType":  {Id: "sigType", Name: "SigType", FilePath: "termy.go"},
	}
	edges := []*schema.Edge{
		{Source: "callable", Target: "sigType", Kind: goextract.RefKindReturns},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}

	fileGraphScore := map[string]float64{"termy.go": 0.1} // low mass
	fileTermHits := map[string]int{"termy.go": 2}         // but >=2 term hits
	fileScores := map[string]float64{}

	rescued, err := applyBuriedRescue(r, []string{"callable"}, fileGraphScore, 100.0, fileTermHits, fileScores)
	if err != nil {
		t.Fatalf("applyBuriedRescue: unexpected error: %v", err)
	}
	if rescued["termy.go"] {
		t.Errorf("expected termy.go NOT rescued (termHits >= 2), got rescued: %v", rescued)
	}
}

// TestBuriedRescue_NonSignatureEdgeKindIgnored pins H16's edge-kind
// filter: a "calls" edge from a tier-seed is not a signature-type edge
// and its target is never considered for rescue.
func TestBuriedRescue_NonSignatureEdgeKindIgnored(t *testing.T) {
	nodes := map[string]*schema.Node{
		"callable": {Id: "callable", Name: "Callable", FilePath: "svc.go"},
		"callee":   {Id: "callee", Name: "Callee", FilePath: "buried.go"},
	}
	edges := []*schema.Edge{
		{Source: "callable", Target: "callee", Kind: goextract.RefKindCalls},
	}
	r := &scoringFakeReader{nodes: nodes, edges: edges}

	fileGraphScore := map[string]float64{"buried.go": 0.0}
	fileTermHits := map[string]int{}
	fileScores := map[string]float64{}

	rescued, err := applyBuriedRescue(r, []string{"callable"}, fileGraphScore, 100.0, fileTermHits, fileScores)
	if err != nil {
		t.Fatalf("applyBuriedRescue: unexpected error: %v", err)
	}
	if rescued["buried.go"] {
		t.Errorf("expected buried.go NOT rescued (calls edge is not a signature-type edge), got rescued: %v", rescued)
	}
}
