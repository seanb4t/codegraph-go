package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newImpactCmd builds `codegraph impact <symbol>` (QRY-06): symbol's
// depth-bounded reverse blast radius, a BFS over the D-04 reverse-
// adjacency map bounded by --depth (clamped/validated by the engine —
// T-03-04-DoS). --json emits the golden impact.json shape
// (query.MarshalImpactJSON).
func newImpactCmd() *cobra.Command {
	var path string
	var depth int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "impact <symbol>",
		Short: "Depth-bounded reverse blast radius of a symbol",
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

			result, err := eng.Impact(args[0], depth)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalImpactJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			fmt.Fprintf(out, "%s impact (depth=%d): %d node(s), %d edge(s)\n",
				result.Symbol, result.Depth, result.NodeCount, result.EdgeCount)
			for _, l := range result.Affected {
				fmt.Fprintf(out, "  %s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().IntVarP(&depth, "depth", "d", 0, "BFS depth (default 2, max 50)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")

	return cmd
}
