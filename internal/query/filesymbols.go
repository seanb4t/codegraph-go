package query

import (
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// MaxFileSymbols bounds the number of symbols FileSymbols returns for a
// single file (T-05-30). It has exactly ONE declaration, here, beside the
// engine method that applies it: the engine is what enforces the cap, and
// internal/uiserver imports internal/query — never the reverse
// (internal/uiserver/handlers.go:11) — so a constant declared on the wire
// side and applied here would need a duplicated literal, an undeclared
// parameter, or an import cycle (review H-2). internal/uiserver
// references query.MaxFileSymbols; it never redeclares it. 2000 is well
// above any hand-written file and below the point a generated file's
// symbol count could threaten transportSendMaxBytes.
const MaxFileSymbols = 2000

// FileSymbolsResult is Engine.FileSymbols()'s return value: the symbols
// declared in one file, capped at MaxFileSymbols, alongside the TRUE
// total (uncapped) and whether the returned list was capped. A partial
// list is never returned without both of these riding along — Total
// names the real count and Truncated says so explicitly, mirroring this
// server's existing source-truncation discipline (T-05-30).
type FileSymbolsResult struct {
	Symbols   []*schema.Node
	Total     int64
	Truncated bool
}

// FileSymbols enumerates the symbols one file declares (GRF-03), computed
// fresh per call from a full IterateNodes() scan — no cache, no
// precomputed projection, the same fresh-per-call discipline
// BuildReverseAdjacency and FileGraph already follow: a long-lived
// process must never serve a stale point-in-time symbol list across
// multiple calls.
//
// path is validated through the Engine's existing
// ValidateRepoRelativePath confinement FIRST, before the store is
// touched — a precondition, not a filter applied after scanning
// (T-05-28). A path escaping the repository root, or an absolute path,
// is rejected here and the scan never runs.
//
// FileSymbols reads INDEX records only, never the file on disk
// (T-05-32): reading source here would answer what the file currently
// contains on disk, which can disagree with what the index — and
// therefore the rest of the graph — currently believes. A path the index
// does not carry (a miss) returns an empty result with Total 0 and no
// error: the caller just received this path from another rpc reading the
// same index, so a miss means the index changed concurrently, not that
// the caller asked for something invalid.
func (e *Engine) FileSymbols(path string) (FileSymbolsResult, error) {
	if err := e.ValidateRepoRelativePath(path); err != nil {
		return FileSymbolsResult{}, err
	}

	it, err := e.reader.IterateNodes()
	if err != nil {
		return FileSymbolsResult{}, err
	}
	defer it.Close()

	var matches []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.FilePath != path {
			continue
		}
		// The file's own file-kind record is not a symbol the file
		// declares (it IS the file), and a synthetic package-kind
		// pseudo-node is excluded for the same reason it is excluded
		// from Engine.FileGraph's rollup (D-08, filegraphKindPackage).
		if n.Kind == goextract.KindFile || n.Kind == filegraphKindPackage {
			continue
		}
		matches = append(matches, n)
	}
	if err := it.Err(); err != nil {
		return FileSymbolsResult{}, err
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].StartLine != matches[j].StartLine {
			return matches[i].StartLine < matches[j].StartLine
		}
		return matches[i].Name < matches[j].Name
	})

	total := int64(len(matches))
	truncated := false
	if len(matches) > MaxFileSymbols {
		matches = matches[:MaxFileSymbols]
		truncated = true
	}

	return FileSymbolsResult{Symbols: matches, Total: total, Truncated: truncated}, nil
}
