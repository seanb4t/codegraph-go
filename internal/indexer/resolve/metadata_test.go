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

// TestDataReferenceMetadata_CallEvidence covers the merge of calls edges
// with one source and target: a direct call and a call through a data
// symbol keep one edge marked direct, with every binding path and context
// of the derived candidates, whatever their order.
func TestDataReferenceMetadata_CallEvidence(t *testing.T) {
	call := func(line int32, via, ctx string) *schema.Edge {
		e := &schema.Edge{Source: "f", Target: "fn", Kind: goextract.RefKindCalls, Line: line}
		if via != "" {
			e.Metadata = map[string]string{goextract.MetaVia: via, goextract.MetaContext: ctx}
		}
		return e
	}
	cases := []struct {
		name             string
		edges            []*schema.Edge
		via, ctx, direct string
	}{
		{"direct only", []*schema.Edge{call(1, "", ""), call(2, "", "")}, "", "", ""},
		{"direct and derived", []*schema.Edge{call(1, "", ""), call(2, "v", ContextGoProduction)}, "v", ContextGoProduction, "true"},
		{"two paths", []*schema.Edge{call(1, "w", ContextGoTest), call(2, "v", ContextGoProduction)}, "v,w", "go-production,go-test", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reversed := slices.Clone(tc.edges)
			slices.Reverse(reversed)
			for _, order := range [][]*schema.Edge{tc.edges, reversed} {
				got := CollapseEdges(slices.Clone(order))
				if len(got) != 1 {
					t.Fatalf("CollapseEdges kept %d edges, want 1", len(got))
				}
				md := got[0].Metadata
				if md[goextract.MetaVia] != tc.via || md[goextract.MetaContext] != tc.ctx || md[goextract.MetaDirect] != tc.direct {
					t.Errorf("metadata = %v, want via %q, context %q, direct %q", md, tc.via, tc.ctx, tc.direct)
				}
			}
		})
	}
}
