package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// resolveStartPath resolves the -p/--path flag every query/serve command
// accepts (D-01a): p itself when given, else the current working
// directory — the flag-driven analog of targetRoot's positional-arg
// cwd-default convention (init.go/index.go take a [path] arg; the query
// commands take -p/--path instead, per CONTEXT D-01a).
func resolveStartPath(p string) (string, error) {
	if p != "" {
		return p, nil
	}
	return os.Getwd()
}

// writeJSONLine writes already-marshaled JSON data to cmd's configured
// stdout followed by a trailing newline — the shared --json output
// primitive every structured command uses instead of writing to
// os.Stdout directly (03-PATTERNS.md's output-discipline pattern), for
// commands that render through a dedicated internal/query Marshal*JSON
// helper rather than encoding a Go value directly.
func writeJSONLine(cmd *cobra.Command, data []byte) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

// newQueryCmd builds `codegraph query <term>` (QRY-01): full node records
// whose name or qualifiedName matches term, optionally filtered by --kind
// and capped at --limit (D-06). --json emits the golden query.json
// envelope shape via query.MarshalQueryJSON; the default renders one
// concise line per match.
func newQueryCmd() *cobra.Command {
	var path, kind string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "query <term>",
		Short: "Search full node records by name/qualifiedName",
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

			nodes, err := eng.Query(args[0], kind, limit)
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalQueryJSON(nodes)
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
			for _, n := range nodes {
				fmt.Fprintf(out, "%s (%s) %s:%d\n", n.Name, n.Kind, n.FilePath, n.StartLine)
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
