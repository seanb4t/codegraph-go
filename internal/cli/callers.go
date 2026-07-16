package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newCallersCmd builds `codegraph callers <symbol>` (QRY-04): symbol's
// reverse callers, computed via the D-04 in-memory reverse-adjacency map.
// --json emits the golden callers.json shape (query.MarshalCallersJSON).
func newCallersCmd() *cobra.Command {
	var path string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "callers <symbol>",
		Short: "List a symbol's reverse callers",
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

			result, err := eng.Callers(args[0], limit)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalCallersJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch()))
			fmt.Fprintf(out, "%s has %d caller(s):\n", result.Symbol, len(result.Callers))
			for _, l := range result.Callers {
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
