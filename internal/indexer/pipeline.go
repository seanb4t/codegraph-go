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

	// FilesReparsed, FilesPruned, NodesRemoved, EdgesRemoved, and
	// DependentsRecomputed are Sync-only summary counts (Phase 4 D-01b) —
	// zero-valued for a from-scratch Run. FilesReparsed is the size of the
	// Extract() batch (added ∪ modified ∪ dependent); FilesPruned counts
	// modified+deleted files whose subgraph was pruned via the x/ index;
	// NodesRemoved/EdgesRemoved count the individual n/e records
	// point-deleted during that prune; DependentsRecomputed counts files
	// re-extracted purely because a symbol they referenced was pruned
	// (RESEARCH Pitfall 2), not because their own content changed.
	FilesReparsed        int
	FilesPruned          int
	NodesRemoved         int
	EdgesRemoved         int
	DependentsRecomputed int
}

// resolveFunc is Pass 2's entry point, matching Resolve's signature. It is
// injected as a parameter to run (never hard-coded to the package-level
// Resolve) purely so a test can simulate a resolve-time failure and prove
// the GraphStore is still Closed on that path — mirroring extract.go's
// parserFactory testing seam; production code always goes through Run,
// which binds it to the real Resolve.
type resolveFunc func(store graphstore.GraphStore, results []goextract.FileResult, modulePath string) (int, error)

// Run executes the full from-scratch indexing pipeline (D-04, D-01a):
// Discover walks repoRoot for every source file whose extension is claimed
// by a registered LanguageSpec (Phase 5 D-03 — Go was the only such
// language through Phase 4), Extract runs Pass 1 (parallel, bounded worker
// pool, selecting a parser+extractor per file's own Language — Phase 5
// Pitfall 1) over them, and Resolve runs Pass 2 (the single coordinated
// writer) against the GraphStore at storeDir, committing the resolved
// graph and stamping Meta.
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

	// LANG-07/D-08/D-09: opt-in, per-framework route detection runs once
	// per from-scratch Run, after Pass 1 (route detectors re-parse only
	// the files whose language has a FIRED detector — never a blanket
	// AST re-walk) and before Pass 2, so its route nodes/heuristic calls
	// edges flow through the SAME resolveRefsWithIndex/collapseEdges/
	// writeGraph path every other edge already uses (D-06/D-07: no
	// parallel commit). Wired into Run only, not Sync (incremental sync
	// stays out of this plan's scope — see 05-12-SUMMARY.md).
	routeNodes, routeEdges, err := detectRoutes(repoRoot, files, results)
	if err != nil {
		return Stats{Files: len(files), Duration: time.Since(start)}, err
	}
	if len(routeNodes) > 0 || len(routeEdges) > 0 {
		results = append(results, routeResultFrom(routeNodes, routeEdges))
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
