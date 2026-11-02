package rubyextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestReopenedClassIsOneNode checks design.md D16 for Ruby: a class
// opened twice in one file is one node whose span covers both blocks,
// and the methods of both blocks are its members.
func TestReopenedClassIsOneNode(t *testing.T) {
	src := `class Widget
  def first
  end
end

class Widget
  def second
  end
end
`
	p := newTestParser(t)
	res, err := Extract(p, "widget", "widget.rb", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	var classes []*goextract.ExtractedNode
	for i := range res.Nodes {
		if res.Nodes[i].Node.Kind == goextract.KindStruct && res.Nodes[i].Node.Name == "Widget" {
			classes = append(classes, &res.Nodes[i])
		}
	}
	if len(classes) != 1 {
		t.Fatalf("Widget has %d nodes, want 1", len(classes))
	}
	w := classes[0].Node
	if w.StartLine != 1 || w.EndLine != 9 {
		t.Errorf("Widget spans lines %d-%d, want 1-9", w.StartLine, w.EndLine)
	}
	for _, name := range []string{"first", "second"} {
		m := findNode(res, goextract.KindMethod, name)
		if m == nil {
			t.Errorf("no method %s", name)
			continue
		}
		found := false
		for _, e := range res.IntraEdges {
			if e.Edge.Kind == "contains" && e.Edge.Source == w.Id && e.Edge.Target == m.Node.Id {
				found = true
			}
		}
		if !found {
			t.Errorf("Widget does not contain %s", name)
		}
	}
}
