package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// priorCoverageGeneration (CR-01, 10-REVIEW.md iteration 3) reads
// Meta.CoverageGeneration from whatever store currently lives at
// storeDir, BEFORE this command's own RemoveAll wipes it — read-only,
// via graphstore's normal Open/Snapshot path (no Writer is ever opened),
// and the store is fully Closed before returning, releasing Pebble's
// lock file so the RemoveAll that follows never contends with a lock
// this call is still holding (mirrors needsFileIndexBackfill's own
// isolated open/close discipline, internal/indexer/sync.go).
//
// Every failure mode tolerates to 0: a missing/never-indexed store, a
// store with no Meta record yet, or any other Open/read error (a
// corrupted or otherwise unreadable store) all report "nothing to float
// against". This call can only ever RAISE the floor writeGraph would
// otherwise stamp on its own — never lower it — so a failure here
// cannot regress behavior below what iteration 2's fix already
// provided; it can only fail to close the gap CR-01 identifies.
func priorCoverageGeneration(storeDir string) int64 {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return 0
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		return 0
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		return 0
	}
	return meta.GetCoverageGeneration()
}

// newIndexCmd builds the `codegraph index` command: a deterministic
// from-scratch rebuild against an already-initialized .codegraph/
// (INDX-02). Requires .codegraph/ to already exist — ErrNotInitialized
// otherwise, directing the user to `codegraph init` first (D-01a).
// --force rebuilds without prompting; without --force the user must
// confirm interactively, since a from-scratch rebuild replaces the
// existing store outright rather than layering on top of it.
func newIndexCmd() *cobra.Command {
	var force, quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "index [path]",
		Short: "Deterministically rebuild the graph from scratch",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}

			codegraphDir := filepath.Join(root, codegraphDirName)
			if _, err := os.Stat(codegraphDir); os.IsNotExist(err) {
				return fmt.Errorf("%w: %s does not exist — run `codegraph init` first", ErrNotInitialized, codegraphDir)
			} else if err != nil {
				return err
			}

			if !force {
				ok, err := confirm(cmd, fmt.Sprintf("Rebuild the graph at %s from scratch?", codegraphDir))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "aborted (pass --force to rebuild without confirming)")
					return nil
				}
			}

			storeDir := filepath.Join(codegraphDir, storeDirName)
			// CR-01 (10-REVIEW.md iteration 3): read the store's own
			// prior CoverageGeneration BEFORE wiping it below, so it can
			// float the rebuild's stamped generation at least that high
			// (Options.CoverageGenerationFloor, internal/indexer's
			// pipeline.go). Without this, the wiped store's first
			// coverage commit reads back ErrNotFound from its own
			// now-empty Meta and resets to generation 1 on every single
			// `codegraph index` run, aliasing onto whatever generation a
			// still-live client's outstanding page token already
			// carries (10-REVIEW.md CR-01's concrete two-request
			// reproduction).
			coverageGenerationFloor := priorCoverageGeneration(storeDir)

			// A from-scratch rebuild must be deterministic (D-01a): clear
			// any prior store contents before re-running the pipeline
			// rather than layering a new run on top of stale state, so
			// the result never depends on what was there before.
			if err := os.RemoveAll(storeDir); err != nil {
				return err
			}
			if err := os.MkdirAll(storeDir, 0o755); err != nil {
				return err
			}

			// TUI-05/D-07/D-08: same TTY-gated spinner as init.go, via the
			// shared helper (WR-04 06-REVIEW.md, see progress_cli.go). Stop
			// is deferred so teardown runs even on indexer.Run error.
			defer startProgress(quiet, "indexing")()

			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers:                 workers,
				Verbose:                 verbose,
				Quiet:                   quiet,
				CoverageGenerationFloor: coverageGenerationFloor,
			})
			if err != nil {
				return err
			}
			printSummary(cmd, stats, quiet, verbose)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "rebuild without prompting for confirmation")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the extraction worker pool (default: number of CPUs)")

	return cmd
}
