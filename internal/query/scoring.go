// Package query — internal/query/scoring.go implements the per-file
// scoring, hard-exclusion, and change-surface buried-rescue heuristics
// H14-H16 (RESEARCH §C.2, D-10): the stage that converts node-level RWR
// mass (plan 06, rwr.go) plus the file candidate set into the per-file
// "score" the relevance gate (H17, plan 14) filters/sorts on. The
// original source is no longer readable on this machine (see
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
	"regexp"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// --- H14: per-file score tiers ---

// H14's exact, cited constants (RESEARCH §C.2/H14, mcp/tools.js:2632-2647
// [cited from the frozen RESEARCH capture]):
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

// --- H15: hard test/spec exclusion ---

// isIconOrI18nFile is this plan's documented D-02 substitute for the
// documented icon/i18n low-value-file component of H15 (RESEARCH §C.2/H15 cites the
// RULE — "test/spec/icon/i18n" — but no further source detail on the
// icon/i18n predicate's exact patterns survives the frozen capture, and
// the live TS dist JS is unreadable on this machine, see gather.go's
// package doc comment). A conservative, path-based heuristic mirroring
// isTestFile's own directory-segment + filename-pattern shape: a file is
// icon/i18n low-value when an icon/i18n-named directory segment appears
// in its path, OR its base filename itself is icon/i18n-shaped.
var (
	iconI18nDirNames = map[string]bool{
		"icon": true, "icons": true,
		"i18n": true, "l10n": true,
		"locale": true, "locales": true,
		"translation": true, "translations": true,
	}
	iconI18nFilenamePattern = regexp.MustCompile(`(?i)^(icons?|i18n|l10n|locales?|translations?)[._-]`)
)

func isIconOrI18nFile(filePath string) bool {
	if filePath == "" {
		return false
	}
	norm := strings.ReplaceAll(filePath, "\\", "/")
	segments := strings.Split(norm, "/")
	base := segments[len(segments)-1]
	if iconI18nFilenamePattern.MatchString(base) {
		return true
	}
	for _, seg := range segments[:len(segments)-1] {
		if seg == "" {
			continue
		}
		if iconI18nDirNames[strings.ToLower(seg)] {
			return true
		}
	}
	return false
}

// isLowValueFile is H15's low-value predicate: test/spec (gather.go's
// shared isTestFile, plan 07) OR icon/i18n (isIconOrI18nFile).
func isLowValueFile(filePath string) bool {
	return isTestFile(filePath) || isIconOrI18nFile(filePath)
}

// queryMentionsTest is H15's own exemption-check wording (RESEARCH
// §C.2/H15: "unless query mentions test") — deliberately narrower than
// gather.go's queryMentionsTestOrSpec (H7's "test/spec" wording): the two
// heuristics cite DIFFERENT query substrings in RESEARCH's frozen
// capture, so this is its own function rather than a reuse.
func queryMentionsTest(query string) bool {
	return strings.Contains(strings.ToLower(query), "test")
}

// applyHardTestExclusion is H15 (RESEARCH §C.2, mcp/tools.js:2652-2684):
// drops every low-value (isLowValueFile) file from fileScores UNLESS the
// query mentions "test" (queryMentionsTest) AND at least 2 non-low-value
// files remain in the candidate set — in which case NO file is dropped
// at all (matching the documented short-circuit: the exemption is
// all-or-nothing, not a partial keep). Mutates fileScores in place.
func applyHardTestExclusion(fileScores map[string]float64, query string) {
	var lowValue []string
	nonLowValueCount := 0
	for f := range fileScores {
		if isLowValueFile(f) {
			lowValue = append(lowValue, f)
		} else {
			nonLowValueCount++
		}
	}
	if len(lowValue) == 0 {
		return
	}
	if queryMentionsTest(query) && nonLowValueCount >= 2 {
		return
	}
	for _, f := range lowValue {
		delete(fileScores, f)
	}
}

// --- H16: change-surface buried-rescue ---

// H16's exact, cited constants (RESEARCH §C.2/H16, mcp/tools.js:2574-2613,
// 2733-2762 [cited from the frozen RESEARCH capture]): rescue a
// signature-type file only if genuinely buried
// (fileGraphScore < maxGraph*0.06 AND termHits < 2); a rescued file is
// force-kept with score = max(score, 45).
const (
	buriedRescueMassFraction = 0.06
	buriedRescueMaxTermHits  = 2
	buriedRescueScoreFloor   = 45.0
)

// signatureTypeEdgeKinds is H16's edge-kind filter: a tier-seed
// callable's "signature types" are the nodes it points at via
// references/type_of/returns edges (RESEARCH §C.2/H16 names all three
// explicitly) — the new D-09 edge kinds this heuristic exists to
// exercise.
var signatureTypeEdgeKinds = map[string]bool{
	goextract.RefKindReferences: true,
	goextract.RefKindTypeOf:     true,
	goextract.RefKindReturns:    true,
}

// signatureTypeTargets resolves srcID's OWN references/type_of/returns
// edges (a direct IterateEdges(srcID) scan, mirroring expand.go's
// calleeCallEdges) to their Target node ids, deterministically ordered
// (sorted).
func signatureTypeTargets(r graphstore.Reader, srcID string) ([]string, error) {
	it, err := r.IterateEdges(srcID)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var out []string
	for it.Next() {
		e := it.Edge()
		if !signatureTypeEdgeKinds[e.Kind] {
			continue
		}
		out = append(out, e.Target)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// isGenuinelyBuried is H16's buried test (RESEARCH §C.2/H16): a file is
// "genuinely buried" only when its graph mass is under 6% of the maximum
// AND it has fewer than 2 distinct query-term hits — BOTH conditions,
// not either (a file with real term hits, or real graph mass, is not
// "buried" no matter how low the other signal is).
func isGenuinelyBuried(filePath string, fileGraphScore map[string]float64, maxGraph float64, fileTermHits map[string]int) bool {
	return fileGraphScore[filePath] < maxGraph*buriedRescueMassFraction && fileTermHits[filePath] < buriedRescueMaxTermHits
}

// applyBuriedRescue is H16 (RESEARCH §C.2, mcp/tools.js:2574-2613,
// 2733-2762): for each tier-seed callable id in tierSeedIDs, follows its
// signature-type edges (signatureTypeTargets) to their target nodes'
// files, and rescues (isGenuinelyBuried) ONLY the genuinely-buried
// ones — a signature-type file with enough graph mass or term hits is
// left alone entirely, not partially rescued. A rescued file is
// force-kept in fileScores at score = max(existing, 45) (mutated in
// place — a file that was already dropped by H15/never scored by H14 is
// ADDED to fileScores at exactly 45, since max(0, 45) == 45) and
// recorded in the returned rescued set, which the H17 gate consumes as
// one of its 5-way OR admission conditions ("changeSurfaceFiles"). WR-04:
// a signature-type edge whose Target no longer resolves is skipped, not
// an error. Deterministic: tierSeedIDs are walked in sorted order and
// each id's own targets are already sorted (signatureTypeTargets).
func applyBuriedRescue(r graphstore.Reader, tierSeedIDs []string, fileGraphScore map[string]float64, maxGraph float64, fileTermHits map[string]int, fileScores map[string]float64) (map[string]bool, error) {
	sorted := append([]string(nil), tierSeedIDs...)
	sort.Strings(sorted)

	rescued := make(map[string]bool)
	for _, seedID := range sorted {
		targets, err := signatureTypeTargets(r, seedID)
		if err != nil {
			return nil, err
		}
		for _, tid := range targets {
			n, err := r.GetNode(tid)
			if err != nil {
				if errors.Is(err, graphstore.ErrNotFound) {
					continue
				}
				return nil, err
			}
			if n.FilePath == "" {
				continue
			}
			if !isGenuinelyBuried(n.FilePath, fileGraphScore, maxGraph, fileTermHits) {
				continue
			}
			rescued[n.FilePath] = true
			if fileScores[n.FilePath] < buriedRescueScoreFloor {
				fileScores[n.FilePath] = buriedRescueScoreFloor
			}
		}
	}
	return rescued, nil
}
