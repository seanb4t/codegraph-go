package query

import (
	"encoding/json"
	"errors"
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

// buildReverseAdjacency is BuildReverseAdjacency behind an unexported
// package var — the same test-only control-seam convention
// internal/graphstore/pebble_store.go's openLockRetrySleep uses — so a
// test can count invocations of the reverse-adjacency build without an
// exported setter and with no production behavior change. detail.go's
// buildSingleDefDetail and buildMultiDefDetail route through this var
// rather than calling BuildReverseAdjacency directly, so the once-per-
// call discipline they preserve is counted rather than merely asserted
// from the code (ENG-01 performance invariant).
var buildReverseAdjacency = BuildReverseAdjacency

// filegraphKindPackage is internal/indexer/resolve.go's unexported
// kindPackage — its value is duplicated here as a string literal, not
// imported, because it is deliberately unexported by internal/indexer and
// this comment is what ties the two together (mirrors validate.go's
// knownKinds convention); if kindPackage's value ever changes, this
// literal must change with it. D-08 forbids touching resolve.go itself —
// the correction stays inside this phase's new code, at the read site.
const filegraphKindPackage = "package"

// FileGraphNode is one file-granularity node in Engine.FileGraph()'s
// rollup (ENG-03, GRF-02).
type FileGraphNode struct {
	// Path is the file's repo-relative path (schema.Node.FilePath).
	Path string
	// Language is the file's language, taken from its own KindFile
	// record when one is present, else the first non-empty Language seen
	// among the file's retained symbol records.
	Language string
	// SymbolCount is the number of DECLARED symbols in the file. The
	// file's own file-kind record is NOT a symbol the file declares and
	// is never counted, nor is a synthetic package record (both are
	// excluded before this tally runs) — a file declaring three symbols
	// reports three, never four (review M-4).
	SymbolCount int64
	// CycleID is 0 for a node in no strongly-connected cycle and is
	// populated by the cycle detector, not by this scan.
	CycleID int
}

// FileGraphEdge is one aggregated source-file-to-target-file edge in
// Engine.FileGraph()'s rollup, with a sparse per-kind count.
type FileGraphEdge struct {
	SourceFile string
	TargetFile string
	// KindCounts holds one entry per observed edge kind rolled into this
	// file pair. A kind with zero observed edges is absent from the map,
	// never present with value 0 (mirrors edgesByKind's sparse-map
	// convention, status.go).
	KindCounts map[string]int64
	TotalCount int64
	// InCycle is true when both this edge's endpoints carry the same
	// non-zero CycleID — populated by the cycle detector, not by this
	// scan.
	InCycle bool
}

// FileGraphResult is Engine.FileGraph()'s return value: the aggregated
// file-granularity node and edge rollup, plus counts of every exclusion
// the scan made — an exclusion is never silent (D-03, D-08).
type FileGraphResult struct {
	Nodes []FileGraphNode
	Edges []FileGraphEdge
	// ExcludedPackageNodes counts synthetic "package"-kind pseudo-nodes
	// (and any other record with an empty FilePath) skipped in the node
	// scan (D-08).
	ExcludedPackageNodes int64
	// ExcludedSelfEdges counts edges of a retained kind whose source and
	// target roll up to the same file (D-03).
	ExcludedSelfEdges int64
	// ExcludedContainsEdges counts edges of kind "contains" excluded
	// before aggregation (D-03).
	ExcludedContainsEdges int64
	// CycleCount is the number of distinct strongly-connected components
	// of size 2 or more found over the aggregated file adjacency —
	// populated by the cycle detector, not by this scan.
	CycleCount int
}

// fileAgg accumulates FileGraph's scan-one state per distinct file path:
// the language to report and the count of declared (non-file-kind,
// non-excluded) symbol records.
type fileAgg struct {
	language             string
	languageFromFileKind bool
	symbolCount          int64
}

// filePairKey identifies one aggregated source-file/target-file pair in
// FileGraph's scan-two accumulation map.
type filePairKey struct {
	source string
	target string
}

// FileGraph computes Engine.FileGraph()'s file-granularity rollup
// (ENG-03, GRF-02, D-05/D-09).
//
// It is built FRESH inside every call, with no package-level cache and no
// once-latch — mirroring BuildReverseAdjacency's fresh-per-call
// discipline above (D-05).
//
// It needs TWO scans, not one: schema.Edge carries no file path and node
// IDs are opaque content hashes, so resolving an edge's endpoints to
// their containing files requires a full node scan (mapping node id to
// file path) before the edge scan can be aggregated (D-09, correcting
// D-05's single-scan assumption).
//
// Synthetic "package"-kind pseudo-nodes (internal/indexer/resolve.go's
// unexported kindPackage) are excluded from the node scan: they are
// synthetic intra-module-import targets with no owning file (FilePath ==
// "" always), and a naive node-to-file lookup would otherwise roll their
// edges into a phantom "" file (D-08).
//
// "contains" edges and file-level self-edges (an edge whose source and
// target roll up to the SAME file) are excluded from the edge scan, for
// every retained kind — "contains" is symbol-in-file containment, which
// at file granularity is near-universal self-edge noise; an edge between
// two different symbols in the same file is not a dependency between
// files (D-03).
func (e *Engine) FileGraph() (FileGraphResult, error) {
	// Scan one: IterateNodes() over the whole n/ namespace, building the
	// node-id-to-file-path map that scan two's edge aggregation depends
	// on, and the per-file symbol tally and language.
	nodeIt, err := e.reader.IterateNodes()
	if err != nil {
		return FileGraphResult{}, err
	}
	defer nodeIt.Close()

	nodeFile := make(map[string]string)
	fileAggs := make(map[string]*fileAgg)
	var excludedPackageNodes int64

	for nodeIt.Next() {
		n := nodeIt.Node()
		if n.Kind == filegraphKindPackage || n.FilePath == "" {
			excludedPackageNodes++
			continue
		}
		nodeFile[n.Id] = n.FilePath

		agg, ok := fileAggs[n.FilePath]
		if !ok {
			agg = &fileAgg{}
			fileAggs[n.FilePath] = agg
		}
		if n.Kind == goextract.KindFile {
			agg.language = n.Language
			agg.languageFromFileKind = true
			continue
		}
		agg.symbolCount++
		if !agg.languageFromFileKind && agg.language == "" && n.Language != "" {
			agg.language = n.Language
		}
	}
	if err := nodeIt.Err(); err != nil {
		return FileGraphResult{}, err
	}

	// Scan two: IterateEdges("") — the empty prefix, because ten of the
	// eleven edge kinds are retained and only "contains" is dropped, so a
	// per-kind-filtered scan would need to run ten times over.
	edgeIt, err := e.reader.IterateEdges("")
	if err != nil {
		return FileGraphResult{}, err
	}
	defer edgeIt.Close()

	pairCounts := make(map[filePairKey]map[string]int64)
	var excludedContainsEdges, excludedSelfEdges int64

	for edgeIt.Next() {
		edge := edgeIt.Edge()
		if edge.Kind == goextract.RefKindContains {
			excludedContainsEdges++
			continue
		}
		srcFile, ok := nodeFile[edge.Source]
		if !ok {
			continue
		}
		tgtFile, ok := nodeFile[edge.Target]
		if !ok {
			continue
		}
		if srcFile == tgtFile {
			excludedSelfEdges++
			continue
		}
		key := filePairKey{source: srcFile, target: tgtFile}
		counts, ok := pairCounts[key]
		if !ok {
			counts = make(map[string]int64)
			pairCounts[key] = counts
		}
		counts[edge.Kind]++
	}
	if err := edgeIt.Err(); err != nil {
		return FileGraphResult{}, err
	}

	// Materialize nodes, sorted ascending by path.
	paths := make([]string, 0, len(fileAggs))
	for path := range fileAggs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	resultNodes := make([]FileGraphNode, 0, len(paths))
	for _, path := range paths {
		agg := fileAggs[path]
		resultNodes = append(resultNodes, FileGraphNode{
			Path:        path,
			Language:    agg.language,
			SymbolCount: agg.symbolCount,
		})
	}

	// Materialize edges, sorted ascending by source file then target file.
	pairKeys := make([]filePairKey, 0, len(pairCounts))
	for key := range pairCounts {
		pairKeys = append(pairKeys, key)
	}
	sort.Slice(pairKeys, func(i, j int) bool {
		if pairKeys[i].source != pairKeys[j].source {
			return pairKeys[i].source < pairKeys[j].source
		}
		return pairKeys[i].target < pairKeys[j].target
	})
	resultEdges := make([]FileGraphEdge, 0, len(pairKeys))
	for _, key := range pairKeys {
		counts := pairCounts[key]
		var total int64
		for _, c := range counts {
			total += c
		}
		resultEdges = append(resultEdges, FileGraphEdge{
			SourceFile: key.source,
			TargetFile: key.target,
			KindCounts: counts,
			TotalCount: total,
		})
	}

	result := FileGraphResult{
		Nodes:                 resultNodes,
		Edges:                 resultEdges,
		ExcludedPackageNodes:  excludedPackageNodes,
		ExcludedSelfEdges:     excludedSelfEdges,
		ExcludedContainsEdges: excludedContainsEdges,
	}

	// Cycle detection (GRF-04, D-06) runs server-side over the aggregated
	// file adjacency this scan just built — kind-agnostic, since a
	// dependency cycle is a cycle regardless of which edge kinds compose
	// it. Built fresh here alongside everything else FileGraph derives;
	// no separate cache.
	cycleAdj := make(map[string][]string, len(resultEdges))
	for _, edge := range resultEdges {
		cycleAdj[edge.SourceFile] = append(cycleAdj[edge.SourceFile], edge.TargetFile)
	}
	cycleIDs := stronglyConnectedCycles(cycleAdj)

	distinctCycles := make(map[int]struct{})
	for i := range result.Nodes {
		if id, ok := cycleIDs[result.Nodes[i].Path]; ok {
			result.Nodes[i].CycleID = id
			distinctCycles[id] = struct{}{}
		}
	}
	for i := range result.Edges {
		srcID, srcOK := cycleIDs[result.Edges[i].SourceFile]
		tgtID, tgtOK := cycleIDs[result.Edges[i].TargetFile]
		// An edge between two DIFFERENT cycles is not itself part of a
		// cycle and must not be marked.
		result.Edges[i].InCycle = srcOK && tgtOK && srcID == tgtID
	}
	result.CycleCount = len(distinctCycles)

	return result, nil
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
		return nil, notFoundf("query: symbol %q not found", symbol)
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

	// WR-05/WR-02: sort BEFORE applying limit/MaxLimit caps — mirrors
	// Callers/Impact/Affected's deterministic (FilePath, Name, StartLine)
	// ordering rather than leaving Callees at the mercy of
	// IterateEdges's raw Pebble key-range scan order.
	sortLocations(locs)
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
// above), but with the documented test-files-as-leaves pruning rule instead
// of Impact's "expand everything": a dependent that passes isTestSymbol
// is recorded as an affected test AND is a leaf — it is never queued
// for further expansion, so its own dependents can never surface at any
// depth. A non-test dependent is queued for the next hop and is never
// itself recorded as an affected test. There is no golden oracle for
// this command (D-07a); behavior is proved structurally in
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
						// documented test-files-as-leaves semantics: record, but do
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
