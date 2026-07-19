package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// BuildReverseAdjacency builds an in-memory reverse-adjacency map keyed
// by edge.Target, from one full IterateEdges("") scan (D-04). It is
// filtered to goextract.RefKindCalls only — callers/impact/affected are
// call-graph traversals, and the golden callers.json/callees.json/
// impact.json shapes contain only call targets, never contains/embeds/
// imports edges, so a raw unfiltered scan would leak unrelated
// relationships into caller/blast-radius results.
//
// This is built fresh inside every caller (Callers/Impact/Affected) —
// no package-level cache, no sync.Once (RESEARCH Pitfall 2 /
// T-03-04-Stale): a long-lived process (the future MCP server) must
// never serve a stale point-in-time reverse view across multiple calls,
// even though Phase 3's CLI invocations are one-scan-per-process anyway.
// Exported (Phase 4, 04-02) so internal/indexer.Sync() can reuse this
// exact scan for dependent-file detection (D-02a) — callers outside
// this package MUST follow the same fresh-per-call discipline: never
// cache the result across a Sync() invocation.
func BuildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
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

// BuildImplementsIndex builds an in-memory index of "implements" edges
// keyed by edge.Target (the interface node), from one full
// IterateEdges("") scan — mirrors BuildReverseAdjacency's shape exactly
// (same fresh-per-call discipline, no package-level cache/sync.Once), but
// is a SEPARATE, purpose-built index (RES-02/D-06): dispatch traversal is
// name-joined (an interface method's callers must also reach every
// concrete implementation's SAME-NAMED method), not identity-followed
// like a "calls" edge, so BuildReverseAdjacency's goextract.RefKindCalls
// -only filter is deliberately NOT widened to admit "implements" edges —
// this is new, separate traversal code (per 05-PATTERNS.md).
func BuildImplementsIndex(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	idx := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.EdgeKindImplements {
			continue
		}
		idx[e.Target] = append(idx[e.Target], e)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	for _, edges := range idx {
		sort.Slice(edges, func(i, j int) bool { return edges[i].Source < edges[j].Source })
	}
	return idx, nil
}

// buildContainsIndex builds two in-memory views of "contains" edges from
// one full IterateEdges("") scan, mirroring BuildReverseAdjacency's/
// BuildImplementsIndex's fresh-per-call discipline: byType (a type node
// id -> the "contains" edges it owns, i.e. its methods) and methodOwner
// (a method node id -> its OWN enclosing type's id, the reverse
// direction). dispatchSiblingIDs uses both to find, for a queried method
// node, which type declares it, and then which OTHER types' methods to
// compose in.
func buildContainsIndex(r graphstore.Reader) (byType map[string][]*schema.Edge, methodOwner map[string]string, err error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, nil, err
	}
	defer it.Close()

	byType = make(map[string][]*schema.Edge)
	methodOwner = make(map[string]string)
	for it.Next() {
		e := it.Edge()
		if e.Kind != "contains" {
			continue
		}
		byType[e.Source] = append(byType[e.Source], e)
		methodOwner[e.Target] = e.Source
	}
	if err := it.Err(); err != nil {
		return nil, nil, err
	}
	return byType, methodOwner, nil
}

// implementedInterfaces inverts implementsIdx (Target/interface-keyed)
// into a Source/struct-keyed view (typeID -> []interfaceID) — cheap,
// built once per Callers/Impact call from the SAME already-materialized
// implementsIdx (no second IterateEdges scan): the number of implements
// edges is small relative to the whole graph.
func implementedInterfaces(implementsIdx map[string][]*schema.Edge) map[string][]string {
	out := make(map[string][]string)
	ifaceIDs := make([]string, 0, len(implementsIdx))
	for ifaceID := range implementsIdx {
		ifaceIDs = append(ifaceIDs, ifaceID)
	}
	sort.Strings(ifaceIDs)
	for _, ifaceID := range ifaceIDs {
		for _, e := range implementsIdx[ifaceID] {
			out[e.Source] = append(out[e.Source], ifaceID)
		}
	}
	return out
}

// dispatchSiblingIDs returns, for a resolved method node, every OTHER
// concrete implementation's same-named method id reachable through a
// shared "implements" edge (RES-02, D-06's name-joined traversal): node's
// own enclosing type's implemented interfaces, every OTHER type also
// implementing one of those interfaces, and — for each — its own method
// sharing node.Name, if any. Deterministic (sorted) output. Built fresh
// per call from indices that are themselves fresh-per-call
// (Callers/Impact's existing no-cache discipline is preserved, not
// relaxed, by this composition).
func dispatchSiblingIDs(reader graphstore.Reader, node *schema.Node, methodOwner map[string]string, containsByType map[string][]*schema.Edge, implementsIdx map[string][]*schema.Edge) ([]string, error) {
	ownerType, ok := methodOwner[node.Id]
	if !ok {
		return nil, nil
	}
	ifaces := implementedInterfaces(implementsIdx)[ownerType]
	if len(ifaces) == 0 {
		return nil, nil
	}

	seenImpl := map[string]bool{ownerType: true}
	var siblings []string
	for _, ifaceID := range ifaces {
		for _, e := range implementsIdx[ifaceID] {
			implID := e.Source
			if seenImpl[implID] {
				continue
			}
			seenImpl[implID] = true
			for _, ce := range containsByType[implID] {
				if ce.Target == node.Id {
					continue
				}
				m, err := reader.GetNode(ce.Target)
				if err != nil {
					if errors.Is(err, graphstore.ErrNotFound) {
						continue
					}
					return nil, err
				}
				if m.Name == node.Name {
					siblings = append(siblings, m.Id)
				}
			}
		}
	}
	sort.Strings(siblings)
	return siblings, nil
}

// resolveSymbolNode resolves a CLI-supplied symbol string to a concrete
// schema.Node by scanning IterateNodes() for an exact Name match
// (D-03 — no per-name key lookup exists, so this is a full scan like
// matchNodes in search.go). When multiple nodes share the same Name
// (e.g. same-named methods on different types), the pick is
// deterministic: the node with the lexicographically lowest Id wins, so
// repeated calls for the same symbol always resolve to the same node.
func (e *Engine) resolveSymbolNode(symbol string) (*schema.Node, error) {
	it, err := e.reader.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var best *schema.Node
	for it.Next() {
		n := it.Node()
		if n.Name != symbol {
			continue
		}
		if best == nil || n.Id < best.Id {
			best = n
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	if best == nil {
		return nil, fmt.Errorf("query: symbol %q not found", symbol)
	}
	return best, nil
}

// nodeLocation projects a schema.Node to the locations-only Location
// shape (search.go, D-06) shared by callers/callees/impact/affected.
func nodeLocation(n *schema.Node) Location {
	return Location{
		Name:      n.Name,
		Kind:      n.Kind,
		FilePath:  n.FilePath,
		StartLine: n.StartLine,
	}
}

// sortLocations orders locs by FilePath, then Name, then StartLine
// (WR-05, 08-REVIEW.md) — mirrors files.go's sortFileTree discipline
// ("--json output must be reproducible", D-06). Impact/Callers/Affected's
// result order previously depended entirely on the underlying
// graphstore.Reader's IterateNodes/IterateEdges enumeration order plus
// Go's frontier-processing order — a guarantee that lives outside this
// package and is not asserted by any test here. This final sort makes
// determinism a property of this package's own code, not an unverified
// upstream iteration-order contract. sort.SliceStable (not sort.Slice)
// is used so ties on the full (FilePath, Name, StartLine) key still
// resolve deterministically across repeated calls.
func sortLocations(locs []Location) {
	sort.SliceStable(locs, func(i, j int) bool {
		if locs[i].FilePath != locs[j].FilePath {
			return locs[i].FilePath < locs[j].FilePath
		}
		if locs[i].Name != locs[j].Name {
			return locs[i].Name < locs[j].Name
		}
		return locs[i].StartLine < locs[j].StartLine
	})
}

// CalleesResult mirrors the golden callees.json shape: {"symbol",
// "callees": [locations]}.
type CalleesResult struct {
	Symbol  string     `json:"symbol"`
	Callees []Location `json:"callees"`
}

// CallersResult mirrors the golden callers.json shape: {"symbol",
// "callers": [locations]}.
type CallersResult struct {
	Symbol  string     `json:"symbol"`
	Callers []Location `json:"callers"`
}

// ImpactResult mirrors the golden impact.json shape: {"symbol","depth",
// "nodeCount","edgeCount","affected": [locations]}.
type ImpactResult struct {
	Symbol    string     `json:"symbol"`
	Depth     int        `json:"depth"`
	NodeCount int        `json:"nodeCount"`
	EdgeCount int        `json:"edgeCount"`
	Affected  []Location `json:"affected"`
}

// AffectedResult has no golden oracle (D-07a) — this shape is this
// plan's own design: the changed files that were queried, plus the
// impacted test locations derived at query time (D-07).
type AffectedResult struct {
	Files         []string   `json:"files"`
	AffectedTests []Location `json:"affectedTests"`
}

// Callees returns symbol's forward call targets via a direct
// IterateEdges(srcID) range scan (D-04 — no reverse-adjacency scan
// needed for the forward direction), capped at limit (0/negative means
// unlimited, validated by validateLimit before any scan runs, V5).
func (e *Engine) Callees(symbol string, limit int) (CalleesResult, error) {
	if err := validateLimit(limit); err != nil {
		return CalleesResult{}, err
	}

	node, err := e.resolveSymbolNode(symbol)
	if err != nil {
		return CalleesResult{}, err
	}

	it, err := e.reader.IterateEdges(node.Id)
	if err != nil {
		return CalleesResult{}, err
	}
	defer it.Close()

	locs := []Location{}
	for it.Next() {
		edge := it.Edge()
		if edge.Kind != goextract.RefKindCalls {
			continue
		}
		target, err := e.reader.GetNode(edge.Target)
		if err != nil {
			// WR-04: a dangling edge (target node pruned/missing) degrades
			// gracefully — skip it rather than aborting the whole call.
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return CalleesResult{}, err
		}
		locs = append(locs, nodeLocation(target))
	}
	if err := it.Err(); err != nil {
		return CalleesResult{}, err
	}

	if limit > 0 && limit < len(locs) {
		locs = locs[:limit]
	}
	if len(locs) > MaxLimit {
		locs = locs[:MaxLimit]
	}
	return CalleesResult{Symbol: symbol, Callees: locs}, nil
}

// Callers returns symbol's reverse callers via the D-04 in-memory
// reverse-adjacency map (built fresh, BuildReverseAdjacency), capped at
// limit identically to Callees. RES-02: when symbol resolves to a method
// declared on a type that implements one or more interfaces, callers ALSO
// include callers of every OTHER implementer's same-named method — a call
// dispatched dynamically through the shared interface could have reached
// symbol, so its callers are relevant dispatch targets too (D-06's
// name-joined traversal, composed from BuildImplementsIndex + a
// "contains" index — both fresh-per-call, same discipline as
// BuildReverseAdjacency).
func (e *Engine) Callers(symbol string, limit int) (CallersResult, error) {
	if err := validateLimit(limit); err != nil {
		return CallersResult{}, err
	}

	node, err := e.resolveSymbolNode(symbol)
	if err != nil {
		return CallersResult{}, err
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return CallersResult{}, err
	}
	implementsIdx, err := BuildImplementsIndex(e.reader)
	if err != nil {
		return CallersResult{}, err
	}
	containsByType, methodOwner, err := buildContainsIndex(e.reader)
	if err != nil {
		return CallersResult{}, err
	}
	siblings, err := dispatchSiblingIDs(e.reader, node, methodOwner, containsByType, implementsIdx)
	if err != nil {
		return CallersResult{}, err
	}
	targetIDs := append([]string{node.Id}, siblings...)

	seen := make(map[string]bool)
	locs := []Location{}
	for _, tid := range targetIDs {
		for _, edge := range rev[tid] {
			if seen[edge.Source] {
				continue
			}
			src, err := e.reader.GetNode(edge.Source)
			if err != nil {
				// WR-04: skip a dangling edge rather than aborting.
				if errors.Is(err, graphstore.ErrNotFound) {
					continue
				}
				return CallersResult{}, err
			}
			seen[edge.Source] = true
			locs = append(locs, nodeLocation(src))
		}
	}

	// WR-05: sort BEFORE applying limit/MaxLimit caps, so the capped
	// subset itself is a deterministic "first N in sorted order" rather
	// than depending on the underlying reverse-adjacency map's
	// enumeration order.
	sortLocations(locs)
	if limit > 0 && limit < len(locs) {
		locs = locs[:limit]
	}
	if len(locs) > MaxLimit {
		locs = locs[:MaxLimit]
	}
	return CallersResult{Symbol: symbol, Callers: locs}, nil
}

// Impact returns the depth-bounded reverse blast radius of symbol: a
// BFS over the D-04 reverse-adjacency map, bounded by clampDepth(depth)
// (T-03-04-DoS, RESEARCH Pitfall 4). NodeCount is the count of distinct
// visited nodes including symbol itself; EdgeCount is the count of
// reverse edges inspected while expanding each depth's frontier — this
// counting rule is cross-checked against
// testdata/golden/corpus/weft-go/impact.json's arithmetic in
// traverse_test.go's TestImpact doc comment. RES-02: at each frontier
// node, dispatch siblings (same composition Callers uses — every OTHER
// implementer's same-named method, reached via a shared "implements"
// edge) are ALSO expanded, so a change's blast radius includes callers
// reachable only through dynamic dispatch. When no implements edges
// exist in the graph (the common case, and every pre-existing fixture
// this method's tests exercise), dispatch siblings are always empty and
// this composition is a no-op — the arithmetic above is unchanged.
func (e *Engine) Impact(symbol string, depth int) (ImpactResult, error) {
	if err := validateDepth(depth); err != nil {
		return ImpactResult{}, err
	}
	depth = clampDepth(depth)

	node, err := e.resolveSymbolNode(symbol)
	if err != nil {
		return ImpactResult{}, err
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return ImpactResult{}, err
	}
	implementsIdx, err := BuildImplementsIndex(e.reader)
	if err != nil {
		return ImpactResult{}, err
	}
	containsByType, methodOwner, err := buildContainsIndex(e.reader)
	if err != nil {
		return ImpactResult{}, err
	}

	visited := map[string]bool{node.Id: true}
	affected := []Location{nodeLocation(node)}
	frontier := []*schema.Node{node}
	edgeCount := 0

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []*schema.Node
		for _, n := range frontier {
			siblings, err := dispatchSiblingIDs(e.reader, n, methodOwner, containsByType, implementsIdx)
			if err != nil {
				return ImpactResult{}, err
			}
			targetIDs := append([]string{n.Id}, siblings...)

			for _, tid := range targetIDs {
				for _, edge := range rev[tid] {
					edgeCount++
					if visited[edge.Source] {
						continue
					}
					visited[edge.Source] = true
					srcNode, err := e.reader.GetNode(edge.Source)
					if err != nil {
						// WR-04: a dangling reverse-edge source is skipped —
						// it cannot be expanded further, but must not abort
						// the whole BFS.
						if errors.Is(err, graphstore.ErrNotFound) {
							continue
						}
						return ImpactResult{}, err
					}
					affected = append(affected, nodeLocation(srcNode))
					next = append(next, srcNode)
				}
			}
		}
		frontier = next
	}

	// WR-05: sort for deterministic output — NodeCount/EdgeCount are
	// unaffected (computed from len(affected)/the edgeCount counter
	// above, not from order).
	sortLocations(affected)

	return ImpactResult{
		Symbol:    symbol,
		Depth:     depth,
		NodeCount: len(affected),
		EdgeCount: edgeCount,
		Affected:  affected,
	}, nil
}

// isTestSymbol is D-07's test-file heuristic: a node counts as a test
// symbol if its file ends _test.go.
//
// WR-07 (08-REVIEW.md): the previous name-based fallback (any node whose
// Name started with Test*/Benchmark*, regardless of file) is dropped — a
// production helper merely named e.g. TestConnectionPool or
// BenchmarkSuiteResults is not a runnable Go test/benchmark function and
// was being misclassified as one, inflating affected's output with false
// positives. The _test.go-suffix check alone already covers Go's own
// test-function-naming convention: a TestXxx function not in a _test.go
// file cannot be run by `go test` in the first place, so the fallback
// added false positives without ever covering a real gap.
func isTestSymbol(n *schema.Node) bool {
	return strings.HasSuffix(n.FilePath, "_test.go")
}

// Affected derives impacted test files/symbols for a set of changed
// files via a depth-bounded BFS over the D-04 reverse-adjacency map,
// bounded by clampAffectedDepth(depth) (SURF-04/D-05, CONTEXT D-05,
// RESEARCH Pitfall 2) — NOT the single-hop lookup this used to be.
// Mirrors Impact's frontier/next-frontier loop shape (traverse.go
// above), but with TS 1.3.1's test-files-as-leaves pruning rule instead
// of Impact's "expand everything": a dependent that passes isTestSymbol
// is recorded as an affected test AND is a leaf — it is never queued
// for further expansion, so its own dependents can never surface at any
// depth. A non-test dependent is queued for the next hop and is never
// itself recorded as an affected test. There is no golden oracle for
// this command (D-07a); parity is structural, proved in
// traverse_test.go against seeded call chains.
//
// RES-02 (WR-04 — 08-REVIEW.md): each frontier node's dispatch siblings
// (the same dispatchSiblingIDs composition Callers/Impact already apply
// — every OTHER concrete implementation's same-named method, reached
// through a shared "implements" edge) are ALSO expanded at every hop, so
// a changed file's dispatch-reachable test dependents are not silently
// excluded from affectedTests just because Affected historically only
// composed the direct reverse-adjacency map. When no implements edges
// exist in the graph (the common case), dispatch siblings are always
// empty and this composition is a no-op.
func (e *Engine) Affected(files []string, depth int) (AffectedResult, error) {
	if err := validateDepth(depth); err != nil {
		return AffectedResult{}, err
	}
	depth = clampAffectedDepth(depth)

	// WR-02: normalize a nil files argument to an empty slice so
	// AffectedResult.Files never marshals as JSON null — mirrors the
	// existing `tests := []Location{}` convention this function (and
	// Callees/Callers) already applies to their own array fields. The
	// current CLI caller (collectAffectedFiles) always builds a non-nil
	// slice, masking this for today's only caller, but any other caller
	// of the exported Engine.Affected (a future MCP tool) that passes nil
	// must not observe a null array.
	if files == nil {
		files = []string{}
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return AffectedResult{}, err
	}
	implementsIdx, err := BuildImplementsIndex(e.reader)
	if err != nil {
		return AffectedResult{}, err
	}
	containsByType, methodOwner, err := buildContainsIndex(e.reader)
	if err != nil {
		return AffectedResult{}, err
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	it, err := e.reader.IterateNodes()
	if err != nil {
		return AffectedResult{}, err
	}
	defer it.Close()

	visited := make(map[string]bool)
	var frontier []*schema.Node
	for it.Next() {
		n := it.Node()
		if fileSet[n.FilePath] {
			frontier = append(frontier, n)
			visited[n.Id] = true
		}
	}
	if err := it.Err(); err != nil {
		return AffectedResult{}, err
	}

	tests := []Location{}
	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []*schema.Node
		for _, n := range frontier {
			siblings, err := dispatchSiblingIDs(e.reader, n, methodOwner, containsByType, implementsIdx)
			if err != nil {
				return AffectedResult{}, err
			}
			targetIDs := append([]string{n.Id}, siblings...)

			for _, tid := range targetIDs {
				for _, edge := range rev[tid] {
					if visited[edge.Source] {
						continue
					}
					dep, err := e.reader.GetNode(edge.Source)
					if err != nil {
						// WR-04: skip a dangling edge rather than aborting —
						// same convention as Callees/Callers/Impact above.
						if errors.Is(err, graphstore.ErrNotFound) {
							continue
						}
						return AffectedResult{}, err
					}
					visited[edge.Source] = true
					if isTestSymbol(dep) {
						// TS test-files-as-leaves semantics: record, but do
						// NOT queue for expansion — a test dependent's own
						// dependents are never pulled into the BFS.
						tests = append(tests, nodeLocation(dep))
						continue
					}
					next = append(next, dep)
				}
			}
		}
		frontier = next
	}

	// WR-05: sort for deterministic output, mirroring Impact/Callers above.
	sortLocations(tests)

	return AffectedResult{Files: files, AffectedTests: tests}, nil
}

// MarshalCalleesJSON, MarshalCallersJSON, MarshalImpactJSON, and
// MarshalAffectedJSON colocate --json shaping with the traversal
// methods that produce these results (matching search.go's
// MarshalQueryJSON convention, 03-03) — each result struct is already
// tagged to its golden shape, so marshaling is a thin passthrough.
func MarshalCalleesJSON(r CalleesResult) ([]byte, error) { return json.Marshal(r) }
func MarshalCallersJSON(r CallersResult) ([]byte, error) { return json.Marshal(r) }
func MarshalImpactJSON(r ImpactResult) ([]byte, error)   { return json.Marshal(r) }
func MarshalAffectedJSON(r AffectedResult) ([]byte, error) {
	return json.Marshal(r)
}
