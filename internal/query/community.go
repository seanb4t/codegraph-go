package query

import (
	"math/rand/v2"
	"sort"

	"gonum.org/v1/gonum/graph/community"
	"gonum.org/v1/gonum/graph/simple"
)

// community.go computes file-level community membership over an
// undirected weighted file adjacency (GRF-06, GRF-08). It does NOT read
// the store, does NOT know about edge kinds, and never runs
// client-side — it takes the already-aggregated node/edge rollup
// FileGraph() built and returns a community assignment, nothing more
// (mirrors filegraph_cycles.go's contract exactly).
//
// The assignment is computed FRESH on every FileGraph() call, never
// cached, never persisted (D-15). The reserved Node 50-59 range
// (community_id = 50) stays unwritten in this phase — it is exercised
// only if GRF-09's measured threshold fails and the phase executes the
// documented promote fallback (corpora/graph-cluster-threshold.json's
// onFailure).

// communitySeed1 and communitySeed2 are DETERMINISM constants — they
// seed the fixed math/rand/v2 source that drives Louvain's move
// ordering (D-02b). They are chosen once, committed in the clear, and
// never changed; they are NOT a secret (11-RESEARCH V6) — nothing
// cryptographic depends on them, and changing them would only change
// which of several equally-valid modularity-tying partitions wins a
// tie, never the correctness of the result. The literal bytes spell
// "codegrap" / "louvain1" in ASCII, chosen for readability, not
// entropy.
const (
	communitySeed1 uint64 = 0x636f646567726170
	communitySeed2 uint64 = 0x6c6f757661696e31
)

// communityResolution is the Louvain resolution parameter (D-01):
// 1.0, the standard, unweighted-toward-either-extreme modularity
// resolution.
const communityResolution = 1.0

// communityOptions is a TEST-ONLY seam (D-04): production code always
// runs with communityDefaults, in which both bools are false. Task 3's
// RED-control tests flip reverseInsertion and skipCanonicalRelabel
// through assignCommunitiesWith directly to prove the fixed seed and
// the canonical relabel step are load-bearing, not cosmetic.
type communityOptions struct {
	seed1, seed2 uint64
	// reverseInsertion, when true, assigns gonum node ids from the
	// REVERSED sorted-path slice instead of the ascending one — the
	// perturbation D-04's RED control and D-02c's canonical-relabel
	// demonstration both use. Never true in production.
	reverseInsertion bool
	// skipCanonicalRelabel, when true, labels communities by gonum's own
	// internal community index + 1 instead of by smallest-member-path
	// order — demonstrating that canonical relabeling, not gonum
	// itself, is what neutralises insertion order (D-02c). Never true
	// in production.
	skipCanonicalRelabel bool
}

// communityDefaults is the only package-level var in this file — the
// production options value, with both test-only seam bools left false
// (D-15's fresh-per-call, no-shared-state discipline: this is
// immutable configuration, never mutated after init, and carries no
// per-call state).
var communityDefaults = communityOptions{seed1: communitySeed1, seed2: communitySeed2}

// AssignCommunities computes a 1-based canonical community id for every
// node in nodes, from the undirected weighted adjacency edges encodes
// (GRF-06). It is exported so tools/graphcluster (11-02) can time it in
// isolation against the GRF-09 threshold. Every input node receives
// exactly one id in the returned map (D-03) — 0 is never emitted once
// this function runs; a node absent from the returned map means it was
// never passed in, not "not computed".
func AssignCommunities(nodes []FileGraphNode, edges []FileGraphEdge) map[string]int {
	return assignCommunitiesWith(nodes, edges, communityDefaults)
}

// assignCommunitiesWith is AssignCommunities' implementation, taking an
// explicit communityOptions so tests can perturb the two test-only
// seam fields. Production code (traverse.go) calls AssignCommunities
// only, never this function directly.
func assignCommunitiesWith(nodes []FileGraphNode, edges []FileGraphEdge, opts communityOptions) map[string]int {
	result := make(map[string]int, len(nodes))
	if len(nodes) == 0 {
		return result
	}

	// Determinism mechanism (a): nodes are inserted in sorted,
	// byte-wise file-path order — never Go map-iteration order — with
	// sequential int64 ids (D-02a). Encoding is never case-folded or
	// normalised: sort.Strings is a pure byte-wise comparison.
	paths := make([]string, len(nodes))
	for i, n := range nodes {
		paths[i] = n.Path
	}
	sort.Strings(paths)

	idByPath := make(map[string]int64, len(paths))
	pathByID := make([]string, len(paths))
	g := simple.NewWeightedUndirectedGraph(0, 0)
	if opts.reverseInsertion {
		for i, j := 0, len(paths)-1; j >= 0; i, j = i+1, j-1 {
			p := paths[j]
			idByPath[p] = int64(i)
			pathByID[i] = p
			g.AddNode(simple.Node(int64(i)))
		}
	} else {
		for i, p := range paths {
			idByPath[p] = int64(i)
			pathByID[i] = p
			g.AddNode(simple.Node(int64(i)))
		}
	}

	weights := undirectedPairWeights(idByPath, edges)
	pairs := make([][2]int64, 0, len(weights))
	for pair := range weights {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}
		return pairs[i][1] < pairs[j][1]
	})
	for _, pair := range pairs {
		w := weights[pair]
		g.SetWeightedEdge(g.NewWeightedEdge(simple.Node(pair[0]), simple.Node(pair[1]), w))
	}

	// Determinism mechanism (b): a fixed math/rand/v2 PCG source drives
	// Louvain's move ordering.
	reduced := community.Modularize(g, communityResolution, rand.NewPCG(opts.seed1, opts.seed2))

	members := make([][]string, 0)
	for _, comm := range reduced.Communities() {
		group := make([]string, len(comm))
		for i, n := range comm {
			group[i] = pathByID[n.ID()]
		}
		members = append(members, group)
	}

	// Determinism mechanism (c): canonical relabeling by smallest
	// member path neutralises gonum's own internal community numbering
	// (D-02c) — skipped only by the test-only seam.
	var labels map[string]int
	if opts.skipCanonicalRelabel {
		labels = make(map[string]int, len(nodes))
		for idx, group := range members {
			for _, p := range group {
				labels[p] = idx + 1
			}
		}
	} else {
		labels = canonicalCommunityLabels(members)
	}
	for path, id := range labels {
		result[path] = id
	}
	return result
}

// undirectedPairWeights folds FileGraph's directed, per-file-pair edges
// into an unordered adjacency keyed by the ascending gonum node id pair,
// SUMMING both directions of a file pair's TotalCount — measured this
// session: simple.WeightedUndirectedGraph.SetWeightedEdge REPLACES an
// existing pair's weight rather than accumulating it, so the sum must
// happen here, before any SetWeightedEdge call. Self-edges, zero-or-
// negative counts, and edges naming an endpoint absent from idByPath
// (a defensive guard; FileGraph never produces one) are all skipped.
func undirectedPairWeights(idByPath map[string]int64, edges []FileGraphEdge) map[[2]int64]float64 {
	weights := make(map[[2]int64]float64)
	for _, e := range edges {
		if e.TotalCount <= 0 {
			continue
		}
		u, ok := idByPath[e.SourceFile]
		if !ok {
			continue
		}
		v, ok := idByPath[e.TargetFile]
		if !ok {
			continue
		}
		if u == v {
			continue
		}
		lo, hi := u, v
		if lo > hi {
			lo, hi = hi, lo
		}
		weights[[2]int64{lo, hi}] += float64(e.TotalCount)
	}
	return weights
}

// canonicalCommunityLabels renumbers a set of communities (each a slice
// of member paths) 1..N, ordered by each community's lexically-smallest
// (byte-wise) member path — so two equal partitions compare equal
// regardless of gonum's own internal numbering (D-02c). Every member of
// every input community appears exactly once in the returned map.
func canonicalCommunityLabels(members [][]string) map[string]int {
	type keyed struct {
		smallest string
		group    []string
	}
	keyedGroups := make([]keyed, len(members))
	for i, group := range members {
		cp := make([]string, len(group))
		copy(cp, group)
		sort.Strings(cp)
		smallest := ""
		if len(cp) > 0 {
			smallest = cp[0]
		}
		keyedGroups[i] = keyed{smallest: smallest, group: cp}
	}
	sort.Slice(keyedGroups, func(i, j int) bool {
		return keyedGroups[i].smallest < keyedGroups[j].smallest
	})

	labels := make(map[string]int)
	for idx, kg := range keyedGroups {
		for _, p := range kg.group {
			labels[p] = idx + 1
		}
	}
	return labels
}
