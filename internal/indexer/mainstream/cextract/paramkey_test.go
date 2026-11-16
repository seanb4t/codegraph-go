package cextract

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// keyOf extracts `int f(<params>);` and returns the parameter key of f.
func keyOf(t testtree.T, p parser.Parser, file, params string) string {
	t.Helper()
	res, err := Extract(p, file, file, []byte("int f("+params+");\n"))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract(%q): %v / %v", params, err, res.Err)
	}
	n := findNode(res, goextract.KindFunction, "f")
	if n == nil {
		t.Fatalf("Extract(%q): no node f", params)
	}
	return n.Node.GetCParamKey()
}

func TestParamKey(t *testing.T) {
	c, cpp := newCTestParser(t), newCppTestParser(t)
	type side struct {
		cpp    bool
		params string
	}
	key := func(s side) string {
		if s.cpp {
			return keyOf(t, cpp, "f.cpp", s.params)
		}
		return keyOf(t, c, "f.c", s.params)
	}
	C := func(p string) side { return side{params: p} }
	X := func(p string) side { return side{cpp: true, params: p} }

	equal := [][2]side{
		{C("const char *"), C("const char* s")},
		{C("char\t*s"), C("char *s")},
		{C("char\n*s"), C("char *s")},
		{X("int a, int b = 0"), X("int a, int b")},
		{C("int /* x */ a"), C("int a")},
		{C("int a[10]"), C("int[10]")},
		{C("char s[]"), C("char[]")},
		{C("int[N]"), C("int a[N]")},
		{C("int (*)(int)"), C("int(*)(int)")},
		{C("void (*cb)(int x)"), C("void (*)(int)")},
		{C("void (*cb) (int)"), C("void (*)(int)")},
		{C("void (*cb )(int)"), C("void (*)(int)")},
		{X("void (&cb)(int)"), X("void (&)(int)")},
		{X("int &r"), X("int &")},
		{X("int &&r"), X("int &&")},
		{C("void (**pp)(int)"), C("void (**)(int)")},
		{C("void (*fp[3])(int)"), C("void (*[3])(int)")},
		{C("void"), C("")},
		{C("int a, ..."), C("int, ...")},
		{X("int a, ..."), X("int, ...")},
	}
	for _, q := range []string{"const", "volatile", "restrict", "_Atomic", "_Nonnull"} {
		equal = append(equal, [2]side{C("void (*" + q + " cb)(int)"), C("void (*" + q + ")(int)")})
	}
	for _, pair := range equal {
		if a, b := key(pair[0]), key(pair[1]); a != b {
			t.Errorf("key(%q) = %q, key(%q) = %q, want equal", pair[0].params, a, pair[1].params, b)
		}
	}

	unequal := [][2]side{
		{C("void *"), C("void")},
		{C("void *"), C("")},
		{C("int *"), C("int")},
		{C("char *"), C("char **")},
		{X("int &"), X("int")},
		{C("const char *"), C("char *")},
		{C("int[10]"), C("int[11]")},
		{C("int"), C("const int x")},
		{C("int, ..."), C("int")},
		{X("int, ..."), X("int")},
	}
	for _, q := range []string{"const", "volatile", "restrict", "_Atomic", "_Nonnull"} {
		unequal = append(unequal, [2]side{C("void (*" + q + " cb)(int)"), C("void (*)(int)")})
	}
	for _, pair := range unequal {
		if a, b := key(pair[0]), key(pair[1]); a == b {
			t.Errorf("key(%q) = key(%q) = %q, want unequal", pair[0].params, pair[1].params, a)
		}
	}
}

// paramType is a drawn parameter type: a base type with qualifiers,
// pointer levels with their qualifiers, and an array size or a
// reference.
type paramType struct {
	quals     []string
	base      string
	pointers  [][]string
	size      string
	array     bool
	reference string
}

func drawParamType(t *rapid.T) paramType {
	qualifier := rapid.SampledFrom([]string{"const", "volatile"})
	pt := paramType{
		quals: rapid.SliceOfNDistinct(qualifier, 0, 2, rapid.ID[string]).Draw(t, "quals"),
		base:  rapid.SampledFrom([]string{"int", "char", "long", "unsigned int", "size_t", "struct S"}).Draw(t, "base"),
	}
	for range rapid.IntRange(0, 2).Draw(t, "pointers") {
		pt.pointers = append(pt.pointers, rapid.SliceOfNDistinct(qualifier, 0, 1, rapid.ID[string]).Draw(t, "pointerQuals"))
	}
	switch rapid.IntRange(0, 2).Draw(t, "suffix") {
	case 1:
		pt.array = true
		pt.size = rapid.SampledFrom([]string{"", "3", "N"}).Draw(t, "size")
	case 2:
		pt.reference = rapid.SampledFrom([]string{"&", "&&"}).Draw(t, "reference")
	}
	return pt
}

func (pt paramType) equal(o paramType) bool {
	if strings.Join(pt.quals, " ") != strings.Join(o.quals, " ") || pt.base != o.base ||
		len(pt.pointers) != len(o.pointers) || pt.array != o.array || pt.size != o.size || pt.reference != o.reference {
		return false
	}
	for i := range pt.pointers {
		if strings.Join(pt.pointers[i], " ") != strings.Join(o.pointers[i], " ") {
			return false
		}
	}
	return true
}

// render writes the parameter with an optional name, and with gap
// between its tokens.
func (pt paramType) render(name string, gap func() string) string {
	var tokens []string
	tokens = append(tokens, pt.quals...)
	tokens = append(tokens, pt.base)
	for _, quals := range pt.pointers {
		tokens = append(tokens, "*")
		tokens = append(tokens, quals...)
	}
	if pt.reference != "" {
		tokens = append(tokens, pt.reference)
	}
	if name != "" {
		tokens = append(tokens, name)
	}
	if pt.array {
		tokens = append(tokens, "[", pt.size, "]")
	}
	var b strings.Builder
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		b.WriteString(gap())
		b.WriteString(tok)
	}
	return b.String()
}

// TestParamKeyIgnoresNamesCommentsAndSpacing draws parameter
// declarations and checks that the key depends on the type alone.
func TestParamKeyIgnoresNamesCommentsAndSpacing(t *testing.T) {
	p := newCppTestParser(t)
	rapid.Check(t, func(t *rapid.T) {
		gap := func() string {
			return rapid.SampledFrom([]string{" ", "  ", "\t", "\n", " /* c */ ", "\n\t "}).Draw(t, "gap")
		}
		a, b := drawParamType(t), drawParamType(t)
		plain := keyOf(t, p, "f.cpp", a.render("", func() string { return " " }))
		named := keyOf(t, p, "f.cpp", a.render(rapid.SampledFrom([]string{"x", "value"}).Draw(t, "name"), gap))
		if plain != named {
			t.Fatalf("key %q of %+v changes to %q with a name, comments and spacing", plain, a, named)
		}
		if other := keyOf(t, p, "f.cpp", b.render("", func() string { return " " })); !a.equal(b) && other == plain {
			t.Fatalf("types %+v and %+v share the key %q", a, b, plain)
		}
	})
}
