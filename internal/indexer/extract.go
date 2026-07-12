package indexer

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
)

// parserFactory constructs a fresh parser.Parser for the given registered
// language ID. Tests inject a counting factory to prove the "at most
// workers * distinct-languages parsers, never per-file" bound (D-04,
// RESEARCH Pitfall 1 / Open Question 1); production code always uses
// defaultParserFactory, which routes through the languages.go registry.
type parserFactory func(languageID string) (parser.Parser, error)

// defaultParserFactory looks up languageID in the LanguageSpec registry and
// constructs a parser via its NewParser — the single dispatch point every
// worker's per-language parser cache (below) consults.
func defaultParserFactory(languageID string) (parser.Parser, error) {
	spec, ok := lookupLanguageByID(languageID)
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
// limit <= 0 defaults to runtime.NumCPU().
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
func Extract(files []DiscoveredFile, limit int) ([]goextract.FileResult, error) {
	return extractWithFactory(files, limit, defaultParserFactory)
}

// extractWithFactory is Extract's implementation, parameterized on the
// parser factory so tests can inject a counting stand-in without depending
// on CGo parser internals.
//
// Design note (deviation from the naive `errgroup.Group.SetLimit` +
// "create a parser inside each g.Go closure" idiom): that pattern spawns
// one goroutine execution PER FILE (SetLimit only bounds how many run
// CONCURRENTLY, it does not turn separate g.Go calls into a fixed set of
// long-lived, file-reusing workers) — so it would construct one parser per
// file whenever there are more files than the limit, violating "bounded
// parser count, never per-file." Instead, exactly `min(limit, len(files))`
// persistent worker goroutines are started up front, each owning its own
// language-keyed parser cache and pulling file indices from a shared
// atomic counter until none remain.
func extractWithFactory(files []DiscoveredFile, limit int, newParser parserFactory) ([]goextract.FileResult, error) {
	if limit <= 0 {
		limit = runtime.NumCPU()
	}

	results := make([]goextract.FileResult, len(files))
	if len(files) == 0 {
		return results, nil
	}
	if limit > len(files) {
		limit = len(files)
	}

	var next atomic.Int64

	g := new(errgroup.Group)
	for w := 0; w < limit; w++ {
		g.Go(func() error {
			// Per-worker language-keyed parser cache (Pitfall 1 fix):
			// lazily constructed on first encounter of a language, reused
			// for every subsequent file of that same language this worker
			// claims, and closed for every cached entry at worker exit —
			// never one parser per file, never one parser total per
			// worker regardless of how many languages it sees.
			parsers := make(map[string]parser.Parser)
			defer func() {
				for _, p := range parsers {
					p.Close()
				}
			}()

			for {
				i := int(next.Add(1)) - 1
				if i >= len(files) {
					return nil
				}
				f := files[i]

				src, err := os.ReadFile(f.AbsPath)
				if err != nil {
					// A read failure is per-file, not fatal to the
					// batch — recorded the same way goextract.Extract
					// records a parse failure (Pitfall 4). Language is
					// stamped from the discovered file itself, not
					// hardcoded, so a non-Go file's read failure is
					// correctly attributed.
					results[i] = goextract.FileResult{
						ImportPath:  f.ImportPath,
						RelPath:     f.RelPath,
						Language:    f.Language,
						MtimeUnixNs: f.MtimeUnixNs,
						SizeBytes:   f.SizeBytes,
						Err:         fmt.Errorf("indexer: reading %s: %w", f.AbsPath, err),
					}
					continue
				}

				spec, ok := lookupLanguageByID(f.Language)
				if !ok {
					// An unrecognized language is a batch-fatal
					// condition — Discover only ever emits files whose
					// extension matched a registered language, so
					// reaching here means the registry changed out from
					// under a running batch, which is not a per-file
					// problem to paper over.
					return fmt.Errorf("indexer: no registered language %q for %s", f.Language, f.RelPath)
				}

				p, cached := parsers[f.Language]
				if !cached {
					p, err = newParser(f.Language)
					if err != nil {
						return fmt.Errorf("indexer: creating %s parser: %w", f.Language, err)
					}
					parsers[f.Language] = p
				}

				r, err := spec.Extract(p, f.ImportPath, f.RelPath, src)
				if err != nil {
					// spec.Extract's own contract (mirroring
					// goextract.Extract) is to never return a non-nil
					// error for a per-file problem — it records Err on
					// the result instead. Reaching here means something
					// outside that contract broke (e.g. a parser factory
					// bug), which IS fatal to the batch.
					return fmt.Errorf("indexer: extracting %s: %w", f.RelPath, err)
				}
				// Phase 4 D-01a: stamp the on-disk stat info Discover
				// already captured onto the result, so Pass 2 can carry
				// it onto the committed File record.
				r.MtimeUnixNs = f.MtimeUnixNs
				r.SizeBytes = f.SizeBytes
				// Disjoint index, unique per file — safe under
				// concurrent writes from other workers (each worker's
				// `i` is obtained from a single atomic counter, so no
				// two workers ever write the same index).
				results[i] = r
			}
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
