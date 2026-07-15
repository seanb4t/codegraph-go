// Package query — internal/query/expand.go ports TS CodeGraph 1.3.1's
// subgraph-construction heuristics H10-H12 (RESEARCH §C.2, D-10): the
// DoS-bounding stage (T-01-18) that turns explore's gathered candidate
// set (H3-H6, gather.go) into the bounded subgraph computeGraphRelevance
// (rwr.go) later ranks over. The live TS dist is no longer readable on
// this machine (see gather.go's package doc comment) — every constant
// below is cited from the frozen 01-RESEARCH.md §C.2 capture, not
// re-derived from a fresh source read:
//
//   - H10 Type-hierarchy expansion  — context/index.js:921-955; traversal.js:332-380
//   - H11 BFS traversal bounds      — mcp/tools.js:2422-2427
//   - H12 Glue-node injection       — mcp/tools.js:2439-2467
//
// A later plan (16, "wire time") composes expandTypeHierarchy,
// expandBFS, and expandGlueNodes into Explore()'s subgraph-gathering
// pipeline; this plan lands the three primitives as pure,
// graphstore.Reader-driven functions, independently unit-testable
// against a synthetic index (mirroring gather.go's/traverse.go's own
// fresh-per-call, no-cache discipline throughout).
package query

import (
	"errors"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// H11's explore-override bounds and H12's cap (RESEARCH §C.2,
// mcp/tools.js:2422-2427 / 2439-2467 [VERIFIED: TS 1.3.1 dist — cited
// from the frozen RESEARCH capture]) — ported VERBATIM. These are
// explore's OWN overrides of TS's more permissive library defaults; this
// port carries only the override values, since explore is the only
// caller these primitives currently serve.
const (
	ExpandMaxNodes       = 200
	ExpandTraversalDepth = 3
	ExpandMinScore       = 0.2
	ExpandSearchLimit    = 8
	GlueNodeCap          = 60
)

// DefaultExploreBFSBounds bundles H11's four override constants above
// into the ExpandBFSBounds shape expandBFS consumes.
var DefaultExploreBFSBounds = ExpandBFSBounds{
	MaxNodes:       ExpandMaxNodes,
	TraversalDepth: ExpandTraversalDepth,
	MinScore:       ExpandMinScore,
	SearchLimit:    ExpandSearchLimit,
}

// --- H10: type-hierarchy expansion ---

// expandHierarchyKinds is H10's type-kind filter (class/interface/
// struct/trait/protocol per RESEARCH §C.2). Reuses gather.go's
// definitionKinds rather than re-declaring an identical map: this
// codebase's Kind vocabulary already collapses that whole TS set onto
// exactly KindStruct/KindInterface/KindTypeAlias (see gather.go's
// definitionKinds doc comment, D-02) — H10's slightly narrower TS
// wording ("class/interface/struct/trait/protocol", no enum/type_alias)
// still lands on the identical 3 Go Kind values, so a second map would
// be a duplicate, not a distinct filter.
var expandHierarchyKinds = definitionKinds

// typeHierarchyEdgeKinds is the extends/implements pair H10 walks —
// RESEARCH §C.2/H10 names both explicitly ("via extends/implements").
var typeHierarchyEdgeKinds = map[string]bool{
	goextract.EdgeKindExtends:    true,
	goextract.EdgeKindImplements: true,
}

// expandHierarchyBudget is H10's exact, cited constant: ceil(maxNodes/4)
// (RESEARCH §C.2/H10).
func expandHierarchyBudget(maxNodes int) int {
	if maxNodes <= 0 {
		return 0
	}
	return (maxNodes + 3) / 4
}

// buildTypeHierarchyIndex builds, from one full IterateEdges("") scan,
// two in-memory views over extends/implements edges: parents (a type id
// -> the ids of its DIRECT extends/implements targets, i.e. its
// supertypes) and children (a type id -> the ids of the OTHER types that
// DIRECTLY extend/implement it, i.e. its immediate subtypes) — the
// forward and reverse directions H10's two-pass walk needs. Mirrors
// BuildReverseAdjacency's/BuildImplementsIndex's fresh-per-call,
// no-package-cache discipline (traverse.go) and their sorted-slice
// determinism.
func buildTypeHierarchyIndex(r graphstore.Reader) (parents, children map[string][]string, err error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, nil, err
	}
	defer it.Close()

	parents = make(map[string][]string)
	children = make(map[string][]string)
	for it.Next() {
		e := it.Edge()
		if !typeHierarchyEdgeKinds[e.Kind] {
			continue
		}
		parents[e.Source] = append(parents[e.Source], e.Target)
		children[e.Target] = append(children[e.Target], e.Source)
	}
	if err := it.Err(); err != nil {
		return nil, nil, err
	}
	for k := range parents {
		sort.Strings(parents[k])
	}
	for k := range children {
		sort.Strings(children[k])
	}
	return parents, children, nil
}

// transitiveWalk BFS-walks index (parents or children) starting from
// root's DIRECT neighbors, following the SAME direction repeatedly — so
// calling it with the parents index yields every TRANSITIVE ancestor of
// root, and with the children index, every TRANSITIVE descendant.
// Cycle-safe (a visited set) and deterministic: each BFS frontier level
// is sorted before it is walked or used to build the next level
// (mirrors traverse.go's Impact BFS discipline).
func transitiveWalk(root string, index map[string][]string) []string {
	visited := map[string]bool{root: true}
	var out []string
	frontier := append([]string(nil), index[root]...)
	sort.Strings(frontier)
	for len(frontier) > 0 {
		var next []string
		for _, id := range frontier {
			if visited[id] {
				continue
			}
			visited[id] = true
			out = append(out, id)
			next = append(next, index[id]...)
		}
		sort.Strings(next)
		frontier = next
	}
	return out
}

// expandTypeHierarchy is H10: for each focal type-kind node in
// focalIDs, add its FULL transitive extends/implements ancestor chain
// and FULL transitive descendant tree (Pass 1), then — for every
// ancestor newly discovered in Pass 1 — add that ancestor's OTHER
// direct children (siblings of the focal chain not already found in
// Pass 1, Pass 2 per RESEARCH §C.2/H10) — all bounded to a total of
// expandHierarchyBudget(maxNodes) newly-added node ids. A focalIDs entry
// that does not resolve to an expandHierarchyKinds node (H10 only fires
// for class/interface/struct/trait/protocol-shaped nodes) is silently
// skipped, not an error. focalIDs themselves are never counted against
// the budget or re-added — they are assumed already present in the
// subgraph. Deterministic: focal ids are walked in sorted order, and
// the returned id slice is sorted.
func expandTypeHierarchy(r graphstore.Reader, focalIDs []string, maxNodes int) ([]string, error) {
	budget := expandHierarchyBudget(maxNodes)
	if budget <= 0 || len(focalIDs) == 0 {
		return nil, nil
	}

	parents, children, err := buildTypeHierarchyIndex(r)
	if err != nil {
		return nil, err
	}

	candidateFocal := append([]string(nil), focalIDs...)
	sort.Strings(candidateFocal)
	var sortedFocal []string
	for _, id := range candidateFocal {
		n, err := r.GetNode(id)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if !expandHierarchyKinds[n.Kind] {
			continue
		}
		sortedFocal = append(sortedFocal, id)
	}
	focalSet := make(map[string]bool, len(sortedFocal))
	for _, id := range sortedFocal {
		focalSet[id] = true
	}

	added := make(map[string]bool)
	var order []string
	newParents := make(map[string]bool) // "newly-found parents" Pass 2 walks

	// add reports whether the budget is now exhausted (stop == true), so
	// every call site can break out of its walk loop immediately.
	add := func(id string) (stop bool) {
		if focalSet[id] || added[id] {
			return false
		}
		added[id] = true
		order = append(order, id)
		return len(added) >= budget
	}

pass1:
	for _, focal := range sortedFocal {
		for _, id := range transitiveWalk(focal, parents) {
			newParents[id] = true
			if add(id) {
				break pass1
			}
		}
		for _, id := range transitiveWalk(focal, children) {
			if add(id) {
				break pass1
			}
		}
	}

	if len(added) < budget {
		parentIDs := make([]string, 0, len(newParents))
		for id := range newParents {
			parentIDs = append(parentIDs, id)
		}
		sort.Strings(parentIDs)
	pass2:
		for _, p := range parentIDs {
			for _, sib := range children[p] {
				if add(sib) {
					break pass2
				}
			}
		}
	}

	sort.Strings(order)
	return order, nil
}

// --- H11: BFS traversal bounds ---

// ExpandBFSBounds groups H11's four explicit override bounds (RESEARCH
// §C.2/H11) so callers pass one value, not four positional
// ints/floats.
type ExpandBFSBounds struct {
	MaxNodes       int
	TraversalDepth int
	MinScore       float64
	SearchLimit    int
}

// buildExpandAdjacency builds, from one full IterateEdges("") scan, an
// undirected adjacency map (RankEdges-filtered, self-loop-excluded —
// mirrors buildRWRAdjacency's exact kind filter and exclusion rules,
// rwr.go) plus the flat RankEdges-filtered edge list expandBFS returns
// alongside its node id set, so the subgraph RWR later ranks over
// (rwr.go's computeGraphRelevance) is exactly the edge universe this
// BFS itself walked — no separate, potentially-inconsistent re-scan.
func buildExpandAdjacency(r graphstore.Reader) (map[string][]string, []*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, nil, err
	}
	defer it.Close()

	adj := make(map[string][]string)
	var edges []*schema.Edge
	for it.Next() {
		e := it.Edge()
		if !RankEdges[e.Kind] {
			continue
		}
		if e.Source == e.Target {
			continue
		}
		edges = append(edges, e)
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source) // undirected
	}
	if err := it.Err(); err != nil {
		return nil, nil, err
	}
	return adj, edges, nil
}

// expandBFS is H11: build the bounded subgraph node/edge set
// computeGraphRelevance (rwr.go) later ranks over. Input roots are the
// H3-H6 gathered/scored candidates (gather.go's gatherCandidate); any
// root scoring below bounds.MinScore is pruned BEFORE traversal (a
// low-relevance candidate never seeds expansion), the surviving roots
// are trimmed to bounds.SearchLimit (highest score first, D-04's
// score-desc-then-Id-asc tie-break via sortGatherCandidates), and the
// subgraph then BFS-expands from those seed roots along RankEdges
// (undirected) — bounded to bounds.TraversalDepth hops and
// bounds.MaxNodes total nodes (roots count toward the cap). Frontier
// and neighbor ordering is sorted at every level for deterministic
// admission when the cap binds mid-traversal (RESEARCH §C.2/H11's own
// admission order is not recoverable from the frozen capture, so D-04's
// sorted-Id convention is used here, matching the plan's own
// instruction to "keep deterministic frontier ordering"). WR-04: an
// edge endpoint that does not resolve to an actual node (a dangling
// edge) is skipped rather than visited, mirroring Impact's BFS
// (traverse.go) dangling-skip discipline.
func expandBFS(r graphstore.Reader, roots []gatherCandidate, bounds ExpandBFSBounds) (nodeIDs, seedIDs []string, edges []*schema.Edge, err error) {
	pruned := make([]gatherCandidate, 0, len(roots))
	for _, c := range roots {
		if c.Score < bounds.MinScore {
			continue
		}
		pruned = append(pruned, c)
	}
	sortGatherCandidates(pruned)
	if bounds.SearchLimit > 0 && len(pruned) > bounds.SearchLimit {
		pruned = pruned[:bounds.SearchLimit]
	}

	seedSet := make(map[string]bool, len(pruned))
	for _, c := range pruned {
		seedSet[c.Node.Id] = true
	}
	seedIDs = make([]string, 0, len(seedSet))
	for id := range seedSet {
		seedIDs = append(seedIDs, id)
	}
	sort.Strings(seedIDs)
	if len(seedIDs) == 0 {
		return nil, nil, nil, nil
	}

	adj, allEdges, err := buildExpandAdjacency(r)
	if err != nil {
		return nil, nil, nil, err
	}

	visited := make(map[string]bool, bounds.MaxNodes)
	var order []string
	for _, id := range seedIDs {
		if bounds.MaxNodes > 0 && len(order) >= bounds.MaxNodes {
			break
		}
		visited[id] = true
		order = append(order, id)
	}

	frontier := append([]string(nil), order...)
	for depth := 0; depth < bounds.TraversalDepth && len(frontier) > 0; depth++ {
		if bounds.MaxNodes > 0 && len(order) >= bounds.MaxNodes {
			break
		}
		var next []string
		for _, id := range frontier {
			neighbors := append([]string(nil), adj[id]...)
			sort.Strings(neighbors)
			for _, n := range neighbors {
				if bounds.MaxNodes > 0 && len(order) >= bounds.MaxNodes {
					break
				}
				if visited[n] {
					continue
				}
				if _, gerr := r.GetNode(n); gerr != nil {
					if errors.Is(gerr, graphstore.ErrNotFound) {
						visited[n] = true // WR-04: dangling edge target, never revisit
						continue
					}
					return nil, nil, nil, gerr
				}
				visited[n] = true
				order = append(order, n)
				next = append(next, n)
			}
			if bounds.MaxNodes > 0 && len(order) >= bounds.MaxNodes {
				break
			}
		}
		frontier = next
	}

	inSubgraph := make(map[string]bool, len(order))
	for _, id := range order {
		inSubgraph[id] = true
	}
	for _, e := range allEdges {
		if inSubgraph[e.Source] && inSubgraph[e.Target] {
			edges = append(edges, e)
		}
	}

	sort.Strings(order)
	return order, seedIDs, edges, nil
}

// --- H12: glue-node injection ---

// subgraphFileSet resolves each id in ids to its Node.FilePath and
// returns the set of DISTINCT non-empty paths — "the files the subgraph
// already surfaces" that H12's same-file-only constraint checks
// against. WR-04: an id that no longer resolves (already-pruned/
// dangling) is skipped, not an error.
func subgraphFileSet(r graphstore.Reader, ids []string) (map[string]bool, error) {
	files := make(map[string]bool)
	for _, id := range ids {
		n, err := r.GetNode(id)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if n.FilePath != "" {
			files[n.FilePath] = true
		}
	}
	return files, nil
}

// calleeCallEdges returns srcID's forward RefKindCalls edges via a
// direct IterateEdges(srcID) range scan — mirrors Callees' (traverse.go)
// exact filter, reused here rather than re-implemented, since H12's
// "callees" are precisely that forward call-graph edge set.
func calleeCallEdges(r graphstore.Reader, srcID string) ([]*schema.Edge, error) {
	it, err := r.IterateEdges(srcID)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var out []*schema.Edge
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		out = append(out, e)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// admitGlueCandidate looks up id and, if it is not already a subgraph
// root and its FilePath is in surfacedFiles (H12's same-file-only
// constraint), records it in candidates. WR-04: a dangling edge
// endpoint is skipped, not an error.
func admitGlueCandidate(r graphstore.Reader, id string, rootSet, surfacedFiles, candidates map[string]bool) error {
	if rootSet[id] {
		return nil
	}
	n, err := r.GetNode(id)
	if err != nil {
		if errors.Is(err, graphstore.ErrNotFound) {
			return nil
		}
		return err
	}
	if !surfacedFiles[n.FilePath] {
		return nil
	}
	candidates[id] = true
	return nil
}

// expandGlueNodes is H12: for every root in rootIDs, find its DIRECT
// callers (via BuildReverseAdjacency, traverse.go) and callees (via
// calleeCallEdges) — RefKindCalls only, "callers+callees" per RESEARCH
// §C.2/H12 — and inject each ONLY if it lives in a file already in
// surfacedFiles (the subgraph's already-surfaced file set, e.g. from
// subgraphFileSet); a caller/callee in a NOT-yet-surfaced file is never
// pulled in as a glue node, no matter how central. Total glue nodes
// across every root are capped at glueCap; when the cap binds,
// candidates are kept in sorted-Id order — RESEARCH §C.2/H12's own
// admission order is not recoverable from the frozen capture, so D-04's
// lowest-Id-first convention is used (matching
// resolveSymbolNode/sortRWRScores), giving repeated calls over the same
// input an identical result.
func expandGlueNodes(r graphstore.Reader, rootIDs []string, surfacedFiles map[string]bool, glueCap int) ([]string, error) {
	if glueCap <= 0 || len(rootIDs) == 0 {
		return nil, nil
	}

	rev, err := BuildReverseAdjacency(r)
	if err != nil {
		return nil, err
	}

	sortedRoots := append([]string(nil), rootIDs...)
	sort.Strings(sortedRoots)
	rootSet := make(map[string]bool, len(sortedRoots))
	for _, id := range sortedRoots {
		rootSet[id] = true
	}

	candidates := make(map[string]bool)
	for _, root := range sortedRoots {
		for _, e := range rev[root] { // callers
			if err := admitGlueCandidate(r, e.Source, rootSet, surfacedFiles, candidates); err != nil {
				return nil, err
			}
		}
		callees, err := calleeCallEdges(r, root)
		if err != nil {
			return nil, err
		}
		for _, e := range callees {
			if err := admitGlueCandidate(r, e.Target, rootSet, surfacedFiles, candidates); err != nil {
				return nil, err
			}
		}
	}

	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > glueCap {
		ids = ids[:glueCap]
	}
	return ids, nil
}
