package resolve

import (
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/extract"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

func TestIsIncludeEdge(t *testing.T) {
	mark := map[string]string{goextract.MetaInclude: goextract.IncludeQuoted}
	cases := []struct {
		name string
		edge *schema.Edge
		want bool
	}{
		{"an imports edge with the mark", &schema.Edge{Kind: goextract.RefKindImports, Metadata: mark}, true},
		{"an imports edge without the mark", &schema.Edge{Kind: goextract.RefKindImports}, false},
		{"an imports edge with another include value", &schema.Edge{Kind: goextract.RefKindImports, Metadata: map[string]string{goextract.MetaInclude: "angle"}}, false},
		{"an edge of another kind with the mark", &schema.Edge{Kind: goextract.RefKindReferences, Metadata: mark}, false},
	}
	for _, tc := range cases {
		if got := IsIncludeEdge(tc.edge); got != tc.want {
			t.Errorf("%s: IsIncludeEdge = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestIncludeWaitsMeetNoSymbol checks that the waits of an include
// (design D2) are under IncludeModule, and that no language gives a file
// that module key: not a header whose path an include names, and not a
// root file named after the directive.
func TestIncludeWaitsMeetNoSymbol(t *testing.T) {
	file := func(rel string) goextract.ExtractedNode {
		return goextract.ExtractedNode{Node: &schema.Node{
			Id: nodeid.NodeID(goextract.KindFile, rel, rel), Kind: goextract.KindFile,
			Name: rel, QualifiedName: rel, FilePath: rel, Language: "c",
		}}
	}
	main := goextract.FileResult{
		ImportPath: "src/main.c", RelPath: "src/main.c", Language: "c",
		Nodes: []goextract.ExtractedNode{file("src/main.c")},
		Unresolved: []goextract.UnresolvedRef{{
			FromID: file("src/main.c").Node.Id, Kind: goextract.RefKindImports, Name: "add.h", Quoted: true,
		}},
	}
	header := goextract.FileResult{
		ImportPath: "add.h", RelPath: "add.h", Language: "c",
		Nodes: []goextract.ExtractedNode{file("add.h")},
	}
	b := symbols.NewBuilder()
	b.AddResults(declarationsOf([]goextract.FileResult{header, main}))
	out := Types(b.Freeze(), "", main)
	var waits []string
	for _, u := range out.file.GetUnresolvedNames() {
		if u.GetModuleKey() != IncludeModule {
			t.Errorf("include wait %q under module %q, want %q", u.GetName(), u.GetModuleKey(), IncludeModule)
		}
		waits = append(waits, u.GetName())
	}
	if !slices.Equal(waits, []string{"src/add.h"}) {
		t.Errorf("include waits = %q, want [src/add.h]", waits)
	}

	for _, id := range extract.RegisteredLanguageIDs() {
		spec, _ := extract.ByID(id)
		for _, rel := range []string{"add.h", "src/add.h", "#include", "#include.py", "#include.rb"} {
			if key := spec.ModuleKey(nil, rel); key == IncludeModule {
				t.Errorf("%s gives %q the module key of an include wait", id, rel)
			}
		}
	}
}
