// Package query — internal/query/gather.go ports TS CodeGraph 1.3.1's
// hybrid candidate-gathering heuristics H3-H6 (RESEARCH §C.2,
// context/index.js:449-606 [VERIFIED: TS 1.3.1 dist — the constants
// below are cited from the frozen 01-RESEARCH.md capture; the live dist
// JS itself is no longer present on this machine, only its .d.ts type
// declarations remain, so this port works from RESEARCH's pinned
// constants rather than a fresh source read]): three independently-
// scored channels feeding explore's RWR candidate set, merged
// max-score-wins. This REPLACES the naive lexical matchNodes as
// explore's input construction (RESEARCH Pitfall 1) — a later plan (10)
// wires this into Explore() and extends this file with H7+ rerankers;
// this plan lands only H3-H6 plus the shared isTestFile path predicate
// H7 (plan 10) and the relevance gate (plan 14) both reuse.
package query

import (
	"regexp"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// gatherChannelKind names which of H3/H4/H5 surfaced a gatherCandidate —
// gatherMerge (H6) unions this set across every channel that hit the
// same node id, so downstream heuristics (plan 10) can see provenance,
// not just the winning score.
type gatherChannelKind string

const (
	gatherChannelExactName   gatherChannelKind = "exact_name"       // H3
	gatherChannelTitlePrefix gatherChannelKind = "titlecase_prefix" // H4
	gatherChannelFTS         gatherChannelKind = "fts"              // H5
)

// gatherCandidate is one scored node in the hybrid gather's candidate
// set (RESEARCH §C.2, H3-H6) — node + accumulated score + which
// channel(s) surfaced it.
type gatherCandidate struct {
	Node     *schema.Node
	Score    float64
	Channels map[gatherChannelKind]bool
}

const (
	// channel1BaseScore is H3's per-exact-name-match score before the
	// co-location boost is added. RESEARCH §C.2/H3 pins only the boost
	// delta (channel1CoLocationBoost) — the flat per-match base was not
	// captured verbatim before the TS install's dist JS became
	// unreadable on this machine (see the package doc comment), so a
	// documented default is used, chosen to match channel3BaseScore for
	// cross-channel scale consistency.
	channel1BaseScore = 10.0

	// channel1CoLocationBoost is H3's exact, cited constant:
	// +20*(distinctSymbolsInFile-1) (RESEARCH §C.2/H3,
	// context/index.js:449-478).
	channel1CoLocationBoost = 20.0

	// defaultSearchLimit is H11's explore override (searchLimit:8,
	// mcp/tools.js:2422-2427) — Channel 1 trims its result set to
	// searchLimit*2.
	defaultSearchLimit = 8

	// channel2PrefixBase, channel2BrevityMax and channel2BrevityDiv are
	// H4's exact, cited constants: score = 15 + max(0,
	// 10-(nameLen-prefixLen)/3) (RESEARCH §C.2/H4,
	// context/index.js:485-528).
	channel2PrefixBase = 15.0
	channel2BrevityMax = 10.0
	channel2BrevityDiv = 3.0

	// channel3BaseScore mirrors channel1BaseScore's documented-default
	// reasoning: RESEARCH §C.2/H5 pins only the +5*(termHits-1)
	// multi-term boost, not the single-hit base.
	channel3BaseScore = 10.0

	// channel3MultiTermBoost is H5's exact, cited constant:
	// +5*(termHits-1) (RESEARCH §C.2/H5, context/index.js:530-575).
	channel3MultiTermBoost = 5.0

	// channel3ImportKind mirrors TS's "import" node Kind for H5's
	// exclusion rule. No priority-4 extractor in this repo currently
	// emits an "import" Node.Kind (imports are captured as
	// goextract.RefKindImports EDGES, not nodes — see
	// internal/indexer/goextract/types.go), so this exclusion is
	// presently a no-op against real indexes; ported verbatim for
	// behavioral parity and so a future extractor never silently slips
	// import-declaration nodes into the FTS channel.
	channel3ImportKind = "import"
)

// definitionKinds is H4's Channel-2 kind filter — TS's
// "class/interface/struct/trait/protocol/enum/type_alias" set. In this
// codebase's Kind vocabulary, every priority-4 extractor already
// collapses class/struct/trait/protocol/enum into goextract.KindStruct
// (see e.g. javaextract/types.go, pyextract/types.go, tsextract/types.go
// doc comments: "Reusing KindStruct rather than minting a new class kind
// keeps struct/class-shaped downstream logic unified") — there is no
// separate "class"/"enum"/"trait"/"protocol" Kind value anywhere in this
// repo's extractors to have missed. So the full TS set maps onto exactly
// three Go Kind values: KindStruct + KindInterface + KindTypeAlias (a
// documented D-02 consolidation, not a dropped kind).
var definitionKinds = map[string]bool{
	goextract.KindStruct:    true,
	goextract.KindInterface: true,
	goextract.KindTypeAlias: true,
}

// sortGatherCandidates orders candidates score-descending then
// Id-ascending (D-04's codebase-wide stable tie-break, reused verbatim
// from resolveSymbolNode/sortRWRScores).
func sortGatherCandidates(c []gatherCandidate) {
	sort.SliceStable(c, func(i, j int) bool {
		if c[i].Score != c[j].Score {
			return c[i].Score > c[j].Score
		}
		return c[i].Node.Id < c[j].Node.Id
	})
}

// gatherChannel1 is H3: exact-name lookup + co-location boost (RESEARCH
// §C.2, context/index.js:449-478). Every node whose Name exactly equals
// one of symbols (extractSymbolsFromQuery's output) is a match, scored
// channel1BaseScore. A file that co-locates >1 DISTINCT query symbol
// earns every match in that file an additional
// +20*(distinctSymbolsInFile-1) boost. Results are trimmed to
// searchLimit*2 (searchLimit<=0 falls back to defaultSearchLimit),
// highest score first.
func gatherChannel1(r graphstore.Reader, symbols []string, searchLimit int) ([]gatherCandidate, error) {
	if len(symbols) == 0 {
		return nil, nil
	}
	if searchLimit <= 0 {
		searchLimit = defaultSearchLimit
	}
	want := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		want[s] = true
	}

	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var matches []*schema.Node
	fileSymbols := make(map[string]map[string]bool)
	for it.Next() {
		n := it.Node()
		if !want[n.Name] {
			continue
		}
		matches = append(matches, n)
		if n.FilePath != "" {
			set, ok := fileSymbols[n.FilePath]
			if !ok {
				set = make(map[string]bool)
				fileSymbols[n.FilePath] = set
			}
			set[n.Name] = true
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	candidates := make([]gatherCandidate, 0, len(matches))
	for _, n := range matches {
		score := channel1BaseScore
		if n.FilePath != "" {
			if distinct := len(fileSymbols[n.FilePath]); distinct > 1 {
				score += channel1CoLocationBoost * float64(distinct-1)
			}
		}
		candidates = append(candidates, gatherCandidate{
			Node:     n,
			Score:    score,
			Channels: map[gatherChannelKind]bool{gatherChannelExactName: true},
		})
	}

	sortGatherCandidates(candidates)
	limit := searchLimit * 2
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

// titlecase upper-cases s's first rune, leaving the rest unchanged — H4's
// "make this query token a class-name-shaped prefix" transform (e.g.
// "widget" -> "Widget").
func titlecase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// gatherChannel2 is H4: titlecase definition-prefix search (RESEARCH
// §C.2, context/index.js:485-528). Every symbols token is titlecased
// into a prefix; every definitionKinds node whose Name has that prefix
// scores 15 + brevityBonus, brevityBonus = max(0,
// 10-(nameLen-prefixLen)/3) — a name close in length to the prefix
// (near-exact match) scores near the full +10 bonus, a much longer name
// scores less.
//
// Divergence (D-02, inherited from 01-03-SUMMARY.md): TS's stem-variant
// expansion (getStemVariants) is NOT applied here — 01-03 deliberately
// deferred stem-variant support from extractSymbolsFromQuery/
// extractSearchTerms entirely (no stub parameter), so this channel
// searches the literal titlecased symbol tokens only. A follow-on plan
// can add stem expansion here once getStemVariants is ported.
func gatherChannel2(r graphstore.Reader, symbols []string) ([]gatherCandidate, error) {
	if len(symbols) == 0 {
		return nil, nil
	}
	prefixes := make([]string, 0, len(symbols))
	seen := make(map[string]bool)
	for _, s := range symbols {
		if s == "" {
			continue
		}
		p := titlecase(s)
		if !seen[p] {
			seen[p] = true
			prefixes = append(prefixes, p)
		}
	}
	if len(prefixes) == 0 {
		return nil, nil
	}

	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var candidates []gatherCandidate
	for it.Next() {
		n := it.Node()
		if !definitionKinds[n.Kind] {
			continue
		}
		for _, p := range prefixes {
			if !strings.HasPrefix(n.Name, p) {
				continue
			}
			brevity := channel2BrevityMax - float64(len(n.Name)-len(p))/channel2BrevityDiv
			if brevity < 0 {
				brevity = 0
			}
			candidates = append(candidates, gatherCandidate{
				Node:     n,
				Score:    channel2PrefixBase + brevity,
				Channels: map[gatherChannelKind]bool{gatherChannelTitlePrefix: true},
			})
			break // one match per node — first matching prefix (deterministic scan order) wins
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	sortGatherCandidates(candidates)
	return candidates, nil
}

// --- isTestFile (RESEARCH §5) ---

var (
	// testFilenameSnakeDotPattern matches lowercase snake/dot test-file
	// naming conventions: test_foo.py, foo_test.go, foo.test.js,
	// foo-spec.js.
	testFilenameSnakeDotPattern = regexp.MustCompile(`^test_|_test\.|\.test\.|-spec\.`)

	// testFilenameSuffixPattern matches PascalCase test-file suffix
	// conventions: FooTest.java, FooTests.cs, FooTestCase.py, FooSpec.kt.
	testFilenameSuffixPattern = regexp.MustCompile(`(Test|Tests|TestCase|Spec)\.[^./]+$`)

	// camelTestDirPattern matches a CamelCase directory segment
	// containing "Test" (Gradle/Kotlin-style, e.g. "androidTest").
	camelTestDirPattern = regexp.MustCompile(`Test`)
)

// testDirNames and nonProductionDirNames are RESEARCH §5's exact
// directory-segment sets (matched case-insensitively):
// /tests?//__tests__//specs?//testlib//testing/, plus the
// non-production dirs (integration, sample(s), example(s), fixture(s),
// benchmark(s), demo(s)).
var (
	testDirNames = map[string]bool{
		"test": true, "tests": true, "__tests__": true,
		"spec": true, "specs": true, "testlib": true, "testing": true,
	}
	nonProductionDirNames = map[string]bool{
		"integration": true,
		"sample":      true, "samples": true,
		"example": true, "examples": true,
		"fixture": true, "fixtures": true,
		"benchmark": true, "benchmarks": true,
		"demo": true, "demos": true,
	}
)

// gatherChannel3 is H5: FTS-style multi-term text search (RESEARCH
// §C.2, context/index.js:530-575). Every node whose Name or
// QualifiedName contains (case-insensitively — terms are already
// lowercased by extractSearchTerms) at least one term is a match, scored
// channel3BaseScore + 5*(termHits-1), where termHits is the count of
// DISTINCT terms that matched. A node whose Kind is channel3ImportKind
// ("import") is excluded when kindFilter is empty (mirrors query/
// search's "" == no filter convention, see ValidateKind); an explicit
// kindFilter both admits import-kind nodes and restricts every other
// node to that one kind.
func gatherChannel3(r graphstore.Reader, terms []string, kindFilter string) ([]gatherCandidate, error) {
	if len(terms) == 0 {
		return nil, nil
	}

	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var candidates []gatherCandidate
	for it.Next() {
		n := it.Node()
		if kindFilter == "" {
			if n.Kind == channel3ImportKind {
				continue
			}
		} else if n.Kind != kindFilter {
			continue
		}

		nameLower := strings.ToLower(n.Name)
		qualLower := strings.ToLower(n.QualifiedName)
		hits := 0
		for _, t := range terms {
			if t == "" {
				continue
			}
			if strings.Contains(nameLower, t) || strings.Contains(qualLower, t) {
				hits++
			}
		}
		if hits == 0 {
			continue
		}

		score := channel3BaseScore
		if hits > 1 {
			score += channel3MultiTermBoost * float64(hits-1)
		}
		candidates = append(candidates, gatherCandidate{
			Node:     n,
			Score:    score,
			Channels: map[gatherChannelKind]bool{gatherChannelFTS: true},
		})
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	sortGatherCandidates(candidates)
	return candidates, nil
}

// gatherMerge is H6: dedup by node id across any number of channel
// result sets, keeping the MAX score seen for any given id (not summed)
// and the UNION of every channel that surfaced it (RESEARCH §C.2/H6,
// context/index.js:580-606).
func gatherMerge(channelResults ...[]gatherCandidate) []gatherCandidate {
	merged := make(map[string]*gatherCandidate)
	var order []string
	for _, ch := range channelResults {
		for _, c := range ch {
			id := c.Node.Id
			existing, ok := merged[id]
			if !ok {
				cp := gatherCandidate{
					Node:     c.Node,
					Score:    c.Score,
					Channels: make(map[gatherChannelKind]bool, len(c.Channels)),
				}
				for k := range c.Channels {
					cp.Channels[k] = true
				}
				merged[id] = &cp
				order = append(order, id)
				continue
			}
			for k := range c.Channels {
				existing.Channels[k] = true
			}
			if c.Score > existing.Score {
				existing.Score = c.Score
				existing.Node = c.Node
			}
		}
	}

	out := make([]gatherCandidate, 0, len(order))
	for _, id := range order {
		out = append(out, *merged[id])
	}
	sortGatherCandidates(out)
	return out
}

// isTestFile is TS's file-PATH predicate (RESEARCH §5,
// search/query-utils.js:300-332 [VERIFIED: TS 1.3.1 dist]) — NOT
// traverse.go's isTestSymbol (a SYMBOL-name heuristic; that one is left
// unwidened, this is ported as a separate function). Defined once, here,
// so neither H7 (plan 10's test-file dampening) nor the relevance gate
// (plan 14) duplicates it.
func isTestFile(filePath string) bool {
	if filePath == "" {
		return false
	}
	norm := strings.ReplaceAll(filePath, "\\", "/")
	segments := strings.Split(norm, "/")
	base := segments[len(segments)-1]

	if testFilenameSnakeDotPattern.MatchString(base) || testFilenameSuffixPattern.MatchString(base) {
		return true
	}

	for _, seg := range segments[:len(segments)-1] {
		if seg == "" {
			continue
		}
		lower := strings.ToLower(seg)
		if testDirNames[lower] || nonProductionDirNames[lower] {
			return true
		}
		if camelTestDirPattern.MatchString(seg) {
			return true
		}
	}
	return false
}
