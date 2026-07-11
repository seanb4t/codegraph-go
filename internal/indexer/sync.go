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
// files, plus their direct call-graph dependents, are reparsed;
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
// nodes (newSymbolIndexFromStore), not just the reparse batch, so a
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

	files, modulePath, err := Discover(repoRoot)
	if err != nil {
		return Stats{}, err
	}

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

	if len(added) == 0 && len(modified) == 0 && len(deleted) == 0 {
		if len(mtimeRefresh) == 0 {
			// Fully no-op sync: nothing changed on disk.
			return Stats{
				Files:    len(files),
				Nodes:    int(meta.GetNodeCount()),
				Edges:    int(meta.GetEdgeCount()),
				Duration: time.Since(start),
			}, nil
		}
		// WR-03: no content actually changed, but at least one File
		// record's stored mtime/size is stale relative to disk — persist
		// the refresh in its own small commit so the stat pre-filter's
		// fast path is restored on the next Sync. NodeCount/EdgeCount are
		// untouched (nothing was reparsed or pruned).
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
		newMeta := schema.NewMeta()
		newMeta.HasFileIndex = true
		newMeta.LastSyncUnixMs = time.Now().UnixMilli()
		newMeta.NodeCount = meta.GetNodeCount()
		newMeta.EdgeCount = meta.GetEdgeCount()
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
	changedSet := make(map[string]struct{}, len(added)+len(modified))
	for _, f := range added {
		changedSet[f.RelPath] = struct{}{}
	}
	for _, f := range modified {
		changedSet[f.RelPath] = struct{}{}
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

	idx, err := newSymbolIndexFromStore(r0, modulePath, excludePaths)
	if err != nil {
		w.Close()
		return Stats{}, err
	}
	idx.overlay(results)

	nodes, packageNodes, edges, fileRecords, unresolvedCount := resolveRefsWithIndex(results, modulePath, idx)

	nodeFilePath := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeFilePath[n.Id] = n.FilePath
	}
	collapsedEdges := collapseEdges(edges, nodeFilePath)

	// CR-03: collapseEdges' own representative-selection above only needs
	// nodeFilePath for edges whose Source is in THIS cycle's reparse batch
	// (true for calls/embeds/imports, whose Source is always ref.FromID —
	// the file currently being parsed). A `contains` edge's Source is the
	// receiver TYPE's node id, which idx.resolveUnqualified can resolve
	// into a completely unchanged, non-reparsed file (the "type in one
	// file, methods in another" Go idiom) — nodeFilePath alone would miss
	// it, silently writing the edge with ownerPath="" and permanently
	// escaping the x/ index (never enumerable by a later prune of that
	// type's file). ownerPathFor below falls back to r0 — the pre-Sync
	// snapshot, unaffected by this cycle's staged-but-uncommitted writes —
	// for exactly that case.

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
		if err := w.PutEdge(e, ownerPathFor(e.Source, nodeFilePath, r0)); err != nil {
			w.Close()
			return Stats{}, err
		}
	}

	newMeta := schema.NewMeta()
	newMeta.HasFileIndex = true
	newMeta.LastSyncUnixMs = time.Now().UnixMilli()
	newMeta.NodeCount = meta.GetNodeCount() - int64(nodesRemoved) + int64(nodesAdded)
	newMeta.EdgeCount = meta.GetEdgeCount() - int64(edgesRemoved) + int64(len(collapsedEdges))
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
// Meta record yet, or a Meta record whose has_file_index flag is false
// (D-02b) — opening and closing the store in isolation from Sync's own
// main open, so the two never race a Pebble lock against each other.
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
	return !meta.GetHasFileIndex(), nil
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

// ownerPathFor resolves e's ownerPath for CR-03: batch first (this cycle's
// freshly reparsed nodes — always sufficient for calls/embeds/imports,
// whose Source is always in-batch), falling back to a lookup against r0 —
// the pre-Sync snapshot — when id belongs to a node that was NOT reparsed
// this cycle, which is exactly the cross-file `contains` case (the
// receiver type's node lives in an unchanged sibling file). r0 is a
// point-in-time snapshot unaffected by this Sync's own staged-but-
// uncommitted writes, so this lookup cannot observe a partial commit; it
// also cannot resolve to a node id that is itself being pruned/superseded
// this cycle, since newSymbolIndexFromStore's excludePaths already keeps
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
