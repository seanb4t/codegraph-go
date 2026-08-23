package query

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// newDetailFixtureEngine builds a small, fully-controlled Engine over an
// in-repo t.TempDir() fixture — not a fetched corpus (the corpus-scale
// proof is plan 01-02's TestGoldensMatchLiveEngineOutput oracle; this
// file's job is the NodeDetail shape contract). It follows node_test.go's
// existing traverseFakeReader pattern (see traverse_test.go), paired with
// NewWithRoot over a real temp directory so readSourceFile's on-disk
// reads exercise real files, not a mock.
//
// Node/edge shape:
//   - Alpha:  1 match,  Calls=[Beta],  CalledBy=[Gamma]  (single-def, non-trivial)
//   - Beta:   not queried directly; exists only as Alpha/Multi1's call target
//   - Gamma:  1 match,  Calls=[Alpha], CalledBy=nil
//   - Solo:   1 match,  Calls=nil,     CalledBy=nil       (nilness-preservation case)
//   - Multi:  2 matches (multi01 Calls=[Beta], multi02 Calls=nil) — proves
//     each multi-def candidate's Definition returns THAT candidate's own
//     source/calls/calledBy, not a shared or collapsed value.
func newDetailFixtureEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()

	nodes := map[string]*schema.Node{
		"alpha1":  {Id: "alpha1", Name: "Alpha", Kind: "function", FilePath: "alpha.go", StartLine: 1, EndLine: 3, Signature: "func Alpha()"},
		"beta1":   {Id: "beta1", Name: "Beta", Kind: "function", FilePath: "beta.go", StartLine: 1, EndLine: 3, Signature: "func Beta()"},
		"gamma1":  {Id: "gamma1", Name: "Gamma", Kind: "function", FilePath: "gamma.go", StartLine: 1, EndLine: 3, Signature: "func Gamma()"},
		"solo1":   {Id: "solo1", Name: "Solo", Kind: "function", FilePath: "solo.go", StartLine: 1, EndLine: 3, Signature: "func Solo()"},
		"multi01": {Id: "multi01", Name: "Multi", Kind: "function", FilePath: "multi1.go", StartLine: 1, EndLine: 3, Signature: "func Multi() // 1"},
		"multi02": {Id: "multi02", Name: "Multi", Kind: "function", FilePath: "multi2.go", StartLine: 1, EndLine: 3, Signature: "func Multi() // 2"},
	}
	edges := []*schema.Edge{
		{Source: "alpha1", Target: "beta1", Kind: goextract.RefKindCalls},
		{Source: "gamma1", Target: "alpha1", Kind: goextract.RefKindCalls},
		{Source: "multi01", Target: "beta1", Kind: goextract.RefKindCalls},
	}

	files := map[string]string{
		"alpha.go":   "package p\n\nfunc Alpha() {}\n",
		"beta.go":    "package p\n\nfunc Beta() {}\n",
		"gamma.go":   "package p\n\nfunc Gamma() {}\n",
		"solo.go":    "package p\n\nfunc Solo() {}\n",
		"multi1.go":  "package p\n\nfunc Multi() {} // 1\n",
		"multi2.go":  "package p\n\nfunc Multi() {} // 2\n",
		"readme.txt": "hello\nworld\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file %s: %v", name, err)
		}
	}

	return NewWithRoot(&traverseFakeReader{nodes: nodes, edges: edges}, dir)
}

// TestNodeDetailCoversAllThreeShapes pins D-02: NodeDetail covers all
// three shapes Node() can return, with exactly one of File, Definition
// and Multi populated per mode.
func TestNodeDetailCoversAllThreeShapes(t *testing.T) {
	e := newDetailFixtureEngine(t)

	t.Run("file", func(t *testing.T) {
		d, err := e.NodeDetail("", "readme.txt", nil)
		if err != nil {
			t.Fatalf("NodeDetail(file): unexpected error: %v", err)
		}
		if d.Mode != NodeDetailModeFile {
			t.Fatalf("NodeDetail(file): got Mode %v, want NodeDetailModeFile", d.Mode)
		}
		if d.File == nil || d.Definition != nil || d.Multi != nil {
			t.Fatalf("NodeDetail(file): got File=%v Definition=%v Multi=%v, want ONLY File non-nil", d.File, d.Definition, d.Multi)
		}
		want, err := os.ReadFile(filepath.Join(e.repoRoot, "readme.txt"))
		if err != nil {
			t.Fatalf("read fixture file: %v", err)
		}
		if !bytes.Equal(d.File.Source, want) {
			t.Fatalf("NodeDetail(file): Source = %q, want %q", d.File.Source, want)
		}
	})

	t.Run("single-def", func(t *testing.T) {
		d, err := e.NodeDetail("Alpha", "", nil)
		if err != nil {
			t.Fatalf("NodeDetail(Alpha): unexpected error: %v", err)
		}
		if d.Mode != NodeDetailModeSingleDef {
			t.Fatalf("NodeDetail(Alpha): got Mode %v, want NodeDetailModeSingleDef", d.Mode)
		}
		if d.Definition == nil || d.File != nil || d.Multi != nil {
			t.Fatalf("NodeDetail(Alpha): got File=%v Definition=%v Multi=%v, want ONLY Definition non-nil", d.File, d.Definition, d.Multi)
		}
		got, err := e.Node("Alpha", "", nil)
		if err != nil {
			t.Fatalf("Node(Alpha): unexpected error: %v", err)
		}
		want := RenderNode(d.Definition.Node, d.Definition.Calls, d.Definition.CalledBy)
		if got != want {
			t.Fatalf("Node(Alpha) = %q, want RenderNode(d.Definition.Node, d.Definition.Calls, d.Definition.CalledBy) = %q", got, want)
		}
	})

	t.Run("multi-def", func(t *testing.T) {
		d, err := e.NodeDetail("Multi", "", nil)
		if err != nil {
			t.Fatalf("NodeDetail(Multi): unexpected error: %v", err)
		}
		if d.Mode != NodeDetailModeMultiDef {
			t.Fatalf("NodeDetail(Multi): got Mode %v, want NodeDetailModeMultiDef", d.Mode)
		}
		if d.Multi == nil || d.File != nil || d.Definition != nil {
			t.Fatalf("NodeDetail(Multi): got File=%v Definition=%v Multi=%v, want ONLY Multi non-nil", d.File, d.Definition, d.Multi)
		}
		if d.Multi.Symbol != "Multi" {
			t.Fatalf("NodeDetail(Multi): Multi.Symbol = %q, want %q", d.Multi.Symbol, "Multi")
		}
		if len(d.Multi.Matches) != 2 {
			t.Fatalf("NodeDetail(Multi): got %d matches, want 2", len(d.Multi.Matches))
		}

		// Each candidate's Definition must return THAT candidate's own
		// source, calls and called-by — not a shared or collapsed value.
		for _, m := range d.Multi.Matches {
			dd, err := d.Multi.Definition(m)
			if err != nil {
				t.Fatalf("Multi.Definition(%s): unexpected error: %v", m.Id, err)
			}
			if dd.Node != m {
				t.Fatalf("Multi.Definition(%s): Node = %v, want the same match pointer %v", m.Id, dd.Node, m)
			}
			wantSource, err := os.ReadFile(filepath.Join(e.repoRoot, m.FilePath))
			if err != nil {
				t.Fatalf("read fixture file %s: %v", m.FilePath, err)
			}
			if !bytes.Equal(dd.Source, wantSource) {
				t.Fatalf("Multi.Definition(%s): Source = %q, want %q", m.Id, dd.Source, wantSource)
			}
			switch m.Id {
			case "multi01":
				if len(dd.Calls) != 1 || dd.Calls[0].Id != "beta1" {
					t.Fatalf("Multi.Definition(multi01): Calls = %v, want exactly [beta1]", dd.Calls)
				}
			case "multi02":
				if dd.Calls != nil {
					t.Fatalf("Multi.Definition(multi02): Calls = %v, want nil (multi02 calls nothing)", dd.Calls)
				}
			default:
				t.Fatalf("unexpected match id %q in Multi.Matches", m.Id)
			}
		}
	})
}

// TestNodeDetailSingleVsMultiBoundary pins that the single/multi split is
// at the COUNT of narrowed matches — exactly one versus exactly two —
// never a heuristic.
func TestNodeDetailSingleVsMultiBoundary(t *testing.T) {
	e := newDetailFixtureEngine(t)

	t.Run("exactly-one-match", func(t *testing.T) {
		d, err := e.NodeDetail("Alpha", "", nil)
		if err != nil {
			t.Fatalf("NodeDetail(Alpha): unexpected error: %v", err)
		}
		if d.Mode != NodeDetailModeSingleDef {
			t.Fatalf("NodeDetail(Alpha): got Mode %v, want NodeDetailModeSingleDef for a symbol narrowing to exactly one match", d.Mode)
		}
	})

	t.Run("exactly-two-matches", func(t *testing.T) {
		d, err := e.NodeDetail("Multi", "", nil)
		if err != nil {
			t.Fatalf("NodeDetail(Multi): unexpected error: %v", err)
		}
		if d.Mode != NodeDetailModeMultiDef {
			t.Fatalf("NodeDetail(Multi): got Mode %v, want NodeDetailModeMultiDef for a symbol narrowing to exactly two matches", d.Mode)
		}
		if len(d.Multi.Matches) != 2 {
			t.Fatalf("NodeDetail(Multi): got %d matches, want exactly 2", len(d.Multi.Matches))
		}
	})
}

// TestNodeDetailPreservesFetcherNilness pins ENG-01's "the builders do
// not normalise" property: for a symbol with no callers and no callees,
// the gathered Calls/CalledBy are exactly what fetchCalls/fetchCalledBy
// returned (nil, not an empty non-nil slice), and Node()'s rendered bytes
// are unchanged from what RenderNode produces over that exact detail.
// This asserts the extraction does not normalise; it makes no claim
// about how the renderer itself treats a nil slice.
func TestNodeDetailPreservesFetcherNilness(t *testing.T) {
	e := newDetailFixtureEngine(t)

	d, err := e.NodeDetail("Solo", "", nil)
	if err != nil {
		t.Fatalf("NodeDetail(Solo): unexpected error: %v", err)
	}
	if d.Mode != NodeDetailModeSingleDef || d.Definition == nil {
		t.Fatalf("NodeDetail(Solo): got Mode %v Definition %v, want NodeDetailModeSingleDef with a non-nil Definition", d.Mode, d.Definition)
	}
	if d.Definition.Calls != nil {
		t.Fatalf("NodeDetail(Solo): Definition.Calls = %#v, want exactly nil (fetchCalls returned nil; must not be normalised to an empty slice)", d.Definition.Calls)
	}
	if d.Definition.CalledBy != nil {
		t.Fatalf("NodeDetail(Solo): Definition.CalledBy = %#v, want exactly nil (fetchCalledBy returned nil; must not be normalised to an empty slice)", d.Definition.CalledBy)
	}

	got, err := e.Node("Solo", "", nil)
	if err != nil {
		t.Fatalf("Node(Solo): unexpected error: %v", err)
	}
	want := RenderNode(d.Definition.Node, d.Definition.Calls, d.Definition.CalledBy)
	if got != want {
		t.Fatalf("Node(Solo) = %q, want RenderNode over the preserved-nil detail = %q", got, want)
	}
}

// TestNodeDetailErrorsMatchNode pins that Node() and (*Engine).NodeDetail
// return byte-identical errors for the same three failure inputs: the
// missing-arguments error, the symbol-not-found error, and
// resolveSourcePath's repo-root confinement refusal reached through the
// file-only branch. Literals captured from internal/query/node.go's
// pre-extraction source (read in full this session): the argument error
// at Node()'s empty-symbol/empty-file branch, the not-found error after
// enumerateSymbolDefs yields nothing, and resolveSourcePath's ".."-prefix
// escape check.
func TestNodeDetailErrorsMatchNode(t *testing.T) {
	e := newDetailFixtureEngine(t)

	t.Run("empty-arguments", func(t *testing.T) {
		_, nodeErr := e.Node("", "", nil)
		_, detailErr := e.NodeDetail("", "", nil)
		if nodeErr == nil || detailErr == nil {
			t.Fatalf(`Node/NodeDetail("","",nil): got nodeErr=%v detailErr=%v, want both non-nil`, nodeErr, detailErr)
		}
		if nodeErr.Error() != detailErr.Error() {
			t.Fatalf(`Node/NodeDetail("","",nil): error strings differ: Node=%q NodeDetail=%q`, nodeErr.Error(), detailErr.Error())
		}
		const want = "query: node requires a symbol name or a file path"
		if nodeErr.Error() != want {
			t.Fatalf(`Node("","",nil) error = %q, want recorded literal %q`, nodeErr.Error(), want)
		}
	})

	t.Run("symbol-not-found", func(t *testing.T) {
		_, nodeErr := e.Node("nosuchsymbol", "", nil)
		_, detailErr := e.NodeDetail("nosuchsymbol", "", nil)
		if nodeErr == nil || detailErr == nil {
			t.Fatalf("Node/NodeDetail(nosuchsymbol): got nodeErr=%v detailErr=%v, want both non-nil", nodeErr, detailErr)
		}
		if nodeErr.Error() != detailErr.Error() {
			t.Fatalf("Node/NodeDetail(nosuchsymbol): error strings differ: Node=%q NodeDetail=%q", nodeErr.Error(), detailErr.Error())
		}
		const want = `query: symbol "nosuchsymbol" not found`
		if nodeErr.Error() != want {
			t.Fatalf("Node(nosuchsymbol) error = %q, want recorded literal %q", nodeErr.Error(), want)
		}
	})

	t.Run("path-escapes-repo-root", func(t *testing.T) {
		_, nodeErr := e.Node("", "../outside.txt", nil)
		_, detailErr := e.NodeDetail("", "../outside.txt", nil)
		if nodeErr == nil || detailErr == nil {
			t.Fatalf("Node/NodeDetail(escaping path): got nodeErr=%v detailErr=%v, want both non-nil", nodeErr, detailErr)
		}
		if nodeErr.Error() != detailErr.Error() {
			t.Fatalf("Node/NodeDetail(escaping path): error strings differ: Node=%q NodeDetail=%q", nodeErr.Error(), detailErr.Error())
		}
		const want = `query: path "../outside.txt" escapes the repo root`
		if nodeErr.Error() != want {
			t.Fatalf("Node(escaping path) error = %q, want recorded literal %q", nodeErr.Error(), want)
		}
	})
}

// TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates proves
// the multi-definition laziness is real, not incidental: a fixture
// declares one more definition of "Big" than nodeMultiDefHardCap, with
// the LAST candidate (the one RenderNodeMultiDef's hard-cap early-
// continue never fetches) backed by a DIRECTORY rather than a regular
// file. os.ReadFile on a directory reliably fails with "is a directory"
// regardless of the running user's privileges — unlike a permission-bit
// approach, which a root-run test process would not observe — so this
// deterministically proves unreadability without a skip-if-root escape
// hatch. Both Node and NodeDetail must succeed (neither reads that
// candidate); asking the multi-definition value for that candidate
// explicitly must return the read error, proving the data is reachable
// on demand rather than silently swallowed.
func TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates(t *testing.T) {
	dir := t.TempDir()
	const total = nodeMultiDefHardCap + 1 // one more definition than the hard cap ever fetches

	nodes := make(map[string]*schema.Node, total)
	var lastID string
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("big%02d", i)
		filePath := fmt.Sprintf("big%02d.go", i)
		nodes[id] = &schema.Node{Id: id, Name: "Big", Kind: "function", FilePath: filePath, StartLine: 1, EndLine: 1, Signature: "func Big()"}

		if i == total-1 {
			if err := os.MkdirAll(filepath.Join(dir, filePath), 0o755); err != nil {
				t.Fatalf("mkdir unreadable candidate dir: %v", err)
			}
			lastID = id
			continue
		}
		content := fmt.Sprintf("package p\n\nfunc Big() {} // %d\n", i)
		if err := os.WriteFile(filepath.Join(dir, filePath), []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file %s: %v", filePath, err)
		}
	}

	e := NewWithRoot(&traverseFakeReader{nodes: nodes}, dir)

	if _, err := e.Node("Big", "", nil); err != nil {
		t.Fatalf("Node(Big): unexpected error — an out-of-cap unreadable candidate must never be read: %v", err)
	}

	d, err := e.NodeDetail("Big", "", nil)
	if err != nil {
		t.Fatalf("NodeDetail(Big): unexpected error — an out-of-cap unreadable candidate must never be read: %v", err)
	}
	if d.Mode != NodeDetailModeMultiDef {
		t.Fatalf("NodeDetail(Big): got Mode %v, want NodeDetailModeMultiDef", d.Mode)
	}

	var lastMatch *schema.Node
	for _, m := range d.Multi.Matches {
		if m.Id == lastID {
			lastMatch = m
		}
	}
	if lastMatch == nil {
		t.Fatalf("NodeDetail(Big): Matches does not include the last candidate %q", lastID)
	}

	if _, err := d.Multi.Definition(lastMatch); err == nil {
		t.Fatalf("Multi.Definition(%s): expected a read error for the directory-backed candidate, got nil — laziness is not real if this candidate was already read elsewhere", lastID)
	}
}

// TestMultiDefReverseAdjacencyBuiltOnce swaps the buildReverseAdjacency
// seam (internal/query/traverse.go) for a counting wrapper, restores it
// via t.Cleanup, and asserts the counter is exactly 1 after one
// NodeDetail call on a multi-definition symbol plus one Definition call
// per returned match — counting builds through the seam rather than
// asserting from the source.
func TestMultiDefReverseAdjacencyBuiltOnce(t *testing.T) {
	e := newDetailFixtureEngine(t)

	var buildCount int
	original := buildReverseAdjacency
	buildReverseAdjacency = func(r graphstore.Reader) (map[string][]*schema.Edge, error) {
		buildCount++
		return original(r)
	}
	t.Cleanup(func() { buildReverseAdjacency = original })

	d, err := e.NodeDetail("Multi", "", nil)
	if err != nil {
		t.Fatalf("NodeDetail(Multi): unexpected error: %v", err)
	}
	if d.Mode != NodeDetailModeMultiDef {
		t.Fatalf("NodeDetail(Multi): got Mode %v, want NodeDetailModeMultiDef", d.Mode)
	}
	for _, m := range d.Multi.Matches {
		if _, err := d.Multi.Definition(m); err != nil {
			t.Fatalf("Multi.Definition(%s): unexpected error: %v", m.Id, err)
		}
	}

	if buildCount != 1 {
		t.Fatalf("reverse-adjacency build count = %d after one NodeDetail call plus %d Definition calls, want exactly 1 (built once, shared across every candidate)", buildCount, len(d.Multi.Matches))
	}
}

// TestNodeErrorStringsAreUnchanged pins the three error strings
// Node/NodeDetail can produce — the argument error, the not-found error,
// and resolveSourcePath's repo-root confinement refusal — captured
// verbatim from internal/query/node.go's pre-extraction source (read in
// full this session; recorded in 01-04-SUMMARY.md), and asserts both
// entry points return them byte-identically for the same inputs.
func TestNodeErrorStringsAreUnchanged(t *testing.T) {
	e := newDetailFixtureEngine(t)

	cases := []struct {
		name         string
		symbol, file string
		line         *int
		want         string
	}{
		{"argument error", "", "", nil, "query: node requires a symbol name or a file path"},
		{"not-found error", "nosuchsymbol", "", nil, `query: symbol "nosuchsymbol" not found`},
		{"repo-root confinement refusal", "", "../outside.txt", nil, `query: path "../outside.txt" escapes the repo root`},
	}

	for _, c := range cases {
		_, nodeErr := e.Node(c.symbol, c.file, c.line)
		_, detailErr := e.NodeDetail(c.symbol, c.file, c.line)
		if nodeErr == nil || detailErr == nil {
			t.Fatalf("%s: got Node err=%v NodeDetail err=%v, want both non-nil", c.name, nodeErr, detailErr)
		}
		if nodeErr.Error() != c.want {
			t.Fatalf("%s: Node error = %q, want recorded literal %q", c.name, nodeErr.Error(), c.want)
		}
		if detailErr.Error() != c.want {
			t.Fatalf("%s: NodeDetail error = %q, want recorded literal %q", c.name, detailErr.Error(), c.want)
		}
	}
}

// zeroMetaFakeReader wraps traverseFakeReader with a GetMeta that returns
// graphstore.ErrNotFound rather than traverseFakeReader's own "not
// implemented" error — buildExploreResult's GetMeta call tolerates
// ErrNotFound (a pre-upgrade/never-synced graph) but treats any other
// error as fatal, so an explore-pipeline fixture needs this override
// where a node-only fixture (newDetailFixtureEngine) never did.
type zeroMetaFakeReader struct {
	*traverseFakeReader
}

func (f *zeroMetaFakeReader) GetMeta() (*schema.Meta, error) {
	return nil, graphstore.ErrNotFound
}

// TestExploreResultCarriesRenderInputsUnchanged pins D-01's "the view
// model is the argument tuple RenderExplore already takes": a populated
// ExploreDetail result, fed into RenderExplore in the exact argument order
// Explore() itself uses, must render byte-identically to Explore()'s own
// output for the identical query/maxFiles — proving buildExploreResult
// changed no render input. Also pins bullet 6: Stale, Sources and
// SkeletonFiles carry through, with Sources keyed by each group's Path.
//
// Uses the local, network-free behavioral corpus fixture (behavioralEngine,
// explore_test.go) rather than the plan's illustrative hugo/"page content"
// example: this in-package unit test must not depend on fetching the
// external hugo/guava/serilog/requests golden corpora over the network.
// The corpus-scale byte-identity proof over all four of those corpora
// already exists and is what this plan re-runs after every task —
// TestGoldensMatchLiveEngineOutput (testdata/golden/byte_identity_test.go,
// plan 01-02).
func TestExploreResultCarriesRenderInputsUnchanged(t *testing.T) {
	e := behavioralEngine(t)

	const query = "account balance"
	const maxFiles = 5

	r, err := e.ExploreDetail(query, maxFiles)
	if err != nil {
		t.Fatalf("ExploreDetail(%q, %d): unexpected error: %v", query, maxFiles, err)
	}
	if r.Empty {
		t.Fatalf("ExploreDetail(%q, %d): got Empty=true, want a populated result", query, maxFiles)
	}
	if len(r.Groups) == 0 {
		t.Fatalf("ExploreDetail(%q, %d): got 0 Groups, want >= 1", query, maxFiles)
	}

	want, err := e.Explore(query, maxFiles)
	if err != nil {
		t.Fatalf("Explore(%q, %d): unexpected error: %v", query, maxFiles, err)
	}
	got := RenderExplore(r.Query, len(r.Groups), r.SymbolCount, r.Groups, r.Blasts, r.Sources, r.Stale, r.SkeletonFiles)
	if got != want {
		t.Fatalf("RenderExplore over ExploreDetail's fields != Explore(%q, %d):\ngot:\n%s\nwant:\n%s", query, maxFiles, got, want)
	}

	if r.Sources == nil {
		t.Fatalf("ExploreDetail(%q, %d): Sources is nil, want a populated map keyed by group path", query, maxFiles)
	}
	for _, g := range r.Groups {
		if _, ok := r.Sources[g.Path]; !ok {
			t.Fatalf("ExploreDetail(%q, %d): Sources missing entry for group path %q — Sources must be keyed by ExploreFileGroup.Path", query, maxFiles, g.Path)
		}
	}
}

// TestExploreResultZeroMatchModeCoversEveryBranch pins D-02: every one of
// Explore()'s five early "no results" returns funnels through the SAME
// ExploreResult{Empty: true} representation rather than escaping as a
// pre-rendered string, and Explore()'s output for that input is
// byte-identical to exploreZeroResult(query, stale) for every branch that
// is reachable through the public API.
//
// The five branches, enumerated top to bottom in buildExploreResult
// (internal/query/detail.go, line numbers as of this commit): :360 (no
// gather candidates AND no named-symbol seeds), :424 (the full
// BFS/hierarchy/glue expansion surfaces no live node), :477 (H15's hard
// test/spec/icon/i18n exclusion empties fileScores entirely), :540 (H17's
// relevance gate empties fileOrder), :626 (groupMatchesByFile produces no
// file group).
//
// Branches 1-3 are driven through Explore()'s real public API (a real
// on-disk-indexed engine for branch 1, the local behavioral fixture with a
// temporarily inflated DefaultExploreBFSBounds.MinScore for branch 2, and
// a purpose-built isolated-test-file fixture for branch 3). Branches 4 and
// 5 are PROVEN UNREACHABLE given the current implementations of
// fileRelevanceGate/fiveTierFileSort (explore_gate.go) and
// groupMatchesByFile/clampMaxFiles (explore.go/validate.go): each of those
// functions is shown, by direct call in its corresponding subtest, to
// never turn a non-empty input into an empty output — so the non-empty
// fileScores branch 3 already requires can never produce an empty
// fileOrder or an empty groups downstream. This was reached by direct
// code-path tracing during this task's execution (recorded in
// 01-05-SUMMARY.md), not assumed without support, and those two subtests
// assert the invariant rather than driving Explore() through a state nothing
// can reach.
func TestExploreResultZeroMatchModeCoversEveryBranch(t *testing.T) {
	t.Run("no-candidates-no-seeds", func(t *testing.T) {
		dir := copyFixture(t)
		indexFixture(t, dir)
		e, closer, err := OpenAt(dir)
		if err != nil {
			t.Fatalf("OpenAt: unexpected error: %v", err)
		}
		defer closer.Close()

		const query = "zzzznonexistentqueryterm12345"
		r, err := e.ExploreDetail(query, 5)
		if err != nil {
			t.Fatalf("ExploreDetail(%q): unexpected error: %v", query, err)
		}
		if !r.Empty {
			t.Fatalf("ExploreDetail(%q): got Empty=false, want true (no gather candidates and no named-symbol seeds)", query)
		}

		got, err := e.Explore(query, 5)
		if err != nil {
			t.Fatalf("Explore(%q): unexpected error: %v", query, err)
		}
		want := exploreZeroResult(r.Query, r.Stale)
		if got != want {
			t.Fatalf("Explore(%q) = %q, want exploreZeroResult(...) = %q", query, got, want)
		}
	})

	t.Run("bfs-fully-pruned-no-final-nodes", func(t *testing.T) {
		e := behavioralEngine(t)
		orig := DefaultExploreBFSBounds
		DefaultExploreBFSBounds = ExpandBFSBounds{
			MinScore:       1e9, // prunes every real candidate's score below the BFS-root floor
			MaxNodes:       orig.MaxNodes,
			TraversalDepth: orig.TraversalDepth,
			SearchLimit:    orig.SearchLimit,
		}
		t.Cleanup(func() { DefaultExploreBFSBounds = orig })

		// "recovery_test" resolves real gather candidates (confirmed by
		// hand during this task's investigation: 4 candidates, 0 named
		// seeds) but zero named-symbol seeds, so with every candidate
		// pruned by the inflated MinScore, BFS/hierarchy/glue surface no
		// live node at all — finalNodeIDs is empty.
		const query = "recovery_test"
		r, err := e.ExploreDetail(query, 5)
		if err != nil {
			t.Fatalf("ExploreDetail(%q): unexpected error: %v", query, err)
		}
		if !r.Empty {
			t.Fatalf("ExploreDetail(%q): got Empty=false, want true (every gather candidate pruned below the inflated MinScore)", query)
		}

		got, err := e.Explore(query, 5)
		if err != nil {
			t.Fatalf("Explore(%q): unexpected error: %v", query, err)
		}
		want := exploreZeroResult(r.Query, r.Stale)
		if got != want {
			t.Fatalf("Explore(%q) = %q, want exploreZeroResult(...) = %q", query, got, want)
		}
	})

	t.Run("hard-test-exclusion-empties-file-scores", func(t *testing.T) {
		dir := t.TempDir()
		nodes := map[string]*schema.Node{
			"tf1": {Id: "tf1", Name: "IsolatedFunc", Kind: "function", FilePath: "pkg/handler_test.go", StartLine: 1, EndLine: 1, Signature: "func IsolatedFunc()"},
		}
		e := NewWithRoot(&zeroMetaFakeReader{&traverseFakeReader{nodes: nodes}}, dir)

		// IsolatedFunc's only definition lives in a _test.go file with no
		// edges to any other file. H15's hard exclusion drops it — it is
		// the ONLY file in fileScores, so the "query mentions test AND
		// >=2 non-low-value files remain" exemption cannot apply — emptying
		// fileScores entirely.
		const query = "IsolatedFunc"
		r, err := e.ExploreDetail(query, 5)
		if err != nil {
			t.Fatalf("ExploreDetail(%q): unexpected error: %v", query, err)
		}
		if !r.Empty {
			t.Fatalf("ExploreDetail(%q): got Empty=false, want true (H15 excludes the only surfaced file)", query)
		}

		got, err := e.Explore(query, 5)
		if err != nil {
			t.Fatalf("Explore(%q): unexpected error: %v", query, err)
		}
		want := exploreZeroResult(r.Query, r.Stale)
		if got != want {
			t.Fatalf("Explore(%q) = %q, want exploreZeroResult(...) = %q", query, got, want)
		}
	})

	t.Run("relevance-gate-never-empties-a-non-empty-input", func(t *testing.T) {
		// Branch 4 (detail.go:540, len(fileOrder)==0) is reached only when
		// fileRelevanceGate's return value is empty. fileRelevanceGate's
		// own documented "never prunes below 2 files" guard
		// (explore_gate.go) means it returns either the pre-gate paths
		// unchanged, or a >=2-file gated subset — never empty for a
		// non-empty input. fiveTierFileSort is a stable in-place sort: it
		// returns exactly len(input) elements, never fewer. Composed, a
		// non-empty candidatePaths — which branch 3 already requires,
		// since candidatePaths is built directly from the same fileScores
		// branch 3 checks — can never reach fileOrder==0. This subtest
		// drives both functions directly with an adversarial input
		// designed to fail every one of the gate's 5 clauses, and asserts
		// neither function ever returns fewer elements than it was given.
		paths := []string{"a.go", "b.go"}
		fileScores := map[string]float64{"a.go": 0, "b.go": 0}     // fails clause 3 (entry/named)
		fileGraphScore := map[string]float64{"a.go": 0, "b.go": 0} // maxGraph==0: gate short-circuits to "unchanged"
		centralFiles := map[string]bool{}                          // fails clause 2
		rescuedFiles := map[string]bool{}                          // fails clause 4
		fileTermHits := map[string]int{"a.go": 0, "b.go": 0}       // fails clause 5

		gated := fileRelevanceGate(paths, fileScores, fileGraphScore, centralFiles, rescuedFiles, fileTermHits)
		if len(gated) == 0 {
			t.Fatalf("fileRelevanceGate(%v, ...): got an empty result for a non-empty input — this WOULD make branch 4 reachable; investigate before trusting it dead", paths)
		}

		fileNodeCounts := map[string]int{"a.go": 1, "b.go": 1}
		order := fiveTierFileSort(gated, fileScores, fileGraphScore, centralFiles, fileTermHits, fileNodeCounts)
		if len(order) != len(gated) {
			t.Fatalf("fiveTierFileSort(%v, ...): returned %d elements, want exactly %d (a sort must never drop elements)", gated, len(order), len(gated))
		}
	})

	t.Run("group-assembly-never-empties-a-non-empty-ranked-set", func(t *testing.T) {
		// Branch 5 (detail.go:626, len(groups)==0) is reached only when
		// groupMatchesByFile(ranked, maxFiles) returns no groups despite a
		// non-empty ranked slice. groupMatchesByFile only skips a NEW file
		// once len(groups) >= maxFiles, and maxFiles is always >= 1 by the
		// time it reaches groupMatchesByFile (clampMaxFiles and
		// clampExploreBudget both floor it at 1) — so the FIRST ranked
		// node with a non-empty FilePath always seeds a group. This
		// subtest proves that directly: a single ranked node with
		// maxFiles=1 still produces exactly one group.
		n := &schema.Node{Id: "n1", Name: "N", Kind: "function", FilePath: "only.go"}
		groups, symbolCount := groupMatchesByFile([]rankedNode{{node: n}}, 1)
		if len(groups) == 0 {
			t.Fatal("groupMatchesByFile(single ranked node, maxFiles=1): got 0 groups, want 1 — this WOULD make branch 5 reachable; investigate before trusting it dead")
		}
		if symbolCount != 1 {
			t.Fatalf("groupMatchesByFile: symbolCount = %d, want 1", symbolCount)
		}
	})
}

// TestExploreResultMaxFilesBoundary asserts all three maxFiles boundary
// rows: maxFiles equal to the matched-file count includes every group;
// maxFiles one below it drops exactly one; and maxFiles == 0 produces the
// same result the pre-extraction code produces for the same input,
// computed independently in this test (via countIndexedFiles/
// getExploreOutputBudget/clampExploreBudget — the exact H21 formula
// buildExploreResult itself calls) rather than asserted as a hardcoded
// constant, so the "behaves exactly as it does today" claim has a failing
// input.
func TestExploreResultMaxFilesBoundary(t *testing.T) {
	e := behavioralEngine(t)
	const query = "account balance"

	// A generous maxFiles that cannot itself truncate establishes the full
	// untruncated group count.
	full, err := e.ExploreDetail(query, 20)
	if err != nil {
		t.Fatalf("ExploreDetail(%q, 20): unexpected error: %v", query, err)
	}
	if full.Empty {
		t.Fatalf("ExploreDetail(%q, 20): got Empty=true, want a populated multi-file result", query)
	}
	n := len(full.Groups)
	if n < 2 {
		t.Fatalf("ExploreDetail(%q, 20): got %d groups, want >= 2 so the boundary subtests below are meaningful", query, n)
	}

	t.Run("exactly-at-match-count", func(t *testing.T) {
		r, err := e.ExploreDetail(query, n)
		if err != nil {
			t.Fatalf("ExploreDetail(%q, %d): unexpected error: %v", query, n, err)
		}
		if len(r.Groups) != n {
			t.Fatalf("ExploreDetail(%q, %d): got %d groups, want exactly %d (maxFiles == match count truncates nothing)", query, n, len(r.Groups), n)
		}
	})

	t.Run("one-below", func(t *testing.T) {
		r, err := e.ExploreDetail(query, n-1)
		if err != nil {
			t.Fatalf("ExploreDetail(%q, %d): unexpected error: %v", query, n-1, err)
		}
		if len(r.Groups) != n-1 {
			t.Fatalf("ExploreDetail(%q, %d): got %d groups, want exactly %d (one below the match count drops exactly one group)", query, n-1, len(r.Groups), n-1)
		}
	})

	t.Run("zero-is-unchanged", func(t *testing.T) {
		fileCount, err := e.countIndexedFiles()
		if err != nil {
			t.Fatalf("countIndexedFiles: unexpected error: %v", err)
		}
		wantMaxFiles := clampExploreBudget(getExploreOutputBudget(fileCount))

		zero, err := e.ExploreDetail(query, 0)
		if err != nil {
			t.Fatalf("ExploreDetail(%q, 0): unexpected error: %v", query, err)
		}
		explicit, err := e.ExploreDetail(query, wantMaxFiles)
		if err != nil {
			t.Fatalf("ExploreDetail(%q, %d): unexpected error: %v", query, wantMaxFiles, err)
		}
		if len(zero.Groups) != len(explicit.Groups) {
			t.Fatalf("ExploreDetail(%q, 0): got %d groups, want %d (the independently-computed H21 adaptive budget for this index)", query, len(zero.Groups), len(explicit.Groups))
		}
		gotRender := RenderExplore(zero.Query, len(zero.Groups), zero.SymbolCount, zero.Groups, zero.Blasts, zero.Sources, zero.Stale, zero.SkeletonFiles)
		wantRender := RenderExplore(explicit.Query, len(explicit.Groups), explicit.SymbolCount, explicit.Groups, explicit.Blasts, explicit.Sources, explicit.Stale, explicit.SkeletonFiles)
		if gotRender != wantRender {
			t.Fatalf("ExploreDetail(%q, 0) rendered output differs from the independently-computed explicit-budget equivalent:\ngot:\n%s\nwant:\n%s", query, gotRender, wantRender)
		}
	})
}
