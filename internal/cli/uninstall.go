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
// (ResolveTargetFlag("all", ...)) without prompting, since Uninstall never
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
			"Reports removed / not-configured / unsupported per agent and never\n" +
			"errors on an agent that was never installed.",
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
			switch {
			case cmd.Flags().Changed("target"):
				// D-13: an explicit --target wins over --yes — --yes only
				// supplies the non-interactive default when no target was
				// named.
				targets, err = agents.ResolveTargetFlag(target, loc)
			case yes:
				// D-15/Pitfall 6: --yes still short-circuits BEFORE the TTY
				// branch, not merely skip rendering the picker, whenever
				// --target was not given.
				targets, err = agents.ResolveTargetFlag("all", loc)
			case interactiveAllowed(cmd):
				targets, err = runAgentPicker(cmd, loc)
			default:
				// D-13: no TTY (or CI) never blocks on a prompt — resolve
				// to the historical default (every registered target),
				// same as an explicit --target all.
				targets, err = agents.ResolveTargetFlag("all", loc)
			}
			if err != nil {
				return fmt.Errorf("codegraph uninstall: %w", err)
			}

			return printAgentResults(cmd, targets, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Uninstall(loc)
			}, uninstallStatus)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "all", "which agents to reverse: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVarP(&location, "location", "l", string(agents.LocationGlobal), "config scope: global|local")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the interactive picker; use the non-interactive default set (all)")

	return cmd
}

// uninstallStatus rolls WriteResult's per-file actions up into D-08's
// three-word status: "removed" if any file's codegraph entry/section was
// actually deleted, else "not-configured" (nothing was there to begin
// with — never an error, per D-08).
func uninstallStatus(result agents.WriteResult) string {
	for _, f := range result.Files {
		if f.Action == agents.ActionRemoved {
			return "removed"
		}
	}
	return "not-configured"
}
