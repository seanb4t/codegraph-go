package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newExploreCmd builds `codegraph explore <query>` (QRY-08, the flagship
// command, D-05a): matched symbols grouped by file (capped at
// --max-files distinct files), each file's blast radius, and each
// selected file's verbatim source read fresh from disk in one round
// trip. Emits markdown text only — no --json, per D-01b (the golden
// explore.json corpus wraps this exact markdown as {command, output}).
func newExploreCmd() *cobra.Command {
	var path string
	var maxFiles int

	cmd := &cobra.Command{
		Use:   "explore <query>",
		Short: "Explore relevant symbols: verbatim source, call paths, blast radius",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			eng, closer, err := query.OpenAt(start)
			if err != nil {
				return err
			}
			defer closer.Close()

			out, err := eng.Explore(args[0], maxFiles)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().IntVar(&maxFiles, "max-files", 0, "cap on distinct files returned (default 5)")

	return cmd
}
