package indexer

import (
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

type recordingSink struct {
	nodes  []string
	files  []string
	edges  []resolve.EdgeTriple
	owners []string
}

func (s *recordingSink) PutNode(n *schema.Node) error {
	s.nodes = append(s.nodes, n.Id)
	return nil
}

func (s *recordingSink) PutFile(f *schema.File) error {
	s.files = append(s.files, f.Path)
	return nil
}

func (s *recordingSink) PutEdge(e *schema.Edge, ownerPath string) error {
	s.edges = append(s.edges, resolve.EdgeTriple{Source: e.Source, Kind: e.Kind, Target: e.Target})
	s.owners = append(s.owners, ownerPath)
	return nil
}

func TestStagePutsRecordsInKeyOrder(t *testing.T) {
	g := resolve.Graph{
		Nodes:        []*schema.Node{{Id: "c"}, {Id: "a"}, {Id: "b"}, {Id: "a"}},
		PackageNodes: []*schema.Node{{Id: "0pkg"}},
		Files:        []*schema.File{{Path: "z.go"}, {Path: "a.go"}, {Path: "m.go"}},
		Edges: []*schema.Edge{
			{Source: "b", Kind: "calls", Target: "c"},
			{Source: "a", Kind: "contains", Target: "b"},
			{Source: "a", Kind: "calls", Target: "c"},
			{Source: "a", Kind: "calls", Target: "b"},
		},
	}
	nodesBefore := slices.Clone(g.Nodes)

	var sink recordingSink
	n, err := stage(&sink, g, func(e *schema.Edge) string { return e.Source + ".go" })
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"0pkg", "a", "a", "b", "c"}; !slices.Equal(sink.nodes, want) {
		t.Errorf("nodes = %v, want %v", sink.nodes, want)
	}
	if n != 4 {
		t.Errorf("distinct node count = %d, want 4", n)
	}
	if want := []string{"a.go", "m.go", "z.go"}; !slices.Equal(sink.files, want) {
		t.Errorf("files = %v, want %v", sink.files, want)
	}
	wantEdges := []resolve.EdgeTriple{
		{Source: "a", Kind: "calls", Target: "b"},
		{Source: "a", Kind: "calls", Target: "c"},
		{Source: "a", Kind: "contains", Target: "b"},
		{Source: "b", Kind: "calls", Target: "c"},
	}
	if !slices.Equal(sink.edges, wantEdges) {
		t.Errorf("edges = %v, want %v", sink.edges, wantEdges)
	}
	if want := []string{"a.go", "a.go", "a.go", "b.go"}; !slices.Equal(sink.owners, want) {
		t.Errorf("owners = %v, want %v", sink.owners, want)
	}
	if !slices.Equal(g.Nodes, nodesBefore) {
		t.Error("stage reordered the caller's node slice")
	}
}
