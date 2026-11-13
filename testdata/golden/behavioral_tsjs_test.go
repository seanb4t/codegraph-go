// testdata/golden/behavioral_tsjs_test.go
//
// TestCorpusBehavior_TSJS is LANG-05's D-12 acceptance gate for TypeScript,
// TSX, and JavaScript — now resolved through the hermetic locked-corpus
// resolver (02-03, D-10) that loads the manifest via internal/corpora and
// FAILS LOUDLY on an absent or integrity-failed locked tree (t.Fatalf, never
// t.Skip). tsjs resolves to the hugo corpus through the language slug map,
// because hugo's 25 JS files supply the tsjs leg despite its manifest
// language being "go".
//
// The old env-var-based resolver (CODEGRAPH_TSJS_CORPUS) and its sibling-
// checkout fallback with t.Skip are retired — this test always runs against
// the Phase-1-locked hugo corpus when that corpus is present, and fails
// loudly with an actionable message when it is not.
package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// TestCorpusBehavior_TSJS validates, against the Phase-1-locked hugo corpus
// (whose 25 JS files supply the tsjs leg):
//
//  1. Shape coverage — every node/edge kind tsextract/types.go documents
//     (struct/function + calls/imports/embeds) actually appears with a
//     non-zero count on real source, across the javascript files within
//     the hugo tree, proving extraction genuinely fires rather than
//     silently producing an empty graph.
//  2. Call resolution — at least one "calls" edge joins two tsjs nodes. Hugo
//     is mostly Go, so an edge counts only when both endpoints are tsjs. The
//     hugo JavaScript has no cross-file call, so cross-file resolution is
//     asserted by TestSemanticTypeScriptCorpusEdges on a TypeScript corpus.
//  3. Determinism (D-01a, project-wide) — a second from-scratch run over
//     the same corpus yields byte-identical aggregate node/edge/file
//     counts.
func TestCorpusBehavior_TSJS(t *testing.T) {
	corpusDir := lockedCorpusDir(t, "tsjs")

	storeDir1 := t.TempDir()
	stats1, err := indexer.Run(corpusDir, storeDir1, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("indexer.Run (first pass): %v", err)
	}
	if stats1.Files == 0 {
		t.Fatalf("indexer.Run found 0 files in the locked tsjs corpus %s", corpusDir)
	}

	store1, err := graphstore.Open(storeDir1)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store1.Close()
	reader1, err := store1.Snapshot()
	if err != nil {
		t.Fatalf("store.Snapshot: %v", err)
	}
	defer reader1.Close()

	tsjsLanguages := map[string]bool{"typescript": true, "tsx": true, "javascript": true}
	languagesSeen := make(map[string]int)
	nodeKindCounts := make(map[string]int)
	tsjsNodes := make(map[string]bool)
	nit, err := reader1.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		n := nit.Node()
		if tsjsLanguages[n.Language] {
			nodeKindCounts[n.Kind]++
			languagesSeen[n.Language]++
			tsjsNodes[n.Id] = true
		}
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}

	edgeKindCounts := make(map[string]int)
	eit, err := reader1.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		if tsjsNodes[e.Source] && tsjsNodes[e.Target] {
			edgeKindCounts[e.Kind]++
		}
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}

	if len(languagesSeen) == 0 {
		t.Fatalf("shape check: 0 typescript/tsx/javascript nodes extracted from the configured corpus %s", corpusDir)
	}

	// Shape check 1: a representative real repo MUST produce at
	// least one node of each of tsextract's core kinds — a zero count here
	// means extraction silently failed to fire on real source.
	for _, kind := range []string{"struct", "function"} {
		if nodeKindCounts[kind] == 0 {
			t.Errorf("shape check: 0 %q nodes extracted from the tsjs corpus, want > 0", kind)
		}
	}
	// Shape check 2: a resolved call must join two tsjs nodes. Edges of the
	// Go majority of this corpus do not count.
	if edgeKindCounts["calls"] == 0 {
		t.Error(`shape check: 0 "calls" edges between tsjs nodes, want > 0`)
	}

	t.Logf("tsjs corpus %s: files=%d nodes=%d edges=%d languages=%v nodeKinds=%v edgeKinds=%v",
		corpusDir, stats1.Files, stats1.Nodes, stats1.Edges, languagesSeen, nodeKindCounts, edgeKindCounts)

	// Determinism (D-01a, project-wide guarantee): a second from-scratch
	// run over the same corpus must yield byte-identical aggregate counts.
	storeDir2 := t.TempDir()
	stats2, err := indexer.Run(corpusDir, storeDir2, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("indexer.Run (second pass): %v", err)
	}
	if stats1.Files != stats2.Files || stats1.Nodes != stats2.Nodes || stats1.Edges != stats2.Edges {
		t.Errorf("non-deterministic rebuild: pass1={files:%d nodes:%d edges:%d} pass2={files:%d nodes:%d edges:%d}",
			stats1.Files, stats1.Nodes, stats1.Edges, stats2.Files, stats2.Nodes, stats2.Edges)
	}
}

// typeScriptCorpusRepo is the manifest entry that supplies TypeScript source.
const typeScriptCorpusRepo = "pmndrs/zustand"

// reviewedTypeScriptCalls are every cross-file value call under src/ that the
// corpus source establishes and CodeGraph resolves. They are read from the
// source, never generated from CodeGraph output.
var reviewedTypeScriptCalls = []string{
	"src/react/shallow.ts:useShallow -> src/vanilla/shallow.ts:shallow",
}

// knownMissingTypeScriptCalls are the rest of that set. Each caller is an
// unexported arrow constant, a form #239 plans.
var knownMissingTypeScriptCalls = []string{
	"src/react.ts:createImpl -> src/vanilla.ts:createStore",
	"src/traditional.ts:createWithEqualityFnImpl -> src/vanilla.ts:createStore",
}

// corpusSample is a reviewed part of one pinned source collection. Reviewed
// and KnownMissing are the labeled denominator. Every other edge is unlabeled:
// the test reports that count and draws no accuracy claim from it.
type corpusSample struct {
	Repo         string
	Language     string
	Reviewed     []string
	KnownMissing map[string]string
}

var corpusSamples = []corpusSample{
	{
		Repo:     "gohugoio/hugo",
		Language: "go",
		Reviewed: []string{
			"common/paths/path.go:ToSlashTrimLeading -> common/paths/path.go:TrimLeading",
			"resources/resource/resources.go:Get -> common/paths/path.go:AddLeadingSlash",
			"resources/resource/resources.go:GetMatch -> common/paths/path.go:AddLeadingSlash",
			"resources/resource/resources.go:GetMatch -> hugofs/hglob/glob.go:GetGlob",
			"resources/resource/resources.go:Match -> common/paths/path.go:AddLeadingSlash",
		},
	},
	{
		Repo:     "psf/requests",
		Language: "python",
		Reviewed: []string{
			"src/requests/api.py:get -> src/requests/api.py:request",
			"src/requests/sessions.py:get -> src/requests/sessions.py:request",
			"src/requests/sessions.py:prepare_request -> src/requests/cookies.py:merge_cookies",
			"src/requests/sessions.py:prepare_request -> src/requests/sessions.py:merge_setting",
			"src/requests/sessions.py:rebuild_auth -> src/requests/sessions.py:should_strip_auth",
			"src/requests/sessions.py:resolve_redirects -> src/requests/sessions.py:get_redirect_target",
		},
	},
	{
		Repo:     "google/guava",
		Language: "java",
		Reviewed: []string{
			"guava/src/com/google/common/base/Joiner.java:appendTo -> guava/src/com/google/common/base/Joiner.java:iterable",
			"guava/src/com/google/common/base/Splitter.java:computeNext -> guava/src/com/google/common/base/AbstractIterator.java:endOfData",
			"guava/src/com/google/common/base/Splitter.java:computeNext -> guava/src/com/google/common/base/Splitter.java:separatorEnd",
			"guava/src/com/google/common/base/Splitter.java:fixedLength -> guava/src/com/google/common/base/Preconditions.java:checkArgument",
		},
		KnownMissing: map[string]string{
			"guava/src/com/google/common/base/Joiner.java:Joiner -> guava/src/com/google/common/base/Preconditions.java:checkNotNull":   "#263",
			"guava/src/com/google/common/base/Splitter.java:limit -> guava/src/com/google/common/base/Preconditions.java:checkArgument": "#263",
		},
	},
}

// corpusCalls indexes dir and returns its calls edges between language nodes
// as "file:name -> file:name".
func corpusCalls(t *testing.T, dir, language string) map[string]bool {
	t.Helper()
	storeDir := t.TempDir()
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()
	reader, err := store.Snapshot()
	if err != nil {
		t.Fatalf("store.Snapshot: %v", err)
	}
	defer reader.Close()

	type node struct{ file, name string }
	nodes := make(map[string]node)
	nit, err := reader.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		if n := nit.Node(); n.Language == language {
			nodes[n.Id] = node{n.FilePath, n.Name}
		}
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}

	calls := make(map[string]bool)
	eit, err := reader.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		src, srcOK := nodes[e.Source]
		dst, dstOK := nodes[e.Target]
		if e.Kind == "calls" && srcOK && dstOK {
			calls[fmt.Sprintf("%s:%s -> %s:%s", src.file, src.name, dst.file, dst.name)] = true
		}
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	return calls
}

// TestSemanticTypeScriptCorpusEdges fails when any reviewed edge is absent,
// however many other edges the graph holds. Unlike a total, it cannot pass on
// edges of another language or on edges inside one file.
func TestSemanticTypeScriptCorpusEdges(t *testing.T) {
	calls := make(map[string]bool)
	for edge := range corpusCalls(t, manifestCorpusDir(t, typeScriptCorpusRepo), "typescript") {
		from, to, _ := strings.Cut(edge, " -> ")
		fromFile, _, _ := strings.Cut(from, ":")
		toFile, _, _ := strings.Cut(to, ":")
		if strings.HasPrefix(fromFile, "src/") && fromFile != toFile {
			calls[edge] = true
		}
	}
	if len(calls) == 0 {
		t.Fatalf("%s has no cross-file TypeScript calls edge under src/", typeScriptCorpusRepo)
	}
	for _, want := range reviewedTypeScriptCalls {
		if !calls[want] {
			t.Errorf("missing reviewed edge %s", want)
		}
		delete(calls, want)
	}
	for _, gap := range knownMissingTypeScriptCalls {
		if calls[gap] {
			t.Errorf("%s now resolves: move it to reviewedTypeScriptCalls", gap)
		}
		delete(calls, gap)
	}
	for extra := range calls {
		t.Errorf("edge %s is not in the reviewed set", extra)
	}
}

// TestSemanticCorpusSamples checks the reviewed edges of each pinned sample.
// A sample proves only its labeled edges, so it reports the unlabeled count
// and never a precision over the whole repository.
func TestSemanticCorpusSamples(t *testing.T) {
	for _, s := range corpusSamples {
		t.Run(s.Language, func(t *testing.T) {
			calls := corpusCalls(t, manifestCorpusDir(t, s.Repo), s.Language)
			if len(calls) == 0 || len(s.Reviewed) == 0 {
				t.Fatalf("%s: %d edges, %d reviewed: a sample needs both", s.Repo, len(calls), len(s.Reviewed))
			}
			for _, want := range s.Reviewed {
				if !calls[want] {
					t.Errorf("missing reviewed edge %s", want)
				}
			}
			for gap, issue := range s.KnownMissing {
				if calls[gap] {
					t.Errorf("%s now resolves: remove the %s exception", gap, issue)
				}
			}
			t.Logf("%s %s: labeled=%d known-gaps=%d unlabeled=%d", s.Repo, s.Language,
				len(s.Reviewed)+len(s.KnownMissing), len(s.KnownMissing), len(calls)-len(s.Reviewed))
		})
	}
}

// manifestCorpusDir resolves repo's manifest entry, locked or not, and fails
// loudly when the tree is absent.
func manifestCorpusDir(t *testing.T, repo string) string {
	t.Helper()
	m, err := corpora.Load(filepath.Join("..", "..", "corpora", "manifest.json"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	root, err := corpora.CorpusRoot()
	if err != nil {
		t.Fatalf("CorpusRoot: %v", err)
	}
	for _, e := range m.Corpora {
		if e.Repo != repo {
			continue
		}
		dir := e.Dir(root)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("corpus %s is absent at %s, run task corpora:fetch: %v", repo, dir, err)
		}
		return dir
	}
	t.Fatalf("corpus %s has no manifest entry", repo)
	return ""
}
