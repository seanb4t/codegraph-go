package resolve

import (
	"slices"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Graph is the resolved graph of one resolution, before edge collapse.
type Graph struct {
	// Nodes are every node pass 1 extracted, then the overload set nodes.
	Nodes []*schema.Node
	// PackageNodes are the synthetic package nodes minted for
	// intra-module imports (RQ-1), one per id.
	PackageNodes []*schema.Node
	// Edges are every intra-file edge pass 1 extracted, plus every
	// resolved and synthesized edge.
	Edges []*schema.Edge
	// Files has one record per result with a path.
	Files []*schema.File
	// Unresolved counts the references that did not resolve (never
	// silently dropped, D-06a; surfaced later through --verbose).
	Unresolved int
}

// pendingCall is a call that failed to resolve in its file, paired with
// the File record whose EdgeCount a retry success increments.
type pendingCall struct {
	ref  goextract.UnresolvedRef
	file *schema.File
	// wait is the name file waits on if the retry fails too, or nil.
	wait *schema.UnresolvedName
}

// Link assembles the outputs of Types and Calls, one pair per result in
// the same order, into the graph, and adds what needs every file: the
// structural implements and overrides edges, the conformance retry of
// the calls that did not resolve, and the overload set edges.
//
// It assembles the edges in the order design D6 states, because
// CollapseEdges keeps the first of two duplicates that tie on file, line
// and column: the type-level edges of every file in result order, then
// the calls without a deferred type in result order, then the deferred
// calls of every file, with the files in path order. The pending calls of
// the retry follow the same rule. So the graph does not depend on the
// order of the results. Link does not open the store or start a
// goroutine.
func Link(view symbols.View, h symbols.Hierarchy, types []TypeOutput, calls []CallOutput) Graph {
	if len(types) != len(calls) {
		panic("resolve: Link needs one CallOutput per TypeOutput")
	}
	var g Graph
	packageNodeIDs := make(map[string]struct{})
	for _, t := range types {
		g.Nodes = append(g.Nodes, t.nodes...)
		for _, pn := range t.packageNodes {
			if _, seen := packageNodeIDs[pn.Id]; !seen {
				packageNodeIDs[pn.Id] = struct{}{}
				g.PackageNodes = append(g.PackageNodes, pn)
			}
		}
		g.Edges = append(g.Edges, t.edges...)
		g.Unresolved += t.unresolved
	}

	var pending []pendingCall
	for i, c := range calls {
		g.Edges = append(g.Edges, c.edges...)
		for _, p := range c.pending {
			pending = append(pending, pendingCall{ref: p.ref, file: types[i].file, wait: p.wait})
		}
	}
	byPath := make([]int, len(calls))
	for i := range byPath {
		byPath[i] = i
	}
	sort.SliceStable(byPath, func(a, b int) bool { return calls[byPath[a]].relPath < calls[byPath[b]].relPath })
	for _, i := range byPath {
		g.Edges = append(g.Edges, calls[i].deferredEdges...)
		for _, p := range calls[i].deferredPending {
			pending = append(pending, pendingCall{ref: p.ref, file: types[i].file, wait: p.wait})
		}
	}

	g.Nodes = append(g.Nodes, h.SetNodes()...)

	for i, t := range types {
		if t.file == nil {
			continue
		}
		t.file.EdgeCount += int64(calls[i].resolved)
		g.Files = append(g.Files, t.file)
	}

	withStoredEdges := func(es []*schema.Edge) []*schema.Edge {
		stored := view.StoredEdges()
		if len(stored) == 0 {
			return es
		}
		return append(slices.Clip(es), stored...)
	}

	g.Edges = append(g.Edges, synthesizeGoImplements(view, withStoredEdges(g.Edges))...)

	methodKey := make(map[string]string)
	for n := range view.Nodes() {
		if k := goextract.SignatureKeyOfNode(n); k != "" {
			methodKey[n.Id] = k
		}
	}
	g.Edges = append(g.Edges, synthesizeOverrides(view, withStoredEdges(g.Edges), methodKey)...)

	g.Edges = append(g.Edges, retryConformanceCalls(pending, withStoredEdges(g.Edges), view, &g.Unresolved)...)

	// The overrides, implements and conformance synthesis above read a
	// contains edge as type to member in one hop, so the set edges are
	// made only now, after them.
	g.Edges = rewireSetEdges(view, h, g.Edges)

	for _, f := range g.Files {
		f.UnresolvedNames = sortedUniqueNames(f.UnresolvedNames)
	}
	return g
}
