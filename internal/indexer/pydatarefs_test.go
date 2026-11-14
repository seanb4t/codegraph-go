package indexer

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// TestPythonDataReferences covers the Python references of the
// data-symbols contract. A method's first parameter is its receiver
// whatever its name. An instance write in a subclass establishes the
// subclass's own field, which its reads then reach, and a read of an
// attribute the subclass does not bind reaches the base class. A
// property stays a method. Two methods that write one instance attribute
// write one field, apart from the same-named field of another class. A
// class attribute that holds an instance of a
// descriptor class of the file declares no field, so its reads and writes
// through an instance resolve to nothing. An explicit classmethod's class parameter
// reaches the class, and a staticmethod's parameter named self reaches
// nothing. A local, and a name the function assigns without a global
// statement, shadows the module variable. A call through a module
// variable or a class attribute reaches the function it holds, unless a
// subclass can replace the attribute. An augmented assignment to a
// module variable withholds the call through it.
func TestPythonDataReferences(t *testing.T) {
	r, _ := runAndSnapshot(t, testtree.Write(t, map[string]string{
		"lib.py": "def run():\n    return 0\n\n\ndef other():\n    return 1\n",
		"app.py": `from lib import run, other

handler = run
total = 0


class Base:
    limit = 10
    hook = run

    def fire(me):
        me.hook()
        return me.limit


class Child(Base):
    def __init__(self):
        self.limit = 5

    def show(self):
        return self.limit

    def inherited(self):
        return self.hook


class Shape:
    callback = other

    def __init__(self, w):
        self.width = w

    def resize(self, w):
        self.width = w

    @property
    def area(self):
        return self.width

    def describe(self):
        return self.area

    def ping(self):
        self.callback()

    @classmethod
    def create(cls):
        cls.made = True
        return cls.made

    @staticmethod
    def util(self):
        return self.width


class Other:
    def __init__(self):
        self.width = 1

    def stretch(self):
        self.width = 2


class Typed:
    def __get__(self, obj, owner):
        return 0

    def __set__(self, obj, value):
        pass


class Model:
    label = Typed()
    size = 3

    def __init__(self):
        self.label = "x"

    def show(self):
        return self.label + self.size


def reads():
    return total


def writes():
    global total
    total = 7
    total += 1


def shadow():
    total = 3
    handler = 4
    return total + handler


def call_handler():
    handler()


grown = run


def grow():
    global grown
    grown += other


def call_grown():
    grown()
`,
	}))
	const app = "app.py"
	refs := func(kind, src string) map[string]string {
		return dataEdges(t, r, app, kind, src, goextract.RefKindReferences)
	}
	calls := func(kind, src string) map[string]string {
		return dataEdges(t, r, app, kind, src, goextract.RefKindCalls)
	}

	if got := refs(goextract.KindMethod, "Base.fire")["field Base.limit"]; got != "read" {
		t.Errorf("fire -> Base.limit access = %q, want read through the receiver me", got)
	}
	if c := calls(goextract.KindMethod, "Base.fire"); len(c) != 0 {
		t.Errorf("fire calls %v, want none: Child extends Base, so it can replace hook", c)
	}
	init := refs(goextract.KindMethod, "Child.__init__")
	if init["field Child.limit"] != "write" {
		t.Errorf("Child.__init__ references %v, want a write of Child.limit", init)
	}
	if _, ok := init["field Base.limit"]; ok {
		t.Errorf("Child.__init__ writes Base.limit: %v", init)
	}
	if got := refs(goextract.KindMethod, "Child.show")["field Child.limit"]; got != "read" {
		t.Errorf("Child.show -> Child.limit access = %q, want read", got)
	}
	if got := refs(goextract.KindMethod, "Child.inherited")["field Base.hook"]; got != "read" {
		t.Errorf("Child.inherited -> Base.hook access = %q, want read", got)
	}

	describe := refs(goextract.KindMethod, "Shape.describe")
	if _, ok := describe["method Shape.area"]; !ok {
		t.Errorf("describe references %v, want the property method Shape.area", describe)
	}
	if _, ok := describe["field Shape.area"]; ok {
		t.Errorf("the property area became a field: %v", describe)
	}
	if _, ok := calls(goextract.KindMethod, "Shape.ping")["function other"]; !ok {
		t.Errorf("ping calls %v, want other through the class attribute callback", calls(goextract.KindMethod, "Shape.ping"))
	}
	if got := refs(goextract.KindMethod, "Shape.create")["field Shape.made"]; got != "read_write" {
		t.Errorf("create -> Shape.made access = %q, want read_write through cls", got)
	}
	if util := refs(goextract.KindMethod, "Shape.util"); len(util) != 0 {
		t.Errorf("the static method util references %v through a parameter named self", util)
	}

	for _, m := range []string{"Shape.__init__", "Shape.resize", "Other.__init__", "Other.stretch"} {
		owner, _, _ := strings.Cut(m, ".")
		got := refs(goextract.KindMethod, m)
		if got["field "+owner+".width"] != "write" || len(got) != 1 {
			t.Errorf("%s references %v, want only a write of %s.width", m, got, owner)
		}
	}
	widths := 0
	wit, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	for wit.Next() {
		if n := wit.Node(); n.Kind == goextract.KindField && n.QualifiedName == "Shape.width" {
			widths++
		}
	}
	wit.Close()
	if widths != 1 {
		t.Errorf("Shape has %d width fields, want one for both writing methods", widths)
	}

	show := refs(goextract.KindMethod, "Model.show")
	if show["field Model.size"] != "read" {
		t.Errorf("Model.show references %v, want a read of Model.size", show)
	}
	if len(show) != 1 {
		t.Errorf("Model.show references %v, want only Model.size: label holds a descriptor", show)
	}
	if init := refs(goextract.KindMethod, "Model.__init__"); len(init) != 0 {
		t.Errorf("Model.__init__ references %v, want none: label holds a descriptor", init)
	}
	modelCalls := calls(goextract.KindStruct, "Model")
	if _, ok := modelCalls["struct Typed"]; !ok || len(modelCalls) != 1 {
		t.Errorf("Model calls %v, want only the descriptor class Typed", modelCalls)
	}
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	for it.Next() {
		if n := it.Node(); n.QualifiedName == "Model.label" {
			t.Errorf("the descriptor attribute label became the %s %s", n.Kind, n.QualifiedName)
		}
	}
	it.Close()

	if got := refs(goextract.KindFunction, "reads")["variable total"]; got != "read" {
		t.Errorf("reads -> total access = %q, want read", got)
	}
	if got := refs(goextract.KindFunction, "writes")["variable total"]; got != "read_write" {
		t.Errorf("writes -> total access = %q, want read_write through global", got)
	}
	if s := refs(goextract.KindFunction, "shadow"); len(s) != 0 {
		t.Errorf("shadow references %v through its locals", s)
	}
	if _, ok := calls(goextract.KindFunction, "call_handler")["function run"]; !ok {
		t.Errorf("call_handler calls %v, want run through handler", calls(goextract.KindFunction, "call_handler"))
	}
	if got := refs(goextract.KindFunction, "call_handler")["variable handler"]; got != "read" {
		t.Errorf("call_handler -> handler access = %q, want read", got)
	}
	if c := calls(goextract.KindFunction, "call_grown"); len(c) != 0 {
		t.Errorf("call_grown calls %v, want none: grow augments grown with other", c)
	}
	if got := refs(goextract.KindFunction, "call_grown")["variable grown"]; got != "read" {
		t.Errorf("call_grown -> grown access = %q, want read", got)
	}
	if got := refs(goextract.KindFunction, "grow")["variable grown"]; got != "read_write" {
		t.Errorf("grow -> grown access = %q, want read_write from the augmented assignment", got)
	}
}

// TestPythonInheritedFieldOwnership covers bounded inherited field
// resolution. A read of an attribute that the class does not bind
// reaches the one base class that supplies it, through a single chain or
// through a diamond that reaches the same base twice. When two bases
// supply the name from different classes, the read reaches neither.
func TestPythonInheritedFieldOwnership(t *testing.T) {
	r, _ := runAndSnapshot(t, testtree.Write(t, map[string]string{
		"app.py": `class Root:
    x = 1


class A(Root):
    pass


class B:
    x = 2


class E(Root):
    pass


class Mixed(A, B):
    def read(self):
        return self.x


class Single(A):
    def read(self):
        return self.x


class Diamond(A, E):
    def read(self):
        return self.x
`,
	}))
	refs := func(src string) map[string]string {
		return dataEdges(t, r, "app.py", goextract.KindMethod, src, goextract.RefKindReferences)
	}
	if got := refs("Single.read")["field Root.x"]; got != "read" {
		t.Errorf("Single.read -> Root.x access = %q, want read through the single chain", got)
	}
	if got := refs("Diamond.read")["field Root.x"]; got != "read" {
		t.Errorf("Diamond.read -> Root.x access = %q, want read: both bases reach Root", got)
	}
	if m := refs("Mixed.read"); len(m) != 0 {
		t.Errorf("Mixed.read references %v, want none: A and B supply x from different classes", m)
	}
}
