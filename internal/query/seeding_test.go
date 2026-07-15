package query

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// seedingFakeReader is a minimal in-memory graphstore.Reader driving H13
// against a fully-controlled synthetic node/edge set (mirrors
// expand_test.go's/gather_test.go's own "each plan may need its own
// reader test double" convention). GetNode returns graphstore.ErrNotFound
// for any id not present in nodes, matching a real Reader's dangling-
// reference contract (WR-04).
type seedingFakeReader struct {
	nodes map[string]*schema.Node
	edges []*schema.Edge
}

func (f *seedingFakeReader) GetNode(id string) (*schema.Node, error) {
	n, ok := f.nodes[id]
	if !ok {
		return nil, graphstore.ErrNotFound
	}
	return n, nil
}
func (f *seedingFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("seedingFakeReader: GetFile not implemented")
}
func (f *seedingFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("seedingFakeReader: GetMeta not implemented")
}
func (f *seedingFakeReader) GetMigration() ([]byte, error) {
	return nil, errors.New("seedingFakeReader: GetMigration not implemented")
}
func (f *seedingFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("seedingFakeReader: IterateFiles not implemented")
}
func (f *seedingFakeReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("seedingFakeReader: IterateFileIndex not implemented")
}
func (f *seedingFakeReader) Close() error { return nil }

func (f *seedingFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	nodes := make([]*schema.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		nodes = append(nodes, n)
	}
	return &seedingFakeNodeIterator{nodes: nodes}, nil
}

func (f *seedingFakeReader) IterateEdges(prefix string) (graphstore.EdgeIterator, error) {
	var filtered []*schema.Edge
	for _, e := range f.edges {
		if prefix == "" || e.Source == prefix {
			filtered = append(filtered, e)
		}
	}
	return &seedingFakeEdgeIterator{edges: filtered}, nil
}

type seedingFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *seedingFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}
func (it *seedingFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *seedingFakeNodeIterator) Err() error         { return nil }
func (it *seedingFakeNodeIterator) Close() error       { return nil }

type seedingFakeEdgeIterator struct {
	edges []*schema.Edge
	i     int
}

func (it *seedingFakeEdgeIterator) Next() bool {
	if it.i >= len(it.edges) {
		return false
	}
	it.i++
	return true
}
func (it *seedingFakeEdgeIterator) Edge() *schema.Edge { return it.edges[it.i-1] }
func (it *seedingFakeEdgeIterator) Err() error         { return nil }
func (it *seedingFakeEdgeIterator) Close() error       { return nil }

func seedingContainsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// --- TestSeedingResolve: full-scan exact-name resolution ---

// TestSeedingResolve_ExactNameOnlyNotFTS pins H13's "getNodesByName, not
// FTS" rule: a query token "Process" must resolve ONLY nodes whose Name
// is EXACTLY "Process" — a node named "ProcessOrder" (which an FTS/
// substring channel would match) must never be seeded.
func TestSeedingResolve_ExactNameOnlyNotFTS(t *testing.T) {
	r := &seedingFakeReader{nodes: map[string]*schema.Node{
		"exact":    {Id: "exact", Kind: goextract.KindFunction, Name: "Process"},
		"substr":   {Id: "substr", Kind: goextract.KindFunction, Name: "ProcessOrder"},
		"unrelate": {Id: "unrelate", Kind: goextract.KindFunction, Name: "Widget"},
	}}

	res, err := seedNamedSymbols(r, "Process", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	if !seedingContainsID(res.SeedIDs, "exact") {
		t.Fatalf("expected exact-name match %q seeded, got SeedIDs=%v", "exact", res.SeedIDs)
	}
	if seedingContainsID(res.SeedIDs, "substr") {
		t.Fatalf("a substring/FTS-style match %q must NOT be seeded by getNodesByName-equivalent resolution, got SeedIDs=%v", "substr", res.SeedIDs)
	}
	if seedingContainsID(res.SeedIDs, "unrelate") {
		t.Fatalf("an unrelated name must never be seeded, got SeedIDs=%v", res.SeedIDs)
	}
}

// TestSeedingResolve_TokenMinLength pins H13's own extra token filter
// (on top of H1's extractSymbolsFromQuery): a token shorter than 3 chars
// must be ignored entirely — never resolved, never seeded — even if a
// node with that exact (short) name exists.
func TestSeedingResolve_TokenMinLength(t *testing.T) {
	r := &seedingFakeReader{nodes: map[string]*schema.Node{
		"short": {Id: "short", Kind: goextract.KindFunction, Name: "AB"},
		"long":  {Id: "long", Kind: goextract.KindFunction, Name: "Widget"},
	}}

	res, err := seedNamedSymbols(r, "AB Widget", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	if seedingContainsID(res.SeedIDs, "short") {
		t.Fatalf("a <3-char token must be ignored (never resolved/seeded), got SeedIDs=%v", res.SeedIDs)
	}
	if !seedingContainsID(res.SeedIDs, "long") {
		t.Fatalf("a >=3-char token naming an existing symbol must be seeded, got SeedIDs=%v", res.SeedIDs)
	}
}

// TestSeedingResolve_TokenMaxCountSixteen pins H13's 16-token cap: given
// seedQueryTokens directly (the token-filter unit, independent of any
// Reader), more than 16 qualifying tokens must be capped to the first 16
// in scan order.
func TestSeedingResolve_TokenMaxCountSixteen(t *testing.T) {
	// 20 distinct, qualifying (>=3 char, PascalCase) tokens in one query.
	query := "AlphaOne AlphaTwo AlphaThree AlphaFour AlphaFive AlphaSix AlphaSeven AlphaEight AlphaNine AlphaTen " +
		"AlphaEleven AlphaTwelve AlphaThirteen AlphaFourteen AlphaFifteen AlphaSixteen AlphaSeventeen AlphaEighteen AlphaNineteen AlphaTwenty"

	tokens := seedQueryTokens(query)
	if len(tokens) != seedTokenMaxCount {
		t.Fatalf("expected exactly %d tokens (16-token cap), got %d: %v", seedTokenMaxCount, len(tokens), tokens)
	}
	if tokens[0] != "AlphaOne" {
		t.Fatalf("expected first-seen scan order preserved, got first token %q", tokens[0])
	}
	for _, tok := range tokens {
		if tok == "AlphaSeventeen" || tok == "AlphaEighteen" || tok == "AlphaNineteen" || tok == "AlphaTwenty" {
			t.Fatalf("token %q beyond the 16-token cap must not appear, got tokens=%v", tok, tokens)
		}
	}
}

// --- TestSeedingSmallOverload: <=3-defs inject-all + caller-threshold tier ---

// TestSeedingSmallOverload_BothInjectedTierByCallerRatio pins H13's
// small-overload branch end to end: a name with 2 defs -> BOTH are
// injected into the seed set (inject-all), and the seed tier is def0
// (lowest-Id) plus any OTHER def whose caller count is
// >=0.25*maxCallers. Here def "hot" (4 callers) is def0 (lower Id than
// "cold"), and "cold" has 1 caller: 1 >= 0.25*4 (=1) so BOTH land in the
// tier too.
func TestSeedingSmallOverload_BothInjectedTierByCallerRatio(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a-hot":  {Id: "a-hot", Kind: goextract.KindFunction, Name: "Process"},
		"b-cold": {Id: "b-cold", Kind: goextract.KindFunction, Name: "Process"},
		"caller1": {Id: "caller1", Kind: goextract.KindFunction, Name: "Caller1"},
		"caller2": {Id: "caller2", Kind: goextract.KindFunction, Name: "Caller2"},
		"caller3": {Id: "caller3", Kind: goextract.KindFunction, Name: "Caller3"},
		"caller4": {Id: "caller4", Kind: goextract.KindFunction, Name: "Caller4"},
		"caller5": {Id: "caller5", Kind: goextract.KindFunction, Name: "Caller5"},
	}
	edges := []*schema.Edge{
		{Source: "caller1", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller2", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller3", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller4", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller5", Target: "b-cold", Kind: goextract.RefKindCalls},
	}
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	res, err := seedNamedSymbols(r, "Process", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	if len(res.Names) != 1 {
		t.Fatalf("expected exactly 1 resolved name, got %d: %+v", len(res.Names), res.Names)
	}
	got := res.Names[0]
	if got.Name != "Process" {
		t.Fatalf("expected resolved name %q, got %q", "Process", got.Name)
	}
	if len(got.Injected) != 2 || !seedingContainsID(got.Injected, "a-hot") || !seedingContainsID(got.Injected, "b-cold") {
		t.Fatalf("expected BOTH defs injected (inject-all, <=3 defs), got Injected=%v", got.Injected)
	}
	if len(got.Primary) != 2 || !seedingContainsID(got.Primary, "a-hot") || !seedingContainsID(got.Primary, "b-cold") {
		t.Fatalf("expected def0 (a-hot) + threshold-passing co-named def (b-cold, 1>=0.25*4) in the seed tier, got Primary=%v", got.Primary)
	}
	if !seedingContainsID(res.SeedIDs, "a-hot") || !seedingContainsID(res.SeedIDs, "b-cold") {
		t.Fatalf("expected both defs in the top-level SeedIDs union, got %v", res.SeedIDs)
	}
}

// TestSeedingSmallOverload_BelowThresholdExcludedFromTier pins the
// caller-ratio boundary: a co-named def whose caller count is BELOW
// 0.25*maxCallers is injected (inject-all still applies) but excluded
// from the seed tier.
func TestSeedingSmallOverload_BelowThresholdExcludedFromTier(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a-hot":  {Id: "a-hot", Kind: goextract.KindFunction, Name: "Process"},
		"b-cold": {Id: "b-cold", Kind: goextract.KindFunction, Name: "Process"},
		"caller1": {Id: "caller1", Kind: goextract.KindFunction, Name: "Caller1"},
		"caller2": {Id: "caller2", Kind: goextract.KindFunction, Name: "Caller2"},
		"caller3": {Id: "caller3", Kind: goextract.KindFunction, Name: "Caller3"},
		"caller4": {Id: "caller4", Kind: goextract.KindFunction, Name: "Caller4"},
		"caller5": {Id: "caller5", Kind: goextract.KindFunction, Name: "Caller5"},
		"caller6": {Id: "caller6", Kind: goextract.KindFunction, Name: "Caller6"},
		"caller7": {Id: "caller7", Kind: goextract.KindFunction, Name: "Caller7"},
		"caller8": {Id: "caller8", Kind: goextract.KindFunction, Name: "Caller8"},
	}
	edges := []*schema.Edge{
		{Source: "caller1", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller2", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller3", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller4", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller5", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller6", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller7", Target: "a-hot", Kind: goextract.RefKindCalls},
		{Source: "caller8", Target: "a-hot", Kind: goextract.RefKindCalls},
		// maxCallers = 8, threshold = 0.25*8 = 2; b-cold has 0 callers < 2.
	}
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	res, err := seedNamedSymbols(r, "Process", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	got := res.Names[0]
	if len(got.Injected) != 2 {
		t.Fatalf("expected BOTH defs still injected (inject-all applies regardless of tier), got Injected=%v", got.Injected)
	}
	if seedingContainsID(got.Primary, "b-cold") {
		t.Fatalf("b-cold (0 callers < 0.25*8=2) must be excluded from the seed tier, got Primary=%v", got.Primary)
	}
	if !seedingContainsID(got.Primary, "a-hot") {
		t.Fatalf("def0 (a-hot) must always be in the seed tier, got Primary=%v", got.Primary)
	}
}

// TestSeedingSmallOverload_ThreeDefsAllInjected pins the <=3 boundary
// itself (not just the 2-def case): exactly 3 defs still inject-all.
func TestSeedingSmallOverload_ThreeDefsAllInjected(t *testing.T) {
	nodes := map[string]*schema.Node{
		"a": {Id: "a", Kind: goextract.KindFunction, Name: "Process"},
		"b": {Id: "b", Kind: goextract.KindFunction, Name: "Process"},
		"c": {Id: "c", Kind: goextract.KindFunction, Name: "Process"},
	}
	r := &seedingFakeReader{nodes: nodes}

	res, err := seedNamedSymbols(r, "Process", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	if len(res.SeedIDs) != 3 {
		t.Fatalf("expected all 3 defs injected at the <=3 boundary, got SeedIDs=%v", res.SeedIDs)
	}
}
