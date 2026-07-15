package javaextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestJavaInstantiates proves a Java `new T(...)` object_creation_expression
// emits a RefKindInstantiates ref to T's simple name — D-09 Pass-1 capture
// (01-RESEARCH.md §B), mirroring goextract's TestExtract_Instantiates.
func TestJavaInstantiates(t *testing.T) {
	src := `package p;

public class Widget {
    public Widget make() {
        Widget w = new Widget();
        return w;
    }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindInstantiates, "Widget", "") {
		t.Errorf("expected instantiates ref to Widget (new Widget()), got %+v", result.Unresolved)
	}
}

// TestJavaTypeOf proves a Java class field's declared type (`Helper f;`)
// emits a RefKindTypeOf ref anchored at the ENCLOSING TYPE (this extractor
// emits no field node — a documented D-02 precision note) — D-09 Pass-1
// capture (01-RESEARCH.md §B). A primitive-typed field emits no ref.
func TestJavaTypeOf(t *testing.T) {
	src := `package p;

public class Widget {
    private Helper field1;
    private int primField;
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
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

// TestJavaReturns proves a Java method's declared return type (`Helper
// make()`) emits a RefKindReturns ref from the method's own node to Helper's
// name — D-09 Pass-1 capture (01-RESEARCH.md §B), reusing the
// already-parsed return type. A primitive/void return emits no ref.
func TestJavaReturns(t *testing.T) {
	src := `package p;

public class Widget {
    public Helper make() { return null; }
    public int count() { return 0; }
    public void doNothing() { }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReturns, "Helper", "") {
		t.Errorf("expected returns ref from make to Helper, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReturns && (u.Name == "int" || u.Name == "void") {
			t.Errorf("expected NO returns ref for a primitive/void return, got %+v", u)
		}
	}
}

// TestJavaReferences proves a Java field-access value read on a type that
// is NEITHER called NOR imported (`Helper.NAME` used as a value, not
// `Helper.assist()`) emits a RefKindReferences ref, and that a CALLED
// symbol never also emits one (de-dup vs calls) — D-09 Pass-1 capture
// (01-RESEARCH.md §B).
func TestJavaReferences(t *testing.T) {
	src := `package p;

public class Widget {
    public void run() {
        Object x = Helper.NAME;
        Helper.assist();
    }
}
`
	p := newTestParser(t)
	result, err := Extract(p, "p", "Widget.java", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReferences, "NAME", "") {
		t.Errorf("expected references ref to NAME (Helper.NAME value read, not called/imported), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "") {
		t.Errorf("expected calls ref to assist, got %+v", result.Unresolved)
	}
	if hasUnresolved(result, goextract.RefKindReferences, "assist", "") {
		t.Errorf("expected NO references ref for a called symbol (de-dup), got %+v", result.Unresolved)
	}
}
