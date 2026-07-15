package tsextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestTSInstantiates proves a TS/JS `new Foo()` new_expression emits a
// RefKindInstantiates ref to Foo's simple name — D-09 Pass-1 capture
// (01-RESEARCH.md §B), mirroring goextract's TestExtract_Instantiates. A
// qualified `new mod.Foo()` resolves Foo via memberAccessAlias, exactly
// like recordCall's own member_expression handling.
func TestTSInstantiates(t *testing.T) {
	src := `class Widget {
    make(): Widget {
        const w = new Widget();
        return w;
    }
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "widget", "widget.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindInstantiates, "Widget", "") {
		t.Errorf("expected instantiates ref to Widget (new Widget()), got %+v", result.Unresolved)
	}
}

// TestTSTypeOf proves a TS annotated local variable (`const x: Helper =
// ...`) emits a RefKindTypeOf ref anchored at the enclosing method — D-09
// Pass-1 capture (01-RESEARCH.md §B). A predefined/primitive-typed
// declaration (`number`) emits no ref, and an untyped JS-style declaration
// (no annotation at all) emits no ref either — both absence, not error.
func TestTSTypeOf(t *testing.T) {
	src := `class Widget {
    run(): void {
        const x: Helper = null;
        const n: number = 1;
        const untyped = 5;
    }
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "widget", "widget.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	run := findNode(result, goextract.KindMethod, "run")
	if run == nil {
		t.Fatal("Widget.run node not found")
	}

	found := false
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf && u.Name == "Helper" && u.FromID == run.Node.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("expected type_of ref from run to Helper (annotated local), got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf && (u.Name == "number" || u.Name == "untyped" || u.Name == "n") {
			t.Errorf("expected NO type_of ref for a primitive-typed or untyped declaration, got %+v", u)
		}
	}
}

// TestTSReturns proves a TS function's declared return type (`function
// make(): Helper`) emits a RefKindReturns ref from the function's own node
// to Helper's name — D-09 Pass-1 capture (01-RESEARCH.md §B), reusing the
// already-parsed return_type field. A primitive/void return emits no ref.
func TestTSReturns(t *testing.T) {
	src := `function make(): Helper {
    return null;
}

function count(): number {
    return 0;
}

function doNothing(): void {
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "widget", "widget.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if !hasUnresolved(result, goextract.RefKindReturns, "Helper", "") {
		t.Errorf("expected returns ref from make to Helper, got %+v", result.Unresolved)
	}
	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindReturns && (u.Name == "number" || u.Name == "void") {
			t.Errorf("expected NO returns ref for a primitive/void return, got %+v", u)
		}
	}
}

// TestTSReferences proves a TS/JS attribute value read that is NEITHER
// called NOR imported (`Helper.NAME` used as a value, not `Helper.assist()`)
// emits a RefKindReferences ref, and that a CALLED symbol never also emits
// one (de-dup vs calls) — D-09 Pass-1 capture (01-RESEARCH.md §B).
func TestTSReferences(t *testing.T) {
	src := `function run(): void {
    const x = Helper.NAME;
    Helper.assist();
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "widget", "widget.ts", []byte(src))
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

// TestTSUntypedJSAbsence proves a plain JavaScript file (no type annotations
// at all — JS has no type-annotation syntax) emits no type_of/returns refs,
// a documented D-02 divergence (01-RESEARCH.md §B) mirroring Python's
// dynamic-typing absence: JS is untyped by construction, not merely
// "un-annotated," but the resulting Pass-1 behavior is identical — absence,
// not a fabricated guess.
func TestTSUntypedJSAbsence(t *testing.T) {
	src := `function run() {
    const x = 5;
    return x;
}
`
	p := newTestParser(t, "js")
	result, err := Extract(p, "widget", "widget.js", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	for _, u := range result.Unresolved {
		if u.Kind == goextract.RefKindTypeOf {
			t.Errorf("expected NO type_of ref in untyped JS, got %+v", u)
		}
		if u.Kind == goextract.RefKindReturns {
			t.Errorf("expected NO returns ref in untyped JS, got %+v", u)
		}
	}
}
