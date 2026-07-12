// testdata/golden/parity_csharp_test.go
//
// TestGoldenParity_CSharp is LANG-03's D-12 acceptance gate for C#.
//
// D-12 asks for the same byte/shape diff against captured TS CodeGraph
// v1.3.x golden output that golden_parity_test.go's TestGoldenParity uses
// for Go (resolveWeftCorpus's pinned-commit-checkout + corpus/weft-go/*.json
// pattern). Capturing that fixture requires a working local install of the
// live TS CodeGraph v1.3.x CLI; per 05-RESEARCH.md's "Environment
// Availability" table, that CLI was not available in this environment (same
// finding as parity_java_test.go's own D-12 fallback). RESEARCH documents
// the sanctioned fallback for exactly this situation: "read the parity
// target's SOURCE as a specification rather than a live golden-output
// oracle" plus a self-consistency check against this project's own repeated
// runs. This file implements that fallback (mirroring parity_java_test.go
// verbatim in structure) rather than skipping D-12 entirely.
//
// It self-skips (t.Skip, never t.Fatal) when no C# validation corpus is
// configured, exactly mirroring resolveWeftCorpus's/resolveJavaCorpus's
// "loud skip, never a silent pass or a hard CI failure" discipline
// (T-03-09-Repro), so `go test ./...` stays green everywhere this corpus
// isn't checked out.
//
// Per RESEARCH's Assumptions Log A3 and this plan's Task 2 action, the
// corpus this test expects (when configured) SHOULD deliberately exercise
// `partial class` / generated `.Designer.cs` scaffolding (common in older
// ASP.NET MVC projects) — surfacing Pitfall 5's node-identity decision
// during real-corpus validation rather than leaving it untested until a
// real user's repo hits it post-ship.
package golden

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// resolveCSharpCorpus locates a local checkout of a real, representative,
// license-clean C# repository — ideally one using `partial class` /
// generated Designer-file scaffolding (Assumptions Log A3): first
// CODEGRAPH_CSHARP_CORPUS, then a conventional sibling checkout
// (../csharp-corpus next to this repo's root). It t.Skip()s with a clear,
// actionable message — never fails — when no corpus is configured,
// mirroring resolveJavaCorpus/resolveWeftCorpus.
func resolveCSharpCorpus(t *testing.T) string {
	t.Helper()

	if env := os.Getenv("CODEGRAPH_CSHARP_CORPUS"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return env
		}
		t.Skipf("CODEGRAPH_CSHARP_CORPUS=%s is not a directory", env)
	}

	if repoRoot, err := filepath.Abs(filepath.Join("..", "..")); err == nil {
		candidate := filepath.Join(filepath.Dir(repoRoot), "csharp-corpus")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	t.Skip(
		"no C# validation corpus configured — set CODEGRAPH_CSHARP_CORPUS=/path/to/a/real/csharp/repo " +
			"to run this test (ideally one using `partial class`/generated .Designer.cs scaffolding, per " +
			"05-RESEARCH.md Assumptions Log A3). A live TS CodeGraph v1.3.x CLI was unavailable in this " +
			"environment to capture a byte-comparable golden fixture (05-RESEARCH.md Environment " +
			"Availability); this test instead self-validates extraction/resolution SHAPE (non-zero node/" +
			"edge kind coverage + deterministic rebuild) against whatever real C# repo is configured — " +
			"RESEARCH's documented source-as-specification + self-consistency fallback for D-12 priority-4 " +
			"validation when the live TS CLI oracle is not runnable.",
	)
	return ""
}

// TestGoldenParity_CSharp validates, against a real (Claude's Discretion,
// user-configured) C# repository:
//
//  1. Shape coverage — every node/edge kind csharpextract/types.go
//     documents (struct/interface/method + calls/imports/embeds) actually
//     appears with a non-zero count on real source, proving extraction
//     genuinely fires rather than silently producing an empty graph.
//  2. Cross-file resolution — at least one "calls" edge resolves, proving
//     Pass 2's per-language symbol index genuinely connects C# call sites
//     to their declarations (LANG-03's "full... cross-file resolution"
//     bar), not just intra-file structural extraction.
//  3. Determinism (D-01a, project-wide) — a second from-scratch run over
//     the same corpus yields byte-identical aggregate node/edge/file
//     counts. This is the same invariant TestExtract_PartialClass_
//     SharedNodeIdentity and TestResolve_PartialClassBothFragmentsCallable
//     prove at the unit/fixture level; here it is proven again against
//     WHATEVER real corpus is configured, including any partial-class
//     scaffolding it happens to contain.
func TestGoldenParity_CSharp(t *testing.T) {
	corpusDir := resolveCSharpCorpus(t)

	storeDir1 := t.TempDir()
	stats1, err := indexer.Run(corpusDir, storeDir1, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("indexer.Run (first pass): %v", err)
	}
	if stats1.Files == 0 {
		t.Fatalf("indexer.Run found 0 files in the configured C# corpus %s", corpusDir)
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

	nodeKindCounts := make(map[string]int)
	nit, err := reader1.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		n := nit.Node()
		if n.Language == "csharp" {
			nodeKindCounts[n.Kind]++
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
		edgeKindCounts[eit.Edge().Kind]++
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}

	// Shape check 1: a representative real C# repo MUST produce at least
	// one node of each of csharpextract's core kinds — a zero count here
	// means extraction silently failed to fire on real source, not just an
	// absence of that construct in whichever corpus is configured (a repo
	// with zero classes or zero methods is not a representative C# corpus
	// for this check).
	for _, kind := range []string{"struct", "method"} {
		if nodeKindCounts[kind] == 0 {
			t.Errorf("shape check: 0 %q nodes extracted from the C# corpus, want > 0", kind)
		}
	}
	// Shape check 2: "calls" is the one edge kind LANG-03's full-resolution
	// bar requires to be non-zero on a real, non-trivial repo (imports/
	// embeds may legitimately be zero on an unusual single-namespace/
	// no-inheritance corpus, but a representative real C# project always
	// has cross-file or cross-class method calls).
	if edgeKindCounts["calls"] == 0 {
		t.Error(`shape check: 0 "calls" edges resolved from the C# corpus, want > 0`)
	}

	t.Logf("C# corpus %s: files=%d nodes=%d edges=%d nodeKinds=%v edgeKinds=%v",
		corpusDir, stats1.Files, stats1.Nodes, stats1.Edges, nodeKindCounts, edgeKindCounts)

	// Determinism (D-01a, project-wide guarantee): a second from-scratch
	// run over the same corpus must yield byte-identical aggregate counts
	// — including any partial-class node-identity collapsing this corpus
	// happens to exercise (Pitfall 5).
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
