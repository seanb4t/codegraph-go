package cli

import "github.com/spf13/cobra"

// addWorkersFlag registers the --workers flag on cmd, bound to workers.
// Every command that indexes calls it, so the name, the default and the
// help text have one place. The default 0 means indexer.DefaultWorkers().
func addWorkersFlag(cmd *cobra.Command, workers *int) {
	cmd.Flags().IntVar(workers, "workers", 0, "bound the worker pool of every indexing pass (default: GOMAXPROCS)")
}
