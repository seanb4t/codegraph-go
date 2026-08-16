// Package query — internal/query/gather.go implements the hybrid
// candidate-gathering heuristics H3-H6 (RESEARCH §C.2,
// context/index.js:449-606 [cited from the frozen 01-RESEARCH.md
// capture — the original source is no longer present on this machine,
// only its .d.ts type declarations remain, so this file works from
// RESEARCH's pinned constants rather than a fresh source read]): three
// independently-
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

	// channel3ImportKind matches the documented "import" node Kind for
	// H5's exclusion rule. No priority-4 extractor in this repo currently
	// emits an "import" Node.Kind (imports are captured as
	// goextract.RefKindImports EDGES, not nodes — see
	// internal/indexer/goextract/types.go), so this exclusion is
	// presently a no-op against real indexes; kept so a future extractor
	// never silently slips import-declaration nodes into the FTS
	// channel.
	channel3ImportKind = "import"
)

// definitionKinds is H4's Channel-2 kind filter — the documented
// "class/interface/struct/trait/protocol/enum/type_alias" set. In this
// codebase's Kind vocabulary, every priority-4 extractor already
// collapses class/struct/trait/protocol/enum into goextract.KindStruct
// (see e.g. javaextract/types.go, pyextract/types.go, tsextract/types.go
// doc comments: "Reusing KindStruct rather than minting a new class kind
// keeps struct/class-shaped downstream logic unified") — there is no
// separate "class"/"enum"/"trait"/"protocol" Kind value anywhere in this
// repo's extractors to have missed. So the full documented set maps onto exactly
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
// Divergence (D-02, inherited from 01-03-SUMMARY.md): the documented
// stem-variant expansion (getStemVariants) is NOT applied here — 01-03
// deliberately deferred stem-variant support from
// extractSymbolsFromQuery/extractSearchTerms entirely (no stub
// parameter), so this channel searches the literal titlecased symbol
// tokens only. A follow-on plan can add stem expansion here once
// getStemVariants is implemented.
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

// --- H7 test-file dampening + H8 core-directory boost (RESEARCH §C.2,
// context/index.js:607-647) ---
//
// These are post-merge rerankers plan 10 layers on top of H3-H6's merged
// candidate set (gatherMerge's output) — pure functions over
// []gatherCandidate, no reader access. Both mutate candidates IN PLACE
// (matching gatherMerge's own already-deterministic ordering contract);
// callers that need a fresh sort afterward call sortGatherCandidates
// themselves (see applyPostMergeRerankers below).

const (
	// testFileDampeningFactor is H7's exact, cited constant: score *= 0.3
	// for test-file nodes (RESEARCH §C.2/H7, context/index.js:607-616).
	testFileDampeningFactor = 0.3

	// coreDirectoryBoost is H8's exact, cited constant: +25 to every
	// candidate sharing a dominant file's directory prefix (RESEARCH
	// §C.2/H8, context/index.js:617-647).
	coreDirectoryBoost = 25.0

	// coreDirectoryDominanceRatio is H8's exact, cited threshold: a file
	// is "dominant" when its count is >=3x the next-most-frequent file's
	// count (RESEARCH §C.2/H8).
	coreDirectoryDominanceRatio = 3.0
)

// queryMentionsTestOrSpec is H7's exemption check: the raw query text
// (case-insensitive substring) mentions "test" or "spec" — RESEARCH
// §C.2/H7 pins this exact wording ("unless query mentions test/spec"),
// not a tokenized/stemmed match.
func queryMentionsTestOrSpec(query string) bool {
	q := strings.ToLower(query)
	return strings.Contains(q, "test") || strings.Contains(q, "spec")
}

// applyTestFileDampening is H7 (RESEARCH §C.2, context/index.js:607-616):
// multiplies every test-file candidate's score by testFileDampeningFactor
// (0.3), UNLESS the query mentions test/spec (queryMentionsTestOrSpec) —
// in which case NO candidate is dampened at all, per the documented
// short-circuit. exemptIDs (keyed by Node.Id) skips dampening for candidates
// H9's distinctive-identifier exact-match exemption has already flagged
// (plan 10 Task 2, applyPostMergeRerankers) — nil/empty exemptIDs is a
// no-op filter, so Task 1's tests (no exemption yet in play) pass nil.
func applyTestFileDampening(candidates []gatherCandidate, query string, exemptIDs map[string]bool) {
	if queryMentionsTestOrSpec(query) {
		return
	}
	for i := range candidates {
		if exemptIDs[candidates[i].Node.Id] {
			continue
		}
		if isTestFile(candidates[i].Node.FilePath) {
			candidates[i].Score *= testFileDampeningFactor
		}
	}
}

// fileDir returns the directory portion of a forward-slash-normalized
// file path ("pkg/sub/foo.go" -> "pkg/sub"; a root-level "foo.go" -> "").
func fileDir(filePath string) string {
	norm := strings.ReplaceAll(filePath, "\\", "/")
	idx := strings.LastIndex(norm, "/")
	if idx < 0 {
		return ""
	}
	return norm[:idx]
}

// sharesDirectoryPrefix reports whether dir is the SAME directory as
// prefixDir, or a nested subdirectory of it (H8's "shares a dominant
// file's directory prefix", RESEARCH §C.2/H8).
func sharesDirectoryPrefix(dir, prefixDir string) bool {
	if dir == prefixDir {
		return true
	}
	return strings.HasPrefix(dir, prefixDir+"/")
}

// applyCoreDirectoryBoost is H8 (RESEARCH §C.2, context/index.js:617-647):
// finds the file with the most candidates in the set (a per-file
// candidate-density proxy for the documented per-file graph-edge count — this pure
// candidate-set function has no reader/graph access to count actual
// edges, a documented substitution consistent with plan 07's base-score
// defaults). If that file's count is >=3x the NEXT most-frequent
// distinct file's count, it is "dominant": every candidate whose file
// shares its directory prefix (sharesDirectoryPrefix, including nested
// subdirectories) earns a flat +coreDirectoryBoost. Deterministic:
// candidates are grouped/sorted by (count desc, file asc) before
// selecting the dominant file, so ties never depend on map iteration
// order — and any tie for the max count trivially fails the >=3x test
// (ratio 1), so tie-break order can never change the pass/fail outcome.
func applyCoreDirectoryBoost(candidates []gatherCandidate) {
	counts := make(map[string]int)
	for _, c := range candidates {
		if c.Node.FilePath == "" {
			continue
		}
		counts[c.Node.FilePath]++
	}
	if len(counts) == 0 {
		return
	}

	type fileCount struct {
		file  string
		count int
	}
	fcs := make([]fileCount, 0, len(counts))
	for f, n := range counts {
		fcs = append(fcs, fileCount{f, n})
	}
	sort.SliceStable(fcs, func(i, j int) bool {
		if fcs[i].count != fcs[j].count {
			return fcs[i].count > fcs[j].count
		}
		return fcs[i].file < fcs[j].file
	})

	dominant := fcs[0]
	var next int
	if len(fcs) > 1 {
		next = fcs[1].count
	}
	if float64(dominant.count) < coreDirectoryDominanceRatio*float64(next) {
		return
	}

	dominantDir := fileDir(dominant.file)
	for i := range candidates {
		if candidates[i].Node.FilePath == "" {
			continue
		}
		if sharesDirectoryPrefix(fileDir(candidates[i].Node.FilePath), dominantDir) {
			candidates[i].Score += coreDirectoryBoost
		}
	}
}

// --- H9 multi-term co-occurrence re-rank + distinctive-identifier
// exemption (RESEARCH §C.2, context/index.js:648-712+) ---

const (
	// multiTermRerankStep is H9's exact, cited constant: score *= 1 +
	// matchCount*0.5 when >=2 stem-grouped term groups match a
	// candidate's name/directory (RESEARCH §C.2/H9,
	// context/index.js:648-712+).
	multiTermRerankStep = 0.5

	// multiTermRerankMinGroups is H9's ">=2 groups match" gate.
	multiTermRerankMinGroups = 2

	// distinctiveIdentifierMinLength is isDistinctiveIdentifier's length
	// floor. RESEARCH §C.2/H9 cites isDistinctiveIdentifier
	// (search/query-utils.js) as gating the H9-exempts-H7 rule but does
	// not pin its exact algorithm in the frozen citation set, and the TS
	// dist JS is unreadable on this machine (see the package doc
	// comment) — this is a documented, conservative substitute (see
	// isDistinctiveIdentifier's own doc comment), consistent with plan
	// 07's precedent of documenting undocumented-but-cited constants.
	distinctiveIdentifierMinLength = 6
)

// stemTerm reduces a lowercase query term to a naive stem by stripping
// the most common English inflectional suffixes (plural -s/-es/-ies,
// verb -ing/-ed). This is a documented, deterministic, lightweight
// substitute for the documented getStemVariants() algorithm
// (search/query-utils.js:129-175), which 01-03 explicitly deferred
// implementing (tokenize.go's extractSearchTerms doc comment, D-02) —
// RESEARCH §C.2/H9 requires terms be "stem-grouped" but does not pin a
// specific stemming algorithm, only the grouping OUTCOME (near-duplicate
// inflections of the same root count as one group). Not a full
// Porter-stemmer implementation; sufficient to merge
// "handler"/"handlers"-shaped variants without over-engineering a case
// RESEARCH doesn't cite an exact constant for.
func stemTerm(term string) string {
	t := strings.ToLower(term)
	switch {
	case strings.HasSuffix(t, "ies") && len(t) > 4:
		return t[:len(t)-3] + "y"
	case strings.HasSuffix(t, "es") && len(t) > 4:
		return t[:len(t)-2]
	case strings.HasSuffix(t, "ing") && len(t) > 5:
		return t[:len(t)-3]
	case strings.HasSuffix(t, "ed") && len(t) > 4:
		return t[:len(t)-2]
	case strings.HasSuffix(t, "s") && !strings.HasSuffix(t, "ss") && len(t) > 3:
		return t[:len(t)-1]
	}
	return t
}

// groupTermsByStem groups terms by stemTerm's result, preserving
// first-seen group order (D-04 determinism — never map iteration for
// output order).
func groupTermsByStem(terms []string) [][]string {
	var order []string
	groups := make(map[string][]string)
	for _, t := range terms {
		if t == "" {
			continue
		}
		s := stemTerm(t)
		if _, ok := groups[s]; !ok {
			order = append(order, s)
		}
		groups[s] = append(groups[s], t)
	}
	out := make([][]string, 0, len(order))
	for _, s := range order {
		out = append(out, groups[s])
	}
	return out
}

// applyMultiTermReRank is H9's co-occurrence multiplier (RESEARCH §C.2,
// context/index.js:648-712+): groups terms by stem (groupTermsByStem),
// then for each candidate counts how many DISTINCT groups have at least
// one member term appearing (case-insensitive substring) in the
// candidate's Name or its file's directory (fileDir). When
// multiTermRerankMinGroups (2) or more groups match, the score is
// multiplied by 1 + matchCount*multiTermRerankStep.
func applyMultiTermReRank(candidates []gatherCandidate, terms []string) {
	groups := groupTermsByStem(terms)
	if len(groups) == 0 {
		return
	}
	for i := range candidates {
		nameLower := strings.ToLower(candidates[i].Node.Name)
		dirLower := strings.ToLower(fileDir(candidates[i].Node.FilePath))

		matchCount := 0
		for _, group := range groups {
			for _, term := range group {
				if term == "" {
					continue
				}
				if strings.Contains(nameLower, term) || strings.Contains(dirLower, term) {
					matchCount++
					break
				}
			}
		}
		if matchCount >= multiTermRerankMinGroups {
			candidates[i].Score *= 1 + float64(matchCount)*multiTermRerankStep
		}
	}
}

// isDistinctiveIdentifier is a documented heuristic substitute for the
// documented isDistinctiveIdentifier (search/query-utils.js — cited by RESEARCH
// §C.2/H9 as gating the H9-exempts-H7 rule, but its exact algorithm was
// not captured in RESEARCH's frozen citations and the TS dist JS is
// unreadable on this machine — see the package doc comment). Captures
// the load-bearing intent RESEARCH DOES pin: a "distinctive identifier"
// is a structured, multi-word symbol name — not a short/common single
// English word — via a conservative, deterministic rule: name is
// distinctive when it is at least distinctiveIdentifierMinLength (6)
// runes AND contains either an underscore (snake_case) or an internal
// case transition (camelCase/PascalCase, i.e. an uppercase rune after
// position 0). Short or single-case names ("Run", "foo") are never
// distinctive.
func isDistinctiveIdentifier(name string) bool {
	if len([]rune(name)) < distinctiveIdentifierMinLength {
		return false
	}
	if strings.Contains(name, "_") {
		return true
	}
	for i, r := range name {
		if i == 0 {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

// distinctiveExactMatchExemptIDs computes the set of candidate Node.Ids
// that are a distinctive-identifier EXACT match against one of terms —
// RESEARCH §C.2/H9's "distinctive-identifier exact matches are exempt
// from the H7 dampening" rule. A candidate is exempt when its Node.Name
// case-insensitively equals one of terms AND that name is distinctive
// (isDistinctiveIdentifier). Computing this set is a prerequisite step
// for applyPostMergeRerankers, which must run it BEFORE applyTestFileDampening
// (order-sensitive per RESEARCH §C.2's H7/H9 interaction note) so the
// exemption can actually gate the dampening rather than being overridden
// by it.
func distinctiveExactMatchExemptIDs(candidates []gatherCandidate, terms []string) map[string]bool {
	exempt := make(map[string]bool)
	for _, c := range candidates {
		name := c.Node.Name
		if name == "" {
			continue
		}
		nameLower := strings.ToLower(name)
		for _, t := range terms {
			if t == "" {
				continue
			}
			if strings.ToLower(t) == nameLower && isDistinctiveIdentifier(name) {
				exempt[c.Node.Id] = true
				break
			}
		}
	}
	return exempt
}

// applyPostMergeRerankers applies H7, H8 and H9 (RESEARCH §C.2,
// context/index.js:607-712+) to a merged candidate set (gatherMerge's
// output), in RESEARCH's pipeline order — with one deliberate exception:
// the H9 distinctive-identifier exemption set is computed FIRST, before
// H7 runs, so a distinctive-identifier exact match that also happens to
// be a test file is never dampened (RESEARCH §C.2/H9's explicit
// order-sensitive note: "Implement H7 and H9 so the exemption is
// honored"). Mutates candidates in place and returns the same slice,
// re-sorted (score-desc/Id-asc, D-04) after every mutation.
func applyPostMergeRerankers(candidates []gatherCandidate, query string, terms []string) []gatherCandidate {
	exempt := distinctiveExactMatchExemptIDs(candidates, terms)
	applyTestFileDampening(candidates, query, exempt)
	applyCoreDirectoryBoost(candidates)
	applyMultiTermReRank(candidates, terms)
	sortGatherCandidates(candidates)
	return candidates
}

// isTestFile is the documented file-PATH predicate (RESEARCH §5,
// search/query-utils.js:300-332 [cited from the frozen RESEARCH
// capture]) — NOT traverse.go's isTestSymbol (a SYMBOL-name heuristic;
// that one is left unwidened, this is implemented as a separate
// function). Defined once, here, so neither H7 (plan 10's test-file
// dampening) nor the relevance gate (plan 14) duplicates it.
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
