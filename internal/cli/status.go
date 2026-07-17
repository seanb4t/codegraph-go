package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// newStatusCmd builds `codegraph status` (QRY-09): reports index
// health/counts by scanning the frozen graph (see query.StatusResult's
// doc comment for the full TS-to-Go/Pebble key remapping table, D-05).
// The human branch renders D-09's sectioned plain-text layout via
// query.RenderStatusText (CodeGraph Status / Project: / the verbose
// worktree warning when present / Index Statistics: / Nodes by Kind: /
// Files by Language: / the live staleness/reindex advisory) — replacing
// the old terse `backend=… files=… stale=…` one-liner. On a real TTY
// (and NO_COLOR unset), present.RenderStatus renders the same content
// lipgloss-styled instead (TUI-02, D-03/D-04/D-05); piped/non-TTY output
// stays byte-identical to this plain path.
// --json emits query.MarshalStatusJSON's StatusResult shape.
// That function body is UNCHANGED by this plan — it is shared with the
// CLI --json contract and the golden-parity oracle (D-16/D-17).
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

			result, err := eng.Status(cmd.Context())
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

			if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
				return present.RenderStatus(result, start, cmd.OutOrStdout())
			}

			// RenderStatusText already embeds the verbose worktree warning
			// (from result.WorktreeMismatch, live since plan 02-04) at D-09's
			// structural position — no separate warning print here, which
			// would double it.
			fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")

	return cmd
}
