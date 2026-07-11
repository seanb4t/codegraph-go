package query

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// buildReverseAdjacency builds an in-memory reverse-adjacency map keyed
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

	var locs []Location
	for it.Next() {
		edge := it.Edge()
		if edge.Kind != goextract.RefKindCalls {
			continue
		}
		target, err := e.reader.GetNode(edge.Target)
		if err != nil {
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
// reverse-adjacency map (built fresh, buildReverseAdjacency), capped at
// limit identically to Callees.
func (e *Engine) Callers(symbol string, limit int) (CallersResult, error) {
	if err := validateLimit(limit); err != nil {
		return CallersResult{}, err
	}

	node, err := e.resolveSymbolNode(symbol)
	if err != nil {
		return CallersResult{}, err
	}

	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return CallersResult{}, err
	}

	var locs []Location
	for _, edge := range rev[node.Id] {
		src, err := e.reader.GetNode(edge.Source)
		if err != nil {
			return CallersResult{}, err
		}
		locs = append(locs, nodeLocation(src))
	}

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
// traverse_test.go's TestImpact doc comment.
func (e *Engine) Impact(symbol string, depth int) (ImpactResult, error) {
	depth = clampDepth(depth)

	node, err := e.resolveSymbolNode(symbol)
	if err != nil {
		return ImpactResult{}, err
	}

	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return ImpactResult{}, err
	}

	visited := map[string]bool{node.Id: true}
	affected := []Location{nodeLocation(node)}
	frontier := []string{node.Id}
	edgeCount := 0

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string
		for _, id := range frontier {
			for _, edge := range rev[id] {
				edgeCount++
				if visited[edge.Source] {
					continue
				}
				visited[edge.Source] = true
				srcNode, err := e.reader.GetNode(edge.Source)
				if err != nil {
					return ImpactResult{}, err
				}
				affected = append(affected, nodeLocation(srcNode))
				next = append(next, edge.Source)
			}
		}
		frontier = next
	}

	return ImpactResult{
		Symbol:    symbol,
		Depth:     depth,
		NodeCount: len(affected),
		EdgeCount: edgeCount,
		Affected:  affected,
	}, nil
}

// isTestSymbol is D-07's test-file heuristic: a node counts as a test
// symbol if its file ends _test.go, or its own name matches Test*/
// Benchmark* naming (covering table-driven subtests and benchmarks
// declared with an otherwise-unconventional file name).
func isTestSymbol(n *schema.Node) bool {
	if strings.HasSuffix(n.FilePath, "_test.go") {
		return true
	}
	return strings.HasPrefix(n.Name, "Test") || strings.HasPrefix(n.Name, "Benchmark")
}

// Affected derives impacted test files/symbols for a set of changed
// files at query time (D-07): no persisted test-coverage edge kind —
// walk the D-04 reverse-adjacency map from every symbol defined in the
// changed files, keeping reverse-caller targets that pass the
// isTestSymbol heuristic. There is no golden oracle for this command
// (D-07a); parity is structural, proved in traverse_test.go against a
// seeded test->symbol calls edge.
func (e *Engine) Affected(files []string) (AffectedResult, error) {
	rev, err := buildReverseAdjacency(e.reader)
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

	var seedIDs []string
	for it.Next() {
		n := it.Node()
		if fileSet[n.FilePath] {
			seedIDs = append(seedIDs, n.Id)
		}
	}
	if err := it.Err(); err != nil {
		return AffectedResult{}, err
	}

	visited := make(map[string]bool)
	var tests []Location
	for _, id := range seedIDs {
		for _, edge := range rev[id] {
			if visited[edge.Source] {
				continue
			}
			target, err := e.reader.GetNode(edge.Source)
			if err != nil {
				return AffectedResult{}, err
			}
			if !isTestSymbol(target) {
				continue
			}
			visited[edge.Source] = true
			tests = append(tests, nodeLocation(target))
		}
	}

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
