package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// newFilesCmd builds `codegraph files` (QRY-07): browses the indexed file
// structure from the frozen graph (never a live filesystem walk),
// narrowed by --pattern (glob), --filter (exact language match), --dir
// (directory-path prefix match, SURF-02 — orthogonal to and composed AND
// with the language --filter, CONTEXT D-03 add-alongside), and --depth
// (directory-nesting
// cap; 0 means unlimited), projected as either the default "flat" list or
// a nested "--format tree". --json (short -j, SURF-03) emits
// query.MarshalFilesJSON's FilesResult shape — this plan's own design,
// no golden oracle exists for files (D-07a).
func newFilesCmd() *cobra.Command {
	var path, pattern, filter, dir, format string
	var depth int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "files",
		Short: "Browse the indexed file structure",
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

			result, err := eng.Files(query.FilesOptions{
				Pattern: pattern,
				Filter:  filter,
				Dir:     dir,
				Depth:   depth,
				Format:  format,
			})
			if err != nil {
				return err
			}

			if jsonOut {
				data, err := query.MarshalFilesJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()
			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json early return above —
			// see explore.go's call site for the full rationale.
			notice := query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context()))
			mode := resolveColor(cmd)
			if mode.Styled {
				// WR-01 (04-REVIEW.md): route the notice through
				// present.RenderNotice on the styled branch, exactly as
				// every sibling read command (explore/node/search/
				// callers/callees/impact/affected) does, so it renders in
				// the Warning role instead of unstyled plain text.
				w := mode.Writer(out)
				pal := present.NewPalette(mode.Dark)
				if err := present.RenderNotice(notice, pal, w); err != nil {
					return err
				}
				return present.RenderFiles(result, pal, w)
			}

			fmt.Fprint(out, notice)

			if result.Format == "tree" {
				printFileTree(out, result.Tree, "")
			} else {
				for _, f := range result.Files {
					fmt.Fprintf(out, "%s (%s)\n", f.Path, f.Language)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVar(&pattern, "pattern", "", "shell glob narrowing the result set")
	cmd.Flags().StringVar(&filter, "filter", "", "restrict to one language")
	cmd.Flags().StringVar(&dir, "dir", "", "directory-path prefix filter")
	cmd.Flags().IntVar(&depth, "depth", 0, "directory-nesting cap (0 = unlimited)")
	cmd.Flags().StringVar(&format, "format", "", `"flat" (default) or "tree"`)
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")

	return cmd
}

// printFileTree renders a query.FileTreeNode slice as indented plain text
// for --format tree's default (non-JSON) output.
func printFileTree(out io.Writer, nodes []*query.FileTreeNode, indent string) {
	for _, n := range nodes {
		if n.IsDir {
			fmt.Fprintf(out, "%s%s/\n", indent, n.Name)
			printFileTree(out, n.Children, indent+"  ")
		} else {
			fmt.Fprintf(out, "%s%s (%s)\n", indent, n.Name, n.Language)
		}
	}
}
