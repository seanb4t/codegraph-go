package symbols

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

const csharpDefault = "internal"

// copyOf is one copy of the node id "struct:Order" as one file emits it.
func copyOf(visibility string, exported bool, startLine int32) *schema.Node {
	return &schema.Node{
		Id: "struct:Order", Kind: goextract.KindStruct, Name: "Order", Language: "csharp",
		Visibility: visibility, IsExported: exported, StartLine: startLine,
	}
}

func declOf(relPath string, nodes ...*schema.Node) Declarations {
	return Declarations{ImportPath: "App", RelPath: relPath, Language: "csharp", Nodes: nodes, DefaultVisibility: csharpDefault}
}

func TestBuilderMergesSharedID(t *testing.T) {
	for _, tc := range []struct {
		name           string
		decls          []Declarations
		wantVisibility string
		wantExported   bool
		wantStartLine  int32
	}{
		{
			name:           "no modifier, then public",
			decls:          []Declarations{declOf("A.cs", copyOf("internal", false, 1)), declOf("B.cs", copyOf("public", true, 1))},
			wantVisibility: "public", wantExported: true, wantStartLine: 1,
		},
		{
			name:           "no modifier, then private",
			decls:          []Declarations{declOf("A.cs", copyOf("internal", false, 1)), declOf("B.cs", copyOf("private", false, 1))},
			wantVisibility: "private", wantStartLine: 1,
		},
		{
			name:           "two parts with no modifier",
			decls:          []Declarations{declOf("A.cs", copyOf("internal", false, 1)), declOf("B.cs", copyOf("internal", false, 1))},
			wantVisibility: "internal", wantStartLine: 1,
		},
		{
			name:           "two copies of one route id from one file",
			decls:          []Declarations{declOf("app.js", copyOf("", false, 3), copyOf("", false, 7))},
			wantVisibility: "", wantStartLine: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantVisibility == "" {
				for i := range tc.decls {
					tc.decls[i].DefaultVisibility = ""
				}
			}
			b := NewBuilder()
			b.AddResults(tc.decls)
			view := b.Freeze()

			count := 0
			for range view.Nodes() {
				count++
			}
			if count != 1 {
				t.Fatalf("the view holds %d nodes, want the one merged node", count)
			}
			got := view.Node("struct:Order")
			if got.Visibility != tc.wantVisibility || got.IsExported != tc.wantExported || got.StartLine != tc.wantStartLine {
				t.Errorf("merged node: visibility %q, exported %v, start line %d; want %q, %v, %d",
					got.Visibility, got.IsExported, got.StartLine, tc.wantVisibility, tc.wantExported, tc.wantStartLine)
			}
			if cands := view.CandidatesIn("App", "Order"); len(cands) != 1 || cands[0].Node != got {
				t.Errorf("the candidates of Order are %+v, want the one merged node", cands)
			}
			if view.Collisions() != 0 {
				t.Errorf("Collisions = %d, want 0: the copies of one id are one symbol", view.Collisions())
			}
		})
	}
}

// TestBuilderMergeProperty checks the merge against a reference fold over
// the copies in path order, and that a copy added twice changes nothing.
func TestBuilderMergeProperty(t *testing.T) {
	visibilities := rapid.SampledFrom([]string{"internal", "public", "private", "protected"})
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 5).Draw(t, "copies")
		copies := make([]*schema.Node, n)
		for i := range copies {
			copies[i] = copyOf(visibilities.Draw(t, "visibility"), rapid.Bool().Draw(t, "exported"), rapid.Int32Range(1, 50).Draw(t, "startLine"))
		}
		twice := rapid.IntRange(0, n-1).Draw(t, "addedTwice")

		// The reference: the first copy, the first visibility that is not
		// the default, and exported if any copy is.
		want := proto.Clone(copies[0]).(*schema.Node)
		want.Visibility = csharpDefault
		want.IsExported = false
		for _, c := range copies {
			if want.Visibility == csharpDefault && c.Visibility != csharpDefault {
				want.Visibility = c.Visibility
			}
			want.IsExported = want.IsExported || c.IsExported
		}

		var decls []Declarations
		for i, c := range copies {
			path := string(rune('A'+i)) + ".cs"
			decls = append(decls, declOf(path, proto.Clone(c).(*schema.Node)))
			if i == twice {
				decls = append(decls, declOf(path, proto.Clone(c).(*schema.Node)))
			}
		}
		b := NewBuilder()
		b.AddResults(decls)
		got := b.Freeze().Node("struct:Order")

		if !proto.Equal(got, want) {
			t.Fatalf("merged node = %v, want %v", got, want)
		}
	})
}
