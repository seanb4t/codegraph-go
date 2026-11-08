package resolve

import (
	"bytes"
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"
	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// storedForm is the part of a Graph that a full index writes: the nodes
// in id order, the files in path order and the collapsed edges, each
// marshaled deterministically.
type storedForm struct {
	nodes, files, edges [][]byte
	unresolved          int
}

func marshal(t *rapid.T, m proto.Message) []byte {
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func storedFormOf(t *rapid.T, g Graph) storedForm {
	nodeFilePath := make(map[string]string, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeFilePath[n.Id] = n.FilePath
	}
	var f storedForm
	for _, n := range append(slices.Clip(g.Nodes), g.PackageNodes...) {
		f.nodes = append(f.nodes, marshal(t, n))
	}
	for _, file := range g.Files {
		f.files = append(f.files, marshal(t, file))
	}
	for _, e := range CollapseEdges(g.Edges, nodeFilePath) {
		f.edges = append(f.edges, marshal(t, e))
	}
	// Two nodes or files can share a key, so the comparison sorts the
	// records themselves.
	slices.SortFunc(f.nodes, bytes.Compare)
	slices.SortFunc(f.files, bytes.Compare)
	f.unresolved = g.Unresolved
	return f
}

// TestLinkIgnoresResultOrder checks that a resolution of the
// overloadfixture and repeatfixture results in any order gives the graph
// that the files' own order gives: the same nodes, files and collapsed
// edges, and the same unresolved count. The Builder, Types, Calls and
// Link all see the permuted order.
func TestLinkIgnoresResultOrder(t *testing.T) {
	for _, root := range []string{"../testdata/overloadfixture", "../testdata/repeatfixture"} {
		t.Run(root, func(t *testing.T) {
			results, modulePath := extractTree(t, root)
			resolveAll := func(rt *rapid.T, rs []goextract.FileResult) storedForm {
				nodes, packageNodes, edges, files, unresolved := resolveRefs(rs, modulePath)
				return storedFormOf(rt, Graph{Nodes: nodes, PackageNodes: packageNodes, Edges: edges, Files: files, Unresolved: unresolved})
			}
			rapid.Check(t, func(rt *rapid.T) {
				want := resolveAll(rt, results)
				if len(want.nodes) == 0 || len(want.edges) == 0 || len(want.files) == 0 {
					rt.Fatalf("%s resolved to %d nodes, %d edges and %d files, want more than zero of each", root, len(want.nodes), len(want.edges), len(want.files))
				}
				got := resolveAll(rt, rapid.Permutation(results).Draw(rt, "order"))
				compare := func(what string, got, want [][]byte) {
					if !slices.EqualFunc(got, want, bytes.Equal) {
						rt.Fatalf("%s: %d %s for a permuted order, %d for the files' own order, or the same count with different records", root, len(got), what, len(want))
					}
				}
				compare("nodes", got.nodes, want.nodes)
				compare("files", got.files, want.files)
				compare("edges", got.edges, want.edges)
				if got.unresolved != want.unresolved {
					rt.Fatalf("%s: %d unresolved for a permuted order, %d for the files' own order", root, got.unresolved, want.unresolved)
				}
			})
		})
	}
}
