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
	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
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

	// nodeKindByID/nodeNameByID are GLOBAL (cross-file) node-id lookups,
	// built BEFORE any ref is resolved (Phase 5 RES-02 Pattern 2/3): the
	// RefKindEmbeds branch below needs to know a resolved target's Kind
	// (interface vs. not) to decide whether to promote the edge to
	// "implements" — and that target may be declared in a DIFFERENT file
	// than the one currently being resolved, exactly like
	// TestResolve_CrossFileMethodContainment's cross-file struct/method
	// pair. synthesizeGoImplements (below) reuses nodeNameByID for O(1)
	// method-name lookups instead of a per-edge linear scan over every
	// result's Nodes. This is a cheap, separate pass over results' own
	// Nodes; it does not touch the `nodes`/`edges` slices this function
	// accumulates incrementally below.
	nodeKindByID := make(map[string]string)
	nodeNameByID := make(map[string]string)
	nodeStartLineByID := make(map[string]int32)
	for _, r := range results {
		for _, en := range r.Nodes {
			nodeKindByID[en.Node.Id] = en.Node.Kind
			nodeNameByID[en.Node.Id] = en.Node.Name
			nodeStartLineByID[en.Node.Id] = en.Node.StartLine
		}
	}

	// pending accumulates every RefKindCalls ref that fails pass 1 —
	// Pitfall 3's conformance retry (Task 2) re-attempts these AFTER
	// implements/extends edges exist for the whole graph, walking the
	// supertype chain, rather than interleaving the retry into this same
	// loop. pc.file is filled in once this file's own *schema.File is
	// built below (files holds pointers, so mutating pc.file.EdgeCount
	// after appending to files is safe and visible).
	var pending []pendingCall

	for _, r := range results {
		for _, en := range r.Nodes {
			nodes = append(nodes, en.Node)
		}
		for _, ie := range r.IntraEdges {
			edges = append(edges, ie.Edge)
		}

		resolvedForFile := 0
		var filePending []goextract.UnresolvedRef
		for _, ref := range r.Unresolved {
			switch ref.Kind {
			case goextract.RefKindCalls:
				calleeID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					// Deferred, not counted as unresolved yet (Pitfall 3):
					// this call may still resolve via the conformance
					// retry below, once the calling method's enclosing
					// type's supertype chain is known.
					filePending = append(filePending, ref)
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
				// Pattern 2 (RES-02): a class/struct extends/implements
				// reference is syntactically undistinguished at
				// extraction time (Java's implements/extends, C#'s
				// comma-separated base_list, Go's struct/interface
				// embedding all emit the same RefKindEmbeds shape). Now
				// that the target is resolved, promote to "implements"
				// iff the target is an interface AND the source is not
				// itself an interface — a class-extends-class (or
				// interface-embeds-interface) stays a plain "embeds"
				// edge. This is deliberately NOT gated by r.Language: a
				// Go struct embedding an interface value
				// (`type W struct { io.Reader }`) genuinely does
				// promote that interface's methods onto the struct too
				// (real Go semantics), so the same structural rule
				// applies uniformly across every language.
				kind := "embeds"
				provenance := "ast"
				var metadata map[string]string
				if nodeKindByID[targetID] == goextract.KindInterface && nodeKindByID[ref.FromID] != goextract.KindInterface {
					kind = goextract.EdgeKindImplements
					provenance = "heuristic"
					metadata = map[string]string{"synthesizedBy": "declared-implements"}
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: kind,
					Line: ref.Line, Col: ref.Col, Provenance: provenance,
					Metadata: metadata,
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

		// RelPath == "" marks a synthetic, non-file result (Phase 5 LANG-07:
		// pipeline.go's Run appends exactly one such result carrying every
		// detected route's nodes/IntraEdges, via detectRoutes) — it
		// contributes Nodes/IntraEdges above like any other result, but
		// mints NO schema.File record, since it has no real source file to
		// attribute node/edge counts to (a synthetic "" path would collide
		// with nothing but would still be a misleading phantom File entry
		// in every file-listing surface).
		if r.RelPath == "" {
			continue
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

		for _, ref := range filePending {
			pending = append(pending, pendingCall{ref: ref, file: f})
		}
	}

	edges = append(edges, synthesizeGoImplements(results, edges, nodeKindByID, nodeNameByID, nodeStartLineByID)...)

	// Pitfall 3's conformance retry (Task 2): now that every "contains"
	// (type->method) and "embeds"/"implements" (type->supertype) edge
	// exists for the WHOLE graph — including this file's own
	// synthesized implements edges above — retry every call that failed
	// pass 1, walking the supertype chain from the calling method's own
	// enclosing type. This is a SEPARATE loop over the deferred refs, not
	// interleaved into the resolution loop above (Pitfall 3's explicit
	// requirement).
	edges = append(edges, retryConformanceCalls(pending, edges, nodeNameByID, &unresolvedCount)...)

	return nodes, packageNodes, edges, files, unresolvedCount
}

// pendingCall is a RefKindCalls ref that failed pass-1 resolution,
// deferred to the conformance retry (Pitfall 3), paired with the
// *schema.File its resolvedForFile/EdgeCount bookkeeping belongs to — a
// retry success increments file.EdgeCount exactly like a pass-1
// resolution would have.
type pendingCall struct {
	ref  goextract.UnresolvedRef
	file *schema.File
}

// retryConformanceCalls implements Pitfall 3's two-pass conformance
// retry: for each pending call, find the calling method's OWN enclosing
// type (via a "contains" edge, type->method) and walk that type's
// supertype chain (via "embeds"/"implements" edges, bounded by a
// visited-set-guarded BFS — cycle-safe, independent of graph width)
// looking for a method sharing the call's bare name. A call whose
// FromID has no enclosing type at all (e.g. a plain function, not a
// method — TestResolve_UnresolvedMethodCall's local-variable-receiver
// case) never resolves here, exactly matching pass 1's own behavior for
// that case. unresolvedCount is incremented for exactly those refs that
// fail BOTH passes — never double-counted (pass 1 no longer increments
// it for a deferred call).
func retryConformanceCalls(pending []pendingCall, edges []*schema.Edge, nodeNameByID map[string]string, unresolvedCount *int) []*schema.Edge {
	typeMethods := make(map[string]map[string]string) // typeID -> methodName -> methodID
	methodOwner := make(map[string]string)             // methodID -> typeID
	typeSupertypes := make(map[string][]string)        // typeID -> []supertypeID

	for _, e := range edges {
		switch e.Kind {
		case "contains":
			methods := typeMethods[e.Source]
			if methods == nil {
				methods = make(map[string]string)
				typeMethods[e.Source] = methods
			}
			methods[nodeNameByID[e.Target]] = e.Target
			methodOwner[e.Target] = e.Source
		case "embeds", goextract.EdgeKindImplements:
			typeSupertypes[e.Source] = append(typeSupertypes[e.Source], e.Target)
		}
	}

	var resolved []*schema.Edge
	for _, pc := range pending {
		ownerType, ok := methodOwner[pc.ref.FromID]
		var targetID string
		if ok {
			targetID, ok = walkSupertypesForMethod(ownerType, pc.ref.Name, typeMethods, typeSupertypes)
		}
		if !ok {
			*unresolvedCount++
			continue
		}
		resolved = append(resolved, &schema.Edge{
			Source: pc.ref.FromID, Target: targetID, Kind: "calls",
			Line: pc.ref.Line, Col: pc.ref.Col, Provenance: "ast",
		})
		pc.file.EdgeCount++
	}
	return resolved
}

// walkSupertypesForMethod performs a visited-set-guarded BFS from
// typeID's OWN supertypes (never typeID itself — pass 1 already
// attempted the calling type's own method set via resolveUnqualified)
// looking for methodName in typeMethods, returning the first match's
// node id. Bounded by the number of distinct types reachable through
// the supertype graph — cycle-safe regardless of malformed/cyclic
// embeds data.
func walkSupertypesForMethod(typeID, methodName string, typeMethods map[string]map[string]string, typeSupertypes map[string][]string) (string, bool) {
	visited := map[string]bool{typeID: true}
	queue := append([]string(nil), typeSupertypes[typeID]...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if id, ok := typeMethods[cur][methodName]; ok {
			return id, true
		}
		queue = append(queue, typeSupertypes[cur]...)
	}
	return "", false
}

// synthesizeGoImplements wires dispatch.SynthesizeImplements (RES-02
// Pattern 3, Go's structural method-set match) into Pass 2: it composes a
// struct node id -> method-spec-set map from the fully-resolved "contains"
// edges (a struct's method set may span multiple files, exactly like
// TestResolve_CrossFileMethodContainment) and an interface node id ->
// method-spec-set map directly from every FileResult's own
// InterfaceMethods, then hands both — plus an interface-embeds-interface
// adjacency derived from the already-resolved "embeds" edges — to
// dispatch.SynthesizeImplements. The returned edges are appended to the
// SAME edges slice this function already builds, so they flow through
// collapseEdges exactly like every other edge (D-06/D-07: no parallel
// dedup path, additive within SchemaVersion 1).
func synthesizeGoImplements(results []goextract.FileResult, edges []*schema.Edge, nodeKindByID, nodeNameByID map[string]string, nodeStartLineByID map[string]int32) []*schema.Edge {
	methodArity := make(map[string]int32)
	interfaceMethods := make(dispatch.InterfaceSpecs)
	for _, r := range results {
		for id, arity := range r.MethodArity {
			methodArity[id] = arity
		}
		for ifaceID, specs := range r.InterfaceMethods {
			interfaceMethods[ifaceID] = append(interfaceMethods[ifaceID], specs...)
		}
	}

	structMethods := make(dispatch.TypeMethods)
	interfaceEmbeds := make(map[string][]string)
	for _, e := range edges {
		switch e.Kind {
		case "contains":
			if nodeKindByID[e.Source] != goextract.KindStruct || nodeKindByID[e.Target] != goextract.KindMethod {
				continue
			}
			structMethods[e.Source] = append(structMethods[e.Source], goextract.MethodSpec{
				Name:  nodeNameByID[e.Target],
				Arity: methodArity[e.Target],
			})
		case "embeds":
			if nodeKindByID[e.Source] == goextract.KindInterface && nodeKindByID[e.Target] == goextract.KindInterface {
				interfaceEmbeds[e.Source] = append(interfaceEmbeds[e.Source], e.Target)
			}
		}
	}

	synthesized := dispatch.SynthesizeImplements(structMethods, interfaceMethods, interfaceEmbeds)
	// dispatch.SynthesizeImplements has no source-location data of its
	// own (it operates purely on name/arity sets) — anchor each
	// synthesized edge's Line at the implementing struct's OWN
	// declaration line (RES-03's "where a source location exists"
	// qualifier: unlike a declared `implements X` clause, Go's implicit
	// satisfaction has no single syntactic reference site, so the
	// struct's own declaration is the closest meaningful anchor).
	for _, e := range synthesized {
		e.Line = nodeStartLineByID[e.Source]
	}
	return synthesized
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
