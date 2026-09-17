package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// newExploreCmd builds `codegraph explore <query...>` (QRY-08, the
// flagship command, D-05a, EXPL-01): matched symbols grouped by file
// (capped at --max-files distinct files), each file's blast radius, and
// each selected file's verbatim source read fresh from disk in one round
// trip. Emits markdown text only — no --json, per D-01b (the golden
// explore.json corpus wraps this exact markdown as {command, output}).
// query is variadic (cobra.MinimumNArgs(1) + strings.Join) so a
// multi-word query like `explore user account manager` tokenizes as one
// query string instead of being rejected (RESEARCH §10 — the old
// cobra.ExactArgs(1) forced quoting for every multi-word query). No MCP
// change: the MCP surface already passes a single string arg to the same
// Engine.Explore (EXPL-05).
func newExploreCmd() *cobra.Command {
	var path string
	var maxFiles int

	cmd := &cobra.Command{
		Use:   "explore <query...>",
		Short: "Explore relevant symbols: verbatim source, call paths, blast radius",
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

			exploreQuery := strings.Join(args, " ")

			// Styled branch (D-07/D-09/D-10, CLI-01): lives strictly
			// before the eng.Explore markdown call below, so a styled run
			// never computes the markdown string and a plain run never
			// builds the ExploreDetail struct. present.RenderExplore
			// strips back byte-for-byte to that plain path's markdown
			// (04-05-SUMMARY.md's contract tests).
			mode := resolveColor(cmd)
			if mode.Styled {
				r, err := eng.ExploreDetail(exploreQuery, maxFiles)
				if err != nil {
					return err
				}
				w := mode.Writer(cmd.OutOrStdout())
				pal := present.NewPalette(mode.Dark)
				if err := present.RenderNotice(query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())), pal, w); err != nil {
					return err
				}
				return present.RenderExplore(r, pal, w)
			}

			out, err := eng.Explore(exploreQuery, maxFiles)
			if err != nil {
				return err
			}

			// Compact worktree notice (WORK-02, D-12): printed AFTER the
			// query succeeds (WR-05 — previously printed before the query
			// ran, so a failing query left a bare notice on stdout with
			// nothing to explain it; the other 6 non-status CLI commands —
			// search/callers/callees/impact/files/query/affected — already
			// print strictly after their own engine call succeeds), on
			// stdout (deliberately placed alongside normal CLI output, not
			// stderr). query.WorktreeNotice is nil-safe and returns ""
			// when there is no mismatch, so a clean tree prints nothing.
			//
			// This CLI placement extends the compact notice to all 9 read
			// commands, not just `status`. It is deliberate design,
			// granted as Claude's Discretion (02-CONTEXT.md), chosen to
			// mirror `status`'s own placement (project context, then the
			// warning, then the output) across the other 8 read commands.
			fmt.Fprint(cmd.OutOrStdout(), query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().IntVar(&maxFiles, "max-files", 0, "cap on distinct files returned (default 5)")

	return cmd
}
