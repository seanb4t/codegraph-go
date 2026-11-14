package javaextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestJavaDataDeclarations covers the Java declarations of the
// data-symbols contract. Each declarator of a class or interface field
// declaration is one field, contained by its type. A static final field
// stays a field, an interface field is public, and a field and a method
// of one name are two symbols. A record component and an enum constant
// declare no field.
func TestJavaDataDeclarations(t *testing.T) {
	src := `package p;

public class Box {
    int width, height;
    private static final int LIMIT = 10;
    private int size;
    public int size() { return size; }
}

interface Shape {
    int SIDES = 4;
}

record Point(int x, int y) {}

enum Color { RED, GREEN }
`
	result, err := Extract(newTestParser(t), "p", "Box.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	fields := map[string]*goextract.ExtractedNode{}
	for i := range result.Nodes {
		if n := result.Nodes[i].Node; n.Kind == goextract.KindField {
			fields[n.QualifiedName] = &result.Nodes[i]
		}
	}
	owners := map[string]string{"Box.width": "Box", "Box.height": "Box", "Box.LIMIT": "Box", "Box.size": "Box", "Shape.SIDES": "Shape"}
	if len(fields) != len(owners) {
		t.Errorf("fields %v, want exactly %v", keys(fields), owners)
	}
	for qn, owner := range owners {
		f := fields[qn]
		if f == nil {
			t.Errorf("no field %s", qn)
			continue
		}
		o := findNode(result, goextract.KindStruct, owner)
		if o == nil {
			o = findNode(result, goextract.KindInterface, owner)
		}
		if o == nil || !hasIntraEdge(result, o.Node.Id, f.Node.Id, goextract.RefKindContains) {
			t.Errorf("no contains edge from %s to %s", owner, qn)
		}
	}
	if f := fields["Shape.SIDES"]; f != nil && f.Node.Visibility != "public" {
		t.Errorf("Shape.SIDES visibility = %q, want public", f.Node.Visibility)
	}
	if f := fields["Box.LIMIT"]; f != nil && f.Node.Visibility != "private" {
		t.Errorf("Box.LIMIT visibility = %q, want private", f.Node.Visibility)
	}
	method := findNode(result, goextract.KindMethod, "size")
	if method == nil || fields["Box.size"] == nil || method.Node.Id == fields["Box.size"].Node.Id {
		t.Errorf("field and method size do not have distinct ids: method %v", method)
	}
}

func keys(m map[string]*goextract.ExtractedNode) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
