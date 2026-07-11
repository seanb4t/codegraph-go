package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// codegraphDirName is the directory (D-01b) created at the target repo
// root by init and removed wholesale by uninit.
const codegraphDirName = ".codegraph"

// storeDirName is the Pebble store's subdirectory under codegraphDirName
// (D-01b: layout is executor's discretion; the store lives in a subdir
// alongside room for future non-store state without a separate on-disk
// config file).
const storeDirName = "store"

// newInitCmd builds the `codegraph init` command: creates .codegraph/
// (with its Pebble store subdirectory, D-01b) and runs a full
// from-scratch index in one step (INDX-01). If .codegraph/ already exists
// it returns ErrAlreadyInitialized instead of touching the existing store
// (D-01a) — the guidance directs the user to `codegraph index --force` to
// rebuild.
func newInitCmd() *cobra.Command {
	var quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Create .codegraph/ and build the full graph in one step",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}

			codegraphDir := filepath.Join(root, codegraphDirName)
			if _, err := os.Stat(codegraphDir); err == nil {
				return fmt.Errorf("%w: %s already exists — use `codegraph index --force` to rebuild", ErrAlreadyInitialized, codegraphDir)
			} else if !os.IsNotExist(err) {
				return err
			}

			storeDir := filepath.Join(codegraphDir, storeDirName)
			if err := os.MkdirAll(storeDir, 0o755); err != nil {
				return err
			}
			if err := writeGitignoreHint(codegraphDir); err != nil {
				return err
			}

			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if err != nil {
				return err
			}
			printSummary(cmd, stats, quiet, verbose)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the extraction worker pool (default: number of CPUs)")

	return cmd
}

// targetRoot resolves the repo root a command should operate on: args[0]
// (absolute-ified) if given, else the current working directory.
func targetRoot(args []string) (string, error) {
	if len(args) > 0 {
		return filepath.Abs(args[0])
	}
	return os.Getwd()
}

// writeGitignoreHint writes a .gitignore inside .codegraph/ that ignores
// its own contents entirely (the same self-contained pattern Terraform
// uses for .terraform/.gitignore) rather than editing the repo's root
// .gitignore — init never risks corrupting a file it doesn't own (A4,
// executor's discretion).
func writeGitignoreHint(codegraphDir string) error {
	return os.WriteFile(filepath.Join(codegraphDir, ".gitignore"), []byte("*\n"), 0o644)
}

// printSummary prints the concise end-of-run summary (files, nodes,
// edges, duration) that init/index report by default (D-01a). --quiet
// suppresses it entirely; --verbose adds unresolved/skipped counts on top
// of the default line.
func printSummary(cmd *cobra.Command, stats indexer.Stats, quiet, verbose bool) {
	if quiet {
		return
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "files=%d nodes=%d edges=%d duration=%s\n",
		stats.Files, stats.Nodes, stats.Edges, stats.Duration.Round(time.Millisecond))
	if verbose {
		fmt.Fprintf(out, "unresolved=%d skipped=%d\n", stats.Unresolved, stats.Skipped)
	}
}
