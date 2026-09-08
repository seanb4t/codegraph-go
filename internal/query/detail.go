package query

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// NodeDetailMode discriminates which of NodeDetail's three pointer fields
// is populated (D-02): exactly one of File, Definition and Multi is
// non-nil for any given mode, matching the three shapes Node() can
// render — file-only source, a single definition, or an overloaded
// symbol's multiple definitions.
type NodeDetailMode int

const (
	// NodeDetailModeFile means File is populated (Definition and Multi are
	// nil) — Node()'s empty-symbol, file-only branch.
	NodeDetailModeFile NodeDetailMode = iota
	// NodeDetailModeSingleDef means Definition is populated (File and
	// Multi are nil) — a symbol that resolves to exactly one node.
	NodeDetailModeSingleDef
	// NodeDetailModeMultiDef means Multi is populated (File and
	// Definition are nil) — a symbol that resolves to more than one node.
	NodeDetailModeMultiDef
)

// NodeDetail is the structured ENG-01 seam Node() renders from (D-01,
// D-02, D-03): a plain Go sum type, not a generated protobuf message, so
// internal/query stays wire-agnostic and the RPC layer owns the mapping
// to wire messages. Mode identifies which of File, Definition and Multi
// is populated; the other two are nil.
type NodeDetail struct {
	Mode       NodeDetailMode
	File       *FileDetail
	Definition *DefinitionDetail
	Multi      *MultiDefDetail
}

// FileDetail is Node()'s file-only shape: the empty-symbol, file-path
// branch that reads a whole file's verbatim content via readSourceFile
// (renderNumberedSource's input).
type FileDetail struct {
	Path   string
	Source []byte
}

// DefinitionDetail is Node()'s single-definition shape — the exact
// argument tuple RenderNode already takes (node, calls, calledBy), plus
// Source for the multi-definition candidate case, where each candidate's
// verbatim body is read (renderNodeSection's input). The single-
// definition path never populates Source (RenderNode does not take one);
// a consumer that wants single-definition source asks for it explicitly
// through (*Engine).SourceFor rather than finding it here.
//
// Calls and CalledBy are exactly what fetchCalls/fetchCalledBy returned —
// nil stays nil, zero-length stays zero-length. The builders below never
// normalise these values (ENG-01): this type is a transcription of the
// fetchers' output, not a reinterpretation of it.
type DefinitionDetail struct {
	Node     *schema.Node
	Source   []byte
	Calls    []*schema.Node
	CalledBy []*schema.Node
}

// MultiDefDetail is Node()'s multi-definition shape, modelled as what it
// actually is: a LAZY per-candidate protocol, not a flat struct (D-02).
// RenderNodeMultiDef takes a fetch callback and invokes it lazily, only
// for matches below nodeMultiDefHardCap, with the body-budget decision
// interleaved into the same loop — a flat struct with one Node, one
// Calls and one CalledBy cannot represent that, and an eager gather over
// every match would perform I/O the current code never performs. Definition
// gathers exactly one candidate on demand, in the same source-then-calls-
// then-calledBy order the existing render closure uses, so a candidate the
// renderer never asks for is never read from disk and no readSourceFile
// error is surfaced where today there is none.
//
// MultiDefDetail imposes no cap of its own: Node()'s markdown budget
// (nodeMultiDefHardCap / nodeMultiDefBodyBudget) is RenderNodeMultiDef's
// business, and a wire consumer applies a different one.
type MultiDefDetail struct {
	Symbol     string
	Matches    []*schema.Node
	definition func(*schema.Node) (*DefinitionDetail, error)
}

// NewMultiDefDetail is MultiDefDetail's exported constructor, so a
// package outside internal/query (internal/uiserver) can declare mapper
// signatures over the type and construct one in its own tests.
func NewMultiDefDetail(symbol string, matches []*schema.Node, definition func(*schema.Node) (*DefinitionDetail, error)) *MultiDefDetail {
	return &MultiDefDetail{Symbol: symbol, Matches: matches, definition: definition}
}

// Definition gathers exactly one candidate's DefinitionDetail on demand —
// nothing is gathered until Definition is called, so a candidate the
// renderer skips is never read from disk. This is the property the whole
// lazy design exists to preserve.
func (m *MultiDefDetail) Definition(n *schema.Node) (*DefinitionDetail, error) {
	return m.definition(n)
}

// buildFileNodeDetail reads file fresh from disk through the existing
// repo-root-confined readSourceFile (T-03-06-Path) — no new read path —
// and returns it as a FileDetail. This is Node()'s file-only branch's
// entire gather step: there is nothing to gather beyond the read.
func (e *Engine) buildFileNodeDetail(file string) (*FileDetail, error) {
	content, err := e.readSourceFile(file)
	if err != nil {
		return nil, err
	}
	return &FileDetail{Path: file, Source: content}, nil
}

// buildSingleDefDetail performs exactly the body the pre-extraction
// single-def render path performed — fetchCalls, then the reverse
// adjacency (via the buildReverseAdjacency seam), then fetchCalledBy —
// and returns the populated struct with Source left nil. It returns
// whatever the fetchers returned without normalising: nil stays nil,
// zero-length stays zero-length (ENG-01).
func (e *Engine) buildSingleDefDetail(node *schema.Node) (*DefinitionDetail, error) {
	calls, err := e.fetchCalls(node)
	if err != nil {
		return nil, err
	}
	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return nil, err
	}
	calledBy, err := e.fetchCalledBy(node, rev)
	if err != nil {
		return nil, err
	}
	return &DefinitionDetail{Node: node, Calls: calls, CalledBy: calledBy}, nil
}

// buildMultiDefDetail builds the reverse adjacency ONCE (via the
// buildReverseAdjacency seam) and closes over it, exactly as the
// pre-extraction multi-def render closure did — fetchCalledBy's own doc
// comment states why the map is shared: the multi-definition path fetches
// trails for up to nodeMultiDefHardCap candidates, and rebuilding the
// O(edges) map per candidate would multiply that scan. The returned
// MultiDefDetail's Definition gathers one candidate at a time, in
// source-then-calls-then-calledBy order — the same order that
// pre-extraction fetch closure used — so which error surfaces
// first for a given candidate is unchanged.
func (e *Engine) buildMultiDefDetail(symbol string, matches []*schema.Node) (*MultiDefDetail, error) {
	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return nil, err
	}

	definition := func(n *schema.Node) (*DefinitionDetail, error) {
		source, err := e.readSourceFile(n.FilePath)
		if err != nil {
			return nil, err
		}
		calls, err := e.fetchCalls(n)
		if err != nil {
			return nil, err
		}
		calledBy, err := e.fetchCalledBy(n, rev)
		if err != nil {
			return nil, err
		}
		return &DefinitionDetail{Node: n, Source: source, Calls: calls, CalledBy: calledBy}, nil
	}

	return NewMultiDefDetail(symbol, matches, definition), nil
}

// buildNodeDetail reproduces Node()'s branch structure exactly — the
// empty-symbol file-only branch, the file-with-no-line exact-match fast
// path through resolveNodeForDetail, enumerateSymbolDefs, the not-found
// error, narrowNodeMatches, and the single-versus-multi split at
// len(matches) == 1 — returning the structured NodeDetail instead of
// rendering it. The branches are not reordered or merged and no error
// string differs from Node()'s own (NODE-04): the fast path exists
// specifically so every pre-existing exact-match caller gets byte-for-
// byte identical output, and the error strings are part of what the
// goldens and the CLI both observe.
//
// This is the ONE gather path per shape (D-01): Node() and
// (*Engine).NodeDetail both call this exact function, so CLI, MCP and UI
// cannot disagree about what a node is.
func (e *Engine) buildNodeDetail(symbol, file string, line *int) (NodeDetail, error) {
	if symbol == "" {
		if file == "" {
			return NodeDetail{}, invalidArgumentf("query: node requires a symbol name or a file path")
		}
		fd, err := e.buildFileNodeDetail(file)
		if err != nil {
			return NodeDetail{}, err
		}
		return NodeDetail{Mode: NodeDetailModeFile, File: fd}, nil
	}

	// NODE-04: file supplied with no line hint tries the pre-CR-02
	// exact-match single-def path first, returning immediately on
	// success — unchanged output for every existing exact-match caller.
	if file != "" && line == nil {
		if node, err := e.resolveNodeForDetail(symbol, file); err == nil {
			d, err := e.buildSingleDefDetail(node)
			if err != nil {
				return NodeDetail{}, err
			}
			return NodeDetail{Mode: NodeDetailModeSingleDef, Definition: d}, nil
		}
	}

	matches, err := e.enumerateSymbolDefs(symbol)
	if err != nil {
		return NodeDetail{}, err
	}
	if len(matches) == 0 {
		return NodeDetail{}, notFoundf("query: symbol %q not found", symbol)
	}

	// NODE-03: narrow by substring file hint and/or line containment; a
	// no-op when neither hint is set (narrowNodeMatches returns matches
	// unchanged), preserving NODE-01/02's identical behavior for the
	// plain `node <symbol>` case.
	matches = narrowNodeMatches(matches, file, line)

	if len(matches) == 1 {
		d, err := e.buildSingleDefDetail(matches[0])
		if err != nil {
			return NodeDetail{}, err
		}
		return NodeDetail{Mode: NodeDetailModeSingleDef, Definition: d}, nil
	}

	md, err := e.buildMultiDefDetail(symbol, matches)
	if err != nil {
		return NodeDetail{}, err
	}
	return NodeDetail{Mode: NodeDetailModeMultiDef, Multi: md}, nil
}

// NodeDetail is the exported ENG-01 seam: the same gather Node() renders
// from (buildNodeDetail), returned structured and unrendered so a wire
// consumer (the UI RPC layer, from Phase 3) can map it without going
// through markdown. It is a plain Go type with no protobuf or wire
// dependency (D-03) — internal/query imports nothing from the UI wire
// layer.
func (e *Engine) NodeDetail(symbol, file string, line *int) (NodeDetail, error) {
	return e.buildNodeDetail(symbol, file, line)
}

// SourceFor reads relPath fresh from disk through the same repo-root-
// confined readSourceFile safety gate Node (file mode) and the multi-
// definition path already use (T-03-06-Path) — the only sanctioned way
// for a caller outside this package to obtain verbatim source. This is a
// wrapper over the existing confinement gate, not a second read path.
func (e *Engine) SourceFor(path string) ([]byte, error) {
	return e.readSourceFile(path)
}

// ExploreResult is the structured ENG-02 seam Explore() renders from (D-01,
// D-02, D-03): a plain Go struct, not a generated protobuf message, so
// internal/query stays wire-agnostic and the RPC layer owns the mapping to
// wire messages. The field set is exactly the argument tuple RenderExplore
// already takes, plus an Empty mode flag covering every one of Explore()'s
// five early "no results" returns — nothing invented (D-01's "the view
// model is the argument tuple the render function already takes").
//
// When Empty is true, Query and Stale carry the values the pre-extraction
// exploreZeroResult(query, stale) call would have used, and every other
// field is its zero value; Explore() renders exploreZeroResult(Query,
// Stale) for that case rather than calling RenderExplore. This is the
// ExploreResult twin of NodeDetail's Mode discriminator (D-02) — a boolean
// is sufficient here because there are exactly two shapes (empty vs
// populated), not three.
//
// Sources is keyed by each group's Path (ExploreFileGroup.Path) — this
// keying is what plan 01-09's wire mapping must reproduce, and it is not
// obvious from the type alone.
type ExploreResult struct {
	Query         string
	Empty         bool
	Stale         bool
	Groups        []ExploreFileGroup
	SymbolCount   int
	Blasts        []ExploreBlast
	Sources       map[string][]byte
	SkeletonFiles map[string]bool
}

// buildExploreResult is Explore()'s entire gather pipeline (D-01, ENG-02),
// returning the structured ExploreResult instead of rendering it. This is
// the ONE gather path for Explore — Explore() and (*Engine).ExploreDetail
// both call this exact function, so CLI, MCP and UI cannot disagree about
// what an exploration is (mirroring buildNodeDetail's role for ENG-01).
//
// Explore() has FIVE early "no results" returns — the pipeline can come up
// empty at several distinct stages (H13's seeding, H11/H12's expansion,
// H15's hard exclusion, H18's file sort, and H19's file-group assembly).
// Every one funnels through this function's ExploreResult{Empty: true, ...}
// return instead of escaping as a pre-rendered string — a missed branch
// would be a silent hole in the ENG-02 seam Phases 3-6 are forbidden from
// being planned around until it is proven complete (D-02).
func (e *Engine) buildExploreResult(query string, maxFiles int) (ExploreResult, error) {
	if strings.TrimSpace(query) == "" {
		return ExploreResult{}, invalidArgumentf("query: explore query must not be empty")
	}
	if err := validateMaxFiles(maxFiles); err != nil {
		return ExploreResult{}, err
	}
	explicitMaxFiles := maxFiles
	maxFiles = clampMaxFiles(maxFiles)

	meta, err := e.reader.GetMeta()
	if err != nil && !errors.Is(err, graphstore.ErrNotFound) {
		return ExploreResult{}, err
	}
	stale, err := e.computeStale(meta)
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{}, err
	}
	ch2, err := gatherChannel2(e.reader, symbols)
	if err != nil {
		return ExploreResult{}, err
	}
	ch3, err := gatherChannel3(e.reader, terms, "")
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{}, err
	}

	if len(candidates) == 0 && len(seeds.SeedIDs) == 0 {
		return ExploreResult{Query: query, Empty: true, Stale: stale}, nil
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
		return ExploreResult{}, err
	}

	// H11: bounded BFS from the gathered/reranked candidates.
	bfsNodeIDs, bfsRootIDs, _, err := expandBFS(e.reader, candidates, DefaultExploreBFSBounds)
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{}, err
	}

	// H12: glue-node injection around the BFS roots and named seeds.
	glueRoots := make([]string, 0, len(bfsRootIDs)+len(seeds.SeedIDs))
	glueRoots = append(glueRoots, bfsRootIDs...)
	glueRoots = append(glueRoots, seeds.SeedIDs...)
	glueIDs, err := expandGlueNodes(e.reader, glueRoots, surfacedFiles, GlueNodeCap)
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{Query: query, Empty: true, Stale: stale}, nil
	}

	// Rebuild the RankEdges-filtered edge list over the FULL final node
	// set: expandBFS's own internal edge list is scoped to just its own
	// BFS-discovered nodes, but H10/H12/H13 may have added nodes (and thus
	// edges among them) it never saw.
	_, allRankEdges, err := buildExpandAdjacency(e.reader)
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{}, err
	}
	fileGraphScore, err := aggregateFileGraphScore(e.reader, finalNodeIDs, rwrScores)
	if err != nil {
		return ExploreResult{}, err
	}

	// H15: hard test/spec/icon/i18n exclusion.
	applyHardTestExclusion(fileScores, query)
	if len(fileScores) == 0 {
		return ExploreResult{Query: query, Empty: true, Stale: stale}, nil
	}

	fileTermHits, err := computeFileTermHits(e.reader, finalNodeIDs, terms)
	if err != nil {
		return ExploreResult{}, err
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
		return ExploreResult{}, err
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
			return ExploreResult{}, gerr
		}
		if n.FilePath != "" {
			fileNodeCounts[n.FilePath]++
		}
	}

	// H18: the 5-tier file sort — this is the ranked file ORDER
	// groupMatchesByFile below consumes instead of lexical match order.
	fileOrder := fiveTierFileSort(gated, fileScores, fileGraphScore, centralFiles, fileTermHits, fileNodeCounts)
	if len(fileOrder) == 0 {
		return ExploreResult{Query: query, Empty: true, Stale: stale}, nil
	}

	// H21: the adaptive output budget only overrides the DEFAULT (the
	// caller passed no explicit --max-files, i.e. explicitMaxFiles<=0);
	// an explicit value's validate+clamp above stays verbatim.
	if explicitMaxFiles <= 0 {
		fileCount, cerr := e.countIndexedFiles()
		if cerr != nil {
			return ExploreResult{}, cerr
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
			return ExploreResult{}, gerr
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
			return ExploreResult{}, gerr
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
		return ExploreResult{Query: query, Empty: true, Stale: stale}, nil
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return ExploreResult{}, err
	}

	blasts := make([]ExploreBlast, 0, symbolCount)
	for _, g := range groups {
		for _, n := range g.Symbols {
			bl, err := e.buildBlastEntry(n, rev)
			if err != nil {
				return ExploreResult{}, err
			}
			blasts = append(blasts, bl)
		}
	}

	sources := make(map[string][]byte, len(groups))
	for _, g := range groups {
		content, err := e.readSourceFile(g.Path)
		if err != nil {
			return ExploreResult{}, err
		}
		sources[g.Path] = content
	}

	// H20: polymorphic-sibling skeletonization for off-spine files.
	implementsIdx, err := BuildImplementsIndex(e.reader)
	if err != nil {
		return ExploreResult{}, err
	}
	skeletonFiles := computeSkeletonFiles(groups, centralFiles, implementsIdx)

	return ExploreResult{
		Query:         query,
		Stale:         stale,
		Groups:        groups,
		SymbolCount:   symbolCount,
		Blasts:        blasts,
		Sources:       sources,
		SkeletonFiles: skeletonFiles,
	}, nil
}

// ExploreDetail is the exported ENG-02 seam: the same gather Explore()
// renders from (buildExploreResult), returned structured and unrendered so
// a wire consumer (the UI RPC layer, from Phase 3) can map it without going
// through markdown. It is named ExploreDetail rather than ExploreResult so
// the method name and the type name cannot be confused at a call site in
// another package, and so godoc reads unambiguously — mirroring
// (*Engine).NodeDetail's naming choice (plan 01-04). It is a plain Go type
// with no protobuf or wire dependency (D-03) — internal/query imports
// nothing from the UI wire layer.
func (e *Engine) ExploreDetail(query string, maxFiles int) (ExploreResult, error) {
	return e.buildExploreResult(query, maxFiles)
}
