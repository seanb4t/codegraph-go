package indexer

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// countingParser wraps a real parser.Parser but is otherwise a plain
// pass-through — its only purpose is to exist as a distinct instance so
// the test factory below can count how many are constructed.
type countingParser struct {
	inner parser.Parser
}

func (c *countingParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return c.inner.Parse(source, oldTree)
}

func (c *countingParser) Close() error { return c.inner.Close() }

// init registers "go-dup" — a second registry entry whose NewParser and
// Extract are the SAME real cgo.NewGoParser/goextract.Extract as "go".
// This proves the worker pool's per-language parser-cache selection is
// keyed correctly (a genuinely distinct registry ID, not a special-cased
// string) without depending on a not-yet-shipped second language's
// extractor (RESEARCH Pitfall 1 / Open Question 1) — the mechanism under
// test is "does the worker select the right cache entry per file's
// Language field", which "go-dup" exercises identically to a real second
// language.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "go-dup",
		Extensions: []string{".godup"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewGoParser()
		},
		Extract: goextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			return relPath
		},
	})
}

// TestExtractPool_OrderStable proves Pass 1's results are index-addressed:
// the returned slice is always in the same order as the input files,
// independent of goroutine scheduling (D-01a determinism, D-04).
func TestExtractPool_OrderStable(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	for i := 0; i < 5; i++ {
		results, err := Extract(files, 0)
		if err != nil {
			t.Fatalf("Extract: %v", err)
		}
		if len(results) != len(files) {
			t.Fatalf("len(results) = %d, want %d", len(results), len(files))
		}
		for j, f := range files {
			if results[j].RelPath != f.RelPath {
				t.Fatalf("run %d: results[%d].RelPath = %q, want %q (order must match input)", i, j, results[j].RelPath, f.RelPath)
			}
			if results[j].Err != nil {
				t.Errorf("run %d: results[%d] (%s) unexpected Err: %v", i, j, f.RelPath, results[j].Err)
			}
		}
	}
}

// TestExtractPool_BoundedNotPerFile proves the pool creates at most
// `limit` parsers for a single-language batch — never one per file — by
// injecting a counting parser factory (RESEARCH Pitfall 3 / acceptance
// criteria).
func TestExtractPool_BoundedNotPerFile(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) < 3 {
		t.Fatalf("fixture has only %d files, need at least 3 to prove bounding", len(files))
	}

	const limit = 2
	var created atomic.Int64
	factory := func(languageID string) (parser.Parser, error) {
		created.Add(1)
		p, err := defaultParserFactory(languageID)
		if err != nil {
			return nil, err
		}
		return &countingParser{inner: p}, nil
	}

	results, err := extractWithFactory(files, limit, factory)
	if err != nil {
		t.Fatalf("extractWithFactory: %v", err)
	}
	if len(results) != len(files) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(files))
	}
	if got := created.Load(); got > limit {
		t.Fatalf("created %d parsers, want at most %d (one per worker, not per file)", got, limit)
	}
}

// TestExtractPool_MultiLanguage proves a batch containing files of two
// distinct registered languages ("go" and "go-dup") produces correct
// FileResults for BOTH in a single Extract() call — the Pitfall-1
// regression the pre-fix one-parser-per-worker-lifetime design could not
// survive (a worker claiming a "go" file then a "go-dup" file had no way
// to swap grammars/extractors).
func TestExtractPool_MultiLanguage(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("fixture has only %d files, need at least 2", len(files))
	}

	batch := make([]DiscoveredFile, 0, len(files)*2)
	batch = append(batch, files...)
	for _, f := range files {
		dup := f
		dup.Language = "go-dup"
		batch = append(batch, dup)
	}

	results, err := Extract(batch, 4)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(results) != len(batch) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(batch))
	}

	for i, f := range batch {
		r := results[i]
		if r.Err != nil {
			t.Errorf("results[%d] (%s, lang=%s) unexpected Err: %v", i, f.RelPath, f.Language, r.Err)
			continue
		}
		if r.RelPath != f.RelPath {
			t.Errorf("results[%d].RelPath = %q, want %q", i, r.RelPath, f.RelPath)
		}
		if len(r.Nodes) == 0 {
			t.Errorf("results[%d] (%s, lang=%s) has zero Nodes, want a real extraction", i, f.RelPath, f.Language)
		}
	}
}

// TestExtractPool_MultiLanguage_ParserCountBounded proves the total number
// of constructed parsers across a two-language batch stays bounded to
// workers * distinct-languages, never one-per-file (RESEARCH Open
// Question 1's recommended per-worker language-keyed cache bound).
func TestExtractPool_MultiLanguage_ParserCountBounded(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) < 3 {
		t.Fatalf("fixture has only %d files, need at least 3", len(files))
	}

	batch := make([]DiscoveredFile, 0, len(files)*2)
	batch = append(batch, files...)
	for _, f := range files {
		dup := f
		dup.Language = "go-dup"
		batch = append(batch, dup)
	}

	const limit = 2
	const distinctLanguages = 2
	var created atomic.Int64
	factory := func(languageID string) (parser.Parser, error) {
		created.Add(1)
		p, err := defaultParserFactory(languageID)
		if err != nil {
			return nil, err
		}
		return &countingParser{inner: p}, nil
	}

	results, err := extractWithFactory(batch, limit, factory)
	if err != nil {
		t.Fatalf("extractWithFactory: %v", err)
	}
	if len(results) != len(batch) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(batch))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d] unexpected Err: %v", i, r.Err)
		}
	}

	if got, max := created.Load(), int64(limit*distinctLanguages); got > max {
		t.Fatalf("created %d parsers, want at most %d (workers * distinct languages, never per-file)", got, max)
	}
}

// TestExtractPool_OversizedFileContained proves a single oversized file
// is recorded on its own FileResult.Err and does not abort extraction of
// the rest of the batch (RESEARCH Pitfall 4, threat T-02-03).
func TestExtractPool_OversizedFileContained(t *testing.T) {
	files, _, err := Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	dir := t.TempDir()
	bigPath := filepath.Join(dir, "big.go")
	big := make([]byte, parser.MaxSourceBytes+1)
	for i := range big {
		big[i] = 'x'
	}
	mustWrite(t, bigPath, string(big))

	all := append([]DiscoveredFile{{
		AbsPath:    bigPath,
		RelPath:    "big.go",
		ImportPath: "example.com/big",
		Language:   "go",
	}}, files...)

	results, err := Extract(all, 0)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(results) != len(all) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(all))
	}

	if results[0].RelPath != "big.go" {
		t.Fatalf("results[0].RelPath = %q, want %q", results[0].RelPath, "big.go")
	}
	if results[0].Err == nil {
		t.Fatalf("results[0].Err = nil, want parser.ErrSourceTooLarge")
	}
	if !errors.Is(results[0].Err, parser.ErrSourceTooLarge) {
		t.Errorf("results[0].Err = %v, want wrapping parser.ErrSourceTooLarge", results[0].Err)
	}

	for i := 1; i < len(results); i++ {
		if results[i].Err != nil {
			t.Errorf("results[%d] (%s) unexpected Err after oversized file: %v", i, all[i].RelPath, results[i].Err)
		}
	}
}
