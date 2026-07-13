package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/migrate"
)

// newMigrateCmd builds the `codegraph migrate` command: the MIGR-01
// single-command surface that converts a TS CodeGraph SQLite index into a
// new-format Pebble store, delegating entirely to migrate.Run (D-08). Both
// --from and --to default to the TS `.codegraph/` in cwd — the common case
// is an in-place conversion: read the existing TS index, write the
// new-format store into a sibling partial directory, and atomically swap it
// into the SAME path once validation (D-09) passes.
//
// Non-destructive-to-source is structural, not a flag: migrate.Run only
// ever opens `from` read-only (OpenSource's mode=ro DSN) and never writes to
// it (D-08). Non-destructive-to-target is enforced twice: this command
// prompts for confirmation before overwriting any non-empty target
// directory (unless --force), and migrate.Run's own checkTargetOverwrite is
// the authoritative last line of defense regardless of what the CLI does.
func newMigrateCmd() *cobra.Command {
	var from, to string
	var force, dropDangling bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Convert a TypeScript CodeGraph index into the new format",
		Long: "migrate performs a one-way, one-step conversion of a TypeScript " +
			"CodeGraph SQLite index (.codegraph/) into a new-format codegraph-go " +
			"Pebble store. The source is never mutated or deleted. Because the " +
			"migrated graph preserves the source's node ids verbatim (which differ " +
			"from natively-computed ids), the first `codegraph sync` or " +
			"`codegraph index` run after migrating will perform a full re-index — " +
			"this is expected, not an error.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(nil)
			if err != nil {
				return err
			}
			defaultDir := filepath.Join(root, codegraphDirName)

			resolvedFrom := defaultDir
			if from != "" {
				resolvedFrom, err = filepath.Abs(from)
				if err != nil {
					return fmt.Errorf("resolve --from %q: %w", from, err)
				}
			}
			resolvedTo := defaultDir
			if to != "" {
				resolvedTo, err = filepath.Abs(to)
				if err != nil {
					return fmt.Errorf("resolve --to %q: %w", to, err)
				}
			}

			if !force {
				nonEmpty, err := dirNonEmpty(resolvedTo)
				if err != nil {
					return err
				}
				if nonEmpty {
					ok, err := confirm(cmd, fmt.Sprintf("Overwrite existing contents of %s with the migrated store?", resolvedTo))
					if err != nil {
						return err
					}
					if !ok {
						fmt.Fprintln(cmd.OutOrStdout(), "aborted (pass --force to overwrite without confirming)")
						return nil
					}
					force = true
				}
			}

			result, err := migrate.Run(resolvedFrom, resolvedTo, migrate.Options{
				Force:        force,
				DropDangling: dropDangling,
			})
			if err != nil {
				if errors.Is(err, migrate.ErrNotATSSource) {
					return fmt.Errorf("%s does not look like a TypeScript CodeGraph index (missing schema_versions/nodes/edges tables): %w", resolvedFrom, err)
				}
				return err
			}

			printMigrateReport(cmd, result)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "path to the TS .codegraph/ directory or *.db file (default: .codegraph/ in the current directory)")
	cmd.Flags().StringVar(&to, "to", "", "path for the new-format .codegraph/ directory (default: .codegraph/ in the current directory, in place)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite a non-empty target without prompting for confirmation")
	cmd.Flags().BoolVar(&dropDangling, "drop-dangling", false, "drop dangling (non-file:) edges instead of failing loud")

	return cmd
}

// dirNonEmpty reports whether dir exists and contains at least one entry. A
// missing directory is reported as empty (nonEmpty=false, err=nil) — nothing
// to confirm before migrating into it.
func dirNonEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return len(entries) > 0, nil
}

// printMigrateReport prints the D-09 count reconciliation (source vs.
// migrated, per table) plus Result.HealthMessage — which already carries
// the D-01 first-sync/index full-reindex note (migrate.go's healthMessage)
// — to cmd's configured stdout.
func printMigrateReport(cmd *cobra.Command, result migrate.Result) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "migrated: files=%d/%d nodes=%d/%d edges=%d/%d (migrated/source)\n",
		result.Files, result.Report.Files.Source,
		result.Nodes, result.Report.Nodes.Source,
		result.Edges, result.Report.Edges.Source)
	if result.Report.Dropped > 0 {
		fmt.Fprintf(out, "dropped %d dangling edge(s) (--drop-dangling)\n", result.Report.Dropped)
	}
	if result.Resumed {
		fmt.Fprintln(out, "resumed from a previously interrupted migration")
	}
	if result.HealthMessage != "" {
		fmt.Fprintln(out, result.HealthMessage)
	}
}
