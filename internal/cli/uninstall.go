package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// newUninstallCmd builds `codegraph uninstall` (AGNT-02, D-02): mirrors
// install's flag/resolve/report shape, reversing everything install wrote
// for each selected target. On a TTY with no --target, uninstall now
// presents the SAME bubbles checkbox multi-select install uses (D-14),
// pre-checked from agents.DetectAll(loc) — off-TTY or -y/--yes (D-15)
// keeps the historical default of resolving to every registered target
// (ResolveTargets("all", ...)) without prompting, since Uninstall never
// errors on a target that was never configured (D-08) and this keeps the
// no-TTY/CI path trivially non-blocking (D-13's never-hang guarantee
// applies here too, by construction).
func newUninstallCmd() *cobra.Command {
	var target string
	var location string
	var yes bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove codegraph's configuration from coding agents",
		Long: "Reverse everything `codegraph install` wrote for the selected agents —\n" +
			"the MCP server entry and, where present, the marker-fenced instruction\n" +
			"block — while preserving every unrelated key, entry, and section in\n" +
			"every file it touches. It removes codegraph's skill package too; a\n" +
			"shared skill directory's package is deleted only when no other agent\n" +
			"that installed it remains, and a skill directory codegraph did not\n" +
			"write is never touched. It also removes the Claude Code and Codex\n" +
			"CLI PreToolUse nudge hooks and their guard scripts when present.\n" +
			"Reports removed, kept, not-configured, unsupported, failed or partial\n" +
			"per agent, and never errors on an agent that was never installed.\n" +
			"With --target all or --target auto, uninstall skips an agent whose\n" +
			"detection fails, for example a CODEX_HOME that is not an existing\n" +
			"absolute directory. It prints a \"<id>: skipped: <reason>\" line on\n" +
			"stderr and a skipped row in the report. Naming the agent with --target\n" +
			"reports the error instead.",
		Example: "  codegraph uninstall\n" +
			"  codegraph uninstall --target all --location global\n" +
			"  codegraph uninstall --target claude,cursor",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := parseLocationFlag(location)
			if err != nil {
				return fmt.Errorf("codegraph uninstall: %w", err)
			}

			var targets []agents.AgentTarget
			var skipped []agents.SkippedTarget
			switch {
			case cmd.Flags().Changed("target"):
				// An explicit --target wins over --yes, which only supplies
				// the non-interactive default when no target was named.
				targets, skipped, err = resolveTargets(cmd, target, loc)
			case yes:
				// --yes short-circuits before the TTY branch whenever
				// --target was not given.
				targets, skipped, err = resolveTargets(cmd, "all", loc)
			case interactiveAllowed(cmd):
				targets, err = runAgentPicker(cmd, loc)
			default:
				// No TTY (or CI) never blocks on a prompt: resolve to every
				// registered target, as an explicit --target all does.
				targets, skipped, err = resolveTargets(cmd, "all", loc)
			}
			if err != nil {
				return fmt.Errorf("codegraph uninstall: %w", err)
			}

			return printAgentResults(cmd, targets, skipped, func(s agents.SkippedTarget) string { return skippedUninstallNote(s, loc) }, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Uninstall(loc)
			}, uninstallStatus)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "all", "which agents to reverse: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVarP(&location, "location", "l", string(agents.LocationGlobal), "config scope: global|local")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the interactive picker; use the non-interactive default set (all)")

	return cmd
}

// skippedUninstallNote is the note under the row of a target that uninstall
// skipped. It says that the shared skill package still lists the target as a
// requester, and is empty when the manifest at loc does not list it.
func skippedUninstallNote(s agents.SkippedTarget, loc agents.Location) string {
	if !agents.IsSharedSkillRequester(s.ID, loc) {
		return ""
	}
	return fmt.Sprintf("the shared skill package keeps %s as a requester until its detection is fixed and `codegraph uninstall --target %s` runs", s.ID, s.ID)
}

// uninstallStatus returns "removed" if any file's codegraph entry or section
// was deleted, "kept" if nothing was removed and some entry was kept in
// place, and "not-configured" otherwise. A result with errors is "failed"
// when nothing was removed and "partial" when something was.
func uninstallStatus(result agents.WriteResult) string {
	removed := false
	for _, f := range result.Files {
		if f.Action == agents.ActionRemoved {
			removed = true
		}
	}
	return statusWithErrors(result, removed, "removed", "not-configured")
}
