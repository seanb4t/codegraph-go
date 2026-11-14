package goextract

import (
	"testing"
)

// TestGoDataDeclarations covers the Go declarations of the data-symbols
// contract: each package var and const of a grouped or multi-name spec,
// no symbol for the blank identifier, and one field node for each
// explicitly named struct field, owned by its struct. An embedded field
// stays an embeds reference and gets no field node.
func TestGoDataDeclarations(t *testing.T) {
	src := `package p

import "example.com/q"

var (
	a, b = 1, 2
	_    = 3
)

const c, _ = 4, 5

type Point struct {
	X, Y int
	label string
	q.Base
	*Other
}

type Other struct {
	X int
}

func (p Point) Size() int { return p.X }
`
	result, err := Extract(newTestParser(t), "example.com/p", "p.go", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	type key struct{ kind, qname string }
	count := map[key]int{}
	byKey := map[key]*ExtractedNode{}
	for i := range result.Nodes {
		n := result.Nodes[i].Node
		k := key{n.Kind, n.QualifiedName}
		count[k]++
		byKey[k] = &result.Nodes[i]
	}
	for _, k := range []key{
		{KindVariable, "a"}, {KindVariable, "b"}, {KindConstant, "c"},
		{KindField, "Point.X"}, {KindField, "Point.Y"}, {KindField, "Point.label"},
		{KindField, "Other.X"},
	} {
		if count[k] != 1 {
			t.Errorf("%s %s: %d nodes, want 1", k.kind, k.qname, count[k])
		}
	}
	for k, c := range count {
		if k.qname == "_" || k.qname == "Point._" {
			t.Errorf("blank identifier became a %s node (%d)", k.kind, c)
		}
		if k.kind == KindField && (k.qname == "Point.Base" || k.qname == "Point.Other") {
			t.Errorf("embedded field became a field node: %s", k.qname)
		}
	}

	point := byKey[key{KindStruct, "Point"}]
	for _, f := range []string{"Point.X", "Point.Y", "Point.label"} {
		fn := byKey[key{KindField, f}]
		if fn == nil || point == nil {
			continue
		}
		if !hasIntraEdge(result, point.Node.Id, fn.Node.Id, RefKindContains) {
			t.Errorf("no contains edge from Point to %s", f)
		}
		if fn.Node.FilePath != "p.go" || fn.Node.StartLine == 0 {
			t.Errorf("%s has no source location: %+v", f, fn.Node)
		}
	}
	if x := byKey[key{KindField, "Point.X"}]; x != nil {
		if x.Node.Name != "X" || !x.Node.IsExported || x.Node.Language != "go" {
			t.Errorf("Point.X = %+v, want name X, exported, language go", x.Node)
		}
	}
	if l := byKey[key{KindField, "Point.label"}]; l != nil && l.Node.IsExported {
		t.Errorf("Point.label is exported, want unexported")
	}
	if px, ox := byKey[key{KindField, "Point.X"}], byKey[key{KindField, "Other.X"}]; px != nil && ox != nil && px.Node.Id == ox.Node.Id {
		t.Errorf("Point.X and Other.X share id %s", px.Node.Id)
	}
	if !hasUnresolved(result, RefKindEmbeds, "Base", "q") || !hasUnresolved(result, RefKindEmbeds, "Other", "") {
		t.Errorf("embedded fields lost their embeds references: %+v", result.Unresolved)
	}
}
