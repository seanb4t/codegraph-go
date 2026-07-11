package indexer

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/parser"
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
// `limit` parsers — never one per file — by injecting a counting parser
// factory (RESEARCH Pitfall 3 / acceptance criteria).
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
	factory := func() (parser.Parser, error) {
		created.Add(1)
		p, err := defaultParserFactory()
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
