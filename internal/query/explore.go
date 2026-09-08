package query

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// ExploreFileGroup is one distinct matched-file's projection: its path
// plus every matched symbol defined in it, in ranked-match order (D-05a's
// per-file "**`path`** — sym(kind), ..." header). Exported (ENG-02) as part
// of the ExploreResult seam Phases 3 through 6 consume — a consumer outside
// internal/query must be able to name this type, not just read field
// values off an already-obtained instance.
type ExploreFileGroup struct {
	Path    string
	Symbols []*schema.Node
}

// ExploreBlast is one blast-radius bullet's data (D-05a): the matched
// symbol, its total caller count, and the distinct files among those
// callers that are test symbols (D-07's isTestSymbol heuristic). Exported
// (ENG-02) as part of the ExploreResult seam Phases 3 through 6 consume —
// a consumer outside internal/query must be able to name this type, not
// just read field values off an already-obtained instance.
type ExploreBlast struct {
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
func groupMatchesByFile(ranked []rankedNode, maxFiles int) ([]ExploreFileGroup, int) {
	var groups []ExploreFileGroup
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
			groups = append(groups, ExploreFileGroup{Path: n.FilePath})
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
func (e *Engine) buildBlastEntry(n *schema.Node, rev map[string][]*schema.Edge) (ExploreBlast, error) {
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
			return ExploreBlast{}, err
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

	return ExploreBlast{Symbol: n, CallerCount: len(callers), TestFiles: testFiles}, nil
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

// Explore is the flagship one-round-trip command (QRY-08, D-05a, EXPL-02) —
// a THIN WRAPPER over buildExploreResult (D-01, ENG-02): it gathers exactly
// once via the single shared builder — the same one (*Engine).ExploreDetail
// calls (internal/query/detail.go) — then renders exploreZeroResult for the
// Empty case or calls the existing pure RenderExplore over the result's
// fields for the populated case. Explore never gathers a second time and
// never diverges from what ExploreDetail would return for the same
// arguments — CLI, MCP and UI cannot disagree about what an exploration is.
//
// See buildExploreResult's doc comment for the full pipeline this wraps:
// tokenize (H1/H2) -> hybrid gather (H3-H6) -> post-merge rerank (H7-H9) ->
// named-symbol seeding (H13) -> type-hierarchy expansion (H10) -> bounded
// BFS (H11) -> glue-node injection (H12) -> RWR graph relevance -> per-file
// score tiers + hard exclusion + buried-rescue (H14-H16) -> central-file
// selection + the 5-way relevance gate + the 5-tier sort (H17-H19) ->
// render, with H21's adaptive output budget sizing the render.
func (e *Engine) Explore(query string, maxFiles int) (string, error) {
	r, err := e.buildExploreResult(query, maxFiles)
	if err != nil {
		return "", err
	}
	if r.Empty {
		return exploreZeroResult(r.Query, r.Stale), nil
	}
	return RenderExplore(r.Query, len(r.Groups), r.SymbolCount, r.Groups, r.Blasts, r.Sources, r.Stale, r.SkeletonFiles), nil
}
