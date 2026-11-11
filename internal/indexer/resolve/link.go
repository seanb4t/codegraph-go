package resolve

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Graph is the resolved graph of one resolution.
type Graph struct {
	// Nodes are every node pass 1 extracted, then the overload set nodes.
	Nodes []*schema.Node
	// PackageNodes are the synthetic package nodes minted for
	// intra-module imports (RQ-1), one per id.
	PackageNodes []*schema.Node
	// Edges are every intra-file edge pass 1 extracted, plus every
	// resolved and synthesized edge, collapsed to one edge per triple.
	Edges []*schema.Edge
	// Files has one record per result with a path.
	Files []*schema.File
	// Unresolved counts the references that did not resolve (never
	// silently dropped, D-06a; surfaced later through --verbose).
	Unresolved int

	// OwnerPath returns the owner path of an edge of the graph: the path
	// under which the store files the edge in its file index. Link sets
	// it to the function that chose the unit of each edge, so the unit and
	// the file-index entry of an edge have one source.
	OwnerPath func(e *schema.Edge) string
}

// Link assembles the outputs of Types and Calls, one pair per result in
// the same order, into the graph, and adds what needs every file: the
// structural implements and overrides edges, the conformance retry of
// the calls that did not resolve, and the overload set edges.
//
// It is a loop over Global.Assemble, the per-file assembly that a full
// index runs in its last pass, so the edge order has one implementation.
// After the loop come the whole-graph edges of each unit that has no
// result among types, which only a sync has, and the no-file unit. The
// edges of the returned graph are collapsed in each unit, and a triple
// has all its candidates in one unit, so no triple repeats. The graph does
// not depend on the order of the results. Link does not open the store or
// start a goroutine.
//
// An error wraps goextract.ErrInternal: an edge in a unit that cannot hold
// it, which is a defect of the resolver.
func Link(view symbols.View, h symbols.Hierarchy, types []TypeOutput, calls []CallOutput) (Graph, error) {
	if len(types) != len(calls) {
		panic("resolve: Link needs one CallOutput per TypeOutput")
	}
	var structural []*schema.Edge
	for _, t := range types {
		structural = append(structural, StructuralEdges(t.edges)...)
	}
	global := NewGlobal(view, h, structural)

	g := Graph{OwnerPath: global.OwnerPath}
	packageNodeIDs := make(map[string]struct{})
	hasResult := make(map[string]bool, len(types))
	noFile := make([]NoFileEdges, 0, len(types))
	for i := range types {
		u, err := global.Assemble(types[i], calls[i])
		if err != nil {
			return Graph{}, err
		}
		g.Nodes = append(g.Nodes, u.Nodes...)
		for _, pn := range u.PackageNodes {
			if _, seen := packageNodeIDs[pn.Id]; !seen {
				packageNodeIDs[pn.Id] = struct{}{}
				g.PackageNodes = append(g.PackageNodes, pn)
			}
		}
		g.Edges = append(g.Edges, u.Edges...)
		g.Unresolved += u.Unresolved
		if u.File != nil {
			g.Files = append(g.Files, u.File)
			hasResult[u.File.Path] = true
		}
		noFile = append(noFile, u.NoFile)
	}
	g.Edges = append(g.Edges, global.Rest(func(path string) bool { return hasResult[path] })...)
	g.Edges = append(g.Edges, global.CollapseNoFile(noFile)...)
	g.Nodes = append(g.Nodes, h.SetNodes()...)
	return g, nil
}
