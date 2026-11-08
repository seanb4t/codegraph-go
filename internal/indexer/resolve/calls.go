package resolve

import (
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// CallOutput is what Calls resolves in one file. It keeps the calls
// without a deferred type apart from the calls with one, because Link
// orders the two lists apart (design D6).
type CallOutput struct {
	relPath string
	// edges and pending are the calls without a deferred type that
	// resolved and that did not, in reference order.
	edges   []*schema.Edge
	pending []pendingRef
	// deferredEdges and deferredPending are the calls with a deferred
	// type, in end-position order.
	deferredEdges   []*schema.Edge
	deferredPending []pendingRef
	resolved        int
}

// pendingRef is a call that did not resolve, kept for the conformance
// retry of Link, with the name its file waits on when the retry fails
// too, or nil.
type pendingRef struct {
	ref  goextract.UnresolvedRef
	wait *schema.UnresolvedName
}

// Calls resolves the calls of r against view and h, against candidate
// sets (design D3). The calls with a deferred type go last, once the
// calls they depend on are resolved (D17): a call that ends earlier never
// waits on one that ends later, because the inner link of a chain, or the
// call that initializes a local, ends before the call that reads its
// type. An undecidable call reaches its overload set (D14). Calls depends
// only on r, view and h, and it starts no goroutine.
func Calls(view symbols.View, h symbols.Hierarchy, r goextract.FileResult) CallOutput {
	rs := newResolver(view, h)
	out := CallOutput{relPath: r.RelPath}
	var deferred []goextract.UnresolvedRef
	resolveOne := func(ref goextract.UnresolvedRef, allowDeferred bool, edges *[]*schema.Edge, pending *[]pendingRef) {
		o := rs.resolveCall(r, ref, allowDeferred)
		switch {
		case o.deferred:
			deferred = append(deferred, ref)
		case o.target != "":
			*edges = append(*edges, callEdge(ref, o.target, false))
			rs.recordTarget(r.RelPath, ref, o.target)
			out.resolved++
		case o.set != nil:
			*edges = append(*edges, callEdge(ref, o.set.Node().Id, true))
			rs.recordTarget(r.RelPath, ref, o.set.Node().Id)
			out.resolved++
		default:
			// Not counted as unresolved yet (Pitfall 3): the call may still
			// resolve through the conformance retry.
			var w *schema.UnresolvedName
			if o.waitName != "" {
				w = &schema.UnresolvedName{ModuleKey: o.waitModule, Name: o.waitName}
			}
			*pending = append(*pending, pendingRef{ref: ref, wait: w})
		}
	}
	for _, ref := range r.Unresolved {
		if ref.Kind == goextract.RefKindCalls {
			resolveOne(ref, false, &out.edges, &out.pending)
		}
	}
	sort.SliceStable(deferred, func(i, j int) bool {
		a, b := deferred[i], deferred[j]
		if a.EndLine != b.EndLine {
			return a.EndLine < b.EndLine
		}
		return a.EndCol < b.EndCol
	})
	for _, ref := range deferred {
		resolveOne(ref, true, &out.deferredEdges, &out.deferredPending)
	}
	return out
}
