package csharpextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestCSharpInstantiates proves a C# `new T(...)` object_creation_expression
// emits a RefKindInstantiates ref to T's simple name — D-09 Pass-1 capture
// (01-RESEARCH.md §B), mirroring goextract's TestExtract_Instantiates.
func TestCSharpInstantiates(t *testing.T) {
	src := `namespace P;

public class Widget
{
    public Widget Make()
    {
        Widget w = new Widget();
        return w;
    }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "P", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	found := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindInstantiates && u.Name == "Widget" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected instantiates ref to Widget (new Widget()), got %+v", result.Unresolved)
	}
}

// TestCSharpTypeOf proves a C# class field's declared type (`Helper f;`)
// emits a RefKindTypeOf ref anchored at the ENCLOSING TYPE (this extractor
// emits no field node — a documented D-02 precision note) — D-09 Pass-1
// capture (01-RESEARCH.md §B). A primitive-typed field emits no ref.
func TestCSharpTypeOf(t *testing.T) {
	src := `namespace P;

public class Widget
{
    private Helper field1;
    private int primField;
}
`
	p := newTestParser(t)
	result, err := Extract(p, "P", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	widget := findNode(result, goextract.KindStruct, "Widget")
	if widget == nil {
		t.Fatal("Widget node not found")
	}

	found := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf && u.Name == "Helper" && u.FromID == widget.Node.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("expected type_of ref from Widget to Helper (field declared type), got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf && (u.Name == "int" || u.Name == "primField") {
			t.Errorf("expected NO type_of ref for a primitive-typed field, got %+v", u)
		}
	}
}

// TestCSharpReturns proves a C# method's declared return type (`Helper
// Make()`) emits a RefKindReturns ref from the method's own node to Helper's
// name — D-09 Pass-1 capture (01-RESEARCH.md §B), reusing the
// already-parsed return type. A primitive/void return emits no ref.
func TestCSharpReturns(t *testing.T) {
	src := `namespace P;

public class Widget
{
    public Helper Make() { return null; }
    public int Count() { return 0; }
    public void DoNothing() { }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "P", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	found := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReturns && u.Name == "Helper" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected returns ref from Make to Helper, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReturns && (u.Name == "int" || u.Name == "void") {
			t.Errorf("expected NO returns ref for a primitive/void return, got %+v", u)
		}
	}
}

// TestCSharpReferences proves a C# member-access value read on a type that
// is NEITHER called NOR imported (`Helper.NAME` used as a value, not
// `Helper.Assist()`) emits a RefKindReferences ref, and that a CALLED
// symbol never also emits one (de-dup vs calls) — D-09 Pass-1 capture
// (01-RESEARCH.md §B).
func TestCSharpReferences(t *testing.T) {
	src := `namespace P;

public class Widget
{
    public void Run()
    {
        object x = Helper.NAME;
        Helper.Assist();
    }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "P", "Widget.cs", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	foundRef := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReferences && u.Name == "NAME" {
			foundRef = true
		}
	}
	if !foundRef {
		t.Errorf("expected references ref to NAME (Helper.NAME value read, not called/imported), got %+v", result.Unresolved)
	}

	foundCall := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindCalls && u.Name == "Assist" {
			foundCall = true
		}
	}
	if !foundCall {
		t.Errorf("expected calls ref to Assist, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReferences && u.Name == "Assist" {
			t.Errorf("expected NO references ref for a called symbol (de-dup), got %+v", u)
		}
	}
}
