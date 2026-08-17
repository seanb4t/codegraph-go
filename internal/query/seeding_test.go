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
		"a-hot":   {Id: "a-hot", Kind: goextract.KindFunction, Name: "Process"},
		"b-cold":  {Id: "b-cold", Kind: goextract.KindFunction, Name: "Process"},
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
		"a-hot":   {Id: "a-hot", Kind: goextract.KindFunction, Name: "Process"},
		"b-cold":  {Id: "b-cold", Kind: goextract.KindFunction, Name: "Process"},
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

// --- TestSeedingLargeOverload: >3-defs type-token-corroborated (<=4) /
// top-1-by-substance disambiguation ---

// sixOverloadedDefs builds 6 "Process" methods, each owned (via a
// "contains" edge) by a distinct type, plus the 6 owner type nodes
// themselves — the >3-defs fixture shared by the large-overload tests.
// ownerNames lets each test choose which owner type names corroborate.
func sixOverloadedDefs(ownerNames [6]string, spans [6]int32) (nodes map[string]*schema.Node, edges []*schema.Edge, defs []*schema.Node) {
	nodes = make(map[string]*schema.Node)
	defIDs := [6]string{"d1", "d2", "d3", "d4", "d5", "d6"}
	ownerIDs := [6]string{"own1", "own2", "own3", "own4", "own5", "own6"}
	for i := 0; i < 6; i++ {
		nodes[ownerIDs[i]] = &schema.Node{Id: ownerIDs[i], Kind: goextract.KindStruct, Name: ownerNames[i]}
		nodes[defIDs[i]] = &schema.Node{
			Id: defIDs[i], Kind: goextract.KindMethod, Name: "Process",
			StartLine: 1, EndLine: spans[i],
		}
		edges = append(edges, &schema.Edge{Source: ownerIDs[i], Target: defIDs[i], Kind: goextract.RefKindContains})
		defs = append(defs, nodes[defIDs[i]])
	}
	return nodes, edges, defs
}

// TestSeedingLargeOverload_TypeTokenCorroboration pins H13's >3-defs
// corroboration rule directly: a query PascalCase type token matching 2
// of 6 defs' OWNING types (here, 2 owners both named "OrderProcessor")
// seeds exactly those 2 corroborated defs — the other 4, whose owners
// don't match, are excluded.
func TestSeedingLargeOverload_TypeTokenCorroboration(t *testing.T) {
	owners := [6]string{"OrderProcessor", "OrderProcessor", "InvoiceHandler", "ReportBuilder", "ExportWriter", "ImportReader"}
	spans := [6]int32{5, 5, 5, 5, 5, 5}
	nodes, edges, defs := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	got, err := largeOverloadSeed(r, defs, []string{"OrderProcessor"})
	if err != nil {
		t.Fatalf("largeOverloadSeed: unexpected error: %v", err)
	}
	if len(got) != 2 || !seedingContainsID(got, "d1") || !seedingContainsID(got, "d2") {
		t.Fatalf("expected exactly the 2 defs owned by the matching type token, got %v", got)
	}
}

// TestSeedingLargeOverload_CorroboratedCapAtFour pins the <=4 cap: 5 of 6
// defs corroborate, but only the first 4 (in Id-sorted order) are seeded.
func TestSeedingLargeOverload_CorroboratedCapAtFour(t *testing.T) {
	owners := [6]string{"Matcher", "Matcher", "Matcher", "Matcher", "Matcher", "Other"}
	spans := [6]int32{5, 5, 5, 5, 5, 5}
	nodes, edges, defs := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	got, err := largeOverloadSeed(r, defs, []string{"Matcher"})
	if err != nil {
		t.Fatalf("largeOverloadSeed: unexpected error: %v", err)
	}
	if len(got) != largeOverloadCorroboratedCap {
		t.Fatalf("expected the corroborated set capped at %d, got %d: %v", largeOverloadCorroboratedCap, len(got), got)
	}
	for _, want := range []string{"d1", "d2", "d3", "d4"} {
		if !seedingContainsID(got, want) {
			t.Fatalf("expected the first %d (Id-sorted) corroborated defs, missing %q in %v", largeOverloadCorroboratedCap, want, got)
		}
	}
	if seedingContainsID(got, "d5") {
		t.Fatalf("expected the 5th corroborated def dropped by the <=4 cap, got %v", got)
	}
}

// TestSeedingLargeOverload_TopOneBySubstance pins the no-corroboration
// fallback: when no query type token corroborates any of the 6 defs, the
// single def with the greatest body substance (line span) is seeded
// alone.
func TestSeedingLargeOverload_TopOneBySubstance(t *testing.T) {
	owners := [6]string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta"}
	spans := [6]int32{5, 5, 5, 40, 5, 5} // d4 has by far the largest body
	nodes, edges, defs := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	got, err := largeOverloadSeed(r, defs, []string{"NoMatch"})
	if err != nil {
		t.Fatalf("largeOverloadSeed: unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "d4" {
		t.Fatalf("expected the single greatest-body-substance def (d4) seeded alone, got %v", got)
	}
}

// TestSeedingLargeOverload_NoTypeTokensFallsBackToSubstance pins the
// empty-typeTokens case (a query naming only the overloaded symbol
// itself, no PascalCase bias token at all): falls straight to
// top-1-by-substance.
func TestSeedingLargeOverload_NoTypeTokensFallsBackToSubstance(t *testing.T) {
	owners := [6]string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta"}
	spans := [6]int32{5, 5, 5, 5, 5, 40} // d6 has the largest body
	nodes, edges, defs := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	got, err := largeOverloadSeed(r, defs, nil)
	if err != nil {
		t.Fatalf("largeOverloadSeed: unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "d6" {
		t.Fatalf("expected the single greatest-body-substance def (d6) seeded alone, got %v", got)
	}
}

// TestSeedingLargeOverload_WiredThroughSeedNamedSymbols is a thin
// integration check that seedNamedSymbols actually calls the >3-def
// branch (not just leaves it stubbed): 6 real "Process" defs behind a
// query naming only "Process" (no corroborating token in scope) — the
// "Process" entry seeds exactly one def, the top-substance one.
func TestSeedingLargeOverload_WiredThroughSeedNamedSymbols(t *testing.T) {
	owners := [6]string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta"}
	spans := [6]int32{5, 5, 5, 5, 40, 5} // d5 has the largest body
	nodes, edges, _ := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	res, err := seedNamedSymbols(r, "Process", "")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	var got *seedName
	for i := range res.Names {
		if res.Names[i].Name == "Process" {
			got = &res.Names[i]
		}
	}
	if got == nil {
		t.Fatalf("expected a resolved %q entry, got Names=%+v", "Process", res.Names)
	}
	if len(got.Injected) != 1 || got.Injected[0] != "d5" {
		t.Fatalf("expected exactly the top-substance def (d5) seeded for the >3-def branch, got Injected=%v", got.Injected)
	}
	if len(got.Primary) != 1 || got.Primary[0] != "d5" {
		t.Fatalf("expected Primary == Injected for the large-overload branch (the selection IS the tier), got Primary=%v", got.Primary)
	}
}

// --- TestSeedingProjectNameExcluded: project name never biases overload selection ---

// TestSeedingProjectNameExcluded_TokenFiltered is a direct unit test of
// pascalCaseTypeTokens: the project name must never appear in the
// PascalCase type-token bias set, matched case-insensitively.
func TestSeedingProjectNameExcluded_TokenFiltered(t *testing.T) {
	got := pascalCaseTypeTokens("Widget MyProject Report myproject", "MyProject")
	if seedingContainsID(got, "MyProject") {
		t.Fatalf("expected the project name excluded (case-sensitive spelling), got %v", got)
	}
	if !seedingContainsID(got, "Widget") || !seedingContainsID(got, "Report") {
		t.Fatalf("expected other PascalCase tokens preserved, got %v", got)
	}
}

// TestSeedingProjectNameExcluded_NoFalseCorroboration pins the
// end-to-end behavior: when the ONLY PascalCase bias token in a query is
// the project's own name, and one def's owning type happens to share
// that name, the project-name exclusion must prevent that def from being
// spuriously corroborated — seedNamedSymbols must fall back to
// top-1-by-substance instead.
func TestSeedingProjectNameExcluded_NoFalseCorroboration(t *testing.T) {
	owners := [6]string{"Acme", "Beta", "Gamma", "Delta", "Epsilon", "Zeta"}
	spans := [6]int32{1, 40, 1, 1, 1, 1} // d2 (owned by "Beta") has the largest body, NOT d1 (owned by "Acme")
	nodes, edges, _ := sixOverloadedDefs(owners, spans)
	r := &seedingFakeReader{nodes: nodes, edges: edges}

	res, err := seedNamedSymbols(r, "Process Acme", "Acme")
	if err != nil {
		t.Fatalf("seedNamedSymbols: unexpected error: %v", err)
	}
	var got *seedName
	for i := range res.Names {
		if res.Names[i].Name == "Process" {
			got = &res.Names[i]
		}
	}
	if got == nil {
		t.Fatalf("expected a resolved %q entry, got Names=%+v", "Process", res.Names)
	}
	if seedingContainsID(got.Injected, "d1") {
		t.Fatalf("expected d1 (owned by the excluded project-name type) NEVER corroborated, got Injected=%v", got.Injected)
	}
	if len(got.Injected) != 1 || got.Injected[0] != "d2" {
		t.Fatalf("expected the fallback top-1-by-substance def (d2) seeded instead, got Injected=%v", got.Injected)
	}
}
