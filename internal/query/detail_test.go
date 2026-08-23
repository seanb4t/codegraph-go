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
