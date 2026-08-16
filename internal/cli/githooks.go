package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/githooks"
)

// newGithooksCmd builds the `codegraph githooks` command tree: install /
// remove / status, each taking an optional [path] resolved via the
// package-level targetRoot (D-11), matching init/uninit/sync. This is a
// project-native command surface wrapping internal/githooks directly —
// init/uninit also call the equivalent functions internally, but this
// gives users a standalone entry point (D-01).
func newGithooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "githooks",
		Short: "Manage git sync hooks (post-commit/post-merge/post-checkout)",
	}
	cmd.AddCommand(newGithooksInstallCmd(), newGithooksRemoveCmd(), newGithooksStatusCmd())
	return cmd
}

// newGithooksInstallCmd builds `githooks install [path]`.
func newGithooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install marker-fenced git sync hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			result := githooks.Install(cmd.Context(), root)
			if result.Skipped != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipped: %s\n", result.Skipped)
				return nil
			}
			printHookErrors(cmd, result.Errors)
			if len(result.Installed) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Could not install git hooks. Run `codegraph sync` after changes instead.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed git %s hooks — the index refreshes in the background after each.\n",
				strings.Join(result.Installed, ", "))
			fmt.Fprintln(cmd.OutOrStdout(), "Run `codegraph sync` anytime to refresh immediately.")
			return nil
		},
	}
}

// newGithooksRemoveCmd builds `githooks remove [path]`.
func newGithooksRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [path]",
		Short: "Remove codegraph's git sync hook blocks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			result := githooks.Remove(cmd.Context(), root)
			if result.Skipped != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipped: %s\n", result.Skipped)
				return nil
			}
			printHookErrors(cmd, result.Errors)
			if len(result.Removed) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No git sync hooks were installed — nothing to remove.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed git %s sync hook%s\n",
				strings.Join(result.Removed, ", "), plural(len(result.Removed)))
			return nil
		},
	}
}

// printHookErrors prints one line per per-hook write/delete failure
// accumulated in InstallResult.Errors/RemoveResult.Errors (WR-01), so a
// partial success (e.g. one hook unwritable while the other two succeed)
// is no longer silently indistinguishable from "that hook was never
// touched." A no-op when errs is empty.
func printHookErrors(cmd *cobra.Command, errs []error) {
	for _, e := range errs {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", e)
	}
}

// newGithooksStatusCmd builds `githooks status [path]`.
func newGithooksStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [path]",
		Short: "Show git sync hook install state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			result := githooks.Status(cmd.Context(), root)
			if result.Skipped != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipped: %s\n", result.Skipped)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "hooks dir: %s\n", result.HooksDir)
			for _, h := range result.Hooks {
				state := "not installed"
				if h.Installed && !h.Executable {
					// IN-03: the marker text is present but the exec bit
					// isn't — git will never actually run this hook.
					state = "installed but not executable"
				} else if h.Installed {
					state = "installed"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", h.Name, state)
			}
			return nil
		},
	}
}
