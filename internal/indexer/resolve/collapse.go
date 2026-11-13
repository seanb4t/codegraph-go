package resolve

import (
	"sort"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// EdgeTriple is the (source, kind, target) identity a duplicate edge
// collapses on (D-05) — mirrors graphstore/keys.go's edgeKey shape exactly,
// without needing to import that package's unexported key-building code.
type EdgeTriple struct {
	Source, Kind, Target string
}

// EdgeOwnerID returns the node whose file owns e in the x/ file index:
// the method for a contains edge, since resolving the method's file makes
// the edge even when its type is declared in another file, and the source
// for every other edge.
func EdgeOwnerID(e *schema.Edge) string {
	if e.Kind == "contains" {
		return e.Target
	}
	return e.Source
}

// CollapseEdges aggregates every candidate edge sharing a (source, kind,
// target) triple and chooses ONE representative: the first candidate by
// (line, col) (RESEARCH Pitfall 1). The candidates of one triple share
// their source, and so its file, so the file is no part of the order. Two
// candidates that tie on both keep the one that came first in edges, so
// the order of edges is part of the result. The returned slice is in
// sorted triple order, so staging it via PutEdge is also deterministic.
func CollapseEdges(edges []*schema.Edge) []*schema.Edge {
	groups := make(map[EdgeTriple][]*schema.Edge, len(edges))
	for _, e := range edges {
		k := EdgeTriple{e.Source, e.Kind, e.Target}
		groups[k] = append(groups[k], e)
	}

	keys := make([]EdgeTriple, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Source != keys[j].Source {
			return keys[i].Source < keys[j].Source
		}
		if keys[i].Kind != keys[j].Kind {
			return keys[i].Kind < keys[j].Kind
		}
		return keys[i].Target < keys[j].Target
	})

	collapsed := make([]*schema.Edge, 0, len(keys))
	for _, k := range keys {
		candidates := groups[k]
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := candidates[i], candidates[j]
			if ci.Line != cj.Line {
				return ci.Line < cj.Line
			}
			return ci.Col < cj.Col
		})
		collapsed = append(collapsed, withMergedAccess(candidates))
	}
	return collapsed
}

// withMergedAccess returns the representative of candidates, the first,
// carrying the access role of all of them: an edge that one candidate
// reads through and another writes through is read_write. The
// representative is copied when its role changes, so no candidate is
// modified.
func withMergedAccess(candidates []*schema.Edge) *schema.Edge {
	rep := candidates[0]
	access := rep.Metadata["access"]
	for _, c := range candidates[1:] {
		access = goextract.MergeAccess(access, c.Metadata["access"])
	}
	if access == rep.Metadata["access"] {
		return rep
	}
	merged := proto.Clone(rep).(*schema.Edge)
	if merged.Metadata == nil {
		merged.Metadata = map[string]string{}
	}
	merged.Metadata["access"] = access
	return merged
}
