package csharpextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestAlternativesUnderIfAreDistinctDeclarations checks design.md D16 for
// C#: the two `Process` methods under the branches of one `#if` are two
// declarations with different ids, both members of their type, so that a
// call to `Process` reaches their set as ambiguous.
func TestAlternativesUnderIfAreDistinctDeclarations(t *testing.T) {
	src := `public class Processor
{
#if FEATURE_SPAN
    public void Process(string template)
    {
    }
#else
    public void Process(string template)
    {
    }
#endif
}
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "Processor.cs", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	var ids []string
	for _, en := range res.Nodes {
		if en.Node.Kind == goextract.KindMethod && en.Node.Name == "Process" {
			ids = append(ids, en.Node.Id)
			if en.Node.Nested {
				t.Errorf("Process at line %d is marked nested, want a type member", en.Node.StartLine)
			}
		}
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("Process ids = %v, want two distinct ids", ids)
	}
	typ := findNode(res, goextract.KindStruct, "Processor")
	if typ == nil {
		t.Fatal("no Processor node")
	}
	for _, id := range ids {
		if !hasIntraEdge(res, typ.Node.Id, id, "contains") {
			t.Errorf("Processor does not contain Process %s", id)
		}
	}
}
