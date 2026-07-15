package query

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// gatherFakeReader is a minimal in-memory graphstore.Reader used only to
// drive the H3-H6 hybrid gather channels against a fully-controlled
// synthetic node set (mirrors search_test.go's searchFakeReader; named
// with a gather-specific prefix per that file's own "each plan may need
// its own reader test double" convention — only IterateNodes is
// meaningful here).
type gatherFakeReader struct {
	nodes []*schema.Node
}

func (f *gatherFakeReader) GetNode(string) (*schema.Node, error) {
	return nil, errors.New("gatherFakeReader: GetNode not implemented")
}
func (f *gatherFakeReader) GetFile(string) (*schema.File, error) {
	return nil, errors.New("gatherFakeReader: GetFile not implemented")
}
func (f *gatherFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, errors.New("gatherFakeReader: GetMeta not implemented")
}
func (f *gatherFakeReader) GetMigration() ([]byte, error) {
	return nil, errors.New("gatherFakeReader: GetMigration not implemented")
}
func (f *gatherFakeReader) IterateEdges(string) (graphstore.EdgeIterator, error) {
	return nil, errors.New("gatherFakeReader: IterateEdges not implemented")
}
func (f *gatherFakeReader) IterateFiles() (graphstore.FileIterator, error) {
	return nil, errors.New("gatherFakeReader: IterateFiles not implemented")
}
func (f *gatherFakeReader) IterateFileIndex(string) (graphstore.FileIndexIterator, error) {
	return nil, errors.New("gatherFakeReader: IterateFileIndex not implemented")
}
func (f *gatherFakeReader) Close() error { return nil }

func (f *gatherFakeReader) IterateNodes() (graphstore.NodeIterator, error) {
	return &gatherFakeNodeIterator{nodes: f.nodes}, nil
}

type gatherFakeNodeIterator struct {
	nodes []*schema.Node
	i     int
}

func (it *gatherFakeNodeIterator) Next() bool {
	if it.i >= len(it.nodes) {
		return false
	}
	it.i++
	return true
}
func (it *gatherFakeNodeIterator) Node() *schema.Node { return it.nodes[it.i-1] }
func (it *gatherFakeNodeIterator) Err() error         { return nil }
func (it *gatherFakeNodeIterator) Close() error       { return nil }

func findGatherCandidate(t *testing.T, candidates []gatherCandidate, id string) gatherCandidate {
	t.Helper()
	for _, c := range candidates {
		if c.Node.Id == id {
			return c
		}
	}
	t.Fatalf("candidate %q not found in %+v", id, candidates)
	return gatherCandidate{}
}

// TestGatherChannel1_CoLocationBoost pins H3's exact constant: a file
// defining 2 distinct query symbols gives each of those nodes an
// additional +20*(2-1)=+20 boost over a node whose file has no
// co-located query symbol.
func TestGatherChannel1_CoLocationBoost(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:Foo:multi", Kind: goextract.KindStruct, Name: "Foo", FilePath: "pkg/multi.go"},
		{Id: "func:Bar:multi", Kind: goextract.KindFunction, Name: "Bar", FilePath: "pkg/multi.go"},
		{Id: "struct:Foo:single", Kind: goextract.KindStruct, Name: "Foo", FilePath: "pkg/single.go"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel1(r, []string{"Foo", "Bar"}, 8)
	if err != nil {
		t.Fatalf("gatherChannel1: unexpected error: %v", err)
	}

	coLocated := findGatherCandidate(t, got, "struct:Foo:multi")
	alone := findGatherCandidate(t, got, "struct:Foo:single")

	if delta := coLocated.Score - alone.Score; delta != channel1CoLocationBoost {
		t.Fatalf("co-location boost = %v, want exactly %v (coLocated=%v alone=%v)", delta, channel1CoLocationBoost, coLocated.Score, alone.Score)
	}

	other := findGatherCandidate(t, got, "func:Bar:multi")
	if other.Score != coLocated.Score {
		t.Fatalf("both co-located matches in the same file must earn the identical boost: Foo=%v Bar=%v", coLocated.Score, other.Score)
	}

	if !coLocated.Channels[gatherChannelExactName] {
		t.Errorf("expected gatherChannelExactName recorded on the candidate, got %+v", coLocated.Channels)
	}
}

// TestGatherChannel1_NoMatchNoCoLocationBoost asserts a node matched
// alone in its file (no second distinct query symbol present) gets no
// co-location boost at all — i.e. exactly channel1BaseScore.
func TestGatherChannel1_NoMatchNoCoLocationBoost(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:Foo:single", Kind: goextract.KindStruct, Name: "Foo", FilePath: "pkg/single.go"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel1(r, []string{"Foo"}, 8)
	if err != nil {
		t.Fatalf("gatherChannel1: unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1: %+v", len(got), got)
	}
	if got[0].Score != channel1BaseScore {
		t.Fatalf("score = %v, want exactly channel1BaseScore=%v (no co-location boost)", got[0].Score, channel1BaseScore)
	}
}

// TestGatherChannel1_TrimsToSearchLimitTimesTwo pins H3's cap: results
// are trimmed to searchLimit*2, keeping the highest-scoring entries.
func TestGatherChannel1_TrimsToSearchLimitTimesTwo(t *testing.T) {
	var nodes []*schema.Node
	for i := 0; i < 10; i++ {
		nodes = append(nodes, &schema.Node{
			Id:       string(rune('a' + i)),
			Kind:     goextract.KindFunction,
			Name:     "Widget",
			FilePath: "pkg/" + string(rune('a'+i)) + ".go",
		})
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel1(r, []string{"Widget"}, 2) // searchLimit=2 -> cap 4
	if err != nil {
		t.Fatalf("gatherChannel1: unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d candidates, want exactly searchLimit*2=4: %+v", len(got), got)
	}
}

// TestGatherChannel1_EmptySymbols asserts no symbols means no candidates
// (and no scan-time panic).
func TestGatherChannel1_EmptySymbols(t *testing.T) {
	r := &gatherFakeReader{nodes: []*schema.Node{{Id: "a", Name: "Foo"}}}
	got, err := gatherChannel1(r, nil, 8)
	if err != nil {
		t.Fatalf("gatherChannel1: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d candidates, want 0 for empty symbols", len(got))
	}
}

// TestGatherChannel2_TitlecasePrefixBrevity pins H4's exact formula:
// score = 15 + max(0, 10-(nameLen-prefixLen)/3). A name near the same
// length as the titlecased prefix scores near the full +10 brevity
// bonus; a much longer name scores less.
func TestGatherChannel2_TitlecasePrefixBrevity(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:Widget", Kind: goextract.KindStruct, Name: "Widget"},                                             // len 6, prefix "Widget" len 6 -> brevity 10
		{Id: "struct:WidgetServiceProviderFactory", Kind: goextract.KindStruct, Name: "WidgetServiceProviderFactory"}, // len 28 -> brevity 10-22/3
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel2(r, []string{"widget"})
	if err != nil {
		t.Fatalf("gatherChannel2: unexpected error: %v", err)
	}

	exact := findGatherCandidate(t, got, "struct:Widget")
	wantExact := channel2PrefixBase + channel2BrevityMax
	if exact.Score != wantExact {
		t.Fatalf("near-exact-length name score = %v, want %v (15 + full brevity bonus)", exact.Score, wantExact)
	}

	long := findGatherCandidate(t, got, "struct:WidgetServiceProviderFactory")
	wantLongBrevity := channel2BrevityMax - float64(len("WidgetServiceProviderFactory")-len("Widget"))/channel2BrevityDiv
	wantLong := channel2PrefixBase + wantLongBrevity
	if long.Score != wantLong {
		t.Fatalf("long name score = %v, want %v", long.Score, wantLong)
	}
	if long.Score >= exact.Score {
		t.Fatalf("expected long name score (%v) < near-exact-length name score (%v)", long.Score, exact.Score)
	}

	if !exact.Channels[gatherChannelTitlePrefix] {
		t.Errorf("expected gatherChannelTitlePrefix recorded, got %+v", exact.Channels)
	}
}

// TestGatherChannel2_OnlyDefinitionKindsParticipate asserts a
// non-definition kind (e.g. a function) with a matching titlecase prefix
// is excluded, even though its name matches.
func TestGatherChannel2_OnlyDefinitionKindsParticipate(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:Widget", Kind: goextract.KindStruct, Name: "Widget"},
		{Id: "func:WidgetHelper", Kind: goextract.KindFunction, Name: "WidgetHelper"},
		{Id: "iface:WidgetLike", Kind: goextract.KindInterface, Name: "WidgetLike"},
		{Id: "alias:WidgetAlias", Kind: goextract.KindTypeAlias, Name: "WidgetAlias"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel2(r, []string{"widget"})
	if err != nil {
		t.Fatalf("gatherChannel2: unexpected error: %v", err)
	}

	ids := make(map[string]bool, len(got))
	for _, c := range got {
		ids[c.Node.Id] = true
	}
	if ids["func:WidgetHelper"] {
		t.Errorf("expected function-kind node excluded from channel 2, got it in results: %+v", got)
	}
	for _, want := range []string{"struct:Widget", "iface:WidgetLike", "alias:WidgetAlias"} {
		if !ids[want] {
			t.Errorf("expected definition-kind node %q included in channel 2 results, got %+v", want, got)
		}
	}
}

// TestGatherChannel2_NoPrefixMatchExcluded asserts a definition-kind node
// whose name does not start with any titlecased query token is excluded.
func TestGatherChannel2_NoPrefixMatchExcluded(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "struct:Gadget", Kind: goextract.KindStruct, Name: "Gadget"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel2(r, []string{"widget"})
	if err != nil {
		t.Fatalf("gatherChannel2: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d candidates, want 0 (no prefix match): %+v", len(got), got)
	}
}

// TestIsTestFile pins RESEARCH §5's exact rule set: filename patterns
// (test_*/*_test.*/*.test.*/*-spec.*/*Test.ext/*Tests.ext/*TestCase.ext/
// *Spec.ext), directory patterns (/tests?//__tests__//specs?//testlib/
// /testing/, CamelCase *Test* Gradle/Kotlin dirs), and non-production
// dirs (integration, sample(s), example(s), fixture(s), benchmark(s),
// demo(s)).
func TestIsTestFile(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"plain go file", "internal/query/gather.go", false},
		{"go _test.go suffix", "internal/query/gather_test.go", true},
		{"nested _test.go suffix", "internal/pkg/foo_test.go", true},
		{"js .test. infix", "src/foo.test.js", true},
		{"hyphen -spec. infix", "src/foo-spec.js", true},
		{"test_ prefix", "test_foo.py", true},
		{"__tests__ dir", "src/__tests__/foo.js", true},
		{"tests dir", "src/tests/foo.go", true},
		{"test dir singular", "src/test/foo.go", true},
		{"spec dir", "spec/foo_helper.rb", true},
		{"testlib dir", "internal/testlib/util.go", true},
		{"testing dir", "internal/testing/helper.go", true},
		{"CamelCase Test dir (Gradle/Kotlin)", "app/src/androidTest/java/FooTest.java", true},
		{"PascalCase Test.ext suffix", "FooTest.java", true},
		{"PascalCase Tests.ext suffix", "FooTests.cs", true},
		{"PascalCase TestCase.ext suffix", "FooTestCase.py", true},
		{"PascalCase Spec.ext suffix", "FooSpec.kt", true},
		{"integration dir (non-production)", "integration/setup.go", true},
		{"examples dir (non-production)", "examples/demo.go", true},
		{"fixtures dir (non-production)", "fixtures/data.go", true},
		{"benchmarks dir (non-production)", "benchmarks/bench_test.go", true},
		{"demo dir (non-production)", "demo/main.go", true},
		{"unrelated PascalCase filename", "Manifest.go", false},
		{"empty path", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTestFile(tc.path); got != tc.want {
				t.Errorf("isTestFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// TestGatherChannel3_MultiTermBoost pins H5's exact constant: a node
// matching 3 distinct query terms scores channel3BaseScore +
// 5*(3-1)=+10 over a node matching only 1 term.
func TestGatherChannel3_MultiTermBoost(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "func:fooBarBaz", Kind: goextract.KindFunction, Name: "fooBarBaz", QualifiedName: "pkg.fooBarBaz"},
		{Id: "func:foo", Kind: goextract.KindFunction, Name: "fooOnly", QualifiedName: "pkg.fooOnly"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel3(r, []string{"foo", "bar", "baz"}, "")
	if err != nil {
		t.Fatalf("gatherChannel3: unexpected error: %v", err)
	}

	multi := findGatherCandidate(t, got, "func:fooBarBaz")
	single := findGatherCandidate(t, got, "func:foo")

	wantMulti := channel3BaseScore + channel3MultiTermBoost*2
	if multi.Score != wantMulti {
		t.Fatalf("3-term-hit score = %v, want %v (base + 5*(3-1))", multi.Score, wantMulti)
	}
	if single.Score != channel3BaseScore {
		t.Fatalf("1-term-hit score = %v, want exactly channel3BaseScore=%v (no multi-term boost)", single.Score, channel3BaseScore)
	}
	if delta := multi.Score - single.Score; delta != channel3MultiTermBoost*2 {
		t.Fatalf("multi-term boost delta = %v, want exactly %v", delta, channel3MultiTermBoost*2)
	}

	if !multi.Channels[gatherChannelFTS] {
		t.Errorf("expected gatherChannelFTS recorded, got %+v", multi.Channels)
	}
}

// TestGatherChannel3_ExcludesImportKindWithoutFilter asserts an
// import-Kind node is excluded from channel 3 results when no explicit
// kind filter is given, but included when the caller explicitly filters
// to "import".
func TestGatherChannel3_ExcludesImportKindWithoutFilter(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "import:foo", Kind: "import", Name: "foo", QualifiedName: "foo"},
	}
	r := &gatherFakeReader{nodes: nodes}

	withoutFilter, err := gatherChannel3(r, []string{"foo"}, "")
	if err != nil {
		t.Fatalf("gatherChannel3: unexpected error: %v", err)
	}
	if len(withoutFilter) != 0 {
		t.Fatalf("expected import-kind node excluded without an explicit kind filter, got %+v", withoutFilter)
	}

	withFilter, err := gatherChannel3(r, []string{"foo"}, "import")
	if err != nil {
		t.Fatalf("gatherChannel3: unexpected error: %v", err)
	}
	if len(withFilter) != 1 || withFilter[0].Node.Id != "import:foo" {
		t.Fatalf("expected import-kind node included with an explicit kind filter, got %+v", withFilter)
	}
}

// TestGatherChannel3_NoTermHitExcluded asserts a node matching none of
// the terms is excluded entirely (not merely scored 0).
func TestGatherChannel3_NoTermHitExcluded(t *testing.T) {
	nodes := []*schema.Node{
		{Id: "func:unrelated", Kind: goextract.KindFunction, Name: "unrelated", QualifiedName: "pkg.unrelated"},
	}
	r := &gatherFakeReader{nodes: nodes}

	got, err := gatherChannel3(r, []string{"foo"}, "")
	if err != nil {
		t.Fatalf("gatherChannel3: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d candidates, want 0 (no term hit): %+v", len(got), got)
	}
}

// TestGatherMerge_MaxScoreWinsNotSummed pins H6's exact merge rule: a
// node hit by two channels merges once, keeping the MAX score seen (not
// the sum), and the union of channels that hit it.
func TestGatherMerge_MaxScoreWinsNotSummed(t *testing.T) {
	node := &schema.Node{Id: "struct:Widget", Kind: goextract.KindStruct, Name: "Widget"}

	channel1 := []gatherCandidate{
		{Node: node, Score: 30, Channels: map[gatherChannelKind]bool{gatherChannelExactName: true}},
	}
	channel3 := []gatherCandidate{
		{Node: node, Score: 15, Channels: map[gatherChannelKind]bool{gatherChannelFTS: true}},
	}

	got := gatherMerge(channel1, channel3)
	if len(got) != 1 {
		t.Fatalf("got %d merged candidates, want exactly 1 (deduped by node id): %+v", len(got), got)
	}
	if got[0].Score != 30 {
		t.Fatalf("merged score = %v, want exactly 30 (max, not 45=sum)", got[0].Score)
	}
	if !got[0].Channels[gatherChannelExactName] || !got[0].Channels[gatherChannelFTS] {
		t.Fatalf("expected the union of both channels recorded, got %+v", got[0].Channels)
	}
}

// TestGatherMerge_UnionAcrossDistinctNodes asserts candidates from
// different channels for DIFFERENT node ids are all preserved (a plain
// union, not just a max-collapse).
func TestGatherMerge_UnionAcrossDistinctNodes(t *testing.T) {
	a := &schema.Node{Id: "a", Name: "A"}
	b := &schema.Node{Id: "b", Name: "B"}
	c := &schema.Node{Id: "c", Name: "C"}

	channel1 := []gatherCandidate{{Node: a, Score: 10, Channels: map[gatherChannelKind]bool{gatherChannelExactName: true}}}
	channel2 := []gatherCandidate{{Node: b, Score: 20, Channels: map[gatherChannelKind]bool{gatherChannelTitlePrefix: true}}}
	channel3 := []gatherCandidate{{Node: c, Score: 5, Channels: map[gatherChannelKind]bool{gatherChannelFTS: true}}}

	got := gatherMerge(channel1, channel2, channel3)
	if len(got) != 3 {
		t.Fatalf("got %d merged candidates, want 3 (a, b, c all distinct): %+v", len(got), got)
	}
	// Deterministic order: score-descending then Id-ascending (D-04).
	wantOrder := []string{"b", "a", "c"}
	for i, id := range wantOrder {
		if got[i].Node.Id != id {
			t.Fatalf("position %d: got %q, want %q (full: %+v)", i, got[i].Node.Id, id, got)
		}
	}
}

// TestGatherMerge_Empty asserts merging zero channels (or channels with
// no candidates) returns an empty, non-nil-panicking slice.
func TestGatherMerge_Empty(t *testing.T) {
	got := gatherMerge()
	if len(got) != 0 {
		t.Fatalf("got %d candidates, want 0", len(got))
	}
	got = gatherMerge(nil, []gatherCandidate{})
	if len(got) != 0 {
		t.Fatalf("got %d candidates, want 0", len(got))
	}
}
