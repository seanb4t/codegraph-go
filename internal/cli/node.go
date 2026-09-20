package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// newNodeCmd builds `codegraph node [symbol]` (QRY-02, D-05b): symbol
// detail (signature, location, forward/reverse call trail) when symbol is
// given, or a line-numbered verbatim read of --file when symbol is
// omitted. --file additionally disambiguates symbol when both are
// supplied (multiple same-named symbols, matched to the one defined in
// file). --line optionally narrows a multi-definition match further
// (NODE-03) — the line number containing (or nearest to) the intended
// definition; 0 (the flag's default, and not a valid 1-indexed line
// number) means "no line hint", matching narrowNodeMatches'/Engine.Node's
// own nil-means-unset convention. Emits markdown text only — no --json,
// per D-01b (the golden node.json corpus wraps this exact markdown as
// {command, output}).
func newNodeCmd() *cobra.Command {
	var path, file string
	var line int

	cmd := &cobra.Command{
		Use:   "node [symbol]",
		Short: "Show a symbol's signature, calls, and callers, or a line-numbered file read",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var symbol string
			if len(args) > 0 {
				symbol = args[0]
			}

			var lineHint *int
			if line != 0 {
				lineHint = &line
			}

			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			eng, closer, err := query.OpenAt(start)
			if err != nil {
				return err
			}
			defer closer.Close()

			// Styled branch (D-07/D-09/D-10, CLI-01): lives strictly
			// before the eng.Node markdown call below, so a styled run
			// never computes the markdown string and a plain run never
			// builds the NodeDetail struct. present.RenderNode strips
			// back byte-for-byte to that plain path's markdown
			// (04-05-SUMMARY.md's contract tests).
			mode := resolveColor(cmd)
			if mode.Styled {
				d, err := eng.NodeDetail(symbol, file, lineHint)
				if err != nil {
					return err
				}
				w := mode.Writer(cmd.OutOrStdout())
				pal := present.NewPalette(mode.Dark)
				if err := present.RenderNotice(query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())), pal, w); err != nil {
					return err
				}
				return present.RenderNode(d, pal, w)
			}

			out, err := eng.Node(symbol, file, lineHint)
			if err != nil {
				return err
			}

			// Compact worktree notice (WORK-02, D-12) — printed AFTER the
			// query succeeds (WR-05, see explore.go's call site for the full
			// rationale). This placement is codegraph-go's own design call:
			// a failing query must not leave a bare notice on stdout.
			fmt.Fprint(cmd.OutOrStdout(), query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "file path — disambiguates symbol, or selects file-mode when symbol is omitted")
	cmd.Flags().IntVarP(&line, "line", "l", 0, "line number — narrows an overloaded symbol to the definition containing (or nearest) this line (NODE-03)")

	return cmd
}
