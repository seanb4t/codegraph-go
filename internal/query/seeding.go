// Package query — internal/query/seeding.go implements the named-symbol
// seeding heuristic H13 (RESEARCH §C.2, mcp/tools.js:2477-2562 [cited
// from the frozen 01-RESEARCH.md capture]): the stage that resolves the agent's named
// query symbols into RWR seeds and gives their files the dominant +50
// score (plan 13, H14). Feeds directly off extractSymbolsFromQuery (H1,
// tokenize.go) and computeGraphRelevance's seedIDs restart vector
// (rwr.go).
//
// H13's rule set, transcribed from the RESEARCH §C.2 row:
//   - tokenize the query again (H1), keep only tokens >=3 chars, capped
//     at the first 16 (in scan order)
//   - resolve each token via a full-scan exact-name lookup (getNodesByName
//     — NOT the FTS/gather channels H3-H6 use)
//   - <=3 defs for a name: INJECT ALL of them into the RWR seed set; the
//     "seed tier" (the subset plan 13's +50 named-seed file score keys
//     off) is def0 (the D-04 lowest-Id def, substituting for the
//     documented unordered SELECT per Assumption A3) plus any OTHER co-named def
//     whose caller count is >= 0.25*maxCallers among that name's defs
//   - >3 defs for a name: only the disambiguated subset is injected (and
//     IS the seed tier, no further split) — PascalCase type tokens from
//     the query (excluding the project name) corroborate up to 4 defs by
//     matching a def's OWNING type's name (via the contains index,
//     traverse.go's buildContainsIndex); if none corroborate, the single
//     def with the greatest "body substance" wins
//
// Divergence (D-02, no verbatim source survives for these specifics —
// the original source is no longer readable on this machine, see gather.go's
// package doc comment for the same constraint): the RESEARCH capture pins
// H13's constants and branch structure but not (a) the exact
// "body-substance" measure the documented design uses to rank a large-overload def with no
// corroborating type token, or (b) the exact mechanism it uses to
// correlate a PascalCase type token with an overloaded def. This plan's
// own, documented design:
//   - body substance = a def's own line span (EndLine-StartLine+1) — a
//     cheap, Reader-only proxy for "how much implementation a def
//     contains" without a second disk read (this function stays a pure
//     graphstore.Reader-driven algorithm, mirroring rwr.go/expand.go/
//     gather.go's discipline)
//   - corroboration = a def's OWNING type (the type that "contains" it,
//     per traverse.go's buildContainsIndex) has a Name matching one of
//     the query's PascalCase type tokens. A def with no owning type (a
//     free function, not a method) never corroborates via a type token —
//     it can only win via the top-1-by-substance fallback. Deliberately
//     does NOT also match a def's own Name against the type-token set:
//     the resolved query token itself is frequently PascalCase-shaped
//     (e.g. "Process"), which would otherwise trivially self-corroborate
//     every def sharing that name and defeat the disambiguation entirely.
//   - project name = the caller-supplied projectName string (typically
//     filepath.Base(repoRoot)), excluded case-insensitively from the
//     PascalCase type-token set before corroboration runs.
package query

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// H13's exact, cited constants (RESEARCH §C.2/H13).
const (
	seedTokenMinLen              = 3
	seedTokenMaxCount            = 16
	smallOverloadMaxDefs         = 3
	smallOverloadCallerRatio     = 0.25
	largeOverloadCorroboratedCap = 4
)

// pascalCaseTokenPattern matches a whole token shaped like a type name —
// an initial uppercase letter followed by any run of letters/digits (no
// separators; extractSymbolsFromQuery has already split compounds into
// individual identifier-shaped tokens by the time this runs).
var pascalCaseTokenPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// seedName is one query token's full-scan resolution + disambiguation-tier
// result.
type seedName struct {
	// Name is the resolved query token.
	Name string
	// Injected is every def id actually added to the RWR seed set for
	// this name: ALL <=3 defs (small-overload "inject all"), or the >3-def
	// disambiguated selection (large-overload).
	Injected []string
	// Primary is the "seed tier" subset RESEARCH §C.2/H13 names — plan
	// 13's +50 named-seed file score keys off this, not Injected, when a
	// small overload injects more defs than the tier itself contains.
	// Small-overload: def0 + co-named callers>=0.25*maxCallers.
	// Large-overload: identical to Injected — the disambiguated selection
	// IS the tier once >3 defs force a cut.
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

// pascalCaseTypeTokens extracts H13's PascalCase type-token bias set from
// query — every extractSymbolsFromQuery token shaped like a type name
// (initial uppercase, no separators), excluding projectName
// (case-insensitive) so a query that simply names the project itself
// never spuriously corroborates an overload. Unlike seedQueryTokens, this
// is NOT length- or count-capped — H13's biasing signal is a separate
// concern from the >=3/<=16 name-resolution filter.
func pascalCaseTypeTokens(query, projectName string) []string {
	all := extractSymbolsFromQuery(query)
	var out []string
	for _, t := range all {
		if !pascalCaseTokenPattern.MatchString(t) {
			continue
		}
		if projectName != "" && strings.EqualFold(t, projectName) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// resolveDefsByName is H13's getNodesByName-equivalent: a full
// IterateNodes() scan (NOT the FTS/gather channels of H3-H6) collecting
// every node whose Name exactly equals name. The documented SELECT has no
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

// bodySubstance is this plan's documented body-substance proxy (package
// doc comment Divergence note): a def's own line span, a cheap
// Reader-only measure of "how much implementation this def contains"
// without a second disk read.
func bodySubstance(n *schema.Node) int {
	span := int(n.EndLine) - int(n.StartLine) + 1
	if span < 0 {
		return 0
	}
	return span
}

// topBySubstance returns the id of the def with the greatest bodySubstance
// among defs. defs is assumed already Id-sorted (resolveDefsByName's
// contract); ties are resolved by keeping the first (lowest-Id) def seen
// with the max substance, matching the codebase-wide D-04 tie-break.
func topBySubstance(defs []*schema.Node) string {
	best := defs[0]
	bestSubstance := bodySubstance(best)
	for _, d := range defs[1:] {
		if s := bodySubstance(d); s > bestSubstance {
			best = d
			bestSubstance = s
		}
	}
	return best.Id
}

// corroboratedDefs is this plan's documented type-token corroboration
// mechanism (package doc comment Divergence note): a def is corroborated
// if its OWNING type's Name (via traverse.go's buildContainsIndex — the
// type that "contains" this def as a method) matches one of typeTokens. A
// def with no owning type never corroborates. defs is assumed already
// Id-sorted; returned ids preserve that order (deterministic).
func corroboratedDefs(r graphstore.Reader, defs []*schema.Node, typeTokens []string) ([]string, error) {
	if len(typeTokens) == 0 {
		return nil, nil
	}
	wanted := make(map[string]bool, len(typeTokens))
	for _, t := range typeTokens {
		wanted[t] = true
	}

	_, methodOwner, err := buildContainsIndex(r)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, d := range defs {
		ownerID, ok := methodOwner[d.Id]
		if !ok {
			continue
		}
		owner, err := r.GetNode(ownerID)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue // WR-04: dangling owner reference, not an error
			}
			return nil, err
		}
		if wanted[owner.Name] {
			out = append(out, d.Id)
		}
	}
	return out, nil
}

// largeOverloadSeed implements H13's >3-defs branch: prefer defs
// corroborated by a query PascalCase type token (capped at 4, in
// defs' Id-sorted order); when none corroborate, fall back to the single
// def with the greatest body substance. The selected subset both is
// injected into the seed set AND is the seed tier (no further split, per
// RESEARCH §C.2/H13).
func largeOverloadSeed(r graphstore.Reader, defs []*schema.Node, typeTokens []string) ([]string, error) {
	corroborated, err := corroboratedDefs(r, defs, typeTokens)
	if err != nil {
		return nil, err
	}
	if len(corroborated) > 0 {
		if len(corroborated) > largeOverloadCorroboratedCap {
			corroborated = corroborated[:largeOverloadCorroboratedCap]
		}
		return corroborated, nil
	}
	return []string{topBySubstance(defs)}, nil
}

// seedNamedSymbols is H13 end-to-end: re-tokenize query (seedQueryTokens),
// resolve each token via a full-scan exact-name lookup
// (resolveDefsByName), and apply the small-overload (<=3 defs) or
// large-overload (>3 defs) disambiguation tier per name — a token that
// resolves to zero defs is silently skipped (not every query token names
// an existing symbol). projectName (typically filepath.Base(repoRoot))
// is excluded from the PascalCase type-token bias set before large-overload
// corroboration runs. Deterministic throughout (D-04): tokens are
// processed in first-seen scan order, each name's defs are Id-sorted, and
// the returned SeedIDs union is lexicographically sorted.
func seedNamedSymbols(r graphstore.Reader, query, projectName string) (seedResult, error) {
	tokens := seedQueryTokens(query)
	if len(tokens) == 0 {
		return seedResult{}, nil
	}
	typeTokens := pascalCaseTypeTokens(query, projectName)

	var names []seedName
	seedSet := make(map[string]bool)
	for _, tok := range tokens {
		defs, err := resolveDefsByName(r, tok)
		if err != nil {
			return seedResult{}, err
		}
		if len(defs) == 0 {
			continue
		}

		var injected, primary []string
		if len(defs) <= smallOverloadMaxDefs {
			injected, primary, err = smallOverloadSeed(r, defs)
		} else {
			injected, err = largeOverloadSeed(r, defs, typeTokens)
			primary = injected
		}
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
