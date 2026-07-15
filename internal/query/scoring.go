// Package query — internal/query/scoring.go ports TS CodeGraph 1.3.1's
// per-file scoring, hard-exclusion, and change-surface buried-rescue
// heuristics H14-H16 (RESEARCH §C.2, D-10): the stage that converts
// node-level RWR mass (plan 06, rwr.go) plus the file candidate set into
// the per-file "score" the relevance gate (H17, plan 14) filters/sorts
// on. The live TS dist is no longer readable on this machine (see
// gather.go's package doc comment) — every constant below is cited from
// the frozen 01-RESEARCH.md §C.2 capture, not re-derived from a fresh
// source read:
//
//   - H14 Per-file score tiers         — mcp/tools.js:2632-2647
//   - H15 Hard test/spec exclusion     — mcp/tools.js:2652-2684
//   - H16 Change-surface buried-rescue — mcp/tools.js:2574-2613, 2733-2762
//
// Division of labor / D-02 scoping note: RESEARCH's frozen citations pin
// H14-H16's constants and rule STRUCTURE, but not the exact upstream
// wiring of "which node ids are named-seed/entry/tier-seed" or "how
// fileTermHits is computed" — those come from earlier pipeline stages
// (H11's BFS roots via expand.go, H13's named-symbol seed tiers via
// seeding.go, and H5's per-term-hit tracking via gather.go) that a LATER
// wiring plan (per expand.go's own "primitives now, wiring later"
// precedent) composes together. This file's functions therefore accept
// those sets/maps as caller-supplied parameters rather than re-deriving
// them, keeping scoring.go a pure, Reader-driven primitive independently
// unit-testable against a synthetic index — mirroring expand.go's/
// seeding.go's own discipline.
package query

import (
	"errors"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// --- H14: per-file score tiers ---

// H14's exact, cited constants (RESEARCH §C.2/H14, mcp/tools.js:2632-2647
// [VERIFIED: TS 1.3.1 dist — cited from the frozen RESEARCH capture]):
// named-seed +50, entry (root/named) +10, connected-to-entry +3, other
// +1; keep files with score >= 3.
const (
	fileScoreNamedSeed     = 50.0
	fileScoreEntry         = 10.0
	fileScoreConnected     = 3.0
	fileScoreOther         = 1.0
	fileScoreKeepThreshold = 3.0
)

// classifyNodeTier assigns id one of H14's four tier values: namedSeedIDs
// wins over entryIDs wins over connectedIDs wins over the "other" floor —
// mirroring RESEARCH §C.2/H14's own priority order (named-seed is the
// single highest tier, entry next, connected-to-entry next, everything
// else "other").
func classifyNodeTier(id string, namedSeedIDs, entryIDs, connectedIDs map[string]bool) float64 {
	switch {
	case namedSeedIDs[id]:
		return fileScoreNamedSeed
	case entryIDs[id]:
		return fileScoreEntry
	case connectedIDs[id]:
		return fileScoreConnected
	default:
		return fileScoreOther
	}
}

// computeConnectedToEntry returns the set of node ids with a DIRECT
// RankEdges-filtered edge (either direction — RWR's own adjacency is
// undirected, rwr.go) to a namedSeedIDs or entryIDs node, EXCLUDING ids
// already in either of those two sets (a named-seed/entry node keeps its
// own higher tier; it is never downgraded to "connected").
func computeConnectedToEntry(edges []*schema.Edge, namedSeedIDs, entryIDs map[string]bool) map[string]bool {
	isEntryTier := func(id string) bool { return namedSeedIDs[id] || entryIDs[id] }

	connected := make(map[string]bool)
	for _, e := range edges {
		if !RankEdges[e.Kind] {
			continue
		}
		if isEntryTier(e.Source) && !isEntryTier(e.Target) {
			connected[e.Target] = true
		}
		if isEntryTier(e.Target) && !isEntryTier(e.Source) {
			connected[e.Source] = true
		}
	}
	return connected
}

// computeFileScoreTiers is H14 (RESEARCH §C.2, mcp/tools.js:2632-2647):
// assigns every node in nodeIDs one of the four tier values
// (classifyNodeTier), then rolls each FILE's score up to the HIGHEST
// tier among the nodes it contains — a file with even one named-seed
// node scores 50, regardless of how many "other"-tier nodes it also
// holds; per-file score tiers, not per-node accumulation. Files whose
// rolled-up score is below fileScoreKeepThreshold (3, i.e. every node in
// the file is "other"-tier) are dropped from the returned map entirely
// ("keep files with score >= 3"). WR-04: a nodeIDs entry that no longer
// resolves (already pruned/dangling) is skipped, not an error.
// Deterministic: nodeIDs are walked in sorted order before rolling up.
func computeFileScoreTiers(r graphstore.Reader, nodeIDs []string, edges []*schema.Edge, namedSeedIDs, entryIDs map[string]bool) (map[string]float64, error) {
	connected := computeConnectedToEntry(edges, namedSeedIDs, entryIDs)

	sorted := append([]string(nil), nodeIDs...)
	sort.Strings(sorted)

	fileScores := make(map[string]float64)
	for _, id := range sorted {
		n, err := r.GetNode(id)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if n.FilePath == "" {
			continue
		}
		tier := classifyNodeTier(id, namedSeedIDs, entryIDs, connected)
		if tier > fileScores[n.FilePath] {
			fileScores[n.FilePath] = tier
		}
	}

	for f, s := range fileScores {
		if s < fileScoreKeepThreshold {
			delete(fileScores, f)
		}
	}
	return fileScores, nil
}

// aggregateFileGraphScore sums each node's RWR mass (rwrScores,
// computeGraphRelevance's output, rwr.go) into its file's fileGraphScore
// (RESEARCH §4/§C.2: "fileGraphScore.get(fp) = sum of node-level RWR mass
// for every node in that file") — the per-file mass H16's buried-rescue
// and H17's relevance gate (plan 14) both key off. WR-04: a nodeIDs entry
// that no longer resolves is skipped, not an error. Deterministic:
// nodeIDs are summed in sorted-Id order so float addition order never
// varies run-to-run (D-04) — rwrScores' own values are already rounded
// to 1e-9 (rwr.go); this function does not re-round the summed result.
func aggregateFileGraphScore(r graphstore.Reader, nodeIDs []string, rwrScores map[string]float64) (map[string]float64, error) {
	sorted := append([]string(nil), nodeIDs...)
	sort.Strings(sorted)

	out := make(map[string]float64)
	for _, id := range sorted {
		n, err := r.GetNode(id)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if n.FilePath == "" {
			continue
		}
		out[n.FilePath] += rwrScores[id]
	}
	return out, nil
}
