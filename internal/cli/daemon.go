package cli

import (
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// newDaemonCmd builds the `codegraph daemon` command (D-05, SYNC-04): the
// long-running shared watch/index server multiple agent sessions share.
// Mirrors newServeCmd's -p/--path resolution, then blocks on
// daemon.Run(ctx) — the long-running analog of server.ServeStdio — until
// Ctrl-C/SIGTERM cancels ctx, at which point Run releases the lockfile and
// returns cleanly. --workers/--quiet/--verbose (WR-04) mirror `codegraph
// sync`'s own flags and thread through to every debounced indexer.Sync
// this daemon drives — the daemon has no summary output of its own (a
// failed sync is logged, not printed to this command's stdout), so
// --quiet/--verbose only affect a future daemon-side logging format, not
// this command's own output.
func newDaemonCmd() *cobra.Command {
	var path string
	var quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the shared watch/index server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			d, err := daemon.New(start, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return d.Run(ctx)
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress daemon-driven sync progress output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail for daemon-driven syncs")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the daemon's extraction worker pool (default: number of CPUs)")

	return cmd
}
