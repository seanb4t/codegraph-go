package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
	"github.com/seanb4t/codegraph-go/internal/cli/tui"
)

// interactiveAllowed and runAgentPicker are package-level func vars —
// tui.InteractiveAllowed/tui.RunAgentPicker by default — so
// install_test.go/uninstall_test.go can force the interactive branch (and
// stub out the picker itself) without a real pty and without ever
// constructing a real tea.Program in a test process. Mirrors the
// project's existing injectable-seam convention (the old
// installStdinIsInteractive var this replaces, upgrade.go's
// upgradeRunFunc).
var interactiveAllowed = tui.InteractiveAllowed
var runAgentPicker = tui.RunAgentPicker

// parseLocationFlag validates --location against the two values
// agents.Location supports; any other value is a clear, immediate error
// rather than a silently-wrong config write (T-06-04-01's "unknown id"
// discipline applied to --location too).
func parseLocationFlag(raw string) (agents.Location, error) {
	switch agents.Location(raw) {
	case agents.LocationGlobal, agents.LocationLocal:
		return agents.Location(raw), nil
	default:
		return "", fmt.Errorf("--location must be \"global\" or \"local\" (got %q)", raw)
	}
}

// newInstallCmd builds `codegraph install` (AGNT-01, D-02): resolves
// --target/--location, resolves the running binary's absolute path once
// via os.Executable() (D-04 — the MCP config written points at THIS
// binary, not a PATH guess), selects targets from the internal/agents
// registry (an explicit --target, an interactive TTY multi-select, or the
// non-interactive auto fallback per D-03), and prints a per-agent status
// line. Contains no agent-specific logic — every quirk lives in the
// target's own file (06-02/06-03); this command only iterates the
// registry and delegates.
func newInstallCmd() *cobra.Command {
	var target string
	var location string
	var autoAllow bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Configure coding agents to use this codegraph binary as their MCP server",
		Long: "Detect and configure the agent roster (Claude Code, Cursor, Codex CLI,\n" +
			"opencode, Gemini CLI, Antigravity, Hermes, Kiro): write each agent's MCP\n" +
			"server entry plus, for the agents that support it, a short marker-fenced\n" +
			"instruction block. Idempotent — re-running install is a no-op when\n" +
			"nothing changed.",
		Example: "  codegraph install\n" +
			"  codegraph install --target all --location global\n" +
			"  codegraph install --target claude,cursor\n" +
			"  codegraph install --target none",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := parseLocationFlag(location)
			if err != nil {
				return fmt.Errorf("codegraph install: %w", err)
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("codegraph install: resolve running binary path: %w", err)
			}

			var targets []agents.AgentTarget
			switch {
			case yes:
				// D-15/Pitfall 6: --yes must short-circuit BEFORE the TTY
				// branch, not merely skip rendering the picker — checked
				// first in the switch so it always wins regardless of
				// stdin/stdout's actual state.
				targets, err = agents.ResolveTargetFlag("auto", loc)
			case cmd.Flags().Changed("target"):
				targets, err = agents.ResolveTargetFlag(target, loc)
			case interactiveAllowed(cmd):
				targets, err = runAgentPicker(cmd, loc)
			default:
				// D-13: no TTY (or CI) never blocks on a prompt — resolve
				// straight to auto, same as an explicit --target auto.
				targets, err = agents.ResolveTargetFlag("auto", loc)
			}
			if err != nil {
				return fmt.Errorf("codegraph install: %w", err)
			}

			opts := agents.InstallOptions{AutoAllow: autoAllow, ExecPath: execPath}
			return printAgentResults(cmd, targets, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Install(loc, opts)
			}, installStatus)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "auto", "which agents to configure: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVarP(&location, "location", "l", string(agents.LocationGlobal), "config scope: global|local")
	cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "also add mcp__codegraph__* to Claude Code's permissions.allow list")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the interactive picker; use the non-interactive default set (auto)")

	return cmd
}

// installStatus rolls WriteResult's per-file actions up into one word for
// install's per-agent summary line: "unchanged" only when every touched
// file was already correct (D-07 idempotency), "configured" otherwise.
func installStatus(result agents.WriteResult) string {
	for _, f := range result.Files {
		if f.Action != agents.ActionUnchanged {
			return "configured"
		}
	}
	return "unchanged"
}

// printAgentResults is install's and uninstall's shared per-agent
// reporting loop: skip (report "unsupported") any target that doesn't
// support loc without calling do() at all — Install/Uninstall return an
// empty WriteResult for an unsupported location, which would otherwise
// print as a confusing no-op rather than an explicit status (D-08). For
// every supported target, call do(), print statusOf(result) as the
// headline, then one indented line per touched file and note.
//
// Any WriteResult.Errors are printed as "  error: ..." lines and joined
// into the returned error (CR-01): a hard write failure — EACCES, a full
// disk, a failed MkdirAll — must never look identical to "unchanged"/
// "not-configured" in the CLI's own status line, and `codegraph
// install`/`uninstall` must exit non-zero when it happens. A nil return
// means every touched file's write/remove actually completed.
func printAgentResults(cmd *cobra.Command, targets []agents.AgentTarget, loc agents.Location, do func(agents.AgentTarget) agents.WriteResult, statusOf func(agents.WriteResult) string) error {
	out := cmd.OutOrStdout()
	if len(targets) == 0 {
		fmt.Fprintln(out, "no agents selected")
		return nil
	}
	var errs []error
	for _, t := range targets {
		if !t.SupportsLocation(loc) {
			fmt.Fprintf(out, "%s: unsupported (%s not supported)\n", t.DisplayName(), loc)
			continue
		}
		result := do(t)
		fmt.Fprintf(out, "%s: %s\n", t.DisplayName(), statusOf(result))
		for _, f := range result.Files {
			fmt.Fprintf(out, "  %s: %s\n", f.Action, f.Path)
		}
		for _, note := range result.Notes {
			fmt.Fprintf(out, "  note: %s\n", note)
		}
		for _, e := range result.Errors {
			fmt.Fprintf(out, "  error: %v\n", e)
			errs = append(errs, fmt.Errorf("%s: %w", t.DisplayName(), e))
		}
	}
	return errors.Join(errs...)
}
