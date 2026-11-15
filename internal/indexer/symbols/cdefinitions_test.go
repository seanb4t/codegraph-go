package symbols

import (
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

func TestCDefinitionsNamed(t *testing.T) {
	fn := func(id, kind, file, lang string, prototype, exported bool) *schema.Node {
		return &schema.Node{
			Id: id, Kind: kind, Name: "f", QualifiedName: "f", FilePath: file, Language: lang,
			Prototype: prototype, IsExported: exported, CParamKey: "int",
		}
	}
	b := NewBuilder()
	for _, n := range []*schema.Node{
		fn("d3", goextract.KindFunction, "c.c", "c", false, true),
		fn("p1", goextract.KindFunction, "f.h", "c", true, true),
		fn("s1", goextract.KindFunction, "s.c", "c", false, false),
		fn("m1", goextract.KindMethod, "s.cpp", "cpp", false, true),
		fn("m2", goextract.KindMethod, "s.cpp", "cpp", false, true),
		fn("d1", goextract.KindFunction, "a.cpp", "cpp", false, true),
		fn("g1", goextract.KindFunction, "f.go", "go", false, true),
	} {
		b.AddNode(n)
	}
	if got, want := b.Freeze().CDefinitionsNamed("f"), []string{"d1", "d3"}; !slices.Equal(got, want) {
		t.Errorf("CDefinitionsNamed(f) = %q, want %q", got, want)
	}
}
