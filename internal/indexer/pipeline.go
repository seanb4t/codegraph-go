package indexer

import (
	"runtime"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// Options configures one Run invocation.
type Options struct {
	// Workers bounds Pass 1's extraction worker pool. <= 0 defaults to
	// runtime.NumCPU() — Extract's own default (D-04), applied here too so
	// callers observe the same behavior whether they pass Options{} or
	// Options{Workers: runtime.NumCPU()} explicitly.
	Workers int

	// Verbose and Quiet are carried through for the CLI's summary output
	// (Stats already reports Unresolved/Skipped counts regardless of
	// these flags); Run itself does no logging.
	Verbose bool
	Quiet   bool
}

// Stats summarizes one Run invocation: how many files were discovered, how
// many nodes/edges landed in the committed graph, how many cross-file
// references could not be resolved (D-06a — never silently dropped), how
// many files were skipped outright (parser.ErrSourceTooLarge or a read
// failure, RESEARCH Pitfall 4), and how long the whole run took.
type Stats struct {
	Files      int
	Nodes      int
	Edges      int
	Unresolved int
	Skipped    int
	Duration   time.Duration
}

// resolveFunc is Pass 2's entry point, matching Resolve's signature. It is
// injected as a parameter to run (never hard-coded to the package-level
// Resolve) purely so a test can simulate a resolve-time failure and prove
// the GraphStore is still Closed on that path — mirroring extract.go's
// parserFactory testing seam; production code always goes through Run,
// which binds it to the real Resolve.
type resolveFunc func(store graphstore.GraphStore, results []goextract.FileResult, modulePath string) (int, error)

// Run executes the full from-scratch indexing pipeline (D-04, D-01a):
// Discover walks repoRoot for every Go source file the build context
// includes, Extract runs Pass 1 (parallel, bounded worker pool) over them,
// and Resolve runs Pass 2 (the single coordinated writer) against the
// GraphStore at storeDir, committing the resolved graph and stamping Meta.
//
// The store is opened exactly once and Closed on every return path —
// success or failure — mirroring pebble_store.Open's Close-once lifecycle
// discipline (T-02-11): a failure partway through Pass 2 never leaves an
// open engine handle or lock behind.
func Run(repoRoot, storeDir string, opts Options) (Stats, error) {
	return run(repoRoot, storeDir, opts, Resolve)
}

// run is Run's implementation, parameterized on the Pass-2 entry point so
// tests can inject a failing resolveFunc without depending on a real
// Resolve error condition.
func run(repoRoot, storeDir string, opts Options, resolve resolveFunc) (Stats, error) {
	start := time.Now()

	files, modulePath, err := Discover(repoRoot)
	if err != nil {
		return Stats{}, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	results, err := Extract(files, workers)
	if err != nil {
		return Stats{Files: len(files), Duration: time.Since(start)}, err
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		return Stats{Files: len(files), Duration: time.Since(start)}, err
	}
	defer store.Close()

	unresolved, err := resolve(store, results, modulePath)
	if err != nil {
		return Stats{Files: len(files), Unresolved: unresolved, Duration: time.Since(start)}, err
	}

	skipped := 0
	for _, r := range results {
		if r.Err != nil {
			skipped++
		}
	}

	nodes, edges, err := readGraphCounts(store)
	if err != nil {
		return Stats{}, err
	}

	return Stats{
		Files:      len(files),
		Nodes:      nodes,
		Edges:      edges,
		Unresolved: unresolved,
		Skipped:    skipped,
		Duration:   time.Since(start),
	}, nil
}

// readGraphCounts reads the just-committed Meta record's node/edge counts
// back from store via a fresh Snapshot, so Stats always reflects exactly
// what writeGraph stamped (never a separately re-derived count that could
// drift from it).
func readGraphCounts(store graphstore.GraphStore) (nodes, edges int, err error) {
	r, err := store.Snapshot()
	if err != nil {
		return 0, 0, err
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		return 0, 0, err
	}
	return int(meta.NodeCount), int(meta.EdgeCount), nil
}
