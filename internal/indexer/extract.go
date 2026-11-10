package indexer

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/extract"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
)

// parserFactory constructs a fresh parser.Parser for the given registered
// language ID. Tests inject a counting factory to prove the "at most
// workers * distinct-languages parsers, never per-file" bound (D-04,
// RESEARCH Pitfall 1 / Open Question 1); production code always uses
// defaultParserFactory, which routes through the languages.go registry.
type parserFactory func(languageID string) (parser.Parser, error)

// defaultParserFactory looks up languageID in the extract.LanguageSpec registry and
// constructs a parser via its NewParser — the single dispatch point every
// worker's per-language parser cache (below) consults.
func defaultParserFactory(languageID string) (parser.Parser, error) {
	spec, ok := extract.ByID(languageID)
	if !ok {
		return nil, fmt.Errorf("indexer: no registered language %q", languageID)
	}
	return spec.NewParser()
}

// Extract runs Pass 1 of the two-pass indexing pipeline (D-04): a bounded
// pool of persistent workers walks every discovered file and produces one
// goextract.FileResult per file. Each worker owns a small per-language
// parser cache (map[string]parser.Parser, lazily populated, closed at
// worker exit) instead of a single parser for its whole lifetime — the
// Pitfall-1 fix required the moment a single Extract() call spans more
// than one language: a worker that claims a Java file then a TypeScript
// file must be able to swap grammars without reconstructing a parser per
// file.
//
// limit caps the number of workers (and therefore the parser-cache count);
// limit <= 0 defaults to DefaultWorkers().
//
// Results are written to a pre-allocated slice at each file's own index —
// never appended in completion order — so the returned slice is always in
// the SAME order as files, regardless of which worker happened to claim
// which file or when it finished (the first line of defense for D-01a
// determinism; RESEARCH Pattern 2).
//
// A per-file extraction failure (e.g. parser.ErrSourceTooLarge) is
// recorded on that file's FileResult.Err and does NOT abort the rest of
// the batch (RESEARCH Pitfall 4, threat T-02-03) — Extract itself returns
// a non-nil error only for a condition that makes the whole batch
// meaningless, such as a worker failing to construct a language's parser,
// an unregistered language, or an unreadable file.
func Extract(files []discover.DiscoveredFile, limit int) ([]goextract.FileResult, error) {
	return extractWithFactory(files, limit, defaultParserFactory)
}

// DefaultWorkers is the worker count of a run that names none: the value
// of runtime.GOMAXPROCS(0). Unlike runtime.NumCPU(), it follows the CPU
// limit of a container, so a run in a limited container does not start
// more workers than the limit lets run at once.
func DefaultWorkers() int {
	return runtime.GOMAXPROCS(0)
}

// parserCache holds the parsers of one worker, one for each language that
// the worker met. A worker makes a parser the first time it sees a
// language and uses it for every later file of that language, so a run
// makes at most one parser for each worker and language, never one for
// each file (Pitfall 1).
type parserCache struct {
	newParser parserFactory
	parsers   map[string]parser.Parser
}

func newParserCache(newParser parserFactory) *parserCache {
	return &parserCache{newParser: newParser, parsers: make(map[string]parser.Parser)}
}

// get returns the parser of language, and makes it on first use.
func (c *parserCache) get(language string) (parser.Parser, error) {
	if p, ok := c.parsers[language]; ok {
		return p, nil
	}
	p, err := c.newParser(language)
	if err != nil {
		return nil, fmt.Errorf("indexer: creating %s parser: %w", language, err)
	}
	c.parsers[language] = p
	return p, nil
}

func (c *parserCache) close() {
	for _, p := range c.parsers {
		p.Close()
	}
}

// parserSetup returns the setup of a forEachFile loop whose workers need a
// parser cache.
func parserSetup(newParser parserFactory) func() (*parserCache, func(), error) {
	return func() (*parserCache, func(), error) {
		c := newParserCache(newParser)
		return c, c.close, nil
	}
}

// extractWithFactory is Extract's implementation, parameterized on the
// parser factory so tests can inject a counting stand-in without depending
// on CGo parser internals. Each result goes to the index of its file, so
// the returned slice is in the order of files, whichever worker took
// which file.
func extractWithFactory(files []discover.DiscoveredFile, limit int, newParser parserFactory) ([]goextract.FileResult, error) {
	results := make([]goextract.FileResult, len(files))
	err := forEachFile(context.Background(), claimLoop{step: stepExtract}, len(files), limit, parserSetup(newParser),
		func(_ context.Context, parsers *parserCache, i int, release func()) error {
			defer release()
			r, err := extractOne(parsers, files[i])
			if err != nil {
				return err
			}
			results[i] = r
			return nil
		})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// extractOne is the per-file body that Extract and pass 1 of a full index
// share. It reads the file, takes the parser of its language from the
// cache of the worker, extracts the file, and stamps the stat data that
// discovery captured (Phase 4 D-01a).
//
// A file that cannot be read gives a result with Err set and no error: the
// failure is that of one file, not of the batch (Pitfall 4). An error
// means that the whole batch is void: no registered language, a parser
// that cannot be made, or an extractor that failed on the file.
func extractOne(parsers *parserCache, f discover.DiscoveredFile) (goextract.FileResult, error) {
	src, err := os.ReadFile(f.AbsPath)
	if err != nil {
		return goextract.FileResult{
			ImportPath:  f.ImportPath,
			RelPath:     f.RelPath,
			Language:    f.Language,
			MtimeUnixNs: f.MtimeUnixNs,
			SizeBytes:   f.SizeBytes,
			Err:         fmt.Errorf("indexer: reading %s: %w", f.AbsPath, err),
		}, nil
	}
	// Discover only emits files whose extension matched a registered
	// language, so a missing one means the registry changed under a
	// running batch, which no single file can answer for.
	spec, ok := extract.ByID(f.Language)
	if !ok {
		return goextract.FileResult{}, fmt.Errorf("indexer: no registered language %q for %s", f.Language, f.RelPath)
	}
	p, err := parsers.get(f.Language)
	if err != nil {
		return goextract.FileResult{}, err
	}
	r, err := extract.File(spec, p, f.ImportPath, f.RelPath, src)
	if err != nil {
		return goextract.FileResult{}, err
	}
	r.MtimeUnixNs = f.MtimeUnixNs
	r.SizeBytes = f.SizeBytes
	return r, nil
}
