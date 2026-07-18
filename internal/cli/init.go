package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/githooks"
	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
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

			// TUI-05/D-07/D-08: TTY-gated spinner feedback on stderr for the
			// long-running indexer.Run call (WR-04 06-REVIEW.md: shared
			// helper, see progress_cli.go). Stop is deferred so teardown
			// runs even if indexer.Run errors.
			defer startProgress(quiet, "indexing")()

			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if err != nil {
				return err
			}
			printSummary(cmd, stats, quiet, verbose)
			printWatchFallbackAdvisory(cmd, root)
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

// printWatchFallbackAdvisory is a non-interactive plain-text port of TS
// offerWatchFallback (installer/index.js ~476-525, D-07), wired into init's
// success path and ONLY there this phase (D-08 — the already-initialized
// early-return branch above is untouched). Gate-for-gate:
//  1. watch.WatchDisabledReason == "" (watcher runs normally) -> print
//     nothing. This is the not-always-on guarantee (HOOK-03's narrower
//     trigger) — hooks are surfaced as a fallback, not an always-on feature.
//  2. Reason non-empty -> warn, plus the frozen-index explanation line.
//  3. Not a git repo -> point at `codegraph sync` and stop.
//  4. Hooks already installed (any of the 3, TS isSyncHookInstalled's
//     some() semantics) -> the already-installed info line and stop.
//  5. Otherwise -> point at `codegraph githooks install` (no auto-install
//     without explicit user action in v1.0; the interactive select is
//     Phase 7 territory).
//
// watch.Probe{} is the zero value here, but it is NOT hardcoded-unreachable
// (D-13's test seam): WatchDisabledReason defaults a nil Probe.Env to
// os.Getenv, so a test can force this advisory to fire deterministically
// via t.Setenv("CODEGRAPH_NO_WATCH", "1") — the same seam serve.go's own
// --no-watch flag threads through, just driven by the env var side of the
// OR instead of the flag side.
func printWatchFallbackAdvisory(cmd *cobra.Command, root string) {
	reason := watch.WatchDisabledReason(root, watch.Probe{})
	if reason == "" {
		return
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Live file watching is disabled here — %s.\n", reason)
	fmt.Fprintln(out, "Until you re-sync, the CodeGraph index stays frozen — it will not pick up edits on its own.")

	if !gitmeta.IsGitRepo(cmd.Context(), root) {
		fmt.Fprintln(out, "Run `codegraph sync` after changing files to refresh the index.")
		return
	}

	status := githooks.Status(cmd.Context(), root)
	installed := false
	for _, h := range status.Hooks {
		if h.Installed {
			installed = true
			break
		}
	}
	if installed {
		fmt.Fprintln(out, "Git sync hooks are already installed — the index refreshes after commit / pull / checkout.")
		return
	}
	fmt.Fprintln(out, "Run `codegraph githooks install` to keep the index fresh automatically.")
}
