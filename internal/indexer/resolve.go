// resolve.go runs pass 2 of the two-pass indexing pipeline (D-04)
// through the symbols and resolve layers, and commits the whole resolved
// graph through exactly one batched GraphStore.Writer (D-04a).
package indexer

import (
	"sort"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
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
func resolveGraph(results []goextract.FileResult, modulePath string, view symbols.View) resolve.Graph {
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

// resolveRefs resolves results alone, as a full index does.
func resolveRefs(results []goextract.FileResult, modulePath string) resolve.Graph {
	b := symbols.NewBuilder()
	b.AddResults(results)
	return resolveGraph(results, modulePath, b.Freeze())
}

// Resolve runs the whole of Pass 2: builds the global symbol index,
// resolves every file's Unresolved references into edges, deterministically
// collapses duplicates, and commits the resolved graph through exactly one
// batched GraphStore.Writer (D-04a). It returns the count of references
// that could not be resolved (surfaced later via --verbose, never silently
// dropped — D-06a).
//
// commitSHA is the git commit HEAD pointed at when the caller's operation
// began (ENG-04, D-05), resolved exactly once by run (the full index run's
// entry point) and threaded down here rather than re-resolved per call —
// HEAD can move between two resolutions within one run (a rebase, a
// checkout, a concurrent commit), and two write sites in the same
// operation recording different commits would be worse than recording
// none. An empty commitSHA is a legitimate value (non-git checkout, or git
// unavailable) and is stamped as-is; schema.IndexedCommitSHA treats it as
// absent.
//
// excluded (Phase 10 D-01/D-07) is DiscoverAll's exclusion-reason list,
// threaded through unchanged into writeGraph's SAME commit batch.
//
// coverageGenerationFloor (CR-01, 10-REVIEW.md iteration 3) is threaded
// through unchanged to writeGraph — see Options.CoverageGenerationFloor's
// doc comment (pipeline.go) for why a caller that wipes storeDir before
// calling Run needs this lower bound.
func Resolve(store graphstore.GraphStore, results []goextract.FileResult, modulePath string, commitSHA string, excluded []*schema.ExcludedFile, coverageGenerationFloor int64) (int, error) {
	g := resolveRefs(results, modulePath)
	if err := writeGraph(store, g.Nodes, g.PackageNodes, g.Edges, g.Files, commitSHA, excluded, coverageGenerationFloor); err != nil {
		return g.Unresolved, err
	}
	return g.Unresolved, nil
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

// writeGraph collapses edges deterministically and stages the whole
// resolved graph — package pseudo-nodes and symbol nodes (sorted by id),
// then files (sorted by path), then collapsed edges (sorted by
// source/kind/target), then excluded-file records (sorted by path,
// Phase 10 D-07) — through exactly one GraphStore.Writer, committing once
// (D-04a). Any staging error releases the batch via Close() (never a
// partial Commit) and returns the error.
//
// coverageGenerationFloor (CR-01, 10-REVIEW.md iteration 3) is a lower
// bound on the generation this commit stamps — see
// Options.CoverageGenerationFloor's doc comment (pipeline.go) for the
// full rationale. Zero is a no-op: the store's own prior-generation read
// below is used unchanged.
func writeGraph(store graphstore.GraphStore, nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, commitSHA string, excluded []*schema.ExcludedFile, coverageGenerationFloor int64) error {
	nodeFilePath := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeFilePath[n.Id] = n.FilePath
	}

	collapsedEdges := resolve.CollapseEdges(edges, nodeFilePath)

	// WR-01 (iteration 2): read-modify-write the prior CoverageGeneration
	// via a snapshot Reader — never a second Writer — so a from-scratch
	// rewrite over an EXISTING store (Sync's D-02b backfill path delegates
	// here too) still increments monotonically rather than resetting to 1
	// every time. A genuinely fresh store (no Meta record yet) has no
	// prior generation to read, so it starts at 0 and this commit stamps
	// 1, mirroring needsFileIndexBackfill's own ErrNotFound-means-fresh
	// handling.
	var priorGeneration int64
	r, snapErr := store.Snapshot()
	if snapErr != nil {
		return snapErr
	}
	priorMeta, metaErr := r.GetMeta()
	r.Close()
	if metaErr != nil && metaErr != graphstore.ErrNotFound {
		return metaErr
	}
	if metaErr == nil {
		priorGeneration = priorMeta.GetCoverageGeneration()
	}

	var size graphstore.BatchSize
	for _, n := range nodes {
		size.Node(n)
	}
	for _, n := range packageNodes {
		size.Node(n)
	}
	for _, f := range files {
		size.File(f)
	}
	for _, e := range collapsedEdges {
		size.Edge(e, nodeFilePath[resolve.EdgeOwnerID(e)])
	}
	for _, x := range excluded {
		size.ExcludedFile(x)
	}
	w, err := store.NewWriterSized(size.Bytes())
	if err != nil {
		return err
	}

	// Sync's backfill path (needsFileIndexBackfill) rewrites an existing
	// store through here. Delete the old graph first, in this same batch,
	// so that records the new graph does not have, such as the nodes of a
	// file deleted while no watcher ran, do not survive the rewrite, and a
	// reader sees the old graph or the new one, never a mix.
	if err := w.DeleteGraph(); err != nil {
		w.Close()
		return err
	}

	staged := resolve.Graph{Nodes: nodes, PackageNodes: packageNodes, Files: files, Edges: collapsedEdges}
	nodeCount, err := stage(w, staged, func(e *schema.Edge) string { return nodeFilePath[resolve.EdgeOwnerID(e)] })
	if err != nil {
		w.Close()
		return err
	}

	if err := w.DeleteAllExcludedFiles(); err != nil {
		w.Close()
		return err
	}

	// Phase 10 D-07: excluded-file records are staged on this SAME
	// Writer, in the SAME batch as every other record kind (T-10-10) —
	// never a second commit. Sorted by path, mirroring sortedFiles' own
	// determinism discipline just above.
	sortedExcluded := make([]*schema.ExcludedFile, len(excluded))
	copy(sortedExcluded, excluded)
	sort.Slice(sortedExcluded, func(i, j int) bool { return sortedExcluded[i].GetPath() < sortedExcluded[j].GetPath() })
	for _, x := range sortedExcluded {
		if err := w.PutExcludedFile(x); err != nil {
			w.Close()
			return err
		}
	}

	meta := schema.NewMeta()
	meta.NodeCount = int64(nodeCount)
	meta.EdgeCount = int64(len(collapsedEdges))
	stampCapabilities(meta)
	// Phase 10 D-07: every from-scratch writeGraph run (Run's own path,
	// and Sync's D-02b backfill path which delegates to run()) stages
	// the complete exclusion-reason set above — genuinely HAS coverage
	// recorded by the time this Commit lands, regardless of caller.
	// Stamping the flag here mirrors HasFileIndex's own precedent: only a
	// GENUINELY pre-Phase-10 store is ever missing it (D-06).
	meta.HasCoverage = true
	// WR-01 (iteration 2): stamp the read-modify-written generation
	// computed above — see this function's own comment at the Snapshot
	// read for why a from-scratch rewrite still increments monotonically
	// rather than resetting.
	//
	// CR-01 (iteration 3): priorGeneration alone is only the TARGET
	// store's own history — genuinely 0 for a store that was just wiped
	// (RemoveAll+MkdirAll, internal/cli/index.go), even if the store that
	// USED to live at that path had climbed much higher. A caller that
	// read the prior store's generation before wiping it passes that
	// value as coverageGenerationFloor, so the stamped generation never
	// regresses below a value a live client's page token may already
	// carry. A caller with nothing to float against (a genuinely fresh
	// `codegraph init`, or Sync's own backfill/incremental paths, which
	// never wipe) passes 0, leaving this identical to iteration 2's
	// priorGeneration+1.
	generation := priorGeneration
	if coverageGenerationFloor > generation {
		generation = coverageGenerationFloor
	}
	meta.CoverageGeneration = generation + 1
	// Phase 4 D-04a: stamp LastSyncUnixMs here too, not just in Sync's own
	// meta-write step (internal/indexer/sync.go) — otherwise a graph built
	// via a from-scratch `codegraph index` (this path) carries a zero
	// last_sync_unix_ms forever until the first incremental sync runs,
	// which would make query.Engine.Status's newest-mtime-vs-last_sync
	// staleness fallback (04-06) report every freshly indexed repo as
	// permanently stale.
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	// ENG-04/D-05: stamp the commit HEAD pointed at when this operation
	// began, resolved once by run and threaded down — for the same stated
	// reason HasFileIndex and LastSyncUnixMs are stamped here rather than
	// only in Sync's own meta-write steps: stamping at only one site
	// leaves graphs built through the other paths permanently missing it.
	meta.CommitSha = commitSHA
	if err := w.PutMeta(meta); err != nil {
		w.Close()
		return err
	}

	return w.Commit()
}
