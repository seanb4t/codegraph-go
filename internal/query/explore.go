package query

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// exploreFileGroup is one distinct matched-file's projection: its path
// plus every matched symbol defined in it, in ranked-match order (D-05a's
// per-file "**`path`** — sym(kind), ..." header).
type exploreFileGroup struct {
	Path    string
	Symbols []*schema.Node
}

// exploreBlast is one blast-radius bullet's data (D-05a): the matched
// symbol, its total caller count, and the distinct files among those
// callers that are test symbols (D-07's isTestSymbol heuristic).
type exploreBlast struct {
	Symbol      *schema.Node
	CallerCount int
	TestFiles   []string
}

// groupMatchesByFile buckets ranked matches by FilePath, preserving match
// order within each bucket, and caps the number of distinct files at
// maxFiles (RESEARCH Pitfall 4 / T-03-06-DoS — explore's per-file
// verbatim-source read is the expensive part this cap bounds). Once
// maxFiles distinct files have been selected, further matches in NEW files
// are dropped; matches in an already-selected file are still included
// (so a file with many matched symbols isn't truncated just because it
// was the maxFiles-th file admitted). Matches with no FilePath (the
// synthetic "package" pseudo-node kind, internal/indexer/resolve.go) are
// skipped — there is no on-disk file to read verbatim source from.
func groupMatchesByFile(ranked []rankedNode, maxFiles int) ([]exploreFileGroup, int) {
	var groups []exploreFileGroup
	index := make(map[string]int)
	symbolCount := 0

	for _, r := range ranked {
		n := r.node
		if n.FilePath == "" {
			continue
		}
		idx, ok := index[n.FilePath]
		if !ok {
			if len(groups) >= maxFiles {
				continue
			}
			groups = append(groups, exploreFileGroup{Path: n.FilePath})
			idx = len(groups) - 1
			index[n.FilePath] = idx
		}
		groups[idx].Symbols = append(groups[idx].Symbols, n)
		symbolCount++
	}

	return groups, symbolCount
}

// buildBlastEntry computes n's blast-radius bullet data from the shared
// D-04 reverse-adjacency map: total caller count, plus the distinct files
// among those callers that pass isTestSymbol (D-07).
func (e *Engine) buildBlastEntry(n *schema.Node, rev map[string][]*schema.Edge) (exploreBlast, error) {
	callers := rev[n.Id]

	testFileSet := make(map[string]bool)
	for _, edge := range callers {
		src, err := e.reader.GetNode(edge.Source)
		if err != nil {
			// WR-04: a dangling caller reference is skipped rather than
			// aborting the whole Explore call.
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return exploreBlast{}, err
		}
		if isTestSymbol(src) {
			testFileSet[src.FilePath] = true
		}
	}
	testFiles := make([]string, 0, len(testFileSet))
	for f := range testFileSet {
		testFiles = append(testFiles, f)
	}
	sort.Strings(testFiles)

	return exploreBlast{Symbol: n, CallerCount: len(callers), TestFiles: testFiles}, nil
}

// exploreZeroResult is the shared "no results" render (WR-05's sibling for
// a query that tokenizes to nothing matchable, or whose full pipeline
// gates every candidate file out) — kept as a single helper so the empty
// message text lives in exactly one place.
func exploreZeroResult(query string, stale bool) string {
	return fmt.Sprintf("%s**Exploration: %s**\n\nFound 0 symbols across 0 files.\n", staleBanner(stale), query)
}

// getExploreOutputBudget is H21's adaptive per-project-size default
// (RESEARCH §C.2/H21, mcp/tools.js:2410-2417) — sets explore's default
// maxFiles by project size (a tiny project gets a tighter default, a large
// one a wider one). Divergence (D-02): the live TS dist is unreadable on
// this machine (gather.go's package doc comment) — only the qualitative
// rule ("bigger project, wider budget") and the [1,20] clamp survive in
// the frozen RESEARCH capture, not the exact original step function; this
// monotonic step function is this plan's own documented substitute.
func getExploreOutputBudget(fileCount int) int {
	switch {
	case fileCount <= 20:
		return 3
	case fileCount <= 100:
		return 5
	case fileCount <= 500:
		return 8
	default:
		return 12
	}
}

// clampExploreBudget clamps n to H21's [1,20] range (RESEARCH §C.2/H21).
func clampExploreBudget(n int) int {
	if n < 1 {
		return 1
	}
	if n > 20 {
		return 20
	}
	return n
}

// countIndexedFiles counts the distinct non-empty FilePath values across
// the whole index (a full IterateNodes scan) — the "project file count"
// H21's adaptive budget sizes itself against.
func (e *Engine) countIndexedFiles() (int, error) {
	it, err := e.reader.IterateNodes()
	if err != nil {
		return 0, err
	}
	defer it.Close()

	files := make(map[string]bool)
	for it.Next() {
		n := it.Node()
		if n.FilePath != "" {
			files[n.FilePath] = true
		}
	}
	if err := it.Err(); err != nil {
		return 0, err
	}
	return len(files), nil
}

// computeFileTermHits counts, per file, how many DISTINCT query terms
// (extractSearchTerms' output) case-insensitively match a Name or
// QualifiedName substring among nodeIDs — the fileTermHits input H16's
// buried-rescue and H17/H18/H19's gate/sort/central-selection all consume
// (RESEARCH §4/§C.2). Neither plan 13 nor plan 14 computed this — each
// documented it as a still-unwired, caller-supplied input; this plan's
// wiring is what actually produces it. WR-04: a nodeIDs entry that no
// longer resolves is skipped, not an error. Deterministic: nodeIDs are
// walked in sorted order (the returned counts are order-independent, but
// the scan itself never depends on map iteration order).
func computeFileTermHits(r graphstore.Reader, nodeIDs []string, terms []string) (map[string]int, error) {
	sorted := append([]string(nil), nodeIDs...)
	sort.Strings(sorted)

	hitSets := make(map[string]map[string]bool)
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
		nameLower := strings.ToLower(n.Name)
		qualLower := strings.ToLower(n.QualifiedName)
		for _, t := range terms {
			if t == "" {
				continue
			}
			if strings.Contains(nameLower, t) || strings.Contains(qualLower, t) {
				set, ok := hitSets[n.FilePath]
				if !ok {
					set = make(map[string]bool)
					hitSets[n.FilePath] = set
				}
				set[t] = true
			}
		}
	}

	out := make(map[string]int, len(hitSets))
	for f, s := range hitSets {
		out[f] = len(s)
	}
	return out, nil
}

// Explore is the flagship one-round-trip command (QRY-08, D-05a, EXPL-02):
// wires the full explore pipeline in the documented stage order
// (RESEARCH Architecture Diagram) — tokenize (H1/H2) -> hybrid gather
// (H3-H6) -> post-merge rerank (H7-H9) -> named-symbol seeding (H13) ->
// type-hierarchy expansion (H10) -> bounded BFS (H11) -> glue-node
// injection (H12) -> RWR graph relevance (computeGraphRelevance) -> per-file
// score tiers + hard exclusion + buried-rescue (H14-H16) -> central-file
// selection + the 5-way relevance gate + the 5-tier sort (H17-H19) ->
// render, with H21's adaptive output budget sizing the render. RWR output
// (not lexical match order) feeds groupMatchesByFile — the RESEARCH
// Pitfall 1 fix: this REPLACES the lexical matchNodes input construction,
// not a rerank layered on top of it. groupMatchesByFile/buildBlastEntry/
// readSourceFile are unchanged downstream (D-05's "extend, don't replace").
//
// "Matched" symbols shown per selected file (the header's "sym(kind), ..."
// list and the blast-radius bullets) are the query's actual gather+seed
// candidates in that file, RWR-score-ordered (not the whole bounded
// subgraph — showing every BFS/glue/hierarchy-expanded node would flood a
// file's header with symbols the query never textually or nominally
// matched, and would break the single-exact-match "1 symbol, 1 file"
// contract). A file selected purely through structural connectivity (no
// direct candidate landed in it) falls back to its single highest-RWR-mass
// subgraph node, so a structurally-selected file is never rendered with an
// empty symbol list.
//
// query is rejected if empty/whitespace-only (WR-05, mirroring
// Query/Search) rather than falling through to a tokenizer's degenerate
// "matches everything" case; the tokenizers themselves also return empty
// for empty input (plan 03) — two layers, per the threat model's T-01-25
// disposition.
func (e *Engine) Explore(query string, maxFiles int) (string, error) {
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query: explore query must not be empty")
	}
	if err := validateMaxFiles(maxFiles); err != nil {
		return "", err
	}
	explicitMaxFiles := maxFiles
	maxFiles = clampMaxFiles(maxFiles)

	meta, err := e.reader.GetMeta()
	if err != nil && !errors.Is(err, graphstore.ErrNotFound) {
		return "", err
	}
	stale, err := e.computeStale(meta)
	if err != nil {
		return "", err
	}

	// H1/H2: tokenize the query for named-symbol seeding (H1) and the FTS
	// gather channel (H2), respectively (RESEARCH Anti-Pattern: never
	// conflate the two).
	symbols := extractSymbolsFromQuery(query)
	terms := extractSearchTerms(query)

	// H3-H6: hybrid gather (exact-name+co-location, titlecase
	// definition-prefix, FTS multi-term) merged max-score-wins.
	ch1, err := gatherChannel1(e.reader, symbols, 0)
	if err != nil {
		return "", err
	}
	ch2, err := gatherChannel2(e.reader, symbols)
	if err != nil {
		return "", err
	}
	ch3, err := gatherChannel3(e.reader, terms, "")
	if err != nil {
		return "", err
	}
	candidates := gatherMerge(ch1, ch2, ch3)

	// H7-H9: test-file dampening, core-directory boost, multi-term
	// co-occurrence re-rank (with the H9-before-H7 exemption ordering
	// applyPostMergeRerankers already encodes).
	candidates = applyPostMergeRerankers(candidates, query, terms)

	// H13: named-symbol seeding + per-overload disambiguation tiers.
	projectName := filepath.Base(e.repoRoot)
	seeds, err := seedNamedSymbols(e.reader, query, projectName)
	if err != nil {
		return "", err
	}

	if len(candidates) == 0 && len(seeds.SeedIDs) == 0 {
		return exploreZeroResult(query, stale), nil
	}

	// H10: type-hierarchy expansion — focal ids are the candidates whose
	// Kind is itself a type/definition kind (struct/interface/type_alias).
	var focalIDs []string
	for _, c := range candidates {
		if definitionKinds[c.Node.Kind] {
			focalIDs = append(focalIDs, c.Node.Id)
		}
	}
	hierarchyIDs, err := expandTypeHierarchy(e.reader, focalIDs, ExpandMaxNodes)
	if err != nil {
		return "", err
	}

	// H11: bounded BFS from the gathered/reranked candidates.
	bfsNodeIDs, bfsRootIDs, _, err := expandBFS(e.reader, candidates, DefaultExploreBFSBounds)
	if err != nil {
		return "", err
	}

	// The node universe so far (BFS + hierarchy expansion + named seeds) —
	// H12's glue-node injection needs "files already surfaced" from this
	// set before it can decide what to pull in.
	nodeSet := make(map[string]bool, len(bfsNodeIDs)+len(hierarchyIDs)+len(seeds.SeedIDs))
	for _, id := range bfsNodeIDs {
		nodeSet[id] = true
	}
	for _, id := range hierarchyIDs {
		nodeSet[id] = true
	}
	for _, id := range seeds.SeedIDs {
		nodeSet[id] = true
	}

	surfacedIDs := make([]string, 0, len(nodeSet))
	for id := range nodeSet {
		surfacedIDs = append(surfacedIDs, id)
	}
	surfacedFiles, err := subgraphFileSet(e.reader, surfacedIDs)
	if err != nil {
		return "", err
	}

	// H12: glue-node injection around the BFS roots and named seeds.
	glueRoots := make([]string, 0, len(bfsRootIDs)+len(seeds.SeedIDs))
	glueRoots = append(glueRoots, bfsRootIDs...)
	glueRoots = append(glueRoots, seeds.SeedIDs...)
	glueIDs, err := expandGlueNodes(e.reader, glueRoots, surfacedFiles, GlueNodeCap)
	if err != nil {
		return "", err
	}
	for _, id := range glueIDs {
		nodeSet[id] = true
	}

	finalNodeIDs := make([]string, 0, len(nodeSet))
	for id := range nodeSet {
		finalNodeIDs = append(finalNodeIDs, id)
	}
	sort.Strings(finalNodeIDs)

	if len(finalNodeIDs) == 0 {
		return exploreZeroResult(query, stale), nil
	}

	// Rebuild the RankEdges-filtered edge list over the FULL final node
	// set: expandBFS's own internal edge list is scoped to just its own
	// BFS-discovered nodes, but H10/H12/H13 may have added nodes (and thus
	// edges among them) it never saw.
	_, allRankEdges, err := buildExpandAdjacency(e.reader)
	if err != nil {
		return "", err
	}
	finalEdges := make([]*schema.Edge, 0, len(allRankEdges))
	for _, edge := range allRankEdges {
		if nodeSet[edge.Source] && nodeSet[edge.Target] {
			finalEdges = append(finalEdges, edge)
		}
	}

	// computeGraphRelevance: RWR restarts uniformly over the named seeds
	// (H13) plus the BFS roots (H11) — the pipeline's two notions of
	// "root" both feed the restart vector.
	rwrSeedIDs := make(map[string]bool, len(seeds.SeedIDs)+len(bfsRootIDs))
	for _, id := range seeds.SeedIDs {
		rwrSeedIDs[id] = true
	}
	for _, id := range bfsRootIDs {
		rwrSeedIDs[id] = true
	}
	rwrScores := computeGraphRelevance(finalNodeIDs, finalEdges, rwrSeedIDs)

	// H14: per-file score tiers (named-seed/entry/connected/other).
	namedSeedIDs := make(map[string]bool)
	for _, nm := range seeds.Names {
		for _, id := range nm.Primary {
			namedSeedIDs[id] = true
		}
	}
	entryIDs := make(map[string]bool, len(bfsRootIDs))
	for _, id := range bfsRootIDs {
		entryIDs[id] = true
	}
	fileScores, err := computeFileScoreTiers(e.reader, finalNodeIDs, finalEdges, namedSeedIDs, entryIDs)
	if err != nil {
		return "", err
	}
	fileGraphScore, err := aggregateFileGraphScore(e.reader, finalNodeIDs, rwrScores)
	if err != nil {
		return "", err
	}

	// H15: hard test/spec/icon/i18n exclusion.
	applyHardTestExclusion(fileScores, query)
	if len(fileScores) == 0 {
		return exploreZeroResult(query, stale), nil
	}

	fileTermHits, err := computeFileTermHits(e.reader, finalNodeIDs, terms)
	if err != nil {
		return "", err
	}

	var maxGraph float64
	for _, v := range fileGraphScore {
		if v > maxGraph {
			maxGraph = v
		}
	}

	// H16: change-surface buried-rescue over the tier-seed callables
	// (named-seed union entry tier).
	tierSeedIDs := make([]string, 0, len(namedSeedIDs)+len(entryIDs))
	for id := range namedSeedIDs {
		tierSeedIDs = append(tierSeedIDs, id)
	}
	for id := range entryIDs {
		if !namedSeedIDs[id] {
			tierSeedIDs = append(tierSeedIDs, id)
		}
	}
	rescued, err := applyBuriedRescue(e.reader, tierSeedIDs, fileGraphScore, maxGraph, fileTermHits, fileScores)
	if err != nil {
		return "", err
	}

	candidatePaths := make([]string, 0, len(fileScores))
	for f := range fileScores {
		candidatePaths = append(candidatePaths, f)
	}
	sort.Strings(candidatePaths)

	// H19: central-file selection (feeds the gate's clause 2 and the
	// sort's tier 2), computed before the gate per RESEARCH's own
	// dependency order.
	centralFiles := centralFileSelection(candidatePaths, fileGraphScore, fileTermHits)

	// H17: the 5-way OR relevance gate (EXPL-03's core).
	gated := fileRelevanceGate(candidatePaths, fileScores, fileGraphScore, centralFiles, rescued, fileTermHits)

	fileNodeCounts := make(map[string]int)
	for _, id := range finalNodeIDs {
		n, gerr := e.reader.GetNode(id)
		if gerr != nil {
			if errors.Is(gerr, graphstore.ErrNotFound) {
				continue
			}
			return "", gerr
		}
		if n.FilePath != "" {
			fileNodeCounts[n.FilePath]++
		}
	}

	// H18: the 5-tier file sort — this is the ranked file ORDER
	// groupMatchesByFile below consumes instead of lexical match order.
	fileOrder := fiveTierFileSort(gated, fileScores, fileGraphScore, centralFiles, fileTermHits, fileNodeCounts)
	if len(fileOrder) == 0 {
		return exploreZeroResult(query, stale), nil
	}

	// H21: the adaptive output budget only overrides the DEFAULT (the
	// caller passed no explicit --max-files, i.e. explicitMaxFiles<=0);
	// an explicit value's validate+clamp above stays verbatim.
	if explicitMaxFiles <= 0 {
		fileCount, cerr := e.countIndexedFiles()
		if cerr != nil {
			return "", cerr
		}
		maxFiles = clampExploreBudget(getExploreOutputBudget(fileCount))
	}

	// matchedByFile: the query's actual gather+seed candidates, per file —
	// what a file's header/blast-radius shows when it has any.
	matchedByFile := make(map[string][]*schema.Node)
	seenMatched := make(map[string]bool)
	addMatched := func(n *schema.Node) {
		if n == nil || n.FilePath == "" || seenMatched[n.Id] {
			return
		}
		seenMatched[n.Id] = true
		matchedByFile[n.FilePath] = append(matchedByFile[n.FilePath], n)
	}
	for _, c := range candidates {
		addMatched(c.Node)
	}
	for _, id := range seeds.SeedIDs {
		n, gerr := e.reader.GetNode(id)
		if gerr != nil {
			if errors.Is(gerr, graphstore.ErrNotFound) {
				continue
			}
			return "", gerr
		}
		addMatched(n)
	}

	// nodesByFile: every final-subgraph node, per file — the fallback
	// source for a file selected purely through structural connectivity
	// (no direct candidate landed in it).
	nodesByFile := make(map[string][]*schema.Node)
	for _, id := range finalNodeIDs {
		n, gerr := e.reader.GetNode(id)
		if gerr != nil {
			if errors.Is(gerr, graphstore.ErrNotFound) {
				continue
			}
			return "", gerr
		}
		if n.FilePath == "" {
			continue
		}
		nodesByFile[n.FilePath] = append(nodesByFile[n.FilePath], n)
	}

	rwrOrder := func(nodes []*schema.Node) {
		sort.SliceStable(nodes, func(i, j int) bool {
			si, sj := rwrScores[nodes[i].Id], rwrScores[nodes[j].Id]
			if si != sj {
				return si > sj
			}
			return nodes[i].Id < nodes[j].Id
		})
	}
	for f := range matchedByFile {
		rwrOrder(matchedByFile[f])
	}
	for f := range nodesByFile {
		rwrOrder(nodesByFile[f])
	}

	var ranked []rankedNode
	for _, f := range fileOrder {
		syms := matchedByFile[f]
		if len(syms) == 0 && len(nodesByFile[f]) > 0 {
			syms = nodesByFile[f][:1]
		}
		for _, n := range syms {
			ranked = append(ranked, rankedNode{node: n})
		}
	}

	groups, symbolCount := groupMatchesByFile(ranked, maxFiles)
	if len(groups) == 0 {
		return exploreZeroResult(query, stale), nil
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return "", err
	}

	blasts := make([]exploreBlast, 0, symbolCount)
	for _, g := range groups {
		for _, n := range g.Symbols {
			bl, err := e.buildBlastEntry(n, rev)
			if err != nil {
				return "", err
			}
			blasts = append(blasts, bl)
		}
	}

	sources := make(map[string][]byte, len(groups))
	for _, g := range groups {
		content, err := e.readSourceFile(g.Path)
		if err != nil {
			return "", err
		}
		sources[g.Path] = content
	}

	// H20: polymorphic-sibling skeletonization for off-spine files.
	implementsIdx, err := BuildImplementsIndex(e.reader)
	if err != nil {
		return "", err
	}
	skeletonFiles := computeSkeletonFiles(groups, centralFiles, implementsIdx)

	return RenderExplore(query, len(groups), symbolCount, groups, blasts, sources, stale, skeletonFiles), nil
}
