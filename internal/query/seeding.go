// Package query — internal/query/seeding.go ports TS CodeGraph 1.3.1's
// named-symbol seeding heuristic H13 (RESEARCH §C.2,
// mcp/tools.js:2477-2562 [VERIFIED: TS 1.3.1 dist — cited from the frozen
// 01-RESEARCH.md capture]): the stage that resolves the agent's named
// query symbols into RWR seeds and gives their files the dominant +50
// score (plan 13, H14). Feeds directly off extractSymbolsFromQuery (H1,
// tokenize.go) and computeGraphRelevance's seedIDs restart vector
// (rwr.go).
//
// H13's rule set, ported verbatim from the RESEARCH §C.2 row:
//   - tokenize the query again (H1), keep only tokens >=3 chars, capped
//     at the first 16 (in scan order)
//   - resolve each token via a full-scan exact-name lookup (getNodesByName
//     — NOT the FTS/gather channels H3-H6 use)
//   - <=3 defs for a name: INJECT ALL of them into the RWR seed set; the
//     "seed tier" (the subset plan 13's +50 named-seed file score keys
//     off) is def0 (the D-04 lowest-Id def, substituting for TS's
//     unordered SELECT per Assumption A3) plus any OTHER co-named def
//     whose caller count is >= 0.25*maxCallers among that name's defs
//   - >3 defs for a name: only the disambiguated subset is injected (and
//     IS the seed tier) — this task lands the small-overload (<=3 defs)
//     branch only; a following task (RESEARCH §C.2/H13's PascalCase
//     type-token corroboration + top-1-by-substance) adds the >3-def
//     branch. Until then, a name resolving to >3 defs is conservatively
//     skipped (not seeded, not added to Names) rather than seeding an
//     unbounded/undisambiguated set.
package query

import (
	"sort"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// H13's exact, cited constants (RESEARCH §C.2/H13).
const (
	seedTokenMinLen          = 3
	seedTokenMaxCount        = 16
	smallOverloadMaxDefs     = 3
	smallOverloadCallerRatio = 0.25
)

// seedName is one query token's full-scan resolution + disambiguation-tier
// result.
type seedName struct {
	// Name is the resolved query token.
	Name string
	// Injected is every def id actually added to the RWR seed set for
	// this name: ALL <=3 defs (small-overload "inject all"), or (a
	// following task) the >3-def disambiguated selection.
	Injected []string
	// Primary is the "seed tier" subset RESEARCH §C.2/H13 names — plan
	// 13's +50 named-seed file score keys off this, not Injected, when a
	// small overload injects more defs than the tier itself contains.
	// Small-overload: def0 + co-named callers>=0.25*maxCallers.
	Primary []string
}

// seedResult is H13's full output: the deterministic RWR seed-node-id set
// (computeGraphRelevance's restart-vector seedIDs input) plus per-name
// tier metadata for plan 13's downstream +50 named-seed file score.
type seedResult struct {
	// SeedIDs is the deduplicated, sorted union of every resolved name's
	// Injected def ids.
	SeedIDs []string
	// Names holds one entry per resolved query token (a token with zero
	// defs is silently skipped, not an error), in the query's first-seen
	// token scan order (D-04 determinism).
	Names []seedName
}

// seedQueryTokens re-tokenizes query via H1's extractSymbolsFromQuery
// (tokenize.go), then applies H13's own extra filter on top: keep only
// tokens with length >=3, capped at the first 16 in scan order.
// extractSymbolsFromQuery already returns tokens in first-seen,
// deduplicated order (D-04) — this preserves that order.
func seedQueryTokens(query string) []string {
	all := extractSymbolsFromQuery(query)
	out := make([]string, 0, seedTokenMaxCount)
	for _, t := range all {
		if len(t) < seedTokenMinLen {
			continue
		}
		out = append(out, t)
		if len(out) >= seedTokenMaxCount {
			break
		}
	}
	return out
}

// resolveDefsByName is H13's getNodesByName-equivalent: a full
// IterateNodes() scan (NOT the FTS/gather channels of H3-H6) collecting
// every node whose Name exactly equals name. TS's own SELECT has no
// ORDER BY (Assumption A3, RESEARCH); this substitutes the codebase-wide
// D-04 lowest-Id-first convention as a deterministic, reproducible order.
func resolveDefsByName(r graphstore.Reader, name string) ([]*schema.Node, error) {
	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var defs []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.Name == name {
			defs = append(defs, n)
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Id < defs[j].Id })
	return defs, nil
}

// smallOverloadSeed implements H13's <=3-defs branch: inject every def
// (already Id-sorted by resolveDefsByName) into the seed set, and compute
// the seed tier as def0 (defs[0], the lowest-Id def) plus any OTHER def
// whose caller count (via BuildReverseAdjacency) is >= 0.25*maxCallers
// among defs' own caller counts. When every def has zero callers,
// maxCallers is 0 and the threshold (0) is trivially met by all —
// consistent with "def0 plus any co-named def meeting the ratio", not a
// special case.
func smallOverloadSeed(r graphstore.Reader, defs []*schema.Node) (injected, primary []string, err error) {
	rev, err := BuildReverseAdjacency(r)
	if err != nil {
		return nil, nil, err
	}

	injected = make([]string, 0, len(defs))
	callers := make(map[string]int, len(defs))
	maxCallers := 0
	for _, d := range defs {
		c := len(rev[d.Id])
		callers[d.Id] = c
		if c > maxCallers {
			maxCallers = c
		}
		injected = append(injected, d.Id)
	}

	threshold := smallOverloadCallerRatio * float64(maxCallers)
	primarySet := map[string]bool{defs[0].Id: true} // def0
	for _, d := range defs {
		if float64(callers[d.Id]) >= threshold {
			primarySet[d.Id] = true
		}
	}
	for _, d := range defs {
		if primarySet[d.Id] {
			primary = append(primary, d.Id)
		}
	}
	return injected, primary, nil
}

// seedNamedSymbols is H13 end-to-end: re-tokenize query (seedQueryTokens),
// resolve each token via a full-scan exact-name lookup
// (resolveDefsByName), and apply the small-overload (<=3 defs)
// disambiguation tier per name — a token that resolves to zero defs is
// silently skipped (not every query token names an existing symbol). A
// name resolving to >3 defs is provisionally skipped (see package doc
// comment); a following task replaces this branch with the >3-def
// type-token-corroborated/top-1-by-substance disambiguation. projectName
// (typically filepath.Base(repoRoot)) is accepted now so this task's
// signature is stable for that task's PascalCase type-token exclusion —
// it is unused until that branch lands. Deterministic throughout (D-04):
// tokens are processed in first-seen scan order, each name's defs are
// Id-sorted, and the returned SeedIDs union is lexicographically sorted.
func seedNamedSymbols(r graphstore.Reader, query, projectName string) (seedResult, error) {
	_ = projectName // consumed by the >3-def branch (following task)

	tokens := seedQueryTokens(query)
	if len(tokens) == 0 {
		return seedResult{}, nil
	}

	var names []seedName
	seedSet := make(map[string]bool)
	for _, tok := range tokens {
		defs, err := resolveDefsByName(r, tok)
		if err != nil {
			return seedResult{}, err
		}
		if len(defs) == 0 || len(defs) > smallOverloadMaxDefs {
			continue
		}

		injected, primary, err := smallOverloadSeed(r, defs)
		if err != nil {
			return seedResult{}, err
		}

		names = append(names, seedName{Name: tok, Injected: injected, Primary: primary})
		for _, id := range injected {
			seedSet[id] = true
		}
	}

	seedIDs := make([]string, 0, len(seedSet))
	for id := range seedSet {
		seedIDs = append(seedIDs, id)
	}
	sort.Strings(seedIDs)

	return seedResult{SeedIDs: seedIDs, Names: names}, nil
}
