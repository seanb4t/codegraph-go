// resolve.go holds the resolution that a sync runs over the files it
// reparses, through the symbols and resolve layers. A full index runs the
// same layers in three passes (fullindex.go).
package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// resolveGraph resolves results against view: the type-level references
// of every file, then the hierarchy those make, then the calls of every
// file, then the whole-graph passes of resolve.Link. Run passes a view
// of results alone, and Sync a view of results over what loadStoreView
// read from the files it does not reparse, so a reference into an
// unchanged file still resolves (RESEARCH Pitfall 1), and the kind
// lookups and the implements, override and conformance-retry passes see
// the whole graph as a fresh index does.
func resolveGraph(results []goextract.FileResult, modulePath string, view symbols.View) (resolve.Graph, error) {
	types := make([]resolve.TypeOutput, len(results))
	var typeEdges []*schema.Edge
	for i := range results {
		types[i] = resolve.Types(view, modulePath, results[i])
		typeEdges = append(typeEdges, types[i].Edges()...)
	}
	h := symbols.NewHierarchy(view, typeEdges)
	calls := make([]resolve.CallOutput, len(results))
	for i := range results {
		calls[i] = resolve.Calls(view, h, results[i])
	}
	return resolve.Link(view, h, types, calls)
}

// declarationsOf returns the declarations of each result, with the default
// visibility of its language.
func declarationsOf(results []goextract.FileResult) []symbols.Declarations {
	decls := make([]symbols.Declarations, len(results))
	for i, r := range results {
		decls[i] = declarationOf(r)
	}
	return decls
}

// stampCapabilities sets every Meta flag whose absence makes
// needsFileIndexBackfill rebuild the graph. Every writer of Meta calls it,
// because each one builds a fresh Meta: a flag that only one writer set
// would drop on the next incremental sync (design D5). A new flag goes here and into
// needsFileIndexBackfill.
func stampCapabilities(meta *schema.Meta) {
	// Phase 4 D-02b: every PutNode/PutEdge a Writer stages populates the
	// x/ file-owned secondary index.
	meta.HasFileIndex = true
	// resolveRefsWithIndex fills every File record's unresolved_names.
	meta.HasUnresolvedNames = true
	// resolveRefsWithIndex fills declared_module_key wherever a file
	// declares its module.
	meta.HasDeclaredModuleKeys = true
	// resolveRefsWithIndex stamps arity and interface_methods on every
	// node.
	meta.HasMethodSpecs = true
	// The Go extractor emits a node for every variable of a grouped var
	// block.
	meta.HasGroupedVars = true
	// Each declaration has its own node: nodeid.Allocator gives the
	// second and later copies of a repeated declaration their own ids.
	meta.HasUniqueDeclIds = true
}
