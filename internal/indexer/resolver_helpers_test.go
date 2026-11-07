package indexer

import "github.com/seanb4t/codegraph-go/internal/indexer/symbols"

// newTestResolver freezes b and returns a resolver over it, with the
// hierarchy of b's declarations and no type-level edges.
func newTestResolver(b *symbols.Builder) *resolver {
	view := b.Freeze()
	return &resolver{view: view, h: symbols.NewHierarchy(view, nil), callTargets: make(map[callPos]string)}
}
