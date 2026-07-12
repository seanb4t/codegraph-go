package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// newUninstallCmd builds `codegraph uninstall` (AGNT-02, D-02): mirrors
// install's flag/resolve/report shape, reversing everything install wrote
// for each selected target. Unlike install, there is no interactive
// multi-select — a destructive-by-default reversal defaults to "all"
// without prompting when --target is omitted, since Uninstall never
// errors on a target that was never configured (D-08); this also keeps
// the no-TTY/CI path trivially non-blocking (D-03's DoS mitigation
// applies here too, by construction).
func newUninstallCmd() *cobra.Command {
	var target string
	var location string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove codegraph's configuration from coding agents",
		Long: "Reverse everything `codegraph install` wrote for the selected agents —\n" +
			"the MCP server entry and, where present, the marker-fenced instruction\n" +
			"block — while preserving every unrelated key, entry, and section in\n" +
			"every file it touches. Reports removed / not-configured / unsupported\n" +
			"per agent and never errors on an agent that was never installed.",
		Example: "  codegraph uninstall\n" +
			"  codegraph uninstall --target all --location global\n" +
			"  codegraph uninstall --target claude,cursor",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := parseLocationFlag(location)
			if err != nil {
				return fmt.Errorf("codegraph uninstall: %w", err)
			}

			targets, err := agents.ResolveTargetFlag(target, loc)
			if err != nil {
				return fmt.Errorf("codegraph uninstall: %w", err)
			}

			return printAgentResults(cmd, targets, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Uninstall(loc)
			}, uninstallStatus)
		},
	}

	cmd.Flags().StringVar(&target, "target", "all", "which agents to reverse: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVar(&location, "location", string(agents.LocationGlobal), "config scope: global|local")

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
