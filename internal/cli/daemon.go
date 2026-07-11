package cli

import (
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
)

// newDaemonCmd builds the `codegraph daemon` command (D-05, SYNC-04): the
// long-running shared watch/index server multiple agent sessions share.
// Mirrors newServeCmd's -p/--path resolution, then blocks on
// daemon.Run(ctx) — the long-running analog of server.ServeStdio — until
// Ctrl-C/SIGTERM cancels ctx, at which point Run releases the lockfile and
// returns cleanly.
func newDaemonCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the shared watch/index server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			d, err := daemon.New(start)
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return d.Run(ctx)
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")

	return cmd
}
