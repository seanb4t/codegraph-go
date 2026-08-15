package cli

import (
	"errors"
	"fmt"
	"io"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/tui"
	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// runDaemonPicker is tui.RunDaemonPicker indirected behind a package-level
// func var — mirroring install.go's interactiveAllowed/runAgentPicker seam
// (D-10) — so daemon_test.go can force the interactive branch and stub the
// picker without a real pty or tea.Program. interactiveAllowed itself is
// install.go's existing package-level var, reused verbatim here since both
// live in package cli — not redeclared.
var runDaemonPicker = tui.RunDaemonPicker

// daemonList is daemon.List indirected behind a package-level func var, the
// same convention, so daemon_test.go can seed the bare RunE's records
// deterministically without depending on any real OS process's liveness
// (List's own self-heal would otherwise prune any record whose pid isn't
// actually alive in the test process).
var daemonList = daemon.List

// newDaemonCmd builds the `codegraph daemon` command tree (D-01, DMON-01/
// DMON-02): bare (no args) opens the interactive bubbletea picker over
// every running daemon on a TTY, current-project first
// (tui.RunDaemonPicker) — or, off a TTY (D-12), prints the SAME ordering as
// a plain read-only list and exits 0, never blocking on stdin (TUI-04).
// `daemon start` is the old foreground server (moved verbatim, unchanged
// behavior, new name); `daemon stop [--all]` explicitly signals without
// ever opening a picker. Neither the bare command nor `stop` ever calls
// daemon.Run — only `daemon start` does (D-03, no silent auto-spawn).
// Resolves a naming ambiguity noted in 07-CONTEXT.md D-01: bare
// `daemon` was a foreground server here before this plan; it is now
// the interactive picker.
func newDaemonCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "List and manage running codegraph daemons",
		Long: "With no subcommand: on a TTY, open an interactive picker of every\n" +
			"running daemon (current project first) to stop one, stop all, or\n" +
			"cancel; off a TTY, print the same list and exit 0. Use `daemon start`\n" +
			"to run the shared watch/index server in the foreground, and\n" +
			"`daemon stop [--all]` to stop it non-interactively.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}
			currentRepo, err := filepath.Abs(start)
			if err != nil {
				return err
			}

			records, err := daemonList()
			if err != nil {
				return err
			}

			// Only open the interactive picker when there is actually
			// something to pick. An empty registry has nothing to select, so
			// it must NOT construct a bubbletea Program even on a TTY: doing
			// so emits terminal capability probes (DECRQM 2026/2027) whose
			// responses leak to stdout when the Program quits immediately on
			// the empty set (07-UAT test 1 / G-07-1). Fall through to the
			// plain "no running daemons" notice instead — identical output on
			// TTY and non-TTY for the empty case.
			if interactiveAllowed(cmd) && len(records) > 0 {
				return runDaemonPicker(cmd, currentRepo, records)
			}
			printDaemonList(cmd, currentRepo, records)
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path for current-project-first ordering (default: cwd)")
	cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd())

	return cmd
}

// printDaemonList renders D-12's non-TTY fallback: tui.SortRecordsCurrentFirst's
// ordering — the SAME ordering RunDaemonPicker's Model uses (TUI-04) — one
// line per daemon, or a "no running daemons" notice when records is empty.
// Never an error; the caller always returns nil after calling this.
func printDaemonList(cmd *cobra.Command, currentRepo string, records []daemon.Record) {
	out := cmd.OutOrStdout()
	if len(records) == 0 {
		fmt.Fprintln(out, "no running daemons")
		return
	}
	sorted := tui.SortRecordsCurrentFirst(records, currentRepo)
	fmt.Fprintln(out, "pid\trepo\tstarted")
	for _, r := range sorted {
		fmt.Fprintf(out, "%d\t%s\t%s\n", r.PID, r.RepoRoot, r.StartedAt.Format(time.RFC3339))
	}
}

// newDaemonStartCmd builds `daemon start` (D-01/D-02): the explicit
// foreground blocking watch/index server multiple agent sessions share —
// reuses daemon.New/Run/RunWithRetry + the .codegraph/daemon.lock
// single-writer lockfile as-is. This is the ENTIRE body of the old bare
// `daemon` RunE, moved verbatim (same behavior, new name), including the
// watch.DisabledError friendly-exit branch (03-REVIEW.md IN-06/WR-01).
func newDaemonStartCmd() *cobra.Command {
	var path string
	var quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the shared watch/index server in the foreground",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			d, err := daemon.New(start, indexer.Options{
				Workers: workers,
				Verbose: verbose,
				Quiet:   quiet,
			})
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			err = d.Run(ctx)
			// WR-01 (03-REVIEW.md): daemon.Run's shared policy gate
			// (WATCH-03/D-11) returns a watch.ErrWatchDisabled-matching
			// error on a WSL2 /mnt/<drive> repo or with CODEGRAPH_NO_WATCH=1
			// exported — before this branch existed, this command exited
			// nonzero with the raw sentinel and none of the D-12 guidance
			// serve.go prints. Mirror serve.go's friendly message
			// (internal/cli/serve.go, serveWatchStart's ErrWatchDisabled
			// branch) and exit cleanly: a policy-disabled watcher is a
			// deliberate, explained state, not a failure.
			//
			// IN-05: the reason is extracted via errors.As from the typed
			// watch.DisabledError daemon.Run returned — the exact string
			// its own policy gate saw (already computed on the absolutized
			// root) — never re-derived here, where a divergent root
			// normalization could desynchronize it. The "[CodeGraph MCP]"
			// banner serve.go keeps (D-12) is deliberately dropped here:
			// this is the standalone daemon command, not the MCP server.
			var disabled *watch.DisabledError
			if errors.As(err, &disabled) {
				fmt.Fprintf(cmd.ErrOrStderr(), "File watcher disabled — %s. "+
					"The graph will not auto-update; run `codegraph sync` "+
					"(or install the git sync hooks via `codegraph init`) to refresh.\n", disabled.Reason)
				return nil
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress daemon-driven sync progress output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail for daemon-driven syncs")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the daemon's extraction worker pool (default: number of CPUs)")

	return cmd
}

// daemonStopMatching/daemonStopAll are daemon.StopMatching/daemon.StopAll
// indirected behind package-level func vars — the same injectable-seam
// convention as runDaemonPicker/interactiveAllowed above — so
// daemon_test.go can assert `daemon stop`'s dispatch without ever
// delivering a real OS signal.
var daemonStopMatching = daemon.StopMatching
var daemonStopAll = daemon.StopAll

// newDaemonStopCmd builds `daemon stop [--all]` (D-02, A2): explicit,
// non-interactive signaling — it never opens a Program and never calls
// daemon.Run (D-03, no auto-spawn). `-p`/`--path` defaults to the current
// repo (mirroring `daemon start`'s own flag); `--all` stops every live
// daemon instead. Both daemonStopMatching/daemonStopAll (daemon.StopMatching/
// StopAll, 07-04) re-corroborate every target's liveness immediately before
// signaling (isStale defense-in-depth) — this command only decides WHICH
// targets, never how they're validated. An empty/no-match result is a
// clean "no running daemon(s)" notice, exit 0 — not an error; an aggregated
// per-target stop error IS surfaced as a non-zero exit.
func newDaemonStopCmd() *cobra.Command {
	var path string
	var all bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current-repo daemon, or every running daemon (--all)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if all {
				stopped, err := daemonStopAll()
				printStoppedDaemons(out, stopped)
				if len(stopped) == 0 {
					fmt.Fprintln(out, "no running daemons")
				}
				return err
			}

			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}
			repoRoot, err := filepath.Abs(start)
			if err != nil {
				return err
			}

			stopped, err := daemonStopMatching(repoRoot)
			printStoppedDaemons(out, stopped)
			if len(stopped) == 0 {
				fmt.Fprintf(out, "no running daemon for %s\n", repoRoot)
			}
			return err
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path to stop (default: cwd)")
	cmd.Flags().BoolVar(&all, "all", false, "stop every running daemon, not just the current repo's")
	// WR-02 (07-REVIEW.md): before this, `--all --path <p>` silently
	// dropped --path (the `if all { ...; return }` branch above never
	// looked at path) with no error and no --help indication the two
	// flags conflict. cobra surfaces the conflict itself (both in --help
	// and as a RunE error) rather than one flag winning silently.
	cmd.MarkFlagsMutuallyExclusive("path", "all")

	return cmd
}

// printStoppedDaemons prints one "stopped pid N (repo)" line per record
// daemonStopMatching/daemonStopAll actually signaled.
func printStoppedDaemons(out io.Writer, stopped []daemon.Record) {
	for _, rec := range stopped {
		fmt.Fprintf(out, "stopped pid %d (%s)\n", rec.PID, rec.RepoRoot)
	}
}
