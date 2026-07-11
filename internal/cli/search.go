package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newSearchCmd builds `codegraph search <term>` (QRY-01 sibling): the
// lightweight locations-only projection of query (D-06) — name/kind/
// filePath/startLine, no source body, no signature. --json emits the raw
// []query.Location array directly (no dedicated MarshalSearchJSON
// helper exists — query.Location already carries its own json tags,
// matching internal/mcp's companionHandler "search" branch, D-08b).
func newSearchCmd() *cobra.Command {
	var path, kind string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Lexically search symbol names/qualified names (locations only)",
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

			locs, err := eng.Search(args[0], kind, limit)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := json.Marshal(locs)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			for _, l := range locs {
				fmt.Fprintf(out, "%s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVar(&kind, "kind", "", "restrict to one node kind")
	cmd.Flags().IntVar(&limit, "limit", 0, "cap on results returned")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON output")

	return cmd
}
