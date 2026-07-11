package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

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

			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
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
