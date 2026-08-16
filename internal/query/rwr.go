// Package query — internal/query/rwr.go implements EXPL-02's load-bearing
// algorithm: computeGraphRelevance, the documented Random-Walk-with-Restart (RWR)
// relevance ranker (RESEARCH §3, mcp/tools.js:2321-2386 [cited from the
// frozen RESEARCH capture]). This file is pure — no graphstore.Reader dependency — so
// it is fully unit-testable on synthetic in-memory subgraphs.
package query

import (
	"math"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// RankEdges is the Go RANK_EDGES-equivalent set (RESEARCH §C.1) — plain
// Set membership only, undirected and UNWEIGHTED (no per-kind weights,
// despite the phase description's phrasing — D-09 confirmed). Sourced
// from goextract's shared RefKind*/EdgeKind* constants (one definition,
// never re-declared literals, mirroring goextract's own discipline).
var RankEdges = map[string]bool{
	goextract.RefKindCalls:        true,
	goextract.RefKindReferences:   true,
	goextract.EdgeKindExtends:     true,
	goextract.EdgeKindImplements:  true,
	goextract.EdgeKindOverrides:   true,
	goextract.RefKindInstantiates: true,
	goextract.RefKindReturns:      true,
	goextract.RefKindTypeOf:       true,
	goextract.RefKindImports:      true,
}

// buildRWRAdjacency builds an in-memory undirected adjacency list (index
// positions into nodeIDs, not node ids themselves) from edges whose Kind
// is in RankEdges. Mirrors BuildReverseAdjacency's (traverse.go)
// fresh-per-call, no-cache discipline — a long-lived MCP process must
// never serve a stale graph view — and its WR-04 dangling-edge tolerance:
// an edge whose endpoint is absent from the candidate node set is
// skipped, not an error. A self-loop (i==j) is also excluded. Built fresh
// every call.
func buildRWRAdjacency(nodeIDs []string, edges []*schema.Edge) [][]int {
	idx := make(map[string]int, len(nodeIDs))
	for i, id := range nodeIDs {
		idx[id] = i
	}

	adj := make([][]int, len(nodeIDs))
	for _, e := range edges {
		if !RankEdges[e.Kind] {
			continue
		}
		i, iok := idx[e.Source]
		j, jok := idx[e.Target]
		if !iok || !jok || i == j {
			continue
		}
		adj[i] = append(adj[i], j)
		adj[j] = append(adj[j], i) // undirected — reachable either direction
	}
	return adj
}

// rwrAlpha and rwrIterations are RWR's fixed parameters (RESEARCH §3,
// D-04): FIXED 25 iterations with NO convergence early-exit (an early
// exit could vary run-to-run and break the golden-corpus determinism
// contract), restart probability alpha=0.25.
const (
	rwrAlpha      = 0.25
	rwrIterations = 25

	// rwrScorePrecision rounds final scores to a fixed precision (1e-9,
	// D-04) before they are ever compared/sorted downstream, so float
	// jitter can never reorder results.
	rwrScorePrecision = 1e9
)

// computeGraphRelevance runs a FIXED-25-iteration, undirected, unweighted
// power-iteration Random-Walk-with-Restart (RESEARCH §3, α=0.25) over the
// candidate subgraph (nodeIDs + edges filtered through RankEdges via
// buildRWRAdjacency), restarting to a uniform distribution over the
// seedIDs present in the candidate set (falling back to uniform-over-all
// nodes if no seed lands in the set). A degree-0 (dangling) node retains
// its own mass across iterations rather than losing it.
//
// D-04 determinism: seed iteration order is sorted (never ranged directly
// from the seedIDs map into the restart vector) and every final score is
// rounded to a fixed precision (rwrScorePrecision) before being returned,
// so two runs over the same input always produce bit-identical (post-
// rounding) score maps — the golden-corpus contract this function is the
// load-bearing implementation of.
//
// Precondition (T-01-10, DoS): this function is O(iterations * len(edges))
// and assumes nodeIDs/edges are ALREADY bounded by the caller — the
// upstream subgraph-gathering caps (maxNodes=200 / GLUE_NODE_CAP=60,
// implemented in later plans) are enforced BEFORE this function is invoked,
// not inside it.
func computeGraphRelevance(nodeIDs []string, edges []*schema.Edge, seedIDs map[string]bool) map[string]float64 {
	out := make(map[string]float64)
	n := len(nodeIDs)
	if n == 0 {
		return out
	}

	adj := buildRWRAdjacency(nodeIDs, edges)

	idx := make(map[string]int, n)
	for i, id := range nodeIDs {
		idx[id] = i
	}

	// Restart vector: uniform over seeds present in the candidate set.
	// D-04: sort seed keys first — never range a map directly into the
	// restart vector, so seeding order can never vary run-to-run.
	sortedSeeds := make([]string, 0, len(seedIDs))
	for id := range seedIDs {
		sortedSeeds = append(sortedSeeds, id)
	}
	sort.Strings(sortedSeeds)

	r := make([]float64, n)
	rsum := 0.0
	for _, id := range sortedSeeds {
		if i, ok := idx[id]; ok {
			r[i] = 1
			rsum++
		}
	}
	if rsum == 0 {
		// Fallback: no seed landed in the candidate set — uniform over all.
		for i := range r {
			r[i] = 1
		}
		rsum = float64(n)
	}
	for i := range r {
		r[i] /= rsum
	}

	s := make([]float64, n)
	copy(s, r)
	for iter := 0; iter < rwrIterations; iter++ { // FIXED 25 iterations, no early-exit
		next := make([]float64, n)
		for i, si := range s {
			if si == 0 {
				continue
			}
			d := len(adj[i])
			if d == 0 {
				next[i] += si // dangling: keep its own mass
				continue
			}
			share := si / float64(d)
			for _, j := range adj[i] {
				next[j] += share
			}
		}
		for i := range s {
			s[i] = (1-rwrAlpha)*next[i] + rwrAlpha*r[i]
		}
	}

	for i, id := range nodeIDs {
		out[id] = roundRWRScore(s[i])
	}
	return out
}

// roundRWRScore rounds v to a fixed precision (D-04) so float jitter can
// never reorder a downstream comparison or tie-break.
func roundRWRScore(v float64) float64 {
	return math.Round(v*rwrScorePrecision) / rwrScorePrecision
}

// rwrScoredNode pairs a node id with its RWR score for stable, ordered
// consumption downstream.
type rwrScoredNode struct {
	ID    string
	Score float64
}

// sortRWRScores orders a score map score-descending then Id-ascending
// (D-04's stable tie-break — reuses the codebase-wide lowest-Id
// convention from resolveSymbolNode, traverse.go).
func sortRWRScores(scores map[string]float64) []rwrScoredNode {
	out := make([]rwrScoredNode, 0, len(scores))
	for id, score := range scores {
		out = append(out, rwrScoredNode{ID: id, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
