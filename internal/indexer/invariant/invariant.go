// Package invariant checks the properties every committed graph must have,
// whatever wrote it: a full index, an incremental sync or a backfill. The
// tests of the indexer call it after each write. It has no tests of its
// own: the indexer tests that fail without the fixes it guards prove it.
package invariant

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Counts is what Check scanned. A caller asserts that each count it relies
// on is above zero, so that a check over an empty store cannot pass.
type Counts struct {
	Nodes int
	Edges int
	// Nesting is the number of contains edges that the range rule checked.
	Nesting int
}

// nestingLanguages are the languages whose contains edge from a type to a
// member in the same file stands for lexical nesting. A Go method belongs
// to its receiver type but is declared outside the type body, so Go is not
// here.
var nestingLanguages = map[string]bool{
	"typescript": true,
	"tsx":        true,
	"javascript": true,
	"java":       true,
	"python":     true,
	"csharp":     true,
	"php":        true,
	"ruby":       true,
}

// nestingParentKinds are the node kinds that hold their members lexically:
// classes, structs and enums are KindStruct, interfaces KindInterface.
var nestingParentKinds = map[string]bool{
	goextract.KindStruct:      true,
	goextract.KindInterface:   true,
	goextract.KindOverloadSet: true,
}

// Check scans r and returns an error that lists every violation of these
// rules:
//
//   - Each edge has a source and a target that are stored nodes.
//   - Meta.NodeCount equals the number of stored nodes, and Meta.EdgeCount
//     the number of stored edges.
//   - Each contains edge between a type and a member of the same file, in
//     a language where that edge stands for lexical nesting, has the
//     member's line range inside the type's line range.
func Check(r graphstore.Reader) (Counts, error) {
	var c Counts
	var errs []error

	nodes, err := readNodes(r)
	if err != nil {
		return c, err
	}
	c.Nodes = len(nodes)

	edges, err := readEdges(r)
	if err != nil {
		return c, err
	}
	c.Edges = len(edges)

	for _, e := range edges {
		src, okSrc := nodes[e.Source]
		dst, okDst := nodes[e.Target]
		if !okSrc {
			errs = append(errs, fmt.Errorf("edge %s -%s-> %s: source is not a stored node", e.Source, e.Kind, e.Target))
		}
		if !okDst {
			errs = append(errs, fmt.Errorf("edge %s -%s-> %s: target is not a stored node", e.Source, e.Kind, e.Target))
		}
		if !okSrc || !okDst || !isNesting(e, src, dst) {
			continue
		}
		c.Nesting++
		if dst.StartLine < src.StartLine || dst.EndLine > src.EndLine {
			errs = append(errs, fmt.Errorf("contains %s %q (lines %d-%d) -> %s %q (lines %d-%d) in %s: the member is outside the type",
				src.Kind, src.QualifiedName, src.StartLine, src.EndLine,
				dst.Kind, dst.QualifiedName, dst.StartLine, dst.EndLine, src.FilePath))
		}
	}

	meta, err := r.GetMeta()
	if err != nil {
		return c, fmt.Errorf("invariant: read meta: %w", err)
	}
	if meta.GetNodeCount() != int64(c.Nodes) {
		errs = append(errs, fmt.Errorf("Meta.NodeCount = %d, the store holds %d nodes", meta.GetNodeCount(), c.Nodes))
	}
	if meta.GetEdgeCount() != int64(c.Edges) {
		errs = append(errs, fmt.Errorf("Meta.EdgeCount = %d, the store holds %d edges", meta.GetEdgeCount(), c.Edges))
	}
	return c, errors.Join(errs...)
}

// isNesting reports whether e is a contains edge that stands for lexical
// nesting. A merged C# partial type has a sentinel file path, so the
// same-file condition leaves it out.
func isNesting(e *schema.Edge, parent, child *schema.Node) bool {
	return e.Kind == goextract.RefKindContains &&
		parent.FilePath == child.FilePath &&
		nestingParentKinds[parent.Kind] &&
		nestingLanguages[parent.Language]
}

// Equal compares the nodes and edges of a and b and returns an error that
// names the first difference, or nil. Route nodes and the edges that touch
// them are left out: route detection runs only in a full index.
func Equal(a, b graphstore.Reader) error {
	an, err := readNodes(a)
	if err != nil {
		return err
	}
	bn, err := readNodes(b)
	if err != nil {
		return err
	}
	dropRoutes(an)
	dropRoutes(bn)
	for _, id := range sortedIDs(an, bn) {
		x, inA := an[id]
		y, inB := bn[id]
		switch {
		case !inA:
			return fmt.Errorf("node %s (%s %q in %s) is only in the second store", id, y.Kind, y.QualifiedName, y.FilePath)
		case !inB:
			return fmt.Errorf("node %s (%s %q in %s) is only in the first store", id, x.Kind, x.QualifiedName, x.FilePath)
		case !proto.Equal(x, y):
			return fmt.Errorf("node %s differs:\n first: %v\nsecond: %v", id, x, y)
		}
	}

	ae, err := readEdges(a)
	if err != nil {
		return err
	}
	be, err := readEdges(b)
	if err != nil {
		return err
	}
	am := edgesByKey(ae)
	bm := edgesByKey(be)
	keys := make([]string, 0, len(am)+len(bm))
	for k := range am {
		keys = append(keys, k)
	}
	for k := range bm {
		if _, ok := am[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		x, inA := am[k]
		y, inB := bm[k]
		switch {
		case !inA:
			return fmt.Errorf("edge %s is only in the second store", k)
		case !inB:
			return fmt.Errorf("edge %s is only in the first store", k)
		case !proto.Equal(x, y):
			return fmt.Errorf("edge %s differs:\n first: %v\nsecond: %v", k, x, y)
		}
	}
	return nil
}

func readNodes(r graphstore.Reader) (map[string]*schema.Node, error) {
	it, err := r.IterateNodes()
	if err != nil {
		return nil, fmt.Errorf("invariant: iterate nodes: %w", err)
	}
	defer it.Close()
	nodes := make(map[string]*schema.Node)
	for it.Next() {
		n := it.Node()
		nodes[n.Id] = n
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("invariant: iterate nodes: %w", err)
	}
	return nodes, nil
}

func readEdges(r graphstore.Reader) ([]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, fmt.Errorf("invariant: iterate edges: %w", err)
	}
	defer it.Close()
	var edges []*schema.Edge
	for it.Next() {
		edges = append(edges, it.Edge())
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("invariant: iterate edges: %w", err)
	}
	return edges, nil
}

func dropRoutes(nodes map[string]*schema.Node) {
	for id, n := range nodes {
		if n.Kind == goextract.KindRoute {
			delete(nodes, id)
		}
	}
}

// edgesByKey keys each edge by its store identity, leaving out an edge
// that touches a route node.
func edgesByKey(edges []*schema.Edge) map[string]*schema.Edge {
	m := make(map[string]*schema.Edge, len(edges))
	for _, e := range edges {
		if strings.HasPrefix(e.Source, routePrefix) || strings.HasPrefix(e.Target, routePrefix) {
			continue
		}
		m[e.Source+" -"+e.Kind+"-> "+e.Target] = e
	}
	return m
}

// routePrefix starts the id of every route node: nodeid.NodeID puts the
// kind first.
const routePrefix = goextract.KindRoute + ":"

func sortedIDs(a, b map[string]*schema.Node) []string {
	ids := make([]string, 0, len(a)+len(b))
	for id := range a {
		ids = append(ids, id)
	}
	for id := range b {
		if _, ok := a[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
