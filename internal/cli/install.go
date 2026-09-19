package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
	"github.com/seanb4t/codegraph-go/internal/cli/present"
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
	var pretoolNudge bool
	var yes bool
	var printCfgStyle bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Configure coding agents to use this codegraph binary as their MCP server",
		Long: "Detect and configure the agent roster (Claude Code, Cursor, Codex CLI,\n" +
			"opencode, Gemini CLI, Antigravity, Hermes, Kiro): write each agent's MCP\n" +
			"server entry plus, for the agents that support it, a short marker-fenced\n" +
			"instruction block. Install also writes the codegraph skill package\n" +
			"(SKILL.md plus a sidecar manifest) into the skill directory each agent\n" +
			"reads; a directory shared by several agents holds one package they own\n" +
			"jointly, and a codegraph/ skill directory codegraph did not write is\n" +
			"left untouched. --print-config-style prints what each agent receives\n" +
			"without writing anything. With --pretool-nudge (Claude Code and Codex\n" +
			"CLI), install also registers a PreToolUse hook that adds a one-line\n" +
			"pointer to codegraph_explore when the agent searches an indexed\n" +
			"repository; it never blocks a tool call, and the choice is\n" +
			"remembered until --pretool-nudge=false or uninstall. Codex skips a\n" +
			"new or changed hook until it is trusted in /hooks. Idempotent —\n" +
			"re-running install is a no-op when nothing changed.",
		Example: "  codegraph install\n" +
			"  codegraph install --target all --location global\n" +
			"  codegraph install --target claude,cursor\n" +
			"  codegraph install --target none\n" +
			"  codegraph install --print-config-style --location local\n" +
			"  codegraph install --target claude --pretool-nudge\n" +
			"  codegraph install --target codex --location local --pretool-nudge",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := parseLocationFlag(location)
			if err != nil {
				return fmt.Errorf("codegraph install: %w", err)
			}

			// D-04: --print-config-style is read-only and short-circuits
			// BEFORE os.Executable() and the target-resolution switch below
			// — it never opens the interactive picker and never writes a
			// byte. It still honours -t/--target and -l/--location like
			// every other branch, resolving "all" when --target was not
			// explicitly changed on this invocation.
			if printCfgStyle {
				targets := agents.AllTargets()
				if cmd.Flags().Changed("target") {
					targets, err = agents.ResolveTargetFlag(target, loc)
					if err != nil {
						return fmt.Errorf("codegraph install: %w", err)
					}
				}
				return printConfigStyle(cmd, targets, loc)
			}

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("codegraph install: resolve running binary path: %w", err)
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
				// branch, not merely skip rendering the picker — checked
				// here so it always wins regardless of stdin/stdout's
				// actual state whenever --target was not given.
				targets, err = agents.ResolveTargetFlag("auto", loc)
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

			// v0.14.0 Phase 6 D-10: --pretool-nudge is a tri-state read
			// through Changed. Not given keeps (and refreshes) a recorded
			// opt-in and never adds one; given as true opts in; given as
			// --pretool-nudge=false opts out.
			nudge := agents.PreToolNudgeKeep
			if cmd.Flags().Changed("pretool-nudge") {
				nudge = agents.PreToolNudgeOff
				if pretoolNudge {
					nudge = agents.PreToolNudgeOn
				}
				// D-09 (widened 07-08): the flag configures Claude Code and
				// Codex CLI; say so when it was given but neither is a
				// resolved target. Plain stderr — stdout carries the
				// per-agent report.
				hasClaudeOrCodex := slices.ContainsFunc(targets, func(t agents.AgentTarget) bool {
					return t.ID() == agents.Claude || t.ID() == agents.Codex
				})
				if !hasClaudeOrCodex {
					fmt.Fprintln(cmd.ErrOrStderr(), "note: --pretool-nudge only configures Claude Code and Codex CLI, neither of which is among the selected agents; nothing was changed for them")
				}
			}
			opts := agents.InstallOptions{AutoAllow: autoAllow, ExecPath: execPath, PreToolNudge: nudge}
			return printAgentResults(cmd, targets, loc, func(t agents.AgentTarget) agents.WriteResult {
				return t.Install(loc, opts)
			}, installStatus)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "auto", "which agents to configure: auto|all|none|<comma-separated ids>")
	cmd.Flags().StringVarP(&location, "location", "l", string(agents.LocationGlobal), "config scope: global|local")
	cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "also add mcp__codegraph__* to Claude Code's permissions.allow list")
	cmd.Flags().BoolVar(&pretoolNudge, "pretool-nudge", false, "Claude Code and Codex CLI: register a PreToolUse hook that points the agent at codegraph_explore when it searches; remembered across install and upgrade until --pretool-nudge=false or uninstall")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the interactive picker; use the non-interactive default set (auto)")
	cmd.Flags().BoolVar(&printCfgStyle, "print-config-style", false, "print each agent's capability table (scopes, MCP config, format, instructions, skill dir, hooks) and exit without writing anything")

	return cmd
}

// installStatus rolls WriteResult's per-file actions up into one word for
// install's per-agent summary line: "unchanged" only when every touched
// file was already correct (D-07 idempotency) or reports agents.ActionKeptForeign
// (a foreign skill directory codegraph left untouched is not a change
// codegraph made — D-14), "configured" otherwise.
func installStatus(result agents.WriteResult) string {
	for _, f := range result.Files {
		if f.Action != agents.ActionUnchanged && f.Action != agents.ActionKeptForeign {
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
	mode := resolveColor(cmd)
	var pal present.Palette
	var w io.Writer
	if mode.Styled {
		pal = present.NewPalette(mode.Dark)
		w = mode.Writer(out)
	}

	if len(targets) == 0 {
		if mode.Styled {
			_ = present.Line(w, pal, present.RoleLabel, "no agents selected")
		} else {
			fmt.Fprintln(out, "no agents selected")
		}
		return nil
	}
	var errs []error
	for _, t := range targets {
		if !t.SupportsLocation(loc) {
			if mode.Styled {
				_, _ = io.WriteString(w, pal.Value.Render(t.DisplayName()+":")+" "+
					pal.Warning.Render(fmt.Sprintf("unsupported (%s not supported)", loc))+"\n")
			} else {
				fmt.Fprintf(out, "%s: unsupported (%s not supported)\n", t.DisplayName(), loc)
			}
			continue
		}
		result := do(t)
		if mode.Styled {
			_, _ = io.WriteString(w, pal.Header.Render(t.DisplayName()+":")+" "+pal.Value.Render(statusOf(result))+"\n")
		} else {
			fmt.Fprintf(out, "%s: %s\n", t.DisplayName(), statusOf(result))
		}
		for _, f := range result.Files {
			if mode.Styled {
				// CR-01/T-04-24: f.Path is filesystem-derived (a resolved
				// config location) — sanitized before pal.Path.Render like
				// every other adversarial-capable path this phase styles.
				actionRole := present.RoleWarning
				switch f.Action {
				case agents.ActionUnchanged, agents.ActionKept, agents.ActionNotFound:
					actionRole = present.RoleLabel
				}
				_, _ = io.WriteString(w, "  "+pal.Style(actionRole).Render(string(f.Action)+":")+" "+
					pal.Path.Render(sanitizePathForDisplay(f.Path))+"\n")
			} else {
				fmt.Fprintf(out, "  %s: %s\n", f.Action, f.Path)
			}
		}
		for _, note := range result.Notes {
			if mode.Styled {
				_, _ = io.WriteString(w, "  "+pal.Label.Render("note:")+" "+pal.Value.Render(note)+"\n")
			} else {
				fmt.Fprintf(out, "  note: %s\n", note)
			}
		}
		for _, e := range result.Errors {
			if mode.Styled {
				_, _ = io.WriteString(w, "  "+pal.Error.Render("error: "+sanitizePathForDisplay(e.Error()))+"\n")
			} else {
				fmt.Fprintf(out, "  error: %v\n", e)
			}
			errs = append(errs, fmt.Errorf("%s: %w", t.DisplayName(), e))
		}
	}
	return errors.Join(errs...)
}
