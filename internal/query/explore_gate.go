// Package query — internal/query/explore_gate.go ports TS CodeGraph
// 1.3.1's final file selection+ordering stage of the explore pipeline:
// H17 (the EXPL-03 relevance gate), H18 (the 5-tier file sort), and H19
// (central-file selection). RESEARCH §C.2/§4 (cited, [VERIFIED: TS 1.3.1
// dist]):
//
//   - H17 Relevance GATE — mcp/tools.js:2763-2783
//   - H18 5-tier file sort — mcp/tools.js:2823-2863
//   - H19 Central-file selection — mcp/tools.js:2716-2720
//
// RESEARCH's single most emphasized pitfall for this stage: H17 is a
// 5-way boolean OR, NOT a bare `fileGraphScore >= maxGraph*0.06`
// threshold (D-08). A single-clause port under-selects files TS would
// keep (e.g. a named-by-agent file with near-zero RWR mass). Every
// clause below is ported independently-sufficient, exactly mirroring
// TS's own `||` chain.
//
// Division of labor / D-02 scoping note (same precedent as scoring.go):
// RESEARCH's frozen citations pin H17-H19's constants and rule
// STRUCTURE, but the functions below accept the upstream per-file
// score/flag maps (plan 13's fileScores/fileGraphScore/rescued, plus a
// caller-supplied fileTermHits and fileNodeCounts) as parameters rather
// than deriving them internally — the actual wiring into Explore() is a
// later plan's job, mirroring expand.go's/scoring.go's own "primitives
// now, wiring later" discipline.
package query

import (
	"sort"
)

// --- H17: relevance gate ---

// fileRelevanceGateMassFraction is H17's clause-(1) threshold — the SAME
// cited 6% (buriedRescueMassFraction, scoring.go) H16 uses for its own
// buried test. RESEARCH's adjacent citations (mcp/tools.js:2733-2762 for
// H16, :2763-2783 for H17) show this is one constant reused by both
// heuristics, not two independently-tuned values.
const fileRelevanceGateMassFraction = buriedRescueMassFraction

// fileRelevanceGateMinTermHits is H17's clause-(5) threshold: >= 2
// distinct query-term hits.
const fileRelevanceGateMinTermHits = 2

// fileRelevanceGateMinFiles is the "never prunes below 2 files" guard
// (RESEARCH §4: "if (gated.length >= 2) relevantFiles = gated;") — when
// gating would leave fewer than this many files, the gate is not applied
// at all and the pre-gate set is returned unchanged.
const fileRelevanceGateMinFiles = 2

// fileRelevanceGate is H17 (RESEARCH §4/§C.2, mcp/tools.js:2763-2783):
// keeps a file from paths if ANY of 5 independently-sufficient clauses
// holds:
//
//  1. fileGraphScore[fp] >= maxGraph * 6%     (graph mass)
//  2. centralFiles[fp]                        (H19 central selection)
//  3. fileScores[fp] >= fileScoreEntry         (entry/named-file tier, H14)
//  4. rescuedFiles[fp]                        (H16 change-surface-rescued)
//  5. fileTermHits[fp] >= 2                    (distinct term hits)
//
// The gate is applied ONLY when maxGraph (the max fileGraphScore across
// paths) is > 0 — otherwise the pre-gate set is returned unchanged
// (RESEARCH §4: "if (maxGraph > 0) { ... }"). Even when maxGraph > 0, the
// gated result only REPLACES the working set when it retains at least
// fileRelevanceGateMinFiles (2) files — mirroring groupMatchesByFile's
// (explore.go) cap-not-truncate discipline: this stage narrows, it never
// starves the pipeline down to 0 or 1 files.
//
// Deterministic: paths is walked in caller-supplied order and the
// returned slice preserves that same relative order (a filter, not a
// sort — H18's fiveTierFileSort owns ordering).
func fileRelevanceGate(paths []string, fileScores, fileGraphScore map[string]float64, centralFiles, rescuedFiles map[string]bool, fileTermHits map[string]int) []string {
	var maxGraph float64
	for _, p := range paths {
		if fileGraphScore[p] > maxGraph {
			maxGraph = fileGraphScore[p]
		}
	}
	if maxGraph <= 0 {
		return append([]string(nil), paths...)
	}

	var gated []string
	for _, p := range paths {
		clause1GraphMass := fileGraphScore[p] >= maxGraph*fileRelevanceGateMassFraction
		clause2Central := centralFiles[p]
		clause3EntryNamed := fileScores[p] >= fileScoreEntry
		clause4Rescued := rescuedFiles[p]
		clause5TermHits := fileTermHits[p] >= fileRelevanceGateMinTermHits

		if clause1GraphMass || clause2Central || clause3EntryNamed || clause4Rescued || clause5TermHits {
			gated = append(gated, p)
		}
	}

	if len(gated) >= fileRelevanceGateMinFiles {
		return gated
	}
	return append([]string(nil), paths...)
}

// --- H19: central-file selection ---

// centralFileCap is H19's "1-2 files" ceiling: at most 2 files are ever
// selected as central.
const centralFileCap = 2

// centralFileMinTermHits is H19's hard filter: a file must have at least
// 1 distinct query-term hit to be eligible for central selection at all,
// regardless of how dominant its graph mass is.
const centralFileMinTermHits = 1

// centralFileSelection is H19 (RESEARCH §C.2, mcp/tools.js:2716-2720):
// among paths with >= 1 distinct term hit, selects the top
// centralFileCap (2) by graph mass — these earn the larger whole-file
// render ceiling downstream, and feed fileRelevanceGate's clause (2).
// A file with zero term hits is excluded outright, no matter how high
// its graph mass (the eligibility filter is a hard gate, not part of the
// mass tie-break). Fewer than centralFileCap files are returned when
// fewer than that many qualify — the result is never padded.
// Deterministic: eligible paths are sorted by mass descending, then by
// path ascending (D-04 tie-break; TS's own Map/Set iteration order is
// not reproducible in Go).
func centralFileSelection(paths []string, fileGraphScore map[string]float64, fileTermHits map[string]int) map[string]bool {
	var eligible []string
	for _, p := range paths {
		if fileTermHits[p] >= centralFileMinTermHits {
			eligible = append(eligible, p)
		}
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		a, b := eligible[i], eligible[j]
		if fileGraphScore[a] != fileGraphScore[b] {
			return fileGraphScore[a] > fileGraphScore[b]
		}
		return a < b
	})

	if len(eligible) > centralFileCap {
		eligible = eligible[:centralFileCap]
	}

	central := make(map[string]bool, len(eligible))
	for _, p := range eligible {
		central[p] = true
	}
	return central
}

// --- H18: 5-tier file sort ---

// fileSortMassEpsilonFraction is H18 tier (3)'s tolerance: two files
// whose graph mass differs by no more than 1% of the max are treated as
// TIED on this tier and fall through to tier (4) — a float-jitter-safe
// band (T-01-22), not a bare `>` comparison.
const fileSortMassEpsilonFraction = 0.01

// fiveTierFileSort is H18 (RESEARCH §C.2, mcp/tools.js:2823-2863): orders
// paths by a strict 5-tier precedence, each tier a higher-priority
// discriminator than the next, followed by a deterministic tail:
//
//  1. named-seed file first          (fileScores[fp] >= fileScoreNamedSeed)
//  2. corroborated                   ((entry/named OR central) AND >=2 term hits)
//  3. graph mass, 1%-of-max epsilon  (within epsilon = tie, falls to tier 4)
//  4. term hits                      (higher first)
//  5. !low-value                     (non-test/icon/i18n first, scoring.go isLowValueFile)
//
// then !generated (node.go isGeneratedFile), then score (fileScores),
// then node count (fileNodeCounts), then path ascending — the final
// deterministic tie-break (D-04). TS files carry no stable "Id"; path is
// the only natural per-file identity, so it is this port's documented
// substitute for the node-level lowest-Id tail convention used elsewhere
// in this codebase.
//
// sort.SliceStable so that any residual tie (all tiers AND the full tail
// equal) preserves the caller's input order rather than reordering
// arbitrarily.
func fiveTierFileSort(paths []string, fileScores, fileGraphScore map[string]float64, centralFiles map[string]bool, fileTermHits map[string]int, fileNodeCounts map[string]int) []string {
	out := append([]string(nil), paths...)

	var maxGraph float64
	for _, p := range out {
		if fileGraphScore[p] > maxGraph {
			maxGraph = fileGraphScore[p]
		}
	}
	epsilon := maxGraph * fileSortMassEpsilonFraction

	isNamedSeed := func(p string) bool { return fileScores[p] >= fileScoreNamedSeed }
	isEntryOrCentral := func(p string) bool { return fileScores[p] >= fileScoreEntry || centralFiles[p] }
	isCorroborated := func(p string) bool {
		return isEntryOrCentral(p) && fileTermHits[p] >= fileRelevanceGateMinTermHits
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]

		// Tier 1: named-seed file first.
		if an, bn := isNamedSeed(a), isNamedSeed(b); an != bn {
			return an
		}

		// Tier 2: corroborated (entry/central AND >=2 terms).
		if ac, bc := isCorroborated(a), isCorroborated(b); ac != bc {
			return ac
		}

		// Tier 3: graph mass, with a 1%-of-max epsilon tie band.
		ag, bg := fileGraphScore[a], fileGraphScore[b]
		if diff := ag - bg; diff > epsilon || diff < -epsilon {
			return ag > bg
		}

		// Tier 4: term hits.
		if at, bt := fileTermHits[a], fileTermHits[b]; at != bt {
			return at > bt
		}

		// Tier 5: !low-value (non-test/icon/i18n first).
		if al, bl := isLowValueFile(a), isLowValueFile(b); al != bl {
			return !al
		}

		// Tail: !generated, then score, then node count, then path
		// (D-04 deterministic tie-break).
		if agen, bgen := isGeneratedFile(a), isGeneratedFile(b); agen != bgen {
			return !agen
		}
		if as, bs := fileScores[a], fileScores[b]; as != bs {
			return as > bs
		}
		if an, bn := fileNodeCounts[a], fileNodeCounts[b]; an != bn {
			return an > bn
		}
		return a < b
	})

	return out
}
