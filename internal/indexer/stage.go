package indexer

import (
	"cmp"
	"slices"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Sink receives the records that stage writes. graphstore.Writer is one.
type Sink interface {
	PutNode(n *schema.Node) error
	PutFile(f *schema.File) error
	PutEdge(e *schema.Edge, ownerPath string) error
}

// stage puts the nodes and package nodes of g in id order, then its
// files in path order, then its edges in (source, kind, target) order.
// g.Edges are the edges to write, already collapsed. ownerPath names the
// file that owns an edge in the x/ file index. stage sorts copies, so g
// keeps its order. Two files with one path keep the order sort.Slice gives
// them, and the store keeps the last. A node id that several files emit
// reaches stage as one merged node (symbols.Builder), so every copy that
// stage puts has the same bytes.
func stage(sink Sink, g resolve.Graph, ownerPath func(e *schema.Edge) string) error {
	nodes := make([]*schema.Node, 0, len(g.Nodes)+len(g.PackageNodes))
	nodes = append(nodes, g.Nodes...)
	nodes = append(nodes, g.PackageNodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Id < nodes[j].Id })
	files := slices.Clone(g.Files)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	edges := g.Edges
	byTriple := func(a, b *schema.Edge) int {
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Target, b.Target))
	}
	if !slices.IsSortedFunc(edges, byTriple) {
		edges = slices.Clone(edges)
		slices.SortStableFunc(edges, byTriple)
	}
	for _, n := range nodes {
		if err := sink.PutNode(n); err != nil {
			return err
		}
	}
	for _, f := range files {
		if err := sink.PutFile(f); err != nil {
			return err
		}
	}
	for _, e := range edges {
		if err := sink.PutEdge(e, ownerPath(e)); err != nil {
			return err
		}
	}
	return nil
}
