package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newStatusCmd builds `codegraph status` (QRY-09): reports index
// health/counts by scanning the frozen graph (see query.StatusResult's
// doc comment for the full TS-to-Go/Pebble key remapping table, D-05).
// --json emits query.MarshalStatusJSON's StatusResult shape.
func newStatusCmd() *cobra.Command {
	var path string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report index health and counts",
		Args:  cobra.NoArgs,
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

			result, err := eng.Status()
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalStatusJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "backend=%s files=%d nodes=%d edges=%d state=%s reindexRecommended=%t stale=%t\n",
				result.Backend, result.FileCount, result.NodeCount, result.EdgeCount,
				result.Index.State, result.Index.ReindexRecommended, result.Stale)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")

	return cmd
}
