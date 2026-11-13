package resolve

import (
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestDataReferenceMetadata covers the merge of references edges that
// share a source, kind and target: the stored edge carries the access
// role of every candidate, whatever order extraction gave them in, and
// no candidate is modified.
func TestDataReferenceMetadata(t *testing.T) {
	ref := func(line int32, access string) *schema.Edge {
		e := &schema.Edge{Source: "f", Target: "v", Kind: goextract.RefKindReferences, Line: line}
		if access != "" {
			e.Metadata = map[string]string{"access": access}
		}
		return e
	}
	cases := []struct {
		name  string
		roles []string
		want  string
	}{
		{"read only", []string{"read", "read"}, "read"},
		{"write only", []string{"write"}, "write"},
		{"read and write", []string{"read", "write"}, "read_write"},
		{"compound and read", []string{"read_write", "read"}, "read_write"},
		{"unknown and write", []string{"", "write"}, "write"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var edges []*schema.Edge
			for i, role := range tc.roles {
				edges = append(edges, ref(int32(i+1), role))
			}
			reversed := slices.Clone(edges)
			slices.Reverse(reversed)
			for _, order := range [][]*schema.Edge{edges, reversed} {
				in := slices.Clone(order)
				got := CollapseEdges(in)
				if len(got) != 1 {
					t.Fatalf("CollapseEdges kept %d edges, want 1", len(got))
				}
				if got[0].Metadata["access"] != tc.want {
					t.Errorf("access = %q, want %q", got[0].Metadata["access"], tc.want)
				}
				if got[0].Line != 1 {
					t.Errorf("representative line = %d, want the first site, 1", got[0].Line)
				}
			}
			for i, e := range edges {
				if e.Metadata["access"] != tc.roles[i] {
					t.Errorf("candidate %d access changed to %q", i, e.Metadata["access"])
				}
			}
		})
	}
}
