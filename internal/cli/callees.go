package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newCalleesCmd builds `codegraph callees <symbol>` (QRY-05): symbol's
// forward call targets, computed via a direct IterateEdges(srcID) range
// scan (D-04 — no reverse-adjacency scan needed). --json emits the golden
// callees.json shape (query.MarshalCalleesJSON).
func newCalleesCmd() *cobra.Command {
	var path string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "callees <symbol>",
		Short: "List a symbol's forward call targets",
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

			result, err := eng.Callees(args[0], limit)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalCalleesJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s calls %d callee(s):\n", result.Symbol, len(result.Callees))
			for _, l := range result.Callees {
				fmt.Fprintf(out, "  %s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().IntVar(&limit, "limit", 0, "cap on results returned")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")

	return cmd
}
