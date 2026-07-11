package indexer

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// parserFactory constructs a fresh parser.Parser for one worker's entire
// lifetime. Tests inject a counting factory to prove the "one Parser per
// worker, never per file" bound (D-04, RESEARCH Pitfall 3); production
// code always uses defaultParserFactory.
type parserFactory func() (parser.Parser, error)

func defaultParserFactory() (parser.Parser, error) {
	return cgo.NewGoParser()
}

// Extract runs Pass 1 of the two-pass indexing pipeline (D-04): a bounded
// pool of persistent workers, each owning exactly one parser.Parser for
// its whole lifetime, walks every discovered file and produces one
// goextract.FileResult per file.
//
// limit caps the number of workers (and therefore the number of parsers
// constructed); limit <= 0 defaults to runtime.NumCPU().
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
// meaningless, such as a worker failing to construct its parser or an
// unreadable file.
func Extract(files []DiscoveredFile, limit int) ([]goextract.FileResult, error) {
	return extractWithFactory(files, limit, defaultParserFactory)
}

// extractWithFactory is Extract's implementation, parameterized on the
// parser factory so tests can inject a counting stand-in without
// depending on CGo parser internals.
//
// Design note (deviation from the naive `errgroup.Group.SetLimit` +
// "create a parser inside each g.Go closure" idiom): that pattern spawns
// one goroutine execution PER FILE (SetLimit only bounds how many run
// CONCURRENTLY, it does not turn separate g.Go calls into a fixed set of
// long-lived, file-reusing workers) — so it would construct one parser
// per file whenever there are more files than the limit, violating "one
// parser per worker, never per file." Instead, exactly `min(limit,
// len(files))` persistent worker goroutines are started up front, each
// creating ONE parser and then pulling file indices from a shared atomic
// counter until none remain — the actual shape "one Parser per worker"
// requires.
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
			p, err := newParser()
			if err != nil {
				return fmt.Errorf("indexer: creating worker parser: %w", err)
			}
			defer p.Close()

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
					// records a parse failure (Pitfall 4).
					results[i] = goextract.FileResult{
						ImportPath:  f.ImportPath,
						RelPath:     f.RelPath,
						Language:    "go",
						MtimeUnixNs: f.MtimeUnixNs,
						SizeBytes:   f.SizeBytes,
						Err:         fmt.Errorf("indexer: reading %s: %w", f.AbsPath, err),
					}
					continue
				}

				r, err := goextract.Extract(p, f.ImportPath, f.RelPath, src)
				if err != nil {
					// goextract.Extract's own contract is to never
					// return a non-nil error for a per-file problem —
					// it records Err on the result instead. Reaching
					// here means something outside that contract broke
					// (e.g. a parser factory bug), which IS fatal to
					// the batch.
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
