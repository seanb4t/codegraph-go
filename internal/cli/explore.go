package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

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

			// Compact worktree notice (WORK-02, D-12): printed immediately
			// after OpenAt succeeds, before the command's main output, on
			// stdout (TS's CLI warn() is console.log = stdout, matched here
			// for parity). query.WorktreeNotice is nil-safe and returns ""
			// when there is no mismatch, so a clean tree prints nothing.
			//
			// This CLI placement has NO TS precedent: TS never wires the
			// compact notice into any command but `status` (verified: zero
			// other call sites in mcp/tools.js's withWorktreeNotice or
			// bin/codegraph.js). It is deliberate Go-side design, granted as
			// Claude's Discretion (02-CONTEXT.md), chosen to mirror
			// `status`'s own placement (project context, then the warning,
			// then the output) across the other 7 read commands.
			fmt.Fprint(cmd.OutOrStdout(), query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))

			exploreQuery := strings.Join(args, " ")
			out, err := eng.Explore(exploreQuery, maxFiles)
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
