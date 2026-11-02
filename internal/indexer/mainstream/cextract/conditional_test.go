package cextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestAlternativesUnderIfdefAreDistinctDeclarations checks design.md D16
// for C: the two `f` definitions under the branches of one `#ifdef` are
// two declarations with different ids, and the call in `g` records `f`.
func TestAlternativesUnderIfdefAreDistinctDeclarations(t *testing.T) {
	src := "#ifdef FAST\nint f(int a) { return 1; }\n#else\nint f(int a) { return 2; }\n#endif\nint g(void) { return f(1); }\n"
	p := newCTestParser(t)
	res, err := Extract(p, "m", "m.c", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	var ids []string
	for _, en := range res.Nodes {
		if en.Node.Kind == goextract.KindFunction && en.Node.Name == "f" {
			ids = append(ids, en.Node.Id)
		}
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("f ids = %v, want two distinct ids", ids)
	}
	calls := 0
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls && r.Name == "f" {
			calls++
		}
	}
	if calls != 1 {
		t.Errorf("recorded %d calls to f, want 1", calls)
	}
}

// TestPrototypeWithDefinitionInFileIsOneNode checks design.md D16 for C:
// a prototype is a forward declaration of the definition in the same
// file, so `f` gets one node and the call in `g` has one target. A
// prototype with no definition in the file keeps its node.
func TestPrototypeWithDefinitionInFileIsOneNode(t *testing.T) {
	src := "int f(int a);\nint h(void);\nint f(int a) { return a; }\nint g(void) { return f(1) + h(); }\n"
	p := newCTestParser(t)
	res, err := Extract(p, "m", "m.c", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	count := map[string]int{}
	for _, en := range res.Nodes {
		if en.Node.Kind == goextract.KindFunction {
			count[en.Node.Name]++
		}
	}
	if count["f"] != 1 || count["h"] != 1 || count["g"] != 1 {
		t.Errorf("function nodes = %v, want one each of f, h and g", count)
	}
}
