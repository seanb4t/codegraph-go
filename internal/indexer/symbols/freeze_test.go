package symbols

import (
	"slices"
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// declaration is one generated input of a Builder: a symbol under a
// module key, or a member under an owner.
type declaration struct {
	module, owner string
	node          *schema.Node
}

func feed(decls []declaration) View {
	b := NewBuilder()
	for _, d := range decls {
		b.AddNode(d.node)
		if d.owner != "" {
			b.AddMember(d.owner, d.node)
		} else {
			b.Add(d.module, d.node)
		}
	}
	return b.Freeze()
}

func ids(cands []Candidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.ID
	}
	slices.Sort(out)
	return out
}

// TestFreezeIgnoresInsertionOrder checks that two Builders fed one set of
// declarations in two orders freeze to Views that give each lookup the
// same answer: the same candidate set from CandidatesIn and Members, the
// same type from TypeIn, and the same collision count. CandidatesIn and
// Members keep the order of the input, so the property compares their
// answers as sets.
func TestFreezeIgnoresInsertionOrder(t *testing.T) {
	modules := []string{"example.com/a", "example.com/b"}
	owners := []string{"struct:A", "struct:B"}
	names := []string{"Run", "Stop", "Shape"}
	kinds := []string{goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface}
	answered := 0
	rapid.Check(t, func(t *rapid.T) {
		decls := make([]declaration, rapid.IntRange(1, 12).Draw(t, "decls"))
		for i := range decls {
			name := rapid.SampledFrom(names).Draw(t, "name")
			file := rapid.SampledFrom([]string{"a.go", "b.go"}).Draw(t, "file")
			decls[i] = declaration{
				module: rapid.SampledFrom(modules).Draw(t, "module"),
				node: &schema.Node{
					Id:            "n" + strconv.Itoa(rapid.IntRange(0, 15).Draw(t, "id")),
					Kind:          rapid.SampledFrom(kinds).Draw(t, "kind"),
					Name:          name,
					QualifiedName: name,
					FilePath:      file,
				},
			}
			if rapid.Bool().Draw(t, "member") {
				decls[i].owner = rapid.SampledFrom(owners).Draw(t, "owner")
			}
		}
		// One id names one declaration, as the extractors guarantee.
		seen := make(map[string]bool)
		decls = slices.DeleteFunc(decls, func(d declaration) bool {
			dup := seen[d.node.Id]
			seen[d.node.Id] = true
			return dup
		})
		order := rapid.Permutation(decls).Draw(t, "order")

		a, b := feed(decls), feed(order)
		if a.Collisions() != b.Collisions() {
			t.Fatalf("Collisions = %d and %d for two orders", a.Collisions(), b.Collisions())
		}
		for _, m := range modules {
			for _, n := range names {
				got, want := ids(b.CandidatesIn(m, n)), ids(a.CandidatesIn(m, n))
				if !slices.Equal(got, want) {
					t.Fatalf("CandidatesIn(%s, %s) = %v and %v for two orders", m, n, want, got)
				}
				ta, oka := a.TypeIn(m, n)
				tb, okb := b.TypeIn(m, n)
				if oka != okb || ta.ID != tb.ID {
					t.Fatalf("TypeIn(%s, %s) = %q, %v and %q, %v for two orders", m, n, ta.ID, oka, tb.ID, okb)
				}
				if len(want) > 0 {
					answered++
				}
			}
		}
		for _, o := range owners {
			for _, n := range names {
				got, want := ids(b.Members(o, n)), ids(a.Members(o, n))
				if !slices.Equal(got, want) {
					t.Fatalf("Members(%s, %s) = %v and %v for two orders", o, n, want, got)
				}
				if len(want) > 0 {
					answered++
				}
			}
		}
	})
	if answered == 0 {
		t.Fatal("no lookup found a declaration in any run")
	}
}
