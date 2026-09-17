package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
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

			// Styled branch (D-08/D-09/D-10, CLI-01): lives strictly after
			// the --json early return above and before the frozen plain
			// path below. present.RenderNotice + RenderCallees strip back
			// byte-for-byte to that plain path's output.
			mode := resolveColor(cmd)
			if mode.Styled {
				w := mode.Writer(out)
				pal := present.NewPalette(mode.Dark)
				if err := present.RenderNotice(query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())), pal, w); err != nil {
					return err
				}
				return present.RenderCallees(result, pal, w)
			}

			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			fmt.Fprintf(out, "%s calls %d callee(s):\n", result.Symbol, len(result.Callees))
			for _, l := range result.Callees {
				fmt.Fprintf(out, "  %s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")

	return cmd
}
