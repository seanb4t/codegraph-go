package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
)

// newUnlockCmd builds the `codegraph unlock` command (D-05, SYNC-05):
// clears a stale daemon lockfile left behind by a crash. Mirrors
// newUninitCmd's guarded shape (targetRoot(args) positional-arg
// resolution), but delegates the stale-vs-live decision to daemon.Unlock —
// the CLI never force-removes the lockfile itself (T-04-07-01).
// daemon.Unlock already treats an absent lockfile as a clean no-op (its
// own human-readable message) and refuses a live lock via ErrLockLive, so
// there is no separate guard here and no confirm prompt: unlock only ever
// removes a genuinely stale lock.
func newUnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock [path]",
		Short: "Clear a stale daemon lock",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}

			codegraphDir := filepath.Join(root, codegraphDirName)
			msg, err := daemon.Unlock(codegraphDir)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), msg)
			return nil
		},
	}

	return cmd
}
