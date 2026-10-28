package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Sync performs an incremental update of the graph at storeDir against the
// current on-disk state of repoRoot (INDX-03): only content-hash-changed
// files, plus their direct call-graph dependents and the files waiting on
// a symbol this sync adds, are reparsed;
// changed/deleted files' subgraphs are pruned via the x/ file-owned
// secondary index (D-02); and the whole prune+write lands in exactly ONE
// atomic Writer commit (D-01b — never per-symbol). One routine, three
// callers: `codegraph sync`, the debounced daemon cycle, and MCP-reconnect
// reconcile — see 04-RESEARCH.md Pattern 1 for the algorithm this
// implements.
//
// Sync mirrors Run's open-once/Close-on-every-path lifecycle (D-04), but
// performs a store-seeded incremental resolve instead of a from-scratch
// one: the global symbol index is seeded from the store's own committed
// nodes (loadStoreView's symbols), not just the reparse batch, so a
// reference into an unchanged file still resolves (RESEARCH Pitfall 1).
func Sync(repoRoot, storeDir string, opts Options) (Stats, error) {
	start := time.Now()

	backfill, err := needsFileIndexBackfill(storeDir)
	if err != nil {
		return Stats{}, err
	}
	if backfill {
		// D-02b: a pre-Phase-4 graph (or a brand-new store) has no x/
		// index yet. Delegate to the exact from-scratch code path Run
		// uses — writeGraph unconditionally stamps Meta.HasFileIndex=true
		// (every PutNode/PutEdge it stages already populates the x/
		// namespace, 04-01) — so every subsequent Sync is incremental.
		stats, err := run(repoRoot, storeDir, opts, Resolve)
		stats.FilesReparsed = stats.Files
		return stats, err
	}

	// ENG-04/D-05: resolve HEAD exactly once for this Sync invocation,
	// before either of the two write sites below construct Meta. HEAD can
	// move mid-run (a rebase, a checkout, a concurrent commit), and this
	// call's two write sites are mutually exclusive per invocation (the
	// mtime-refresh-only early return below, or the incremental path
	// further down) — so resolving once here, rather than at each site,
	// both satisfies "resolved once per operation" and avoids adding a
	// subprocess to the mtime-refresh-only path, which otherwise does no
	// process work at all.
	headCommitSHA := resolveHeadCommitSHA(repoRoot)

	store, err := graphstore.Open(storeDir)
	if err != nil {
		return Stats{}, err
	}
	defer store.Close()

	r0, err := store.Snapshot()
	if err != nil {
		return Stats{}, err
	}
	defer r0.Close()

	meta, err := r0.GetMeta()
	if err != nil {
		return Stats{}, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Phase 10 D-07: DiscoverAll's exclusion-reason list (discovery.Excluded)
	// feeds the c/ namespace diff below (loadStoredExclusions +
	// diffExclusions), the same walk result Files/ModulePath come from.
	discovery, err := DiscoverAll(repoRoot)
	if err != nil {
		return Stats{}, err
	}
	files, modulePath := discovery.Files, discovery.ModulePath

	discovered := make(map[string]DiscoveredFile, len(files))
	for _, f := range files {
		discovered[f.RelPath] = f
	}

	// Stat pre-filter -> content-hash confirm (D-01a).
	var added, modified []DiscoveredFile
	// mtimeRefresh (WR-03) collects files whose stat metadata (mtime/size)
	// differs from the stored File record but whose recomputed content
	// hash still matches — a touch, a git checkout with no real content
	// change, or a mtime-only editor save. These are NOT reparsed (their
	// content is unchanged), but the stored File record's stale
	// mtime/size must still be refreshed to the current on-disk values;
	// otherwise every subsequent Sync re-fails this cheap stat check for
	// the same file, forever, and pays the full contentHash read+hash cost
	// again each time.
	var mtimeRefresh []*schema.File
	for _, f := range files {
		stored, err := r0.GetFile(f.RelPath)
		if err != nil {
			if err != graphstore.ErrNotFound {
				return Stats{}, err
			}
			added = append(added, f)
			continue
		}
		if stored.GetMtimeUnixNs() == f.MtimeUnixNs && stored.GetSizeBytes() == f.SizeBytes {
			continue
		}
		hash, err := contentHash(f.AbsPath)
		if err != nil {
			return Stats{}, err
		}
		if hash == stored.GetContentHash() {
			refreshed := proto.Clone(stored).(*schema.File)
			refreshed.MtimeUnixNs = f.MtimeUnixNs
			refreshed.SizeBytes = f.SizeBytes
			mtimeRefresh = append(mtimeRefresh, refreshed)
			continue
		}
		modified = append(modified, f)
	}

	// Deleted: a stored File whose path is no longer discovered.
	var deleted []string
	fit, err := r0.IterateFiles()
	if err != nil {
		return Stats{}, err
	}
	for fit.Next() {
		p := fit.File().GetPath()
		if _, ok := discovered[p]; !ok {
			deleted = append(deleted, p)
		}
	}
	if ferr := fit.Err(); ferr != nil {
		fit.Close()
		return Stats{}, ferr
	}
	fit.Close()

	if changed := len(added) + len(modified) + len(deleted); opts.MaxChanged > 0 && changed > opts.MaxChanged {
		return Stats{}, &ErrTooManyChanges{Changed: changed, Total: len(files)}
	}

	// Phase 10 D-07: diff the c/ namespace exactly like the `deleted`
	// diff above compares stored File records against the freshly
	// discovered file set — upsert what is absent/changed, prune what is
	// gone. coverageDirty extends the no-op gate below so an
	// exclusion-only change (a new excluded file appears; an excluded
	// file disappears) is never mistaken for "nothing changed", and so a
	// pre-Phase-10 graph (has_coverage unset) is backfilled by the very
	// next Sync even when no file on disk changed at all.
	storedExcl, err := loadStoredExclusions(r0)
	if err != nil {
		return Stats{}, err
	}
	exclUpserts, exclPrunes := diffExclusions(storedExcl, discovery.Excluded)
	coverageDirty := len(exclUpserts) > 0 || len(exclPrunes) > 0 || !meta.GetHasCoverage()

	if len(added) == 0 && len(modified) == 0 && len(deleted) == 0 {
		if len(mtimeRefresh) == 0 && !coverageDirty {
			// Fully no-op sync: nothing changed on disk AND coverage is
			// already recorded.
			return Stats{
				Files:    len(files),
				Nodes:    int(meta.GetNodeCount()),
				Edges:    int(meta.GetEdgeCount()),
				Duration: time.Since(start),
			}, nil
		}
		// WR-03 / Phase 10 D-07: no content actually changed, but either
		// at least one File record's stored mtime/size is stale relative
		// to disk, or the c/ namespace has upserts/prunes to stage, or
		// coverage was never recorded — persist the refresh/diff in its
		// own small commit so the stat pre-filter's fast path is
		// restored on the next Sync. NodeCount/EdgeCount are untouched
		// (nothing was reparsed or pruned).
		w, err := store.NewWriter()
		if err != nil {
			return Stats{}, err
		}
		for _, f := range mtimeRefresh {
			if err := w.PutFile(f); err != nil {
				w.Close()
				return Stats{}, err
			}
		}
		if err := stageExclusionDiff(w, exclUpserts, exclPrunes); err != nil {
			w.Close()
			return Stats{}, err
		}
		newMeta := schema.NewMeta()
		newMeta.HasFileIndex = true
		newMeta.HasUnresolvedNames = true
		newMeta.HasDeclaredModuleKeys = true
		newMeta.HasMethodSpecs = true
		newMeta.LastSyncUnixMs = time.Now().UnixMilli()
		newMeta.NodeCount = meta.GetNodeCount()
		newMeta.EdgeCount = meta.GetEdgeCount()
		newMeta.CommitSha = syncCommitSHA(headCommitSHA, meta)
		// Phase 10 D-07: every meta-write site stamps HasCoverage
		// independently (Pitfall 3) — this commit's batch always
		// includes the exclusion diff computed above (even an empty
		// one, when the only reason we are here is coverage never having
		// been recorded), so it genuinely HAS coverage by the time this
		// Commit lands.
		newMeta.HasCoverage = true
		// WR-01 (iteration 2): read-modify-write the prior generation from
		// the SAME meta this function already read (r0.GetMeta() at the
		// top of Sync), incrementing by exactly 1 — a monotonic counter
		// cannot alias within the same wall-clock millisecond the way
		// LastSyncUnixMs can.
		newMeta.CoverageGeneration = meta.GetCoverageGeneration() + 1
		if err := w.PutMeta(newMeta); err != nil {
			w.Close()
			return Stats{}, err
		}
		if err := w.Commit(); err != nil {
			return Stats{}, err
		}
		return Stats{
			Files:    len(files),
			Nodes:    int(newMeta.NodeCount),
			Edges:    int(newMeta.EdgeCount),
			Duration: time.Since(start),
		}, nil
	}

	w, err := store.NewWriter()
	if err != nil {
		return Stats{}, err
	}

	nodesRemoved := 0
	edgesRemoved := 0
	goneIDs := make(map[string]struct{})

	prunePaths := make([]string, 0, len(modified)+len(deleted))
	for _, f := range modified {
		prunePaths = append(prunePaths, f.RelPath)
	}
	prunePaths = append(prunePaths, deleted...)
	sort.Strings(prunePaths)

	for _, path := range prunePaths {
		if err := pruneFileSubgraph(r0, w, path, goneIDs, &nodesRemoved, &edgesRemoved); err != nil {
			w.Close()
			return Stats{}, err
		}
		if err := w.DeleteFileSubgraph(path); err != nil {
			w.Close()
			return Stats{}, err
		}
	}

	// Dependent detection (D-02a): a file with an edge targeting a
	// now-gone node id must be re-extracted so its Unresolved refs
	// regenerate against the current graph state (RESEARCH Pitfall 2).
	//
	// WR-05: changedSet also includes `deleted` (not just added/modified)
	// — a now-deleted file that used to call a pruned symbol would
	// otherwise be added to dependentPaths too, running through
	// pruneOwnedEdgesOnly redundantly (a harmless no-op against records
	// pruneFileSubgraph's DeleteFileSubgraph already range-deleted for
	// that same path above) before being silently dropped from
	// batchFiles anyway (it is absent from `discovered`, since it no
	// longer exists on disk) — unnecessary churn on every sync that
	// deletes a file with outgoing edges, not a correctness bug (the
	// redundant work is idempotent).
	changedSet := make(map[string]struct{}, len(added)+len(modified)+len(deleted))
	for _, f := range added {
		changedSet[f.RelPath] = struct{}{}
	}
	for _, f := range modified {
		changedSet[f.RelPath] = struct{}{}
	}
	for _, p := range deleted {
		changedSet[p] = struct{}{}
	}

	dependentPaths := make(map[string]struct{})
	if len(goneIDs) > 0 {
		rev, err := buildReverseAdjacency(r0)
		if err != nil {
			w.Close()
			return Stats{}, err
		}
		for id := range goneIDs {
			for _, edge := range rev[id] {
				srcNode, err := r0.GetNode(edge.Source)
				if err != nil {
					if err == graphstore.ErrNotFound {
						continue
					}
					w.Close()
					return Stats{}, err
				}
				path := srcNode.GetFilePath()
				if _, already := changedSet[path]; already {
					continue
				}
				dependentPaths[path] = struct{}{}
			}
		}
	}

	// A method's file owns its contains edge from a type declared in
	// another file, since resolving the method's file makes it. When this
	// sync removes the type's node, those files are dependents, so their
	// edges follow the type: remade if it still exists, dropped if not.
	for _, id := range sortedKeys(goneIDs) {
		methodFiles, err := containedMethodFiles(r0, id)
		if err != nil {
			w.Close()
			return Stats{}, err
		}
		for _, mf := range methodFiles {
			if _, already := changedSet[mf]; !already {
				dependentPaths[mf] = struct{}{}
			}
		}
	}

	// Narrow-prune dependents: only their owned OUTGOING edges — their
	// own nodes/File record stay (content unchanged, RESEARCH Pattern 1
	// step 8).
	for path := range dependentPaths {
		if err := pruneOwnedEdgesOnly(r0, w, path, &edgesRemoved); err != nil {
			w.Close()
			return Stats{}, err
		}
	}

	// Extract batch = added ∪ modified ∪ dependent.
	batchFiles := make([]DiscoveredFile, 0, len(added)+len(modified)+len(dependentPaths))
	batchFiles = append(batchFiles, added...)
	batchFiles = append(batchFiles, modified...)
	for path := range dependentPaths {
		if f, ok := discovered[path]; ok {
			batchFiles = append(batchFiles, f)
		}
	}
	sort.Slice(batchFiles, func(i, j int) bool { return batchFiles[i].RelPath < batchFiles[j].RelPath })

	results, err := Extract(batchFiles, workers)
	if err != nil {
		w.Close()
		return Stats{}, err
	}

	// A file whose reference did not resolve when it was last indexed is
	// re-extracted when this sync adds a symbol it names, so the
	// reference resolves now. Its owned edges are pruned like a
	// dependent's, before any edge below is staged.
	addedNames, err := newNames(r0, results)
	if err != nil {
		w.Close()
		return Stats{}, err
	}
	waiting, err := filesWaitingOn(r0, addedNames, changedSet, dependentPaths)
	if err != nil {
		w.Close()
		return Stats{}, err
	}
	if len(waiting) > 0 {
		waitingFiles := make([]DiscoveredFile, 0, len(waiting))
		for _, path := range waiting {
			if err := pruneOwnedEdgesOnly(r0, w, path, &edgesRemoved); err != nil {
				w.Close()
				return Stats{}, err
			}
			dependentPaths[path] = struct{}{}
			if f, ok := discovered[path]; ok {
				waitingFiles = append(waitingFiles, f)
			}
		}
		more, err := Extract(waitingFiles, workers)
		if err != nil {
			w.Close()
			return Stats{}, err
		}
		batchFiles = append(batchFiles, waitingFiles...)
		results = append(results, more...)
	}

	// The store-seeded index must exclude every path whose symbol
	// identity is being superseded or removed: the reparse batch (about
	// to be overlaid fresh) AND deleted files (never re-extracted, so a
	// deleted symbol must not remain resolvable via the stale store scan
	// — RESEARCH Pitfall 1's store-seeding must not leak pruned symbols).
	excludePaths := make(map[string]bool, len(batchFiles)+len(deleted))
	for _, f := range batchFiles {
		excludePaths[f.RelPath] = true
	}
	for _, p := range deleted {
		excludePaths[p] = true
	}

	view, err := loadStoreView(r0, discovered, excludePaths)
	if err != nil {
		w.Close()
		return Stats{}, err
	}
	view.symbols.overlay(results)

	nodes, packageNodes, edges, fileRecords, unresolvedCount := resolveRefsWithIndex(results, modulePath, view.symbols, view)

	nodeFilePath := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeFilePath[n.Id] = n.FilePath
	}
	collapsedEdges := collapseEdges(edges, nodeFilePath)

	// The resolve above synthesized implements and overrides over the
	// whole graph. A synthesized edge an unchanged file already stores is
	// skipped, and a stored one the synthesis did not produce is deleted,
	// so Meta.EdgeCount counts each change once.
	kept := make(map[edgeTriple]bool)
	writes := make([]*schema.Edge, 0, len(collapsedEdges))
	for _, e := range collapsedEdges {
		k := edgeTriple{e.Source, e.Kind, e.Target}
		if _, stored := view.synthesized[k]; stored && isSynthesizedEdge(e) {
			kept[k] = true
			continue
		}
		writes = append(writes, e)
	}
	collapsedEdges = writes
	for _, k := range sortedTriples(view.synthesized) {
		if kept[k] {
			continue
		}
		if err := deleteOwnedEdge(w, view.synthesized[k], k); err != nil {
			w.Close()
			return Stats{}, err
		}
		edgesRemoved++
	}

	// CR-03: an edge's owner (edgeOwnerID) is not always in THIS cycle's
	// reparse batch: a synthesized implements or override edge can start
	// at a node in an unchanged file. nodeFilePath alone would miss it,
	// silently writing the edge with ownerPath="" and permanently escaping
	// the x/ index (never enumerable by a later prune). ownerPathFor below
	// falls back to r0 — the pre-Sync snapshot, unaffected by this cycle's
	// staged-but-uncommitted writes — for exactly that case.

	allNodes := make([]*schema.Node, 0, len(nodes)+len(packageNodes))
	allNodes = append(allNodes, nodes...)
	allNodes = append(allNodes, packageNodes...)
	sort.Slice(allNodes, func(i, j int) bool { return allNodes[i].Id < allNodes[j].Id })

	sortedFiles := make([]*schema.File, len(fileRecords))
	copy(sortedFiles, fileRecords)
	sort.Slice(sortedFiles, func(i, j int) bool { return sortedFiles[i].Path < sortedFiles[j].Path })

	// nodesAdded counts only node ids that are GENUINELY new to the
	// committed graph after this Sync: either they were just pruned above
	// (goneIDs — a node id is stable across a content-unrelated edit
	// elsewhere in its file, D-02/nodeid.NodeID depends only on
	// (kind,qualifiedName,filePath), so "pruned then re-added with the
	// same id" nets to zero) or they never existed in the base graph at
	// all (a genuinely new symbol, or a dependent/package node getting
	// re-staged with an id that already existed and was NOT pruned — that
	// case must NOT be double-counted as an addition).
	nodesAdded := 0
	for _, n := range allNodes {
		if _, gone := goneIDs[n.Id]; gone {
			nodesAdded++
			continue
		}
		if _, err := r0.GetNode(n.Id); err != nil {
			if err != graphstore.ErrNotFound {
				w.Close()
				return Stats{}, err
			}
			nodesAdded++
		}
	}

	for _, n := range allNodes {
		if err := w.PutNode(n); err != nil {
			w.Close()
			return Stats{}, err
		}
	}
	for _, f := range sortedFiles {
		if err := w.PutFile(f); err != nil {
			w.Close()
			return Stats{}, err
		}
	}
	// WR-03: also persist the mtime/size-only refresh for any file whose
	// stat metadata was stale but content hash matched — these files are
	// not part of sortedFiles (they were never reparsed), so they need
	// their own PutFile pass here.
	for _, f := range mtimeRefresh {
		if err := w.PutFile(f); err != nil {
			w.Close()
			return Stats{}, err
		}
	}
	for _, e := range collapsedEdges {
		if err := w.PutEdge(e, ownerPathFor(edgeOwnerID(e), nodeFilePath, r0)); err != nil {
			w.Close()
			return Stats{}, err
		}
	}

	if err := stageExclusionDiff(w, exclUpserts, exclPrunes); err != nil {
		w.Close()
		return Stats{}, err
	}

	newMeta := schema.NewMeta()
	newMeta.HasFileIndex = true
	newMeta.HasUnresolvedNames = true
	newMeta.HasDeclaredModuleKeys = true
	newMeta.HasMethodSpecs = true
	newMeta.LastSyncUnixMs = time.Now().UnixMilli()
	newMeta.NodeCount = meta.GetNodeCount() - int64(nodesRemoved) + int64(nodesAdded)
	newMeta.EdgeCount = meta.GetEdgeCount() - int64(edgesRemoved) + int64(len(collapsedEdges))
	newMeta.CommitSha = syncCommitSHA(headCommitSHA, meta)
	// Phase 10 D-07: every meta-write site stamps HasCoverage
	// independently (Pitfall 3) — this commit's batch always stages the
	// exclusion diff computed above (via stageExclusionDiff, just before
	// this write), so it genuinely HAS coverage by the time this Commit
	// lands, mirroring HasFileIndex's own precedent (D-06).
	newMeta.HasCoverage = true
	// WR-01 (iteration 2): read-modify-write the prior generation from the
	// SAME meta this function already read (r0.GetMeta() at the top of
	// Sync), incrementing by exactly 1 — see the sibling small-commit
	// write site above for the full rationale.
	newMeta.CoverageGeneration = meta.GetCoverageGeneration() + 1
	if err := w.PutMeta(newMeta); err != nil {
		w.Close()
		return Stats{}, err
	}

	if err := w.Commit(); err != nil {
		return Stats{}, err
	}

	skipped := 0
	for _, r := range results {
		if r.Err != nil {
			skipped++
		}
	}

	return Stats{
		Files:                len(files),
		Nodes:                int(newMeta.NodeCount),
		Edges:                int(newMeta.EdgeCount),
		Unresolved:           unresolvedCount,
		Skipped:              skipped,
		FilesReparsed:        len(batchFiles),
		FilesPruned:          len(modified) + len(deleted),
		NodesRemoved:         nodesRemoved,
		EdgesRemoved:         edgesRemoved,
		DependentsRecomputed: len(dependentPaths),
		Duration:             time.Since(start),
	}, nil
}

// buildReverseAdjacency builds an in-memory reverse-adjacency map keyed by
// edge.Target, filtered to goextract.RefKindCalls only — a package-local
// duplicate of internal/query.BuildReverseAdjacency's exact algorithm
// (D-02a), not a design preference: internal/query's own white-box test
// files (engine_test.go et al.) import internal/indexer to seed real
// fixtures via indexer.Run, so internal/indexer importing internal/query
// directly creates a real, Go-toolchain-enforced "import cycle not
// allowed in test" for internal/query's own test binary (verified via `go
// vet ./...`) — contradicting 04-RESEARCH.md/04-02-SUMMARY.md's claim of
// a safe, non-circular indexer->query edge, which checked query's
// PRODUCTION imports of indexer/goextract but not query's TEST files'
// import of indexer itself. query.BuildReverseAdjacency remains exported
// and is still the implementation internal/query's own Callers/Impact/
// Affected use; this is Sync's own copy, built fresh per invocation
// (never cached), matching that function's same discipline.
func buildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	rev := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		rev[e.Target] = append(rev[e.Target], e)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return rev, nil
}

// needsFileIndexBackfill reports whether the store at storeDir has no
// Meta record yet, or a Meta record whose has_file_index,
// has_unresolved_names, has_declared_module_keys or has_method_specs flag
// is false (D-02b) — opening and closing the
// store in isolation from Sync's own main open, so the two never race a
// Pebble lock against each other.
func needsFileIndexBackfill(storeDir string) (bool, error) {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return false, err
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		return false, err
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		if err == graphstore.ErrNotFound {
			return true, nil
		}
		return false, err
	}
	return !meta.GetHasFileIndex() || !meta.GetHasUnresolvedNames() || !meta.GetHasDeclaredModuleKeys() || !meta.GetHasMethodSpecs(), nil
}

// containedMethodFiles returns, sorted, the files of the methods that
// typeID's contains edges in r0 point to.
func containedMethodFiles(r0 graphstore.Reader, typeID string) ([]string, error) {
	it, err := r0.IterateEdges(typeID)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	seen := make(map[string]struct{})
	for it.Next() {
		e := it.Edge()
		if e.Kind != "contains" {
			continue
		}
		n, err := r0.GetNode(e.Target)
		if err != nil {
			if err == graphstore.ErrNotFound {
				continue
			}
			return nil, err
		}
		seen[n.GetFilePath()] = struct{}{}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return sortedKeys(seen), nil
}

// sortedKeys returns m's keys in order.
func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// deleteOwnedEdge stages the delete of edge k and of owner's file-index
// entry for it.
func deleteOwnedEdge(w graphstore.Writer, owner string, k edgeTriple) error {
	if err := w.DeleteEdge(k.source, k.kind, k.target); err != nil {
		return err
	}
	return w.DeleteFileIndexEdge(owner, k.source, k.kind, k.target)
}

// sortedTriples returns m's keys sorted by (source, kind, target).
func sortedTriples(m map[edgeTriple]string) []edgeTriple {
	keys := make([]edgeTriple, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.source != b.source {
			return a.source < b.source
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		return a.target < b.target
	})
	return keys
}

// unresolvedKey is a (module key, name) pair as schema.UnresolvedName
// holds it, usable as a map key.
type unresolvedKey struct {
	moduleKey, name string
}

// newNames returns the (module key, name) pairs results declare under a
// node id r0 does not hold: the symbols this sync adds.
func newNames(r0 graphstore.Reader, results []goextract.FileResult) (map[unresolvedKey]struct{}, error) {
	names := make(map[unresolvedKey]struct{})
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		for _, en := range r.Nodes {
			if en.Node.Kind == goextract.KindFile {
				continue
			}
			_, err := r0.GetNode(en.Node.Id)
			if err == nil {
				continue
			}
			if err != graphstore.ErrNotFound {
				return nil, err
			}
			names[unresolvedKey{moduleKey: r.ImportPath, name: en.Node.Name}] = struct{}{}
		}
	}
	return names, nil
}

// filesWaitingOn returns, sorted, the paths of r0's files that are in
// neither changed nor dependents and whose unresolved names include one
// of names.
func filesWaitingOn(r0 graphstore.Reader, names map[unresolvedKey]struct{}, changed, dependents map[string]struct{}) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	it, err := r0.IterateFiles()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var paths []string
	for it.Next() {
		f := it.File()
		path := f.GetPath()
		if _, ok := changed[path]; ok {
			continue
		}
		if _, ok := dependents[path]; ok {
			continue
		}
		for _, u := range f.GetUnresolvedNames() {
			if _, ok := names[unresolvedKey{moduleKey: u.GetModuleKey(), name: u.GetName()}]; ok {
				paths = append(paths, path)
				break
			}
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// pruneFileSubgraph enumerates path's owned x/ entries in r0 and stages a
// DeleteNode/DeleteEdge on w for each — the mandatory
// IterateFileIndex-before-DeleteFileSubgraph ordering 04-01's Writer
// contract documents. Every deleted node id is recorded into goneIDs for
// the caller's dependent-detection step (D-02a).
func pruneFileSubgraph(r0 graphstore.Reader, w graphstore.Writer, path string, goneIDs map[string]struct{}, nodesRemoved, edgesRemoved *int) error {
	it, err := r0.IterateFileIndex(path)
	if err != nil {
		return err
	}
	defer it.Close()

	for it.Next() {
		entry := it.Entry()
		if entry.IsNode {
			if err := w.DeleteNode(entry.NodeID); err != nil {
				return err
			}
			goneIDs[entry.NodeID] = struct{}{}
			*nodesRemoved++
		} else {
			if err := w.DeleteEdge(entry.Source, entry.Kind, entry.Target); err != nil {
				return err
			}
			*edgesRemoved++
		}
	}
	return it.Err()
}

// pruneOwnedEdgesOnly enumerates path's owned x/ entries in r0 and stages
// a DeleteEdge on w for each EDGE entry only — node entries are skipped
// so a dependent file's own (content-unchanged) nodes and File record
// survive (RESEARCH Pattern 1 step 8): only its outgoing edges are
// discarded, to be regenerated fresh once it is re-extracted and
// re-resolved.
//
// CR-04: DeleteEdge alone only removes the edge's own e/ record — it never
// touches path's x/<path>/e/... file-index entry (that is what
// DeleteFileSubgraph's whole-region range-delete is for, and this is
// deliberately NOT a full-file prune, since path's own nodes/File record
// must survive). Without the paired DeleteFileIndexEdge call below, a
// dependent whose re-resolution produces a DIFFERENT edge target (or no
// edge at all) leaves the OLD x/ entry stranded — no matching e/ record,
// never cleaned up — until a later DIRECT prune of path enumerates it via
// IterateFileIndex and phantom-counts a no-op delete as a real removal,
// corrupting Meta.EdgeCount's arithmetic.
func pruneOwnedEdgesOnly(r0 graphstore.Reader, w graphstore.Writer, path string, edgesRemoved *int) error {
	it, err := r0.IterateFileIndex(path)
	if err != nil {
		return err
	}
	defer it.Close()

	for it.Next() {
		entry := it.Entry()
		if entry.IsNode {
			continue
		}
		if err := w.DeleteEdge(entry.Source, entry.Kind, entry.Target); err != nil {
			return err
		}
		if err := w.DeleteFileIndexEdge(path, entry.Source, entry.Kind, entry.Target); err != nil {
			return err
		}
		*edgesRemoved++
	}
	return it.Err()
}

// ownerPathFor resolves the file of id, an edge's owner (edgeOwnerID), for
// CR-03: batch first (this cycle's freshly reparsed nodes — always
// sufficient for an edge resolved from a reference, whose owner is in the
// file being resolved), falling back to a lookup against r0 — the
// pre-Sync snapshot — when id belongs to a node that was NOT reparsed
// this cycle, which is the synthesized implements and override case. r0 is a
// point-in-time snapshot unaffected by this Sync's own staged-but-
// uncommitted writes, so this lookup cannot observe a partial commit; it
// also cannot resolve to a node id that is itself being pruned/superseded
// this cycle, since loadStoreView's excludePaths already keeps
// idx from ever resolving a reference into the batch's own files or a
// deleted file in the first place (RESEARCH Pitfall 1's exclusion set).
func ownerPathFor(id string, batch map[string]string, r0 graphstore.Reader) string {
	if p, ok := batch[id]; ok {
		return p
	}
	n, err := r0.GetNode(id)
	if err != nil {
		return ""
	}
	return n.GetFilePath()
}

// contentHash returns hex(sha256(file contents)) for the file at absPath
// — the same SHA-256 confirm step goextract.Extract computes internally,
// duplicated here (rather than exported from goextract) because Sync's
// stat-pre-filter needs this BEFORE deciding whether a file is even a
// reparse candidate, ahead of/independent from a full Extract call.
func contentHash(absPath string) (string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("indexer: reading %s: %w", absPath, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
