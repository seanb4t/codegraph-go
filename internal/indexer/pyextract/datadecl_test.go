package pyextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestPythonDataDeclarations covers the Python declarations of the
// data-symbols contract. A module assignment or annotation declares a
// variable whatever its spelling, a tuple or list target declares each
// name, and a repeated assignment declares nothing more: it is a write.
// A name that an import, a function or a class declares gets no
// variable. A class body binding and a write through the receiver of a
// method declare one field per class and name, anchored at the earliest.
// A static method's parameter named self, a method under an unknown
// decorator, and a name the class declares as a method declare no field.
func TestPythonDataDeclarations(t *testing.T) {
	src := `import os
from lib import run

LIMIT = 3
count: int = 0
pending: list
first, second = 1, 2
[left, right] = pair
_ = 5
count = 4
run = 5

def helper():
    local = 1
    return local

class Widget:
    size_hint = 1
    label: str

    def __init__(me, v):
        me.value = v
        me.size_hint = 2

    @property
    def area(self):
        return self.value

    @area.setter
    def area(self, a):
        self.area = a

    @staticmethod
    def build(self):
        self.made = 1

    @classmethod
    def make(cls):
        cls.registry = []

    @custom
    def odd(self):
        self.strange = 1
`
	result, err := Extract(newTestParser(t), "widget", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	type key struct{ kind, qname string }
	count := map[key]int{}
	byKey := map[key]*goextract.ExtractedNode{}
	for i := range result.Nodes {
		n := result.Nodes[i].Node
		k := key{n.Kind, n.QualifiedName}
		count[k]++
		byKey[k] = &result.Nodes[i]
	}
	for _, k := range []key{
		{goextract.KindVariable, "LIMIT"}, {goextract.KindVariable, "count"},
		{goextract.KindVariable, "pending"}, {goextract.KindVariable, "first"},
		{goextract.KindVariable, "second"}, {goextract.KindVariable, "left"},
		{goextract.KindVariable, "right"}, {goextract.KindVariable, "_"},
		{goextract.KindField, "Widget.size_hint"}, {goextract.KindField, "Widget.label"},
		{goextract.KindField, "Widget.value"}, {goextract.KindField, "Widget.registry"},
	} {
		if count[k] != 1 {
			t.Errorf("%s %s: %d nodes, want 1", k.kind, k.qname, count[k])
		}
	}
	for k := range count {
		switch {
		case k.kind == goextract.KindConstant:
			t.Errorf("uppercase spelling made %s a constant", k.qname)
		case k.kind == goextract.KindVariable && (k.qname == "os" || k.qname == "run" || k.qname == "helper" || k.qname == "Widget" || k.qname == "local"):
			t.Errorf("%s became a variable", k.qname)
		case k.kind == goextract.KindField && (k.qname == "Widget.area" || k.qname == "Widget.made" || k.qname == "Widget.strange"):
			t.Errorf("unexpected field %s", k.qname)
		}
	}

	widget := byKey[key{goextract.KindStruct, "Widget"}]
	if widget == nil {
		t.Fatal("no Widget class node")
	}
	for _, f := range []string{"Widget.size_hint", "Widget.label", "Widget.value", "Widget.registry"} {
		fn := byKey[key{goextract.KindField, f}]
		if fn == nil {
			continue
		}
		if !hasIntraEdge(result, widget.Node.Id, fn.Node.Id, goextract.RefKindContains) {
			t.Errorf("no contains edge from Widget to %s", f)
		}
	}
	if sh := byKey[key{goextract.KindField, "Widget.size_hint"}]; sh != nil && sh.Node.StartLine != 18 {
		t.Errorf("Widget.size_hint anchored at line %d, want the class body binding at 18", sh.Node.StartLine)
	}
	if c := byKey[key{goextract.KindVariable, "count"}]; c != nil && c.Node.StartLine != 5 {
		t.Errorf("count anchored at line %d, want its first binding at 5", c.Node.StartLine)
	}
}
