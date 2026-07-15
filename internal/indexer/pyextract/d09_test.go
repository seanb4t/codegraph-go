package pyextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestPyInstantiates proves a Python `Foo()` call whose callee LOOKS like a
// class reference (PascalCase, per this extractor's isLikelyTypeName
// convention gate) emits a RefKindInstantiates ref candidate ALONGSIDE the
// ordinary RefKindCalls ref — D-09 Pass-1 capture (01-RESEARCH.md §B).
// Python instantiation is syntactically identical to a plain call (`Foo()`
// may call a function OR construct a class); resolve.go's existing Kind-check
// disambiguation (plan 05, target must be a struct/class node) is what
// actually decides whether this becomes a real edge — this test only proves
// the Pass-1 candidate is emitted. A lowercase (snake_case) callee, the near-
// universal Python function-naming convention, emits NO instantiates
// candidate at all.
func TestPyInstantiates(t *testing.T) {
	src := `class Widget:
    def make(self):
        w = Widget()
        h = helper_func()
        return w
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindInstantiates, "Widget", "") {
		t.Errorf("expected instantiates ref to Widget (Widget()), got %+v", result.Unresolved)
	}
	if hasUnresolved(result, goextract.RefKindInstantiates, "helper_func", "") {
		t.Errorf("expected NO instantiates ref for a snake_case call (helper_func()), got %+v", result.Unresolved)
	}
}

// TestPyTypeOf proves a Python class-body annotated attribute (`field1:
// Helper`) emits a RefKindTypeOf ref anchored at the ENCLOSING CLASS (this
// extractor emits no field node — a documented D-02 precision note mirroring
// Java/C#'s identical field-type_of anchor choice), and that a function-body
// annotated LOCAL variable (`local: Helper = None`) emits its own
// RefKindTypeOf ref anchored at the enclosing method — D-09 Pass-1 capture
// (01-RESEARCH.md §B). A Python built-in type name (`int`) emits no ref
// (noise filter, mirrors goextract's isGoPredeclaredType discipline).
func TestPyTypeOf(t *testing.T) {
	src := `class Widget:
    field1: Helper
    field2: int

    def run(self):
        local: Helper = None
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	widget := findNode(result, goextract.KindStruct, "Widget")
	if widget == nil {
		t.Fatal("Widget node not found")
	}
	run := findNode(result, goextract.KindMethod, "run")
	if run == nil {
		t.Fatal("Widget.run node not found")
	}

	foundField := false
	foundLocal := false
	for _, u := range result.Unresolved {
		if u.Kind != goextract.RefKindTypeOf || u.Name != "Helper" {
			continue
		}
		switch u.FromID {
		case widget.Node.Id:
			foundField = true
		case run.Node.Id:
			foundLocal = true
		}
	}
	if !foundField {
		t.Errorf("expected type_of ref from Widget to Helper (class-body annotated field), got %+v", result.Unresolved)
	}
	if !foundLocal {
		t.Errorf("expected type_of ref from Widget.run to Helper (annotated local variable), got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf && u.Name == "int" {
			t.Errorf("expected NO type_of ref for a built-in type (int), got %+v", u)
		}
	}
}

// TestPyReturns proves a Python function's annotated return type (`-> Helper`)
// emits a RefKindReturns ref from the function's own node to Helper's name —
// D-09 Pass-1 capture (01-RESEARCH.md §B), reusing the already-parsed
// return_type field. A built-in-typed return (`-> int`) and an un-annotated
// return (no `->` at all) both emit NO ref.
func TestPyReturns(t *testing.T) {
	src := `class Widget:
    def make(self) -> Helper:
        return None

    def count(self) -> int:
        return 0

    def do_nothing(self):
        pass
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReturns, "Helper", "") {
		t.Errorf("expected returns ref from make to Helper, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReturns && u.Name == "int" {
			t.Errorf("expected NO returns ref for a built-in-typed return (int), got %+v", u)
		}
	}
}

// TestPyReturns_GenericAnnotation proves a subscripted/generic return
// annotation (`-> Optional[Helper]`) resolves to the OUTER named type
// (`Optional`) per RESEARCH §B's "generic/composite types resolve to the
// outer named type" precision note — the type parameter itself (Helper) is
// never separately resolved as its own returns ref.
func TestPyReturns_GenericAnnotation(t *testing.T) {
	src := `class Widget:
    def maybe(self) -> Optional[Helper]:
        return None
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReturns, "Optional", "") {
		t.Errorf("expected returns ref to the outer generic name Optional, got %+v", result.Unresolved)
	}
	if hasUnresolved(result, goextract.RefKindReturns, "Helper", "") {
		t.Errorf("expected NO separate returns ref to the type PARAMETER Helper, got %+v", result.Unresolved)
	}
}

// TestPyReferences proves a Python attribute value read that is NEITHER
// called NOR annotated (`Helper.NAME` used as a value, not `Helper.assist()`)
// emits a RefKindReferences ref, and that a CALLED symbol never also emits
// one (de-dup vs calls) — D-09 Pass-1 capture (01-RESEARCH.md §B).
func TestPyReferences(t *testing.T) {
	src := `class Widget:
    def run(self):
        x = Helper.NAME
        Helper.assist()
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReferences, "NAME", "") {
		t.Errorf("expected references ref to NAME (Helper.NAME value read, not called), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "") {
		t.Errorf("expected calls ref to assist, got %+v", result.Unresolved)
	}
	if hasUnresolved(result, goextract.RefKindReferences, "assist", "") {
		t.Errorf("expected NO references ref for a called symbol (de-dup), got %+v", result.Unresolved)
	}
}

// TestPyUnannotatedAbsence proves Python's dynamic-typing D-02 divergence
// (01-RESEARCH.md §B, CONTEXT.md D-09/D-10): an un-annotated variable
// assignment and an un-annotated function return emit NO type_of/returns ref
// at all — absence, not error, not a fabricated guess at the runtime type.
func TestPyUnannotatedAbsence(t *testing.T) {
	src := `class Widget:
    other = 5

    def run(self):
        local = 5
        return local

    def do_nothing(self):
        pass
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "widget.py", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf {
			t.Errorf("expected NO type_of ref for any un-annotated assignment, got %+v", u)
		}
		if u.Kind == goextract.RefKindReturns {
			t.Errorf("expected NO returns ref for any un-annotated function, got %+v", u)
		}
	}
}
