package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newAffectedCmd builds `codegraph affected <files...>` (QRY-06's
// affected sibling, D-07): the test symbols reachable from the given
// changed files via the D-04 reverse-adjacency map, filtered to
// isTestSymbol matches (query-time derivation, not a persisted
// test-coverage edge — see D-07's documented divergence). --json emits
// this plan's own AffectedResult shape (query.MarshalAffectedJSON) — no
// golden oracle exists for this command (D-07a).
func newAffectedCmd() *cobra.Command {
	var path string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "affected <files...>",
		Short: "List test symbols impacted by changes to the given files",
		Args:  cobra.MinimumNArgs(1),
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

			result, err := eng.Affected(args)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalAffectedJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale. WR-04: this
			// command was previously omitted (all 7 other read commands had
			// it) with no documented reason — an oversight, now closed.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			fmt.Fprintf(out, "%d affected test(s):\n", len(result.AffectedTests))
			for _, l := range result.AffectedTests {
				fmt.Fprintf(out, "  %s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")

	return cmd
}
