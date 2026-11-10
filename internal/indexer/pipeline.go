package indexer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// Options configures one Run invocation.
type Options struct {
	// Workers bounds the worker pool of every indexing pass. A value of 0
	// or below means DefaultWorkers().
	Workers int

	// Verbose and Quiet are carried through for the CLI's summary output
	// (Stats already reports Unresolved/Skipped counts regardless of
	// these flags); Run itself does no logging.
	Verbose bool
	Quiet   bool

	// MaxChanged makes Sync return *ErrTooManyChanges, before it writes
	// anything, when more files were added, modified or deleted than it
	// allows. Zero means no limit. Run ignores it, and so does a Sync that
	// rebuilds a store without the file index.
	MaxChanged int
}

// ErrTooManyChanges is the error Sync returns when more files changed than
// Options.MaxChanged allows. The store is unchanged. Match it with
// errors.As.
type ErrTooManyChanges struct {
	// Changed counts the added, modified and deleted files.
	Changed int
	// Total counts the files discovered in the repository.
	Total int
}

func (e *ErrTooManyChanges) Error() string {
	return fmt.Sprintf("indexer: %d of %d files changed, more than the sync limit allows", e.Changed, e.Total)
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

	// PriorGenerationErr is the error of a full index that could not read
	// the coverage generation of the store it replaces. The run went on
	// and stamped generation 1. Nil means that the read worked, or that
	// the store was never indexed.
	PriorGenerationErr error
}

// Run is a full index of the repository at repoRoot into the store at
// storeDir. It builds the new graph in a scratch directory beside the
// store and puts it in the place of the store in one atomic step, so a
// run that fails or stops halfway leaves the store as it was. See run for
// the steps.
func Run(repoRoot, storeDir string, opts Options) (Stats, error) {
	return run(context.Background(), repoRoot, storeDir, opts)
}

// internalError returns the first extraction failure that is a defect of
// the indexer rather than of the source file, such as a provisional id
// that Finalize could not rewrite. A file that does not parse is skipped
// and counted; a file the indexer mishandles fails the run, so the
// defect is seen.
func internalError(results []goextract.FileResult) error {
	for _, r := range results {
		if r.Err != nil && errors.Is(r.Err, goextract.ErrInternal) {
			return fmt.Errorf("extract %s: %w", r.RelPath, r.Err)
		}
	}
	return nil
}
