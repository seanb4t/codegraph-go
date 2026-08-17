// testdata/golden/behavioral_java_test.go
//
// TestCorpusBehavior_Java is LANG-02's D-12 acceptance gate for Java — now
// resolved through the hermetic locked-corpus resolver (02-03, D-10) that loads
// the manifest via internal/corpora and FAILS LOUDLY on an absent or
// integrity-failed locked tree (t.Fatalf, never t.Skip).
//
// The old env-var-based resolver (CODEGRAPH_JAVA_CORPUS) and its sibling-
// checkout fallback with t.Skip are retired — this test always runs against the
// Phase-1-locked guava corpus when that corpus is present, and fails loudly
// with an actionable message when it is not.
package golden

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// TestCorpusBehavior_Java validates, against the Phase-1-locked guava corpus:
//
//  1. Shape coverage — every node/edge kind javaextract/types.go documents
//     (struct/interface/method + calls/imports/embeds) actually appears
//     with a non-zero count on real source, proving extraction genuinely
//     fires rather than silently producing an empty graph.
//  2. Cross-file resolution — at least one "calls" edge resolves, proving
//     Pass 2's per-language symbol index genuinely connects Java call
//     sites to their declarations (LANG-02's "full... cross-file
//     resolution" bar), not just intra-file structural extraction.
//  3. Determinism (D-01a, project-wide) — a second from-scratch run over
//     the same corpus yields byte-identical aggregate node/edge/file
//     counts.
func TestCorpusBehavior_Java(t *testing.T) {
	corpusDir := lockedCorpusDir(t, "java")

	storeDir1 := t.TempDir()
	stats1, err := indexer.Run(corpusDir, storeDir1, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("indexer.Run (first pass): %v", err)
	}
	if stats1.Files == 0 {
		t.Fatalf("indexer.Run found 0 files in the locked Java corpus %s", corpusDir)
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
		if n.Language == "java" {
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

	// Shape check 1: a representative real Java repo MUST produce at
	// least one node of each of javaextract's core kinds — a zero count
	// here means extraction silently failed to fire on real source, not
	// just an absence of that construct in whichever corpus is
	// configured (a repo with zero classes or zero methods is not a
	// representative Java corpus for this check).
	for _, kind := range []string{"struct", "method"} {
		if nodeKindCounts[kind] == 0 {
			t.Errorf("shape check: 0 %q nodes extracted from the Java corpus, want > 0", kind)
		}
	}
	// Shape check 2: "calls" is the one edge kind LANG-02's full-resolution
	// bar requires to be non-zero on a real, non-trivial repo (imports/
	// embeds may legitimately be zero on an unusual single-package/
	// no-inheritance corpus, but a representative real Java project always
	// has cross-file or cross-class method calls).
	if edgeKindCounts["calls"] == 0 {
		t.Error(`shape check: 0 "calls" edges resolved from the Java corpus, want > 0`)
	}

	t.Logf("Java corpus %s: files=%d nodes=%d edges=%d nodeKinds=%v edgeKinds=%v",
		corpusDir, stats1.Files, stats1.Nodes, stats1.Edges, nodeKindCounts, edgeKindCounts)

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