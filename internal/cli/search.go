package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
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

// renderFullLine renders the D-01 second line for a --full hit: four-space
// indent, QualifiedName, then (if Signature is non-empty) two spaces and
// Signature. Line 1 (the default search line, byte-identical) is rendered
// separately by the existing "%s (%s) %s:%d\n" format — this helper only
// ever emits line 2. Plain concatenation: no truncation, escaping, or
// normalisation of the QualifiedName/Signature bytes (D-01).
func renderFullLine(n *schema.Node) string {
	if n.Signature == "" {
		return "    " + n.QualifiedName
	}
	return "    " + n.QualifiedName + "  " + n.Signature
}

// newSearchCmd builds `codegraph search <term>` (QRY-01 sibling; VERB-01/
// VERB-03 fold): lexical search over symbol names/qualified names. By
// default it renders the lightweight locations-only projection (name/kind/
// filePath/startLine, no source body, no signature) via eng.Search and a
// raw []query.Location JSON array. --full (Phase 3 VERB-01) selects the
// full-node-record shape that the now-removed `query` verb used to render:
// eng.Query, the query.MarshalQueryJSON envelope under --json, and a
// two-line human render (line 1 identical to the default line, line 2 the
// D-01 renderFullLine second line). The WORK-02 worktree-mismatch notice
// sits in both human branches at the same position — after the --json
// early return (D-02) — never in either JSON branch.
func newSearchCmd() *cobra.Command {
	var path, kind string
	var limit int
	var jsonOut bool
	var full bool

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Lexically search symbol names/qualified names",
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

			if full {
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
				// the human-output branch, AFTER the --json early return above,
				// in the exact same position as the default branch below (D-02).
				fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
				for _, n := range nodes {
					fmt.Fprintf(out, "%s (%s) %s:%d\n", n.Name, n.Kind, n.FilePath, n.StartLine)
					fmt.Fprintln(out, renderFullLine(n))
				}
				return nil
			}

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
			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale. This
			// placement is codegraph-go's own design call.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			for _, l := range locs {
				fmt.Fprintf(out, "%s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVarP(&kind, "kind", "k", "", "restrict to one node kind")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
	cmd.Flags().BoolVar(&full, "full", false, "return full node records (signature, qualified name) instead of locations")

	return cmd
}
