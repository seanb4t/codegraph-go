package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// newSyncCmd builds the `codegraph sync` command: the user-facing entry to
// indexer.Sync (INDX-03) — a store-seeded incremental update rather than a
// from-scratch rebuild. Mirrors newIndexCmd exactly: requires .codegraph/
// to already exist (ErrNotInitialized otherwise, directing the user to
// `codegraph init` first), same --quiet/--verbose flags (D-01b).
func newSyncCmd() *cobra.Command {
	var quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "sync [path]",
		Short: "Incrementally update the graph from changed files",
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

			storeDir := filepath.Join(codegraphDir, storeDirName)

			// TUI-05/D-07/D-08: same TTY-gated spinner as init.go/index.go,
			// via the shared helper (WR-04 06-REVIEW.md, see
			// progress_cli.go). Stop is deferred so teardown runs even on
			// indexer.Sync error.
			defer startProgress(cmd.Context(), quiet, "syncing")()

			stats, err := indexer.Sync(root, storeDir, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if err != nil {
				return err
			}
			printSyncSummary(cmd, stats, quiet, verbose)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the extraction worker pool (default: number of CPUs)")

	return cmd
}

// printSyncSummary extends printSummary's default files/nodes/edges/
// duration line with a second, sync-only line (D-01b): files reparsed/
// pruned, nodes/edges removed by pruning, and dependents recomputed purely
// because a symbol they referenced was pruned. Reuses printSummary rather
// than forking a second summary printer — --quiet suppresses both lines
// identically via printSummary's own guard.
func printSyncSummary(cmd *cobra.Command, stats indexer.Stats, quiet, verbose bool) {
	printSummary(cmd, stats, quiet, verbose)
	if quiet {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "reparsed=%d pruned=%d nodesRemoved=%d edgesRemoved=%d dependentsRecomputed=%d\n",
		stats.FilesReparsed, stats.FilesPruned, stats.NodesRemoved, stats.EdgesRemoved, stats.DependentsRecomputed)
}
