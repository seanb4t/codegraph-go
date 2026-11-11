package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// newIndexCmd builds the `codegraph index` command: a deterministic
// from-scratch rebuild against an already-initialized .codegraph/
// (INDX-02). Requires .codegraph/ to already exist — ErrNotInitialized
// otherwise, directing the user to `codegraph init` first (D-01a).
// --force rebuilds without prompting; without --force the user must
// confirm interactively, since a from-scratch rebuild replaces the
// existing store outright rather than layering on top of it. The store
// is replaced only when the rebuild succeeds: a rebuild that fails or
// stops halfway leaves the previous graph in place.
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
			// TUI-05/D-07/D-08: same TTY-gated spinner as init.go, via the
			// shared helper (WR-04 06-REVIEW.md, see progress_cli.go). Stop
			// is deferred so teardown runs even on indexer.Run error.
			defer startProgress(quiet, "indexing")()

			// indexer.Run builds the new graph beside the store and puts
			// it in the place of the store only when it succeeds, so this
			// command no longer deletes the store first. Run also reads
			// the prior coverage generation and takes the locks itself.
			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if errors.Is(err, graphstore.ErrStoreLocked) {
				// Another process holds the store open or writes to it
				// (D-10): name the holder and the two live remedies.
				// `codegraph daemon unlock` is the live verb (Phase 3
				// VERB-04, D-08). The store did not change.
				return fmt.Errorf("%w: another process holds %s open — stop it first with `codegraph daemon stop`, or run `codegraph daemon unlock` once it has exited", err, storeDir)
			}
			if err != nil {
				return err
			}
			if stats.PriorGenerationErr != nil {
				// A store that is corrupt or cannot be read (D-11): the
				// run could not read its prior coverage generation, so
				// it started again from 0. `--force` is the sanctioned
				// way to repair exactly this case.
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not read prior coverage generation from %s (%v); rebuilding with floor 0\n", storeDir, stats.PriorGenerationErr)
			}
			warnAfterReplace(cmd, storeDir, stats)
			printSummary(cmd, stats, quiet, verbose)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "rebuild without prompting for confirmation")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
	addWorkersFlag(cmd, &workers)

	return cmd
}

// warnAfterReplace prints a warning for a full index that put its new
// store in place and then met an error. The index succeeded, so the command
// does too.
func warnAfterReplace(cmd *cobra.Command, storeDir string, stats indexer.Stats) {
	if stats.AfterReplaceErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: the new index is in place at %s, but a later step failed: %v\n", storeDir, stats.AfterReplaceErr)
	}
}
