package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newNodeCmd builds `codegraph node [symbol]` (QRY-02, D-05b): symbol
// detail (signature, location, forward/reverse call trail) when symbol is
// given, or a line-numbered verbatim read of --file when symbol is
// omitted. --file additionally disambiguates symbol when both are
// supplied (multiple same-named symbols, matched to the one defined in
// file). Emits markdown text only — no --json, per D-01b (the golden
// node.json corpus wraps this exact markdown as {command, output}).
func newNodeCmd() *cobra.Command {
	var path, file string

	cmd := &cobra.Command{
		Use:   "node [symbol]",
		Short: "Show a symbol's signature, calls, and callers, or a line-numbered file read",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var symbol string
			if len(args) > 0 {
				symbol = args[0]
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

			out, err := eng.Node(symbol, file)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "file path — disambiguates symbol, or selects file-mode when symbol is omitted")

	return cmd
}
