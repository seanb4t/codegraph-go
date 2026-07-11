package query

import (
	"fmt"
	"sort"

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

// Explore is the flagship one-round-trip command (QRY-08, D-05a): select
// symbols matching query via the 03-03 lexical matcher, group them by
// file capped at maxFiles distinct files, compute each matched symbol's
// blast radius via the D-04 reverse-adjacency map, then render each
// selected file's verbatim source read fresh from disk (D-05a — never
// from the stored Node/File record), confined to the repo root
// (T-03-06-Path).
func (e *Engine) Explore(query string, maxFiles int) (string, error) {
	if err := validateMaxFiles(maxFiles); err != nil {
		return "", err
	}
	maxFiles = clampMaxFiles(maxFiles)

	ranked, err := e.matchNodes(query, "")
	if err != nil {
		return "", err
	}

	groups, symbolCount := groupMatchesByFile(ranked, maxFiles)
	if len(groups) == 0 {
		return fmt.Sprintf("**Exploration: %s**\n\nFound 0 symbols across 0 files.\n", query), nil
	}

	rev, err := buildReverseAdjacency(e.reader)
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

	return RenderExplore(query, len(groups), symbolCount, groups, blasts, sources), nil
}
