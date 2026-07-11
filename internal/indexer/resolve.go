// Package indexer's resolve.go implements Pass 2 of the two-pass indexing
// pipeline (D-04): build a global symbol index over every Pass-1 result,
// resolve calls/imports/embeds/contains references into ground-truth
// edges, deterministically collapse duplicate edges (D-05), and commit the
// whole resolved graph through exactly one batched GraphStore.Writer
// (D-04a).
package indexer

import (
	"sort"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// kindPackage is the synthetic node kind for an intra-module import target
// (RQ-1 recommendation (a)) — not one of goextract's declared node kinds,
// since no source declaration produces it; it exists purely so an
// `imports` edge has something in-repo to target.
const kindPackage = "package"

// resolveRefs builds the global symbol index over results and settles
// every file's Unresolved references into ground-truth edges. It returns:
//   - nodes: every symbol/file node Pass 1 already extracted, unchanged
//   - packageNodes: the synthetic "package" pseudo-nodes minted for
//     intra-module imports (RQ-1), deduplicated by id
//   - edges: every intra-file edge Pass 1 already extracted, PLUS every
//     successfully resolved calls/imports/embeds/contains edge
//   - files: one schema.File per Pass-1 result
//   - unresolvedCount: how many Unresolved refs could NOT be resolved
//     (never silently dropped — D-06a; surfaced later via --verbose)
//
// modulePath is the repo's own module path (from Discover), used to
// classify an "imports" ref's target as intra-module (gets a package
// pseudo-node + edge) vs. external/stdlib (no node, no edge).
func resolveRefs(results []goextract.FileResult, modulePath string) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolvedCount int) {
	return resolveRefsWithIndex(results, modulePath, newSymbolIndex(results))
}

// resolveRefsWithIndex is resolveRefs' implementation, parameterized on
// the symbol index (Phase 4 D-01/RESEARCH Pattern 1 step 11): resolveRefs
// keeps building its own from-scratch newSymbolIndex(results) unconditio-
// nally; Sync() instead injects a store-seeded index overlaid with the
// reparse batch (newSymbolIndexFromStore + symbolIndex.overlay), so an
// unqualified/qualified reference into an UNCHANGED file still resolves
// (RESEARCH Pitfall 1). Every other line here is reused verbatim by both
// callers.
func resolveRefsWithIndex(results []goextract.FileResult, modulePath string, idx *symbolIndex) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolvedCount int) {
	packageNodeIDs := make(map[string]struct{})

	for _, r := range results {
		for _, en := range r.Nodes {
			nodes = append(nodes, en.Node)
		}
		for _, ie := range r.IntraEdges {
			edges = append(edges, ie.Edge)
		}

		resolvedForFile := 0
		for _, ref := range r.Unresolved {
			switch ref.Kind {
			case goextract.RefKindCalls:
				calleeID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: calleeID, Kind: "calls",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindEmbeds:
				targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: "embeds",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindContains:
				// A method whose receiver type lives in a different file
				// (02-03's 4th UnresolvedRef kind). The receiver type is
				// always declared in the method's own package in valid
				// Go, so this is always an unqualified lookup.
				typeID, ok := idx.resolveUnqualified(r.ImportPath, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: typeID, Target: ref.FromID, Kind: "contains",
					Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindImports:
				if !isIntraModule(modulePath, ref.Name) {
					// External/stdlib import: no in-repo node to target,
					// and D-03a forbids inventing edges beyond ground
					// truth — no node, no edge (RQ-1 recommendation (a)).
					continue
				}
				pkgID := nodeid.NodeID(kindPackage, ref.Name, ref.Name)
				if _, seen := packageNodeIDs[pkgID]; !seen {
					packageNodeIDs[pkgID] = struct{}{}
					packageNodes = append(packageNodes, &schema.Node{
						Id:            pkgID,
						Kind:          kindPackage,
						Name:          lastPathSegment(ref.Name),
						QualifiedName: ref.Name,
					})
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: pkgID, Kind: "imports",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++
			}
		}

		f := &schema.File{
			Path:        r.RelPath,
			ContentHash: r.ContentHash,
			Language:    r.Language,
			NodeCount:   int64(len(r.Nodes)),
			EdgeCount:   int64(len(r.IntraEdges) + resolvedForFile),
			MtimeUnixNs: r.MtimeUnixNs,
			SizeBytes:   r.SizeBytes,
		}
		if r.Err != nil {
			f.Errors = []string{r.Err.Error()}
		}
		files = append(files, f)
	}

	return nodes, packageNodes, edges, files, unresolvedCount
}

// resolveNameRef resolves a (pkgAlias, name) reference against idx: an
// empty pkgAlias is an unqualified, same-package reference; a non-empty
// pkgAlias must be a real import alias in r's Imports map (RQ-2's
// narrowest-safe-set boundary — resolveSelector's own alias check is what
// keeps a local-variable-receiver call like `w.Describe()` from ever
// resolving here, since "w" is never a key in r.Imports).
func resolveNameRef(idx *symbolIndex, r goextract.FileResult, pkgAlias, name string) (string, bool) {
	if pkgAlias == "" {
		return idx.resolveUnqualified(r.ImportPath, name)
	}
	return idx.resolveSelector(r.Imports, pkgAlias, name)
}

// isIntraModule reports whether importPath belongs to the same module as
// modulePath (either the module root itself or a subpackage of it).
func isIntraModule(modulePath, importPath string) bool {
	if importPath == modulePath {
		return true
	}
	return strings.HasPrefix(importPath, modulePath+"/")
}

// lastPathSegment returns the final "/"-delimited segment of importPath,
// the conventional default package name TS-parity behavior expects on the
// synthetic package pseudo-node (RQ-1).
func lastPathSegment(importPath string) string {
	if i := strings.LastIndexByte(importPath, '/'); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}

// Resolve runs the whole of Pass 2: builds the global symbol index,
// resolves every file's Unresolved references into edges, deterministically
// collapses duplicates, and commits the resolved graph through exactly one
// batched GraphStore.Writer (D-04a). It returns the count of references
// that could not be resolved (surfaced later via --verbose, never silently
// dropped — D-06a).
func Resolve(store graphstore.GraphStore, results []goextract.FileResult, modulePath string) (int, error) {
	nodes, packageNodes, edges, files, unresolvedCount := resolveRefs(results, modulePath)
	if err := writeGraph(store, nodes, packageNodes, edges, files); err != nil {
		return unresolvedCount, err
	}
	return unresolvedCount, nil
}

// edgeTriple is the (source, kind, target) identity a duplicate edge
// collapses on (D-05) — mirrors graphstore/keys.go's edgeKey shape exactly,
// without needing to import that package's unexported key-building code.
type edgeTriple struct {
	source, kind, target string
}

// collapseEdges aggregates every candidate edge sharing a (source, kind,
// target) triple, chooses ONE representative by sorting the candidates by
// a TOTAL ORDER — (filePath, line, col), via nodeFilePath[candidate.Source]
// — and takes the first (RESEARCH Pitfall 1). This is never "whichever
// candidate was appended first/last" (processing order); the same input
// set, in any order, always yields the same representative. The returned
// slice is itself in sorted triple order, so staging it via PutEdge is
// also deterministic.
func collapseEdges(edges []*schema.Edge, nodeFilePath map[string]string) []*schema.Edge {
	groups := make(map[edgeTriple][]*schema.Edge, len(edges))
	for _, e := range edges {
		k := edgeTriple{e.Source, e.Kind, e.Target}
		groups[k] = append(groups[k], e)
	}

	keys := make([]edgeTriple, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].source != keys[j].source {
			return keys[i].source < keys[j].source
		}
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		return keys[i].target < keys[j].target
	})

	collapsed := make([]*schema.Edge, 0, len(keys))
	for _, k := range keys {
		candidates := groups[k]
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := candidates[i], candidates[j]
			fi, fj := nodeFilePath[ci.Source], nodeFilePath[cj.Source]
			if fi != fj {
				return fi < fj
			}
			if ci.Line != cj.Line {
				return ci.Line < cj.Line
			}
			return ci.Col < cj.Col
		})
		collapsed = append(collapsed, candidates[0])
	}
	return collapsed
}

// writeGraph collapses edges deterministically and stages the whole
// resolved graph — package pseudo-nodes and symbol nodes (sorted by id),
// then files (sorted by path), then collapsed edges (sorted by
// source/kind/target) — through exactly one GraphStore.Writer, committing
// once (D-04a). Any staging error releases the batch via Close() (never a
// partial Commit) and returns the error.
func writeGraph(store graphstore.GraphStore, nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File) error {
	nodeFilePath := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeFilePath[n.Id] = n.FilePath
	}

	allNodes := make([]*schema.Node, 0, len(nodes)+len(packageNodes))
	allNodes = append(allNodes, nodes...)
	allNodes = append(allNodes, packageNodes...)
	sort.Slice(allNodes, func(i, j int) bool { return allNodes[i].Id < allNodes[j].Id })

	sortedFiles := make([]*schema.File, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool { return sortedFiles[i].Path < sortedFiles[j].Path })

	collapsedEdges := collapseEdges(edges, nodeFilePath)

	w, err := store.NewWriter()
	if err != nil {
		return err
	}

	for _, n := range allNodes {
		if err := w.PutNode(n); err != nil {
			w.Close()
			return err
		}
	}
	for _, f := range sortedFiles {
		if err := w.PutFile(f); err != nil {
			w.Close()
			return err
		}
	}
	for _, e := range collapsedEdges {
		if err := w.PutEdge(e, nodeFilePath[e.Source]); err != nil {
			w.Close()
			return err
		}
	}

	meta := schema.NewMeta()
	meta.NodeCount = int64(len(allNodes))
	meta.EdgeCount = int64(len(collapsedEdges))
	// Phase 4 D-02b: every PutNode/PutEdge this Writer stages already
	// populates the x/ file-owned secondary index unconditionally (04-01)
	// — so any graph committed through writeGraph genuinely HAS a
	// complete x/ index by the time this Commit lands, regardless of
	// whether the caller was a from-scratch Run or Sync. Stamping the
	// flag here (not just in Sync's own backfill path) means only a
	// GENUINELY pre-Phase-4 store (built before this field/namespace
	// existed) is ever missing it.
	meta.HasFileIndex = true
	// Phase 4 D-04a: stamp LastSyncUnixMs here too, not just in Sync's own
	// meta-write step (internal/indexer/sync.go) — otherwise a graph built
	// via a from-scratch `codegraph index` (this path) carries a zero
	// last_sync_unix_ms forever until the first incremental sync runs,
	// which would make query.Engine.Status's newest-mtime-vs-last_sync
	// staleness fallback (04-06) report every freshly indexed repo as
	// permanently stale.
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	if err := w.PutMeta(meta); err != nil {
		w.Close()
		return err
	}

	return w.Commit()
}
